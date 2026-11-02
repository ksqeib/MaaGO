// Package config owns the persisted application settings.
//
// The config is a single JSON document. Reads go through Load, writes go
// through Update so concurrent callers cannot lose each other's changes.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"MaaGO/internal/apppath"
)

// ConfigVersion is bumped when the on-disk shape changes.
const ConfigVersion = 7

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

// TaskType identifies a task in the queue.
type TaskType string

const (
	TaskStartUp TaskType = "startup"
	TaskFight   TaskType = "fight"
	TaskInfrast TaskType = "infrast"
	TaskRecruit TaskType = "recruit"
	TaskMall    TaskType = "mall"
	TaskAward   TaskType = "award"
)

// ValidTaskTypes is the set of tasks the trimmed UI supports.
var ValidTaskTypes = []TaskType{
	TaskStartUp,
	TaskFight,
	TaskInfrast,
	TaskRecruit,
	TaskMall,
	TaskAward,
}

// IsValidTaskType reports whether t is a supported task type.
func IsValidTaskType(t TaskType) bool {
	for _, candidate := range ValidTaskTypes {
		if candidate == t {
			return true
		}
	}
	return false
}

// TaskConfig is one entry in the task queue.
//
// Options is intentionally open-ended: each task type owns its option keys and
// the frontend renders them from the same metadata table. Keeping the queue
// order in the slice means no separate ordering field is needed.
type TaskConfig struct {
	ID      string         `json:"id"`
	Type    TaskType       `json:"type"`
	Name    string         `json:"name"`
	Enabled bool           `json:"enabled"`
	Options map[string]any `json:"options"`
}

// ConnectionConfig describes how MaaGO reaches the emulator or PC client.
type ConnectionConfig struct {
	Mode              string `json:"mode"` // adb | win32
	Address           string `json:"address"`
	AdbPath           string `json:"adbPath"`
	AutoDetect        bool   `json:"autoDetect"`
	Emulator          string `json:"emulator"`  // auto | mumu | ld | bluestacks | nox | memu | custom
	TouchMode         string `json:"touchMode"` // minitouch | maatouch | adb
	ScreenshotEnhance bool   `json:"screenshotEnhance"`
	ClientType        string `json:"clientType"`
}

// RunConfig groups options that apply to the whole queue.
type RunConfig struct {
	ClientType                string `json:"clientType"`
	BlockSleep                bool   `json:"blockSleep"`
	KeepScreenOn              bool   `json:"keepScreenOn"`
	RunDurationMinutes        int    `json:"runDurationMinutes"`
	PostAction                string `json:"postAction"`
	PostActionOnlyWhenNoOther bool   `json:"postActionOnlyWhenNoOther"`
	PostActionSkipOnError     bool   `json:"postActionSkipOnError"`
}

// ScheduleConfig controls the daily scheduled run.
type ScheduleConfig struct {
	Enabled       bool   `json:"enabled"`
	Time          string `json:"time"`
	Weekdays      []int  `json:"weekdays"` // 1=Monday ... 7=Sunday
	Force         bool   `json:"force"`
	NotifyBefore  bool   `json:"notifyBefore"`
	NotifyMinutes int    `json:"notifyMinutes"`
	LastTriggered string `json:"lastTriggered"` // yyyy-mm-dd
	LastNotified  string `json:"lastNotified"`  // yyyy-mm-dd
}

// NotificationConfig controls system notifications.
type NotificationConfig struct {
	SystemEnabled bool `json:"systemEnabled"`
	OnComplete    bool `json:"onComplete"`
	OnError       bool `json:"onError"`
	OnSchedule    bool `json:"onSchedule"`
}

// ProxyConfig controls outbound HTTP(S) traffic used by the updater.
type ProxyConfig struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

// UpdateConfig controls core/resource update downloads.
type UpdateConfig struct {
	Mirror        string `json:"mirror"`
	TargetVersion string `json:"targetVersion"`
}

// LastRun records the result of the most recent queue run.
type LastRun struct {
	StartedAt       string `json:"startedAt"`
	FinishedAt      string `json:"finishedAt"`
	Status          string `json:"status"`
	DurationSeconds int    `json:"durationSeconds"`
	CompletedTasks  int    `json:"completedTasks"`
	TotalTasks      int    `json:"totalTasks"`
}

// AppConfig is the persisted user configuration.
type AppConfig struct {
	ConfigVersion   int                `json:"configVersion"`
	WindowWidth     int                `json:"windowWidth"`
	WindowHeight    int                `json:"windowHeight"`
	WindowMaximised bool               `json:"windowMaximised"`
	MinimizeToTray  bool               `json:"minimizeToTray"`
	CloseToTray     bool               `json:"closeToTray"`
	Theme           ThemeMode          `json:"theme"`
	AccentColor     string             `json:"accentColor"`
	Language        string             `json:"language"`
	Layout          LayoutMode         `json:"layout"`
	Connection      ConnectionConfig   `json:"connection"`
	Run             RunConfig          `json:"run"`
	Schedule        ScheduleConfig     `json:"schedule"`
	Notification    NotificationConfig `json:"notification"`
	Proxy           ProxyConfig        `json:"proxy"`
	Update          UpdateConfig       `json:"update"`
	Tasks           []TaskConfig       `json:"tasks"`
	LastRun         LastRun            `json:"lastRun"`
}

