// agentloop —— 双层循环（turn/step）状态模型。
//
// 参照业界 Agent 循环通用设计（ReactLoop 风格）的
// turn/step 双层循环设计移植到 Go 侧：
//
//	一次 Run 调用 = 一个 turn（turn/start → turn/end）
//	每次 LLM 调用 + 工具执行 = 一个 step（step/start → step/end）
//
// 与业界通用模型的对应关系：
//   - TurnEndReason      ↔ TurnEndReason（completed / max-tokens / aborted / error / blocked）
//   - AgentCancelCause   ↔ AgentCancelCause（user / parent / hook / disposed）
//   - Loop.PreStep       ↔ agent/pre-step 瀑布（可改写进入模型的输入或拒绝整个 turn）
//   - Loop.hadMaxTokens  ↔ max-tokens sticky（一旦某 step 触顶，后续正常 step 不降级结果）
//   - steer/followUp 双队列 ↔ Inbox 的 next-step / next-turn 双列表
//
// 外部接口（Run/OnEvent/Approve/OnBatchPersist…）与消息模型保持兼容，本模块只
// 增强循环的边界语义，不改动持久化格式与前端事件协议。
package agent

import "fmt"

// TurnEndReason 一轮 turn（一次 Run 调用）的结束原因，结构化枚举。
// 对应 turn 结束原因：completed / max-tokens / aborted / error / blocked。
// 扩展了 gou-ide 特有的 content-loop（内容循环兜底）与 max-iterations（迭代上限）。
type TurnEndReason string

const (
	// TurnCompleted 自然终止：模型输出正文且无工具调用（任务完成）。
	TurnCompleted TurnEndReason = "completed"
	// TurnMaxTokens 输出被 token 上限截断（stop_reason=length）。
	// sticky：turn 内任一 step 触顶后，后续正常完成的 step 不得把结果降级为 completed。
	TurnMaxTokens TurnEndReason = "max-tokens"
	// TurnAborted 被外部取消（ctx 取消 / cancel 调用）。
	TurnAborted TurnEndReason = "aborted"
	// TurnError 内部错误（LLM 调用失败 / 工具致命错误）。
	TurnError TurnEndReason = "error"
	// TurnBlocked 被 pre-step 拦截拒绝（agent/pre-step → reject），turn 未消费任何模型请求。
	TurnBlocked TurnEndReason = "blocked"
	// TurnContentLoop 连续多轮只输出文字不调工具，内容循环兜底结束。
	TurnContentLoop TurnEndReason = "content-loop"
	// TurnMaxIterations 达到最大迭代数仍未完成。
	TurnMaxIterations TurnEndReason = "max-iterations"
)

// AgentCancelCause 一次 turn 被取消的原因。
// 对应 agent 取消原因：user / parent / hook / disposed。
type AgentCancelCause struct {
	// Kind 取消来源：user（用户停止）/ parent（父 agent 中止）/ hook（钩子拦截）/
	// disposed（agent 被销毁）/ context（Go context 取消，gou-ide 特有）。
	Kind string `json:"kind"`
	// Reason 补充说明（可为空）。
	Reason string `json:"reason,omitempty"`
}

// 取消来源常量（AgentCancelCause.Kind 取值）。
const (
	CancelByUser    = "user"     // 用户主动停止
	CancelByParent  = "parent"   // 父 agent 中止子 agent
	CancelByHook    = "hook"     // 钩子/审核拦截
	CancelByDispose = "disposed" // agent 生命周期销毁
	CancelByContext = "context"  // Go context 取消（gou-ide 特有映射）
	CancelByPlugin  = "plugin"   // 插件请求停止（ctx.get('loop').requestStop()，一切皆插件）
)

// openTurn 打开一轮新 turn（一次 Run 调用）。重置 step 计数与 sticky 状态。
// 对应 turn/start：每次 Run 前由主循环调用一次。
// ★ t4 L2：同步全局钩子轮次（loopHookCurrentTurn，供 hook payload 透传）。
func (l *Loop) openTurn() {
	l.TurnNo++
	l.StepNo = 0
	l.hadMaxTokens = false
	l.CancelCause = AgentCancelCause{}
	loopHookCurrentTurn.Store(int64(l.TurnNo))
	l.emit(Event{Type: EventNotice, Content: fmt.Sprintf("[turn/%d/start] 开始第 %d 轮对话", l.TurnNo, l.TurnNo)})
}

// beginStep 进入本轮 turn 的下一个 step（一次 LLM 调用 + 工具执行）。
// 对应 step/start。
func (l *Loop) beginStep() {
	l.StepNo++
	l.emit(Event{Type: EventNotice, Content: fmt.Sprintf("[step/%d.%d/start] 模型调用 #%d", l.TurnNo, l.StepNo, l.StepNo)})
}

// endStep 收尾一个 step：记录本次 step 的工具执行结果概要。
// 对应 step/end。每轮迭代工具执行完成后调用一次。
func (l *Loop) endStep(summary string) {
	if summary != "" {
		l.emit(Event{Type: EventNotice, Content: fmt.Sprintf("[step/%d.%d/end] %s", l.TurnNo, l.StepNo, summary)})
	} else {
		l.emit(Event{Type: EventNotice, Content: fmt.Sprintf("[step/%d.%d/end]", l.TurnNo, l.StepNo)})
	}
}

// endTurn 收尾一轮 turn：记录结构化结束原因（无显式设置时按 err/ctx 推断）。
// 对应 turn/end 事件。defer 中调用一次即可。
// ★ t4 L2：轮次结束后清零全局钩子轮次（钩子不再透传旧轮次）。
func (l *Loop) endTurn(err error, ctxDone bool) {
	loopHookCurrentTurn.Store(0)
	if l.LastTurnReason == "" {
		switch {
		case ctxDone:
			l.LastTurnReason = TurnAborted
		case err != nil:
			l.LastTurnReason = TurnError
		default:
			l.LastTurnReason = TurnCompleted
		}
		// 推断路径（未显式设置原因，即无 EventDone 发出）：补发 turn/end notice。
		// 显式设置路径（completed/blocked/content-loop 等）原因已由 EventDone 携带，
		// 不再重复发 notice，避免事件流末尾出现非 done 事件干扰断言。
		l.emit(Event{Type: EventNotice, Content: fmt.Sprintf("[turn/%d/end] 原因=%s", l.TurnNo, l.LastTurnReason)})
	}
}

// turnStickyReason 应用 max-tokens sticky 语义：
// 本轮 turn 内任一 step 曾触发输出长度截断（hadMaxTokens）时，
// 最终结束原因固定为 max-tokens，不得被后续正常完成的 step 降级为 completed。
// 对应「max-tokens stays sticky」的处理。
func (l *Loop) turnStickyReason(base TurnEndReason) TurnEndReason {
	if l.hadMaxTokens {
		return TurnMaxTokens
	}
	return base
}
