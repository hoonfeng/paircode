package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestAskUserNoHostTimeout ask_user 等阻塞型工具默认不限时（2026-08-22 调整）：
// 宿主不再为工具 execute 强加 30s 的 jsToolTimeout——ask_user 等待用户回答
// 期间不应被 VM Interrupt 打断（否则 Agent 收到 Error 重试 → 提问重复显示），
// 真实超时由会话层（session_manager）控制；插件如需护栏自行声明 timeout。
// 端到端：假会话桥 250ms 后回答，工具（无 timeout 声明）应正常返回。
func TestAskUserNoHostTimeout(t *testing.T) {
	// 假会话桥：WaitAnswer 模拟用户 250ms 后回答
	SetSessionBridge(&SessionBridge{
		WaitAnswer: func(ctx context.Context, convID string) (string, error) {
			select {
			case <-time.After(250 * time.Millisecond):
				return "用户回答：方案B", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		},
		GetWorkspaceRoot: func(convID string) string { return "C:/ws" },
	})
	defer SetSessionBridge(&SessionBridge{})

	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(`
	return {
		name: 'askuser-wait',
		apply(ctx) {
			ctx.tools.register({
				name: 'ask_user',
				description: 'ask',
				parameters: { type: 'object', properties: {}, required: [] },
				execute: (args) => ctx.hostTool.exec('ask_user', args || {}),
			})
		},
	}`, "askuser-wait")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	start := time.Now()
	out, err := reg.Execute(context.Background(), "ask_user", `{"question":"Q","_convID":"conv-1"}`)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("ask_user 不应被 JS 超时中断: err=%v", err)
	}
	if !strings.Contains(out, "方案B") {
		t.Fatalf("应返回用户回答, got %q", out)
	}
	if strings.Contains(out, "执行超时") {
		t.Fatalf("不应出现执行超时: %q", out)
	}
	if elapsed < 200*time.Millisecond {
		t.Fatalf("等待窗口不合理: %v", elapsed)
	}
}
