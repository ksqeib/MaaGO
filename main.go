package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"

	"MaaGO/internal/app"
	"MaaGO/internal/apppath"
	"MaaGO/internal/config"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	win "golang.org/x/sys/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var appIcon []byte

const (
	minWindowWidth      = 960
	minWindowHeight     = 600
	defaultWindowWidth  = 1024
	defaultWindowHeight = 640

	singleInstanceName = `Global\MaaGO_SingleInstance`
)

var (
	user32                  = win.NewLazySystemDLL("user32.dll")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procIsIconic            = user32.NewProc("IsIconic")

	singleInstanceGuard win.Handle
)

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

// focusExistingWindow brings the already-running instance forward.
func focusExistingWindow() {
	title, err := win.UTF16PtrFromString("MaaGO")
	if err != nil {
		return
	}
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return
	}
	cmd := uintptr(5) // SW_SHOW
	if iconic, _, _ := procIsIconic.Call(hwnd); iconic != 0 {
		cmd = 9 // SW_RESTORE
	}
	_, _, _ = procShowWindow.Call(hwnd, cmd)
	_, _, _ = procSetForegroundWindow.Call(hwnd)
}

func ensureSingleInstance() (bool, error) {
	h, err := acquireSingleInstanceMutex(singleInstanceName, 0)
	if err == win.ERROR_ALREADY_EXISTS {
		focusExistingWindow()
		return false, nil
	}
	if err != nil {
		return false, err
	}
	singleInstanceGuard = h
	return true, nil
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

	first, err := ensureSingleInstance()
	if err != nil {
		log.Printf("startup: single instance: %v", err)
		return
	}
	if !first {
		return
	}
	defer win.CloseHandle(singleInstanceGuard)

	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Printf("startup: embedded frontend: %v", err)
		return
	}

	service := app.NewService()
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

	var persistOnce sync.Once
	persistWindowSize := func() {
		persistOnce.Do(func() {
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
		})
	}
	wailsApp.OnShutdown(persistWindowSize)
	window.RegisterHook(events.Common.WindowClosing, func(*application.WindowEvent) {
		persistWindowSize()
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
