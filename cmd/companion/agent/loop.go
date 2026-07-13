package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	// OnFeedback 用户运行时反馈钩子（可空）。每次 LLM 调用前检查，返回非空字符串时，
	// 将内容作为 [User] 消息注入本轮上下文，让 Agent 在下一次回复中响应用户的补充/纠正。
	// 宿主（web/UI）通过此回调将用户的实时反馈传递给 Loop。
	OnFeedback func() string

	// OnBatchPersist 批量持久化回调（可空）。每 5 轮由 Loop.Run 内部回调一次当前完整消息列表。
	// loop.Run 返回后（defer 中）会额外调用一次以确保最后一次写盘。
	// 用于 agent 运行中途写盘，防止进程崩溃导致全部数据丢失。
	OnBatchPersist func(msgs []Message)

	// ── 上下文压缩（可空；复刻参考 context/manager.ts，见 compress.go）──
	// MaxContextTokens>0 时启用：每次 LLM 调用前，若 tokens/Max 超阈值，把中段老消息压成一条摘要。
	// Compressor 非空→用它（轻量压缩模型）做 LLM 摘要，否则/失败→规则式摘要。
	Compressor       Provider
	MaxContextTokens int

	// CompressedSummaries 累积的上下文压缩摘要列表。
	// 每次 maybeCompact 压缩中段老消息后追加一条摘要。
	// 这些摘要不插入历史消息，而是注入系统提示的可变部分（buildSystemWithSummaries），
	// 以保持 system 前缀稳定、语义清晰（系统提示的「项目当前状态」段自然包含摘要）。
	CompressedSummaries []string

	// compressedSummariesInjected 已注入系统提示的摘要数量。
	// 用于避免每次迭代都重新注入（仅在新增摘要时更新 msgs[0]）。
	compressedSummariesInjected int

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

	// AutoReview 审核开关。true=AI审核（内部创建审核Agent把关写操作）；
	// false=走外部Approve回调（人工审批）或自动放行（Autonomous=true时）。
	// 外部（桌面版/Web）只需设置此字段，审核决策完全由Loop内部决定。
	AutoReview bool

	// ReviewProvider 审核模型的 Provider（AutoReview=true 时使用）。
	// 由外部在创建 Loop 时设置，Loop 内部用它懒创建审核 Reviewer。
	ReviewProvider Provider

	// reviewer 内部懒创建的审核 Agent（由 ReviewProvider 创建，不导出）。
	reviewer *Reviewer

	// contentOnlyIters 连续 content-only（无 tool_call）轮数计数器。
	// 防止 Agent 只输出文字不调用 finish_task 导致自我循环。
	contentOnlyIters int

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

