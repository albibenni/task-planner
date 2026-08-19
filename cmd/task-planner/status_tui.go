package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type setupStatus struct {
	supabaseConfigured bool
	supabaseReachable  bool
	supabaseError      error
	todoistConfigured  bool
}

func localSetupStatus() setupStatus {
	connectionURL, err := dbURL()
	status := setupStatus{todoistConfigured: todoistConfigured()}
	if err != nil || validateDatabaseURL(connectionURL) != nil {
		return status
	}
	status.supabaseConfigured = true
	status.supabaseError = databaseReachable(connectionURL)
	status.supabaseReachable = status.supabaseError == nil
	return status
}

func (status setupStatus) ready() bool {
	return status.supabaseReachable && status.todoistConfigured
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
	builder.WriteString(titleStyle.Render("task-planner setup") + "\n")
	builder.WriteString(databaseStatusLine(m.status))
	builder.WriteString("\n")
	builder.WriteString(statusLine("Todoist login", m.status.todoistConfigured))
	builder.WriteString("\n\n")
	if m.status.ready() {
		builder.WriteString(successStyle.Render("Ready to add and schedule shared plans.") + "\n")
	} else {
		builder.WriteString(promptStyle.Render("Complete the missing setup steps:") + "\n")
		if !m.status.supabaseConfigured {
			builder.WriteString("  task-planner config\n")
		} else if !m.status.supabaseReachable {
			builder.WriteString("  Supabase Dashboard → Connect → Session Pooler, then task-planner config\n")
		}
		if !m.status.todoistConfigured {
			builder.WriteString("  task-planner auth login\n")
		}
	}
	builder.WriteString("\n" + mutedStyle.Render("Press any key to close.") + "\n")
	return builder.String()
}

func databaseStatusLine(status setupStatus) string {
	if !status.supabaseConfigured {
		return warningStyle.Render("! Supabase database URL: missing")
	}
	if !status.supabaseReachable {
		return warningStyle.Render("! Supabase database: unreachable")
	}
	return successStyle.Render("✓ Supabase database: reachable")
}

func statusLine(label string, configured bool) string {
	if configured {
		return successStyle.Render(fmt.Sprintf("✓ %s: configured", label))
	}
	return warningStyle.Render(fmt.Sprintf("! %s: missing", label))
}

func showStatus() error {
	_, err := tea.NewProgram(statusModel{status: localSetupStatus()}, tea.WithAltScreen()).Run()
	return err
}
