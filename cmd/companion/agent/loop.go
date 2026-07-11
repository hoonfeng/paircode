package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrConsecToolError 连续多轮工具执行失败，由 Loop.Run 返回，桥接层应据此终止本轮后续阶段。
var ErrConsecToolError = errors.New("连续 3 轮工具执行失败，已停止")

// ErrCirclingLoop 绕圈检测连续触发多次，由 Loop.Run 返回。

var ErrCirclingLoop = errors.New("绕圈检测连续 3 次触发，仍在重复同一操作，已停止")
// ErrMaxIterations 已达最大迭代数仍未完成，由 Loop.Run 返回。
var ErrMaxIterations = errors.New("已达最大迭代数，停止")

// EventType 循环对外广播的事件类型（供 UI 流式展示）。
type EventType string

const (
	EventThinking   EventType = "thinking"    // LLM 思考链增量
	EventContent    EventType = "content"     // LLM 正文增量
	EventToolCall   EventType = "tool_call"   // 即将执行某工具
	EventToolResult EventType = "tool_result" // 工具结果回来
	EventFinal      EventType = "final"       // 任务完成（仅 delegate 单轮委托用；主 Loop 用 EventDone）
	EventError      EventType = "error"       // 出错/止损
	EventCompacted  EventType = "compacted"   // 上下文已压缩（中段老消息压成摘要；UI 显示一行素色提示）
	EventEvaluation EventType = "evaluation"  // 任务评测评分（完成后评测模型打分；UI 显示评分卡）
	EventCircling   EventType = "circling"    // 检测到重复绕圈，已注入「换思路」提示打破死循环（UI 显示一行提示）
	// EventApproval 等待用户审批某次写类工具调用。由宿主（UI 桥）在 Approve 钩子里 emit，
	// loop 自身不直接发——loop 只通过 Approve 回调阻塞等待裁决（见 agent_bridge.go）。
	EventUsage    EventType = "usage"     // LLM 调用完成后的 token 用量（含缓存命中/未命中）
	EventApproval EventType = "approval"

	EventNotice   EventType = "notice"    // 后台任务通知（jobs 包用；UI 显示一行素色提示）
	EventPhase    EventType = "phase"     // 阶段切换（自主模式下的规划/执行/评测等阶段指示）
	EventDone     EventType = "done"      // 结构化完成信号（供 delegate/子 agent 使用；主 Loop Exit 走此事件）
)

// Event 一条循环事件。
type Event struct {
	Type    EventType
	Content string // thinking/content/final/error/tool_result 的文本
	Tool    string // tool_call/tool_result 的工具名
	Args    string // tool_call 的参数 JSON
	CallID  string
	Usage   *Usage // EventUsage 时携带 API 返回的 token 用量
	// AgentName 事件来源 Agent 名。空串 = 父/主 Agent；非空 = 子 Agent（供前端区分）。
	AgentName string
	// DoneReason 完成原因（仅 EventDone 时设置）。
	// 取值："task_complete"（自然完成）、"finish_task"（调 finish_task 工具）。
	DoneReason string
}



