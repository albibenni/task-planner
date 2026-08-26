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
