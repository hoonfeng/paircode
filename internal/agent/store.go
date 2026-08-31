package agent

// ConversationStore 对话存储接口。
// DBStore（SQLite 新版）和 MessageStore（JSONL 旧版）均实现此接口。
// SessionManager 通过此接口操作存储，调用方无需关心后端实现。
type ConversationStore interface {
	// 对话 CRUD
	CreateConversation(convID, title, workspaceRoot string) error
	GetConversation(convID string) (*ConversationMeta, error)
	ListConversations(workspaceRoot string) ([]ConversationMeta, error)
	DeleteConversation(convID string) error
	UpdateTitle(convID, title string) error
	SetSummary(convID, summary, summaryAt string) error
	SetCtxStats(convID string, stats *Usage) error
	// SetInterrupted 更新对话的异常中断标记（前端据此显示"未完成可继续"）。
	SetInterrupted(convID string, interrupted bool) error
	// SetConvModel 设置会话级模型路由（服务商 + 执行模型；★ 2026-08-31
	// 模型切换只作用于本会话，不改全局设置、不影响历史对话）。
	SetConvModel(convID, provider, model string) error

	// 消息操作
	AppendMessage(convID string, msg Message, segments []Segment) error
	AppendUserMessage(convID, content string) error
	AppendUserMessageWithImages(convID, content string, images []ImagePart) error // ★ 2026-08-21 多模态：带图片的用户消息落盘
	PersistNewMessages(convID string, hist []Message) error
	LoadLatest(convID string, limit int) ([]StoredMessage, int, error)
	LoadBefore(convID string, beforeIdx int, limit int) ([]StoredMessage, error)
	// ★ 2026-08-22 展示专用（全量合并后再按合并条数 limit）：LoadLatest/LoadBefore
	//   按原始行 limit，JSONL 行 = 一次迭代/tool 行，长对话合并后每轮仅 1~5 条，
	//   前端历史加载极慢。展示接口返回合并后的 limit 条消息。
	LoadLatestForDisplay(convID string, limit int) ([]StoredMessage, int, error)
	LoadBeforeForDisplay(convID string, beforeIdx int, limit int) ([]StoredMessage, error)
	LoadAll(convID string) ([]Message, error)
	Count(convID string) (int, error)
	GetPersistedCount(convID string) int
	MergeLastAssistantRun(convID string) error
	MergeAllAssistantRuns(convID string) error

	// 压缩摘要
	SaveCompressedSummaries(convID string, summaries []string) error
	LoadCompressedSummaries(convID string) ([]string, error)

	// ReplaceHistory 替换整个对话的消息历史（压缩后调用——删除旧消息，写入压缩后的版本）。
	ReplaceHistory(convID string, msgs []Message) error

	// 迁移
	MigrateFromLegacy(conversationsJSONPath, historyCacheJSONPath string) error

	TruncateTo(convID string, count int) error
}
