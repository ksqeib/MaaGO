package updater

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadCoreStages(t *testing.T) {
	if os.Getenv("MAAGO_TEST_DOWNLOAD_CORE") == "" {
		t.Skip("set MAAGO_TEST_DOWNLOAD_CORE=1 to download the core release")
	}
	root := os.Getenv("MAAGO_TEST_DOWNLOAD_DIR")
	if root == "" {
		root = t.TempDir()
	} else {
		_ = os.RemoveAll(root)
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	core := filepath.Join(root, "core")
	if err := os.MkdirAll(core, 0o755); err != nil {
		t.Fatal(err)
	}
	// The SDK archive only contains MaaCore/MaaUtils; runtime DLLs from the old
	// core must be carried over so the staged core stays loadable.
	if err := os.WriteFile(filepath.Join(core, "opencv_world4_maa.dll"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(core, "MaaWin32ControlUnit.dll"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := New(core, nil)
	service.skipRuntime = true
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	version := os.Getenv("MAAGO_TEST_CORE_VERSION")
	if version == "" {
		version = "0.1.4"
	}
	err := service.DownloadCore(ctx, version)
	status := service.Status()
	t.Logf("status=%+v err=%v", status, err)
	if err != nil {
		t.Fatal(err)
	}
	pending, ok, err := readPending(filepath.Join(root, ".maago-updates"))
	if err != nil || !ok {
		t.Fatalf("pending marker: ok=%v err=%v", ok, err)
	}
	if _, err := os.Stat(filepath.Join(pending.Staging, "MaaCore.dll")); err != nil {
		t.Fatalf("staged core is incomplete: %v", err)
	}
	for _, name := range []string{"deps.json", "opencv_world4_maa.dll", "MaaWin32ControlUnit.dll"} {
		if _, err := os.Stat(filepath.Join(pending.Staging, name)); err != nil {
			t.Fatalf("staged core is missing %s: %v", name, err)
		}
	}
}
