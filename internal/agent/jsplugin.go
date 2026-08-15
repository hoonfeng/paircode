// ═══════════════════════════════════════════════════════════════
// plugin_js.go — JS 动态插件运行时（goja 沙箱）
//
// 对齐 deepseek-harness 的 cordis-host-runner / sandbox：
//   - 插件代码形态：`(async () => { <body> })()`，body 里 `return` 一个插件对象
//     { name, apply(ctx) }；apply(ctx) 中经 ctx 注册工具/系统提示/服务/事件。
//   - 沙箱暴露（与 harness 同名的全局）：
//       ctx     → get / provide / on / effect / tools.register / systemPrompt.section
//       harness → defineTool / registerTool / handle
//       console / btoa / atob / TextEncoder / TextDecoder
//   - require / setTimeout 等 Node API 不存在（对齐 harness 的沙箱纪律，
//     需要时经 ctx 服务；第一版不提供 timer，后续可按需补 ctx.timeout）。
//   - 求值通过注入 __resolve 回调 + `(async () => {...})().then(v=>__resolve(v))`
//     取回插件对象（goja 在 RunString 返回前已 drain 微任务队列，已验证）。
//
// 信任立场与 harness 一致：沙箱隔离全局，但不是安全边界——JS 插件应
// 视同 bash 访问同等信任级别。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"wb-ui.com/goja"
)

// ─── JS 插件定义 ───────────────────────────────────────────

// jsPluginDef 一个 JS 动态插件定义（cordis_define 登记；进程内存，不落盘）。
type jsPluginDef struct {
	id        string    // dyn-<n>
	lang      string    // 源码语言 "js" | "ts"（登记时探测/指定；code 存转译后 JS）
	name      string    // 插件名（默认取代码返回的 name）
	purpose   string    // 用途说明
	code      string    // host 半代码（async 函数体，return 插件对象）
	version   string    // "dyn-<n>"
	provides  []string  // 提供服务的键（插件运行时从 ctx.provide 收集）
	createdAt time.Time
}

// ─── JS 插件适配器 ─────────────────────────────────────────

// jsPluginAdapter 把 goja 里求值出的 JS 插件对象适配为 Plugin。
type jsPluginAdapter struct {
	host    *PluginHost
	def     *jsPluginDef
	vm      *goja.Runtime
	applyFn goja.Callable

	mu       sync.Mutex
	handlers map[string]func(args any) (any, error) // harness.handle 注册的方法
}

// Name 实现 Plugin。
func (p *jsPluginAdapter) Name() string { return p.def.name }

// Apply 实现 Plugin：创建绑定本插件上下文的 ctx 沙箱对象，调用 JS apply。
func (p *jsPluginAdapter) Apply(pc *PluginContext) error {
	ctxObj, err := p.buildContextObject(pc)
	if err != nil {
		return err
	}
	_, callErr := p.applyFn(goja.Undefined(), ctxObj)
	if callErr != nil {
		return fmt.Errorf("JS 插件 %s apply 执行失败: %v", p.def.name, jsErrorText(callErr))
	}
	return nil
}

// Invoke 调用插件用 harness.handle 注册的方法（供宿主/其他代码调用）。
func (p *jsPluginAdapter) Invoke(method string, args any) (any, error) {
	p.mu.Lock()
	fn, ok := p.handlers[method]
	p.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("JS 插件 %s 未注册 handler %q", p.def.name, method)
	}
	return fn(args)
}

