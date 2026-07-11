package main

import (
	"fmt"
	"os"
	"time"
	"strconv"

	"github.com/hoonfeng/gwui/css"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/html5"
	"github.com/hoonfeng/gwui/webrender"
)

func main() {
	os.Setenv("CGO_ENABLED", "1")

	// 1. 加载 IDE HTML
	htmlBytes, err := os.ReadFile("F:\\syproject\\gou-ide\\cmd\\companion\\resources\\html\\shell.html")
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取shell.html失败: %v\n", err)
		os.Exit(1)
	}
	htmlStr := string(htmlBytes)

	// 2. 加载 CSS
	cssBytes, err := os.ReadFile("F:\\syproject\\gou-ide\\cmd\\companion\\resources\\css\\theme.css")
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取theme.css失败: %v\n", err)
		os.Exit(1)
	}
	cssStr := string(cssBytes)

	// 3. 解析 CSS
	tCss := time.Now()
	cssSheet := css.ParseStyleSheet(cssStr)
	fmt.Fprintf(os.Stderr, "[profile] CSS解析: %v (%d 规则)\n", time.Since(tCss), len(cssSheet.Rules))

	// 4. 解析 HTML
	tHtml := time.Now()
	doc, _ := html5.Parse(htmlStr)
	fmt.Fprintf(os.Stderr, "[profile] HTML解析: %v\n", time.Since(tHtml))
	_ = cssSheet

	// 5. 统计 DOM 节点
	domCount := 0
	countNodes(doc.Root(), &domCount)
	fmt.Fprintf(os.Stderr, "[profile] DOM节点数: %d\n", domCount)

	// 6. 使用 RecordingCanvas（快速文本估算，无 CGO）
	canvas := webrender.NewRecordingCanvas(1400, 900)
	pipeline := webrender.NewRenderPipeline(doc, canvas, 1400, 900)

	// 7. BuildRenderTree
	tBuild := time.Now()
	pipeline.View = pipeline.Bridge.BuildRenderTree(doc, 1400, 900)
	fmt.Fprintf(os.Stderr, "[profile] BuildRenderTree: %v\n", time.Since(tBuild))

	// 8. 统计渲染树节点
	objCount := 0
	countObjs(pipeline.View, &objCount)
	fmt.Fprintf(os.Stderr, "[profile] 渲染树节点数: %d\n", objCount)

	// 9. 全量 Layout（含子节点计时）
	// 先把所有节点标记为需要布局
	markDirty(pipeline.View)
	
	fmt.Fprintf(os.Stderr, "[profile] 开始布局...\n")
	tLayout := time.Now()
	pipeline.PerformLayout()
	layoutDur := time.Since(tLayout)
	fmt.Fprintf(os.Stderr, "[profile] 布局合计: %v\n", layoutDur)

	// 10. 渲染
	fmt.Fprintf(os.Stderr, "[profile] 开始渲染...\n")
	tRender := time.Now()
	pipeline.Render()
	renderDur := time.Since(tRender)
	fmt.Fprintf(os.Stderr, "[profile] 渲染: %v\n", renderDur)

	fmt.Fprintf(os.Stderr, "[profile] 首帧合计: %v\n", layoutDur+renderDur)
}

func countNodes(n dom.Node, count *int) {
	if n == nil { return }
	*count++
	for _, c := range n.Children() {
		countNodes(c, count)
	}
}

func countObjs(obj webrender.RenderObject, count *int) {
	if obj == nil { return }
	*count++
	if el, ok := obj.(webrender.RenderElement); ok {
		el.ForEachChild(func(c webrender.RenderObject) {
			countObjs(c, count)
		})
	}
}

func markDirty(obj webrender.RenderObject) {
	if box := obj.AsRenderBox(); box != nil {
		box.SetNeedsLayout(webrender.MarkContainingBlockChain)
	}
	if el, ok := obj.(webrender.RenderElement); ok {
		el.ForEachChild(func(c webrender.RenderObject) {
			markDirty(c)
		})
	}
}

func init() {
	// 让 strconv 被使用（避免编译警告）
	_ = strconv.Itoa
}
