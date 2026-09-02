// provider_responses.go — OpenAI Responses API 适配器（★ 2026-09-02）
//
// 协议：POST {base}/v1/responses，Bearer 认证。与 /chat/completions 的核心差异：
//   - 系统指令为顶层 instructions 字段
//   - input 为「消息/事件混合数组」：function_call_output 是顶层元素（非 role=tool 消息）
//   - 工具的 function_call 参数为顶层 item（call_id 关联输出）
//   - SSE 事件流：response.created → output_item.added → content_part.added →
//     output_text.delta / function_call_arguments.delta → ...done → response.completed
//   - max_output_tokens（而非 max_tokens）；reasoning 经 include 请求
//
// 参考：ref/deepseek-harness/packages/subagent/subagent-codex/tests/responses-fixture.ts
// （Codex 0.147.0 消费的最小事件序列，此处按同样结构解析）。

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

// ResponsesProvider OpenAI Responses API 流式适配器。
// 差异仅 BaseURL+Model+APIKey；内部消息/工具/图片自动转换为 Responses 形态。
type ResponsesProvider struct {
	BaseURL      string // 基础地址（如 https://api.openai.com/v1）；完整端点由 ResolveEndpointURL 拼接
	Protocol     string // ★ 恒为 openai-responses（可空）
	APIKey       string
	Model        string
	Temperature  float64      // <0 = 不下发（用服务端默认）；>=0 下发
	MaxTokens    int          // >0 时下发 max_output_tokens
	ThinkingMode string       // 映射 reasoning.effort（不传=服务端默认）
	Multimodal   bool         // ★ 多模态：带 Images 的 user 消息转 input_image 块
	Client       *http.Client // nil → 默认客户端（超时见 llm* 变量）

	// OnRetry 重试通知回调（同 OpenAIProvider）。
	OnRetry func(attempt int, maxRetries int, errMsg string)

	clientOnce  sync.Once
	clientCache *http.Client
}

// SetOnRetry 绑定重试通知（RetryNotifier 接口）。
func (p *ResponsesProvider) SetOnRetry(fn func(attempt, maxRetries int, errMsg string)) {
	p.OnRetry = fn
}

func (p *ResponsesProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	p.clientOnce.Do(func() {
		p.clientCache = defaultLLMClient()
	})
	return p.clientCache
}

func (p *ResponsesProvider) Name() string { return "responses:" + p.Model }

// Chat 实现 Provider 接口：消息/工具/图片 → Responses API 形态 → 流式 POST。
func (p *ResponsesProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	messages = sanitizeToolPairing(messages)

	for _, m := range messages {
		if len(m.Images) > 0 {
			log.Printf("[llm] responses 多模态消息: role=%s images=%d multimodal=%v mime=%s",
				m.Role, len(m.Images), p.Multimodal, firstMime(m.Images))
		}
	}

	system, msgs := separateSystem(messages)
	body := map[string]any{
		"model":  p.Model,
		"stream": true,
	}
	if system != "" {
		body["instructions"] = system
	}
	body["input"] = buildResponsesInput(msgs, p.Multimodal)
	if len(tools) > 0 {
		body["tools"] = toResponsesTools(tools)
	}
	if p.Temperature >= 0 {
		body["temperature"] = p.Temperature
	}
	if p.MaxTokens > 0 {
		body["max_output_tokens"] = p.MaxTokens
	}
	// 思考档位 → reasoning.effort（OpenAI 官方字段；none 不传用服务端默认关闭）
	if eff := normalizeThinkingMode(p.ThinkingMode); eff != "" && eff != "none" {
		body["reasoning"] = map[string]any{"effort": eff}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return Message{}, err
	}

	url := ResolveEndpointURL(p.BaseURL, ProtocolOpenAIResponses)
	call := llmStreamCall{
		URL:    url,
		Body:   buf,
		APIKey: p.APIKey, // 默认 Authorization Bearer
	}
	return postLLMStream(ctx, p.client(), call, func(r io.Reader) (Message, error) {
		return parseResponsesSSE(r, onChunk)
	}, func(attempt int, errMsg string) {
		if p.OnRetry != nil {
			p.OnRetry(attempt, llmMaxRetries, errMsg)
		}
	})
}

