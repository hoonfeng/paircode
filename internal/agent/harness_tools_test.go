package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ─── harness 基座注册 ────────────────────────────────────────

// TestRegisterHarnessTools_Base 验证 harness 命名基座工具已以新名注册
// （Round3 别名层删除：read/write/edit/bash/glob/grep 直接由 registerCoreTools
// 注册，RegisterHarnessTools 仅补 run_code（★ Round4：str_replace_editor 已删）。
func TestRegisterHarnessTools_Base(t *testing.T) {
	root := t.TempDir()
	r := NewRegistry()
	RegisterDefaultTools(r, root)
	RegisterHarnessTools(r, root)

	// 基座工具直接存在（不再依赖旧名 + 别名层）
	base := map[string]bool{
		"read": true, "write": true, "edit": true,
		"glob": true, "grep": true, "bash": true,
	}
	for name := range base {
		tool, ok := r.Get(name)
		if !ok {
			t.Fatalf("基座工具 %q 未注册", name)
		}
		if tool.Handler == nil {
			t.Fatalf("%q handler 为空", name)
		}
	}
	// 旧名注册面已删除（零旧名）
	for _, old := range []string{"read_file", "write_file", "edit_file", "list_files", "run_command", "search_content", "search_files"} {
		if _, ok := r.Get(old); ok {
			t.Errorf("旧名 %q 不应再注册（Round3 清理）", old)
		}
	}
	// 只读/审批语义
	if ro, _ := r.Get("read"); !ro.ReadOnly {
		t.Error("read 应只读")
	}
	if wa, _ := r.Get("write"); !wa.RequiresApproval {
		t.Error("write 应需审批")
	}
	// web 工具已有同名，不重复注册
	if _, ok := r.Get("web_search"); !ok {
		t.Error("web_search 应已存在")
	}
	if _, ok := r.Get("run_code"); !ok {
		t.Error("run_code 未注册")
	}
}

// TestHarnessAlias_ReadWriteEdit 验证别名工具实际可用（读写编辑闭环）。
func TestHarnessAlias_ReadWriteEdit(t *testing.T) {
	root := t.TempDir()
	r := NewRegistry()
	RegisterDefaultTools(r, root)
	RegisterHarnessTools(r, root)

	ctx := context.Background()
	// write 创建文件
	out, err := r.Execute(ctx, "write", `{"path":"a.txt","content":"line1\nline2\nline3\n"}`)
	if err != nil {
		t.Fatalf("write 失败: %v", err)
	}
	if !strings.Contains(out, "a.txt") {
		t.Errorf("write 输出异常: %s", out)
	}
	// read 读回
	out, err = r.Execute(ctx, "read", `{"path":"a.txt"}`)
	if err != nil {
		t.Fatalf("read 失败: %v", err)
	}
	if !strings.Contains(out, "line1") || !strings.Contains(out, "line2") {
		t.Errorf("read 输出异常: %s", out)
	}
	// edit 替换
	out, err = r.Execute(ctx, "edit", `{"path":"a.txt","old_string":"line2","new_string":"LINE2"}`)
	if err != nil {
		t.Fatalf("edit 失败: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if !strings.Contains(string(data), "LINE2") {
		t.Errorf("edit 未生效: %s", string(data))
	}
}

// TestHarnessAlias_GlobGrepBash 验证 glob/grep/bash 别名。
func TestHarnessAlias_GlobGrepBash(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "x.go"), []byte("package main\nfunc hello() {}\n"), 0o644)
	os.WriteFile(filepath.Join(root, "y.py"), []byte("def world():\n    pass\n"), 0o644)

	r := NewRegistry()
	RegisterDefaultTools(r, root)
	RegisterHarnessTools(r, root)
	ctx := context.Background()

	out, err := r.Execute(ctx, "glob", `{"pattern":"*.go"}`)
	if err != nil {
		t.Fatalf("glob 失败: %v", err)
	}
	if !strings.Contains(out, "x.go") {
		t.Errorf("glob 未找到 x.go: %s", out)
	}
	if strings.Contains(out, "y.py") {
		t.Errorf("glob 误匹配 y.py: %s", out)
	}

	out, err = r.Execute(ctx, "grep", `{"pattern":"hello"}`)
	if err != nil {
		t.Fatalf("grep 失败: %v", err)
	}
	if !strings.Contains(out, "x.go") {
		t.Errorf("grep 未命中 x.go: %s", out)
	}

	out, err = r.Execute(ctx, "bash", `{"command":"echo BASH_ALIAS_OK"}`)
	if err != nil {
		t.Fatalf("bash 失败: %v", err)
	}
	if !strings.Contains(out, "BASH_ALIAS_OK") {
		t.Errorf("bash 输出异常: %s", out)
	}
}

// ─── run_code ────────────────────────────────────────────────
func TestRunCode_DetectLang(t *testing.T) {
	cases := map[string]string{
		"package main\nfunc main() {}":       "go",
		"def f():\n    return 1\nprint(f())": "python",
		"console.log('hi')":                  "node",
		"import os\nprint(os.getcwd())":      "python",
	}
	for code, want := range cases {
		if got := detectCodeLang(code); got != want {
			t.Errorf("detectCodeLang(%q)=%q 期望 %q", code, got, want)
		}
	}
}

// TestRunCode_Go 验证 run_code 执行 Go 代码（若本机有 go 环境）。
func TestRunCode_Go(t *testing.T) {
	if !hasGoEnv() {
		t.Skip("本机无 go 环境，跳过")
	}
	root := t.TempDir()
	r := NewRegistry()
	RegisterDefaultTools(r, root)
	RegisterHarnessTools(r, root)
	ctx := context.Background()

	out, err := r.Execute(ctx, "run_code", `{"language":"go","code":"package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"RUN_CODE_OK\") }"}`)
	if err != nil {
		t.Fatalf("run_code go 失败: %v", err)
	}
	if !strings.Contains(out, "RUN_CODE_OK") {
		t.Errorf("run_code 输出异常: %s", out)
	}
}

func hasGoEnv() bool {
	_, err := exec.LookPath("go")
	return err == nil
}

// ★ 2026-08-19 工具成功率改进验证：
// view_range 超界容错（不报错，自动 clamp）+ str_replace 诊断增强（相似行）。

// TestStrReplaceEditor_ViewRangeClamp view_range 次元素超界 → 自动截断不报错。
func TestUnknownToolFriendly(t *testing.T) {
	r := NewRegistry()
	_, err := r.Execute(context.Background(), "no_such_tool_xyz", `{}`)
	if err == nil || !strings.Contains(err.Error(), "工具集") {
		t.Errorf("未知工具错误应提示工具集，got err=%v", err)
	}
}

// TestGlobListNotFound glob 目录列举模式（无 pattern）目录不存在 → 明确提示。
func TestGlobListNotFound(t *testing.T) {
	root := t.TempDir()
	r := NewRegistry()
	RegisterDefaultTools(r, root)
	_, err := r.Execute(context.Background(), "glob", `{"path":"no_such_dir_abc"}`)
	if err == nil || !strings.Contains(err.Error(), "目录不存在") {
		t.Errorf("glob 目录不存在应明确提示，got err=%v", err)
	}
}
