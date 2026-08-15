// plugin_test.go — 插件框架 + JS 动态插件运行时验证。
package agent

import (
	"context"
	"strings"
	"testing"
)

// ─── PluginHost 基本 ───────────────────────────────────────

func TestPluginHostBasic(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	registerBuiltinPlugins(host)

	// sysinfo 插件已 Use（Running）
	if host.State("sysinfo") != PluginRunning {
		t.Fatalf("sysinfo 应 running, got %v", host.State("sysinfo"))
	}
	// ctx.provide 服务可 get
	if v := host.Context().Get("workspaceRoot"); v != `C:\ws` {
		t.Fatalf("workspaceRoot 服务 = %v, want C:\\ws", v)
	}
	// Inspect 报告
	recs := host.Inspect()
	if len(recs) != 1 || recs[0].Name != "sysinfo" {
		t.Fatalf("Inspect = %+v, want [sysinfo]", recs)
	}
	// Unload 回收
	if err := host.Unload("sysinfo"); err != nil {
		t.Fatal(err)
	}
	if host.State("sysinfo") != PluginStopped {
		t.Fatalf("unload 后应 stopped")
	}
	// 同名重复注册报错
	dup := &GoPlugin{NameField: "sysinfo", ApplyFn: func(*PluginContext) error { return nil }}
	if err := host.Use(dup); err == nil {
		t.Fatalf("重复注册应报错")
	}
}

func TestPluginContextOnEffect(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	got := []string{}
	host.Use(&GoPlugin{
		NameField: "listener",
		ApplyFn: func(ctx *PluginContext) error {
			ctx.On("evt", func(p any) { got = append(got, p.(string)) })
			ctx.Effect(func() { got = append(got, "cleanup") })
			return nil
		},
	})
	host.EventBus().Emit("evt", "hello")
	if len(got) != 1 || got[0] != "hello" {
		t.Fatalf("Emit 后 got = %v", got)
	}
	// Unload 触发 effect
	if err := host.Unload("listener"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1] != "cleanup" {
		t.Fatalf("Unload 后 got = %v, want [hello cleanup]", got)
	}
	// 事件监听已回收
	host.EventBus().Emit("evt", "again")
	if len(got) != 2 {
		t.Fatalf("监听未回收: got = %v", got)
	}
}

// ─── JS 动态插件全链路 ─────────────────────────────────────

const demoJSPlugin = `
return {
  name: 'demo-js',
  apply(ctx) {
    ctx.provide('demoService', { ok: true })
    ctx.systemPrompt.section({ name: 'demo', order: 50, text: 'demo section' })
    ctx.tools.register(harness.defineTool({
      name: 'demo_hello',
      description: 'Say hello from JS plugin',
      parameters: { type: 'object', properties: { who: { type: 'string' } } },
      execute: (args) => ({ text: 'hello ' + (args.who || 'world') })
    }))
    ctx.tools.register({
      name: 'demo_add',
      description: 'Add two numbers',
      parameters: { type: 'object', properties: { a: { type: 'number' }, b: { type: 'number' } } },
      execute: async (args) => ({ text: 'sum=' + (args.a + args.b) })
    })
  }
}`

