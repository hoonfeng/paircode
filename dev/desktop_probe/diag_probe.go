// Command diag_probe loads the REAL companion frontend through wb-ui with
// WB_DIAG=style,paint and aggregates the cascade/paint diagnostics into a
// concise report:
//   1. cascade overrides — a (class, property) set by several rules with
//      DIFFERENT values (the "wrong value wins" bug class)
//   2. text ellipsis — every text truncated by text-overflow:ellipsis with
//      the overflow geometry (premature truncation suspect)
//
// Run: set WB_DIAG=style,paint && go run ./dev/desktop_probe/diag_probe.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

// asRB mirrors rendering.asRenderBox's type switch.
func asRB(o rendering.RenderObject) *rendering.RenderBox {
	switch v := o.(type) {
	case *rendering.RenderBox:
		return v
	case *rendering.RenderBlock:
		return &v.RenderBox
	case *rendering.RenderBlockFlow:
		return &v.RenderBlock.RenderBox
	case *rendering.RenderView:
		return &v.RenderBlockFlow.RenderBlock.RenderBox
	}
	return nil
}

const mockJS = `
(function(){
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
		if (u.indexOf('/api/health') === 0) {
			return makeResp({status: 'ok', workspace: 'F:\\syproject\\gou-ide', folders: ['F:\\syproject\\gou-ide']});
		}
		if (u.indexOf('/api/fs/list') === 0) {
			return makeResp([
				{name: 'cmd', isDir: true, size: 0},
				{name: 'go.mod', isDir: false, size: 100},
				{name: 'internal', isDir: true, size: 0},
				{name: 'pkg', isDir: true, size: 0},
				{name: 'companion.exe', isDir: false, size: 300},
				{name: 'config', isDir: true, size: 0},
				{name: 'main_very_long_file_name_for_testing.txt', isDir: false, size: 10}
			]);
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

var (
	reStyle    = regexp.MustCompile(`^\[diag/style\] (\S+): ([a-z-]+) = "([^"]*)"  via (.*) \(origin=(\d+) imp=(\w+) spec=([\d.]+)\)$`)
	reEllipsis = regexp.MustCompile(`^\[diag/paint\] ellipsis (\S+): segRight=([\d.-]+) maxTextRight=([\d.-]+) segX=([\d.-]+) toCB.w=([\d.-]+) text="(.*)"$`)
)

func main() {
	_ = os.Setenv("WB_DIAG", "style,paint")
	// 重定向 stderr（Diagf 写 os.Stderr）到临时文件，结束读回聚合。
	tmp, err := os.CreateTemp("", "wbdiag*.log")
	if err != nil {
		fmt.Println("tmp:", err)
		os.Exit(1)
	}
	defer os.Remove(tmp.Name())
	origStderr := os.Stderr
	os.Stderr = tmp

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
	for i := 0; i < 5; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	_, _ = wv.Render() // 触发 paint 诊断

	os.Stderr = origStderr
	tmp.Close()
	data, _ := os.ReadFile(tmp.Name())
	// 额外：布局溢出检测（子元素超出父内容盒，无 overflow 处理 → 重叠/截断嫌疑）
	overflowReport(wv)
	report(string(data))
}

