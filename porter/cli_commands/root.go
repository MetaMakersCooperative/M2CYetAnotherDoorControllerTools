package cli_commands

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "porter",
	Short: "Collection of management tools for M2C door controllers",
	Long:  "Collection of management tools for M2C door controllers",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var username string
var password string
var mqttUri string

func init() {
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "Username used to authenicate with the MQTT Broker")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Password used to authenicate with the MQTT Broker")
	rootCmd.PersistentFlags().StringVarP(&mqttUri, "mqtt_uri", "m", "", "Uri used to connect to the mqtt broker")
}
