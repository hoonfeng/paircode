// plugin_test.go — 插件框架 + JS 动态插件运行时验证。
package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ─── PluginHost 基本 ───────────────────────────────────────

func TestPluginHostBasic(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)

	// ★ 框架能力已内联 NewPluginHost（不再以插件形态存在）：
	// workspaceRoot 服务直接可用（原 sysinfo 插件）
	if v := host.Context().Get("workspaceRoot"); v != `C:\ws` {
		t.Fatalf("workspaceRoot 服务 = %v, want C:\\ws", v)
	}
	// 内置工具集模板已注册（原 toolset-tpl-core 插件）
	if host.Template("toolset.tpl.project-helper") == nil {
		t.Fatal("内置模板 toolset.tpl.project-helper 未注册")
	}
	if host.Template("toolset.tpl.git-flow") == nil {
		t.Fatal("内置模板 toolset.tpl.git-flow 未注册")
	}
	// 插件列表不再含 sysinfo / toolset-tpl-core（无任何 Use）
	recs := host.Inspect()
	if len(recs) != 0 {
		t.Fatalf("Inspect 应为空（框架能力不占插件位）, got %d: %+v", len(recs), recs)
	}
	// 同名重复注册报错
	dup := &GoPlugin{NameField: "dup-test", ApplyFn: func(*PluginContext) error { return nil }}
	if err := host.Use(dup); err != nil {
		t.Fatalf("首次注册应成功: %v", err)
	}
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
	if def.name != "demo-js" { // ★ define 时静态提取插件名（审批键 name 需 define 时可用）
		t.Fatalf("定义时 name 应为静态提取的插件名, got %q", def.name)
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

// ─── TS 动态插件（内置编译器）────────────────────────────

const demoTSPlugin = `
interface ToolArgs { who?: string }
interface PluginCtx {
  tools: { register(t: any): void }
  provide(name: string, value: any): void
}
const greet = (args: ToolArgs): string => 'ts hello ' + (args.who ?? 'world')

return {
  name: 'ts-demo',
  apply(ctx: PluginCtx) {
    ctx.provide('tsDemoService', { lang: 'ts' })
    ctx.tools.register({
      name: 'ts_hello',
      description: 'TS 插件工具',
      parameters: { type: 'object', properties: { who: { type: 'string' } } },
      execute: (args: ToolArgs) => ({ text: greet(args) })
    })
  }
}`

func TestTSDynamicPlugin(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)

	// 自动探测：TS 注解 → 内置编译转译
	id, err := host.DefineJS(demoTSPlugin, "ts demo")
	if err != nil {
		t.Fatalf("DefineJS(TS): %v", err)
	}
	def, ok := host.GetJSDef(id)
	if !ok {
		t.Fatalf("定义未登记")
	}
	if def.lang != "ts" {
		t.Fatalf("语言应探测为 ts, got %q", def.lang)
	}
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic(TS): %v", err)
	}
	if host.State("ts-demo") != PluginRunning {
		t.Fatalf("ts-demo 应 running")
	}
	out, err := reg.Execute(context.Background(), "ts_hello", `{"who":"TS"}`)
	if err != nil {
		t.Fatalf("ts_hello: %v", err)
	}
	if !strings.Contains(out, "ts hello TS") {
		t.Fatalf("ts_hello 输出 = %q", out)
	}
	if v := host.Context().Get("tsDemoService"); v == nil {
		t.Fatalf("tsDemoService 服务未提供")
	}
}

func TestTSDynamicPluginLanguageParam(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	// 显式 language=ts 强制转译
	id, err := host.DefineJSCode(demoTSPlugin, "ts", "explicit")
	if err != nil {
		t.Fatalf("DefineJSCode(ts): %v", err)
	}
	def, _ := host.GetJSDef(id)
	if def.lang != "ts" {
		t.Fatalf("显式 ts 应记录 lang=ts")
	}
	// 显式 language=js：TS 语法不被转译 → goja 语法错误
	_, err = host.DefineJSCode(`interface A { x: string } return { name: 'a', apply() {} }`, "js", "js-forced")
	if err == nil {
		t.Fatalf("强制 js 下含 interface 的代码应语法报错")
	}
}

func TestTSCompileError(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	// TS 语法错误 → 编译错误信息含行号
	_, err := host.DefineJSCode(`interface A { x: string } return { name: 'a', apply(ctx: A) { xxx }`, "ts", "bad")
	if err == nil {
		t.Fatalf("TS 编译错误应被拒绝")
	}
	if !strings.Contains(err.Error(), "编译") {
		t.Fatalf("错误信息应含编译提示, got %v", err)
	}
}

// ─── 多文件 TS 插件（bundle）──────────────────────────────

// ─── JS 插件 timer 服务（ctx.timeout/interval + 跨 goroutine 锁）─────────

