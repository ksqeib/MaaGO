package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyPendingCoreSwapsAndBacksUp(t *testing.T) {
	root := t.TempDir()
	coreDir := filepath.Join(root, "core")
	updatesDir := filepath.Join(root, ".maago-updates")
	staging := filepath.Join(updatesDir, "core-v1", "core")

	mustWrite(t, filepath.Join(coreDir, "MaaCore.dll"), "old")
	mustWrite(t, filepath.Join(coreDir, "resource", "version.json"), "old")
	mustWrite(t, filepath.Join(staging, "MaaCore.dll"), "new")
	mustWrite(t, filepath.Join(staging, "resource", "version.json"), "new")
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
	if _, err := os.Stat(filepath.Join(updatesDir, "pending.json")); !os.IsNotExist(err) {
		t.Fatal("pending marker should be removed")
	}
}

func TestApplyPendingResourceMerges(t *testing.T) {
	root := t.TempDir()
	coreDir := filepath.Join(root, "core")
	updatesDir := filepath.Join(root, ".maago-updates")
	staging := filepath.Join(updatesDir, "resource-1", "resource")

	mustWrite(t, filepath.Join(coreDir, "resource", "version.json"), "old")
	mustWrite(t, filepath.Join(staging, "version.json"), "new")
	mustWrite(t, filepath.Join(staging, "tasks", "new.json"), "task")
	if err := writePending(updatesDir, pendingUpdate{Kind: "resource", Staging: staging}); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyPending(coreDir); err != nil {
		t.Fatal(err)
	}
	if data := mustRead(t, filepath.Join(coreDir, "resource", "version.json")); data != "new" {
		t.Fatalf("resource was not merged: %q", data)
	}
	if data := mustRead(t, filepath.Join(coreDir, "resource", "tasks", "new.json")); data != "task" {
		t.Fatalf("new resource file missing: %q", data)
	}
}

func TestReleaseForVersionAndMirror(t *testing.T) {
	release := releaseForVersion("6.20.1")
	if release.Tag != "v6.20.1" {
		t.Fatalf("unexpected tag: %s", release.Tag)
	}
	if release.AssetName != "MAA-v6.20.1-win-x64.zip" {
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
