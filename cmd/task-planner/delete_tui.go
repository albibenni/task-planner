package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	deletePageSize       = 10
	deleteSearchDebounce = 300 * time.Millisecond
)

type deleteSearchDueMsg struct {
	query    string
	revision int
}

type deletePlansLoadedMsg struct {
	query    string
	revision int
	page     int
	total    int
	plans    []plan
	err      error
}

type deleteCompletedMsg struct {
	deletedTasks int
	databaseOnly bool
	err          error
}

type deleteModel struct {
	query, errorMessage string
	plans               []plan
	page, cursor, total int
	searchRevision      int
	loading, confirming bool
	searchPending       bool
	searching           bool
	selected            *plan
	done, cancelled     bool
	pastSchedule        bool
	databaseOnly        bool
	deletedTasks        int
}

func (m deleteModel) Init() tea.Cmd { return nil }

func (m deleteModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case deleteSearchDueMsg:
		if msg.query != m.query || msg.revision != m.searchRevision || !m.searchPending {
			return m, nil
		}
		m.searchPending = false
		m.searching = true
		return m, searchDeletePlans(m.query, m.page, m.searchRevision)
	case deletePlansLoadedMsg:
		if !m.searching || msg.query != m.query || msg.revision != m.searchRevision || msg.page != m.page {
			return m, nil
		}
		m.searching = false
		m.errorMessage = ""
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.plans = msg.plans
		m.total = msg.total
		if m.cursor >= len(m.plans) {
			m.cursor = max(0, len(m.plans)-1)
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
		m.databaseOnly = msg.databaseOnly
		m.done = true
		m.confirming = false
		m.cursor = 0
		return m, nil
	}

	key, ok := message.(tea.KeyMsg)
	if !ok || m.loading {
		return m, nil
	}
	pressed := key.String()
	if m.done {
		return m.updateCompletedDelete(key)
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
		choiceCount := len(m.confirmChoices())
		switch pressed {
		case "left", "up":
			m.cursor = (m.cursor + choiceCount - 1) % choiceCount
		case "right", "down", "tab":
			m.cursor = (m.cursor + 1) % choiceCount
		case "y":
			m.cursor = 0
		case "n":
			m.cursor = choiceCount - 1
		case "enter":
			if m.cursor == 0 {
				m.loading = true
				return m, deleteSelectedPlan(*m.selected)
			}
			if m.pastSchedule && m.cursor == 1 {
				m.loading = true
				return m, deleteDatabaseOnlyPlan(*m.selected)
			}
			m.confirming = false
			m.cursor = 0
		}
		return m, nil
	}

	if pressed == "backspace" {
		query := []rune(m.query)
		if len(query) > 0 {
			return m.changeQuery(string(query[:len(query)-1]))
		}
		return m, nil
	}
	if len(key.Runes) > 0 {
		return m.changeQuery(m.query + string(key.Runes))
	}
	if m.searchPending || m.searching {
		return m, nil
	}
	switch pressed {
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down":
		if m.cursor < len(m.plans)-1 {
			m.cursor++
		}
	case "left":
		if m.page > 0 {
			m.page--
			m.cursor = 0
			m.plans = nil
			m.searching = true
			return m, searchDeletePlans(m.query, m.page, m.searchRevision)
		}
	case "right":
		if (m.page+1)*deletePageSize < m.total {
			m.page++
			m.cursor = 0
			m.plans = nil
			m.searching = true
			return m, searchDeletePlans(m.query, m.page, m.searchRevision)
		}
	case "enter":
		if len(m.plans) > 0 {
			selected := m.plans[m.cursor]
			m.selected = &selected
			m.confirming = true
			m.pastSchedule = scheduleIsPast(selected.EndDate, time.Now())
			m.cursor = len(m.confirmChoices()) - 1
		}
	}
	return m, nil
}

func (m deleteModel) changeQuery(query string) (tea.Model, tea.Cmd) {
	m.query = query
	m.page, m.cursor, m.total = 0, 0, 0
	m.plans = nil
	m.errorMessage = ""
	m.searchRevision++
	m.searching = false
	m.searchPending = len([]rune(query)) >= 2
	if !m.searchPending {
		return m, nil
	}
	revision := m.searchRevision
	return m, tea.Tick(deleteSearchDebounce, func(time.Time) tea.Msg {
		return deleteSearchDueMsg{query: query, revision: revision}
	})
}

