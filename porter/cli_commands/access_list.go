package cli_commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blockloop/scan/v2"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"metamakers.org/door-controller-mqtt/mqtt"
)

var accessListCmd = &cobra.Command{
	Use:   "access_list",
	Short: "Publishes the access card list to each connected door controller",
	Long:  "Publishes the access card list to each connected door controller",
	Run:   runAccessList,
}

var dbUri string

func init() {
	porterCmd.AddCommand(accessListCmd)

	accessListCmd.Flags().StringVarP(&dbUri, "db_uri", "d", "", "Uri used to connect to the database")
	accessListCmd.MarkFlagRequired("db_uri")
}

type AccessControl struct {
	ID      int    `db:"id"`
	CardNum int    `db:"rfid_card_num"`
	CardVal int    `db:"rfid_card_val"`
	Status  string `db:"status"`
	Comment string `db:"comment"`
}

func runAccessList(cmd *cobra.Command, args []string) {
	// App will run until cancelled by user (e.g. ctrl-c)
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan bool, 1)
	queryErr := make(chan error, 1)
	cardList := make(chan string, 1)

	db, err := sql.Open("mysql", dbUri)
	if err != nil {
		log.Error().
			Str("error", err.Error()).
			Str("event", "DatabaseConnection").
			Msg(fmt.Sprintf("Failed to connect to mysql database: %v", err))
		return
	}
	defer db.Close()

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	go func() {
		query := "select * from accesscontrol where status = ?;"
		rows, err := db.QueryContext(ctx, query, "active")
		if err != nil {
			queryErr <- err
			return
		}

		accessCodes := make([]AccessControl, 0)
		err = scan.Rows(&accessCodes, rows)
		if err != nil {
			queryErr <- err
			return
		}

		var list string
		for idx, code := range accessCodes {
			log.Info().
				Str("event", "AddingCard").
				Int("card_number", code.CardNum).
				Msg(fmt.Sprintf("Adding card %d to list", code.CardNum))
			list += fmt.Sprintf("%d", code.CardNum)
			if idx < len(accessCodes)-1 {
				list += "\n"
			}
		}

		cardList <- list
	}()

	serverUrl, err := url.Parse(mqttUri)
	if err != nil {
		log.Error().
			Str("error", err.Error()).
			Str("event", "URLParse").
			Msg(fmt.Sprintf("Url parse Error: %v\n", err))
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

			timeout := time.NewTimer(time.Second * 30)
			select {
			case <-timeout.C:
				log.Warn().
					Str("event", "PublishTimeout").
					Msg("Failed to receive card list")
			case list := <-cardList:
				if _, err = connectionManager.Publish(ctx, &paho.Publish{
					QoS:     2,
					Topic:   mqtt.AccessListTopic,
					Payload: []byte(list),
				}); err != nil {
					if ctx.Err() == nil {
						log.Error().
							Str("error", err.Error()).
							Str("event", "AccessListPublish").
							Msg(fmt.Sprintf("Failed to publish: %v", err))
					} else {
						log.Warn().
							Str("error", err.Error()).
							Str("event", "AccessListPublish").
							Msg(fmt.Sprintf("Published cancelled by context: %v", err))
					}
				}
			case err := <-queryErr:
				log.Error().
					Str("error", err.Error()).
					Str("event", "DatabaseQuery").
					Msg(fmt.Sprintf("Failed to query database: %v", err))
			}

			done <- true
		},
		OnConnectError: func(err error) {
			log.Error().
				Str("error", err.Error()).
				Str("event", "OnConnectError").
				Msg(fmt.Sprintf("MQTT Connection error: %v", err))
			done <- true
		},
		ClientConfig: paho.ClientConfig{
			ClientID: username,
			OnClientError: func(err error) {
				log.Error().
					Str("error", err.Error()).
					Str("event", "OnClientError").
					Msg(fmt.Sprintf("MQTT Client error: %v", err))
				done <- true
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

	select {
	case <-done:
		log.Info().
			Str("event", "done").
			Msg("Finished publishing access list")
		serverConnection.Disconnect(ctx)
	case <-ctx.Done():
		log.Info().
			Str("event", "stopping").
			Msg("Termination signal received")
		syscall.Exit(0)
	}

	<-serverConnection.Done()
}
