package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const defaultProjectPickerPageSize = 10

type defaultProjectPickerModel struct {
	projects  []project
	cursor    int
	confirmed bool
}

func newDefaultProjectPickerModel(projects []project, savedID string) defaultProjectPickerModel {
	model := defaultProjectPickerModel{projects: projects}
	for index, candidate := range projects {
		if candidate.ID == savedID {
			model.cursor = index
			break
		}
	}
	return model
}

func (m defaultProjectPickerModel) Init() tea.Cmd { return nil }

func (m defaultProjectPickerModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.projects)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.projects) > 0 {
			m.confirmed = true
			return m, tea.Quit
		}
	case "esc", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m defaultProjectPickerModel) View() string {
	var builder strings.Builder
	builder.WriteString(titleStyle.Render("Choose a default destination project"))
	builder.WriteString("\n")
	if len(m.projects) == 0 {
		builder.WriteString("No Todoist projects are available.\n")
	} else {
		start := (m.cursor / defaultProjectPickerPageSize) * defaultProjectPickerPageSize
		end := min(start+defaultProjectPickerPageSize, len(m.projects))
		for index := start; index < end; index++ {
			candidate := m.projects[index]
			line := fmt.Sprintf("%s (%s)", candidate.Name, candidate.ID)
			if index == m.cursor {
				builder.WriteString(selectedStyle.Render("› " + line))
			} else {
				builder.WriteString("  " + line)
			}
			builder.WriteString("\n")
		}
		builder.WriteString(mutedStyle.Render(fmt.Sprintf("%d of %d", m.cursor+1, len(m.projects))))
		builder.WriteString("\n")
	}
	builder.WriteString(mutedStyle.Render("↑/↓ choose · Enter save · Esc cancel"))
	builder.WriteString("\n")
	return builder.String()
}

func selectDefaultProject() error {
	projects, err := todoistProjects()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return fmt.Errorf("no Todoist projects are available")
	}
	savedID, err := readDefaultProjectID()
	if err != nil {
		return err
	}
	final, err := tea.NewProgram(newDefaultProjectPickerModel(projects, savedID), tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	selected := final.(defaultProjectPickerModel)
	if !selected.confirmed {
		return nil
	}
	return saveDefaultProject(selected.projects[selected.cursor])
}
