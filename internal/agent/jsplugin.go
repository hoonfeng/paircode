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
//       CordisApi → 内置真 cordis 运行时（@cordisjs/core bundle）：new CordisApi.api.Context()
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
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hoonfeng/paircode/internal/core"
	"wb-ui/goja"
)

// ─── 内置 cordis 运行时（CordisApi 全局）────────────────────

// cordisBundleJS @cordisjs/core 3.18.1 经 esbuild bundle 的 IIFE 单文件
// （--platform=neutral，无 require/process/Buffer）。沙箱执行后全局挂 CordisApi：
// 插件代码可直接 `new CordisApi.api.Context()` 建真 cordis app，跑 cordis 生态
// 多插件协作（app.plugin/ctx.set/get 服务/ctx.on 事件）。重建方法见
// .pair/project-info/关键点/修复记录-cordis核心goja验证+trap对齐2026-08-15.md。
//
//go:embed assets/cordis.bundle.js
var cordisBundleEmbed string

// cordisBundleSource 返回 cordis 运行时源码。★ 资源外置（主程序只保留框架）：
// 优先读 <exe 目录>/.pair/assets/runtime/cordis.bundle.js（可独立更新替换），
// 缺失回退内嵌 embed（单文件分发兜底）。
func cordisBundleSource() string {
	if s, ok := LoadRuntimeAssetString("cordis.bundle.js", cordisBundleEmbed); ok {
		return s
	}
	return cordisBundleEmbed
}

// ─── JS 执行超时防护（goja Interrupt）─────────────────────

// JS 执行超时值：防插件死循环卡死进程（goja RunString/函数调用在 JS 代码
// 内可被 Interrupt 强制中断；原生 Go 调用如 ctx.fs/web/bash 自身带超时）。
// var 而非 const：测试中可调小验证超时路径。
var (
	jsEvalTimeout     = 5 * time.Second  // 插件代码求值（RunString）
	jsApplyTimeout    = 5 * time.Second  // apply(ctx, config)
	jsToolTimeout     = 30 * time.Second // 工具 execute
	jsHandlerTimeout  = 10 * time.Second // harness.handle 方法
	jsCallbackTimeout = 5 * time.Second  // 事件/timer 回调
)

// jsTimeoutErr VM 执行超时标记（vm.Interrupt 携带值）。
var jsTimeoutErr = errors.New("JS 执行超时（疑似死循环，已强制中断）")

// runJSWithTimeout 在 vm 上带超时执行同步 JS 调用 fn。
// 超时经 vm.Interrupt 在 JS 指令边界强制中断（返回 *goja.InterruptedError，
// Value() == jsTimeoutErr）。线程安全：Interrupt 可从其他 goroutine 调用。
// fn 正常返回后清除 interrupt flag，避免与超时 goroutine 竞争污染下一次调用。
func runJSWithTimeout(vm *goja.Runtime, timeout time.Duration, fn func() error) error {
	if timeout <= 0 {
		return fn()
	}
	stopped := make(chan struct{})
	go func() {
		select {
		case <-time.After(timeout):
			vm.Interrupt(jsTimeoutErr)
		case <-stopped:
		}
	}()
	err := fn()
	close(stopped)
	vm.ClearInterrupt() // 竞态消除：JS 结束后 goroutine 若已置位 interrupt flag，清除之
	return err
}

// isJSTimeout 判断 err 是否为 runJSWithTimeout 的超时中断。
func isJSTimeout(err error) bool {
	var ie *goja.InterruptedError
	if errors.As(err, &ie) {
		return ie.Value() == jsTimeoutErr
	}
	return errors.Is(err, jsTimeoutErr)
}

// ─── JS 插件定义 ───────────────────────────────────────────

// jsPluginDef 一个 JS 动态插件定义（cordis_define 登记；进程内存，不落盘）。
//
// ★ 版本化 package 模型（对齐 harness registry.ts）：pluginId 是稳定插件身份
// （跨版本不变，默认=首次定义的 dyn id），packageId 是本次定义（不可变）；
// 同一插件多次 define → 同一 pluginId 下追加版本（pluginVersions 链）。
type jsPluginDef struct {
	id         string         // dyn-<n>（package 精确 id，defs map 的 key）
	pluginId   string         // 稳定插件身份（跨版本；默认 = id）
	packageId  string         // pkg-<n>（本次定义不可变版本标识）
	lang       string         // 源码语言 "js" | "ts"（登记时探测/指定；code 存转译后 JS）
	name       string         // 插件名（默认取代码返回的 name；函数形态取函数名或 id）
	purpose    string         // 用途说明
	code       string         // host 半代码（async 函数体，return 插件对象/函数）
	clientCode string         // client 半代码（浏览器端执行；可为空=纯 host 插件）
	version    string         // 版本号（v1/v2/…；默认 = 首次定义 v1）
	provides   []string       // 提供服务的键（插件运行时从 ctx.provide 收集）
	inject     []string       // 插件声明的硬依赖服务（apply 前校验宿主是否提供）
	config     map[string]any // 插件配置（cordis_run 传入，apply(ctx, config) 第二参）
	isFunc     bool           // 函数形态插件（export 为 (ctx, config) => void）
	scope      string         // 生效作用域："global"=全局插件（UI 类，跨工作区；存 <InstallDir>/.pair/plugins/dynamic.json，独立于工具集）；""/"project"=项目插件（工作区工具集 dynamic，按工作区加载）
	dir        string         // ★ 插件目录（磁盘插件包：<InstallDir>/.pair/plugins/<name>/；cordis_define dir 参数）；ctx.binary 服务据此定位 bin/<name>.exe 与 assets/
	createdAt  time.Time

	// ★ 状态机与运行诊断（对齐 harness CordisRunStatus + CordisRunDiagnostic）
	status     PluginState // stopped/running/waiting/rejected/failed/cancelled
	waitingFor []string    // status=waiting 时缺的服务清单
	lastError  string      // 最近一次装载失败原因
	diag       []string    // 运行诊断（阶段记录，最新在后）
	console    []string    // 本次装载的 console 输出（log/info/warn/debug/error；cordis_run 返回时附加）
}

// setStatus 更新定义状态（线程安全；h.mu 保护）。
func (d *jsPluginDef) setStatus(s PluginState, waitingFor []string) {
	if d == nil {
		return
	}
	d.status = s
	d.waitingFor = waitingFor
}

// addDiag 追加一条运行诊断（线程安全；h.mu 保护）。
func (d *jsPluginDef) addDiag(line string) {
	if d == nil {
		return
	}
	d.diag = append(d.diag, line)
	if len(d.diag) > 20 {
		d.diag = d.diag[len(d.diag)-20:]
	}
}

// addConsole 追加一条插件 console 输出（本次装载捕获；上限 30 条防刷屏）。
// 宿主 stdout 流对模型不可见，必须捕获进 def.console 供 cordis_run 返回展示。
func (d *jsPluginDef) addConsole(line string) {
	if d == nil {
		return
	}
	d.console = append(d.console, line)
	if len(d.console) > 30 {
		d.console = d.console[len(d.console)-30:]
	}
}

// ConsoleText 插件本次装载的 console 输出文本（空=无输出；供 cordis_run 返回附加）。
func (d *jsPluginDef) ConsoleText() string {
	if d == nil || len(d.console) == 0 {
		return ""
	}
	return strings.Join(d.console, "\n")
}

// Name 插件名（公开访问；jsPluginDef 字段私有）。
func (d *jsPluginDef) Name() string { return d.name }

// ─── JS 插件适配器 ─────────────────────────────────────────

// jsPluginAdapter 把 goja 里求值出的 JS 插件对象适配为 Plugin。
type jsPluginAdapter struct {
	host    *PluginHost
	def     *jsPluginDef
	vm      *goja.Runtime
	applyFn goja.Callable

	mu       sync.Mutex
	handlers map[string]func(args any) (any, error) // harness.handle 注册的方法

	timersMu   sync.Mutex
	timers     []func() // 活动 timer 的取消函数（Unload 时统一清理）
	cleanupsMu sync.Mutex
	cleanups   []func() // 其他 JS 侧资源撤销函数（如 ctx.provide 的服务撤销）
}

// withLock 在 VM 执行锁保护下运行 fn：timer 回调、事件回调、工具 execute
// 等可能从其他 goroutine 进入 JS 的入口必须经此调用（goja 非并发安全，
// 见 wb-ui/goja Runtime.Lock/Unlock）。
func (p *jsPluginAdapter) withLock(fn func()) {
	p.vm.Lock()
	defer p.vm.Unlock()
	fn()
}

// addTimer 登记一个 timer 取消函数（stopTimers 统一清理）。
func (p *jsPluginAdapter) addTimer(cancel func()) {
	p.timersMu.Lock()
	defer p.timersMu.Unlock()
	p.timers = append(p.timers, cancel)
}

// stopTimers 取消全部活动 timer（插件卸载时调用，防泄漏）。
func (p *jsPluginAdapter) stopTimers() {
	p.timersMu.Lock()
	timers := p.timers
	p.timers = nil
	p.timersMu.Unlock()
	for _, c := range timers {
		c()
	}
}

// addCleanup 登记一个 JS 侧资源撤销函数（如 ctx.provide 的服务撤销）。
func (p *jsPluginAdapter) addCleanup(fn func()) {
	p.cleanupsMu.Lock()
	defer p.cleanupsMu.Unlock()
	p.cleanups = append(p.cleanups, fn)
}

// cleanupJS 插件卸载时回收全部 JS 侧资源：timer + 服务撤销 + 其他清理。
func (p *jsPluginAdapter) cleanupJS() {
	p.stopTimers()
	p.cleanupsMu.Lock()
	fns := p.cleanups
	p.cleanups = nil
	p.cleanupsMu.Unlock()
	for _, fn := range fns {
		func() {
			defer func() { _ = recover() }()
			fn()
		}()
	}
}

// Name 实现 Plugin。
func (p *jsPluginAdapter) Name() string { return p.def.name }

