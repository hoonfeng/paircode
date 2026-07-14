// MessageStore: 对话消息持久化的唯一权威，JSONL 单文件存储 + index.json 元数据
package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Segment 前端展示用的消息分段（thinking/content/tool_call/tool_result/ask_user 等）。
type Segment struct {
	Type     string `json:"type"`               // thinking | content | tool_call | tool_result | ask_user
	Content  string `json:"content,omitempty"`  // 文本内容（thinking/content/tool_result）
	Name     string `json:"name,omitempty"`     // 工具名（tool_call）
	ArgsRaw  string `json:"argsRaw,omitempty"`  // 工具参数 JSON 字符串（tool_call）
	Result   string `json:"result,omitempty"`   // 工具结果（tool_call）
	Question string `json:"question,omitempty"` // 问题文本（ask_user）
	CallID   string `json:"callId,omitempty"`   // 工具调用 ID（tool_call/ask_user）
	Answer   string `json:"answer,omitempty"`   // 用户答案（ask_user）
	AskType  string   `json:"askType,omitempty"`  // 提问类型：text(默认) | single(单选) | multi(多选) | single-with-input(单选+自由输入)
	Options  []string `json:"options,omitempty"`  // 选项列表（ask_user 选择类用），如 ["是","否","不确定"]
}

// StoredMessage JSONL 中的一行。
type StoredMessage struct {
	Idx       int       `json:"idx"`       // 自增序号（0-based），用于分页游标
	Message   Message   `json:"message"`   // 完整 agent.Message
	Segments  []Segment `json:"segments"`  // 前端展示用 segments
	Timestamp string    `json:"timestamp"` // 写入时间 RFC3339
}

// ConversationMeta 对话元数据（存于 index.json）。
type ConversationMeta struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
	MsgCount      int    `json:"msgCount"`
	Summary       string `json:"summary,omitempty"`
	SummaryAt     string `json:"summaryAt,omitempty"`
	CtxStats      *Usage  `json:"ctxStats,omitempty"`
}

// MessageStore 对话消息持久化的唯一权威。
// 每对话一个 JSONL 文件 + 集中的 index.json 元数据。
// 并发安全：per-conv mutex（sync.Map[convID]*sync.Mutex）防同一文件并发 append 冲突，
// index.json 全局 mutex 防元数据读写冲突。
type MessageStore struct {
	root           string          // 工作区根路径
	convMu         sync.Map        // map[string]*sync.Mutex  key=convID，懒初始化
	indexMu        sync.Mutex      // index.json 全局锁
	persistedCount map[string]int  // convID → 已持久化的非 System 消息数（内存计数器，避免每轮读文件+全量遍历）
	pcMu           sync.RWMutex    // persistedCount 的并发锁
}

// NewMessageStore 创建消息存储器，初始化 .pair/conversations/ 目录。
func NewMessageStore(root string) *MessageStore {
	s := &MessageStore{root: root, persistedCount: make(map[string]int)}
	_ = os.MkdirAll(s.conversationsDir(), 0o755)
	return s
}

// SegmentsFromMessage 从 agent.Message 自动构建前端展示用 segments。
// 用于后端 diff-based 持久化时，为 loop.History 中的原始消息补充 segments，
// 使刷新页面后历史消息仍能看到思考、工具调用及结果等内容。
//
// 映射规则：
//   - RoleTool 消息 → tool_result segment（Content 作为 result，ToolCallID 关联调用）
//   - assistant 消息的 Reasoning → thinking segment
//   - assistant 消息的 ToolCalls → tool_call segments（含 name/argsRaw/callId）
//   - assistant/user 消息的 Content → content segment
//
// 若传入完整 history 和当前 index，会向前查找 tool result 填入对应 tool_call segment，
// 使 tool_call 与 result 在同一条 assistant 消息内完整呈现（前端无需渲染独立的 tool 消息）。
func SegmentsFromMessage(msg Message, hist []Message, idx int) []Segment {
	if msg.Role == RoleTool {
		return []Segment{{
			Type:   "tool_result",
			Result: msg.Content,
			CallID: msg.ToolCallID,
		}}
	}
	var segs []Segment
	if msg.Reasoning != "" {
		segs = append(segs, Segment{Type: "thinking", Content: msg.Reasoning})
	}
	for _, tc := range msg.ToolCalls {
		seg := Segment{
			Type:    "tool_call",
			Name:    tc.Function.Name,
			ArgsRaw: tc.Function.Arguments,
			CallID:  tc.ID,
		}
		// 向前查找对应的 tool result（RoleTool 消息，ToolCallID 匹配）
		if hist != nil {
			for j := idx + 1; j < len(hist); j++ {
				if hist[j].Role == RoleTool && hist[j].ToolCallID == tc.ID {
					seg.Result = hist[j].Content
					break
				}
			}
		}
		segs = append(segs, seg)
	}
	if msg.Content != "" {
		segs = append(segs, Segment{Type: "content", Content: msg.Content})
	}
	return segs
}

