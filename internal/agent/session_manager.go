package agent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"

	pkgdb "github.com/hoonfeng/paircode/pkg/db"
)

// Compressor 上下文压缩器（语义别名，复用 Provider 接口）。
// 非空时用轻量压缩模型做 LLM 摘要；空则规则式摘要。
type Compressor = Provider

// LoopOpts 创建 Loop 所需的全部参数（供 SessionManager.Start 使用）。
// 把原本散落在 web 层的 Loop 构造逻辑收敛到一处，便于并行会话统一创建。
type LoopOpts struct {
	Provider         Provider    // LLM 提供方
	Registry         *Registry   // 工具注册表（Start 会在此注册 ask_user 工具）
	System           string      // 系统提示词
	MaxIterations    int         // 最大迭代数（<=0 时 Loop 内部默认 30）
	MaxContextTokens int         // 上下文 token 上限（>0 启用压缩）
	Compressor       Compressor  // 上下文压缩器（可空）
	History          []Message   // 初始历史（首次为空；续跑时传上一轮 History）
	CompressedSummaries []string // 已持久化的压缩摘要（页面刷新后恢复）
	Autonomous       bool        // 自主模式标志
	WorkspaceRoot    string      // 工作区根路径（用于跨工作区并行对话的状态指示与隔离）
	// AutoReview AI 审核开关。true=Loop 内部 AI 审核把关写操作；
	// false+Autonomous=true=自动放行；false+Autonomous=false=人工审批（前端弹窗）。
	AutoReview bool
	// ReviewProvider 审核模型的 Provider（AutoReview=true 时用）。Loop 内部用它懒建 Reviewer。
	ReviewProvider Provider
	// AutoCommit 任务完成时自动 git add + git commit。
	AutoCommit bool
	// PlanProvider 规划模型的 Provider（自主模式用）。非空时启动外层设计者 Loop（update_plan + delegate_task），
	// 而非直接跑单层 Loop。桌面端/web端的行为统一。
	PlanProvider Provider
}

// GlobalEvent 是全局订阅者收到的事件：携带 convID 用于前端路由。
// WebSocket 端点通过 SubscribeAll 获取所有会话事件，每条都带 convID。
type GlobalEvent struct {
	ConvID string
	Event  Event
}

// Session 一次 agent 运行会话。从 web 层 webAgentSession 下沉而来，
// 封装 Loop、事件通道、交互通道与订阅者 fan-out，供 SessionManager 统一管理。
type Session struct {
	ConvID        string
	WorkspaceRoot string // 工作区根路径（跨工作区并行对话的状态指示）
	Loop          *Loop
	Events        chan Event // Loop OnEvent 写入此通道；fan-out 从此通道分发
	Cancel        context.CancelFunc
	History       []Message
	Running       bool
	StartedAt     time.Time

	// 交互通道（从 web 层 webAgentSession 迁移）
	askCh      chan string // ask_user 工具阻塞等用户回答
	approvalCh chan bool   // Approve 钩子阻塞等用户裁决
	feedbackCh chan string // OnFeedback 每轮 LLM 调用前检查

	// 订阅者 fan-out：多个 SSE 客户端可订阅同一会话事件
	subscribers []chan Event
	subMu       sync.RWMutex
	finished    bool // fan-out 已退出（subMu 保护）；true 时新订阅直接返回已关闭 channel

	// stopped 标记用户主动停止（区别于自然结束，避免多发错误事件）
	stopped bool
}

// SessionManager 并行会话架构核心：管理多个并行 Session。
// web/gui 层只是套壳——创建会话、订阅事件、发送交互信号都走这里。
// 会话结束后保留条目（GetHistory 仍可读），上限 maxSessions 防内存泄漏。
type SessionManager struct {
	mu          sync.RWMutex
	sessions    map[string]*Session // key: convID
	maxSessions int                 // 已结束会话保留上限（默认 100）

	// 全局订阅者：WebSocket 端点订阅所有会话事件（跨工作区并行）
	globalSubMu       sync.RWMutex
	globalSubscribers []chan GlobalEvent

	store ConversationStore // 消息持久化存储（通过 SetWorkspaceRoot 注入）

	// ds 统一 SQLite 数据库实例，codegraph/其他组件共用同一连接。
	ds *pkgdb.SQLiteDB

	// 内部持久化 worker 状态
	persistWorkerStarted bool

	// OnDone 会话完成回调（由 web 层设置，用于生成对话摘要等副作用）。
	// 在 Session goroutine 写盘后调用，convID 为刚结束的会话 ID。
	OnDone func(convID string)
}

// NewSessionManager 创建会话管理器，默认保留 100 个已结束会话。
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:    make(map[string]*Session),
		maxSessions: 100,
	}
}

// SetWorkspaceRoot 注入工作区根路径，初始化 MessageStore（JSONL 消息存储）和 SQLite（供 codegraph 使用）。
// 消息持久化使用 MessageStore（JSONL），codegraph 等组件通过 RawDB() 获取同一 *sql.DB 实例。
func (m *SessionManager) SetWorkspaceRoot(root string) {
	m.mu.Lock()

	// 初始化 MessageStore（JSONL 消息存储）
	m.store = NewMessageStore(root)

	// 初始化 SQLite（供 codegraph 等组件使用，不再担当消息存储）
	dbPath := filepath.Join(root, ".pair", "pair.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		fmt.Printf("[session] 创建数据库目录失败: %v\n", err)
		m.ds = nil
		m.mu.Unlock()
		return
	}
	if ds, err := pkgdb.NewSQLiteDB(dbPath); err == nil {
		m.ds = ds
	} else {
		fmt.Printf("[session] 打开 SQLite 数据库失败: %v\n", err)
		m.ds = nil
	}

	shouldStart := !m.persistWorkerStarted
	if shouldStart {
		m.persistWorkerStarted = true
	}
	m.mu.Unlock()

	if shouldStart {
		m.startPersistWorker()
	}
}

