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

	"github.com/hoonfeng/paircode/internal/core"
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

// TestAdapterToolCallRoot 验证：插件 ctx 服务根解析优先级（2026-08-27 双上下文）——
// 当前工具调用会话根（会话上下文）＞ UI invoke 绑定主根（UI 上下文）＞ 插件装载快照。
// ★ 不再回落全局主根 primaryWorkspaceRoot：正在执行的对话（Loop 内非工具 JS
//   调用）必须保持装载/会话根，切换全局工作区不得带偏。
func TestAdapterToolCallRoot(t *testing.T) {
	pc := &PluginContext{WorkspaceRoot: `F:\mount-snapshot`}
	oldRoots := WorkspaceRoots
	defer func() { WorkspaceRoots = oldRoots }()
	ad := &jsPluginAdapter{}

	// 1) 无任何绑定：回落装载快照（全局主根存在时也不介入——防对话中切工作区带偏）
	WorkspaceRoots = []string{`F:\ws-now`}
	if r := ad.ctxServiceRoot(pc); r != `F:\mount-snapshot` {
		t.Fatalf("无绑定应回落装载快照（不被全局主根带偏）, got %q", r)
	}

	// 2) UI invoke 绑定：跟随当前主根（ui-quick-exec 切工作区后命令跟随新工作区）
	ad.uiWsRoot = `F:\ws-now`
	if r := ad.ctxServiceRoot(pc); r != `F:\ws-now` {
		t.Fatalf("UI 绑定应取主根, got %q", r)
	}

	// 3) 工具调用绑定：会话根优先（即使全局已切走，运行中对话仍用旧工作区）
	ad.setToolCallRoot(`F:\ws-x`)
	if r := ad.ctxServiceRoot(pc); r != `F:\ws-x` {
		t.Fatalf("有工具绑定应取会话根, got %q", r)
	}
	// 清除工具绑定后回落 UI 根
	ad.setToolCallRoot("")
	if r := ad.ctxServiceRoot(pc); r != `F:\ws-now` {
		t.Fatalf("清除工具绑定应回落 UI 根, got %q", r)
	}
	// 清除 UI 绑定后回落装载快照
	ad.uiWsRoot = ""
	if r := ad.ctxServiceRoot(pc); r != `F:\mount-snapshot` {
		t.Fatalf("清除 UI 绑定应回落装载快照, got %q", r)
	}
}

// TestClientInvokeUIFollowsMainRoot 验证：client 半 invoke（registerClientMethod）
// 执行期间绑定「invoke 发起时刻的当前主根」→ ctx.fs 等基础服务按它解析路径，
// 每次 invoke 重新取（切换工作区即生效）；工具执行（harness 注册）不绑定 UI 根。
func TestClientInvokeUIFollowsMainRoot(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `F:\mount-snapshot`)
	oldRoots := WorkspaceRoots
	defer func() { WorkspaceRoots = oldRoots }()
	dirA := t.TempDir()
	dirB := t.TempDir()
	// 探针文件放主根（WorkspaceRoots[0]）——tempdir 本身即主根
	if err := os.WriteFile(filepath.Join(dirA, ".wsprobe"), []byte("PROBE-A"), 0o644); err != nil {
		t.Fatalf("write probe A: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirB, ".wsprobe"), []byte("PROBE-B"), 0o644); err != nil {
		t.Fatalf("write probe B: %v", err)
	}

	const code = `
return {
  name: 'ui-root-probe',
  inject: ['fs'],
  apply(ctx) {
    ctx.registerClientMethod('readProbe', () => ctx.fs.readFile('.wsprobe'))
    ctx.registerClientMethod('getWS', () => ctx.app.workspaceRoot)
  }
}`
	id, err := host.DefineJS(code, "ui root probe")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	defer host.Unload("ui-root-probe")

	// 主根=A：invoke 读探针 → PROBE-A（fs 按 UI 绑定的主根解析）
	WorkspaceRoots = []string{dirA}
	got, err := host.InvokeClientMethod("ui-root-probe", "readProbe", nil)
	if err != nil {
		t.Fatalf("invoke readProbe(A): %v", err)
	}
	if got != "PROBE-A" {
		t.Fatalf("主根 A 时应读 A 探针, got %v", got)
	}
	// workspaceRoot 为实时主根 A
	gotWS, _ := host.InvokeClientMethod("ui-root-probe", "getWS", nil)
	if gotWS != dirA {
		t.Fatalf("app.workspaceRoot 应为主根 A, got %v", gotWS)
	}

	// 切换主根=B：再 invoke → PROBE-B（每次 invoke 重新绑定，切换即生效）
	WorkspaceRoots = []string{dirB}
	got, err = host.InvokeClientMethod("ui-root-probe", "readProbe", nil)
	if err != nil {
		t.Fatalf("invoke readProbe(B): %v", err)
	}
	if got != "PROBE-B" {
		t.Fatalf("主根 B 时应读 B 探针, got %v", got)
	}
}

