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
	for _, n := range []string{"update_tasks", "tool_stats", "ask_user",
		"task_create", "history_search", "history_list", "history_count"} {
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
	for _, n := range []string{"skill_list", "load_skill", "mcp_list", "submit_image"} {
		reg.Register(&Tool{Name: n, Handler: noopHandler})
	}
	return reg
}

// mkVisibilityHost 构造带临时全局工具集的 PluginHost：
// default.json 声明 tool-foo 插件（工具经 pluginTools 模拟注册）+ builtin:core 内置组。
// ★ 2026-09-04 工具集全局化：全局工具集目录重定向到临时目录（测试隔离），
//   NewPluginHost 仅保留 root 用于其他上下文（工作区隔离语义已由全局通用集合取代）。
func mkVisibilityHost(t *testing.T, reg *Registry) (*PluginHost, string) {
	t.Helper()
	root := t.TempDir()
	SetGlobalToolsetDirForTest(t.TempDir())
	t.Cleanup(func() { SetGlobalToolsetDirForTest(testGlobalToolsetDir) })
	tsDir := globalToolsetDir()
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
	// 模拟插件注册：tool-foo（工具集插件）注册 4 个工具（装载后 running——
	// ★ 2026-08-2x：白名单仅含 running 插件的工具，未启用插件不暴露）
	h.mu.Lock()
	h.pluginTools["tool-foo"] = []string{"codegraph_search", "codegraph_impact", "memory_read", "git_diff"}
	h.states["tool-foo"] = PluginRunning
	h.mu.Unlock()
	return h, root
}

