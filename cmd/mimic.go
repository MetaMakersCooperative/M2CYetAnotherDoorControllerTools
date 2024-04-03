package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var mimicCmd = &cobra.Command{
	Use:   "mimic",
	Short: "Minics a door controller for easier testing",
	Long:  "Minics what a door controller would publish for easier testing",
	Run:   runMimic,
}

var mqttUri string

func init() {
	porterCmd.AddCommand(mimicCmd)

	mimicCmd.Flags().StringVarP(&mqttUri, "mqtt_uri", "m", "", "Uri used to connect to the mqtt broker")
	mimicCmd.MarkFlagRequired("mqtt_uri")
}

type orientation struct {
	top    int
	left   int
	bottom int
	right  int
}

type border struct {
	top    bool
	left   bool
	bottom bool
	right  bool
}

type window struct {
	height  int
	width   int
	padding orientation
	margin  orientation
	border  border
}

func (w *window) SetHeight(height int) {
	w.height = height - w.margin.top - w.margin.bottom
	if w.border.top {
		w.height -= 1
	}
	if w.border.bottom {
		w.height -= 1
	}
}

func (w *window) SetWidth(width int) {
	w.width = width - w.margin.left - w.margin.right
	if w.border.left {
		w.width -= 1
	}
	if w.border.right {
		w.width -= 1
	}
}

func (w *window) GetInnerWidth() int {
	return w.width - w.padding.left - w.padding.right
}

func (w *window) GetInnerHeight() int {
	return w.height - w.padding.top - w.padding.bottom
}

type statusWindow struct {
	spinner     spinner.Model
	isConnected bool
	initialized bool
	err         error
	window
}

type logsWindow struct {
	logs     []string
	err      error
	viewport viewport.Model
	window
}

func (logsWindow *logsWindow) Log(prefix string, format string, args ...any) {
	logsWindow.logs = append(
		logsWindow.logs,
		fmt.Sprintf(
			"%s %s: "+format+"\n",
			append([]any{time.Now().Format("15:04:05"), prefix}, args...)...,
		),
	)
}

func (logsWindow *logsWindow) Info(format string, args ...any) {
	logsWindow.Log("INFO", format, args...)
}

func (logsWindow *logsWindow) Warn(format string, args ...any) {
	logsWindow.Log("WARN", format, args...)
}

func (logsWindow *logsWindow) Error(format string, args ...any) {
	logsWindow.Log("ERROR", format, args...)
}

func (logsWindow *logsWindow) Render() string {
	cursor := 0
	text := ""
	for cursor < len(logsWindow.logs) {
		text += wrap.String(
			wordwrap.String(logsWindow.logs[cursor], logsWindow.GetInnerWidth()),
			logsWindow.GetInnerWidth(),
		)
		cursor += 1
	}
	return text
}

type documentWindow struct {
	window
}

type mimicModel struct {
	err                  error
	serverConnection     *autopaho.ConnectionManager
	ctx                  context.Context
	documentWindow       documentWindow
	logsWindow           logsWindow
	statusWindow         statusWindow
	mqttMessages         chan mqttMessage
	mqttConnectionStatus chan mqttStatus
}

func initMinicModel(ctx context.Context) mimicModel {
	physicalWidth, physicalHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}

	statusSpinnger := spinner.New()
	statusSpinnger.Spinner = spinner.Dot
	statusSpinnger.Style = spinnerStyle

	model := mimicModel{
		err:                  nil,
		ctx:                  ctx,
		mqttConnectionStatus: make(chan mqttStatus),
		mqttMessages:         make(chan mqttMessage),
		serverConnection:     nil,
		documentWindow: documentWindow{
			window: window{
				width:   0,
				height:  0,
				margin:  orientation{0, 0, 0, 0},
				padding: orientation{1, 2, 1, 2},
				border:  border{false, false, false, false},
			},
		},
		logsWindow: logsWindow{
			logs:     make([]string, 0),
			viewport: viewport.New(0, 0),
			window: window{
				width:   0,
				height:  0,
				margin:  orientation{1, 1, 0, 0},
				padding: orientation{1, 2, 1, 2},
				border:  border{true, true, true, true},
			},
		},
		statusWindow: statusWindow{
			spinner:     statusSpinnger,
			isConnected: false,
			initialized: false,
			window: window{
				width:   0,
				height:  0,
				margin:  orientation{1, 1, 0, 0},
				padding: orientation{1, 2, 1, 2},
				border:  border{true, true, true, true},
			},
		},
	}

	return model.UpdateDimensions(physicalWidth, physicalHeight)
}

func (model mimicModel) UpdateDimensions(width int, height int) mimicModel {
	model.documentWindow.SetWidth(width)
	model.documentWindow.SetHeight(height)

	innerWidth := model.documentWindow.GetInnerWidth()
	innerHeight := model.documentWindow.GetInnerHeight()

	model.statusWindow.SetWidth(35)
	model.statusWindow.SetHeight(innerHeight)

	model.logsWindow.SetWidth(innerWidth - 35)
	model.logsWindow.SetHeight(innerHeight)

	model.logsWindow.viewport.Height = model.logsWindow.GetInnerHeight()
	model.logsWindow.viewport.Width = model.logsWindow.GetInnerWidth()

	return model
}

func (model mimicModel) Init() tea.Cmd {
	return tea.Batch(
		initConnection(model.ctx, model.mqttConnectionStatus, model.mqttMessages),
		waitForStatus(model.mqttConnectionStatus),
		model.statusWindow.spinner.Tick,
	)
}

