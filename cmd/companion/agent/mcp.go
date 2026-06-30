// MCP 客户端：用官方 go-sdk（github.com/modelcontextprotocol/go-sdk）经 stdio 连接 MCP 服务器，
// 发现其工具并注册进 Registry（handler 代理到 tools/call）。
// 能力：自动重连（Ping 检测 + 重启进程）、分页拉取（ListTools cursor）、结构化输出
// （StructuredContent 优先 + TextContent 兜底 + IsError）、超时（默认 30s）、细粒度 HITL
// （MCP 工具 RequiresApproval=true，由 Registry.BeforeTool 钩子统一审批）。

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServerConfig 一个 MCP 服务器启动配置。
type MCPServerConfig struct {
	Name    string            // 工具名前缀（来自配置 key）
	Command string            // 可执行命令
	Args    []string          // 参数
	Env     map[string]string // 额外环境变量
}

// mcpConnection 一个 MCP 连接（go-sdk）：持 session 支持重连，持 cmd 显式保活。
type mcpConnection struct {
	cfg         MCPServerConfig
	client      *mcp.Client
	mu          sync.Mutex
	session     *mcp.ClientSession
	cmd         *exec.Cmd // 显式保活引用（go-sdk 内部也持有，便于诊断）
	transport   mcp.Transport // 可注入（测试用 InMemoryTransport）；nil 时 connect 内部建 CommandTransport
	callTimeout time.Duration
}

// newMCPConnection 创建连接对象（未连接）。
func newMCPConnection(cfg MCPServerConfig) *mcpConnection {
	return &mcpConnection{
		cfg:         cfg,
		client:      mcp.NewClient(&mcp.Implementation{Name: "companion", Version: "1.0"}, nil),
		callTimeout: 30 * time.Second,
	}
}

// connect 启动进程并建立 session（调用方持 mu）。Connect 内部完成 initialize 握手。
// 若 transport 字段非 nil（测试注入），直接用它；否则建 CommandTransport 启动子进程。
func (c *mcpConnection) connect(ctx context.Context) error {
	var transport mcp.Transport
	if c.transport != nil {
		transport = c.transport
	} else {
		cmd := exec.Command(c.cfg.Command, c.cfg.Args...)
		if len(c.cfg.Env) > 0 {
			cmd.Env = os.Environ()
			for k, v := range c.cfg.Env {
				cmd.Env = append(cmd.Env, k+"="+v)
			}
		}
		transport = &mcp.CommandTransport{Command: cmd}
		c.cmd = cmd
	}
	session, err := c.client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("MCP %s 连接失败: %w", c.cfg.Name, err)
	}
	c.session = session
	return nil
}

// close 关闭连接（SIGTERM/SIGKILL 由 CommandTransport.Close 处理）。
func (c *mcpConnection) close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session != nil {
		err := c.session.Close()
		c.session = nil
		c.cmd = nil
		return err
	}
	return nil
}

// ensureAlive Ping 检测连接活性，失败则重连。
func (c *mcpConnection) ensureAlive(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == nil {
		return c.connect(ctx)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := c.session.Ping(pingCtx, nil); err != nil {
		// Ping 失败：关闭旧 session 并重连
		_ = c.session.Close()
		c.session = nil
		c.cmd = nil
		return c.connect(ctx)
	}
	return nil
}

// withRetry 可刷新错误重试一次（泛型）。先 ensureAlive，调用 fn，若返回可刷新错误
// （连接断开类）则重连后重试一次。
func withRetry[T any](ctx context.Context, c *mcpConnection, fn func(*mcp.ClientSession) (T, error)) (T, error) {
	var zero T
	if err := c.ensureAlive(ctx); err != nil {
		return zero, err
	}
	res, err := fn(c.session)
	if err == nil {
		return res, nil
	}
	if !isRefreshable(err) {
		return zero, err
	}
	// 重连重试一次
	c.mu.Lock()
	if c.session != nil {
		_ = c.session.Close()
		c.session = nil
		c.cmd = nil
	}
	c.mu.Unlock()
	if err := c.ensureAlive(ctx); err != nil {
		return zero, err
	}
	return fn(c.session)
}

// isRefreshable 判断可刷新错误（连接断开类，值得重连重试）。
func isRefreshable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection closed") ||
		strings.Contains(msg, "session missing") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset")
}

