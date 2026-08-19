package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type addModel struct {
	step, cursor                  int
	content, startInput, endInput string
	startDate, endDate            time.Time
	recurrence                    string
	weekdays                      map[int16]bool
	priority                      int16
	projects                      []project
	done, cancelled               bool
	errorMessage                  string
}

func (m addModel) Init() tea.Cmd { return nil }

func (m addModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	pressed := key.String()
	if pressed == "ctrl+c" || pressed == "esc" {
		m.cancelled = true
		return m, tea.Quit
	}
	if m.step <= 2 {
		return m.updateTextStep(key)
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
	if m.step == 4 && pressed == " " {
		day := int16((m.cursor + 1) % 7)
		m.weekdays[day] = !m.weekdays[day]
		return m, nil
	}
	if pressed == "enter" {
		switch m.step {
		case 3:
			m.recurrence = []string{"daily", "alternate", "weekdays"}[m.cursor]
			m.step++
			m.cursor = 0
			if m.recurrence != "weekdays" {
				m.step++
			}
		case 4:
			if len(m.weekdays) == 0 {
				m.errorMessage = "select at least one weekday"
				return m, nil
			}
			m.step++
			m.cursor = 0
		case 5:
			m.priority = int16(m.cursor + 1)
			m.step++
			m.cursor = 0
		case 6:
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m addModel) updateTextStep(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	pressed := key.String()
	value := &m.content
	label := "task text"
	if m.step == 1 {
		value, label = &m.startInput, "start date"
	}
	if m.step == 2 {
		value, label = &m.endInput, "end date"
	}
	switch pressed {
	case "backspace":
		runes := []rune(*value)
		if len(runes) > 0 {
			*value = string(runes[:len(runes)-1])
		}
	case "enter":
		if strings.TrimSpace(*value) == "" {
			m.errorMessage = label + " is required"
			return m, nil
		}
		if m.step > 0 {
			date, err := parseDate(*value, time.Now())
			if err != nil {
				m.errorMessage = err.Error()
				return m, nil
			}
			if m.step == 1 {
				m.startDate = date
			} else {
				if date.Before(m.startDate) {
					m.errorMessage = "end date must be on or after the start date"
					return m, nil
				}
				m.endDate = date
			}
		}
		m.errorMessage = ""
		m.step++
		m.cursor = 0
	default:
		if len(key.Runes) > 0 {
			*value += string(key.Runes)
		}
	}
	return m, nil
}

func (m addModel) choices() []string {
	switch m.step {
	case 3:
		return []string{"Every day", "Every other day", "Selected weekdays"}
	case 4:
		return []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	case 5:
		return []string{"1 — normal", "2 — medium", "3 — high", "4 — urgent"}
	case 6:
		choices := make([]string, len(m.projects))
		for i, p := range m.projects {
			choices[i] = p.Name
		}
		return choices
	}
	return nil
}

func (m addModel) View() string {
	if m.cancelled {
		return mutedStyle.Render("Cancelled.") + "\n"
	}
	var builder strings.Builder
	builder.WriteString(titleStyle.Render("Add a shared Todoist schedule") + "\n")
	if m.step <= 2 {
		labels := []string{"Task text", "Start date", "End date"}
		values := []string{m.content, m.startInput, m.endInput}
		builder.WriteString(promptStyle.Render(labels[m.step]) + "\n\n")
		builder.WriteString(inputStyle.Render(values[m.step]+"█") + "\n\n")
		if m.step > 0 {
			builder.WriteString(mutedStyle.Render("Dates: today · tomorrow · 2026-08-20 · 20-08-2026 · 20/8/26") + "\n")
		}
		builder.WriteString(mutedStyle.Render("Enter continue · Backspace edit · Esc cancel") + "\n")
	} else {
		labels := map[int]string{3: "Repeat", 4: "Choose weekdays (Space toggles)", 5: "Priority", 6: "Destination project"}
		builder.WriteString(promptStyle.Render(labels[m.step]) + "\n\n")
		for i, choice := range m.choices() {
			prefix := "  "
			if m.step == 4 && m.weekdays[int16((i+1)%7)] {
				prefix = "✓ "
			}
			line := prefix + choice
			if i == m.cursor {
				builder.WriteString(selectedStyle.Render("› " + line))
			} else {
				builder.WriteString("  " + line)
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n" + mutedStyle.Render("↑/↓ select · Enter continue · Esc cancel") + "\n")
	}
	if m.errorMessage != "" {
		builder.WriteString("\n" + warningStyle.Render("! "+m.errorMessage) + "\n")
	}
	return builder.String()
}

func guidedAdd() error {
	projects, err := todoistProjects()
	if err != nil {
		return err
	}
	final, err := tea.NewProgram(addModel{projects: projects, weekdays: map[int16]bool{}}, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	model := final.(addModel)
	if model.cancelled {
		return nil
	}
	weekdays := make([]int16, 0, len(model.weekdays))
	for day := int16(0); day < 7; day++ {
		if model.weekdays[day] {
			weekdays = append(weekdays, day)
		}
	}
	sum := sha256.Sum256([]byte(model.content))
	selectedProject := model.projects[model.cursor]
	p := plan{ID: fmt.Sprintf("%s-%x", slug(model.content), sum[:4]), Content: model.content, ProjectID: selectedProject.ID, StartDate: model.startDate, EndDate: model.endDate, Recurrence: model.recurrence, Weekdays: weekdays, Priority: &model.priority}
	candidates, err := similarPlans(p.Content)
	if err != nil {
		return err
	}
	if len(candidates) > 0 {
		duplicate, err := confirmDuplicate(p.Content, candidates)
		if err != nil || duplicate {
			return err
		}
	}
	if !confirmSchedule(p) {
		return nil
	}
	if err := addPlan(p); err != nil {
		return err
	}
	if err := createScheduleTasks(p); err != nil {
		return fmt.Errorf("schedule saved but task creation did not finish: %w", err)
	}
	fmt.Printf("Created %d Todoist tasks for %q.\n", len(occurrences(p)), p.Content)
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

func parseDate(value string, now time.Time) (time.Time, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch value {
	case "today":
		return today, nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	}
	for _, layout := range []string{"2006-01-02", "02-01-2006", "02/01/2006", "2-1-2006", "2/1/2006", "2-1-06", "2/1/06"} {
		if date, err := time.ParseInLocation(layout, value, now.Location()); err == nil {
			return date, nil
		}
	}
	return time.Time{}, errors.New("use today, tomorrow, YYYY-MM-DD, DD-MM-YYYY, or DD/MM/YY")
}
