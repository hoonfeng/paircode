package agent

// ═══════════════════════════════════════════════════════════════
// tool_budget_test.go — 段预算（★ 双闸门：步数 + 工具调用）分段执行 + 自动续跑 单测
//
// 覆盖：
//   · 默认预算 / 计数 / 耗尽判定 / 不限与自定义预算（工具闸 + 步数闸各自归一化）；
//   · 双闸门判定（任一达上限即耗尽；同时命中报 step_budget；单一闸门不限时不耗尽）；
//   · 续跑标记（markSegmentPending → TakeSegmentContinue 消费一次）；
//   · 续跑消息文案；
//   · JS 循环端到端：工具闸 / 步数闸耗尽 → agentloop 返回 segment → 宿主标记续跑
//     （done 事件 doneReason=tool_budget；LLM 调用次数 = 预算段数）。
// ═══════════════════════════════════════════════════════════════

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestToolBudget_DefaultAndCounting 默认预算与计数/耗尽判定。
func TestToolBudget_DefaultAndCounting(t *testing.T) {
	l := &Loop{}
	if got := l.ToolCallBudgetOrDefault(); got != DefaultToolCallBudget {
		t.Fatalf("默认预算应为 %d，得 %d", DefaultToolCallBudget, got)
	}
	used, budget, exhausted := l.ToolBudgetState()
	if used != 0 || budget != DefaultToolCallBudget || exhausted {
		t.Fatalf("初始状态异常: used=%d budget=%d exhausted=%v", used, budget, exhausted)
	}
	for i := 0; i < DefaultToolCallBudget-1; i++ {
		l.noteToolCall()
	}
	if _, _, exhausted = l.ToolBudgetState(); exhausted {
		t.Fatal("未达预算不应耗尽")
	}
	l.noteToolCall()
	if _, _, exhausted = l.ToolBudgetState(); !exhausted {
		t.Fatal("达到预算应耗尽")
	}
}

// TestToolBudget_UnlimitedAndCustom 负数=不限；自定义预算生效。
func TestToolBudget_UnlimitedAndCustom(t *testing.T) {
	l := &Loop{ToolCallBudget: -1}
	if got := l.ToolCallBudgetOrDefault(); got != 0 {
		t.Fatalf("负数=不限应返回 0，得 %d", got)
	}
	for i := 0; i < 1000; i++ {
		l.noteToolCall()
	}
	if _, _, exhausted := l.ToolBudgetState(); exhausted {
		t.Fatal("不限模式不应耗尽")
	}
	l2 := &Loop{ToolCallBudget: 5}
	for i := 0; i < 5; i++ {
		l2.noteToolCall()
	}
	if _, budget, exhausted := l2.ToolBudgetState(); !exhausted || budget != 5 {
		t.Fatalf("自定义预算 5 应耗尽: budget=%d exhausted=%v", budget, exhausted)
	}
}

// TestToolBudget_SegmentContinue 续跑标记：一次消费 + Run 重置。
func TestToolBudget_SegmentContinue(t *testing.T) {
	l := &Loop{ToolCallBudget: 3}
	l.noteToolCall()
	l.noteToolCall()
	l.noteToolCall()
	if _, ok := l.TakeSegmentContinue(); ok {
		t.Fatal("未标记时不应续跑")
	}
	l.markSegmentPending()
	st, ok := l.TakeSegmentContinue()
	if !ok || st.UsedTools != 3 || st.ToolBudget != 3 {
		t.Fatalf("续跑标记消费异常: ok=%v state=%+v", ok, st)
	}
	if _, ok := l.TakeSegmentContinue(); ok {
		t.Fatal("标记应只消费一次")
	}
	l.resetToolCallsRun()
	if used, _, _ := l.ToolBudgetState(); used != 0 {
		t.Fatalf("重置后计数应为 0，得 %d", used)
	}
}

