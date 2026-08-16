// toolset_tools.go — 工具集 agent 工具：动态构建/列表/查看/导出/导入/删除。
//
// 工作流（对齐用户需求）：
//   1. 无工具集配置时：toolset_build 分析项目 → 模板组合生成插件 → 定义装载
//      → 固化到工作区 .pair/toolsets/（后续启动自动装载）
//   2. 显式调用 toolset_build 可重新分析更新工具集（overwrite 覆盖固化）
//   3. toolset_export 导出可移植 JSON → 可导入全局（跨项目）或发布市场/GitHub
//   4. toolset_import 导入（scope=project|user）
//   5. 工具集本身插件化：plugins 是 JS 动态插件（host 半），经 PluginHost 装载；
//      构建模板也可由任意插件注册（ctx.toolset.registerTemplate），市场可扩展。

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RegisterToolsetTools 注册工具集管理工具（web/desktop/agent 共用）。
func RegisterToolsetTools(r *Registry, root string, ph *PluginHost) {
	// ── toolset_build：动态构建/更新工具集 ──
	r.Register(&Tool{
		Name: "toolset_build",
		Description: "动态构建工作区工具集：分析项目（语言/框架/文件结构 + ★LLM 理解项目实际目的）→ " +
			"组合匹配的工具集模板生成插件 → 定义装载（插件化）→ 固化到 .pair/toolsets/{name}.json。" +
			"★ LLM 参与分析：读 README/文件结构判断项目要实现的目的，输出真实构建/测试/运行/lint 命令" +
			"与推荐工具类别（不按语言固化命令；LLM 未配置或失败自动回退静态特征分析）。" +
			"★ LLM 检测模板覆盖不到的项目专属能力缺口时，现场生成专属插件代码并入工具集（define 预检，失败剔除并提示）。" +
			"无工具集配置时首次构建用；有则显式调用可更新（overwrite=true 覆盖已固化版本）。" +
			"可用 toolset_list 查看现有工具集，toolset_export 导出（导入全局/发布市场）。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"name":        mStrProp("工具集名（默认 default；与现有同名时按 overwrite 决定是否覆盖）"),
			"description": mStrProp("工具集用途描述（可选，优先于 LLM 分析出的项目目的）"),
			"requirement": mStrProp("要求描述：期望工具集覆盖的能力（如「Web 前端脚手架 + 接口调试」），LLM 分析时参考"),
			"project":     mStrProp("目标项目目录（默认主工作区；多项目可指定 basename 或路径）"),
			"overwrite":   mStrProp("已存在同名固化工具集时 true=覆盖并重建，false=报错（默认 false）"),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			projectDir := root
			if p := mArgStr(args, "project"); p != "" {
				dir, err := resolveWorkspaceProject(root, p)
				if err != nil {
					return "", err
				}
				projectDir = dir
			}
			name := mArgStr(args, "name")
			if name == "" {
				name = "default"
			}
			name = strings.ToLower(strings.TrimSpace(name))
			if !validToolsetName(name) {
				return "", fmt.Errorf("工具集名只能含小写字母/数字/-/_：%q", name)
			}
			overwrite := mArgStr(args, "overwrite") == "true"
			// 已固化检查（不覆盖时拒绝重建）
			if !overwrite {
				if _, err := os.Stat(toolsetPath(projectDir, toolsetProject, name)); err == nil {
					return "", fmt.Errorf("工具集 %q 已固化；如需重建请加 overwrite=true", name)
				}
			}
			// 旧插件（同名工具集装载过的）先卸载，避免残留
			if ts, err := loadToolset(projectDir, "", name); err == nil {
				UnloadToolsetPlugins(ph, ts)
			}
			ts, err := BuildToolset(ph, projectDir, name, mArgStr(args, "description"), mArgStr(args, "requirement"))
			if err != nil {
				return "", err
			}
			if err := saveToolset(projectDir, toolsetProject, ts); err != nil {
				return "", err
			}
			var b strings.Builder
			fmt.Fprintf(&b, "✅ 工具集 %q 已构建并固化到 .pair/toolsets/%s.json\n", ts.Name, ts.Name)
			fmt.Fprintf(&b, "项目: %s｜插件数: %d｜版本: %s\n", ts.Project, len(ts.Plugins), ts.Version)
			for _, p := range ts.Plugins {
				fmt.Fprintf(&b, "  - %s：%s\n", p.Name, p.Purpose)
			}
			fmt.Fprintf(&b, "\n已装载可用；下次启动自动装配。toolset_export 可导出分享/发布市场。")
			return b.String(), nil
		},
	})

	// ── toolset_edit：手动编辑工具集（插件化思路：加插件/删插件/摘工具/恢复工具）──
	r.Register(&Tool{
		Name: "toolset_edit",
		Description: "手动编辑工具集（插件化思路）：\n" +
			"① add_plugin 向工具集添加插件——来源：宿主已定义 JS 动态插件（cordis_define 定义、" +
			".pair/cordis.patch.json 装配、npm 市场安装的都在宿主 defs 中）、其他工具集（from_toolset）、" +
			"或 plugin_json 直接给插件定义；★ 可选 tools 参数（逗号分隔）只加入插件内指定工具——插件整体装载、白名单外的工具自动摘除（插件内工具可单独加入工具集，enable_tool 可恢复）；\n" +
			"② rm_plugin 从工具集移除插件（其注册的全部工具一并卸载）；\n" +
			"③ rm_tool 摘除插件下单个工具（插件保留、工具对 agent 不可见，enable_tool 可恢复）；\n" +
			"④ enable_tool 恢复被摘除的工具。\n" +
			"★ 内置工具包（name=builtin）：被过滤的 pair 独有工具按内置插件组（core/git/codegraph/memory/…）" +
			"管理——add_builtin {builtin_group=组名} 选择加入（组内工具全部对 agent 可见并固化 .pair/toolsets/builtin.json）；" +
			"add_builtin_all 强制全部内置组加入（开关）；rm_plugin {plugin_name=builtin:组名} 移出（恢复默认过滤）；" +
			"rm_tool/enable_tool 摘除/恢复组内单个工具。\n" +
			"操作即时热装载（装卸对应插件）并回写固化 .pair/toolsets/{name}.json（重启自动装配保持）。" +
			"用 toolset_show 查看工具集现有插件。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"name":          mStrProp("工具集名（必填；builtin=内置工具包）"),
			"scope":         mStrProp("project/global（默认自动：先工作区再全局；builtin 固定工作区）"),
			"action":        mStrProp("add_plugin / rm_plugin / rm_tool / enable_tool / add_builtin / add_builtin_all（必填）"),
			"plugin_name":   mStrProp("插件名（add/rm 均需；内置组用 builtin:组名）"),
			"builtin_group": mStrProp("add_builtin 时的内置分组名（core/git/codegraph/…；toolset_show builtin 查看）"),
			"from_toolset":  mStrProp("add_plugin 时从其他工具集拷贝插件（不填=从宿主已定义插件找）"),
			"tool":          mStrProp("rm_tool / enable_tool 时的工具名"),
			"tools":         mStrProp("add_plugin 可选：逗号分隔的工具白名单——只加入插件内指定工具，其余自动摘除（插件内工具可单独加入工具集）"),
			"plugin_json":   mStrProp("add_plugin 直接给插件 JSON：{\"name\":\"…\",\"purpose\":\"…\",\"code\":\"…\",\"client\":\"…\"}"),
			"overwrite":     mStrProp("add_plugin 遇重名时 true=覆盖重装（先删旧），false=报错（默认）"),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return toolsetEdit(ph, root, args)
		},
	})

	// ── toolset_list：列出工具集 ──
	r.Register(&Tool{
		Name:        "toolset_list",
		Description: "列出工作区全部工具集（名称/用途/插件数/来源；工具集仅工作区级，无全局工具集）。",
		ReadOnly:    true,
		Parameters:  mObjSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			metas := listAllToolsets(root)
			if len(metas) == 0 {
				return "（暂无工具集。可用 toolset_build 按项目动态构建一个）", nil
			}
			var b strings.Builder
			b.WriteString("## 工具集\n")
			for _, m := range metas {
				scope := "工作区"
				switch m.Scope {
				case "global":
					scope = "全局"
				case "builtin":
					scope = "内置"
				}
				fmt.Fprintf(&b, "- **%s** [%s]（%s，%d 个%s）\n", m.Name, scope, m.Description, m.PluginCount,
					boolStr(m.Scope == "builtin", "分组", "插件"))
			}
			b.WriteString("\ntoolset_show {name} 查看详情；toolset_build 构建/更新；toolset_export 导出。\n" +
				"内置工具集 builtin：被过滤的 pair 独有工具分组（core/git/codegraph/…），默认不加入；\n" +
				"toolset_edit {name=builtin, action=add_builtin, builtin_group=组名} 选择加入，add_builtin_all 强制全部。")
			return b.String(), nil
		},
	})

	// ── toolset_show：查看工具集详情 ──
	r.Register(&Tool{
		Name:        "toolset_show",
		Description: "查看工具集详情：插件清单（名称/用途）、来源作用域、版本。name=builtin 查看内置工具包分组（含工具启用状态）。",
		ReadOnly:    true,
		Parameters: mObjSchema(map[string]any{
			"name": mStrProp("工具集名（builtin=内置工具包）"),
		}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name := mArgStr(args, "name")
			if name == builtinToolsetName {
				// ★ 内置工具集：虚拟派生展示（分组 + 工具 + 启用状态 + 已加入标记）
				ph := GetGlobalPluginHost()
				reg := (*Registry)(nil)
				if ph != nil && ph.Context() != nil {
					reg = ph.Context().Tools
				}
				info := BuiltinToolsetInfoOf(reg, ph, root)
				var b strings.Builder
				fmt.Fprintf(&b, "## 内置工具集 builtin（%d 组 / %d 工具，已启用 %d）\n", len(info.Groups), info.ToolTotal, info.EnabledTotal)
				b.WriteString("默认不加入工作区（组内工具对 agent 不可见）；选择加入后固化 .pair/toolsets/builtin.json。\n")
				b.WriteString("已加入分组: " + boolStr(len(info.Joined) > 0, "["+strings.Join(info.Joined, ", ")+"]", "（无）") + "\n\n")
				for _, g := range info.Groups {
					mark := boolStr(g.Enabled, "●已启用", boolStr(g.Partial, "◐部分", "○已过滤"))
					fmt.Fprintf(&b, "### %s [%s]（%d 工具）\n%s\n", g.Name, mark, len(g.Tools), g.Desc)
					var names []string
					for _, t := range g.Tools {
						s := t.Name
						if !t.Enabled {
							s += "（过滤）"
						}
						names = append(names, s)
					}
					fmt.Fprintf(&b, "  %s\n\n", strings.Join(names, " "))
				}
				b.WriteString("加入: toolset_edit {name=builtin, action=add_builtin, builtin_group=组名}；强制全部: action=add_builtin_all；移出: action=rm_plugin, plugin_name=builtin:组名")
				return b.String(), nil
			}
			ts, err := loadToolset(root, "", name)
			if err != nil {
				return "", err
			}
			var b strings.Builder
			fmt.Fprintf(&b, "## 工具集 %s\n- 用途: %s\n- 项目: %s\n- 版本: %s\n- 来源: %s\n- 创建: %s\n\n## 插件\n",
				ts.Name, ts.Description, ts.Project, ts.Version, "工作区", ts.CreatedAt)
			// ★ 每个插件列出其工具清单与启用状态（PluginToolsByPlugin 取该插件注册的工具；
			// DisabledTools 摘除清单内工具标记「已摘除」；插件未装载时提示）
			pluginTools := map[string][]string{}
			if ph != nil {
				pluginTools = ph.PluginToolsByPlugin()
			}
			for _, p := range ts.Plugins {
				extra := ""
				if p.Client != "" {
					extra = "（含 client 半）"
				}
				if p.Builtin != "" {
					extra += "（内置组）"
				}
				fmt.Fprintf(&b, "- **%s**：%s%s\n", p.Name, p.Purpose, extra)
				disabled := map[string]bool{}
				for _, tn := range p.DisabledTools {
					disabled[tn] = true
				}
				tools := pluginTools[p.Name]
				if len(tools) == 0 && len(disabled) == 0 {
					continue // 插件未装载/无工具注册：不展示工具行
				}
				var names []string
				for _, tn := range tools {
					if disabled[tn] {
						names = append(names, tn+"（已摘除）")
					} else {
						names = append(names, tn)
					}
				}
				for tn := range disabled {
					if !strSliceContains(tools, tn) {
						names = append(names, tn+"（已摘除·未注册）")
					}
				}
				if len(names) > 0 {
					fmt.Fprintf(&b, "    └ 工具: %s\n", strings.Join(names, " "))
				}
			}
			return b.String(), nil
		},
	})

	// ── toolset_export：导出（可移植 JSON，可导入全局/发布市场）──
	r.Register(&Tool{
		Name:        "toolset_export",
		Description: "导出工具集为可移植 JSON（含全部插件代码）：to 为文件路径时写入文件（可 git 提交发布 GitHub），否则返回 JSON 内容。可用于导入全局（跨项目）或发布市场。",
		ReadOnly:    true,
		Parameters: mObjSchema(map[string]any{
			"name":   mStrProp("工具集名"),
			"to":     mStrProp("导出文件路径（可选；缺省返回 JSON 内容）"),
			"tags":   mStrProp("逗号分隔标签（市场发布用）"),
			"author": mStrProp("作者（市场发布用）"),
			"repo":   mStrProp("目标仓库 github:owner/repo（市场发布用）"),
		}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name := mArgStr(args, "name")
			ts, err := loadToolset(root, "", name)
			if err != nil {
				return "", err
			}
			var tags []string
			if t := mArgStr(args, "tags"); t != "" {
				for _, x := range strings.Split(t, ",") {
					if x = strings.TrimSpace(x); x != "" {
						tags = append(tags, x)
					}
				}
			}
			content, err := ExportToolsetJSON(ts, tags, mArgStr(args, "author"), mArgStr(args, "repo"))
			if err != nil {
				return "", err
			}
			to := mArgStr(args, "to")
			if to == "" {
				return content, nil
			}
			abs, err := filepath.Abs(to)
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
				return "", fmt.Errorf("写入导出文件失败: %w", err)
			}
			return fmt.Sprintf("✅ 工具集 %q 已导出到 %s\n（可提交到 GitHub 仓库发布市场；或用 toolset_import 导入全局/其他工作区）", name, abs), nil
		},
	})

	// ── toolset_import：导入（scope=project|user）──
	r.Register(&Tool{
		Name:             "toolset_import",
		Description:      "导入工具集：json 为发布 JSON 内容、file 为发布 JSON 文件路径。scope=user 导入全局（跨项目可用）、project 导入工作区。导入后立即装载。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"json":  mStrProp("工具集发布 JSON 内容（与 file 二选一）"),
			"file":  mStrProp("发布 JSON 文件路径（与 json 二选一）"),
			"scope": mStrProp("project（默认，工作区）/ user（全局）"),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			content := mArgStr(args, "json")
			if content == "" {
				f := mArgStr(args, "file")
				if f == "" {
					return "", fmt.Errorf("需要 json 或 file 参数")
				}
				data, err := os.ReadFile(f)
				if err != nil {
					return "", fmt.Errorf("读取导入文件失败: %w", err)
				}
				content = string(data)
			}
			var pub ToolsetPublish
			if err := json.Unmarshal([]byte(content), &pub); err != nil {
				return "", fmt.Errorf("导入 JSON 解析失败（应为 toolset_export 输出格式）: %w", err)
			}
			ts := &pub.Toolset
			if ts.Name == "" || len(ts.Plugins) == 0 {
				return "", fmt.Errorf("导入内容不是有效工具集（缺 name/plugins）")
			}
			// ★ 工具集仅工作区级（没有全局工具集）；scope=user 拒绝
			if mArgStr(args, "scope") == "user" {
				return "", fmt.Errorf("工具集仅工作区级（没有全局工具集）；导入 scope 只支持 project。全局生效的是插件（UI 类），用 cordis_define scope=global 创建")
			}
			if err := saveToolset(root, toolsetProject, ts); err != nil {
				return "", err
			}
			if err := installToolset(ph, ts); err != nil {
				return "", fmt.Errorf("工具集已固化但装载失败: %w", err)
			}
			return fmt.Sprintf("✅ 工具集 %q 已导入（工作区）并装载（%d 个插件）", ts.Name, len(ts.Plugins)), nil
		},
	})

	// ── toolset_remove：删除工具集 ──
	r.Register(&Tool{
		Name:             "toolset_remove",
		Description:      "删除工具集（scope 指定删除作用域；缺省两个作用域同名都删）。已装载插件同步卸载。内置工具集 builtin 不可删除（用 toolset_edit 分组移出）。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"name":  mStrProp("工具集名（builtin 不可删除）"),
			"scope": mStrProp("project（仅工作区级；没有全局工具集）"),
		}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name := mArgStr(args, "name")
			if name == builtinToolsetName {
				return "", fmt.Errorf("内置工具集 builtin 不可删除；用 toolset_edit {name=builtin, action=rm_plugin, plugin_name=builtin:组名} 分组移出，或 add_builtin_all 强制全部加入")
			}
			if mArgStr(args, "scope") == "global" {
				return "", fmt.Errorf("工具集仅工作区级（没有全局工具集）；全局生效的是插件（UI 类），用 cordis_define scope=global 管理")
			}
			// 先卸载已装载插件/恢复内置条目
			if ts, err := loadToolset(root, toolsetProject, name); err == nil {
				UnloadToolsetPlugins(ph, ts)
			}
			if err := removeToolset(root, toolsetProject, name); err != nil {
				return "", err
			}
			return fmt.Sprintf("已删除工具集 %q（插件已卸载）", name), nil
		},
	})
}

