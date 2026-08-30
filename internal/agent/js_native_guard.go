// js_native_guard.go —— JS 超时看门狗「原生调用感知」（native-aware watchdog）
//
// ★ 2026-09-01 修复「handler <method> 执行超时（疑似死循环，已强制中断）」误判：
//
//	问题：runJSWithTimeout 原为固定墙钟超时——timer 到点直接 vm.Interrupt。
//	但 goja 的 Interrupt 只置 flag，在「JS 阻塞于 Go 原生调用」（ctx.bash.exec
//	跑 10 分钟命令、ctx.tools.run 执行宿主工具、ctx.agents 子 agent、ctx.llm
//	请求等）期间不生效；原生调用在 Go 侧成功返回后，JS 恢复执行的第一条指令
//	立刻撞上 flag → 被误判为死循环（表现：任务实际成功却报「执行超时」）。
//
//	历史打补丁式修法：ctx.binary.exec 单点 vm.ClearInterrupt()（2026-08-17），
//	但 2026-08-22 快速执行插件改走 ctx.bash.exec 后老 bug 复现——说明单点修
//	不可持续（任何阻塞型原生调用都会踩）。
//
//	本文件的系统修法：
//	 ① 记录每个 VM 的「原生调用深度」（nativeDepth，进入/退出 Go 原生调用时增减）
//	 ② 看门狗改为轮询：仅当 VM「不处于原生调用中」时累计计时，原生等待期间
//	    计时归零——即超时语义从「墙钟总时长」变为「纯 JS 连续执行时长」，
//	    纯 JS 死循环仍被 Interrupt 打断，原生阻塞等待永不误判。
//	 ③ 插桩方式：ctx 服务对象（fs/bash/tools/agents/llm/…）在构建完成后统一
//	    递归包装（wrapNativeGuards），无需逐处改写，新增服务方法自动受保护。
package agent

import (
	"sync"
	"sync/atomic"

	"github.com/hoonfeng/paircode/goja"
)

// jsNativeDepths VM → 原生调用深度（>0 = 当前阻塞在 Go 原生调用中）。
var jsNativeDepths sync.Map // map[*goja.Runtime]*int64

// jsNativeCounter 取（或建）某 VM 的深度计数器。
func jsNativeCounter(vm *goja.Runtime) *int64 {
	if vm == nil {
		return nil
	}
	if v, ok := jsNativeDepths.Load(vm); ok {
		return v.(*int64)
	}
	n := new(int64)
	actual, _ := jsNativeDepths.LoadOrStore(vm, n)
	return actual.(*int64)
}

// jsEnterNative 标记 VM 进入 Go 原生调用（返回 release 退出标记）。
// 嵌套安全（深度计数）：原生调用内再调原生（如 tools.run → 另一插件工具）不误判。
func jsEnterNative(vm *goja.Runtime) func() {
	c := jsNativeCounter(vm)
	if c == nil {
		return func() {}
	}
	atomic.AddInt64(c, 1)
	return func() { atomic.AddInt64(c, -1) }
}

// jsNativeBusy VM 当前是否阻塞在 Go 原生调用中（看门狗据此暂停计时）。
func jsNativeBusy(vm *goja.Runtime) bool {
	if vm == nil {
		return false
	}
	v, ok := jsNativeDepths.Load(vm)
	if !ok {
		return false
	}
	return atomic.LoadInt64(v.(*int64)) > 0
}

// jsForgetNative 丢弃某 VM 的计数（VM 废弃时清理，防 map 累积）。
func jsForgetNative(vm *goja.Runtime) {
	if vm != nil {
		jsNativeDepths.Delete(vm)
	}
}

// jsNativeGuardServices 需要「原生调用感知」包装的 ctx 服务名单——凡可能阻塞
// （子进程/网络/子 agent/文件 IO/宿主工具执行）的服务全部在列；纯内存注册面
// （events/store/systemPrompt/prompts/app/toolset 注册等）不包，以保留死循环检测灵敏度。
var jsNativeGuardServices = []string{
	"fs", "web", "bash", "tools", "hostTool", "binary", "http", "webServer",
	"process", "npm", "plugins", "market", "mcp", "skill", "agents", "llm",
	"kernel", "commands", "sse", "ws",
}

// wrapNativeGuards 递归包装对象上的函数属性：调用期间标记 VM 处于原生调用
// （看门狗暂停计时）。depth 限制递归层数（服务对象一般 1~2 层）。
func wrapNativeGuards(vm *goja.Runtime, obj *goja.Object, depth int) {
	if vm == nil || obj == nil || depth < 0 {
		return
	}
	for _, key := range obj.Keys() {
		val := obj.Get(key)
		if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
			continue
		}
		if fn, ok := goja.AssertFunction(val); ok {
			obj.Set(key, wrapNativeCallable(vm, fn))
			continue
		}
		if o, ok := val.(*goja.Object); ok && depth > 0 {
			wrapNativeGuards(vm, o, depth-1)
		}
	}
}

// wrapNativeCallable 把 goja 可调用值包成「原生调用感知」函数：
// 调用期间 nativeDepth++（看门狗不计时），返回/抛错时恢复。
// 错误语义保持不变：原实现 panic(vm.NewGoError(..)) 产生的 JS 异常经 panic 原样传播
// （含 *goja.InterruptedError——中断仍可正常向上冒泡，不被吞掉）。
func wrapNativeCallable(vm *goja.Runtime, fn goja.Callable) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		release := jsEnterNative(vm)
		defer release()
		res, err := fn(call.This, call.Arguments...)
		if err != nil {
			panic(err)
		}
		if res == nil {
			return goja.Undefined()
		}
		return res
	}
}

// wrapNativeGlobals 包装 VM 顶层全局函数（workflow 的 agent() 等阻塞型全局）。
func wrapNativeGlobals(vm *goja.Runtime, names ...string) {
	if vm == nil {
		return
	}
	g := vm.GlobalObject()
	for _, name := range names {
		val := g.Get(name)
		if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
			continue
		}
		if fn, ok := goja.AssertFunction(val); ok {
			_ = g.Set(name, wrapNativeCallable(vm, fn))
		}
	}
}

// applyNativeGuards 对 ctx 对象按服务名单批量包装（buildContextObject 收尾调用）。
func applyNativeGuards(vm *goja.Runtime, ctxObj *goja.Object) {
	if vm == nil || ctxObj == nil {
		return
	}
	for _, name := range jsNativeGuardServices {
		val := ctxObj.Get(name)
		if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
			continue
		}
		if o, ok := val.(*goja.Object); ok {
			wrapNativeGuards(vm, o, 1)
		}
	}
}
