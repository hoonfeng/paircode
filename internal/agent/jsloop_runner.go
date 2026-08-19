package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"wb-ui/goja"
)

// ── jsLoopRunner：JS 循环的能力代理（Go 能力 → JS 对象）──
//
// buildArgs 构造传给 JS run 的参数字典：
//
//	{
//	  task, msgs, tools, meta, loop
//	}
//
// loop 为能力代理对象，全部方法均为 Go 回调（同步阻塞、持 VM 锁执行）：
//
//	llm.chat(msgs, tools, onChunk) → assistant      // Provider.Chat 流式
//	tools.list() → [ToolDefinition]                 // ApplyConcise 后的定义
//	tools.run(name, argsJson) → {content, error}    // Registry.Execute
//	tools.runParallel([{id,name,args}]) → 结果|null // 纯只读并行（含写退回串行）
//	events.emit(event)                              // l.emit（自动回填 turn/step）
//	persist.batch(msgs)                             // OnBatchPersist + currentMsgs 同步
//	approve.ask(tc) → {approved, feedback, blocked, reason}  // 审核门（黑白名单/mode/连续驳回）
//	context.build(msgs, ephemeral) → callMsgs       // buildCallContext（含背景注入/日志）
//	compact(msgs) → msgs                            // maybeCompact（Run 内手动压缩）
//	circling.track(name, args, failed)              // 绕圈签名记录
//	circling.detect() → nudge|""                    // 绕圈检测
//	store.get(key) / store.set(key, value)          // l.State 跨 Run 共享
//	ctrl.*（暂停/停止/队列/钩子/日志/step 边界）
type jsLoopRunner struct {
	loop *Loop
	ctx  context.Context
	impl *jsLoopImpl
}

// buildArgs 构造传给 JS run 的完整参数字典。
func (r *jsLoopRunner) buildArgs(task string, msgs []Message, tools []ToolDefinition, max int) map[string]any {
	l := r.loop
	vm := r.impl.vm
	return map[string]any{
		"task":  task,
		"msgs":  msgsToJS(vm, msgs),
		"tools": toolsToJS(vm, tools),
		"meta": map[string]any{
			"maxIterations":        max,
			"autonomous":           l.Autonomous,
			"reviewMode":           l.getReviewMode(),
			"workspaceRoot":        l.WorkspaceRoot,
			"system":               l.System,
			"maxContextTokens":     l.MaxContextTokens,
			"maxAutonomousMinutes": l.maxAutonomousMinutes,
			"checkpointInterval":   l.checkpointInterval,
			"reviewBlacklist":      l.ReviewBlacklist,
			"reviewWhitelist":      l.ReviewWhitelist,
			"turn":                 l.TurnNo,
			"compressedSummaries":  l.CompressedSummaries,
		},
		"loop": r.buildProxy(),
	}
}

// toolsToJS 工具定义数组 → JS 数组。
func toolsToJS(vm *goja.Runtime, tools []ToolDefinition) []any {
	out := make([]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
			"type": t.Type,
			"function": map[string]any{
				"name":        t.Function.Name,
				"description": t.Function.Description,
				"parameters":  t.Function.Parameters,
			},
		})
	}
	return out
}

