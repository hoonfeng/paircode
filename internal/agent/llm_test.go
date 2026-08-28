package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

// ★ 2026-08-21 多模态：buildOpenAIMessages 把带 Images 的 user 消息转 content 块数组。
func TestBuildOpenAIMessagesMultimodal(t *testing.T) {
	msgs := []Message{
		{Role: RoleUser, Content: "看这张图", Images: []ImagePart{
			{Data: "data:image/png;base64,AAAA", MimeType: "image/png", Detail: "auto"},
		}},
		{Role: RoleUser, Content: "纯文本"},
	}
	// multimodal=true：带图消息转块数组
	out := buildOpenAIMessages(msgs, true)
	if len(out) != 2 {
		t.Fatalf("应 2 条消息，得 %d", len(out))
	}
	blocks, ok := out[0]["content"].([]map[string]any)
	if !ok {
		t.Fatalf("multimodal 消息 content 应为块数组，得 %T", out[0]["content"])
	}
	if len(blocks) != 2 {
		t.Fatalf("应 2 个块（text+image），得 %d", len(blocks))
	}
	if blocks[0]["type"] != "text" || blocks[0]["text"] != "看这张图" {
		t.Errorf("块[0] 应为 text，得 %v", blocks[0])
	}
	iu, ok := blocks[1]["image_url"].(map[string]any)
	if !ok {
		t.Fatalf("块[1] 应为 image_url，得 %v", blocks[1])
	}
	if iu["url"] != "data:image/png;base64,AAAA" || iu["detail"] != "auto" {
		t.Errorf("image_url = %v", iu)
	}
	// 纯文本消息保持字符串 content
	if s, ok := out[1]["content"].(string); !ok || s != "纯文本" {
		t.Errorf("纯文本消息 content 应为字符串，得 %v", out[1]["content"])
	}

	// multimodal=false：即使带 Images 也保持字符串（避免非视觉模型 400）
	out2 := buildOpenAIMessages(msgs, false)
	if s, ok := out2[0]["content"].(string); !ok || s != "看这张图" {
		t.Errorf("非多模态时 content 应为字符串，得 %v", out2[0]["content"])
	}
}

// ★ 2026-08-21 SSE 流中断重试策略测试：
// ① 未产出内容的流中断（空流 EOF）→ 安全重试，最终成功；
// ② 已产出内容后中断（EOF 无 [DONE]）→ 不重试（避免重复输出），返回带累积内容的错误。
// ③ parseSSE 直接喂「无完成标记」的流 → 应报「提前结束」错误且携带已累积内容。
func TestOpenAIProviderSSEStreamRetry(t *testing.T) {
	oldBase := llmRetryBaseDelay
	llmRetryBaseDelay = time.Millisecond
	defer func() { llmRetryBaseDelay = oldBase }()

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		if n == 1 {
			// 第一次：直接断开（未产出任何内容）
			w.(http.Flusher).Flush()
			return // handler 返回 = 连接 EOF（无 [DONE]）
		}
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"重试成功\"}}]}\n\ndata: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer srv.Close()

	prov := &OpenAIProvider{BaseURL: srv.URL, APIKey: "k", Model: "m"}
	msg, err := prov.Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil, func(Chunk) {})
	if err != nil {
		t.Fatalf("空流中断应重试后成功，得 err=%v", err)
	}
	if atomic.LoadInt32(&hits) < 2 {
		t.Errorf("应至少请求 2 次，得 %d", atomic.LoadInt32(&hits))
	}
	if msg.Content != "重试成功" {
		t.Errorf("content = %q", msg.Content)
	}
}

func TestOpenAIProviderSSEStreamInterruptNoRetry(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		// 产出部分内容后直接断开（无 [DONE]）
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"一半\"}}]}\n\n")
	}))
	defer srv.Close()

	prov := &OpenAIProvider{BaseURL: srv.URL, APIKey: "k", Model: "m"}
	_, err := prov.Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil, func(Chunk) {})
	if err == nil {
		t.Fatal("已产出内容后中断应返回错误")
	}
	if !strings.Contains(err.Error(), "提前结束") && !strings.Contains(err.Error(), "中断") {
		t.Errorf("错误信息应说明流中断，得 %v", err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("已产出内容不应重试，请求次数 = %d（期望 1）", atomic.LoadInt32(&hits))
	}
}

func TestParseSSEIncompleteStream(t *testing.T) {
	// 无 [DONE]/finish_reason 的流：内容部分保留，错误说明已累积量
	msg, err := parseSSE(strings.NewReader(
		"data: {\"choices\":[{\"delta\":{\"content\":\"你好\"}}]}\n\n"+
			"data: {\"choices\":[{\"delta\":{\"content\":\"，世界\"}}]}\n\n"+
			"data: {\"choices\":[{\"delta\":{\"content\":\"！\"}}]}\n\n"), nil)
	if err == nil {
		t.Fatal("无完成标记的流应报错（提前结束）")
	}
	if msg.Content != "你好，世界！" {
		t.Errorf("错误时应保留已累积内容，得 %q", msg.Content)
	}
	if !strings.Contains(err.Error(), "提前结束") {
		t.Errorf("应提示提前结束，得 %v", err)
	}
}

// TestOpenAIProviderURLNoAppend ★ 2026-08-27：配置 URL 即完整请求端点（含 /chat/completions），
// 直接使用不再拼接——防止 URL 二次拼接（如 /chat/completions/chat/completions）导致服务商无法访问。
func TestOpenAIProviderURLNoAppend(t *testing.T) {
	sse := "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"
	var gotPath, gotRaw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotRaw = r.URL.RequestURI()
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse)
	}))
	defer srv.Close()

	full := srv.URL + "/v1/chat/completions"
	prov := &OpenAIProvider{BaseURL: full, APIKey: "k", Model: "m"}
	_, err := prov.Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil, func(Chunk) {})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if gotRaw != "/v1/chat/completions" {
		t.Errorf("应直接使用配置 URL，得 %q", gotRaw)
	}
	if gotPath != "/v1/chat/completions" {
		t.Errorf("路径异常：%q", gotPath)
	}
}
