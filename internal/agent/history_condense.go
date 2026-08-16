package agent

import (
	"fmt"
	"strings"
)

// history_condense.go — 历史消息精简：跨轮次加载时将旧轮次压缩为结构化摘要。
//
// 设计原则（v3）：
//   - 最近 1 轮完整保留（保留完整工具调用链，Agent 需要知道最近干了什么）
//   - 倒数第 2 轮「半压缩」：用户消息 + 助手正文保留，工具调用子链合并为一行摘要
//     （工具输出是体积大头，半压缩后该轮体积骤降，但语义连续性不丢）
//   - 更早的轮次压缩为结构化摘要：用户请求 + 使用的工具 + 最终结果
//   - 摘要保留工具名和执行摘要，Agent 据此知道"之前用了哪些工具、做了什么操作"
//   - 不再丢弃 tool_call/tool_result 信息（v1 最大的问题）
//   - 防摘要嵌套：已存在的【历史对话摘要】消息整体并入新摘要（截断），不再被当普通轮次递归压缩
//   - 摘要总量上限：防止对话无限拉长后摘要本身膨胀

const (
	// keepFullRounds 最近完整保留的轮次数（完整工具调用链）。
	keepFullRounds = 1
	// keepSemiRounds 倒数第 2 层半压缩的轮次数（工具调用子链合并为一行摘要）。
	keepSemiRounds = 1
	// maxCondensedChars 压缩摘要总量上限（rune）。
	// 超过后停止追加新轮次条目，保证摘要本身不膨胀。
	maxCondensedChars = 1500
	// oldSummaryMaxChars 已存在的旧摘要合并进新摘要时的截断上限。
	// 旧摘要本身可能已接近上限，合并时优先保留新轮次内容（旧摘要只截断保留开头）。
	oldSummaryMaxChars = 600
	// maxFullRoundMsgs 完整保留轮的消息数上限。
	// 最近 1 轮若超过此数（极端自主长跑），更早的迭代子链合并为一行摘要，
	// 只保留尾部关键迭代——防止单轮超长上下文拖垮下一轮注入体积。
	maxFullRoundMsgs = 30
)

// CondenseHistoryByPressure 按 token 压力触发历史精简（★ 对齐 harness
// compaction-basic：thresholdRatio 触发 + 保留尾部，2026-08-17）。
//
// 背景：原 CondenseHistory 按「轮数」强制压缩（历史轮次 > 2 即压缩）——
// 小对话（仅几轮、token 远未达窗口）也会被改写历史前缀，导致每次请求的
// 前缀都与上一轮不同，KV 缓存从压缩点整段断裂、命中率骤降。
// harness 只在请求压力达阈值（默认 0.8 × 上下文窗口）时才压缩，小对话
// 保持原始消息逐字节不变 → 前缀稳定 → 连续轮次共享缓存命中。
//
// 策略（对齐 maybeCompact 的两档阈值）：
//   - 估算历史 token 占比 < compactRatioEarly（0.45）且未达硬地板 → 不压缩
//   - 达到阈值 → CondenseHistory 压缩（保留最近轮 + 摘要）
//   - maxTokens <= 0（未配置窗口）→ 以 compactHardFloor 绝对量兜底
func CondenseHistoryByPressure(msgs []Message, maxTokens int) []Message {
	if len(msgs) < 4 {
		return msgs
	}
	if maxTokens <= 0 {
		maxTokens = compactHardFloor
	}
	tokens := estimateTokens(msgs)
	ratio := float64(tokens) / float64(maxTokens)
	// 未达预压缩阈值且未触碰硬地板：保持原始历史（KV 前缀稳定，缓存可命中）
	if ratio < compactRatioEarly && tokens < compactHardFloor {
		return msgs
	}
	return CondenseHistory(msgs)
}

