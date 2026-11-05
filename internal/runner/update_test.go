package runner

import (
	"testing"

	"MaaGO/internal/config"
)

func TestSplitUpdateTaskAndOptions(t *testing.T) {
	tasks := []config.TaskConfig{
		{ID: "update", Type: config.TaskUpdate, Options: map[string]any{"checkBase": false}},
		{ID: "startup", Type: config.TaskStartUp},
	}
	update, rest := splitUpdateTask(tasks)
	if update == nil || update.ID != "update" {
		t.Fatalf("update=%+v", update)
	}
	if len(rest) != 1 || rest[0].ID != "startup" {
		t.Fatalf("rest=%+v", rest)
	}
	options := updateTaskOptions(*update)
	if !options.Core || options.Base || !options.Overlay || !options.Download {
		t.Fatalf("options=%+v", options)
	}
}