// buildContextObject 构造沙箱 ctx 对象（绑定 pc 与 vm）。
// harness 全局也在此注入（插件 apply 内可用 harness.defineTool/registerTool/handle）。
func (p *jsPluginAdapter) buildContextObject(pc *PluginContext) (*goja.Object, error) {
	vm := p.vm
	injectHarness(vm, p, pc)
	ctxObj := vm.NewObject()

	ctxObj.Set("get", func(call goja.FunctionCall) goja.Value {
		v := pc.Get(call.Argument(0).String())
		if v == nil {
			return goja.Undefined()
		}
		return vm.ToValue(v)
	})
	ctxObj.Set("provide", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		cancel := pc.Provide(name, call.Argument(1).Export())
		p.mu.Lock()
		if !strInSlice(p.def.provides, name) {
			p.def.provides = append(p.def.provides, name)
		}
		p.mu.Unlock()
		_ = cancel
		return goja.Undefined()
	})
	ctxObj.Set("on", func(call goja.FunctionCall) goja.Value {
		evt := call.Argument(0).String()
		fnVal := call.Argument(1)
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			panic(vm.NewTypeError("ctx.on: 第二参数必须是函数"))
		}
		cancel := pc.On(evt, func(payload any) {
			_, _ = fn(goja.Undefined(), vm.ToValue(payload))
		})
		_ = cancel
		return goja.Undefined()
	})
	ctxObj.Set("effect", func(call goja.FunctionCall) goja.Value {
		fnVal := call.Argument(0)
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			panic(vm.NewTypeError("ctx.effect: 参数必须是函数"))
		}
		pc.Effect(func() {
			_, _ = fn(goja.Undefined())
		})
		return goja.Undefined()
	})

	// ctx.tools.register(toolDef)
	toolsObj := vm.NewObject()
	toolsObj.Set("register", func(call goja.FunctionCall) goja.Value {
		tool, tErr := jsToolToGo(vm, call.Argument(0))
		if tErr != nil {
			panic(vm.NewGoError(tErr))
		}
		pc.RegisterTool(tool)
		return goja.Undefined()
	})
	toolsObj.Set("list", func(call goja.FunctionCall) goja.Value {
		names := pc.Tools.Names()
		return vm.ToValue(names)
	})
	ctxObj.Set("tools", toolsObj)

	// ctx.systemPrompt.section({name, order, text})
	sysObj := vm.NewObject()
	sysObj.Set("section", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0).ToObject(vm)
		sec := &PromptSection{
			Name:  arg.Get("name").String(),
			Order: int(arg.Get("order").ToInteger()),
			Text:  arg.Get("text").String(),
		}
		if sec.Order == 0 {
			sec.Order = 100
		}
		if sec.Name == "" {
			sec.Name = fmt.Sprintf("js:%s", p.def.name)
		}
		pc.AddSystemPromptSection(sec)
		return goja.Undefined()
	})
	ctxObj.Set("systemPrompt", sysObj)

	// ctx.app：宿主基本信息（可选）
	appObj := vm.NewObject()
	appObj.Set("workspaceRoot", pc.WorkspaceRoot)
	ctxObj.Set("app", appObj)

	return ctxObj, nil
}

// ─── 沙箱创建与求值 ────────────────────────────────────────

// newJSSandbox 创建插件沙箱：注入 console/btoa/atob/TextEncoder/TextDecoder
// 与 __resolve 回调。返回 runtime 与 resolve 回调（goja 值 → 插件对象导出）。
func newJSSandbox(id string) (*goja.Runtime, *goja.Object) {
	vm := goja.New()

	// console（对齐 harness：带包名 tag，写透到宿主 stdout）
	tag := fmt.Sprintf("[js-plugin:%s]", id)
	consoleObj := vm.NewObject()
	logFn := func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, a := range call.Arguments {
			parts[i] = jsConsoleArg(a)
		}
		log.Printf("%s %s", tag, strings.Join(parts, " "))
		return goja.Undefined()
	}
	for _, m := range []string{"log", "info", "warn", "debug", "error"} {
		consoleObj.Set(m, logFn)
	}
	vm.Set("console", consoleObj)

	// btoa / atob
	vm.Set("btoa", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(base64.StdEncoding.EncodeToString([]byte(call.Argument(0).String())))
	})
	vm.Set("atob", func(call goja.FunctionCall) goja.Value {
		b, err := base64.StdEncoding.DecodeString(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(string(b))
	})

	// TextEncoder / TextDecoder（polyfill：Go 侧 UTF-8 编解码）
	vm.Set("__encodeUtf8", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue([]byte(call.Argument(0).String()))
	})
	vm.Set("__decodeUtf8", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(string(jsBytesOf(call.Argument(0))))
	})
	if _, err := vm.RunString(textCodecPolyfill); err != nil {
		log.Printf("[plugin_js] TextEncoder polyfill 失败: %v", err)
	}

	// __resolve 回调（求值结果交接点）：把 JS 值转成 Go 对象
	resolveObj := vm.NewObject()
	return vm, resolveObj
}

// textCodecPolyfill 标准 TextEncoder/TextDecoder（基于 __encodeUtf8/__decodeUtf8）。
const textCodecPolyfill = `
globalThis.TextEncoder = class {
  constructor() {}
  encode(s) { return __encodeUtf8(String(s)); }
}
globalThis.TextDecoder = class {
  constructor(label) { this.label = label || 'utf-8'; }
  decode(buf) {
    if (buf == null) return '';
    if (typeof buf === 'string') return buf;
    const b = (buf instanceof Uint8Array) ? buf : new Uint8Array(buf);
    return __decodeUtf8(Array.from(b));
  }
}
`

