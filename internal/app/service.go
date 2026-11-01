// Package app contains the services exposed to the frontend.
package app

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"MaaGO/internal/apppath"
	"MaaGO/internal/config"
	"MaaGO/internal/connection"
	"MaaGO/internal/lifecycle"
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
	emit      func(name string, payload any)
}

// NewService creates the application service.
func NewService(r *runner.Runner, s *scheduler.Scheduler, emit func(name string, payload any)) *Service {
	return &Service{runner: r, scheduler: s, emit: emit}
}

// AppInfo describes the running build.
type AppInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	DataDir   string `json:"dataDir"`
	LogsDir   string `json:"logsDir"`
}

// CoreStatus reports whether the MaaCore runtime is present.
type CoreStatus struct {
	Available bool   `json:"available"`
	Path      string `json:"path"`
	Message   string `json:"message"`
	Demo      bool   `json:"demo"`
}

// GetAppInfo returns build and environment metadata for the About page.
func (s *Service) GetAppInfo() AppInfo {
	return AppInfo{
		Name:      "MaaGO",
		Version:   MaaVersion.Version,
		Commit:    MaaVersion.Commit,
		BuildDate: MaaVersion.BuildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		DataDir:   apppath.Root(),
		LogsDir:   apppath.LogsDir(),
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

// GetCoreStatus checks for a MaaCore runtime next to the executable.
func (s *Service) GetCoreStatus() CoreStatus {
	candidates := []string{
		filepath.Join(apppath.InstallDir(), "MaaCore.dll"),
		filepath.Join(apppath.InstallDir(), "libMaaCore.dll"),
		filepath.Join(apppath.InstallDir(), "core", "MaaCore.dll"),
		filepath.Join(apppath.InstallDir(), "install", "MaaCore.dll"),
		filepath.Join(apppath.Root(), "core", "MaaCore.dll"),
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return CoreStatus{
				Available: true,
				Path:      path,
				Message:   "已找到 MaaCore，但调用层尚未接入；当前为界面演示模式。",
				Demo:      true,
			}
		}
	}
	return CoreStatus{
		Available: false,
		Message:   "未找到 MaaCore，当前为界面演示模式，不会操作游戏。",
		Demo:      true,
	}
}

// TestConnection checks whether one emulator ADB endpoint is reachable.
func (s *Service) TestConnection(address string) connection.TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return connection.Test(ctx, address)
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
