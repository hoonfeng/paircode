package agent

// ═══════════════════════════════════════════════════════════════
// goal_test.go — Round3 ③.1：goal 宿主状态机测试
//
// 覆盖：create/get/update 全 action + revision 冲突拒绝 + 持久化 +
// 自动续轮判定（轮次上限 / pause / resume / 同一阻塞条件连续 ≥3 轮自动 blocked）。
// ═══════════════════════════════════════════════════════════════

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoalManager create/get/update 全 action + revision 冲突 + 持久化。
func TestGoalManager(t *testing.T) {
	wsRoot := t.TempDir()
	convID := "conv-goal-1"
	gm := NewGoalManager()

	// create（默认轮次上限 3）
	g, err := gm.Create(wsRoot, convID, "修复登录超时 bug", 3)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if g.Revision != 1 || !g.Armed || g.Phase != GoalPhaseActive || g.Rounds != 0 {
		t.Errorf("create 初始状态异常: %+v", g)
	}
	// 重复 create 拒绝
	if _, err := gm.Create(wsRoot, convID, "again", 3); err == nil {
		t.Error("重复 create 应拒绝")
	}
	// 空 objective 拒绝
	if _, err := gm.Create(wsRoot, "conv-goal-2", "  ", 3); err == nil {
		t.Error("空 objective 应拒绝")
	}

	// Get（内存命中）
	if g2 := gm.Get(wsRoot, convID); g2 == nil || g2.Objective != "修复登录超时 bug" {
		t.Errorf("Get 异常: %+v", g2)
	}

	// update edit（revision 乐观锁）
	g, err = gm.Update(wsRoot, convID, 1, "edit", "修复登录超时并补测试", 5, "")
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if g.Revision != 2 || g.Objective != "修复登录超时并补测试" || g.RoundLimit != 5 {
		t.Errorf("edit 结果异常: %+v", g)
	}
	// revision 冲突拒绝（旧 revision=1 已过期）
	if _, err := gm.Update(wsRoot, convID, 1, "edit", "x", -1, ""); err == nil {
		t.Error("过期 revision 应拒绝")
	}
	// 非法 action 拒绝
	if _, err := gm.Update(wsRoot, convID, 2, "bogus", "", -1, ""); err == nil {
		t.Error("非法 action 应拒绝")
	}

	// pause → 停续轮
	g, _ = gm.Update(wsRoot, convID, 2, "pause", "", -1, "")
	if g.Armed {
		t.Error("pause 后 Armed 应为 false")
	}
	if msg := g.ContinueMessage(); msg != "" {
		t.Errorf("pause 后不应续轮，得 %q", msg)
	}
	// resume → 重挂
	g, _ = gm.Update(wsRoot, convID, 3, "resume", "", -1, "")
	if !g.Armed || g.ContinueMessage() == "" {
		t.Error("resume 后应可续轮")
	}

	// complete → 终态不再续轮
	g, _ = gm.Update(wsRoot, convID, 4, "complete", "", -1, "")
	if g.Phase != GoalPhaseCompleted || g.ContinueMessage() != "" {
		t.Errorf("complete 后应终态且不续轮: %+v", g)
	}
	// 终态 resume 拒绝
	if _, err := gm.Update(wsRoot, convID, 5, "resume", "", -1, ""); err == nil {
		t.Error("终态 resume 应拒绝")
	}

	// blocked
	g2, err := gm.Create(wsRoot, "conv-goal-3", "目标C", 3)
	if err != nil {
		t.Fatal(err)
	}
	g2, _ = gm.Update(wsRoot, "conv-goal-3", g2.Revision, "blocked", "", -1, "依赖服务不可用")
	if g2.Phase != GoalPhaseBlocked || !strings.Contains(g2.BlockerReason, "依赖服务不可用") {
		t.Errorf("blocked 状态异常: %+v", g2)
	}

	// 持久化：新管理器实例读盘恢复（跨重启）
	gm2 := NewGoalManager()
	restored := gm2.Get(wsRoot, "conv-goal-3")
	if restored == nil || restored.Phase != GoalPhaseBlocked || restored.BlockerReason != "依赖服务不可用" {
		t.Errorf("持久化恢复异常: %+v", restored)
	}
	// 持久化文件确实存在
	if _, err := os.Stat(filepath.Join(wsRoot, ".pair", "goals", "conv-goal-3.json")); err != nil {
		t.Errorf("goal 持久化文件缺失: %v", err)
	}
}