// Default returns the configuration used on a fresh install.
func Default() AppConfig {
	return AppConfig{
		ConfigVersion:   ConfigVersion,
		WindowWidth:     1100,
		WindowHeight:    720,
		WindowMaximised: false,
		MinimizeToTray:  true,
		CloseToTray:     true,
		Theme:           ThemeSystem,
		AccentColor:     "#10b981",
		Language:        "zh_CN",
		Layout:          LayoutSidebar,
		Connection: ConnectionConfig{
			Mode:       "adb",
			AutoDetect: true,
			Emulator:   "auto",
			TouchMode:  "adb",
			ClientType: "Official",
		},
		Run: RunConfig{
			ClientType:   "Official",
			BlockSleep:   true,
			KeepScreenOn: false,
			PostAction:   "none",
		},
		Schedule: ScheduleConfig{
			Enabled:       false,
			Time:          "04:00",
			Weekdays:      []int{1, 2, 3, 4, 5, 6, 7},
			Force:         false,
			NotifyBefore:  true,
			NotifyMinutes: 10,
		},
		Notification: NotificationConfig{
			SystemEnabled: true,
			OnComplete:    true,
			OnError:       true,
			OnSchedule:    true,
		},
		Proxy: ProxyConfig{
			Enabled: false,
			URL:     "",
		},
		Update: UpdateConfig{
			Mirror:        "",
			TargetVersion: "",
		},
		Tasks: defaultTasks(),
		LastRun: LastRun{
			Status: "idle",
		},
	}
}

