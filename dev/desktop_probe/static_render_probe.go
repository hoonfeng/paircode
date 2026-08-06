// Command static_render_probe loads the REAL dist with a REAL conversation
// (containing agent output: thinking / tool_call / assistant messages) and
// with ZERO user interaction samples render frames in three ways:
//   1. A/B: two immediate Paints of the same state  -> render-layer determinism
//   2. A/C: Paint, wait 400ms (JS timers may fire), Paint again
//          -> JS-layer activity (timer/animation/transition mutating DOM)
//   3. diff report: bounding boxes of changed pixels, to localize which
//      element region is unstable
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func setupLoaders(wv *webkit.WebView, distDir string) {
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
				if err != nil {
					return "", err
				}
				return string(data), nil
			}
		}
	}
}

func saveC(c *graphics.Canvas, path string) {
	w, h := c.Width(), c.Height()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	px := c.Pixels()
	for i := 0; i < w*h; i++ {
		img.Pix[i*4+0] = px[i*4+0]
		img.Pix[i*4+1] = px[i*4+1]
		img.Pix[i*4+2] = px[i*4+2]
		img.Pix[i*4+3] = px[i*4+3]
	}
	f, err := os.Create(path)
	if err != nil {
		log.Printf("save %s: %v", path, err)
		return
	}
	defer f.Close()
	png.Encode(f, img)
	log.Printf("saved %s (%dx%d)", path, w, h)
}

// diffPixels returns per-row changed-pixel counts and the global bounding box.
func diffPixels(a, b *graphics.Canvas) (total int, rows []int, bbox [4]int) {
	w, h := a.Width(), a.Height()
	pa, pb := a.Pixels(), b.Pixels()
	rows = make([]int, h)
	bbox = [4]int{-1, -1, -1, -1}
	for y := 0; y < h; y++ {
		cnt := 0
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			if pa[i] != pb[i] || pa[i+1] != pb[i+1] || pa[i+2] != pb[i+2] {
				cnt++
				if bbox[0] == -1 || x < bbox[0] {
					bbox[0] = x
				}
				if x > bbox[1] {
					bbox[1] = x
				}
				if bbox[2] == -1 || y < bbox[2] {
					bbox[2] = y
				}
				if y > bbox[3] {
					bbox[3] = y
				}
			}
		}
		rows[y] = cnt
		total += cnt
	}
	return
}

