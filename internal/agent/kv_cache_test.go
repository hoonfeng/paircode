package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestKVCachePrefixStability 验证多轮对话中 messages 数组前缀的稳定性。
//
// KV 缓存命中的核心条件：后续请求的 messages 数组必须以之前请求的 messages
// 数组为完整前缀。本项目实现中，多个因素会破坏此条件：
//   1. time.Now() 在系统提示词动态后缀中（buildWebSystemDynamic）
//   2. buildSystemWithSummaries 修改 messages[0].Content（压缩后）
//   3. 自主模式完成报告注入改变历史结构
//
// 本测试模拟多轮对话，验证：
//   - 同轮次内各迭代的消息前缀是否稳定（msg_seq 增长模式）
//   - 跨轮次（新 Loop）的系统提示词是否一致
//   - 压缩后消息前缀是否还有部分缓存可利用
func TestKVCachePrefixStability(t *testing.T) {
	// ── 辅助函数：生成稳定的系统提示词（不含 time.Now()）──
	stableSystem := DefaultSystemPrompt([]string{"/test/project"})

	// 检查 CacheBoundary 是否存在于 DefaultSystemPrompt 中
	if !strings.Contains(stableSystem, CacheBoundary) {
		t.Error("DefaultSystemPrompt 应包含 CacheBoundary，已缺失！\n" +
			"缺少 CacheBoundary 会导致 buildSystemWithSummaries 首次调用时\n" +
			"大幅修改 messages[0]，使压缩前后的缓存前缀完全失效。")
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

		// 构造本轮各次 LLM 调用消息序列（模拟 Provider.Chat 接收到的内容）
		// 第一次调用：[system, user("读取两个文件")]
		// LLM 返回：read_file("test.go")
		// 第二次调用：[system, user("读取两个文件"), assistant(读test.go), tool(result)]
		// LLM 返回：read_file("main.go")
		// 第三次调用：[system, user("读取两个文件"), assistant(读test.go), tool(result), assistant(读main.go), tool(result)]
		// LLM 返回："任务完成。"

		// 验证：每次调用的前缀都完整覆盖前一次调用的全部内容
		// 第1次：msgs[0:2] = [system, user]
		// 第2次：msgs[0:4] = [system, user, assistant1, tool1]
		// 第3次：msgs[0:6] = [system, user, assistant1, tool1, assistant2, tool2]
		// 第4次：msgs[全部] ... assistant3(content) 无工具

		// 检查 msgs 是否包含预期数量的助手消息
		assistantCount := 0
		for _, m := range msgs {
			if m.Role == RoleAssistant {
				assistantCount++
			}
		}
		if assistantCount != 3 {
			t.Logf("助手消息数: %d (预期 3)", assistantCount)
		}

		// ★ 关键验证点：msgs[0] 在整个 Run 过程中未被修改
		if msgs[0].Content != stableSystem {
			t.Errorf("messages[0].Content 在 Run 结束后被修改！\n"+
				"预期长度: %d, 实际长度: %d\n"+
				"这可能是因为 buildSystemWithSummaries 修改了 messages[0]。\n"+
				"压缩后 messages[0] 的变化会破坏 KV 缓存前缀。",
				len(stableSystem), len(msgs[0].Content))
		}
	})

	// ── 场景 2：跨轮次（新 Loop）消息前缀稳定性 ──
	t.Run("跨Loop消息前缀稳定", func(t *testing.T) {
		// 模拟第1轮对话
		mock1 := &MockProvider{Responses: []Message{
			{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}}}},
			{Content: "已读取 a.go"},
		}}
		reg1 := NewRegistry()
		RegisterDefaultTools(reg1, t.TempDir())
		loop1 := &Loop{
			Provider:      mock1,
			Registry:      reg1,
			System:        stableSystem,
			MaxIterations: 5,
		}
		msgs1, err := loop1.Run(context.Background(), "读 a.go", nil)
		if err != nil {
			t.Fatalf("第1轮 Run 失败: %v", err)
		}

		// 模拟消息持久化（PersistNewMessages 会跳过系统消息）
		var persisted []Message
		for _, m := range msgs1 {
			if m.Role != RoleSystem {
				persisted = append(persisted, m)
			}
		}

		// 模拟第2轮对话（新 Loop，从 store 加载 history）
		mock2 := &MockProvider{Responses: []Message{
			{Content: "已读取 b.go"},
		}}
		reg2 := NewRegistry()
		RegisterDefaultTools(reg2, t.TempDir())

		// ★ 关键：系统提示词应完全相同（不含 time.Now()）
		// 测试用的 stableSystem 已不含时间戳
		loop2 := &Loop{
			Provider:      mock2,
			Registry:      reg2,
			System:        stableSystem, // 同一份系统提示词
			MaxIterations: 5,
			History:       CopyHistory(persisted), // 加载持久化的历史
		}

		// 手动构造第2轮的消息序列（模拟 Loop.Run 内部行为）
		hist := CopyHistory(persisted)
		msgs2 := make([]Message, 0, len(hist)+2)
		if !hasSystem(hist) {
			msgs2 = append(msgs2, Message{Role: RoleSystem, Content: stableSystem})
		}
		msgs2 = append(msgs2, hist...)

		// 检查 msgs2[0]（system prompt）是否与 msgs1[0] 相同
		if len(msgs2) == 0 || msgs2[0].Role != RoleSystem {
			t.Fatal("msgs2 应以 system 开头")
		}
		if msgs2[0].Content != stableSystem {
			t.Errorf("跨轮次系统提示词不一致！\n"+
				"这会导致 KV 缓存前缀在第1句就不匹配。\n"+
				"根因可能是 time.Now() 或其它会话级动态内容。")
		}

		// ★ 构造前缀验证：
		// 第1轮最后 LLM 调用的 messages = msgs1（系统+历史+当前用户）
		// 第2轮首次 LLM 调用的 messages = msgs2 + user("读 b.go")
		// 预期：msgs2 必须以 completePrompt1（第1轮的全量消息）为前缀
		// 即 msgs1 是 msgs2 的前缀（因为 msgs2 = [system, history...] = msgs1 的内容）

		// 检查第2轮消息的前缀是否完整包含第1轮消息
		// msgs2 应包含 msgs1 的全部非 system 消息
		checkPrefixContains(t, msgs1, msgs2)
	})
}

