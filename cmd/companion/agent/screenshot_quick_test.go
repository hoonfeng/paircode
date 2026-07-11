// screenshot_quick_test.go — 截图工具快速端到端验证

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaptureDesktop(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd 失败: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(root, "bin", "tesseract")); err == nil {
			break
		}
		root = filepath.Dir(root)
	}

	result, err := captureDesktop(root, "test_desktop")
	if err != nil {
		t.Fatalf("桌面截图失败: %v", err)
	}
	t.Logf("桌面截图结果:\n%s", result)

	if !strings.Contains(result, "✅") {
		t.Error("结果应包含 ✅ 标记")
	}
	if !strings.Contains(result, "screenshots") {
		t.Error("结果应包含 screenshots 目录")
	}

	cleanupScreenshots(t, root)
}

func TestCaptureArea(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd 失败: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(root, "bin", "tesseract")); err == nil {
			break
		}
		root = filepath.Dir(root)
	}

	result, err := captureArea(root, "10%", "10%", "50%", "50%", "test_area")
	if err != nil {
		t.Fatalf("区域截图失败: %v", err)
	}
	t.Logf("区域截图结果:\n%s", result)

	if !strings.Contains(result, "✅") {
		t.Error("结果应包含 ✅ 标记")
	}

	cleanupScreenshots(t, root)
}

func TestCaptureRect(t *testing.T) {
	img, err := captureRect(0, 0, 100, 100)
	if err != nil {
		t.Fatalf("captureRect 失败: %v", err)
	}
	if img == nil {
		t.Fatal("captureRect 返回 nil")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("尺寸错误: 期望 100x100, 实际 %dx%d", bounds.Dx(), bounds.Dy())
	}
	if len(img.Pix) != 100*100*4 {
		t.Errorf("像素数据长度错误: 期望 %d, 实际 %d", 100*100*4, len(img.Pix))
	}
	t.Logf("✅ captureRect 正常: %dx%d, Pix=%d 字节", bounds.Dx(), bounds.Dy(), len(img.Pix))
}

func TestParseCoord(t *testing.T) {
	tests := []struct {
		input string
		total int
		want  int
	}{
		{"100", 1920, 100},
		{"50%", 1920, 960},
		{"10%", 1080, 108},
		{"0", 100, 0},
		{"100%", 800, 800},
	}

	for _, tc := range tests {
		got, err := parseCoord(tc.input, tc.total)
		if err != nil {
			t.Errorf("parseCoord(%q, %d) 报错: %v", tc.input, tc.total, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseCoord(%q, %d) = %d, 期望 %d", tc.input, tc.total, got, tc.want)
		}
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com/path", "example.com"},
		{"http://google.com", "google.com"},
		{"https://www.baidu.com/s?wd=test", "www.baidu.com"},
		{"localhost:8080", "localhost:8080"},
	}

	for _, tc := range tests {
		got := extractHost(tc.input)
		if got != tc.want {
			t.Errorf("extractHost(%q) = %q, 期望 %q", tc.input, got, tc.want)
		}
	}
}

func TestEnumWindowsVisible(t *testing.T) {
	ws, err := enumWindows("")
	if err != nil {
		t.Fatalf("enumWindows 失败: %v", err)
	}
	if len(ws) == 0 {
		t.Log("⚠️ 未找到可见窗口（可能无桌面环境）")
	} else {
		for i, w := range ws {
			if i >= 3 {
				break
			}
			t.Logf("  窗口: HWND=0x%X 标题=%q", w.HWND, w.Title)
		}
		t.Logf("✅ 共 %d 个可见窗口", len(ws))
	}
}

func TestScreenshotToolsRegistered(t *testing.T) {
	r := NewRegistry()
	root, _ := os.Getwd()
	registerScreenshotTools(r, root)

	for _, name := range []string{"screenshot_desktop", "screenshot_window", "screenshot_area", "screenshot_webpage"} {
		tool, ok := r.Get(name)
		if !ok {
			t.Errorf("工具 %q 未注册", name)
			continue
		}
		if tool.Handler == nil {
			t.Errorf("工具 %q 的 Handler 为空", name)
		}
	}
}

func cleanupScreenshots(t *testing.T, root string) {
	dir := filepath.Join(root, "screenshots")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), "test_") {
			if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
				t.Logf("  已清理: %s", e.Name())
			}
		}
	}
}
