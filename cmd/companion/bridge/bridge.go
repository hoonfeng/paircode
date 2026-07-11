// Agent 引擎 ↔ 聊天面板 桥接：把 agent.Loop 接到对话 UI。
// send → 异步跑 TAOR 循环（goroutine）；事件（thinking/工具调用/结果/final）经**定时器帧泵**
// 周期在 UI 线程 drain，流式写进当前助手消息（复刻终端面板跨线程模式，见 terminal.go / AGENTS.md）。
// 本文件还含 Agent 消息卡的富渲染（思考块 + 工具活动行 + 正文）。
//
//go:build windows

package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	"github.com/hoonfeng/paircode/cmd/companion/agenttools"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/roleprompts"
	"github.com/hoonfeng/paircode/pkg/memory"
	"github.com/hoonfeng/paircode/pkg/summary"
	chat "github.com/hoonfeng/paircode/cmd/companion/ui/chat"
	editorpanel "github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	filetreepanel "github.com/hoonfeng/paircode/cmd/companion/ui/filetree"
	mcppanel "github.com/hoonfeng/paircode/cmd/companion/ui/mcp"
	skillspanel "github.com/hoonfeng/paircode/cmd/companion/ui/skills"
	"github.com/hoonfeng/paircode/cmd/companion/ui/state"
)

// SetIntervalFunc 由 main 注入：周期回调（替代 goui animation.Controller 帧泵），返回 timer ID。
var SetIntervalFunc func(interval time.Duration, fn func()) int

// ClearIntervalFunc 由 main 注入：取消定时器。
var ClearIntervalFunc func(id int)

// agentBridge 持有 Agent 引擎与一次流式回复的运行态。挂在 chat.ChatState 上（懒建）。
type AgentBridge struct {
	Cs           *chat.ChatState
	loop         *agent.Loop      // 懒建（provider 来自环境变量）；测试可预置 mock
	planner      *agent.Planner   // 规划 Agent（自主模式下先列计划；nil=不规划）。每次发送按设置重建
	reviewer     *agent.Reviewer  // 审核 Agent（AI 审核模式下把关写操作；nil=不审）。每次发送按设置重建
	evaluator    *agent.Evaluator // 评测 Agent（任务完成后打分；nil=不评测）。每次发送按设置重建
	luaToolNames []string         // 上轮加载的 Lua 自定义工具名（热重载时先卸载）
	root         string           // 工作区根（= 工具/文件树/编辑器同根），用于把工具 path 解析为绝对路径

	mu        sync.Mutex
	running   bool
	pending   []agent.Event // loop 协程写、帧泵 drain
	pumpID    int           // SetInterval 返回的定时器 ID（0=未运行）
	runThread *state.Thread // 当前流式回复所在会话（捕获，防用户切会话）
	runIdx    int           // 流式回复在该会话的消息索引

	cancel  context.CancelFunc // 取消当前 loop（停止按钮）；start 置、stop 调
	stopped bool               // 本轮被用户主动停止（UI 线程读写）：抑制错误显示、收尾标[已停止]

	// 审批（手动审核模式）：loop 协程在 approve() 里登记裁决通道并阻塞，UI 线程点「允许/拒绝」经 resolveApproval 送回。
	approvalCh     chan bool
	approvalCallID string

	// ask_user：loop 协程在 askUser() 里登记回答通道并阻塞，UI 线程选项/输入经 resolveAsk 送回（同一时刻仅一个）。
	askCh chan string

	lastDrainTime time.Time
}

// NewAgentBridge 创建一个绑定到指定 ChatState 的 AgentBridge（由 main 注入 chat.NewBridge）。
func NewAgentBridge(cs *chat.ChatState) *AgentBridge {
	return &AgentBridge{Cs: cs}
}

func (b *AgentBridge) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

// runningThread 返回当前正被运行的会话（未运行返回 nil）。会话列表状态灯着色用。
func (b *AgentBridge) RunningThread() *state.Thread {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running {
		return b.runThread
	}
	return nil
}

// stop 用户主动停止当前运行：取消 ctx（loop 在下个迭代/Chat/审批等待处停下）。收尾标记 [已停止]。
func (b *AgentBridge) Stop() {
	b.mu.Lock()
	b.stopped = true
	cancel := b.cancel
	b.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	b.Cs.SaveHistory() // 停止时保存对话历史，防止消息丢失
}

// resetForNewRoot 项目根切换后清掉已建 loop，下条消息用新根重建（运行中则不动，避免打断）。
func (b *AgentBridge) ResetForNewRoot() {
	if b.IsRunning() {
		return
	}
	b.loop = nil
	b.root = ""
}

// autonomousParams 据自主开关算（实际下发给 LLM 的任务文本, 迭代上限）。
// 自主：追加「列计划→连续完成所有步骤→全部完成再调用 finish_task」提示 + 放宽迭代上限。
func AutonomousParams(task string, autonomous bool) (string, int) {
	base := core.Settings.MaxIterations // 设置面板「最大迭代步数」；未设=30
	if base <= 0 {
		base = 30
	}
	if autonomous {
		return task + "\n\n（自主模式：先用 update_plan 列出完整计划，然后连续完成所有步骤、全部完成后调用 finish_task 工具。）", base * 2
	}
	return task, base
}

