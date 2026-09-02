// MessageStore — 对话消息持久化的活动实现（每对话一个 JSONL 文件 + index.json 元数据）。
// 落盘单位为带事件语义标注的 StoredMessage（EventType/Turn/Step，见下方类型定义），
// 使 JSONL 具备可重建的 turn/step 事件流结构（对齐 session event 思路）。
package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// SegmentQuestion 多问题提问的单条问题（Round3 ⑤ ask_user 多问题）。
type SegmentQuestion struct {
	ID          string   `json:"id"`                    // 问题 ID（回答回灌用）
	Question    string   `json:"question"`              // 问题文本
	Options     []string `json:"options,omitempty"`     // 选项（single/multi 类）
	MultiSelect bool     `json:"multiSelect,omitempty"` // 是否多选
}

// SegmentAnswer 多问题提问的单条回答（Round3 ⑤）。
type SegmentAnswer struct {
	ID     string `json:"id"`
	Answer string `json:"answer"`
}

// Segment 前端展示用的消息分段（thinking/content/tool_call/tool_result/ask_user 等）。
type Segment struct {
	Type     string   `json:"type"`               // thinking | content | tool_call | tool_result | ask_user
	Content  string   `json:"content,omitempty"`  // 文本内容（thinking/content/tool_result）
	Name     string   `json:"name,omitempty"`     // 工具名（tool_call）
	ArgsRaw  string   `json:"argsRaw,omitempty"`  // 工具参数 JSON 字符串（tool_call）
	Result   string   `json:"result,omitempty"`   // 工具结果（tool_call）
	Question string   `json:"question,omitempty"` // 问题文本（ask_user 单问题路径）
	CallID   string   `json:"callId,omitempty"`   // 工具调用 ID（tool_call/ask_user）
	Answer   string   `json:"answer,omitempty"`   // 用户答案（ask_user 单问题路径）
	AskType  string   `json:"askType,omitempty"`  // 提问类型：text(默认) | single(单选) | multi(多选) | single-with-input(单选+自由输入)
	Options  []string `json:"options,omitempty"`  // 选项列表（ask_user 选择类用），如 ["是","否","不确定"]
	// ★ Round3 ⑤ 多问题：Questions/Answers 数组（旧 Question/Options/Answer 字段
	//   保留序列化兼容——旧前端/历史数据不受影响）
	Questions []SegmentQuestion `json:"questions,omitempty"`
	Answers   []SegmentAnswer   `json:"answers,omitempty"`
}

// StoredMessage JSONL 中的一行。
type StoredMessage struct {
	Idx       int       `json:"idx"`       // 自增序号（0-based），用于分页游标
	Message   Message   `json:"message"`   // 完整 agent.Message
	Segments  []Segment `json:"segments"`  // 前端展示用 segments
	Timestamp string    `json:"timestamp"` // 写入时间 RFC3339

	// ── 事件语义标注（对齐 session event 词汇，2026-08-15）──
	// 落盘单位仍是 Message（兼容前端协议），但每条消息附事件元数据，
	// 使 JSONL 从「纯消息数组」升级为「带 turn/step 结构的事件流」，
	// 可重建 deepseek 式 surface 投影（user/message、assistant/message、tool/result）。
	// EventType 事件类型（surface 三类，见 EventType* 常量）。
	// 旧数据（本字段为空）读取时按 Message.Role 推导。
	EventType string `json:"eventType,omitempty"`
	// Turn 所属轮次（1 基；agentloop 语义：一次 Run = 一个 turn，落盘 user 消息递增）。
	// 0 = 旧数据未标注。
	Turn int `json:"turn,omitempty"`
	// Step 所属步骤（1 基；agentloop 语义：每次 LLM 调用 + 工具执行 = 一个 step，
	// 落盘 assistant 消息递增，其后 tool/result 归并同 step）。0 = 旧数据未标注。
	Step int `json:"step,omitempty"`
}

// 事件类型常量（对齐 session event 词汇的 surface 子集；
// turn/end、assistant/chunk 等非 surface 事件不单独落盘——消息流本身即
// surface 投影，turn/step 结构由 Turn/Step 字段重建）。
const (
	// EventTypeUserMessage 用户消息（对应 session event user/message）。
	EventTypeUserMessage = "user/message"
	// EventTypeAssistantMessage 助手消息（含 tool-call 块，对应 assistant/message）。
	EventTypeAssistantMessage = "assistant/message"
	// EventTypeToolResult 工具结果（对应 tool/result）。
	EventTypeToolResult = "tool/result"
)

// deriveEventType 按消息角色推导事件类型（对齐 surface 投影规则：
// 只有 user/message、assistant/message、tool/result 三类进入模型可见消息；
// system 不落盘、其余角色不投影）。
func deriveEventType(m Message) string {
	switch m.Role {
	case RoleUser:
		return EventTypeUserMessage
	case RoleAssistant:
		return EventTypeAssistantMessage
	case RoleTool:
		return EventTypeToolResult
	default:
		return ""
	}
}

// turnStepFor 按消息序列推导每条消息的 (turn, step) 标注（与输入等长的 [turn,step] 数组）。
// 对齐 agentloop 语义：
//   - 一次 Run = 一个 turn：落盘的 user 消息（每轮任务消息）递增 turn，并重置 step；
//   - 每次 LLM 调用 + 工具执行 = 一个 step：assistant 消息递增 step；
//   - tool/result 归并到前一条 assistant 的 step（不递增）。
//
// 注：nudge/背景注入等 ephemeral 消息不落盘，故落盘序列中 user 消息 = 每轮一个，
// turn 计数与实际对话轮次一致。
func turnStepFor(msgs []Message) [][2]int {
	out := make([][2]int, len(msgs))
	turn, step := 0, 0
	for i, m := range msgs {
		switch m.Role {
		case RoleUser:
			// ★ 背景上下文快照（【背景上下文·非当前任务】前缀）不是用户任务轮次：
			//   它是循环同步进消息流的背景信息（历史摘要/状态/记忆/知识库），
			//   不递增 turn/step——避免 turnStepFor 把它推为新一轮任务（EventType
			//   仍是 user/message，模型可见；仅 turn/step 语义标注忽略）。
			if strings.HasPrefix(m.Content, backgroundCtxMarker) {
				break
			}
			turn++
			step = 0
		case RoleAssistant:
			step++
		}
		out[i] = [2]int{turn, step}
	}
	return out
}

