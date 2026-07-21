package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

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

// CacheBoundary 分隔系统提示词静态前缀与动态后缀。
// 静态前缀（CacheBoundary 之前）在每次请求中保持不变，LLM API 通过公共前缀检测
// 实现 KV Cache 复用，大幅减少首 token 延迟和计算成本。
// 动态后缀（CacheBoundary 之后）可容纳每轮变化的会话特定内容，不影响前缀缓存。
// 参考：Claude Code 的 SYSTEM_PROMPT_DYNAMIC_BOUNDARY、DeepSeek 上下文缓存。
const CacheBoundary = "\n\n<!--- CACHE_BOUNDARY --->\n\n"

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
	DoneReason string
}



// Loop TAOR 编排器：think(LLM 决策)→act(执行工具)→observe(结果回灌)→repeat。
// 停止：自然终止（无 tool_call + 有正文）/ 达最大迭代 / 外部取消。
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

	// OnBatchPersist 批量持久化回调（可空）。每轮迭代结束立即回调一次当前完整消息列表，
	// 确保 tool_call 与 tool_result 配对完整写入磁盘。loop.Run 返回后 defer 中会额外调用
	// 一次以确保最后一轮写盘（PersistNewMessages 内部 diff 去重，无重复写开销）。
	OnBatchPersist func(msgs []Message)

	// OnMessagePersist 单条消息强制持久化（可空）。用于 delegate_task 等场景：
	// 委托前将外层助手消息刷盘，并将委派任务作为用户消息独立存储，使前端看到清晰的层次。
	OnMessagePersist func(msg Message) error

	// ── 上下文压缩（可空；复刻参考 context/manager.ts，见 compress.go）──
	// MaxContextTokens>0 时启用：每次 LLM 调用前，若 tokens/Max 超阈值，把中段老消息压成一条摘要。
	// Compressor 非空→用它（轻量压缩模型）做 LLM 摘要，否则/失败→规则式摘要。
	Compressor       Provider
	MaxContextTokens int

	// CompressedSummaries 累积的上下文压缩摘要列表。
	// 每次 maybeCompact 压缩中段老消息后追加一条摘要。
	// 这些摘要不作为 system message 的可变部分注入（那会破坏 KV Cache 前缀），
	// 而是在循环迭代中通过 buildInjectionMessage 构建并作为 user 消息插入历史。
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
	finishResult   *string        // 退出信号（子 Loop：子 agent 结束时的最终内容）
	commitMessage  string         // agent 通过 generate_commit_message 工具显式设置的提交信息

	// ── 连续驳回追踪 ──
	rejectionCount   int    // 连续被驳回次数，达 3 次自动停止
	lastRejectedTool string // 上次被驳回的工具名

	WorkspaceRoot string // 工作区根路径（用于 SaveTokenUsage 等工作区级持久化）
	transferTarget string         // transfer_to_agent 目标名（非空=当前 Loop 应退出，控制权转移给目标 agent）
	CompactRequested bool         // 外部设置后下轮迭代触发上下文压缩（供主动压缩 API 使用）
	Autonomous     bool           // 自主模式标志（供并行子 agent 继承）

	mu sync.Mutex // 保护 ReviewMode 的并发读写（SetReviewMode/getReviewMode）

	// ReviewMode 审核模式："auto"=AI审核, "manual"=手动审批, "off"=全部放行。
	ReviewMode string
	// ReviewBlacklist 审核黑名单：命中此列表的工具需要审核（为空=全部工具按 ReviewMode 审核）。
	ReviewBlacklist []string
	// ReviewWhitelist 审核白名单：命中此列表的工具跳过审核（黑名单优先）。
	ReviewWhitelist []string
	// ReviewProvider 审核模型的 Provider（ReviewMode="auto" 时使用）。
	// 由外部在创建 Loop 时设置，Loop 内部用它懒创建审核 Reviewer。
	ReviewProvider Provider

	// reviewer 内部懒创建的审核 Agent（由 ReviewProvider 创建，不导出）。

	reviewer *Reviewer

	// contentOnlyIters 连续 content-only（无 tool_call）轮数计数器。
	// contentOnlyIters 连续 content-only（无 tool_call）轮数计数器。
	// 防止 Agent 只输出文字导致自我循环。
	contentOnlyIters int

	// ── 自主模式长时任务（新架构） ──

	// ephemeralMsgs 临时内部消息（不被持久化）。
	// 存放系统注入的阶段提示、绕圈检测、内容循环提示等，仅在本次 LLM 调用时合并到上下文。
	// 调用 buildCallContext 后自动清空，确保不会污染持久化历史。
	ephemeralMsgs []Message

	// autonomousStartTime 自主模式启动时间（用于时间预算检查）。
	autonomousStartTime time.Time

	// maxAutonomousMinutes 自主模式最大运行分钟数（0=无限制）。
	maxAutonomousMinutes int

	// checkpointInterval 检查点间隔（每 N 轮迭代保存一次 Loop 状态，用于崩溃恢复）。
	// 0 表示不启用检查点（默认 5）。
	checkpointInterval int

	// iterSinceCheckpoint 自上次检查点以来的迭代次数。
	iterSinceCheckpoint int
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

