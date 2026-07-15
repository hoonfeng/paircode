package agent

// DBStore — SQLite 驱动的对话存储，替代 JSONL MessageStore。
// Segments 读时计算，不再冗余存储（缓存列 segments 仅用于前端展示首次计算后缓存）。
// 单文件 .pair/pair.db，零外部依赖（modernc.org/sqlite 纯 Go）。

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// DBStore 对话存储的 SQLite 实现。
// 以单数据库文件存储全部对话和消息，无需 JSONL 逐行扫描。
type DBStore struct {
	db        *sql.DB
	root      string
	mu        sync.Mutex // 写操作串行化（SQLite 单连接）
	migrated  bool       // 是否已完成从 JSONL 的迁移
}

///////////////////////////////
// ── 对话方法 ──
///////////////////////////////

// NewDBStore 创建或打开 SQLite 对话存储。
// dbPath 为数据库文件路径（如 .pair/pair.db）。
// 自动建表和迁移旧 JSONL 数据。
func NewDBStore(root string) *DBStore {
	dbPath := filepath.Join(root, ".pair", "pair.db")
	os.MkdirAll(filepath.Dir(dbPath), 0o755)

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil
	}
	db.SetMaxOpenConns(1)

	s := &DBStore{db: db, root: root}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil
	}
	return s
}

