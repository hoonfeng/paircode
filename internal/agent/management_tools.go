// Agent 自管理工具 —— 让 Agent 自己检索/读取/安装/修改/删除 Skills 与 MCP 服务器。
// 从 cmd/companion/agenttools/tools.go 迁移，全部使用 agent 内部 API。
// 所有写类工具均设置 RequiresApproval=true，由 Registry.BeforeTool 统一审批。
package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/pkg/memory"
)

// ─── JSON Schema 辅助函数（m_ 前缀避免与 tools.go 重名）──

func mStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func mObjSchema(props map[string]any, required ...string) map[string]any {
	req := required
	if req == nil {
		req = []string{}
	}
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func mStrProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func mArgStr(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func modeLabel(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "auto":
		return "按需"
	case "always":
		return "始终"
	case "manual":
		return "手动"
	}
	return "按需"
}

// RegisterManagementTools 注册 Agent 自管理工具。
// root 为工作区根路径，每个会话传自己的实现多工作区隔离。
func RegisterManagementTools(r *Registry, root string) {
	skillProjectDir := ""
	if root != "" {
		skillProjectDir = filepath.Join(root, ".pair", "skills")
	}

	// ── Skills ──
	r.Register(&Tool{
		Name: "skill_list", Description: "列出所有可用技能（名/描述/激活模式/层级）。", ReadOnly: true,
		Parameters: mObjSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			return listSkillsText(root), nil
		},
	})
	r.Register(&Tool{
		Name: "load_skill",
		Description: "加载某技能的完整 SKILL.md 正文（L2 渐进式披露）。",
		ReadOnly: true,
		Parameters: mObjSchema(map[string]any{"name": mStrProp("技能名")}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return loadSkillFull(mArgStr(args, "name"), root)
		},
	})
	r.Register(&Tool{
		Name: "load_skill_resource",
		Description: "加载某技能的子资源文件（L3 渐进式披露）。",
		ReadOnly: true,
		Parameters: mObjSchema(map[string]any{
			"name": mStrProp("技能名"), "path": mStrProp("资源相对路径"),
		}, "name", "path"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return loadSkillResource(mArgStr(args, "name"), mArgStr(args, "path"), root)
		},
	})
	r.Register(&Tool{
		Name: "skill_write",
		Description: "创建或更新一个技能（写入 .pair/skills/<名>/SKILL.md）。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"name": mStrProp("技能名"), "description": mStrProp("一句话描述"),
			"mode": mStrProp("激活模式：auto/always/manual，默认 auto"),
			"content": mStrProp("技能正文"),
		}, "name", "content"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return writeSkillTool(args, skillProjectDir)
		},
	})
	r.Register(&Tool{
		Name: "skill_delete", Description: "删除一个项目级技能。", RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{"name": mStrProp("技能名")}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			if err := DeleteSkill(skillProjectDir, mArgStr(args, "name")); err != nil {
				return "", err
			}
			return "已删除技能：" + mArgStr(args, "name"), nil
		},
	})

	// ── MCP ──
	r.Register(&Tool{
		Name: "mcp_list", Description: "列出已配置的 MCP 服务器。", ReadOnly: true,
		Parameters: mObjSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) { return listMCPText(), nil },
	})
	r.Register(&Tool{
		Name: "mcp_add",
		Description: "新增一个 MCP 服务器。scope 可选 user 或 project。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"name": mStrProp("服务器名"), "command": mStrProp("启动命令"),
			"args": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"scope": mStrProp("user/project"),
		}, "name", "command"),
		Handler: func(_ context.Context, args map[string]any) (string, error) { return mcpAddTool(args) },
	})
	r.Register(&Tool{
		Name: "mcp_remove", Description: "删除一个 MCP 服务器。", RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{"name": mStrProp("服务器名")}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			if err := MCPDelete(MCPLevelUser, mArgStr(args, "name")); err != nil {
				return "", err
			}
			return "已删除 MCP 服务器：" + mArgStr(args, "name"), nil
		},
	})

	// ── 市场 ──
	r.Register(&Tool{
		Name: "marketplace_search", Description: "在市场精选注册表里检索可安装的服务器与技能。",
		ReadOnly: true,
		Parameters: mObjSchema(map[string]any{
			"query": mStrProp("关键词"), "kind": mStrProp("mcp/skill/all"),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return marketSearchText(mArgStr(args, "query"), mArgStr(args, "kind")), nil
		},
	})
	r.Register(&Tool{
		Name: "marketplace_install",
		Description: "从市场按 id 安装一个 MCP 或技能。scope 可选 user/project。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"id": mStrProp("条目 id"), "scope": mStrProp("user/project"),
		}, "id"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			scope := mArgStr(args, "scope")
			if scope == "" {
				scope = "user"
			}
			return MarketInstallScoped(mArgStr(args, "id"), true, scope)
		},
	})

	// ── 已完成对话历史 ──
	r.Register(&Tool{
		Name: "history_search", Description: "按关键词搜索已完成对话的历史记录（标题/摘要/标签/关键点）。", ReadOnly: true,
		Parameters: mObjSchema(map[string]any{"query": mStrProp("搜索关键词")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return searchHistoryText(mArgStr(args, "query")), nil
		},
	})
	r.Register(&Tool{
		Name: "history_list", Description: "列出所有已完成对话的历史记录（按完成时间倒序）。",
		ReadOnly: true, Parameters: mObjSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) { return listHistoryText(), nil },
	})
	r.Register(&Tool{
		Name: "history_count", Description: "查询已完成对话的历史记录总数。",
		ReadOnly: true, Parameters: mObjSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) { return historyCountText(), nil },
	})
}