// conversationsDir 返回 {root}/.pair/conversations/ 路径。
func (s *MessageStore) conversationsDir() string {
	return filepath.Join(s.root, ".pair", "conversations")
}

// convFilePath 返回 {root}/.pair/conversations/{convID}.jsonl 路径。
func (s *MessageStore) convFilePath(convID string) string {
	return filepath.Join(s.conversationsDir(), convID+".jsonl")
}

// indexPath 返回 {root}/.pair/conversations/index.json 路径。
func (s *MessageStore) indexPath() string {
	return filepath.Join(s.conversationsDir(), "index.json")
}

// getConvMutex 懒初始化并返回 per-conv mutex（LoadOrStore 模式）。
func (s *MessageStore) getConvMutex(convID string) *sync.Mutex {
	v, _ := s.convMu.LoadOrStore(convID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// loadIndex 读取 index.json，文件不存在返回空切片。
func (s *MessageStore) loadIndex() ([]ConversationMeta, error) {
	data, err := os.ReadFile(s.indexPath())
	if err != nil {
		if os.IsNotExist(err) {
			return []ConversationMeta{}, nil
		}
		return nil, err
	}
	var metas []ConversationMeta
	if err := json.Unmarshal(data, &metas); err != nil {
		return nil, err
	}
	if metas == nil {
		metas = []ConversationMeta{}
	}
	return metas, nil
}

// saveIndex 写入 index.json（全量覆盖，MarshalIndent 缩进）。
func (s *MessageStore) saveIndex(metas []ConversationMeta) error {
	data, err := json.MarshalIndent(metas, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.indexPath(), data, 0o644)
}

// readJSONL 读取 JSONL 文件全部行并解码为 StoredMessage 切片。
// 解码失败的行跳过（容错）。文件不存在返回 nil, nil。
// 使用 10MB buffer（工具结果可能很长）。
func (s *MessageStore) readJSONL(convID string) ([]StoredMessage, error) {
	f, err := os.Open(s.convFilePath(convID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB buffer
	var msgs []StoredMessage
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var sm StoredMessage
		if err := json.Unmarshal(line, &sm); err != nil {
			continue // 容错：跳过解码失败的行
		}
		msgs = append(msgs, sm)
	}
	return msgs, scanner.Err()
}

// countJSONLLines 统计 JSONL 文件的非空行数。文件不存在返回 0。
func (s *MessageStore) countJSONLLines(convID string) (int, error) {
	f, err := os.Open(s.convFilePath(convID))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB buffer
	count := 0
	for scanner.Scan() {
		if len(bytes.TrimSpace(scanner.Bytes())) > 0 {
			count++
		}
	}
	return count, scanner.Err()
}

// CreateConversation 创建对话元数据（幂等：已存在则直接返回 nil）。
// 不创建空 JSONL 文件（按需在 AppendMessage 时创建）。
func (s *MessageStore) CreateConversation(convID, title, workspaceRoot string) error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("CreateConversation: 读取 index 失败: %w", err)
	}

	// 幂等：已存在则直接返回
	for _, m := range metas {
		if m.ID == convID {
			return nil
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	metas = append(metas, ConversationMeta{
		ID:            convID,
		Title:         title,
		CreatedAt:     now,
		UpdatedAt:     now,
		WorkspaceRoot: workspaceRoot,
		MsgCount:      0,
	})

	if err := s.saveIndex(metas); err != nil {
		return fmt.Errorf("CreateConversation: 写入 index 失败: %w", err)
	}
	return nil
}

// AppendMessage 追加一条消息到对话的 JSONL 文件，并更新 index.json。
// 获取 per-conv mutex 后：读取当前 JSONL 行数作为新 Idx（文件不存在则 Idx=0）。
func (s *MessageStore) AppendMessage(convID string, msg Message, segments []Segment) error {
	mu := s.getConvMutex(convID)
	mu.Lock()
	defer mu.Unlock()

	// 读取当前 JSONL 行数作为新 Idx
	count, err := s.countJSONLLines(convID)
	if err != nil {
		return fmt.Errorf("AppendMessage: 统计行数失败: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	sm := StoredMessage{
		Idx:       count,
		Message:   msg,
		Segments:  segments,
		Timestamp: now,
	}

	// JSON 编码 + 追加一行
	data, err := json.Marshal(sm)
	if err != nil {
		return fmt.Errorf("AppendMessage: JSON 编码失败: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(s.convFilePath(convID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("AppendMessage: 打开文件失败: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("AppendMessage: 写入失败: %w", err)
	}

	// 更新 index.json：对应 conv 的 MsgCount++、UpdatedAt=now（若不存在则创建 meta）
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("AppendMessage: 读取 index 失败: %w", err)
	}

	found := false
	for i := range metas {
		if metas[i].ID == convID {
			metas[i].MsgCount++
			metas[i].UpdatedAt = now
			found = true
			break
		}
	}
	if !found {
		// 若不存在则创建 meta
		metas = append(metas, ConversationMeta{
			ID:        convID,
			Title:     "新对话 " + time.Now().UTC().Format("15:04"),
			CreatedAt: now,
			UpdatedAt: now,
			MsgCount:  1,
		})
	}

	if err := s.saveIndex(metas); err != nil {
		return fmt.Errorf("AppendMessage: 写入 index 失败: %w", err)
	}

	// ★ 同步 persistedCount，使 AppendMessage（如预写入用户消息）后
	// PersistNewMessages 能准确跳过已持久化的消息，避免重写或误过滤。
	s.pcMu.Lock()
	s.persistedCount[convID] = count + 1
	s.pcMu.Unlock()

	return nil
}

// AppendUserMessage 便捷封装：追加一条用户消息。
func (s *MessageStore) AppendUserMessage(convID, content string) error {
	return s.AppendMessage(convID, Message{Role: RoleUser, Content: content}, nil)
}

// PersistNewMessages 原子性追加 hist 中尚未持久化的新消息到 store。
// 在 per-conv mutex 保护下完成：读当前行数 → 比较 diff → 逐条写入 → 更新 index。
//
// 消除 Ticker/EventDone/Session goroutine 三方并发写导致的 TOCTOU 重复
// （三者此前都用「先 LoadAll 计数、再 AppendMessage」的非原子方式，并发下产生重复）。
//
// 与 AppendMessage 的区别：
//   - AppendMessage 追加单条（由 handleChatSend 预写用户消息等调用）
//   - PersistNewMessages 批量追加且内部完成 diff 检查（供持久化 worker 调用）
func (s *MessageStore) PersistNewMessages(convID string, hist []Message) error {
	mu := s.getConvMutex(convID)
	mu.Lock()
	defer mu.Unlock()

	// 从内存计数器读取已持久化的非 System 消息数——避免每轮读文件+全量遍历
	s.pcMu.RLock()
	count := s.persistedCount[convID]
	s.pcMu.RUnlock()

	// 首次访问：内存计数器为 0 但文件可能已有内容（如 AppendMessage 预写的用户消息），
	// 从文件同步一次计数器，避免重写已有消息。
	if count == 0 {
		fileCount, err := s.countJSONLLines(convID)
		if err != nil {
			return fmt.Errorf("PersistNewMessages: 统计行数失败: %w", err)
		}
		if fileCount > 0 {
			s.pcMu.Lock()
			s.persistedCount[convID] = fileCount
			s.pcMu.Unlock()
			count = fileCount
		}
	}

	// 统计 hist 中非 System 消息数（System 由 Loop 动态构建，不应持久化）.
	// 只需计算超出 count 的部分，不用遍历整个 hist.
	if count > 0 {
		// 跳过前 count 条非 System 消息，直接从第 count 条开始遍历
		idx := 0 // idx 表示 hist 中的绝对索引
		nonSystemSeen := 0
		for idx < len(hist) && nonSystemSeen < count {
			if hist[idx].Role != RoleSystem {
				nonSystemSeen++
			}
			idx++
		}
		// hist 中非 System 消息数不大于 count → 无新消息
		if nonSystemSeen < count {
			return nil // 不会发生，但兜底
		}
		if idx >= len(hist) && nonSystemSeen == count {
			return nil // 无新消息
		}
	} else if count == 0 && len(hist) == 0 {
		return nil // 第一次调用但 hist 为空
	}

	// 直接用 append 模式写新增的消息（跳过已持久化的部分）
	s.pcMu.RLock()
	startCount := s.persistedCount[convID]
	s.pcMu.RUnlock()

	now := time.Now().UTC().Format(time.RFC3339)
	newCount := startCount
	nonSystemSeen := 0

	// 找到要写入的起始位置：跳过前 startCount 条非 System 消息
	startIdx := 0
	for startIdx < len(hist) {
		if hist[startIdx].Role != RoleSystem {
			if nonSystemSeen >= startCount {
				break
			}
			nonSystemSeen++
		}
		startIdx++
	}

	for i := startIdx; i < len(hist); i++ {
		m := hist[i]
		if m.Role == RoleSystem {
			continue
		}
		// ★ 跳过 RoleUser 消息：真实用户消息已在 AppendPersistedUserMessage
		// 预写入 store（计入 persistedCount），新出现的 User 消息是由自主模式内层
		// Loop 注入的子任务(subTask)，不应作为用户消息被持久化。
		if m.Role == RoleUser {
			continue
		}
		sm := StoredMessage{
			Idx:       newCount,
			Message:   m,
			Segments:  SegmentsFromMessage(m, hist, i),
			Timestamp: now,
		}
		data, err := json.Marshal(sm)
		if err != nil {
			return fmt.Errorf("PersistNewMessages: JSON 编码失败: %w", err)
		}
		data = append(data, '\n')

		f, err := os.OpenFile(s.convFilePath(convID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("PersistNewMessages: 打开文件失败: %w", err)
		}
		if _, err := f.Write(data); err != nil {
			f.Close()
			return fmt.Errorf("PersistNewMessages: 写入失败: %w", err)
		}
		f.Close()
		newCount++
	}

	// 更新内存计数器（在释放 conv mutex 前完成，保证并发可见性）
	s.pcMu.Lock()
	s.persistedCount[convID] = newCount
	s.pcMu.Unlock()

	// 更新 index.json：MsgsCount = newCount，UpdatedAt = now
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("PersistNewMessages: 读取 index 失败: %w", err)
	}

	found := false
	for i := range metas {
		if metas[i].ID == convID {
			metas[i].MsgCount = newCount
			metas[i].UpdatedAt = now
			found = true
			break
		}
	}
	if !found {
		metas = append(metas, ConversationMeta{
			ID:        convID,
			Title:     "新对话 " + time.Now().UTC().Format("15:04"),
			CreatedAt: now,
			UpdatedAt: now,
			MsgCount:  newCount,
		})
	}

	return s.saveIndex(metas)
}

// rebuildSegmentsIfMissing 为 segments 为空的消息重建 segments。
// 需要完整消息列表做 look-ahead（tool result 可能在后面的消息中）。
// 旧数据（修复前用 segments=nil 保存）通过此方法在读取时动态补全，
// 使刷新页面后历史消息仍能看到思考、工具调用及结果等内容。
func rebuildSegmentsIfMissing(msgs []StoredMessage) {
	if len(msgs) == 0 {
		return
	}
	// 先提取完整 Message 列表用于 look-ahead
	hist := make([]Message, len(msgs))
	for i, sm := range msgs {
		hist[i] = sm.Message
	}
	// 对 segments 为空的消息重建
	for i := range msgs {
		if len(msgs[i].Segments) == 0 {
			msgs[i].Segments = SegmentsFromMessage(msgs[i].Message, hist, i)
		}
	}
}

// LoadLatest 加载对话最新的消息。
// 返回：消息切片、总数、错误。
// limit <= 0 或 limit >= total 时返回全部；否则返回最后 limit 条（idx 升序）。
// 文件不存在返回空切片、total=0、nil。JSON 解码失败的行跳过（容错）。
// 对 segments 为空的旧数据会动态重建 segments（基于完整 history 做 look-ahead）。
func (s *MessageStore) LoadLatest(convID string, limit int) ([]StoredMessage, int, error) {
	msgs, err := s.readJSONL(convID)
	if err != nil {
		return nil, 0, err
	}
	if msgs == nil {
		msgs = []StoredMessage{}
	}
	// 在完整消息列表上重建 segments（look-ahead 需要 tool result）
	rebuildSegmentsIfMissing(msgs)
	total := len(msgs)
	if limit <= 0 || limit >= total {
		return msgs, total, nil
	}
	return msgs[total-limit:], total, nil
}

// LoadBefore 加载 idx < beforeIdx 的最新 limit 条消息（idx 升序）。
// limit <= 0 时默认 50。供前端向上分页 prepend。
// 对 segments 为空的旧数据会动态重建 segments（基于完整 history 做 look-ahead）。
func (s *MessageStore) LoadBefore(convID string, beforeIdx int, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	msgs, err := s.readJSONL(convID)
	if err != nil {
		return nil, err
	}
	// 先在完整消息列表上重建 segments（look-ahead 需要 tool result）
	rebuildSegmentsIfMissing(msgs)

	// 过滤 Idx < beforeIdx（msgs 已按 idx 升序）
	var filtered []StoredMessage
	for _, m := range msgs {
		if m.Idx < beforeIdx {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) == 0 {
		return []StoredMessage{}, nil
	}

	// 取最新 limit 条（filtered 已按 idx 升序，取末尾 limit 条）
	if len(filtered) <= limit {
		return filtered, nil
	}
	return filtered[len(filtered)-limit:], nil
}

// LoadAll 加载对话全部消息（仅 Message，不含 Segments），供 LLM 上下文恢复。
func (s *MessageStore) LoadAll(convID string) ([]Message, error) {
	msgs, err := s.readJSONL(convID)
	if err != nil {
		return nil, err
	}
	out := make([]Message, 0, len(msgs))
	for _, sm := range msgs {
		out = append(out, sm.Message)
	}
	return out, nil
}

// Count 返回对话 JSONL 行数。文件不存在返回 0。
func (s *MessageStore) Count(convID string) (int, error) {
	return s.countJSONLLines(convID)
}

// DeleteConversation 删除对话：移除 JSONL 文件 + 从 index.json 移除 meta + 清理 per-conv mutex。
func (s *MessageStore) DeleteConversation(convID string) error {
	mu := s.getConvMutex(convID)
	mu.Lock()
	defer mu.Unlock()

	// 删除 JSONL 文件（忽略 IsNotExist）
	if err := os.Remove(s.convFilePath(convID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("DeleteConversation: 删除 JSONL 失败: %w", err)
	}

	// 从 index.json 移除该 conv meta
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("DeleteConversation: 读取 index 失败: %w", err)
	}

	newMetas := make([]ConversationMeta, 0, len(metas))
	for _, m := range metas {
		if m.ID != convID {
			newMetas = append(newMetas, m)
		}
	}

	if err := s.saveIndex(newMetas); err != nil {
		return fmt.Errorf("DeleteConversation: 写入 index 失败: %w", err)
	}

	// 清理 per-conv mutex（Delete from sync.Map）
	s.convMu.Delete(convID)
	return nil
}

// ListConversations 列出指定工作区的对话（按 UpdatedAt 倒序）。
// 兼容旧数据：WorkspaceRoot 为空视为属于传入的 workspaceRoot（一并返回）。
func (s *MessageStore) ListConversations(workspaceRoot string) ([]ConversationMeta, error) {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return nil, fmt.Errorf("ListConversations: 读取 index 失败: %w", err)
	}

	out := make([]ConversationMeta, 0, len(metas))
	for _, m := range metas {
		if m.WorkspaceRoot == workspaceRoot || m.WorkspaceRoot == "" {
			out = append(out, m)
		}
	}

	// 按 UpdatedAt 倒序
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt > out[j].UpdatedAt
	})

	return out, nil
}

// UpdateTitle 更新对话标题及 UpdatedAt。
func (s *MessageStore) UpdateTitle(convID, title string) error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("UpdateTitle: 读取 index 失败: %w", err)
	}

	for i := range metas {
		if metas[i].ID == convID {
			metas[i].Title = title
			metas[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			return s.saveIndex(metas)
		}
	}
	return nil // 不存在则无操作
}

// SetSummary 设置对话摘要及 SummaryAt、UpdatedAt。
func (s *MessageStore) SetSummary(convID, summary, summaryAt string) error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("SetSummary: 读取 index 失败: %w", err)
	}

	for i := range metas {
		if metas[i].ID == convID {
			metas[i].Summary = summary
			metas[i].SummaryAt = summaryAt
			metas[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			return s.saveIndex(metas)
		}
	}
	return nil
}

