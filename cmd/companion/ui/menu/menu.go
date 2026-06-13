// 菜单动作落地：视图缩放/专注/Minimap 的助手 + Agent 监控 / 性能监控 / 扩展市场 / 更新日志 对话框。

//go:build windows


package menuactions

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/ui/chat"
	"github.com/hoonfeng/paircode/cmd/companion/ui/marketplace"
		"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// relayout 触发整树重排（字号/minimap 等改尺寸的设置即时生效）。
func Relayout() {
	if widget.OnNeedsLayout != nil {
		widget.OnNeedsLayout()
	}
}

func EditorFontSize() int {
	if core.Settings.EditorFontSize > 0 {
		return core.Settings.EditorFontSize
	}
	return 14 // 默认
}

func SetEditorFontSize(sz int) {
	if sz < 8 {
		sz = 8
	} else if sz > 40 {
		sz = 40
	}
	core.Settings.EditorFontSize = sz
	core.Save()
	Relayout()
}

func onOff(b bool) string {
	if b {
		return "开"
	}
	return "关"
}

// showContentDialog 通用信息弹窗：标题 + 内容 + 关闭按钮。替代用 ShowAlert 塞大段文本（排版会乱）。
func ShowContentDialog(title string, width float64, content widget.Widget) {
	var id int
	dlg := widget.NewDialog(title, content).WithWidth(width).WithTransition("fade").WithFooter(
		&widget.Button{Text: "关闭", OnClick: func() { widget.HideOverlay(id) },
			Color: *ui.BgMuted, TextColor: *ui.Fg, FontSize: 12,
			MinWidth: 50, MinHeight: 24, Padding: types.EdgeInsetsLTRB(12, 2, 13, 3)},
	)
	id = widget.ShowDialog(dlg)
}

// kvRow 键值行：左键名(muted)定宽 + 右值。
func kvRow(k, v string) widget.Widget {
	return widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
		widget.Div(widget.Style{Width: 130}, ui.TextC(k, *ui.FgMuted, 12)),
		ui.Expand(ui.TextC(v, *ui.Fg, 12)),
	)
}

// dialogColumn 弹窗内容列（定宽、纵向 stretch、内边距）。
func dialogColumn(width float64, kids ...widget.Widget) widget.Widget {
	return widget.Div(widget.Style{Width: width, FlexDirection: "column", AlignItems: "stretch",
		Padding: types.EdgeInsetsLTRB(6, 6, 6, 6)}, kids)
}

// showAgentMonitor Agent 监控：模型 / 运行态 / 编排开关 / 会话规模。
func ShowAgentMonitor() {
	run := "空闲"
	if chatpanel.TheState.Bridge != nil && chatpanel.TheState.Bridge.IsRunning() {
		run = "运行中"
	}
	autonomous := chatpanel.TheState.Autonomous
	msgs := 0
	if chatpanel.TheState != nil {
		if th := chatpanel.TheState.Store.Active(); th != nil {
			msgs = len(th.Messages)
		}
	}
	ShowContentDialog("Agent 监控", 440, dialogColumn(404,
		kvRow("主模型", core.MainModel()),
		kvRow("运行状态", run),
		kvRow("自主模式", onOff(autonomous)),
		kvRow("AI 审核", onOff(core.Settings.AIReview)),
		kvRow("任务评测", onOff(core.Settings.Benchmark)),
		kvRow("Lua 工具", onOff(core.Settings.LuaTools)),
		kvRow("当前对话消息数", fmt.Sprintf("%d", msgs)),
	))
}

// showPerfMonitor 性能监控：运行时内存 / GC / 协程。
func ShowPerfMonitor() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	ShowContentDialog("性能监控", 440, dialogColumn(404,
		kvRow("堆内存占用", fmt.Sprintf("%.1f MB", float64(m.HeapAlloc)/1e6)),
		kvRow("系统保留", fmt.Sprintf("%.1f MB", float64(m.Sys)/1e6)),
		kvRow("GC 次数", fmt.Sprintf("%d", m.NumGC)),
		kvRow("活跃 Goroutine", fmt.Sprintf("%d", runtime.NumGoroutine())),
		kvRow("CPU 核心", fmt.Sprintf("%d", runtime.NumCPU())),
		kvRow("Go 版本", runtime.Version()),
	))
}