// jsBytesOf 把 JS 值（数组/Uint8Array/ArrayBuffer）转 []byte。
func jsBytesOf(v goja.Value) []byte {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	exp := v.Export()
	switch t := exp.(type) {
	case []byte:
		return t
	case []any:
		b := make([]byte, len(t))
		for i, e := range t {
			b[i] = byte(toInt64(e))
		}
		return b
	default:
		return []byte(v.String())
	}
}

// ─── JS 插件求值/装载 ─────────────────────────────────────

// evalJSPlugin 求值 host 半代码（async 函数体），返回插件对象（name/apply）。
// 对齐 harness evaluateHostCode：`(async () => { code })()` + then 交接。
func evalJSPlugin(vm *goja.Runtime, code, id string) (*goja.Object, error) {
	var resolved *goja.Object
	var resolveErr error
	done := make(chan struct{})

	vm.Set("__resolve", func(call goja.FunctionCall) goja.Value {
		defer close(done)
		v := call.Argument(0)
		if o, ok := v.(*goja.Object); ok {
			resolved = o
		} else {
			resolveErr = fmt.Errorf("插件求值未返回对象（得到 %v）", v)
		}
		return goja.Undefined()
	})

	src := "(async () => {\n" + code + "\n})().then(__resolve, e => __resolve({ __error: String(e) }))"
	if _, err := vm.RunString(src); err != nil {
		return nil, fmt.Errorf("插件 %s 语法错误: %v", id, jsErrorText(err))
	}
	<-done // 微任务已同步 drain，通道即时关闭
	if resolveErr != nil {
		return nil, resolveErr
	}
	if e := resolved.Get("__error"); e != nil && !goja.IsUndefined(e) && e.String() != "" {
		return nil, fmt.Errorf("插件 %s 求值失败: %s", id, e.String())
	}
	return resolved, nil
}

// LoadJSDynamic 求值并装载一个 JS 动态插件（对齐 cordis_run 的 host 半）。
// def 由 cordis_define 登记；装载后插件立即 apply（注册工具等）。
func (h *PluginHost) LoadJSDynamic(def *jsPluginDef) error {
	if def == nil || strings.TrimSpace(def.code) == "" {
		return fmt.Errorf("插件 %s: 代码为空", def.id)
	}
	vm, _ := newJSSandbox(def.id)
	obj, err := evalJSPlugin(vm, def.code, def.id)
	if err != nil {
		return err
	}

	nameVal := obj.Get("name")
	applyVal := obj.Get("apply")
	if nameVal == nil || nameVal.String() == "" {
		return fmt.Errorf("插件 %s: 缺少 name 字段（必须 return { name, apply(ctx) }）", def.id)
	}
	applyFn, ok := goja.AssertFunction(applyVal)
	if !ok {
		return fmt.Errorf("插件 %s: apply 必须是函数", def.id)
	}

	name := nameVal.String()
	def.name = name
	if def.version == "" {
		def.version = def.id
	}

	adapter := &jsPluginAdapter{
		host:     h,
		def:      def,
		vm:       vm,
		applyFn:  applyFn,
		handlers: map[string]func(args any) (any, error){},
	}

	// 登记 + 装载
	if err := h.Register(adapter, PluginSourceJS); err != nil {
		return err
	}
	return h.Load(name)
}

// ─── harness 全局注入 ─────────────────────────────────────

