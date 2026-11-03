// Package app contains the services exposed to the frontend.
package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"MaaGO/internal/apppath"
	"MaaGO/internal/config"
	"MaaGO/internal/connection"
	"MaaGO/internal/diagnostics"
	"MaaGO/internal/elevation"
	"MaaGO/internal/lifecycle"
	"MaaGO/internal/maa"
	"MaaGO/internal/notify"
	"MaaGO/internal/rdp"
	"MaaGO/internal/runner"
	"MaaGO/internal/scheduler"
	"MaaGO/internal/stages"
	"MaaGO/internal/tray"
	"MaaGO/internal/updater"
	MaaVersion "MaaGO/internal/version"
)

// Service is the root Wails service. Frontend bindings are generated from its
// exported methods, so keep the surface small and intentional.
type Service struct {
	runner    *runner.Runner
	scheduler *scheduler.Scheduler
	core      *maa.Core
	updates   *updater.Service
	emit      func(name string, payload any)
}

// NewService creates the application service.
func NewService(r *runner.Runner, s *scheduler.Scheduler, emit func(name string, payload any), core *maa.Core, updates *updater.Service) *Service {
	return &Service{runner: r, scheduler: s, emit: emit, core: core, updates: updates}
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
		// Scheduler marks are written by the backend; keep them when the
		// frontend saves a stale snapshot of the schedule entries.
		marksByID := make(map[string]config.ScheduleEntry, len(current.Schedule.Entries))
		for _, entry := range current.Schedule.Entries {
			marksByID[entry.ID] = entry
		}
		for i := range cfg.Schedule.Entries {
			if existing, ok := marksByID[cfg.Schedule.Entries[i].ID]; ok {
				cfg.Schedule.Entries[i].LastTriggered = existing.LastTriggered
				cfg.Schedule.Entries[i].LastNotified = existing.LastNotified
			}
		}
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

// CheckCoreUpdate asks GitHub for the latest MaaCore release.
func (s *Service) CheckCoreUpdate() (updater.ReleaseInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	return s.updates.Check(ctx)
}

// StartCoreUpdate downloads and stages a MaaCore release. Progress is emitted
// as maago:update; the staged files are installed on the next start.
func (s *Service) StartCoreUpdate(version string) error {
	if s.runner != nil && s.runner.Busy() {
		return fmt.Errorf("stop the running task queue first")
	}
	if s.updates.Pending() {
		return fmt.Errorf("an update is already staged; restart MaaGO first")
	}
	if s.updates.Busy() {
		return fmt.Errorf("an update is already running")
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
		defer cancel()
		_ = s.updates.DownloadCore(ctx, version)
	}()
	return nil
}

// StartResourceUpdate downloads and stages MaaResource hot-update files.
func (s *Service) StartResourceUpdate() error {
	if s.runner != nil && s.runner.Busy() {
		return fmt.Errorf("stop the running task queue first")
	}
	if s.updates.Pending() {
		return fmt.Errorf("an update is already staged; restart MaaGO first")
	}
	if s.updates.Busy() {
		return fmt.Errorf("an update is already running")
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		_ = s.updates.DownloadResource(ctx)
	}()
	return nil
}

// GetUpdateStatus returns the current update progress.
func (s *Service) GetUpdateStatus() updater.Status {
	return s.updates.Status()
}

// TestProxy checks GitHub API reachability through the supplied proxy URL.
func (s *Service) TestProxy(proxyURL string) updater.ProxyTestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return updater.TestProxy(ctx, proxyURL)
}

// PrivilegeStatus reports whether MaaGO has administrator rights. PC-window
// mode needs elevation because the game client runs at high integrity.
type PrivilegeStatus struct {
	Elevated bool `json:"elevated"`
}

// GetPrivilegeStatus tells the frontend whether a UAC restart is needed.
func (s *Service) GetPrivilegeStatus() PrivilegeStatus {
	return PrivilegeStatus{Elevated: elevation.IsElevated()}
}

// RestartAsAdmin relaunches MaaGO through the UAC prompt. It is required for
// Win32 window control when the game client is elevated as well.
func (s *Service) RestartAsAdmin() error {
	if elevation.IsElevated() {
		return fmt.Errorf("MaaGO 已经以管理员身份运行")
	}
	if err := elevation.RestartAsAdmin(); err != nil {
		return err
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		lifecycle.Quit()
	}()
	return nil
}

// RestartApp relaunches MaaGO so a staged update can be applied.
func (s *Service) RestartApp() error {
	return lifecycle.Restart()
}

