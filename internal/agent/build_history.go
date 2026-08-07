package agent

import "strings"

// BuildHistory 从消息列表构建 LLM 上下文（供 loop.Run 作为 history 参数）。
//
// 消息列表的最后一条应是当前用户消息——调用方通常在追加当前消息到存储后调用本函数。
// loop.Run 内部会把 task（即当前消息）再次添加到 messages 末尾，
// 故需排除这条用户消息以避免重复。
//
// 安全边界：仅当最后一条消息 Role == RoleUser 时才排除。若最后一条不是用户消息（例如
// 助手回复），说明调用方尚未追加当前用户消息到存储，则不排除任何消息——防止丢失上轮回复。
//
// 调用方需自行完成原始消息到 agent.Message 的转换（含角色映射、摘要拼接等）。
func BuildHistory(messages []Message) []Message {
	if len(messages) == 0 {
		return nil
	}
	// 只排除最后一条用户消息（loop.Run 会通过 task 参数重新添加它）。
	// 若最后一条不是用户消息，说明当前用户消息尚未存入列表，不排除以免丢失助理回复。
	end := len(messages)
	if messages[end-1].Role == RoleUser {
		end--
	}
	hist := make([]Message, end)
	copy(hist, messages[:end])
	return hist
}

// CopyHistory 复制历史消息切片（供下游 append 不污染源）。
// 与 BuildHistory 配合使用：BuildHistory 返回的切片底层引用源数组，
// 后续 append 可能意外覆盖，故在传给 loop.Run 前应深复制。
func CopyHistory(hist []Message) []Message {
	out := make([]Message, len(hist))
	copy(out, hist)
	return out
}

// historyUserMarker 历史轮次用户消息标注前缀。
// 同一对话线程内发起新轮次时，历史轮次的用户消息与当前任务都是 RoleUser，
// 不加标注时 LLM 容易把旧轮次的用户消息误认为「本次提交的信息」，造成理解污染。
const historyUserMarker = "【历史轮次消息·非当前任务】\n"

// MarkHistoryUserMessages 给消息列表中除最后一条用户消息（=当前任务）外的
// 所有用户消息加历史轮次标注，使 Agent 明确区分「历史轮次消息」与「本次提交的任务」。
//
// skipPrefix：跳过前 N 条消息不标注（不检查内容）。用于 delegate 子 Loop——
// 子 Loop 继承父 msgs 作 history 前缀（KV Cache 前缀一致要求逐字节相同），
// 父 msgs 已由父 Loop 标注过（或当前任务保持未标注），子 Loop 不得再改动。
//
// 幂等：已有标注的消息不重复加前缀（防止跨 Run 复用 msgs 时前缀嵌套）。
// 只作用于 LLM 上下文副本，不写回持久化历史（持久化走原始消息，UI 展示无损）。
func MarkHistoryUserMessages(msgs []Message, skipPrefix int) {
	if len(msgs) < 2 {
		return
	}
	// 最后一条 RoleUser 即当前任务，不标注
	lastUser := -1
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleUser {
			lastUser = i
			break
		}
	}
	if lastUser < 0 {
		return
	}
	for i := range msgs {
		if i < skipPrefix || i == lastUser || msgs[i].Role != RoleUser {
			continue
		}
		if strings.HasPrefix(msgs[i].Content, historyUserMarker) {
			continue // 已标注（幂等）
		}
		msgs[i].Content = historyUserMarker + msgs[i].Content
	}
}
