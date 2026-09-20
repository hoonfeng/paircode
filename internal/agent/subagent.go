// subagent.go — 「子 agent 回合」内核能力（一切皆插件：能力 Go / 策略 JS）
//
// 场景：插件需要在一个会话里派生一个**独立 agent 回合**——独立系统提示与起始任务、
// 可裁剪的工具面、自己的历史（不污染工作会话历史）、事件带来源标注（Event.AgentName）
// 实时下发前端（看板分区），并把轨迹与结论交回插件。
//
// 首要用例 = 自主模式（autopilot 插件的「监督者」）：工作 agent 自然结束（无 tool_call +
// 有正文）→ 宿主在会话续轮处调用插件注册的决策器（见 jsplugin_autopilot.go）→ 插件在
// 决策器里经本能力派生一个监督者回合（可调全部工具，自行核查证据）→ 返回裁决
// （完成 / 继续 + 下一步指令）→ 宿主据此唤醒工作 agent 继续或收尾整轮。
//
// 设计要点（勿回退）：
//  1. ★ 子 agent 强制走 **Go 循环**（Loop.forceGoLoop）：决策器回调运行在插件 JS 栈上，
//     若子 agent 再走 agentloop 的 JS 循环实现，会在同一 VM 上自锁。策略仍全部由插件
//     提供（角色提示词 / 任务书 / 裁决语义），内核只提供「跑一个回合」的能力。
//  2. ★ 子 agent 历史**不落工作会话消息文件**（无 OnBatchPersist）：它只把自己的事件
//     下发给前端（带 AgentName）；回合记录由插件按自己的策略落盘（可溯源）。
//  3. ★ 会话运行期绑定 SubagentRuntime（SessionManager.Start 内注册、结束解除）；
//     决策器调用期间内核额外设「当前 runtime」——插件未指定 spec.ConvID 时按当前会话解析。
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// ── 规格与结果 ────────────────────────────────────────────────

// SubagentResultTool 子 agent 的「结构化提交」工具规格。
// 内核用 Go handler 注册它（收集参数 + 结束本回合），插件提供名字/描述/schema（策略）。
type SubagentResultTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// SubagentSpec 派生一个子 agent 回合的参数（由插件提供；字段含义见各注释）。
type SubagentSpec struct {
	// Name 来源标注：该回合的所有事件带 AgentName=Name（前端看板分区/溯源用）。
	Name string `json:"name"`
	// Round 回合序号（内核只透传标注，语义由插件定义，如「第 N 次监督」）。
	Round int `json:"round,omitempty"`
	// System 系统提示（角色设定，策略由插件提供）。
	System string `json:"system,omitempty"`
	// Task 起始任务（插件构造的任务书）。
	Task string `json:"task"`
	// Tools 工具白名单（空 = 继承会话全部已启用工具）；名单外的工具不可见/不可调用。
	Tools []string `json:"tools,omitempty"`
	// ExtraTools 白名单之外额外保留的工具名（如插件自带工具）。
	ExtraTools []string `json:"extraTools,omitempty"`
	// StepBudget 段内步数预算（0 = 内核默认）。
	StepBudget int `json:"stepBudget,omitempty"`
	// ToolCallBudget 段内工具调用预算（0 = 内核默认）。
	ToolCallBudget int `json:"toolCallBudget,omitempty"`
	// MaxContextTokens 上下文窗口（0 = 继承会话配置）。
	MaxContextTokens int `json:"maxContextTokens,omitempty"`
	// TimeoutMs 本回合超时（毫秒；0 = 不限；始终受会话取消约束）。
	TimeoutMs int `json:"timeoutMs,omitempty"`
	// ResultTool 可选：结构化提交工具（调用即记录参数并结束本回合）。
	ResultTool *SubagentResultTool `json:"resultTool,omitempty"`
	// ConvID 目标会话（空 = 当前决策器绑定的会话）。
	ConvID string `json:"convId,omitempty"`
}

