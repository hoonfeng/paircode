package agent

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// mkBuiltinHost 构造测试用 PluginHost（内置插件 + cordis 工具 + 工具集工具装配）。
func mkBuiltinHost(t *testing.T) (*PluginHost, string) {
	t.Helper()
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	RegisterHarnessTools(reg, root)
	RegisterHostFrameworkTools(reg, root) // ★ 宿主框架自举工具（update_tasks/update_plan 等）
	ph := NewPluginHost(reg, nil, root)
	RegisterCordisTools(reg, ph, root)
	RegisterToolsetTools(reg, root, ph)
	SetGlobalPluginHost(ph)
	t.Cleanup(func() { SetGlobalPluginHost(nil) })
	return ph, root
}

func TestBuiltinPluginToolGroups(t *testing.T) {
	groups := builtinPluginToolGroups()
	// core 组：文件工具
	core := groups["core"]
	if len(core) == 0 {
		t.Fatal("core 组不应为空")
	}
	found := false
	for _, n := range core {
		if n == "read_file" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("core 组应含 read_file，实际 %v", core)
	}
	// codegraph 组存在（被过滤工具的载体）
	if len(groups["codegraph"]) == 0 {
		t.Error("codegraph 组不应为空")
	}
	// 所有内置组互不重叠（工具唯一归属）
	seen := map[string]bool{}
	for _, list := range groups {
		for _, n := range list {
			if seen[n] {
				t.Errorf("工具 %s 重复归属多个内置组", n)
			}
			seen[n] = true
		}
	}
}

func TestBuiltinGroupsOf(t *testing.T) {
	ph, _ := mkBuiltinHost(t)
	reg := ph.Context().Tools
	// 全量模式：全部工具启用（harness 过滤默认开启，先全量保证分组完整）
	groups := BuiltinGroupsOf(reg, ph)
	if len(groups) == 0 {
		t.Fatal("内置分组不应为空")
	}
	names := map[string]bool{}
	for _, g := range groups {
		names[g.Name] = true
		if len(g.Tools) == 0 {
			t.Errorf("分组 %s 无工具", g.Name)
		}
	}
	// 核心分组存在（★ 2026-08-16 第三轮：内置 20 组迁移磁盘插件，不再展示——
	// 只保留管理分组 plugin-mgmt/toolset-mgmt/system + 已加入的内置组）
	for _, want := range []string{"plugin-mgmt", "toolset-mgmt", "system"} {
		if !names[want] {
			t.Errorf("缺少分组 %s（实际 %v）", want, keysOf(names))
		}
	}
	// 已迁移磁盘插件的内置组不应再静态派生（避免与插件面板重复展示）
	for _, gone := range []string{"core", "git", "codegraph", "fs-search", "web"} {
		if names[gone] {
			t.Errorf("内置组 %s 不应再展示（已迁移磁盘插件）", gone)
		}
	}
	// plugin-mgmt 含 cordis_define
	for _, g := range groups {
		if g.Name == "plugin-mgmt" {
			if !containsToolName(g.Tools, "cordis_define") {
				t.Errorf("plugin-mgmt 应含 cordis_define，实际 %v", toolNamesOf(g))
			}
		}
	}
	// system 含 harness 别名 read
	for _, g := range groups {
		if g.Name == "system" {
			if !containsToolName(g.Tools, "read") {
				t.Errorf("system 应含 harness 别名 read，实际 %v", toolNamesOf(g))
			}
		}
	}
	// harness 模式：被过滤组（codegraph）工具应 Enabled=false
	t.Setenv("WB_HARNESS", "1")
	ApplyHarnessToolFilter(reg, nil)
	groups = BuiltinGroupsOf(reg, ph)
	for _, g := range groups {
		if g.Name == "codegraph" {
			for _, ti := range g.Tools {
				if ti.Enabled {
					t.Errorf("harness 模式下 codegraph 组工具 %s 应禁用", ti.Name)
				}
			}
		}
	}
}

