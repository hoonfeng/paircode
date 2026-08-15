// ═══════════════════════════════════════════════════════════════
// builtin_plugins.go — 内置插件装配（对齐 harness「一切皆插件」）
//
// 现有功能组（文件/搜索/Git/Web/记忆/任务/图谱…）全部以内置 Go 插件形态
// 装配：每个插件 { name, apply(ctx) }，apply 里经 ctx.Tools 注册一组工具。
//
// 双入口共享同一份规格（builtinPluginSpecs）：
//   - AgentBase.Init → registerBuiltinPlugins(ph)：经 PluginHost.Use 装配，
//     cordis_inspect 可见插件→工具归属，Unload 可回收
//   - RegisterDefaultTools(r, root)（测试/独立宿主）：直接 apply 到 Registry
// ═══════════════════════════════════════════════════════════════

package agent

import "fmt"

// builtinPluginSpec 一个内置插件规格。
type builtinPluginSpec struct {
	name string // 插件名（cordis_inspect 展示；同名注册冲突时报错）
	desc string // 插件用途
	apply func(c *PluginContext)
}

// builtinPluginSpecs 内置插件规格全表（顺序即装配顺序，core 最先）。
func builtinPluginSpecs(root string) []builtinPluginSpec {
	eh := newEditHistory() // ★ v2: 编辑行号偏移追踪器
	bg := globalBG         // ★ 全局共享后台进程注册表（跨轮次/跨 Registry 存活，见 shell.go）
	return []builtinPluginSpec{
		{"core", "文件读写/编辑/命令执行（read_file/write_file/edit_file/multi_edit/run_command/move_file/delete_file）",
			func(c *PluginContext) { registerCoreTools(c.Tools, root, eh, bg) }},
		{"fs-search", "全文/文件名搜索（search_content/search_files）",
			func(c *PluginContext) { registerSearchTools(c.Tools, root) }},
		{"git", "Git 操作（git_status/diff/log/show/blame/add/commit/…）",
			func(c *PluginContext) { registerGitTools(c.Tools, root) }},
		{"web", "联网（web_fetch/web_search）",
			func(c *PluginContext) { registerWebTools(c.Tools) }},
		{"shell", "后台命令（run_background/read_output/kill_process）",
			func(c *PluginContext) { registerShellTools(c.Tools, bg, root) }},
		{"memory", "跨会话记忆（memory_write/read/list/search）",
			func(c *PluginContext) { registerMemoryTools(c.Tools, root) }},
		{"verify", "知识库过期验证（memory_verify/project_info_verify）",
			func(c *PluginContext) { registerVerifyTools(c.Tools, root) }},
		{"task", "任务追踪（update_tasks）",
			func(c *PluginContext) { registerTaskTools(c.Tools, root) }},
		{"project-info", "项目知识库（project_info_write/read/list/search/delete/explore）",
			func(c *PluginContext) { registerProjectInfoTools(c.Tools, root) }},
		{"binary", "二进制读写（inspect_binary/write_binary）",
			func(c *PluginContext) { registerBinaryTools(c.Tools, root) }},
		{"binary-re", "二进制正则（binary_strings/find/patch/info/hash/entropy）",
			func(c *PluginContext) { registerBinaryRETools(c.Tools, root) }},
		{"debug", "调试工具（debug_inject_log/run_capture/analyze_output/parse_stack/cleanup_logs/watch/evaluate_session）",
			func(c *PluginContext) { registerDebugTools(c.Tools, root) }},
		{"vision", "图像视觉（image_analyze/image_ocr）",
			func(c *PluginContext) { registerVisionTools(c.Tools, root) }},
		{"screenshot", "截图（screenshot_desktop/window/area/webpage）",
			func(c *PluginContext) { registerScreenshotTools(c.Tools, root) }},
		{"web-debug", "网页验证（web_debug）",
			func(c *PluginContext) { registerWebDebugTool(c.Tools, root) }},
		{"bug", "BUG 检测与修复（bug_detect/bug_analyze/bug_fix）",
			func(c *PluginContext) { RegisterBugTools(c.Tools, root) }},
		{"office", "办公文档（csv_read/csv_write/json_to_table/table_stats/text_report/word_read）",
			func(c *PluginContext) { registerOfficeTools(c.Tools, root) }},
		{"lsp", "LSP 代码导航（lsp_definition/references/hover/diagnostics）",
			func(c *PluginContext) { registerLSPTools(c.Tools, root) }},
		{"codegraph", "代码知识图谱（codegraph_build/search/impact/…）",
			func(c *PluginContext) { registerCodeGraphTools(c.Tools, root) }},
		{"codegraph-extra", "图谱扩展（codegraph_find_by_signature/explore）",
			func(c *PluginContext) { registerExtraCodeGraphTools(c.Tools, root) }},
		{"lua-tools", "Lua 自定义工具管理（lua_tool_list/create/update/delete）",
			func(c *PluginContext) { registerLuaToolTools(c.Tools, root) }},
	}
}

// RegisterBuiltinPlugins 把全部内置插件经 PluginHost 装配（AgentBase.Init 与 web 模式共用）。
// 之后 cordis_inspect 可查看插件→工具归属；停止某插件可回收其工具。
func RegisterBuiltinPlugins(h *PluginHost) {
	_ = h.Use(&GoPlugin{
		NameField: "sysinfo",
		ApplyFn: func(ctx *PluginContext) error {
			ctx.Provide("workspaceRoot", ctx.WorkspaceRoot)
			return nil
		},
	})
	for _, s := range builtinPluginSpecs(h.ctx.WorkspaceRoot) {
		spec := s // 闭包捕获
		if err := h.Use(&GoPlugin{
			NameField: spec.name,
			ApplyFn: func(ctx *PluginContext) error {
				spec.apply(ctx)
				return nil
			},
		}); err != nil {
			// 同名插件冲突（理论上内置名唯一）——记录但不中断装配
			_ = fmt.Errorf("内置插件 %s 装配失败: %w", spec.name, err)
		}
	}
	// ★ 工具集构建模板插件（toolset_build 的动态组合数据源，本身可被市场/用户扩展）
	if err := h.Use(registerToolsetTemplates()); err != nil {
		_ = fmt.Errorf("内置工具集模板插件装配失败: %w", err)
	}
}
