// ═══════════════════════════════════════════════════════════════
// tool_core_js_test.go — tool-core JS 原生化（ctx.fs）行为验证
//
// ★ 2026-08-19 工具合并：read_file→read、write_file→write、edit_file→edit、
//   run_command→bash（均在 tool-harness 插件）；multi_edit/move_file/delete_file
//   保留在 tool-core。本测试装载 tool-core + tool-harness 两插件验证合并后行为。
// ★ 2026-09 Round3 ③.4：bash 工具从工具侧移除（长进程误用风险，短查询走宿主
//   执行通道）；run_command 实现函数已从 tool-core 删除。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadToolCorePlugin 装载磁盘 tool-core + tool-harness 插件（JS 原生化版）。
func loadToolCorePlugin(t *testing.T, root string) (*PluginHost, *Registry) {
	t.Helper()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, root)
	for _, name := range []string{"tool-core", "tool-harness"} {
		src, err := os.ReadFile(filepath.Join("..", "..", ".pair", "plugins", name, "index.js"))
		if err != nil {
			t.Skipf("%s 插件不存在: %v", name, err)
		}
		id, err := host.DefineJSCodeFull(string(src), "js", name+" 测试", "", "")
		if err != nil {
			t.Fatalf("define %s 失败: %v", name, err)
		}
		def, _ := host.GetJSDef(id)
		if err := host.LoadJSDynamic(def); err != nil {
			t.Fatalf("装载 %s 失败: %v", name, err)
		}
		t.Cleanup(func(n string) func() { return func() { _ = host.Unload(n) } }(def.name))
	}
	return host, reg
}

func execTool(t *testing.T, reg *Registry, name string, args map[string]any) (string, error) {
	t.Helper()
	tool, ok := reg.Get(name)
	if !ok {
		t.Fatalf("工具 %s 未注册（JS 插件未接管）", name)
	}
	return tool.Handler(nil, args)
}

func TestToolCoreJSNative(t *testing.T) {
	root := t.TempDir()
	_, reg := loadToolCorePlugin(t, root)

	// ① write（原 write_file）：父目录自动创建
	_, err := execTool(t, reg, "write", map[string]any{
		"path": "a/b/c.txt", "content": "line1\nline2\nline3\n",
	})
	if err != nil {
		t.Fatalf("write 失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "b", "c.txt")); err != nil {
		t.Fatalf("文件未创建: %v", err)
	}

	// ② read（原 read_file）：全文
	out, err := execTool(t, reg, "read", map[string]any{"path": "a/b/c.txt"})
	if err != nil {
		t.Fatalf("read 失败: %v", err)
	}
	if !strings.Contains(out, "line2") {
		t.Fatalf("read 全文异常: %q", out)
	}

	// ③ read：offset/limit 分页
	out, err = execTool(t, reg, "read", map[string]any{"path": "a/b/c.txt", "offset": 2, "limit": 1})
	if err != nil {
		t.Fatalf("read 分页失败: %v", err)
	}
	if !strings.Contains(out, "line2") {
		t.Fatalf("read 分页异常: %q", out)
	}

	// ④ read：二进制保护
	if err := os.WriteFile(filepath.Join(root, "bin.dat"), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := execTool(t, reg, "read", map[string]any{"path": "bin.dat"}); err == nil || !strings.Contains(err.Error(), "二进制") {
		t.Fatalf("二进制保护应拒绝: %v", err)
	}

	// ⑤ edit（原 edit_file）：精确替换
	_, err = execTool(t, reg, "edit", map[string]any{
		"path": "a/b/c.txt", "old_string": "line2", "new_string": "line2-edited",
	})
	if err != nil {
		t.Fatalf("edit 失败: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a", "b", "c.txt"))
	if !strings.Contains(string(data), "line2-edited") {
		t.Fatalf("edit 未生效: %q", string(data))
	}

	// ⑥ edit：唯一性校验（重复 old_string 报错）
	if err := os.WriteFile(filepath.Join(root, "dup.txt"), []byte("x\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := execTool(t, reg, "edit", map[string]any{
		"path": "dup.txt", "old_string": "x", "new_string": "y",
	}); err == nil || !strings.Contains(err.Error(), "唯一") {
		t.Fatalf("重复 old_string 应报错: %v", err)
	}

	// ⑦ multi_edit：原子多编辑
	_, err = execTool(t, reg, "multi_edit", map[string]any{
		"path": "a/b/c.txt",
		"edits": []any{
			map[string]any{"old_string": "line2-edited", "new_string": "l2"},
			map[string]any{"old_string": "line3", "new_string": "l3"},
		},
	})
	if err != nil {
		t.Fatalf("multi_edit 失败: %v", err)
	}
	data, _ = os.ReadFile(filepath.Join(root, "a", "b", "c.txt"))
	if !strings.Contains(string(data), "l2") || !strings.Contains(string(data), "l3") {
		t.Fatalf("multi_edit 未生效: %q", string(data))
	}

	// ⑧ move_file + delete_file
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
	// ⑨（已删除，2026-09 ③.4）：原 bash（run_command）工具已从工具侧移除——
	//    短查询走宿主执行通道；长进程统一 run_background/read_output/kill_process
	//    （行为验证见 harness_js_test.go ⑥ 后台进程段）
}
