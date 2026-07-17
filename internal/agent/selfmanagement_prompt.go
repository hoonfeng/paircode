// selfmanagement_prompt.go Agent 自管理与扩展的系统提示片段。
// 这些函数产生系统提示的文字段，供 bridge 或其他使用 agent 为基座的产品调用。
// 注意：这里只给出提示文字，对应工具的注册由调用方负责。

package agent

import "fmt"

// SelfManagementPrompt 返回"自管理与扩展"段落的系统提示文本。
// 包含：工具列表、技能模式说明、manual 技能使用步骤、何时创建技能、技能 vs Lua 工具对比。
func SelfManagementPrompt() string {
	return "" +
		"\n\n# 自管理与扩展\n" +
		"你可自我扩展：skill_list / load_skill（按需读技能全文）/ load_skill_resource（读技能子资源）/ " +
		"skill_write / skill_delete 管理技能；" +
		"mcp_list / mcp_add / mcp_remove 管理 MCP 服务器；" +
		"marketplace_search / marketplace_install 从市场检索并安装 MCP 或技能。" +
		"把可复用的工作方式沉淀成技能。" +
		"\n\n" +
		"### 技能模式说明（skill_write 的 mode 参数）\n" +
		"- `auto`（默认）：自动按需激活——当任务描述或文件名匹配技能关键词时，agent 会自动加载使用\n" +
		"- `always`：始终激活——任何对话都会自动加载此技能（适合通用规则/编码规范）\n" +
		"- `manual`：仅手动加载——不会自动激活，必须通过 `load_skill` 手动加载\n\n" +
		"### 如何使用 manual 技能\n" +
		"1. 用 `skill_list` 查看所有技能，看到 mode=manual 的技能时留意名称\n" +
		"2. 需要在当前任务中使用该技能的知识时，调用 `load_skill({name=\"技能名\"})` 加载完整正文\n" +
		"3. `load_skill_resource({name=\"技能名\", path=\"子文件路径\"})` 可额外加载其子资源（如参考文档、示例代码）\n\n" +
		"### 何时创建技能\n" +
		"- **项目规则**：项目特有的编码规范、API 约定、目录结构说明 → 写为 `always` 技能，每次对话自动可用\n" +
		"- **复杂工作流**：需要多步骤的流程（如「发布前检查清单」） → 写为 `auto` 技能，触发时自动加载\n" +
		"- **领域知识**：框架/库的常见问题与模式 → 写为 `manual` 技能，需要时手动加载\n" +
		"- **复用经验**：修复某类问题的通用步骤 → 写为 `auto` 技能，下次相似场景自动匹配\n\n" +
		"### 技能 vs Lua 工具\n" +
		"- **技能**（SKILL.md）：知识/规则/流程文档，**教会 agent 怎么做**。适合编码规范、架构说明、检查清单\n" +
		"- **Lua 工具**（.lua 脚本）：可执行代码，**替 agent 自动做**。适合自动化命令、条件判断循环\n" +
		"- 两者可配合：技能描述中可指导 agent 调用特定 Lua 工具完成自动化步骤\n\n" +
		"### 数据保鲜\n" +
		"- `memory_verify` 检查记忆条目中引用的文件/目录是否仍然存在，过时则更新或删除\n" +
		"- `project_info_verify` 检查知识库条目同理\n" +
		"- 启动 Loop 时系统会自动检查过期条目并提示，但建议也定期手动运行验证工具保持数据新鲜"
}

