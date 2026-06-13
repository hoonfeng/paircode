package state

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ── history.json 文件格式 ─────────────────────────────────────
//
// historyFile 映射 .pair/conversations/history.json 的文件结构。
// 这是项目已有的对话历史格式，不要擅自改变。
type historyFile struct {
	Version int       `json:"version"`
	Seq     int       `json:"seq"`
	Threads []*Thread `json:"threads"`
}

// Role 消息角色。
type Role int

const (
	User Role = iota
	Assistant
)

// TimelineEntry 按 LLM 输出顺序记录的一条事件（思考/正文/工具调用）。
// 保留独立字段而非与 Thinking/Text/Activities 合并，使得 agent_card 渲染时
// 能按真实事件流顺序展示（而非固定分三块），还原 LLM 输出的自然节奏。
type TimelineEntry struct {
	Kind    string // "thinking" / "content" / "tool"
	Content string // thinking/content 的文本内容

	// tool 专用字段（同 Activity）
	Tool             string
	Args             string
	CallID           string
	Result           string
	Done             bool
	Expanded         bool
	AwaitingApproval bool
}

// Message 一条聊天消息。Assistant 消息可携带 Agent 流式结构（思考链 + 工具活动）。
type Message struct {
	Role Role
	Text string

	// Rounds 保留 Agent 执行轮次记录（与 .pair/conversations/history.json 兼容）。
	// 仅用于持久化兼容，业务逻辑不使用此字段。
	Rounds json.RawMessage `json:"Rounds,omitempty"`

	// ── Agent 流式（仅 Assistant，可选）──
	Thinking   string     // 思考链（DeepSeek reasoning，dim 折叠显示）
	Activities []Activity // 本轮工具调用活动（按发生顺序）
	Notes      []string   // 系统级提示（如「上下文已压缩」），素色显示在卡内（非 LLM 正文）
	Eval       *Eval      // 任务评测评分（完成后由评测模型打分；nil=未评测）
	Streaming  bool       // 仍在流式生成中（显示「思考中…」/进行态）

	// Timeline 按 LLM 输出事件流顺序记录的条目列表。与 Thinking/Text/Activities
	// 保持同步（EventThinking→Thinking 追加 + Timeline thinking 条目，
	// EventContent→Text 追加 + Timeline content 条目，
	// EventToolCall→Activities 追加 + Timeline tool 条目）。
	// 渲染时优先使用 Timeline（若存在）还原真实输出顺序；无 Timeline
	// 时回退到旧版三块独立渲染（向后兼容已存储的旧对话）。
	Timeline []TimelineEntry `json:"Timeline,omitempty"`

	// ── 折叠视图状态（UI 持久，跨 relayout 存活于 store）──
	Collapsed        bool // 整卡折叠（完成后可收起，autoCollapse 开则完成即收）
	ThinkingExpanded bool // 思考块展开（默认折叠；流式时强制展开看实时）
}

// TimelineEntryAsActivity 将 tool 类型的 TimelineEntry 转为 Activity
// （供 activityInline 复用现有工具活动展示组件）。
func (e TimelineEntry) TimelineEntryAsActivity() Activity {
	return Activity{
		CallID:           e.CallID,
		Tool:             e.Tool,
		Args:             e.Args,
		Result:           e.Result,
		Done:             e.Done,
		Expanded:         e.Expanded,
		AwaitingApproval: e.AwaitingApproval,
	}
}

// Eval 任务评测评分（评测模型 LLM-as-Judge 打分；复刻参考 bench/evaluator 的 4 维度 + 总分 + 优缺点 + 反馈）。
type Eval struct {
	Total                                      int      // 总分 0-100
	Completion, Correctness, Depth, Efficiency int      // 4 维度分（满分 40/30/20/10）
	Strengths, Weaknesses                      []string // 优点 / 不足
	Feedback                                   string   // 一句话总评
}

