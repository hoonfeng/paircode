// MessageStore 单元测试
package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestMessageStore_AppendAndLoadLatest 测试追加消息与加载最新消息。
func TestMessageStore_AppendAndLoadLatest(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	if err := store.CreateConversation("conv1", "测试", "/ws"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// 连续追加 5 条消息（user/assistant/user/assistant/user），每条带不同 segments
	type caseData struct {
		msg      Message
		segments []Segment
	}
	cases := []caseData{
		{
			msg:      Message{Role: RoleUser, Content: "你好"},
			segments: []Segment{{Type: "content", Content: "你好"}},
		},
		{
			msg: Message{Role: RoleAssistant, Content: "你好，有什么可以帮你？"},
			segments: []Segment{
				{Type: "thinking", Content: "分析用户意图"},
				{Type: "content", Content: "你好，有什么可以帮你？"},
			},
		},
		{
			msg:      Message{Role: RoleUser, Content: "帮我搜索文件"},
			segments: []Segment{{Type: "content", Content: "帮我搜索文件"}},
		},
		{
			msg: Message{
				Role:    RoleAssistant,
				Content: "",
				ToolCalls: []ToolCall{
					{ID: "call1", Type: "function", Function: FunctionCall{Name: "search", Arguments: `{"pattern":"*.go"}`}},
				},
			},
			segments: []Segment{{Type: "tool_call", Name: "search", ArgsRaw: `{"pattern":"*.go"}`, CallID: "call1"}},
		},
		{
			msg:      Message{Role: RoleUser, Content: "找到了吗？"},
			segments: []Segment{{Type: "content", Content: "找到了吗？"}},
		},
	}
	for i, c := range cases {
		if err := store.AppendMessage("conv1", c.msg, c.segments); err != nil {
			t.Fatalf("AppendMessage[%d]: %v", i, err)
		}
	}

	t.Run("LoadLatest_limit3", func(t *testing.T) {
		msgs, total, err := store.LoadLatest("conv1", 3)
		if err != nil {
			t.Fatalf("LoadLatest: %v", err)
		}
		if total != 5 {
			t.Errorf("total 应为 5, got %d", total)
		}
		if len(msgs) != 3 {
			t.Fatalf("应返回 3 条, got %d", len(msgs))
		}
		// 应返回 idx 2,3,4
		for i, m := range msgs {
			if m.Idx != 2+i {
				t.Errorf("idx 应为 %d, got %d", 2+i, m.Idx)
			}
		}
	})

	t.Run("LoadLatest_limit0_返回全部", func(t *testing.T) {
		msgs, total, err := store.LoadLatest("conv1", 0)
		if err != nil {
			t.Fatalf("LoadLatest: %v", err)
		}
		if total != 5 {
			t.Errorf("total 应为 5, got %d", total)
		}
		if len(msgs) != 5 {
			t.Errorf("应返回 5 条, got %d", len(msgs))
		}
	})

	t.Run("LoadLatest_limit100_返回全部", func(t *testing.T) {
		msgs, total, err := store.LoadLatest("conv1", 100)
		if err != nil {
			t.Fatalf("LoadLatest: %v", err)
		}
		if total != 5 {
			t.Errorf("total 应为 5, got %d", total)
		}
		if len(msgs) != 5 {
			t.Errorf("应返回 5 条, got %d", len(msgs))
		}
	})

	t.Run("LoadLatest_不存在的对话", func(t *testing.T) {
		msgs, total, err := store.LoadLatest("nonexistent", 10)
		if err != nil {
			t.Fatalf("LoadLatest: %v", err)
		}
		if total != 0 {
			t.Errorf("total 应为 0, got %d", total)
		}
		if len(msgs) != 0 {
			t.Errorf("应返回空切片, got %d 条", len(msgs))
		}
	})
}

// TestMessageStore_LoadBefore 测试向前分页加载（idx < beforeIdx）。
func TestMessageStore_LoadBefore(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	if err := store.CreateConversation("conv1", "测试", "/ws"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// 准备 5 条消息（idx 0-4）
	for i := 0; i < 5; i++ {
		role := RoleUser
		if i%2 == 1 {
			role = RoleAssistant
		}
		msg := Message{Role: role, Content: "msg-" + strconv.Itoa(i)}
		if err := store.AppendMessage("conv1", msg, nil); err != nil {
			t.Fatalf("AppendMessage[%d]: %v", i, err)
		}
	}

	t.Run("beforeIdx4_limit2_返回idx2_3", func(t *testing.T) {
		msgs, err := store.LoadBefore("conv1", 4, 2)
		if err != nil {
			t.Fatalf("LoadBefore: %v", err)
		}
		if len(msgs) != 2 {
			t.Fatalf("应返回 2 条, got %d", len(msgs))
		}
		if msgs[0].Idx != 2 || msgs[1].Idx != 3 {
			t.Errorf("应返回 idx 2,3, got %d,%d", msgs[0].Idx, msgs[1].Idx)
		}
	})

	t.Run("beforeIdx0_limit10_返回空", func(t *testing.T) {
		msgs, err := store.LoadBefore("conv1", 0, 10)
		if err != nil {
			t.Fatalf("LoadBefore: %v", err)
		}
		if len(msgs) != 0 {
			t.Errorf("应返回空, got %d 条", len(msgs))
		}
	})

	t.Run("beforeIdx5_limit2_返回idx3_4", func(t *testing.T) {
		msgs, err := store.LoadBefore("conv1", 5, 2)
		if err != nil {
			t.Fatalf("LoadBefore: %v", err)
		}
		if len(msgs) != 2 {
			t.Fatalf("应返回 2 条, got %d", len(msgs))
		}
		if msgs[0].Idx != 3 || msgs[1].Idx != 4 {
			t.Errorf("应返回 idx 3,4, got %d,%d", msgs[0].Idx, msgs[1].Idx)
		}
	})

	t.Run("beforeIdx3_limit0_默认50_返回idx0_1_2", func(t *testing.T) {
		msgs, err := store.LoadBefore("conv1", 3, 0)
		if err != nil {
			t.Fatalf("LoadBefore: %v", err)
		}
		if len(msgs) != 3 {
			t.Fatalf("应返回 3 条, got %d", len(msgs))
		}
		for i, m := range msgs {
			if m.Idx != i {
				t.Errorf("应返回 idx %d, got %d", i, m.Idx)
			}
		}
	})
}

// TestMessageStore_LoadAll 测试加载全部消息（仅 Message，不含 Segments）。
func TestMessageStore_LoadAll(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	if err := store.CreateConversation("conv1", "测试", "/ws"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// 准备 3 条消息，其中 assistant 消息带 ToolCalls 和 Reasoning
	msgs := []Message{
		{Role: RoleUser, Content: "帮我搜索"},
		{
			Role:      RoleAssistant,
			Content:   "正在搜索",
			Reasoning: "分析搜索意图",
			ToolCalls: []ToolCall{
				{ID: "call1", Type: "function", Function: FunctionCall{Name: "search", Arguments: `{"q":"test"}`}},
			},
		},
		{Role: RoleTool, Content: "找到 3 个结果", ToolCallID: "call1", Name: "search"},
	}
	for i, m := range msgs {
		if err := store.AppendMessage("conv1", m, nil); err != nil {
			t.Fatalf("AppendMessage[%d]: %v", i, err)
		}
	}

	loaded, err := store.LoadAll("conv1")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("应返回 3 条, got %d", len(loaded))
	}

	// 验证第二条（assistant）的 ToolCalls 和 Reasoning 完整保留
	if loaded[1].Reasoning != "分析搜索意图" {
		t.Errorf("Reasoning 应为 '分析搜索意图', got %q", loaded[1].Reasoning)
	}
	if len(loaded[1].ToolCalls) != 1 {
		t.Fatalf("ToolCalls 应有 1 条, got %d", len(loaded[1].ToolCalls))
	}
	tc := loaded[1].ToolCalls[0]
	if tc.ID != "call1" || tc.Function.Name != "search" || tc.Function.Arguments != `{"q":"test"}` {
		t.Errorf("ToolCall 字段不匹配: %+v", tc)
	}

	// 用 reflect.DeepEqual 验证整条 assistant 消息
	if !reflect.DeepEqual(loaded[1], msgs[1]) {
		t.Errorf("assistant 消息应完全相等: got %+v, want %+v", loaded[1], msgs[1])
	}

	// LoadAll 不返回 Segments（返回类型为 []Message，不含 Segments 字段）
	if loaded[0].Role != RoleUser || loaded[0].Content != "帮我搜索" {
		t.Errorf("第一条消息不匹配: %+v", loaded[0])
	}
}

// TestMessageStore_ConcurrentAppend 测试并发追加消息的线程安全性。
func TestMessageStore_ConcurrentAppend(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	if err := store.CreateConversation("conv1", "并发测试", "/ws"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// 10 个 goroutine 各追加 10 条消息（共 100 条）
	const goroutines = 10
	const perGoroutine = 10
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				msg := Message{
					Role:    RoleUser,
					Content: "g" + strconv.Itoa(gid) + "-m" + strconv.Itoa(i),
				}
				if err := store.AppendMessage("conv1", msg, nil); err != nil {
					t.Errorf("AppendMessage(g=%d,i=%d): %v", gid, i, err)
				}
			}
		}(g)
	}
	wg.Wait()

	// 验证总数为 100
	count, err := store.Count("conv1")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != goroutines*perGoroutine {
		t.Errorf("Count 应为 %d, got %d", goroutines*perGoroutine, count)
	}

	// 验证 LoadLatest 返回 100 条，且 Idx 0-99 唯一不重复
	msgs, total, err := store.LoadLatest("conv1", 0)
	if err != nil {
		t.Fatalf("LoadLatest: %v", err)
	}
	if total != goroutines*perGoroutine {
		t.Errorf("total 应为 %d, got %d", goroutines*perGoroutine, total)
	}
	if len(msgs) != goroutines*perGoroutine {
		t.Fatalf("应返回 %d 条, got %d", goroutines*perGoroutine, len(msgs))
	}

	seen := make(map[int]bool, len(msgs))
	for _, m := range msgs {
		if seen[m.Idx] {
			t.Errorf("重复的 idx: %d", m.Idx)
		}
		seen[m.Idx] = true
	}
	for i := 0; i < goroutines*perGoroutine; i++ {
		if !seen[i] {
			t.Errorf("缺少 idx: %d", i)
		}
	}
}

