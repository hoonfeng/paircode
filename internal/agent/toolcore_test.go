// ═══════════════════════════════════════════════════════════════
// tool_core_js_test.go — tool-core JS 原生化（ctx.fs/ctx.bash）行为验证
//
// 装载 .pair/plugins/tool-core/index.js（磁盘插件）到临时工作区，
// 验证 read_file 分页/截断/二进制保护、write_file 父目录创建、
// edit_file 精确/行号模式、multi_edit 原子、move/delete、run_command。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadToolCorePlugin 装载磁盘 tool-core 插件（JS 原生化版）。
func loadToolCorePlugin(t *testing.T, root string) (*PluginHost, *Registry) {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "..", ".pair", "plugins", "tool-core", "index.js"))
	if err != nil {
		t.Skipf("tool-core 插件不存在: %v", err)
	}
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, root)
	id, err := host.DefineJSCodeFull(string(src), "js", "tool-core 测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}
	t.Cleanup(func() { _ = host.Unload(def.name) })
	return host, reg
}

func execTool(t *testing.T, reg *Registry, name string, args map[string]any) (string, error) {
	t.Helper()
	tool, ok := reg.Get(name)
	if !ok {
		t.Fatalf("工具 %s 未注册（tool-core JS 插件未接管）", name)
	}
	return tool.Handler(nil, args)
}

func TestToolCoreJSNative(t *testing.T) {
	root := t.TempDir()
	_, reg := loadToolCorePlugin(t, root)

	// ① write_file：父目录自动创建
	_, err := execTool(t, reg, "write_file", map[string]any{
		"path": "a/b/c.txt", "content": "line1\nline2\nline3\n",
	})
	if err != nil {
		t.Fatalf("write_file 失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "b", "c.txt")); err != nil {
		t.Fatalf("文件未创建: %v", err)
	}

	// ② read_file：全文
	out, err := execTool(t, reg, "read_file", map[string]any{"path": "a/b/c.txt"})
	if err != nil {
		t.Fatalf("read_file 失败: %v", err)
	}
	if !strings.Contains(out, "line2") {
		t.Fatalf("read_file 全文异常: %q", out)
	}

	// ③ read_file：offset/limit 分页
	out, err = execTool(t, reg, "read_file", map[string]any{"path": "a/b/c.txt", "offset": 2, "limit": 1})
	if err != nil {
		t.Fatalf("read_file 分页失败: %v", err)
	}
	if out != "line2" {
		t.Fatalf("read_file 分页异常: %q", out)
	}

	// ④ read_file：二进制保护
	if err := os.WriteFile(filepath.Join(root, "bin.dat"), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := execTool(t, reg, "read_file", map[string]any{"path": "bin.dat"}); err == nil || !strings.Contains(err.Error(), "二进制") {
		t.Fatalf("二进制保护应拒绝: %v", err)
	}

	// ⑤ edit_file：精确替换
	_, err = execTool(t, reg, "edit_file", map[string]any{
		"path": "a/b/c.txt", "old_string": "line2", "new_string": "line2-edited",
	})
	if err != nil {
		t.Fatalf("edit_file 失败: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a", "b", "c.txt"))
	if !strings.Contains(string(data), "line2-edited") {
		t.Fatalf("edit_file 未生效: %q", string(data))
	}

	// ⑥ edit_file：行号定位
	_, err = execTool(t, reg, "edit_file", map[string]any{
		"path": "a/b/c.txt", "line_start": 3, "new_string": "line3-replaced",
	})
	if err != nil {
		t.Fatalf("edit_file 行号模式失败: %v", err)
	}
	data, _ = os.ReadFile(filepath.Join(root, "a", "b", "c.txt"))
	if !strings.Contains(string(data), "line3-replaced") || strings.Contains(string(data), "line3\n") {
		t.Fatalf("edit_file 行号模式未生效: %q", string(data))
	}

	// ⑦ edit_file：唯一性校验（重复 old_string 报错）
	if err := os.WriteFile(filepath.Join(root, "dup.txt"), []byte("x\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := execTool(t, reg, "edit_file", map[string]any{
		"path": "dup.txt", "old_string": "x", "new_string": "y",
	}); err == nil || !strings.Contains(err.Error(), "唯一") {
		t.Fatalf("重复 old_string 应报错: %v", err)
	}

	// ⑧ multi_edit：原子多编辑
	_, err = execTool(t, reg, "multi_edit", map[string]any{
		"path": "a/b/c.txt",
		"edits": []any{
			map[string]any{"old_string": "line2-edited", "new_string": "l2"},
			map[string]any{"old_string": "line3-replaced", "new_string": "l3"},
		},
	})
	if err != nil {
		t.Fatalf("multi_edit 失败: %v", err)
	}
	data, _ = os.ReadFile(filepath.Join(root, "a", "b", "c.txt"))
	if !strings.Contains(string(data), "l2") || !strings.Contains(string(data), "l3") {
		t.Fatalf("multi_edit 未生效: %q", string(data))
	}

	// ⑨ move_file + delete_file
	if _, err := execTool(t, reg, "move_file", map[string]any{"from": "a/b/c.txt", "to": "a/moved.txt"}); err != nil {
		t.Fatalf("move_file 失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "moved.txt")); err != nil {
		t.Fatalf("move 后目标不存在: %v", err)
	}
	if _, err := execTool(t, reg, "delete_file", map[string]any{"path": "a/moved.txt"}); err != nil {
		t.Fatalf("delete_file 失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "moved.txt")); !os.IsNotExist(err) {
		t.Fatalf("delete 后文件仍存在: %v", err)
	}

	// ⑩ run_command：ctx.bash
	out, err = execTool(t, reg, "run_command", map[string]any{"command": "echo js-native-ok"})
	if err != nil {
		t.Fatalf("run_command 失败: %v", err)
	}
	if !strings.Contains(out, "js-native-ok") {
		t.Fatalf("run_command 输出异常: %q", out)
	}
}
