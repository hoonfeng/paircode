// Command ftree_probe loads the companion frontend with simulated workspace
// data (health + fs/list) so the file tree renders, then dumps the geometry
// of item-row / item-name / ws-item to diagnose premature text ellipsis.
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

const mockJS = `
(function(){
	window.__fetchLog = [];
	window.__origFetch = window.fetch;
	function makeResp(obj) {
		return Promise.resolve({
			ok: true, status: 200, statusText: 'OK',
			json: function() { return Promise.resolve(obj); },
			text: function() { return Promise.resolve(JSON.stringify(obj)); }
		});
	}
	window.fetch = function(url, opts) {
		var u = String(url);
		window.__fetchLog.push(u);
		if (u.indexOf('/api/health') === 0) {
			return makeResp({status: 'ok', workspace: 'F:\\syproject\\gou-ide', folders: ['F:\\syproject\\gou-ide']});
		}
		if (u.indexOf('/api/fs/list') === 0) {
			var entries = [
				{name: 'cmd', isDir: true, size: 0},
				{name: 'go.mod', isDir: false, size: 100},
				{name: 'internal', isDir: true, size: 0},
				{name: 'main_very_long_file_name_for_testing.txt', isDir: false, size: 10},
				{name: 'companion.exe', isDir: false, size: 300},
				{name: 'config', isDir: true, size: 0}
			];
			return makeResp(entries);
		}
		if (u.indexOf('/api/settings') === 0) {
			return makeResp({ok: true, recentProjects: ['F:\\syproject\\gou-ide'], workspaceFolderLists: {}});
		}
		if (u.indexOf('/api/') === 0) {
			return makeResp({ok: true, data: []});
		}
		return window.__origFetch.apply(window, arguments);
	};
})()
`

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
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
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
		rt.RunJS(mockJS)
	}
	_ = wv.JSInterpreter()
	wv.LoadHTML(string(html))
	// 多次 setTimeout 让 Vue 异步链（fetch → json → state 更新 → DOM）跑完
	for i := 0; i < 6; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	}
	// 检查 Vue 状态
	for _, c := range []string{
		`'wsList=' + JSON.stringify(window.wsList ? 'has' : 'no')`,
		`(function(){var app=document.querySelector('#app').__vue_app__; return app ? 'HAS_APP' : 'NO_APP';})()`,
		`'wsItems=' + document.querySelectorAll('.ws-item').length`,
		`'ftItems=' + document.querySelectorAll('.item-row').length`,
		`'fetchLog=' + JSON.stringify(window.__fetchLog).slice(0, 200)`,
		`(function(){var p=fetch('/api/settings').then(function(r){return r.json()}).then(function(d){window.__settingsRes=JSON.stringify(d)}); return 'pending';})()`,
		`(function(){var p=fetch('/api/health').then(function(r){return r.json()}).then(function(d){window.__healthRes=JSON.stringify(d)}); return 'pending';})()`,
	} {
		v, err := wv.JSInterpreter().RunJS(c)
		if err != nil {
			fmt.Printf("eval err: %v\n", err)
		} else {
			fmt.Printf("%v\n", v)
		}
	}
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	for _, c := range []string{
		`'settingsRes=' + (window.__settingsRes||'none')`,
		`'healthRes=' + (window.__healthRes||'none')`,
	} {
		v, err := wv.JSInterpreter().RunJS(c)
		if err != nil {
			fmt.Printf("eval2 err: %v\n", err)
		} else {
			fmt.Printf("%v\n", v)
		}
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()

	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cls := el.GetAttribute("class")
			for _, want := range []string{"item-name", "item-row", "ws-item", "ws-name", "file-explorer", "project-section", "ws-section"} {
				if strings.Contains(cls, want) {
					x, y, w, h, ok2 := rendering.BoxGeometry(o)
					textW := 0.0
					if ok2 && (want == "item-name" || want == "ws-name") {
						textW = layout.MeasureTextFunc("", 13, 400, "", el.TextContent())
					}
					fmt.Printf("%s: x=%.1f y=%.1f w=%.1f h=%.1f textW=%.1f %s\n",
						want, x, y, w, h, textW, map[bool]string{true: "OK", false: "TRUNC"}[w >= textW || textW == 0])
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
}
