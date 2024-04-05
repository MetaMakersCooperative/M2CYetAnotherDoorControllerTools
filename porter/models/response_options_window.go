package models

import tea "github.com/charmbracelet/bubbletea"

type ResponseOptionsWindow struct {
	ResponseOptions Options
	Window
}

func NewResponseOptionsWindow(focused bool, width int) ResponseOptionsWindow {
	responseOptions := NewOptions(
		false,
		KeyLabelPair{Key: "access_list", Label: "Error on access list"},
		KeyLabelPair{Key: "health_check", Label: "Fail health check"},
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

func (responseOptionsWindow ResponseOptionsWindow) Update(msg tea.Msg) ResponseOptionsWindow {
	responseOptionsWindow.ResponseOptions = responseOptionsWindow.ResponseOptions.Update(msg)
	return responseOptionsWindow
}

func (responseOptionsWindow ResponseOptionsWindow) Render() string {
	return responseOptionsWindow.Window.Render(responseOptionsWindow.ResponseOptions.Render())
}
