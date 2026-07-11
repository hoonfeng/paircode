//go:build windows

// menurender — 标题栏菜单渲染坐标验证工具
// 使用真实管线渲染含菜单的标题栏，输出所有组件和文本的精确坐标。
//
// 运行：
//   cd F:\syproject\GWui
//   set CGO_ENABLED=1
//   go run F:\syproject\gou-ide\cmd\menurender\main.go

package main

import (
	"fmt"
	"strings"

	"github.com/hoonfeng/gwui/css"
	"github.com/hoonfeng/gwui/html5"
	"github.com/hoonfeng/gwui/webrender"
)

func main() {
	html := buildHTML()
	w, h := 1400, 840

	// 解析 HTML
	doc, err := html5.Parse(html)
	if err != nil {
		panic(err)
	}

	// 提取 CSS
	cssText := doc.GetStyleSheetText()

	// 解析 CSS
	var sheet *css.StyleSheet
	if cssText != "" {
		sheet = css.ParseStyleSheet(cssText)
	}

	// 创建 RecordingCanvas
	rc := webrender.NewRecordingCanvas(
		webrender.LayoutUnit(w),
		webrender.LayoutUnit(h),
	)

	// 创建 Pipeline
	pipeline := webrender.NewRenderPipeline(
		doc, rc,
		webrender.LayoutUnit(w),
		webrender.LayoutUnit(h),
	)

	// 设置 CSS
	if sheet != nil {
		pipeline.SetStyleSheet(sheet)
	}

	// 渲染
	pipeline.Render()

	// 获取绘制日志
	paintLog := ""
	if r, ok := pipeline.GetCanvas().(*webrender.RecordingCanvas); ok {
		paintLog = r.PaintLog()
	}

	lines := strings.Split(strings.TrimRight(paintLog, "\n"), "\n")

	// ===== 输出标题栏组件坐标 =====
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  标题栏菜单渲染 · 坐标验证报告")
	fmt.Println("  视口: 1400×840  字体: 13px  引擎: WebRender")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()

	// 解析所有 DrawText 操作
	type TextInfo struct {
		Text     string
		AbsX, AbsY float64
		FontSize  int
		Color     string
	}

	var texts []TextInfo
	var fillRects []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "DrawTextWithWeight") {
			var x, y float64
			var sz int
			var col, text string
			_, err := fmt.Sscanf(trimmed,
				"DrawTextWithWeight  abs=(%f,%f), alpha=1.00, color=%s, fontSize=%d",
				&x, &y, &col, &sz)
			if err == nil {
				if idx := strings.Index(trimmed, `text="`); idx >= 0 {
					end := strings.LastIndex(trimmed, `"`)
					if end > idx+6 {
						text = trimmed[idx+6 : end]
					}
				}
				texts = append(texts, TextInfo{
					Text: text, AbsX: x, AbsY: y,
					FontSize: sz, Color: col,
				})
			}
		}
		if strings.Contains(trimmed, "FillRect") {
			fillRects = append(fillRects, trimmed)
		}
	}

	// ── 1. 标题栏整体 ──
	fmt.Println("◆ 标题栏 (TitleBar)")
	fmt.Println("  CSS: height=36px, background=#3c3c3d, flex row")
	fmt.Println("  预期: 全宽 1400×36, 顶部 (0,0)")
	for _, fr := range fillRects {
		if strings.Contains(fr, "#3c3c3d") {
			fmt.Printf("  ✓ 实际: %s\n", fr)
		}
	}

	fmt.Println()
	fmt.Println("◆ Logo 图标")
	fmt.Println("  CSS: width=36px, height=36px, color=#0e639c")
	fmt.Println("  预期: x=0, y=0~36, 文字居中")
	for _, t := range texts {
		if t.Text == "✦" {
			fmt.Printf("  ✓ 实际: text=%q abs=(%.0f, %.0f) fontSize=%d color=%s\n",
				t.Text, t.AbsX, t.AbsY, t.FontSize, t.Color)
		}
	}

	fmt.Println()
	fmt.Println("◆ 菜单按钮 × 6")
	fmt.Println("  CSS: flex:0 0 auto, padding:0 10px, font-size:13px, color=#cccccc")
	fmt.Println("  布局: 在 menu-bar 内从左排列 (flex-start)")
	fmt.Println("  menu-bar: x=36, w=613 (flex:1)")
	fmt.Println()
	fmt.Println("  ┌──────────┬────────┬──────┬─────────────────────────┐")
	fmt.Println("  │ 按钮文字  │ X坐标  │ 宽度  │ 文字在按钮内居中位置    │")
	fmt.Println("  ├──────────┼────────┼──────┼─────────────────────────┤")

	buttonOrder := []string{"文件", "编辑", "视图", "终端", "Agent", "帮助"}
	buttonData := make(map[string]TextInfo)
	for _, t := range texts {
		for _, name := range buttonOrder {
			if t.Text == name {
				buttonData[name] = t
			}
		}
	}

	var prevX float64
	for i, name := range buttonOrder {
		if t, ok := buttonData[name]; ok {
			// 估算按钮宽度（文本宽 + padding 20px）
			// 13px 字体下，中文每字约 13px
			textWidth := float64(len([]rune(name))) * 7.8
			if len([]rune(name)) > 1 {
				textWidth = float64(len([]rune(name))) * 13.0
			}
			btnW := textWidth + 20.0
			// 文字在按钮内居中，所以按钮 x = textX - (btnW-textWidth)/2
			btnX := t.AbsX - (btnW-textWidth)/2
			if i > 0 {
				gap := btnX - prevX
				fmt.Printf("  │ %-8s │ %6.0f │ %4.0f │ gap=%.0f (与前一个)         │\n",
					name, btnX, btnW, gap)
			} else {
				fmt.Printf("  │ %-8s │ %6.0f │ %4.0f │ 首按钮                   │\n",
					name, btnX, btnW)
			}
			prevX = btnX + btnW
		} else {
			fmt.Printf("  │ %-8s │  未找到 │      │                         │\n", name)
		}
	}
	fmt.Println("  └──────────┴────────┴──────┴─────────────────────────┘")

	fmt.Println()
	fmt.Println("◆ 标题文字 (TitleCenter)")
	fmt.Println("  CSS: flex:1, text-align:center, font-size:13px, color=#8c8c8c")
	fmt.Println("  预期在 x=649 开始的 613px 区域内居中")
	for _, t := range texts {
		if t.Text == "Pair CodeAgent" {
			fmt.Printf("  ✓ 实际: text=%q abs=(%.0f, %.0f) fontSize=%d color=%s\n",
				t.Text, t.AbsX, t.AbsY, t.FontSize, t.Color)
		}
	}

	fmt.Println()
	fmt.Println("◆ 窗口控制按钮 × 3")
	fmt.Println("  CSS: width=46px, height=36px, color=#cccccc")
	fmt.Println("  预期区域: x=1262~1400")
	for _, t := range texts {
		if t.Text == "─" || t.Text == "☐" || t.Text == "✕" {
			fmt.Printf("  ✓ 实际: text=%q abs=(%.0f, %.0f) fontSize=%d color=%s\n",
				t.Text, t.AbsX, t.AbsY, t.FontSize, t.Color)
		}
	}

	fmt.Println()
	fmt.Println("◆ 状态栏")
	fmt.Println("  CSS: height=26px, background=#2d2d30, color=#8c8c8c, font-size=12px")
	fmt.Println("  预期区域: 标题栏下 814px 处开始")
	for _, t := range texts {
		if t.Text == "就绪" || t.Text == "v0.1.0" {
			fmt.Printf("  ✓ 实际: text=%q abs=(%.0f, %.0f) fontSize=%d color=%s\n",
				t.Text, t.AbsX, t.AbsY, t.FontSize, t.Color)
		}
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  颜色验证")
	fmt.Println("═══════════════════════════════════════════════════")
	colorCheck := map[string]string{
		"#3c3c3d": "标题栏背景",
		"#0e639c": "Logo 颜色",
		"#8c8c8c": "标题/状态栏文字",
		"#cccccc": "菜单按钮/窗口按钮文字",
		"#2c2c2c": "活动栏背景",
		"#2d2d30": "状态栏背景",
		"#1e1e1e": "编辑器背景",
	}
	for color, desc := range colorCheck {
		found := false
		for _, line := range lines {
			if strings.Contains(line, color) {
				found = true
				break
			}
		}
		status := "✓"
		if !found {
			status = "✗ 缺失"
		}
		fmt.Printf("  %s %s → %s\n", status, color, desc)
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  弹出菜单坐标")
	fmt.Println("═══════════════════════════════════════════════════")

	// 弹出菜单的 HTML 单独渲染
	popupHTML := buildPopupHTML()
	popupDoc, _ := html5.Parse(popupHTML)
	popupCSS := popupDoc.GetStyleSheetText()
	popupSheet := css.ParseStyleSheet(popupCSS)

	popupRC := webrender.NewRecordingCanvas(1400, 840)
	popupPipeline := webrender.NewRenderPipeline(popupDoc, popupRC, 1400, 840)
	if popupSheet != nil {
		popupPipeline.SetStyleSheet(popupSheet)
	}
	popupPipeline.Render()

	popupLog := ""
	if r, ok := popupPipeline.GetCanvas().(*webrender.RecordingCanvas); ok {
		popupLog = r.PaintLog()
	}

	popupLines := strings.Split(strings.TrimRight(popupLog, "\n"), "\n")

	var popupTexts []TextInfo
	for _, line := range popupLines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "DrawTextWithWeight") {
			var x, y float64
			var sz int
			var col, text string
			_, err := fmt.Sscanf(trimmed,
				"DrawTextWithWeight  abs=(%f,%f), alpha=1.00, color=%s, fontSize=%d",
				&x, &y, &col, &sz)
			if err == nil {
				if idx := strings.Index(trimmed, `text="`); idx >= 0 {
					end := strings.LastIndex(trimmed, `"`)
					if end > idx+6 {
						text = trimmed[idx+6 : end]
					}
				}
				popupTexts = append(popupTexts, TextInfo{
					Text: text, AbsX: x, AbsY: y,
					FontSize: sz, Color: col,
				})
			}
		}
	}

	fmt.Println("  弹出菜单位置: position:absolute; top:36px; left:36px;")
	fmt.Println("  CSS: min-width:200px, background=#252526, border=#454545")
	fmt.Println("  菜单项: padding:4px 28px 4px 24px, height=26px")
	fmt.Println()
	fmt.Println("  ┌──────────────────────────────┬────────┬────────┐")
	fmt.Println("  │ 菜单项                       │ X坐标  │ Y坐标  │")
	fmt.Println("  ├──────────────────────────────┼────────┼────────┤")
	for _, t := range popupTexts {
		if t.FontSize <= 14 {
			fmt.Printf("  │ %-28s │ %6.0f │ %6.0f │\n",
				t.Text, t.AbsX, t.AbsY)
		}
	}
	fmt.Println("  └──────────────────────────────┴────────┴────────┘")

	// 验证弹出菜单背景色
	for _, line := range popupLines {
		if strings.Contains(line, "#252526") && strings.Contains(line, "FillRect") {
			fmt.Printf("\n  ✓ 弹出菜单背景: %s\n", strings.TrimSpace(line))
			break
		}
	}
	for _, line := range popupLines {
		if strings.Contains(line, "#454545") && strings.Contains(line, "FillRect") {
			fmt.Printf("  ✓ 分隔线: %s\n", strings.TrimSpace(line))
			break
		}
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  验证结论")
	fmt.Println("═══════════════════════════════════════════════════")

	// 统计
	fmt.Printf("  总绘制操作: %d\n", len(lines))
	saves, restores := 0, 0
	for _, l := range lines {
		if strings.Contains(l, "Save") {
			saves++
		}
		if strings.Contains(l, "Restore") {
			restores++
		}
	}
	fmt.Printf("  Save=%d Restore=%d %s\n", saves, restores,
		map[bool]string{true: "✓ 平衡", false: "✗ 不平衡"}[saves == restores])

	fmt.Printf("  文本绘制: %d 项\n", len(texts))
	fmt.Printf("  填充矩形: %d 个\n", len(fillRects))

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  完整绘制日志")
	fmt.Println("═══════════════════════════════════════════════════")
	for _, line := range lines {
		fmt.Println(line)
	}
}