// TestAppWorkspaceRootLive 验证：ctx.app.workspaceRoot 为实时主根 accessor——
// 插件装载后切换工作区（WorkspaceRoots 变化），读取返回新根而非装载快照
// （ui-quick-exec 菜单工作区名不变的根因回归守卫）。
func TestAppWorkspaceRootLive(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `F:\mount-snapshot`)
	oldRoots := WorkspaceRoots
	defer func() { WorkspaceRoots = oldRoots }()

	const code = `
return {
  name: 'app-ws-probe',
  apply(ctx) {
    ctx.registerClientMethod('getWS', () => ctx.app.workspaceRoot)
    ctx.registerClientMethod('getRoot', () => ctx.app.root)
  }
}`
	id, err := host.DefineJS(code, "app ws probe")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	defer host.Unload("app-ws-probe")

	// 主机根（装载时）
	WorkspaceRoots = []string{`F:\mount-snapshot`}
	got, err := host.InvokeClientMethod("app-ws-probe", "getWS", nil)
	if err != nil {
		t.Fatalf("invoke getWS: %v", err)
	}
	if got != `F:\mount-snapshot` {
		t.Fatalf("装载时应返回主根, got %v", got)
	}

	// 切换工作区：WorkspaceRoots 更新 → accessor 实时返回新根（修复前为快照）
	WorkspaceRoots = []string{`F:\ws-switched`}
	got, err = host.InvokeClientMethod("app-ws-probe", "getWS", nil)
	if err != nil {
		t.Fatalf("invoke getWS(切换后): %v", err)
	}
	if got != `F:\ws-switched` {
		t.Fatalf("切换后应实时返回新主根, got %v", got)
	}
	// root accessor（core.Root()）语义不变
	if r := core.Root(); r != "" {
		t.Logf("core.Root()=%s（本测试未动 core.Folders，主根为测试环境原值）", r)
	}
}

// TestToolExecRootNotAffectedBySwitch 验证：对话工具 execute 绑定会话根
// （callWsRoot）——执行期间全局主根切换（模拟对话中切换工作区），工具内
// ctx.fs 仍按会话根解析路径，不回落到全局主根/装载根（2026-08-27 双上下文守卫）。
func TestToolExecRootNotAffectedBySwitch(t *testing.T) {
	dirLoad := t.TempDir() // 插件装载根（host.NewPluginHost root）
	dirWS := t.TempDir()   // 会话根（对话绑定）
	dirNow := t.TempDir()  // 切换后的全局主根
	for _, d := range []string{dirLoad, dirWS, dirNow} {
		if err := os.WriteFile(filepath.Join(d, ".wsprobe"), []byte("PROBE-"+filepath.Base(d)), 0o644); err != nil {
			t.Fatalf("write probe: %v", err)
		}
	}
	oldRoots := WorkspaceRoots
	defer func() { WorkspaceRoots = oldRoots }()

	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dirLoad)
	id, err := host.DefineJS(`
	return {
		name: 'tool-root-probe',
		inject: ['fs'],
		apply(ctx) {
			ctx.tools.register({
				name: 'probe',
				description: '读本根探针',
				parameters: { type: 'object', properties: {}, required: [] },
				execute: () => ctx.fs.readFile('.wsprobe'),
			})
		},
	}`, "tool root probe")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}

	// 模拟「对话进行中切换工作区」：全局主根已切到 dirNow
	WorkspaceRoots = []string{dirNow}
	// 对话会话根 = dirWS（Execute ctx 注入）
	ctx := WithSessionWorkspaceRoot(context.Background(), dirWS)
	out, err := reg.Execute(ctx, "probe", `{}`)
	if err != nil {
		t.Fatalf("execute probe: %v", err)
	}
	if out != "PROBE-"+filepath.Base(dirWS) {
		t.Fatalf("工具应读会话根探针（全局主根切换不得带偏）, got %q", out)
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
