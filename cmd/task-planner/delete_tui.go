package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const deletePageSize = 10

type deletePlansLoadedMsg struct {
	plans []plan
	err   error
}

type deleteCompletedMsg struct {
	deletedTasks int
	err          error
}

type deleteModel struct {
	query, errorMessage string
	plans               []plan
	page, cursor        int
	loading, confirming bool
	selected            *plan
	done, cancelled     bool
	deletedTasks        int
}

func (m deleteModel) Init() tea.Cmd {
	return loadAllDeletePlans()
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
		m.plans = msg.plans
		if m.cursor >= len(m.pagePlans()) {
			m.cursor = max(0, len(m.pagePlans())-1)
		}
		return m, nil
	case deleteCompletedMsg:
		m.loading = false
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
		case "left", "right", "up", "down", "tab":
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
		if m.cursor < len(m.pagePlans())-1 {
			m.cursor++
		}
	case "left", "h":
		if m.page > 0 {
			m.page--
			m.cursor = 0
		}
	case "right", "l":
		if (m.page+1)*deletePageSize < len(m.filteredPlans()) {
			m.page++
			m.cursor = 0
		}
	case "backspace":
		query := []rune(m.query)
		if len(query) > 0 {
			m.query = string(query[:len(query)-1])
			m.page, m.cursor = 0, 0
		}
	case "enter":
		pagePlans := m.pagePlans()
		if len(pagePlans) > 0 {
			selected := pagePlans[m.cursor]
			m.selected = &selected
			m.confirming = true
			m.cursor = 1
		}
	default:
		if len(key.Runes) > 0 {
			m.query += string(key.Runes)
			m.page, m.cursor = 0, 0
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
	pagePlans := m.pagePlans()
	if len(pagePlans) == 0 {
		builder.WriteString(mutedStyle.Render("No active plans match this search.") + "\n\n")
		builder.WriteString(mutedStyle.Render("Type to search · Esc to close") + "\n")
		return builder.String()
	}
	for index, p := range pagePlans {
		entry := fmt.Sprintf("%s  ·  %s to %s", p.Content, p.StartDate.Format("02 Jan 2006"), p.EndDate.Format("02 Jan 2006"))
		if index == m.cursor {
			builder.WriteString(selectedStyle.Render("› " + entry))
		} else {
			builder.WriteString("  " + entry)
		}
		builder.WriteString("\n")
	}
	total := len(m.filteredPlans())
	pages := (total + deletePageSize - 1) / deletePageSize
	builder.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Page %d of %d · %d active plan(s) · ↑/↓ select · ←/→ page · Enter delete · Esc close", m.page+1, pages, total)) + "\n")
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
	builder.WriteString("\n" + mutedStyle.Render("↑/↓ or ←/→ select · Enter confirm · Esc cancel") + "\n")
	return builder.String()
}

func (m deleteModel) filteredPlans() []plan {
	if m.query == "" {
		return m.plans
	}
	query := strings.ToLower(m.query)
	filtered := make([]plan, 0)
	for _, p := range m.plans {
		if strings.Contains(strings.ToLower(p.Content), query) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (m deleteModel) pagePlans() []plan {
	filtered := m.filteredPlans()
	start := m.page * deletePageSize
	if start >= len(filtered) {
		return nil
	}
	end := min(start+deletePageSize, len(filtered))
	return filtered[start:end]
}

func loadAllDeletePlans() tea.Cmd {
	return func() tea.Msg {
		allPlans := make([]plan, 0)
		for offset := 0; ; offset += deletePageSize {
			page, err := plansPage("", deletePageSize, offset)
			if err != nil {
				return deletePlansLoadedMsg{err: err}
			}
			allPlans = append(allPlans, page...)
			if len(page) < deletePageSize {
				return deletePlansLoadedMsg{plans: allPlans}
			}
		}
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
