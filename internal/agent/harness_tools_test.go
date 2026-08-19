package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ─── harness 别名注册 ────────────────────────────────────────

// TestRegisterHarnessTools_Aliases 验证 harness 别名工具注册且复用旧 handler。
func TestRegisterHarnessTools_Aliases(t *testing.T) {
	root := t.TempDir()
	r := NewRegistry()
	RegisterDefaultTools(r, root)
	RegisterHarnessTools(r, root)

	pairs := [][2]string{
		{"read", "read_file"},
		{"write", "write_file"},
		{"edit", "edit_file"},
		{"glob", "search_files"},
		{"grep", "search_content"},
		{"bash", "run_command"},
	}
	for _, p := range pairs {
		alias, ok := r.Get(p[0])
		if !ok {
			t.Fatalf("harness 别名 %q 未注册", p[0])
		}
		src, ok := r.Get(p[1])
		if !ok {
			t.Fatalf("旧工具 %q 未注册", p[1])
		}
		if alias.Handler == nil || src.Handler == nil {
			t.Fatalf("%q/%q handler 为空", p[0], p[1])
		}
		if alias.ReadOnly != src.ReadOnly {
			t.Errorf("%q ReadOnly=%v 与 %q 不一致（%v）", p[0], alias.ReadOnly, p[1], src.ReadOnly)
		}
		if alias.RequiresApproval != src.RequiresApproval {
			t.Errorf("%q RequiresApproval=%v 与 %q 不一致（%v）", p[0], alias.RequiresApproval, p[1], src.RequiresApproval)
		}
	}
	// web 工具已有同名，不重复注册
	if _, ok := r.Get("web_search"); !ok {
		t.Error("web_search 应已存在")
	}
	if _, ok := r.Get("str_replace_editor"); !ok {
		t.Error("str_replace_editor 未注册")
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

// ─── str_replace_editor ──────────────────────────────────────

func setupSRE(t *testing.T) (string, *Registry) {
	t.Helper()
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "doc.txt"),
		[]byte("alpha\nbeta\ngamma\nbeta\n"), 0o644)
	r := NewRegistry()
	RegisterDefaultTools(r, root)
	RegisterHarnessTools(r, root)
	return root, r
}

// TestStrReplaceEditor_View 验证 view 命令（文件带行号 + 目录列 2 层）。
func TestStrReplaceEditor_View(t *testing.T) {
	root, r := setupSRE(t)
	ctx := context.Background()

	out, err := r.Execute(ctx, "str_replace_editor", `{"command":"view","path":"doc.txt"}`)
	if err != nil {
		t.Fatalf("view 文件失败: %v", err)
	}
	for _, want := range []string{"1  alpha", "2  beta", "4  beta"} {
		if !strings.Contains(out, want) {
			t.Errorf("view 缺少行号 %q:\n%s", want, out)
		}
	}

	// view_range
	out, err = r.Execute(ctx, "str_replace_editor", `{"command":"view","path":"doc.txt","view_range":[2,3]}`)
	if err != nil {
		t.Fatalf("view range 失败: %v", err)
	}
	if strings.Contains(out, "alpha") || !strings.Contains(out, "2  beta") {
		t.Errorf("view_range 未生效:\n%s", out)
	}

	// 目录
	out, err = r.Execute(ctx, "str_replace_editor", `{"command":"view","path":"."}`)
	if err != nil {
		t.Fatalf("view 目录失败: %v", err)
	}
	if !strings.Contains(out, "doc.txt") {
		t.Errorf("view 目录未列出 doc.txt:\n%s", out)
	}
	_ = root
}

// TestStrReplaceEditor_Create 验证 create 命令（新建 + 已存在拒绝）。
func TestStrReplaceEditor_Create(t *testing.T) {
	root, r := setupSRE(t)
	ctx := context.Background()

	out, err := r.Execute(ctx, "str_replace_editor", `{"command":"create","path":"new.txt","file_text":"hello world"}`)
	if err != nil {
		t.Fatalf("create 失败: %v", err)
	}
	if !strings.Contains(out, "new.txt") {
		t.Errorf("create 输出异常: %s", out)
	}
	data, _ := os.ReadFile(filepath.Join(root, "new.txt"))
	if string(data) != "hello world" {
		t.Errorf("create 内容异常: %q", string(data))
	}

	// 已存在 → 拒绝
	_, err = r.Execute(ctx, "str_replace_editor", `{"command":"create","path":"doc.txt","file_text":"x"}`)
	if err == nil || !strings.Contains(err.Error(), "已存在") {
		t.Errorf("create 已存在文件应报错，got err=%v", err)
	}
}