// Loop TAOR 编排器：think(LLM 决策)→act(执行工具)→observe(结果回灌)→repeat。
// 停止：调 finish_task / 连续 3 轮工具全错 / 达最大迭代 / 外部取消。
type Loop struct {
	Provider      Provider
	Registry      *Registry
	System        string // 系统提示词
	MaxIterations int    // 默认 30
	OnEvent       func(Event)
	// Approve 审批钩子（可空）。设置后，每次执行 RequiresApproval 的写类工具前调用它，
	// 返回 (false, feedback) 即拒绝执行——feedback 非空则作为观察回灌（让模型据此改道），空则用默认拒绝语。
	// 只读工具永不经过它。nil = 自动审核（全部放行）。宿主可在此阻塞等用户点「允许/拒绝」(人工审核)，
	// 或调审核模型自动裁决并回灌建议(AI 审核)（见 agent_bridge.go）。
	Approve func(ctx context.Context, tc ToolCall) (approved bool, feedback string)

	// ── 上下文压缩（可空；复刻参考 context/manager.ts，见 compress.go）──
	// MaxContextTokens>0 时启用：每次 LLM 调用前，若 tokens/Max 超阈值，把中段老消息压成一条摘要。
	// Compressor 非空→用它（轻量压缩模型）做 LLM 摘要，否则/失败→规则式摘要。
	Compressor       Provider
	MaxContextTokens int

	lastPromptTokens int // 上一轮 API 实测 prompt_tokens（驱动压缩阈值，比纯估算可信）
	compactCooldown  int // 压缩后冷却剩余轮数（防每轮重复压缩，复刻参考 refreshCooldown）

	recentCalls []toolSig // 最近若干次工具调用签名+成败（绕圈检测，见 circling.go）

	// ── 多 agent 编排（阶段四，均可空；空=普通单 agent 模式）──
	AgentTree      *AgentTree     // agent 编排树（delegate_task/delegate_single_turn 用）
	State          map[string]any // 跨 agent 共享状态（子 Loop 继承父引用，避免塞进 messages 撑爆上下文）
	currentMsgs    []Message      // Run 期间当前消息列表（供 delegate handler 读父历史，保缓存前缀命中）
	finishResult   *string        // finish_task 退出信号（子 Loop：子 agent 调 finish_task 后置；delegate handler 据此取子结果）

	transferTarget string         // transfer_to_agent 目标名（非空=当前 Loop 应退出，控制权转移给目标 agent）
	Autonomous     bool           // 自主模式标志（供并行子 agent 继承）

	// History 跨 Run 调用的持久化对话消息（自闭环）。
	// 设计意图：Agent 独立维护自己的消息历史，前端只发信号（当前用户消息文本）。
	// 首次 Run 前为 nil；每次 Run 返回后更新为当轮完整 msgs。
	// Run 的 history 参数传 nil 时自动使用此持久化历史——前端无需自行构建/传递历史。
	// 传非 nil history 时仍保持向后兼容。
	History []Message
}

func (l *Loop) emit(e Event) {
	if l.OnEvent != nil {
		l.OnEvent(e)
	}
}