// start 异步跑一轮 Agent 任务（UI 线程调用）。无 API key 则只提示、不跑。
func (b *AgentBridge) Start(task string) {
	task = strings.TrimSpace(task)
	if task == "" {
		return
	}
	th := b.Cs.Store.Active()
	if th == nil {
		return
	}
	// 懒建 loop：provider 来自环境变量；无 key → 给一条提示、不进循环。
	if b.loop == nil {
		prov := BuildProvider()
		if prov == nil {
			th.Messages = append(th.Messages, state.Message{Role: state.Assistant,
				Text: "未配置 API key。请设置环境变量 DEEPSEEK_API_KEY / OPENAI_API_KEY / DASHSCOPE_API_KEY / MOONSHOT_API_KEY / OPENROUTER_API_KEY 之一后重启，即可与我对话。"})
			return
		}
		root := core.Root() // 当前项目根（与文件树/终端统一）
		b.root = root
		memory.SetRoot(root)
		ApplyIgnoreDirs(root) // 全局设置 + 项目级 .pair/ignore → 注入搜索/探索的忽略目录
		reg := agent.NewRegistry()
		agent.WorkspaceRoots = core.Folders
		// Skills 三级渐进披露：注入全局变量供 agent/skill_loader 使用
		// （agent 包不依赖 core，故由 bridge 启动时注入，与 WorkspaceRoots 模式一致）。
		agent.SkillSystemDir = filepath.Join(core.ConfigDir(), "skills")    // 内置技能：安装目录/config/skills
		agent.SkillProjectDir = filepath.Join(root, ".pair", "skills")      // 项目级技能：工作区 .pair/skills
		agent.SkillEnabled = core.Settings.SkillEnabledOverrides            // 按 level::name 过滤（不存在默认启用）
		agent.RegisterDefaultTools(reg, root)
		// finish_task
		reg.Register(&agent.Tool{
			Name:        "finish_task",
			Description: "任务完成信号：全部任务完成时调用此工具结束本轮。result 为完成摘要。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"result": map[string]any{"type": "string", "description": "任务完成摘要"},
				},
				"required": []string{"result"},
			},
			Handler: func(_ context.Context, args map[string]any) (string, error) {
				r, _ := args["result"].(string)
				return r, nil
			},
		})
		b.registerAskTool(reg)                             // ask_user：handler 闭包持有 bridge（需 UI 交互），故在此注册而非默认集
		agenttools.RegisterManagementTools(reg)            // Agent 自管理 Skills/MCP + 市场 + 技能渐进式披露(load_skill)
		if cfgs := mcppanel.LoadConfigs(); len(cfgs) > 0 { // 外部 MCP 服务器（mcp.json；失败跳过、不阻断；首条消息时一次性连接）
			agentCfgs := make([]agent.MCPServerConfig, len(cfgs))
			for i, c := range cfgs {
				agentCfgs[i] = agent.MCPServerConfig{Name: c.Name, Command: c.Command, Args: c.Args, Env: c.Env}
			}
			agent.RegisterMCPServers(reg, agentCfgs)
		}
		sys := agent.DefaultSystemPrompt(core.Folders)
		if si := strings.TrimSpace(core.Settings.SystemInstructions); si != "" { // 设置里的系统级指令
			sys += "\n\n# 系统级指令（务必遵守）\n" + si
		}
		sys += roleprompts.PhilosophyPrompt() // 思想 tab：指导思想（启用时）
		sys += skillspanel.Prompt()             // Skills tab：可用技能（.pair/skills，渐进式披露）
		sys += "\n\n# 自管理与扩展\n你可自我扩展：skill_list / load_skill（按需读技能全文）/ load_skill_resource（读技能子资源）/ skill_write / skill_delete 管理技能；" +
			"mcp_list / mcp_add / mcp_remove 管理 MCP 服务器；marketplace_search / marketplace_install 从市场检索并安装 MCP 或技能。把可复用的工作方式沉淀成技能。"
		if core.Settings.LuaTools { // 告知 Agent 可自建/优化 Lua 工具（沙箱）
			sys += "\n\n# 自定义工具（Lua）\n可在工作区 .pair/tools/ 下写 .lua 脚本自定义工具（沙箱：仅 string/table/math，无文件/系统访问、单次 10s 超时）。" +
				"脚本须 return {name=, description=, parameters=(JSON Schema 表), run=function(args) return 结果字符串 end}。写好后下次发送即热加载可用——按需扩展或优化你的工具集。"
		}
		sys += "\n\n# 长时记忆检索\n你可以使用以下内部工具检索历史已完成对话的记忆（用于了解之前的工作成果）：\n" +
			"- `memory_search` 搜索历史记忆（标题/摘要/标签/关键点），按关键词筛选\n" +
			"- `memory_list` 列出所有历史记忆（按完成时间倒序）\n" +
			"- `memory_count` 查询记忆总数\n" +
			"注意：新对话开始时系统已自动注入最近的对话摘要到本提示中；如需更详细的历史记录可使用上述工具检索。"
		sys += agent.ProjectRules(root)
		sys += agent.ProjectKnowledge(root, 2500) // 项目知识库概览（.pair/project-info，渐进式披露）
		b.loop = &agent.Loop{Provider: prov, Registry: reg, System: sys, MaxIterations: 30}
	}
	// ── 构建对话上下文 ──
	// 自闭环：loop 已有持久化历史时，前端只发信号（nil → loop.Run 使用 l.History）。
	// 首次调用才从 state.Thread 构建历史。
	var hist []agent.Message
	if len(b.loop.History) > 0 {
		hist = nil // 复用 loop 内部持久化历史
	} else {
		hist = b.history(th)
	}

	// 多 Agent 编排：规划/审核 Agent 每次发送按当前设置重建（改设置即时生效）。
	b.reviewer = nil
	if core.Settings.AIReview {
		b.reviewer = MakeReviewer()
	}
	b.planner = MakePlanner()
	b.evaluator = nil
	if core.Settings.Benchmark {
		b.evaluator = MakeEvaluator()
	}
	b.reloadLuaTools() // Lua 自定义工具：每次发送热重载（.pair/tools/*.lua 增删改即时生效）
	// 写类工具审批门（每次发送按当前设置重设）：
	//   AI 审核开 → 审核模型自动裁决（驳回回灌建议）；否则 手动审核(非自主)→ 用户裁决；其余 → 放行。
	switch {
	case b.reviewer != nil:
		b.loop.Approve = b.aiReviewApprove
	case !b.Cs.AutoReview && !b.Cs.Autonomous:
		b.loop.Approve = b.approve
	default:
		b.loop.Approve = nil
	}

	// 上下文压缩（按当前设置重设，改设置即时生效）：压缩模型 + 窗口上限灌进 loop（见 agent/compress.go）。
	b.loop.Compressor = BuildCompressor()
	b.loop.MaxContextTokens = core.Settings.ContextMaxTokens

	// 自动召回相关项目记忆，附在任务后（让记忆主动浮现，免 Agent 漏查；pull→auto-recall）。
	if mem := agent.RecallMemories(b.root, task, 3); mem != "" {
		task += mem
	}

	// 自主模式：放宽迭代上限 + 提示连续完成整份计划（配合 update_plan 清单一气呵成，不中途停等）。
	task, b.loop.MaxIterations = AutonomousParams(task, b.Cs.Autonomous)

	// 流式助手消息（占位，事件到来时填充）。
	th.Messages = append(th.Messages, state.Message{Role: state.Assistant, Streaming: true})
	b.runThread = th
	b.runIdx = len(th.Messages) - 1

	ctx, cancel := context.WithCancel(context.Background())
	b.mu.Lock()
	b.running = true
	b.stopped = false
	b.cancel = cancel
	b.pending = nil
	b.mu.Unlock()

	b.loop.OnEvent = b.pushEvent // loop 协程调用 → 缓冲（加锁），不碰 UI/Element
	loop := b.loop
	planner := b.planner
	autonomous := b.Cs.Autonomous                                            // 自主模式：跑完整编排（规划→探索→执行→验证→评测→迭代）
	usePlanner := autonomous && planner != nil                               // + 配了规划模型 → 先由规划 Agent 列计划
	autoIter := core.Settings.AutoIterate && core.Settings.ReviewRetries > 0 // 自动迭代改进开关 + 上限
	reviewRetries := core.Settings.ReviewRetries
	go func() {
		runTask := task
		if usePlanner { // 规划阶段：规划 Agent(planModel) 先分解任务 → 计划卡 + 注入执行纲领
			if plan, perr := planner.Plan(ctx, task, hist); perr == nil && len(plan.Steps) > 0 {
				b.pushEvent(agent.Event{Type: agent.EventToolCall, Tool: "update_plan", Args: planToUpdateArgs(plan)})
				runTask = task + "\n\n（规划 Agent 已制定以下计划，请据此连续执行、用 update_plan 更新各步状态）：\n" + planStepsText(plan)
			}
		}
		if autonomous && ctx.Err() == nil { // 探索阶段：只读 Explorer 收集上下文（编排子 Agent）
			b.pushNote("探索阶段：探索 Agent 只读了解项目上下文…")
			if found := b.runRolePhase(ctx, "explorer", roleprompts.DefaultExplorerPrompt, "（只读探索，勿改文件）了解与以下任务相关的项目结构、关键文件与上下文，简要汇报发现：\n"+task, hist); strings.TrimSpace(found) != "" {
				runTask += "\n\n（探索 Agent 的发现，供参考）：\n" + found
			}
		}
		finalMsgs, loopErr := loop.Run(ctx, runTask, hist)
		// 连续错误或达最大迭代：标记 stopped 以跳过后续验证/评测阶段，由 drain 处理收尾。
		if loopErr != nil && (errors.Is(loopErr, agent.ErrConsecToolError) || errors.Is(loopErr, agent.ErrMaxIterations)) {
			b.mu.Lock()
			b.stopped = true
			b.mu.Unlock()
		}
		if autonomous && ctx.Err() == nil && !b.stopped { // 验证阶段：只读 Verifier 确认改动生效（编排子 Agent）
			b.pushNote("验证阶段：验证 Agent 只读确认改动是否生效…")
			b.runRolePhase(ctx, "verifier", roleprompts.DefaultVerifierPrompt, "（只读验证，勿改文件）确认上述任务是否真正完成、改动是否生效，列出遗留问题：\n"+task, stripSystemMsgs(finalMsgs))
		}
		// 评测阶段：任务完成 → 评测模型打分 → 评分卡（复刻参考 bench）。
		if ev := b.evaluateRun(ctx, runTask, finalMsgs); ev != nil && autoIter && !b.stopped {
			// 自动迭代改进（调试角色）：据评测不足重跑 + 重评，分数不再改善即收敛停。
			b.autoIterate(ctx, loop, *ev, finalMsgs, reviewRetries)
		}
		cancel() // 释放 ctx 资源
		b.mu.Lock()
		wasStopped := b.stopped
		b.running = false
		b.mu.Unlock()

		// 对话完成 → 异步生成摘要并写入记忆索引（不影响主流程）
		if !wasStopped {
			th := b.Cs.Store.Active()
			if th != nil && len(th.Messages) >= 2 {
				go b.indexConversation(th)
			}
		}
	}()
	b.startPump()
}

