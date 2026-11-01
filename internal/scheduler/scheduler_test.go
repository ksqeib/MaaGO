package scheduler

import (
	"testing"
	"time"

	"MaaGO/internal/config"
)

func TestNextOccurrenceRespectsWeekdays(t *testing.T) {
	// 2026-11-01 is a Sunday.
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.Local)
	cfg := config.ScheduleConfig{
		Enabled:  true,
		Time:     "04:00",
		Weekdays: []int{1}, // Monday only
	}

	next, ok := nextOccurrence(cfg, now)
	if !ok {
		t.Fatal("expected a next occurrence")
	}
	if next.Weekday() != time.Monday {
		t.Fatalf("expected Monday, got %s", next.Weekday())
	}
	if next.Hour() != 4 || next.Minute() != 0 {
		t.Fatalf("unexpected time: %s", next)
	}
}

func TestNextOccurrenceDisabled(t *testing.T) {
	if _, ok := nextOccurrence(config.ScheduleConfig{Enabled: false}, time.Now()); ok {
		t.Fatal("disabled schedule should not have a next occurrence")
	}
}
