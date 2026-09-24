package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hoonfeng/paircode/pkg/executil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	pkgdb "github.com/hoonfeng/paircode/pkg/db"
)

// hideCmd 隐藏子进程控制台窗口（Windows；非 Windows 原样返回）。
// 父进程无控制台（后台/服务方式启动）时，console 子进程会自己弹窗，须显式隐藏。
func hideCmd(c *exec.Cmd) *exec.Cmd {
	if runtime.GOOS == "windows" {
		if c.SysProcAttr == nil {
			c.SysProcAttr = &syscall.SysProcAttr{}
		}
		executil.HideWindow(c)
	}
	return c
}

// Compressor 上下文压缩器（语义别名，复用 Provider 接口）。
// 非空时用轻量压缩模型做 LLM 摘要；空则规则式摘要。
type Compressor = Provider

// LoopOpts 创建 Loop 所需的全部参数（供 SessionManager.Start 使用）。
// 把原本散落在 web 层的 Loop 构造逻辑收敛到一处，便于并行会话统一创建。
type LoopOpts struct {
	Provider Provider  // LLM 提供方
	Registry *Registry // 工具注册表（Start 会在此注册 ask_user 工具）
	System   string    // 系统提示词
	// ★ 2026-09-12：「最大迭代数」配置项已移除——段内迭代安全上限由工具预算派生
	//   （tool_budget.go IterationLimit），段结束由预算 + 自动续跑（下端两项）负责。
	// StepBudget 单段（一次 Run）步数预算（一步 = 一次 LLM 调用）：0=默认 120；负数=不限。
	//   ★ 2026-09-12 配置化：来源 agentloop 插件注册的 stepBudget
	//   （pluginSettings.agentloop，装配时经 overrides 透传；见 tool_budget.go）。
	//   ★ 双闸门：与 ToolCallBudget 任一达上限即结束本段并自动续跑。
	StepBudget int
	// ToolCallBudget 单段（一次 Run）工具调用轮次预算：0=默认 120；负数=不限。
	//   ★ 2026-09-12 配置化：来源 agentloop 插件注册的 toolCallBudget
	//   （pluginSettings.agentloop，装配时经 overrides 透传；见 tool_budget.go）。
	ToolCallBudget int
	// MaxToolBudgetSegments 单轮任务最多自动续跑段数：0=默认 20。
	//   ★ 2026-09-12 配置化：来源 agentloop 插件注册的 maxToolBudgetSegments
	//   （pluginSettings.agentloop，装配时经 overrides 透传；见 tool_budget.go）。
	MaxToolBudgetSegments int
	MaxContextTokens      int        // 上下文 token 上限（>0 启用压缩）
	Compressor            Compressor // 上下文压缩器（可空）
	History               []Message  // 初始历史（首次为空；续跑时传上一轮 History）。传入时可能已被 CondenseHistory 压缩。
	HistoryOriginal       []Message  // 原始未压缩历史（与 History 对应，用于持久化而非 LLM 上下文）。
	CompressedSummaries   []string   // 已持久化的压缩摘要（页面刷新后恢复）
	Autonomous            bool       // 自主模式标志
	MaxAutonomousMinutes  int        // 自主模式时间预算（分钟，0=无限制）
	CheckpointInterval    int        // 检查点间隔（迭代数，0=默认5）
	WorkspaceRoot         string     // 工作区根路径（用于跨工作区并行对话的状态指示与隔离）
	// ConvID 会话标识（仅诊断用：缓存前缀诊断据此区分会话——同一会话内的
	// 视图重排（会话交接生效/刷新、跨轮精简）应报为「前缀断裂」并给出原因，
	// 而非被误判为「新会话」跳过比较）。
	ConvID string
	// ReviewMode 审核模式："auto"=AI审核, "manual"=手动审批, "off"=全部放行。
	// "auto"=Loop 内部 AI 审核把关写操作；"off"=全部放行（不经过任何审核）；"manual"=人工审批（前端弹窗）。
	ReviewMode string
	// ReviewBlacklist 审核黑名单：命中此列表的工具需要审核（为空=全部工具按 ReviewMode 审核）。
	ReviewBlacklist []string
	// ReviewWhitelist 审核白名单：命中此列表的工具跳过审核（黑名单优先）。
	ReviewWhitelist []string
	// ReviewProvider 审核模型的 Provider（ReviewMode="auto" 时用）。Loop 内部用它懒建 Reviewer。
	ReviewProvider Provider
	// MaxSuperviseRounds 自主模式监督轮数上限（0/负 = 内核默认 DefaultMaxSuperviseRounds）。
	//   ★ 2026-09-21：旧自主模式的 PlanProvider（外层规划 Loop / 设计者-执行者双层架构）
	//   已随该架构删除——自主模式现由插件决策器驱动（autopilot.go），监督者回合经
	//   ctx.subagent 能力复用本会话模型（不再需要独立规划模型槽位）。
	MaxSuperviseRounds int
	// ★ 2026-09-21 SkipPluginAssembly 跳过插件装配器链（子 agent 回合专用，见 subagent.go）：
	//   子 agent 由插件 JS 回调栈内的宿主能力创建（如 autopilot 决策器经 ctx.subagent.run），
	//   此时调用插件装配器要取该插件 VM 锁——若正是当前持锁插件即自死锁（goja 锁非重入）。
	SkipPluginAssembly bool
	// ★ 2026-09-13：原 ResumeContext（会话连贯性上下文 → 背景上下文快照）已随该实现删除。
	//   会话连贯性上下文改由提交消息（handoff.go 交接视图）与任务清单工具承载；
	//   历史进度/摘要等不再每轮注入（每轮必变会稀释前缀缓存命中率）。
}

// GlobalEvent 是全局订阅者收到的事件：携带 convID 用于前端路由。
// WebSocket 端点通过 SubscribeAll 获取所有会话事件，每条都带 convID。
type GlobalEvent struct {
	ConvID string
	Event  Event
}

// ApprovalResult 审批结果（由用户通过前端 ApprovalBar 提交）。
type ApprovalResult struct {
	Approved bool
	Reply    string // 用户输入的回复内容（拒绝时填写原因，允许时可选）
}

// AskAnswer 一条 ask_user 回答（Round3 ⑤ 多问题：ID 关联问题；单问题 ID 为空）。
type AskAnswer struct {
	ID     string `json:"id"`
	Answer string `json:"answer"`
}

var DefaultApproved = ApprovalResult{Approved: true, Reply: ""}
var DefaultDenied = ApprovalResult{Approved: false, Reply: "用户拒绝了此操作"}

// ─── 全局会话管理器引用（web 层启动时注入一次）─────────────────
// ★ 2026-09：原 subagent_registry.go 负责，注册表删除后迁到此处。
// 消费方：工具结果路由（storeForConvLookup）、会话唤醒投递（session_wake.go）。

var (
	globalSessionMgrMu sync.RWMutex
	globalSessionMgr   *SessionManager
)

// SetGlobalSessionManager 注入全局会话管理器（web 层启动时调用一次）。
func SetGlobalSessionManager(m *SessionManager) {
	globalSessionMgrMu.Lock()
	globalSessionMgr = m
	globalSessionMgrMu.Unlock()
}

// GlobalSessionManager 取全局会话管理器（未注入返回 nil）。
func GlobalSessionManager() *SessionManager {
	globalSessionMgrMu.RLock()
	defer globalSessionMgrMu.RUnlock()
	return globalSessionMgr
}

