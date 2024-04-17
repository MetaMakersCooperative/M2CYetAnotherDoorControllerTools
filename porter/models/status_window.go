package models

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/eclipse/paho.golang/autopaho"
	"metamakers.org/door-controller-mqtt/commands"
	"metamakers.org/door-controller-mqtt/messages"
	"metamakers.org/door-controller-mqtt/mqtt"
)

type StatusWindow struct {
	serverConnection      *autopaho.ConnectionManager
	ctx                   context.Context
	mqttMessages          chan messages.MqttMessage
	mqttConnectionStatus  chan messages.MqttStatus
	clientID              string
	tabIndex              int
	maxTabIndex           int
	accessListState       bool
	failHealthCheckState  bool
	unluckState           bool
	deniedAccessState     bool
	code                  string
	Err                   error
	Spinner               spinner.Model
	IsConnected           bool
	Initialized           bool
	ResponseOptionsWindow ResponseOptionsWindow
	DoorTopicWindow       DoorTopicWindow
	TextInputWindow       TextInputWindow
	Window
}

const (
	AccessListKey      = "access_list"
	FailHealthCheckKey = "fail_health_check"
	DeniedAccessKey    = "denied_access"
	UnlockKey          = "unlock"
)

func NewStatusWindow(ctx context.Context, focused bool) StatusWindow {
	statusSpinnger := spinner.New()
	statusSpinnger.Spinner = spinner.Dot
	statusSpinnger.Style = spinnerStyle

	return StatusWindow{
		ctx:                  ctx,
		mqttConnectionStatus: make(chan messages.MqttStatus),
		mqttMessages:         make(chan messages.MqttMessage),
		clientID:             "",
		serverConnection:     nil,
		tabIndex:             0,
		maxTabIndex:          2,
		accessListState:      false,
		failHealthCheckState: false,
		Err:                  nil,
		Spinner:              statusSpinnger,
		IsConnected:          false,
		Initialized:          false,
		ResponseOptionsWindow: NewResponseOptionsWindow(
			false,
			0,
			KeyLabelPair{Key: AccessListKey, Label: "Error on access list"},
			KeyLabelPair{Key: FailHealthCheckKey, Label: "Fail health check"},
		),
		DoorTopicWindow: NewDoorTopicWindow(
			false,
			0,
			KeyLabelPair{Key: "unlock", Label: "Send unlock success"},
			KeyLabelPair{Key: "denied_access", Label: "Send unlock denied"},
		),
		TextInputWindow: NewTextInputWindow(
			false,
			func(value string) tea.Msg {
				return messages.DoorCodeTextMessage(value)
			},
			0,
		),
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
		switch msg.Topic {
		case mqtt.HealthCheckTopic:
			if !statusWindow.failHealthCheckState {
				cmds = append(cmds, commands.HealthCheckHandler(statusWindow.serverConnection, statusWindow.ctx, statusWindow.clientID))
			} else {
				cmds = append(cmds, commands.FailHealthCheckHandler(statusWindow.clientID))
			}
		case mqtt.AccessListTopic:
			if !statusWindow.accessListState {
				cmds = append(cmds, commands.AccessListHandler(statusWindow.serverConnection, statusWindow.ctx, statusWindow.clientID))
			} else {
				cmds = append(cmds, commands.FailAccessListHandler(statusWindow.serverConnection, statusWindow.ctx, statusWindow.clientID))
			}
		}
		cmds = append(cmds, commands.WaitForMessage(statusWindow.mqttMessages))
	case messages.MqttCredentials:
		statusWindow.clientID = msg.Username
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
	case messages.ResponseOptionsSelectionMessage:
		var exists bool
		if statusWindow.accessListState, exists = msg[AccessListKey]; !exists {
			statusWindow.accessListState = false
		}
		if statusWindow.failHealthCheckState, exists = msg[FailHealthCheckKey]; !exists {
			statusWindow.failHealthCheckState = false
		}
	case messages.DoorTopicSelectionMessage:
		var exists bool
		if statusWindow.unluckState, exists = msg[UnlockKey]; !exists {
			statusWindow.unluckState = false
		}
		if statusWindow.deniedAccessState, exists = msg[DeniedAccessKey]; !exists {
			statusWindow.deniedAccessState = false
		}
	case messages.DoorCodeTextMessage:
		statusWindow.code = string(msg)
		if statusWindow.unluckState {
			cmds = append(
				cmds,
				commands.PublishUnlock(statusWindow.serverConnection, statusWindow.ctx, statusWindow.clientID, statusWindow.code),
				commands.DelayCommandBy(
					time.Second*8,
					commands.PublishLock(statusWindow.serverConnection, statusWindow.ctx, statusWindow.clientID, statusWindow.code),
				),
			)
		} else if statusWindow.deniedAccessState {
			cmds = append(
				cmds,
				commands.PublishDeniedAccess(statusWindow.serverConnection, statusWindow.ctx, statusWindow.clientID, statusWindow.code),
			)
		}
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
			statusWindow.DoorTopicWindow = statusWindow.DoorTopicWindow.Blur()
			statusWindow.TextInputWindow = statusWindow.TextInputWindow.Blur()
		case 1:
			statusWindow.DoorTopicWindow = statusWindow.DoorTopicWindow.Focus()
			statusWindow.ResponseOptionsWindow = statusWindow.ResponseOptionsWindow.Blur()
			statusWindow.TextInputWindow = statusWindow.TextInputWindow.Blur()
		case 2:
			statusWindow.TextInputWindow = statusWindow.TextInputWindow.Focus()
			statusWindow.DoorTopicWindow = statusWindow.DoorTopicWindow.Blur()
			statusWindow.ResponseOptionsWindow = statusWindow.ResponseOptionsWindow.Blur()
		}
	} else {
		statusWindow.DoorTopicWindow = statusWindow.DoorTopicWindow.Blur()
		statusWindow.ResponseOptionsWindow = statusWindow.ResponseOptionsWindow.Blur()
		statusWindow.TextInputWindow = statusWindow.TextInputWindow.Blur()
	}

	var responseOptionsWindowCmd tea.Cmd
	statusWindow.ResponseOptionsWindow, responseOptionsWindowCmd = statusWindow.ResponseOptionsWindow.Update(msg)
	var doorTopicCmd tea.Cmd
	statusWindow.DoorTopicWindow, doorTopicCmd = statusWindow.DoorTopicWindow.Update(msg)
	var textInputCmd tea.Cmd
	statusWindow.TextInputWindow, textInputCmd = statusWindow.TextInputWindow.Update(msg)
	cmds = append(cmds, doorTopicCmd, textInputCmd, responseOptionsWindowCmd)

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
	statusWindow.DoorTopicWindow.SetWidth(statusWindow.GetInnerWidth())
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
		statusWindow.DoorTopicWindow.Render(),
		statusWindow.TextInputWindow.Render(),
	)
}
