package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type oldPlansLoadedMsg struct {
	plans []plan
	err   error
}

type oldDeleteCompletedMsg struct {
	deletedCount int
	err          error
}

type oldDeleteModel struct {
	plans        []plan
	page, cursor int
	deletedCount int
	loading      bool
	confirming   bool
	done         bool
	errorMessage string
}

func (m oldDeleteModel) Init() tea.Cmd { return loadOldPlans() }

func (m oldDeleteModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case oldPlansLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.plans = msg.plans
		m.page = 0
		m.errorMessage = ""
		return m, nil
	case oldDeleteCompletedMsg:
		m.loading = false
		m.confirming = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.deletedCount = msg.deletedCount
		m.done = true
		return m, nil
	}

	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	pressed := key.String()
	if m.loading {
		if !m.confirming && (pressed == "esc" || pressed == "ctrl+c") {
			return m, tea.Quit
		}
		return m, nil
	}
	if pressed == "ctrl+c" {
		return m, tea.Quit
	}
	if m.done {
		if pressed == "enter" || pressed == "esc" {
			return m, tea.Quit
		}
		return m, nil
	}
	if m.confirming {
		switch pressed {
		case "up", "down", "left", "right", "tab":
			m.cursor = 1 - m.cursor
		case "y":
			m.cursor = 0
		case "n":
			m.cursor = 1
		case "enter":
			if m.cursor == 0 {
				m.loading = true
				return m, deleteOldPlans(m.plans)
			}
			m.confirming = false
		case "esc":
			m.confirming = false
		}
		return m, nil
	}
	if pressed == "esc" {
		return m, tea.Quit
	}
	if pressed == "r" && m.errorMessage != "" {
		m.loading = true
		return m, loadOldPlans()
	}
	if m.errorMessage != "" {
		return m, nil
	}
	pageCount := (len(m.plans) + deletePageSize - 1) / deletePageSize
	switch pressed {
	case "up", "left":
		if m.page > 0 {
			m.page--
		}
	case "down", "right":
		if m.page+1 < pageCount {
			m.page++
		}
	case "enter":
		if len(m.plans) > 0 {
			m.confirming = true
			m.cursor = 1
		}
	}
	return m, nil
}

func (m oldDeleteModel) View() string {
	var builder strings.Builder
	if m.done {
		builder.WriteString(successStyle.Render(fmt.Sprintf("✓ Deleted %d past schedules from Supabase only. Todoist tasks were left unchanged.", m.deletedCount)))
		builder.WriteString("\n\n" + mutedStyle.Render("Enter or Esc close") + "\n")
		return builder.String()
	}
	if m.loading {
		if m.confirming {
			builder.WriteString(titleStyle.Render("Deleting previewed past schedules from Supabase…") + "\n")
		} else {
			builder.WriteString(titleStyle.Render("Loading past schedules…") + "\n")
		}
		return builder.String()
	}
	if m.confirming {
		builder.WriteString(titleStyle.Render(fmt.Sprintf("Delete all %d past schedules from Supabase?", len(m.plans))) + "\n\n")
		builder.WriteString(warningStyle.Render("All schedules shown in the preview and their stored task IDs will be removed from Supabase. Todoist tasks will stay unchanged; task-planner can no longer delete them."))
		builder.WriteString("\n\n")
		for index, choice := range []string{"Yes, delete all from Supabase only", "No, keep them"} {
			if index == m.cursor {
				builder.WriteString(selectedStyle.Render("› " + choice))
			} else {
				builder.WriteString("  " + choice)
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n" + mutedStyle.Render("↑/↓ select · Enter confirm · Esc return to preview") + "\n")
		return builder.String()
	}
	builder.WriteString(titleStyle.Render("Past schedules to delete from Supabase") + "\n")
	if m.errorMessage != "" {
		builder.WriteString(warningStyle.Render("! "+m.errorMessage) + "\n")
		builder.WriteString(mutedStyle.Render("R reload preview · Esc close") + "\n")
		return builder.String()
	}
	if len(m.plans) == 0 {
		builder.WriteString(mutedStyle.Render("No past schedules remain in Supabase. · Esc close") + "\n")
		return builder.String()
	}
	start := m.page * deletePageSize
	end := min(start+deletePageSize, len(m.plans))
	for _, p := range m.plans[start:end] {
		fmt.Fprintf(&builder, "  %q  ·  %s to %s\n", p.Content, p.StartDate.Format("02 Jan 2006"), p.EndDate.Format("02 Jan 2006"))
	}
	pageCount := (len(m.plans) + deletePageSize - 1) / deletePageSize
	builder.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Page %d of %d · %d past schedules · ↑/↓ or ←/→ page · Enter review deleting all · Esc close", m.page+1, pageCount, len(m.plans))) + "\n")
	return builder.String()
}

func loadOldPlans() tea.Cmd {
	return func() tea.Msg {
		plans, err := pastPlans(time.Now().Format(time.DateOnly))
		return oldPlansLoadedMsg{plans: plans, err: err}
	}
}

func deleteOldPlans(plans []plan) tea.Cmd {
	ids := make([]string, len(plans))
	for index, p := range plans {
		ids[index] = p.ID
	}
	return func() tea.Msg {
		err := removePastPlans(ids, time.Now().Format(time.DateOnly))
		return oldDeleteCompletedMsg{deletedCount: len(ids), err: err}
	}
}

func guidedDeleteOld() error {
	_, err := tea.NewProgram(oldDeleteModel{loading: true}, tea.WithAltScreen()).Run()
	return err
}
