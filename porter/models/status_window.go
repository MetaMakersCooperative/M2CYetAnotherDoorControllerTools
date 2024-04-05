package models

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/eclipse/paho.golang/autopaho"
	"metamakers.org/door-controller-mqtt/commands"
	"metamakers.org/door-controller-mqtt/messages"
)

type StatusWindow struct {
	serverConnection      *autopaho.ConnectionManager
	ctx                   context.Context
	mqttMessages          chan messages.MqttMessage
	mqttConnectionStatus  chan messages.MqttStatus
	tabIndex              int
	maxTabIndex           int
	Err                   error
	Spinner               spinner.Model
	IsConnected           bool
	Initialized           bool
	ResponseOptionsWindow ResponseOptionsWindow
	DoorMessageWindow     DoorMessageWindow
	TextInputWindow       TextInputWindow
	Window
}

func NewStatusWindow(ctx context.Context, focused bool) StatusWindow {
	statusSpinnger := spinner.New()
	statusSpinnger.Spinner = spinner.Dot
	statusSpinnger.Style = spinnerStyle

	return StatusWindow{
		ctx:                   ctx,
		mqttConnectionStatus:  make(chan messages.MqttStatus),
		mqttMessages:          make(chan messages.MqttMessage),
		serverConnection:      nil,
		tabIndex:              0,
		maxTabIndex:           2,
		Err:                   nil,
		Spinner:               statusSpinnger,
		IsConnected:           false,
		Initialized:           false,
		ResponseOptionsWindow: NewResponseOptionsWindow(false, 0),
		DoorMessageWindow:     NewDoorMessageWindow(false, 0),
		TextInputWindow:       NewTextInputWindow(false, 0),
		Window: Window{
			focused: focused,
			Width:   0,
			Height:  0,
			Margin:  Orientation{1, 1, 0, 0},
			Padding: Orientation{1, 2, 1, 2},
			Border:  Border{true, true, true, true},
		},
	}
}

