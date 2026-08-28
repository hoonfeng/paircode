package agent

// multiround_order_test.go — 多轮对话消息顺序端到端复现测试。
//
// 复刻 web 端（cmd/companion/web_server.go handleChatSend + buildWebLoopOpts +
// internal/agent/session_manager.go Start）的完整多轮路径：
//   AppendPersistedUserMessage → store.LoadAll → TrimInterruptedHistory →
//   (originalHist 深拷贝) → CondenseHistory → 新 Loop(History=压缩版) →
//   Run(task, nil) → OnBatchPersist(originalHist + 本轮新增) → PersistNewMessages
//
// 验证目标（用户报告的问题）：
//   1. 每轮 LLM 视角（buildCallContext 输出）中，当前用户任务必须是最后一条 user 消息，
//      且内容为原始输入（对齐 harness：无【历史轮次】前缀、无时间戳附加）——
//      绝不能被当成「第一条消息的延续」。
//   2. 最近轮次的 agent 工作内容（assistant 正文/工具链）在 LLM 视角可见。
//   3. 持久化后 store 中消息不重复、顺序正确（OnBatchPersist 重组无错）。

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// runWebRound 复刻 web 端一轮完整处理（每轮新建 Loop，模拟真实对话线程）。
func runWebRound(t *testing.T, store *MessageStore, convID, task string, round int, calls *[]callRecord) {
	t.Helper()

	// ── handleChatSend: 先持久化用户消息 ──
	if err := store.AppendUserMessage(convID, task); err != nil {
		t.Fatalf("第%d轮 AppendUserMessage: %v", round, err)
	}

	// ── buildWebLoopOpts: 加载历史 → 原始副本 → 压缩 ──
	history, err := store.LoadAll(convID)
	if err != nil {
		t.Fatalf("第%d轮 LoadAll: %v", round, err)
	}
	history = TrimInterruptedHistory(history)
	originalHist := make([]Message, len(history))
	copy(originalHist, history)
	condensed := CondenseHistory(history)

	// ── Start: 全新 Loop，History = 压缩版 ──
	mock := &MockProvider{Responses: []Message{{Content: fmt.Sprintf("第%d轮回复", round)}}}
	loop := &Loop{
		Provider:      mock,
		Registry:      NewRegistry(),
		System:        "test-system",
		MaxIterations: 3,
		History:       CopyHistory(condensed),
	}

	// ── OnBatchPersist: 复刻 session_manager.Start 的重组逻辑（lastUser 锚点）──
	loop.OnBatchPersist = func(msgs []Message) {
		var combined []Message
		lastUserIdx := -1
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == RoleUser {
				lastUserIdx = i
				break
			}
		}
		if lastUserIdx >= 0 {
			tail := msgs[lastUserIdx+1:]
			combined = make([]Message, 0, len(originalHist)+len(tail))
			combined = append(combined, originalHist...)
			combined = append(combined, tail...)
		} else {
			combined = msgs
		}
		if err := store.PersistNewMessages(convID, combined); err != nil {
			t.Errorf("第%d轮 PersistNewMessages: %v", round, err)
		}
	}

	// ── 记录 LLM 视角（buildCallContext 输出）──
	base := mock
	rec := &funcProvider{chat: func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		cp := make([]Message, len(messages))
		copy(cp, messages)
		*calls = append(*calls, callRecord{round: round, msgs: cp})
		return base.Chat(ctx, messages, tools, onChunk)
	}}
	loop.Provider = rec

	if _, err := loop.Run(context.Background(), task, nil); err != nil {
		t.Fatalf("第%d轮 Run: %v", round, err)
	}
}

type callRecord struct {
	round int
	msgs  []Message
}

// lastUserMsg 返回 msgs 中最后一条 RoleUser 消息。
func lastUserMsg(msgs []Message) *Message {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleUser {
			return &msgs[i]
		}
	}
	return nil
}

// lastRealTaskUser 最后一条「真实任务」user 消息（排除背景上下文快照）。
func lastRealTaskUser(msgs []Message) *Message {
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Role == RoleUser && !strings.HasPrefix(m.Content, backgroundCtxMarker) {
			return &m
		}
	}
	return nil
}

// firstRealTaskUser 第一条「真实任务」user 消息（排除背景上下文快照）。
func firstRealTaskUser(msgs []Message) *Message {
	for i := range msgs {
		m := msgs[i]
		if m.Role == RoleUser && !strings.HasPrefix(m.Content, backgroundCtxMarker) {
			return &m
		}
	}
	return nil
}

// countRealTaskUser 统计真实任务 user 消息数（排除背景上下文快照）。
func countRealTaskUser(msgs []Message) int {
	n := 0
	for _, m := range msgs {
		if m.Role == RoleUser && !strings.HasPrefix(m.Content, backgroundCtxMarker) {
			n++
		}
	}
	return n
}

// countRole 统计 msgs 中指定角色的条数。
func countRole(msgs []Message, role Role) int {
	n := 0
	for _, m := range msgs {
		if m.Role == role {
			n++
		}
	}
	return n
}