// ShowPerfDemo 性能测试：模拟IDE环境加载大量数据，排查渲染性能瓶颈。
// 通过生成大量模拟数据（文件节点、代码行、列表项），观察帧率变化。
func ShowPerfDemo() {
	ShowContentDialog("性能测试", 520, dialogColumn(480,
		kvRow("诊断引擎", "Pipeline 帧率诊断（5秒周期）"),
		widget.Div(widget.Style{Height: 8}),
		ui.TextC("点击下方按钮执行压力测试，观察 stderr 日志中的 [perf] 输出。", *ui.FgMuted, 11),
		widget.Div(widget.Style{Height: 12}),

		// 测试1：大量文本行
		&widget.Button{Text: "① 10000行文本压力测试", OnClick: func() { PerfTestMassiveText() },
			Color: *ui.Bg, TextColor: *ui.Fg, FontSize: 12, MinWidth: 460, MinHeight: 28},
		widget.Div(widget.Style{Height: 6}),

		// 测试2：大量按钮/控件
		&widget.Button{Text: "② 1000个控件渲染测试", OnClick: func() { PerfTestManyWidgets() },
			Color: *ui.Bg, TextColor: *ui.Fg, FontSize: 12, MinWidth: 460, MinHeight: 28},
		widget.Div(widget.Style{Height: 6}),

		// 测试3：大文件编辑器
		&widget.Button{Text: "③ 打开5000行大文件", OnClick: func() { PerfTestLargeFile() },
			Color: *ui.Bg, TextColor: *ui.Fg, FontSize: 12, MinWidth: 460, MinHeight: 28},
		widget.Div(widget.Style{Height: 6}),

		// 测试4：复杂嵌套布局
		&widget.Button{Text: "④ 复杂嵌套布局测试", OnClick: func() { PerfTestNestedLayout() },
			Color: *ui.Bg, TextColor: *ui.Fg, FontSize: 12, MinWidth: 460, MinHeight: 28},
		widget.Div(widget.Style{Height: 6}),

		// 测试5：全压力测试（组合所有场景）
		&widget.Button{Text: "⑤ 全压力测试（组合所有场景）", OnClick: func() { PerfTestFullStress() },
			Color: *ui.Accent, TextColor: *ui.White, FontSize: 12, MinWidth: 460, MinHeight: 28},
		widget.Div(widget.Style{Height: 8}),

		ui.TextC("测试结果将输出到 stderr，格式：[perf] fps | frames | build/layout/paint/flush", *ui.FgMuted, 10),
	))
}

// ── 性能测试场景实现 ──

// PerfTestMassiveText 大量文本行测试：生成包含10000行文本的页面。
func PerfTestMassiveText() {
	var id int
	lines := make([]widget.Widget, 10000)
	for i := 0; i < 10000; i++ {
		lines[i] = ui.TextC(fmt.Sprintf("第 %d 行 - 性能测试文本行，用于测试渲染引擎对大量文本的绘制性能。", i+1), *ui.Fg, 12)
	}
	content := widget.NewScrollView(ui.VBox(lines...))
	dlg := widget.NewDialog("10000行文本压力测试", content).WithWidth(700).WithHeight(600).WithTransition("fade").WithFooter(
		&widget.Button{Text: "关闭", OnClick: func() { widget.HideOverlay(id) },
			Color: *ui.BgMuted, TextColor: *ui.Fg, FontSize: 12, MinWidth: 50, MinHeight: 24},
	)
	id = widget.ShowDialog(dlg)
}

// PerfTestManyWidgets 大量控件测试：生成包含1000个按钮/控件的页面。
func PerfTestManyWidgets() {
	var id int
	kids := make([]widget.Widget, 0, 1000)
	for i := 0; i < 200; i++ {
		rowKids := make([]widget.Widget, 0, 5)
		for j := 0; j < 5; j++ {
			idx := i*5 + j + 1
			rowKids = append(rowKids, &widget.Button{
				Text: fmt.Sprintf("Btn%d", idx), FontSize: 10,
				Color: *ui.Bg, TextColor: *ui.Fg, MinWidth: 60, MinHeight: 22,
				Padding: types.EdgeInsetsLTRB(4, 1, 4, 1),
			})
		}
		kids = append(kids, widget.Div(widget.Style{FlexDirection: "row", Gap: 4, Padding: types.EdgeInsetsLTRB(0, 2, 0, 2)}, rowKids...))
	}
	content := widget.NewScrollView(ui.VBox(kids...))
	dlg := widget.NewDialog("1000个控件渲染测试", content).WithWidth(700).WithHeight(600).WithTransition("fade").WithFooter(
		&widget.Button{Text: "关闭", OnClick: func() { widget.HideOverlay(id) },
			Color: *ui.BgMuted, TextColor: *ui.Fg, FontSize: 12, MinWidth: 50, MinHeight: 24},
	)
	id = widget.ShowDialog(dlg)
}

