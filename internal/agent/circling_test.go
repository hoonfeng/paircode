package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestDetectCircling 反复失败 / 反复执行 → 提示；不同操作不提示；超窗口裁剪。
func TestDetectCircling(t *testing.T) {
	// 同一操作失败 2 次 → 提示
	l := &Loop{}
	l.trackCall("edit_file", `{"path":"x.go"}`, true)
	l.trackCall("edit_file", `{"path":"x.go"}`, true)
	if n := l.detectCircling(); n == "" || !strings.Contains(n, "失败") {
		t.Errorf("同一操作失败 2 次应提示，得 %q", n)
	}

	// 同一操作重复 3 次（即便成功）→ 提示
	l2 := &Loop{}
	for i := 0; i < 3; i++ {
		l2.trackCall("read_file", `{"path":"y.go"}`, false)
	}
	if n := l2.detectCircling(); n == "" || !strings.Contains(n, "重复") {
		t.Errorf("同一操作重复 3 次应提示，得 %q", n)
	}

	// 不同操作 → 不提示
	l3 := &Loop{}
	l3.trackCall("read_file", `{"path":"a.go"}`, false)
	l3.trackCall("edit_file", `{"path":"b.go"}`, false)
	if l3.detectCircling() != "" {
		t.Error("不同操作不应提示绕圈")
	}

	// 超窗口裁剪
	l4 := &Loop{}
	for i := 0; i < 20; i++ {
		l4.trackCall("read_file", `{"path":"z`+strconv.Itoa(i)+`.go"}`, false)
	}
	if len(l4.recentCalls) > circlingWindow {
		t.Errorf("应只保留窗口 %d 内，得 %d", circlingWindow, len(l4.recentCalls))
	}
}

// TestLoopRunBreaksCircling 端到端：工具反复失败 → loop 注入绕圈提示（EventCircling）。
func TestLoopRunBreaksCircling(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{Name: "boom", Description: "总是失败", ReadOnly: true, Parameters: objSchema(props{}),
		Handler: func(context.Context, map[string]any) (string, error) { return "", fmt.Errorf("总是失败") }})
	var resp []Message
	for i := 0; i < 5; i++ {
		resp = append(resp, Message{ToolCalls: []ToolCall{{ID: "c", Type: "function", Function: FunctionCall{Name: "boom", Arguments: "{}"}}}})
	}
	var circled int
	l := &Loop{Provider: &MockProvider{Responses: resp}, Registry: reg, MaxIterations: 10,
		OnEvent: func(e Event) {
			if e.Type == EventCircling {
				circled++
			}
		}}
	l.Run(context.Background(), "干活", nil)
	if circled == 0 {
		t.Error("反复失败应触发绕圈提示（EventCircling）")
	}
}

// TestRecallMemories 自动召回相关项目记忆（关键词/CJK 二元组打分），无关的不召回。
func TestRecallMemories(t *testing.T) {
	root := t.TempDir()
	dir := memoryDir(root)
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "config-refactor.md"),
		[]byte("---\nname: config-refactor\ndescription: 配置模块重构教训\n---\n重构配置时注意向后兼容"), 0o644)
	os.WriteFile(filepath.Join(dir, "unrelated.md"),
		[]byte("---\nname: unrelated\ndescription: 无关\n---\n数据库连接池调优"), 0o644)

	out := RecallMemories(root, "帮我重构配置模块", 3)
	if !strings.Contains(out, "config-refactor") {
		t.Errorf("应召回相关记忆，得 %q", out)
	}
	if strings.Contains(out, "unrelated") {
		t.Errorf("不应召回无关记忆，得 %q", out)
	}
	if got := RecallMemories(root, "画一只小猫散步", 3); got != "" {
		t.Errorf("无匹配应返回空，得 %q", got)
	}
}
