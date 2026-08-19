package main

import (
	"testing"
	"time"
)

func TestPeriodKey(t *testing.T) {
	now := time.Date(2026, time.August, 19, 10, 0, 0, 0, time.UTC)
	cases := map[string]string{"day": "2026-08-19", "week": "2026-W34", "month": "2026-08"}
	for period, want := range cases {
		if got := periodKey(period, now); got != want {
			t.Errorf("periodKey(%q) = %q, want %q", period, got, want)
		}
	}
}

func TestTaskMarkerUsesStablePlanAndPeriodIdentity(t *testing.T) {
	p := plan{ID: "plan-the-day-a1b2c3d4"}
	if got, want := taskMarker(p, "2026-08-19"), "[task-planner:plan-the-day-a1b2c3d4:2026-08-19]"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
