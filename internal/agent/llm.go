package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
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
		msg = Message{Role: RoleAssistant, Content: "完成"} // 脚本耗尽兜底：结束循环，防越界
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

// applyThinking 把思考模式下发到请求体。
// DeepSeek V4 系模型（model 含 "v4"）：完整支持 thinking{enabled/disabled} + reasoning_effort。
// 非 v4 模型（如 OpenAI o-series、其他兼容模型）：仅下发 reasoning_effort（OpenAI 兼容），
// 不支持的服务端会忽略未知参数，安全无副作用。non-thinking 对非 v4 不下发（用服务端默认）。
func applyThinking(body map[string]any, model, mode string) {
	if mode == "" {
		return
	}
	// DeepSeek V4 系：完整 thinking + reasoning_effort 支持
	if strings.Contains(model, "v4") {
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
		return
	}
	// 非 v4 模型：尝试 reasoning_effort（OpenAI o-series / 兼容模型）
	// 仅对 thinking/thinking_max 下发；non-thinking 不下发（用服务端默认行为）
	if mode == "thinking" || mode == "thinking_max" {
		eff := "high"
		if mode == "thinking_max" {
			eff = "max"
		}
		body["reasoning_effort"] = eff
	}
}

// sanitizeToolPairing 修复消息列表中的工具调用配对，确保满足 OpenAI 兼容 API 的契约：
// 每条 assistant 消息的 tool_calls 必须后跟对应数量的 role=tool 消息，
// 孤立 role=tool 消息将被丢弃，缺失 tool result 将被填充占位符。
//
// ★ 防御性去重：跨 assistant 跟踪已配对的 tool_call_id。
// 如果某段 assistant 的全部 tool_call ID 都已在前面的配对中出现过，
// 说明该段是历史消息重复段，直接跳过——防止 "Duplicate value for 'tool_call_id'" 错误。
func sanitizeToolPairing(msgs []Message) []Message {
	out := make([]Message, 0, len(msgs))
	seenCallIDs := make(map[string]struct{}) // 跨 assistant 去重

	for i := 0; i < len(msgs); {
		m := msgs[i]
		if m.Role == RoleAssistant && len(m.ToolCalls) > 0 {
			// ★ 检查此 assistant 的所有 tool_call ID 是否都已处理过
			allSeen := true
			for _, tc := range m.ToolCalls {
				if _, ok := seenCallIDs[tc.ID]; !ok {
					allSeen = false
					break
				}
			}
			if allSeen {
				// 全部 ID 都已在前面的配对段中出现 → 跳过整个段（含后续连续 tool result）
				j := i + 1
				for j < len(msgs) && msgs[j].Role == RoleTool {
					j++
				}
				i = j
				continue
			}

			// 收集后续连续的 role=tool 消息
			j := i + 1
			for j < len(msgs) && msgs[j].Role == RoleTool {
				j++
			}
			// 配对：按 ID 匹配或按顺序匹配
			avail := msgs[i+1 : j]
			out = append(out, m)
			out = append(out, pairToolResults(m.ToolCalls, avail)...)
			// 记录已配对的 tool_call_id
			for _, tc := range m.ToolCalls {
				seenCallIDs[tc.ID] = struct{}{}
			}
			i = j
			continue
		}
		if m.Role == RoleTool {
			i++ // 孤立 role=tool 消息，丢弃
			continue
		}
		out = append(out, m)
		i++
	}
	return out
}

const interruptedToolResult = "[no result: the previous turn was interrupted before this tool call completed]"

// pairToolResults 将 tool call 与 tool result 配对。
// 优先按 ToolCall.ID 匹配；ID 不全或重复则按顺序匹配。
func pairToolResults(calls []ToolCall, avail []Message) []Message {
	out := make([]Message, 0, len(calls))
	// 检查所有 ToolCall 的 ID 是否唯一且非空
	idsDistinct := true
	seen := make(map[string]struct{}, len(calls))
	for _, tc := range calls {
		if tc.ID == "" {
			idsDistinct = false
			break
		}
		if _, dup := seen[tc.ID]; dup {
			idsDistinct = false
			break
		}
		seen[tc.ID] = struct{}{}
	}
	if idsDistinct {
		// 按 ID 匹配
		byID := make(map[string]Message, len(avail))
		for _, r := range avail {
			byID[r.ToolCallID] = r
		}
		for _, tc := range calls {
			if r, ok := byID[tc.ID]; ok {
				out = append(out, r)
			} else {
				out = append(out, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: interruptedToolResult})
			}
		}
		return out
	}
	// ID 不全或重复，按顺序匹配
	for k, tc := range calls {
		if k < len(avail) {
			r := avail[k]
			r.ToolCallID = tc.ID
			out = append(out, r)
		} else {
			out = append(out, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: interruptedToolResult})
		}
	}
	return out
}

func (p *OpenAIProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second, // 服务器须在 60s 内返回响应头
		},
		Timeout: 600 * time.Second, // SSE 流式读取：10 分钟兜底，覆盖长推理场景
	}
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	// ★ 修复消息中的 tool 调用配对，确保满足 OpenAI API 的 role=tool 契约
	messages = sanitizeToolPairing(messages)

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

	const maxRetries = 10
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避 + 抖动：0.5s, 1s, 2s, 4s, 8s, 16s, 30s, 30s...
			delay := time.Duration(500*(1<<(attempt-1))) * time.Millisecond
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			delay += time.Duration(rand.Intn(250)) * time.Millisecond
			select {
			case <-ctx.Done():
				return Message{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
		if err != nil {
			return Message{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
		req.Header.Set("Accept", "text/event-stream")

		resp, err := p.client().Do(req)
		if err != nil {
			// 网络级错误（连接重置、DNS 失败等）→ 可重试
			// 但 context 取消/超时不重试
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return Message{}, err
			}
			lastErr = fmt.Errorf("LLM 请求失败 (第%d次): %w", attempt+1, err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return parseSSE(resp.Body, onChunk)
		}

		// 处理非 200 状态码
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()

		statusCode := resp.StatusCode
		bodyStr := strings.TrimSpace(string(b))

		// 401/403 → 认证错误，不重试
		if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
			return Message{}, fmt.Errorf("LLM HTTP %d (认证失败): %s", statusCode, bodyStr)
		}

		// 可重试状态码：408（超时）、429（限流）、5xx（服务端错误）
		if statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests || (statusCode >= 500 && statusCode <= 599) {
			lastErr = fmt.Errorf("LLM HTTP %d (第%d次): %s", statusCode, attempt+1, bodyStr)
			continue
		}

		// 其他 4xx → 客户端错误，不重试
		return Message{}, fmt.Errorf("LLM HTTP %d: %s", statusCode, bodyStr)
	}

	return Message{}, fmt.Errorf("LLM 请求失败（已达最大重试次数 %d）: %w", maxRetries, lastErr)
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
