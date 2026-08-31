package agent

import (
	"fmt"
	"log"

	"github.com/hoonfeng/paircode/goja"
)

// ── ctx.loopFactory.registerLoop(impl)：注册 JS 循环实现（agentloop 核心外置）──
//
// 对齐 setFactory 装配：整体替换循环实现 = 装配期
// 换一个实现 AgentFactory 的插件。本服务把 JS 插件声明为「循环实现」：
//
//	ctx.loopFactory.registerLoop({
//	  id: 'agentloop',
//	  run: async ({ task, msgs, tools, meta, loop }) => {
//	    // 循环策略：turn/step、LLM 调用、工具执行、审核、自然终止检测……
//	    return { msgs, error: undefined };  // msgs = 完整消息列表
//	  },
//	});
//
// 与现有 register(apply)（参数级装配器）互补：
//   - register(apply)：CreateLoop 时覆盖装配参数（提示词/迭代上限/审核模式…）
//   - registerLoop(impl)：Run 时接管循环业务（Go 保留能力，JS 写策略）
//
// 卸载 → 自动还原 Go 默认循环；停用插件即回退，零风险。

// attachLoopRegister 在 ctx.loopFactory 对象上挂 registerLoop 方法（jsplugin.go
// 的 buildContextObject 里 loopFactoryObj 创建后调用）。
func (p *jsPluginAdapter) attachLoopRegister(loopFactoryObj *goja.Object) {
	vm := p.vm
	loopFactoryObj.Set("registerLoop", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
			panic(vm.NewTypeError("ctx.loopFactory.registerLoop: 需要一个对象 {id, run}"))
		}
		obj := arg.ToObject(vm)
		id := obj.Get("id").String()
		if id == "" {
			id = p.def.name
		}
		runVal := obj.Get("run")
		runFn, ok := goja.AssertFunction(runVal)
		if !ok {
			panic(vm.NewTypeError("ctx.loopFactory.registerLoop: run 必须是 async 函数 ({task, msgs, tools, meta, loop}) → {msgs, error?}"))
		}

		impl := &jsLoopImpl{id: id, run: runFn, plugin: p, vm: vm}
		// ★ 2026-08-30 影子实例（并行会话隔离）：只把实现捕获给本影子，
		//   不写全局注册表、不产生卸载钩子（主实例已注册；见 jsloop_pool.go）。
		if p.shadow {
			p.shadowLoop = impl
			return vm.ToValue(map[string]any{"id": id, "ok": true, "shadow": true})
		}
		restore := RegisterJSLoop(impl)
		// 插件卸载时还原：若当前生效的是本实现则还原到之前的实现
		p.addCleanup(func() { UnregisterJSLoop(impl) })
		_ = restore
		p.def.addDiag(fmt.Sprintf("注册 JS 循环实现 %q（Loop.Run 委托 JS；卸载自动还原 Go 循环）", id))
		log.Printf("[js-plugin:%s] registerLoop: 已注册 JS 循环实现 %q，agent 循环将委托 JS 驱动", p.def.id, id)
		return vm.ToValue(map[string]any{"id": id, "ok": true})
	})
}