func TestApplyToolsetVisibilityFilter(t *testing.T) {
	reg := mkVisibilityReg()
	h, _ := mkVisibilityHost(t, reg)
	n := ApplyToolsetVisibilityFilter(reg, h, h.root)
	// 非工具集工具：skill_list/load_skill/mcp_list/submit_image 应禁用（4 个）
	if n != 4 {
		t.Errorf("应禁用 4 个非工具集工具，实际 %d", n)
	}
	// 协议工具保持启用
	for _, name := range []string{"update_tasks", "tool_stats", "ask_user",
		"task_create", "history_search",
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
	for _, name := range []string{"skill_list", "load_skill", "mcp_list", "submit_image"} {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("非工具集工具 %s 应保留注册", name)
		}
		if reg.IsEnabled(name) {
			t.Errorf("非工具集工具 %s 应禁用（agent 不可见）", name)
		}
	}
	// Definitions 只导出启用项
	for _, d := range reg.Definitions() {
		if d.Function.Name == "skill_list" || d.Function.Name == "submit_image" {
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
	if !reg.IsEnabled("skill_list") || !reg.IsEnabled("submit_image") {
		t.Error("harness 模式全部工具应保持启用")
	}
}

func TestEnsureDefaultWorkspaceToolset(t *testing.T) {
	SetGlobalToolsetDirForTest(t.TempDir())
	t.Cleanup(func() { SetGlobalToolsetDirForTest(testGlobalToolsetDir) })
	root := t.TempDir()
	// 空工具集目录 → 播种预置模式（default 应生成，含内置组条目）
	if err := ensureDefaultWorkspaceToolset(nil, root); err != nil {
		t.Fatal(err)
	}
	ts, err := loadToolset("", toolsetProject, "default")
	if err != nil {
		t.Fatalf("default.json 应已生成: %v", err)
	}
	if len(ts.Plugins) == 0 {
		t.Fatalf("预置 default 应含条目，实际 %+v", ts.Plugins)
	}
	// 幂等：再次调用不覆盖/不报错
	if err := ensureDefaultWorkspaceToolset(nil, root); err != nil {
		t.Fatal(err)
	}
	// 已有预置 default → 不生成新内容（不再有工作区隔离语义）
	root2 := t.TempDir()
	if err := ensureDefaultWorkspaceToolset(nil, root2); err != nil {
		t.Fatal(err)
	}
	// ★ 2026-09-04：工具集已全局化——root2 与 root 共用同一全局目录，
	//   default 已存在（播种幂等），再次调用不重复生成。
	if ts2, err := loadToolset("", toolsetProject, "default"); err != nil {
		t.Fatal("default 应全局唯一存在")
	} else if len(ts2.Plugins) != len(ts.Plugins) {
		t.Errorf("default 幂等不变量被破坏：%d → %d", len(ts.Plugins), len(ts2.Plugins))
	}
}

// TestApplyWorkspaceToolsetWhitelist 白名单模型：agent 只暴露「工作区工具集声明 +
// 框架本身提供的工具」——无工具集先自动创建基础工具集；有工具集按声明收敛。
// ★ 2026-08-17：有配置只暴露配置里的（+框架自举工具）；未声明的插件/内置包工具禁用。
func TestApplyWorkspaceToolsetWhitelist(t *testing.T) {
	SetGlobalToolsetDirForTest(t.TempDir())
	t.Cleanup(func() { SetGlobalToolsetDirForTest(testGlobalToolsetDir) })
	ph, root := mkBuiltinHost(t)
	reg := ph.Context().Tools
	// 无工具集：白名单应用前应自动创建基础工具集（播种预置 default）
	ApplyWorkspaceToolsetWhitelist(ph, reg, root)
	if !hasWorkspaceToolsets() {
		t.Fatal("无工具集时应自动创建基础工具集")
	}
	// 框架自举工具恒可用：SystemTool（update_tasks）、cordis_*、toolset_*
	for _, tn := range []string{"update_tasks", "cordis_define", "cordis_run", "toolset_edit", "toolset_build"} {
		if !reg.IsEnabled(tn) {
			t.Errorf("框架自举工具 %s 应可用（白名单兜底）", tn)
		}
	}
	// 极简核心可用（默认工具集 system 条目声明）
	for _, tn := range []string{"read", "write", "edit", "glob", "grep", "bash", "run_code"} {
		if !reg.IsEnabled(tn) {
			t.Errorf("核心工具 %s 应可用（基础工具集声明）", tn)
		}
	}
	// 未声明的内置包工具禁用（codegraph_search 等非框架宿主工具）
	if reg.IsEnabled("codegraph_search") {
		t.Error("未声明的 codegraph_search 应对 agent 隐藏")
	}
	// 加入 codegraph 组 → 其工具可用
	if _, err := SetBuiltinGroupEnabled(ph, root, "codegraph", true); err != nil {
		t.Fatalf("加入 codegraph 失败: %v", err)
	}
	ApplyWorkspaceToolsetWhitelist(ph, reg, root)
	if !reg.IsEnabled("codegraph_search") {
		t.Error("加入 codegraph 后 codegraph_search 应可用")
	}
}

// TestApplyConvToolsetWhitelist 会话级工具集（通用集合）白名单：
// 会话元数据选择集合 → agent 工具面按所选集合收敛（不是全局并集）；
// 未选择（空）→ default 集合；会话不存在/找不到 → 回落 default。
// ★ 2026-09-04 工具集全局化：任何 scope 都读全局通用集合目录。
func TestApplyConvToolsetWhitelist(t *testing.T) {
	SetGlobalToolsetDirForTest(t.TempDir())
	t.Cleanup(func() { SetGlobalToolsetDirForTest(testGlobalToolsetDir) })
	ph, root := mkBuiltinHost(t)
	reg := ph.Context().Tools
	tsDir := globalToolsetDir()
	// 造两个集合：default（core 组 + tool-foo 插件工具）与 dev（core + codegraph）
	mkTs := func(name string, plugins []ToolsetPlugin) {
		ts := Toolset{Name: name, Plugins: plugins, BuiltinsInited: true}
		data, _ := json.Marshal(ts)
		if err := os.WriteFile(filepath.Join(tsDir, name+".json"), data, 0644); err != nil {
			t.Fatalf("写工具集 %s: %v", name, err)
		}
	}
	mkTs("default", []ToolsetPlugin{
		{Name: "builtin:core", Builtin: "core", Tools: []string{"read", "write", "edit"}},
		{Name: "tool-foo", Purpose: "foo"},
	})
	mkTs("dev", []ToolsetPlugin{
		{Name: "builtin:core", Builtin: "core", Tools: []string{"read", "write", "edit"}},
		{Name: "builtin:codegraph", Builtin: "codegraph", Tools: []string{"codegraph_search", "codegraph_impact"}},
	})
	// 模拟插件工具注册（tool-foo：工具集插件；tool-bar：未声明插件）
	ph.mu.Lock()
	ph.pluginTools["tool-foo"] = []string{"memory_read", "git_diff"}
	ph.pluginTools["tool-bar"] = []string{"skill_list", "load_skill"}
	ph.states["tool-foo"] = PluginRunning
	ph.states["tool-bar"] = PluginRunning
	ph.mu.Unlock()
	for _, tn := range []string{"memory_read", "git_diff", "skill_list", "load_skill", "codegraph_search", "codegraph_impact"} {
		reg.Register(&Tool{Name: tn, Handler: noopHandler})
	}

	// ① 会话未设置（空）→ default 集合：tool-foo 工具可见、codegraph 隐藏、skill_list 隐藏
	ApplyConvToolsetWhitelist(ph, reg, "", root)
	if !reg.IsEnabled("memory_read") || !reg.IsEnabled("git_diff") {
		t.Error("default 集合：tool-foo 工具（memory_read/git_diff）应启用")
	}
	if reg.IsEnabled("codegraph_search") || reg.IsEnabled("codegraph_impact") {
		t.Error("default 集合：codegraph 工具应隐藏")
	}
	if reg.IsEnabled("skill_list") || reg.IsEnabled("load_skill") {
		t.Error("default 集合：未声明插件工具 skill_list/load_skill 应隐藏")
	}
	// ② 会话选择 dev → dev 集合：codegraph 可见、tool-foo 隐藏
	ApplyToolsetWhitelistByName(ph, reg, "dev")
	if !reg.IsEnabled("codegraph_search") || !reg.IsEnabled("codegraph_impact") {
		t.Error("dev 集合：codegraph 工具应启用")
	}
	if reg.IsEnabled("memory_read") || reg.IsEnabled("git_diff") {
		t.Error("dev 集合：tool-foo 工具应隐藏（仅所选集合声明）")
	}
	// ③ 框架自举工具恒可用（两个集合均如此）
	for _, tn := range []string{"update_tasks", "cordis_define", "toolset_edit"} {
		if !reg.IsEnabled(tn) {
			t.Errorf("框架自举工具 %s 应始终可用", tn)
		}
	}
}
