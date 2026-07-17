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
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := a.rawDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// ★ 全量替换：删除该对话所有旧消息，重新写入 hist 中非 system 消息。
	//   参考伴随式 codeagent 每次完整写盘的方式，避免增量计数在压缩后错乱。
	if _, err := tx.Exec(`DELETE FROM messages WHERE conv_id = ?`, convID); err != nil {
		return fmt.Errorf("PersistNewMessages 清理: %w", err)
	}

	ins, err := tx.Prepare(
		`INSERT INTO messages (conv_id, idx, role, content, reasoning, tool_calls, tool_call_id, name, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer ins.Close()

	idx := 0
	for _, m := range hist {
		if m.Role == RoleSystem {
			continue
		}
		tcJSON := "[]"
		if len(m.ToolCalls) > 0 {
			jdata, _ := json.Marshal(m.ToolCalls)
			tcJSON = string(jdata)
		}
		if _, err := ins.Exec(convID, idx, string(m.Role), m.Content, m.Reasoning, tcJSON, m.ToolCallID, m.Name, now); err != nil {
			return fmt.Errorf("PersistNewMessages: %w", err)
		}
		idx++
	}

	if idx > 0 {
		_, _ = tx.Exec(`UPDATE conversations SET msg_count=?, updated_at=? WHERE id=?`, idx, now, convID)
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

func (a *DBAdapter) TruncateTo(convID string, count int) error {
	if _, err := a.rawDB.Exec(`DELETE FROM messages WHERE conv_id = ? AND idx >= ?`, convID, count); err != nil {
		return err
	}
	_, err := a.rawDB.Exec(`UPDATE conversations SET msg_count = ? WHERE id = ?`, count, convID)
	return err
}

func (a *DBAdapter) GetPersistedCount(convID string) int {
	var count int
	_ = a.rawDB.QueryRow(`SELECT COUNT(*) FROM messages WHERE conv_id = ? AND role != 'system'`, convID).Scan(&count)
	return count
}

// ── 合并方法 ──

// ── 合并方法 ──

// MergeLastAssistantRun 合并末尾最近一次 user 消息后的连续 assistant 消息。
// PersistNewMessages 每轮迭代写一条 assistant 消息（可能带空 content + tool_calls），
// 合并后只保留第一条，累积所有 content 和 tool_calls。
func (a *DBAdapter) MergeLastAssistantRun(convID string) error {
	// 使用事务：读全部 → 合并 → 全量重写
	tx, err := a.rawDB.Begin()
	if err != nil {
		return fmt.Errorf("MergeLastAssistantRun: 事务开始失败: %w", err)
	}
	defer tx.Rollback()

	// 读全部消息
	rows, err := tx.Query(
		`SELECT idx, role, content, reasoning, tool_calls, tool_call_id, name, created_at FROM messages WHERE conv_id = ? ORDER BY idx ASC`,
		convID,
	)
	if err != nil {
		return fmt.Errorf("MergeLastAssistantRun: 查询失败: %w", err)
	}

	type row struct {
		idx                                 int
		role, content, reasoning, toolCalls string
		toolCallID, name, createdAt        string
	}
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.idx, &r.role, &r.content, &r.reasoning, &r.toolCalls, &r.toolCallID, &r.name, &r.createdAt); err != nil {
			rows.Close()
			return fmt.Errorf("MergeLastAssistantRun: 扫描失败: %w", err)
		}
		all = append(all, r)
	}
	rows.Close()

	if len(all) < 2 {
		return nil
	}

	// 从末尾找最后一个 user
	lastUser := -1
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].role == "user" {
			lastUser = i
			break
		}
	}
	if lastUser < 0 || lastUser >= len(all)-1 {
		return nil
	}

	// 收集 user 后的连续 assistant 索引
	assistantIdxs := []int{}
	for i := lastUser + 1; i < len(all); i++ {
		r := all[i].role
		if r == "assistant" {
			assistantIdxs = append(assistantIdxs, i)
		} else if r == "tool" {
			continue
		} else {
			break
		}
	}
	if len(assistantIdxs) <= 1 {
		return nil
	}

	// 合并到第一条 assistant
	base := all[assistantIdxs[0]]
	type tcItem struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	var allToolCalls []tcItem
	if base.toolCalls != "" && base.toolCalls != "[]" {
		json.Unmarshal([]byte(base.toolCalls), &allToolCalls)
	}
	for idx := 1; idx < len(assistantIdxs); idx++ {
		next := all[assistantIdxs[idx]]
		if next.content != "" {
			if base.content == "" {
				base.content = next.content
			} else {
				base.content += "\n" + next.content
			}
		}
		if next.toolCalls != "" && next.toolCalls != "[]" {
			var extra []tcItem
			if json.Unmarshal([]byte(next.toolCalls), &extra) == nil {
				allToolCalls = append(allToolCalls, extra...)
			}
		}
	}
	mergedTC, _ := json.Marshal(allToolCalls)
	base.toolCalls = string(mergedTC)

	// 重建：lastUser前的全部 + 合并后的第一条assistant + 中间的tool消息 + 之后的非assistant消息
	var result []row
	result = append(result, all[:lastUser+1]...)
	result = append(result, base)
	for i := lastUser + 1; i < len(all); i++ {
		isAssistant := false
		for _, ai := range assistantIdxs {
			if i == ai {
				isAssistant = true
				break
			}
		}
		if !isAssistant {
			result = append(result, all[i])
		}
	}

	// 重写索引
	for i := range result {
		result[i].idx = i
	}

	// 全量删除旧消息
	if _, err := tx.Exec(`DELETE FROM messages WHERE conv_id = ?`, convID); err != nil {
		return fmt.Errorf("MergeLastAssistantRun: 删除旧消息失败: %w", err)
	}

	// 批量插入
	stmt, err := tx.Prepare(
		`INSERT INTO messages (conv_id, idx, role, content, reasoning, tool_calls, tool_call_id, name, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("MergeLastAssistantRun: 准备插入失败: %w", err)
	}
	defer stmt.Close()
	for _, r := range result {
		if _, err := stmt.Exec(convID, r.idx, r.role, r.content, r.reasoning, r.toolCalls, r.toolCallID, r.name, r.createdAt); err != nil {
			return fmt.Errorf("MergeLastAssistantRun: 插入失败: %w", err)
		}
	}

	return tx.Commit()
}

