package version

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/philokun/gvm/internal/config"
	"github.com/philokun/gvm/internal/utils"
)

const (
	GoDownloadURL     = "https://go.dev/dl/?mode=json"
	DefaultInstallDir = ".gvm/versions"
)

type GoVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	Files   []struct {
		Filename string `json:"filename"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		Version  string `json:"version"`
		SHA256   string `json:"sha256"`
		Size     int    `json:"size"`
	} `json:"files"`
}

type VersionManager struct {
	installDir string
}

func New() *VersionManager {
	homeDir, _ := os.UserHomeDir()
	return &VersionManager{
		installDir: filepath.Join(homeDir, DefaultInstallDir),
	}
}

func (vm *VersionManager) GetInstallDir() string {
	return vm.installDir
}

func (vm *VersionManager) GetAvailableVersions() ([]GoVersion, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(GoDownloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Go versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch versions page: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var versions []GoVersion
	if err := json.Unmarshal(body, &versions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal versions JSON: %w", err)
	}

	return versions, nil
}

func (vm *VersionManager) GetInstalledVersions() ([]string, error) {
	versions := []string{}

	entries, err := os.ReadDir(vm.installDir)
	if err != nil {
		if os.IsNotExist(err) {
			return versions, nil
		}
		return nil, fmt.Errorf("failed to read install directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "go") {
			versions = append(versions, entry.Name())
		}
	}

	return versions, nil
}

func (vm *VersionManager) GetCurrentVersion() (string, error) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("go command not found in PATH")
	}

	goRoot := filepath.Dir(filepath.Dir(goPath))
	if !strings.Contains(goRoot, vm.installDir) {
		return "system", nil
	}

	version := filepath.Base(goRoot)
	return version, nil
}

func (vm *VersionManager) InstallVersion(version string) error {
	// 检查版本是否已安装
	installed, err := vm.IsVersionInstalled(version)
	if err != nil {
		return err
	}
	if installed {
		return fmt.Errorf("version %s is already installed", version)
	}

	// 获取可用的版本信息
	availableVersions, err := vm.GetAvailableVersions()
	if err != nil {
		return err
	}

	// 找到对应的版本信息
	var targetVersion *GoVersion
	for i := range availableVersions {
		if availableVersions[i].Version == version {
			targetVersion = &availableVersions[i]
			break
		}
	}

	if targetVersion == nil {
		return fmt.Errorf("version %s not found in available versions", version)
	}

	// 找到适合当前系统的安装包
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	var targetFile *struct {
		Filename string `json:"filename"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		Version  string `json:"version"`
		SHA256   string `json:"sha256"`
		Size     int    `json:"size"`
	}

	for i := range targetVersion.Files {
		if targetVersion.Files[i].OS == runtime.GOOS && targetVersion.Files[i].Arch == runtime.GOARCH {
			targetFile = &targetVersion.Files[i]
			break
		}
	}

	if targetFile == nil {
		return fmt.Errorf("no suitable package found for %s", platform)
	}

	// 下载并安装
	downloadURL := fmt.Sprintf("https://go.dev/dl/%s", targetFile.Filename)
	installPath := filepath.Join(vm.installDir, version)

	// 确保安装目录存在
	if err := utils.EnsureDir(vm.installDir); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// 下载到临时文件
	tempFile := filepath.Join(os.TempDir(), targetFile.Filename)
	fmt.Printf("Downloading %s...\n", targetFile.Filename)
	if err := utils.DownloadFile(downloadURL, tempFile); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer os.Remove(tempFile)

	// 解压文件
	fmt.Printf("Extracting to %s...\n", installPath)
	if err := utils.ExtractTarGz(tempFile, installPath); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	// 更新配置
	if err := config.AddVersion(version); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	return nil
}

func (vm *VersionManager) IsVersionInstalled(version string) (bool, error) {
	installPath := filepath.Join(vm.installDir, version)
	_, err := os.Stat(installPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (vm *VersionManager) UseVersion(version string) error {
	installed, err := vm.IsVersionInstalled(version)
	if err != nil {
		return err
	}
	if !installed {
		return fmt.Errorf("version %s is not installed", version)
	}

	// 更新PATH环境变量
	goBinPath := filepath.Join(vm.installDir, version, "bin")

	// 更新配置文件
	if err := config.SetCurrentVersion(version); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	// 更新shell配置
	if err := utils.UpdatePathInShellConfig(goBinPath); err != nil {
		return fmt.Errorf("failed to update shell config: %w", err)
	}

	return nil
}

func (vm *VersionManager) UninstallVersion(version string) error {
	installed, err := vm.IsVersionInstalled(version)
	if err != nil {
		return err
	}
	if !installed {
		return fmt.Errorf("version %s is not installed", version)
	}

	// 检查是否是当前使用的版本
	current, _ := vm.GetCurrentVersion()
	if current == version {
		return fmt.Errorf("cannot uninstall currently active version %s", version)
	}

	installPath := filepath.Join(vm.installDir, version)
	if err := os.RemoveAll(installPath); err != nil {
		return fmt.Errorf("failed to remove installation directory: %w", err)
	}

	// 更新配置
	if err := config.RemoveVersion(version); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	return nil
}