// TestMessageStore_DeleteConversation 测试删除对话。
func TestMessageStore_DeleteConversation(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	if err := store.CreateConversation("conv1", "对话1", "/ws"); err != nil {
		t.Fatalf("CreateConversation conv1: %v", err)
	}
	if err := store.CreateConversation("conv2", "对话2", "/ws"); err != nil {
		t.Fatalf("CreateConversation conv2: %v", err)
	}

	// conv1 追加 5 条，conv2 追加 3 条
	for i := 0; i < 5; i++ {
		if err := store.AppendMessage("conv1", Message{Role: RoleUser, Content: "m" + strconv.Itoa(i)}, nil); err != nil {
			t.Fatalf("AppendMessage conv1[%d]: %v", i, err)
		}
	}
	for i := 0; i < 3; i++ {
		if err := store.AppendMessage("conv2", Message{Role: RoleUser, Content: "m" + strconv.Itoa(i)}, nil); err != nil {
			t.Fatalf("AppendMessage conv2[%d]: %v", i, err)
		}
	}

	// 删除 conv1
	if err := store.DeleteConversation("conv1"); err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}

	// 验证 conv1 计数为 0
	count, err := store.Count("conv1")
	if err != nil {
		t.Fatalf("Count conv1: %v", err)
	}
	if count != 0 {
		t.Errorf("conv1 Count 应为 0, got %d", count)
	}

	// 验证 conv1 元数据已删除（返回 nil, nil）
	meta, err := store.GetConversation("conv1")
	if err != nil {
		t.Fatalf("GetConversation conv1: %v", err)
	}
	if meta != nil {
		t.Errorf("conv1 元数据应返回 nil, got %+v", meta)
	}

	// 验证 conv2 仍存在
	count2, err := store.Count("conv2")
	if err != nil {
		t.Fatalf("Count conv2: %v", err)
	}
	if count2 != 3 {
		t.Errorf("conv2 Count 应为 3, got %d", count2)
	}
}