func buildHTML() string {
	return `<!DOCTYPE html>
<html><head><style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-size: 13px; color: #cccccc; background: #1e1e1e; overflow: hidden; display:flex; flex-direction:column; width:100%; height:100%; }
.titlebar { display: flex; flex-direction: row; height: 36px; background: #3c3c3d; align-items: center; user-select: none; overflow: hidden; }
.titlebar .logo { width: 36px; height: 36px; display: flex; align-items: center; justify-content: center; font-size: 18px; color: #0e639c; }
.titlebar .menu-bar { display: flex; flex-direction: row; flex: 1; height: 36px; align-items: center; }
.titlebar .title-center { flex: 1; text-align: center; font-size: 13px; color: #8c8c8c; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.titlebar .win-btns { display: flex; flex-direction: row; }
.titlebar .win-btn { width: 46px; height: 36px; display: flex; align-items: center; justify-content: center; cursor: pointer; color: #cccccc; }
.body { display: flex; flex-direction: row; flex: 1; overflow: hidden; }
.activity-bar { width: 48px; flex-shrink: 0; display: flex; flex-direction: column; align-items: center; background: #2c2c2c; padding: 4px 0; }
.activity-item { width: 48px; height: 48px; display: flex; align-items: center; justify-content: center; color: #868686; }
.activity-item.active { color: #ffffff; }
.activity-spacer { flex: 1; }
.center-panel { display: flex; flex-direction: column; flex: 1; background: #1e1e1e; overflow: hidden; }
.statusbar { display: flex; flex-direction: row; height: 26px; background: #2d2d30; align-items: center; justify-content: space-between; padding: 0 8px; font-size: 12px; color: #8c8c8c; }
.popup-menu { display: flex; flex-direction: column; min-width: 200px; background: #252526; border: 1px solid #454545; padding: 4px 0; font-size: 13px; color: #cccccc; user-select: none; }
.popup-menu-item { padding: 4px 28px 4px 24px; height: 26px; display: flex; flex-direction: row; align-items: center; cursor: pointer; white-space: nowrap; color: #cccccc; }
.popup-menu-divider { height: 1px; background: #454545; margin: 4px 0; }
</style></head>
<body>
  <div class="titlebar">
    <div class="logo"><span>✦</span></div>
    <div class="menu-bar" id="menu-bar">
      <button style="flex:0 0 auto;display:inline-flex;align-items:center;justify-content:center;height:36px;padding:0 10px;font-size:13px;color:#cccccc;background:transparent;border:none;">文件</button>
      <button style="flex:0 0 auto;display:inline-flex;align-items:center;justify-content:center;height:36px;padding:0 10px;font-size:13px;color:#cccccc;background:transparent;border:none;">编辑</button>
      <button style="flex:0 0 auto;display:inline-flex;align-items:center;justify-content:center;height:36px;padding:0 10px;font-size:13px;color:#cccccc;background:transparent;border:none;">视图</button>
      <button style="flex:0 0 auto;display:inline-flex;align-items:center;justify-content:center;height:36px;padding:0 10px;font-size:13px;color:#cccccc;background:transparent;border:none;">终端</button>
      <button style="flex:0 0 auto;display:inline-flex;align-items:center;justify-content:center;height:36px;padding:0 10px;font-size:13px;color:#cccccc;background:transparent;border:none;">Agent</button>
      <button style="flex:0 0 auto;display:inline-flex;align-items:center;justify-content:center;height:36px;padding:0 10px;font-size:13px;color:#cccccc;background:transparent;border:none;">帮助</button>
    </div>
    <div class="title-center">Pair CodeAgent</div>
    <div class="win-btns">
      <div class="win-btn"><span>─</span></div>
      <div class="win-btn"><span>☐</span></div>
      <div class="win-btn close"><span>✕</span></div>
    </div>
  </div>
  <div class="popup-menu" style="position:absolute;top:36px;left:36px;">
    <div class="popup-menu-item">新建文件   (Ctrl+N)</div>
    <div class="popup-menu-item">打开文件…   (Ctrl+O)</div>
    <div class="popup-menu-divider"></div>
    <div class="popup-menu-item">保存   (Ctrl+S)</div>
    <div class="popup-menu-divider"></div>
    <div class="popup-menu-item">关闭项目</div>
  </div>
  <div class="body">
    <div class="activity-bar">
      <div class="activity-item active">📁</div>
      <div class="activity-spacer"></div>
    </div>
    <div class="center-panel"></div>
  </div>
  <div class="statusbar">
    <div class="status-left"><span>就绪</span></div>
    <div class="status-right"><span>v0.1.0</span></div>
  </div>
</body>
</html>`
}

