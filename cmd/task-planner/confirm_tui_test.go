package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirmModelSupportsVerticalNavigation(t *testing.T) {
	model := confirmModel{cursor: 1}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got := updated.(confirmModel).cursor; got != 0 {
		t.Fatalf("up should select the first choice, got cursor %d", got)
	}
}
