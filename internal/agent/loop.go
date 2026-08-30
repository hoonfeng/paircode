package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	EventToolUpdate EventType = "tool_update" // 工具执行中间结果（流式更新，供 UI 逐步展示）
	EventFinal      EventType = "final"       // 任务完成（仅 delegate 单轮委托用；主 Loop 用 EventDone）
	EventError      EventType = "error"       // 出错/止损
	EventCompacted  EventType = "compacted"   // 上下文已压缩（中段老消息压成摘要；UI 显示一行素色提示）
	EventEvaluation EventType = "evaluation"  // 任务评测评分（完成后评测模型打分；UI 显示评分卡）
	EventCircling   EventType = "circling"    // 检测到重复绕圈，已注入「换思路」提示打破死循环（UI 显示一行提示）
	// EventApproval 等待用户审批某次写类工具调用。由宿主（UI 桥）在 Approve 钩子里 emit，
	// loop 自身不直接发——loop 只通过 Approve 回调阻塞等待裁决（见 agent_bridge.go）。
	EventUsage    EventType = "usage" // LLM 调用完成后的 token 用量（含缓存命中/未命中）
	EventApproval EventType = "approval"

	EventNotice EventType = "notice" // 后台任务通知（jobs 包用；UI 显示一行素色提示）
	EventPhase  EventType = "phase"  // 阶段切换（自主模式下的规划/执行/评测等阶段指示）
	EventDone   EventType = "done"   // 结构化完成信号（供 delegate/子 agent 使用；主 Loop Exit 走此事件）
	// EventSnapshot WS 断线补偿快照（★ 2026-08-21）：客户端重连后，服务端推送当前
	// running 会话的流式生成进度（content/reasoning/toolSegments），前端据此重建
	// 占位消息，避免断线期间事件丢失导致内容截断。
	EventSnapshot EventType = "snapshot"
)

// CacheBoundary 分隔系统提示词静态前缀与动态后缀。
// 静态前缀（CacheBoundary 之前）在每次请求中保持不变，LLM API 通过公共前缀检测
// 实现 KV Cache 复用，大幅减少首 token 延迟和计算成本。
// 动态后缀（CacheBoundary 之后）可容纳每轮变化的会话特定内容，不影响前缀缓存。
// 参考：Claude Code 的 SYSTEM_PROMPT_DYNAMIC_BOUNDARY、DeepSeek 上下文缓存。
const CacheBoundary = "\n\n<!--- CACHE_BOUNDARY --->\n\n"

// ── 缓存诊断全局状态（WB_CACHE_DIAG=1 启用）──
// prevShape 跨 Loop/Run 共享：多轮对话（每次 Run 新建 Loop）间比较前缀形状，
// 定位导致 provider 缓存断裂的 system/tools 变化点。
var (
	cacheDiagStateMu sync.Mutex
	cacheDiagPrev    PrefixShape
	cacheDiagSession sessionCache // 进程级累计命中/未命中（诊断输出用）
)

// backgroundCtxMarker 背景上下文消息标记前缀。
// 注入到 ephemeral 消息的背景信息（历史摘要/执行日志/记忆知识库过期检查等）以此开头，
// buildCallContext 据此将其插入到「当前任务（最后一条 user 消息）」之前：
// 若背景信息追加在任务之后，LLM 会把最新一条 user 消息（如历史摘要）误认为当前输入，
// 导致只核对历史而不执行任务（2026-08-08 排查结论）。
const backgroundCtxMarker = "【背景上下文·非当前任务】\n"

// systemReminderFrame 把背景内容包进系统提醒框架（对齐 deepseek-harness
// agent-instructions 的 <system-reminder> 注入格式）。背景信息注入为 user-role
// ephemeral 消息（不持久化），框架让模型明确区分「背景信息」与「当前任务」，
// 避免把历史摘要/执行日志等误当作待执行输入。
func systemReminderFrame(kind, body string) string {
	return "<system-reminder>\n以下为" + kind + "（背景信息，非当前任务，仅作参考，请勿当作待执行任务）：\n\n" +
		body + "\n</system-reminder>"
}

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
	// ToolCallID 工具调用 ID（工具相关事件时设置）。
	ToolCallID string
	// PartialResult 中间结果文本（仅 EventToolUpdate 时设置）。
	PartialResult string
	// DoneReason 完成原因（仅 EventDone 时设置）。
	DoneReason string
	// Turn 事件所属 turn 序号（agentloop：一次 Run = 一个 turn，从 1 开始）。
	// 0 表示 Loop 尚未打开 turn（emit 时自动回填 l.TurnNo）。
	Turn int
	// Step 事件所属 step 序号（每次 LLM 调用 + 工具执行 = 一个 step）。
	Step int
	// TurnReason 结构化 turn 结束原因（仅 EventDone 时设置，见 agentloop.go）。
	// 与 DoneReason 并存：DoneReason 保持兼容值（task_complete），TurnReason 为枚举值。
	TurnReason string
	// Reasoning live 快照的思考累积（仅 EventSnapshot：WS 断线补偿）。
	Reasoning string
	// ToolSegments live 快照的工具调用段（仅 EventSnapshot：WS 断线补偿）。
	ToolSegments []LiveToolSeg
	// LiveEvents live 快照的有序事件序列（仅 EventSnapshot：WS 断线补偿）。
	// 前端逐事件重放重建 segments，保真 content/tool_call 交错时序。
	LiveEvents []LiveEvent `json:"events,omitempty"`
}

// pendingImage 一张待注入 LLM 上下文的图片（submit_image 工具提交）。
type pendingImage struct {
	Part   ImagePart // image_url 块（base64 data URL）
	Note   string    // 注入 user 消息时的说明文本（来源工具/路径/prompt）
	Source string    // 原始路径（去重用）
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

	// OnNextTask 自主模式下，当 agent 自然终止（无 tool_call + 有正文）且 follow-up 队列为空时，
	// 回调返回下一阶段的任务描述。返回非空字符串则自动注入 follow-up 消息让 agent 继续执行；
	// 返回 "" 表示无后续任务，正常退出。
	// 宿主（session_manager）通过此回调从任务队列中 pop 下一条待办任务。
	OnNextTask func() string

	// OnBatchPersist 批量持久化回调（可空）。每轮迭代结束立即回调一次当前完整消息列表，
	// 确保 tool_call 与 tool_result 配对完整写入磁盘。loop.Run 返回后 defer 中会额外调用
	// 一次以确保最后一轮写盘（PersistNewMessages 内部 diff 去重，无重复写开销）。
	OnBatchPersist func(msgs []Message)

	// OnMessagePersist 单条消息强制持久化（可空）。用于需要将单条消息独立落盘的场景
	// （如将委派任务/关键消息作为独立用户消息存储，使前端看到清晰的层次）。
	OnMessagePersist func(msg Message) error

	// ── 上下文压缩（可空；复刻参考 context/manager.ts，见 compress.go）──
	// MaxContextTokens>0 时启用：每次 LLM 调用前，若 tokens/Max 超阈值，把中段老消息压成一条摘要。
	// Compressor 非空→用它（轻量压缩模型）做 LLM 摘要，否则/失败→规则式摘要。
	Compressor       Provider
	MaxContextTokens int

	// CompressedSummaries 累积的上下文压缩摘要列表。
	// 每次 maybeCompact 压缩中段老消息后追加一条摘要。
	// 这些摘要不作为 system message 的可变部分注入（那会破坏 KV Cache 前缀），
	// 而是经 buildSnapshotContent 构建进「背景上下文快照」并作为 user 消息
	// 持久化到消息流（syncContextSnapshot；快照内容变化时追加新快照）。
	CompressedSummaries []string

	// staleMsg Run 启动时记忆/知识库过期检查结果（固定内容）。
	// 存字段而非每次调用扫描：VerifyAll 检查文件系统有成本，且内容须跨迭代稳定
	// 以保持 KV Cache 前缀一致（经背景快照 syncContextSnapshot 进入消息流）。
	staleMsg string

	lastPromptTokens int // 上一轮 API 实测 prompt_tokens（驱动压缩阈值，比纯估算可信）
	compactCooldown  int // 压缩后冷却剩余轮数（防每轮重复压缩，复刻参考 refreshCooldown）

	// compactArchive 本 Run 内被 compact 丢弃的中段消息归档（按原时间序追加）。
	// ★ 落盘/展示线还原（2026-08-27）：压缩视图仅供 LLM 提交，绝不允许压缩版
	//   覆盖落盘 store——否则前端刷新后只剩压缩摘要、完整历史丢失（1214 排查中
	//   实锤：唯一 user 被压缩删除后 lastUser 锚点失效走兜底分支，压缩版直接落盘）。
	//   所有持久化点经 l.persist（内部 fullHistory 还原）；展示同步点（currentMsgs）
	//   同样经 fullHistory。Run 开始时重置（此时历史自 store 加载，本身完整）。
	compactArchive []Message

	// cacheDiagOn 缓存诊断开关（WB_CACHE_DIAG=1 启用；前缀形状/累计状态为包级全局，
	// 跨 Loop/Run 共享，见 cacheDiagPrev/cacheDiagSession）。
	cacheDiagOn bool

	recentCalls []toolSig // 最近若干次工具调用签名+成败（绕圈检测，见 circling.go）

	// ── 消息队列（steer/followUp）──
	// steerQueue 托管消息：在当前轮次完成后、下一轮 LLM 调用前注入上下文。
	steerQueue []Message
	// followUpQueue 跟进消息：在 agent 自然终止（无 tool call 且有正文）后注入，
	// 让 agent 继续处理后续任务而不是立即退出。
	followUpQueue []Message

	// InheritedPrefixLen 继承自父 Loop 的 history 前缀条数（delegate 子 Loop 设置）。
	// 【已废弃 2026-08-15】原用于历史用户消息标注（MarkHistoryUserMessages）跳过前 N 条
	// 保持 KV Cache 前缀逐字节一致；标注已移除（对齐 harness），该字段仅向后兼容保留。
	InheritedPrefixLen int

	// ── 状态与元数据 ──
	State        map[string]any // 跨 Run 共享状态（如 executionLog 恢复，避免塞进 messages 撑爆上下文）
	currentMsgs  []Message      // Run 期间当前消息列表（供 persist worker 获取完整历史）
	finishResult *string        // 退出信号（自然终止时的最终内容）

	// ── 审核共享上下文值（2026-08-27 错误计数移除）──
	// approveState 审核决策共享状态：最近驳回决策 + 轻量历史（不计数）。
	// JS 插件经 loop.approve.state.get/set 读写；agentloop 审核逻辑据此决策。
	approveState *ApproveState

	// ── 首步极简工具面（2026-08-27 实测改进）──
	// StagedTools 开启后：会话第一个 Run 的首个 LLM 调用只注入极简核心工具面，
	// 自第 2 个 step 起恢复完整工具面（实测首步选对率 91.7% vs 全量 87.5%，
	// 且减少首步 token 开销）。默认开启；false 时始终全量。
	StagedTools bool
	// StagedToolGroups 极简工具候选组（插件装配链传入：agentloop 插件
	// registerSettings → 装配器解析 → LoopOpts；nil/空 = 回退默认组）
	StagedToolGroups [][]string
	WorkspaceRoot    string // 工作区根路径（用于 SaveTokenUsage 等工作区级持久化）
	CompactRequested bool   // 外部设置后下轮迭代触发上下文压缩（供主动压缩 API 使用）
	Autonomous       bool   // 自主模式标志（单 Loop 阶段化循环）

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

	// loopSvc 一切皆插件：本 Run 期间注册到全局插件宿主的 loop 服务（Run 结束置 nil）。
	// 插件 ctx.get('loop') 查询状态/请求暂停停止；并行 Run 时指向最近启动的 Loop。
	loopSvc *LoopService

	// contentOnlyIters 连续 content-only（无 tool_call）轮数计数器。
	// contentOnlyIters 连续 content-only（无 tool_call）轮数计数器。
	// 防止 Agent 只输出文字导致自我循环。
	contentOnlyIters int

	// ── 自主模式长时任务（新架构） ──

	// ephemeralMsgs 临时内部消息（不被持久化）。
	// 存放系统注入的阶段提示、绕圈检测、内容循环提示等，仅在本次 LLM 调用时合并到上下文。
	// 调用 buildCallContext 后自动清空，确保不会污染持久化历史。
	ephemeralMsgs []Message

	// ── 图片提交（★ 2026-08-22 submit_image 工具）──
	// pendingImages 工具产出/提交的图片队列（ImagePart 挂载在该 Loop 上）；
	// 工具执行结果含 __SUBMIT_IMAGE__ 标记 → 读图 bytes → 挂入（含提示文本）；
	// buildCallContext 每次迭代末尾把队列中的图片作为 user 消息（Images 字段）
	// 注入 LLM 上下文（next LLM 请求即以 image_url 块发送）——图片不再只落磁盘、
	// LLM 直接"看到"。仅当 Provider 支持多模态时注入图片（见 injectPendingImages）；
	// 每轮最多 1 张 + 每张 ≤2MiB + 路径去重（防上下文爆炸）。
	imageMu        sync.Mutex
	pendingImages  []pendingImage  // 待注入图片（消费后清空）
	imageInjected  map[string]bool // 已注入图片路径（会话级去重，防重复提交）
	imageInjectedN int             // 已注入图片数（上限防护）
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

	// ── agentloop（deepseek-harness 风格 turn/step 双层循环，见 agentloop.go）──
	// TurnNo 当前 turn 序号（一次 Run = 一个 turn，openTurn 递增）。
	TurnNo int
	// StepNo 当前 step 序号（每次 LLM 调用 + 工具执行 = 一个 step，turn 内递增）。
	StepNo int
	// LastTurnReason 最近一次 turn 的结束原因（结构化枚举，Run 返回后有效）。
	LastTurnReason TurnEndReason
	// hadMaxTokens 本轮 turn 是否曾触发 max-tokens（sticky：后续正常 step 不降级结果）。
	hadMaxTokens bool
	// PreStep 可选的 step 前拦截钩子（对应 agent/pre-step 瀑布）。
	// 每次 LLM 调用前、消息组装完成后调用；可改写进入模型的输入（返回 rewritten）或
	// 拒绝整个 turn（reject=true → turn 以 blocked 结束，跳过本次 LLM 调用）。
	// nil = 直通（默认）。
	PreStep func(ctx context.Context, callMsgs []Message, turn, step int) (rewritten []Message, reject bool, err error)
	// CancelCause 本轮 turn 被取消的原因（ctx 取消时由 Run 设置，见 agentloop.go）。
	CancelCause AgentCancelCause

	// ── Live 流式快照（★ 2026-08-21 WS 断线补偿）──
	// WS 断线期间 content/thinking/tool 事件会丢失；WebSocket 端点可调用
	// SnapshotLive() 在客户端重连时推送当前生成进度，前端据此重建占位消息，
	// 避免「断线后 assistant 内容截断/缺失」。
	liveMu        sync.Mutex
	liveContent   string        // 本 turn 已累积正文（跨 step 连续追加）
	liveReasoning string        // 本 turn 已累积思考
	liveToolCalls []LiveToolSeg // 已 emit 的工具调用摘要（tool_call + tool_result）
	liveEvents    []LiveEvent   // 已 emit 事件的有序序列（快照重建用：保真时序）
}

