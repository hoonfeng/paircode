package agent

// llmtrace.go — LLM 请求/响应完整追踪（缓存前缀断裂分析数据面）。
//
// 背景（2026-09）：用户需要 LLM 请求完整日志（messages + tools + 响应 + usage）
// 用于分析 provider 前缀缓存断裂（系统提示/工具定义变化导致跨轮次缓存 miss）。
// 现有诊断（WB_CACHE_DIAG=1 的 emitCacheShape）只记录形状哈希，无完整内容。
//
// 设计：注册表模式（对齐 RegisterLoopHook/RegisterJSLoop）。
//   - RegisterLLMTracer(fn)：宿主代码/JS 插件注册追踪回调（多注册者并存）；
//   - FireLLMTracer(ev)：主循环 LLM 调用前后触发（request/response 两相），
//     panic 隔离——追踪器异常不影响主循环。
//
// 触发点（2026-09 首版）：agentloop JS 循环的 llm.chat（jsloop_runner.go）——
//   该路径运行在 goja VM 锁内，JS 插件注册的追踪回调可安全同步调用。
//   Go 默认循环（已 deprecated 回退路径）暂不触发，避免跨 VM 锁调用 goja 函数。

import (
	"log"
	"sync"
	"time"
)

// LLMTraceEvent 一次 LLM 调用的追踪事件（request 或 response 一相）。
// Messages/Tools 仅 request 相非空；Usage/Content/Reasoning/ToolCalls 仅
// response 相非空；Err 非空 = 调用失败（response 相）。
type LLMTraceEvent struct {
	Phase      string    // "request" | "response"
	When       time.Time // 该相时间戳
	Turn       int       // 主循环 turn（agentloop：一次 Run = 一个 turn）
	Step       int       // 主循环 step（每次 LLM 调用 + 工具执行 = 一个 step）
	Provider   string    // provider 名（含模型：如 openai:deepseek-chat）
	Messages   []Message // 完整请求消息（request 相）
	Tools      []ToolDefinition
	Usage      *Usage // 返回 token 用量（含缓存命中/未命中，response 相）
	Content    string // 响应正文（response 相）
	Reasoning  string // 思考链（response 相）
	ToolCalls  []ToolCall
	StopReason string
	Err        string // 调用错误（response 相）
}

// llmTrace 注册表（全局；多插件/宿主代码可并存注册）。
var (
	llmTraceMu      sync.RWMutex
	llmTraceSeq     int
	llmTraceEntries []llmTraceEntry
)

// llmTraceEntry 一次注册（id 令牌保证精确撤销——函数值不可比较）。
type llmTraceEntry struct {
	id int
	fn func(LLMTraceEvent)
}

// RegisterLLMTracer 注册 LLM 请求/响应追踪回调。返回撤销函数（插件卸载时调用）。
func RegisterLLMTracer(fn func(LLMTraceEvent)) (restore func()) {
	if fn == nil {
		return func() {}
	}
	llmTraceMu.Lock()
	llmTraceSeq++
	id := llmTraceSeq
	llmTraceEntries = append(llmTraceEntries, llmTraceEntry{id: id, fn: fn})
	llmTraceMu.Unlock()
	return func() {
		llmTraceMu.Lock()
		defer llmTraceMu.Unlock()
		for i, e := range llmTraceEntries {
			if e.id == id {
				llmTraceEntries = append(llmTraceEntries[:i], llmTraceEntries[i+1:]...)
				return
			}
		}
	}
}

// FireLLMTracer 依次通知所有追踪器（panic 隔离，不影响主循环）。
func FireLLMTracer(ev LLMTraceEvent) {
	llmTraceMu.RLock()
	fns := make([]func(LLMTraceEvent), 0, len(llmTraceEntries))
	for _, e := range llmTraceEntries {
		fns = append(fns, e.fn)
	}
	llmTraceMu.RUnlock()
	if len(fns) == 0 {
		return
	}
	if ev.When.IsZero() {
		ev.When = time.Now()
	}
	for _, fn := range fns {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[llmtrace] 追踪器 panic 已隔离: %v", r)
				}
			}()
			fn(ev)
		}()
	}
}