// TestJSTimerService ctx.timeout 一次性定时器：回调跨 goroutine 触发，
// 经 VM 执行锁保护，Invoke 可读到回调后状态。
func TestJSTimerService(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(`let state = 'initial'
return {
  name: 'js-timer',
  apply(ctx) {
    harness.handle('getState', () => state)
    ctx.timeout(() => { state = 'after-timer' }, 60)
  }
}`, "timer-demo")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatal(err)
	}
	plug, ok := host.Get("js-timer")
	if !ok {
		t.Fatalf("js-timer 未注册")
	}
	adapter, ok := plug.(*jsPluginAdapter)
	if !ok {
		t.Fatalf("js-timer 应为 jsPluginAdapter")
	}
	// 初始状态
	got, err := adapter.Invoke("getState", nil)
	if err != nil || got != "initial" {
		t.Fatalf("初始 state = %v, err=%v", got, err)
	}
	// 等待 timeout 触发（60ms + 缓冲）
	deadline := time.Now().Add(2 * time.Second)
	for {
		got, err = adapter.Invoke("getState", nil)
		if err == nil && got == "after-timer" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout 未触发: state=%v err=%v", got, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	// 卸载应无 panic（timer 清理）
	if err := host.Unload("js-timer"); err != nil {
		t.Fatalf("Unload: %v", err)
	}
}

// TestJSIntervalAndCancel ctx.interval 周期定时器 + cancel 停止。
func TestJSIntervalAndCancel(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(`let n = 0
return {
  name: 'js-intv',
  apply(ctx) {
    const cancel = ctx.interval(() => { n++ }, 20)
    harness.handle('count', () => n)
    harness.handle('stop', () => { cancel(); return 'stopped' })
  }
}`, "interval-demo")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatal(err)
	}
	plug, _ := host.Get("js-intv")
	adapter, ok := plug.(*jsPluginAdapter)
	if !ok {
		t.Fatalf("js-intv 应为 jsPluginAdapter")
	}

	count := func() int {
		v, err := adapter.Invoke("count", nil)
		if err != nil {
			return -1
		}
		switch n := v.(type) {
		case float64:
			return int(n)
		case int64:
			return int(n)
		case int:
			return n
		}
		return -2
	}
	// 等待计数增长
	deadline := time.Now().Add(2 * time.Second)
	for count() < 1 {
		if time.Now().After(deadline) {
			t.Fatalf("interval 未触发, count=%d", count())
		}
		time.Sleep(10 * time.Millisecond)
	}
	// 停止
	if _, err := adapter.Invoke("stop", nil); err != nil {
		t.Fatalf("stop: %v", err)
	}
	before := count()
	time.Sleep(100 * time.Millisecond)
	after := count()
	if after != before {
		t.Fatalf("cancel 后计数仍增长: before=%d after=%d", before, after)
	}
	_ = host.Unload("js-intv")
}

// TestJSTimerErrorIsolation timer 回调抛错不崩宿主（recover 吞掉）。
func TestJSTimerErrorIsolation(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(`return {
  name: 'js-timer-err',
  apply(ctx) {
    ctx.timeout(() => { throw new Error('timer boom') }, 30)
  }
}`, "timer-err")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatal(err)
	}
	// 等 timer 触发（抛错应被 recover）
	time.Sleep(200 * time.Millisecond)
	// 宿主仍可用：再装载一个工具类操作
	if host.State("js-timer-err") != PluginRunning {
		t.Fatalf("插件应仍 running")
	}
	_ = host.Unload("js-timer-err")
}

// TestTSMultiFilePlugin 验证多文件 TS 插件：相对 import（./util）内联打包、
// 非相对包 import（@deepseek-ai/cordis）mock 成空模块、export default 导出插件。
// 对象导出形态：export default { name, apply(ctx) }。
func TestTSMultiFilePlugin(t *testing.T) {
	dir := t.TempDir()
	// 目录里放一个被插件 import 的辅助模块（util.ts）
	utilPath := filepath.Join(dir, "util.ts")
	if err := os.WriteFile(utilPath, []byte(`export function greet(n: string): string { return 'ts-multi hi ' + n }`), 0o644); err != nil {
		t.Fatal(err)
	}

	// 插件源码：相对导入 util + harness 生态包 import（仅类型用途，bundle 后擦除/mock）
	pluginSrc := `
import { greet } from './util'
import type { Context } from '@deepseek-ai/cordis'
interface PluginArgs { who?: string }
const who = (ctx: any): string => ctx?.who ?? 'world'
export default {
  name: 'ts-multi',
  apply(ctx: Context) {
    ctx.tools.register({
      name: 'ts_multi_hello',
      description: 'multi-file ts plugin demo',
      execute: (args: PluginArgs) => ({ output: greet(args?.who ?? who(ctx)) })
    })
  }
}`
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	id, err := host.DefineJSCodeDir(pluginSrc, "ts", "multi-file demo", dir)
	if err != nil {
		t.Fatalf("DefineJSCodeDir(多文件 TS): %v", err)
	}
	def, ok := host.GetJSDef(id)
	if !ok {
		t.Fatalf("定义未登记")
	}
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic(多文件): %v", err)
	}
	if host.State("ts-multi") != PluginRunning {
		t.Fatalf("ts-multi 应 running")
	}
	out, err := reg.Execute(context.Background(), "ts_multi_hello", `{"who":"multifile"}`)
	if err != nil {
		t.Fatalf("ts_multi_hello: %v", err)
	}
	if !strings.Contains(out, "ts-multi hi multifile") {
		t.Fatalf("输出 = %q", out)
	}
}

