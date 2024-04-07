package cli_commands

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"metamakers.org/door-controller-mqtt/models"
)

var mimicCmd = &cobra.Command{
	Use:   "mimic",
	Short: "Minics a door controller for easier testing",
	Long:  "Minics what a door controller would publish for easier testing",
	Run:   runMimic,
}

func init() {
	porterCmd.AddCommand(mimicCmd)
}

func runMimic(cmd *cobra.Command, args []string) {
	if _, err := tea.NewProgram(
		models.InitMinicModel(cmd.Context(), mqttUri, username, password),
		tea.WithAltScreen(),
	).Run(); err != nil {
		log.Error().
			Str("error", err.Error()).
			Str("event", "TUI").
			Msg(fmt.Sprintf("Error running TUI: %v", err))
	}
}
