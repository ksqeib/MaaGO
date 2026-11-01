// Package config owns the persisted application settings.
//
// The config is a single JSON document. Reads go through Load, writes go
// through Update so concurrent callers cannot lose each other's changes.
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"MaaGO/internal/apppath"
)

var (
	mu       sync.RWMutex
	cached   AppConfig
	isLoaded bool
)

// ThemeMode controls the appearance of the window.
type ThemeMode string

const (
	ThemeLight  ThemeMode = "light"
	ThemeDark   ThemeMode = "dark"
	ThemeSystem ThemeMode = "system"
)

// LayoutMode controls whether navigation is a side rail or a top bar.
type LayoutMode string

const (
	LayoutSidebar LayoutMode = "sidebar"
	LayoutNavbar  LayoutMode = "navbar"
)

// AppConfig is the persisted user configuration.
type AppConfig struct {
	WindowWidth     int        `json:"window_width"`
	WindowHeight    int        `json:"window_height"`
	WindowMaximised bool       `json:"window_maximised"`
	Theme           ThemeMode  `json:"theme"`
	AccentColor     string     `json:"accent_color"`
	Language        string     `json:"language"`
	Layout          LayoutMode `json:"layout"`
	MinimizeToTray  bool       `json:"minimize_to_tray"`
}

// Default returns the configuration used on a fresh install.
func Default() AppConfig {
	return AppConfig{
		WindowWidth:  1024,
		WindowHeight: 640,
		Theme:        ThemeSystem,
		AccentColor:  "#10b981",
		Language:     "",
		Layout:       LayoutSidebar,
	}
}

// Normalize repairs values that would break the UI.
func (c *AppConfig) Normalize() {
	if c.WindowWidth < 960 {
		c.WindowWidth = 960
	}
	if c.WindowHeight < 600 {
		c.WindowHeight = 600
	}
	if c.Theme == "" {
		c.Theme = ThemeSystem
	}
	if c.Layout == "" {
		c.Layout = LayoutSidebar
	}
	if c.AccentColor == "" {
		c.AccentColor = Default().AccentColor
	}
}

// Load returns the cached configuration, reading it from disk on first use.
func Load() (AppConfig, error) {
	mu.RLock()
	if isLoaded {
		c := cached
		mu.RUnlock()
		return c, nil
	}
	mu.RUnlock()

	return Reload()
}

// Reload discards the cache and reads config.json again.
func Reload() (AppConfig, error) {
	mu.Lock()
	defer mu.Unlock()
	return reloadLocked()
}

func reloadLocked() (AppConfig, error) {
	path := apppath.ConfigPath()
	cfg := Default()

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Printf("config: invalid %s: %v", path, err)
			return AppConfig{}, fmt.Errorf("config is corrupted: %w", err)
		}
	case os.IsNotExist(err):
		cfg.Normalize()
		if err := saveLocked(cfg); err != nil {
			return cfg, err
		}
	default:
		log.Printf("config: read %s: %v", path, err)
		return AppConfig{}, fmt.Errorf("read config: %w", err)
	}

	cfg.Normalize()
	cached = cfg
	isLoaded = true
	return cfg, nil
}

// Update atomically mutates and persists the configuration.
func Update(mutate func(*AppConfig)) error {
	mu.Lock()
	defer mu.Unlock()

	cfg := cached
	if !isLoaded {
		var err error
		if cfg, err = reloadLocked(); err != nil {
			return err
		}
	}
	mutate(&cfg)
	cfg.Normalize()
	return saveLocked(cfg)
}

func saveLocked(cfg AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(apppath.ConfigPath(), data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	cached = cfg
	isLoaded = true
	return nil
}
