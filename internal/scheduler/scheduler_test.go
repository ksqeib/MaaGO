package scheduler

import (
	"testing"
	"time"

	"MaaGO/internal/config"
)

func TestNextOccurrenceRespectsWeekdays(t *testing.T) {
	// 2026-11-01 is a Sunday.
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.Local)
	entry := config.ScheduleEntry{
		ID:       "morning",
		Time:     "04:00",
		Weekdays: []int{1}, // Monday only
		Enabled:  true,
	}

	next, ok := nextOccurrence(entry, now)
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
	entry := config.ScheduleEntry{Time: "04:00", Weekdays: []int{1, 2, 3, 4, 5, 6, 7}, Enabled: false}
	if _, ok := nextOccurrence(entry, time.Now()); ok {
		t.Fatal("disabled entry should not have a next occurrence")
	}
}

func TestStatusPicksEarliestEntry(t *testing.T) {
	scheduler := New(nil, nil, nil)
	scheduler.Update(config.ScheduleConfig{
		Enabled: true,
		Entries: []config.ScheduleEntry{
			{ID: "late", Time: "23:00", Weekdays: []int{1, 2, 3, 4, 5, 6, 7}, Enabled: true},
			{ID: "early", Time: "00:30", Weekdays: []int{1, 2, 3, 4, 5, 6, 7}, Enabled: true},
		},
	})
	status := scheduler.Status()
	if len(status.Entries) != 2 {
		t.Fatalf("entries = %d", len(status.Entries))
	}
	if status.NextRun == "" {
		t.Fatal("expected a next run")
	}
	earliest := int64(0)
	for _, entry := range status.Entries {
		if entry.NextRunUnix == 0 {
			continue
		}
		if earliest == 0 || entry.NextRunUnix < earliest {
			earliest = entry.NextRunUnix
		}
	}
	if earliest == 0 || status.NextRunUnix != earliest {
		t.Fatalf("NextRunUnix = %d, want earliest %d", status.NextRunUnix, earliest)
	}
}
