package agent

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
)

// TestReviewUserPromptPathAliases 送审路径归一化：现代工具的路径参数名
// （file_path / target / file / files[] / 补丁文件头）都要能出现在送审 prompt 里
// ——否则审核器看到「文件：」为空，只能保守驳回。
func TestReviewUserPromptPathAliases(t *testing.T) {
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"write(file_path)", map[string]any{"file_path": "internal/a.go", "content": "x"}, "文件：internal/a.go"},
		{"write(path)", map[string]any{"path": "internal/b.go", "content": "x"}, "文件：internal/b.go"},
		{"apply_patch", map[string]any{"patch": "*** Begin Patch\n*** Update File: internal/c.go\n@@\n-a\n+b\n*** End Patch"}, "文件：internal/c.go"},
		{"git_add", map[string]any{"files": []any{"a.go", "b.go"}}, "文件：a.go, b.go"},
		{"git_checkout", map[string]any{"target": "main"}, "文件：main"},
	}
	for _, c := range cases {
		got := reviewUserPrompt(c.name, c.args)
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: 送审 prompt 应含 %q，得:\n%s", c.name, c.want, got)
		}
	}
}

// TestReviewUserPromptContentAliases 送审内容归一化：new_string 才是「将要写入的新代码」，
// 不得把 old_string（旧代码）当内容送审；edits[] 拼接各段新代码。
func TestReviewUserPromptContentAliases(t *testing.T) {
	got := reviewUserPrompt("edit", map[string]any{"file_path": "a.go", "old_string": "旧代码", "new_string": "新代码"})
	if !strings.Contains(got, "新代码") {
		t.Errorf("应送审 new_string 新代码，得:\n%s", got)
	}
	if strings.Contains(got, "旧代码") {
		t.Errorf("不应把 old_string 旧代码送审，得:\n%s", got)
	}
	multi := reviewUserPrompt("multi_edit", map[string]any{"edits": []any{
		map[string]any{"old_string": "o1", "new_string": "n1"},
		map[string]any{"old_string": "o2", "new_string": "n2"},
	}})
	if !strings.Contains(multi, "#edit1:\nn1") || !strings.Contains(multi, "#edit2:\nn2") {
		t.Errorf("edits[] 应拼接各段新代码，得:\n%s", multi)
	}
}

// TestReviewUserPromptGitTools git 类工具送审：参数是 message/all/project（无 path/content），
// 必须用 project / 仓库根兜底 path 并汇总关键参数，否则审核器「路径空 + 内容空」必驳回。
func TestReviewUserPromptGitTools(t *testing.T) {
	got := reviewUserPrompt("git_commit", map[string]any{"message": "修复登录超时", "all": true, "project": "ref"})
	for _, want := range []string{"文件：（项目）ref", "操作：Git 操作", "message=修复登录超时", "all=true", "project=ref"} {
		if !strings.Contains(got, want) {
			t.Errorf("git_commit 送审应含 %q，得:\n%s", want, got)
		}
	}
	// 无 project → 仓库工作区兜底
	plain := reviewUserPrompt("git_stash", map[string]any{"action": "pop"})
	if !strings.Contains(plain, "文件：（git 仓库工作区）") || !strings.Contains(plain, "action=pop") {
		t.Errorf("git_stash 应兜底路径并汇总参数，得:\n%s", plain)
	}
}

// TestReviewUserPromptShellCommand Shell 命令仍走专用分支（不因归一化而改变）。
func TestReviewUserPromptShellCommand(t *testing.T) {
	got := reviewUserPrompt("exec_command", map[string]any{"command": "go build ./..."})
	if !strings.Contains(got, "[审核：Shell 命令]") || !strings.Contains(got, "go build ./...") {
		t.Errorf("Shell 分支异常，得:\n%s", got)
	}
}

// reviewCaptureProvider 记录送审请求消息的审核 Provider（断言送审 prompt 用）。
type reviewCaptureProvider struct{ msgs []Message }

func (c *reviewCaptureProvider) Name() string { return "review-capture" }

func (c *reviewCaptureProvider) Chat(_ context.Context, messages []Message, _ []ToolDefinition, _ func(Chunk)) (Message, error) {
	c.msgs = append(c.msgs, messages...)
	return Message{Role: RoleAssistant, Content: `{"verdict":"通过","confidence":1,"summary":"无风险","suggestions":[]}`}, nil
}

// TestResolveApprovalGitCommitSendPrompt 集成（审核门链路）：ReviewMode=auto 下
// git_commit 走 AI 审核（NeedsReview 对 git_* 为真），送审 prompt 必须带提交信息与
// 目标项目——修复前 path/content 双空 → 审核模型看不到内容只能驳回（实测 100% 驳回）。
func TestResolveApprovalGitCommitSendPrompt(t *testing.T) {
	cap := &reviewCaptureProvider{}
	reg := NewRegistry()
	var executed atomic.Int32
	reg.Register(&Tool{
		Name: "git_commit", Description: "提交", RequiresApproval: true,
		Parameters: objSchema(props{"message": strProp("提交信息"), "all": boolProp("先 -a"), "project": projectSchemaProp()}, "message"),
		Handler: func(context.Context, map[string]any) (string, error) {
			executed.Add(1)
			return "committed", nil
		},
	})
	loop := &Loop{Registry: reg, ReviewProvider: cap, ReviewMode: "auto"}
	tc := ToolCall{ID: "c1", Type: "function",
		Function: FunctionCall{Name: "git_commit", Arguments: `{"message":"修复登录超时","all":true,"project":"ref"}`}}

	approved, feedback := loop.resolveApproval(context.Background(), tc)
	if !approved || feedback != "" {
		t.Fatalf("审核通过时 resolveApproval 应返回 (true, \"\")，得 (%v, %q)", approved, feedback)
	}
	var sent string
	for _, m := range cap.msgs {
		if m.Role == RoleUser {
			sent = m.Content
		}
	}
	if sent == "" {
		t.Fatal("审核 Provider 未收到送审请求")
	}
	for _, want := range []string{"文件：（项目）ref", "操作：Git 操作", "message=修复登录超时", "all=true"} {
		if !strings.Contains(sent, want) {
			t.Errorf("送审 prompt 应含 %q，得:\n%s", want, sent)
		}
	}
	if executed.Load() != 0 {
		t.Errorf("resolveApproval 只做审核判定，不应执行工具，执行次数=%d", executed.Load())
	}
}