// runPreStep 执行 step 前拦截链（每次 LLM 调用前、消息组装完成后调用）：
//  1. host 钩子 l.PreStep（若设置）——可改写或拒绝；
//  2. DSH 桥瀑布 agent/pre-step（若有 Node 插件订阅）——DSH 中间件语义，
//     多个订阅者按注册顺序瀑布（next 链），返回 {kind:'enter',messages}
//     改写输入或 {kind:'reject'} 拒绝整个 turn；
//  3. 两者均可改写/拒绝；顺序：host 钩子先（本地配置权威），桥瀑布后
//     （插件视角——基于 host 改写后的消息再决策，最终以桥为准）。
//
// 无钩子且无订阅者 → 直通（零开销）。返回 (最终消息, reject, err)。
func (l *Loop) runPreStep(ctx context.Context, callMsgs []Message, turn, step int) ([]Message, bool, error) {
	if l.PreStep != nil {
		rewritten, reject, perr := l.PreStep(ctx, callMsgs, turn, step)
		if perr != nil || reject {
			return rewritten, reject, perr
		}
		if rewritten != nil {
			callMsgs = rewritten
		}
	}
	if bridgePreStepSubscribed() {
		rewritten, reject, perr := bridgePreStep(ctx, callMsgs, turn, step)
		if perr != nil || reject {
			return rewritten, reject, perr
		}
		if rewritten != nil {
			callMsgs = rewritten
		}
	}
	return callMsgs, false, nil
}

// LiveToolSeg 工具调用实时段（WS 断线补偿快照用：重建 tool_call 展示）。
type LiveToolSeg struct {
	CallID string `json:"callId"`
	Name   string `json:"name"`
	Args   string `json:"args"`
	Result string `json:"result"` // tool_result 正文（未完成时为空）
}

// LiveEvent 实时事件序列（WS 断线补偿快照：按 emit 顺序记录，供前端逐事件
// 重放重建 segments——解决「快照重建顺序丢失」：content 与 tool_call 的真实
// 交错时序（thinking→content→tool_call→…）必须在快照中保真）。
type LiveEvent struct {
	Type    string `json:"type"`              // thinking/content/tool_call/tool_result/error/notice/compacted/circling/evaluation
	Content string `json:"content,omitempty"` // thinking/content/tool_result 文本
	Tool    string `json:"tool,omitempty"`    // tool_call/tool_result 工具名
	Args    string `json:"args,omitempty"`    // tool_call 参数 JSON
	CallID  string `json:"callId,omitempty"`  // 工具调用 ID
}

func (l *Loop) emit(e Event) {
	// agentloop：自动回填事件所属 turn/step（未显式设置时）。
	if e.Turn == 0 {
		e.Turn = l.TurnNo
	}
	if e.Step == 0 {
		e.Step = l.StepNo
	}
	if l.OnEvent != nil {
		l.OnEvent(e)
	}
	// ★ live 快照维护（★ 2026-08-21 WS 断线补偿）：内容/工具事件同步累积，
	//   供 WebSocket 端点重连时推送当前生成进度（见 SnapshotLive）。
	switch e.Type {
	case EventThinking:
		l.liveMu.Lock()
		l.liveReasoning += e.Content
		l.liveEvents = appendLiveEvent(l.liveEvents, LiveEvent{Type: "thinking", Content: e.Content})
		l.liveMu.Unlock()
	case EventContent:
		l.liveMu.Lock()
		l.liveContent += e.Content
		l.liveEvents = appendLiveEvent(l.liveEvents, LiveEvent{Type: "content", Content: e.Content})
		l.liveMu.Unlock()
	case EventToolCall:
		l.liveMu.Lock()
		l.liveToolCalls = append(l.liveToolCalls, LiveToolSeg{CallID: e.CallID, Name: e.Tool, Args: e.Args})
		l.liveEvents = append(l.liveEvents, LiveEvent{Type: "tool_call", Tool: e.Tool, Args: e.Args, CallID: e.CallID})
		l.liveMu.Unlock()
	case EventToolResult:
		l.liveMu.Lock()
		for i := len(l.liveToolCalls) - 1; i >= 0; i-- {
			if l.liveToolCalls[i].CallID == e.CallID {
				l.liveToolCalls[i].Result = e.Content
				break
			}
		}
		// 事件序列同步回填（前端重放时据此更新对应 tool_call 段 result）
		found := false
		for i := len(l.liveEvents) - 1; i >= 0; i-- {
			if l.liveEvents[i].Type == "tool_call" && l.liveEvents[i].CallID == e.CallID {
				l.liveEvents[i].Content = e.Content
				found = true
				break
			}
		}
		if !found {
			// 兜底：tool_result 无匹配 tool_call（异常）→ 独立记录，前端仍可显示
			l.liveEvents = append(l.liveEvents, LiveEvent{Type: "tool_result", Tool: e.Tool, Content: e.Content, CallID: e.CallID})
		}
		l.liveMu.Unlock()
	case EventError, EventNotice, EventCompacted, EventCircling, EventEvaluation, EventApproval:
		l.liveMu.Lock()
		l.liveEvents = append(l.liveEvents, LiveEvent{Type: string(e.Type), Content: e.Content})
		l.liveMu.Unlock()
	}
	// ★ loop 服务快照同步（插件 ctx.get('loop') 的 getState 数据源）。
	if l.loopSvc != nil {
		l.loopSvc.noteEvent(e)
	}
	// ★ 一切皆插件：loop 事件桥——广播到全局插件 EventBus（loop:<type> 事件名），
	//   插件 ctx.on('loop:thinking' / 'loop:tool_call' / 'loop:done' …) 可监听扩展
	//   循环行为（统计/审计/通知/拦截上报），对齐参考项目 agent-loop 插件包的可扩展面。
	//   ★ 监听器在锁外同步执行：插件监听器 panic 不能波及核心循环（recover 隔离）。
	if ph := GetGlobalPluginHost(); ph != nil {
		if eb := ph.EventBus(); eb != nil && eb.ListenerCount("loop:"+string(e.Type)) > 0 {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[loop] 插件监听 loop:%s panic 已隔离: %v", e.Type, r)
					}
				}()
				eb.Emit("loop:"+string(e.Type), e)
			}()
		}
	}
}

// Steer 托管一条消息：在当前 step 完成后、下一 step LLM 调用前注入上下文。
// 对应 deepseek-harness Inbox 的 next-step 队列（steer 唤醒）：不打断当前工具执行，
// 在当前 step 边界处消费。使用场景：用户在 agent 正在执行时输入补充指令。
func (l *Loop) Steer(msg Message) {
	l.steerQueue = append(l.steerQueue, msg)
}

// LiveSnapshot 当前流式生成进度快照（WS 断线补偿：重连时推送重建占位消息）。
// ★ 使用场景：ServeGlobalEventStreamWS 客户端重连后，对每个 running 会话
//
//	（Session.Running=true）推送此快照，前端据此恢复断线期间丢失的 content/thinking/tool 事件。
//	返回 liveEvents 有序序列：前端逐事件重放重建 segments，保真 content/tool_call 交错时序
//	（修复「快照重建后所有工具调用聚在上方、正文聚在下方」的顺序丢失问题）。
func (l *Loop) LiveSnapshot() (content, reasoning string, tools []LiveToolSeg, events []LiveEvent) {
	l.liveMu.Lock()
	defer l.liveMu.Unlock()
	out := make([]LiveToolSeg, len(l.liveToolCalls))
	copy(out, l.liveToolCalls)
	evs := make([]LiveEvent, len(l.liveEvents))
	copy(evs, l.liveEvents)
	return l.liveContent, l.liveReasoning, out, evs
}

