package agent

import (
	"sync"

	"wb-ui/goja"
)

// ── JS 循环注册表（单槽位，对齐 harness AgentRegistry.setFactory）──
//
// agentloop 核心外置（2026-08-19）：循环策略（turn/step 双层循环、审核决策、
// 自然终止检测、content-only/绕圈防护）由 JS 插件实现；Go 保留能力
// （Provider.Chat / Registry.Execute / emit / persist / approve / buildCallContext 等），
// 经能力代理对象注入 JS run({task, msgs, tools, meta, loop})。
//
// 注册语义：
//   - ctx.loopFactory.registerLoop({id, run}) → RegisterJSLoop（后注册覆盖先注册）
//   - 插件卸载 → 自动还原（UnregisterJSLoop）
//   - Loop.Run 入口检查 CurrentJSLoop()：非空 → runWithJS 委托；空 → 现有 Go 循环
//
// 可回退：停用/卸载插件即还原 Go 循环，零影响。

// jsLoopImpl 一个已注册的 JS 循环实现。
type jsLoopImpl struct {
	id     string
	run    goja.Callable // async ({task, msgs, tools, meta, loop}) → {msgs, error?}
	plugin *jsPluginAdapter
	vm     *goja.Runtime
}

var (
	jsLoopMu  sync.RWMutex
	jsLoopVal *jsLoopImpl
)

// RegisterJSLoop 注册 JS 循环实现（后注册覆盖先注册），返回还原函数。
func RegisterJSLoop(impl *jsLoopImpl) (restore func()) {
	jsLoopMu.Lock()
	prev := jsLoopVal
	jsLoopVal = impl
	jsLoopMu.Unlock()
	return func() {
		jsLoopMu.Lock()
		if jsLoopVal == impl {
			jsLoopVal = prev
		}
		jsLoopMu.Unlock()
	}
}

// UnregisterJSLoop 注销指定 JS 循环（插件卸载时调用）。
func UnregisterJSLoop(impl *jsLoopImpl) {
	jsLoopMu.Lock()
	if jsLoopVal == impl {
		jsLoopVal = nil
	}
	jsLoopMu.Unlock()
}

// CurrentJSLoop 返回当前生效的 JS 循环实现（nil=未注册，走 Go 默认循环）。
func CurrentJSLoop() *jsLoopImpl {
	jsLoopMu.RLock()
	defer jsLoopMu.RUnlock()
	return jsLoopVal
}
