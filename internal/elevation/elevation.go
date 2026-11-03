// Package elevation reports whether MaaGO runs with an elevated token and can
// relaunch itself through the UAC prompt.
package elevation

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// IsElevated reports whether the current process has Administrator rights.
func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// RestartAsAdmin starts a new elevated instance of MaaGO after a short delay and
// returns once the helper process was spawned. The delay lets the current
// instance release the single-instance lock before the new process starts.
// The caller is expected to quit right away.
func RestartAsAdmin() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	escaped := strings.ReplaceAll(executable, "'", "''")
	script := fmt.Sprintf(
		"Start-Sleep -Seconds 2; Start-Process -FilePath '%s' -Verb RunAs",
		escaped,
	)
	command := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Start(); err != nil {
		return fmt.Errorf("无法启动提权助手：%w", err)
	}
	return nil
}
