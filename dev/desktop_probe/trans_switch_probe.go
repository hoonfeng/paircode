// Command trans_switch_probe 自动化验证：checked 切换 + rebuild 后，
// 开关 ::after 伪元素（圆点）的 left 是否被 transition 引擎渐进插值。
// 不依赖手动操作——直接驱动 ApplyAnimations 时间轴并打印伪元素 left。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"wb-ui/dom"
	"wb-ui/html5"
	"wb-ui/page"
	"wb-ui/rendering"
)

func main() {
	log.SetFlags(0)
	wd, _ := os.Getwd()
	data, err := os.ReadFile(filepath.Join(wd, "cmd", "desktop", "web-ui", "interact_test.html"))
	if err != nil {
		log.Fatalf("cannot read interact_test.html: %v", err)
	}
	p := page.NewPage(nil)
	fr := p.MainFrame()
	if err := fr.LoadHTML(string(data)); err != nil {
		log.Fatalf("load html: %v", err)
	}

	// 首次构建：settle（记录伪元素 left=2px 为初始值）
	rendering.AnimationTime = 0
	rendering.ApplyAnimations(fr.RenderView())
	fmt.Println("=== 初始（未选中） ===")
	dumpPseudoLeft(fr.RenderView())

	// 找到第一个 switch 的 checkbox
	var input *dom.Element
	for _, el := range fr.Document().GetElementsByTagName("input") {
		if el.GetAttribute("type") == "checkbox" {
			input = el
			break
		}
	}
	if input == nil {
		log.Fatal("no switch checkbox found")
	}
	inEl, ok := html5.ToInputElement(input)
	if !ok {
		log.Fatal("input is not an InputElement")
	}
	fmt.Printf("\n=== 切换 checked: %v -> %v ===", inEl.Checked(), true)
	inEl.SetChecked(true)
	fr.RebuildRenderTree()

	fmt.Println("\n=== 重建后（未驱动动画，目标应为 left=18px） ===")
	dumpPseudoLeft(fr.RenderView())

	// 驱动 transition 时间轴
	fmt.Println("\n=== 驱动 ApplyAnimations 时间轴（每帧 Layout 后所有伪元素几何） ===")
	for t := 0.0; t <= 0.55; t += 0.1 {
		rendering.AnimationTime = t
		inflight := rendering.ApplyAnimations(fr.RenderView())
		if fr.View() != nil {
			fr.View().Layout()
		}
		left, _, found := firstPseudoLeft(fr.RenderView())
		geoms := allPseudoGeometry(fr.RenderView())
		if found {
			fmt.Printf("t=%.2f inFlight=%-5v first-style-left=%s geoms=%s\n", t, inflight, left, geoms)
		} else {
			fmt.Printf("t=%.2f inFlight=%-5v first-style-left=<NONE>\n", t, inflight)
		}
	}

	// 反向切换：关
	fmt.Println("\n=== 反向切换: 取消选中 ===")
	inEl.SetChecked(false)
	fr.RebuildRenderTree()
	for t := 0.55; t <= 1.15; t += 0.1 {
		rendering.AnimationTime = t
		inflight := rendering.ApplyAnimations(fr.RenderView())
		if fr.View() != nil {
			fr.View().Layout()
		}
		left, _, found := firstPseudoLeft(fr.RenderView())
		geoms := allPseudoGeometry(fr.RenderView())
		if found {
			fmt.Printf("t=%.2f inFlight=%-5v first-style-left=%s geoms=%s\n", t, inflight, left, geoms)
		} else {
			fmt.Printf("t=%.2f inFlight=%-5v first-style-left=<NONE>\n", t, inflight)
		}
	}
}

type pseudoGeom struct{ x, w float64 }

func allPseudoGeometry(rv *rendering.RenderView) string {
	out := ""
	count := 0
	if rv == nil {
		return "<nil>"
	}
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if st := o.Style(); st != nil {
			if s := st.Properties["left"]; s != "" {
				link := "L"
				if o.LayoutBox() == nil {
					link = "X"
				}
				x, _, w, _, ok := rendering.BoxGeometry(o)
				if ok {
					out += fmt.Sprintf("[%s%s l=%.1f g=%.1f]", link, s, w, x)
				} else {
					out += fmt.Sprintf("[%s%s no-geom]", link, s)
				}
				count++
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	if count == 0 {
		return "<none>"
	}
	return out
}

// dumpPseudoLeft 打印渲染树中所有带 Properties["left"] 的伪元素 box。
func dumpPseudoLeft(rv *rendering.RenderView) {
	if rv == nil {
		fmt.Println("  <nil render view>")
		return
	}
	count := 0
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if st := o.Style(); st != nil {
			if s := st.Properties["left"]; s != "" {
				lb := ""
				if o.LayoutBox() != nil {
					lb = "linked"
				} else {
					lb = "NO-LINK"
				}
				x, _, w, _, ok := rendering.BoxGeometry(o)
				ge := "-"
				if ok {
					ge = fmt.Sprintf("%.1f", x)
				}
				fmt.Printf("  pseudo-left=%-8s dur=%.2f %s drawnX=%-6s w=%.0f\n", s, st.TransitionDuration, lb, ge, w)
				count++
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	if count == 0 {
		fmt.Println("  <无带 left 的伪元素 box>")
	}
}

func firstPseudoLeft(rv *rendering.RenderView) (left string, dur float64, found bool) {
	if rv == nil {
		return "", 0, false
	}
	var walk func(o rendering.RenderObject) bool
	walk = func(o rendering.RenderObject) bool {
		if o == nil {
			return false
		}
		if st := o.Style(); st != nil {
			if s := st.Properties["left"]; s != "" {
				left = s
				dur = st.TransitionDuration
				found = true
				return true
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			if walk(c) {
				return true
			}
		}
		return false
	}
	walk(rendering.RenderObject(rv))
	return
}
