// wr_debug — WebRender 管线自诊断工具
// 工具说明：
// 1. 无头渲染 IDE 页面到 wr_debug_output.png
// 2. 打印完整渲染树 + 元素尺寸详情
// 3. 自动检测布局和样式问题
//
//go:build debug

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hoonfeng/gwui/backend/skia"
	"github.com/hoonfeng/gwui/css"
	"github.com/hoonfeng/gwui/html5"
	"github.com/hoonfeng/gwui/webrender"
)

func main() {
	// 1. 加载 HTML + CSS
	baseDir := filepath.Join("cmd", "companion", "resources")
	htmlPath := filepath.Join(baseDir, "html", "shell.html")
	cssPath := filepath.Join(baseDir, "css", "theme.css")

	// 支持 -html <path> / -css <path> / -w <int> / -h <int> 参数
	cw, ch := 1400, 840
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-html":
			if i+1 < len(os.Args) {
				htmlPath = os.Args[i+1]
				// 默认 css 仍然加载，除非显式 -css ""
				i++
			}
		case "-css":
			if i+1 < len(os.Args) {
				cssPath = os.Args[i+1]
				i++
			}
		case "-w":
			if i+1 < len(os.Args) {
				if v, err := strconv.Atoi(os.Args[i+1]); err == nil {
					cw = v
				}
				i++
			}
		case "-h":
			if i+1 < len(os.Args) {
				if v, err := strconv.Atoi(os.Args[i+1]); err == nil {
					ch = v
				}
				i++
			}
		}
	}

	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读HTML失败: %v\n", err)
		os.Exit(1)
	}
	var cssBytes []byte
	if cssPath != "" {
		cssBytes, err = os.ReadFile(cssPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "读CSS失败: %v\n", err)
			os.Exit(1)
		}
	}

	// 2. 解析 HTML5 → DOM
	doc, err := html5.Parse(string(htmlBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析HTML失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("DOM解析: %d element + text nodes\n", len(doc.Body().Children()))

	// 3. 解析 CSS → StyleSheet（合并外部 CSS + HTML 内联 <style>）
	cssText := string(cssBytes)
	if doc != nil {
		cssText = cssText + "\n" + doc.GetStyleSheetText()
	}
	sheet := css.ParseStyleSheet(cssText)
	if sheet == nil {
		fmt.Fprintf(os.Stderr, "解析CSS失败\n")
		os.Exit(1)
	}
	fmt.Printf("CSS解析: %d 条规则\n", len(sheet.Rules))

	// 4. 创建 Skia 后端（无头渲染，不创建窗口）
	be := skia.New()
	if be == nil {
		fmt.Fprintf(os.Stderr, "Backend初始化失败\n")
		os.Exit(1)
	}

	surface := be.NewRasterSurface(cw, ch)
	if surface == nil {
		fmt.Fprintf(os.Stderr, "创建 Surface 失败\n")
		os.Exit(1)
	}
	canvas := surface.Canvas()

	// 5. 构建桥接 + 渲染树
	bridge := webrender.NewDOMBridge()
	bridge.SetStyleSheet(sheet)
	view := bridge.BuildRenderTree(doc, LayoutUnit(cw), LayoutUnit(ch))
	if view == nil {
		fmt.Fprintf(os.Stderr, "BuildRenderTree 返回 nil\n")
		os.Exit(1)
	}
	fmt.Printf("渲染树构建: %d DOM→渲染映射\n", len(bridge.DOMToRenderMap()))

	// 6. 布局
	view.Layout()

	// 7. 创建 CanvasSkia 适配器 + PaintInfo
	cs := webrender.NewCanvasSkia(canvas, be)
	if cs == nil {
		fmt.Fprintf(os.Stderr, "CanvasSkia创建失败\n")
		os.Exit(1)
	}
	cs.Clear(webrender.Color{240, 240, 240, 255}) // 浅灰背景

	pi := webrender.PaintInfo{
		Canvas:   cs,
		ClipRect: webrender.LayoutRect{0, 0, LayoutUnit(cw), LayoutUnit(ch)},
		Phase:    webrender.PaintPhaseForeground,
	}

	// 8. 绘制
	view.Paint(pi)

	// 测试：直接画一个红矩形验证 Canvas 正常工作（在 view.Paint 之后以免被覆盖）
	cs.FillRect(webrender.LayoutRect{100, 100, 50, 50}, webrender.Color{255, 0, 0, 255})
	// 测试：直接画文本
	cs.DrawText("HelloWR", 100, 200, 24, webrender.Color{0, 0, 255, 255})
	// Raster surface 不需要显式 Flush

	// 9. 保存 PNG（直接用 Surface.EncodePNG）
	pngBytes, err := surface.EncodePNG()
	if err != nil {
		fmt.Fprintf(os.Stderr, "PNG编码失败: %v\n", err)
		os.Exit(1)
	}

	outPath := "wr_debug_output.png"
	if err := os.WriteFile(outPath, pngBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入PNG失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nPNG已保存: %s (%dx%d, %d bytes)\n", outPath, cw, ch, len(pngBytes))

	fmt.Println("\n═══ 渲染树 ═══")
	printRenderTree(view, 0)

	// 12. 检查文本节点
	textCount := 0
	walkForText(view, &textCount)
	fmt.Printf("文本节点: %d\n", textCount)
}

type LayoutUnit = webrender.LayoutUnit

func printRenderTree(obj webrender.RenderObject, depth int) {
	indent := strings.Repeat("  ", depth)
	fr := obj.FrameRect()
	name := obj.RenderName()

	var bgStr, colorStr, fontSizeStr, boxStr string
	if el, ok := obj.(webrender.RenderElement); ok {
		if s := el.Style(); s != nil {
			bgStr = fmt.Sprintf("#%02x%02x%02x", s.BackgroundColor.R, s.BackgroundColor.G, s.BackgroundColor.B)
			colorStr = fmt.Sprintf("#%02x%02x%02x", s.Color.R, s.Color.G, s.Color.B)
			fontSizeStr = fmt.Sprintf("%.0f", s.FontSize)
			// 显示盒模型：padding/border + width/box-sizing
			widthStr := "auto"
			if !s.Width.IsAuto {
				widthStr = fmt.Sprintf("%.0f", s.Width.Value)
				if s.Width.IsPercent {
					widthStr += "%"
				}
			}
			boxStr = fmt.Sprintf(" [w=%s pad=%.0f/%.0f/%.0f/%.0f bw=%.0f/%.0f/%.0f/%.0f bs=%s]",
				widthStr,
				s.Padding.Top, s.Padding.Right, s.Padding.Bottom, s.Padding.Left,
				s.BorderWidth.Top, s.BorderWidth.Right, s.BorderWidth.Bottom, s.BorderWidth.Left,
				boxSizingName(s.BoxSizing))
		}
	}

	textStr := ""
	if txt, ok := obj.(*webrender.RenderText); ok {
		t := txt.Text()
		if len(t) > 50 {
			t = t[:50] + "..."
		}
		textStr = fmt.Sprintf(" %q", t)
	}

	fmt.Printf("%s%s frame=(%.0f,%.0f %.0fx%.0f) bg=%s color=%s f%s%s%s\n",
		indent, name,
		fr.X, fr.Y, fr.W, fr.H,
		bgStr, colorStr, fontSizeStr, textStr, boxStr)

	if el, ok := obj.(webrender.RenderElement); ok {
		for child := el.FirstChild(); child != nil; child = child.NextSibling() {
			printRenderTree(child, depth+1)
		}
	}
}

func walkForText(obj webrender.RenderObject, count *int) {
	if _, ok := obj.(*webrender.RenderText); ok {
		*count++
	}
	if el, ok := obj.(webrender.RenderElement); ok {
		for child := el.FirstChild(); child != nil; child = child.NextSibling() {
			walkForText(child, count)
		}
	}
}

func boxSizingName(bs webrender.BoxSizingType) string {
	switch bs {
	case webrender.BoxSizingContentBox:
		return "content-box"
	case webrender.BoxSizingBorderBox:
		return "border-box"
	}
	return "?"
}
