package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"MaaGO/internal/config"
	"MaaGO/internal/corelog"
	"MaaGO/internal/maa"
)

// TestThemeRecognition is opt-in. It starts the game and runs Award to check
// whether the current game main-screen theme is recognised.
func TestThemeRecognition(t *testing.T) {
	if os.Getenv("MAAGO_TEST_THEME") == "" {
		t.Skip("set MAAGO_TEST_THEME=1")
	}
	address := os.Getenv("MAAGO_TEST_ADB_ADDRESS")
	if address == "" {
		t.Skip("set MAAGO_TEST_ADB_ADDRESS")
	}
	appData, err := os.MkdirTemp("", "maago-theme-user-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("APPDATA=%s", appData)
	t.Setenv("APPDATA", appData)
	if _, err := config.Reload(); err != nil {
		t.Fatal(err)
	}
	if err := config.Update(func(c *config.AppConfig) {
		c.Connection.Address = address
		c.Connection.AutoDetect = false
		c.Connection.TouchMode = "adb"
		c.Run.ClientType = "Official"
		c.Tasks = []config.TaskConfig{
			{ID: "award", Type: config.TaskAward, Enabled: true, Options: map[string]any{
				"dailyWeekly": true, "mail": true, "orundum": true, "mining": true, "specialAccess": true, "signIn": true, "freeGacha": false,
			}},
		}
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
	deadline := time.Now().Add(180 * time.Second)
	for time.Now().Before(deadline) {
		state := r.State()
		if state.Status == StatusSuccess || state.Status == StatusFailed || state.Status == StatusStopped {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	state := r.State()
	t.Logf("final state: %s · %s · %d/%d", state.Status, state.Message, state.Completed, state.Total)
	for _, entry := range r.Logs() {
		t.Logf("[%s] %s", entry.Level, entry.Message)
	}
	if state.Status != StatusSuccess {
		t.Fatalf("theme recognition still failing: %+v", state)
	}
}
