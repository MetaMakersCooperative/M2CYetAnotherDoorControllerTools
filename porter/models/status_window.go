package models

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

type StatusWindow struct {
	Err         error
	Spinner     spinner.Model
	IsConnected bool
	Initialized bool
	Options     Options
	Window
}

func NewStatusWindow(style lipgloss.Style) StatusWindow {
	statusSpinnger := spinner.New()
	statusSpinnger.Spinner = spinner.Dot
	statusSpinnger.Style = style

	return StatusWindow{
		Spinner:     statusSpinnger,
		IsConnected: false,
		Initialized: false,
		Options: NewOptions(
			KeyLabelPair{Key: "access_list", Label: "Error on access list"},
			KeyLabelPair{Key: "health_check", Label: "Fail health check"},
		),
		Window: Window{
			Width:   0,
			Height:  0,
			Margin:  Orientation{1, 1, 0, 0},
			Padding: Orientation{1, 2, 1, 2},
			Border:  Border{true, true, true, true},
		},
	}
}