// rowRuns summarizes contiguous row bands with changes (top 12).
func rowRuns(rows []int) string {
	var parts []string
	inRun := false
	start := 0
	for y, cnt := range rows {
		if cnt > 0 && !inRun {
			inRun = true
			start = y
		}
		if cnt == 0 && inRun {
			inRun = false
			parts = append(parts, fmt.Sprintf("%d-%d(%d)", start, y-1, rows[start]))
			if len(parts) >= 12 {
				return strings.Join(parts, ",")
			}
		}
	}
	if inRun {
		parts = append(parts, fmt.Sprintf("%d-%d(%d)", start, len(rows)-1, rows[start]))
	}
	return strings.Join(parts, ",")
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
	htmlData, err := os.ReadFile(filepath.Join(absDist, "index.html"))
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
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}
	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))

	waitJS := func(ms int) {
		_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	}

	// wait for mount
	waitJS(400)
	waitJS(600)

	// ★ load real conversations (same as conv_click_probe_clean)
	loadJS := `
(function(){
	var out = {};
	try {
		var st = window.__state;
		if (!st) { out.fatal = 'no __state'; return JSON.stringify(out); }
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		st.workspaceFolders = ['F:\\syproject\\gou-ide'];
		var p = fetch('/api/conversations?workspace=' + encodeURIComponent('F:\\syproject\\gou-ide')).then(function(r){ return r.json(); }).then(function(list){
			if (!Array.isArray(list) || !list[0]) { out.noConv = true; return; }
			st.conversations = list;
			st.currentConvId = list[0].id;
			st.messages = [];
			return fetch('/api/conversations/' + encodeURIComponent(list[0].id) + '/messages').then(function(r){ return r.json(); }).then(function(msgs){
				var arr = (msgs && Array.isArray(msgs.messages)) ? msgs.messages : [];
				if (arr.length) { st.messages = arr; st.messagesByConv[list[0].id] = arr; }
				out.msgCount = arr.length;
				out.total = msgs && msgs.total;
				out.firstTitle = (list[0].title || '').slice(0, 40);
				out.done = true;
			});
		}).catch(function(e){ out.fetchErr = String(e).slice(0,200); });
		window.__loadP = p;
		return JSON.stringify(out);
	} catch(e) { out.err = String(e).slice(0,200); return JSON.stringify(out); }
})()`
	iv, _ := wv.JSInterpreter().RunJS(loadJS)
	fmt.Printf("[static] fetch start: %s\n", iv.ToString())
	waitJS(900)

	// ★ let any loading spinners / transitions settle
	waitJS(600)

	// snapshot JS-side state (caret / active / loading flags)
	snapJS := `(function(){
		var st = window.__state || {};
		var el = document.activeElement;
		return JSON.stringify({
			msgCount: st.messages ? st.messages.length : -1,
			curConv: st.currentConvId || '',
			activeTag: el ? (el.tagName || '') + '.' + (el.className || '') : 'null',
			caretVisible: !!(st.caretVisible),
			spinners: document.querySelectorAll('.spinner, .loading, [class*="spin"]').length,
			animEls: document.querySelectorAll('[class*="anim"], [class*="transition"], [class*="fade"]').length
		});
	})()`
	iv, _ = wv.JSInterpreter().RunJS(snapJS)
	fmt.Printf("[static] state: %s\n", iv.ToString())

	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	W, H := 1280, 800
	outDir := filepath.Join(wd, "dev", "desktop_probe")

	// Frame A: immediate paint
	cA := graphics.NewCanvas(W, H)
	rendering.Paint(rv, cA, rendering.Rect{X: 0, Y: 0, Width: float64(W), Height: float64(H)})
	saveC(cA, filepath.Join(outDir, "static_A.png"))

	// Frame B: immediate second paint, same state (render determinism)
	cB := graphics.NewCanvas(W, H)
	rendering.Paint(rv, cB, rendering.Rect{X: 0, Y: 0, Width: float64(W), Height: float64(H)})
	saveC(cB, filepath.Join(outDir, "static_B.png"))

	total, rows, bbox := diffPixels(cA, cB)
	fmt.Printf("[static] A vs B (render determinism): total=%d bbox=%v\n", total, bbox)
	if total > 0 {
		fmt.Printf("[static]   bands: %s\n", rowRuns(rows))
	}

	// Frame C: wait 500ms (JS timers / animations may fire), then paint again
	waitJS(500)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	cC := graphics.NewCanvas(W, H)
	rendering.Paint(rv, cC, rendering.Rect{X: 0, Y: 0, Width: float64(W), Height: float64(H)})
	saveC(cC, filepath.Join(outDir, "static_C.png"))

	total2, rows2, bbox2 := diffPixels(cA, cC)
	fmt.Printf("[static] A vs C (JS activity after 500ms): total=%d bbox=%v\n", total2, bbox2)
	if total2 > 0 {
		fmt.Printf("[static]   bands: %s\n", rowRuns(rows2))
	}

	// Frame D: wait another 500ms
	waitJS(500)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	cD := graphics.NewCanvas(W, H)
	rendering.Paint(rv, cD, rendering.Rect{X: 0, Y: 0, Width: float64(W), Height: float64(H)})
	saveC(cD, filepath.Join(outDir, "static_D.png"))

	total3, rows3, bbox3 := diffPixels(cC, cD)
	fmt.Printf("[static] C vs D (JS activity 500-1000ms): total=%d bbox=%v\n", total3, bbox3)
	if total3 > 0 {
		fmt.Printf("[static]   bands: %s\n", rowRuns(rows3))
	}
	fmt.Printf("[static] done\n")
}