// PerfTestLargeFile 大文件编辑器测试：在当前工作区生成一个5000行的大文件并打开。
func PerfTestLargeFile() {
	// 获取当前工作区目录
	if len(core.Folders) == 0 {
		widget.MessageWarning("请先打开一个工作区")
		return
	}
	root := core.Folders[0]
	fpath := filepath.Join(root, ".perf_test_large.go")

	// 生成5000行Go代码
	var buf strings.Builder
	buf.WriteString("package perf_test\n\n")
	buf.WriteString("// 性能测试大文件 - 自动生成\n")
	buf.WriteString("var (\n")
	for i := 0; i < 5000; i++ {
		buf.WriteString(fmt.Sprintf("\tperfVar%d = %d // 第%d行性能测试变量\n", i, i*100, i+1))
	}
	buf.WriteString(")\n")

	if err := os.WriteFile(fpath, []byte(buf.String()), 0644); err != nil {
		widget.MessageWarning("写入测试文件失败: " + err.Error())
		return
	}
	editorpanel.Editor.Open(fpath)
	widget.MessageInfo("已打开5000行大文件，请观察帧率变化")
}

// PerfTestNestedLayout 复杂嵌套布局测试。
func PerfTestNestedLayout() {
	var id int
	buildNested := func(depth, width int) widget.Widget {
		if depth <= 0 || width <= 0 {
			return widget.Div(widget.Style{Width: 10, Height: 10, BackgroundColor: *ui.Border})
		}
		kids := make([]widget.Widget, 0, width)
		for i := 0; i < width; i++ {
			kids = append(kids, buildNested(depth-1, width-1))
		}
		return widget.Div(
			widget.Style{FlexDirection: "row", Gap: 2, Padding: types.EdgeInsets(2),
				BackgroundColor: *ui.BgMuted, BorderColor: *ui.Border, BorderWidth: 1},
			kids...,
		)
	}
	content := widget.NewScrollView(
		ui.VBox(
			ui.TextC("复杂嵌套布局（深度4，宽度4）：共 1+4+12+24 = 41 个容器", *ui.FgMuted, 11),
			widget.Div(widget.Style{Height: 8}),
			buildNested(4, 4),
		),
	)
	dlg := widget.NewDialog("复杂嵌套布局测试", content).WithWidth(600).WithHeight(500).WithTransition("fade").WithFooter(
		&widget.Button{Text: "关闭", OnClick: func() { widget.HideOverlay(id) },
			Color: *ui.BgMuted, TextColor: *ui.Fg, FontSize: 12, MinWidth: 50, MinHeight: 24},
	)
	id = widget.ShowDialog(dlg)
}

// PerfTestFullStress 全压力测试：组合所有场景，最大化渲染压力。
func PerfTestFullStress() {
	// 先打开大文件
	if len(core.Folders) > 0 {
		root := core.Folders[0]
		fpath := filepath.Join(root, ".perf_test_stress.go")
		var buf strings.Builder
		buf.WriteString("package perf_test\n\n")
		buf.WriteString("// 全压力测试文件\n")
		buf.WriteString("func PerfStressTest() {\n")
		for i := 0; i < 3000; i++ {
			buf.WriteString(fmt.Sprintf("\t_ = %d + %d // line %d\n", i, i*2, i+1))
		}
		buf.WriteString("}\n")
		os.WriteFile(fpath, []byte(buf.String()), 0644)
		editorpanel.Editor.Open(fpath)
	}

	// 再弹大量文本对话框
	var id int
	lines := make([]widget.Widget, 5000)
	for i := 0; i < 5000; i++ {
		lines[i] = ui.TextC(fmt.Sprintf("Stress %d: 全压力测试文本行 - 渲染引擎性能极限测试", i+1), *ui.Fg, 12)
	}
	content := widget.NewScrollView(ui.VBox(lines...))
	dlg := widget.NewDialog("全压力测试", content).WithWidth(800).WithHeight(700).WithTransition("fade").WithFooter(
		&widget.Button{Text: "关闭", OnClick: func() { widget.HideOverlay(id) },
			Color: *ui.BgMuted, TextColor: *ui.Fg, FontSize: 12, MinWidth: 50, MinHeight: 24},
	)
	id = widget.ShowDialog(dlg)
	widget.MessageInfo("全压力测试已启动，请观察stderr日志中的[perf]帧率数据")
}

// showMarketplace 扩展市场：模态对话框，三 Tab(MCP/技能/已安装) + 搜索 + 作用域 + 卡片（见 marketplace.go）。
func ShowMarketplace() {
	marketplacepanel.OpenDialog()
}

