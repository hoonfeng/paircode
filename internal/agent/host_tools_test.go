package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestPluginTakeoverHostTool 插件接管宿主工具：claimTool 自动存档宿主执行器，
// 插件同名注册后 Registry 工具由插件接管，hostTool 仍可执行宿主 Go 实现。
// ★ 2026-08-16 第三轮：业务工具（read_file 等）已迁移磁盘插件、宿主不再
// 注册——接管对象改为宿主框架协议工具（tool_stats，RegisterHostFrameworkTools
// 注册，无持久化副作用，跨测试安全）。
func TestPluginTakeoverHostTool(t *testing.T) {
	reg := NewRegistry()
	root := t.TempDir()
	host := NewPluginHost(reg, nil, root)
	RegisterHostFrameworkTools(reg, root)

	// 框架 tool_stats 已注册
	if _, ok := reg.Get("tool_stats"); !ok {
		t.Fatal("框架 tool_stats 未注册")
	}

	// JS 插件接管 tool_stats（execute 调 ctx.hostTool.exec）
	code := `
return {
  name: 'takeover-test',
  apply(ctx) {
    ctx.tools.register({
      name: 'tool_stats',
      description: '接管后的 tool_stats',
      parameters: { type: 'object', properties: { min_calls: { type: 'integer' } } },
      execute: (args) => {
        // 演示插件编排：调用宿主执行器并附加标记
        const out = ctx.hostTool.exec('tool_stats', args)
        return '[插件接管]' + out
      },
    })
  },
}
`
	id, err := host.DefineJSCodeFull(code, "js", "接管测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if def == nil {
		t.Fatalf("定义 %s 不存在", id)
	}
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}

	// ① 宿主执行器已存档（被接管的 tool_stats）
	if _, ok := HostToolMeta("tool_stats"); !ok {
		t.Fatal("tool_stats 宿主执行器未存档")
	}
	// 未被接管的 task_create 不应在 hostExecutors（会话级工具，不在宿主框架注册）
	if _, ok := HostToolMeta("task_create"); ok {
		t.Fatal("task_create 未被接管不应存档")
	}
	if _, ok := reg.Get("task_create"); ok {
		t.Fatal("task_create 不应在宿主框架 Registry（会话级注册）")
	}

	// ② Registry 中 tool_stats 已被插件接管（Handler 为 JS 桥）
	tool, ok := reg.Get("tool_stats")
	if !ok {
		t.Fatal("tool_stats 不存在")
	}
	if !strings.Contains(tool.Description, "接管后的 tool_stats") {
		t.Fatalf("tool_stats 未被插件接管，描述=%q", tool.Description)
	}

	// ③ hostTool 可执行宿主实现（无副作用：统计未启用时返回提示而非崩溃）
	if _, err := ExecuteHostTool("tool_stats", map[string]any{}); err != nil {
		t.Fatalf("ExecuteHostTool 失败: %v", err)
	}

	// ④ 插件工具 execute 走 JS 桥（含插件编排标记）
	argsJSON, _ := json.Marshal(map[string]any{})
	res, err := reg.Execute(nil, "tool_stats", string(argsJSON))
	if err != nil {
		t.Fatalf("插件 tool_stats 执行失败: %v", err)
	}
	if !strings.Contains(res, "[插件接管]") {
		t.Fatalf("插件 tool_stats 未走 JS 编排，结果=%q", res)
	}
}

// TestHostToolUnknown 未存档工具执行报错。
func TestHostToolUnknown(t *testing.T) {
	if _, err := ExecuteHostTool("not_exist_tool_xyz", nil); err == nil {
		t.Fatal("未存档工具应报错")
	}
}
