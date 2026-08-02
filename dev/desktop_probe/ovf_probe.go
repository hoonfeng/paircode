// Command ovf_probe evals the computed overflow of key containers through wb-ui
// to see why companion CSS overflow rules are not applied.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
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
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 200); })`)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	rt := wv.JSInterpreter()
	checks := []string{
		`'divCount=' + document.querySelectorAll('div').length`,
		`'psCount=' + document.querySelectorAll('.project-section').length`,
		`(function(){var els=document.querySelectorAll('.project-section'); if(!els.length) return 'NO_PS'; var el=els[0]; return JSON.stringify({cls: el.className, dv: el.getAttribute('data-v-bee51e0d'), cs: getComputedStyle(el).overflowY});})()`,
		`(function(){var els=document.querySelectorAll('.sidebar-content'); if(!els.length) return 'NO_SC'; var el=els[0]; return JSON.stringify({cls: el.className, dv: el.getAttribute('data-v-076f0a91'), cs: getComputedStyle(el).overflow});})()`,
		`'bodyHasPS=' + document.body.innerHTML.indexOf('project-section')`,
	}
	for _, c := range checks {
		v, err := rt.RunJS(c)
		if err != nil {
			fmt.Printf("ERR %v\n", err)
		} else {
			fmt.Printf("%s\n", v)
		}
	}
}
