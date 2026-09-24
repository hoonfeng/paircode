package agent

import (
	"context"
	"sync"
)

// LoopFactory 装配 agent 循环的工厂接口（对齐 AgentFactory 单槽位语义）。
//
// 外部 AgentLoop 插件：是一个 cordis
// 插件，通过 ctx.plugin(AgentLoop, config) 在装配期装载；循环实现经
// AgentRegistry.setFactory() 注册为唯一工厂槽位（重复注册抛错），整体替换 =
// 装配期换一个实现 AgentFactory 的插件。
//
// gou-ide 对应物：Loop 是 Go 核心（internal/agent/loop.go），默认实现 goLoopFactory
// 用现有 *Loop 构建；宿主 Go 代码可用 ReplaceLoopFactory 显式替换工厂（单槽位，
// 返回还原函数），插件（JS 装配器）经 ctx.loopFactory 服务注册参数级装配。
// 会话层（Session/持久化/事件协议）与 *Loop 字段深度耦合，故 LoopHandle 提供
// Loop() 访问器暴露内核——工厂决定「如何装配内核」，会话层不变。
type LoopFactory interface {
	// Create 依据装配参数创建循环句柄。
	Create(opts LoopOpts) (LoopHandle, error)
}

// LoopHandle 循环句柄：Run 启动并阻塞至结束（*Loop 天然满足）。
// Loop() 返回底层内核 *Loop（默认实现即自身；自定义实现须包装一个 *Loop，
// 因为会话层字段/回调直接挂在 *Loop 上）。
type LoopHandle interface {
	Run(ctx context.Context, task string, history []Message) ([]Message, error)
	Loop() *Loop
}

// loopHandleImpl 默认句柄：直接持有 *Loop。
type loopHandleImpl struct{ loop *Loop }

func (h loopHandleImpl) Run(ctx context.Context, task string, history []Message) ([]Message, error) {
	return h.loop.Run(ctx, task, history)
}
func (h loopHandleImpl) Loop() *Loop { return h.loop }

// goLoopFactory 默认工厂：用现有 Loop 核心构建。
type goLoopFactory struct{}

func (goLoopFactory) Create(opts LoopOpts) (LoopHandle, error) {
	return loopHandleImpl{loop: newLoop(opts)}, nil
}

// ── 全局工厂单槽位（对齐 setFactory：同一时刻只有一个生效工厂） ──

var (
	loopFactoryMu  sync.RWMutex
	loopFactoryVal LoopFactory = goLoopFactory{}
)

// ReplaceLoopFactory 替换全局工厂（显式覆盖当前槽位），返回还原函数。
// 宿主 Go 代码或插件桥（JS 装配器）注册替代实现时调用；还原函数恢复旧工厂，
// 供插件卸载/测试恢复使用。
func ReplaceLoopFactory(f LoopFactory) (restore func()) {
	loopFactoryMu.Lock()
	defer loopFactoryMu.Unlock()
	prev := loopFactoryVal
	loopFactoryVal = f
	return func() {
		loopFactoryMu.Lock()
		defer loopFactoryMu.Unlock()
		loopFactoryVal = prev
	}
}

// LoopFactoryNow 返回当前生效工厂。
func LoopFactoryNow() LoopFactory {
	loopFactoryMu.RLock()
	defer loopFactoryMu.RUnlock()
	return loopFactoryVal
}

// CreateLoop 走当前全局工厂创建循环句柄（会话/自闭环统一入口）。
//
//   - 默认：先按注册顺序应用全部插件装配器（applyLoopAssemblers），再走全局工厂；
//   - opts.SkipPluginAssembly=true（子 agent 回合）：跳过装配器链——子 agent 由
//     插件 JS 回调栈内的宿主能力创建（如 autopilot 决策器经 ctx.subagent.run），
//     此时调用装配器要取所属插件的 VM 锁；若该插件正是当前持锁者即**自死锁**
//     （goja VM 锁非重入）。子 agent 的装配参数已由宿主显式给出，无需插件装配。
func CreateLoop(opts LoopOpts) (LoopHandle, error) {
	if !opts.SkipPluginAssembly {
		opts = applyLoopAssemblers(opts)
	}
	return LoopFactoryNow().Create(opts)
}

// ── 插件装配器链（LoopAssembler）──────────────────────────────

