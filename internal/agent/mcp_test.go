// MCP 客户端测试：用 go-sdk 的 InMemoryTransport + Server 进程内验证。
// 覆盖：主流程（list/call）、结构化输出、IsError、分页、自动重连、超时、细粒度 HITL。

package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// testTool 一对工具定义 + handler，便于构造测试服务器。
type testTool struct {
	def     *mcp.Tool
	handler mcp.ToolHandler
}

// emptySchema 最小合法输入 schema（type=object，AddTool 强制要求非 nil）。
// tools.go 已有 objSchema(props, ...string)，这里用无参版避免冲突。
func emptySchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// startTestServer 启动进程内 MCP 测试服务器，返回 client 侧 transport 和 cancel。
// server 必须先于 client 连接（go-sdk 要求），所以 go routine 启动 server.Run。
func startTestServer(t *testing.T, pageSize int, tools ...testTool) (*mcp.InMemoryTransport, context.CancelFunc) {
	t.Helper()
	clientT, serverT := mcp.NewInMemoryTransports()
	opts := &mcp.ServerOptions{PageSize: pageSize}
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "1.0"}, opts)
	for _, tt := range tools {
		server.AddTool(tt.def, tt.handler)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = server.Run(ctx, serverT)
	}()
	return clientT, cancel
}

// newTestConn 用注入的 transport 创建并连接 mcpConnection（测试用，不走子进程）。
func newTestConn(t *testing.T, name string, transport mcp.Transport) *mcpConnection {
	t.Helper()
	conn := newMCPConnection(MCPServerConfig{Name: name})
	conn.transport = transport
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := conn.connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	return conn
}

// TestMCPListAndCall 主流程：listAllTools 发现工具 + callTool 返回 TextContent。
func TestMCPListAndCall(t *testing.T) {
	ct, cancel := startTestServer(t, 100, testTool{
		def:     &mcp.Tool{Name: "echo", Description: "回显文本", InputSchema: emptySchema()},
		handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "echoed"}}}, nil
		},
	})
	defer cancel()
	conn := newTestConn(t, "test", ct)
	defer conn.close()

	tools, err := conn.listAllTools(context.Background())
	if err != nil {
		t.Fatalf("listAllTools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("工具列表 = %+v，期望 [echo]", tools)
	}

	out, err := conn.callTool(context.Background(), "echo", nil)
	if err != nil {
		t.Fatalf("callTool: %v", err)
	}
	if out != "echoed" {
		t.Errorf("结果 = %q，期望 'echoed'", out)
	}
}

// TestMCPStructuredContent 结构化输出：StructuredContent 优先于 TextContent 兜底。
func TestMCPStructuredContent(t *testing.T) {
	ct, cancel := startTestServer(t, 100, testTool{
		def: &mcp.Tool{Name: "struct", Description: "结构化输出", InputSchema: emptySchema()},
		handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				StructuredContent: map[string]any{"code": 200, "msg": "ok"},
				Content:           []mcp.Content{&mcp.TextContent{Text: "should-be-ignored"}},
			}, nil
		},
	})
	defer cancel()
	conn := newTestConn(t, "struct", ct)
	defer conn.close()

	out, err := conn.callTool(context.Background(), "struct", nil)
	if err != nil {
		t.Fatalf("callTool: %v", err)
	}
	if !strings.Contains(out, `"code":200`) || !strings.Contains(out, `"msg":"ok"`) {
		t.Errorf("结果 = %q，期望包含结构化 JSON（code/msg）", out)
	}
}

// TestMCPIsError 工具返回 IsError=true：parseCallToolResult 应转为 error。
func TestMCPIsError(t *testing.T) {
	ct, cancel := startTestServer(t, 100, testTool{
		def: &mcp.Tool{Name: "fail", Description: "失败工具", InputSchema: emptySchema()},
		handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "执行失败: 权限不足"}},
			}, nil
		},
	})
	defer cancel()
	conn := newTestConn(t, "fail", ct)
	defer conn.close()

	_, err := conn.callTool(context.Background(), "fail", nil)
	if err == nil {
		t.Fatal("期望 IsError 错误，得 nil")
	}
	if !strings.Contains(err.Error(), "权限不足") {
		t.Errorf("错误 = %q，期望包含 '权限不足'", err.Error())
	}
}

// TestMCPPagination 分页拉取：服务器 PageSize=5，注册 12 个工具，验证全部拉取。
func TestMCPPagination(t *testing.T) {
	const pageSize = 5
	const total = 12
	tools := make([]testTool, total)
	for i := 0; i < total; i++ {
		idx := i
		tools[i] = testTool{
			def: &mcp.Tool{
				Name:        fmt.Sprintf("tool%d", idx),
				Description: fmt.Sprintf("工具%d", idx),
				InputSchema: emptySchema(),
			},
			handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("r%d", idx)}}}, nil
			},
		}
	}
	ct, cancel := startTestServer(t, pageSize, tools...)
	defer cancel()
	conn := newTestConn(t, "page", ct)
	defer conn.close()

	list, err := conn.listAllTools(context.Background())
	if err != nil {
		t.Fatalf("listAllTools: %v", err)
	}
	if len(list) != total {
		t.Fatalf("分页拉取工具数 = %d，期望 %d（PageSize=%d）", len(list), total, pageSize)
	}
}

