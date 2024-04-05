package models

import (
	"context"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"metamakers.org/door-controller-mqtt/commands"
)

type MimicModel struct {
	username       string
	password       string
	mqttUri        string
	DocumentWindow DocumentWindow
}

func InitMinicModel(ctx context.Context, mqttUri string, username string, password string) MimicModel {
	physicalWidth, physicalHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}

	return MimicModel{
		mqttUri:        mqttUri,
		username:       username,
		password:       password,
		DocumentWindow: NewDocumentWindow(ctx, physicalWidth, physicalHeight),
	}
}

func (model MimicModel) UpdateDimensions(width int, height int) MimicModel {
	model.DocumentWindow = model.DocumentWindow.UpdateDimensions(width, height)
	return model
}

func (model MimicModel) Init() tea.Cmd {
	return commands.Init(
		model.mqttUri,
		model.username,
		model.password,
	)
}

func (model MimicModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model = model.UpdateDimensions(msg.Width, msg.Height)
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			cmds = append(cmds, tea.Quit)
		}
	}

	var documentWindowCmd tea.Cmd
	model.DocumentWindow, documentWindowCmd = model.DocumentWindow.Update(msg)
	cmds = append(cmds, documentWindowCmd)

	return model, tea.Batch(cmds...)
}

func (model MimicModel) View() string {
	return model.DocumentWindow.Render()
}
