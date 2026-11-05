package updater

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckForQueueTaskNoDownload(t *testing.T) {
	if os.Getenv("MAAGO_TEST_NETWORK") == "" {
		t.Skip("set MAAGO_TEST_NETWORK=1 to query GitHub")
	}
	root := t.TempDir()
	service := New(filepath.Join(root, "core"), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	result, err := service.CheckForQueueTask(ctx, QueueUpdateOptions{Core: true, Base: true, Overlay: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.CoreLatest == "" || result.BaseLatest == "" || result.OverlayLatest == "" {
		t.Fatalf("missing latest versions: %+v", result)
	}
	if result.Staged != "" {
		t.Fatalf("download was disabled but staged %q", result.Staged)
	}
	t.Logf("core=%s base=%s overlay=%s staged=%q message=%q", result.CoreLatest, result.BaseLatest, result.OverlayLatest, result.Staged, result.Message)
}
