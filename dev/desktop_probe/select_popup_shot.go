// Command select_popup_shot 验证 <select> 下拉 popup：打开 popup 后截图，
// 对比 Edge 原生下拉的位置/样式。同时验证 select 框内显示 option 文本。
//go:build ignore

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"wb-ui/app"
	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func runJobs(wv *webkit.WebView) {
	for i := 0; i < 15; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func main() {
	log.SetFlags(log.Ltime)
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	html := `<!DOCTYPE html><html><head><style>
		body { margin:0; padding:20px; font-family:"Segoe UI", sans-serif; background:#3a3f44; }
		select { width:220px; height:32px; font-size:14px; margin-bottom:8px; }
	</style></head><body>
		<select id="svc">
			<option value="deepseek">DeepSeek 官方</option>
			<option value="openai">OpenAI</option>
			<option value="qwen">通义千问</option>
			<option value="disabled_opt" disabled>禁用项</option>
			<option value="claude">Claude</option>
		</select>
		<div id="info"></div>
	</body></html>`

	wv := webkit.NewWebView()
	wv.Resize(640, 480)
	_ = wv.JSInterpreter()
	wv.SetConsoleLogger(&jsc.BufferLogger{})
	desktopbridge.Init(wv)
	wv.LoadHTML(html)
	for i := 0; i < 3; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 50); })`)
		runJobs(wv)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 选中第一个 option（模拟浏览器默认选中）
	jsRun(wv, `document.getElementById('svc').options[0].selected = true;`)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	doc := wv.Document()
	sel := findSelectByID(doc, "svc")
	if sel == nil {
		fmt.Println("select not found")
		os.Exit(1)
	}
	rv := wv.RenderView()
	h := app.NewHostForTest(wv, 640, 480)

	// select 框内文本（应显示 "DeepSeek 官方" 而非 value "deepseek"）
	fmt.Println("[select-text]", jsRun(wv, `document.getElementById('svc').selectedOptions[0].textContent`))
	fmt.Println("[select-cs]", jsRun(wv, `(function(){
		var s = document.getElementById('svc');
		var cs = getComputedStyle(s);
		var b = document.body;
		var bcs = getComputedStyle(b);
		return 'sel.fontFamily=' + cs.fontFamily + ' sel.fontSize=' + cs.fontSize +
			' body.fontFamily=' + bcs.fontFamily + ' sel.color=' + cs.color;
	})()`))

	// 打开 popup
	h.MockSelectClick(sel, rv)
	wv.EnsureLayout()
	wv.RebuildRenderTree()

	if h.SelectPopupOpen() {
		fmt.Println("[popup] OPENED")
	} else {
		fmt.Println("[popup] NOT OPENED")
		os.Exit(1)
	}

	// 截图
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	fmt.Printf("[render] bytes=%d nonzero=%d\n", len(pngBytes), countNonzero(pngBytes))
	// dump 几个像素样本（页面深色 #3a3f44，popup 应白色 #ffffff）
	// popup 位置：select(20,20) h32 → popup top=52, 5*24+4=124 高 → y 52..176
	// option 行：y 53..77(1) 77..101(2) 101..125(3) 125..149(4) 149..173(5)
	for _, p := range [][2]int{
		{5, 5},          // 页面深色背景（padding 内）
		{230, 60},       // option1 行内右侧（选中项 → 240 灰）
		{230, 90},       // option2 行内右侧（OpenAI 非选中 → 应白色 255）
		{230, 114},      // option3 行内右侧（qwen 非选中 → 应白色 255）
		{230, 138},      // option4 行内右侧（disabled → 应白色 255）
		{230, 162},      // option5 行内右侧（claude 非选中 → 应白色 255）
		{100, 174},      // popup 底部 4px 空隙 → 应为白色 #ffffff
		{210, 40},       // select 框内（y 21..49）→ 白色
		{300, 150},      // popup 外页面 → 深色 #3a3f44（当前透明=画布背景缺失）
		{300, 300},      // 页面下部 → 深色
	} {
		off := (p[1]*wv.Width() + p[0]) * 4
		if off+3 < len(pngBytes) {
			fmt.Printf("[px] (%d,%d) rgba=(%d,%d,%d,%d)\n", p[0], p[1], pngBytes[off], pngBytes[off+1], pngBytes[off+2], pngBytes[off+3])
		}
	}
	wd, _ := os.Getwd()
	out := filepath.Join(wd, "dev", "desktop_probe", "select_popup_shot.png")
	f, _ := os.Create(out)
	if f != nil {
		img := image.NewRGBA(image.Rect(0, 0, wv.Width(), wv.Height()))
		for y := 0; y < wv.Height(); y++ {
			for x := 0; x < wv.Width(); x++ {
				off := (y*wv.Width() + x) * 4
				if off+3 < len(pngBytes) {
					img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
				}
			}
		}
		_ = png.Encode(f, img)
		f.Close()
		fmt.Println("[shot] → " + out)
	}

	// popup 几何信息
	fmt.Println("[popup-info]", jsRun(wv, `(function(){
		var p = document.querySelector('.select-popup');
		if (!p) return 'no popup';
		var r = p.getBoundingClientRect();
		var sel = document.getElementById('svc').getBoundingClientRect();
		return 'select at(' + sel.x + ',' + sel.y + ') h=' + sel.height +
			' popup at(' + r.x + ',' + r.y + ') w=' + r.width + ' h=' + r.height +
			' options=' + p.children.length;
	})()`))
	fmt.Println("DONE")
}

func jsRun(wv *webkit.WebView, code string) string {
	iv, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "[ERR] " + err.Error()
	}
	return iv.ToString()
}

func countNonzero(b []byte) int {
	n := 0
	for i := 0; i < len(b); i += 4 {
		if b[i] != 0 || b[i+1] != 0 || b[i+2] != 0 || b[i+3] != 0 {
			n++
		}
	}
	return n
}

func findSelectByID(doc *dom.Document, id string) *dom.Element {
	for _, el := range doc.GetElementsByTagName("select") {
		if el.GetAttribute("id") == id {
			return el
		}
	}
	return nil
}
