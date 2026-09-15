package main

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type addModel struct {
	step, cursor, projectIndex    int
	textCursor                    int
	content, startInput, endInput string
	startDate, endDate            time.Time
	recurrence                    string
	weekdays                      map[int16]bool
	priority                      int16
	projects                      []project
	duplicateCandidates           []plan
	cancelled, loading            bool
	creating, completed           bool
	createdTasks                  int
	creationError                 error
	errorMessage                  string
}

type similarPlansLoadedMsg struct {
	candidates []plan
	err        error
}

type projectsLoadedMsg struct {
	projects []project
	err      error
}

type scheduleCreatedMsg struct {
	count int
	err   error
}

func (m addModel) Init() tea.Cmd { return nil }

func (m addModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case similarPlansLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.duplicateCandidates = msg.candidates
		if len(msg.candidates) > 0 {
			m.step, m.cursor = 7, 0
			return m, nil
		}
		m.loading = true
		return m, loadAddProjects()
	case projectsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.projects, m.step, m.cursor = msg.projects, 1, 0
		return m, nil
	case scheduleCreatedMsg:
		m.creating = false
		m.createdTasks = msg.count
		m.creationError = msg.err
		m.completed = msg.err == nil
		return m, nil
	}
	key, ok := message.(tea.KeyMsg)
	if !ok || m.loading || m.creating {
		return m, nil
	}
	if m.completed {
		return m.updateCompletedSchedule(key)
	}
	if m.creationError != nil {
		return m, tea.Quit
	}
	pressed := key.String()
	if pressed == "ctrl+c" || pressed == "esc" {
		m.cancelled = true
		return m, tea.Quit
	}
	if pressed == "shift+tab" || (pressed == "backspace" && m.step > 2) {
		return m.goBack(), nil
	}
	if m.step <= 2 {
		return m.updateTextStep(key)
	}
	if m.step == 7 || m.step == 8 {
		return m.updateConfirmation(key)
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
			m.projectIndex = m.cursor
			m.step, m.cursor = 8, 1
		}
	}
	return m, nil
}

func (m addModel) goBack() addModel {
	switch m.step {
	case 0:
		return m
	case 5:
		if m.recurrence == "weekdays" {
			m.step = 4
		} else {
			m.step = 3
		}
	case 7:
		m.step = 0
	case 8:
		m.step = 6
	default:
		m.step--
	}
	m.errorMessage = ""
	m.cursor = 0
	m.textCursor = 0
	switch m.step {
	case 0:
		m.textCursor = len([]rune(m.content))
	case 1:
		m.textCursor = len([]rune(m.startInput))
	case 2:
		m.textCursor = len([]rune(m.endInput))
	case 3:
		for i, recurrence := range []string{"daily", "alternate", "weekdays"} {
			if m.recurrence == recurrence {
				m.cursor = i
				break
			}
		}
	case 5:
		if m.priority > 0 {
			m.cursor = int(m.priority - 1)
		}
	case 6:
		m.cursor = m.projectIndex
	}
	return m
}

func (m addModel) updateCompletedSchedule(key tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			return newAddModel(), nil
		}
		return m, tea.Quit
	}
	return m, nil
}

func newAddModel() addModel {
	return addModel{weekdays: map[int16]bool{}}
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
	case "left":
		if m.textCursor > 0 {
			m.textCursor--
		}
	case "right":
		if m.textCursor < len([]rune(*value)) {
			m.textCursor++
		}
	case "home", "ctrl+a":
		m.textCursor = 0
	case "end", "ctrl+e":
		m.textCursor = len([]rune(*value))
	case "backspace":
		if m.textCursor == 0 && m.step > 0 {
			return m.goBack(), nil
		}
		runes := []rune(*value)
		if m.textCursor > 0 {
			*value = string(joinRunes(runes[:m.textCursor-1], runes[m.textCursor:]))
			m.textCursor--
		}
	case "delete":
		runes := []rune(*value)
		if m.textCursor < len(runes) {
			*value = string(joinRunes(runes[:m.textCursor], runes[m.textCursor+1:]))
		}
	case "enter":
		if strings.TrimSpace(*value) == "" {
			m.errorMessage = label + " is required"
			return m, nil
		}
		if m.step > 0 {
			var date time.Time
			var err error
			if m.step == 1 {
				date, err = parseDate(*value, time.Now())
			} else {
				date, err = parseEndDate(*value, m.startDate, time.Now())
			}
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
		m.textCursor = 0
		if m.step == 1 {
			m.loading = true
			return m, loadSimilarPlans(m.content)
		}
	default:
		if len(key.Runes) > 0 {
			runes := []rune(*value)
			*value = string(joinRunes(runes[:m.textCursor], key.Runes, runes[m.textCursor:]))
			m.textCursor += len(key.Runes)
		}
	}
	return m, nil
}

