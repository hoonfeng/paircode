package agent

// ── 首步极简工具面（Staged Tools）────────────────────────────────────
//
// ★ 2026-08-27 实测驱动改进：
//   对比实验（16 任务 × 3 轮 = 48 次采样，deepseek-v4-flash）：
//   全量工具面(104) 首步选对率 87.5%（精确 77%），极简面(6) 91.7%（精确 89.6%）。
//   结论：工具越少选择越准（全量面被语义相近工具干扰：text_report/search_files 迷惑）。
//
// 设计（对齐参考设计）：**首次详细（会话第一个 Run 的首个 LLM 调用）只注入
// 极简核心工具面；自第 2 个 step 起恢复完整工具面**。首步以探索/理解为目标，
// 后续恢复全部能力——提升首步工具选择准确率与编排质量，同时减少首步 token 开销。
//
// ★ 2026-08-27 插件注册化（配置经插件装配链，不走 Settings 顶层）：
//   候选组由 agentloop 插件 registerSettings 注册（settings.json
//   pluginSettings["agentloop"]["stagedToolGroups"]，textarea 每行一组、组内
//   逗号分隔）→ 装配器解析为 [][]string → Loop.StagedToolGroups；
//   未配置/为空 → 回退内核默认组 defaultStagedGroups。
//
// 命名适配：默认组用「候选组」表达（每组语义相同，取面中存在的第一个名字）：
//   - 宿主 harness 面（tool-harness 磁盘插件）：read/grep/glob/bash/write/edit/
//     str_replace_editor/run_code
//   - 内置组面（RegisterDefaultTools）：read_file/search_content/search_files/
//     list_files/run_command/write_file/edit_file/multi_edit
// 两组各命中 8/8 与 7/8，极简面维持「读/搜/找/命令/写/编辑」基础闭环。

// defaultStagedGroups 默认极简工具候选组（插件未配置时的回退值）。
// ★ Round3：以下为「历史映射」等价表——左列旧名（read_file/run_command 等）仅作
//   staging 对历史消息/旧配置的兼容匹配，右列新名为当前注册名；Go 注册面已零旧名。
var defaultStagedGroups = [][]string{
	{"read_file", "read"},                  // 读文件
	{"search_content", "grep"},             // 内容搜索
	{"search_files", "list_files", "glob"}, // 找/列文件
	{"run_command", "bash"},                // 命令执行
	{"write_file", "write"},                // 写文件
	{"edit_file", "edit"},                  // 单点编辑
	{"multi_edit", "str_replace_editor"},   // 批量/结构化编辑
	{"run_code"},                           // 代码执行（host 专用）
}

// DefaultStagedGroups 返回内置默认极简候选组（供装配/测试查询）。
func DefaultStagedGroups() [][]string { return defaultStagedGroups }

// FilterStagedTools 首步极简工具面过滤：groups 非空用之，为空回退默认组。
// （JT 后续步骤恢复全量不调用。）
func FilterStagedTools(tools []ToolDefinition, groups [][]string) []ToolDefinition {
	return FilterMinimalToolsWith(tools, groups)
}

// FilterMinimalTools 按默认候选组过滤极简工具面（无 Loop 场景/测试用）。
func FilterMinimalTools(tools []ToolDefinition) []ToolDefinition {
	return filterMinimal(tools, defaultStagedGroups)
}

// FilterMinimalToolsWith 按给定候选组过滤：groups 为空回退默认组。
func FilterMinimalToolsWith(tools []ToolDefinition, groups [][]string) []ToolDefinition {
	if len(groups) == 0 {
		groups = defaultStagedGroups
	}
	return filterMinimal(tools, groups)
}

// filterMinimal 按候选组过滤工具面：每个候选组取面中第一个存在的名字。
func filterMinimal(tools []ToolDefinition, groups [][]string) []ToolDefinition {
	if len(tools) == 0 {
		return tools
	}
	exist := make(map[string]bool, len(tools))
	for _, d := range tools {
		exist[d.Function.Name] = true
	}
	out := make([]ToolDefinition, 0, len(groups))
	for _, group := range groups {
		for _, name := range group {
			if exist[name] {
				for _, d := range tools {
					if d.Function.Name == name {
						out = append(out, d)
						break
					}
				}
				break
			}
		}
	}
	// 工具面为空兜底：不启用极简（返回原工具面，避免模型无工具可用）。
	if len(out) == 0 {
		return tools
	}
	return out
}