// LoopAssembler 循环装配器：读插件设置 → 返回覆盖后的装配参数（Go 值语义）。
//
// ★ 2026-09-21 语义修正：装配器是**多插件叠加**的（按注册顺序依次应用），不再
// 沿用旧的「单槽位后注册覆盖」——后者会让后装载插件（如 autopilot）把先装载
// 插件（如 agentloop：systemAppend / 分段预算 / 审核模式…）的装配参数整体吃掉
// （实测真 bug：autopilot 装载后 agentloop 的装配全部失效）。
// 同键（插件名）重复注册 = 替换该项（插件重装载不叠加）。
type LoopAssembler func(LoopOpts) LoopOpts

type namedLoopAssembler struct {
	name string
	fn   LoopAssembler
}

var (
	loopAssemblersMu sync.RWMutex
	loopAssemblers   []namedLoopAssembler
)

// RegisterLoopAssembler 注册装配器（name = 插件名，同键替换），返回还原函数。
func RegisterLoopAssembler(name string, fn LoopAssembler) (restore func()) {
	if fn == nil {
		return func() {}
	}
	loopAssemblersMu.Lock()
	for i := range loopAssemblers {
		if loopAssemblers[i].name == name {
			prev := loopAssemblers[i].fn
			loopAssemblers[i].fn = fn
			loopAssemblersMu.Unlock()
			return func() {
				loopAssemblersMu.Lock()
				for j := range loopAssemblers {
					if loopAssemblers[j].name == name {
						loopAssemblers[j].fn = prev
						break
					}
				}
				loopAssemblersMu.Unlock()
			}
		}
	}
	loopAssemblers = append(loopAssemblers, namedLoopAssembler{name: name, fn: fn})
	loopAssemblersMu.Unlock()
	return func() {
		loopAssemblersMu.Lock()
		for i := range loopAssemblers {
			if loopAssemblers[i].name == name {
				loopAssemblers = append(loopAssemblers[:i], loopAssemblers[i+1:]...)
				break
			}
		}
		loopAssemblersMu.Unlock()
	}
}

// LoopAssemblerNames 当前装配器注册顺序（诊断/测试用）。
func LoopAssemblerNames() []string {
	loopAssemblersMu.RLock()
	defer loopAssemblersMu.RUnlock()
	out := make([]string, 0, len(loopAssemblers))
	for _, a := range loopAssemblers {
		out = append(out, a.name)
	}
	return out
}

// applyLoopAssemblers 按注册顺序应用全部装配器（快照遍历：应用期间的增删不生效于本次）。
func applyLoopAssemblers(opts LoopOpts) LoopOpts {
	loopAssemblersMu.RLock()
	snapshot := make([]namedLoopAssembler, len(loopAssemblers))
	copy(snapshot, loopAssemblers)
	loopAssemblersMu.RUnlock()
	for _, a := range snapshot {
		if a.fn == nil {
			continue
		}
		opts = a.fn(opts)
	}
	return opts
}

// newLoop 用 LoopOpts 构建 *Loop 内核（从 session_manager/agent 两个创建点提取的
// 公共构造逻辑；会话级回调/状态由调用方在 Create 后挂载，保持与现状一致）。
func newLoop(opts LoopOpts) *Loop {
	return &Loop{
		Provider: opts.Provider,
		Registry: opts.Registry,
		System:   opts.System,
		// ★ 2026-09-12 配置化：分段预算/续跑段数上限随装配参数透传（来源见
		//   tool_budget.go；0 = 用默认，负数预算 = 不限）。
		StepBudget:            opts.StepBudget,
		ToolCallBudget:        opts.ToolCallBudget,
		MaxToolBudgetSegments: opts.MaxToolBudgetSegments,
		MaxContextTokens:      opts.MaxContextTokens,
		Compressor:            opts.Compressor,
		Autonomous:            opts.Autonomous,
		MaxSuperviseRounds:    opts.MaxSuperviseRounds,
		maxAutonomousMinutes:  opts.MaxAutonomousMinutes,
		checkpointInterval:    opts.CheckpointInterval,
		History:               CopyHistory(opts.History),
		CompressedSummaries:   opts.CompressedSummaries,
		WorkspaceRoot:         opts.WorkspaceRoot,
		ConvID:                opts.ConvID,
		ReviewMode:            opts.ReviewMode,
		ReviewBlacklist:       opts.ReviewBlacklist,
		ReviewWhitelist:       opts.ReviewWhitelist,
		ReviewProvider:        opts.ReviewProvider,
	}
}
