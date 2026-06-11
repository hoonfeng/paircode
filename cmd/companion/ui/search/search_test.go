//go:build windows

package searchpanel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/gou-ide/cmd/companion/core"
	"github.com/user/gou-ide/cmd/companion/ui/editor"
)

// useWorkspace 测试内把工作区设为单文件夹 dir（core.Root 即 dir），结束自动还原。
func useWorkspace(t *testing.T, dir string) {
	prev := core.Folders
	core.Folders = []string{dir}
	t.Cleanup(func() { core.Folders = prev })
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSearchRun 验证跨文件搜索：大小写不敏感(默认)→敏感的命中数变化。
func TestSearchRun(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.go"), "package main\nfunc Foo() {}\n// foo bar\n")
	mustWrite(t, filepath.Join(dir, "b.txt"), "hello FOO world\n")

	ps := theSearch
	defer func() { theSearch = ps }()
	useWorkspace(t, dir)
	theSearch = &searchState{collapsed: map[string]bool{}, query: "foo"}

	theSearch.run() // 不区分大小写：Foo + foo + FOO = 3 命中，2 文件
	if theSearch.totalMatches != 3 {
		t.Errorf("insensitive matches=%d, want 3", theSearch.totalMatches)
	}
	if len(theSearch.files) != 2 {
		t.Errorf("files=%d, want 2", len(theSearch.files))
	}

	theSearch.caseSensitive = true
	theSearch.run() // 区分大小写：只有 a.go 的 "foo" = 1 命中
	if theSearch.totalMatches != 1 {
		t.Errorf("sensitive matches=%d, want 1", theSearch.totalMatches)
	}
}

// TestSearchReplace 验证全部替换：foo(不敏感)→bar，整文件字面替换。
func TestSearchReplace(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.go"), "foo and Foo and FOO\n")

	ps, pe := theSearch, editorpanel.Editor
	defer func() { theSearch, editorpanel.Editor = ps, pe }()
	useWorkspace(t, dir)
	editorpanel.Reset()
	theSearch = &searchState{collapsed: map[string]bool{}, query: "foo", replaceText: "bar"}

	theSearch.run()
	if theSearch.totalMatches != 3 {
		t.Fatalf("matches=%d, want 3", theSearch.totalMatches)
	}
	if n := theSearch.doReplace(); n != 1 {
		t.Errorf("replaced files=%d, want 1", n)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "a.go"))
	if string(data) != "bar and bar and bar\n" {
		t.Errorf("content=%q, want all bar", string(data))
	}
}

// TestSearchReplaceFile 验证单文件替换：只改命中的第一个文件，另一个不动。
func TestSearchReplaceFile(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.go"), "foo\n")
	mustWrite(t, filepath.Join(dir, "b.go"), "foo\n")

	ps, pe := theSearch, editorpanel.Editor
	defer func() { theSearch, editorpanel.Editor = ps, pe }()
	useWorkspace(t, dir)
	editorpanel.Reset()
	theSearch = &searchState{collapsed: map[string]bool{}, query: "foo", replaceText: "bar"}

	theSearch.run()
	if len(theSearch.files) != 2 {
		t.Fatalf("files=%d, want 2", len(theSearch.files))
	}
	if !theSearch.doReplaceFile(theSearch.files[0]) { // a.go（WalkDir 字母序在前）
		t.Fatal("doReplaceFile returned false")
	}
	a, _ := os.ReadFile(filepath.Join(dir, "a.go"))
	b, _ := os.ReadFile(filepath.Join(dir, "b.go"))
	if string(a) != "bar\n" {
		t.Errorf("a.go=%q, want bar", string(a))
	}
	if string(b) != "foo\n" {
		t.Errorf("b.go=%q, want unchanged foo", string(b))
	}
}
