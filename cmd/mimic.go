package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/muesli/reflow/wordwrap"
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

var fileLogger *Logger

type mqttMessage struct {
	topic   string
	payload string
}

type mqttStatus struct {
	reason    string
	code      byte
	err       error
	connected bool
}

type urlParseError struct {
	uri string
	err error
}

type mqttServerConnection struct {
	connnection *autopaho.ConnectionManager
	err         error
}

type initialized byte

type publishMessage struct {
	payload string
	topic   string
	err     error
}

type subscribeMessage struct {
	topic string
	err   error
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
	logs      []string
	err       error
	logBuffer *bytes.Buffer
	viewport  viewport.Model
	window
}

func (logsWindow *logsWindow) Update() {
	logMessage := bytes.NewBuffer([]byte(""))
	totalSize := 0
	for {
		r, _, err := logsWindow.logBuffer.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logsWindow.err = err
			break
		}
		if r == '\r' || r == '\n' {
			break
		}
		totalSize += 1
		logMessage.WriteRune(r)
	}
	if totalSize == 0 {
		return
	}
	if len(logsWindow.logs) == 200 {
		logsWindow.logs = logsWindow.logs[1:]
	}

	logsWindow.logs = append(logsWindow.logs, logMessage.String()+"\n")
}

func (logsWindow *logsWindow) Render() string {
	cursor := 0
	text := ""
	for cursor < len(logsWindow.logs) {
		logWordWrapper := wordwrap.NewWriter(logsWindow.GetInnerWidth() - 1)
		logWordWrapper.Breakpoints = []rune{' '}
		logWordWrapper.Newline = []rune{'\n'}
		logWordWrapper.Write([]byte(logsWindow.logs[cursor]))
		logWordWrapper.Close()
		text += logWordWrapper.String()
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
	logger               Logger
	documentWindow       documentWindow
	logsWindow           logsWindow
	statusWindow         statusWindow
	mqttMessages         chan mqttMessage
	mqttConnectionStatus chan mqttStatus
}

func initMinicModel(ctx context.Context) *mimicModel {
	physicalWidth, physicalHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}

	statusSpinnger := spinner.New()
	statusSpinnger.Spinner = spinner.Dot
	statusSpinnger.Style = spinnerStyle

	logBuf := bytes.NewBuffer([]byte(""))

	model := mimicModel{
		err:                  nil,
		ctx:                  ctx,
		logger:               *NewLogger(logBuf, false, false),
		mqttConnectionStatus: make(chan mqttStatus, 5),
		mqttMessages:         make(chan mqttMessage, 5),
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
			logs:      make([]string, 0),
			logBuffer: logBuf,
			viewport:  viewport.New(0, 0),
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

	model.UpdateDimensions(physicalWidth, physicalHeight)

	return &model
}

func (model *mimicModel) UpdateDimensions(width int, height int) {
	model.documentWindow.SetWidth(width)
	model.documentWindow.SetHeight(height)

	innerWidth := model.documentWindow.GetInnerWidth()
	if innerWidth%3 > 0 {
		innerWidth -= innerWidth % 3
	}
	innerWidth /= 3

	innerHeight := model.documentWindow.GetInnerHeight()

	model.statusWindow.SetWidth(innerWidth)
	model.statusWindow.SetHeight(innerHeight)

	model.logsWindow.SetWidth(innerWidth * 2)
	model.logsWindow.SetHeight(innerHeight)

	model.logsWindow.viewport.Height = model.logsWindow.GetInnerHeight()
	model.logsWindow.viewport.Width = model.logsWindow.GetInnerWidth()
}

func (model mimicModel) Init() tea.Cmd {
	return tea.Batch(
		initConnection(model.ctx, model.mqttConnectionStatus, model.mqttMessages),
		waitForStatus(model.mqttConnectionStatus),
		model.statusWindow.spinner.Tick,
	)
}

func (model mimicModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model.logsWindow.Update()

	cmds := make([]tea.Cmd, 1)

	var viewportCmd tea.Cmd
	model.logsWindow.viewport, viewportCmd = model.logsWindow.viewport.Update(msg)
	cmds = append(cmds, viewportCmd)
	isAtBottom := model.logsWindow.viewport.AtBottom()
	model.logsWindow.viewport.SetContent(model.logsWindow.Render())
	if isAtBottom {
		model.logsWindow.viewport.GotoBottom()
	}

	switch msg := msg.(type) {
	case urlParseError:
		model.logger.Error("Failed to parse url: %s - Error: ", msg.uri, msg.err)
	case mqttServerConnection:
		if msg.err != nil {
			model.logger.Error("Failed to create connection to MQTT Broker: %v", msg.err)
			cmds = append(cmds, waitForStatus(model.mqttConnectionStatus))
			break
		}
		model.statusWindow.initialized = true
		model.serverConnection = msg.connnection
		cmds = append(cmds, model.statusWindow.spinner.Tick)
	case mqttStatus:
		model.statusWindow.isConnected = msg.connected
		if msg.err == nil && msg.connected {
			model.logger.Info("Connected to MQTT Broker")
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
			model.logger.Error("Failed to connect to MQTT Broker: %v", msg.err)
		} else if msg.code == 255 {
			model.logger.Error("MQTT Client error: %v", msg.err)
		} else {
			model.logger.Error("MQTT disconnect with reason: %s - code: %d", msg.reason, msg.code)
		}
		cmds = append(cmds, waitForStatus(model.mqttConnectionStatus))
	case mqttMessage:
		model.logger.Info("Received message from: %s", msg.topic)
		model.logger.Info("Payload is: %s", msg.payload)
		cmds = append(cmds, waitForMessage(model.mqttMessages))
	case publishMessage:
		if msg.err != nil {
			model.logger.Error("Failed to publish to: %s - Error: %v", msg.topic, msg.err)
		} else {
			model.logger.Info("Published to topic: %s - Payload: %s", msg.topic, msg.payload)
		}
	case subscribeMessage:
		if msg.err != nil {
			model.logger.Error("Failed to subscribe to: %s - Error: %v", msg.topic, msg.err)
		} else {
			model.logger.Info("Subscribed to topic: %s", msg.topic)
		}
	case tea.WindowSizeMsg:
		model.UpdateDimensions(msg.Width, msg.Height)
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
	return model, tea.Batch(cmds...)
}

func (model mimicModel) View() string {

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

func initConnection(ctx context.Context, mqttConnectionStatus chan mqttStatus, mqttMessages chan mqttMessage) tea.Cmd {
	return func() tea.Msg {
		serverUrl, err := url.Parse(mqttUri)
		if err != nil {
			return urlParseError{
				uri: mqttUri,
				err: err,
			}
		}

		clientConfig := autopaho.ClientConfig{
			ServerUrls:                    []*url.URL{serverUrl},
			ConnectUsername:               username,
			ConnectPassword:               []byte(password),
			KeepAlive:                     20,
			CleanStartOnInitialConnection: false,
			SessionExpiryInterval:         60,
			ConnectRetryDelay:             time.Second * 5,
			OnConnectionUp: func(connectionManager *autopaho.ConnectionManager, connectionAck *paho.Connack) {
				mqttConnectionStatus <- mqttStatus{connected: true, err: nil, reason: "", code: 0}
			},
			OnConnectError: func(err error) {
				mqttConnectionStatus <- mqttStatus{connected: false, err: err, reason: "", code: 254}
			},
			ClientConfig: paho.ClientConfig{
				ClientID: username,
				OnPublishReceived: []func(paho.PublishReceived) (bool, error){
					func(publishReveived paho.PublishReceived) (bool, error) {
						publish := publishReveived.Packet.Packet()
						mqttMessages <- mqttMessage{
							topic:   publish.Topic,
							payload: string(publish.Payload),
						}
						return true, nil
					},
				},
				OnClientError: func(err error) {
					mqttConnectionStatus <- mqttStatus{connected: false, err: err, reason: "", code: 255}
				},
				OnServerDisconnect: func(disconnect *paho.Disconnect) {
					mqttConnectionStatus <- mqttStatus{
						connected: false,
						err:       err,
						reason:    disconnect.Properties.ReasonString,
						code:      disconnect.ReasonCode,
					}
				},
			},
		}

		serverConnection, err := autopaho.NewConnection(ctx, clientConfig)
		if err != nil {
			return mqttServerConnection{
				connnection: serverConnection,
				err:         err,
			}
		}

		return mqttServerConnection{
			connnection: serverConnection,
			err:         nil,
		}
	}
}

func waitForMessage(mqttMessages chan mqttMessage) tea.Cmd {
	return func() tea.Msg {
		return <-mqttMessages
	}
}

func waitForStatus(mqttConnectionStatus chan mqttStatus) tea.Cmd {
	return func() tea.Msg {
		return <-mqttConnectionStatus
	}
}

func publishUnlock(serverConnection *autopaho.ConnectionManager, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		payload := "0001234567|" + time.Now().Format("2006-01-02 15:04:05")
		if _, err := serverConnection.Publish(ctx, &paho.Publish{
			QoS:     2,
			Topic:   unlockTopic,
			Payload: []byte(payload),
		}); err != nil {
			return publishMessage{unlockTopic, payload, err}
		}
		return publishMessage{unlockTopic, payload, nil}
	}
}

func subscribeToAccessList(serverConnection *autopaho.ConnectionManager, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		if serverConnection == nil {
			return subscribeMessage{
				accessListTopic,
				errors.New(
					fmt.Sprintf("Connection is nil! Cannot subscribe to: %s", accessListTopic),
				),
			}
		}
		if _, err := serverConnection.Subscribe(ctx, &paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{Topic: accessListTopic, QoS: 2},
			},
		}); err != nil {
			return subscribeMessage{accessListTopic, err}
		}

		return subscribeMessage{accessListTopic, nil}
	}
}

func subscribeToHealthCheck(serverConnection *autopaho.ConnectionManager, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		if serverConnection == nil {
			return subscribeMessage{
				accessListTopic,
				errors.New(
					fmt.Sprintf("Connection is nil! Cannot subscribe to: %s", healthCheckTopic),
				),
			}
		}
		if _, err := serverConnection.Subscribe(ctx, &paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{Topic: healthCheckTopic, QoS: 2},
			},
		}); err != nil {
			return subscribeMessage{healthCheckTopic, err}
		}

		return subscribeMessage{healthCheckTopic, nil}
	}
}

func healthCheckHandler(serverConnection *autopaho.ConnectionManager, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		if _, err := serverConnection.Publish(ctx, &paho.Publish{
			QoS:     2,
			Topic:   checkInTopic,
			Payload: []byte(username),
		}); err != nil {
			return publishMessage{checkInTopic, username, err}
		}
		return publishMessage{checkInTopic, username, nil}
	}
}

func runMimic(cmd *cobra.Command, args []string) {
	if _, err := tea.NewProgram(
		initMinicModel(cmd.Context()),
		tea.WithAltScreen(),
	).Run(); err != nil {
		logger.Error("Error running TUI: %v", err)
	}
}
