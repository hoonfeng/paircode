// Package handler — 共享 handler 实现
package handler

// RegisterAll 注册所有 API 路由到 Router。
// 被 cmd/companion/web_server.go 和 cmd/desktop/main.go 共用。
func RegisterAll(r *Router) {
	// 系统
	r.Handle("GET", "/api/health", HandleHealth)
	r.Handle("GET", "/api/system/info", HandleSysInfo)
	r.Handle("POST", "/api/system/exec", HandleExec)

	// 文件系统
	r.Handle("GET", "/api/fs/drives", HandleFSDrives)
	r.Handle("GET", "/api/fs/list", HandleFSList)
	r.Handle("GET", "/api/fs/read", HandleFSRead)
	r.Handle("POST", "/api/fs/write", HandleFSWrite)
	r.Handle("POST", "/api/fs/rename", HandleFSRename)
	r.Handle("DELETE", "/api/fs/delete", HandleFSDelete)
	r.Handle("POST", "/api/fs/mkdir", HandleFSMkdir)
	r.Handle("GET", "/api/fs/search", HandleFSSearch)
	r.Handle("GET", "/api/fs/image", HandleFSImage)
	r.Handle("GET", "/api/fs/file-info", HandleFSFileInfo)
	r.Handle("GET", "/api/fs/hex", HandleFSHex)

	// 工作区 / 设置
	r.Handle("GET", "/api/workspace", HandleWorkspace)
	r.Handle("POST", "/api/workspace", HandleWorkspacePost)
	r.Handle("GET", "/api/settings", HandleSettings)
	r.Handle("PUT", "/api/settings", HandleSettingsPut)

	// Git
	r.Handle("GET", "/api/git/status", HandleGitStatus)
	r.Handle("GET", "/api/git/diff", HandleGitDiff)
	r.Handle("POST", "/api/git/add", HandleGitAdd)
	r.Handle("POST", "/api/git/commit", HandleGitCommit)
	r.Handle("GET", "/api/git/log", HandleGitLog)
	r.Handle("POST", "/api/git/branch", HandleGitBranch)
	r.Handle("POST", "/api/git/checkout", HandleGitCheckout)
	r.Handle("POST", "/api/git/stash", HandleGitStash)
	r.Handle("GET", "/api/git/stash-list", HandleGitStashList)
	r.Handle("GET", "/api/git/ignore", HandleGitIgnore)
	r.Handle("POST", "/api/git/discard", HandleGitDiscard)
	r.Handle("POST", "/api/git/push", HandleGitPush)
	r.Handle("POST", "/api/git/pull", HandleGitPull)
	r.Handle("GET", "/api/git/remote", HandleGitRemote)
	r.Handle("GET", "/api/git/init", HandleGitInit)
	r.Handle("POST", "/api/git/reset", HandleGitReset)

	// 对话
	r.Handle("POST", "/api/chat/send", HandleChatSend)
	r.Handle("POST", "/api/chat/stop", HandleChatStop)
	r.Handle("POST", "/api/chat/answer", HandleChatAnswer)
	r.Handle("POST", "/api/chat/approve", HandleChatApprove)
	r.Handle("POST", "/api/chat/feedback", HandleChatFeedback)
	r.Handle("POST", "/api/chat/rollback", HandleChatRollback)

	// 对话列表 / 消息
	r.Handle("GET", "/api/conversations", HandleConversations)
	r.Handle("POST", "/api/conversations", HandleConversationCreate)
	r.Handle("GET", "/api/conversations/", HandleConversationByID)

	// Tasks / Plan
	r.Handle("GET", "/api/tasks", HandleTasks)
	r.Handle("GET", "/api/taskplan", HandleTaskPlan)

	// 模型 / 指令 / 思想
	r.Handle("GET", "/api/models", HandleModels)
	r.Handle("GET", "/api/instructions", HandleInstructions)
	r.Handle("PUT", "/api/instructions", HandleInstructionsPut)
	r.Handle("GET", "/api/philosophy", HandlePhilosophy)
	r.Handle("PUT", "/api/philosophy", HandlePhilosophyPut)

	// 工具配置（启用开关 + 审核黑白名单）
	r.Handle("GET", "/api/tools", HandleTools)
	r.Handle("PUT", "/api/tools/save", HandleToolsSave)
	r.Handle("GET", "/api/tools/review", HandleReviewConfig)
	r.Handle("PUT", "/api/tools/review", HandleReviewConfig)

	// MCP / Skills
	r.Handle("GET", "/api/mcp/list", HandleMCPList)
	r.Handle("POST", "/api/mcp/save", HandleMCPSave)
	r.Handle("GET", "/api/skills/list", HandleSkillsList)
	r.Handle("GET", "/api/skills/read", HandleSkillsRead)
	r.Handle("POST", "/api/skills/save", HandleSkillsSave)
	r.Handle("POST", "/api/skills/delete", HandleSkillsDelete)

	// Token / Debug
	r.Handle("GET", "/api/tokens/stats", HandleTokensStats)
	r.Handle("GET", "/api/debug/logs", HandleDebugLogs)
	r.Handle("GET", "/api/debug/logs/", HandleDebugLogByID)

	// 市场
	r.Handle("GET", "/api/marketplace/search", HandleMarketplaceSearch)
	r.Handle("POST", "/api/marketplace/install", HandleMarketplaceInstall)
	r.Handle("POST", "/api/marketplace/refresh", HandleMarketplaceRefresh)

	// 记忆
	r.Handle("GET", "/api/memory/search", HandleMemorySearch)
	r.Handle("GET", "/api/memory/list", HandleMemoryList)
	r.Handle("POST", "/api/memory/rebuild", HandleMemoryRebuild)

	// 插件（管理 + 使用 + host/client 事件桥）
	r.Handle("GET", "/api/plugins", HandlePlugins)
	r.Handle("GET", "/api/plugins/detail", HandlePluginDetail)
	r.Handle("POST", "/api/plugins/action", HandlePluginAction)
	r.Handle("POST", "/api/plugins/define", HandlePluginDefine)
	r.Handle("POST", "/api/plugins/event", HandlePluginEvent)
	r.Handle("GET", "/api/plugins/client-events", HandlePluginClientEvents)
}