// Store 返回 ConversationStore 引用（供 web 层调用 AppendMessage/LoadLatest 等）。
// 若 SetWorkspaceRoot 未调用则返回 nil（web 层需自行判空）。
func (m *SessionManager) Store() ConversationStore {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store
}

// RawDB 返回底层 *sql.DB（供 codegraph 等组件共享同一连接）。
// 返回 nil 表示数据库未初始化。
func (m *SessionManager) RawDB() *sql.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ds == nil {
		return nil
	}
	return m.ds.RawDB().(*sql.DB)
}

// SQLiteDB 返回 pkg/db.SQLiteDB 实例（供 codegraph SQLiteStore 等组件使用）。
func (m *SessionManager) SQLiteDB() *pkgdb.SQLiteDB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ds
}

// ErrSessionRunning convID 已有运行中的会话（同一对话不可并行跑两个 Loop）。
var ErrSessionRunning = errors.New("该会话已有运行中的任务")

// ErrSessionNotFound convID 无会话（SendAnswer/Approve 等交互时找不到目标）。
var ErrSessionNotFound = errors.New("会话不存在")

// TrimInterruptedHistory 从历史消息中移除因用户主动停止而未完成的最后一段 assistant/tool 消息，
// 但保留后续的用户消息（由 AppendPersistedUserMessage 预写入的新消息）。
// 判定规则：从末尾向前找最后一条 assistant 消息，如果它有 tool_call 但无匹配的 tool_result 则判定为中断。
func TrimInterruptedHistory(history []Message) []Message {
	if len(history) == 0 {
		return history
	}
	// 从末尾向前找最后一条 assistant 消息
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == RoleAssistant {
			// 如果最后一条 assistant 有正文且无 tool_call → 自然完成，保留全部
			if strings.TrimSpace(history[i].Content) != "" && len(history[i].ToolCalls) == 0 {
				return history
			}
			// 有 tool_call：检查是否有匹配的 tool_result
			hasAllResults := true
			for _, tc := range history[i].ToolCalls {
				hasResult := false
				for j := i + 1; j < len(history); j++ {
					if history[j].Role == RoleTool && history[j].ToolCallID == tc.ID {
						hasResult = true
						break
					}
				}
				if !hasResult {
					hasAllResults = false
					break
				}
			}
			if hasAllResults {
				// 所有 tool_call 都有结果 → 正常完成，保留全部
				return history
			}
			// 有未匹配的 tool_call → 会话被中断
			// 从 i 开始向后扫描，移除 assistant/tool 消息，但保留用户消息
			nextUserIdx := -1
			for j := i + 1; j < len(history); j++ {
				if history[j].Role == RoleUser {
					nextUserIdx = j
					break
				}
			}
			if nextUserIdx > 0 {
				result := make([]Message, 0, i+1+(len(history)-nextUserIdx))
				result = append(result, history[:i]...)
				for j := nextUserIdx; j < len(history); j++ {
					if history[j].Role == RoleUser {
						result = append(result, history[j])
					}
				}
				return result
			}
			// 没有后续用户消息，直接截断到 i-1
			return history[:i]
		}
	}
	return history
}

// ErrSessionNotRunning 会话未在运行（向已结束的会话发交互信号）。
var ErrSessionNotRunning = errors.New("会话未在运行")

