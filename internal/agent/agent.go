// Agent 独立基座 —— 自闭环生命周期管理。
//
// 设计目标：
//
//	AgentBase 封装 Agent 的完整生命周期（Init → Run → Shutdown），
//	使 Agent 可作为独立库嵌入任意 Go 程序（Web 服务器、CLI 工具、桌面应用等）。
//
// 自闭环原则：
//   - 所有子系统（Loop、SessionManager、Registry、Store）在 Init 中初始化
//   - Run 阻塞运行主循环，外部通过 context 取消
//   - Shutdown 安全释放所有资源
//   - 宿主只需三行调用：NewAgentBase → Init → Run → Shutdown
package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// ─── Agent 基座配置 ─────────────────────────────────────────

// AgentConfig 宿主注入的全部配置（依赖倒置：Agent 不感知宿主具体实现）。
type AgentConfig struct {
	// 工作区根路径（必填）。用于对话存储、快照、Skill 加载等。
	WorkspaceRoot string
	// LLM Provider（必填）。Agent 通过它调用模型。
	Provider Provider
	// 系统提示词（可选）。空则用默认系统提示。
	SystemPrompt string
	// 上下文压缩器（可选）。空则规则式摘要。
	Compressor Compressor
	// 规划 Provider（可选）。非空时启用自主模式双层 Loop。
	PlanProvider Provider
	// 审核 Provider（可选）。非空+ReviewMode="auto" 时启用 AI 审核。
	ReviewProvider Provider

	// OnEvent 事件回调（可选）。宿主通过此回调接收 Agent 流式事件。
	OnEvent func(Event)
	// OnFeedback 用户反馈回调（可选）。每次 LLM 调用前检查。
	OnFeedback func() string

	// 最大迭代数（<=0 使用内部默认 30）
	MaxIterations int
	// 上下文 token 上限（>0 启用压缩）
	MaxContextTokens int

	// 自主模式标志（双层 Loop：设计者→执行者）
	Autonomous bool
	// 审核模式："auto"=AI审核, "manual"=手动审批, "off"=全部放行
	ReviewMode string
}

// ─── Agent 基座 ─────────────────────────────────────────────

// AgentBase 是 Agent 的独立运行基座，封装完整的自闭环生命周期。
// 使用方法：
//
//	base := NewAgentBase(cfg)
//	if err := base.Init(); err != nil { ... }
//	go base.Run(ctx)  // 或 base.Run(ctx) 阻塞
//	// ... 等待信号 ...
//	base.Shutdown()
type AgentBase struct {
	Config AgentConfig

	// 核心组件（Init 后可用）
	SessionMgr    *SessionManager
	Registry      *Registry
	Store         ConversationStore
	ExecutionPlan *ExecutionManager
	Plugins       *PluginHost // 插件宿主（对齐 harness 一切皆插件）

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	done    chan struct{}
}

// NewAgentBase 创建 Agent 基座实例。
// 仅做配置保存，不执行任何初始化。需调用 Init() 完成初始化。
func NewAgentBase(cfg AgentConfig) *AgentBase {
	return &AgentBase{
		Config: cfg,
		done:   make(chan struct{}),
	}
}

