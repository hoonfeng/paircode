package agent

import (
	"sync"
	"time"
)

// ── 审核共享上下文值（approve.state）─────────────────────────────
//
// ★ 2026-08-27 错误计数移除：连续驳回计数/自动停止（rejectionCount/
//   maxRejections/blocked）已删除——驳回仅反馈继续，打破死循环由绕圈检测
//   （circling.failStop）兜底。审核决策状态改为「共享上下文的值」：
//   Go 会话级持有（Loop 实例），JS 插件经 loop.approve.state.get/set
//   读写，agentloop 审核逻辑据此决策（如：同一工具刚被驳回 → 驳回反馈
//   追加提醒，不依赖计数）。

// RejectRecord 一次驳回记录（共享状态中的历史条目）。
type RejectRecord struct {
	Tool   string `json:"tool"`
	Reason string `json:"reason"`
	At     int64  `json:"at"` // unix 毫秒
}

// ApproveState 审核共享状态：最近驳回决策 + 轻量历史（最近 5 条，不计数）。
type ApproveState struct {
	mu               sync.Mutex
	LastRejectedTool string         `json:"lastRejectedTool"`
	LastRejectedAt   int64          `json:"lastRejectedAt"`
	RejectedHistory  []RejectRecord `json:"rejectedHistory"`
}

const maxRejectHistory = 5

// getApproveState 懒初始化共享审核状态（Loop 方法，l.mu 保护）。
func (l *Loop) getApproveState() *ApproveState {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.approveState == nil {
		l.approveState = &ApproveState{}
	}
	return l.approveState
}

// Snapshot 返回状态快照（含锁）。
func (s *ApproveState) Snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]any{
		"lastRejectedTool":   s.LastRejectedTool,
		"lastRejectedAt":     s.LastRejectedAt,
		"rejectedHistoryLen": len(s.RejectedHistory),
	}
	if len(s.RejectedHistory) > 0 {
		hist := make([]map[string]any, 0, len(s.RejectedHistory))
		for _, r := range s.RejectedHistory {
			hist = append(hist, map[string]any{"tool": r.Tool, "reason": r.Reason, "at": r.At})
		}
		out["rejectedHistory"] = hist
	}
	return out
}

// Set 合并更新（JSON 对象覆盖字段；供 JS 插件读写共享上下文值）。
// 支持重置：{lastRejectedTool: ""} 清空最近驳回标记。
func (s *ApproveState) Set(obj map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := obj["lastRejectedTool"].(string); ok {
		s.LastRejectedTool = v
		if v == "" {
			s.LastRejectedAt = 0
		}
	}
	if v, ok := obj["lastRejectedAt"].(float64); ok {
		s.LastRejectedAt = int64(v)
	}
	if v, ok := obj["lastRejectedAt"].(int64); ok {
		s.LastRejectedAt = v
	}
	// rejectedHistory 整体替换（校验结构）
	if v, ok := obj["rejectedHistory"].([]any); ok {
		hist := make([]RejectRecord, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				rec := RejectRecord{}
				if ts, ok := m["tool"].(string); ok {
					rec.Tool = ts
				}
				if rs, ok := m["reason"].(string); ok {
					rec.Reason = rs
				}
				if at, ok := m["at"].(float64); ok {
					rec.At = int64(at)
				}
				hist = append(hist, rec)
			}
		}
		s.RejectedHistory = hist
	}
}

// recordReject 记录一次驳回（ask 内部调用）：更新最近驳回 + 追加历史（截断 5 条）。
func (s *ApproveState) recordReject(tool, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli()
	s.LastRejectedTool = tool
	s.LastRejectedAt = now
	s.RejectedHistory = append(s.RejectedHistory, RejectRecord{Tool: tool, Reason: reason, At: now})
	if len(s.RejectedHistory) > maxRejectHistory {
		s.RejectedHistory = s.RejectedHistory[len(s.RejectedHistory)-maxRejectHistory:]
	}
}

// clearTool 工具通过审核时清掉对该工具的最近驳回标记。
func (s *ApproveState) clearTool(tool string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tool == "" || s.LastRejectedTool == tool {
		s.LastRejectedTool = ""
		s.LastRejectedAt = 0
	}
}
