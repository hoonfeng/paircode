package agent

// ═══════════════════════════════════════════════════════════════
// tool_budget.go — 段预算（双闸门：步数 + 工具调用）与分段执行/自动续跑
//
// 背景（2026-09）：单 Agent 多步执行长任务时，单段（一次 Run）内的模型调用
// 与工具调用可能上百次，工具结果持续累积 → 上下文膨胀。策略：
//
//   · 预算（★ 2026-09-12 双闸门）：每次 Run（= 一个执行段）受两道闸约束——
//     ① 步数预算（StepBudget，默认 120）：一步 = 一次 LLM 调用；段内步数
//     取 Loop.StepNo（每段 Run 由 openTurn 从 0 起算）；
//     ② 工具调用预算（ToolCallBudget，默认 120）：段内实际执行的工具调用数。
//     **任一达上限即结束本段**（不再发起新的 LLM 调用）并自动续跑下一段；
//     两者同时命中时按步数闸报告（SegmentState.Reason）。
//     ★ 配置化：两项预算与续跑段数上限均可配置——配置项由 agentloop 插件注册
//     （设置面板「Agent」组：stepBudget / toolCallBudget / maxToolBudgetSegments，
//     存 settings.json 的 pluginSettings.agentloop），装配时经
//     ctx.loopFactory.register 的 overrides 透传到 Loop 字段；
//     DefaultStepBudget / DefaultToolCallBudget / MaxToolBudgetSegments 仅作为
//     未配置（0/缺省）时的默认值。
//   · 续跑：SessionManager 在 Run 返回后检查 TakeSegmentContinue()，
//     自动以 continuation 消息发起下一段——同一会话、历史保留。
//     ★ 2026-09：段内**只追加、不中途精简**（改写已发送字节会断前缀缓存）；
//     注入体积由「下一段起点」的跨段精简控制（CondenseHistoryByPressure），
//     因此分段续跑本身就是上下文膨胀的主闸。
//   · 计数口径：步数按「每轮 LLM 调用」+1（含并列多个 tool_call 的轮次只算 1 步）；
//     工具调用按「模型发起的实际工具执行」计数（一次 LLM 返回多个 tool_call 时
//     逐个计数）。审批驳回 / 截断未执行的不计数（未消耗预算）；执行失败仍计数
//     （已实际执行）。
//   · 上限：MaxToolBudgetSegments 段（防失控）；超过后停止自动续跑，
//     用户再发消息即可继续。
//
// 实现落点：
//   · 步数计数：Loop.StepNo（agentloop.go beginStep；JS 循环每轮调
//     loop.ctrl.beginStep，Go 回退在 loop.go 每轮调 beginStep）；
//   · 工具调用计数：jsloop_runner.go（JS 循环 tools.run）、loop.go（Go 回退
//     串行）执行后 noteToolCall；
//   · 段结束检查：agentloop 插件（JS）经 loop.ctrl.toolBudget() 在 step
//     收尾检查；Go 回退路径在 loop.go step 收尾检查——均设置
//     segmentPending 并 emit done(doneReason=tool_budget)；
//   · JS 循环经返回值 {segment:…} 告知宿主（jsloop_run.go 解析）。
// ═══════════════════════════════════════════════════════════════

import "fmt"

const (
	// DefaultToolCallBudget 默认单段工具调用预算（Loop.ToolCallBudget 零值 = 用默认；负数 = 不限）。
	DefaultToolCallBudget = 120
	// DefaultStepBudget 默认单段步数预算（Loop.StepBudget 零值 = 用默认；负数 = 不限）。
	//   一步 = 一次 LLM 调用；一段内的步数取 Loop.StepNo（每段 Run 从 0 起算）。
	DefaultStepBudget = 120
	// MaxToolBudgetSegments 单轮任务最大自动续跑段数（防失控；超过后停止续跑，用户可再发消息继续）。
	MaxToolBudgetSegments = 20
	// MaxToolCallBudgetLimit 单段工具调用预算的钳制上限（配置误填超大值时钳制，防单段失控）。
	MaxToolCallBudgetLimit = 10000
	// MaxStepBudgetLimit 单段步数预算的钳制上限（口径同工具预算）。
	MaxStepBudgetLimit = 10000
	// MaxToolBudgetSegmentsLimit 自动续跑段数上限的钳制上限。
	MaxToolBudgetSegmentsLimit = 1000
	// LoopIterationSlack 段内迭代安全上限相对段预算的余量：
	//   双闸门中「步数预算」本身就是按 LLM 轮次计数，「工具调用预算」按实际执行
	//   计数（一轮可能 0 次或多于 1 次）——余量保证正常流程由预算（而非迭代
	//   上限）成为段结束的真正闸门。
	LoopIterationSlack = 5
)

// 段结束原因（SegmentState.Reason / {segment}.reason）取值。
const (
	// SegmentReasonToolBudget 工具调用预算用尽（段内实际执行次数达上限）。
	SegmentReasonToolBudget = "tool_budget"
	// SegmentReasonStepBudget 步数预算用尽（段内 LLM 调用轮数达上限）。
	SegmentReasonStepBudget = "step_budget"
)