// GetConversation 查找对话元数据，返回副本。不存在返回 nil, nil。
func (s *MessageStore) GetConversation(convID string) (*ConversationMeta, error) {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return nil, fmt.Errorf("GetConversation: 读取 index 失败: %w", err)
	}

	for _, m := range metas {
		if m.ID == convID {
			cp := m // 返回副本
			return &cp, nil
		}
	}
	return nil, nil
}

// SetCtxStats 更新对话的上下文 token 统计及 UpdatedAt。
func (s *MessageStore) SetCtxStats(convID string, stats *Usage) error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("SetCtxStats: 读取 index 失败: %w", err)
	}

	for i := range metas {
		if metas[i].ID == convID {
			metas[i].CtxStats = stats
			metas[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			return s.saveIndex(metas)
		}
	}
	return nil
}

// ─── 旧格式迁移 ───

// legacyMessage conversations.json 中的简化消息格式（无 ToolCalls/Reasoning）。
type legacyMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// legacyConversation conversations.json 的对话结构。
type legacyConversation struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	CreatedAt     string          `json:"createdAt"`
	UpdatedAt     string          `json:"updatedAt"`
	WorkspaceRoot string          `json:"workspaceRoot"`
	Messages      []legacyMessage `json:"messages,omitempty"`
	Summary       string          `json:"summary,omitempty"`
	SummaryAt     string          `json:"summaryAt,omitempty"`
	CtxStats      *Usage          `json:"ctxStats,omitempty"`
}

