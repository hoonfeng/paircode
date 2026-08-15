// plugin_e2e_test.go — 插件管理 API 配套：client 半、host/client 事件桥、Reload、inspect_query。
package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestJSPluginClientHalf define 带 client 半：语法预检 + 存储 + InspectDetail 返回。
func TestJSPluginClientHalf(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, `C:\ws`)
	clientCode := `(ui) => { ui.registerPanel({ id: 'p', title: 'P', render(el) { el.textContent = 'hi' } }); }`
	id, err := host.DefineJSCodeFull(`
  return { name: 'client-half-demo', apply(ctx) { return; } };
`, "", "测试 client 半", "", clientCode)
	if err != nil {
		t.Fatalf("DefineJSCodeFull: %v", err)
	}
	if !strings.HasPrefix(id, "dyn-") {
		t.Fatalf("id 应为 dyn-*: %s", id)
	}
	if err := host.LoadJSDynamic(mustDef(t, host, id)); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	rec := host.InspectDetail("client-half-demo")
	if rec == nil {
		t.Fatal("InspectDetail 应为 nil")
	}
	if !rec.HasClient {
		t.Fatal("HasClient 应为 true")
	}
	if rec.ClientCode != clientCode {
		t.Fatalf("ClientCode 不一致:\n got %q\nwant %q", rec.ClientCode, clientCode)
	}
	if rec.DefID != id {
		t.Fatalf("DefID = %s, want %s", rec.DefID, id)
	}
	// 列表也要有
	recs := host.Inspect()
	found := false
	for _, r := range recs {
		if r.Name == "client-half-demo" && r.HasClient {
			found = true
		}
	}
	if !found {
		t.Fatal("Inspect 列表应含 client-half-demo(hasClient)")
	}
	_ = host.Undefine("client-half-demo")
}

// TestJSPluginClientHalfSyntaxError client 半语法错误应被定义期拦截。
func TestJSPluginClientHalfSyntaxError(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	_, err := host.DefineJSCodeFull(`return { name: 'x', apply(ctx){} };`, "", "", "", `(ui) => { ui.`)
	if err == nil {
		t.Fatal("client 半语法错误应报错")
	}
	if !strings.Contains(err.Error(), "client") {
		t.Fatalf("错误应提及 client 半: %v", err)
	}
}

// TestClientEventBridge host/client 事件桥闭环：
//   - 插件 ctx.emit('ui:xxx') → ClientEventsSince 队列收到
//   - EmitHostEvent('host:xxx') → 插件 ctx.on 响应 → 再次 emit → 队列追加
func TestClientEventBridge(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	_, err := host.DefineJSCodeFull(`
  return {
    name: 'bridge-demo',
    apply(ctx) {
      ctx.on('host:ping', (p) => { ctx.emit('ui:pong', { echo: p }); });
    }
  };
`, "", "", "", `(ui) => {}`)
	if err != nil {
		t.Fatalf("define: %v", err)
	}
	// name 由 LoadJSDynamic 解析，这里直接用列表里的 def
	defs := host.JSDefs()
	if len(defs) != 1 {
		t.Fatalf("应有 1 个定义, got %d", len(defs))
	}
	if err := host.LoadJSDynamic(defs[0]); err != nil {
		t.Fatalf("load: %v", err)
	}

	// 1. host→client：浏览器轮询拿到 ui: 事件
	host.PushClientEvent("ui:notice", map[string]any{"n": 1})
	evs, last := host.ClientEventsSince(0)
	if len(evs) != 1 || evs[0].Name != "ui:notice" || last != 1 {
		t.Fatalf("client 事件队列异常: %+v last=%d", evs, last)
	}

	// 2. client→host→插件→host：host:ping 触发插件回调 emit ui:pong
	host.EmitHostEvent("host:ping", map[string]any{"n": 2})
	// 事件回调同步执行（emit 内部直接调监听器），队列应立即可见
	evs2, last2 := host.ClientEventsSince(last)
	if len(evs2) != 1 || evs2[0].Name != "ui:pong" {
		t.Fatalf("桥接响应未入队: %+v", evs2)
	}
	if p, ok := evs2[0].Payload.(map[string]any); !ok || p["echo"] == nil {
		t.Fatalf("pong payload 异常: %+v", evs2[0].Payload)
	}
	if last2 != 2 {
		t.Fatalf("lastSeq = %d, want 2", last2)
	}
	_ = host.Undefine("bridge-demo")
}