// TestToolBudget_ContinueMessage 续跑消息包含预算信息与「继续」指引。
func TestToolBudget_ContinueMessage(t *testing.T) {
	msg := SegmentContinueMessage(SegmentState{
		Reason: SegmentReasonToolBudget, UsedTools: 120, ToolBudget: 120, UsedSteps: 98, StepBudget: 120,
	})
	if !strings.Contains(msg, "120") || !strings.Contains(msg, "98") || !strings.Contains(msg, "继续") {
		t.Fatalf("续跑消息缺少关键信息: %q", msg)
	}
	// 步数闸：文案应指明「步数预算用尽」并给出步数口径
	msg2 := SegmentContinueMessage(SegmentState{
		Reason: SegmentReasonStepBudget, UsedSteps: 120, StepBudget: 120, UsedTools: 137, ToolBudget: 200,
	})
	if !strings.Contains(msg2, "步数预算用尽") || !strings.Contains(msg2, "120 步") {
		t.Fatalf("步数闸续跑消息应指明步数预算用尽与步数口径: %q", msg2)
	}
}

// TestToolBudget_NormalizeConfiguredValues 配置化（2026-09-12）：单段预算归一化——
// 未配置→默认、自定义生效、负数=不限、越界钳制；并核对 Loop 消费侧一致。
// 配置真源：agentloop 插件注册（pluginSettings.agentloop）经装配器透传。
func TestToolBudget_NormalizeConfiguredValues(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"未配置(0)→默认", 0, DefaultToolCallBudget},
		{"显式等于默认", DefaultToolCallBudget, DefaultToolCallBudget},
		{"自定义 5", 5, 5},
		{"负数=不限(保持负号)", -1, -1},
		{"上限内最大值", MaxToolCallBudgetLimit, MaxToolCallBudgetLimit},
		{"越界→钳制上限", MaxToolCallBudgetLimit + 1, MaxToolCallBudgetLimit},
		{"超大值→钳制上限", 1 << 30, MaxToolCallBudgetLimit},
	}
	for _, c := range cases {
		if got := NormalizeToolCallBudget(c.in); got != c.want {
			t.Errorf("%s: NormalizeToolCallBudget(%d) = %d，期望 %d", c.name, c.in, got, c.want)
		}
	}
	// Loop 消费侧：0=默认；负数=不限（内部以 0 表示）；越界钳制
	if got := (&Loop{}).ToolCallBudgetOrDefault(); got != DefaultToolCallBudget {
		t.Errorf("Loop{} 生效预算 = %d，期望默认 %d", got, DefaultToolCallBudget)
	}
	if got := (&Loop{ToolCallBudget: -3}).ToolCallBudgetOrDefault(); got != 0 {
		t.Errorf("负数预算应表示「不限」(内部 0)，得 %d", got)
	}
	if got := (&Loop{ToolCallBudget: 7}).ToolCallBudgetOrDefault(); got != 7 {
		t.Errorf("自定义预算应生效，得 %d", got)
	}
	if got := (&Loop{ToolCallBudget: MaxToolCallBudgetLimit + 100}).ToolCallBudgetOrDefault(); got != MaxToolCallBudgetLimit {
		t.Errorf("越界预算应钳制到 %d，得 %d", MaxToolCallBudgetLimit, got)
	}
}

