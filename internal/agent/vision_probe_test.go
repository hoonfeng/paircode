package agent

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeProbeTestImage 生成测试图片：200x150，白底 + 红色矩形(50,30)-(150,70) + 蓝色边框矩形(20,90)-(180,130)。
func makeProbeTestImage(t *testing.T) string {
	img := image.NewRGBA(image.Rect(0, 0, 200, 150))
	white := color.RGBA{255, 255, 255, 255}
	red := color.RGBA{220, 40, 30, 255}
	blue := color.RGBA{30, 80, 200, 255}
	// 白底
	for y := 0; y < 150; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, white)
		}
	}
	// 红色矩形（实心）
	for y := 30; y < 70; y++ {
		for x := 50; x < 150; x++ {
			img.Set(x, y, red)
		}
	}
	// 蓝色边框矩形（仅边框 4px）
	for y := 90; y < 130; y++ {
		for x := 20; x < 180; x++ {
			if x < 24 || x >= 176 || y < 94 || y >= 126 {
				img.Set(x, y, blue)
			} else {
				img.Set(x, y, white)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "probe_test.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestProbePixel 坐标取色：红色矩形中心应返回红色系。
func TestProbePixel(t *testing.T) {
	path := makeProbeTestImage(t)
	out, err := probeImage(path, `[{"type":"pixel","x":100,"y":50}]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "#DC281E") && !strings.Contains(out, "红") {
		t.Errorf("红色矩形中心应识别为红色，得: %s", truncateOut(out))
	}
	// 白底区域
	out2, _ := probeImage(path, `[{"type":"pixel","x":10,"y":10}]`)
	if !strings.Contains(out2, "白色") && !strings.Contains(out2, "#FFFFFF") {
		t.Errorf("白底应识别为白色，得: %s", truncateOut(out2))
	}
}

// TestProbeRegion 区域颜色：红色矩形区域主色应为红；带期望色校验。
func TestProbeRegion(t *testing.T) {
	path := makeProbeTestImage(t)
	// 期望色匹配（红色区域含 #DC281E）
	out, err := probeImage(path, `[{"type":"region","x1":50,"y1":30,"x2":150,"y2":70,"color":"#DC281E"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "通过") {
		t.Errorf("红色区域匹配期望色应通过，得: %s", truncateOut(out))
	}
	// 期望色不匹配（白底区域期望红色 → 失败）
	out2, _ := probeImage(path, `[{"type":"region","x1":0,"y1":0,"x2":30,"y2":20,"color":"#FF0000"}]`)
	if !strings.Contains(out2, "失败") {
		t.Errorf("白底区域不应匹配红色期望，得: %s", truncateOut(out2))
	}
}

// TestProbeColorSearch 颜色定位：找红色应返回红色矩形位置。
func TestProbeColorSearch(t *testing.T) {
	path := makeProbeTestImage(t)
	out, err := probeImage(path, `[{"type":"color_search","color":"#DC281E","tolerance":40}]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "找到") {
		t.Errorf("应找到红色区域，得: %s", truncateOut(out))
	}
	if !strings.Contains(out, "(50,30)-(150,70)") {
		t.Errorf("红色矩形位置应为 (50,30)-(150,70)，得: %s", truncateOut(out))
	}
}

// TestProbeBorder 边框检测：蓝色边框矩形四边应完整。
func TestProbeBorder(t *testing.T) {
	path := makeProbeTestImage(t)
	out, err := probeImage(path, `[
		{"type":"border","x1":20,"y1":90,"x2":180,"y2":130,"side":"top","color":"#1E50C8"},
		{"type":"border","x1":20,"y1":90,"x2":180,"y2":130,"side":"bottom","color":"#1E50C8"}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "通过") {
		t.Errorf("蓝色边框应检测完整，得: %s", truncateOut(out))
	}
	// 红色矩形上边不是蓝色 → 失败
	out2, _ := probeImage(path, `[{"type":"border","x1":50,"y1":30,"x2":150,"y2":70,"side":"top","color":"#1E50C8"}]`)
	if !strings.Contains(out2, "失败") {
		t.Errorf("红色矩形顶部不应匹配蓝色边框，得: %s", truncateOut(out2))
	}
}

// TestProbeContrast 对比度：黑 vs 白 = 21:1，红 vs 白不达 AA。
func TestProbeContrast(t *testing.T) {
	out, err := probeImage("", `[{"type":"contrast","color":"#000000","color2":"#FFFFFF"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "21.00") && !strings.Contains(out, "AAA") {
		t.Errorf("黑 vs 白对比度应 ≈21:1 达 AAA，得: %s", truncateOut(out))
	}
	// 浅粉 vs 白（#FFC0CB）对比度 ≈1.5 应不达 AA
	out2, _ := probeImage("", `[{"type":"contrast","color":"#FFC0CB","color2":"#FFFFFF"}]`)
	if strings.Contains(out2, "达到 AA") || strings.Contains(out2, "达到 AAA") {
		t.Errorf("浅粉 vs 白不应达 AA，得: %s", truncateOut(out2))
	}
	if !strings.Contains(out2, "1.54") {
		t.Errorf("浅粉 vs 白对比度应 ≈1.54:1，得: %s", truncateOut(out2))
	}
}

// TestProbeBadRules 无效规则处理。
func TestProbeBadRules(t *testing.T) {
	if _, err := probeImage("x.png", `not-json`); err == nil {
		t.Error("非法 JSON 应报错")
	}
	if _, err := probeImage("x.png", `[{"type":"unknown"}]`); err == nil {
		t.Error("未知规则类型应报错")
	}
	if _, err := probeImage("nofile.png", `[{"type":"pixel","x":1,"y":1}]`); err == nil {
		t.Error("不存在的文件应报错")
	}
}

func truncateOut(s string) string {
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}
