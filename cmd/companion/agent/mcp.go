// MCP 客户端：经 stdio JSON-RPC 连接 MCP 服务器，发现其工具并注册进 Registry（handler 代理到 tools/call），
// 让 agent 用上任意外部 MCP 工具（filesystem/github/...）。stdio 传输=换行分隔的 JSON-RPC 2.0。
// 传输层抽象成 io.Writer/Reader → 协议逻辑可用 io.Pipe + 假服务器离线测（见 mcp_test.go）。

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// MCPServerConfig 一个 MCP 服务器启动配置。
type MCPServerConfig struct {
	Name    string            // 工具名前缀（来自配置 key）
	Command string            // 可执行命令
	Args    []string          // 参数
	Env     map[string]string // 额外环境变量
}

// mcpClient 一个 MCP 连接：同步 JSON-RPC（agent 串行调用，一把锁足够）。
type mcpClient struct {
	mu     sync.Mutex
	w      io.Writer
	r      *bufio.Reader
	nextID int
}

func newMCPClient(w io.Writer, r io.Reader) *mcpClient {
	return &mcpClient{w: w, r: bufio.NewReader(r)}
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcMessage struct {
	ID     *int            `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

// call 同步请求：写一行 JSON，读到匹配 id 的响应（跳过日志行/通知）。
func (c *mcpClient) call(method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextID++
	id := c.nextID
	msg := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	b, _ := json.Marshal(msg)
	if _, err := c.w.Write(append(b, '\n')); err != nil {
		return nil, err
	}
	for {
		line, err := c.r.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		var m rpcMessage
		if json.Unmarshal(line, &m) != nil || m.ID == nil || *m.ID != id {
			continue // 非 JSON / 通知 / 别的 id → 跳过
		}
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, m.Error.Message)
		}
		return m.Result, nil
	}
}

func (c *mcpClient) notify(method string, params any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	msg := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		msg["params"] = params
	}
	b, _ := json.Marshal(msg)
	_, err := c.w.Write(append(b, '\n'))
	return err
}

func (c *mcpClient) initialize() error {
	if _, err := c.call("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "companion", "version": "1.0"},
	}); err != nil {
		return err
	}
	return c.notify("notifications/initialized", nil)
}

type mcpToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func (c *mcpClient) listTools() ([]mcpToolDef, error) {
	res, err := c.call("tools/list", nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Tools []mcpToolDef `json:"tools"`
	}
	err = json.Unmarshal(res, &out)
	return out.Tools, err
}

func (c *mcpClient) callTool(name string, args map[string]any) (string, error) {
	res, err := c.call("tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return "", err
	}
	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(res, &out); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, ct := range out.Content {
		if ct.Text != "" {
			sb.WriteString(ct.Text)
			sb.WriteByte('\n')
		}
	}
	txt := strings.TrimSpace(sb.String())
	if out.IsError {
		return "", fmt.Errorf("%s", txt)
	}
	return txt, nil
}

// registerClientTools 把 client 的工具注册进 Registry，名加 "mcp.<server>." 前缀防冲突。返回工具数。
func registerClientTools(r *Registry, serverName string, c *mcpClient) (int, error) {
	tools, err := c.listTools()
	if err != nil {
		return 0, err
	}
	for _, td := range tools {
		schema := td.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		mcpName := td.Name // 捕获
		r.Register(&Tool{
			Name:             "mcp." + serverName + "." + td.Name,
			Description:      "[MCP:" + serverName + "] " + td.Description,
			Parameters:       schema,
			RequiresApproval: true, // 外部工具默认需审批（安全）
			Handler: func(ctx context.Context, args map[string]any) (string, error) {
				return c.callTool(mcpName, args)
			},
		})
	}
	return len(tools), nil
}

// connectMCP 启动一个 MCP 服务器进程并完成初始化（带超时，防卡住启动）。调用方持有 cmd 负责保活。
func connectMCP(cfg MCPServerConfig) (*mcpClient, *exec.Cmd, error) {
	c := exec.Command(cfg.Command, cfg.Args...)
	if len(cfg.Env) > 0 {
		c.Env = os.Environ()
		for k, v := range cfg.Env {
			c.Env = append(c.Env, k+"="+v)
		}
	}
	stdin, err := c.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := c.Start(); err != nil {
		return nil, nil, err
	}
	client := newMCPClient(stdin, stdout)
	done := make(chan error, 1)
	go func() { done <- client.initialize() }()
	select {
	case err := <-done:
		if err != nil {
			_ = c.Process.Kill()
			return nil, nil, err
		}
	case <-time.After(15 * time.Second):
		_ = c.Process.Kill()
		return nil, nil, fmt.Errorf("MCP 服务器 %s 初始化超时", cfg.Name)
	}
	return client, c, nil
}

// RegisterMCPServers 连接每个配置的 MCP 服务器并注册其工具；起不来的跳过（不阻断 agent）。返回注册的工具总数。
func RegisterMCPServers(r *Registry, configs []MCPServerConfig) int {
	total := 0
	for _, cfg := range configs {
		client, _, err := connectMCP(cfg)
		if err != nil {
			continue
		}
		if n, err := registerClientTools(r, cfg.Name, client); err == nil {
			total += n
		}
	}
	return total
}
