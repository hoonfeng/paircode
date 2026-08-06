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

	// 消息操作
	AppendMessage(convID string, msg Message, segments []Segment) error
	AppendUserMessage(convID, content string) error
	PersistNewMessages(convID string, hist []Message) error
	LoadLatest(convID string, limit int) ([]StoredMessage, int, error)
	LoadBefore(convID string, beforeIdx int, limit int) ([]StoredMessage, error)
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
