package agent

// db_adapter.go — 将 pkg/db.SQLiteDB 适配为 agent.ConversationStore。
// 类型翻译层：agent.ConversationMeta ↔ db.Conversation，agent.Message ↔ db.Message。
// store_kv 表用于 CtxStats/SummaryAt 等 KV 元数据。

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	pkgdb "github.com/hoonfeng/paircode/pkg/db"
	_ "modernc.org/sqlite"
)

// DBAdapter 包装 pkg/db.SQLiteDB，实现 ConversationStore 接口。
type DBAdapter struct {
	db     *pkgdb.SQLiteDB
	rawDB  *sql.DB
	root   string
}

// NewDBAdapter 创建适配器。db 为已打开的 pkg/db.SQLiteDB 实例。
func NewDBAdapter(database *pkgdb.SQLiteDB, root string) *DBAdapter {
	raw := database.RawDB().(*sql.DB)
	return &DBAdapter{
		db:    database,
		rawDB: raw,
		root:  root,
	}
}

// ── 对话 CRUD ──

func (a *DBAdapter) CreateConversation(convID, title, workspaceRoot string) error {
	now := time.Now().UTC()
	conv := &pkgdb.Conversation{
		ID:            convID,
		Title:         title,
		WorkspaceRoot: workspaceRoot,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return a.db.CreateConversation(conv)
}

func (a *DBAdapter) GetConversation(convID string) (*ConversationMeta, error) {
	conv, err := a.db.GetConversation(convID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if conv == nil {
		return nil, nil
	}
	meta := &ConversationMeta{
		ID:            conv.ID,
		Title:         conv.Title,
		WorkspaceRoot: conv.WorkspaceRoot,
		CreatedAt:     conv.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     conv.UpdatedAt.Format(time.RFC3339),
		Summary:       conv.Summary,
		MsgCount:      conv.MsgCount,
	}
	// 从 KV 读 CtxStats
	if statsJSON, _ := a.db.ReadKV("ctx_" + convID); statsJSON != "" {
		var stats Usage
		if json.Unmarshal([]byte(statsJSON), &stats) == nil {
			meta.CtxStats = &stats
		}
	}
	return meta, nil
}

func (a *DBAdapter) ListConversations(workspaceRoot string) ([]ConversationMeta, error) {
	convs, err := a.db.ListConversationsByWorkspace(workspaceRoot)
	if err != nil {
		return nil, err
	}
	metas := make([]ConversationMeta, 0, len(convs))
	for _, conv := range convs {
		meta := ConversationMeta{
			ID:            conv.ID,
			Title:         conv.Title,
			WorkspaceRoot: conv.WorkspaceRoot,
			CreatedAt:     conv.CreatedAt.Format(time.RFC3339),
			UpdatedAt:     conv.UpdatedAt.Format(time.RFC3339),
			Summary:       conv.Summary,
			MsgCount:      conv.MsgCount,
		}
		if statsJSON, _ := a.db.ReadKV("ctx_" + conv.ID); statsJSON != "" {
			var stats Usage
			if json.Unmarshal([]byte(statsJSON), &stats) == nil {
				meta.CtxStats = &stats
			}
		}
		metas = append(metas, meta)
	}
	return metas, nil
}

func (a *DBAdapter) DeleteConversation(convID string) error {
	return a.db.DeleteConversation(convID)
}

func (a *DBAdapter) UpdateTitle(convID, title string) error {
	return a.db.UpdateTitle(convID, title)
}

func (a *DBAdapter) SetSummary(convID, summary, summaryAt string) error {
	_, err := a.rawDB.Exec(`UPDATE conversations SET summary=?, updated_at=? WHERE id=?`, summary, time.Now().UTC().Format(time.RFC3339), convID)
	if err != nil {
		return err
	}
	return a.db.WriteKV("summary_at_"+convID, summaryAt)
}

func (a *DBAdapter) SetCtxStats(convID string, stats *Usage) error {
	if stats == nil {
		_, _ = a.rawDB.Exec(`UPDATE conversations SET updated_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339), convID)
		return nil
	}
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	if err := a.db.WriteKV("ctx_"+convID, string(data)); err != nil {
		return err
	}
	_, _ = a.rawDB.Exec(`UPDATE conversations SET updated_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339), convID)
	return nil
}

// ── 消息操作 ──

func (a *DBAdapter) AppendMessage(convID string, msg Message, _ []Segment) error {
	// 获取当前序号：使用 MAX(idx)+1 而非 COUNT(*)，避免 PersistNewMessages 产生 idx 空洞后冲突
	var idx int
	_ = a.rawDB.QueryRow(`SELECT COALESCE(MAX(idx), -1) + 1 FROM messages WHERE conv_id = ?`, convID).Scan(&idx)

	now := time.Now().UTC().Format(time.RFC3339)
	tcJSON := "[]"
	if len(msg.ToolCalls) > 0 {
		jdata, _ := json.Marshal(msg.ToolCalls)
		tcJSON = string(jdata)
	}

	_, err := a.rawDB.Exec(
		`INSERT INTO messages (conv_id, idx, role, content, reasoning, tool_calls, tool_call_id, name, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		convID, idx, string(msg.Role), msg.Content, msg.Reasoning, tcJSON, msg.ToolCallID, msg.Name, now,
	)
	if err != nil {
		return fmt.Errorf("AppendMessage: %w", err)
	}
	_, _ = a.rawDB.Exec(`UPDATE conversations SET msg_count=msg_count+1, updated_at=? WHERE id=?`, now, convID)
	return nil
}

func (a *DBAdapter) AppendUserMessage(convID, content string) error {
	return a.AppendMessage(convID, Message{Role: RoleUser, Content: content}, nil)
}

// PersistNewMessages 增量持久化 hist 中的新消息（idx > 已持久化最大 idx 的写入事务）。
func (a *DBAdapter) PersistNewMessages(convID string, hist []Message) error {
	var maxIdx int
	_ = a.rawDB.QueryRow(`SELECT COALESCE(MAX(idx), -1) FROM messages WHERE conv_id = ?`, convID).Scan(&maxIdx)

	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := a.rawDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertStmt, err := tx.Prepare(
		`INSERT OR IGNORE INTO messages (conv_id, idx, role, content, reasoning, tool_calls, tool_call_id, name, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	newCount := 0
	for i, m := range hist {
		if m.Role == RoleSystem || m.Role == RoleUser {
			continue
		}
		if i <= maxIdx {
			continue
		}
		tcJSON := "[]"
		if len(m.ToolCalls) > 0 {
			jdata, _ := json.Marshal(m.ToolCalls)
			tcJSON = string(jdata)
		}
		if _, err := insertStmt.Exec(convID, i, string(m.Role), m.Content, m.Reasoning, tcJSON, m.ToolCallID, m.Name, now); err != nil {
			return fmt.Errorf("PersistNewMessages: %w", err)
		}
		newCount++
	}
	if newCount > 0 {
		_, _ = tx.Exec(`UPDATE conversations SET msg_count=msg_count+?, updated_at=? WHERE id=?`, newCount, now, convID)
	}
	return tx.Commit()
}

// LoadLatest 加载对话最新的 limit 条消息。
func (a *DBAdapter) LoadLatest(convID string, limit int) ([]StoredMessage, int, error) {
	var total int
	_ = a.rawDB.QueryRow(`SELECT COUNT(*) FROM messages WHERE conv_id = ?`, convID).Scan(&total)

	var rows *sql.Rows
	var err error
	if limit > 0 && total > limit {
		rows, err = a.rawDB.Query(
			`SELECT id, idx, role, content, reasoning, tool_calls, tool_call_id, name, segments, created_at FROM messages WHERE conv_id = ? ORDER BY idx ASC LIMIT ? OFFSET ?`,
			convID, limit, total-limit,
		)
	} else {
		rows, err = a.rawDB.Query(
			`SELECT id, idx, role, content, reasoning, tool_calls, tool_call_id, name, segments, created_at FROM messages WHERE conv_id = ? ORDER BY idx ASC`,
			convID,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	msgs := a.scanMessages(rows)
	if msgs == nil {
		msgs = []StoredMessage{}
	}
	return msgs, total, nil
}

// LoadBefore 加载 idx < beforeIdx 的最新 limit 条消息。
func (a *DBAdapter) LoadBefore(convID string, beforeIdx int, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := a.rawDB.Query(
		`SELECT id, idx, role, content, reasoning, tool_calls, tool_call_id, name, segments, created_at FROM messages WHERE conv_id = ? AND idx < ? ORDER BY idx DESC LIMIT ?`,
		convID, beforeIdx, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgs := a.scanMessages(rows)
	// 反转成 ASC
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	if msgs == nil {
		msgs = []StoredMessage{}
	}
	return msgs, nil
}

// LoadAll 加载对话全部消息（仅 Message，不含 Segments）。
func (a *DBAdapter) LoadAll(convID string) ([]Message, error) {
	rows, err := a.rawDB.Query(
		`SELECT role, content, reasoning, tool_calls, tool_call_id, name FROM messages WHERE conv_id = ? ORDER BY idx ASC`,
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var msg Message
		var role, tcJSON, toolCallID, name string
		if err := rows.Scan(&role, &msg.Content, &msg.Reasoning, &tcJSON, &toolCallID, &name); err != nil {
			continue
		}
		msg.Role = Role(role)
		msg.ToolCallID = toolCallID
		msg.Name = name
		if tcJSON != "" && tcJSON != "[]" {
			json.Unmarshal([]byte(tcJSON), &msg.ToolCalls)
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func (a *DBAdapter) Count(convID string) (int, error) {
	return a.db.GetMessageCount(convID)
}

func (a *DBAdapter) GetPersistedCount(convID string) int {
	var count int
	_ = a.rawDB.QueryRow(`SELECT COUNT(*) FROM messages WHERE conv_id = ? AND role != 'system'`, convID).Scan(&count)
	return count
}

// ── 合并方法（SQLite 模式下均为无操作）──

func (a *DBAdapter) MergeLastAssistantRun(convID string) error { return nil }
func (a *DBAdapter) MergeAllAssistantRuns(convID string) error { return nil }

// ── 压缩摘要 ──

func (a *DBAdapter) SaveCompressedSummaries(convID string, summaries []string) error {
	return a.db.SaveCompressedSummaries(convID, summaries)
}

func (a *DBAdapter) LoadCompressedSummaries(convID string) ([]string, error) {
	return a.db.LoadCompressedSummaries(convID)
}

// ── 旧格式迁移 ──

func (a *DBAdapter) MigrateFromLegacy(conversationsJSONPath, historyCacheJSONPath string) error {
	var count int
	if err := a.rawDB.QueryRow(`SELECT COUNT(*) FROM conversations`).Scan(&count); err == nil && count > 0 {
		return nil // 已有数据，跳过迁移
	}
	legacyStore := &MessageStore{root: a.root}
	_ = os.MkdirAll(filepath.Join(a.root, ".pair", "conversations"), 0o755)
	return legacyStore.MigrateFromLegacy(conversationsJSONPath, historyCacheJSONPath)
}

// ── 辅助：scanMessages ──

type rawRow struct {
	id, idx                  int64
	role, content, reasoning string
	tcJSON, toolCallID, name string
	segJSON, createdAt       string
}

func (a *DBAdapter) scanMessages(rows *sql.Rows) []StoredMessage {
	var raw []rawRow
	for rows.Next() {
		var r rawRow
		if err := rows.Scan(&r.id, &r.idx, &r.role, &r.content, &r.reasoning, &r.tcJSON, &r.toolCallID, &r.name, &r.segJSON, &r.createdAt); err != nil {
			continue
		}
		raw = append(raw, r)
	}
	if len(raw) == 0 {
		return nil
	}

	// 构建完整 hist 用于 SegmentsFromMessage 的 look-ahead
	hist := make([]Message, len(raw))
	for i, r := range raw {
		msg := Message{
			Role:       Role(r.role),
			Content:    r.content,
			Reasoning:  r.reasoning,
			ToolCallID: r.toolCallID,
			Name:       r.name,
		}
		if r.tcJSON != "" && r.tcJSON != "[]" {
			json.Unmarshal([]byte(r.tcJSON), &msg.ToolCalls)
		}
		hist[i] = msg
	}

	out := make([]StoredMessage, len(raw))
	for i, r := range raw {
		msg := Message{
			Role:       Role(r.role),
			Content:    r.content,
			Reasoning:  r.reasoning,
			ToolCallID: r.toolCallID,
			Name:       r.name,
		}
		if r.tcJSON != "" && r.tcJSON != "[]" {
			json.Unmarshal([]byte(r.tcJSON), &msg.ToolCalls)
		}
		var segs []Segment
		if r.segJSON != "" && r.segJSON != "[]" {
			json.Unmarshal([]byte(r.segJSON), &segs)
		}
		if len(segs) == 0 {
			segs = SegmentsFromMessage(msg, hist, i)
		}
		out[i] = StoredMessage{
			Idx:       int(r.idx),
			Message:   msg,
			Segments:  segs,
			Timestamp: r.createdAt,
		}
	}
	return out
}