func TestJSDynamicPluginFullLifecycle(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)

	// 1. define（语法预检 + 登记）
	id, err := host.DefineJS(demoJSPlugin, "demo plugin")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	if !strings.HasPrefix(id, "dyn-") {
		t.Fatalf("id = %q, want dyn-N", id)
	}
	def, ok := host.GetJSDef(id)
	if !ok {
		t.Fatalf("定义未登记")
	}
	if def.name != "" { // 未装载时 name 未定
		t.Fatalf("定义时 name 应为空, got %q", def.name)
	}

	// 2. run（装载 + apply）
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	if host.State("demo-js") != PluginRunning {
		t.Fatalf("demo-js 应 running")
	}

	// 3. JS 插件注册的工具可用
	for _, name := range []string{"demo_hello", "demo_add"} {
		tool, ok := reg.Get(name)
		if !ok {
			t.Fatalf("工具 %s 未注册", name)
		}
		if tool.Description == "" {
			t.Fatalf("工具 %s description 为空", name)
		}
	}

	// 4. 执行同步 JS 工具
	out, err := reg.Execute(context.Background(), "demo_hello", `{"who":"插件"}`)
	if err != nil {
		t.Fatalf("demo_hello: %v", err)
	}
	if !strings.Contains(out, "hello 插件") {
		t.Fatalf("demo_hello 输出 = %q", out)
	}

	// 5. 执行 async JS 工具（Promise）
	out, err = reg.Execute(context.Background(), "demo_add", `{"a":2,"b":3}`)
	if err != nil {
		t.Fatalf("demo_add: %v", err)
	}
	if !strings.Contains(out, "sum=5") {
		t.Fatalf("demo_add 输出 = %q", out)
	}

	// 6. ctx.provide 的服务可跨插件 get
	if v := host.Context().Get("demoService"); v == nil {
		t.Fatalf("demoService 服务未提供")
	}

	// 7. 系统提示 section 已贡献
	sections := host.Sections()
	found := false
	for _, s := range sections {
		if s.Name == "demo" && s.Order == 50 && s.Text == "demo section" {
			found = true
		}
	}
	if !found {
		t.Fatalf("section demo 未找到: %+v", sections)
	}

	// 8. 工具对 LLM 可见（EnabledDefinitions）
	defs := reg.EnabledDefinitions()
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Function.Name] = true
	}
	if !names["demo_hello"] || !names["demo_add"] {
		t.Fatalf("JS 工具未进入 LLM 视图")
	}

	// 9. stop 回收工具
	if err := host.Unload("demo-js"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"demo_hello", "demo_add"} {
		if _, ok := reg.Get(name); ok {
			t.Fatalf("stop 后工具 %s 应被回收", name)
		}
	}

	// 10. undefine 删除定义
	if err := host.RemoveJSDef(id); err != nil {
		t.Fatal(err)
	}
	if _, ok := host.GetJSDef(id); ok {
		t.Fatalf("undefine 后定义应删除")
	}
}

func TestJSPluginSyntaxError(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	_, err := host.DefineJS(`return { name: 'bad' })(`, "bad")
	if err == nil {
		t.Fatalf("语法错误应被拒绝")
	}
	if !strings.Contains(err.Error(), "语法") {
		t.Fatalf("错误信息应含语法提示, got %v", err)
	}
}

func TestJSPluginMissingName(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	id, err := host.DefineJS(`return { apply(ctx) {} }`, "noname")
	if err != nil {
		t.Fatal(err)
	}
	def, _ := host.GetJSDef(id)
	err = host.LoadJSDynamic(def)
	if err == nil {
		t.Fatalf("缺 name 的插件应拒绝装载")
	}
}

func TestJSPluginErrorField(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, "")
	id, _ := host.DefineJS(`
return {
  name: 'err-plugin',
  apply(ctx) {
    ctx.tools.register({
      name: 'err_tool',
      description: 'always fail',
      execute: () => ({ error: 'boom' })
    })
  }
}`, "err")
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatal(err)
	}
	_, err := reg.Execute(context.Background(), "err_tool", `{}`)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("错误工具应返回 error, got %v", err)
	}
}

// ─── AgentBase.Init 冒烟 ─────────────────────────────────

func TestAgentBaseInitPlugins(t *testing.T) {
	dir := t.TempDir()
	base := NewAgentBase(AgentConfig{WorkspaceRoot: dir})
	if err := base.Init(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		base.Shutdown()
		if db := base.SessionMgr.SQLiteDB(); db != nil {
			_ = db.Close() // 释放 pair.db 句柄，避免 TempDir 清理失败
		}
	}()
	if base.Plugins == nil {
		t.Fatalf("Init 后 Plugins 应为非 nil")
	}
	// sysinfo 内置插件 running
	if base.Plugins.State("sysinfo") != PluginRunning {
		t.Fatalf("sysinfo 应 running")
	}
	// cordis 工具已注册且对 LLM 可见
	for _, name := range []string{"cordis_inspect", "cordis_define", "cordis_run", "cordis_stop", "cordis_undefine"} {
		tool, ok := base.Registry.Get(name)
		if !ok {
			t.Fatalf("工具 %s 未注册", name)
		}
		if !tool.Enabled {
			t.Fatalf("工具 %s 应启用", name)
		}
	}
	// workspaceRoot 服务可用
	if v := base.Plugins.Context().Get("workspaceRoot"); v != dir {
		t.Fatalf("workspaceRoot = %v, want %v", v, dir)
	}
}
