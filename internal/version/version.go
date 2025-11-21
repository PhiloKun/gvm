package version

// 包 version 提供了 Go 版本管理的核心功能，包括获取可用版本、安装、卸载和切换版本。

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"
    "time"

    "github.com/philokun/gvm/internal/config"
    "github.com/philokun/gvm/internal/utils"
)

const (
    DefaultInstallDir = ".gvm/versions"
)

// getBaseURL 返回下载与版本 JSON 的基址，支持通过环境变量覆盖镜像
func getBaseURL() string {
    if v := os.Getenv("GVM_DL_MIRROR"); v != "" {
        return strings.TrimRight(v, "/")
    }
    return "https://go.dev"
}

func getAltBaseURL() string {
    return "https://golang.google.cn"
}

// GoVersion 表示一个 Go 版本及其相关文件信息。
type GoVersion struct {
	Version string `json:"version"` // 版本号，例如 "go1.20.5"
	Stable  bool   `json:"stable"`  // 是否为稳定版本
	Files   []struct {
		Filename string `json:"filename"` // 文件名
		OS       string `json:"os"`       // 操作系统
		Arch     string `json:"arch"`     // 架构
		Version  string `json:"version"`  // 版本号
		SHA256   string `json:"sha256"`   // 文件的 SHA256 校验值
		Size     int    `json:"size"`     // 文件大小
	} `json:"files"`
}

// VersionManager 是 Go 版本管理器，封装了所有版本管理相关的方法。
type VersionManager struct {
	installDir string // 安装目录
}

// New 创建一个新的 VersionManager 实例。
func New() *VersionManager {
	homeDir, _ := os.UserHomeDir()
	return &VersionManager{
		installDir: filepath.Join(homeDir, DefaultInstallDir),
	}
}

// GetInstallDir 返回安装目录路径。
func (vm *VersionManager) GetInstallDir() string {
	return vm.installDir
}

// GetAvailableVersions 获取 Go 官方提供的可用版本列表。
func (vm *VersionManager) GetAvailableVersions() ([]GoVersion, error) {
    client := &http.Client{Timeout: 30 * time.Second}
    bases := []string{getBaseURL(), getAltBaseURL()}
    var lastErr error
    for _, base := range bases {
        url := fmt.Sprintf("%s/dl/?mode=json", base)
        for i := 0; i < 3; i++ {
            resp, err := client.Get(url)
            if err != nil {
                lastErr = err
                time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
                continue
            }
            if resp.StatusCode != http.StatusOK {
                lastErr = fmt.Errorf("bad status: %s", resp.Status)
                resp.Body.Close()
                time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
                continue
            }
            body, err := io.ReadAll(resp.Body)
            resp.Body.Close()
            if err != nil {
                lastErr = err
                time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
                continue
            }
            var versions []GoVersion
            if err := json.Unmarshal(body, &versions); err != nil {
                lastErr = err
                time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
                continue
            }
            return versions, nil
        }
    }
    return nil, fmt.Errorf("failed to fetch Go versions: %w", lastErr)
}

// GetLatestStable 返回最新稳定版的版本号（如 go1.21.5）
func (vm *VersionManager) GetLatestStable() (string, error) {
    versions, err := vm.GetAvailableVersions()
    if err != nil {
        return "", err
    }
    for _, v := range versions {
        if v.Stable {
            return v.Version, nil
        }
    }
    return "", fmt.Errorf("no stable versions found")
}

// GetInstalledVersions 获取已安装的 Go 版本列表。
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

// GetCurrentVersion 获取当前正在使用的 Go 版本。
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

