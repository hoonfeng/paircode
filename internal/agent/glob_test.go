package agent

import "testing"

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern, name string
		want          bool
	}{
		// 精确单层
		{"*.go", "a.go", true},
		{"*.go", "a.txt", false},
		{"*.go", "foo/a.go", false}, // * 不跨 /
		// ** 递归
		{"**/*.go", "a.go", true},
		{"**/*.go", "foo/a.go", true},
		{"**/*.go", "foo/bar/a.go", true},
		{"**/*.go", "foo/bar/a.txt", false},
		// ** 中间
		{"src/**/auth*", "src/auth.go", true},
		{"src/**/auth*", "src/foo/auth.go", true},
		{"src/**/auth*", "src/foo/bar/auth_service.go", true},
		{"src/**/auth*", "lib/auth.go", false},
		// ** 末尾
		{"src/**", "src/a.go", true},
		{"src/**", "src/foo/a.go", true},
		{"src/**", "lib/a.go", false},
		// ** 单独
		{"**", "a.go", true},
		{"**", "foo/bar/a.go", true},
		// 路径精确
		{"internal/*/main.go", "internal/foo/main.go", true},
		{"internal/*/main.go", "internal/foo/bar/main.go", false}, // * 单层
		// 问号
		{"a?c.go", "abc.go", true},
		{"a?c.go", "ac.go", false},
	}
	for _, c := range cases {
		got := matchGlob(c.pattern, c.name)
		if got != c.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", c.pattern, c.name, got, c.want)
		}
	}
}

func TestMatchGlobFilter(t *testing.T) {
	// 纯文件名模式 → 匹配 base（任意深度同名文件）
	if !matchGlobFilter("*.go", "main.go", "cmd/main.go") {
		t.Error("*.go 应匹配 base main.go")
	}
	if matchGlobFilter("*.go", "main.txt", "cmd/main.go") {
		t.Error("*.go 不应匹配 main.txt")
	}
	// 路径模式 → 匹配 rel
	if !matchGlobFilter("cmd/**/*.go", "main.go", "cmd/foo/main.go") {
		t.Error("cmd/**/*.go 应匹配 rel cmd/foo/main.go")
	}
	if matchGlobFilter("cmd/**/*.go", "main.go", "lib/main.go") {
		t.Error("cmd/**/*.go 不应匹配 lib/main.go")
	}
	// ** 模式按 rel 匹配
	if !matchGlobFilter("**/*.test.ts", "a.test.ts", "src/a.test.ts") {
		t.Error("**/*.test.ts 应匹配 src/a.test.ts")
	}
}
