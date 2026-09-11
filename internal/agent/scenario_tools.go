// ═══════════════════════════════════════════════════════════════
// scenario_tools.go — 场景（工具集）创造工具（scenario_scan / scenario_create）
//
// 「创造模式」的宿主能力面（2026-09-11）：
//
//	tool-scenario 磁盘插件注册 /创造 命令（按需激活）——执行后其工具
//	scenario_scan / scenario_create 注册进当前会话 agent（MergePluginToolsForConv
//	按激活状态合并），execute 经 ctx.hostTool 路由回本文件实现（对齐
//	「编排在插件、能力在宿主」seam，同 tool-system 模式）。
//
//   - scenario_scan   盘点当前可用能力（插件与工具清单、内置组），供组合决策。
//   - scenario_create 按需求一次性创建「场景工具集」并固化到安装目录
//     .pair/toolsets/<name>.json（内置场景/预设同目录）——创建后对话面板
//     选择该场景即生效（工具面每轮重建，无需重启）。
//
// ★ 注册位置：宿主框架注册表（web_server initReg / AgentBase registry）——
// 供磁盘插件 claimTool 存档（hostTool 执行）；**不注册进会话 reg**——
// 会话可见性由 tool-scenario 按需激活控制（未激活不进 agent 工具面）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// RegisterScenarioTools 注册场景创造工具（宿主框架面；web/desktop/agent 共用）。
// root 为工作区根（兼容保留——场景工具集已全局化，落安装目录），ph 为插件宿主
// （取插件/工具清单）。调用点须在磁盘插件装载（LoadGlobalPlugins）之前——
// claimTool 依赖同名宿主工具在位（存档 hostExecutors 供 ctx.hostTool）。
func RegisterScenarioTools(r *Registry, root string, ph *PluginHost) {
	if r == nil {
		return
	}
	registerScenarioScan(r, ph)
	registerScenarioCreate(r, ph)
}

// ─── scenario_scan：可用能力盘点 ─────────────────────────────

func registerScenarioScan(r *Registry, ph *PluginHost) {
	r.Register(&Tool{
		Name: "scenario_scan",
		Description: "盘点当前可用能力（插件与内置工具组清单），供创建「场景工具集」时组合决策：" +
			"分节列出磁盘/动态插件（插件名 + 用途 + 工具名）与内置工具组（builtin:<组名> + 工具名）。" +
			"detail=full 附每个工具的一行描述。先本工具盘点，再用 scenario_create 组合创建。",
		ReadOnly: true,
		Parameters: mObjSchema(map[string]any{
			"detail": mStrProp("summary（默认）=插件/组+工具名；full=附工具描述"),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return buildScenarioInventory(ph, mArgStr(args, "detail")), nil
		},
	})
}

// buildScenarioInventory 生成能力盘点文本（插件 → 工具；内置组 → 工具）。
func buildScenarioInventory(ph *PluginHost, detail string) string {
	full := strings.EqualFold(strings.TrimSpace(detail), "full")
	if ph == nil {
		return "（插件宿主未装配，无法盘点。）"
	}
	toolDesc := map[string]string{}
	if reg := ph.Context().Tools; reg != nil {
		for _, meta := range reg.AllToolMeta() {
			toolDesc[meta.Name] = trimToolDesc(meta.Description)
		}
	}
	toolLine := func(tn string) string {
		if !full {
			return tn
		}
		if d := toolDesc[tn]; d != "" {
			return tn + "（" + d + "）"
		}
		return tn
	}
	var b strings.Builder
	b.WriteString("## 场景能力盘点（组合创建场景工具集用）\n")
	b.WriteString("组合单位 = 插件（plugins 参数写插件名，可带 tools 白名单）｜内置组（plugins 参数写 \"builtin:<组名>\"，整组加入）。\n")

	// ── 插件（有工具注册的）──
	byPlugin := ph.PluginToolsByPlugin()
	names := make([]string, 0, len(byPlugin))
	for n, list := range byPlugin {
		if n == "" || len(list) == 0 {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)
	b.WriteString("\n### 插件\n")
	if len(names) == 0 {
		b.WriteString("（无）\n")
	}
	for _, n := range names {
		purpose := diskPluginPurpose(n)
		if purpose == "" {
			purpose = jsDefPurpose(ph, n)
		}
		list := append([]string(nil), byPlugin[n]...)
		sort.Strings(list)
		line := "- " + n
		if purpose != "" {
			line += "： " + purpose
		}
		line += " —— 工具：" + strings.Join(mapStrings(list, toolLine), ", ")
		b.WriteString(line + "\n")
	}

	// ── 内置工具组 ──
	b.WriteString("\n### 内置工具组\n")
	groups := builtinPluginToolGroups()
	gnames := make([]string, 0, len(groups))
	for g, list := range groups {
		if g == "" || len(list) == 0 {
			continue
		}
		gnames = append(gnames, g)
	}
	sort.Strings(gnames)
	for _, g := range gnames {
		list := append([]string(nil), groups[g]...)
		sort.Strings(list)
		desc := builtinGroupDesc(g)
		line := "- builtin:" + g
		if desc != "" {
			line += "： " + desc
		}
		line += " —— 工具：" + strings.Join(mapStrings(list, toolLine), ", ")
		b.WriteString(line + "\n")
	}

	b.WriteString("\n使用：scenario_create {name, description, plugins: [\"tool-office\", \"builtin:core\", ...]} " +
		"→ 固化到 .pair/toolsets/<name>.json（对话面板可选择该场景）。\n")
	return b.String()
}

// jsDefPurpose 动态插件（cordis 定义）用途查询（磁盘插件查不到时兜底）。
func jsDefPurpose(ph *PluginHost, name string) string {
	for _, d := range ph.JSDefs() {
		if d.Name() == name || d.id == name {
			return d.purpose
		}
	}
	return ""
}

// mapStrings 列表映射（小工具，避免引入额外依赖）。
func mapStrings(in []string, fn func(string) string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, fn(s))
	}
	return out
}