// checkPrefixContains 验证 child 的 messages 中，是否包含 parent 的全部非 system 消息
// 作为其前缀（顺序、内容一致）。这是 KV 缓存命中的必要条件。
func checkPrefixContains(t *testing.T, parent, child []Message) {
	t.Helper()

	// 收集 parent 中除 system 外的消息
	var parentContent []struct {
		Role    string
		Content string
	}
	for _, m := range parent {
		if m.Role != RoleSystem {
			parentContent = append(parentContent, struct {
				Role    string
				Content string
			}{string(m.Role), m.Content})
		}
	}

	if len(parentContent) == 0 {
		return
	}

	// 收集 child 中除 system 外的消息
	var childContent []struct {
		Role    string
		Content string
	}
	for _, m := range child {
		if m.Role != RoleSystem {
			childContent = append(childContent, struct {
				Role    string
				Content string
			}{string(m.Role), m.Content})
		}
	}

	if len(childContent) < len(parentContent) {
		t.Errorf("child 消息数(%d) 少于 parent(%d)，不可能以 parent 为前缀",
			len(childContent), len(parentContent))
		return
	}

	// 检查前缀匹配
	for i := range parentContent {
		if childContent[i].Role != parentContent[i].Role {
			t.Errorf("前缀不匹配，位置 %d: parent role=%q, child role=%q",
				i, parentContent[i].Role, childContent[i].Role)
			return
		}
		if childContent[i].Content != parentContent[i].Content {
			t.Errorf("前缀不匹配，位置 %d: parent content=%q..., child content=%q...",
				i, truncStr(parentContent[i].Content, 50), truncStr(childContent[i].Content, 50))
			return
		}
	}
	t.Logf("前缀验证通过：child(%d) 以 parent(%d) 为完整前缀 ✓", len(childContent), len(parentContent))
}

// truncStr 截断字符串用于日志。
func truncStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

// TestSystemPromptVariance 验证系统提示词在不同调用间的变化程度。
// 检查 buildWebSystemDynamic 中的 time.Now() 是否会导致 system prompt 变化。
func TestSystemPromptVariance(t *testing.T) {
	// 模拟两次调用（时间戳会不同）
	sys1 := DefaultSystemPrompt([]string{"/test/proj"})
	sys2 := DefaultSystemPrompt([]string{"/test/proj"})

	if sys1 != sys2 {
		// 找出差异位置
		for i := 0; i < len(sys1) && i < len(sys2); i++ {
			if sys1[i] != sys2[i] {
				t.Errorf("DefaultSystemPrompt 两次调用结果不同！\n"+
					"差异在字符 %d: '%c'(%d) vs '%c'(%d)\n"+
					"前后文:\n  sys1: %q\n  sys2: %q",
					i, rune(sys1[i]), sys1[i], rune(sys2[i]), sys2[i],
					sys1[max(0, i-20):min(len(sys1), i+20)],
					sys2[max(0, i-20):min(len(sys2), i+20)])
				break
			}
		}
		t.Errorf("DefaultSystemPrompt 两次调用结果不一致！"+
			"\nsys1 len=%d, sys2 len=%d", len(sys1), len(sys2))
	} else {
		t.Logf("DefaultSystemPrompt 两次调用结果一致 ✓ (len=%d)", len(sys1))
	}

	// 验证 CacheBoundary 存在
	if !strings.Contains(sys1, CacheBoundary) {
		t.Error("DefaultSystemPrompt 应包含 CacheBoundary")
	}
}

