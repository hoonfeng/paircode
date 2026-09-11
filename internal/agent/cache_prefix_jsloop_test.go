package agent

// cache_prefix_jsloop_test.go — JS 循环路径「段内只追加」回归测试
//
// 历史问题（根因 B）：精简策略外置到 agentloop（JS）后，JS 路径不再执行 Go 的
// maybeCompact，冷却递减只写在 Go 侧 —— 冷却一旦设为 10 就永不递减，Run 内只精简
// 一次、历史一路膨胀、跨 Run 首请求固定 miss 一整窗。
//
// ★ 2026-09 决策：段内**只追加，不中途精简**（改写已发送字节必然断前缀缓存，
// 实测单次白付 13K~54K tokens）。体积控制交给 ① 每 toolCallBudget 次（默认 120，
// 设置面板可配）工具调用分段续跑（tool_budget.go）；
// ② 下一段开始时跨段精简（CondenseHistoryByPressure）。
// 本测试断言：极小窗口 + 70 步工具调用的长 Run 内
//   - 不产生任何 EventCompacted（段内不精简）；
//   - 每步视图都是历史的同一份追加（长度 == 历史长度，前缀零断裂）。
// 回归价值：若将来有人恢复「每步精简」，这里会立刻失败。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJSLoopNoMidRunCompact JS 循环路径段内不得中途精简（只追加）。
func TestJSLoopNoMidRunCompact(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	dir := t.TempDir()
	// 大结果文件：历史上这正是「每步都远超阈值 → 每步精简」的触发条件。
	os.WriteFile(filepath.Join(dir, "big.txt"), []byte(strings.Repeat("0123456789ABCDEF", 400)), 0o644)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	resps := make([]Message, 0, 71)
	for i := 0; i < 70; i++ {
		resps = append(resps, Message{ToolCalls: []ToolCall{{ID: fmt.Sprintf("c%d", i), Type: "function",
			Function: FunctionCall{Name: "read", Arguments: `{"path":"big.txt"}`}}}})
	}
	resps = append(resps, Message{Content: "done"})
	mock := &MockProvider{Responses: resps}

	var compacted int
	loop := &Loop{Provider: mock, Registry: reg, System: "s",
		MaxContextTokens: 300, // 极小窗口：历史实现下这里会疯狂精简
		OnEvent: func(e Event) {
			if e.Type == EventCompacted {
				compacted++
			}
		}}

	msgs, err := loop.Run(context.Background(), "跑", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if compacted != 0 {
		t.Fatalf("段内不应触发精简事件，得 %d（只追加语义）", compacted)
	}
	if loop.compactSlots != 0 || len(loop.compactArchive) != 0 {
		t.Errorf("段内不应产生视图块/归档：slots=%d archive=%d", loop.compactSlots, len(loop.compactArchive))
	}
	// 历史必须是完整追加的：工具结果全在（70 步 × 1 条 tool 消息）
	toolMsgs := 0
	for _, m := range msgs {
		if m.Role == RoleTool {
			toolMsgs++
		}
	}
	if toolMsgs != 70 {
		t.Errorf("70 步工具结果应全部保留（只追加），得 %d 条 tool 消息", toolMsgs)
	}
	t.Logf("长 Run 只追加：msgs=%d（其中 tool 结果 %d 条），段内精简事件 0", len(msgs), toolMsgs)
}
