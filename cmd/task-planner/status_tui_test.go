package main

import (
	"strings"
	"testing"
)

func TestStatusModelShowsMissingSetupCommands(t *testing.T) {
	view := (statusModel{}).View()
	for _, expected := range []string{"Supabase database URL: missing", "Todoist login: missing", "task-planner config", "task-planner auth login"} {
		if !strings.Contains(view, expected) {
			t.Errorf("status view lacks %q", expected)
		}
	}
}

func TestStatusModelShowsReadyState(t *testing.T) {
	view := (statusModel{status: setupStatus{supabaseConfigured: true, todoistConfigured: true}}).View()
	if !strings.Contains(view, "Ready to add and schedule shared plans.") {
		t.Fatalf("unexpected ready view: %s", view)
	}
}
