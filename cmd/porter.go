package cmd

import (
	"github.com/spf13/cobra"
)

var porterCmd = &cobra.Command{
	Use:   "porter",
	Short: "",
	Long:  ``,
}

var username string
var password string

func init() {
	rootCmd.AddCommand(porterCmd)

	porterCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "Username used to authenicate with the MQTT Broker")
	porterCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Password used to authenicate with the MQTT Broker")
	porterCmd.MarkFlagRequired("username")
	porterCmd.MarkFlagRequired("password")
}
