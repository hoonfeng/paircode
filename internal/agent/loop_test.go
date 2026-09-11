package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProjectRules 工作区有 AGENTS.md → 作为项目约定拼进提示；无则空。
func TestProjectRules(t *testing.T) {
	dir := t.TempDir()
	if got := ProjectRules(dir); got != "" {
		t.Errorf("无约定文件应返回空，得 %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# 规矩\n用 4 空格缩进"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ProjectRules(dir)
	if !strings.Contains(got, "用 4 空格缩进") || !strings.Contains(got, "AGENTS.md") {
		t.Errorf("应含约定内容与来源，得 %q", got)
	}
}

// 端到端 TAOR：MockProvider 脚本「第1轮调 read → 第2轮自然终止」，
// 验证 think→act→observe(结果回灌)→think→done 全链路。
func TestLoopToolThenFinal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("WORLD_123"), 0o644)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read", Arguments: `{"path":"hello.txt"}`}}}},
		{Content: "读到了 WORLD_123"},
	}}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "test",
		OnEvent: func(e Event) { events = append(events, e) }}

	msgs, err := loop.Run(context.Background(), "读 hello.txt 告诉我内容", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if mock.Calls() != 2 {
		t.Errorf("LLM 应调用 2 次，得 %d", mock.Calls())
	}

	// 观察回灌：应有一条 role=tool 且含 WORLD_123
	foundTool := false
	for _, m := range msgs {
		if m.Role == RoleTool && strings.Contains(m.Content, "WORLD_123") {
			foundTool = true
		}
	}
	if !foundTool {
		t.Error("未把 read 结果作 role=tool 消息回灌")
	}

	// 末事件应为 done（task_complete），且内容含 WORLD_123
	last := events[len(events)-1]
	if last.Type != EventDone || last.DoneReason != "task_complete" || !strings.Contains(last.Content, "WORLD_123") {
		t.Errorf("末事件应为 EventDone(task_complete)，得 %+v", last)
	}

	// 应广播过 tool_call(read) 与 tool_result(含结果) 事件
	var sawCall, sawResult bool
	for _, e := range events {
		if e.Type == EventToolCall && e.Tool == "read" {
			sawCall = true
		}
		if e.Type == EventToolResult && strings.Contains(e.Content, "WORLD_123") {
			sawResult = true
		}
	}
	if !sawCall || !sawResult {
		t.Errorf("缺事件：tool_call=%v tool_result=%v", sawCall, sawResult)
	}
}

// 自然终止（无工具调用 + 正文内容）→ 立即退出，仅需 1 次 LLM 调用。
func TestLoopNaturalFinish(t *testing.T) {
	mock := &MockProvider{Responses: []Message{
		{Content: "任务完成"},
	}}
	loop := &Loop{Provider: mock, Registry: NewRegistry()}
	if _, err := loop.Run(context.Background(), "完成", nil); err != nil {
		t.Fatal(err)
	}
	if mock.Calls() != 1 {
		t.Errorf("应 1 次 LLM 调用，得 %d", mock.Calls())
	}
}

// 永远调用工具、从不自然终止 → 由工具预算耗尽结束本段（并标记自动续跑）。
type alwaysToolProvider struct{ n int }

func (a *alwaysToolProvider) Name() string { return "always" }
func (a *alwaysToolProvider) Chat(ctx context.Context, m []Message, td []ToolDefinition, oc func(Chunk)) (Message, error) {
	a.n++
	return Message{Role: RoleAssistant, ToolCalls: []ToolCall{
		{ID: "x", Type: "function", Function: FunctionCall{Name: "read", Arguments: `{"path":"x.txt"}`}},
	}}, nil
}

// ★ 2026-09-12：原「最大迭代数」配置项已移除——段不再由「迭代数」止损，改由
// 工具预算（本测试用 3 次）结束本段并标记续跑（SessionManager 自动开下一段）；
// 段内迭代安全上限由预算派生（tool_budget.go IterationLimit），仅作防失控兜底。
func TestLoopToolBudgetSegment(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0o644)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	prov := &alwaysToolProvider{}
	loop := &Loop{Provider: prov, Registry: reg, ToolCallBudget: 3}
	if _, err := loop.Run(context.Background(), "loop forever", nil); err != nil {
		t.Fatalf("预算耗尽应正常结束本段（不报错），得 %v", err)
	}
	if prov.n != 3 {
		t.Errorf("应调用 3 次(=工具预算)，得 %d", prov.n)
	}
	if st, ok := loop.TakeSegmentContinue(); !ok || st.UsedTools != 3 || st.ToolBudget != 3 {
		t.Errorf("应标记分段续跑（工具闸 3/3），得 ok=%v state=%+v", ok, st)
	}
}

