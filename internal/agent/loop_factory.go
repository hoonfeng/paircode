package agent

import (
	"context"
	"sync"
)

// LoopFactory 装配 agent 循环的工厂接口（对齐 deepseek-harness AgentFactory 单槽位语义）。
//
// 参考项目（ref/deepseek-harness/packages/core/agent-loop）：AgentLoop 是一个 cordis
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
func CreateLoop(opts LoopOpts) (LoopHandle, error) {
	return LoopFactoryNow().Create(opts)
}

// newLoop 用 LoopOpts 构建 *Loop 内核（从 session_manager/agent 两个创建点提取的
// 公共构造逻辑；会话级回调/状态由调用方在 Create 后挂载，保持与现状一致）。
func newLoop(opts LoopOpts) *Loop {
	return &Loop{
		Provider:             opts.Provider,
		Registry:             opts.Registry,
		System:               opts.System,
		MaxIterations:        opts.MaxIterations,
		MaxContextTokens:     opts.MaxContextTokens,
		Compressor:           opts.Compressor,
		Autonomous:           opts.Autonomous,
		maxAutonomousMinutes: opts.MaxAutonomousMinutes,
		checkpointInterval:   opts.CheckpointInterval,
		History:              CopyHistory(opts.History),
		CompressedSummaries:  opts.CompressedSummaries,
		WorkspaceRoot:        opts.WorkspaceRoot,
		ReviewMode:           opts.ReviewMode,
		ReviewBlacklist:      opts.ReviewBlacklist,
		ReviewWhitelist:      opts.ReviewWhitelist,
		ReviewProvider:       opts.ReviewProvider,
		// ★ 2026-08-27 首步极简工具面默认开启（实测改进，tools_staging.go）
		StagedTools:       true,
		StagedToolGroups:  opts.StagedToolGroups,
	}
}