// annotateStoredEvents 给 stored 列表原地填充事件语义标注：
//   - EventType：按角色推导（deriveEventType，surface 三类）；
//   - Turn/Step：按消息序列推导（turnStepFor，重建 turn/step 事件流结构）。
func annotateStoredEvents(stored []StoredMessage) {
	if len(stored) == 0 {
		return
	}
	msgs := make([]Message, len(stored))
	for i := range stored {
		msgs[i] = stored[i].Message
	}
	notes := turnStepFor(msgs)
	for i := range stored {
		stored[i].EventType = deriveEventType(stored[i].Message)
		stored[i].Turn = notes[i][0]
		stored[i].Step = notes[i][1]
	}
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
	CtxStats      *Usage `json:"ctxStats,omitempty"`
	// ★ 2026-08-31 会话级模型路由：模型选择只作用于本会话（此前切模型写全局
	//   settings，导致所有历史对话的模型一起被改）。空=沿用全局默认配置。
	//   Provider=服务商名（models.json 的键），Model=执行模型名。
	//   ★ 2026-09-03 Preset=所选 AI 配置名（ai-presets.json 的键）——装配时按配置名
	//   整套展开（含该配置的 Key）；仅存 provider/model 时 Key 只能按服务商猜测
	//   （同服务商多配置会取错 Key）。空=未设（回落全局/按服务商匹配）。
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Preset   string `json:"preset,omitempty"`
	// ★ 2026-08-31 会话级审核模式：会话内切换审核模式只影响本会话并持久化
	//   （空=沿用全局/工作区配置；此前切换写全局 settings 串扰其他会话且重启丢失）。
	ReviewMode string `json:"reviewMode,omitempty"`
	// Interrupted 标记该对话上次运行是否异常中断（LLM API 错误/panic/崩溃等非用户停止）。
	// 前端据此展示"未完成，可继续"引导，用户在原对话继续即可恢复上下文与任务进度。
	Interrupted bool `json:"interrupted,omitempty"`
}

// MessageStore 对话消息持久化的唯一权威。
// 每对话一个 JSONL 文件 + 集中的 index.json 元数据。
// 并发安全：per-conv mutex（sync.Map[convID]*sync.Mutex）防同一文件并发 append 冲突，
// index.json 全局 mutex 防元数据读写冲突。
type MessageStore struct {
	root           string         // 工作区根路径
	convMu         sync.Map       // map[string]*sync.Mutex  key=convID，懒初始化
	indexMu        sync.Mutex     // index.json 全局锁
	persistedCount map[string]int // convID → 已持久化的非 System 消息数（内存计数器，避免每轮读文件+全量遍历）
	pcMu           sync.RWMutex   // persistedCount 的并发锁
}

// NewMessageStore 创建消息存储器，初始化 .pair/conversations/ 目录。
func NewMessageStore(root string) *MessageStore {
	s := &MessageStore{root: root, persistedCount: make(map[string]int)}
	_ = os.MkdirAll(s.conversationsDir(), 0o755)
	s.CleanupTempFiles()
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
	// ★ 正文（content）必须在工具调用（tool_call）之前，使刷新后顺序与实时流一致。
	if msg.Content != "" {
		segs = append(segs, Segment{Type: "content", Content: msg.Content})
	}
	for _, tc := range msg.ToolCalls {
		name := tc.Function.Name
		// ★ 委托工具（已随多角色模型移除 2026-08-16）不再生成 tool_call segment——
		//   保留 ask_user 等交互式工具的 segment 逻辑
		if name == "ask_user" {
			// ★ ask_user 生成独立交互式 segment（Round3 ⑤：多问题 questions 数组）
			question, askType, options, questions := parseAskArgsV2(tc.Function.Arguments)
			seg := Segment{
				Type:      "ask_user",
				Question:  question,
				AskType:   askType,
				Options:   options,
				Questions: questions,
				CallID:    tc.ID,
			}
			if hist != nil {
				for j := idx + 1; j < len(hist); j++ {
					if hist[j].Role == RoleTool && hist[j].ToolCallID == tc.ID {
						seg.Answer, seg.Answers = parseAskResultV2(hist[j].Content)
						break
					}
				}
			}
			segs = append(segs, seg)
			continue
		}
		// 普通工具调用：生成 tool_call segment
		seg := Segment{
			Type:    "tool_call",
			Name:    name,
			ArgsRaw: tc.Function.Arguments,
			CallID:  tc.ID,
		}
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
	return segs
}

// MergeConsecutiveAssistants 合并连续相邻的 assistant 消息为一条。
// 将各条 assistant 的 segments 按出现顺序拼接（保留 thinking/content/tool_call/ask_user 时序）。
// 用于 API 返回前，使前端将一个 agent 回复的多轮 LLM 迭代显示为单个气泡。
// 非 assistant 消息（user/system）保持不变，作为合并的天然边界。
// ★ RoleTool 消息跳过（不打断合并，也不输出到结果）——其工具执行结果已通过
//
//	SegmentsFromMessage 合并到对应 assistant 消息的 tool_call segment 中。
func MergeConsecutiveAssistants(msgs []StoredMessage) []StoredMessage {
	if len(msgs) == 0 {
		return msgs
	}
	out := make([]StoredMessage, 0, len(msgs))
	var pending *StoredMessage
	for i := range msgs {
		m := msgs[i]
		if m.Message.Role == RoleAssistant {
			if pending == nil {
				pending = &StoredMessage{
					Idx:       m.Idx,
					Message:   m.Message,
					Segments:  append([]Segment{}, m.Segments...),
					Timestamp: m.Timestamp,
				}
			} else {
				pending.Segments = append(pending.Segments, m.Segments...)
				if m.Message.Content != "" {
					if pending.Message.Content != "" {
						pending.Message.Content += "\n\n" + m.Message.Content
					} else {
						pending.Message.Content = m.Message.Content
					}
				}
				if len(m.Message.ToolCalls) > 0 {
					pending.Message.ToolCalls = append(pending.Message.ToolCalls, m.Message.ToolCalls...)
				}
				if m.Message.Reasoning != "" {
					if pending.Message.Reasoning != "" {
						pending.Message.Reasoning += "\n\n" + m.Message.Reasoning
					} else {
						pending.Message.Reasoning = m.Message.Reasoning
					}
				}
			}
		} else if m.Message.Role == RoleTool {
			// ★ Tool 消息跳过：不打断合并也不输出，其结果已通过 SegmentsFromMessage
			//   合并到前一条 assistant 的 tool_call segment 中。
			continue
		} else {
			if pending != nil {
				out = append(out, *pending)
				pending = nil
			}
			out = append(out, m)
		}
	}
	if pending != nil {
		out = append(out, *pending)
	}
	return out
}

// parseAskArgs 从问用户工具的 JSON arguments 中提取结构化字段。
// normalizeAskType 将模型可能生成的各种 askType 变体归一化为标准值：
// text | single | multi | single-with-input。
func normalizeAskType(raw string) string {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), "_", "-")) {
	case "single", "choice", "selected", "radio", "single-choice":
		return "single"
	case "multi", "multiple", "checkbox", "multi-choice", "multi-select":
		return "multi"
	case "single-with-input", "single-with-custom", "choice-with-input", "single-input":
		return "single-with-input"
	case "text", "input", "free-text", "string", "":
		return "text"
	default:
		// 未知变体：默认文本输入（保证输入框一定出现）
		return "text"
	}
}

