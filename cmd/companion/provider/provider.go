// Package provider defines the model-backend abstraction and a registry mapping
// a provider "kind" to a factory. Concrete implementations live in subpackages
// (e.g. provider/openai) and self-register via init(). The core resolves
// providers by kind from config and never hardcodes a specific model.
//
// Ported from DeepSeek-Reasonix/internal/provider/ (Phase 3 upgrade).
package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Role is the role of a message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is a single conversation message.
type Message struct {
	Role               Role      `json:"role"`
	Content            string    `json:"content,omitempty"`
	ReasoningContent   string    `json:"reasoning_content,omitempty"`   // assistant: thinking-mode chain-of-thought
	ReasoningSignature string    `json:"reasoning_signature,omitempty"` // opaque proof for reasoning (Anthropic)
	ToolCalls          []ToolCall `json:"tool_calls,omitempty"`         // set by assistant
	ToolCallID         string    `json:"tool_call_id,omitempty"`        // links a tool result to its call
	Name               string    `json:"name,omitempty"`                // tool message: tool name
}

// ToolCall is a tool invocation requested by the model. Arguments is raw JSON.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolSchema is a tool definition exposed to the model. Parameters is JSON Schema.
type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Request is a single completion request.
type Request struct {
	Messages       []Message
	Tools          []ToolSchema
	Temperature    float64
	MaxTokens      int
	ThinkingMode   string // non-thinking / thinking / thinking_max; "" = no override
}

// interruptedToolResult stands in for a tool result that never landed.
const interruptedToolResult = "[no result: the previous turn was interrupted before this tool call completed]"

// SanitizeToolPairing repairs a history so it satisfies the tool-call contract
// that OpenAI-compatible APIs enforce: every assistant tool_calls entry must be
// answered by a following tool message, and orphan tool messages are dropped.
func SanitizeToolPairing(msgs []Message) []Message {
	out := make([]Message, 0, len(msgs))
	for i := 0; i < len(msgs); {
		m := msgs[i]
		if m.Role == RoleAssistant && len(m.ToolCalls) > 0 {
			j := i + 1
			for j < len(msgs) && msgs[j].Role == RoleTool {
				j++
			}
			out = append(out, repairToolCallArgs(m))
			out = append(out, pairToolResults(m.ToolCalls, msgs[i+1:j])...)
			i = j
			continue
		}
		if m.Role == RoleTool {
			i++ // orphan tool message — drop
			continue
		}
		out = append(out, m)
		i++
	}
	return out
}

func repairToolCallArgs(m Message) Message {
	broken := false
	for _, tc := range m.ToolCalls {
		if tc.Arguments != "" && !json.Valid([]byte(tc.Arguments)) {
			broken = true
			break
		}
	}
	if !broken {
		return m
	}
	calls := make([]ToolCall, len(m.ToolCalls))
	copy(calls, m.ToolCalls)
	for i := range calls {
		if calls[i].Arguments == "" || json.Valid([]byte(calls[i].Arguments)) {
			continue
		}
		calls[i].Arguments = closeTruncatedJSON(calls[i].Arguments)
	}
	m.ToolCalls = calls
	return m
}

func closeTruncatedJSON(s string) string {
	var stack []byte
	inStr, esc := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			stack = append(stack, '}')
		case '[':
			stack = append(stack, ']')
		case '}', ']':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	out := s
	if esc {
		out = out[:len(out)-1]
	}
	if inStr {
		out += `"`
	}
	trimmed := strings.TrimRight(out, " \t\r\n")
	switch {
	case strings.HasSuffix(trimmed, ","):
		out = trimmed[:len(trimmed)-1]
	case strings.HasSuffix(trimmed, ":"):
		out = trimmed + "null"
	}
	for i := len(stack) - 1; i >= 0; i-- {
		out += string(stack[i])
	}
	if !json.Valid([]byte(out)) {
		return "{}"
	}
	return out
}

func pairToolResults(calls []ToolCall, avail []Message) []Message {
	out := make([]Message, 0, len(calls))
	if idDistinct(calls) {
		byID := make(map[string]Message, len(avail))
		for _, r := range avail {
			byID[r.ToolCallID] = r
		}
		for _, tc := range calls {
			if r, ok := byID[tc.ID]; ok {
				out = append(out, r)
			} else {
				out = append(out, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Name, Content: interruptedToolResult})
			}
		}
		return out
	}
	for k, tc := range calls {
		if k < len(avail) {
			r := avail[k]
			r.ToolCallID = tc.ID
			out = append(out, r)
		} else {
			out = append(out, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Name, Content: interruptedToolResult})
		}
	}
	return out
}

func idDistinct(calls []ToolCall) bool {
	seen := make(map[string]struct{}, len(calls))
	for _, tc := range calls {
		if tc.ID == "" {
			return false
		}
		if _, dup := seen[tc.ID]; dup {
			return false
		}
		seen[tc.ID] = struct{}{}
	}
	return true
}