// TestMCPReconnect 自动重连：关闭 server1 模拟断连，替换 transport 为 server2，
// callTool 内部 ensureAlive 检测 Ping 失败 → 重连 → 调用新 server 工具。
func TestMCPReconnect(t *testing.T) {
	ct1, cancel1 := startTestServer(t, 100, testTool{
		def:     &mcp.Tool{Name: "v1", Description: "版本1", InputSchema: emptySchema()},
		handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "v1"}}}, nil
		},
	})
	conn := newTestConn(t, "recon", ct1)

	// 初始可用
	out, err := conn.callTool(context.Background(), "v1", nil)
	if err != nil || out != "v1" {
		t.Fatalf("初始调用: %v / %q", err, out)
	}

	// 关闭 server1 模拟断连
	cancel1()

	// 启动 server2（新 transport）
	ct2, cancel2 := startTestServer(t, 100, testTool{
		def:     &mcp.Tool{Name: "v2", Description: "版本2", InputSchema: emptySchema()},
		handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "v2"}}}, nil
		},
	})
	defer cancel2()

	// 替换 transport（持锁，与 ensureAlive 互斥）
	conn.mu.Lock()
	conn.transport = ct2
	conn.mu.Unlock()

	// callTool 内部 ensureAlive：Ping 失败 → 重连 ct2 → 调用 v2
	out, err = conn.callTool(context.Background(), "v2", nil)
	if err != nil {
		t.Fatalf("重连后调用: %v", err)
	}
	if out != "v2" {
		t.Errorf("重连后结果 = %q，期望 'v2'", out)
	}
	conn.close()
}

// TestMCPTimeout 超时：callTimeout 缩短为 100ms，慢工具 2s 才返回，验证提前超时返回错误。
func TestMCPTimeout(t *testing.T) {
	ct, cancel := startTestServer(t, 100, testTool{
		def: &mcp.Tool{Name: "slow", Description: "慢工具", InputSchema: emptySchema()},
		handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			select {
			case <-time.After(2 * time.Second):
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "done"}}}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	})
	defer cancel()
	conn := newTestConn(t, "timeout", ct)
	defer conn.close()
	conn.callTimeout = 100 * time.Millisecond // 缩短超时

	start := time.Now()
	_, err := conn.callTool(context.Background(), "slow", nil)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("期望超时错误，得 nil")
	}
	if elapsed > 2*time.Second {
		t.Errorf("未在超时内返回，耗时 %v（期望 <2s）", elapsed)
	}
}

// TestMCPRegistryHITL 细粒度 HITL：MCP 工具注册到 Registry，RequiresApproval=true；
// BeforeTool 钩子可拒绝（短路）或放行（执行 handler）。
func TestMCPRegistryHITL(t *testing.T) {
	ct, cancel := startTestServer(t, 100, testTool{
		def:     &mcp.Tool{Name: "danger", Description: "危险工具", InputSchema: emptySchema()},
		handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "executed"}}}, nil
		},
	})
	defer cancel()
	conn := newTestConn(t, "hitl", ct)
	defer conn.close()

	reg := NewRegistry()
	n, err := registerClientTools(reg, conn)
	if err != nil || n != 1 {
		t.Fatalf("注册: %v / %d", err, n)
	}
	tool, ok := reg.Get("mcp__hitl__danger")
	if !ok {
		t.Fatal("mcp__hitl__danger 未注册")
	}
	if !tool.RequiresApproval {
		t.Error("MCP 外部工具应 RequiresApproval=true")
	}

	// BeforeTool 拒绝（短路，不执行 handler）
	reg.BeforeTool = func(ctx context.Context, name string, args map[string]any) (bool, string, error) {
		return false, "", fmt.Errorf("用户拒绝执行 %s", name)
	}
	_, err = reg.Execute(context.Background(), "mcp__hitl__danger", "{}")
	if err == nil {
		t.Fatal("期望被拒绝错误，得 nil")
	}
	if !strings.Contains(err.Error(), "用户拒绝执行") {
		t.Errorf("拒绝错误 = %q", err.Error())
	}

	// BeforeTool 放行（执行 handler）
	var called bool
	reg.BeforeTool = func(ctx context.Context, name string, args map[string]any) (bool, string, error) {
		called = true
		return true, "", nil
	}
	res, err := reg.Execute(context.Background(), "mcp__hitl__danger", "{}")
	if err != nil {
		t.Fatalf("放行后执行: %v", err)
	}
	if !called {
		t.Error("BeforeTool 未被调用")
	}
	if res != "executed" {
		t.Errorf("执行结果 = %q，期望 'executed'", res)
	}
}