func parseAskArgs(argsRaw string) (question, askType string, options []string) {
	question, askType, options, _ = parseAskArgsV2(argsRaw)
	return
}

// parseAskArgsV2 解析 ask_user 参数（Round3 ⑤）：questions 数组优先，
// 缺省回落单问题路径（question/askType/options）。
func parseAskArgsV2(argsRaw string) (question, askType string, options []string, questions []SegmentQuestion) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(argsRaw), &raw); err != nil {
		return argsRaw, "text", nil, nil
	}
	// 多问题路径：questions: [{id, question, options?, multi_select?}]
	if qs, ok := raw["questions"].([]any); ok && len(qs) > 0 {
		for _, it := range qs {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			q := SegmentQuestion{}
			q.ID, _ = m["id"].(string)
			q.Question, _ = m["question"].(string)
			q.MultiSelect, _ = m["multi_select"].(bool)
			if opts, ok := m["options"].([]any); ok {
				for _, o := range opts {
					if s, ok := o.(string); ok {
						q.Options = append(q.Options, s)
					}
				}
			}
			if q.Question != "" {
				questions = append(questions, q)
			}
		}
		if len(questions) > 0 {
			// 多问题：旧单问题字段不再填充（前端按 questions 渲染）
			return "", "text", nil, questions
		}
	}
	question, _ = raw["question"].(string)
	if question == "" {
		question = argsRaw
	}
	askType, _ = raw["askType"].(string)
	if askType == "" {
		// 容错：部分模型按描述生成 type 而非 askType
		if v, ok := raw["type"].(string); ok {
			askType = v
		}
	}
	askType = normalizeAskType(askType)
	if opts, ok := raw["options"].([]any); ok {
		for _, o := range opts {
			if s, ok := o.(string); ok {
				options = append(options, s)
			}
		}
	}
	return
}

// askQuestionsFromArgs 从已解析的工具参数取多问题列表（Round3 ⑤；无则 nil）。
func askQuestionsFromArgs(args map[string]any) []SegmentQuestion {
	raw, ok := args["questions"].([]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	var out []SegmentQuestion
	for _, it := range raw {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		q := SegmentQuestion{}
		q.ID, _ = m["id"].(string)
		q.Question, _ = m["question"].(string)
		q.MultiSelect, _ = m["multi_select"].(bool)
		if opts, ok := m["options"].([]any); ok {
			for _, o := range opts {
				if s, ok := o.(string); ok {
					q.Options = append(q.Options, s)
				}
			}
		}
		if q.Question != "" {
			out = append(out, q)
		}
	}
	return out
}

// validateAskAnswers 校验多问题回答数组（t4 F6）：回答 ID 必须全部属于问题 ID 集合。
func validateAskAnswers(qs []SegmentQuestion, answers []AskAnswer) error {
	known := make(map[string]bool, len(qs))
	for _, q := range qs {
		known[q.ID] = true
	}
	seen := make(map[string]bool, len(answers))
	for _, a := range answers {
		if a.ID == "" {
			return fmt.Errorf("ask_user(多问题)：回答缺少问题 ID（应为 %d 个问题的 answers 数组）", len(qs))
		}
		if !known[a.ID] {
			return fmt.Errorf("ask_user(多问题)：回答 ID %q 不属于问题集合（%d 个问题）——回答与提问错配", a.ID, len(qs))
		}
		if seen[a.ID] {
			return fmt.Errorf("ask_user(多问题)：问题 %q 收到重复回答", a.ID)
		}
		seen[a.ID] = true
	}
	for _, q := range qs {
		if !seen[q.ID] {
			return fmt.Errorf("ask_user(多问题)：问题 %q 缺回答（应 %d 条 answers）", q.ID, len(qs))
		}
	}
	return nil
}

// parseAskResultV2 解析 ask_user 工具结果回灌：多问题（answers JSON 数组）优先，
// 缺省回落单问题（整段文本即答案）。
func parseAskResultV2(content string) (answer string, answers []SegmentAnswer) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "{") {
		var raw struct {
			Answers []SegmentAnswer `json:"answers"`
		}
		if err := json.Unmarshal([]byte(content), &raw); err == nil && len(raw.Answers) > 0 {
			return "", raw.Answers
		}
	}
	return content, nil
}

// conversationsDir 返回 {root}/.pair/conversations/ 路径。
func (s *MessageStore) conversationsDir() string {
	return filepath.Join(s.root, ".pair", "conversations")
}

// CleanupTempFiles 清理因异常中断残留的 .tmp 和 .bak 文件。
// - 删除孤立的 .tmp 文件（对应的 .jsonl 已存在或不存在）
// - 若 .jsonl 不存在而 .bak 存在，将 .bak 恢复为 .jsonl
// 在 MessageStore 初始化后调用。
func (s *MessageStore) CleanupTempFiles() {
	dir := s.conversationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".tmp") {
			// 删除所有残留 .tmp 文件（它们是不完整的写入）
			os.Remove(filepath.Join(dir, name))
		} else if strings.HasSuffix(name, ".bak") {
			jsonlPath := filepath.Join(dir, strings.TrimSuffix(name, ".bak"))
			bakPath := filepath.Join(dir, name)
			if _, err := os.Stat(jsonlPath); os.IsNotExist(err) {
				// .jsonl 不存在但 .bak 存在 → 恢复
				os.Rename(bakPath, jsonlPath)
				fmt.Printf("[MessageStore] 从 .bak 恢复对话文件: %s\n", jsonlPath)
			} else {
				// .jsonl 已存在，.bak 可以安全删除
				os.Remove(bakPath)
			}
		}
	}
}

