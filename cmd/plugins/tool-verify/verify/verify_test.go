package verify

// verify_test.go — 验证器引用提取与豁免逻辑测试。
//
// 2026-08-15 修复回归锁：
//  1. fullPathRE 不匹配裸文件名（painter.go）——知识库口语化引用不算路径
//  2. .json 不被 .js 交替截断（加 \b 结尾）
//  3. 点扩展名序列（.js/.jsx 语言表）不被当路径
//  4. .pair/ 元数据目录引用（设计模板）被豁免
//  5. dirPathRE 拆出的目录引用不紧跟文件扩展名（vision.go → vision 误报）

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExtractFileRefs 关键提取规则。
func TestExtractFileRefs(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		// 裸文件名不匹配（口语化引用）
		{"修复了 painter.go 和 host.go", nil},
		// 带路径的文件引用匹配
		{"改动 internal/agent/tools.go", []string{"internal/agent/tools.go"}},
		// .json 不被 .js 截断
		{"从 config/models.json 加载", []string{"config/models.json"}},
		// 点扩展名序列（语言表）不匹配
		{".js/.jsx/.mjs/.ts/.tsx 扩展名", nil},
		{"Kotlin .kt/.kts", nil},
		// .pair/ 元数据目录豁免（设计模板）
		{"项目级指令 {workspace}/.pair/instructions.md", nil},
		// import 路径豁免
		{"github.com/dlclark/regexp2/v2 依赖", nil},
	}
	for _, c := range cases {
		got := extractFileRefs(c.text)
		if len(got) != len(c.want) {
			t.Errorf("%q → got %v, want %v", c.text, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q → got %v, want %v", c.text, got, c.want)
			}
		}
	}
}

// TestExtractDirRefs 目录引用：文件路径拆出的目录前缀不报。
func TestExtractDirRefs(t *testing.T) {
	// vision.go 被 dirPathRE 拆出 internal/agent/vision（后跟 .go）→ 跳过
	got := extractDirRefs("改动 internal/agent/vision.go 和 internal/agent/loop")
	foundFile := false
	for _, d := range got {
		if d == "internal/agent/vision" {
			foundFile = true
		}
	}
	if foundFile {
		t.Errorf("vision.go 拆出的目录 internal/agent/vision 不应被报告：%v", got)
	}
	// 真实目录引用 internal/agent/loop 应保留
	foundDir := false
	for _, d := range got {
		if d == "internal/agent/loop" {
			foundDir = true
		}
	}
	if !foundDir {
		t.Errorf("真实目录 internal/agent/loop 应被报告：%v", got)
	}
}

// TestFileExistsMultiRoot 多根查找 + .pair 豁免后的条目级验证冒烟。
func TestVerifyAllNoFalsePositive(t *testing.T) {
	root := t.TempDir()
	// 造一个真实的文件 + 目录
	os.MkdirAll(filepath.Join(root, "internal", "agent"), 0o755)
	os.WriteFile(filepath.Join(root, "internal", "agent", "tools.go"), []byte("package agent"), 0o644)
	os.MkdirAll(filepath.Join(root, "cmd", "companion", "config"), 0o755)
	os.WriteFile(filepath.Join(root, "cmd", "companion", "config", "models.json"), []byte("{}"), 0o644)

	v := &Verifier{WorkspaceRoots: []string{root}}
	// 正常条目：真实文件 + 裸名 + 语言表 + .pair 模板 —— 不应过期
	ok := "修复了 painter.go；改动 internal/agent/tools.go；从 cmd/companion/config/models.json 加载；" +
		".js/.jsx 扩展名；{.pair}/instructions.md"
	// 过期条目：真实不存在的带路径引用 —— 应过期
	stale := "改动 internal/agent/ghost.go 和 cmd/companion/config/missing.json"

	r := v.VerifyAll(nil, []KBEntry{
		{Path: "ok", Title: "正常条目", Content: ok},
		{Path: "stale", Title: "过期条目", Content: stale},
	})
	if len(r.Stale) != 1 {
		t.Fatalf("应恰好 1 条过期，实际 %d: %+v", len(r.Stale), r.Stale)
	}
	if r.Stale[0].ID != "stale" {
		t.Errorf("过期条目应为 stale，实际 %s", r.Stale[0].ID)
	}
	if !strings.Contains(strings.Join(r.Stale[0].Issues, ";"), "ghost.go") {
		t.Errorf("过期原因应含 ghost.go: %v", r.Stale[0].Issues)
	}
}
