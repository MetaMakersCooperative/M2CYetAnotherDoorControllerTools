package messages

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"

	"metamakers.org/door-controller-mqtt/mqtt"
)

func InitConnection(
	ctx context.Context,
	mqttConnectionStatus chan MqttStatus,
	mqttMessages chan MqttMessage,
	mqttUri string,
	username string,
	password string,
) tea.Cmd {
	return func() tea.Msg {
		serverUrl, err := url.Parse(mqttUri)
		if err != nil {
			return UrlParseError{
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
				mqttConnectionStatus <- MqttStatus{Connected: true, Err: nil, Reason: "", Code: 0}
			},
			OnConnectError: func(err error) {
				mqttConnectionStatus <- MqttStatus{Connected: false, Err: err, Reason: "", Code: 254}
			},
			ClientConfig: paho.ClientConfig{
				ClientID: username,
				OnPublishReceived: []func(paho.PublishReceived) (bool, error){
					func(publishReveived paho.PublishReceived) (bool, error) {
						publish := publishReveived.Packet.Packet()
						mqttMessages <- MqttMessage{
							Topic:   publish.Topic,
							Payload: string(publish.Payload),
						}
						return true, nil
					},
				},
				OnClientError: func(err error) {
					mqttConnectionStatus <- MqttStatus{Connected: false, Err: err, Reason: "", Code: 255}
				},
				OnServerDisconnect: func(disconnect *paho.Disconnect) {
					mqttConnectionStatus <- MqttStatus{
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
			return MqttServerConnection{
				Connnection: serverConnection,
				Err:         err,
			}
		}

		return MqttServerConnection{
			Connnection: serverConnection,
			Err:         nil,
		}
	}
}

func WaitForMessage(mqttMessages chan MqttMessage) tea.Cmd {
	return func() tea.Msg {
		return <-mqttMessages
	}
}

func WaitForStatus(mqttConnectionStatus chan MqttStatus) tea.Cmd {
	return func() tea.Msg {
		return <-mqttConnectionStatus
	}
}

func PublishUnlock(serverConnection *autopaho.ConnectionManager, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		payload := "0001234567|" + time.Now().Format("2006-01-02 15:04:05")
		if _, err := serverConnection.Publish(ctx, &paho.Publish{
			QoS:     2,
			Topic:   mqtt.UnlockTopic,
			Payload: []byte(payload),
		}); err != nil {
			return PublishMessage{mqtt.UnlockTopic, payload, err}
		}
		return PublishMessage{mqtt.UnlockTopic, payload, nil}
	}
}

func SubscribeToAccessList(serverConnection *autopaho.ConnectionManager, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		if serverConnection == nil {
			return SubscribeMessage{
				mqtt.AccessListTopic,
				errors.New(
					fmt.Sprintf("Connection is nil! Cannot subscribe to: %s", mqtt.AccessListTopic),
				),
			}
		}
		if _, err := serverConnection.Subscribe(ctx, &paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{Topic: mqtt.AccessListTopic, QoS: 2},
			},
		}); err != nil {
			return SubscribeMessage{mqtt.AccessListTopic, err}
		}

		return SubscribeMessage{mqtt.AccessListTopic, nil}
	}
}

func SubscribeToHealthCheck(serverConnection *autopaho.ConnectionManager, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		if serverConnection == nil {
			return SubscribeMessage{
				mqtt.AccessListTopic,
				errors.New(
					fmt.Sprintf("Connection is nil! Cannot subscribe to: %s", mqtt.HealthCheckTopic),
				),
			}
		}
		if _, err := serverConnection.Subscribe(ctx, &paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{Topic: mqtt.HealthCheckTopic, QoS: 2},
			},
		}); err != nil {
			return SubscribeMessage{mqtt.HealthCheckTopic, err}
		}

		return SubscribeMessage{mqtt.HealthCheckTopic, nil}
	}
}

func HealthCheckHandler(serverConnection *autopaho.ConnectionManager, ctx context.Context, username string) tea.Cmd {
	return func() tea.Msg {
		if _, err := serverConnection.Publish(ctx, &paho.Publish{
			QoS:     2,
			Topic:   mqtt.CheckInTopic,
			Payload: []byte(username),
		}); err != nil {
			return PublishMessage{mqtt.CheckInTopic, username, err}
		}
		return PublishMessage{mqtt.CheckInTopic, username, nil}
	}
}
