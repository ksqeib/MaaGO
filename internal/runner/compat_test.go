package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"MaaGO/internal/config"
	"MaaGO/internal/corelog"
	"MaaGO/internal/maa"
)

// TestCompatRun is opt-in. It copies the currently configured task queue, drops
// the Fight task and runs the remaining daily tasks against the real device to
// check that theme/UI recognition works on the installed game version.
func TestCompatRun(t *testing.T) {
	if os.Getenv("MAAGO_TEST_COMPAT") == "" {
		t.Skip("set MAAGO_TEST_COMPAT=1")
	}
	address := os.Getenv("MAAGO_TEST_ADB_ADDRESS")
	if address == "" {
		t.Skip("set MAAGO_TEST_ADB_ADDRESS")
	}

	source := filepath.Join(os.Getenv("APPDATA"), "MaaGO", "config.json")
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read %s: %v", source, err)
	}
	appData, err := os.MkdirTemp("", "maago-compat-user-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(appData, "MaaGO"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appData, "MaaGO", "config.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPDATA", appData)
	t.Setenv("MAAGO_CONFIG_DIR", appData)
	t.Logf("APPDATA=%s", appData)
	if _, err := config.Reload(); err != nil {
		t.Fatal(err)
	}
	if err := config.Update(func(c *config.AppConfig) {
		c.Connection.Address = address
		c.Connection.AutoDetect = false
		c.Connection.TouchMode = "adb"
		filter := map[string]bool{}
		for _, name := range strings.Split(os.Getenv("MAAGO_TEST_COMPAT_TASKS"), ",") {
			if name = strings.TrimSpace(name); name != "" {
				filter[name] = true
			}
		}
		kept := make([]config.TaskConfig, 0, len(c.Tasks))
		for _, task := range c.Tasks {
			if len(filter) > 0 && !filter[string(task.Type)] {
				continue
			}
			if task.Type == config.TaskFight {
				if !filter[string(config.TaskFight)] {
					continue
				}
				if task.Options == nil {
					task.Options = map[string]any{}
				}
				task.Options["times"] = float64(1)
			}
			if task.Type == config.TaskRecruit && task.Options != nil {
				task.Options["maxTimes"] = float64(1)
			}
			if task.Type == config.TaskAward && os.Getenv("MAAGO_TEST_AWARD_NO_SIGNIN") != "" {
				if task.Options == nil {
					task.Options = map[string]any{}
				}
				task.Options["signIn"] = false
			}
			kept = append(kept, task)
		}
		c.Tasks = kept
	}); err != nil {
		t.Fatal(err)
	}

	dir, err := filepath.Abs(filepath.Join("..", "..", "core"))
	if err != nil {
		t.Fatal(err)
	}
	core := maa.NewAt(dir)
	if !core.Available() {
		t.Skipf("MaaCore unavailable: %v", core.Err())
	}
	if err := core.SetUserDir(filepath.Join(appData, "core")); err != nil {
		t.Fatal(err)
	}
	coreLogs := corelog.New(filepath.Join(appData, "logs"))
	defer coreLogs.Close()

	r := New(nil, nil, nil, core, coreLogs)
	defer r.Close()
	if err := r.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(12 * time.Minute)
	for time.Now().Before(deadline) {
		state := r.State()
		if state.Status == StatusSuccess || state.Status == StatusFailed || state.Status == StatusStopped {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	state := r.State()
	t.Logf("final state: %s · %s · %d/%d", state.Status, state.Message, state.Completed, state.Total)
	for _, entry := range r.Logs() {
		t.Logf("[%s] %s", entry.Level, entry.Message)
	}
	if state.Status != StatusSuccess {
		t.Fatalf("daily queue did not succeed: %+v", state)
	}
}