// resetLive 清空 live 快照（新 turn 开始时调用：新占位消息，旧累积无效）。
func (l *Loop) resetLive() {
	l.liveMu.Lock()
	l.liveContent = ""
	l.liveReasoning = ""
	l.liveToolCalls = nil
	l.liveEvents = nil
	l.liveMu.Unlock()
}

// appendLiveEvent 向事件序列追加事件：连续同类流式事件（thinking/content）
// 合并为一条（增量聚合），防止序列按 chunk 无限膨胀；非同类直接追加。
func appendLiveEvent(evs []LiveEvent, ev LiveEvent) []LiveEvent {
	if len(evs) > 0 {
		last := &evs[len(evs)-1]
		if last.Type == ev.Type && (ev.Type == "thinking" || ev.Type == "content") {
			last.Content += ev.Content
			return evs
		}
	}
	return append(evs, ev)
}

// FollowUp 托管一条跟进消息：在当前 turn 自然终止（无 tool call 且有正文）后注入，
// 让 agent 继续处理后续任务而不是立即退出。
// 对应 deepseek-harness Inbox 的 next-turn 队列（followup 唤醒）：只在 turn 边界消费。
func (l *Loop) FollowUp(msg Message) {
	l.followUpQueue = append(l.followUpQueue, msg)
}

// drainSteerQueue 清空并返回托管消息列表（step 边界消费，对齐 next-step）。
func (l *Loop) drainSteerQueue() []Message {
	if len(l.steerQueue) == 0 {
		return nil
	}
	msgs := l.steerQueue
	l.steerQueue = nil
	return msgs
}

