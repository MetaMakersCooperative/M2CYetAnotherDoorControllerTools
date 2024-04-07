package cli_commands

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var diaryCmd = &cobra.Command{
	Use:   "diary",
	Short: "Collects & aggregates messages from door controllers",
	Long:  "Collects & aggregates messages from door controllers",
	Run:   runDiaryCmd,
}

func init() {
	porterCmd.AddCommand(diaryCmd)
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

func runDiaryCmd(cmd *cobra.Command, _ []string) {
	// App will run until cancelled by user (e.g. ctrl-c)
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGUSR1, syscall.SIGTERM)
	defer stop()

	// Reload on SIGHUP
	reloadCtx, cancel := signal.NotifyContext(ctx, syscall.SIGHUP)
	defer cancel()

	serverUrl, err := url.Parse(mqttUri)
	if err != nil {
		log.Error().
			Str("error", err.Error()).
			Str("event", "URLParse").
			Msg(fmt.Sprintf("Url parse Error: %v\n", err))
		syscall.Exit(2)
		return
	}

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
		},
		OnConnectError: func(err error) {
			log.Error().
				Str("error", err.Error()).
				Str("event", "OnConnectError").
				Msg(fmt.Sprintf("MQTT Connection error: %v", err))
		},
		ClientConfig: paho.ClientConfig{
			ClientID: username,
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
			return
		}
	}
	if err = serverConnection.AwaitConnection(ctx); err != nil {
		log.Warn().
			Str("error", err.Error()).
			Str("event", "AwaitConnection").
			Msg(fmt.Sprintf("Server await connection error: %v", err))
		if errors.Is(err, context.Canceled) {
			return
		}
	}

	if err := notifyReady(); err != nil {
		if !errors.Is(err, NotifySocketNotFound) {
			stop()
			cancel()
		}
	}

	select {
	case <-reloadCtx.Done():
		serverConnection.Disconnect(ctx)

		if err := notifyReloading(); err != nil {
			if !errors.Is(err, NotifySocketNotFound) {
				stop()
				cancel()
			}
		}

		if err = serverConnection.AwaitConnection(ctx); err != nil {
			log.Warn().
				Str("error", err.Error()).
				Str("event", "AwaitConnection").
				Msg(fmt.Sprintf("Server await connection error: %v", err))
			if errors.Is(err, context.Canceled) {
				return
			}
		}

		if err := notifyReady(); err != nil {
			if !errors.Is(err, NotifySocketNotFound) {
				stop()
				cancel()
			}
		}
	case <-ctx.Done():
		log.Info().
			Str("event", "stopping").
			Msg("Termination signal received")
		syscall.Exit(0)
	}

	<-serverConnection.Done()
}