// legacyCachedSession history_cache.json 的缓存会话结构。
// 含完整 History（含 ToolCalls/Reasoning），供迁移时保留完整上下文。
// 同时兼容 Messages 字段（旧数据可能用此字段名）。
type legacyCachedSession struct {
	History  []Message `json:"history"`
	Messages []Message `json:"messages"`
}

// MigrateFromLegacy 从旧格式（conversations.json + history_cache.json）迁移到新格式。
//
// 迁移逻辑：
//   - 解析 conversations.json，对每个对话：
//   - 若 convID 已存在（之前迁移过），跳过整个对话
//   - 调 CreateConversation(id, title, workspaceRoot)
//   - 若 history_cache.json 中有对应 convID 的 History，用 History 中的 Message 追加（segments=nil）
//   - 否则用 conversations.json 的简化 messages 重建（无 ToolCalls/Reasoning）
//   - 同时设置 summary/summaryAt（若有）、ctxStats（若有）
//
// 迁移成功后将旧文件重命名为 .bak（不删除，防回滚）。若 .bak 已存在则覆盖。
func (s *MessageStore) MigrateFromLegacy(conversationsJSONPath, historyCacheJSONPath string) error {
	// 1. 解析 conversations.json
	convData, err := os.ReadFile(conversationsJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 无旧文件，无需迁移
		}
		return fmt.Errorf("migrate: 读取 conversations.json 失败: %w", err)
	}
	var legacyConvs []legacyConversation
	if err := json.Unmarshal(convData, &legacyConvs); err != nil {
		return fmt.Errorf("migrate: 解析 conversations.json 失败: %w", err)
	}

	// 2. 解析 history_cache.json（可选，文件不存在则跳过）
	var historyCache map[string]*legacyCachedSession
	if hcData, err := os.ReadFile(historyCacheJSONPath); err == nil {
		_ = json.Unmarshal(hcData, &historyCache)
	}

	// 3. 逐个迁移对话
	for _, lc := range legacyConvs {
		// 若 convID 已存在（之前迁移过），跳过整个对话，避免重复
		existing, err := s.GetConversation(lc.ID)
		if err != nil {
			return fmt.Errorf("migrate: 查询对话 %s 失败: %w", lc.ID, err)
		}
		if existing != nil {
			continue
		}

		// 创建对话元数据
		if err := s.CreateConversation(lc.ID, lc.Title, lc.WorkspaceRoot); err != nil {
			return fmt.Errorf("migrate: 创建对话 %s 失败: %w", lc.ID, err)
		}

		// 追加消息：优先用 history_cache 的完整 History（含 ToolCalls/Reasoning）
		usedHistory := false
		if cached, ok := historyCache[lc.ID]; ok && cached != nil {
			var msgs []Message
			if len(cached.History) > 0 {
				msgs = cached.History
			} else if len(cached.Messages) > 0 {
				// 兼容旧数据可能用 Messages 字段
				msgs = cached.Messages
			}
			if len(msgs) > 0 {
				for _, msg := range msgs {
					if err := s.AppendMessage(lc.ID, msg, nil); err != nil {
						return fmt.Errorf("migrate: 追加消息失败 (conv=%s): %w", lc.ID, err)
					}
				}
				usedHistory = true
			}
		}

		// 若未用 history_cache，则用 conversations.json 的简化 messages 重建
		if !usedHistory {
			for _, lm := range lc.Messages {
				msg := Message{
					Role:    Role(lm.Role),
					Content: lm.Content,
				}
				if err := s.AppendMessage(lc.ID, msg, nil); err != nil {
					return fmt.Errorf("migrate: 追加消息失败 (conv=%s): %w", lc.ID, err)
				}
			}
		}

		// 设置摘要（若有）
		if lc.Summary != "" || lc.SummaryAt != "" {
			if err := s.SetSummary(lc.ID, lc.Summary, lc.SummaryAt); err != nil {
				return fmt.Errorf("migrate: 设置摘要失败 (conv=%s): %w", lc.ID, err)
			}
		}

		// 设置上下文统计（若有）
		if lc.CtxStats != nil {
			if err := s.SetCtxStats(lc.ID, lc.CtxStats); err != nil {
				return fmt.Errorf("migrate: 设置上下文统计失败 (conv=%s): %w", lc.ID, err)
			}
		}
	}

	// 4. 将旧文件重命名为 .bak（不删除，防回滚）。若 .bak 已存在则覆盖。
	if err := renameToBak(conversationsJSONPath); err != nil {
		return fmt.Errorf("migrate: 重命名 conversations.json 失败: %w", err)
	}
	// history_cache.json 可能不存在，忽略 IsNotExist 错误
	if err := renameToBak(historyCacheJSONPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("migrate: 重命名 history_cache.json 失败: %w", err)
	}

	return nil
}

// renameToBak 将文件重命名为 .bak。若 .bak 已存在则先删除再重命名（覆盖）。
// 源文件不存在时返回 IsNotExist 错误。
func renameToBak(src string) error {
	if _, err := os.Stat(src); err != nil {
		return err
	}
	bakPath := src + ".bak"
	// 若 .bak 已存在则先删除（覆盖）
	_ = os.Remove(bakPath)
	return os.Rename(src, bakPath)
}
