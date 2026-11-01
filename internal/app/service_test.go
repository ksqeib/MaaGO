package app_test

import (
	"testing"
	"time"

	"MaaGO/internal/app"
	"MaaGO/internal/config"
	"MaaGO/internal/runner"
	"MaaGO/internal/scheduler"
)

func TestStartQueueCompletesAndRecordsLastRun(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	if _, err := config.Reload(); err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if err := config.Update(func(c *config.AppConfig) {
		c.Tasks = []config.TaskConfig{{
			ID:      "task-test-startup",
			Type:    config.TaskStartUp,
			Enabled: true,
			Options: map[string]any{"launchClient": false},
		}}
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	r := runner.New(nil, nil, nil, nil, nil)
	defer r.Close()
	service := app.NewService(r, scheduler.New(r, nil, nil), nil, nil, nil)

	if err := service.StartQueue(); err != nil {
		t.Fatalf("start queue: %v", err)
	}

	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		state := service.GetRunState()
		if state.Status == runner.StatusSuccess {
			break
		}
		if state.Status == runner.StatusFailed || state.Status == runner.StatusStopped {
			t.Fatalf("unexpected final state: %s (%s)", state.Status, state.Message)
		}
		time.Sleep(50 * time.Millisecond)
	}

	state := service.GetRunState()
	if state.Status != runner.StatusSuccess {
		t.Fatalf("queue did not finish successfully: %+v", state)
	}
	if state.Completed != 1 || state.Total != 1 {
		t.Fatalf("unexpected task counters: %+v", state)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.LastRun.Status != string(runner.StatusSuccess) {
		t.Fatalf("last run not recorded: %+v", cfg.LastRun)
	}
	if len(service.GetLogs()) == 0 {
		t.Fatal("expected run logs")
	}
}