func TestSetBuiltinGroupEnabled_JoinAndLeave(t *testing.T) {
	ph, root := mkBuiltinHost(t)
	reg := ph.Context().Tools
	t.Setenv("WB_HARNESS", "1")
	ApplyHarnessToolFilter(reg, nil)

	// 加入 codegraph 组 → 组内工具全部启用 + 固化工作区工具集（default.json）
	msg, err := SetBuiltinGroupEnabled(ph, root, "codegraph", true)
	if err != nil {
		t.Fatalf("加入 codegraph 失败: %v", err)
	}
	if !strings.Contains(msg, "codegraph") {
		t.Errorf("返回信息应含组名，实际 %s", msg)
	}
	if !reg.IsEnabled("codegraph_search") {
		t.Error("加入后 codegraph_search 应启用（agent 可见）")
	}
	// 工作区主工具集固化（内置组条目并入 default.json，无独立 builtin.json）
	ts, err := loadToolset(root, toolsetProject, "default")
	if err != nil {
		t.Fatalf("工作区工具集未固化: %v", err)
	}
	found := false
	for _, p := range ts.Plugins {
		if p.Builtin == "codegraph" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("default.json 应有 codegraph 内置条目，实际 %+v", ts.Plugins)
	}
	if _, err := os.Stat(toolsetPath(root, toolsetProject, builtinToolsetName)); !os.IsNotExist(err) {
		t.Error("不应存在独立 builtin.json（已合并到工作区工具集）")
	}

	// 移出 → 恢复默认（harness 模式下禁用）；default.json 仍保留（工作区主工具集永存）
	msg, err = SetBuiltinGroupEnabled(ph, root, "codegraph", false)
	if err != nil {
		t.Fatalf("移出 codegraph 失败: %v", err)
	}
	if reg.IsEnabled("codegraph_search") {
		t.Error("移出后 codegraph_search 应恢复默认（禁用）")
	}
	if _, err := os.Stat(toolsetPath(root, toolsetProject, "default")); err != nil {
		t.Error("default.json 应保留（工作区主工具集永存，内置组移出后回到基础工具集）")
	}
}

func TestEnableAllBuiltin(t *testing.T) {
	ph, root := mkBuiltinHost(t)
	reg := ph.Context().Tools
	t.Setenv("WB_HARNESS", "1")
	ApplyHarnessToolFilter(reg, nil)
	if reg.IsEnabled("codegraph_search") {
		t.Fatal("前置：codegraph_search 应被过滤禁用")
	}
	msg, err := EnableAllBuiltin(ph, root)
	if err != nil {
		t.Fatalf("强制全部失败: %v", err)
	}
	if !strings.Contains(msg, "强制加入全部") {
		t.Errorf("返回信息异常: %s", msg)
	}
	// 强制全部后全部内置工具启用
	groups := BuiltinGroupsOf(reg, ph)
	for _, g := range groups {
		for _, ti := range g.Tools {
			if !ti.Enabled {
				t.Errorf("强制全部后工具 %s（组 %s）应启用", ti.Name, g.Name)
			}
		}
	}
	// 固化到工作区主工具集（default.json；无独立 builtin.json）
	if _, err := os.Stat(toolsetPath(root, toolsetProject, "default")); err != nil {
		t.Errorf("default.json 应存在: %v", err)
	}
	if _, err := os.Stat(toolsetPath(root, toolsetProject, builtinToolsetName)); !os.IsNotExist(err) {
		t.Error("不应存在独立 builtin.json（已合并到工作区工具集）")
	}
}

func TestApplyToolsetBuiltinState(t *testing.T) {
	ph, root := mkBuiltinHost(t)
	reg := ph.Context().Tools
	t.Setenv("WB_HARNESS", "1")
	ApplyHarnessToolFilter(reg, nil)
	if _, err := SetBuiltinGroupEnabled(ph, root, "memory", true); err != nil {
		t.Fatalf("加入 memory 失败: %v", err)
	}

	// 模拟会话级独立注册表（重新注册 + 过滤 + 应用内置状态）
	reg2 := NewRegistry()
	RegisterDefaultTools(reg2, root)
	ApplyHarnessToolFilter(reg2, nil)
	if reg2.IsEnabled("memory_write") {
		t.Fatal("前置：会话注册表 memory_write 应被过滤")
	}
	ApplyToolsetBuiltinState(reg2, root)
	if !reg2.IsEnabled("memory_write") {
		t.Error("ApplyToolsetBuiltinState 后 memory_write 应启用（工作区 builtin 状态生效）")
	}
	if reg2.IsEnabled("codegraph_search") {
		t.Error("未加入的 codegraph_search 应保持禁用")
	}
}

// ─── 辅助 ─────────────────────────────────────────────────

func keysOf(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func toolNamesOf(g BuiltinGroupInfo) []string {
	var out []string
	for _, ti := range g.Tools {
		out = append(out, ti.Name)
	}
	return out
}

func containsToolName(list []BuiltinToolInfo, s string) bool {
	for _, ti := range list {
		if ti.Name == s {
			return true
		}
	}
	return false
}

var _ = filepath.Join
