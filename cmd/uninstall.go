package cmd

import (
	"fmt"
	"strings"

	"github.com/philokun/gvm/internal/version"
	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall [version]",
	Short: "Uninstall a specific Go version",
	Long:  `Remove a specific version of Go from your system.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		versionStr := args[0]

		// 标准化版本号格式
		if !strings.HasPrefix(versionStr, "go") {
			versionStr = "go" + versionStr
		}

		vm := version.New()

		fmt.Printf("Uninstalling Go %s...\n", versionStr)

		if err := vm.UninstallVersion(versionStr); err != nil {
			return fmt.Errorf("failed to uninstall version %s: %w", versionStr, err)
		}

		fmt.Printf("Successfully uninstalled Go %s\n", versionStr)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
