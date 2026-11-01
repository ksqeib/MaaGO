package updater

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractCoreSelectsRequiredFiles(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "core.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entries := map[string]string{
		"MAA-v1/MaaCore.dll":              "core",
		"MAA-v1/MaaUtils.dll":             "utils",
		"MAA-v1/README.txt":               "skip",
		"MAA-v1/resource/version.json":    "resource",
		"MAA-v1/resource/tasks/task.json": "task",
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
	if data := mustRead(t, filepath.Join(staging, "resource", "version.json")); data != "resource" {
		t.Fatalf("unexpected resource content: %q", data)
	}
	if _, err := os.Stat(filepath.Join(staging, "README.txt")); !os.IsNotExist(err) {
		t.Fatal("unneeded files should not be extracted")
	}
}
