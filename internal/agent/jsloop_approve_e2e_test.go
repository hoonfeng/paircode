package agent

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
)

// ─── 审核共享状态 e2e（错误计数移除后）──

// 注册一个需审批的测试工具（approve_me）。
func regApproveTool(r *Registry, name string) *atomic.Int32 {
	var calls atomic.Int32
	r.Register(&Tool{
		Name:            name,
		Description:     "需审批的测试工具",
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			calls.Add(1)
			return "OK:" + name, nil
		},
	})
	return &calls
}

// TestJSLoopApproveRejectState 审核驳回 → 共享状态记录 → 同一工具重试免打扰自动驳回。
func TestJSLoopApproveRejectState(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	reg := NewRegistry()
	toolCalls := regApproveTool(reg, "approve_me")

	// 用户审批回调：永远驳回（第 2 次申请应走免打扰自动驳回，不再触发此回调）
	var approveCalls atomic.Int32
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{
			{ID: "c1", Type: "function", Function: FunctionCall{Name: "approve_me", Arguments: `{"v":1}`}},
		}},
		// 被驳回后 LLM 原样重试同一工具
		{ToolCalls: []ToolCall{
			{ID: "c2", Type: "function", Function: FunctionCall{Name: "approve_me", Arguments: `{"v":2}`}},
		}},
		{Content: "好的，不执行该写入"},
	}}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "approve-e2e", MaxIterations: 5,
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) {
			approveCalls.Add(1)
			return false, "用户：不要执行写入"
		},
		OnEvent: func(e Event) { events = append(events, e) },
	}

	msgs, err := loop.Run(context.Background(), "调用 approve_me", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if mock.Calls() != 3 {
		t.Errorf("LLM 应调用 3 次，得 %d", mock.Calls())
	}
	// ① 用户审批回调只触发 1 次（第 2 次申请走免打扰自动驳回）
	if approveCalls.Load() != 1 {
		t.Errorf("审批回调应触发 1 次（免打扰自动驳回），得 %d", approveCalls.Load())
	}
	// ② 工具不应真正执行（两次都被驳回）
	if toolCalls.Load() != 0 {
		t.Errorf("approve_me 不应被执行，得 %d 次", toolCalls.Load())
	}
	// ③ 两条 tool 消息：第 1 条用户反馈，第 2 条免打扰自动驳回（含前次理由提示）
	var toolMsgs []Message
	for _, m := range msgs {
		if m.Role == RoleTool {
			toolMsgs = append(toolMsgs, m)
		}
	}
	if len(toolMsgs) != 2 {
		t.Fatalf("应有 2 条 tool 消息，得 %d", len(toolMsgs))
	}
	if !strings.Contains(toolMsgs[0].Content, "不要执行写入") {
		t.Errorf("第 1 条驳回反馈 = %q", toolMsgs[0].Content)
	}
	if !strings.Contains(toolMsgs[1].Content, "前一次驳回理由仍有效") {
		t.Errorf("第 2 条应为免打扰自动驳回（含前次理由提示），得 %q", toolMsgs[1].Content)
	}
	// ④ 共享审核状态：最近驳回 = approve_me，历史 1 条（自动驳回不重复记录）
	st := loop.getApproveState().Snapshot()
	if st["lastRejectedTool"] != "approve_me" {
		t.Errorf("lastRejectedTool = %v", st["lastRejectedTool"])
	}
	hist, _ := st["rejectedHistory"].([]map[string]any)
	if len(hist) != 1 {
		t.Errorf("历史应为 1 条（自动驳回不记录），得 %d", len(hist))
	}
	// ⑤ 无 blocked 停止（错误计数移除后不再自动停 turn）
	for _, e := range events {
		if e.Type == EventError && strings.Contains(e.Content, "自动停止") {
			t.Errorf("不应出现连续驳回自动停止事件：%q", e.Content)
		}
	}
}

// TestJSLoopApproveThenPass 驳回后换策略直通（白名单）→ 工具执行 + 状态清除。
func TestJSLoopApproveThenPass(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	reg := NewRegistry()
	toolCalls := regApproveTool(reg, "approve_me")

	var approveCalls atomic.Int32
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{
			{ID: "c1", Type: "function", Function: FunctionCall{Name: "approve_me", Arguments: `{"v":1}`}},
		}},
		{Content: "完成"},
	}}
	// 白名单直通：approve_me 免审
	loop := &Loop{Provider: mock, Registry: reg, System: "approve-pass-e2e", MaxIterations: 5,
		ReviewWhitelist: []string{"approve_me"},
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) {
			approveCalls.Add(1)
			return false, "不应被调用"
		},
	}
	// 先手动预置最近驳回标记（模拟上次会话驳回），直通应清掉
	loop.getApproveState().recordReject("approve_me", "历史驳回")

	msgs, err := loop.Run(context.Background(), "调用 approve_me", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if approveCalls.Load() != 0 {
		t.Errorf("白名单直通不应触发审批回调，得 %d", approveCalls.Load())
	}
	if toolCalls.Load() != 1 {
		t.Errorf("approve_me 应执行 1 次，得 %d", toolCalls.Load())
	}
	if mock.Calls() != 2 {
		t.Errorf("LLM 应调用 2 次，得 %d", mock.Calls())
	}
	// 直通后最近驳回标记应被清空
	if loop.getApproveState().Snapshot()["lastRejectedTool"] != "" {
		t.Error("直通后该工具的最近驳回标记应清空")
	}
	_ = msgs
}
