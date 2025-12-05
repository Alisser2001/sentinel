package ui

import "github.com/charmbracelet/lipgloss"

// Styles split for readability
var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("cyan")).
			Align(lipgloss.Left)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)

	highCPUStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("red"))
	medCPUStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("yellow"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("red")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("green")).
			Bold(true)

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("yellow")).
			Bold(true).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("yellow")).
			Padding(1, 2)

	// Visual help styles
	sortedColumnStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("green")).
				Bold(true).
				Underline(true)

	helpBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			MarginTop(1)

	keybindStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	keybindDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true).
			Padding(0, 1)
)
