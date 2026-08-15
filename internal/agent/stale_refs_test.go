package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScanStaleRefs 自然语言/技术短语不误报；真实缺失路径仍报。
func TestScanStaleRefs(t *testing.T) {
	root := t.TempDir()
	// 真实存在的路径
	if err := os.MkdirAll(filepath.Join(root, "internal", "agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "internal", "agent", "tools.go"), []byte("x"), 0o644)

	// 含中文导航文案（知识库树形导航）——不误报
	text := "[[目标]] 项目目标/愿景/里程碑（目标-项目定位）\n" +
		"[[架构]] 分层架构、模块职责（12 篇：模块-*/架构-*）\n" +
		"表单元素包装器（input/select/textarea/form/button/a/script）"
	refs := scanStaleRefs(text, root)
	if len(refs) != 0 {
		t.Errorf("中文/技术短语不应误报，实际：%v", refs)
	}

	// 技术名词复合（≥3 段纯字母数字）——不误报
	text2 := "参考 WebKit/Source/WebCore 架构、HTML/CSS/JS 加载、loader/css/html 子系统"
	refs2 := scanStaleRefs(text2, root)
	if len(refs2) != 0 {
		t.Errorf("技术名词复合不应误报，实际：%v", refs2)
	}

	// 真实缺失路径——仍报
	text3 := "见 internal/agent/tools.go 和 page/frame.go"
	refs3 := scanStaleRefs(text3, root)
	if len(refs3) != 1 || refs3[0] != "page/frame.go" {
		t.Errorf("真实缺失路径应报 page/frame.go，实际：%v", refs3)
	}

	// 存在的真实路径——不报
	text4 := "见 internal/agent/tools.go"
	if refs4 := scanStaleRefs(text4, root); len(refs4) != 0 {
		t.Errorf("存在的路径不应报，实际：%v", refs4)
	}
}

// TestBuildKBStaleness 概览.md 树形导航不再触发过期警告。
func TestBuildKBStaleness(t *testing.T) {
	root := t.TempDir()
	pairDir := filepath.Join(root, ".pair", "project-info")
	if err := os.MkdirAll(pairDir, 0o755); err != nil {
		t.Fatal(err)
	}
	overview := "# 项目概览\n\n" +
		"## 知识库导航（树）\n" +
		"- [[目标]] 项目目标/愿景/里程碑（目标-项目定位）\n" +
		"- [[架构]] 分层架构、模块职责、数据流（12 篇：模块-*/架构-*）\n" +
		"- [[实现]] 机制/功能/工作流细节（3 篇）\n"
	os.WriteFile(filepath.Join(pairDir, "概览.md"), []byte(overview), 0o644)

	got := buildKBStaleness([]string{root})
	if strings.Contains(got, "概览") {
		t.Errorf("概览树形导航不应触发过期警告：\n%s", got)
	}

	// 但真实缺失路径引用仍会报
	os.WriteFile(filepath.Join(pairDir, "模块-x.md"), []byte("引用 page/frame.go（不存在）"), 0o644)
	got2 := buildKBStaleness([]string{root})
	if !strings.Contains(got2, "模块-x") || !strings.Contains(got2, "page/frame.go") {
		t.Errorf("真实缺失引用仍应报告：\n%s", got2)
	}
}
