// plugin_activation.go — 插件「按需激活」机制（★ 2026-08-31）
//
// 背景：agent-teams 这类重型插件把一大段协议写进系统提示 + 挂一堆工具，
// 导致 agent 每次对话都能看到协议并「自我触发」，造成不必要的调用。
//
// 机制：插件声明 activation（on-demand + 触发命令）后——
//   - 该插件的系统提示段对 agent 隐藏（不注入系统提示）
//   - 该插件的工具对 agent 隐藏（不合并进会话注册表）
//   - 用户在对话里执行对应 /命令（如 /agent-teams）时，本会话激活该插件：
//     提示段 + 工具立即可见（从下一轮 LLM 请求起生效）
//
// 激活状态为会话级内存态（进程重启后需重新 /命令，可接受：协议文本随
// 命令注入，agent 当轮即知道该怎么做）。
package agent

import (
	"strings"
	"sync"
)

// onDemandPlugin 一个按需插件：命令 → 插件名（反向：插件名 → 命令）。
var (
	onDemandMu      sync.RWMutex
	onDemandByCmd   = map[string]string{} // 命令名 → 插件名（命令触发激活）
	onDemandPlugins = map[string]bool{}   // 插件名 → 是否按需（isOnDemand 查询）
	// 会话级激活状态：convID → 已激活的插件名集合
	convActivated = map[string]map[string]bool{}
)

// DeclareOnDemandPlugin 声明插件为「按需激活」（插件 apply 时调用；重复声明幂等）。
// command 为触发命令名（不含 '/'；空则仅隐藏不可自动激活）。
func DeclareOnDemandPlugin(plugin, command string) {
	onDemandMu.Lock()
	defer onDemandMu.Unlock()
	onDemandPlugins[plugin] = true
	if command != "" {
		onDemandByCmd[command] = plugin
	}
}

// IsOnDemandPlugin 插件是否为按需激活类（未声明 → false = 常驻注入）。
func IsOnDemandPlugin(plugin string) bool {
	onDemandMu.RLock()
	defer onDemandMu.RUnlock()
	return onDemandPlugins[plugin]
}

// IsPluginActiveInConv 插件在指定会话是否对 agent 可见：
// 常驻插件恒 true；按需插件须已激活（convID 空 → false，未开会话不算）。
func IsPluginActiveInConv(convID, plugin string) bool {
	if convID == "" {
		return !IsOnDemandPlugin(plugin)
	}
	if !IsOnDemandPlugin(plugin) {
		return true
	}
	onDemandMu.RLock()
	defer onDemandMu.RUnlock()
	if convIDs, ok := convActivated[convID]; ok {
		return convIDs[plugin]
	}
	return false
}

// ActivatePluginInConv 激活指定会话的按需插件（幂等）。返回是否真的激活。
func ActivatePluginInConv(convID, plugin string) bool {
	if convID == "" || plugin == "" || !IsOnDemandPlugin(plugin) {
		return false
	}
	onDemandMu.Lock()
	defer onDemandMu.Unlock()
	set, ok := convActivated[convID]
	if !ok {
		set = map[string]bool{}
		convActivated[convID] = set
	}
	if set[plugin] {
		return false
	}
	set[plugin] = true
	return true
}

// ActivateByCommand 命令执行时调用：命令触发某按需插件 → 激活该会话。
// 返回被激活的插件名（未匹配返回 ""）。
func ActivateByCommand(convID, command string) string {
	if command == "" {
		return ""
	}
	onDemandMu.RLock()
	plugin := onDemandByCmd[command]
	onDemandMu.RUnlock()
	if plugin == "" || !ActivatePluginInConv(convID, plugin) {
		return ""
	}
	return plugin
}

// OnDemandCommandMapping 命令清单打标用：命令名 → 插件名（按需激活命令）。
func OnDemandCommandMapping() map[string]string {
	onDemandMu.RLock()
	defer onDemandMu.RUnlock()
	out := make(map[string]string, len(onDemandByCmd))
	for k, v := range onDemandByCmd {
		out[k] = v
	}
	return out
}

// ResetConvActivations 清空指定会话激活状态（会话删除时调用；convID 空=全清）。
func ResetConvActivations(convID string) {
	onDemandMu.Lock()
	defer onDemandMu.Unlock()
	if convID == "" {
		convActivated = map[string]map[string]bool{}
		return
	}
	delete(convActivated, convID)
}

// PluginActivationNotice 返回按需插件的协议说明文本（其全部系统提示段拼接），
// 供命令执行时追加到注入消息——agent 当轮即知协议，无需等待提示词重装。
func PluginActivationNotice(plugin string) string {
	if plugin == "" {
		return ""
	}
	host := GetGlobalPluginHost()
	if host == nil {
		return ""
	}
	var b strings.Builder
	for _, s := range host.Sections() {
		if s == nil || s.Plugin != plugin || strings.TrimSpace(s.Text) == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(s.Text)
	}
	return b.String()
}
