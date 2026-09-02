// provider_anthropic.go — Anthropic Messages API 适配器（★ 2026-09-02）
//
// 协议：POST {base}/v1/messages，认证 x-api-key + anthropic-version: 2023-06-01。
// 与 OpenAI 兼容协议的核心差异：
//   - system 是顶层字段（不在 messages 里）
//   - 工具调用为内容块：assistant content=[{type:tool_use,id,name,input}]
//     后跟 user content=[{type:tool_result,tool_use_id,content}]
//   - 图片为 content 块 {type:image,source:{type:base64,media_type,data}}
//   - SSE 事件流：message_start / content_block_start / content_block_delta /
//     content_block_stop / message_delta / message_stop（无 [DONE]）
//   - max_tokens 为必填
//
// 未覆盖（扩展点，注释留痕）：Anthropic 原生 thinking 块（budget_tokens）——
// 需要预算字段，暂不映射 ThinkingMode（默认模型自带能力，disabled 安全）。另：MCP 工具的
// tool_use_type 字段未用（保持与 OpenAI 契约一致的 name/input 形态）。

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

// anthropicVersion Anthropic API 版本头。
const anthropicVersion = "2023-06-01"

// anthropicDefaultMaxTokens max_tokens 为必填；未配置时兜底值。
const anthropicDefaultMaxTokens = 4096

// AnthropicProvider Anthropic Messages API 流式适配器。
// 差异仅 BaseURL+Model+APIKey；内部消息/工具/图片自动转换为 Anthropic 形态。
type AnthropicProvider struct {
	BaseURL      string // 基础地址（如 https://api.anthropic.com）；完整端点由 ResolveEndpointURL 拼接
	Protocol     string // ★ 恒为 anthropic-messages（可空，ResolveEndpointURL 默认按 base 判定）
	APIKey       string
	Model        string
	Temperature  float64      // <0 = 不下发（用服务端默认）；>=0 下发
	MaxTokens    int          // >0 时下发（必填，未配置用 anthropicDefaultMaxTokens）
	ThinkingMode string       // 扩展点：暂不映射（见文件头注释）
	Multimodal   bool         // ★ 多模态：带 Images 的 user 消息转 image content 块
	Client       *http.Client // nil → 默认客户端（超时见 llm* 变量）

	// OnRetry 重试通知回调（同 OpenAIProvider）。
	OnRetry func(attempt int, maxRetries int, errMsg string)

	clientOnce  sync.Once
	clientCache *http.Client
}

// SetOnRetry 绑定重试通知（RetryNotifier 接口）。
func (p *AnthropicProvider) SetOnRetry(fn func(attempt, maxRetries int, errMsg string)) {
	p.OnRetry = fn
}

func (p *AnthropicProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	p.clientOnce.Do(func() {
		p.clientCache = defaultLLMClient()
	})
	return p.clientCache
}

func (p *AnthropicProvider) Name() string { return "anthropic:" + p.Model }

// Chat 实现 Provider 接口：消息/工具/图片 → Anthropic Messages 形态 → 流式 POST。
func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	messages = sanitizeToolPairing(messages)

	// 多模态诊断（对齐 OpenAIProvider）
	for _, m := range messages {
		if len(m.Images) > 0 {
			log.Printf("[llm] anthropic 多模态消息: role=%s images=%d multimodal=%v mime=%s",
				m.Role, len(m.Images), p.Multimodal, firstMime(m.Images))
		}
	}

	system, msgs := separateSystem(messages)
	body := map[string]any{
		"model":      p.Model,
		"max_tokens": p.MaxTokens, // Anthropic 必填
		"stream":     true,
	}
	if p.MaxTokens <= 0 {
		body["max_tokens"] = anthropicDefaultMaxTokens
	}
	if system != "" {
		body["system"] = system
	}
	body["messages"] = buildAnthropicMessages(msgs, p.Multimodal)
	if len(tools) > 0 {
		body["tools"] = toAnthropicTools(tools)
	}
	if p.Temperature >= 0 {
		body["temperature"] = p.Temperature
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return Message{}, err
	}

	url := ResolveEndpointURL(p.BaseURL, ProtocolAnthropicMessages)
	call := llmStreamCall{
		URL:        url,
		Body:       buf,
		Headers:    map[string]string{"anthropic-version": anthropicVersion},
		AuthHeader: "x-api-key", // Anthropic 认证：x-api-key（非 Bearer）
		APIKey:     p.APIKey,
	}
	return postLLMStream(ctx, p.client(), call, func(r io.Reader) (Message, error) {
		return parseAnthropicSSE(r, onChunk)
	}, func(attempt int, errMsg string) {
		if p.OnRetry != nil {
			p.OnRetry(attempt, llmMaxRetries, errMsg)
		}
	})
}

// separateSystem 把 system 消息合并为顶层 system 字段（Anthropic messages 内不允许 system 角色）。
func separateSystem(msgs []Message) (string, []Message) {
	var sb strings.Builder
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == RoleSystem {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(m.Content)
		} else {
			out = append(out, m)
		}
	}
	return sb.String(), out
}