func (m addModel) updateConfirmation(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "left", "right", "up", "down", "tab":
		m.cursor = 1 - m.cursor
	case "y":
		m.cursor = 0
	case "n":
		m.cursor = 1
	case "enter":
		if m.step == 7 {
			if m.cursor == 0 {
				m.cancelled = true
				return m, tea.Quit
			}
			m.loading = true
			return m, loadAddProjects()
		}
		if m.cursor == 0 {
			m.creating = true
			return m, createAddSchedule(m.toPlan())
		} else {
			m.cancelled = true
		}
		return m, tea.Quit
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
	if m.creating {
		builder.WriteString("\n" + promptStyle.Render("Creating Todoist tasks…") + "\n\n")
		builder.WriteString(mutedStyle.Render("Saving the shared schedule and creating every task in the selected range. Do not close this screen.") + "\n")
		return builder.String()
	}
	if m.completed {
		builder.WriteString("\n" + successStyle.Render(fmt.Sprintf("✓ Created %d Todoist task(s) for %q.", m.createdTasks, m.content)) + "\n\n")
		builder.WriteString(successStyle.Render(fmt.Sprintf("Scheduled from %s till %s.", formatScheduleDate(m.startDate), formatScheduleDate(m.endDate))) + "\n\n")
		builder.WriteString(promptStyle.Render("Do you want to schedule another task?") + "\n\n")
		return m.confirmationChoices(&builder, "Yes, schedule another task", "No, close")
	}
	if m.creationError != nil {
		builder.WriteString("\n" + warningStyle.Render("! Creation did not finish: "+m.creationError.Error()) + "\n\n")
		builder.WriteString(mutedStyle.Render("The schedule may contain partially created tasks. Press any key to close.") + "\n")
		return builder.String()
	}
	if m.loading {
		builder.WriteString("\n" + mutedStyle.Render("Checking existing schedules…") + "\n")
		return builder.String()
	}
	if m.step == 7 {
		var names []string
		for _, candidate := range m.duplicateCandidates {
			names = append(names, "• "+candidate.Content)
		}
		fmt.Fprintf(&builder, "%q is similar to:\n%s\n\nIs it a duplicate?\n\n", m.content, strings.Join(names, "\n"))
		return m.confirmationChoices(&builder, "Yes, block this schedule", "No, create it")
	}
	if m.step == 8 {
		p := m.toPlan()
		fmt.Fprintf(&builder, "Create %d Todoist task(s) from %s to %s?\nRepeat: %s\n\n", len(occurrences(p)), p.StartDate.Format("02 Jan 2006"), p.EndDate.Format("02 Jan 2006"), recurrenceLabel(p))
		return m.confirmationChoices(&builder, "Yes, create tasks", "No, cancel creation")
	}
	if m.step <= 2 {
		labels := []string{"Task text", "Start date", "End date"}
		values := []string{m.content, m.startInput, m.endInput}
		builder.WriteString(promptStyle.Render(labels[m.step]) + "\n\n")
		runes := []rune(values[m.step])
		input := string(runes[:m.textCursor]) + "█" + string(runes[m.textCursor:])
		builder.WriteString(inputStyle.Render(input) + "\n\n")
		if m.step > 0 {
			dateHelp := "Dates: today · tomorrow · 2026-08-20 · 20-08-2026 · 20/8/26"
			if m.step == 2 {
				dateHelp += " · End date: 1w (one week) · 2m (two months)"
			}
			builder.WriteString(mutedStyle.Render(dateHelp) + "\n")
		}
		help := "←/→ move · Home/End jump · Backspace/Delete edit · Enter continue"
		if m.step > 0 {
			help += " · Shift+Tab or Backspace at start: previous"
		}
		builder.WriteString(mutedStyle.Render(help+" · Esc cancel") + "\n")
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
		builder.WriteString("\n" + mutedStyle.Render("↑/↓ select · Enter continue · Shift+Tab/Backspace previous · Esc cancel") + "\n")
	}
	if m.errorMessage != "" {
		builder.WriteString("\n" + warningStyle.Render("! "+m.errorMessage) + "\n")
	}
	return builder.String()
}

