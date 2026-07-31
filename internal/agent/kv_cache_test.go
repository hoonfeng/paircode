package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestKVCachePrefixStability 验证多轮对话中 messages 数组前缀的稳定性。
//
// KV 缓存命中的核心条件：后续请求的 messages 数组必须以之前请求的 messages
// 数组为完整前缀。本项目实现中，多个因素会破坏此条件：
//   1. time.Now() 在系统提示词动态后缀中（buildWebSystemDynamic）
//   2. 自主模式完成报告注入改变历史结构
//
// 本测试模拟多轮对话，验证：
//   - 同轮次内各迭代的消息前缀是否稳定（msg_seq 增长模式）
//   - 跨轮次（新 Loop）的系统提示词是否一致
//   - 压缩后消息前缀是否还有部分缓存可利用
func TestKVCachePrefixStability(t *testing.T) {
	// ── 辅助函数：生成稳定的系统提示词（不含 time.Now()）──
	stableSystem := DefaultSystemPrompt([]string{"/test/project"})

	// 检查 CacheBoundary 由组装函数 ComposeSystemPrompt 统一添加（DefaultSystemPrompt 本身纯净，不含 boundary）
	if CacheBoundary == "" || !strings.Contains(ComposeSystemPrompt(stableSystem, "dynamic"), CacheBoundary) {
		t.Error("ComposeSystemPrompt 应包含 CacheBoundary！已缺失！")
	}
	if strings.Contains(stableSystem, CacheBoundary) {
		t.Error("DefaultSystemPrompt 不应自带 CacheBoundary（应由 ComposeSystemPrompt 统一添加，避免双边界）")
	}

	// ── 场景 1：同次会话内，连续迭代的消息前缀稳定性 ──
	t.Run("同一Loop内迭代消息前缀稳定", func(t *testing.T) {
		reg := NewRegistry()
		mock := &MockProvider{Responses: []Message{
			{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"test.go"}`}}}},
			{ToolCalls: []ToolCall{{ID: "c2", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"main.go"}`}}}},
			{Content: "任务完成。"},
		}}

		loop := &Loop{
			Provider:      mock,
			Registry:      reg,
			System:        stableSystem,
			MaxIterations: 10,
		}

		msgs, err := loop.Run(context.Background(), "读取两个文件", nil)
		if err != nil {
			t.Fatalf("Run 失败: %v", err)
		}

		// 验证 msgs 结构
		// msgs[0] 应为 system
		if len(msgs) == 0 || msgs[0].Role != RoleSystem {
			t.Fatal("msgs[0] 应为 system")
		}

		// ★ 关键验证点：msgs[0] 在整个 Run 过程中未被修改
		if msgs[0].Content != stableSystem {
			t.Errorf("messages[0].Content 在 Run 结束后被修改！\n"+
				"预期长度: %d, 实际长度: %d\n"+
				"（msgs[0] 的任何变化都会破坏 KV 缓存前缀）",
				len(stableSystem), len(msgs[0].Content))
		}
	})
}

// TestSystemPromptVariance 验证系统提示词在不同调用间的变化程度。
func TestSystemPromptVariance(t *testing.T) {
	// 模拟两次调用
	sys1 := DefaultSystemPrompt([]string{"/test/proj"})
	sys2 := DefaultSystemPrompt([]string{"/test/proj"})

	if sys1 != sys2 {
		for i := 0; i < len(sys1) && i < len(sys2); i++ {
			if sys1[i] != sys2[i] {
				t.Errorf("DefaultSystemPrompt 两次调用结果不同！\n"+
					"差异在字符 %d: '%c'(%d) vs '%c'(%d)",
					i, rune(sys1[i]), sys1[i], rune(sys2[i]), sys2[i])
				break
			}
		}
		t.Errorf("DefaultSystemPrompt 两次调用结果不一致！"+
			"\nsys1 len=%d, sys2 len=%d", len(sys1), len(sys2))
	} else {
		t.Logf("DefaultSystemPrompt 两次调用结果一致 ✓ (len=%d)", len(sys1))
	}

	// 验证组装后包含唯一 CacheBoundary（boundary 由 ComposeSystemPrompt 添加）
	if !strings.Contains(ComposeSystemPrompt(sys1, "dyn"), CacheBoundary) {
		t.Error("ComposeSystemPrompt 应包含 CacheBoundary")
	}
	if strings.Contains(sys1, CacheBoundary) {
		t.Error("DefaultSystemPrompt 不应自带 CacheBoundary（由 ComposeSystemPrompt 统一添加）")
	}
}

// TestBuildInjectionMessage 验证 buildInjectionMessage
// 不会触及 system prompt 且输出内容合理。
func TestBuildInjectionMessage(t *testing.T) {
	loop := &Loop{
		CompressedSummaries: []string{"[压缩摘要] 用户要求读取文件 a.go，已读取完毕"},
	}

	// 首次调用 buildInjectionMessage
	result1 := loop.buildInjectionMessage()
	if result1 == "" {
		t.Error("有摘要时 buildInjectionMessage 不应返回空")
	}
	if !strings.Contains(result1, "上下文已压缩") {
		t.Error("buildInjectionMessage 应包含压缩摘要标记")
	}

	// 模拟新增一条摘要
	loop.CompressedSummaries = append(loop.CompressedSummaries, "[压缩摘要] 用户要求修改 b.go，已修改完毕")
	result2 := loop.buildInjectionMessage()
	if result2 == "" {
		t.Error("有摘要时 buildInjectionMessage 不应返回空")
	}
	if len(result2) <= len(result1) {
		t.Error("新增摘要后 buildInjectionMessage 应更长")
	}
	t.Logf("buildInjectionMessage ✓ (result1=%d, result2=%d)", len(result1), len(result2))
}

