// MCP 客户端：用官方 go-sdk（github.com/modelcontextprotocol/go-sdk）经 stdio 连接 MCP 服务器，
// 发现其工具并注册进 Registry（handler 代理到 tools/call）。
// 能力：自动重连（Ping 检测 + 重启进程）、分页拉取（ListTools cursor）、结构化输出
// （StructuredContent 优先 + TextContent 兜底 + IsError）、超时（默认 30s）、细粒度 HITL
// （MCP 工具 RequiresApproval=true，由 Registry.BeforeTool 钩子统一审批）。
//
// ★ 2026-09-15 连接池化（产品修复）：
//   - 跨会话复用：按「配置指纹」缓存连接，同一配置只 spawn 一次子进程
//     （原实现每条消息/每个会话都重建全部 MCP 连接，npx 首启动 5~15s/台且串行叠加）；
//   - 并行注册：RegisterMCPServers 并发连接全部服务器（原逐台串行，N 台 ×15s 超时）；
//   - 空闲回收：10 分钟（PAIR_MCP_IDLE_TTL_SEC 可调）未被使用的连接自动关闭子进程
//     （保留连接对象，下次使用按需重连）→ 不泄漏；扫描周期 PAIR_MCP_REAP_SEC 可调（默认 60s）；
//   - 退出清理：CloseAllMCPConnections 供 main 退出钩子统一回收，避免孤儿进程。
//   - 失败负缓存：起不来的服务器 30s 内不重复重试（避免每会话白等 15s 超时）。
//   - 失败清理提速：CommandTransport.TerminateDuration=2s（挂起服务器的关停尾巴从 5s 缩短）。

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hoonfeng/paircode/pkg/executil"
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
	cmd         *exec.Cmd     // 显式保活引用（go-sdk 内部也持有，便于诊断）
	transport   mcp.Transport // 可注入（测试用 InMemoryTransport）；nil 时 connect 内部建 CommandTransport
	callTimeout time.Duration
	lastUsed    atomic.Int64 // 最后使用时间（unixnano；空闲回收依据）
}

// newMCPConnection 创建连接对象（未连接）。
func newMCPConnection(cfg MCPServerConfig) *mcpConnection {
	c := &mcpConnection{
		cfg:         cfg,
		client:      mcp.NewClient(&mcp.Implementation{Name: "companion", Version: "1.0"}, nil),
		callTimeout: 30 * time.Second,
	}
	c.touch()
	return c
}

// touch 刷新最后使用时间（空闲回收依据）。
func (c *mcpConnection) touch() { c.lastUsed.Store(time.Now().UnixNano()) }

