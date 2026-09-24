// embedded_tools_test.go — 宿主内嵌工具内核（binary.exec 回退）验证。

package agent

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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
		"codegraph_build", "codegraph_search", "codegraph_relations", // tool-codegraph
		"codegraph_find_entry_points", "codegraph_explore", // tool-codegraph-extra
		"inspect_binary", "write_binary", "binary_strings", "binary_find", // tool-binary
		"screenshot_desktop", "screenshot_area", "screenshot_window", // tool-screenshot
		"web_debug",                          // tool-web-debug
		"run_code",                           // tool-harness
		"word_read", "read_xlsx", "read_pdf", // tool-office（保留内核）
	} {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("内嵌内核应注册 %s", name)
		}
	}
}

// TestEmbeddedToolRegistryFollowsRoot 多工作区「串根」回归（2026-09-17）：
// 用户实测反馈的「截图不落盘」，根因是内嵌内核注册表曾「首次 root 永久缓存」——
// 切换工作区/新会话后仍把 screenshot / web_debug / codegraph 的产物写到旧 root。
// 本测试锁定修复后的不变量：同 root 幂等（同实例）；root 变化或回切 → 重建。
func TestEmbeddedToolRegistryFollowsRoot(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()

	ra1 := InitEmbeddedToolRegistry(rootA)
	ra2 := InitEmbeddedToolRegistry(rootA)
	if ra1 != ra2 {
		t.Fatal("同 root 应幂等复用（同实例）")
	}

	rb := InitEmbeddedToolRegistry(rootB)
	if rb == ra1 {
		t.Fatal("root 变化必须重建注册表 —— 否则跨工作区串根（落盘产物写进旧工作区）")
	}

	// 回切旧工作区同样必须重建（不得沿用 B 的实例，否则又串到 B）
	ra3 := InitEmbeddedToolRegistry(rootA)
	if ra3 == rb {
		t.Fatal("root 回切应重建，不得沿用其他 root 的实例")
	}
	// 重建后注册面完整（重建不丢工具）
	for _, name := range []string{"screenshot_desktop", "web_debug", "inspect_binary", "run_code"} {
		if _, ok := ra3.Get(name); !ok {
			t.Errorf("重建后的注册表应仍注册 %s", name)
		}
	}
}

// TestScreenshotLandsInCallingRoot 端到端：截图必须落在「本次调用传入的 root」。
// ★ 2026-09-17 多工作区串根的真实验收（用户反馈「截图不落盘」）：先用 rootA 初始化
// （模拟第一个工作区已用过内核），再以 rootB 调用 screenshot_area —— 产物必须出现在
// rootB/screenshots 下。修复前会落到 rootA/screenshots（新工作区里「看不到文件」）。
// 无桌面环境（CI）截图失败时跳过（不误报）。
func TestScreenshotLandsInCallingRoot(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("桌面截图仅 Windows 支持")
	}
	rootA := t.TempDir()
	rootB := t.TempDir()
	InitEmbeddedToolRegistry(rootA) // 模拟：第一个工作区先用过内嵌内核

	text, found, err := callEmbeddedTool(context.Background(), rootB, "screenshot_area", map[string]any{
		"left": "0", "top": "0", "right": "20", "bottom": "20",
	})
	if !found {
		t.Fatal("screenshot_area 应注册在内嵌内核中")
	}
	if err != nil {
		t.Skipf("无桌面环境（跳过落盘断言）：%v", err)
	}

	entriesB, _ := os.ReadDir(filepath.Join(rootB, "screenshots"))
	if len(entriesB) == 0 {
		t.Fatalf("截图未落在调用方 root（%s/screenshots 为空）——多工作区串根回归！工具输出: %s", rootB, text)
	}
	if !strings.Contains(text, rootB) {
		t.Errorf("工具返回路径应指向调用方 root：%q", text)
	}
	if _, err := os.Stat(filepath.Join(rootA, "screenshots")); err == nil {
		t.Errorf("截图误落入旧工作区 %s/screenshots（串根未修复）", rootA)
	}
}