// history 把当前会话的既往消息转成 LLM 上下文。
// 用 agent.BuildHistory 统一排除末条（当前用户消息），避免 loop.Run 重复添加。
func (b *AgentBridge) history(th *state.Thread) []agent.Message {
	msgs := th.Messages
	all := make([]agent.Message, 0, len(msgs))
	for _, m := range msgs {
		content := strings.TrimSpace(m.Text)
		if m.Role == state.Assistant { // 连续性：带上本轮工具活动摘要，免下轮重复探索
			if s := activitySummary(m.Activities); s != "" {
				if content != "" {
					content += "\n"
				}
				content += s
			}
		}
		if m.Streaming || content == "" {
			continue
		}
		role := agent.RoleAssistant
		if m.Role == state.User {
			role = agent.RoleUser
		}
		all = append(all, agent.Message{Role: role, Content: content})
	}
	return agent.BuildHistory(all)
}

// activitySummary 把一条助手消息的工具活动压成一行摘要，供跨轮上下文连续性（Agent 知道上轮做过什么、免重复）。
func activitySummary(acts []state.Activity) string {
	if len(acts) == 0 {
		return ""
	}
	var parts []string
	for _, a := range acts {
		mark := ""
		if a.Done {
			mark = "✓"
			if strings.HasPrefix(strings.TrimSpace(a.Result), "Error:") || strings.Contains(a.Result, "拒绝") {
				mark = "✗"
			}
		}
		parts = append(parts, a.Tool+"("+chat.ArgPreview(a.Args)+")"+mark)
	}
	return "[上轮已执行：" + strings.Join(parts, "、") + "]"
}

