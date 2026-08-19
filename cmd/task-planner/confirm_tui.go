package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type confirmModel struct {
	title, message string
	yesLabel       string
	noLabel        string
	cursor         int
	confirmed      bool
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit
	case "left", "right", "tab":
		m.cursor = 1 - m.cursor
	case "y":
		m.cursor = 0
	case "n":
		m.cursor = 1
	case "enter":
		m.confirmed = m.cursor == 0
		return m, tea.Quit
	}
	return m, nil
}

func (m confirmModel) View() string {
	choices := []string{m.yesLabel, m.noLabel}
	var builder strings.Builder
	builder.WriteString(titleStyle.Render(m.title) + "\n\n")
	builder.WriteString(m.message + "\n\n")
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

func confirmDuplicate(content string, candidates []plan) (bool, error) {
	var names []string
	for _, candidate := range candidates {
		names = append(names, fmt.Sprintf("• %s", candidate.Content))
	}
	message := fmt.Sprintf("%q is similar to:\n%s\n\nIs it a duplicate?", content, strings.Join(names, "\n"))
	final, err := tea.NewProgram(confirmModel{title: "Possible duplicate", message: message, yesLabel: "Yes, block this schedule", noLabel: "No, create it", cursor: 0}, tea.WithAltScreen()).Run()
	if err != nil {
		return false, err
	}
	return final.(confirmModel).confirmed, nil
}

func confirmSchedule(p plan) bool {
	count := len(occurrences(p))
	message := fmt.Sprintf("Create %d Todoist task(s) from %s to %s?\nRepeat: %s", count, p.StartDate.Format("02 Jan 2006"), p.EndDate.Format("02 Jan 2006"), recurrenceLabel(p))
	final, err := tea.NewProgram(confirmModel{title: "Create schedule", message: message, yesLabel: "Yes, create tasks", noLabel: "No, go back", cursor: 1}, tea.WithAltScreen()).Run()
	return err == nil && final.(confirmModel).confirmed
}

func recurrenceLabel(p plan) string {
	switch p.Recurrence {
	case "alternate":
		return "every other day"
	case "weekdays":
		names := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		var selected []string
		for _, day := range p.Weekdays {
			selected = append(selected, names[day])
		}
		return strings.Join(selected, ", ")
	default:
		return "every day"
	}
}
