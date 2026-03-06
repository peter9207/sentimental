package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sentimental",
	Short: "Sentimental CLI",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	monitorCmd.AddCommand(stocksCmd)
	monitorCmd.AddCommand(bitcoinCmd)
	rootCmd.AddCommand(monitorCmd)
}
