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
	t.Setenv("WB_HARNESS", "1")
	reg := mkHarnessReg()
	before := len(reg.AllToolMeta())
	n := ApplyHarnessToolFilter(reg, nil)
	if n != before-len(HarnessAlignedToolNames) {
		t.Errorf("应禁用 %d 个工具，实际禁用 %d（注册 %d / 保留 %d）", before-len(HarnessAlignedToolNames), n, before, len(HarnessAlignedToolNames))
	}
	// harness 工具保留且启用
	for name := range HarnessAlignedToolNames {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("保留清单工具 %s 被误删", name)
		}
		if !reg.IsEnabled(name) {
			t.Errorf("保留清单工具 %s 应保持启用", name)
		}
	}
	// pair 独有工具保留在注册表但被禁用（agent 不可见，前端可见可恢复）
	for _, name := range []string{"read_file", "write_file", "codegraph_search", "memory_read",
		"project_info_write", "git_diff", "debug_inject_log", "binary_hash", "csv_read", "web_debug",
		"go_build", "fix_flex_autoheight"} {
		tool, ok := reg.Get(name)
		if !ok {
			t.Errorf("pair 独有工具 %s 应从注册表保留（禁用而非删除，内置工具集可恢复）", name)
			continue
		}
		if tool.Enabled {
			t.Errorf("pair 独有工具 %s 应被禁用（agent 不可见）", name)
		}
	}
	// 全部工具仍在注册表（前端 /api/tools 可见），但 Definitions 只导出启用项
	if got := len(reg.AllToolMeta()); got != before {
		t.Errorf("禁用后注册表工具数不应变化，before=%d after=%d", before, got)
	}
	if defs := reg.Definitions(); len(defs) != len(HarnessAlignedToolNames) {
		t.Errorf("Definitions 应只导出 %d 个启用工具（agent 可调用），实际 %d", len(HarnessAlignedToolNames), len(defs))
	}
	// 禁用工具调用被拦截
	if _, err := reg.Execute(context.Background(), "codegraph_search", "{}"); err == nil {
		t.Error("禁用工具 codegraph_search 调用应被拦截")
	}
	// 恢复（内置工具集加入场景）：SetToolEnabled(true) → agent 可见
	reg.SetToolEnabled("codegraph_search", true)
	if !reg.IsEnabled("codegraph_search") {
		t.Error("SetToolEnabled(true) 应恢复工具（内置工具集加入语义）")
	}
	if defs := reg.Definitions(); len(defs) != len(HarnessAlignedToolNames)+1 {
		t.Errorf("恢复后 Definitions 应多 1 个（codegraph_search），实际 %d", len(defs))
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
	t.Setenv("WB_HARNESS", "1")
	reg := mkHarnessReg()
	before := len(reg.AllToolMeta())
	first := ApplyHarnessToolFilter(reg, nil)
	second := ApplyHarnessToolFilter(reg, nil)
	if second != 0 {
		t.Errorf("第二次过滤应禁用 0 个（幂等），实际 %d（第一次 %d）", second, first)
	}
	if got := len(reg.AllToolMeta()); got != before {
		t.Errorf("幂等后工具数不应变化，before=%d after=%d", before, got)
	}
	if defs := reg.Definitions(); len(defs) != len(HarnessAlignedToolNames) {
		t.Errorf("幂等后 Definitions 应剩 %d 个启用工具，实际 %d", len(HarnessAlignedToolNames), len(defs))
	}
}

func TestApplyHarnessToolFilter_KeepsHooks(t *testing.T) {
	t.Setenv("WB_HARNESS", "1")
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
	// ★ 2026-08-16 反转默认：默认全量工具集（插件面板默认全勾，对 agent 可见），
	//   harness 对齐模式需显式 WB_HARNESS=1 开启
	t.Setenv("WB_FULL_TOOLS", "")
	t.Setenv("WB_HARNESS", "")
	if HarnessOnlyTools() {
		t.Error("默认（未设任何开关）应处于全量工具模式（harness 对齐默认关闭）")
	}
	t.Setenv("WB_FULL_TOOLS", "1")
	if HarnessOnlyTools() {
		t.Error("WB_FULL_TOOLS=1 应强制全量模式")
	}
	t.Setenv("WB_FULL_TOOLS", "")
	t.Setenv("WB_HARNESS", "1")
	if !HarnessOnlyTools() {
		t.Error("WB_HARNESS=1 应开启 harness 对齐模式")
	}
}

// 被移除的 pair 独有工具名（harness 精简提示词中不应出现）。
var trimmedPromptBannedTools = []string{
	"codegraph", "memory_", "project_info", "history_", "git_", "debug_", "binary_",
	"csv_", "word_", "xlsx", "read_pdf", "lsp_", "skill_", "mcp_",
	"marketplace", "web_debug", "bug_", "screenshot", "multi_edit", "list_files",
	"run_background", "update_plan", "read_file", "edit_file", "write_file", "run_command",
	"search_content", "search_files", "find_symbol", "go_build", "go_run", "run_test",
	"fix_flex_autoheight", "image_",
}

func TestPromptTrimmedInHarnessMode(t *testing.T) {
	t.Setenv("WB_HARNESS", "1")
	roots := []string{"/test/project"}

	// 自管理/记忆两段在 harness 模式下应裁剪为空（引用工具已被移除）
	if s := SelfManagementPrompt(); s != "" {
		t.Errorf("harness 模式 SelfManagementPrompt 应为空，实际: %.80s…", s)
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
	// ★ 工具描述已取消（2026-08-17）：提示词中不再注入任何工具名/用法说明——
	//   协议工具（update_tasks/generate_commit_message 等）同样不点名，
	//   工具名称与用法完全由 tools 参数 schema 提供。
	for _, banned := range []string{"update_tasks", "generate_commit_message", "update_plan",
		"read_file", "edit_file", "write_file", "run_command", "web_search", "web_fetch",
		"cordis_define", "cordis_run", "cordis_inspect", "toolset_build", "toolset_show",
		"ask_user", "str_replace_editor"} {
		if strings.Contains(p, banned) {
			t.Errorf("harness 精简提示词不应引用工具名 %q（工具信息以 tools 参数 schema 为准）", banned)
		}
	}
}

func TestPromptFullInFullToolsMode(t *testing.T) {
	t.Setenv("WB_FULL_TOOLS", "1")
	roots := []string{"/test/project"}
	if s := SelfManagementPrompt(); s == "" {
		t.Error("WB_FULL_TOOLS=1 时 SelfManagementPrompt 不应为空")
	}
	if s := LongTermMemoryPrompt(); s == "" {
		t.Error("WB_FULL_TOOLS=1 时 LongTermMemoryPrompt 不应为空")
	}
	// 完整版提示词长度应显著大于精简版（保留验证流程等行为段）
	full := DefaultSystemPrompt(roots)
	t.Setenv("WB_FULL_TOOLS", "")
	t.Setenv("WB_HARNESS", "1")
	trimmed := DefaultSystemPrompt(roots)
	if len(full) <= len(trimmed) {
		t.Errorf("完整版提示词应比精简版长：full=%d trimmed=%d", len(full), len(trimmed))
	}
	// ★ 工具描述已取消（2026-08-17）：完整版同样不应包含 codegraph 等工具说明
	if strings.Contains(full, "codegraph") {
		t.Error("WB_FULL_TOOLS=1 完整版提示词不应包含 codegraph 工具说明（工具信息以 tools 参数 schema 为准）")
	}
}
