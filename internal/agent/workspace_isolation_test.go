package agent

// workspace_isolation_test.go — 工作区隔离（2026-08-23 重大 BUG 修复）行为验证。
//
// 背景：前端切换工作区时，正在执行的对话（绑定旧工作区）的工具/持久化此前
// 使用全局当前工作区（core.Folders/WorkspaceRoots/Store()），导致工具在错误
// 工作区执行、消息落进错误工作区存储。修复后：
//   1. 会话 ctx 链注入会话根（WithSessionWorkspaceRoot）
//   2. 插件工具执行期间绑定 callWsRoot → ctx 基础服务按它解析路径
//   3. SessionManager 按工作区根缓存 store/DB，切换不关闭旧句柄

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestSessionWorkspaceRootCtx 验证：会话根经 ctx 链注入/提取（工具执行读根的通道）。
func TestSessionWorkspaceRootCtx(t *testing.T) {
	ctx := context.Background()
	if r := SessionWorkspaceRoot(ctx); r != "" {
		t.Fatalf("无注入时应为空, got %q", r)
	}
	ctx = WithSessionWorkspaceRoot(ctx, `F:\ws-x`)
	if r := SessionWorkspaceRoot(ctx); r != `F:\ws-x` {
		t.Fatalf("注入后应取回, got %q", r)
	}
	// 链式（WithSessionConvID 后值不丢）
	ctx2 := WithSessionConvID(ctx, "conv_1")
	if r := SessionWorkspaceRoot(ctx2); r != `F:\ws-x` {
		t.Fatalf("链式后会话根应保留, got %q", r)
	}
	if c := SessionConvID(ctx2); c != "conv_1" {
		t.Fatalf("convID 应保留, got %q", c)
	}
	// nil ctx
	if r := SessionWorkspaceRoot(nil); r != "" {
		t.Fatalf("nil ctx 应返回空, got %q", r)
	}
}

// TestAdapterToolCallRoot 验证：插件 ctx 服务根解析优先级——
// 当前工具调用会话根 ＞ 插件装载快照（串台修复的本体）。
func TestAdapterToolCallRoot(t *testing.T) {
	pc := &PluginContext{WorkspaceRoot: `F:\mount-snapshot`}

	// 无工具调用绑定：回落装载快照（保持旧行为）
	ad := &jsPluginAdapter{}
	if r := ad.ctxServiceRoot(pc); r != `F:\mount-snapshot` {
		t.Fatalf("无绑定应回落装载快照, got %q", r)
	}

	// 有工具调用绑定：会话根优先（即使全局已切走，运行中对话仍用旧工作区）
	ad.setToolCallRoot(`F:\ws-x`)
	if r := ad.ctxServiceRoot(pc); r != `F:\ws-x` {
		t.Fatalf("有绑定应取会话根, got %q", r)
	}
	// 清除绑定后回落快照
	ad.setToolCallRoot("")
	if r := ad.ctxServiceRoot(pc); r != `F:\mount-snapshot` {
		t.Fatalf("清除绑定应回落装载快照, got %q", r)
	}
}

// TestStoreForIsolation 验证：多工作区 store 缓存——SetWorkspaceRoot 切换后
// 旧工作区句柄保留（运行中会话隔离），CloseWorkspaceDB 显式关闭。
func TestStoreForIsolation(t *testing.T) {
	dirX := t.TempDir()
	dirY := t.TempDir()
	m := NewSessionManager()
	defer func() {
		m.CloseWorkspaceDB(dirX)
		m.CloseWorkspaceDB(dirY)
	}()

	m.SetWorkspaceRoot(dirX)
	if m.StoreFor(dirX) == nil {
		t.Fatalf("dirX store 应为空? 初始化后应存在")
	}
	// 切换到 Y
	m.SetWorkspaceRoot(dirY)
	// 旧根句柄保留
	if m.StoreFor(dirX) == nil {
		t.Fatalf("切换后 dirX store 应保留（运行中会话持久化仍走 X）")
	}
	if m.RawDBFor(dirX) == nil {
		t.Fatalf("切换后 dirX DB 应保留（codegraph 运行中会话仍走 X）")
	}
	// 新根可用
	if m.RawDBFor(dirY) == nil {
		t.Fatalf("dirY DB 应已打开")
	}
	// 消息落盘隔离：向 X 写一条用户消息 → 只出现在 X 的 store
	if err := m.AppendPersistedUserMessageTo(dirX, "conv_iso", "hello-x"); err != nil {
		t.Fatalf("写入 X 失败: %v", err)
	}
	msgsX, err := m.StoreFor(dirX).LoadAll("conv_iso")
	if err != nil || len(msgsX) == 0 {
		t.Fatalf("X 应读到消息, len=%d err=%v", len(msgsX), err)
	}
	msgsY, err := m.StoreFor(dirY).LoadAll("conv_iso")
	if err != nil {
		t.Fatalf("Y 读取失败: %v", err)
	}
	if len(msgsY) != 0 {
		t.Fatalf("Y 不应读到 X 的消息（串台）got %d", len(msgsY))
	}
	// CloseWorkspaceDB 后句柄释放（缓存丢弃；StoreFor 惰性重建是设计允许——后续
	// 意外访问旧根会自动恢复，但不会在删除瞬间阻止目录删除）
	m.CloseWorkspaceDB(dirX)
	m.wsMu.Lock()
	_, cachedX := m.stores[dirX]
	m.wsMu.Unlock()
	if cachedX {
		t.Fatalf("CloseWorkspaceDB 后 dirX store 缓存应丢弃")
	}
	if err := os.RemoveAll(dirX); err != nil {
		t.Fatalf("CloseWorkspaceDB 后 dirX 应可删除: %v", err)
	}
	_ = filepath.Join(dirX, "unused")
}
