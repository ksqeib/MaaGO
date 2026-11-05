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

func TestApplyPendingBasePackageReplacesAndOverlayMerges(t *testing.T) {
	root := t.TempDir()
	core := filepath.Join(root, "core")
	base := filepath.Join(root, "base")
	resource := filepath.Join(base, "resource")
	updates := filepath.Join(root, ".maago-updates")

	writeFile(t, filepath.Join(base, "MaaCore.dll"), "old core")
	writeFile(t, filepath.Join(base, "MAA.exe"), "old gui")
	writeFile(t, filepath.Join(resource, "old.txt"), "old")

	baseStage := filepath.Join(updates, "base-1", "base")
	writeFile(t, filepath.Join(baseStage, "MaaCore.dll"), "official core")
	writeFile(t, filepath.Join(baseStage, "MAA.exe"), "official gui")
	writeFile(t, filepath.Join(baseStage, "resource", "version.json"), "{}")
	writeFile(t, filepath.Join(baseStage, "resource", "base.txt"), "base")
	writeFile(t, filepath.Join(baseStage, "resource", "platform_diff", "PC", "tasks.json"), "{}")
	if err := writePending(updates, pendingUpdate{Kind: "base-package", Staging: baseStage, Version: "v9.9.9"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPending(core); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(resource, "old.txt")); !os.IsNotExist(err) {
		t.Fatal("base apply should replace stale files")
	}
	if data := mustRead(t, filepath.Join(base, "MaaCore.dll")); data != "official core" {
		t.Fatalf("official core was not installed: %q", data)
	}
	if _, err := os.Stat(filepath.Join(resource, "base.txt")); err != nil {
		t.Fatal("base file missing:", err)
	}
	if data, err := os.ReadFile(filepath.Join(base, ".maago-base-version")); err != nil || string(data) != "v9.9.9" {
		t.Fatalf("base marker: %q err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(root, "base.backup")); !os.IsNotExist(err) {
		t.Fatal("backup should be removed after a successful install")
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

func TestApplyPendingOverlayRequiresBase(t *testing.T) {
	root := t.TempDir()
	core := filepath.Join(root, "core")
	updates := filepath.Join(root, ".maago-updates")
	overlayStage := filepath.Join(updates, "overlay-resource-1", "resource")
	writeFile(t, filepath.Join(overlayStage, "version.json"), "{}")
	if err := writePending(updates, pendingUpdate{Kind: "resource-overlay", Staging: overlayStage, Version: "abc1234"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPending(core); err == nil {
		t.Fatal("overlay without a base package must fail")
	}
}

func TestApplyPendingLegacyResourceBase(t *testing.T) {
	root := t.TempDir()
	core := filepath.Join(root, "core")
	updates := filepath.Join(root, ".maago-updates")
	base := filepath.Join(root, "base")
	writeFile(t, filepath.Join(base, "resource", "old.txt"), "old")

	staging := filepath.Join(updates, "base-resource-1", "resource")
	writeFile(t, filepath.Join(staging, "version.json"), "{}")
	writeFile(t, filepath.Join(staging, "new.txt"), "new")
	if err := writePending(updates, pendingUpdate{Kind: "resource-base", Staging: staging, Version: "v9.9.9"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPending(core); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, "resource", "old.txt")); !os.IsNotExist(err) {
		t.Fatal("legacy base apply should replace stale files")
	}
	if data, err := os.ReadFile(filepath.Join(base, ".maago-base-version")); err != nil || string(data) != "v9.9.9" {
		t.Fatalf("base marker: %q err=%v", data, err)
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

func TestVersionsRequiresOfficialCore(t *testing.T) {
	root := t.TempDir()
	service := New(filepath.Join(root, "core"), nil)
	writeFile(t, filepath.Join(root, "base", "resource", ".maago-base-version"), "v6.21.4")
	writeFile(t, filepath.Join(root, "base", "resource", "version.json"), "{}")
	if got := service.Versions().Base; got != "" {
		t.Fatalf("resource-only base must not report a base version, got %q", got)
	}

	writeFile(t, filepath.Join(root, "base", "MaaCore.dll"), "core")
	writeFile(t, filepath.Join(root, "base", ".maago-base-version"), "v6.21.5")
	if got := service.Versions().Base; got != "v6.21.5" {
		t.Fatalf("base version = %q, want v6.21.5", got)
	}
}