func (m deleteModel) updateCompletedDelete(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "left", "right", "up", "down", "tab":
		m.cursor = 1 - m.cursor
	case "y":
		m.cursor = 0
	case "n":
		m.cursor = 1
	case "ctrl+c", "esc":
		return m, tea.Quit
	case "enter":
		if m.cursor == 0 {
			return deleteModel{}, nil
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m deleteModel) View() string {
	if m.cancelled {
		return mutedStyle.Render("Cancelled.") + "\n"
	}
	if m.done {
		var builder strings.Builder
		if m.databaseOnly {
			builder.WriteString(successStyle.Render(fmt.Sprintf("✓ Deleted %q from Supabase only. Todoist tasks were left unchanged.", m.selected.Content)) + "\n\n")
		} else {
			builder.WriteString(successStyle.Render(fmt.Sprintf("✓ Deleted %q and %d matching Todoist task(s).", m.selected.Content, m.deletedTasks)) + "\n\n")
		}
		builder.WriteString(promptStyle.Render("Do you want to delete another plan?") + "\n\n")
		for index, choice := range []string{"Yes, delete another plan", "No, close"} {
			if index == m.cursor {
				builder.WriteString(selectedStyle.Render("› " + choice))
			} else {
				builder.WriteString("  " + choice)
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n" + mutedStyle.Render("↑/↓ or ←/→ select · Enter confirm · Esc close") + "\n")
		return builder.String()
	}
	if m.confirming {
		return m.confirmView()
	}
	var builder strings.Builder
	builder.WriteString(titleStyle.Render("Delete a shared Todoist plan") + "\n")
	builder.WriteString(promptStyle.Render("Search task text") + "\n\n")
	builder.WriteString(inputStyle.Render(m.query+"█") + "\n\n")
	if len([]rune(m.query)) < 2 {
		builder.WriteString(mutedStyle.Render("Type at least 2 characters to search Supabase · Esc close") + "\n")
		return builder.String()
	}
	if m.searchPending {
		builder.WriteString(mutedStyle.Render("Waiting for typing to pause…") + "\n")
		return builder.String()
	}
	if m.searching {
		builder.WriteString(mutedStyle.Render("Searching Supabase…") + "\n")
		return builder.String()
	}
	if m.errorMessage != "" {
		builder.WriteString(warningStyle.Render("! "+m.errorMessage) + "\n")
		return builder.String()
	}
	if len(m.plans) == 0 {
		builder.WriteString(mutedStyle.Render("No active plans match this search in Supabase.") + "\n\n")
		builder.WriteString(mutedStyle.Render("Type to search · Esc to close") + "\n")
		return builder.String()
	}
	for index, p := range m.plans {
		entry := fmt.Sprintf("%s  ·  %s to %s", p.Content, p.StartDate.Format("02 Jan 2006"), p.EndDate.Format("02 Jan 2006"))
		if index == m.cursor {
			builder.WriteString(selectedStyle.Render("› " + entry))
		} else {
			builder.WriteString("  " + entry)
		}
		builder.WriteString("\n")
	}
	pages := (m.total + deletePageSize - 1) / deletePageSize
	noun := "plans"
	if m.total == 1 {
		noun = "plan"
	}
	builder.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Page %d of %d · %d matching %s · ↑/↓ select · ←/→ page · Enter delete · Esc close", m.page+1, pages, m.total, noun)) + "\n")
	return builder.String()
}

func (m deleteModel) confirmView() string {
	choices := m.confirmChoices()
	var builder strings.Builder
	builder.WriteString(titleStyle.Render("Delete shared plan?") + "\n\n")
	if m.pastSchedule {
		fmt.Fprintf(&builder, "%q ended on %s. Choose what to delete.\nDatabase only removes the schedule and stored task IDs; Todoist tasks stay as they are and task-planner can no longer delete them.\n\n", m.selected.Content, m.selected.EndDate.Format("02 Jan 2006"))
	} else {
		fmt.Fprintf(&builder, "This will delete %q from Supabase and its matching active Todoist task(s).\n\n", m.selected.Content)
	}
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

func (m deleteModel) confirmChoices() []string {
	if m.pastSchedule {
		return []string{"Delete schedule and Todoist tasks", "Delete schedule from database only", "No, keep it"}
	}
	return []string{"Yes, delete", "No, keep it"}
}

func scheduleIsPast(endDate, today time.Time) bool {
	return !endDate.IsZero() && endDate.Format(time.DateOnly) < today.Format(time.DateOnly)
}

func searchDeletePlans(query string, page, revision int) tea.Cmd {
	return func() tea.Msg {
		total, err := plansCount(query)
		if err != nil {
			return deletePlansLoadedMsg{query: query, revision: revision, page: page, err: err}
		}
		if total == 0 {
			return deletePlansLoadedMsg{query: query, revision: revision, page: page}
		}
		plans, err := plansPage(query, deletePageSize, page*deletePageSize)
		return deletePlansLoadedMsg{query: query, revision: revision, page: page, total: total, plans: plans, err: err}
	}
}

func deleteSelectedPlan(p plan) tea.Cmd {
	return func() tea.Msg {
		deletedTasks, err := deletePlanAndTodoistTasks(p)
		return deleteCompletedMsg{deletedTasks: deletedTasks, err: err}
	}
}

func deleteDatabaseOnlyPlan(p plan) tea.Cmd {
	return func() tea.Msg {
		return deleteCompletedMsg{databaseOnly: true, err: removePlan(p.ID)}
	}
}

func guidedDelete() error {
	_, err := tea.NewProgram(deleteModel{}, tea.WithAltScreen()).Run()
	return err
}
