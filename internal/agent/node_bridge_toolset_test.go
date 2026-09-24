package agent

import (
	"context"
	"strings"
	"testing"
)

// TestNodeBridgePluginToolsetAdd Node 桥插件的「工具集 → 添加」链路（2026-09-19 修复回归）。
//
// 背景：桥装载的插件（市场 npm 安装 / 磁盘桥轨插件交接）在宿主里的工具归属名带
// "node-bridge:" 前缀（pluginTools 键 = "node-bridge:<name>"），而插件记录
// （/api/plugins）、工具集条目、前端「工具集 → 添加」提交用的都是**不带前缀**的 <name>。
// 修复前：add_plugin 报「宿主未定义插件 <name>」；且可见性白名单因
// ph.State(<name>) != PluginRunning 整条跳过 → 即使加进工具集，工具对 agent 仍不可见。
// 修复：PluginToolsByName / HasPluginByName 做归属键回退，工具集编辑与白名单统一走它。
func TestNodeBridgePluginToolsetAdd(t *testing.T) {
	project := mkToolsetGoProject(t)
	SetGlobalToolsetDirForTest(t.TempDir())
	t.Cleanup(func() { SetGlobalToolsetDirForTest(testGlobalToolsetDir) })

	reg := NewRegistry()
	host := NewPluginHost(reg, nil, project)

	// 模拟桥装载结果：工具归属 node-bridge:tool-voice（桥 ready 后 Go 侧注册的真实形态）
	bridgeTools := []string{"voice_import", "voice_analyze", "voice_add", "voice_edit", "voice_render", "voice_verify"}
	host.pluginTools["node-bridge:tool-voice"] = append([]string(nil), bridgeTools...)
	for _, tn := range bridgeTools {
		name := tn
		reg.Register(&Tool{
			Name:        name,
			Description: "桥插件探针工具 " + name,
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
			Handler:     func(context.Context, map[string]any) (string, error) { return "ok", nil },
			Enabled:     true,
		})
		host.toolOwner[name] = "node-bridge:tool-voice"
	}

	// 初始工具集（空插件集）
	ts := &Toolset{Name: "bridge-probe", Description: "桥插件工具集集成验证", Plugins: []ToolsetPlugin{}}
	if err := saveToolset(project, toolsetProject, ts); err != nil {
		t.Fatalf("saveToolset: %v", err)
	}
	if err := installToolset(host, ts); err != nil {
		t.Fatalf("installToolset: %v", err)
	}

	// ① add_plugin：插件名不带前缀（= /api/plugins 记录名 / 前端提交形态）
	// 前提断言（等价反向验证）：桥插件的 pluginTools 键只在带前缀处；
	// 直查不带前缀为空 ⇒ 修复前的 `PluginToolsByPlugin()[pn]` 必然查不到、
	// `State(pn) != PluginRunning` 必然为真（本用例因此能捕获旧行为）。
	if got := host.PluginToolsByPlugin()["tool-voice"]; len(got) != 0 {
		t.Fatalf("前提不成立：不带前缀的键不应存在（实际 %v）", got)
	}
	if host.State("tool-voice") == PluginRunning {
		t.Fatal("前提不成立：桥插件不应有 JSDef running 状态")
	}
	if got := host.PluginToolsByName("tool-voice"); len(got) != len(bridgeTools) {
		t.Fatalf("回退取工具应命中 %d 个，实际 %v", len(bridgeTools), got)
	}
	msg, err := toolsetEdit(host, project, map[string]any{
		"name": "bridge-probe", "action": "add_plugin", "plugin_name": "tool-voice",
	})
	if err != nil {
		t.Fatalf("add_plugin 桥插件应成功（修复前报「宿主未定义插件」）: %v", err)
	}
	if !strings.Contains(msg, "tool-voice") {
		t.Fatalf("返回消息应含插件名: %s", msg)
	}

	// ② 条目落盘：插件名 + 工具清单（Builtin=bridge ⇒ 装载=启用工具，不重复 JS 装载）
	ts2, err := loadToolset(project, toolsetProject, "bridge-probe")
	if err != nil {
		t.Fatalf("loadToolset: %v", err)
	}
	if len(ts2.Plugins) != 1 || ts2.Plugins[0].Name != "tool-voice" {
		t.Fatalf("工具集条目异常: %+v", ts2.Plugins)
	}
	if got := len(ts2.Plugins[0].Tools); got != len(bridgeTools) {
		t.Fatalf("应登记 %d 个工具，实际 %d（%v）", len(bridgeTools), got, ts2.Plugins[0].Tools)
	}

	// ③ 工具对 agent 可见（Enabled=true）
	for _, tn := range bridgeTools {
		tool, ok := reg.Get(tn)
		if !ok {
			t.Fatalf("工具 %s 未注册", tn)
		}
		if !tool.Enabled {
			t.Fatalf("工具 %s 应启用（加入工具集后对 agent 可见）", tn)
		}
	}

	// ④ 可见性白名单包含桥插件工具（修复前被 ph.State != PluginRunning 整条跳过）
	keep := host.workspaceToolsetVisibleTools()
	for _, tn := range bridgeTools {
		if !keep[tn] {
			t.Fatalf("工具 %s 应进 agent 可见白名单", tn)
		}
	}
}