// buildProxy 构建 loop 能力代理对象。
func (r *jsLoopRunner) buildProxy() *goja.Object {
	l := r.loop
	vm := r.impl.vm
	proxy := vm.NewObject()

	// ── llm.chat(msgs, tools, onChunk) → assistantMessage ──
	llmObj := vm.NewObject()
	llmObj.Set("chat", func(call goja.FunctionCall) goja.Value {
		jmsgs, jerr := jsToMsgs(vm, call.Argument(0))
		if jerr != nil {
			panic(vm.NewGoError(jerr))
		}
		var jtools []ToolDefinition
		if v := call.Argument(1); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			if exp := v.Export(); exp != nil {
				if arr, ok := exp.([]any); ok {
					jtools = jsToolsToGo(arr)
				}
			}
		}
		onChunkVal := call.Argument(2)
		var onChunkFn goja.Callable
		if onChunkVal != nil && !goja.IsUndefined(onChunkVal) && !goja.IsNull(onChunkVal) {
			onChunkFn, _ = goja.AssertFunction(onChunkVal)
		}

		var stopReason string
		callStart := time.Now()
		log.Printf("[loop-js] LLM 调用开始 turn=%d step=%d provider=%s msgs=%d tools=%d",
			l.TurnNo, l.StepNo, l.Provider.Name(), len(jmsgs), len(jtools))
		assistant, cerr := l.Provider.Chat(r.ctx, jmsgs, jtools, func(c Chunk) {
			if c.StopReason != "" {
				stopReason = c.StopReason
			}
			// 流式片段透传给 JS（JS 决定 emit thinking/content）
			if onChunkFn != nil {
				chunkObj := map[string]any{
					"content":    c.Content,
					"reasoning":  c.Reasoning,
					"stopReason": c.StopReason,
					"done":       c.Done,
				}
				if c.Usage != nil {
					chunkObj["usage"] = map[string]any{
						"promptTokens":          c.Usage.PromptTokens,
						"completionTokens":      c.Usage.CompletionTokens,
						"totalTokens":           c.Usage.TotalTokens,
						"promptCacheHitTokens":  c.Usage.PromptCacheHitTokens,
						"promptCacheMissTokens": c.Usage.PromptCacheMissTokens,
					}
				}
				if _, e := onChunkFn(goja.Undefined(), vm.ToValue(chunkObj)); e != nil {
					log.Printf("[loop-js] onChunk 回调异常: %v", e)
				}
			}
			// Go 侧公共流式处理：usage 记录/估算/落盘/事件（与 Go Run 行为一致）
			if c.Usage != nil && c.Usage.PromptTokens > 0 {
				l.lastPromptTokens = c.Usage.PromptTokens
				usage := *c.Usage
				if usage.PromptBreakdown.SystemTokens == 0 {
					pb := EstimateBreakdown(jmsgs, l.Registry.Definitions(), usage.PromptTokens)
					usage.PromptBreakdown = pb
				}
				l.emit(Event{Type: EventUsage, Usage: &usage})
				if l.cacheDiagOn {
					l.emitCacheUsage(&usage)
				}
				if l.WorkspaceRoot != "" {
					SaveTokenUsageForRoot(l.WorkspaceRoot, &usage)
				} else {
					SaveTokenUsage(&usage)
				}
			}
		})
		if cerr != nil {
			log.Printf("[loop-js] LLM 调用失败 turn=%d step=%d 耗时=%s err=%v",
				l.TurnNo, l.StepNo, time.Since(callStart).Round(time.Millisecond), cerr)
			panic(vm.NewGoError(fmt.Errorf("LLM 调用失败: %w", cerr)))
		}
		log.Printf("[loop-js] LLM 调用完成 turn=%d step=%d 耗时=%s stop=%s len=%d",
			l.TurnNo, l.StepNo, time.Since(callStart).Round(time.Millisecond), stopReason, len(assistant.Content))
		return vm.ToValue(map[string]any{
			"role":       string(assistant.Role),
			"content":    assistant.Content,
			"reasoning":  assistant.Reasoning,
			"stopReason": stopReason,
			"toolCalls": func() []any {
				if len(assistant.ToolCalls) == 0 {
					return nil
				}
				out := make([]any, 0, len(assistant.ToolCalls))
				for _, tc := range assistant.ToolCalls {
					out = append(out, map[string]any{
						"id":   tc.ID,
						"type": tc.Type,
						"function": map[string]any{
							"name":      tc.Function.Name,
							"arguments": tc.Function.Arguments,
						},
					})
				}
				return out
			}(),
		})
	})
	proxy.Set("llm", llmObj)

	// ── tools.list() / tools.run(name, argsJson) ──
	toolsObj := vm.NewObject()
	toolsObj.Set("list", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(toolsToJS(vm, l.Registry.Definitions()))
	})
	toolsObj.Set("run", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		argsJSON := ""
		if v := call.Argument(1); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			argsJSON = v.String()
		}
		result, terr := l.Registry.Execute(r.ctx, name, argsJSON)
		out := map[string]any{"content": result, "error": nil}
		if terr != nil {
			out["content"] = "Error: " + terr.Error()
			out["error"] = terr.Error()
		}
		return vm.ToValue(out)
	})
	// tools.runParallel([{id, name, args}, ...]) → 结果数组 或 null
	// 契约：调用方（JS）负责先逐个 emit tool_call；本函数仅对「纯只读」工具
	//   并行执行（与 Go 默认 canParallelize 保守策略一致：含写/需审批 → 返回
	//   null 退回串行）。执行后按传入顺序 emit tool_result + trackCall，
	//   返回 [{id, name, content, error}]。调用方收到结果后只负责组装 tool
	//   消息（不再 emit / 不再 track，避免重复）。
	toolsObj.Set("runParallel", func(call goja.FunctionCall) goja.Value {
		v := call.Argument(0)
		if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
			return goja.Null()
		}
		exp := v.Export()
		arr, ok := exp.([]any)
		if !ok || len(arr) < 2 {
			return goja.Null()
		}
		calls := make([]ToolCall, 0, len(arr))
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				return goja.Null()
			}
			id, _ := m["id"].(string)
			name, _ := m["name"].(string)
			args := ""
			if s, ok := m["args"].(string); ok {
				args = s
			}
			if name == "" {
				return goja.Null()
			}
			calls = append(calls, ToolCall{ID: id, Function: FunctionCall{Name: name, Arguments: args}})
		}
		// 仅纯只读才并行（含写/需审批 → 退回串行）
		for _, tc := range calls {
			t, ok := l.Registry.Get(tc.Function.Name)
			if !ok || !t.ReadOnly || t.RequiresApproval {
				return goja.Null()
			}
		}
		log.Printf("[loop-js] 并行执行 %d 个只读工具（turn=%d step=%d）", len(calls), l.TurnNo, l.StepNo)
		// 并行执行（结果按原始顺序收集）
		type presult struct {
			tc     ToolCall
			output string
			err    error
		}
		results := make([]presult, len(calls))
		var wg sync.WaitGroup
		for i, tc := range calls {
			wg.Add(1)
			go func(idx int, tc ToolCall) {
				defer wg.Done()
				out, err := l.Registry.Execute(r.ctx, tc.Function.Name, tc.Function.Arguments)
				results[idx] = presult{tc: tc, output: out, err: err}
			}(i, tc)
		}
		wg.Wait()
		// 按序 emit tool_result + trackCall + 组装返回值
		outArr := make([]any, 0, len(calls))
		for _, pr := range results {
			output := pr.output
			if pr.err != nil {
				output = "Error: " + pr.err.Error()
			}
			l.emit(Event{Type: EventToolResult, Tool: pr.tc.Function.Name, Content: output, CallID: pr.tc.ID})
			l.trackCall(pr.tc.Function.Name, pr.tc.Function.Arguments, pr.err != nil || strings.HasPrefix(strings.TrimSpace(output), "Error:"))
			if pr.tc.Function.Name == "generate_commit_message" {
				l.commitMessage = output
			}
			errAny := any(nil)
			if pr.err != nil {
				errAny = pr.err.Error()
			}
			outArr = append(outArr, map[string]any{
				"id":      pr.tc.ID,
				"name":    pr.tc.Function.Name,
				"content": output,
				"error":   errAny,
			})
		}
		return vm.ToValue(outArr)
	})
	proxy.Set("tools", toolsObj)

	// ── events.emit(event) ──
	eventsObj := vm.NewObject()
	eventsObj.Set("emit", func(call goja.FunctionCall) goja.Value {
		v := call.Argument(0)
		if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
			return goja.Undefined()
		}
		exp := v.Export()
		obj, ok := exp.(map[string]any)
		if !ok {
			panic(vm.NewTypeError("loop.events.emit: 期望事件对象 {type, content, tool, args, callId, ...}"))
		}
		e := Event{}
		if s, ok := obj["type"].(string); ok {
			e.Type = EventType(s)
		}
		if s, ok := obj["content"].(string); ok {
			e.Content = s
		}
		if s, ok := obj["tool"].(string); ok {
			e.Tool = s
		}
		if s, ok := obj["args"].(string); ok {
			e.Args = s
		}
		if s, ok := obj["callId"].(string); ok {
			e.CallID = s
		}
		if s, ok := obj["callID"].(string); ok {
			e.CallID = s
		}
		if s, ok := obj["doneReason"].(string); ok {
			e.DoneReason = s
		}
		if s, ok := obj["turnReason"].(string); ok {
			e.TurnReason = s
		}
		if s, ok := obj["agentName"].(string); ok {
			e.AgentName = s
		}
		l.emit(e)
		// ★ 结构化 turn 结束原因同步：JS 发 done 事件时携带 turnReason →
		//   同步到 l.LastTurnReason（endTurn defer 据此判断不补发 notice，
		//   事件流末尾保持 done 结尾，与 Go 循环行为一致）。
		if e.Type == EventDone && e.TurnReason != "" && l.LastTurnReason == "" {
			l.LastTurnReason = TurnEndReason(e.TurnReason)
		}
		return goja.Undefined()
	})
	proxy.Set("events", eventsObj)

	// ── persist.batch(msgs) ──
	persistObj := vm.NewObject()
	persistObj.Set("batch", func(call goja.FunctionCall) goja.Value {
		jmsgs, jerr := jsToMsgs(vm, call.Argument(0))
		if jerr != nil {
			panic(vm.NewGoError(jerr))
		}
		l.currentMsgs = jmsgs
		if l.OnBatchPersist != nil {
			l.OnBatchPersist(jmsgs)
		}
		return goja.Undefined()
	})
	proxy.Set("persist", persistObj)

	// ── approve.ask(tc) → {approved, feedback, blocked, reason} ──
	approveObj := vm.NewObject()
	approveObj.Set("ask", func(call goja.FunctionCall) goja.Value {
		tc, terr := jsToTC(call.Argument(0).Export())
		if terr != nil {
			panic(vm.NewGoError(terr))
		}
		approved, feedback := l.resolveApproval(r.ctx, tc)
		if approved {
			// 通过 → 重置驳回追踪（若正是上次被驳回的工具）
			if tc.Function.Name == l.lastRejectedTool {
				l.rejectionCount = 0
				l.lastRejectedTool = ""
			}
			return vm.ToValue(map[string]any{"approved": true, "feedback": ""})
		}
		// 驳回 → 连续驳回追踪（同一工具连续 3 次 → 自动停止）
		if tc.Function.Name == l.lastRejectedTool {
			l.rejectionCount++
		} else {
			l.rejectionCount = 1
			l.lastRejectedTool = tc.Function.Name
		}
		blocked := false
		reason := ""
		if l.rejectionCount >= 3 {
			blocked = true
			reason = "操作 " + tc.Function.Name + " 已被连续驳回 3 次，自动停止"
		}
		return vm.ToValue(map[string]any{
			"approved": false,
			"feedback": feedback,
			"blocked":  blocked,
			"reason":   reason,
		})
	})
	proxy.Set("approve", approveObj)

	// ── context.build(msgs, ephemeral) → callMsgs ──
	ctxObj := vm.NewObject()
	ctxObj.Set("build", func(call goja.FunctionCall) goja.Value {
		jmsgs, jerr := jsToMsgs(vm, call.Argument(0))
		if jerr != nil {
			panic(vm.NewGoError(jerr))
		}
		// 暂存 JS ephemeral 消息到 l.ephemeralMsgs，由 buildCallContext 合并
		if v := call.Argument(1); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			eph, eerr := jsToMsgs(vm, v)
			if eerr != nil {
				panic(vm.NewGoError(eerr))
			}
			l.ephemeralMsgs = eph
		} else {
			l.ephemeralMsgs = nil
		}
		callMsgs := l.buildCallContext(jmsgs)
		return vm.ToValue(msgsToJS(vm, callMsgs))
	})
	proxy.Set("context", ctxObj)

	// ── compact(msgs) → msgs ──
	proxy.Set("compact", func(call goja.FunctionCall) goja.Value {
		jmsgs, jerr := jsToMsgs(vm, call.Argument(0))
		if jerr != nil {
			panic(vm.NewGoError(jerr))
		}
		out := l.maybeCompact(r.ctx, jmsgs)
		return vm.ToValue(msgsToJS(vm, out))
	})

	// ── circling.track / circling.detect ──
	circlingObj := vm.NewObject()
	circlingObj.Set("track", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		args := ""
		if v := call.Argument(1); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			args = v.String()
		}
		failed := false
		if v := call.Argument(2); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			failed = v.ToBoolean()
		}
		l.trackCall(name, args, failed)
		return goja.Undefined()
	})
	circlingObj.Set("detect", func(call goja.FunctionCall) goja.Value {
		nudge := l.detectCircling()
		if nudge != "" {
			l.recentCalls = nil // 提示后清零，给新思路干净起点
		}
		return vm.ToValue(nudge)
	})
	proxy.Set("circling", circlingObj)

	// ── store.get / store.set（l.State 跨 Run 共享）──
	storeObj := vm.NewObject()
	storeObj.Set("get", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		if l.State == nil {
			return goja.Undefined()
		}
		v, ok := l.State[key]
		if !ok {
			return goja.Undefined()
		}
		return vm.ToValue(v)
	})
	storeObj.Set("set", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		val := call.Argument(1).Export()
		if l.State == nil {
			l.State = map[string]any{}
		}
		l.State[key] = val
		return goja.Undefined()
	})
	proxy.Set("store", storeObj)

	// ── delegate({task, system?, maxIterations?, agentName?}) → 子 agent ──
	// 子 Loop 复用父 Provider/Registry，独立消息历史；事件经 SubAgentSink 过滤
	// （丢弃子生命周期事件，工具/思考/内容/用量转发并标记 agentName）。
	// 子 Loop 同样走 JS 循环（CurrentJSLoop 生效），可嵌套（深度限制 3 层）。
	delegateObj := vm.NewObject()
	delegateObj.Set("run", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
			panic(vm.NewTypeError("loop.delegate.run: 需要一个对象 {task, system?, maxIterations?, agentName?}"))
		}
		obj := arg.ToObject(vm)
		task := obj.Get("task").String()
		if task == "" {
			panic(vm.NewTypeError("loop.delegate.run: task 不能为空"))
		}
		agentName := "sub"
		if v := obj.Get("agentName"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) && v.String() != "" {
			agentName = v.String()
		}
		// 嵌套深度限制（防无限 delegate 递归）
		depth := jsLoopDepth(r.ctx)
		if depth >= 3 {
			return vm.ToValue(map[string]any{"error": "delegate 嵌套超过 3 层上限", "content": "", "msgs": nil})
		}
		subOpts := LoopOpts{
			Provider:           l.Provider,
			Registry:           l.Registry,
			WorkspaceRoot:      l.WorkspaceRoot,
			Autonomous:         false,
			ReviewMode:         l.getReviewMode(),
			ReviewBlacklist:    l.ReviewBlacklist,
			ReviewWhitelist:    l.ReviewWhitelist,
			ReviewProvider:     l.ReviewProvider,
			MaxContextTokens:   l.MaxContextTokens,
			Compressor:         l.Compressor,
			MaxIterations:      10,
			System:             l.System,
			CompressedSummaries: nil,
		}
		if v := obj.Get("system"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) && v.String() != "" {
			subOpts.System = v.String()
		}
		if v := obj.Get("maxIterations"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) && v.ToInteger() > 0 {
			subOpts.MaxIterations = int(v.ToInteger())
		}
		sub := newLoop(subOpts)
		// 子事件经 SubAgentSink 过滤（父 EventFinal/Done/Error/Circling/Compacted 不泄漏）
		sub.OnEvent = SubAgentSink(l.OnEvent, agentName)
		sub.OnFeedback = l.OnFeedback
		// 子 Loop 深度 +1（嵌套限制）；标记父 JS 锁内（子 runWithJS 不重复加锁）
		subCtx := context.WithValue(context.WithValue(r.ctx, jsLoopDepthKey{}, depth+1), jsLoopInLockKey{}, true)
		_, subErr := sub.Run(subCtx, task, nil) // 子 Loop 自己处理 History
		// 结果：最后一条 assistant 正文（无则空）
		content := ""
		msgs := CopyHistory(sub.History)
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == RoleAssistant {
				content = msgs[i].Content
				break
			}
		}
		errStr := ""
		if subErr != nil {
			errStr = subErr.Error()
		}
		return vm.ToValue(map[string]any{
			"content": content,
			"msgs":    msgsToJS(vm, msgs),
			"error":   errStr,
		})
	})
	proxy.Set("delegate", delegateObj)

	// ── ctrl.*（暂停/停止/队列/钩子/日志/step 边界）──
	ctrlObj := vm.NewObject()

	// paused()：阻塞等待直到恢复；返回 false=被取消/停止
	ctrlObj.Set("paused", func(call goja.FunctionCall) goja.Value {
		if l.loopSvc != nil {
			if !l.loopSvc.waitIfPaused(r.ctx) {
				return vm.ToValue(false)
			}
		}
		return vm.ToValue(true)
	})
	// stopRequested()：返回停止原因（""=未请求）
	ctrlObj.Set("stopRequested", func(call goja.FunctionCall) goja.Value {
		if l.loopSvc != nil {
			if reason := l.loopSvc.shouldStop(); reason != "" {
				return vm.ToValue(reason)
			}
		}
		return vm.ToValue("")
	})
	// stop(reason)：请求停止
	ctrlObj.Set("stop", func(call goja.FunctionCall) goja.Value {
		reason := "用户请求停止"
		if v := call.Argument(0); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) && v.String() != "" {
			reason = v.String()
		}
		l.CancelCause = AgentCancelCause{Kind: CancelByPlugin, Reason: reason}
		l.LastTurnReason = TurnAborted
		return goja.Undefined()
	})
	// pause()：请求暂停（下轮迭代开始处生效）
	ctrlObj.Set("pause", func(call goja.FunctionCall) goja.Value {
		if l.loopSvc != nil {
			l.loopSvc.Pause()
		}
		return goja.Undefined()
	})
	// nextTask()：自主模式下一阶段任务
	ctrlObj.Set("nextTask", func(call goja.FunctionCall) goja.Value {
		if l.OnNextTask == nil {
			return vm.ToValue("")
		}
		return vm.ToValue(l.OnNextTask())
	})
	// feedback()：用户运行时反馈
	ctrlObj.Set("feedback", func(call goja.FunctionCall) goja.Value {
		if l.OnFeedback == nil {
			return vm.ToValue("")
		}
		return vm.ToValue(l.OnFeedback())
	})
	// steer()：清空并返回托管消息
	ctrlObj.Set("steer", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(msgsToJS(vm, l.drainSteerQueue()))
	})
	// followUp()：清空并返回跟进消息
	ctrlObj.Set("followUp", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(msgsToJS(vm, l.drainFollowUpQueue()))
	})
	// preStep(callMsgs, turn, step) → {rewritten, reject, error}
	ctrlObj.Set("preStep", func(call goja.FunctionCall) goja.Value {
		if l.PreStep == nil {
			return vm.ToValue(map[string]any{"reject": false, "error": ""})
		}
		jmsgs, jerr := jsToMsgs(vm, call.Argument(0))
		if jerr != nil {
			panic(vm.NewGoError(jerr))
		}
		turn := int(call.Argument(1).ToInteger())
		if turn == 0 {
			turn = l.TurnNo
		}
		step := int(call.Argument(2).ToInteger())
		if step == 0 {
			step = l.StepNo
		}
		rewritten, reject, perr := l.PreStep(r.ctx, jmsgs, turn, step)
		out := map[string]any{"reject": reject, "error": ""}
		if perr != nil {
			out["error"] = perr.Error()
		}
		if rewritten != nil {
			out["rewritten"] = msgsToJS(vm, rewritten)
		}
		return vm.ToValue(out)
	})
	// logEntry / logAnalysis
	ctrlObj.Set("logEntry", func(call goja.FunctionCall) goja.Value {
		agentName := call.Argument(0).String()
		phase := call.Argument(1).String()
		summary := call.Argument(2).String()
		l.LogEntry(agentName, phase, summary)
		return goja.Undefined()
	})
	ctrlObj.Set("logAnalysis", func(call goja.FunctionCall) goja.Value {
		l.LogAnalysis(call.Argument(0).String())
		return goja.Undefined()
	})
	// beginStep / endStep：turn/step 状态机边界
	ctrlObj.Set("beginStep", func(call goja.FunctionCall) goja.Value {
		l.beginStep()
		return goja.Undefined()
	})
	ctrlObj.Set("endStep", func(call goja.FunctionCall) goja.Value {
		summary := ""
		if v := call.Argument(0); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			summary = v.String()
		}
		l.endStep(summary)
		return goja.Undefined()
	})
	// stickyReason(base)：turn 结束原因（hadMaxTokens sticky 保护）
	ctrlObj.Set("stickyReason", func(call goja.FunctionCall) goja.Value {
		base := TurnEndReason(call.Argument(0).String())
		return vm.ToValue(string(l.turnStickyReason(base)))
	})
	// markMaxTokens()：本轮 turn 曾触发 max-tokens（sticky）
	ctrlObj.Set("markMaxTokens", func(call goja.FunctionCall) goja.Value {
		l.hadMaxTokens = true
		return goja.Undefined()
	})
	// emitNotice(content)：便捷通知事件
	ctrlObj.Set("emitNotice", func(call goja.FunctionCall) goja.Value {
		l.emit(Event{Type: EventNotice, Content: call.Argument(0).String()})
		return goja.Undefined()
	})
	// isCanceled()：外部 ctx 是否已取消
	ctrlObj.Set("isCanceled", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(r.ctx.Err() != nil)
	})
	// compactRequested()：外部是否请求了手动压缩
	ctrlObj.Set("compactRequested", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(l.CompactRequested)
	})
	// resetCompactRequest()：清除手动压缩请求（压缩完成后）
	ctrlObj.Set("resetCompactRequest", func(call goja.FunctionCall) goja.Value {
		l.CompactRequested = false
		return goja.Undefined()
	})
	// truncStr(s, n)：截断长字符串（进入下一阶段提示用）
	ctrlObj.Set("truncStr", func(call goja.FunctionCall) goja.Value {
		s := call.Argument(0).String()
		n := int(call.Argument(1).ToInteger())
		if n <= 0 {
			n = 80
		}
		return vm.ToValue(truncStr(s, n))
	})
	proxy.Set("ctrl", ctrlObj)

	return proxy
}

// jsToolsToGo JS 工具定义数组 → []ToolDefinition。
func jsToolsToGo(arr []any) []ToolDefinition {
	out := make([]ToolDefinition, 0, len(arr))
	for _, it := range arr {
		if obj, ok := it.(map[string]any); ok {
			var td ToolDefinition
			td.Type, _ = obj["type"].(string)
			if fn, ok := obj["function"].(map[string]any); ok {
				td.Function.Name, _ = fn["name"].(string)
				td.Function.Description, _ = fn["description"].(string)
				if p, ok := fn["parameters"].(map[string]any); ok {
					td.Function.Parameters = p
				}
			}
			out = append(out, td)
		}
	}
	return out
}

// truncStr 截断字符串（定义见 autonomous_controller.go）。
