package agent

import (
	"context"
	"testing"
)

// TestNodeBridgeBindHost 验证：桥工具注册到旧宿主后，bindHost 新宿主时
// 补注册（agent 实例有独立 registry，新会话必须能看到桥工具）。
func TestNodeBridgeBindHost(t *testing.T) {
	store := NewMessageStore(t.TempDir())
	reg1 := NewRegistry()
	ph1 := NewPluginHost(reg1, store, t.TempDir())
	reg2 := NewRegistry()
	ph2 := NewPluginHost(reg2, store, t.TempDir())

	b := &nodeBridge{ph: ph1, dir: t.TempDir(), pending: map[int64]chan bridgeResult{}, tools: map[string]*Tool{}}
	// 模拟 handleToolMsg 注册到旧宿主
	tool := &Tool{Name: "hello_bridge", Description: "test", Handler: func(ctx context.Context, args map[string]any) (string, error) { return "pong", nil }}
	b.tools["hello_bridge"] = tool
	if err := ph1.Context().RegisterTool(tool); err != nil {
		t.Fatalf("注册到 ph1 失败: %v", err)
	}
	if _, ok := reg2.Get("hello_bridge"); ok {
		t.Fatalf("ph2 不应预先有工具")
	}

	// 新宿主 bindHost → 补注册
	b.bindHost(ph2)
	if _, ok := reg2.Get("hello_bridge"); !ok {
		t.Fatalf("bindHost 后 ph2 应有 hello_bridge（补注册失败）")
	}
	if got := b.ph; got != ph2 {
		t.Fatalf("bindHost 后 ph 应为最新宿主")
	}
	t.Log("bindHost 补注册 OK")
}

// TestMergePluginTools 验证：会话级 reg 合并插件注册的业务工具。
func TestMergePluginTools(t *testing.T) {
	store := NewMessageStore(t.TempDir())
	phReg := NewRegistry()
	ph := NewPluginHost(phReg, store, t.TempDir())
	// 插件注册业务工具（Node 桥 / goja 插件）
	pt := &Tool{Name: "hello_bridge", Description: "plugin tool", Handler: func(ctx context.Context, args map[string]any) (string, error) { return "pong", nil }}
	if err := ph.Context().RegisterTool(pt); err != nil {
		t.Fatal(err)
	}
	// 宿主内置工具（无插件归属）
	phReg.Register(&Tool{Name: "read", Description: "builtin", Handler: noopHandler})

	reg := NewRegistry()
	reg.Register(&Tool{Name: "read", Description: "session builtin", Handler: noopHandler}) // 会话同名内置
	MergePluginTools(reg, ph)

	if _, ok := reg.Get("hello_bridge"); !ok {
		t.Fatalf("合并后应有插件工具 hello_bridge")
	}
	// 同名内置不覆盖
	if tool2, _ := reg.Get("read"); tool2.Description != "session builtin" {
		t.Fatalf("会话内置工具不应被插件宿主覆盖: %s", tool2.Description)
	}
	t.Log("MergePluginTools OK")
}