// TestStrReplaceEditor_StrReplace 验证 str_replace（唯一替换 + 不唯一拒绝 + 未找到）。
func TestStrReplaceEditor_StrReplace(t *testing.T) {
	_, r := setupSRE(t)
	ctx := context.Background()

	// 唯一替换（"alpha" 只出现一次）
	out, err := r.Execute(ctx, "str_replace_editor", `{"command":"str_replace","path":"doc.txt","old_str":"alpha","new_str":"ALPHA"}`)
	if err != nil {
		t.Fatalf("str_replace 失败: %v", err)
	}
	if !strings.Contains(out, "已编辑") {
		t.Errorf("str_replace 输出异常: %s", out)
	}

	// "beta" 出现两次 → 不唯一拒绝
	_, err = r.Execute(ctx, "str_replace_editor", `{"command":"str_replace","path":"doc.txt","old_str":"beta","new_str":"BETA"}`)
	if err == nil || !strings.Contains(err.Error(), "不唯一") {
		t.Errorf("str_replace 不唯一应报错，got err=%v", err)
	}

	// 未找到
	_, err = r.Execute(ctx, "str_replace_editor", `{"command":"str_replace","path":"doc.txt","old_str":"nope","new_str":"x"}`)
	if err == nil || !strings.Contains(err.Error(), "未找到") {
		t.Errorf("str_replace 未找到应报错，got err=%v", err)
	}
}

// TestStrReplaceEditor_Insert 验证 insert 命令（行后插入）。
func TestStrReplaceEditor_Insert(t *testing.T) {
	root, r := setupSRE(t)
	ctx := context.Background()

	out, err := r.Execute(ctx, "str_replace_editor", `{"command":"insert","path":"doc.txt","insert_line":2,"new_str":"INSERTED"}`)
	if err != nil {
		t.Fatalf("insert 失败: %v", err)
	}
	if !strings.Contains(out, "已编辑") {
		t.Errorf("insert 输出异常: %s", out)
	}
	data, _ := os.ReadFile(filepath.Join(root, "doc.txt"))
	lines := strings.Split(string(data), "\n")
	// 第 2 行后插入 → 原 beta 之后应为 INSERTED
	if lines[2] != "INSERTED" {
		t.Errorf("insert 位置错误，第3行=%q（期望 INSERTED）:\n%s", lines[2], string(data))
	}
}

// ─── run_code ────────────────────────────────────────────────

// TestRunCode_DetectLang 验证语言探测。
func TestRunCode_DetectLang(t *testing.T) {
	cases := map[string]string{
		"package main\nfunc main() {}":          "go",
		"def f():\n    return 1\nprint(f())":    "python",
		"console.log('hi')":                     "node",
		"import os\nprint(os.getcwd())":          "python",
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
func TestStrReplaceEditor_ViewRangeClamp(t *testing.T) {
	_, r := setupSRE(t)
	ctx := context.Background()

	// 次元素远超文件行数 → 截断到文件尾 + 提示
	out, err := r.Execute(ctx, "str_replace_editor", `{"command":"view","path":"doc.txt","view_range":[1,999]}`)
	if err != nil {
		t.Fatalf("view_range 超界应容错（不报错）: %v", err)
	}
	for _, want := range []string{"1  alpha", "4  beta", "自动修正"} {
		if !strings.Contains(out, want) {
			t.Errorf("截断输出缺少 %q:\n%s", want, out)
		}
	}

	// 首元素超界 → 显示最后一行
	out, err = r.Execute(ctx, "str_replace_editor", `{"command":"view","path":"doc.txt","view_range":[999,1000]}`)
	if err != nil {
		t.Fatalf("view_range 首元素超界应容错: %v", err)
	}
	if !strings.Contains(out, "4  beta") {
		t.Errorf("首元素超界应显示最后一行:\n%s", out)
	}
}

// TestStrReplaceEditor_StrReplaceDiagnose str_replace 未找到 → 诊断含相似行与建议。
func TestStrReplaceEditor_StrReplaceDiagnose(t *testing.T) {
	_, r := setupSRE(t)
	ctx := context.Background()

	// 未找到但 old_str 关键词在文件中（beta 存在于 doc.txt）
	_, err := r.Execute(ctx, "str_replace_editor", `{"command":"str_replace","path":"doc.txt","old_str":"betta","new_str":"x"}`)
	if err == nil {
		t.Fatal("str_replace 应报错")
	}
	if !strings.Contains(err.Error(), "相似行") {
		t.Errorf("诊断应含相似行提示，got: %.200s", err.Error())
	}
	if !strings.Contains(err.Error(), "建议") {
		t.Errorf("诊断应含恢复建议，got: %.200s", err.Error())
	}
}

// TestUnknownToolFriendly 未知工具错误提示工具集/恢复路径。
func TestUnknownToolFriendly(t *testing.T) {
	r := NewRegistry()
	_, err := r.Execute(context.Background(), "no_such_tool_xyz", `{}`)
	if err == nil || !strings.Contains(err.Error(), "工具集") {
		t.Errorf("未知工具错误应提示工具集，got err=%v", err)
	}
}

// TestListFilesNotFound list_files 目录不存在 → 明确提示。
func TestListFilesNotFound(t *testing.T) {
	root := t.TempDir()
	r := NewRegistry()
	RegisterDefaultTools(r, root)
	_, err := r.Execute(context.Background(), "list_files", `{"path":"no_such_dir_abc"}`)
	if err == nil || !strings.Contains(err.Error(), "目录不存在") {
		t.Errorf("list_files 目录不存在应明确提示，got err=%v", err)
	}
}
