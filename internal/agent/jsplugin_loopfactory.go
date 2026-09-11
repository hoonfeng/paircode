package agent

import (
	"fmt"

	"github.com/hoonfeng/paircode/goja"
)

// jsLoopFactoryBridge 把 JS 装配器（ctx.loopFactory.register 的 apply 函数）接到
// LoopFactory 接口：Create 时把 opts 的可装配快照传给 JS，合并返回的 overrides
// 后走默认 Go 工厂构建 *Loop。
//
// 对齐 setFactory 语义：外部 AgentLoop 插件在装配期决定循环参数/实现；
// gou-ide 的 Loop 是 Go 核心（无法在 goja 沙箱内构造 *Loop），故 JS 侧现实能力
// 为「参数级装配」——apply 返回非空字段即覆盖默认装配参数；返回 null/undefined
// 表示不改动。真正「换内核」留给宿主 Go 代码经 ReplaceLoopFactory 完成。
type jsLoopFactoryBridge struct {
	vm     *goja.Runtime
	apply  goja.Callable // apply(optsSnapshot) → overrides | null
	plugin *jsPluginAdapter
}

// Create 实现 LoopFactory：JS 装配 → 参数合并 → 默认工厂。
func (b *jsLoopFactoryBridge) Create(opts LoopOpts) (LoopHandle, error) {
	snap := b.buildSnapshot(opts)
	var (
		ret     goja.Value
		callErr error
	)
	// goja 非并发安全：JS apply 必须持 VM 锁执行（会话创建可能来自任意 goroutine）。
	b.plugin.withLock(func() {
		v, err := b.apply(goja.Undefined(), b.vm.ToValue(snap))
		ret, callErr = v, err
	})
	if callErr != nil {
		return nil, fmt.Errorf("loop 装配器执行失败: %w", callErr)
	}
	merged := opts
	if ret != nil && !goja.IsUndefined(ret) && !goja.IsNull(ret) {
		merged = b.applyOverrides(opts, ret.ToObject(b.vm))
	}
	return goLoopFactory{}.Create(merged)
}

// buildSnapshot 构造 JS 可见的可装配参数快照（内部字段 Provider/Registry/回调等
// 不可序列化，不暴露）。
func (b *jsLoopFactoryBridge) buildSnapshot(opts LoopOpts) map[string]any {
	return map[string]any{
		"system": opts.System,
		// ★ 2026-09-12：「最大迭代数」配置项已移除（装配器不再可覆盖迭代上限，
		//   段内迭代安全上限由宿主按段预算派生，见 tool_budget.go IterationLimit）。
		// ★ 2026-09-12 分段续跑配置化（★ 双闸门）：stepBudget / toolCallBudget / 段数上限
		//   对装配器可见（插件可覆盖，未返回则保持宿主 settings 值；语义见 tool_budget.go）。
		"stepBudget":            opts.StepBudget,
		"toolCallBudget":        opts.ToolCallBudget,
		"maxToolBudgetSegments": opts.MaxToolBudgetSegments,
		"maxContextTokens":      opts.MaxContextTokens,
		"autonomous":            opts.Autonomous,
		"maxAutonomousMinutes":  opts.MaxAutonomousMinutes,
		"checkpointInterval":    opts.CheckpointInterval,
		"workspaceRoot":         opts.WorkspaceRoot,
		"reviewMode":            opts.ReviewMode,
		"reviewBlacklist":       opts.ReviewBlacklist,
		"reviewWhitelist":       opts.ReviewWhitelist,
	}
}

// applyOverrides 从 JS 返回对象读取非空字段并覆盖到 opts。
func (b *jsLoopFactoryBridge) applyOverrides(opts LoopOpts, obj *goja.Object) LoopOpts {
	out := opts
	setStr := func(key string, dst *string) {
		if v := obj.Get(key); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			*dst = v.String()
		}
	}
	setInt := func(key string, dst *int) {
		if v := obj.Get(key); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			*dst = int(v.ToInteger())
		}
	}
	setBool := func(key string, dst *bool) {
		if v := obj.Get(key); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			*dst = v.ToBoolean()
		}
	}
	setStrs := func(key string, dst *[]string) {
		if v := obj.Get(key); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			if arr, ok := v.Export().([]any); ok {
				var out []string
				for _, it := range arr {
					if s, ok := it.(string); ok && s != "" {
						out = append(out, s)
					}
				}
				*dst = out
			}
		}
	}
	// ★ 2026-09-03 极简工具面已移除：setGroups 装配不再需要（tools 统一全量面）。
	setStr("system", &out.System)
	setInt("stepBudget", &out.StepBudget)
	setInt("toolCallBudget", &out.ToolCallBudget)
	setInt("maxToolBudgetSegments", &out.MaxToolBudgetSegments)
	setInt("maxContextTokens", &out.MaxContextTokens)
	setBool("autonomous", &out.Autonomous)
	setInt("maxAutonomousMinutes", &out.MaxAutonomousMinutes)
	setInt("checkpointInterval", &out.CheckpointInterval)
	setStr("workspaceRoot", &out.WorkspaceRoot)
	setStr("reviewMode", &out.ReviewMode)
	setStrs("reviewBlacklist", &out.ReviewBlacklist)
	setStrs("reviewWhitelist", &out.ReviewWhitelist)
	// ★ 2026-09-03 极简工具面已移除：stagedToolGroups 装配不再透传（tools 统一全量面，
	//   跨轮次前缀稳定——极简/全量切换会让 DeepSeek 缓存从头断前缀）。
	return out
}