// TestGoalAutoContinue 自动续轮：轮次上限 / 阻塞连续判定 / 系统提示注入段。
func TestGoalAutoContinue(t *testing.T) {
	wsRoot := t.TempDir()
	convID := "conv-goal-auto"
	gm := NewGoalManager()

	// 轮次上限：limit=3 → 第 1、2 轮后各续 1 次（共 3 轮），第 3 轮后停
	g, err := gm.Create(wsRoot, convID, "自动续轮目标", 3)
	if err != nil {
		t.Fatal(err)
	}
	rounds := 0
	for {
		g = gm.MarkRound(wsRoot, convID, nil)
		if g == nil || g.ContinueMessage() == "" {
			break
		}
		rounds++
		if rounds > 10 {
			t.Fatal("续轮未按上限停止")
		}
	}
	if rounds != 2 || g.Rounds != 3 {
		t.Errorf("续轮轮数异常: rounds=%d g.Rounds=%d（limit=3 应续 2 次共 3 轮）", rounds, g.Rounds)
	}
	// 续轮消息携带目标上下文
	g = gm.MarkRound(wsRoot, "conv-goal-auto", nil)
	_ = g
	msg := gm.Get(wsRoot, convID)
	_ = msg
	// 重建一个 limit=5 的目标验证消息内容
	g5, _ := gm.Create(wsRoot, "conv-goal-msg", "带上下文的续轮", 5)
	if m := g5.ContinueMessage(); !strings.Contains(m, "带上下文的续轮") || !strings.Contains(m, "自动续轮") {
		t.Errorf("续轮消息缺目标上下文: %q", m)
	}
	// 系统提示注入段
	sec := goalSystemSection(g5)
	if !strings.Contains(sec, goalSystemMarker) || !strings.Contains(sec, "带上下文的续轮") {
		t.Errorf("系统提示段异常: %q", sec)
	}

	// 同一阻塞条件连续 ≥3 轮 → 自动 blocked
	convB := "conv-goal-blocked"
	if _, err := gm.Create(wsRoot, convB, "阻塞目标", 5); err != nil {
		t.Fatal(err)
	}
	blkErr := errors.New("LLM API 500: upstream error")
	var last *Goal
	for i := 1; i <= 3; i++ {
		last = gm.MarkRound(wsRoot, convB, blkErr)
	}
	if last.Phase != GoalPhaseBlocked {
		t.Fatalf("连续 3 轮同一错误应自动 blocked，得 %+v", last)
	}
	if !strings.Contains(last.BlockerReason, "LLM API 500") {
		t.Errorf("blocked_reason 应记录阻塞条件: %q", last.BlockerReason)
	}
	if last.ContinueMessage() != "" {
		t.Error("blocked 后不应续轮")
	}
	// 阻塞计数跨不同错误重置：错误变化 → streak 重置（再来 3 轮不同错误不触发）
	convC := "conv-goal-streak"
	if _, err := gm.Create(wsRoot, convC, "重置目标", 10); err != nil {
		t.Fatal(err)
	}
	gm.MarkRound(wsRoot, convC, errors.New("err-A"))
	gm.MarkRound(wsRoot, convC, errors.New("err-B"))
	gm.MarkRound(wsRoot, convC, errors.New("err-C"))
	if g := gm.Get(wsRoot, convC); g.Phase == GoalPhaseBlocked {
		t.Error("不同错误不累计，不应 blocked")
	}
}

// TestGoalArchiveExecutors goal 路由执行器（create/get/update）经 _convID/_wsRoot 工作。
func TestGoalArchiveExecutors(t *testing.T) {
	wsRoot := t.TempDir()
	convID := "conv-goal-exec"

	// create（_convID/_wsRoot 由会话工具链注入 args）
	out, err := ExecuteHostTool("create_goal", map[string]any{"_convID": convID, "_wsRoot": wsRoot, "objective": "执行器目标", "max_goal_rounds": 4})
	if err != nil {
		t.Fatalf("create_goal 执行器: %v", err)
	}
	if !strings.Contains(out, "执行器目标") {
		t.Errorf("create 输出异常: %s", out)
	}

	// get（返回 JSON 字段齐全）
	out, err = ExecuteHostTool("get_goal", map[string]any{"_convID": convID, "_wsRoot": wsRoot})
	if err != nil {
		t.Fatalf("get_goal 执行器: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("get_goal 应返回 JSON: %v（%s）", err, out)
	}
	for _, k := range []string{"goal_id", "revision", "objective", "phase", "rounds", "roundLimit", "blockerReason", "armed"} {
		if _, ok := got[k]; !ok {
			t.Errorf("get_goal 缺字段 %s", k)
		}
	}

	// update complete（revision=1 → 2）
	out, err = ExecuteHostTool("update_goal", map[string]any{"_convID": convID, "_wsRoot": wsRoot, "goal_id": convID, "revision": 1, "action": "complete"})
	if err != nil {
		t.Fatalf("update_goal 执行器: %v", err)
	}
	if !strings.Contains(out, "completed") {
		t.Errorf("update 输出异常: %s", out)
	}
	// 缺 _convID → 明确报错
	if _, err := ExecuteHostTool("create_goal", map[string]any{"objective": "x"}); err == nil {
		t.Error("缺 _convID 应报错")
	}
}