// TestMessageStore_ListConversations 测试按工作区列出对话。
func TestMessageStore_ListConversations(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	// 创建 conv1（ws="/ws1"）+ conv2（ws="/ws2"）+ conv3（ws="" 兼容旧数据）
	if err := store.CreateConversation("conv1", "对话1", "/ws1"); err != nil {
		t.Fatalf("CreateConversation conv1: %v", err)
	}
	if err := store.CreateConversation("conv2", "对话2", "/ws2"); err != nil {
		t.Fatalf("CreateConversation conv2: %v", err)
	}
	if err := store.CreateConversation("conv3", "对话3", ""); err != nil {
		t.Fatalf("CreateConversation conv3: %v", err)
	}

	// ListConversations("/ws1") 应返回 conv1 + conv3（ws="" 视为属于该工作区）
	list, err := store.ListConversations("/ws1")
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}

	ids := make(map[string]bool, len(list))
	for _, m := range list {
		ids[m.ID] = true
	}
	if !ids["conv1"] {
		t.Errorf("应包含 conv1")
	}
	if !ids["conv3"] {
		t.Errorf("应包含 conv3（ws='' 兼容旧数据）")
	}
	if ids["conv2"] {
		t.Errorf("不应包含 conv2（ws='/ws2'）")
	}

	// 验证按 UpdatedAt 倒序
	for i := 0; i < len(list)-1; i++ {
		if list[i].UpdatedAt < list[i+1].UpdatedAt {
			t.Errorf("应按 UpdatedAt 倒序: list[%d]=%s < list[%d]=%s", i, list[i].UpdatedAt, i+1, list[i+1].UpdatedAt)
		}
	}
}

