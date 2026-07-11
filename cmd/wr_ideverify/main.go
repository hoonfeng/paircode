// wr_ideverify — IDE UI 渲染验证工具
// 用 RecordingCanvas 替代 Skia 后端，完整记录绘制调用
// 然后按 WebKit 规范验证布局 + 绘制的正确性
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/gwui/css"
	"github.com/hoonfeng/gwui/html5"
	"github.com/hoonfeng/gwui/webrender"
)

func main() {
	// 默认 HTML 文件
	htmlPath := "F:\\syproject\\GWui\\ide_test.html"
	cssPath := "F:\\syproject\\gou-ide\\cmd\\companion\\resources\\css\\theme.css"

	if len(os.Args) > 1 {
		htmlPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		cssPath = os.Args[2]
	}

	// 1. 读取 HTML + CSS
	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读HTML失败: %v\n", err)
		os.Exit(1)
	}
	var cssBytes []byte
	if cssPath != "" && cssPath != "none" {
		cssBytes, err = os.ReadFile(cssPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "读CSS失败: %v\n", err)
			os.Exit(1)
		}
	}

	// 2. 解析 HTML
	doc, err := html5.Parse(string(htmlBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析HTML失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("DOM解析: %d 个元素+文本节点\n", len(doc.Body().Children()))

	// 3. 解析 CSS
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

	// 4. 创建 RecordingCanvas（模拟渲染，不依赖 Skia DLL）
	cw, ch := webrender.LayoutUnit(1400), webrender.LayoutUnit(840)
	rc := webrender.NewRecordingCanvas(cw, ch)

	// 5. 构建桥接 + 渲染树（与 wr_debug 一致：先设 stylesheet，再 BuildRenderTree）
	bridge := webrender.NewDOMBridge()
	bridge.SetStyleSheet(sheet)
	view := bridge.BuildRenderTree(doc, cw, ch)
	if view == nil {
		fmt.Fprintf(os.Stderr, "BuildRenderTree 返回 nil\n")
		os.Exit(1)
	}
	fmt.Printf("渲染树构建: %d DOM→渲染映射\n", len(bridge.DOMToRenderMap()))

	// 6. 布局 + 绘制（⚠️ 不能用 view.Paint()——它走 Layer 路径不完整）
	//    必须模拟 pipeline.paintRecursive 的 render tree walk
	view.Layout()

	// 使用 paintRecursive 逻辑：
	// 先 Clear 背景，然后递归遍历 render tree
	backgroundColor := webrender.Color{255, 255, 255, 255} // 默认白
	if view.Style() != nil && view.Style().BackgroundColor.A > 0 {
		backgroundColor = view.Style().BackgroundColor
	}
	rc.Clear(backgroundColor)

	paintInfo := webrender.NewPaintInfo(rc, webrender.LayoutRect{0, 0, cw, ch})

	// 模拟 pipeline.paintRecursive 的 render tree walk
	paintRecursive(view, webrender.LayoutPoint{}, paintInfo, rc)

	paintLog := rc.PaintLog()

	// 保存绘制日志到文件
	logPath := "ide_paint_log.txt"
	if err := os.WriteFile(logPath, []byte(paintLog), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入绘制日志失败: %v\n", err)
	} else {
		fmt.Printf("\n绘制日志已保存: %s (%d 行)\n", logPath, strings.Count(paintLog, "\n"))
	}

	// 9. 布局坐标提取与验证
	fmt.Println("\n═══════════════════════════════════════")
	fmt.Println("  布局验证报告")
	fmt.Println("═══════════════════════════════════════")
	layoutIssues := analyzeLayout(view)
	fmt.Printf("\n布局问题: %d\n", len(layoutIssues))
	for i, issue := range layoutIssues {
		fmt.Printf("  #%d: %s\n", i+1, issue)
	}

	// 10. 绘制日志分析
	fmt.Println("\n═══════════════════════════════════════")
	fmt.Println("  绘制日志分析")
	fmt.Println("═══════════════════════════════════════")
	paintIssues := analyzePaintLog(paintLog, view)
	fmt.Printf("\n绘制问题: %d\n", len(paintIssues))
	for i, issue := range paintIssues {
		fmt.Printf("  #%d: %s\n", i+1, issue)
	}

	// 11. 保存完整报告
	reportPath := "ide_verify_report.txt"
	var report strings.Builder
	report.WriteString("═══════════════════════════════════════════\n")
	report.WriteString("  IDE UI 渲染验证报告\n")
	report.WriteString("═══════════════════════════════════════════\n\n")
	report.WriteString(fmt.Sprintf("HTML: %s\n", htmlPath))
	report.WriteString(fmt.Sprintf("CSS:  %s\n", filepath.Base(cssPath)))
	report.WriteString(fmt.Sprintf("尺寸: %dx%d\n\n", cw, ch))

	report.WriteString("--- 布局问题 ---\n")
	for i, issue := range layoutIssues {
		report.WriteString(fmt.Sprintf("%d. %s\n", i+1, issue))
	}

	report.WriteString("\n--- 绘制问题 ---\n")
	for i, issue := range paintIssues {
		report.WriteString(fmt.Sprintf("%d. %s\n", i+1, issue))
	}

	report.WriteString(fmt.Sprintf("\n--- 总计 ---\n布局问题: %d\n绘制问题: %d\n", len(layoutIssues), len(paintIssues)))
	if err := os.WriteFile(reportPath, []byte(report.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入报告失败: %v\n", err)
	} else {
		fmt.Printf("\n验证报告已保存: %s\n", reportPath)
	}

	// 12. 打印渲染树小结
	fmt.Println("\n═══════════════════════════════════════")
	fmt.Println("  渲染树关键节点")
	fmt.Println("═══════════════════════════════════════")
	printKeyNodes(view, 0, 3)
}

// analyzeLayout 分析布局问题
// 对比 WebKit 行为，检测布局异常
func analyzeLayout(view webrender.RenderObject) []string {
	var issues []string

	// 收集所有 RenderElement
	var elements []webrender.RenderElement
	collectElements(view, &elements)

	// 检查每个元素的布局
	for _, el := range elements {
		fr := el.FrameRect()
		parent := el.Parent()
		s := el.Style()
		var pf webrender.LayoutRect
		if parent != nil {
			pf = parent.FrameRect()
		}

		// 1. 检查子元素是否超出父容器（未应用 overflow:hidden 时只是记录）
		if parent != nil {
			if pf.W > 0 && fr.W > pf.W+1 && !isDividerLine(el) {
				// 宽度溢出父容器
				issues = append(issues, fmt.Sprintf("溢出: %s sub=%s(%.0fx%.0f) > parent=%s(%.0fx%.0f), diff=%.0f",
					shortName(el), el.RenderName(), fr.W, fr.H,
					parent.RenderName(), pf.W, pf.H, fr.W-pf.W))
			}
		}

		// 2. 检查负坐标
		if fr.X < -5 || fr.Y < -5 {
			issues = append(issues, fmt.Sprintf("负坐标: %s sub=%s pos=(%.0f,%.0f)",
				shortName(el), el.RenderName(), fr.X, fr.Y))
		}

		// 3. 检查零尺寸内容节点（非正常的空节点）
		if fr.W == 0 && fr.H == 0 && s != nil && s.BackgroundColor.A > 0 {
			issues = append(issues, fmt.Sprintf("零尺寸带背景: %s sub=%s bg=#%02x%02x%02x",
				shortName(el), el.RenderName(),
				s.BackgroundColor.R, s.BackgroundColor.G, s.BackgroundColor.B))
		}

		// 4. 检查兄弟节点 width 不一致（flex 列中同层应有相同宽度）
		if parent != nil {
			siblings := getChildFlexCount(parent)
			if siblings > 1 {
				// 只检查 flex-direction:column 的直接子元素宽度
				if pf.W > 0 && fr.W != pf.W && fr.W > 0 && !isSpacer(el) {
					// 非 stretch 元素在 flex 中宽度可能不同，仅记录过大偏差
					if fr.W > pf.W+2 || fr.W < pf.W-2 {
						issues = append(issues, fmt.Sprintf("宽度偏差: %s sub=%s w=%.0f 父容器=%.0f (%.0f)",
							shortName(el), el.RenderName(), fr.W, pf.W, fr.W-pf.W))
					}
				}
			}
		}

		// 5. 检查 text-align:center 的文本是否居中
		if s != nil && s.TextAlign == webrender.TextAlignCenter {
			checkTextCenter(el, &issues)
		}
	}

	return issues
}

// analyzePaintLog 分析绘制日志
func analyzePaintLog(log string, view webrender.RenderObject) []string {
	var issues []string
	lines := strings.Split(strings.TrimRight(log, "\n"), "\n")

	// 检查 Clear 颜色
	for _, line := range lines {
		// 检查是否有任何绘制调用
		if strings.Contains(line, "FillRect") || strings.Contains(line, "StrokeRect") ||
			strings.Contains(line, "DrawRoundRect") || strings.Contains(line, "DrawText") {
			break
		}
	}

	// 统计绘制调用类型
	fillCount := 0
	textCount := 0
	saveCount := 0
	restoreCount := 0
	translateCount := 0
	for _, line := range lines {
		if strings.Contains(line, "FillRect") {
			fillCount++
		}
		if strings.Contains(line, "DrawText") || strings.Contains(line, "DrawTextWithWeight") {
			textCount++
		}
		if strings.Contains(line, "Save") {
			saveCount++
		}
		if strings.Contains(line, "Restore") {
			restoreCount++
		}
		if strings.Contains(line, "Translate") {
			translateCount++
		}
	}

	if saveCount != restoreCount {
		issues = append(issues, fmt.Sprintf("Save/Restore 不匹配: Save=%d Restore=%d", saveCount, restoreCount))
	}

	// 检查最小绘制数量
	if fillCount < 10 {
		issues = append(issues, fmt.Sprintf("绘制调用过少: FillRect=%d (预期 10+)", fillCount))
	}
	if textCount < 10 {
		issues = append(issues, fmt.Sprintf("文本绘制调用过少: DrawText=%d (预期 10+)", textCount))
	}

	return issues
}

// ── 辅助函数 ──

func collectElements(obj webrender.RenderObject, out *[]webrender.RenderElement) {
	if el, ok := obj.(webrender.RenderElement); ok {
		*out = append(*out, el)
		for child := el.FirstChild(); child != nil; child = child.NextSibling() {
			collectElements(child, out)
		}
	}
}

func isDividerLine(el webrender.RenderElement) bool {
	name := el.RenderName()
	return strings.Contains(name, "vdivider") || strings.Contains(name, "hdivider")
}

func isSpacer(el webrender.RenderElement) bool {
	name := el.RenderName()
	return strings.Contains(name, "spacer") || strings.Contains(name, "Spacer")
}

func shortName(el webrender.RenderElement) string {
	s := el.Style()
	if s == nil {
		return "?"
	}
	// 尝试从渲染名推断
	name := el.RenderName()
	if len(name) > 25 {
		name = name[:22] + "..."
	}
	return name
}

func getChildFlexCount(parent webrender.RenderElement) int {
	count := 0
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(webrender.RenderElement); ok {
			count++
		}
	}
	return count
}

func checkTextCenter(el webrender.RenderElement, issues *[]string) {
	// 简单检查：寻找文本子节点
	// 如果文本节点在父容器中不是居中位置，标记
	fr := el.FrameRect()
	for child := el.FirstChild(); child != nil; child = child.NextSibling() {
		if txt, ok := child.(*webrender.RenderText); ok {
			tf := txt.FrameRect()
			centerX := (fr.W - tf.W) / 2
			if tf.X > centerX+5 || tf.X < centerX-5 {
				*issues = append(*issues, fmt.Sprintf("文本未居中: %q (文本x=%.0f, 期望中心x=%.0f)",
					shortText(txt.Text()), tf.X, centerX))
			}
		}
	}
}

func shortText(t string) string {
	if len(t) > 30 {
		return t[:27] + "..."
	}
	return t
}

func printKeyNodes(obj webrender.RenderObject, depth int, maxDepth int) {
	if depth > maxDepth {
		return
	}
	indent := strings.Repeat("  ", depth)
	fr := obj.FrameRect()
	name := obj.RenderName()

	var extra string
	if el, ok := obj.(webrender.RenderElement); ok {
		if s := el.Style(); s != nil {
			extra = fmt.Sprintf(" bg=#%02x%02x%02x w=%.0f",
				s.BackgroundColor.R, s.BackgroundColor.G, s.BackgroundColor.B, s.Width.Value)
		}
	}
	if txt, ok := obj.(*webrender.RenderText); ok {
		t := txt.Text()
		if len(t) > 40 {
			t = t[:37] + "..."
		}
		extra = fmt.Sprintf(" %q", t)
	}

	fmt.Printf("%s%s frame=(%.0f,%.0f %.0fx%.0f)%s\n",
		indent, name, fr.X, fr.Y, fr.W, fr.H, extra)

	if el, ok := obj.(webrender.RenderElement); ok {
		for child := el.FirstChild(); child != nil; child = child.NextSibling() {
			printKeyNodes(child, depth+1, maxDepth)
		}
	}
}

// paintRecursive 模拟 pipeline 的绘制遍历（render tree walk）
// 复制自 renderpipeline.go 的 paintRecursive 逻辑
func paintRecursive(obj webrender.RenderObject, offset webrender.LayoutPoint, info webrender.PaintInfo, rc *webrender.RecordingCanvas) {
	// 处理文本节点
	if rt := obj.AsRenderText(); rt != nil {
		rc.Save()
		fr := rt.FrameRect()
		rc.Translate(offset.X+fr.X, offset.Y+fr.Y)
		rt.Paint(info)
		rc.Restore()
		return
	}

	// 处理非 RenderBox 的 RenderElement（如 RenderInline）
	if el, ok := obj.(webrender.RenderElement); ok && obj.AsRenderBox() == nil {
		el.ForEachChild(func(child webrender.RenderObject) {
			paintRecursive(child, offset, info, rc)
		})
		return
	}

	box := obj.AsRenderBox()
	if box == nil {
		return
	}

	rc.Save()

	// 应用偏移
	boxFrame := box.FrameRect()
	totalOffset := webrender.LayoutPoint{
		X: offset.X + boxFrame.X,
		Y: offset.Y + boxFrame.Y,
	}

	// 如果有 Layer，应用滚动偏移
	if box.Layer() != nil {
		layer := box.Layer()
		totalOffset.X -= layer.ScrollLeft()
		totalOffset.Y -= layer.ScrollTop()
	}

	rc.Translate(totalOffset.X, totalOffset.Y)

	// 裁剪
	style := box.Style()
	var clipRect webrender.LayoutRect
	if style != nil {
		needsClip := style.OverflowY == webrender.OverflowHidden || style.OverflowY == webrender.OverflowAuto || style.OverflowY == webrender.OverflowScroll ||
			style.OverflowX == webrender.OverflowHidden || style.OverflowX == webrender.OverflowAuto || style.OverflowX == webrender.OverflowScroll
		if needsClip {
			clipRect = webrender.LayoutRect{
				X: box.BorderLeft(), Y: box.BorderTop(),
				W: box.ContentWidth() + box.PaddingLeft() + box.PaddingRight(),
				H: box.ContentHeight() + box.PaddingTop() + box.PaddingBottom(),
			}
		} else {
			clipRect = webrender.LayoutRect{0, 0, box.BorderBoxWidth(), box.BorderBoxHeight()}
		}
	} else {
		clipRect = webrender.LayoutRect{0, 0, box.BorderBoxWidth(), box.BorderBoxHeight()}
	}

	if clipRect.W > 0 && clipRect.H > 0 {
		rc.ClipRect(clipRect)
	}

	// 应用 CSS opacity
	childAlpha := info.Alpha
	if style != nil && style.Opacity > 0 && style.Opacity < 1.0 {
		if childAlpha > 0 {
			childAlpha *= style.Opacity
		} else {
			childAlpha = style.Opacity
		}
	}

	// 设置透明度
	if childAlpha < 1.0 {
		rc.SetOpacity(childAlpha)
	}

	// 绘制元素自身（背景、边框）
	selfInfo := info
	selfInfo.Alpha = childAlpha

	selfInfo.Phase = webrender.PaintPhaseBlockBackground
	box.Paint(selfInfo)

	// 切换至前景阶段，使 RenderText.Paint() 可以绘制文本
	selfInfo.Phase = webrender.PaintPhaseForeground
	box.Paint(selfInfo)

	// 递归绘制子元素（继承 Foreground Phase，使文本节点可绘制）
	if el, ok := obj.(webrender.RenderElement); ok {
		el.ForEachChild(func(child webrender.RenderObject) {
			paintRecursive(child, webrender.LayoutPoint{}, selfInfo, rc)
		})
	}

	rc.Restore()
}