// indexConversation 将已完成对话生成摘要并写入记忆索引（供后续 memory_search 等工具检索）。
func (b *AgentBridge) indexConversation(th *state.Thread) {
	if th == nil || len(th.Messages) < 2 {
		return
	}
	// 已有摘要则跳过（避免重复）
	if th.Title != "" && th.Title != "新对话" {
		// 检查记忆索引中是否已有此条
		for _, m := range memory.Search(th.ID) {
			if m.ID == th.ID {
				return
			}
		}
	}

	// 构建消息列表
	msgs := make([]summary.Message, len(th.Messages))
	allContents := make([]string, len(th.Messages))
	assistantContents := make([]string, 0)
	for i, m := range th.Messages {
		role := "user"
		if m.Role == state.Assistant {
			role = "assistant"
		}
		text := m.Text
		msgs[i] = summary.Message{Role: role, Content: text}
		allContents[i] = text
		if m.Role == state.Assistant {
			assistantContents = append(assistantContents, text)
		}
	}

	convInfo := summary.ConvInfo{
		ID:        th.ID,
		Title:     th.Title,
		CreatedAt: "",
		UpdatedAt: "",
		Messages:  msgs,
	}
	s := summary.Generate(convInfo)
	if s == "" {
		return
	}

	completedAt := time.Now().Format("2006-01-02 15:04:05")
	memory.Upsert(memory.Entry{
		ID:           th.ID,
		Title:        th.Title,
		Summary:      s,
		CreatedAt:    convInfo.CreatedAt,
		UpdatedAt:    convInfo.UpdatedAt,
		MessageCount: len(th.Messages),
		Tags:         memory.ExtractTags(allContents),
		KeyPoints:    memory.ExtractKeyPoints(assistantContents),
		CompletedAt:  completedAt,
	})
}

// startPump 帧泵：运行期间周期 drain 把事件应用到流式消息（同终端面板）。
// 用 app.SetInterval 替代 goui animation.Controller（GWui 无帧泵控制器）。
func (b *AgentBridge) startPump() {
	if b.pumpID != 0 || SetIntervalFunc == nil {
		return
	}
	b.pumpID = SetIntervalFunc(120*time.Millisecond, func() { b.drain() })
}

func (b *AgentBridge) stopPump() {
	if b.pumpID != 0 && ClearIntervalFunc != nil {
		ClearIntervalFunc(b.pumpID)
		b.pumpID = 0
	}
}

// approve 审批钩子（loop 协程调用，**阻塞**）：登记裁决通道 + 推一条 EventApproval（让帧泵把对应
// 活动标为「待批准」并渲染允许/拒绝按钮），然后阻塞等用户点击或 ctx 取消。
// 跨线程铁律：只在锁内写 pending/通道、不碰 Element/store。通道与事件在同一把锁内设置，
// 保证 drain 看到 EventApproval 时通道已就绪（resolveApproval 必能找到它）。
func (b *AgentBridge) approve(ctx context.Context, tc agent.ToolCall) (bool, string) {
	ch := make(chan bool, 1)
	b.mu.Lock()
	b.approvalCh = ch
	b.approvalCallID = tc.ID
	b.pending = append(b.pending, agent.Event{Type: agent.EventApproval, CallID: tc.ID, Tool: tc.Function.Name, Args: tc.Function.Arguments})
	b.mu.Unlock()
	select {
	case d := <-ch:
		return d, "" // 人工裁决：拒绝用默认拒绝语（loop 内置）
	case <-ctx.Done():
		b.mu.Lock()
		if b.approvalCh == ch {
			b.approvalCh = nil
			b.approvalCallID = ""
		}
		b.mu.Unlock()
		return false, ""
	}
}

// resolveApproval UI 线程（用户点「允许/拒绝」）：把裁决送给阻塞中的 loop 协程，清审批态 + 隐藏按钮。
// 结果（执行输出或「已拒绝」观察）随后由 loop 的 tool_result 事件回填。
func (b *AgentBridge) ResolveApproval(callID string, ok bool) {
	b.mu.Lock()
	ch := b.approvalCh
	match := ch != nil && b.approvalCallID == callID
	if match {
		b.approvalCh = nil
		b.approvalCallID = ""
	}
	b.mu.Unlock()
	if !match {
		return
	}
	ch <- ok // 缓冲=1，非阻塞
	if m := b.streamingMsg(); m != nil {
		for i := range m.Activities {
			if m.Activities[i].CallID == callID {
				m.Activities[i].AwaitingApproval = false
			}
		}
	}
	b.Cs.SetState()
}

// askUser ask_user 工具处理器（loop 协程调用，**阻塞**）：登记回答通道，阻塞等用户回答或 ctx 取消。
// 问答卡 UI 由 drain 处理 EventToolCall(ask_user) 时据参数渲染（见 applyEvent），故此处只管阻塞取答。
func (b *AgentBridge) askUser(ctx context.Context, args map[string]any) (string, error) {
	ch := make(chan string, 1)
	b.mu.Lock()
	b.askCh = ch
	b.mu.Unlock()
	select {
	case ans := <-ch:
		return ans, nil
	case <-ctx.Done():
		b.mu.Lock()
		if b.askCh == ch {
			b.askCh = nil
		}
		b.mu.Unlock()
		return "（用户未回答，已取消提问）", nil
	}
}

// registerAskTool 注册 ask_user（handler 闭包持有 bridge → 能阻塞等 UI 回答）。
func (b *AgentBridge) registerAskTool(r *agent.Registry) {
	r.Register(&agent.Tool{
		Name: "ask_user",
		Description: "向用户提问并等待回答（用于关键决策、歧义澄清，别滥用）。question 必填；options 可选(给用户快捷选项)；" +
			"用户也可自由输入。调用会阻塞直到用户回答。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{"type": "string", "description": "要问用户的问题"},
				"options":  map[string]any{"type": "array", "description": "可选：快捷选项", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"question"},
		},
		ReadOnly: true, // 提问非写操作，免审批
		Handler:  b.askUser,
	})
}

// resolveAsk UI 线程（用户点选项/输入回答）：把回答送给阻塞中的 loop 协程，清问答卡。
func (b *AgentBridge) ResolveAsk(answer string) {
	b.mu.Lock()
	ch := b.askCh
	b.askCh = nil
	b.mu.Unlock()
	b.Cs.ClearAsk()
	if ch != nil {
		ch <- answer // 缓冲=1，非阻塞
	}
	b.Cs.SetState()
}