// Activity 一次工具调用的展示记录（调用 + 结果）。
type Activity struct {
	CallID           string
	Tool             string // 工具名，如 read_file
	Args             string // 参数 JSON（预览用）
	Result           string // 结果文本（done 后填，预览用）
	Done             bool   // 结果是否已回来
	AwaitingApproval bool   // 手动审核：等待用户点「允许/拒绝」（写类工具，见 agent_bridge.go）
	Expanded         bool   // 展开看全量结果（默认折叠只显首行预览）
}

// TokenUsage 记录对话的 token 使用统计（按消息内容启发式估算）。
// 提供图标展示所需的 Prompt/Completion/Total 三组数据。
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 输入 token（用户消息）
	CompletionTokens int `json:"completion_tokens"` // 输出 token（助手回复）
	TotalTokens      int `json:"total_tokens"`      // 总计
}

// Thread 一个对话会话。
type Thread struct {
	ID         string
	Title      string
	Messages   []Message
	TokenUsage TokenUsage `json:"TokenUsage,omitempty"` // 该会话 token 统计，持久化
}

// textTokens 文本 token 估算：CJK 字 ×1.5、其余 ×0.25（与 agent/compress.go 保持一致的启发式算法）。
func textTokens(s string) float64 {
	cjk, other := 0, 0
	for _, r := range s {
		if isCJK(r) {
			cjk++
		} else {
			other++
		}
	}
	return float64(cjk)*1.5 + float64(other)*0.25
}

// isCJK 是否中日韩表意字（与 agent/compress.go 保持一致的 Unicode 区间判断）。
func isCJK(r rune) bool {
	switch {
	case r >= 0x4E00 && r <= 0x9FFF: // CJK 统一表意
		return true
	case r >= 0x3400 && r <= 0x4DBF: // 扩展 A
		return true
	case r >= 0xF900 && r <= 0xFAFF: // 兼容表意
		return true
	}
	return false
}

// CalculateTokenUsage 遍历会话的所有消息，按角色和内容启发式估算 token 消耗。
// 用户消息算作 PromptTokens，助手消息（含思考链、工具活动、正文）算作 CompletionTokens。
// 每次访问时实时计算，确保数据始终与消息内容一致。
func (t *Thread) CalculateTokenUsage() TokenUsage {
	var prompt, completion float64
	for _, m := range t.Messages {
		if m.Role == User {
			prompt += 4                               // 消息开销
			prompt += textTokens(m.Text)               // 正文
		} else {
			completion += 4                            // 消息开销
			completion += textTokens(m.Text)           // 正文
			completion += textTokens(m.Thinking)       // 思考链
			for _, a := range m.Activities {
				completion += textTokens(a.Tool)       // 工具名
				completion += float64(len([]rune(a.Args))) * 0.25 // 工具参数
				completion += textTokens(a.Result)     // 工具结果
			}
			for _, entry := range m.Timeline {
				if entry.Kind == "tool" {
					completion += textTokens(entry.Tool)
					completion += float64(len([]rune(entry.Args))) * 0.25
				}
			}
		}
	}
	pt := int(math.Ceil(prompt))
	ct := int(math.Ceil(completion))
	return TokenUsage{
		PromptTokens:     pt,
		CompletionTokens: ct,
		TotalTokens:      pt + ct,
	}
}

// ChatStore 多会话聊天状态：会话列表 + 当前会话 + 输入草稿。是对话面板的唯一真相来源。
type ChatStore struct {
	Threads  []*Thread
	ActiveID string
	Draft    string
	seq      int
}

// NewChatStore 新建聊天状态（含一个空会话）。
func NewChatStore() *ChatStore {
	s := &ChatStore{}
	s.NewThread()
	return s
}

const welcome = "你好！我是伴随式 CodeAgent。告诉我你想做什么——写代码、改 bug、跑命令，我来陪你一起。"