// Start 为 convID 创建并启动一个新会话。
//
// 流程：
//  1. 若 convID 已有运行中的 session，返回 ErrSessionRunning
//  2. 创建 Session（设置 History、交互通道、订阅者切片）
//  3. 创建 Loop（用 opts 字段），挂载 OnEvent/Approve/OnFeedback 回调
//  4. 在 opts.Registry 上注册 ask_user 工具（从 askCh 读用户回答）
//  5. 启动 goroutine 跑 loop.Run（自闭环模式：传 nil history 用 loop.History）
//  6. 启动 fan-out goroutine：从 Events 读取分发到所有 subscribers
//
// 会话结束后（Run 返回）标记 Running=false，保留 History 供 GetHistory 读取。
func (m *SessionManager) Start(ctx context.Context, convID string, task string, opts LoopOpts) error {
	if convID == "" {
		return errors.New("convID 不能为空")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// 已有运行中的会话则拒绝（同一对话不可并行跑两个 Loop）
	if s, ok := m.sessions[convID]; ok && s.Running {
		return ErrSessionRunning
	}

	// 持久化：确保 store 中存在该对话的元数据。
	// store 为 nil（web 层未注入）时跳过；CreateConversation 失败只警告不阻塞 Start。
	if m.store != nil {
		existing, getErr := m.store.GetConversation(convID)
		if getErr != nil {
			fmt.Printf("[session] 查询对话 %s 元数据失败: %v\n", convID, getErr)
		} else if existing == nil {
			title := task
			if len(title) > 30 {
				title = title[:30]
			}
			if createErr := m.store.CreateConversation(convID, title, opts.WorkspaceRoot); createErr != nil {
				fmt.Printf("[session] 创建对话 %s 元数据失败: %v\n", convID, createErr)
			}
		}
	}

	// 创建会话上下文（使用独立的 context.Background()，不依赖调用方 ctx，
	// 避免 handleChatSend 的 defer setupCancel() 级联取消 runCtx，
	// 导致 Loop 尚未开始就 ctx.Err() != nil 直接返回）。
	runCtx, cancel := context.WithCancel(context.Background())

	sess := &Session{
		ConvID:        convID,
		WorkspaceRoot: opts.WorkspaceRoot,
		Events:        make(chan Event, 5000),
		Cancel:        cancel,
		History:       CopyHistory(opts.History), // 深复制避免外部修改
		Running:       true,
		StartedAt:     time.Now(),
		askCh:         make(chan string, 1),
		approvalCh:    make(chan bool, 1),
		feedbackCh:    make(chan string, 5),
	}

	// 创建 Loop，挂载回调
	loop := &Loop{
		Provider:         opts.Provider,
		Registry:         opts.Registry,
		System:           opts.System,
		MaxIterations:    opts.MaxIterations,
		MaxContextTokens: opts.MaxContextTokens,
		Compressor:       opts.Compressor,
		Autonomous:       opts.Autonomous,
		History:          CopyHistory(opts.History), // 自闭环模式：loop 自己管理持久历史
		CompressedSummaries: opts.CompressedSummaries, // 恢复已持久化的压缩摘要
		WorkspaceRoot:    opts.WorkspaceRoot, // 工作区根路径（用于 SaveTokenUsage 等工作区级持久化）
	}

	// ★ 恢复上一轮的执行日志（跨轮感知：无论自主还是非自主，新 Loop 都能知道之前每轮的分析/操作）
	if opts.WorkspaceRoot != "" {
		if savedLog := LoadExecutionLog(opts.WorkspaceRoot, convID); savedLog != nil && len(savedLog.Entries) > 0 {
			if loop.State == nil {
				loop.State = map[string]any{}
			}
			loop.State["executionLog"] = savedLog
		}
	}

	// OnEvent：将事件写入 session.Events（非阻塞，满则丢弃防阻塞 Loop）
	loop.OnEvent = func(e Event) {
		select {
		case sess.Events <- e:
		default:
		}
	}

	// Approve：写类工具执行前阻塞等用户裁决。
	// 先发 EventApproval 通知前端弹审批框，再从 approvalCh 读结果。
	// 当 AutoReview=false + Autonomous=false 时，由 Loop.Run 内部调用本回调。
	loop.Approve = func(actx context.Context, tc ToolCall) (bool, string) {
		select {
		case sess.Events <- Event{
			Type:   EventApproval,
			Tool:   tc.Function.Name,
			Args:   tc.Function.Arguments,
			CallID: tc.ID,
		}:
		default:
		}
		select {
		case approved := <-sess.approvalCh:
			return approved, ""
		case <-actx.Done():
			return false, "用户取消了操作"
		}
	}
	// 审核开关由 Loop 内部自决，外部只需传进来
	loop.SetAutoReview(opts.AutoReview)
	loop.ReviewProvider = opts.ReviewProvider

	// OnFeedback：每轮 LLM 调用前检查用户运行时反馈（非阻塞）
	loop.OnFeedback = func() string {
		select {
		case fb := <-sess.feedbackCh:
			return fb
		default:
			return ""
		}
	}

	// OnBatchPersist：每轮迭代由 loop.Run 内部回调，将当前完整消息列表写盘。
	// loop.Run 返回后 defer 中会额外调用一次 OnBatchPersist 作为兜底。
	// ★ 注意：不能调用 m.Store()，因为 Start 已持有 m.mu.Lock() 写锁，而 m.Store() 会尝试读锁导致死锁。
	// 直接使用已持有的 m.store 变量。
	if m.store != nil {
		store := m.store
		loop.OnBatchPersist = func(msgs []Message) {
			err := store.PersistNewMessages(convID, msgs)
			if err != nil {
				fmt.Printf("[persist] OnBatchPersist 失败 conv=%s err=%v\n", convID, err)
			} else {
				// 同时持久化压缩摘要，确保页面刷新后能恢复
				if serr := store.SaveCompressedSummaries(convID, loop.CompressedSummaries); serr != nil {
					fmt.Printf("[persist] SaveCompressedSummaries 失败 conv=%s err=%v\n", convID, serr)
				}
			}
		}
		// OnMessagePersist：单条消息强制落盘（delegate_task 委派任务用）
		loop.OnMessagePersist = func(msg Message) error {
			return store.AppendMessage(convID, msg, nil)
		}
	}

	sess.Loop = loop

	sess.Loop = loop

	// 注册 ask_user 工具：阻塞等用户回答（从 askCh 读）
	// Register 同名覆盖，安全替换调用方可能已注册的旧版本。
	if opts.Registry != nil {
		opts.Registry.Register(&Tool{
			Name: "ask_user",
			Description: "向用户提问并等待回答（用于关键决策、歧义澄清，别滥用）。" +
				"question 必填；type 可选(text/single/multi/single-with-input)，默认 text 纯文本输入；" +
				"options 可选(选择类 question 的选项列表)。调用会阻塞直到用户回答。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"question": map[string]any{"type": "string", "description": "向用户提出的问题"},
					"askType":   map[string]any{"type": "string", "enum": []string{"text", "single", "multi", "single-with-input"}, "description": "提问类型：text(纯文本)/single(单选)/multi(多选)/single-with-input(单选+自由输入)"},
					"options":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "选择类问题用：可选项列表"},
				},
				"required": []string{"question"},
			},
			RequiresApproval: false,
			Handler: func(hctx context.Context, args map[string]any) (string, error) {
				question, _ := args["question"].(string)
				if question == "" {
					question = "（无问题内容）"
				}
				select {
				case answer := <-sess.askCh:
					return strings.TrimSpace(answer), nil
				case <-hctx.Done():
					return "", hctx.Err()
				}
			},
		})

		// 注册本对话专属的 task_create：捕获 sess.ConvID 写入任务持久化记录
		opts.Registry.Register(&Tool{
			Name: "task_create",
			Description: "创建新的子任务。创建后必须立即执行该任务：先调用 task_update 标记为 in_progress 开始执行，" +
				"执行完成后调用 task_update 标记为 completed 并说明结果。重复此流程直到所有子任务完成。",
			Parameters: objSchema(props{
				"subject":      strProp("任务标题，用祈使句（如\"修复登录超时\"）"),
				"description":  strProp("详细描述：做什么、涉及哪些文件。不要包含文件原始内容，只写摘要。"),
				"dependencies": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "依赖的任务 ID 列表"},
			}, "subject", "description"),
			Handler: func(hctx context.Context, args map[string]any) (string, error) {
				subject := argStr(args, "subject")
				desc := argStr(args, "description")
				deps := argStrSlice(args, "dependencies")
				root := sess.WorkspaceRoot
				if root == "" {
					root = ""
				}
				tm := UseTaskManager(root)
				task := tm.Create(subject, desc, deps, sess.ConvID)
				return fmt.Sprintf("✅ 已创建任务 [%s] %s\n> %s\n\n状态: ⏳ 待执行\nID: `%s`", task.ID, task.Subject, task.Description, task.ID), nil
			},
		})

	}

	// 存入 map（覆盖已结束的旧会话），并做已结束会话上限淘汰
	m.sessions[convID] = sess
	m.evictIfNeeded()

	// fan-out goroutine：从 Events 读取，写入所有 subscribers（非阻塞，满则丢弃）。
	// 同时写入全局订阅者（WebSocket 端点），让跨工作区的所有会话事件都可通过单一连接传输。
	// Events 关闭后退出，并关闭所有剩余订阅者 channel（通知订阅者流结束）。
	go func() {
		for e := range sess.Events {
			sess.subMu.RLock()
			subs := sess.subscribers
			sess.subMu.RUnlock()
			for _, sub := range subs {
				select {
				case sub <- e:
				default:
				}
			}
			// 全局订阅者 fan-out（WebSocket）
			m.globalSubMu.RLock()
			gsubs := m.globalSubscribers
			m.globalSubMu.RUnlock()
			ge := GlobalEvent{ConvID: convID, Event: e}
			for _, gsub := range gsubs {
				select {
				case gsub <- ge:
				default:
				}
			}
		}
		// Events 已关闭：清空 subscribers 并逐个 close（通知订阅者流结束）。
		// 与 Unsubscribe 可能并发 close 同一 channel，用 recover 兜底防 panic。
		// 设置 finished=true 防止后续 Subscribe 添加永不关闭的订阅者。
		sess.subMu.Lock()
		subs := sess.subscribers
		sess.subscribers = nil
		sess.finished = true
		sess.subMu.Unlock()
		for _, sub := range subs {
			closeChan(sub)
		}
	}()

	// Loop.Run goroutine：结束后标记 Running=false、更新 History、关闭 Events。
	go func() {
		defer func() {
			// panic recovery：确保会话状态和事件通道始终被清理
			if r := recover(); r != nil {
				fmt.Printf("[session] Loop goroutine panic conv=%s: %v\n", convID, r)
				// ★ panic 恢复时也发送 EventError，否则前端无任何信号（assistant 永久 loading）
				// 注：用 select+default 非阻塞发送，不阻塞 defer 关闭流程，也不向已关闭 channel panic
				select {
				case sess.Events <- Event{Type: EventError, Content: fmt.Sprintf("Agent 异常终止: %v", r)}:
				default:
				}
				sess.History = loop.History
			}
			// 更新 History（loop.Run 的 defer 已更新 loop.History，同步到 session）
			sess.History = loop.History
			// 标记结束
			m.mu.Lock()
			sess.Running = false
			m.mu.Unlock()
			// 关闭 Events → fan-out goroutine 退出，subscribers 检测到通道关闭
			close(sess.Events)
		}()

		// 自闭环模式：传 nil history，loop.Run 内部使用 loop.History
		var msgs []Message
		var err error
		if opts.Autonomous && opts.PlanProvider != nil {
			// 自主模式：外层设计者 Loop（update_plan + delegate_task）→ 内层执行 Loop
			_, err = RunAutonomous(runCtx, opts.PlanProvider, loop, task)
			msgs = loop.History

			// ★ 自主模式消息回灌：只注入最后的完成报告
			// 不再注入完整历史上下文（之前的 extractAutonomousRunSummary 提取多条 assistant 消息，
			// 导致上下文膨胀且携带过多无关中间细节）。
			// 改为只注入 loop.finishResult（最终完成报告），让下一轮感知上一轮的核心成果。
			if err == nil && !sess.stopped && m.store != nil {
				summary := ""
				if loop.finishResult != nil {
					summary = *loop.finishResult
				}
				if summary == "" {
					// 兜底：取最后一条非空 assistant 消息作为完成报告
					summary = extractLastAssistantContent(msgs)
				}
				if summary != "" {
					if len(summary) > 1000 {
						summary = summary[:1000] + "…（完成报告已截断）"
					}
					feedbackMsg := Message{
						Role:    RoleUser,
						Content: "【上一轮自主执行完成报告】\n" + summary + "\n（以上为上一轮自主执行的最终完成报告。后续任务请在此基础上继续推进。）",
					}
					msgs = append(msgs, feedbackMsg)
					loop.History = msgs

					// 持久化到 store，确保页面刷新后仍可读取
					_ = m.store.AppendMessage(convID, feedbackMsg, nil)
				}
			}

		} else {
			msgs, err = loop.Run(runCtx, task, nil)
		}
		sess.History = msgs

		// ★ 持久化执行日志到磁盘（无论自主还是非自主，保证下轮能感知本轮分析和操作）
		if opts.WorkspaceRoot != "" {
			SaveExecutionLog(opts.WorkspaceRoot, convID, loop.GetExecutionLog())
		}

		// ★ 自动完成所有未完成任务（自然结束）
		// 确保任务列表与运行状态一致，避免前端显示未完成的遗留任务
		if opts.WorkspaceRoot != "" {
			tm := UseTaskManager(opts.WorkspaceRoot)
			tm.CompleteAllInProgress(convID)
			// 发射 Event 通知前端刷新计划面板
			select {
			case sess.Events <- Event{Type: EventNotice, Content: "任务已全部标记为完成"}:
			default:
			}
		}

		// ★ 直接持久化：已由 OnBatchPersist（PersistNewMessages）在 loop.Run 内部增量处理，
		// 此处不再重复写入，避免并发竞态导致用户消息重复存盘。

		// ── ReplaceHistory 已移除 ──
		// 之前在此处调用了 ReplaceHistory，将压缩后的精简版历史写入 DB，
		// 导致下个 loop 从数据库加载全量历史时拿到的是摘要版，Agent 遗忘过多。
		// 现在 DB 中始终保留完整原始消息，压缩仅在运行时内存中进行，
		// 下次 loop 从 DB LoadAll 时仍能拿到完整上下文。

		// ★ 不复存在：MergeLastAssistantRun 已移除。
		// 之前这里调用了 MergeLastAssistantRun 将末尾连续多条 assistant 消息合并成一条，
		// 导致：
		//   1. 各轮次的思考（reasoning）丢失——只有第一条 assistant 的 reasoning 保留
		//   2. 各轮次正文被简单拼接成一个长字符串
		//   3. 所有工具调用聚到同一条 assistant 中，失去了"思考→工具→结果→再思考"的时序结构
		//   4. 前端拿到后无法展示不同轮次的 thinking 段
		// 现在每轮 assistant 独立存储，由 SegmentsFromMessage 在读取时通过 look-ahead
		// 自动将 tool_result 嵌入对应 tool_call segment，形成完整的"工具+结果"链路。

		// ★ Auto commit：任务正常完成时自动 git 提交
		if opts.AutoCommit && err == nil {
			result := ""
			if loop.finishResult != nil {
				result = *loop.finishResult
			}
			commitMsg := loop.Registry.CommitMessage
			doAutoCommit(opts.WorkspaceRoot, task, result, commitMsg)
		}

		// ★ 错误/停止处理：确保前端总收到结束信号（防止 assistant 消息永久 loading）
		// Loop.Run 内部已对多数错误发射 EventError，此处补发 Loop 未覆盖的信号：
		//   1. 用户主动停止 → 发 EventDone (stopped)，触发前端 processAgentDone 清理
		//   2. 异常终止（非用户停止）→ 用 select+default 非阻塞发送 EventError
		//      （不阻塞关闭流程，且通道满时静默放弃，无 goroutine 泄露风险）
		//   3. Loop 正常完成（err==nil）→ EventDone 已由 Loop.Run 内部发射，无需重复
		if sess.stopped {
			select {
			case sess.Events <- Event{Type: EventDone, DoneReason: "stopped", Content: "用户终止了任务"}:
			default:
			}
		} else if err != nil {
			finalErr := err.Error()
			select {
			case sess.Events <- Event{Type: EventError, Content: finalErr}:
			default:
			}
		}
	}()

	return nil
}

