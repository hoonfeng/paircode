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
	"codegraph_search", "codegraph_impact", "lsp_definition", "bug_detect", "bug_fix",
	"csv_read", "word_read", "screenshot_desktop", "web_debug", "image_analyze",
	"cordis_define", "cordis_run", "toolset_build", "mcp_add", "generate_commit_message",
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

// 1) 注册顺序稳定性：按 r.order 输出的 Definitions 依赖装配时序；
//    两次调用同一 Registry 应一致（确定性），但不同装配顺序会导致不同前缀。
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
	t.Logf("注册顺序不同 → tools JSON 字节一致? %v (len1=%d len2=%d)", string(b1) == string(b2), len(b1), len(b2))
	if string(b1) != string(b2) {
		t.Log("★ 结论：tools 顺序依赖注册时序 —— 装配顺序变化即破坏 KV 缓存前缀（参考实现按字典序排序保证跨次稳定）")
	}
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
	out := strings.TrimSpace(desc[:cut])
	if !utf8.ValidString(out) {
		t.Errorf("★ 截断结果含无效 UTF-8（cut=%d，切在多字节字符中间）→ json.Marshal 会转义为 \\ufffd，工具描述乱码；虽不破坏前缀稳定性，但 schema 质量受损", cut)
	} else {
		t.Logf("截断结果有效 UTF-8（cut=%d）", cut)
	}
	// 边界：正好 100 字节落在多字节字符中间的情况
	if !utf8.ValidString(desc[:100]) {
		t.Logf("★ desc[:100] 本身即无效 UTF-8（第 100 字节落在多字节字符中间）→ LastIndex/TrimSpace 在无效字节上操作，截断点随内容漂移")
	}
}

// 4) Usage 解析：OpenAI 兼容端点返回 prompt_tokens_details.cached_tokens，
//    而 agent.Usage 只绑定 prompt_cache_hit_tokens —— 命中统计会显示为 0。
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
