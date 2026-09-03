// embedded_tools_test.go — 宿主内嵌工具内核（binary.exec 回退）验证。

package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmbeddedToolFallback 验证内嵌内核可执行（codegraph_stats 等较重工具不直接调，
// 用轻量 inspect_binary 探测真实执行路径）。
func TestEmbeddedToolFallback(t *testing.T) {
	root := jsNativeWorkspace
	InitEmbeddedToolRegistry(root)

	// 构造测试文件（工作区内——内核 rescue 检查路径在工作区范围）
	fp := filepath.Join(root, "_temp", "embedded_probe.bin")
	_ = os.MkdirAll(filepath.Dir(fp), 0o755)
	defer os.Remove(fp)
	if err := os.WriteFile(fp, []byte{0x4D, 0x5A, 0x00, 0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatalf("写测试文件失败: %v", err)
	}

	text, found, err := callEmbeddedTool(context.Background(), root, "inspect_binary", map[string]any{"path": fp})
	if err != nil {
		t.Fatalf("inspect_binary 执行失败: %v", err)
	}
	if !found {
		t.Fatal("inspect_binary 应在内嵌内核中注册")
	}
	if !strings.Contains(text, "MZ") && !strings.Contains(text, "PE") {
		t.Errorf("inspect_binary 结果应含魔数: %q", text)
	}

	// 未注册工具 → found=false（调用方走原报错路径）
	_, found, _ = callEmbeddedTool(context.Background(), root, "no_such_tool_xxx", nil)
	if found {
		t.Fatal("no_such_tool_xxx 不应在内嵌内核中注册")
	}

	// 幂等：二次初始化不重建
	r1 := InitEmbeddedToolRegistry(root)
	r2 := InitEmbeddedToolRegistry(root)
	if r1 != r2 {
		t.Fatal("InitEmbeddedToolRegistry 应幂等（同一实例）")
	}
}

// TestEmbeddedToolRegistryCoverage 内嵌内核注册面覆盖（JS 插件 binary.exec 依赖的工具名）。
func TestEmbeddedToolRegistryCoverage(t *testing.T) {
	root := t.TempDir()
	reg := InitEmbeddedToolRegistry(root)
	for _, name := range []string{
		"codegraph_build", "codegraph_search", "codegraph_impact", // tool-codegraph
		"codegraph_find_entry_points", "codegraph_explore", // tool-codegraph-extra
		"inspect_binary", "write_binary", "binary_strings", "binary_find", // tool-binary
		"screenshot_desktop", "screenshot_area", "screenshot_window", // tool-screenshot
		"web_debug",           // tool-web-debug
		"run_code",            // tool-harness
		"word_read", "read_xlsx", "read_pdf", // tool-office（保留内核）
	} {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("内嵌内核应注册 %s", name)
		}
	}
}