// closeChan 安全关闭 channel（已关闭时 recover 防 panic，供 fan-out 与 Unsubscribe 并发调用）。
func closeChan(ch chan Event) {
	defer func() { recover() }()
	close(ch)
}

// evictIfNeeded 当已结束会话数超过 maxSessions 时淘汰最早的（调用方需持锁）。
// 仅淘汰 Running=false 的会话；运行中的不淘汰。
func (m *SessionManager) evictIfNeeded() {
	if len(m.sessions) <= m.maxSessions {
		return
	}
	// 找最早结束的非运行会话淘汰
	var oldestID string
	var oldestTime time.Time
	for id, s := range m.sessions {
		if s.Running {
			continue
		}
		if oldestID == "" || s.StartedAt.Before(oldestTime) {
			oldestID = id
			oldestTime = s.StartedAt
		}
	}
	if oldestID != "" {
		delete(m.sessions, oldestID)
	}
}

// Stop 取消指定会话的运行（用户主动停止）。不删除会话，保留历史。
func (m *SessionManager) Stop(convID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sess, ok := m.sessions[convID]
	if !ok {
		return
	}
	sess.stopped = true
	sess.Running = false
	sess.Cancel()
}

// Subscribe 订阅指定会话的事件流（fan-out 分发）。
// 返回一个缓冲 100 的只读 channel；会话不存在返回 nil。
// 若会话已结束（fan-out 已退出），返回一个已关闭的 channel（订阅者立即收到流结束信号）。
// 调用方应在结束后调 Unsubscribe 释放资源。
func (m *SessionManager) Subscribe(convID string) <-chan Event {
	m.mu.Lock()
	sess, ok := m.sessions[convID]
	m.mu.Unlock()
	if !ok {
		return nil
	}
	ch := make(chan Event, 100)
	sess.subMu.Lock()
	if sess.finished {
		// fan-out 已退出：直接返回已关闭 channel，避免永不关闭的僵尸订阅者
		sess.subMu.Unlock()
		close(ch)
		return ch
	}
	sess.subscribers = append(sess.subscribers, ch)
	sess.subMu.Unlock()
	return ch
}

