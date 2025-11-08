package output

import (
	"fmt"
	"os"
	"strings"
)

// Color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

// PrintSuccess 打印成功消息
func PrintSuccess(message string) {
	fmt.Printf("%s✓%s %s\n", ColorGreen, ColorReset, message)
}

// PrintError 打印错误消息
func PrintError(message string) {
	fmt.Fprintf(os.Stderr, "%s✗%s %s\n", ColorRed, ColorReset, message)
}

// PrintWarning 打印警告消息
func PrintWarning(message string) {
	fmt.Printf("%s⚠%s %s\n", ColorYellow, ColorReset, message)
}

// PrintInfo 打印信息消息
func PrintInfo(message string) {
	fmt.Printf("%sℹ%s %s\n", ColorBlue, ColorReset, message)
}

// PrintProgress 打印进度消息
func PrintProgress(message string) {
	fmt.Printf("%s⟳%s %s\n", ColorCyan, ColorReset, message)
}

// PrintHeader 打印标题
func PrintHeader(title string) {
	fmt.Printf("\n%s%s%s\n", ColorPurple, strings.ToUpper(title), ColorReset)
	fmt.Println(strings.Repeat("=", len(title)))
}

// PrintTableHeader 打印表格头部
func PrintTableHeader(headers ...string) {
	for i, header := range headers {
		if i == 0 {
			fmt.Printf("%s%-20s%s", ColorBlue, header, ColorReset)
		} else {
			fmt.Printf("%-15s", header)
		}
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 20+15*(len(headers)-1)))
}

// PrintTableRow 打印表格行
func PrintTableRow(values ...string) {
	for i, value := range values {
		if i == 0 {
			fmt.Printf("%-20s", value)
		} else {
			fmt.Printf("%-15s", value)
		}
	}
	fmt.Println()
}

// Confirm 询问用户确认
func Confirm(prompt string) bool {
	fmt.Printf("%s?%s %s (y/N): ", ColorYellow, ColorReset, prompt)

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// Spinner 显示加载动画
func Spinner(message string) func() {
	done := make(chan bool)
	go func() {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-done:
				fmt.Printf("\r%s%s%s\n", ColorGreen, "✓", ColorReset)
				return
			default:
				fmt.Printf("\r%s%s%s %s", ColorCyan, spinner[i%len(spinner)], ColorReset, message)
				i++
			}
		}
	}()

	return func() {
		done <- true
	}
}
