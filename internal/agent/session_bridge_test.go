package agent

import (
	"context"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════
// session_bridge_test.go — 会话桥（ask_user/task_create 插件化路由）
//
// 覆盖：ctx 会话标识注入 → JS 插件工具 _convID 注入 → hostTool 路由执行器
// → SessionBridge（WaitAnswer / GetWorkspaceRoot）→ 会话按 convID 精确路由。
// ═══════════════════════════════════════════════════════════════

// TestSessionConvID 验证 ctx 会话标识往返。
func TestSessionConvID(t *testing.T) {
	if got := SessionConvID(nil); got != "" {
		t.Fatalf("nil ctx 应为空串，got %q", got)
	}
	if got := SessionConvID(context.Background()); got != "" {
		t.Fatalf("无注入 ctx 应为空串，got %q", got)
	}
	ctx := WithSessionConvID(context.Background(), "conv-1")
	if got := SessionConvID(ctx); got != "conv-1" {
		t.Fatalf("期望 conv-1，got %q", got)
	}
}

// TestSessionBridgeRouting 验证路由执行器：注入假桥后按 _convID 路由。
func TestSessionBridgeRouting(t *testing.T) {
	// ★ 注：UseTaskManager 为进程级单例（tmInitOnce，root 仅首调生效），
	//   与其他测试（如 TestPluginTakeoverHostTool 的 TempDir）共享实例、
	//   且 TempDir 在测试结束后即被清理——本测试不依赖任务文件落盘，
	//   只验证路由执行器返回值与桥调用（convID 透传）。
	root := t.TempDir()
	// 注入假桥（记录收到的 convID）
	var gotConv string
	SetSessionBridge(&SessionBridge{
		WaitAnswer: func(ctx context.Context, convID string) (string, error) {
			gotConv = convID
			return "用户回答", nil
		},
		GetWorkspaceRoot: func(convID string) string {
			return root
		},
	})
	defer SetSessionBridge(nil)

	// task_create 路由：成功创建（handler 走通，convID 透传给桥）
	out, err := ExecuteHostTool("task_create", map[string]any{
		"_convID":     "conv-9",
		"subject":     "测试任务",
		"description": "会话桥路由验证",
	})
	if err != nil {
		t.Fatalf("task_create 路由失败: %v", err)
	}
	if !strings.HasPrefix(out, "✅ 已创建任务") {
		t.Fatalf("task_create 应创建成功，got %q", out)
	}

	// ask_user 路由：假桥返回回答
	answer, err := ExecuteHostTool("ask_user", map[string]any{
		"_convID":  "conv-9",
		"question": "测试问题",
	})
	if err != nil {
		t.Fatalf("ask_user 路由失败: %v", err)
	}
	if answer != "用户回答" {
		t.Fatalf("期望假桥回答，got %q", answer)
	}
	if gotConv != "conv-9" {
		t.Fatalf("桥应收到 conv-9，got %q", gotConv)
	}
}

// TestJSToolConvIDInjection 验证 JS 插件工具 execute 收到 _convID 注入
// （jsToolToGo 包装：Loop ctx 链带 convID → 复制 args 注入内部键）。
func TestJSToolConvIDInjection(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, t.TempDir())

	code := `
return {
  name: 'conv-inject-test',
  apply(ctx) {
    ctx.tools.register({
      name: 'echo_conv',
      description: '回显会话标识',
      parameters: { type: 'object', properties: {} },
      execute: (args) => JSON.stringify({ conv: args._convID || null, has: '_convID' in args }),
    })
  },
}
`
	id, err := host.DefineJSCodeFull(code, "js", "会话注入测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if def == nil {
		t.Fatalf("定义不存在")
	}
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}
	MergePluginTools(reg, host)

	tool, ok := reg.Get("echo_conv")
	if !ok {
		t.Fatal("echo_conv 未注册")
	}

	// ① 带会话标识的 ctx → execute 收到 _convID
	ctx := WithSessionConvID(context.Background(), "conv-42")
	out, err := tool.Handler(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if !strings.Contains(out, `"conv":"conv-42"`) {
		t.Fatalf("execute 应收到 conv-42，got %q", out)
	}

	// ② 无会话 ctx → 不注入（args 保持原样）
	out2, err := tool.Handler(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if !strings.Contains(out2, `"conv":null`) {
		t.Fatalf("无会话时不应注入，got %q", out2)
	}

	// ③ 原 args 不被污染（AfterTool 观察用原值）
	orig := map[string]any{"k": "v"}
	if _, err := tool.Handler(ctx, orig); err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if _, has := orig["_convID"]; has {
		t.Fatal("原 args 不应被注入 _convID（复制注入）")
	}
}