// editorReferences 查找引用结果 → 弹「引用」对话框列出各处，点击跳转。
func EditorReferences(refs []widget.CodeLoc) {
	if len(refs) == 0 {
		widget.MessageWarning("未找到引用")
		return
	}
	var id int
	rows := make([]widget.Widget, 0, len(refs))
	for _, r := range refs {
		ref := r
		rows = append(rows, jumpRow(relPathLabel(ref.File)+":"+strconv.Itoa(ref.Line), func() {
			editorpanel.Editor.OpenAt(ref.File, ref.Line)
			widget.HideOverlay(id)
		}))
	}
	id = showJumpDialog(fmt.Sprintf("引用（%d 处）", len(refs)), rows)
}

// editorSymbols 文档符号大纲 → 弹「符号」对话框（按层级缩进），点击跳到当前文件该行。
func EditorSymbols(syms []widget.CodeSym) {
	if len(syms) == 0 {
		widget.MessageWarning("未找到符号")
		return
	}
	file := ""
	if t := editorpanel.Editor.ActiveTab(); t != nil {
		file = t.Path()
	}
	var id int
	rows := make([]widget.Widget, 0, len(syms))
	for _, s := range syms {
		sym := s
		txt := strings.Repeat("    ", sym.Depth) + symKindLabel(sym.Kind) + " " + sym.Name
		rows = append(rows, jumpRow(txt, func() {
			if file != "" {
				editorpanel.Editor.OpenAt(file, sym.Line)
			}
			widget.HideOverlay(id)
		}))
	}
	id = showJumpDialog(fmt.Sprintf("符号大纲（%d）", len(syms)), rows)
}

// jumpRow 一个可点击的跳转行（左对齐，悬停高亮）。
func jumpRow(text string, onClick func()) widget.Widget {
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(10, 5, 10, 5)},
			ui.TextC(text, *ui.Fg, 12),
		)},
		OnClick:    onClick,
		HoverColor: *ui.BgMuted,
	}
}

// showJumpDialog 通用「可点击列表」对话框（引用/符号共用），返回 overlay id。
func showJumpDialog(title string, rows []widget.Widget) int {
	body := widget.Div(widget.Style{Width: 520, Height: 440, FlexDirection: "column", AlignItems: "stretch"},
		ui.Expand(widget.NewScrollView(widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", Gap: 1}, rows))),
	)
	var id int
	dlg := widget.NewDialog(title, body).WithWidth(556).WithTransition("fade").WithFooter(
		&widget.Button{Text: "关闭", OnClick: func() { widget.HideOverlay(id) },
			Color: *ui.BgMuted, TextColor: *ui.Fg, FontSize: 12,
			MinWidth: 50, MinHeight: 24, Padding: types.EdgeInsetsLTRB(12, 2, 13, 3)},
	)
	id = widget.ShowDialog(dlg)
	return id
}

// relPathLabel 文件相对工作区根的路径（不在根下则用文件名）。
func relPathLabel(file string) string {
	if rel, err := filepath.Rel(core.Root(), file); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.Base(file)
}

// symKindLabel LSP SymbolKind → 简短文字标签（避免缺字形图标）。
func symKindLabel(k int) string {
	switch k {
	case 12, 6, 9: // Function / Method / Constructor
		return "func"
	case 23, 5: // Struct / Class
		return "type"
	case 11: // Interface
		return "iface"
	case 14: // Constant
		return "const"
	case 13, 8, 7: // Variable / Field / Property
		return "var"
	case 4, 3: // Package / Namespace
		return "pkg"
	}
	return "·"
}

const ChangelogText = `伴随式 CodeAgent — 更新日志

近期更新
• 二进制逆向工具集：strings / find / patch / info(PE·ELF·Mach-O) / hash / entropy（+ inspect / write）
• 忽略目录可配置：全局（设置面板）+ 项目级 .pair/ignore，防上下文爆炸
• 项目知识库：Agent 菜单「探索项目知识库」→ 构建可浏览的中文知识库
• 记忆系统：MEMORY.md 中文总览索引 + 可更新/删除（不再堆重复记忆）
• 防绕圈：检测重复失败/原地打转 → 自动提示换思路，打破死循环
• 多角色编排：规划/探索/执行/验证/审核/调试，自主模式全链路
• 上下文压缩、任务评测评分、Lua 自定义工具、MCP/Skills 市场
• 自管理工具：Agent 自己检索/安装/修改/删除 Skills 与 MCP

视图与体验
• 专注模式、编辑器缩放（放大/缩小）、Minimap 开关
• 多主题（深色/浅色/高对比/Solarized/Dracula）`
