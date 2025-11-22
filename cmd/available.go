package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/philokun/gvm/internal/output"
	"github.com/philokun/gvm/internal/version"
	"github.com/spf13/cobra"
)

var (
	flagStable bool
	flagLimit  int
	flagJSON   bool
	flagMirror string
)

// availableCmd represents the available command
var availableCmd = &cobra.Command{
	Use:   "available",
	Short: "List available Go versions",
	Long:  "Fetch and list available Go versions from the official source or configured mirror.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(flagMirror) != "" {
			os.Setenv("GVM_DL_MIRROR", strings.TrimRight(flagMirror, "/"))
		}
		vm := version.New()
		versions, err := vm.GetAvailableVersions()
		if err != nil {
			return fmt.Errorf("failed to fetch available versions: %w", err)
		}

		// filter: if --stable flag is set, only show stable versions; otherwise show all
		filtered := make([]version.GoVersion, 0, len(versions))
		for _, v := range versions {
			// 如果设置了 --stable 标志，只显示稳定版本；否则显示所有版本
			if flagStable {
				if v.Stable {
					filtered = append(filtered, v)
				}
			} else {
				// 默认显示所有版本（包括不稳定的）
				filtered = append(filtered, v)
			}
		}

		// sort by version string descending (newest first)
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Version > filtered[j].Version })
		// API 已按最新在前返回；如需限制，截断
		if flagLimit > 0 && flagLimit < len(filtered) {
			filtered = filtered[:flagLimit]
		}

		if flagJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(filtered)
		}

		// 分类版本
		current, lts, oldStable, oldUnstable := categorizeVersions(filtered)

		// 显示多列表格
		output.PrintHeader("Available Go versions")
		printVersionTable(current, lts, oldStable, oldUnstable)
		return nil
	},
}

// parseVersionNumber 解析版本号，返回主版本号和次版本号
func parseVersionNumber(version string) (major, minor int, isUnstable bool) {
	// 移除 "go" 前缀
	version = strings.TrimPrefix(version, "go")

	// 检查是否是不稳定版本
	isUnstable = strings.Contains(strings.ToLower(version), "rc") ||
		strings.Contains(strings.ToLower(version), "beta")

	// 提取主版本号和次版本号 (例如: 1.25.4 -> major=1, minor=25)
	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) >= 3 {
		major, _ = strconv.Atoi(matches[1])
		minor, _ = strconv.Atoi(matches[2])
	}
	return
}

// categorizeVersions 将版本分类为 CURRENT, LTS, OLD STABLE, OLD UNSTABLE
func categorizeVersions(versions []version.GoVersion) (current, lts, oldStable, oldUnstable []version.GoVersion) {
	if len(versions) == 0 {
		return
	}

	// 找出最新的次版本号（Go 版本格式是 go1.x.y，主版本号都是1）
	maxMinor := 0
	for _, v := range versions {
		_, minor, _ := parseVersionNumber(v.Version)
		if minor > maxMinor {
			maxMinor = minor
		}
	}

	// CURRENT: 只包含最新次版本的所有版本（例如：如果最新是 1.25，则只包含 1.25.x）
	// LTS: 包含其他较新的稳定版本（例如：1.24.x, 1.23.x 等稳定版本）
	// OLD STABLE: 包含更旧的稳定版本（例如：1.9.x, 1.8.x 等）
	// OLD UNSTABLE: 包含旧的不稳定版本

	// 计算 LTS 和 OLD STABLE 的分界线（比如 1.20 之前的算 OLD STABLE）
	oldStableThreshold := 20

	for _, v := range versions {
		_, minor, isUnstable := parseVersionNumber(v.Version)

		if minor == maxMinor {
			// CURRENT: 最新次版本的所有版本（包括稳定和不稳定）
			current = append(current, v)
		} else if v.Stable {
			// 稳定版本：根据次版本号判断是 LTS 还是 OLD STABLE
			if minor >= oldStableThreshold {
				// LTS: 较新的稳定版本（1.20 及以上）
				lts = append(lts, v)
			} else {
				// OLD STABLE: 旧的稳定版本（1.19 及以下）
				oldStable = append(oldStable, v)
			}
		} else if isUnstable {
			// OLD UNSTABLE: 旧的不稳定版本（不在 CURRENT 中的不稳定版本）
			oldUnstable = append(oldUnstable, v)
		}
	}

	// 对每个分类进行排序（降序）
	sortVersions := func(vs []version.GoVersion) {
		sort.Slice(vs, func(i, j int) bool {
			return vs[i].Version > vs[j].Version
		})
	}
	sortVersions(current)
	sortVersions(lts)
	sortVersions(oldStable)
	sortVersions(oldUnstable)

	return
}

