// Package summary 提供对话摘要生成与压缩。
// 将对话消息压缩为结构化摘要，并在历史摘要过多时自动压缩归档。
package summary

import (
	"fmt"
	"strings"
)

// maxTotalChars 所有摘要总长度上限，超限则触发压缩。
const maxTotalChars = 3000

// maxConvs 保留独立摘要的对话数上限，超出后压缩最早的。
const maxConvs = 8

// maxConvMsgs 用于生成摘要的最大消息数（取最近 N 条）。
const maxConvMsgs = 30

// Message 一条对话消息（summary 包的最小依赖接口）。
type Message struct {
	Role    string // "user" 或 "assistant"
	Content string
}

// ConvInfo 对话摘要所需的信息。
type ConvInfo struct {
	ID        string
	Title     string
	CreatedAt string
	UpdatedAt string
	Summary   string
	SummaryAt string
	Messages  []Message
}

// Generate 生成一条对话的结构化摘要（规则式）。
func Generate(conv ConvInfo) string {
	msgs := conv.Messages
	if len(msgs) == 0 {
		return ""
	}
	if len(msgs) > maxConvMsgs {
		msgs = msgs[len(msgs)-maxConvMsgs:]
	}

	var b strings.Builder
	for _, msg := range msgs {
		role := "用户"
		if msg.Role == "assistant" {
			role = "助手"
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		runes := []rune(content)
		if len(runes) > 500 {
			content = string(runes[:500]) + "…"
		}
		b.WriteString(role + ": " + content + "\n\n")
	}
	convText := b.String()
	if convText == "" {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(convText), "\n\n")
	var goal, lastAssistant string
	toolCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "用户: ") {
			userContent := strings.TrimPrefix(line, "用户: ")
			if goal == "" {
				goal = userContent
			}
		} else if strings.HasPrefix(line, "助手: ") {
			lastAssistant = strings.TrimPrefix(line, "助手: ")
		}
		if strings.Contains(line, "read_file") || strings.Contains(line, "write_file") ||
			strings.Contains(line, "edit_file") || strings.Contains(line, "search_") ||
			strings.Contains(line, "run_") {
			toolCount++
		}
	}
	goal = Truncate(goal, 200)
	lastAssistant = Truncate(lastAssistant, 300)

	parts := make([]string, 0, 4)
	if goal != "" {
		parts = append(parts, "目标: "+goal)
	}
	if toolCount > 0 {
		parts = append(parts, fmt.Sprintf("进行了 %d+ 步工具操作", toolCount))
	}
	if lastAssistant != "" {
		parts = append(parts, "关键上下文: "+lastAssistant)
	}
	parts = append(parts, "状态: 已完成")

	return strings.Join(parts, " | ")
}

// Truncate 按 rune 截断文本，超长加省略号。
func Truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

// 可压缩的对话集
type compressConv struct {
	idx       int
	createdAt string
}

// MaybeCompress 检查所有历史摘要的总长度，如果超限则将最早的一半合并压缩。
// convs 是对话列表，summaryAccessor 用于读写对话的摘要。
func MaybeCompress(convs []ConvInfo) []ConvInfo {
	entries := make([]compressConv, 0, len(convs))
	for i, c := range convs {
		if c.Summary != "" {
			entries = append(entries, compressConv{idx: i, createdAt: c.CreatedAt})
		}
	}

	totalChars := 0
	for _, e := range entries {
		totalChars += len([]rune(convs[e.idx].Summary))
	}
	if len(entries) <= maxConvs && totalChars <= maxTotalChars {
		return convs
	}

	keepCount := (len(entries) + 1) / 2
	if keepCount < 2 {
		keepCount = 2
	}
	if keepCount >= len(entries) {
		return convs
	}

	toCompress := entries[keepCount:]
	toKeep := entries[:keepCount]

	var sb strings.Builder
	sb.WriteString("【历史对话归档】")
	for _, e := range toCompress {
		c := convs[e.idx]
		title := c.Title
		if title == "" || title == "新对话" {
			title = "对话"
		}
		s := c.Summary
		if len([]rune(s)) > 300 {
			s = string([]rune(s)[:300]) + "…"
		}
		sb.WriteString(title + ": " + s + " | ")
	}
	archiveSummary := strings.TrimSuffix(sb.String(), " | ")

	if len(toKeep) > 0 {
		lastKeep := toKeep[len(toKeep)-1]
		existing := convs[lastKeep.idx].Summary
		if existing != "" {
			convs[lastKeep.idx].Summary = existing + " | " + archiveSummary
		} else {
			convs[lastKeep.idx].Summary = archiveSummary
		}
	}

	for _, e := range toCompress {
		convs[e.idx].Summary = ""
		convs[e.idx].SummaryAt = ""
	}

	return convs
}
