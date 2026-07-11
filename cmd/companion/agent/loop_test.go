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

// 端到端 TAOR：MockProvider 脚本「第1轮调 read_file → 第2轮调 finish_task」，
// 验证 think→act→observe(结果回灌)→think→done 全链路。
func TestLoopToolThenFinal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("WORLD_123"), 0o644)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"hello.txt"}`}}}},
		{ToolCalls: []ToolCall{{ID: "f1", Type: "function", Function: FunctionCall{Name: "finish_task", Arguments: `{"result":"读到了 WORLD_123"}`}}}},
	}}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5,
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
		t.Error("未把 read_file 结果作 role=tool 消息回灌")
	}

	// 末事件应为 done（finish_task），且内容含 WORLD_123
	last := events[len(events)-1]
	if last.Type != EventDone || last.DoneReason != "finish_task" || !strings.Contains(last.Content, "WORLD_123") {
		t.Errorf("末事件应为 EventDone(finish_task)，得 %+v", last)
	}

	// 应广播过 tool_call(read_file) 与 tool_result(含结果) 事件
	var sawCall, sawResult bool
	for _, e := range events {
		if e.Type == EventToolCall && e.Tool == "read_file" {
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

// 调 finish_task 工具 → 立即退出，仅需 1 次 LLM 调用。
func TestLoopFinishTaskExits(t *testing.T) {
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "f1", Type: "function", Function: FunctionCall{Name: "finish_task", Arguments: `{"result":"任务完成"}`}}}},
	}}
	loop := &Loop{Provider: mock, Registry: NewRegistry(), MaxIterations: 5}
	if _, err := loop.Run(context.Background(), "完成", nil); err != nil {
		t.Fatal(err)
	}
	if mock.Calls() != 1 {
		t.Errorf("应 1 次 LLM 调用，得 %d", mock.Calls())
	}
}

// 永远调用工具、从不 finish_task → 在 MaxIterations 处止损。
type alwaysToolProvider struct{ n int }

func (a *alwaysToolProvider) Name() string { return "always" }
func (a *alwaysToolProvider) Chat(ctx context.Context, m []Message, td []ToolDefinition, oc func(Chunk)) (Message, error) {
	a.n++
	return Message{Role: RoleAssistant, ToolCalls: []ToolCall{
		{ID: "x", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"x.txt"}`}},
	}}, nil
}

func TestLoopMaxIterations(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0o644)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	prov := &alwaysToolProvider{}
	var lastErr string
	loop := &Loop{Provider: prov, Registry: reg, MaxIterations: 3,
		OnEvent: func(e Event) {
			if e.Type == EventError {
				lastErr = e.Content
			}
		}}
	loop.Run(context.Background(), "loop forever", nil)
	if prov.n != 3 {
		t.Errorf("应调用 3 次(=MaxIterations)，得 %d", prov.n)
	}
	if !strings.Contains(lastErr, "最大迭代") {
		t.Errorf("应因最大迭代停止，lastErr=%q", lastErr)
	}
}

// 审批拒绝：写类工具被 Approve 拒绝 → 不执行、不写盘，拒绝作观察回灌，最终调 finish_task 退出。
func TestLoopApprovalReject(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "w1", Type: "function", Function: FunctionCall{Name: "write_file", Arguments: `{"path":"out.txt","content":"DATA"}`}}}},
		{ToolCalls: []ToolCall{{ID: "f1", Type: "function", Function: FunctionCall{Name: "finish_task", Arguments: `{"result":"放弃写文件"}`}}}},
	}}
	var approvedTools []string
	loop := &Loop{Provider: mock, Registry: reg, MaxIterations: 5,
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) { approvedTools = append(approvedTools, tc.Function.Name); return false, "" }}

	msgs, err := loop.Run(context.Background(), "写个文件", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, e := os.Stat(filepath.Join(dir, "out.txt")); e == nil {
		t.Error("被拒绝的 write_file 不应写盘")
	}
	if len(approvedTools) != 1 || approvedTools[0] != "write_file" {
		t.Errorf("Approve 应被 write_file 调用一次，得 %v", approvedTools)
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
		{ToolCalls: []ToolCall{{ID: "w1", Type: "function", Function: FunctionCall{Name: "write_file", Arguments: `{"path":"out.txt","content":"DATA"}`}}}},
		{ToolCalls: []ToolCall{{ID: "f1", Type: "function", Function: FunctionCall{Name: "finish_task", Arguments: `{"result":"已写入文件"}`}}}},
	}}
	loop := &Loop{Provider: mock, Registry: reg, MaxIterations: 5,
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) { return true, "" }}
	if _, err := loop.Run(context.Background(), "写个文件", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if b, e := os.ReadFile(filepath.Join(dir, "out.txt")); e != nil || string(b) != "DATA" {
		t.Errorf("通过审批的 write_file 应写盘，得 %q err=%v", b, e)
	}
}

// 只读工具不经审批门：即便设了 Approve，read_file（RequiresApproval=false）也不应触发它。
func TestLoopApprovalSkipsReadOnly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.txt"), []byte("hi"), 0o644)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "r1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"x.txt"}`}}}},
		{ToolCalls: []ToolCall{{ID: "f1", Type: "function", Function: FunctionCall{Name: "finish_task", Arguments: `{"result":"已读取文件"}`}}}},
	}}
	called := false
	loop := &Loop{Provider: mock, Registry: reg, MaxIterations: 5,
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) { called = true; return false, "" }}
	if _, err := loop.Run(context.Background(), "读 x.txt", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if called {
		t.Error("只读工具 read_file 不应触发审批门")
	}
}

// 外部 ctx 取消 → 立即返回 ctx 错误。
func TestLoopContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mock := &MockProvider{Responses: []Message{{ToolCalls: []ToolCall{{ID: "f1", Type: "function", Function: FunctionCall{Name: "finish_task", Arguments: `{"result":"x"}`}}}}}}
	loop := &Loop{Provider: mock, Registry: NewRegistry(), MaxIterations: 5}
	if _, err := loop.Run(ctx, "task", nil); err == nil {
		t.Error("已取消的 ctx 应使 Run 返回错误")
	}
}
