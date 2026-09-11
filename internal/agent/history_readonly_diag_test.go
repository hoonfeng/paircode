package agent

// history_readonly_diag_test.go — 「对话历史只读」诊断/回归（缓存命中率排查）。
//
// 铁律：对话历史（store JSONL / Loop.History）是 append-only 真源，
// 上下文只能由历史**构建**（副本 + 末尾追加），任何原地改写
// （改写已落盘消息内容 / 删除中段 / 插入占位 / 调换顺序）都会
// 切断 provider 前缀缓存（KV cache）——已发送过的字节一旦变化，
// 其后的全部历史在下次请求中必然 miss（实测单次 13K~54K tokens）。
//
// 本文件两条断言 + 一条诊断：
//  1. 跨轮：后一轮 store 的 Message 语义层 = 前一轮 + 追加（前缀逐字节稳定）；
//  2. 段内：相邻两次 LLM 请求的 messages 前缀零断裂（前一次是后一次的前缀）；
//  3. 诊断打印：记录层（StoredMessage）哪些字段被重写（展示字段允许，
//     Message 内容字段不允许）。
//
// 失败信息直接定位「第几条消息、哪一轮之间、变成什么」。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// recProvider 记录每次 LLM 请求的 messages 深拷贝（供前缀比对）。
type recProvider struct {
	inner *toolProvider
	reqs  [][]Message
}

func (p *recProvider) Name() string { return "rec" }

func (p *recProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	p.reqs = append(p.reqs, cloneMsgs(messages))
	return p.inner.Chat(ctx, messages, tools, onChunk)
}

// cloneMsgs 深拷贝消息列表（JSON 往返，避开 slice 共享底层数组）。
func cloneMsgs(msgs []Message) []Message {
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, cloneMsg(m))
	}
	return out
}

func cloneMsg(m Message) Message {
	b, err := json.Marshal(m)
	if err != nil {
		return m
	}
	var c Message
	if err := json.Unmarshal(b, &c); err != nil {
		return m
	}
	return c
}

func msgJSON(m Message) string {
	b, _ := json.Marshal(m)
	return string(b)
}

// diagRegistry 注册只读回显工具（不触碰文件系统，结果确定性）。
func diagRegistry() *Registry {
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
	return reg
}

// diagRound 复刻 web 端一轮（store 落盘 + 新 Loop + OnBatchPersist 重组），
// 返回本轮所有 LLM 请求的 messages 快照。
func diagRound(t *testing.T, store *MessageStore, convID, task, dir string, round int) [][]Message {
	t.Helper()
	if err := store.AppendUserMessage(convID, task); err != nil {
		t.Fatalf("第%d轮 AppendUserMessage: %v", round, err)
	}
	history, err := store.LoadAll(convID)
	if err != nil {
		t.Fatalf("第%d轮 LoadAll: %v", round, err)
	}
	// 真实链路：base = 未压缩原始历史（HistoryOriginal）；LLM 上下文 = 压力精简版
	base := cloneMsgs(history)
	condensed := CondenseHistoryByPressure(cloneMsgs(history), 128000)

	rp := &recProvider{inner: &toolProvider{rounds: round}}
	loop := &Loop{
		Provider:         rp,
		Registry:         diagRegistry(),
		System:           "test-system",
		MaxContextTokens: 128000,
		WorkspaceRoot:    dir,
		History:          cloneMsgs(condensed),
	}
	loop.OnBatchPersist = func(msgs []Message) {
		combined := composePersistMessages(base, msgs)
		if err := store.PersistNewMessages(convID, combined); err != nil {
			t.Errorf("第%d轮 PersistNewMessages: %v", round, err)
		}
	}
	if _, err := loop.Run(context.Background(), task, nil); err != nil {
		t.Fatalf("第%d轮 Run: %v", round, err)
	}
	return rp.reqs
}

