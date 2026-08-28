package agent

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/hoonfeng/paircode/internal/core"
)

// WorkspaceToolConfig 工作区审核配置文件结构（★ 2026-09 t1 报告 G1 缺口闭环：
// 审核配置已统一到 settings.json 顶层（core.Settings），本结构仅用于
// .pair/tools.json 遗留文件的**一次性迁移**读取）。
// 路径：.pair/tools.json
// 格式：
//
//	{
//	  "reviewMode": "auto",
//	  "reviewBlacklist": ["delete_file"],
//	  "reviewWhitelist": ["read"]
//	}
//
// ★ 旧版 tools 工具开关字段（Tools/ToolConfigItem）已随「工具集（插件化）」机制
// 废弃删除：工具开关由 .pair/toolsets/*.json 的 DisabledTools 管理，
// 不再从 tools.json 读写 enabled。删除记录见 2026-08-16 提交。
type WorkspaceToolConfig struct {
	ReviewMode      string   `json:"reviewMode,omitempty"` // auto/manual/off
	ReviewBlacklist []string `json:"reviewBlacklist,omitempty"`
	ReviewWhitelist []string `json:"reviewWhitelist,omitempty"`
}

// WorkspaceToolConfigPath 返回遗留工作区审核配置文件（.pair/tools.json）的路径。
// ★ 仅迁移用：新配置写入 settings.json（core.Settings 顶层）。
func WorkspaceToolConfigPath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".pair", "tools.json")
}

// migrateLegacyReviewConfig 一次性迁移 .pair/tools.json 的审核字段 → settings.json：
//   - tools.json 存在且含审核字段（非默认值）→ 合并进 core.Settings（settings 同字段
//     为空/默认时采用），core.Save() 落盘；
//   - 迁移成功后删除 tools.json（仅含审核字段时整删；含未知字段时仅删审核字段保留文件）。
//
// ★ t1 G1 闭环：审核配置单一来源 = settings.json 顶层（UI 设置面板与 /api/tools/review
//
//	同写一处），tools.json 不再双源分叉。
func migrateLegacyReviewConfig(workspaceRoot string) {
	if workspaceRoot == "" {
		return
	}
	path := WorkspaceToolConfigPath(workspaceRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		return // 无遗留文件
	}
	var cfg WorkspaceToolConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return // 坏文件不迁移（保留原样，不阻塞）
	}
	// 未知字段检测：仅当文件含审核字段之外的 key 时保留文件（删审核字段）
	var raw map[string]json.RawMessage
	hasUnknown := false
	if json.Unmarshal(data, &raw) == nil {
		for k := range raw {
			switch k {
			case "reviewMode", "reviewBlacklist", "reviewWhitelist":
			default:
				hasUnknown = true
			}
		}
	}
	// 合并：仅在 settings 同字段为空/默认时采用遗留值——settings.json 已是
	// 用户显式配置（reviewMode 非默认 auto、列表非空）时以 settings 为准，
	// 遗留 tools.json 只补「从未配置过」的字段（一次性；迁移后 tools.json
	// 删除，settings.json 成为唯一来源，此后用户经设置面板/API 修改只写 settings）
	changed := false
	if cfg.ReviewMode != "" && cfg.ReviewMode != "auto" &&
		(core.Settings.ReviewMode == "" || core.Settings.ReviewMode == "auto") {
		core.Settings.ReviewMode = cfg.ReviewMode
		changed = true
	}
	if len(cfg.ReviewBlacklist) > 0 && len(core.Settings.ReviewBlacklist) == 0 {
		core.Settings.ReviewBlacklist = append([]string(nil), cfg.ReviewBlacklist...)
		changed = true
	}
	if len(cfg.ReviewWhitelist) > 0 && len(core.Settings.ReviewWhitelist) == 0 {
		core.Settings.ReviewWhitelist = append([]string(nil), cfg.ReviewWhitelist...)
		changed = true
	}
	if changed {
		core.Save()
	}
	// 删除/清理遗留文件
	if hasUnknown {
		// 含未知字段：删掉审核字段，保留文件其余内容
		if clean, err := json.MarshalIndent(cleanReviewConfigKeys(raw), "", "  "); err == nil {
			_ = os.WriteFile(path, clean, 0o644)
		}
	} else {
		if err := os.Remove(path); err == nil {
			log.Printf("[toolconfig] 已迁移 .pair/tools.json 审核配置 → settings.json（遗留文件删除）")
		}
	}
}

// cleanReviewConfigKeys 从原始 JSON 剔除审核字段（保留未知字段）。
func cleanReviewConfigKeys(raw map[string]json.RawMessage) map[string]any {
	out := map[string]any{}
	for k, v := range raw {
		switch k {
		case "reviewMode", "reviewBlacklist", "reviewWhitelist":
			continue
		}
		var val any
		if json.Unmarshal(v, &val) == nil {
			out[k] = val
		}
	}
	return out
}

// LoadWorkspaceReviewConfig 读取审核配置（reviewMode / reviewBlacklist / reviewWhitelist）。
// ★ t1 G1 闭环：单一来源 = settings.json 顶层（core.Settings）——与 UI 设置面板
//
//	经 /api/settings 写入的同一存储；.pair/tools.json 仅作一次性迁移遗留（见
//	migrateLegacyReviewConfig）。返回 settings 值（缺省 auto / nil 列表）。
func LoadWorkspaceReviewConfig(workspaceRoot string) (mode string, blacklist, whitelist []string) {
	migrateLegacyReviewConfig(workspaceRoot)
	mode = core.Settings.ReviewMode
	if mode == "" {
		mode = "auto"
	}
	blacklist = core.Settings.ReviewBlacklist
	whitelist = core.Settings.ReviewWhitelist
	return mode, blacklist, whitelist
}

// SaveWorkspaceReviewConfig 保存审核配置。
// ★ t1 G1 闭环：写入 settings.json 顶层（core.Settings + core.Save()）——与
//
//	/api/settings 同存储，双源分叉消除；.pair/tools.json 遗留文件一并清理。
func SaveWorkspaceReviewConfig(workspaceRoot string, mode string, blacklist, whitelist []string) error {
	if mode == "" {
		mode = "auto"
	}
	core.Settings.ReviewMode = mode
	core.Settings.ReviewBlacklist = append([]string(nil), blacklist...)
	core.Settings.ReviewWhitelist = append([]string(nil), whitelist...)
	core.Save()
	// 清理遗留 tools.json（迁移完成，双源归一）
	if workspaceRoot != "" {
		_ = os.Remove(WorkspaceToolConfigPath(workspaceRoot))
	}
	return nil
}
