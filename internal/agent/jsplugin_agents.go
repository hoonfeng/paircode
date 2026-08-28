// ═══════════════════════════════════════════════════════════
// jsplugin_agents.go — JS 插件的「子 Agent 编排」与「模型目录」能力面
//
// 提供两个 ctx 服务（inject 声明后可用）：
//
//   ctx.agents —— 成员会话（可续聊子 Agent）编排：
//     start({task, convId?, parentConvId?, label?, team?, member?,
//            system?, model?, wsRoot?, denyTools?}) → {convId, state, ...}
//     followup(convId, text)      → {queued:bool}（忙则排队，轮次结束自动续发）
//     stop(convId)                → true
//     running(convId)             → bool
//     status(convId)              → 记录对象（未登记返回 null）
//     list({parentConvId?, team?}) → 记录数组
//     lastText(convId)            → 该会话最近助手正文（汇总用）
//     ready()                     → 宿主会话启动器是否就绪
//
//   ctx.llm —— 模型路由信息（成员模型覆盖用）：
//     models()  → [{provider, model, label, isDefault}]
//     current() → {provider, model}
//
// 真正的会话启动能力由 web 层注入（SetSubAgentSpawner，复用 /api/chat/send
// 的 LoopOpts 构造链）；本文件只做 JS ↔ Go 的参数编解码与错误提示。
// ═══════════════════════════════════════════════════════════

package agent

import (
	"fmt"

	"wb-ui/goja"
)

