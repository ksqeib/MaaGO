package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"MaaGO/internal/apppath"
)

func TestNormalizePCPostActionAndGamePath(t *testing.T) {
	cfg := Default()
	cfg.Run.PostAction = "shutdown"
	cfg.Connection.RDP.GamePath = ""
	cfg.Normalize()
	if cfg.Run.PostAction != "none" {
		t.Fatalf("postAction=%q, want none", cfg.Run.PostAction)
	}
	if cfg.Connection.RDP.GamePath != DefaultRDPGamePath {
		t.Fatalf("gamePath=%q, want %q", cfg.Connection.RDP.GamePath, DefaultRDPGamePath)
	}
	cfg.Run.PostAction = "exitArknights"
	cfg.Normalize()
	if cfg.Run.PostAction != "exitArknights" {
		t.Fatalf("postAction=%q, want exitArknights", cfg.Run.PostAction)
	}
}

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
		Schedule: ScheduleConfig{
			Enabled:       true,
			Time:          "04:00",
			Weekdays:      []int{1, 2},
			Force:         true,
			NotifyBefore:  true,
			NotifyMinutes: 10,
			LastTriggered: "2026-11-01",
		},
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
	if len(loaded.Schedule.Entries) != 1 {
		t.Fatalf("schedule entries = %d, want 1", len(loaded.Schedule.Entries))
	}
	entry := loaded.Schedule.Entries[0]
	if entry.Time != "04:00" || !entry.Force || !entry.Enabled || entry.LastTriggered != "2026-11-01" {
		t.Fatalf("unexpected migrated entry: %+v", entry)
	}

	newPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new config not written: %v", err)
	}
	if apppath.ConfigPath() != newPath {
		t.Fatalf("ConfigPath = %q, want %q", apppath.ConfigPath(), newPath)
	}
}

func TestNormalizeCoreSource(t *testing.T) {
	cfg := Default()
	if cfg.Core.Source != CoreCustom {
		t.Fatalf("default core source = %q, want %q", cfg.Core.Source, CoreCustom)
	}
	cfg.Core.Source = CoreOfficial
	cfg.Normalize()
	if cfg.Core.Source != CoreOfficial {
		t.Fatalf("core source = %q, want %q", cfg.Core.Source, CoreOfficial)
	}
	cfg.Core.Source = "unsupported"
	cfg.Normalize()
	if cfg.Core.Source != CoreCustom {
		t.Fatalf("core source = %q, want %q", cfg.Core.Source, CoreCustom)
	}
}