// OpenUpdatesDir opens the staging directory.
func (s *Service) OpenUpdatesDir() error {
	if s.updates == nil {
		return fmt.Errorf("updater is unavailable")
	}
	return openPath(s.updates.Dir())
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

// GetStageCatalog returns the stage picker catalog. refresh forces a download
// of MAA's activity snapshot; otherwise a cache older than 6 hours is refreshed.
func (s *Service) GetStageCatalog(refresh bool) (stages.Catalog, error) {
	cfg, _ := config.Load()
	resourceDir := ""
	if s.core != nil {
		resourceDir = s.core.ResourceDir()
	}
	if resourceDir == "" {
		resourceDir = filepath.Join(filepath.Dir(maa.DefaultCoreDir()), "resource")
	}
	cachePath := filepath.Join(apppath.CacheDir(), stages.ActivityCacheFile)
	stale := true
	if info, err := os.Stat(cachePath); err == nil {
		stale = time.Since(info.ModTime()) > 6*time.Hour
	}
	if refresh || stale {
		if client, err := updater.NewHTTPClient(cfg); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_ = stages.FetchActivity(ctx, client, cachePath)
			cancel()
		}
	}
	return stages.Build(resourceDir, cfg.Run.ClientType, cachePath, time.Now())
}

// OpenDiagnosticsDir opens the diagnostic screenshot directory.
func (s *Service) OpenDiagnosticsDir() error {
	return openPath(apppath.DiagnosticsDir())
}

// OpenFailureShotsDir opens MaaCore's own interface screenshot directory. It
// holds the images MaaCore saved when recognition gave up.
func (s *Service) OpenFailureShotsDir() error {
	userDir := ""
	if s.core != nil {
		userDir = s.core.UserDir()
	}
	if userDir == "" {
		userDir = filepath.Join(apppath.Root(), "core")
	}
	dir := filepath.Join(userDir, "debug", "interface")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return openPath(dir)
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
	rdp.Shutdown()
	lifecycle.Quit()
}

// RDPPreview is one JPEG frame for the embedded preview.
type RDPPreview struct {
	Connected   bool   `json:"connected"`
	Host        string `json:"host"`
	Username    string `json:"username"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ImageWidth  int    `json:"imageWidth"`
	ImageHeight int    `json:"imageHeight"`
	Image       string `json:"image"`
	Error       string `json:"error,omitempty"`
}

func (s *Service) rdpConfig() config.RDPConfig {
	cfg, err := config.Load()
	if err != nil {
		return config.RDPConfig{}
	}
	return cfg.Connection.RDP
}

// GetRDPPreview returns the latest remote frame as a data URL. It is polled by
// the dashboard card and the enlarged preview dialog.
func (s *Service) GetRDPPreview(maxWidth int) RDPPreview {
	rc := s.rdpConfig()
	preview := RDPPreview{
		Host:     rc.Host,
		Username: rc.Username,
		Width:    rc.Width,
		Height:   rc.Height,
	}
	session := rdp.Current()
	if session == nil || !session.Alive() {
		return preview
	}
	cfg := session.Config()
	preview.Host = cfg.Host
	preview.Username = cfg.Username
	preview.Width = cfg.Width
	preview.Height = cfg.Height
	data, w, h, err := session.SnapshotJPEG(maxWidth, 70)
	if err != nil {
		preview.Error = err.Error()
		return preview
	}
	preview.Connected = true
	preview.ImageWidth = w
	preview.ImageHeight = h
	preview.Image = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data)
	return preview
}

// ConnectRDP opens the RDP session without running a task, so the preview and
// diagnostics can be used on demand.
func (s *Service) ConnectRDP() (RDPPreview, error) {
	cfg, err := config.Load()
	if err != nil {
		return RDPPreview{}, err
	}
	rc := cfg.Connection.RDP
	session, err := rdp.Ensure(rdp.Config{
		Host:     rc.Host,
		Port:     rc.Port,
		Username: rc.Username,
		Password: rc.Password,
		Domain:   rc.Domain,
		Width:    rc.Width,
		Height:   rc.Height,
	})
	if err != nil {
		return RDPPreview{Host: rc.Host, Username: rc.Username, Width: rc.Width, Height: rc.Height, Error: err.Error()}, err
	}
	if s.core != nil {
		_ = rdp.InstallPrintWindowHook(filepath.Join(s.core.Dir(), "MaaWin32ControlUnit.dll"))
	}
	session.UnlockIfLocked(rc.Password)
	return s.GetRDPPreview(640), nil
}

// SendRDPInput forwards user input from the in-app preview to the remote
// desktop. It is only called while the user has enabled operation mode.
func (s *Service) SendRDPInput(action string, x, y int, text, key string) error {
	session := rdp.Current()
	if session == nil || !session.Alive() {
		return fmt.Errorf("RDP 尚未连接")
	}
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "click":
		session.ClickAt(x, y)
	case "type":
		session.TypeText(text)
	case "key":
		return session.PressKeyName(key)
	case "wheel":
		session.WheelAt(x, y, 1)
	default:
		return fmt.Errorf("未知 RDP 输入：%s", action)
	}
	return nil
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
