// Package runner executes the task queue.
//
// When MaaCore is available the queue is handed to the real C API; otherwise
// the runner falls back to a clearly labelled demo mode. The frontend contract
// (state events, task events and logs) is identical in both paths.
package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"MaaGO/internal/apppath"
	"MaaGO/internal/config"
	"MaaGO/internal/corelog"
	"MaaGO/internal/maa"
)

// Status is the lifecycle state of the queue.
type Status string

const (
	StatusIdle     Status = "idle"
	StatusRunning  Status = "running"
	StatusStopping Status = "stopping"
	StatusSuccess  Status = "success"
	StatusFailed   Status = "failed"
	StatusStopped  Status = "stopped"
)

// TaskStatus is the lifecycle state of a single task.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskSkipped   TaskStatus = "skipped"
	TaskError     TaskStatus = "error"
)

// LogEntry is a single line in the run log.
type LogEntry struct {
	ID        int64  `json:"id"`
	Time      string `json:"time"`
	Timestamp int64  `json:"timestamp"`
	Level     string `json:"level"` // info | success | warn | error
	TaskID    string `json:"taskId"`
	Message   string `json:"message"`
}

// TaskState describes one task in the queue.
type TaskState struct {
	TaskID   string     `json:"taskId"`
	Status   TaskStatus `json:"status"`
	Progress int        `json:"progress"`
}

// RunState is the snapshot the frontend renders.
type RunState struct {
	Status          Status `json:"status"`
	CurrentTaskID   string `json:"currentTaskId"`
	CurrentTaskName string `json:"currentTaskName"`
	Completed       int    `json:"completed"`
	Total           int    `json:"total"`
	Progress        int    `json:"progress"`
	StartedAt       string `json:"startedAt"`
	FinishedAt      string `json:"finishedAt"`
	LastError       string `json:"lastError"`
	Message         string `json:"message"`
	Demo            bool   `json:"demo"`
}

// EmitFunc publishes a Wails event.
type EmitFunc func(name string, payload any)

// FinishFunc runs after a queue finishes.
type FinishFunc func(state RunState)

// Runner owns the queue state, its log ring and the simulated execution.
type Runner struct {
	mu          sync.RWMutex
	state       RunState
	logs        []LogEntry
	nextID      int64
	cancel      context.CancelFunc
	emit        EmitFunc
	onFinish    FinishFunc
	notify      func(title, message string)
	logFile     *os.File
	core        *maa.Core
	coreLog     *corelog.Writer
	updateCheck UpdateCheckFunc
	coreSession bool

	logMu sync.Mutex
}

// New creates a runner. emit and onFinish may be nil in tests.
func New(emit EmitFunc, onFinish FinishFunc, notify func(title, message string), core *maa.Core, coreLog *corelog.Writer) *Runner {
	r := &Runner{
		state:    RunState{Status: StatusIdle, Demo: core == nil || !core.Available()},
		emit:     emit,
		onFinish: onFinish,
		notify:   notify,
		core:     core,
		coreLog:  coreLog,
		logs:     make([]LogEntry, 0, 256),
	}
	path := filepath.Join(apppath.LogsDir(), "run.log")
	if file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
		r.logFile = file
	}
	r.loadRecentLogs(path)
	return r
}

// loadRecentLogs hydrates the in-memory ring with the tail of the previous log
// so reopening the app does not present an empty log page.
func (r *Runner) loadRecentLogs(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return
	}
	const tailBytes = 256 * 1024
	offset := int64(0)
	if info.Size() > tailBytes {
		offset = info.Size() - tailBytes
		if _, err := file.Seek(offset, 0); err != nil {
			return
		}
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)
	entries := make([]LogEntry, 0, 512)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first && offset > 0 {
			// The seek may start in the middle of a line.
			first = false
			continue
		}
		first = false
		entry, ok := parseLogLine(line)
		if !ok {
			continue
		}
		entries = append(entries, entry)
	}
	if len(entries) > 500 {
		entries = entries[len(entries)-500:]
	}
	for index := range entries {
		entries[index].ID = int64(index + 1)
	}
	r.logs = entries
	r.nextID = int64(len(entries))
}

func parseLogLine(line string) (LogEntry, bool) {
	// Format: 15:04:05 [level] message
	if len(line) < 12 || line[8] != ' ' || line[9] != '[' {
		return LogEntry{}, false
	}
	end := strings.IndexByte(line[10:], ']')
	if end < 0 {
		return LogEntry{}, false
	}
	end += 10
	return LogEntry{
		Time:    line[:8],
		Level:   line[10:end],
		Message: strings.TrimSpace(line[end+1:]),
	}, true
}

// Close releases the run log file.
func (r *Runner) Close() {
	r.logMu.Lock()
	defer r.logMu.Unlock()
	if r.logFile != nil {
		_ = r.logFile.Close()
		r.logFile = nil
	}
}

// State returns a copy of the current run state.
func (r *Runner) State() RunState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

