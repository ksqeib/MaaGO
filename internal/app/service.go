// Package app contains the services exposed to the frontend.
package app

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"MaaGO/internal/apppath"
	"MaaGO/internal/config"
	"MaaGO/internal/connection"
	"MaaGO/internal/diagnostics"
	"MaaGO/internal/lifecycle"
	"MaaGO/internal/maa"
	"MaaGO/internal/notify"
	"MaaGO/internal/runner"
	"MaaGO/internal/scheduler"
	"MaaGO/internal/tray"
	MaaVersion "MaaGO/internal/version"
)

// Service is the root Wails service. Frontend bindings are generated from its
// exported methods, so keep the surface small and intentional.
type Service struct {
	runner    *runner.Runner
	scheduler *scheduler.Scheduler
	core      *maa.Core
	emit      func(name string, payload any)
}

// NewService creates the application service.
func NewService(r *runner.Runner, s *scheduler.Scheduler, emit func(name string, payload any), core *maa.Core) *Service {
	return &Service{runner: r, scheduler: s, emit: emit, core: core}
}

// AppInfo describes the running build.
type AppInfo struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Commit         string `json:"commit"`
	BuildDate      string `json:"buildDate"`
	GoVersion      string `json:"goVersion"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	DataDir        string `json:"dataDir"`
	LogsDir        string `json:"logsDir"`
	CoreLogsDir    string `json:"coreLogsDir"`
	DiagnosticsDir string `json:"diagnosticsDir"`
}

// CoreStatus reports whether the MaaCore runtime is present.
type CoreStatus struct {
	Available bool   `json:"available"`
	Path      string `json:"path"`
	Message   string `json:"message"`
	Demo      bool   `json:"demo"`
	Version   string `json:"version"`
}

// GetAppInfo returns build and environment metadata for the About page.
func (s *Service) GetAppInfo() AppInfo {
	return AppInfo{
		Name:           "MaaGO",
		Version:        MaaVersion.Version,
		Commit:         MaaVersion.Commit,
		BuildDate:      MaaVersion.BuildDate,
		GoVersion:      runtime.Version(),
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		DataDir:        apppath.Root(),
		LogsDir:        apppath.LogsDir(),
		CoreLogsDir:    apppath.CoreLogsDir(),
		DiagnosticsDir: apppath.DiagnosticsDir(),
	}
}

// GetConfig returns the persisted application configuration.
func (s *Service) GetConfig() (config.AppConfig, error) {
	return config.Load()
}

// SaveConfig persists the application configuration.
//
// Window geometry and run history are owned by the backend; they are copied
// from the current config so a stale frontend snapshot cannot reset them.
func (s *Service) SaveConfig(cfg config.AppConfig) error {
	var saved config.AppConfig
	err := config.Update(func(current *config.AppConfig) {
		cfg.WindowWidth = current.WindowWidth
		cfg.WindowHeight = current.WindowHeight
		cfg.WindowMaximised = current.WindowMaximised
		cfg.LastRun = current.LastRun
		cfg.Schedule.LastTriggered = current.Schedule.LastTriggered
		cfg.Schedule.LastNotified = current.Schedule.LastNotified
		*current = cfg
		saved = *current
	})
	if err != nil {
		return err
	}
	s.scheduler.Update(saved.Schedule)
	if s.emit != nil {
		s.emit("maago:config", saved)
	}
	return nil
}

// GetCoreStatus reports the loaded MaaCore runtime.
func (s *Service) GetCoreStatus() CoreStatus {
	if s.core != nil && s.core.Available() {
		message := "MaaCore 已加载"
		if version := s.core.Version(); version != "" {
			message = fmt.Sprintf("MaaCore %s 已加载", version)
		}
		if err := s.core.ResourceError(); err != nil {
			return CoreStatus{
				Available: false,
				Path:      filepath.Join(s.core.Dir(), "MaaCore.dll"),
				Message:   fmt.Sprintf("MaaCore 资源加载失败：%v", err),
				Demo:      true,
				Version:   s.core.Version(),
			}
		}
		return CoreStatus{
			Available: true,
			Path:      filepath.Join(s.core.Dir(), "MaaCore.dll"),
			Message:   message,
			Demo:      false,
			Version:   s.core.Version(),
		}
	}

	path := ""
	message := "未找到可用的 MaaCore，当前为界面演示模式。"
	if s.core != nil {
		path = filepath.Join(s.core.Dir(), "MaaCore.dll")
		if err := s.core.Err(); err != nil {
			message = fmt.Sprintf("MaaCore 不可用：%v", err)
		}
	}
	return CoreStatus{
		Available: false,
		Path:      path,
		Message:   message,
		Demo:      true,
	}
}

// DiagnoseConnection runs the staged ADB / MaaCore / screenshot / touch checks.
func (s *Service) DiagnoseConnection(address string) diagnostics.Result {
	if s.runner != nil && s.runner.Busy() {
		return diagnostics.Result{
			Address: address,
			Message: "stop the running queue first",
			Steps: []diagnostics.Step{{
				Key:    diagnostics.StepCore,
				Status: diagnostics.StatusFail,
				Detail: "a task queue is currently running",
			}},
		}
	}
	cfg, err := config.Load()
	if err != nil {
		return diagnostics.Result{
			Address: address,
			Message: err.Error(),
			Steps: []diagnostics.Step{{
				Key:    diagnostics.StepCore,
				Status: diagnostics.StatusFail,
				Detail: err.Error(),
			}},
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return diagnostics.Run(ctx, s.core, cfg, address)
}

// OpenCoreLogsDir opens the date-rotated MaaCore log directory.
func (s *Service) OpenCoreLogsDir() error {
	return openPath(apppath.CoreLogsDir())
}

// OpenDiagnosticsDir opens the diagnostic screenshot directory.
func (s *Service) OpenDiagnosticsDir() error {
	return openPath(apppath.DiagnosticsDir())
}

// DetectConnection scans common emulator ports.
func (s *Service) DetectConnection() []connection.Candidate {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	candidates := connection.Detect(ctx)
	cfg, err := config.Load()
	if err == nil && strings.TrimSpace(cfg.Connection.Address) != "" {
		result := connection.Test(ctx, cfg.Connection.Address)
		if result.OK {
			duplicate := false
			for _, candidate := range candidates {
				if candidate.Address == result.Address {
					duplicate = true
					break
				}
			}
			if !duplicate {
				candidates = append([]connection.Candidate{{
					Name:      "当前配置",
					Address:   result.Address,
					LatencyMs: result.LatencyMs,
				}}, candidates...)
			}
		}
	}
	if err == nil && s.core != nil && s.core.Available() {
		adbPath := maa.ResolveADB(s.core.Dir(), cfg.Connection.AdbPath)
		for _, serial := range maa.ADBDevices(adbPath) {
			duplicate := false
			for _, candidate := range candidates {
				if candidate.Address == serial {
					duplicate = true
					break
				}
			}
			if !duplicate {
				candidates = append(candidates, connection.Candidate{
					Name:    "ADB 设备",
					Address: serial,
				})
			}
		}
	}
	return candidates
}

// GetScheduleStatus returns the next scheduled run.
func (s *Service) GetScheduleStatus() scheduler.Status {
	return s.scheduler.Status()
}

// TestNotification sends a system notification.
func (s *Service) TestNotification() error {
	return notify.Send("MaaGO · 测试通知", "系统通知已配置成功。")
}

// SetTrayLabels retranslates the tray menu.
func (s *Service) SetTrayLabels(show, startText, stopText, quit string) {
	tray.SetLabels(show, startText, stopText, quit)
}

// OpenDataDir opens the user data directory in Explorer.
func (s *Service) OpenDataDir() error {
	return openPath(apppath.Root())
}

// OpenLogsDir opens the log directory in Explorer.
func (s *Service) OpenLogsDir() error {
	return openPath(apppath.LogsDir())
}

// ShowMainWindow restores the launcher window.
func (s *Service) ShowMainWindow() {
	lifecycle.Restore()
}

// QuitApp terminates the application.
func (s *Service) QuitApp() {
	lifecycle.Quit()
}

// StartQueue starts every enabled task.
func (s *Service) StartQueue() error {
	return s.runner.Start()
}

// StopQueue asks the running queue to stop.
func (s *Service) StopQueue() error {
	return s.runner.Stop()
}

// StartTask runs one task by ID.
func (s *Service) StartTask(taskID string) error {
	return s.runner.StartTask(taskID)
}

// GetRunState returns the current queue state.
func (s *Service) GetRunState() runner.RunState {
	return s.runner.State()
}

// GetLogs returns the in-memory run log.
func (s *Service) GetLogs() []runner.LogEntry {
	return s.runner.Logs()
}

// ClearLogs empties the in-memory run log.
func (s *Service) ClearLogs() {
	s.runner.ClearLogs()
}
