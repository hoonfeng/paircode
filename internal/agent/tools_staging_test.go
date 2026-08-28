package agent

import (
	"context"
	"testing"
)

// ── 首步极简工具面（StagedTools）测试 ──

// TestFilterMinimalTools 验证极简工具面过滤。
func TestFilterMinimalTools(t *testing.T) {
	mk := func(names ...string) []ToolDefinition {
		var out []ToolDefinition
		for _, n := range names {
			out = append(out, ToolDefinition{Type: "function", Function: FunctionDefinition{Name: n, Description: "d"}})
		}
		return out
	}
	full := mk("read_file", "grep", "codegraph_build", "write_file", "binary_patch", "run_command", "edit_file")
	min := FilterMinimalTools(full)
	if len(min) != 5 { // read_file/write_file/run_command/edit_file + grep
		t.Fatalf("极简面应为 5 个，得 %d: %+v", len(min), min)
	}
	for _, d := range min {
		switch d.Function.Name {
		case "read_file", "write_file", "run_command", "edit_file", "grep":
		default:
			t.Errorf("不应保留 %s", d.Function.Name)
		}
	}
	// harness 命名面：候选组选别名（read/grep/glob/bash/write/edit/str_replace_editor）
	host := mk("read", "grep", "glob", "bash", "write", "edit", "str_replace_editor", "run_code", "binary_patch")
	if got := FilterMinimalTools(host); len(got) != 8 {
		t.Fatalf("harness 面极简应为 8 个，得 %d: %+v", len(got), got)
	}
	// 全非极简 → 兜底返回原面（防无工具可用）
	other := mk("codegraph_build", "binary_patch")
	if got := FilterMinimalTools(other); len(got) != 2 {
		t.Errorf("兜底应返回原面，得 %d", len(got))
	}
	// 空安全
	if got := FilterMinimalTools(nil); len(got) != 0 {
		t.Errorf("空输入应返回空，得 %d", len(got))
	}
}

// recordingProvider 记录每次 Chat 收到的工具面（用于 staged 验证）。
type recordingProvider struct {
	inner       *MockProvider
	toolsSizes  []int
	firstTools  []string
	secondTools []string
}

func (r *recordingProvider) Name() string { return "recording" }
func (r *recordingProvider) Calls() int   { return r.inner.Calls() }

func (r *recordingProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	names := make([]string, 0, len(tools))
	for _, d := range tools {
		names = append(names, d.Function.Name)
	}
	r.toolsSizes = append(r.toolsSizes, len(tools))
	if len(r.toolsSizes) == 1 {
		r.firstTools = names
	} else if len(r.toolsSizes) == 2 {
		r.secondTools = names
	}
	return r.inner.Chat(ctx, messages, tools, onChunk)
}

// TestJSLoopStagedToolsFirstStepMinimal 首步极简 → 后续恢复（JS 循环路径）。
func TestJSLoopStagedToolsFirstStepMinimal(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	reg := NewRegistry()
	RegisterDefaultTools(reg, t.TempDir())
	fullCount := len(reg.Definitions())

	rp := &recordingProvider{inner: &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{
			{ID: "s1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.txt"}`}},
		}},
		{Content: "完成"},
	}}}
	loop := &Loop{Provider: rp, Registry: reg, System: "staged-e2e", MaxIterations: 5, StagedTools: true}
	if _, err := loop.Run(context.Background(), "读 a.txt", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rp.toolsSizes) != 2 {
		t.Fatalf("应有 2 次 LLM 调用，得 %d", len(rp.toolsSizes))
	}
	// 首步极简（长度以当前面的过滤结果为准）
	wantMin := len(FilterMinimalTools(reg.Definitions()))
	if rp.toolsSizes[0] != wantMin {
		t.Errorf("首步工具面应为 %d 个，得 %d", wantMin, rp.toolsSizes[0])
	}
	if len(rp.firstTools) != wantMin {
		t.Errorf("首步工具 = %v", rp.firstTools)
	}
	// 第二步恢复全量
	if rp.toolsSizes[1] != fullCount {
		t.Errorf("第二步应恢复全量 %d 个，得 %d", fullCount, rp.toolsSizes[1])
	}
}

// TestJSLoopStagedToolsDisabled 关闭后两步均全量。
func TestJSLoopStagedToolsDisabled(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	reg := NewRegistry()
	RegisterDefaultTools(reg, t.TempDir())
	fullCount := len(reg.Definitions())

	rp := &recordingProvider{inner: &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{
			{ID: "s1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.txt"}`}},
		}},
		{Content: "完成"},
	}}}
	loop := &Loop{Provider: rp, Registry: reg, System: "staged-off-e2e", MaxIterations: 5, StagedTools: false}
	if _, err := loop.Run(context.Background(), "读 a.txt", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rp.toolsSizes) != 2 {
		t.Fatalf("应有 2 次 LLM 调用，得 %d", len(rp.toolsSizes))
	}
	if rp.toolsSizes[0] != fullCount || rp.toolsSizes[1] != fullCount {
		t.Errorf("关闭后应为全量×2，得 %v", rp.toolsSizes)
	}
}

