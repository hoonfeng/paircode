// 市场安装逻辑 —— 自闭环：从内置注册表安装 MCP 服务器和技能到本地配置。
// 从 cmd/companion/ui/marketplace/marketplace.go 迁移而来，去掉对外部的 mcppanel/uiapi 依赖，
// 全部使用 agent 内部 API（MCPUpsert、MCPReadLevel、WriteSkill、LoadAllSkills、MarketFind）。
// 无 //go:build 标签，全平台可用。

package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ─── 安装 ───

// MarketInstallScoped 从市场按 id 安装一个 MCP 服务器或技能。
// 使用 MarketFind 查找内置条目。
// scope 可选 "user"（默认，用户级/全局）或 "project"（项目级/工作区）。
func MarketInstallScoped(id string, auto bool, scope ...string) (string, error) {
	s := "user"
	if len(scope) > 0 && scope[0] != "" {
		s = scope[0]
	}
	entry := MarketFind(id)
	if entry == nil {
		return "", fmt.Errorf("市场未找到条目 %q", id)
	}
	return MarketInstallEntry(*entry, auto, s)
}

// MarketInstallEntry 直接从 MarketEntry 安装 MCP、技能或插件（不查注册表）。
// 适用于前端搜索结果已包含 command/args 的场景。
// scope 可选 "user"（默认）或 "project"。
func MarketInstallEntry(entry MarketEntry, auto bool, scope ...string) (string, error) {
	s := "user"
	if len(scope) > 0 && scope[0] != "" {
		s = scope[0]
	}
	switch entry.Kind {
	case "mcp":
		return marketInstallMCP(entry, auto, s)
	case "skill":
		return marketInstallSkill(entry, auto)
	case "plugin":
		// npm/cordis 插件（Source=npm:<pkg>）走 npm 市场安装；否则为内置工具集
		if strings.HasPrefix(entry.Source, "npm:") {
			return marketInstallNPMPlugin(entry, auto)
		}
		return marketInstallPlugin(entry, auto, s)
	default:
		return "", fmt.Errorf("未知条目类型: %s", entry.Kind)
	}
}

// marketInstallPlugin 安装插件/工具集：Content 为 toolset 发布 JSON
// （toolset_export 格式：{kind:"toolset", toolset:{...}}）。
// scope=project → 固化到工作区 .pair/toolsets/；user → 全局。立即装载（全局宿主）。
func marketInstallPlugin(entry MarketEntry, auto bool, scope string) (string, error) {
	var pub ToolsetPublish
	if err := json.Unmarshal([]byte(entry.Content), &pub); err != nil {
		return "", fmt.Errorf("插件条目 %s 内容不是有效发布 JSON: %w", entry.ID, err)
	}
	ts := &pub.Toolset
	if ts.Name == "" || len(ts.Plugins) == 0 {
		return "", fmt.Errorf("插件条目 %s 缺 name/plugins", entry.ID)
	}
	// 目标作用域
	projectRoot := ""
	if scope == "project" {
		if ph := GetGlobalPluginHost(); ph != nil && ph.Context() != nil && ph.Context().WorkspaceRoot != "" {
			projectRoot = ph.Context().WorkspaceRoot
		} else {
			projectRoot = primaryWorkspaceRoot()
		}
	}
	tsScope := toolsetProject // ★ 工具集仅工作区级（没有全局工具集）
	if err := saveToolset(projectRoot, tsScope, ts); err != nil {
		return "", fmt.Errorf("固化工具集失败: %w", err)
	}
	// 立即装载（全局宿主存在时）；失败回滚固化文件，避免状态不一致
	if ph := GetGlobalPluginHost(); ph != nil {
		if err := installToolset(ph, ts); err != nil {
			_ = removeToolset(projectRoot, tsScope, ts.Name)
			return "", fmt.Errorf("工具集装载失败已回滚（未固化）: %w", err)
		}
	}
	level := "全局"
	if scope == "project" {
		level = "工作区"
	}
	msg := fmt.Sprintf("✅ 已安装插件工具集「%s」（%s，%d 个插件）", ts.Name, level, len(ts.Plugins))
	if !auto {
		msg += "。已立即装载可用。"
	}
	return msg, nil
}

// primaryWorkspaceRoot 取主工作区根（WorkspaceRoots[0] 或全局根）。
func primaryWorkspaceRoot() string {
	if len(WorkspaceRoots) > 0 {
		return WorkspaceRoots[0]
	}
	return ""
}

