package agent

// 缓存命中诊断测试（2026-08-17 排查：缓存命中率降低）
// 验证：① Definitions() 顺序稳定性；② 注册顺序 vs 字典序差异；
//       ③ 描述截断逻辑（无效 UTF-8 风险）；④ Usage 解析 OpenAI 兼容字段。

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

// 模拟一批真实工具名（插件化后的注册来源多样：内置组、磁盘插件、工具集、MCP）。
var diagToolNames = []string{
	"read_file", "write_file", "edit_file", "multi_edit", "run_command", "move_file",
	"delete_file", "search_content", "search_files", "git_status", "git_diff", "git_log",
	"web_fetch", "web_search", "run_background", "read_output", "kill_process",
	"memory_write", "memory_read", "memory_search", "project_info_read", "project_info_tree",
	"update_tasks", "update_plan", "tool_stats", "history_search", "codegraph_build",
	"codegraph_search", "codegraph_impact", "bug_detect", "bug_fix",
	"csv_read", "word_read", "screenshot_desktop", "web_debug",
	"cordis_define", "cordis_run", "toolset_build", "mcp_add",
	"ask_user", "run_code", "str_replace_editor", "debug_start", "debug_stop",
	"memory_verify", "project_info_verify", "skill_load", "skill_write",
}

func diagToolDef(name string) ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        name,
			Description: "工具 " + name + " 的用途描述：用于完成特定开发任务，包含参数说明与使用注意事项。",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string", "description": "文件路径"}}},
		},
	}
}

//  1. 注册顺序稳定性：按 r.order 输出的 Definitions 依赖装配时序；
//     两次调用同一 Registry 应一致（确定性），但不同装配顺序会导致不同前缀。
func TestDiag_DefinitionsRegistrationOrder(t *testing.T) {
	// 模拟两种装配顺序（如插件加载时序不同）：正序 vs 逆序
	r1 := NewRegistry()
	r2 := NewRegistry()
	for i, n := range diagToolNames {
		d := diagToolDef(n)
		r1.Register(&Tool{Name: d.Function.Name, Description: d.Function.Description, Parameters: d.Function.Parameters})
		_ = i
	}
	for i := len(diagToolNames) - 1; i >= 0; i-- {
		n := diagToolNames[i]
		d := diagToolDef(n)
		r2.Register(&Tool{Name: d.Function.Name, Description: d.Function.Description, Parameters: d.Function.Parameters})
	}

	b1, _ := json.Marshal(r1.Definitions())
	b2, _ := json.Marshal(r2.Definitions())
	// ★ 2026-08-17 修复后：Definitions 按 name 字典序排序，不同装配时序输出一致
	if string(b1) != string(b2) {
		t.Fatalf("★ 回归：不同装配时序 Definitions 不一致（KV 缓存前缀断裂）")
	}
	t.Logf("✓ 注册顺序不同（正序/逆序）→ Definitions 字典序排序后 JSON 完全一致（len=%d）", len(b1))
}

// 2) 字典序排序后的稳定性：任意装配顺序，排序后一致。
func TestDiag_DefinitionsSortedStable(t *testing.T) {
	sorted := func(r *Registry) []byte {
		defs := r.Definitions()
		// 与 harness orderTools 同语义：按 name 字典序（code-unit）
		for i := 1; i < len(defs); i++ {
			for j := i; j > 0 && defs[j].Function.Name < defs[j-1].Function.Name; j-- {
				defs[j], defs[j-1] = defs[j-1], defs[j]
			}
		}
		b, _ := json.Marshal(defs)
		return b
	}
	r1 := NewRegistry()
	r2 := NewRegistry()
	for _, n := range diagToolNames {
		d := diagToolDef(n)
		r1.Register(&Tool{Name: d.Function.Name, Description: d.Function.Description, Parameters: d.Function.Parameters})
	}
	for i := len(diagToolNames) - 1; i >= 0; i-- {
		n := diagToolNames[i]
		d := diagToolDef(n)
		r2.Register(&Tool{Name: d.Function.Name, Description: d.Function.Description, Parameters: d.Function.Parameters})
	}
	b1, b2 := sorted(r1), sorted(r2)
	if string(b1) != string(b2) {
		t.Fatalf("字典序排序后仍不一致！")
	}
	t.Logf("字典序排序后两种装配顺序 tools JSON 完全一致（len=%d）→ 前缀稳定", len(b1))
}

