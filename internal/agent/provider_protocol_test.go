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

// ─── 协议端点解析 ──────────────────────────────────────────────

// TestResolveEndpointURL 协议 → 端点拼接（基础地址语义 2026-09-02 定稿）：
// 配置只存 base；完整端点按 Protocol 拼接；旧「完整端点」配置原样使用不重复拼。
func TestResolveEndpointURL(t *testing.T) {
	cases := []struct {
		base, protocol, want string
	}{
		{"https://api.deepseek.com/v1", "", "https://api.deepseek.com/v1/chat/completions"},
		{"https://api.deepseek.com/v1", "openai-completions", "https://api.deepseek.com/v1/chat/completions"},
		{"https://api.deepseek.com/v1/", "openai-completions", "https://api.deepseek.com/v1/chat/completions"},
		// 旧完整端点配置 → 原样使用
		{"https://api.deepseek.com/v1/chat/completions", "openai-completions", "https://api.deepseek.com/v1/chat/completions"},
		// responses 协议
		{"https://api.openai.com/v1", "openai-responses", "https://api.openai.com/v1/responses"},
		{"https://api.openai.com", "openai-responses", "https://api.openai.com/v1/responses"},
		// 旧 chat/completions 配置切到 responses → 剥后缀重拼 v1/responses
		{"https://api.openai.com/v1/chat/completions", "openai-responses", "https://api.openai.com/v1/responses"},
		// anthropic 协议
		{"https://api.anthropic.com/v1", "anthropic-messages", "https://api.anthropic.com/v1/messages"},
		{"https://api.anthropic.com", "anthropic-messages", "https://api.anthropic.com/v1/messages"},
	}
	for _, c := range cases {
		if got := ResolveEndpointURL(c.base, c.protocol); got != c.want {
			t.Errorf("ResolveEndpointURL(%q,%q) = %q，期望 %q", c.base, c.protocol, got, c.want)
		}
	}
}

// ─── CreateProvider 协议路由 ──────────────────────────────────

func TestCreateProviderByProtocol(t *testing.T) {
	if p := CreateProvider(ProviderParams{Protocol: "anthropic-messages", Model: "m"}); p == nil {
		t.Fatal("anthropic-messages 应创建 provider")
	} else if _, ok := p.(*AnthropicProvider); !ok {
		t.Fatalf("anthropic-messages 应返回 *AnthropicProvider，got %T", p)
	}
	if p := CreateProvider(ProviderParams{Protocol: "openai-responses", Model: "m"}); p == nil {
		t.Fatal("openai-responses 应创建 provider")
	} else if _, ok := p.(*ResponsesProvider); !ok {
		t.Fatalf("openai-responses 应返回 *ResponsesProvider，got %T", p)
	}
	if p := CreateProvider(ProviderParams{Protocol: "", Model: "m"}); p == nil {
		t.Fatal("空协议应创建 provider")
	} else if _, ok := p.(*OpenAIProvider); !ok {
		t.Fatalf("空协议应回退 *OpenAIProvider，got %T", p)
	}
	if p := CreateProvider(ProviderParams{Protocol: "openai-completions", Model: "m"}); p == nil {
		t.Fatal("openai-completions 应创建 provider")
	} else if _, ok := p.(*OpenAIProvider); !ok {
		t.Fatalf("openai-completions 应返回 *OpenAIProvider，got %T", p)
	}
}

// ─── Anthropic 适配器 ──────────────────────────────────────────

