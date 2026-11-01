package maa

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveADB finds an adb executable. An explicit path from settings wins, then
// the bundled core/adb, then common emulator install locations and PATH.
func ResolveADB(coreDir, configured string) string {
	if value := cleanPath(configured); value != "" {
		if isFile(value) {
			return value
		}
	}

	candidates := []string{
		filepath.Join(coreDir, "adb", "adb.exe"),
	}

	if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
		candidates = append(candidates,
			filepath.Join(programFiles, "Netease", "MuMuPlayer-12.0", "shell", "adb.exe"),
			filepath.Join(programFiles, "Netease", "MuMuPlayer", "adb.exe"),
			filepath.Join(programFiles, "BlueStacks_nxt", "HD-Adb.exe"),
			filepath.Join(programFiles, "Nox", "bin", "adb.exe"),
			filepath.Join(programFiles, "Microvirt", "MEmu", "adb.exe"),
		)
	}
	candidates = append(candidates,
		`C:\leidian\LDPlayer9\adb.exe`,
		`C:\LDPlayer\LDPlayer9\adb.exe`,
	)

	for _, candidate := range candidates {
		if isFile(candidate) {
			return candidate
		}
	}

	if path, err := exec.LookPath("adb.exe"); err == nil {
		return path
	}
	if path, err := exec.LookPath("adb"); err == nil {
		return path
	}
	return ""
}

func cleanPath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return os.ExpandEnv(value)
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// ADBDevices lists serials reported by `adb devices` that are ready.
func ADBDevices(adbPath string) []string {
	if adbPath == "" {
		return nil
	}
	output, err := exec.Command(adbPath, "devices").Output()
	if err != nil {
		return nil
	}
	serials := make([]string, 0, 2)
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Split(strings.TrimSpace(line), "\t")
		if len(fields) >= 2 && strings.TrimSpace(fields[1]) == "device" {
			if serial := strings.TrimSpace(fields[0]); serial != "" {
				serials = append(serials, serial)
			}
		}
	}
	return serials
}