// TestTSMultiFileFunctionForm harness 生态惯例形态：export default function(ctx)
// 直接作为插件 apply（bundle 包装成 {name, apply}）。
func TestTSMultiFileFunctionForm(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "util.ts"), []byte(`export const tag = (n: string): string => 'fn-form ' + n`), 0o644)
	pluginSrc := `
import { tag } from './util'
export default function (ctx: any) {
  ctx.tools.register({
    name: 'ts_fn_hello',
    description: 'function form plugin',
    execute: (args: { who?: string }) => ({ output: tag(args?.who ?? 'world') })
  })
}`
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	id, err := host.DefineJSCodeDir(pluginSrc, "ts", "fn-form", dir)
	if err != nil {
		t.Fatalf("DefineJSCodeDir: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	if host.State(def.name) != PluginRunning {
		t.Fatalf("%s 应 running", def.name)
	}
	out, err := reg.Execute(context.Background(), "ts_fn_hello", `{"who":"F"}`)
	if err != nil {
		t.Fatalf("ts_fn_hello: %v", err)
	}
	if !strings.Contains(out, "fn-form F") {
		t.Fatalf("输出 = %q", out)
	}
}

// TestTSMultiFileNoDefault 多文件 bundle 但缺 export default → 运行时报错提示。
func TestTSMultiFileNoDefault(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	// 有 import 但只 export 命名函数（无 default）
	src := `
import { greet } from './util'
export function helper(n: string): string { return greet(n) }`
	utilPath := filepath.Join(dir, "util.ts")
	_ = os.WriteFile(utilPath, []byte(`export function greet(n: string): string { return 'g ' + n }`), 0o644)

	id, err := host.DefineJSCodeDir(src, "ts", "no-default", dir)
	if err != nil {
		t.Fatalf("定义应成功（bundle 编译期不检查 default）: %v", err)
	}
	def, _ := host.GetJSDef(id)
	err = host.LoadJSDynamic(def)
	if err == nil {
		t.Fatalf("缺 export default 运行时应报错")
	}
	if !strings.Contains(err.Error(), "undefined") {
		t.Fatalf("错误应指向 default undefined, got %v", err)
	}
}

// ─── 自举闭环端到端（cordis_define → cordis_run → 工具/事件/timer/服务）─────

