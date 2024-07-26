package cli_commands

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"metamakers.org/door-controller-mqtt/mqtt"
)

var diaryCmd = &cobra.Command{
	Use:   "diary",
	Short: "Collects & aggregates messages from door controllers",
	Long:  "Collects & aggregates messages from door controllers",
	Run:   runDiaryCmd,
}

func init() {
	rootCmd.AddCommand(diaryCmd)
}

var (
	NotifySocketNotFound = errors.New("Notify socket was not found!")
)

func handleNotifyError(state bool, err error, notification string) error {
	if !state && err != nil {
		log.Error().
			Str("error", err.Error()).
			Str("event", "SystemdNotify").
			Str("notification", notification).
			Msg(fmt.Sprintf("Systemd notify supported but failed: %v", err))
		return err
	}
	if !state && err == nil {
		log.Warn().
			Str("event", "SystemdNotify").
			Str("notification", notification).
			Msg("Systemd notify not supported")
		return NotifySocketNotFound
	}
	if state && err == nil {
		log.Info().
			Str("event", "SystemdNotify").
			Str("notification", notification).
			Msg("Systemd notify is supported and ready message has been sent")
	}
	return err
}

func notifyReady() error {
	log.Info().
		Str("event", "SystemdNotify").
		Str("notification", "ready").
		Msg("Sending ready notificaiton")
	state, err := daemon.SdNotify(false, daemon.SdNotifyReady)
	return handleNotifyError(state, err, "ready")
}

func notifyReloading() error {
	log.Info().
		Str("event", "SystemdNotify").
		Str("notification", "reloading").
		Msg("Sending reloading notificaiton")
	state, err := daemon.SdNotify(false, daemon.SdNotifyReloading)
	return handleNotifyError(state, err, "reloading")
}

var unhealthyDuration = time.Minute * 5
var sendHealthCheckDuration = time.Minute * 2
var checkHealthDuration = time.Second * 15

type ClientState int

const (
	Healthy = iota
	Unhealthy
)

var clientStateNames = map[ClientState]string{
	Healthy:   "healthy",
	Unhealthy: "unhealthy",
}

func (clientState ClientState) String() string {
	return clientStateNames[clientState]
}

type ClientHealth struct {
	LastSeen       time.Time
	State          ClientState
	UnhealthyAfter time.Time
}

func NewClientHealth() ClientHealth {
	now := time.Now()
	return ClientHealth{
		now,
		Healthy,
		now.Add(unhealthyDuration),
	}
}

func (clientHealth ClientHealth) Transitioned() (ClientHealth, bool) {
	now := time.Now()
	switch clientHealth.State {
	case Healthy:
		if clientHealth.UnhealthyAfter.Before(now) {
			clientHealth.State = Unhealthy
			return clientHealth, true
		}
	case Unhealthy:
		if clientHealth.UnhealthyAfter.After(now) {
			clientHealth.State = Healthy
			return clientHealth, true
		}
	}
	return clientHealth, false
}

func (clientHealth ClientHealth) BumpLastSeen() ClientHealth {
	clientHealth.LastSeen = time.Now()
	clientHealth.UnhealthyAfter = clientHealth.LastSeen.Add(unhealthyDuration)
	return clientHealth
}

