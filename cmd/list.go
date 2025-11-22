package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/philokun/gvm/internal/output"
	"github.com/philokun/gvm/internal/version"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed Go versions",
	Long:    `List all Go versions that are currently installed on your system.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vm := version.New()
		versions, err := vm.GetInstalledVersions()
		if err != nil {
			return fmt.Errorf("failed to get installed versions: %w", err)
		}

		current, _ := vm.GetCurrentVersion()
		sysVer := detectSystemGo(vm)

		// 收集所有版本（系统版本 + gvm 安装的版本）
		allVersions := make([]versionInfo, 0)

		// 添加系统版本
		if sysVer != "" {
			isCurrent := current == "system"
			allVersions = append(allVersions, versionInfo{
				version: sysVer,
				source:  "system",
				current: isCurrent,
			})
		}

		// 添加 gvm 安装的版本
		for _, v := range versions {
			isCurrent := v == current
			allVersions = append(allVersions, versionInfo{
				version: v,
				source:  "gvm",
				current: isCurrent,
			})
		}

		// 如果没有版本，显示提示
		if len(allVersions) == 0 {
			output.PrintWarning("No Go found. Use 'gvm install <version>' to install one.")
			return nil
		}

		// 排序：当前版本在前，其他版本按版本号降序
		sortVersions(allVersions)

		// 仿照 nvm 的显示方式：简单列表，当前版本用 * 标记
		for _, v := range allVersions {
			if v.current {
				// 当前版本：显示 * 和详细信息
				arch := runtime.GOARCH
				fmt.Printf("* %s (Currently using %s executable)\n", v.version, arch)
			} else {
				// 其他版本：只显示版本号
				fmt.Println(v.version)
			}
		}

		return nil
	},
}

type versionInfo struct {
	version string
	source  string
	current bool
}

// sortVersions 排序版本：当前版本在前，其他版本按版本号降序
func sortVersions(versions []versionInfo) {
	sort.Slice(versions, func(i, j int) bool {
		// 当前版本优先
		if versions[i].current && !versions[j].current {
			return true
		}
		if !versions[i].current && versions[j].current {
			return false
		}
		// 其他版本按版本号降序
		return versions[i].version > versions[j].version
	})
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func detectSystemGo(vm *version.VersionManager) string {
	var ver string
	// 优先通过环境变量 GOROOT 读取版本文件
	if goroot := os.Getenv("GOROOT"); strings.TrimSpace(goroot) != "" {
		vf := filepath.Join(goroot, "VERSION")
		if b, err := os.ReadFile(vf); err == nil {
			lines := strings.Split(string(b), "\n")
			for _, ln := range lines {
				ln = strings.TrimSpace(ln)
				if ln == "" {
					continue
				}
				if strings.HasPrefix(ln, "go") {
					ver = ln
					break
				}
			}
		}
	}
	// 回退：通过 go 可执行路径推断 GOROOT 并读取 VERSION
	if ver == "" {
		goPath, err := exec.LookPath("go")
		if err == nil {
			goRoot := filepath.Dir(filepath.Dir(goPath))
			if !strings.Contains(goRoot, vm.GetInstallDir()) {
				vf := filepath.Join(goRoot, "VERSION")
				if b, err := os.ReadFile(vf); err == nil {
					lines := strings.Split(string(b), "\n")
					for _, ln := range lines {
						ln = strings.TrimSpace(ln)
						if ln == "" {
							continue
						}
						if strings.HasPrefix(ln, "go") {
							ver = ln
							break
						}
					}
				}
				// 如果 VERSION 不可用，解析 `go version` 输出
				if ver == "" {
					out, err := exec.Command(goPath, "version").CombinedOutput()
					if err == nil {
						fields := strings.Fields(string(out))
						for _, f := range fields {
							if strings.HasPrefix(f, "go") && len(f) > 2 && f[2] >= '0' && f[2] <= '9' {
								ver = f
								break
							}
						}
					}
				}
			}
		}
	}

	if ver == "" && runtime.GOOS == "windows" {
		pf := os.Getenv("ProgramFiles")
		candidate := filepath.Join(pf, "Go")
		vf := filepath.Join(candidate, "VERSION")
		if b, err := os.ReadFile(vf); err == nil {
			lines := strings.Split(string(b), "\n")
			for _, ln := range lines {
				ln = strings.TrimSpace(ln)
				if ln == "" {
					continue
				}
				if strings.HasPrefix(ln, "go") {
					ver = ln
					break
				}
			}
		} else {
			goexe := filepath.Join(candidate, "bin", "go.exe")
			if _, err := os.Stat(goexe); err == nil {
				out, err := exec.Command(goexe, "version").CombinedOutput()
				if err == nil {
					fields := strings.Fields(string(out))
					for _, f := range fields {
						if strings.HasPrefix(f, "go") && len(f) > 2 && f[2] >= '0' && f[2] <= '9' {
							ver = f
							break
						}
					}
				}
			}
		}
	}
	return ver
}
