//go:build windows

package filetreepanel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hoonfeng/paircode/cmd/companion/core"
)

// TestWorkspaceBuildRoots 单/多文件夹工作区都正确构建各根并加载内容（VS Code 多根模型）。
func TestWorkspaceBuildRoots(t *testing.T) {
	a := t.TempDir()
	if err := os.WriteFile(filepath.Join(a, "x.txt"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(a, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	b := t.TempDir()
	if err := os.WriteFile(filepath.Join(b, "y.txt"), []byte("2"), 0o644); err != nil {
		t.Fatal(err)
	}

	prevFT, prevWF := FileTree, core.Folders
	defer func() { FileTree, core.Folders = prevFT, prevWF }()
	FileTree = &fileTreeState{}

	// 单文件夹工作区
	core.Folders = []string{a}
	FileTree.buildRoots()
	if len(FileTree.roots) != 1 || FileTree.roots[0].path != a {
		t.Fatalf("单根失败：%+v", FileTree.roots)
	}
	if len(FileTree.roots[0].children) != 2 {
		t.Errorf("根 a 应 2 子，got %d", len(FileTree.roots[0].children))
	}

	// 多文件夹工作区
	core.Folders = []string{a, b}
	FileTree.buildRoots()
	if len(FileTree.roots) != 2 {
		t.Fatalf("多根应 2 个，got %d", len(FileTree.roots))
	}
	if FileTree.roots[1].path != b || len(FileTree.roots[1].children) != 1 {
		t.Errorf("根 b 失败：%+v", FileTree.roots[1])
	}
}

// TestRootDragReorder 拖拽手柄重排：累积位移每过一行高与相邻根换位。
func TestRootDragReorder(t *testing.T) {
	prevFT, prevWF := FileTree, core.Folders
	defer func() { FileTree, core.Folders = prevFT, prevWF }()
	core.Folders = []string{"A", "B", "C"}
	FileTree = &fileTreeState{roots: []*FileNode{{path: "A"}, {path: "B"}, {path: "C"}}}

	FileTree.onRootDragStart("A", 100)
	FileTree.onRootDragMove(100 + 2*rootRowH) // 向下拖两行 → A 移到末尾
	if got := core.Folders; !(got[0] == "B" && got[1] == "C" && got[2] == "A") {
		t.Fatalf("下拖两行后 = %v，want [B C A]", got)
	}
	FileTree.onRootDragMove(100 + rootRowH) // 向上回一行 → A 回中间
	if core.Folders[1] != "A" {
		t.Errorf("上拖一行后 = %v，A 应在中间", core.Folders)
	}
	FileTree.dragPath = "" // 清理，勿触发 onRootDragEnd 落盘
}

// TestProjectName 工作区显示名：单根=文件夹名；多根=「工作区 (N)」。
func TestProjectName(t *testing.T) {
	prev := core.Folders
	defer func() { core.Folders = prev }()

	core.Folders = []string{`F:\my-proj`}
	if got := core.ProjectName(); got != "my-proj" {
		t.Errorf("单根 projectName = %q，want my-proj", got)
	}
	core.Folders = []string{`F:\a`, `F:\b`}
	if got := core.ProjectName(); got != "工作区 (2 个文件夹)" {
		t.Errorf("多根 projectName = %q", got)
	}
}