// TestHistoryRecordImmutableAcrossTurns 跨轮落盘记录只读（前缀单调延展）。
func TestHistoryRecordImmutableAcrossTurns(t *testing.T) {
	dir := t.TempDir()
	store := NewMessageStore(dir)
	convID := "conv_readonly"
	if err := store.CreateConversation(convID, "只读校验", dir); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	var prev []Message
	var prevReqLast []Message
	var prevReqFirst []Message

	for round := 1; round <= 4; round++ {
		reqs := diagRound(t, store, convID, fmt.Sprintf("任务%d", round), dir, round)
		if len(reqs) == 0 {
			t.Fatalf("第%d轮没有任何 LLM 请求", round)
		}

		// ── 断言 1：段内相邻请求前缀零断裂（前一次必须是后一次的前缀）──
		for k := 1; k < len(reqs); k++ {
			a, b := reqs[k-1], reqs[k]
			if len(b) < len(a) {
				t.Errorf("第%d轮 请求#%d→#%d 消息数倒退：%d → %d（历史被删减）", round, k, k+1, len(a), len(b))
				continue
			}
			for i := 0; i < len(a); i++ {
				if msgJSON(a[i]) != msgJSON(b[i]) {
					t.Errorf("★ 第%d轮 请求#%d→#%d 前缀断裂于第 %d 条消息（role=%s）\n  前: %.200s\n  后: %.200s",
						round, k, k+1, i, a[i].Role, msgJSON(a[i]), msgJSON(b[i]))
					break
				}
			}
		}

		// ── 断言 2：跨轮 store 的 Message 语义层前缀稳定 ──
		cur, err := store.LoadAll(convID)
		if err != nil {
			t.Fatalf("第%d轮 LoadAll: %v", round, err)
		}
		if len(prev) > 0 {
			if len(cur) < len(prev) {
				t.Errorf("★ 第%d轮 store 条数倒退：%d → %d（历史被截断/改写）", round, len(prev), len(cur))
			}
			n := len(prev)
			if len(cur) < n {
				n = len(cur)
			}
			for i := 0; i < n; i++ {
				if msgJSON(prev[i]) != msgJSON(cur[i]) {
					t.Errorf("★ 第%d轮 store 第 %d 条消息被改写（role=%s）\n  轮%d: %.200s\n  轮%d: %.200s",
						round, i, prev[i].Role, round-1, msgJSON(prev[i]), round, msgJSON(cur[i]))
					break
				}
			}
		}

		// ── 断言 3：跨轮请求前缀（上轮末次请求 vs 本轮首次请求）──
		if round > 1 && prevReqFirst != nil {
			// 上轮首次请求：system + 历史 + 任务(r-1)
			// 本轮首次请求：system + 历史(+上轮新增) + 任务(r)：其前 len(prevReqFirst) 条应逐字节相同
			n := len(prevReqFirst)
			if len(reqs[0]) < n {
				t.Errorf("★ 第%d轮首次请求比上轮首次请求短：%d < %d", round, len(reqs[0]), n)
			} else {
				for i := 0; i < n; i++ {
					if msgJSON(prevReqFirst[i]) != msgJSON(reqs[0][i]) {
						t.Errorf("★ 跨轮（第%d轮→第%d轮）请求前缀断裂于第 %d 条（role=%s）\n  上轮: %.200s\n  本轮: %.200s",
							round-1, round, i, prevReqFirst[i].Role, msgJSON(prevReqFirst[i]), msgJSON(reqs[0][i]))
						break
					}
				}
			}
		}
		_ = prevReqLast

		prev = cur
		prevReqFirst = reqs[0]
		prevReqLast = reqs[len(reqs)-1]
		t.Logf("第%d轮：store %d 条；LLM 请求 %d 次；末次请求消息数 %d", round, len(cur), len(reqs), len(reqs[len(reqs)-1]))
	}
}