// Session 一次 agent 运行会话。从 web 层 webAgentSession 下沉而来，
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
	// ★ Round3 ⑤：askCh 结构化（多问题 answers 数组；单问题=单元素数组，向后兼容）
	askCh      chan []AskAnswer    // ask_user 工具阻塞等用户回答
	approvalCh chan ApprovalResult // Approve 钩子阻塞等用户裁决
	feedbackCh chan string         // OnFeedback 每轮 LLM 调用前检查

	// 订阅者 fan-out：多个 SSE 客户端可订阅同一会话事件
	subscribers []chan Event
	subMu       sync.RWMutex
	finished    bool // fan-out 已退出（subMu 保护）；true 时新订阅直接返回已关闭 channel

	// stopped 标记用户主动停止（区别于自然结束，避免多发错误事件）
	stopped bool

	// ★ 2026-09-25 askPending：当前是否有 ask_user 真正阻塞等待用户回答
	//   （= askCh 上有消费者）。置位处：会话级 ask_user Handler 与 WaitAnswers。
	//   用途：SendAnswers 判定「无消费者」——旧行为下 askCh（1 格缓冲）投递
	//   成功但无人消费，回答被静默丢弃：前端显示「已回答」，历史却查无此答。
	askPending atomic.Bool
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

	store ConversationStore // 消息持久化存储（通过 SetWorkspaceRoot 注入；＝当前工作区的缓存 store）

	// ds 统一 SQLite 数据库实例，codegraph/其他组件共用同一连接（＝当前工作区的缓存 DB）。
	ds *pkgdb.SQLiteDB

	// ★ 2026-08-23 多工作区隔离（重大 BUG 修复）：按工作区根缓存 store/DB。
	//   此前 SetWorkspaceRoot 每切一次工作区就替换全局 store 并关闭旧 DB——
	//   正在运行的会话（绑定旧工作区）继续持久化/查询会落进新工作区，
	//   工具执行由全局根（WorkspaceRoots）驱动直接串台。
	//   缓存后：switch 只切换「当前指针」，已存在根保持句柄不死；
	//   删除工作区时经 CloseWorkspaceDB 显式关闭（Windows 文件占用问题保留解决路径）。
	wsMu    sync.Mutex
	stores  map[string]*MessageStore
	dbs     map[string]*pkgdb.SQLiteDB
	curRoot string

	// 内部持久化 worker 状态
	persistWorkerStarted bool

	// starting 启动中会话集合（异步 Start 防重：HTTP 立即返回后 Start 在后台执行，
	// 同一 conv 的二次请求在此拒绝，避免并发双启）。
	startingMu sync.Mutex
	starting   map[string]bool

	// OnDone 会话完成回调（由 web 层设置，用于生成对话摘要等副作用）。
	// 在 Session goroutine 写盘后调用，convID 为刚结束的会话 ID。
	OnDone func(convID string)
}

// NewSessionManager 创建会话管理器，默认保留 100 个已结束会话。
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:    make(map[string]*Session),
		maxSessions: 100,
		starting:    make(map[string]bool),
		stores:      make(map[string]*MessageStore),
		dbs:         make(map[string]*pkgdb.SQLiteDB),
	}
}

// SetWorkspaceRoot 设置「当前工作区根」，切换当前 store/DS 指针（惰性缓存，不关闭旧句柄）。
// ★ 2026-08-23 多工作区隔离：不再替换全局 store / 关闭旧 DB——正在运行的会话
//
//	（绑定启动时工作区）继续用自己根的 store/DB（storeFor 按根路由），切换只影响新会话。
//	工作区删除时经 CloseWorkspaceDB(root) 显式关闭该根句柄（Windows 文件占用）。
func (m *SessionManager) SetWorkspaceRoot(root string) {
	m.wsMu.Lock()
	m.curRoot = root
	m.wsMu.Unlock()

	m.store = m.storeFor(root)
	m.ds = m.dbFor(root)

	m.mu.Lock()
	shouldStart := !m.persistWorkerStarted
	if shouldStart {
		m.persistWorkerStarted = true
	}
	m.mu.Unlock()

	if shouldStart {
		m.startPersistWorker()
	}
}

// storeFor 返回指定工作区根的消息存储（惰性创建并缓存；root 为空返回 nil）。
func (m *SessionManager) storeFor(root string) *MessageStore {
	if root == "" {
		return nil
	}
	m.wsMu.Lock()
	if s, ok := m.stores[root]; ok {
		m.wsMu.Unlock()
		return s
	}
	m.wsMu.Unlock()

	// 锁外创建（NewMessageStore 只做目录准备/读 index，无并发副作用；double-check 防重复缓存）
	s := NewMessageStore(root)
	m.wsMu.Lock()
	if exist, ok := m.stores[root]; ok {
		m.wsMu.Unlock()
		return exist
	}
	m.stores[root] = s
	m.wsMu.Unlock()
	return s
}

// dbFor 返回指定工作区根的 SQLite（惰性创建并缓存；root 为空返回 nil）。
func (m *SessionManager) dbFor(root string) *pkgdb.SQLiteDB {
	if root == "" {
		return nil
	}
	m.wsMu.Lock()
	if d, ok := m.dbs[root]; ok {
		m.wsMu.Unlock()
		return d
	}
	m.wsMu.Unlock()

	var ds *pkgdb.SQLiteDB
	dbPath := filepath.Join(root, ".pair", "pair.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err == nil {
		if d, err := pkgdb.NewSQLiteDB(dbPath); err == nil {
			ds = d
		} else {
			fmt.Printf("[session] 打开 SQLite 数据库失败 %s: %v\n", root, err)
		}
	} else {
		fmt.Printf("[session] 创建数据库目录失败 %s: %v\n", root, err)
	}
	m.wsMu.Lock()
	if exist, ok := m.dbs[root]; ok {
		m.wsMu.Unlock()
		return exist
	}
	m.dbs[root] = ds
	m.wsMu.Unlock()
	return ds
}

// StoreFor 返回指定工作区根的消息存储（web 层按请求工作区路由；root 为空回落当前）。
func (m *SessionManager) StoreFor(root string) ConversationStore {
	if root == "" {
		return m.Store()
	}
	return m.storeFor(root)
}

// FindConversation 跨已打开的 store 查找会话（workspaceRoot 参数缺失/错位时的兜底：
// 前端旧版本/参数错位可能把配置名传进 workspaceRoot，StoreFor 落空导致误报「会话不存在」）。
// 返回 (会话元数据, 归属工作区根)；找不到返回 (nil, "")。
// 只读已缓存的 store，不新建；正常路径（StoreFor 命中）不经过本方法。
func (m *SessionManager) FindConversation(convID string) (*ConversationMeta, string) {
	if convID == "" {
		return nil, ""
	}
	// 1. 当前 store 优先
	if cur := m.Store(); cur != nil {
		if meta, err := cur.GetConversation(convID); err == nil && meta != nil {
			m.wsMu.Lock()
			curRoot := m.curRoot
			m.wsMu.Unlock()
			return meta, curRoot
		}
	}
	// 2. 遍历已打开隔离 store（锁内取引用快照，锁外查询——MessageStore 自带 indexMu）
	m.wsMu.Lock()
	type cand struct {
		root string
		st   *MessageStore
	}
	cands := make([]cand, 0, len(m.stores))
	for root, st := range m.stores {
		cands = append(cands, cand{root, st})
	}
	m.wsMu.Unlock()
	for _, c := range cands {
		if c.st == nil {
			continue
		}
		if meta, err := c.st.GetConversation(convID); err == nil && meta != nil {
			return meta, c.root
		}
	}
	return nil, ""
}

// RawDBFor 返回指定工作区根的 *sql.DB（codegraph 按会话根路由，防止切换后串库）。
func (m *SessionManager) RawDBFor(root string) *sql.DB {
	ds := m.dbFor(root)
	if ds == nil {
		return nil
	}
	return ds.RawDB().(*sql.DB)
}

