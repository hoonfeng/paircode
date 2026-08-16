package agent

// toolset_visibility_test.go — 工具可见性收敛测试（★ 装载 ≠ agent 可用）。
//
// 语义（2026-08-17）：全部插件照常装载（cordis 可见、可管理），但 agent 执行
// 任务时只能看到「工作区工具集声明的工具 + 自举管理工具（SystemTool +
// cordis_*/toolset_*）」。未加入工具集的插件工具禁用（注册保留）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// mkVisibilityReg 构造可见性收敛测试注册表：
// 协议工具（SystemTool/cordis_*/toolset_*）+ 工具集插件工具 + 非工具集插件工具。
func mkVisibilityReg() *Registry {
	reg := NewRegistry()
	// 协议/管理工具（恒对 agent 可见）
	for _, n := range []string{"update_tasks", "update_plan", "tool_stats", "ask_user",
		"generate_commit_message", "task_create", "history_search", "history_list", "history_count"} {
		reg.Register(&Tool{Name: n, Handler: noopHandler, SystemTool: true})
	}
	for _, n := range []string{"cordis_inspect", "cordis_define", "cordis_run", "cordis_stop",
		"cordis_undefine", "cordis_service_list", "cordis_inspect_query"} {
		reg.Register(&Tool{Name: n, Handler: noopHandler})
	}
	for _, n := range []string{"toolset_build", "toolset_list", "toolset_show", "toolset_export",
		"toolset_import", "toolset_remove", "toolset_edit"} {
		reg.Register(&Tool{Name: n, Handler: noopHandler})
	}
	// 内置组工具（builtin:core 声明，应保留启用）
	for _, n := range []string{"read", "write", "edit"} {
		reg.Register(&Tool{Name: n, Handler: noopHandler})
	}
	// 工具集插件工具（tool-foo 注册，应保留启用）
	for _, n := range []string{"codegraph_search", "codegraph_impact", "memory_read", "git_diff"} {
		reg.Register(&Tool{Name: n, Handler: noopHandler})
	}
	// 非工具集插件工具（tool-bar 注册，应隐藏）
	for _, n := range []string{"skill_list", "load_skill", "mcp_list", "image_ocr"} {
		reg.Register(&Tool{Name: n, Handler: noopHandler})
	}
	return reg
}

