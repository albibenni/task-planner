package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDeleteModelSearchesAndShowsPagedPlans(t *testing.T) {
	model := deleteModel{plans: []plan{{Content: "Plan the day", Period: "day"}, {Content: "Write report", Period: "week"}}}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("plan")})
	model = updated.(deleteModel)
	if model.query != "plan" || model.loading || command != nil {
		t.Fatalf("search should filter the fetched plans locally: %#v", model)
	}

	view := model.View()
	for _, expected := range []string{"Plan the day", "Page 1 of 1", "1 active plan(s)"} {
		if !strings.Contains(view, expected) {
			t.Errorf("picker view lacks %q", expected)
		}
	}
}

func TestDeleteModelRequiresConfirmation(t *testing.T) {
	model := deleteModel{plans: []plan{{Content: "Plan the day", ID: "plan-the-day"}}}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(deleteModel)
	if !model.confirming || model.selected == nil || model.cursor != 1 {
		t.Fatalf("delete should default to no confirmation: %#v", model)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	model = updated.(deleteModel)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(deleteModel)
	if !model.loading || command == nil {
		t.Fatalf("confirmed deletion did not start: %#v", model)
	}
}