// CloseWorkspaceDB 关闭并丢弃指定工作区根的 store/DB（工作区删除时调用；
// Windows 上不关闭句柄会导致删除失败）。
func (m *SessionManager) CloseWorkspaceDB(root string) {
	m.wsMu.Lock()
	store := m.stores[root]
	delete(m.stores, root)
	ds := m.dbs[root]
	delete(m.dbs, root)
	if m.curRoot == root {
		m.curRoot = ""
	}
	m.wsMu.Unlock()
	_ = store
	if ds != nil {
		_ = ds.Close()
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

// ErrAskNotPending 回答提交时该会话已无 ask_user 在等待（提问超时/被停止）。
// ★ 2026-09-25：此时回答已落盘为会话消息（用户历史可见、下一轮模型可见），
// web 层据此回「已记录为消息」而不是报错——旧行为是投递进缓冲通道后
// 无人消费，回答静默丢失（用户看到「已回答」但历史无记录）。
var ErrAskNotPending = errors.New("提问已结束（超时/停止）：回答已记录为会话消息")

// composePersistMessages 组合持久化消息：已落盘基准 + 锚点之后的新增。
//
// 锚点 = msgs 中最后一条「真实任务」RoleUser（非系统注入消息，识别见
// injected_msg.go / isInjectedUserMessage）——注入物（历史压缩摘要/交接视图）
// 也是 RoleUser，但它们是系统产物（且可能位于任务之后），不能作为锚点
// （否则 tail 为空、其后的真实消息全部丢失）。tail = 锚点之后的所有消息。
//
// ★ 2026-09-11 配套「可变底账」（refreshPersistBase）：分段续跑时基准会推进为
// store 当前内容，「基准 + tail」在每段上继续追加——修复此前固定 originalHist
// 导致第二段全量覆盖写丢「第一段新增消息」的问题。
// 兜底（异常：msgs 无 user 消息）：直接返回 msgs 全量。
func composePersistMessages(base []Message, msgs []Message) []Message {
	lastUserIdx := -1
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleUser && !hasPrefixInjected(msgs[i].Content) {
			lastUserIdx = i
			break
		}
	}
	if lastUserIdx < 0 {
		return msgs
	}
	tail := msgs[lastUserIdx+1:]
	combined := make([]Message, 0, len(base)+len(tail))
	combined = append(combined, base...)
	combined = append(combined, tail...)
	return combined
}

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

	// ★ 2026-08-22 异步 Start 防重：HTTP 层立即返回后 Start 在后台 goroutine 执行，
	//   同一 convID 的并发请求（用户快速连点/重试）在此直接拒绝，避免双 Loop 竞态。
	m.startingMu.Lock()
	if m.starting[convID] {
		m.startingMu.Unlock()
		return ErrSessionRunning
	}
	m.starting[convID] = true
	m.startingMu.Unlock()
	defer func() {
		m.startingMu.Lock()
		delete(m.starting, convID)
		m.startingMu.Unlock()
	}()

	// ★ 2026-08-30 锁粒度收敛（并行会话）：原先 Start 全程持 m.mu 写锁（500 行，
	//   含 CreateLoop → JS 装配器/循环插件 VM 锁 → 可能等另一个会话跑完），
	//   于是「一个对话在跑 → SessionManager 全部读方法（状态/历史/消息落盘）
	//   跟着阻塞」→ 前端任何请求 30s 超时（RWMutex 写锁等待会挡住后续 RLock）。
	//   现在只在「查重」与「挂载会话」两处短暂持锁，重活（store 元数据/CreateLoop/
	//   回调绑定/工具注册）全部在锁外执行。
	// 已有运行中的会话则拒绝（同一对话不可并行跑两个 Loop）
	m.mu.RLock()
	if s, ok := m.sessions[convID]; ok && s.Running {
		m.mu.RUnlock()
		return ErrSessionRunning
	}
	m.mu.RUnlock()

	// 持久化：确保 store 中存在该对话的元数据。
	// ★ 2026-08-23 会话级 store 路由：会话绑定 opts.WorkspaceRoot（启动时工作区），
	//   写入对应根的 MessageStore——运行中切换工作区不影响本会话落盘位置。
	//   store 为 nil（web 层未注入）时跳过；CreateConversation 失败只警告不阻塞 Start。
	sessStore := m.storeFor(opts.WorkspaceRoot)
	if sessStore != nil {
		existing, getErr := sessStore.GetConversation(convID)
		if getErr != nil {
			fmt.Printf("[session] 查询对话 %s 元数据失败: %v\n", convID, getErr)
		} else if existing == nil {
			title := task
			if len(title) > 30 {
				title = title[:30]
			}
			if createErr := sessStore.CreateConversation(convID, title, opts.WorkspaceRoot); createErr != nil {
				fmt.Printf("[session] 创建对话 %s 元数据失败: %v\n", convID, createErr)
			}
		} else {
			// ★ 用户在本对话继续发送消息 → 清除历史中断标记，
			//   表示上一轮的中断正在被继续处理（前端据此隐藏"未完成"提示）。
			if existing.Interrupted {
				// ★ 2026-08-30 修正串库：用会话自己的 store（sessStore）而非全局
				//   m.store——会话工作区与当前全局工作区不同时，旧写法写错库。
				if clrErr := sessStore.SetInterrupted(convID, false); clrErr != nil {
					fmt.Printf("[session] 清除对话 %s 中断标记失败: %v\n", convID, clrErr)
				}
			}
		}
	}

	// 创建会话上下文（使用独立的 context.Background()，不依赖调用方 ctx，
	// 避免 handleChatSend 的 defer setupCancel() 级联取消 runCtx，
	// 导致 Loop 尚未开始就 ctx.Err() != nil 直接返回）。
	runCtx, cancel := context.WithCancel(context.Background())
	// ★ 会话标识注入 ctx 链：Loop.Run → Registry.Execute 同源 ctx，
	//   JS 插件工具包装时可提取 convID（_convID 注入），ask_user/task_create
	//   经会话桥按 convID 路由（多会话并发不串）。
	runCtx = WithSessionConvID(runCtx, convID)
	// ★ 2026-08-23 会话工作区根注入 ctx 链（工作区隔离）：工具执行/持久化按
	//   会话绑定根路由，切换全局工作区不再带偏正在执行的对话。
	if opts.WorkspaceRoot != "" {
		runCtx = WithSessionWorkspaceRoot(runCtx, opts.WorkspaceRoot)
	}

	sess := &Session{
		ConvID:        convID,
		WorkspaceRoot: opts.WorkspaceRoot,
		Events:        make(chan Event, 5000),
		Cancel:        cancel,
		History:       CopyHistory(opts.History), // 深复制避免外部修改
		Running:       true,
		StartedAt:     time.Now(),
		askCh:         make(chan []AskAnswer, 1),
		approvalCh:    make(chan ApprovalResult, 1),
		feedbackCh:    make(chan string, 5),
	}

	// 创建 Loop（★ 走全局 LoopFactory：插件装配器可覆盖参数/实现），挂载回调
	loopHandle, loopErr := CreateLoop(opts)
	if loopErr != nil {
		cancel()
		return fmt.Errorf("创建 agent 循环失败: %w", loopErr)
	}
	loop := loopHandle.Loop()

	// ★ 后端运行统计（run_stats.go，2026-09-12）：挂载本次运行的统计累加器。
	//   步数（beginStep）、LLM 调用次数/生成耗时/token 用量、工具调用次数与耗时
	//   全部由 Loop 在运行期累加（Go 循环与 JS 循环共用同一入口）；
	//   本会话结束（正常/用户停止/异常）时在下方 goroutine 的 defer 中定格并落盘。
	//   ★ 一次会话 = 一次运行：段预算自动续跑的多段在同一 Loop 上累加，不重复计数。
	loop.SetRunStats(BeginRunStatsFor(opts.WorkspaceRoot, convID))

	// ★ 2026-09-13（S3 死代码清理 + 背景上下文实现移除）：此处曾把「重启后首轮」的 goal 段
	//   追加到 loop.ResumeContext，指望经背景上下文快照注入（Round3 ③.1）。快照同步链
	//   2026-09-04 已整体停用、2026-09-13 实现与 Loop.ResumeContext 字段一并删除 → 该写入
	//   早已不会进入任何请求；且不该硬往 system（messages 第一条）拼：goal 段含每轮递增的
	//   Rounds，会在 system 尾部切断系统提示 + 整个历史的前缀缓存。
	//   现状：goal 上下文由**续轮消息**承载（Goal.ContinueMessage 已含目标/阶段/轮次，
	//   在下方的续轮循环里作为新 user 消息注入 = 历史尾部追加，不破前缀）；
	//   重启后首轮若需要目标上下文，agent 可用 get_goal 工具主动查询。
	//   ★ 将来如需恢复注入：请走「历史尾部追加独立消息」，勿回到 ResumeContext / system 拼接。

	// ★ 2026-08-21 LLM 重试通知：Provider 支持 RetryNotifier 时绑定重试回调，
	//   重试期间以 notice 事件推送到前端——用户不再「干等无响应」。
	if rn, ok := loop.Provider.(RetryNotifier); ok {
		rn.SetOnRetry(func(attempt, maxRetries int, errMsg string) {
			msg := fmt.Sprintf("LLM 请求失败（%s），正在自动重试 第%d/%d次…", shortenErr(errMsg, 120), attempt, maxRetries)
			loop.emit(Event{Type: EventNotice, Content: msg})
		})
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
	// ★ 丢弃时打日志（排查「无响应」：前端消费慢/无订阅导致事件丢失的可见化）
	eventDropped := 0
	loop.OnEvent = func(e Event) {
		select {
		case sess.Events <- e:
		default:
			eventDropped++
			if eventDropped == 1 || eventDropped%100 == 0 {
				log.Printf("[session] 事件被丢弃 conv=%s 累计%d条（Events 通道满，前端消费慢/未订阅）", convID, eventDropped)
			}
		}
	}

	// Approve：写类工具执行前阻塞等用户裁决。
	// 先发 EventApproval 通知前端弹审批框，再从 approvalCh 读结果。
	// 当 ReviewMode="manual" 时，由 Loop.Run 内部调用本回调。
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
		// ★ 5 分钟超时：前端断连/页面关闭时不会永久阻塞 Loop goroutine
		select {
		case ar := <-sess.approvalCh:
			return ar.Approved, ar.Reply
		case <-actx.Done():
			return false, "用户取消了操作"
		case <-time.After(5 * time.Minute):
			fmt.Printf("[session] 审批超时 conv=%s tool=%s\n", convID, tc.Function.Name)
			return false, "审批超时（5 分钟），操作已自动拒绝"
		}
	}
	// 审核模式由 Loop 内部自决，外部只需传进来
	loop.SetReviewMode(opts.ReviewMode)
	loop.ReviewProvider = opts.ReviewProvider
	loop.ReviewBlacklist = opts.ReviewBlacklist
	loop.ReviewWhitelist = opts.ReviewWhitelist

	// ★ 2026-09-21 自主模式（插件化）：绑定会话运行环境——插件（如 autopilot）经
	//   ctx.subagent 能力派生「监督者」等子 agent 回合时，复用本会话的模型/工具面/
	//   事件通道/审核策略/精简配置；子 agent 事件带 AgentName 经同一通道下发前端
	//   （看板分区）。会话结束（下方 Run goroutine 的 defer）解除绑定。
	subagentRt := &SubagentRuntime{
		ConvID:           convID,
		WorkspaceRoot:    opts.WorkspaceRoot,
		Provider:         loop.Provider,
		Registry:         opts.Registry,
		System:           opts.System,
		Events:           sess.Events,
		Approve:          loop.Approve,
		ReviewMode:       opts.ReviewMode,
		ReviewProvider:   opts.ReviewProvider,
		ReviewBlacklist:  opts.ReviewBlacklist,
		ReviewWhitelist:  opts.ReviewWhitelist,
		Compressor:       opts.Compressor,
		MaxContextTokens: opts.MaxContextTokens,
		StepBudget:       opts.StepBudget,
		ToolCallBudget:   opts.ToolCallBudget,
		ParentCtx:        runCtx,
	}
	unregisterSubagent := RegisterSubagentRuntime(subagentRt)

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
	// ★ 2026-08-30：用会话自己的 store（sessStore = opts.WorkspaceRoot 路由）而非全局
	//   m.store——会话运行中用户切换工作区时，旧写法会把消息落到新工作区
	//   的库（串库）；Start 也不再全程持锁，无需旧的“避开 m.Store() 死锁”约束。
	// ★ 2026-09-11 持久化基准（可变底账）：修「多段续跑全量覆盖丢消息」。
	//   历史：combined = originalHist(固定) + tail(最后一条真实 user 之后)。
	//   分段续跑第二段的 tail 从续跑消息之后算起，「第一段新增消息」既不在
	//   originalHist 也不在 tail → 第二段的全量覆盖写把它们丢掉（刷新后缺段）。
	//   现在：持久化基准改为可变底账，每段/每轮 Run 前经 refreshPersistBase
	//   推进为 store 当前内容（=上一段已落盘的完整时间线），
	//   保证 combined = 已落盘完整时间线 + 本轮新增（continue 追加语义）。
	var persistBaseMu sync.Mutex
	persistBase := opts.HistoryOriginal
	refreshPersistBase := func() {
		if sessStore == nil {
			return
		}
		saved, lerr := sessStore.LoadAll(convID)
		if lerr != nil {
			fmt.Printf("[persist] 持久化基准推进失败 conv=%s: %v\n", convID, lerr)
			return
		}
		if len(saved) == 0 {
			return
		}
		persistBaseMu.Lock()
		persistBase = saved
		persistBaseMu.Unlock()
	}

	if sessStore != nil {
		store := sessStore
		// ★ 基准来源（persistBase）：首段 = opts.HistoryOriginal（原始未压缩历史，
		//   防 msgs 头部压缩版写回覆盖）；后续段由 refreshPersistBase 推进。
		//   正确做法：已落盘完整时间线 + 新增尾部 = 持久化版本。
		loop.OnBatchPersist = func(msgs []Message) {
			// ★ 重组：已落盘基准（未压缩）+ 本轮新增消息 = 持久化版本。
			// msgs 结构：[system(可能), ...历史, 当前用户消息, ...本轮新增(assistant/tool)]
			// 锚点：最后一条「真实任务」RoleUser = 当前任务（Run 保证存在）。
			//   ★ 系统注入消息（历史压缩摘要/交接视图/历史遗留快照，见 injected_msg.go）
			//   也是 RoleUser 但不是用户任务，不能作为锚点（否则 tail 为空、其后
			//   真实消息全部丢失）。tail = 锚点之后的所有消息。
			// ⚠️ 不能再用「condensedLen 固定偏移」定位 tail：
			//   Run 开头的跨段精简/历史 condense 会改写 msgs（段内只追加，但跨段会精简）、
			//   删除中段历史 → len(msgs) 可能 < condensedLen → 旧逻辑误走兜底把压缩版
			//   写回 store，原始历史被覆盖、assistant 消息丢失（表现为 user 后直接 tool）。
			//   lastUser 锚点与历史长度无关，压缩/未压缩均正确。
			persistBaseMu.Lock()
			base := persistBase
			persistBaseMu.Unlock()
			combined := composePersistMessages(base, msgs)
			err := store.PersistNewMessages(convID, combined)
			if err != nil {
				fmt.Printf("[persist] OnBatchPersist 失败 conv=%s err=%v\n", convID, err)
			} else {
				// 同时持久化压缩摘要，确保页面刷新后能恢复
				if serr := store.SaveCompressedSummaries(convID, loop.CompressedSummaries); serr != nil {
					fmt.Printf("[persist] SaveCompressedSummaries 失败 conv=%s err=%v\n", convID, serr)
				}
			}
		}
		// OnMessagePersist：单条消息强制落盘（供委派/关键消息独立存储）
		loop.OnMessagePersist = func(msg Message) error {
			return store.AppendMessage(convID, msg, nil)
		}
	}

	sess.Loop = loop

	// ★ 2026-09-21 旧自主模式删除：原 OnNextTask 回调（从 TaskManager 队列拉「下一阶段任务」）
	//   已被「自主模式决策器」（插件 ctx.loopFactory.registerAutopilot，见 autopilot.go）
	//   取代——由插件扮演「人」角色审核/评判/决定下一步，宿主在下方的续轮循环中调用。

	// 注册 ask_user 工具：阻塞等用户回答（从 askCh 读）
	// Register 同名覆盖，安全替换调用方可能已注册的旧版本。
	// ★ 条件化：磁盘插件（tool-system）已注册同名工具（hostTool 路由版）时
	//   不再注册会话级版本——插件接管 agent 可见面，执行经 _convID 路由回本会话。
	//   插件未装载/停用时回退本版本（行为与旧版一致）。
	if opts.Registry != nil {
		if _, exists := opts.Registry.Get("ask_user"); !exists {
			opts.Registry.Register(&Tool{
				Name:       "ask_user",
				SystemTool: true,
				Description: "向用户提问并等待回答（用于关键决策、歧义澄清，别滥用）。" +
					"question 必填（或 questions 数组多问题）；askType 可选(text/single/multi/single-with-input)，默认 text 纯文本输入；" +
					"★ options 当 askType 为 single/multi/single-with-input 时必须提供（至少 2 个，如 [\"方案A\",\"方案B\"]），text 时可省略。" +
					"多问题：questions:[{id, question, options?, multi_select?}]（questions 优先，缺省回落单问题）。调用会阻塞直到用户回答。",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question": map[string]any{"type": "string", "description": "向用户提出的问题（单问题路径；与 questions 二选一）"},
						"askType":  map[string]any{"type": "string", "enum": []string{"text", "single", "multi", "single-with-input"}, "description": "提问类型：text(纯文本)/single(单选)/multi(多选)/single-with-input(单选+自由输入)"},
						"options":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "选项列表（当 askType 为 single/multi/single-with-input 时必须提供，至少 2 个，例如 [\"方案A\",\"方案B\"]；text 时可省略）"},
						"questions": map[string]any{
							"type":        "array",
							"description": "多问题数组（与 question 二选一；questions 优先）。每项 {id, question, options?, multi_select?}",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"id":           map[string]any{"type": "string", "description": "问题 ID（回答回灌时对应）"},
									"question":     map[string]any{"type": "string", "description": "问题文本"},
									"options":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "选项（选择类问题）"},
									"multi_select": map[string]any{"type": "boolean", "description": "是否多选（默认 false 单选）"},
								},
								"required": []string{"id", "question"},
							},
						},
					},
					"required": []string{},
				},
				RequiresApproval: false,
				Handler: func(hctx context.Context, args map[string]any) (string, error) {
					// ★ 2026-09-25：标记「正在等待用户回答」——SendAnswers 据此判定
					//   有无消费者（无消费者时把回答落盘为会话消息而非静默丢弃）。
					sess.askPending.Store(true)
					defer sess.askPending.Store(false)
					// ★ Round3 ⑤：questions 数组优先，缺省回落单问题路径
					if qs := askQuestionsFromArgs(args); len(qs) > 0 {
						select {
						case answers := <-sess.askCh:
							b, _ := json.MarshalIndent(map[string]any{"answers": answers}, "", "  ")
							return string(b), nil
						case <-hctx.Done():
							return "", hctx.Err()
						case <-time.After(5 * time.Minute):
							fmt.Printf("[session] ask_user(多问题) 超时 conv=%s\n", convID)
							return "", fmt.Errorf("等待用户回答超时（5 分钟）")
						}
					}
					question, _ := args["question"].(string)
					if question == "" {
						question = "（无问题内容）"
					}
					select {
					case answers := <-sess.askCh:
						if len(answers) > 0 {
							return strings.TrimSpace(answers[0].Answer), nil
						}
						return "", nil
					case <-hctx.Done():
						return "", hctx.Err()
					case <-time.After(5 * time.Minute):
						fmt.Printf("[session] ask_user 超时 conv=%s\n", convID)
						return "", fmt.Errorf("等待用户回答超时（5 分钟）")
					}
				},
			})

		}
	}

	// 存入 map（覆盖已结束的旧会话），并做已结束会话上限淘汰。
	// ★ 2026-08-30：挂载时短暂持写锁（原先整个 Start 持锁）；锁内二次查重
	//   ——锁外重活期间可能已有同 convID 会话挂载（startingMu 已拦同 convID
	//   并发 Start，此处为跨路径兜底）。
	m.mu.Lock()
	if prev, ok := m.sessions[convID]; ok && prev.Running {
		m.mu.Unlock()
		cancel()
		return ErrSessionRunning
	}
	m.sessions[convID] = sess
	m.evictIfNeeded()
	m.mu.Unlock()

	// fan-out goroutine：从 Events 读取，写入所有 subscribers（非阻塞，满则丢弃）。
	// 同时写入全局订阅者（WebSocket 端点），让跨工作区的所有会话事件都可通过单一连接传输。
	// Events 关闭后退出，并关闭所有剩余订阅者 channel（通知订阅者流结束）。
	go func() {
		// ★ 2026-09-12：订阅者投递失败（缓冲满）此前完全静默——排查「前端突然不再
		//   输出 agent 内容」时无任何线索。现累计并低频打日志（首 3 次 + 每 500 条），
		//   便于定位「事件被丢弃」与「客户端消费慢/写阻塞」。
		subDropped := 0
		for e := range sess.Events {
			sess.subMu.RLock()
			subs := sess.subscribers
			sess.subMu.RUnlock()
			for _, sub := range subs {
				select {
				case sub <- e:
				default:
					subDropped++
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
					subDropped++
				}
			}
			if subDropped > 0 && (subDropped <= 3 || subDropped%500 == 0) {
				log.Printf("[session] 订阅者事件丢弃 conv=%s 累计=%d（订阅者缓冲满：前端消费慢/WS 写阻塞 → 客户端缺事件，重连时由 snapshot 补偿）", convID, subDropped)
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
		// ★ 会话级 store（按 opts.WorkspaceRoot 路由）：不用全局 m.store，
		//   避开「运行中切换工作区 → 中断标记写错库」与并发读 m.store。
		store := sessStore
		interrupted := false // 会话结束时的中断状态（defer 中据此写回持久化标记）
		// ★ 会话运行环境（子 agent 能力）随会话结束解除：之后插件再调用 ctx.subagent
		//   会得到「能力不可用」错误，而不是拿到已结束会话的模型/工具面。
		defer unregisterSubagent()
		defer func() {
			// panic recovery：确保会话状态和事件通道始终被清理
			if r := recover(); r != nil {
				// ★ 2026-08-21：打印完整堆栈（原仅 %v 无法定位 nil 指针来源）
				fmt.Printf("[session] Loop goroutine panic conv=%s: %v\n%s\n", convID, r, debug.Stack())
				// ★ panic 恢复时也发送 EventError，否则前端无任何信号（assistant 永久 loading）
				// 注：用 select+default 非阻塞发送，不阻塞 defer 关闭流程，也不向已关闭 channel panic
				select {
				case sess.Events <- Event{Type: EventError, Content: fmt.Sprintf("Agent 异常终止: %v", r)}:
				default:
				}
				sess.History = loop.History
				interrupted = true // panic → 异常中断
			}
			// 更新 History（loop.Run 的 defer 已更新 loop.History，同步到 session）
			sess.History = loop.History
			// ★ 后端运行统计：定格本次运行（endAt + 速度派生 + 落盘 .pair/run-stats.json）。
			//   放在 History 同步之后、状态标记之前：任何结束路径（正常完成/用户停止/
			//   异常/panic）都会执行 → 前端刷新后仍能读到本轮结果。
			EndRunStatsFor(opts.WorkspaceRoot, convID)
			// ★ 持久化中断状态：异常/用户停止 → interrupted=true（任务未完成，前端显示"可继续"），
			//   正常完成（err==nil）→ false。此标记写入 index.json，跨进程重启保留。
			if store != nil {
				if setErr := store.SetInterrupted(convID, interrupted); setErr != nil {
					fmt.Printf("[session] SetInterrupted 失败 conv=%s err=%v\n", convID, setErr)
				}
			}
			// 标记结束
			m.mu.Lock()
			sess.Running = false
			m.mu.Unlock()
			// ★ 2026-09-21 状态刷新信号（见 loop.go EventStatusRefresh 注释）：
			//   放在 Running=false 之后、close(Events) 之前 —— WS 端点据此补推一次
			//   status（runningConvs 快照），修正自主模式收尾后前端「运行中」状态残留
			//   （done 事件早于会话真正结束，done 后那份 status 仍含本会话，之后无事件）。
			//   非阻塞发送：缓冲满时丢弃无妨（前端重连 / 下轮 done 仍会校正）。
			select {
			case sess.Events <- Event{Type: EventStatusRefresh}:
			default:
			}
			// 关闭 Events → fan-out goroutine 退出，subscribers 检测到通道关闭
			close(sess.Events)
		}()

		// 自闭环模式：传 nil history，loop.Run 内部使用 loop.History
		var msgs []Message
		var err error
		// ★ 2026-09-21 自主模式（插件化）：Autonomous=true 时 Loop 仍按普通一轮执行，
		//   「监督 → 续跑」由下方续轮循环调用插件决策器实现（旧的内层阶段化循环已删除）。
		if opts.Autonomous {
			loop.Autonomous = true
		}
		msgs, err = loop.Run(runCtx, task, nil)
		sess.History = msgs

		// ★ 段预算分段续跑（2026-09，tool_budget.go；★ 2026-09-12 双闸门）：
		//   本 Run 因达到段预算而结束（步数默认 120 步 / 工具调用默认 120 次，
		//   由 agentloop 插件注册的 stepBudget / toolCallBudget 覆盖；**任一达上限
		//   即触发**，命中闸门见 SegmentState.Reason）→ 自动发起下一段
		//   （同会话、历史保留；段内只追加不精简，体积控制见 tool_budget.go 与
		//    compress.go maybeCompact 的说明——分段边界 + 跨段精简）。
		//   segmentNo 统计本轮任务的自动续跑段数，超过生效上限 segLimit
		//   停止（防失控；用户再发消息即可继续）。用户停止（sess.stopped）即退出。
		segmentNo := 0
		// ★ 2026-09-12 配置化：续跑段数上限取本 Loop 生效值（装配参数
		//   maxToolBudgetSegments 透传；0/缺省 = 默认 20，见 tool_budget.go）。
		segLimit := loop.MaxToolBudgetSegmentsOrDefault()
		// ★ 2026-09-21 自主模式（插件化）：监督续轮计数与上限。
		//   工作 agent 每次自然结束 → 唤醒插件决策器（「人」角色：审核/评判/决定下一步）；
		//   判定继续则把指令作为新任务唤醒工作 agent，判定完成/插件未生效则收尾。
		superviseNo := 0
		//   上限取本 Loop 生效值（装配参数 maxSuperviseRounds 透传；0/缺省 = 默认 20）。
		superviseLimit := loop.MaxSuperviseRoundsOrDefault()
		// ★ Round3 ③.1 goal 自动续轮（对齐 DSH「同会话完成目标」语义）：
		//   会话 Run 结束后，goal Armed && 非终态 && Rounds < RoundLimit →
		//   自动发起下一轮（continuation 消息）。pause 停续轮、resume 重挂；
		//   同一阻塞条件连续 ≥3 轮自动 blocked（MarkRound 内判定）。
		//   零行为变化保证：无 goal(op=create) 时 goalManager.Get 返回 nil，循环直接退出。
		for !sess.stopped {
			if seg, segOK := loop.TakeSegmentContinue(); segOK {
				segmentNo++
				if segmentNo > segLimit {
					notice := fmt.Sprintf("段预算分段已达上限（%d 段），自动续跑停止；如仍需继续，请再发一条消息。", segLimit)
					log.Printf("[session] 分段续跑达上限 conv=%s segments=%d limit=%d", convID, segmentNo-1, segLimit)
					select {
					case sess.Events <- Event{Type: EventNotice, Content: notice}:
					default:
					}
					break
				}
				contMsg := SegmentContinueMessage(seg)

				// ★ 2026-09-11 会话交接（handoff.go）：段边界对累计历史做一次判断/整理——
				//   达阈值时把上一段历史折叠为「提交消息」，替代全量历史注入
				//   下一段（防续跑上下文膨胀）；未达阈值保持原样（同会话历史保留）。
				//   整理只替换喂 LLM 的历史视图；落盘/展示仍为完整时间线。
				// ★ 2026-09-12 策略外置：整理由 agentloop 插件（registerHandoff.onSegment）
				//   实现；宿主只提供执行位置与能力（provider/store/口径工具），未注册或
				//   执行失败即回退 Go 默认实现（handoff.go，语义不变）。
				//   ★ 段边界是**跨轮边界**：发生在上一段已结束、下一段尚未开始处，
				//   轮内（一次 Run 的 step 之间）不整理——保证轮内前缀 append-only。
				nextHist := []Message(nil)
				if view, hok, hnotice := HandoffSegmentView(runCtx, loop, store, convID, contMsg); hok {
					nextHist = view
					if hnotice == "" {
						hnotice = fmt.Sprintf(
							"已把此前对话整理为「会话交接·提交消息」（历史 %d 条 → 交接要点 + 近期 %d 条），本段从交接要点继续",
							len(loop.History), len(view)-1)
					}
					select {
					case sess.Events <- Event{Type: EventNotice, Content: hnotice}:
					default:
					}
				}

				log.Printf("[session] 段预算分段续跑 conv=%s segment=%d reason=%s 步数=%d/%s 工具调用=%d/%s",
					convID, segmentNo, seg.Reason,
					seg.UsedSteps, budgetText(seg.StepBudget), seg.UsedTools, budgetText(seg.ToolBudget))
				select {
				case sess.Events <- Event{Type: EventNotice, Content: contMsg}:
				default:
				}
				// ★ 开新一轮前推进持久化基准（防多段全量覆盖丢消息；见 refreshPersistBase）。
				refreshPersistBase()
				msgs, err = loop.Run(runCtx, contMsg, nextHist)
				sess.History = msgs
				continue
			}
			segmentNo = 0 // 非分段结束 → 段计数归零（goal 续轮属于新一轮任务阶段）

			// ★ 2026-09-21 自主模式（插件化）：工作 agent 自然结束 → 唤醒「人」角色决策器。
			//   决策器 = 插件经 ctx.loopFactory.registerAutopilot 注册的实现（策略 JS：
			//   角色提示词/任务书/裁决语义/记录落盘）；它通常经 ctx.subagent 能力派生一个
			//   带全部工具的「监督者」回合自行核查（能力 Go，见 subagent.go）。
			//   判定 continue → 指令作为新任务唤醒工作 agent（历史尾部追加，不破前缀）；
			//   判定 done / 未生效 → 收尾。轮数上限与用户停止为硬保护。
			// ★ 2026-09-21 诊断：打印判定要素（排查「监督未触发 / 会话卡在决策器」）——
			//   自主模式下每次自然结束都记一行，运维可据此确认监督分支是否进入。
			if opts.Autonomous {
				log.Printf("[session] 自主模式判定 conv=%s err=%v available=%v turnReason=%q superviseNo=%d limit=%d",
					convID, err, AutopilotAvailable(), loop.LastTurnReason, superviseNo, superviseLimit)
			}
			if opts.Autonomous && err == nil && AutopilotAvailable() &&
				(loop.LastTurnReason == TurnCompleted || loop.LastTurnReason == TurnContentLoop) {
				superviseNo++
				if superviseLimit > 0 && superviseNo > superviseLimit {
					notice := fmt.Sprintf("自主模式：监督轮数已达上限（%d），自动收尾；如仍需继续，请再发一条消息。", superviseLimit)
					log.Printf("[session] 自主模式监督轮数达上限 conv=%s limit=%d", convID, superviseLimit)
					select {
					case sess.Events <- Event{Type: EventNotice, Content: notice}:
					default:
					}
					break
				}
				req := AutopilotRequest{
					ConvID:        convID,
					WorkspaceRoot: opts.WorkspaceRoot,
					Round:         superviseNo,
					MaxRounds:     superviseLimit,
					Objective:     FirstUserTask(loop.History),
					WorkerReport:  LastAssistantContent(msgs),
					WorkerTurns:   loop.TurnNo,
					RecentHistory: RecentHistoryTail(loop.History, AutopilotHistoryTail),
				}
				// 决策器调用期间绑定本会话环境：插件 ctx.subagent.run 据此复用本会话
				// 的模型/工具面/事件通道（事件带 AgentName 下发前端，看板分区）。
				restoreCur := SetCurrentSubagentRuntime(subagentRt)
				decision := RunAutopilotDecision(runCtx, req)
				restoreCur()
				if notice := AutopilotDecisionNotice(decision, superviseNo); notice != "" {
					select {
					case sess.Events <- Event{Type: EventNotice, Content: notice}:
					default:
					}
				}
				if !decision.Continue() {
					if decision.Error != "" {
						log.Printf("[session] 自主模式决策失败 conv=%s round=%d err=%s", convID, superviseNo, decision.Error)
					}
					break
				}
				contMsg := decision.Task
				log.Printf("[session] 自主模式监督续跑 conv=%s round=%d/%d 指令=%q",
					convID, superviseNo, superviseLimit, truncStr(contMsg, 80))
				// 开新一轮前推进持久化基准（防上一轮新增被覆盖）。
				refreshPersistBase()
				msgs, err = loop.Run(runCtx, contMsg, nil)
				sess.History = msgs
				continue
			}

			g := goalManager.MarkRound(opts.WorkspaceRoot, convID, err)
			if g == nil || g.ContinueMessage() == "" {
				break
			}
			// ★ 2026-09-13（S3）：原先此处把 goal 段挂在 loop.ResumeContext 上（指望背景快照注入），
			//   快照链停用后该路径无消费者 → 已移除（详见本文件 CreateLoop 处的说明）。
			//   goal 上下文由下面的续轮消息承载：ContinueMessage 已含目标/阶段/轮次，
			//   作为新 user 消息追加 = 历史尾部追加，不破坏前缀缓存。
			lmsg := g.ContinueMessage()
			log.Printf("[session] goal 自动续轮 conv=%s round=%d/%d objective=%q",
				convID, g.Rounds, g.RoundLimit, truncStr(g.Objective, 60))
			select {
			case sess.Events <- Event{Type: EventNotice, Content: lmsg}:
			default:
			}
			// ★ 开新一轮前推进持久化基准（goal 续轮同样适用：防上一轮新增被覆盖）。
			refreshPersistBase()
			msgs, err = loop.Run(runCtx, lmsg, nil)
			sess.History = msgs
		}
		// ★ 记录会话结束方式：err != nil（LLM API 错误/panic/ctx 取消含用户停止）
		//   → 任务未正常完成，标记为"可继续"；正常完成（err==nil）→ 保持 false。
		if err != nil {
			interrupted = true
			// ★ Loop 异常结束日志（排查「无响应」：确认 Loop 退出的原因）
			log.Printf("[session] Loop 异常结束 conv=%s stopped=%v err=%v", convID, sess.stopped, err)
		} else {
			log.Printf("[session] Loop 正常结束 conv=%s msgs=%d", convID, len(msgs))
		}

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

// Compact 请求压缩上下文。
// 如果有运行中的 Loop，设置 CompactRequested 让下轮迭代触发压缩。
// 同时直接压缩已存储的历史消息（无论 Loop 是否运行），使下次加载时上下文更小。
func (m *SessionManager) Compact(convID string) {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	store := m.store
	m.mu.RUnlock()

	// 有运行中的 Loop → 标记下次迭代压缩
	if ok && sess.Loop != nil && sess.Running {
		sess.Loop.CompactRequested = true
	}

	// 直接精简已存储的历史消息
	if store == nil {
		return
	}
	msgs, err := store.LoadAll(convID)
	if err != nil || len(msgs) < 3 {
		return
	}
	// 使用历史精简：旧轮次压缩为 [用户消息, 助理最终报告]，当前轮次完整保留
	condensed := CondenseHistory(msgs)
	// 只在确实有精简时才写回
	if len(condensed) < len(msgs) {
		if err := store.ReplaceHistory(convID, condensed); err != nil {
			fmt.Printf("[session] Compact 替换 store 失败 conv=%s: %v\n", convID, err)
		}
	}
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
	if ok && sess.WorkspaceRoot != "" {
		store = m.storeFor(sess.WorkspaceRoot)
	}
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

// SetReviewMode 实时更新指定会话的审核模式（只影响当前对话）。
// 用户在工具栏切换审核模式时立即生效，无需新消息。
func (m *SessionManager) SetReviewMode(convID string, v string) {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if ok && sess.Loop != nil {
		sess.Loop.SetReviewMode(v)
		fmt.Printf("[session] 实时更新审核模式 conv=%s reviewMode=%s\n", convID, v)
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

// SendAnswer 向指定会话发送 ask_user 的用户回答（单问题路径：编为单元素数组）。
// 会话不存在或未运行返回错误。
func (m *SessionManager) SendAnswer(convID string, answer string) error {
	return m.SendAnswers(convID, []AskAnswer{{Answer: answer}})
}

// SendAnswers 向指定会话发送 ask_user 的回答数组（Round3 ⑤ 多问题）。
// 会话不存在或未运行返回错误。
//
// ★ 2026-09-25 新增语义：若该会话当前没有 ask_user 在等待回答（提问已超时/被
// 停止），回答不再投递进 askCh（1 格缓冲，投递「成功」但无人消费 → 回答永不
// 落盘：前端显示「已回答」、历史查无此答），而是落盘为一条会话 user 消息并返回
// ErrAskNotPending；web 层据此回 {"ok":true,"recorded":"message"}。落盘幂等：
// 与「会话最后一条 user 消息」同文本时直接跳过（前端重试/重复提交不产生重复消息）。
//
// 锁范围：m.mu 仅保护 sessions map 查表；askPending 为 atomic.Bool 自同步；
// storeFor 内部用 wsMu、MessageStore 自身按 convID 加锁——本函数不嵌套持锁，
// 落盘全程在 m.mu 之外执行。
func (m *SessionManager) SendAnswers(convID string, answers []AskAnswer) error {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if !ok {
		return ErrSessionNotFound
	}
	if !sess.Running {
		return ErrSessionNotRunning
	}
	// 无消费者兜底（见函数注释）：atomic 读，无需 m.mu。
	if !sess.askPending.Load() {
		if err := m.persistLateAskAnswer(sess, answers); err != nil {
			return fmt.Errorf("提问已结束（超时/停止），且回答写入会话消息失败：%w", err)
		}
		return ErrAskNotPending
	}
	select {
	case sess.askCh <- answers:
		return nil
	default:
		return errors.New("回答通道已满（可能已有待处理回答）")
	}
}

// IsAwaitingAnswer 该会话当前是否有 ask_user 在等待回答（无会话/未运行返回 false）。
func (m *SessionManager) IsAwaitingAnswer(convID string) bool {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	return ok && sess.askPending.Load()
}

// persistLateAskAnswer 把「提问已结束」后才提交的回答落盘为会话 user 消息。
// 走会话自己的工作区 store（不写全局 m.store——会话运行中用户切换工作区时避免写错库）；
// 幂等：内容与最近一条 user 消息相同时跳过（重复提交不生重复消息）。
func (m *SessionManager) persistLateAskAnswer(sess *Session, answers []AskAnswer) error {
	store := m.storeFor(sess.WorkspaceRoot)
	if store == nil {
		return errors.New("消息存储未就绪（会话工作区根为空）")
	}
	text := formatLateAskAnswer(answers)
	if strings.TrimSpace(text) == "" {
		return errors.New("回答内容为空")
	}
	if last, _, err := store.LoadLatest(sess.ConvID, 1); err == nil && len(last) > 0 {
		prev := last[len(last)-1].Message
		if prev.Role == RoleUser && strings.TrimSpace(prev.Content) == strings.TrimSpace(text) {
			return nil // 幂等：同一回答已落盘
		}
	}
	if err := store.AppendUserMessage(sess.ConvID, text); err != nil {
		return err
	}
	log.Printf("[session] ask_user 迟到回答已落盘为会话消息 conv=%s（提问已超时/停止）", sess.ConvID)
	return nil
}

// formatLateAskAnswer 组装迟到回答的消息文本：单问题（ID 空）直接用答案原文；
// 多问题逐条列出（带问题 ID，便于模型/用户对应）。
func formatLateAskAnswer(answers []AskAnswer) string {
	if len(answers) == 1 && strings.TrimSpace(answers[0].ID) == "" {
		return strings.TrimSpace(answers[0].Answer)
	}
	var b strings.Builder
	b.WriteString("（提问已超时结束，这是我对该提问的补充回答）")
	for i, a := range answers {
		ans := strings.TrimSpace(a.Answer)
		if ans == "" {
			continue
		}
		if strings.TrimSpace(a.ID) != "" {
			b.WriteString(fmt.Sprintf("\n%d. [%s] %s", i+1, a.ID, ans))
		} else {
			b.WriteString(fmt.Sprintf("\n%d. %s", i+1, ans))
		}
	}
	return b.String()
}

// WaitAnswer 按 convID 等待用户回答（ask_user 会话桥路由入口，单问题路径）。
// 与 SendAnswer 配对：前端 /api/answer → SendAnswer → askCh → 本方法返回。
// 会话不存在/未运行报错；ctx 取消或 5 分钟超时返回错误（不永久阻塞）。
func (m *SessionManager) WaitAnswer(ctx context.Context, convID string) (string, error) {
	answers, err := m.WaitAnswers(ctx, convID)
	if err != nil {
		return "", err
	}
	if len(answers) == 0 {
		return "", nil
	}
	return strings.TrimSpace(answers[0].Answer), nil
}

// WaitAnswers 按 convID 等待用户回答数组（Round3 ⑤ 多问题）。
func (m *SessionManager) WaitAnswers(ctx context.Context, convID string) ([]AskAnswer, error) {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrSessionNotFound
	}
	if !sess.Running {
		return nil, ErrSessionNotRunning
	}
	// ★ 2026-09-25：标记等待中——SendAnswers 据此判定有无消费者（无消费者时
	//   把回答落盘为会话消息，避免「已回答」却历史查无此答）。
	sess.askPending.Store(true)
	defer sess.askPending.Store(false)
	select {
	case answers := <-sess.askCh:
		return answers, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Minute):
		fmt.Printf("[session] ask_user 超时 conv=%s\n", convID)
		return nil, fmt.Errorf("等待用户回答超时（5 分钟）")
	}
}

// GetSessionWorkspaceRoot 取会话工作区根（task_create 持久化路由用；无会话返回空串）。
func (m *SessionManager) GetSessionWorkspaceRoot(convID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.sessions[convID]; ok {
		return s.WorkspaceRoot
	}
	return ""
}

// Approve 向指定会话发送审批结果。
// approved=true=允许，approved=false=拒绝。reply 为用户输入的回复内容。
func (m *SessionManager) Approve(convID string, approved bool, reply ...string) error {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if !ok {
		return ErrSessionNotFound
	}
	if !sess.Running {
		return ErrSessionNotRunning
	}
	r := ""
	if len(reply) > 0 {
		r = reply[0]
	}
	select {
	case sess.approvalCh <- ApprovalResult{Approved: approved, Reply: r}:
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
	// ★ 2026-09-12：容量 200 → 2000。流式事件（thinking/content 每 chunk 一条）在长任务下
	//   可达每秒数十条；客户端写慢（浏览器主线程忙/后台标签节流）时 200 缓冲瞬间打满 →
	//   后续事件被 fan-out 静默丢弃 → 前端「agent 在跑但内容缺块/看起来停了」。
	//   加大缓冲显著降低概率；配合 fan-out 的丢弃日志（可见化）与重连 snapshot 补偿。
	ch := make(chan GlobalEvent, 2000)
	m.globalSubMu.Lock()
	m.globalSubscribers = append(m.globalSubscribers, ch)
	m.globalSubMu.Unlock()
	return ch
}

// PushStartError 向全局订阅者推送会话启动失败错误事件。
// 供 handleChatSend 异步化后使用：Start 在后台 goroutine 执行，HTTP 已立即返回，
// 失败时经 WebSocket 事件流推送到前端（前端按 convID 路由到 assistant 占位消息，
// 显示「[错误] …」并清理 loading 状态）。
func (m *SessionManager) PushStartError(convID, errMsg string) {
	m.globalSubMu.RLock()
	gsubs := m.globalSubscribers
	m.globalSubMu.RUnlock()
	ge := GlobalEvent{ConvID: convID, Event: Event{Type: EventError, Content: errMsg}}
	for _, gsub := range gsubs {
		select {
		case gsub <- ge:
		default:
		}
	}
	log.Printf("[session] PushStartError conv=%s err=%s", convID, errMsg)
}

// PushNotice 向全局订阅者推送提示事件（非错误；web 层异步路径的用户可见提示，
// 如会话交接整理完成）。前端按 convID 路由显示为 notice。
func (m *SessionManager) PushNotice(convID, text string) {
	m.globalSubMu.RLock()
	gsubs := m.globalSubscribers
	m.globalSubMu.RUnlock()
	ge := GlobalEvent{ConvID: convID, Event: Event{Type: EventNotice, Content: text}}
	for _, gsub := range gsubs {
		select {
		case gsub <- ge:
		default:
		}
	}
	log.Printf("[session] PushNotice conv=%s text=%s", convID, shortenErr(text, 80))
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

// shortenErr 截断错误文本（服务端 notice/日志用：过长错误不美观且占满通道）。
func shortenErr(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
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

// LiveSnapshotEvent (convID) 生成当前会话的流式生成进度快照事件。
// 供 WebSocket 端点重连补偿：客户端 WS 断线期间 content/thinking/tool 事件丢失，
// 重连后推送本事件，前端据此重建占位消息（不再出现内容截断）。
// convID 未运行或无 Loop 时返回 nil。
func (m *SessionManager) LiveSnapshotEvent(convID string) *Event {
	m.mu.RLock()
	sess, ok := m.sessions[convID]
	m.mu.RUnlock()
	if !ok || sess == nil || !sess.Running || sess.Loop == nil {
		return nil
	}
	content, reasoning, tools, events := sess.Loop.LiveSnapshot()
	if content == "" && reasoning == "" && len(tools) == 0 && len(events) == 0 {
		return nil // 无累积（尚未产出）→ 无需补偿
	}
	return &Event{
		Type:         EventSnapshot,
		Content:      content,
		Reasoning:    reasoning,
		ToolSegments: tools,
		LiveEvents:   events,
	}
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

// AppendPersistedMessage 追加一条消息到 MessageStore（持久化）。
// 供 web 层 startEventPersistWorker 在 EventDone 等时机调用，
// 将 loop.History 中的新消息增量持久化。
// 若 store 为 nil 则无操作（web 层未注入时静默跳过）。
func (m *SessionManager) AppendPersistedMessage(convID string, msg Message, segments []Segment) error {
	m.mu.RLock()
	store := m.store
	if s, ok := m.sessions[convID]; ok && s.WorkspaceRoot != "" {
		store = m.storeFor(s.WorkspaceRoot)
	}
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
	if s, ok := m.sessions[convID]; ok && s.WorkspaceRoot != "" {
		store = m.storeFor(s.WorkspaceRoot)
	}
	m.mu.RUnlock()
	if store == nil {
		return nil
	}
	return store.AppendUserMessage(convID, content)
}

// AppendPersistedUserMessageWithImages 追加一条带图片的用户消息（★ 2026-08-21 多模态）。
// 供 web 层 handleChatSend 在调 Start 前调用：前端结构化发送 images 数组 → 落盘 →
// Provider.Chat 转 OpenAI content 块数组发送。
func (m *SessionManager) AppendPersistedUserMessageWithImages(convID, content string, images []ImagePart) error {
	m.mu.RLock()
	store := m.store
	if s, ok := m.sessions[convID]; ok && s.WorkspaceRoot != "" {
		store = m.storeFor(s.WorkspaceRoot)
	}
	m.mu.RUnlock()
	if store == nil {
		return nil
	}
	return store.AppendUserMessageWithImages(convID, content, images)
}

// AppendPersistedUserMessageTo 按指定工作区根追加用户消息（★ 2026-08-23 工作区隔离：
// handleChatSend 提交的 workspaceRoot 直接路由到对应 store——切换工作区后继续向
// 旧工作区会话发消息不再落进新工作区存储）。
func (m *SessionManager) AppendPersistedUserMessageTo(wsRoot, convID, content string) error {
	store := m.StoreFor(wsRoot)
	if store == nil {
		return nil
	}
	return store.AppendUserMessage(convID, content)
}

// AppendPersistedUserMessageToWithImages 按指定工作区根追加带图片用户消息。
func (m *SessionManager) AppendPersistedUserMessageToWithImages(wsRoot, convID, content string, images []ImagePart) error {
	store := m.StoreFor(wsRoot)
	if store == nil {
		return nil
	}
	return store.AppendUserMessageWithImages(convID, content, images)
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
				// ★ 2026-08-23 按会话工作区根路由（运行中切换工作区不串库）。
				m.mu.RLock()
				var store ConversationStore
				if s, ok := m.sessions[ge.ConvID]; ok && s.WorkspaceRoot != "" {
					store = m.storeFor(s.WorkspaceRoot)
				} else {
					store = m.store
				}
				m.mu.RUnlock()
				if store != nil {
					_ = store.SetCtxStats(ge.ConvID, ge.Event.Usage)
				}
			}
		}
	}()
}
