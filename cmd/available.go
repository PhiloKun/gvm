package cmd

import (
    "encoding/json"
    "fmt"
    "os"
    "sort"
    "strings"
    "text/tabwriter"

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

        // optionally filter stable
        filtered := make([]version.GoVersion, 0, len(versions))
        for _, v := range versions {
            if !flagStable || v.Stable {
                filtered = append(filtered, v)
            }
        }

        // sort by version string descending (strings compare works for goX.Y.Z)
        sort.Slice(filtered, func(i, j int) bool { return filtered[i].Version < filtered[j].Version })
        // API 已按最新在前返回；如需限制，截断
        if flagLimit > 0 && flagLimit < len(filtered) {
            filtered = filtered[:flagLimit]
        }

        if flagJSON {
            enc := json.NewEncoder(os.Stdout)
            enc.SetIndent("", "  ")
            return enc.Encode(filtered)
        }

        output.PrintHeader("Available Go versions")
        w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
        fmt.Fprintln(w, "VERSION\t\tSTABLE")
        fmt.Fprintln(w, "-------\t\t------")
        for _, v := range filtered {
            stable := ""
            if v.Stable {
                stable = output.ColorGreen + "yes" + output.ColorReset
            } else {
                stable = "no"
            }
            fmt.Fprintf(w, "%s\t\t%s\n", v.Version, stable)
        }
        w.Flush()
        return nil
    },
}

func init() {
    rootCmd.AddCommand(availableCmd)
    availableCmd.Flags().BoolVar(&flagStable, "stable", false, "show only stable versions")
    availableCmd.Flags().IntVar(&flagLimit, "limit", 0, "limit the number of results")
    availableCmd.Flags().BoolVar(&flagJSON, "json", false, "output as JSON")
    availableCmd.Flags().StringVar(&flagMirror, "mirror", "", "override download mirror base URL")
}