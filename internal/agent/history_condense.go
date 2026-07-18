package agent

import "strings"

// history_condense.go — 历史消息精简。
//
// 每轮 = 用户发一次消息 + agent 完整回复。当新轮次开始时，之前所有已完成的轮次
// 只需要保留【用户消息 + 助理最终报告】，中间的 tool_call/tool_result 全部移除。
// 当前轮次（最后一次用户消息之后的所有消息）完整保留。
//
// 这样每轮 LLM 提交的数据量大幅减少（tool_result 常含大段文件内容/命令输出），
// 而语义连续性不受影响——新 agent 知道用户之前要求了什么、最终完成了什么。

// CondenseHistory 精简已完成的历史轮次，每个旧轮次只保留 [用户消息, 助理最终报告]。
// 当前轮次完整保留在尾部。
//
// 跨轮次缓存前缀必然变化（新旧 User 消息不同），精简省下的 token >> 缓存损失。
func CondenseHistory(msgs []Message) []Message {
	if len(msgs) < 4 {
		return msgs
	}

	// 找到所有 user 消息位置（轮次边界）
	var userIdx []int
	for i, m := range msgs {
		if m.Role == RoleUser {
			userIdx = append(userIdx, i)
		}
	}
	if len(userIdx) <= 1 {
		return msgs // 只有一个或零个轮次，无需精简
	}
	lastUserPos := userIdx[len(userIdx)-1]

	// 构建结果：保留 system 前缀 + 精简的旧轮次 + 完整的最新轮次
	out := make([]Message, 0, lastUserPos+4)

	// 先复制 system 前缀（msgs[0..userIdx[0]] 之间的可能非 user 消息）
	out = append(out, msgs[:userIdx[0]]...)

	// 处理每个旧轮次（最后一个用户消息之前的轮次）
	for t := 0; t < len(userIdx)-1; t++ {
		start := userIdx[t]
		end := userIdx[t+1]
		if t == len(userIdx)-2 {
			end = lastUserPos // 最后一个旧轮次结束于最新 user 之前
		}

		// 保留用户消息
		out = append(out, msgs[start])

		// 找该轮次最后一条有正文的 assistant 消息（最终报告）
		var finalReport string
		for i := end - 1; i > start; i-- {
			if msgs[i].Role == RoleAssistant && strings.TrimSpace(msgs[i].Content) != "" {
				finalReport = msgs[i].Content
				break
			}
		}

		// 注入精简的最终报告（包裹标记，截断到 500 字）
		if finalReport != "" {
			if len(finalReport) > 500 {
				finalReport = finalReport[:500] + "…（后续报告已截断）"
			}
			out = append(out, Message{
				Role:    RoleAssistant,
				Content: "\n【上轮完成报告】\n" + finalReport + "\n【报告结束】\n",
			})
		}
	}

	// 追加最新一轮完整消息
	out = append(out, msgs[lastUserPos:]...)

	return out
}
