// mcp_pool_test.go — MCP 连接池测试（★ 2026-09-15 产品修复：跨会话复用/并行/回收/清理）。
//
// 覆盖点：
//  1. 配置指纹：同配置一致 / 异参数不同（连接复用 key 正确性）；
//  2. 跨会话复用：同配置二次注册不重建连接（核心语义——原实现每会话全量重建）；
//  3. 并行注册：N 台并行连接（耗时≈单台而非 N 倍叠加）；
//  4. 并发合流：同配置并发 acquire 只连接一次（防惊群重复 spawn）；
//  5. 失败负缓存：连接失败后冷却期内不重试；
//  6. 空闲回收：TTL 到期关闭子进程（保留对象），再次使用按需重连；
//  7. 退出清理：CloseAllMCPConnections 清空池且关闭全部会话。
//
// 测试注入 mcpTestTransportFactory（InMemoryTransport），不做子进程；dials 计数 = 连接建立次数。

package agent

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// resetMCPPoolForTest 清空连接池/占位/负缓存（测试隔离）。
func resetMCPPoolForTest() { CloseAllMCPConnections() }

// setupMCPPoolTest 重置池状态并注入可计数的测试 transport 工厂，返回 dials（连接新建次数）。
func setupMCPPoolTest(t *testing.T) *atomic.Int64 { return setupMCPPoolTestDelay(t, 0) }

// setupMCPPoolTestDelay 同上，但每次连接建立前 sleep delay（模拟慢启动，如 npx 拉包）。
func setupMCPPoolTestDelay(t *testing.T, delay time.Duration) *atomic.Int64 {
	t.Helper()
	resetMCPPoolForTest()
	var dials atomic.Int64
	oldFactory := mcpTestTransportFactory
	mcpTestTransportFactory = func(cfg MCPServerConfig) mcp.Transport {
		dials.Add(1)
		if delay > 0 {
			time.Sleep(delay)
		}
		clientT, serverT := mcp.NewInMemoryTransports()
		server := mcp.NewServer(&mcp.Implementation{Name: "pool-test", Version: "1.0"}, nil)
		server.AddTool(&mcp.Tool{Name: "ping", Description: "池测试工具", InputSchema: emptySchema()},
			func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "pong"}}}, nil
			})
		go func() { _ = server.Run(context.Background(), serverT) }()
		return clientT
	}
	t.Cleanup(func() {
		mcpTestTransportFactory = oldFactory
		resetMCPPoolForTest()
	})
	return &dials
}

// TestMCPPoolFingerprint 配置指纹：同名同参一致，任一字段不同即不同（防连接误共享）。
func TestMCPPoolFingerprint(t *testing.T) {
	a := MCPServerConfig{Name: "srv", Command: "node", Args: []string{"a.js"}}
	b := MCPServerConfig{Name: "srv", Command: "node", Args: []string{"a.js"}}
	c := MCPServerConfig{Name: "srv", Command: "node", Args: []string{"b.js"}}
	d := MCPServerConfig{Name: "srv2", Command: "node", Args: []string{"a.js"}}
	e := MCPServerConfig{Name: "srv", Command: "node", Args: []string{"a.js"}, Env: map[string]string{"K": "V"}}

	if mcpFingerprint(a) != mcpFingerprint(b) {
		t.Error("同配置指纹应一致（得以复用连接）")
	}
	if mcpFingerprint(a) == mcpFingerprint(c) {
		t.Error("不同参数指纹应不同（不得误共享连接）")
	}
	if mcpFingerprint(a) == mcpFingerprint(d) {
		t.Error("不同名字指纹应不同")
	}
	if mcpFingerprint(a) == mcpFingerprint(e) {
		t.Error("不同环境变量指纹应不同")
	}
}

