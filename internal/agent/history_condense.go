package agent

import (
	"fmt"
	"strings"
)

// history_condense.go — 历史消息精简：跨轮次加载时将旧轮次的所有正文内容
// 拼接为一条 user 消息，丢弃消息结构开销（role/tool_call/tool_call_id 等元数据），
// 保留全部文字信息。

// CondenseHistory 将已完成的旧轮次压缩成一条 user 消息，包含旧轮次全部正文。
// 当前轮次（最后一次用户消息之后）完整保留。
//
// 例：
//   输入: [User("改A"), Asst(tool1), Tool("file ok"), Asst("已改A"), User("改B"), ...当前轮...]
//   输出: [User("【历史对话】\n用户：改A\n助手：已改A\n\n用户：改B\n..."), User("改B"), ...当前轮...]
//                             ↑ 旧轮次合并为一条
func CondenseHistory(msgs []Message) []Message {
	if len(msgs) < 4 {
		return msgs
	}

	// 找到 user 消息位置（轮次边界）
	var userIdx []int
	for i, m := range msgs {
		if m.Role == RoleUser {
			userIdx = append(userIdx, i)
		}
	}
	if len(userIdx) <= 1 {
		return msgs // 只有一个轮次，无需精简
	}
	lastUserPos := userIdx[len(userIdx)-1]

	// 拼接所有旧轮次的正文内容（只保留用户消息和助理正文）
	var history strings.Builder
	history.WriteString("【历史对话】\n\n")

	for t := 0; t < len(userIdx)-1; t++ {
		start := userIdx[t]
		end := userIdx[t+1]
		if t == len(userIdx)-2 {
			end = lastUserPos
		}

		// 遍历该轮次所有消息，提取正文
		for i := start; i < end; i++ {
			m := msgs[i]
			content := strings.TrimSpace(m.Content)
			if content == "" {
				continue
			}

			switch m.Role {
			case RoleUser:
				history.WriteString(fmt.Sprintf("### 用户\n%s\n\n", content))
			case RoleAssistant:
				history.WriteString(fmt.Sprintf("### 助手\n%s\n\n", content))
			// RoleTool 跳过——工具结果对后续理解无帮助
			}
		}
	}

	// 构建结果：system 前缀 + 合并的历史块 + 当前轮次完整保留
	out := make([]Message, 0, len(userIdx)+1)
	out = append(out, msgs[:userIdx[0]]...)
	out = append(out, Message{Role: RoleUser, Content: history.String()})

	// 保留当前轮次（从 lastUserPos 开始，保持其用户消息原文）
	out = append(out, msgs[lastUserPos:]...)

	return out
}
