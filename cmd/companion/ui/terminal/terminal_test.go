//go:build windows

package termpanel

import (
	"strings"
	"testing"

	"github.com/user/gou-ide/cmd/companion/vterm"
)

// TestShellLabel shell 短标签。
func TestShellLabel(t *testing.T) {
	if shellLabel("powershell") != "PowerShell" || shellLabel("gitbash") != "Bash" || shellLabel("cmd") != "CMD" {
		t.Error("shellLabel 标签错")
	}
}

// TestPtyShellFor 内部 shell 码 → 探测到的 pty.Shell；未知码回落默认。
func TestPtyShellFor(t *testing.T) {
	if s := ptyShellFor("cmd"); s.Name == "" || s.Path == "" {
		t.Errorf("ptyShellFor(cmd) 空：%+v", s)
	}
	if s := ptyShellFor("不存在的码"); s.Name == "" {
		t.Error("ptyShellFor(未知) 应回落默认 shell")
	}
}

// TestTerminalCopyAll CopyAll 取屏幕全部文本（喂 vterm 后读回）。
func TestTerminalCopyAll(t *testing.T) {
	ts := &terminalState{vt: vterm.New(20, 4), cols: 20, rows: 4}
	ts.vt.Write([]byte("line one\r\nline two"))
	got := ts.CopyAll()
	if !strings.Contains(got, "line one") || !strings.Contains(got, "line two") {
		t.Errorf("CopyAll=%q 期望含 line one/two", got)
	}
}

// TestClearScreenNoPTY 未起 PTY 时清屏直接重置 vt。
func TestClearScreenNoPTY(t *testing.T) {
	ts := &terminalState{vt: vterm.New(20, 4), cols: 20, rows: 4, shell: "cmd"}
	ts.vt.Write([]byte("junk content"))
	ts.ClearScreen()
	if c := ts.CopyAll(); strings.Contains(c, "junk") {
		t.Errorf("清屏后应空，得 %q", c)
	}
}
