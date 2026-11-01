// Package apppath centralizes every filesystem location used by MaaGO.
//
// Keeping path construction in one place means the application can later move
// its data directory without touching call sites, and makes it obvious which
// files are user data versus logs or cached assets.
package apppath

import (
	"os"
	"path/filepath"
	"strings"
)

// AppName is used for the per-user storage directory.
const AppName = "MaaGO"

// launcherDir returns the directory containing the running executable.
func launcherDir() string {
	exe, err := os.Executable()
	if err != nil {
		cwd, _ := os.Getwd()
		return cwd
	}
	return filepath.Dir(exe)
}

// appDataRoot returns %APPDATA% when available.
func appDataRoot() string {
	if v := strings.TrimSpace(os.Getenv("APPDATA")); v != "" {
		return v
	}
	if v, err := os.UserCacheDir(); err == nil && strings.TrimSpace(v) != "" {
		return v
	}
	return launcherDir()
}

// Root is the application's user data directory.
func Root() string {
	dir := filepath.Join(appDataRoot(), AppName)
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// ConfigPath is the location of config.json.
func ConfigPath() string {
	return filepath.Join(Root(), "config.json")
}

// LogsDir stores application logs.
func LogsDir() string {
	dir := filepath.Join(Root(), "logs")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// CacheDir stores disposable data such as downloaded resources.
func CacheDir() string {
	dir := filepath.Join(Root(), "cache")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// StartupLogPath is the log file written while the application boots.
func StartupLogPath() string {
	return filepath.Join(LogsDir(), "startup.log")
}

// CoreLogsDir stores the date-rotated MaaCore stdout logs.
func CoreLogsDir() string {
	dir := filepath.Join(LogsDir(), "core")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// DiagnosticsDir stores connection diagnostic screenshots.
func DiagnosticsDir() string {
	dir := filepath.Join(CacheDir(), "diagnostics")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// InstallDir is where the MaaCore runtime and resources are expected. It is a
// placeholder until the core runtime layout is decided.
func InstallDir() string {
	return launcherDir()
}
