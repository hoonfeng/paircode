// Agent 自管理工具 —— 让 Agent 自己检索/读取/安装/修改/删除 Skills 与 MCP 服务器。
// 从 cmd/companion/agenttools/tools.go 迁移，全部使用 agent 内部 API。
// 所有写类工具均设置 RequiresApproval=true，由 Registry.BeforeTool 统一审批。
package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/internal/core"
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
// ★ 2026-09-12 修复（重大 BUG）：本函数注册的工具经磁盘插件 tool-system 接管
//   后存档进全局 hostExecutors（启动时一次性存档）——闭包捕获的 root/技能
//   目录随启动冻结：启动时未开工作区 root="" → skill_write 执行
//   WriteSkill("") → filepath.Join("", name) 生成相对路径 → 写到进程
//   CWD（安装目录根）下，与实际使用的 skills 目录脱节；切换工作区也不刷新。
//   现改为执行时运行时解析（会话绑定 _wsRoot 注入 → ctx 会话根 → 工作区
//   实时快照），与 goal/ask_user 路由执行器同构。
func RegisterManagementTools(r *Registry, root string) {
	// ── Skills ──
	r.Register(&Tool{
		Name: "skill_list", Description: "列出所有可用技能（名/描述/激活模式/层级：内置/工作区级/全局）。", ReadOnly: true,
		Parameters: mObjSchema(map[string]any{}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			// ★ args 全量传入（勿用 nil）：jsToolToGo 会注入 _wsRoot（会话根），
			//   skill_list 无参 schema 但 hostTool 透传仍带内部键；传 nil 丢失
			//   会话根 → 列表退化为仅 system 级（工作区技能「找不到」）。
			return listSkillsText(skillRuntimeRoot(ctx, args)), nil
		},
	})
	r.Register(&Tool{
		Name:        "load_skill",
		Description: "加载某技能的完整 SKILL.md 正文（L2 渐进式披露）。所有层级（内置/工作区/全局）同名时工作区优先。",
		ReadOnly:    true,
		Parameters:  mObjSchema(map[string]any{"name": mStrProp("技能名")}, "name"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return loadSkillFull(mArgStr(args, "name"), skillRuntimeRoot(ctx, args))
		},
	})
	r.Register(&Tool{
		Name:        "load_skill_resource",
		Description: "加载某技能的子资源文件（L3 渐进式披露）。",
		ReadOnly:    true,
		Parameters: mObjSchema(map[string]any{
			"name": mStrProp("技能名"), "path": mStrProp("资源相对路径"),
		}, "name", "path"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return loadSkillResource(mArgStr(args, "name"), mArgStr(args, "path"), skillRuntimeRoot(ctx, args))
		},
	})
	r.Register(&Tool{
		Name:             "skill_write",
		Description:      "创建或更新一个技能（目录式 <skills>/名/SKILL.md）。默认写当前工作区级（.pair/skills/）；传 scope=global 写全局（跨工作区生效，随程序安装目录共享）。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"name": mStrProp("技能名"), "description": mStrProp("一句话描述"),
			"mode":    mStrProp("激活模式：auto/always/manual，默认 auto"),
			"content": mStrProp("技能正文"),
			"scope":   mStrProp("层级：project=工作区级（默认，仅当前工作区）/ global=全局（跨工作区，<InstallDir>/.pair/skills/）"),
		}, "name", "content"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return writeSkillTool(ctx, args, root)
		},
	})
	r.Register(&Tool{
		Name:             "skill_delete",
		Description:      "删除一个技能（工作区级默认；scope=global 删全局，scope=system 删内置）。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"name":  mStrProp("技能名"),
			"scope": mStrProp("层级：project=工作区级（默认）/ global=全局 / system=内置"),
		}, "name"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir, label, err := skillTargetDir(ctx, args, root)
			if err != nil {
				return "", err
			}
			if err := DeleteSkill(dir, mArgStr(args, "name")); err != nil {
				return "", err
			}
			return "已删除技能：" + mArgStr(args, "name") + "（" + label + "）", nil
		},
	})

	// ── MCP ──
	r.Register(&Tool{
		Name: "mcp_list", Description: "列出已配置的 MCP 服务器。", ReadOnly: true,
		Parameters: mObjSchema(map[string]any{}),
		Handler:    func(_ context.Context, _ map[string]any) (string, error) { return listMCPText(), nil },
	})
	r.Register(&Tool{
		Name:             "mcp_add",
		Description:      "新增一个 MCP 服务器。scope 可选 user 或 project。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"name": mStrProp("服务器名"), "command": mStrProp("启动命令"),
			"args":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"scope": mStrProp("user/project"),
		}, "name", "command"),
		Handler: func(_ context.Context, args map[string]any) (string, error) { return mcpAddTool(args) },
	})
	r.Register(&Tool{
		Name:             "mcp_remove",
		Description:      "删除一个 MCP 服务器。scope 可选 user 或 project（默认 user；project 需当前工作区有该服务器）。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"name":  mStrProp("服务器名"),
			"scope": mStrProp("层级：user（默认，全局）或 project（工作区级）"),
		}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name := mArgStr(args, "name")
			if mArgStr(args, "scope") == "project" {
				if err := MCPDelete(MCPLevelProject, name); err != nil {
					return "", fmt.Errorf("工作区级未找到 MCP 服务器 %q（%v）；全局级删除请传 scope=user", name, err)
				}
				return "已删除 MCP 服务器：" + name + "（工作区级）", nil
			}
			if err := MCPDelete(MCPLevelUser, name); err != nil {
				return "", fmt.Errorf("全局级未找到 MCP 服务器 %q（%v）；工作区级删除请传 scope=project", name, err)
			}
			return "已删除 MCP 服务器：" + name + "（用户级）", nil
		},
	})

	// ── 已完成对话历史 ──
	r.Register(&Tool{
		Name: "history_search", Description: "按关键词搜索已完成对话的历史记录（标题/摘要/标签/关键点）。", ReadOnly: true,
		SystemTool: true,
		Parameters: mObjSchema(map[string]any{"query": mStrProp("搜索关键词")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return searchHistoryText(mArgStr(args, "query")), nil
		},
	})
	r.Register(&Tool{
		Name: "history_list", Description: "列出所有已完成对话的历史记录（按完成时间倒序）。",
		ReadOnly: true, SystemTool: true, Parameters: mObjSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) { return listHistoryText(), nil },
	})
	r.Register(&Tool{
		Name: "history_count", Description: "查询已完成对话的历史记录总数。",
		ReadOnly: true, SystemTool: true, Parameters: mObjSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) { return historyCountText(), nil },
	})
}