// SubagentResult 一个子 agent 回合的结果（返回插件）。
type SubagentResult struct {
	// Content 本回合正文（所有 content 段拼接；末段即最终答复）。
	Content string `json:"content"`
	// Segments 轨迹（thinking/content/tool_call；与前端展示结构同构，供插件落盘溯源）。
	Segments []Segment `json:"segments"`
	// Submitted 结构化提交工具收到的参数（未调用则为 nil）。
	Submitted map[string]any `json:"submitted,omitempty"`
	// SubmittedRaw 结构化提交的原始 JSON 文本（便于诊断/落盘）。
	SubmittedRaw string `json:"submittedRaw,omitempty"`
	// Steps / ToolCalls 本回合步数与工具调用次数。
	Steps     int `json:"steps"`
	ToolCalls int `json:"toolCalls"`
	// PromptTokens / OutputTokens 本回合 token 用量（累计）。
	PromptTokens int `json:"promptTokens"`
	OutputTokens int `json:"outputTokens"`
	// DurationMs 本回合墙钟耗时。
	DurationMs int64 `json:"durationMs"`
	// Error 本回合错误（为空 = 正常结束）。
	Error string `json:"error,omitempty"`
	// Ended 结束方式：completed（自然结束）/ submitted（结构化提交）/ budget（预算耗尽）/ error。
	Ended string `json:"ended"`
}

// ── 会话运行环境（子 agent 的父环境）────────────────────────────

// SubagentRuntime 会话运行环境快照：子 agent 复用宿主会话的模型/工具面/事件通道/审核策略。
// 由 SessionManager.Start 注册（会话运行期有效），会话结束解除。
type SubagentRuntime struct {
	ConvID        string
	WorkspaceRoot string
	Provider      Provider
	Registry      *Registry
	System        string
	// Events 会话事件通道（子 agent 事件经此下发给前端；带 AgentName）。
	Events chan<- Event
	// Approve 审批钩子（复用会话策略：manual 挂起 / AI 审核 / off 放行）。
	Approve         func(ctx context.Context, tc ToolCall) (bool, string)
	ReviewMode      string
	ReviewProvider  Provider
	ReviewBlacklist []string
	ReviewWhitelist []string
	// Compressor / MaxContextTokens 上下文精简（复用会话配置；0 = 不精简）。
	Compressor       Provider
	MaxContextTokens int
	// StepBudget / ToolCallBudget 默认预算（spec 未指定时使用）。
	StepBudget     int
	ToolCallBudget int
	// ParentCtx 父上下文（取消即子 agent 一起停；通常为会话 runCtx）。
	ParentCtx context.Context
}

var (
	subagentRtMu   sync.RWMutex
	subagentRtByID = map[string]*SubagentRuntime{}
	// subagentRtCur 当前决策器绑定的会话环境（SetCurrentSubagentRuntime 设置；
	// 插件未指定 spec.ConvID 时按它解析）。
	subagentRtCur *SubagentRuntime
)

// RegisterSubagentRuntime 绑定会话运行环境，返回解除函数（会话结束时调用）。
func RegisterSubagentRuntime(rt *SubagentRuntime) func() {
	if rt == nil || rt.ConvID == "" {
		return func() {}
	}
	subagentRtMu.Lock()
	subagentRtByID[rt.ConvID] = rt
	subagentRtMu.Unlock()
	return func() {
		subagentRtMu.Lock()
		if cur, ok := subagentRtByID[rt.ConvID]; ok && cur == rt {
			delete(subagentRtByID, rt.ConvID)
		}
		subagentRtMu.Unlock()
	}
}

// SetCurrentSubagentRuntime 设置「当前会话环境」（决策器调用前后成对使用），返回还原函数。
func SetCurrentSubagentRuntime(rt *SubagentRuntime) func() {
	subagentRtMu.Lock()
	prev := subagentRtCur
	subagentRtCur = rt
	subagentRtMu.Unlock()
	return func() {
		subagentRtMu.Lock()
		subagentRtCur = prev
		subagentRtMu.Unlock()
	}
}

// SubagentRuntimeFor 取指定会话的运行环境（无则 nil）。
func SubagentRuntimeFor(convID string) *SubagentRuntime {
	subagentRtMu.RLock()
	defer subagentRtMu.RUnlock()
	if convID == "" {
		return subagentRtCur
	}
	if rt, ok := subagentRtByID[convID]; ok {
		return rt
	}
	return nil
}

// ── 回合执行 ──────────────────────────────────────────────────

// ErrSubagentUnavailable 当前无可用会话环境（无运行中的会话 / 插件在错误时机调用）。
var ErrSubagentUnavailable = fmt.Errorf("子 agent 能力不可用：当前没有绑定的会话运行环境")