func (m addModel) confirmationChoices(builder *strings.Builder, yes, no string) string {
	for index, choice := range []string{yes, no} {
		if index == m.cursor {
			builder.WriteString(selectedStyle.Render("› " + choice))
		} else {
			builder.WriteString("  " + choice)
		}
		builder.WriteString("\n")
	}
	help := "↑/↓ or ←/→ select · Enter confirm"
	if !m.completed {
		help += " · Shift+Tab/Backspace previous"
	}
	builder.WriteString("\n" + mutedStyle.Render(help+" · Esc cancel") + "\n")
	return builder.String()
}

func guidedAdd() error {
	model, err := runAddModel(newAddModel())
	if err != nil {
		return err
	}
	if model.cancelled {
		return nil
	}
	return nil
}

func (m addModel) toPlan() plan {
	weekdays := make([]int16, 0, len(m.weekdays))
	for day := int16(0); day < 7; day++ {
		if m.weekdays[day] {
			weekdays = append(weekdays, day)
		}
	}
	selectedProject := m.projects[m.projectIndex]
	return plan{ID: newScheduleID(m.content), Content: m.content, ProjectID: selectedProject.ID, StartDate: m.startDate, EndDate: m.endDate, Recurrence: m.recurrence, Weekdays: weekdays, Priority: &m.priority}
}

func newScheduleID(content string) string {
	sum := sha256.Sum256([]byte(content))
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return fmt.Sprintf("%s-%x-%d", slug(content), sum[:4], time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%x-%x", slug(content), sum[:4], randomBytes)
}

func runAddModel(model addModel) (addModel, error) {
	final, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return addModel{}, err
	}
	return final.(addModel), nil
}

func loadSimilarPlans(content string) tea.Cmd {
	return func() tea.Msg {
		candidates, err := similarPlans(content)
		return similarPlansLoadedMsg{candidates: candidates, err: err}
	}
}

func loadAddProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := todoistProjects()
		return projectsLoadedMsg{projects: projects, err: err}
	}
}

func createAddSchedule(p plan) tea.Cmd {
	return func() tea.Msg {
		if err := addPlan(p); err != nil {
			return scheduleCreatedMsg{err: err}
		}
		if err := createScheduleTasks(p); err != nil {
			if rollbackErr := removePlan(p.ID); rollbackErr != nil {
				return scheduleCreatedMsg{err: fmt.Errorf("task creation failed and the schedule could not be rolled back: %w", errors.Join(err, rollbackErr))}
			}
			return scheduleCreatedMsg{err: fmt.Errorf("task creation failed; the schedule was rolled back: %w", err)}
		}
		return scheduleCreatedMsg{count: len(occurrences(p))}
	}
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

func parseEndDate(value string, startDate, now time.Time) (time.Time, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) >= 2 {
		amount, err := strconv.Atoi(value[:len(value)-1])
		if err == nil && amount > 0 {
			switch value[len(value)-1] {
			case 'w':
				return startDate.AddDate(0, 0, amount*7), nil
			case 'm':
				return startDate.AddDate(0, amount, 0), nil
			}
		}
	}
	return parseDate(value, now)
}

func formatScheduleDate(date time.Time) string {
	return fmt.Sprintf("%d%s of %s %d", date.Day(), ordinalSuffix(date.Day()), date.Month(), date.Year())
}

func ordinalSuffix(day int) string {
	if day%100 >= 11 && day%100 <= 13 {
		return "th"
	}
	switch day % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}
