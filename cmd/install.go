package cmd

import (
    "fmt"
    "os"
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
	Args: cobra.ExactArgs(1),// 确保只接收一个版本参数
    RunE: func(cmd *cobra.Command, args []string) error {
        versionStr := args[0]// 获取版本参数

        vm := version.New()

    // 处理 latest 别名
    lower := strings.ToLower(strings.TrimSpace(versionStr))
    if lower == "latest" || lower == "go latest" || lower == "golatest" {
        v, err := vm.GetLatestStable()
        if err != nil {
            output.PrintError(fmt.Sprintf("Failed to resolve latest version: %s", err.Error()))
            return err
        }
        versionStr = v
    } else {
        // 标准化版本号格式，确保以 "go" 开头
        if !strings.HasPrefix(versionStr, "go") {
            versionStr = "go" + versionStr
        }
    }
    // 创建 VersionManager 实例
        // 打印安装进度
        output.PrintProgress(fmt.Sprintf("Installing Go %s...", versionStr))

    // 安装 Go 版本
    if err := vm.InstallVersion(versionStr); err != nil {
        output.PrintError(fmt.Sprintf("Failed to install version %s: %s", versionStr, err.Error()))
        return err
    }
		// 打印安装成功信息
		output.PrintSuccess(fmt.Sprintf("Successfully installed Go %s", versionStr))
		// 打印切换提示信息
		output.PrintInfo(fmt.Sprintf("Use 'gvm use %s' to switch to this version", versionStr))

        return nil
    },
}

func init() {
    rootCmd.AddCommand(installCmd)
    installCmd.Flags().String("mirror", "", "override download mirror base URL")
    installCmd.PreRun = func(cmd *cobra.Command, args []string) {
        m, _ := cmd.Flags().GetString("mirror")
        if strings.TrimSpace(m) != "" {
            os.Setenv("GVM_DL_MIRROR", strings.TrimRight(m, "/"))
        }
    }
}