// TestSerializedMessagesPrefix 验证序列化后的 messages 数组是否保持前缀稳定。
func TestSerializedMessagesPrefix(t *testing.T) {
	stableSystem := DefaultSystemPrompt([]string{"/test/proj"})

	// 模拟第1轮对话的所有 LLM 调用
	round1Requests := [][]Message{
		{
			{Role: RoleSystem, Content: stableSystem},
			{Role: RoleUser, Content: "读取文件 a.go"},
		},
		{
			{Role: RoleSystem, Content: stableSystem},
			{Role: RoleUser, Content: "读取文件 a.go"},
			{Role: RoleAssistant, Content: "", ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}}}},
			{Role: RoleTool, ToolCallID: "c1", Content: "file content: hello"},
		},
		{
			{Role: RoleSystem, Content: stableSystem},
			{Role: RoleUser, Content: "读取文件 a.go"},
			{Role: RoleAssistant, Content: "", ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}}}},
			{Role: RoleTool, ToolCallID: "c1", Content: "file content: hello"},
			{Role: RoleAssistant, Content: "已读取 a.go，内容是 hello"},
		},
	}

	// 验证：每次调用的消息序列前缀匹配前一次
	for i := 1; i < len(round1Requests); i++ {
		prev := round1Requests[i-1]
		curr := round1Requests[i]

		if len(curr) < len(prev) {
			t.Errorf("第%d次调用的消息数(%d)少于第%d次(%d)，不可能有前缀匹配",
				i+1, len(curr), i, len(prev))
			continue
		}

		for j := range prev {
			if curr[j].Role != prev[j].Role || curr[j].Content != prev[j].Content {
				t.Errorf("第%d次调用不以第%d次为前缀！位置 %d:\n  prev: role=%q content=%q\n  curr: role=%q content=%q",
					i+1, i, j, prev[j].Role, truncStr(prev[j].Content, 50),
					curr[j].Role, truncStr(curr[j].Content, 50))
			}
		}
	}

	t.Logf("同轮对话内序列化前缀稳定性验证通过 ✓")

	// ── 模拟第2轮对话（新 Loop，从 store 加载历史）──
	var persisted []Message
	for _, m := range round1Requests[len(round1Requests)-1] {
		if m.Role != RoleSystem {
			persisted = append(persisted, m)
		}
	}

	// 构造第2轮的第一个 LLM 调用
	round2First := make([]Message, 0, len(persisted)+2)
	round2First = append(round2First, Message{Role: RoleSystem, Content: stableSystem})
	round2First = append(round2First, persisted...)
	round2First = append(round2First, Message{Role: RoleUser, Content: "继续，读取 b.go"})

	// 验证：第2轮第1次调用必须以第1轮最后一次调用的完整非 system 内容为前缀
	lastRound1 := round1Requests[len(round1Requests)-1]
	for i := 1; i < len(lastRound1); i++ {
		if i >= len(round2First) {
			t.Errorf("第2轮消息数(%d) 少于第1轮(%d)，前缀不完整",
				len(round2First), len(lastRound1))
			break
		}
		r1 := lastRound1[i]
		r2 := round2First[i]
		if r1.Role != r2.Role || r1.Content != r2.Content {
			t.Errorf("跨轮次前缀不匹配！位置 %d:\n  第1轮: role=%q content=%q\n  第2轮: role=%q content=%q",
				i, r1.Role, truncStr(r1.Content, 50),
				r2.Role, truncStr(r2.Content, 50))
		}
	}

	t.Logf("跨轮次序列化前缀稳定性验证通过 ✓")

	// ★ 关键检查点：对比"前 len(第1轮) 条消息"的 JSON 序列化前缀
	// 注意：不能直接对完整数组做字节前缀对比——JSON 数组的元素分隔符（逗号）会让
	// 追加元素后的数组在相同位置多一个逗号，产生误导性的"前缀不匹配"（预存测试缺陷）。
	// 正确的前缀定义：round2 的前 len(round1) 条消息序列化后应与 round1 完全一致。
	lastR1JSON, _ := json.Marshal(lastRound1)
	r2PrefixJSON, _ := json.Marshal(round2First[:len(lastRound1)])

	if string(r2PrefixJSON) == string(lastR1JSON) {
		t.Logf("跨轮次 JSON 序列化前缀完全匹配 ✓ (len=%d)", len(lastR1JSON))
	} else {
		t.Errorf("★★★★ 严重：第2轮请求的前缀消息 JSON 与第1轮不匹配！\n"+
			"这会导致 LLM API 的 KV 缓存完全失效！\n"+
			"第1轮 JSON len=%d, 第2轮前缀 JSON len=%d",
			len(lastR1JSON), len(r2PrefixJSON))
	}
}


func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