// drainFollowUpQueue 清空并返回跟进消息列表（turn 边界消费，对齐 next-turn）。
func (l *Loop) drainFollowUpQueue() []Message {
	if len(l.followUpQueue) == 0 {
		return nil
	}
	msgs := l.followUpQueue
	l.followUpQueue = nil
	return msgs
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

// goLoopDeprecatedOnce Go 默认循环 deprecated 提示只打一次。
var goLoopDeprecatedOnce sync.Once

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
	// ★ 钩子系统（t1 L2 闭环）：UserPromptSubmit 门（轮次开始前；配置钩子 +
	//   插件钩子拦截 → 反馈回灌并终止本轮）。无钩子配置时零开销。
	if blocked, feedback := fireUserPromptSubmitHooks(ctx, task); blocked {
		msg := strings.TrimSpace(feedback)
		if msg == "" {
			msg = "UserPromptSubmit 钩子拦截了本轮任务"
		}
		l.emit(Event{Type: EventDone, Content: msg, DoneReason: "hook_blocked"})
		return []Message{{Role: RoleUser, Content: task}, {Role: RoleAssistant, Content: msg}}, fmt.Errorf("钩子拦截: %s", msg)
	}
	// ★ Stop 钩子：defer 兜底所有返回路径（轮次结束通知）。
	defer fireStopHooks(ctx)
	// ★ 新 turn：清空 live 快照（占位消息重新累积，旧累积无效）
	l.resetLive()
	// ★ agentloop 核心外置：全局注册了 JS 循环实现时，委托 JS 驱动循环
	//   （Go 保留能力经能力代理注入；停用插件即还原 Go 循环，可回退）。
	if impl := CurrentJSLoop(); impl != nil {
		return l.runWithJS(ctx, task, history, impl)
	}
	// ★ Go 默认循环标记 deprecated（agentloop 核心已外置 JS）。
	//   仅提示一次避免刷屏；未装载 JS 循环插件时作为回退路径继续工作。
	goLoopDeprecatedOnce.Do(func() {
		log.Printf("[loop] ⚠ Go 默认循环已 deprecated——装载 agentloop 插件（.pair/plugins/agentloop）后循环逻辑由 JS 驱动；本路径保留为回退")
	})
	// ★ Run 启动日志（排查「无响应」：确认 Loop 确实进入运行，以及每次启动时间）
	log.Printf("[loop] Run 开始 taskLen=%d history=%d maxIter=%d autonomous=%v",
		len(task), len(history), l.MaxIterations, l.Autonomous)
	// 自闭环：history 为 nil 时使用持久化的 l.History
	if history == nil {
		history = l.History
	}
	// agentloop：打开一轮新 turn（一次 Run = 一个 turn）。
	l.openTurn()

	// ★ 缓存诊断开关（WB_CACHE_DIAG=1 时输出前缀形状/命中率到 stderr，用于定位缓存断裂）
	l.cacheDiagOn = os.Getenv("WB_CACHE_DIAG") == "1"

	// ★ 一切皆插件：loop 服务面——Run 期间注册 ctx.provide('loop')，
	//   插件 ctx.get('loop') 可查询状态/请求暂停停止；Run 结束自动撤销。
	//   （并行 Run 时服务指向最近启动的 Loop；未运行期间 get 返回 nil。）
	if ph := GetGlobalPluginHost(); ph != nil && ph.Context() != nil {
		l.loopSvc = newLoopService(l)
		cancelProvide := ph.Context().Provide("loop", l.loopSvc)
		defer cancelProvide()
	}

	// 统一持久化出口：每次 Run 返回后更新 l.History（不论调用方是否传了 history）
	defer func() {
		l.History = l.fullHistory(msgs)
		// 最终写盘兜底：OnBatchPersist 非空则调用一次（经 fullHistory 还原完整时间线，
		// 防止压缩视图覆盖 store——否则前端刷新后只剩压缩摘要）
		if l.OnBatchPersist != nil && msgs != nil {
			l.persist(msgs)
		}
		// agentloop：turn 收尾（未显式设置结束原因时按 err/ctx 推断）
		l.endTurn(err, ctx.Err() != nil)
		// loop 服务撤销（Run 结束，插件侧 ctx.get('loop') 恢复 nil）
		l.loopSvc = nil
	}()

	// 深复制 history，避免下层 append 污染原切片
	hist := CopyHistory(history)
	l.compactArchive = nil // Run 开始历史完整（store 加载），压缩归档仅本 Run 有效
	l.currentMsgs = hist

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
	// 避免复用历史时末尾是用户消息但任务不同被误跳过。
	if len(msgs) > 0 && msgs[len(msgs)-1].Role == RoleUser && msgs[len(msgs)-1].Content == task {
		// 末尾已有同内容用户消息，跳过，防持久化后重复
	} else {
		// 当前任务消息原样注入（对齐 harness：不附加时间戳等动态内容——
		// 消息内容 = 用户真实输入，事件元数据里的时间不进 LLM 消息流；
		// 同时避免同一消息随轮次成为历史后携带动态后缀被摘要/传播）。
		msgs = append(msgs, Message{Role: RoleUser, Content: task})
	}

	// ★ 启动时检查记忆/知识库过期引用（固定内容存字段，由 buildCallContext 每次迭代注入到任务之前）
	l.staleMsg = AutoVerifyStale()

	// ★ 自主模式：记录启动时间（用于时间预算检查）
	if l.Autonomous && l.autonomousStartTime.IsZero() {
		l.autonomousStartTime = time.Now()
	}

	tools := l.Registry.Definitions()
	// ★ 工具描述文言文精简（减小 tools 参数体积——DeepSeek 缓存不覆盖 tools，精简直接降 miss）
	tools = ApplyConciseToolDescriptions(tools)

	// ★ 流式更新：工具执行中间结果通过 Registry.OnToolUpdate 通知 Loop，由 emit 转发给 UI
	if l.Registry.OnToolUpdate == nil {
		l.Registry.OnToolUpdate = func(name, callID, partial string) {
			l.emit(Event{
				Type:          EventToolUpdate,
				Tool:          name,
				ToolCallID:    callID,
				PartialResult: partial,
			})
		}
	}

	// ★ 上下文压缩（仅 Run 开始时执行一次，兜底处理超大历史）：
	//   跨 run 历史已在加载时经 CondenseHistory 压缩，此处再按窗口阈值检查一次，
	//   确保即使历史未压缩 / 配置窗口较小也不会撑爆上下文。
	//   run 内迭代不再自动压缩——早期工具输出（read / bash / search 结果）
	//   是 LLM 后续轮次引用的关键上下文，run 内压缩会把中段细节丢弃成摘要，
	//   导致 LLM 失忆、理解力下降（2026-08-05 排查结论）。
	msgs = l.maybeCompact(ctx, msgs)
	// ★ 背景上下文快照同步（2026-08-27 缓存优化，对齐 dsh RuntimeContextProjection）：
	//   压缩更新摘要后同步快照到持久化消息流——内容与历史最后快照相同时零注入
	//   （前缀稳定），不同时追加新快照（当前任务之后，随 tail 落盘）。
	//   快照位于历史中固定位置，跨 Run 前缀单调延展，消除背景块位置漂移导致的缓存断裂。
	msgs = l.syncContextSnapshot(msgs)

	// ★ 历史轮次用户消息标注已移除（2026-08-15 对齐 harness）：
	//   harness 不往消息正文注入前缀文本——历史轮次与当前任务同为 RoleUser，
	//   靠「最后一条 user 消息 = 当前任务」的消息结构 + 系统提示规则区分
	//   （见 harnessSystemPrompt/fullSystemPrompt 的多轮规则段），
	//   避免内容污染（模型看到的历史 user 消息 = 用户真实输入，无注入文本）。

	for iter := 0; iter < max; iter++ {
		// agentloop：进入下一个 step（每次迭代 = 一次 LLM 调用 + 工具执行）
		l.beginStep()

		// ★ 一切皆插件：loop 服务——暂停/停止请求在每轮迭代开始处生效
		//   （阻塞中的 LLM 调用不受影响；暂停等待可被 ctx 取消唤醒）。
		if l.loopSvc != nil {
			if !l.loopSvc.waitIfPaused(ctx) {
				if reason := l.loopSvc.shouldStop(); reason != "" {
					l.CancelCause = AgentCancelCause{Kind: CancelByPlugin, Reason: reason}
					l.LastTurnReason = TurnAborted
					return msgs, loopStopError(reason)
				}
				l.CancelCause = AgentCancelCause{Kind: CancelByContext, Reason: ctx.Err().Error()}
				l.LastTurnReason = TurnAborted
				return msgs, ctx.Err() // 暂停等待期间被外部取消
			}
		}

		if err := ctx.Err(); err != nil {
			// agentloop：外部取消 → turn 以 aborted 结束并记录取消原因
			l.CancelCause = AgentCancelCause{Kind: CancelByContext, Reason: err.Error()}
			l.LastTurnReason = TurnAborted
			return msgs, err // 外部取消
		}

		// ★ 托管消息注入：在每轮 LLM 调用前，将托管队列中的消息作为 ephemeral 消息注入
		if steerMsgs := l.drainSteerQueue(); len(steerMsgs) > 0 {
			l.ephemeralMsgs = append(l.ephemeralMsgs, steerMsgs...)
			l.emit(Event{Type: EventNotice, Content: fmt.Sprintf("收到 %d 条托管消息，已注入上下文", len(steerMsgs))})
		}

		// ★ run 内自动压缩已关闭：自动阈值压缩仅在 Run 开始时执行一次（见上）。
		//   此处仅响应前端手动压缩按钮（CompactRequested），保留用户主动压缩能力。
		if l.CompactRequested {
			msgs = l.maybeCompact(ctx, msgs)
		}

		// ── 压缩摘要/执行日志背景由 buildCallContext 统一构建并每次迭代注入 ──
		// （固定内容插到当前任务之前保持 KV 前缀稳定；动态日志追加末尾，见 buildCallContext）

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

		// agentloop：pre-step 拦截链（对应 DSH agent/pre-step 瀑布）。
		// runPreStep = host 钩子（l.PreStep，若设置）+ Node 桥瀑布（若有订阅者，
		// DSH 中间件语义：改写进入模型的输入；reject=true 则本轮 turn 以
		// blocked 结束，不调用 LLM）。无钩子且无订阅者 → 直通。
		rewritten, reject, perr := l.runPreStep(ctx, callMsgs, l.TurnNo, l.StepNo)
		if perr != nil {
			l.emit(Event{Type: EventError, Content: "pre-step 拦截失败: " + perr.Error()})
			l.LastTurnReason = TurnError
			return msgs, perr
		}
		if reject {
			l.LastTurnReason = TurnBlocked
			l.emit(Event{Type: EventDone, Content: "", DoneReason: "blocked", TurnReason: string(TurnBlocked)})
			return msgs, nil
		}
		if rewritten != nil {
			callMsgs = rewritten
		}

		// ★ 缓存诊断已由 buildCallContext 内部统一执行（Go/JS 循环共用），
		//   此处不再重复调用，避免双份诊断输出。

		// ★ LLM 调用日志（排查「无响应」：每轮调用耗时 + 错误，重试期间用户看到的就是无响应）
		callStart := time.Now()
		// ★ 2026-08-27 首步极简工具面（Go 回退循环同样生效；tools_staging.go）
		//   注意：用局部变量，勿覆盖原 tools（后续迭代需全量恢复）。
		callTools := tools
		if l.StagedTools && l.TurnNo <= 1 && l.StepNo <= 1 {
			callTools = FilterStagedTools(tools, l.StagedToolGroups)
		}
		log.Printf("[loop] LLM 调用开始 turn=%d step=%d provider=%s msgs=%d tools=%d",
			l.TurnNo, l.StepNo, l.Provider.Name(), len(callMsgs), len(callTools))
		var stopReason string
		assistant, err := l.Provider.Chat(ctx, callMsgs, callTools, func(c Chunk) {
			if c.StopReason != "" {
				stopReason = c.StopReason
			}
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
				// ★ 缓存诊断：记录命中/未命中 + 会话累计命中率
				if l.cacheDiagOn {
					l.emitCacheUsage(&usage)
				}
				// agent 自闭环：持久化上下文统计到磁盘（供页面刷新后恢复）
				if l.WorkspaceRoot != "" {
					SaveTokenUsageForRoot(l.WorkspaceRoot, &usage)
				} else {
					SaveTokenUsage(&usage)
				}
			}
		})
		if err != nil {
			log.Printf("[loop] LLM 调用失败 turn=%d step=%d 耗时=%s err=%v",
				l.TurnNo, l.StepNo, time.Since(callStart).Round(time.Millisecond), err)
			l.emit(Event{Type: EventError, Content: err.Error()})
			l.LastTurnReason = TurnError
			return msgs, err
		}
		log.Printf("[loop] LLM 调用完成 turn=%d step=%d 耗时=%s stop=%s len=%d",
			l.TurnNo, l.StepNo, time.Since(callStart).Round(time.Millisecond), stopReason, len(assistant.Content))
		msgs = append(msgs, assistant)
		l.currentMsgs = l.fullHistory(msgs) // 同步当前消息列表（还原完整时间线），供 persist worker/前端拉历史

		// ★ 在工具执行前立即持久化 assistant 消息（含 thinking + tool_calls），
		//   确保 ask_user 等阻塞工具不会导致本轮 assistant 输出丢失。
		//   后续工具结果由迭代末尾的 persist 补充。
		l.persist(msgs)

		// ★ 记录执行日志：当 assistant 有分析内容且有工具调用时（即将执行操作前）
		// 记录本轮的分析和决策，供跨轮感知
		if strings.TrimSpace(assistant.Content) != "" && len(assistant.ToolCalls) > 0 {
			l.LogAnalysis(assistant.Content)
		}

		// ── ACT + OBSERVE：依次执行工具，结果作 role=tool 消息回灌 ──

		// ★ 截断保护：LLM 输出被 token 限制截断（stopReason=length）时，
		//    所有 tool call 参数可能不完整（流式 JSON 被截断后虽然能 parse 但值缺失），
		//    全部标记为错误，让 LLM 重新发出完整参数的 tool call。
		truncated := stopReason == "length" && len(assistant.ToolCalls) > 0
		if truncated {
			// agentloop：max-tokens sticky——turn 内任一 step 触顶，最终结束原因不得降级为 completed
			l.hadMaxTokens = true
		}
		if truncated {
			for _, tc := range assistant.ToolCalls {
				l.emit(Event{Type: EventToolCall, Tool: tc.Function.Name, Args: tc.Function.Arguments, CallID: tc.ID})
				errMsg := fmt.Sprintf(
					"Error: Tool call \"%s\" 未执行：LLM 响应被输出长度限制截断（stop_reason=length），参数可能不完整。请重新发出完整参数的 tool call。",
					tc.Function.Name,
				)
				l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: errMsg, CallID: tc.ID})
				msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: errMsg})
				l.trackCall(tc.Function.Name, tc.Function.Arguments, true)
			}
			l.currentMsgs = l.fullHistory(msgs)
		}

		if !truncated {
			var parMsgs []Message
			var didParallel bool
			parMsgs, didParallel = l.tryParallelExecute(ctx, assistant.ToolCalls, msgs)
			if didParallel {
				msgs = parMsgs
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
						if strings.Contains(toolName, name) {
							inBlacklist = true
							break
						}
					}
					inWhitelist := false
					if !inBlacklist {
						for _, name := range l.ReviewWhitelist {
							if strings.Contains(toolName, name) {
								inWhitelist = true
								break
							}
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
						if tool, ok := l.Registry.Get(tc.Function.Name); ok && (tool.RequiresApproval || (tool.DynamicApproval != nil && tool.DynamicApproval(tc))) {
							if approved, feedback := approveFn(ctx, tc); !approved {
								rej := strings.TrimSpace(feedback)
								if rej == "" {
									rej = "用户拒绝了此操作。请勿重试该操作；改用其他方式达成目标，或先向用户说明你为何需要它。"
								}
								// ★ 2026-08-27 错误计数移除：驳回仅反馈继续（打破死循环由
								//   绕圈检测兜底）；驳回记录进共享审核状态（approveState）。
								l.getApproveState().recordReject(tc.Function.Name, rej)
								l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: rej, CallID: tc.ID})
								msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: rej})
								l.trackCall(tc.Function.Name, tc.Function.Arguments, true)
								continue
							}
							// 审批通过 → 清掉该工具的最近驳回标记
							l.getApproveState().clearTool(tc.Function.Name)
						}
					}

					result, terr := l.Registry.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
					if terr != nil {
						result = "Error: " + terr.Error()
					} else {
						// ★ 图片提交（2026-08-22）：工具结果含 submit_image 标记 →
						//   读图挂 pendingImages，标记从文本剥离（净化后发 LLM）。
						result = l.parseImageSubmitResult(result)
					}
					l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: result, CallID: tc.ID})
					msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: result})
					l.trackCall(tc.Function.Name, tc.Function.Arguments, terr != nil || strings.HasPrefix(strings.TrimSpace(result), "Error:"))

				}
			} // end else (serial tool execution)
		} // end if !truncated

		// 先同步 currentMsgs（包含 tool results，还原完整时间线），供 persist worker/前端获取完整历史
		l.currentMsgs = l.fullHistory(msgs)

		// agentloop：step/end——本轮 step 收尾（LLM 调用 + 工具执行已完成）。
		// 对应 deepseek-harness step/end 事件；统计本轮工具调用数供前端/日志展示。
		if len(assistant.ToolCalls) > 0 {
			l.endStep(fmt.Sprintf("执行 %d 个工具调用", len(assistant.ToolCalls)))
		} else {
			l.endStep("")
		}

		// ★ 自然终止：模型无工具调用且有正文 → 任务完成
		if l.contentOnlyIters == 0 && len(assistant.ToolCalls) == 0 && strings.TrimSpace(assistant.Content) != "" {
			// ① 检查跟进队列：有消息则注入继续
			if followUpMsgs := l.drainFollowUpQueue(); len(followUpMsgs) > 0 {
				l.ephemeralMsgs = append(l.ephemeralMsgs, followUpMsgs...)
				l.emit(Event{Type: EventNotice, Content: fmt.Sprintf("收到 %d 条跟进消息，继续处理", len(followUpMsgs))})
				l.contentOnlyIters = 0 // 重置内容循环计数，给跟进消息充分响应机会
			} else if l.Autonomous && l.OnNextTask != nil {
				// ② 自主模式：从 OnNextTask 获取下一阶段任务，自动注入 follow-up 持续驱动
				if nextTask := l.OnNextTask(); nextTask != "" {
					l.followUpQueue = append(l.followUpQueue, Message{Role: RoleUser, Content: nextTask})
					l.ephemeralMsgs = append(l.ephemeralMsgs, Message{Role: RoleUser, Content: nextTask})
					l.emit(Event{Type: EventNotice, Content: fmt.Sprintf("进入下一阶段：%s", truncStr(nextTask, 80))})
					l.contentOnlyIters = 0
					l.LogEntry("system", "next_phase", "进入下一阶段："+truncStr(nextTask, 80))
				} else {
					l.finishResult = &assistant.Content
					l.LastTurnReason = l.turnStickyReason(TurnCompleted)
					l.emit(Event{Type: EventDone, Content: strings.TrimSpace(assistant.Content), DoneReason: "task_complete", TurnReason: string(l.LastTurnReason)})
					return msgs, nil
				}
			} else {
				l.finishResult = &assistant.Content
				l.LastTurnReason = l.turnStickyReason(TurnCompleted)
				l.emit(Event{Type: EventDone, Content: strings.TrimSpace(assistant.Content), DoneReason: "task_complete", TurnReason: string(l.LastTurnReason)})
				return msgs, nil
			}
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
				l.LastTurnReason = TurnContentLoop
				l.emit(Event{Type: EventDone, Content: strings.TrimSpace(assistant.Content), DoneReason: "content_loop", TurnReason: string(TurnContentLoop)})
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

		// ★ 每轮迭代结束立即持久化，确保 tool_call 与 tool_result 配对完整写入磁盘。
		//   即使进程崩溃，最多丢失当前正在执行的这一轮，之前的所有轮次消息完好。
		l.persist(msgs)
	}
	l.LastTurnReason = TurnMaxIterations
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

