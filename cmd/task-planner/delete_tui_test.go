package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDeleteModelSearchesAndShowsPagedPlans(t *testing.T) {
	model := deleteModel{plans: []plan{{Content: "Plan the day"}, {Content: "Write report"}}}
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
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(deleteModel)
	if model.cursor != 0 {
		t.Fatalf("up should select deletion confirmation: %#v", model)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	model = updated.(deleteModel)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(deleteModel)
	if !model.loading || command == nil {
		t.Fatalf("confirmed deletion did not start: %#v", model)
	}
}

func TestDeleteCompletionCanStartAnotherDeletion(t *testing.T) {
	model := deleteModel{
		query:        "Plan",
		plans:        []plan{{ID: "deleted", Content: "Plan the day"}},
		selected:     &plan{ID: "deleted", Content: "Plan the day"},
		confirming:   true,
		deletedTasks: 3,
	}
	updated, _ := model.Update(deleteCompletedMsg{deletedTasks: 3})
	model = updated.(deleteModel)
	if !model.done || !strings.Contains(model.View(), "Do you want to delete another plan?") {
		t.Fatalf("successful deletion should offer another deletion: %#v, %s", model, model.View())
	}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(deleteModel)
	if model.done || !model.loading || model.query != "" || model.selected != nil || model.deletedTasks != 0 || command == nil {
		t.Fatalf("another deletion should reopen a clean picker and reload plans: %#v", model)
	}
	updated, _ = model.Update(deletePlansLoadedMsg{plans: []plan{{ID: "remaining", Content: "Write report"}}})
	model = updated.(deleteModel)
	if model.loading || len(model.plans) != 1 || model.plans[0].ID != "remaining" {
		t.Fatalf("the next picker should show freshly loaded plans: %#v", model)
	}
}

func TestDeleteCompletionCanClose(t *testing.T) {
	model := deleteModel{done: true, selected: &plan{Content: "Plan the day"}, deletedTasks: 2}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	model = updated.(deleteModel)
	if model.cursor != 1 {
		t.Fatalf("No should select closing the delete TUI: %#v", model)
	}
	_, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if command == nil {
		t.Fatal("closing should quit the delete TUI")
	}
	if _, ok := command().(tea.QuitMsg); !ok {
		t.Fatalf("closing should quit rather than reload plans: %T", command())
	}
}

func TestPastScheduleOffersDatabaseOnlyDeletion(t *testing.T) {
	old := plan{ID: "old", Content: "Old plan", EndDate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	model := deleteModel{plans: []plan{old}}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(deleteModel)
	view := model.View()
	if model.cursor != 2 || !strings.Contains(view, "database only") || !strings.Contains(view, "Todoist tasks stay") {
		t.Fatalf("past schedules should offer a database-only choice and default to keeping the schedule: %#v, %s", model, view)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(deleteModel)
	if model.cursor != 1 {
		t.Fatalf("one step up should select database-only deletion: %#v", model)
	}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(deleteModel)
	if !model.loading || command == nil {
		t.Fatalf("database-only deletion should start after explicit selection: %#v", model)
	}
	updated, _ = model.Update(deleteCompletedMsg{databaseOnly: true})
	model = updated.(deleteModel)
	if !model.done || !strings.Contains(model.View(), "Todoist tasks were left unchanged") {
		t.Fatalf("completion should identify database-only deletion: %#v, %s", model, model.View())
	}
}

func TestDatabaseOnlyChoiceIsLimitedToPastSchedules(t *testing.T) {
	future := plan{ID: "future", Content: "Future plan", EndDate: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)}
	model := deleteModel{plans: []plan{future}}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(deleteModel)
	if model.cursor != 1 || strings.Contains(model.View(), "database only") {
		t.Fatalf("ongoing schedules should keep the original two-choice confirmation: %#v, %s", model, model.View())
	}
}

func TestPastScheduleUsesCalendarDate(t *testing.T) {
	today := time.Date(2026, 9, 15, 23, 0, 0, 0, time.FixedZone("Rome", 2*60*60))
	sameDay := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	previousDay := sameDay.AddDate(0, 0, -1)
	if scheduleIsPast(sameDay, today) || !scheduleIsPast(previousDay, today) {
		t.Fatal("a schedule is past only when its end date is before today's calendar date")
	}
}
