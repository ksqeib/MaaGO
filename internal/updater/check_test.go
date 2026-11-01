package updater

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckLatest(t *testing.T) {
	if os.Getenv("MAAGO_TEST_NETWORK") == "" {
		t.Skip("set MAAGO_TEST_NETWORK=1 to query GitHub")
	}
	root := t.TempDir()
	core := filepath.Join(root, "core")
	service := New(core, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	info, err := service.Check(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.Tag == "" || info.AssetURL == "" {
		t.Fatalf("incomplete release info: %+v", info)
	}
	t.Logf("latest=%s asset=%s size=%d", info.Tag, info.AssetName, info.Size)
}