// ChunkType identifies the kind of a streamed increment.
type ChunkType int

const (
	ChunkText          ChunkType = iota
	ChunkReasoning
	ChunkToolCallStart
	ChunkToolCall
	ChunkUsage
	ChunkDone
	ChunkError
)

// Usage reports token accounting for a completion.
type Usage struct {
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CacheHitTokens   int    `json:"cache_hit_tokens"`
	CacheMissTokens  int    `json:"cache_miss_tokens"`
	ReasoningTokens  int    `json:"reasoning_tokens"`
	FinishReason     string `json:"finish_reason"`
}

// Pricing is a provider's per-1M-token rates.
type Pricing struct {
	CacheHit float64 `json:"cache_hit"`
	Input    float64 `json:"input"`
	Output   float64 `json:"output"`
	Currency string  `json:"currency"`
}

// Cost estimates the spend for a usage record.
func (p *Pricing) Cost(u *Usage) float64 {
	if p == nil || u == nil {
		return 0
	}
	return (float64(u.CacheHitTokens)*p.CacheHit +
		float64(u.CacheMissTokens)*p.Input +
		float64(u.CompletionTokens)*p.Output) / 1e6
}

// Symbol returns the currency display symbol, defaulting to "¥".
func (p *Pricing) Symbol() string {
	if p == nil || p.Currency == "" {
		return "¥"
	}
	return p.Currency
}

// Chunk is a single streamed event. Read the field matching Type.
type Chunk struct {
	Type      ChunkType
	Text      string    // ChunkText, ChunkReasoning
	Signature string    // ChunkReasoning: opaque proof (Anthropic thinking signature)
	ToolCall  *ToolCall // ChunkToolCallStart (ID+Name only), ChunkToolCall (complete)
	Usage     *Usage    // ChunkUsage
	Err       error     // ChunkError
}

// StreamInterruptedError marks a recoverable transport cut that happened after
// the caller had already received model output.
type StreamInterruptedError struct {
	Err error
}

func (e *StreamInterruptedError) Error() string {
	if e == nil || e.Err == nil {
		return "stream interrupted"
	}
	return e.Err.Error()
}

func (e *StreamInterruptedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func IsStreamInterrupted(err error) bool {
	var interrupted *StreamInterruptedError
	return errors.As(err, &interrupted)
}

// Provider is a chat-capable model backend.
type Provider interface {
	// Name returns the provider instance name, e.g. "deepseek" / "openai".
	Name() string
	// Stream starts a streaming completion, pushing increments on the channel.
	// Cancelling ctx must abort the underlying request; a closed channel marks
	// the end of the completion.
	Stream(ctx context.Context, req Request) (<-chan Chunk, error)
}

// Config is a resolved provider instance configuration.
type Config struct {
	Name    string         // instance name, e.g. "deepseek"
	BaseURL string         // OpenAI-compatible endpoint
	Model   string         // model id
	APIKey  string         // resolved from api_key_env
	Extra   map[string]any // kind-specific options
}

// AuthError reports that a provider rejected the API key (HTTP 401/403).
type AuthError struct {
	Provider string // the provider instance name
	KeyEnv   string // the environment variable the key is read from
	Status   int    // the HTTP status (401 or 403)
}

func (e *AuthError) Error() string {
	key := "the API key"
	if e.KeyEnv != "" {
		key = e.KeyEnv
	}
	return fmt.Sprintf("authentication failed for provider %q (HTTP %d): %s is invalid or expired",
		e.Provider, e.Status, key)
}

// APIError reports a non-OK HTTP status that isn't an auth failure.
type APIError struct {
	Provider string
	Status   int
	Body     string
}

func (e *APIError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("%s: status %d", e.Provider, e.Status)
	}
	return fmt.Sprintf("%s: status %d: %s", e.Provider, e.Status, e.Body)
}

// Factory builds a Provider from a resolved Config.
type Factory func(cfg Config) (Provider, error)

var registry = map[string]Factory{}

// Register adds a factory under a kind (e.g. "openai"). Intended for init().
// It panics on a duplicate kind.
func Register(kind string, f Factory) {
	if _, dup := registry[kind]; dup {
		panic("provider: duplicate kind " + kind)
	}
	registry[kind] = f
}

// New instantiates the provider of the given kind.
func New(kind string, cfg Config) (Provider, error) {
	f, ok := registry[kind]
	if !ok {
		return nil, fmt.Errorf("provider: unknown kind %q (registered: %v)", kind, Kinds())
	}
	p, err := f(cfg)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("provider: factory %q returned nil provider", kind)
	}
	return p, nil
}

// Kinds returns the registered kinds, sorted.
func Kinds() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