// TestMessageStore_UpdateTitle 测试更新对话标题。
func TestMessageStore_UpdateTitle(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	if err := store.CreateConversation("conv1", "原标题", "/ws"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// 获取原始 UpdatedAt
	meta, err := store.GetConversation("conv1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	oldUpdatedAt := meta.UpdatedAt

	// 等待 1 秒确保 UpdatedAt 时间戳不同（RFC3339 精度为秒）
	time.Sleep(time.Second)

	if err := store.UpdateTitle("conv1", "新标题"); err != nil {
		t.Fatalf("UpdateTitle: %v", err)
	}

	meta2, err := store.GetConversation("conv1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if meta2.Title != "新标题" {
		t.Errorf("Title 应为 '新标题', got %q", meta2.Title)
	}
	if meta2.UpdatedAt == oldUpdatedAt {
		t.Errorf("UpdatedAt 应被更新, 仍为 %s", oldUpdatedAt)
	}
}

// TestMessageStore_SetSummary 测试设置对话摘要。
func TestMessageStore_SetSummary(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	if err := store.CreateConversation("conv1", "测试", "/ws"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	if err := store.SetSummary("conv1", "对话摘要", "2026-07-13T00:00:00Z"); err != nil {
		t.Fatalf("SetSummary: %v", err)
	}

	meta, err := store.GetConversation("conv1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if meta.Summary != "对话摘要" {
		t.Errorf("Summary 应为 '对话摘要', got %q", meta.Summary)
	}
	if meta.SummaryAt != "2026-07-13T00:00:00Z" {
		t.Errorf("SummaryAt 应为 '2026-07-13T00:00:00Z', got %q", meta.SummaryAt)
	}
}

// TestMessageStore_SetCtxStats 测试设置上下文 token 统计。
func TestMessageStore_SetCtxStats(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	if err := store.CreateConversation("conv1", "测试", "/ws"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	stats := &Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
	if err := store.SetCtxStats("conv1", stats); err != nil {
		t.Fatalf("SetCtxStats: %v", err)
	}

	meta, err := store.GetConversation("conv1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if meta.CtxStats == nil {
		t.Fatal("CtxStats 应不为 nil")
	}
	if meta.CtxStats.PromptTokens != 100 {
		t.Errorf("PromptTokens 应为 100, got %d", meta.CtxStats.PromptTokens)
	}
	if meta.CtxStats.CompletionTokens != 50 {
		t.Errorf("CompletionTokens 应为 50, got %d", meta.CtxStats.CompletionTokens)
	}
	if meta.CtxStats.TotalTokens != 150 {
		t.Errorf("TotalTokens 应为 150, got %d", meta.CtxStats.TotalTokens)
	}
}

// TestMessageStore_SetInterrupted 测试异常中断标记的写入/清除/持久化。
func TestMessageStore_SetInterrupted(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	if err := store.CreateConversation("conv1", "测试", "/ws"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// 初始应无中断标记
	meta, err := store.GetConversation("conv1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if meta.Interrupted {
		t.Error("新建对话 Interrupted 应为 false")
	}

	// 标记中断
	if err := store.SetInterrupted("conv1", true); err != nil {
		t.Fatalf("SetInterrupted(true): %v", err)
	}
	meta2, err := store.GetConversation("conv1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if !meta2.Interrupted {
		t.Error("SetInterrupted(true) 后 Interrupted 应为 true")
	}

	// 清除中断（继续对话）
	if err := store.SetInterrupted("conv1", false); err != nil {
		t.Fatalf("SetInterrupted(false): %v", err)
	}
	meta3, err := store.GetConversation("conv1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if meta3.Interrupted {
		t.Error("SetInterrupted(false) 后 Interrupted 应为 false")
	}

	// 列表也应反映标记（持久化验证：新建 store 重新读取）
	store2 := NewMessageStore(root)
	if err := store2.SetInterrupted("conv1", true); err != nil {
		t.Fatalf("store2 SetInterrupted(true): %v", err)
	}
	metas, err := store2.ListConversations("/ws")
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	found := false
	for _, m := range metas {
		if m.ID == "conv1" {
			found = m.Interrupted
			break
		}
	}
	if !found {
		t.Error("ListConversations 应返回 interrupted=true 的对话")
	}

	// 不存在的对话无操作不报错
	if err := store.SetInterrupted("no_such_conv", true); err != nil {
		t.Errorf("SetInterrupted 不存在的对话应无操作: %v", err)
	}
}

// TestPersistKeepsInterrupted 验证 PersistNewMessages 等常规写操作不会抹掉
// interrupted 标记（loadIndex→saveIndex 全量写回必须保留该字段，否则前端
// "未完成可继续"提示会在下一轮消息落盘后消失）。
func TestPersistKeepsInterrupted(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	if err := store.CreateConversation("conv1", "测试", "/ws"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if err := store.SetInterrupted("conv1", true); err != nil {
		t.Fatalf("SetInterrupted: %v", err)
	}

	// 模拟会话落盘：追加几条消息（触发 PersistNewMessages 写 index.json）
	hist := []Message{
		{Role: RoleUser, Content: "任务开始"},
		{Role: RoleAssistant, Content: "分析中...", Reasoning: "思考"},
		{Role: RoleUser, Content: "继续"},
	}
	if err := store.PersistNewMessages("conv1", hist); err != nil {
		t.Fatalf("PersistNewMessages: %v", err)
	}

	// 再模拟 UpdateTitle / SetCtxStats（都走 loadIndex→saveIndex）
	if err := store.UpdateTitle("conv1", "新标题"); err != nil {
		t.Fatalf("UpdateTitle: %v", err)
	}
	if err := store.SetCtxStats("conv1", &Usage{PromptTokens: 10}); err != nil {
		t.Fatalf("SetCtxStats: %v", err)
	}

	meta, err := store.GetConversation("conv1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if !meta.Interrupted {
		t.Error("PersistNewMessages/UpdateTitle/SetCtxStats 后 interrupted 标记应保留")
	}
}

// TestMessageStore_MigrateFromLegacy 测试从旧格式迁移（有 history_cache）。
func TestMessageStore_MigrateFromLegacy(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	convJSONPath := filepath.Join(root, "conversations.json")
	hcJSONPath := filepath.Join(root, "history_cache.json")

	// 创建 conversations.json
	convJSON := `[
  {"id":"conv1","title":"测试1","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","workspaceRoot":"/ws","messages":[{"role":"user","content":"hello"},{"role":"assistant","content":"hi"}],"summary":"摘要","summaryAt":"2026-01-03T00:00:00Z","ctxStats":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}}
]`
	if err := os.WriteFile(convJSONPath, []byte(convJSON), 0o644); err != nil {
		t.Fatalf("写入 conversations.json: %v", err)
	}

	// 创建 history_cache.json（含完整 History，含 Reasoning）
	hcJSON := `{"conv1":{"history":[{"role":"user","content":"完整历史hello"},{"role":"assistant","content":"完整历史hi","reasoning_content":"思考过程"}]}}`
	if err := os.WriteFile(hcJSONPath, []byte(hcJSON), 0o644); err != nil {
		t.Fatalf("写入 history_cache.json: %v", err)
	}

	// 调用迁移
	if err := store.MigrateFromLegacy(convJSONPath, hcJSONPath); err != nil {
		t.Fatalf("MigrateFromLegacy: %v", err)
	}

	t.Run("对话存在且标题正确", func(t *testing.T) {
		meta, err := store.GetConversation("conv1")
		if err != nil {
			t.Fatalf("GetConversation: %v", err)
		}
		if meta == nil {
			t.Fatal("conv1 应存在")
		}
		if meta.Title != "测试1" {
			t.Errorf("Title 应为 '测试1', got %q", meta.Title)
		}
	})

	t.Run("消息来自history_cache", func(t *testing.T) {
		msgs, err := store.LoadAll("conv1")
		if err != nil {
			t.Fatalf("LoadAll: %v", err)
		}
		if len(msgs) != 2 {
			t.Fatalf("应返回 2 条 Message, got %d", len(msgs))
		}
		// 第一条应为完整历史（content 含 "完整历史"），而非 conversations.json 的简化版
		if msgs[0].Content != "完整历史hello" {
			t.Errorf("第一条 Content 应为 '完整历史hello', got %q", msgs[0].Content)
		}
		// 第二条的 Reasoning 应为 "思考过程"（验证用 history_cache 而非 conversations.json）
		if msgs[1].Reasoning != "思考过程" {
			t.Errorf("第二条 Reasoning 应为 '思考过程', got %q", msgs[1].Reasoning)
		}
	})

	t.Run("摘要字段正确", func(t *testing.T) {
		meta, err := store.GetConversation("conv1")
		if err != nil {
			t.Fatalf("GetConversation: %v", err)
		}
		if meta.Summary != "摘要" {
			t.Errorf("Summary 应为 '摘要', got %q", meta.Summary)
		}
	})

	t.Run("CtxStats正确", func(t *testing.T) {
		meta, err := store.GetConversation("conv1")
		if err != nil {
			t.Fatalf("GetConversation: %v", err)
		}
		if meta.CtxStats == nil {
			t.Fatal("CtxStats 应不为 nil")
		}
		if meta.CtxStats.PromptTokens != 100 {
			t.Errorf("PromptTokens 应为 100, got %d", meta.CtxStats.PromptTokens)
		}
	})

	t.Run("旧文件重命名为bak", func(t *testing.T) {
		// conversations.json.bak 应存在
		if _, err := os.Stat(convJSONPath + ".bak"); err != nil {
			t.Errorf("conversations.json.bak 应存在: %v", err)
		}
		// 原 conversations.json 应不存在
		if _, err := os.Stat(convJSONPath); !os.IsNotExist(err) {
			t.Errorf("原 conversations.json 应不存在")
		}
		// history_cache.json.bak 应存在
		if _, err := os.Stat(hcJSONPath + ".bak"); err != nil {
			t.Errorf("history_cache.json.bak 应存在: %v", err)
		}
		// 原 history_cache.json 应不存在
		if _, err := os.Stat(hcJSONPath); !os.IsNotExist(err) {
			t.Errorf("原 history_cache.json 应不存在")
		}
	})

	t.Run("再次迁移应跳过已迁移的对话", func(t *testing.T) {
		// 恢复 conversations.json 从 .bak
		bakData, err := os.ReadFile(convJSONPath + ".bak")
		if err != nil {
			t.Fatalf("读取 .bak: %v", err)
		}
		if err := os.WriteFile(convJSONPath, bakData, 0o644); err != nil {
			t.Fatalf("恢复 conversations.json: %v", err)
		}

		// 再次调用 MigrateFromLegacy，应跳过已迁移的对话
		if err := store.MigrateFromLegacy(convJSONPath, hcJSONPath); err != nil {
			t.Fatalf("第二次迁移不应报错: %v", err)
		}

		// 验证不重复
		count, err := store.Count("conv1")
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		if count != 2 {
			t.Errorf("第二次迁移后应仍为 2 条, got %d", count)
		}
	})
}

// TestMessageStore_MigrateFromLegacy_NoHistoryCache 测试迁移时无 history_cache.json。
func TestMessageStore_MigrateFromLegacy_NoHistoryCache(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)

	convJSONPath := filepath.Join(root, "conversations.json")
	hcJSONPath := filepath.Join(root, "history_cache.json") // 不存在

	// 创建 conversations.json（简化 messages，无 ToolCalls/Reasoning）
	convJSON := `[
  {"id":"conv1","title":"测试1","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","workspaceRoot":"/ws","messages":[{"role":"user","content":"hello"},{"role":"assistant","content":"hi"}]}
]`
	if err := os.WriteFile(convJSONPath, []byte(convJSON), 0o644); err != nil {
		t.Fatalf("写入 conversations.json: %v", err)
	}

	// 调用迁移（history_cache.json 不存在）
	if err := store.MigrateFromLegacy(convJSONPath, hcJSONPath); err != nil {
		t.Fatalf("MigrateFromLegacy: %v", err)
	}

	// 验证对话存在
	meta, err := store.GetConversation("conv1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if meta == nil {
		t.Fatal("conv1 应存在")
	}
	if meta.Title != "测试1" {
		t.Errorf("Title 应为 '测试1', got %q", meta.Title)
	}

	// 验证使用 conversations.json 的简化 messages 重建（无 ToolCalls/Reasoning）
	msgs, err := store.LoadAll("conv1")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("应返回 2 条, got %d", len(msgs))
	}
	if msgs[0].Role != RoleUser || msgs[0].Content != "hello" {
		t.Errorf("第一条消息不匹配: %+v", msgs[0])
	}
	if msgs[0].Reasoning != "" {
		t.Errorf("第一条 Reasoning 应为空, got %q", msgs[0].Reasoning)
	}
	if len(msgs[0].ToolCalls) != 0 {
		t.Errorf("第一条 ToolCalls 应为空, got %d", len(msgs[0].ToolCalls))
	}
	if msgs[1].Role != RoleAssistant || msgs[1].Content != "hi" {
		t.Errorf("第二条消息不匹配: %+v", msgs[1])
	}
	if msgs[1].Reasoning != "" {
		t.Errorf("第二条 Reasoning 应为空, got %q", msgs[1].Reasoning)
	}
	if len(msgs[1].ToolCalls) != 0 {
		t.Errorf("第二条 ToolCalls 应为空, got %d", len(msgs[1].ToolCalls))
	}

	// 验证旧 conversations.json 被重命名为 .bak
	if _, err := os.Stat(convJSONPath + ".bak"); err != nil {
		t.Errorf("conversations.json.bak 应存在: %v", err)
	}
	if _, err := os.Stat(convJSONPath); !os.IsNotExist(err) {
		t.Errorf("原 conversations.json 应不存在")
	}

	// history_cache.json 不存在时不报错（已在上方调用无错误验证）
}

// TestMessageStore_CheckAndArchive 验证自动归档：超过阈值后最早消息移入 .archived.jsonl，
// 主文件第一条为 role=user 的【历史归档】摘要（而非孤立 assistant——避免 LLM 上下文污染）。
func TestMessageStore_CheckAndArchive(t *testing.T) {
	root := t.TempDir()
	store := NewMessageStore(root)
	if err := store.CreateConversation("conv_arch", "归档测试", "/ws"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// 追加 ArchiveThreshold 条消息（第 500 条触发 AppendMessage 内的自动归档）
	total := ArchiveThreshold // 500
	for i := 0; i < total; i++ {
		content := fmt.Sprintf("消息 %d", i)
		role := RoleUser
		if i%2 == 1 {
			role = RoleAssistant
		}
		if err := store.AppendMessage("conv_arch", Message{Role: role, Content: content},
			[]Segment{{Type: "content", Content: content}}); err != nil {
			t.Fatalf("AppendMessage[%d]: %v", i, err)
		}
	}

	// 归档后主文件 = 摘要 + 保留最新 1/4（125 条）
	expectKeep := ArchiveThreshold / ArchiveRatio // 125
	expectTotal := expectKeep + 1                 // 摘要占 idx=0

	msgs, err := store.ReadAll("conv_arch")
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(msgs) != expectTotal {
		t.Fatalf("归档后主文件应 %d 条（摘要+%d 保留）, got %d", expectTotal, expectKeep, len(msgs))
	}

	t.Run("首条为 user 角色归档摘要", func(t *testing.T) {
		first := msgs[0]
		if first.Message.Role != RoleUser {
			t.Errorf("摘要 Role 应为 RoleUser（避免孤立 assistant）, got %q", first.Message.Role)
		}
		if !strings.Contains(first.Message.Content, "【历史归档】") {
			t.Errorf("摘要内容应含【历史归档】, got %q", first.Message.Content)
		}
		if first.Idx != 0 {
			t.Errorf("摘要 Idx 应为 0, got %d", first.Idx)
		}
	})

	t.Run("保留消息为最新且 Idx 连续", func(t *testing.T) {
		// 最早保留的消息 = 消息 (total-keep)
		firstKeep := ArchiveThreshold - expectKeep // 375
		for i, m := range msgs {
			if m.Idx != i {
				t.Errorf("Idx 应连续 %d, got %d", i, m.Idx)
			}
		}
		if msgs[1].Message.Content != fmt.Sprintf("消息 %d", firstKeep) {
			t.Errorf("第一条保留消息应为 消息 %d, got %q", firstKeep, msgs[1].Message.Content)
		}
		last := msgs[len(msgs)-1]
		if last.Message.Content != fmt.Sprintf("消息 %d", ArchiveThreshold-1) {
			t.Errorf("最后一条应为 消息 %d, got %q", ArchiveThreshold-1, last.Message.Content)
		}
	})

	t.Run("归档文件包含全部最早消息", func(t *testing.T) {
		archivedPath := store.convFilePath("conv_arch") + ArchivedFileSuffix
		raw, err := os.ReadFile(archivedPath)
		if err != nil {
			t.Fatalf("读取归档文件: %v", err)
		}
		expectArchived := ArchiveThreshold - expectKeep // 375
		if lines := strings.Count(string(raw), "\n"); lines != expectArchived {
			t.Errorf("归档文件应有 %d 条, got %d", expectArchived, lines)
		}
	})

	t.Run("persistedCount 与主文件一致", func(t *testing.T) {
		if n := store.GetPersistedCount("conv_arch"); n != expectTotal {
			t.Errorf("GetPersistedCount 应为 %d, got %d", expectTotal, n)
		}
	})

	t.Run("LoadAll 首条为 user 摘要（LLM 上下文无孤立 assistant）", func(t *testing.T) {
		all, err := store.LoadAll("conv_arch")
		if err != nil {
			t.Fatalf("LoadAll: %v", err)
		}
		if len(all) != expectTotal {
			t.Fatalf("LoadAll 应 %d 条, got %d", expectTotal, len(all))
		}
		if all[0].Role != RoleUser {
			t.Errorf("LoadAll 首条 Role 应为 user, got %q", all[0].Role)
		}
	})

	t.Run("归档后继续追加不立即再次归档", func(t *testing.T) {
		if err := store.AppendMessage("conv_arch", Message{Role: RoleUser, Content: "归档后消息"},
			[]Segment{{Type: "content", Content: "归档后消息"}}); err != nil {
			t.Fatalf("AppendMessage: %v", err)
		}
		msgs, err := store.ReadAll("conv_arch")
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if len(msgs) != expectTotal+1 {
			t.Errorf("归档后追加应 %d 条, got %d", expectTotal+1, len(msgs))
		}
	})
}

// TestNormalizeAskType 归一化 ask_type 变体（模型可能生成 various 变体）。
func TestNormalizeAskType(t *testing.T) {
	cases := map[string]string{
		"":                    "text",
		"text":                "text",
		"input":               "text",
		"free-text":           "text",
		"single":              "single",
		"single-choice":       "single",
		"radio":               "single",
		"choice":              "single",
		"multi":               "multi",
		"multiple":            "multi",
		"checkbox":            "multi",
		"multi-select":        "multi",
		"single_with_input":   "single-with-input",
		"single-with-input":   "single-with-input",
		"single-input":        "single-with-input",
		"choice-with-input":   "single-with-input",
		"single_with_custom":  "single-with-input",
		"unknown_variant_xyz": "text", // 未知变体回退 text（保证输入框出现）
	}
	for in, want := range cases {
		if got := normalizeAskType(in); got != want {
			t.Errorf("normalizeAskType(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestParseAskArgs_Variants parseAskArgs 对 askType 变体与 type 字段容错。
func TestParseAskArgs_Variants(t *testing.T) {
	// 变体：type=single_with_input + options
	q, at, opts := parseAskArgs(`{"question":"继续吗?","type":"single_with_input","options":["是","否"]}`)
	if q != "继续吗?" {
		t.Errorf("question = %q", q)
	}
	if at != "single-with-input" {
		t.Errorf("askType = %q, want single-with-input", at)
	}
	if len(opts) != 2 {
		t.Errorf("options = %v", opts)
	}

	// 变体：askType=choice（single 语义）
	_, at2, _ := parseAskArgs(`{"question":"选一个","askType":"choice","options":["A","B"]}`)
	if at2 != "single" {
		t.Errorf("askType = %q, want single", at2)
	}

	// 未知类型回退 text
	_, at3, _ := parseAskArgs(`{"question":"x","askType":"whatever"}`)
	if at3 != "text" {
		t.Errorf("askType = %q, want text", at3)
	}

	// 无参数 → 原始文本 + text
	q4, at4, _ := parseAskArgs(`裸问题`)
	if q4 != `裸问题` || at4 != "text" {
		t.Errorf("got q=%q at=%q", q4, at4)
	}
}
