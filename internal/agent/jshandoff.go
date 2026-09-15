package agent

// jshandoff.go — 会话交接（handoff）插件化：JS 实现注册位 + 能力桥接。
//
// 背景（2026-09-12，用户决策）：handoff 的「判断 / 生成 / 复用刷新 / 锚点定位 /
// 视图组装」属**策略层**，整体外置到 agentloop 插件实现；Go 侧 handoff.go 保留
// 默认实现作为**回退**（插件未注册 / 已停用 / 执行失败 → 走 Go 默认，零行为变化）。
//
// 与 registerLoop / register(apply) 同模式：单槽位注册、后注册覆盖、卸载自动还原。
//
// ★ 执行位置仍在宿主（物理约束，非设计选择）：
//   - 用户输入边界（新提交 / 点继续）：web_server 构建 LoopOpts 时——此刻会话与
//     插件尚未装配，插件无从自主介入；
//   - 段续跑边界：SessionManager 的段循环（Run 之外），插件循环体覆盖不到。
//
// 宿主在以上两处调用注册实现；返回 view 即启用（宿主负责替换喂 LLM 的历史视图，
// **落盘不动**）。未注册 → Go 默认实现。
//
// 能力面（注入 JS 的 args）：
//
//	args = {
//	  kind: 'userTurn' | 'segment',
//	  convID, workspaceRoot, task, maxContextTokens,
//	  history: [ {role, content, toolCalls?, toolCallId?, name?} ],   // 完整历史（JSON 往返）
//	  provider: { chat(msgs) → {role, content} } | null,               // 整理用（主/压缩模型）
//	  judge:    { chat(msgs) → {role, content} } | null,               // 判官（仅用户输入边界）
//	  store:    { loadAll() → msgs, loadRecord() → rec|null, saveRecord(rec) },
//	  handoff:  { marker, title, enabled, thresholds(maxCtx), estimateTokens(msgs),
//	              ruleSummary(msgs), keepForRelevance(rel), parseRelevance(s),
//	              fingerprint(msg), stripSystem(msgs), isHandoffText(s) }
//	}
//
//	返回 = { view: [...], applied?: bool, notice?: string }

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/goja"
)

// ── 注册表（单槽位）──

// jsHandoffImpl 一个已注册的 JS 会话交接实现。
type jsHandoffImpl struct {
	id         string
	onUserTurn goja.Callable // async (args) → {view, applied?, notice?}
	onSegment  goja.Callable // async (args) → {view, applied?, notice?}
	plugin     *jsPluginAdapter
	vm         *goja.Runtime
}

var (
	jsHandoffMu  sync.RWMutex
	jsHandoffVal *jsHandoffImpl
)

// RegisterJSHandoff 注册 JS 会话交接实现（后注册覆盖先注册），返回还原函数。
func RegisterJSHandoff(impl *jsHandoffImpl) (restore func()) {
	jsHandoffMu.Lock()
	prev := jsHandoffVal
	jsHandoffVal = impl
	jsHandoffMu.Unlock()
	return func() {
		jsHandoffMu.Lock()
		if jsHandoffVal == impl {
			jsHandoffVal = prev
		}
		jsHandoffMu.Unlock()
	}
}

// UnregisterJSHandoff 注销指定实现（插件卸载时调用）。
func UnregisterJSHandoff(impl *jsHandoffImpl) {
	jsHandoffMu.Lock()
	if jsHandoffVal == impl {
		jsHandoffVal = nil
	}
	jsHandoffMu.Unlock()
}

// CurrentJSHandoff 返回当前生效的 JS 交接实现（nil = 未注册 → 走 Go 默认实现）。
// 环境变量 PAIR_HANDOFF_JS=0/off/false/no 时强制走 Go 默认实现
// （对照验证 / 排障开关：同一场景下两种实现应产生同样的整理与断裂模式）。
func CurrentJSHandoff() *jsHandoffImpl {
	if !handoffJSEnabled() {
		return nil
	}
	jsHandoffMu.RLock()
	defer jsHandoffMu.RUnlock()
	return jsHandoffVal
}

// handoffJSEnabled 是否允许委托 JS 交接实现（默认允许）。
func handoffJSEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PAIR_HANDOFF_JS"))) {
	case "0", "off", "false", "no", "disable", "disabled":
		return false
	}
	return true
}

