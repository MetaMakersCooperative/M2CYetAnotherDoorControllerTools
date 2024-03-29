package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blockloop/scan/v2"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
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
	logger := log.Default()
	logger.SetPrefix("INFO: ")
	logger.SetOutput(os.Stdout)

	log := func(format string, args ...any) {
		logger.Output(2, fmt.Sprintf(format, args...))
	}

	log("access_list called")
	// App will run until cancelled by user (e.g. ctrl-c)
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan bool, 1)
	queryErr := make(chan error, 1)
	cardList := make(chan string, 1)

	db, err := sql.Open("mysql", dbUri)
	if err != nil {
		log("failed to connect to mysql database")
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
			log("Adding card %d to list", code.CardNum)
			list += fmt.Sprintf("%d", code.CardNum)
			if idx < len(accessCodes)-1 {
				list += "\n"
			}
		}

		cardList <- list
	}()

	serverUrl, err := url.Parse("mqtt://localhost:1883")
	if err != nil {
		log("Url parse Error: %v\n", err)
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
			log("mqtt connection up")

			timeout := time.NewTimer(time.Second * 30)
			select {
			case <-timeout.C:
				log("Reached timeout. Aborting")
			case list := <-cardList:
				if _, err = connectionManager.Publish(ctx, &paho.Publish{
					QoS:     2,
					Topic:   accessListTopic,
					Payload: []byte(list),
				}); err != nil {
					if ctx.Err() == nil {
						log("Failed to publish: %v", err)
					} else {
						log("Publish cancelled by context: %v", err)
					}
				}
			case err := <-queryErr:
				log("Failed to retreive data from database: %v", err)
			}

			done <- true
		},
		OnConnectError: func(err error) {
			log("error whilst attempting connection: %s\n", err)
			done <- true
		},
		ClientConfig: paho.ClientConfig{
			ClientID: username,
			OnClientError: func(err error) {
				log("client error: %s\n", err)
				done <- true
			},
			OnServerDisconnect: func(disconnect *paho.Disconnect) {
				if disconnect.Properties != nil {
					log("server requested disconnect: %s\n", disconnect.Properties.ReasonString)
				} else {
					log("server requested disconnect; reason code: %d\n", disconnect.ReasonCode)
				}
			},
		},
	}

	serverConnection, err := autopaho.NewConnection(ctx, clientConfig)
	if err != nil {
		log("Server connection error: %v\n", err)
		if errors.Is(err, context.Canceled) {
			return
		} else {
			log("MQTT connection error: %v", err)
		}
	}
	if err = serverConnection.AwaitConnection(ctx); err != nil {
		log("Server await connection error: %v\n", err)
		if errors.Is(err, context.Canceled) {
			return
		} else {
			log("MQTT connection error: %v", err)
		}
	}

	select {
	case <-done:
		log("Finished")
		serverConnection.Disconnect(ctx)
	case <-ctx.Done():
		log("signal caught - exiting")
	}

	<-serverConnection.Done()
}
