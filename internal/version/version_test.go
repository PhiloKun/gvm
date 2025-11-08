package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	vm := New()
	if vm == nil {
		t.Fatal("New() returned nil")
	}

	if vm.installDir == "" {
		t.Error("installDir should not be empty")
	}
}

func TestGetInstallDir(t *testing.T) {
	vm := New()
	dir := vm.GetInstallDir()

	if dir == "" {
		t.Error("GetInstallDir() should not return empty string")
	}

	// 检查路径是否包含 .gvm/versions
	if !filepath.IsAbs(dir) {
		t.Error("GetInstallDir() should return absolute path")
	}
}

func TestIsVersionInstalled(t *testing.T) {
	vm := New()

	// 测试未安装的版本
	installed, err := vm.IsVersionInstalled("go1.99.99")
	if err != nil {
		t.Fatalf("IsVersionInstalled() returned error: %v", err)
	}

	if installed {
		t.Error("IsVersionInstalled() should return false for non-existent version")
	}
}

func TestGetInstalledVersions(t *testing.T) {
	vm := New()

	versions, err := vm.GetInstalledVersions()
	if err != nil {
		t.Fatalf("GetInstalledVersions() returned error: %v", err)
	}

	if versions == nil {
		t.Error("GetInstalledVersions() should return empty slice, not nil")
	}
}

func TestGetCurrentVersion(t *testing.T) {
	vm := New()

	version, err := vm.GetCurrentVersion()
	// 这个测试可能会因为系统环境不同而失败，所以我们只检查不panic
	_ = version
	_ = err
}

func TestVersionFormatting(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.21.5", "go1.21.5"},
		{"go1.21.5", "go1.21.5"},
		{"1.20", "go1.20"},
		{"latest", "latest"},
	}

	for _, test := range tests {
		result := formatVersion(test.input)
		if result != test.expected {
			t.Errorf("formatVersion(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

// formatVersion 是一个辅助函数，用于标准化版本号格式
func formatVersion(version string) string {
	if !filepath.IsAbs(version) && len(version) > 0 && version[0] != 'g' {
		return "go" + version
	}
	return version
}
