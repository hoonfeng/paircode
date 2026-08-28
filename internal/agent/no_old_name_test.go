package agent

// ═══════════════════════════════════════════════════════════════
// no_old_name_test.go — Round3 ② 阶段 D：旧名零注册断言
//
// 目标态：Go 生产 + 测试注册面零旧名
// （read_file/write_file/edit_file/run_command/search_content/
//   search_files/list_files），实现以新 harness 名保留为测试/归档基座。
// 本测试对「默认注册面」做 grep 式断言：注册表不应出现任何旧名，
// 新名（read/write/edit/bash/glob/grep）必须存在。
// ═══════════════════════════════════════════════════════════════

import "testing"

// TestNoOldNameRegistration 默认工具注册面零旧名（Round3 ② 阶段 D 断言）。
func TestNoOldNameRegistration(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	RegisterHarnessTools(reg, root)
	RegisterHostFrameworkTools(reg, root)

	oldNames := []string{
		"read_file", "write_file", "edit_file", "run_command",
		"search_content", "search_files", "list_files",
	}
	for _, old := range oldNames {
		if _, ok := reg.Get(old); ok {
			t.Errorf("旧名 %q 仍注册（Round3 应零旧名）", old)
		}
	}

	newNames := []string{"read", "write", "edit", "bash", "glob", "grep"}
	for _, n := range newNames {
		tool, ok := reg.Get(n)
		if !ok {
			t.Errorf("新名基座工具 %q 未注册", n)
			continue
		}
		if tool.Handler == nil {
			t.Errorf("%q handler 为空", n)
		}
	}
}
