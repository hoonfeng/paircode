package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// WorkspaceToolConfig 工作区审核配置文件结构（审核机制独立于工具集，保留）。
// 路径：.pair/tools.json
// 格式：
//
//	{
//	  "reviewMode": "auto",
//	  "reviewBlacklist": ["delete_file"],
//	  "reviewWhitelist": ["read_file"]
//	}
//
// ★ 旧版 tools 工具开关字段（Tools/ToolConfigItem）已随「工具集（插件化）」机制
// 废弃删除：工具开关由 .pair/toolsets/*.json 的 DisabledTools 管理，
// 不再从 tools.json 读写 enabled。删除记录见 2026-08-16 提交。
type WorkspaceToolConfig struct {
	ReviewMode      string   `json:"reviewMode,omitempty"`      // auto/manual/off
	ReviewBlacklist []string `json:"reviewBlacklist,omitempty"`
	ReviewWhitelist []string `json:"reviewWhitelist,omitempty"`
}

// WorkspaceToolConfigPath 返回工作区审核配置文件的路径。
func WorkspaceToolConfigPath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".pair", "tools.json")
}

// readWorkspaceToolConfig 读取工作区配置文件的原始 JSON 内容。
func readWorkspaceToolConfig(workspaceRoot string) ([]byte, error) {
	return os.ReadFile(WorkspaceToolConfigPath(workspaceRoot))
}

// writeWorkspaceToolConfigRaw 将原始 JSON 写入工作区配置文件。
func writeWorkspaceToolConfigRaw(workspaceRoot string, data []byte) error {
	path := WorkspaceToolConfigPath(workspaceRoot)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadWorkspaceReviewConfig 从工作区加载审核配置（reviewMode / reviewBlacklist / reviewWhitelist）。
func LoadWorkspaceReviewConfig(workspaceRoot string) (mode string, blacklist, whitelist []string) {
	if workspaceRoot == "" {
		return "auto", nil, nil
	}
	data, err := readWorkspaceToolConfig(workspaceRoot)
	if err != nil {
		return "auto", nil, nil
	}
	var cfg WorkspaceToolConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "auto", nil, nil
	}
	if cfg.ReviewMode == "" {
		return "auto", cfg.ReviewBlacklist, cfg.ReviewWhitelist
	}
	return cfg.ReviewMode, cfg.ReviewBlacklist, cfg.ReviewWhitelist
}

// SaveWorkspaceReviewConfig 将审核配置保存到工作区配置文件。
func SaveWorkspaceReviewConfig(workspaceRoot string, mode string, blacklist, whitelist []string) error {
	if workspaceRoot == "" {
		return nil
	}
	path := WorkspaceToolConfigPath(workspaceRoot)
	data, err := os.ReadFile(path)
	var cfg WorkspaceToolConfig
	if err == nil {
		json.Unmarshal(data, &cfg)
	}
	cfg.ReviewMode = mode
	cfg.ReviewBlacklist = blacklist
	cfg.ReviewWhitelist = whitelist
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return writeWorkspaceToolConfigRaw(workspaceRoot, out)
}
