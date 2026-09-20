package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ── 子 agent 回合能力（ctx.subagent 的 Go 侧实现）──────────────────

// TestRunSubagent_SubmitAndTrace 子 agent 回合：工具调用 + 结构化提交收尾，
// 事件带来源标注（AgentName）+ 回合号（turn），轨迹/提交参数回传插件。
func TestRunSubagent_SubmitAndTrace(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{
		Name: "echo", Description: "回显", Parameters: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (string, error) { return "echo-ok", nil },
	})
	// 脚本：① 调 echo ② 调提交工具收尾
	provider := &MockProvider{Responses: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "echo", Arguments: "{}"}}}},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c2", Type: "function", Function: FunctionCall{
			Name: "submit_result", Arguments: `{"action":"continue","assessment":"还需验证","next_task":"跑 go test"}`}}}},
	}}
	events := make(chan Event, 256)
	rt := &SubagentRuntime{
		ConvID: "conv-sub", Provider: provider, Registry: reg, System: "sys",
		Events: events, StepBudget: 10, ToolCallBudget: 10, ParentCtx: context.Background(),
	}
	unreg := RegisterSubagentRuntime(rt)
	defer unreg()
	// 模拟宿主调用决策器时的绑定（SessionManager 在 RunAutopilotDecision 前设置）
	restoreCur := SetCurrentSubagentRuntime(rt)
	defer restoreCur()

	res, err := RunSubagent(nil, SubagentSpec{
		Name: "supervisor", Round: 3, Task: "任务书", StepBudget: 10, ToolCallBudget: 10,
		ResultTool: &SubagentResultTool{Name: "submit_result", Description: "提交裁决"},
	})
	if err != nil {
		t.Fatalf("RunSubagent 失败: %v", err)
	}
	if res.Ended != "submitted" {
		t.Errorf("Ended = %q, want submitted（结构化提交路径）", res.Ended)
	}
	if res.Submitted == nil || res.Submitted["action"] != "continue" {
		t.Errorf("Submitted = %v, want action=continue", res.Submitted)
	}
	if got := res.Submitted["next_task"]; got != "跑 go test" {
		t.Errorf("Submitted[next_task] = %v", got)
	}
	if res.ToolCalls != 2 {
		t.Errorf("ToolCalls = %d, want 2（echo + submit_result）", res.ToolCalls)
	}
	if res.Steps == 0 {
		t.Errorf("Steps = 0, want > 0")
	}
	// 轨迹里应能看到 echo 调用与结果
	var sawEcho bool
	for _, s := range res.Segments {
		if s.Type == "tool_call" && s.Name == "echo" {
			sawEcho = true
			if s.Result != "echo-ok" {
				t.Errorf("echo 段结果 = %q, want echo-ok", s.Result)
			}
		}
	}
	if !sawEcho {
		t.Errorf("轨迹中缺少 echo 工具调用段：%+v", res.Segments)
	}

	// 事件来源标注：AgentName=supervisor，Turn=Round（前端按 (来源, 回合) 分区渲染看板）
	close(events)
	var sawToolCall, sawAgentName bool
	for e := range events {
		if e.AgentName != "supervisor" {
			t.Errorf("事件来源标注丢失：type=%s agentName=%q", e.Type, e.AgentName)
		}
		if e.Turn != 3 {
			t.Errorf("事件 turn = %d, want 3（= 回合号）", e.Turn)
		}
		if e.Type == EventToolCall && e.Tool == "echo" {
			sawToolCall = true
			sawAgentName = true
		}
	}
	if !sawToolCall || !sawAgentName {
		t.Errorf("未收到 echo 的 tool_call 事件（sawToolCall=%v）", sawToolCall)
	}
}

// TestRunSubagent_NoRuntime 无会话运行环境时明确报错（而非静默成功）。
func TestRunSubagent_NoRuntime(t *testing.T) {
	_, err := RunSubagent(nil, SubagentSpec{Name: "supervisor", Task: "x", ConvID: "nonexistent-conv"})
	if err == nil || !strings.Contains(err.Error(), "不可用") {
		t.Fatalf("err = %v, want 能力不可用", err)
	}
}

// TestRunSubagent_ToolWhitelist 工具白名单裁剪：名单外工具不可见/不可调用。
func TestRunSubagent_ToolWhitelist(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{Name: "read", Parameters: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (string, error) { return "read-ok", nil }})
	reg.Register(&Tool{Name: "write", Parameters: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (string, error) { return "write-ok", nil }})
	provider := &MockProvider{Responses: []Message{
		{Role: RoleAssistant, Content: "已核查完毕。"},
	}}
	rt := &SubagentRuntime{ConvID: "conv-wl", Provider: provider, Registry: reg, Events: make(chan Event, 16),
		ParentCtx: context.Background()}
	unreg := RegisterSubagentRuntime(rt)
	defer unreg()
	restoreCur := SetCurrentSubagentRuntime(rt)
	defer restoreCur()

	res, err := RunSubagent(nil, SubagentSpec{Name: "supervisor", Task: "核查", Tools: []string{"read"}})
	if err != nil {
		t.Fatalf("RunSubagent 失败: %v", err)
	}
	if res.Ended != "completed" {
		t.Errorf("Ended = %q, want completed", res.Ended)
	}
	// 白名单外工具不存在时不应报错（Subset 静默丢弃），空名单则报错
	if _, err := RunSubagent(nil, SubagentSpec{Name: "supervisor", Task: "x", Tools: []string{"nope"}}); err == nil {
		t.Errorf("白名单全未命中时应报错")
	}
}

