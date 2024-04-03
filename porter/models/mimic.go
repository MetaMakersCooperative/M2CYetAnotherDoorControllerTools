package models

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eclipse/paho.golang/autopaho"
	"golang.org/x/term"

	"metamakers.org/door-controller-mqtt/commands"
	"metamakers.org/door-controller-mqtt/messages"
)

type MimicModel struct {
	err                  error
	serverConnection     *autopaho.ConnectionManager
	username             string
	password             string
	mqttUri              string
	ctx                  context.Context
	mqttMessages         chan messages.MqttMessage
	mqttConnectionStatus chan messages.MqttStatus
	DocumentWindow       DocumentWindow
	LogWindow            LogWindow
	StatusWindow         StatusWindow
}

func InitMinicModel(ctx context.Context, mqttUri string, username string, password string) MimicModel {
	physicalWidth, physicalHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}

	model := MimicModel{
		err:                  nil,
		ctx:                  ctx,
		mqttUri:              mqttUri,
		username:             username,
		password:             password,
		mqttConnectionStatus: make(chan messages.MqttStatus),
		mqttMessages:         make(chan messages.MqttMessage),
		serverConnection:     nil,
		DocumentWindow:       NewDocumentWindow(),
		LogWindow:            NewLogWindow(),
		StatusWindow:         NewStatusWindow(spinnerStyle),
	}

	return model.UpdateDimensions(physicalWidth, physicalHeight)
}

func (model MimicModel) UpdateDimensions(width int, height int) MimicModel {
	model.DocumentWindow.SetWidth(width)
	model.DocumentWindow.SetHeight(height)

	innerWidth := model.DocumentWindow.GetInnerWidth()
	innerHeight := model.DocumentWindow.GetInnerHeight()

	model.StatusWindow.SetWidth(35)
	model.StatusWindow.SetHeight(innerHeight)

	model.LogWindow.SetWidth(innerWidth - 35)
	model.LogWindow.SetHeight(innerHeight)

	model.LogWindow.Viewport.Height = model.LogWindow.GetInnerHeight()
	model.LogWindow.Viewport.Width = model.LogWindow.GetInnerWidth()

	return model
}

func (model MimicModel) Init() tea.Cmd {
	return tea.Batch(
		commands.InitConnection(
			model.ctx,
			model.mqttConnectionStatus,
			model.mqttMessages,
			model.mqttUri,
			model.username,
			model.password,
		),
		commands.WaitForStatus(model.mqttConnectionStatus),
		model.StatusWindow.Spinner.Tick,
	)
}

func (model MimicModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 1)

	var viewportCmd tea.Cmd
	model.LogWindow.Viewport, viewportCmd = model.LogWindow.Viewport.Update(msg)
	cmds = append(cmds, viewportCmd)

	switch msg := msg.(type) {
	case messages.UrlParseError:
		model.LogWindow.Error("Failed to parse url: %s - Error: ", msg.URI, msg.Err)
	case messages.MqttServerConnection:
		if msg.Err != nil {
			model.LogWindow.Error("Failed to create connection to MQTT Broker: %v", msg.Err)
			cmds = append(cmds, commands.WaitForStatus(model.mqttConnectionStatus))
			break
		}
		model.StatusWindow.Initialized = true
		model.serverConnection = msg.Connnection
		cmds = append(cmds, model.StatusWindow.Spinner.Tick)
	case messages.MqttStatus:
		model.StatusWindow.IsConnected = msg.Connected
		if msg.Err == nil && msg.Connected {
			model.LogWindow.Info("Connected to MQTT Broker")
			cmds = append(
				cmds,
				commands.SubscribeToAccessList(model.serverConnection, model.ctx),
				commands.SubscribeToHealthCheck(model.serverConnection, model.ctx),
				commands.WaitForStatus(model.mqttConnectionStatus),
				commands.WaitForMessage(model.mqttMessages),
			)
			break
		}
		model.StatusWindow.Err = msg.Err
		if msg.Code == 254 {
			model.LogWindow.Error("Failed to connect to MQTT Broker: %v", msg.Err)
		} else if msg.Code == 255 {
			model.LogWindow.Error("MQTT Client error: %v", msg.Err)
		} else {
			model.LogWindow.Error("MQTT disconnect with reason: %s - code: %d", msg.Reason, msg.Code)
		}
		cmds = append(cmds, commands.WaitForStatus(model.mqttConnectionStatus))
	case messages.MqttMessage:
		model.LogWindow.Info("Received message from: %s", msg.Topic)
		model.LogWindow.Info("Payload is: %s", msg.Payload)
		cmds = append(cmds, commands.WaitForMessage(model.mqttMessages))
	case messages.PublishMessage:
		if msg.Err != nil {
			model.LogWindow.Error("Failed to publish to: %s - Error: %v", msg.Topic, msg.Err)
		} else {
			model.LogWindow.Info("Published to topic: %s - Payload: %s", msg.Topic, msg.Payload)
		}
	case messages.SubscribeMessage:
		if msg.Err != nil {
			model.LogWindow.Error("Failed to subscribe to: %s - Error: %v", msg.Topic, msg.Err)
		} else {
			model.LogWindow.Info("Subscribed to topic: %s", msg.Topic)
		}
	case tea.WindowSizeMsg:
		model = model.UpdateDimensions(msg.Width, msg.Height)
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			if model.serverConnection != nil {
				model.serverConnection.Disconnect(model.ctx)
			}
			return model, tea.Quit
		}
	case spinner.TickMsg:
		var spinnerCmd tea.Cmd
		model.StatusWindow.Spinner, spinnerCmd = model.StatusWindow.Spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	// Needs to happen after tea.WindowSizeMsg is handled so that
	// the log lines are rendered correctly
	isAtBottom := model.LogWindow.Viewport.AtBottom()
	model.LogWindow.Viewport.SetContent(model.LogWindow.Render())
	if isAtBottom {
		model.LogWindow.Viewport.GotoBottom()
	}

	return model, tea.Batch(cmds...)
}