// SetReviewMode 运行时更新审核模式（线程安全）。
// 修改立即生效：正在运行的 Loop 在下一个工具调用前会读到新值。
func (l *Loop) SetReviewMode(v string) {
	l.mu.Lock()
	l.ReviewMode = v
	l.mu.Unlock()
}

// getReviewMode 线程安全读取 ReviewMode（供 approve 门使用）。
func (l *Loop) getReviewMode() string {
	l.mu.Lock()
	v := l.ReviewMode
	l.mu.Unlock()
	return v
}

// aiReviewApprove 内部 AI 审核裁决（ReviewMode="auto" 时由 Loop.Run 调用）。
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
		// 时间戳注入用户消息（而非系统提示词），避免破坏 KV Cache 前缀命中。
		timestamp := time.Now().Format("2006-01-02 15:04:05 MST (UTC-07:00)")
		taskWithTs := task + "\n\n**消息时间**: " + timestamp
		msgs = append(msgs, Message{Role: RoleUser, Content: taskWithTs})
	}

	// ★ 启动时检查记忆/知识库过期引用，如发现则注入为 ephemeralMsg（不持久化）
	if staleMsg := AutoVerifyStale(); staleMsg != "" {
		l.ephemeralMsgs = append(l.ephemeralMsgs, Message{Role: RoleUser, Content: staleMsg})
	}

	// ★ 自主模式：记录启动时间（用于时间预算检查）
	if l.Autonomous && l.autonomousStartTime.IsZero() {
		l.autonomousStartTime = time.Now()
	}

	tools := l.Registry.Definitions()

	for iter := 0; iter < max; iter++ {
		if err := ctx.Err(); err != nil {
			return msgs, err // 外部取消
		}

		msgs = l.maybeCompact(ctx, msgs) // 超窗口阈值则把中段老消息压成摘要（见 compress.go）

		// ── 将压缩摘要和（自主模式下）执行日志作为 ephemeralMsg 注入 ──
		// 不被持久化，仅在本次 LLM 调用时作为背景上下文
		if len(l.CompressedSummaries) > l.compressedSummariesInjected && len(msgs) > 0 && msgs[0].Role == RoleSystem {
			if summaryMsg := l.buildInjectionMessage(); summaryMsg != "" {
				l.ephemeralMsgs = append(l.ephemeralMsgs, Message{Role: RoleUser, Content: summaryMsg})
			}
			l.compressedSummariesInjected = len(l.CompressedSummaries)
		}

		// ── 检查用户运行时反馈（补充/纠正）──
		if l.OnFeedback != nil {
			if fb := l.OnFeedback(); fb != "" {
				l.ephemeralMsgs = append(l.ephemeralMsgs, Message{Role: RoleUser, Content: "【用户反馈】" + fb})
				l.emit(Event{Type: EventNotice, Content: "收到用户反馈，Agent 将据此调整"})
			}
		}

		// ★ 时间预算检查（自主模式长时任务）
		if l.Autonomous && l.maxAutonomousMinutes > 0 {
			elapsed := time.Since(l.autonomousStartTime)
			if elapsed > time.Duration(l.maxAutonomousMinutes)*time.Minute {
				l.ephemeralMsgs = append(l.ephemeralMsgs, Message{
					Role: RoleUser,
					Content: fmt.Sprintf(
						"⚠️ 时间预算已超（已运行 %s，限额 %d 分钟）。请自然总结成果，完成任务。",
						elapsed.Round(time.Minute).String(), l.maxAutonomousMinutes,
					),
				})
			}
		}

		// ── THINK：LLM 决策（buildCallContext 合并 ephemeralMsgs，不被持久化）──
		callMsgs := l.buildCallContext(msgs)
		assistant, err := l.Provider.Chat(ctx, callMsgs, tools, func(c Chunk) {
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
					pb := EstimateBreakdown(callMsgs, l.Registry.Definitions(), usage.PromptTokens)
					usage.PromptBreakdown = pb
				}
				l.emit(Event{Type: EventUsage, Usage: &usage})
				// agent 自闭环：持久化上下文统计到磁盘（供页面刷新后恢复）
				if l.WorkspaceRoot != "" {
					SaveTokenUsageForRoot(l.WorkspaceRoot, &usage)
				} else {
					SaveTokenUsage(&usage)
				}
			}
		})
		if err != nil {
			l.emit(Event{Type: EventError, Content: err.Error()})
			return msgs, err
		}
        msgs = append(msgs, assistant)
		l.currentMsgs = msgs // 同步：供 delegate handler 读父历史（含本轮 assistant；handler 剥离末尾未配对 tool_call 保前缀稳定）

		// ★ 在工具执行前立即持久化 assistant 消息（含 thinking + tool_calls），
		//   确保 ask_user 等阻塞工具不会导致本轮 assistant 输出丢失。
		//   后续工具结果由迭代末尾的 OnBatchPersist 补充。
		if l.OnBatchPersist != nil {
			l.OnBatchPersist(msgs)
		}

		// ★ 记录执行日志：当 assistant 有分析内容且有工具调用时（即将执行委托/操作前）
		// 外层 agent 的分析和决策，在 delegate_task 之前可见
		if strings.TrimSpace(assistant.Content) != "" && len(assistant.ToolCalls) > 0 {
			l.LogAnalysis(assistant.Content)
		}

		// ── ACT + OBSERVE：依次执行工具，结果作 role=tool 消息回灌 ──

		// ★ 并行优化：2+ 个只读工具时并发执行
		if canParallelize(assistant.ToolCalls, l.Registry) {
			msgs = l.executeToolsParallel(ctx, assistant.ToolCalls, msgs)
		} else {

		for _, tc := range assistant.ToolCalls {
			l.emit(Event{Type: EventToolCall, Tool: tc.Function.Name, Args: tc.Function.Arguments, CallID: tc.ID})

			// ★ 审批门：Loop 内部根据 ReviewMode + 黑白名单自决审核策略 ★
			// - ReviewMode="auto" → 内部 AI 审核（用 ReviewProvider 懒建 Reviewer）
			// - ReviewMode="off" → 全部放行（nil=全部通过，不经过任何审核）
			// - ReviewMode="manual" → 走外部 l.Approve（人工审批）
			// ★ 黑白名单优先于 ReviewMode：
			//   - 若 ReviewBlacklist 非空且命中 → 强制审核
			//   - 若 ReviewWhitelist 非空且命中 → 跳过审核
			//   - 黑名单优先于白名单
			approveFn := l.Approve
			toolName := tc.Function.Name
			// 检查黑白名单
			inBlacklist := false
			for _, name := range l.ReviewBlacklist {
				if strings.Contains(toolName, name) { inBlacklist = true; break }
			}
			inWhitelist := false
			if !inBlacklist {
				for _, name := range l.ReviewWhitelist {
					if strings.Contains(toolName, name) { inWhitelist = true; break }
				}
			}
			if inBlacklist {
				// 黑名单命中：按 ReviewMode 审核（即使 mode=off 也审核）
				switch l.getReviewMode() {
				case "auto":
					approveFn = l.aiReviewApprove
				default:
					approveFn = l.Approve
				}
			} else if inWhitelist {
				// 白名单命中：跳过审核
				approveFn = nil
			} else {
				// 不在黑白名单中：按 ReviewMode 执行
				switch l.getReviewMode() {
				case "auto":
					approveFn = l.aiReviewApprove
				case "off":
					approveFn = nil
				}
			}
			if approveFn != nil {
				if tool, ok := l.Registry.Get(tc.Function.Name); ok && tool.RequiresApproval {
					if approved, feedback := approveFn(ctx, tc); !approved {
						rej := strings.TrimSpace(feedback)
						if rej == "" {
							rej = "用户拒绝了此操作。请勿重试该操作；改用其他方式达成目标，或先向用户说明你为何需要它。"
						}
						// ★ 连续驳回追踪：同一工具连续驳回 3 次→自动停止
						if tc.Function.Name == l.lastRejectedTool {
							l.rejectionCount++
						} else {
							l.rejectionCount = 1
							l.lastRejectedTool = tc.Function.Name
						}
						if l.rejectionCount >= 3 {
							l.emit(Event{Type: EventError, Content: "操作 " + tc.Function.Name + " 已被连续驳回 3 次，自动停止"})
							return msgs, errors.New("连续驳回 3 次，自动停止")
						}
						l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: rej, CallID: tc.ID})
						msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: rej})
						l.trackCall(tc.Function.Name, tc.Function.Arguments, true)
						continue
					}
					// 审批通过 → 重置驳回追踪
					if tc.Function.Name == l.lastRejectedTool {
						l.rejectionCount = 0
						l.lastRejectedTool = ""
					}
				}
			}

			result, terr := l.Registry.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if terr != nil {
				result = "Error: " + terr.Error()
			}
			l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: result, CallID: tc.ID})
			msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: result})
			l.trackCall(tc.Function.Name, tc.Function.Arguments, terr != nil || strings.HasPrefix(strings.TrimSpace(result), "Error:"))

			// ★ 自主模式：generate_commit_message 记录提交信息（供最终完成时使用）
			// 不再用于区分阶段/完成，仅记录 commit message 字符串
			if tc.Function.Name == "generate_commit_message" {
				l.commitMessage = result
			}

			// ★ finish_task：子 agent 报告完成（主 Loop 中作兜底；正常完成靠自然输出）
			if tc.Function.Name == "finish_task" {
				l.finishResult = &result
				l.emit(Event{Type: EventDone, Content: result, DoneReason: "finish_task"})
				return msgs, nil
			}
		}
		} // end else (serial tool execution)

		// 先同步 currentMsgs（包含 tool results），供 persist worker 获取完整历史
		l.currentMsgs = msgs

		// 先同步 currentMsgs（包含 tool results），供 persist worker 获取完整历史
		l.currentMsgs = msgs

		// ★ 自然终止：模型无工具调用且有正文 → 任务完成
		if l.contentOnlyIters == 0 && len(assistant.ToolCalls) == 0 && strings.TrimSpace(assistant.Content) != "" {
			l.finishResult = &assistant.Content
			l.emit(Event{Type: EventDone, Content: strings.TrimSpace(assistant.Content), DoneReason: "task_complete"})
			return msgs, nil
		}

		// ★ content-only 防护：连续多轮只输出文字不调工具 → 死循环兜底
		if len(assistant.ToolCalls) == 0 && strings.TrimSpace(assistant.Content) != "" {
			l.contentOnlyIters++
			if l.contentOnlyIters == 3 {
				nudge := "[系统提示] 你已经连续三轮只输出文字而没有调用任何工具。如果任务已完成，直接自然总结；如还需继续，请调用工具推进。"
				l.emit(Event{Type: EventNotice, Content: nudge})
				l.ephemeralMsgs = append(l.ephemeralMsgs, Message{Role: RoleUser, Content: nudge})
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
			l.ephemeralMsgs = append(l.ephemeralMsgs, Message{Role: RoleUser, Content: nudge})
			l.recentCalls = nil // 提示后清零，给新思路一个干净起点
		}

		// transfer_to_agent：当前 agent 退出，控制权转移给目标 agent（由调用方接管同一 []Message）。
		if l.transferTarget != "" {
			return msgs, nil
		}

		// ★ 每轮迭代结束立即持久化，确保 tool_call 与 tool_result 配对完整写入磁盘。
		//   即使进程崩溃，最多丢失当前正在执行的这一轮，之前的所有轮次消息完好。
		if l.OnBatchPersist != nil {
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

// buildCallContext 合并持久化消息和临时内部消息（ephemeralMsgs），
// 返回完整的 LLM 调用上下文。调用后自动清空 ephemeralMsgs，
// 确保内部消息不会被持久化。
func (l *Loop) buildCallContext(msgs []Message) []Message {
	if len(l.ephemeralMsgs) == 0 {
		return msgs
	}
	result := make([]Message, len(msgs)+len(l.ephemeralMsgs))
	copy(result, msgs)
	copy(result[len(msgs):], l.ephemeralMsgs)
	l.ephemeralMsgs = nil // 清空，确保不会重复注入
	return result
}

// buildInjectionMessage 构建注入历史消息中的背景上下文。
// 包含：压缩摘要（上下文压缩后产生）+ 自主模式下的执行日志。
// 不修改 system message（msgs[0]），不破坏 KV Cache 前缀。
func (l *Loop) buildInjectionMessage() string {
	var b strings.Builder

	// 历史摘要（上下文压缩后产生）
	if len(l.CompressedSummaries) > 0 {
		b.WriteString("# 上下文已压缩——历史摘要\n\n")
		b.WriteString("> 以下为之前轮次的消息摘要，Agent 应据此感知已完成的历史上下文。\n> 请勿重复执行摘要中已包含的任务。\n\n")
		for i, s := range l.CompressedSummaries {
			if i > 0 {
				b.WriteString("\n\n---\n\n")
			}
			b.WriteString(s)
		}
	}

	// ★ 执行日志：记录各轮的分析与操作，不受上下文压缩影响
	// 仅在自主模式注入——非自主模式下执行日志与消息历史冗余。
	if l.Autonomous {
		if logStr := l.FormatExecutionLog(); logStr != "" {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(logStr)
		}
	}

	return b.String()
}



// DefaultSystemPrompt 核心铁律的系统提示词（中文 lock / 改前 read / 工作区限定）。
// roots 为工作区所有根目录（支持多根工作区）；roots[0] 为主根。
// roots 为空时使用当前工作目录作为兜底根目录。
func DefaultSystemPrompt(roots []string) string {
	primaryRoot := "（未设置工作区）"
	if len(roots) > 0 {
		primaryRoot = roots[0]
	}
	rootInfo := "根目录: " + primaryRoot
	if len(roots) > 1 {
		rootInfo += "\n工作区包含以下所有项目目录（均可访问）："
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
		"- 首次遇到环境/编译问题时，先读 .pair/project.md 获取已知环境配置（已注入系统提示）；\n" +
		"  若问题未记录，在解决后用 edit_file 更新 .pair/project.md（编译方式、多端目标、CGO 开关等），\n" +
		"  避免后续对话反复探测同一问题浪费 token。\n" +
		"- 【完成标记】任务完成时调用 generate_commit_message 记录提交信息，然后输出最终完成总结。" +
			" 切勿在正文中输出 [FINAL] 等标记。系统自动检测到无工具调用+有正文时视为完成。\n\n" +
		"# ★ 调研优先（强制——违反必出错）\n" +
		"收到任务后，第一回合必须先收集资料、理解上下文，再动手改代码：\n" +
		"- 先用 search_content / search_files / codegraph_search / find_symbol 定位相关文件和函数，搞清楚代码结构和调用关系（搜函数/类型名优先用 codegraph_search，更精确）。\n" +
		"- 用 read_file 细读关键文件的目标区域，确认当前实现、变量名、缩进风格、上下文逻辑。\n" +
		"- 如果涉及多个文件，先用 codegraph_impact（函数级）/ check_impact（文件级）或 codegraph_callers（调用者）/ find_symbol_usages（符号引用）了解影响范围，不要漏改调用方。\n" +
		"- 对于不熟悉的库/框架用法，用 web_search 查证最新文档，别凭记忆臆测 API。\n" +
		"- 只有在充分理解代码现状后，才开始动手修改。宁可多花 2 轮调研，也不要在不了解全貌时动手。\n" +
		"- ★ 禁止凭任务描述就臆测代码内容——你的记忆可能是旧版或错误的，必须以 read_file 看到的实际内容为准。\n\n" +
		"# 任务追踪（核心机制）\n" +
		"任何需要 3+ 步骤或多文件操作的任务，必须先用 update_plan 制定执行计划（步骤清单），再用 update_tasks 追踪进度：\n" +
		"- 收到任务后第一轮：调用 update_plan 列出完整执行计划（每项含 step + status），展示给用户。\n" +
		"- 后续每轮：调用 update_tasks 创建完整子任务清单，每项含 subject + status（pending/in_progress/completed）。\n" +
		"- 状态变化时重传整份清单（全量替换模式），系统自动持久化到磁盘。\n" +
		"- ★ 全量替换：每次传入全部任务，已不在列表中的旧任务将自动清理。\n" +
		"- 发现新前置依赖或方案不可行时即时调整计划。\n" +
		"- 所有任务全部完成后结束本轮任务。\n\n" +
		"读文件时必须串行推进——读完一个文件，分析内容，再决定下一个读什么。\n" +
		"禁止一次性发出 3+ 个 read_file——你预判需要的文件往往有一半是多余的。\n" +
		"- 查找函数/类定义时，优先用 codegraph_function（附签名，支持34种语言）；仅 Go 语言可用 find_symbol。\n" +
		"- 了解文件对外接口时，优先用 get_file_symbols。\n" +
		"- 查看 struct/interface 完整层次结构时，优先用 codegraph_class。\n" +
		"- 修改文件前，先调用 codegraph_impact（函数级影响链）或 check_impact（文件级导入依赖）了解影响范围。\n" +
		"- 每次最多并行 2 个读操作（仅在两文件明显互不依赖时）。\n" +
		"- 写操作和读操作不要混在同一轮——先读完确认，再写。\n\n" +
		"# ★ 代码知识图谱（codegraph）使用指南\n" +
		"codegraph 11 个工具基于结构化理解（实体+关系+调用图），比旧版纯文本工具更精确、更智能。\n" +
		"决策规则：搜函数/类型/变量名 → codegraph_search（优于 search_content）；找函数定义 → codegraph_function（多语言，优于 find_symbol）；\n" +
		"查调用者 → codegraph_callers（优于 find_symbol_usages）；函数级影响分析 → codegraph_impact（优于 check_impact）；\n" +
		"看类型结构 → codegraph_class（优于 get_file_symbols）。\n" +
		"覆盖 34 种语言（Go/JS/TS/Python/Rust/Java/C++/C#/Ruby/PHP/Swift/Kotlin/Dart/Lua/Bash/SQL/Vue/HTML/CSS/JSON/YAML/Markdown 等）。\n\n" +
		"# 错误恢复\n" +
		"- 工具调用失败后分析错误原因，换一种方式重试（最多 3 次）。\n" +
		"- edit_file/multi_edit 已内置 CRLF 归一化与空白折叠匹配，常规差异无需重读。\n" +
		"  失败时诊断信息含行号上下文：优先改用 line_start/line_end 行号定位（最可靠）；\n" +
		"  若仍失败再 read_file 确认最新内容。★ 绝不要因匹配失败就改用 write_file 覆盖整个文件。\n" +
		"- 工具执行失败后分析错误原因，换一种方式重试。\n" +
		"- run_command 失败 → 检查 stderr 输出，不要只靠 exit code 判断。\n" +
		"- ★ run_command 被 isBlockingCommand 拦截 → 说明命令是长期进程（如 dev server）。" +
			"你用了错误的工具！请改用 run_background 执行，不要用 run_command 重试。\n\n" +
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
		"2. 调用 web_debug 打开页面 URL，检查控制台错误 + 截图\n" +
		"3. 如有交互逻辑，通过 web_debug 的 type_selector/click_selector 模拟操作\n" +
		"4. 用 eval 参数执行 JS 检查 DOM 状态\n\n" +
		"## 后端 API / Go 代码 / 桌面端改动\n" +
		"1. go_build 确认编译通过，run_test 执行相关测试\n" +
		"2. 启动 server 后用 web_debug 或 curl 验证接口\n" +
		"3. 复杂逻辑用 debug_start 设置断点调试\n\n" +
		"## 验证纪律\n" +
		"- 每次代码改动后必须验证，不允许只编译就声称完成\n" +
		"- 验证失败时先修复再继续\n\n" +
		"# ★ 调研优先：善用 codegraph\n" +
		"项目已预构建代码知识图谱（codegraph），能秒级定位函数/类型定义、调用关系、影响范围，无需全文读取。\n" +
		"应优先使用 codegraph 工具而非 search_content 全文搜索或 list_files 遍历——它们是结构化的，更省 token。\n" +
		"搜函数→codegraph_search / 找定义→codegraph_function / 查调用者→codegraph_callers / 查影响→codegraph_impact。\n\n" +
		"其他工具：编辑(read_file/edit_file/multi_edit)、运行(run_command/run_background)、\n" +
		"联网(web_fetch/web_search)、截图(screenshot_*)、调试(debug_*)、Git(git_*)、记忆(memory_*)、\n" +
		"BUG检测(bug_*)、办公(csv_*/word_*/read_pdf)、MCP/技能(skill_*/mcp_*)、任务(update_tasks)。\n\n" +
		"# 工作方式\n" +
		"按「思考 → 调用工具 → 观察结果 → 再决策」循环推进，直至完成。\n" +
		"复杂或多步任务先用 update_tasks 列出细分任务，再逐步执行并更新状态。\n" +
		"先用 search_* 定位、read_file 细读，再动手；改动优先 edit_file（小而准），大改才 write_file。\n" +
		"不确定的库用法/报错/最新信息，用 web_search / web_fetch 查证，别凭记忆臆测。\n" +
		"写类操作在手动审核模式下需用户批准；若被拒绝，换思路或先解释原因，勿反复重试同一操作。\n\n" +
		"# 防止卡死\n" +
		"- 不要连续 3 轮只输出分析文本而不调用任何工具。\n" +
		"- 不确定时宁可声明完成并向用户汇报，让用户决定是否继续。\n" +
		"- 不要在「让我再看看…」和「也许还需要…」之间反复循环。\n" +
		"- run_command 阻塞预防：长期进程用 run_background（后台不阻塞）。\n" +
		"- 完成后输出 Markdown 总结：改了哪些文件、如何验证、遗留问题。" +
		CacheBoundary
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
