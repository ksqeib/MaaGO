package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"MaaGO/internal/apppath"
	"MaaGO/internal/config"
	"MaaGO/internal/corelog"
	"MaaGO/internal/maa"
	"MaaGO/internal/rdp"
)

// TestRealRDPRun is opt-in. It connects to a remote Windows machine over RDP,
// binds MaaCore to the local proxy window, and runs one StartUp task.
func TestRealRDPRun(t *testing.T) {
	if os.Getenv("MAAGO_TEST_RDP_RUN") == "" {
		t.Skip("set MAAGO_TEST_RDP_RUN=1 to drive the remote machine")
	}
	host := os.Getenv("MAAGO_TEST_RDP_HOST")
	user := os.Getenv("MAAGO_TEST_RDP_USER")
	pass := os.Getenv("MAAGO_TEST_RDP_PASS")
	if host == "" || user == "" || pass == "" {
		t.Skip("set MAAGO_TEST_RDP_HOST, MAAGO_TEST_RDP_USER and MAAGO_TEST_RDP_PASS")
	}

	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	t.Setenv("MAAGO_CONFIG_DIR", appData)
	if _, err := config.Reload(); err != nil {
		t.Fatal(err)
	}
	if err := config.Update(func(c *config.AppConfig) {
		c.Connection.Mode = "rdp"
		c.Connection.RDP.Host = host
		c.Connection.RDP.Port = 3389
		c.Connection.RDP.Username = user
		c.Connection.RDP.Password = pass
		c.Connection.RDP.Width = 1920
		c.Connection.RDP.Height = 1080
		c.Run.ClientType = "Official"
		task := config.TaskConfig{
			ID:      "task-real-rdp-startup",
			Type:    config.TaskStartUp,
			Enabled: true,
			Options: map[string]any{
				"launchClient": false,
				"clientType":   "Official",
			},
		}
		if strings.EqualFold(os.Getenv("MAAGO_TEST_RDP_TASK"), "award") {
			task = config.TaskConfig{
				ID:      "task-real-rdp-award",
				Type:    config.TaskAward,
				Enabled: true,
				Options: map[string]any{
					"dailyWeekly":   true,
					"mail":          true,
					"orundum":       true,
					"mining":        true,
					"specialAccess": true,
					"signIn":        true,
					"freeGacha":     false,
				},
			}
		}
		if strings.EqualFold(os.Getenv("MAAGO_TEST_RDP_TASK"), "recruit") {
			task = config.TaskConfig{
				ID:      "task-real-rdp-recruit",
				Type:    config.TaskRecruit,
				Enabled: true,
				Options: map[string]any{
					"maxTimes":     2,
					"expedited":    false,
					"confirm3":     true,
					"confirm4":     true,
					"confirm5":     true,
					"autoRefresh":  true,
					"strategy":     "default",
					"preserveTags": "",
				},
			}
		}
		c.Tasks = []config.TaskConfig{task}
	}); err != nil {
		t.Fatal(err)
	}

	coreLogs := corelog.New(apppath.CoreLogsDir())
	defer coreLogs.Close()

	dir, err := filepath.Abs(filepath.Join("..", "..", "core"))
	if err != nil {
		t.Fatal(err)
	}
	core := maa.NewAt(dir)
	if !core.Available() {
		t.Skipf("MaaCore unavailable: %v", core.Err())
	}
	userDir, err := os.MkdirTemp("", "maago-rdp-user-")
	if err != nil {
		t.Fatal(err)
	}
	if err := core.SetUserDir(userDir); err != nil {
		t.Fatalf("set user dir: %v", err)
	}

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
	if session := rdp.Current(); session != nil {
		paint, prints, stretch, empty, down, up, move := session.Stats()
		t.Logf("proxy stats: paint=%d print=%d stretch=%d empty=%d mouse=%d/%d/%d", paint, prints, stretch, empty, down, up, move)
		framePath := filepath.Join(os.TempDir(), "maago-rdp-frame.png")
		if err := session.SaveFrame(framePath); err == nil {
			t.Logf("rdp frame: %s", framePath)
		} else {
			t.Logf("save rdp frame: %v", err)
		}
	}
	for _, entry := range r.Logs() {
		t.Logf("[%s] %s", entry.Level, entry.Message)
	}
	if state.Status != StatusSuccess {
		t.Fatalf("RDP run did not succeed: %+v", state)
	}
	if state.Demo {
		t.Fatal("RDP run was marked as demo")
	}
}
