package agent

import (
	"context"
	"testing"
)

func noopHandler(_ context.Context, _ map[string]any) (string, error) { return "", nil }

// mkHarnessReg 构造含 harness 工具 + pair 独有工具的注册表（供过滤测试）。
func mkHarnessReg() *Registry {
	reg := NewRegistry()
	// harness 工具集（含别名与原生 web）
	for _, n := range []string{"read", "write", "edit", "glob", "grep", "str_replace_editor", "bash", "web_search", "web_fetch", "run_code"} {
		reg.Register(&Tool{Name: n, Handler: noopHandler})
	}
	// 对话协议基础设施
	for _, n := range []string{"update_tasks", "ask_user", "generate_commit_message"} {
		reg.Register(&Tool{Name: n, Handler: noopHandler, SystemTool: true})
	}
	// pair 独有工具（应被移除）
	for _, n := range []string{"read_file", "write_file", "edit_file", "multi_edit", "list_files", "run_command",
		"codegraph_search", "memory_read", "project_info_write", "git_diff", "debug_inject_log",
		"binary_hash", "csv_read", "web_debug", "go_build", "fix_flex_autoheight"} {
		reg.Register(&Tool{Name: n, Handler: noopHandler})
	}
	return reg
}

func TestApplyHarnessToolFilter_RemovesPairTools(t *testing.T) {
	t.Setenv("WB_FULL_TOOLS", "")
	reg := mkHarnessReg()
	before := len(reg.AllToolMeta())
	n := ApplyHarnessToolFilter(reg)
	if n != before-len(HarnessAlignedToolNames) {
		t.Errorf("应移除 %d 个工具，实际移除 %d（注册 %d / 保留 %d）", before-len(HarnessAlignedToolNames), n, before, len(HarnessAlignedToolNames))
	}
	// harness 工具保留
	for name := range HarnessAlignedToolNames {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("保留清单工具 %s 被误删", name)
		}
	}
	// pair 独有工具全部移除
	for _, name := range []string{"read_file", "write_file", "codegraph_search", "memory_read",
		"project_info_write", "git_diff", "debug_inject_log", "binary_hash", "csv_read", "web_debug",
		"go_build", "fix_flex_autoheight"} {
		if _, ok := reg.Get(name); ok {
			t.Errorf("pair 独有工具 %s 未被移除", name)
		}
	}
	if got := len(reg.AllToolMeta()); got != len(HarnessAlignedToolNames) {
		t.Errorf("过滤后应剩 %d 个工具，实际 %d", len(HarnessAlignedToolNames), got)
	}
}

func TestApplyHarnessToolFilter_FullToolsKeepsAll(t *testing.T) {
	t.Setenv("WB_FULL_TOOLS", "1")
	reg := mkHarnessReg()
	before := len(reg.AllToolMeta())
	if n := ApplyHarnessToolFilter(reg); n != 0 {
		t.Errorf("WB_FULL_TOOLS=1 时不应移除任何工具，实际移除 %d", n)
	}
	if got := len(reg.AllToolMeta()); got != before {
		t.Errorf("WB_FULL_TOOLS=1 时工具数不应变化，before=%d after=%d", before, got)
	}
	if _, ok := reg.Get("codegraph_search"); !ok {
		t.Error("WB_FULL_TOOLS=1 时 codegraph_search 应保留")
	}
}

func TestApplyHarnessToolFilter_Idempotent(t *testing.T) {
	t.Setenv("WB_FULL_TOOLS", "")
	reg := mkHarnessReg()
	first := ApplyHarnessToolFilter(reg)
	second := ApplyHarnessToolFilter(reg)
	if second != 0 {
		t.Errorf("第二次过滤应移除 0 个（幂等），实际 %d（第一次 %d）", second, first)
	}
	if got := len(reg.AllToolMeta()); got != len(HarnessAlignedToolNames) {
		t.Errorf("幂等后应剩 %d 个工具，实际 %d", len(HarnessAlignedToolNames), got)
	}
}

func TestApplyHarnessToolFilter_KeepsHooks(t *testing.T) {
	t.Setenv("WB_FULL_TOOLS", "")
	reg := mkHarnessReg()
	called := false
	reg.BeforeTool = func(_ context.Context, _ string, _ map[string]any) (bool, string, error) {
		called = true
		return true, "", nil
	}
	ApplyHarnessToolFilter(reg)
	if reg.BeforeTool == nil {
		t.Fatal("过滤后 BeforeTool 钩子应保留")
	}
	if _, err := reg.Execute(context.Background(), "read", "{}"); err != nil {
		t.Fatalf("过滤后 read 应可执行：%v", err)
	}
	if !called {
		t.Error("过滤后 BeforeTool 钩子应被触发")
	}
}

func TestHarnessOnlyTools_Default(t *testing.T) {
	t.Setenv("WB_FULL_TOOLS", "")
	if !HarnessOnlyTools() {
		t.Error("默认（未设 WB_FULL_TOOLS）应处于 harness 对齐模式")
	}
	t.Setenv("WB_FULL_TOOLS", "1")
	if HarnessOnlyTools() {
		t.Error("WB_FULL_TOOLS=1 应关闭 harness 对齐模式")
	}
}
