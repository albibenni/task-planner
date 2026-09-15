package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestCheckReportsPastSchedulesStillInDatabase(t *testing.T) {
	today := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	all := []plan{
		{Content: "Old plan", StartDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)},
		{Content: "Today plan", EndDate: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)},
		{Content: "Future plan", EndDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
	}
	var output bytes.Buffer
	reportPastSchedules(&output, all, today)
	got := output.String()
	for _, expected := range []string{"1 past schedule", "Old plan", "20 Aug 2026", "task-planner delete old", "Supabase only"} {
		if !strings.Contains(got, expected) {
			t.Errorf("check report lacks %q: %s", expected, got)
		}
	}
	for _, unexpected := range []string{"Today plan", "Future plan"} {
		if strings.Contains(got, unexpected) {
			t.Errorf("check report includes %q before it has ended: %s", unexpected, got)
		}
	}
}

func TestCheckReportsWhenNoPastSchedulesRemain(t *testing.T) {
	var output bytes.Buffer
	reportPastSchedules(&output, nil, time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC))
	if !strings.Contains(output.String(), "No past schedules remain in Supabase") {
		t.Fatalf("empty check report should be clear: %s", output.String())
	}
}
