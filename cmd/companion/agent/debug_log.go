package agent

// debug_log.go — 运行时错误日志持久化框架。
// 将 agent 运行时的 panic、构建错误、工具执行错误等写入 .pair/tasks/debug-logs/ 目录。
// 支持自动清理旧日志（保留最近 50 条），支持按对话/会话筛选。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── 类型定义 ───────────────────────────────────────────────

// LogLevel 日志级别。
type LogLevel string

const (
	LogDebug   LogLevel = "debug"
	LogInfo    LogLevel = "info"
	LogWarning LogLevel = "warning"
	LogError   LogLevel = "error"
	LogPanic   LogLevel = "panic"
	LogBuild   LogLevel = "build"   // 构建错误
	LogDebugOp LogLevel = "debugop" // 调试操作（断点/单步等）
)

// LogEntry 一条日志记录。
type LogEntry struct {
	ID        string            `json:"id"`
	Time      string            `json:"time"`      // ISO 格式时间戳
	Level     LogLevel          `json:"level"`     // 日志级别
	Source    string            `json:"source"`    // 来源（如 "runOrchestrationLoop"、"handleChatSend"）
	Message   string            `json:"message"`   // 日志正文
	Context   map[string]string `json:"context,omitempty"` // 上下文（如 convID、sessionID、toolName、filePath）
	Stack     string            `json:"stack,omitempty"`   // panic 时的完整堆栈
	SessionID string            `json:"sessionId,omitempty"`
	ConvID    string            `json:"convId,omitempty"`
}

// DebugLogger 调试日志管理器（并发安全）。
type DebugLogger struct {
	mu         sync.Mutex
	logsDir    string // .pair/tasks/debug-logs/
	maxLogs    int    // 最大日志文件数（超过自动清理）
}

// NewDebugLogger 创建新的调试日志管理器。
// root 为工作区根目录；maxLogs 为最大日志文件数（默认 50）。
func NewDebugLogger(root string, maxLogs int) *DebugLogger {
	if maxLogs <= 0 {
		maxLogs = 50
	}
	dir := filepath.Join(root, ".pair", "tasks", "debug-logs")
	os.MkdirAll(dir, 0755)
	return &DebugLogger{
		logsDir: dir,
		maxLogs: maxLogs,
	}
}

// GlobalDebugLogger 全局调试日志实例（由 InitDebugLogger 初始化）。
var GlobalDebugLogger *DebugLogger

// InitDebugLogger 初始化全局调试日志管理器。
func InitDebugLogger(root string, maxLogs int) *DebugLogger {
	GlobalDebugLogger = NewDebugLogger(root, maxLogs)
	return GlobalDebugLogger
}

// ── 日志写入 ───────────────────────────────────────────────

// Log 写入一条日志。
func (dl *DebugLogger) Log(level LogLevel, source, message string, ctx map[string]string) {
	if dl == nil {
		return
	}
	dl.mu.Lock()
	defer dl.mu.Unlock()

	now := time.Now()
	id := fmt.Sprintf("log_%s_%d", now.Format("150405"), now.UnixMilli()%1000)
	entry := LogEntry{
		ID:      id,
		Time:    now.Format("2006-01-02T15:04:05.000Z07:00"),
		Level:   level,
		Source:  source,
		Message: message,
		Context: ctx,
	}
	if ctx != nil {
		if s, ok := ctx["sessionId"]; ok {
			entry.SessionID = s
		}
		if c, ok := ctx["convId"]; ok {
			entry.ConvID = c
		}
	}

	dl.writeEntryLocked(entry)
}

// LogError 简化：写入一条错误日志。
func (dl *DebugLogger) LogError(source, message string, ctx map[string]string) {
	dl.Log(LogError, source, message, ctx)
}

// LogBuildError 写入一条构建错误日志。
func (dl *DebugLogger) LogBuildError(source, output string, ctx map[string]string) {
	if ctx == nil {
		ctx = make(map[string]string)
	}
	ctx["buildOutput"] = output
	dl.Log(LogBuild, source, "项目构建失败", ctx)
}

// LogFromPanic 从 panic recovery 中写入日志（含堆栈）。
func (dl *DebugLogger) LogFromPanic(source string, r any, stack string, ctx map[string]string) {
	if dl == nil {
		return
	}
	dl.mu.Lock()
	defer dl.mu.Unlock()

	now := time.Now()
	id := fmt.Sprintf("panic_%s_%d", now.Format("150405"), now.UnixMilli()%1000)
	entry := LogEntry{
		ID:      id,
		Time:    now.Format("2006-01-02T15:04:05.000Z07:00"),
		Level:   LogPanic,
		Source:  source,
		Message: fmt.Sprintf("PANIC: %v", r),
		Stack:   stack,
		Context: ctx,
	}
	if ctx != nil {
		if s, ok := ctx["sessionId"]; ok {
			entry.SessionID = s
		}
		if c, ok := ctx["convId"]; ok {
			entry.ConvID = c
		}
	}

	dl.writeEntryLocked(entry)
}

func (dl *DebugLogger) writeEntryLocked(entry LogEntry) {
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return
	}
	filePath := filepath.Join(dl.logsDir, entry.ID+".json")
	os.WriteFile(filePath, data, 0644)

	// 写入后检查是否超量，超量则清理最旧的
	dl.cleanupOldLocked()
}

// ── 日志查询 ───────────────────────────────────────────────