// connect 启动进程并建立 session（调用方持 mu）。Connect 内部完成 initialize 握手。
// 若 transport 字段非 nil（测试注入），直接用它；否则建 CommandTransport 启动子进程。
func (c *mcpConnection) connect(ctx context.Context) error {
	var transport mcp.Transport
	if c.transport != nil {
		transport = c.transport
	} else if mcpTestTransportFactory != nil {
		// 测试钩子：进程内 transport（InMemoryTransport），不走子进程
		transport = mcpTestTransportFactory(c.cfg)
	} else {
		cmd := exec.Command(c.cfg.Command, c.cfg.Args...)
		// 隐藏子进程控制台窗口（无控制台父进程时 console 程序会自己弹窗）
		if runtime.GOOS == "windows" {
			executil.HideWindow(cmd)
		}
		if len(c.cfg.Env) > 0 {
			cmd.Env = os.Environ()
			for k, v := range c.cfg.Env {
				cmd.Env = append(cmd.Env, k+"="+v)
			}
		}
		// TerminateDuration：关闭连接时等子进程自退的时长（默认 5s）。
		// 缩短到 2s：服务器已挂起（不响应 initialize/EOF）时，失败清理更快收尾；
		// 正常服务器收到 stdin EOF 会毫秒级自退，不受影响。
		transport = &mcp.CommandTransport{Command: cmd, TerminateDuration: 2 * time.Second}
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

// closeIdle 仅关闭底层会话（回收子进程），保留连接对象（下次使用自动重连）。
// 返回是否实际执行了关闭（幂等：无会话时返回 false，不重复计）。
func (c *mcpConnection) closeIdle() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == nil {
		return false
	}
	_ = c.session.Close()
	c.session = nil
	c.cmd = nil
	return true
}

// ensureAlive Ping 检测连接活性，失败则重连。
func (c *mcpConnection) ensureAlive(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == nil {
		if err := c.connect(ctx); err != nil {
			return err
		}
		log.Printf("[MCP] %s 会话重连成功（空闲回收/关闭后按需重连）", c.cfg.Name)
		return nil
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := c.session.Ping(pingCtx, nil); err != nil {
		// Ping 失败：关闭旧 session 并重连
		_ = c.session.Close()
		c.session = nil
		c.cmd = nil
		if err := c.connect(ctx); err != nil {
			return err
		}
		log.Printf("[MCP] %s 会话重连成功（Ping 失败后重建）", c.cfg.Name)
		return nil
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
	sess := c.getSession()
	if sess == nil {
		return zero, fmt.Errorf("MCP %s 会话不可用", c.cfg.Name)
	}
	res, err := fn(sess)
	if err == nil {
		return res, nil
	}
	if !isRefreshable(err) {
		return zero, err
	}
	// 重连重试一次：不粗暴关闭会话（可能已被并发调用者重连好），
	// 交给 ensureAlive 做 Ping 体检——活则复用，挂则重连。
	if err := c.ensureAlive(ctx); err != nil {
		return zero, err
	}
	sess = c.getSession()
	if sess == nil {
		return zero, fmt.Errorf("MCP %s 会话不可用", c.cfg.Name)
	}
	return fn(sess)
}

// getSession 持锁读取当前会话（并发安全：重连可能置 nil）。
func (c *mcpConnection) getSession() *mcp.ClientSession {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session
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
	c.touch()
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

// sanitizeToolName 把工具名中的非法字符替换为下划线。
// OpenAI/DeepSeek API 要求工具名匹配 ^[a-zA-Z0-9_-]+$，中文/点号/空格等需过滤。
func sanitizeToolName(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// registerClientTools 把连接的工具注册进 Registry，名加 "mcp__<server>__" 前缀防冲突。返回工具数。
// MCP 工具 RequiresApproval=true（外部工具默认需审批）；细粒度 HITL 由 Registry.BeforeTool 钩子
// 统一处理（阶段一已加钩子链），调用方可在 BeforeTool 中按 "mcp__<server>__<tool>" 前缀做白名单。
// 用下划线而非点号：OpenAI/DeepSeek API 要求工具名匹配 ^[a-zA-Z0-9_-]+$，点号不被接受。
// 会对 serverName 和 td.Name 做 sanitize 过滤非法字符。
func registerClientTools(r *Registry, conn *mcpConnection) (int, error) {
	// ★ 2026-09-15：工具列表拉取加超时（原 context.Background() 无超时——
	//   服务器连上但不响应 ListTools 时会卡死整个会话启动）。
	ctx, cancel := context.WithTimeout(context.Background(), mcpDialTimeout)
	defer cancel()
	tools, err := conn.listAllTools(ctx)
	if err != nil {
		return 0, err
	}
	serverName := sanitizeToolName(conn.cfg.Name)
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
		toolName := td.Name                  // 原始名，传给 MCP 服务器用（不 sanitize）
		regName := sanitizeToolName(td.Name) // 注册到 Registry 的名，必须符合 ^[a-zA-Z0-9_-]+$
		toolDesc := td.Description
		r.Register(&Tool{
			Name:             "mcp__" + serverName + "__" + regName,
			Description:      "[MCP:" + conn.cfg.Name + "] " + toolDesc,
			Parameters:       schema,
			RequiresApproval: true,
			Handler: func(ctx context.Context, args map[string]any) (string, error) {
				return conn.callTool(ctx, toolName, args)
			},
		})
	}
	return len(tools), nil
}

// ─── 连接池（★ 2026-09-15：跨会话复用 + 并发合流 + 空闲回收）───

const (
	mcpDialTimeout  = 15 * time.Second // 单台连接建立超时（含 initialize 握手与工具拉取）
	mcpIdleTTL      = 10 * time.Minute // 空闲连接回收阈值（PAIR_MCP_IDLE_TTL_SEC 可覆盖）
	mcpReapInterval = time.Minute      // 空闲回收扫描周期
	mcpFailCooldown = 30 * time.Second // 连接失败负缓存（冷却期内不重试）
)

var (
	mcpPoolMu    sync.Mutex
	mcpPool      = map[string]*mcpConnection{} // key=配置指纹 → 已建立连接（跨会话复用）
	mcpSlots     = map[string]*mcpPoolSlot{}   // 连接中的占位（同 key 并发只连一次）
	mcpFailUntil = map[string]time.Time{}      // 负缓存：key → 冷却截止时间
	mcpReapOnce  sync.Once
)

// mcpPoolSlot 连接占位：同配置并发 acquire 时只执行一次连接（once），
// 其余调用等待并共享结果（防惊群重复 spawn 子进程）。
type mcpPoolSlot struct {
	once sync.Once
	conn *mcpConnection
	err  error
}

// mcpTestTransportFactory 测试注入钩子：非 nil 时 connect 用它构建 transport
// （如 InMemoryTransport），不走子进程。生产恒为 nil。
var mcpTestTransportFactory func(cfg MCPServerConfig) mcp.Transport

// mcpFingerprint 配置指纹（连接复用 key）：名字+命令+参数+环境变量。
// 不同工作区同名不同参的服务器 → 不同指纹 → 独立连接（安全）。
func mcpFingerprint(cfg MCPServerConfig) string {
	var sb strings.Builder
	sb.WriteString(cfg.Name)
	sb.WriteByte(0)
	sb.WriteString(cfg.Command)
	sb.WriteByte(0)
	sb.WriteString(strings.Join(cfg.Args, "\x01"))
	sb.WriteByte(0)
	if len(cfg.Env) > 0 {
		keys := make([]string, 0, len(cfg.Env))
		for k := range cfg.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(cfg.Env[k])
			sb.WriteByte(1)
		}
	}
	return sb.String()
}

// mcpIdleTTLFromEnv 空闲回收阈值（PAIR_MCP_IDLE_TTL_SEC，单位秒；缺省 10 分钟；
// <=0 表示禁用空闲回收——连接常驻，交给进程退出清理）。
func mcpIdleTTLFromEnv() time.Duration {
	if v := os.Getenv("PAIR_MCP_IDLE_TTL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return mcpIdleTTL
}

// ensureMCPReaper 惰性启动空闲回收循环（首次注册 MCP 时启动，进程内只一次）。
func ensureMCPReaper() {
	mcpReapOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(mcpReapIntervalFromEnv())
			defer ticker.Stop()
			for range ticker.C {
				reapMCPIdleOnce(mcpIdleTTLFromEnv())
			}
		}()
	})
}

// mcpReapIntervalFromEnv 空闲回收扫描周期（PAIR_MCP_REAP_SEC，单位秒；最小 1s；
// 缺省 60s。缩短用于测试验证或运维调优）。
func mcpReapIntervalFromEnv() time.Duration {
	if v := os.Getenv("PAIR_MCP_REAP_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			return time.Duration(n) * time.Second
		}
	}
	return mcpReapInterval
}

