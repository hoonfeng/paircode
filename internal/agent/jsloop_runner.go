package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/goja"
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
//	approve.ask(tc) → {approved, feedback}                 // 审核门（黑白名单/mode；驳回记录进 state）
//	approve.state.get() / approve.state.set(obj)          // 共享审核状态（最近驳回/历史，不计数）
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
		// ★ 2026-08-27 首步极简工具面（实测改进）：会话首个 Run 的首个 LLM 调用
		//   只注入极简核心工具，自第 2 个 step 起恢复完整工具面（tools_staging.go）。
		//   依据：极简面首步选对率 91.7% vs 全量 87.5%（48 次采样），
		//   且首步 token 开销显著降低。
		if l.StagedTools && l.TurnNo <= 1 && l.StepNo <= 1 {
			jtools = FilterStagedTools(jtools, l.StagedToolGroups)
		}
		callStart := time.Now()
		log.Printf("[loop-js] LLM 调用开始 turn=%d step=%d provider=%s msgs=%d tools=%d",
			l.TurnNo, l.StepNo, l.getProvider().Name(), len(jmsgs), len(jtools))
		assistant, cerr := l.getProvider().Chat(r.ctx, jmsgs, jtools, func(c Chunk) {
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
		} else {
			// ★ 图片提交（2026-08-22）：工具结果含 submit_image 标记 → 读图挂
			//   pendingImages（标记剥离，净化文本给 JS 循环组装 tool 消息）。
			out["content"] = l.parseImageSubmitResult(result)
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
			} else {
				// ★ 图片提交（2026-08-22）：工具结果含 submit_image 标记 → 读图挂
				//   pendingImages（标记剥离，净化文本给 JS 循环组装 tool 消息）。
				output = l.parseImageSubmitResult(output)
			}
			l.emit(Event{Type: EventToolResult, Tool: pr.tc.Function.Name, Content: output, CallID: pr.tc.ID})
			l.trackCall(pr.tc.Function.Name, pr.tc.Function.Arguments, pr.err != nil || strings.HasPrefix(strings.TrimSpace(output), "Error:"))
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
		l.currentMsgs = l.fullHistory(jmsgs) // 还原完整时间线（压缩视图仅供 LLM 提交）
		l.persist(jmsgs)
		return goja.Undefined()
	})
	proxy.Set("persist", persistObj)

	// ── approve：审核门（策略判定 JS / 动作执行 Go）──
	// ★ 2026-08-27 错误计数移除：不再有连续驳回计数/自动停止（blocked 已删除），
	//   驳回仅反馈继续（打破死循环由绕圈检测兜底）；审核决策状态改为共享上下文值
	//   approve.state（Go 会话级持有，JS get/set 读写），agentloop 审核逻辑据此决策。
	approveObj := vm.NewObject()
	// policy()：审批策略数据面（黑白名单/审核模式/需审工具映射）。
	approveObj.Set("policy", func(call goja.FunctionCall) goja.Value {
		ra := map[string]bool{}
		for _, name := range l.Registry.Names() {
			if t, ok := l.Registry.Get(name); ok && t.RequiresApproval {
				ra[name] = true
			}
		}
		return vm.ToValue(map[string]any{
			"reviewMode":       l.getReviewMode(),
			"reviewBlacklist":  append([]string(nil), l.ReviewBlacklist...),
			"reviewWhitelist":  append([]string(nil), l.ReviewWhitelist...),
			"requiresApproval": ra,
		})
	})
	// state 共享审核状态（共享上下文的值）：get 读快照 / set 合并改写。
	stateObj := vm.NewObject()
	stateObj.Set("get", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(l.getApproveState().Snapshot())
	})
	stateObj.Set("set", func(call goja.FunctionCall) goja.Value {
		if v := call.Argument(0); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			if obj, ok := v.Export().(map[string]any); ok {
				l.getApproveState().Set(obj)
			}
		}
		return goja.Undefined()
	})
	approveObj.Set("state", stateObj)
	approveObj.Set("ask", func(call goja.FunctionCall) goja.Value {
		tc, terr := jsToTC(call.Argument(0).Export())
		if terr != nil {
			panic(vm.NewGoError(terr))
		}
		approved, feedback := l.resolveApproval(r.ctx, tc)
		if approved {
			// 通过 → 清掉该工具的最近驳回标记
			l.getApproveState().clearTool(tc.Function.Name)
			return vm.ToValue(map[string]any{"approved": true, "feedback": ""})
		}
		// 驳回 → 共享状态记录（不计数）
		l.getApproveState().recordReject(tc.Function.Name, feedback)
		return vm.ToValue(map[string]any{"approved": false, "feedback": feedback})
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

	// ── context.snapshot（2026-08-27 背景快照同步，对齐 dsh RuntimeContextProjection）──
	//   · snapshotParts() → 数据面：{stale, summaries, memory, knowledge, autonomous}
	//     （文本格式组装策略在 JS 插件；Go 只提供原始数据，能力/策略分离）
	//   · snapshot.sync(msgs, text) → msgs：与历史最后快照比较；不同则追加到
	//     msgs 末尾（当前任务之后，随 tail 落盘）并立即持久化；相同零注入。
	//   ★ 快照持久化到消息流后位置固定，跨 Run 前缀单调延展——KV 缓存不再因
	//     背景块位置漂移而断裂（对应 Go 回退路径 syncContextSnapshot）。
	snapObj := vm.NewObject()
	snapObj.Set("parts", func(call goja.FunctionCall) goja.Value {
		summaries := make([]string, 0, len(l.CompressedSummaries))
		summaries = append(summaries, l.CompressedSummaries...)
		knowledge := ""
		if l.WorkspaceRoot != "" {
			knowledge = ProjectKnowledge(l.WorkspaceRoot, 2500)
		}
		return vm.ToValue(map[string]any{
			"stale":      l.staleMsg,
			"summaries":  summaries,
			"memory":     LongTermMemoryPrompt(),
			"knowledge":  knowledge,
			"autonomous": l.Autonomous,
		})
	})
	snapObj.Set("sync", func(call goja.FunctionCall) goja.Value {
		msgsArg := call.Argument(0)
		jmsgs, jerr := jsToMsgs(vm, msgsArg)
		if jerr != nil {
			panic(vm.NewGoError(jerr))
		}
		text := ""
		if v := call.Argument(1); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			text = v.String()
		}
		log.Printf("[snapshot-sync] 收到 JS 快照请求 len=%d textLen=%d msgs=%d", len(msgsArg.String()), len(text), len(jmsgs))
		if text == "" {
			return msgsArg // 无内容：不注入（历史已有旧快照保留，避免删消息破坏前缀）
		}
		full := backgroundCtxMarker + systemReminderFrame("会话上下文摘要与状态提示", text)
		if last, ok := findLastSnapshotContent(jmsgs); ok && last == full {
			return msgsArg // 内容未变：零注入，前缀稳定
		}
		jmsgs = append(jmsgs, Message{Role: RoleUser, Content: full})
		log.Printf("[snapshot-sync] 注入新快照 textLen=%d msgs=%d -> %d", len(text), len(jmsgs)-1, len(jmsgs))
		l.persist(jmsgs) // 还原完整时间线落盘（防压缩视图覆盖 store）
		return vm.ToValue(msgsToJS(vm, jmsgs))
	})
	ctxObj.Set("snapshot", snapObj)
	proxy.Set("context", ctxObj)

	// ── compact.estimate(msgs) / compact.apply(msgs, mode) ──
	// ★ 2026-08-27 压缩策略外置：阈值/冷却/硬地板判定在 JS（agentloop），
	//   Go 提供数据面（estimate：token 估算+配置）与执行面（apply：early/full）。
	compactObj := vm.NewObject()
	compactObj.Set("estimate", func(call goja.FunctionCall) goja.Value {
		jmsgs, jerr := jsToMsgs(vm, call.Argument(0))
		if jerr != nil {
			panic(vm.NewGoError(jerr))
		}
		tokens := estimateTokens(jmsgs)
		if l.lastPromptTokens > tokens {
			tokens = l.lastPromptTokens
		}
		max := l.MaxContextTokens
		ratio := 0.0
		if max > 0 {
			ratio = float64(tokens) / float64(max)
		}
		if max > compactHardFloor && tokens >= compactHardFloor {
			ratio = compactRatio
		}
		return vm.ToValue(map[string]any{
			"tokens":           tokens,
			"lastPromptTokens": l.lastPromptTokens,
			"maxContextTokens": max,
			"ratio":            ratio,
			"cooldown":         l.compactCooldown,
			"cooldownEarly":    compactCooldownEarly,
			"cooldownFull":     compactCooldownTurns,
			"thresholdEarly":   compactRatioEarly,
			"thresholdFull":    compactRatio,
			"hardFloor":        compactHardFloor,
			"minDrop":          compactMinDrop,
		})
	})
	compactObj.Set("apply", func(call goja.FunctionCall) goja.Value {
		jmsgs, jerr := jsToMsgs(vm, call.Argument(0))
		if jerr != nil {
			panic(vm.NewGoError(jerr))
		}
		mode := call.Argument(1).String()
		if mode == "early" {
			out := l.earlyCompact(jmsgs)
			return vm.ToValue(map[string]any{"msgs": msgsToJS(vm, out), "dropped": 0, "mode": "early"})
		}
		out, summary, dropped := l.compact(r.ctx, jmsgs)
		if dropped > 0 {
			const maxSummaries = 3
			l.CompressedSummaries = append(l.CompressedSummaries, summary)
			if len(l.CompressedSummaries) > maxSummaries {
				l.CompressedSummaries = l.CompressedSummaries[len(l.CompressedSummaries)-maxSummaries:]
			}
			l.compactCooldown = compactCooldownTurns
			l.lastPromptTokens = 0
		}
		return vm.ToValue(map[string]any{"msgs": msgsToJS(vm, out), "dropped": dropped, "mode": "full"})
	})
	proxy.Set("compact", compactObj)

	// ── circling.state() / circling.clear() + track / detect（detect 保留为回退）──
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
	// state()：绕圈数据面（最近调用签名+成败）——★ 判定策略已移 JS（阈值/窗口/提示文本可配）
	circlingObj.Set("state", func(call goja.FunctionCall) goja.Value {
		out := make([]map[string]any, 0, len(l.recentCalls))
		for _, c := range l.recentCalls {
			out = append(out, map[string]any{"sig": c.sig, "failed": c.failed})
		}
		return vm.ToValue(out)
	})
	circlingObj.Set("clear", func(call goja.FunctionCall) goja.Value {
		l.recentCalls = nil
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
			Provider:            l.getProvider(),
			Registry:            l.Registry,
			WorkspaceRoot:       l.WorkspaceRoot,
			Autonomous:          false,
			ReviewMode:          l.getReviewMode(),
			ReviewBlacklist:     l.ReviewBlacklist,
			ReviewWhitelist:     l.ReviewWhitelist,
			ReviewProvider:      l.getReviewProvider(),
			MaxContextTokens:    l.MaxContextTokens,
			Compressor:          l.Compressor,
			MaxIterations:       10,
			System:              l.System,
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
		// 子 Loop 深度 +1（嵌套限制）；★ 2026-08-30 实例池：传父实例引用——
		// 子 runWithJS 优先租借独立实例（正常加锁），池满时复用父实例不重复加锁。
		subCtx := context.WithValue(context.WithValue(r.ctx, jsLoopDepthKey{}, depth+1), jsLoopParentImplKey{}, r.impl)
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
	// ★ 2026-08-30：统一走 runPreStep（host 钩子 + 外部桥瀑布 agent/pre-step）——
	//   Node 插件订阅者（dsh-agent-teams 激活指令注入）在 JS 循环同样生效。
	ctrlObj.Set("preStep", func(call goja.FunctionCall) goja.Value {
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
		rewritten, reject, perr := l.runPreStep(r.ctx, jmsgs, turn, step)
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
