package state

import "testing"

func TestChatStoreSend(t *testing.T) {
	s := NewChatStore()
	a := s.Active()
	if a == nil {
		t.Fatal("应有当前会话")
	}
	n0 := len(a.Messages)

	s.Draft = "hello"
	if !s.Send("hello") {
		t.Fatal("Send 应成功")
	}
	// Send 只加 user 消息；助手回复由 Agent 引擎异步追加。
	if len(a.Messages) != n0+1 {
		t.Errorf("应追加 1 条 user 消息，得 %d", len(a.Messages)-n0)
	}
	if a.Messages[n0].Role != User || a.Messages[n0].Text != "hello" {
		t.Error("追加的应为 user 'hello'")
	}
	if s.Draft != "" {
		t.Errorf("发送后 Draft 应清空，得 %q", s.Draft)
	}
	if a.Title != "hello" {
		t.Errorf("首条消息应成标题，得 %q", a.Title)
	}
	if s.Send("   ") {
		t.Error("纯空白不应发送")
	}
}

func TestChatStoreThreads(t *testing.T) {
	s := NewChatStore()
	id1 := s.ActiveID
	if len(s.Threads) != 1 {
		t.Fatalf("初始应 1 会话，得 %d", len(s.Threads))
	}
	t2 := s.NewThread()
	if s.ActiveID != t2.ID {
		t.Error("新建后应切到新会话")
	}
	if len(s.Threads) != 2 {
		t.Errorf("应 2 会话，得 %d", len(s.Threads))
	}
	if s.Threads[0].ID != t2.ID {
		t.Error("新会话应置顶")
	}
	s.Switch(id1)
	if s.ActiveID != id1 {
		t.Error("切换失败")
	}
	s.Delete(id1)
	if s.ActiveID == id1 {
		t.Error("删当前会话后应切走")
	}
	if len(s.Threads) != 1 {
		t.Errorf("删后应剩 1，得 %d", len(s.Threads))
	}
	s.Delete(s.ActiveID)
	if len(s.Threads) != 1 {
		t.Error("删空应自动新建 1 个")
	}
}