// buildResponsesInput 内部 Message 列表 → Responses input 数组（消息/事件混合）。
//   - user 文本 → {"role":"user","content":[{"type":"input_text",...}, 图片 input_image 块]}
//   - assistant 带 ToolCalls → content=[function_call 块]（call_id 为关联键）
//   - tool 结果 → 顶层 {"type":"function_call_output","call_id":...,"output":...}
//   - assistant 纯文本 → content=[output_text 块]
func buildResponsesInput(msgs []Message, multimodal bool) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case RoleSystem:
			continue // 已并入 instructions
		case RoleUser:
			if m.ToolCallID != "" {
				// tool 结果 → 顶层 function_call_output（非消息）
				out = append(out, map[string]any{
					"type":    "function_call_output",
					"call_id": m.ToolCallID,
					"output":  m.Content,
				})
			} else if multimodal && len(m.Images) > 0 {
				blocks := make([]any, 0, 1+len(m.Images))
				if m.Content != "" {
					blocks = append(blocks, map[string]any{"type": "input_text", "text": m.Content})
				}
				for _, img := range m.Images {
					if b := toResponsesImageBlock(img); b != nil {
						blocks = append(blocks, b)
					}
				}
				if len(blocks) == 0 {
					blocks = append(blocks, map[string]any{"type": "input_text", "text": m.Content})
				}
				out = append(out, map[string]any{"role": "user", "content": blocks})
			} else {
				out = append(out, map[string]any{
					"role": "user",
					"content": []any{
						map[string]any{"type": "input_text", "text": m.Content},
					},
				})
			}
		case RoleAssistant:
			if len(m.ToolCalls) > 0 {
				blocks := make([]any, 0, len(m.ToolCalls))
				for _, tc := range m.ToolCalls {
					blocks = append(blocks, map[string]any{
						"type":      "function_call",
						"call_id":   tc.ID,
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					})
				}
				out = append(out, map[string]any{"role": "assistant", "content": blocks})
			} else {
				out = append(out, map[string]any{
					"role": "assistant",
					"content": []any{
						map[string]any{"type": "output_text", "text": m.Content},
					},
				})
			}
		case RoleTool:
			// 孤立 tool 结果（sanitize 后罕见）→ 兜底 function_call_output
			out = append(out, map[string]any{
				"type":    "function_call_output",
				"call_id": m.ToolCallID,
				"output":  m.Content,
			})
		}
	}
	return out
}

// toResponsesImageBlock ImagePart → Responses input_image 块（image_url 为 data URL 或 http URL）。
func toResponsesImageBlock(img ImagePart) map[string]any {
	if img.Data == "" {
		return nil
	}
	b := map[string]any{
		"type":      "input_image",
		"image_url": img.Data,
	}
	if img.Detail != "" {
		b["detail"] = img.Detail
	}
	return b
}

// toResponsesTools ToolDefinition → Responses tools（type=function 顶层形态）。
func toResponsesTools(tools []ToolDefinition) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		params := t.Function.Parameters
		if params == nil {
			params = map[string]any{"type": "object"}
		}
		out = append(out, map[string]any{
			"type":        "function",
			"name":        t.Function.Name,
			"description": t.Function.Description,
			"parameters":  params,
		})
	}
	return out
}