// ── 注册入口：ctx.loopFactory.registerHandoff({id?, onUserTurn?, onSegment?}) ──

// attachHandoffRegister 在 ctx.loopFactory 对象上挂 registerHandoff 方法
// （jsplugin.go 的 buildContextObject 里 loopFactoryObj 创建后调用）。
func (p *jsPluginAdapter) attachHandoffRegister(loopFactoryObj *goja.Object) {
	vm := p.vm
	loopFactoryObj.Set("registerHandoff", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
			panic(vm.NewTypeError("ctx.loopFactory.registerHandoff: 需要 {id?, onUserTurn?, onSegment?}"))
		}
		obj := arg.ToObject(vm)
		id := obj.Get("id").String()
		if id == "" {
			id = p.def.name
		}
		// ★ 影子实例（并行会话 VM 副本，jsloop_pool）：**不写全局槽位**。
		//   交接是会话级策略、单槽位注册——影子注册会覆盖主实例实现，且影子 VM
		//   随会话结束销毁 → 槽位指向已销毁 Runtime（调用即崩）。
		//   与 registerLoop 的 shadow 分支同语义（影子只捕获循环实现）。
		if p.shadow {
			return vm.ToValue(map[string]any{"id": id, "ok": true, "shadow": true})
		}
		impl := &jsHandoffImpl{id: id, plugin: p, vm: vm}
		if fn, ok := goja.AssertFunction(obj.Get("onUserTurn")); ok {
			impl.onUserTurn = fn
		}
		if fn, ok := goja.AssertFunction(obj.Get("onSegment")); ok {
			impl.onSegment = fn
		}
		if impl.onUserTurn == nil && impl.onSegment == nil {
			panic(vm.NewTypeError("ctx.loopFactory.registerHandoff: 至少需要 onUserTurn 或 onSegment 之一"))
		}
		restore := RegisterJSHandoff(impl)
		p.addCleanup(restore)
		p.def.addDiag(fmt.Sprintf(
			"注册 JS 会话交接实现 %q（userTurn=%v segment=%v；卸载自动还原 Go 默认实现）",
			id, impl.onUserTurn != nil, impl.onSegment != nil))
		log.Printf("[js-plugin:%s] registerHandoff: 已注册 JS 会话交接实现 %q（会话交接策略委托 JS）", p.def.id, id)
		return vm.ToValue(map[string]any{"id": id, "ok": true})
	})
}

// ── 调用：宿主两个边界 ──

// UserTurnView 用户输入边界（新提交对话 / 点继续执行）调用 JS 交接实现。
// 返回 (view, applied, notice)；applied=false 时调用方保持原逻辑
// （原样历史或按 token 压力精简）。err 非 nil 时调用方应回退 Go 默认实现。
func (h *jsHandoffImpl) UserTurnView(convID, workspaceRoot string, history []Message, task string,
	maxContextTokens int, prov, judge Provider, store ConversationStore) ([]Message, bool, string, error) {
	if h == nil || h.onUserTurn == nil {
		return nil, false, "", fmt.Errorf("未注册 onUserTurn")
	}
	return h.invoke("userTurn", func(vm *goja.Runtime) *goja.Object {
		args := handoffArgsObject(vm, map[string]any{
			"kind":            "userTurn",
			"convID":          convID,
			"workspaceRoot":   workspaceRoot,
			"task":            task,
			"maxContextTokens": maxContextTokens,
			"history":         history,
		})
		args.Set("provider", providerHandle(vm, prov, handoffTimeoutDefault))
		args.Set("judge", providerHandle(vm, judge, handoffJudgeTimeout))
		args.Set("store", handoffStoreHandle(vm, store, convID))
		args.Set("handoff", handoffUtilsObject(vm))
		return args
	}, h.onUserTurn)
}

// SegmentView 段续跑边界调用 JS 交接实现（同一策略，judge=nil：同一任务的延续）。
func (h *jsHandoffImpl) SegmentView(convID, workspaceRoot string, history []Message, task string,
	maxContextTokens int, prov Provider, store ConversationStore) ([]Message, bool, string, error) {
	if h == nil || h.onSegment == nil {
		return nil, false, "", fmt.Errorf("未注册 onSegment")
	}
	return h.invoke("segment", func(vm *goja.Runtime) *goja.Object {
		args := handoffArgsObject(vm, map[string]any{
			"kind":             "segment",
			"convID":           convID,
			"workspaceRoot":    workspaceRoot,
			"task":             task,
			"maxContextTokens": maxContextTokens,
			"history":          history,
		})
		args.Set("provider", providerHandle(vm, prov, handoffTimeoutDefault))
		args.Set("judge", goja.Null())
		args.Set("store", handoffStoreHandle(vm, store, convID))
		args.Set("handoff", handoffUtilsObject(vm))
		return args
	}, h.onSegment)
}

