package agent

// jsplugin_subagent.go — 子 agent 能力面的 JS 桥（ctx.subagent，能力 Go / 策略 JS）。
//
// 插件用法（host 半）：
//
//	const res = ctx.subagent.run({
//	  name: 'supervisor',        // 来源标注（事件 AgentName；前端看板分区）
//	  round: 1,                  // 回合序号（事件 turn = round，供前端按回合分组）
//	  system: '<角色提示>',       // 系统提示（策略由插件提供）
//	  task: '<任务书>',           // 起始任务
//	  tools: null,               // 工具白名单（null/[] = 继承会话全部已启用工具）
//	  stepBudget: 40, toolCallBudget: 40,
//	  timeoutMs: 900000,         // 可选：本回合超时（0 = 不限；受会话取消约束）
//	  resultTool: { name: 'submit_result', description: '...', parameters: {...} },
//	})
//	// → { content, segments, submitted, steps, toolCalls, promptTokens, outputTokens,
//	//     durationMs, error, ended }
//
// ★ 同步阻塞：宿主在插件回调栈上直接跑完整个子 agent 回合（含 LLM 调用与工具执行），
//   期间子 agent 的事件实时下发前端（带 AgentName）。工具执行走 Go 注册表；子 agent
//   循环强制 Go 路径（不能在 JS 回调栈上重入 agentloop 的 JS 循环实现，见 subagent.go）。

import (
	"encoding/json"

	"github.com/hoonfeng/paircode/goja"
)

// attachSubagentObject 在 ctx 上挂 subagent 能力面（jsplugin.go 的 buildContextObject 调用）。
func (p *jsPluginAdapter) attachSubagentObject(ctxObj *goja.Object) {
	vm := p.vm
	obj := vm.NewObject()
	obj.Set("run", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
			panic(vm.NewTypeError("ctx.subagent.run: 需要 {name, task, ...} 规格对象"))
		}
		raw, err := jsonStringify(vm, arg)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		var spec SubagentSpec
		if err := json.Unmarshal([]byte(raw), &spec); err != nil {
			panic(vm.NewGoError(err))
		}
		// ctx 传 nil：RunSubagent 内部用「当前会话运行环境」的父上下文（用户停止即中止）。
		res, rerr := RunSubagent(nil, spec)
		if rerr != nil {
			panic(vm.NewGoError(rerr))
		}
		out, jerr := jsonToJSObject(vm, res)
		if jerr != nil {
			panic(vm.NewGoError(jerr))
		}
		return out
	})
	// available()：当前是否具备会话运行环境（插件可在 decide 里先探测）。
	obj.Set("available", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(SubagentRuntimeFor("") != nil)
	})
	ctxObj.Set("subagent", obj)
}
