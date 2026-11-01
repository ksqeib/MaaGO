package updater

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractCoreRootLevel(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "core.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entries := map[string]string{
		"MaaCore.dll":              "core",
		"resource/version.json":    "resource",
		"resource/tasks/root.json": "task",
	}
	for name, content := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	staging := filepath.Join(root, "staging")
	if err := extractCore(zipPath, staging, "v1", func(Status) {}); err != nil {
		t.Fatal(err)
	}
	if data := mustRead(t, filepath.Join(staging, "MaaCore.dll")); data != "core" {
		t.Fatalf("unexpected core content: %q", data)
	}
	if data := mustRead(t, filepath.Join(staging, "resource", "tasks", "root.json")); data != "task" {
		t.Fatalf("unexpected resource content: %q", data)
	}
}
