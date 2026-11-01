package main

import (
	"context"
	"embed"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"MaaGO/internal/app"
	"MaaGO/internal/apppath"
	"MaaGO/internal/config"
	"MaaGO/internal/lifecycle"
	"MaaGO/internal/notify"
	"MaaGO/internal/runner"
	"MaaGO/internal/scheduler"
	"MaaGO/internal/tray"

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

	taskRunner := runner.New(emit, nil, sendNotification)
	defer taskRunner.Close()

	taskScheduler := scheduler.New(taskRunner, emit, sendNotification)
	taskScheduler.Update(cfg.Schedule)

	service := app.NewService(taskRunner, taskScheduler, emit)

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

	if cfg.WindowMaximised {
		window.Maximise()
	}

	start := time.Now()
	if err := wailsApp.Run(); err != nil {
		log.Printf("startup: run: %v", err)
		return
	}
	log.Printf("shutdown: clean exit after %s", time.Since(start).Round(time.Millisecond))
}
