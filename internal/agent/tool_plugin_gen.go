//go:build toolsgen

// ═══════════════════════════════════════════════════════════════
// tool_plugin_gen.go — 磁盘工具插件生成器（build tag: toolsgen）
//
// 用法（工作区根）：
//   go run -tags toolsgen ./dev/tool_plugin_gen
//
// 作用：遍历尚未外置的复杂内置工具组（git/memory/verify/task/
// project-info/binary（含逆向，2026-08-16 并入 binary-re）/debug/vision/
// screenshot/web-debug/bug/office/lsp/codegraph/codegraph-extra），把每组
// 注册的工具定义（name/description/usageGuide/category/readOnly/
// requiresApproval/parameters）完整导出为 .pair/plugins/tool-<组>/index.js
// 磁盘插件。
//
// 生成产物与 tool-core 等手工迁移插件同构：api 声明在插件、execute
// 调 ctx.hostTool.exec 复用宿主 Go 执行器（seam 编排在插件/能力在宿主）。
// Go 侧工具描述变更后重跑生成器即可同步（幂等，仅重写有差异的文件）。
// SystemTool 内部工具不迁移（不暴露给 LLM，无外置意义）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// genToolGroup 一个待生成的工具组。
type genToolGroup struct {
	plugin string // 磁盘插件目录名（tool-<组>）
	desc   string // 插件用途（purpose/头注释）
	apply  func(r *Registry, root string) // 组注册函数
	names  []string // 可选白名单：只挑选这些工具（nil=全部非 SystemTool 工具）
	// binary 指定 execute 调度形态（2026-08-16 第三轮：全部独立二进制）：
	//   ""     → ctx.hostTool.exec（宿主内执行器，会话/宿主状态依赖——tool-system）
	//   "self" → ctx.binary.exec（本插件目录 bin/<插件名>.exe，独立二进制；
	//            源码 plugins-src/plugins/tool-<组>/，import agent 复用内置组实现）
	binary string
}

// genToolGroups 待生成工具组全表（15 复杂组 + tool-system 系统内部组；
// core/fs-search/web/shell 已手工迁移为 tool-core/tool-search/tool-web/tool-shell，
// 不在此列）。tool-system 内的 SystemTool 工具（update_tasks/update_plan/
// tool_stats/history_*）对 LLM 可见但前端 UI 隐藏，同样外置可更换；
// ask_user/task_create 已插件化（2026-08-16 会话桥机制，见 session_bridge.go：
// JS 包装经 _convID 路由回宿主 SessionBridge，非「不可外置」）。
func genToolGroups() []genToolGroup {
	return []genToolGroup{
		{"tool-git", "Git 操作（git_status/diff/log/show/blame/add/commit/…）", registerGitTools, nil, "self"},
		{"tool-memory", "跨会话记忆（memory_write/read/list/search）", registerMemoryTools, nil, "self"},
		{"tool-verify", "知识库过期验证（memory_verify/project_info_verify）", registerVerifyTools, nil, "self"},
		{"tool-project-info", "项目知识库（project_info_write/read/list/search/delete/explore）", registerProjectInfoTools, nil, "self"},
		{"tool-binary", "二进制读写 + 逆向分析（inspect_binary/write_binary/binary_strings/find/patch/info/hash/entropy，含 2026-08-16 并入的 tool-binary-re 逆向 6 工具）", registerBinaryTools, nil, "self"},
		{"tool-debug", "调试工具（debug_inject_log/run_capture/analyze_output/parse_stack/cleanup_logs/watch/evaluate_session）", registerDebugTools, nil, "self"},
		{"tool-vision", "图像视觉（image_analyze/image_ocr）", registerVisionTools, nil, "self"},
		{"tool-screenshot", "截图（screenshot_desktop/window/area/webpage）", registerScreenshotTools, nil, "self"},
		{"tool-web-debug", "网页验证（web_debug）", registerWebDebugTool, nil, "self"},
		{"tool-bug", "BUG 检测与修复（bug_detect/bug_analyze/bug_fix）", RegisterBugTools, nil, "self"},
		{"tool-office", "办公文档（csv_read/csv_write/json_to_table/table_stats/text_report/word_read）", registerOfficeTools, nil, "self"},
		{"tool-lsp", "LSP 代码导航（lsp_definition/references/hover/diagnostics）", registerLSPTools, nil, "self"},
		{"tool-codegraph", "代码知识图谱（codegraph_build/search/impact/…）", registerCodeGraphTools, nil, "self"},
		{"tool-codegraph-extra", "图谱扩展（codegraph_find_by_signature/explore）", registerExtraCodeGraphTools, nil, "self"},
		// tool-system：SystemTool 内部工具 + Skills/MCP/市场/提交信息
		// （ask_user/task_create 经会话桥插件化，见 session_bridge.go）
		{"tool-system", "系统内部工具（SystemTool + Skills/MCP/市场/提交信息：update_tasks/update_plan/tool_stats/history_*/skill_*/mcp_*/marketplace_*/generate_commit_message）——全部可更换",
			func(r *Registry, root string) {
				RegisterManagementTools(r, root)
				registerPlanTool(r)
				registerToolStatsTool(r)
				registerTaskTools(r, root)
				RegisterCommitMessageTool(r)
			},
			[]string{
				"update_tasks", "update_plan", "tool_stats",
				"history_search", "history_list", "history_count",
				"skill_list", "load_skill", "load_skill_resource", "skill_write", "skill_delete",
				"mcp_list", "mcp_add", "mcp_remove",
				"marketplace_search", "marketplace_install",
				"generate_commit_message",
			},
			"",
		},
	}
}