// 3) 描述截断逻辑：无中文句号的超长中文描述 → 是否产生无效 UTF-8。
func TestDiag_DescTruncationValidUTF8(t *testing.T) {
	// 超长中文描述（无中文句号，有英文句号）
	desc := strings.Repeat("这是一个没有中文句号的超长中文工具描述用于测试截断逻辑是否会产生无效的UTF8字节序列", 3) + " with english period."
	if len(desc) <= 120 {
		t.Fatalf("测试数据不够长: %d", len(desc))
	}
	cut := strings.LastIndex(desc[:120], "。")
	if cut < 60 {
		cut = len(desc[:100])
		for cut > 60 && desc[cut] != ' ' && desc[cut] != ',' {
			cut--
		}
	}
	// ★ 2026-08-17 修复后：trimToolDesc 用 []rune 截断，始终落在字符边界
	out2 := trimToolDesc(desc)
	if !utf8.ValidString(out2) {
		t.Errorf("★ 修复后 trimToolDesc 仍产生无效 UTF-8：%q", out2)
	} else {
		t.Logf("✓ trimToolDesc 截断结果有效 UTF-8（len(runes)=%d, out=%d runes）", len([]rune(desc)), len([]rune(out2)))
	}
	// 原实现字节截断的破坏性对照（desc[:100] 落在多字节字符中间）
	if !utf8.ValidString(desc[:100]) {
		t.Logf("对照：旧实现 desc[:100] 字节切片为无效 UTF-8（乱码根源，已修复）")
	}
}

//  4. Usage 解析：OpenAI 兼容端点返回 prompt_tokens_details.cached_tokens，
//     而 agent.Usage 只绑定 prompt_cache_hit_tokens —— 命中统计会显示为 0。
func TestDiag_UsageOpenAICompat(t *testing.T) {
	// DeepSeek 专有拼写
	rawDeepSeek := `{"prompt_tokens":1000,"completion_tokens":50,"total_tokens":1050,"prompt_cache_hit_tokens":800,"prompt_cache_miss_tokens":200}`
	var u1 Usage
	if err := json.Unmarshal([]byte(rawDeepSeek), &u1); err != nil {
		t.Fatal(err)
	}
	// OpenAI 兼容拼写（网关/其他端点）
	rawOpenAI := `{"prompt_tokens":1000,"completion_tokens":50,"total_tokens":1050,"prompt_tokens_details":{"cached_tokens":800}}`
	var u2 Usage
	if err := json.Unmarshal([]byte(rawOpenAI), &u2); err != nil {
		t.Fatal(err)
	}
	t.Logf("DeepSeek 拼写: hit=%d miss=%d", u1.PromptCacheHitTokens, u1.PromptCacheMissTokens)
	t.Logf("OpenAI 兼容拼写: hit=%d miss=%d ← 参考实现 mapUsage 两者都处理；本实现若只绑定 DeepSeek 字段，命中显示 0", u2.PromptCacheHitTokens, u2.PromptCacheMissTokens)
	if u2.PromptCacheHitTokens == 0 && u2.PromptCacheMissTokens == 0 {
		t.Log("★ 结论：OpenAI 兼容端点命中统计丢失（prompt_tokens_details.cached_tokens 未解析）")
	}
}

