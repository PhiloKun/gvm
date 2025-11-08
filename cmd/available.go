package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/philokun/gvm/internal/version"
	"github.com/spf13/cobra"
)

// availableCmd represents the available command
var availableCmd = &cobra.Command{
	Use:   "available",
	Short: "List available Go versions for installation",
	Long:  `Display all Go versions that are available for installation from the official Go website.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vm := version.New()

		versions, err := vm.GetAvailableVersions()
		if err != nil {
			return fmt.Errorf("failed to get available versions: %w", err)
		}

		fmt.Println("Available Go versions:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "VERSION\tSTABLE\tRELEASE DATE")
		fmt.Fprintln(w, "-------\t------\t------------")

		// 只显示最新的20个版本
		count := 0
		for _, v := range versions {
			if count >= 20 {
				break
			}
			stable := "No"
			if v.Stable {
				stable = "Yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", v.Version, stable, "N/A") // TODO: 添加发布日期
			count++
		}
		w.Flush()

		fmt.Println("\nUse 'gvm install <version>' to install a specific version.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(availableCmd)
}