// Apply 实现 Plugin：创建绑定本插件上下文的 ctx 沙箱对象，调用 JS apply(ctx, config)。
func (p *jsPluginAdapter) Apply(pc *PluginContext) error {
	ctxObj, err := p.buildContextObject(pc)
	if err != nil {
		return err
	}
	// 插件卸载时清理活动 timer + JS 侧资源（防 goroutine/ticker/服务泄漏）
	pc.Effect(func() { p.cleanupJS() })
	// apply(ctx, config)：config 为 cordis_run 传入的插件配置（无则 undefined）
	configVal := goja.Undefined()
	if p.def.config != nil {
		configVal = p.vm.ToValue(p.def.config)
	}
	applyErr := runJSWithTimeout(p.vm, jsApplyTimeout, func() error {
		_, err := p.applyFn(goja.Undefined(), ctxObj, configVal)
		return err
	})
	if applyErr != nil {
		if isJSTimeout(applyErr) {
			return fmt.Errorf("JS 插件 %s apply 执行超时（%.0fs，疑似死循环，已强制中断）", p.def.name, jsApplyTimeout.Seconds())
		}
		return fmt.Errorf("JS 插件 %s apply 执行失败: %v", p.def.name, jsErrorText(applyErr))
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
		p.addCleanup(cancel) // 卸载时撤销服务
		// ★ D3：新服务出现 → 激活等待该服务的插件（inject 等待语义）
		p.host.retryWaiting(name)
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
			// 事件可能在任意 goroutine 触发 → 执行锁保护
			p.withLock(func() {
				if err := runJSWithTimeout(p.vm, jsCallbackTimeout, func() error {
					_, err := fn(goja.Undefined(), vm.ToValue(payload))
					return err
				}); err != nil {
					log.Printf("[js-plugin:%s] 事件 %s 回调异常: %v", p.def.id, evt, err)
				}
			})
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
	// ctx.emit(name, payload)：广播事件（对齐 cordis ctx.emit）。
	// ui:/client: 前缀事件经宿主事件总线自动转发浏览器 client 半。
	ctxObj.Set("emit", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		var payload any
		if !goja.IsUndefined(call.Argument(1)) && !goja.IsNull(call.Argument(1)) {
			payload = call.Argument(1).Export()
		}
		pc.Emit(name, payload)
		return goja.Undefined()
	})

	// ctx.timeout(fn, ms) / ctx.interval(fn, ms)：受控定时器（沙箱纪律：
	// 不暴露全局 setTimeout，定时能力经 ctx 提供）。回调在 VM 执行锁保护下
	// 运行（与主执行流互斥）。返回取消函数；插件卸载时统一清理。
	// 回调抛错不崩 goroutine（recover 吞掉并 console.error）。
	jsTimer := func(repeat bool) func(call goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			fnVal := call.Argument(0)
			fn, ok := goja.AssertFunction(fnVal)
			if !ok {
				panic(vm.NewTypeError("ctx.timeout/interval: 第一参数必须是函数"))
			}
			ms := call.Argument(1).ToInteger()
			if ms < 0 {
				ms = 0
			}
			d := time.Duration(ms) * time.Millisecond
			fire := func() {
				p.withLock(func() {
					if err := runJSWithTimeout(p.vm, jsCallbackTimeout, func() error {
						_, err := fn(goja.Undefined())
						return err
					}); err != nil {
						log.Printf("[js-plugin:%s] timer 回调异常: %v", p.def.id, err)
					}
				})
			}
			var cancel func()
			if repeat {
				ticker := time.NewTicker(d)
				stop := make(chan struct{})
				done := make(chan struct{})
				cancel = func() {
					select {
					case <-stop:
					default:
						close(stop)
					}
				}
				p.addTimer(cancel)
				go func() {
					defer close(done)
					for {
						select {
						case <-stop:
							ticker.Stop()
							return
						case <-ticker.C:
							fire()
						}
					}
				}()
			} else {
				t := time.AfterFunc(d, fire)
				cancel = func() { t.Stop() }
				p.addTimer(cancel)
			}
			return vm.ToValue(func(call goja.FunctionCall) goja.Value {
				cancel()
				return goja.Undefined()
			})
		}
	}
	ctxObj.Set("timeout", jsTimer(false))
	ctxObj.Set("interval", jsTimer(true))

	// ctx.tools.register(toolDef)
	toolsObj := vm.NewObject()
	toolsObj.Set("register", func(call goja.FunctionCall) goja.Value {
		tool, tErr := jsToolToGo(vm, call.Argument(0), p.withLock)
		if tErr != nil {
			panic(vm.NewGoError(tErr))
		}
		if rErr := pc.RegisterTool(tool); rErr != nil {
			panic(vm.NewGoError(rErr))
		}
		return goja.Undefined()
	})
	toolsObj.Set("list", func(call goja.FunctionCall) goja.Value {
		names := pc.Tools.Names()
		return vm.ToValue(names)
	})
	ctxObj.Set("tools", toolsObj)

	// ctx.hostTool(name, args)：执行宿主存档工具（Go 实现库）。
	// 迁移模式（2026-08-16）：磁盘工具插件注册同名工具接管 agent 可见面，
	// execute 内可调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：
	// 编排在插件、能力在宿主）。args 为对象（工具参数），返回结果字符串；
	// 宿主无此执行器 → 抛错。ctx.hostToolMeta(name) 返回宿主工具元数据
	// （name/description/parameters）供插件声明 schema 时对齐。
	hostToolObj := vm.NewObject()
	hostToolObj.Set("exec", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if name == "" {
			panic(vm.NewTypeError("ctx.hostTool.exec: 工具名不能为空"))
		}
		var args map[string]any
		if v := call.Argument(1); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			if m, ok := v.Export().(map[string]any); ok {
				args = m
			} else {
				panic(vm.NewTypeError("ctx.hostTool.exec: 第二参数必须是参数对象"))
			}
		}
		out, hErr := ExecuteHostTool(name, args)
		if hErr != nil {
			panic(vm.NewGoError(hErr))
		}
		return vm.ToValue(out)
	})
	hostToolObj.Set("meta", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		t, ok := HostToolMeta(name)
		if !ok {
			return goja.Null()
		}
		meta := map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"usageGuide":  t.UsageGuide,
			"category":    t.Category,
			"parameters":  t.Parameters,
			"readOnly":    t.ReadOnly,
		}
		return vm.ToValue(meta)
	})
	hostToolObj.Set("names", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(HostToolNames())
	})
	ctxObj.Set("hostTool", hostToolObj)

	// ctx.binary — 独立插件二进制服务（★ 依赖 Go 内核的工具独立成插件目录下的
	// 单独二进制项目，主程序只做框架调度）。协议：
	//   ctx.binary.exec(tool, args[, opts]) → text
	//     stdin  JSON {tool, args, root} → stdout JSON {ok, text} | {ok:false, error}
	//     二进制约定：<插件目录>/bin/<插件名>.exe（Windows；其它平台无后缀）
	//     opts.bin 可指定其它二进制文件名（跨插件共用统一二进制，如 {bin:"tool-binary"}）
	//     opts.timeout 毫秒（默认 60s）
	//     资源约定：<插件目录>/assets/（ctx.binary.dir() 返回插件目录，JS 可拼接）
	//   ctx.binary.dir() → 插件目录绝对路径
	// 未声明插件目录（非磁盘插件）时抛错。
	binaryObj := vm.NewObject()
	binaryObj.Set("dir", func(call goja.FunctionCall) goja.Value {
		if p.def.dir == "" {
			panic(vm.NewTypeError("ctx.binary.dir: 插件目录未知（仅磁盘插件包可用；cordis_define 可传 dir 参数声明源码目录）"))
		}
		return vm.ToValue(p.def.dir)
	})
	binaryObj.Set("exec", func(call goja.FunctionCall) goja.Value {
		if p.def.dir == "" {
			panic(vm.NewTypeError("ctx.binary.exec: 插件目录未知（仅磁盘插件包可用；cordis_define 可传 dir 参数声明源码目录）"))
		}
		tool := call.Argument(0).String()
		if tool == "" {
			panic(vm.NewTypeError("ctx.binary.exec: 工具名不能为空"))
		}
		args := map[string]any{}
		if a := call.Argument(1); !goja.IsUndefined(a) && !goja.IsNull(a) {
			if m, ok := a.Export().(map[string]any); ok {
				args = m
			}
		}
		timeout := 60 * time.Second
		binName := ""
		if o := call.Argument(2); !goja.IsUndefined(o) && !goja.IsNull(o) {
			if om, ok := o.Export().(map[string]any); ok {
				if t, ok := om["timeout"]; ok {
					// ★ 2026-08-17 修复：goja 对 JS 整数字面量 Export 为 int64（非 float64），
					// 原仅断言 float64 会让 {timeout: 86400000} 等整数超时静默失效 →
					// 回退 60s 默认，长命令（>60s）被外层提前 kill（表现为任务被中断）。
					var msNum float64
					switch n := t.(type) {
					case float64:
						msNum = n
					case int64:
						msNum = float64(n)
					case int:
						msNum = float64(n)
					case json.Number:
						msNum, _ = n.Float64()
					}
					if msNum > 0 {
						timeout = time.Duration(msNum) * time.Millisecond
					}
				}
				// opts.bin 指定二进制文件名（相对插件目录 bin/，缺省=插件名）。
				// 支持跨插件共用统一二进制（如 {bin:"tool-binary"} 承载多组工具）。
				if b, ok := om["bin"]; ok {
					if s, ok := b.(string); ok && s != "" {
						binName = s
					}
				}
			}
		}
		if binName == "" {
			binName = p.def.name
		}
		exeName := binName
		if runtime.GOOS == "windows" && !strings.HasSuffix(exeName, ".exe") {
			exeName += ".exe"
		}
		// 定位二进制：opts.bin 非空 = 跨插件共用统一二进制（在 <插件根>/<bin>/bin/ 下）；
		// 缺省 = 本插件目录 bin/<插件名>.exe。
		var exePath string
		if binName != p.def.name {
			exePath = filepath.Join(filepath.Dir(p.def.dir), binName, "bin", exeName)
		} else {
			exePath = filepath.Join(p.def.dir, "bin", exeName)
		}
		if _, err := os.Stat(exePath); err != nil {
			panic(vm.NewGoError(fmt.Errorf("ctx.binary.exec: 插件二进制不存在 %s（编译：go build -o %s ./plugins-src/plugins/<name>）", exePath, exePath)))
		}
		reqJSON, _ := json.Marshal(map[string]any{"tool": tool, "args": args, "root": pc.WorkspaceRoot})
		ctxTO, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd := exec.CommandContext(ctxTO, exePath)
		cmd.Stdin = strings.NewReader(string(reqJSON))
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
		if err := cmd.Run(); err != nil {
			if ctxTO.Err() == context.DeadlineExceeded {
				panic(vm.NewGoError(fmt.Errorf("ctx.binary.exec: %s 执行超时（%.0fs）", tool, timeout.Seconds())))
			}
			panic(vm.NewGoError(fmt.Errorf("ctx.binary.exec: %s 执行失败: %v（stderr: %s）", tool, err, strings.TrimSpace(errBuf.String()))))
		}
		var resp struct {
			OK    bool   `json:"ok"`
			Text  string `json:"text"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal(outBuf.Bytes(), &resp); err != nil {
			panic(vm.NewGoError(fmt.Errorf("ctx.binary.exec: %s 响应解析失败: %v（输出: %.200s）", tool, err, outBuf.String())))
		}
		if !resp.OK {
			panic(vm.NewGoError(fmt.Errorf("ctx.binary.%s: %s", tool, resp.Error)))
		}
		// ★ 2026-08-17 修复：外部进程执行可远长于 jsHandlerTimeout（默认 10s）——
		// runJSWithTimeout 的 timer 触发 vm.Interrupt 只置 flag，JS 阻塞在本原生调用
		// 期间不生效；命令（如打包）在 Go 侧成功跑完后，JS 恢复执行才撞上 flag，
		// 被误判「handler 执行超时（疑似死循环）」（表现为任务成功却报超时失败）。
		// 此处 ClearInterrupt：进程执行等待不计入 JS 超时检测；纯 JS 死循环不经过
		// 本调用，仍会被 Interrupt 在指令边界正常中断。
		vm.ClearInterrupt()
		return vm.ToValue(map[string]any{"text": resp.Text})
	})
	ctxObj.Set("binary", binaryObj)

	// ctx.http：HTTP 接口插件化服务（对齐 harness webServer 路由注册）。
	//   ctx.http.register(method, path, fn) → unregister()
	//     method: GET/POST/PUT/DELETE/…；path: 绝对路径，"/*" 结尾=前缀匹配
	//     fn(req) → resp：req = {method, path, query, headers, body}
	//                   resp = {status, body, headers} 或字符串
	//     重复 (method, path) 注册报错；插件卸载自动注销；宿主 mux 路由优先保留
	//   （未命中插件路由才走内置 /api/* 与静态文件，插件路由在 mux 之前拦截）。
	// ★ ctx.http：HTTP 接口插件化（旧形态，向后兼容）——每个插件都可注册
	//   自己的接口：接口定义（method+path）与处理逻辑（fn）都在插件 JS 中，
	//   服务能力走 ctx.fs/ctx.web/ctx.tools/ctx.process 等 Go 服务。
	//   - register(method, path, fn)：fn 支持 async（返回 Promise 同步 drain）；
	//     fn(req) → string | {status, body, headers} | 任意 JSON 值；req 含
	//     method/path/query/headers/body 与 json() 便捷解析。
	//     返回 disposer 函数（取消注册）。重复 (method, path) 注册报错（装配契约）。
	//   - list()：已注册路由清单（含内核接口经 core-api 安装的条目）。
	//   - 插件卸载自动注销全部路由（addCleanup）。
	// ★ 2026-08-18：新插件请用 ctx.webServer（对齐 harness dsh-host-webserver：
	//   register({kind, path, handler})，handler 为 Node 风格 (req, res)）。
	//   ctx.http 保留以兼容现有插件（web-api 等）；二者共用 ext 路由表。
	httpObj := vm.NewObject()
	httpObj.Set("register", func(call goja.FunctionCall) goja.Value {
		method := call.Argument(0).String()
		path := call.Argument(1).String()
		fnVal := call.Argument(2)
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			panic(vm.NewTypeError("ctx.http.register: 第三参数必须是处理函数 fn(req) → resp"))
		}
		handler := func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			headers := map[string]string{}
			for k, vs := range r.Header {
				if len(vs) > 0 {
					headers[k] = vs[0]
				}
			}
			// req 对象（goja 构造，附 json() 便捷解析）
			reqObj := vm.NewObject()
			reqObj.Set("method", r.Method)
			reqObj.Set("path", r.URL.Path)
			reqObj.Set("query", r.URL.RawQuery)
			reqObj.Set("headers", headers)
			reqObj.Set("body", string(body))
			reqObj.Set("json", func(call goja.FunctionCall) goja.Value {
				var out any
				if err := json.Unmarshal(body, &out); err != nil {
					panic(vm.NewGoError(fmt.Errorf("req.json(): 请求体不是合法 JSON: %w", err)))
				}
				return vm.ToValue(out)
			})
			var (
				ret     goja.Value
				callErr error
			)
			p.withLock(func() {
				v, err := fn(goja.Undefined(), reqObj)
				if err == nil {
					// ★ async handler：返回值若是 Promise，同步 drain 取结果
					v, err = awaitJSValue(vm, v)
				}
				ret, callErr = v, err
			})
			if callErr != nil {
				http.Error(w, "ctx.http handler 执行失败: "+jsErrorText(callErr), http.StatusInternalServerError)
				return
			}
			status := http.StatusOK
			respBody := ""
			respHeaders := map[string]string{}
			if ret != nil && !goja.IsUndefined(ret) && !goja.IsNull(ret) {
				if s, ok := ret.Export().(string); ok {
					respBody = s
				} else if obj := ret.ToObject(vm); obj != nil {
					if v := obj.Get("status"); v != nil {
						status = int(v.ToInteger())
					}
					if v := obj.Get("body"); v != nil {
						respBody = v.String()
					}
					if v := obj.Get("headers"); v != nil {
						if m, ok := v.Export().(map[string]any); ok {
							for k, vv := range m {
								respHeaders[k] = fmt.Sprint(vv)
							}
						}
					}
				} else {
					respBody = ret.String()
				}
			}
			for k, v := range respHeaders {
				w.Header().Set(k, v)
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(respBody))
		}
		dispose, err := RegisterExtRoute(method, path, handler)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		p.addCleanup(dispose) // 插件卸载自动注销
		return vm.ToValue(func(goja.FunctionCall) goja.Value {
			dispose()
			return goja.Undefined()
		})
	})
	// ctx.http.list()：已注册路由清单（含内核安装条目；插件自描述/调试用）
	httpObj.Set("list", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(RegisteredExtRoutes())
	})
	ctxObj.Set("http", httpObj)

	// ── ctx.webServer：HTTP 接口插件化（对齐 harness dsh-host-webserver）──
	// ★ 2026-08-18：兼容 dsh 插件生态——与参考项目同形态：
	//   ctx.webServer.register({ kind, path, handler }) → disposer
	//   - kind: 'exact'（逐字匹配）| 'prefix'（path 与 path/<anything>）
	//   - handler: (req, res) => void | Promise<void> —— Node 风格，完全持有
	//     响应生命周期：res.writeHead(status, headers) / res.setHeader /
	//     res.write(chunk) / res.end(chunk) / res.statusCode 属性赋值；
	//     req: { method, url, path, query, headers, body, json(), on('data'|'end') }。
	//   - 不区分 HTTP 方法（对齐 harness：method 由 handler 自行判断 req.method）
	//   - 重复 (kind, path) 报错（装配层契约）；插件卸载自动注销。
	//   - 兼容便捷形态 register(kind, path, handler)（三参数）。
	//   - 另兼容旧 ctx.http 返回对象形态：{ status, body, headers }。
	webServerObj := vm.NewObject()
	webServerObj.Set("register", func(call goja.FunctionCall) goja.Value {
		var kind, path string
		var fn goja.Callable
		arg0 := call.Argument(0)
		if obj := arg0.ToObject(vm); obj != nil {
			if kv := obj.Get("kind"); kv != nil && !goja.IsUndefined(kv) && !goja.IsNull(kv) {
				if pv := obj.Get("path"); pv != nil && !goja.IsUndefined(pv) && !goja.IsNull(pv) {
					kind, path = kv.String(), pv.String()
					fn, _ = goja.AssertFunction(obj.Get("handler"))
				}
			}
		}
		if fn == nil {
			kind = arg0.String()
			path = call.Argument(1).String()
			fn, _ = goja.AssertFunction(call.Argument(2))
		}
		if kind != "exact" && kind != "prefix" {
			panic(vm.NewTypeError("ctx.webServer.register: kind 必须是 'exact'|'prefix'"))
		}
		if path == "" {
			panic(vm.NewTypeError("ctx.webServer.register: path 不能为空"))
		}
		if fn == nil {
			panic(vm.NewTypeError("ctx.webServer.register: handler 必须是函数 (req, res)"))
		}
		regPath := path
		if kind == "prefix" && !strings.HasSuffix(regPath, "/*") {
			regPath = strings.TrimSuffix(regPath, "/") + "/*"
		}
		handler := p.buildNodeHTTPHandler(fn)
		dispose, err := RegisterExtRouteAny(regPath, handler)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		p.addCleanup(dispose)
		return vm.ToValue(func(goja.FunctionCall) goja.Value {
			dispose()
			return goja.Undefined()
		})
	})
	ctxObj.Set("webServer", webServerObj)

	// ctx.process：后台进程服务（run_background/read_output/kill_process 的能力面）。
	// globalBG 为全局单例（跨 agent 轮次存活）；cwd 相对工作区根解析（越界拦截）。
	processObj := vm.NewObject()
	processObj.Set("runBackground", func(call goja.FunctionCall) goja.Value {
		command := strings.TrimSpace(call.Argument(0).String())
		if command == "" {
			panic(vm.NewTypeError("ctx.process.runBackground: command 不能为空"))
		}
		dir := pc.WorkspaceRoot
		if cwd := call.Argument(1).String(); cwd != "" {
			var err error
			if dir, err = resolvePath(pc.WorkspaceRoot, cwd); err != nil {
				panic(vm.NewGoError(err))
			}
		}
		id, err := globalBG.start(command, dir)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(map[string]any{"id": id})
	})
	processObj.Set("readOutput", func(call goja.FunctionCall) goja.Value {
		id := int(call.Argument(0).ToInteger())
		p := globalBG.get(id)
		if p == nil {
			panic(vm.NewGoError(fmt.Errorf("无此后台进程 id %d", id)))
		}
		out, done, exitErr := p.snapshot()
		return vm.ToValue(map[string]any{
			"output":  out,
			"done":    done,
			"exitErr": exitErr,
			"status":  map[bool]string{true: "已结束", false: "运行中"}[done],
		})
	})
	processObj.Set("kill", func(call goja.FunctionCall) goja.Value {
		id := int(call.Argument(0).ToInteger())
		p := globalBG.get(id)
		if p == nil {
			panic(vm.NewGoError(fmt.Errorf("无此后台进程 id %d", id)))
		}
		if p.cmd != nil && p.cmd.Process != nil {
			killProcessTree(p.cmd.Process.Pid)
		}
		return vm.ToValue(map[string]any{"ok": true, "id": id})
	})
	ctxObj.Set("process", processObj)

	// ctx.loopFactory.register(apply)：注册 agent 循环装配器（对齐 harness setFactory 单槽位）。
	// apply(opts) → overrides | null：
	//   opts = { system, maxIterations, maxContextTokens, autonomous,
	//            maxAutonomousMinutes, checkpointInterval, workspaceRoot, reviewMode }
	//   返回同形状对象时非空字段覆盖默认装配参数（如追加提示词/调迭代上限/切换审核模式）；
	//   返回 null/undefined 表示不改动。注册即替换全局 LoopFactory 单槽位（后注册覆盖先注册），
	//   插件卸载时自动还原默认工厂。真正替换循环内核留给宿主 Go 代码（ReplaceLoopFactory）。
	loopFactoryObj := vm.NewObject()
	loopFactoryObj.Set("register", func(call goja.FunctionCall) goja.Value {
		applyVal := call.Argument(0)
		applyFn, ok := goja.AssertFunction(applyVal)
		if !ok {
			panic(vm.NewTypeError("ctx.loopFactory.register: 参数必须是函数 apply(opts) → overrides"))
		}
		bridge := &jsLoopFactoryBridge{vm: vm, apply: applyFn, plugin: p}
		restore := ReplaceLoopFactory(bridge)
		p.addCleanup(restore)
		p.def.addDiag("注册 agent 循环装配器（LoopFactory 单槽位，卸载自动还原）")
		return goja.Undefined()
	})
	ctxObj.Set("loopFactory", loopFactoryObj)

	// ctx.registerClientMethod(method, fn)：host 半暴露方法给浏览器 client 半
	// 远程调用（D11 invoke RPC；对齐 harness @Remote('invoke') 的方法注册面）。
	// 与 harness.handle 共用存储，但语义显式面向 client 半；浏览器侧经
	// ui.invoke(plugin, method, args) 调用。
	ctxObj.Set("registerClientMethod", func(call goja.FunctionCall) goja.Value {
		method := call.Argument(0).String()
		if method == "" {
			panic(vm.NewTypeError("ctx.registerClientMethod: 方法名不能为空"))
		}
		fnVal := call.Argument(1)
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			panic(vm.NewTypeError("ctx.registerClientMethod: 第二参数必须是函数"))
		}
		registerJSHandler(vm, p, method, fn)
		p.def.addDiag(fmt.Sprintf("注册 client 方法 %s", method))
		return goja.Undefined()
	})

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
	// ctx.systemPrompt.variable({name, provider})：注册 {{name}} 提示词变量（组装时求值）
	sysObj.Set("variable", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
			panic(vm.NewTypeError("ctx.systemPrompt.variable: 需要一个对象 {name, provider}"))
		}
		obj := arg.ToObject(vm)
		name := obj.Get("name").String()
		if name == "" {
			panic(vm.NewTypeError("ctx.systemPrompt.variable: name 不能为空"))
		}
		provFn, ok := goja.AssertFunction(obj.Get("provider"))
		if !ok {
			panic(vm.NewTypeError("ctx.systemPrompt.variable: provider 必须是函数"))
		}
		pc.AddSystemPromptVariable(&PromptVariable{
			Name: name,
			Provider: func() string {
				defer func() {
					if r := recover(); r != nil { // JS provider 抛错 → 本次无值
					}
				}()
				v, err := provFn(goja.Undefined())
				if err != nil || v == nil {
					return ""
				}
				return v.String()
			},
		})
		return goja.Undefined()
	})
	ctxObj.Set("systemPrompt", sysObj)

	// ctx.toolset.registerTemplate({id, title, match?, generate?})：注册工具集构建
	// 模板（插件化：市场/用户插件可提供专属模板，toolset_build 动态组合）。
	// match(profile) 判定适用；generate(profile, requirement) 返回插件定义数组
	// [{name, purpose, code, client?}]。
	toolsetObj := vm.NewObject()
	toolsetObj.Set("registerTemplate", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
			panic(vm.NewTypeError("ctx.toolset.registerTemplate: 需要一个模板对象"))
		}
		obj := arg.ToObject(vm)
		id := obj.Get("id").String()
		title := ""
		if t := obj.Get("title"); t != nil {
			title = t.String()
		}
		if id == "" {
			panic(vm.NewTypeError("ctx.toolset: 模板 id 不能为空"))
		}
		tpl := &ToolsetTemplate{ID: id, Title: title}
		// match/generate 可选（缺省全匹配/空生成）
		if m := obj.Get("match"); m != nil && !goja.IsUndefined(m) && !goja.IsNull(m) {
			if fn, ok := goja.AssertFunction(m); ok {
				tpl.jsMatch = fn
				tpl.jsVM = vm
			}
		}
		if g := obj.Get("generate"); g != nil && !goja.IsUndefined(g) && !goja.IsNull(g) {
			if fn, ok := goja.AssertFunction(g); ok {
				tpl.jsGenerate = fn
				tpl.jsVM = vm
			}
		}
		if tpl.jsGenerate == nil {
			panic(vm.NewTypeError("ctx.toolset: 模板需要 generate(profile, requirement) 函数"))
		}
		tpl.jsLock = p.withLock
		if err := p.host.RegisterTemplate(tpl); err != nil {
			panic(vm.NewGoError(err))
		}
		// 插件卸载时移除模板
		p.addCleanup(func() { p.host.RemoveTemplate(id) })
		return goja.Undefined()
	})
	ctxObj.Set("toolset", toolsetObj)

	// ctx.app：宿主基本信息（可选）
	appObj := vm.NewObject()
	appObj.Set("workspaceRoot", pc.WorkspaceRoot)
	ctxObj.Set("app", appObj)

	// ctx.registerSettings(schema) / ctx.getSettings(key?) / ctx.setSettings(key, value)
	// —— 插件配置注册机制（2026-08-16，见 core/settings_registry.go）：
	//   · registerSettings(schema)：注册本插件的配置段（前端设置面板动态渲染 tab）。
	//     schema = { key?, title?, fields: [{ name, label, type, default?, options?, hint?, group? }] }
	//       key 缺省=插件名（命名空间）；type: text|password|number|checkbox|select|textarea
	//     return { key, value }（当前已存值，合并默认后）。
	//   · getSettings(key?)：读配置值（key 缺省=插件名），返回对象（未设→{}）。
	//   · setSettings(key?, value)：写配置值并持久化（key 缺省=插件名），返回 true。
	//   值存 settings.json 的 pluginSettings[key]，与核心字段隔离；重启保留。
	ctxObj.Set("registerSettings", func(call goja.FunctionCall) goja.Value {
		raw := call.Argument(0).Export()
		m, ok := raw.(map[string]any)
		if !ok {
			panic(vm.NewTypeError("ctx.registerSettings: 参数必须是 schema 对象"))
		}
		key, _ := m["key"].(string)
		if key == "" {
			key = p.def.name
		}
		title, _ := m["title"].(string)
		if title == "" {
			title = key
		}
		fields := []core.SettingField{}
		if farr, ok := m["fields"].([]any); ok {
			for _, fv := range farr {
				fm, ok := fv.(map[string]any)
				if !ok {
					continue
				}
				f := core.SettingField{}
				f.Name, _ = fm["name"].(string)
				f.Label, _ = fm["label"].(string)
				f.Type, _ = fm["type"].(string)
				if f.Type == "" {
					f.Type = "text"
				}
				f.Default = fm["default"]
				f.Hint, _ = fm["hint"].(string)
				f.Group, _ = fm["group"].(string)
				if opts, ok := fm["options"].([]any); ok {
					for _, o := range opts {
						if s, ok := o.(string); ok {
							f.Options = append(f.Options, s)
						}
					}
				}
				if f.Name != "" {
					fields = append(fields, f)
				}
			}
		}
		core.RegisterPluginSettingSchema(core.SettingSchema{Key: key, Title: title, Fields: fields})
		// 返回当前值（默认合并）
		cur := core.Settings.PluginSettingValue(key)
		for k, d := range core.PluginSettingDefaults(key) {
			if _, ok := cur[k]; !ok {
				cur[k] = d
			}
		}
		return vm.ToValue(map[string]any{"key": key, "value": cur})
	})
	ctxObj.Set("getSettings", func(call goja.FunctionCall) goja.Value {
		key := p.def.name
		if a := call.Argument(0); !goja.IsUndefined(a) && !goja.IsNull(a) && a.String() != "" {
			key = a.String()
		}
		return vm.ToValue(core.Settings.PluginSettingValue(key))
	})
	ctxObj.Set("setSettings", func(call goja.FunctionCall) goja.Value {
		key := p.def.name
		if a := call.Argument(0); !goja.IsUndefined(a) && !goja.IsNull(a) && a.String() != "" {
			key = a.String()
		}
		value := call.Argument(1).Export()
		if m, ok := value.(map[string]any); ok {
			if core.Settings.PluginSettings == nil {
				core.Settings.PluginSettings = map[string]map[string]any{}
			}
			core.Settings.PluginSettings[key] = m
			core.Save()
			return vm.ToValue(true)
		}
		panic(vm.NewTypeError("ctx.setSettings: 第二参数必须是对象"))
	})

	// ★ 按 inject 声明注入服务属性（对齐 harness：只读声明过的服务，
	//   可选服务用 ctx.get(name) 判 undefined；未声明即访问为 undefined）
	for _, s := range p.def.inject {
		switch s {
		case "fs":
			ctxObj.Set("fs", p.buildFSService(pc))
		case "web":
			ctxObj.Set("web", p.buildWebService(pc))
		case "bash":
			ctxObj.Set("bash", p.buildBashService(pc))
		case "sse":
			ctxObj.Set("sse", p.buildSSEService(pc))
		case "ws":
			ctxObj.Set("ws", p.buildWSService(pc))
		case "logger":
			ctxObj.Set("logger", p.buildLoggerService())
		case "timer":
			ctxObj.Set("timer", p.buildTimerService(ctxObj))
		case "kernel":
			ctxObj.Set("kernel", p.buildKernelService())
		case "market":
			ctxObj.Set("market", p.buildMarketService())
		}
	}

	return ctxObj, nil
}

// ─── ctx 服务实现（inject 声明后按属性可用） ──────────────

// buildFSService 受限文件服务（ctx.fs）：读写限定在工作区根内
// （resolvePath 越界拦截）。方法同步实现——await 同步值直接通过，
// cordis 插件写法 `await ctx.fs.readFile(...)` 兼容。
func (p *jsPluginAdapter) buildFSService(pc *PluginContext) goja.Value {
	vm := p.vm
	root := pc.WorkspaceRoot
	resolve := func(path string) (string, error) {
		if root == "" {
			return "", fmt.Errorf("ctx.fs: 工作区根为空，无法解析路径 %q", path)
		}
		return resolvePath(root, path)
	}
	fs := vm.NewObject()
	fs.Set("readFile", func(call goja.FunctionCall) goja.Value {
		full, err := resolve(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		b, err := os.ReadFile(full)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(string(b))
	})
	fs.Set("writeFile", func(call goja.FunctionCall) goja.Value {
		full, err := resolve(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		if err := os.WriteFile(full, []byte(call.Argument(1).String()), 0o644); err != nil {
			panic(vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	fs.Set("appendFile", func(call goja.FunctionCall) goja.Value {
		full, err := resolve(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		f, err := os.OpenFile(full, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		if _, err := f.WriteString(call.Argument(1).String()); err != nil {
			f.Close()
			panic(vm.NewGoError(err))
		}
		f.Close()
		return goja.Undefined()
	})
	fs.Set("exists", func(call goja.FunctionCall) goja.Value {
		full, err := resolve(call.Argument(0).String())
		if err != nil {
			return vm.ToValue(false) // 越界视为不存在
		}
		return vm.ToValue(pathExists(full))
	})
	fs.Set("readdir", func(call goja.FunctionCall) goja.Value {
		full, err := resolve(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		entries, err := os.ReadDir(full)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		return vm.ToValue(names)
	})
	fs.Set("stat", func(call goja.FunctionCall) goja.Value {
		full, err := resolve(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		fi, err := os.Stat(full)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(map[string]any{
			"name":  fi.Name(),
			"size":  fi.Size(),
			"isDir": fi.IsDir(),
			"mtime": fi.ModTime().Format(time.RFC3339),
		})
	})
	fs.Set("mkdir", func(call goja.FunctionCall) goja.Value {
		full, err := resolve(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		recursive := call.Argument(1).ToBoolean()
		if recursive {
			err = os.MkdirAll(full, 0o755)
		} else {
			err = os.Mkdir(full, 0o755)
		}
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	fs.Set("rm", func(call goja.FunctionCall) goja.Value {
		full, err := resolve(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		recursive := call.Argument(1).ToBoolean()
		if recursive {
			err = os.RemoveAll(full)
		} else {
			err = os.Remove(full)
		}
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	fs.Set("rename", func(call goja.FunctionCall) goja.Value {
		from, err := resolve(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		to, err := resolve(call.Argument(1).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		if err := os.Rename(from, to); err != nil {
			panic(vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	// grep：正则内容搜索（复用 search_content 实现）。grep(pattern, opts?)
	// opts = {path, glob, case_insensitive, max_results}；返回结果文本（含统计行）。
	fs.Set("grep", func(call goja.FunctionCall) goja.Value {
		pattern := call.Argument(0).String()
		opts := map[string]any{}
		if a := call.Argument(1); a != nil && !goja.IsUndefined(a) && !goja.IsNull(a) {
			if o, ok := a.Export().(map[string]any); ok {
				opts = o
			}
		}
		args := map[string]any{"pattern": pattern}
		for k, v := range opts {
			args[k] = v
		}
		out, err := searchContentHandler(root)(context.Background(), args)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(out)
	})
	// glob：通配符查找文件（复用 search_files 实现）。glob(pattern, opts?)
	// opts = {path, language, max_results}；返回结果文本（含统计行）。
	fs.Set("glob", func(call goja.FunctionCall) goja.Value {
		pattern := call.Argument(0).String()
		opts := map[string]any{}
		if a := call.Argument(1); a != nil && !goja.IsUndefined(a) && !goja.IsNull(a) {
			if o, ok := a.Export().(map[string]any); ok {
				opts = o
			}
		}
		args := map[string]any{"pattern": pattern}
		for k, v := range opts {
			args[k] = v
		}
		out, err := searchFilesHandler(root)(context.Background(), args)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(out)
	})
	return vm.ToValue(fs)
}

// buildWebService HTTP 服务（ctx.web.fetch）：GET 抓取 URL，
// 返回 { ok, status, text }（60s 超时，最大 4MB）。
func (p *jsPluginAdapter) buildWebService(pc *PluginContext) goja.Value {
	vm := p.vm
	web := vm.NewObject()
	web.Set("fetch", func(call goja.FunctionCall) goja.Value {
		url := call.Argument(0).String()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		body, status, err := httpGetBytes(ctx, url, 4<<20)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(map[string]any{
			"ok":     status >= 200 && status < 300,
			"status": status,
			"text":   string(body),
		})
	})
	return vm.ToValue(web)
}

// buildBashService 进程服务（ctx.bash.exec）：执行 shell 命令
// （复用 runShellWithTimeout：120s 超时 + 输出截断），返回 { output, error }。
// 第二参可选 cwd（相对工作区根解析）。
func (p *jsPluginAdapter) buildBashService(pc *PluginContext) goja.Value {
	vm := p.vm
	bsh := vm.NewObject()
	bsh.Set("exec", func(call goja.FunctionCall) goja.Value {
		cmd := call.Argument(0).String()
		dir := pc.WorkspaceRoot
		// 第二参可选 cwd：undefined/null/空串时保持工作区根
		if d := call.Argument(1); d != nil && !goja.IsUndefined(d) && !goja.IsNull(d) && d.String() != "" {
			if resolved, err := resolvePath(pc.WorkspaceRoot, d.String()); err == nil {
				dir = resolved
			}
		}
		out, exitErr := runShellWithTimeout(context.Background(), cmd, dir)
		res := map[string]any{"output": out}
		if exitErr != "" {
			res["error"] = exitErr
		}
		return vm.ToValue(res)
	})
	return vm.ToValue(bsh)
}

// buildSSEService 事件推送服务（ctx.sse）：注册 SSE 实时推送端点。
// 对齐 harness webServer 的事件流形态——插件可向浏览器/外部客户端推送
// 事件流（进度、日志、通知等），宿主统一管理连接生命周期。
//
// JS 用法（inject 声明 'sse'）：
//   const unregister = ctx.sse.register('/api/ext/stream', (emit, params) => {
//     emit('hello', { ts: Date.now() })            // 建立时先推一条
//     push = emit                                  // 保存供外部触发
//     return () => { push = null }                 // 断连清理（可选）
//   })
// emit(event, payload)：payload 会被 JSON 序列化；连接断开后调用抛错。
// handler 在连接建立时于 VM 锁内调用一次（可 await）；返回 cleanup。
func (p *jsPluginAdapter) buildSSEService(pc *PluginContext) goja.Value {
	vm := p.vm
	sse := vm.NewObject()
	sse.Set("register", func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		if path == "" {
			panic(vm.NewGoError(fmt.Errorf("ctx.sse.register: path 不能为空")))
		}
		handlerFn, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			panic(vm.NewGoError(fmt.Errorf("ctx.sse.register: 第二个参数必须是函数 (emit, params) => cleanup")))
		}
		// 宿主 Go 侧 SSEHandler（在 SSE 连接 goroutine 中执行）
		goH := func(params map[string]string, emit func(event string, payload any) error, done <-chan struct{}) func() {
			// JS 侧 emit：跨 goroutine 可调（进 VM 锁；连接断开抛错）
			jsEmit := func(call goja.FunctionCall) goja.Value {
				event := call.Argument(0).String()
				var payload any
				if a := call.Argument(1); a != nil && !goja.IsUndefined(a) && !goja.IsNull(a) {
					payload = a.Export()
				}
				if err := emit(event, payload); err != nil {
					panic(vm.NewGoError(err))
				}
				return goja.Undefined()
			}
			jsEmitFn := vm.ToValue(jsEmit)
			// handler 在 VM 锁内调用一次（同步语义，可 await）
			var cleanupFn func()
			p.withLock(func() {
				ret, err := handlerFn(goja.Undefined(), jsEmitFn, vm.ToValue(params))
				if err != nil {
					fmt.Printf("[js-plugin:%s] ctx.sse %s handler 执行失败: %s\n", p.def.id, path, jsErrorText(err))
					return
				}
				if ret != nil && !goja.IsUndefined(ret) && !goja.IsNull(ret) {
					if retObj, isObj := ret.(*goja.Object); isObj {
						if cf, isFn := goja.AssertFunction(retObj); isFn {
							cleanupFn = func() {
								p.withLock(func() { _, _ = cf(goja.Undefined()) })
							}
							return
						}
					}
					// 非函数返回值：drain Promise（async handler）
					_, _ = jsResultToText(vm, ret)
				}
			})
			return cleanupFn
		}
		disposer, err := RegisterExtSSE(path, goH)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		// 返回 unregister 函数；插件卸载经 addCleanup 自动注销
		unreg := func(call goja.FunctionCall) goja.Value {
			disposer()
			return goja.Undefined()
		}
		p.addCleanup(disposer)
		return vm.ToValue(unreg)
	})
	return vm.ToValue(sse)
}

// buildWSService 双向 WebSocket 服务（ctx.ws）：注册双向实时端点。
// JS 用法（inject 声明 'ws'）：
//   ctx.ws.register(path, (conn, params) => {
//     conn.onMessage((payload) => { conn.send({echo: payload}) })   // 收消息
//     conn.send('hello')                                            // 发消息（跨调用可存 conn）
//     return () => { /* 断连清理 */ }
//   })
// conn 方法：
//   - send(payload)：发送文本帧（string 直传；其他值 JSON 序列化）。
//     连接断开后抛错。
//   - onMessage(fn)：注册消息回调（Go 读循环 → VM 锁内调用）。
//     文本帧尝试 JSON.parse → 对象/数组/数字；解析失败给字符串。
//   - close()：发送 close 帧并关闭连接。
// 生命周期：handler(conn, params) 在连接建立后于 VM 锁内调用一次（可
// await）；返回函数为 cleanup，连接断开时调用。连接断开后 send 抛错。
func (p *jsPluginAdapter) buildWSService(pc *PluginContext) goja.Value {
	vm := p.vm
	ws := vm.NewObject()
	ws.Set("register", func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		if path == "" {
			panic(vm.NewGoError(fmt.Errorf("ctx.ws.register: path 不能为空")))
		}
		handlerFn, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			panic(vm.NewGoError(fmt.Errorf("ctx.ws.register: 第二个参数必须是函数 (conn, params) => cleanup")))
		}
		// 宿主 Go 侧 WS handler（连接建立后在 ServeExtWS goroutine 中执行）
		goH := func(conn *WSConn, params map[string]string, done chan struct{}) func() {
			// JS 侧连接对象
			connObj := vm.NewObject()
			var msgCb goja.Callable // onMessage 回调（读循环触发，VM 锁内调用）

			// send：文本帧（string 直传 / 其他 JSON 序列化）；断开抛错
			connObj.Set("send", func(call goja.FunctionCall) goja.Value {
				var data []byte
				if a := call.Argument(0); a != nil && !goja.IsUndefined(a) && !goja.IsNull(a) {
					exp := a.Export()
					if s, isStr := exp.(string); isStr {
						data = []byte(s)
					} else if b, err := json.Marshal(exp); err == nil {
						data = b
					} else {
						data = []byte(fmt.Sprint(exp))
					}
				}
				if err := conn.WriteTextFrame(data); err != nil {
					panic(vm.NewGoError(fmt.Errorf("ws send 失败: %w", err)))
				}
				return goja.Undefined()
			})
			// close：发 close 帧并关闭
			connObj.Set("close", func(call goja.FunctionCall) goja.Value {
				_ = conn.WriteCloseFrame()
				_ = conn.Close()
				return goja.Undefined()
			})
			// onMessage：注册消息回调（先于读循环启动，无竞态）
			connObj.Set("onMessage", func(call goja.FunctionCall) goja.Value {
				fn, ok := goja.AssertFunction(call.Argument(0))
				if !ok {
					panic(vm.NewGoError(fmt.Errorf("ctx.ws onMessage: 参数必须是函数 (payload) => void")))
				}
				msgCb = fn
				return goja.Undefined()
			})
			// handler 在 VM 锁内调用一次（同步语义，可 await）
			var cleanupFn func()
			p.withLock(func() {
				ret, err := handlerFn(goja.Undefined(), connObj, vm.ToValue(params))
				if err != nil {
					fmt.Printf("[js-plugin:%s] ctx.ws %s handler 执行失败: %s\n", p.def.id, path, jsErrorText(err))
					return
				}
				if ret != nil && !goja.IsUndefined(ret) && !goja.IsNull(ret) {
					if retObj, isObj := ret.(*goja.Object); isObj {
						if cf, isFn := goja.AssertFunction(retObj); isFn {
							cleanupFn = func() {
								p.withLock(func() { _, _ = cf(goja.Undefined()) })
							}
							return
						}
					}
					// 非函数返回值：drain Promise（async handler）
					_, _ = jsResultToText(vm, ret)
				}
			})
			// 读循环 goroutine：读帧 → 锁内调 msgCb；断开 → close(done)
			go func() {
				defer close(done)
				for {
					op, payload, err := conn.ReadFrame()
					if err != nil {
						return
					}
					// 文本帧尝试 JSON 解析 → JS 值；失败给字符串。二进制帧给字符串（UTF-8）。
					var data any
					if op == 0x1 {
						if err := json.Unmarshal(payload, &data); err != nil {
							data = string(payload)
						}
					} else {
						data = string(payload)
					}
					p.withLock(func() {
						if msgCb != nil {
							if _, cbErr := msgCb(goja.Undefined(), vm.ToValue(data)); cbErr != nil {
								fmt.Printf("[js-plugin:%s] ctx.ws %s onMessage 回调失败: %s\n", p.def.id, path, jsErrorText(cbErr))
							}
						}
					})
				}
			}()
			return cleanupFn
		}
		disposer, err := RegisterExtWS(path, goH)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		// 返回 unregister 函数；插件卸载经 addCleanup 自动注销
		unreg := func(call goja.FunctionCall) goja.Value {
			disposer()
			return goja.Undefined()
		}
		p.addCleanup(disposer)
		return vm.ToValue(unreg)
	})
	return vm.ToValue(ws)
}

// buildLoggerService 日志服务（ctx.logger(scope)）：cordis 语义，
// 返回带 scope 的 logger（log/info/warn/debug/error），写透宿主 stdout。
func (p *jsPluginAdapter) buildLoggerService() goja.Value {	vm := p.vm
	loggerFn := func(call goja.FunctionCall) goja.Value {
		scope := call.Argument(0).String()
		tag := fmt.Sprintf("[js-plugin:%s:%s]", p.def.id, scope)
		l := vm.NewObject()
		mk := func(level string) func(goja.FunctionCall) goja.Value {
			return func(call goja.FunctionCall) goja.Value {
				parts := make([]string, len(call.Arguments))
				for i, a := range call.Arguments {
					parts[i] = jsConsoleArg(a)
				}
				log.Printf("%s %s", tag, strings.Join(parts, " "))
				return goja.Undefined()
			}
		}
		for _, m := range []string{"log", "info", "warn", "debug", "error"} {
			l.Set(m, mk(m))
		}
		return l
	}
	return vm.ToValue(loggerFn)
}

// buildTimerService 定时器服务（ctx.timer）：复用 ctx.timeout/ctx.interval。
func (p *jsPluginAdapter) buildTimerService(ctxObj *goja.Object) goja.Value {
	vm := p.vm
	t := vm.NewObject()
	t.Set("timeout", ctxObj.Get("timeout"))
	t.Set("interval", ctxObj.Get("interval"))
	return vm.ToValue(t)
}

// ─── ctx.kernel：内核 API 路由服务（接口插件化） ─────────────
//
// 内置 /api/* 接口的能力（Go handler）保留在宿主内核路由表
// （internal/agent/kernel_api.go），路由挂载权经本服务交给插件：
//   - ctx.kernel.routes()      → 全部内核接口清单（[{key,method,path,desc}]）
//   - ctx.kernel.install(list) → 把清单里指定 key 的路由挂到 ext 表
//     （list = [{key}] 或 [{key,method,path}]；忽略未知 key，返回统计）
//   - ctx.kernel.installed()   → 已安装的 key 列表
//   - ctx.kernel.total()       → 内核表容量
//
// 插件卸载时，install 登记的 disposer 自动摘除路由（接口随插件生灭）。
func (p *jsPluginAdapter) buildKernelService() goja.Value {
	vm := p.vm
	k := vm.NewObject()

	k.Set("routes", func(call goja.FunctionCall) goja.Value {
		// 手动转 map（goja 对 Go struct 用字段名序列化，需按 json tag 给 JS 侧 key）
		metas := KernelAPIRoutes()
		out := make([]map[string]any, 0, len(metas))
		for _, m := range metas {
			out = append(out, map[string]any{
				"key":    m.Key,
				"method": m.Method,
				"path":   m.Path,
				"desc":   m.Desc,
			})
		}
		return vm.ToValue(out)
	})
	k.Set("installed", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(KernelAPIInstalledKeys())
	})
	k.Set("total", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(KernelAPITotal())
	})
	k.Set("install", func(call goja.FunctionCall) goja.Value {
		listVal := call.Argument(0)
		if goja.IsUndefined(listVal) || goja.IsNull(listVal) {
			panic(vm.NewTypeError("ctx.kernel.install: 需要一个路由清单数组（ctx.kernel.routes() 可查）"))
		}
		exported := listVal.Export()
		items, ok := exported.([]any)
		if !ok {
			panic(vm.NewTypeError("ctx.kernel.install: 参数必须是数组 [{key,method,path}, ...]"))
		}
		installed := 0
		missing := 0
		for _, it := range items {
			obj, ok := it.(map[string]any)
			if !ok {
				continue
			}
			key, _ := obj["key"].(string)
			if key == "" {
				continue
			}
			dispose, err := KernelAPIInstall(key)
			if err != nil {
				missing++
				log.Printf("[js-plugin:%s] ctx.kernel.install %q 失败: %v", p.def.id, key, err)
				continue
			}
			installed++
			// 卸载时自动摘除路由（与 ctx.http.register 同生命周期纪律）
			disposeFn := dispose
			p.addCleanup(disposeFn)
		}
		return vm.ToValue(map[string]any{
			"installed": installed,
			"missing":   missing,
			"total":     len(items),
		})
	})

	return vm.ToValue(k)
}

// ─── ctx.market：市场源服务（市场插件化）─────────────────
//
// 磁盘插件 apply 时声明自己提供的市场（skill/mcp/plugin）：
//   - ctx.market.register({kind, source, name, desc}) → 注册/覆盖该市场；
//     source 标识搜索实现：github（skill）/ npm（mcp）/ npm-cordis（plugin）
//   - ctx.market.unregister(kind) → 注销市场
//   - ctx.market.list()           → 已注册市场列表
//
// 插件卸载/停用时自动注销其注册的市场（与 ctx.kernel.install 同生命周期纪律）。
func (p *jsPluginAdapter) buildMarketService() goja.Value {
	vm := p.vm
	m := vm.NewObject()

	m.Set("register", func(call goja.FunctionCall) goja.Value {
		a := call.Argument(0)
		if goja.IsUndefined(a) || goja.IsNull(a) {
			panic(vm.NewTypeError("ctx.market.register: 需要 {kind, source, name, desc}"))
		}
		obj, ok := a.Export().(map[string]any)
		if !ok {
			panic(vm.NewTypeError("ctx.market.register: 参数必须是对象 {kind, source, name, desc}"))
		}
		kind, _ := obj["kind"].(string)
		if kind == "" {
			panic(vm.NewTypeError("ctx.market.register: 缺 kind（skill/mcp/plugin）"))
		}
		source, _ := obj["source"].(string)
		name, _ := obj["name"].(string)
		desc, _ := obj["desc"].(string)
		RegisterMarketSource(MarketSourceMeta{Kind: kind, Name: name, Source: source, Desc: desc})
		// 插件卸载 → 自动注销市场（同 kernel.install 生命周期纪律）
		p.addCleanup(func() { UnregisterMarketSource(kind) })
		return vm.ToValue(true)
	})
	m.Set("unregister", func(call goja.FunctionCall) goja.Value {
		UnregisterMarketSource(call.Argument(0).String())
		return vm.ToValue(true)
	})
	m.Set("list", func(call goja.FunctionCall) goja.Value {
		srcs := MarketSources()
		out := make([]map[string]any, 0, len(srcs))
		for _, s := range srcs {
			out = append(out, map[string]any{"kind": s.Kind, "name": s.Name, "source": s.Source, "desc": s.Desc})
		}
		return vm.ToValue(out)
	})

	return vm.ToValue(m)
}

// ─── 沙箱创建与求值 ────────────────────────────────────────

// newJSSandbox 创建插件沙箱：注入 console/btoa/atob/TextEncoder/TextDecoder
// 与 __resolve 回调。返回 runtime 与 resolve 回调（goja 值 → 插件对象导出）。
// def 非空时 console 输出同步捕获进 def.console（宿主 stdout 流对模型不可见，
// 需经 cordis_run 返回展示；见 def.addConsole / ConsoleText）。
func newJSSandbox(def *jsPluginDef) (*goja.Runtime, *goja.Object) {
	vm := goja.New()

	// console（对齐 harness：带包名 tag，写透到宿主 stdout + 捕获进 def.console）
	tag := fmt.Sprintf("[js-plugin:%s]", def.id)
	consoleObj := vm.NewObject()
	logFn := func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, a := range call.Arguments {
			parts[i] = jsConsoleArg(a)
		}
		line := strings.Join(parts, " ")
		log.Printf("%s %s", tag, line)
		if def != nil {
			def.addConsole(line)
		}
		return goja.Undefined()
	}
	for _, m := range []string{"log", "info", "warn", "debug", "error"} {
		consoleObj.Set(m, logFn)
	}
	vm.Set("console", consoleObj)

	// process shim（npm cordis 插件常用 node 全局；只读常用字段，危险操作禁用）
	processObj := vm.NewObject()
	envObj := vm.NewObject()
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 {
			envObj.Set(kv[:i], kv[i+1:])
		}
	}
	processObj.Set("env", envObj)
	processObj.Set("platform", runtime.GOOS)
	processObj.Set("arch", runtime.GOARCH)
	processObj.Set("version", "v22.0.0") // 兼容占位（沙箱非 node 运行时）
	processObj.Set("cwd", func(call goja.FunctionCall) goja.Value {
		wd, _ := os.Getwd()
		return vm.ToValue(wd)
	})
	processObj.Set("exit", func(call goja.FunctionCall) goja.Value {
		panic(vm.NewTypeError("process.exit is not available in the dynamic package sandbox — let the plugin return normally"))
	})
	vm.Set("process", processObj)

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

	// Node API trap（对齐 harness NODE_API_REDIRECTS）：require/setTimeout/fetch 等
	// 在沙箱中不可用，调用即抛教学错误，引导走 ctx 服务（与 harness 沙箱纪律一致）。
	nodeAPI := map[string]string{
		"require":       "Node modules are unavailable. Use the cordis services on ctx instead — e.g. ctx.fs for files, ctx.web for HTTP, ctx.bash for processes; query cordis_inspect_query for available services.",
		"setTimeout":    "Node timers are unavailable. Use ctx.timeout(callback, delay) instead (cordis timer service).",
		"setInterval":   "Node timers are unavailable. Use ctx.interval(callback, delay) instead (cordis timer service).",
		"setImmediate":  "Node timers are unavailable. Use ctx.timeout(callback, 0) instead (cordis timer service).",
		"clearTimeout":  "Node timers are unavailable. ctx.timeout / ctx.interval return dispose functions that clear the timer.",
		"clearInterval": "Node timers are unavailable. ctx.timeout / ctx.interval return dispose functions that clear the timer.",
		"fetch":         "Network access goes through the cordis web service — use ctx.web instead (query cordis_inspect_query for its methods).",
	}
	for name, redirect := range nodeAPI {
		n, r := name, redirect
		vm.Set(n, func(call goja.FunctionCall) goja.Value {
			panic(vm.NewTypeError("%s is not available in the dynamic package sandbox — %s", n, r))
		})
	}

	// 内置 cordis 运行时：执行 bundle 后全局挂 CordisApi（真 cordis Context），
	// 插件代码可 new CordisApi.api.Context() 建 cordis app 跑生态插件协作。
	// 失败不致命：log 警告，沙箱其余能力不受影响。
	if _, err := vm.RunString(cordisBundleSource()); err != nil {
		log.Printf("[plugin_js] cordis bundle 装载失败（CordisApi 不可用）: %v", err)
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
	if err := runJSWithTimeout(vm, jsEvalTimeout, func() error {
		_, err := vm.RunString(src)
		return err
	}); err != nil {
		if isJSTimeout(err) {
			return nil, fmt.Errorf("插件 %s 求值超时（%.0fs，疑似死循环，已强制中断）", id, jsEvalTimeout.Seconds())
		}
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
//
// ★ 插件形态（对齐 harness isPlugin，兼容 cordis 生态）：
//   - 对象形态：return { name, apply(ctx, config), inject?: [...] }（apply 必须）
//   - 函数形态：return (ctx, config) => void —— cordis 生态惯例
//     （module.exports = function(ctx, config) {}）；函数名作插件名，匿名用 def.id；
//     也支持 fn.inject = [...] 静态属性声明硬依赖。
func (h *PluginHost) LoadJSDynamic(def *jsPluginDef) error {
	if def == nil || strings.TrimSpace(def.code) == "" {
		return fmt.Errorf("插件 %s: 代码为空", def.id)
	}
	// 重置运行诊断起点（保留历史 diag 但标记新装载阶段）
	def.addDiag(fmt.Sprintf("[%s] run 开始（pluginId=%s pkg=%s）", time.Now().Format("15:04:05"), def.pluginId, def.packageId))
	def.console = nil // 每次 run 独立捕获 console 输出

	vm, _ := newJSSandbox(def)
	obj, err := evalJSPlugin(vm, def.code, def.id)
	if err != nil {
		def.addDiag("求值失败: " + err.Error())
		def.lastError = err.Error()
		h.mu.Lock()
		def.setStatus(PluginRejected, nil)
		h.mu.Unlock()
		return err
	}
	def.addDiag("求值通过")

	var applyFn goja.Callable
	name := ""
	if fn, ok := goja.AssertFunction(obj); ok {
		// 函数形态插件
		applyFn = fn
		def.isFunc = true
		if nv := obj.Get("name"); nv != nil && nv.String() != "" {
			name = nv.String()
		} else if def.name != "" {
			name = def.name
		} else {
			name = def.id // 匿名函数插件名 = dyn id
		}
	} else {
		// 对象形态插件
		nameVal := obj.Get("name")
		applyVal := obj.Get("apply")
		if nameVal == nil || nameVal.String() == "" {
			return fmt.Errorf("插件 %s: 缺少 name 字段（对象形态必须 return { name, apply(ctx, config) }；或直接 return 函数 (ctx, config) => void）", def.id)
		}
		applyFn, ok = goja.AssertFunction(applyVal)
		if !ok {
			return fmt.Errorf("插件 %s: apply 必须是函数", def.id)
		}
		name = nameVal.String()
	}

	// inject 声明解析（对象/函数形态统一读 obj.inject 属性）
	if inj := obj.Get("inject"); inj != nil && !goja.IsUndefined(inj) && !goja.IsNull(inj) {
		if arr, ok := inj.Export().([]any); ok {
			def.inject = def.inject[:0]
			for _, it := range arr {
				if s, ok := it.(string); ok && s != "" {
					def.inject = append(def.inject, s)
				}
			}
		}
	}
	// ★ 硬依赖校验（D3 等待语义，对齐 harness lifecycle）：
	//   inject 声明的服务缺失 → 插件进入 waiting（不装载、不 apply），
	//   服务出现后经 retryWaiting 自动激活；可选服务请用 ctx.get(name) 判 undefined。
	if missing := h.missingServices(def); len(missing) > 0 {
		def.addDiag(fmt.Sprintf("inject 等待: %v（可用服务: %v）", missing, h.availableServices()))
		def.name = name
		h.waitForServices(def, missing)
		return nil // 等待不是错误：返回成功，由调用方检查 def.status 提示等待
	}

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

	// ★ D8 restart 语义（对齐 harness run mode=run）：同名插件已注册/运行 →
	// 先卸载旧实例并清出注册表，再装载新版本（不再报同名冲突）。
	h.mu.Lock()
	oldName := ""
	if _, exists := h.plugins[name]; exists {
		oldName = name
	}
	h.mu.Unlock()
	if oldName != "" {
		if err := h.Unload(oldName); err != nil {
			def.lastError = err.Error()
			return err
		}
		h.mu.Lock()
		// 旧实例对应定义状态复位（版本链中其他版本不再 running）
		if oldAdapter, ok := h.plugins[oldName].(*jsPluginAdapter); ok {
			oldAdapter.def.setStatus(PluginStopped, nil)
		}
		delete(h.plugins, oldName)
		delete(h.sources, oldName)
		for i, n := range h.order {
			if n == oldName {
				h.order = append(h.order[:i], h.order[i+1:]...)
				break
			}
		}
		h.mu.Unlock()
		def.addDiag(fmt.Sprintf("restart：旧实例 %s 已卸载，装载新版本", oldName))
	}

	// 登记 + 装载
	if err := h.Register(adapter, PluginSourceJS); err != nil {
		def.lastError = err.Error()
		h.mu.Lock()
		def.setStatus(PluginRejected, nil)
		h.mu.Unlock()
		return err
	}
	if err := h.Load(name); err != nil {
		def.addDiag("apply 失败: " + err.Error())
		def.lastError = err.Error()
		h.mu.Lock()
		def.setStatus(PluginFailed, nil)
		h.mu.Unlock()
		return err
	}
	def.addDiag("apply 通过，运行中")
	h.mu.Lock()
	def.setStatus(PluginRunning, nil)
	h.mu.Unlock()
	// ★ 2026-08-17：装载 ≠ agent 可用——非工作区工具集插件的工具默认对 agent 隐藏
	//   （cordis/前端仍可见可管理；toolset_edit add_plugin 加入工具集后可见）。
	//   工具集插件（applyToolsetPlugin 装载）的工具在白名单内，不受影响。
	h.applyPluginToolVisibility(name)
	return nil
}

// ─── inject 硬依赖校验 ────────────────────────────────────

// missingServices 返回 inject 声明但宿主未提供的服务（空=可装载）。
// 对齐 harness 语义：inject 是硬依赖但会等待（waiting），不是直接拒绝；
// 可选服务用 ctx.get(name) 并判 undefined。
func (h *PluginHost) missingServices(def *jsPluginDef) []string {
	if len(def.inject) == 0 {
		return nil
	}
	var missing []string
	for _, s := range def.inject {
		if !h.hasService(s) {
			missing = append(missing, s)
		}
	}
	return missing
}

// checkInjects 校验插件声明的 inject 硬依赖（保留：报错式引导，供诊断展示）。
// 返回等待语义的完整报错（含可用服务清单）。
func (h *PluginHost) checkInjects(def *jsPluginDef) error {
	missing := h.missingServices(def)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf(
		"插件 %s 声明了 inject: %v，但宿主未提供服务: %v。可用服务: %v。可选服务请改用 ctx.get(%q) 并判 undefined",
		def.id, def.inject, missing, h.availableServices(), missing[0])
}

// hasService 判断宿主是否提供某服务（静态服务键 + 动态 ctx.provide 服务）。
func (h *PluginHost) hasService(name string) bool {
	switch name {
	case "fs", "web", "bash", "sse", "ws", "logger", "timer", "tools", "events", "store", "app", "workspaceRoot", "kernel", "market":
		return true
	}
	return h.ctx.Get(name) != nil
}

// availableServices 宿主可用服务清单（供报错引导/文档展示）。
func (h *PluginHost) availableServices() []string {
	names := []string{"fs", "web", "bash", "sse", "ws", "logger", "timer", "tools", "events", "store", "app", "workspaceRoot", "kernel", "market"}
	h.ctx.servicesMu.RLock()
	for n := range h.ctx.services {
		names = append(names, n)
	}
	h.ctx.servicesMu.RUnlock()
	sort.Strings(names)
	return names
}

// ─── harness 全局注入 ─────────────────────────────────────

// injectHarness 向沙箱注入 harness 对象（defineTool/registerTool/handle）。
// 与 buildContextObject 共用 jsToolToGo 桥；pc 为当前插件的上下文（注册归属）。
func injectHarness(vm *goja.Runtime, adapter *jsPluginAdapter, pc *PluginContext) *goja.Object {
	harnessObj := vm.NewObject()
	harnessObj.Set("defineTool", func(call goja.FunctionCall) goja.Value {
		// 预检工具定义可转换（不注册）；返回原对象
		if _, err := jsToolToGo(vm, call.Argument(0), adapter.withLock); err != nil {
			panic(vm.NewGoError(err))
		}
		return call.Argument(0)
	})
	harnessObj.Set("registerTool", func(call goja.FunctionCall) goja.Value {
		tool, err := jsToolToGo(vm, call.Argument(0), adapter.withLock)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		if pc != nil {
			err = pc.RegisterTool(tool)
		} else {
			err = adapter.host.Context().RegisterTool(tool)
		}
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	harnessObj.Set("handle", func(call goja.FunctionCall) goja.Value {
		method := call.Argument(0).String()
		if method == "" {
			panic(vm.NewTypeError("harness.handle: 方法名不能为空"))
		}
		fnVal := call.Argument(1)
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			panic(vm.NewTypeError("harness.handle: 第二参数必须是函数"))
		}
		registerJSHandler(vm, adapter, method, fn)
		return goja.Undefined()
	})
	vm.Set("harness", harnessObj)
	return harnessObj
}

// registerJSHandler 把 JS 函数注册为 host 侧处理器（harness.handle /
// ctx.registerClientMethod 共用存储）。浏览器 client 半可经 InvokeClientMethod
// 远程调用（invoke RPC）；Agent 侧经 harness 调用。
func registerJSHandler(vm *goja.Runtime, p *jsPluginAdapter, method string, fn goja.Callable) {
	p.mu.Lock()
	p.handlers[method] = func(args any) (any, error) {
		var res goja.Value
		var hErr error
		// Invoke 可能来自任意 goroutine → 执行锁保护
		p.withLock(func() {
			if e := runJSWithTimeout(p.vm, jsHandlerTimeout, func() error {
				r, err := fn(goja.Undefined(), vm.ToValue(args))
				res = r
				return err
			}); e != nil {
				if isJSTimeout(e) {
					hErr = fmt.Errorf("handler %s 执行超时（疑似死循环，已强制中断）", method)
				} else {
					hErr = fmt.Errorf("handler %s 异常: %v", method, jsErrorText(e))
				}
			}
		})
		if hErr != nil {
			return nil, hErr
		}
		return res.Export(), nil
	}
	p.mu.Unlock()
}

// ─── JS 工具定义 → Go Tool 桥 ─────────────────────────────

// jsSchemaValidTypes JSON Schema 合法 type（含 harness 扩展 'json'）。
var jsSchemaValidTypes = map[string]bool{
	"string": true, "number": true, "integer": true, "boolean": true,
	"object": true, "array": true, "null": true, "json": true,
}

// normalizeToolSchema 规范化插件工具参数 schema（不修改入参，返回新 map）：
//   - type: 'json'（harness 扩展，非标准 JSON Schema type）→ 移除 type（=任意值，
//     避免 LLM function-calling 端误读）
//   - $ref: '#/$defs/x' / '#/definitions/x' → 从同 schema 定义表内联解析（深度受限）；
//     无法解析的 $ref → 移除（避免 LLM 端收到悬空引用）
func normalizeToolSchema(params map[string]any) map[string]any {
	if params == nil {
		return nil
	}
	defs := map[string]any{}
	if d, ok := params["$defs"].(map[string]any); ok {
		for k, v := range d {
			defs[k] = v
		}
	}
	if d, ok := params["definitions"].(map[string]any); ok {
		for k, v := range d {
			defs[k] = v
		}
	}
	seen := map[string]bool{}
	var walk func(m map[string]any, depth int)
	walk = func(m map[string]any, depth int) {
		if depth > 8 {
			return
		}
		if ref, ok := m["$ref"].(string); ok && ref != "" {
			if strings.HasPrefix(ref, "#/$defs/") || strings.HasPrefix(ref, "#/definitions/") {
				key := strings.TrimPrefix(strings.TrimPrefix(ref, "#/$defs/"), "#/definitions/")
				key = strings.ReplaceAll(key, "~1", "/")
				key = strings.ReplaceAll(key, "~0", "~")
				if sub, ok := defs[key]; ok {
					if sm, ok := sub.(map[string]any); ok && !seen[key] {
						seen[key] = true
						walk(sm, depth+1)
						delete(m, "$ref")
						for k, v := range sm {
							m[k] = v
						}
						return
					}
				}
			}
			delete(m, "$ref") // 无法解析的 $ref：移除
		}
		if t, ok := m["type"].(string); ok && t == "json" {
			delete(m, "type") // 'json' → 任意值
		}
		if props, ok := m["properties"].(map[string]any); ok {
			for _, p := range props {
				if pm, ok := p.(map[string]any); ok {
					walk(pm, depth+1)
				}
			}
		}
		if items, ok := m["items"].(map[string]any); ok {
			walk(items, depth+1)
		}
		for _, key := range []string{"allOf", "anyOf", "oneOf"} {
			if arr, ok := m[key].([]any); ok {
				for _, b := range arr {
					if bm, ok := b.(map[string]any); ok {
						walk(bm, depth+1)
					}
				}
			}
		}
	}
	walk(params, 0)
	return params
}

// validateToolSchema 定义期校验插件工具参数 schema（轻量：type 合法性 + 结构要点 +
// realm 安全：整棵可 JSON 序列化、拒绝外部 $ref 与原型污染键）。
// 不合法返回 error——cordis_define/registerTool 提前暴露，避免运行期才崩。
// （对齐 harness guard.ts：schema type 白名单 + cloneJson 无损克隆；goja 单 realm
//   天然豁免跨 realm instanceof，此处补序列化与引用边界。）
func validateToolSchema(params map[string]any) error {
	if params == nil {
		return nil
	}
	// ① 整棵可 JSON 序列化（防函数/循环引用/不可导出值混入 → LLM 端崩溃）
	if _, err := json.Marshal(params); err != nil {
		return fmt.Errorf("schema 含不可序列化值（函数/循环引用？）: %v", err)
	}
	// ② 原型污染键检测（__proto__/constructor.prototype 等）
	var polluteErr error
	var polluteWalk func(m map[string]any, depth int)
	polluteWalk = func(m map[string]any, depth int) {
		if polluteErr != nil || depth > 10 {
			return
		}
		for k, v := range m {
			if k == "__proto__" || k == "prototype" || strings.Contains(k, "constructor") {
				polluteErr = fmt.Errorf("schema 含可疑键 %q（原型污染防护）", k)
				return
			}
			if sm, ok := v.(map[string]any); ok {
				polluteWalk(sm, depth+1)
			}
			if arr, ok := v.([]any); ok {
				for _, e := range arr {
					if sm, ok := e.(map[string]any); ok {
						polluteWalk(sm, depth+1)
					}
				}
			}
		}
	}
	polluteWalk(params, 0)
	if polluteErr != nil {
		return polluteErr
	}
	// ③ 外部 $ref 拒绝（只允许 #/ 内部引用，防 file:// http:// 等逃逸）
	var refErr error
	var refWalk func(m map[string]any, depth int)
	refWalk = func(m map[string]any, depth int) {
		if refErr != nil || depth > 10 {
			return
		}
		if ref, ok := m["$ref"].(string); ok && ref != "" && !strings.HasPrefix(ref, "#/") {
			refErr = fmt.Errorf("$ref 只允许 #/ 内部引用，收到 %q", ref)
			return
		}
		for _, v := range m {
			if sm, ok := v.(map[string]any); ok {
				refWalk(sm, depth+1)
			}
			if arr, ok := v.([]any); ok {
				for _, e := range arr {
					if sm, ok := e.(map[string]any); ok {
						refWalk(sm, depth+1)
					}
				}
			}
		}
	}
	refWalk(params, 0)
	if refErr != nil {
		return refErr
	}
	if t, ok := params["type"]; ok {
		switch tv := t.(type) {
		case string:
			if !jsSchemaValidTypes[tv] {
				return fmt.Errorf("type %q 非法（合法: string/number/integer/boolean/object/array/null/json）", tv)
			}
		case []any:
			for _, e := range tv {
				s, ok := e.(string)
				if !ok || !jsSchemaValidTypes[s] {
					return fmt.Errorf("type 数组含非法值 %v", e)
				}
			}
		default:
			return fmt.Errorf("type 必须是字符串或数组, got %T", t)
		}
	}
	if props, ok := params["properties"]; ok {
		if _, ok := props.(map[string]any); !ok {
			return fmt.Errorf("properties 必须是对象")
		}
	}
	if _, ok := params["items"]; ok {
		if t, _ := params["type"].(string); t != "array" {
			return fmt.Errorf("items 仅在 type=array 时允许")
		}
	}
	return nil
}

// jsToolToGo 把 JS 工具定义对象转成 *Tool。
// 支持 execute: (args) => result | Promise<result>（result 可为 {text} 或任意 JSON 值）。
func jsToolToGo(vm *goja.Runtime, v goja.Value, lockFn func(func())) (*Tool, error) {
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
	// P1: schema 规范化 + 定义期校验（'json' type → 任意值；$ref 内联；非法结构提前报错）
	if err := validateToolSchema(params); err != nil {
		return nil, fmt.Errorf("工具 %s 参数 schema 非法: %v", name.String(), err)
	}
	params = normalizeToolSchema(params)
	execVal := obj.Get("execute")
	if execVal == nil || goja.IsUndefined(execVal) || goja.IsNull(execVal) {
		return nil, fmt.Errorf("工具 %s 缺少 execute 函数", name.String())
	}
	execFn, ok := goja.AssertFunction(execVal)
	if !ok {
		return nil, fmt.Errorf("工具 %s execute 必须是函数", name.String())
	}

	handler := func(ctx context.Context, args map[string]any) (string, error) {
		var out string
		var hErr error
		run := func() {
			// ★ 会话标识注入：Loop ctx 链携带 convID 时，复制 args 并注入
			//   `_convID` 内部键（不污染原 args——AfterTool 观察/统计用原值）。
			//   插件工具 execute 内可读 args._convID 经 ctx.hostTool 路由回宿主
			//   （ask_user/task_create 会话桥；其余工具可忽略）。无会话则原样。
			injected := args
			if convID := SessionConvID(ctx); convID != "" {
				injected = make(map[string]any, len(args)+1)
				for k, v := range args {
					injected[k] = v
				}
				injected["_convID"] = convID
			}
			var res goja.Value
			execErr := runJSWithTimeout(vm, jsToolTimeout, func() error {
				r, err := execFn(goja.Undefined(), vm.ToValue(injected))
				res = r
				return err
			})
			if execErr != nil {
				if isJSTimeout(execErr) {
					hErr = fmt.Errorf("JS 工具 %s 执行超时（%.0fs，疑似死循环，已强制中断）", name.String(), jsToolTimeout.Seconds())
				} else {
					hErr = fmt.Errorf("JS 工具 %s 执行失败: %v", name.String(), jsErrorText(execErr))
				}
				return
			}
			out, hErr = jsResultToText(vm, res)
		}
		if lockFn != nil {
			lockFn(run)
		} else {
			run()
		}
		return out, hErr
	}

	return &Tool{
		Name:        name.String(),
		Description: desc,
		Parameters:  params,
		Handler:     handler,
		UsageGuide:  strField(obj, "usageGuide"),
		Category:    strField(obj, "category"),
		ReadOnly:    boolField(obj, "readOnly"),
		RequiresApproval: boolField(obj, "requiresApproval"),
		SystemTool:  boolField(obj, "systemTool"),
	}, nil
}

// strField 读对象字符串字段（缺省空串）。
func strField(obj *goja.Object, key string) string {
	if v := obj.Get(key); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		return v.String()
	}
	return ""
}

// boolField 读对象布尔字段（缺省 false）。
func boolField(obj *goja.Object, key string) bool {
	if v := obj.Get(key); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		return v.ToBoolean()
	}
	return false
}

// awaitJSValue 同步等待 JS Promise 并返回 resolve 值（goja 同步 VM：
// 微任务队列由 RunString("") 驱动 drain）。async 函数返回的 Promise 在
// 微任务队列中已排队，drain 后立即取到结果——用于 ctx.http.register 的
// async handler 与工具 execute 的 async 返回（对齐 harness run 语义）。
// 非 Promise 值原样返回；reject → error。
func awaitJSValue(vm *goja.Runtime, v goja.Value) (goja.Value, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return v, nil
	}
	o, ok := v.(*goja.Object)
	if !ok {
		return v, nil
	}
	then := o.Get("then")
	if then == nil || goja.IsUndefined(then) || goja.IsNull(then) {
		return v, nil
	}
	thenFn, ok := goja.AssertFunction(then)
	if !ok {
		return v, nil
	}
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
		return nil, err
	}
	_, _ = vm.RunString("") // drain 微任务队列（Promise.then 回调在 job 中执行）
	if gotErr != nil {
		return nil, gotErr
	}
	return awaitJSValue(vm, got) // 递归：then 链上的嵌套 promise
}

// ── ctx.webServer：Node 风格 HTTP handler 桥（对齐 harness dsh-host-webserver）──

// buildNodeHTTPHandler 构造 Node 风格 HTTP handler：JS 插件以
// handler(req, res) 形态实现处理逻辑（接口定义 + 逻辑都在插件中），
// 服务能力经 ctx.fs/ctx.web/ctx.tools 等 Go 服务访问。
//   req:  { method, url, path, query, headers, body, httpVersion, json(),
//          on('data'|'end', cb)（body 已整体读入：data 触发一次、end 立即）}
//   res:  { statusCode（属性赋值）, statusMessage, writeHead(code, headers?),
//          setHeader(k,v), getHeader(k), hasHeader(k), removeHeader(k),
//          write(chunk), end(chunk?), on('finish'|'close', cb) }
// 兼容旧 ctx.http 返回对象形态：{ status, body, headers }（未调 end 时）。
func (p *jsPluginAdapter) buildNodeHTTPHandler(fn goja.Callable) http.HandlerFunc {
	vm := p.vm
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		headers := map[string]string{}
		for k, vs := range r.Header {
			if len(vs) > 0 {
				headers[k] = vs[0]
			}
		}
		// ── req 桥 ──
		reqObj := vm.NewObject()
		reqObj.Set("method", r.Method)
		reqObj.Set("url", r.URL.RequestURI())
		reqObj.Set("path", r.URL.Path)
		reqObj.Set("query", r.URL.RawQuery)
		reqObj.Set("headers", headers)
		reqObj.Set("body", string(body))
		reqObj.Set("httpVersion", r.Proto)
		reqObj.Set("json", func(call goja.FunctionCall) goja.Value {
			var out any
			if err := json.Unmarshal(body, &out); err != nil {
				panic(vm.NewGoError(fmt.Errorf("req.json(): 请求体不是合法 JSON: %w", err)))
			}
			return vm.ToValue(out)
		})
		var dataCbs, endCbs []goja.Callable
		reqObj.Set("on", func(call goja.FunctionCall) goja.Value {
			ev := call.Argument(0).String()
			cb, ok := goja.AssertFunction(call.Argument(1))
			if ok {
				switch ev {
				case "data":
					dataCbs = append(dataCbs, cb)
				case "end":
					endCbs = append(endCbs, cb)
				}
			}
			return reqObj // 链式
		})
		// ── res 桥 ──
		type resStateT struct {
			status  int
			headers map[string]string
			chunks  []string
			ended   bool
		}
		rs := &resStateT{status: http.StatusOK, headers: map[string]string{}}
		var finishCbs []goja.Callable
		resObj := vm.NewObject()
		resObj.Set("writeHead", func(call goja.FunctionCall) goja.Value {
			rs.status = int(call.Argument(0).ToInteger())
			if h := call.Argument(1); h != nil && !goja.IsUndefined(h) && !goja.IsNull(h) {
				if m, ok := h.Export().(map[string]any); ok {
					for k, v := range m {
						rs.headers[k] = fmt.Sprint(v)
					}
				}
			}
			return resObj
		})
		resObj.Set("setHeader", func(call goja.FunctionCall) goja.Value {
			rs.headers[call.Argument(0).String()] = call.Argument(1).String()
			return resObj
		})
		resObj.Set("getHeader", func(call goja.FunctionCall) goja.Value {
			if v, ok := rs.headers[call.Argument(0).String()]; ok {
				return vm.ToValue(v)
			}
			return goja.Undefined()
		})
		resObj.Set("hasHeader", func(call goja.FunctionCall) goja.Value {
			_, ok := rs.headers[call.Argument(0).String()]
			return vm.ToValue(ok)
		})
		resObj.Set("removeHeader", func(call goja.FunctionCall) goja.Value {
			delete(rs.headers, call.Argument(0).String())
			return goja.Undefined()
		})
		resObj.Set("write", func(call goja.FunctionCall) goja.Value {
			if v := call.Argument(0); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
				rs.chunks = append(rs.chunks, v.String())
			}
			return vm.ToValue(true) // 可继续写
		})
		resObj.Set("end", func(call goja.FunctionCall) goja.Value {
			if v := call.Argument(0); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
				rs.chunks = append(rs.chunks, v.String())
			}
			rs.ended = true
			return vm.ToValue(resObj)
		})
		resObj.Set("on", func(call goja.FunctionCall) goja.Value {
			if ev := call.Argument(0).String(); ev == "finish" || ev == "close" {
				if cb, ok := goja.AssertFunction(call.Argument(1)); ok {
					finishCbs = append(finishCbs, cb)
				}
			}
			return resObj
		})
		// ── 调用 handler（async 支持：返回值 Promise 同步 drain）──
		var ret goja.Value
		var callErr error
		p.withLock(func() {
			v, err := fn(goja.Undefined(), reqObj, resObj)
			if err == nil {
				v, err = awaitJSValue(vm, v)
			}
			ret, callErr = v, err
		})
		if callErr != nil {
			http.Error(w, "ctx.webServer handler 执行失败: "+jsErrorText(callErr), http.StatusInternalServerError)
			return
		}
		// ── flush 响应 ──
		status := rs.status
		if sv := resObj.Get("statusCode"); sv != nil && !goja.IsUndefined(sv) && !goja.IsNull(sv) {
			status = int(sv.ToInteger())
		}
		respHeaders := rs.headers
		respBody := strings.Join(rs.chunks, "")
		if !rs.ended && ret != nil && !goja.IsUndefined(ret) && !goja.IsNull(ret) {
			// 兼容旧 ctx.http 返回对象形态：{ status, body, headers } / 字符串
			if s, ok := ret.Export().(string); ok {
				respBody = s
			} else if obj := ret.ToObject(vm); obj != nil {
				if v := obj.Get("status"); v != nil {
					status = int(v.ToInteger())
				}
				if v := obj.Get("body"); v != nil {
					respBody = v.String()
				}
				if v := obj.Get("headers"); v != nil {
					if m, ok := v.Export().(map[string]any); ok {
						for k, vv := range m {
							respHeaders[k] = fmt.Sprint(vv)
						}
					}
				}
			} else {
				respBody = ret.String()
			}
		}
		for k, v := range respHeaders {
			w.Header().Set(k, v)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
		// res 'finish'/'close' 回调（end 后触发）
		for _, cb := range finishCbs {
			_, _ = cb(goja.Undefined())
		}
	}
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
	// Promise：awaitJSValue 同步 drain（then 链），再递归转文本
	if o, ok := v.(*goja.Object); ok {
		if then := o.Get("then"); then != nil {
			if _, ok := goja.AssertFunction(then); ok {
				got, err := awaitJSValue(vm, v)
				if err != nil {
					return "", err
				}
				return jsResultToText(vm, got)
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
	return h.DefineJSCodeFull(code, language, purpose, dir, "")
}

// DefineJSCodeFull 登记动态插件定义（完整签名：host 半 + 可选 client 半 + 多文件 dir）。
// clientCode 是浏览器端执行的插件代码（可为空=纯 host 插件）：
// 形态 (ui) => void，ui 提供 on/emit/registerPanel/http 等浏览器侧服务
// （契约见 cmd/companion/web-ui/src/plugin-runtime.js）。
// 返回新分配的 dyn id（新插件）或最新版本 dyn id（existing 追加）。
func (h *PluginHost) DefineJSCodeFull(code, language, purpose, dir, clientCode string) (string, error) {
	return h.DefineJSCodeVersioned(code, language, purpose, dir, clientCode, "")
}

// DefineJSCodeVersioned 版本化登记：pluginId 为空 → 新建插件（分配稳定 pluginId）；
// pluginId 非空 → existing 模式：向该插件追加一个版本（对齐 harness define existing append）。
// 返回 def.id（dyn-n，精确版本 id）；cordis_run 传 pluginId 或该 id 均可装载。
func (h *PluginHost) DefineJSCodeVersioned(code, language, purpose, dir, clientCode, pluginId string) (string, error) {
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
	// client 半语法预检（浏览器端用 new Function 求值；此处仅验证语法）
	if strings.TrimSpace(clientCode) != "" {
		if _, err := goja.Compile("cordis-client.js", "("+clientCode+")", false); err != nil {
			return "", fmt.Errorf("插件 client 半语法错误: %v", jsErrorText(err))
		}
	}
	seq := dynSeq.Add(1)
	id := fmt.Sprintf("dyn-%d", seq)
	pkgID := fmt.Sprintf("pkg-%d", seq)

	h.mu.Lock()
	var verNo int
	stable := id // 默认：新插件，pluginId = 自身 dyn id
	if strings.TrimSpace(pluginId) != "" {
		chain := h.pluginVersions[pluginId]
		if len(chain) == 0 {
			h.mu.Unlock()
			return "", fmt.Errorf("插件 %s 不存在，无法追加版本（首次 define 请不传 pluginId）", pluginId)
		}
		stable = pluginId // existing：复用稳定身份
		verNo = len(chain)
	}
	def := &jsPluginDef{
		id:         id,
		pluginId:   stable,
		packageId:  pkgID,
		name:       extractJSPluginName(js), // ★ 静态提取（审批键 name 需 define 时可用；装载时按实际对象再解析）
		purpose:    purpose,
		code:       js, // 存转译后的 JS（运行时 goja 直接执行）
		clientCode: clientCode,
		lang:       lang,
		dir:        dir, // 插件目录（ctx.binary 据此定位 bin/<name>.exe 与 assets/）
		version:    fmt.Sprintf("v%d", verNo+1),
		status:     PluginStopped,
		createdAt:  time.Now(),
	}
	h.defs[id] = def
	h.pluginVersions[stable] = append(h.pluginVersions[stable], def)
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
// 删除整个 pluginId 的全部版本（版本化模型：undefine 按稳定身份清链）。
func (h *PluginHost) RemoveJSDef(id string) error {
	h.mu.RLock()
	def, ok := h.defs[id]
	h.mu.RUnlock()
	if !ok {
		// 支持传 pluginId：解析到最新版本
		resolved, err := h.resolveJSDef(id)
		if err != nil {
			return err
		}
		def = resolved
	}
	_ = h.Unload(def.name)
	h.mu.Lock()
	// 删除该 pluginId 版本链上的全部 defs
	for _, d := range h.pluginVersions[def.pluginId] {
		delete(h.defs, d.id)
	}
	delete(h.pluginVersions, def.pluginId)
	delete(h.waiting, def.id)
	def.setStatus(PluginCancelled, nil)
	h.mu.Unlock()
	// ★ 同步删除磁盘插件包（防重启 LoadGlobalPlugins 扫描目录重新装配「复活」）；
	//   纯内存定义（无磁盘包）时静默跳过。
	dir := def.dir
	if dir == "" {
		dir = filepath.Join(globalPluginsDir(), def.name)
	}
	return removeGlobalPluginPackage(dir)
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

// ─── 静态插件装配（cordis.patch.json，P2）──────────────────

// LoadCordisPatch 从 JSON 文件装配静态插件（跨重启存续，对齐 harness
// cordis.yml 的叶子装配入口，简化为零依赖 JSON）：
//
//	{ "plugins": [ { "code": "...", "language": "js|ts", "purpose": "...", "config": {...} } ] }
//
// 文件不存在 → 正常返回（无装配）；条目失败不致命（log 警告，继续后续条目）。
func (h *PluginHost) LoadCordisPatch(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // 无装配文件 = 正常
		}
		return err
	}
	var doc struct {
		Plugins []struct {
			Code     string         `json:"code"`
			Language string         `json:"language"`
			Purpose  string         `json:"purpose"`
			Config   map[string]any `json:"config"`
		} `json:"plugins"`
	}
	needNodeBridge := false
	if err := json.Unmarshal(b, &doc); err != nil {
		return fmt.Errorf("cordis patch 解析失败: %v", err)
	}
	for i, p := range doc.Plugins {
		if strings.TrimSpace(p.Code) == "" {
			// Node 桥型插件（依赖 npm 生态）：启动时由 Node 桥装载
			if rt, _ := p.Config["runtime"].(string); rt == "node" {
				needNodeBridge = true
			}
			continue
		}
		id, err := h.DefineJSCodeDir(p.Code, p.Language, p.Purpose, "")
		if err != nil {
			log.Printf("[cordis-patch] 第 %d 个插件登记失败: %v", i+1, err)
			continue
		}
		def, _ := h.GetJSDef(id)
		if p.Config != nil {
			def.config = p.Config
		}
		if err := h.LoadJSDynamic(def); err != nil {
			log.Printf("[cordis-patch] 插件 %s 装载失败: %v", id, err)
			continue
		}
		log.Printf("[cordis-patch] 插件 %s (%s) 已装配", def.name, id)
	}
	// Node 桥型插件：启动 Node 桥（真实 node 环境执行 npm 依赖插件）
	if needNodeBridge {
		if _, err := ensureNodeBridge(h, nodeBridgeDir()); err != nil {
			log.Printf("[cordis-patch] Node 桥启动失败: %v", err)
		}
	}
	return nil
}