// overflowReport 遍历渲染树，找出子元素边界超出父内容盒的 box（无 overflow
// auto/hidden 时即"内容溢出未处理"——重叠/截断布局问题的直接证据）。
func overflowReport(wv *webkit.WebView) {
	rv := wv.RenderView()
	if rv == nil {
		return
	}
	type boxInfo struct {
		name    string
		x, y, w, h float64
		parent  string
		px, py, pw, ph float64
		ovfX, ovfY int
	}
	var out []boxInfo
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if rb := asRB(o); rb != nil {
			if el, ok := o.Node().(*dom.Element); ok {
				g := rb.PaddingBoxRect()
				st := o.Style()
				// 只检查有实体的元素 box
				parent := o.Parent()
				if parent != nil {
					if prb := asRB(parent); prb != nil {
						pg := prb.PaddingBoxRect()
						// 子右/下边缘超出父 padding-box（>1px 容差）
						overR := g.X+g.Width > pg.X+pg.Width+1
						overB := g.Y+g.Height > pg.Y+pg.Height+1
						if overR || overB {
							// 排除滚动容器自身（auto 允许溢出）
							ovfX, ovfY := 0, 0
							if st != nil {
								ovfX, ovfY = int(st.OverflowX), int(st.OverflowY)
							}
							ignore := ovfY == 3 || ovfY == 2 || ovfX == 3 || ovfX == 2
							if ignore {
								return
							}
							name := el.LocalName()
							if cls := el.GetAttribute("class"); cls != "" {
								name += "." + strings.Fields(cls)[0]
							}
							pname := "?"
							if pel, ok := parent.Node().(*dom.Element); ok {
								pname = pel.LocalName()
								if cls := pel.GetAttribute("class"); cls != "" {
									pname += "." + strings.Fields(cls)[0]
								}
							}
							out = append(out, boxInfo{name: name, x: g.X, y: g.Y, w: g.Width, h: g.Height,
								parent: pname, px: pg.X, py: pg.Y, pw: pg.Width, ph: pg.Height})
							return
						}
					}
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
	fmt.Printf("\n=== 布局溢出（子元素超出父内容盒，无 overflow 处理） ===\n")
	if len(out) == 0 {
		fmt.Println("  (无溢出)")
		return
	}
	for _, b := range out {
		overR := b.x + b.w - (b.px + b.pw)
		overB := b.y + b.h - (b.py + b.ph)
		fmt.Printf("  %s(%.0f,%.0f %.0fx%.0f) 超出 %s(%.0f,%.0f %.0fx%.0f) 右 %+.1f 下 %+.1f\n",
			b.name, b.x, b.y, b.w, b.h, b.parent, b.px, b.py, b.pw, b.ph, overR, overB)
	}
}

func report(raw string) {
	// ── 1. 覆盖冲突：同一 (元素, 属性) 被多条规则赋予不同值 ──
	type hit struct {
		el   string
		prop string
		val  string
		via  string
		imp  bool
	}
	var hits []hit
	for _, line := range strings.Split(raw, "\n") {
		m := reStyle.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		hits = append(hits, hit{el: m[1], prop: m[2], val: m[3], via: m[4], imp: m[6] == "true"})
	}
	// 分组：prop → el → []val
	propEl := map[string]map[string][]hit{}
	for _, h := range hits {
		if propEl[h.prop] == nil {
			propEl[h.prop] = map[string][]hit{}
		}
		propEl[h.prop][h.el] = append(propEl[h.prop][h.el], h)
	}
	// 输出冲突（同 el 同 prop 有 ≥2 个不同值）
	fmt.Printf("=== 覆盖冲突（同元素同属性多值） ===\n")
	conflicts := 0
	var props []string
	for p := range propEl {
		props = append(props, p)
	}
	sort.Strings(props)
	for _, p := range props {
		for el, list := range propEl[p] {
			vals := map[string]bool{}
			for _, h := range list {
				vals[h.val] = true
			}
			if len(vals) < 2 {
				continue
			}
			conflicts++
			fmt.Printf("\n[%s] %s  — %d 条赋值: %v\n", p, el, len(list), keysOf(vals))
			for _, h := range list {
				fmt.Printf("    %-14q via %-42s imp=%v\n", h.val, h.via, h.imp)
			}
		}
	}
	if conflicts == 0 {
		fmt.Println("  (无同属性多值冲突)")
	}
	fmt.Printf("\n总计: %d 条关键属性级联记录, %d 个冲突元素-属性\n", len(hits), conflicts)

	// ── 2. 文本省略 ──
	fmt.Printf("\n=== 文本省略（text-overflow:ellipsis 触发） ===\n")
	ell := 0
	for _, line := range strings.Split(raw, "\n") {
		m := reEllipsis.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		ell++
		fmt.Printf("  %s: 容器宽=%-5s 段右缘=%-7s 文本=%s\n", m[1], m[5], m[2], m[6])
	}
	if ell == 0 {
		fmt.Println("  (无省略触发)")
	}
	fmt.Printf("\n省略触发 %d 处\n", ell)
}

func keysOf(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