// migrate 创建表并执行 JSONL→SQLite 迁移。
func (s *DBStore) migrate() error {
	tables := []string{
		// 对话元数据表
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT DEFAULT '',
			workspace_root TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			summary TEXT DEFAULT '',
			msg_count INTEGER DEFAULT 0
		)`,
		// 消息表：原始字段存储，Segments 读时计算
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conv_id TEXT NOT NULL REFERENCES conversations(id),
			idx INTEGER NOT NULL,
			role TEXT NOT NULL,
			content TEXT DEFAULT '',
			reasoning TEXT DEFAULT '',
			tool_calls TEXT DEFAULT '[]',
			tool_call_id TEXT DEFAULT '',
			name TEXT DEFAULT '',
			segments TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(conv_id, idx)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_db_messages_conv ON messages(conv_id, idx)`,
		// KV 存储：适用于 CtxStats 等结构化元数据
		`CREATE TABLE IF NOT EXISTS store_kv (
			key TEXT PRIMARY KEY,
			value TEXT DEFAULT ''
		)`,
		// 压缩摘要存储
		`CREATE TABLE IF NOT EXISTS conv_summaries (
			conv_id TEXT NOT NULL,
			idx INTEGER NOT NULL,
			summary TEXT DEFAULT '',
			PRIMARY KEY(conv_id, idx)
		)`,
	}
	for _, ddl := range tables {
		if _, err := s.db.Exec(ddl); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// Close 关闭数据库连接。
func (s *DBStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// checkDB 检查数据库连接是否有效，无效时尝试重连。
// 所有公共方法应在获取 s.mu 后最先调用此方法。
func (s *DBStore) checkDB() error {
	if s.db == nil {
		return s.ensureDB()
	}
	return nil
}

// ── 对话 CRUD ──

func (s *DBStore) CreateConversation(convID, title, workspaceRoot string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO conversations (id, title, workspace_root, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		convID, title, workspaceRoot, now, now,
	)
	return err
}

func (s *DBStore) GetConversation(convID string) (*ConversationMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	row := s.db.QueryRow(
		`SELECT id, title, workspace_root, created_at, updated_at, summary, msg_count FROM conversations WHERE id = ?`,
		convID,
	)
	var meta ConversationMeta
	var createdAt, updatedAt string
	if err := row.Scan(&meta.ID, &meta.Title, &meta.WorkspaceRoot, &createdAt, &updatedAt, &meta.Summary, &meta.MsgCount); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	// 读取 KV 中的 CtxStats
	if statsJSON, _ := s.readKV("ctx_" + convID); statsJSON != "" {
		var stats Usage
		if json.Unmarshal([]byte(statsJSON), &stats) == nil {
			meta.CtxStats = &stats
		}
	}
	return &meta, nil
}

func (s *DBStore) UpdateConversation(meta *ConversationMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`UPDATE conversations SET title=?, updated_at=?, summary=?, msg_count=? WHERE id=?`,
		meta.Title, meta.UpdatedAt, meta.Summary, meta.MsgCount, meta.ID,
	)
	return err
}

func (s *DBStore) ListConversations(workspaceRoot string) ([]ConversationMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT id, title, workspace_root, created_at, updated_at, summary, msg_count FROM conversations WHERE workspace_root=? OR workspace_root='' ORDER BY updated_at DESC`,
		workspaceRoot,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var metas []ConversationMeta
	for rows.Next() {
		var meta ConversationMeta
		var createdAt, updatedAt string
		if err := rows.Scan(&meta.ID, &meta.Title, &meta.WorkspaceRoot, &createdAt, &updatedAt, &meta.Summary, &meta.MsgCount); err != nil {
			continue
		}
		// 读 KV 中的 CtxStats
		if statsJSON, _ := s.readKV("ctx_" + meta.ID); statsJSON != "" {
			var stats Usage
			if json.Unmarshal([]byte(statsJSON), &stats) == nil {
				meta.CtxStats = &stats
			}
		}
		metas = append(metas, meta)
	}
	if metas == nil {
		metas = []ConversationMeta{}
	}
	return metas, nil
}

func (s *DBStore) DeleteConversation(convID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return err
	}
	_, _ = s.db.Exec(`DELETE FROM messages WHERE conv_id = ?`, convID)
	_, _ = s.db.Exec(`DELETE FROM conv_summaries WHERE conv_id = ?`, convID)
	_, _ = s.db.Exec(`DELETE FROM store_kv WHERE key = ?`, "ctx_"+convID)
	_, err := s.db.Exec(`DELETE FROM conversations WHERE id = ?`, convID)
	return err
}

// ── 消息操作 ──

// AppendMessage 追加一条消息。
// segments 参数保留兼容签名，但不再持久化（读时从 Message 重新计算）。
func (s *DBStore) AppendMessage(convID string, msg Message, _ []Segment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return err
	}
	// 获取当前序号
	var idx int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE conv_id = ?`, convID).Scan(&idx)
	if err != nil {
		idx = 0
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tcJSON := "[]"
	if len(msg.ToolCalls) > 0 {
		data, _ := json.Marshal(msg.ToolCalls)
		tcJSON = string(data)
	}

	_, err = s.db.Exec(
		`INSERT INTO messages (conv_id, idx, role, content, reasoning, tool_calls, tool_call_id, name, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		convID, idx, string(msg.Role), msg.Content, msg.Reasoning, tcJSON, msg.ToolCallID, msg.Name, now,
	)
	if err != nil {
		return fmt.Errorf("AppendMessage: %w", err)
	}

	// 更新 conversations 计数
	_, _ = s.db.Exec(`UPDATE conversations SET msg_count=msg_count+1, updated_at=? WHERE id=?`, now, convID)
	return nil
}

// AppendUserMessage 便捷封装：追加用户消息。
func (s *DBStore) AppendUserMessage(convID, content string) error {
	return s.AppendMessage(convID, Message{Role: RoleUser, Content: content}, nil)
}

// LoadLatest 加载对话最新的消息。
// limit <= 0 时返回全部。
func (s *DBStore) LoadLatest(convID string, limit int) ([]StoredMessage, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return nil, 0, err
	}
	var total int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE conv_id = ?`, convID).Scan(&total)

	// 读取消息（最新 limit 条或全部）
	var rows *sql.Rows
	var err error
	if limit > 0 && total > limit {
		rows, err = s.db.Query(
			`SELECT id, idx, role, content, reasoning, tool_calls, tool_call_id, name, segments, created_at FROM messages WHERE conv_id = ? ORDER BY idx ASC LIMIT ? OFFSET ?`,
			convID, limit, total-limit,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, idx, role, content, reasoning, tool_calls, tool_call_id, name, segments, created_at FROM messages WHERE conv_id = ? ORDER BY idx ASC`,
			convID,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	msgs := s.scanMessages(rows)
	if msgs == nil {
		msgs = []StoredMessage{}
	}
	return msgs, total, nil
}

// LoadBefore 加载 idx < beforeIdx 的最新 limit 条消息。
func (s *DBStore) LoadBefore(convID string, beforeIdx int, limit int) ([]StoredMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT id, idx, role, content, reasoning, tool_calls, tool_call_id, name, segments, created_at FROM messages WHERE conv_id = ? AND idx < ? ORDER BY idx DESC LIMIT ?`,
		convID, beforeIdx, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgs := s.scanMessages(rows)
	// 反转成 ASC 顺序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	if msgs == nil {
		msgs = []StoredMessage{}
	}
	return msgs, nil
}

// LoadAll 加载对话全部消息（仅 Message，不含 Segments）。
func (s *DBStore) LoadAll(convID string) ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
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

// Count 返回对话消息数。
func (s *DBStore) Count(convID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return 0, err
	}
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE conv_id = ?`, convID).Scan(&count)
	return count, err
}

// GetPersistedCount 返回已持久化的非 System 消息数。
// SQLite 模式下，所有消息均已持久化，直接返回 COUNT。
func (s *DBStore) GetPersistedCount(convID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return 0
	}
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE conv_id = ? AND role != 'system'`, convID).Scan(&count)
	return count
}

