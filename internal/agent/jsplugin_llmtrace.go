package agent

// jsplugin_llmtrace.go — 给 JS 插件暴露 ctx.llmtrace（LLM 请求/响应完整追踪）。
//
// 用法（JS 插件）：
//   ctx.llmtrace.register((ev) => { ... })  // ev 见 llmTraceJS 注释；每次 register 覆盖旧回调（单注册语义）
//   ctx.llmtrace.unregister()                // 主动撤销（插件卸载时自动撤销）
//
// 回调在宿主 VM 锁内同步触发（agentloop JS 循环的 llm.chat 前后），
// 阻塞会拖慢主循环——插件侧应只做轻量持久化（如追加 JSONL）。

import (
	"log"
	"time"

	"github.com/hoonfeng/paircode/goja"
)

// attachLLMTrace 在 ctx 对象上挂 llmtrace 服务。
func (p *jsPluginAdapter) attachLLMTrace(ctxObj *goja.Object) {
	if p.vm == nil {
		return
	}
	vm := p.vm
	obj := vm.NewObject()
	var restore func()
	obj.Set("register", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(vm.NewTypeError("ctx.llmtrace.register(fn): fn 必须是函数"))
		}
		if restore != nil {
			restore()
		}
		restore = RegisterLLMTracer(func(ev LLMTraceEvent) {
			if _, e := fn(goja.Undefined(), vm.ToValue(llmTraceToJS(vm, ev))); e != nil {
				log.Printf("[jsplugin] llmtrace 回调异常: %v", e)
			}
		})
		return goja.Undefined()
	})
	obj.Set("unregister", func(call goja.FunctionCall) goja.Value {
		if restore != nil {
			restore()
			restore = nil
		}
		return goja.Undefined()
	})
	p.addCleanup(func() {
		if restore != nil {
			restore()
			restore = nil
		}
	})
	ctxObj.Set("llmtrace", obj)
}

// llmTraceToJS 把 LLMTraceEvent 转成 JS 可读对象。
// 结构：{phase, when(ISO8601), turn, step, provider, stopReason, err,
//
//	     msgs?, tools?, usage?, content?, reasoning?, toolCalls?}
//	usage = {promptTokens, completionTokens, totalTokens, promptCacheHitTokens,
//	         promptCacheMissTokens, systemTokens, skillsTokens, mcpTokens,
//	         toolTokens, historyTokens, otherTokens}
func llmTraceToJS(vm *goja.Runtime, ev LLMTraceEvent) map[string]any {
	m := map[string]any{
		"phase":      ev.Phase,
		"when":       ev.When.Format(time.RFC3339Nano),
		"turn":       ev.Turn,
		"step":       ev.Step,
		"provider":   ev.Provider,
		"stopReason": ev.StopReason,
		"err":        ev.Err,
	}
	if ev.Messages != nil {
		m["msgs"] = msgsToJS(vm, ev.Messages)
	}
	if ev.Tools != nil {
		m["tools"] = toolsToJS(vm, ev.Tools)
	}
	if ev.Usage != nil {
		u := ev.Usage
		um := map[string]any{
			"promptTokens":          u.PromptTokens,
			"completionTokens":      u.CompletionTokens,
			"totalTokens":           u.TotalTokens,
			"promptCacheHitTokens":  u.PromptCacheHitTokens,
			"promptCacheMissTokens": u.PromptCacheMissTokens,
		}
		pb := u.PromptBreakdown
		um["systemTokens"] = pb.SystemTokens
		um["skillsTokens"] = pb.SkillsTokens
		um["mcpTokens"] = pb.MCPTokens
		um["toolTokens"] = pb.ToolTokens
		um["historyTokens"] = pb.HistoryTokens
		um["otherTokens"] = pb.OtherTokens
		m["usage"] = um
	}
	if ev.Content != "" {
		m["content"] = ev.Content
	}
	if ev.Reasoning != "" {
		m["reasoning"] = ev.Reasoning
	}
	if ev.ToolCalls != nil {
		out := make([]any, 0, len(ev.ToolCalls))
		for _, tc := range ev.ToolCalls {
			out = append(out, map[string]any{
				"id":   tc.ID,
				"type": tc.Type,
				"function": map[string]any{
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				},
			})
		}
		m["toolCalls"] = out
	}
	return m
}