// buildAgentsService 构造 ctx.agents（成员会话编排）。
func (p *jsPluginAdapter) buildAgentsService(pc *PluginContext) goja.Value {
	vm := p.vm
	a := vm.NewObject()

	// 记录 → JS 对象（按 json tag 显式给 key，避免 goja 用 Go 字段名）
	toJS := func(rec *SubAgentRecord) goja.Value {
		if rec == nil {
			return goja.Null()
		}
		return vm.ToValue(map[string]any{
			"convId":       rec.ConvID,
			"parentConvId": rec.ParentConv,
			"label":        rec.Label,
			"team":         rec.Team,
			"member":       rec.Member,
			"model":        rec.Model,
			"provider":     rec.Provider,
			"wsRoot":       rec.WsRoot,
			"denyTools":    rec.DenyTools,
			"createdAt":    rec.CreatedAt,
			"lastActiveAt": rec.LastActive,
			"turns":        rec.Turns,
			"state":        rec.State,
			"lastError":    rec.LastError,
			"pending":      rec.Pending,
		})
	}

	a.Set("ready", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(SubAgentSpawnerReady())
	})

	a.Set("start", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if goja.IsUndefined(arg) || goja.IsNull(arg) {
			panic(vm.NewTypeError("ctx.agents.start: 需要一个对象 {task, label?, system?, model?, ...}"))
		}
		obj, ok := arg.Export().(map[string]any)
		if !ok {
			panic(vm.NewTypeError("ctx.agents.start: 参数必须是对象"))
		}
		spec := SubAgentSpec{
			ConvID:     mapStr(obj, "convId"),
			ParentConv: mapStr(obj, "parentConvId"),
			Label:      mapStr(obj, "label"),
			Team:       mapStr(obj, "team"),
			Member:     mapStr(obj, "member"),
			Task:       mapStr(obj, "task"),
			System:     mapStr(obj, "system"),
			Model:      mapStr(obj, "model"),
			Provider:   mapStr(obj, "provider"),
			WsRoot:     mapStr(obj, "wsRoot"),
			DenyTools:  mapStrSlice(obj, "denyTools"),
			MaxIter:    mapInt(obj, "maxIterations"),
		}
		if spec.WsRoot == "" {
			spec.WsRoot = p.effectiveWsRoot(pc)
		}
		rec, err := SpawnSubAgent(spec)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("ctx.agents.start 失败: %v", err)))
		}
		return toJS(rec)
	})

	a.Set("followup", func(call goja.FunctionCall) goja.Value {
		convID := call.Argument(0).String()
		text := call.Argument(1).String()
		queued, err := FollowupSubAgent(convID, text)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("ctx.agents.followup 失败: %v", err)))
		}
		return vm.ToValue(map[string]any{"ok": true, "queued": queued, "convId": convID})
	})

	a.Set("stop", func(call goja.FunctionCall) goja.Value {
		convID := call.Argument(0).String()
		if err := StopSubAgent(convID); err != nil {
			panic(vm.NewGoError(fmt.Errorf("ctx.agents.stop 失败: %v", err)))
		}
		return vm.ToValue(true)
	})

	a.Set("running", func(call goja.FunctionCall) goja.Value {
		convID := call.Argument(0).String()
		rec := SubAgentInfo(convID)
		if rec == nil {
			// 未登记会话（如队长自己）：直接问会话管理器
			if mgr := GlobalSessionManager(); mgr != nil {
				return vm.ToValue(mgr.IsRunning(convID))
			}
			return vm.ToValue(false)
		}
		return vm.ToValue(rec.State == "running")
	})

	a.Set("status", func(call goja.FunctionCall) goja.Value {
		return toJS(SubAgentInfo(call.Argument(0).String()))
	})

	a.Set("list", func(call goja.FunctionCall) goja.Value {
		parent, team := "", ""
		if arg := call.Argument(0); !goja.IsUndefined(arg) && !goja.IsNull(arg) {
			if obj, ok := arg.Export().(map[string]any); ok {
				parent = mapStr(obj, "parentConvId")
				team = mapStr(obj, "team")
			}
		}
		recs := ListSubAgents(parent, team)
		out := make([]any, 0, len(recs))
		for _, rec := range recs {
			out = append(out, map[string]any{
				"convId":       rec.ConvID,
				"parentConvId": rec.ParentConv,
				"label":        rec.Label,
				"team":         rec.Team,
				"member":       rec.Member,
				"model":        rec.Model,
				"provider":     rec.Provider,
				"turns":        rec.Turns,
				"state":        rec.State,
				"lastError":    rec.LastError,
				"pending":      rec.Pending,
				"lastActiveAt": rec.LastActive,
			})
		}
		return vm.ToValue(out)
	})

	a.Set("lastText", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(SubAgentLastText(call.Argument(0).String()))
	})

	return vm.ToValue(a)
}

// buildLLMService 构造 ctx.llm（模型目录 / 当前模型）。
func (p *jsPluginAdapter) buildLLMService(pc *PluginContext) goja.Value {
	vm := p.vm
	l := vm.NewObject()

	l.Set("models", func(call goja.FunctionCall) goja.Value {
		models := SubAgentModels()
		if models == nil {
			return vm.ToValue([]any{})
		}
		out := make([]any, 0, len(models))
		for _, m := range models {
			out = append(out, m)
		}
		return vm.ToValue(out)
	})

	l.Set("current", func(call goja.FunctionCall) goja.Value {
		cur := SubAgentCurrentModel()
		if cur == nil {
			return vm.ToValue(map[string]any{})
		}
		return vm.ToValue(cur)
	})

	return vm.ToValue(l)
}

// ─── 参数解码小工具 ─────────────────────────────────────────

func mapStr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func mapInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func mapStrSlice(m map[string]any, key string) []string {
	raw, ok := m[key]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, it := range v {
			if s, ok := it.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// effectiveWsRoot 解析插件当前生效的工作区根（工具调用会话根优先）。
func (p *jsPluginAdapter) effectiveWsRoot(pc *PluginContext) string {
	if r := p.toolCallRoot(); r != "" {
		return r
	}
	if r := p.uiWsRootValue(); r != "" {
		return r
	}
	if pc != nil {
		return pc.WorkspaceRoot
	}
	return ""
}
