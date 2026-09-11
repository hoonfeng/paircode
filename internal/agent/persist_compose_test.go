package agent

// persist_compose_test.go — 持久化组合（composePersistMessages）单测：
//   - 基准 + 最后一条真实 user 之后的新增；
//   - 背景快照（backgroundCtxMarker）/ 交接视图（handoffTitle）不作为锚点、不入盘；
//   - 多段累积（「可变底账」核心修复：第二段不丢第一段新增）。

import (
	"testing"
)

// TestComposePersist_Basic 基本组合：基准 + 最后一条真实 user 之后的新增。
func TestComposePersist_Basic(t *testing.T) {
	base := []Message{{Role: RoleUser, Content: "u1"}, {Role: RoleAssistant, Content: "a1"}}
	msgs := []Message{
		{Role: RoleUser, Content: "u1"}, {Role: RoleAssistant, Content: "a1"},
		{Role: RoleUser, Content: "u2"}, {Role: RoleAssistant, Content: "a2"},
	}
	combined := composePersistMessages(base, msgs)
	if len(combined) != 3 || combined[2].Content != "a2" {
		t.Fatalf("组合应为 base + [a2]，实际 %d 条: %+v", len(combined), combined)
	}
}

// TestComposePersist_BackgroundNotAnchor 背景快照（backgroundCtxMarker）不作为锚点。
func TestComposePersist_BackgroundNotAnchor(t *testing.T) {
	base := []Message{{Role: RoleUser, Content: "u1"}}
	msgs := []Message{
		{Role: RoleUser, Content: "u1"},
		{Role: RoleAssistant, Content: "a1"},
		{Role: RoleUser, Content: backgroundCtxMarker + "快照内容"},
		{Role: RoleAssistant, Content: "a2"},
	}
	combined := composePersistMessages(base, msgs)
	// 锚 = u1（背景快照被跳过）→ tail = [a1, 快照, a2] → combined = [u1, a1, 快照, a2]
	if len(combined) != 4 {
		t.Fatalf("锚应跳过背景快照，实际 %d 条: %+v", len(combined), combined)
	}
	if combined[2].Content != backgroundCtxMarker+"快照内容" {
		t.Fatalf("背景快照应随 tail 入盘: %+v", combined)
	}
}

// TestComposePersist_HandoffView 交接视图注入时：视图（交接消息/保留段旧消息）
// 不重复入盘，当前任务之后的新增正常追加。
func TestComposePersist_HandoffView(t *testing.T) {
	base := []Message{{Role: RoleUser, Content: "任务A"}, {Role: RoleAssistant, Content: "a1"}}
	handoffText := backgroundCtxMarker + handoffTitle + "\n交接正文"
	msgs := []Message{
		{Role: RoleUser, Content: handoffText}, // 交接视图（以背景前缀开头，非真实任务）
		{Role: RoleAssistant, Content: "旧a1"},  // 保留段（历史消息，已在 base 内）
		{Role: RoleUser, Content: "任务B"},       // 当前任务（锚点）
		{Role: RoleAssistant, Content: "a2"},   // 本轮新增
	}
	combined := composePersistMessages(base, msgs)
	// 锚 = 任务B → tail = [a2] → [任务A, a1, a2]（交接与保留段不入盘、不重复）
	if len(combined) != 3 || combined[0].Content != "任务A" || combined[2].Content != "a2" {
		t.Fatalf("交接视图不应入盘，组合应为 [任务A, a1, a2]，实际 %d 条: %+v", len(combined), combined)
	}
}

// TestComposePersist_NoUserFallback 无 user 消息 → 全量兜底。
func TestComposePersist_NoUserFallback(t *testing.T) {
	base := []Message{{Role: RoleUser, Content: "u1"}}
	msgs := []Message{{Role: RoleAssistant, Content: "a1"}}
	combined := composePersistMessages(base, msgs)
	if len(combined) != 1 || combined[0].Content != "a1" {
		t.Fatalf("无 user 应兜底全量，实际 %+v", combined)
	}
}

// TestComposePersist_MultiSegmentAccumulation 多段累积（「可变底账」核心修复）：
// 第二段的组合必须保留「第一段新增消息」——旧逻辑（固定 base=首段提交时快照）
// 的对照结果会丢掉第一段新增。
func TestComposePersist_MultiSegmentAccumulation(t *testing.T) {
	original := []Message{{Role: RoleUser, Content: "任务A"}} // 首轮提交时快照

	// 第一段结束：msgs = [任务A, a1]
	msgs1 := []Message{
		{Role: RoleUser, Content: "任务A"},
		{Role: RoleAssistant, Content: "a1"},
	}
	combined1 := composePersistMessages(original, msgs1)
	if len(combined1) != 2 || combined1[1].Content != "a1" {
		t.Fatalf("第一段组合应为 [任务A, a1]，实际 %+v", combined1)
	}

	// 段边界推进基准 = store 当前内容（combined1）
	base2 := combined1

	// 第二段：msgs = [任务A, a1, 续跑消息(锚), a2]
	msgs2 := []Message{
		{Role: RoleUser, Content: "任务A"},
		{Role: RoleAssistant, Content: "a1"},
		{Role: RoleUser, Content: "上一执行段因工具调用轮次预算用尽…继续"},
		{Role: RoleAssistant, Content: "a2"},
	}
	combined2 := composePersistMessages(base2, msgs2)
	// 期望：[任务A, a1, a2]（a1 保留！续跑消息不落盘）
	if len(combined2) != 3 || combined2[1].Content != "a1" || combined2[2].Content != "a2" {
		t.Fatalf("第二段应保留第一段新增（3 条），实际 %d 条: %+v", len(combined2), combined2)
	}

	// 对照：旧逻辑（固定 base=首轮快照）会丢第一段新增 a1
	oldCombined := composePersistMessages(original, msgs2)
	if len(oldCombined) != 2 || oldCombined[1].Content != "a2" {
		t.Fatalf("对照组应为 2 条（丢 a1），实际 %d 条: %+v", len(oldCombined), oldCombined)
	}
}