// RunSubagent 派生并执行一个子 agent 回合（阻塞至结束）。
// ctx 取消即中止；返回结果含轨迹/用量/结构化提交（err 仅表示能力级失败）。
func RunSubagent(ctx context.Context, spec SubagentSpec) (*SubagentResult, error) {
	rt := SubagentRuntimeFor(spec.ConvID)
	if rt == nil {
		return nil, ErrSubagentUnavailable
	}
	// ★ 诊断：回合开始（宿主 Go 侧进入点）——与「回合执行开始」「回合结束」配对，
	//   定位卡点在装配阶段还是执行阶段。
	log.Printf("[subagent] 回合开始 name=%s conv=%s round=%d tools=%d stepBudget=%d toolBudget=%d timeoutMs=%d",
		spec.Name, rt.ConvID, spec.Round, len(spec.Tools), spec.StepBudget, spec.ToolCallBudget, spec.TimeoutMs)
	if ctx == nil {
		ctx = rt.ParentCtx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if spec.TimeoutMs > 0 {
		var cancelTimeout context.CancelFunc
		ctx, cancelTimeout = context.WithTimeout(ctx, time.Duration(spec.TimeoutMs)*time.Millisecond)
		defer cancelTimeout()
	}
	if strings.TrimSpace(spec.Name) == "" {
		spec.Name = "subagent"
	}
	if spec.Round <= 0 {
		spec.Round = 1
	}

	// ── 工具面：白名单裁剪（空 = 继承全部；ResultTool 由内核注册在子表）──
	var reg *Registry
	if rt.Registry == nil {
		return nil, fmt.Errorf("子 agent 能力不可用：会话未提供工具注册表")
	}
	if len(spec.Tools) == 0 {
		reg = rt.Registry.Copy()
	} else {
		names := make([]string, 0, len(spec.Tools)+len(spec.ExtraTools)+1)
		names = append(names, spec.Tools...)
		names = append(names, spec.ExtraTools...)
		reg = rt.Registry.Subset(names)
		if reg == nil || len(reg.Names()) == 0 {
			return nil, fmt.Errorf("子 agent 工具白名单为空（spec.Tools=%v 未命中任何已注册工具）", spec.Tools)
		}
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// ── 结构化提交工具：记录参数 + 结束本回合（插件提供 schema，内核提供 handler）──
	box := &subagentSubmitBox{}
	if spec.ResultTool != nil && strings.TrimSpace(spec.ResultTool.Name) != "" {
		desc := spec.ResultTool.Description
		if strings.TrimSpace(desc) == "" {
			desc = "提交本回合的结构化结论（调用后本回合结束）。"
		}
		params := spec.ResultTool.Parameters
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		reg.Register(&Tool{
			Name:        spec.ResultTool.Name,
			Description: desc,
			Parameters:  params,
			Handler: func(_ context.Context, args map[string]any) (string, error) {
				box.set(args)
				cancel() // 提交即结束本回合（循环在下一步前退出）
				return "已提交，本回合结束。", nil
			},
		})
	}

	// ── 装配子 Loop（Go 循环；不落会话历史）──
	stepBudget := spec.StepBudget
	if stepBudget == 0 {
		stepBudget = rt.StepBudget
	}
	toolBudget := spec.ToolCallBudget
	if toolBudget == 0 {
		toolBudget = rt.ToolCallBudget
	}
	maxCtx := spec.MaxContextTokens
	if maxCtx == 0 {
		maxCtx = rt.MaxContextTokens
	}
	sys := spec.System
	if strings.TrimSpace(sys) == "" {
		sys = rt.System
	}
	handle, err := CreateLoop(LoopOpts{
		Provider:         rt.Provider,
		Registry:         reg,
		System:           sys,
		StepBudget:       stepBudget,
		ToolCallBudget:   toolBudget,
		MaxContextTokens: maxCtx,
		Compressor:       rt.Compressor,
		WorkspaceRoot:    rt.WorkspaceRoot,
		ConvID:           rt.ConvID,
		ReviewMode:       rt.ReviewMode,
		ReviewProvider:   rt.ReviewProvider,
		ReviewBlacklist:  rt.ReviewBlacklist,
		ReviewWhitelist:  rt.ReviewWhitelist,
		Autonomous:       false, // ★ 子 agent 不触发自主监督（防递归）
		// ★ 跳过插件装配器链：本回合在插件 JS 回调栈内创建（autopilot 决策器 decide 内），
		//   装配器要取所属插件 VM 锁——正是当前持锁插件时会自死锁（goja 锁非重入，实测
		//   decide → ctx.subagent.run → CreateLoop → autopilot 装配器 → 永久卡死）。
		//   子 agent 装配参数已由本处显式给出，无需插件装配。
		SkipPluginAssembly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("子 agent 装配失败: %w", err)
	}
	l := handle.Loop()
	// ★ 强制 Go 循环：决策器回调在插件 JS 栈上，不能再进入 agentloop 的 JS 循环实现（自锁）。
	l.forceGoLoop = true
	l.AgentName = spec.Name
	log.Printf("[subagent] 循环已装配 name=%s conv=%s（强制 Go 循环，不走插件 JS 循环）", spec.Name, rt.ConvID)
	// 回合序号 → 事件的 turn：前端按 (AgentName, turn) 分区渲染看板（同一回合的多步事件同 turn）。
	l.TurnNo = spec.Round - 1
	l.Approve = rt.Approve

	tr := &subagentTranscript{}
	l.OnEvent = func(e Event) {
		tr.add(e)
		// 实时下发前端（非阻塞：通道满即丢，不拖慢子 agent）
		if rt.Events != nil {
			select {
			case rt.Events <- e:
			default:
			}
		}
	}

	start := time.Now()
	var runErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				runErr = fmt.Errorf("子 agent 回合 panic: %v", r)
				log.Printf("[subagent] 回合 panic name=%s conv=%s: %v", spec.Name, rt.ConvID, r)
			}
		}()
		log.Printf("[subagent] 回合执行开始 name=%s conv=%s", spec.Name, rt.ConvID)
		_, runErr = handle.Run(runCtx, spec.Task, nil)
	}()
	duration := time.Since(start)
	log.Printf("[subagent] 回合结束 name=%s conv=%s 耗时=%s err=%v", spec.Name, rt.ConvID, duration.Round(time.Millisecond), runErr)

	res := tr.result(box, duration, l.StepNo)
	if runErr != nil {
		// 结构化提交会主动取消本回合（预期的结束路径，不算错误）
		if box.get() != nil && (runErr == context.Canceled || strings.Contains(runErr.Error(), "context canceled")) {
			res.Ended = "submitted"
		} else {
			res.Error = runErr.Error()
			if res.Ended == "" {
				res.Ended = "error"
			}
		}
	}
	return res, nil
}