// TestLoop_IterationLimit 段内迭代安全上限（2026-09-12，★ 双闸门）：原「最大迭代数」
// 配置项已移除——上限改由 max(生效步数预算, 生效工具调用预算) + LoopIterationSlack
// 派生（双闸门均不限时以预算钳制上限兜底），始终为有限值（防失控），无配置入口。
func TestLoop_IterationLimit(t *testing.T) {
	cases := []struct {
		name  string
		tools int
		steps int
		want  int
	}{
		{"均未配置→默认 120 + 余量", 0, 0, DefaultToolCallBudget + LoopIterationSlack},
		{"仅工具预算 5→步数默认 120 更大", 5, 0, DefaultStepBudget + LoopIterationSlack},
		{"仅步数预算 200→步数胜出", 0, 200, 200 + LoopIterationSlack},
		{"工具更大", 400, 300, 400 + LoopIterationSlack},
		{"步数更大", 300, 400, 400 + LoopIterationSlack},
		{"二者均不限→预算钳制上限兜底", -1, -1, MaxToolCallBudgetLimit + LoopIterationSlack},
		{"越界钳制后取较大者", MaxToolCallBudgetLimit + 1000, MaxStepBudgetLimit + 7, MaxToolCallBudgetLimit + LoopIterationSlack},
	}
	for _, c := range cases {
		if got := (&Loop{ToolCallBudget: c.tools, StepBudget: c.steps}).IterationLimit(); got != c.want {
			t.Errorf("%s: IterationLimit() = %d，期望 %d", c.name, got, c.want)
		}
	}
	// 不变式：上限严格大于两侧生效预算——保证正常流程由预算先触发分段续跑，
	// 迭代上限只在预算与轮次严重不同步（空转轮）时兜底。
	for _, l := range []*Loop{
		{}, {StepBudget: 500}, {ToolCallBudget: 500, StepBudget: 30}, {ToolCallBudget: -1, StepBudget: 700},
	} {
		maxB := l.ToolCallBudgetOrDefault()
		if sb := l.StepBudgetOrDefault(); sb > maxB {
			maxB = sb
		}
		if got := l.IterationLimit(); got <= maxB {
			t.Errorf("迭代安全上限应大于生效预算，得 %d（预算 %d）", got, maxB)
		}
	}
}

// TestToolBudget_SegmentsNormalize 配置化（2026-09-12）：续跑段数上限归一化——
// 未配置/负数→默认（不允许「不限」，防失控）、自定义生效、越界钳制。
func TestToolBudget_SegmentsNormalize(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"未配置(0)→默认", 0, MaxToolBudgetSegments},
		{"负数→默认", -5, MaxToolBudgetSegments},
		{"自定义 1", 1, 1},
		{"自定义 50", 50, 50},
		{"越界→钳制上限", MaxToolBudgetSegmentsLimit + 1, MaxToolBudgetSegmentsLimit},
	}
	for _, c := range cases {
		if got := NormalizeMaxToolBudgetSegments(c.in); got != c.want {
			t.Errorf("%s: NormalizeMaxToolBudgetSegments(%d) = %d，期望 %d", c.name, c.in, got, c.want)
		}
	}
	if got := (&Loop{}).MaxToolBudgetSegmentsOrDefault(); got != MaxToolBudgetSegments {
		t.Errorf("Loop{} 生效段数上限 = %d，期望默认 %d", got, MaxToolBudgetSegments)
	}
	if got := (&Loop{MaxToolBudgetSegments: 3}).MaxToolBudgetSegmentsOrDefault(); got != 3 {
		t.Errorf("自定义段数上限应生效，得 %d", got)
	}
}

// TestStepBudget_NormalizeConfiguredValues 步数预算归一化（2026-09-12 双闸门）：
// 未配置→默认、自定义生效、负数=不限、越界钳制；并核对 Loop 消费侧一致。
// 配置真源：agentloop 插件注册的 stepBudget（pluginSettings.agentloop）经装配器透传。
func TestStepBudget_NormalizeConfiguredValues(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"未配置(0)→默认", 0, DefaultStepBudget},
		{"显式等于默认", DefaultStepBudget, DefaultStepBudget},
		{"自定义 5", 5, 5},
		{"负数=不限(保持负号)", -1, -1},
		{"上限内最大值", MaxStepBudgetLimit, MaxStepBudgetLimit},
		{"越界→钳制上限", MaxStepBudgetLimit + 1, MaxStepBudgetLimit},
		{"超大值→钳制上限", 1 << 30, MaxStepBudgetLimit},
	}
	for _, c := range cases {
		if got := NormalizeStepBudget(c.in); got != c.want {
			t.Errorf("%s: NormalizeStepBudget(%d) = %d，期望 %d", c.name, c.in, got, c.want)
		}
	}
	if got := (&Loop{}).StepBudgetOrDefault(); got != DefaultStepBudget {
		t.Errorf("Loop{} 生效步数预算 = %d，期望默认 %d", got, DefaultStepBudget)
	}
	if got := (&Loop{StepBudget: -3}).StepBudgetOrDefault(); got != 0 {
		t.Errorf("负数步数预算应表示「不限」(内部 0)，得 %d", got)
	}
	if got := (&Loop{StepBudget: 7}).StepBudgetOrDefault(); got != 7 {
		t.Errorf("自定义步数预算应生效，得 %d", got)
	}
	if got := (&Loop{StepBudget: MaxStepBudgetLimit + 100}).StepBudgetOrDefault(); got != MaxStepBudgetLimit {
		t.Errorf("越界步数预算应钳制到 %d，得 %d", MaxStepBudgetLimit, got)
	}
}

