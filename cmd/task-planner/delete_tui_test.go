package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDeleteSearchWaitsForTwoCharactersAndTypingPause(t *testing.T) {
	model := deleteModel{}
	if command := model.Init(); command != nil {
		t.Fatal("opening delete must wait for a search instead of loading every plan")
	}
	if !strings.Contains(model.View(), "Type at least 2 characters") {
		t.Fatalf("empty picker should explain when search begins: %s", model.View())
	}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	model = updated.(deleteModel)
	if command != nil || model.query != "p" || len(model.plans) != 0 {
		t.Fatalf("one character should not search Supabase: %#v", model)
	}
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	model = updated.(deleteModel)
	if command == nil || model.query != "pl" || !model.searchPending || model.searching {
		t.Fatalf("two characters should schedule a debounced search: %#v", model)
	}
	updated, searchCommand := model.Update(deleteSearchDueMsg{query: "pl", revision: model.searchRevision})
	model = updated.(deleteModel)
	if searchCommand == nil || !model.searching {
		t.Fatalf("search should begin only after the typing pause: %#v", model)
	}
	updated, _ = model.Update(deletePlansLoadedMsg{query: "pl", revision: model.searchRevision, page: 0, plans: []plan{{Content: "Plan the day"}}, total: 1})
	model = updated.(deleteModel)
	view := model.View()
	for _, expected := range []string{"Plan the day", "Page 1 of 1", "1 matching plan"} {
		if !strings.Contains(view, expected) {
			t.Errorf("picker view lacks %q: %s", expected, view)
		}
	}
}

func TestDeleteSearchIgnoresRepliesFromEarlierTyping(t *testing.T) {
	model := deleteModel{}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ab")})
	model = updated.(deleteModel)
	oldRevision := model.searchRevision
	updated, _ = model.Update(deleteSearchDueMsg{query: "ab", revision: oldRevision})
	model = updated.(deleteModel)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	model = updated.(deleteModel)
	if command == nil || model.query != "abc" || !model.searchPending || model.searchRevision == oldRevision {
		t.Fatalf("typing during a search should schedule a fresh query: %#v", model)
	}
	updated, _ = model.Update(deletePlansLoadedMsg{query: "ab", revision: oldRevision, page: 0, plans: []plan{{Content: "Stale result"}}, total: 1})
	model = updated.(deleteModel)
	if len(model.plans) != 0 || !model.searchPending {
		t.Fatalf("old results should not replace the newer search: %#v", model)
	}
	_, command = model.Update(deleteSearchDueMsg{query: "ab", revision: oldRevision})
	if command != nil {
		t.Fatal("an earlier debounce timer should not start a stale query")
	}
	updated, _ = model.Update(deleteSearchDueMsg{query: "abc", revision: model.searchRevision})
	model = updated.(deleteModel)
	updated, _ = model.Update(deletePlansLoadedMsg{query: "abc", revision: model.searchRevision, page: 0, plans: []plan{{Content: "Current result"}}, total: 1})
	model = updated.(deleteModel)
	if len(model.plans) != 1 || model.plans[0].Content != "Current result" {
		t.Fatalf("current search results should appear: %#v", model)
	}
}

func TestDeleteSearchPagesThroughMatchingDatabaseResults(t *testing.T) {
	firstPage := make([]plan, deletePageSize)
	for index := range firstPage {
		firstPage[index] = plan{Content: "Matching plan"}
	}
	model := deleteModel{query: "ma", plans: firstPage, total: 21}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(deleteModel)
	if command == nil || model.page != 1 || !model.searching || len(model.plans) != 0 {
		t.Fatalf("next page should be fetched from Supabase: %#v", model)
	}
	updated, _ = model.Update(deletePlansLoadedMsg{query: "ma", revision: model.searchRevision, page: 1, plans: []plan{{Content: "Page two result"}}, total: 21})
	model = updated.(deleteModel)
	if !strings.Contains(model.View(), "Page 2 of 3") || !strings.Contains(model.View(), "Page two result") {
		t.Fatalf("returned page should replace the earlier page: %s", model.View())
	}
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(deleteModel)
	if command == nil || model.page != 0 || !model.searching {
		t.Fatalf("previous page should also be fetched from Supabase: %#v", model)
	}
}

func TestDeleteSearchBackspaceBelowMinimumCancelsSearch(t *testing.T) {
	model := deleteModel{}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ab")})
	model = updated.(deleteModel)
	oldRevision := model.searchRevision
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = updated.(deleteModel)
	if command != nil || model.query != "a" || model.searchPending || len(model.plans) != 0 {
		t.Fatalf("less than two characters should clear results and stop searching: %#v", model)
	}
	_, command = model.Update(deleteSearchDueMsg{query: "ab", revision: oldRevision})
	if command != nil {
		t.Fatal("old timer should not search after the query becomes too short")
	}
}

func TestDeleteSearchAcceptsLettersUsedByNavigationShortcuts(t *testing.T) {
	model := deleteModel{}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model = updated.(deleteModel)
	if model.query != "j" || command != nil {
		t.Fatalf("j should be typed into the search rather than moving selection: %#v", model)
	}
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	model = updated.(deleteModel)
	if model.query != "jk" || command == nil {
		t.Fatalf("k should complete the search prefix: %#v", model)
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
	if model.done || model.loading || model.query != "" || model.selected != nil || model.deletedTasks != 0 || command != nil {
		t.Fatalf("another deletion should reopen a clean picker without loading every plan: %#v", model)
	}
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Wr")})
	model = updated.(deleteModel)
	if command == nil || !model.searchPending {
		t.Fatalf("the next search should wait for a typing pause: %#v", model)
	}
	updated, _ = model.Update(deleteSearchDueMsg{query: "Wr", revision: model.searchRevision})
	model = updated.(deleteModel)
	updated, _ = model.Update(deletePlansLoadedMsg{query: "Wr", revision: model.searchRevision, page: 0, plans: []plan{{ID: "remaining", Content: "Write report"}}, total: 1})
	model = updated.(deleteModel)
	if len(model.plans) != 1 || model.plans[0].ID != "remaining" {
		t.Fatalf("the next picker should show search results from Supabase: %#v", model)
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