// ListLogs 列出所有日志文件（按时间倒序）。limit 限制最大返回数（0=全部）。
func (dl *DebugLogger) ListLogs(limit int) []*LogEntry {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	entries, err := os.ReadDir(dl.logsDir)
	if err != nil {
		return nil
	}

	logs := make([]*LogEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		entry := dl.readEntryLocked(id)
		if entry != nil {
			logs = append(logs, entry)
		}
	}

	// 按时间倒序
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Time > logs[j].Time
	})

	if limit > 0 && len(logs) > limit {
		logs = logs[:limit]
	}
	return logs
}

// FilterLogs 按条件筛选日志。
func (dl *DebugLogger) FilterLogs(level LogLevel, source string, limit int) []*LogEntry {
	all := dl.ListLogs(limit)
	if all == nil {
		return nil
	}

	filtered := make([]*LogEntry, 0, len(all))
	for _, l := range all {
		if level != "" && l.Level != level {
			continue
		}
		if source != "" && !strings.Contains(l.Source, source) {
			continue
		}
		filtered = append(filtered, l)
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}

// GetLog 按 ID 读取单条日志。
func (dl *DebugLogger) GetLog(id string) *LogEntry {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	return dl.readEntryLocked(id)
}

// GetErrorSummary 获取错误摘要（最近 N 条 error/panic/build 级别的日志摘要）。
func (dl *DebugLogger) GetErrorSummary(n int) string {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	entries, err := os.ReadDir(dl.logsDir)
	if err != nil {
		return ""
	}

	var errors []*LogEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		entry := dl.readEntryLocked(id)
		if entry != nil && (entry.Level == LogError || entry.Level == LogPanic || entry.Level == LogBuild) {
			errors = append(errors, entry)
		}
	}

	sort.Slice(errors, func(i, j int) bool {
		return errors[i].Time > errors[j].Time
	})

	if n > 0 && len(errors) > n {
		errors = errors[:n]
	}

	if len(errors) == 0 {
		return "（无错误日志）"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("最近 %d 条错误日志:\n\n", len(errors)))
	for i, entry := range errors {
		sb.WriteString(fmt.Sprintf("--- [%d] %s ---\n", i+1, entry.Time))
		sb.WriteString(fmt.Sprintf("级别: %s | 来源: %s\n", entry.Level, entry.Source))
		sb.WriteString(fmt.Sprintf("消息: %s\n", entry.Message))
		if entry.Stack != "" {
			stackLines := strings.Split(entry.Stack, "\n")
			if len(stackLines) > 5 {
				stackLines = stackLines[:5]
			}
			sb.WriteString(fmt.Sprintf("堆栈: %s\n", strings.Join(stackLines, "\n  ")))
		}
		if entry.Context != nil {
			if bo, ok := entry.Context["buildOutput"]; ok {
				boLines := strings.Split(bo, "\n")
				maxLines := 15
				if len(boLines) > maxLines {
					boLines = append(boLines[:maxLines], "...（已截断）")
				}
				sb.WriteString(fmt.Sprintf("构建输出:\n  %s\n", strings.Join(boLines, "\n  ")))
			}
		}
		if i < len(errors)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// CountLogs 统计各级别日志数量。
func (dl *DebugLogger) CountLogs() map[LogLevel]int {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	entries, err := os.ReadDir(dl.logsDir)
	if err != nil {
		return nil
	}

	counts := make(map[LogLevel]int)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		entry := dl.readEntryLocked(id)
		if entry != nil {
			counts[entry.Level]++
		}
	}
	return counts
}

// ── 内部 ───────────────────────────────────────────────────

func (dl *DebugLogger) readEntryLocked(id string) *LogEntry {
	data, err := os.ReadFile(filepath.Join(dl.logsDir, id+".json"))
	if err != nil {
		return nil
	}
	var entry LogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}
	return &entry
}

// cleanupOldLocked 清理超量日志（保留最新的 maxLogs 条）。
func (dl *DebugLogger) cleanupOldLocked() {
	entries, err := os.ReadDir(dl.logsDir)
	if err != nil {
		return
	}

	if len(entries) <= dl.maxLogs {
		return
	}

	// 收集所有日志文件（按修改时间排序）
	type logFile struct {
		name    string
		modTime time.Time
	}
	var files []logFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, logFile{name: e.Name(), modTime: info.ModTime()})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	// 删除超量的最旧文件
	for i := dl.maxLogs; i < len(files); i++ {
		os.Remove(filepath.Join(dl.logsDir, files[i].name))
	}
}

// ── 便捷函数 ───────────────────────────────────────────────

// WriteDebug 快捷：写入 debug 日志。
func WriteDebug(source, message string, ctx map[string]string) {
	if GlobalDebugLogger != nil {
		GlobalDebugLogger.Log(LogDebug, source, message, ctx)
	}
}

// WriteError 快捷：写入 error 日志。
func WriteError(source, message string, ctx map[string]string) {
	if GlobalDebugLogger != nil {
		GlobalDebugLogger.Log(LogError, source, message, ctx)
	}
}

// WritePanic 快捷：从 panic 恢复中写入日志。
func WritePanic(source string, r any, stack string, ctx map[string]string) {
	if GlobalDebugLogger != nil {
		GlobalDebugLogger.LogFromPanic(source, r, stack, ctx)
	}
}

// WriteBuildError 快捷：写入构建错误日志。
func WriteBuildError(source, output string, ctx map[string]string) {
	if GlobalDebugLogger != nil {
		GlobalDebugLogger.LogBuildError(source, output, ctx)
	}
}
