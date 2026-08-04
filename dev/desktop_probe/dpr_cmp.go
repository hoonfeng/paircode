// Command dpr_cmp 用静态 HTML（文本 + comp-bar 样式圆角条）验证 host 端
// DPR 超采样路径：WB_DPR=1（GPU 直绘）与 WB_DPR=2（CPU 2x + 缩放上屏）
// 分别截图，像素对比文本/圆头平滑度。内容完全静态，排除页面状态干扰。
//
// Run: go run ./dev/desktop_probe/dpr_cmp.go   （WB_DPR 环境变量控制倍率）
package main

import (
	"log"
	"os"
	"strconv"

	"wb-ui/app"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

const htmlDoc = `<html><head><style>
body { margin:0; font-family:'Microsoft YaHei','Segoe UI',sans-serif; }
.row { display:flex; align-items:center; gap:8px; padding:8px 12px; border-bottom:1px solid #e8e8e8; }
.h1 { font-size:18px; font-weight:600; color:#1a1a1a; }
.h2 { font-size:13px; color:#333; }
.comp-bar { height:12px; width:160px; border-radius:6px; padding:0 5px; box-sizing:border-box;
  display:flex; align-items:center; overflow:hidden;
  background:linear-gradient(90deg,#ff5722 0%,#ff9800 40%,#ffeb3b 70%,#4caf50 100%); }
.comp-bar span { font-size:10px; font-weight:700; color:#fff; letter-spacing:0.5px; text-shadow:0 1px 1px rgba(0,0,0,.4); }
.rounded { width:72px; height:36px; border-radius:18px; background:#2196f3; }
</style></head><body>
  <div class="row"><span class="h1">超采样对比 Title 1234567890</span></div>
  <div class="row"><div class="comp-bar"><span>62.3%</span></div><span class="h2">comp-bar 圆头</span></div>
  <div class="row"><span class="h2">中文文本：你好世界 Hello World 平滑度对比</span></div>
  <div class="row"><div class="rounded"></div><span class="h2">大圆角按钮 18px</span></div>
</body></html>`

func main() {
	log.SetFlags(log.Ltime)
	dpr := 1
	if v := os.Getenv("WB_DPR"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 1 {
			dpr = n
		}
	}
	// Mirror host.go font init so measurement matches painting.
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	wv.LoadHTML(htmlDoc)
	host, err := app.NewHostWithDPR(wv, 800, 420, "dpr cmp", dpr)
	if err != nil {
		log.Fatalf("create window: %v", err)
	}
	log.Printf("[dpr_cmp] DPR=%d 窗口已启动（静态对照页）", dpr)
	host.Run()
}