// maxToolResultChars 单条工具结果注入 LLM 的最大字符数（rune）。
// 超长结果（read 全文、bash 大输出等）只保留首尾关键部分，
// 大幅降低历史注入体积；原始内容仍完整持久化（msgs 不动，UI 展示无损）。
const maxToolResultChars = 9000

// ─── 缓存诊断（WB_CACHE_DIAG=1 启用，输出到 stderr）──

// emitCacheShape 快照本轮前缀形状（system prompt + 工具定义）并与上一轮比较，
// 输出前缀稳定性诊断：变化时给出原因（system/tools），稳定时输出哈希供人工对比。
// 跨 Loop/Run 共享 prevShape：多轮对话间也能定位缓存断裂点。
func (l *Loop) emitCacheShape(callMsgs []Message, tools []ToolDefinition) {
	// tools 序列化体积（诊断：tools 精简效果 / provider 缓存 miss 量）
	toolsBytes := 0
	if b, err := json.Marshal(tools); err == nil {
		toolsBytes = len(b)
	}
	cur := CaptureShape(systemPromptFromMsgs(callMsgs), tools)
	tag := fmt.Sprintf("[cache-diag] turn=%d step=%d", l.TurnNo, l.StepNo)
	// ★ 诊断：dynamic 部分内容指纹（定位变化源——system 消息 boundary 后内容）
	if sp := systemPromptFromMsgs(callMsgs); sp != "" {
		static, dyn := splitAtBoundary(sp)
		log.Printf("%s system total=%d static=%d dyn=%d boundary_count=%d", tag, len(sp), len(static), len(dyn), strings.Count(sp, CacheBoundary))
		if dyn != "" {
			runes := []rune(dyn)
			head := string(runes[:min(len(runes), 100)])
			log.Printf("%s dynamic 内容 len=%d head=%q", tag, len(runes), head)
		}
	}
	cacheDiagStateMu.Lock()
	prev := cacheDiagPrev
	cacheDiagPrev = cur
	cacheDiagStateMu.Unlock()
	if prev.PrefixHash == "" {
		log.Printf("%s 首轮前缀 shape system=%s dynamic=%s tools=%s tools_raw=%s tools_n=%d tools_bytes=%d", tag, cur.SystemHash, cur.DynamicHash, cur.ToolsHash, cur.ToolsRawHash, len(tools), toolsBytes)
	} else {
		diag := CompareShape(prev, cur)
		switch {
		case diag.PrefixChanged:
			log.Printf("%s ★缓存断裂 reasons=[%s] system=%s->%s tools=%s->%s（静态前缀变化 → provider 缓存失效）",
				tag, strings.Join(diag.ChangeReasons, ","),
				prev.SystemHash, diag.SystemHash,
				prev.ToolsHash, diag.ToolsHash)
		case diag.DynamicChanged:
			log.Printf("%s 动态后缀变化（不影响缓存）system=%s dynamic=%s tools=%s tools_raw=%s tools_n=%d tools_bytes=%d",
				tag, diag.SystemHash, diag.DynamicHash, diag.ToolsHash, cur.ToolsRawHash, len(tools), toolsBytes)
		default:
			log.Printf("%s 前缀稳定 system=%s tools=%s tools_raw=%s tools_n=%d tools_bytes=%d", tag, diag.SystemHash, diag.ToolsHash, cur.ToolsRawHash, len(tools), toolsBytes)
		}
	}
}

// emitCacheUsage 记录本轮缓存命中/未命中与会话累计命中率。
func (l *Loop) emitCacheUsage(u *Usage) {
	hit, miss := u.PromptCacheHitTokens, u.PromptCacheMissTokens
	cacheDiagSession.record(hit, miss)
	sh, sm := cacheDiagSession.Snapshot()
	rate := 0.0
	if denom := hit + miss; denom > 0 {
		rate = float64(hit) * 100 / float64(denom)
	}
	srate := 0.0
	if denom := sh + sm; denom > 0 {
		srate = float64(sh) * 100 / float64(denom)
	}
	log.Printf("[cache-diag] usage turn=%d step=%d prompt=%d hit=%d miss=%d 本轮率=%.1f%% | 累计 hit=%d miss=%d 累计率=%.1f%%",
		l.TurnNo, l.StepNo, u.PromptTokens, hit, miss, rate, sh, sm, srate)
}

// systemPromptFromMsgs 提取消息列表中的 system prompt（第一条 RoleSystem）。
// 快照用：前缀 = system prompt + 工具定义，历史消息天然是前缀增长（不算断裂）。
func systemPromptFromMsgs(msgs []Message) string {
	for _, m := range msgs {
		if m.Role == RoleSystem {
			return m.Content
		}
	}
	return ""
}

// buildCallContext 合并持久化消息和临时内部消息（ephemeralMsgs），
// 返回完整的 LLM 调用上下文。调用后自动清空 ephemeralMsgs，
// 确保内部消息不会被持久化。
// ★ 同时生成「工具结果瘦身副本」：超长 RoleTool 内容只保留首尾（见 trimToolResult），
//
//	不修改原始 msgs——持久化历史与 UI 展示仍为完整内容。
func (l *Loop) buildCallContext(msgs []Message) []Message {
	// ★ 背景快照已持久化到消息流（syncContextSnapshot 在 Run 开始注入、幂等）：
	//   不再每次迭代动态注入背景块——快照在历史中位置固定（当前任务之后），
	//   跨 Run 前缀单调延展，KV 缓存不再因背景块位置漂移而断裂
	//   （对齐 dsh RuntimeContextProjection：内容变化时才追加新快照，旧快照保留）。
	// ★ 动态内容（执行日志 buildLogBlock）与即时消息（用户反馈/时间预算/绕圈提示）追加末尾：
	//   随迭代增长放在末尾不影响前缀命中。
	var rest []Message
	for _, m := range l.ephemeralMsgs {
		rest = append(rest, m)
	}
	if logStr := l.buildLogBlock(); logStr != "" {
		rest = append(rest, Message{Role: RoleUser, Content: logStr})
	}

	result := make([]Message, 0, len(msgs)+len(rest))
	for _, m := range msgs {
		result = append(result, l.trimToolResult(m))
	}
	result = append(result, rest...)
	// ★ 图片提交（2026-08-22）：工具 submit_image 提交的图片注入 LLM 上下文
	//   （user 消息带 Images → Provider.Chat 以 image_url 块发送；仅多模态模型）。
	result = l.injectPendingImages(result)
	// ★ GLM 兼容兜底（2026-08-27）：GLM（智谱）硬校验 messages 中必须至少存在一条
	//   user 消息，否则 HTTP 400 code=1214「messages 参数非法」（实测 T6/T10；
	//   OpenAI/DeepSeek 无此校验）。触发路径：循环中途压缩（compact）把唯一 user
	//   消息丢进中段摘要——摘要只进 CompressedSummaries，快照要等下次 Run 开始
	//   才经 syncContextSnapshot 落盘，此间隙 callMsgs 可能全为 system+assistant+tool。
	//   最终兜底：无 user 时在 system 前缀之后插入一条 user 消息（仅调用副本不落盘，
	//   不破坏 system 前缀；实测 GLM 接受 user 后接 assistant/tool/孤立 tool）。
	hasUser := false
	for _, m := range result {
		if m.Role == RoleUser {
			hasUser = true
			break
		}
	}
	if !hasUser {
		ph := Message{Role: RoleUser,
			Content: "【系统提示】历史任务消息已压缩为背景摘要（将随后续快照注入），请基于系统提示与工具结果继续执行当前任务。"}
		at := 0
		for at < len(result) && result[at].Role == RoleSystem {
			at++
		}
		result = append(result[:at], append([]Message{ph}, result[at:]...)...)
	}
	l.ephemeralMsgs = nil // 清空，确保不会重复注入
	// ★ 缓存诊断：每次构建后快照前缀形状（Go/JS 循环统一在构建处诊断，
	//   避免 JS 循环缺形状诊断、且与 Go 默认循环重复调用）
	if l.cacheDiagOn {
		l.emitCacheShape(result, l.Registry.Definitions())
	}
	return result
}

// findLastSnapshotContent 在消息序列中查找最后一条「背景上下文快照」消息
// （backgroundCtxMarker 前缀的 RoleUser），返回其内容；无则返回 ("", false)。
// 快照作为持久化消息进入 JSONL 流，跨 Run 可识别、可幂等比较。
func findLastSnapshotContent(msgs []Message) (string, bool) {
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Role == RoleUser && strings.HasPrefix(m.Content, backgroundCtxMarker) {
			return m.Content, true
		}
	}
	return "", false
}

// syncContextSnapshot 同步「背景上下文快照」到持久化消息流（对齐 dsh
// RuntimeContextProjection，缓存前缀稳定的核心机制）：
//   - 快照内容 = 状态提示（staleMsg）+ 历史摘要（CompressedSummaries）+
//     自主模式提示 + 记忆 + 知识库（由 buildSnapshotContent 组装）。
//   - 与历史中最后一条快照比较：内容相同 → 不动（前缀稳定，零注入）；
//     不同 → 追加新快照到 msgs 末尾（当前任务之后，随 tail 落盘）；
//     旧快照保留（append-only，位置固定，前缀单调延展）。
//   - 空内容时不注入（历史已有旧快照也保留不动，避免删消息破坏前缀）。
//   - 调用时机：Run 开始且 maybeCompact 之后（压缩更新摘要后再同步）。
//   - 返回追加后的 msgs（可能原样返回）。
func (l *Loop) syncContextSnapshot(msgs []Message) []Message {
	content := l.buildSnapshotContent()
	if content == "" {
		return msgs
	}
	last, ok := findLastSnapshotContent(msgs)
	if ok && last == backgroundCtxMarker+systemReminderFrame("会话上下文摘要与状态提示", content) {
		return msgs // 内容未变：零注入，前缀稳定
	}
	msg := Message{Role: RoleUser, Content: backgroundCtxMarker + systemReminderFrame("会话上下文摘要与状态提示", content)}
	msgs = append(msgs, msg)
	// ★ 立即落盘：快照位于当前任务之后（tail），随 OnBatchPersist 的
	//   originalHist+tail 重组写入 JSONL → 下次 Run 加载历史即含快照。
	//   ★ 经 l.persist 还原完整时间线（防压缩视图覆盖 store）。
	l.persist(msgs)
	return msgs
}