// responsesSSEFrame Responses SSE 帧（只解析需要的字段）。
type responsesSSEFrame struct {
	Type        string `json:"type"`
	Delta       string `json:"delta"`
	Text        string `json:"text"`       // output_text.done / reasoning_summary_text.done
	Arguments   string `json:"arguments"`  // function_call_arguments.done 帧的完整参数
	ItemID      string `json:"item_id"`
	OutputIndex int    `json:"output_index"` // 部分实现无 item_id 时的兜底关联键
	Item        struct {
		Type      string `json:"type"`
		ID        string `json:"id"`
		CallID    string `json:"call_id"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"item"`
	Response struct {
		Status string `json:"status"`
		Usage  *Usage `json:"usage"`
		IncompleteDetails *struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Output []struct {
			Type      string `json:"type"`
			CallID    string `json:"call_id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"output"`
	} `json:"response"`
}

// parseResponsesSSE 解析 Responses 事件流，累积成最终 assistant Message。
// 完成标记：response.completed 或 [DONE]（两者皆认）。
func parseResponsesSSE(r io.Reader, onChunk func(Chunk)) (Message, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var contentB, reasoningB strings.Builder
	type fcItem struct {
		callID string
		name   string
	}
	fcItems := map[string]*fcItem{}   // item_id → {callID,name}
	argBufs := map[string]*strings.Builder{} // item_id → arguments 增量
	var toolCalls []ToolCall
	var usage *Usage
	var stopReason string
	sawCompleted := false // response.completed / [DONE]

	finish := func() (Message, error) {
		msg := Message{Role: RoleAssistant, Content: contentB.String(), Reasoning: reasoningB.String()}
		msg.ToolCalls = toolCalls
		if err := sc.Err(); err != nil {
			return msg, fmt.Errorf("读取 Responses SSE 流失败: %w", err)
		}
		if !sawCompleted {
			return msg, fmt.Errorf("Responses SSE 流提前结束（未收到 response.completed，已累积 %d 字符）",
				len(msg.Content)+len(msg.Reasoning))
		}
		if onChunk != nil {
			onChunk(Chunk{Done: true, Usage: usage, StopReason: stopReason})
		}
		return msg, nil
	}

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			sawCompleted = true
			return finish()
		}
		if data == "" {
			continue
		}
		var frame responsesSSEFrame
		if err := json.Unmarshal([]byte(data), &frame); err != nil {
			continue // 容错：忽略坏帧
		}

		switch frame.Type {
		case "response.output_item.added":
			if frame.Item.Type == "function_call" {
				fcItems[frame.Item.ID] = &fcItem{callID: frame.Item.CallID, name: frame.Item.Name}
			}
		case "response.output_text.delta":
			contentB.WriteString(frame.Delta)
			if onChunk != nil {
				onChunk(Chunk{Content: frame.Delta})
			}
		case "response.reasoning_summary_text.delta":
			reasoningB.WriteString(frame.Delta)
			if onChunk != nil {
				onChunk(Chunk{Reasoning: frame.Delta})
			}
		case "response.function_call_arguments.delta":
			key := frame.ItemID
			if key == "" {
				key = fmt.Sprintf("#%d", frame.OutputIndex) // 兼容：无 item_id 用 output_index
			}
			buf := argBufs[key]
			if buf == nil {
				buf = &strings.Builder{}
				argBufs[key] = buf
			}
			buf.WriteString(frame.Delta)
		case "response.function_call_arguments.done":
			key := frame.ItemID
			if key == "" {
				key = fmt.Sprintf("#%d", frame.OutputIndex)
			}
			args := frame.Arguments
			if buf, ok := argBufs[key]; ok {
				args = buf.String()
			}
			it := fcItems[key]
			if it == nil {
				// 容错：done 帧未见过 added 帧，用 done 帧自身信息（item_id 即标识）
				it = &fcItem{callID: key, name: ""}
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:   it.callID,
				Type: "function",
				Function: FunctionCall{
					Name:      it.name,
					Arguments: args,
				},
			})
		case "response.completed":
			sawCompleted = true
			if frame.Response.Usage != nil {
				usage = frame.Response.Usage
			}
			// 完成原因：status/incomplete_details 推断；有 function_call 输出 → tool_calls
			switch frame.Response.Status {
			case "incomplete":
				if frame.Response.IncompleteDetails != nil && frame.Response.IncompleteDetails.Reason == "max_output_tokens" {
					stopReason = "length"
				} else {
					stopReason = "incomplete"
				}
			case "completed", "in_progress":
				hasFC := false
				for _, o := range frame.Response.Output {
					if o.Type == "function_call" {
						hasFC = true
						break
					}
				}
				if hasFC {
					stopReason = "tool_calls"
				} else {
					stopReason = "stop"
				}
			default:
				stopReason = frame.Response.Status
			}
			return finish()
		}
		// ping / 其他事件：忽略
	}
	return finish()
}
