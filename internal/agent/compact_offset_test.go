package agent

// compact_offset_test.go — 聚焦验证：maybeCompact（compact 分支）压缩 msgs 后，
// OnBatchPersist 用「压缩前」的 condensedLen 计算 newTailStart 是否错位/失效。

import (
	"context"
	"strings"
	"testing"
)

// TestCompactBreaksPersistOffset 构造触发 compact 的长历史，复刻 OnBatchPersist
// 的固定偏移重组，验证是否会把压缩版历史写进 store（旧消息丢失）。
func TestCompactBreaksPersistOffset(t *testing.T) {
	// 构造长历史：30 条大消息（每条约 400 token）→ 总计约 12000 token
	// MaxContextTokens=8000 → 90% 阈值 7200 → 触发 compact
	hist := make([]Message, 0, 32)
	hist = append(hist, Message{Role: RoleUser, Content: "最早任务"})
	for i := 0; i < 15; i++ {
		hist = append(hist, Message{Role: RoleAssistant, Content: "分析第" + itoa(i) + "轮: " + strings.Repeat("数", 300)})
		hist = append(hist, Message{Role: RoleTool, Name: "run_command", Content: strings.Repeat("输出", 150) + "END"})
	}
	hist = append(hist, Message{Role: RoleUser, Content: "当前任务"})

	// 复刻 web 端：condensedLen = len(CondenseHistory(hist))（压缩前固定值）
	condensed := CondenseHistory(hist)
	condensedLen := len(condensed)
	originalHist := make([]Message, len(hist))
	copy(originalHist, hist)

	l := &Loop{MaxContextTokens: 8000}
	msgs := make([]Message, 0, len(condensed)+4)
	msgs = append(msgs, Message{Role: RoleSystem, Content: "test-sys"})
	msgs = append(msgs, condensed...)
	t.Logf("compact 前: len(condensed)=%d, msgs=%d, 末条=%s", condensedLen, len(msgs), msgs[len(msgs)-1].Role)

	compressed := l.maybeCompact(context.Background(), msgs)
	if len(compressed) == len(msgs) {
		t.Skipf("compact 未触发（len=%d）——调整参数", len(msgs))
	}
	t.Logf("compact 后: msgs=%d → %d, 末条=%s", len(msgs), len(compressed), compressed[len(compressed)-1].Role)
	if compressed[len(compressed)-1].Role != RoleUser || !strings.Contains(compressed[len(compressed)-1].Content, "当前任务") {
		t.Fatalf("compact 后当前任务丢失！末条=%q", shortContent(compressed[len(compressed)-1].Content))
	}

	// ── 复刻修复后的 OnBatchPersist 重组（lastUser 锚点，session_manager.go）──
	// 模拟第一轮迭代：追加一条带 tool_call 的 assistant
	compressed = append(compressed, Message{Role: RoleAssistant, Content: "计划", ToolCalls: []ToolCall{{ID: "c1", Function: FunctionCall{Name: "test_echo"}}}})

	// lastUser 锚点：最后一条 user（当前任务）之后 = 本轮新增
	var combined []Message
	lastUserIdx := -1
	for i := len(compressed) - 1; i >= 0; i-- {
		if compressed[i].Role == RoleUser {
			lastUserIdx = i
			break
		}
	}
	if lastUserIdx >= 0 {
		tail := compressed[lastUserIdx+1:]
		combined = make([]Message, 0, len(originalHist)+len(tail))
		combined = append(combined, originalHist...)
		combined = append(combined, tail...)
	} else {
		combined = compressed
	}

	// 验证 combined：
	//  1. 末尾必须包含新增 assistant（compressed 追加的那条）
	//  2. combined 长度 >= 原始历史长度 + 1
	t.Logf("combined 长度=%d（原始历史 %d + 新增 assistant 1 应 = %d）", len(combined), len(originalHist), len(originalHist)+1)
	if len(combined) != len(originalHist)+1 {
		t.Errorf("combined 长度错误：得 %d，期望 %d（assistant 可能丢失或历史被压缩覆盖）", len(combined), len(originalHist)+1)
	}
	last := combined[len(combined)-1]
	if last.Role != RoleAssistant || !strings.Contains(last.Content, "计划") {
		t.Errorf("combined 末条应为新增 assistant（计划），得 %s: %q", last.Role, shortContent(last.Content))
	}
	// 检查历史完整性：combined 前段应保留全部原始历史（无压缩覆盖）
	for i, m := range originalHist {
		if combined[i].Role != m.Role || combined[i].Content != m.Content {
			t.Errorf("combined[%d] 与原始历史不一致：%s vs %s", i, shortContent(combined[i].Content), shortContent(m.Content))
			break
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
