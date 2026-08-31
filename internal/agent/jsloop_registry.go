package agent

import (
	"sync"

	"github.com/hoonfeng/paircode/goja"
)

// ── JS 循环注册表（单槽位，对齐 setFactory 注册）──
//
// agentloop 核心外置（2026-08-19）：循环策略（turn/step 双层循环、审核决策、
// 自然终止检测、content-only/绕圈防护、背景快照组装）由 JS 插件实现；Go 保留能力
// （Provider.Chat / Registry.Execute / emit / persist / approve / buildCallContext /
// snapshot.parts+sync 等），经能力代理对象注入 JS run({task, msgs, tools, meta, loop})。
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

	// ★ 2026-08-30 并行会话：实例池归属与影子标记（见 jsloop_pool.go）。
	//   shadow=true 表示本实例是为并行会话派生的独立 Runtime 副本。
	pool   *jsLoopPool
	shadow bool
}

var (
	jsLoopMu   sync.RWMutex
	jsLoopVal  *jsLoopImpl
	jsLoopPoolCur *jsLoopPool // 当前生效实现的实例池（并行会话隔离）
)

// RegisterJSLoop 注册 JS 循环实现（后注册覆盖先注册），返回还原函数。
// ★ 同时为该实现建实例池：并行会话各租一个独立 Runtime，互不持锁。
func RegisterJSLoop(impl *jsLoopImpl) (restore func()) {
	pool := newJSLoopPool(impl)
	jsLoopMu.Lock()
	prev, prevPool := jsLoopVal, jsLoopPoolCur
	jsLoopVal, jsLoopPoolCur = impl, pool
	jsLoopMu.Unlock()
	if prevPool != nil {
		prevPool.close()
	}
	return func() {
		jsLoopMu.Lock()
		var drop *jsLoopPool
		if jsLoopVal == impl {
			jsLoopVal = prev
			drop, jsLoopPoolCur = jsLoopPoolCur, nil
		}
		jsLoopMu.Unlock()
		if drop != nil {
			drop.close()
		}
	}
}

// UnregisterJSLoop 注销指定 JS 循环（插件卸载时调用）。
func UnregisterJSLoop(impl *jsLoopImpl) {
	jsLoopMu.Lock()
	var drop *jsLoopPool
	if jsLoopVal == impl {
		jsLoopVal = nil
		drop, jsLoopPoolCur = jsLoopPoolCur, nil
	}
	jsLoopMu.Unlock()
	if drop != nil {
		drop.close()
	}
}

// CurrentJSLoop 返回当前生效的 JS 循环实现（nil=未注册，走 Go 默认循环）。
func CurrentJSLoop() *jsLoopImpl {
	jsLoopMu.RLock()
	defer jsLoopMu.RUnlock()
	return jsLoopVal
}

// CurrentJSLoopPool 返回当前生效实现的实例池（nil=未注册）。
func CurrentJSLoopPool() *jsLoopPool {
	jsLoopMu.RLock()
	defer jsLoopMu.RUnlock()
	return jsLoopPoolCur
}
