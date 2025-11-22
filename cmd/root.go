package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gvm",
	Short: "A Golang Version Manager",
	Long: `GVM (Go Version Manager) is a command-line tool that helps you manage multiple Go versions on your system.
	
Features:
  • Install and manage multiple Go versions
  • Switch between Go versions easily
  • List installed and available versions

Examples:
  gvm list                   # List installed versions (current version marked with *)
  gvm install go1.21.5       # Install Go 1.21.5
  gvm use go1.21.5           # Switch to Go 1.21.5
  gvm available              # List available versions

For more information, visit: https://github.com/philokun/gvm`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() // 显示帮助信息
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// 移除默认的toggle标志
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	// rootCmd.Flags().MarkHidden("toggle") // 隐藏这个标志，因为我们不需要它
}
