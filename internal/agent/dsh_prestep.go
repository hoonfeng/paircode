package agent

// ═══════════════════════════════════════════════════════════
// dsh_prestep.go — DSH agent/pre-step 中间件瀑布桥
//
// 外部语义（@nanmicoder/dsh-agent-teams installAgentTeamsGestureBoundary）：
//   ctx.on('agent/pre-step', async ({messages, signal}, next) => decision)
//   - messages: 外部格式消息数组（role/content blocks/source）
//   - next(): 下游决策（基线 = {kind:'enter', messages: 原始 messages}）
//   - 返回 decision：{kind:'enter', messages:[...]}（改写进入模型的输入）
//      或 {kind:'reject'}（拒绝整个 turn）
//   - 多个订阅者按注册顺序组成 koa 式瀑布（每个 handler 的 next 指向下游）
//
// 宿主接入语义（loop.go runPreStep）：
//   1. host 钩子 l.PreStep（若有）先执行（可改写/拒绝）
//   2. 桥瀑布（若有订阅者）基于（改写后的）消息再决策——最终以桥结果为准
//   无订阅者/桥未就绪 → 零开销直通（与 emitBridgeEvent 白名单门控一致）。
// ═══════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// ── Go Message ↔ DSH Message 转换 ──────────────────────────
//
// DSH message 形态（@deepseek-ai/dsh-llm types/message.d.ts）：
//   { role: 'system'|'user'|'assistant',
//     content: ContentBlock[],
//     source: {kind:'user'|'plugin'|'model'|'tool', ...} }
//   ContentBlock: {type:'text',text} | {type:'reasoning',text} |
//                 {type:'tool-call',id,name,arguments} |
//                 {type:'tool-result',toolCallId,content,isError} |
//                 {type:'image',attachment}

// goMsgToDSH 把单条 Go Message 转换为 DSH message（map 形态，便于 JSON 往返）。
// tool 消息按 外部惯例转 role='user' + source.kind='tool'（createToolResultMessage 同构）。
func goMsgToDSH(m Message) map[string]any {
	blocks := make([]map[string]any, 0, 3)
	switch m.Role {
	case RoleTool:
		// DSH：tool 结果 = user-role + source.kind='tool'
		return map[string]any{
			"role": "user",
			"source": map[string]any{
				"kind":   "tool",
				"callId": m.ToolCallID,
			},
			"content": []any{map[string]any{
				"type":       "tool-result",
				"toolCallId": m.ToolCallID,
				"isError":    false,
				"content":    []any{map[string]any{"type": "text", "text": m.Content}},
			}},
		}
	case RoleAssistant:
		if m.Reasoning != "" {
			blocks = append(blocks, map[string]any{"type": "reasoning", "text": m.Reasoning})
		}
		if m.Content != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
		}
		for _, tc := range m.ToolCalls {
			blocks = append(blocks, map[string]any{
				"type":      "tool-call",
				"id":        tc.ID,
				"name":      tc.Function.Name,
				"arguments": tc.Function.Arguments,
			})
		}
		return map[string]any{
			"role":    "assistant",
			"content": blocks,
			"source":  map[string]any{"kind": "model", "provider": "", "model": ""},
		}
	case RoleSystem:
		if m.Content != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
		}
		// 外部无 system 专用 source kind：用 plugin（non-user 即安全）
		return map[string]any{
			"role":    "system",
			"content": blocks,
			"source":  map[string]any{"kind": "plugin", "plugin": "gou-ide"},
		}
	default: // user
		if m.Content != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
		}
		for _, img := range m.Images {
			att := map[string]any{}
			if img.Data != "" {
				att["url"] = img.Data
			}
			if img.MimeType != "" {
				att["mimeType"] = img.MimeType
			}
			if len(att) == 0 {
				continue
			}
			blocks = append(blocks, map[string]any{"type": "image", "attachment": att})
		}
		return map[string]any{
			"role":    "user",
			"content": blocks,
			"source":  map[string]any{"kind": "user"},
		}
	}
}

// dshToGoMsg 把 DSH message 转回 Go Message（逐块还原：text/reasoning/
// tool-call/tool-result/image）。未知块忽略（forward-compatible）。
// ★ content 可能以 []any（JSON 反序列化）或 []map[string]any（内存构造）两种
//
//	形态出现——逐项转换到统一 any 列表再处理。
func dshToGoMsg(raw map[string]any) Message {
	role, _ := raw["role"].(string)
	src, _ := raw["source"].(map[string]any)
	srcKind, _ := src["kind"].(string)

	blocks := contentBlocksAny(raw["content"])

	// DSH tool 结果：role='user' + source.kind='tool'
	if role == "user" && srcKind == "tool" {
		callID, _ := src["callId"].(string)
		var content strings.Builder
		for _, b := range blocks {
			blk, ok := b.(map[string]any)
			if !ok {
				continue
			}
			if blk["type"] == "tool-result" {
				if cid, _ := blk["toolCallId"].(string); cid != "" {
					callID = cid
				}
				content.WriteString(blockText(blk["content"]))
			} else if blk["type"] == "text" {
				content.WriteString(textOf(blk))
			}
		}
		return Message{Role: RoleTool, Content: strings.TrimSpace(content.String()), ToolCallID: callID}
	}

	var text, reasoning strings.Builder
	var tcs []ToolCall
	var images []ImagePart
	for _, b := range blocks {
		blk, ok := b.(map[string]any)
		if !ok {
			continue
		}
		switch blk["type"] {
		case "text":
			s := textOf(blk)
			if s != "" {
				text.WriteString(s)
			}
		case "reasoning":
			if s, _ := blk["text"].(string); s != "" {
				reasoning.WriteString(s)
			}
		case "tool-call":
			id, _ := blk["id"].(string)
			name, _ := blk["name"].(string)
			args, _ := blk["arguments"].(string)
			tcs = append(tcs, ToolCall{
				ID:   id,
				Type: "function",
				Function: FunctionCall{
					Name:      name,
					Arguments: args,
				},
			})
		case "image":
			if att, ok := blk["attachment"].(map[string]any); ok {
				img := ImagePart{}
				if u, _ := att["url"].(string); u != "" {
					img.Data = u
				}
				if mt, _ := att["mimeType"].(string); mt != "" {
					img.MimeType = mt
				}
				if img.Data != "" || img.MimeType != "" {
					images = append(images, img)
				}
			}
		}
	}
	return Message{
		Role:      Role(roleText(role)),
		Content:   text.String(),
		ToolCalls: tcs,
		Reasoning: reasoning.String(),
		Images:    images,
	}
}

