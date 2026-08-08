package main

// Command settings_phil_probe 验证「思想」tab 完整渲染（7 子角色）下的
// modal 高度与 settings-body 滚动：
//   - 预置 core.Settings.PhilosophyEnabled=true + 7 角色哲学内容（模拟真实配置）
//   - 打开设置 → 切「思想」tab → 输出 modal/modal-body/settings-body 几何、
//     settings-body 滚动状态（scrollH/clientH/scrollTop）、主 Agent rows=3 +
//     子角色 rows=2 textarea 几何
//   目标：modal 高度 clamp 到 max-height:80vh（视口 800 → 640），内容超高时
//   settings-body 出现滚动条（与 Edge 对齐）。
import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func spRunJobs(wv *webkit.WebView) {
	for i := 0; i < 12; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
}

func spJs(wv *webkit.WebView, code string) string {
	iv, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "[ERR] " + err.Error()
	}
	return iv.ToString()
}

func spLayout(wv *webkit.WebView) {
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	spRunJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
}

// runJobs2 与 spRunJobs 同义（渲染前 flush 任务）。
func runJobs2(wv *webkit.WebView) { spRunJobs(wv) }

// rowFingerprint 取渲染字节某行的一段像素指纹（非背景像素的 x 位置 hash）。
// rowFingerprint 保留（可能未来用）。
func rowFingerprint(pngBytes []byte, x0, y int) uint32 {
	w := 1280
	h := 800
	if y < 0 || y >= h {
		return 0
	}
	var h1, h2 uint32
	cnt := 0
	for x := x0; x < x0+400 && x < w; x++ {
		off := (y*w + x) * 4
		if off+3 >= len(pngBytes) {
			break
		}
		r, g, b, a := pngBytes[off], pngBytes[off+1], pngBytes[off+2], pngBytes[off+3]
		if a > 200 && !(r > 235 && g > 235 && b > 235) {
			h1 = h1*31 + uint32(x)
			h2 += uint32(r) + uint32(g) + uint32(b)
			cnt++
		}
	}
	return h1 ^ (h2 << 7) ^ uint32(cnt*1009)
}

// rowDiff 比较两张渲染图两行的像素差异比例（0 相同，1 全不同）。
func rowDiff(a []byte, ya int, b []byte, yb int) float64 {
	w, h := 1280, 800
	if ya < 0 || ya >= h || yb < 0 || yb >= h {
		return 1
	}
	diff, total := 0, 0
	for x := 0; x < w; x++ {
		oa := (ya*w + x) * 4
		ob := (yb*w + x) * 4
		if oa+3 >= len(a) || ob+3 >= len(b) {
			break
		}
		total++
		if a[oa] != b[ob] || a[oa+1] != b[ob+1] || a[oa+2] != b[ob+2] {
			diff++
		}
	}
	if total == 0 {
		return 1
	}
	return float64(diff) / float64(total)
}


func spBox(rv *rendering.RenderView, el *dom.Element) (x, y, w, h float64, ok bool) {
	b := rv.FindRenderBoxForNode(el)
	if b == nil {
		return 0, 0, 0, 0, false
	}
	return b.X(), b.Y(), b.Width(), b.Height(), true
}

// spBoxByClass 遍历 div 找 class 含关键词的 box。
func spBoxByClass(wv *webkit.WebView, doc *dom.Document, classSub string) (float64, float64, bool) {
	for _, el := range doc.GetElementsByTagName("div") {
		if strings.Contains(el.GetAttribute("class"), classSub) {
			if b := wv.RenderView().FindRenderBoxForNode(el); b != nil {
				return b.Width(), b.Height(), true
			}
		}
	}
	return 0, 0, false
}

