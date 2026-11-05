package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLayoutMovesLegacyCoreResource(t *testing.T) {
	root := t.TempDir()
	coreDir := filepath.Join(root, "core")
	mustWrite(t, filepath.Join(coreDir, "MaaCore.dll"), "core")
	mustWrite(t, filepath.Join(coreDir, "resource", "version.json"), "old")

	message, err := MigrateLayout(coreDir)
	if err != nil {
		t.Fatal(err)
	}
	if message == "" {
		t.Fatal("expected a migration message")
	}
	if data := mustRead(t, filepath.Join(root, "base", "resource", "version.json")); data != "old" {
		t.Fatalf("resource not moved: %q", data)
	}
	if _, err := os.Stat(filepath.Join(coreDir, "resource")); !os.IsNotExist(err) {
		t.Fatal("old resource directory should be gone")
	}

	message, err = MigrateLayout(coreDir)
	if err != nil {
		t.Fatal(err)
	}
	if message != "" {
		t.Fatalf("second migration should be a no-op, got %q", message)
	}
}

func TestMigrateLayoutMovesRootResourceToBase(t *testing.T) {
	root := t.TempDir()
	coreDir := filepath.Join(root, "core")
	mustWrite(t, filepath.Join(root, "resource", "version.json"), "old")

	message, err := MigrateLayout(coreDir)
	if err != nil {
		t.Fatal(err)
	}
	if message == "" {
		t.Fatal("expected a migration message")
	}
	if data := mustRead(t, filepath.Join(root, "base", "resource", "version.json")); data != "old" {
		t.Fatalf("resource not moved: %q", data)
	}
	if _, err := os.Stat(filepath.Join(root, "resource")); !os.IsNotExist(err) {
		t.Fatal("root resource directory should be gone")
	}
}

func TestApplyPendingCoreSwapsAndBacksUp(t *testing.T) {
	root := t.TempDir()
	coreDir := filepath.Join(root, "core")
	updatesDir := filepath.Join(root, ".maago-updates")
	staging := filepath.Join(updatesDir, "core-v1", "core")

	mustWrite(t, filepath.Join(coreDir, "MaaCore.dll"), "old")
	mustWrite(t, filepath.Join(root, "base", "resource", "version.json"), "resource")
	mustWrite(t, filepath.Join(staging, "MaaCore.dll"), "new")
	mustWrite(t, filepath.Join(staging, "MaaUtils.dll"), "utils")
	if err := writePending(updatesDir, pendingUpdate{Kind: "core", Staging: staging, Version: "v1"}); err != nil {
		t.Fatal(err)
	}

	message, err := ApplyPending(coreDir)
	if err != nil {
		t.Fatal(err)
	}
	if message == "" {
		t.Fatal("expected an applied message")
	}
	if data := mustRead(t, filepath.Join(coreDir, "MaaCore.dll")); data != "new" {
		t.Fatalf("core was not replaced: %q", data)
	}
	if data := mustRead(t, filepath.Join(coreDir+".backup", "MaaCore.dll")); data != "old" {
		t.Fatalf("backup was not created: %q", data)
	}
	if data := mustRead(t, filepath.Join(root, "base", "resource", "version.json")); data != "resource" {
		t.Fatalf("shared resource should be untouched: %q", data)
	}
	if _, err := os.Stat(filepath.Join(updatesDir, "pending.json")); !os.IsNotExist(err) {
		t.Fatal("pending marker should be removed")
	}
}

func TestApplyPendingResourceMerges(t *testing.T) {
	root := t.TempDir()
	coreDir := filepath.Join(root, "core")
	updatesDir := filepath.Join(root, ".maago-updates")
	staging := filepath.Join(updatesDir, "resource-1", "resource")

	mustWrite(t, filepath.Join(root, "base", "resource", "version.json"), "old")
	mustWrite(t, filepath.Join(staging, "version.json"), "new")
	mustWrite(t, filepath.Join(staging, "tasks", "new.json"), "task")
	if err := writePending(updatesDir, pendingUpdate{Kind: "resource", Staging: staging}); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyPending(coreDir); err != nil {
		t.Fatal(err)
	}
	if data := mustRead(t, filepath.Join(root, "base", "resource", "version.json")); data != "new" {
		t.Fatalf("resource was not merged: %q", data)
	}
	if data := mustRead(t, filepath.Join(root, "base", "resource", "tasks", "new.json")); data != "task" {
		t.Fatalf("new resource file missing: %q", data)
	}
}

func TestApplyPendingResourceCreatesDirectory(t *testing.T) {
	root := t.TempDir()
	coreDir := filepath.Join(root, "core")
	updatesDir := filepath.Join(root, ".maago-updates")
	staging := filepath.Join(updatesDir, "resource-1", "resource")

	mustWrite(t, filepath.Join(coreDir, "MaaCore.dll"), "core")
	mustWrite(t, filepath.Join(staging, "version.json"), "new")
	if err := writePending(updatesDir, pendingUpdate{Kind: "resource", Staging: staging}); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyPending(coreDir); err != nil {
		t.Fatal(err)
	}
	if data := mustRead(t, filepath.Join(root, "base", "resource", "version.json")); data != "new" {
		t.Fatalf("base resource was not installed: %q", data)
	}
}

func TestReleaseForVersionAndMirror(t *testing.T) {
	release := releaseForVersion("0.1.1")
	if release.Tag != "v0.1.1" {
		t.Fatalf("unexpected tag: %s", release.Tag)
	}
	if release.AssetName != "MaaCore-sdk-windows-x64-v0.1.1.zip" {
		t.Fatalf("unexpected asset: %s", release.AssetName)
	}
	got := withMirror("https://github.com/a", "https://mirror/")
	if got != "https://mirror/https://github.com/a" {
		t.Fatalf("unexpected mirror URL: %s", got)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestReleaseVersionNightly(t *testing.T) {
	body := "Nightly MaaCore SDK from upstream-sync nightly-20261105-528a563\nupstream-commit=528a563e27ee57175b1c80ab507fb0591cd7e6b4"
	if got := releaseVersion("nightly", body, "2026-11-05T08:43:14Z"); got != "nightly-20261105-528a563" {
		t.Fatalf("nightly version = %q", got)
	}
	if got := releaseVersion("nightly", "", "2026-11-05T08:43:14Z"); got != "nightly-20261105-084314" {
		t.Fatalf("nightly fallback version = %q", got)
	}
	if got := releaseVersion("v0.1.5", body, ""); got != "v0.1.5" {
		t.Fatalf("release version = %q", got)
	}
}

func TestNightlyVersionSelection(t *testing.T) {
	for _, value := range []string{"", "latest", "nightly", "nightly-20261105-528a563", "NIGHTLY"} {
		if !isNightlyVersion(value) {
			t.Fatalf("%q should resolve to nightly", value)
		}
	}
	if isNightlyVersion("0.1.5") || isNightlyVersion("v0.1.5") {
		t.Fatal("explicit releases must not resolve to nightly")
	}
	release := releaseForVersion("nightly")
	if release.Tag != "nightly" || release.AssetName != "MaaCore-sdk-windows-x64-nightly.7z" {
		t.Fatalf("unexpected nightly release: %+v", release)
	}
	if release.AssetURL != "https://github.com/ksqeib/MaaCoreBuild/releases/download/nightly/MaaCore-sdk-windows-x64-nightly.7z" {
		t.Fatalf("unexpected nightly URL: %s", release.AssetURL)
	}
}
