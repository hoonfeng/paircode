// 配置管理 —— 读取/写入项目配置
//
//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 启动面板配置
type Config struct {
	Port      int                 `json:"port"`
	AutoStart bool                `json:"autoStart"`
	Models    map[string][]string `json:"models"`
	BaseURL   string              `json:"baseURL"`
	APIKey    string              `json:"apiKey"`
	Provider  string              `json:"provider"`
}

// 默认配置路径
func configPath() string {
	// 与 companion 共享配置目录
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	return filepath.Join(dir, "..", "config", "tray-settings.json")
}

// companion 配置路径
func companionConfigPath() string {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	return filepath.Join(dir, "..", "config", "settings.json")
}

func companionModelsPath() string {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	return filepath.Join(dir, "..", "config", "models.json")
}

// loadConfig 加载配置，如文件不存在则用默认值
func loadConfig() *Config {
	cfg := &Config{
		Port:      9090,
		AutoStart: false,
		Models:    make(map[string][]string),
		BaseURL:   "https://api.deepseek.com/v1",
		Provider:  "deepseek",
	}

	// 尝试从 companion 的 settings.json 读取
	if data, err := os.ReadFile(companionConfigPath()); err == nil {
		var companionCfg struct {
			BaseURL      string `json:"baseURL"`
			APIKey       string `json:"apiKey"`
			Provider     string `json:"provider"`
			Model        string `json:"model"`
			PlanModel    string `json:"planModel"`
			ExecuteModel string `json:"executeModel"`
			ReviewModel  string `json:"reviewModel"`
		}
		if json.Unmarshal(data, &companionCfg) == nil {
			cfg.BaseURL = companionCfg.BaseURL
			cfg.APIKey = companionCfg.APIKey
			cfg.Provider = companionCfg.Provider
		}
	}

	// 从 companion 的 models.json 读取模型列表
	if data, err := os.ReadFile(companionModelsPath()); err == nil {
		var models map[string][]string
		if json.Unmarshal(data, &models) == nil {
			cfg.Models = models
		}
	}

	// 从 tray 专用配置读取（覆盖）
	if data, err := os.ReadFile(configPath()); err == nil {
		var trayCfg Config
		if json.Unmarshal(data, &trayCfg) == nil {
			if trayCfg.Port != 0 {
				cfg.Port = trayCfg.Port
			}
			cfg.AutoStart = trayCfg.AutoStart
		}
	}

	return cfg
}

// Save 保存配置
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	path := configPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置失败: %w", err)
	}

	// 同步模型列表到 companion 的 models.json
	modelsData, err := json.MarshalIndent(c.Models, "", "  ")
	if err == nil {
		os.WriteFile(companionModelsPath(), modelsData, 0644)
	}

	return nil
}

// GetCompanionExePath 获取 companion 可执行文件路径
func GetCompanionExePath() string {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	// tray 和 companion 在同一目录
	return filepath.Join(dir, "companion.exe")
}

// GetCompanionWorkDir 获取 companion 工作目录
func GetCompanionWorkDir() string {
	exe, _ := os.Executable()
	return filepath.Dir(filepath.Dir(exe))
}
