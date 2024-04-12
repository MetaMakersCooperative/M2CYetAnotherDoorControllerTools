package models

import (
	tea "github.com/charmbracelet/bubbletea"
	"metamakers.org/door-controller-mqtt/messages"
)

type ResponseOptionsWindow struct {
	ResponseOptions Options
	Window
}

func NewResponseOptionsWindow(focused bool, width int, keyLabelPairs ...KeyLabelPair) ResponseOptionsWindow {
	responseOptions := NewOptions(
		false,
		func(state map[string]bool) tea.Msg {
			return messages.ResponseOptionsSelectionMessage(state)
		},
		keyLabelPairs...,
	)
	responseOptionsWindow := ResponseOptionsWindow{
		ResponseOptions: responseOptions,
		Window: Window{
			focused: focused,
			Width:   width,
			Height:  len(responseOptions.options),
			Margin:  Orientation{1, 0, 0, 0},
			Padding: Orientation{0, 0, 0, 0},
			Border:  Border{false, false, false, true},
		},
	}
	responseOptionsWindow.SetWidth(width)
	responseOptionsWindow.SetHeight(len(responseOptions.options))
	return responseOptionsWindow
}

func (responseOptionsWindow ResponseOptionsWindow) Focus() ResponseOptionsWindow {
	responseOptionsWindow.Window.Focus()
	responseOptionsWindow.ResponseOptions = responseOptionsWindow.ResponseOptions.Focus()
	return responseOptionsWindow
}

func (responseOptionsWindow ResponseOptionsWindow) Blur() ResponseOptionsWindow {
	responseOptionsWindow.Window.Blur()
	responseOptionsWindow.ResponseOptions = responseOptionsWindow.ResponseOptions.Blur()
	return responseOptionsWindow
}

func (responseOptionsWindow ResponseOptionsWindow) Update(msg tea.Msg) (ResponseOptionsWindow, tea.Cmd) {
	var responseOptionsWindowCmd tea.Cmd
	responseOptionsWindow.ResponseOptions, responseOptionsWindowCmd = responseOptionsWindow.ResponseOptions.Update(msg)
	return responseOptionsWindow, responseOptionsWindowCmd
}

func (responseOptionsWindow ResponseOptionsWindow) Render() string {
	return responseOptionsWindow.Window.Render(responseOptionsWindow.ResponseOptions.Render())
}
