package updater

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadResourceStagesFiles(t *testing.T) {
	if os.Getenv("MAAGO_TEST_NETWORK") == "" {
		t.Skip("set MAAGO_TEST_NETWORK=1 to download MaaResource")
	}
	root := t.TempDir()
	core := filepath.Join(root, "core")
	if err := os.MkdirAll(filepath.Join(root, "resource"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "resource", "version.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := New(core, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := service.DownloadResource(ctx); err != nil {
		t.Fatal(err)
	}
	status := service.Status()
	if status.Stage != StageReady || !status.NeedsRestart {
		t.Fatalf("unexpected status: %+v", status)
	}
	pending, ok, err := readPending(filepath.Join(root, ".maago-updates"))
	if err != nil || !ok {
		t.Fatalf("pending marker: ok=%v err=%v", ok, err)
	}
	if _, err := os.Stat(filepath.Join(pending.Staging, "version.json")); err != nil {
		t.Fatalf("staged resource is incomplete: %v", err)
	}
}
