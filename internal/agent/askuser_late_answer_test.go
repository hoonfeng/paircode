// askuser_late_answer_test.go — ★ 2026-09-25 ask_user「迟到回答」落盘兜底测试。
//
// 背景（真实缺陷）：ask_user 提问超时（5 分钟）或被停止后，agent 已不再读 askCh；
// 此时用户提交回答，旧实现把回答投递进 1 格缓冲的通道 → 投递「成功」（API 返回 ok、
// 前端显示「已回答」）但无人消费 → 回答永不落盘，历史查无此答。
// 修复：SendAnswers 检测无等待者（askPending=false）→ 落盘为会话 user 消息并返回
// ErrAskNotPending；重复提交幂等。
package agent

import (
	"errors"
	"strings"
	"testing"
)

func newLateTestManager(t *testing.T, convID string) (*SessionManager, *MessageStore, *Session) {
	t.Helper()
	m := NewSessionManager()
	root := t.TempDir()
	store := m.storeFor(root)
	if store == nil {
		t.Fatal("storeFor 返回 nil")
	}
	if err := store.CreateConversation(convID, "迟到回答测试", root); err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	sess := &Session{ConvID: convID, WorkspaceRoot: root, Running: true}
	m.mu.Lock()
	m.sessions[convID] = sess
	m.mu.Unlock()
	return m, store, sess
}

// 无等待者（提问超时/停止）→ 回答落盘为会话消息，不静默丢弃。
func TestSendAnswersLatePersistsMessage(t *testing.T) {
	m, store, _ := newLateTestManager(t, "c_late")

	answers := []AskAnswer{{ID: "q1", Answer: "选项A"}, {ID: "q2", Answer: "这是我的自定义意见"}}
	err := m.SendAnswers("c_late", answers)
	if !errors.Is(err, ErrAskNotPending) {
		t.Fatalf("无等待者时应返回 ErrAskNotPending，实际: %v", err)
	}

	msgs, err := store.LoadAll("c_late")
	if err != nil {
		t.Fatalf("LoadAll 失败: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("应落盘 1 条回答消息，实际 %d 条", len(msgs))
	}
	if msgs[0].Role != RoleUser {
		t.Fatalf("落盘消息角色应为 user，实际 %q", msgs[0].Role)
	}
	for _, want := range []string{"超时", "[q1]", "选项A", "[q2]", "这是我的自定义意见"} {
		if !strings.Contains(msgs[0].Content, want) {
			t.Fatalf("落盘内容缺少 %q；实际内容：%s", want, msgs[0].Content)
		}
	}
}

// 幂等：同一回答重复提交（前端重试/双击）不产生重复消息。
func TestSendAnswersLateIsIdempotent(t *testing.T) {
	m, store, _ := newLateTestManager(t, "c_idem")
	answers := []AskAnswer{{ID: "q1", Answer: "选项A"}}

	for i := 0; i < 3; i++ {
		if err := m.SendAnswers("c_idem", answers); !errors.Is(err, ErrAskNotPending) {
			t.Fatalf("第 %d 次提交应返回 ErrAskNotPending，实际: %v", i+1, err)
		}
	}
	msgs, err := store.LoadAll("c_idem")
	if err != nil {
		t.Fatalf("LoadAll 失败: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("重复提交应只落盘 1 条，实际 %d 条", len(msgs))
	}
}

// 单问题路径（answer 文本，无 ID）：落盘内容即答案原文（不加装饰前缀，保持自然）。
func TestSendAnswersLateSingleText(t *testing.T) {
	m, store, _ := newLateTestManager(t, "c_single")
	if err := m.SendAnswers("c_single", []AskAnswer{{Answer: "我选第三种方案"}}); !errors.Is(err, ErrAskNotPending) {
		t.Fatalf("应返回 ErrAskNotPending，实际: %v", err)
	}
	msgs, _ := store.LoadAll("c_single")
	if len(msgs) != 1 || strings.TrimSpace(msgs[0].Content) != "我选第三种方案" {
		t.Fatalf("单问题落盘内容不符：%+v", msgs)
	}
}

// 有等待者（askPending=true）→ 走原通道投递，不落盘消息（不改变正常路径行为）。
func TestSendAnswersWithPendingDelivers(t *testing.T) {
	m, store, sess := newLateTestManager(t, "c_wait")
	sess.askCh = make(chan []AskAnswer, 1)
	sess.askPending.Store(true)

	if err := m.SendAnswers("c_wait", []AskAnswer{{Answer: "hi"}}); err != nil {
		t.Fatalf("有等待者时应投递成功，实际: %v", err)
	}
	select {
	case got := <-sess.askCh:
		if len(got) != 1 || got[0].Answer != "hi" {
			t.Fatalf("通道内容不符: %+v", got)
		}
	default:
		t.Fatal("回答未进入 askCh")
	}
	if !m.IsAwaitingAnswer("c_wait") {
		t.Fatal("IsAwaitingAnswer 在等待中应返回 true")
	}
	msgs, _ := store.LoadAll("c_wait")
	if len(msgs) != 0 {
		t.Fatalf("正常投递路径不应落盘消息，实际 %d 条", len(msgs))
	}
}

// 会话不存在 / 未运行：保持原有错误语义不变。
func TestSendAnswersErrorsUnchanged(t *testing.T) {
	m := NewSessionManager()
	if err := m.SendAnswers("nope", []AskAnswer{{Answer: "x"}}); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("未知会话应返回 ErrSessionNotFound，实际: %v", err)
	}
	if m.IsAwaitingAnswer("nope") {
		t.Fatal("未知会话 IsAwaitingAnswer 应为 false")
	}
}
