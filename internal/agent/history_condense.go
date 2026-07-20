package agent

import (
	"fmt"
	"strings"
)

// history_condense.go — 历史消息精简：跨轮次加载时将旧轮次压缩为结构化摘要。
//
// 设计原则（v2）：
//   - 最近 2 轮完整保留（保留完整工具调用链，Agent 需要知道最近干了什么）
//   - 更早的轮次压缩为结构化摘要：用户请求 + 使用的工具 + 最终结果
//   - 摘要保留工具名和执行摘要，Agent 据此知道"之前用了哪些工具、做了什么操作"
//   - 不再丢弃 tool_call/tool_result 信息（v1 最大的问题）

// keepFullRounds 最近保留完整交互的轮次数。
const keepFullRounds = 2

// CondenseHistory 将已完成的旧轮次压缩，保留最近 keepFullRounds 轮完整交互。
//
// 输出结构：
//
//	[system(如有)] + [压缩摘要 user msg] + [最近2轮完整保留]
//
// 压缩摘要格式：
//
//	【历史对话摘要】
//	轮次1: 用户说"..."，助手使用了 read_file, edit_file，结果：已修改配置
//	轮次2: 用户说"..."，助手使用了 run_command，结果：构建成功
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
	totalRounds := len(userIdx)
	if totalRounds <= keepFullRounds {
		return msgs // 轮次不足，无需精简
	}

	// 需要压缩的轮次：从第 0 轮到第 (totalRounds - keepFullRounds - 1) 轮
	compressEnd := totalRounds - keepFullRounds
	lastCompressUserPos := userIdx[compressEnd] // 最后一个需要压缩轮次的 user 消息位置

	// 构建压缩摘要
	var summary strings.Builder
	summary.WriteString("【历史对话摘要】\n")
	summary.WriteString("> 以下为更早轮次的摘要（最近 " + fmt.Sprint(keepFullRounds) + " 轮完整保留在下方）。\n")
	summary.WriteString("> Agent 应据此了解之前的操作，避免重复工作。\n\n")

	for t := 0; t < compressEnd; t++ {
		start := userIdx[t]
		end := len(msgs) // 默认到末尾
		if t+1 < len(userIdx) {
			end = userIdx[t+1]
		}

		roundNum := t + 1
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

		// 构建本轮摘要
		summary.WriteString(fmt.Sprintf("**轮次 %d**：", roundNum))
		if userText != "" {
			summary.WriteString(fmt.Sprintf(" 用户「%s」", truncateRunes(userText, 150)))
		}

		if len(tools) > 0 {
			// 去重工具名
			seen := make(map[string]bool)
			var uniqueTools []string
			for _, t := range tools {
				if !seen[t] {
					seen[t] = true
					uniqueTools = append(uniqueTools, t)
				}
			}
			summary.WriteString(fmt.Sprintf(" → 使用了 %s", strings.Join(uniqueTools, ", ")))
		}

		if len(toolResults) > 0 {
			summary.WriteString("（" + strings.Join(toolResults, "；") + "）")
		}

		if assistantFinal != "" {
			final := truncateRunes(assistantFinal, 200)
			summary.WriteString(fmt.Sprintf(" → %s", final))
		}

		summary.WriteString("\n")
	}

	// 构建结果：system 前缀 + 压缩摘要 + 保留的完整轮次
	out := make([]Message, 0, len(userIdx)+2)
	// 保留 system 消息（如有）
	out = append(out, msgs[:userIdx[0]]...)
	// 注入压缩摘要
	out = append(out, Message{Role: RoleUser, Content: summary.String()})
	// 保留最近 keepFullRounds 轮完整交互
	out = append(out, msgs[lastCompressUserPos:]...)

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
		return "更新了任务列表"
	case "finish_task":
		return "提交了完成结果"
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
