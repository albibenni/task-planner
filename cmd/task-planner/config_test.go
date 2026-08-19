package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestValidateDatabaseURL(t *testing.T) {
	valid := []string{
		"postgres://user:password@db.example.com:5432/postgres",
		"postgresql://user:password@db.example.com/database",
	}
	for _, value := range valid {
		if err := validateDatabaseURL(value); err != nil {
			t.Errorf("expected %q to be valid: %v", value, err)
		}
	}

	for _, value := range []string{"", "https://example.com", "postgres://"} {
		if err := validateDatabaseURL(value); err == nil {
			t.Errorf("expected %q to be invalid", value)
		}
	}
}

func TestConfigModelEditsAtCursor(t *testing.T) {
	model := updateConfigModel(t, configModel{}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ab")})
	model = updateConfigModel(t, model, tea.KeyMsg{Type: tea.KeyLeft})
	model = updateConfigModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	if model.value != "aXb" || model.cursor != 2 {
		t.Fatalf("inserting in the middle corrupted input: %#v", model)
	}
	model = updateConfigModel(t, model, tea.KeyMsg{Type: tea.KeyDelete})

	if model.value != "aX" || model.cursor != 2 {
		t.Fatalf("unexpected editor state: %#v", model)
	}
}

func TestConfigModelKeepsInvalidURLEditable(t *testing.T) {
	model := updateConfigModel(t, configModel{value: "not-a-url", cursor: len("not-a-url")}, tea.KeyMsg{Type: tea.KeyEnter})
	if model.done || model.validationError == "" {
		t.Fatalf("invalid URL should remain editable: %#v", model)
	}
}

func updateConfigModel(t *testing.T, model configModel, message tea.Msg) configModel {
	t.Helper()
	updated, _ := model.Update(message)
	return updated.(configModel)
}