// listAllTools 分页拉取所有工具（按 NextCursor 翻页，直到无下一页）。
func (c *mcpConnection) listAllTools(ctx context.Context) ([]*mcp.Tool, error) {
	return withRetry(ctx, c, func(s *mcp.ClientSession) ([]*mcp.Tool, error) {
		var all []*mcp.Tool
		cursor := ""
		for {
			res, err := s.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
			if err != nil {
				return nil, err
			}
			all = append(all, res.Tools...)
			if res.NextCursor == "" {
				break
			}
			cursor = res.NextCursor
		}
		return all, nil
	})
}

// callTool 调用工具（带超时 + 重连重试）。
func (c *mcpConnection) callTool(ctx context.Context, name string, args map[string]any) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	return withRetry(cctx, c, func(s *mcp.ClientSession) (string, error) {
		res, err := s.CallTool(cctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			return "", err
		}
		return parseCallToolResult(res)
	})
}

// parseCallToolResult 解析工具结果：StructuredContent 优先（结构化输出）+ TextContent 兜底 + IsError 错误。
func parseCallToolResult(res *mcp.CallToolResult) (string, error) {
	// 优先 StructuredContent（结构化输出，JSON 对象）
	if res.StructuredContent != nil {
		if b, err := json.Marshal(res.StructuredContent); err == nil {
			text := string(b)
			if res.IsError {
				return "", fmt.Errorf("%s", text)
			}
			return text, nil
		}
	}
	// 兜底 Content（TextContent 拼接）
	var sb strings.Builder
	for _, ct := range res.Content {
		if tc, ok := ct.(*mcp.TextContent); ok && tc.Text != "" {
			sb.WriteString(tc.Text)
			sb.WriteByte('\n')
		}
	}
	text := strings.TrimSpace(sb.String())
	if res.IsError {
		return "", fmt.Errorf("%s", text)
	}
	return text, nil
}

// registerClientTools 把连接的工具注册进 Registry，名加 "mcp.<server>." 前缀防冲突。返回工具数。
// MCP 工具 RequiresApproval=true（外部工具默认需审批）；细粒度 HITL 由 Registry.BeforeTool 钩子
// 统一处理（阶段一已加钩子链），调用方可在 BeforeTool 中按 "mcp.<server>.<tool>" 前缀做白名单。
func registerClientTools(r *Registry, conn *mcpConnection) (int, error) {
	tools, err := conn.listAllTools(context.Background())
	if err != nil {
		return 0, err
	}
	serverName := conn.cfg.Name
	for _, td := range tools {
		// InputSchema 客户端侧为 map[string]any（go-sdk 文档），但类型是 any，需断言
		var schema map[string]any
		if td.InputSchema != nil {
			if m, ok := td.InputSchema.(map[string]any); ok {
				schema = m
			}
		}
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		toolName := td.Name // 捕获
		toolDesc := td.Description
		r.Register(&Tool{
			Name:             "mcp." + serverName + "." + td.Name,
			Description:      "[MCP:" + serverName + "] " + toolDesc,
			Parameters:       schema,
			RequiresApproval: true,
			Handler: func(ctx context.Context, args map[string]any) (string, error) {
				return conn.callTool(ctx, toolName, args)
			},
		})
	}
	return len(tools), nil
}

// connectMCP 启动并初始化一个 MCP 服务器连接（带 15s 超时，防卡住启动）。
func connectMCP(cfg MCPServerConfig) (*mcpConnection, error) {
	conn := newMCPConnection(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := conn.connect(ctx); err != nil {
		return nil, err
	}
	return conn, nil
}

// RegisterMCPServers 连接每个配置的 MCP 服务器并注册其工具；起不来的跳过（不阻断 agent）。返回注册的工具总数。
func RegisterMCPServers(r *Registry, configs []MCPServerConfig) int {
	total := 0
	for _, cfg := range configs {
		conn, err := connectMCP(cfg)
		if err != nil {
			continue
		}
		if n, err := registerClientTools(r, conn); err == nil {
			total += n
		}
	}
	return total
}
