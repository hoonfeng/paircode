// Command wheel_scroll_probe simulates the REAL desktop wheel-scroll path
// (app/host.go EventScroll): HitTestScrollContainer → BoxScrollOffset →
// VerticalScrollbarMetrics(MaxScroll) clamp → SetBoxScrollOffset.
//
// The previous probes wrote BoxScrollOffset directly (unclamped), which
// bypasses the MaxScroll limit that real wheel scrolling is clamped to. If
// BoxContentSize under-reports a container's content height, MaxScroll is
// too small and wheel scrolling can never reveal content that was clipped
// at scroll 0 — "初始被裁切的内容永远不会展示".
//
// This probe uses the SAME viewport as desktop (1280x800 via
// wv.Resize after LoadHTML, mirroring app.NewHost) so layout matches.
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/desktopbridge"
	"github.com/hoonfeng/paircode/internal/server/handler"
)

func setupLoaders3(wv *webkit.WebView, distDir string) {
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
				if err != nil {
					return "", err
				}
				re := regexp.MustCompile(`\[data-v-[a-f0-9]+\]`)
				return re.ReplaceAllString(string(data), ""), nil
			}
		}
	}
}

func asRB3(o rendering.RenderObject) *rendering.RenderBox {
	switch v := o.(type) {
	case *rendering.RenderBox:
		return v
	case *rendering.RenderBlock:
		return &v.RenderBox
	case *rendering.RenderBlockFlow:
		return &v.RenderBlock.RenderBox
	}
	return nil
}

func clsOf3(rb *rendering.RenderBox) string {
	if el, ok := rb.Node().(*dom.Element); ok {
		return el.GetAttribute("class")
	}
	return ""
}

func saveCanvas3(c *graphics.Canvas, path string) {
	w, h := c.Width(), c.Height()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	px := c.Pixels()
	for i := 0; i < w*h; i++ {
		img.Pix[i*4+0] = px[i*4+0]
		img.Pix[i*4+1] = px[i*4+1]
		img.Pix[i*4+2] = px[i*4+2]
		img.Pix[i*4+3] = px[i*4+3]
	}
	f, err := os.Create(path)
	if err != nil {
		log.Printf("save %s: %v", path, err)
		return
	}
	defer f.Close()
	png.Encode(f, img)
	log.Printf("saved %s (%dx%d)", path, w, h)
}