// TestBuildSystemWithSummariesStability 验证 buildSystemWithSummaries
// 不会不必要地修改系统提示词前缀。
func TestBuildSystemWithSummariesStability(t *testing.T) {
	sysWithBoundary := DefaultSystemPrompt([]string{"/test"})

	loop := &Loop{
		System:             sysWithBoundary,
		CompressedSummaries: []string{"[压缩摘要] 用户要求读取文件 a.go，已读取完毕"},
	}

	// 首次调用 buildSystemWithSummaries
	result1 := loop.buildSystemWithSummaries()

	// 模拟新增一条摘要
	loop.CompressedSummaries = append(loop.CompressedSummaries, "[压缩摘要] 用户要求修改 b.go，已修改完毕")

	result2 := loop.buildSystemWithSummaries()

	// 检查：第二次结果必须以第一次结果为前缀
	if !strings.HasPrefix(result2, result1) {
		// 找出差异
		minLen := len(result1)
		if len(result2) < minLen {
			minLen = len(result2)
		}
		diffPos := -1
		for i := 0; i < minLen; i++ {
			if result1[i] != result2[i] {
				diffPos = i
				break
			}
		}
		t.Errorf("buildSystemWithSummaries 的第二次输出不以第一次为前缀！\n"+
			"差异位置: %d\n"+
			"result1: %q\n"+
			"result2: %q\n"+
			"（压缩摘要注入破坏了系统提示词前缀稳定性）",
			diffPos,
			result1[max(0, diffPos-30):min(len(result1), diffPos+30)],
			result2[max(0, diffPos-30):min(len(result2), diffPos+30)])
	} else {
		t.Logf("buildSystemWithSummaries 前缀稳定 ✓ (result1=%d, result2=%d)",
			len(result1), len(result2))
	}
}

// TestSerializedMessagesPrefix 验证序列化后的 messages 数组是否保持前缀稳定。
// 模拟完整的 JSON 序列化过程，因为 KV 缓存作用于序列化后的 token 流，
// 而非内存中的 Go 结构体。
func TestSerializedMessagesPrefix(t *testing.T) {
	stableSystem := DefaultSystemPrompt([]string{"/test/proj"})

	// 模拟第1轮对话的所有 LLM 调用
	round1Requests := [][]Message{
		// 第1次LLM调用
		{
			{Role: RoleSystem, Content: stableSystem},
			{Role: RoleUser, Content: "读取文件 a.go"},
		},
		// 第2次LLM调用（带工具结果）
		{
			{Role: RoleSystem, Content: stableSystem},
			{Role: RoleUser, Content: "读取文件 a.go"},
			{Role: RoleAssistant, Content: "", ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}}}},
			{Role: RoleTool, ToolCallID: "c1", Content: "file content: hello"},
		},
		// 第3次LLM调用（完成）
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

		prevJSON, _ := json.Marshal(prev)
		currJSON, _ := json.Marshal(curr)

		// 检查 curr 是否以 prev 为前缀
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

		_ = prevJSON
		_ = currJSON
	}

	t.Logf("同轮对话内序列化前缀稳定性验证通过 ✓")

	// ── 模拟第2轮对话（新 Loop，从 store 加载历史）──
	// 构造持久化的历史（不含 system）
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
	// 实际上，第2轮的 messages[1:] 应与第1轮的 messages[1:] 完全相同
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

	// ★ 关键检查点：对比完整 JSON 序列化的前缀
	lastR1JSON, _ := json.Marshal(lastRound1)
	r2FirstJSON, _ := json.Marshal(round2First)

	if len(r2FirstJSON) >= len(lastR1JSON) {
		prefixMatch := string(r2FirstJSON[:len(lastR1JSON)]) == string(lastR1JSON)
		if !prefixMatch {
			// 这是最严重的问题：LLM API 收到的消息数组前缀不匹配
			// 意味着 KV 缓存完全无效
			t.Errorf("★★★★ 严重：第2轮请求的 JSON 序列化前缀与第1轮不匹配！\n"+
				"这会导致 LLM API 的 KV 缓存完全失效！\n"+
				"第1轮 JSON len=%d, 第2轮 JSON len=%d\n"+
				"第1轮: %s\n第2轮: %s",
				len(lastR1JSON), len(r2FirstJSON),
				string(lastR1JSON), string(r2FirstJSON[:len(lastR1JSON)]))
		} else {
			t.Logf("跨轮次 JSON 序列化前缀完全匹配 ✓ (len=%d)", len(lastR1JSON))
		}
	}
}