// SegmentState 段预算状态（双闸门快照：步数 + 工具调用）。
//   - UsedSteps = 本段已走过的步数（Loop.StepNo，每段 Run 由 openTurn 归零）；
//   - 预算字段为**生效值**：0 = 不限（负数配置归一为 0）；
//   - Exhausted=true 时 Reason 标明命中的闸门（两者同时命中报 step_budget）。
type SegmentState struct {
	UsedTools  int    `json:"usedTools"`
	ToolBudget int    `json:"toolBudget"`
	UsedSteps  int    `json:"usedSteps"`
	StepBudget int    `json:"stepBudget"`
	Exhausted  bool   `json:"exhausted"`
	Reason     string `json:"reason,omitempty"`
}

// ToolCallBudgetOrDefault 返回生效的单段工具调用预算（0=默认 120；负数=不限返回 0）。
func (l *Loop) ToolCallBudgetOrDefault() int {
	v := NormalizeToolCallBudget(l.ToolCallBudget)
	if v < 0 {
		return 0 // 负数 = 不限（内部以 0 表示「不限」，见 segmentBudgetState）
	}
	return v
}

// StepBudgetOrDefault 返回生效的单段步数预算（0=默认 120；负数=不限返回 0）。
func (l *Loop) StepBudgetOrDefault() int {
	v := NormalizeStepBudget(l.StepBudget)
	if v < 0 {
		return 0 // 负数 = 不限（内部以 0 表示「不限」，见 segmentBudgetState）
	}
	return v
}

// IterationLimit 段内迭代安全上限（Loop 内部派生，**不是配置项**）：
//
//	= max(生效工具预算, 生效步数预算) + LoopIterationSlack；两者均不限（0）时以
//	  MaxToolCallBudgetLimit 兜底。始终为有限值。
//
// ★ 2026-09-12：「最大迭代数」配置项已移除（原 core.AppSettings.MaxIterations /
// 插件 maxIterations）——段的结束由双闸门预算 + 自动续跑（受 MaxToolBudgetSegments
// 约束）负责，取消 / 绕圈检测 / 自然终止仍各自生效。本上限只是防失控的最后一道
// 保险（有限值，不会死循环）：正常情况下预算先耗尽并触发分段续跑，本上限不会命中。
func (l *Loop) IterationLimit() int {
	budget := l.ToolCallBudgetOrDefault()
	if sb := l.StepBudgetOrDefault(); sb > budget {
		budget = sb
	}
	if budget <= 0 { // 双闸门均不限 → 以预算钳制上限兜底（仍是有限值）
		budget = MaxToolCallBudgetLimit
	}
	return budget + LoopIterationSlack
}

// NormalizeToolCallBudget 把配置值归一化为 Loop.ToolCallBudget 语义：
// 0（/缺省）→ DefaultToolCallBudget（默认 120）；负数 → 原样（负数 = 不限）；
// 超上限 → 钳制到 MaxToolCallBudgetLimit（防配置误填超大值）。
//
// 来源：agentloop 插件注册的 toolCallBudget（pluginSettings.agentloop），
// 经 ctx.loopFactory.register 的 overrides 透传为 Loop.ToolCallBudget。
func NormalizeToolCallBudget(v int) int {
	switch {
	case v == 0:
		return DefaultToolCallBudget
	case v < 0:
		return v
	case v > MaxToolCallBudgetLimit:
		return MaxToolCallBudgetLimit
	default:
		return v
	}
}

// NormalizeStepBudget 把配置值归一化为 Loop.StepBudget 语义（口径同工具预算）：
// 0（/缺省）→ DefaultStepBudget（默认 120）；负数 → 原样（负数 = 不限）；
// 超上限 → 钳制到 MaxStepBudgetLimit。
//
// 来源：agentloop 插件注册的 stepBudget（pluginSettings.agentloop），
// 经 ctx.loopFactory.register 的 overrides 透传为 Loop.StepBudget。
func NormalizeStepBudget(v int) int {
	switch {
	case v == 0:
		return DefaultStepBudget
	case v < 0:
		return v
	case v > MaxStepBudgetLimit:
		return MaxStepBudgetLimit
	default:
		return v
	}
}

// NormalizeMaxToolBudgetSegments 把配置值归一化为「单轮最大自动续跑段数」：
// 0/缺省/负数 → MaxToolBudgetSegments（默认 20；此上限是防失控闸，不提供「不限」）；
// 超上限 → 钳制到 MaxToolBudgetSegmentsLimit。
//
// 来源：agentloop 插件注册的 maxToolBudgetSegments（pluginSettings.agentloop），
// 经 ctx.loopFactory.register 的 overrides 透传为 Loop.MaxToolBudgetSegments。
func NormalizeMaxToolBudgetSegments(v int) int {
	switch {
	case v <= 0:
		return MaxToolBudgetSegments
	case v > MaxToolBudgetSegmentsLimit:
		return MaxToolBudgetSegmentsLimit
	default:
		return v
	}
}

// MaxToolBudgetSegmentsOrDefault 返回生效的单轮最大自动续跑段数（0/缺省 = 默认 20）。
func (l *Loop) MaxToolBudgetSegmentsOrDefault() int {
	return NormalizeMaxToolBudgetSegments(l.MaxToolBudgetSegments)
}

