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
//	     msgs?, tools?, usage?, usageEstimated?, content?, reasoning?, toolCalls?}
//
//	usage = {source:"api", promptTokens, completionTokens, totalTokens,
//	         promptCacheHitTokens, promptCacheMissTokens,
//	         cacheHitRate?, apiRaw?}
//	        —— **API 真实返回**的用量：前六项为归一化值，apiRaw 为 provider
//	           原始报文（未归一化，含厂商/网关扩展字段）。
//	usageEstimated = {source:"local-estimate", systemTokens, skillsTokens,
//	         mcpTokens, toolTokens, historyTokens, otherTokens}
//	        —— 本地估算的 prompt 构成（EstimateBreakdown），**仅**供占比可视化，
//	           不得当作真实用量做成本/命中率分析。
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
		// ★ usage = **API 真实返回**的用量（provider 报文经归一化后的值；不含任何本地估算）。
		//   source=api 显式标注来源，供分析面区分真实值与估算值。
		um := map[string]any{
			"source":                "api",
			"promptTokens":          u.PromptTokens,
			"completionTokens":      u.CompletionTokens,
			"totalTokens":           u.TotalTokens,
			"promptCacheHitTokens":  u.PromptCacheHitTokens,
			"promptCacheMissTokens": u.PromptCacheMissTokens,
		}
		// 命中率（派生值：hit/(hit+miss)；分母为 0 时不输出，避免伪造 0%）
		if d := u.PromptCacheHitTokens + u.PromptCacheMissTokens; d > 0 {
			um["cacheHitRate"] = float64(u.PromptCacheHitTokens) / float64(d)
		}
		// ★ apiRaw = provider **原始报文**（未归一化、未推导）——含 DeepSeek 的
		//   prompt_cache_hit_tokens、OpenAI 的 prompt_tokens_details.cached_tokens、
		//   以及各网关/厂商的扩展字段；核对真实用量时以此为准。
		if len(u.Raw) > 0 {
			um["apiRaw"] = u.Raw
		}
		m["usage"] = um
		// ★ usageEstimated = 本地**估算**的 prompt 构成（非 API 返回）。
		//   仅用于前端占比可视化参考，**不得**当作真实用量参与成本/命中率分析。
		if pb := u.PromptBreakdown; pb != (PromptBreakdown{}) {
			m["usageEstimated"] = map[string]any{
				"source":        "local-estimate",
				"systemTokens":  pb.SystemTokens,
				"skillsTokens":  pb.SkillsTokens,
				"mcpTokens":     pb.MCPTokens,
				"toolTokens":    pb.ToolTokens,
				"historyTokens": pb.HistoryTokens,
				"otherTokens":   pb.OtherTokens,
			}
		}
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
