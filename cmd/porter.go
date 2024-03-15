package cmd

import (
	"github.com/spf13/cobra"
)

var porterCmd = &cobra.Command{
	Use:   "porter",
	Short: "",
	Long: ``,
}

func init() {
	rootCmd.AddCommand(porterCmd)
}
