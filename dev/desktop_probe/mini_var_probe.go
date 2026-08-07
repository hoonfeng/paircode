// Command mini_var_probe 最小复现 var(--a, var(--b)) 嵌套 fallback 解析。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

const htmlDoc = `<!DOCTYPE html><html><head><style>
:root { --border-color: #30363d; }
.item { border-bottom: 1px solid var(--border-subtle, var(--border-color)); }
.item2 { border-bottom: 1px solid var(--border-color); }
.item3 { border-bottom: 1px solid var(--undefined-var, #123456); }
</style></head><body>
<div class="item" id="a">item nested var</div>
<div class="item2" id="b">item single var</div>
<div class="item3" id="c">item plain fallback</div>
</body></html>`

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

	wv := webkit.NewWebView()
	wv.Resize(600, 200)
	if err := wv.LoadHTML(htmlDoc); err != nil {
		log.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	wv.JSInterpreter().RunJobs()
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 检查 item 的 raw 声明解析（通过 JS getComputedStyle 的 border 属性）
	fmt.Println("[js-computed]")
	if iv, err := wv.JSInterpreter().RunJS(`(function(){
		var a = document.getElementById('a');
		var cs = getComputedStyle(a);
		return JSON.stringify({borderBottom: cs.borderBottom, borderColor: cs.borderBottomColor, borderStyle: cs.borderBottomStyle});
	})()`); err == nil {
		fmt.Println(iv.ToString())
	} else {
		fmt.Println("[ERR]", err)
	}
	fmt.Println("[raw-props]")
	if iv, err := wv.JSInterpreter().RunJS(`(function(){
		var a = document.getElementById('a');
		var s = a.style;
		return 'style attr: ' + (a.getAttribute('style') || '(none)');
	})()`); err == nil {
		fmt.Println(iv.ToString())
	} else {
		fmt.Println("[ERR]", err)
	}

	rv := wv.RenderView()
	var walk func(o rendering.RenderObject, depth int)
	walk = func(o rendering.RenderObject, depth int) {
		if o == nil || depth > 20 {
			return
		}
		if nn := o.Node(); nn != nil {
			if el, ok := nn.(*dom.Element); ok {
				st := o.Style()
				fmt.Printf("[%s.%s] borderBottom=#%02x%02x%02x%02x bw=%.0f\n",
					el.LocalName(), el.GetAttribute("class"),
					st.BorderBottomColor.R, st.BorderBottomColor.G, st.BorderBottomColor.B, st.BorderBottomColor.A,
					st.BorderBottomWidth.Value)
				if el.GetAttribute("class") == "item" {
					fmt.Printf("  customProps: %v\n", st.CustomProperties)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	walk(rendering.RenderObject(rv), 0)
	_ = filepath.Join
	_ = os.Getwd
	_ = fmt.Println
}
