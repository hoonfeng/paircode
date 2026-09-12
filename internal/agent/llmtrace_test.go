package agent

// llmtrace 注册表/触发链测试（LLM 追踪数据面）。

import (
	"strings"
	"testing"
	"time"

	"github.com/hoonfeng/paircode/goja"
)

// 保存/恢复全局注册表（避免污染他测）。
func llmTraceSnapshot(t *testing.T) (restore func()) {
	t.Helper()
	llmTraceMu.Lock()
	saved := llmTraceEntries
	llmTraceEntries = nil
	llmTraceMu.Unlock()
	return func() {
		llmTraceMu.Lock()
		llmTraceEntries = saved
		llmTraceMu.Unlock()
	}
}

func TestLLMTraceRegisterFire(t *testing.T) {
	defer llmTraceSnapshot(t)()

	var got []LLMTraceEvent
	restore := RegisterLLMTracer(func(ev LLMTraceEvent) { got = append(got, ev) })
	defer restore()

	req := LLMTraceEvent{Phase: "request", Turn: 1, Step: 2, Provider: "openai:mock",
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
		Tools:    []ToolDefinition{{Function: FunctionDefinition{Name: "read"}}}}
	FireLLMTracer(req)
	resp := LLMTraceEvent{Phase: "response", Turn: 1, Step: 2, Provider: "openai:mock",
		Content: "hello", Usage: &Usage{PromptTokens: 100, PromptCacheHitTokens: 90, PromptCacheMissTokens: 10}}
	FireLLMTracer(resp)

	if len(got) != 2 {
		t.Fatalf("应收到 2 个事件，得 %d", len(got))
	}
	if got[0].Phase != "request" || got[0].When.IsZero() {
		t.Errorf("request 事件异常: %+v", got[0])
	}
	if len(got[0].Messages) != 1 || got[0].Messages[0].Content != "hi" {
		t.Errorf("request 应带完整消息: %+v", got[0].Messages)
	}
	if got[1].Phase != "response" || got[1].Usage == nil || got[1].Usage.PromptCacheHitTokens != 90 {
		t.Errorf("response 事件应带 usage（含缓存命中）: %+v", got[1])
	}
	// 撤销后不再触发
	restore()
	FireLLMTracer(LLMTraceEvent{Phase: "request"})
	if len(got) != 2 {
		t.Errorf("撤销后不应再收到事件，得 %d", len(got))
	}
}

func TestLLMTracePanicIsolation(t *testing.T) {
	defer llmTraceSnapshot(t)()

	var gotOK, gotErr bool
	RegisterLLMTracer(func(ev LLMTraceEvent) { panic("tracer boom") }) // 故意 panic
	RegisterLLMTracer(func(ev LLMTraceEvent) {
		if ev.Phase == "request" {
			gotOK = true
		}
	})
	RegisterLLMTracer(func(ev LLMTraceEvent) {
		if ev.Err != "" {
			gotErr = true
		}
	})

	// panic tracer 不应影响主流程与其余追踪器
	FireLLMTracer(LLMTraceEvent{Phase: "request"})
	FireLLMTracer(LLMTraceEvent{Phase: "response", Err: "boom"})
	if !gotOK {
		t.Error("panic 隔离失败：后续追踪器未收到事件")
	}
	if !gotErr {
		t.Error("response 事件应透传 Err")
	}
}

func TestLLMTraceUnregisterMultiple(t *testing.T) {
	defer llmTraceSnapshot(t)()

	a, b, c := 0, 0, 0
	ra := RegisterLLMTracer(func(ev LLMTraceEvent) { a++ })
	rb := RegisterLLMTracer(func(ev LLMTraceEvent) { b++ })
	rc := RegisterLLMTracer(func(ev LLMTraceEvent) { c++ })
	ra()
	FireLLMTracer(LLMTraceEvent{Phase: "request"})
	if a != 0 || b != 1 || c != 1 {
		t.Errorf("撤销错误: a=%d b=%d c=%d（期望 0/1/1）", a, b, c)
	}
	rb()
	rc()
	FireLLMTracer(LLMTraceEvent{Phase: "request"})
	if b != 1 || c != 1 {
		t.Errorf("撤销后仍触发: b=%d c=%d（期望 1/1）", b, c)
	}
}

// llmTraceToJS 字段完整性（JS 插件消费的数据形状）
func TestLLMTraceToJSShape(t *testing.T) {
	vm := goja.New()
	ev := LLMTraceEvent{Phase: "response", When: time.Now(), Turn: 3, Step: 4,
		Provider: "openai:mock", Content: "c", Reasoning: "r",
		StopReason: "stop",
		Usage: &Usage{PromptTokens: 50, CompletionTokens: 10, TotalTokens: 60,
			PromptCacheHitTokens: 40, PromptCacheMissTokens: 10,
			PromptBreakdown: PromptBreakdown{SystemTokens: 30, ToolTokens: 15, HistoryTokens: 5},
			Raw:             map[string]any{"prompt_cache_hit_tokens": float64(40)}},
		ToolCalls: []ToolCall{{ID: "t1", Type: "function", Function: FunctionCall{Name: "read", Arguments: `{"path":"a"}`}}}}
	got := llmTraceToJS(vm, ev)
	for _, k := range []string{"phase", "when", "turn", "step", "provider",
		"content", "reasoning", "stopReason", "usage", "usageEstimated", "toolCalls"} {
		if _, ok := got[k]; !ok {
			t.Errorf("缺少字段 %s", k)
		}
	}
	usage, _ := got["usage"].(map[string]any)
	if usage == nil || usage["promptCacheHitTokens"] != 40 || usage["source"] != "api" {
		t.Errorf("usage（API 真实值）字段异常: %+v", usage)
	}
	// ★ 职责分离：usage 只放 API 真实值，估算构成不得混入（防止被当真实用量分析）
	if _, bad := usage["systemTokens"]; bad {
		t.Errorf("usage 混入了估算字段 systemTokens: %+v", usage)
	}
	// ★ 原始报文必须透传（llm-trace 核对真实用量的依据）
	if raw, _ := usage["apiRaw"].(map[string]any); raw == nil || raw["prompt_cache_hit_tokens"] != float64(40) {
		t.Errorf("usage.apiRaw 未透传: %+v", usage["apiRaw"])
	}
	// ★ 派生命中率
	if rate, _ := usage["cacheHitRate"].(float64); rate != 0.8 {
		t.Errorf("cacheHitRate 异常: %+v", usage["cacheHitRate"])
	}
	est, _ := got["usageEstimated"].(map[string]any)
	if est == nil || est["systemTokens"] != 30 || est["source"] != "local-estimate" {
		t.Errorf("usageEstimated（本地估算）字段异常: %+v", est)
	}
	tcs, _ := got["toolCalls"].([]any)
	if len(tcs) != 1 {
		t.Fatalf("toolCalls 异常: %+v", tcs)
	}
	fn := tcs[0].(map[string]any)["function"].(map[string]any)
	if fn["name"] != "read" || !strings.Contains(fn["arguments"].(string), "a") {
		t.Errorf("toolCalls 内容异常: %+v", tcs)
	}
}
