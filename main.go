package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"MaaGO/internal/app"
	"MaaGO/internal/apppath"
	"MaaGO/internal/autostart"
	"MaaGO/internal/config"
	"MaaGO/internal/corelog"
	"MaaGO/internal/lifecycle"
	"MaaGO/internal/maa"
	"MaaGO/internal/notify"
	"MaaGO/internal/rdp"
	"MaaGO/internal/runner"
	"MaaGO/internal/scheduler"
	"MaaGO/internal/tray"
	"MaaGO/internal/updater"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var appIcon []byte

const (
	minWindowWidth  = 1024
	minWindowHeight = 600
)

func init() {
	application.RegisterEvent[runner.LogEntry]("maago:log")
	application.RegisterEvent[runner.RunState]("maago:run")
	application.RegisterEvent[runner.TaskState]("maago:task")
	application.RegisterEvent[scheduler.Status]("maago:schedule")
	application.RegisterEvent[config.AppConfig]("maago:config")
	application.RegisterEvent[updater.Status]("maago:update")
}

// setupLogging mirrors logs to a file and, when requested, the console.
func setupLogging(debug bool) *os.File {
	writers := []io.Writer{}
	file, err := os.OpenFile(apppath.StartupLogPath(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err == nil {
		writers = append(writers, file)
	}
	if debug {
		writers = append(writers, os.Stderr)
	}
	if len(writers) == 0 {
		writers = append(writers, io.Discard)
	}
	log.SetOutput(io.MultiWriter(writers...))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	return file
}

func isDebugArg(args []string) bool {
	for _, arg := range args {
		switch strings.ToLower(strings.TrimSpace(arg)) {
		case "debug", "-debug", "--debug", "/debug":
			return true
		}
	}
	return false
}

func isAutoStartArg(args []string) bool {
	for _, arg := range args {
		switch strings.ToLower(strings.TrimSpace(arg)) {
		case "autostart", "-autostart", "--autostart", "/autostart":
			return true
		}
	}
	return false
}

func main() {
	logFile := setupLogging(isDebugArg(os.Args[1:]))
	if logFile != nil {
		defer logFile.Close()
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("fatal panic: %v", recovered)
			panic(recovered)
		}
	}()

	cfg, err := config.Load()
	if err != nil {
		log.Printf("startup: load config: %v", err)
	}
	autoStart := isAutoStartArg(os.Args[1:])
	if err := autostart.Sync(cfg.Run.AutoStart); err != nil {
		log.Printf("startup: sync autostart: %v", err)
	}

	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Printf("startup: embedded frontend: %v", err)
		return
	}

	emit := func(name string, payload any) {
		if app := application.Get(); app != nil {
			app.Event.Emit(name, payload)
		}
	}
	sendNotification := func(title, message string) {
		if err := notify.Send(title, message); err != nil {
			log.Printf("notification failed: %v", err)
		}
	}

	coreLogs := corelog.New(apppath.CoreLogsDir())
	defer coreLogs.Close()
	if capture, err := maa.StartStdoutCapture(coreLogs); err != nil {
		log.Printf("startup: MaaCore stdout capture: %v", err)
	} else {
		defer capture.Stop()
	}

	coreDir := maa.DefaultCoreDir()
	installRoot := filepath.Dir(filepath.Clean(coreDir))
	baseDir := filepath.Join(installRoot, "base")
	if message, err := updater.MigrateLayout(coreDir); err != nil {
		log.Printf("startup: migrate resource layout: %v", err)
	} else if message != "" {
		log.Printf("startup: %s", message)
	}
	if message, err := updater.ApplyPending(coreDir); err != nil {
		log.Printf("startup: apply pending update: %v", err)
	} else if message != "" {
		log.Printf("startup: %s", message)
	}

	// Both cores read the shared resource tree under base/. The official core
	// is the official MAA package root, whose MaaCore.dll sits next to
	// resource/.
	activeCoreDir := coreDir
	if cfg.Core.Source == config.CoreOfficial {
		if _, err := os.Stat(filepath.Join(baseDir, "MaaCore.dll")); err == nil {
			activeCoreDir = baseDir
		} else {
			log.Printf("startup: official core requested but %s is missing; using the custom core", filepath.Join(baseDir, "MaaCore.dll"))
		}
	}
	log.Printf("startup: core source %s (%s)", cfg.Core.Source, activeCoreDir)
	core := maa.NewAtWithResource(activeCoreDir, baseDir)
	if core.Available() {
		userDir := filepath.Join(apppath.Root(), "core")
		if err := core.SetUserDir(userDir); err != nil {
			log.Printf("startup: MaaCore user dir: %v", err)
		}
		applyPerformanceOptions(core, cfg)
		if err := core.LoadResource(); err != nil {
			log.Printf("startup: MaaCore resource: %v", err)
		} else {
			log.Printf("startup: MaaCore %s loaded from %s", core.Version(), core.Dir())
		}
	} else {
		log.Printf("startup: MaaCore unavailable: %v", core.Err())
	}

	var taskRunner *runner.Runner
	runPostAction := func(state runner.RunState) {
		if state.Demo {
			return
		}
		cfg, err := config.Load()
		if err != nil {
			if taskRunner != nil {
				taskRunner.SystemLog("warn", fmt.Sprintf("结束后动作读取配置失败：%v", err))
			}
			return
		}
		if cfg.Run.PostAction != "exitArknights" {
			return
		}
		if cfg.Run.PostActionSkipOnError && state.Status != runner.StatusSuccess {
			if taskRunner != nil {
				taskRunner.SystemLog("info", "结束后动作：本次运行未成功，按设置跳过关闭")
			}
			return
		}
		switch cfg.Connection.Mode {
		case "rdp":
			session := rdp.Current()
			if session == nil || !session.Alive() {
				if taskRunner != nil {
					taskRunner.SystemLog("warn", "结束后动作：RDP 会话不在线，无法关闭远端明日方舟")
				}
				return
			}
			if taskRunner != nil {
				taskRunner.SystemLog("info", "结束后动作：正在关闭远端明日方舟…")
			}
			if err := session.CloseArknights(); err != nil {
				if taskRunner != nil {
					taskRunner.SystemLog("error", fmt.Sprintf("结束后动作：关闭远端明日方舟失败：%v", err))
				}
				return
			}
			if taskRunner != nil {
				taskRunner.SystemLog("success", "结束后动作：已发送关闭明日方舟命令")
			}
		case "win32":
			if taskRunner != nil {
				taskRunner.SystemLog("info", "结束后动作：正在关闭本机明日方舟…")
			}
			if err := exec.Command("taskkill", "/IM", "Arknights.exe", "/F").Run(); err != nil {
				if taskRunner != nil {
					taskRunner.SystemLog("warn", fmt.Sprintf("结束后动作：关闭本机明日方舟失败：%v", err))
				}
				return
			}
			if taskRunner != nil {
				taskRunner.SystemLog("success", "结束后动作：已关闭本机明日方舟")
			}
		default:
			if taskRunner != nil {
				taskRunner.SystemLog("warn", "结束后动作：关闭明日方舟仅适用于 PC / RDP 连接")
			}
		}
	}
	updates := updater.New(coreDir, emit)
	taskRunner = runner.New(emit, runPostAction, sendNotification, core, coreLogs)
	taskRunner.SetUpdateCheck(func(ctx context.Context, options runner.UpdateCheckOptions) (runner.UpdateCheckResult, error) {
		result, err := updates.CheckForQueueTask(ctx, updater.QueueUpdateOptions{
			Core:     options.Core,
			Base:     options.Base,
			Overlay:  options.Overlay,
			Download: options.Download,
		})
		return runner.UpdateCheckResult{
			CoreCurrent:    result.CoreCurrent,
			CoreLatest:     result.CoreLatest,
			BaseCurrent:    result.BaseCurrent,
			BaseLatest:     result.BaseLatest,
			OverlayCurrent: result.OverlayCurrent,
			OverlayLatest:  result.OverlayLatest,
			Staged:         result.Staged,
			Message:        result.Message,
		}, err
	})
	defer taskRunner.Close()

	taskScheduler := scheduler.New(taskRunner, emit, sendNotification)
	taskScheduler.Update(cfg.Schedule)

	service := app.NewService(taskRunner, taskScheduler, emit, core, updates)

	wailsApp := application.New(application.Options{
		Name:        "MaaGO",
		Description: "MaaAssistantArknights desktop companion",
		Icon:        appIcon,
		LogLevel:    slog.LevelDebug,
		ErrorHandler: func(err error) {
			log.Printf("wails error: %v", err)
		},
		Services: []application.Service{
			application.NewService(service),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "org.maago.desktop",
			ExitCode: 0,
			OnSecondInstanceLaunch: func(_ application.SecondInstanceData) {
				lifecycle.Restore()
			},
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(sub),
		},
	})

	width := max(cfg.WindowWidth, minWindowWidth)
	height := max(cfg.WindowHeight, minWindowHeight)

	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "MaaGO",
		Width:            width,
		Height:           height,
		MinWidth:         minWindowWidth,
		MinHeight:        minWindowHeight,
		Frameless:        true,
		BackgroundColour: application.RGBA{Red: 248, Green: 250, Blue: 252, Alpha: 255},
		URL:              "/",
		EnableFileDrop:   true,
		Hidden:           autoStart,
	})
	lifecycle.SetWindow(window)

	tray.Setup(wailsApp, appIcon, window, func() {
		if err := taskRunner.Start(); err != nil {
			taskRunner.SystemLog("warn", err.Error())
		}
	}, func() {
		_ = taskRunner.Stop()
	})

	schedulerCtx, cancelScheduler := context.WithCancel(context.Background())
	go taskScheduler.Start(schedulerCtx)

	persistWindowSize := func() {
		w, h := window.Width(), window.Height()
		if w <= 0 || h <= 0 {
			return
		}
		err := config.Update(func(c *config.AppConfig) {
			c.WindowWidth = max(w, minWindowWidth)
			c.WindowHeight = max(h, minWindowHeight)
			c.WindowMaximised = window.IsMaximised()
		})
		if err != nil {
			log.Printf("shutdown: persist window: %v", err)
		}
	}

	wailsApp.OnShutdown(func() {
		persistWindowSize()
		cancelScheduler()
		_ = taskRunner.Stop()
		taskRunner.Close()
	})

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		persistWindowSize()
		latest, err := config.Load()
		if err != nil {
			return
		}
		if latest.CloseToTray && !lifecycle.QuitRequested() {
			event.Cancel()
			window.Hide()
		}
	})

	window.OnWindowEvent(events.Common.WindowMinimise, func(*application.WindowEvent) {
		latest, err := config.Load()
		if err != nil || lifecycle.QuitRequested() {
			return
		}
		if latest.MinimizeToTray {
			go window.Hide()
		}
	})

	if cfg.WindowMaximised && !autoStart {
		window.Maximise()
	}

	start := time.Now()
	if err := wailsApp.Run(); err != nil {
		log.Printf("startup: run: %v", err)
		return
	}
	log.Printf("shutdown: clean exit after %s", time.Since(start).Round(time.Millisecond))
}

// applyPerformanceOptions selects the OCR inference provider before the
// resource (and OCR models) are loaded. GPU OCR uses ONNX/FastDeploy DirectML.
func applyPerformanceOptions(core *maa.Core, cfg config.AppConfig) {
	if core == nil || !core.Available() {
		return
	}
	if cfg.Performance.GPUOCR {
		device := strings.TrimSpace(cfg.Performance.GPUDevice)
		if device == "" {
			device = "0"
		}
		if ok, err := core.SetStaticOption(maa.StaticOptionGpuOCR, device); err != nil || !ok {
			log.Printf("startup: GPU OCR unavailable (%v); using CPU OCR", err)
		} else {
			log.Printf("startup: OCR GPU acceleration enabled (device %s)", device)
		}
		return
	}
	if ok, err := core.SetStaticOption(maa.StaticOptionCpuOCR, ""); err != nil || !ok {
		log.Printf("startup: set CPU OCR: %v", err)
	}
}