// PersistNewMessages 原子性追加新消息。
// hist 中 idx 大于当前最大 idx 的消息将被写入。
func (s *DBStore) PersistNewMessages(convID string, hist []Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return err
	}
	// 获取当前最大 idx
	var maxIdx int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(idx), -1) FROM messages WHERE conv_id = ?`, convID).Scan(&maxIdx)

	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
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
		// 只处理未持久化的消息
		if i <= maxIdx {
			continue
		}
		tcJSON := "[]"
		if len(m.ToolCalls) > 0 {
			data, _ := json.Marshal(m.ToolCalls)
			tcJSON = string(data)
		}
		_, err := insertStmt.Exec(convID, i, string(m.Role), m.Content, m.Reasoning, tcJSON, m.ToolCallID, m.Name, now)
		if err != nil {
			return fmt.Errorf("PersistNewMessages: %w", err)
		}
		newCount++
	}

	if newCount > 0 {
		_, _ = tx.Exec(`UPDATE conversations SET msg_count=msg_count+?, updated_at=? WHERE id=?`, newCount, now, convID)
	}

	return tx.Commit()
}

// MergeLastAssistantRun 合并 JSONL 末尾最近的连续助理条目。
// SQLite 模式下，消息已经是原子写入的，无需合并。
// 保留此方法用于向后兼容。
func (s *DBStore) MergeLastAssistantRun(convID string) error {
	// SQLite 模式下每条消息独立写入，不存在合并问题
	return nil
}

// MergeAllAssistantRuns 全量合并 JSONL 中所有 user→assistant 的连续条目。
// SQLite 模式下无需合并。
func (s *DBStore) MergeAllAssistantRuns(convID string) error {
	return nil
}

// ── 元数据操作 ──

// UpdateTitle 更新对话标题。
func (s *DBStore) UpdateTitle(convID, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE conversations SET title=?, updated_at=? WHERE id=?`, title, now, convID)
	return err
}

// SetSummary 设置对话摘要。
func (s *DBStore) SetSummary(convID, summary, summaryAt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE conversations SET summary=?, updated_at=? WHERE id=?`, summary, now, convID)
	if err != nil {
		return err
	}
	// 摘要时间存在 KV 中（conversations 表无此列）
	return s.writeKV("summary_at_"+convID, summaryAt)
}

// SetCtxStats 设置对话的上下文 token 统计。
func (s *DBStore) SetCtxStats(convID string, stats *Usage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if stats == nil {
		_, _ = s.db.Exec(`UPDATE conversations SET updated_at=? WHERE id=?`, now, convID)
		return nil
	}
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	if err := s.writeKV("ctx_"+convID, string(data)); err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE conversations SET updated_at=? WHERE id=?`, now, convID)
	return err
}

// ── 压缩摘要 ──

