//go:build windows

package gitpanel

import (
	"path/filepath"
	"testing"
)

// TestParseHunkNewLine 验证 hunk 头解析出新文件起始行。
func TestParseHunkNewLine(t *testing.T) {
	cases := map[string]int{
		"@@ -1,2 +3,4 @@ func foo()": 3,
		"@@ -10 +12 @@":              12,
		"@@ -0,0 +1,5 @@":            1,
		"not a hunk":                 0,
	}
	for s, want := range cases {
		if got := parseHunkNewLine(s); got != want {
			t.Errorf("parseHunkNewLine(%q)=%d, want %d", s, got, want)
		}
	}
}

// TestComputeGitSnapshot 真起临时 git 仓：改 1 文件 + 新增 1 文件 → 快照应含 1 修改 + 1 未跟踪 + 1 提交。
func TestComputeGitSnapshot(t *testing.T) {
	dir := t.TempDir()
	runGit(dir, "init")
	runGit(dir, "config", "user.email", "t@t.dev")
	runGit(dir, "config", "user.name", "t")
	mustWrite(t, filepath.Join(dir, "a.txt"), "hello\n")
	runGit(dir, "add", "-A")
	if _, err := runGit(dir, "commit", "-m", "init"); err != nil {
		t.Skipf("git commit 不可用，跳过：%v", err)
	}
	mustWrite(t, filepath.Join(dir, "a.txt"), "hello world\n") // 改
	mustWrite(t, filepath.Join(dir, "b.txt"), "new\n")         // 新增

	d := computeGitSnapshot(dir)
	if !d.isRepo {
		t.Fatal("isRepo=false")
	}
	if len(d.modified) != 1 {
		t.Errorf("modified=%d, want 1", len(d.modified))
	}
	if len(d.untracked) != 1 {
		t.Errorf("untracked=%d, want 1", len(d.untracked))
	}
	if len(d.commits) != 1 {
		t.Errorf("commits=%d, want 1", len(d.commits))
	}
	if d.branch == "" {
		t.Error("branch empty")
	}
}

// TestFirstChangedLine 真起一个临时 git 仓：提交后改第 3 行，firstChangedLine 应返回 3。
func TestFirstChangedLine(t *testing.T) {
	dir := t.TempDir()
	runGit(dir, "init")
	runGit(dir, "config", "user.email", "t@t.dev")
	runGit(dir, "config", "user.name", "t")
	mustWrite(t, filepath.Join(dir, "f.txt"), "a\nb\nc\nd\ne\n")
	runGit(dir, "add", "-A")
	if _, err := runGit(dir, "commit", "-m", "init"); err != nil {
		t.Skipf("git commit 不可用，跳过：%v", err)
	}
	mustWrite(t, filepath.Join(dir, "f.txt"), "a\nb\nCHANGED\nd\ne\n") // 改第 3 行
	if got := firstChangedLine(dir, "f.txt"); got != 3 {
		t.Errorf("firstChangedLine=%d, want 3", got)
	}
}

// TestGitCategorize 验证 porcelain 状态分类（已暂存/已修改/未跟踪/冲突）。
func TestGitCategorize(t *testing.T) {
	g := &gitState{collapsed: map[string]bool{}}
	g.categorize('M', ' ', "staged.go")   // 仅暂存
	g.categorize(' ', 'M', "mod.go")      // 仅工作区修改
	g.categorize('?', '?', "new.txt")     // 未跟踪
	g.categorize('U', 'U', "conflict.go") // 冲突
	g.categorize('A', 'M', "both.go")     // 暂存新增 + 工作区修改 → 两段都进

	if len(g.staged) != 2 { // staged.go + both.go
		t.Errorf("staged=%d, want 2", len(g.staged))
	}
	if len(g.modified) != 2 { // mod.go + both.go
		t.Errorf("modified=%d, want 2", len(g.modified))
	}
	if len(g.untracked) != 1 {
		t.Errorf("untracked=%d, want 1", len(g.untracked))
	}
	if len(g.conflict) != 1 {
		t.Errorf("conflict=%d, want 1", len(g.conflict))
	}
	if g.changeCount() != 6 {
		t.Errorf("changeCount=%d, want 6", g.changeCount())
	}
}

// TestGitBadge 验证状态徽标符号/语义。
func TestGitBadge(t *testing.T) {
	cases := []struct {
		st     byte
		staged bool
		sym    string
	}{
		{'?', false, "?"}, {'A', true, "+"}, {'D', true, "-"}, {'R', true, "→"}, {'M', false, "~"}, {'U', false, "!"},
	}
	for _, c := range cases {
		if s, _ := badge(c.st, c.staged); s != c.sym {
			t.Errorf("badge(%c,%v)=%q want %q", c.st, c.staged, s, c.sym)
		}
	}
}