// validToolsetName 校验工具集名（小写字母/数字/-/_）。
func validToolsetName(name string) bool {
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return name != ""
}

// boolStr 条件字符串。
func boolStr(cond bool, yes, no string) string {
	if cond {
		return yes
	}
	return no
}

// strSliceContains 判断字符串切片是否包含目标。
func strSliceContains(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

// resolveWorkspaceProject 解析 project 参数（多项目）：basename 或路径 → 目录。
func resolveWorkspaceProject(primaryRoot, project string) (string, error) {
	// 绝对路径或相对主工作区路径
	if filepath.IsAbs(project) {
		if st, err := os.Stat(project); err == nil && st.IsDir() {
			return project, nil
		}
		return "", fmt.Errorf("项目目录不存在: %s", project)
	}
	cand := project
	if !filepath.IsAbs(cand) {
		cand = filepath.Join(primaryRoot, project)
	}
	if st, err := os.Stat(cand); err == nil && st.IsDir() {
		return cand, nil
	}
	// basename 匹配工作区文件夹
	for _, w := range WorkspaceRoots {
		if filepath.Base(w) == project {
			return w, nil
		}
	}
	return "", fmt.Errorf("未找到项目 %q（当前工作区: %s）", project, primaryRoot)
}

// ─── toolset_edit 各 action 实现 ───────────────────────────

// toolsetEditAddPlugin 向工具集添加插件（来源：宿主 defs / 其他工具集 / plugin_json）。
func toolsetEditAddPlugin(ph *PluginHost, root string, scope toolsetScope, ts *Toolset, args map[string]any) (string, error) {
	pn := strings.TrimSpace(mArgStr(args, "plugin_name"))
	pj := mArgStr(args, "plugin_json")
	var src ToolsetPlugin
	switch {
	case pj != "":
		if err := json.Unmarshal([]byte(pj), &src); err != nil {
			return "", fmt.Errorf("plugin_json 解析失败: %v", err)
		}
		if src.Name == "" || strings.TrimSpace(src.Code) == "" {
			return "", fmt.Errorf("plugin_json 需含 name 与 code（可含 purpose/client）")
		}
	case pn != "":
		from := mArgStr(args, "from_toolset")
		if from != "" {
			other, err := loadToolset(root, "", from)
			if err != nil {
				return "", err
			}
			found := false
			for _, p := range other.Plugins {
				if p.Name == pn {
					src = p
					found = true
					break
				}
			}
			if !found {
				return "", fmt.Errorf("工具集 %q 中没有插件 %q（toolset_show %s 查看）", from, pn, from)
			}
		} else {
			// 宿主 defs：cordis_define 定义 / .pair/cordis.patch.json 装配 / npm 安装的都在
			found := false
			for _, d := range ph.JSDefs() {
				if d.Name() == pn || d.id == pn {
					src = ToolsetPlugin{Name: d.Name(), Purpose: d.purpose, Code: d.code, Client: d.clientCode}
					found = true
					break
				}
			}
			if !found {
				return "", fmt.Errorf("宿主未定义插件 %q（可用 cordis_define 定义、npm 市场安装，或 from_toolset/plugin_json 提供）", pn)
			}
		}
	default:
		return "", fmt.Errorf("add_plugin 需要 plugin_name（宿主/其他工具集来源）或 plugin_json（直接给定义）")
	}
	// 重名处理
	for i, p := range ts.Plugins {
		if p.Name == src.Name {
			if mArgStr(args, "overwrite") != "true" {
				return "", fmt.Errorf("工具集已有插件 %q；overwrite=true 覆盖重装或先 rm_plugin", src.Name)
			}
			if _, ok := ph.Get(src.Name); ok {
				_ = ph.Unload(src.Name)
				_ = ph.Undefine(src.Name)
			}
			ts.Plugins = append(ts.Plugins[:i], ts.Plugins[i+1:]...)
			break
		}
	}
	// 预检装载（失败不写入固化文件，避免状态不一致）
	if err := applyToolsetPlugin(ph, &src); err != nil {
		return "", fmt.Errorf("插件装载失败（未写入工具集）: %w", err)
	}
	// ★ tools 白名单（可选）：只加入插件内指定工具——插件已整体装载（全部工具注册），
	// 按 PluginToolsByPlugin 查询该插件实际注册的工具，白名单外的写入 DisabledTools
	// （重装载应用禁用）；白名单中未注册的工具名给警告但不阻塞。
	// ★ 2026-08-17：白名单内工具必须显式启用（SetToolEnabled(true)）——harness
	//   对齐模式下工具可能被预置禁用（Enabled=false 但不在 DisabledTools），
	//   否则 add_plugin 报「仅启用 X」但 X 实际仍不可见。
	if whitelist := mArgStr(args, "tools"); whitelist != "" {
		want := map[string]bool{}
		for _, t := range strings.Split(whitelist, ",") {
			if t = strings.TrimSpace(t); t != "" {
				want[t] = true
			}
		}
		all := ph.PluginToolsByPlugin()[src.Name]
		if len(all) == 0 {
			return "", fmt.Errorf("插件 %q 已装载但未注册任何工具（无法按 tools 白名单筛选；去掉 tools 参数整插件加入）", src.Name)
		}
		var disabled []string
		var enabled []string
		var unknown []string
		for _, tn := range all {
			if want[tn] {
				enabled = append(enabled, tn)
			} else {
				disabled = append(disabled, tn)
			}
		}
		for t := range want {
			found := false
			for _, tn := range all {
				if tn == t {
					found = true
					break
				}
			}
			if !found {
				unknown = append(unknown, t)
			}
		}
		if len(disabled) > 0 {
			src.DisabledTools = disabled
			// 重装载应用禁用清单（插件已装载，先卸载再重定义）
			if err := applyToolsetPlugin(ph, &src); err != nil {
				return "", fmt.Errorf("应用工具白名单失败: %w", err)
			}
		}
		// 白名单内工具显式启用（幂等：已启用跳过）
		if ph.Context() != nil && ph.Context().Tools != nil {
			for _, tn := range enabled {
				if _, ok := ph.Context().Tools.Get(tn); ok {
					ph.Context().Tools.SetToolEnabled(tn, true)
				}
			}
		}
		ts.Plugins = append(ts.Plugins, src)
		if err := saveToolset(root, scope, ts); err != nil {
			return "", err
		}
		msg := fmt.Sprintf("✅ 插件 %q 已加入工具集 %q（%s），仅启用工具: %s", src.Name, ts.Name, scope, strings.Join(enabled, ", "))
		if len(unknown) > 0 {
			msg += fmt.Sprintf("；白名单中未注册的工具（已忽略）: %s", strings.Join(unknown, ", "))
		}
		return msg, nil
	}
	ts.Plugins = append(ts.Plugins, src)
	if err := saveToolset(root, scope, ts); err != nil {
		return "", err
	}
	return fmt.Sprintf("✅ 插件 %q 已加入工具集 %q（%s）并装载。工具集现有 %d 个插件。",
		src.Name, ts.Name, scope, len(ts.Plugins)), nil
}

// toolsetEditRmPlugin 从工具集移除插件（卸载其注册的全部工具）。
func toolsetEditRmPlugin(ph *PluginHost, root string, scope toolsetScope, ts *Toolset, args map[string]any) (string, error) {
	pn := strings.TrimSpace(mArgStr(args, "plugin_name"))
	if pn == "" {
		return "", fmt.Errorf("rm_plugin 需要 plugin_name")
	}
	idx := -1
	for i, p := range ts.Plugins {
		if p.Name == pn {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("工具集 %q 中没有插件 %q（toolset_show 查看现有插件）", ts.Name, pn)
	}
	unloadToolsetPlugin(ph, &ts.Plugins[idx])
	ts.Plugins = append(ts.Plugins[:idx], ts.Plugins[idx+1:]...)
	if err := saveToolset(root, scope, ts); err != nil {
		return "", err
	}
	return fmt.Sprintf("✅ 插件 %q 已从工具集 %q 移除（工具已卸载）。剩余 %d 个插件。",
		pn, ts.Name, len(ts.Plugins)), nil
}

// toolsetEditRmTool 摘除插件下单个工具（插件保留；工具禁用 → agent 不可见）。
func toolsetEditRmTool(ph *PluginHost, root string, scope toolsetScope, ts *Toolset, args map[string]any) (string, error) {
	pn := strings.TrimSpace(mArgStr(args, "plugin_name"))
	tool := strings.TrimSpace(mArgStr(args, "tool"))
	if pn == "" || tool == "" {
		return "", fmt.Errorf("rm_tool 需要 plugin_name 与 tool")
	}
	idx := -1
	for i, p := range ts.Plugins {
		if p.Name == pn {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("工具集 %q 中没有插件 %q", ts.Name, pn)
	}
	p := &ts.Plugins[idx]
	// 去重加入禁用清单
	for _, t := range p.DisabledTools {
		if t == tool {
			return fmt.Sprintf("工具 %q 已在插件 %q 的摘除清单中", tool, pn), nil
		}
	}
	// 工具存在性提示（不阻塞：插件未运行时工具本就不注册，记录后下次装载生效）
	registered := false
	if ph.Context() != nil && ph.Context().Tools != nil {
		if _, ok := ph.Context().Tools.Get(tool); ok {
			registered = true
		}
	}
	if !registered {
		return "", fmt.Errorf("工具 %q 当前未注册（插件可能未运行或工具名有误；插件面板/工具集详情可查该插件工具列表）", tool)
	}
	p.DisabledTools = append(p.DisabledTools, tool)
	// 重装载该插件（apply 重新注册 → 应用新禁用清单）
	if err := applyToolsetPlugin(ph, p); err != nil {
		return "", fmt.Errorf("重装载失败（未写入工具集）: %w", err)
	}
	if err := saveToolset(root, scope, ts); err != nil {
		return "", err
	}
	return fmt.Sprintf("✅ 工具 %q 已从插件 %q 摘除（插件保留，工具对 agent 不可见；enable_tool 可恢复）", tool, pn), nil
}

// toolsetEditEnableTool 恢复被摘除的工具。
func toolsetEditEnableTool(ph *PluginHost, root string, scope toolsetScope, ts *Toolset, args map[string]any) (string, error) {
	pn := strings.TrimSpace(mArgStr(args, "plugin_name"))
	tool := strings.TrimSpace(mArgStr(args, "tool"))
	if pn == "" || tool == "" {
		return "", fmt.Errorf("enable_tool 需要 plugin_name 与 tool")
	}
	idx := -1
	for i, p := range ts.Plugins {
		if p.Name == pn {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("工具集 %q 中没有插件 %q", ts.Name, pn)
	}
	p := &ts.Plugins[idx]
	removed := false
	for i, t := range p.DisabledTools {
		if t == tool {
			p.DisabledTools = append(p.DisabledTools[:i], p.DisabledTools[i+1:]...)
			removed = true
			break
		}
	}
	// 恢复启用：无论是否在摘除清单，只要工具已注册且当前 disabled → 启用。
	// （★ 2026-08-17：harness 对齐模式下工具可能被预置禁用（Enabled=false）但不在
	//   DisabledTools 中——「已加入插件的整组恢复」必须幂等地恢复这类工具，
	//   否则 enable_tool 报「不在摘除清单中」而工具仍不可见。）
	reg := (*Registry)(nil)
	if ph != nil && ph.Context() != nil {
		reg = ph.Context().Tools
	}
	enabled := false
	if reg != nil {
		if t, ok := reg.Get(tool); ok {
			enabled = t.Enabled
			if !t.Enabled {
				reg.SetToolEnabled(tool, true)
			}
		}
	}
	if err := saveToolset(root, scope, ts); err != nil {
		return "", err
	}
	if !removed && enabled {
		return fmt.Sprintf("工具 %q 不在插件 %q 的摘除清单中（已启用，无需操作）", tool, pn), nil
	}
	return fmt.Sprintf("✅ 工具 %q 已恢复（插件 %q 的工具重新对 agent 可见）", tool, pn), nil
}

// toolsetEdit 工具集手动编辑核心逻辑（toolset_edit 工具与 /api/toolsets/edit 共用）。
func toolsetEdit(ph *PluginHost, root string, args map[string]any) (string, error) {
	name := strings.TrimSpace(mArgStr(args, "name"))
	if name == "" {
		return "", fmt.Errorf("需要 name（工具集名）")
	}
	action := mArgStr(args, "action")
	if action == "" {
		return "", fmt.Errorf("需要 action：add_plugin / rm_plugin / rm_tool / enable_tool / add_builtin / add_builtin_all")
	}

	// ★ 内置工具集（builtin）特殊通道：add_builtin（加入一组）/ add_builtin_all（强制全部）
	if name == builtinToolsetName {
		switch action {
		case "add_builtin":
			gn := strings.TrimSpace(mArgStr(args, "builtin_group"))
			if gn == "" {
				return "", fmt.Errorf("add_builtin 需要 builtin_group（内置分组名；toolset_show builtin 查看分组）")
			}
			return SetBuiltinGroupEnabled(ph, root, gn, true)
		case "add_builtin_all":
			return EnableAllBuiltin(ph, root)
		case "rm_plugin":
			pn := strings.TrimSpace(mArgStr(args, "plugin_name"))
			gn := strings.TrimPrefix(pn, "builtin:")
			if gn == "" || gn == pn {
				return "", fmt.Errorf("移除内置分组请用 plugin_name=builtin:组名（toolset_show builtin 查看分组）")
			}
			return SetBuiltinGroupEnabled(ph, root, gn, false)
		case "rm_tool", "enable_tool":
			// 内置条目工具级摘除/恢复（操作 builtin.json 中对应条目）
			return toolsetEditBuiltinTool(ph, root, args, action)
		default:
			return "", fmt.Errorf("内置工具集不支持 action %q（add_builtin/add_builtin_all/rm_plugin/rm_tool/enable_tool）", action)
		}
	}

	// ★ 工具集仅工作区级（没有全局工具集）
	if mArgStr(args, "scope") == "global" {
		return "", fmt.Errorf("工具集仅工作区级（没有全局工具集）；全局生效的是插件（UI 类），用 cordis_define scope=global 管理")
	}
	ts, err := loadToolset(root, toolsetProject, name)
	if err != nil {
		return "", err
	}
	resolved := toolsetProject
	switch action {
	case "add_plugin":
		return toolsetEditAddPlugin(ph, root, resolved, ts, args)
	case "rm_plugin":
		return toolsetEditRmPlugin(ph, root, resolved, ts, args)
	case "rm_tool":
		return toolsetEditRmTool(ph, root, resolved, ts, args)
	case "enable_tool":
		return toolsetEditEnableTool(ph, root, resolved, ts, args)
	case "add_builtin", "add_builtin_all":
		return "", fmt.Errorf("内置分组加入请用 name=builtin（toolset_edit {name=builtin, action=%s}）", action)
	default:
		return "", fmt.Errorf("未知 action %q（add_plugin/rm_plugin/rm_tool/enable_tool/add_builtin/add_builtin_all）", action)
	}
}

// toolsetEditBuiltinTool 内置工具集条目的工具级摘除/恢复（rm_tool/enable_tool）。
// 直接修改工作区主工具集（default.json）中对应内置条目的 DisabledTools 并热应用。
func toolsetEditBuiltinTool(ph *PluginHost, root string, args map[string]any, action string) (string, error) {
	pn := strings.TrimSpace(mArgStr(args, "plugin_name"))
	tool := strings.TrimSpace(mArgStr(args, "tool"))
	gn := strings.TrimPrefix(pn, "builtin:")
	if gn == "" || gn == pn || tool == "" {
		return "", fmt.Errorf("需要 plugin_name=builtin:组名 与 tool（工具名）")
	}
	ts, err := workspaceMainToolset(root)
	if err != nil {
		return "", err
	}
	idx := -1
	for i := range ts.Plugins {
		if ts.Plugins[i].Builtin == gn {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("内置组 %q 未加入工作区（toolset_edit {name=builtin, action=add_builtin, builtin_group=%s} 先加入）", gn, gn)
	}
	p := &ts.Plugins[idx]
	if action == "enable_tool" {
		// 从摘除清单移除 + 启用
		removed := false
		for i, t := range p.DisabledTools {
			if t == tool {
				p.DisabledTools = append(p.DisabledTools[:i], p.DisabledTools[i+1:]...)
				removed = true
				break
			}
		}
		if !removed {
			return fmt.Sprintf("工具 %q 不在内置组 %q 的摘除清单中（无需恢复）", tool, gn), nil
		}
		if ph.Context() != nil && ph.Context().Tools != nil {
			ph.Context().Tools.SetToolEnabled(tool, true)
		}
	} else {
		// 摘除（去重）
		for _, t := range p.DisabledTools {
			if t == tool {
				return fmt.Sprintf("工具 %q 已在内置组 %q 的摘除清单中", tool, gn), nil
			}
		}
		p.DisabledTools = append(p.DisabledTools, tool)
		if ph.Context() != nil && ph.Context().Tools != nil {
			ph.Context().Tools.SetToolEnabled(tool, false)
		}
	}
	if err := saveToolset(root, toolsetProject, ts); err != nil {
		return "", err
	}
	if action == "enable_tool" {
		return fmt.Sprintf("✅ 工具 %q 已恢复（内置组 %q 的工具重新对 agent 可见）", tool, gn), nil
	}
	return fmt.Sprintf("✅ 工具 %q 已从内置组 %q 摘除（组保留，工具对 agent 不可见；enable_tool 可恢复）", tool, gn), nil
}

// EditToolsetPublic 工具集手动编辑（公开导出，web_server/前端面板直调）。
// args 与 toolset_edit 工具参数一致：name/scope/action/plugin_name/from_toolset/tool/plugin_json/overwrite。
func EditToolsetPublic(ph *PluginHost, root string, args map[string]any) (string, error) {
	return toolsetEdit(ph, root, args)
}
