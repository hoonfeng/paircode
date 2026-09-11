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

	// ── 任务 ──
	"update_tasks": "维护任务清单（全量替换）：subject 必填、status 定态（pending/in_progress/completed/cancelled），可附 description/dependencies。",
	"tool_stats":   "观工具调用统计（次数/成败/成功率）；min_calls 滤低频、recent 观近录。",

	// ── 文件编辑 ──
	"read":        "读文件内容；path 相对主项目根（跨项目传 project 或绝对路径），可 offset+limit 读片段，缺省读全（超 2000 行截断）。",
	"write":       "写 content 至 path（覆盖，父目录自动建）；path 相对主项目根（跨项目传 project 或绝对路径）；需审核批准。",
	"apply_patch": "应用 codex 语法补丁修改文件（*** Begin Patch/Add File/Update File/Delete File/Move to/*** End Patch）；一次多文件多操作，上下文行定位（免行号/免 JSON 转义）。",

	// ── 代码执行 / 命令 ──
	"run_code":     "执行代码片段（auto/go/python/node）并返输出。",
	"exec_command": "执行 shell 命令：短命令同步返输出（含退出码）；长进程等待 yield_time_ms 后返 session_id，以 write_stdin 轮询/写 stdin、kill_process 止。",
	"write_stdin":  "向会话进程（session_id）写 stdin（chars 空=仅轮询）并读增量输出（自上次读取起）。",
	"kill_process": "止会话进程（session_id）；仅限 exec_command 所启，已结束会话调用幂等无害。",

	// ── 搜索 ──
	// ★ Round3：search_content/search_files 旧名注册已删除（并入 glob/grep），死条目随删
	"glob": "按通配符递归找文件返相对路径；含 / 或 ** 按路径模式，否则按文件名；path 限子目录（相对主项目根，跨项目传 project 或绝对路径）；无 pattern 时列目录（目录在前）；自动跳过依赖/VCS/构建/IDE 运行数据目录。",
	"grep": "以 RE2 正则搜文件内容返「路径:行号: 行」；path/glob/case_insensitive 可限（path 相对主项目根，跨项目传 project 或绝对路径）；自动跳过依赖/VCS/构建/IDE 运行数据目录。",

	// ── 网络 ──
	"web_fetch":  "抓取 http(s) 网页返纯文本（去标签，超长截断）。",
	"web_search": "搜网返题/链/摘（SearXNG 优先，否则 DuckDuckGo）。",

	// ── 记忆 ──
	"memory": "跨会话记忆（.pair/memory/）单工具：op=write 写/更新（先查重同则融合，勿碎片化；需批准）/read/search/list/delete。",

	// ── 插件 cordis（单工具 op 分派）──
	"cordis": "动态插件单工具：op=inspect 观运行时（无 id 摘要/版本链/源码诊断）；define 登记 JS/TS（预检不运行；code 为 async 函数体，pluginId 非空追加版本）；run 装载（goja apply）；stop 停收；undefine 删定义；services 列宿主服务签名；query 协议查询（provider/method）。",

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
