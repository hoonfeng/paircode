package agent

import (
	"context"
	"strings"
	"testing"
)

// TestEvaluatorScore 评测模型返回合法 JSON → 解析维度分/总分/反馈。
func TestEvaluatorScore(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Content: `这是评分：
{"scores":{"completion":36,"correctness":27,"depth":16,"efficiency":6},"total":85,"strengths":["计划清晰"],"weaknesses":["缺测试"],"feedback":"完成良好"}`}}}
	e := &Evaluator{Provider: mock}
	ev, err := e.Evaluate(context.Background(), "重构配置", "工具调用: 5 次\n错误: 0 次")
	if err != nil {
		t.Fatal(err)
	}
	if ev.Total != 85 || ev.Scores.Completion != 36 || ev.Scores.Efficiency != 6 {
		t.Errorf("评分解析错：%+v", ev)
	}
	if len(ev.Strengths) != 1 || ev.Feedback != "完成良好" {
		t.Errorf("优缺点/反馈错：%+v", ev)
	}
}

// TestEvaluatorTotalFromSum 无 total 字段 → 取各维度之和（复刻参考）。
func TestEvaluatorTotalFromSum(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Content: `{"scores":{"completion":30,"correctness":20,"depth":10,"efficiency":5}}`}}}
	e := &Evaluator{Provider: mock}
	ev, _ := e.Evaluate(context.Background(), "t", "s")
	if ev.Total != 65 {
		t.Errorf("无 total 应取维度和 65，得 %d", ev.Total)
	}
}

// TestParseEvaluationFallback 评测模型乱回 → 0 分 + 失败说明。
func TestParseEvaluationFallback(t *testing.T) {
	ev := parseEvaluation("我觉得做得不错")
	if ev.Total != 0 || len(ev.Weaknesses) == 0 {
		t.Errorf("解析失败应 0 分 + 失败说明，得 %+v", ev)
	}
}

// TestSummarizeRun 从消息历史提炼摘要：工具/错误计数 + 最终答复。
func TestSummarizeRun(t *testing.T) {
	msgs := []Message{
		{Role: RoleUser, Content: "改文件"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}}}},
		{Role: RoleTool, Content: "文件内容"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{Function: FunctionCall{Name: "write_file", Arguments: `{"path":"a.go"}`}}}},
		{Role: RoleTool, Content: "Error: 权限不足"},
		{Role: RoleAssistant, Content: "已完成修改 [FINAL]"},
	}
	s := SummarizeRun(msgs)
	if !strings.Contains(s, "工具调用: 2 次") || !strings.Contains(s, "错误: 1 次") {
		t.Errorf("计数错：%q", s)
	}
	if !strings.Contains(s, "read_file") || !strings.Contains(s, "已完成修改") {
		t.Errorf("最近调用/最终答复缺失：%q", s)
	}
}

// TestHasToolActivity 有工具调用→true，纯问答→false。
func TestHasToolActivity(t *testing.T) {
	if !HasToolActivity([]Message{{Role: RoleAssistant, ToolCalls: []ToolCall{{Function: FunctionCall{Name: "x"}}}}}) {
		t.Error("有工具调用应 true")
	}
	if HasToolActivity([]Message{{Role: RoleAssistant, Content: "答复"}}) {
		t.Error("纯问答应 false")
	}
}
