package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSlug(t *testing.T) {
	if got, want := slug("Plan the day!"), "plan-the-day"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAddModelCollectsDateRangeAndRecurrence(t *testing.T) {
	model := addModel{weekdays: map[int16]bool{}}
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Plan")})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateAddModel(t, model, similarPlansLoadedMsg{})
	model = updateAddModel(t, model, projectsLoadedMsg{projects: []project{{ID: "coding", Name: "Coding"}}})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("20/8/26")})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("22/8/26")})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if !model.done || model.content != "Plan" || model.recurrence != "daily" || model.startDate.Format(time.DateOnly) != "2026-08-20" || model.endDate.Format(time.DateOnly) != "2026-08-22" || model.priority != 1 {
		t.Fatalf("unexpected final model: %#v", model)
	}
}

func TestParseDateFormats(t *testing.T) {
	now := time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)
	for _, value := range []string{"2026-08-20", "20-08-2026", "20/08/2026", "20/8/26"} {
		date, err := parseDate(value, now)
		if err != nil || date.Format(time.DateOnly) != "2026-08-20" {
			t.Errorf("parseDate(%q) = %v, %v", value, date, err)
		}
	}
}

func TestWeekdayPickerStartsWithMonday(t *testing.T) {
	model := addModel{step: 4, weekdays: map[int16]bool{}}
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeySpace})
	if !model.weekdays[int16(time.Monday)] {
		t.Fatalf("the first weekday choice should select Monday: %#v", model.weekdays)
	}
}

func TestTaskTextChecksDuplicatesBeforeScheduleDetails(t *testing.T) {
	model := addModel{content: "Plan the day", weekdays: map[int16]bool{}}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(addModel)
	if model.step != 1 || !model.loading || command == nil {
		t.Fatalf("task text stage should start duplicate checking: %#v", model)
	}
}

func TestAddConfirmationViewsRender(t *testing.T) {
	duplicateView := (addModel{step: 7, content: "Plan my day", duplicateCandidates: []plan{{Content: "Plan the day"}}}).View()
	if !strings.Contains(duplicateView, "Possible") && !strings.Contains(duplicateView, "similar") {
		t.Fatalf("unexpected duplicate confirmation view: %s", duplicateView)
	}
	finalView := (addModel{step: 8, content: "Plan the day", projects: []project{{ID: "inbox"}}, startDate: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), endDate: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), recurrence: "daily"}).View()
	if !strings.Contains(finalView, "Create 1 Todoist task") {
		t.Fatalf("unexpected final confirmation view: %s", finalView)
	}
}

func updateAddModel(t *testing.T, model addModel, message tea.Msg) addModel {
	t.Helper()
	updated, _ := model.Update(message)
	return updated.(addModel)
}