// genToolDef JS 侧工具定义（对齐 jsToolToGo 的 json 字段，omitempty 保持整洁）。
type genToolDef struct {
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	UsageGuide       string         `json:"usageGuide,omitempty"`
	Category         string         `json:"category,omitempty"`
	Parameters       map[string]any `json:"parameters"`
	ReadOnly         bool           `json:"readOnly,omitempty"`
	RequiresApproval bool           `json:"requiresApproval,omitempty"`
	SystemTool       bool           `json:"systemTool,omitempty"`
}

// collectGroupTools 对一组调用注册函数，收集可外置的工具定义（排除系统内部工具）。
func collectGroupTools(g genToolGroup, root string) ([]genToolDef, error) {
	r := NewRegistry()
	g.apply(r, root)
	want := map[string]bool{}
	for _, n := range g.names {
		want[n] = true
	}
	defs := make([]genToolDef, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		if t == nil {
			continue
		}
		if len(want) > 0 {
			if !want[name] {
				continue // 白名单模式：跳过名单外工具
			}
		} else if t.SystemTool {
			continue // 非白名单模式：跳过系统内部工具（不暴露给 LLM，不迁移）
		}
		defs = append(defs, genToolDef{
			Name:             t.Name,
			Description:      t.Description,
			UsageGuide:       t.UsageGuide,
			Category:         t.Category,
			Parameters:       t.Parameters,
			ReadOnly:         t.ReadOnly,
			RequiresApproval: t.RequiresApproval,
			SystemTool:       t.SystemTool,
		})
	}
	if len(defs) == 0 {
		return nil, fmt.Errorf("组 %s 无可外置工具（全部 SystemTool 或未注册）", g.plugin)
	}
	return defs, nil
}

