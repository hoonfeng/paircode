// 诊断 fold_sim 全黑问题：LoadHTML 后检查 console 错误、Vue 是否 mount、DOM/渲染树状态。
//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

func setupLoaders(wv *webkit.WebView, dir string) {
	absDir, _ := filepath.Abs(dir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDir, clean))
				if err != nil {
					log.Printf("[SCRIPT] FAIL src=%q err=%v", src, err)
				}
				return string(data), err
			}
		}
	}
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	dir := filepath.Join(wd, "dev", "fold_sim")

	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	htmlData, _ := os.ReadFile(filepath.Join(dir, "fold_sim.html"))
	fmt.Printf("[diag] HTML %d bytes\n", len(htmlData))

	wv := webkit.NewWebView()
	setupLoaders(wv, dir)
	wv.Resize(1280, 800)

	// 在页面脚本执行前（Vue mount 前）检查 #app 的 DOM 结构与 innerHTML 序列化
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		v, err := wv.EvalJS(`(function(){
			var e = document.getElementById('app');
			if (!e) return 'no #app';
			var kids = e.children;
			var desc = [];
			for (var i = 0; i < kids.length && i < 6; i++) {
				desc.push(kids[i].tagName + '#' + (kids[i].getAttribute('class') || ''));
			}
			return 'children=' + kids.length + ' [' + desc.join(',') + '] innerHTML_len=' + e.innerHTML.length +
			       ' first8=' + e.innerHTML.slice(0, 80).replace(/\n/g, '\\n');
		})()`)
		if err != nil {
			fmt.Printf("[diag] pre-mount EvalJS err: %v\n", err)
		} else {
			fmt.Printf("[diag] PRE-MOUNT #app => %s\n", v.ToString())
		}
		// 注入错误捕获：记录 Vue 运行时错误 + 备份 DOM 模板
		wv.EvalJS(`(function(){
			window.__tpl = document.getElementById('app').innerHTML;
			var origErr = console.error || function(){};
			var log = [];
			console.error = function(){ log.push(Array.prototype.slice.call(arguments).join(' ')); origErr.apply(console, arguments); };
			window.__capturedErrors = log;
			window.addEventListener('error', function(ev){
				var stack = '';
				try { stack = (ev && ev.error && ev.error.stack) ? ('\\n' + ev.error.stack) : ''; } catch(_) {}
				try { window.__capturedErrors.push('window.onerror: ' + (ev && ev.message) + stack); } catch(_) {}
			});
		})()`)
	}

	if err := wv.LoadHTML(string(htmlData)); err != nil {
		fmt.Printf("[diag] LoadHTML err: %v\n", err)
	}
	time.Sleep(800 * time.Millisecond)

	// 导出完整模板到文件分析乱码
	if v, err := wv.EvalJS("window.__tpl || ''"); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "_tpl_dump.txt"), []byte(v.ToString()), 0644)
		fmt.Printf("[diag] template dumped: %d bytes\n", len(v.ToString()))
	}

	if out := wv.ConsoleOutput(); out != "" {
		fmt.Printf("[diag] CONSOLE (%d chars):\n%s\n", len(out), out)
	} else {
		fmt.Println("[diag] CONSOLE: (empty)")
	}

	checks := []struct{ name, expr string }{
		{"typeof Vue", "typeof Vue"},
		{"#app exists", "(function(){var e=document.getElementById('app');return e?('tag='+e.tagName):'null'})()"},
		{"#app children", "(function(){var e=document.getElementById('app');return e?('children='+e.children.length):'null'})()"},
		{"#app innerHTML len", "(function(){var e=document.getElementById('app');return e?('len='+e.innerHTML.length):'null'})()"},
		{"window.ChatApp", "typeof ChatApp"},
		{"captured errors", "(function(){return (window.__capturedErrors||[]).join(' | ') || '(none)'})()"},
		{"mount stage", `(function(){
  // 备份存在 window.__tpl（BeforePageScripts 注入）
  var el = document.getElementById('app');
  var tpl = window.__tpl || '';
  if (!tpl) return 'no backup';
  el.innerHTML = tpl;
  var result = '';
  try {
    var comp = (window.app && window.app._component) || null;
    result = 'comp=' + (comp ? 'yes' : 'no');
    var a2 = Vue.createApp(comp);
    // 复制注册的组件
    try {
      var reg = window.app._context && window.app._context.components;
      if (reg) { for (var k in reg) { a2.component(k, reg[k]); } }
    } catch(_) {}
    a2.config.errorHandler = function(err, inst, info) {
      result += ' | ERR info=' + info + ' msg=' + (err && err.message);
      var st = '';
      try { st = err.stack || ''; } catch(_) {}
      result += ' stack=' + String(st).split('\\n').slice(0,4).join(' <= ');
    };
    a2.mount('#app');
    result += ' | mounted';
    var e2 = document.getElementById('app');
    result += ' children=' + e2.children.length + ' len=' + e2.innerHTML.length;
  } catch (e) {
    result += ' | THROW ' + (e && e.message ? e.message : String(e));
    var st2 = '';
    try { st2 = e.stack || ''; } catch(_) {}
    result += ' stack=' + String(st2).split('\\n').slice(0,5).join(' <= ');
  }
  return result;
})()`},
		{"data sanity", `(function(){
  try {
    var c = window.app._component.data();
    var out = 'combos=' + c.combos.length;
    for (var i = 0; i < c.combos.length; i++) {
      var combo = c.combos[i];
      var a = combo && combo.assistant;
      var segs = a ? a.segments : null;
      out += ' | c' + i + ':assistant=' + (a ? 'yes' : 'NO');
      out += ' segs=' + (segs ? segs.length : 'null');
      if (segs) {
        for (var j = 0; j < segs.length; j++) {
          var s = segs[j];
          if (!s || typeof s.type !== 'string') { out += ' BAD[' + j + ']=' + JSON.stringify(s); }
          else out += ' ' + s.type;
        }
      }
    }
    return out;
  } catch (e) { return 'THROW ' + (e && e.message); }
})()`},
		{"vue reactive", `(function(){
  try {
    var r = Vue.reactive({ a: [{type:'x'}, {type:'y'}], b: 'z' });
    var ar = r.a;
    return 'len=' + ar.length + ' t0=' + ar[0].type + ' t1=' + ar[1].type + ' b=' + r.b +
      ' isProxy=' + Vue.isProxy(ar);
  } catch (e) { return 'THROW ' + (e && e.message) + ' stack=' + String(e && e.stack || '').split('\\n').slice(0,3).join(' | '); }
})()`},
		{"seg-only chat-view", `(function(){
  var el = document.getElementById('app');
  el.innerHTML = window.__tpl;
  while (el.children.length > 1) { el.removeChild(el.children[el.children.length - 1]); }
  var out = 'children=' + el.children.length;
  try {
    var a2 = Vue.createApp(window.app._component);
    try { var reg = window.app._context.components; for (var k in reg) a2.component(k, reg[k]); } catch(_) {}
    a2.config.errorHandler = function(err, inst, info) { out += ' | ERR: ' + info + ': ' + (err && err.message); };
    a2.mount('#app');
    out += ' | mounted children=' + el.children.length + ' len=' + el.innerHTML.length;
  } catch (e) { out += ' | THROW ' + (e && e.message); }
  return out;
})()`},
		{"seg-only status-panel", `(function(){
  var el = document.getElementById('app');
  el.innerHTML = window.__tpl;
  while (el.children.length > 1) { el.removeChild(el.children[0]); }
  var out = 'children=' + el.children.length;
  try {
    var a2 = Vue.createApp(window.app._component);
    try { var reg = window.app._context.components; for (var k in reg) a2.component(k, reg[k]); } catch(_) {}
    a2.config.errorHandler = function(err, inst, info) { out += ' | ERR: ' + info + ': ' + (err && err.message); };
    a2.mount('#app');
    out += ' | mounted children=' + el.children.length + ' len=' + el.innerHTML.length;
  } catch (e) { out += ' | THROW ' + (e && e.message); }
  return out;
})()`},
		{"serialize template", `(function(){
  var t = document.createElement('div');
  t.innerHTML = '<template><span>x</span></template><p>y</p>';
  return 'inner=' + JSON.stringify(t.innerHTML) + ' kids=' + t.children.length;
})()`},
		{"compile source", `(function(){
  try {
    var tpl = window.__tpl;
    var res = Vue.compile(tpl);
    var out = 'typeof compile=' + typeof Vue.compile;
    out += ' typeof res=' + typeof res;
    if (typeof res === 'function') {
      var src = res.toString();
      var out2 = 'renderFn len=' + src.length;
      var idx = -1;
      var i = 0;
      while (i < 20) {
        idx = src.indexOf('.type', idx + 1);
        if (idx < 0) break;
        out2 += '\\n@' + idx + ': ...' + src.slice(Math.max(0, idx - 90), idx + 40) + '...';
        i++;
      }
      out += out2;
    } else {
      out += ' resKeys=' + (res ? Object.keys(res).join(',') : 'null');
      if (res && res.errors && res.errors.length) out += ' ERRORS: ' + String(res.errors).slice(0, 300);
    }
    return out;
  } catch (e) { return 'compile THROW: ' + (e && e.message) + ' stack=' + String(e && e.stack || '').split('\\n').slice(0,4).join(' | '); }
})()`},
		{"re-mount err", `(function(){
  var el = document.getElementById('app');
  var tpl = el.innerHTML;
  var out = 'tpl=' + tpl.length;
  try {
    var a = Vue.createApp({
      data: function(){ return {combos: []}; },
      template: '<div class="chat-view"><div class="cv-header">MINI</div><div class="cv-messages" v-if="combos.length"></div></div>'
    });
    a.component('svg-icon', {props:['name','size'], template:'<svg/>'});
    a.mount('#app');
    out += ' mount=OK';
    var el2 = document.getElementById('app');
    out += ' children=' + el2.children.length + ' len=' + el2.innerHTML.length;
    el.innerHTML = tpl;
  } catch (e) {
    out += ' ERR=' + (e && e.message ? e.message : String(e));
  }
  return out;
})()`},
	}
	for _, c := range checks {
		v, err := wv.EvalJS(c.expr)
		if err != nil {
			fmt.Printf("[diag] %-20s ERR: %v\n", c.name, err)
		} else {
			fmt.Printf("[diag] %-20s => %s\n", c.name, v.ToString())
		}
	}

	// 渲染树对象数
	rv := wv.MainFrame().RenderView()
	if rv == nil {
		fmt.Println("[diag] RenderView: nil")
	} else {
		fmt.Printf("[diag] RenderView: obj=%d\n", countObjects(rv))
	}

	// Paint 一次统计非背景像素
	px, err := wv.Render()
	if err != nil {
		fmt.Printf("[diag] Render err: %v\n", err)
	} else {
		fmt.Printf("[diag] Render pixels=%d\n", len(px))
	}
}

func countObjects(ro rendering.RenderObject) int {
	count := 0
	var walk func(r rendering.RenderObject)
	walk = func(r rendering.RenderObject) {
		count++
		for c := r.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(ro)
	return count
}
