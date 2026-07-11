// subagent_sink.go 子 Agent 事件过滤器：防止子 Agent 的完成/错误等生命周期事件泄漏到父 Agent。
//
// 核心设计（参考 DeepSeek-Reasonix subSink 模式）：
//   子 Agent 的 OnEvent 不直接指向父 OnEvent，而是经过 SubAgentSink 过滤后转发。
//   只转发「有意义」的事件（工具调用、思考、内容、用量），丢弃生命周期事件（完成、错误）。
//   同时标记 AgentName，供前端视觉区分。
//
// 泄漏问题修复：
//   - 子 Agent 的 EventFinal 不再设置 bridge.go 的 m.Streaming=false，防止提前结束整轮对话
//   - 子 Agent 的 EventError 不再触发父 Agent 的错误处理逻辑
//   - 并行执行的子 Agent 事件带上 AgentName，前端可据此渲染不同标签

package agent

// SubAgentSink 创建子 Agent 的事件过滤器。
// 子 Agent 的 OnEvent 应使用此函数包装，而非直接使用父 OnEvent。
//
// 过滤规则：
//   - EventFinal：丢弃（子完成 ≠ 父完成，结果由 finishTask 或 session 回传）
//   - EventDone：丢弃（子 Agent 的结构化完成信号不应泄漏到父事件流）
//   - EventError：丢弃（子内部错误不结束父 Loop）
//   - EventCircling：丢弃（子的绕圈是内部问题）
//   - EventCompacted：丢弃（子的压缩是内部优化，不通知前端）
//   - 其余事件：标记 agentName 后转发
//
// parentOnEvent 为父 Loop.OnEvent（可空，空则全部丢弃）。
// agentName 为子 Agent 名（用于前端视觉区分）。
func SubAgentSink(parentOnEvent func(Event), agentName string) func(Event) {
	if parentOnEvent == nil {
		return func(Event) {} // 无父 → 全部丢弃
	}
	return func(e Event) {
		switch e.Type {
		case EventFinal, EventDone, EventError, EventCircling, EventCompacted:
			// 丢弃：子 Agent 的生命周期事件不应泄漏到父事件流
			return
		default:
			e.AgentName = agentName // 标记来源 Agent 名
			parentOnEvent(e)
		}
	}
}