// ─── Skills 工具实现 ──

func listSkillsText(root string) string {
	skills := LoadAllSkillsFromRoot(root, SkillSystemDir, SkillEnabled)
	var b strings.Builder
	for _, s := range skills {
		lvl := "工作区级"
		if s.Level == LevelSystem {
			lvl = "内置"
		}
		fmt.Fprintf(&b, "- [%s] %s（%s）：%s\n", lvl, s.Name, modeLabel(mStr(s.Mode, "auto")), s.Description)
	}
	if b.Len() == 0 {
		return "（暂无技能）"
	}
	return b.String()
}

func loadSkillFull(name string, root string) (string, error) {
	skills := LoadAllSkillsFromRoot(root, SkillSystemDir, SkillEnabled)
	s := FindSkill(skills, name)
	if s == nil {
		return "", fmt.Errorf("未找到技能 %q", name)
	}
	return "# 技能：" + s.Name + "\n" + s.Description + "\n\n" + SkillBodyWithTools(*s), nil
}

func loadSkillResource(name, path string, root string) (string, error) {
	skills := LoadAllSkillsFromRoot(root, SkillSystemDir, SkillEnabled)
	s := FindSkill(skills, name)
	if s == nil {
		return "", fmt.Errorf("未找到技能 %q", name)
	}
	return LoadSkillResource(s, path, 10*1024*1024)
}

func writeSkillTool(args map[string]any, projectDir string) (string, error) {
	s := Skill{
		Name: mArgStr(args, "name"), Description: mArgStr(args, "description"),
		Mode: mStr(mArgStr(args, "mode"), "auto"), Body: mArgStr(args, "content"),
	}
	if s.Name == "" || s.Body == "" {
		return "", fmt.Errorf("name 与 content 必填")
	}
	if err := WriteSkill(projectDir, s); err != nil {
		return "", err
	}
	return "已写入技能 " + s.Name, nil
}

// ─── MCP 工具实现 ──

func listMCPText() string {
	var b strings.Builder
	for _, lv := range MCPLevels {
		for _, e := range MCPReadLevel(lv.ID) {
			on := "禁用"
			if MCPEnabled(lv.ID, e.Name) {
				on = "启用"
			}
			fmt.Fprintf(&b, "- [%s] %s（%s）：%s %s\n", lv.Name, e.Name, on, e.Command, strings.Join(e.Args, " "))
		}
	}
	return b.String()
}

