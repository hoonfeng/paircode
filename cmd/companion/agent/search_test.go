package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchContent(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	mustWrite(t, dir, "a.go", "package main\nfunc Hello() {}\n// TODO: fix\n")
	mustWrite(t, dir, "sub/b.go", "package sub\nfunc hello() {}\n")
	mustWrite(t, dir, "readme.md", "Hello world\n")

	// 基本正则匹配 func ...(（递归到 sub）
	out, err := reg.Execute(ctx, "search_content", `{"pattern":"func \\w+\\("}`)
	if err != nil {
		t.Fatalf("search_content: %v", err)
	}
	if !strings.Contains(out, "a.go:2:") || !strings.Contains(out, "sub/b.go:2:") {
		t.Errorf("应匹配两处 func，得:\n%s", out)
	}

	// 大小写敏感：Hello（大写）匹配 a.go/readme.md，不匹配 b.go 的小写 hello
	out, _ = reg.Execute(ctx, "search_content", `{"pattern":"Hello"}`)
	if !strings.Contains(out, "a.go:2:") || strings.Contains(out, "b.go") {
		t.Errorf("大小写敏感匹配错误:\n%s", out)
	}
	// 忽略大小写：hello 同时匹配 a.go 与 b.go
	out, _ = reg.Execute(ctx, "search_content", `{"pattern":"hello","case_insensitive":true}`)
	if !strings.Contains(out, "a.go") || !strings.Contains(out, "b.go") {
		t.Errorf("忽略大小写应匹配 a.go 与 b.go:\n%s", out)
	}

	// glob 仅搜 *.md
	out, _ = reg.Execute(ctx, "search_content", `{"pattern":"Hello","glob":"*.md"}`)
	if !strings.Contains(out, "readme.md") || strings.Contains(out, ".go") {
		t.Errorf("glob 过滤错误:\n%s", out)
	}

	// path 限定子目录 sub
	out, _ = reg.Execute(ctx, "search_content", `{"pattern":"func","path":"sub"}`)
	if !strings.Contains(out, "sub/b.go") || strings.Contains(out, "a.go:") {
		t.Errorf("path 限定错误:\n%s", out)
	}

	// 无匹配 / 非法正则
	if out, _ = reg.Execute(ctx, "search_content", `{"pattern":"NOTHING_MATCHES_XYZ"}`); !strings.Contains(out, "未找到") {
		t.Errorf("无匹配应提示，得 %q", out)
	}
	if _, err := reg.Execute(ctx, "search_content", `{"pattern":"("}`); err == nil {
		t.Error("非法正则应报错")
	}
}

func TestSearchContentSkipsBinaryAndIgnoredDirs(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	mustWrite(t, dir, "code.go", "needle here\n")
	os.WriteFile(filepath.Join(dir, "blob.bin"), []byte("needle\x00\x00here"), 0o644) // 含空字节=二进制
	mustWrite(t, dir, ".git/config", "needle in git\n")
	mustWrite(t, dir, "node_modules/pkg/index.js", "needle in nm\n")

	out, _ := reg.Execute(context.Background(), "search_content", `{"pattern":"needle"}`)
	if !strings.Contains(out, "code.go") {
		t.Errorf("应匹配 code.go:\n%s", out)
	}
	if strings.Contains(out, "blob.bin") {
		t.Errorf("二进制文件不应被搜:\n%s", out)
	}
	if strings.Contains(out, ".git") || strings.Contains(out, "node_modules") {
		t.Errorf("忽略目录不应被搜:\n%s", out)
	}
}

func TestSearchFiles(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	for _, rel := range []string{"main.go", "util.go", "readme.md", "internal/app/app.go", ".git/hooks/x"} {
		mustWrite(t, dir, rel, "x")
	}

	// *.go 文件名匹配（递归、跳过 .git）
	out, err := reg.Execute(ctx, "search_files", `{"pattern":"*.go"}`)
	if err != nil {
		t.Fatalf("search_files: %v", err)
	}
	for _, want := range []string{"main.go", "util.go", "internal/app/app.go"} {
		if !strings.Contains(out, want) {
			t.Errorf("应含 %s:\n%s", want, out)
		}
	}
	if strings.Contains(out, ".md") || strings.Contains(out, ".git") {
		t.Errorf("不应含 md/.git:\n%s", out)
	}

	// 含 / 的路径通配
	if out, _ = reg.Execute(ctx, "search_files", `{"pattern":"internal/*/app.go"}`); !strings.Contains(out, "internal/app/app.go") {
		t.Errorf("路径通配应匹配:\n%s", out)
	}
	// 无匹配
	if out, _ = reg.Execute(ctx, "search_files", `{"pattern":"*.rs"}`); !strings.Contains(out, "未找到") {
		t.Errorf("无匹配应提示，得 %q", out)
	}
}

// mustWrite 在 root 下按 slash 相对路径写文件（自动建父目录）。供本包测试共用。
func mustWrite(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