// drain 每次定时器回调（UI 线程）把缓冲事件应用到流式消息 + 重绘；结束即停泵。
func (b *AgentBridge) drain() {
	b.mu.Lock()
	evs := b.pending
	b.pending = nil
	done := !b.running
	b.mu.Unlock()

	if len(evs) > 0 {
		for _, e := range evs {
			b.applyEvent(e)
		}
		b.syncWorkspaceEdits(evs)
		b.Cs.SendSeq++
	}

	// 节流：每秒最多 4 次 SetState + SaveHistory（流式输出时不需要每帧刷新 UI）
	now := time.Now()
	throttled := !done && now.Sub(b.lastDrainTime) < 250*time.Millisecond

	if len(evs) > 0 && !throttled {
		b.lastDrainTime = now
		b.Cs.SetState()
		b.Cs.SaveHistory()
	}
	if done {
		if msg := b.streamingMsg(); msg != nil {
			msg.Streaming = false
			if b.stopped && !strings.Contains(msg.Text, "[已停止]") {
				if strings.TrimSpace(msg.Text) != "" {
					msg.Text += "\n\n"
				}
				msg.Text += "[已停止]"
			}
			if b.Cs.AutoCollapse {
				msg.Collapsed = true
			}
		}
		b.Cs.ClearAsk()
		b.Cs.SaveHistory()
		b.stopPump()
		if len(evs) == 0 || b.stopped || !throttled {
			b.Cs.SetState()
		}
	}
}

// streamingMsg 取当前流式消息指针（越界返回 nil）。
func (b *AgentBridge) streamingMsg() *state.Message {
	if b.runThread == nil || b.runIdx < 0 || b.runIdx >= len(b.runThread.Messages) {
		return nil
	}
	return &b.runThread.Messages[b.runIdx]
}

// applyPlan 解析 update_plan 工具参数里的计划清单，存入 ChatState（置顶渲染）。
func (b *AgentBridge) applyPlan(argsJSON string) {
	b.Cs.ApplyPlan(argsJSON)
}

func (b *AgentBridge) applyEvent(e agent.Event) {
	m := b.streamingMsg()
	if m == nil {
		return
	}
	switch e.Type {
	case agent.EventThinking:
		m.Thinking += e.Content
		// Timeline：合并连续 thinking 条目，保留输出顺序
		if n := len(m.Timeline); n > 0 && m.Timeline[n-1].Kind == "thinking" {
			m.Timeline[n-1].Content += e.Content
		} else {
			m.Timeline = append(m.Timeline, state.TimelineEntry{Kind: "thinking", Content: e.Content, AgentName: e.AgentName})
		}
	case agent.EventContent:
		m.Text += e.Content
		// Timeline：合并连续 content 条目，保留输出顺序
		if n := len(m.Timeline); n > 0 && m.Timeline[n-1].Kind == "content" {
			m.Timeline[n-1].Content += e.Content
		} else {
			m.Timeline = append(m.Timeline, state.TimelineEntry{Kind: "content", Content: e.Content, AgentName: e.AgentName})
		}
	case agent.EventToolCall:
		switch e.Tool {
		case "update_plan": // 计划单独渲染为置顶清单卡，不作通用工具活动行
			b.applyPlan(e.Args)
		case "ask_user": // 提问单独渲染为问答卡（输入区上方），不作通用活动行
			b.Cs.ApplyAsk(e.Args)
		default:
			m.Activities = append(m.Activities, state.Activity{CallID: e.CallID, Tool: e.Tool, Args: e.Args, AgentName: e.AgentName})
			m.Timeline = append(m.Timeline, state.TimelineEntry{
				Kind: "tool", Tool: e.Tool, Args: e.Args, CallID: e.CallID, AgentName: e.AgentName,
			})
		}
	case agent.EventApproval:
		// 标记对应活动为「待批准」（tool_call 已建活动，按 CallID 找；兜底补建）。
		found := false
		for i := range m.Activities {
			if m.Activities[i].CallID == e.CallID {
				m.Activities[i].AwaitingApproval = true
				found = true
				break
			}
		}
		if !found {
			m.Activities = append(m.Activities, state.Activity{CallID: e.CallID, Tool: e.Tool, Args: e.Args, AgentName: e.AgentName, AwaitingApproval: true})
		}
		// Timeline 同步标记
		for i := range m.Timeline {
			if m.Timeline[i].Kind == "tool" && m.Timeline[i].CallID == e.CallID {
				m.Timeline[i].AwaitingApproval = true
				break
			}
		}
	case agent.EventToolResult:
		for i := len(m.Activities) - 1; i >= 0; i-- {
			if m.Activities[i].CallID == e.CallID || (m.Activities[i].Tool == e.Tool && !m.Activities[i].Done) {
				m.Activities[i].Result = e.Content
				m.Activities[i].Done = true
				m.Activities[i].AwaitingApproval = false
				break
			}
		}
		// Timeline 同步更新结果（反向搜索以匹配最新）
		for i := len(m.Timeline) - 1; i >= 0; i-- {
			if m.Timeline[i].Kind == "tool" && (m.Timeline[i].CallID == e.CallID || (m.Timeline[i].Tool == e.Tool && !m.Timeline[i].Done)) {
				m.Timeline[i].Result = e.Content
				m.Timeline[i].Done = true
				m.Timeline[i].AwaitingApproval = false
				break
			}
		}
	case agent.EventCompacted, agent.EventCircling, agent.EventNotice:
		m.Notes = append(m.Notes, e.Content) // 系统提示（压缩/绕圈）：素色一行显示在卡内
	case agent.EventUsage:
		if e.Usage != nil && b.runThread != nil {
			b.runThread.AccumulateAPIUsage(
				e.Usage.PromptTokens,
				e.Usage.CompletionTokens,
				e.Usage.PromptCacheHitTokens,
				e.Usage.PromptCacheMissTokens,
			)
			// 持久化到项目级统计（安装目录 .pair/stats.json，独立于对话记录）
			modelName := core.Settings.ExecuteModel
			if modelName == "" {
				modelName = core.Settings.Model
			}
			if modelName == "" {
				modelName = "unknown"
			}
			core.RecordLLMCall(
				e.Usage.PromptTokens,
				e.Usage.CompletionTokens,
				e.Usage.PromptCacheHitTokens,
				e.Usage.PromptCacheMissTokens,
				0,
				modelName,
				0,
				"¥",
			)
		}
	case agent.EventEvaluation:
		var ev agent.Evaluation
		if json.Unmarshal([]byte(e.Args), &ev) == nil {
			m.Eval = &state.Eval{Total: ev.Total, Completion: ev.Scores.Completion, Correctness: ev.Scores.Correctness,
				Depth: ev.Scores.Depth, Efficiency: ev.Scores.Efficiency,
				Strengths: ev.Strengths, Weaknesses: ev.Weaknesses, Feedback: ev.Feedback}
		}
	case agent.EventFinal, agent.EventDone:
		if strings.TrimSpace(e.Content) != "" {
			m.Text = e.Content
		}
		m.Streaming = false
		// DoneReason 在 EventDone 时携带完成原因，前端可据此显示不同收尾提示
	case agent.EventError:
		if b.stopped {
			return // 用户主动停止：抑制底层取消错误，由 drain 收尾标 [已停止]
		}
		if strings.TrimSpace(m.Text) != "" {
			m.Text += "\n\n"
		}
		m.Text += "[错误] " + e.Content
		m.Streaming = false
	}
}

