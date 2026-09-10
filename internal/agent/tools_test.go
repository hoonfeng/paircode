package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolsReadWritePatchList(t *testing.T) {
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

	// apply_patch（上下文行替换；Round5 取代 edit）
	if _, err = reg.Execute(ctx, "apply_patch", `{"patch":"*** Begin Patch\n*** Update File: sub/a.txt\n@@\n-hello WORLD\n+hello GOUI\n*** End Patch\n"}`); err != nil {
		t.Fatalf("apply_patch: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "sub", "a.txt")); string(b) != "hello GOUI" {
		t.Errorf("apply_patch 后 = %q", b)
	}

	// 供 glob 用例的文件
	os.WriteFile(filepath.Join(dir, "dup.txt"), []byte("x x x"), 0o644)

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
	out, err := reg.Execute(context.Background(), "exec_command", `{"command":"echo CMD_OK_88"}`)
	if err != nil {
		t.Fatalf("exec_command: %v", err)
	}
	if !strings.Contains(out, "CMD_OK_88") {
		t.Errorf("exec_command 输出 = %q", out)
	}
}

func TestApplyPatchMoveDelete(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewRegistry()
	RegisterDefaultTools(r, dir)
	ctx := context.Background()

	// 纯 Move（Update File + Move to，无内容变更）
	if _, err := r.Execute(ctx, "apply_patch", `{"patch":"*** Begin Patch\n*** Update File: a.txt\n*** Move to: sub/b.txt\n*** End Patch\n"}`); err != nil {
		t.Fatalf("move: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
		t.Error("a.txt 应已移走")
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "b.txt")); err != nil {
		t.Errorf("sub/b.txt 应存在：%v", err)
	}

	// Delete File
	if _, err := r.Execute(ctx, "apply_patch", `{"patch":"*** Begin Patch\n*** Delete File: sub/b.txt\n*** End Patch\n"}`); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "b.txt")); !os.IsNotExist(err) {
		t.Error("b.txt 应已删除")
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

func TestApplyPatchMultiHunk(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.go"), []byte("aaa bbb ccc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewRegistry()
	RegisterDefaultTools(r, dir)
	ctx := context.Background()

	// 一个 @@ 段整体替换（上下文行定位）
	if _, err := r.Execute(ctx, "apply_patch", `{"patch":"*** Begin Patch\n*** Update File: f.go\n@@\n-aaa bbb ccc\n+A bbb C\n*** End Patch\n"}`); err != nil {
		t.Fatalf("apply_patch: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "f.go"))
	if string(got) != "A bbb C\n" {
		t.Errorf("内容 = %q，期望 'A bbb C\\n'", string(got))
	}
	// 上下文不匹配 → 报错且不写
	if err := os.WriteFile(filepath.Join(dir, "g.go"), []byte("x x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Execute(ctx, "apply_patch", `{"patch":"*** Begin Patch\n*** Update File: g.go\n@@\n-x\n+y\n*** End Patch\n"}`); err == nil {
		t.Error("上下文不匹配应报错")
	}
	if g, _ := os.ReadFile(filepath.Join(dir, "g.go")); string(g) != "x x\n" {
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
		"read", "write", "apply_patch", "glob", "exec_command", "write_stdin", "kill_process",
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
