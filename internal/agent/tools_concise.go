package agent

// ═══════════════════════════════════════════════════════════════
// tools_concise.go — 工具描述文言文精简（发送给 LLM 前替换 description）
//
// 背景（2026-08-17 实测）：DeepSeek 上下文缓存不覆盖 tools 参数——
// 工具定义（35 个 ≈6200 token）每次请求恒 miss。精简 description
// 文本体积，直接减少每次请求的 miss 量（prompt 成本下降、有效命中率上升）。
//
// 原则：
//   - 保留语义核心：工具用途 + 关键参数 + 行为差异（必填项、警告）
//   - 文言文简洁风格（信息密度高、字少）
//   - 未收录的工具保留原描述；name/parameters 一律不变
// ═══════════════════════════════════════════════════════════════

// conciseToolDescriptions 工具名 → 文言文精简描述。
var conciseToolDescriptions = map[string]string{
	// ── 会话 / 历史 ──
	"history_search": "搜已毕对话之史（题/摘/标/要）。",
	"history_list":   "列已毕对话（按完成时倒序）。",
	"history_count":  "计已毕对话之数。",

	// ── 计划 / 任务 ──
	"update_plan":  "维护任务计划清单（全量重传）：每步 step+status（pending/in_progress/done）；复杂任务先立之。",
	"update_tasks": "维护任务清单（全量替换）：subject 必填、status 定态（pending/in_progress/completed/cancelled），可附 description/dependencies/plan_step_index。",
	"task_create":  "建子任务；立后即 task_update 标 in_progress 执行，毕则标 completed。",
	"tool_stats":   "观工具调用统计（次数/成败/成功率）；min_calls 滤低频、recent 观近录。",

	// ── 文件编辑 ──
	"str_replace_editor": "观/建/改文件：view 览（带行号，可 view_range 限行）、create 建（file_text）、str_replace 换（old_str 须唯一精确）、insert 行后插（insert_line）。",
	"read":               "读文件内容（工作区内 path；可 offset+limit 读片段，缺省读全，超 2000 行截断）。",
	"write":              "写 content 至 path（覆盖，父目录自动建）；需审核批准。",
	"edit":               "以 new_string 换文中唯一 old_string（智能匹配 CRLF/空白；败则用 line_start/line_end 定位）。",

	// ── 代码执行 / 命令 ──
	"run_code":       "执行代码片段（auto/go/python/node）并返输出。",
	"bash":           "同步执行一条 shell 命令返输出（独立 shell 无状态）；禁用于长驻进程（用 run_background）。",
	"run_background": "后台启动长命令（dev server/watch 等）返进程 id；以 read_output 读输出、kill_process 止。",
	"read_output":    "读后台进程（id）累积输出与运行状态。",
	"kill_process":   "止后台进程（id）；仅限 run_background 所启，不能动外部进程。",

	// ── 搜索 ──
	// ★ Round3：search_content/search_files 旧名注册已删除（并入 glob/grep），死条目随删
	"glob": "按通配符递归找文件返相对路径；含 / 或 ** 按路径模式，否则按文件名；path 限子目录；无 pattern 时列目录（目录在前）。",
	"grep": "以 RE2 正则搜文件内容返「路径:行号: 行」；path/glob/case_insensitive 可限。",

	// ── 网络 ──
	"web_fetch":  "抓取 http(s) 网页返纯文本（去标签，超长截断）。",
	"web_search": "搜网返题/链/摘（SearXNG 优先，否则 DuckDuckGo）。",

	// ── 记忆 ──
	"memory_write":  "写或更持久记忆（跨会话存 .pair/memory/）；先查有无同名，有则融合更新，勿碎片化。",
	"memory_delete": "删过时/错误记忆（按 name）。",
	"memory_read":   "按 name 读记忆全文。",
	"memory_list":   "列记忆总览（名+摘要）。",
	"memory_search": "按关键词搜记忆（名/摘要/正文）返名+摘要。",

	// ── 插件 cordis ──
	"cordis_inspect":       "观插件运行时：无 id 摘要、id 版本链、id+version=vN 源码与诊断。",
	"cordis_define":        "登记 JS/TS 动态插件（预检不运行）：code 为 async 函数体，可 ctx.tools.register/systemPrompt/on/provide；含 client 半自动 global；scope 定 project/global；pluginId 非空则追加版本。",
	"cordis_run":           "装载已登记插件（id 或 pluginId）：goja 求值并 apply(ctx, config)；重复 run 先卸旧再装新。",
	"cordis_stop":          "停止运行中插件，回收其工具/提示/监听；定义保留可再 run。",
	"cordis_undefine":      "删插件定义（先停后忘）；删后不可再 run。",
	"cordis_service_list":  "列宿主服务与方法签名（写插件先查；静态服务 ctx.xxx 访问，动态 ctx.get）。",
	"cordis_inspect_query": "按协议查询插件运行时（platform=host 只读）：provider=service/tool/event/plugin，method=list*/get*。",

	// ── 工具集 ──
	"toolset_build":  "动态构建工具集：分析项目 → 模板组合生成插件 → 装载固化 .pair/toolsets/{name}.json；overwrite=true 覆盖。",
	"toolset_edit":   "编辑工具集：add_plugin 加插件（tools 白名单）、rm_plugin 移除、rm_tool 摘工具、enable_tool 恢复；name=builtin 时 add_builtin(组)/add_builtin_all/rm_plugin(builtin:组)。",
	"toolset_list":   "列工作区全部工具集（名/用途/插件数/来源）。",
	"toolset_show":   "观工具集详情（插件清单/来源/版本）；name=builtin 观内置分组与启用状态。",
	"toolset_export": "导出工具集为可移植 JSON（to 写文件或返回内容）。",
	"toolset_import": "导入工具集（json 内容或 file 路径；scope=user 全局/project 工作区）。",
	"toolset_remove": "删工具集（scope 指定作用域；builtin 不可删）。",

	// ── 其他 ──
	"ask_user": "向用户提问等答（关键决策/歧义澄清，勿滥用）；question 必填（或 questions 数组多问题），askType 定 text/single/multi/single-with-input；多问题传 questions:[{id,question,options?,multi_select?}]。",
}

// ApplyConciseToolDescriptions 精简工具描述：深拷贝并替换 description 为文言文精简版。
// 未收录的工具保留原描述；name/parameters 不变。
func ApplyConciseToolDescriptions(tools []ToolDefinition) []ToolDefinition {
	out := make([]ToolDefinition, len(tools))
	for i, t := range tools {
		out[i] = t
		if d, ok := conciseToolDescriptions[t.Function.Name]; ok && d != "" {
			out[i].Function.Description = d
		}
	}
	return out
}
