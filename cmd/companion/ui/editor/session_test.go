//go:build windows

package editorpanel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hoonfeng/paircode/cmd/companion/core"
)

// TestEditorSessionPersistRestore 验证打开文件按工作区持久化 + 恢复读磁盘当前内容 + 跳过已删文件。
func TestEditorSessionPersistRestore(t *testing.T) {
	tmp := t.TempDir() // 临时工作区，自动清理，不碰真实目录
	saved := core.Folders
	defer func() { core.Folders = saved }()
	core.Folders = []string{tmp}

	aPath := filepath.Join(tmp, "a.go")
	bPath := filepath.Join(tmp, "b.go")
	if err := os.WriteFile(aPath, []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte("package b\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := &editorState{}
	e.tabs = []*editorTab{{path: aPath}, {path: bPath}}
	e.active = 1
	e.persistSession()

	if _, err := os.Stat(editorSessionPath()); err != nil {
		t.Fatalf("会话文件应写入 主文件夹/.pair/：%v", err)
	}

	// 清空后恢复：标签数/激活项还原，内容为磁盘当前内容
	e.tabs, e.active = nil, 0
	e.RestoreSession()
	if len(e.tabs) != 2 {
		t.Fatalf("恢复标签数=%d，期望 2", len(e.tabs))
	}
	if e.active != 1 {
		t.Errorf("恢复激活项=%d，期望 1", e.active)
	}
	if e.tabs[0].content != "package a\n" {
		t.Errorf("恢复应读磁盘内容，得 %q", e.tabs[0].content)
	}

	// 磁盘内容改变后再恢复应拿到最新内容（"打开的文件能更新内容"）
	if err := os.WriteFile(aPath, []byte("package a // updated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.RestoreSession()
	if e.tabs[0].content != "package a // updated\n" {
		t.Errorf("恢复应读磁盘最新内容，得 %q", e.tabs[0].content)
	}

	// 已删除的文件恢复时跳过
	if err := os.Remove(bPath); err != nil {
		t.Fatal(err)
	}
	e.RestoreSession()
	if len(e.tabs) != 1 {
		t.Fatalf("删文件后恢复标签数=%d，期望 1（跳过已删）", len(e.tabs))
	}
}
