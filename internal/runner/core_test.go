package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"MaaGO/internal/config"
	"MaaGO/internal/maa"
)

func TestWantsLaunchClient(t *testing.T) {
	tests := []struct {
		name  string
		tasks []config.TaskConfig
		want  bool
	}{
		{"no startup", []config.TaskConfig{{Type: config.TaskAward}}, false},
		{"default", []config.TaskConfig{{Type: config.TaskStartUp}}, true},
		{"explicit false", []config.TaskConfig{{Type: config.TaskStartUp, Options: map[string]any{"launchClient": false}}}, false},
		{"explicit true", []config.TaskConfig{{Type: config.TaskStartUp, Options: map[string]any{"launchClient": true}}}, true},
	}
	for _, test := range tests {
		if got := wantsLaunchClient(test.tasks); got != test.want {
			t.Errorf("%s: got %v, want %v", test.name, got, test.want)
		}
	}
}

func TestCoreRunnerRequiresAddress(t *testing.T) {
	appData, err := os.MkdirTemp("", "maago-runner-user-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPDATA", appData)
	t.Setenv("MAAGO_CONFIG_DIR", appData)
	if _, err := config.Reload(); err != nil {
		t.Fatal(err)
	}
	if err := config.Update(func(c *config.AppConfig) {
		c.Connection.Address = ""
		c.Connection.AutoDetect = false
		c.Tasks = []config.TaskConfig{{
			ID:      "task-startup",
			Type:    config.TaskStartUp,
			Enabled: true,
			Options: map[string]any{"launchClient": false},
		}}
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

	r := New(nil, nil, nil, core, nil)
	defer r.Close()
	if err := r.Start(); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if r.State().Status == StatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if state := r.State(); state.Status != StatusFailed {
		t.Fatalf("expected failure, got %+v", state)
	}
	if state := r.State(); state.Demo {
		t.Fatal("core path should not be marked as demo")
	}
}
