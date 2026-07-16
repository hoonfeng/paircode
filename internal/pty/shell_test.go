package pty

import "testing"

// TestDetectShells 本机至少探测到一个可用解释器，字段非空，DefaultShell/ShellByName 有兜底。
func TestDetectShells(t *testing.T) {
	shells := DetectShells()
	if len(shells) == 0 {
		t.Fatal("应至少探测到一个 shell")
	}
	for _, s := range shells {
		if s.Name == "" || s.Path == "" {
			t.Errorf("探测到的 shell 字段不应为空：%+v", s)
		}
		t.Logf("探测到：%-14s %s %v", s.Name, s.Path, s.Args)
	}
	if DefaultShell().Path == "" {
		t.Error("DefaultShell 不应为空")
	}
	if ShellByName("绝不存在的_shell_xyz").Path == "" {
		t.Error("ShellByName 找不到时应回落默认（非空）")
	}
}
