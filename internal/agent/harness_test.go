// ═══════════════════════════════════════════════════════════════
// harness_js_test.go — tool-harness JS 原生化行为验证
//
// 装载 .pair/plugins/tool-harness/index.js（磁盘插件）到临时工作区，
// 验证 read/write/apply_patch/glob/grep 的 JS 实现（ctx.fs）；run_code 保持
// hostTool（宿主执行器）。★ 2026-09 工具重构：后台进程 4 件套移交 tool-exec
// （见 exec_plugin_test.go）；★ Round5：编辑面统一 apply_patch（ctx.fs.applyPatch）。
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

	// ③ apply_patch：Update File（上下文行定位）+ Add File + Move to + Delete File
	if _, err := execTool(t, reg, "apply_patch", map[string]any{"patch": "*** Begin Patch\n*** Update File: x/y.txt\n@@\n a\n-b\n+b2\n c\n*** End Patch\n"}); err != nil {
		t.Fatalf("apply_patch update 失败: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "x", "y.txt"))
	if !strings.Contains(string(data), "b2") {
		t.Fatalf("apply_patch update 未生效: %q", string(data))
	}
	// Add File：新文件（每行 + 前缀）
	if _, err := execTool(t, reg, "apply_patch", map[string]any{"patch": "*** Begin Patch\n*** Add File: x/new.txt\n+hello\n+world\n*** End Patch\n"}); err != nil {
		t.Fatalf("apply_patch add 失败: %v", err)
	}
	if nb, _ := os.ReadFile(filepath.Join(root, "x", "new.txt")); string(nb) != "hello\nworld\n" {
		t.Fatalf("apply_patch add 内容异常: %q", string(nb))
	}
	// Move to（纯移动）：x/new.txt → x/moved.txt
	if _, err := execTool(t, reg, "apply_patch", map[string]any{"patch": "*** Begin Patch\n*** Update File: x/new.txt\n*** Move to: x/moved.txt\n*** End Patch\n"}); err != nil {
		t.Fatalf("apply_patch move 失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "x", "moved.txt")); err != nil {
		t.Fatalf("apply_patch move 目标不存在: %v", err)
	}
	// Delete File
	if _, err := execTool(t, reg, "apply_patch", map[string]any{"patch": "*** Begin Patch\n*** Delete File: x/moved.txt\n*** End Patch\n"}); err != nil {
		t.Fatalf("apply_patch delete 失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "x", "moved.txt")); !os.IsNotExist(err) {
		t.Fatalf("apply_patch delete 后文件仍存在: %v", err)
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
	// ⑥ 后台进程 4 件套（run_background/read_output/kill_process/job_list）
	// ★ 2026-09 工具重构：已移交 tool-exec 插件（exec_command 会话式模型）——
	//   本插件的对应验证见 exec_plugin_test.go。

	// ⑧ run_code：保持 hostTool（宿主执行器）
	out, err = execTool(t, reg, "run_code", map[string]any{"code": "console.log('rc-ok')", "language": "node"})
	if err != nil || !strings.Contains(out, "rc-ok") {
		t.Fatalf("run_code 异常: %q err=%v", out, err)
	}
}

// execTool 取注册表工具直接执行（JS 插件测试共用；原 toolcore_test.go，Round5
//   tool-core 移除后随迁——tool-harness 是唯一装载的磁盘插件宿主）。
func execTool(t *testing.T, reg *Registry, name string, args map[string]any) (string, error) {
	t.Helper()
	tool, ok := reg.Get(name)
	if !ok {
		t.Fatalf("工具 %s 未注册（JS 插件未接管）", name)
	}
	return tool.Handler(nil, args)
}
