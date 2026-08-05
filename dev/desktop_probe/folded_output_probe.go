// Command folded_output_probe 复现「无操作时 agent 输出影响渲染」的重大 BUG：
//   1. panel-only 模式加载真实 dist（右侧面板独立展示）
//   2. 注入模拟对话（assistant 消息含 tool_call → 折叠摘要条 _folded=true）
//   3. 渲染基线 → dump 折叠摘要条几何
//   4. 不做任何用户操作（不点击/不悬停/不收缩），仅模拟 agent 输出
//      （向 segments 追加 content 段——等价 WS pushSegment 事件）
//   5. 再次渲染 → dump 折叠摘要条几何
//   6. 对比：折叠摘要条/已渲染消息是否发生位置、尺寸、折行变化
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func setupLoadersO(wv *webkit.WebView, distDir string) {
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
				return string(data), err
			}
		}
	}
}

func waitJSO(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobsO(wv *webkit.WebView) {
	for i := 0; i < 12; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
}

type osInfo struct {
	Class  string   `json:"class"`
	X      float64  `json:"x"`
	Y      float64  `json:"y"`
	W      float64  `json:"w"`
	H      float64  `json:"h"`
	Right  float64  `json:"right"`
	Bottom float64  `json:"bottom"`
	Lines  int      `json:"lines"`
	Texts  []string `json:"texts"`
}

// dumpFoldedTree 递归 dump folded-summary 完整子树（含所有子元素几何）。
func dumpFoldedTree(wv *webkit.WebView, tag string) {
	rv := wv.RenderView()
	if rv == nil {
		return
	}
	fmt.Printf("[foldout] == %s (folded subtree) ==\n", tag)
	var walk func(o rendering.RenderObject, depth int)
	walk = func(o rendering.RenderObject, depth int) {
		cn := ""
		if el, ok := o.Node().(*dom.Element); ok {
			cn = el.GetAttribute("class") + " <" + el.LocalName() + ">"
		} else if tn, ok := o.Node().(*dom.Text); ok {
			t := tn.Data()
			if len(t) > 30 {
				t = t[:30] + "…"
			}
			cn = "TEXT(" + t + ")"
		}
		lbInfo := ""
		if lb := o.LayoutBox(); lb != nil && rv.LayoutState() != nil {
			g := rv.LayoutState().GeometryForBox(lb)
			lbInfo = fmt.Sprintf(" rect=(%.1f,%.1f %.1fx%.1f) right=%.1f", g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(), g.Left()+g.BorderBoxWidth())
		}
		fmt.Printf("[foldout] %*s%s %q%s\n", depth*2, "", fmt.Sprintf("%T", o), cn, lbInfo)
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	// 从 rv 找 folded-summary
	var find func(o rendering.RenderObject, depth int) bool
	find = func(o rendering.RenderObject, depth int) bool {
		if el, ok := o.Node().(*dom.Element); ok {
			if strings.Contains(el.GetAttribute("class"), "folded-summary") {
				walk(o, depth)
				return true
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			if find(c, depth+1) {
				return true
			}
		}
		return false
	}
	find(rendering.RenderObject(rv), 0)
}

// dumpFoldedO 递归收集折叠摘要条及其文本几何。
func dumpFoldedO(wv *webkit.WebView, tag string) []osInfo {
	rv := wv.RenderView()
	if rv == nil {
		log.Fatalf("no render view at %s", tag)
	}
	var out []osInfo
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			if strings.Contains(cn, "folded-summary") || strings.Contains(cn, "folded-title") || strings.Contains(cn, "folded-desc") {
				lb := o.LayoutBox()
				if lb != nil && rv.LayoutState() != nil {
					g := rv.LayoutState().GeometryForBox(lb)
					fi := osInfo{Class: cn, X: g.Left(), Y: g.Top(), W: g.BorderBoxWidth(), H: g.BorderBoxHeight(),
						Right: g.Left() + g.BorderBoxWidth(), Bottom: g.Top() + g.BorderBoxHeight()}
					var texts []string
					var lines int
					var walkT func(s rendering.RenderObject)
					walkT = func(s rendering.RenderObject) {
						if rt, ok := s.(*rendering.RenderText); ok {
							t := strings.TrimSpace(rt.Text())
							if t != "" {
								sb := rt.LayoutBox()
								if sb != nil && rv.LayoutState() != nil {
									sg := rv.LayoutState().GeometryForBox(sb)
									lines++
									if len(texts) < 6 {
										if len(t) > 20 {
											t = t[:20] + "…"
										}
										texts = append(texts, fmt.Sprintf("(%.0f,%.0f)%s", sg.Left(), sg.Top(), t))
									}
								}
							}
						}
						for c := s.FirstChild(); c != nil; c = c.NextSibling() {
							walkT(c)
						}
					}
					walkT(o)
					fi.Lines = lines
					fi.Texts = texts
					out = append(out, fi)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	fmt.Printf("[foldout] == %s == count=%d\n", tag, len(out))
	for _, fi := range out {
		fmt.Printf("[foldout]   cls=%.20s rect=(%.0f,%.0f %.0fx%.0f) right=%.0f bot=%.0f lines=%d texts=%v\n",
			fi.Class, fi.X, fi.Y, fi.W, fi.H, fi.Right, fi.Bottom, fi.Lines, fi.Texts)
	}
	return out
}

// dumpAllMsg 收集所有 msg-item 的几何（用于观察已渲染消息是否移动）。
func dumpAllMsg(wv *webkit.WebView, tag string) {
	rv := wv.RenderView()
	if rv == nil {
		return
	}
	fmt.Printf("[foldout] == %s (msg-items) ==\n", tag)
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			if strings.Contains(cn, "msg-item") {
				lb := o.LayoutBox()
				if lb != nil && rv.LayoutState() != nil {
					g := rv.LayoutState().GeometryForBox(lb)
					idx := el.GetAttribute("data-idx")
					fmt.Printf("[foldout]   msg-idx=%s rect=(%.0f,%.0f %.0fx%.0f) cls=%.30s\n",
						idx, g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(), cn)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
}

func renderPNGO(wv *webkit.WebView, path string) {
	if pngBytes, err := wv.Render(); err != nil {
		log.Printf("Render: %v", err)
	} else {
		w, h := wv.Width(), wv.Height()
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				off := (y*w + x) * 4
				if off+3 < len(pngBytes) {
					img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
				}
			}
		}
		f, _ := os.Create(path)
		defer f.Close()
		_ = png.Encode(f, img)
		fmt.Printf("[foldout] rendered %dx%d -> %s\n", w, h, path)
	}
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, err := os.ReadFile(filepath.Join(distDir, "index.html"))
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
	setupLoadersO(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	// panel-only：只渲染右侧面板
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		rt.RunJS(`window.__DESKTOP_PANEL_MODE__ = true;`)
	}
	wv.LoadHTML(string(htmlData))
	waitJSO(wv, 500)
	waitJSO(wv, 500)
	runJobsO(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// ── 注入模拟对话（不经 API，直接写 state）──
	iv, _ := wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		if (!st) { return JSON.stringify({fatal: 'no state'}); }
		var convId = 'c_foldout_1';
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		st.workspaceFolders = ['F:\\syproject\\gou-ide'];
		st.conversations = [{id: convId, title: '折叠复现测试', updatedAt: '2026-08-06T10:00:00Z'}];
		// 历史消息：user → assistant（tool_call + content）
		var segs = [
			{type: 'tool_call', name: 'read_file', argsRaw: JSON.stringify({path: 'F:/syproject/gou-ide/internal/core/config.go'}), result: 'ok', _expanded: false},
			{type: 'content', content: '已经读取了配置文件，接下来继续排查。'}
		];
		var msgs = [
			{role: 'user', content: '排查一下配置文件加载问题', segments: [], toolCalls: [], _key: 'm_u1', _idx: 0, _time: '10:00'},
			{role: 'assistant', content: '', segments: segs, toolCalls: [], _key: 'm_a1', _idx: 1, _time: '10:01', _folded: true}
		];
		st.messagesByConv[convId] = msgs;
		st.messages = msgs;
		st.currentConvId = convId;
		return JSON.stringify({ok: 1, convId: convId, msgs: msgs.length});
	})()`)
	fmt.Printf("[foldout] inject: %s\n", iv.ToString())
	waitJSO(wv, 600)
	runJobsO(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsO(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// ── 基线：折叠摘要条几何 + 消息几何 + 截图 ──
	base := dumpFoldedO(wv, "BASE")
	dumpFoldedTree(wv, "BASE-TREE")
	dumpAllMsg(wv, "BASE")
	renderPNGO(wv, filepath.Join(wd, "dev", "desktop_probe", "foldout_base.png"))

	// ── 无任何用户操作！仅模拟 agent 输出：向 assistant 消息追加 content 段
	//    （等价 WS pushSegment 事件，流式输出继续）──
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		if (!st) return JSON.stringify({fatal: 'no state'});
		var convId = st.currentConvId;
		var msgs = st.messagesByConv[convId];
		var last = msgs[msgs.length - 1];
		// 追加一段新的 content（模拟 agent 流式输出继续）
		last.segments.push({type: 'content', content: '追加的第二段输出内容，用于验证无操作时 agent 输出是否影响渲染。'});
		// 同时更新该消息 content（前端 pushSegment 语义）
		last.content = '已经读取了配置文件，接下来继续排查。追加的第二段输出内容';
		return JSON.stringify({ok: 1, segs: last.segments.length});
	})()`)
	fmt.Printf("[foldout] simulate agent output (no user interaction): %s\n", iv.ToString())
	waitJSO(wv, 600)
	runJobsO(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsO(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// ── 输出后：再次 dump ──
	after := dumpFoldedO(wv, "AFTER-OUTPUT")
	dumpFoldedTree(wv, "AFTER-TREE")
	dumpAllMsg(wv, "AFTER-OUTPUT")
	renderPNGO(wv, filepath.Join(wd, "dev", "desktop_probe", "foldout_after_output.png"))

	// ── 对比：折叠摘要条几何是否变化 ──
	fmt.Printf("[foldout] ==== 对比 (无操作, 仅 agent 输出) ====\n")
	if len(base) != len(after) {
		fmt.Printf("[foldout] 数量变化: base=%d after=%d ← 异常！\n", len(base), len(after))
	}
	for i := 0; i < len(base) && i < len(after); i++ {
		b, a := base[i], after[i]
		dx, dy := a.X-b.X, a.Y-b.Y
		dw, dh := a.W-b.W, a.H-b.H
		flag := "OK"
		if dx != 0 || dy != 0 || dw != 0 || dh != 0 {
			flag = "CHANGED"
		}
		fmt.Printf("[foldout] %-7s %-18s d=(%+.1f,%+.1f) size=(%+.1f,%+.1f) lines %d→%d\n",
			flag, b.Class, dx, dy, dw, dh, b.Lines, a.Lines)
	}

	// ── 窄窗口（600px）：验证"完成摘要"标题是否折行（Edge 每行两字）──
	wv.Resize(600, 800)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsO(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	dumpFoldedO(wv, "W600")
	renderPNGO(wv, filepath.Join(wd, "dev", "desktop_probe", "foldout_w600.png"))

	// ── 超长摘要：验证 ellipsis 截断（不膨胀、不折行）──
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		if (!st) return JSON.stringify({fatal: 'no state'});
		var msgs = st.messagesByConv[st.currentConvId];
		var last = msgs[msgs.length - 1];
		// 追加超长 content（摘要文本将远超容器宽度）
		last.segments.push({type: 'content', content: '这是一段非常非常长的输出内容，用来验证折叠摘要条的描述文本在超长情况下是否会被 ellipsis 截断而不是无限膨胀导致布局跳动或标题折行。'.repeat(3)});
		return JSON.stringify({ok: 1, segs: last.segments.length});
	})()`)
	fmt.Printf("[foldout] inject long summary: %s\n", iv.ToString())
	waitJSO(wv, 400)
	runJobsO(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsO(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	dumpFoldedO(wv, "LONG-SUMMARY")
	renderPNGO(wv, filepath.Join(wd, "dev", "desktop_probe", "foldout_long.png"))

	// ── 滚动条诊断：chat-messages 是否出现垂直滚动条（解释 2px 偏移）──
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var el = document.querySelector('.chat-messages');
		if (!el) return JSON.stringify({noEl: true});
		return JSON.stringify({
			clientH: el.clientHeight, scrollH: el.scrollHeight,
			clientW: el.clientWidth, scrollW: el.scrollWidth,
			overflowY: getComputedStyle ? getComputedStyle(el).overflowY : '',
			hasVScrollbar: el.scrollHeight > el.clientHeight
		});
	})()`)
	fmt.Printf("[foldout] scrollbar diag (w600+long): %s\n", iv.ToString())

	fmt.Printf("[foldout] done\n")
}