// Logs returns a copy of the log ring, oldest first.
func (r *Runner) Logs() []LogEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]LogEntry, len(r.logs))
	copy(out, r.logs)
	return out
}

// ClearLogs empties the in-memory log ring. The on-disk log is preserved.
func (r *Runner) ClearLogs() {
	r.mu.Lock()
	r.logs = r.logs[:0]
	r.mu.Unlock()
}

// Busy reports whether a queue is running.
func (r *Runner) Busy() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state.Status == StatusRunning || r.state.Status == StatusStopping
}

// Start runs every enabled task in queue order.
func (r *Runner) Start() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	tasks := enabledTasks(cfg.Tasks)
	if len(tasks) == 0 {
		return errors.New("没有启用的任务")
	}
	return r.start(tasks, cfg)
}

// StartTask runs a single task by ID, regardless of its enabled flag.
func (r *Runner) StartTask(taskID string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	for _, task := range cfg.Tasks {
		if task.ID == taskID {
			return r.start([]config.TaskConfig{task}, cfg)
		}
	}
	return fmt.Errorf("任务不存在: %s", taskID)
}

func (r *Runner) start(tasks []config.TaskConfig, cfg config.AppConfig) error {
	r.mu.Lock()
	if r.state.Status == StatusRunning || r.state.Status == StatusStopping {
		r.mu.Unlock()
		return errors.New("任务正在运行")
	}

	ctx, cancel := context.WithCancel(context.Background())
	if cfg.Run.RunDurationMinutes > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(cfg.Run.RunDurationMinutes)*time.Minute)
	}
	useCore := r.core != nil && r.core.Available()
	r.cancel = cancel
	now := time.Now()
	message := "MaaCore 运行模式"
	if !useCore {
		message = "演示模式：未找到可用的 MaaCore"
	}
	r.state = RunState{
		Status:    StatusRunning,
		Total:     len(tasks),
		StartedAt: now.Format(time.RFC3339),
		Progress:  0,
		Demo:      !useCore,
		Message:   message,
	}
	r.mu.Unlock()

	r.emitState(r.State())
	r.SystemLog("info", "========================================")
	r.SystemLog("info", fmt.Sprintf("开始执行任务队列，共 %d 个任务", len(tasks)))
	if useCore {
		r.SystemLog("info", fmt.Sprintf("MaaCore %s · %s", r.core.Version(), r.core.Dir()))
		go r.runCore(ctx, tasks, cfg)
	} else {
		r.SystemLog("warn", "当前为界面演示模式：未找到可用的 MaaCore，不会操作游戏。")
		go r.runQueue(ctx, tasks)
	}
	return nil
}

func (r *Runner) runQueue(ctx context.Context, tasks []config.TaskConfig) {
	defer r.finishRun()

	completed := 0
	stopped := false
	for index, task := range tasks {
		if ctx.Err() != nil {
			stopped = true
			break
		}

		name := taskDisplayName(task)
		r.mu.Lock()
		r.state.CurrentTaskID = task.ID
		r.state.CurrentTaskName = name
		r.mu.Unlock()

		r.emitTask(TaskState{TaskID: task.ID, Status: TaskRunning})
		r.emitState(r.State())
		r.SystemLog("info", fmt.Sprintf("[%d/%d] 开始：%s", index+1, len(tasks), name))

		steps := scriptFor(task)
		for stepIndex, step := range steps {
			if !sleepContext(ctx, step.Delay) {
				stopped = true
				break
			}
			r.SystemLog(step.Level, step.Message)

			progress := int(float64(index)/float64(len(tasks))*100 +
				float64(stepIndex+1)/float64(len(steps))*100/float64(len(tasks)))
			r.mu.Lock()
			r.state.Progress = min(progress, 99)
			r.mu.Unlock()
			r.emitState(r.State())
		}

		if stopped {
			r.emitTask(TaskState{TaskID: task.ID, Status: TaskSkipped})
			r.SystemLog("warn", fmt.Sprintf("已跳过：%s", name))
			break
		}

		completed++
		r.emitTask(TaskState{TaskID: task.ID, Status: TaskCompleted, Progress: 100})
		r.SystemLog("success", fmt.Sprintf("完成：%s", name))

		r.mu.Lock()
		r.state.Completed = completed
		r.state.Progress = completed * 100 / len(tasks)
		r.mu.Unlock()
		r.emitState(r.State())
	}

	status := StatusFailed
	message := "部分任务未完成"
	switch {
	case stopped || ctx.Err() == context.DeadlineExceeded:
		status = StatusStopped
		message = "任务已停止"
	case completed == len(tasks):
		status = StatusSuccess
		message = "全部任务完成"
	}
	r.complete(status, message, completed, len(tasks))
}

