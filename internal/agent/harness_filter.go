package agent

// harness_filter.go — 对齐 deepseek-harness 工具注册（暂时移除 pair 独有工具）
//
// 背景：自举迭代（用 agent 开发 agent）要求 agent 暴露给 LLM 的工具集与
// deepseek-harness 对齐。默认进入 harness 对齐模式——只保留 harness 工具集
// + 对话协议基础设施，其余 pair 独有工具（codegraph_*/memory_*/project_info_*/
// git_*/debug_*/binary_*/office 等 130+ 个）暂时从注册表移除。
//
// ★ 开关：环境变量 WB_FULL_TOOLS=1 恢复全量工具集（关闭过滤）。
// ★ 幂等：可重复调用；必须在 LoadAllWorkspaceToolConfigs（.pair/tools.json）
//   之后调用——先应用工作区开关，再整体移除 pair 独有工具。

import "os"

// HarnessOnlyTools 判断是否处于 harness 对齐模式（默认开启，WB_FULL_TOOLS=1 关闭）。
func HarnessOnlyTools() bool {
	return os.Getenv("WB_FULL_TOOLS") == ""
}

// HarnessAlignedToolNames 保留清单（过滤时仅保留以下工具）：
//  ① harness 原生工具集：read/write/edit（tool-fs）、glob/grep（tool-fs-search）、
//     str_replace_editor、bash（tool-bash）、web_search/web_fetch（tool-web）、
//     run_code（code-mode）
//  ② 对话协议基础设施：update_tasks（任务追踪，前端任务面板依赖）、
//     ask_user（提问）、generate_commit_message（完成标记，Loop 读取 CommitMessage）——
//     属循环协议而非 pair 独有编码能力，保留以维持 agent 循环契约。
var HarnessAlignedToolNames = map[string]bool{
	// harness 原生工具集
	"read": true, "write": true, "edit": true,
	"glob": true, "grep": true,
	"str_replace_editor": true,
	"bash":               true,
	"web_search":         true, "web_fetch": true,
	"run_code": true,
	// 对话协议基础设施
	"update_tasks":           true,
	"ask_user":               true,
	"generate_commit_message": true,
}

// ApplyHarnessToolFilter 从注册表移除不在保留清单内的工具（pair 独有工具），
// 返回移除数量。开关关闭（WB_FULL_TOOLS=1）时不做任何事，返回 0。
// 使用 Unregister 反向过滤（而非 Subset 重建），保留钩子/CommitMessage 等注册表字段。
func ApplyHarnessToolFilter(r *Registry) int {
	if !HarnessOnlyTools() {
		return 0
	}
	removed := 0
	for _, m := range r.AllToolMeta() {
		if !HarnessAlignedToolNames[m.Name] {
			r.Unregister(m.Name)
			removed++
		}
	}
	return removed
}
