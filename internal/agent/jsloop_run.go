package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"wb-ui/goja"
)

// ── agentloop 核心外置：Loop.Run 委托 JS 循环（runWithJS）──
//
// 职责划分：
//   - Go（本文件）：Run 前置准备（turn 打开/loop 服务/消息组装/缓存诊断/工具定义
//     精简/上下文压缩）+ 能力代理构建（llm/tools/events/persist/approve/context/
//     circling/store/ctrl）+ 收尾（History 更新/兜底持久化/endTurn/loop 服务撤销）
//   - JS（agentloop 插件）：循环策略——iter 循环结构、流式事件分发、审核决策调用、
//     自然终止/内容循环/绕圈检测、跟进队列处理
//
// 可回退：CurrentJSLoop() 为空 → Loop.Run 走现有 Go 循环，本文件不参与。

// ── Go ↔ JS 消息转换 ─────────────────────────────────────────

// msgsToJS 把 []Message 转为 JS 数组（role/content/toolCalls/toolCallId/name/reasoning）。
func msgsToJS(vm *goja.Runtime, msgs []Message) []any {
	out := make([]any, 0, len(msgs))
	for _, m := range msgs {
		obj := map[string]any{
			"role":    string(m.Role),
			"content": m.Content,
		}
		if m.ToolCallID != "" {
			obj["toolCallId"] = m.ToolCallID
		}
		if m.Name != "" {
			obj["name"] = m.Name
		}
		if m.Reasoning != "" {
			obj["reasoning"] = m.Reasoning
		}
		if len(m.ToolCalls) > 0 {
			tcs := make([]any, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				tcs = append(tcs, map[string]any{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			obj["toolCalls"] = tcs
		}
		out = append(out, obj)
	}
	return out
}

// jsToMsgs 把 JS 数组/对象转为 []Message。
func jsToMsgs(vm *goja.Runtime, v goja.Value) ([]Message, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, nil
	}
	exported := v.Export()
	arr, ok := exported.([]any)
	if !ok {
		return nil, fmt.Errorf("jsToMsgs: 期望数组，得到 %T", exported)
	}
	out := make([]Message, 0, len(arr))
	for _, it := range arr {
		m, err := jsObjToMsg(it)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// jsObjToMsg 把单个 JS 消息对象转为 Message。
func jsObjToMsg(v any) (Message, error) {
	obj, ok := v.(map[string]any)
	if !ok {
		return Message{}, fmt.Errorf("jsObjToMsg: 期望对象，得到 %T", v)
	}
	m := Message{}
	if s, ok := obj["role"].(string); ok {
		m.Role = Role(s)
	}
	if s, ok := obj["content"].(string); ok {
		m.Content = s
	}
	if s, ok := obj["toolCallId"].(string); ok {
		m.ToolCallID = s
	}
	if s, ok := obj["name"].(string); ok {
		m.Name = s
	}
	if s, ok := obj["reasoning"].(string); ok {
		m.Reasoning = s
	}
	if tcs, ok := obj["toolCalls"].([]any); ok {
		for _, it := range tcs {
			if tcm, ok := it.(map[string]any); ok {
				var tc ToolCall
				tc.ID, _ = tcm["id"].(string)
				tc.Type, _ = tcm["type"].(string)
				if fn, ok := tcm["function"].(map[string]any); ok {
					tc.Function.Name, _ = fn["name"].(string)
					tc.Function.Arguments, _ = fn["arguments"].(string)
				}
				m.ToolCalls = append(m.ToolCalls, tc)
			}
		}
	}
	return m, nil
}

// jsToTC 把 JS 工具调用对象转为 ToolCall。
func jsToTC(v any) (ToolCall, error) {
	obj, ok := v.(map[string]any)
	if !ok {
		return ToolCall{}, fmt.Errorf("jsToTC: 期望对象，得到 %T", v)
	}
	var tc ToolCall
	tc.ID, _ = obj["id"].(string)
	tc.Type, _ = obj["type"].(string)
	if fn, ok := obj["function"].(map[string]any); ok {
		tc.Function.Name, _ = fn["name"].(string)
		tc.Function.Arguments, _ = fn["arguments"].(string)
	}
	return tc, nil
}

// jsLoopDepthKey 嵌套 delegate 深度标记（防无限递归）。
type jsLoopDepthKey struct{}

// jsLoopDepth 读取当前 delegate 嵌套深度（0=顶层）。
func jsLoopDepth(ctx context.Context) int {
	if v := ctx.Value(jsLoopDepthKey{}); v != nil {
		if n, ok := v.(int); ok {
			return n
		}
	}
	return 0
}

// jsLoopInLockKey 标记「父 JS 执行锁已持有」（delegate 子 Loop 用）。
// 子 Loop.runWithJS 检测到该标志时不重复 vm.Lock（非重入，二次加锁死锁）。
type jsLoopInLockKey struct{}

// ── runWithJS：Loop.Run 的 JS 委托实现 ───────────────────────

// runWithJS 由 Loop.Run 在 CurrentJSLoop() 非空时调用：Go 做前置准备与收尾，
// 循环业务委托 JS 实现。
func (l *Loop) runWithJS(ctx context.Context, task string, history []Message, impl *jsLoopImpl) (msgs []Message, err error) {
	log.Printf("[loop-js] Run 开始（JS 循环实现 %q）taskLen=%d history=%d maxIter=%d autonomous=%v",
		impl.id, len(task), len(history), l.MaxIterations, l.Autonomous)

	// ★ 2026-08-21：Provider 判空兜底——清空配置后 buildWebProvider 返回 nil，
	//   JS 循环首次 loop.llm.chat → l.Provider.Chat nil pointer panic。
	//   此处提前拦截返回明确错误（handleChatSend 已前置提示，此为其他入口兜底）。
	if l.Provider == nil {
		return msgs, fmt.Errorf("Loop.Provider 为空：未配置 AI 服务商（APIKey/BaseURL 缺失），无法启动循环")
	}

	// ── 前置准备（与 Go Run 一致）──
	if history == nil {
		history = l.History
	}
	l.openTurn()
	l.cacheDiagOn = os.Getenv("WB_CACHE_DIAG") == "1"

	if ph := GetGlobalPluginHost(); ph != nil && ph.Context() != nil {
		l.loopSvc = newLoopService(l)
		cancelProvide := ph.Context().Provide("loop", l.loopSvc)
		defer cancelProvide()
	}

	defer func() {
		l.History = msgs
		if l.OnBatchPersist != nil && msgs != nil {
			l.OnBatchPersist(msgs)
		}
		l.endTurn(err, ctx.Err() != nil)
		l.loopSvc = nil
	}()

	hist := CopyHistory(history)
	max := l.MaxIterations
	if max <= 0 {
		max = 30
	}
	msgs = make([]Message, 0, len(hist)+4)
	if l.System != "" && !hasSystem(hist) {
		msgs = append(msgs, Message{Role: RoleSystem, Content: l.System})
	}
	msgs = append(msgs, hist...)
	if len(msgs) > 0 && msgs[len(msgs)-1].Role == RoleUser && msgs[len(msgs)-1].Content == task {
		// 末尾已有同内容用户消息，跳过
	} else {
		msgs = append(msgs, Message{Role: RoleUser, Content: task})
	}
	l.staleMsg = AutoVerifyStale()
	if l.Autonomous && l.autonomousStartTime.IsZero() {
		l.autonomousStartTime = time.Now()
	}
	tools := ApplyConciseToolDescriptions(l.Registry.Definitions())
	if l.Registry.OnToolUpdate == nil {
		l.Registry.OnToolUpdate = func(name, callID, partial string) {
			l.emit(Event{Type: EventToolUpdate, Tool: name, ToolCallID: callID, PartialResult: partial})
		}
	}
	msgs = l.maybeCompact(ctx, msgs)

	// ── 构建能力代理并委托 JS ──
	runner := &jsLoopRunner{loop: l, ctx: ctx, impl: impl}
	var (
		result  goja.Value
		callErr error
	)
	// ★ 锁语义：顶层 Run 经 withLock 加 VM 执行锁（独占 JS）。
	//   delegate 子 Loop（ctx 带 jsLoopInLockKey=true，父 JS 调用栈内同步执行）
	//   不再重复加锁——vm.lock 非重入，二次 Lock 同 goroutine 死锁。
	runJS := func() error {
		return runJSWithTimeout(impl.vm, 0, func() error {
			v, e := impl.run(goja.Undefined(), impl.vm.ToValue(runner.buildArgs(task, msgs, tools, max)))
			if e != nil {
				return e
			}
			// async 函数返回 Promise → 同步等待（goja 微任务 drain）
			av, aerr := awaitJSValue(impl.vm, v)
			if aerr != nil {
				return aerr
			}
			result = av
			return nil
		})
	}
	if inParentLock, _ := ctx.Value(jsLoopInLockKey{}).(bool); inParentLock {
		// delegate 嵌套：父 runWithJS 已持锁（同一 goroutine），直接执行
		callErr = runJS()
	} else {
		impl.plugin.withLock(func() { callErr = runJS() })
	}
	if callErr != nil {
		log.Printf("[loop-js] JS 循环 %q 执行失败: %v", impl.id, callErr)
		l.emit(Event{Type: EventError, Content: "JS 循环执行失败: " + callErr.Error()})
		l.LastTurnReason = TurnError
		return msgs, fmt.Errorf("JS 循环 %q 执行失败: %w", impl.id, callErr)
	}

	// ── 解析 JS 返回：{ msgs: [...], error?: string } ──
	if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
		l.LastTurnReason = TurnError
		return msgs, errors.New("JS 循环返回空结果（缺少 {msgs, error?}）")
	}
	retObj := result.ToObject(impl.vm)
	if v := retObj.Get("msgs"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		jmsgs, jerr := jsToMsgs(impl.vm, v)
		if jerr != nil {
			l.LastTurnReason = TurnError
			return msgs, fmt.Errorf("JS 循环返回 msgs 解析失败: %w", jerr)
		}
		msgs = jmsgs
	}
	// JS 返回 error 字符串 → 转 Go 错误
	if ev := retObj.Get("error"); ev != nil && !goja.IsUndefined(ev) && !goja.IsNull(ev) && ev.String() != "" {
		msg := ev.String()
		if strings.Contains(msg, "最大迭代") || strings.Contains(msg, "max_iterations") || strings.Contains(msg, ErrMaxIterations.Error()) {
			l.LastTurnReason = TurnMaxIterations
			return msgs, ErrMaxIterations
		}
		l.LastTurnReason = TurnError
		return msgs, errors.New(msg)
	}
	log.Printf("[loop-js] Run 完成（JS 循环 %q）msgs=%d", impl.id, len(msgs))
	return msgs, nil
}
