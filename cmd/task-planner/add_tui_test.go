package main

import (
	"errors"
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

func TestDuplicateSchedulesGetDistinctIDs(t *testing.T) {
	model := addModel{
		content:  "Plan the day",
		projects: []project{{ID: "inbox"}},
		weekdays: map[int16]bool{},
	}
	if first, second := model.toPlan(), model.toPlan(); first.ID == second.ID {
		t.Fatalf("duplicate schedules must have distinct IDs, got %q", first.ID)
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
	if !model.creating {
		t.Fatalf("final confirmation should start in-TUI creation: %#v", model)
	}
	model = updateAddModel(t, model, scheduleCreatedMsg{count: 3})

	if !model.completed || model.content != "Plan" || model.recurrence != "daily" || model.startDate.Format(time.DateOnly) != "2026-08-20" || model.endDate.Format(time.DateOnly) != "2026-08-22" || model.priority != 1 {
		t.Fatalf("unexpected final model: %#v", model)
	}
}

func TestAddModelShowsCreationFailureInsideTUI(t *testing.T) {
	model := addModel{creationError: errors.New("Todoist unavailable")}
	if view := model.View(); !strings.Contains(view, "Creation did not finish") {
		t.Fatalf("expected in-TUI creation error, got %s", view)
	}
}

func TestCompletedScheduleCanStartAnotherSchedule(t *testing.T) {
	model := addModel{
		completed:    true,
		createdTasks: 2,
		content:      "Plan the day",
		weekdays:     map[int16]bool{},
	}
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.completed || model.step != 0 || model.content != "" || model.createdTasks != 0 {
		t.Fatalf("choosing another schedule should reset the add flow: %#v", model)
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

func TestParseEndDateRelativeToStart(t *testing.T) {
	start := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	now := start.Add(12 * time.Hour)
	for value, want := range map[string]string{
		"1w": "2026-08-27",
		"2w": "2026-09-03",
		"1m": "2026-09-20",
		"2m": "2026-10-20",
	} {
		date, err := parseEndDate(value, start, now)
		if err != nil || date.Format(time.DateOnly) != want {
			t.Errorf("parseEndDate(%q) = %v, %v; want %s", value, date, err, want)
		}
	}
}

func TestParseEndDateKeepsAbsoluteDates(t *testing.T) {
	start := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	now := start.Add(12 * time.Hour)
	date, err := parseEndDate("22/8/26", start, now)
	if err != nil || date.Format(time.DateOnly) != "2026-08-22" {
		t.Fatalf("parseEndDate() = %v, %v", date, err)
	}
}

func TestFormatScheduleDate(t *testing.T) {
	for day, want := range map[int]string{
		1:  "1st of August 2026",
		2:  "2nd of August 2026",
		3:  "3rd of August 2026",
		4:  "4th of August 2026",
		11: "11th of August 2026",
		12: "12th of August 2026",
		13: "13th of August 2026",
		26: "26th of August 2026",
	} {
		date := time.Date(2026, time.August, day, 0, 0, 0, 0, time.UTC)
		if got := formatScheduleDate(date); got != want {
			t.Errorf("formatScheduleDate(%v) = %q, want %q", date, got, want)
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

func TestAddTextFieldsSupportCursorEditing(t *testing.T) {
	model := addModel{content: "10 abs", textCursor: len([]rune("10 abs")), weekdays: map[int16]bool{}}
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyLeft})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyLeft})
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	if model.content != "10 aXbs" {
		t.Fatalf("unexpected edited task text: %q", model.content)
	}
}

func TestAddFlowCanReturnToPreviousTextInput(t *testing.T) {
	model := addModel{step: 2, startInput: "20/8/26", endInput: "22/8/26", weekdays: map[int16]bool{}}
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyShiftTab})
	if model.step != 1 || model.startInput != "20/8/26" || model.endInput != "22/8/26" || model.textCursor != len([]rune(model.startInput)) {
		t.Fatalf("going back should reopen the previous populated input: %#v", model)
	}
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyBackspace})
	if model.startInput != "20/8/2" {
		t.Fatalf("previous input should be editable: %#v", model)
	}
}

func TestBackspaceReturnsFromChoiceAndBeginningOfInput(t *testing.T) {
	choice := addModel{step: 8, projectIndex: 1, projects: []project{{ID: "inbox"}, {ID: "coding"}}}
	choice = updateAddModel(t, choice, tea.KeyMsg{Type: tea.KeyBackspace})
	if choice.step != 6 || choice.cursor != 1 {
		t.Fatalf("Backspace should return from confirmation to the selected project: %#v", choice)
	}
	input := addModel{step: 2, startInput: "20/8/26", endInput: "22/8/26", textCursor: 0}
	input = updateAddModel(t, input, tea.KeyMsg{Type: tea.KeyBackspace})
	if input.step != 1 || input.endInput != "22/8/26" || input.textCursor != len([]rune(input.startInput)) {
		t.Fatalf("Backspace at the beginning should return without deleting the input: %#v", input)
	}
	first := updateAddModel(t, addModel{content: "Plan", textCursor: 0}, tea.KeyMsg{Type: tea.KeyBackspace})
	if first.step != 0 || first.content != "Plan" {
		t.Fatalf("Backspace on the first screen should leave the task text intact: %#v", first)
	}
}

