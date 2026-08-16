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
// harness 对齐模式下返回空（skill_*/mcp_*/marketplace_* 工具已被禁用）。
func SelfManagementPrompt() string {
	if HarnessOnlyTools() {
		return ""
	}
	return "" +
		"\n\n# 自管理与扩展\n" +
		"你可自我扩展：skill_list/load_skill/skill_write/skill_delete 管理技能；" +
		"mcp_list/mcp_add/mcp_remove 管理 MCP；marketplace_search/marketplace_install 安装扩展。" +
		"可复用流程沉淀为技能。\n" +
		"技能 mode：auto（按需激活）/ always（始终激活）/ manual（仅手动 load_skill）。" +
		"skill_write 时指定。\n" +
		"memory_verify/project_info_verify 可检查记忆和知识库是否过时。\n" +
		"★ 会话连贯性（最高优先级——收到任务后第一件事）：" +
		"系统已主动注入「会话连贯性上下文」（任务进度、对话摘要、相关记忆、项目归属、工作区结构、" +
		"Git 变更感知、代码图谱统计）。" +
		"你在收到任务后必须先理解这些信息：看清楚当前项目中哪些文件已变更、任务进度在哪、之前干过什么、" +
		"有哪些相关记忆——然后直接继续推进，严禁从零开始重新分析项目。" +
		"如果上下文中有「最近提交」，说明项目代码已有变更，应先确认 codegraph 是否需要重建。"
}

// LongTermMemoryPrompt 返回"长时记忆检索"段落的系统提示文本。
// ★ v3：强化为主动指令。告知 Agent 系统已主动注入了什么，以及当注入不足时该做什么。
// harness 对齐模式下返回空（memory_*/project_info_* 工具已被禁用）。
func LongTermMemoryPrompt() string {
	if HarnessOnlyTools() {
		return ""
	}
	return "" +
		"\n\n# 长时记忆检索\n" +
		"系统已在会话连贯性上下文中主动注入了：任务进度、对话摘要、相关记忆（自动召回）、" +
		"项目归属推断、工作区结构、Git 变更感知、代码图谱统计。\n" +
		"你首先应该消化这些已注入的信息，然后可用以下工具补充检索：\n" +
		"- `memory_search` 搜索历史记忆（标题/摘要/标签/关键点），按关键词筛选\n" +
		"- `memory_list` 列出所有历史记忆（按完成时间倒序）\n" +
		"- `memory_count` 查询记忆总数\n" +
		"★ 注意：新对话开始时系统已自动召回与当前任务相关的记忆（基于关键词+对话摘要+历史消息）。" +
		"如果自动召回结果不足以理解当前状态，再主动调用上述工具。\n" +
		"历史对话记录本身是精简版（旧轮次已压缩为摘要），完整上下文在记忆系统中。" +
		"项目的完整结构理解在项目知识库（project_info_*）中，代码在代码图谱（codegraph_*）中。"
}

// ─── 工具描述辅助 ─────────────────────────────────────────

// ToolDescription 返回指定工具的简短描述文本。
// 供工具自动生成系统提示中的工具列表用（如 loop.go 中的 DefaultSystemPrompt 末尾的工具列表）。
func ToolDescription(name, desc string) string {
	return fmt.Sprintf("- %s：%s", name, desc)
}
