package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DownloadFile 下载文件到指定路径
func DownloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// 创建临时文件
	out, err := os.CreateTemp("", "gvm-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()
	defer os.Remove(out.Name())

	// 写入临时文件
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	// 移动到最终位置
	if err := os.Rename(out.Name(), filepath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// ExtractTarGz 解压 tar.gz 文件到指定目录
func ExtractTarGz(tarGzPath, destPath string) error {
	// 打开 tar.gz 文件
	file, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	// 创建 gzip 读取器
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// 创建 tar 读取器
	tarReader := tar.NewReader(gzReader)

	// 创建目标目录
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 解压文件
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// 构建目标路径
		targetPath := filepath.Join(destPath, strings.TrimPrefix(header.Name, "go/"))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := extractFile(tarReader, targetPath, header.Mode); err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}
		}
	}

	return nil
}

func extractFile(reader *tar.Reader, path string, mode int64) error {
	// 创建文件
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(mode))
	if err != nil {
		return err
	}
	defer file.Close()

	// 复制内容
	_, err = io.Copy(file, reader)
	return err
}

// GetPlatform 获取当前平台信息
func GetPlatform() string {
	return fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// EnsureDir 确保目录存在，如果不存在则创建
func EnsureDir(path string) error {
	if !FileExists(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// GetHomeDir 获取用户主目录
func GetHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return home, nil
}

// GetShellConfigFile 获取当前用户的shell配置文件路径
func GetShellConfigFile() (string, error) {
	home, err := GetHomeDir()
	if err != nil {
		return "", err
	}

	// 检测当前shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "", fmt.Errorf("unable to detect current shell")
	}

	shellName := filepath.Base(shell)

	switch shellName {
	case "bash":
		bashrc := filepath.Join(home, ".bashrc")
		if FileExists(bashrc) {
			return bashrc, nil
		}
		return filepath.Join(home, ".bash_profile"), nil
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s", shellName)
	}
}

// UpdatePathInShellConfig 更新shell配置文件中的PATH
func UpdatePathInShellConfig(goBinPath string) error {
	configFile, err := GetShellConfigFile()
	if err != nil {
		return err
	}

	// 读取现有内容
	content, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read shell config: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	newLines := []string{}

	// 移除旧的GVM PATH设置
	for _, line := range lines {
		if !strings.Contains(line, "# GVM PATH") && !strings.Contains(line, ".gvm/versions") {
			newLines = append(newLines, line)
		}
	}

	// 添加新的PATH设置
	exportLine := fmt.Sprintf("export PATH=\"%s:$PATH\" # GVM PATH", goBinPath)
	newLines = append(newLines, exportLine)

	// 写回文件
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(configFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to update shell config: %w", err)
	}

	return nil
}
