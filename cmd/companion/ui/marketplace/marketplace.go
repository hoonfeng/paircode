// Package marketplace 是 MCP/Skills 市场安装实现。
// 提供 市场搜索/安装 功能（供 agenttools 和 UI 面板共用）。
//
//go:build windows

package marketplace

import (
	"fmt"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	mcppanel "github.com/hoonfeng/paircode/cmd/companion/ui/mcp"
	"github.com/hoonfeng/paircode/cmd/companion/uiapi"
)

// ─── 对外搜索 ───

// SearchText 返回市场搜索的纯文本结果。
func SearchText(query, kind string) string {
	results := Search(query, kind)
	if len(results) == 0 {
		return "未找到匹配的市场条目。"
	}
	var b strings.Builder
	for _, e := range results {
		fmt.Fprintf(&b, "[%s] %s（%s）：%s\n", e.Kind, e.Name, e.ID, e.Description)
	}
	return b.String()
}

// ─── 安装 ───

// InstallScoped 从市场按 id 安装一个 MCP 服务器或技能。
// auto=true 表示由 Agent 自动安装；auto=false 表示 UI 触发。
func InstallScoped(id string, auto bool) (string, error) {
	entry := Find(id)
	if entry == nil {
		return "", fmt.Errorf("市场未找到条目 %q", id)
	}

	switch entry.Kind {
	case "mcp":
		return installMCP(*entry, auto)
	case "skill":
		return installSkill(*entry, auto)
	default:
		return "", fmt.Errorf("未知条目类型: %s", entry.Kind)
	}
}

// InstallAndNotify 安装并发送 UI 通知（供 UI 面板调用）。
func InstallAndNotify(id string) {
	msg, err := InstallScoped(id, false)
	if err != nil {
		uiapi.MessageError("安装失败: " + err.Error())
		return
	}
	uiapi.MessageSuccess(msg)
}

func installMCP(entry RegistryEntry, auto bool) (string, error) {
	e := mcppanel.Entry{
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
	if err := mcppanel.Upsert(mcppanel.LevelUser, e); err != nil {
		return "", fmt.Errorf("写入 MCP 配置失败: %w", err)
	}
	msg := fmt.Sprintf("✅ 已安装 MCP 服务器「%s」（用户级 mcp.json）", entry.Name)
	if auto {
		msg += "。下次对话连接生效。"
	}
	return msg, nil
}

func installSkill(entry RegistryEntry, auto bool) (string, error) {
	s := agent.Skill{
		Name:        entry.ID,
		Description: entry.Description,
		Mode:        entry.Activation,
		Body:        entry.Content,
	}
	if s.Mode == "" {
		s.Mode = "auto"
	}
	if err := agent.WriteSkill(agent.SkillProjectDir, s); err != nil {
		return "", fmt.Errorf("写入技能失败: %w", err)
	}
	msg := fmt.Sprintf("✅ 已安装技能「%s」（项目级 .pair/skills）", entry.Name)
	if auto {
		msg += "。下次对话注入 system prompt。"
	}
	return msg, nil
}

// ─── 已安装状态 ───

// IsInstalled 检查某条目是否已安装。
func IsInstalled(id string) bool {
	entry := Find(id)
	if entry == nil {
		return false
	}
	switch entry.Kind {
	case "mcp":
		for _, e := range mcppanel.ReadLevel(mcppanel.LevelUser) {
			if e.Name == id {
				return true
			}
		}
		for _, e := range mcppanel.ReadLevel(mcppanel.LevelProject) {
			if e.Name == id {
				return true
			}
		}
		return false
	case "skill":
		for _, s := range agent.LoadAllSkills() {
			if s.Name == id {
				return true
			}
		}
		return false
	}
	return false
}
