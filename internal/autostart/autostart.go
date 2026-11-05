// Package autostart manages the per-user Windows Run entry for MaaGO.
package autostart

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName = "MaaGO"
)

// CommandLine returns the command stored in the Run key. The --autostart flag
// tells MaaGO to start hidden in the tray.
func CommandLine() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	executable = strings.ReplaceAll(executable, `"`, `\"`)
	return fmt.Sprintf(`"%s" --autostart`, executable), nil
}

// Set adds or removes the current user's Run entry.
func Set(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if !enabled {
		err := key.DeleteValue(valueName)
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}

	command, err := CommandLine()
	if err != nil {
		return err
	}
	return key.SetStringValue(valueName, command)
}

// Enabled reports whether the Run entry exists.
func Enabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer key.Close()
	_, _, err = key.GetStringValue(valueName)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Sync makes the registry match the persisted preference. It also refreshes
// the executable path after the app is moved or updated.
func Sync(enabled bool) error {
	return Set(enabled)
}
