package agent

// jsloop_llmtrace_e2e_test.go — LLM 追踪完整链端到端：
// 装载真实 agentloop 磁盘插件（JS 循环）+ 真实 llm-trace 磁盘插件（ctx.llmtrace.register
// → ctx.fs 写 JSONL），用 MockProvider 驱动一轮「工具调用→最终回答」，
// 断言 JSONL 落盘且 request/response 两相字段完整。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJSLoopLLMTracePluginWritesJSONL(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()

	root := t.TempDir() // 工作区根 = 临时目录，fs 写日志落在其内

	// 装载 agentloop 磁盘插件（宿主绑定 root）
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, root)
	loadJS := func(t *testing.T, rel string) {
		t.Helper()
		code, err := os.ReadFile(filepath.Join("..", "..", rel))
		if err != nil {
			t.Skipf("无磁盘插件源码 %s: %v", rel, err)
		}
		id, err := host.DefineJS(string(code), "e2e "+rel)
		if err != nil {
			t.Fatalf("DefineJS(%s): %v", rel, err)
		}
		def, _ := host.GetJSDef(id)
		if err := host.LoadJSDynamic(def); err != nil {
			t.Fatalf("LoadJSDynamic(%s): %v", rel, err)
		}
		t.Cleanup(func() { _ = host.Unload(def.name) })
	}
	loadJS(t, filepath.Join(".pair", "plugins", "agentloop", "index.js"))
	if CurrentJSLoop() == nil {
		t.Fatal("agentloop 装载后 CurrentJSLoop 应为非空")
	}
	loadJS(t, filepath.Join(".pair", "plugins", "llm-trace", "index.js"))

	RegisterDefaultTools(reg, root)
	os.WriteFile(filepath.Join(root, "hello.txt"), []byte("LLMTRACE_E2E"), 0o644)

	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read", Arguments: `{"path":"hello.txt"}`}}}},
		{Content: "读到了 LLMTRACE_E2E"},
	}}
	loop := &Loop{Provider: mock, Registry: reg, System: "test-llm-trace",
		OnEvent: func(e Event) {}}

	if _, err := loop.Run(context.Background(), "读 hello.txt", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if mock.Calls() != 2 {
		t.Fatalf("LLM 应调用 2 次，得 %d", mock.Calls())
	}

	// 断言 JSONL：2 请求 + 2 响应 = 4 行
	day := time.Now().Format("20060102")
	logFile := filepath.Join(root, ".pair", "logs", "llm-trace", "llm-trace-"+day+".jsonl")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("读 JSONL 失败（插件未落盘）: %v（期望路径 %s）", err, logFile)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 4 {
		t.Fatalf("JSONL 应有 4 行（2 req + 2 resp），得 %d", len(lines))
	}
	var sawReqMsgs, sawRespContent, sawTools, sawTurn, sawRespCalls bool
	for _, ln := range lines {
		var ev map[string]any
		if err := json.Unmarshal([]byte(ln), &ev); err != nil {
			t.Fatalf("JSONL 行解析失败: %v\n%s", err, ln[:min(200, len(ln))])
		}
		switch ev["phase"] {
		case "request":
			if msgs, ok := ev["msgs"].([]any); !ok || len(msgs) == 0 {
				t.Error("request 行应含 msgs 数组")
			} else {
				sawReqMsgs = true
			}
			if tools, ok := ev["tools"].([]any); !ok || len(tools) == 0 {
				t.Error("request 行应含 tools 数组")
			} else {
				sawTools = true
			}
		case "response":
			if c, ok := ev["content"].(string); ok && c != "" {
				sawRespContent = true
			}
			if tcs, ok := ev["toolCalls"].([]any); ok && len(tcs) > 0 {
				sawRespCalls = true
			}
			if ev["provider"] == nil {
				t.Error("response 行应含 provider")
			}
		default:
			t.Errorf("未知 phase: %v", ev["phase"])
		}
		if tn, ok := ev["turn"].(float64); !ok || tn < 1 {
			t.Errorf("事件应含 turn>=1，得 %v", ev["turn"])
		} else {
			sawTurn = true
		}
	}
	if !sawReqMsgs || !sawRespContent || !sawTools || !sawTurn || !sawRespCalls {
		t.Errorf("字段覆盖不全: reqMsgs=%v respContent=%v respCalls=%v tools=%v turn=%v",
			sawReqMsgs, sawRespContent, sawRespCalls, sawTools, sawTurn)
	}
}
