package main

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			MarginBottom(1)
	promptStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252"))
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86"))
	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))
	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))
	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
)
