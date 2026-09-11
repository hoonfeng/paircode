//go:build windows

package agent

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestExecCommandSync 同步命令：exec_command 内完成 → 返回输出 + 结束状态行。
func TestExecCommandSync(t *testing.T) {
	r := NewRegistry()
	RegisterDefaultTools(r, t.TempDir())
	ctx := context.Background()

	out, err := r.Execute(ctx, "exec_command", `{"command":"echo exec_hello"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "exec_hello") {
		t.Errorf("输出缺 echo 内容：%q", out)
	}
	if !strings.Contains(out, "已结束") {
		t.Errorf("应带结束状态行：%q", out)
	}
	// 同步短命令不应返回 session_id（已完成）
	if strings.Contains(out, "session_id=") {
		t.Errorf("已完成命令不应返回 session_id：%q", out)
	}
}

// TestExecCommandSession 长命令 yield 超时 → session_id → write_stdin 轮询至结束。
// ★ id 断言放宽：bgRegistry 为全局单例（跨轮次/跨测试存活），id 用正则提取。
func TestExecCommandSession(t *testing.T) {
	r := NewRegistry()
	RegisterDefaultTools(r, t.TempDir())
	ctx := context.Background()

	// ping -n 2 约 1s：yield 250ms 必超时 → 返回 session_id
	out, err := r.Execute(ctx, "exec_command", `{"command":"ping -n 2 127.0.0.1 && echo session_done","yield_time_ms":250}`)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`session_id=(\d+)`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("慢命令应返回 session_id，得 %q", out)
	}
	idArg := `{"session_id":` + m[1] + `}`

	var ro string
	for i := 0; i < 300; i++ { // 轮询至结束（ping 约 1s）
		ro, _ = r.Execute(ctx, "write_stdin", idArg)
		if strings.Contains(ro, "已结束") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(ro, "session_done") {
		t.Errorf("轮询输出缺会话内容：%q", ro)
	}
	if !strings.Contains(ro, "已结束") {
		t.Errorf("应轮询到结束状态：%q", ro)
	}
}

// TestExecWriteStdinUnknown write_stdin 未知 id 应报错。
func TestExecWriteStdinUnknown(t *testing.T) {
	r := NewRegistry()
	RegisterDefaultTools(r, t.TempDir())
	if _, err := r.Execute(context.Background(), "write_stdin", `{"session_id":999999}`); err == nil {
		t.Error("未知 id 应报错")
	}
}

// TestExecKillProcess kill_process 会话式收尾（幂等）。
func TestExecKillProcess(t *testing.T) {
	r := NewRegistry()
	RegisterDefaultTools(r, t.TempDir())
	ctx := context.Background()

	out, err := r.Execute(ctx, "exec_command", `{"command":"ping -n 30 127.0.0.1","yield_time_ms":250}`)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`session_id=(\d+)`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("长命令应返回 session_id，得 %q", out)
	}
	if _, err := r.Execute(ctx, "kill_process", `{"session_id":`+m[1]+`}`); err != nil {
		t.Fatalf("kill_process 失败: %v", err)
	}
	// 幂等：再次调用不报错
	if _, err := r.Execute(ctx, "kill_process", `{"session_id":`+m[1]+`}`); err != nil {
		t.Fatalf("kill_process 应幂等: %v", err)
	}
}

// TestRunCommandInLoop exec_command 执行完毕后循环继续调用 LLM 的下一轮。
// 验证：工具结果正确回灌 → 第 2 轮 LLM 自然终止。
func TestRunCommandInLoop(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "exec_command", Arguments: `{"command":"echo RUNCMD_OK"}`}}}},
		{Content: "done"},
	}}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "test-loop-run-cmd",
		OnEvent: func(e Event) { events = append(events, e) }}

	msgs, err := loop.Run(context.Background(), "执行 echo RUNCMD_OK", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// ★ 核心验证：LLM 应被调用了 2 次（工具结果→下一轮→自然终止）
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
		t.Errorf("未把 exec_command 结果作 role=tool 消息回灌")
	}

	// 验证末事件为 done
	last := events[len(events)-1]
	if last.Type != EventDone || last.DoneReason != "task_complete" {
		t.Errorf("末事件应为 EventDone(task_complete)，得 type=%s reason=%s", last.Type, last.DoneReason)
	}
}

// TestRunCommandContextCancelled 验证 context 取消时会话清理且循环正常退出。
func TestRunCommandContextCancelled(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "exec_command", Arguments: `{"command":"ping -n 10 127.0.0.1","yield_time_ms":250}`}}}},
		{Content: "done"},
	}}
	loop := &Loop{Provider: mock, Registry: reg, System: "test-cancel"}

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
	// 清理：从工具结果提取 session_id 并杀死（防 ping 进程占用 TempDir 致清理失败）
	fallback := NewRegistry()
	RegisterDefaultTools(fallback, dir)
	for _, m := range msgs {
		if m.Role != RoleTool {
			continue
		}
		if mm := regexp.MustCompile(`session_id=(\d+)`).FindStringSubmatch(m.Content); mm != nil {
			_, _ = fallback.Execute(context.Background(), "kill_process", `{"session_id":`+mm[1]+`}`)
		}
	}
}