// reapMCPIdleOnce 扫描池，关闭空闲超时的底层连接（保留连接对象，下次使用自动重连）。
func reapMCPIdleOnce(ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	now := time.Now().UnixNano()
	limit := int64(ttl)
	mcpPoolMu.Lock()
	var idle []*mcpConnection
	for _, conn := range mcpPool {
		if now-conn.lastUsed.Load() > limit {
			idle = append(idle, conn)
		}
	}
	mcpPoolMu.Unlock()
	for _, conn := range idle {
		if conn.closeIdle() {
			log.Printf("[MCP] %s 空闲超过 %v，已回收子进程（连接对象保留，下次使用自动重连）", conn.cfg.Name, ttl)
		}
	}
}

// acquireMCPConnection 获取（或建立）配置对应的连接（并发安全、同 key 合流）。
// 命中缓存直接复用；未命中则按 mcpDialTimeout 连接；失败记冷却（期内不重试）。
func acquireMCPConnection(cfg MCPServerConfig) (*mcpConnection, error) {
	key := mcpFingerprint(cfg)

	// 1) 命中已建立连接 / 失败冷却
	mcpPoolMu.Lock()
	if conn, ok := mcpPool[key]; ok {
		mcpPoolMu.Unlock()
		conn.touch()
		return conn, nil
	}
	if until, ok := mcpFailUntil[key]; ok && time.Now().Before(until) {
		mcpPoolMu.Unlock()
		return nil, fmt.Errorf("MCP %s 上次连接失败，%v 冷却期内不重试", cfg.Name, mcpFailCooldown)
	}
	mcpPoolMu.Unlock()

	// 2) 未命中：取（或建）同 key 占位——同配置并发只连一次
	mcpPoolMu.Lock()
	slot, ok := mcpSlots[key]
	if !ok {
		slot = &mcpPoolSlot{}
		mcpSlots[key] = slot
	}
	mcpPoolMu.Unlock()

	slot.once.Do(func() {
		slot.conn, slot.err = connectMCP(cfg)
		if slot.err == nil {
			// 仅在真正新建时打一次（并发合流时多个等待者共享此结果，不重复打）
			log.Printf("[MCP] %s 新连接已建立（跨会话复用）", cfg.Name)
		}
	})

	if slot.err != nil {
		mcpPoolMu.Lock()
		if old, ok := mcpSlots[key]; ok && old == slot {
			delete(mcpSlots, key)
		}
		mcpFailUntil[key] = time.Now().Add(mcpFailCooldown)
		mcpPoolMu.Unlock()
		return nil, slot.err
	}

	// 3) 成功：入池（竞态兜底：若池中已有更新的连接，回收本次多余连接）
	mcpPoolMu.Lock()
	if old, ok := mcpSlots[key]; ok && old == slot {
		delete(mcpSlots, key)
	}
	if existing, ok := mcpPool[key]; ok && existing != slot.conn {
		mcpPoolMu.Unlock()
		_ = slot.conn.close()
		existing.touch()
		return existing, nil
	}
	mcpPool[key] = slot.conn
	delete(mcpFailUntil, key)
	mcpPoolMu.Unlock()
	slot.conn.touch()
	return slot.conn, nil
}