// ── 自主模式决策器注册位 ────────────────────────────────────────

// TestRunAutopilotDecision_NoImpl 未注册决策器（未装插件）→ 未处理（宿主按无监督收尾）。
func TestRunAutopilotDecision_NoImpl(t *testing.T) {
	old := jsAutopilotVal
	jsAutopilotVal = nil
	defer func() { jsAutopilotVal = old }()

	d := RunAutopilotDecision(context.Background(), AutopilotRequest{ConvID: "c1", Round: 1})
	if d.Handled {
		t.Errorf("Handled = true, want false（未注册实现）")
	}
	if d.Continue() {
		t.Errorf("未处理时不应要求继续")
	}
	if AutopilotAvailable() {
		t.Errorf("AutopilotAvailable = true, want false")
	}
	if n := AutopilotDecisionNotice(d, 1); n != "" {
		t.Errorf("未处理不应产生通知，得到 %q", n)
	}
}

// TestLoop_MaxSuperviseRoundsOrDefault 监督轮数上限的默认与覆盖语义。
func TestLoop_MaxSuperviseRoundsOrDefault(t *testing.T) {
	l := &Loop{}
	if got := l.MaxSuperviseRoundsOrDefault(); got != DefaultMaxSuperviseRounds {
		t.Errorf("默认上限 = %d, want %d", got, DefaultMaxSuperviseRounds)
	}
	l.MaxSuperviseRounds = 5
	if got := l.MaxSuperviseRoundsOrDefault(); got != 5 {
		t.Errorf("覆盖上限 = %d, want 5", got)
	}
}

// ── 决策请求取材辅助 ────────────────────────────────────────────

// TestAutopilotHelpers 决策请求上下文的取材口径（跳过系统注入消息、取尾部、截断）。
func TestAutopilotHelpers(t *testing.T) {
	hist := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "【历史压缩】旧摘要（注入物，非任务）"},
		{Role: RoleUser, Content: "实现自主模式插件"},
		{Role: RoleAssistant, Content: "先看代码结构"},
		{Role: RoleTool, Name: "read", Content: strings.Repeat("x", autopilotHistoryMsgChars+50)},
		{Role: RoleAssistant, Content: "已完成初版"},
	}
	if got := FirstUserTask(hist); got != "实现自主模式插件" {
		t.Errorf("FirstUserTask = %q（应跳过注入消息）", got)
	}
	if got := LastAssistantContent(hist); got != "已完成初版" {
		t.Errorf("LastAssistantContent = %q", got)
	}
	tail := RecentHistoryTail(hist, 3)
	if len(tail) != 3 {
		t.Fatalf("RecentHistoryTail 条数 = %d, want 3", len(tail))
	}
	if tail[1].Name != "read" {
		t.Errorf("尾部第 2 条应是 read 工具消息，得到 %+v", tail[1])
	}
	if len([]rune(tail[1].Content)) > autopilotHistoryMsgChars+10 {
		t.Errorf("工具结果未截断：长度 %d", len(tail[1].Content))
	}
	if got := RecentHistoryTail(nil, 5); got != nil {
		t.Errorf("空历史应返回 nil")
	}
}

// TestSuperviseDecideTimeout_Env 决策器调用超时：默认 20 分钟；环境变量可覆盖（排障用）。
func TestSuperviseDecideTimeout_Env(t *testing.T) {
	t.Setenv("PAIR_AUTOPILOT_TIMEOUT", "")
	if got := SuperviseDecideTimeout(); got != DefaultSuperviseDecideTimeout {
		t.Fatalf("默认超时 = %v，期望 %v", got, DefaultSuperviseDecideTimeout)
	}
	t.Setenv("PAIR_AUTOPILOT_TIMEOUT", "30")
	if got := SuperviseDecideTimeout(); got != 30*time.Second {
		t.Fatalf("环境变量覆盖超时 = %v，期望 30s", got)
	}
	t.Setenv("PAIR_AUTOPILOT_TIMEOUT", "abc")
	if got := SuperviseDecideTimeout(); got != DefaultSuperviseDecideTimeout {
		t.Fatalf("非法值应回落默认，得 %v", got)
	}
}
