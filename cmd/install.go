package cmd

import (
	"fmt"
	"strings"

	"github.com/philokun/gvm/internal/output"
	"github.com/philokun/gvm/internal/version"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [version]",
	Short: "Install a specific Go version",
	Long: `Install a specific version of Go. 
	
You can specify the version as:
- full version: go1.21.5
- short version: 1.21.5
- latest: installs the latest stable version`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		versionStr := args[0]

		// 标准化版本号格式
		if !strings.HasPrefix(versionStr, "go") {
			versionStr = "go" + versionStr
		}

		vm := version.New()

		output.PrintProgress(fmt.Sprintf("Installing Go %s...", versionStr))

		if err := vm.InstallVersion(versionStr); err != nil {
			output.PrintError(fmt.Sprintf("Failed to install version %s: %s", versionStr, err.Error()))
			return err
		}

		output.PrintSuccess(fmt.Sprintf("Successfully installed Go %s", versionStr))
		output.PrintInfo(fmt.Sprintf("Use 'gvm use %s' to switch to this version", versionStr))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