// ─── Skills 工具实现 ──

// skillRuntimeRoot 运行时解析工作区根（★ 2026-09-12 修复：不再依赖注册时闭包
// 冻结的 root）。优先级：args._wsRoot（JS 插件工具链会话注入）→ ctx 会话绑定根
// （SessionWorkspaceRoot）→ 工作区实时快照（workspaceRootsSnapshot，含运行中
// 添加的项目）→ 注册时 root（自闭环/测试兜底）。
func skillRuntimeRoot(ctx context.Context, args map[string]any) string {
	if args != nil {
		if r := mArgStr(args, "_wsRoot"); r != "" {
			return r
		}
	}
	if r := SessionWorkspaceRoot(ctx); r != "" {
		return r
	}
	if roots := workspaceRootsSnapshot(); len(roots) > 0 {
		return roots[0]
	}
	return ""
}

// skillTargetDir 按 scope 解析技能写入/删除目标目录。
// 返回（目录, 层级中文标签, 错误）。scope：global=全局 / system=内置 / 其他=工作区级。
// ★ 2026-09-12 修复核心：工作区根为空时不再退化为相对路径写入（原 BUG 落点），
//   而是显式回落全局技能目录（<InstallDir>/.pair/skills/）。
func skillTargetDir(ctx context.Context, args map[string]any, registerRoot string) (string, string, error) {
	scope := mArgStr(args, "scope")
	switch scope {
	case "global":
		dir := SkillGlobalDir
		if dir == "" {
			dir = filepath.Join(core.InstallDir(), ".pair", "skills")
		}
		return dir, "全局（跨工作区）", nil
	case "system":
		if SkillSystemDir == "" {
			return "", "", fmt.Errorf("内置技能目录未初始化（SkillSystemDir 为空）")
		}
		return SkillSystemDir, "内置", nil
	}
	// project（默认）：运行时解析工作区根
	root := skillRuntimeRoot(ctx, args)
	if root == "" {
		root = registerRoot
	}
	if root == "" {
		// 未开工作区：显式落全局目录（修复原 BUG：filepath.Join("", name)
		// 相对路径 → 写进进程 CWD=安装目录根）
		dir := SkillGlobalDir
		if dir == "" {
			dir = filepath.Join(core.InstallDir(), ".pair", "skills")
		}
		return dir, "全局（未打开工作区，回落）", nil
	}
	return filepath.Join(root, ".pair", "skills"), "工作区级", nil
}

func listSkillsText(root string) string {
	skills := LoadAllSkillsFromRoot(root, SkillSystemDir, SkillEnabled)
	var b strings.Builder
	for _, s := range skills {
		lvl := "工作区级"
		switch s.Level {
		case LevelSystem:
			lvl = "内置"
		case LevelGlobal:
			lvl = "全局"
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

func writeSkillTool(ctx context.Context, args map[string]any, registerRoot string) (string, error) {
	s := Skill{
		Name: mArgStr(args, "name"), Description: mArgStr(args, "description"),
		Mode: mStr(mArgStr(args, "mode"), "auto"), Body: mArgStr(args, "content"),
	}
	if s.Name == "" || s.Body == "" {
		return "", fmt.Errorf("name 与 content 必填")
	}
	dir, label, err := skillTargetDir(ctx, args, registerRoot)
	if err != nil {
		return "", err
	}
	if err := WriteSkill(dir, s); err != nil {
		return "", err
	}
	return "已写入技能 " + s.Name + "（" + label + "：" + filepath.Join(dir, s.Name) + "）", nil
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
		level = MCPLevelProject
		levelLabel = "工作区级"
	default:
		level = MCPLevelUser
		levelLabel = "用户级（全局）"
	}
	if err := MCPUpsert(level, e); err != nil {
		return "", err
	}
	return "已添加 MCP 服务器 " + e.Name + "（" + levelLabel + "）", nil
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