// noteToolCall 记录一次实际执行的工具调用（一次 LLM 返回多个调用时逐个计）。
func (l *Loop) noteToolCall() { l.toolCallsRun++ }

// segmentBudgetState 返回本段的双闸门预算状态（步数 + 工具调用）。
//   - 步数：Loop.StepNo（每段 Run 由 openTurn 归零，beginStep 每轮 +1）；
//   - 工具调用：toolCallsRun（noteToolCall 每次实际执行 +1）；
//   - 耗尽判定：任一达生效上限即 Exhausted=true；两者同时命中报 step_budget
//     （步数是「一轮一次」的粗粒度闸，工具调用是细粒度闸，前者更能代表段边界）。
//   - 预算 ≤0 = 不限，该闸门永不耗尽。
func (l *Loop) segmentBudgetState() SegmentState {
	st := SegmentState{
		UsedTools:  l.toolCallsRun,
		ToolBudget: l.ToolCallBudgetOrDefault(),
		UsedSteps:  l.StepNo,
		StepBudget: l.StepBudgetOrDefault(),
	}
	switch {
	case st.StepBudget > 0 && st.UsedSteps >= st.StepBudget:
		st.Exhausted, st.Reason = true, SegmentReasonStepBudget
	case st.ToolBudget > 0 && st.UsedTools >= st.ToolBudget:
		st.Exhausted, st.Reason = true, SegmentReasonToolBudget
	}
	return st
}

// SegmentBudgetState 对外暴露双闸门预算状态（测试/诊断用）。
func (l *Loop) SegmentBudgetState() SegmentState { return l.segmentBudgetState() }

// toolBudgetState 返回（已用工具调用、生效工具预算、是否耗尽）——**工具闸门视图**
// （保留供诊断与既有单测；完整双闸门状态见 segmentBudgetState）。
// exhausted 只反映工具调用闸门自身，不因步数闸门而置真（保持旧语义）。
func (l *Loop) toolBudgetState() (used, budget int, exhausted bool) {
	st := l.segmentBudgetState()
	return st.UsedTools, st.ToolBudget, st.ToolBudget > 0 && st.UsedTools >= st.ToolBudget
}

// ToolBudgetState 对外暴露工具闸门预算状态（测试/诊断用）。
func (l *Loop) ToolBudgetState() (used, budget int, exhausted bool) { return l.toolBudgetState() }

// markSegmentPending 标记本 Run 因段预算（步数或工具调用）分段结束（待 SessionManager 续跑）。
func (l *Loop) markSegmentPending() { l.segmentPending = true }

// TakeSegmentContinue 消费「分段续跑」标记：ok=true 表示本 Run 因段预算结束、
// 应自动开启下一段；返回段结束时的双闸门状态（用于续跑消息、日志与通知）。
func (l *Loop) TakeSegmentContinue() (SegmentState, bool) {
	if !l.segmentPending {
		return SegmentState{}, false
	}
	l.segmentPending = false
	return l.segmentBudgetState(), true
}

// resetToolCallsRun 段计数清零（Run 开始时调用——每次 Run 即一个执行段）。
// 步数计数（StepNo）由 openTurn 在每段 Run 开始时归零。
func (l *Loop) resetToolCallsRun() { l.toolCallsRun = 0 }

// budgetText 预算文本（0/负数 = 不限）。
func budgetText(b int) string {
	if b <= 0 {
		return "不限"
	}
	return fmt.Sprintf("%d", b)
}

// segmentEndNotice 段结束通知文案（JS 循环与 Go 回退共用同一措辞，防两边漂移）。
func segmentEndNotice(st SegmentState) string {
	why := "执行预算"
	switch st.Reason {
	case SegmentReasonStepBudget:
		why = "步数预算"
	case SegmentReasonToolBudget:
		why = "工具调用预算"
	}
	return fmt.Sprintf("本段已达%s上限（已执行 %d 步 / 上限 %s 步；工具调用 %d 次 / 上限 %s 次），分段结束，自动开启下一段继续",
		why, st.UsedSteps, budgetText(st.StepBudget), st.UsedTools, budgetText(st.ToolBudget))
}

// SegmentContinueMessage 分段续跑消息（注入为下一段的用户消息）。
func SegmentContinueMessage(st SegmentState) string {
	why := "执行预算用尽"
	switch st.Reason {
	case SegmentReasonStepBudget:
		why = "步数预算用尽"
	case SegmentReasonToolBudget:
		why = "工具调用预算用尽"
	}
	return fmt.Sprintf("上一执行段因%s而结束（本段已执行 %d 步 / 上限 %s 步；工具调用 %d 次 / 上限 %s 次），任务尚未完成。请接着上一段的进度继续：先依据上一步的工具结果与当前状态判断还差什么，然后继续推进（不要重头开始、不要重复已完成的步骤）；若任务已完成，直接给出最终结论。",
		why, st.UsedSteps, budgetText(st.StepBudget), st.UsedTools, budgetText(st.ToolBudget))
}
