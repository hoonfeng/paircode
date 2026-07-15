// 市场安装逻辑 —— 自闭环：从内置注册表安装 MCP 服务器和技能到本地配置。
// 从 cmd/companion/ui/marketplace/marketplace.go 迁移而来，去掉对外部的 mcppanel/uiapi 依赖，
// 全部使用 agent 内部 API（MCPUpsert、MCPReadLevel、WriteSkill、LoadAllSkills、MarketFind）。
// 无 //go:build 标签，全平台可用。

package agent

import (
	"fmt"
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

// MarketInstallEntry 直接从 MarketEntry 安装 MCP 或技能（不查注册表）。
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
	default:
		return "", fmt.Errorf("未知条目类型: %s", entry.Kind)
	}
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
		return false
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
	}
	return false
}