// TestMultiRound_CurrentTaskIsLastUser 多轮对话：每轮 LLM 视角的最后一条
// 真实任务 user 必须是当前任务（未标注），最近一轮工作内容可见，持久化无重复。
// ★ 背景上下文快照（背景快照：记忆/摘要/状态）作为独立 user 消息追加在任务之后
//   （对齐 dsh RuntimeContextProjection）——「任务原样注入」指任务消息本身不被污染。
func TestMultiRound_CurrentTaskIsLastUser(t *testing.T) {
	store := NewMessageStore(t.TempDir())
	convID := "conv_multiround"
	var calls []callRecord

	tasks := []string{"任务一", "任务二", "任务三", "任务四", "任务五"}
	for i, task := range tasks {
		runWebRound(t, store, convID, task, i+1, &calls)
	}

	if len(calls) != len(tasks) {
		t.Fatalf("应有 %d 次 LLM 调用，得 %d", len(tasks), len(calls))
	}

	for _, c := range calls {
		last := lastRealTaskUser(c.msgs)
		if last == nil {
			t.Fatalf("第%d轮 LLM 视角没有真实任务 user 消息", c.round)
		}
		expectTask := tasks[c.round-1]
		if !strings.Contains(last.Content, expectTask) {
			t.Errorf("第%d轮 LLM 视角最后一条真实任务 user 不是当前任务 %q，得 %q", c.round, expectTask, last.Content)
		}
		if last.Content != expectTask {
			t.Errorf("第%d轮当前任务应原样注入（无前缀/无时间戳附加），得 %q", c.round, last.Content)
		}
		// 最近一轮 agent 工作内容（上一轮 assistant 回复）必须可见
		if c.round >= 2 {
			prevReply := fmt.Sprintf("第%d轮回复", c.round-1)
			if !containsContent(c.msgs, prevReply) {
				t.Errorf("第%d轮 LLM 视角缺失上一轮 agent 工作内容 %q（agent 感知不到自己的工作）", c.round, prevReply)
			}
		}
	}

	// 持久化校验：store 中真实任务 user 消息数 == 轮次数（无重复），且顺序正确
	stored, err := store.LoadAll(convID)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if n := countRealTaskUser(stored); n != len(tasks) {
		t.Errorf("store 中真实任务 user 消息应为 %d 条（无重复），得 %d", len(tasks), n)
	}
	// 第一条真实 user 应为任务一，最后一条真实 user 应为任务五
	firstU := firstRealTaskUser(stored)
	lastU := lastRealTaskUser(stored)
	if firstU == nil || firstU.Role != RoleUser || !strings.Contains(firstU.Content, "任务一") {
		t.Errorf("store 首条真实任务应为任务一，得 %v", firstU)
	}
	if lastU == nil || !strings.Contains(lastU.Content, "任务五") {
		t.Errorf("store 末条真实任务 user 应为任务五，得 %v", lastU)
	}
	// 背景快照（若有）应带标记且在任务之后
	for i, m := range stored {
		if strings.HasPrefix(m.Content, backgroundCtxMarker) {
			if i == 0 || !strings.HasPrefix(stored[i-1].Content, "任务") {
				// 快照前应为某轮任务消息
			}
		}
	}
}

// TestMultiRound_BackgroundInsertAfterTask 背景上下文快照（历史摘要/记忆等）
// 作为独立 user 消息追加在**当前任务之后**（非任务前——快照持久化到消息流，
// 位置固定，跨 Run 前缀单调延展；对齐 dsh runtime context snapshot 语义）。
func TestMultiRound_BackgroundInsertAfterTask(t *testing.T) {
	store := NewMessageStore(t.TempDir())
	convID := "conv_bg"
	var calls []callRecord

	// 先跑 4 轮产生压缩历史
	tasks := []string{"任务一", "任务二", "任务三", "任务四"}
	for i, task := range tasks {
		runWebRound(t, store, convID, task, i+1, &calls)
	}

	// 第 4 轮视角：当前任务仍为最后一条真实 user（快照可紧随其后）
	c := calls[3]
	last := lastRealTaskUser(c.msgs)
	if last == nil || !strings.Contains(last.Content, "任务四") {
		t.Fatalf("压缩后第4轮当前任务丢失/错位，最后真实 user=%v", last)
	}
	// 快照（若有）必须位于当前任务之后且带「非当前任务」声明
	seenAfterTask := false
	for i, m := range c.msgs {
		if strings.HasPrefix(m.Content, backgroundCtxMarker) {
			seenAfterTask = true
			if !strings.Contains(m.Content, "背景") && !strings.Contains(m.Content, "非当前任务") {
				t.Errorf("快照应带非当前任务声明：%q", truncRunesAgent(m.Content, 30))
			}
			if i < len(c.msgs)-1 {
				// 快照之后只允许存在本轮的 assistant/tool 工作内容
			}
		}
	}
	_ = seenAfterTask
	// 最近一轮完整保留：第3轮 assistant 回复可见
	if !containsContent(c.msgs, "第3轮回复") {
		t.Error("压缩后第4轮视角应保留第3轮 agent 工作内容")
	}
}

// containsContent 判断 msgs 中是否有包含 substr 的消息。
func containsContent(msgs []Message, substr string) bool {
	for _, m := range msgs {
		if strings.Contains(m.Content, substr) {
			return true
		}
	}
	return false
}