// Run 跑一轮任务。history 为先前对话（可空）。
//
// 自闭环模式：history 传 nil 时使用 l.History（持久化历史），前端只需传 task。
// 首次调用传 nil，Run 返回后 l.History 自动更新为当轮完整对话。
// 第二次调用再传 nil，自动使用上一轮保存的 l.History。
//
// 向后兼容：传非 nil history 时保持原行为（不更新 l.History）。
//
// 返回在 history/l.History 基础上追加了 system(首轮)/user/assistant/tool
// 等本轮全部消息的完整对话。
func (l *Loop) Run(ctx context.Context, task string, history []Message) (msgs []Message, err error) {
	// 自闭环：history 为 nil 时使用持久化的 l.History
	if history == nil {
		history = l.History
	}
	// 统一持久化出口：每次 Run 返回后更新 l.History（不论调用方是否传了 history）
	defer func() {
		l.History = msgs
	}()

	// 深复制 history，避免下层 append 污染原切片
	hist := CopyHistory(history)

	max := l.MaxIterations
	if max <= 0 {
		max = 30
	}
	msgs = make([]Message, 0, len(hist)+4)
	if l.System != "" && !hasSystem(hist) {
		msgs = append(msgs, Message{Role: RoleSystem, Content: l.System})
	}
	msgs = append(msgs, hist...)
	msgs = append(msgs, Message{Role: RoleUser, Content: task})

	tools := l.Registry.Definitions()
	consecErr := 0

	for iter := 0; iter < max; iter++ {
		if err := ctx.Err(); err != nil {
			return msgs, err // 外部取消
		}

		msgs = l.maybeCompact(ctx, msgs) // 超窗口阈值则把中段老消息压成摘要（见 compress.go）

		// ── THINK：LLM 决策（流式 thinking/content 经事件外发）──
		assistant, err := l.Provider.Chat(ctx, msgs, tools, func(c Chunk) {
			if c.Reasoning != "" {
				l.emit(Event{Type: EventThinking, Content: c.Reasoning})
			}
			if c.Content != "" {
				l.emit(Event{Type: EventContent, Content: c.Content})
			}
			if c.Usage != nil && c.Usage.PromptTokens > 0 {
				l.lastPromptTokens = c.Usage.PromptTokens // 实测用量驱动下轮压缩判定
				// 发射 token 用量事件，供 UI 侧栏统计缓存命中/未命中
				usage := *c.Usage
				// 估算 prompt 构成 breakdown（前端 ConvSidebar 渲染构成占比条用）
				if usage.PromptBreakdown.SystemTokens == 0 { // 仅 Provider 未返回时估算
					pb := EstimateBreakdown(msgs, l.Registry.Definitions(), usage.PromptTokens)
					usage.PromptBreakdown = pb
				}
				l.emit(Event{Type: EventUsage, Usage: &usage})
			}
		})
		if err != nil {
			l.emit(Event{Type: EventError, Content: err.Error()})
			return msgs, err
		}
		msgs = append(msgs, assistant)
		l.currentMsgs = msgs // 同步：供 delegate handler 读父历史（含本轮 assistant；handler 剥离末尾未配对 tool_call 保前缀稳定）



		// ── ACT + OBSERVE：依次执行工具，结果作 role=tool 消息回灌 ──
		iterErr := false
		for _, tc := range assistant.ToolCalls {
			l.emit(Event{Type: EventToolCall, Tool: tc.Function.Name, Args: tc.Function.Arguments, CallID: tc.ID})

			// 审批门：写类工具（RequiresApproval）在手动审核下需用户批准。被拒则不执行，
			// 把拒绝作为观察回灌（让模型改道，而非当成工具错误计入连续失败）。
			if l.Approve != nil {
				if tool, ok := l.Registry.Get(tc.Function.Name); ok && tool.RequiresApproval {
					if approved, feedback := l.Approve(ctx, tc); !approved {
						rej := strings.TrimSpace(feedback)
						if rej == "" {
							rej = "用户拒绝了此操作。请勿重试该操作；改用其他方式达成目标，或先向用户说明你为何需要它。"
						}
						l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: rej, CallID: tc.ID})
						msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: rej})
						l.trackCall(tc.Function.Name, tc.Function.Arguments, true) // 被拒也算一次未成、计入绕圈检测
						continue
					}
				}
			}

			result, terr := l.Registry.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if terr != nil {
				result = "Error: " + terr.Error()
				iterErr = true
			}
			l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: result, CallID: tc.ID})
			msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: result})
			l.trackCall(tc.Function.Name, tc.Function.Arguments, terr != nil || strings.HasPrefix(strings.TrimSpace(result), "Error:"))

			// finish_task 退出信号：子 agent 调 finish_task 表示任务完成，记录 result 退出循环。
			// delegate handler 从 child.finishResult 取子最终结果。仅子 Loop 注册此工具。
			if tc.Function.Name == "finish_task" {
				l.finishResult = &result
				// finish_task 退出：发射 EventDone（含 result），不发射 EventFinal
				l.emit(Event{Type: EventDone, Content: result, DoneReason: "finish_task"})
				return msgs, nil
			}
		}

		// 绕圈检测：同一操作反复失败/反复执行 → 注入「换思路」提示打破死循环（见 circling.go）。
		if nudge := l.detectCircling(); nudge != "" {
			l.emit(Event{Type: EventCircling, Content: "检测到重复操作/反复失败，已提示 Agent 换思路打破死循环"})
			msgs = append(msgs, Message{Role: RoleUser, Content: nudge})
			l.recentCalls = nil // 提示后清零，给新思路一个干净起点
		}

		// 连续 3 轮工具全有错 → 止损停（复刻参考源 3-consecutive-error）。
		// 返回 sentinel 错误供桥接层判断，避免误以为正常完成而继续验证/评测阶段。
		if iterErr {
			if consecErr++; consecErr >= 3 {
				l.emit(Event{Type: EventError, Content: ErrConsecToolError.Error()})
				return msgs, ErrConsecToolError
			}
		} else {
			consecErr = 0
		}

		// transfer_to_agent：当前 agent 退出，控制权转移给目标 agent（由调用方接管同一 []Message）。
		if l.transferTarget != "" {
			return msgs, nil
		}
	}
	l.emit(Event{Type: EventError, Content: ErrMaxIterations.Error()})
	return msgs, ErrMaxIterations
}