// marketInstallMCP 内部安装 MCP 服务器到指定层级配置。
func marketInstallMCP(entry MarketEntry, auto bool, scope string) (string, error) {
	e := MCPEntry{
		Name:    entry.ID,
		Command: entry.Command,
		Args:    entry.Args,
	}
	if e.Name == "" {
		e.Name = entry.ID
	}
	if e.Command == "" {
		e.Command = "npx"
	}
	var level MCPLevel
	var levelLabel string
	switch scope {
	case "project":
		level = MCPLevelProject
		levelLabel = "工作区级"
	default:
		level = MCPLevelUser
		levelLabel = "用户级（全局）"
	}
	if err := MCPUpsert(level, e); err != nil {
		return "", fmt.Errorf("写入 MCP 配置失败: %w", err)
	}
	msg := fmt.Sprintf("✅ 已安装 MCP 服务器「%s」（%s）", entry.Name, levelLabel)
	if auto {
		msg += "。下次对话连接生效。"
	}
	return msg, nil
}

// marketInstallSkill 内部安装技能到项目级 skills 目录。
func marketInstallSkill(entry MarketEntry, auto bool) (string, error) {
	s := Skill{
		Name:        entry.ID,
		Description: entry.Description,
		Mode:        entry.Activation,
		Body:        entry.Content,
	}
	if s.Mode == "" {
		s.Mode = "auto"
	}
	if err := WriteSkill(SkillProjectDir, s); err != nil {
		return "", fmt.Errorf("写入技能失败: %w", err)
	}
	msg := fmt.Sprintf("✅ 已安装技能「%s」（工作区级 .pair/skills）", entry.Name)
	if auto {
		msg += "。下次对话注入 system prompt。"
	}
	return msg, nil
}

// ─── 已安装状态 ───

// MarketIsInstalled 检查某条目是否已安装。
func MarketIsInstalled(id string) bool {
	entry := MarketFind(id)
	if entry == nil {
		// 不在内置注册表 → 可能是 npm 插件（ID=包名）
		return npmPluginInstalled(id)
	}
	switch entry.Kind {
	case "mcp":
		for _, e := range MCPReadLevel(MCPLevelUser) {
			if e.Name == id {
				return true
			}
		}
		for _, e := range MCPReadLevel(MCPLevelProject) {
			if e.Name == id {
				return true
			}
		}
		return false
	case "skill":
		for _, s := range LoadAllSkills() {
			if s.Name == id {
				return true
			}
		}
		return false
	case "plugin":
		if strings.HasPrefix(entry.Source, "npm:") {
			return npmPluginInstalled(id)
		}
		// 内置工具集：ID = "plugin-" + 工具集名
		name := strings.TrimPrefix(id, "plugin-")
		projectRoot := primaryWorkspaceRoot()
		for _, ts := range listToolsets(projectRoot, toolsetProject) {
			if ts.Name == name {
				return true
			}
		}
		return false
	}
	return false
}

// MarketUninstall 按 kind 卸载市场条目（MCP/技能/插件工具集）。
// npm 插件请用 UninstallNPMPlugin（或直接调 MarketUninstall 传 source 前缀由调用方分发）。
func MarketUninstall(id, kind string) (string, error) {
	switch kind {
	case "mcp":
		err := MCPDelete(MCPLevelUser, id)
		if err != nil && !strings.Contains(err.Error(), "not exist") {
			return "", err
		}
		_ = MCPDelete(MCPLevelProject, id) // 项目级一并清理（容错）
		return "已卸载 MCP 服务器 " + id, nil
	case "skill":
		if err := DeleteSkill(SkillProjectDir, id); err != nil {
			return "", err
		}
		return "已卸载技能 " + id, nil
	case "plugin":
		// 工具集：ID = "plugin-" + 工具集名（★ 仅工作区级）
		name := strings.TrimPrefix(id, "plugin-")
		projectRoot := primaryWorkspaceRoot()
		if err := removeToolset(projectRoot, toolsetProject, name); err == nil {
			return fmt.Sprintf("已卸载工具集「%s」（工作区）", name), nil
		}
		return "", fmt.Errorf("工具集 %s 未找到", name)
	}
	return "", fmt.Errorf("未知类型 %s", kind)
}

// UninstallNPMPlugin 卸载 npm 插件（导出，供 web_server /marketplace/uninstall 用）。
func UninstallNPMPlugin(pkg string) error {
	return uninstallNPMPlugin(pkg)
}
