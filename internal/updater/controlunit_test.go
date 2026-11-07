package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyOptionalCoreFilesKeepsRdpBridge(t *testing.T) {
	root := t.TempDir()
	oldCore := filepath.Join(root, "core")
	staging := filepath.Join(root, "staging")
	for _, dir := range []string{oldCore, staging} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(dir, name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(oldCore, "MaaCore.dll", "old core")
	write(oldCore, "MaaWin32ControlUnit.dll", "bridge")
	write(oldCore, "MaaWin32ControlUnit.official.dll", "official")
	write(staging, "MaaCore.dll", "new core")
	// The SDK package already contains the official control unit; the bridge
	// pair from the installed core must still win.
	write(staging, "MaaWin32ControlUnit.dll", "sdk official")

	if err := copyOptionalCoreFiles(oldCore, staging); err != nil {
		t.Fatal(err)
	}
	bridge, err := os.ReadFile(filepath.Join(staging, "MaaWin32ControlUnit.dll"))
	if err != nil || string(bridge) != "bridge" {
		t.Fatalf("bridge DLL was not preserved: %q err=%v", string(bridge), err)
	}
	official, err := os.ReadFile(filepath.Join(staging, "MaaWin32ControlUnit.official.dll"))
	if err != nil || string(official) != "official" {
		t.Fatalf("official fallback was not preserved: %q err=%v", string(official), err)
	}
}
