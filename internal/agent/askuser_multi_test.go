package agent

// ═══════════════════════════════════════════════════════════════
// askuser_multi_test.go — Round3 ⑤：ask_user 多问题宿主协议测试
//
// 覆盖：参数解析（questions 数组）、回答回灌解析（answers JSON）、
// SessionManager 结构化路由（SendAnswers/WaitAnswers 回环 + 单问题兼容）、
// 存档执行器多问题路径（假会话桥 WaitAnswers）。
// ═══════════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestAskUserMultiQuestion_Parse 参数/结果解析：questions 数组编解码。
func TestAskUserMultiQuestion_Parse(t *testing.T) {
	// ① 参数解析：questions 优先
	_, _, _, questions := parseAskArgsV2(`{
		"questions": [
			{"id": "q1", "question": "选哪个方案？", "options": ["方案A", "方案B"]},
			{"id": "q2", "question": "预算多少？", "multi_select": true}
		]
	}`)
	if len(questions) != 2 {
		t.Fatalf("questions 应 2 条，得 %d", len(questions))
	}
	if questions[0].ID != "q1" || questions[0].Question != "选哪个方案？" ||
		len(questions[0].Options) != 2 || questions[1].MultiSelect != true {
		t.Errorf("questions 解析异常: %+v", questions)
	}
	// ② 单问题回落（无 questions）
	q, askType, opts, qs := parseAskArgsV2(`{"question":"是或否？","askType":"single","options":["是","否"]}`)
	if qs != nil || q != "是或否？" || askType != "single" || len(opts) != 2 {
		t.Errorf("单问题回落异常: %q %q %v %v", q, askType, opts, qs)
	}
	// ③ 回答回灌解析：answers JSON 数组
	_, answers := parseAskResultV2(`{"answers":[{"id":"q1","answer":"方案A"},{"id":"q2","answer":"100万"}]}`)
	if len(answers) != 2 || answers[0].ID != "q1" || answers[0].Answer != "方案A" {
		t.Errorf("answers 解析异常: %+v", answers)
	}
	// ④ 非 JSON 内容 → 单问题回落（整段即答案）
	ans, ans2 := parseAskResultV2("直接文本回答")
	if ans != "直接文本回答" || ans2 != nil {
		t.Errorf("单问题回落异常: %q %v", ans, ans2)
	}
}

// TestAskUserMultiQuestion_Routing SessionManager 结构化路由回环 + 单问题兼容。
func TestAskUserMultiQuestion_Routing(t *testing.T) {
	mgr := NewSessionManager()
	mgr.mu.Lock()
	mgr.sessions["conv-mq"] = &Session{
		ConvID:  "conv-mq",
		Running: true,
		askCh:   make(chan []AskAnswer, 1),
	}
	mgr.mu.Unlock()

	// 多问题：SendAnswers → WaitAnswers 回环
	go func() {
		time.Sleep(30 * time.Millisecond)
		if err := mgr.SendAnswers("conv-mq", []AskAnswer{{ID: "q1", Answer: "方案B"}, {ID: "q2", Answer: "80万"}}); err != nil {
			t.Errorf("SendAnswers: %v", err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	answers, err := mgr.WaitAnswers(ctx, "conv-mq")
	if err != nil {
		t.Fatalf("WaitAnswers: %v", err)
	}
	if len(answers) != 2 || answers[0].ID != "q1" || answers[1].Answer != "80万" {
		t.Errorf("answers 回环异常: %+v", answers)
	}

	// 单问题兼容：SendAnswer → WaitAnswer（单元素数组编解码）
	go func() {
		time.Sleep(30 * time.Millisecond)
		_ = mgr.SendAnswer("conv-mq", " 单问题回答 ")
	}()
	answer, err := mgr.WaitAnswer(ctx, "conv-mq")
	if err != nil {
		t.Fatalf("WaitAnswer: %v", err)
	}
	if answer != "单问题回答" {
		t.Errorf("单问题回环异常: %q", answer)
	}
}

// TestAskUserMultiQuestion_Executor 存档执行器多问题路径（假会话桥 WaitAnswers）。
func TestAskUserMultiQuestion_Executor(t *testing.T) {
	SetSessionBridge(&SessionBridge{
		WaitAnswers: func(ctx context.Context, convID string) ([]AskAnswer, error) {
			return []AskAnswer{{ID: "q1", Answer: "方案A"}, {ID: "q2", Answer: "50万"}}, nil
		},
		WaitAnswer: func(ctx context.Context, convID string) (string, error) {
			return "单问题回答", nil
		},
		GetWorkspaceRoot: func(convID string) string { return "C:/ws" },
	})
	defer SetSessionBridge(&SessionBridge{})

	// 多问题 → answers JSON 回灌
	out, err := ExecuteHostTool("ask_user", map[string]any{
		"_convID":   "conv-1",
		"questions": []any{map[string]any{"id": "q1", "question": "A?"}, map[string]any{"id": "q2", "question": "B?"}},
	})
	if err != nil {
		t.Fatalf("多问题执行器: %v", err)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("多问题应返回 answers JSON: %v（%s）", err, out)
	}
	as, _ := res["answers"].([]any)
	if len(as) != 2 {
		t.Errorf("answers 应 2 条: %s", out)
	}
	if !strings.Contains(out, "方案A") || !strings.Contains(out, "50万") {
		t.Errorf("answers 内容异常: %s", out)
	}

	// 单问题回落 → WaitAnswer 路径
	out, err = ExecuteHostTool("ask_user", map[string]any{"_convID": "conv-1", "question": "是或否？"})
	if err != nil {
		t.Fatalf("单问题执行器: %v", err)
	}
	if out != "单问题回答" {
		t.Errorf("单问题回落异常: %q", out)
	}
}
