// Command switch_probe renders a switch (toggle) control with ::after
// pseudo-element thumb through the full wb-ui pipeline and checks that the
// pseudo-element box exists and has geometry — the previous engine never
// generated ::before/::after boxes, so the thumb was missing.
package main

import (
	"fmt"
	"os"

	"wb-ui/dom"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

const html = `<!DOCTYPE html><html><head><style>
.switch { display: inline-flex; align-items: center; gap: 8px; }
.switch input { display: none; }
.switch .track {
  width: 34px; height: 18px; border-radius: 9px;
  background: #333; border: 1px solid #555;
  position: relative; display: inline-block;
}
.switch .track::after {
  content: ""; position: absolute; top: 2px; left: 2px;
  width: 12px; height: 12px; border-radius: 50%;
  background: #888;
}
</style></head><body>
<label class="switch"><input type="checkbox" id="switch-1"><span class="track" id="track1"></span>自动审核</label>
</body></html>`

func main() {
	wv := webkit.NewWebView()
	if err := wv.LoadHTML(html); err != nil {
		fmt.Println("LoadHTML err:", err)
		os.Exit(1)
	}
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		fmt.Println("FAIL: no render view")
		os.Exit(1)
	}
	state := rv.LayoutState()

	// 遍历 render tree，找 .track（id=track1）及其伪元素子 box。
	var track, after rendering.RenderObject
	var walk func(ro rendering.RenderObject, depth int)
	walk = func(ro rendering.RenderObject, depth int) {
		if ro == nil {
			return
		}
		name := ro.RenderName()
		var info string
		if n := ro.Node(); n != nil {
			info = "node=" + n.NodeName()
			if e, ok := n.(*dom.Element); ok {
				info += " id=" + e.GetAttribute("id") + " class=" + e.GetAttribute("class")
			}
		} else {
			info = "ANON"
		}
		if depth <= 8 {
			fmt.Printf("%*s%s %s\n", depth*2, "", name, info)
		}
		if n := ro.Node(); n != nil {
			if e, ok := n.(*dom.Element); ok && e.GetAttribute("id") == "track1" {
				track = ro
			}
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	walk(rendering.RenderObject(rv), 0)

	// 修正：after 查找（track 之后的匿名子 box）在 dump 后单独做
	if track != nil {
		// track 的最后一个匿名子 box 是 ::after
		for c := track.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Node() == nil {
				after = c
			}
		}
	}

	// 修正：after 查找（track 之后的匿名子 box）在 dump 后单独做
	if track != nil {
		// track 的最后一个匿名子 box 是 ::after
		for c := track.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Node() == nil {
				after = c
			}
		}
	}

	if track == nil {
		fmt.Println("FAIL: .track box not found")
		os.Exit(1)
	}
	tg := state.GeometryForBox(track.LayoutBox())
	fmt.Printf("track: x=%.0f y=%.0f w=%.0f h=%.0f\n", tg.Left(), tg.Top(), tg.BorderBoxWidth(), tg.BorderBoxHeight())

	if after == nil {
		fmt.Println("FAIL: ::after pseudo-element box not generated (thumb missing)")
		os.Exit(1)
	}
	ag := state.GeometryForBox(after.LayoutBox())
	fmt.Printf("::after thumb: x=%.0f y=%.0f w=%.0f h=%.0f\n", ag.Left(), ag.Top(), ag.BorderBoxWidth(), ag.BorderBoxHeight())
	if ag.BorderBoxWidth() < 10 || ag.BorderBoxHeight() < 10 {
		fmt.Printf("FAIL: thumb too small (%fx%f), expected ~12x12\n", ag.BorderBoxWidth(), ag.BorderBoxHeight())
		os.Exit(1)
	}
	fmt.Println("OK: switch thumb rendered")
}
