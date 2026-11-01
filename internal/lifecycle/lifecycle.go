// Package lifecycle owns window visibility and explicit-quit state.
package lifecycle

import (
	"sync"
	"sync/atomic"

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