// Unsubscribe 取消订阅并 close 该 channel（从 subscribers 移除）。
// 传入 Subscribe 返回的只读 channel；若不存在则无操作。
func (m *SessionManager) Unsubscribe(convID string, ch <-chan Event) {
	m.mu.Lock()
	sess, ok := m.sessions[convID]
	m.mu.Unlock()
	if !ok {
		return
	}
	sess.subMu.Lock()
	defer sess.subMu.Unlock()
	for i, sub := range sess.subscribers {
		// 用 reflect 比较底层指针：双向 chan 与只读 <-chan 类型不同无法直接 ==，
		// 但底层指向同一 channel 对象时 reflect.Pointer 相等。
		if reflect.ValueOf(sub).Pointer() == reflect.ValueOf(ch).Pointer() {
			sess.subscribers = append(sess.subscribers[:i], sess.subscribers[i+1:]...)
			closeChan(sub)
			return
		}
	}
}

// SessionStatus 会话状态快照。
type SessionStatus struct {
	Running       bool      `json:"running"`
	Stopped       bool      `json:"stopped"`
	StartedAt     time.Time `json:"startedAt"`
	WorkspaceRoot string    `json:"workspaceRoot,omitempty"`
}

// GetStatus 查询指定会话的完整运行状态，会话不存在时返回 nil。
func (m *SessionManager) GetStatus(convID string) *SessionStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[convID]
	if !ok {
		return nil
	}
	return &SessionStatus{
		Running:       sess.Running,
		Stopped:       sess.stopped,
		StartedAt:     sess.StartedAt,
		WorkspaceRoot: sess.WorkspaceRoot,
	}
}

