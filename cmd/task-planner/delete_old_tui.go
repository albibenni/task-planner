package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type oldPlansLoadedMsg struct {
	page, total int
	plans       []plan
	err         error
}

type oldDeleteCompletedMsg struct {
	err error
}

type oldDeleteModel struct {
	plans        []plan
	page, total  int
	cursor       int
	selected     *plan
	loading      bool
	confirming   bool
	done         bool
	errorMessage string
}

func (m oldDeleteModel) Init() tea.Cmd { return loadOldPlans(m.page) }

func (m oldDeleteModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case oldPlansLoadedMsg:
		if msg.page != m.page {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.errorMessage = ""
		m.total = msg.total
		if m.total == 0 {
			m.page = 0
		}
		if m.total > 0 && m.page*deletePageSize >= m.total {
			m.page = (m.total - 1) / deletePageSize
			m.loading = true
			return m, loadOldPlans(m.page)
		}
		m.plans = msg.plans
		m.cursor = min(m.cursor, max(0, len(m.plans)-1))
		return m, nil
	case oldDeleteCompletedMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			m.confirming = false
			m.cursor = 0
			return m, nil
		}
		m.confirming = false
		m.done = true
		m.cursor = 0
		return m, nil
	}

	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	pressed := key.String()
	if pressed == "ctrl+c" {
		return m, tea.Quit
	}
	if m.loading {
		if pressed == "esc" {
			return m, tea.Quit
		}
		return m, nil
	}
	if m.done {
		switch pressed {
		case "up", "down", "left", "right", "tab":
			m.cursor = 1 - m.cursor
		case "y":
			m.cursor = 0
		case "n":
			m.cursor = 1
		case "enter":
			if m.cursor == 1 {
				return m, tea.Quit
			}
			m.done = false
			m.selected = nil
			m.plans = nil
			m.loading = true
			return m, loadOldPlans(m.page)
		case "esc":
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
				return m, deleteOldSelectedPlan(*m.selected)
			}
			m.confirming = false
			m.cursor = 0
		case "esc":
			m.confirming = false
			m.cursor = 0
		}
		return m, nil
	}
	if pressed == "esc" {
		return m, tea.Quit
	}
	if pressed == "r" && m.errorMessage != "" {
		m.loading = true
		m.plans = nil
		return m, loadOldPlans(m.page)
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
			m.plans = nil
			m.cursor = 0
			m.loading = true
			return m, loadOldPlans(m.page)
		}
	case "right":
		if (m.page+1)*deletePageSize < m.total {
			m.page++
			m.plans = nil
			m.cursor = 0
			m.loading = true
			return m, loadOldPlans(m.page)
		}
	case "enter":
		if len(m.plans) > 0 {
			selected := m.plans[m.cursor]
			m.selected = &selected
			m.confirming = true
			m.cursor = 1
		}
	}
	return m, nil
}

func (m oldDeleteModel) View() string {
	var builder strings.Builder
	if m.done {
		builder.WriteString(successStyle.Render(fmt.Sprintf("✓ Deleted %q from Supabase only. Todoist tasks were left unchanged.", m.selected.Content)))
		builder.WriteString("\n\n")
		builder.WriteString(promptStyle.Render("Do you want to delete another past schedule?"))
		builder.WriteString("\n\n")
		for index, choice := range []string{"Yes, show past schedules", "No, close"} {
			if index == m.cursor {
				builder.WriteString(selectedStyle.Render("› " + choice))
			} else {
				builder.WriteString("  " + choice)
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n" + mutedStyle.Render("↑/↓ select · Enter confirm · Esc close") + "\n")
		return builder.String()
	}
	if m.confirming {
		builder.WriteString(titleStyle.Render("Delete past schedule from Supabase?") + "\n\n")
		fmt.Fprintf(&builder, "%q — %s to %s\n\n", m.selected.Content, m.selected.StartDate.Format("02 Jan 2006"), m.selected.EndDate.Format("02 Jan 2006"))
		builder.WriteString(warningStyle.Render("The schedule and stored task IDs will be removed from Supabase. Todoist tasks will stay unchanged; task-planner can no longer delete them."))
		builder.WriteString("\n\n")
		for index, choice := range []string{"Yes, delete from Supabase only", "No, keep it"} {
			if index == m.cursor {
				builder.WriteString(selectedStyle.Render("› " + choice))
			} else {
				builder.WriteString("  " + choice)
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n" + mutedStyle.Render("↑/↓ select · Enter confirm · Esc cancel") + "\n")
		return builder.String()
	}
	builder.WriteString(titleStyle.Render("Delete a past schedule from Supabase") + "\n")
	if m.loading {
		builder.WriteString(mutedStyle.Render("Loading past schedules…") + "\n")
		return builder.String()
	}
	if m.errorMessage != "" {
		builder.WriteString(warningStyle.Render("! "+m.errorMessage) + "\n")
		builder.WriteString(mutedStyle.Render("R retry · Esc close") + "\n")
		if len(m.plans) == 0 {
			return builder.String()
		}
	}
	if len(m.plans) == 0 {
		builder.WriteString(mutedStyle.Render("No past schedules remain in Supabase. · Esc close") + "\n")
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
	builder.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Page %d of %d · %d past schedules · ↑/↓ select · ←/→ page · Enter delete · Esc close", m.page+1, pages, m.total)) + "\n")
	return builder.String()
}

func loadOldPlans(page int) tea.Cmd {
	return func() tea.Msg {
		today := time.Now().Format(time.DateOnly)
		total, err := pastPlansCount(today)
		if err != nil {
			return oldPlansLoadedMsg{page: page, err: err}
		}
		if total == 0 {
			return oldPlansLoadedMsg{page: page}
		}
		plans, err := pastPlansPage(today, deletePageSize, page*deletePageSize)
		return oldPlansLoadedMsg{page: page, total: total, plans: plans, err: err}
	}
}

func deleteOldSelectedPlan(p plan) tea.Cmd {
	return func() tea.Msg {
		return oldDeleteCompletedMsg{err: removePastPlan(p.ID, time.Now().Format(time.DateOnly))}
	}
}

func guidedDeleteOld() error {
	_, err := tea.NewProgram(oldDeleteModel{loading: true}, tea.WithAltScreen()).Run()
	return err
}