// buildPluginJS 组装磁盘插件源码（对齐 tool-core 模板：api 声明 + hostTool 编排）。
func buildPluginJS(g genToolGroup, defs []genToolDef) (string, error) {
	raw, err := json.MarshalIndent(defs, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化 %s 工具定义: %w", g.plugin, err)
	}
	names := make([]string, 0, len(defs))
	for _, d := range defs {
		names = append(names, d.Name)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "// ═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(&b, "// %s — %s\n//\n", g.plugin, g.desc)
	fmt.Fprintf(&b, "// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go\n")
	execLine := "execute: (args) => ctx.hostTool.exec(t.name, args || {}),"
	execDoc := "execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。"
	switch g.binary {
	case "self":
		execLine = "execute: (args) => ctx.binary.exec(t.name, args || {}),"
		execDoc = "execute 调 ctx.binary 复用本插件目录 bin/ 下的独立二进制（源码 plugins-src/plugins/<name>/，改实现重编译即更换）。"
	case "tool-binary":
		execLine = "execute: (args) => ctx.binary.exec(t.name, args || {}, {bin: 'tool-binary'}),"
		execDoc = "execute 调 ctx.binary 复用统一宿主二进制（.pair/plugins/tool-binary/bin/，源码 plugins-src/plugins/tool-binary/，承载全部内置工具组实现）。"
	}
	fmt.Fprintf(&b, "// 自动生成，schema 完整外置拷贝）。api 声明在插件，%s\n", execDoc)
	fmt.Fprintf(&b, "// 工具清单：%s\n", strings.Join(names, "、"))
	fmt.Fprintf(&b, "// ═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(&b, "const tools = %s;\n\n", raw)
	fmt.Fprintf(&b, "return {\n")
	fmt.Fprintf(&b, "  name: '%s',\n", g.plugin)
	fmt.Fprintf(&b, "  purpose: '%s（自动生成，迁移自内置 Go 工具组）',\n", g.desc)
	fmt.Fprintf(&b, "  apply(ctx) {\n")
	fmt.Fprintf(&b, "    for (const t of tools) {\n")
	fmt.Fprintf(&b, "      ctx.tools.register({\n")
	fmt.Fprintf(&b, "        name: t.name,\n")
	fmt.Fprintf(&b, "        description: t.description,\n")
	fmt.Fprintf(&b, "        usageGuide: t.usageGuide,\n")
	fmt.Fprintf(&b, "        category: t.category,\n")
	fmt.Fprintf(&b, "        readOnly: t.readOnly,\n")
	fmt.Fprintf(&b, "        requiresApproval: t.requiresApproval,\n")
	fmt.Fprintf(&b, "        systemTool: t.systemTool,\n")
	fmt.Fprintf(&b, "        parameters: t.parameters,\n")
	fmt.Fprintf(&b, "        %s\n", execLine)
	fmt.Fprintf(&b, "      })\n")
	fmt.Fprintf(&b, "    }\n")
	fmt.Fprintf(&b, "  },\n")
	fmt.Fprintf(&b, "}\n")
	return b.String(), nil
}

// GenerateToolPlugins 生成全部未外置工具组的磁盘插件（幂等，仅重写有差异文件）。
// outDir 为 .pair/plugins 目录；每个插件包写 index.js + package.json
// （LoadGlobalPlugins 按 package.json 扫描装载）。返回写入的文件列表。
func GenerateToolPlugins(root, outDir string) ([]string, error) {
	var written []string
	for _, g := range genToolGroups() {
		defs, err := collectGroupTools(g, root)
		if err != nil {
			return written, err
		}
		js, err := buildPluginJS(g, defs)
		if err != nil {
			return written, err
		}
		dir := filepath.Join(outDir, g.plugin)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return written, fmt.Errorf("创建 %s: %w", dir, err)
		}
		// ① index.js（host 半源码）
		f := filepath.Join(dir, "index.js")
		if err := os.WriteFile(f, []byte(js), 0o644); err != nil {
			return written, fmt.Errorf("写入 %s: %w", f, err)
		}
		written = append(written, f)
		// ② package.json（装载清单：LoadGlobalPlugins 按此扫描）
		pkg := map[string]any{
			"name":    g.plugin,
			"purpose": g.desc + "（自动生成，迁移自内置 Go 工具组）",
			"version": "1.0.0",
			"scope":   "global",
			"type":    "plugin",
			"main":    "index.js",
		}
		pkgJSON, err := json.MarshalIndent(pkg, "", "  ")
		if err != nil {
			return written, fmt.Errorf("序列化 %s package.json: %w", g.plugin, err)
		}
		pf := filepath.Join(dir, "package.json")
		if err := os.WriteFile(pf, append(pkgJSON, '\n'), 0o644); err != nil {
			return written, fmt.Errorf("写入 %s: %w", pf, err)
		}
		written = append(written, pf)
	}
	return written, nil
}
