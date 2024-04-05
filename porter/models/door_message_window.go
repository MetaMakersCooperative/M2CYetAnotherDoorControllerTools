package models

import (
	tea "github.com/charmbracelet/bubbletea"
)

type DoorMessageWindow struct {
	DoorOptions Options
	Window
}

func NewDoorMessageWindow(focused bool, width int) DoorMessageWindow {
	doorOptions := NewOptions(
		true,
		KeyLabelPair{Key: "unlock", Label: "Send unlock denied"},
		KeyLabelPair{Key: "denied", Label: "Send unlock success"},
	)
	responseOptionsWindow := DoorMessageWindow{
		DoorOptions: doorOptions,
		Window: Window{
			focused: focused,
			Width:   width,
			Height:  len(doorOptions.options) + 1,
			Margin:  Orientation{1, 0, 0, 0},
			Padding: Orientation{0, 0, 0, 0},
			Border:  Border{false, false, false, true},
		},
	}
	responseOptionsWindow.SetWidth(width)
	responseOptionsWindow.SetHeight(len(doorOptions.options) + 1)
	return responseOptionsWindow
}

func (doorMessageWindow DoorMessageWindow) Focus() DoorMessageWindow {
	doorMessageWindow.Window.Focus()
	doorMessageWindow.DoorOptions = doorMessageWindow.DoorOptions.Focus()
	return doorMessageWindow
}

func (doorMessageWindow DoorMessageWindow) Blur() DoorMessageWindow {
	doorMessageWindow.Window.Blur()
	doorMessageWindow.DoorOptions = doorMessageWindow.DoorOptions.Blur()
	return doorMessageWindow
}

func (doorMessageWindow DoorMessageWindow) Update(msg tea.Msg) (DoorMessageWindow, tea.Cmd) {
	doorMessageWindow.DoorOptions = doorMessageWindow.DoorOptions.Update(msg)
	return doorMessageWindow, nil
}

func (doorMessageWindow DoorMessageWindow) Render() string {
	return doorMessageWindow.Window.Render(
		doorMessageWindow.DoorOptions.Render(),
	)
}