// 审批拒绝：写类工具被 Approve 拒绝 → 不执行、不写盘，拒绝作观察回灌，最终自然终止退出。
func TestLoopApprovalReject(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "w1", Type: "function", Function: FunctionCall{Name: "write", Arguments: `{"path":"out.txt","content":"DATA"}`}}}},
		{Content: "放弃写文件"},
	}}
	var approvedTools []string
	loop := &Loop{Provider: mock, Registry: reg,
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) {
			approvedTools = append(approvedTools, tc.Function.Name)
			return false, ""
		}}

	msgs, err := loop.Run(context.Background(), "写个文件", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, e := os.Stat(filepath.Join(dir, "out.txt")); e == nil {
		t.Error("被拒绝的 write 不应写盘")
	}
	if len(approvedTools) != 1 || approvedTools[0] != "write" {
		t.Errorf("Approve 应被 write 调用一次，得 %v", approvedTools)
	}
	var fedBack bool
	for _, m := range msgs {
		if m.Role == RoleTool && strings.Contains(m.Content, "拒绝") {
			fedBack = true
		}
	}
	if !fedBack {
		t.Error("拒绝应作为 role=tool 观察回灌给模型")
	}
}

// 审批通过：Approve 返回 true → 工具正常执行、写盘。
func TestLoopApprovalApprove(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "w1", Type: "function", Function: FunctionCall{Name: "write", Arguments: `{"path":"out.txt","content":"DATA"}`}}}},
		{Content: "已写入文件"},
	}}
	loop := &Loop{Provider: mock, Registry: reg,
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) { return true, "" }}
	if _, err := loop.Run(context.Background(), "写个文件", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if b, e := os.ReadFile(filepath.Join(dir, "out.txt")); e != nil || string(b) != "DATA" {
		t.Errorf("通过审批的 write 应写盘，得 %q err=%v", b, e)
	}
}

// 只读工具不经审批门：即便设了 Approve，read（RequiresApproval=false）也不应触发它。
func TestLoopApprovalSkipsReadOnly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.txt"), []byte("hi"), 0o644)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "r1", Type: "function", Function: FunctionCall{Name: "read", Arguments: `{"path":"x.txt"}`}}}},
		{Content: "已读取文件"},
	}}
	called := false
	loop := &Loop{Provider: mock, Registry: reg,
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) { called = true; return false, "" }}
	if _, err := loop.Run(context.Background(), "读 x.txt", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if called {
		t.Error("只读工具 read 不应触发审批门")
	}
}

// 外部 ctx 取消 → 立即返回 ctx 错误。
func TestLoopContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mock := &MockProvider{Responses: []Message{{Content: "已取消"}}}
	loop := &Loop{Provider: mock, Registry: NewRegistry()}
	if _, err := loop.Run(ctx, "task", nil); err == nil {
		t.Error("已取消的 ctx 应使 Run 返回错误")
	}
}

// ── 新增组件单元测试 ──

// ★ 2026-09：TestCanParallelize 已随「工具并行执行」能力删除——
//   canParallelize / tryParallelExecute / executeReadOnlyParallel（loop_parallel.go）
//   整体移除，工具调用一律串行执行。串行等价行为由
//   jsloop_e2e_test.go TestJSLoopMultipleToolCallsSerial 覆盖。

// TestToolError 验证统一错误类型。
func TestToolError(t *testing.T) {
	e := NewToolError("edit", "替换失败", WithRetryable(true), WithSuggestion("请用行号定位"), WithSeverity("warn"))
	if !e.Retryable {
		t.Error("应可重试")
	}
	if e.Severity != "warn" {
		t.Errorf("severity 应为 warn: %s", e.Severity)
	}
	msg := e.Error()
	if !strings.Contains(msg, "edit") || !strings.Contains(msg, "替换失败") || !strings.Contains(msg, "行号定位") {
		t.Errorf("错误消息不完整: %s", msg)
	}
	e2 := NewToolError("run", "超时")
	if e2.Severity != "error" {
		t.Errorf("默认 severity=error: %s", e2.Severity)
	}
}

