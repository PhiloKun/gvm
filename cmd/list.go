package cmd

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"
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

        current, _ := vm.GetCurrentVersion()

        output.PrintHeader("Installed Go versions")
        w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
        fmt.Fprintln(w, "VERSION\t\tSOURCE\t\tSTATUS")
        fmt.Fprintln(w, "-------\t\t------\t\t------")

        sysVer, sysSource, sysStatus := detectSystemGo(vm)
        if sysVer != "" {
            fmt.Fprintf(w, "%s\t\t%s\t\t%s\n", sysVer, sysSource, sysStatus)
        }

        for _, v := range versions {
            status := ""
            if v == current {
                status = output.ColorGreen + "current" + output.ColorReset
            }
            fmt.Fprintf(w, "%s\t\t%s\t\t%s\n", v, "gvm", status)
        }
        if sysVer == "" && len(versions) == 0 {
            output.PrintWarning("No Go found. Use 'gvm install <version>' to install one.")
        }
        w.Flush()

        return nil
    },
}

func init() {
    rootCmd.AddCommand(listCmd)
}

func detectSystemGo(vm *version.VersionManager) (ver string, src string, status string) {
    src = "system"
    status = ""
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
    cur, _ := vm.GetCurrentVersion()
    if cur == "system" && ver != "" {
        status = output.ColorGreen + "current" + output.ColorReset
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
    return
}
