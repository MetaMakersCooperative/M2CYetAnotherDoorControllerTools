package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

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
