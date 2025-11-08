package cmd

import (
	"fmt"

	"github.com/philokun/gvm/internal/version"
	"github.com/spf13/cobra"
)

// currentCmd represents the current command
var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current Go version",
	Long:  `Display the Go version that is currently active.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vm := version.New()

		current, err := vm.GetCurrentVersion()
		if err != nil {
			return fmt.Errorf("failed to get current version: %w", err)
		}

		if current == "system" {
			fmt.Println("Using system Go installation")
		} else {
			fmt.Printf("Current Go version: %s\n", current)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
