package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDeleteOldPreviewsEveryPastScheduleWithoutSelection(t *testing.T) {
	model := oldDeleteModel{}
	if model.Init() == nil {
		t.Fatal("delete old should load past schedules immediately")
	}
	plans := []plan{
		{ID: "one", Content: "Old one", EndDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "two", Content: "Old two", EndDate: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)},
	}
	updated, _ := model.Update(oldPlansLoadedMsg{plans: plans})
	model = updated.(oldDeleteModel)
	view := model.View()
	for _, expected := range []string{"Old one", "Old two", "2 past schedules", "Enter review deleting all"} {
		if !strings.Contains(view, expected) {
			t.Errorf("preview lacks %q: %s", expected, view)
		}
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(oldDeleteModel)
	if model.page != 0 || model.confirming {
		t.Fatalf("navigation should only change the preview: %#v", model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(oldDeleteModel)
	if !model.confirming || model.cursor != 1 || !strings.Contains(model.View(), "Delete all 2") {
		t.Fatalf("enter should confirm all previewed schedules and default to No: %#v, %s", model, model.View())
	}
}

func TestDeleteOldRequiresExplicitBulkConfirmation(t *testing.T) {
	model := oldDeleteModel{plans: []plan{{ID: "one", Content: "Old one"}, {ID: "two", Content: "Old two"}}}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(oldDeleteModel)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(oldDeleteModel)
	if model.loading || command != nil || model.confirming {
		t.Fatalf("default No should leave every schedule untouched: %#v", model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(oldDeleteModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(oldDeleteModel)
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(oldDeleteModel)
	if !model.loading || command == nil {
		t.Fatalf("explicit Yes should start one bulk deletion: %#v", model)
	}
	updated, cancelCommand := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(oldDeleteModel)
	if cancelCommand != nil || !model.loading {
		t.Fatal("an already-confirmed bulk delete must remain visible until it completes")
	}
	updated, _ = model.Update(oldDeleteCompletedMsg{deletedCount: 2})
	model = updated.(oldDeleteModel)
	if !model.done || !strings.Contains(model.View(), "Deleted 2 past schedules") || !strings.Contains(model.View(), "Todoist tasks were left unchanged") {
		t.Fatalf("completion should report the bulk result: %#v, %s", model, model.View())
	}
	_, command = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if command == nil {
		t.Fatal("completion should close instead of asking to select another schedule")
	}
}

func TestDeleteOldPagesThroughFullPreview(t *testing.T) {
	plans := make([]plan, 21)
	for index := range plans {
		plans[index] = plan{ID: "old", Content: "Old plan"}
	}
	model := oldDeleteModel{plans: plans}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(oldDeleteModel)
	if model.page != 1 || command != nil || !strings.Contains(model.View(), "Page 2 of 3") {
		t.Fatalf("right should show the next part of the already loaded preview: %#v, %s", model, model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(oldDeleteModel)
	if model.page != 2 || !strings.Contains(model.View(), "Page 3 of 3") {
		t.Fatalf("the last preview page should be reachable: %#v, %s", model, model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(oldDeleteModel)
	if !strings.Contains(model.View(), "Delete all 21") {
		t.Fatalf("confirmation should cover the full preview, independent of page: %s", model.View())
	}
}
