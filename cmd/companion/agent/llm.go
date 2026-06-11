package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Provider LLM 提供方抽象。Chat 发起一次（可流式）对话：content/reasoning 增量经 onChunk 回调，
// 最终组装好的 assistant Message（含 tool_calls）作返回值。多 LLM 适配即实现本接口。
type Provider interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error)
	Name() string
}

// ─── Mock 提供方（测试/离线用）────────────────────────────────

// MockProvider 脚本化提供方：按 Responses 顺序每次 Chat 返回下一条（用于无网络端到端测 TAOR 循环）。
type MockProvider struct {
	Responses []Message
	calls     int
}

func (m *MockProvider) Name() string { return "mock" }

// Calls 已被调用次数（测试断言用）。
func (m *MockProvider) Calls() int { return m.calls }

func (m *MockProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	var msg Message
	if m.calls < len(m.Responses) {
		msg = m.Responses[m.calls]
	} else {
		msg = Message{Role: RoleAssistant, Content: "[FINAL]"} // 脚本耗尽兜底：结束循环，防越界
	}
	m.calls++
	if msg.Role == "" {
		msg.Role = RoleAssistant
	}
	if onChunk != nil {
		onChunk(Chunk{Content: msg.Content, Reasoning: msg.Reasoning, ToolCalls: msg.ToolCalls, Done: true})
	}
	return msg, nil
}

// ─── OpenAI 兼容适配器（DeepSeek / OpenAI / Qwen / Moonshot…）──

// OpenAIProvider OpenAI 兼容 /chat/completions 适配器。各家差异仅 BaseURL+Model+APIKey。
// SSE 流式：逐行解析 data:，累积 content/reasoning_content 与 tool_calls（按 index 拼 arguments）。
type OpenAIProvider struct {
	BaseURL      string // 如 https://api.deepseek.com/v1（不含 /chat/completions）
	APIKey       string
	Model        string
	Temperature  float64      // <0 = 不下发（用服务端默认）；>=0 下发
	MaxTokens    int          // >0 时下发 max_tokens
	ThinkingMode string       // non-thinking/thinking/thinking_max；空=不下发思考参数（仅 DeepSeek V4 系生效）
	Client       *http.Client // nil → 默认 120s 超时
}

func (p *OpenAIProvider) Name() string { return "openai:" + p.Model }

// applyThinking 把思考模式下发到请求体——1:1 复刻参考源 llm/adapter.ts：
// 仅对 DeepSeek V4 系模型（model 含 "v4"）生效；非 v4 模型不带思考参数（避免被服务端拒绝）。
// 非 non-thinking → thinking{enabled} + reasoning_effort(high|max)；non-thinking → thinking{disabled}。
func applyThinking(body map[string]any, model, mode string) {
	if mode == "" || !strings.Contains(model, "v4") {
		return
	}
	if mode == "non-thinking" {
		body["thinking"] = map[string]any{"type": "disabled"}
		return
	}
	body["thinking"] = map[string]any{"type": "enabled"}
	eff := "high"
	if mode == "thinking_max" {
		eff = "max"
	}
	body["reasoning_effort"] = eff
}

func (p *OpenAIProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return &http.Client{Timeout: 120 * time.Second}
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	body := map[string]any{
		"model":          p.Model,
		"messages":       messages,
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	if len(tools) > 0 {
		body["tools"] = tools
	}
	if p.Temperature >= 0 {
		body["temperature"] = p.Temperature
	}
	if p.MaxTokens > 0 {
		body["max_tokens"] = p.MaxTokens
	}
	applyThinking(body, p.Model, p.ThinkingMode)
	buf, err := json.Marshal(body)
	if err != nil {
		return Message{}, err
	}
	url := strings.TrimRight(p.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client().Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Message{}, fmt.Errorf("LLM HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return parseSSE(resp.Body, onChunk)
}

// sseResp 是 SSE 每帧的解析目标（OpenAI 流式 chunk 结构）。
type sseResp struct {
	Choices []struct {
		Delta struct {
			Content   string         `json:"content"`
			Reasoning string         `json:"reasoning_content"`
			ToolCalls []sseToolDelta `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *Usage `json:"usage"`
}

type sseToolDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// parseSSE 解析 OpenAI 兼容 SSE 流，累积成最终 assistant Message。可独立测（喂 io.Reader）。
func parseSSE(r io.Reader, onChunk func(Chunk)) (Message, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // 容大帧

	var content, reasoning strings.Builder
	toolAccum := map[int]*ToolCall{}
	var toolOrder []int
	var usage *Usage

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if !strings.HasPrefix(line, "data:") {
			continue // 跳过空行 / event: 行 / 注释
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var frame sseResp
		if err := json.Unmarshal([]byte(data), &frame); err != nil {
			continue // 容错：忽略坏帧
		}
		if frame.Usage != nil {
			usage = frame.Usage
		}
		if len(frame.Choices) == 0 {
			continue
		}
		d := frame.Choices[0].Delta
		if d.Content != "" {
			content.WriteString(d.Content)
		}
		if d.Reasoning != "" {
			reasoning.WriteString(d.Reasoning)
		}
		for _, tc := range d.ToolCalls {
			acc, ok := toolAccum[tc.Index]
			if !ok {
				acc = &ToolCall{Type: "function"}
				toolAccum[tc.Index] = acc
				toolOrder = append(toolOrder, tc.Index)
			}
			if tc.ID != "" {
				acc.ID = tc.ID
			}
			if tc.Type != "" {
				acc.Type = tc.Type
			}
			if tc.Function.Name != "" {
				acc.Function.Name = tc.Function.Name
			}
			acc.Function.Arguments += tc.Function.Arguments
		}
		if onChunk != nil && (d.Content != "" || d.Reasoning != "") {
			onChunk(Chunk{Content: d.Content, Reasoning: d.Reasoning})
		}
	}
	if err := sc.Err(); err != nil {
		return Message{}, fmt.Errorf("读取 SSE 流失败: %w", err)
	}

	msg := Message{Role: RoleAssistant, Content: content.String(), Reasoning: reasoning.String()}
	for _, idx := range toolOrder {
		msg.ToolCalls = append(msg.ToolCalls, *toolAccum[idx])
	}
	if onChunk != nil {
		onChunk(Chunk{Done: true, Usage: usage})
	}
	return msg, nil
}