// syncWorkspaceEdits 在 Agent 成功写/改文件后，刷新文件树并重载已打开（且无未存改动）的对应文件，
// 让改动在 IDE 里即时可见（闭环）。仅 UI 线程（drain 内）调用。文件树未构建/文件未打开时均安全无操作。
func (b *AgentBridge) syncWorkspaceEdits(evs []agent.Event) {
	root := b.root
	if root == "" {
		root = core.Root()
	}
	refreshed := false
	for _, e := range evs {
		if e.Type != agent.EventToolResult || (e.Tool != "write_file" && e.Tool != "edit_file") {
			continue
		}
		r := strings.TrimSpace(e.Content)
		if strings.HasPrefix(r, "Error:") || strings.Contains(r, "拒绝") {
			continue // 失败或被拒，磁盘未变
		}
		if p := b.toolPath(root, e.CallID); p != "" {
			if filetreepanel.Panel != nil {
				filetreepanel.Panel.RefreshPath(p)
			}
			editorpanel.Editor.ReloadIfOpen(p)
			refreshed = true
		}
	}
	if !refreshed {
		// 所有事件都无有效路径时 fallback 到全量刷新
		if filetreepanel.Panel != nil {
			filetreepanel.Panel.Refresh()
		}
	}
}

// toolPath 从活动（按 CallID 找）参数 JSON 取 path，解析为工作区内绝对路径（与编辑器标签路径同形可比）。
func (b *AgentBridge) toolPath(root, callID string) string {
	m := b.streamingMsg()
	if m == nil {
		return ""
	}
	for _, a := range m.Activities {
		if a.CallID != callID {
			continue
		}
		var args map[string]any
		if json.Unmarshal([]byte(a.Args), &args) != nil {
			return ""
		}
		p, _ := args["path"].(string)
		if strings.TrimSpace(p) == "" {
			return ""
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		return filepath.Clean(p)
	}
	return ""
}

// buildProvider 选 Provider：优先用「设置」里配的（帮助→打开设置），否则回退环境变量。无 key 返回 nil。
func BuildProvider() agent.Provider {
	if core.Configured() {
		return &agent.OpenAIProvider{BaseURL: core.Settings.BaseURL, APIKey: core.Settings.APIKey, Model: core.MainModel(),
			Temperature: core.Temperature(), MaxTokens: core.Settings.MaxTokens, ThinkingMode: core.Settings.ThinkingMode}
	}
	for _, c := range []struct{ env, base, model string }{
		{"DEEPSEEK_API_KEY", "https://api.deepseek.com/v1", "deepseek-chat"},
		{"OPENAI_API_KEY", "https://api.openai.com/v1", "gpt-4o"},
		{"DASHSCOPE_API_KEY", "https://dashscope.aliyuncs.com/compatible-mode/v1", "qwen-plus"},
		{"MOONSHOT_API_KEY", "https://api.moonshot.cn/v1", "moonshot-v1-8k"},
		{"OPENROUTER_API_KEY", "https://openrouter.ai/api/v1", "anthropic/claude-3.5-sonnet"},
	} {
		if k := os.Getenv(c.env); k != "" {
			return &agent.OpenAIProvider{BaseURL: c.base, APIKey: k, Model: c.model, Temperature: -1}
		}
	}
	return nil
}

// buildCompressor 据「压缩模型」设置建轻量压缩 Provider（上下文压缩用，见 agent/compress.go）。
// CompressEnabled 关 / 配不齐 → 返回 nil（压缩退化为规则式摘要）。Key/地址留空则复用主模型。
// 复刻参考 agent-manager.ts：默认 non-thinking、temperature 0.3、maxTokens 4096。
func BuildCompressor() agent.Provider {
	if !core.Settings.CompressEnabled {
		return nil
	}
	key := strings.TrimSpace(core.Settings.CompressAPIKey)
	if key == "" {
		key = strings.TrimSpace(core.Settings.APIKey) // 留空复用主模型 Key
	}
	base := strings.TrimSpace(core.Settings.CompressBaseURL)
	if base == "" {
		base = strings.TrimSpace(core.Settings.BaseURL)
	}
	model := strings.TrimSpace(core.Settings.CompressModel)
	if model == "" || key == "" || base == "" {
		return nil // 配不齐 → 规则式兜底
	}
	mode := core.Settings.CompressThinkingMode
	if mode == "" {
		mode = "non-thinking"
	}
	return &agent.OpenAIProvider{BaseURL: base, APIKey: key, Model: model,
		Temperature: 0.3, MaxTokens: 4096, ThinkingMode: mode}
}

// buildPlanProvider / buildReviewProvider 用规划/审核模型建 Provider（多 Agent 编排用，见 agent/planner.go、reviewer.go）。
// 复用主模型接口+Key、仅换模型名 + non-thinking（复刻参考 agent-manager：规划/审核用 non-thinking）。
// 模型名空 / 主模型未配 → nil（不启用该角色 Agent，退化为单循环）。
func BuildPlanProvider() agent.Provider   { return BuildRoleProvider(core.Settings.PlanModel) }
func BuildReviewProvider() agent.Provider { return BuildRoleProvider(core.Settings.ReviewModel) }

func BuildRoleProvider(model string) agent.Provider {
	model = strings.TrimSpace(model)
	base := strings.TrimSpace(core.Settings.BaseURL)
	key := strings.TrimSpace(core.Settings.APIKey)
	if model == "" || base == "" || key == "" {
		return nil
	}
	return &agent.OpenAIProvider{BaseURL: base, APIKey: key, Model: model, Temperature: -1, ThinkingMode: "non-thinking"}
}

// makePlanner / makeReviewer 建规划/审核 Agent（按设置）。函数变量便于测试注入 mock provider。
// 角色系统提示从 config/roles/<id>.md 加载（缺失回退内置默认）+ 该角色哲学（rolePhilosophy）——非硬编码。
var MakePlanner = func() *agent.Planner {
	if pp := BuildPlanProvider(); pp != nil {
		return &agent.Planner{Provider: pp, SystemPrompt: roleprompts.LoadRolePrompt("planner", agent.DefaultPlannerPrompt()) + roleprompts.RolePhilosophy("planner")}
	}
	return nil
}
var MakeReviewer = func() *agent.Reviewer {
	if rp := BuildReviewProvider(); rp != nil {
		return &agent.Reviewer{Provider: rp, SystemPrompt: roleprompts.LoadRolePrompt("reviewer", agent.DefaultReviewerPrompt()) + roleprompts.RolePhilosophy("reviewer")}
	}
	return nil
}

// buildJudgeProvider 评测模型 Provider：用执行模型 + temperature 0.2 + maxTokens 2000（复刻参考 evaluator.ts）。
func BuildJudgeProvider() agent.Provider {
	base := strings.TrimSpace(core.Settings.BaseURL)
	key := strings.TrimSpace(core.Settings.APIKey)
	model := core.MainModel()
	if base == "" || key == "" || model == "" {
		return nil
	}
	return &agent.OpenAIProvider{BaseURL: base, APIKey: key, Model: model, Temperature: 0.2, MaxTokens: 2000}
}

var MakeEvaluator = func() *agent.Evaluator {
	if jp := BuildJudgeProvider(); jp != nil {
		return &agent.Evaluator{Provider: jp, SystemPrompt: roleprompts.LoadRolePrompt("judge", agent.DefaultJudgePrompt())}
	}
	return nil
}

// reloadLuaTools 热重载 Lua 自定义工具：卸载上轮加载的，再从工作区 .pair/tools/*.lua 沙箱加载当前的。
// 每次发送调用，使 Agent/用户增删改 .lua 即时生效（动态工具增减/优化，见 agent/luatool.go）。
func (b *AgentBridge) reloadLuaTools() {
	if b.loop == nil || b.loop.Registry == nil {
		return
	}
	for _, n := range b.luaToolNames {
		// 		b.loop.Registry.Unregister(n)
		_ = n // Unregister not available
	}
	b.luaToolNames = nil
	if !core.Settings.LuaTools {
		return
	}
	root := b.root
	if root == "" {
		root = core.Root()
	}
	b.luaToolNames = agent.LoadLuaTools(b.loop.Registry, filepath.Join(root, ".pair", "tools"))
}

// applyIgnoreDirs 合并全局设置 + 项目级 .pair/ignore 的忽略目录，注入 search/explore 工具。
func ApplyIgnoreDirs(root string) {
	dirs := core.Settings.IgnoreDirs
	if proj := readProjectIgnore(root); len(proj) > 0 {
		dirs = append(dirs, proj...)
	}
	if len(dirs) > 0 {
		agent.SetExtraSkipDirs(dirs)
	} else {
		agent.SetExtraSkipDirs(nil)
	}
}

// readProjectIgnore 读项目级 .pair/ignore（一行一个目录名；# 注释与空行忽略）。
func readProjectIgnore(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, ".pair", "ignore"))
	if err != nil {
		return nil
	}
	var out []string
	for _, ln := range strings.Split(string(data), "\n") {
		if ln = strings.TrimSpace(ln); ln != "" && !strings.HasPrefix(ln, "#") {
			out = append(out, ln)
		}
	}
	return out
}

