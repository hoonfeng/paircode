// Package marketplace 是 MCP/Skills 市场安装的桩实现。
// 完整实现待后续迁移；当前仅提供 agenttools 需要的 InstallScoped 函数。
//
//go:build windows

package marketplace

// InstallScoped 从市场按 id 安装一个 MCP 服务器或技能。
// 桩实现：返回未找到提示（市场注册表待实现）。
func InstallScoped(id string, auto bool) (string, error) {
	return "市场安装功能尚未实现（id=" + id + "）。请手动配置 MCP 服务器或技能。", nil
}