// invoke 统一调用路径：持 VM 锁构造参数 → 调用 async 函数 → 同步等待 Promise →
// 解析返回值（全程持锁：goja 非并发安全）。
func (h *jsHandoffImpl) invoke(kind string, buildArgs func(vm *goja.Runtime) *goja.Object, fn goja.Callable) ([]Message, bool, string, error) {
	var (
		ret     goja.Value
		view    []Message
		applied bool
		notice  string
		err     error
	)
	run := func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("JS 交接实现 panic: %v", r)
			}
		}()
		args := buildArgs(h.vm)
		v, cerr := fn(goja.Undefined(), args)
		if cerr != nil {
			err = cerr
			return
		}
		av, aerr := awaitJSValue(h.vm, v) // async 函数 → 同步等待（微任务 drain）
		if aerr != nil {
			err = aerr
			return
		}
		ret = av
		view, applied, notice = parseJSHandoffResult(h.vm, ret)
	}
	// goja 非并发安全：JS 调用必须持 VM 锁（会话创建/续跑来自任意 goroutine）。
	// plugin 为 nil（单测直接构造 impl）时无锁可持，直接执行。
	if h.plugin != nil {
		h.plugin.withLock(run)
	} else {
		run()
	}
	if err != nil {
		return nil, false, "", fmt.Errorf("JS 交接实现 %s(%s) 失败: %w", h.id, kind, err)
	}
	return view, applied, notice, nil
}

// ── 参数构造 / 返回值解析 ──

// handoffArgsObject 把 Go 数据经 JSON 往返构造为 JS 对象
// （Message 含 tool_calls/reasoning/images 等嵌套结构，手写字段映射易漂移）。
func handoffArgsObject(vm *goja.Runtime, raw map[string]any) *goja.Object {
	obj, err := jsonToJSObject(vm, raw)
	if err != nil {
		log.Printf("[handoff-js] 参数序列化失败: %v", err)
		return vm.NewObject()
	}
	return obj
}

// jsonToJSObject Go 值 → JSON 字符串 → JS 对象（JSON.parse）。
func jsonToJSObject(vm *goja.Runtime, v any) (*goja.Object, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	vj, err := vm.RunString("(" + string(b) + ")")
	if err != nil {
		return nil, err
	}
	return vj.ToObject(vm), nil
}

// jsValueToMessages JS 值 → []Message（JSON.stringify 往返；解析失败返回错误）。
func jsValueToMessages(vm *goja.Runtime, v goja.Value) ([]Message, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, nil
	}
	s, err := jsonStringify(vm, v)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(s) == "" || s == "null" {
		return nil, nil
	}
	var msgs []Message
	if err := json.Unmarshal([]byte(s), &msgs); err != nil {
		return nil, fmt.Errorf("消息列表解析失败: %w", err)
	}
	return msgs, nil
}

// jsonStringify 调用 JS 侧 JSON.stringify（保证函数/undefined 等被规整）。
func jsonStringify(vm *goja.Runtime, v goja.Value) (string, error) {
	jsonObj := vm.Get("JSON").ToObject(vm)
	fn, ok := goja.AssertFunction(jsonObj.Get("stringify"))
	if !ok {
		return "", fmt.Errorf("JSON.stringify 不可用")
	}
	out, err := fn(goja.Undefined(), v)
	if err != nil {
		return "", err
	}
	if out == nil || goja.IsUndefined(out) || goja.IsNull(out) {
		return "", nil
	}
	return out.String(), nil
}

// parseJSHandoffResult 解析 JS 返回的 {view, applied?, notice?}。
func parseJSHandoffResult(vm *goja.Runtime, ret goja.Value) ([]Message, bool, string) {
	if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
		return nil, false, ""
	}
	obj := ret.ToObject(vm)
	notice := ""
	if nv := obj.Get("notice"); nv != nil && !goja.IsUndefined(nv) && !goja.IsNull(nv) {
		notice = nv.String()
	}
	if av := obj.Get("applied"); av != nil && !goja.IsUndefined(av) && !goja.IsNull(av) && !av.ToBoolean() {
		return nil, false, notice
	}
	msgs, err := jsValueToMessages(vm, obj.Get("view"))
	if err != nil {
		log.Printf("[handoff-js] 返回值 view 解析失败: %v", err)
		return nil, false, notice
	}
	if len(msgs) == 0 {
		return nil, false, notice
	}
	return msgs, true, notice
}

