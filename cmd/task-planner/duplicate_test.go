package main

import (
	"testing"
	"time"
)

func TestSimilarText(t *testing.T) {
	for _, pair := range [][2]string{{"Plan the day", "Plan my day"}, {"Review project tasks", "Review tasks for project"}, {"Write report", "Write the report"}} {
		if !similarText(pair[0], pair[1]) {
			t.Errorf("expected %q and %q to be similar", pair[0], pair[1])
		}
	}
	if similarText("Plan the day", "Plan weekly review") {
		t.Fatal("unrelated task text should not be similar")
	}
}

func TestPastSchedulesAreNotDuplicateCandidates(t *testing.T) {
	today := time.Date(2026, time.October, 17, 12, 0, 0, 0, time.FixedZone("Rome", 2*60*60))
	plans := []plan{{
		Content: "Plan day",
		EndDate: today.AddDate(0, 0, -1),
	}}

	if matches := duplicateCandidates("Plan day", plans, today); len(matches) != 0 {
		t.Fatalf("a schedule that ended before today should not be a duplicate candidate: %#v", matches)
	}
}

func TestCurrentSchedulesRemainDuplicateCandidates(t *testing.T) {
	today := time.Date(2026, time.October, 17, 23, 0, 0, 0, time.FixedZone("Rome", 2*60*60))
	plans := []plan{
		{ID: "ends-today", Content: "Plan day", EndDate: time.Date(2026, time.October, 17, 0, 0, 0, 0, time.UTC)},
		{ID: "ends-later", Content: "Plan day", EndDate: today.AddDate(0, 0, 1)},
	}

	matches := duplicateCandidates("Plan day", plans, today)
	if len(matches) != 2 || matches[0].ID != "ends-today" || matches[1].ID != "ends-later" {
		t.Fatalf("schedules ending today or later should remain duplicate candidates: %#v", matches)
	}
}
