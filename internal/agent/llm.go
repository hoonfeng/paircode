package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
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

// ─── LLM HTTP 超时与重试配置（★ 2026-08-21 调优：拉长超时 + 流中断重试）──
// 包级变量：测试可临时改小；后续可经配置系统暴露给用户（settings 化）。
var (
	llmDialTimeout           = 30 * time.Second       // TCP 拨号超时
	llmTLSHandshakeTimeout   = 30 * time.Second       // TLS 握手超时（慢网络下 10s 不够）
	llmResponseHeaderTimeout = 180 * time.Second      // ★ 响应头超时（原 60s）：DeepSeek thinking 高峰排队/首 token 慢时 60s 不够
	llmClientTimeout         = 600 * time.Second      // 整个请求（含 SSE 流式读取）兜底超时
	llmMaxRetries            = 10                     // 最大重试次数（网络错误/超时/408/429/5xx）
	llmRetryBaseDelay        = 500 * time.Millisecond // 指数退避基数：0.5s,1s,2s,4s...
	llmRetryMaxDelay         = 30 * time.Second       // 退避上限
)

// OpenAIProvider OpenAI 兼容 /chat/completions 适配器。各家差异仅 BaseURL+Model+APIKey。
// SSE 流式：逐行解析 data:，累积 content/reasoning_content 与 tool_calls（按 index 拼 arguments）。
type OpenAIProvider struct {
	BaseURL      string // ★ 基础地址（如 https://api.deepseek.com/v1）；完整端点由 ResolveEndpointURL 按 Protocol 拼接
	Protocol     string // ★ 2026-09-02 LLM 协议（空=openai-completions）
	APIKey       string
	Model        string
	Temperature  float64      // <0 = 不下发（用服务端默认）；>=0 下发
	MaxTokens    int          // >0 时下发 max_tokens
	ThinkingMode string       // non-thinking/thinking/thinking_max；空=不下发思考参数（仅 DeepSeek V4 系生效）
	Multimodal   bool         // ★ 2026-08-21 多模态：模型支持图片输入 → 带 Images 的 user 消息转 content 块数组发送
	Client       *http.Client // nil → 默认客户端（超时见 llm* 变量）

	// OnRetry 重试通知回调（attempt 从 1 起，max 总次数）。loop 层绑定后可将
	// 「LLM 请求失败，正在自动重试」以 notice 事件推送到前端——避免用户干等。
	OnRetry func(attempt int, maxRetries int, errMsg string)

	// 默认客户端缓存：复用连接池/keep-alive，避免每次请求重建 Transport（每次重建 = DNS+TLS 全部重来）
	clientOnce  sync.Once
	clientCache *http.Client
}

// SetOnRetry 为 Provider 绑定重试通知（RetryNotifier 接口）。
func (p *OpenAIProvider) SetOnRetry(fn func(attempt, maxRetries int, errMsg string)) {
	p.OnRetry = fn
}

// RetryNotifier 可选接口：Provider 内部发生重试时通知调用方（用于 UI 展示重试进度）。
type RetryNotifier interface {
	SetOnRetry(func(attempt, maxRetries int, errMsg string))
}

// notifyRetry 触发重试通知（nil 安全）。
func (p *OpenAIProvider) notifyRetry(attempt int, errMsg string) {
	if p.OnRetry != nil {
		p.OnRetry(attempt, llmMaxRetries, errMsg)
	}
}

func (p *OpenAIProvider) Name() string { return "openai:" + p.Model }

// normalizeThinkingMode 把存储值归一为 OpenAI 思考档位枚举：
// none / minimal / low / medium / high / xhigh / max（OpenAI 官方 ReasoningEffort 定义）。
// ★ 兼容旧值：non-thinking→none（关闭）、thinking→high、thinking_max→max。
func normalizeThinkingMode(mode string) string {
	switch mode {
	case "":
		return "" // 未配置：不下发思考参数（用服务端默认）
	case "non-thinking", "none":
		return "none"
	case "minimal":
		return "minimal"
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "thinking", "high":
		return "high"
	case "xhigh":
		return "xhigh"
	case "thinking_max", "max":
		return "max"
	}
	return ""
}