func TestAddFlowCanReviewEarlierChoices(t *testing.T) {
	for _, test := range []struct {
		name       string
		model      addModel
		wantStep   int
		wantCursor int
	}{
		{"repeat from end date", addModel{step: 3, endInput: "22/8/26"}, 2, len("22/8/26")},
		{"weekdays from repeat", addModel{step: 4, recurrence: "weekdays"}, 3, 2},
		{"priority from weekdays", addModel{step: 5, recurrence: "weekdays", weekdays: map[int16]bool{int16(time.Monday): true}}, 4, 0},
		{"priority from daily", addModel{step: 5, recurrence: "daily"}, 3, 0},
		{"project from priority", addModel{step: 6, priority: 3}, 5, 2},
		{"confirmation from project", addModel{step: 8, projectIndex: 2, projects: []project{{ID: "a"}, {ID: "b"}, {ID: "c"}}}, 6, 2},
		{"duplicate check from task text", addModel{step: 7, content: "Plan", duplicateCandidates: []plan{{Content: "Planning"}}}, 0, len("Plan")},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := test.model
			model.errorMessage = "old validation error"
			model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyShiftTab})
			if model.step != test.wantStep || model.errorMessage != "" {
				t.Fatalf("previous screen should open without a stale error: %#v", model)
			}
			cursor := model.cursor
			if model.step <= 2 {
				cursor = model.textCursor
			}
			if cursor != test.wantCursor {
				t.Fatalf("previous answer should be selected for editing: %#v", model)
			}
		})
	}
}

func TestAddFlowShowsPreviousShortcutAndWaitsDuringCreation(t *testing.T) {
	model := addModel{step: 8, creating: true}
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyShiftTab})
	if model.step != 8 || !model.creating {
		t.Fatalf("creation in progress must stay on its progress screen: %#v", model)
	}
	for _, step := range []int{1, 4, 7, 8} {
		view := (addModel{step: step, projects: []project{{ID: "inbox"}}, startDate: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), endDate: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), recurrence: "daily"}).View()
		if !strings.Contains(view, "Shift+Tab") || !strings.Contains(view, "Backspace") || !strings.Contains(view, "previous") {
			t.Fatalf("step %d does not show how to return: %s", step, view)
		}
	}
}

func TestAddFlowPreselectsConfiguredDestinationProject(t *testing.T) {
	model := updateAddModel(t, newAddModel(), projectsLoadedMsg{
		projects:         []project{{ID: "inbox", Name: "Inbox"}, {ID: "coding", Name: "Coding"}},
		defaultProjectID: "coding",
	})
	if model.step != 1 {
		t.Fatalf("projects should load before schedule details: %#v", model)
	}
	model.step = 5
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.step != 6 || model.cursor != 1 {
		t.Fatalf("the configured destination should be selected in the picker: %#v", model)
	}
}

func TestAddFlowKeepsChosenProjectWhenDetailsReload(t *testing.T) {
	model := addModel{projects: []project{{ID: "inbox"}, {ID: "coding"}}, projectIndex: 1}
	model = updateAddModel(t, model, projectsLoadedMsg{
		projects:         []project{{ID: "coding"}, {ID: "inbox"}},
		defaultProjectID: "inbox",
	})
	model.step = 5
	model = updateAddModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.cursor != 0 || model.projects[model.cursor].ID != "coding" {
		t.Fatalf("reloading details should keep the project chosen earlier: %#v", model)
	}
}

func TestAddFlowWarnsWhenConfiguredProjectIsUnavailable(t *testing.T) {
	model := updateAddModel(t, newAddModel(), projectsLoadedMsg{
		projects:         []project{{ID: "inbox", Name: "Inbox"}},
		defaultProjectID: "deleted",
	})
	model.step = 6
	if !strings.Contains(model.View(), "default destination project") {
		t.Fatalf("the picker should identify a stale configured project: %s", model.View())
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

func TestAddCompletionViewShowsScheduledRange(t *testing.T) {
	model := addModel{
		completed:    true,
		createdTasks: 2,
		content:      "Plan the day",
		startDate:    time.Date(2026, time.August, 26, 0, 0, 0, 0, time.UTC),
		endDate:      time.Date(2026, time.September, 5, 0, 0, 0, 0, time.UTC),
	}
	view := model.View()
	if !strings.Contains(view, "Scheduled from 26th of August 2026 till 5th of September 2026.") {
		t.Fatalf("completion view does not show scheduled range: %s", view)
	}
}

func updateAddModel(t *testing.T, model addModel, message tea.Msg) addModel {
	t.Helper()
	updated, _ := model.Update(message)
	return updated.(addModel)
}
