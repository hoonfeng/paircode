package state

import (
	"strconv"
	"strings"
)

// Role 消息角色。
type Role int

const (
	User Role = iota
	Assistant
)

// Message 一条聊天消息。Assistant 消息可携带 Agent 流式结构（思考链 + 工具活动）。
type Message struct {
	Role Role
	Text string

	// ── Agent 流式（仅 Assistant，可选）──
	Thinking   string     // 思考链（DeepSeek reasoning，dim 折叠显示）
	Activities []Activity // 本轮工具调用活动（按发生顺序）
	Notes      []string   // 系统级提示（如「上下文已压缩」），素色显示在卡内（非 LLM 正文）
	Eval       *Eval      // 任务评测评分（完成后由评测模型打分；nil=未评测）
	Streaming  bool       // 仍在流式生成中（显示「思考中…」/进行态）

	// ── 折叠视图状态（UI 持久，跨 relayout 存活于 store）──
	Collapsed        bool // 整卡折叠（完成后可收起，autoCollapse 开则完成即收）
	ThinkingExpanded bool // 思考块展开（默认折叠；流式时强制展开看实时）
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

// Thread 一个对话会话。
type Thread struct {
	ID       string
	Title    string
	Messages []Message
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
