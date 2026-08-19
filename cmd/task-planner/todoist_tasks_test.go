package main

import (
	"testing"
	"time"
)

func TestOccurrences(t *testing.T) {
	start := time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)
	p := plan{StartDate: start, EndDate: start.AddDate(0, 0, 6), Recurrence: "alternate"}
	if got := len(occurrences(p)); got != 4 {
		t.Fatalf("alternate schedule created %d occurrences, want 4", got)
	}
	p.Recurrence, p.Weekdays = "weekdays", []int16{int16(time.Monday), int16(time.Friday)}
	if got := len(occurrences(p)); got != 2 {
		t.Fatalf("weekday schedule created %d occurrences, want 2", got)
	}
}

func TestTaskMarkerUsesStableScheduleAndDateIdentity(t *testing.T) {
	p := plan{ID: "plan-the-day-a1b2c3d4"}
	dueDate := time.Date(2026, time.August, 19, 0, 0, 0, 0, time.UTC)
	if got, want := taskMarker(p, dueDate), "[task-planner:plan-the-day-a1b2c3d4:2026-08-19]"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
