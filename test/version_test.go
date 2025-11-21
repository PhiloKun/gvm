package test

import (
	"testing"
	"os"
)



func TestVersionManager(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	t.Logf("Home directory: %s", homeDir)
}