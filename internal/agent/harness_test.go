// ═══════════════════════════════════════════════════════════════
// harness_js_test.go — tool-harness JS 原生化行为验证
//
// 装载 .pair/plugins/tool-harness/index.js（磁盘插件）到临时工作区，
// 验证 read/write/edit/glob/grep/bash/run_code 的 JS 实现
// （ctx.fs/ctx.bash）；run_code 保持 hostTool（宿主执行器）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

	// ② read：全文 + 分页（★ R2-7 外部对齐：行号输出 + total footer）
	out, _ := execTool(t, reg, "read", map[string]any{"path": "x/y.txt"})
	if !strings.Contains(out, "b") || !strings.Contains(out, "End of file - total 3 lines") {
		t.Fatalf("read 全文异常: %q", out)
	}
	out, _ = execTool(t, reg, "read", map[string]any{"path": "x/y.txt", "offset": 2, "limit": 1})
	if !strings.Contains(out, "2: b") || strings.Contains(out, "1: a") {
		t.Fatalf("read 分页异常: %q", out)
	}
	// file_path 别名（外部参数名）等价可用
	out, _ = execTool(t, reg, "read", map[string]any{"file_path": "x/y.txt", "offset": 3, "limit": 1})
	if !strings.Contains(out, "3: c") {
		t.Fatalf("read file_path 别名异常: %q", out)
	}

	// ③ edit：精确替换 + 唯一性 + replace_all（R2-7）
	if _, err := execTool(t, reg, "edit", map[string]any{"path": "x/y.txt", "old_string": "b", "new_string": "b2"}); err != nil {
		t.Fatalf("edit 失败: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "x", "y.txt"))
	if !strings.Contains(string(data), "b2") {
		t.Fatalf("edit 未生效: %q", string(data))
	}
	// 多处出现 → 默认拒绝（须唯一）
	if err := os.WriteFile(filepath.Join(root, "x", "dup.txt"), []byte("x\ny\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := execTool(t, reg, "edit", map[string]any{"path": "x/dup.txt", "old_string": "x", "new_string": "X"}); err == nil {
		t.Fatalf("edit 多处出现应拒绝（须唯一）")
	}
	// replace_all=true → 全部替换
	if _, err := execTool(t, reg, "edit", map[string]any{"path": "x/dup.txt", "old_string": "x", "new_string": "X", "replace_all": true}); err != nil {
		t.Fatalf("edit replace_all 失败: %v", err)
	}
	dupData, _ := os.ReadFile(filepath.Join(root, "x", "dup.txt"))
	if string(dupData) != "X\ny\nX\n" {
		t.Fatalf("edit replace_all 未全部替换: %q", string(dupData))
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

	// ⑥ 后台进程（③.4 并入自 tool-shell）：run_background → job_list → read_output → kill_process
	out, err = execTool(t, reg, "run_background", map[string]any{"command": "echo bg-smoke-ok"})
	if err != nil || !strings.Contains(out, "id=") {
		t.Fatalf("run_background 异常: %q err=%v", out, err)
	}
	idM := regexp.MustCompile(`id=(\d+)`).FindStringSubmatch(out)
	if idM == nil {
		t.Fatalf("run_background 未返回进程 id: %q", out)
	}
	bgID, _ := strconv.Atoi(idM[1])
	out, err = execTool(t, reg, "job_list", map[string]any{})
	if err != nil || !strings.Contains(out, "id="+idM[1]) {
		t.Fatalf("job_list 异常: %q err=%v", out, err)
	}
	out, err = execTool(t, reg, "read_output", map[string]any{"id": bgID})
	if err != nil || !strings.Contains(out, "[") {
		t.Fatalf("read_output 异常: %q err=%v", out, err)
	}
	if _, err := execTool(t, reg, "kill_process", map[string]any{"id": bgID}); err != nil {
		t.Fatalf("kill_process 失败: %v", err)
	}

	// ⑧ run_code：保持 hostTool（宿主执行器）
	out, err = execTool(t, reg, "run_code", map[string]any{"code": "console.log('rc-ok')", "language": "node"})
	if err != nil || !strings.Contains(out, "rc-ok") {
		t.Fatalf("run_code 异常: %q err=%v", out, err)
	}
}
