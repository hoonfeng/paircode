// Command open_editor_show launches the desktop IDE in a real window and
// auto-clicks the first file in the file tree after the Vue app mounts,
// so the CodeMirror editor path (companion's real editor component) can be
// inspected visually and in the console.
//
// Run: go run ./dev/desktop_probe/open_editor_show.go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"wb-ui/app"
	"wb-ui/jsc"
	"wb-ui/webkit"
)

func setupLoaders(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				log.Printf("[SCRIPT] src=%q len=%d err=%v", src, len(data), err)
				return string(data), err
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				if err != nil {
					return "", err
				}
				re := regexp.MustCompile(`\[data-v-[a-f0-9]+\]`)
				return re.ReplaceAllString(string(data), ""), nil
			}
		}
	}
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "desktop", "web-ui", "dist")
	htmlData, err := os.ReadFile(distDir + "/index.html")
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()

	// ★ 注入自动点击：Vue 挂载后（3.5s）找文件树中第一个真实文件并点击。
	//   CodeMirror 编辑器组件随后挂载，可观察其渲染与 console 错误。
	//   状态写入 window.__autoState，探针通过 eval 查询；console.log 不实时
	//   显示（BufferLogger 仅 LoadHTML 时 dump 一次）。
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		rt.RunJS(`(function(){
			window.__DESKTOP_MODE__ = true;
			window.__autoState = 'pending';
			setTimeout(function(){
				try {
					var found = null;
					var all = document.querySelectorAll('span, div, a');
					for (var i = 0; i < all.length; i++) {
						var el = all[i];
						if (!el || el.children && el.children.length > 0) continue;
						var t = (el.textContent || '').trim();
						if (t.length > 3 && /\.(go|md|js|ts|vue|json|txt|html|css)$/.test(t)) {
							found = el;
							break;
						}
					}
					if (found) {
						window.__autoState = 'clicking:' + found.textContent;
						var ev = new MouseEvent('click', {bubbles: true, cancelable: true});
						found.dispatchEvent(ev);
						window.__autoState += '|dispatched';
					} else {
						window.__autoState = 'no-file-item';
					}
				} catch(e) {
					window.__autoState = 'error:' + (e && e.message || e);
				}
			}, 3500);
		})()`)
	}

	wv.LoadHTML(string(htmlData))

	// 状态轮询：定期把 window.__autoState 输出到 stderr（console.log 不实时）
	go func() {
		time.Sleep(2 * time.Second)
		for i := 0; i < 20; i++ {
			time.Sleep(1500 * time.Millisecond)
			if rt := wv.JSInterpreter(); rt != nil {
				if v, err := rt.RunJS(`window.__autoState || 'unset'`); err == nil {
					fmt.Fprintf(os.Stderr, "[autoState] %v\n", v.ToString())
				}
			}
		}
	}()

	host, err := app.NewHost(wv, 1280, 800, "PairCode IDE - 编辑器验证")
	if err != nil {
		log.Fatalf("create window: %v", err)
	}
	wv.EnsureLayout()
	fmt.Fprintln(os.Stderr, "[open_editor_show] 窗口已启动；3.5s 后自动点击第一个文件（看编辑区 CodeMirror）")
	host.Run()
}
