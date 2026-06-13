// Package debugger 提供 DAP (Debug Adapter Protocol) 调试器支持。
// 通过 dlv dap 启动 Delve 调试服务器，使用 DAP JSON-RPC 协议通信。
// 支持断点管理、单步执行、变量查看、栈帧跟踪、表达式求值等调试功能。
package debugger

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ─── DAP 协议基础类型 ────────────────────────────────────────────

// MessageType DAP 消息类型。
type MessageType string

const (
	MsgRequest  MessageType = "request"
	MsgResponse MessageType = "response"
	MsgEvent    MessageType = "event"
)

// Message DAP 基础消息结构。
type Message struct {
	Seq        int64            `json:"seq"`
	Type       MessageType      `json:"type"`
	Command    string           `json:"command,omitempty"`
	RequestSeq int64            `json:"request_seq,omitempty"`
	Success    bool             `json:"success,omitempty"`
	Event      string           `json:"event,omitempty"`
	Body       json.RawMessage `json:"body,omitempty"`
	Message    string           `json:"message,omitempty"`
}

// Request 构造 DAP 请求。
func Request(command string, args any) Message {
	body, _ := json.Marshal(args)
	return Message{
		Type:    MsgRequest,
		Command: command,
		Body:    body,
	}
}

// ─── DAP 请求参数类型 ─────────────────────────────────────────────

// InitializeRequest DAP initialize 请求参数。
type InitializeRequest struct {
	ClientID                string `json:"clientID,omitempty"`
	ClientName              string `json:"clientName,omitempty"`
	AdapterID               string `json:"adapterID"`
	Locale                  string `json:"locale,omitempty"`
	LinesStartAt1           bool   `json:"linesStartAt1"`
	ColumnsStartAt1         bool   `json:"columnsStartAt1"`
	PathFormat              string `json:"pathFormat,omitempty"`
	SupportsVariableType    bool   `json:"supportsVariableType,omitempty"`
	SupportsVariablePaging  bool   `json:"supportsVariablePaging,omitempty"`
	SupportsRunInTerminal   bool   `json:"supportsRunInTerminal,omitempty"`
	SupportsMemoryReferences bool  `json:"supportsMemoryReferences,omitempty"`
}

// LaunchRequest DAP launch 请求参数（用于 dlv dap）。
type LaunchRequest struct {
	Program       string `json:"program"`
	Args          []string `json:"args,omitempty"`
	Cwd           string `json:"cwd,omitempty"`
	BuildFlags    string `json:"buildFlags,omitempty"`
	NoDebug       bool   `json:"noDebug,omitempty"`
	StopOnEntry   bool   `json:"stopOnEntry,omitempty"`
	ShowRegisters bool   `json:"showRegisters,omitempty"`
	ShowVariables string `json:"showVariables,omitempty"`
}

// SetBreakpointsRequest DAP setBreakpoints 请求参数。
type SetBreakpointsRequest struct {
	Source        Source        `json:"source"`
	Lines         []int         `json:"lines"`
	Breakpoints   []Breakpoint  `json:"breakpoints,omitempty"`
	SourceModified bool         `json:"sourceModified,omitempty"`
}

// Breakpoint DAP 断点定义。
type Breakpoint struct {
	ID       int    `json:"id,omitempty"`
	Line     int    `json:"line"`
	Column   int    `json:"column,omitempty"`
	Verified bool   `json:"verified,omitempty"`
	Message  string `json:"message,omitempty"`
	Source   Source `json:"source,omitempty"`
}

// Source DAP 源文件信息。
type Source struct {
	Name        string `json:"name,omitempty"`
	Path        string `json:"path,omitempty"`
	SourceRef   int    `json:"sourceReference,omitempty"`
	PresentationHint string `json:"presentationHint,omitempty"`
}

// StackTraceRequest DAP stackTrace 请求参数。
type StackTraceRequest struct {
	ThreadID   int `json:"threadId"`
	StartFrame int `json:"startFrame,omitempty"`
	Levels     int `json:"levels,omitempty"`
}

// StackFrame DAP 栈帧。
type StackFrame struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Source        Source `json:"source,omitempty"`
	Line          int    `json:"line"`
	Column        int    `json:"column"`
	EndLine       int    `json:"endLine,omitempty"`
	EndColumn     int    `json:"endColumn,omitempty"`
	InstructionPtrRef int `json:"instructionPointerReference,omitempty"`
}

// VariablesRequest DAP variables 请求参数。
type VariablesRequest struct {
	VariablesReference int `json:"variablesReference"`
	Start              int `json:"start,omitempty"`
	Count              int `json:"count,omitempty"`
}

// Variable DAP 变量。
type Variable struct {
	Name               string `json:"name"`
	Value              string `json:"value"`
	Type               string `json:"type,omitempty"`
	VariablesReference int    `json:"variablesReference"`
	NamedVariables     int    `json:"namedVariables,omitempty"`
	IndexedVariables   int    `json:"indexedVariables,omitempty"`
}

// EvaluateRequest DAP evaluate 请求参数。
type EvaluateRequest struct {
	Expression string `json:"expression"`
	FrameID    int    `json:"frameId,omitempty"`
	Context    string `json:"context,omitempty"`
}

// EvaluateResponse DAP evaluate 响应体。
type EvaluateResponse struct {
	Result             string `json:"result"`
	Type               string `json:"type,omitempty"`
	VariablesReference int    `json:"variablesReference"`
	NamedVariables     int    `json:"namedVariables,omitempty"`
	IndexedVariables   int    `json:"indexedVariables,omitempty"`
}

// NextRequest DAP next/stepOver 请求参数。
type NextRequest struct {
	ThreadID int `json:"threadId"`
}