// complete publishes the final run state, persists history and notifies.
func (r *Runner) complete(status Status, message string, completed, total int) {
	r.mu.Lock()
	r.state.Status = status
	r.state.Message = message
	r.state.Completed = completed
	r.state.Total = total
	r.state.CurrentTaskID = ""
	r.state.CurrentTaskName = ""
	r.state.FinishedAt = time.Now().Format(time.RFC3339)
	if status == StatusSuccess {
		r.state.Progress = 100
	}
	state := r.state
	r.mu.Unlock()

	r.emitState(state)
	r.SystemLog("info", fmt.Sprintf("任务队列结束：%s（%d/%d）", state.Message, completed, total))
	r.SystemLog("info", "========================================")

	if !state.Demo && r.coreLog != nil && r.coreSession {
		r.coreSession = false
		started, _ := time.Parse(time.RFC3339, state.StartedAt)
		duration := time.Duration(0)
		if !started.IsZero() {
			duration = time.Since(started)
		}
		r.coreLog.EndSession(string(status), message, duration)
	}

	_ = config.Update(func(c *config.AppConfig) {
		started, _ := time.Parse(time.RFC3339, state.StartedAt)
		finished, _ := time.Parse(time.RFC3339, state.FinishedAt)
		seconds := 0
		if !started.IsZero() && !finished.IsZero() {
			seconds = int(finished.Sub(started).Seconds())
		}
		c.LastRun = config.LastRun{
			StartedAt:       state.StartedAt,
			FinishedAt:      state.FinishedAt,
			Status:          string(state.Status),
			DurationSeconds: seconds,
			CompletedTasks:  completed,
			TotalTasks:      total,
		}
	})

	r.NotifyForState(state)

	if r.onFinish != nil {
		r.onFinish(state)
	}
}

func (r *Runner) finishRun() {
	r.mu.Lock()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.mu.Unlock()
}

// Stop asks the running queue to stop at the next safe point.
func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state.Status != StatusRunning && r.state.Status != StatusStopping {
		return nil
	}
	r.state.Status = StatusStopping
	r.state.Message = "正在停止…"
	if r.cancel != nil {
		r.cancel()
	}
	state := r.state
	go func() {
		r.emitState(state)
		r.SystemLog("warn", "收到停止请求，正在停止任务…")
	}()
	return nil
}

// SystemLog appends a line that does not belong to a single task.
func (r *Runner) SystemLog(level, message string) {
	r.appendLog(LogEntry{
		Time:      time.Now().Format("15:04:05"),
		Timestamp: time.Now().UnixMilli(),
		Level:     level,
		Message:   message,
	})
}

func (r *Runner) appendLog(entry LogEntry) {
	r.mu.Lock()
	r.nextID++
	entry.ID = r.nextID
	r.logs = append(r.logs, entry)
	if len(r.logs) > 1000 {
		r.logs = append(r.logs[:0], r.logs[len(r.logs)-1000:]...)
	}
	emit := r.emit
	r.mu.Unlock()

	r.logMu.Lock()
	if r.logFile != nil {
		_, _ = fmt.Fprintf(r.logFile, "%s [%s] %s\n", entry.Time, entry.Level, entry.Message)
	}
	r.logMu.Unlock()

	if emit != nil {
		emit("maago:log", entry)
	}
}

func (r *Runner) emitState(state RunState) {
	if r.emit != nil {
		r.emit("maago:run", state)
	}
}

func (r *Runner) emitTask(state TaskState) {
	if r.emit != nil {
		r.emit("maago:task", state)
	}
}

// Notify sends a system notification when configured.
func (r *Runner) Notify(title, message string) {
	if r.notify != nil {
		r.notify(title, message)
	}
}

// SetNotify replaces the notification callback.
func (r *Runner) SetNotify(fn func(title, message string)) {
	r.mu.Lock()
	r.notify = fn
	r.mu.Unlock()
}

// NotifyForState applies the user's notification preferences to a finished run.
func (r *Runner) NotifyForState(state RunState) {
	cfg, err := config.Load()
	if err != nil || !cfg.Notification.SystemEnabled || r.notify == nil {
		return
	}
	switch {
	case state.Status == StatusSuccess && cfg.Notification.OnComplete:
		r.notify("MaaGO · 任务完成", fmt.Sprintf("%d/%d 个任务已完成（演示模式）", state.Completed, state.Total))
	case (state.Status == StatusFailed || state.Status == StatusStopped) && cfg.Notification.OnError:
		r.notify("MaaGO · 任务结束", fmt.Sprintf("%s（%d/%d）", state.Message, state.Completed, state.Total))
	}
}

func enabledTasks(tasks []config.TaskConfig) []config.TaskConfig {
	out := make([]config.TaskConfig, 0, len(tasks))
	for _, task := range tasks {
		if task.Enabled {
			out = append(out, task)
		}
	}
	return out
}

func taskDisplayName(task config.TaskConfig) string {
	if task.Name != "" {
		return task.Name
	}
	return defaultNames[task.Type]
}

var defaultNames = map[config.TaskType]string{
	config.TaskStartUp: "开始唤醒",
	config.TaskFight:   "理智作战",
	config.TaskInfrast: "基建换班",
	config.TaskRecruit: "自动公招",
	config.TaskMall:    "信用收支",
	config.TaskAward:   "领取奖励",
	config.TaskUpdate:  "资源更新",
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
