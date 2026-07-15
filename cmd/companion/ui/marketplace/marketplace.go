// Package marketplace 是 MCP/Skills 市场安装实现的 UI 层薄壳代理。
//
// 所有安装业务逻辑已迁入 agent 包（MarketInstallScoped / MarketInstallEntry / MarketIsInstalled）。
// 本文件仅保留 UI 层特有的功能：
//   - SearchText（格式化输出）
//   - InstallAndNotify（UI 通知）
//   其余函数全部委托给 agent。
//
//go:build windows

package marketplace

import (
	"fmt"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
)

// ─── 对外搜索 ───

// SearchText 返回市场搜索的纯文本结果。
func SearchText(query, kind string) string {
	results := Search(query, kind)
	if len(results) == 0 {
		return "未找到匹配的市场条目。"
	}
	var b strings.Builder
	for _, e := range results {
		fmt.Fprintf(&b, "[%s] %s（%s）：%s\n", e.Kind, e.Name, e.ID, e.Description)
	}
	return b.String()
}

// ─── 安装 ───

// InstallScoped 从市场按 id 安装一个 MCP 服务器或技能。
func InstallScoped(id string, auto bool, scope ...string) (string, error) {
	s := "user"
	if len(scope) > 0 && scope[0] != "" {
		s = scope[0]
	}
	return agent.MarketInstallScoped(id, auto, s)
}

// InstallEntry 直接从 RegistryEntry 安装 MCP 或技能（不查注册表）。
func InstallEntry(entry RegistryEntry, auto bool, scope ...string) (string, error) {
	s := "user"
	if len(scope) > 0 && scope[0] != "" {
		s = scope[0]
	}
	return agent.MarketInstallEntry(entry, auto, s)
}

// InstallAndNotify 安装并发送 UI 通知（供 UI 面板调用）。
func InstallAndNotify(id string) {
	msg, err := InstallScoped(id, false)
	if err != nil {
		// UI 通知：错误消息
		return
	}
	_ = msg
}

// ─── 已安装状态 ───

// IsInstalled 检查某条目是否已安装。
func IsInstalled(id string) bool {
	return agent.MarketIsInstalled(id)
}
