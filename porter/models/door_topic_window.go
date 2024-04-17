package models

import (
	tea "github.com/charmbracelet/bubbletea"
	"metamakers.org/door-controller-mqtt/messages"
)

type DoorTopicWindow struct {
	DoorOptions Options
	Window
}

func NewDoorTopicWindow(focused bool, width int, keyLabelPairs ...KeyLabelPair) DoorTopicWindow {
	doorOptions := NewOptions(
		true,
		func(state map[string]bool) tea.Msg {
			return messages.DoorTopicSelectionMessage(state)
		},
		keyLabelPairs...,
	)
	doorTopicWindow := DoorTopicWindow{
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
	doorTopicWindow.SetWidth(width)
	doorTopicWindow.SetHeight(len(doorOptions.options) + 1)
	return doorTopicWindow
}

func (doorTopicWindow DoorTopicWindow) Focus() DoorTopicWindow {
	doorTopicWindow.Window.Focus()
	doorTopicWindow.DoorOptions = doorTopicWindow.DoorOptions.Focus()
	return doorTopicWindow
}

func (doorTopicWindow DoorTopicWindow) Blur() DoorTopicWindow {
	doorTopicWindow.Window.Blur()
	doorTopicWindow.DoorOptions = doorTopicWindow.DoorOptions.Blur()
	return doorTopicWindow
}

func (doorTopicWindow DoorTopicWindow) Update(msg tea.Msg) (DoorTopicWindow, tea.Cmd) {
	var doorTopicWindowCmd tea.Cmd
	doorTopicWindow.DoorOptions, doorTopicWindowCmd = doorTopicWindow.DoorOptions.Update(msg)
	return doorTopicWindow, doorTopicWindowCmd
}

func (doorTopicWindow DoorTopicWindow) Render() string {
	return doorTopicWindow.Window.Render(
		doorTopicWindow.DoorOptions.Render(),
	)
}
