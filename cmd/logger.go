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

// loggerCmd represents the logger command
var loggerCmd = &cobra.Command{
	Use:   "logger",
	Short: "",
	Long: ``,
	Run: run,
}

func init() {
	porterCmd.AddCommand(loggerCmd)

	loggerCmd.PersistentFlags().String("username", "", "Username used to authenicate with the MQTT Broker")
	loggerCmd.PersistentFlags().String("password", "", "Password used to authenicate with the MQTT Broker")
	loggerCmd.MarkFlagRequired("username")
	loggerCmd.MarkFlagRequired("password")
}

const clientID = "porter"
const unlockTopic = "door_controller/unlock"
const accessListTopic = "door_controller/access_list"

func run(cmd *cobra.Command, args []string) {
	fmt.Println("logger called")
	fmt.Printf("Args: %v", args)

	username := args[0]
	password := args[1]

	// App will run until cancelled by user (e.g. ctrl-c)
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverUrl, err := url.Parse("mqtt://localhost:1883")
	if err != nil {
		fmt.Printf("Url parse Error: %v\n", err)
		panic(err)
	}

	clientConfig := autopaho.ClientConfig{
		ServerUrls: []*url.URL{serverUrl},
		ConnectUsername: username,
		ConnectPassword: []byte(password),
		KeepAlive:  20,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval: 60,
		OnConnectionUp: func(connectionManager *autopaho.ConnectionManager, connectionAck *paho.Connack) {
			fmt.Println("mqtt connection up")

			if _, err := connectionManager.Subscribe(ctx, &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: accessListTopic, QoS: 2},
				},
			}); err != nil {
				fmt.Printf("failed to subscribe (%s). This is likely to mean no messages will be received.", err)
			}

			fmt.Println("mqtt subscription made")
		},
		OnConnectError: func(err error) {
			fmt.Printf("error whilst attempting connection: %s\n", err)
		},
		ClientConfig: paho.ClientConfig{
			ClientID: clientID,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(publishReveived paho.PublishReceived) (bool, error) {
					fmt.Printf(
						"received message on topic %s; body: %s (retain: %t)\n",
						publishReveived.Packet.Topic,
						publishReveived.Packet.Payload,
						publishReveived.Packet.Retain,
					)
					return true, nil
				},
			},
			OnClientError: func(err error) { 
				fmt.Printf("client error: %s\n", err)
			},
			OnServerDisconnect: func(disconnect *paho.Disconnect) {
				if disconnect.Properties != nil {
					fmt.Printf("server requested disconnect: %s\n", disconnect.Properties.ReasonString)
				} else {
					fmt.Printf("server requested disconnect; reason code: %d\n", disconnect.ReasonCode)
				}
			},
		},
	}

	serverConnection, err := autopaho.NewConnection(ctx, clientConfig)
	if err != nil {
		fmt.Printf("Server connection error: %v\n", err)
		if errors.Is(err, context.Canceled) {
			return
		} else {
			panic(err)
		}
	}
	if err = serverConnection.AwaitConnection(ctx); err != nil {
		fmt.Printf("Server await connection error: %v\n", err)
		if errors.Is(err, context.Canceled) {
			return
		} else {
			panic(err)
		}
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err = serverConnection.Publish(ctx, &paho.Publish{
				QoS:     1,
				Topic:   unlockTopic,
				Payload: []byte("1234567|" + time.Now().UTC().String()),
			}); err != nil {
				if ctx.Err() == nil {
					panic(err)
				}
			}
			continue
		case <-ctx.Done():
			fmt.Println("ctx.Done")
		}
		break
	}

	fmt.Println("signal caught - exiting")
	<-serverConnection.Done()
}