// TestStepBudget_SegmentGate 双闸门判定（2026-09-12）：
//
//	· 步数达上限（工具调用远未达）→ 耗尽，Reason=step_budget；
//	· 两者同时达上限 → 报 step_budget（步数优先）；
//	· 步数不限 → 仅工具闸生效，Reason=tool_budget。
func TestStepBudget_SegmentGate(t *testing.T) {
	// ① 仅步数达上限
	l := &Loop{StepBudget: 3, ToolCallBudget: 1000}
	l.StepNo = 2
	if st := l.SegmentBudgetState(); st.Exhausted {
		t.Fatalf("步数 2 < 3 不应耗尽: %+v", st)
	}
	l.StepNo = 3
	st := l.SegmentBudgetState()
	if !st.Exhausted || st.Reason != SegmentReasonStepBudget {
		t.Fatalf("步数达上限应报 step_budget: %+v", st)
	}
	if st.UsedSteps != 3 || st.StepBudget != 3 || st.UsedTools != 0 || st.ToolBudget != 1000 {
		t.Fatalf("状态字段异常: %+v", st)
	}
	if _, _, toolExhausted := l.ToolBudgetState(); toolExhausted {
		t.Error("工具闸门视图不应因步数闸门而报耗尽（保持旧语义）")
	}

	// ② 两者同时达上限 → 步数优先
	l2 := &Loop{StepBudget: 2, ToolCallBudget: 2}
	l2.StepNo = 5
	l2.noteToolCall()
	l2.noteToolCall()
	if st := l2.SegmentBudgetState(); !st.Exhausted || st.Reason != SegmentReasonStepBudget {
		t.Fatalf("同时命中应报 step_budget: %+v", st)
	}

	// ③ 步数不限 → 仅工具闸
	l3 := &Loop{StepBudget: -1, ToolCallBudget: 2}
	l3.StepNo = 999
	if st := l3.SegmentBudgetState(); st.Exhausted {
		t.Fatalf("步数不限时不应因步数耗尽: %+v", st)
	}
	l3.noteToolCall()
	l3.noteToolCall()
	if st := l3.SegmentBudgetState(); !st.Exhausted || st.Reason != SegmentReasonToolBudget {
		t.Fatalf("工具达上限应报 tool_budget: %+v", st)
	}

	// ④ 工具不限 → 仅步数闸
	l4 := &Loop{ToolCallBudget: -1, StepBudget: 4}
	l4.StepNo = 4
	for i := 0; i < 500; i++ {
		l4.noteToolCall()
	}
	if st := l4.SegmentBudgetState(); !st.Exhausted || st.Reason != SegmentReasonStepBudget {
		t.Fatalf("工具不限时步数达上限应报 step_budget: %+v", st)
	}
}

