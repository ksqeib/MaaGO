package updater

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeZip(t *testing.T, zipPath string, entries map[string]string) {
	t.Helper()
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
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
}

func TestExtractCoreBundledBin(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "core.zip")
	writeZip(t, zipPath, map[string]string{
		"bin/MaaCore.dll":           "core",
		"bin/MaaUtils.dll":          "utils",
		"bin/opencv_world4_maa.dll": "opencv",
		"bin/MaaCore.pdb":           "symbols",
		"include/AsstCaller.h":      "header",
		"README.md":                 "readme",
	})

	staging := filepath.Join(root, "staging")
	if err := extractCore(zipPath, staging, "v0.1.1", func(Status) {}); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"MaaCore.dll":           "core",
		"MaaUtils.dll":          "utils",
		"opencv_world4_maa.dll": "opencv",
	} {
		if data := mustRead(t, filepath.Join(staging, name)); data != want {
			t.Fatalf("%s: unexpected content %q", name, data)
		}
	}
	for _, name := range []string{"MaaCore.pdb", "AsstCaller.h", "README.md", "include"} {
		if _, err := os.Stat(filepath.Join(staging, name)); !os.IsNotExist(err) {
			t.Fatalf("%s should not be extracted", name)
		}
	}
}

func TestExtractCoreFlatRoot(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "core.zip")
	writeZip(t, zipPath, map[string]string{
		"MaaCore.dll":  "core",
		"MaaUtils.dll": "utils",
		"MaaCore.pdb":  "symbols",
	})

	staging := filepath.Join(root, "staging")
	if err := extractCore(zipPath, staging, "v1", func(Status) {}); err != nil {
		t.Fatal(err)
	}
	if data := mustRead(t, filepath.Join(staging, "MaaCore.dll")); data != "core" {
		t.Fatalf("unexpected core content: %q", data)
	}
	if _, err := os.Stat(filepath.Join(staging, "MaaCore.pdb")); !os.IsNotExist(err) {
		t.Fatal("pdb should not be extracted")
	}
}
