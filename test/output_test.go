package test

import "testing"

import "github.com/philokun/gvm/internal/output"


func TestPrintProgress(t *testing.T) {
	output.PrintProgress("Installing Go 1.19.4...")
}

func TestPrinSuccess(t *testing.T) {
	output.PrintSuccess("Go 1.19.4 installed")
}
func TestPrintError(t *testing.T) {
	output.PrintError("Failed to install version go1.19.4")
}
func TestPrintWarning(t *testing.T) {
	output.PrintWarning("Version go1.19.4 is not available")
}
func TestPrintInfo(t *testing.T) {
	output.PrintInfo("Use 'gvm use go1.19.4' to switch to this version")
}
func TestPrintTableHeader(t *testing.T) {
	output.PrintTableHeader("Version", "Status")
}

func TestSpinner(t *testing.T) {
	output.Spinner("Installing Go 1.19.4...")
}