// TestJSLoopToolBudgetSegment JS 循环端到端：**工具闸**耗尽 → segment → 宿主续跑标记
// （步数预算放宽到 1000，确保命中的是工具调用闸）。
func TestJSLoopToolBudgetSegment(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("BUDGET_WORLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	// 模型每轮都调 read（无限工作）——预算 2 应恰好执行 2 次后分段结束
	responses := make([]Message, 0, 6)
	for i := 0; i < 6; i++ {
		responses = append(responses, Message{ToolCalls: []ToolCall{{
			ID: fmt.Sprintf("c%d", i), Type: "function",
			Function: FunctionCall{Name: "read", Arguments: `{"path":"hello.txt"}`},
		}}})
	}
	mock := &MockProvider{Responses: responses}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "test-budget",
		ToolCallBudget: 2, StepBudget: 1000, OnEvent: func(e Event) { events = append(events, e) }}

	if _, err := loop.Run(context.Background(), "读文件直到我说停", nil); err != nil {
		t.Fatalf("Run(JS 循环): %v", err)
	}
	st, ok := loop.TakeSegmentContinue()
	if !ok || st.UsedTools != 2 || st.ToolBudget != 2 || st.Reason != SegmentReasonToolBudget {
		t.Fatalf("工具闸耗尽应标记续跑: ok=%v state=%+v", ok, st)
	}
	sawSegmentDone := false
	for _, e := range events {
		if e.Type == EventDone && e.DoneReason == "tool_budget" {
			sawSegmentDone = true
		}
	}
	if !sawSegmentDone {
		t.Errorf("应发出 doneReason=tool_budget 的 done 事件，事件=%v", eventTypes(events))
	}
	if mock.Calls() != 2 {
		t.Errorf("预算 2 应恰好 2 次 LLM 调用，得 %d", mock.Calls())
	}
}

// TestJSLoopStepBudgetSegment JS 循环端到端：**步数闸**耗尽 → segment → 宿主续跑标记。
// 模型每轮执行 1 次工具调用、工具预算放宽到 1000，步数预算 2 → 恰好 2 步（2 次 LLM
// 调用）后分段；证明段边界可由「步数」而非「工具调用次数」决定（2026-09-12 双闸门）。
func TestJSLoopStepBudgetSegment(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("STEP_WORLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	responses := make([]Message, 0, 6)
	for i := 0; i < 6; i++ {
		responses = append(responses, Message{ToolCalls: []ToolCall{{
			ID: fmt.Sprintf("s%d", i), Type: "function",
			Function: FunctionCall{Name: "read", Arguments: `{"path":"hello.txt"}`},
		}}})
	}
	mock := &MockProvider{Responses: responses}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "test-step-budget",
		StepBudget: 2, ToolCallBudget: 1000, OnEvent: func(e Event) { events = append(events, e) }}

	if _, err := loop.Run(context.Background(), "读文件直到我说停", nil); err != nil {
		t.Fatalf("Run(JS 循环): %v", err)
	}
	st, ok := loop.TakeSegmentContinue()
	if !ok || st.UsedSteps != 2 || st.StepBudget != 2 || st.Reason != SegmentReasonStepBudget {
		t.Fatalf("步数闸耗尽应标记续跑: ok=%v state=%+v", ok, st)
	}
	if st.UsedTools != 2 {
		t.Errorf("2 步应各执行 1 次工具调用，得 %d", st.UsedTools)
	}
	if mock.Calls() != 2 {
		t.Errorf("步数预算 2 应恰好 2 次 LLM 调用，得 %d", mock.Calls())
	}
	sawDone, sawStepNotice := false, false
	for _, e := range events {
		if e.Type == EventDone && e.DoneReason == "tool_budget" {
			sawDone = true
		}
		if e.Type == EventNotice && strings.Contains(e.Content, "步数预算上限") {
			sawStepNotice = true
		}
	}
	if !sawDone {
		t.Errorf("应发出 doneReason=tool_budget 的 done 事件，事件=%v", eventTypes(events))
	}
	if !sawStepNotice {
		t.Errorf("应发出「步数预算上限」分段通知（含步数口径），事件=%v", eventTypes(events))
	}
}

// eventTypes 事件类型摘要（断言失败时打印用）。
func eventTypes(events []Event) string {
	parts := make([]string, 0, len(events))
	for _, e := range events {
		parts = append(parts, string(e.Type)+"/"+e.DoneReason)
	}
	return strings.Join(parts, ", ")
}
