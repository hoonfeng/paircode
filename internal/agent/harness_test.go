// ═══════════════════════════════════════════════════════════════
// harness_js_test.go — tool-harness JS 原生化行为验证
//
// 装载 .pair/plugins/tool-harness/index.js（磁盘插件）到临时工作区，
// 验证 read/write/edit/glob/grep/bash/str_replace_editor 的 JS 实现
// （ctx.fs/ctx.bash）；run_code 保持 hostTool（宿主执行器）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadHarnessPlugin 装载磁盘 tool-harness 插件（JS 原生化版）。
func loadHarnessPlugin(t *testing.T, root string) (*PluginHost, *Registry) {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "..", ".pair", "plugins", "tool-harness", "index.js"))
	if err != nil {
		t.Skipf("tool-harness 插件不存在: %v", err)
	}
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, root)
	RegisterBuiltinPlugins(host)
	id, err := host.DefineJSCodeFull(string(src), "js", "tool-harness 测试",
		filepath.Join("..", "..", ".pair", "plugins", "tool-harness"), "")
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

func TestToolHarnessJSNative(t *testing.T) {
	root := t.TempDir()
	_, reg := loadHarnessPlugin(t, root)

	// ① write：父目录创建
	if _, err := execTool(t, reg, "write", map[string]any{"path": "x/y.txt", "content": "a\nb\nc\n"}); err != nil {
		t.Fatalf("write 失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "x", "y.txt")); err != nil {
		t.Fatalf("write 未创建: %v", err)
	}

	// ② read：全文 + 分页
	out, _ := execTool(t, reg, "read", map[string]any{"path": "x/y.txt"})
	if !strings.Contains(out, "b") {
		t.Fatalf("read 全文异常: %q", out)
	}
	out, _ = execTool(t, reg, "read", map[string]any{"path": "x/y.txt", "offset": 2, "limit": 1})
	if out != "b" {
		t.Fatalf("read 分页异常: %q", out)
	}

	// ③ edit：精确替换 + 唯一性
	if _, err := execTool(t, reg, "edit", map[string]any{"path": "x/y.txt", "old_string": "b", "new_string": "b2"}); err != nil {
		t.Fatalf("edit 失败: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "x", "y.txt"))
	if !strings.Contains(string(data), "b2") {
		t.Fatalf("edit 未生效: %q", string(data))
	}

	// ④ glob：找到文件（相对路径）
	out, err := execTool(t, reg, "glob", map[string]any{"pattern": "y.txt"})
	if err != nil || !strings.Contains(out, "x/y.txt") {
		t.Fatalf("glob 异常: %q err=%v", out, err)
	}

	// ⑤ grep：内容命中
	if err := os.WriteFile(filepath.Join(root, "x", "g.txt"), []byte("needle-here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = execTool(t, reg, "grep", map[string]any{"pattern": "needle"})
	if err != nil || !strings.Contains(out, "g.txt:1") {
		t.Fatalf("grep 异常: %q err=%v", out, err)
	}

	// ⑥ bash：ctx.bash 执行
	out, err = execTool(t, reg, "bash", map[string]any{"command": "echo harness-js-ok"})
	if err != nil || !strings.Contains(out, "harness-js-ok") {
		t.Fatalf("bash 异常: %q err=%v", out, err)
	}

	// ⑦ str_replace_editor：create → view → str_replace → insert
	if _, err := execTool(t, reg, "str_replace_editor", map[string]any{"command": "create", "path": "sre.txt", "file_text": "l1\nl2\n"}); err != nil {
		t.Fatalf("sre create 失败: %v", err)
	}
	out, _ = execTool(t, reg, "str_replace_editor", map[string]any{"command": "view", "path": "sre.txt"})
	if !strings.Contains(out, "l1") {
		t.Fatalf("sre view 异常: %q", out)
	}
	if _, err := execTool(t, reg, "str_replace_editor", map[string]any{"command": "str_replace", "path": "sre.txt", "old_str": "l1", "new_str": "L1"}); err != nil {
		t.Fatalf("sre str_replace 失败: %v", err)
	}
	if _, err := execTool(t, reg, "str_replace_editor", map[string]any{"command": "insert", "path": "sre.txt", "insert_line": 1, "new_str": "inserted"}); err != nil {
		t.Fatalf("sre insert 失败: %v", err)
	}
	data, _ = os.ReadFile(filepath.Join(root, "sre.txt"))
	if !strings.Contains(string(data), "L1") || !strings.Contains(string(data), "inserted") {
		t.Fatalf("sre 编辑未生效: %q", string(data))
	}

	// ⑧ run_code：保持 hostTool（宿主执行器）
	out, err = execTool(t, reg, "run_code", map[string]any{"code": "console.log('rc-ok')", "language": "node"})
	if err != nil || !strings.Contains(out, "rc-ok") {
		t.Fatalf("run_code 异常: %q err=%v", out, err)
	}
}