func (statusWindow StatusWindow) Update(msg tea.Msg) (StatusWindow, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case messages.UrlParseError:
		statusWindow.Err = msg.Err
	case messages.MqttServerConnection:
		if msg.Err != nil {
			statusWindow.Initialized = false
			statusWindow.Err = msg.Err
			cmds = append(cmds, commands.WaitForStatus(statusWindow.mqttConnectionStatus))
			break
		}
		statusWindow.Initialized = true
		statusWindow.serverConnection = msg.Connnection
		cmds = append(cmds, statusWindow.Spinner.Tick)
	case messages.MqttStatus:
		statusWindow.IsConnected = msg.Connected
		if msg.Err == nil && msg.Connected {
			cmds = append(
				cmds,
				commands.SubscribeToAccessList(statusWindow.serverConnection, statusWindow.ctx),
				commands.SubscribeToHealthCheck(statusWindow.serverConnection, statusWindow.ctx),
				commands.WaitForStatus(statusWindow.mqttConnectionStatus),
				commands.WaitForMessage(statusWindow.mqttMessages),
			)
			break
		}
		cmds = append(cmds, commands.WaitForStatus(statusWindow.mqttConnectionStatus))
	case messages.MqttMessage:
		cmds = append(cmds, commands.WaitForMessage(statusWindow.mqttMessages))
	case messages.MqttCredentials:
		cmds = append(cmds,
			commands.InitConnection(
				statusWindow.ctx,
				statusWindow.mqttConnectionStatus,
				statusWindow.mqttMessages,
				msg.URI,
				msg.Username,
				msg.Password,
			),
			commands.WaitForStatus(statusWindow.mqttConnectionStatus),
			statusWindow.Spinner.Tick,
		)
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			if statusWindow.serverConnection != nil {
				statusWindow.serverConnection.Disconnect(statusWindow.ctx)
			}
		} else if msg.Type == tea.KeyTab {
			if statusWindow.IsFocused() {
				if statusWindow.tabIndex+1 > statusWindow.maxTabIndex {
					statusWindow.tabIndex = 0
				} else {
					statusWindow.tabIndex += 1
				}
			}
		}
	case spinner.TickMsg:
		var spinnerCmd tea.Cmd
		statusWindow.Spinner, spinnerCmd = statusWindow.Spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	if statusWindow.Window.IsFocused() {
		switch statusWindow.tabIndex {
		case 0:
			statusWindow.ResponseOptionsWindow = statusWindow.ResponseOptionsWindow.Focus()
			statusWindow.DoorMessageWindow = statusWindow.DoorMessageWindow.Blur()
			statusWindow.TextInputWindow = statusWindow.TextInputWindow.Blur()
		case 1:
			statusWindow.DoorMessageWindow = statusWindow.DoorMessageWindow.Focus()
			statusWindow.ResponseOptionsWindow = statusWindow.ResponseOptionsWindow.Blur()
			statusWindow.TextInputWindow = statusWindow.TextInputWindow.Blur()
		case 2:
			statusWindow.TextInputWindow = statusWindow.TextInputWindow.Focus()
			statusWindow.DoorMessageWindow = statusWindow.DoorMessageWindow.Blur()
			statusWindow.ResponseOptionsWindow = statusWindow.ResponseOptionsWindow.Blur()
		}
	} else {
		statusWindow.DoorMessageWindow = statusWindow.DoorMessageWindow.Blur()
		statusWindow.ResponseOptionsWindow = statusWindow.ResponseOptionsWindow.Blur()
		statusWindow.TextInputWindow = statusWindow.TextInputWindow.Blur()
	}

	statusWindow.ResponseOptionsWindow = statusWindow.ResponseOptionsWindow.Update(msg)
	var doorMessageCmd tea.Cmd
	statusWindow.DoorMessageWindow, doorMessageCmd = statusWindow.DoorMessageWindow.Update(msg)
	var textInputCmd tea.Cmd
	statusWindow.TextInputWindow, textInputCmd = statusWindow.TextInputWindow.Update(msg)
	cmds = append(cmds, doorMessageCmd, textInputCmd)

	return statusWindow, tea.Batch(cmds...)
}

func (statusWindow *StatusWindow) Focus() {
	statusWindow.Window.Focus()
}

func (statusWindow *StatusWindow) Blur() {
	statusWindow.Window.Blur()
}

func (statusWindow StatusWindow) UpdateDimensions(width int, height int) StatusWindow {
	statusWindow.SetWidth(width)
	statusWindow.SetHeight(height)
	statusWindow.ResponseOptionsWindow.SetWidth(statusWindow.GetInnerWidth())
	statusWindow.DoorMessageWindow.SetWidth(statusWindow.GetInnerWidth())
	statusWindow.TextInputWindow.SetWidth(statusWindow.GetInnerWidth())
	return statusWindow
}

func (statusWindow *StatusWindow) Render() string {
	var status string
	if statusWindow.Err != nil && !statusWindow.Initialized {
		status = fmt.Sprintf("%s Failed to start connection manager", statusWindow.Spinner.View())
	} else if !statusWindow.Initialized {
		status = fmt.Sprintf("%s Starting connected manager", statusWindow.Spinner.View())
	} else if statusWindow.Initialized && !statusWindow.IsConnected {
		status = fmt.Sprintf("%s Attempting to connect", statusWindow.Spinner.View())
	} else if statusWindow.Initialized && statusWindow.IsConnected {
		status = fmt.Sprintf("%s Connected to MQTT Broker", statusWindow.Spinner.View())
	} else {
		status = fmt.Sprintf("%s Unknown status", statusWindow.Spinner.View())
	}

	return statusWindow.Window.Render(
		header.Render("Connection Status"),
		statusText.Render(status),
		header.Copy().MarginTop(2).Render("Options"),
		statusWindow.ResponseOptionsWindow.Render(),
		header.Copy().MarginTop(2).Render("Send Door Message"),
		statusWindow.DoorMessageWindow.Render(),
		statusWindow.TextInputWindow.Render(),
	)
}
