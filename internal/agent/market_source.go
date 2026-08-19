// market_source.go — 市场源注册表（市场插件化：能力与挂载分离）
//
// 2026-08-18：skill/mcp/plugin 三个市场全部插件化——市场由磁盘插件
// （.pair/plugins/market-skill|market-mcp|market-plugin）声明挂载：
//   - 能力层（本文件 + market_registry.go）：搜索/安装实现留在 Go 内核；
//   - 挂载层（磁盘插件 JS）：apply 时 ctx.market.register({kind, source, name})
//     声明该市场「启用」；插件停用/删除 → 对应市场从面板与搜索中消失。
//
// 搜索实现按 source 标识分派：
//   - "github"     → searchMarketGitHub（GitHub 仓库 → skill 条目）
//   - "npm"        → searchMarketNPM（npm registry → mcp 条目）
//   - "npm-paircode" → searchMarketNPMPlugins（npm PairCode 插件 → plugin 条目）
//
// 无 //go:build 标签，全平台可用。

package agent

import (
	"sort"
	"sync"
)

// MarketSourceMeta 市场源元数据（插件声明，前端动态 tab 用）。
type MarketSourceMeta struct {
	Kind   string `json:"kind"`   // "skill" / "mcp" / "plugin"（唯一）
	Name   string `json:"name"`   // 显示名（如 "技能市场"）
	Source string `json:"source"` // 搜索实现标识：github / npm / npm-cordis
	Desc   string `json:"desc"`   // 用途说明
}

var (
	marketSourceMu     sync.RWMutex
	marketSources      = map[string]MarketSourceMeta{} // kind -> meta（已注册的市场）
	marketSourceKinds  = []string{"skill", "mcp", "plugin"}
	marketSourceKnown  = map[string]bool{"skill": true, "mcp": true, "plugin": true}
)

// RegisterMarketSource 注册一个市场源（磁盘插件 apply 时调用）。
// kind 须为内核已知的搜索类别（skill/mcp/plugin），重复注册幂等覆盖。
func RegisterMarketSource(meta MarketSourceMeta) {
	if !marketSourceKnown[meta.Kind] {
		return
	}
	if meta.Name == "" {
		meta.Name = meta.Kind + " 市场"
	}
	marketSourceMu.Lock()
	marketSources[meta.Kind] = meta
	marketSourceMu.Unlock()
}

// UnregisterMarketSource 注销一个市场源（插件 stop/卸载时调用）。
func UnregisterMarketSource(kind string) {
	marketSourceMu.Lock()
	delete(marketSources, kind)
	marketSourceMu.Unlock()
}

// MarketSources 返回当前已注册的市场源（按 kind 固定顺序，前端 tab 用）。
func MarketSources() []MarketSourceMeta {
	marketSourceMu.RLock()
	defer marketSourceMu.RUnlock()
	out := make([]MarketSourceMeta, 0, len(marketSources))
	for _, k := range marketSourceKinds {
		if m, ok := marketSources[k]; ok {
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out
}

// marketEnabledKinds 返回已注册市场的 kind 集合。
func marketEnabledKinds() map[string]bool {
	marketSourceMu.RLock()
	defer marketSourceMu.RUnlock()
	out := make(map[string]bool, len(marketSources))
	for k := range marketSources {
		out[k] = true
	}
	return out
}

// marketEnabled 判断某市场 kind 是否已注册（插件未装载 → 该市场不可搜）。
func marketEnabled(kind string) bool {
	marketSourceMu.RLock()
	_, ok := marketSources[kind]
	marketSourceMu.RUnlock()
	return ok
}

// marketSourceOf 返回 kind 对应的搜索实现标识（未知 kind 返回空）。
func marketSourceOf(kind string) string {
	marketSourceMu.RLock()
	m, ok := marketSources[kind]
	marketSourceMu.RUnlock()
	if !ok {
		return ""
	}
	return m.Source
}
