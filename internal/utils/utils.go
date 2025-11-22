package utils

import (
    "archive/tar"
    "archive/zip"
    "bufio"
    "compress/gzip"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "time"
)

// DownloadFile 下载文件到指定路径（保持向后兼容）
func DownloadFile(url, destPath string) error {
	return DownloadFileWithProgress(url, destPath, 0)
}

// DownloadFileWithProgress 下载文件到指定路径，带进度显示
func DownloadFileWithProgress(url, destPath string, expectedSize int64) error {
	// 优化 HTTP 客户端：使用更激进的设置以提高下载速度
	transport := &http.Transport{
		DisableCompression:    true, // 文件已压缩，不需要再次压缩
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:    10,
		IdleConnTimeout:        90 * time.Second,
		ResponseHeaderTimeout:  30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// 禁用 HTTP/2，使用 HTTP/1.1 可能在某些情况下更快
		ForceAttemptHTTP2: false,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   0, // 无超时限制，因为文件可能很大
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// 设置请求头，优化下载
	req.Header.Set("User-Agent", "gvm/1.0")
	req.Header.Set("Accept-Encoding", "identity") // 禁用压缩，因为文件已压缩
	req.Header.Set("Connection", "keep-alive")     // 保持连接
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// 获取实际文件大小
	contentLength := resp.ContentLength
	if contentLength == -1 && expectedSize > 0 {
		contentLength = expectedSize
	}

	dir := filepath.Dir(destPath)
	if err := EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to ensure download dir: %w", err)
	}
	
	out, err := os.CreateTemp(dir, "gvm-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempName := out.Name()
	defer out.Close()

	// 使用 io.CopyBuffer 而不是手动循环，Go 标准库已经高度优化
	// 使用更大的缓冲区（1MB）以提高速度
	buf := make([]byte, 1024*1024) // 1MB 缓冲区
	
	// 使用带缓冲的写入
	bufferedOut := bufio.NewWriterSize(out, 4*1024*1024) // 4MB 写入缓冲区
	defer bufferedOut.Flush()
	
	// 创建带进度跟踪的 Reader
	startTime := time.Now()
	lastUpdateTime := startTime
	lastWritten := int64(0)
	lastProgress := int64(-1)
	
	progressReader := &progressReader{
		reader:        resp.Body,
		contentLength: contentLength,
		onProgress: func(written int64) {
			now := time.Now()
			if contentLength > 0 {
				progress := (written * 100) / contentLength
				elapsed := now.Sub(startTime).Seconds()
				shouldUpdate := (progress != lastProgress && progress%2 == 0) || 
				               (now.Sub(lastUpdateTime) >= 500*time.Millisecond)
				if shouldUpdate && elapsed > 0 {
					// 计算瞬时速度（最近0.5秒的速度）
					timeDiff := now.Sub(lastUpdateTime).Seconds()
					if timeDiff > 0 {
						recentSpeed := float64(written-lastWritten) / timeDiff
						
						fmt.Printf("\rProgress: %d%% (%.2f MB / %.2f MB) - %.2f MB/s", 
							progress, 
							float64(written)/(1024*1024), 
							float64(contentLength)/(1024*1024),
							recentSpeed/(1024*1024))
						lastProgress = progress
						lastUpdateTime = now
						lastWritten = written
					}
				}
			}
		},
	}
	
	// 使用 io.CopyBuffer 进行高效复制
	written, err := io.CopyBuffer(bufferedOut, progressReader, buf)
	if err != nil {
		os.Remove(tempName)
		return fmt.Errorf("failed to download file: %w", err)
	}
	
	// 完成进度显示
	if contentLength > 0 {
		elapsed := time.Since(startTime).Seconds()
		avgSpeed := float64(written) / elapsed
		fmt.Printf("\rProgress: 100%% (%.2f MB / %.2f MB) - Complete! (%.2f MB/s avg)\n",
			float64(written)/(1024*1024),
			float64(contentLength)/(1024*1024),
			avgSpeed/(1024*1024))
	}
	
	if err := out.Sync(); err != nil {
		os.Remove(tempName)
		return fmt.Errorf("failed to flush file: %w", err)
	}
	if err := out.Close(); err != nil {
		os.Remove(tempName)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	
	if FileExists(destPath) {
		_ = os.Remove(destPath)
	}
	if err := os.Rename(tempName, destPath); err != nil {
		// 回退到复制方案
		in, errOpen := os.Open(tempName)
		if errOpen != nil {
			os.Remove(tempName)
			return fmt.Errorf("failed to move file: %w", err)
		}
		defer in.Close()
		outFinal, errCreate := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if errCreate != nil {
			os.Remove(tempName)
			return fmt.Errorf("failed to move file: %w", err)
		}
		if _, errCopy := io.Copy(outFinal, in); errCopy != nil {
			outFinal.Close()
			os.Remove(tempName)
			return fmt.Errorf("failed to move file: %w", err)
		}
		outFinal.Close()
		os.Remove(tempName)
	}

	return nil
}

// progressReader 包装 io.Reader 以跟踪下载进度
type progressReader struct {
	reader        io.Reader
	contentLength int64
	written       int64
	onProgress    func(int64)
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.written += int64(n)
	if pr.onProgress != nil {
		pr.onProgress(pr.written)
	}
	return n, err
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

// ExtractZip 解压 zip 文件到指定目录（去除顶层 go/ 前缀）
func ExtractZip(zipPath, destPath string) error {
    r, err := zip.OpenReader(zipPath)
    if err != nil {
        return fmt.Errorf("failed to open zip: %w", err)
    }
    defer r.Close()

    if err := os.MkdirAll(destPath, 0755); err != nil {
        return fmt.Errorf("failed to create destination directory: %w", err)
    }

    for _, f := range r.File {
        name := strings.TrimPrefix(f.Name, "go/")
        targetPath := filepath.Join(destPath, name)

        if f.FileInfo().IsDir() {
            if err := os.MkdirAll(targetPath, f.Mode()); err != nil {
                return fmt.Errorf("failed to create directory: %w", err)
            }
            continue
        }

        if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
            return fmt.Errorf("failed to create parent directory: %w", err)
        }

        rc, err := f.Open()
        if err != nil {
            return fmt.Errorf("failed to open zipped file: %w", err)
        }

        out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
        if err != nil {
            rc.Close()
            return fmt.Errorf("failed to create file: %w", err)
        }

        if _, err := io.Copy(out, rc); err != nil {
            rc.Close()
            out.Close()
            return fmt.Errorf("failed to write file: %w", err)
        }
        rc.Close()
        out.Close()
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

// ComputeSHA256 计算文件的 SHA256 摘要
func ComputeSHA256(path string) (string, error) {
    f, err := os.Open(path)
    if err != nil {
        return "", fmt.Errorf("failed to open file for sha256: %w", err)
    }
    defer f.Close()

    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return "", fmt.Errorf("failed to hash file: %w", err)
    }
    sum := h.Sum(nil)
    return hex.EncodeToString(sum), nil
}

// VerifySHA256 校验文件的 SHA256 摘要
func VerifySHA256(path, expected string) error {
    sum, err := ComputeSHA256(path)
    if err != nil {
        return err
    }
    if !strings.EqualFold(sum, expected) {
        return fmt.Errorf("sha256 mismatch: expected %s, got %s", expected, sum)
    }
    return nil
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

// UpdatePathForWindows 使用 PowerShell profile 加载 ~/.gvm/env.ps1 以更新 PATH
func UpdatePathForWindows(goBinPath string) error {
    home, err := GetHomeDir()
    if err != nil {
        return err
    }
    gvmDir := filepath.Join(home, ".gvm")
    if err := EnsureDir(gvmDir); err != nil {
        return fmt.Errorf("failed to ensure gvm dir: %w", err)
    }
    envPs1 := filepath.Join(gvmDir, "env.ps1")
    content := fmt.Sprintf("$env:PATH=\"%s;\"+$env:PATH # GVM PATH\n", goBinPath)
    if err := os.WriteFile(envPs1, []byte(content), 0644); err != nil {
        return fmt.Errorf("failed to write env.ps1: %w", err)
    }

    // 为 cmd 提供 env.bat，以便当前会话可通过 call 立即生效
    envBat := filepath.Join(gvmDir, "env.bat")
    batContent := fmt.Sprintf("set PATH=%s;%%PATH%%\r\n", goBinPath)
    if err := os.WriteFile(envBat, []byte(batContent), 0644); err != nil {
        return fmt.Errorf("failed to write env.bat: %w", err)
    }

    profile := filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
    if err := EnsureDir(filepath.Dir(profile)); err != nil {
        return fmt.Errorf("failed to ensure powershell profile dir: %w", err)
    }
    var existing string
    if FileExists(profile) {
        b, _ := os.ReadFile(profile)
        existing = string(b)
    }
    dotSource := fmt.Sprintf(". \"%s\" # GVM INIT\n", envPs1)
    if !strings.Contains(existing, "# GVM INIT") && !strings.Contains(existing, envPs1) {
        existing = existing + dotSource
        if err := os.WriteFile(profile, []byte(existing), 0644); err != nil {
            return fmt.Errorf("failed to update powershell profile: %w", err)
        }
    }

    return nil
}

// GetShimsDir 返回 shims 目录路径
func GetShimsDir() (string, error) {
    home, err := GetHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".gvm", "shims"), nil
}

// UpdateShims 更新 go 可执行的 shim 以指向指定版本的 go 二进制
func UpdateShims(goBinPath string) error {
    shimsDir, err := GetShimsDir()
    if err != nil {
        return err
    }
    if err := EnsureDir(shimsDir); err != nil {
        return err
    }

    if runtime.GOOS == "windows" {
        // 生成 go.cmd 调用选定版本的 go.exe
        target := filepath.Join(goBinPath, "go.exe")
        cmdPath := filepath.Join(shimsDir, "go.cmd")
        content := fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", target)
        if err := os.WriteFile(cmdPath, []byte(content), 0644); err != nil {
            return fmt.Errorf("failed to write shim go.cmd: %w", err)
        }
    } else {
        // Unix: 创建/更新符号链接 ~/.gvm/shims/go -> <install>/bin/go
        target := filepath.Join(goBinPath, "go")
        linkPath := filepath.Join(shimsDir, "go")
        if FileExists(linkPath) {
            _ = os.Remove(linkPath)
        }
        if err := os.Symlink(target, linkPath); err != nil {
            return fmt.Errorf("failed to create go shim symlink: %w", err)
        }
    }

    return nil
}
