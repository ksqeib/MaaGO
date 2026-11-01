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
)

// TestRealCoreRun is opt-in. It starts the game, runs one StartUp task and
// verifies that MaaCore stdout was captured into the dated core log.
func TestRealCoreRun(t *testing.T) {
	if os.Getenv("MAAGO_TEST_REAL_RUN") == "" {
		t.Skip("set MAAGO_TEST_REAL_RUN=1 to operate the device")
	}
	address := os.Getenv("MAAGO_TEST_ADB_ADDRESS")
	if address == "" {
		t.Skip("set MAAGO_TEST_ADB_ADDRESS")
	}

	t.Setenv("APPDATA", t.TempDir())
	if _, err := config.Reload(); err != nil {
		t.Fatal(err)
	}
	if err := config.Update(func(c *config.AppConfig) {
		c.Connection.Address = address
		c.Connection.AutoDetect = false
		c.Connection.Emulator = "auto"
		c.Connection.TouchMode = "adb"
		c.Run.ClientType = "Official"
		c.Tasks = []config.TaskConfig{{
			ID:      "task-real-startup",
			Type:    config.TaskStartUp,
			Enabled: true,
			Options: map[string]any{
				"launchClient": true,
				"clientType":   "Official",
			},
		}}
	}); err != nil {
		t.Fatal(err)
	}

	coreLogs := corelog.New(apppath.CoreLogsDir())
	defer coreLogs.Close()
	capture, err := maa.StartStdoutCapture(coreLogs)
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Stop()

	dir, err := filepath.Abs(filepath.Join("..", "..", "core"))
	if err != nil {
		t.Fatal(err)
	}
	core := maa.NewAt(dir)
	if !core.Available() {
		t.Skipf("MaaCore unavailable: %v", core.Err())
	}
	userDir, err := os.MkdirTemp("", "maago-core-user-")
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

	deadline := time.Now().Add(150 * time.Second)
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
		t.Fatalf("real run did not succeed: %+v", state)
	}
	if state.Demo {
		t.Fatal("real run was marked as demo")
	}

	path := coreLogs.CurrentPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read core log: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "MaaGO core session") {
		t.Fatal("core log is missing the session separator")
	}
	if !strings.Contains(text, "session ended") {
		t.Fatal("core log is missing the end separator")
	}
	if !strings.Contains(text, "[INF]") && !strings.Contains(text, "LoadResource") && !strings.Contains(text, "MaaCore") {
		t.Fatalf("core log has no MaaCore stdout lines:\n%s", text)
	}
}