// buildAnthropicMessages 内部 Message 列表 → Anthropic messages 数组。
//   - assistant 带 ToolCalls → content=[{type:tool_use,...}] 块
//   - role=tool（工具结果）→ user content=[{type:tool_result,...}] 块；连续结果合并进同一 user 消息
//   - user 带 Images（multimodal）→ content 块数组（text + image）
func buildAnthropicMessages(msgs []Message, multimodal bool) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case RoleSystem:
			continue // 已归入顶层 system
		case RoleUser:
			if m.ToolCallID != "" {
				block := map[string]any{
					"type":        "tool_result",
					"tool_use_id": m.ToolCallID,
					"content":     m.Content,
				}
				// 连续 tool_result 合并进同一条 user 消息（Anthropic 要求结果紧跟 tool_use）
				if last := lastUserToolResult(out); last != nil {
					last["content"] = append(last["content"].([]any), block)
				} else {
					out = append(out, map[string]any{"role": "user", "content": []any{block}})
				}
			} else if multimodal && len(m.Images) > 0 {
				blocks := make([]any, 0, 1+len(m.Images))
				if m.Content != "" {
					blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
				}
				for _, img := range m.Images {
					if b := toAnthropicImageBlock(img); b != nil {
						blocks = append(blocks, b)
					}
				}
				if len(blocks) == 0 {
					blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
				}
				// ★ tool_result 序列后紧跟普通 user → 合并（Anthropic 要求 user/assistant 交替，
				//   不允许连续两条 user；把新文本块追加进同一条 user 消息）
				if last := lastUserToolResult(out); last != nil {
					last["content"] = append(last["content"].([]any), blocks...)
				} else {
					out = append(out, map[string]any{"role": "user", "content": blocks})
				}
			} else {
				// ★ tool_result 序列后紧跟普通 user → 合并文本（Alternating 规则）
				if last := lastUserToolResult(out); last != nil {
					last["content"] = append(last["content"].([]any), map[string]any{"type": "text", "text": m.Content})
				} else {
					out = append(out, map[string]any{"role": "user", "content": m.Content})
				}
			}
		case RoleAssistant:
			if len(m.ToolCalls) > 0 {
				blocks := make([]any, 0, len(m.ToolCalls))
				for _, tc := range m.ToolCalls {
					var input any
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
						input = map[string]any{} // 参数非法 → 空对象（Anthropic 要求 input 为对象）
					}
					blocks = append(blocks, map[string]any{
						"type":  "tool_use",
						"id":    tc.ID,
						"name":  tc.Function.Name,
						"input": input,
					})
				}
				out = append(out, map[string]any{"role": "assistant", "content": blocks})
			} else {
				out = append(out, map[string]any{"role": "assistant", "content": m.Content})
			}
		case RoleTool:
			// 孤立 tool 结果（sanitize 后罕见）→ 兜底转 tool_result 块
			block := map[string]any{
				"type":        "tool_result",
				"tool_use_id": m.ToolCallID,
				"content":     m.Content,
			}
			out = append(out, map[string]any{"role": "user", "content": []any{block}})
		}
	}
	return out
}

// lastUserToolResult 若 out 末条是「纯 tool_result 序列的 user 消息」则返回它（供合并）。
func lastUserToolResult(out []map[string]any) map[string]any {
	if len(out) == 0 {
		return nil
	}
	last := out[len(out)-1]
	if last["role"] != "user" {
		return nil
	}
	arr, ok := last["content"].([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	first, ok := arr[0].(map[string]any)
	if !ok || first["type"] != "tool_result" {
		return nil
	}
	return last
}

// toAnthropicImageBlock ImagePart → Anthropic image content 块。
// 支持 data URL（base64 source）与 http(s) URL（url source；需 Anthropic 可公网访问）。
func toAnthropicImageBlock(img ImagePart) map[string]any {
	if strings.HasPrefix(img.Data, "data:") {
		comma := strings.Index(img.Data, ",")
		if comma < 0 || comma == len(img.Data)-1 {
			return nil
		}
		meta := img.Data[:comma] // data:image/png;base64
		payload := img.Data[comma+1:]
		mime := img.MimeType
		if mime == "" {
			// 从 meta 解析（data:image/png;base64 → image/png）
			md := strings.TrimPrefix(meta, "data:")
			mime = strings.SplitN(md, ";", 2)[0]
		}
		if mime == "" {
			mime = "image/png"
		}
		return map[string]any{
			"type": "image",
			"source": map[string]any{
				"type":       "base64",
				"media_type": mime,
				"data":       payload,
			},
		}
	}
	if strings.HasPrefix(img.Data, "http://") || strings.HasPrefix(img.Data, "https://") {
		return map[string]any{
			"type": "image",
			"source": map[string]any{
				"type": "url",
				"url":  img.Data,
			},
		}
	}
	return nil
}

// toAnthropicTools ToolDefinition → Anthropic tools（input_schema）。
func toAnthropicTools(tools []ToolDefinition) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		params := t.Function.Parameters
		if params == nil {
			params = map[string]any{"type": "object"}
		}
		out = append(out, map[string]any{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": params,
		})
	}
	return out
}

