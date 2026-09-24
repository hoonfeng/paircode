package agent

// ═══════════════════════════════════════════════════════════════
// tool_budget_config_test.go — 分段续跑配置化端到端（2026-09-12；★ 双闸门）
//
// 覆盖完整配置链路（配置项由 agentloop 插件注册，宿主不内置字段）：
//
//	settings.json 的 pluginSettings.agentloop（插件 registerSettings 注册的
//	stepBudget / toolCallBudget / maxToolBudgetSegments）
//	  → 插件装配器 ctx.loopFactory.register 内 ctx.getSettings('agentloop')
//	  → overrides 对象 → jsLoopFactoryBridge.applyOverrides
//	  → LoopOpts → Loop 生效值（StepBudgetOrDefault / ToolCallBudgetOrDefault /
//	    MaxToolBudgetSegmentsOrDefault）
//
// 断言：自定义生效、0=不覆盖（回退默认）、负数预算=不限、负数段数=默认、
// 越界钳制、非法值（非数字）不覆盖。
// ═══════════════════════════════════════════════════════════════

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hoonfeng/paircode/internal/core"
)

// withAgentloopSettings 临时写入 pluginSettings.agentloop（测试结束自动还原）。
func withAgentloopSettings(t *testing.T, vals map[string]any) {
	t.Helper()
	if core.Settings.PluginSettings == nil {
		core.Settings.PluginSettings = map[string]map[string]any{}
	}
	prev, hadPrev := core.Settings.PluginSettings["agentloop"]
	t.Cleanup(func() {
		if hadPrev {
			core.Settings.PluginSettings["agentloop"] = prev
		} else {
			delete(core.Settings.PluginSettings, "agentloop")
		}
	})
	core.Settings.PluginSettings["agentloop"] = vals
}

// mustCreate 创建 Loop 句柄（失败即终止）。
func mustCreate(t *testing.T) *Loop {
	t.Helper()
	h, err := CreateLoop(LoopOpts{})
	if err != nil {
		t.Fatalf("CreateLoop: %v", err)
	}
	return h.Loop()
}