// applyThinking 把思考档位下发到请求体（按 OpenAI reasoning_effort 档位分档）。
// DeepSeek V4 系模型（model 含 "v4"）：完整支持 thinking{enabled/disabled} + reasoning_effort。
// 非 v4 模型（如 OpenAI o-series、其他兼容模型）：仅下发 reasoning_effort（OpenAI 兼容），
// 不支持的服务端会忽略未知参数，安全无副作用。none 关闭思考（v4 显式 disabled；非 v4 不下发用服务端默认）。
func applyThinking(body map[string]any, model, mode string) {
	eff := normalizeThinkingMode(mode)
	if eff == "" {
		return // 未配置：不下发思考参数（用服务端默认行为）
	}
	if eff == "none" {
		// 显式关闭：v4 系下 thinking disabled；非 v4 不下发（用服务端默认行为）
		if strings.Contains(model, "v4") {
			body["thinking"] = map[string]any{"type": "disabled"}
		}
		return
	}
	// DeepSeek V4 系：thinking enabled + reasoning_effort 档位
	if strings.Contains(model, "v4") {
		body["thinking"] = map[string]any{"type": "enabled"}
	}
	// 非 v4 模型仅下发 reasoning_effort（OpenAI o-series / 兼容模型；不支持的服务端会忽略）
	body["reasoning_effort"] = eff
}