// IsRunning 查询指定会话是否在运行。
func (m *SessionManager) IsRunning(convID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[convID]
	if !ok {
		return false
	}
	return sess.Running
}

// GetHistory 返回指定会话的已结束 History 副本（深复制 + 剥离 Reasoning）。
// 会话不存在时 fallback 到 MessageStore 加载完整历史（含 ToolCalls/Reasoning）。
// 若会话刚结束（EventDone 已发射但 Running 尚未清除），
// 从 Loop.History 读取以消除竞态。
func (m *SessionManager) GetHistory(convID string) []Message {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	store := m.store
	m.mu.RUnlock()
	if !ok {
		// 会话不在内存中：fallback 到 store 加载完整历史
		if store == nil {
			return nil
		}
		msgs, err := store.LoadAll(convID)
		if err != nil {
			fmt.Printf("[session] GetHistory 从 store 加载 %s 失败: %v\n", convID, err)
			return nil
		}
		return msgs
	}
	if !sess.Running && sess.History != nil {
		return copyHistoryNoReasoning(sess.History)
	}
	// 正在运行中：从 Loop.History 读取实时历史（EventDone 已发射但 Running 未清除时使用）
	if sess.Loop != nil && sess.Loop.History != nil {
		return copyHistoryNoReasoning(sess.Loop.History)
	}
	return nil
}

// GetCurrentHistory 返回指定会话的当前运行中 History 副本（深复制 + 剥离 Reasoning）。
// 会话正在运行时读取 Loop 的实时历史（含本轮所有 tool_calls/tool_results）。
// 会话不存在或 Loop 尚未开始返回 nil。
func (m *SessionManager) GetCurrentHistory(convID string) []Message {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if !ok || sess.Loop == nil {
		return nil
	}
	return copyHistoryNoReasoning(sess.Loop.History)
}

// copyHistoryNoReasoning 深复制消息列表并剥离所有 Reasoning（防止 reasoning_content
// 通过 historyCache 回传给 LLM API 时被错误解释为用户输入，导致 self-loop）。
func copyHistoryNoReasoning(hist []Message) []Message {
	out := make([]Message, len(hist))
	for i, m := range hist {
		out[i] = m
		out[i].Reasoning = ""
	}
	return out
}

// copyHistoryRaw 深复制消息列表，保留全部字段（含 Reasoning）。
// 供 persistRunningHistories 和 EventDone 持久化时使用，
// 确保 SegmentsFromMessage 能读取 Reasoning 创建 thinking segment。
func copyHistoryRaw(hist []Message) []Message {
	out := make([]Message, len(hist))
	copy(out, hist)
	return out
}