// aiReviewApprove 内部 AI 审核裁决（AutoReview=true 时由 Loop.Run 调用）。
// 通过 ReviewProvider 懒创建 Reviewer，写类工具交审核模型判。
// 通过放行，驳回/需要修改则把建议作反馈回灌。审核模型故障→放行（审核是增强非强制）。
func (l *Loop) aiReviewApprove(ctx context.Context, tc ToolCall) (bool, string) {
	if !NeedsReview(tc.Function.Name) {
		return true, ""
	}
	if l.reviewer == nil {
		if l.ReviewProvider == nil {
			return true, "" // 无审核模型 → 放行
		}
		l.reviewer = &Reviewer{Provider: l.ReviewProvider, SystemPrompt: DefaultReviewerPrompt()}
	}
	v, err := l.reviewer.Review(ctx, tc)
	if err != nil || v.Approved() {
		return true, ""
	}
	return false, v.FeedbackText()
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
	log.Printf("[trace] loop.Run 开始 task=%q provider=%s", task[:min(len(task), 50)], l.Provider.Name())
	// 自闭环：history 为 nil 时使用持久化的 l.History
	if history == nil {
		history = l.History
	}
	// 统一持久化出口：每次 Run 返回后更新 l.History（不论调用方是否传了 history）
	defer func() {
		l.History = msgs
		// 最终写盘兜底：OnBatchPersist 非空则调用一次
		if l.OnBatchPersist != nil && msgs != nil {
			l.OnBatchPersist(msgs)
		}
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
	// 防重复用户消息：hist（来自 store.LoadAll）末尾可能已有一条内容相同的 RoleUser
	// （由 handleChatSend 写入 store 后再 LoadAll 取回）。仅当内容一致时才跳过追加，
	// 避免子 agent（delegate_task）的 history 末尾是父 agent 的用户消息但任务不同时被误跳过。
	if len(msgs) > 0 && msgs[len(msgs)-1].Role == RoleUser && msgs[len(msgs)-1].Content == task {
		// 末尾已有同内容用户消息，跳过，防持久化后重复
	} else {
		msgs = append(msgs, Message{Role: RoleUser, Content: task})
	}

	tools := l.Registry.Definitions()
	consecErr := 0

	for iter := 0; iter < max; iter++ {
		if err := ctx.Err(); err != nil {
			return msgs, err // 外部取消
		}

		msgs = l.maybeCompact(ctx, msgs) // 超窗口阈值则把中段老消息压成摘要（见 compress.go）

		// ── 将压缩摘要注入系统提示的可变部分（而非插入历史消息）──
		// 仅在新增摘要时更新 msgs[0]。Content，避免每轮都改导致缓存前缀变化。
		if len(l.CompressedSummaries) > l.compressedSummariesInjected && len(msgs) > 0 && msgs[0].Role == RoleSystem {
			msgs[0].Content = l.buildSystemWithSummaries()
			l.compressedSummariesInjected = len(l.CompressedSummaries)
		}

		// ── 检查用户运行时反馈（补充/纠正）──
		if l.OnFeedback != nil {
			if fb := l.OnFeedback(); fb != "" {
				fbMsg := Message{Role: RoleUser, Content: "【用户反馈】" + fb}
				msgs = append(msgs, fbMsg)
				l.emit(Event{Type: EventNotice, Content: "收到用户反馈，Agent 将据此调整"})
			}
		}

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
				if usage.PromptBreakdown.SystemTokens == 0 { // 仅 Provider 未返回时估算
					pb := EstimateBreakdown(msgs, l.Registry.Definitions(), usage.PromptTokens)
					usage.PromptBreakdown = pb
				}
				l.emit(Event{Type: EventUsage, Usage: &usage})
				// agent 自闭环：持久化上下文统计到磁盘（供页面刷新后恢复）
				SaveTokenUsage(&usage)
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

			// ★ 审批门：Loop 内部根据 AutoReview 自决审核策略 ★
			// - AutoReview=true → 内部 AI 审核（用 ReviewProvider 懒建 Reviewer）
			// - AutoReview=false + Autonomous=true → 自动放行（nil=全部通过）
			// - AutoReview=false + Autonomous=false → 走外部 l.Approve（人工审批）
			approveFn := l.Approve
			switch {
			case l.AutoReview:
				approveFn = l.aiReviewApprove
			case l.Autonomous:
				approveFn = nil
			}
			if approveFn != nil {
				if tool, ok := l.Registry.Get(tc.Function.Name); ok && tool.RequiresApproval {
					if approved, feedback := approveFn(ctx, tc); !approved {
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

		}

		// ★ finish_task 检测：Agent 调用了 finish_task → 任务完成，保存结果并退出循环
		// 先同步 currentMsgs（包含 tool results），供 persist worker 获取完整历史
		l.currentMsgs = msgs
		for _, tc := range assistant.ToolCalls {
			if tc.Function.Name == "finish_task" {
				// 从 msgs 中找到最后一条 finish_task 的工具结果
				for i := len(msgs) - 1; i >= 0; i-- {
					if msgs[i].Role == RoleTool && msgs[i].Name == "finish_task" {
						l.finishResult = &msgs[i].Content
						break
					}
				}
				result := *l.finishResult
				l.emit(Event{Type: EventDone, Content: result, DoneReason: "finish_task"})
				return msgs, nil
			}
		}

		// ★ content-only 防护：Agent 连续输出文字但不调用任何工具（含 finish_task）
		// 说明 Agent 可能在自我循环，注入停止提示。
		// 阈值放宽到 3 轮提示、4 轮结束——复杂任务可能需要多轮分析总结。
		if len(assistant.ToolCalls) == 0 && strings.TrimSpace(assistant.Content) != "" {
			l.contentOnlyIters++
			if l.contentOnlyIters == 3 {
				nudge := "[系统提示] 你已经连续三轮只输出文字而没有调用任何工具。如果任务已完成，请调用 finish_task 工具结束本轮；如果还需要继续工作，请调用相应工具推进。"
				l.emit(Event{Type: EventNotice, Content: nudge})
				msgs = append(msgs, Message{Role: RoleUser, Content: nudge})
			} else if l.contentOnlyIters >= 4 {
				l.emit(Event{Type: EventNotice, Content: "检测到内容循环，自动结束"})
				l.emit(Event{Type: EventDone, Content: strings.TrimSpace(assistant.Content), DoneReason: "content_loop"})
				return msgs, nil
			}
		} else {
			l.contentOnlyIters = 0
		}
		// 绕圈检测：同一操作反复失败/反复执行 → 注入「换思路」提示打破死循环（见 circling.go）。
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

		// 每 5 轮调用一次 OnBatchPersist（中途写盘防崩溃丢数据）
		if l.OnBatchPersist != nil && iter > 0 && iter%5 == 0 {
			l.OnBatchPersist(msgs)
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

// buildSystemWithSummaries 构建包含压缩摘要的系统提示词。
// 在 l.System 的可变部分末尾追加「上下文压缩摘要」段。
// 摘要仅在中段老消息被压缩时变化，保持 system 前缀稳定、最大化缓存命中。
func (l *Loop) buildSystemWithSummaries() string {
	if len(l.CompressedSummaries) == 0 {
		return l.System
	}
	var b strings.Builder
	b.WriteString(l.System)
	b.WriteString("\n\n# 上下文已压缩——历史摘要\n\n")
	b.WriteString("> 以下为之前轮次的消息摘要，Agent 应据此感知已完成的历史上下文。\n> 请勿重复执行摘要中已包含的任务。\n\n")
	for i, s := range l.CompressedSummaries {
		if i > 0 {
			b.WriteString("\n\n---\n\n")
		}
		b.WriteString(s)
	}
	return b.String()
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
		"# ★ 调研优先（强制——违反必出错）\n" +
		"收到任务后，第一回合必须先收集资料、理解上下文，再动手改代码：\n" +
		"- 先用 search_content / search_files / find_symbol 定位相关文件和函数，搞清楚代码结构和调用关系。\n" +
		"- 用 read_file 细读关键文件的目标区域，确认当前实现、变量名、缩进风格、上下文逻辑。\n" +
		"- 如果涉及多个文件，先用 check_impact / find_symbol_usages 了解影响范围，不要漏改调用方。\n" +
		"- 对于不熟悉的库/框架用法，用 web_search 查证最新文档，别凭记忆臆测 API。\n" +
		"- 只有在充分理解代码现状后，才开始动手修改。宁可多花 2 轮调研，也不要在不了解全貌时动手。\n" +
		"- ★ 禁止凭任务描述就臆测代码内容——你的记忆可能是旧版或错误的，必须以 read_file 看到的实际内容为准。\n\n" +
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
		"- run_command 失败 → 检查 stderr 输出，不要只靠 exit code 判断。\n\n" +
		"# 代码修改纪律（严格遵守，防改错）\n" +
		"★★ 以下规则是反复改出语法错误后总结的铁律，必须遵守 ★★\n\n" +
		"## 改前准备\n" +
		"1. 修改前先用 read_file 完整读取目标区域（至少 20 行上下文），分析清楚结构和缩进风格。\n" +
		"2. 一次只改一个文件的一个逻辑块——不在一轮中交叉修改多个文件。\n" +
		"3. 同一文件的多次改动用 multi_edit 在一次工具调用中完成，不要分多次 edit_file。\n\n" +
		"## 修改方式\n" +
		"1. 小改动（≤5 行）：用 edit_file 精确替换，确保 old_string 在文件中唯一。\n" +
		"2. 大改动（>5 行或整段替换）：改用 write_file 写入整个目标区域（先用 read_file 确认内容后，精确写需要替换的行范围）。\n" +
		"3. ★ 换行符兼容 ★ edit_file 已内置 CRLF/空白折叠匹配，不需要手动调整换行符格式。\n" +
		"4. ★ 行号定位优先 ★ 当 edit_file 匹配失败时，优先用 line_start/line_end 行号定位，不再尝试 old_string 匹配。\n" +
		"5. ★ 文件结构错乱时 ★ 如果文件已经因为反复修改而结构错乱（重复定义、大括号不匹配），先用 git checkout -- 文件 恢复原始版本，再重新做完整修改——不在乱文件上继续打补丁。\n\n" +
		"## 验证\n" +
		"1. 改完后必须运行对应语言的编译/语法检查工具验证无错误（如 gofmt -e、vite build、tsc --noEmit 等）。\n" +
		"2. 编译通过≠功能正确，仍需执行相应运行时验证。\n\n" +
		"# 验证原则\n" +
		"每次工具调用后，先验证再行动：文件读取后确认行号匹配；run_command 后检查 stdout 内容；\n" +
		"搜索结果确认匹配正确。不要声称改动成功除非看到了证据。\n\n" +
		"# ⚠️ 验证流程（核心要求——不允许跳过）\n" +
		"写完代码后，编译通过 ≠ 功能正常。必须根据改动类型执行实际验证：\n\n" +
		"## Web 前端改动（Vue/React/HTML/CSS/JS）\n" +
		"1. 确认 dev server 正在运行（run_background 启动 npm run dev / go run 等）\n" +
		"2. 调用 web_debug 打开页面 URL，检查：\n" +
		"   - 控制台是否有 error/warning（JS 异常、接口 404、编译错误）\n" +
		"   - 页面文字长度是否 >0（白屏检测）\n" +
		"   - 截图是否正常（可用 image_analyze 分析截图内容）\n" +
		"3. 如有交互逻辑，通过 web_debug 的 type_selector/click_selector 参数模拟用户操作后再截图\n" +
		"4. 如需检查 DOM 状态，用 eval 参数执行 JS（如 'document.querySelector(\".app\").innerHTML'）\n\n" +
		"## 后端 API 改动\n" +
		"1. 确认 server 正在运行\n" +
		"2. 用 run_command 执行 curl 请求验证接口：curl -s http://localhost:PORT/api/xxx\n" +
		"3. 检查返回的 HTTP 状态码和 JSON 内容是否符合预期\n" +
		"4. 如需调试运行时行为，用 debug_start 启动 DAP 调试器，设断点单步执行\n\n" +
		"## Go 代码改动\n" +
		"1. go_build 确认编译通过\n" +
		"2. run_test 执行相关测试\n" +
		"3. 如涉及 HTTP handler，启动 server 后用 curl 或 web_debug 验证\n" +
		"4. 如涉及复杂逻辑，用 debug_start 设置断点，debug_variables 查看变量状态\n\n" +
		"## GUI 桌面端改动\n" +
		"1. 编译通过后运行程序\n" +
		"2. 用 screenshot_desktop 或 screenshot_window 截图查看界面\n" +
		"3. 用 image_analyze 分析截图（颜色/布局/元素位置）\n\n" +
		"## 验证纪律\n" +
		"- 每次代码改动后必须验证，不允许只编译就声称完成\n" +
		"- 验证失败时先修复再继续，不要带着已知问题往下走\n" +
		"- 验证结果要写入 finish_task 摘要（如\"web_debug 验证：0 错误，页面正常渲染\"）\n\n" +
		"# 工具\n" +
		"- 浏览定位：search_files（按通配符找文件）、search_content（按正则搜内容）、list_files、find_files_by_pattern（glob 查文件）。\n" +
		"- 读改：read_file（改前必读）、edit_file（小处精确替换，首选）、multi_edit（一次改多处）、write_file（整文件覆盖/新建）、move_file（移动/重命名）、delete_file（删文件）。\n" +
		"- 运行：run_command（同步，等结果）；run_background（后台长任务）→ read_output 看输出、kill_process 停。\n" +

		"- 联网：web_fetch（抓网页）、web_search（搜索引擎）——查文档/报错/库用法。\n" +
		"- ⚡ 网页验证：web_debug（一站式——打开URL+控制台错误+截图+JS执行+交互，首选验证工具）；headless_browser（JS渲染页面文本提取）；screenshot_webpage（网页截图）。\n" +
		"- 截图分析：screenshot_desktop/window/area（桌面截图）→ image_analyze（分析颜色/色块/图形）/ image_ocr（识别文字）。\n" +
		"- 文件符号与定位：find_symbol（查函数/类型定义）、get_file_symbols（查看文件符号列表）、find_symbol_usages（查找引用）、check_impact（分析改动影响）、list_exported_symbols（列出导出符号）、get_file_dependencies（查看文件依赖）、find_circular_deps（检测循环依赖）。\n" +
		"- 调试器：debug_start（启动 DAP 调试）→ debug_breakpoint（设断点）→ debug_continue/next/step_in/step_out（控制执行）→ debug_stack/variables/evaluate（查看状态）→ debug_stop（停止）；debug_status（查看状态）。\n" +
		"- Git：git_status / git_diff / git_log / git_show / git_blame（只读）；git_add / git_commit / git_branch / git_checkout / git_stash（写类需审批）。\n" +
		"- 记忆与知识库：memory_search / memory_read / memory_write / memory_list / memory_count；project_info_write/read/list/search/delete/explore（项目知识库）。\n" +
		"- BUG 检测与修复：bug_detect（全量检测）、bug_analyze（分析构建输出）、bug_fix（自动修复）。\n" +
		"- 二进制：inspect_binary（分析二进制）、write_binary（写二进制）、binary_strings/find/patch/info/hash/entropy（逆向分析）。\n" +
		"- 任务追踪：task_create / task_update / task_list / task_delete / task_summary。\n" +
		"- 规划与进度：update_plan（列出步骤清单）、progress_checker（查看进度）。\n" +
		"- 快照：restore_snapshot（从快照恢复文件）、list_snapshots（查看快照列表）。\n" +
		"- 提问：ask_user（向用户提问，等待回答）。\n" +
		"- 技能与扩展：skill_list / load_skill / load_skill_resource / skill_write / skill_delete；mcp_list / mcp_add / mcp_remove；marketplace_search / marketplace_install。\n\n" +
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