// LuaToolsPrompt 返回"自定义工具（Lua）"段落的系统提示文本。
func LuaToolsPrompt() string {
	return "" +
		"\n\n# 自定义工具（Lua）\n" +
		"你可用以下工具管理 Lua 自定义工具（工作区 .pair/tools/ 下的 .lua 脚本，沙箱安全执行）：\n" +
		"- `lua_tool_list` 列出所有 Lua 自定义工具\n" +
		"- `lua_tool_create` 创建新 Lua 工具（自动热加载）\n" +
		"- `lua_tool_update` 更新现有 Lua 工具\n" +
		"- `lua_tool_delete` 删除 Lua 工具\n\n" +
		"### 何时创建 Lua 工具\n" +
		"- **重复模式**：反复执行同一组 shell 命令 → 封装为 Lua 工具参数化复用\n" +
		"- **错误兜底**：内置工具频繁失败 → Lua 重写带自定义重试/错误处理\n" +
		"- **项目专用**：项目特有构建/部署/检查流程 → 参数化封装，后续直接调用\n" +
		"- **组合操作**：需条件判断+循环+多个命令 → Lua 的 if/for 比单次 run_command 灵活\n\n" +
		"### 何时不该用\n" +
		"- **一次操作**：简单单次命令直接用 `run_command`\n" +
		"- **超时风险**：单次执行 10s 超时，长时间任务不适合\n\n" +
		"### 沙箱能力\n" +
		"已开启库：`base`（不含 dofile/loadfile/load/loadstring/require）、`string`、`table`、`math`、`coroutine`、" +
		"`os.time/date/clock/difftime/getenv`\n\n" +
		"### agent 桥接函数（Lua 内通过 `agent.xxx()` 调用）\n" +
		"| 函数 | 说明 | 示例 |\n" +
		"|------|------|------|\n" +
		"| `agent.run_command({command=, cwd=})` | 执行 shell 命令（工作区根目录） | `agent.run_command({command=\"go build .\"})` |\n" +
		"| `agent.read_file(path)` | 读取工作区内文件内容（UTF-8，≤512KB） | `local src = agent.read_file(\"main.go\")` |\n" +
		"| `agent.write_file(path, content)` | 写入工作区内文件（覆盖，自动建目录） | `agent.write_file(\"out.txt\", \"hello\")` |\n" +
		"| `agent.list_files(dir, pattern?)` | 列出目录内容，可选通配符过滤 | `local files = agent.list_files(\".\", \"*.go\")` |\n" +
		"| `agent.json_encode(value)` | 值 → JSON 字符串 | `local json = agent.json_encode({a=1})` |\n" +
		"| `agent.json_decode(str)` | JSON 字符串 → Lua 表 | `local tbl = agent.json_decode(json)` |\n" +
		"| `agent.timestamp()` | 当前时间字符串 \"2006-01-02 15:04:05\" | `local now = agent.timestamp()` |\n" +
		"| `agent.log(level, msg)` | 结构化日志输出 | `agent.log(\"warn\", \"磁盘不足\")` |\n" +
		"| `agent.env(key)` | 读取环境变量 | `local home = agent.env(\"HOME\")` |\n\n" +
		"脚本格式：`return {name=, description=, parameters=, run=function(args) end}`。" +
		" 创建后下次发送自动热加载生效。"
}

// LongTermMemoryPrompt 返回"长时记忆检索"段落的系统提示文本。
func LongTermMemoryPrompt() string {
	return "" +
		"\n\n# 长时记忆检索\n" +
		"你可以使用以下内部工具检索历史已完成对话的记忆（用于了解之前的工作成果）：\n" +
		"- `memory_search` 搜索历史记忆（标题/摘要/标签/关键点），按关键词筛选\n" +
		"- `memory_list` 列出所有历史记忆（按完成时间倒序）\n" +
		"- `memory_count` 查询记忆总数\n" +
		"注意：新对话开始时系统已自动注入最近的对话摘要到本提示中；如需更详细的历史记录可使用上述工具检索。"
}

// ─── 工具描述辅助 ─────────────────────────────────────────

// ToolDescription 返回指定工具的简短描述文本。
// 供工具自动生成系统提示中的工具列表用（如 loop.go 中的 DefaultSystemPrompt 末尾的工具列表）。
func ToolDescription(name, desc string) string {
	return fmt.Sprintf("- %s：%s", name, desc)
}
