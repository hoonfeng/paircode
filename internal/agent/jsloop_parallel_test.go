package agent

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ─── 并行会话测试（JS 循环外置后的并发退化回归）──
//
// 背景（2026-08-30 实测 BUG）：agentloop 核心外置到 JS 插件后，Loop.Run 的整个
// 循环在 `impl.plugin.withLock`（插件 VM 执行锁）内执行——LLM 调用、工具执行
// 全程持锁。于是「只要有一个对话在跑，新开对话就卡死」（前端 30s 超时）：
// 第二个会话的 CreateLoop / Run 都要等第一个会话跑完才能进 VM。
//
// 本测试用「首次 Chat 阻塞」的 Provider 占住会话 A，再要求会话 B 独立跑完，
// 从而把「并行会话是否被 VM 锁串行化」变成可回归的断言。

// blockingProvider 首次 Chat 进入后阻塞直到 release 关闭（模拟长 LLM 调用）。
type blockingProvider struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	calls   int
}

func (p *blockingProvider) Name() string { return "blocking-mock" }

func (p *blockingProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	p.mu.Lock()
	p.calls++
	first := p.calls == 1
	p.mu.Unlock()
	if first {
		p.once.Do(func() { close(p.entered) })
		select {
		case <-p.release:
		case <-ctx.Done():
			return Message{}, ctx.Err()
		}
	}
	msg := Message{Role: RoleAssistant, Content: "会话A完成"}
	if onChunk != nil {
		onChunk(Chunk{Content: msg.Content, Done: true})
	}
	return msg, nil
}

// TestJSLoopParallelSessionsNotSerialized 会话 A 卡在 LLM 调用中时，
// 会话 B 必须能独立跑完（不被 A 的 JS 循环 VM 锁挡住）。
func TestJSLoopParallelSessionsNotSerialized(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	dir := t.TempDir()
	regA := NewRegistry()
	RegisterDefaultTools(regA, dir)
	regB := NewRegistry()
	RegisterDefaultTools(regB, dir)

	pa := &blockingProvider{entered: make(chan struct{}), release: make(chan struct{})}
	loopA := &Loop{Provider: pa, Registry: regA, System: "并行测试-A", MaxIterations: 3,
		OnEvent: func(Event) {}}
	loopB := &Loop{Provider: &MockProvider{Responses: []Message{{Content: "会话B完成"}}},
		Registry: regB, System: "并行测试-B", MaxIterations: 3, OnEvent: func(Event) {}}

	doneA := make(chan error, 1)
	go func() {
		_, err := loopA.Run(context.Background(), "任务 A（长 LLM 调用）", nil)
		doneA <- err
	}()

	select {
	case <-pa.entered:
	case <-time.After(15 * time.Second):
		t.Fatal("会话 A 未进入 LLM 调用（JS 循环未启动）")
	}

	doneB := make(chan error, 1)
	go func() {
		_, err := loopB.Run(context.Background(), "任务 B（应立即完成）", nil)
		doneB <- err
	}()

	select {
	case err := <-doneB:
		if err != nil {
			close(pa.release)
			<-doneA
			t.Fatalf("会话 B 执行失败: %v", err)
		}
	case <-time.After(8 * time.Second):
		close(pa.release)
		<-doneA
		t.Fatal("并行会话被串行化：会话 A 的 LLM 调用未结束时会话 B 无法运行（JS 循环插件 VM 锁独占）")
	}

	close(pa.release)
	if err := <-doneA; err != nil {
		t.Fatalf("会话 A 执行失败: %v", err)
	}
}

// TestSessionManagerParallelStartNotBlocked 复现用户实测症状：
// 「只要有在进行的对话，开启新对话直接 30 秒超时」。
// 根因两层：① JS 循环单实例 → 新会话 CreateLoop/Run 等 VM 锁；
//
//	② SessionManager.Start 全程持 m.mu 写锁 → 卡在 ① 时所有读方法（状态/落盘/
//	   历史）连带阻塞 → 前端任何请求 30s 超时。
//
// 断言：会话 A 的 LLM 调用未结束时，会话 B 能启动，且管理器读方法立即响应。
func TestSessionManagerParallelStartNotBlocked(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t) // 同时注册 loop 装配器（CreateLoop 走 JS 装配 → 需主实例 VM 锁）

	dir := t.TempDir()
	m := NewSessionManager()
	pa := &blockingProvider{entered: make(chan struct{}), release: make(chan struct{})}

	regA := NewRegistry()
	RegisterDefaultTools(regA, dir)
	optsA := LoopOpts{Provider: pa, Registry: regA, System: "并行会话-A",
		MaxIterations: 3, WorkspaceRoot: dir}
	if err := m.Start(context.Background(), "convA", "任务 A（长 LLM 调用）", optsA); err != nil {
		t.Fatalf("会话 A Start 失败: %v", err)
	}
	defer func() {
		select {
		case <-pa.release:
		default:
			close(pa.release)
		}
	}()

	select {
	case <-pa.entered:
	case <-time.After(15 * time.Second):
		t.Fatal("会话 A 未进入 LLM 调用")
	}

	// ① 读方法必须立即响应（旧实现：Start 持写锁 → RLock 全部排队）
	readDone := make(chan bool, 1)
	go func() {
		running := m.IsRunning("convA")
		_ = m.GetStatus("convA")
		_ = m.ListRunning()
		readDone <- running
	}()
	select {
	case running := <-readDone:
		if !running {
			t.Error("会话 A 应处于运行中")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SessionManager 读方法被阻塞（Start 持写锁做重活）")
	}

	// ② 新会话必须能在 A 运行期间启动（旧实现：CreateLoop 等 VM 锁 → 30s 超时）
	regB := NewRegistry()
	RegisterDefaultTools(regB, dir)
	optsB := LoopOpts{Provider: &MockProvider{Responses: []Message{{Content: "会话B完成"}}},
		Registry: regB, System: "并行会话-B", MaxIterations: 3, WorkspaceRoot: dir}
	startB := make(chan error, 1)
	go func() {
		startB <- m.Start(context.Background(), "convB", "任务 B（应立即启动）", optsB)
	}()
	select {
	case err := <-startB:
		if err != nil {
			t.Fatalf("会话 B Start 失败: %v", err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("会话 B Start 被阻塞：并行会话仍被串行化")
	}

	// 会话 B 应能独立跑完（不等 A）
	deadline := time.Now().Add(8 * time.Second)
	for m.IsRunning("convB") && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if m.IsRunning("convB") {
		t.Fatal("会话 B 未能在 A 阻塞期间跑完（循环实例仍互相阻塞）")
	}
	if !m.IsRunning("convA") {
		t.Error("会话 A 应仍在运行（LLM 未返回）")
	}

	close(pa.release)
	deadline = time.Now().Add(15 * time.Second)
	for m.IsRunning("convA") && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if m.IsRunning("convA") {
		t.Error("会话 A 未在 LLM 返回后结束")
	}
}
