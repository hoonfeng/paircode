package agent

import (
	"bufio"
	"encoding/json"
	"io"
	"testing"
)

// TestMCPClient 用 io.Pipe + 假 MCP 服务器离线验证：initialize → tools/list 注册 → tools/call 代理。
func TestMCPClient(t *testing.T) {
	r1, w1 := io.Pipe() // client → server
	r2, w2 := io.Pipe() // server → client
	go fakeMCPServer(r1, w2)
	client := newMCPClient(w1, r2)

	if err := client.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	reg := NewRegistry()
	n, err := registerClientTools(reg, "test", client)
	if err != nil {
		t.Fatalf("listTools: %v", err)
	}
	if n != 1 {
		t.Fatalf("应注册 1 个工具，得 %d", n)
	}
	tool, ok := reg.Get("mcp.test.echo")
	if !ok {
		t.Fatal("mcp.test.echo 未注册")
	}
	if !tool.RequiresApproval {
		t.Error("MCP 外部工具应需审批")
	}

	out, err := client.callTool("echo", map[string]any{"text": "hi"})
	if err != nil {
		t.Fatalf("callTool: %v", err)
	}
	if out != "echoed: hi" {
		t.Errorf("结果 = %q，期望 'echoed: hi'", out)
	}
}

// fakeMCPServer 极简 MCP 服务器：按换行读 JSON-RPC，回 initialize/tools/list/tools/call。
func fakeMCPServer(r io.Reader, w io.Writer) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var req struct {
			ID     *int            `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(sc.Bytes(), &req) != nil || req.ID == nil {
			continue // 非法 / 通知（无 id）不回
		}
		var result any
		switch req.Method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{}}
		case "tools/list":
			result = map[string]any{"tools": []map[string]any{{
				"name": "echo", "description": "回显文本",
				"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}}},
			}}}
		case "tools/call":
			var p struct {
				Arguments map[string]any `json:"arguments"`
			}
			_ = json.Unmarshal(req.Params, &p)
			txt, _ := p.Arguments["text"].(string)
			result = map[string]any{"content": []map[string]any{{"type": "text", "text": "echoed: " + txt}}}
		default:
			result = map[string]any{}
		}
		resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": *req.ID, "result": result})
		_, _ = w.Write(append(resp, '\n'))
	}
}
