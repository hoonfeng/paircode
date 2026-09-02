package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkTs 构造含指定条目的 default 工具集并落盘（★ 全局化：写入重定向的全局工具集目录）。
func mkTs(t *testing.T, root string, plugins []ToolsetPlugin) {
	t.Helper()
	ts := &Toolset{Name: "default", Project: filepath.Base(root), Plugins: plugins}
	if err := saveToolset(root, toolsetProject, ts); err != nil {
		t.Fatalf("saveToolset: %v", err)
	}
}

// TestVisibleToolsSkipUnloadedPlugin 白名单过滤：stopped/未装载插件的工具不进白名单，
// builtin 条目未注册工具不进白名单。
func TestVisibleToolsSkipUnloadedPlugin(t *testing.T) {
	ph, root := mkBuiltinHost(t)

	// 模拟两个插件：plug-a running、plug-b stopped
	ph.mu.Lock()
	ph.pluginTools["plug-a"] = []string{"read", "tool_a_only"}
	ph.pluginTools["plug-b"] = []string{"tool_b_only"}
	ph.states["plug-a"] = PluginRunning
	ph.states["plug-b"] = PluginStopped
	ph.mu.Unlock()

	mkTs(t, root, []ToolsetPlugin{
		{Name: "plug-a", Code: "x", Purpose: "running 插件"},
		{Name: "plug-b", Code: "x", Purpose: "stopped 插件"},
		{Name: "builtin:test", Builtin: "test", Tools: []string{"read", "fake_not_registered"}},
	})

	keep := ph.workspaceToolsetVisibleTools()
	// running 插件的工具在白名单
	if !keep["read"] {
		t.Errorf("plug-a（running）的工具 read 应在白名单")
	}
	if !keep["tool_a_only"] {
		t.Errorf("plug-a（running）的工具 tool_a_only 应在白名单")
	}
	// stopped 插件的工具不在白名单
	if keep["tool_b_only"] {
		t.Errorf("plug-b（stopped）的工具 tool_b_only 不应在白名单")
	}
	// builtin 条目未注册工具不在白名单
	if keep["fake_not_registered"] {
		t.Errorf("builtin 条目未注册工具 fake_not_registered 不应在白名单")
	}
}

// TestPruneUnavailableFromToolsets 实装后清理：未装载插件条目移除、未注册工具移除、
// 整组未暴露（无工具/全 disabled）条目移除，变更落盘保存。
func TestPruneUnavailableFromToolsets(t *testing.T) {
	ph, root := mkBuiltinHost(t)

	// plug-a running（保留）；plug-b stopped（应移除）
	ph.mu.Lock()
	ph.pluginTools["plug-a"] = []string{"read"}
	ph.states["plug-a"] = PluginRunning
	ph.states["plug-b"] = PluginStopped
	ph.mu.Unlock()

	mkTs(t, root, []ToolsetPlugin{
		// ① JS 条目：running 插件 → 保留
		{Name: "plug-a", Code: "x", Purpose: "running"},
		// ② JS 条目：未装载插件 → 整条移除
		{Name: "plug-b", Code: "x", Purpose: "stopped"},
		// ③ builtin 条目：含未注册工具 → 该工具移除，注册工具保留
		{Name: "builtin:g1", Builtin: "g1", Tools: []string{"read", "fake_missing"}},
		// ④ builtin 条目：工具全部被 DisabledTools 摘除（面板未启用）→ 整条移除
		{Name: "builtin:g2", Builtin: "g2", Tools: []string{"read"}, DisabledTools: []string{"read"}},
		// ⑤ builtin 条目：无工具声明 → 整条移除
		{Name: "builtin:g3", Builtin: "g3", Tools: []string{}},
		// ⑥ builtin 条目：正常（保留）
		{Name: "builtin:g4", Builtin: "g4", Tools: []string{"read"}},
	})

	cleaned := pruneUnavailableFromToolsets(ph, root)
	if cleaned != 1 {
		t.Fatalf("期望 1 个工具集被清理，实际 %d", cleaned)
	}

	// 落盘校验（★ 全局化：读全局工具集目录）
	data, err := os.ReadFile(toolsetPath("", toolsetProject, "default"))
	if err != nil {
		t.Fatalf("读 default.json: %v", err)
	}
	raw := string(data)
	for _, want := range []string{"plug-a", "builtin:g1", "builtin:g4"} {
		if !strings.Contains(raw, want) {
			t.Errorf("工具集应保留条目 %s（实际内容:\n%s）", want, raw)
		}
	}
	for _, gone := range []string{"plug-b", "builtin:g2", "builtin:g3", "fake_missing"} {
		if strings.Contains(raw, gone) {
			t.Errorf("工具集不应再含 %s（实际内容:\n%s）", gone, raw)
		}
	}
}