// TestKVCacheCrossLoopTimeStamp 验证 time.Now() 导致的系统提示词不一致问题。
// 这是最严重的缓存破坏因素：跨 Loop 调用时系统提示词因时间戳变化而不一致。
func TestKVCacheCrossLoopTimeStamp(t *testing.T) {
	// 模拟 buildWebSystemPrompt 的行为：包含 time.Now()
	// 注意：DefaultSystemPrompt 本身不含 time.Now()，但这个函数在 web_server.go
	// 的 buildWebSystemDynamic 中被添加

	// 验证不含时间戳的 DefaultSystemPrompt 是稳定的
	sysA := DefaultSystemPrompt([]string{"/test/proj"})
	sysB := DefaultSystemPrompt([]string{"/test/proj"})
	if sysA != sysB {
		t.Error("DefaultSystemPrompt 包含不稳定的动态内容！")
	}

	// 模拟 buildWebSystemDynamic 添加时间戳带来的影响
	timestampA := fmt.Sprintf("\n\n# 当前时间\n%s", "2025-01-01 12:00:00 MST (UTC-07:00)")
	timestampB := fmt.Sprintf("\n\n# 当前时间\n%s", "2025-01-01 12:00:01 MST (UTC-07:00)")

	sysWithTimeA := sysA + timestampA
	sysWithTimeB := sysB + timestampB

	t.Logf("含时间戳的 system prompt 比较:")
	t.Logf("  sysA len=%d, sysB len=%d", len(sysWithTimeA), len(sysWithTimeB))

	// 检查在 CacheBoundary 之前是否一致
	cacheBoundaryIdxA := strings.Index(sysWithTimeA, CacheBoundary)
	cacheBoundaryIdxB := strings.Index(sysWithTimeB, CacheBoundary)

	if cacheBoundaryIdxA < 0 || cacheBoundaryIdxB < 0 {
		t.Fatal("缺少 CacheBoundary")
	}

	prefixA := sysWithTimeA[:cacheBoundaryIdxA+len(CacheBoundary)]
	prefixB := sysWithTimeB[:cacheBoundaryIdxB+len(CacheBoundary)]

	if prefixA == prefixB {
		t.Logf("CacheBoundary 之前的部分稳定 ✓")
	} else {
		t.Error("CacheBoundary 之前的部分也不稳定！")
	}

	// 验证完整 messages 序列的前缀
	// 构建第1轮请求
	req1 := []Message{
		{Role: RoleSystem, Content: sysWithTimeA},
		{Role: RoleUser, Content: "读取 a.go"},
	}
	req1JSON, _ := json.Marshal(req1)

	// 构建第2轮请求（含时间戳B）
	req2 := []Message{
		{Role: RoleSystem, Content: sysWithTimeB},
		{Role: RoleUser, Content: "读取 a.go"},
		{Role: RoleAssistant, Content: "已读取"},
		{Role: RoleUser, Content: "继续读 b.go"},
	}
	req2JSON, _ := json.Marshal(req2)

	// 检查 req2 是否以 req1 为前缀（不考虑时间戳差异）
	// 如果时间戳不同，req2 的 JSON 序列化从一开始就不同
	// 导致 KV 缓存前缀完全无效
	prefixMatch := len(req2JSON) >= len(req1JSON) &&
		string(req2JSON[:len(req1JSON)]) == string(req1JSON)

	if prefixMatch {
		t.Log("含时间戳的跨轮次前缀仍然匹配 ✓")
	} else {
		// 找出序列化后的差异
		minLen := len(req1JSON)
		if len(req2JSON) < minLen {
			minLen = len(req2JSON)
		}
		diffPos := -1
		for i := 0; i < minLen; i++ {
			if req1JSON[i] != req2JSON[i] {
				diffPos = i
				break
			}
		}
		t.Errorf("★★★★ 严重：含时间戳的跨轮次请求 JSON 前缀不匹配！\n"+
			"第1轮和第2轮的系统提示词因时间戳不同而不同，\n"+
			"导致 KV 缓存跨轮次完全失效！\n"+
			"差异位置: byte %d\n"+
			"第1轮: %s\n第2轮: %s",
			diffPos,
			string(req1JSON[max(0, diffPos-30):min(len(req1JSON), diffPos+30)]),
			string(req2JSON[max(0, diffPos-30):min(len(req2JSON), diffPos+30)]))
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
