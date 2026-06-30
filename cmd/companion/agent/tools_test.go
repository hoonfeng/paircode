package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolsReadWriteEditList(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// write_file（含自动建父目录）
	out, err := reg.Execute(ctx, "write_file", `{"path":"sub/a.txt","content":"hello WORLD"}`)
	if err != nil {
		t.Fatalf("write_file: %v", err)
	}
	if !strings.Contains(out, "已写入") {
		t.Errorf("write_file 返回 %q", out)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "sub", "a.txt")); string(b) != "hello WORLD" {
		t.Errorf("写入内容 = %q", b)
	}

	// read_file
	out, err = reg.Execute(ctx, "read_file", `{"path":"sub/a.txt"}`)
	if err != nil || out != "hello WORLD" {
		t.Errorf("read_file = %q, err=%v", out, err)
	}

	// edit_file（唯一替换）
	if _, err = reg.Execute(ctx, "edit_file", `{"path":"sub/a.txt","old_string":"WORLD","new_string":"GOUI"}`); err != nil {
		t.Fatalf("edit_file: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "sub", "a.txt")); string(b) != "hello GOUI" {
		t.Errorf("edit 后 = %q", b)
	}

	// edit_file：old_string 非唯一 → 报错
	os.WriteFile(filepath.Join(dir, "dup.txt"), []byte("x x x"), 0o644)
	if _, err = reg.Execute(ctx, "edit_file", `{"path":"dup.txt","old_string":"x","new_string":"y"}`); err == nil {
		t.Error("edit_file 非唯一 old_string 应报错")
	}

	// list_files
	out, err = reg.Execute(ctx, "list_files", `{}`)
	if err != nil {
		t.Fatalf("list_files: %v", err)
	}
	if !strings.Contains(out, "sub/") || !strings.Contains(out, "dup.txt") {
		t.Errorf("list_files = %q", out)
	}
	// list_files + pattern
	if out, _ = reg.Execute(ctx, "list_files", `{"pattern":"*.txt"}`); !strings.Contains(out, "dup.txt") {
		t.Errorf("pattern 过滤 = %q", out)
	}
}

func TestToolsPathTraversalBlocked(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	for _, p := range []string{"../escape.txt", "../../etc/hosts", "sub/../../out.txt"} {
		if _, err := reg.Execute(context.Background(), "read_file", `{"path":"`+p+`"}`); err == nil {
			t.Errorf("越界路径 %q 应被拒", p)
		}
	}
}

func TestToolRunCommand(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	out, err := reg.Execute(context.Background(), "run_command", `{"command":"echo CMD_OK_88"}`)
	if err != nil {
		t.Fatalf("run_command: %v", err)
	}
	if !strings.Contains(out, "CMD_OK_88") {
		t.Errorf("run_command 输出 = %q", out)
	}
}

func TestMoveAndDeleteFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewRegistry()
	RegisterDefaultTools(r, dir)
	ctx := context.Background()

	if _, err := r.Execute(ctx, "move_file", `{"from":"a.txt","to":"sub/b.txt"}`); err != nil {
		t.Fatalf("move: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
		t.Error("a.txt 应已移走")
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "b.txt")); err != nil {
		t.Errorf("sub/b.txt 应存在：%v", err)
	}

	if _, err := r.Execute(ctx, "delete_file", `{"path":"sub/b.txt"}`); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "b.txt")); !os.IsNotExist(err) {
		t.Error("b.txt 应已删除")
	}
	if _, err := r.Execute(ctx, "delete_file", `{"path":"sub"}`); err == nil {
		t.Error("delete_file 应拒绝目录")
	}
}

func TestReadFileRange(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("L1\nL2\nL3\nL4\nL5"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewRegistry()
	RegisterDefaultTools(r, dir)
	ctx := context.Background()

	out, err := r.Execute(ctx, "read_file", `{"path":"f.txt","offset":2,"limit":2}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "L2\nL3" {
		t.Errorf("片段 = %q，期望 'L2\\nL3'", out)
	}
	if full, _ := r.Execute(ctx, "read_file", `{"path":"f.txt"}`); full != "L1\nL2\nL3\nL4\nL5" {
		t.Errorf("全文 = %q", full)
	}
	if _, err := r.Execute(ctx, "read_file", `{"path":"f.txt","offset":99}`); err == nil {
		t.Error("offset 越界应报错")
	}
}

func TestMultiEdit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.go"), []byte("aaa bbb ccc"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewRegistry()
	RegisterDefaultTools(r, dir)
	ctx := context.Background()

	if _, err := r.Execute(ctx, "multi_edit", `{"path":"f.go","edits":[{"old_string":"aaa","new_string":"A"},{"old_string":"ccc","new_string":"C"}]}`); err != nil {
		t.Fatalf("multi_edit: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "f.go"))
	if string(got) != "A bbb C" {
		t.Errorf("内容 = %q，期望 'A bbb C'", string(got))
	}
	// 非唯一 old_string 应报错且不写
	os.WriteFile(filepath.Join(dir, "g.go"), []byte("x x"), 0o644)
	if _, err := r.Execute(ctx, "multi_edit", `{"path":"g.go","edits":[{"old_string":"x","new_string":"y"}]}`); err == nil {
		t.Error("不唯一 old_string 应报错")
	}
	if g, _ := os.ReadFile(filepath.Join(dir, "g.go")); string(g) != "x x" {
		t.Errorf("失败时不应写入，g.go = %q", string(g))
	}
}

func TestRegistryDefinitions(t *testing.T) {
	reg := NewRegistry()
	RegisterDefaultTools(reg, t.TempDir())
	defs := reg.Definitions()
	// 下限断言：补齐 project_info/binary/binary_re/debug 后总数远超旧硬编码 45；
	// 用下限避免每次增减工具都改测试，同时仍能捕获"注册链路整体缺失"的回归。
	if len(defs) < 50 {
		t.Fatalf("默认工具数应 >= 50（含核心/git/memory/project_info/binary/binary_re/debug 等），得 %d", len(defs))
	}
	if defs[0].Type != "function" || defs[0].Function.Name != "read_file" {
		t.Errorf("首个定义 = %+v", defs[0])
	}
	req, _ := defs[0].Function.Parameters["required"].([]string)
	if len(req) == 0 || req[0] != "path" {
		t.Errorf("read_file required = %v", req)
	}
	// 关键工具必须可见（覆盖各注册组）
	mustHave := []string{
		"read_file", "write_file", "edit_file", "multi_edit", "list_files", "run_command",
		"git_status", "memory_write", "find_files_by_pattern", "find_symbol",
		"get_file_symbols", "task_create", "go_build", "run_test", "go_run",
		"project_info_write", "project_info_read", "inspect_binary", "binary_strings",
		"debug_start", "debug_stop", "debug_status",
	}
	have := map[string]bool{}
	for _, d := range defs {
		have[d.Function.Name] = true
	}
	for _, name := range mustHave {
		if !have[name] {
			t.Errorf("默认工具 %q 未注册", name)
		}
	}
	// 未知工具 → 报错
	if _, err := reg.Execute(context.Background(), "no_such_tool", `{}`); err == nil {
		t.Error("未知工具应报错")
	}
}
