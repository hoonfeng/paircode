package agent

// 执行日志 — 自主模式下，外层 agent 的各轮分析/决策/结果的结构化日志。
// 存储在 Loop.State["executionLog"]，与消息历史隔离，不受上下文压缩影响。
// 每轮 LLM 调用前注入系统提示的【执行日志】段。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExecutionEntry 执行日志的一条记录。
type ExecutionEntry struct {
	Round   int    `json:"round"`   // 轮次（从1开始）
	Agent   string `json:"agent"`   // 来源 agent 名（"outer"/"coder"/"planner"等）
	Phase   string `json:"phase"`   // 阶段：analysis / delegation / execution / review / feedback
	Summary string `json:"summary"` // 关键摘要（约200字以内）
}

// ExecutionLog 持久化的执行日志，按轮次组织。
// 每条 log 条目对应一轮委托的完整「分析→委托→执行→结果」循环。
type ExecutionLog struct {
	Entries []ExecutionEntry `json:"entries"`
	Round   int              `json:"round"` // 当前轮次计数器
}

const maxExecutionLogEntries = 15 // 最多保留15条，超出时移除最旧的

// GetExecutionLog 从 Loop.State 获取或初始化执行日志。
func (l *Loop) GetExecutionLog() *ExecutionLog {
	if l.State == nil {
		l.State = map[string]any{}
	}
	raw, ok := l.State["executionLog"]
	if !ok || raw == nil {
		log := &ExecutionLog{Entries: []ExecutionEntry{}, Round: 0}
		l.State["executionLog"] = log
		return log
	}
	if log, ok := raw.(*ExecutionLog); ok {
		return log
	}
	// 类型不匹配时重建
	log := &ExecutionLog{Entries: []ExecutionEntry{}, Round: 0}
	l.State["executionLog"] = log
	return log
}

// LogEntry 追加一条执行日志条目（自动递增轮次、自动截断）。
func (l *Loop) LogEntry(agent, phase, summary string) {
	log := l.GetExecutionLog()
	log.Round++
	entry := ExecutionEntry{
		Round:   log.Round,
		Agent:   agent,
		Phase:   phase,
		Summary: summary,
	}
	log.Entries = append(log.Entries, entry)
	// 超出上限时移除最旧的
	if len(log.Entries) > maxExecutionLogEntries {
		log.Entries = log.Entries[len(log.Entries)-maxExecutionLogEntries:]
	}
}

// LogAnalysis 便捷方法：记录外层 agent 的一轮分析内容。
// content 是 assistant 返回的纯分析文本（不含工具调用部分）。
func (l *Loop) LogAnalysis(content string) {
	if content == "" {
		return
	}
	summary := l.condense(content)
	l.LogEntry("outer", "analysis", summary)
}

// LogDelegation 便捷方法：记录一次委托给子 agent。
func (l *Loop) LogDelegation(agentName, task string) {
	summary := l.condense(task)
	l.LogEntry(agentName, "delegation", summary)
}

// LogResult 便捷方法：记录子 agent 的执行结果。
func (l *Loop) LogResult(agentName, result string) {
	summary := l.condense(result)
	l.LogEntry(agentName, "execution", summary)
}

// FormatExecutionLog 格式化执行日志为可读文本，用于注入系统提示。
// 返回最近 8 条摘要，每条约 80 字符。
func (l *Loop) FormatExecutionLog() string {
	log := l.GetExecutionLog()
	if len(log.Entries) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n# 执行日志（本轮分析与之前各轮摘要）\n")
	b.WriteString("以下是本轮及之前各轮的分析和操作记录，供参考：\n\n")

	// 显示最近 8 条
	start := 0
	if len(log.Entries) > 8 {
		start = len(log.Entries) - 8
		b.WriteString(fmt.Sprintf("（仅显示最近 8 条，共 %d 条）\n\n", len(log.Entries)))
	}
	for _, e := range log.Entries[start:] {
		short := e.Summary
		if len(short) > 80 {
			short = short[:80] + "…"
		}
		b.WriteString(fmt.Sprintf("  [第%d轮/%s] %s: %s\n", e.Round, e.Agent, e.Phase, short))
	}
	return b.String()
}

// condense 截断过长文本。
func (l *Loop) condense(s string) string {
	if len(s) <= 200 {
		return s
	}
	return s[:197] + "…"
}

// logKeyExists 工具函数：检查执行日志中是否已有某轮某阶段的记录（避免重复追加）。
func (l *Loop) logKeyExists(phase string) bool {
	log := l.GetExecutionLog()
	for i := len(log.Entries) - 1; i >= 0; i-- {
		if log.Entries[i].Phase == phase {
			return true
		}
	}
	return false
}

// ── 磁盘持久化 ──

// executionLogDir 返回执行日志存储目录。
func executionLogDir(root string) string {
	return filepath.Join(root, ".pair", "execution_logs")
}

// executionLogPath 返回指定 convID 的执行日志文件路径。
func executionLogPath(root, convID string) string {
	return filepath.Join(executionLogDir(root), convID+".json")
}

// SaveExecutionLog 将执行日志持久化到磁盘。
// 在自主模式每轮结束时调用，供下一轮恢复。
func SaveExecutionLog(root, convID string, log *ExecutionLog) {
	if log == nil || len(log.Entries) == 0 {
		return
	}
	dir := executionLogDir(root)
	os.MkdirAll(dir, 0o755)
	path := executionLogPath(root, convID)
	data, err := json.Marshal(log)
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0o644)
}

// LoadExecutionLog 从磁盘加载执行日志。
// 新自主开始时调用，恢复上一轮的完整执行日志。
func LoadExecutionLog(root, convID string) *ExecutionLog {
	path := executionLogPath(root, convID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var log ExecutionLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil
	}
	return &log
}

// ClearExecutionLog 清除指定 convID 的执行日志文件。
// 对话删除时调用，避免残留。
func ClearExecutionLog(root, convID string) {
	path := executionLogPath(root, convID)
	os.Remove(path)
}
