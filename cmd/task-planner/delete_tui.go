package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const deletePageSize = 10

type deletePlansLoadedMsg struct {
	plans []plan
	total int
	err   error
}

type deleteCompletedMsg struct {
	deletedTasks int
	err          error
}

type deleteModel struct {
	query, errorMessage string
	plans               []plan
	total, page, cursor int
	loading, confirming bool
	selected            *plan
	done, cancelled     bool
	deletedTasks        int
}

func (m deleteModel) Init() tea.Cmd {
	return loadDeletePlans("", 0)
}

func (m deleteModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case deletePlansLoadedMsg:
		m.loading = false
		m.errorMessage = ""
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.plans, m.total = msg.plans, msg.total
		if m.cursor >= len(m.plans) {
			m.cursor = max(0, len(m.plans)-1)
		}
		return m, nil
	case deleteCompletedMsg:
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			m.confirming = false
			return m, nil
		}
		m.deletedTasks = msg.deletedTasks
		m.done = true
		return m, nil
	}

	key, ok := message.(tea.KeyMsg)
	if !ok || m.loading {
		return m, nil
	}
	pressed := key.String()
	if m.done {
		return m, tea.Quit
	}
	if pressed == "ctrl+c" || pressed == "esc" {
		if m.confirming {
			m.confirming = false
			return m, nil
		}
		m.cancelled = true
		return m, tea.Quit
	}
	if m.confirming {
		switch pressed {
		case "left", "right", "tab":
			m.cursor = 1 - m.cursor
		case "y":
			m.cursor = 0
		case "n":
			m.cursor = 1
		case "enter":
			if m.cursor == 0 {
				m.loading = true
				return m, deleteSelectedPlan(*m.selected)
			}
			m.confirming = false
			m.cursor = 0
		}
		return m, nil
	}

	switch pressed {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.plans)-1 {
			m.cursor++
		}
	case "left", "h":
		if m.page > 0 {
			m.page--
			m.cursor = 0
			m.loading = true
			return m, loadDeletePlans(m.query, m.page)
		}
	case "right", "l":
		if (m.page+1)*deletePageSize < m.total {
			m.page++
			m.cursor = 0
			m.loading = true
			return m, loadDeletePlans(m.query, m.page)
		}
	case "backspace":
		query := []rune(m.query)
		if len(query) > 0 {
			m.query = string(query[:len(query)-1])
			m.page, m.cursor, m.loading = 0, 0, true
			return m, loadDeletePlans(m.query, m.page)
		}
	case "enter":
		if len(m.plans) > 0 {
			selected := m.plans[m.cursor]
			m.selected = &selected
			m.confirming = true
			m.cursor = 1
		}
	default:
		if len(key.Runes) > 0 {
			m.query += string(key.Runes)
			m.page, m.cursor, m.loading = 0, 0, true
			return m, loadDeletePlans(m.query, m.page)
		}
	}
	return m, nil
}

func (m deleteModel) View() string {
	if m.cancelled {
		return mutedStyle.Render("Cancelled.") + "\n"
	}
	if m.done {
		return successStyle.Render(fmt.Sprintf("✓ Deleted %q and %d matching Todoist task(s).", m.selected.Content, m.deletedTasks)) + "\n\n" + mutedStyle.Render("Press any key to close.") + "\n"
	}
	if m.confirming {
		return m.confirmView()
	}
	var builder strings.Builder
	builder.WriteString(titleStyle.Render("Delete a shared Todoist plan") + "\n")
	builder.WriteString(promptStyle.Render("Search task text") + "\n\n")
	builder.WriteString(inputStyle.Render(m.query+"█") + "\n\n")
	if m.loading {
		builder.WriteString(mutedStyle.Render("Loading plans…") + "\n")
		return builder.String()
	}
	if m.errorMessage != "" {
		builder.WriteString(warningStyle.Render("! "+m.errorMessage) + "\n")
		return builder.String()
	}
	if len(m.plans) == 0 {
		builder.WriteString(mutedStyle.Render("No active plans match this search.") + "\n\n")
		builder.WriteString(mutedStyle.Render("Type to search · Esc to close") + "\n")
		return builder.String()
	}
	for index, p := range m.plans {
		entry := fmt.Sprintf("%s  ·  %s", p.Content, p.Period)
		if index == m.cursor {
			builder.WriteString(selectedStyle.Render("› " + entry))
		} else {
			builder.WriteString("  " + entry)
		}
		builder.WriteString("\n")
	}
	pages := (m.total + deletePageSize - 1) / deletePageSize
	builder.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Page %d of %d · %d active plan(s) · ↑/↓ select · ←/→ page · Enter delete · Esc close", m.page+1, pages, m.total)) + "\n")
	return builder.String()
}

func (m deleteModel) confirmView() string {
	choices := []string{"Yes, delete", "No, keep it"}
	var builder strings.Builder
	builder.WriteString(titleStyle.Render("Delete shared plan?") + "\n\n")
	fmt.Fprintf(&builder, "This will delete %q from Supabase and its matching active Todoist task(s).\n\n", m.selected.Content)
	for index, choice := range choices {
		if index == m.cursor {
			builder.WriteString(selectedStyle.Render("› " + choice))
		} else {
			builder.WriteString("  " + choice)
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\n" + mutedStyle.Render("←/→ select · Enter confirm · Esc cancel") + "\n")
	return builder.String()
}

func loadDeletePlans(query string, page int) tea.Cmd {
	return func() tea.Msg {
		plans, total, err := plansPage(query, deletePageSize, page*deletePageSize)
		return deletePlansLoadedMsg{plans: plans, total: total, err: err}
	}
}

func deleteSelectedPlan(p plan) tea.Cmd {
	return func() tea.Msg {
		deletedTasks, err := deletePlanAndTodoistTasks(p)
		return deleteCompletedMsg{deletedTasks: deletedTasks, err: err}
	}
}

func guidedDelete() error {
	_, err := tea.NewProgram(deleteModel{}, tea.WithAltScreen()).Run()
	return err
}
