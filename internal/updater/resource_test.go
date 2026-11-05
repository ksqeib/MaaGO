package updater

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractBasePackageKeepsPackageLayout(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "base.zip")
	writeZip(t, zipPath, map[string]string{
		"MAA.exe":                     "gui",
		"MaaCore.dll":                 "core",
		"MaaUtils.dll":                "utils",
		"opencv_world4_maa.dll":       "opencv",
		"resource/version.json":       `{"last_updated":"2026-11-01"}`,
		"resource/tasks/startup.json": "{}",
	})

	destination := filepath.Join(root, "stage", "base")
	if err := extractPackage(zipPath, destination, func(Status) {}); err != nil {
		t.Fatal(err)
	}
	if data := mustRead(t, filepath.Join(destination, "MaaCore.dll")); data != "core" {
		t.Fatalf("MaaCore.dll: %q", data)
	}
	if data := mustRead(t, filepath.Join(destination, "resource", "version.json")); data == "" {
		t.Fatal("resource/version.json missing")
	}
	if _, err := os.Stat(filepath.Join(destination, "resource", "tasks", "startup.json")); err != nil {
		t.Fatal("resource tasks missing:", err)
	}
	if err := validateBasePackage(destination); err != nil {
		t.Fatal(err)
	}
}

func TestExtractBasePackageStripsTopLevelFolder(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "base.zip")
	writeZip(t, zipPath, map[string]string{
		"MAA-pkg/MAA.exe":               "gui",
		"MAA-pkg/MaaCore.dll":           "core",
		"MAA-pkg/resource/version.json": "{}",
	})

	destination := filepath.Join(root, "base")
	if err := extractPackage(zipPath, destination, func(Status) {}); err != nil {
		t.Fatal(err)
	}
	if data := mustRead(t, filepath.Join(destination, "MaaCore.dll")); data != "core" {
		t.Fatalf("MaaCore.dll: %q", data)
	}
	if _, err := os.Stat(filepath.Join(destination, "MAA-pkg")); !os.IsNotExist(err) {
		t.Fatal("top-level folder should be stripped")
	}
}

func TestValidateBasePackageRejectsResourceOnly(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "resource", "version.json"), "{}")
	if err := validateBasePackage(dir); err == nil {
		t.Fatal("resource-only staging must not pass base validation")
	}
}

func writeOrderedZip(t *testing.T, path string, entries [][2]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		target, err := writer.Create(entry[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := target.Write([]byte(entry[1])); err != nil {
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

func TestResourceArchiveRootIgnoresClientResourceCopy(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "resource.zip")
	writeOrderedZip(t, zipPath, [][2]string{
		{"MaaResource-main/resource/global/YoStarEN/resource/version.json", `{"last_updated":"2026-10-28"}`},
		{"MaaResource-main/resource/global/YoStarEN/resource/tasks/en.json", "en-task"},
		{"MaaResource-main/resource/version.json", `{"last_updated":"2026-11-04"}`},
		{"MaaResource-main/resource/tasks/root.json", "root-task"},
	})

	staging := filepath.Join(root, "staging")
	if err := extractResource(zipPath, staging, func(Status) {}); err != nil {
		t.Fatal(err)
	}
	if data := mustRead(t, filepath.Join(staging, "version.json")); data != `{"last_updated":"2026-11-04"}` {
		t.Fatalf("wrong version.json selected: %q", data)
	}
	if data := mustRead(t, filepath.Join(staging, "tasks", "root.json")); data != "root-task" {
		t.Fatalf("root tasks missing: %q", data)
	}
	if data := mustRead(t, filepath.Join(staging, "global", "YoStarEN", "resource", "tasks", "en.json")); data != "en-task" {
		t.Fatalf("global resource missing: %q", data)
	}
}

func TestExtractPackageIgnoresNestedResourceVersion(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "base.zip")
	writeOrderedZip(t, zipPath, [][2]string{
		{"MAA-pkg/resource/global/YoStarEN/resource/version.json", "old"},
		{"MAA-pkg/resource/version.json", "root"},
		{"MAA-pkg/MaaCore.dll", "core"},
	})

	destination := filepath.Join(root, "base")
	if err := extractPackage(zipPath, destination, func(Status) {}); err != nil {
		t.Fatal(err)
	}
	if data := mustRead(t, filepath.Join(destination, "resource", "version.json")); data != "root" {
		t.Fatalf("wrong resource version: %q", data)
	}
	if data := mustRead(t, filepath.Join(destination, "MaaCore.dll")); data != "core" {
		t.Fatalf("core missing: %q", data)
	}
}