// mkVisibilityHost 构造带临时工作区工具集的 PluginHost：
// default.json 声明 tool-foo 插件（工具经 pluginTools 模拟注册）+ builtin:core 内置组。
func mkVisibilityHost(t *testing.T, reg *Registry) (*PluginHost, string) {
	t.Helper()
	root := t.TempDir()
	tsDir := filepath.Join(root, ".pair", "toolsets")
	if err := os.MkdirAll(tsDir, 0755); err != nil {
		t.Fatal(err)
	}
	def := Toolset{
		Name: "default",
		Plugins: []ToolsetPlugin{
			{Name: "tool-foo", Purpose: "foo"},
			{Name: "builtin:core", Builtin: "core", Tools: []string{"read", "write", "edit"}},
		},
	}
	data, _ := json.Marshal(def)
	if err := os.WriteFile(filepath.Join(tsDir, "default.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	h := NewPluginHost(reg, nil, root)
	// 模拟插件注册：tool-foo（工具集插件）注册 4 个工具
	h.mu.Lock()
	h.pluginTools["tool-foo"] = []string{"codegraph_search", "codegraph_impact", "memory_read", "git_diff"}
	h.mu.Unlock()
	return h, root
}

func TestApplyToolsetVisibilityFilter(t *testing.T) {
	reg := mkVisibilityReg()
	h, _ := mkVisibilityHost(t, reg)
	n := ApplyToolsetVisibilityFilter(reg, h, h.root)
	// 非工具集工具：skill_list/load_skill/mcp_list/image_ocr 应禁用（4 个）
	if n != 4 {
		t.Errorf("应禁用 4 个非工具集工具，实际 %d", n)
	}
	// 协议工具保持启用
	for _, name := range []string{"update_tasks", "update_plan", "tool_stats", "ask_user",
		"generate_commit_message", "task_create", "history_search",
		"cordis_inspect", "cordis_define", "cordis_run", "toolset_build", "toolset_edit"} {
		if !reg.IsEnabled(name) {
			t.Errorf("协议工具 %s 应保持启用", name)
		}
	}
	// 工具集插件工具启用（pluginTools[tool-foo]）
	for _, name := range []string{"codegraph_search", "codegraph_impact", "memory_read", "git_diff"} {
		if !reg.IsEnabled(name) {
			t.Errorf("工具集插件工具 %s 应启用", name)
		}
	}
	// 内置组 Tools 启用
	for _, name := range []string{"read", "write", "edit"} {
		if !reg.IsEnabled(name) {
			t.Errorf("内置组工具 %s 应启用", name)
		}
	}
	// 非工具集插件工具禁用（保留注册）
	for _, name := range []string{"skill_list", "load_skill", "mcp_list", "image_ocr"} {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("非工具集工具 %s 应保留注册", name)
		}
		if reg.IsEnabled(name) {
			t.Errorf("非工具集工具 %s 应禁用（agent 不可见）", name)
		}
	}
	// Definitions 只导出启用项
	for _, d := range reg.Definitions() {
		if d.Function.Name == "skill_list" || d.Function.Name == "image_ocr" {
			t.Errorf("Definitions 不应含禁用工具 %s", d.Function.Name)
		}
	}
}

func TestApplyPluginToolVisibility(t *testing.T) {
	reg := mkVisibilityReg()
	h, _ := mkVisibilityHost(t, reg)
	h.mu.Lock()
	// tool-bar：非工具集插件，注册 skill_list（非白名单）+ read（builtin 白名单）
	h.pluginTools["tool-bar"] = []string{"skill_list", "read"}
	h.mu.Unlock()
	h.applyPluginToolVisibility("tool-bar")
	if reg.IsEnabled("skill_list") {
		t.Error("非工具集插件工具 skill_list 应被禁用")
	}
	if !reg.IsEnabled("read") {
		t.Error("builtin 白名单工具 read 应保持启用")
	}
	// 工具集插件（tool-foo）不受影响
	h.applyPluginToolVisibility("tool-foo")
	if !reg.IsEnabled("codegraph_search") {
		t.Error("工具集插件工具 codegraph_search 应保持启用")
	}
}

func TestApplyToolsetVisibilityFilter_HarnessModeSkips(t *testing.T) {
	t.Setenv("WB_HARNESS", "1")
	reg := mkVisibilityReg()
	h, _ := mkVisibilityHost(t, reg)
	if n := ApplyToolsetVisibilityFilter(reg, h, h.root); n != 0 {
		t.Errorf("harness 模式不应干预，实际禁用 %d", n)
	}
	if !reg.IsEnabled("skill_list") || !reg.IsEnabled("image_ocr") {
		t.Error("harness 模式全部工具应保持启用")
	}
}

func TestEnsureDefaultWorkspaceToolset(t *testing.T) {
	root := t.TempDir()
	// 空工作区 → 生成基础工具集
	if err := ensureDefaultWorkspaceToolset(root); err != nil {
		t.Fatal(err)
	}
	ts, err := loadToolset(root, toolsetProject, "default")
	if err != nil {
		t.Fatalf("default.json 应已生成: %v", err)
	}
	if len(ts.Plugins) != 1 || ts.Plugins[0].Builtin != "system" {
		t.Fatalf("基础工具集应含 1 个 builtin:system 条目，实际 %+v", ts.Plugins)
	}
	want := []string{"read", "write", "edit", "glob", "grep", "bash", "str_replace_editor", "run_code"}
	got := ts.Plugins[0].Tools
	if len(got) != len(want) {
		t.Fatalf("基础工具应 %v，实际 %v", want, got)
	}
	// 幂等：再次调用不覆盖/不报错
	if err := ensureDefaultWorkspaceToolset(root); err != nil {
		t.Fatal(err)
	}
	// 已有项目工具集（非 builtin）→ 不生成新内容
	root2 := t.TempDir()
	os.MkdirAll(filepath.Join(root2, ".pair", "toolsets"), 0755)
	custom := Toolset{Name: "custom", Plugins: []ToolsetPlugin{{Name: "p1"}}}
	data, _ := json.Marshal(custom)
	os.WriteFile(filepath.Join(root2, ".pair", "toolsets", "custom.json"), data, 0644)
	if err := ensureDefaultWorkspaceToolset(root2); err != nil {
		t.Fatal(err)
	}
	if _, err := loadToolset(root2, toolsetProject, "default"); err == nil {
		t.Error("已有项目工具集时不应生成 default.json")
	}
	// 只有旧版 builtin.json → 迁移（并入 default 后删除旧文件）→ 生成 default.json
	root3 := t.TempDir()
	os.MkdirAll(filepath.Join(root3, ".pair", "toolsets"), 0755)
	os.WriteFile(filepath.Join(root3, ".pair", "toolsets", "builtin.json"), []byte(`{"name":"builtin","plugins":[{"name":"builtin:memory","builtin":"memory","tools":["memory_write"]}]}`), 0644)
	if err := ensureDefaultWorkspaceToolset(root3); err != nil {
		t.Fatal(err)
	}
	ts3, err := loadToolset(root3, toolsetProject, "default")
	if err != nil {
		t.Fatal("仅旧版 builtin.json 时应生成 default.json")
	}
	// 旧文件条目并入 default
	found := false
	for _, p := range ts3.Plugins {
		if p.Builtin == "memory" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("旧版 builtin.json 的 memory 条目应并入 default，实际 %+v", ts3.Plugins)
	}
	if _, err := os.Stat(filepath.Join(root3, ".pair", "toolsets", "builtin.json")); !os.IsNotExist(err) {
		t.Error("迁移后旧版 builtin.json 应被删除")
	}
}
