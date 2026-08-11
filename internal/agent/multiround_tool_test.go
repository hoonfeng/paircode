package agent

// multiround_tool_test.go — 带工具调用的多轮端到端测试。
// 复现真实场景：每轮 user 消息后 LLM 先返回带 tool_call 的 assistant，
// 执行工具后返回最终 assistant。验证 OnBatchPersist 持久化后
// 「user 后必有 assistant」且顺序正确（无 assistant 丢失、无 user→tool 粘连）。

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// toolProvider 按轮返回 [tool_call assistant, 最终 assistant]。
type toolProvider struct {
	rounds int
	call   int
}

func (p *toolProvider) Name() string { return "tool" }

func (p *toolProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	p.call++
	// 每轮 2 次调用：先 tool_call，再最终回复
	if p.call%2 == 1 {
		msg := Message{
			Role:    RoleAssistant,
			Content: fmt.Sprintf("第%d轮 工具调用计划", p.rounds),
			ToolCalls: []ToolCall{{
				ID:   fmt.Sprintf("call_%d_%d", p.rounds, p.call),
				Type: "function",
				Function: FunctionCall{
					Name:      "test_echo",
					Arguments: fmt.Sprintf(`{"text":"第%d轮工具输入"}`, p.rounds),
				},
			}},
		}
		if onChunk != nil {
			onChunk(Chunk{Content: msg.Content, ToolCalls: msg.ToolCalls, Done: true})
		}
		return msg, nil
	}
	msg := Message{Role: RoleAssistant, Content: fmt.Sprintf("第%d轮最终回复", p.rounds)}
	if onChunk != nil {
		onChunk(Chunk{Content: msg.Content, Done: true})
	}
	return msg, nil
}

// runToolRound 复刻 web 端一轮（store 持久化 + 新 Loop + OnBatchPersist 重组）。
func runToolRound(t *testing.T, store *MessageStore, convID, task string, round int) {
	t.Helper()
	if err := store.AppendUserMessage(convID, task); err != nil {
		t.Fatalf("第%d轮 AppendUserMessage: %v", round, err)
	}
	history, err := store.LoadAll(convID)
	if err != nil {
		t.Fatalf("第%d轮 LoadAll: %v", round, err)
	}
	history = TrimInterruptedHistory(history)
	originalHist := make([]Message, len(history))
	copy(originalHist, history)
	condensed := CondenseHistory(history)

	reg := NewRegistry()
	reg.Register(&Tool{
		Name:        "test_echo",
		Description: "回显文本",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}}},
		Handler: func(hctx context.Context, args map[string]any) (string, error) {
			text, _ := args["text"].(string)
			return "echo: " + text, nil
		},
	})

	prov := &toolProvider{rounds: round}
	loop := &Loop{
		Provider:      prov,
		Registry:      reg,
		System:        "test-system",
		MaxIterations: 3,
		History:       CopyHistory(condensed),
	}
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
	if _, err := loop.Run(context.Background(), task, nil); err != nil {
		t.Fatalf("第%d轮 Run: %v", round, err)
	}
}

// TestMultiRound_WithToolCalls_AssistantNeverLost 三轮工具调用对话，
// 每轮后验证 store 结构：user 后必有 assistant（无 user→tool 粘连），
// 且最新 user 消息在 store 末尾（非首条）。
func TestMultiRound_WithToolCalls_AssistantNeverLost(t *testing.T) {
	store := NewMessageStore(t.TempDir())
	convID := "conv_tools"

	tasks := []string{"任务一", "任务二", "任务三", "任务四"}
	for i, task := range tasks {
		runToolRound(t, store, convID, task, i+1)
	}

	stored, err := store.LoadAll(convID)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	t.Logf("store 共 %d 条消息", len(stored))

	// 结构断言：每条 user 后必须紧跟 assistant（可多轮迭代）
	userCount := 0
	for i, m := range stored {
		if m.Role != RoleUser {
			continue
		}
		userCount++
		if i+1 >= len(stored) {
			t.Errorf("第 %d 条 user（%q）是最后一条消息——其后应有 assistant 回复", i, shortContent(m.Content))
			continue
		}
		if stored[i+1].Role != RoleAssistant {
			t.Errorf("第 %d 条 user（%q）后紧跟 %s（%q）——assistant 丢失，user 与 tool 粘连",
				i, shortContent(m.Content), stored[i+1].Role, shortContent(stored[i+1].Content))
		}
	}
	if userCount != len(tasks) {
		t.Errorf("user 消息数应为 %d，得 %d", len(tasks), userCount)
	}

	// 最新用户消息必须在 store 末尾（且其前一条是 assistant 回复）
	last := stored[len(stored)-1]
	if last.Role == RoleUser {
		t.Errorf("store 末条是 user（%q）——最新任务后无 assistant 回复", shortContent(last.Content))
	}
	lastUser := lastUserMsg(stored)
	if lastUser == nil || !strings.Contains(lastUser.Content, "任务四") {
		t.Errorf("store 末条 user 应为任务四，得 %v", lastUser)
	}
	if lastUser != nil {
		// 找到 lastUser 的索引，其后必须有 assistant
		for i := range stored {
			if &stored[i] == lastUser || (stored[i].Role == RoleUser && strings.Contains(stored[i].Content, "任务四")) {
				if i+1 >= len(stored) || stored[i+1].Role != RoleAssistant {
					t.Errorf("最新 user（任务四）后无 assistant 回复")
				}
				break
			}
		}
	}
}

func shortContent(s string) string {
	r := []rune(s)
	if len(r) > 30 {
		return string(r[:30]) + "…"
	}
	return s
}
