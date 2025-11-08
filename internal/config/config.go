package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	CurrentVersion string                 `json:"current_version"`
	InstallDir     string                 `json:"install_dir"`
	Versions       map[string]VersionInfo `json:"versions"`
}

type VersionInfo struct {
	InstalledDate string `json:"installed_date"`
	Active        bool   `json:"active"`
}

var (
	defaultConfig Config
	configPath    string
)

func init() {
	homeDir, _ := os.UserHomeDir()
	configPath = filepath.Join(homeDir, ".gvm", "config.json")
	defaultConfig = Config{
		InstallDir: filepath.Join(homeDir, ".gvm", "versions"),
		Versions:   make(map[string]VersionInfo),
	}
}

func Load() (*Config, error) {
	config := defaultConfig

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，创建默认配置
			if err := Save(&config); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			return &config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func Save(config *Config) error {
	// 确保配置目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func GetCurrentVersion() (string, error) {
	config, err := Load()
	if err != nil {
		return "", err
	}
	return config.CurrentVersion, nil
}

func SetCurrentVersion(version string) error {
	config, err := Load()
	if err != nil {
		return err
	}

	// 重置所有版本的状态
	for k := range config.Versions {
		info := config.Versions[k]
		info.Active = false
		config.Versions[k] = info
	}

	// 设置新版本为激活状态
	if info, exists := config.Versions[version]; exists {
		info.Active = true
		config.Versions[version] = info
	}

	config.CurrentVersion = version
	return Save(config)
}

func AddVersion(version string) error {
	config, err := Load()
	if err != nil {
		return err
	}

	config.Versions[version] = VersionInfo{
		InstalledDate: time.Now().Format("2006-01-02 15:04:05"),
		Active:        false,
	}

	return Save(config)
}

func RemoveVersion(version string) error {
	config, err := Load()
	if err != nil {
		return err
	}

	delete(config.Versions, version)

	// 如果删除的是当前版本，重置当前版本
	if config.CurrentVersion == version {
		config.CurrentVersion = ""
	}

	return Save(config)
}

func GetInstallDir() (string, error) {
	config, err := Load()
	if err != nil {
		return "", err
	}
	return config.InstallDir, nil
}