func runDiaryCmd(cmd *cobra.Command, _ []string) {
	// App will run until cancelled by user (e.g. ctrl-c)
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGUSR1, syscall.SIGTERM)
	defer stop()

	// Reload on SIGHUP
	reloadCtx, cancel := signal.NotifyContext(ctx, syscall.SIGHUP)
	defer cancel()

	if result, found := os.LookupEnv("MQTT_URI"); found {
		mqttUri = result
	}

	if result, found := os.LookupEnv("MQTT_USER"); found {
		username = result
	}

	if result, found := os.LookupEnv("MQTT_PASSWORD"); found {
		password = result
	}

	serverUrl, err := url.Parse(mqttUri)
	if err != nil {
		log.Error().
			Str("error", err.Error()).
			Str("event", "URLParse").
			Msg(fmt.Sprintf("Url parse Error: %v\n", err))
		syscall.Exit(2)
		return
	}

	lastSeen := make(map[string]ClientHealth, 0)

	router := paho.NewStandardRouter()
	router.RegisterHandler(mqtt.RootLevel+"/#", func(publish *paho.Publish) {

		topicChunks := strings.Split(publish.Topic, "/")

		if len(topicChunks) < 3 {
			log.Error().
				Str("event", "TopicParser").
				Uint16("packet_id", publish.PacketID).
				Bool("duplicate", publish.Duplicate()).
				Bool("retain", publish.Retain).
				Str("qos", string(publish.QoS)).
				Str("topic", publish.Topic).
				Str("content_type", publish.Properties.ContentType).
				Str("payload", string(publish.Payload)).
				Msg("Unable to parse topic! Received less than 3 chunks")
			return
		}

		var logLevel *zerolog.Event
		switch topicChunks[1] {
		case mqtt.LogFatalLevel:
			logLevel = log.Error()
		case mqtt.LogWarnLevel, mqtt.DeniedAccessLevel:
			logLevel = log.Warn()
		default:
			logLevel = log.Info()
		}

		clientID := topicChunks[len(topicChunks)-1]
		// Any message received from a client should bump
		// its last seem value
		if client, found := lastSeen[clientID]; found {
			lastSeen[clientID] = client.BumpLastSeen()
		} else {
			lastSeen[clientID] = NewClientHealth()
		}

		logLevel.
			Str("event", "PublishHandler").
			Uint16("packet_id", publish.PacketID).
			Bool("duplicate", publish.Duplicate()).
			Bool("retain", publish.Retain).
			Str("qos", string(publish.QoS)).
			Str("clientID", clientID).
			Str("topic", publish.Topic).
			Str("content_type", publish.Properties.ContentType).
			Str("payload", string(publish.Payload)).
			Msg("Publish payload was handled")
	})

	clientConfig := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverUrl},
		ConnectUsername:               username,
		ConnectPassword:               []byte(password),
		KeepAlive:                     20,
		CleanStartOnInitialConnection: true,
		SessionExpiryInterval:         60,
		OnConnectionUp: func(connectionManager *autopaho.ConnectionManager, connectionAck *paho.Connack) {
			log.Info().
				Str("event", "OnConnectionUp").
				Str("response", connectionAck.Properties.ResponseInfo).
				Msg("Connected to MQTT broker")

			if _, err := connectionManager.Subscribe(ctx, &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: mqtt.LogInfoTopic + "/#", QoS: 1},
					{Topic: mqtt.LogWarnTopic + "/#", QoS: 1},
					{Topic: mqtt.LogFatalTopic + "/#", QoS: 1},
					{Topic: mqtt.LockTopic + "/#", QoS: 1},
					{Topic: mqtt.UnlockTopic + "/#", QoS: 1},
					{Topic: mqtt.DeniedAccessTopic + "/#", QoS: 1},
					{Topic: mqtt.CheckInTopic + "/#", QoS: 1},
				},
			}); err != nil {
				log.Error().
					Str("error", err.Error()).
					Str("event", "MQTTSubscribe").
					Msg(fmt.Sprintf("MQTT failed to subscribe: %v", err))
			}
		},
		OnConnectError: func(err error) {
			log.Error().
				Str("error", err.Error()).
				Str("event", "OnConnectError").
				Msg(fmt.Sprintf("MQTT Connection error: %v", err))
		},
		ClientConfig: paho.ClientConfig{
			ClientID: username,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(publish paho.PublishReceived) (bool, error) {
					router.Route(publish.Packet.Packet())
					return true, nil
				},
			},
			OnClientError: func(err error) {
				log.Error().
					Str("error", err.Error()).
					Str("event", "OnClientError").
					Msg(fmt.Sprintf("MQTT Client error: %v", err))
			},
			OnServerDisconnect: func(disconnect *paho.Disconnect) {
				if disconnect.Properties != nil {
					log.Warn().
						Str("error", err.Error()).
						Str("reason", disconnect.Properties.ReasonString).
						Str("event", "OnServerDisconnect").
						Msg(fmt.Sprintf("MQTT client disconnect: %v", err))
				} else {
					log.Warn().
						Str("error", err.Error()).
						Str("event", "OnServerDisconnect").
						Msg(fmt.Sprintf("MQTT client disconnect: %v", err))
				}
			},
		},
	}

	serverConnection, err := autopaho.NewConnection(ctx, clientConfig)
	if err != nil {
		log.Warn().
			Str("error", err.Error()).
			Str("event", "NewConnection").
			Msg(fmt.Sprintf("New connection start interrupted: %v", err))
		if errors.Is(err, context.Canceled) {
			syscall.Exit(1)
			return
		}
	}

	if err := notifyReady(); err != nil {
		if !errors.Is(err, NotifySocketNotFound) {
			stop()
			cancel()
		}
	}

	sendHealthCheckTicker := time.NewTicker(sendHealthCheckDuration)
	checkHealthTicker := time.NewTicker(checkHealthDuration)
	for {
		select {
		case <-checkHealthTicker.C:
			for key, clientHealth := range lastSeen {
				newClientHealth, transitioned := clientHealth.Transitioned()
				if transitioned {
					lastSeen[key] = newClientHealth
					switch newClientHealth.State {
					case Unhealthy:
						log.Error().
							Str("event", "Unhealthy").
							Str("client_id", key).
							Str("from", clientHealth.State.String()).
							Str("to", newClientHealth.State.String()).
							Str("last_seen", newClientHealth.LastSeen.String()).
							Str("unhealthy_after", newClientHealth.UnhealthyAfter.String()).
							Str("unhealthy_at", time.Now().String()).
							Msg(fmt.Sprintf("Client %s is now unhealthy", key))
					case Healthy:
						log.Info().
							Str("event", "Healthy").
							Str("client_id", key).
							Str("from", clientHealth.State.String()).
							Str("to", newClientHealth.State.String()).
							Str("last_seen", newClientHealth.LastSeen.String()).
							Str("unhealthy_after", newClientHealth.UnhealthyAfter.String()).
							Msg(fmt.Sprintf("Client %s is now healthy", key))
					}
				}
			}

		case <-sendHealthCheckTicker.C:
			log.Info().
				Str("event", "HealthCheckTicker").
				Msg("Sending health check")

			if _, err = serverConnection.Publish(ctx, &paho.Publish{
				QoS:     1,
				Topic:   mqtt.HealthCheckTopic,
				Payload: []byte(username),
			}); err != nil {
				if ctx.Err() == nil {
					log.Error().
						Str("error", err.Error()).
						Str("event", "MQTTPublish").
						Str("topic", mqtt.HealthCheckTopic).
						Msg(fmt.Sprintf("Failed to publish: %v", err))
				} else {
					log.Warn().
						Str("error", err.Error()).
						Str("event", "MQTTPublish").
						Str("topic", mqtt.HealthCheckTopic).
						Msg(fmt.Sprintf("Published cancelled by context: %v", err))
				}

				continue
			}

			log.Info().
				Str("event", "HealthCheckTicker").
				Msg("Health checks sent")

		case <-reloadCtx.Done():
			serverConnection.Disconnect(ctx)

			if err := notifyReloading(); err != nil {
				if !errors.Is(err, NotifySocketNotFound) {
					stop()
					cancel()
					break
				}
			}

			if err = serverConnection.AwaitConnection(ctx); err != nil {
				log.Warn().
					Str("error", err.Error()).
					Str("event", "AwaitConnection").
					Msg(fmt.Sprintf("Server await connection error: %v", err))
				syscall.Exit(3)
				return
			}

			if err := notifyReady(); err != nil {
				if !errors.Is(err, NotifySocketNotFound) {
					stop()
					cancel()
				}
			}

		case <-ctx.Done():
			log.Info().
				Str("event", "ContextCancelled").
				Msg("Termination signal received")
			syscall.Exit(0)

		case <-serverConnection.Done():
			log.Info().
				Str("event", "ConnectionClosed").
				Msg("MQTT server connection closed")
			break
		}
	}
}