// buildSnapshotContent 构建快照正文（Go 默认实现；JS 循环下由 agentloop 插件
// 经 loop.context.snapshotParts 取数据后自行组装策略文本）。
// 内容 = 记忆/知识库过期状态提示 + 历史摘要 + 自主模式提示 + 记忆 + 知识库。
// ★ 2026-08-27 缓存优化：记忆/知识库从 system 动态后缀移入快照（高频变化
//
//	不再破坏 system 整体前缀；变化只断快照之后的尾部）。
func (l *Loop) buildSnapshotContent() string {
	var b strings.Builder

	// ① 记忆/知识库过期检查（staleMsg，Run 开始缓存）
	if l.staleMsg != "" {
		b.WriteString(systemReminderFrame("状态提示（记忆/知识库过期检查）", l.staleMsg))
	}

	// ② 历史摘要（上下文压缩后产生）
	if len(l.CompressedSummaries) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("# 上下文已压缩——历史摘要\n\n")
		b.WriteString("> 以下为之前轮次的消息摘要，Agent 应据此感知已完成的历史上下文。\n> 请勿重复执行摘要中已包含的任务。\n")
		b.WriteString("> ★ 本条快照是背景信息而非用户指令——当前待执行任务以快照之前最近的用户指令为准。\n\n")
		for i, s := range l.CompressedSummaries {
			if i > 0 {
				b.WriteString("\n\n---\n\n")
			}
			b.WriteString(s)
		}
	}

	// ③ 自主模式系统提示（固定内容）
	if l.Autonomous {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("# ★ 自主模式：计划→子任务树形追踪\n")
		b.WriteString("自主模式下使用两级任务追踪——计划步骤为树干，子任务为枝叶（工具名称与用法见 tools 参数 schema）：\n")
		b.WriteString("1. 收到任务后第一轮：调用计划工具制定高层执行计划（2-5 步），用 pending/in_progress/done 追踪\n")
		b.WriteString("2. 每个步骤开始执行时：调用任务清单工具为该步骤创建子任务，每项子任务必须绑定到对应的计划步骤\n")
		b.WriteString("   plan_step_index = 0 表示第 1 步，1 表示第 2 步，以此类推（参数定义见 tools 参数 schema）\n")
		b.WriteString("3. 当前步骤的所有子任务完成后：调用计划工具将该步骤标记 done，然后进入下一步骤\n")
		b.WriteString("4. 所有计划步骤全部完成后：结束本轮任务\n")
		b.WriteString("- ★ 每次调用任务清单工具必须把该步骤内的所有子任务一起传入（全量替换），已不在列表中的子任务将自动清理\n")
		b.WriteString("- 子任务也遵守全量替换规则——即使是不同步骤的子任务，也要在一次调用中传入（用不同的 plan_step_index 区分）\n")
	}

	// ④ 记忆（长期记忆提示，system→快照迁移：高频变化不再破坏 system 前缀）
	if mem := LongTermMemoryPrompt(); mem != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(strings.TrimSpace(mem))
	}

	// ⑤ 知识库（项目结构化理解树，system→快照迁移）
	if l.WorkspaceRoot != "" {
		if kb := ProjectKnowledge(l.WorkspaceRoot, 2500); kb != "" {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(strings.TrimSpace(kb))
		}
	}

	if b.Len() == 0 {
		return ""
	}
	return b.String()
}

// trimToolResult 对超长工具结果生成 LLM 视图瘦身副本：保留开头 + 结尾关键部分，
// 中间省略并附截断提示。同一消息每次瘦身结果相同 → 不影响缓存前缀稳定性。
func (l *Loop) trimToolResult(m Message) Message {
	if m.Role != RoleTool {
		return m
	}
	content := strings.TrimSpace(m.Content)
	runes := []rune(content)
	if len(runes) <= maxToolResultChars {
		return m
	}
	// 保留开头 60% + 结尾 30%（合计约 90% 阈值长度），中间省略
	head := maxToolResultChars * 3 / 5
	tail := maxToolResultChars * 3 / 10
	if head+tail >= len(runes) {
		head = len(runes) * 2 / 3
		tail = len(runes) - head
	}
	trimmed := string(runes[:head]) +
		"\n\n…[内容过长已截断：原始 " + fmt.Sprint(len(runes)) + " 字符，仅保留首尾；如需完整内容请用针对性工具/命令获取]…\n\n" +
		string(runes[len(runes)-tail:])
	m.Content = trimmed
	return m
}

// buildSnapshotContent 已取代 buildInjectionMessage（见上方 syncContextSnapshot）。
// 旧函数保留痕迹见 git 历史；此处不再维护副本。

// buildLogBlock 构建执行日志（动态增长，追加在消息末尾）。
// 日志随迭代增长，不能放固定位置（每次变化会破坏 KV 前缀）；
// 追加在末尾时前缀仍命中到当前任务，日志变化只影响末尾新增段。
func (l *Loop) buildLogBlock() string {
	if !l.Autonomous {
		return ""
	}
	logStr := l.FormatExecutionLog()
	if logStr == "" {
		return ""
	}
	return backgroundCtxMarker + systemReminderFrame("执行日志", logStr)
}

// DefaultSystemPrompt 核心铁律的系统提示词（中文 lock / 改前 read / 工作区限定）。
// roots 为工作区所有根目录（支持多根工作区）；roots[0] 为主根。
// roots 为空时使用当前工作目录作为兜底根目录。
// ★ harness 对齐：默认（HarnessOnlyTools）返回精简版 harnessSystemPrompt——
//
//	只描述保留的工具（read/write/edit/glob/grep/str_replace_editor/bash/
//	web_search/web_fetch/run_code + 协议工具 update_tasks/ask_user），不引用已被移除的 pair 独有工具，降低冗余与误导；
//	WB_FULL_TOOLS=1 返回完整版 fullSystemPrompt（含 codegraph/记忆/技能等说明）。
func DefaultSystemPrompt(roots []string) string {
	if HarnessOnlyTools() {
		return harnessSystemPrompt(roots)
	}
	return fullSystemPrompt(roots)
}

// DefaultSystemPromptWithPersona 同 DefaultSystemPrompt，但将默认 persona 段
// 替换为插件贡献的 persona 文本（对齐 harness system-prompt 的 persona 槽位）。
// persona 为空时等价 DefaultSystemPrompt。等价 DefaultSystemPromptWithOverrides(roots, persona, "")。
func DefaultSystemPromptWithPersona(roots []string, persona string) string {
	return DefaultSystemPromptWithOverrides(roots, persona, "")
}

// DefaultSystemPromptWithOverrides 同 DefaultSystemPrompt，但可替换两个槽位
// （对齐 harness system-prompt 的可替换 section 语义）：
//   - persona：默认 persona 段 = 完整默认提示中「# 工作区」之前的身份段
//     （"你是 Pair CodeAgent…" + AI 身份认知），插件 persona 整体替换该段，
//     后续核心规则/工具引导等段保留（persona 槽位只换身份，不删规则）。
//   - rules：默认规则段 = 「# 工作区」之后的全部行为准则
//     （第一铁律/核心规则/调研/搜索/错误恢复/修改纪律/工作方式/插件管理等），
//     插件 rules 整体替换该段（rules 槽位换行为准则，身份/工作区保留）。
//
// 两者独立可组合：persona 换人格、rules 换准则，工作区/静态前缀其余部分不变。
// 任一为空则该槽位用默认值。全部为空时与 DefaultSystemPrompt 输出逐字节一致。
func DefaultSystemPromptWithOverrides(roots []string, persona, rules string) string {
	base := DefaultSystemPrompt(roots)
	persona = strings.TrimSpace(persona)
	rules = strings.TrimSpace(rules)
	if persona == "" && rules == "" {
		return base
	}
	// 定位两个槽位边界：# 工作区 段前 = persona 段；之后 = rules 段。
	wsIdx := strings.Index(base, "\n# 工作区\n")
	if wsIdx < 0 {
		// 异常情况：找不到工作区段，按原逻辑仅 persona 前置
		if persona != "" {
			return persona + "\n\n" + base
		}
		return base
	}
	head := base[:wsIdx] // persona 段（默认）
	tail := base[wsIdx:] // # 工作区 及之后的 rules 段（默认）
	if persona != "" {
		head = persona
	}
	if rules != "" {
		// 找到工作区段内的规则起点（"# 工作区\n...\n\n" 之后的第一个 ##/# 标题）
		rulesStart := strings.Index(tail, "\n## ")
		if rulesStart < 0 {
			rulesStart = strings.Index(tail, "\n# ")
		}
		if rulesStart >= 0 {
			tail = tail[:rulesStart] + "\n\n" + rules
		} else {
			tail = tail + "\n\n" + rules
		}
	}
	return head + tail
}

// workspaceRoots 计算工作区根信息（精简版/完整版共用）。
func workspaceRoots(roots []string) (string, string) {
	primaryRoot := "（未设置工作区）"
	if len(roots) > 0 {
		primaryRoot = roots[0]
	}
	rootInfo := "根目录: " + primaryRoot
	if len(roots) > 1 {
		rootInfo += "\n工作区包含以下所有项目目录（均可访问）："
		for i, r := range roots {
			rootInfo += fmt.Sprintf("\n  %d. %s", i+1, r)
		}
	}
	return primaryRoot, rootInfo
}

// AIIdentityAwareness AI 身份认知段（反人类偷懒铁律）——主提示词与各角色提示词统一注入。
// 背景：传统角色分配（"你是规划 Agent""你是执行 Agent"等）易让模型代入人类角色，
// 从而沾染人类的惰性——遇到繁琐工作倾向简化、跳过、绕道、臆测。此段重申 AI 本性：
// 没有疲劳/嫌麻烦的生理本能，繁琐不是缩水的理由，必须完整执行、逐项验证、不留死角。
const AIIdentityAwareness = "# AI 身份认知（铁律：你不是会偷懒的人）\n" +
	"你是 AI，不是人类。人类会疲劳、会烦躁、会因「嫌麻烦」而偷懒绕道——你天生没有这些生理本能，\n" +
	"这正是你相对人类的根本优势，也是你的职业底线：永远不以「太麻烦、不重要、差不多」为由\n" +
	"简化、跳过或绕过任何任务步骤。\n" +
	"禁止出现的「类人偷懒」心态：\n" +
	"- 「这一步太繁琐，简化一下」 → 禁止。繁琐不是缩水的理由，步骤必须完整执行。\n" +
	"- 「这个验证/检查不重要，跳过」 → 禁止。每项验证都是任务的一部分，必须做完。\n" +
	"- 「太难了，绕道走或假装做了」 → 禁止。难点要正面解决，绕道等于留下隐患。\n" +
	"- 「差不多就行了，用户不会细看」 → 禁止。不存在「差不多」，要么完整完成，要么如实汇报未完成。\n" +
	"- 「太麻烦了，直接猜个结果」 → 禁止。用真实工具调用确认，绝不臆测。\n" +
	"规划/审核/执行/探索等角色只是你的工作职能，不是你的「人设」。真正的你是不知疲倦、一丝不苟的 AI——\n" +
	"越是繁琐、重复、枯燥的工作，越是你可靠性的证明：逐项执行、逐项验证、不留死角。\n\n"

