package agent

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

// TestImagePipelineRealScreenshot 真实截图端到端（无截图时跳过）：
// 源图 → 准入探测 → 归一化（2048²/4MiB）→ DeepSeek 路由投影（640k 像素/1MiB），
// 打印各阶段像素/字节与 data URL 体积，作为 token 节省的实测依据。
func TestImagePipelineRealScreenshot(t *testing.T) {
	dir := os.Getenv("PAIR_IMAGE_E2E_DIR")
	if dir == "" {
		dir = filepath.Join("..", "..", "screenshots")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("无截图目录（%s）：%v", dir, err)
	}
	var chosen string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".png" {
			continue
		}
		chosen = filepath.Join(dir, e.Name())
		break
	}
	if chosen == "" {
		t.Skip("截图目录无 PNG")
	}
	data, err := os.ReadFile(chosen)
	if err != nil {
		t.Fatalf("读图失败: %v", err)
	}
	facts, _, err := decodeImageBytes(data)
	if err != nil {
		t.Fatalf("探测失败: %v", err)
	}
	norm, err := normalizeImageBytes(data, normalizationImageLimits)
	if err != nil {
		t.Fatalf("归一化失败: %v", err)
	}
	proj, err := projectImageForRequest(norm, deepseekRequestImageLimits)
	if err != nil {
		t.Fatalf("投影失败: %v", err)
	}
	dataURL := len("data:"+proj.MediaType+";base64,") + base64.StdEncoding.EncodedLen(len(proj.Data))
	rawDataURL := len("data:"+facts.MediaType+";base64,") + base64.StdEncoding.EncodedLen(len(data))
	t.Logf("源图 %s: %dx%d, %d bytes（data URL %d 字符）",
		filepath.Base(chosen), facts.Width, facts.Height, len(data), rawDataURL)
	t.Logf("归一化: %dx%d %s, %d bytes", norm.Width, norm.Height, norm.MediaType, len(norm.Data))
	t.Logf("DeepSeek 请求变体: %dx%d %s, %d bytes（data URL %d 字符）",
		proj.Width, proj.Height, proj.MediaType, len(proj.Data), dataURL)
	if int64(proj.Width)*int64(proj.Height) > deepseekRequestImageLimits.MaxPixels {
		t.Errorf("投影像素超预算: %dx%d", proj.Width, proj.Height)
	}
	if int64(len(proj.Data)) > deepseekRequestImageLimits.MaxBytes {
		t.Errorf("投影字节超预算: %d", len(proj.Data))
	}
	if dataURL > rawDataURL {
		t.Errorf("投影后体积不应大于原图 data URL: %d > %d", dataURL, rawDataURL)
	}
}