// NewThread 新建会话并置为当前（置顶）。
func (s *ChatStore) NewThread() *Thread {
	s.seq++
	t := &Thread{
		ID:       "t" + strconv.Itoa(s.seq),
		Title:    "新对话",
		Messages: []Message{{Role: Assistant, Text: welcome}},
	}
	s.Threads = append([]*Thread{t}, s.Threads...) // 新会话置顶
	s.ActiveID = t.ID
	s.Draft = ""
	return t
}

// Active 返回当前会话（无则返回首个/nil）。
func (s *ChatStore) Active() *Thread {
	for _, t := range s.Threads {
		if t.ID == s.ActiveID {
			return t
		}
	}
	if len(s.Threads) > 0 {
		return s.Threads[0]
	}
	return nil
}

// Switch 切换当前会话。
func (s *ChatStore) Switch(id string) {
	s.ActiveID = id
	s.Draft = ""
}

// Delete 删除会话；若删的是当前会话则切到首个，删空则新建一个。
func (s *ChatStore) Delete(id string) {
	out := s.Threads[:0]
	for _, t := range s.Threads {
		if t.ID != id {
			out = append(out, t)
		}
	}
	s.Threads = out
	if s.ActiveID == id {
		if len(s.Threads) > 0 {
			s.ActiveID = s.Threads[0].ID
		} else {
			s.NewThread()
		}
	}
}

// Send 把文本作为用户消息发到当前会话、设标题、清草稿。回复由 Agent 引擎异步流式追加
// （见 cmd/companion/agent_bridge.go）。首条用户消息成为会话标题。返回是否实际发送（空白不发）。
func (s *ChatStore) Send(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	t := s.Active()
	if t == nil {
		return false
	}
	t.Messages = append(t.Messages, Message{Role: User, Text: text})
	if t.Title == "新对话" {
		title := text
		if r := []rune(title); len(r) > 16 {
			title = string(r[:16]) + "…"
		}
		t.Title = title
	}
	s.Draft = ""
	return true
}

// Save 把聊天状态持久化到 .pair/conversations/history.json。
// 保存前自动更新每个会话的 TokenUsage 统计，确保持久化数据包含最新 token 用量。
// 格式为 historyFile（含 version/seq/threads），与项目已有的对话历史格式兼容。
func (s *ChatStore) Save(root string) {
	for _, t := range s.Threads {
		t.TokenUsage = t.CalculateTokenUsage()
	}
	hf := historyFile{
		Version: 1,
		Seq:     s.seq,
		Threads: s.Threads,
	}
	data, err := json.Marshal(hf)
	if err != nil {
		return
	}
	dir := filepath.Join(root, ".pair", "conversations")
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, "history.json"), data, 0644)
}

// Load 从 .pair/conversations/history.json 加载聊天状态。成功返回 true。
func (s *ChatStore) Load(root string) bool {
	data, err := os.ReadFile(filepath.Join(root, ".pair", "conversations", "history.json"))
	if err != nil {
		return false
	}
	var hf historyFile
	if err := json.Unmarshal(data, &hf); err != nil {
		return false
	}
	s.Threads = hf.Threads
	s.seq = hf.Seq
	// 折叠所有已完成的历史消息，大幅减少首次渲染的 widget 树复杂度
	//（折叠态每条消息约 5 个 widget vs 展开态 100+ widget），
	// 避免大量历史消息同步 Build 阻塞 UI 主线程导致窗口白屏无响应。
	totalMsgs := 0
	for _, t := range s.Threads {
		totalMsgs += len(t.Messages)
		for i := range t.Messages {
			// 安全兜底：持久化的消息不可能还在流式（保存时已标记完成），
			// 防止旧数据/异常中断导致 Streaming=true 让 UI 误显"思考中/运行中"。
			if t.Messages[i].Streaming {
				t.Messages[i].Streaming = false
			}
			t.Messages[i].Collapsed = true
		}
	}
	if len(s.Threads) > 0 {
		s.ActiveID = s.Threads[0].ID
	} else {
		s.NewThread()
	}
	s.Draft = ""
	return true
}
