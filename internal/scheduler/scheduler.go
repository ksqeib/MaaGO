// Package scheduler triggers the task queue at one or more configured times.
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

// EntryStatus is the frontend-facing snapshot of one trigger time.
type EntryStatus struct {
	ID              string `json:"id"`
	Time            string `json:"time"`
	Weekdays        []int  `json:"weekdays"`
	Enabled         bool   `json:"enabled"`
	Force           bool   `json:"force"`
	NextRun         string `json:"nextRun"`
	NextRunUnix     int64  `json:"nextRunUnix"`
	SecondsUntilRun int64  `json:"secondsUntilRun"`
	LastTriggered   string `json:"lastTriggered"`
}

// Status is the frontend-facing schedule snapshot.
type Status struct {
	Enabled         bool          `json:"enabled"`
	NotifyBefore    bool          `json:"notifyBefore"`
	NotifyMinutes   int           `json:"notifyMinutes"`
	Entries         []EntryStatus `json:"entries"`
	NextRun         string        `json:"nextRun"`
	NextRunUnix     int64         `json:"nextRunUnix"`
	SecondsUntilRun int64         `json:"secondsUntilRun"`
	LastTriggered   string        `json:"lastTriggered"`
}

// Scheduler owns the timer loop.
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
	entries := append([]config.ScheduleEntry(nil), cfg.Entries...)
	s.mu.RUnlock()

	now := time.Now()
	status := Status{
		Enabled:       cfg.Enabled,
		NotifyBefore:  cfg.NotifyBefore,
		NotifyMinutes: cfg.NotifyMinutes,
		Entries:       make([]EntryStatus, 0, len(entries)),
	}
	var earliest time.Time
	lastTriggered := ""
	for _, entry := range entries {
		info := EntryStatus{
			ID:            entry.ID,
			Time:          entry.Time,
			Weekdays:      append([]int(nil), entry.Weekdays...),
			Enabled:       entry.Enabled,
			Force:         entry.Force,
			LastTriggered: entry.LastTriggered,
		}
		if entry.LastTriggered > lastTriggered {
			lastTriggered = entry.LastTriggered
		}
		if next, ok := nextOccurrence(entry, now); ok {
			info.NextRun = next.Format("2006-01-02 15:04")
			info.NextRunUnix = next.Unix()
			info.SecondsUntilRun = int64(time.Until(next).Seconds())
			if earliest.IsZero() || next.Before(earliest) {
				earliest = next
			}
		}
		status.Entries = append(status.Entries, info)
	}
	status.LastTriggered = lastTriggered
	if !earliest.IsZero() {
		status.NextRun = earliest.Format("2006-01-02 15:04")
		status.NextRunUnix = earliest.Unix()
		status.SecondsUntilRun = int64(time.Until(earliest).Seconds())
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
	date := now.Format("2006-01-02")
	entries := append([]config.ScheduleEntry(nil), cfg.Entries...)
	changed := false
	firedMinutes := map[string]bool{}
	notifiedMinutes := map[string]bool{}

	for index := range entries {
		entry := &entries[index]
		if !entry.Enabled {
			continue
		}
		hour, minute, ok := parseClock(entry.Time)
		if !ok {
			continue
		}
		target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		if !weekdayEnabled(entry.Weekdays, now) {
			continue
		}

		if cfg.NotifyBefore && cfg.NotifyMinutes > 0 {
			if next, ok := nextOccurrence(*entry, now); ok {
				notifyAt := next.Add(-time.Duration(cfg.NotifyMinutes) * time.Minute)
				notifyDate := next.Format("2006-01-02")
				if entry.LastNotified != notifyDate && sameMinute(now, notifyAt) {
					entry.LastNotified = notifyDate
					changed = true
					key := notifyDate + "|" + entry.Time
					if !notifiedMinutes[key] {
						notifiedMinutes[key] = true
						s.sendNotification(entry.Time, cfg.NotifyMinutes)
					}
				}
			}
		}

		if entry.LastTriggered == date || !sameMinute(now, target) {
			continue
		}
		entry.LastTriggered = date
		changed = true
		if firedMinutes[entry.Time] {
			continue
		}
		firedMinutes[entry.Time] = true
		s.trigger(entry)
	}

	if changed {
		s.mu.Lock()
		s.cfg.Entries = entries
		s.mu.Unlock()
		persistMarks(entries)
		s.publish()
	}
}

func (s *Scheduler) sendNotification(clock string, minutes int) {
	if s.notify == nil {
		return
	}
	appCfg, err := config.Load()
	if err != nil || !appCfg.Notification.SystemEnabled || !appCfg.Notification.OnSchedule {
		return
	}
	s.notify("MaaGO · 定时任务", fmt.Sprintf("任务将在 %d 分钟后开始（%s）", minutes, clock))
}

func (s *Scheduler) trigger(entry *config.ScheduleEntry) {
	s.runner.SystemLog("info", fmt.Sprintf("定时任务触发（%s）", entry.Time))
	if s.runner.Busy() {
		if entry.Force {
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
		return
	}
	if err := s.runner.Start(); err != nil {
		s.runner.SystemLog("error", fmt.Sprintf("定时任务启动失败：%v", err))
	}
}

func persistMarks(entries []config.ScheduleEntry) {
	byID := make(map[string]config.ScheduleEntry, len(entries))
	for _, entry := range entries {
		if entry.ID != "" {
			byID[entry.ID] = entry
		}
	}
	_ = config.Update(func(c *config.AppConfig) {
		for i := range c.Schedule.Entries {
			updated, ok := byID[c.Schedule.Entries[i].ID]
			if !ok {
				continue
			}
			c.Schedule.Entries[i].LastTriggered = updated.LastTriggered
			c.Schedule.Entries[i].LastNotified = updated.LastNotified
		}
	})
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

func nextOccurrence(entry config.ScheduleEntry, now time.Time) (time.Time, bool) {
	if !entry.Enabled {
		return time.Time{}, false
	}
	hour, minute, ok := parseClock(entry.Time)
	if !ok {
		return time.Time{}, false
	}

	for offset := 0; offset <= 8; offset++ {
		day := now.AddDate(0, 0, offset)
		candidate := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, now.Location())
		if !candidate.After(now) {
			continue
		}
		if weekdayEnabled(entry.Weekdays, candidate) {
			return candidate, true
		}
	}
	return time.Time{}, false
}