// CondenseHistory 将已完成的旧轮次压缩：最近 1 轮完整保留、倒数第 2 轮半压缩、
// 更早轮次压缩为结构化摘要。
//
// 输出结构：
//
//	[system(如有)] + [半压缩轮] + [最近1轮完整交互] + [压缩摘要 user msg] + [当前用户消息]
//
// 压缩摘要格式：
//
//	【历史对话摘要】
//	轮次1: 用户说"..."，助手使用了 read_file, edit_file，结果：已修改配置
//	轮次2: 用户说"..."，助手使用了 run_command，结果：构建成功
//
// ★ KV 缓存权衡说明：
// 压缩（删除/替换中段消息）必然导致消息数组位置错位，跨轮次首请求的缓存前缀
// 会从被压缩位置断裂（这是压缩与 KV 前缀的根本矛盾）。为尽量缓解：
//  1. 摘要作为回顾性 user 消息放在最近轮次之后、当前用户消息之前（不占位置 2）；
//  2. 完整保留轮控制在 1 轮 + 半压缩 1 轮，降低保留体积的同时不让压缩频率失控。
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
	totalRounds := len(userIdx) // 含最后一条 user 消息（当前任务）
	keepRounds := keepFullRounds + keepSemiRounds
	if totalRounds-1 <= keepRounds {
		return msgs // 历史轮次不足（保留轮 + 当前消息），无需精简
	}

	// 分层边界（最后一条 user = 当前消息，不参与压缩/保留轮）：
	//   摘要轮 = userIdx[0 .. compressEnd)
	//   半压缩轮 = userIdx[compressEnd]（倒数第 2 层）
	//   完整轮 = userIdx[compressEnd+keepSemiRounds]（最近 1 轮）
	//   当前消息 = userIdx[totalRounds-1]
	compressEnd := totalRounds - keepRounds - 1
	semiStart := userIdx[compressEnd]
	fullStart := userIdx[compressEnd+keepSemiRounds]
	lastUserIdx := userIdx[totalRounds-1]

	// 构建压缩摘要（只含摘要轮，不含半压缩/完整轮）
	summary := buildCondensedSummary(msgs, userIdx, compressEnd)

	// 构建结果：system 前缀 + 半压缩轮 + 最近 1 轮完整原始消息 + 压缩摘要（回顾） + 当前用户消息
	out := make([]Message, 0, len(userIdx)+2)
	// 保留 system 消息（如有）
	out = append(out, msgs[:userIdx[0]]...)
	// 半压缩轮：工具调用子链合并为一行摘要，其余保留
	if keepSemiRounds > 0 && semiStart < fullStart {
		out = append(out, condenseRoundSemi(msgs, semiStart, fullStart)...)
	}
	// 最近 1 轮完整交互（原始消息，保持位置对齐）
	// ★ 消息数上限：超长轮（极端自主长跑）更早的迭代子链合并为一行摘要，
	//   只保留尾部 maxFullRoundMsgs 条关键迭代，防止单轮超长上下文拖垮下一轮。
	if fullStart < lastUserIdx {
		if fullLen := lastUserIdx - fullStart; fullLen > maxFullRoundMsgs {
			split := lastUserIdx - maxFullRoundMsgs
			// split 后移越过孤立的 tool 结果（其配对 assistant 落在合并段内）
			for split < lastUserIdx && msgs[split].Role == RoleTool {
				split++
			}
			if split > fullStart {
				out = append(out, condenseRoundSemi(msgs, fullStart, split)...)
			}
			out = append(out, msgs[split:lastUserIdx]...)
		} else {
			out = append(out, msgs[fullStart:lastUserIdx]...)
		}
	}
	// 压缩摘要：回顾性 user 消息，位于当前用户消息之前
	out = append(out, Message{Role: RoleUser, Content: summary})
	// 当前用户消息（及之后，如有）
	out = append(out, msgs[lastUserIdx:]...)

	return out
}

// buildCondensedSummary 构建压缩摘要文本（仅含 [0, compressEnd) 的轮次）。
// 旧摘要消息（以【历史对话摘要】开头）整体并入（截断），不当作普通轮次递归压缩；
// 其余轮次逐条构建，受 maxCondensedChars 总量上限约束。
func buildCondensedSummary(msgs []Message, userIdx []int, compressEnd int) string {
	var summary strings.Builder
	summary.WriteString("【历史对话摘要】\n")
	summary.WriteString("> 以下为更早轮次的摘要（最近 " + fmt.Sprint(keepFullRounds+keepSemiRounds) + " 轮完整/半完整保留在下方）。\n")
	summary.WriteString("> Agent 应据此了解之前的操作，避免重复工作。\n\n")

	chars := 0
	for t := 0; t < compressEnd; t++ {
		start := userIdx[t]
		end := len(msgs) // 默认到末尾
		if t+1 < len(userIdx) {
			end = userIdx[t+1]
		}

		userText := ""
		var tools []string
		var toolResults []string
		assistantFinal := ""

		for i := start; i < end && i < len(msgs); i++ {
			m := msgs[i]
			switch m.Role {
			case RoleUser:
				userText = strings.TrimSpace(m.Content)
			case RoleAssistant:
				// 提取工具调用
				for _, tc := range m.ToolCalls {
					tools = append(tools, tc.Function.Name)
				}
				// 提取正文（最后一条非空正文作为本轮结果）
				if content := strings.TrimSpace(m.Content); content != "" {
					assistantFinal = content
				}
			case RoleTool:
				// 提取工具执行结果的关键信息
				if m.Content != "" {
					result := summarizeToolResult(m.Name, m.Content)
					if result != "" {
						toolResults = append(toolResults, result)
					}
				}
			}
		}

		// ★ 防摘要嵌套：旧摘要消息不当作普通轮次，整体并入（截断）
		if strings.HasPrefix(userText, "【历史对话摘要】") {
			if chars < maxCondensedChars {
				old := truncateRunes(userText, oldSummaryMaxChars)
				summary.WriteString("**前序摘要**：" + old + "\n\n")
				chars += len([]rune(old)) + 16
			}
			continue
		}

		// 构建本轮摘要
		var round strings.Builder
		round.WriteString(fmt.Sprintf("**轮次 %d**：", t+1))
		if userText != "" {
			round.WriteString(fmt.Sprintf(" 用户「%s」", truncateRunes(userText, 150)))
		}

		if len(tools) > 0 {
			// 去重工具名
			seen := make(map[string]bool)
			var uniqueTools []string
			for _, name := range tools {
				if !seen[name] {
					seen[name] = true
					uniqueTools = append(uniqueTools, name)
				}
			}
			round.WriteString(fmt.Sprintf(" → 使用了 %s", strings.Join(uniqueTools, ", ")))
		}

		// 工具结果只保留最多 2 条，避免摘要膨胀
		if len(toolResults) > 2 {
			toolResults = toolResults[:2]
		}
		if len(toolResults) > 0 {
			round.WriteString("（" + strings.Join(toolResults, "；") + "）")
		}

		if assistantFinal != "" {
			final := truncateRunes(assistantFinal, 200)
			round.WriteString(fmt.Sprintf(" → %s", final))
		}

		round.WriteString("\n")
		roundText := round.String()
		if chars+len([]rune(roundText)) > maxCondensedChars {
			break // 摘要已到上限，不再追加
		}
		summary.WriteString(roundText)
		chars += len([]rune(roundText))
	}
	return summary.String()
}

