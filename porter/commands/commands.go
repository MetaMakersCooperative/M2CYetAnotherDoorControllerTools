package commands

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"

	"metamakers.org/door-controller-mqtt/messages"
	"metamakers.org/door-controller-mqtt/mqtt"
)

func Init(mqttUri string, username string, password string) tea.Cmd {
	return func() tea.Msg {
		return messages.MqttCredentials{
			URI:      mqttUri,
			Username: username,
			Password: password,
		}
	}
}

func InitConnection(
	ctx context.Context,
	mqttConnectionStatus chan messages.MqttStatus,
	mqttMessages chan messages.MqttMessage,
	mqttUri string,
	username string,
	password string,
) tea.Cmd {
	return func() tea.Msg {
		serverUrl, err := url.Parse(mqttUri)
		if err != nil {
			return messages.UrlParseError{
				URI: mqttUri,
				Err: err,
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
				mqttConnectionStatus <- messages.MqttStatus{Connected: true, Err: nil, Reason: "", Code: 0}
			},
			OnConnectError: func(err error) {
				mqttConnectionStatus <- messages.MqttStatus{Connected: false, Err: err, Reason: "", Code: 254}
			},
			ClientConfig: paho.ClientConfig{
				ClientID: username,
				OnPublishReceived: []func(paho.PublishReceived) (bool, error){
					func(publishReveived paho.PublishReceived) (bool, error) {
						publish := publishReveived.Packet.Packet()
						mqttMessages <- messages.MqttMessage{
							Topic:   publish.Topic,
							Payload: string(publish.Payload),
						}
						return true, nil
					},
				},
				OnClientError: func(err error) {
					mqttConnectionStatus <- messages.MqttStatus{Connected: false, Err: err, Reason: "", Code: 255}
				},
				OnServerDisconnect: func(disconnect *paho.Disconnect) {
					mqttConnectionStatus <- messages.MqttStatus{
						Connected: false,
						Err:       err,
						Reason:    disconnect.Properties.ReasonString,
						Code:      disconnect.ReasonCode,
					}
				},
			},
		}

		serverConnection, err := autopaho.NewConnection(ctx, clientConfig)
		if err != nil {
			return messages.MqttServerConnection{
				Connnection: serverConnection,
				Err:         err,
			}
		}

		return messages.MqttServerConnection{
			Connnection: serverConnection,
			Err:         nil,
		}
	}
}

func WaitForMessage(mqttMessages chan messages.MqttMessage) tea.Cmd {
	return func() tea.Msg {
		return <-mqttMessages
	}
}

func WaitForStatus(mqttConnectionStatus chan messages.MqttStatus) tea.Cmd {
	return func() tea.Msg {
		return <-mqttConnectionStatus
	}
}

func PublishUnlock(serverConnection *autopaho.ConnectionManager, ctx context.Context, clientID string, code int) tea.Cmd {
	topic := mqtt.UnlockTopic + "/" + clientID
	return func() tea.Msg {
		payload := fmt.Sprintf("%010d|%s", code, time.Now().Format("2006-01-02 15:04:05"))
		if _, err := serverConnection.Publish(ctx, &paho.Publish{
			QoS:     1,
			Topic:   topic,
			Payload: []byte(payload),
		}); err != nil {
			return messages.PublishMessage{
				Topic:   topic,
				Payload: payload,
				Err:     err,
			}
		}
		return messages.PublishMessage{
			Topic:   topic,
			Payload: payload,
			Err:     nil,
		}
	}
}

func SubscribeToAccessList(serverConnection *autopaho.ConnectionManager, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		if serverConnection == nil {
			return messages.SubscribeMessage{
				Topic: mqtt.AccessListTopic,
				Err: errors.New(
					fmt.Sprintf("Connection is nil! Cannot subscribe to: %s", mqtt.AccessListTopic),
				),
			}
		}
		if _, err := serverConnection.Subscribe(ctx, &paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{Topic: mqtt.AccessListTopic, QoS: 1},
			},
		}); err != nil {
			return messages.SubscribeMessage{Topic: mqtt.AccessListTopic, Err: err}
		}

		return messages.SubscribeMessage{Topic: mqtt.AccessListTopic, Err: nil}
	}
}

func SubscribeToHealthCheck(serverConnection *autopaho.ConnectionManager, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		if serverConnection == nil {
			return messages.SubscribeMessage{
				Topic: mqtt.AccessListTopic,
				Err: errors.New(
					fmt.Sprintf("Connection is nil! Cannot subscribe to: %s", mqtt.HealthCheckTopic),
				),
			}
		}
		if _, err := serverConnection.Subscribe(ctx, &paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{Topic: mqtt.HealthCheckTopic, QoS: 1},
			},
		}); err != nil {
			return messages.SubscribeMessage{Topic: mqtt.HealthCheckTopic, Err: err}
		}

		return messages.SubscribeMessage{Topic: mqtt.HealthCheckTopic, Err: nil}
	}
}

func publishMessage(serverConnection *autopaho.ConnectionManager, ctx context.Context, topic string, payload string) tea.Cmd {
	return func() tea.Msg {
		if _, err := serverConnection.Publish(ctx, &paho.Publish{
			QoS:     1,
			Topic:   topic,
			Payload: []byte(payload),
		}); err != nil {
			return messages.PublishMessage{Topic: topic, Payload: payload, Err: err}
		}
		return messages.PublishMessage{Topic: topic, Payload: payload, Err: nil}
	}
}

func HealthCheckHandler(serverConnection *autopaho.ConnectionManager, ctx context.Context, clientID string) tea.Cmd {
	topic := mqtt.CheckInTopic + "/" + clientID
	return publishMessage(serverConnection, ctx, topic, clientID)
}

func FailHealthCheckHandler(clientID string) tea.Cmd {
	topic := mqtt.CheckInTopic + "/" + clientID
	return func() tea.Msg {
		return messages.PublishMessage{Topic: topic, Payload: clientID, Err: errors.New("Set to fail health checks")}
	}
}

func AccessListHandler(serverConnection *autopaho.ConnectionManager, ctx context.Context, clientID string) tea.Cmd {
	logInfoTopic := mqtt.LogInfoTopic + "/" + clientID
	return tea.Batch(
		publishMessage(serverConnection, ctx, logInfoTopic, "Completed rebuilding cards.txt"),
		publishMessage(serverConnection, ctx, logInfoTopic, "Rebuilding cards.txt"),
	)
}

func FailAccessListHandler(serverConnection *autopaho.ConnectionManager, ctx context.Context, clientID string) tea.Cmd {
	logInfoTopic := mqtt.LogInfoTopic + "/" + clientID
	logFatalTopic := mqtt.LogFatalTopic + "/" + clientID
	return tea.Batch(
		publishMessage(serverConnection, ctx, logFatalTopic, "Failed to read cards.txt"),
		publishMessage(serverConnection, ctx, logInfoTopic, "Rebuilding cards.txt"),
	)
}