func (model mimicModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 1)

	var viewportCmd tea.Cmd
	model.logsWindow.viewport, viewportCmd = model.logsWindow.viewport.Update(msg)
	cmds = append(cmds, viewportCmd)

	switch msg := msg.(type) {
	case urlParseError:
		model.logsWindow.Error("Failed to parse url: %s - Error: ", msg.uri, msg.err)
	case mqttServerConnection:
		if msg.err != nil {
			model.logsWindow.Error("Failed to create connection to MQTT Broker: %v", msg.err)
			cmds = append(cmds, waitForStatus(model.mqttConnectionStatus))
			break
		}
		model.statusWindow.initialized = true
		model.serverConnection = msg.connnection
		cmds = append(cmds, model.statusWindow.spinner.Tick)
	case mqttStatus:
		model.statusWindow.isConnected = msg.connected
		if msg.err == nil && msg.connected {
			model.logsWindow.Info("Connected to MQTT Broker")
			cmds = append(
				cmds,
				subscribeToAccessList(model.serverConnection, model.ctx),
				subscribeToHealthCheck(model.serverConnection, model.ctx),
				waitForStatus(model.mqttConnectionStatus),
				waitForMessage(model.mqttMessages),
			)
			break
		}
		model.statusWindow.err = msg.err
		if msg.code == 254 {
			model.logsWindow.Error("Failed to connect to MQTT Broker: %v", msg.err)
		} else if msg.code == 255 {
			model.logsWindow.Error("MQTT Client error: %v", msg.err)
		} else {
			model.logsWindow.Error("MQTT disconnect with reason: %s - code: %d", msg.reason, msg.code)
		}
		cmds = append(cmds, waitForStatus(model.mqttConnectionStatus))
	case mqttMessage:
		model.logsWindow.Info("Received message from: %s", msg.topic)
		model.logsWindow.Info("Payload is: %s", msg.payload)
		cmds = append(cmds, waitForMessage(model.mqttMessages))
	case publishMessage:
		if msg.err != nil {
			model.logsWindow.Error("Failed to publish to: %s - Error: %v", msg.topic, msg.err)
		} else {
			model.logsWindow.Info("Published to topic: %s - Payload: %s", msg.topic, msg.payload)
		}
	case subscribeMessage:
		if msg.err != nil {
			model.logsWindow.Error("Failed to subscribe to: %s - Error: %v", msg.topic, msg.err)
		} else {
			model.logsWindow.Info("Subscribed to topic: %s", msg.topic)
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
		model.statusWindow.spinner, spinnerCmd = model.statusWindow.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	// Needs to happen after tea.WindowSizeMsg is handled so that
	// the log lines are rendered correctly
	isAtBottom := model.logsWindow.viewport.AtBottom()
	model.logsWindow.viewport.SetContent(model.logsWindow.Render())
	if isAtBottom {
		model.logsWindow.viewport.GotoBottom()
	}

	return model, tea.Batch(cmds...)
}

func (model mimicModel) View() string {

	if model.documentWindow.height < 20 || model.documentWindow.width < 80 {
		return lipgloss.NewStyle().
			Foreground(highlight).
			Render(
				"Terminal window needs to be larger than 80x20 to show information",
			)
	}

	docStyle := lipgloss.NewStyle().
		PaddingTop(model.documentWindow.padding.top).
		PaddingLeft(model.documentWindow.padding.left).
		PaddingBottom(model.documentWindow.padding.bottom).
		PaddingRight(model.documentWindow.padding.right)

	docStyle.MaxHeight(model.documentWindow.height)
	docStyle.MaxWidth(model.documentWindow.width)

	doc := strings.Builder{}

	var status string
	if model.err != nil {
		status = fmt.Sprintf("%s Failed to start connection manager", model.statusWindow.spinner.View())
	} else if model.serverConnection == nil {
		status = fmt.Sprintf("%s Starting connected manager", model.statusWindow.spinner.View())
	} else if !model.statusWindow.isConnected {
		status = fmt.Sprintf("%s Attempting to connect", model.statusWindow.spinner.View())
	} else {
		status = fmt.Sprintf("%s Connected to MQTT Broker", model.statusWindow.spinner.View())
	}

	left := leftPanelStyle.
		Copy().
		Height(model.statusWindow.height).
		Width(model.statusWindow.width).
		MarginTop(model.statusWindow.margin.top).
		MarginLeft(model.statusWindow.margin.left).
		MarginBottom(model.statusWindow.margin.bottom).
		MarginRight(model.statusWindow.margin.right).
		PaddingTop(model.statusWindow.padding.top).
		PaddingLeft(model.statusWindow.padding.left).
		PaddingBottom(model.statusWindow.padding.bottom).
		PaddingRight(model.statusWindow.padding.right).
		Align(lipgloss.Left).
		Render(
			header.Render("Connection Status"),
			"\n",
			text.Render(status),
		)

	right := rightPanelStyle.
		Copy().
		Height(model.logsWindow.height).
		Width(model.logsWindow.width).
		MarginTop(model.logsWindow.margin.top).
		MarginLeft(model.logsWindow.margin.left).
		MarginBottom(model.logsWindow.margin.bottom).
		MarginRight(model.logsWindow.margin.right).
		PaddingTop(model.logsWindow.padding.top).
		PaddingLeft(model.logsWindow.padding.left).
		PaddingBottom(model.logsWindow.padding.bottom).
		PaddingRight(model.logsWindow.padding.right).
		Align(lipgloss.Left).
		Render(model.logsWindow.viewport.View())

	doc.WriteString(lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		right,
	))

	doc.WriteString("\n\n")

	return docStyle.Render(doc.String())
}

func runMimic(cmd *cobra.Command, args []string) {
	if _, err := tea.NewProgram(
		initMinicModel(cmd.Context()),
		tea.WithAltScreen(),
	).Run(); err != nil {
		logger.Error("Error running TUI: %v", err)
	}
}
