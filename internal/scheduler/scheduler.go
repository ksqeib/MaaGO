// Package scheduler triggers the task queue at a configured time.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"MaaGO/internal/config"
	"MaaGO/internal/runner"
)

// EmitFunc publishes a Wails event.
type EmitFunc func(name string, payload any)

// Status is the frontend-facing schedule snapshot.
type Status struct {
	Enabled         bool   `json:"enabled"`
	Time            string `json:"time"`
	Weekdays        []int  `json:"weekdays"`
	Force           bool   `json:"force"`
	NotifyBefore    bool   `json:"notifyBefore"`
	NotifyMinutes   int    `json:"notifyMinutes"`
	NextRun         string `json:"nextRun"`
	NextRunUnix     int64  `json:"nextRunUnix"`
	SecondsUntilRun int64  `json:"secondsUntilRun"`
	LastTriggered   string `json:"lastTriggered"`
}

// Scheduler owns the daily timer loop.
type Scheduler struct {
	mu     sync.RWMutex
	cfg    config.ScheduleConfig
	runner *runner.Runner
	emit   EmitFunc
	notify func(title, message string)
}

// New creates a scheduler. The loop starts with Start.
func New(r *runner.Runner, emit EmitFunc, notify func(title, message string)) *Scheduler {
	return &Scheduler{runner: r, emit: emit, notify: notify}
}

// Update replaces the schedule configuration.
func (s *Scheduler) Update(cfg config.ScheduleConfig) {
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
	s.publish()
}

// Status returns the current snapshot.
func (s *Scheduler) Status() Status {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()

	now := time.Now()
	next, ok := nextOccurrence(cfg, now)
	status := Status{
		Enabled:       cfg.Enabled,
		Time:          cfg.Time,
		Weekdays:      append([]int(nil), cfg.Weekdays...),
		Force:         cfg.Force,
		NotifyBefore:  cfg.NotifyBefore,
		NotifyMinutes: cfg.NotifyMinutes,
		LastTriggered: cfg.LastTriggered,
	}
	if ok {
		status.NextRun = next.Format("2006-01-02 15:04")
		status.NextRunUnix = next.Unix()
		status.SecondsUntilRun = int64(time.Until(next).Seconds())
	}
	return status
}

func (s *Scheduler) publish() {
	if s.emit != nil {
		s.emit("maago:schedule", s.Status())
	}
}

// Start runs the timer loop until ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	s.publish()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	s.check()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.check()
		}
	}
}

func (s *Scheduler) check() {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()
	if !cfg.Enabled {
		return
	}

	now := time.Now()
	hour, minute, ok := parseClock(cfg.Time)
	if !ok {
		return
	}

	date := now.Format("2006-01-02")
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	if cfg.NotifyBefore && cfg.NotifyMinutes > 0 && cfg.LastNotified != date {
		notifyAt := target.Add(-time.Duration(cfg.NotifyMinutes) * time.Minute)
		if sameMinute(now, notifyAt) && weekdayEnabled(cfg.Weekdays, now) {
			s.mu.Lock()
			s.cfg.LastNotified = date
			s.mu.Unlock()
			_ = config.Update(func(c *config.AppConfig) { c.Schedule.LastNotified = date })
			if s.notify != nil {
				appCfg, err := config.Load()
				if err == nil && appCfg.Notification.SystemEnabled && appCfg.Notification.OnSchedule {
					s.notify("MaaGO · 定时任务", fmt.Sprintf("任务将在 %d 分钟后开始", cfg.NotifyMinutes))
				}
			}
			s.publish()
		}
	}

	if cfg.LastTriggered == date || !sameMinute(now, target) || !weekdayEnabled(cfg.Weekdays, now) {
		return
	}

	s.mu.Lock()
	s.cfg.LastTriggered = date
	s.mu.Unlock()
	_ = config.Update(func(c *config.AppConfig) { c.Schedule.LastTriggered = date })

	s.runner.SystemLog("info", fmt.Sprintf("定时任务触发（%s）", cfg.Time))
	if s.runner.Busy() {
		if cfg.Force {
			s.runner.SystemLog("warn", "当前有任务运行，强制停止并重新开始")
			_ = s.runner.Stop()
			go func() {
				time.Sleep(2 * time.Second)
				if err := s.runner.Start(); err != nil {
					s.runner.SystemLog("error", fmt.Sprintf("定时任务启动失败：%v", err))
				}
			}()
		} else {
			s.runner.SystemLog("warn", "当前有任务运行，跳过本次定时启动")
		}
	} else if err := s.runner.Start(); err != nil {
		s.runner.SystemLog("error", fmt.Sprintf("定时任务启动失败：%v", err))
	}

	s.publish()
}

func parseClock(value string) (hour, minute int, ok bool) {
	_, err := fmt.Sscanf(value, "%d:%d", &hour, &minute)
	if err != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, false
	}
	return hour, minute, true
}

func sameMinute(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay() &&
		a.Hour() == b.Hour() && a.Minute() == b.Minute()
}

func weekdayEnabled(days []int, now time.Time) bool {
	if len(days) == 0 {
		return true
	}
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	for _, day := range days {
		if day == weekday {
			return true
		}
	}
	return false
}

func nextOccurrence(cfg config.ScheduleConfig, now time.Time) (time.Time, bool) {
	if !cfg.Enabled {
		return time.Time{}, false
	}
	hour, minute, ok := parseClock(cfg.Time)
	if !ok {
		return time.Time{}, false
	}

	for offset := 0; offset <= 8; offset++ {
		day := now.AddDate(0, 0, offset)
		candidate := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, now.Location())
		if !candidate.After(now) {
			continue
		}
		if weekdayEnabled(cfg.Weekdays, candidate) {
			return candidate, true
		}
	}
	return time.Time{}, false
}