// TestMCPPoolReuseAcrossSessions 跨会话复用（核心）：两个会话（新 Registry）注册同一配置，
// 第二次注册不得重建连接（dials 保持 1），且工具在新会话可用。
func TestMCPPoolReuseAcrossSessions(t *testing.T) {
	dials := setupMCPPoolTest(t)
	cfg := MCPServerConfig{Name: "reuse", Command: "node"}

	reg1 := NewRegistry()
	if n := RegisterMCPServers(reg1, []MCPServerConfig{cfg}); n != 1 {
		t.Fatalf("第一次注册工具数 = %d，期望 1", n)
	}
	if got := dials.Load(); got != 1 {
		t.Fatalf("第一次注册后连接建立次数 = %d，期望 1", got)
	}

	// 模拟第二个会话：新 Registry + 同配置
	reg2 := NewRegistry()
	if n := RegisterMCPServers(reg2, []MCPServerConfig{cfg}); n != 1 {
		t.Fatalf("第二次注册工具数 = %d，期望 1", n)
	}
	if got := dials.Load(); got != 1 {
		t.Fatalf("★ 跨会话复用失败：第二次注册后连接建立次数 = %d，期望仍为 1（不重建）", got)
	}

	// 新会话注册的工具可正常调用（走复用连接的 session）
	tool, ok := reg2.Get("mcp__reuse__ping")
	if !ok {
		t.Fatal("mcp__reuse__ping 未注册进第二个会话 Registry")
	}
	res, err := tool.Handler(context.Background(), nil)
	if err != nil {
		t.Fatalf("复用连接调用工具: %v", err)
	}
	if res != "pong" {
		t.Errorf("工具结果 = %q，期望 'pong'", res)
	}
}

// TestMCPPoolParallelRegister 并行注册：3 台各 400ms 慢连接，总耗时≈400ms（并行），
// 而非 3×400ms=1200ms（串行叠加——原实现缺陷）。
func TestMCPPoolParallelRegister(t *testing.T) {
	const dialDelay = 400 * time.Millisecond
	dials := setupMCPPoolTestDelay(t, dialDelay)
	cfgs := []MCPServerConfig{
		{Name: "p1", Command: "node"},
		{Name: "p2", Command: "node"},
		{Name: "p3", Command: "node"},
	}

	reg := NewRegistry()
	start := time.Now()
	n := RegisterMCPServers(reg, cfgs)
	elapsed := time.Since(start)

	if n != 3 {
		t.Fatalf("工具总数 = %d，期望 3", n)
	}
	if got := dials.Load(); got != 3 {
		t.Fatalf("连接建立次数 = %d，期望 3", got)
	}
	// 串行需 ≥ 3×400ms = 1200ms；并行 ≈ 400ms。阈值 900ms 可稳定区分。
	if elapsed > 900*time.Millisecond {
		t.Errorf("★ 并行注册耗时 %v，超过 900ms（串行叠加嫌疑）", elapsed)
	}
	t.Logf("3 台并行注册耗时 %v（单台连接 %v）", elapsed.Round(time.Millisecond), dialDelay)
}

// TestMCPPoolConcurrentAcquire 并发合流（防惊群）：同配置 6 个并发 acquire，
// 只建立 1 次连接，全部拿到同一连接对象。
func TestMCPPoolConcurrentAcquire(t *testing.T) {
	const dialDelay = 500 * time.Millisecond
	dials := setupMCPPoolTestDelay(t, dialDelay)
	cfg := MCPServerConfig{Name: "conc", Command: "node"}

	const N = 6
	conns := make([]*mcpConnection, N)
	errs := make([]error, N)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start // 统一放行：全部先到「池 miss」再抢占位，最大化惊群压力
			conns[i], errs[i] = acquireMCPConnection(cfg)
		}(i)
	}
	close(start)
	wg.Wait()

	for i := 0; i < N; i++ {
		if errs[i] != nil {
			t.Fatalf("第 %d 个 acquire 失败: %v", i, errs[i])
		}
		if conns[i] != conns[0] {
			t.Errorf("第 %d 个 acquire 返回了不同连接对象（应合流共享）", i)
		}
	}
	if got := dials.Load(); got != 1 {
		t.Errorf("★ 并发合流失败：连接建立次数 = %d，期望 1（防惊群重复 spawn）", got)
	}
}

