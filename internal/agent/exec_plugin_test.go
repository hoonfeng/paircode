package agent

// ═══════════════════════════════════════════════════════════════
// exec_plugin_test.go — tool-exec 插件行为验证（2026-09 工具重构）
//
// 装载 .pair/plugins/tool-exec/index.js（磁盘插件），验证会话式执行链：
// exec_command（同步 / yield 转会话）→ write_stdin（增量轮询）→ kill_process（幂等）。
// Go 库侧同源实现（registerShellTools）由 shell_test.go 覆盖。
// ═══════════════════════════════════════════════════════════════

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// loadExecPlugin 装载磁盘 tool-exec 插件（ctx.process 会话式执行）。
func loadExecPlugin(t *testing.T, root string) (*PluginHost, *Registry) {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "..", ".pair", "plugins", "tool-exec", "index.js"))
	if err != nil {
		t.Skipf("tool-exec 插件不存在: %v", err)
	}
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, root)
	id, err := host.DefineJSCodeFull(string(src), "js", "tool-exec 测试",
		filepath.Join("..", "..", ".pair", "plugins", "tool-exec"), "")
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

func TestExecPluginSyncAndSession(t *testing.T) {
	root := t.TempDir()
	_, reg := loadExecPlugin(t, root)

	// ① 同步短命令：输出 + 结束状态行
	out, err := execTool(t, reg, "exec_command", map[string]any{"command": "echo exec_plugin_ok"})
	if err != nil {
		t.Fatalf("exec_command 失败: %v", err)
	}
	if !strings.Contains(out, "exec_plugin_ok") || !strings.Contains(out, "已结束") {
		t.Fatalf("同步输出异常: %q", out)
	}

	// ② 长命令 → session_id（yield 250ms 超时转会话）
	out, err = execTool(t, reg, "exec_command", map[string]any{
		"command": "ping -n 2 127.0.0.1 && echo sess_ok", "yield_time_ms": 250,
	})
	if err != nil {
		t.Fatalf("慢命令启动失败: %v", err)
	}
	m := regexp.MustCompile(`session_id=(\d+)`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("慢命令应返回 session_id: %q", out)
	}
	sid := m[1]

	// ③ write_stdin 轮询至结束（增量输出含 sess_ok）
	var ro string
	for i := 0; i < 300; i++ {
		ro, err = execTool(t, reg, "write_stdin", map[string]any{"session_id": sid})
		if err == nil && strings.Contains(ro, "已结束") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("write_stdin 失败: %v", err)
	}
	if !strings.Contains(ro, "sess_ok") {
		t.Fatalf("轮询输出缺会话内容: %q", ro)
	}

	// ④ kill_process 幂等（会话已结束）
	if _, err := execTool(t, reg, "kill_process", map[string]any{"session_id": sid}); err != nil {
		t.Fatalf("kill_process 失败: %v", err)
	}
}

// TestExecPluginUnknownSession 未知会话 id → 明确报错。
func TestExecPluginUnknownSession(t *testing.T) {
	root := t.TempDir()
	_, reg := loadExecPlugin(t, root)
	if _, err := execTool(t, reg, "write_stdin", map[string]any{"session_id": 999999}); err == nil {
		t.Error("未知会话 write_stdin 应报错")
	}
	if _, err := execTool(t, reg, "kill_process", map[string]any{"session_id": 999999}); err == nil {
		t.Error("未知会话 kill_process 应报错")
	}
}
