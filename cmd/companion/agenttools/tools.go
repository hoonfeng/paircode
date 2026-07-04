// Agent 自管理工具 —— 让 Agent 自己 检索/读取/安装/修改/删除 Skills 与 MCP 服务器，含市场(精选注册表)。
// 复刻参考 marketplace.ts(精选注册表) + auto-install.ts(Agent 自动安装)。写类工具默认需审批。
// 渐进式披露：skill_read 让 Agent 按需读技能全文（L2，配合 skillsPrompt 的 L1 名+描述）。
//
//go:build windows


package agenttools

import (
	"context"
	"fmt"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	mcppanel "github.com/hoonfeng/paircode/cmd/companion/ui/mcp"
	skillspanel "github.com/hoonfeng/paircode/cmd/companion/ui/skills"
	marketplacepanel "github.com/hoonfeng/paircode/cmd/companion/ui/marketplace"
)

// ─── JSON Schema 小助手（agent 包的 objSchema 未导出，这里自建）──
func orStr(s, def string) string { if s == "" { return def }; return s }

func toolObj(props map[string]any, required ...string) map[string]any {
	req := required
	if req == nil {
		req = []string{} // 必须用空数组而非 nil，否则 JSON 序列化为 null 违反 OpenAI schema 约束
	}
	return map[string]any{"type": "object", "properties": props, "required": req}
}
func strParam(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}
func argString(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// ─── 市场注册表（精选，复刻参考 MCP_REGISTRY；包名均为真实 npm/uvx 包）──


// marketSkill 市场技能（精选，含可直接落地的 SKILL.md 正文）。


// registerManagementTools 注册 Agent 自管理工具（在 bridge 建 registry 时调用）。
func RegisterManagementTools(r *agent.Registry) {
	// ── Skills：检索 / 读全文(渐进式披露 L2) / 写 / 删 ──
	r.Register(&agent.Tool{
		Name: "skill_list", Description: "列出所有可用技能（名/描述/激活模式/层级）。", ReadOnly: true,
		Parameters: toolObj(map[string]any{}),
		Handler:    func(_ context.Context, _ map[string]any) (string, error) { return listSkillsText(), nil },
	})
	r.Register(&agent.Tool{
		Name:        "load_skill",
		Description: "加载某技能的完整 SKILL.md 正文（L2 渐进式披露：系统提示只给名+描述，需要细则时用本工具按需加载全文）。",
		ReadOnly:    true,
		Parameters:  toolObj(map[string]any{"name": strParam("技能名")}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return loadSkillFull(argString(args, "name"))
		},
	})
	r.Register(&agent.Tool{
		Name:        "load_skill_resource",
		Description: "加载某技能的子资源文件（L3 渐进式披露）：references/ assets/ scripts/ 下的文件，单文件上限 10MiB。路径穿越被拒。",
		ReadOnly:    true,
		Parameters: toolObj(map[string]any{
			"name": strParam("技能名"), "path": strParam("资源相对路径（如 references/faq.md）"),
		}, "name", "path"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return loadSkillResource(argString(args, "name"), argString(args, "path"))
		},
	})
	r.Register(&agent.Tool{
		Name:             "skill_write",
		Description:      "创建或更新一个技能（写入项目级 .pair/skills/<名>/SKILL.md）。用于把可复用的工作方式沉淀成技能。",
		RequiresApproval: true,
		Parameters: toolObj(map[string]any{
			"name": strParam("技能名"), "description": strParam("一句话描述"),
			"mode":    strParam("激活模式：auto(按需)/always(始终)/manual(仅手动)，默认 auto"),
			"content": strParam("技能正文（Markdown，写清何时用、怎么做）"),
		}, "name", "content"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return writeSkillTool(args)
		},
	})
	r.Register(&agent.Tool{
		Name: "skill_delete", Description: "删除一个项目级技能。", RequiresApproval: true,
		Parameters: toolObj(map[string]any{"name": strParam("技能名")}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			if err := agent.DeleteSkill(agent.SkillProjectDir, argString(args, "name")); err != nil {
				return "", err
			}
			return "已删除技能：" + argString(args, "name"), nil
		},
	})

	// ── MCP：检索 / 增 / 删 ──
	r.Register(&agent.Tool{
		Name: "mcp_list", Description: "列出已配置的 MCP 服务器（名/命令/启用态/层级）。", ReadOnly: true,
		Parameters: toolObj(map[string]any{}),
		Handler:    func(_ context.Context, _ map[string]any) (string, error) { return listMCPText(), nil },
	})
	r.Register(&agent.Tool{
		Name:             "mcp_add",
		Description:      "新增一个 MCP 服务器到用户级配置（mcp.json）。下次对话连接生效。",
		RequiresApproval: true,
		Parameters: toolObj(map[string]any{
			"name": strParam("服务器名（工具前缀）"), "command": strParam("启动命令，如 npx / uvx"),
			"args": map[string]any{"type": "array", "description": "命令参数", "items": map[string]any{"type": "string"}},
		}, "name", "command"),
		Handler: func(_ context.Context, args map[string]any) (string, error) { return mcpAddTool(args) },
	})
	r.Register(&agent.Tool{
		Name: "mcp_remove", Description: "从用户级配置删除一个 MCP 服务器。", RequiresApproval: true,
		Parameters: toolObj(map[string]any{"name": strParam("服务器名")}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			if err := mcppanel.Delete(mcppanel.LevelUser, argString(args, "name")); err != nil {
				return "", err
			}
			return "已删除 MCP 服务器：" + argString(args, "name"), nil
		},
	})

	// ── 市场：检索 / 安装 ──
	r.Register(&agent.Tool{
		Name:        "marketplace_search",
		Description: "在 MCP/Skills 市场（精选注册表）里检索可安装的服务器与技能。",
		ReadOnly:    true,
		Parameters:  toolObj(map[string]any{"query": strParam("关键词（留空=列全部）"), "kind": strParam("mcp / skill / all，默认 all")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return MarketSearch(argString(args, "query"), argString(args, "kind")), nil
		},
	})
	r.Register(&agent.Tool{
		Name:             "marketplace_install",
		Description:      "从市场按 id 安装一个 MCP 服务器（写入用户级 mcp.json）或技能（写入项目级 .pair/skills）。",
		RequiresApproval: true,
		Parameters:       toolObj(map[string]any{"id": strParam("市场条目 id（见 marketplace_search）")}, "id"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return marketplacepanel.InstallScoped(argString(args, "id"), true)
		},
	})
}

// ─── Skills 工具实现 ──
func listSkillsText() string {
	skills := agent.LoadAllSkills()
	var b strings.Builder
	for _, s := range skills {
		lvl := "项目级"
		if s.Level == agent.LevelSystem {
			lvl = "内置"
		}
		fmt.Fprintf(&b, "- [%s] %s（%s）：%s\n", lvl, s.Name, skillspanel.ModeLabel(orStr(s.Mode, "auto")), s.Description)
	}
	if b.Len() == 0 {
		return "（暂无技能。可用 skill_write 创建或 marketplace_search 从市场安装。）"
	}
	return b.String()
}

func loadSkillFull(name string) (string, error) {
	skills := agent.LoadAllSkills()
	s := agent.FindSkill(skills, name)
	if s == nil {
		return "", fmt.Errorf("未找到技能 %q（用 skill_list 看全部）", name)
	}
	return "# 技能：" + s.Name + "\n" + s.Description + "\n\n" + agent.SkillBodyWithTools(*s), nil
}

func loadSkillResource(name, path string) (string, error) {
	skills := agent.LoadAllSkills()
	s := agent.FindSkill(skills, name)
	if s == nil {
		return "", fmt.Errorf("未找到技能 %q（用 skill_list 看全部）", name)
	}
	return agent.LoadSkillResource(s, path, 10*1024*1024) // 10MiB 上限
}

func writeSkillTool(args map[string]any) (string, error) {
	s := agent.Skill{
		Name:        argString(args, "name"),
		Description: argString(args, "description"),
		Mode:        orStr(argString(args, "mode"), "auto"),
		Body:        argString(args, "content"),
	}
	if err := agent.WriteSkill(agent.SkillProjectDir, s); err != nil {
		return "", err
	}
	return "已写入技能 " + s.Name + "（项目级，下次对话注入系统提示，或现在用 load_skill 取用）", nil
}

// ─── MCP 工具实现 ──
func listMCPText() string {
	var b strings.Builder
	for _, lv := range mcppanel.Levels {
		for _, e := range mcppanel.ReadLevel(lv.ID) {
			on := "禁用"
			if mcppanel.Enabled(lv.ID, e.Name) {
				on = "启用"
			}
			fmt.Fprintf(&b, "- [%s] %s（%s）：%s %s\n", lv.Name, e.Name, on, e.Command, strings.Join(e.Args, " "))
		}
	}
	return b.String()
}

func mcpAddTool(args map[string]any) (string, error) {
	e := mcppanel.Entry{Name: argString(args, "name"), Command: argString(args, "command")}
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
	if err := mcppanel.Upsert(mcppanel.LevelUser, e); err != nil {
		return "", err
	}
	return "已添加 MCP 服务器 " + e.Name + "（用户级，下次对话连接生效）", nil
}
// ─── 市场工具实现 ──
func MarketSearch(query, kind string) string {
	results := marketplacepanel.Search(query, kind)
	if len(results) == 0 {
		return "未找到匹配的市场条目。用 marketplace_install <id> 安装。"
	}
	var b strings.Builder
	for _, e := range results {
		installed := ""
		if marketplacepanel.IsInstalled(e.ID) {
			installed = " [已安装]"
		}
		fmt.Fprintf(&b, "- [%s] %s（%s）：%s%s\n", e.Kind, e.Name, e.ID, e.Description, installed)
	}
	fmt.Fprintf(&b, "\n共 %d 个条目。用 marketplace_install <id> 安装。", len(results))
	return b.String()
}

	func marketInstall(id string) (string, error) {
		return marketplacepanel.InstallScoped(id, true)
	}
