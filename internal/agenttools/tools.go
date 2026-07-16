// Agent 自管理工具 —— 薄壳代理。
// 所有管理工具注册和实现逻辑已迁入 agent/management_tools.go。
// 本文件仅做单行转发到 agent.RegisterManagementTools。
//
//go:build windows

package agenttools

import (
	"github.com/hoonfeng/paircode/internal/agent"
)

// RegisterManagementTools 注册 Agent 自管理工具。
// root 为工作区根路径（每个会话传自己的，实现多工作区隔离）。
// 内部全部委托给 agent.RegisterManagementTools。
func RegisterManagementTools(r *agent.Registry, root string) {
	agent.RegisterManagementTools(r, root)
}