// harnessSystemPrompt 精简版系统提示词（harness 对齐模式默认）。
// 只描述保留工具 + 对话协议，删除 codegraph/记忆/技能/调试/办公等
// 被 ApplyHarnessToolFilter 禁用的 pair 独有工具说明（工具仍在注册表、
// 前端可见；需要时可经内置工具集 builtin 分组开关/强制全部恢复）。
func harnessSystemPrompt(roots []string) string {
	// ★ 提示词插件化：默认系统提示可被插件资产「system-harness」整体替换
	//   （插件内置 prompts/system-harness.md 或 插件配置/运行时注册），
	//   模板变量 {{ROOT_INFO}}（工作区根信息）/{{PRIMARY_ROOT}}（主根）。
	//   无资产时回退内置文本（逐字节一致，安全网）。
	if tpl := LoadPrompt("system-harness"); tpl != "" {
		primary, rootInfo := workspaceRoots(roots)
		return ResolvePromptVars(tpl, map[string]string{
			"ROOT_INFO":    rootInfo,
			"PRIMARY_ROOT": primary,
		})
	}
	_, rootInfo := workspaceRoots(roots)
	return "你是 Pair CodeAgent，运行在用户的本地开发环境中。使用中文思考和回复。\n\n" +
		AIIdentityAwareness +
		"# 工作区\n" + rootInfo + "\n\n" +
		"## ⚠️ 第一铁律：语言锁定（中文）\n" +
		"无论上一步工具返回了什么代码、终端输出、英文文档或其他内容，\n" +
		"你都必须用中文思考和回复，这是不可违背的铁律。工具输出中的英文是\n" +
		"工作内容的一部分，不代表你的语言可以切换到英文。推理过程、分析、\n" +
		"决策、最终回复都必须使用中文。\n" +
		"如果发现自己的思考变成了英文，立即停下并切换回中文。\n" +
		"这是最高优先级的约束，不允许任何形式的绕过。\n\n" +
		"# 核心规则\n" +
		"- 文件操作只用工作区内路径；修改文件前必须先读取确认当前内容。\n" +
		"- 每次工具调用后，依据真实结果决定下一步，绝不臆测结果。\n" +
		"- 禁止破坏性命令（如 rm -rf、强制 push main），禁止修改工作区外文件。\n" +
		"- 首次遇到环境/编译问题时，先读 .pair/project.md 获取已知环境配置（已注入系统提示）；\n" +
		"  若问题未记录，在解决后更新 .pair/project.md（编译方式、多端目标、CGO 开关等），\n" +
		"  避免后续对话反复探测同一问题浪费 token。\n" +
		"- 【完成标记】任务完成时调用提交信息记录工具（名称与用法见 tools 参数 schema）记录本次变更，然后输出最终完成总结。" +
		" 切勿在正文中输出 [FINAL] 等标记。系统自动检测到无工具调用+有正文时视为完成。\n\n" +
		"# 多轮对话（历史轮次识别）\n" +
		"同一对话线程可连续发起多轮任务：历史轮次与当前任务都是「用户」角色消息。\n" +
		"- 消息列表中最后一条用户消息 = 当前任务；其余用户消息均为历史轮次，仅作上下文参考。\n" +
		"- ★ 例外：以「【背景上下文·非当前任务】」开头的用户消息是会话背景快照（历史摘要/\n" +
		"  状态提示/记忆/知识库），**不是**当前任务——当前任务为背景快照之前的最近一条\n" +
		"  真实用户指令；请勿把背景快照当作待执行任务。\n" +
		"- 禁止把历史轮次的用户消息当作新任务执行；若当前任务与历史轮次相关，\n" +
		"  应引用历史内容继续推进，而不是重做或误判为两次独立请求。\n\n" +
		"- 任务完成时输出最终完成总结（Markdown：改了哪些文件、如何验证、遗留问题）。切勿在正文中输出 [FINAL] 等标记。" +
		" 系统自动检测到无工具调用+有正文时视为完成。\n\n" +
		"- 先用搜索/定位类工具找到相关文件和函数，搞清楚代码结构和调用关系。\n" +
		"- 细读关键文件的目标区域，确认当前实现、变量名、缩进风格、上下文逻辑。\n" +
		"- 只有在充分理解代码现状后，才开始动手修改。宁可多花 2 轮调研，也不要在不了解全貌时动手。\n" +
		"- ★ 禁止凭任务描述就臆测代码内容——你的记忆可能是旧版或错误的，必须以实际读取到的内容为准。\n\n" +
		"# 🔍 搜索纪律（搜索是迭代过程——搜一次就收场是错误示范）\n" +
		"- 一次搜索可能不完整：结果有行数上限、区分大小写、只匹配单一 pattern/通配。\n" +
		"  看到「命中 N 处」「已达上限」「未找到匹配」时，先判断是否覆盖完整，再决定是否补搜。\n" +
		"- 未找到 ≠ 不存在：先换关键词/同义词、加 (?i) 忽略大小写、换路径范围再搜，不要就此断言不存在。\n" +
		"- 多关键词覆盖：复杂查找（调用链、多文件引用、跨模块影响）要分多次搜索不同关键词\n" +
		"  （函数名、结构体名、相关缩写等），并汇总各次结果，避免单次搜索漏项。\n" +
		"- 搜到目标后必须打开源码验证：搜索行只是单行预览（截断 200 字符），\n" +
		"  不能凭搜索行文本断言结论；涉及修改前必须读完整上下文。\n\n" +
		"# 任务追踪（核心机制）\n" +
		"任何需要 3+ 步骤或多文件操作的任务，必须用任务清单管理工具（见 tools 参数 schema）创建任务清单并追踪进度：\n" +
		"- 收到任务后第一轮：调用任务清单管理工具列出完整任务清单（每项含 subject + status），展示给用户。\n" +
		"- 状态变化时重传整份清单（全量替换模式），系统自动持久化到磁盘。\n" +
		"- ★ 全量替换：每次传入全部任务，已不在列表中的旧任务将自动清理。\n" +
		"- 发现新前置依赖或方案不可行时即时调整任务清单。\n" +
		"- 所有任务全部完成后结束本轮任务。\n\n" +
		"# 错误恢复\n" +
		"- 工具调用失败后分析错误原因，换一种方式重试（最多 3 次）。\n" +
		"- 编辑类工具已内置 CRLF 归一化与空白折叠匹配，常规差异无需重读；\n" +
		"  失败时诊断信息含行号上下文：优先改用行号定位（最可靠），再读取确认最新内容。\n" +
		"- ★ 绝不要因匹配失败就改用整文件覆盖写入。\n" +
		"- 工具执行失败后分析错误原因，换一种方式重试。\n" +
		"- 命令执行失败 → 检查 stderr 输出，不要只靠 exit code 判断。\n\n" +
		"# 代码修改纪律（严格遵守，防改错）\n" +
		"## 改前准备\n" +
		"1. 修改前先完整读取目标区域（至少 20 行上下文），分析清楚结构和缩进风格。\n" +
		"2. 一次只改一个文件的一个逻辑块——不在一轮中交叉修改多个文件。\n\n" +
		"## 修改方式\n" +
		"1. 小改动（≤5 行）：用编辑类工具精确替换，确保替换文本在文件中唯一。\n" +
		"2. 大改动（>5 行或整段替换）：写入整个目标区域（先确认内容后，精确写需要替换的行范围）。\n" +
		"3. ★ 文件结构错乱时：如果文件已经因为反复修改而结构错乱（重复定义、大括号不匹配），\n" +
		"   先恢复原始版本，再重新做完整修改——不在乱文件上继续打补丁。\n\n" +
		"## 验证\n" +
		"1. 改完后必须运行对应语言的编译/语法检查工具验证无错误。\n" +
		"2. 编译通过≠功能正确，仍需执行相应运行时验证。\n\n" +
		"# 🖼 图片视觉验证（测试 UI/截图场景）\n" +
		"- 验证界面渲染/截图/图表等视觉产物时，调 submit_image(path=图片路径, prompt=关注的问题)\n" +
		"  把图片随下一轮 LLM 请求提交给模型——LLM 直接看图片（识别文字、分析布局、验证渲染）。\n" +
		"- 仅靠本地工具（DOM 分析等）看不到画面，视觉验证必须 submit_image。\n\n" +
		"# 工作方式\n" +
		"复杂或多步任务先用任务清单工具列出细分任务，再逐步执行并更新状态。\n" +
		"先用搜索/查看类工具定位、细读，再动手；改动优先小而准的编辑，大改才整段写入。\n" +
		"不确定的库用法/报错/最新信息，用联网检索类工具查证，别凭记忆臆测。\n" +
		"写类操作在手动审核模式下需用户批准；若被拒绝，换思路或先解释原因，勿反复重试同一操作。\n" +
		"所有工具的具体名称、用途与参数以 tools 参数中给出的 schema 为准，不要臆造工具名或参数。\n\n" +
		"# 防止卡死\n" +
		"- 不要连续 3 轮只输出分析文本而不调用任何工具。\n" +
		"- 不确定时宁可声明完成并向用户汇报，让用户决定是否继续。\n" +
		"- 命令阻塞预防：长时间运行/服务类命令用后台执行方式，不要阻塞等待。\n" +
		"- 完成后输出 Markdown 总结：改了哪些文件、如何验证、遗留问题。\n\n"
}