// contentBlocksAny 把 content 块列表归一为 []any（兼容 []any 与 []map[string]any）。
func contentBlocksAny(v any) []any {
	if arr, ok := v.([]any); ok {
		return arr
	}
	if arr, ok := v.([]map[string]any); ok {
		out := make([]any, 0, len(arr))
		for _, m := range arr {
			out = append(out, m)
		}
		return out
	}
	return nil
}

// blockText 提取嵌套 content（tool-result 块的 content 是 ContentBlock[]）。
func blockText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	var sb strings.Builder
	for _, item := range contentBlocksAny(v) {
		if blk, ok := item.(map[string]any); ok {
			if s := textOf(blk); s != "" {
				sb.WriteString(s)
			}
		}
	}
	return sb.String()
}

func textOf(blk map[string]any) string {
	if s, ok := blk["text"].(string); ok {
		return s
	}
	return ""
}

func roleText(r string) string {
	switch r {
	case "system":
		return string(RoleSystem)
	case "user":
		return string(RoleUser)
	case "assistant":
		return string(RoleAssistant)
	default:
		return string(RoleUser)
	}
}

// msgsToDSH / dshToMsgs 批量转换（保持顺序）。
func msgsToDSH(msgs []Message) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, goMsgToDSH(m))
	}
	return out
}

func dshToMsgs(raw []map[string]any) []Message {
	out := make([]Message, 0, len(raw))
	for _, m := range raw {
		out = append(out, dshToGoMsg(m))
	}
	return out
}

// ── 桥瀑布请求（Go → Node）─────────────────────────────────
//
// 协议：发送 {"t":"prestep","id":N,"payload":{"messages":[…],"turn":T,"step":S}}
//       Node 侧执行中间件瀑布，回 {"t":"result","id":N,"ok":true,"data":<决策JSON>}
//       data 形态：{"kind":"enter","messages":[...]} | {"kind":"reject"}
// 无订阅者/桥未就绪 → 直通（rewritten=nil, reject=false, err=nil）。

// bridgePreStepSubscribed 是否有 Node 插件订阅 agent/pre-step（零开销门控）。
func bridgePreStepSubscribed() bool {
	b := globalNodeBridge
	return b != nil && b.isReady() && b.bridgeHasSubscribers("agent/pre-step")
}

// bridgePreStep 执行 DSH agent/pre-step 中间件瀑布。
// 返回：rewritten（enter 时桥改写后的消息；nil=未改写）、reject（reject 决策）、err。
func bridgePreStep(ctx context.Context, callMsgs []Message, turn, step int) ([]Message, bool, error) {
	if !bridgePreStepSubscribed() {
		return nil, false, nil
	}
	b := globalNodeBridge
	b.mu.Lock()
	b.seq++
	id := b.seq
	ch := make(chan bridgeResult, 1)
	b.pending[id] = ch
	b.mu.Unlock()

	payload := map[string]any{
		"messages": msgsToDSH(callMsgs),
		"turn":     turn,
		"step":     step,
	}
	line, _ := json.Marshal(map[string]any{"t": "prestep", "id": id, "payload": payload})
	if err := b.sendLine(line); err != nil {
		return nil, false, fmt.Errorf("pre-step 桥发送失败: %v", err)
	}
	select {
	case r := <-ch:
		if !r.ok {
			return nil, true, fmt.Errorf("pre-step 瀑布失败: %s", r.error)
		}
		return applyPreStepDecision(r.data)
	case <-time.After(60 * time.Second):
		return nil, true, fmt.Errorf("pre-step 瀑布超时（60 秒）")
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}
}

// applyPreStepDecision 解析 Node 回传的决策 JSON。
func applyPreStepDecision(data string) ([]Message, bool, error) {
	var dec struct {
		Kind     string           `json:"kind"`
		Messages []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal([]byte(data), &dec); err != nil {
		return nil, false, fmt.Errorf("pre-step 决策解析失败: %v", err)
	}
	switch dec.Kind {
	case "reject":
		return nil, true, nil
	case "enter":
		if dec.Messages == nil {
			return nil, false, nil
		}
		return dshToMsgs(dec.Messages), false, nil
	default:
		log.Printf("[pre-step] 未知决策 kind=%q（直通）", dec.Kind)
		return nil, false, nil
	}
}
