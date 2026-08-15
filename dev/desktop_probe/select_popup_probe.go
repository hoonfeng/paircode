// select_popup_probe 验证 <select> 下拉 popup：点击 select 打开浮层、
// 点击 option 选择并派发 change、点击外部关闭。
//go:build ignore

package main

import (
	"fmt"
	"os"
	"time"

	"wb-ui/app"
	"wb-ui/dom"
	"wb-ui/html5"
	"wb-ui/jsc"
	"wb-ui/rendering"
	"wb-ui/webkit"
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

func js(wv *webkit.WebView, code string) string {
	iv, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "[ERR] " + err.Error()
	}
	return iv.ToString()
}

func runJS(wv *webkit.WebView, code string) string {
	return js(wv, code)
}

func main() {
	html := `<!DOCTYPE html><html><head><style>
		body { margin:0; font-family:sans-serif; }
		select { width:220px; height:32px; font-size:14px; margin:20px; }
		#log { margin:20px; font-size:13px; white-space:pre; }
	</style></head><body>
		<select id="svc">
			<option value="deepseek">DeepSeek</option>
			<option value="openai">OpenAI</option>
			<option value="qwen">通义千问</option>
			<option value="disabled_opt" disabled>禁用项</option>
			<option value="claude">Claude</option>
		</select>
		<div id="log"></div>
		<script>
			var log = document.getElementById('log');
			var sel = document.getElementById('svc');
			sel.addEventListener('change', function(e){
				log.textContent += 'change:' + sel.value + '\n';
			});
		</script>
	</body></html>`

	// 初始化 jsc WebView（与 settings_probe 一致）
	wv := webkit.NewWebView()
	wv.Resize(640, 480)
	_ = wv.JSInterpreter()
	wv.SetConsoleLogger(&jsc.BufferLogger{})
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

	doc := wv.Document()
	selEl := findID(doc, "svc")
	if selEl == nil {
		fmt.Println("select not found")
		os.Exit(1)
	}
	rv := wv.RenderView()
	h := app.NewHostForTest(wv, 640, 480)

	// 1. 初始值
	se, _ := html5.ToSelectElement(selEl)
	fmt.Println("[init] value=", se.Value())

	// 2. 点击 select → 打开 popup
	h.MockSelectClick(selEl, rv)
	wv.EnsureLayout()
	wv.RebuildRenderTree()

	if h.SelectPopupOpen() {
		fmt.Println("[popup] OPENED:", h.SelectPopupInfo())
		// popup 内 option 数量
		opts := se.Options()
		fmt.Println("[popup] options:", len(opts))
	} else {
		fmt.Println("[popup] NOT OPENED")
		os.Exit(1)
	}

	// 3. 点击第二个 option（openai）
	h.MockSelectOption("openai")
	wv.EnsureLayout()
	wv.RebuildRenderTree()

	se2, _ := html5.ToSelectElement(selEl)
	fmt.Println("[after-option] value=", se2.Value())

	// 4. change 事件日志
	js := `document.getElementById('log').textContent`
	v, err := wv.JSInterpreter().RunJS(js)
	if err == nil {
		fmt.Println("[log]", v.ToString())
	}

	// 5. 再打开 → 点击禁用项 → 值不变
	h.MockSelectClick(selEl, rv)
	wv.EnsureLayout()
	wv.RebuildRenderTree()
	h.MockSelectOption("disabled_opt")
	se3, _ := html5.ToSelectElement(selEl)
	fmt.Println("[disabled-click] value=", se3.Value(), "(should stay openai)")

	// 6. 完整 HitTest 路径验证：打开 popup 后，用 rendering.HitTest
	//    命中 popup 内的 option（fixed 定位层优先）。
	h.MockSelectClick(selEl, rv)
	wv.EnsureLayout()
	wv.RebuildRenderTree()
	rv = wv.RenderView() // ★ 重建后必须重新获取 rv（旧引用是旧树）
	// 诊断：popup 元素是否有 RenderBox
	bodyEl := doc.Body()
	fmt.Println("[diag] body children count:", bodyChildrenCount(bodyEl))
	// 用 JS 检查 popup 的 computed style / 尺寸
	fmt.Println("[diag] js popup info:", runJS(wv, `(function(){
		var p = document.querySelector('.select-popup');
		if (!p) return 'no .select-popup';
		var cs = getComputedStyle(p);
		var r = p.getBoundingClientRect();
		return 'pos=' + cs.position + ' display=' + cs.display + ' rect=' + r.width + 'x' + r.height + ' at(' + r.left + ',' + r.top + ')';
	})()`))
	popupEl := doc.GetElementsByTagName("div")
	for _, d := range popupEl {
		if d.GetAttribute("class") == "select-popup" {
			bx := rv.FindRenderBoxForNode(d)
			if bx == nil {
				fmt.Println("[diag] popup div has NO RenderBox")
			} else {
				fmt.Println(fmt.Sprintf("[diag] popup box at (%.0f,%.0f) w=%.0f h=%.0f", bx.AbsoluteX(), bx.AbsoluteY(), bx.Width(), bx.Height()))
			}
			// option 子元素
			for c := d.FirstChild(); c != nil; c = c.NextSibling() {
				optEl, ok := c.(*dom.Element)
				if !ok {
					continue
				}
				ob := rv.FindRenderBoxForNode(optEl)
				if ob == nil {
					fmt.Println("[diag] option", optEl.GetAttribute("data-value"), "NO RenderBox")
				} else {
					fmt.Println(fmt.Sprintf("[diag] option %s box at (%.0f,%.0f) w=%.0f h=%.0f", optEl.GetAttribute("data-value"), ob.AbsoluteX(), ob.AbsoluteY(), ob.Width(), ob.Height()))
				}
			}
		}
	}
	// popup 实际布局（见 diag）：option 行高 21，从 y=21 起
	// qwen 在第 3 行 → 中心 ≈ (110, 21+2*21+10=73)
	hit := rendering.HitTest(rv, 110, 73, "")
	if hit == nil {
		fmt.Println("[hittest] MISS (no element)")
	} else {
		fmt.Println("[hittest] hit=", hit.LocalName(), "data-value=", hit.GetAttribute("data-value"))
		if hit.GetAttribute("data-value") == "qwen" {
			h.MockSelectOptionAt(hit)
			wv.EnsureLayout()
			wv.RebuildRenderTree()
			se4, _ := html5.ToSelectElement(selEl)
			fmt.Println("[hittest-after] value=", se4.Value(), "(should be qwen)")
		} else {
			fmt.Println("[hittest] WRONG option (should be qwen)")
		}
	}

	// 7. 点击外部关闭
	h.MockSelectClose()
	wv.EnsureLayout()
	wv.RebuildRenderTree()
	if h.SelectPopupOpen() {
		fmt.Println("[close] STILL OPEN")
	} else {
		fmt.Println("[close] CLOSED")
	}
	fmt.Println("DONE")
}

func findID(doc *dom.Document, id string) *dom.Element {
	for _, el := range doc.GetElementsByTagName("select") {
		if el.GetAttribute("id") == id {
			return el
		}
	}
	return nil
}

func bodyChildrenCount(body *dom.Element) int {
	n := 0
	for c := body.FirstChild(); c != nil; c = c.NextSibling() {
		n++
	}
	return n
}
