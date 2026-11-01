// Package tray owns the Windows system tray icon and menu.
package tray

import (
	"sync"

	"MaaGO/internal/lifecycle"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	mu     sync.Mutex
	app    *application.App
	handle *application.SystemTray
	start  func()
	stop   func()

	showLabel  = "Show MaaGO"
	startLabel = "Start queue"
	stopLabel  = "Stop queue"
	quitLabel  = "Quit"
)

// Setup installs the tray icon. Menu labels are retranslated by SetLabels once
// the frontend knows which language it renders in.
func Setup(a *application.App, icon []byte, window application.Window, onStart, onStop func()) {
	mu.Lock()
	defer mu.Unlock()

	app = a
	start = onStart
	stop = onStop

	handle = a.SystemTray.New()
	handle.SetIcon(icon)
	handle.SetTooltip("MaaGO")
	handle.AttachWindow(window)
	handle.OnClick(lifecycle.Restore)
	handle.OnDoubleClick(lifecycle.Restore)
	applyMenu()
}

// SetLabels updates the menu labels for the current UI language.
func SetLabels(show, startText, stopText, quit string) {
	mu.Lock()
	defer mu.Unlock()

	showLabel = show
	startLabel = startText
	stopLabel = stopText
	quitLabel = quit
	applyMenu()
}

func applyMenu() {
	if app == nil || handle == nil {
		return
	}
	menu := app.NewMenu()
	menu.Add(showLabel).OnClick(func(_ *application.Context) {
		lifecycle.Restore()
	})
	menu.AddSeparator()
	menu.Add(startLabel).OnClick(func(_ *application.Context) {
		if start != nil {
			start()
		}
	})
	menu.Add(stopLabel).OnClick(func(_ *application.Context) {
		if stop != nil {
			stop()
		}
	})
	menu.AddSeparator()
	menu.Add(quitLabel).OnClick(func(_ *application.Context) {
		lifecycle.Quit()
	})
	handle.SetMenu(menu)
}