// TestPluginReload 停止后重新装载不报同名冲突（REST start 依赖）。
func TestPluginReload(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	id, err := host.DefineJSCodeFull(`
  return { name: 'reload-demo', apply(ctx) { ctx.provide('reloadDemo', true); } };
`, "", "", "", "")
	if err != nil {
		t.Fatalf("define: %v", err)
	}
	if err := host.LoadJSDynamic(mustDef(t, host, id)); err != nil {
		t.Fatalf("load: %v", err)
	}
	// 停止（Unload）
	if err := host.Unload("reload-demo"); err != nil {
		t.Fatal(err)
	}
	if host.State("reload-demo") != PluginStopped {
		t.Fatal("应 stopped")
	}
	// Reload（start）不报同名冲突
	if err := host.Reload("reload-demo"); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if host.State("reload-demo") != PluginRunning {
		t.Fatal("应 running")
	}
	// 再次 stop + Reload（运行中重载）
	if err := host.Unload("reload-demo"); err != nil {
		t.Fatal(err)
	}
	if err := host.Reload("reload-demo"); err != nil {
		t.Fatalf("Reload(2nd): %v", err)
	}
	_ = host.Undefine("reload-demo")
}

// TestCordisInspectQuery cordis_inspect_query 精确查询协议。
func TestCordisInspectQuery(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	RegisterBuiltinPlugins(host)

	// service listService
	out, err := cordisInspectQuery(host, "host", "service", "listService", nil)
	if err != nil || !strings.Contains(out, "fs") || !strings.Contains(out, "bash") {
		t.Fatalf("listService: %v\n%s", err, out)
	}
	// service getService
	out, err = cordisInspectQuery(host, "host", "service", "getService", map[string]any{"name": "fs"})
	if err != nil || !strings.Contains(out, "readFile") {
		t.Fatalf("getService fs: %v\n%s", err, out)
	}
	// tool listTool（RegisterCordisTools 后才含 cordis_*）
	RegisterCordisTools(reg, host, `C:\ws`)
	out, err = cordisInspectQuery(host, "host", "tool", "listTool", nil)
	if err != nil || !strings.Contains(out, "cordis_inspect_query") {
		t.Fatalf("listTool: %v\n%s", err, out)
	}
	out, err = cordisInspectQuery(host, "host", "tool", "getTool", map[string]any{"name": "cordis_define"})
	if err != nil || !strings.Contains(out, "cordis_define") {
		t.Fatalf("getTool: %v\n%s", err, out)
	}
	// event listEvent
	out, err = cordisInspectQuery(host, "host", "event", "listEvent", nil)
	if err != nil || !strings.Contains(out, "Events") {
		t.Fatalf("listEvent: %v\n%s", err, out)
	}
	// plugin listPlugin
	out, err = cordisInspectQuery(host, "host", "plugin", "listPlugin", nil)
	if err != nil || !strings.Contains(out, "sysinfo") {
		t.Fatalf("listPlugin: %v\n%s", err, out)
	}
	out, err = cordisInspectQuery(host, "host", "plugin", "getPlugin", map[string]any{"name": "sysinfo"})
	if err != nil || !strings.Contains(out, "sysinfo") {
		t.Fatalf("getPlugin: %v\n%s", err, out)
	}
	// client 平台（无上报快照时给离线兜底摘要）
	out, err = cordisInspectQuery(host, "client", "plugin", "listPlugin", nil)
	if err != nil || !strings.Contains(out, "Client runtime") {
		t.Fatalf("client 平台: %v\n%s", err, out)
	}
	if !strings.Contains(out, "浏览器未连接") {
		t.Fatalf("未上报时应提示浏览器未连接: %s", out)
	}
	// 非法 provider/method 报错
	if _, err := cordisInspectQuery(host, "host", "bogus", "x", nil); err == nil {
		t.Fatal("非法 provider 应报错")
	}
	if _, err := cordisInspectQuery(host, "mars", "service", "list", nil); err == nil {
		t.Fatal("非法 platform 应报错")
	}
}

