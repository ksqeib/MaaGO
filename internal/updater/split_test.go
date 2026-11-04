package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestApplyPendingBaseReplacesAndOverlayMerges(t *testing.T) {
	root := t.TempDir()
	core := filepath.Join(root, "core")
	resource := filepath.Join(root, "resource")
	updates := filepath.Join(root, ".maago-updates")

	writeFile(t, filepath.Join(resource, "old.txt"), "old")
	writeFile(t, filepath.Join(resource, "platform_diff", "PC", "keep.txt"), "keep")

	baseStage := filepath.Join(updates, "base-resource-1", "resource")
	writeFile(t, filepath.Join(baseStage, "version.json"), "{}")
	writeFile(t, filepath.Join(baseStage, "base.txt"), "base")
	writeFile(t, filepath.Join(baseStage, "platform_diff", "PC", "tasks.json"), "{}")
	if err := writePending(updates, pendingUpdate{Kind: "resource-base", Staging: baseStage, Version: "v9.9.9"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPending(core); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(resource, "old.txt")); !os.IsNotExist(err) {
		t.Fatal("base apply should replace stale files")
	}
	if _, err := os.Stat(filepath.Join(resource, "base.txt")); err != nil {
		t.Fatal("base file missing:", err)
	}
	if data, err := os.ReadFile(filepath.Join(resource, ".maago-base-version")); err != nil || string(data) != "v9.9.9" {
		t.Fatalf("base marker: %q err=%v", data, err)
	}

	overlayStage := filepath.Join(updates, "overlay-resource-1", "resource")
	writeFile(t, filepath.Join(overlayStage, "version.json"), "{}")
	writeFile(t, filepath.Join(overlayStage, "overlay.txt"), "overlay")
	writeFile(t, filepath.Join(overlayStage, "platform_diff", "PC", "tasks.json"), `{"a":1}`)
	if err := writePending(updates, pendingUpdate{Kind: "resource-overlay", Staging: overlayStage, Version: "abc1234"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPending(core); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(resource, "overlay.txt")); err != nil {
		t.Fatal("overlay file missing:", err)
	}
	if _, err := os.Stat(filepath.Join(resource, "base.txt")); err != nil {
		t.Fatal("overlay apply must keep base files")
	}
	if data, err := os.ReadFile(filepath.Join(resource, ".maago-overlay-version")); err != nil || string(data) != "abc1234" {
		t.Fatalf("overlay marker: %q err=%v", data, err)
	}
}

func TestCheckBaseAndOverlay(t *testing.T) {
	if os.Getenv("MAAGO_TEST_NETWORK") == "" {
		t.Skip("set MAAGO_TEST_NETWORK=1 to query GitHub")
	}
	root := t.TempDir()
	service := New(filepath.Join(root, "core"), nil)
	base, err := service.CheckBase(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("base=%s asset=%s size=%d", base.Tag, base.AssetName, base.Size)
	overlay, err := service.CheckOverlay(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("overlay=%s date=%s", overlay.Tag, overlay.Published)
}