// ─── scenario_create：创建场景工具集 ─────────────────────────

func registerScenarioCreate(r *Registry, ph *PluginHost) {
	r.Register(&Tool{
		Name: "scenario_create",
		Description: "一次性创建「场景工具集」并固化到安装目录 .pair/toolsets/<name>.json" +
			"（内置场景/预设同目录；创建后对话面板选择即生效）。" +
			"name 场景名（中文或小写字母/数字/-/_；预设名保留不可用）；description 用途描述；" +
			"plugins 组合清单，元素为：字符串（\"tool-office\" 整插件 / \"builtin:core\" 整内置组）" +
			"或对象（{plugin:\"tool-office\",tools:[\"csv_read\"]} 仅白名单工具 / {group:\"core\"}）。" +
			"先用 scenario_scan 盘点可用插件/组再组合。场景 = 工具面白名单：选中该场景时" +
			"仅清单内工具对 agent 可见。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"name":        mStrProp("场景名（中文或小写字母/数字/-/_，如 \"音视频\"、\"data-analysis\"；预设名不可用）"),
			"description": mStrProp("场景用途描述（对话面板展示；必填建议）"),
			"plugins": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": []string{"string", "object"},
				},
				"description": "组合清单：\"插件名\"/\"builtin:组名\"，或 {plugin:\"名\",tools:[...]}/{group:\"组名\"}",
			},
			"overwrite": mStrProp("同名场景已存在时 true=覆盖重建（默认 false 报错）"),
		}, "name", "plugins"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return createScenario(ph, args)
		},
	})
}

// scenarioEntry 解析后的组合条目。
type scenarioEntry struct {
	IsGroup bool
	Name    string   // 插件名 或 内置组名
	Tools   []string // 插件：白名单（空=整插件）；内置组：忽略（用全组快照）
	Purpose string   // 可选覆盖
}

