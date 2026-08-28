// jsplugin_providerimpl.go — JS Provider 实现桥（ctx.provider.register，t1 S1 闭环）
//
// ★ 背景（2026-09）：Provider 只有参数可装配（ProviderFactory），实现本身不可
//   插拔——新协议（Anthropic 原生、本地推理等）必须改 Go 内核。本桥提供实现级
//   插件槽位：
//
//	ctx.provider.register(name, impl)  注册 JS 实现的 Provider（同名覆盖）
//	ctx.provider.http(req)             宿主 HTTP 通道（JS 实现调任意 LLM 端点）
//
// 契约（impl）：
//
//	impl = { chat(params, messages, tools) }
//	  params   = ProviderParams 快照（provider/baseURL/apiKey/model/temperature/…）
//	  messages = [{role, content, reasoning, toolCalls?, toolCallId?, name?, images?}]
//	  tools    = [{type:'function', function:{name, description, parameters}}]
//	  → 返回（或 Promise 解析为）{ content, reasoning,
//	      toolCalls: [{id, name, arguments(JSON 字符串)}] }
//
// 非流式契约：Go 侧一次性 emit Done chunk（对齐 MockProvider——循环对非流式
// Provider 完全兼容）。JS 侧用 ctx.provider.http 调端点、自行解析协议响应
// （JSON/SSE 均可），新协议实现 100% 在插件内，无需改 Go 内核。

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"wb-ui/goja"
)

// providerJSChatTimeout JS chat 调用兜底超时（LLM 请求通常分钟级；
// 超过视为插件实现卡死，返回错误不阻塞循环）。
const providerJSChatTimeout = 10 * time.Minute

// jsProviderBridge 把 JS 实现的 chat 接到 Provider 接口（非流式）。
type jsProviderBridge struct {
	vm     *goja.Runtime
	plugin *jsPluginAdapter
	name   string
	chat   goja.Callable  // chat(params, messages, tools) → Promise<result>
	params ProviderParams // 创建时的最终参数快照（CreateProvider 传入，JS chat 可见）
}

// Name 服务商名（诊断/日志展示）。
func (b *jsProviderBridge) Name() string { return b.name }

// Chat 实现 Provider：把 messages/tools 转 JS 值 → 直接调 JS chat（对齐
// jsProviderFactoryBridge 的调用形态，避免 Callable 经全局转发参数错位）→
// awaitJSValue 等 Promise → 解析结果 → 一次性 emit Done chunk。
func (b *jsProviderBridge) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	paramsVal := b.vm.ToValue(providerParamsToJS(b.params))
	msgsVal := b.vm.ToValue(messagesToJS(messages))
	toolsVal := b.vm.ToValue(toolsToJSDefs(tools))

	var (
		ret     goja.Value
		callErr error
	)
	// goja 非并发安全：JS chat 必须持 VM 锁执行（可能来自任意 goroutine）。
	// runJSWithTimeout 兜底：JS 实现卡死（死循环）超时强制中断。
	b.plugin.withLock(func() {
		callErr = runJSWithTimeout(b.vm, providerJSChatTimeout, func() error {
			v, err := b.chat(goja.Undefined(), paramsVal, msgsVal, toolsVal)
			if err != nil {
				return err
			}
			ret, err = awaitJSValue(b.vm, v)
			return err
		})
	})
	if callErr != nil {
		if isJSTimeout(callErr) {
			return Message{}, fmt.Errorf("JS Provider %s chat 超时（%.0fs，实现疑似卡死）", b.name, providerJSChatTimeout.Seconds())
		}
		return Message{}, fmt.Errorf("JS Provider %s chat 执行失败: %w", b.name, callErr)
	}
	select {
	case <-ctx.Done():
		return Message{}, ctx.Err()
	default:
	}
	if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
		return Message{}, fmt.Errorf("JS Provider %s chat 无返回", b.name)
	}
	if e := ret.ToObject(b.vm).Get("__error"); e != nil && !goja.IsUndefined(e) && e.String() != "" {
		return Message{}, fmt.Errorf("JS Provider %s chat 失败: %s", b.name, e.String())
	}
	msg, err := b.parseResult(ret)
	if err != nil {
		return Message{}, err
	}
	if onChunk != nil {
		onChunk(Chunk{Content: msg.Content, Reasoning: msg.Reasoning, ToolCalls: msg.ToolCalls, Done: true})
	}
	return msg, nil
}

