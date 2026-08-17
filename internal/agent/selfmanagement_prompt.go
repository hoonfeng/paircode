// selfmanagement_prompt.go Agent 自管理与扩展的系统提示片段。
// 这些函数产生系统提示的文字段，供 bridge 或其他使用 agent 为基座的产品调用。
// 注意：这里只给出提示文字，对应工具的注册由调用方负责。
//
// ★ harness 对齐（2026-08-15）：本文件引用的工具（skill_*/mcp_*/marketplace_*/
// memory_*/project_info_*）已被 ApplyHarnessToolFilter 禁用
// （Enabled=false——工具仍在注册表、前端可见，agent 不可调用；内置工具集
// builtin 分组开关/强制全部可恢复）。在 harness 对齐模式下（默认）这些段落
// 返回空串，避免提示 LLM 调用不可用的工具；WB_FULL_TOOLS=1 恢复全量工具时返回原文。

package agent

import "fmt"

// SelfManagementPrompt 返回"自管理与扩展"段落的系统提示文本。
// ★ 工具描述已取消（2026-08-17）：不再提示 skill_*/mcp_*/marketplace_*/memory_verify 等
//   具体工具——工具信息以 tools 参数 schema 为准。仅保留非工具性质的会话连贯性行为引导。
// harness 对齐模式下返回空。
func SelfManagementPrompt() string {
	if HarnessOnlyTools() {
		return ""
	}
	return "" +
		"\n\n# 会话连贯性（最高优先级——收到任务后第一件事）\n" +
		"系统已主动注入「会话连贯性上下文」（任务进度、对话摘要、相关记忆、项目归属、工作区结构、" +
		"Git 变更感知、代码图谱统计）。" +
		"你在收到任务后必须先理解这些信息：看清楚当前项目中哪些文件已变更、任务进度在哪、之前干过什么、" +
		"有哪些相关记忆——然后直接继续推进，严禁从零开始重新分析项目。" +
		"如果上下文中有「最近提交」，说明项目代码已有变更，应先确认代码图谱是否需要重建。"
}

// LongTermMemoryPrompt 返回"长时记忆检索"段落的系统提示文本。
// ★ 工具描述已取消（2026-08-17）：不再提示 memory_search/memory_list/memory_count 等
//   具体工具——工具信息以 tools 参数 schema 为准。仅保留机制说明。
// harness 对齐模式下返回空。
func LongTermMemoryPrompt() string {
	if HarnessOnlyTools() {
		return ""
	}
	return "" +
		"\n\n# 长时记忆检索\n" +
		"系统已在会话连贯性上下文中主动注入了：任务进度、对话摘要、相关记忆（自动召回）、" +
		"项目归属推断、工作区结构、Git 变更感知、代码图谱统计。\n" +
		"你首先应该消化这些已注入的信息；若注入不足以支撑当前任务，可用记忆检索类工具补充查询（工具名称与用法见 tools 参数 schema）。\n" +
		"★ 注意：新对话开始时系统已自动召回与当前任务相关的记忆（基于关键词+对话摘要+历史消息）。" +
		"如果自动召回结果不足以理解当前状态，再主动查询补充。\n" +
		"历史对话记录本身是精简版（旧轮次已压缩为摘要），完整上下文在记忆系统中。" +
		"项目的完整结构理解在项目知识库中，代码在代码图谱中。"
}

// ─── 工具描述辅助 ─────────────────────────────────────────

// ToolDescription 返回指定工具的简短描述文本。
// 供工具自动生成系统提示中的工具列表用（如 loop.go 中的 DefaultSystemPrompt 末尾的工具列表）。
func ToolDescription(name, desc string) string {
	return fmt.Sprintf("- %s：%s", name, desc)
}
