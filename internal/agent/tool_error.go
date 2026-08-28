package agent

import "fmt"

// ToolError 统一工具错误类型。
// 携带可重试标记、修复建议和严重性级别，供 AfterTool 钩子做精细化决策。
type ToolError struct {
	Op         string // 失败操作名（如 "edit"）
	Message    string // 简短错误描述
	Suggestion string // 修复建议（如 "请使用 line_start/line_end 行号定位"）
	Severity   string // 严重性：warn/error/fatal
	Retryable  bool   // 是否重试可能成功
	Cause      error  // 原始错误（可选）
}

func (e *ToolError) Error() string {
	s := fmt.Sprintf("[%s/%s] %s", e.Severity, e.Op, e.Message)
	if e.Suggestion != "" {
		s += " — " + e.Suggestion
	}
	if e.Cause != nil {
		s += " | caused by: " + e.Cause.Error()
	}
	return s
}

func (e *ToolError) Unwrap() error { return e.Cause }

// NewToolError 创建工具错误。
func NewToolError(op, msg string, opts ...ToolErrorOption) *ToolError {
	e := &ToolError{Op: op, Message: msg, Severity: "error"}
	for _, o := range opts {
		o(e)
	}
	return e
}

// ToolErrorOption 工具错误可选配置。
type ToolErrorOption func(*ToolError)

func WithRetryable(v bool) ToolErrorOption    { return func(e *ToolError) { e.Retryable = v } }
func WithSuggestion(s string) ToolErrorOption { return func(e *ToolError) { e.Suggestion = s } }
func WithSeverity(s string) ToolErrorOption   { return func(e *ToolError) { e.Severity = s } }
func WithCause(c error) ToolErrorOption       { return func(e *ToolError) { e.Cause = c } }