// TestGoLoopStagedToolsFirstStepMinimal Go 回退循环同样生效。
func TestGoLoopStagedToolsFirstStepMinimal(t *testing.T) {
	reg := NewRegistry()
	RegisterDefaultTools(reg, t.TempDir())
	fullCount := len(reg.Definitions())

	rp := &recordingProvider{inner: &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{
			{ID: "s1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.txt"}`}},
		}},
		{Content: "完成"},
	}}}
	loop := &Loop{Provider: rp, Registry: reg, System: "go-staged", MaxIterations: 5, StagedTools: true}
	if _, err := loop.Run(context.Background(), "读 a.txt", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rp.toolsSizes[0] != len(FilterMinimalTools(reg.Definitions())) {
		t.Errorf("Go 循环首步应极简，得 %d", rp.toolsSizes[0])
	}
	if rp.toolsSizes[1] != fullCount {
		t.Errorf("Go 循环第二步应恢复全量 %d，得 %d", fullCount, rp.toolsSizes[1])
	}
}

// TestMinimalToolGroups 默认组回退 + 自定义组（FilterMinimalToolsWith）。
func TestMinimalToolGroups(t *testing.T) {
	// 默认组（插件未配置）：≥8 组
	if got := DefaultStagedGroups(); len(got) < 8 {
		t.Fatalf("默认组应 ≥8 组，得 %d", len(got))
	}
	// ngroups 为空 → 回退默认组
	full := []ToolDefinition{
		{Type: "function", Function: FunctionDefinition{Name: "read_file", Description: "d"}},
		{Type: "function", Function: FunctionDefinition{Name: "run_command", Description: "d"}},
	}
	if got := FilterMinimalToolsWith(full, nil); len(got) != 2 {
		t.Fatalf("空组应回退默认过滤，得 %d", len(got))
	}

	// 自定义组 → 按配置过滤（含组内别名命中）
	groups := [][]string{
		{"custom_read"},
		{"custom_write"},
		{"custom_grep", "fallback_grep"},
	}
	full = []ToolDefinition{
		{Type: "function", Function: FunctionDefinition{Name: "custom_read", Description: "d"}},
		{Type: "function", Function: FunctionDefinition{Name: "custom_write", Description: "d"}},
		{Type: "function", Function: FunctionDefinition{Name: "fallback_grep", Description: "d"}},
		{Type: "function", Function: FunctionDefinition{Name: "zzz_not_in_groups", Description: "d"}},
	}
	got := FilterMinimalToolsWith(full, groups)
	if len(got) != 3 {
		t.Fatalf("按配置应保留 3 个（含 fallback_grep 别名命中），得 %d: %+v", len(got), got)
	}
	names := make(map[string]bool)
	for _, d := range got {
		names[d.Function.Name] = true
	}
	for _, n := range []string{"custom_read", "custom_write", "fallback_grep"} {
		if !names[n] {
			t.Errorf("缺少 %s", n)
		}
	}
	if names["zzz_not_in_groups"] {
		t.Error("不应保留组外工具")
	}
}

// TestNewLoopStagedGroupsTransferred LoopOpts 候选组经 newLoop 传入 Loop 字段。
func TestNewLoopStagedGroupsTransferred(t *testing.T) {
	groups := [][]string{{"read_file"}, {"write_file"}}
	l := newLoop(LoopOpts{StagedToolGroups: groups})
	if len(l.StagedToolGroups) != 2 || l.StagedToolGroups[0][0] != "read_file" {
		t.Fatalf("候选组未传入 Loop: %+v", l.StagedToolGroups)
	}
	// 未配置 → nil（过滤时回退默认）
	if l2 := newLoop(LoopOpts{}); l2.StagedToolGroups != nil {
		t.Errorf("未配置应为 nil，得 %+v", l2.StagedToolGroups)
	}
}


