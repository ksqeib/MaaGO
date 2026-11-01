// Package lifecycle owns window visibility and explicit-quit state.
package lifecycle

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	quitRequested atomic.Bool
	windowMu      sync.RWMutex
	mainWindow    application.Window
)

// SetWindow records the main window once it has been created.
func SetWindow(window application.Window) {
	windowMu.Lock()
	mainWindow = window
	windowMu.Unlock()
}

// Window returns the main window, or nil before startup finishes.
func Window() application.Window {
	windowMu.RLock()
	defer windowMu.RUnlock()
	return mainWindow
}

// Restore shows and focuses the launcher window.
func Restore() {
	window := Window()
	if window == nil {
		return
	}
	window.Show()
	window.UnMinimise()
	window.Focus()
}

// Hide sends the launcher to the tray.
func Hide() {
	if window := Window(); window != nil {
		window.Hide()
	}
}

// QuitRequested reports whether the current shutdown comes from an explicit
// quit rather than the close button.
func QuitRequested() bool {
	return quitRequested.Load()
}

// Quit terminates the application.
func Quit() {
	quitRequested.Store(true)
	if app := application.Get(); app != nil {
		app.Quit()
	}
}

// Restart launches a new instance after a short delay and quits this one. The
// delay lets the single-instance lock be released before the new process starts.
func Restart() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	escaped := strings.ReplaceAll(executable, "'", "''")
	script := fmt.Sprintf("Start-Sleep -Seconds 3; Start-Process -FilePath '%s'", escaped)
	command := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Start(); err != nil {
		return err
	}
	Quit()
	return nil
}