// TestBootstrapEndToEnd 模型视角的全链路：多文件 TS 插件（import 相对模块 +
// ctx.provide 服务 + ctx.on 事件 + ctx.timeout 定时器 + ctx.tools.register 工具 +
// harness.handle 方法）经 cordis_define 登记 → cordis_run 装载 → 各能力逐项验证
// → cordis_stop 回收。这是自举链路的完整闭环：用 agent 的 cordis 工具集
// 动态注册新能力并驱动。
func TestBootstrapEndToEnd(t *testing.T) {
	dir := t.TempDir()
	// 多文件源码：util.ts 辅助模块 + plugin.ts（相对 import）
	if err := os.WriteFile(filepath.Join(dir, "util.ts"),
		[]byte(`export function greet(n: string): string { return 'boot-greet ' + n }`), 0o644); err != nil {
		t.Fatal(err)
	}
	pluginSrc := `
import { greet } from './util'
let lastEvent: string = 'none'
let timerFired: boolean = false
export default {
  name: 'boot-plugin',
  apply(ctx: any) {
    ctx.provide('bootValue', 'hello-service')
    ctx.on('boot:ping', (payload: any) => { lastEvent = String(payload) })
    ctx.timeout(() => { timerFired = true }, 50)
    harness.handle('getState', () => ({ lastEvent, timerFired }))
    ctx.tools.register({
      name: 'boot_hello',
      description: 'bootstrap plugin tool',
      execute: (args: { who?: string }) => ({ output: greet(args?.who ?? 'world') })
    })
  }
}`

	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	RegisterCordisTools(reg, host, dir)
	ctx := context.Background()

	// ① cordis_define：登记多文件 TS 插件（dir 指定源码目录）
	defOut, err := reg.Execute(ctx, "cordis_define",
		`{"code":`+strconv.Quote(pluginSrc)+`,"language":"ts","dir":`+strconv.Quote(dir)+`,"purpose":"自举闭环 e2e"}`)
	if err != nil {
		t.Fatalf("cordis_define: %v", err)
	}
	if !strings.Contains(defOut, "dyn-") {
		t.Fatalf("cordis_define 应返回 dyn id，实际: %s", defOut)
	}
	m := regexp.MustCompile(`dyn-\d+`).FindString(defOut)
	if m == "" {
		t.Fatalf("无法从输出提取 dyn id: %s", defOut)
	}
	id := m

	// ② cordis_run：装载（apply 注册工具/服务/事件/timer）
	if _, err := reg.Execute(ctx, "cordis_run", `{"id":"`+id+`"}`); err != nil {
		t.Fatalf("cordis_run: %v", err)
	}
	if host.State("boot-plugin") != PluginRunning {
		t.Fatalf("boot-plugin 应 running")
	}

	plug, ok := host.Get("boot-plugin")
	if !ok {
		t.Fatalf("boot-plugin 未注册")
	}
	adapter, ok := plug.(*jsPluginAdapter)
	if !ok {
		t.Fatalf("boot-plugin 应为 jsPluginAdapter")
	}

	// ③ 初始状态（timer 未触发、事件未到）
	got, err := adapter.Invoke("getState", nil)
	if err != nil {
		t.Fatalf("getState 初始: %v", err)
	}
	st, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("getState 应为对象，实际 %T", got)
	}
	if st["lastEvent"] != "none" || st["timerFired"] != false {
		t.Fatalf("初始状态不符: %v", st)
	}

	// ④ 服务可用（ctx.provide）
	if v := host.Context().Get("bootValue"); v != "hello-service" {
		t.Fatalf("bootValue 服务 = %v, want hello-service", v)
	}

	// ⑤ 事件触发（ctx.on + EventBus.Emit，回调跨 goroutine 经执行锁）
	host.EventBus().Emit("boot:ping", "pong")

	// ⑥ 插件注册的工具可调用（模型视角）
	out, err := reg.Execute(ctx, "boot_hello", `{"who":"e2e"}`)
	if err != nil {
		t.Fatalf("boot_hello: %v", err)
	}
	if !strings.Contains(out, "boot-greet e2e") {
		t.Fatalf("boot_hello 输出 = %q", out)
	}

	// ⑦ 等待 timer 触发 + 事件回调生效（50ms + 缓冲）
	deadline := time.Now().Add(2 * time.Second)
	for {
		got, err = adapter.Invoke("getState", nil)
		if err == nil {
			st, _ = got.(map[string]any)
			if st != nil && st["lastEvent"] == "pong" && st["timerFired"] == true {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("事件/timer 未生效: %v (err=%v)", st, err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// ⑧ cordis_stop：回收贡献（插件工具消失、服务移除、timer 清理）
	if _, err := reg.Execute(ctx, "cordis_stop", `{"id":"`+id+`"}`); err != nil {
		t.Fatalf("cordis_stop: %v", err)
	}
	if host.State("boot-plugin") != PluginStopped {
		t.Fatalf("boot-plugin 应 stopped")
	}
	if _, err := reg.Execute(ctx, "boot_hello", `{}`); err == nil {
		t.Error("cordis_stop 后插件工具应已回收")
	}
	if v := host.Context().Get("bootValue"); v != nil {
		t.Errorf("cordis_stop 后服务应移除，实际 %v", v)
	}

	// ⑨ cordis_inspect 报告可见
	report, err := reg.Execute(ctx, "cordis_inspect", `{}`)
	if err != nil {
		t.Fatalf("cordis_inspect: %v", err)
	}
	if !strings.Contains(report, "boot-plugin") {
		t.Errorf("cordis_inspect 应含 boot-plugin：\n%s", report)
	}
}

// TestAgentBaseInitPlugins 冒烟测试（AgentBase.Init 装配 cordis 工具）。
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
	// ★ 框架能力内联：workspaceRoot 服务可用（原 sysinfo 插件已非插件形态）
	if v := base.Plugins.Context().Get("workspaceRoot"); v != dir {
		t.Fatalf("workspaceRoot 服务 = %v, want %v", v, dir)
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
}

// ─── P0：函数形态插件 + config + inject 服务（对齐 harness 生态）────────────

// TestJSFunctionFormPlugin cordis 生态函数形态插件（单文件）：
// return (ctx, config) => void；函数名作插件名；config 透传 apply 第二参。
func TestJSFunctionFormPlugin(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(`
return function myFnPlugin(ctx, config) {
  ctx.tools.register({
    name: 'fn_greet',
    description: 'function-form plugin tool',
    parameters: { type: 'object', properties: {} },
    execute: () => ({ text: 'fn plugin, cfg=' + (config ? config.greeting : 'none') })
  })
}`, "func form")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	def.config = map[string]any{"greeting": "hi"}
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	if host.State("myFnPlugin") != PluginRunning {
		t.Fatalf("函数插件名应为 myFnPlugin, got %v", host.State("myFnPlugin"))
	}
	out, err := reg.Execute(context.Background(), "fn_greet", `{}`)
	if err != nil {
		t.Fatalf("fn_greet: %v", err)
	}
	if !strings.Contains(out, "cfg=hi") {
		t.Fatalf("config 未透传, out=%q", out)
	}
	// 函数形态插件也支持 fn.inject 静态属性
	id2, _ := host.DefineJS(`
function injFn(ctx) { ctx.tools.register({ name: 'fn_inj_ok', description: 'inj', execute: () => ({ text: 'inj-ok' }) }) }
injFn.inject = ['logger']
return injFn`, "fn inject")
	def2, _ := host.GetJSDef(id2)
	if err := host.LoadJSDynamic(def2); err != nil {
		t.Fatalf("fn.inject 声明应支持: %v", err)
	}
	if host.State("injFn") != PluginRunning {
		t.Fatalf("injFn 应 running")
	}
}

// TestJSPluginInjectValidation inject 声明（D3 等待语义）：缺失服务 → 进入 waiting
// （不报错，waitingFor 记录缺失服务）；服务出现后自动激活。声明存在服务 → 装载成功。
func TestJSPluginInjectValidation(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, `C:\ws`)
	// ① 声明不存在的服务 → 进入 waiting（不报错；等待服务出现）
	id, _ := host.DefineJS(`
return {
  name: 'inj-bad',
  inject: ['database'],
  apply(ctx) {}
}`, "inj")
	def, _ := host.GetJSDef(id)
	err := host.LoadJSDynamic(def)
	if err != nil {
		t.Fatalf("inject 缺失应进入 waiting 而非报错: %v", err)
	}
	if def.status != PluginWaiting {
		t.Fatalf("def 应 waiting, got %s", def.status)
	}
	if !strInSlice(def.waitingFor, "database") {
		t.Fatalf("waitingFor 应含 database, got %v", def.waitingFor)
	}
	if host.State("inj-bad") == PluginRunning {
		t.Fatalf("waiting 中不应 running")
	}
	// ② 其他插件提供 database 服务 → waiting 插件自动激活（D3 自动重试）
	providerID, _ := host.DefineJS(`
return {
  name: 'db-provider',
  apply(ctx) {
    ctx.provide('database', { connect() { return 'connected' } })
  }
}`, "db-provider")
	pdef, _ := host.GetJSDef(providerID)
	if err := host.LoadJSDynamic(pdef); err != nil {
		t.Fatalf("provider 装载失败: %v", err)
	}
	if host.State("inj-bad") != PluginRunning {
		t.Fatalf("database 提供后 inj-bad 应自动激活 running, got %s", host.State("inj-bad"))
	}
	if def.status != PluginRunning {
		t.Fatalf("def 应 running, got %s", def.status)
	}
	// ③ 声明 fs（宿主提供）→ 装载成功且 ctx.fs 注入
	id2, _ := host.DefineJS(`
return {
  name: 'inj-ok',
  inject: ['fs'],
  apply(ctx) {
    if (typeof ctx.fs === 'undefined' || ctx.fs === null) throw new Error('fs 未注入')
  }
}`, "inj-ok")
	def2, _ := host.GetJSDef(id2)
	if err := host.LoadJSDynamic(def2); err != nil {
		t.Fatalf("inject fs 应成功: %v", err)
	}
	if host.State("inj-ok") != PluginRunning {
		t.Fatalf("inj-ok 应 running")
	}
}

// TestJSPluginCtxFSService ctx.fs 服务：读写/追加/存在/列出 + 越界拦截。
func TestJSPluginCtxFSService(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, root)
	id, err := host.DefineJS(`
return {
  name: 'fs-plugin',
  inject: ['fs'],
  apply(ctx) {
    ctx.fs.mkdir('notes', true)
    ctx.fs.writeFile('notes/hello.txt', 'hello from plugin')
    const content = ctx.fs.readFile('notes/hello.txt')
    if (content !== 'hello from plugin') throw new Error('readFile mismatch: ' + content)
    if (!ctx.fs.exists('notes/hello.txt')) throw new Error('exists should be true')
    ctx.fs.appendFile('notes/hello.txt', '!')
    if (ctx.fs.readFile('notes/hello.txt') !== 'hello from plugin!') throw new Error('appendFile mismatch')
    const entries = ctx.fs.readdir('notes')
    if (entries.length !== 1 || entries[0] !== 'hello.txt') throw new Error('readdir mismatch: ' + JSON.stringify(entries))
    const st = ctx.fs.stat('notes/hello.txt')
    if (!st || st.size <= 0 || st.isDir) throw new Error('stat mismatch: ' + JSON.stringify(st))
    ctx.tools.register({
      name: 'fs_probe',
      description: 'fs plugin ok',
      parameters: { type: 'object', properties: {} },
      execute: () => ({ text: 'fs-ok' })
    })
  }
}`, "fs svc")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	out, err := reg.Execute(context.Background(), "fs_probe", `{}`)
	if err != nil || !strings.Contains(out, "fs-ok") {
		t.Fatalf("fs_probe: out=%q err=%v", out, err)
	}
	// 落盘验证
	if b, err := os.ReadFile(filepath.Join(root, "notes", "hello.txt")); err != nil || string(b) != "hello from plugin!" {
		t.Fatalf("落盘内容 = %q err=%v", b, err)
	}
	// 越界写入被拦截（apply 内 try/catch 捕获，插件仍装载成功）
	id2, _ := host.DefineJS(`
return {
  name: 'fs-escape',
  inject: ['fs'],
  apply(ctx) {
    try {
      ctx.fs.writeFile('C:/Windows/evil.txt', 'x')
      throw new Error('escape-not-blocked')
    } catch (e) {
      if (String(e).includes('escape-not-blocked')) throw e
    }
  }
}`, "escape")
	def2, _ := host.GetJSDef(id2)
	if err := host.LoadJSDynamic(def2); err != nil {
		t.Fatalf("越界应被拦截且插件装载成功: %v", err)
	}
}

// TestJSPluginCtxBashLogger ctx.bash.exec 与 ctx.logger 服务冒烟。
func TestJSPluginCtxBashLogger(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, root)
	id, err := host.DefineJS(`
return {
  name: 'bash-logger-plugin',
  inject: ['bash', 'logger'],
  apply(ctx) {
    const log = ctx.logger('demo')
    log.info('plugin booting')
    const res = ctx.bash.exec('echo plugin-bash-ok')
    if (!res.output.includes('plugin-bash-ok')) throw new Error('bash exec mismatch: ' + JSON.stringify(res))
    ctx.tools.register({
      name: 'bash_probe',
      description: 'bash plugin ok',
      parameters: { type: 'object', properties: {} },
      execute: () => ({ text: 'bash-ok' })
    })
  }
}`, "bash+logger")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	out, err := reg.Execute(context.Background(), "bash_probe", `{}`)
	if err != nil || !strings.Contains(out, "bash-ok") {
		t.Fatalf("bash_probe: out=%q err=%v", out, err)
	}
}