func defaultTasks() []TaskConfig {
	return []TaskConfig{
		{ID: "task-startup", Type: TaskStartUp, Enabled: true, Options: map[string]any{
			"launchClient":  true,
			"clientType":    "Official",
			"accountSwitch": "",
		}},
		{ID: "task-fight", Type: TaskFight, Enabled: true, Options: map[string]any{
			"stage":              "",
			"stages":             []any{},
			"times":              0,
			"medicine":           false,
			"medicineCount":      1,
			"expiringMedicine":   true,
			"medicineExpireDays": 1,
			"stone":              false,
			"stoneCount":         1,
			"series":             "0",
		}},
		{ID: "task-infrast", Type: TaskInfrast, Enabled: true, Options: map[string]any{
			"mode":          "default",
			"plan":          "",
			"drone":         "Money",
			"threshold":     30,
			"facilities":    []any{"Mfg", "Trade", "Power", "Control", "Reception", "Office", "Dorm"},
			"clueExchange":  true,
			"sendClue":      true,
			"receiveCredit": true,
		}},
		{ID: "task-recruit", Type: TaskRecruit, Enabled: true, Options: map[string]any{
			"maxTimes":     4,
			"expedited":    false,
			"confirm3":     true,
			"confirm4":     true,
			"confirm5":     false,
			"autoRefresh":  true,
			"strategy":     "default",
			"preserveTags": "",
		}},
		{ID: "task-mall", Type: TaskMall, Enabled: true, Options: map[string]any{
			"visitFriends":  true,
			"creditFight":   false,
			"shopping":      true,
			"highPriority":  "招聘许可;加急许可",
			"blacklist":     "碳;家具",
			"onlyDiscount":  false,
			"reserveCredit": true,
		}},
		{ID: "task-award", Type: TaskAward, Enabled: true, Options: map[string]any{
			"dailyWeekly":   true,
			"mail":          true,
			"orundum":       true,
			"mining":        true,
			"specialAccess": true,
			"signIn":        true,
			"freeGacha":     false,
		}},
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
	if os.IsNotExist(err) {
		if legacy := apppath.LegacyConfigPath(); legacy != "" && legacy != path {
			if legacyData, legacyErr := os.ReadFile(legacy); legacyErr == nil {
				data, err = legacyData, nil
				log.Printf("config: migrated %s to %s", legacy, path)
			}
		}
	}
	switch {
	case err == nil:
		data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
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
	if err := saveLocked(cfg); err != nil {
		log.Printf("config: persist normalized config: %v", err)
	}
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

// Normalize repairs values that would break the UI.
func (c *AppConfig) Normalize() {
	c.migrate()
	if c.WindowWidth < 1024 {
		c.WindowWidth = 1024
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
	if c.Language == "" {
		c.Language = Default().Language
	}

	if c.Connection.Mode == "" {
		c.Connection.Mode = "adb"
	}
	if c.Connection.Emulator == "" {
		c.Connection.Emulator = "auto"
	}
	if c.Connection.TouchMode == "" {
		c.Connection.TouchMode = "adb"
	}
	if c.Connection.ClientType == "" {
		c.Connection.ClientType = "Official"
	}

	if c.Run.ClientType == "" {
		c.Run.ClientType = "Official"
	}
	if c.Run.PostAction == "" {
		c.Run.PostAction = "none"
	}
	if c.Run.RunDurationMinutes < 0 {
		c.Run.RunDurationMinutes = 0
	}

	if c.Schedule.Time == "" {
		c.Schedule.Time = "04:00"
	}
	if len(c.Schedule.Weekdays) == 0 {
		c.Schedule.Weekdays = []int{1, 2, 3, 4, 5, 6, 7}
	}
	if c.Schedule.NotifyMinutes < 0 {
		c.Schedule.NotifyMinutes = 10
	}
	c.Proxy.URL = strings.TrimSpace(c.Proxy.URL)
	c.Update.Mirror = strings.TrimSpace(c.Update.Mirror)
	c.Update.TargetVersion = strings.TrimSpace(c.Update.TargetVersion)

	seen := map[string]struct{}{}
	tasks := make([]TaskConfig, 0, len(c.Tasks))
	for _, task := range c.Tasks {
		if !IsValidTaskType(task.Type) {
			continue
		}
		if task.ID == "" {
			task.ID = "task-" + string(task.Type)
		}
		if _, exists := seen[task.ID]; exists {
			task.ID = fmt.Sprintf("%s-%d", task.ID, len(tasks)+1)
		}
		seen[task.ID] = struct{}{}
		if task.Options == nil {
			task.Options = map[string]any{}
		}
		tasks = append(tasks, task)
	}
	c.Tasks = tasks

	if c.LastRun.Status == "" {
		c.LastRun.Status = "idle"
	}
}

// migrate upgrades the old window-only config written before the task queue
// existed. Fields that a user could legitimately set to false are only filled
// for pre-v2 documents.
func (c *AppConfig) migrate() {
	if c.ConfigVersion >= ConfigVersion {
		return
	}
	if c.ConfigVersion < 3 && c.Connection.TouchMode == "minitouch" {
		// ADB touch works on more emulators; users can switch to minitouch later.
		c.Connection.TouchMode = "adb"
	}
	defaults := Default()
	if len(c.Tasks) == 0 {
		c.Tasks = defaults.Tasks
	}
	if !c.Notification.SystemEnabled && !c.Notification.OnComplete &&
		!c.Notification.OnError && !c.Notification.OnSchedule {
		c.Notification = defaults.Notification
	}
	if c.Schedule.Time == "" {
		c.Schedule = defaults.Schedule
	}
	if c.Connection.Mode == "" {
		c.Connection = defaults.Connection
	}
	if c.Run.PostAction == "" {
		c.Run = defaults.Run
	}
	if c.ConfigVersion < 5 {
		for i := range c.Tasks {
			task := &c.Tasks[i]
			if task.Type != TaskFight {
				continue
			}
			if task.Options == nil {
				task.Options = map[string]any{}
			}
			fill := func(key string, value any) {
				if _, ok := task.Options[key]; !ok {
					task.Options[key] = value
				}
			}
			fill("medicineCount", 1)
			fill("expiringMedicine", true)
			fill("medicineExpireDays", 1)
			fill("stoneCount", 1)
			// Up to v4 "medicine" meant "use a normal sanity potion". MaaCore
			// consumes normal potions before applying the expiring-medicine
			// window, so keep normal potions opt-in and default to expiring ones.
			task.Options["medicine"] = false
		}
	}
	if c.ConfigVersion < 6 {
		for i := range c.Tasks {
			task := &c.Tasks[i]
			if task.Type != TaskFight {
				continue
			}
			if task.Options == nil {
				task.Options = map[string]any{}
			}
			if _, ok := task.Options["stages"]; !ok {
				legacy := ""
				if value, ok := task.Options["stage"].(string); ok {
					legacy = strings.TrimSpace(value)
				}
				if legacy != "" {
					task.Options["stages"] = []any{legacy}
				} else {
					task.Options["stages"] = []any{}
				}
			}
		}
	}
	if c.ConfigVersion < 7 {
		for i := range c.Tasks {
			task := &c.Tasks[i]
			if task.Type != TaskFight {
				continue
			}
			if task.Options == nil {
				task.Options = map[string]any{}
			}
			series, ok := task.Options["series"]
			if !ok {
				task.Options["series"] = "0"
				continue
			}
			// Up to v6 the UI default was 1x, which makes MaaCore fight one
			// round at a time. 0 lets MaaCore adjust the auto-repeat count
			// according to the current sanity.
			switch value := series.(type) {
			case string:
				if value == "1" {
					task.Options["series"] = "0"
				}
			case float64:
				if value == 1 {
					task.Options["series"] = "0"
				}
			case int:
				if value == 1 {
					task.Options["series"] = "0"
				}
			}
		}
	}
	c.MinimizeToTray = defaults.MinimizeToTray
	c.CloseToTray = defaults.CloseToTray
	c.ConfigVersion = ConfigVersion
}
