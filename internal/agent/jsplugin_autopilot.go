package agent

// jsplugin_autopilot.go — 自主模式决策器的 JS 桥（ctx.loopFactory.registerAutopilot）。
//
// 与 registerLoop / registerHandoff 同模式：单槽位注册、后注册覆盖、卸载自动还原。
//
//	ctx.loopFactory.registerAutopilot({
//	  id: 'autopilot',
//	  decide: async (req) => {
//	    // req = { convId, workspaceRoot, round, maxRounds, objective,
//	    //         workerReport, workerTurns, recentHistory }
//	    // 插件在此扮演「人」：审核/评判/决定下一步。
//	    //   可经 ctx.subagent.run(...) 派生一个带工具的「监督者」回合自行核查。
//	    return { action: 'continue'|'done', task?: '<下一步指令>', assessment?: '<评判>' }
//	  },
//	})
//
// 宿主（SessionManager 续轮处）在「工作 agent 自然结束且自主开关注」时调用：
// continue → 把 task 作为新任务唤醒工作 agent；done/未返回 → 整轮结束。

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hoonfeng/paircode/goja"
)

// jsAutopilotImpl 一个已注册的 JS 自主模式决策器。
type jsAutopilotImpl struct {
	id  string
	fn  goja.Callable // async (req) → {action, task?, assessment?}
	plugin *jsPluginAdapter
	vm     *goja.Runtime
}

// decide 调用决策器（持 VM 锁；同步等待 Promise）。
func (h *jsAutopilotImpl) decide(ctx context.Context, req AutopilotRequest) (AutopilotDecision, error) {
	var (
		out AutopilotDecision
		err error
	)
	run := func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("JS 自主决策器 panic: %v", r)
			}
		}()
		args, jerr := jsonToJSObject(h.vm, req)
		if jerr != nil {
			err = jerr
			return
		}
		v, cerr := h.fn(goja.Undefined(), args)
		if cerr != nil {
			err = cerr
			return
		}
		av, aerr := awaitJSValue(h.vm, v) // async 函数 → 同步等待（微任务 drain）
		if aerr != nil {
			err = aerr
			return
		}
		if av == nil || goja.IsUndefined(av) || goja.IsNull(av) {
			out = AutopilotDecision{}
			return
		}
		s, serr := jsonStringify(h.vm, av)
		if serr != nil {
			err = serr
			return
		}
		if strings.TrimSpace(s) == "" || s == "null" {
			return
		}
		if uerr := json.Unmarshal([]byte(s), &out); uerr != nil {
			err = fmt.Errorf("决策返回值解析失败: %w（原始值：%s）", uerr, truncStr(s, 200))
		}
	}
	// goja 非并发安全：JS 调用必须持 VM 锁。plugin 为 nil（单测）时直接执行。
	if h.plugin != nil {
		h.plugin.withLock(run)
	} else {
		run()
	}
	if err != nil {
		return AutopilotDecision{}, fmt.Errorf("自主决策器 %s 调用失败: %w", h.id, err)
	}
	return out, nil
}

// attachAutopilotRegister 在 ctx.loopFactory 上挂 registerAutopilot
// （jsplugin.go 的 buildContextObject 里 loopFactoryObj 创建后调用）。
func (p *jsPluginAdapter) attachAutopilotRegister(loopFactoryObj *goja.Object) {
	vm := p.vm
	loopFactoryObj.Set("registerAutopilot", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
			panic(vm.NewTypeError("ctx.loopFactory.registerAutopilot: 需要函数 decide(req) 或 {id?, decide}"))
		}
		id := p.def.name
		decideVal := arg
		// {id?, decide} 形态：入参是对象且不是函数时取 decide 字段
		if _, isFn := goja.AssertFunction(arg); !isFn {
			obj := arg.ToObject(vm)
			if v := obj.Get("id"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) && v.String() != "" {
				id = v.String()
			}
			decideVal = obj.Get("decide")
		}
		fn, ok := goja.AssertFunction(decideVal)
		if !ok {
			panic(vm.NewTypeError("ctx.loopFactory.registerAutopilot: decide 必须是函数 (req) → {action, task?, assessment?}"))
		}
		// ★ 影子实例（并行会话 VM 副本，jsloop_pool）：不写全局槽位——
		//   影子 VM 随会话结束销毁，全局槽位指向已销毁 Runtime 会在下次调用时崩。
		if p.shadow {
			return vm.ToValue(map[string]any{"id": id, "ok": true, "shadow": true})
		}
		impl := &jsAutopilotImpl{id: id, fn: fn, plugin: p, vm: vm}
		restore := RegisterJSAutopilot(impl)
		p.addCleanup(func() { UnregisterJSAutopilot(impl) })
		_ = restore
		p.def.addDiag(fmt.Sprintf("注册自主模式决策器 %q（工作 agent 自然结束时宿主调用，决定继续/完成）", id))
		log.Printf("[js-plugin:%s] registerAutopilot: 已注册自主模式决策器 %q（自主模式由插件驱动）", p.def.id, id)
		return vm.ToValue(map[string]any{"id": id, "ok": true})
	})
}
