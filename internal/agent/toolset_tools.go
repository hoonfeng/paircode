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
				for _, scope := range []toolsetScope{toolsetProject, toolsetGlobal} {
					if _, err := os.Stat(toolsetPath(projectDir, scope, name)); err == nil {
						return "", fmt.Errorf("工具集 %q 已固化（%s）；如需重建请加 overwrite=true", name, scope)
					}
				}
			}
			// 旧插件（同名工具集装载过的）先卸载，避免残留
			if ts, err := loadToolset(projectDir, "", name); err == nil {
				for _, p := range ts.Plugins {
					if _, ok := ph.Get(p.Name); ok {
						_ = ph.Unload(p.Name)
						_ = ph.Undefine(p.Name)
					}
				}
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

	// ── toolset_list：列出工具集 ──
	r.Register(&Tool{
		Name:        "toolset_list",
		Description: "列出工作区与全局全部工具集（名称/用途/插件数/来源）。",
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
				if m.Scope == "global" {
					scope = "全局"
				}
				fmt.Fprintf(&b, "- **%s** [%s]（%s，%d 个插件）\n", m.Name, scope, m.Description, m.PluginCount)
			}
			b.WriteString("\ntoolset_show {name} 查看详情；toolset_build 构建/更新；toolset_export 导出。")
			return b.String(), nil
		},
	})

	// ── toolset_show：查看工具集详情 ──
	r.Register(&Tool{
		Name:        "toolset_show",
		Description: "查看工具集详情：插件清单（名称/用途）、来源作用域、版本。",
		ReadOnly:    true,
		Parameters: mObjSchema(map[string]any{
			"name": mStrProp("工具集名"),
		}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name := mArgStr(args, "name")
			ts, err := loadToolset(root, "", name)
			if err != nil {
				return "", err
			}
			scope := "工作区"
			if _, gerr := os.Stat(toolsetPath(root, toolsetGlobal, name)); gerr == nil {
				scope = "全局"
				if _, perr := os.Stat(toolsetPath(root, toolsetProject, name)); perr == nil {
					scope = "工作区（覆盖全局）"
				}
			}
			var b strings.Builder
			fmt.Fprintf(&b, "## 工具集 %s\n- 用途: %s\n- 项目: %s\n- 版本: %s\n- 来源: %s\n- 创建: %s\n\n## 插件\n",
				ts.Name, ts.Description, ts.Project, ts.Version, scope, ts.CreatedAt)
			for _, p := range ts.Plugins {
				fmt.Fprintf(&b, "- **%s**：%s%s\n", p.Name, p.Purpose, boolStr(p.Client != "", "（含 client 半）", ""))
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
			scope := toolsetProject
			if mArgStr(args, "scope") == "user" {
				scope = toolsetGlobal
			}
			if err := saveToolset(root, scope, ts); err != nil {
				return "", err
			}
			if err := installToolset(ph, ts); err != nil {
				return "", fmt.Errorf("工具集已固化但装载失败: %w", err)
			}
			return fmt.Sprintf("✅ 工具集 %q 已导入（%s）并装载（%d 个插件）",
				ts.Name, map[toolsetScope]string{toolsetProject: "工作区", toolsetGlobal: "全局"}[scope], len(ts.Plugins)), nil
		},
	})

	// ── toolset_remove：删除工具集 ──
	r.Register(&Tool{
		Name:             "toolset_remove",
		Description:      "删除工具集（scope 指定删除作用域；缺省两个作用域同名都删）。已装载插件同步卸载。",
		RequiresApproval: true,
		Parameters: mObjSchema(map[string]any{
			"name":  mStrProp("工具集名"),
			"scope": mStrProp("project/global/缺省（都删）"),
		}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name := mArgStr(args, "name")
			var scope toolsetScope
			switch mArgStr(args, "scope") {
			case "project":
				scope = toolsetProject
			case "global":
				scope = toolsetGlobal
			}
			// 先卸载已装载插件
			if ts, err := loadToolset(root, scope, name); err == nil {
				for _, p := range ts.Plugins {
					if _, ok := ph.Get(p.Name); ok {
						_ = ph.Unload(p.Name)
						_ = ph.Undefine(p.Name)
					}
				}
			}
			if err := removeToolset(root, scope, name); err != nil {
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
