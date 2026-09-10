package agent

// apply_patch_test.go — apply_patch 解析与应用单测（codex 语法对齐，2026-09）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPatchAddFile(t *testing.T) {
	root := t.TempDir()
	patch := "*** Begin Patch\n*** Add File: sub/new.txt\n+hello\n+world\n*** End Patch\n"
	out, err := ApplyPatchText(root, patch)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "A sub/new.txt") {
		t.Errorf("摘要应含 A sub/new.txt: %q", out)
	}
	data, err := os.ReadFile(filepath.Join(root, "sub", "new.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\nworld\n" {
		t.Errorf("内容 = %q", string(data))
	}
}

func TestApplyPatchUpdate(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: a.txt\n line1\n-line2\n+line2b\n line3\n*** End Patch\n"
	out, err := ApplyPatchText(root, patch)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "M a.txt (+1 -1)") {
		t.Errorf("摘要应含 M a.txt (+1 -1): %q", out)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(data) != "line1\nline2b\nline3\n" {
		t.Errorf("内容 = %q", string(data))
	}
}

func TestApplyPatchUpdateMultiSection(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "m.txt"), []byte("a\nb\nc\nd\ne\nf\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 两段 @@（无上下文文本）：按序定位（b→B、e→E）
	patch := "*** Begin Patch\n*** Update File: m.txt\n@@\n-b\n+B\n@@\n-e\n+E\n*** End Patch\n"
	if _, err := ApplyPatchText(root, patch); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "m.txt"))
	if string(data) != "a\nB\nc\nd\nE\nf\n" {
		t.Errorf("内容 = %q", string(data))
	}
}

func TestApplyPatchUpdateContextNotFound(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("real\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: a.txt\n-zzz\n+yyy\n*** End Patch\n"
	_, err := ApplyPatchText(root, patch)
	if err == nil {
		t.Fatal("上下文不存在应报错")
	}
	if !strings.Contains(err.Error(), "上下文未找到") {
		t.Errorf("错误应含诊断（上下文未找到）: %v", err)
	}
}

func TestApplyPatchDelete(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "gone.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Delete File: gone.txt\n*** End Patch\n"
	out, err := ApplyPatchText(root, patch)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "D gone.txt") {
		t.Errorf("摘要应含 D gone.txt: %q", out)
	}
	if _, err := os.Stat(filepath.Join(root, "gone.txt")); err == nil {
		t.Error("文件应已删除")
	}
	// 不存在 → 报错
	if _, err := ApplyPatchText(root, patch); err == nil {
		t.Error("删除不存在文件应报错")
	}
}

func TestApplyPatchMove(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "old.txt"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: old.txt\n*** Move to: dir/new.txt\n keep\n+added\n*** End Patch\n"
	out, err := ApplyPatchText(root, patch)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "M old.txt → dir/new.txt") {
		t.Errorf("摘要应含移动: %q", out)
	}
	if _, err := os.Stat(filepath.Join(root, "old.txt")); err == nil {
		t.Error("源文件应已移动")
	}
	data, err := os.ReadFile(filepath.Join(root, "dir", "new.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep\nadded\n" {
		t.Errorf("内容 = %q", string(data))
	}
}

func TestApplyPatchCRLFPreserved(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "crlf.txt"), []byte("one\r\ntwo\r\nthree\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: crlf.txt\n one\n-two\n+TWO\n three\n*** End Patch\n"
	if _, err := ApplyPatchText(root, patch); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "crlf.txt"))
	if string(data) != "one\r\nTWO\r\nthree\r\n" {
		t.Errorf("CRLF 风格应保留: %q", string(data))
	}
}

func TestApplyPatchParseErrors(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name  string
		patch string
		want  string
	}{
		{"缺 Begin", "*** Update File: a.txt\n-x\n+y\n*** End Patch\n", "Begin Patch"},
		{"缺 End", "*** Begin Patch\n*** Add File: x.txt\n+1\n", "End Patch"},
		{"Add 行非 + 开头", "*** Begin Patch\n*** Add File: x.txt\nplain\n*** End Patch\n", "+\""},
		{"空补丁", "*** Begin Patch\n*** End Patch\n", "没有任何"},
	}
	for _, c := range cases {
		if _, err := ApplyPatchText(root, c.patch); err == nil {
			t.Errorf("%s: 应报错", c.name)
		} else if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: 错误应含 %q，实际: %v", c.name, c.want, err)
		}
	}
}

func TestApplyPatchWhitespaceTolerance(t *testing.T) {
	root := t.TempDir()
	// 文件行尾有空格；补丁无 → 应容忍（TrimRight 比较）
	if err := os.WriteFile(filepath.Join(root, "ws.txt"), []byte("foo   \nbar\t\nbaz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: ws.txt\n foo\n-bar\n+BAR\n baz\n*** End Patch\n"
	if _, err := ApplyPatchText(root, patch); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "ws.txt"))
	if string(data) != "foo   \nBAR\nbaz\n" {
		t.Errorf("行尾空白应容忍且保留上下文原样: %q", string(data))
	}
}
