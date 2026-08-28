package agent

import (
	"testing"
)

// TestApproveStateRecordReject 验证驳回记录进共享审核状态（不计数）。
func TestApproveStateRecordReject(t *testing.T) {
	st := &ApproveState{}
	st.recordReject("write_file", "用户拒绝了写入")
	snap := st.Snapshot()
	if snap["lastRejectedTool"] != "write_file" {
		t.Errorf("lastRejectedTool = %v, want write_file", snap["lastRejectedTool"])
	}
	if snap["lastRejectedAt"].(int64) == 0 {
		t.Error("lastRejectedAt 应为非零")
	}
	hist, ok := snap["rejectedHistory"].([]map[string]any)
	if !ok || len(hist) != 1 {
		t.Fatalf("rejectedHistory = %#v, want 1 条", snap["rejectedHistory"])
	}
	if hist[0]["tool"] != "write_file" || hist[0]["reason"] != "用户拒绝了写入" {
		t.Errorf("历史记录 = %#v", hist[0])
	}
}

// TestApproveStateHistoryTrim 验证历史只保留最近 5 条。
func TestApproveStateHistoryTrim(t *testing.T) {
	st := &ApproveState{}
	for i := 0; i < 8; i++ {
		st.recordReject("tool", "r")
	}
	snap := st.Snapshot()
	hist, _ := snap["rejectedHistory"].([]map[string]any)
	if len(hist) != 5 {
		t.Fatalf("历史长度 = %d, want 5", len(hist))
	}
}

// TestApproveStateSetMerge 验证 set 合并改写与清空。
func TestApproveStateSetMerge(t *testing.T) {
	st := &ApproveState{}
	st.recordReject("write_file", "驳回A")
	// 改写 lastRejectedTool（不丢历史）
	st.Set(map[string]any{"lastRejectedTool": "delete_file"})
	snap := st.Snapshot()
	if snap["lastRejectedTool"] != "delete_file" {
		t.Errorf("lastRejectedTool = %v, want delete_file", snap["lastRejectedTool"])
	}
	// 清空
	st.Set(map[string]any{"lastRejectedTool": ""})
	snap = st.Snapshot()
	if snap["lastRejectedTool"] != "" || snap["lastRejectedAt"] != int64(0) {
		t.Errorf("清空后 = %#v", snap)
	}
}

// TestApproveStateClearTool 验证工具通过审核时清除最近驳回标记。
func TestApproveStateClearTool(t *testing.T) {
	st := &ApproveState{}
	st.recordReject("write_file", "驳回")
	// 其他工具通过 → 不清
	st.clearTool("read_file")
	if st.Snapshot()["lastRejectedTool"] != "write_file" {
		t.Error("无关工具不应清除最近驳回")
	}
	// 该工具通过 → 清
	st.clearTool("write_file")
	if st.Snapshot()["lastRejectedTool"] != "" {
		t.Error("通过后应清除最近驳回")
	}
}

// TestGetApproveStateLazy 验证懒初始化单例。
func TestGetApproveStateLazy(t *testing.T) {
	l := &Loop{}
	a := l.getApproveState()
	b := l.getApproveState()
	if a != b {
		t.Error("getApproveState 应返回同一实例")
	}
	if l.approveState == nil {
		t.Error("approveState 应被初始化")
	}
}
