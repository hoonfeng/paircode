// ═══════════════════════════════════════════════════════════════
// kernel_register.go — 内置 HTTP 接口 → 内核路由表（接口插件化装配）
//
// 背景（2026-08-16）：内置 /api/* 接口实现全部保留在 Go（web_server.go 的
// ws.handleXxx + internal/server/handler 共享 handler），但「挂载权」不再由
// mux.HandleFunc 硬编码——统一注册进内核路由表（internal/agent/kernel_api.go），
// 由 core-api 磁盘插件（.pair/plugins/core-api/）在 apply 时 ctx.kernel.install
// 批量挂到插件 ext 路由表（ExtRouteMiddleware 优先于 mux 拦截）。
//
// 语义：
//   - 接口实现 = 内核能力（Go），接口挂载 = 插件声明（core-api 清单）；
//   - 插件停用/删除 → 接口随之消失（接口随插件生命周期生灭）；
//   - 插件可局部覆盖：清单可增删条目（改 core-api/index.js 路由表）。
//
// 注意：/ws 与 /api/terminal/ws（WebSocket 传输端点）不属于 JSON API，
// 仍在 mux 注册（webui_webonly.go registerExtraHandlers）。
// ═══════════════════════════════════════════════════════════════

package main

import (
	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/server/handler"
)