// Init 初始化 Agent 所有子系统：
//  1. 创建工作区目录结构（.pair/ 等）
//  2. 初始化存储（SQLite/JSONL）
//  3. 创建工具注册表并注册默认工具
//  4. 初始化会话管理器
//  5. 初始化快照跟踪器
//  6. 初始化执行计划管理器
//
// 可多次安全调用（第二次起为 no-op）。
func (a *AgentBase) Init() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running || a.SessionMgr != nil {
		return nil // 已初始化
	}

	root := a.Config.WorkspaceRoot
	if root == "" {
		return fmt.Errorf("AgentBase: WorkspaceRoot 不能为空")
	}

	// 1. 确保 .pair 目录存在
	pairDir := filepath.Join(root, ".pair")
	if err := os.MkdirAll(pairDir, 0755); err != nil {
		return fmt.Errorf("AgentBase: 创建 .pair 目录失败: %w", err)
	}
	for _, sub := range []string{"conversations", "snapshots", "skills", "memory", "project-info", "tools"} {
		if err := os.MkdirAll(filepath.Join(pairDir, sub), 0755); err != nil {
			return fmt.Errorf("AgentBase: 创建 %s 目录失败: %w", sub, err)
		}
	}

	// 2. 初始化存储（JSONL）
	store := NewMessageStore(root)
	a.Store = store

	// 3. 初始化工具注册表（★ 宿主框架协议工具：SystemTool 会话绑定；
	//    业务工具由磁盘插件 .pair/plugins/tool-* 注册）
	registry := NewRegistry()
	RegisterHostFrameworkTools(registry, root)

	// 3.5 初始化插件宿主（对齐 harness「一切皆插件」）+ cordis 动态插件工具
	ph := NewPluginHost(registry, store, root)
	RegisterCordisTools(registry, ph, root)
	// ★ 框架能力（workspaceRoot 服务 + 内置工具集模板）已内联 NewPluginHost，
	//   不再以插件形态装配（不可启停、不出现在插件列表）。
	// ★ 工具集管理工具 + 启动自动装载（.pair/toolsets/ + 全局）
	RegisterToolsetTools(registry, root, ph)
	LoadAllToolsets(ph, root)
	SetGlobalPluginHost(ph)
	a.Plugins = ph

	// 3.6 装配静态插件（.pair/cordis.patch.json，跨重启存续；条目失败不致命，
	//     文件解析失败致命——配置坏了需要用户修复）
	if err := ph.LoadCordisPatch(filepath.Join(pairDir, "cordis.patch.json")); err != nil {
		return fmt.Errorf("AgentBase: 装配 cordis.patch.json 失败: %w", err)
	}

	// 4. 初始化会话管理器
	sm := NewSessionManager()
	sm.SetWorkspaceRoot(root)
	// 全局事件监听：将 SessionManager 的 GlobalEvent 转换为宿主 OnEvent
	go func() {
		ch := sm.SubscribeAll()
		for ge := range ch {
			if a.Config.OnEvent != nil {
				a.Config.OnEvent(ge.Event)
			}
		}
	}()

	// 5. 初始化快照跟踪器
	InitTracker(root)

	// 6. 初始化执行计划管理器（任务追踪用）
	InitExecutionManager(root)
	InitPlanManager(root)

	// 7. 设置 Skills 加载路径
	SkillProjectDir = filepath.Join(root, ".pair", "skills")
	SkillSystemDir = filepath.Join(root, "config", "skills")

	// 8. 设置 CodeGraph DB
	SetCodeGraphDB(sm.RawDB())
	SetCodeGraphRoot(root) // 共享 DB 归属主项目（多项目时其他项目用独立 JSONStore）

	// 9. 初始化执行状态管理器
	InitExecStateManager(root)

	a.Registry = registry
	a.SessionMgr = sm
	log.Printf("[AgentBase] Agent 基座初始化完成 (工作区: %s)", root)
	return nil
}

// Run 启动 Agent 主循环（阻塞）。
// 创建一个 Loop 实例直接运行，直到任务完成或 ctx 取消。
// 这是自闭环模式：Agent 独立运行，不依赖 Web/UI 层。
func (a *AgentBase) Run(ctx context.Context) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		<-a.done
		return nil
	}
	a.running = true
	ctx, a.cancel = context.WithCancel(ctx)
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
		close(a.done)
	}()

	// 直接创建并运行 Loop（自闭环模式；★ 走全局 LoopFactory：插件装配器可覆盖参数/实现）
	loopOpts := LoopOpts{
		Provider:         a.Config.Provider,
		Registry:         a.Registry,
		System:           a.Config.SystemPrompt,
		MaxIterations:    a.Config.MaxIterations,
		MaxContextTokens: a.Config.MaxContextTokens,
		Compressor:       a.Config.Compressor,
		Autonomous:       a.Config.Autonomous,
		WorkspaceRoot:    a.Config.WorkspaceRoot,
		ReviewMode:       a.Config.ReviewMode,
		ReviewProvider:   a.Config.ReviewProvider,
	}
	loopHandle, loopErr := CreateLoop(loopOpts)
	if loopErr != nil {
		return fmt.Errorf("创建 agent 循环失败: %w", loopErr)
	}
	loop := loopHandle.Loop()
	if a.Config.OnEvent != nil {
		loop.OnEvent = a.Config.OnEvent
	}
	if a.Config.OnFeedback != nil {
		loop.OnFeedback = a.Config.OnFeedback
	}

	_, err := loop.Run(ctx, "", nil)
	return err
}

// Shutdown 安全关闭 Agent 所有子系统。
func (a *AgentBase) Shutdown() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running && a.SessionMgr == nil {
		return
	}

	log.Println("[AgentBase] 正在关闭 Agent 基座…")

	// 取消正在运行的 Loop
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}

	// 停止所有会话
	if a.SessionMgr != nil {
		a.SessionMgr.Stop("")
	}

	a.running = false
	log.Println("[AgentBase] Agent 基座已关闭")
}

// IsRunning 查询 Agent 是否在运行中。
func (a *AgentBase) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}
