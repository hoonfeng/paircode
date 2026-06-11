// Package agent 是「伴随式 CodeAgent」的 Go 引擎核心：TAOR 循环 + 工具 + LLM 适配。
// 复刻参考源 F:\syproject\伴随式codeagent\src\agent 的架构（只读参考，绝不修改它）。
//
// 设计：本包**纯 Go**（只用标准库 net/http、os、encoding/json 等），不依赖 goui/Skia，
// 故可独立单测（无 CGO/DLL）。GUI 接入在 cmd/companion 主包，由它驱动本引擎。
package agent

// Role 消息角色（OpenAI 兼容）。
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message 一条对话消息（OpenAI 兼容 + DeepSeek reasoning_content）。
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`  // assistant 请求的工具调用
	ToolCallID string     `json:"tool_call_id,omitempty"` // role=tool 时对应的调用 id
	Name       string     `json:"name,omitempty"`
	Reasoning  string     `json:"reasoning_content,omitempty"` // 思考链（DeepSeek，回传需保留）
}

// ToolCall LLM 请求的一次工具调用。
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // 恒为 "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall 工具调用的函数名 + JSON 字符串参数。
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串（LLM 流式拼接而成）
}

// ToolDefinition 传给 LLM 的工具定义（OpenAI function-calling schema）。
type ToolDefinition struct {
	Type     string             `json:"type"` // 恒为 "function"
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 工具的名/描述/参数（参数为 JSON Schema）。
type FunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema：{"type":"object","properties":{...},"required":[...]}
}

// Usage token 用量（含 DeepSeek KV 缓存命中）。
type Usage struct {
	PromptTokens        int `json:"prompt_tokens"`
	CompletionTokens    int `json:"completion_tokens"`
	TotalTokens         int `json:"total_tokens"`
	PromptCacheHitTokens int `json:"prompt_cache_hit_tokens,omitempty"`
}

// Chunk 流式输出的一片（content/reasoning/toolCalls 为增量）。
type Chunk struct {
	Content   string
	Reasoning string
	ToolCalls []ToolCall
	Done      bool
	Usage     *Usage
}