// TestToolBudgetConfig_FromPluginSettings 配置 → 生效链路的端到端断言。
func TestToolBudgetConfig_FromPluginSettings(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	if _, ok := LoopFactoryNow().(goLoopFactory); !ok {
		t.Skipf("当前 LoopFactory 非默认（%T），跳过避免污染", LoopFactoryNow())
	}
	coreSettingsEnsure()
	loadRealAgentloop(t) // 装载真实插件（含 ctx.loopFactory.register 装配器）
	// ★ 2026-09-21 装配器链：注册进入装配器链（不再替换全局 LoopFactory 单槽位）。
	if names := LoopAssemblerNames(); len(names) == 0 {
		t.Fatalf("装载 agentloop 后装配器链为空，期望含 'agentloop'，实际=%v", names)
	}

	// ① 显式配置（JSON 反序列化后的数值类型 float64）→ 生效
	withAgentloopSettings(t, map[string]any{
		"stepBudget":            float64(5),
		"toolCallBudget":        float64(3),
		"maxToolBudgetSegments": float64(2),
	})
	l := mustCreate(t)
	if got := l.StepBudgetOrDefault(); got != 5 {
		t.Errorf("配置 stepBudget=5 应生效，得 %d", got)
	}
	if got := l.ToolCallBudgetOrDefault(); got != 3 {
		t.Errorf("配置 toolCallBudget=3 应生效，得 %d", got)
	}
	if got := l.MaxToolBudgetSegmentsOrDefault(); got != 2 {
		t.Errorf("配置 maxToolBudgetSegments=2 应生效，得 %d", got)
	}

	// ② 0 = 不覆盖 → 宿主默认（120 步 / 120 次 / 20 段）
	withAgentloopSettings(t, map[string]any{
		"stepBudget":            float64(0),
		"toolCallBudget":        float64(0),
		"maxToolBudgetSegments": float64(0),
	})
	l2 := mustCreate(t)
	if got := l2.StepBudgetOrDefault(); got != DefaultStepBudget {
		t.Errorf("配置 0 应回退默认步数预算 %d，得 %d", DefaultStepBudget, got)
	}
	if got := l2.ToolCallBudgetOrDefault(); got != DefaultToolCallBudget {
		t.Errorf("配置 0 应回退默认 %d，得 %d", DefaultToolCallBudget, got)
	}
	if got := l2.MaxToolBudgetSegmentsOrDefault(); got != MaxToolBudgetSegments {
		t.Errorf("配置 0 应回退默认 %d，得 %d", MaxToolBudgetSegments, got)
	}

	// ③ 负数预算 = 不限（内部 0，两个预算各自生效）；负数段数 = 默认（不允许「不限」）
	withAgentloopSettings(t, map[string]any{
		"stepBudget":            float64(-1),
		"toolCallBudget":        float64(-1),
		"maxToolBudgetSegments": float64(-3),
	})
	l3 := mustCreate(t)
	if got := l3.StepBudgetOrDefault(); got != 0 {
		t.Errorf("负数步数预算应表示「不限」(内部 0)，得 %d", got)
	}
	if got := l3.ToolCallBudgetOrDefault(); got != 0 {
		t.Errorf("负数预算应表示「不限」(内部 0)，得 %d", got)
	}
	if _, _, exhausted := l3.ToolBudgetState(); exhausted {
		t.Error("不限模式不应耗尽")
	}
	if st := l3.SegmentBudgetState(); st.Exhausted {
		t.Errorf("两侧预算均不限时不应耗尽: %+v", st)
	}
	if got := l3.MaxToolBudgetSegmentsOrDefault(); got != MaxToolBudgetSegments {
		t.Errorf("负数段数应回退默认 %d，得 %d", MaxToolBudgetSegments, got)
	}

	// ④ 越界值 → 钳制到上限
	withAgentloopSettings(t, map[string]any{
		"stepBudget":            float64(999999),
		"toolCallBudget":        float64(999999),
		"maxToolBudgetSegments": float64(99999),
	})
	l4 := mustCreate(t)
	if got := l4.StepBudgetOrDefault(); got != MaxStepBudgetLimit {
		t.Errorf("越界步数预算应钳制到 %d，得 %d", MaxStepBudgetLimit, got)
	}
	if got := l4.ToolCallBudgetOrDefault(); got != MaxToolCallBudgetLimit {
		t.Errorf("越界预算应钳制到 %d，得 %d", MaxToolCallBudgetLimit, got)
	}
	if got := l4.MaxToolBudgetSegmentsOrDefault(); got != MaxToolBudgetSegmentsLimit {
		t.Errorf("越界段数应钳制到 %d，得 %d", MaxToolBudgetSegmentsLimit, got)
	}

	// ⑤ 非法值（非数字）→ 不覆盖 → 默认
	withAgentloopSettings(t, map[string]any{
		"stepBudget":            "abc",
		"toolCallBudget":        "abc",
		"maxToolBudgetSegments": "abc",
	})
	l5 := mustCreate(t)
	if got := l5.StepBudgetOrDefault(); got != DefaultStepBudget {
		t.Errorf("非法步数预算应回退默认 %d，得 %d", DefaultStepBudget, got)
	}
	if got := l5.ToolCallBudgetOrDefault(); got != DefaultToolCallBudget {
		t.Errorf("非法预算应回退默认 %d，得 %d", DefaultToolCallBudget, got)
	}
	if got := l5.MaxToolBudgetSegmentsOrDefault(); got != MaxToolBudgetSegments {
		t.Errorf("非法段数应回退默认 %d，得 %d", MaxToolBudgetSegments, got)
	}
}

