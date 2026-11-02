package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"MaaGO/internal/apppath"
)

func TestReloadMigratesLegacyConfigAndSeries(t *testing.T) {
	appData := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("APPDATA", appData)
	t.Setenv("MAAGO_CONFIG_DIR", configDir)

	legacyPath := filepath.Join(appData, "MaaGO", "config.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := AppConfig{
		ConfigVersion: 6,
		WindowWidth:   1100,
		WindowHeight:  720,
		Theme:         ThemeSystem,
		Language:      "zh_CN",
		Layout:        LayoutSidebar,
		Tasks: []TaskConfig{{
			ID:      "task-fight",
			Type:    TaskFight,
			Enabled: true,
			Options: map[string]any{
				"stages": []any{"CE-6"},
				"series": "1",
			},
		}},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := Reload()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ConfigVersion != ConfigVersion {
		t.Fatalf("config version = %d, want %d", loaded.ConfigVersion, ConfigVersion)
	}
	if loaded.Tasks[0].Options["series"] != "0" {
		t.Fatalf("series = %v, want auto (0)", loaded.Tasks[0].Options["series"])
	}

	newPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new config not written: %v", err)
	}
	if apppath.ConfigPath() != newPath {
		t.Fatalf("ConfigPath = %q, want %q", apppath.ConfigPath(), newPath)
	}
}