// TestMCPPoolFailCooldown 失败负缓存：连接失败后冷却期内直接返回冷却错误（不重复重试/等待）。
func TestMCPPoolFailCooldown(t *testing.T) {
	resetMCPPoolForTest()
	t.Cleanup(resetMCPPoolForTest)
	// 不注入工厂：走真实 exec，命令不存在 → 快速失败（无需等 15s 超时）
	cfg := MCPServerConfig{Name: "bad", Command: "definitely-not-exist-pair-test-30291"}

	if _, err := acquireMCPConnection(cfg); err == nil {
		t.Fatal("不存在的命令应连接失败")
	}
	// 第二次：冷却期内直接失败（负缓存命中）
	_, err2 := acquireMCPConnection(cfg)
	if err2 == nil {
		t.Fatal("冷却期内应直接失败")
	}
	if !strings.Contains(err2.Error(), "冷却期") {
		t.Errorf("冷却期内错误 = %v，期望包含「冷却期」提示", err2)
	}
	// 清除冷却（模拟冷却过期）→ 实际重试（同样失败，但不再是冷却错误）
	key := mcpFingerprint(cfg)
	mcpPoolMu.Lock()
	delete(mcpFailUntil, key)
	mcpPoolMu.Unlock()
	_, err3 := acquireMCPConnection(cfg)
	if err3 == nil || strings.Contains(err3.Error(), "冷却期") {
		t.Errorf("冷却清除后应实际重试（err = %v）", err3)
	}
}

// TestMCPPoolIdleReap 空闲回收：TTL 到期关闭底层会话（子进程回收），保留池中对象；
// 再次使用按需自动重连（dials 递增）。
func TestMCPPoolIdleReap(t *testing.T) {
	dials := setupMCPPoolTest(t)
	cfg := MCPServerConfig{Name: "idle", Command: "node"}

	conn, err := acquireMCPConnection(cfg)
	if err != nil {
		t.Fatalf("首次连接: %v", err)
	}
	if conn.getSession() == nil {
		t.Fatal("首次连接后会话应可用")
	}

	// 把 lastUsed 拨到 2 小时前 → 越过 TTL=1s
	conn.lastUsed.Store(time.Now().Add(-2 * time.Hour).UnixNano())
	reapMCPIdleOnce(time.Second)

	if conn.getSession() != nil {
		t.Error("空闲回收后会话应已关闭（子进程回收）")
	}
	mcpPoolMu.Lock()
	_, inPool := mcpPool[mcpFingerprint(cfg)]
	mcpPoolMu.Unlock()
	if !inPool {
		t.Error("空闲回收应保留池中连接对象（只关子进程，不删条目）")
	}
	if dials.Load() != 1 {
		t.Errorf("回收动作本身不应触发新连接，dials = %d", dials.Load())
	}

	// 再次 acquire：复用同一对象（尚未重连）→ 使用时刻按需自动重连
	conn2, err := acquireMCPConnection(cfg)
	if err != nil {
		t.Fatalf("回收后 acquire: %v", err)
	}
	if conn2 != conn {
		t.Error("回收后应复用同一连接对象")
	}
	if _, err := conn2.listAllTools(context.Background()); err != nil {
		t.Fatalf("回收后使用应自动重连: %v", err)
	}
	if dials.Load() != 2 {
		t.Errorf("按需重连后 dials = %d，期望 2", dials.Load())
	}
}

// TestMCPPoolCloseAll 退出清理：CloseAllMCPConnections 清空池/占位/负缓存，并关闭全部会话。
func TestMCPPoolCloseAll(t *testing.T) {
	dials := setupMCPPoolTest(t)
	cfg1 := MCPServerConfig{Name: "c1", Command: "node"}
	cfg2 := MCPServerConfig{Name: "c2", Command: "node"}

	conn1, err := acquireMCPConnection(cfg1)
	if err != nil {
		t.Fatalf("acquire c1: %v", err)
	}
	conn2, err := acquireMCPConnection(cfg2)
	if err != nil {
		t.Fatalf("acquire c2: %v", err)
	}
	if dials.Load() != 2 {
		t.Fatalf("前置连接数 = %d，期望 2", dials.Load())
	}

	CloseAllMCPConnections()

	mcpPoolMu.Lock()
	nPool, nSlots, nFail := len(mcpPool), len(mcpSlots), len(mcpFailUntil)
	mcpPoolMu.Unlock()
	if nPool != 0 || nSlots != 0 || nFail != 0 {
		t.Errorf("CloseAll 后池状态应清空，实际 pool=%d slots=%d fail=%d", nPool, nSlots, nFail)
	}
	if conn1.getSession() != nil || conn2.getSession() != nil {
		t.Error("CloseAll 后全部会话应已关闭（子进程回收）")
	}
}