// registerKernelAPIs 把全部内置接口注册进内核路由表（能力层）。
// 不挂载任何 mux 路由——挂载由 core-api 插件完成（ctx.kernel.install）。
func registerKernelAPIs(s *webServer) {
	// ── 系统 ──
	_ = agent.KernelAPIRegister("health", "GET", "/api/health", "健康检查（工作区根/文件夹）", s.handleHealth)
	_ = agent.KernelAPIRegister("system.info", "GET", "/api/system/info", "系统信息", s.handleSysInfo)
	_ = agent.KernelAPIRegister("system.exec", "POST", "/api/system/exec", "执行 shell 命令", s.handleExec)

	// ── 文件系统 ──
	_ = agent.KernelAPIRegister("fs.drives", "GET", "/api/fs/drives", "磁盘驱动器列表", s.handleFSDrives)
	_ = agent.KernelAPIRegister("fs.list", "GET", "/api/fs/list", "目录列表", s.handleFSList)
	_ = agent.KernelAPIRegister("fs.read", "GET", "/api/fs/read", "读文本文件", s.handleFSRead)
	_ = agent.KernelAPIRegister("fs.write", "POST", "/api/fs/write", "写文件", s.handleFSWrite)
	_ = agent.KernelAPIRegister("fs.rename", "POST", "/api/fs/rename", "重命名/移动", s.handleFSRename)
	_ = agent.KernelAPIRegister("fs.delete", "POST", "/api/fs/delete", "删除文件/目录", s.handleFSDelete)
	_ = agent.KernelAPIRegister("fs.mkdir", "POST", "/api/fs/mkdir", "创建目录", s.handleFSMkdir)
	_ = agent.KernelAPIRegister("fs.search", "GET", "/api/fs/search", "文件搜索", s.handleFSSearch)
	_ = agent.KernelAPIRegister("fs.image", "GET", "/api/fs/image", "图片读取（原始字节）", s.handleFSImage)
	_ = agent.KernelAPIRegister("fs.file-info", "GET", "/api/fs/file-info", "文件类型信息", s.handleFSFileInfo)
	_ = agent.KernelAPIRegister("fs.hex", "GET", "/api/fs/hex", "文件十六进制转储", s.handleFSHex)

	// ── 工作区 / 设置 ──
	_ = agent.KernelAPIRegister("workspace", "GET,POST", "/api/workspace", "工作区查询/变更", s.handleWorkspace)
	_ = agent.KernelAPIRegister("settings", "GET,PUT", "/api/settings", "设置读取/保存", s.handleSettings)
	_ = agent.KernelAPIRegister("ui-assembly", "GET,PUT", "/api/ui-assembly", "UI 装配状态磁盘持久化", handler.HandleUIAssembly)

	// ── 对话 ──
	_ = agent.KernelAPIRegister("chat.send", "POST", "/api/chat/send", "启动 agent 会话", s.handleChatSend)
	_ = agent.KernelAPIRegister("chat.stop", "POST", "/api/chat/stop", "停止会话", s.handleChatStop)
	_ = agent.KernelAPIRegister("chat.answer", "POST", "/api/chat/answer", "ask_user 回答", s.handleChatAnswer)
	_ = agent.KernelAPIRegister("chat.approve", "POST", "/api/chat/approve", "审批结果", s.handleChatApprove)
	_ = agent.KernelAPIRegister("chat.feedback", "POST", "/api/chat/feedback", "运行时反馈", s.handleChatFeedback)
	_ = agent.KernelAPIRegister("chat.rollback", "POST", "/api/chat/rollback", "回滚到指定消息", s.handleChatRollback)
	_ = agent.KernelAPIRegister("chat.compact", "POST", "/api/chat/compact", "会话压缩", s.handleChatCompact)

	// ── 对话列表 / 消息 ──
	_ = agent.KernelAPIRegister("conversations", "GET,POST", "/api/conversations", "会话列表/新建", s.handleConversations)
	_ = agent.KernelAPIRegister("conversations.byID", "GET,PUT,DELETE", "/api/conversations/*", "会话详情/重命名/删除（前缀）", s.handleConversationByID)

	// ── Tasks / Plan ──
	_ = agent.KernelAPIRegister("tasks", "GET", "/api/tasks", "任务列表", s.handleTasks)
	_ = agent.KernelAPIRegister("taskplan", "GET", "/api/taskplan", "任务计划", s.handleTaskPlan)

	// ── 模型 / 指令 / 思想 ──
	_ = agent.KernelAPIRegister("models", "GET", "/api/models", "模型列表", s.handleModels)
	_ = agent.KernelAPIRegister("instructions", "GET,PUT", "/api/instructions", "指令读取/保存", s.handleInstructions)
	_ = agent.KernelAPIRegister("philosophy", "GET,PUT", "/api/philosophy", "思想读取/保存", s.handlePhilosophy)

	// ── 工具配置 ──
	_ = agent.KernelAPIRegister("tools", "GET", "/api/tools", "工具列表", handler.HandleTools)
	_ = agent.KernelAPIRegister("tools.review", "GET,PUT", "/api/tools/review", "审核黑白名单配置", handler.HandleReviewConfig)

	// ── MCP / Skills ──
	_ = agent.KernelAPIRegister("mcp.list", "GET", "/api/mcp/list", "MCP 列表", s.handleMCPList)
	_ = agent.KernelAPIRegister("mcp.save", "POST", "/api/mcp/save", "MCP 保存", s.handleMCPSave)
	_ = agent.KernelAPIRegister("skills.list", "GET", "/api/skills/list", "技能列表", s.handleSkillsList)
	_ = agent.KernelAPIRegister("skills.read", "GET", "/api/skills/read", "技能详情", s.handleSkillsRead)
	_ = agent.KernelAPIRegister("skills.save", "POST", "/api/skills/save", "技能保存", s.handleSkillsSave)
	_ = agent.KernelAPIRegister("skills.delete", "POST", "/api/skills/delete", "技能删除", s.handleSkillsDelete)

	// ── Token / Debug ──
	_ = agent.KernelAPIRegister("tokens.stats", "GET", "/api/tokens/stats", "token 统计", s.handleTokensStats)
	_ = agent.KernelAPIRegister("debug.logs", "GET", "/api/debug/logs", "调试日志列表", s.handleDebugLogs)
	_ = agent.KernelAPIRegister("debug.logs.byID", "GET", "/api/debug/logs/*", "调试日志详情（前缀）", s.handleDebugLogByID)

	// ── Git ──
	_ = agent.KernelAPIRegister("git.status", "GET", "/api/git/status", "Git 状态", s.handleGitStatus)
	_ = agent.KernelAPIRegister("git.init", "GET", "/api/git/init", "Git 初始化", s.handleGitInit)
	_ = agent.KernelAPIRegister("git.diff", "GET", "/api/git/diff", "Git diff", s.handleGitDiff)
	_ = agent.KernelAPIRegister("git.add", "POST", "/api/git/add", "Git add", s.handleGitAdd)
	_ = agent.KernelAPIRegister("git.reset", "POST", "/api/git/reset", "Git reset", s.handleGitReset)
	_ = agent.KernelAPIRegister("git.commit", "POST", "/api/git/commit", "Git commit", s.handleGitCommit)
	_ = agent.KernelAPIRegister("git.log", "GET", "/api/git/log", "Git log", s.handleGitLog)
	_ = agent.KernelAPIRegister("git.log.alias", "GET", "/api/git-log", "Git log（避广告拦截器别名）", s.handleGitLog)
	_ = agent.KernelAPIRegister("git.branch", "POST", "/api/git/branch", "Git 分支", s.handleGitBranch)
	_ = agent.KernelAPIRegister("git.checkout", "POST", "/api/git/checkout", "Git checkout", s.handleGitCheckout)
	_ = agent.KernelAPIRegister("git.stash", "POST", "/api/git/stash", "Git stash", s.handleGitStash)
	_ = agent.KernelAPIRegister("git.stash-list", "GET", "/api/git/stash-list", "Git stash 列表", s.handleGitStashList)
	_ = agent.KernelAPIRegister("git.ignore", "GET", "/api/git/ignore", "Git ignore 读取", s.handleGitIgnore)
	_ = agent.KernelAPIRegister("git.discard", "POST", "/api/git/discard", "Git discard", s.handleGitDiscard)
	_ = agent.KernelAPIRegister("git.push", "POST", "/api/git/push", "Git push", s.handleGitPush)
	_ = agent.KernelAPIRegister("git.pull", "POST", "/api/git/pull", "Git pull", s.handleGitPull)
	_ = agent.KernelAPIRegister("git.remote", "GET", "/api/git/remote", "Git remote", s.handleGitRemote)

	// ── 市场 ──
	_ = agent.KernelAPIRegister("marketplace.search", "GET", "/api/marketplace/search", "市场搜索", s.handleMarketplaceSearch)
	_ = agent.KernelAPIRegister("marketplace.install", "POST", "/api/marketplace/install", "市场安装", s.handleMarketplaceInstall)
	_ = agent.KernelAPIRegister("marketplace.uninstall", "POST", "/api/marketplace/uninstall", "市场卸载", s.handleMarketplaceUninstall)
	_ = agent.KernelAPIRegister("marketplace.refresh", "POST", "/api/marketplace/refresh", "市场刷新", s.handleMarketplaceRefresh)

	// ── 记忆 ──
	_ = agent.KernelAPIRegister("memory.search", "GET", "/api/memory/search", "记忆搜索", s.handleMemorySearch)
	_ = agent.KernelAPIRegister("memory.list", "GET", "/api/memory/list", "记忆列表", s.handleMemoryList)
	_ = agent.KernelAPIRegister("memory.rebuild", "POST", "/api/memory/rebuild", "记忆重建", s.handleMemoryRebuild)

	// ── 插件（管理 + 使用 + host/client 事件桥）──
	// ★ 注意：本组接口也由 core-api 插件装配。停用 core-api 会导致插件管理
	//   面板接口消失（管理面自锁）——与停用 ui-app 同理，属预期行为。
	_ = agent.KernelAPIRegister("plugins", "GET", "/api/plugins", "插件列表", handler.HandlePlugins)
	_ = agent.KernelAPIRegister("plugins.detail", "GET", "/api/plugins/detail", "插件详情", handler.HandlePluginDetail)
	_ = agent.KernelAPIRegister("plugins.action", "POST", "/api/plugins/action", "启停/删除插件", handler.HandlePluginAction)
	_ = agent.KernelAPIRegister("plugins.define", "POST", "/api/plugins/define", "定义插件", handler.HandlePluginDefine)
	_ = agent.KernelAPIRegister("plugins.event", "POST", "/api/plugins/event", "client→host 事件桥", handler.HandlePluginEvent)
	_ = agent.KernelAPIRegister("plugins.invoke", "POST", "/api/plugins/invoke", "client 远程调用 host", handler.HandlePluginInvoke)
	_ = agent.KernelAPIRegister("plugins.client-failure", "POST", "/api/plugins/client-failure", "client 失败上报", handler.HandlePluginClientFailure)
	_ = agent.KernelAPIRegister("plugins.client-events", "GET", "/api/plugins/client-events", "host→浏览器事件轮询", handler.HandlePluginClientEvents)
	_ = agent.KernelAPIRegister("plugins.client-state", "GET,POST", "/api/plugins/client-state", "client 快照上报/读取", handler.HandlePluginClientState)
	_ = agent.KernelAPIRegister("plugins.builtin", "GET,POST", "/api/plugins/builtin", "内置工具包开关", handler.HandleBuiltinPlugins)
	_ = agent.KernelAPIRegister("plugins.tool", "POST", "/api/plugins/tool", "工具级开关", handler.HandlePluginToolToggle)

	// ── 工具集（动态构建/固化/导出/导入）──
	_ = agent.KernelAPIRegister("toolsets", "GET", "/api/toolsets", "工具集列表", handler.HandleToolsetsList)
	_ = agent.KernelAPIRegister("toolsets.build", "POST", "/api/toolsets/build", "工具集构建", handler.HandleToolsetBuild)
	_ = agent.KernelAPIRegister("toolsets.export", "GET", "/api/toolsets/export", "工具集导出", handler.HandleToolsetExport)
	_ = agent.KernelAPIRegister("toolsets.import", "POST", "/api/toolsets/import", "工具集导入", handler.HandleToolsetImport)
	_ = agent.KernelAPIRegister("toolsets.remove", "POST", "/api/toolsets/remove", "工具集删除", handler.HandleToolsetRemove)
	_ = agent.KernelAPIRegister("toolsets.edit", "POST", "/api/toolsets/edit", "工具集编辑（add_plugin/rm_plugin/rm_tool/enable_tool）", handler.HandleToolsetEdit)
}
