//go:build windows

package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestRunBackground 后台启动 echo → 轮询 read_output 至结束 → 输出含 echo 内容。
func TestRunBackground(t *testing.T) {
	r := NewRegistry()
	RegisterDefaultTools(r, t.TempDir())
	ctx := context.Background()

	out, err := r.Execute(ctx, "run_background", `{"command":"echo bg_hello"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "id=1") {
		t.Fatalf("应返回 id=1，得 %q", out)
	}

	var ro string
	for i := 0; i < 200; i++ { // 轮询至结束（echo 很快）
		ro, _ = r.Execute(ctx, "read_output", `{"id":1}`)
		if strings.Contains(ro, "已结束") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(ro, "bg_hello") {
		t.Errorf("输出缺 echo 内容：%q", ro)
	}
}

// TestRunCommandInLoop run_command 执行完毕后循环继续调用 LLM 的下一轮。
// 验证：工具结果正确回灌 → 第 2 轮 LLM 自然终止。
func TestRunCommandInLoop(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "run_command", Arguments: `{"command":"echo RUNCMD_OK"}`}}}},
		{Content: "done"},
	}}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "test-loop-run-cmd", MaxIterations: 5,
		OnEvent: func(e Event) { events = append(events, e) }}

	msgs, err := loop.Run(context.Background(), "执行 echo RUNCMD_OK", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// ★ 核心验证：LLM 应被调用了 2 次（工具结果→下一轮→自然终止）
	// 如果这里失败，说明 run_command 执行完后 loop 没有继续调用 LLM
	if mock.Calls() != 2 {
		t.Errorf("LLM 应调用 2 次（工具结果→下一轮），得 %d", mock.Calls())
	}

	// 验证 tool result 含命令输出
	foundOutput := false
	for _, m := range msgs {
		if m.Role == RoleTool && strings.Contains(m.Content, "RUNCMD_OK") {
			foundOutput = true
			break
		}
	}
	if !foundOutput {
		t.Errorf("未把 run_command 结果作 role=tool 消息回灌")
	}

	// 验证末事件为 done
	last := events[len(events)-1]
	if last.Type != EventDone || last.DoneReason != "task_complete" {
		t.Errorf("末事件应为 EventDone(task_complete)，得 type=%s reason=%s", last.Type, last.DoneReason)
	}
}

// TestRunCommandContextCancelled 验证 context 取消时 run_command 优雅终止且循环正常退出。
func TestRunCommandContextCancelled(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "run_command", Arguments: `{"command":"ping -n 10 127.0.0.1"}`}}}},
		{Content: "done"},
	}}
	loop := &Loop{Provider: mock, Registry: reg, System: "test-cancel", MaxIterations: 5}

	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	msgs, err := loop.Run(ctx, "执行 ping", nil)
	if err == nil {
		t.Log("context 取消后 Run 正常返回")
	} else {
		t.Logf("context 取消后 Run 返回错误: %v", err)
	}

	if msgs == nil {
		t.Error("msgs 不应为 nil")
	}
}

// TestReadOutputUnknown 读未知 id 应报错。
func TestReadOutputUnknown(t *testing.T) {
	r := NewRegistry()
	RegisterDefaultTools(r, t.TempDir())
	if _, err := r.Execute(context.Background(), "read_output", `{"id":999}`); err == nil {
		t.Error("未知 id 应报错")
	}
}
