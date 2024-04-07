package cli_commands

import (
	"github.com/spf13/cobra"
)

var porterCmd = &cobra.Command{
	Use:   "porter",
	Short: "Collection of management tools for M2C door controllers",
	Long:  "Collection of management tools for M2C door controllers",
}

var username string
var password string
var mqttUri string

func init() {
	rootCmd.AddCommand(porterCmd)

	porterCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "Username used to authenicate with the MQTT Broker")
	porterCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Password used to authenicate with the MQTT Broker")
	porterCmd.PersistentFlags().StringVarP(&mqttUri, "mqtt_uri", "m", "", "Uri used to connect to the mqtt broker")
	porterCmd.MarkFlagRequired("mqtt_uri")
	porterCmd.MarkFlagRequired("username")
	porterCmd.MarkFlagRequired("password")
}