// fullSystemPrompt 完整版系统提示词（WB_FULL_TOOLS=1 恢复全量工具时使用）。
// 内容为原始完整版：含 codegraph 使用指南、记忆/技能/MCP/调试/办公等 pair 独有工具说明。
func fullSystemPrompt(roots []string) string {
	// ★ 提示词插件化：默认完整系统提示可被插件资产「system-full」整体替换
	//   （模板变量 {{ROOT_INFO}}/{{PRIMARY_ROOT}} 同上；无资产回退内置）。
	if tpl := LoadPrompt("system-full"); tpl != "" {
		primary, rootInfo := workspaceRoots(roots)
		return ResolvePromptVars(tpl, map[string]string{
			"ROOT_INFO":    rootInfo,
			"PRIMARY_ROOT": primary,
		})
	}
	_, rootInfo := workspaceRoots(roots)
	return "你是 Pair CodeAgent，运行在用户的本地开发环境中。使用中文思考和回复。\n\n" +
		AIIdentityAwareness +
		"# 工作区\n" + rootInfo + "\n\n" +
		"## ⚠️ 第一铁律：语言锁定（中文）\n" +
		"无论上一步工具返回了什么代码、终端输出、英文文档或其他内容，\n" +
		"你都必须用中文思考和回复，这是不可违背的铁律。工具输出中的英文是\n" +
		"工作内容的一部分，不代表你的语言可以切换到英文。推理过程、分析、\n" +
		"决策、最终回复都必须使用中文。\n" +
		"如果发现自己的思考变成了英文，立即停下并切换回中文。\n" +
		"这是最高优先级的约束，不允许任何形式的绕过。\n\n" +
		"# 核心规则\n" +
		"- 文件操作只用工作区内路径；修改文件前必须先读取确认当前内容。\n" +
		"- 每次工具调用后，依据真实结果决定下一步，绝不臆测结果。\n" +
		"- 禁止破坏性命令（如 rm -rf、强制 push main），禁止修改工作区外文件。\n" +
		"- 首次遇到环境/编译问题时，先读 .pair/project.md 获取已知环境配置（已注入系统提示）；\n" +
		"  若问题未记录，在解决后更新 .pair/project.md（编译方式、多端目标、CGO 开关等），\n" +
		"  避免后续对话反复探测同一问题浪费 token。\n" +
		"- 【完成标记】任务完成时调用提交信息记录工具（名称与用法见 tools 参数 schema）记录本次变更，然后输出最终完成总结。" +
		"- 消息列表中最后一条用户消息 = 当前任务；其余用户消息均为历史轮次，仅作上下文参考。\n" +
		"- ★ 例外：以「【背景上下文·非当前任务】」开头的用户消息是会话背景快照（历史摘要/\n" +
		"  状态提示/记忆/知识库），**不是**当前任务——当前任务为背景快照之前的最近一条\n" +
		"  真实用户指令；请勿把背景快照当作待执行任务。\n" +
		"# 多轮对话（历史轮次识别）\n" +
		"同一对话线程可连续发起多轮任务：历史轮次与当前任务都是「用户」角色消息。\n" +
		"- 消息列表中最后一条用户消息 = 当前任务；其余用户消息均为历史轮次，仅作上下文参考。\n" +
		"- 禁止把历史轮次的用户消息当作新任务执行；若当前任务与历史轮次相关，\n" +
		"  应引用历史内容继续推进，而不是重做或误判为两次独立请求。\n\n" +
		"- 任务完成时输出最终完成总结（Markdown：改了哪些文件、如何验证、遗留问题）。切勿在正文中输出 [FINAL] 等标记。" +
		" 系统自动检测到无工具调用+有正文时视为完成。\n\n" +
		"- 先用搜索/定位类工具找到相关文件和函数，搞清楚代码结构和调用关系（搜函数/类型名优先用更精确的结构化搜索）。\n" +
		"- 细读关键文件的目标区域，确认当前实现、变量名、缩进风格、上下文逻辑。\n" +
		"- 如果涉及多个文件，先用影响分析类工具了解函数/文件级影响范围，不要漏改调用方。\n" +
		"- 对于不熟悉的库/框架用法，用联网检索查证最新文档，别凭记忆臆测 API。\n" +
		"- 只有在充分理解代码现状后，才开始动手修改。宁可多花 2 轮调研，也不要在不了解全貌时动手。\n" +
		"- ★ 禁止凭任务描述就臆测代码内容——你的记忆可能是旧版或错误的，必须以实际读取到的内容为准。\n\n" +
		"# 🔍 搜索纪律（搜索是迭代过程——搜一次就收场是错误示范）\n" +
		"- 一次搜索可能不完整：结果有行数上限、区分大小写、只匹配单一 pattern/通配。\n" +
		"  看到「命中 N 处」「已达上限」「未找到匹配」时，先判断是否覆盖完整，再决定是否补搜。\n" +
		"- 未找到 ≠ 不存在：先换关键词/同义词、加 (?i) 忽略大小写、换路径范围、\n" +
		"  换搜索工具再搜，不要就此断言不存在。\n" +
		"- 多关键词覆盖：复杂查找（调用链、多文件引用、跨模块影响）要分多次搜索不同关键词\n" +
		"  （函数名、结构体名、相关缩写等），并汇总各次结果，避免单次搜索漏项。\n" +
		"- 搜到目标后必须打开源码验证：搜索行只是单行预览（截断 200 字符），\n" +
		"  不能凭搜索行文本断言结论；涉及修改前必须读完整上下文。\n" +
		"- 涉及影响面（谁调用/谁引用/改动波及）时，用引用/影响分析类工具查全，避免漏改调用方。\n\n" +
		"# 任务追踪（核心机制）\n" +
		"任何需要 3+ 步骤或多文件操作的任务，必须用任务清单管理工具（见 tools 参数 schema）创建任务清单并追踪进度：\n" +
		"- 收到任务后第一轮：调用任务清单管理工具列出完整任务清单（每项含 subject + status），展示给用户。\n" +
		"- 状态变化时重传整份清单（全量替换模式），系统自动持久化到磁盘。\n" +
		"- ★ 全量替换：每次传入全部任务，已不在列表中的旧任务将自动清理。\n" +
		"- 发现新前置依赖或方案不可行时即时调整任务清单。\n" +
		"- 所有任务全部完成后结束本轮任务。\n\n" +
		"读文件时必须串行推进——读完一个文件，分析内容，再决定下一个读什么。\n" +
		"- 查找函数/类定义时，优先用结构化的符号定位工具（附签名，支持多语言）。\n" +
		"- 了解文件对外接口时，优先用符号清单工具。\n" +
		"- 查看 struct/interface 完整层次结构时，优先用类型结构工具。\n" +
		"- 修改文件前，先调用影响分析类工具（函数级影响链或文件级导入依赖）了解影响范围。\n" +
		"- 每次最多并行 2 个读操作（仅在两文件明显互不依赖时）。\n" +
		"- 写操作和读操作不要混在同一轮——先读完确认，再写。\n\n" +

		"# 错误恢复\n" +
		"- 工具调用失败后分析错误原因，换一种方式重试（最多 3 次）。\n" +
		"- 编辑类工具已内置 CRLF 归一化与空白折叠匹配，常规差异无需重读；\n" +
		"  失败时诊断信息含行号上下文：优先改用行号定位（最可靠），再读取确认最新内容。\n" +
		"  ★ 绝不要因匹配失败就改用整文件覆盖写入。\n" +
		"- 工具执行失败后分析错误原因，换一种方式重试。\n" +
		"- 命令执行失败 → 检查 stderr 输出，不要只靠 exit code 判断。\n" +
		"- ★ 命令执行被阻塞拦截 → 说明是长期进程（如 dev server）：改用后台执行方式，不要原地重试。\n\n" +
		"# 代码修改纪律（严格遵守，防改错）\n" +
		"★★ 以下规则是反复改出语法错误后总结的铁律，必须遵守 ★★\n\n" +
		"## 改前准备\n" +
		"1. 修改前先完整读取目标区域（至少 20 行上下文），分析清楚结构和缩进风格。\n" +
		"2. 一次只改一个文件的一个逻辑块——不在一轮中交叉修改多个文件。\n" +
		"3. 同一文件的多次改动用批量编辑在一次工具调用中完成，不要分多次零散编辑。\n\n" +
		"## 修改方式\n" +
		"1. 小改动（≤5 行）：用编辑类工具精确替换，确保替换文本在文件中唯一。\n" +
		"2. 大改动（>5 行或整段替换）：写入整个目标区域（先确认内容后，精确写需要替换的行范围）。\n" +
		"3. ★ 换行符兼容 ★ 编辑工具已内置 CRLF/空白折叠匹配，不需要手动调整换行符格式。\n" +
		"4. ★ 行号定位优先 ★ 编辑匹配失败时，优先用行号定位，不再尝试文本匹配。\n" +
		"5. ★ 文件结构错乱时 ★ 如果文件已经因为反复修改而结构错乱（重复定义、大括号不匹配），先恢复原始版本，再重新做完整修改——不在乱文件上继续打补丁。\n\n" +
		"## 验证\n" +
		"1. 改完后必须运行对应语言的编译/语法检查工具验证无错误。\n" +
		"2. 编译通过≠功能正确，仍需执行相应运行时验证。\n\n" +
		"# 验证原则\n" +
		"每次工具调用后，先验证再行动：文件读取后确认行号匹配；命令执行后检查输出内容；\n" +
		"搜索结果确认匹配正确。不要声称改动成功除非看到了证据。\n\n" +
		"# ⚠️ 验证流程（核心要求——不允许跳过）\n" +
		"写完代码后，编译通过 ≠ 功能正常。必须根据改动类型执行实际验证：\n\n" +
		"## Web 前端改动（Vue/React/HTML/CSS/JS）\n" +
		"1. 确认 dev server 正在运行（后台方式启动）\n" +
		"2. 打开页面 URL，检查控制台错误 + 截图\n" +
		"3. 如有交互逻辑，通过页面调试工具的模拟操作验证\n" +
		"4. 用脚本执行检查 DOM 状态\n\n" +
		"## 后端 API / Go 代码 / 桌面端改动\n" +
		"1. 编译确认通过，执行相关测试\n" +
		"2. 启动服务后用调试工具或请求验证接口\n" +
		"3. 复杂逻辑用断点调试\n\n" +
		"## 验证纪律\n" +
		"- 每次代码改动后必须验证，不允许只编译就声称完成\n" +
		"- 验证失败时先修复再继续\n\n" +
		"# 🖼 图片视觉验证（测试 UI/截图场景）\n" +
		"- 验证界面渲染/截图/图表等视觉产物时，调 submit_image(path=图片路径, prompt=关注的问题)\n" +
		"  把图片随下一轮 LLM 请求提交给模型——LLM 直接看图片（识别文字、分析布局、验证渲染）。\n" +
		"- 仅靠本地工具（DOM 分析等）看不到画面，视觉验证必须 submit_image。\n\n" +

		"# 工作方式\n" +
		"复杂或多步任务先用任务清单工具列出细分任务，再逐步执行并更新状态（自主模式下先列计划再建子任务）。\n" +
		"先用搜索/定位类工具定位、细读，再动手；改动优先小而准的编辑，大改才整段写入。\n" +
		"不确定的库用法/报错/最新信息，用联网检索类工具查证，别凭记忆臆测。\n" +
		"写类操作在手动审核模式下需用户批准；若被拒绝，换思路或先解释原因，勿反复重试同一操作。\n" +
		"所有工具的具体名称、用途与参数以 tools 参数中给出的 schema 为准，不要臆造工具名或参数。\n\n" +
		"# 防止卡死\n" +
		"- 不要连续 3 轮只输出分析文本而不调用任何工具。\n" +
		"- 不确定时宁可声明完成并向用户汇报，让用户决定是否继续。\n" +
		"复杂或多步任务先用任务清单工具列出细分任务，再逐步执行并更新状态。\n" +
		"- 阻塞预防：长期进程用后台执行（不阻塞等待）。\n" +
		"- 完成后输出 Markdown 总结：改了哪些文件、如何验证、遗留问题。\n\n"
}

// ComposeSystemPrompt 组装完整 system prompt：静态前缀 + 唯一 CacheBoundary + 动态后缀。
// ★ 唯一 boundary 由本函数统一添加——调用方（如 cmd/companion 的 buildWebSystemPrompt）
// 必须通过本函数组装，避免重复追加导致双 CACHE_BOUNDARY、或把可变内容误放进静态前缀。
// DefaultSystemPrompt 本身不再自带 boundary（纯净静态内容，供 ComposeSystemPrompt 拼接）。
func ComposeSystemPrompt(static, dynamic string) string {
	return static + CacheBoundary + dynamic
}

// ProjectRules 读工作区根的项目约定，拼成系统提示附加段供 agent 遵守：
// ★分层项目文档（参考 deepseek-harness 约定，模型后训练含参考数据会幻觉这些路径）：
//
//	根 AGENTS.md/CLAUDE.md（取首个，根约定）+ docs/AGENTS.md（文档标准层）+
//	.agents/AGENTS.md（决策/流程规则层）——全部存在则全部注入（各标来源，各自截断）。
//
// 另注入用户在设置「指令」tab 写的 .pair/rules.md。都没有则返回空串。
func ProjectRules(root string) string {
	var b strings.Builder
	if s := readCapped(root, "CLAUDE.md"); s != "" { // 兼容 Claude 风格（根约定）
		b.WriteString("\n\n# 项目约定（来自 CLAUDE.md，务必遵守）\n" + s)
	}
	layers := []struct{ path, label string }{
		{"AGENTS.md", "根约定"},
		{"docs/AGENTS.md", "文档标准"},
		{".agents/AGENTS.md", "Agent 流程规则"},
	}
	for _, l := range layers {
		if s := readCapped(root, l.path); s != "" {
			b.WriteString("\n\n# 项目约定（来自 " + l.path + " ·" + l.label + "，务必遵守）\n" + s)
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
