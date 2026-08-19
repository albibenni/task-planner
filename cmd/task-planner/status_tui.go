package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type setupStatus struct {
	supabaseConfigured bool
	todoistConfigured  bool
}

func localSetupStatus() setupStatus {
	connectionURL, err := dbURL()
	return setupStatus{
		supabaseConfigured: err == nil && validateDatabaseURL(connectionURL) == nil,
		todoistConfigured:  todoistConfigured(),
	}
}

func (status setupStatus) ready() bool {
	return status.supabaseConfigured && status.todoistConfigured
}

type statusModel struct {
	status setupStatus
}

func (m statusModel) Init() tea.Cmd { return nil }

func (m statusModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := message.(tea.KeyMsg); ok {
		return m, tea.Quit
	}
	return m, nil
}

func (m statusModel) View() string {
	var builder strings.Builder
	builder.WriteString("task-planner setup\n\n")
	builder.WriteString(statusLine("Supabase database URL", m.status.supabaseConfigured))
	builder.WriteString("\n")
	builder.WriteString(statusLine("Todoist login", m.status.todoistConfigured))
	builder.WriteString("\n\n")
	if m.status.ready() {
		builder.WriteString("Ready to add and schedule shared plans.\n")
	} else {
		builder.WriteString("Complete the missing setup steps:\n")
		if !m.status.supabaseConfigured {
			builder.WriteString("  task-planner config\n")
		}
		if !m.status.todoistConfigured {
			builder.WriteString("  task-planner auth login\n")
		}
	}
	builder.WriteString("\nPress any key to close.\n")
	return builder.String()
}

func statusLine(label string, configured bool) string {
	if configured {
		return fmt.Sprintf("✓ %s: configured", label)
	}
	return fmt.Sprintf("! %s: missing", label)
}

func showStatus() error {
	_, err := tea.NewProgram(statusModel{status: localSetupStatus()}, tea.WithAltScreen()).Run()
	return err
}
