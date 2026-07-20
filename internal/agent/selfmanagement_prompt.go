// selfmanagement_prompt.go Agent 自管理与扩展的系统提示片段。
// 这些函数产生系统提示的文字段，供 bridge 或其他使用 agent 为基座的产品调用。
// 注意：这里只给出提示文字，对应工具的注册由调用方负责。

package agent

import "fmt"

// SelfManagementPrompt 返回"自管理与扩展"段落的系统提示文本。
func SelfManagementPrompt() string {
	return "" +
		"\n\n# 自管理与扩展\n" +
		"你可自我扩展：skill_list/load_skill/skill_write/skill_delete 管理技能；" +
		"mcp_list/mcp_add/mcp_remove 管理 MCP；marketplace_search/marketplace_install 安装扩展。" +
		"可复用流程沉淀为技能。\n" +
		"技能 mode：auto（按需激活）/ always（始终激活）/ manual（仅手动 load_skill）。" +
		"skill_write 时指定。\n" +
		"memory_verify/project_info_verify 可检查记忆和知识库是否过时。\n" +
		"★ 会话连贯性：系统已主动注入「会话连贯性上下文」（任务进度、对话摘要、相关记忆、项目归属、工作区结构）。" +
		"你应该在收到任务后首先阅读这些信息，了解当前状态，而不是从零开始分析。"
}

// LuaToolsPrompt 返回"自定义工具（Lua）"段落的系统提示文本。
func LuaToolsPrompt() string {
	return "" +
		"\n\n# 自定义工具（Lua）\n" +
		"工作区 .pair/tools/ 下的 .lua 脚本（沙箱安全执行）：" +
		"lua_tool_list/lua_tool_create/lua_tool_update/lua_tool_delete 管理。\n" +
		"脚本格式 `return {name=, description=, parameters=, run=function(args) end}`。\n" +
		"Lua 内可调 agent.run_command/read_file/write_file/list_files/json_encode/json_decode/timestamp/log/env。"
}

// LongTermMemoryPrompt 返回"长时记忆检索"段落的系统提示文本。
// ★ v2：强化为主动指令，而非被动告知。Agent 必须在每次新对话/新任务开始时主动召回记忆。
func LongTermMemoryPrompt() string {
	return "" +
		"\n\n# 长时记忆检索（主动召回——强制要求）\n" +
		"系统已在会话连贯性上下文中自动注入了相关记忆和项目知识库目录。" +
		"此外，你可以主动使用以下工具：\n" +
		"- `memory_search` 搜索历史记忆（标题/摘要/标签/关键点），按关键词筛选\n" +
		"- `memory_list` 列出所有历史记忆（按完成时间倒序）\n" +
		"- `memory_count` 查询记忆总数\n" +
		"★ 强制规则：每次收到新任务时，如果系统注入的上下文不足以理解当前状态，" +
		"必须先调用 memory_search 或 memory_list 了解之前的工作，再动手。" +
		"不要凭记忆臆测项目状态——你看到的历史对话只是精简版，完整上下文在记忆系统中。"
}

// ─── 工具描述辅助 ─────────────────────────────────────────

// ToolDescription 返回指定工具的简短描述文本。
// 供工具自动生成系统提示中的工具列表用（如 loop.go 中的 DefaultSystemPrompt 末尾的工具列表）。
func ToolDescription(name, desc string) string {
	return fmt.Sprintf("- %s：%s", name, desc)
}
