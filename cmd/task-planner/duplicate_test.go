package main

import "testing"

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
