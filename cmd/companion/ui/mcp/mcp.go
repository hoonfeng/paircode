// package mcp 是 MCP（Model Context Protocol）服务器管理的薄壳代理。
//
// 设计意图：
// 所有配置 CRUD 业务逻辑（JSON 读写、路径管理、排序）已迁入 agent/mcp_config.go。
// 本文件仅做两件事：
//  1. 初始化函数 InitMCP —— 注入用户级和项目级配置路径到 agent 全局变量
//  2. 类型别名 + 函数转发 —— 保持旧 API 兼容（外部调用者无需改代码）
//
// 维护注意事项：
// - 不要在此文件中添加任何业务逻辑，全部委托给 agent.MCPXXX()
// - 类型别名 `=` 不是类型定义，外部代码使用 `mcppanel.Level` 即 `agent.MCPLevel`
// - 如果 agent 包中的 MCP 函数签名变化，本文件的转发函数需同步更新
//
//go:build windows

package mcp

import (
	"github.com/hoonfeng/paircode/cmd/companion/agent"
)

// Level 配置层级（用户级 / 项目级）。
type Level = agent.MCPLevel

const (
	LevelUser    Level = agent.MCPLevelUser    // 用户级 MCP 配置
	LevelProject Level = agent.MCPLevelProject // 项目级（工作区级）MCP 配置
)

// LevelDef 层级描述。
type LevelDef = agent.MCPLevelDef

// Levels 所有层级（显示顺序）。
var Levels = []LevelDef{
	{LevelUser, "用户级"},
	{LevelProject, "工作区级"},
}

// MCPServerConfig MCP 服务器配置（bridge 用，兼容 agent.RegisterMCPServers）。
type MCPServerConfig = agent.MCPServerConfig

// Entry MCP 服务器条目。
type Entry = agent.MCPEntry

// InitMCP 注入用户级和项目级 MCP 配置路径到 agent 全局变量。
// 由 web_server.go 在启动时调用，传参来自 core.ConfigDir() 和 core.Root()。
func InitMCP(userConfigPath, projectConfigPath string) {
	agent.MCPUserConfigPath = userConfigPath
	agent.MCPProjectConfigPath = projectConfigPath
}

// ReadLevel 读某层级的所有 MCP 服务器（按名排序）。
func ReadLevel(lv Level) []Entry {
	return agent.MCPReadLevel(lv)
}

// Upsert 新增/更新某层级的 MCP 服务器。
func Upsert(lv Level, e Entry) error {
	return agent.MCPUpsert(lv, e)
}

// Delete 删除某层级的 MCP 服务器。
func Delete(lv Level, name string) error {
	return agent.MCPDelete(lv, name)
}

// Enabled 检查某层级的 MCP 服务器是否启用（默认启用）。
func Enabled(lv Level, name string) bool {
	return agent.MCPEnabled(lv, name)
}

// SetEnabled 设置某层级 MCP 服务器的启用/禁用状态。
func SetEnabled(lv Level, name string, enabled bool) error {
	return agent.MCPSetEnabled(lv, name, enabled)
}

// LoadConfigs 从所有层级加载 MCP 服务器配置（bridge 用：连接外部 MCP）。
func LoadConfigs() []MCPServerConfig {
	return agent.MCPLoadConfigs()
}