// TestClientInspectSnapshot 浏览器上报快照 → client 平台真实状态查询。
func TestClientInspectSnapshot(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	// 未上报：离线
	out, err := cordisInspectQuery(host, "client", "plugin", "listPlugin", nil)
	if err != nil || !strings.Contains(out, "浏览器未连接") {
		t.Fatalf("未上报应离线: %v\n%s", err, out)
	}
	// 上报快照（模拟浏览器 plugin-runtime）
	host.SetClientState(ClientRuntimeSnapshot{
		Plugins: []ClientPluginSnapshot{
			{Name: "hello-panel", Status: "loaded", Panels: []string{"hello-panel"}, Events: []string{"ui:pong"}, Version: "dyn-1"},
			{Name: "broken", Status: "error", Error: "TypeError: x is not a function"},
		},
		Panels: []string{"hello-panel"},
	})
	// listPlugin：真实状态（loaded/error + 面板/事件计数）
	out, err = cordisInspectQuery(host, "client", "plugin", "listPlugin", nil)
	if err != nil || !strings.Contains(out, "hello-panel") || !strings.Contains(out, "broken") {
		t.Fatalf("listPlugin: %v\n%s", err, out)
	}
	if !strings.Contains(out, "panels=1 events=1") {
		t.Fatalf("listPlugin 应含面板/事件计数: %s", out)
	}
	if !strings.Contains(out, "error: TypeError") {
		t.Fatalf("listPlugin 应含错误状态: %s", out)
	}
	// getPlugin：单插件详情（面板/事件清单）
	out, err = cordisInspectQuery(host, "client", "plugin", "getPlugin", map[string]any{"name": "hello-panel"})
	if err != nil || !strings.Contains(out, "hello-panel") || !strings.Contains(out, "ui:pong") || !strings.Contains(out, "面板") {
		t.Fatalf("getPlugin: %v\n%s", err, out)
	}
	// getPlugin 未知名字报错
	if _, err := cordisInspectQuery(host, "client", "plugin", "getPlugin", map[string]any{"name": "nope"}); err == nil {
		t.Fatal("未知 client 半应报错")
	}
	// event listEvent：全部事件 + 归属
	out, err = cordisInspectQuery(host, "client", "event", "listEvent", nil)
	if err != nil || !strings.Contains(out, "ui:pong") || !strings.Contains(out, "hello-panel") {
		t.Fatalf("listEvent: %v\n%s", err, out)
	}
	// 非法 provider/method
	if _, err := cordisInspectQuery(host, "client", "tool", "listTool", nil); err != nil {
		t.Fatalf("client tool 平台应给摘要而非报错: %v", err)
	}
	if _, err := cordisInspectQuery(host, "client", "event", "getEvent", nil); err == nil {
		t.Fatal("event 平台非法 method 应报错")
	}
}

// TestClientStateTTL 快照过期视为离线。
func TestClientStateTTL(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	host.SetClientState(ClientRuntimeSnapshot{
		Plugins: []ClientPluginSnapshot{{Name: "p1", Status: "loaded"}},
	})
	if snap := host.ClientState(); !snap.Connected {
		t.Fatal("刚上报应 connected")
	}
	// 伪造过期时间
	host.clientStateMu.Lock()
	host.clientState.ReportedAt = time.Now().Unix() - clientStateTTL - 5
	host.clientStateMu.Unlock()
	if snap := host.ClientState(); snap.Connected {
		t.Fatal("超时未上报应离线")
	}
	out, err := cordisInspectQuery(host, "client", "plugin", "listPlugin", nil)
	if err != nil || !strings.Contains(out, "浏览器未连接") {
		t.Fatalf("过期后应提示未连接: %v\n%s", err, out)
	}
}

// mustDef 按 id 取定义（测试失败即终止）。
func mustDef(t *testing.T, host *PluginHost, id string) *jsPluginDef {
	t.Helper()
	def, ok := host.GetJSDef(id)
	if !ok {
		t.Fatalf("定义不存在: %s", id)
	}
	return def
}

// findDefID 按插件名找定义 id（循环遍历 defs）。
func findDefID(t *testing.T, host *PluginHost, name string) string {
	t.Helper()
	for _, d := range host.JSDefs() {
		if d.name == name {
			return d.id
		}
	}
	t.Fatalf("定义未找到: %s", name)
	return ""
}

// 确保 context 导入被使用（cordisInspectQuery 工具注册的 Handler 签名需要）。
var _ = context.Background