// pushEvent loop/角色阶段协程调用 → 缓冲事件（加锁），不碰 UI/Element（帧泵 drain 时应用）。
func (b *AgentBridge) pushEvent(e agent.Event) {
	b.mu.Lock()
	b.pending = append(b.pending, e)
	b.mu.Unlock()
}

// runRolePhase 跑一个只读角色阶段（探索/验证，编排子 Agent）：用「角色提示 + 角色哲学」的 Loop，
// 禁写（任何写/命令工具自动拦截）、限 6 轮，事件流进当前消息，返回末条助手文本（发现/验证结论）。
func (b *AgentBridge) runRolePhase(ctx context.Context, roleID, defPrompt, task string, hist []agent.Message) string {
	if b.loop == nil || b.loop.Provider == nil {
		return ""
	}
	phase := &agent.Loop{
		Provider:      b.loop.Provider,
		Registry:      b.loop.Registry,
		System:        roleprompts.LoadRolePrompt(roleID, defPrompt) + roleprompts.RolePhilosophy(roleID),
		MaxIterations: 6,
		OnEvent:       b.pushEvent,
		Approve: func(context.Context, agent.ToolCall) (bool, string) {
			return false, "本阶段只读，请勿写文件或执行写命令"
		},
	}
	msgs, _ := phase.Run(ctx, task, hist)
	return lastAssistantText(msgs)
}

