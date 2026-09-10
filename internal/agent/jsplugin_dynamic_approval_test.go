package agent

import (
	"context"
	"testing"
)

// TestJSToolDynamicApproval 验证 JS 工具 dynamicApproval 桥（2026-09 工具面合并支撑）：
// 插件注册带 dynamicApproval 的工具 → Tool.DynamicApproval 生效（按调用 args 判定）；
// 未声明时 DynamicApproval==nil（除 RequiresApproval/既有行为外零变化）。
func TestJSToolDynamicApproval(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, t.TempDir())

	const code = `return {
	  name: 'dyn-approval-demo',
	  apply(ctx) {
	    ctx.tools.register({
	      name: 'dyn_demo_tool',
	      description: '动态审批测试工具',
	      parameters: { type: 'object', properties: { op: { type: 'string' } } },
	      // 只写类 op 需要审批；只读不需要
	      dynamicApproval: (args) => args && (args.op === 'write' || args.op === 'delete'),
	      execute: (args) => 'ok:' + (args && args.op),
	    })
	    ctx.tools.register({
	      name: 'dyn_plain_tool',
	      description: '无动态审批工具',
	      parameters: { type: 'object', properties: {} },
	      execute: () => 'plain',
	    })
	  },
	}`
	id, err := host.DefineJSCodeFull(code, "js", "动态审批测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}
	defer host.Unload(def.name)

	tool, ok := reg.Get("dyn_demo_tool")
	if !ok {
		t.Fatal("dyn_demo_tool 应已注册")
	}
	if tool.RequiresApproval {
		t.Error("动态审批工具不应设 RequiresApproval（由 DynamicApproval 决定）")
	}
	if tool.DynamicApproval == nil {
		t.Fatal("dynamicApproval 应解析为 Tool.DynamicApproval")
	}
	cases := []struct {
		args string
		want bool
	}{
		{`{"op":"write"}`, true},
		{`{"op":"delete"}`, true},
		{`{"op":"read"}`, false},
		{`{"op":"list"}`, false},
		{`{}`, false},
		{``, false},
	}
	for _, c := range cases {
		got := tool.DynamicApproval(ToolCall{Function: FunctionCall{Name: "dyn_demo_tool", Arguments: c.args}})
		if got != c.want {
			t.Errorf("args=%q 期望 %v 实得 %v", c.args, c.want, got)
		}
	}

	// 未声明 dynamicApproval 的工具 → nil（零行为变化）
	plain, ok := reg.Get("dyn_plain_tool")
	if !ok {
		t.Fatal("dyn_plain_tool 应已注册")
	}
	if plain.DynamicApproval != nil {
		t.Error("未声明 dynamicApproval 的工具应为 nil")
	}

	// execute 分派正常（args 透传不受影响）
	out, err := reg.Execute(context.Background(), "dyn_demo_tool", `{"op":"read"}`)
	if err != nil {
		t.Fatalf("execute 应正常: %v", err)
	}
	if out != "ok:read" {
		t.Errorf("execute 输出异常: %q", out)
	}
}
