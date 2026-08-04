package agent

// 临时量化验证：模拟多轮真实对话（含大工具输出），对比「压缩前」与「压缩后」的 LLM 注入体积。
// 运行：go test -count=1 ./internal/agent -run TestVerifyCondenseSaves -v

import (
	"fmt"
	"strings"
	"testing"
)

// bigToolResult 模拟大工具输出（read_file 全文 / run_command 大输出）。
func bigToolResult(n int) string {
	return strings.Repeat("行内容数据abcdefghijklmnopqrstuvwxyz0123456789\n", n)
}

// buildRealisticConversation 构造模拟真实对话：N 轮，每轮多次迭代，含大工具结果。
func buildRealisticConversation(rounds, itersPerRound, toolLines int) []Message {
	var msgs []Message
	msgs = append(msgs, Message{Role: RoleSystem, Content: "系统提示词（较长）" + strings.Repeat("铁律", 50)})
	for r := 0; r < rounds; r++ {
		msgs = append(msgs, Message{Role: RoleUser, Content: fmt.Sprintf("第 %d 轮任务：修复登录模块的样式问题", r+1)})
		for i := 0; i < itersPerRound; i++ {
			id := fmt.Sprintf("c%d_%d", r, i)
			msgs = append(msgs, Message{Role: RoleAssistant, Content: "我来分析问题",
				ToolCalls: []ToolCall{{ID: id, Function: FunctionCall{Name: "read_file", Arguments: `{"path":"src/login.vue"}`}}}},
				Message{Role: RoleTool, ToolCallID: id, Name: "read_file", Content: bigToolResult(toolLines)},
			)
		}
		msgs = append(msgs, Message{Role: RoleAssistant, Content: "已修复登录模块样式问题，验证通过"})
	}
	// 当前用户消息
	msgs = append(msgs, Message{Role: RoleUser, Content: "继续优化"})
	return msgs
}

func TestVerifyCondenseSaves(t *testing.T) {
	// 6 轮、每轮 4 次迭代、每次工具结果 300 行 ≈ 13KB（超过 9000 字符瘦身阈值）
	msgs := buildRealisticConversation(6, 4, 300)

	rawTokens := estimateTokens(msgs)

	// 模拟 LLM 视图：工具结果瘦身（buildCallContext 逻辑）
	loop := &Loop{}
	llmView := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		llmView = append(llmView, loop.trimToolResult(m))
	}
	trimmedTokens := estimateTokens(llmView)

	// 模拟跨 Run 压缩（CondenseHistory）
	condensed := CondenseHistory(msgs)
	condensedView := make([]Message, 0, len(condensed))
	for _, m := range condensed {
		condensedView = append(condensedView, loop.trimToolResult(m))
	}
	condensedTokens := estimateTokens(condensedView)

	t.Logf("原始历史全注入:        %6d tokens (%d 条消息)", rawTokens, len(msgs))
	t.Logf("+ 工具结果瘦身:        %6d tokens (省 %d%%)", trimmedTokens, 100*(rawTokens-trimmedTokens)/rawTokens)
	t.Logf("+ 跨Run压缩(Condense): %6d tokens (合计省 %d%%)", condensedTokens, 100*(rawTokens-condensedTokens)/rawTokens)

	if condensedTokens >= rawTokens {
		t.Errorf("压缩后应显著小于原始：%d >= %d", condensedTokens, rawTokens)
	}
	// 报告压缩率供观察
	saved := 100 * (rawTokens - condensedTokens) / rawTokens
	t.Logf(">>> 综合压缩率: %d%%（注入体积大幅下降，上下文语义经摘要保留）", saved)
}