// createScenario 创建场景工具集（校验 → 组合 → 固化）。
func createScenario(ph *PluginHost, args map[string]any) (string, error) {
	if ph == nil {
		return "", fmt.Errorf("插件宿主未装配，无法创建场景")
	}
	name := strings.TrimSpace(mArgStr(args, "name"))
	if name == "" {
		return "", fmt.Errorf("name 必填（场景名）")
	}
	if !validToolsetName(name) {
		return "", fmt.Errorf("场景名 %q 不合法：可用中文或小写字母/数字/-/_", name)
	}
	if isPresetName(name) {
		return "", fmt.Errorf("场景名 %q 为框架预置场景保留名；请换名创建自定义场景（预置内容随版本维护）", name)
	}
	overwrite := strings.EqualFold(strings.TrimSpace(mArgStr(args, "overwrite")), "true")
	if _, err := loadToolset("", toolsetProject, name); err == nil && !overwrite {
		return "", fmt.Errorf("场景 %q 已存在；如需重建请加 overwrite=true", name)
	}

	raw, err := parseScenarioPlugins(args)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("plugins 不能为空：至少组合一个插件或内置组（先 scenario_scan 盘点）")
	}

	// 校验 + 归一化条目
	var entries []scenarioEntry
	seen := map[string]bool{}
	for _, e := range raw {
		key := e.Name
		if e.IsGroup {
			key = "builtin:" + e.Name
		}
		if seen[key] {
			continue // 去重
		}
		seen[key] = true
		if e.IsGroup {
			if !isBuiltinPluginName(e.Name) {
				return "", fmt.Errorf("内置组 %q 不存在（scenario_scan 查看可用组）", e.Name)
			}
			if len(builtinPluginToolGroups()[e.Name]) == 0 {
				return "", fmt.Errorf("内置组 %q 无可用工具", e.Name)
			}
		} else {
			tools := ph.PluginToolsByPlugin()[e.Name]
			if len(tools) == 0 {
				return "", fmt.Errorf("插件 %q 不存在或未装载（scenario_scan 查看可用插件）", e.Name)
			}
			if e.Tools != nil {
				// 白名单：校验工具名都在插件内，白名单外的写入 DisabledTools
				all := map[string]bool{}
				for _, tn := range tools {
					all[tn] = true
				}
				var bad []string
				for _, tn := range e.Tools {
					if !all[tn] {
						bad = append(bad, tn)
					}
				}
				if len(bad) > 0 {
					return "", fmt.Errorf("插件 %q 没有工具 %s（scenario_scan 查看插件工具）", e.Name, strings.Join(bad, ", "))
				}
			}
		}
		entries = append(entries, e)
	}

	// 组装 Toolset（格式与预置场景一致：BuiltinsInited + 框架组条目）
	ts := &Toolset{
		Name:           name,
		Description:    strings.TrimSpace(mArgStr(args, "description")),
		Version:        "1.0.0",
		CreatedAt:      time.Now().Format(time.RFC3339),
		BuiltinsInited: true,
	}
	have := map[string]bool{}
	for _, e := range entries {
		if e.IsGroup {
			g := e.Name
			ts.Plugins = append(ts.Plugins, ToolsetPlugin{
				Name:    "builtin:" + g,
				Purpose: builtinGroupDesc(g),
				Builtin: g,
				Tools:   append([]string(nil), builtinPluginToolGroups()[g]...),
			})
			have["builtin:"+g] = true
			continue
		}
		purpose := e.Purpose
		if purpose == "" {
			purpose = diskPluginPurpose(e.Name)
		}
		if purpose == "" {
			purpose = jsDefPurpose(ph, e.Name)
		}
		sp := ToolsetPlugin{Name: e.Name, Purpose: purpose}
		if e.Tools != nil {
			want := map[string]bool{}
			for _, tn := range e.Tools {
				want[tn] = true
			}
			var disabled []string
			for _, tn := range ph.PluginToolsByPlugin()[e.Name] {
				if !want[tn] {
					disabled = append(disabled, tn)
				}
			}
			sp.DisabledTools = disabled
		}
		ts.Plugins = append(ts.Plugins, sp)
		have[e.Name] = true
	}
	// 框架内置组条目（system/plugin-mgmt/toolset-mgmt；与预置场景同源，去重）
	var reg *Registry
	if ph.Context() != nil {
		reg = ph.Context().Tools
	}
	for _, e := range builtinGroupEntries(reg, ph) {
		if e.Builtin != "" && have["builtin:"+e.Builtin] {
			continue
		}
		ts.Plugins = append(ts.Plugins, e)
	}

	if err := saveToolset("", toolsetProject, ts); err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "✅ 场景工具集 %q 已创建并保存到 .pair/toolsets/%s.json\n", ts.Name, ts.Name)
	if ts.Description != "" {
		fmt.Fprintf(&b, "用途：%s\n", ts.Description)
	}
	fmt.Fprintf(&b, "条目 %d 个：\n", len(ts.Plugins))
	for _, p := range ts.Plugins {
		extra := ""
		if len(p.Tools) > 0 {
			extra = fmt.Sprintf("（%d 工具）", len(p.Tools))
		} else if len(p.DisabledTools) > 0 {
			extra = "（白名单过滤）"
		}
		fmt.Fprintf(&b, "  - %s%s：%s\n", p.Name, extra, p.Purpose)
	}
	b.WriteString("\n使用：对话面板「工具集」选择器选中该场景即生效（工具面按场景白名单收敛；新会话选择同样生效）。")
	return b.String(), nil
}