// ─── P1：VM 同步超时防护 ─────────────────────────────────

// TestJSTimeoutProtection 死循环插件（求值/apply/工具 execute）被 goja
// Interrupt 强制中断：进程不卡死、错误信息明确、超时后 VM 可继续使用。
func TestJSTimeoutProtection(t *testing.T) {
	// 调小超时加快测试（同包测试串行，无并发竞争）
	oldTool := jsToolTimeout
	oldEval := jsEvalTimeout
	oldApply := jsApplyTimeout
	jsToolTimeout = 500 * time.Millisecond
	jsEvalTimeout = 500 * time.Millisecond
	jsApplyTimeout = 500 * time.Millisecond
	defer func() {
		jsToolTimeout = oldTool
		jsEvalTimeout = oldEval
		jsApplyTimeout = oldApply
	}()

	// ① 求值死循环：DefineJS 仅编译（不执行），LoadJSDynamic 求值时死循环 → 超时中断
	host := NewPluginHost(NewRegistry(), nil, `C:\ws`)
	id, err := host.DefineJS(`while (true) {}`, "infinite eval")
	if err != nil {
		t.Fatalf("DefineJS（仅编译）应通过: %v", err)
	}
	def, _ := host.GetJSDef(id)
	start := time.Now()
	err = host.LoadJSDynamic(def)
	if err == nil {
		t.Fatalf("死循环求值应报错")
	}
	if !strings.Contains(err.Error(), "求值超时") {
		t.Fatalf("应提示求值超时, got %v", err)
	}
	if time.Since(start) > 20*time.Second {
		t.Fatalf("超时中断耗时过长: %v", time.Since(start))
	}

	// ② apply 死循环
	id2, _ := host.DefineJS(`return { name: 'apply-hang', apply() { while (true) {} } }`, "apply hang")
	def2, _ := host.GetJSDef(id2)
	err = host.LoadJSDynamic(def2)
	if err == nil || !strings.Contains(err.Error(), "apply 执行超时") {
		t.Fatalf("apply 死循环应报超时, got %v", err)
	}

	// ③ 工具 execute 死循环
	reg := NewRegistry()
	host2 := NewPluginHost(reg, nil, `C:\ws`)
	id3, _ := host2.DefineJS(`
return {
  name: 'tool-hang',
  apply(ctx) {
    ctx.tools.register({
      name: 'hang_tool',
      description: 'infinite loop',
      execute: () => { while (true) {} }
    })
  }
}`, "tool hang")
	def3, _ := host2.GetJSDef(id3)
	if err := host2.LoadJSDynamic(def3); err != nil {
		t.Fatalf("装载: %v", err)
	}
	start = time.Now()
	_, err = reg.Execute(context.Background(), "hang_tool", `{}`)
	if err == nil || !strings.Contains(err.Error(), "执行超时") {
		t.Fatalf("工具死循环应报超时, got %v", err)
	}
	if time.Since(start) > 20*time.Second {
		t.Fatalf("工具超时中断耗时过长: %v", time.Since(start))
	}

	// ④ 超时后 VM/进程仍健康：再装载正常插件 + 执行工具（ClearInterrupt 已清 flag）
	id4, _ := host2.DefineJS(`
return {
  name: 'after-timeout',
  apply(ctx) { ctx.tools.register({ name: 'ok_after', description: 'ok', execute: () => ({ text: 'alive' }) }) }
}`, "after")
	def4, _ := host2.GetJSDef(id4)
	if err := host2.LoadJSDynamic(def4); err != nil {
		t.Fatalf("超时后新插件装载失败（VM 状态污染）: %v", err)
	}
	out, err := reg.Execute(context.Background(), "ok_after", `{}`)
	if err != nil || !strings.Contains(out, "alive") {
		t.Fatalf("超时后工具执行异常: out=%q err=%v", out, err)
	}
}

