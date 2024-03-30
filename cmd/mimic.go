package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/spf13/cobra"
)

var mimicCmd = &cobra.Command{
	Use:   "mimic",
	Short: "Minics a door controller for easier testing",
	Long:  "Minics what a door controller would publish for easier testing",
	Run:   runMimic,
}

var seconds int

func init() {
	porterCmd.AddCommand(mimicCmd)
	mimicCmd.Flags().IntVarP(&seconds, "seconds", "s", 10, "Seconds to wait before publishing new unlock")
}

func healthCheckHandler(ctx context.Context, serverConnection *autopaho.ConnectionManager, mqttError *chan error, username string) paho.MessageHandler {
	return func(publish *paho.Publish) {
		logger.Info("Health Check handler triggered")

		if _, err := serverConnection.Publish(ctx, &paho.Publish{
			QoS:     2,
			Topic:   checkInTopic,
			Payload: []byte(username),
		}); err != nil {
			logger.Error("Check in publish error: %v", err)
			*mqttError <- err
		}
	}
}

func runMimic(cmd *cobra.Command, args []string) {
	fmt.Println("Mimic called")

	router := paho.NewStandardRouter()

	// App will run until cancelled by user (e.g. ctrl-c)
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mqttError := make(chan error, 1)

	serverUrl, err := url.Parse("mqtt://localhost:1883")
	if err != nil {
		logger.Error("Url parse Error: %v\n", err)
		mqttError <- err
	}

	clientConfig := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverUrl},
		ConnectUsername:               username,
		ConnectPassword:               []byte(password),
		KeepAlive:                     20,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		OnConnectionUp: func(connectionManager *autopaho.ConnectionManager, connectionAck *paho.Connack) {
			logger.Info("mqtt connection up")

			router.RegisterHandler(healthCheckTopic, healthCheckHandler(ctx, connectionManager, &mqttError, username))

			if _, err := connectionManager.Subscribe(ctx, &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: accessListTopic, QoS: 2},
				},
			}); err != nil {
				logger.Error("failed to subscribe (%s). This is likely to mean no messages will be received.", err)
				mqttError <- err
				return
			}

			if _, err := connectionManager.Subscribe(ctx, &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: healthCheckTopic, QoS: 2},
				},
			}); err != nil {
				logger.Error("failed to subscribe (%s). This is likely to mean no messages will be received.", err)
				mqttError <- err
				return
			}

			logger.Info("mqtt subscriptions started")
		},
		OnConnectError: func(err error) {
			logger.Info("error whilst attempting connection: %s\n", err)
			mqttError <- err
		},
		ClientConfig: paho.ClientConfig{
			ClientID: username,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(publishReveived paho.PublishReceived) (bool, error) {
					router.Route(publishReveived.Packet.Packet())
					return true, nil
				},
			},
			OnClientError: func(err error) {
				logger.Error("client error: %s\n", err)
				mqttError <- err
			},
			OnServerDisconnect: func(disconnect *paho.Disconnect) {
				router.UnregisterHandler(healthCheckTopic)

				if disconnect.Properties != nil {
					logger.Warn("server requested disconnect: %s\n", disconnect.Properties.ReasonString)
				} else {
					logger.Warn("server requested disconnect; reason code: %d\n", disconnect.ReasonCode)
				}
			},
		},
	}

	serverConnection, err := autopaho.NewConnection(ctx, clientConfig)
	if err != nil {
		logger.Error("Server connection error: %v\n", err)
		if errors.Is(err, context.Canceled) {
			return
		} else {
			mqttError <- err
		}
	}
	if err = serverConnection.AwaitConnection(ctx); err != nil {
		logger.Error("Server await connection error: %v\n", err)
		if errors.Is(err, context.Canceled) {
			return
		} else {
			mqttError <- err
		}
	}

	ticker := time.NewTicker(time.Second * time.Duration(seconds))
	defer ticker.Stop()
	for {
		select {
		case err := <-mqttError:
			logger.Error("Exiting with error: %v", err)
			serverConnection.Disconnect(ctx)
		case <-ticker.C:
			if _, err = serverConnection.Publish(ctx, &paho.Publish{
				QoS:     2,
				Topic:   unlockTopic,
				Payload: []byte("0001234567|" + time.Now().Format("2006-01-02 15:04:05")),
			}); err != nil {
				logger.Error("Unlock publish error: %v", err)
				mqttError <- err
			}
			continue
		case <-ctx.Done():
			logger.Info("Context Canceled")
		}
		break
	}

	<-serverConnection.Done()
}
