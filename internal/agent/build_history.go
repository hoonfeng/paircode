package agent

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