// injectHarness 向沙箱注入 harness 对象（defineTool/registerTool/handle）。
// 与 buildContextObject 共用 jsToolToGo 桥；pc 为当前插件的上下文（注册归属）。
func injectHarness(vm *goja.Runtime, adapter *jsPluginAdapter, pc *PluginContext) *goja.Object {
	harnessObj := vm.NewObject()
	harnessObj.Set("defineTool", func(call goja.FunctionCall) goja.Value {
		// 预检工具定义可转换（不注册）；返回原对象
		if _, err := jsToolToGo(vm, call.Argument(0)); err != nil {
			panic(vm.NewGoError(err))
		}
		return call.Argument(0)
	})
	harnessObj.Set("registerTool", func(call goja.FunctionCall) goja.Value {
		tool, err := jsToolToGo(vm, call.Argument(0))
		if err != nil {
			panic(vm.NewGoError(err))
		}
		if pc != nil {
			pc.RegisterTool(tool)
		} else {
			adapter.host.Context().RegisterTool(tool)
		}
		return goja.Undefined()
	})
	harnessObj.Set("handle", func(call goja.FunctionCall) goja.Value {
		method := call.Argument(0).String()
		fnVal := call.Argument(1)
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			panic(vm.NewTypeError("harness.handle: 第二参数必须是函数"))
		}
		adapter.mu.Lock()
		adapter.handlers[method] = func(args any) (any, error) {
			res, err := fn(goja.Undefined(), vm.ToValue(args))
			if err != nil {
				return nil, err
			}
			return res.Export(), nil
		}
		adapter.mu.Unlock()
		return goja.Undefined()
	})
	vm.Set("harness", harnessObj)
	return harnessObj
}

// ─── JS 工具定义 → Go Tool 桥 ─────────────────────────────

// jsToolToGo 把 JS 工具定义对象转成 *Tool。
// 支持 execute: (args) => result | Promise<result>（result 可为 {text} 或任意 JSON 值）。
func jsToolToGo(vm *goja.Runtime, v goja.Value) (*Tool, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, fmt.Errorf("工具定义为空")
	}
	obj := v.ToObject(vm)
	name := obj.Get("name")
	if name == nil || name.String() == "" {
		return nil, fmt.Errorf("工具缺少 name")
	}
	desc := ""
	if d := obj.Get("description"); d != nil {
		desc = d.String()
	}
	params := map[string]any{"type": "object", "properties": map[string]any{}}
	if p := obj.Get("parameters"); p != nil && !goja.IsUndefined(p) && !goja.IsNull(p) {
		if m, ok := p.Export().(map[string]any); ok {
			params = m
		}
	}
	execVal := obj.Get("execute")
	if execVal == nil || goja.IsUndefined(execVal) || goja.IsNull(execVal) {
		return nil, fmt.Errorf("工具 %s 缺少 execute 函数", name.String())
	}
	execFn, ok := goja.AssertFunction(execVal)
	if !ok {
		return nil, fmt.Errorf("工具 %s execute 必须是函数", name.String())
	}

	handler := func(ctx context.Context, args map[string]any) (string, error) {
		res, err := execFn(goja.Undefined(), vm.ToValue(args))
		if err != nil {
			return "", fmt.Errorf("JS 工具 %s 执行失败: %v", name.String(), jsErrorText(err))
		}
		return jsResultToText(vm, res)
	}

	return &Tool{
		Name:        name.String(),
		Description: desc,
		Parameters:  params,
		Handler:     handler,
	}, nil
}

// jsResultToText 把 JS 工具执行结果转成 (text, error)。
// 规则（对齐 harness 输出块约定）：
//   - { text }          → text
//   - { error }         → error
//   - { output, ... }   → JSON 序列化（含 output 等字段）
//   - 字符串/数字/布尔  → String()
//   - Promise           → 等其 resolve（goja 同步 drain）
func jsResultToText(vm *goja.Runtime, v goja.Value) (string, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return "", nil
	}
	// Promise：调用 then 同步取结果
	if o, ok := v.(*goja.Object); ok {
		if then := o.Get("then"); then != nil {
			if thenFn, ok := goja.AssertFunction(then); ok {
				var got goja.Value
				var gotErr error
				vm.Set("__pResolve", func(call goja.FunctionCall) goja.Value {
					got = call.Argument(0)
					return goja.Undefined()
				})
				vm.Set("__pReject", func(call goja.FunctionCall) goja.Value {
					gotErr = fmt.Errorf("%s", call.Argument(0).String())
					return goja.Undefined()
				})
				if _, err := thenFn(v, vm.Get("__pResolve"), vm.Get("__pReject")); err != nil {
					return "", err
				}
				// drain 微任务队列（Promise.then 回调在 job 中执行）
				_, _ = vm.RunString("")
				if gotErr != nil {
					return "", gotErr
				}
				if got != nil {
					return jsResultToText(vm, got)
				}
				return "", nil
			}
		}
	}

	exp := v.Export()
	switch t := exp.(type) {
	case map[string]any:
		if txt, ok := t["text"].(string); ok {
			return txt, nil
		}
		if e, ok := t["error"].(string); ok {
			return "", fmt.Errorf("%s", e)
		}
		if b, err := json.Marshal(t); err == nil {
			return string(b), nil
		}
	case string:
		return t, nil
	case bool:
		if t {
			return "true", nil
		}
		return "false", nil
	}
	if b, err := json.Marshal(exp); err == nil {
		return string(b), nil
	}
	return fmt.Sprint(exp), nil
}

