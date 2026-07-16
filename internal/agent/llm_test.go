package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestOpenAIProviderParams 温度/maxTokens：>=0 / >0 时下发请求体；-1 / 0 时不下发（用服务端默认）。
func TestOpenAIProviderParams(t *testing.T) {
	sse := "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"
	capture := func(prov *OpenAIProvider) map[string]any {
		var body map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&body)
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w, sse)
		}))
		defer srv.Close()
		prov.BaseURL = srv.URL
		prov.Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil, func(Chunk) {})
		return body
	}
	b := capture(&OpenAIProvider{APIKey: "k", Model: "m", Temperature: 0.3, MaxTokens: 200})
	if b["temperature"] != 0.3 {
		t.Errorf("temperature = %v，期望 0.3", b["temperature"])
	}
	if b["max_tokens"] != float64(200) {
		t.Errorf("max_tokens = %v，期望 200", b["max_tokens"])
	}
	b2 := capture(&OpenAIProvider{APIKey: "k", Model: "m", Temperature: -1, MaxTokens: 0})
	if _, ok := b2["temperature"]; ok {
		t.Error("Temperature<0 不应下发 temperature")
	}
	if _, ok := b2["max_tokens"]; ok {
		t.Error("MaxTokens=0 不应下发 max_tokens")
	}
}

// TestApplyThinking 复刻参考 adapter.ts：思考参数仅 DeepSeek V4 系（model 含 "v4"）下发。
func TestApplyThinking(t *testing.T) {
	// 非 v4 模型：任何模式都不下发思考参数（避免被服务端拒绝）。
	for _, mode := range []string{"thinking", "thinking_max", "non-thinking"} {
		b := map[string]any{}
		applyThinking(b, "gpt-4o", mode)
		if _, ok := b["thinking"]; ok {
			t.Errorf("非 v4 模型(mode=%s)不应下发 thinking", mode)
		}
	}
	// v4 + thinking → enabled + high
	b := map[string]any{}
	applyThinking(b, "deepseek-v4-pro", "thinking")
	if tk, _ := b["thinking"].(map[string]any); tk["type"] != "enabled" {
		t.Errorf("thinking 应 enabled，得 %v", b["thinking"])
	}
	if b["reasoning_effort"] != "high" {
		t.Errorf("reasoning_effort 应 high，得 %v", b["reasoning_effort"])
	}
	// v4 + thinking_max → enabled + max
	b = map[string]any{}
	applyThinking(b, "deepseek-v4-flash", "thinking_max")
	if b["reasoning_effort"] != "max" {
		t.Errorf("reasoning_effort 应 max，得 %v", b["reasoning_effort"])
	}
	// v4 + non-thinking → disabled，无 reasoning_effort
	b = map[string]any{}
	applyThinking(b, "deepseek-v4-pro", "non-thinking")
	if tk, _ := b["thinking"].(map[string]any); tk["type"] != "disabled" {
		t.Errorf("non-thinking 应 disabled，得 %v", b["thinking"])
	}
	if _, ok := b["reasoning_effort"]; ok {
		t.Error("non-thinking 不应带 reasoning_effort")
	}
	// 空模式 → 不下发（宿主未设思考模式时）
	b = map[string]any{}
	applyThinking(b, "deepseek-v4-pro", "")
	if len(b) != 0 {
		t.Errorf("空模式不应下发任何思考参数，得 %v", b)
	}
}

// OpenAI 兼容 SSE 适配器：用 httptest 喂 canned 流（正文分 2 片 + 一个跨 2 片拼接的 tool_call + usage + [DONE]），
// 验证 content 累积、tool_call arguments 按 index 拼接、流式 onChunk、请求体正确。全离线、无真网络。
func TestOpenAIProviderSSE(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"你好"}}]}`,
		`data: {"choices":[{"delta":{"content":"，世界"}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"pa"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"th\":\"a.txt\"}"}}]}}]}`,
		`data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
		`data: [DONE]`,
	}, "\n\n") + "\n\n"

	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization 头 = %q", r.Header.Get("Authorization"))
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse)
	}))
	defer srv.Close()

	prov := &OpenAIProvider{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	tools := []ToolDefinition{{Type: "function", Function: FunctionDefinition{Name: "read_file"}}}
	var streamed strings.Builder
	msg, err := prov.Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, tools,
		func(c Chunk) { streamed.WriteString(c.Content) })
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	// 请求体：stream=true、model 正确、带 tools
	if gotBody["stream"] != true || gotBody["model"] != "test-model" {
		t.Errorf("请求体 stream/model 异常: %+v", gotBody)
	}
	if _, ok := gotBody["tools"]; !ok {
		t.Error("请求体应含 tools")
	}

	// 正文累积 + 流式 onChunk
	if msg.Content != "你好，世界" {
		t.Errorf("content = %q", msg.Content)
	}
	if streamed.String() != "你好，世界" {
		t.Errorf("streamed = %q", streamed.String())
	}

	// tool_call 按 index 拼接
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("应 1 个 tool_call，得 %d", len(msg.ToolCalls))
	}
	tc := msg.ToolCalls[0]
	if tc.ID != "call_1" || tc.Function.Name != "read_file" || tc.Function.Arguments != `{"path":"a.txt"}` {
		t.Errorf("tool_call 拼接结果 = %+v", tc)
	}
}

// 非 200 → 返回带状态码的错误。
func TestOpenAIProviderHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":"invalid key"}`)
	}))
	defer srv.Close()
	prov := &OpenAIProvider{BaseURL: srv.URL, APIKey: "bad", Model: "m"}
	if _, err := prov.Chat(context.Background(), []Message{{Role: RoleUser, Content: "x"}}, nil, nil); err == nil {
		t.Error("HTTP 401 应返回错误")
	} else if !strings.Contains(err.Error(), "401") {
		t.Errorf("错误应含状态码 401，得 %v", err)
	}
}
