package models

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type DoorMessageWindow struct {
	DoorOptions   Options
	DoorTextInput textinput.Model
	Window
}

func NewDoorMessageWindow(focused bool, width int) DoorMessageWindow {
	doorTextInput := textinput.New()
	doorTextInput.Placeholder = "0001234567"
	doorTextInput.CharLimit = 10
	doorTextInput.Width = 10

	doorOptions := NewOptions(
		true,
		KeyLabelPair{Key: "unlock", Label: "Send unlock denied"},
		KeyLabelPair{Key: "denied", Label: "Send unlock success"},
	)
	responseOptionsWindow := DoorMessageWindow{
		DoorOptions:   doorOptions,
		DoorTextInput: doorTextInput,
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

func (doorMessageWindow DoorMessageWindow) Update(msg tea.Msg) DoorMessageWindow {
	doorMessageWindow.DoorOptions = doorMessageWindow.DoorOptions.Update(msg)
	return doorMessageWindow
}

func (doorMessageWindow DoorMessageWindow) Render() string {
	return doorMessageWindow.Window.Render(doorMessageWindow.DoorOptions.Render())
}