// parseResult 解析 JS chat 返回对象 → Message。
// 支持形态：{ content, reasoning, toolCalls: [{id, name, arguments}] }。
func (b *jsProviderBridge) parseResult(v goja.Value) (Message, error) {
	obj := v.ToObject(b.vm)
	msg := Message{Role: RoleAssistant}
	if c := obj.Get("content"); c != nil && !goja.IsUndefined(c) && !goja.IsNull(c) {
		msg.Content = c.String()
	}
	if r := obj.Get("reasoning"); r != nil && !goja.IsUndefined(r) && !goja.IsNull(r) {
		msg.Reasoning = r.String()
	}
	if tc := obj.Get("toolCalls"); tc != nil && !goja.IsUndefined(tc) && !goja.IsNull(tc) {
		exp := tc.Export()
		arr, ok := exp.([]any)
		if !ok {
			return Message{}, fmt.Errorf("JS Provider %s: toolCalls 必须是数组", b.name)
		}
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name, _ := m["name"].(string)
			argsStr := ""
			switch a := m["arguments"].(type) {
			case string:
				argsStr = a
			case map[string]any:
				if bts, err := json.Marshal(a); err == nil {
					argsStr = string(bts)
				}
			}
			if name == "" {
				continue
			}
			id, _ := m["id"].(string)
			if id == "" {
				id = fmt.Sprintf("call_%d", len(msg.ToolCalls))
			}
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{
				ID:   id,
				Type: "function",
				Function: FunctionCall{
					Name:      name,
					Arguments: argsStr,
				},
			})
		}
	}
	return msg, nil
}

// ─── 消息/工具转换 ─────────────────────────────────────────

// messagesToJS 把 Go 消息历史转 JS 可读数组（OpenAI 兼容形状）。
func messagesToJS(msgs []Message) []any {
	out := make([]any, 0, len(msgs))
	for _, m := range msgs {
		item := map[string]any{
			"role":    string(m.Role),
			"content": m.Content,
		}
		if m.Reasoning != "" {
			item["reasoning"] = m.Reasoning
		}
		if m.ToolCallID != "" {
			item["toolCallId"] = m.ToolCallID
		}
		if m.Name != "" {
			item["name"] = m.Name
		}
		if len(m.ToolCalls) > 0 {
			tcs := make([]any, 0, len(m.ToolCalls))
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
			item["toolCalls"] = tcs
		}
		if len(m.Images) > 0 {
			imgs := make([]any, 0, len(m.Images))
			for _, im := range m.Images {
				imgs = append(imgs, map[string]any{
					"data": im.Data, "mimeType": im.MimeType, "detail": im.Detail,
				})
			}
			item["images"] = imgs
		}
		out = append(out, item)
	}
	return out
}

// toolsToJSDefs 把 Go 工具定义转 JS 数组（OpenAI function-calling schema）。
func toolsToJSDefs(tools []ToolDefinition) []any {
	out := make([]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Function.Name,
				"description": t.Function.Description,
				"parameters":  t.Function.Parameters,
			},
		})
	}
	return out
}

// providerParamsToJS 把 ProviderParams 快照转 JS 对象（JS chat 的 params 参数）。
func providerParamsToJS(cur ProviderParams) map[string]any {
	return map[string]any{
		"provider":     cur.Provider,
		"baseURL":      cur.BaseURL,
		"apiKey":       cur.APIKey,
		"model":        cur.Model,
		"temperature":  cur.Temperature,
		"maxTokens":    cur.MaxTokens,
		"thinkingMode": cur.ThinkingMode,
		"multimodal":   cur.Multimodal,
	}
}
