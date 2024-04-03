package models

import "github.com/charmbracelet/lipgloss"

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	checkboxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA"))

	checkboxHighlightStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F25D94")).
				Bold(true)

	leftPanelStyle = lipgloss.NewStyle().
			Align(lipgloss.Left).
			Foreground(lipgloss.Color("#FAFAFA")).
			Border(lipgloss.RoundedBorder()).
			BorderTop(true).
			BorderRight(true).
			BorderBottom(true).
			BorderLeft(true).
			BorderForeground(highlight)

	rightPanelStyle = lipgloss.NewStyle().
			Align(lipgloss.Left).
			Foreground(lipgloss.Color("#FAFAFA")).
			Border(lipgloss.RoundedBorder()).
			BorderTop(true).
			BorderRight(true).
			BorderBottom(true).
			BorderLeft(true).
			BorderForeground(highlight)

	text = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA"))

	header = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(highlight).
		MarginRight(2)
)