type scrollInfo struct {
	cls      string
	box      *rendering.RenderBox
	viewW    float64
	viewH    float64
	totalW   float64
	totalH   float64
	maxScroll float64
	ok       bool
	x, y     float64
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	wd, _ := os.Getwd()
	// cwd 可能是 dev/desktop_probe（go run 在该目录执行），向上找项目根
	root := wd
	for i := 0; i < 4; i++ {
		if _, err := os.Stat(filepath.Join(root, "cmd", "companion", "web-ui", "dist")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}
	distDir := filepath.Join(root, "cmd", "companion", "web-ui", "dist")
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
	setupLoaders3(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	// desktop 视口：app.NewHost(wv, 1280, 800) 在 LoadHTML 之后 Resize。
	wv.Resize(1280, 800)
	log.Printf("viewport = %dx%d", wv.Width(), wv.Height())

	// ── 打开工作区：模拟 desktop 用户打开文件夹（core.LoadLastProject 无历史时
	//    Root 为空 → 消息存储未初始化 → 列表为空）。打开后 SetWorkspaceRoot
	//    初始化 JSONL 存储，前端 /api/conversations 才有真实数据。 ──
	wsRoot := "F:\\syproject\\gou-ide"
	core.SetProject(wsRoot)
	core.PersistWorkspace()
	if mgr := handler.AgentMgr; mgr != nil {
		mgr.SetWorkspaceRoot(core.Root())
		log.Printf("workspace root = %s, store = %v", core.Root(), mgr.Store() != nil)
	}

	// ── 创建会话 + 注入真实消息（模拟已完成的对话：user 提问 + assistant 长答案，
	//    含 markdown 代码块，内容超高以便产生大滚动范围） ──
	convID := "conv_probe_wheel"
	store := handler.AgentMgr.Store()
	if store == nil {
		log.Fatal("store nil")
	}
	if _, err := store.GetConversation(convID); err != nil {
		if err := store.CreateConversation(convID, "滚动复现会话", core.Root()); err != nil {
			log.Printf("CreateConversation: %v", err)
		}
	}
	// 清掉旧消息，写入 2 条长消息
	store.ReplaceHistory(convID, nil)
	var longBody strings.Builder
	longBody.WriteString("请解释一下滚动容器的渲染管线，并给出完整的实现示例。\n\n")
	for i := 0; i < 6; i++ {
		longBody.WriteString(fmt.Sprintf("### 第 %d 节：滚动裁剪与内容平移\n", i+1))
		longBody.WriteString("滚动容器（overflow: auto）的内容区是一个独立的层。当用户滚动时，视口内的内容应当精确平移 scrollOffset 的距离，而视口外的内容被裁剪。\n\n")
		longBody.WriteString("```go\n")
		longBody.WriteString("func (p *Painter) paintScrollContent(box *RenderBox) {\n")
		longBody.WriteString("    off := box.ScrollOffset()\n")
		longBody.WriteString("    p.save()\n")
		longBody.WriteString("    p.clipRect(box.ContentRect())\n")
		longBody.WriteString("    p.translate(0, -off.Y) // 内容上移显示后续部分\n")
		longBody.WriteString("    p.paintChildren(box)\n")
		longBody.WriteString("    p.restore()\n")
		longBody.WriteString("}\n")
		longBody.WriteString("```\n\n")
		longBody.WriteString("关键点：内容必须整体平移，裁剪边界保持不变，这样滚动后原来被裁切的内容才能进入视口。\n\n")
	}
	if err := store.AppendUserMessage(convID, "滚动容器渲染管线说明"); err != nil {
		log.Printf("AppendUserMessage: %v", err)
	}
	if err := store.AppendMessage(convID, agent.Message{Role: agent.RoleAssistant, Content: longBody.String()}, nil); err != nil {
		log.Printf("AppendMessage: %v", err)
	}
	log.Printf("injected conv=%s msgs(long=%d bytes)", convID, len(longBody.String()))

	// 诊断 1：Go 侧 store 直接读取
	if msgs, total, err := store.LoadLatest(convID, 50); err == nil {
		log.Printf("store.LoadLatest: n=%d total=%d", len(msgs), total)
		for i, m := range msgs {
			log.Printf("  msg[%d] idx=%d role=%s len=%d ts=%s", i, m.Idx, m.Message.Role, len(m.Message.Content), m.Timestamp)
		}
	} else {
		log.Printf("store.LoadLatest err: %v", err)
	}

	// ── JS 导航到会话（hash 路由，触发 ChatView watch(convId) → loadMessages） ──
	wsId := "default"
	if root := core.Root(); root != "" {
		wsId = strings.ReplaceAll(filepath.Base(root), " ", "%20")
	}
	_, _ = wv.EvalJS(fmt.Sprintf(`location.hash = '#/workspace/%s/chat/%s'`, wsId, convID))
	log.Printf("navigated to #/workspace/%s/chat/%s", wsId, convID)

	// 诊断 2：JS 侧 fetch（模拟前端 loadMessages 的调用路径）
	if v, err := wv.EvalJS(`(function(){
		try {
			return fetch('/api/conversations/conv_probe_wheel/messages?limit=50')
				.then(function(r){ return r.text(); })
				.then(function(t){ window.__probeMsgsText = t; return t.length; })
				.catch(function(e){ window.__probeMsgsText = 'ERR:' + e; return -1; });
		} catch(e) { return 'SYNC_ERR:' + e; }
	})()`); err == nil {
		log.Printf("fetch async pending: %v", v.ToString())
	} else {
		log.Printf("fetch eval err: %v", err)
	}

	// 等 DOM 稳定（与 chat_scroll_probe 相同：msg-item 连续 3 次相同）
	stable := false
	last := -1
	same := 0
	for i := 0; i < 20 && !stable; i++ {
		v, _ := wv.EvalJS(`document.querySelectorAll('.msg-item').length`)
		msgCount := int(v.ToNumber())
		log.Printf("poll msg-item=%d (same=%d)", msgCount, same)
		if msgCount == last && msgCount > 0 {
			same++
			if same >= 3 {
				stable = true
			}
		} else {
			same = 0
		}
		last = msgCount
		if !stable {
			_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 1000); })`)
		}
	}
	// 诊断 2b：fetch 结果
	if v, err := wv.EvalJS(`(window.__probeMsgsText || '').length + ':' + String(window.__probeMsgsText || '').slice(0, 200)`); err == nil {
		log.Printf("probe fetch result: %s", v.ToString())
	}
	// 诊断 2c：hash 与 DOM 状态
	if v, err := wv.EvalJS(`JSON.stringify({hash: location.hash, chatMsgs: document.querySelectorAll('.chat-messages').length, convName: (document.querySelector('.cv-conv-name')||{}).textContent || ''})`); err == nil {
		log.Printf("dom state: %s", v.ToString())
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	// ── 1) 枚举所有垂直可滚动容器，报告 viewH/totalH/MaxScroll ──
	var infos []scrollInfo
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if rb := asRB3(o); rb != nil && rb.Style() != nil {
			cls := clsOf3(rb)
			if cls != "" {
				vm := rendering.VerticalScrollbarMetrics(rv, rb)
				if vm.OK {
					infos = append(infos, scrollInfo{
						cls: cls, box: rb,
						viewH:  vm.ViewLen, totalH: vm.TotalLen, maxScroll: vm.MaxScroll, ok: vm.OK,
						x: rb.X(), y: rb.Y(),
					})
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	sort.SliceStable(infos, func(i, j int) bool { return infos[i].y < infos[j].y })
	fmt.Println("=== VERTICAL SCROLL CONTAINERS (1280x800) ===")
	fmt.Printf("%-28s %8s %8s %8s %8s   %s\n", "class", "viewH", "totalH", "maxScroll", "boxY", "OK")
	for _, in := range infos {
		fmt.Printf("%-28s %8.0f %8.0f %8.0f %8.0f   %v\n", in.cls, in.viewH, in.totalH, in.maxScroll, in.y, in.ok)
	}

	// ── 2) 模拟 desktop wheel 路径：HitTestScrollContainer → clamp → SetBoxScrollOffset ──
	// 光标放在 conv-list 与 chat-messages 中央，模拟用户把鼠标悬停其上滚轮。
	simWheel := func(label string, cssX, cssY float64, wheelDelta int) {
		fmt.Printf("\n=== WHEEL SIM: %s at (%.0f,%.0f) delta=%d ===\n", label, cssX, cssY, wheelDelta)
		box := rv.HitTestScrollContainer(cssX, cssY)
		if box == nil {
			fmt.Println("  HitTestScrollContainer -> nil (无滚动容器)")
			return
		}
		bname := "?"
		if el, ok := box.Node().(*dom.Element); ok {
			bname = el.LocalName() + "." + el.GetAttribute("class")
		}
		sx, sy := rv.BoxScrollOffset(box)
		deltaY := -wheelDelta
		newSy := int(sy) + deltaY
		if newSy < 0 {
			newSy = 0
		}
		if vm := rendering.VerticalScrollbarMetrics(rv, box); vm.OK {
			maxY := int(vm.MaxScroll)
			if newSy > maxY {
				newSy = maxY
			}
			fmt.Printf("  hit-box=%s cur=(%.0f,%.0f) -> newSy=%d (MaxScroll=%.0f)\n", bname, sx, sy, newSy, vm.MaxScroll)
		} else {
			newSy = 0
			fmt.Printf("  hit-box=%s VerticalScrollbarMetrics !OK -> newSy=0 (滚轮被钳死!)\n", bname)
		}
		rv.SetBoxScrollOffset(box, 0, float64(newSy))
		c := graphics.NewCanvas(1280, 800)
		rendering.Paint(rv, c, rendering.Rect{X: 0, Y: 0, Width: 1280, Height: 800})
		outDir := filepath.Join(root, "dev", "desktop_probe")
		saveCanvas3(c, filepath.Join(outDir, fmt.Sprintf("wheel_%s.png", label)))
	}

	// conv-list：桌面窗口 conv 面板区域（x≈300-340 附近，y≈100-400）
	simWheel("conv", 320, 200, 360)
	// chat-messages：右侧消息区（x≈600，y≈500）
	simWheel("chat", 700, 500, 360)
	// chat 滚到底：连续滚 10 次到达 MaxScroll
	simWheel("chat_bottom", 700, 500, 360*10)

	fmt.Println("\ndone")
}