// ─── 工具函数 ─────────────────────────────────────────────

// jsErrorText 提取 goja 异常的文本（含堆栈首行）。
func jsErrorText(err error) string {
	if err == nil {
		return ""
	}
	if ex, ok := err.(*goja.Exception); ok {
		s := ex.String()
		if strings.Contains(s, "at <eval>") {
			return s
		}
		return s
	}
	return err.Error()
}

// jsConsoleArg 格式化 console 参数。
func jsConsoleArg(v goja.Value) string {
	if v == nil {
		return "<nil>"
	}
	exp := v.Export()
	if b, err := json.Marshal(exp); err == nil {
		s := string(b)
		if len(s) < 300 {
			return s
		}
		return s[:300] + "..."
	}
	return fmt.Sprint(exp)
}

func strInSlice(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case int32:
		return int64(t)
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	}
	return 0
}

// ─── 插件宿主：JS 定义管理 ────────────────────────────────

// dynSeq 动态插件 id 序号（dyn-<n>）。
var dynSeq atomic.Uint64

// DefineJS 登记一个 JS 动态插件定义（cordis_define；不装载）。
// 返回分配的 dyn id。源码语言自动探测（TS 类型注解会经内置编译器转译）。
func (h *PluginHost) DefineJS(code, purpose string) (string, error) {
	return h.DefineJSCode(code, "", purpose)
}

// DefineJSCode 登记动态插件定义，language 显式指定源码语言：
// "js" | "ts" | ""（自动探测）。TS 源码经内置 esbuild 编译器转译后再预检。
func (h *PluginHost) DefineJSCode(code, language, purpose string) (string, error) {
	return h.DefineJSCodeDir(code, language, purpose, "")
}

// DefineJSCodeDir 同 DefineJSCode，额外支持多文件 bundle：
// dir 非空时，含 import 的源码按 dir 解析相对导入（esbuild Build 内联打包，
// 非相对包导入 mock 空模块），插件需 export default 导出插件对象。
func (h *PluginHost) DefineJSCodeDir(code, language, purpose, dir string) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("插件代码为空")
	}
	lang := detectPluginLanguage(code, language)
	js, err := compilePluginSource(code, lang, "cordis-dyn.ts", dir)
	if err != nil {
		return "", fmt.Errorf("插件编译失败: %v", err)
	}
	// 语法预检：compile-only（对齐 harness precheckCode）
	if _, err := goja.Compile("cordis-dyn.js", "(async () => {\n"+js+"\n})()", false); err != nil {
		return "", fmt.Errorf("插件语法错误: %v", jsErrorText(err))
	}
	id := fmt.Sprintf("dyn-%d", dynSeq.Add(1))
	def := &jsPluginDef{
		id:        id,
		purpose:   purpose,
		code:      js, // 存转译后的 JS（运行时 goja 直接执行）
		lang:      lang,
		version:   id,
		createdAt: time.Now(),
	}
	h.mu.Lock()
	h.defs[id] = def
	h.mu.Unlock()
	return id, nil
}

// GetJSDef 取 JS 定义。
func (h *PluginHost) GetJSDef(id string) (*jsPluginDef, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	d, ok := h.defs[id]
	return d, ok
}

// RemoveJSDef 删除 JS 定义（cordis_undefine 用；先停再删）。
func (h *PluginHost) RemoveJSDef(id string) error {
	h.mu.RLock()
	def, ok := h.defs[id]
	h.mu.RUnlock()
	if !ok {
		return fmt.Errorf("插件定义不存在: %s", id)
	}
	_ = h.Unload(def.name)
	h.mu.Lock()
	delete(h.defs, id)
	h.mu.Unlock()
	return nil
}

// JSDefs 全部 JS 定义（按 id 排序）。
func (h *PluginHost) JSDefs() []*jsPluginDef {
	h.mu.RLock()
	defer h.mu.RUnlock()
	defs := make([]*jsPluginDef, 0, len(h.defs))
	for _, d := range h.defs {
		defs = append(defs, d)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].id < defs[j].id })
	return defs
}
