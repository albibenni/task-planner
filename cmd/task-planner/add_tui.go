package main

import (
	"crypto/sha256"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type addModel struct {
	step, cursor    int
	content, due    string
	period          string
	priority        int16
	projects        []project
	done, cancelled bool
}

func (m addModel) Init() tea.Cmd { return nil }

func (m addModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	pressed := key.String()
	if pressed == "ctrl+c" || pressed == "esc" {
		m.cancelled = true
		return m, tea.Quit
	}
	if m.step == 0 {
		if pressed == "enter" && strings.TrimSpace(m.content) != "" {
			m.step++
			m.cursor = 0
		} else if pressed == "backspace" && len(m.content) > 0 {
			m.content = m.content[:len(m.content)-1]
		} else if len(pressed) == 1 {
			m.content += pressed
		}
		return m, nil
	}
	choices := m.choices()
	if pressed == "up" || pressed == "k" {
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	}
	if pressed == "down" || pressed == "j" {
		if m.cursor < len(choices)-1 {
			m.cursor++
		}
		return m, nil
	}
	if pressed == "enter" {
		switch m.step {
		case 1:
			m.period = choices[m.cursor]
			m.step++
			m.cursor = 0
		case 2:
			m.due = choices[m.cursor]
			m.step++
			m.cursor = 0
		case 3:
			m.priority = int16(m.cursor + 1)
			m.step++
			m.cursor = 0
		case 4:
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m addModel) choices() []string {
	switch m.step {
	case 1:
		return []string{"day", "week", "month"}
	case 2:
		return []string{"today", "tomorrow", "No due date"}
	case 3:
		return []string{"1 — normal", "2 — medium", "3 — high", "4 — urgent"}
	case 4:
		choices := make([]string, len(m.projects))
		for i, p := range m.projects {
			choices[i] = p.Name
		}
		return choices
	}
	return nil
}

func (m addModel) View() string {
	if m.done {
		return "Plan ready.\n"
	}
	if m.cancelled {
		return "Cancelled.\n"
	}
	if m.step == 0 {
		return "Add a shared Todoist plan\n\nTask text: " + m.content + "█\n\nEnter to continue · Esc to cancel\n"
	}
	labels := []string{"Period", "Due date", "Priority", "Destination project"}
	var builder strings.Builder
	builder.WriteString("Add a shared Todoist plan\n\n" + labels[m.step-1] + ":\n\n")
	for i, choice := range m.choices() {
		marker := "  "
		if i == m.cursor {
			marker = "› "
		}
		builder.WriteString(marker + choice + "\n")
	}
	builder.WriteString("\n↑/↓ select · Enter continue · Esc cancel\n")
	return builder.String()
}

func guidedAdd() error {
	projects, err := todoistProjects()
	if err != nil {
		return err
	}
	final, err := tea.NewProgram(addModel{projects: projects}, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	model := final.(addModel)
	if model.cancelled {
		return nil
	}
	due := model.due
	if due == "No due date" {
		due = ""
	}
	var duePtr *string
	if due != "" {
		duePtr = &due
	}
	selectedProject := model.projects[model.cursor]
	sum := sha256.Sum256([]byte(model.content))
	id := fmt.Sprintf("%s-%x", slug(model.content), sum[:4])
	newPlan := plan{ID: id, Content: model.content, Period: model.period, ProjectID: selectedProject.ID, DueString: duePtr, Priority: &model.priority}
	if err := addPlan(newPlan); err != nil {
		return err
	}
	fmt.Printf("Added %q to %s.\n", newPlan.Content, selectedProject.Name)
	return nil
}

func slug(text string) string {
	var builder strings.Builder
	dash := false
	for _, character := range strings.ToLower(text) {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			builder.WriteRune(character)
			dash = false
		} else if !dash {
			builder.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
