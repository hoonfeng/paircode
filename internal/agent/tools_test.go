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

	// write（含自动建父目录）
	out, err := reg.Execute(ctx, "write", `{"path":"sub/a.txt","content":"hello WORLD"}`)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(out, "已写入") {
		t.Errorf("write 返回 %q", out)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "sub", "a.txt")); string(b) != "hello WORLD" {
		t.Errorf("写入内容 = %q", b)
	}

	// read
	out, err = reg.Execute(ctx, "read", `{"path":"sub/a.txt"}`)
	if err != nil || out != "hello WORLD" {
		t.Errorf("read = %q, err=%v", out, err)
	}

	// edit（唯一替换）
	if _, err = reg.Execute(ctx, "edit", `{"path":"sub/a.txt","old_string":"WORLD","new_string":"GOUI"}`); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "sub", "a.txt")); string(b) != "hello GOUI" {
		t.Errorf("edit 后 = %q", b)
	}

	// edit：old_string 非唯一 → 报错
	os.WriteFile(filepath.Join(dir, "dup.txt"), []byte("x x x"), 0o644)
	if _, err = reg.Execute(ctx, "edit", `{"path":"dup.txt","old_string":"x","new_string":"y"}`); err == nil {
		t.Error("edit 非唯一 old_string 应报错")
	}

	// glob
	out, err = reg.Execute(ctx, "glob", `{}`)
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if !strings.Contains(out, "sub/") || !strings.Contains(out, "dup.txt") {
		t.Errorf("glob = %q", out)
	}
	// glob + pattern
	if out, _ = reg.Execute(ctx, "glob", `{"pattern":"*.txt"}`); !strings.Contains(out, "dup.txt") {
		t.Errorf("pattern 过滤 = %q", out)
	}
}

func TestToolsPathTraversalBlocked(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	for _, p := range []string{"../escape.txt", "../../etc/hosts", "sub/../../out.txt"} {
		if _, err := reg.Execute(context.Background(), "read", `{"path":"`+p+`"}`); err == nil {
			t.Errorf("越界路径 %q 应被拒", p)
		}
	}
}

func TestToolRunCommand(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	out, err := reg.Execute(context.Background(), "bash", `{"command":"echo CMD_OK_88"}`)
	if err != nil {
		t.Fatalf("bash: %v", err)
	}
	if !strings.Contains(out, "CMD_OK_88") {
		t.Errorf("bash 输出 = %q", out)
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

	out, err := r.Execute(ctx, "read", `{"path":"f.txt","offset":2,"limit":2}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "L2\nL3" {
		t.Errorf("片段 = %q，期望 'L2\\nL3'", out)
	}
	if full, _ := r.Execute(ctx, "read", `{"path":"f.txt"}`); full != "L1\nL2\nL3\nL4\nL5" {
		t.Errorf("全文 = %q", full)
	}
	if _, err := r.Execute(ctx, "read", `{"path":"f.txt","offset":99}`); err == nil {
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
	// ★ 2026-08-17 对齐 harness orderTools：Definitions 按 name 字典序返回
	//   （不再按注册顺序）——首元素是字典序最小的工具，跨装配时序稳定。
	if defs[0].Type != "function" {
		t.Errorf("首个定义 type = %+v", defs[0].Type)
	}
	// 验证字典序（code-unit）：任意相邻两项前项 <= 后项
	for i := 1; i < len(defs); i++ {
		if defs[i-1].Function.Name > defs[i].Function.Name {
			t.Fatalf("Definitions 未按 name 字典序排序：defs[%d]=%q > defs[%d]=%q",
				i-1, defs[i-1].Function.Name, i, defs[i].Function.Name)
		}
	}
	// read 仍在列表中且 required 参数正确（不再假设位置）
	found := false
	for _, d := range defs {
		if d.Function.Name == "read" {
			found = true
			req, _ := d.Function.Parameters["required"].([]string)
			if len(req) == 0 || req[0] != "path" {
				t.Errorf("read required = %v", req)
			}
			break
		}
	}
	if !found {
		t.Error("read 未出现在 Definitions 中")
	}
	// 关键工具必须可见（覆盖各注册组）
	// ★ 注：find_symbol/get_file_symbols → codegraph_search/codegraph_file_structure；
	//   find_files_by_pattern → glob（增加 language 参数）；
	//   task_create → update_tasks。均已合并/更名，这里断言替代后的工具。
	mustHave := []string{
		"read", "write", "edit", "multi_edit", "glob", "bash",
		"git_status", "memory_write", "glob", "grep",
		"update_tasks", "codegraph_search", "codegraph_file_structure",
		"project_info_write", "project_info_read", "inspect_binary", "binary_strings",
		"debug_inject_log", "debug_run_capture", "debug_evaluate_session",
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
