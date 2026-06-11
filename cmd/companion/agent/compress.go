package agent

// 上下文压缩——忠实复刻参考源 src/agent/context/manager.ts（主动压缩 + 容量控制 + LLM/规则两路摘要），
// 按 companion 单循环裁剪：没有 tiktoken/worker/会话记忆，用启发式估算 + 单段摘要。
//
// 触发：每次 LLM 调用前，tokens/MaxContextTokens 超 compactRatio 即把「中段老消息」压成一条摘要。
// 保留：开头 system 前缀 + 最近 compactKeepRecent 条（关键：保 tool_call↔tool_result 配对不破，
//       否则 OpenAI 规范报错）。回灌：丢弃段换成一条 user 角色摘要消息，夹在 前缀 + 摘要 + 最近段 之间。

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	compactRatio         = 0.8 // 触发阈值：tokens/MaxContextTokens 超此即压缩（复刻参考主动压缩 ~0.83）
	compactKeepRecent    = 16  // 恒留最近条数（复刻参考 keepCount=16）
	compactMinDrop       = 2   // 中段可丢条数下限：太少不值得压
	compactLLMSlice      = 40  // LLM 摘要喂入的末尾非 system 条数上限（复刻参考 slice(-40)）
	compactCooldownTurns = 4   // 压缩后冷却轮数：期间不再压缩（复刻参考 refreshCooldown，防每轮重复压缩/反复摘要）
)

// maybeCompact 若上下文超窗口阈值，把中段老消息压成摘要后返回新 msgs；否则原样返回。
// 复刻参考主动压缩：tokens 优先用上一轮 API 实测 prompt_tokens（含模板开销更可信），否则启发式估算。
func (l *Loop) maybeCompact(ctx context.Context, msgs []Message) []Message {
	if l.MaxContextTokens <= 0 {
		return msgs // 未配置窗口上限 → 压缩关闭
	}
	if l.compactCooldown > 0 {
		l.compactCooldown-- // 冷却期：刚压过，先让上下文重新积累几轮，避免反复摘要
		return msgs
	}
	tokens := estimateTokens(msgs)
	if l.lastPromptTokens > tokens { // 取实测与估算较大者
		tokens = l.lastPromptTokens
	}
	if float64(tokens) < compactRatio*float64(l.MaxContextTokens) {
		return msgs // 未超阈值
	}
	out, dropped := l.compact(ctx, msgs)
	if dropped <= 0 {
		return msgs // 没压成（中段太短，或全是要保的配对）
	}
	l.compactCooldown = compactCooldownTurns // 进入冷却，避免下几轮反复压缩
	l.lastPromptTokens = 0                    // 重置：压缩后等下轮实测/重新估算
	l.emit(Event{Type: EventCompacted, Content: fmt.Sprintf("上下文已压缩 · 早期 %d 条对话合并为摘要，保留最近 %d 条", dropped, len(out)-prefixLen(out)-1)})
	return out
}

// compact 执行压缩：系统前缀 + 一条摘要 + 最近段。返回新 msgs 与被压缩(丢弃)的条数。
func (l *Loop) compact(ctx context.Context, msgs []Message) ([]Message, int) {
	prefix := prefixLen(msgs) // 开头连续 system 条数（恒留）
	keepFrom := len(msgs) - compactKeepRecent
	if keepFrom < prefix {
		keepFrom = prefix
	}
	// 关键：最近段不能以孤立 tool 结果开头——其 assistant tool_call 若落在丢弃段，结果就成孤儿，
	// OpenAI 规范报错。前移 keepFrom 越过开头的 tool 条（连同其配对一起并入丢弃段→摘要）。
	for keepFrom < len(msgs) && msgs[keepFrom].Role == RoleTool {
		keepFrom++
	}
	dropped := msgs[prefix:keepFrom]
	if len(dropped) < compactMinDrop {
		return msgs, 0 // 中段太短，不值得压
	}
	summary := l.summarize(ctx, dropped)
	out := make([]Message, 0, prefix+1+len(msgs)-keepFrom)
	out = append(out, msgs[:prefix]...)
	out = append(out, Message{Role: RoleUser, Content: summary})
	out = append(out, msgs[keepFrom:]...)
	return out, len(dropped)
}

// summarize 把丢弃的中段消息压成一条摘要文本（带标记前缀）。
// Compressor 非空→LLM 摘要（参考原版 prompt），失败/空/无 Compressor→规则式摘要。
func (l *Loop) summarize(ctx context.Context, dropped []Message) string {
	if l.Compressor != nil {
		if s := l.llmSummarize(ctx, dropped); s != "" {
			return "[上下文已压缩 — LLM 摘要]\n\n" + s
		}
	}
	return "[上下文已压缩 — 规则摘要]\n\n" + ruleSummarize(dropped)
}

