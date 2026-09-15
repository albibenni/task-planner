package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDeleteOldOpensPastSchedulesWithoutSearch(t *testing.T) {
	model := oldDeleteModel{}
	if model.Init() == nil {
		t.Fatal("delete old should load past schedules immediately")
	}
	updated, _ := model.Update(oldPlansLoadedMsg{plans: []plan{{ID: "old", Content: "Old plan"}}, total: 1})
	model = updated.(oldDeleteModel)
	if view := model.View(); !strings.Contains(view, "Old plan") || strings.Contains(view, "Search task text") {
		t.Fatalf("past schedules should appear without a text search: %s", view)
	}
}

func TestDeleteOldRequiresDatabaseOnlyConfirmation(t *testing.T) {
	old := plan{ID: "old", Content: "Old plan", EndDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}
	model := oldDeleteModel{plans: []plan{old}, total: 1}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(oldDeleteModel)
	if !model.confirming || model.cursor != 1 || !strings.Contains(model.View(), "Todoist tasks") {
		t.Fatalf("confirmation should default to keeping the schedule and explain database-only deletion: %#v, %s", model, model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(oldDeleteModel)
	if model.loading {
		t.Fatal("default confirmation must not delete")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(oldDeleteModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(oldDeleteModel)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(oldDeleteModel)
	if !model.loading || command == nil {
		t.Fatalf("explicit confirmation should start database-only removal: %#v", model)
	}
}

func TestDeleteOldCanRepeatOrClose(t *testing.T) {
	selected := plan{ID: "old", Content: "Old plan"}
	model := oldDeleteModel{selected: &selected, loading: true}
	updated, _ := model.Update(oldDeleteCompletedMsg{})
	model = updated.(oldDeleteModel)
	if !model.done || !strings.Contains(model.View(), "delete another past schedule") {
		t.Fatalf("after deletion the user should choose whether to continue: %#v, %s", model, model.View())
	}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(oldDeleteModel)
	if model.done || !model.loading || command == nil {
		t.Fatalf("continuing should reload past schedules: %#v", model)
	}
	model = oldDeleteModel{done: true, selected: &selected, cursor: 1}
	_, command = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if command == nil {
		t.Fatal("closing should quit")
	}
}

func TestDeleteOldPagesAndRecoversWhenLastPageShrinks(t *testing.T) {
	first := make([]plan, deletePageSize)
	for index := range first {
		first[index] = plan{ID: "old", Content: "Old plan"}
	}
	model := oldDeleteModel{plans: first, total: 11}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(oldDeleteModel)
	if model.page != 1 || !model.loading || command == nil {
		t.Fatalf("right should load the next page: %#v", model)
	}
	updated, _ = model.Update(oldPlansLoadedMsg{page: 1, total: 11, plans: []plan{{ID: "last", Content: "Last old plan"}}})
	model = updated.(oldDeleteModel)
	if !strings.Contains(model.View(), "Page 2 of 2") || !strings.Contains(model.View(), "Last old plan") {
		t.Fatalf("second page should show its schedule: %s", model.View())
	}
	updated, command = model.Update(oldPlansLoadedMsg{page: 1, total: 10})
	model = updated.(oldDeleteModel)
	if model.page != 0 || !model.loading || command == nil {
		t.Fatalf("a page emptied by another deletion should return to the last available page: %#v", model)
	}
}