// ── 能力桥接 ──

// providerHandle 构造 JS 可见的 Provider 句柄：{ chat(msgs) → {role, content} }。
// 同步桥接（与 loop.llm.chat 同语义）：调用即阻塞至返回；失败返回 null 并由
// JS 侧自行决定回退（整理失败 → 规则式交接；判官失败 → 保持现状）。
func providerHandle(vm *goja.Runtime, p Provider, timeout time.Duration) goja.Value {
	if p == nil {
		return goja.Null()
	}
	obj := vm.NewObject()
	obj.Set("chat", func(call goja.FunctionCall) goja.Value {
		msgs, err := jsValueToMessages(vm, call.Argument(0))
		if err != nil {
			log.Printf("[handoff-js] provider.chat 参数解析失败: %v", err)
			return goja.Null()
		}
		ctx := context.Background()
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		out, err := p.Chat(ctx, msgs, nil, nil)
		if err != nil {
			log.Printf("[handoff-js] provider.chat 失败: %v", err)
			return goja.Null()
		}
		res, _ := jsonToJSObject(vm, map[string]any{"role": string(out.Role), "content": out.Content})
		if res == nil {
			return goja.Null()
		}
		return res
	})
	return obj
}

// handoffStoreHandle 构造 JS 可见的会话存储句柄：
//
//	{ loadAll() → msgs, loadRecord() → rec|null, saveRecord(rec) → bool }
//
// loadRecord/saveRecord 依赖 store 实现 handoffStore（未实现则返回 undefined，
// JS 侧据此退化为「不启用交接」，与 Go 默认实现同语义）。
func handoffStoreHandle(vm *goja.Runtime, store ConversationStore, convID string) goja.Value {
	if store == nil {
		return goja.Null()
	}
	obj := vm.NewObject()
	obj.Set("loadAll", func(call goja.FunctionCall) goja.Value {
		msgs, err := store.LoadAll(convID)
		if err != nil {
			log.Printf("[handoff-js] store.loadAll 失败 conv=%s: %v", convID, err)
			return goja.Null()
		}
		v, err := jsonToJSObject(vm, msgs)
		if err != nil {
			return goja.Null()
		}
		return v
	})
	hs, ok := store.(handoffStore)
	if !ok {
		return obj
	}
	obj.Set("loadRecord", func(call goja.FunctionCall) goja.Value {
		rec, err := hs.LoadHandoff(convID)
		if err != nil {
			log.Printf("[handoff-js] store.loadRecord 失败 conv=%s: %v", convID, err)
			return goja.Null()
		}
		if rec == nil {
			return goja.Null()
		}
		v, err := jsonToJSObject(vm, *rec)
		if err != nil {
			return goja.Null()
		}
		return v
	})
	obj.Set("saveRecord", func(call goja.FunctionCall) goja.Value {
		raw, err := jsonStringify(vm, call.Argument(0))
		if err != nil || strings.TrimSpace(raw) == "" || raw == "null" {
			return vm.ToValue(false)
		}
		var rec HandoffRecord
		if err := json.Unmarshal([]byte(raw), &rec); err != nil {
			log.Printf("[handoff-js] store.saveRecord 解析失败 conv=%s: %v", convID, err)
			return vm.ToValue(false)
		}
		if err := hs.SaveHandoff(convID, rec); err != nil {
			log.Printf("[handoff-js] store.saveRecord 失败 conv=%s: %v", convID, err)
			return vm.ToValue(false)
		}
		return vm.ToValue(true)
	})
	return obj
}