func (s *DBStore) SaveCompressedSummaries(convID string, summaries []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return err
	}
	// 删除旧摘要
	_, _ = s.db.Exec(`DELETE FROM conv_summaries WHERE conv_id = ?`, convID)
	if len(summaries) == 0 {
		return nil
	}
	// 批量写入
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO conv_summaries (conv_id, idx, summary) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, summary := range summaries {
		if _, err := stmt.Exec(convID, i, summary); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *DBStore) LoadCompressedSummaries(convID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT summary FROM conv_summaries WHERE conv_id = ? ORDER BY idx ASC`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var summaries []string
	for rows.Next() {
		var summary string
		if err := rows.Scan(&summary); err != nil {
			continue
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

// ── 旧格式迁移 ──

// MigrateFromLegacy 从旧 JSONL 格式迁移到 SQLite。
func (s *DBStore) MigrateFromLegacy(conversationsJSONPath, historyCacheJSONPath string) error {
	// 若已有数据则不迁移
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM conversations`).Scan(&count); err == nil && count > 0 {
		return nil
	}

	// 读取旧 conversations.json（如存在）
	legacyStore := &MessageStore{root: s.root}
	_ = os.MkdirAll(filepath.Join(s.root, ".pair", "conversations"), 0o755)
	return legacyStore.MigrateFromLegacy(conversationsJSONPath, historyCacheJSONPath)
}

// ── 内部辅助 ──

// scanMessages 将 SQL 行扫描为 StoredMessage 列表。
// 计算 Segments 的策略：
//   - 已缓存（segments 非空）→ 直接用缓存
//   - 未缓存 → SegmentsFromMessage 计算并写回缓存
func (s *DBStore) scanMessages(rows *sql.Rows) []StoredMessage {
	// 先扫描全部到内存（需要完整 hist 做 look-ahead）
	type rawRow struct {
		id                              int64
		idx                             int
		role, content, reasoning        string
		tcJSON, toolCallID, name        string
		segJSON, createdAt              string
	}
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

	// 构建完整 hist 用于 look-ahead
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

	// 输出
	out := make([]StoredMessage, len(raw))
	for i, r := range raw {
		msg := Message{
			Role:       Role(r.role),
			Content:    r.content,
			Reasoning:  r.reasoning,
			ToolCalls:  nil,
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
			Idx:       r.idx,
			Message:   msg,
			Segments:  segs,
			Timestamp: r.createdAt,
		}
	}
	return out
}

// readKV 从 store_kv 表读取一个键值。
func (s *DBStore) readKV(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM store_kv WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

// writeKV 写入一个键值到 store_kv 表。
func (s *DBStore) writeKV(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO store_kv (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value,
	)
	return err
}

// ensureDB 确保数据库已连接，未连接时尝试重连。
func (s *DBStore) ensureDB() error {
	if s.db != nil {
		return nil
	}
	dbPath := filepath.Join(s.root, ".pair", "pair.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("重连数据库失败: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := s.migrate(); err != nil {
		db.Close()
		return fmt.Errorf("重新迁移失败: %w", err)
	}
	s.db = db
	return nil
}

// ── 兼容：MessageStore 未提供的额外方法 ──
// 以下方法供 SessionManager 或 web 层统一调用，避免类型判断。

// createConversationMeta 从 db 行创建完整的 ConversationMeta（含 CtxStats）。
func (s *DBStore) createConversationMeta(id, title, workspaceRoot, createdAt, updatedAt, summary string, msgCount int) ConversationMeta {
	meta := ConversationMeta{
		ID:            id,
		Title:         title,
		WorkspaceRoot: workspaceRoot,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		MsgCount:      msgCount,
		Summary:       summary,
	}
	// 尝试读取 CtxStats
	if statsJSON, _ := s.readKV("ctx_" + id); statsJSON != "" {
		var stats Usage
		if json.Unmarshal([]byte(statsJSON), &stats) == nil {
			meta.CtxStats = &stats
		}
	}
	return meta
}

// 添加空行以禁止 gofmt 合并下面的注释
func initStoreKVPragma() {} // 保持 gofmt 兼容
