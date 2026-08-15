package agent

import (
	"context"
	"strings"
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
	// 插件管理工具集（cordis_*，自举链路保留）
	for _, n := range []string{"cordis_inspect", "cordis_define", "cordis_run", "cordis_stop", "cordis_undefine", "cordis_service_list"} {
		reg.Register(&Tool{Name: n, Handler: noopHandler, SystemTool: true})
	}
	// 工具集管理（toolset_*，agent 自主创建/管理工具集，保留）
	for _, n := range []string{"toolset_build", "toolset_list", "toolset_show", "toolset_export", "toolset_import", "toolset_remove", "toolset_edit"} {
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
	n := ApplyHarnessToolFilter(reg, nil)
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
	if n := ApplyHarnessToolFilter(reg, nil); n != 0 {
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
	first := ApplyHarnessToolFilter(reg, nil)
	second := ApplyHarnessToolFilter(reg, nil)
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
	ApplyHarnessToolFilter(reg, nil)
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

// 被移除的 pair 独有工具名（harness 精简提示词中不应出现）。
var trimmedPromptBannedTools = []string{
	"codegraph", "memory_", "project_info", "history_", "git_", "debug_", "binary_",
	"csv_", "word_", "xlsx", "read_pdf", "lsp_", "skill_", "mcp_", "lua_tool",
	"marketplace", "web_debug", "bug_", "screenshot", "multi_edit", "list_files",
	"run_background", "update_plan", "read_file", "edit_file", "write_file", "run_command",
	"search_content", "search_files", "find_symbol", "go_build", "go_run", "run_test",
	"fix_flex_autoheight", "image_",
}

func TestPromptTrimmedInHarnessMode(t *testing.T) {
	t.Setenv("WB_FULL_TOOLS", "")
	roots := []string{"/test/project"}

	// 自管理/记忆/Lua 三段在 harness 模式下应裁剪为空（引用工具已被移除）
	if s := SelfManagementPrompt(); s != "" {
		t.Errorf("harness 模式 SelfManagementPrompt 应为空，实际: %.80s…", s)
	}
	if s := LuaToolsPrompt(); s != "" {
		t.Errorf("harness 模式 LuaToolsPrompt 应为空，实际: %.80s…", s)
	}
	if s := LongTermMemoryPrompt(); s != "" {
		t.Errorf("harness 模式 LongTermMemoryPrompt 应为空，实际: %.80s…", s)
	}

	// 精简提示词不应引用被移除工具
	p := DefaultSystemPrompt(roots)
	for _, banned := range trimmedPromptBannedTools {
		if strings.Contains(p, banned) {
			t.Errorf("harness 精简提示词仍引用被移除工具名 %q", banned)
		}
	}
	// 协议描述保留
	for _, keep := range []string{"update_tasks", "generate_commit_message", "read", "edit", "write", "bash", "web_search", "cordis_define", "cordis_run"} {
		if !strings.Contains(p, keep) {
			t.Errorf("harness 精简提示词缺失协议/保留工具描述 %q", keep)
		}
	}
}

func TestPromptFullInFullToolsMode(t *testing.T) {
	t.Setenv("WB_FULL_TOOLS", "1")
	roots := []string{"/test/project"}
	if s := SelfManagementPrompt(); s == "" {
		t.Error("WB_FULL_TOOLS=1 时 SelfManagementPrompt 不应为空")
	}
	if s := LuaToolsPrompt(); s == "" {
		t.Error("WB_FULL_TOOLS=1 时 LuaToolsPrompt 不应为空")
	}
	if s := LongTermMemoryPrompt(); s == "" {
		t.Error("WB_FULL_TOOLS=1 时 LongTermMemoryPrompt 不应为空")
	}
	// 完整版提示词长度应显著大于精简版（保留 pair 工具说明）
	full := DefaultSystemPrompt(roots)
	t.Setenv("WB_FULL_TOOLS", "")
	trimmed := DefaultSystemPrompt(roots)
	if len(full) <= len(trimmed) {
		t.Errorf("完整版提示词应比精简版长：full=%d trimmed=%d", len(full), len(trimmed))
	}
	if !strings.Contains(full, "codegraph") {
		t.Error("WB_FULL_TOOLS=1 完整版提示词应包含 codegraph 说明")
	}
}