// lastAssistantText 取末条有正文的助手消息（角色阶段的结论）。
func lastAssistantText(msgs []agent.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == agent.RoleAssistant && strings.TrimSpace(msgs[i].Content) != "" {
			return strings.TrimSpace(msgs[i].Content)
		}
	}
	return ""
}

// autoIterateTarget 评分达此即满意，不再自动迭代。
const autoIterateTarget = 80

// evaluateRun 评测一轮 + 推评分卡（未配评测器/无工具活动/被停止/失败 → 返回 nil）。
func (b *AgentBridge) evaluateRun(ctx context.Context, task string, msgs []agent.Message) *agent.Evaluation {
	b.mu.Lock()
	stopped := b.stopped
	b.mu.Unlock()
	if b.evaluator == nil || stopped || !agent.HasToolActivity(msgs) {
		return nil
	}
	ev, err := b.evaluator.Evaluate(ctx, task, agent.SummarizeRun(msgs))
	if err != nil || ev.Total <= 0 {
		return nil
	}
	if data, e := json.Marshal(ev); e == nil {
		b.mu.Lock()
		b.pending = append(b.pending, agent.Event{Type: agent.EventEvaluation, Args: string(data)})
		b.mu.Unlock()
	}
	return &ev
}

// autoIterate 据评测建议自动迭代改进（复刻参考 开始自动迭代改进 + convergence）：
// 分数越低迭代越多（<40→3 / <70→2 / 否则 1），上限 reviewRetries；每轮据不足重跑 + 重评，分数不再改善即收敛停。
func (b *AgentBridge) autoIterate(ctx context.Context, loop *agent.Loop, ev agent.Evaluation, prevMsgs []agent.Message, reviewRetries int) {
	maxIters := 1 // 复刻参考 maxIterations = score<40?3:score<70?2:1
	switch {
	case ev.Total < 40:
		maxIters = 3
	case ev.Total < 70:
		maxIters = 2
	}
	if maxIters > reviewRetries {
		maxIters = reviewRetries // ReviewRetries 设置作上限（用户控制）
	}
	best := ev.Total
	hist := stripSystemMsgs(prevMsgs)
	for iter := 0; iter < maxIters && best < autoIterateTarget; iter++ {
		b.mu.Lock()
		stopped := b.stopped
		b.mu.Unlock()
		if stopped {
			return
		}
		b.pushNote("自动迭代改进 第 " + strconv.Itoa(iter+1) + " 次（上轮 " + strconv.Itoa(best) + "/100，目标 ≥" + strconv.Itoa(autoIterateTarget) + "）…")
		msgs, _ := loop.Run(ctx, buildImproveTask(ev), hist)
		ev2 := b.evaluateRun(ctx, "改进任务", msgs)
		if ev2 == nil || ev2.Total <= best { // 失败或无改善 → 收敛停（复刻 convergence）
			return
		}
		best, ev = ev2.Total, *ev2
		hist = stripSystemMsgs(msgs)
	}
}

// buildImproveTask 据评测不足拼出改进任务（让执行 Agent 实际修改而非只给建议，复刻参考改进 prompt）。
func buildImproveTask(ev agent.Evaluation) string {
	var sb strings.Builder
	sb.WriteString("上一轮任务评分 " + strconv.Itoa(ev.Total) + "/100，存在以下不足，请针对性改进")
	sb.WriteString("（先 read_file 分析再 edit_file/write_file 实际修改，不要只给建议）：\n")
	for _, w := range ev.Weaknesses {
		sb.WriteString("- " + w + "\n")
	}
	if strings.TrimSpace(ev.Feedback) != "" {
		sb.WriteString("\n总评：" + ev.Feedback)
	}
	if dbg := roleprompts.RoleSpecificPhilosophy("debugger"); dbg != "" { // 自动迭代＝调试角色，注入其哲学指导
		sb.WriteString(dbg)
	}
	sb.WriteString("\n\n全部改进完成后调用 finish_task 工具。")
	return sb.String()
}

// pushNote 推一条系统提示（自动迭代等）到流式消息（素色显示）。
func (b *AgentBridge) pushNote(text string) {
	b.mu.Lock()
	b.pending = append(b.pending, agent.Event{Type: agent.EventCompacted, Content: text})
	b.mu.Unlock()
}

// stripSystemMsgs 去掉开头的 system 消息（作为下一轮 history，避免 loop.Run 重复加 system）。复制以防后续追加污染。
func stripSystemMsgs(msgs []agent.Message) []agent.Message {
	i := 0
	for i < len(msgs) && msgs[i].Role == agent.RoleSystem {
		i++
	}
	return append([]agent.Message(nil), msgs[i:]...)
}

// aiReviewApprove AI 审核裁决（loop 协程调用，不阻塞 UI）：写类工具交审核模型判，
// 通过放行，驳回/需要修改则把建议作反馈回灌（loop 据此让执行 Agent 改道）。审核模型故障→放行（审核是增强非强制）。
func (b *AgentBridge) aiReviewApprove(ctx context.Context, tc agent.ToolCall) (bool, string) {
	if b.reviewer == nil || !agent.NeedsReview(tc.Function.Name) {
		return true, ""
	}
	v, err := b.reviewer.Review(ctx, tc)
	if err != nil || v.Approved() {
		return true, ""
	}
	return false, v.FeedbackText()
}

// planToUpdateArgs 把规划 Agent 的计划转成 update_plan 工具参数 JSON（复用既有计划卡渲染路径）。
func planToUpdateArgs(plan agent.Plan) string {
	steps := make([]map[string]string, 0, len(plan.Steps))
	for _, s := range plan.Steps {
		steps = append(steps, map[string]string{"step": s.Description, "status": "pending"})
	}
	out, _ := json.Marshal(map[string]any{"plan": steps})
	return string(out)
}

// planStepsText 把计划转成给执行 Agent 的纲领文本（思路 + 编号步骤）。
func planStepsText(plan agent.Plan) string {
	var sb strings.Builder
	if strings.TrimSpace(plan.Reasoning) != "" {
		sb.WriteString(plan.Reasoning + "\n")
	}
	for i, s := range plan.Steps {
		sb.WriteString(strconv.Itoa(i+1) + ". " + s.Description + "\n")
	}
	return sb.String()
}
