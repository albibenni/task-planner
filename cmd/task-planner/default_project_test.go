package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefaultProjectPickerSelectsAndCancels(t *testing.T) {
	projects := []project{{ID: "inbox", Name: "Inbox"}, {ID: "coding", Name: "Coding"}, {ID: "work", Name: "Work"}}
	model := newDefaultProjectPickerModel(projects, "coding")
	if model.cursor != 1 {
		t.Fatalf("saved default should be selected initially: %d", model.cursor)
	}
	if view := model.View(); !strings.Contains(view, "Coding") || !strings.Contains(view, "Enter") {
		t.Fatalf("picker should show projects and controls: %s", view)
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(defaultProjectPickerModel)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(defaultProjectPickerModel)
	if !model.confirmed || model.projects[model.cursor].ID != "work" || cmd == nil {
		t.Fatalf("enter should confirm the highlighted project: %#v", model)
	}

	model = newDefaultProjectPickerModel(projects, "coding")
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(defaultProjectPickerModel)
	if model.confirmed || cmd == nil {
		t.Fatalf("escape should cancel the picker: %#v", model)
	}
}

func TestDefaultProjectPickerKeepsLongListsNavigable(t *testing.T) {
	projects := make([]project, 20)
	for index := range projects {
		projects[index] = project{ID: string(rune('a' + index)), Name: "Project " + string(rune('a'+index))}
	}
	model := newDefaultProjectPickerModel(projects, projects[18].ID)
	view := model.View()
	if !strings.Contains(view, "Project s") || !strings.Contains(view, "19 of 20") {
		t.Fatalf("saved project should be visible in a long list: %s", view)
	}
}

type projectResponseTransport func(*http.Request) (*http.Response, error)

func (transport projectResponseTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestDefaultProjectSetterAndManualConfigUseSameFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TODOIST_API_TOKEN", "test-token")
	previousClient := todoistHTTPClient
	todoistHTTPClient = &http.Client{Transport: projectResponseTransport(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/v1/projects" {
			t.Fatalf("unexpected Todoist request: %s %s", request.Method, request.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"results":[{"id":"inbox","name":"Inbox"},{"id":"coding","name":"Coding"}]}`)), Header: make(http.Header)}, nil
	})}
	t.Cleanup(func() { todoistHTTPClient = previousClient })

	if err := setDefaultProject("Coding"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(os.Getenv("HOME"), ".config", "task-planner", "default-project-id")
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "coding\n" {
		t.Fatalf("setter should store the stable Todoist ID: %q, %v", raw, err)
	}
	if err := os.WriteFile(path, []byte("inbox\n"), 0600); err != nil {
		t.Fatal(err)
	}
	loaded := loadAddProjects()().(projectsLoadedMsg)
	if loaded.err != nil || loaded.defaultProjectID != "inbox" {
		t.Fatalf("manual edits should set the same picker default: %#v", loaded)
	}
	if err := clearDefaultProject(); err != nil {
		t.Fatal(err)
	}
	if id, err := readDefaultProjectID(); err != nil || id != "" {
		t.Fatalf("clearing should restore the normal project picker: %q, %v", id, err)
	}
}

func TestDefaultProjectSetterRejectsUnknownOrAmbiguousNames(t *testing.T) {
	projects := []project{{ID: "a", Name: "Work"}, {ID: "b", Name: "Work"}, {ID: "c", Name: "Personal"}}
	if _, err := resolveDefaultProject(projects, "Work"); err == nil || !strings.Contains(err.Error(), "ID") {
		t.Fatalf("duplicate names should require an ID: %v", err)
	}
	if _, err := resolveDefaultProject(projects, "Missing"); err == nil {
		t.Fatal("unknown project should be rejected")
	}
	if got, err := resolveDefaultProject(projects, "b"); err != nil || got.ID != "b" {
		t.Fatalf("ID should identify one project even when names repeat: %#v, %v", got, err)
	}
}