// subagentSubmitBox 结构化提交结果的并发安全容器。
type subagentSubmitBox struct {
	mu   sync.Mutex
	args map[string]any
	raw  string
}

func (b *subagentSubmitBox) set(args map[string]any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.args = args
	b.raw = marshalCompact(args)
}

func (b *subagentSubmitBox) get() map[string]any {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.args
}

func (b *subagentSubmitBox) rawText() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.raw
}

// subagentTranscript 子 agent 轨迹收集器（事件 → 展示用 Segment）。
type subagentTranscript struct {
	mu           sync.Mutex
	segments     []Segment
	content      strings.Builder
	toolCalls    int
	promptTokens int
	outputTokens int
}

func (t *subagentTranscript) add(e Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	switch e.Type {
	case EventThinking:
		t.segments = appendSegmentText(t.segments, "thinking", e.Content)
	case EventContent:
		t.content.WriteString(e.Content)
		t.segments = appendSegmentText(t.segments, "content", e.Content)
	case EventToolCall:
		t.toolCalls++
		t.segments = append(t.segments, Segment{
			Type: "tool_call", Name: e.Tool, ArgsRaw: e.Args, CallID: e.CallID,
		})
	case EventToolResult:
		for i := len(t.segments) - 1; i >= 0; i-- {
			if t.segments[i].Type == "tool_call" && t.segments[i].CallID == e.CallID {
				t.segments[i].Result = e.Content
				break
			}
		}
	case EventUsage:
		if e.Usage != nil {
			t.promptTokens += e.Usage.PromptTokens
			t.outputTokens += e.Usage.CompletionTokens
		}
	}
}

func (t *subagentTranscript) result(box *subagentSubmitBox, d time.Duration, steps int) *SubagentResult {
	t.mu.Lock()
	defer t.mu.Unlock()
	res := &SubagentResult{
		Content:      t.content.String(),
		Segments:     append([]Segment(nil), t.segments...),
		Steps:        steps,
		ToolCalls:    t.toolCalls,
		PromptTokens: t.promptTokens,
		OutputTokens: t.outputTokens,
		DurationMs:   d.Milliseconds(),
	}
	if submitted := box.get(); submitted != nil {
		res.Submitted = submitted
		res.SubmittedRaw = box.rawText()
		res.Ended = "submitted"
	} else if res.Ended == "" {
		res.Ended = "completed"
	}
	return res
}

// appendSegmentText 追加同类文本段（末尾同类则拼接，保持事件流的自然分段）。
func appendSegmentText(segs []Segment, typ, text string) []Segment {
	if text == "" {
		return segs
	}
	if n := len(segs); n > 0 && segs[n-1].Type == typ {
		segs[n-1].Content += text
		return segs
	}
	return append(segs, Segment{Type: typ, Content: text})
}

// marshalCompact 紧凑 JSON（失败返回空串；仅诊断/落盘用）。
func marshalCompact(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
