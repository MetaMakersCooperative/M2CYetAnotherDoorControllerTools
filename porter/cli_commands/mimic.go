package cli_commands

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"metamakers.org/door-controller-mqtt/models"
)

var mimicCmd = &cobra.Command{
	Use:   "mimic",
	Short: "Minics a door controller for easier testing",
	Long:  "Minics what a door controller would publish for easier testing",
	Run:   runMimic,
}

var mqttUri string

func init() {
	porterCmd.AddCommand(mimicCmd)

	mimicCmd.Flags().StringVarP(&mqttUri, "mqtt_uri", "m", "", "Uri used to connect to the mqtt broker")
	mimicCmd.MarkFlagRequired("mqtt_uri")
}

func runMimic(cmd *cobra.Command, args []string) {
	if _, err := tea.NewProgram(
		models.InitMinicModel(cmd.Context(), mqttUri, username, password),
		tea.WithAltScreen(),
	).Run(); err != nil {
		logger.Error("Error running TUI: %v", err)
	}
}