func (model MimicModel) View() string {

	if model.DocumentWindow.Height < 20 || model.DocumentWindow.Width < 80 {
		return lipgloss.NewStyle().
			Foreground(highlight).
			Render(
				"Terminal window needs to be larger than 80x20 to show information",
			)
	}

	docStyle := lipgloss.NewStyle().
		PaddingTop(model.DocumentWindow.Padding.Top).
		PaddingLeft(model.DocumentWindow.Padding.Left).
		PaddingBottom(model.DocumentWindow.Padding.Bottom).
		PaddingRight(model.DocumentWindow.Padding.Right)

	docStyle.MaxHeight(model.DocumentWindow.Height)
	docStyle.MaxWidth(model.DocumentWindow.Width)

	doc := strings.Builder{}

	var status string
	if model.err != nil {
		status = fmt.Sprintf("%s Failed to start connection manager", model.StatusWindow.Spinner.View())
	} else if model.serverConnection == nil {
		status = fmt.Sprintf("%s Starting connected manager", model.StatusWindow.Spinner.View())
	} else if !model.StatusWindow.IsConnected {
		status = fmt.Sprintf("%s Attempting to connect", model.StatusWindow.Spinner.View())
	} else {
		status = fmt.Sprintf("%s Connected to MQTT Broker", model.StatusWindow.Spinner.View())
	}

	left := leftPanelStyle.
		Copy().
		Height(model.StatusWindow.Height).
		Width(model.StatusWindow.Width).
		MarginTop(model.StatusWindow.Margin.Top).
		MarginLeft(model.StatusWindow.Margin.Left).
		MarginBottom(model.StatusWindow.Margin.Bottom).
		MarginRight(model.StatusWindow.Margin.Right).
		PaddingTop(model.StatusWindow.Padding.Top).
		PaddingLeft(model.StatusWindow.Padding.Left).
		PaddingBottom(model.StatusWindow.Padding.Bottom).
		PaddingRight(model.StatusWindow.Padding.Right).
		Align(lipgloss.Left).
		Render(
			header.Render("Connection Status"),
			"\n",
			text.Render(status),
			header.Copy().MarginTop(2).Render("Options"),
			"\n",
			model.StatusWindow.Options.Render(),
		)

	right := rightPanelStyle.
		Copy().
		Height(model.LogWindow.Height).
		Width(model.LogWindow.Width).
		MarginTop(model.LogWindow.Margin.Top).
		MarginLeft(model.LogWindow.Margin.Left).
		MarginBottom(model.LogWindow.Margin.Bottom).
		MarginRight(model.LogWindow.Margin.Right).
		PaddingTop(model.LogWindow.Padding.Top).
		PaddingLeft(model.LogWindow.Padding.Left).
		PaddingBottom(model.LogWindow.Padding.Bottom).
		PaddingRight(model.LogWindow.Padding.Right).
		Render(model.LogWindow.Viewport.View())

	doc.WriteString(lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		right,
	))

	doc.WriteString("\n\n")

	return docStyle.Render(doc.String())
}