func hasSystem(msgs []Message) bool {
	for _, m := range msgs {
		if m.Role == RoleSystem {
			return true
		}
	}
	return false
}



// DefaultSystemPrompt 核心铁律的系统提示词（中文 lock / 改前 read / 工作区限定 / finish_task 退出）。
// roots 为工作区所有根目录（支持多根工作区）；roots[0] 为主根。
func DefaultSystemPrompt(roots []string) string {
	rootInfo := "根目录: " + roots[0]
	if len(roots) > 1 {
		rootInfo += "\n工作区包含以下所有项目目录（均可访问）："
		for i, r := range roots {
			rootInfo += fmt.Sprintf("\n  %d. %s", i+1, r)
		}
	}
	return "你是 Pair CodeAgent，运行在用户的本地开发环境中。使用中文思考和回复。\n\n" +
		"# 工作区\n" + rootInfo + "\n\n" +
		"## ⚠️ 第一铁律：语言锁定（中文）\n" +
		"无论上一步工具返回了什么代码、终端输出、英文文档或其他内容，\n" +
		"你都必须用中文思考和回复，这是不可违背的铁律。工具输出中的英文是\n" +
		"工作内容的一部分，不代表你的语言可以切换到英文。推理过程、分析、\n" +
		"决策、最终回复都必须使用中文。\n" +
		"如果发现自己的思考变成了英文，立即停下并切换回中文。\n" +
		"这是最高优先级的约束，不允许任何形式的绕过。\n\n" +
		"# 核心规则\n" +
		"- 文件操作只用工作区内路径；修改文件前必须先 read_file 确认当前内容。\n" +
		"- 每次工具调用后，依据真实结果决定下一步，绝不臆测结果。\n" +
		"- 禁止破坏性命令（如 rm -rf、强制 push main），禁止修改工作区外文件。\n" +
		"- 【完成标记】任务彻底完成时，调用 finish_task 工具提交最终结果摘要，切勿在正文中输出 [FINAL]。\n\n" +
		"# 任务追踪（核心机制）\n" +
		"任何需要 3+ 步骤或多文件操作的任务，必须使用 task_create/task_update 追踪进度：\n" +
		"- 收到任务后第一回合创建完整子任务清单，立即将第一个标记为 in_progress。\n" +
		"- 完成一项更新一项（task_update），绝不批量更新。\n" +
		"- 发现新前置依赖或方案不可行时即时调整计划。\n" +
		"- 所有任务完成后，先调用 task_summary 确认进度摘要，然后调用 finish_task 提交结果。\n\n" +
		"# 读取策略\n" +
		"读文件时必须串行推进——读完一个文件，分析内容，再决定下一个读什么。\n" +
		"禁止一次性发出 3+ 个 read_file——你预判需要的文件往往有一半是多余的。\n" +
		"- 查找函数/类定义时，优先用 find_symbol（零迭代消耗）。\n" +
		"- 了解文件对外接口时，优先用 get_file_symbols。\n" +
		"- 修改文件前，先调用 check_impact 了解影响范围。\n" +
		"- 每次最多并行 2 个读操作（仅在两文件明显互不依赖时）。\n" +
		"- 写操作和读操作不要混在同一轮——先读完确认，再写。\n\n" +
		"# 错误恢复\n" +
		"- 工具调用失败后分析错误原因，换一种方式重试（最多 3 次）。\n" +
		"- edit_file/multi_edit 已内置 CRLF 归一化与空白折叠匹配，常规差异无需重读。\n" +
		"  失败时诊断信息含行号上下文：优先改用 line_start/line_end 行号定位（最可靠）；\n" +
		"  若仍失败再 read_file 确认最新内容。★ 绝不要因匹配失败就改用 write_file 覆盖整个文件。\n" +
		"- 连续 3 次工具执行失败 → 自动终止，向用户报告原因。\n" +
		"- shell_exec 失败 → 检查 stderr 输出，不要只靠 exit code 判断。\n\n" +
		"# 验证原则\n" +
		"每次工具调用后，先验证再行动：文件读取后确认行号匹配；shell_exec 后检查 stdout 内容；\n" +
		"搜索结果确认匹配正确。不要声称改动成功除非看到了证据。\n\n" +
		"# 工具\n" +
		"- 浏览定位：search_files（按通配符找文件）、search_content（按正则搜内容）、list_files。\n" +
		"- 读改：read_file（改前必读）、edit_file（小处精确替换，首选）、multi_edit（一次改多处）、write_file（整文件覆盖/新建）、move_file（移动/重命名）、delete_file（删文件）。\n" +
		"- 运行：run_command（同步，等结果）；run_background（后台长任务）→ read_output 看输出、kill_process 停。\n" +
		"- 联网：web_fetch（抓网页）、web_search（搜索引擎）——查文档/报错/库用法。\n" +
		"- Git：git_status / git_diff / git_log / git_show / git_blame（只读）；git_add / git_commit / git_branch / git_checkout / git_stash（写类需审批）。\n" +
		"- 记忆：用【简短中文】命名；先 memory_search 查有无相关——有则 memory_read 读后融合、用同名 memory_write【更新】；memory_list 看总览。\n" +
		"- 任务追踪：task_create / task_update / task_list / task_delete / task_summary。\n" +
		"- 规划：复杂任务用 update_plan 列出步骤清单，执行中更新状态。\n" +
		"- 提问：关键决策或需求有歧义时用 ask_user 问用户，别自己瞎猜。\n\n" +
		"# 工作方式\n" +
		"按「思考 → 调用工具 → 观察结果 → 再决策」循环推进，直至完成。\n" +
		"复杂或多步任务先用 task_create 分解为子任务，再逐步执行并更新状态。\n" +
		"先用 search_* 定位、read_file 细读，再动手；改动优先 edit_file（小而准），大改才 write_file。\n" +
		"不确定的库用法/报错/最新信息，用 web_search / web_fetch 查证，别凭记忆臆测。\n" +
		"写类操作在手动审核模式下需用户批准；若被拒绝，换思路或先解释原因，勿反复重试同一操作。\n\n" +
		"# 输出规范\n" +
		"- 代码/终端输出使用 ```语言名 代码块（指定语言以获得语法高亮）。\n" +
		"- 表格保持 2-4 列避免过宽。\n" +
		"- 不用 emoji（除非用户明确要求）。\n" +
		"- 完成任务后输出 Markdown 总结：完成了什么、改了哪些文件（路径+改动）、如何验证结果、遗留问题。\n\n" +
		"# 防止卡死\n" +
		"- 不要连续 3 轮只输出分析文本而不调用任何工具。\n" +
		"- 不确定时宁可声明完成并向用户汇报，让用户决定是否继续。\n" +
		"- 不要在「让我再看看…」和「也许还需要…」之间反复循环。"
}

// ProjectRules 读工作区根的项目约定，拼成系统提示附加段供 agent 遵守：
// 项目文档（AGENTS.md / CLAUDE.md 取首个）+ 用户在设置「指令」tab 写的 .pair/rules.md（两者都注入）。
// 都没有则返回空串。每份内容超长截断。
func ProjectRules(root string) string {
	var b strings.Builder
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} { // 项目文档取首个
		if s := readCapped(root, name); s != "" {
			b.WriteString("\n\n# 项目约定（来自 " + name + "，务必遵守）\n" + s)
			break
		}
	}
	if s := readCapped(root, ".pair/rules.md"); s != "" { // 设置「指令」tab 写的（随项目存 .pair/）
		b.WriteString("\n\n# 项目指令（务必遵守）\n" + s)
	}
	return b.String()
}

// readCapped 读 root/name 并裁到 8000 字；不存在/空返回 ""。
func readCapped(root, name string) string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(data))
	if len(s) > 8000 {
		s = s[:8000] + "\n…（已截断）"
	}
	return s
}