// 5) 修复验证：Definitions 现在按字典序排序 → 不同装配时序下 tools JSON 完全一致。
func TestDiag_DefinitionsSortedNowStable(t *testing.T) {
	r1 := NewRegistry()
	r2 := NewRegistry()
	for _, n := range diagToolNames {
		d := diagToolDef(n)
		r1.Register(&Tool{Name: d.Function.Name, Description: d.Function.Description, Parameters: d.Function.Parameters})
	}
	for i := len(diagToolNames) - 1; i >= 0; i-- {
		n := diagToolNames[i]
		d := diagToolDef(n)
		r2.Register(&Tool{Name: d.Function.Name, Description: d.Function.Description, Parameters: d.Function.Parameters})
	}
	b1, _ := json.Marshal(r1.Definitions())
	b2, _ := json.Marshal(r2.Definitions())
	if string(b1) != string(b2) {
		t.Fatalf("★ 修复未生效：不同装配时序下 Definitions 仍不一致")
	}
	// 验证确实是字典序
	defs := r1.Definitions()
	for i := 1; i < len(defs); i++ {
		if defs[i-1].Function.Name > defs[i].Function.Name {
			t.Fatalf("★ 非字典序：%q > %q", defs[i-1].Function.Name, defs[i].Function.Name)
		}
	}
	t.Logf("✓ 修复生效：不同装配时序 Definitions 完全一致（len=%d，字典序）→ 缓存前缀稳定", len(b1))
}

// 6) 修复验证：Usage 兼容 OpenAI 兼容拼写 cached_tokens。
func TestDiag_UsageOpenAICompatFixed(t *testing.T) {
	rawOpenAI := `{"prompt_tokens":1000,"completion_tokens":50,"total_tokens":1050,"prompt_tokens_details":{"cached_tokens":800}}`
	var u Usage
	if err := json.Unmarshal([]byte(rawOpenAI), &u); err != nil {
		t.Fatal(err)
	}
	if u.PromptCacheHitTokens != 800 {
		t.Fatalf("★ 修复未生效：OpenAI 兼容拼写 cached_tokens 未解析，hit=%d", u.PromptCacheHitTokens)
	}
	if u.PromptCacheMissTokens != 200 {
		t.Fatalf("★ miss 未从 prompt_tokens 反推：miss=%d", u.PromptCacheMissTokens)
	}
	t.Logf("✓ 修复生效：OpenAI 兼容拼写 hit=%d miss=%d", u.PromptCacheHitTokens, u.PromptCacheMissTokens)
}

// 7) 修复验证：CondenseHistoryByPressure 小历史不压缩（缓存前缀稳定）。
func TestDiag_NoCompressWhenSmall(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "任务1"},
		{Role: RoleAssistant, Content: "完成1"},
		{Role: RoleUser, Content: "任务2"},
		{Role: RoleAssistant, Content: "完成2"},
		{Role: RoleUser, Content: "任务3"},
		{Role: RoleAssistant, Content: "完成3"},
		{Role: RoleUser, Content: "任务4"},
		{Role: RoleAssistant, Content: "完成4"},
		{Role: RoleUser, Content: "任务5"},
		{Role: RoleAssistant, Content: "完成5"},
		{Role: RoleUser, Content: "任务6"},
	}
	// 小历史（估算 token 远低于 64K 窗口的 45%）→ 不压缩，前缀逐字节稳定
	out := CondenseHistoryByPressure(msgs, 64000)
	if len(out) != len(msgs) {
		t.Fatalf("★ 修复未生效：小历史仍被压缩 in=%d out=%d", len(msgs), len(out))
	}
	for i := range msgs {
		if out[i].Role != msgs[i].Role || out[i].Content != msgs[i].Content {
			t.Fatalf("位置 %d 被修改：%q vs %q", i, out[i].Content, msgs[i].Content)
		}
	}
	t.Logf("✓ 修复生效：小历史（%d 条）保持原样 → 前缀稳定，缓存可连续命中", len(msgs))
	// 对照：旧 CondenseHistory 同输入会压缩（轮数触发）
	old := CondenseHistory(msgs)
	t.Logf("对照：旧按轮数 CondenseHistory 压缩到 %d 条（每轮断裂前缀）", len(old))
}