func spReport(wv *webkit.WebView) {
	doc := wv.Document()
	if doc == nil {
		fmt.Println("  document nil")
		return
	}
	// 窗口尺寸（modal/modal-body/settings-body）
	if w, h, ok := spBoxByClass(wv, doc, "settings-modal"); ok {
		fmt.Printf("  .settings-modal box=%dx%d (期望 max-height:80vh=640)\n", int(w), int(h))
	}
	if w, h, ok := spBoxByClass(wv, doc, "modal-body"); ok {
		fmt.Printf("  .modal-body box=%dx%d\n", int(w), int(h))
	}
	if w, h, ok := spBoxByClass(wv, doc, "settings-body"); ok {
		fmt.Printf("  .settings-body box=%dx%d\n", int(w), int(h))
	}
	// 滚动状态：通过 JS 读 scrollHeight/clientHeight/scrollTop
	sc := spJs(wv, `(function(){
		var b = document.querySelector('.settings-body');
		if (!b) return 'no settings-body';
		return 'scrollH=' + b.scrollHeight + ' clientH=' + b.clientHeight + ' scrollTop=' + b.scrollTop;
	})()`)
	fmt.Println("  settings-body 滚动:", sc)
	// textarea 几何
	fmt.Println("  textarea 几何:")
	els := doc.GetElementsByTagName("textarea")
	for i, el := range els {
		rows := el.GetAttribute("rows")
		x, y, w, h, ok := spBox(wv.RenderView(), el)
		if !ok {
			fmt.Printf("    [%d] rows=%s → 无 box\n", i, rows)
			continue
		}
		// 只显示设置面板内的（宽 > 300 且 y < 1000）
		if w < 300 || y > 1000 {
			continue
		}
		fmt.Printf("    [%d] rows=%s box=(%.0f,%.0f) %dx%d (rows=3≈61, rows=2≈60)\n",
			i, rows, x, y, int(w), int(h))
	}
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, err := os.ReadFile(filepath.Join(distDir, "index.html"))
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	absDist, _ := filepath.Abs(distDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				return string(data), err
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				return string(data), err
			}
		}
	}
	wv.SetConsoleLogger(&jsc.BufferLogger{})
	desktopbridge.Init(wv)
	// ★ Init 内部 core.Load() 重置 Settings —— 在其后预置思想配置
	//   （模拟真实用户已启用 + 7 角色均有哲学内容）
	core.Settings.PhilosophyEnabled = true
	core.Settings.PhilosophySelected = []string{"tao-te-ching", "sunzi-bingfa"}
	if core.Settings.PhilosophyRoles == nil {
		core.Settings.PhilosophyRoles = make(map[string]string)
	}
	core.Settings.PhilosophyRoles["planner"] = "1. 先拆解目标为可执行步骤。\n2. 每个步骤明确输入输出。\n3. 评估风险并规划回退。"
	core.Settings.PhilosophyRoles["reviewer"] = "1. 审查代码的正确性与风格。\n2. 指出潜在边界条件问题。"
	core.Settings.PhilosophyRoles["judge"] = "评测要客观，给出量化结论。"
	core.Settings.PhilosophyRoles["explorer"] = "探索时优先理解整体结构。"
	core.Settings.PhilosophyRoles["verifier"] = "验证要覆盖正常与异常路径。"
	core.Settings.PhilosophyRoles["debugger"] = "调试要定位根因而非打补丁。"
	core.Settings.PhilosophyRoles["executor"] = "执行要稳，先编译再运行。"
	core.Settings.PhilosophyRoles["main"] = "以简洁、可维护为最高准则。"
	wv.LoadHTML(string(htmlData))
	for i := 0; i < 4; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
		time.Sleep(600 * time.Millisecond)
		spRunJobs(wv)
	}
	spLayout(wv)

	// 打开设置面板
	_ = spJs(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
		return 'clicked';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	spLayout(wv)

	// 切「思想」tab（index 6）
	_ = spJs(wv, `(function(){
		var btns = document.querySelectorAll('.settings-tabs button');
		if (btns.length > 6) { var ev = new Event('click', {bubbles:true}); btns[6].dispatchEvent(ev); }
		return 'tab6';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	spLayout(wv)

	// 输出思想 tab 完整报告（含 7 子角色 textarea + 滚动状态）
	fmt.Println("=== 思想 tab ===")
	en := spJs(wv, `(function(){
		var cb = document.querySelector('.settings-modal .setting-group input[type="checkbox"]');
		return cb ? ('checked=' + cb.checked) : 'no cb';
	})()`)
	fmt.Println("  启用checkbox:", en)
	fmt.Printf("  Go core.Settings: enabled=%v roles=%d (%v)\n",
		core.Settings.PhilosophyEnabled, len(core.Settings.PhilosophyRoles),
		func() string {
			keys := make([]string, 0, len(core.Settings.PhilosophyRoles))
			for k := range core.Settings.PhilosophyRoles {
				keys = append(keys, k)
			}
			return strings.Join(keys, ",")
		}())
	// 直接调 api.getPhilosophy 看 roles 数据是否抵达前端
	fmt.Println("  fetch /api/philosophy →", spJs(wv, `(function(){
		var done = false; var out = 'pending';
		fetch('/api/philosophy').then(function(r){ return r.text(); }).then(function(t){
			out = t.slice(0, 300); done = true;
		}).catch(function(e){ out = 'ERR ' + e.message; done = true; });
		return 'wait';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	fmt.Println("  fetch 结果:", spJs(wv, `(function(){
		var done = false; var out = 'pending';
		fetch('/api/philosophy').then(function(r){ return r.text(); }).then(function(t){
			out = t.slice(0, 400); done = true;
		}).catch(function(e){ out = 'ERR ' + e.message; done = true; });
		return 'wait2';
	})()`))
	fmt.Println("  group-title 数量:", spJs(wv, `document.querySelectorAll('.settings-modal .group-title').length`))
	fmt.Println("  子角色 label:", spJs(wv, `Array.prototype.slice.call(document.querySelectorAll('.role-phil-label')).map(function(e){return e.textContent;}).join(',')`))
	spReport(wv)

	// ★ resize 手柄验证：主 Agent textarea（rows=3，box=(397,333) 586x59）
	//   右下角 15px 区域应绘制浏览器风格斜线手柄（深灰+白高光）。
	//   滚动前主 Agent textarea 可见，先渲染基线像素再扫描。
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs2(wv)
	hbase, _ := wv.Render()
	fmt.Println("=== resize 手柄验证 ===")
	{
		// 主 Agent textarea（rows=3）
		doc := wv.Document()
		var taEl *dom.Element
		for _, el := range doc.GetElementsByTagName("textarea") {
			if el.GetAttribute("rows") == "3" {
				taEl = el
				break
			}
		}
		if taEl != nil {
			if b := wv.RenderView().FindRenderBoxForNode(taEl); b != nil {
				tx, ty, tw, th := b.X(), b.Y(), b.Width(), b.Height()
				fmt.Printf("  textarea(rows=3) box=(%.0f,%.0f) %dx%d\n", tx, ty, tw, th)
				// 扫描右下角 15x15 手柄区：统计与四角背景色不同的像素
				hx0, hy0 := int(tx+tw-15), int(ty+th-15)
				bg := [4]uint8{hbase[(hy0*1280+hx0)*4], hbase[(hy0*1280+hx0)*4+1], hbase[(hy0*1280+hx0)*4+2], 0}
				cnt := 0
				hasLight, hasDark := false, false
				for yy := hy0; yy < hy0+15; yy++ {
					for xx := hx0; xx < hx0+15; xx++ {
						off := (yy*1280 + xx) * 4
						if off+3 >= len(hbase) {
							continue
						}
						r, g, b := hbase[off], hbase[off+1], hbase[off+2]
						// 与背景差 > 40 视为手柄线条像素
						dr := int(r) - int(bg[0])
						dg := int(g) - int(bg[1])
						db2 := int(b) - int(bg[2])
						if dr*dr+dg*dg+db2*db2 > 1600 {
							cnt++
							if r > 230 && g > 230 && b > 230 {
								hasLight = true
							}
							if r < 0x60 && g < 0x60 && b < 0x60 {
								hasDark = true
							}
						}
					}
				}
				fmt.Printf("  手柄区 (%d,%d)-(%d,%d) 线条像素=%d 亮线=%v 暗线=%v (期望 >8 且至少一条斜线)\n",
					hx0, hy0, hx0+14, hy0+14, cnt, hasLight, hasDark)
				if cnt >= 8 {
					fmt.Println("  → resize 手柄已绘制 ✓")
				} else {
					fmt.Printf("  → resize 手柄缺失（线条像素=%d）\n", cnt)
				}
			}
		}
	}

	fmt.Println("=== textarea 内部滚动状态 ===")
	{
		// 子角色 textarea rows=2（44px 视口），内容 3 行 → 应内部溢出可滚
		taInfo := spJs(wv, `(function(){
			var t = document.querySelectorAll('.settings-modal textarea');
			var out = [];
			for (var i = 0; i < t.length; i++) {
				var e = t[i];
				if (e.getBoundingClientRect().width < 300) continue;
				out.push('[' + i + '] rows=' + (e.getAttribute('rows')||'?') +
					' text=' + JSON.stringify((e.textContent||'').slice(0, 12)) +
					' scrollH=' + Math.round(e.scrollHeight) +
					' clientH=' + Math.round(e.clientHeight));
			}
			return out.join(' | ');
		})()`)
		fmt.Println("  ", taInfo)
		// Go 端 BoxContentSize 直接验证（跳过 JS 桥）
		doc := wv.Document()
		for _, el := range doc.GetElementsByTagName("textarea") {
			b := wv.RenderView().FindRenderBoxForNode(el)
			if b == nil || b.Width() < 300 {
				continue
			}
			tw, th := wv.RenderView().BoxContentSize(b)
			fmt.Printf("    Go: rows=%s tc=%d chars content=(%.0f,%.0f)\n",
				el.GetAttribute("rows"), len(el.TextContent()), tw, th)
		}
	}

	// ★ 滚动验证：设置 settings-body.scrollTop=400，内容应上移（滚动渲染）
	fmt.Println("=== 滚动验证 ===")
	// 先渲染 scrollTop=0 的基线像素
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs2(wv)
	base, _ := wv.Render()
	// settings-body 绝对位置
	doc := wv.Document()
	var sbEl *dom.Element
	for _, el := range doc.GetElementsByTagName("div") {
		if strings.Contains(el.GetAttribute("class"), "settings-body") {
			sbEl = el
			break
		}
	}
	sbx, sby := 0.0, 0.0
	if sbEl != nil {
		if b := wv.RenderView().FindRenderBoxForNode(sbEl); b != nil {
			sbx, sby = b.X(), b.Y()
		}
	}
	fmt.Printf("  settings-body 绝对位置 (%.0f,%.0f) render字节=%d\n", sbx, sby, len(base))
	// 设置 scrollTop
	_ = spJs(wv, `(function(){
		var b = document.querySelector('.settings-body');
		if (b) b.scrollTop = 400;
		return 'set';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	spLayout(wv)
	st := spJs(wv, `(function(){
		var b = document.querySelector('.settings-body');
		return b ? ('scrollTop=' + Math.round(b.scrollTop)) : 'no';
	})()`)
	fmt.Println("  设置 scrollTop=400 后:", st)
	scrolled, _ := wv.Render()
	// 滚动后 textarea[3]（原 y=462）应画到 y≈158（-304）；检查像素
	for _, pt := range [][2]int{{410, 170}, {700, 175}, {410, 475}, {700, 480}} {
		off := (pt[1]*1280 + pt[0]) * 4
		if off+3 < len(scrolled) {
			fmt.Printf("    scroll后 pixel(%d,%d) RGBA=%d,%d,%d,%d\n", pt[0], pt[1], scrolled[off], scrolled[off+1], scrolled[off+2], scrolled[off+3])
		}
	}
	// 滚动验证（逐行像素差异）：
	//   - 滚动后 y=150 应显示滚动前 y=150+304=454 的内容（同内容 → diff 小）
	//   - 滚动后 y=150 vs 滚动前 y=150（不同内容 → diff 大）
	same := rowDiff(scrolled, int(sby)+20, base, int(sby)+324)
	diff := rowDiff(scrolled, int(sby)+20, base, int(sby)+20)
	fmt.Printf("  滚动后 y=%d vs 滚动前 y=%d 差异=%.2f (同内容应小)\n", int(sby)+20, int(sby)+324, same)
	fmt.Printf("  滚动后 y=%d vs 滚动前 y=%d 差异=%.2f (不同内容应大)\n", int(sby)+20, int(sby)+20, diff)
	if same < 0.35 && diff > 0.1 {
		fmt.Println("  → 滚动渲染正确（内容随 scrollTop 上移）✓")
	} else {
		fmt.Printf("  → 滚动渲染异常（same=%.2f diff=%.2f）\n", same, diff)
	}
	fmt.Println("DONE")
}