// SetAutoReview 实时更新指定会话的审核开关（只影响当前对话）。
// 用户在工具栏切换 autoReview 时立即生效，无需新消息。
func (m *SessionManager) SetAutoReview(convID string, v bool) {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if ok && sess.Loop != nil {
		sess.Loop.SetAutoReview(v)
		fmt.Printf("[session] 实时更新审核开关 conv=%s autoReview=%v\n", convID, v)
	}
}

// GetCurrentHistoryRaw 返回指定会话的当前运行中 History 深复制副本（保留 Reasoning）。
// 供 persistRunningHistories 持久化时使用，确保 SegmentsFromMessage 能读取 reasoning 创建 thinking segment。
// 优先使用 currentMsgs（运行中每轮更新），其次是 Loop.History（Run 退出后由 defer 设置）。
func (m *SessionManager) GetCurrentHistoryRaw(convID string) []Message {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if !ok || sess.Loop == nil {
		return nil
	}
	if sess.Loop.currentMsgs != nil {
		return copyHistoryRaw(sess.Loop.currentMsgs)
	}
	if sess.Loop.History != nil {
		return copyHistoryRaw(sess.Loop.History)
	}
	return nil
}
// 页面刷新后恢复时使用。会话不存在或 Loop 尚未开始返回 nil。
func (m *SessionManager) GetCurrentCompressedSummaries(convID string) []string {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if !ok || sess.Loop == nil {
		return nil
	}
	out := make([]string, len(sess.Loop.CompressedSummaries))
	copy(out, sess.Loop.CompressedSummaries)
	return out
}

// ListRunning 返回所有 Running=true 的 convID 列表。
func (m *SessionManager) ListRunning() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.sessions))
	for id, s := range m.sessions {
		if s.Running {
			out = append(out, id)
		}
	}
	return out
}

// SendAnswer 向指定会话发送 ask_user 的用户回答。
// 会话不存在或未运行返回错误。
func (m *SessionManager) SendAnswer(convID string, answer string) error {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if !ok {
		return ErrSessionNotFound
	}
	if !sess.Running {
		return ErrSessionNotRunning
	}
	select {
	case sess.askCh <- answer:
		return nil
	default:
		return errors.New("回答通道已满（可能已有待处理回答）")
	}
}

// Approve 向指定会话发送审批结果（true=允许，false=拒绝）。
// 会话不存在或未运行返回错误。
func (m *SessionManager) Approve(convID string, approved bool) error {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if !ok {
		return ErrSessionNotFound
	}
	if !sess.Running {
		return ErrSessionNotRunning
	}
	select {
	case sess.approvalCh <- approved:
		return nil
	default:
		return errors.New("审批通道已满（可能已有待处理审批）")
	}
}

// SendFeedback 向指定会话发送运行时反馈（补充/纠正）。
// 每轮 LLM 调用前由 Loop.OnFeedback 检查并注入。会话不存在返回错误。
func (m *SessionManager) SendFeedback(convID string, feedback string) error {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if !ok {
		return ErrSessionNotFound
	}
	if !sess.Running {
		return ErrSessionNotRunning
	}
	select {
	case sess.feedbackCh <- feedback:
		return nil
	default:
		return errors.New("反馈通道已满（缓冲 5 条）")
	}
}

// SubscribeAll 订阅所有会话事件（全局 fan-out）。
// 供 WebSocket 端点使用：单一连接接收所有并行会话事件，每条 GlobalEvent 携带 convID。
// 调用方应在连接关闭时调 UnsubscribeAll 释放资源。
func (m *SessionManager) SubscribeAll() <-chan GlobalEvent {
	ch := make(chan GlobalEvent, 200)
	m.globalSubMu.Lock()
	m.globalSubscribers = append(m.globalSubscribers, ch)
	m.globalSubMu.Unlock()
	return ch
}

// UnsubscribeAll 取消全局订阅并 close 该 channel。
// 传入 SubscribeAll 返回的只读 channel；若不存在则无操作。
func (m *SessionManager) UnsubscribeAll(ch <-chan GlobalEvent) {
	m.globalSubMu.Lock()
	defer m.globalSubMu.Unlock()
	for i, sub := range m.globalSubscribers {
		if reflect.ValueOf(sub).Pointer() == reflect.ValueOf(ch).Pointer() {
			m.globalSubscribers = append(m.globalSubscribers[:i], m.globalSubscribers[i+1:]...)
			closeGlobalChan(sub)
			return
		}
	}
}

// extractLastAssistantContent 从消息历史中提取最后一条非空 assistant 消息的内容。
// 用于自主模式完成后注入对话历史，只保留最终的完成报告而非完整上下文。
func extractLastAssistantContent(msgs []Message) string {
	if len(msgs) == 0 {
		return ""
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleAssistant && strings.TrimSpace(msgs[i].Content) != "" {
			return strings.TrimSpace(msgs[i].Content)
		}
	}
	return ""
}

// closeGlobalChan 安全关闭全局 channel（已关闭时 recover 防 panic）。
func closeGlobalChan(ch chan GlobalEvent) {
	defer func() { recover() }()
	close(ch)
}

// ListRunningByWorkspace 返回指定工作区下所有 Running=true 的 convID 列表。
// 供前端状态指示器使用：显示每个工作区有多少 agent 在运行。
func (m *SessionManager) ListRunningByWorkspace(wsRoot string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.sessions))
	for id, s := range m.sessions {
		if s.Running && s.WorkspaceRoot == wsRoot {
			out = append(out, id)
		}
	}
	return out
}

