package models

import (
	"errors"
	"regexp"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type TextInputWindow struct {
	TextInput     textinput.Model
	submitMessage func(value string) tea.Msg
	Window
}

func NewTextInputWindow(focused bool, submitMessage func(value string) tea.Msg, width int) TextInputWindow {
	textInput := textinput.New()
	textInput.Placeholder = "0001234567"
	textInput.CharLimit = 10
	textInput.Width = 10
	textInput.Validate = func(value string) error {
		matched, _ := regexp.Match(`^[0-9]+$`, []byte(value))
		if !matched && len(value) > 0 {
			return errors.New("Numbers only!")
		}
		return nil
	}

	textInputWindow := TextInputWindow{
		TextInput:     textInput,
		submitMessage: submitMessage,
		Window: Window{
			focused: focused,
			Width:   width,
			Height:  2,
			Margin:  Orientation{1, 0, 0, 0},
			Padding: Orientation{0, 0, 0, 0},
			Border:  Border{false, false, false, true},
		},
	}
	textInputWindow.SetWidth(width)
	textInputWindow.SetHeight(2)
	return textInputWindow
}

func (textInputWindow TextInputWindow) Focus() TextInputWindow {
	textInputWindow.Window.Focus()
	textInputWindow.TextInput.Focus()
	return textInputWindow
}

func (textInputWindow TextInputWindow) Blur() TextInputWindow {
	textInputWindow.Window.Blur()
	textInputWindow.TextInput.Blur()
	return textInputWindow
}

func submitTextCommand(textInputWindow TextInputWindow) tea.Cmd {
	return func() tea.Msg {
		return textInputWindow.submitMessage(textInputWindow.TextInput.Value())
	}
}

func (textInputWindow TextInputWindow) Update(msg tea.Msg) (TextInputWindow, tea.Cmd) {
	if !textInputWindow.Window.IsFocused() {
		return textInputWindow, nil
	}
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyEnter:
			cmds = append(cmds, submitTextCommand(textInputWindow))
			textInputWindow.TextInput.Reset()
		}
	}
	var doorTextCmd tea.Cmd
	textInputWindow.TextInput, doorTextCmd = textInputWindow.TextInput.Update(msg)
	cmds = append(cmds, doorTextCmd)
	return textInputWindow, tea.Batch(cmds...)
}

func (textInputWindow TextInputWindow) Render() string {
	errorMessage := ""
	if textInputWindow.TextInput.Err != nil {
		errorMessage = textInputWindow.TextInput.Err.Error()
	}
	return textInputWindow.Window.Render(
		textInputWindow.TextInput.View(),
		"\n",
		errorMessage,
	)
}