// buildOpenAIMessages 把内部 Message 列表转为 OpenAI 兼容请求消息。
// ★ 2026-08-21 多模态：multimodal=true 且消息带 Images 时，content 转块数组
//
//	[{type:'text',text}, {type:'image_url',image_url:{url,detail}}]；
//	否则保持字符串 content（兼容非视觉模型与历史消息）。
//
// 同时保留 tool_calls / tool_call_id / name / reasoning_content（DeepSeek 回传契约）。
func buildOpenAIMessages(msgs []Message, multimodal bool) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		mm := map[string]any{"role": string(m.Role)}
		if m.Role == RoleTool {
			mm["tool_call_id"] = m.ToolCallID
			mm["content"] = m.Content
		} else if multimodal && len(m.Images) > 0 {
			// 多模态：content 块数组（text + image_url）
			blocks := make([]map[string]any, 0, 1+len(m.Images))
			if m.Content != "" {
				blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
			}
			for _, img := range m.Images {
				iu := map[string]any{"url": img.Data}
				if img.Detail != "" {
					iu["detail"] = img.Detail
				}
				blocks = append(blocks, map[string]any{"type": "image_url", "image_url": iu})
			}
			mm["content"] = blocks
		} else {
			mm["content"] = m.Content
		}
		if len(m.ToolCalls) > 0 {
			tcs := make([]map[string]any, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				tcs = append(tcs, map[string]any{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			mm["tool_calls"] = tcs
		}
		if m.Name != "" {
			mm["name"] = m.Name
		}
		if m.Reasoning != "" {
			mm["reasoning_content"] = m.Reasoning
		}
		out = append(out, mm)
	}
	return out
}

// firstMime 返回图片列表首个 MIME（诊断日志用）。
func firstMime(imgs []ImagePart) string {
	if len(imgs) == 0 {
		return ""
	}
	return imgs[0].MimeType
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

// interruptedToolResult 作为缺失 tool result 的占位（维持 OpenAI 兼容 API 的配对契约：
// tool_call 后必须有对应 role=tool 消息）。内容为空——不再向模型注入「未完成/中断」
// 提示（历史教训：该提示会干扰模型判断、诱导核对/重试），空结果即「无信息」。
const interruptedToolResult = ""

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
	p.clientOnce.Do(func() {
		p.clientCache = defaultLLMClient() // ★ 2026-09-02 提取共享（provider_http.go）
	})
	return p.clientCache
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	// ★ 修复消息中的 tool 调用配对，确保满足 OpenAI API 的 role=tool 契约
	messages = sanitizeToolPairing(messages)

	// ★ 2026-08-21 多模态诊断：确认装配器 multimodal 标记与图片消息进入请求体
	for _, m := range messages {
		if len(m.Images) > 0 {
			log.Printf("[llm] 多模态消息: role=%s images=%d multimodal=%v mime=%s",
				m.Role, len(m.Images), p.Multimodal, firstMime(m.Images))
		}
	}

	body := map[string]any{
		"model":          p.Model,
		"messages":       buildOpenAIMessages(messages, p.Multimodal),
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

	// ★ 2026-08-27/2026-09-02：URL = 基础地址 + 协议路径（ResolveEndpointURL）——
	//   配置只填 base（如 https://x/v1），/chat/completions 由内部按 Protocol 拼接；
	//   BaseURL 已含协议路径后缀（旧「完整端点」配置）时直接使用，不重复拼接。
	url := ResolveEndpointURL(p.BaseURL, p.Protocol)

	maxRetries := llmMaxRetries
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避 + 抖动：0.5s, 1s, 2s, 4s, 8s, 16s, 30s, 30s...
			delay := time.Duration(1<<(attempt-1)) * llmRetryBaseDelay
			if delay > llmRetryMaxDelay {
				delay = llmRetryMaxDelay
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
			log.Printf("[llm] %s 网络请求失败（第%d次，退避后重试）: %v", p.Model, attempt+1, err)
			p.notifyRetry(attempt+1, err.Error())
			continue
		}

		if resp.StatusCode == http.StatusOK {
			msg, perr := parseSSE(resp.Body, onChunk)
			resp.Body.Close()
			if perr == nil {
				return msg, nil
			}
			// ★ 2026-08-21 SSE 流中断重试策略：
			//   未产出任何内容（网络瞬断/读超时/首帧前断）→ 安全重试；
			//   已产出内容 → 不自动重试（重试会重复输出；若流中含 tool_calls 还会导致工具重复执行），
			//   带已累积内容返回错误，由上层决定后续。
			if msg.Content != "" || msg.Reasoning != "" || len(msg.ToolCalls) > 0 {
				return msg, fmt.Errorf("LLM 流式响应中断（已接收内容 %d 字符，不自动重试以免重复）: %w",
					len(msg.Content)+len(msg.Reasoning), perr)
			}
			lastErr = fmt.Errorf("LLM 流式读取失败 (第%d次): %w", attempt+1, perr)
			log.Printf("[llm] %s 流式读取失败（第%d次，退避后重试）: %v", p.Model, attempt+1, perr)
			p.notifyRetry(attempt+1, perr.Error())
			continue
		}

		// 处理非 200 状态码
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()

		statusCode := resp.StatusCode
		bodyStr := strings.TrimSpace(string(b))

		// 401/403 → 认证错误，不重试
		if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
			err := fmt.Errorf("LLM HTTP %d (认证失败): %s", statusCode, bodyStr)
			log.Printf("[llm] %s 认证失败（不重试）: %v", p.Model, err)
			return Message{}, err
		}

		// 可重试状态码：408（超时）、429（限流）、5xx（服务端错误）
		if statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests || (statusCode >= 500 && statusCode <= 599) {
			lastErr = fmt.Errorf("LLM HTTP %d (第%d次): %s", statusCode, attempt+1, bodyStr)
			log.Printf("[llm] %s HTTP %d（第%d次，退避后重试）: %s", p.Model, statusCode, attempt+1, bodyStr)
			p.notifyRetry(attempt+1, fmt.Sprintf("HTTP %d: %s", statusCode, bodyStr))
			continue
		}

		// 其他 4xx → 客户端错误，不重试
		err = fmt.Errorf("LLM HTTP %d: %s", statusCode, bodyStr)
		log.Printf("[llm] %s HTTP %d（客户端错误，不重试）: %s", p.Model, statusCode, bodyStr)
		emitBridgeEvent("agent/request-error", map[string]any{"model": p.Model, "error": err.Error()})
		return Message{}, err
	}

	log.Printf("[llm] %s 请求失败（已达最大重试次数 %d）: %v", p.Model, maxRetries, lastErr)
	emitBridgeEvent("agent/request-error", map[string]any{"model": p.Model, "error": lastErr.Error()})
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
	var stopReason string // 最后一片的 finish_reason
	sawDone := false      // 收到 [DONE] 结束标记
	sawFinish := false    // 收到 finish_reason（部分实现不发 [DONE] 但发 finish_reason）

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if !strings.HasPrefix(line, "data:") {
			continue // 跳过空行 / event: 行 / 注释
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			sawDone = true
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
		// 捕获 finish_reason（如 length=截断）
		if frame.Choices[0].FinishReason != nil {
			sawFinish = true
			stopReason = *frame.Choices[0].FinishReason
		}
		if onChunk != nil && (d.Content != "" || d.Reasoning != "") {
			onChunk(Chunk{Content: d.Content, Reasoning: d.Reasoning})
		}
	}
	msg := Message{Role: RoleAssistant, Content: content.String(), Reasoning: reasoning.String()}
	for _, idx := range toolOrder {
		msg.ToolCalls = append(msg.ToolCalls, *toolAccum[idx])
	}
	if err := sc.Err(); err != nil {
		// ★ 出错时返回已累积内容：供上层判断「是否已产出」→ 决定是否可安全重试
		return msg, fmt.Errorf("读取 SSE 流失败: %w", err)
	}
	// ★ 完成性检测：流以 EOF 结束但既无 [DONE] 也无 finish_reason → 视为提前截断
	//   （服务器/TCP 正常关闭但未发结束标记，sc.Err() 为 nil 但内容不完整——静默截断）。
	//   返回错误携带已累积内容：上层据此决定重试（未产出）或不重试（已产出）。
	if !sawDone && !sawFinish {
		return msg, fmt.Errorf("SSE 流提前结束（未收到 [DONE]/finish_reason 完成标记，已累积 %d 字符）",
			len(msg.Content)+len(msg.Reasoning))
	}
	if onChunk != nil {
		onChunk(Chunk{Done: true, Usage: usage, StopReason: stopReason})
	}
	return msg, nil
}