// anthropicSSEFrame Anthropic SSE 帧（只解析需要的字段）。
type anthropicSSEFrame struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	ContentBlock struct {
		Type  string          `json:"type"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content_block"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		Thinking    string `json:"thinking"`
		PartialJSON string `json:"partial_json"`
	} `json:"delta"`
	Message struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
	MessageDelta struct {
		Delta struct {
			StopReason string `json:"stop_reason"`
		} `json:"delta"`
		Usage struct {
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message_delta"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// pendingAnthropicBlock 累积中的 content block。
type pendingAnthropicBlock struct {
	kind     string          // text | thinking | tool_use
	id       string
	name     string
	inputRaw json.RawMessage // content_block_start 全量 input（部分网关在 start 帧给出，而非 delta 流）
	json     strings.Builder // input_json_delta 累积
}

// parseAnthropicSSE 解析 Anthropic 消息流，累积成最终 assistant Message（对齐 parseSSE 行为：
// 内容增量 onChunk；工具调用仅在最终 Message 中返回；结束统一 onChunk(Done+Usage)）。
func parseAnthropicSSE(r io.Reader, onChunk func(Chunk)) (Message, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var content, reasoning strings.Builder
	pending := map[int]*pendingAnthropicBlock{}
	var toolCalls []ToolCall
	var usage *Usage
	var stopReason string
	sawMessageStop := false // 收到 message_stop = 完整结束

	finish := func() (Message, error) {
		msg := Message{Role: RoleAssistant, Content: content.String(), Reasoning: reasoning.String()}
		msg.ToolCalls = toolCalls
		if err := sc.Err(); err != nil {
			return msg, fmt.Errorf("读取 Anthropic SSE 流失败: %w", err)
		}
		if !sawMessageStop {
			return msg, fmt.Errorf("Anthropic SSE 流提前结束（未收到 message_stop，已累积 %d 字符）",
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
			continue // 跳过 event: 行 / 空行 / ping
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		var frame anthropicSSEFrame
		if err := json.Unmarshal([]byte(data), &frame); err != nil {
			continue // 容错：忽略坏帧
		}

		switch frame.Type {
		case "message_start":
			if usage == nil {
				usage = &Usage{}
			}
			usage.PromptTokens = frame.Message.Usage.InputTokens
			usage.CompletionTokens = frame.Message.Usage.OutputTokens
		case "content_block_start":
			blk := &pendingAnthropicBlock{
				kind:     frame.ContentBlock.Type,
				id:       frame.ContentBlock.ID,
				name:     frame.ContentBlock.Name,
				inputRaw: frame.ContentBlock.Input,
			}
			pending[frame.Index] = blk
		case "content_block_delta":
			blk := pending[frame.Index]
			switch frame.Delta.Type {
			case "text_delta":
				content.WriteString(frame.Delta.Text)
				if onChunk != nil {
					onChunk(Chunk{Content: frame.Delta.Text})
				}
			case "thinking_delta":
				reasoning.WriteString(frame.Delta.Thinking)
				if onChunk != nil {
					onChunk(Chunk{Reasoning: frame.Delta.Thinking})
				}
			case "input_json_delta":
				if blk != nil {
					blk.json.WriteString(frame.Delta.PartialJSON)
				}
			}
		case "content_block_stop":
			blk := pending[frame.Index]
			if blk != nil && blk.kind == "tool_use" {
				argsRaw := blk.json.String()
				// Anthropic 的 input 是对象（服务端完整输出 JSON）；工具执行层吃 JSON 字符串
				if argsRaw == "" {
					// 无 delta（input 在 content_block_start 已全量给出——罕见，兼容之）
					argsRaw = string(blk.inputRaw)
				}
				toolCalls = append(toolCalls, ToolCall{
					ID:   blk.id,
					Type: "function",
					Function: FunctionCall{
						Name:      blk.name,
						Arguments: argsRaw,
					},
				})
			}
		case "message_delta":
			if frame.MessageDelta.Delta.StopReason != "" {
				stopReason = mapAnthropicStopReason(frame.MessageDelta.Delta.StopReason)
			}
			if frame.MessageDelta.Usage.OutputTokens > 0 {
				if usage == nil {
					usage = &Usage{}
				}
				usage.CompletionTokens = frame.MessageDelta.Usage.OutputTokens
			}
			if usage != nil {
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			}
		case "message_stop":
			sawMessageStop = true
			return finish()
		case "error":
			msg := fmt.Sprintf("Anthropic API 错误: %s", frame.Error.Message)
			if frame.Error != nil && frame.Error.Type != "" {
				msg = fmt.Sprintf("Anthropic API 错误 (%s): %s", frame.Error.Type, frame.Error.Message)
			}
			return Message{}, fmt.Errorf("%s", msg)
		}
		// ping / 其他类型：忽略
	}
	return finish()
}

// mapAnthropicStopReason Anthropic stop_reason → 内部完成原因（与 OpenAI finish_reason 对齐）。
func mapAnthropicStopReason(r string) string {
	switch r {
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		return r
	}
}
