// Command ws_wheel_probe verifies DOM wheel event dispatch/listen chain
// in wb-ui: addEventListener('wheel') → dispatchEvent → listener fires?
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	_ = wd
	html := `<!DOCTYPE html><html><head><meta charset="UTF-8" /></head><body>
<div id="t" style="width:200px;height:100px;"></div>
<script>
window.onload = function(){
  var el = document.getElementById('t');
  window.__wheelCount = 0;
  window.__wheelDelta = null;
  el.addEventListener('wheel', function(e){
    window.__wheelCount++;
    window.__wheelDelta = e.deltaY;
    e.preventDefault();
  });
  var we = new WheelEvent('wheel', {deltaY: 100, deltaMode: 0, bubbles: true, cancelable: true});
  var r = el.dispatchEvent(we);
  console.log('[WHEEL] dispatched ok count=' + window.__wheelCount + ' delta=' + window.__wheelDelta + ' ret=' + r);
  // capture listener 测试（xterm 用 {passive:false} 对象 → 引擎解析成 capture）
  window.__capCount = 0;
  el.addEventListener('wheel', function(e){ window.__capCount++; }, {passive: false});
  var we2 = new WheelEvent('wheel', {deltaY: 50, bubbles: true, cancelable: true});
  el.dispatchEvent(we2);
  console.log('[WHEEL] capture test capCount=' + window.__capCount);
  // 冒泡测试：body 上监听
  document.body.addEventListener('wheel', function(e){
    window.__bubbleCount = (window.__bubbleCount||0) + 1;
  });
  var we2 = new WheelEvent('wheel', {deltaY: 50, bubbles: true, cancelable: true});
  el.dispatchEvent(we2);
  console.log('[WHEEL] bubble test bubbleCount=' + window.__bubbleCount);
};
</script>
</body></html>`
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
	wv.Resize(640, 200)
	_ = wv.JSInterpreter()
	wv.LoadHTML(html)
	_, _ = wv.JSInterpreter().RunJS(`if (typeof window.onload === 'function') window.onload();`)
	interp := wv.JSInterpreter()
	if el := interp.GetEventLoop(); el != nil {
		for i := 0; i < 4; i++ {
			el.ProcessTasks(0)
			_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 50); })`)
			el.ProcessTasks(0)
		}
	}
	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println(out)
	}
	_ = strings.TrimSpace
}
