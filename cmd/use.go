package cmd

import (
    "fmt"
    "strings"

    "github.com/philokun/gvm/internal/version"
    "github.com/spf13/cobra"
)

// useCmd represents the use command
var useCmd = &cobra.Command{
	Use:   "use [version]",
	Short: "Switch to a specific Go version",
	Long: `Switch to using a specific version of Go.
	
This command updates your PATH to use the specified Go version.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		versionStr := args[0]

		// 标准化版本号格式
		if !strings.HasPrefix(versionStr, "go") {
			versionStr = "go" + versionStr
		}

		vm := version.New()

		fmt.Printf("Switching to Go %s...\n", versionStr)

		if err := vm.UseVersion(versionStr); err != nil {
			return fmt.Errorf("failed to switch to version %s: %w", versionStr, err)
		}

        fmt.Printf("Now using Go %s\n", versionStr)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