func mcpAddTool(args map[string]any) (string, error) {
	e := MCPEntry{Name: mArgStr(args, "name"), Command: mArgStr(args, "command")}
	if arr, ok := args["args"].([]any); ok {
		for _, a := range arr {
			if s, ok := a.(string); ok {
				e.Args = append(e.Args, s)
			}
		}
	}
	if e.Name == "" || e.Command == "" {
		return "", fmt.Errorf("name 与 command 必填")
	}
	scope := mArgStr(args, "scope")
	var level MCPLevel
	var levelLabel string
	switch scope {
	case "project":
		level = MCPLevelProject; levelLabel = "工作区级"
	default:
		level = MCPLevelUser; levelLabel = "用户级（全局）"
	}
	if err := MCPUpsert(level, e); err != nil {
		return "", err
	}
	return "已添加 MCP 服务器 " + e.Name + "（" + levelLabel + "）", nil
}

// ─── 市场工具实现 ──

func marketSearchText(query, kind string) string {
	results := MarketSearch(query, kind)
	if len(results) == 0 {
		return "未找到匹配的市场条目。用 marketplace_install <id> 安装。"
	}
	var b strings.Builder
	for _, e := range results {
		installed := ""
		if MarketIsInstalled(e.ID) {
			installed = " [已安装]"
		}
		fmt.Fprintf(&b, "- [%s] %s（%s）：%s%s\n", e.Kind, e.Name, e.ID, e.Description, installed)
	}
	fmt.Fprintf(&b, "\n共 %d 个条目。用 marketplace_install <id> 安装。", len(results))
	return b.String()
}

// ─── 已完成对话历史工具实现 ──

func searchHistoryText(query string) string {
	results := memory.Search(query)
	if len(results) == 0 {
		return "未找到匹配的历史记忆。"
	}
	var b strings.Builder
	if query != "" {
		fmt.Fprintf(&b, "搜索「%s」共找到 %d 条历史记忆：\n\n", query, len(results))
	} else {
		fmt.Fprintf(&b, "共 %d 条历史记忆：\n\n", len(results))
	}
	for _, m := range results {
		title := m.Title
		if title == "" {
			title = "（未命名）"
		}
		summary := m.Summary
		if len([]rune(summary)) > 200 {
			summary = string([]rune(summary)[:200]) + "…"
		}
		tags := ""
		if len(m.Tags) > 0 {
			tags = " 标签:" + strings.Join(m.Tags, ",")
		}
		fmt.Fprintf(&b, "- 「%s」%d条消息%s\n", title, m.MessageCount, tags)
		if summary != "" {
			fmt.Fprintf(&b, "  %s\n", summary)
		}
		fmt.Fprintf(&b, "  完成时间: %s\n\n", m.CompletedAt)
	}
	if query != "" {
		b.WriteString("（需要更精确的搜索请用更具体的关键词。）")
	}
	return b.String()
}

func listHistoryText() string {
	results := memory.List()
	if len(results) == 0 {
		return "（暂无已完成对话的历史记录。）"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "共 %d 条已完成对话的历史记录：\n\n", len(results))
	for _, m := range results {
		title := m.Title
		if title == "" {
			title = "（未命名）"
		}
		summary := m.Summary
		if len([]rune(summary)) > 150 {
			summary = string([]rune(summary)[:150]) + "…"
		}
		tags := ""
		if len(m.Tags) > 0 {
			tags = " [" + strings.Join(m.Tags, ", ") + "]"
		}
		fmt.Fprintf(&b, "%d. 「%s」%s (%d条消息, %s)\n  %s\n\n",
			len(results)-historyFindIdx(m, results)+1, title, tags, m.MessageCount, m.CompletedAt, summary)
	}
	b.WriteString("（需要详情可用 history_search 搜索关键词。）")
	return b.String()
}

func historyFindIdx(target memory.Entry, results []memory.Entry) int {
	for i, m := range results {
		if m.ID == target.ID {
			return i
		}
	}
	return -1
}

func historyCountText() string {
	count := memory.Count()
	if count == 0 {
		return "当前没有已完成对话的历史记录。"
	}
	return fmt.Sprintf("已完成对话历史记录索引中共有 %d 条记录。", count)
}