// convFilePath 返回 {root}/.pair/conversations/{convID}.jsonl 路径。
func (s *MessageStore) convFilePath(convID string) string {
	return filepath.Join(s.conversationsDir(), convID+".jsonl")
}

// convSummariesPath 返回 {root}/.pair/conversations/{convID}.summaries.json 路径。
func (s *MessageStore) convSummariesPath(convID string) string {
	return filepath.Join(s.conversationsDir(), convID+".summaries.json")
}

// SaveCompressedSummaries 持久化压缩摘要列表到对话的 summaries 文件。
// 格式：JSON 数组 ["摘要1", "摘要2", ...]，单行写入。
// summaries 为空时删除文件（幂等）。
func (s *MessageStore) SaveCompressedSummaries(convID string, summaries []string) error {
	if len(summaries) == 0 {
		_ = os.Remove(s.convSummariesPath(convID))
		return nil
	}
	data, err := json.Marshal(summaries)
	if err != nil {
		return fmt.Errorf("SaveCompressedSummaries: JSON 编码失败: %w", err)
	}
	return os.WriteFile(s.convSummariesPath(convID), data, 0o644)
}

// LoadCompressedSummaries 从 summaries 文件加载压缩摘要列表。
// 文件不存在或解码失败返回空切片。
func (s *MessageStore) LoadCompressedSummaries(convID string) ([]string, error) {
	data, err := os.ReadFile(s.convSummariesPath(convID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("LoadCompressedSummaries: 读取失败: %w", err)
	}
	var summaries []string
	if err := json.Unmarshal(data, &summaries); err != nil {
		return nil, nil // 容错：损坏文件返回空
	}
	return summaries, nil
}

// indexPath 返回 {root}/.pair/conversations/index.json 路径。
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

// readJSONLBak 尝试从 .bak 备份文件读取 JSONL 数据（崩溃恢复用）。
func (s *MessageStore) readJSONLBak(convID string) ([]StoredMessage, error) {
	bakPath := s.convFilePath(convID) + ".bak"
	f, err := os.Open(bakPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	var msgs []StoredMessage
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var sm StoredMessage
		if err := json.Unmarshal(line, &sm); err != nil {
			continue
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
		// 事件类型按角色推导；Turn/Step 单条追加时无法得知上下文
		// （后续 PersistNewMessages 全量重写会补齐标注）——读取侧按 Role 回退即可。
		EventType: deriveEventType(msg),
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

	// 检查是否需要归档
	if count+1 >= ArchiveThreshold {
		// ★ 必须先关闭写句柄：checkAndArchive 会用 Rename 原子替换主文件，
		//   Windows 不允许重命名被打开的文件（f 由本函数 OpenFile 持有，
		//   defer f.Close() 要到函数返回才执行，此时仍占用 → Rename 必失败、
		//   错误被 _ 吞掉导致归档静默失效）。
		_ = f.Close()
		_ = s.checkAndArchive(convID)
	}

	return nil
}

// AppendUserMessage 便捷封装：追加一条用户消息。
func (s *MessageStore) AppendUserMessage(convID, content string) error {
	return s.AppendMessage(convID, Message{Role: RoleUser, Content: content}, nil)
}

// AppendUserMessageWithImages 便捷封装：追加一条带图片的用户消息（★ 2026-08-21 多模态）。
func (s *MessageStore) AppendUserMessageWithImages(convID, content string, images []ImagePart) error {
	return s.AppendMessage(convID, Message{Role: RoleUser, Content: content, Images: images}, nil)
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

	// 从 hist 中提取所有非 System 消息，全量写回 JSONL。
	// 历史不压缩、不做 diff 计数，每次全量覆盖保证一致性。
	now := time.Now().UTC().Format(time.RFC3339)

	// 收集要写入的消息（跳过 System），同时记录每个消息在原始 hist 中的索引。
	type msgWithIdx struct {
		msg  Message
		hIdx int // 在原始 hist 中的位置
	}
	var msgs []msgWithIdx
	for hi, m := range hist {
		if m.Role != RoleSystem {
			msgs = append(msgs, msgWithIdx{msg: m, hIdx: hi})
		}
	}

	// 先构建 StoredMessage 列表（不合并连续 assistant），
	// 保持每条 assistant 消息独立存储，时序结构与事件流一致。
	// 合并操作仅在 API 读取时由 web_server ConvMerge 按需执行。
	var stored []StoredMessage
	for i, mi := range msgs {
		sm := StoredMessage{Idx: i, Message: mi.msg, Timestamp: now}
		sm.Segments = SegmentsFromMessage(mi.msg, hist, mi.hIdx)
		stored = append(stored, sm)
	}
	for i := range stored {
		stored[i].Idx = i
	}

	// ★ 事件语义标注：EventType + Turn/Step（对齐 surface 投影）。
	// 规则见 turnStepFor/annotateStoredEvents——纯从消息序列推导，无需 Loop 传递状态，
	// 使 JSONL 具备可重建的 turn/step 事件流结构。
	annotateStoredEvents(stored)

	tmpPath := s.convFilePath(convID) + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("PersistNewMessages: 创建临时文件失败: %w", err)
	}
	enc := json.NewEncoder(out)
	for _, sm := range stored {
		if err := enc.Encode(sm); err != nil {
			out.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("PersistNewMessages: JSON 编码失败: %w", err)
		}
	}
	out.Close()

	// ★ 崩溃安全原子替换：先备份旧文件再替换，崩溃后可恢复。
	// Windows 不允许 Rename 覆盖已有文件，故用三步法：
	//   1. Rename dest → dest.bak（备份旧文件）
	//   2. Rename tmp → dest（新文件就位）
	//   3. Remove .bak（新文件确认无误后删备份）
	// 若步骤 1-2 间崩溃 → dest 不存在但 .bak 存在，LoadAll 可从 .bak 恢复。
	// 若步骤 2-3 间崩溃 → dest 已就位，.bak 可被后续写入或启动时清理。
	destPath := s.convFilePath(convID)
	bakPath := destPath + ".bak"
	// 步骤 1：备份旧文件（旧文件不存在时忽略错误）
	os.Rename(destPath, bakPath)
	// 步骤 2：新文件就位
	if err := os.Rename(tmpPath, destPath); err != nil {
		// 替换失败：尝试恢复备份
		os.Rename(bakPath, destPath)
		os.Remove(tmpPath)
		return fmt.Errorf("PersistNewMessages: 替换文件失败: %w", err)
	}
	// 步骤 3：新文件确认无误，删除备份
	os.Remove(bakPath)

	// 更新 index.json
	s.indexMu.Lock()
	metas, err := s.loadIndex()
	if err == nil {
		for i := range metas {
			if metas[i].ID == convID {
				metas[i].MsgCount = len(stored)
				metas[i].UpdatedAt = now
				break
			}
		}
		_ = s.saveIndex(metas)
	}
	s.indexMu.Unlock()

	// 更新内存计数器（仅用于 GetPersistedCount 兼容）
	s.pcMu.Lock()
	s.persistedCount[convID] = len(stored)
	s.pcMu.Unlock()

	return nil
}

// rebuildSegmentsIfMissing

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
	// ★ 仅当 segments 为空或缺少 tool_result 时才重建。
	// 若 JSONL 中已有正确 segments（含 tool_result），直接保留——避免重建时
	// 将已合并的 reasoning/content 压缩成单个 thinking/content segment，
	// 丢失各轮次间的时序分隔。
	needsRebuild := false
	for i := range msgs {
		if len(msgs[i].Segments) == 0 {
			needsRebuild = true
			break
		}
		// 检查是否有 tool_call segment 缺少 result
		for _, seg := range msgs[i].Segments {
			if seg.Type == "tool_call" && seg.Result == "" {
				needsRebuild = true
				break
			}
		}
		if needsRebuild {
			break
		}
	}
	if needsRebuild {
		for i := range msgs {
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

// LoadLatestForDisplay 供前端消息展示：在完整列表上合并（tool 行并入 segments、
// 连续 assistant 合并）后再按合并条数取末尾 limit 条。
// ★ 2026-08-22 性能修复：此前按原始行取 limit——JSONL 中一行 = 一次迭代/tool 行，
//
//	50 行经合并后可能只剩 1~5 条超大消息（一个 run 的全部段），前端 fillViewport
//	每轮只前进 1 条（每条 100~400KB），长对话首屏/滚动加载极慢。返回合并后的
//	limit 条（≥1）、total 保持原始行数。
func (s *MessageStore) LoadLatestForDisplay(convID string, limit int) ([]StoredMessage, int, error) {
	msgs, err := s.readJSONL(convID)
	if err != nil {
		return nil, 0, err
	}
	if msgs == nil {
		msgs = []StoredMessage{}
	}
	rebuildSegmentsIfMissing(msgs)
	total := len(msgs)
	merged := MergeConsecutiveAssistants(msgs)
	if limit <= 0 || limit >= len(merged) {
		return merged, total, nil
	}
	return merged[len(merged)-limit:], total, nil
}

// LoadBeforeForDisplay 供前端向上分页：全量合并后过滤 Idx < beforeIdx 的条目，
// 取末尾 limit 条（idx 升序）。语义对应 LoadLatestForDisplay。
func (s *MessageStore) LoadBeforeForDisplay(convID string, beforeIdx int, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	msgs, err := s.readJSONL(convID)
	if err != nil {
		return nil, err
	}
	rebuildSegmentsIfMissing(msgs)
	merged := MergeConsecutiveAssistants(msgs)
	var filtered []StoredMessage
	for _, m := range merged {
		if m.Idx < beforeIdx {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) == 0 {
		return []StoredMessage{}, nil
	}
	if len(filtered) <= limit {
		return filtered, nil
	}
	return filtered[len(filtered)-limit:], nil
}

// MergeLastAssistantRun 合并 JSONL 末尾最近一个用户→助手的连续助理条目。
// OnBatchPersist 每轮迭代写一条助理消息，导致同一 run 的多个迭代在 JSONL 中
// 是独立行。此方法在 Loop 完成后调用，扫描文件末尾从最近一条 user 消息后的
// 连续 assistant 行，合并为一条（tool 行保留不动）。
// 结果：前端加载后一个 run 只看到 1 条 assistant 消息。
func (s *MessageStore) MergeLastAssistantRun(convID string) error {
	mu := s.getConvMutex(convID)
	mu.Lock()
	defer mu.Unlock()

	stored, err := s.readJSONL(convID)
	if err != nil {
		return fmt.Errorf("MergeLastAssistantRun: 读取失败: %w", err)
	}
	if len(stored) == 0 {
		return nil
	}

	// 从末尾找最后一个 user
	lastUser := -1
	for i := len(stored) - 1; i >= 0; i-- {
		if stored[i].Message.Role == RoleUser {
			lastUser = i
			break
		}
	}
	if lastUser < 0 || lastUser >= len(stored)-1 {
		return nil
	}

	// 收集 user 后的连续 assistant 索引
	assistantIdxs := []int{}
	for i := lastUser + 1; i < len(stored); i++ {
		r := stored[i].Message.Role
		if r == RoleAssistant {
			assistantIdxs = append(assistantIdxs, i)
		} else if r == RoleTool {
			continue
		} else {
			break
		}
	}
	if len(assistantIdxs) <= 1 {
		return nil
	}

	// 合并
	base := stored[assistantIdxs[0]]
	for idx := 1; idx < len(assistantIdxs); idx++ {
		next := stored[assistantIdxs[idx]]
		base.Segments = append(base.Segments, next.Segments...)
		if next.Message.Content != "" {
			if base.Message.Content == "" {
				base.Message.Content = next.Message.Content
			} else {
				base.Message.Content += "\n" + next.Message.Content
			}
		}
		if len(next.Message.ToolCalls) > 0 {
			base.Message.ToolCalls = append(base.Message.ToolCalls, next.Message.ToolCalls...)
		}
	}

	// 重建结果切片
	var result []StoredMessage
	result = append(result, stored[:lastUser+1]...)
	result = append(result, base)
	for i := lastUser + 1; i < len(stored); i++ {
		isAssistant := false
		for _, ai := range assistantIdxs {
			if i == ai {
				isAssistant = true
				break
			}
		}
		if !isAssistant {
			result = append(result, stored[i])
		}
	}

	// 重写索引
	for i := range result {
		result[i].Idx = i
	}

	// 写回（JSONL 格式：每行一个 JSON 对象）
	out, err := os.Create(s.convFilePath(convID))
	if err != nil {
		return fmt.Errorf("MergeLastAssistantRun: 创建文件失败: %w", err)
	}
	defer out.Close()
	enc := json.NewEncoder(out)
	for _, sm := range result {
		if err := enc.Encode(sm); err != nil {
			return fmt.Errorf("MergeLastAssistantRun: JSON 编码失败: %w", err)
		}
	}

	// 更新计数器
	s.pcMu.Lock()
	s.persistedCount[convID] = len(result)
	s.pcMu.Unlock()

	return nil
}

// mergeAssistantRun 合并 stored 中从 startIdx 开始的连续 assistant 条目，
// 返回合并后的结果切片和下一个要处理的位置。
// 供 MergeAllAssistantRuns 内部使用。
func mergeAssistantRun(stored []StoredMessage, startIdx int) ([]StoredMessage, int) {
	// 收集从 startIdx 开始的连续 assistant 索引
	assistantIdxs := []int{}
	for i := startIdx; i < len(stored); i++ {
		r := stored[i].Message.Role
		if r == RoleAssistant {
			assistantIdxs = append(assistantIdxs, i)
		} else if r == RoleTool {
			continue
		} else {
			break
		}
	}

	// 只保留第一条 assistant，其他合并到它
	base := stored[assistantIdxs[0]]
	for idx := 1; idx < len(assistantIdxs); idx++ {
		next := stored[assistantIdxs[idx]]
		base.Segments = append(base.Segments, next.Segments...)
		if next.Message.Content != "" {
			if base.Message.Content == "" {
				base.Message.Content = next.Message.Content
			} else {
				base.Message.Content += "\n" + next.Message.Content
			}
		}
		if len(next.Message.ToolCalls) > 0 {
			base.Message.ToolCalls = append(base.Message.ToolCalls, next.Message.ToolCalls...)
		}
	}

	// 重建：保留 startIdx 之前的内容，追加合并后的 base，跳过被合并的 assistant 行
	var result []StoredMessage
	result = append(result, stored[:assistantIdxs[0]]...)
	result = append(result, base)
	for i := startIdx; i < len(stored); i++ {
		isAssistant := false
		for _, ai := range assistantIdxs {
			if i == ai {
				isAssistant = true
				break
			}
		}
		if !isAssistant {
			result = append(result, stored[i])
		}
	}
	return result, assistantIdxs[0] + 1 // 返回合并后 base 的位置
}

// MergeAllAssistantRuns 全量合并 JSONL 中所有 user→assistant 跑批的连续助理条目。
// 批量扫描用（如初始化修复历史数据），将整个文件所有 run 都合并。
func (s *MessageStore) MergeAllAssistantRuns(convID string) error {
	mu := s.getConvMutex(convID)
	mu.Lock()
	defer mu.Unlock()

	stored, err := s.readJSONL(convID)
	if err != nil {
		return fmt.Errorf("MergeAllAssistantRuns: 读取失败: %w", err)
	}
	if len(stored) == 0 {
		return nil
	}

	merged := false
	i := 1 // 从索引 1 开始（跳过首条，通常是 user）
	for i < len(stored) {
		if stored[i].Message.Role == RoleAssistant {
			// 检查后面还有没有同 run 的 assistant
			hasNext := false
			for j := i + 1; j < len(stored); j++ {
				r := stored[j].Message.Role
				if r == RoleAssistant {
					hasNext = true
					break
				} else if r == RoleTool {
					continue
				} else {
					break
				}
			}
			if hasNext {
				stored, i = mergeAssistantRun(stored, i)
				merged = true
			} else {
				i++
			}
		} else {
			i++
		}
	}

	if !merged {
		return nil
	}

	// 重写索引
	for i := range stored {
		stored[i].Idx = i
	}

	// 写回
	out, err := os.Create(s.convFilePath(convID))
	if err != nil {
		return fmt.Errorf("MergeAllAssistantRuns: 创建文件失败: %w", err)
	}
	defer out.Close()
	enc := json.NewEncoder(out)
	for _, sm := range stored {
		if err := enc.Encode(sm); err != nil {
			return fmt.Errorf("MergeAllAssistantRuns: JSON 编码失败: %w", err)
		}
	}

	// 更新计数器
	s.pcMu.Lock()
	s.persistedCount[convID] = len(stored)
	s.pcMu.Unlock()

	return nil
}

// LoadAll 加载对话全部消息（仅 Message，不含 Segments），供 LLM 上下文恢复。
// 若主文件（.jsonl）不存在，自动尝试从 .bak 备份恢复。
func (s *MessageStore) LoadAll(convID string) ([]Message, error) {
	msgs, err := s.readJSONL(convID)
	if err != nil {
		return nil, err
	}
	if msgs == nil {
		// ★ 主文件不存在：尝试从 .bak 恢复
		bakMsgs, bakErr := s.readJSONLBak(convID)
		if bakErr == nil && bakMsgs != nil {
			// 恢复成功：将 .bak 恢复为正式文件
			destPath := s.convFilePath(convID)
			bakPath := destPath + ".bak"
			os.Rename(bakPath, destPath)
			msgs = bakMsgs
		}
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

// TruncateTo 截断对话消息文件，只保留前 count 条消息。
// 同时更新 index.json 中的 MsgCount 和 UpdatedAt。
func (s *MessageStore) TruncateTo(convID string, count int) error {
	if count < 0 {
		count = 0
	}
	mu := s.getConvMutex(convID)
	mu.Lock()
	defer mu.Unlock()

	fpath := s.convFilePath(convID)
	raw, err := os.ReadFile(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("TruncateTo: 读取文件失败: %w", err)
	}
	lines := strings.Split(string(raw), "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if count >= len(lines) {
		return nil
	}
	kept := strings.Join(lines[:count], "\n") + "\n"
	if err := os.WriteFile(fpath, []byte(kept), 0644); err != nil {
		return fmt.Errorf("TruncateTo: 写入文件失败: %w", err)
	}

	s.indexMu.Lock()
	defer s.indexMu.Unlock()
	metas, _ := s.loadIndex()
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range metas {
		if metas[i].ID == convID {
			metas[i].MsgCount = count
			metas[i].UpdatedAt = now
			break
		}
	}
	return s.saveIndex(metas)
}

// ReplaceHistory 删除对话的全部旧消息，替换为压缩后的 msgs 版本。
// 全量重写 JSONL 文件：只保留 msgs 中的 assistant/tool 消息（跳过 system/user）。
// 在压缩历史后调用，防止 JSONL 文件无限增长。
func (s *MessageStore) ReplaceHistory(convID string, msgs []Message) error {
	mu := s.getConvMutex(convID)
	mu.Lock()
	defer mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	// 只保留 assistant/tool 消息，跳过 system/user
	var kept []StoredMessage
	for i, m := range msgs {
		if m.Role == RoleSystem || m.Role == RoleUser {
			continue
		}
		sm := StoredMessage{
			Idx:       len(kept),
			Message:   m,
			Segments:  SegmentsFromMessage(m, msgs, i),
			Timestamp: now,
		}
		kept = append(kept, sm)
	}

	// 全量重写 JSONL 文件
	fpath := s.convFilePath(convID)
	out, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("ReplaceHistory: 创建文件失败: %w", err)
	}
	defer out.Close()
	enc := json.NewEncoder(out)
	for _, sm := range kept {
		if err := enc.Encode(sm); err != nil {
			return fmt.Errorf("ReplaceHistory: JSON 编码失败: %w", err)
		}
	}

	// 更新内存计数器
	s.pcMu.Lock()
	s.persistedCount[convID] = len(kept)
	s.pcMu.Unlock()

	// 更新 index.json
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("ReplaceHistory: 读取 index 失败: %w", err)
	}

	found := false
	for i := range metas {
		if metas[i].ID == convID {
			metas[i].MsgCount = len(kept)
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
			MsgCount:  len(kept),
		})
	}
	return s.saveIndex(metas)
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
// ★ 2026-08-31 成员会话隔离：多智能体团队的成员会话（conv_sub_*，见
//   subagent_registry.newSubAgentConvID）是「船长会话的子会话」，只在团队活动
//   面板内通过 openMember 打开，不占顶层会话列表（对齐 DSH ActivityPanel 的
//   子会话语义），因此此处统一过滤，避免成员会话污染会话列表/记忆重建等消费方。
func (s *MessageStore) ListConversations(workspaceRoot string) ([]ConversationMeta, error) {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return nil, fmt.Errorf("ListConversations: 读取 index 失败: %w", err)
	}

	out := make([]ConversationMeta, 0, len(metas))
	// ★ 路径归一化匹配：历史数据中 Windows 路径存在单/双反斜杠混写
	//   （如 F:\syproject\gou-ide vs F:\\syproject\gou-ide），精确匹配会漏掉。
	//   filepath.Clean + EqualFold（Windows 路径不区分大小写）统一比较。
	normRoot := strings.TrimSpace(filepath.Clean(workspaceRoot))
	for _, m := range metas {
		if IsSubAgentConvID(m.ID) {
			continue // 团队成员子会话不出现在顶层会话列表（团队面板内打开）
		}
		if m.WorkspaceRoot == "" {
			out = append(out, m)
			continue
		}
		norm := strings.TrimSpace(filepath.Clean(m.WorkspaceRoot))
		if norm == normRoot || strings.EqualFold(norm, normRoot) {
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

// SetConvModel 设置会话级模型路由（服务商 + 执行模型 + 配置名）。
// ★ 2026-08-31：模型切换只改本会话，不动全局 settings、不影响其他/历史对话。
// ★ 2026-09-03 增加 preset（AI 配置名）：装配时按配置整套展开（含该配置 Key）。
// provider/model/preset 全空视为清除（回落全局默认）。不更新 UpdatedAt（避免打乱列表排序）。
func (s *MessageStore) SetConvModel(convID, provider, model, preset string) error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("SetConvModel: 读取 index 失败: %w", err)
	}
	for i := range metas {
		if metas[i].ID == convID {
			if metas[i].Provider == provider && metas[i].Model == model && metas[i].Preset == preset {
				return nil
			}
			metas[i].Provider = provider
			metas[i].Model = model
			metas[i].Preset = preset
			if err := s.saveIndex(metas); err != nil {
				return fmt.Errorf("SetConvModel: 写入 index 失败: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("SetConvModel: 会话 %s 不存在", convID)
}

// ConvModel 读取会话级模型路由（provider, model；均空=未设置，用全局默认）。
func (s *MessageStore) ConvModel(convID string) (string, string) {
	meta, err := s.GetConversation(convID)
	if err != nil || meta == nil {
		return "", ""
	}
	return meta.Provider, meta.Model
}

// SetConvReviewMode 设置会话级审核模式（runtime="auto/manual/off"；空=清除，回落全局）。
// 与 SetConvModel 同理：会话内切换只改本会话并持久化，不动全局 settings。
func (s *MessageStore) SetConvReviewMode(convID, mode string) error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("SetConvReviewMode: 读取 index 失败: %w", err)
	}
	for i := range metas {
		if metas[i].ID == convID {
			if metas[i].ReviewMode == mode {
				return nil
			}
			metas[i].ReviewMode = mode
			if err := s.saveIndex(metas); err != nil {
				return fmt.Errorf("SetConvReviewMode: 写入 index 失败: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("SetConvReviewMode: 会话 %s 不存在", convID)
}

// ConvReviewMode 读取会话级审核模式（空=未设置，用全局/工作区配置）。
func (s *MessageStore) ConvReviewMode(convID string) string {
	meta, err := s.GetConversation(convID)
	if err != nil || meta == nil {
		return ""
	}
	return meta.ReviewMode
}

// SetInterrupted 更新对话的异常中断标记（不更新 UpdatedAt，避免影响列表排序）。
// 会话异常终止（LLM API 错误/panic/进程崩溃等非用户停止）时置 true；
// 正常完成、用户停止或用户重新发起继续时置 false。
// 前端据此在对话列表与输入区展示"未完成，可继续"引导。
func (s *MessageStore) SetInterrupted(convID string, interrupted bool) error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	metas, err := s.loadIndex()
	if err != nil {
		return fmt.Errorf("SetInterrupted: 读取 index 失败: %w", err)
	}

	for i := range metas {
		if metas[i].ID == convID {
			if metas[i].Interrupted != interrupted {
				metas[i].Interrupted = interrupted
				if err := s.saveIndex(metas); err != nil {
					return fmt.Errorf("SetInterrupted: 写入 index 失败: %w", err)
				}
			}
			return nil
		}
	}
	return nil // 不存在则无操作
}

// ── 自动归档 ──

// ArchiveThreshold 触发归档的消息数阈值。
const ArchiveThreshold = 500

// ArchiveRatio 归档后保留的消息比例（保留最新的 1/ArchiveRatio）。
const ArchiveRatio = 4

// SummaryLength 归档摘要的最大字符数。
const SummaryLength = 200

// ArchivedFileSuffix 归档后截断部分的文件后缀。
const ArchivedFileSuffix = ".archived.jsonl"

// checkAndArchive 检查对话消息数，超过阈值则自动归档最早的消息。
// 归档策略：将最早的一部分消息移出主 JSONL，写入 {convID}.archived.jsonl 文件，
// 并在主文件中保留一条归档摘要消息。
func (s *MessageStore) checkAndArchive(convID string) error {
	count := s.GetPersistedCount(convID)
	if count < ArchiveThreshold {
		return nil
	}

	// 读取全部消息
	messages, err := s.ReadAll(convID)
	if err != nil {
		return fmt.Errorf("读取消息以归档: %w", err)
	}

	// 计算保留数量（保留最新的 1/4）
	keepCount := len(messages) / ArchiveRatio
	if keepCount < 50 {
		keepCount = 50 // 至少保留 50 条
	}
	archiveCount := len(messages) - keepCount
	if archiveCount <= 0 {
		return nil
	}

	// 提取要归档的消息（最早的 archiveCount 条）
	archived := messages[:archiveCount]
	keep := messages[archiveCount:]

	// 生成归档摘要
	summary := s.generateArchiveSummary(convID, archived)

	// 将归档消息写入 .archived.jsonl
	archivedPath := s.convFilePath(convID) + ArchivedFileSuffix
	f, err := os.OpenFile(archivedPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("创建归档文件: %w", err)
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	for _, msg := range archived {
		if err := encoder.Encode(msg); err != nil {
			return fmt.Errorf("写入归档消息: %w", err)
		}
	}

	// 重写主文件（只保留 keep + 摘要消息）
	mainPath := s.convFilePath(convID)
	tmpPath := mainPath + ".tmp"
	tf, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("创建临时文件: %w", err)
	}
	mainEncoder := json.NewEncoder(tf)

	// 写入归档摘要消息作为第一条（使前端能看到归档记录）。
	// ★ Role 必须用 RoleUser 而非 RoleAssistant：摘要是系统生成的说明，不是 agent 的回复，
	//   以孤立 assistant 消息排在对话开头会让 LLM 上下文出现无 user 配对的 assistant
	//   （部分 API 拒绝以 assistant 开头，且 LLM 会误认为「自己说过的话」）。
	//   用 RoleUser + 【历史归档】标注与背景块（backgroundCtxMarker 同款 user 注入模式）一致，
	//   LLM 可将其理解为「早期历史被归档的说明」；系统提示多轮规则会把它当历史轮次处理
	//   （最后一条 user 消息才是当前任务），不会污染当前任务识别。
	summaryMsg := StoredMessage{
		Idx:       0,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Message: Message{
			Role:    RoleUser,
			Content: summary,
		},
		Segments: []Segment{{
			Type:    "content",
			Content: summary,
		}},
	}
	if err := mainEncoder.Encode(summaryMsg); err != nil {
		tf.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("写入摘要消息: %w", err)
	}

	// 重编号并写入保留消息
	for i, msg := range keep {
		msg.Idx = i + 1 // 摘要消息占 idx=0，保留消息从 1 开始
		if err := mainEncoder.Encode(msg); err != nil {
			tf.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("写入保留消息: %w", err)
		}
	}
	tf.Close()

	// ★ 原子替换（Windows 不允许 Rename 覆盖已有文件，用三步法：
	//   1. Rename main → main.bak（备份旧文件）
	//   2. Rename tmp → main（新文件就位）
	//   3. Remove main.bak（新文件确认无误后删备份）
	//   与 PersistNewMessages 崩溃安全替换一致；若 1-2 间崩溃，LoadAll 可从 .bak 恢复。
	bakPath := mainPath + ".bak"
	os.Rename(mainPath, bakPath) // 备份旧文件（不存在时忽略错误）
	if err := os.Rename(tmpPath, mainPath); err != nil {
		// 替换失败：尝试恢复备份
		os.Rename(bakPath, mainPath)
		os.Remove(tmpPath)
		return fmt.Errorf("替换主文件: %w", err)
	}
	os.Remove(bakPath)

	// 更新持久化计数
	s.pcMu.Lock()
	s.persistedCount[convID] = len(keep) + 1 // +1 算上摘要
	s.pcMu.Unlock()

	return nil
}

// generateArchiveSummary 从归档消息生成摘要文本。
func (s *MessageStore) generateArchiveSummary(convID string, msgs []StoredMessage) string {
	if len(msgs) == 0 {
		return "（历史消息已归档，无内容）"
	}

	// 统计各类消息数
	userMsgs := 0
	assistantMsgs := 0
	toolCalls := 0
	toolResults := 0

	for _, m := range msgs {
		switch m.Message.Role {
		case RoleUser:
			userMsgs++
		case RoleAssistant:
			assistantMsgs++
			if len(m.Message.ToolCalls) > 0 {
				toolCalls++
			}
		case RoleTool:
			toolResults++
		}
	}

	// 提取第一条用户消息作为上下文
	firstUserMsg := ""
	for _, m := range msgs {
		if m.Message.Role == RoleUser && m.Message.Content != "" {
			firstUserMsg = m.Message.Content
			if len(firstUserMsg) > 100 {
				firstUserMsg = firstUserMsg[:100] + "…"
			}
			break
		}
	}

	summary := fmt.Sprintf("【历史归档】**历史归档**（共 %d 条消息：用户 %d 条、助手 %d 条、工具调用 %d 次）",
		len(msgs), userMsgs, assistantMsgs, toolCalls+toolResults)
	if firstUserMsg != "" {
		summary += fmt.Sprintf("\n最早对话主题：%s", firstUserMsg)
	}
	summary += fmt.Sprintf("\n_已归档至 %s.archived.jsonl，需要时可查看_", convID)

	if len(summary) > SummaryLength {
		summary = summary[:SummaryLength] + "…"
	}
	return summary
}

// GetPersistedCount 获取已持久化的非 System 消息数。
func (s *MessageStore) GetPersistedCount(convID string) int {
	s.pcMu.RLock()
	defer s.pcMu.RUnlock()
	return s.persistedCount[convID]
}

// ReadAll 读取对话全部消息。
func (s *MessageStore) ReadAll(convID string) ([]StoredMessage, error) {
	path := s.convFilePath(convID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var msgs []StoredMessage
	scanner := bufio.NewScanner(f)
	// 增加扫描缓冲区大小（处理长行）
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg StoredMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue // 跳过损坏行
		}
		msgs = append(msgs, msg)
	}
	return msgs, scanner.Err()
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