// handoffUtilsObject 构造 ctx.handoff / args.handoff 无状态工具对象。
// 纯函数与阈值口径**桥接自 Go**（单一真源）——JS 侧无需复刻，避免两侧漂移
// 导致锚点指纹/阈值判定不一致（缓存前缀稳定的前提）。
func handoffUtilsObject(vm *goja.Runtime) *goja.Object {
	obj := vm.NewObject()
	// ★ 2026-09-13：原 marker（「背景上下文·非当前任务」前缀）随该实现移除。
	//   交接文本以 title（handoffTitle）开头，JS 侧只拼 title（与 Go 逐字节一致）。
	//   marker 键保留但恒为空串：仅向后兼容尚未同步的插件副本
	//   （旧代码 `marker + title` 的结果与只拼 title 等价，不会重复标题）。
	obj.Set("marker", "")
	obj.Set("title", handoffTitle)
	obj.Set("enabled", HandoffEnabled())
	obj.Set("thresholds", func(call goja.FunctionCall) goja.Value {
		maxCtx := 0
		if a := call.Argument(0); a != nil && !goja.IsUndefined(a) && !goja.IsNull(a) {
			maxCtx = int(a.ToInteger())
		}
		v, _ := jsonToJSObject(vm, map[string]any{
			"triggerTokens": HandoffTriggerThreshold(maxCtx),
			"refreshTokens": handoffRefreshThreshold(),
			"recheckTokens": handoffRecheckThreshold(),
			"minMsgs":       handoffTriggerMinMsgs,
			"keepRecent":    handoffKeepRecentMsgs,
			"keepHigh":      handoffKeepRelHigh,
			"keepPartial":   handoffKeepRelPartial,
			"keepNone":      handoffKeepRelNone,
			"anchorSeq":     handoffAnchorSeq,
			"anchorSep":     handoffAnchorSep,
			"inputMaxMsgs":  handoffInputMaxMsgs,
			"inputMsgRunes": handoffInputMsgRunes,
			"prevTextRunes": handoffPrevTextRunes,
			"taskRunes":     handoffTaskRunes,
			"judgeTaskRunes": handoffJudgeTaskRunes,
			"judgePrevRunes": handoffJudgePrevRunes,
			"relHigh":        handoffRelHigh,
			"relPartial":     handoffRelPartial,
			"relNone":        handoffRelNone,
		})
		return v
	})
	obj.Set("estimateTokens", func(call goja.FunctionCall) goja.Value {
		msgs, err := jsValueToMessages(vm, call.Argument(0))
		if err != nil {
			return vm.ToValue(0)
		}
		return vm.ToValue(estimateTokens(msgs))
	})
	obj.Set("ruleSummary", func(call goja.FunctionCall) goja.Value {
		msgs, err := jsValueToMessages(vm, call.Argument(0))
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(ruleSummarize(msgs))
	})
	obj.Set("stripSystem", func(call goja.FunctionCall) goja.Value {
		msgs, err := jsValueToMessages(vm, call.Argument(0))
		if err != nil {
			return goja.Null()
		}
		v, err := jsonToJSObject(vm, stripSystemMsgs(msgs))
		if err != nil {
			return goja.Null()
		}
		return v
	})
	obj.Set("isHandoffText", func(call goja.FunctionCall) goja.Value {
		a := call.Argument(0)
		if a == nil || goja.IsUndefined(a) || goja.IsNull(a) {
			return vm.ToValue(false)
		}
		return vm.ToValue(isHandoffText(a.String()))
	})
	obj.Set("fingerprint", func(call goja.FunctionCall) goja.Value {
		raw, err := jsonStringify(vm, call.Argument(0))
		if err != nil || strings.TrimSpace(raw) == "" || raw == "null" {
			return vm.ToValue("")
		}
		var m Message
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(handoffFingerprint(m))
	})
	obj.Set("keepForRelevance", func(call goja.FunctionCall) goja.Value {
		a := call.Argument(0)
		rel := ""
		if a != nil && !goja.IsUndefined(a) && !goja.IsNull(a) {
			rel = a.String()
		}
		return vm.ToValue(keepForRelevance(rel))
	})
	obj.Set("parseRelevance", func(call goja.FunctionCall) goja.Value {
		a := call.Argument(0)
		if a == nil || goja.IsUndefined(a) || goja.IsNull(a) {
			return vm.ToValue("")
		}
		return vm.ToValue(parseRelevance(a.String()))
	})
	// 兜底：把 Go 规则式交接文本生成暴露给 JS（LLM 不可用时的等价回退路径）。
	obj.Set("ruleFallback", func(call goja.FunctionCall) goja.Value {
		msgs, err := jsValueToMessages(vm, call.Argument(0))
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(ruleHandoffFallback(msgs))
	})
	return obj
}