// TestAnthropicProviderRequestShape 验证 Anthropic 请求形态：
// x-api-key 认证、顶层 system、messages 无 system 角色、tool_use/tool_result 块、
// 工具 input_schema、max_tokens 必填。
func TestAnthropicProviderRequestShape(t *testing.T) {
	sse := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_1","usage":{"input_tokens":10,"output_tokens":1}}}`,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text"}}`,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"你好"}}`,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"，世界"}}`,
		`event: content_block_stop`,
		`data: {"type":"content_block_stop","index":0}`,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":7}}`,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
	}, "\n\n") + "\n\n"

	var gotBody map[string]any
	var gotAuth, gotVersion, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse)
	}))
	defer srv.Close()

	prov := &AnthropicProvider{BaseURL: srv.URL, APIKey: "k-ant-1", Model: "claude-4", MaxTokens: 512}
	msgs := []Message{
		{Role: RoleSystem, Content: "系统规则"},
		{Role: RoleUser, Content: "hi"},
	}
	var streamed strings.Builder
	msg, err := prov.Chat(context.Background(), msgs, nil, func(c Chunk) { streamed.WriteString(c.Content) })
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}
	if msg.Content != "你好，世界" {
		t.Errorf("content = %q，期望「你好，世界」", msg.Content)
	}
	if streamed.String() != "你好，世界" {
		t.Errorf("流式 content = %q", streamed.String())
	}
	if gotPath != "/v1/messages" {
		t.Errorf("路径 = %q，期望 /v1/messages", gotPath)
	}
	if gotAuth != "k-ant-1" {
		t.Errorf("x-api-key = %q", gotAuth)
	}
	if gotVersion != "2023-06-01" {
		t.Errorf("anthropic-version = %q", gotVersion)
	}
	if gotBody["system"] != "系统规则" {
		t.Errorf("system 应为顶层字段，得 %v", gotBody["system"])
	}
	if gotBody["max_tokens"] != float64(512) {
		t.Errorf("max_tokens = %v，期望 512", gotBody["max_tokens"])
	}
	msgsArr := gotBody["messages"].([]any)
	if len(msgsArr) != 1 {
		t.Fatalf("messages 应为 1 条（不含 system），得 %d", len(msgsArr))
	}
}