// TestPersistRewriteFieldDiag 诊断：全量覆盖写（PersistNewMessages）改写了
// 记录层的哪些字段——Message 内容字段必须零改写，展示字段（Timestamp/Segments/
// 事件标注）允许重算。输出直接列出差异字段名。
func TestPersistRewriteFieldDiag(t *testing.T) {
	dir := t.TempDir()
	store := NewMessageStore(dir)
	convID := "conv_fields"
	if err := store.CreateConversation(convID, "字段改写诊断", dir); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	reqs := diagRound(t, store, convID, "任务1", dir, 1)
	if len(reqs) == 0 {
		t.Fatal("无 LLM 请求")
	}
	fpath := filepath.Join(dir, ".pair", "conversations", convID+".jsonl")
	if _, err := os.Stat(fpath); err != nil {
		// 落盘路径由 store 决定，兜底用 store 自身 API 读取
		t.Logf("jsonl 未在预期路径（%v）——改用 LoadAll 比对", err)
	}

	before := readStoredRaw(t, store, convID)
	if len(before) == 0 {
		t.Fatal("落盘记录为空")
	}

	// 第二次落盘：同内容重写（模拟下一轮 OnBatchPersist 的全量覆盖）
	msgs, err := store.LoadAll(convID)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if err := store.PersistNewMessages(convID, msgs); err != nil {
		t.Fatalf("PersistNewMessages: %v", err)
	}
	after := readStoredRaw(t, store, convID)
	if len(before) != len(after) {
		t.Fatalf("重写后条数变化：%d → %d", len(before), len(after))
	}

	changed := map[string]int{}
	for i := range before {
		diffs := diffFields(t, before[i], after[i])
		for _, d := range diffs {
			changed[d]++
		}
		if i < 3 || len(diffs) > 0 && i == len(before)-1 {
			t.Logf("第 %d 条（role=%s）重写差异字段: %v", i, before[i].Message.Role, diffs)
		}
	}
	t.Logf("重写差异字段统计（共 %d 条记录）: %v", len(before), changed)
	for _, forbidden := range []string{"Role", "Content", "ToolCalls", "ToolCallID", "Reasoning", "Images", "Name"} {
		if n := changed[forbidden]; n > 0 {
			t.Errorf("★ 记录重写改写了 Message 内容字段 %s（%d 条）——历史必须只读", forbidden, n)
		}
	}
}

// readStoredRaw 读取 JSONL 原始行（StoredMessage 级），失败则返回 nil。
func readStoredRaw(t *testing.T, store *MessageStore, convID string) []StoredMessage {
	t.Helper()
	msgs, err := store.readJSONL(convID)
	if err != nil {
		t.Fatalf("readJSONL: %v", err)
	}
	return msgs
}

// diffFields 返回两条 StoredMessage 的差异字段名（含嵌套 Message 字段）。
func diffFields(t *testing.T, a, b StoredMessage) []string {
	t.Helper()
	var out []string
	if a.Idx != b.Idx {
		out = append(out, "Idx")
	}
	if a.Timestamp != b.Timestamp {
		out = append(out, "Timestamp")
	}
	if a.EventType != b.EventType {
		out = append(out, "EventType")
	}
	if a.Turn != b.Turn {
		out = append(out, "Turn")
	}
	if a.Step != b.Step {
		out = append(out, "Step")
	}
	if len(a.Segments) != len(b.Segments) || msgJSON(Message{Content: segDump(a.Segments)}) != msgJSON(Message{Content: segDump(b.Segments)}) {
		out = append(out, "Segments")
	}
	if msgJSON(a.Message) != msgJSON(b.Message) {
		out = append(out, messageDiffFields(a.Message, b.Message)...)
	}
	return out
}

func segDump(segs []Segment) string {
	b, _ := json.Marshal(segs)
	return string(b)
}

// messageDiffFields 逐字段比较 Message，返回变化的字段名。
func messageDiffFields(a, b Message) []string {
	var out []string
	if a.Role != b.Role {
		out = append(out, "Role")
	}
	if a.Content != b.Content {
		out = append(out, "Content")
	}
	if msgJSON(Message{ToolCalls: a.ToolCalls}) != msgJSON(Message{ToolCalls: b.ToolCalls}) {
		out = append(out, "ToolCalls")
	}
	if a.ToolCallID != b.ToolCallID {
		out = append(out, "ToolCallID")
	}
	if a.Name != b.Name {
		out = append(out, "Name")
	}
	if a.Reasoning != b.Reasoning {
		out = append(out, "Reasoning")
	}
	if msgJSON(Message{Images: a.Images}) != msgJSON(Message{Images: b.Images}) {
		out = append(out, "Images")
	}
	if len(out) == 0 {
		out = append(out, "Message(未知字段)")
	}
	return out
}