func buildPopupHTML() string {
	return `<!DOCTYPE html>
<html><head><style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-size: 13px; color: #cccccc; background: #1e1e1e; overflow: hidden; display:flex; flex-direction:column; width:100%; height:100%; }
.titlebar { display: flex; flex-direction: row; height: 36px; background: #3c3c3d; align-items: center; }
.titlebar .logo { width: 36px; height: 36px; display: flex; align-items: center; justify-content: center; }
.titlebar .menu-bar { display: flex; flex-direction: row; flex: 1; }
.titlebar .title-center { flex: 1; text-align: center; font-size: 13px; color: #8c8c8c; }
.titlebar .win-btns { display: flex; flex-direction: row; }
.titlebar .win-btn { width: 46px; height: 36px; display: flex; align-items: center; justify-content: center; color: #cccccc; }
.popup-menu { display: flex; flex-direction: column; min-width: 200px; background: #252526; border: 1px solid #454545; padding: 4px 0; font-size: 13px; color: #cccccc; }
.popup-menu-item { padding: 4px 28px 4px 24px; height: 26px; display: flex; flex-direction: row; align-items: center; white-space: nowrap; color: #cccccc; }
.popup-menu-divider { height: 1px; background: #454545; margin: 4px 0; }
</style></head>
<body>
  <div class="popup-menu" style="position:absolute;top:36px;left:36px;">
    <div class="popup-menu-item">新建文件   (Ctrl+N)</div>
    <div class="popup-menu-item">打开文件…   (Ctrl+O)</div>
    <div class="popup-menu-divider"></div>
    <div class="popup-menu-item">保存   (Ctrl+S)</div>
    <div class="popup-menu-divider"></div>
    <div class="popup-menu-item">关闭项目</div>
  </div>
</body>
</html>`
}
