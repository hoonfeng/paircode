// ═══════════════════════════════════════════════════════════
// jsplugin_agents.go — JS 插件的「会话唤醒投递」能力面
//
// 提供 ctx 服务（插件 inject 声明后可用）：
//
//   ctx.agents —— 会话唤醒投递（★ 2026-09：子 Agent 派生面已删除，
//                 只保留「向已存在的会话投递输入把它唤醒跑一轮」）：
//     followup(convId, text) → {ok, queued, convId}
//         空闲 → 立即起一轮；忙 → FIFO 排队，轮次结束自动续投。
//         消费方：agent-teams 插件（Web 面板「批准并运行」唤醒队长会话）。
//     ready() → 宿主唤醒器是否就绪
//
// 真正的「启动一轮」能力由 web 层注入（SetSessionWakeHook，复用
// launchConvRun 的 LoopOpts 构建链）；本文件只做 JS ↔ Go 的参数编解码。
//
// 已删除（2026-09）：ctx.agents.start/fork/report/stop/running/status/list/
// lastText 与 ctx.llm（成员模型覆盖用）——随子 Agent 实现一并移除。
// ═══════════════════════════════════════════════════════════

package agent

import (
	"fmt"

	"github.com/hoonfeng/paircode/goja"
)

// buildAgentsService 构造 ctx.agents（会话唤醒投递）。
func (p *jsPluginAdapter) buildAgentsService(pc *PluginContext) goja.Value {
	vm := p.vm
	a := vm.NewObject()

	a.Set("ready", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(SessionWakeReady())
	})

	// followup(convId, text)：向已存在的会话投递一条输入（空闲即起一轮，忙则排队）。
	a.Set("followup", func(call goja.FunctionCall) goja.Value {
		convID := call.Argument(0).String()
		text := call.Argument(1).String()
		queued, err := WakeSession(convID, text)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("ctx.agents.followup 失败: %v", err)))
		}
		return vm.ToValue(map[string]any{"ok": true, "queued": queued, "convId": convID})
	})

	return vm.ToValue(a)
}

// ─── 参数解码小工具（ctx 服务面共用：commands.go 等）───────────

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