// parseScenarioPlugins 解析 plugins 参数（数组；元素 string 或对象）。
func parseScenarioPlugins(args map[string]any) ([]scenarioEntry, error) {
	v, ok := args["plugins"]
	if !ok || v == nil {
		return nil, nil
	}
	list, ok := v.([]any)
	if !ok {
		// 兼容字符串形态（单条）与 JSON 字符串（hostTool 透传为对象数组，此处仅防御）
		if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
			return []scenarioEntry{{Name: strings.TrimSpace(s)}}, nil
		}
		return nil, fmt.Errorf("plugins 必须是数组")
	}
	var out []scenarioEntry
	for i, item := range list {
		switch t := item.(type) {
		case string:
			s := strings.TrimSpace(t)
			if s == "" {
				continue
			}
			if strings.HasPrefix(s, "builtin:") {
				out = append(out, scenarioEntry{IsGroup: true, Name: strings.TrimPrefix(s, "builtin:")})
			} else {
				out = append(out, scenarioEntry{Name: s})
			}
		case map[string]any:
			e := scenarioEntry{}
			if g := strings.TrimSpace(asString(t["group"])); g != "" {
				e.IsGroup = true
				e.Name = g
			} else if p := strings.TrimSpace(asString(t["plugin"])); p != "" {
				e.Name = p
			} else if p := strings.TrimSpace(asString(t["name"])); p != "" {
				e.Name = p
			}
			if e.Name == "" {
				return nil, fmt.Errorf("plugins[%d] 缺 plugin/group 名称", i)
			}
			if raw, ok := t["tools"]; ok && raw != nil {
				var tools []string
				switch tl := raw.(type) {
				case []any:
					for _, x := range tl {
						if s := strings.TrimSpace(asString(x)); s != "" {
							tools = append(tools, s)
						}
					}
				case []string:
					for _, s := range tl {
						if s = strings.TrimSpace(s); s != "" {
							tools = append(tools, s)
						}
					}
				}
				e.Tools = tools // nil=整插件；非 nil=白名单（可为空数组→全禁用语义由校验拦截）
				if len(tools) == 0 {
					return nil, fmt.Errorf("plugins[%d]（%s）tools 白名单为空；去掉 tools 表示整插件加入", i, e.Name)
				}
			}
			if p := strings.TrimSpace(asString(t["purpose"])); p != "" {
				e.Purpose = p
			}
			out = append(out, e)
		default:
			return nil, fmt.Errorf("plugins[%d] 类型不支持（string 或对象）", i)
		}
	}
	return out, nil
}

// asString 值转字符串（string/数字等标量的兜底转换）。
func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}
