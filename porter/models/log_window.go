package models

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
	"metamakers.org/door-controller-mqtt/messages"
)

type LogWindow struct {
	logs     []string
	err      error
	Viewport viewport.Model
	Window
}

func NewLogWindow(focused bool) LogWindow {
	return LogWindow{
		logs:     make([]string, 0),
		Viewport: viewport.New(0, 0),
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

func (logWindow LogWindow) Update(msg tea.Msg) (LogWindow, tea.Cmd) {
	var viewportCmd tea.Cmd
	if logWindow.IsFocused() {
		logWindow.Viewport, viewportCmd = logWindow.Viewport.Update(msg)
	}

	switch msg := msg.(type) {
	case messages.UrlParseError:
		logWindow.Error("Failed to parse url: %s - Error: ", msg.URI, msg.Err)
	case messages.MqttServerConnection:
		if msg.Err != nil {
			logWindow.Error("Failed to create connection to MQTT Broker: %v", msg.Err)
		}
	case messages.MqttStatus:
		if msg.Err == nil && msg.Connected {
			logWindow.Info("Connected to MQTT Broker")
			break
		}
		if msg.Code == 254 {
			logWindow.Error("Failed to connect to MQTT Broker: %v", msg.Err)
		} else if msg.Code == 255 {
			logWindow.Error("MQTT Client error: %v", msg.Err)
		} else {
			logWindow.Error("MQTT disconnect with reason: %s - code: %d", msg.Reason, msg.Code)
		}
	case messages.MqttMessage:
		logWindow.Info("Received message from: %s", msg.Topic)
		logWindow.Info("Payload is: %s", msg.Payload)
	case messages.PublishMessage:
		if msg.Err != nil {
			logWindow.Error("Failed to publish to: %s - Error: %v", msg.Topic, msg.Err)
		} else {
			logWindow.Info("Published to topic: %s - Payload: %s", msg.Topic, msg.Payload)
		}
	case messages.SubscribeMessage:
		if msg.Err != nil {
			logWindow.Error("Failed to subscribe to: %s - Error: %v", msg.Topic, msg.Err)
		} else {
			logWindow.Info("Subscribed to topic: %s", msg.Topic)
		}
	}

	// Needs to happen after tea.WindowSizeMsg is handled so that
	// the log lines are rendered correctly
	isAtBottom := logWindow.Viewport.AtBottom()
	logWindow.Viewport.SetContent(logWindow.RenderContent())
	if isAtBottom {
		logWindow.Viewport.GotoBottom()
	}

	return logWindow, viewportCmd
}

func (logWindow LogWindow) UpdateDimensions(width int, height int) LogWindow {
	logWindow.SetWidth(width)
	logWindow.SetHeight(height)

	logWindow.Viewport.Height = logWindow.GetInnerHeight()
	logWindow.Viewport.Width = logWindow.GetInnerWidth()

	return logWindow
}

func (logWindow *LogWindow) Log(prefix string, format string, args ...any) {
	logWindow.logs = append(
		logWindow.logs,
		fmt.Sprintf(
			"%s %s: "+format+"\n",
			append([]any{time.Now().Format("15:04:05"), prefix}, args...)...,
		),
	)
}

func (logWindow *LogWindow) Info(format string, args ...any) {
	logWindow.Log("INFO", format, args...)
}

func (logWindow *LogWindow) Warn(format string, args ...any) {
	logWindow.Log("WARN", format, args...)
}

func (logWindow *LogWindow) Error(format string, args ...any) {
	logWindow.Log("ERROR", format, args...)
}

func (logWindow *LogWindow) RenderContent() string {
	cursor := 0
	text := ""
	for cursor < len(logWindow.logs) {
		text += wrap.String(
			wordwrap.String(logWindow.logs[cursor], logWindow.GetInnerWidth()),
			logWindow.GetInnerWidth(),
		)
		cursor += 1
	}
	return text
}

func (logWindow *LogWindow) Render() string {
	return logWindow.Window.Render(
		logWindow.Viewport.View(),
	)
}