// ─── P1：工具 schema 规范化（'json' type / $ref / 定义期校验）──────────────

// TestJSToolSchemaNormalization 工具 schema：'json' type → 移除（任意值）；
// $ref → 内联解析；非法 type → 定义期（apply）明确报错。
func TestJSToolSchemaNormalization(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(`
return {
  name: 'schema-plugin',
  apply(ctx) {
    ctx.tools.register({
      name: 'schema_tool',
      description: 'schema normalized tool',
      parameters: {
        type: 'object',
        properties: {
          payload: { type: 'json' },
          user: { $ref: '#/$defs/user' }
        },
        $defs: {
          user: { type: 'object', properties: { id: { type: 'integer' } } }
        }
      },
      execute: (args) => ({ text: 'schema-ok' })
    })
  }
}`, "schema")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	tool, ok := reg.Get("schema_tool")
	if !ok {
		t.Fatalf("schema_tool 未注册")
	}
	props := tool.Parameters["properties"].(map[string]any)
	payload, _ := props["payload"].(map[string]any)
	if _, hasType := payload["type"]; hasType {
		t.Fatalf("payload 的 'json' type 应被移除, got %v", payload)
	}
	user, _ := props["user"].(map[string]any)
	if _, hasRef := user["$ref"]; hasRef {
		t.Fatalf("user 的 $ref 应被内联, got %v", user)
	}
	if user["type"] != "object" {
		t.Fatalf("user 内联后应有 type=object, got %v", user)
	}
	// 工具可正常执行
	out, err := reg.Execute(context.Background(), "schema_tool", `{}`)
	if err != nil || !strings.Contains(out, "schema-ok") {
		t.Fatalf("schema_tool: out=%q err=%v", out, err)
	}

	// ② 非法 type → apply 期（registerTool）明确报错
	id2, _ := host.DefineJS(`
return {
  name: 'bad-schema',
  apply(ctx) {
    ctx.tools.register({
      name: 'bad_tool',
      description: 'bad',
      parameters: { type: 'wat', properties: {} },
      execute: () => ({ text: 'x' })
    })
  }
}`, "bad schema")
	def2, _ := host.GetJSDef(id2)
	err = host.LoadJSDynamic(def2)
	if err == nil || !strings.Contains(err.Error(), "schema 非法") {
		t.Fatalf("非法 type 应定义期报错, got %v", err)
	}
}