// TestHookStore 验证多钩子优先级+短路+移除。
func TestHookStore(t *testing.T) {
	hs := NewHookStore()
	var calls []string
	hs.Add(&Hook{Name: "b2", Kind: HookBefore, Priority: 200, BeforeFn: func(ctx context.Context, n string, a map[string]any) (bool, string, error) {
		calls = append(calls, "b2")
		return true, "", nil
	}})
	hs.Add(&Hook{Name: "b1", Kind: HookBefore, Priority: 50, BeforeFn: func(ctx context.Context, n string, a map[string]any) (bool, string, error) {
		calls = append(calls, "b1")
		return true, "", nil
	}})
	hs.ExecuteBefore(context.Background(), "test", nil)
	if len(calls) != 2 || calls[0] != "b1" || calls[1] != "b2" {
		t.Errorf("应 b1 先于 b2: %v", calls)
	}
	// 短路
	hs2 := NewHookStore()
	hs2.Add(&Hook{Name: "s1", Kind: HookBefore, BeforeFn: func(ctx context.Context, n string, a map[string]any) (bool, string, error) {
		return false, "blocked", nil
	}})
	var shortCalled bool
	hs2.Add(&Hook{Name: "s2", Kind: HookBefore, BeforeFn: func(ctx context.Context, n string, a map[string]any) (bool, string, error) {
		shortCalled = true
		return true, "", nil
	}})
	proceed, override, _ := hs2.ExecuteBefore(context.Background(), "test", nil)
	if proceed || override != "blocked" || shortCalled {
		t.Error("短路钩子应阻止后续")
	}
	// Remove
	hs.Remove("b1")
	calls = nil
	hs.ExecuteBefore(context.Background(), "test", nil)
	if len(calls) != 1 || calls[0] != "b2" {
		t.Errorf("Remove b1 后仅 b2: %v", calls)
	}
}

// ★ 2026-08-21 Live 快照（WS 断线补偿）：emit 累积 content/thinking/tool 事件，
// SnapshotLive 返回完整进度；resetLive 清空（新 turn 占位）。
// ★ 2026-08-22 扩展：liveEvents 有序序列保真时序——前端逐事件重放可还原
//   「thinking→content→tool_call→content」的真实交错顺序（不聚集成工具全上/正文全下）。
func TestLoopLiveSnapshot(t *testing.T) {
	l := &Loop{}
	l.emit(Event{Type: EventThinking, Content: "分析问题"})
	l.emit(Event{Type: EventContent, Content: "我先看看代码"})
	l.emit(Event{Type: EventToolCall, Tool: "read", Args: `{"path":"main.go"}`, CallID: "c1"})
	l.emit(Event{Type: EventToolResult, Tool: "read", Content: "package main", CallID: "c1"})
	l.emit(Event{Type: EventContent, Content: "代码结构清晰"})

	content, reasoning, tools, events := l.LiveSnapshot()
	if content != "我先看看代码代码结构清晰" {
		t.Errorf("content = %q", content)
	}
	if reasoning != "分析问题" {
		t.Errorf("reasoning = %q", reasoning)
	}
	if len(tools) != 1 {
		t.Fatalf("应 1 个工具段，得 %d", len(tools))
	}
	if tools[0].Name != "read" || tools[0].Args != `{"path":"main.go"}` || tools[0].Result != "package main" {
		t.Errorf("工具段 = %+v", tools[0])
	}

	// ★ 事件序列顺序保真：thinking → content → tool_call → content（重放后不丢失交错）
	wantTypes := []string{"thinking", "content", "tool_call", "content"}
	if len(events) != len(wantTypes) {
		t.Fatalf("事件序列长度 = %d，期望 %d: %+v", len(events), len(wantTypes), events)
	}
	for i, wt := range wantTypes {
		if events[i].Type != wt {
			t.Errorf("事件[%d] type = %q，期望 %q", i, events[i].Type, wt)
		}
	}
	// 连续 thinking/content 增量合并
	if events[0].Content != "分析问题" {
		t.Errorf("thinking 事件内容 = %q", events[0].Content)
	}
	if events[1].Content != "我先看看代码" {
		t.Errorf("content 事件内容 = %q", events[1].Content)
	}
	if events[2].CallID != "c1" || events[2].Content != "package main" {
		t.Errorf("tool_call 事件回填 result 失败: %+v", events[2])
	}

	// resetLive（新 turn）清空
	l.resetLive()
	content, reasoning, tools, events = l.LiveSnapshot()
	if content != "" || reasoning != "" || len(tools) != 0 || len(events) != 0 {
		t.Errorf("resetLive 后应清空: content=%q reasoning=%q tools=%d events=%d", content, reasoning, len(tools), len(events))
	}
}

// ★ 2026-08-22 连续流式增量合并：thinking/content 多次 emit 应合并为单条事件。
func TestLoopLiveSnapshotMergeIncrements(t *testing.T) {
	l := &Loop{}
	l.emit(Event{Type: EventContent, Content: "你好，"})
	l.emit(Event{Type: EventContent, Content: "世界"})
	l.emit(Event{Type: EventThinking, Content: "想"})
	l.emit(Event{Type: EventThinking, Content: "考"})
	_, _, _, events := l.LiveSnapshot()
	if len(events) != 2 {
		t.Fatalf("应合并为 2 条事件，得 %d: %+v", len(events), events)
	}
	if events[0].Type != "content" || events[0].Content != "你好，世界" {
		t.Errorf("content 合并失败: %+v", events[0])
	}
	if events[1].Type != "thinking" || events[1].Content != "想考" {
		t.Errorf("thinking 合并失败: %+v", events[1])
	}
}
