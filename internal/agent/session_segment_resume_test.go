// session_segment_resume_test.go — 分段续跑端到端回归（★ 2026-09-12）
//
// 背景：原「最大迭代数」（maxIterations）配置项已移除——段的收束不再由迭代数
// 止损，而由「段预算双闸门（stepBudget 步数默认 120 / toolCallBudget 工具调用默认
// 120，任一达上限即分段）+ 自动续跑（maxToolBudgetSegments，默认 20）」负责
// （见 tool_budget.go）。
//
// 本测试钉死这两项分段续跑配置的端到端行为：用「永远调用工具、从不自然终止」
// 的 mock Provider 触发预算耗尽，验证 SessionManager 确实会自动开启下一段，
// 且续跑段数不超过生效上限（达上限后停止并推送用户可见通知）。
// 依赖 loop_test.go 中的 alwaysToolProvider。
package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSessionManager_ToolBudgetSegmentResume(t *testing.T) {
	// 固定走默认 Go 内核工厂（不受插件装配器影响，保证测试确定性）
	restore := ReplaceLoopFactory(goLoopFactory{})
	defer restore()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	prov := &alwaysToolProvider{}

	m := NewSessionManager()
	convID := "conv-seg-resume"
	// 极小配置强制分段：单段预算 1 次工具调用 / 续跑段数上限 2
	//   → 首段 + 2 次自动续跑 = 3 个执行段；第 3 次预算耗尽时 segmentNo(3) > 2
	//     → 停止续跑并推送上限通知。
	opts := LoopOpts{
		Provider:              prov,
		Registry:              reg,
		WorkspaceRoot:         dir,
		ToolCallBudget:        1,
		MaxToolBudgetSegments: 2,
	}
	if err := m.Start(context.Background(), convID, "请持续调用工具直到预算耗尽", opts); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// 实时消费事件（订阅缓冲仅 100，积压会被丢弃 → 「达上限」通知可能丢）
	ch := m.Subscribe(convID)
	var mu sync.Mutex
	var notices []string
	if ch != nil {
		go func() {
			for e := range ch {
				if e.Type == EventNotice {
					mu.Lock()
					notices = append(notices, e.Content)
					mu.Unlock()
				}
			}
		}()
	}

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) && m.IsRunning(convID) {
		time.Sleep(20 * time.Millisecond)
	}
	if m.IsRunning(convID) {
		t.Fatal("会话未在 20s 内结束（分段续跑可能失控）")
	}
	time.Sleep(300 * time.Millisecond) // 等 fan-out 投递完末段事件

	// ① 每段恰好用满预算 1 次工具调用 → 3 段 = 3 次 LLM 调用
	if prov.n != 3 {
		t.Errorf("应恰好 3 个执行段（预算 1 × 3 段），LLM 调用 %d 次", prov.n)
	}
	// ② 应注入 2 条续跑消息（段上限 2，第 3 次不再续跑）
	hist := m.GetHistory(convID)
	cont := 0
	for _, msg := range hist {
		if msg.Role == RoleUser && strings.Contains(msg.Content, "预算用尽") {
			cont++
		}
	}
	if cont != 2 {
		t.Errorf("应注入 2 条续跑消息（段上限 2），得 %d（历史共 %d 条）", cont, len(hist))
	}
	// ③ 达上限应有用户可见通知
	mu.Lock()
	snapshot := append([]string(nil), notices...)
	mu.Unlock()
	found := false
	for _, n := range snapshot {
		if strings.Contains(n, "分段已达上限") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("达续跑段上限应推送「分段已达上限」通知，实得通知 %v", snapshot)
	}
}