// llmSummarize 用压缩模型生成智能摘要——复刻参考 manager.ts:1126-1150：
// 取末 compactLLMSlice 条非 system、每条裁 200 字、role 中文化，套原版 prompt，单条 user 消息无工具。
func (l *Loop) llmSummarize(ctx context.Context, dropped []Message) string {
	start := 0
	if len(dropped) > compactLLMSlice {
		start = len(dropped) - compactLLMSlice
	}
	var conv strings.Builder
	for _, m := range dropped[start:] {
		if m.Role == RoleSystem {
			continue
		}
		role := "工具"
		switch m.Role {
		case RoleUser:
			role = "用户"
		case RoleAssistant:
			role = "助手"
		}
		toolInfo := ""
		if len(m.ToolCalls) > 0 {
			toolInfo = " [工具调用]"
		}
		conv.WriteString(role + ":" + toolInfo + " " + truncRunesAgent(m.Content, 200) + "\n")
	}
	prompt := "你是上下文压缩助手。根据以下对话内容，生成压缩摘要。保留：用户原始目标、已完成的关键操作与结论、待办事项、关键的推理要点。忽略错误输出。简洁，200 字以内。\n\n" +
		conv.String() + "\n---\n压缩摘要："
	msg, err := l.Compressor.Chat(ctx, []Message{{Role: RoleUser, Content: prompt}}, nil, nil)
	if err != nil {
		return "" // 压缩失败 → 回退规则式（复刻参考 catch）
	}
	s := strings.TrimSpace(msg.Content)
	if len([]rune(s)) < 10 { // 太短视作失败（复刻参考 length>10 判定）
		return ""
	}
	return s
}

// ruleSummarize 规则式拼接摘要（无压缩模型/压缩失败时的兜底）——复刻参考 manager.ts:1157-1189
// 的小节式结构，按 companion 可得信息裁剪：目标 / 进展 / 关键上下文 / 下一步。
func ruleSummarize(dropped []Message) string {
	var goal, lastAssistant string
	toolCalls, toolErrors := 0, 0
	files := map[string]bool{}
	for _, m := range dropped {
		switch m.Role {
		case RoleUser:
			if goal == "" && !strings.HasPrefix(m.Content, "[上下文已压缩") {
				goal = truncRunesAgent(m.Content, 200) // 首条用户消息 ≈ 原始目标
			}
		case RoleAssistant:
			if strings.TrimSpace(m.Content) != "" {
				lastAssistant = truncRunesAgent(m.Content, 400) // 末条有正文的助手消息 ≈ 关键上下文
			}
			toolCalls += len(m.ToolCalls)
			for _, tc := range m.ToolCalls {
				if p := argPath(tc.Function.Arguments); p != "" {
					files[p] = true
				}
			}
		case RoleTool:
			if strings.HasPrefix(strings.TrimSpace(m.Content), "Error:") {
				toolErrors++
			}
		}
	}
	var b strings.Builder
	if goal != "" {
		b.WriteString("## 目标\n" + goal + "\n\n")
	}
	fmt.Fprintf(&b, "## 进展\n已完成 %d 次工具调用", toolCalls)
	if toolErrors > 0 {
		fmt.Fprintf(&b, "（其中 %d 次错误）", toolErrors)
	}
	b.WriteString("。\n")
	if len(files) > 0 {
		b.WriteString("涉及文件: " + strings.Join(sortedKeys(files), ", ") + "\n")
	}
	if lastAssistant != "" {
		b.WriteString("\n## 关键上下文\n" + lastAssistant + "\n")
	}
	b.WriteString("\n## 下一步\n继续执行未完成的任务。")
	return b.String()
}

// estimateTokens 估算消息列表 token 数——复刻参考 manager.ts 启发式（无 tiktoken）：
// CJK 字 ×1.5、其余字符 ×0.25、每条 +4 开销、工具参数 ×0.25+8、tool_call_id +6。向上取整。
func estimateTokens(msgs []Message) int {
	total := 0.0
	for _, m := range msgs {
		total += 4
		total += textTokens(m.Content)
		total += textTokens(m.Reasoning)
		for _, tc := range m.ToolCalls {
			total += textTokens(tc.Function.Name)
			total += float64(len([]rune(tc.Function.Arguments)))*0.25 + 8
		}
		if m.ToolCallID != "" {
			total += 6
		}
	}
	return int(math.Ceil(total))
}

// textTokens 文本 token 估算：CJK 字 ×1.5、其余 ×0.25（复刻参考 CJK/拉丁分计）。
func textTokens(s string) float64 {
	cjk, other := 0, 0
	for _, r := range s {
		if isCJK(r) {
			cjk++
		} else {
			other++
		}
	}
	return float64(cjk)*1.5 + float64(other)*0.25
}

// isCJK 是否中日韩表意字（复刻参考正则 /[一-鿿㐀-䶿豈-﫿]/ 的 Unicode 区间）。
func isCJK(r rune) bool {
	switch {
	case r >= 0x4E00 && r <= 0x9FFF: // CJK 统一表意
		return true
	case r >= 0x3400 && r <= 0x4DBF: // 扩展 A
		return true
	case r >= 0xF900 && r <= 0xFAFF: // 兼容表意
		return true
	}
	return false
}

// prefixLen 开头连续 system 消息条数（压缩时恒留前缀，利于 KV 缓存复用）。
func prefixLen(msgs []Message) int {
	n := 0
	for n < len(msgs) && msgs[n].Role == RoleSystem {
		n++
	}
	return n
}

// argPath 从工具参数 JSON 取 path 字段（规则摘要列已访问文件用）。
func argPath(argsJSON string) string {
	var m map[string]any
	if json.Unmarshal([]byte(argsJSON), &m) == nil {
		if p, ok := m["path"].(string); ok {
			return strings.TrimSpace(p)
		}
	}
	return ""
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// truncRunesAgent 按 rune 截断（去首尾空白），超长加省略号。
func truncRunesAgent(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