// TestAnthropicToolShape 工具调用与结果转换：assistant tool_use 块 + tool_result 合并。
func TestAnthropicToolShape(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"type":"message_start","message":{"id":"m","usage":{"input_tokens":5,"output_tokens":1}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"read_file","input":{}}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"a.txt\"}"}}`,
		`data: {"type":"content_block_stop","index":0}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":9}}`,
		`data: {"type":"message_stop"}`,
	}, "\n\n") + "\n\n"

	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse)
	}))
	defer srv.Close()

	prov := &AnthropicProvider{BaseURL: srv.URL, APIKey: "k", Model: "claude-4"}
	msgs := []Message{
		{Role: RoleUser, Content: "读文件"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "toolu_1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.txt"}`}}}},
		{Role: RoleTool, ToolCallID: "toolu_1", Content: "内容"},
		{Role: RoleUser, Content: "继续"},
	}
	tools := []ToolDefinition{{Type: "function", Function: FunctionDefinition{Name: "read_file", Description: "读文件", Parameters: map[string]any{"type": "object"}}}}
	msg, err := prov.Chat(context.Background(), msgs, tools, nil)
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %d，期望 1", len(msg.ToolCalls))
	}
	tc := msg.ToolCalls[0]
	if tc.ID != "toolu_1" || tc.Function.Name != "read_file" {
		t.Errorf("工具调用 = %+v", tc)
	}
	if tc.Function.Arguments != `{"path":"a.txt"}` {
		t.Errorf("arguments = %q", tc.Function.Arguments)
	}
	if msg.ToolCalls[0].Type != "function" {
		t.Errorf("工具类型应为 function（内部契约），得 %q", tc.Type)
	}

	// 请求体形态
	gotTools := gotBody["tools"].([]any)
	t0 := gotTools[0].(map[string]any)
	if t0["name"] != "read_file" {
		t.Errorf("tools[0] name = %v", t0["name"])
	}
	if _, ok := t0["input_schema"]; !ok {
		t.Errorf("tools[0] 应含 input_schema，得 %v", t0)
	}
	msgsArr := gotBody["messages"].([]any)
	// 期望：user(读文件) → assistant(tool_use) → user(tool_result)，最后的 user(继续) 应成为独立 user
	// 注意：tool_result 后紧跟 user(继续) 时中间无 assistant，会合并进同一条 user（Anthropic 兼容优先）
	if len(msgsArr) != 3 {
		t.Fatalf("messages = %d 条，期望 3（user/assistant/user[tool_result+继续]）", len(msgsArr))
	}
	asm := msgsArr[1].(map[string]any)
	if asm["role"] != "assistant" {
		t.Errorf("msgs[1] role = %v", asm["role"])
	}
	blocks := asm["content"].([]any)
	b0 := blocks[0].(map[string]any)
	if b0["type"] != "tool_use" || b0["id"] != "toolu_1" {
		t.Errorf("assistant content[0] = %v", b0)
	}
}

// TestAnthropicSSEInterruptedError 未收到 message_stop → 提前截断错误（已累积内容返回）。
func TestAnthropicSSEInterruptedError(t *testing.T) {
	sse := "data: {\"type\":\"message_start\",\"message\":{\"id\":\"m\",\"usage\":{\"input_tokens\":1}}}\n\n" +
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\"}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"断流\"}}\n\n"
	prov := &AnthropicProvider{BaseURL: newSSEServer(t, sse), APIKey: "k", Model: "claude-4"}
	msg, err := prov.Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil, nil)
	if err == nil {
		t.Fatal("未收到 message_stop 应报错")
	}
	if msg.Content != "断流" {
		t.Errorf("错误时应带已累积 content=%q", msg.Content)
	}
}

// newSSEServer 简易 SSE 服务器（返回固定流）。
func newSSEServer(t *testing.T, sse string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// ─── Responses 适配器 ──────────────────────────────────────────

// TestResponsesProviderRequestShape 验证 Responses 请求形态：
// instructions 顶层、input 混合数组（function_call_output 顶层）、max_output_tokens、reasoning.effort。
func TestResponsesProviderRequestShape(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"r","status":"in_progress","output":[]}}`,
		`data: {"type":"response.output_item.added","output_index":0,"item":{"id":"msg_1","type":"message","status":"in_progress","role":"assistant","content":[]}}`,
		`data: {"type":"response.content_part.added","item_id":"msg_1","output_index":0,"content_index":0,"part":{"type":"output_text","text":""}}`,
		`data: {"type":"response.output_text.delta","item_id":"msg_1","output_index":0,"content_index":0,"delta":"hi"}`,
		`data: {"type":"response.output_text.done","item_id":"msg_1","output_index":0,"content_index":0,"text":"hi"}`,
		`data: {"type":"response.content_part.done","item_id":"msg_1","output_index":0,"content_index":0,"part":{"type":"output_text","text":"hi"}}`,
		`data: {"type":"response.output_item.done","output_index":0,"item":{"id":"msg_1","type":"message","status":"completed","content":[]}}`,
		`data: {"type":"response.completed","response":{"id":"r","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"hi"}]}],"usage":{"input_tokens":10,"output_tokens":2,"total_tokens":12}}}`,
	}, "\n\n") + "\n\n"

	var gotBody map[string]any
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse)
	}))
	defer srv.Close()

	prov := &ResponsesProvider{BaseURL: srv.URL, APIKey: "k", Model: "gpt-4o", MaxTokens: 300, ThinkingMode: "thinking"}
	msgs := []Message{
		{Role: RoleSystem, Content: "规则"},
		{Role: RoleUser, Content: "hi"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "call_1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a"}`}}}},
		{Role: RoleTool, ToolCallID: "call_1", Content: "结果"},
	}
	var doneUsage *Usage
	msg, err := prov.Chat(context.Background(), msgs, nil, func(c Chunk) {
		if c.Done {
			doneUsage = c.Usage
		}
	})
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}
	if msg.Content != "hi" {
		t.Errorf("content = %q", msg.Content)
	}
	if doneUsage == nil || doneUsage.TotalTokens != 12 {
		t.Errorf("usage = %+v", doneUsage)
	}
	if gotPath != "/v1/responses" {
		t.Errorf("路径 = %q，期望 /v1/responses", gotPath)
	}
	if gotBody["instructions"] != "规则" {
		t.Errorf("instructions = %v", gotBody["instructions"])
	}
	if gotBody["max_output_tokens"] != float64(300) {
		t.Errorf("max_output_tokens = %v", gotBody["max_output_tokens"])
	}
	if r, ok := gotBody["reasoning"].(map[string]any); !ok || r["effort"] != "high" {
		t.Errorf("reasoning = %v，期望 effort=high", gotBody["reasoning"])
	}
	// input 形态：user → assistant(function_call 块) → function_call_output（顶层）
	inp := gotBody["input"].([]any)
	if len(inp) != 3 {
		t.Fatalf("input = %d 元素，期望 3", len(inp))
	}
	last := inp[2].(map[string]any)
	if last["type"] != "function_call_output" || last["call_id"] != "call_1" || last["output"] != "结果" {
		t.Errorf("input[2] = %v", last)
	}
	asm := inp[1].(map[string]any)
	ab := asm["content"].([]any)[0].(map[string]any)
	if ab["type"] != "function_call" || ab["call_id"] != "call_1" {
		t.Errorf("assistant content[0] = %v", ab)
	}
}

// TestResponsesFunctionCallSSE 工具调用事件流：function_call item + 参数 delta 累积。
func TestResponsesFunctionCallSSE(t *testing.T) {
	args := `{"path":"a.txt"}`
	// 用 json.Marshal 构造帧，避免手拼字符串嵌套引号转义错误
	mk := func(v map[string]any) string {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		return "data: " + string(b)
	}
	events := []string{
		mk(map[string]any{"type": "response.created", "response": map[string]any{"id": "r", "status": "in_progress", "output": []any{}}}),
		mk(map[string]any{"type": "response.output_item.added", "output_index": 0, "item": map[string]any{"id": "fc_1", "type": "function_call", "status": "in_progress", "name": "read_file", "arguments": "", "call_id": "call_1"}}),
		mk(map[string]any{"type": "response.function_call_arguments.delta", "item_id": "fc_1", "output_index": 0, "delta": `{"path":`}),
		mk(map[string]any{"type": "response.function_call_arguments.delta", "item_id": "fc_1", "output_index": 0, "delta": `"a.txt"}`}),
		mk(map[string]any{"type": "response.function_call_arguments.done", "item_id": "fc_1", "output_index": 0, "arguments": args}),
		mk(map[string]any{"type": "response.output_item.done", "output_index": 0, "item": map[string]any{"id": "fc_1", "type": "function_call", "status": "completed", "name": "read_file", "arguments": args, "call_id": "call_1"}}),
		mk(map[string]any{"type": "response.completed", "response": map[string]any{
			"id": "r", "status": "completed",
			"output": []any{map[string]any{"type": "function_call", "id": "fc_1", "call_id": "call_1", "name": "read_file", "arguments": args}},
			"usage":  map[string]any{"input_tokens": 5, "output_tokens": 9, "total_tokens": 14},
		}}),
	}
	sse := strings.Join(events, "\n\n") + "\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse)
	}))
	defer srv.Close()

	prov := &ResponsesProvider{BaseURL: srv.URL, APIKey: "k", Model: "gpt-4o"}
	var doneUsage *Usage
	msg, err := prov.Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil, func(c Chunk) {
		if c.Done {
			doneUsage = c.Usage
		}
	})
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %d，期望 1", len(msg.ToolCalls))
	}
	tc := msg.ToolCalls[0]
	if tc.ID != "call_1" || tc.Function.Name != "read_file" || tc.Function.Arguments != args {
		t.Errorf("工具调用 = %+v，期望 ID=call_1 name=read_file args=%s", tc, args)
	}
	if doneUsage == nil || doneUsage.PromptTokens != 5 || doneUsage.CompletionTokens != 9 {
		t.Errorf("usage = %+v", doneUsage)
	}
}

// TestResponsesSSEInterruptedError 未收到 response.completed → 提前截断错误。
func TestResponsesSSEInterruptedError(t *testing.T) {
	sse := "data: {\"type\":\"response.created\",\"response\":{\"id\":\"r\",\"status\":\"in_progress\",\"output\":[]}}\n\n" +
		"data: {\"type\":\"response.output_text.delta\",\"item_id\":\"m\",\"output_index\":0,\"delta\":\"部分\"}\n\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse)
	}))
	defer srv.Close()
	prov := &ResponsesProvider{BaseURL: srv.URL, APIKey: "k", Model: "gpt-4o"}
	msg, err := prov.Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil, nil)
	if err == nil {
		t.Fatal("未收到 response.completed 应报错")
	}
	if msg.Content != "部分" {
		t.Errorf("错误时应带已累积 content=%q", msg.Content)
	}
}
