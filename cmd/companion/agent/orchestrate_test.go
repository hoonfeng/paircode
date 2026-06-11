package agent

import (
	"context"
	"strings"
	"testing"
)

// ─── 规划 Agent ───────────────────────────────────────────────

// TestPlannerPlan 规划模型返回合法 JSON → 解析出 reasoning + steps。
func TestPlannerPlan(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Content: `好的，计划如下：
{"reasoning":"先读配置再重构","steps":[
{"id":"step-1","description":"理解配置模块结构","dependencies":[]},
{"id":"step-2","description":"重构配置加载","dependencies":["step-1"],"isDestructive":false}]}`}}}
	p := &Planner{Provider: mock}
	plan, err := p.Plan(context.Background(), "重构配置模块", nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Reasoning != "先读配置再重构" || len(plan.Steps) != 2 {
		t.Fatalf("计划解析错：%+v", plan)
	}
	if plan.Steps[1].ID != "step-2" || len(plan.Steps[1].Dependencies) != 1 {
		t.Errorf("步骤字段错：%+v", plan.Steps[1])
	}
}

// TestPlannerFallback 连续 3 次返回无法解析 → 回退默认 5 步计划。
func TestPlannerFallback(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Content: "抱歉无法规划"}, {Content: "还是不行"}, {Content: "依旧失败"}}}
	p := &Planner{Provider: mock}
	plan, err := p.Plan(context.Background(), "做点复杂的事", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Steps) != 5 || !strings.Contains(plan.Reasoning, "默认计划") {
		t.Errorf("应回退默认 5 步计划，得 %+v", plan)
	}
	if mock.Calls() != 3 {
		t.Errorf("应重试 3 次，得 %d", mock.Calls())
	}
}

// TestPlannerEmptySteps 问候/闲聊 → steps 空数组也算成功（不回退默认）。
func TestPlannerEmptySteps(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Content: `{"reasoning":"只是问候","steps":[]}`}}}
	p := &Planner{Provider: mock}
	plan, _ := p.Plan(context.Background(), "你好", nil)
	if len(plan.Steps) != 0 || plan.Reasoning != "只是问候" {
		t.Errorf("空步骤计划应原样返回，得 %+v", plan)
	}
	if mock.Calls() != 1 {
		t.Errorf("成功应只调用 1 次，得 %d", mock.Calls())
	}
}

// capProvider 记录收到的 system 提示（测自定义角色提示用）。
type capProvider struct {
	system string
	reply  string
}

func (c *capProvider) Name() string { return "cap" }
func (c *capProvider) Chat(_ context.Context, msgs []Message, _ []ToolDefinition, _ func(Chunk)) (Message, error) {
	for _, m := range msgs {
		if m.Role == RoleSystem {
			c.system = m.Content
		}
	}
	return Message{Role: RoleAssistant, Content: c.reply}, nil
}

// TestRoleCustomPrompt 角色 Agent 用宿主传入的 SystemPrompt（从 config 加载）；空则回退内置默认。
func TestRoleCustomPrompt(t *testing.T) {
	if DefaultPlannerPrompt() == "" || DefaultReviewerPrompt() == "" || DefaultJudgePrompt() == "" {
		t.Fatal("内置默认角色提示不应为空")
	}
	cp := &capProvider{reply: `{"reasoning":"x","steps":[{"id":"s1","description":"d"}]}`}
	(&Planner{Provider: cp, SystemPrompt: "自定义规划提示XYZ"}).Plan(context.Background(), "t", nil)
	if cp.system != "自定义规划提示XYZ" {
		t.Errorf("应用宿主自定义提示，得 %q", cp.system)
	}
	cr := &capProvider{reply: `{"verdict":"通过"}`}
	(&Reviewer{Provider: cr}).Review(context.Background(), writeCall("a.go", "x"))
	if !strings.Contains(cr.system, "审核 Agent") {
		t.Error("空 SystemPrompt 应回退内置默认审核提示")
	}
}

// ─── 审核 Agent ───────────────────────────────────────────────

func writeCall(path, content string) ToolCall {
	return ToolCall{ID: "w", Type: "function", Function: FunctionCall{Name: "write_file",
		Arguments: `{"path":"` + path + `","content":"` + content + `"}`}}
}

// TestReviewerApprove 审核模型判「通过」→ Approved()=true。
func TestReviewerApprove(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Content: `{"verdict":"通过","confidence":0.9,"issues":[],"suggestions":[],"summary":"变更安全合理"}`}}}
	r := &Reviewer{Provider: mock}
	v, err := r.Review(context.Background(), writeCall("a.go", "package main"))
	if err != nil {
		t.Fatal(err)
	}
	if !v.Approved() || v.Summary != "变更安全合理" {
		t.Errorf("应通过，得 %+v", v)
	}
}

// TestReviewerReject 判「驳回」→ 不放行，FeedbackText 含结论与建议（供回灌）。
func TestReviewerReject(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Content: `{"verdict":"驳回","confidence":0.95,"suggestions":["改用参数化查询","校验输入"],"summary":"存在 SQL 注入风险"}`}}}
	r := &Reviewer{Provider: mock}
	v, _ := r.Review(context.Background(), writeCall("db.go", "query"))
	if v.Approved() {
		t.Error("驳回不应放行")
	}
	fb := v.FeedbackText()
	if !strings.Contains(fb, "SQL 注入") || !strings.Contains(fb, "参数化查询") {
		t.Errorf("反馈应含结论与建议，得 %q", fb)
	}
}

// TestReviewerCriticalFile 删除关键文件 → 直接驳回，不调审核模型（省一次 LLM）。
func TestReviewerCriticalFile(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Content: `{"verdict":"通过"}`}}}
	r := &Reviewer{Provider: mock}
	tc := ToolCall{ID: "d", Type: "function", Function: FunctionCall{Name: "delete_file", Arguments: `{"path":"src/go.mod"}`}}
	v, _ := r.Review(context.Background(), tc)
	if v.Approved() || !strings.Contains(v.Summary, "关键文件") {
		t.Errorf("删除关键文件应直接驳回，得 %+v", v)
	}
	if mock.Calls() != 0 {
		t.Errorf("关键文件保护不应调审核模型，得 %d 次", mock.Calls())
	}
}

// TestParseVerdictFallback 审核模型乱回 → 解析失败回退「需要修改」（不放行）。
func TestParseVerdictFallback(t *testing.T) {
	v := parseVerdict("我觉得还行吧")
	if v.Approved() || v.Verdict != "需要修改" {
		t.Errorf("解析失败应回退需要修改，得 %+v", v)
	}
}

// TestNeedsReview 写类工具过审、只读工具放行、只读 git 放行写 git 过审。
func TestNeedsReview(t *testing.T) {
	for _, n := range []string{"write_file", "edit_file", "multi_edit", "delete_file", "run_command", "git_commit", "git_checkout"} {
		if !NeedsReview(n) {
			t.Errorf("%s 应过审", n)
		}
	}
	for _, n := range []string{"read_file", "list_files", "search_content", "git_status", "git_diff", "git_log"} {
		if NeedsReview(n) {
			t.Errorf("%s 不应过审（只读）", n)
		}
	}
}