// ContinueRequest DAP continue 请求参数。
type ContinueRequest struct {
	ThreadID int `json:"threadId"`
}

// Thread DAP 线程信息。
type Thread struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ─── DAP 事件类型 ─────────────────────────────────────────────────

// StoppedEventBody DAP stopped 事件体。
type StoppedEventBody struct {
	Reason       string `json:"reason"`
	Description  string `json:"description,omitempty"`
	ThreadID     int    `json:"threadId,omitempty"`
	Text         string `json:"text,omitempty"`
	AllThreadsStopped bool `json:"allThreadsStopped,omitempty"`
}

// ContinuedEventBody DAP continued 事件体。
type ContinuedEventBody struct {
	ThreadID       int  `json:"threadId"`
	AllThreadsContinued bool `json:"allThreadsContinued,omitempty"`
}

// ExitedEventBody DAP exited 事件体。
type ExitedEventBody struct {
	ExitCode int `json:"exitCode"`
}

// OutputEventBody DAP output 事件体（程序输出）。
type OutputEventBody struct {
	Category string `json:"category"` // stdout, stderr, console
	Output   string `json:"output"`
}

// ─── 调试会话状态 ─────────────────────────────────────────────────

// SessionState 调试会话状态。
type SessionState string

const (
	StateIdle       SessionState = "idle"       // 未启动
	StateInitialized SessionState = "initialized" // 已初始化
	StateRunning    SessionState = "running"    // 程序运行中
	StatePaused     SessionState = "paused"     // 已暂停（断点/单步命中）
	StateExited     SessionState = "exited"     // 程序已退出
	StateError      SessionState = "error"      // 错误状态
)

// StopReason 停止原因。
type StopReason string

const (
	StopBreakpoint  StopReason = "breakpoint"
	StopStep        StopReason = "step"
	StopPause       StopReason = "pause"
	StopException   StopReason = "exception"
	StopEntry       StopReason = "entry"
)

// DebugSession 调试会话，管理一次 dlv dap 调试生命周期。
type DebugSession struct {
	mu          sync.Mutex
	state       SessionState
	dlvCmd      string // dlv 命令路径
	program     string // 被调试的程序路径
	host        string
	port        int
	conn        *dapConn
	nextSeq     int64
	breakpoints []Breakpoint
	threads     []Thread
	stopped     bool
	stopReason  StopReason
	lastBody    json.RawMessage // 最近一次 stopped 事件的 body
	createdAt   time.Time
}

// NewDebugSession 创建新的调试会话。
// dlvCmd: dlv 命令路径（如 "dlv"）；program: 要调试的 Go 程序路径。
func NewDebugSession(dlvCmd, program string) *DebugSession {
	return &DebugSession{
		state:     StateIdle,
		dlvCmd:    dlvCmd,
		program:   program,
		host:      "127.0.0.1",
		port:      0, // 自动选择
		nextSeq:   1,
		createdAt: time.Now(),
	}
}

// State 返回当前会话状态。
func (s *DebugSession) State() SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// IsActive 会话是否活跃（正在调试或暂停中）。
func (s *DebugSession) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == StateRunning || s.state == StatePaused || s.state == StateInitialized
}

// Port 返回 dlv dap 监听的端口。
func (s *DebugSession) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

// Breakpoints 返回当前断点列表。
func (s *DebugSession) Breakpoints() []Breakpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Breakpoint, len(s.breakpoints))
	copy(out, s.breakpoints)
	return out
}

// Scope DAP 作用域信息。
type Scope struct {
	Name               string `json:"name"`
	VariablesReference int    `json:"variablesReference"`
	Expensive          bool   `json:"expensive,omitempty"`
	NamedVariables     int    `json:"namedVariables,omitempty"`
	IndexedVariables   int    `json:"indexedVariables,omitempty"`
}

// ScopesRequest DAP scopes 请求参数。
type ScopesRequest struct {
	FrameID int `json:"frameId"`
}

// Program 返回被调试的程序路径。
func (s *DebugSession) Program() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.program
}

// FormatStopReason 格式化调试会话的停止原因描述（导出函数）。
func FormatStopReason(s *DebugSession) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return formatStoppedReason(s.stopReason)
}

// formatStoppedReason 格式化停止原因描述。
func formatStoppedReason(reason StopReason) string {
	switch reason {
	case StopBreakpoint:
		return "断点命中"
	case StopStep:
		return "单步完成"
	case StopPause:
		return "已暂停"
	case StopException:
		return "异常/panic"
	case StopEntry:
		return "程序入口"
	default:
		return string(reason)
	}
}

// formatState 格式化会话状态描述。
func formatState(s SessionState) string {
	switch s {
	case StateIdle:
		return "未启动"
	case StateInitialized:
		return "已初始化"
	case StateRunning:
		return "运行中"
	case StatePaused:
		return "已暂停"
	case StateExited:
		return "已退出"
	case StateError:
		return "错误"
	default:
		return string(s)
	}
}

// ─── 辅助 ─────────────────────────────────────────────────────────

// findPort 辅助：在发布时解析错误中的端口号。
func extractPortFromOutput(out string) (int, error) {
	// dlv dap 输出格式: "DAP server listening at: 127.0.0.1:12345"
	_, err := fmt.Sscanf(out, "DAP server listening at: 127.0.0.1:%d", &struct{ *int }{})
	_ = err
	// 简单解析
	const prefix = "127.0.0.1:"
	for i := 0; i < len(out); i++ {
		if i+len(prefix) <= len(out) && out[i:i+len(prefix)] == prefix {
			var port int
			if _, err := fmt.Sscanf(out[i:], "127.0.0.1:%d", &port); err == nil && port > 0 {
				return port, nil
			}
		}
	}
	return 0, fmt.Errorf("未能在输出中找到端口号: %s", out)
}