// TestToolBudgetConfig_BehaviorEndToEnd 配置 → 行为闭环：pluginSettings 配
// toolCallBudget=2 → 真实 agentloop 插件装配 → Run 恰好 2 次工具调用后分段结束
// （而非默认 120 次），证明配置不是摆设、真源由配置驱动。
func TestToolBudgetConfig_BehaviorEndToEnd(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	if _, ok := LoopFactoryNow().(goLoopFactory); !ok {
		t.Skipf("当前 LoopFactory 非默认（%T），跳过避免污染", LoopFactoryNow())
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	withAgentloopSettings(t, map[string]any{
		"stepBudget":            float64(1000),
		"toolCallBudget":        float64(2),
		"maxToolBudgetSegments": float64(1),
	})

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("CFG_WORLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	// 模型每轮都调 read（无限工作）——配置预算 2 应恰好执行 2 次后分段结束
	responses := make([]Message, 0, 6)
	for i := 0; i < 6; i++ {
		responses = append(responses, Message{ToolCalls: []ToolCall{{
			ID: fmt.Sprintf("cfg%d", i), Type: "function",
			Function: FunctionCall{Name: "read", Arguments: `{"path":"hello.txt"}`},
		}}})
	}
	mock := &MockProvider{Responses: responses}
	h, err := CreateLoop(LoopOpts{
		Provider: mock, Registry: reg, System: "cfg-budget",
		WorkspaceRoot: dir,
	})
	if err != nil {
		t.Fatalf("CreateLoop: %v", err)
	}
	loop := h.Loop()
	if got := loop.ToolCallBudgetOrDefault(); got != 2 {
		t.Fatalf("装配后生效预算 = %d，期望配置值 2", got)
	}
	if _, err := loop.Run(context.Background(), "读文件直到我说停", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, ok := loop.TakeSegmentContinue()
	if !ok || st.UsedTools != 2 || st.ToolBudget != 2 || st.Reason != SegmentReasonToolBudget {
		t.Errorf("配置工具预算 2 应恰好 2 次后标记续跑（工具闸）：ok=%v state=%+v", ok, st)
	}
	if mock.Calls() != 2 {
		t.Errorf("配置预算 2 应 2 次 LLM 调用后分段，得 %d", mock.Calls())
	}
	if got := loop.MaxToolBudgetSegmentsOrDefault(); got != 1 {
		t.Errorf("装配后生效段数上限 = %d，期望配置值 1", got)
	}
}

// TestStepBudgetConfig_BehaviorEndToEnd 配置 → 行为闭环（★ 双闸门·步数闸）：
// pluginSettings 配 stepBudget=2（工具预算放宽到 1000）→ 真实 agentloop 插件装配 →
// Run 恰好 2 步（2 次 LLM 调用）后分段结束，且 reason=step_budget——证明段边界可由
// 「步数」配置决定，而不是只看工具调用次数。
func TestStepBudgetConfig_BehaviorEndToEnd(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	if _, ok := LoopFactoryNow().(goLoopFactory); !ok {
		t.Skipf("当前 LoopFactory 非默认（%T），跳过避免污染", LoopFactoryNow())
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	withAgentloopSettings(t, map[string]any{
		"stepBudget":            float64(2),
		"toolCallBudget":        float64(1000),
		"maxToolBudgetSegments": float64(1),
	})

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("CFG_STEP"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	responses := make([]Message, 0, 6)
	for i := 0; i < 6; i++ {
		responses = append(responses, Message{ToolCalls: []ToolCall{{
			ID: fmt.Sprintf("st%d", i), Type: "function",
			Function: FunctionCall{Name: "read", Arguments: `{"path":"hello.txt"}`},
		}}})
	}
	mock := &MockProvider{Responses: responses}
	h, err := CreateLoop(LoopOpts{Provider: mock, Registry: reg, System: "cfg-step", WorkspaceRoot: dir})
	if err != nil {
		t.Fatalf("CreateLoop: %v", err)
	}
	loop := h.Loop()
	if got := loop.StepBudgetOrDefault(); got != 2 {
		t.Fatalf("装配后生效步数预算 = %d，期望配置值 2", got)
	}
	if _, err := loop.Run(context.Background(), "读文件直到我说停", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, ok := loop.TakeSegmentContinue()
	if !ok || st.UsedSteps != 2 || st.StepBudget != 2 || st.Reason != SegmentReasonStepBudget {
		t.Errorf("配置步数预算 2 应恰好 2 步后标记续跑（步数闸）：ok=%v state=%+v", ok, st)
	}
	if mock.Calls() != 2 {
		t.Errorf("配置步数预算 2 应 2 次 LLM 调用后分段，得 %d", mock.Calls())
	}
}
