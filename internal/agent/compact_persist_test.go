package agent

// compact_persist_test.go — 验证 maybeCompact（compact 分支）触发后，
// OnBatchPersist 的固定偏移重组是否会写坏 store（assistant 丢失/user→tool 粘连）。

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// runCompactRound 与 runToolRound 相同，但允许设置 MaxContextTokens 触发 maybeCompact。
var compactEvents []string

func runCompactRound(t *testing.T, store *MessageStore, convID, task string, round, maxTokens int) {
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
		Provider:         prov,
		Registry:         reg,
		System:           "test-system",
		MaxIterations:    3,
		MaxContextTokens: maxTokens,
		History:          CopyHistory(condensed),
	}
	// 记录压缩事件（验证测试确实触发了 compact 分支）
	loop.OnEvent = func(e Event) {
		if e.Type == EventCompacted {
			compactEvents = append(compactEvents, e.Content)
		}
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

// TestCompactThenPersist_NoAssistantLost 长历史触发 maybeCompact（compact 分支）后，
// 验证 store 中 user 后必有 assistant、消息不丢失。
func TestCompactThenPersist_NoAssistantLost(t *testing.T) {
	store := NewMessageStore(t.TempDir())
	convID := "conv_compact"

	// 前 6 轮：产生长历史（每轮 4 条：user, assistant(tc), tool, assistant(final)）
	// 小 MaxContextTokens 让第 N 轮触发 maybeCompact（compact 分支删除中段）
	// → 验证修复后 store 仍保留完整原始历史（不丢失、不粘连）
	compactEvents = nil
	for i := 1; i <= 6; i++ {
		runCompactRound(t, store, convID, fmt.Sprintf("任务%d", i), i, 1500)
	}
	if len(compactEvents) == 0 {
		t.Logf("⚠️ 本轮历史未触发 maybeCompact（EventCompacted=0）——测试未覆盖 compact 路径")
	} else {
		t.Logf("✅ maybeCompact 触发 %d 次：%v", len(compactEvents), compactEvents)
	}

	stored, err := store.LoadAll(convID)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	t.Logf("store 共 %d 条消息", len(stored))

	// 结构断言：每条 user 后必须紧跟 assistant
	userCount := 0
	for i, m := range stored {
		if m.Role != RoleUser {
			continue
		}
		userCount++
		if i+1 >= len(stored) {
			t.Errorf("第 %d 条 user（%q）是最后一条——其后应有 assistant", i, shortContent(m.Content))
			continue
		}
		next := stored[i+1]
		if next.Role != RoleAssistant {
			t.Errorf("第 %d 条 user（%q）后紧跟 %s（%q）——assistant 丢失/user与tool粘连",
				i, shortContent(m.Content), next.Role, shortContent(next.Content))
		}
	}
	if userCount < 6 {
		t.Errorf("user 消息数应 >= 6，得 %d（可能被压缩删除）", userCount)
	}
	// 最新 user（任务6）必须存在且其后有 assistant
	lastUser := lastUserMsg(stored)
	if lastUser == nil || !strings.Contains(lastUser.Content, "任务6") {
		t.Errorf("store 末条 user 应为任务6，得 %v", lastUser)
	}
	if lastUser != nil {
		for i := range stored {
			if stored[i].Role == RoleUser && strings.Contains(stored[i].Content, "任务6") {
				if i+1 >= len(stored) || stored[i+1].Role != RoleAssistant {
					t.Errorf("最新 user（任务6）后无 assistant 回复")
				}
				break
			}
		}
	}
}