// ─── P2：同名冲突 + 服务目录 + patch 装配 ────────────────

// TestJSPluginToolNameConflict 插件工具同名冲突：宿主已有工具/其他插件工具
// 不能被静默覆盖；Unload 后归属释放，可再次注册。
func TestJSPluginToolNameConflict(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	// ① 插件接管宿主已有工具 → 合法（宿主执行器存档，供 ctx.hostTool 调用）
	reg.Register(&Tool{Name: "host_tool", Description: "host", Handler: func(ctx context.Context, args map[string]any) (string, error) { return "", nil }})
	id, _ := host.DefineJS(`
return {
  name: 'conflict-1',
  apply(ctx) {
    ctx.tools.register({ name: 'host_tool', description: 'clash', execute: () => ({ text: 'x' }) })
  }
}`, "conflict1")
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("插件接管宿主工具应成功（宿主执行器存档），got %v", err)
	}
	if _, ok := HostToolMeta("host_tool"); !ok {
		t.Fatal("接管后宿主执行器应已存档")
	}
	if tool, ok := reg.Get("host_tool"); !ok || tool.Description != "clash" {
		t.Fatalf("插件应接管 host_tool，got %+v", tool)
	}
	// ② 两插件注册同名工具 → 第二个报错（含占用方与处理建议）
	load := func(name, toolName string) error {
		pid, _ := host.DefineJS(`
return {
  name: '`+name+`',
  apply(ctx) {
    ctx.tools.register({ name: '`+toolName+`', description: 'x', execute: () => ({ text: 'x' }) })
  }
}`, name)
		d, _ := host.GetJSDef(pid)
		return host.LoadJSDynamic(d)
	}
	if err := load("plugin-a", "shared_tool"); err != nil {
		t.Fatalf("plugin-a 注册应成功: %v", err)
	}
	err := load("plugin-b", "shared_tool")
	if err == nil || !strings.Contains(err.Error(), "已被插件 plugin-a 注册") {
		t.Fatalf("plugin-b 同名应报错, got %v", err)
	}
	// ③ Unload plugin-a 后归属释放 → plugin-c 可注册同名
	if err := host.Unload("plugin-a"); err != nil {
		t.Fatal(err)
	}
	if err := load("plugin-c", "shared_tool"); err != nil {
		t.Fatalf("Unload 后应可再注册: %v", err)
	}
}

// TestCordisServiceList 服务目录工具：静态服务清单 + 动态服务（ctx.provide）。
func TestCordisServiceList(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	RegisterCordisTools(reg, host, `C:\ws`)
	// 提供动态服务后目录应列出
	pc := host.Context()
	cancel := pc.Provide("demoService", map[string]any{"ok": true})
	defer cancel()
	out, err := reg.Execute(context.Background(), "cordis_service_list", `{}`)
	if err != nil {
		t.Fatalf("cordis_service_list: %v", err)
	}
	for _, want := range []string{"fs", "web", "bash", "logger", "timer", "demoService"} {
		if !strings.Contains(out, want) {
			t.Fatalf("服务目录缺 %s:\n%s", want, out)
		}
	}
}

