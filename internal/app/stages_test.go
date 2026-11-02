package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"MaaGO/internal/app"
	"MaaGO/internal/config"
	"MaaGO/internal/maa"
	"MaaGO/internal/stages"
)

func TestGetStageCatalogFromInstalledCore(t *testing.T) {
	coreDir, err := filepath.Abs(filepath.Join("..", "..", "core"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(coreDir, "MaaCore.dll")); err != nil {
		t.Skip("installed core is not available")
	}

	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	t.Setenv("MAAGO_CONFIG_DIR", appData)
	cacheDir := filepath.Join(appData, "MaaGO", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A fresh cache file keeps GetStageCatalog offline for this test.
	if err := os.WriteFile(filepath.Join(cacheDir, stages.ActivityCacheFile), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Reload(); err != nil {
		t.Fatal(err)
	}

	core := maa.NewAt(coreDir)
	if !core.Available() {
		t.Skipf("MaaCore unavailable: %v", core.Err())
	}
	if err := core.SetUserDir(filepath.Join(appData, "core")); err != nil {
		t.Fatal(err)
	}
	service := app.NewService(nil, nil, nil, core, nil)

	catalog, err := service.GetStageCatalog(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tabs) == 0 {
		t.Fatal("catalog has no tabs")
	}
	total := 0
	for _, tab := range catalog.Tabs {
		for _, group := range tab.Groups {
			total += len(group.Stages)
		}
	}
	if total == 0 {
		t.Fatal("catalog has no stages")
	}
}