// InstallVersion 安装指定的 Go 版本。
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

    // 下载并安装（带镜像回退与重试）
    bases := []string{getBaseURL(), getAltBaseURL()}
    var downloadURL string
    var tempFile string
    var downloaded bool
    for _, base := range bases {
        downloadURL = fmt.Sprintf("%s/dl/%s", base, targetFile.Filename)
        tempFile = filepath.Join(os.TempDir(), targetFile.Filename)
        var lastErr error
        for i := 0; i < 3; i++ {
            fmt.Printf("Downloading %s...\n", targetFile.Filename)
            if err := utils.DownloadFile(downloadURL, tempFile); err != nil {
                lastErr = err
                time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
                continue
            }
            lastErr = nil
            break
        }
        if lastErr == nil {
            downloaded = true
            break
        }
    }
    if !downloaded {
        return fmt.Errorf("failed to download %s from all mirrors", targetFile.Filename)
    }
    defer os.Remove(tempFile)
    installPath := filepath.Join(vm.installDir, version)

	// 确保安装目录存在
	if err := utils.EnsureDir(vm.installDir); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

    // 下载已完成（上方循环），继续校验与解压

    // 校验文件
    if targetFile.SHA256 != "" {
        if err := utils.VerifySHA256(tempFile, targetFile.SHA256); err != nil {
            return fmt.Errorf("failed to verify sha256: %w", err)
        }
    }

    // 解压文件（根据扩展名）
    fmt.Printf("Extracting to %s...\n", installPath)
    if strings.HasSuffix(strings.ToLower(targetFile.Filename), ".tar.gz") {
        if err := utils.ExtractTarGz(tempFile, installPath); err != nil {
            return fmt.Errorf("failed to extract tar.gz: %w", err)
        }
    } else if strings.HasSuffix(strings.ToLower(targetFile.Filename), ".zip") {
        if err := utils.ExtractZip(tempFile, installPath); err != nil {
            return fmt.Errorf("failed to extract zip: %w", err)
        }
    } else {
        return fmt.Errorf("unsupported package format: %s", targetFile.Filename)
    }

    // 安装后验证：读取 VERSION 文件并检查二进制存在
    verFile := filepath.Join(installPath, "VERSION")
    b, err := os.ReadFile(verFile)
    if err != nil {
        _ = os.RemoveAll(installPath)
        return fmt.Errorf("validation failed: missing VERSION: %w", err)
    }
    installedVer := strings.TrimSpace(string(b))
    if installedVer != version {
        _ = os.RemoveAll(installPath)
        return fmt.Errorf("validation failed: version mismatch: expected %s got %s", version, installedVer)
    }
    goBin := filepath.Join(installPath, "bin", "go")
    if runtime.GOOS == "windows" {
        goBin = filepath.Join(installPath, "bin", "go.exe")
    }
    if _, err := os.Stat(goBin); err != nil {
        _ = os.RemoveAll(installPath)
        return fmt.Errorf("validation failed: go binary missing: %w", err)
    }

    // 更新配置
    if err := config.AddVersion(version); err != nil {
        return fmt.Errorf("failed to update config: %w", err)
    }

	return nil
}

// IsVersionInstalled 检查指定版本是否已安装。
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

// UseVersion 切换当前使用的 Go 版本。
func (vm *VersionManager) UseVersion(version string) error {
	installed, err := vm.IsVersionInstalled(version)
	if err != nil {
		return err
	}
	if !installed {
		return fmt.Errorf("version %s is not installed", version)
	}

    // 目标二进制路径
    goBinPath := filepath.Join(vm.installDir, version, "bin")

	// 更新配置文件
	if err := config.SetCurrentVersion(version); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

    // 更新 shims 指向选定版本
    if err := utils.UpdateShims(goBinPath); err != nil {
        return fmt.Errorf("failed to update shims: %w", err)
    }

    // 确保 PATH 包含 shims 目录（一次性）
    shimsDir, err := utils.GetShimsDir()
    if err != nil {
        return err
    }
    if runtime.GOOS == "windows" {
        if err := utils.UpdatePathForWindows(shimsDir); err != nil {
            return fmt.Errorf("failed to update windows env: %w", err)
        }
    } else {
        if err := utils.UpdatePathInShellConfig(shimsDir); err != nil {
            return fmt.Errorf("failed to update shell config: %w", err)
        }
    }

	return nil
}

// UninstallVersion 卸载指定的 Go 版本。
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