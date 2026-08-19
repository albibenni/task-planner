package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5"
)

func config() error {
	value, err := promptDatabaseURL()
	if err != nil {
		return err
	}
	if value == "" {
		return nil
	}
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "task-planner")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "environment")
	if err = os.WriteFile(path, []byte("SUPABASE_DB_URL="+value+"\n"), 0600); err != nil {
		return err
	}
	_ = os.Chmod(path, 0600)
	os.Setenv("SUPABASE_DB_URL", value)
	if err = withDB(func(context.Context, *pgx.Conn) error { return nil }); err != nil {
		return err
	}
	fmt.Println("Saved connection and initialized shared tables.")
	return nil
}

type configModel struct {
	value, validationError string
	cursor                 int
	done, cancelled        bool
}

func (m configModel) Init() tea.Cmd { return nil }

func (m configModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	pressed := key.String()
	if pressed == "ctrl+c" || pressed == "esc" {
		m.cancelled = true
		return m, tea.Quit
	}
	runes := []rune(m.value)
	switch pressed {
	case "enter":
		m.value = strings.TrimSpace(m.value)
		if err := validateDatabaseURL(m.value); err != nil {
			m.validationError = err.Error()
			return m, nil
		}
		m.done = true
		return m, tea.Quit
	case "left":
		if m.cursor > 0 {
			m.cursor--
		}
	case "right":
		if m.cursor < len(runes) {
			m.cursor++
		}
	case "home", "ctrl+a":
		m.cursor = 0
	case "end", "ctrl+e":
		m.cursor = len(runes)
	case "backspace":
		if m.cursor > 0 {
			m.value = string(joinRunes(runes[:m.cursor-1], runes[m.cursor:]))
			m.cursor--
		}
	case "delete":
		if m.cursor < len(runes) {
			m.value = string(joinRunes(runes[:m.cursor], runes[m.cursor+1:]))
		}
	default:
		if len(key.Runes) > 0 {
			m.value = string(joinRunes(runes[:m.cursor], key.Runes, runes[m.cursor:]))
			m.cursor += len(key.Runes)
		}
	}
	return m, nil
}

func joinRunes(parts ...[]rune) []rune {
	length := 0
	for _, part := range parts {
		length += len(part)
	}
	joined := make([]rune, 0, length)
	for _, part := range parts {
		joined = append(joined, part...)
	}
	return joined
}

func (m configModel) View() string {
	if m.done || m.cancelled {
		return ""
	}
	runes := []rune(m.value)
	input := string(runes[:m.cursor]) + "│" + string(runes[m.cursor:])
	var builder strings.Builder
	builder.WriteString(titleStyle.Render("Configure Supabase") + "\n")
	builder.WriteString(promptStyle.Render("Paste the Session Pooler URL (port 5432)") + "\n\n")
	builder.WriteString(inputStyle.Render(input))
	builder.WriteString("\n\n" + mutedStyle.Render("←/→ move · Home/End jump · Backspace/Delete edit · Enter save · Esc cancel") + "\n")
	if m.validationError != "" {
		builder.WriteString("\n" + warningStyle.Render("! "+m.validationError) + "\n")
	}
	return builder.String()
}

func promptDatabaseURL() (string, error) {
	final, err := tea.NewProgram(configModel{}, tea.WithAltScreen(), tea.WithoutBracketedPaste()).Run()
	if err != nil {
		return "", err
	}
	model := final.(configModel)
	if model.cancelled {
		return "", nil
	}
	return model.value, nil
}

func validateDatabaseURL(value string) error {
	connectionURL, err := url.Parse(value)
	if err != nil || connectionURL.Host == "" || (connectionURL.Scheme != "postgres" && connectionURL.Scheme != "postgresql") {
		return errors.New("enter a valid postgres:// or postgresql:// connection URL")
	}
	return nil
}