func (a *DBAdapter) MergeAllAssistantRuns(convID string) error { return nil }

// ── 压缩摘要 ──

func (a *DBAdapter) SaveCompressedSummaries(convID string, summaries []string) error {
	return a.db.SaveCompressedSummaries(convID, summaries)
}

func (a *DBAdapter) LoadCompressedSummaries(convID string) ([]string, error) {
	return a.db.LoadCompressedSummaries(convID)
}

// ReplaceHistory 委托给底层 DBStore 实现。
func (a *DBAdapter) ReplaceHistory(convID string, msgs []Message) error {
	// 用 rawDB 直接执行 DELETE + INSERT 替换
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := a.rawDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM messages WHERE conv_id = ?`, convID); err != nil {
		return err
	}

	stmt, err := tx.Prepare(
		`INSERT INTO messages (conv_id, idx, role, content, reasoning, tool_calls, tool_call_id, name, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	inserted := 0
	for i, m := range msgs {
		if m.Role == RoleSystem || m.Role == RoleUser {
			continue
		}
		tcJSON := "[]"
		if len(m.ToolCalls) > 0 {
			data, _ := json.Marshal(m.ToolCalls)
			tcJSON = string(data)
		}
		if _, err := stmt.Exec(convID, i, string(m.Role), m.Content, m.Reasoning, tcJSON, m.ToolCallID, m.Name, now); err != nil {
			return err
		}
		inserted++
	}

	if _, err := tx.Exec(`UPDATE conversations SET msg_count=?, updated_at=? WHERE id=?`, inserted, now, convID); err != nil {
		return err
	}
	return tx.Commit()
}

// ── 旧格式迁移 ──
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
		// ★ 不检查 len(segs)==0，始终用完整 hist 重建 segments。
		// PersistNewMessages 不写 segments 列，旧 AppendMessage 写入的 segments
		// 可能在 persist 时 tool_result 尚未生成（tool_call 段缺少 result）。
		// 加载时统一用完整 hist 做 look-ahead 才能拿到所有 tool_result。
		segs = SegmentsFromMessage(msg, hist, i)
		out[i] = StoredMessage{
			Idx:       int(r.idx),
			Message:   msg,
			Segments:  segs,
			Timestamp: r.createdAt,
		}
	}
	return out
}