// printVersionTable 打印多列表格
func printVersionTable(current, lts, oldStable, oldUnstable []version.GoVersion) {
	// 限制显示数量（CURRENT 显示更多，其他列限制数量）
	const maxCurrent = 15
	const maxOther = 20
	if len(current) > maxCurrent {
		current = current[:maxCurrent]
	}
	if len(lts) > maxOther {
		lts = lts[:maxOther]
	}
	if len(oldStable) > maxOther {
		oldStable = oldStable[:maxOther]
	}
	if len(oldUnstable) > maxOther {
		oldUnstable = oldUnstable[:maxOther]
	}

	// 计算最大行数
	maxRows := len(current)
	if len(lts) > maxRows {
		maxRows = len(lts)
	}
	if len(oldStable) > maxRows {
		maxRows = len(oldStable)
	}
	if len(oldUnstable) > maxRows {
		maxRows = len(oldUnstable)
	}

	// 定义列宽
	const colWidth = 18

	// 打印表格顶部边框（使用 ASCII 字符）
	fmt.Printf("\n+%s+%s+%s+%s+\n",
		strings.Repeat("-", colWidth),
		strings.Repeat("-", colWidth),
		strings.Repeat("-", colWidth),
		strings.Repeat("-", colWidth))

	// 打印表头（颜色代码不影响对齐，因为它们是控制字符）
	fmt.Printf("|%s%-*s%s|%s%-*s%s|%s%-*s%s|%s%-*s%s|\n",
		output.ColorCyan, colWidth, "CURRENT", output.ColorReset,
		output.ColorGreen, colWidth, "LTS", output.ColorReset,
		output.ColorBlue, colWidth, "OLD STABLE", output.ColorReset,
		output.ColorYellow, colWidth, "OLD UNSTABLE", output.ColorReset)

	// 打印表头分隔线
	fmt.Printf("+%s+%s+%s+%s+\n",
		strings.Repeat("-", colWidth),
		strings.Repeat("-", colWidth),
		strings.Repeat("-", colWidth),
		strings.Repeat("-", colWidth))

	// 打印表格内容
	for i := 0; i < maxRows; i++ {
		cols := []string{"", "", "", ""}

		if i < len(current) {
			cols[0] = current[i].Version
		}
		if i < len(lts) {
			cols[1] = lts[i].Version
		}
		if i < len(oldStable) {
			cols[2] = oldStable[i].Version
		}
		if i < len(oldUnstable) {
			cols[3] = oldUnstable[i].Version
		}

		// 只打印至少有一列有内容的行
		hasContent := false
		for _, col := range cols {
			if col != "" {
				hasContent = true
				break
			}
		}
		if hasContent {
			// 使用固定宽度对齐，带边框
			fmt.Printf("|%-*s|%-*s|%-*s|%-*s|\n",
				colWidth, cols[0],
				colWidth, cols[1],
				colWidth, cols[2],
				colWidth, cols[3])
		}
	}

	// 打印表格底部边框
	fmt.Printf("+%s+%s+%s+%s+\n",
		strings.Repeat("-", colWidth),
		strings.Repeat("-", colWidth),
		strings.Repeat("-", colWidth),
		strings.Repeat("-", colWidth))
}

func init() {
	rootCmd.AddCommand(availableCmd)
	availableCmd.Flags().BoolVar(&flagStable, "stable", false, "show only stable versions")
	availableCmd.Flags().IntVar(&flagLimit, "limit", 0, "limit the number of results")
	availableCmd.Flags().BoolVar(&flagJSON, "json", false, "output as JSON")
	availableCmd.Flags().StringVar(&flagMirror, "mirror", "", "override download mirror base URL")
}
