package models

import "github.com/charmbracelet/lipgloss"

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	optionsStyle = lipgloss.NewStyle().
			Align(lipgloss.Left).
			Foreground(lipgloss.Color("#FAFAFA"))

	checkboxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA"))

	checkboxHighlightStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F25D94")).
				Bold(true)

	windowStyle = lipgloss.NewStyle().
			Align(lipgloss.Left).
			Foreground(lipgloss.Color("#FAFAFA")).
			BorderForeground(highlight)

	statusText = lipgloss.NewStyle().
			Align(lipgloss.Left).
			MarginTop(1).
			Foreground(lipgloss.Color("#FAFAFA"))

	text = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA"))

	header = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(highlight).
		MarginRight(2)
)
