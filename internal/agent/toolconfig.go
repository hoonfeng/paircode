package agent

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// WorkspaceToolConfig 工作区工具配置文件结构
// 路径：.pair/tools.json
// 格式：
//
//	{
//	  "tools": {
//	    "run_command": { "enabled": false },
//	    "git_status":  { "enabled": true }
//	  }
//	}
type WorkspaceToolConfig struct {
	Tools map[string]ToolConfigItem `json:"tools"`
}

// ToolConfigItem 单个工具的配置
type ToolConfigItem struct {
	Enabled *bool `json:"enabled,omitempty"` // nil=不覆盖，true/false=强制启用/禁用
}

// LoadWorkspaceToolConfig 从工作区加载工具配置文件并应用到 Registry。
// 配置文件路径：{workspaceRoot}/.pair/tools.json
// 文件可选，不存在时不报错。
// 配置覆盖规则：
//   - enabled=true → 强制启用（即使注册时默认值是 false）
//   - enabled=false → 禁用该工具（从 LLM 可见工具列表中移除）
//   - 未在配置中出现的工具 → 保持注册时的默认值
func LoadWorkspaceToolConfig(r *Registry, workspaceRoot string) {
	if workspaceRoot == "" {
		return
	}
	cfgPath := filepath.Join(workspaceRoot, ".pair", "tools.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[WorkspaceToolConfig] 读取失败 %s: %v", cfgPath, err)
		}
		return // 文件不存在是正常情况
	}
	var cfg WorkspaceToolConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("[WorkspaceToolConfig] 解析失败 %s: %v", cfgPath, err)
		return
	}
	if len(cfg.Tools) == 0 {
		return
	}
	for name, item := range cfg.Tools {
		if item.Enabled != nil {
			r.SetToolEnabled(name, *item.Enabled)
			status := "启用"
			if !*item.Enabled {
				status = "禁用"
			}
			log.Printf("[WorkspaceToolConfig] 工具 %s -> %s（工作区配置）", name, status)
		}
	}
}
