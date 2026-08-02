// Command click_probe loads the companion frontend through wb-ui and simulates
// clicks on a button's icon area vs its blank area, checking which element the
// hit-test returns and whether the Vue @click listener fires (state change).
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(0)
	distDir := filepath.Join("cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
	html, _ := os.ReadFile(filepath.Join(absDist, "index.html"))
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
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		rt.RunJS(`(function(){
			if (!window.__origFetch) {
				window.__origFetch = window.fetch;
				window.fetch = function(url, opts) {
					var u = String(url);
					if (u.indexOf('/api/') === 0) {
						return Promise.resolve(new Response('{"ok":true,"data":[]}', {
							status: 200, headers: {'Content-Type':'application/json'}
						}));
					}
					return window.__origFetch.apply(window, arguments);
				};
			}
		})()`)
	}
	_ = wv.JSInterpreter()
	wv.LoadHTML(string(html))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()

	// 找活动栏第一个按钮（icon）和它的 svg
	var findBox func(o rendering.RenderObject, wantCls string) (float64, float64, float64, float64, string)
	findBox = func(o rendering.RenderObject, wantCls string) (float64, float64, float64, float64, string) {
		if el, ok := o.Node().(*dom.Element); ok {
			cls := el.GetAttribute("class")
			if strings.Contains(cls, wantCls) {
				if x, y, w, h, ok2 := boxGeom(o); ok2 {
					return x, y, w, h, cls
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			if x, y, w, h, cls := findBox(c, wantCls); w > 0 {
				return x, y, w, h, cls
			}
		}
		return 0, 0, 0, 0, ""
	}
	btnX, btnY, btnW, btnH, btnCls := findBox(rv, "activity-top")
	fmt.Printf("activity-top: x=%.0f y=%.0f w=%.0f h=%.0f cls=%q\n", btnX, btnY, btnW, btnH, btnCls)
	// 找到 activity-top 节点，取其第一个 button 子节点
	var at rendering.RenderObject
	var findAt func(o rendering.RenderObject) rendering.RenderObject
	findAt = func(o rendering.RenderObject) rendering.RenderObject {
		if el, ok := o.Node().(*dom.Element); ok {
			if strings.Contains(el.GetAttribute("class"), "activity-top") {
				return o
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			if r := findAt(c); r != nil {
				return r
			}
		}
		return nil
	}
	at = findAt(rv)
	var b1, b2 rendering.RenderObject
	if at != nil {
		for c := at.FirstChild(); c != nil; c = c.NextSibling() {
			if el, ok := c.Node().(*dom.Element); ok && el.LocalName() == "button" {
				if b1 == nil {
					b1 = c
				} else if b2 == nil {
					b2 = c
				}
			}
		}
	}
	if b1 == nil || b2 == nil {
		fmt.Println("NO BUTTON FOUND")
		return
	}
	var svg rendering.RenderObject
	for c := b1.FirstChild(); c != nil; c = c.NextSibling() {
		if el, ok := c.Node().(*dom.Element); ok && el.LocalName() == "svg" {
			svg = c
			break
		}
	}
	var bx, by, bw, bh float64
	{
		x, y, w, h, ok := boxGeom(b1)
		if !ok {
			fmt.Println("NO BUTTON GEOM")
			return
		}
		bx, by, bw, bh = x, y, w, h
	}
	fmt.Printf("button: x=%.0f y=%.0f w=%.0f h=%.0f\n", bx, by, bw, bh)
	var svgX, svgY, svgW, svgH float64
	if svg != nil {
		if x, y, w, h, ok := boxGeom(svg); ok {
			svgX, svgY, svgW, svgH = x, y, w, h
		}
	}
	fmt.Printf("svg: x=%.0f y=%.0f w=%.0f h=%.0f\n", svgX, svgY, svgW, svgH)

	// 点击测试：记录点击前的 active 按钮名称，点击后检查是否切换
	activeName := func() string {
		return evalStr(wv, `(function(){var btns=document.querySelectorAll('.activity-top button'); for(var i=0;i<btns.length;i++){if(btns[i].className.indexOf('active')>=0) return 'btn'+i;} return 'none';})()`)
	}
	click := func(px, py float64, name string) {
		deepest := rendering.HitTest(rv, px, py, "")
		var dName string
		if deepest != nil {
			dName = deepest.TagName() + "." + deepest.GetAttribute("class")
		}
		before := activeName()
		if deepest != nil {
			deepest.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
		}
		// 跑事件循环让 Vue 响应式更新 DOM + 重建渲染树
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 50); })`)
		wv.RebuildRenderTree()
		after := activeName()
		fmt.Printf("[click %s] pt=(%.0f,%.0f) deepest=%s | active: %s -> %s %s\n",
			name, px, py, dName, before, after, map[bool]string{true: "✓TRIGGERED", false: "✗no-change"}[before != after])
	}

	// 第一个按钮（icon）点击
	if svgW > 0 {
		click(svgX+svgW/2, svgY+svgH/2, "b1-icon")
	}
	// 第一个按钮空白
	click(bx+bw-5, by+5, "b1-blank")

	// 第二个按钮（非 active，icon 区域）
	var svg2 rendering.RenderObject
	for c := b2.FirstChild(); c != nil; c = c.NextSibling() {
		if el, ok := c.Node().(*dom.Element); ok && el.LocalName() == "svg" {
			svg2 = c
			break
		}
	}
	var bx2, by2, bw2, bh2 float64
	if x, y, w, h, ok := boxGeom(b2); ok {
		bx2, by2, bw2, bh2 = x, y, w, h
	}
	fmt.Printf("button2: x=%.0f y=%.0f w=%.0f h=%.0f\n", bx2, by2, bw2, bh2)
	if svg2 != nil {
		if x, y, w, h, ok := boxGeom(svg2); ok && w > 0 {
			click(x+w/2, y+h/2, "b2-icon")
		}
	} else {
		click(bx2+bw2/2, by2+bh2/2, "b2-center")
	}
	// 第二个按钮空白区域（右上）
	click(bx2+bw2-5, by2+5, "b2-blank")
}

func evalStr(wv *webkit.WebView, js string) string {
	v, err := wv.JSInterpreter().RunJS(js)
	if err != nil {
		return "ERR:" + err.Error()
	}
	return fmt.Sprintf("%v", v)
}

func boxOf(o rendering.RenderObject) *rendering.RenderBox {
	if rb, ok := o.(*rendering.RenderBox); ok {
		return rb
	}
	return nil
}

func boxGeom(o rendering.RenderObject) (x, y, w, h float64, ok bool) {
	return rendering.BoxGeometry(o)
}
