package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/philokun/gvm/internal/output"
	"github.com/philokun/gvm/internal/version"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed Go versions",
	Long:  `List all Go versions that are currently installed on your system.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vm := version.New()
		versions, err := vm.GetInstalledVersions()
		if err != nil {
			return fmt.Errorf("failed to get installed versions: %w", err)
		}

		if len(versions) == 0 {
			output.PrintWarning("No Go versions installed. Use 'gvm install <version>' to install one.")
			return nil
		}

		// 获取当前使用的版本
		current, _ := vm.GetCurrentVersion()

		output.PrintHeader("Installed Go versions")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "VERSION\t\tSTATUS")
		fmt.Fprintln(w, "-------\t\t------")

		for _, v := range versions {
			status := ""
			if v == current {
				status = output.ColorGreen + "current" + output.ColorReset
			}
			fmt.Fprintf(w, "%s\t\t%s\n", v, status)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