// CloseAllMCPConnections 关闭池中全部连接并清空池（进程退出钩子调用），
// 避免 MCP 子进程孤儿化残留。
func CloseAllMCPConnections() {
	mcpPoolMu.Lock()
	conns := make([]*mcpConnection, 0, len(mcpPool))
	for _, conn := range mcpPool {
		conns = append(conns, conn)
	}
	mcpPool = map[string]*mcpConnection{}
	mcpSlots = map[string]*mcpPoolSlot{}
	mcpFailUntil = map[string]time.Time{}
	mcpPoolMu.Unlock()
	for _, conn := range conns {
		_ = conn.close()
	}
	if len(conns) > 0 {
		log.Printf("[MCP] 已关闭全部 %d 个 MCP 连接（进程退出清理）", len(conns))
	}
}

// connectMCP 启动并初始化一个 MCP 服务器连接（带超时，防卡住启动）。
func connectMCP(cfg MCPServerConfig) (*mcpConnection, error) {
	conn := newMCPConnection(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), mcpDialTimeout)
	defer cancel()
	if err := conn.connect(ctx); err != nil {
		return nil, err
	}
	return conn, nil
}

// RegisterMCPServers 连接每个配置的 MCP 服务器并注册其工具；起不来的跳过（不阻断 agent）。
// 返回注册的工具总数。
//
// ★ 2026-09-15 重写：
//   - 并行连接：全部服务器并发建立（原逐台串行——N 台 × 15s 超时叠加卡住会话启动）；
//   - 连接复用：经连接池 acquire（同配置跨会话复用，不重复 spawn 子进程）；
//   - 全程可见日志：每台失败有明确日志（原静默 continue 不可诊断），结束输出汇总。
func RegisterMCPServers(r *Registry, configs []MCPServerConfig) int {
	if len(configs) == 0 {
		return 0
	}
	ensureMCPReaper()
	start := time.Now()
	type mcpRegResult struct {
		tools int
		err   error
	}
	results := make([]mcpRegResult, len(configs))
	var wg sync.WaitGroup
	for i, cfg := range configs {
		wg.Add(1)
		go func(i int, cfg MCPServerConfig) {
			defer wg.Done()
			conn, err := acquireMCPConnection(cfg)
			if err != nil {
				results[i] = mcpRegResult{err: err}
				return
			}
			n, err := registerClientTools(r, conn)
			if err != nil {
				results[i] = mcpRegResult{err: err}
				return
			}
			results[i] = mcpRegResult{tools: n}
		}(i, cfg)
	}
	wg.Wait()
	total, failed := 0, 0
	for i, res := range results {
		if res.err != nil {
			failed++
			log.Printf("[MCP] %s 未就绪（已跳过）: %v", configs[i].Name, res.err)
			continue
		}
		total += res.tools
	}
	log.Printf("[MCP] 注册完成：%d/%d 台就绪，共 %d 个工具（耗时 %v）",
		len(configs)-failed, len(configs), total, time.Since(start).Round(time.Millisecond))
	return total
}
