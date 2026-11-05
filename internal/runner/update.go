package runner

import (
	"context"

	"MaaGO/internal/config"
)

// UpdateCheckOptions selects which components the resource-update task checks.
type UpdateCheckOptions struct {
	Core     bool
	Base     bool
	Overlay  bool
	Download bool
}

// UpdateCheckResult reports current/latest versions and the component staged
// for the next restart.
type UpdateCheckResult struct {
	CoreCurrent    string
	CoreLatest     string
	BaseCurrent    string
	BaseLatest     string
	OverlayCurrent string
	OverlayLatest  string
	Staged         string
	Message        string
}

// UpdateCheckFunc is provided by the app service because the runner does not
// own the updater.
type UpdateCheckFunc func(context.Context, UpdateCheckOptions) (UpdateCheckResult, error)

// SetUpdateCheck wires the resource-update task to the updater.
func (r *Runner) SetUpdateCheck(fn UpdateCheckFunc) {
	r.updateCheck = fn
}

func splitUpdateTask(tasks []config.TaskConfig) (*config.TaskConfig, []config.TaskConfig) {
	var update *config.TaskConfig
	rest := make([]config.TaskConfig, 0, len(tasks))
	for index := range tasks {
		task := tasks[index]
		if task.Type == config.TaskUpdate {
			if update == nil {
				copy := task
				update = &copy
			}
			continue
		}
		rest = append(rest, task)
	}
	return update, rest
}

func taskOptionBool(options map[string]any, key string, fallback bool) bool {
	value, ok := options[key]
	if !ok {
		return fallback
	}
	if enabled, ok := value.(bool); ok {
		return enabled
	}
	return fallback
}

func updateTaskOptions(task config.TaskConfig) UpdateCheckOptions {
	return UpdateCheckOptions{
		Core:     taskOptionBool(task.Options, "checkCore", true),
		Base:     taskOptionBool(task.Options, "checkBase", true),
		Overlay:  taskOptionBool(task.Options, "checkOverlay", true),
		Download: taskOptionBool(task.Options, "download", true),
	}
}