// GetWorkspaceRoot 返回指定会话的 WorkspaceRoot。
// 会话不存在返回空字符串。
func (m *SessionManager) GetWorkspaceRoot(convID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[convID]
	if !ok {
		return ""
	}
	return sess.WorkspaceRoot
}

// doAutoCommit 在任务完成后自动执行 git add + git commit。
// 遍历所有工作区目录（WorkspaceRoots），对每个 git 仓库提交改动。
// 从 agent 的 result（最终输出）中提取第一行实质性内容作为 commit message，
// 不直接使用用户消息，确保提交信息反映 agent 实际完成的工作。
// 自动设置 git user config 避免因全局配置缺失导致提交失败。
// 执行失败时只日志不 panic（不影响 agent 主流程）。
func doAutoCommit(primaryRoot, task, result, commitMsg string) {
	// 收集所有要提交的目录：用 WorkspaceRoots（全量），兜底 primaryRoot
	roots := WorkspaceRoots
	if len(roots) == 0 && primaryRoot != "" {
		roots = []string{primaryRoot}
	}
	if len(roots) == 0 {
		return
	}

	// 优先使用 agent 通过 generate_commit_message 工具生成的提交信息
	msg := strings.TrimSpace(commitMsg)
	if msg == "" {
		msg = extractSummary(result)
	}
	if msg == "" {
		msg = strings.TrimSpace(task)
	}
	if len(msg) > 72 {
		msg = msg[:72]
		if idx := strings.LastIndex(msg, " "); idx > 30 {
			msg = msg[:idx]
		}
	}
	if msg == "" {
		msg = "auto commit"
	}

	for _, root := range roots {
		if root == "" {
			continue
		}
		// 检查是否为 git 仓库
		gitDir := filepath.Join(root, ".git")
		if fi, err := os.Stat(gitDir); err != nil || !fi.IsDir() {
			continue // 不是 git 仓库，跳过
		}

		add := exec.Command("git", "add", "-A")
		add.Dir = root
		if out, err := add.CombinedOutput(); err != nil {
			fmt.Printf("[auto-commit] git add 失败 (%s): %v\n%s\n", root, err, string(out))
			continue
		}

		commit := exec.Command("git",
			"-c", "user.name=Pairode",
			"-c", "user.email=agent@paircode.dev",
			"commit", "-m", "auto: "+msg)
		commit.Dir = root
		if out, err := commit.CombinedOutput(); err != nil {
			errStr := string(out)
			if strings.Contains(errStr, "nothing to commit") {
				exec.Command("git", "reset", "HEAD").Run()
				continue
			}
			fmt.Printf("[auto-commit] git commit 失败 (%s): %v\n%s\n", root, err, errStr)
			exec.Command("git", "reset", "HEAD").Run()
			continue
		}
		fmt.Printf("[auto-commit] ✅ 已自动提交 (%s): auto: %s\n", root, msg)
	}
}

// extractSummary 从 agent 输出的结果中提取第一行实质性内容。
// 跳过 markdown 标题、空行、纯标点行，找到第一段有实际文字的句子。
func extractSummary(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		l := strings.TrimSpace(line)
		// 跳过 markdown 标题、分割线、空行
		l = strings.TrimLeft(l, "#*> \t")
		if l == "" || len(l) < 4 {
			continue
		}
		// 跳过纯标点/符号行（如 "---"、"==="、"****"）
		hasLetter := false
		for _, r := range l {
			if r > 127 || unicode.IsLetter(r) {
				hasLetter = true
				break
			}
		}
		if !hasLetter {
			continue
		}
		// 取第一句（到句号/换行），最长 72 字
		if idx := strings.IndexAny(l, "。.！!\n"); idx > 0 {
			l = l[:idx+1]
		}
		return strings.TrimSpace(l)
	}
	return ""
}

// AppendPersistedMessage 追加一条消息到 MessageStore（持久化）。
// 供 web 层 startEventPersistWorker 在 EventDone 等时机调用，
// 将 loop.History 中的新消息增量持久化。
// 若 store 为 nil 则无操作（web 层未注入时静默跳过）。
func (m *SessionManager) AppendPersistedMessage(convID string, msg Message, segments []Segment) error {
	m.mu.RLock()
	store := m.store
	m.mu.RUnlock()
	if store == nil {
		return nil
	}
	return store.AppendMessage(convID, msg, segments)
}

// AppendPersistedUserMessage 追加一条用户消息到 MessageStore（便捷封装）。
// 供 web 层 handleChatSend 在调 Start 前调用，先持久化用户消息。
// 若 store 为 nil 则无操作。
func (m *SessionManager) AppendPersistedUserMessage(convID, content string) error {
	m.mu.RLock()
	store := m.store
	m.mu.RUnlock()
	if store == nil {
		return nil
	}
	return store.AppendUserMessage(convID, content)
}

// startPersistWorker 启动内部持久化 worker goroutine（在 SetWorkspaceRoot 注入 store 后自动调用）。
func (m *SessionManager) startPersistWorker() {
	ch := make(chan GlobalEvent, 200)
	m.globalSubMu.Lock()
	m.globalSubscribers = append(m.globalSubscribers, ch)
	m.globalSubMu.Unlock()

	go func() {
		for ge := range ch {
			if ge.Event.Type == EventUsage && ge.Event.Usage != nil {
				if store := m.Store(); store != nil {
					_ = store.SetCtxStats(ge.ConvID, ge.Event.Usage)
				}
			}
		}
	}()
}
