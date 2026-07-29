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
// 行为：
//   - 文件不存在 → 自动初始化：用注册表中所有工具的当前 Enabled 状态写入文件
//   - 文件存在但缺失某些工具 → 自动补充缺失的工具到文件
//   - 文件存在 → 读取并应用配置
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
			return
		}
		// 文件不存在 → 自动初始化：用注册表中所有工具的状态写入
		initWorkspaceToolConfig(r, cfgPath)
		return
	}
	var cfg WorkspaceToolConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("[WorkspaceToolConfig] 解析失败 %s: %v", cfgPath, err)
		return
	}
	// 应用配置（设置启用/禁用）
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
	// ★ 补充注册表中存在但配置中缺失的工具（保持配置文件完整）
	mergeMissingTools(r, &cfg, cfgPath)
}

// mergeMissingTools 检查注册表中是否有配置遗漏的工具，补充并写回文件。
func mergeMissingTools(r *Registry, cfg *WorkspaceToolConfig, cfgPath string) {
	metas := r.AllToolMeta()
	needWrite := false
	for _, m := range metas {
		if _, ok := cfg.Tools[m.Name]; !ok {
			e := m.Enabled
			cfg.Tools[m.Name] = ToolConfigItem{Enabled: &e}
			needWrite = true
			log.Printf("[WorkspaceToolConfig] 补充缺失工具 %s -> %v", m.Name, e)
		}
	}
	if !needWrite {
		return
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Printf("[WorkspaceToolConfig] 补充写入序列化失败: %v", err)
		return
	}
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		log.Printf("[WorkspaceToolConfig] 补充写入失败 %s: %v", cfgPath, err)
		return
	}
	log.Printf("[WorkspaceToolConfig] 已补充 %s（共 %d 个工具）", cfgPath, len(cfg.Tools))
}

// initWorkspaceToolConfig 自动初始化 .pair/tools.json，包含注册表中全部工具的当前 Enabled 状态。
func initWorkspaceToolConfig(r *Registry, cfgPath string) {
	metas := r.AllToolMeta()
	if len(metas) == 0 {
		return
	}
	cfg := WorkspaceToolConfig{
		Tools: make(map[string]ToolConfigItem, len(metas)),
	}
	for _, m := range metas {
		e := m.Enabled
		cfg.Tools[m.Name] = ToolConfigItem{Enabled: &e}
	}
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("[WorkspaceToolConfig] 创建目录失败 %s: %v", dir, err)
		return
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Printf("[WorkspaceToolConfig] 序列化失败: %v", err)
		return
	}
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		log.Printf("[WorkspaceToolConfig] 写入失败 %s: %v", cfgPath, err)
		return
	}
	log.Printf("[WorkspaceToolConfig] 已自动初始化 %s（%d 个工具）", cfgPath, len(metas))
}