// TestLoadCordisPatch cordis.patch.json 静态装配：文件不存在正常返回；
// 合法条目装载（config 透传）；坏 JSON 报错。
func TestLoadCordisPatch(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	// 文件不存在 → nil
	if err := host.LoadCordisPatch(filepath.Join(dir, "nope.json")); err != nil {
		t.Fatalf("无装配文件应返回 nil: %v", err)
	}
	// 合法装配
	patch := filepath.Join(dir, "cordis.patch.json")
	content := `{
  "plugins": [
    {
      "purpose": "patch demo",
      "config": { "who": "patch" },
      "code": "return { name: 'patch-plugin', apply(ctx, config) { ctx.tools.register({ name: 'patch_hi', description: 'patch', execute: () => ({ text: 'hi ' + (config ? config.who : 'x') }) }) } }"
    }
  ]
}`
	if err := os.WriteFile(patch, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := host.LoadCordisPatch(patch); err != nil {
		t.Fatalf("LoadCordisPatch: %v", err)
	}
	if host.State("patch-plugin") != PluginRunning {
		t.Fatalf("patch-plugin 应 running")
	}
	out, err := reg.Execute(context.Background(), "patch_hi", `{}`)
	if err != nil || !strings.Contains(out, "hi patch") {
		t.Fatalf("patch_hi: out=%q err=%v", out, err)
	}
	// 坏 JSON → 报错
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := host.LoadCordisPatch(bad); err == nil {
		t.Fatalf("坏 JSON 应报错")
	}
}

// TestJSNodeAPITraps 对齐 harness NODE_API_REDIRECTS：require/setTimeout/fetch 等
// Node API 在沙箱中不可用，调用即抛教学错误引导走 ctx 服务。
func TestJSNodeAPITraps(t *testing.T) {
	cases := []struct {
		call, wantHint string
	}{
		{"require('fs')", "Node modules are unavailable"},
		{"setTimeout(() => {}, 100)", "ctx.timeout"},
		{"setInterval(() => {}, 100)", "ctx.interval"},
		{"fetch('https://x')", "ctx.web"},
	}
	for _, c := range cases {
		reg := NewRegistry()
		host := NewPluginHost(reg, nil, `C:\ws`)
		code := fmt.Sprintf(`return { name: 'trap-test', apply(ctx) { %s } }`, c.call)
		id, err := host.DefineJS(code, "trap")
		if err != nil {
			t.Fatalf("[%s] DefineJS: %v", c.call, err)
		}
		def, _ := host.GetJSDef(id)
		err = host.LoadJSDynamic(def)
		if err == nil {
			t.Fatalf("[%s] 应抛错（Node API trap）", c.call)
		}
		if !strings.Contains(err.Error(), c.wantHint) {
			t.Fatalf("[%s] 错误 %q 应包含引导 %q", c.call, err.Error(), c.wantHint)
		}
		if host.State("trap-test") != PluginStopped {
			t.Fatalf("[%s] 插件应 stopped（apply 失败未装载）", c.call)
		}
	}
}

// ★ 2026-08-19：插件面板工具对勾 = cordis 可见性（与 agent 工具集解耦）。
// ctx.tools.list 应过滤对 cordis 隐藏的工具；SetToolCordisVisible 不影响 agent
// 可见性（Enabled/工具集独立）。
func TestToolCordisVisibility(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{Name: "tool_a", Description: "a", Handler: func(ctx context.Context, args map[string]any) (string, error) { return "a", nil }})
	host := NewPluginHost(reg, nil, `C:\ws`)
	if host == nil {
		t.Fatal("NewPluginHost nil")
	}

	// 默认全部对 cordis 可见
	if !host.IsToolCordisVisible("tool_a") {
		t.Error("默认工具应对 cordis 可见")
	}

	// 隐藏 tool_a → ctx.tools.list 过滤（探针工具观察）
	host.SetToolCordisVisible("tool_a", false)
	code := `return { name: 'cordis-vis-test', apply(ctx) {
	  ctx.tools.register({ name: 'list_probe', description: 'probe', execute: () => ({ text: ctx.tools.list().join(',') }) })
	} }`
	id, err := host.DefineJS(code, "cordis visibility")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	t.Cleanup(func() { _ = host.Unload(def.name) })

	out, err := reg.Execute(context.Background(), "list_probe", `{}`)
	if err != nil {
		t.Fatalf("list_probe: %v", err)
	}
	if strings.Contains(out, "tool_a") {
		t.Errorf("ctx.tools.list 应过滤对 cordis 隐藏的工具（含 tool_a）: %s", out)
	}
	if !strings.Contains(out, "list_probe") {
		t.Errorf("ctx.tools.list 应包含插件自注册工具: %s", out)
	}

	// agent 可见性（Enabled）不受影响
	if !reg.IsEnabled("tool_a") {
		t.Error("对 cordis 隐藏不应影响 agent 可见性（Enabled 仍 true）")
	}

	// 恢复可见 → ctx.tools.list 重新包含
	host.SetToolCordisVisible("tool_a", true)
	if !host.IsToolCordisVisible("tool_a") {
		t.Error("恢复后应对 cordis 可见")
	}
	out, err = reg.Execute(context.Background(), "list_probe", `{}`)
	if err != nil {
		t.Fatalf("list_probe(恢复后): %v", err)
	}
	if !strings.Contains(out, "tool_a") {
		t.Errorf("恢复后 ctx.tools.list 应包含 tool_a: %s", out)
	}
}
