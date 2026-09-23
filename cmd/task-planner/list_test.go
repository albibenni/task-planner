package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestListSchedulesShowsEveryScheduleFromDatabase(t *testing.T) {
	schedules := []plan{
		{Content: "Plan the day", StartDate: time.Date(2026, time.September, 23, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, time.September, 25, 0, 0, 0, 0, time.UTC)},
		{Content: "Write report", StartDate: time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)},
	}
	findAllCalls := 0
	findAll := func() ([]plan, error) {
		findAllCalls++
		return schedules, nil
	}
	var output bytes.Buffer

	if err := listSchedules(&output, findAll); err != nil {
		t.Fatal(err)
	}

	if findAllCalls != 1 {
		t.Fatalf("find all called %d times, want 1", findAllCalls)
	}
	for _, expected := range []string{
		"Plan the day — 2026-09-23 to 2026-09-25",
		"Write report — 2026-10-01 to 2026-10-01",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("list output lacks %q: %s", expected, output.String())
		}
	}
}
