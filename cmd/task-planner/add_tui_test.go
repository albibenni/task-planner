package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSlug(t *testing.T) {
	if got, want := slug("Plan the day!"), "plan-the-day"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAddModelCollectsTextAndSelections(t *testing.T) {
	model := addModel{projects: []project{{ID: "coding", Name: "Coding"}}}
	for _, character := range "Plan" {
		model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{character}})
	}
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if !model.done || model.content != "Plan" || model.period != "week" || model.due != "today" || model.priority != 1 {
		t.Fatalf("unexpected final model: %#v", model)
	}
}

func updateAddModel(t *testing.T, model addModel, message tea.Msg) addModel {
	t.Helper()
	updated, _ := model.Update(message)
	return updated.(addModel)
}
