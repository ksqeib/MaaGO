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
	service := New(core, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	err := service.DownloadCore(ctx, "6.20.1")
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
}
