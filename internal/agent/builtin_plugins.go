// ═══════════════════════════════════════════════════════════════
// builtin_plugins.go — 内置插件规格（对齐 harness「一切皆插件」）
//
// ★ 2026-08-16 第三轮：宿主不再承载工具实现。
//   - 内置 20 组（core/git/codegraph/…）的实现已迁移为磁盘插件
//     （.pair/plugins/tool-*，JS 原生化或独立插件二进制），宿主进程
//     不再注册——builtinPluginSpecs 保留仅作「二进制实现库的组规格」，
//     供独立插件二进制（plugins-src/plugins/tool-*/，经 pkg/toolbin）按组注册。
//   - 宿主只注册框架协议工具（RegisterHostFrameworkTools）：SystemTool
//     （update_tasks/update_plan/tool_stats/history_*）会话绑定，供
//     tool-system 插件 hostTool 承载（同名接管时 ArchiveHostTool 存档）。
//
// ★ 2026-08-16 第四轮：框架能力不再以插件形态存在。
//   - 原「框架能力插件」sysinfo（workspaceRoot 服务）+ toolset-tpl-core
//     （工具集构建模板）已内联为宿主固有（NewPluginHost 构造时直接提供：
//     h.ctx.Provide("workspaceRoot") + registerBuiltinTemplates(h)）——
//     不可启停、不出现在插件列表/Inspect 中，插件列表只含真实可管理的插件。
//
// 双入口共享同一份规格（builtinPluginSpecs）：
//   - 独立二进制：RegisterToolGroups(r, root, "git") 按组注册（plugins-src/plugins/*）
//   - 宿主框架：RegisterHostFrameworkTools（只注册 SystemTool 组）
// ═══════════════════════════════════════════════════════════════

package agent

// builtinPluginSpec 一个内置插件规格。
type builtinPluginSpec struct {
	name  string // 插件名（cordis_inspect 展示；同名注册冲突时报错）
	desc  string // 插件用途
	apply func(c *PluginContext)
}

// builtinPluginSpecs 内置插件规格全表（顺序即装配顺序，core 最先）。
// ★ 仅作二进制实现库的组规格（plugins-src/plugins/tool-*/ 经 RegisterToolGroups
//
//	按组注册）——宿主进程不再 apply 本表。
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
		{"binary", "二进制读写 + 逆向分析（inspect_binary/write_binary/binary_strings/find/patch/info/hash/entropy，含 2026-08-16 并入的 binary-re 逆向 6 工具）",
			func(c *PluginContext) { registerBinaryTools(c.Tools, root); registerBinaryRETools(c.Tools, root) }},
		{"debug", "调试工具（debug_inject_log/run_capture/analyze_output/parse_stack/cleanup_logs/watch/evaluate_session）",
			func(c *PluginContext) { registerDebugTools(c.Tools, root) }},
		{"screenshot", "截图（screenshot_desktop/window/area/webpage）",
			func(c *PluginContext) { registerScreenshotTools(c.Tools, root) }},
		{"web-debug", "网页验证（web_debug）",
			func(c *PluginContext) { registerWebDebugTool(c.Tools, root) }},
		{"bug", "BUG 检测与修复（bug_detect/bug_analyze/bug_fix）",
			func(c *PluginContext) { RegisterBugTools(c.Tools, root) }},
		{"office", "办公文档（csv_read/csv_write/json_to_table/table_stats/text_report/word_read）",
			func(c *PluginContext) { registerOfficeTools(c.Tools, root) }},
		{"codegraph", "代码知识图谱（codegraph_build/search/impact/…）",
			func(c *PluginContext) { registerCodeGraphTools(c.Tools, root) }},
		{"codegraph-extra", "图谱扩展（codegraph_find_by_signature/explore）",
			func(c *PluginContext) { registerExtraCodeGraphTools(c.Tools, root) }},
	}
}

// RegisterToolGroups 按组注册内置工具（groups 为空 = 全部）。供独立插件二进制
// （plugins-src/plugins/tool-*/，经 pkg/toolbin）与测试/示例使用——宿主进程不调用。
func RegisterToolGroups(r *Registry, root string, groups ...string) {
	want := map[string]bool{}
	for _, g := range groups {
		want[g] = true
	}
	for _, s := range builtinPluginSpecs(root) {
		if len(want) > 0 && !want[s.name] {
			continue
		}
		s.apply(&PluginContext{Tools: r})
	}
}

// RegisterDefaultTools 注册全部内置工具组（独立宿主/测试/示例用）。
// ★ 宿主进程（AgentBase.Init / web_server / desktopbridge）不再调用——
//
//	改用 RegisterHostFrameworkTools（工具实现已全部迁移磁盘插件）。
func RegisterDefaultTools(r *Registry, root string) {
	RegisterToolGroups(r, root)
}

// RegisterHostFrameworkTools 宿主框架协议工具注册（会话绑定 SystemTool）。
// 宿主进程（AgentBase.Init / web_server / desktopbridge）唯一的内置注册：
// update_tasks/update_plan/tool_stats/history_*——tool-system 磁盘插件
// hostTool 承载（同名接管时 ArchiveHostTool 存档 Go 实现供 ctx.hostTool）。
func RegisterHostFrameworkTools(r *Registry, root string) {
	RegisterManagementTools(r, root) // history_search/list/count 等
	registerPlanTool(r)              // update_plan
	registerToolStatsTool(r)         // tool_stats
	registerTaskTools(r, root)       // update_tasks（会话绑定 TaskManager）
}

// ★ 框架能力（workspaceRoot 服务 / 内置工具集模板）已内联 NewPluginHost
//   （plugin.go），不再以插件形态装配——本文件不再有 RegisterBuiltinPlugins。