// condenseRoundSemi 半压缩一轮：保留用户消息与助手正文，
// 把「助手 tool_calls + 对应 tool 结果」子链整体合并为一行摘要。
// ★ 子链整体替换（而非删除 tool 结果保留 tool_calls）——保证消息配对完整，
//   不会产生孤立 tool 结果 / 未配对 tool_calls（OpenAI 规范要求）。
func condenseRoundSemi(msgs []Message, start, end int) []Message {
	out := make([]Message, 0, end-start)
	i := start
	for i < end {
		m := msgs[i]
		if m.Role == RoleAssistant && len(m.ToolCalls) > 0 {
			// 工具调用子链：收集工具名，并找到其后连续的 tool 结果（配对）
			toolNames := make([]string, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				toolNames = append(toolNames, tc.Function.Name)
			}
			j := i + 1
			for j < end && msgs[j].Role == RoleTool {
				j++
			}
			toolSummary := "（工具调用 " + strings.Join(toolNames, ", ") + "）"
			content := strings.TrimSpace(m.Content)
			if content == "" {
				content = toolSummary
			} else {
				content = truncateRunes(content, 500) + "\n" + toolSummary
			}
			out = append(out, Message{Role: RoleAssistant, Content: content})
			i = j
			continue
		}
		out = append(out, m)
		i++
	}
	return out
}

// summarizeToolResult 从工具执行结果中提取关键信息。
// 对于不同工具，提取不同格式的摘要。
func summarizeToolResult(toolName, content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	switch toolName {
	case "read_file":
		// 提取首行或文件摘要
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) > 0 {
			return "读取文件: " + truncateRunes(strings.TrimSpace(lines[0]), 80)
		}
	case "write_file", "edit_file", "multi_edit":
		return "已编辑文件"
	case "run_command":
		// 提取命令结果的关键字
		if strings.Contains(content, "exit status 0") || strings.Contains(content, "成功") {
			return "命令执行成功"
		}
		if strings.Contains(content, "error") || strings.Contains(content, "exit status") {
			return "命令执行出错"
		}
		return "已执行命令"
	case "search_content", "search_files":
		if strings.Contains(content, "No matches") || strings.Contains(content, "未找到") {
			return "未找到结果"
		}
		return "已搜索"
	case "go_build":
		if strings.Contains(content, "成功") || strings.Contains(content, "exit status 0") {
			return "构建成功"
		}
		return "构建完成"
	case "run_test":
		if strings.Contains(content, "PASS") {
			return "测试通过"
		}
		return "测试完成"
	case "update_tasks":
	case "web_debug":
		return "已打开页面验证"
	case "codegraph_search", "codegraph_function", "codegraph_impact":
		return "已查询代码图谱"
	default:
		// 通用：取前 60 个字符
		return truncateRunes(content, 60)
	}
	return ""
}

// truncateRunes 按 rune 截断文本。
func truncateRunes(s string, maxLen int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= maxLen {
		return string(runes)
	}
	return string(runes[:maxLen]) + "…"
}
