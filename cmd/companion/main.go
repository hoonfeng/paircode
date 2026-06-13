// 伴随式 CodeAgent —— goui 重构。入口 + 主窗壳（标题栏 / 停靠区 / 状态栏）。
// 详见同目录 AGENTS.md（开发铁律）。
//
//go:build windows

package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/hoonfeng/paircode/cmd/companion/ui/state"
	"github.com/hoonfeng/goui/pkg/app"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"

	_ "github.com/hoonfeng/goui/pkg/platform"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	core "github.com/hoonfeng/paircode/cmd/companion/core"
	chatpanel "github.com/hoonfeng/paircode/cmd/companion/ui/chat"
	editorpanel "github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	filetreepanel "github.com/hoonfeng/paircode/cmd/companion/ui/filetree"
	gitpanel "github.com/hoonfeng/paircode/cmd/companion/ui/git"
	searchpanel "github.com/hoonfeng/paircode/cmd/companion/ui/search"
	termpanel "github.com/hoonfeng/paircode/cmd/companion/ui/terminal"
	settingspanel "github.com/hoonfeng/paircode/cmd/companion/ui/settings"
	ctxmenupanel "github.com/hoonfeng/paircode/cmd/companion/ui/ctxmenu"
	logo "github.com/hoonfeng/paircode/cmd/companion/ui/logo"
	menuactions "github.com/hoonfeng/paircode/cmd/companion/ui/menu"

)

var application *app.Application
var shellStateRef *shellState // 全局 shell 状态引用，供 workspace 关闭后 SetState 重建

// VS Code Dark+ 风格配色（IDE 深色）。
var (
	cTitle   = types.ColorRef(60, 60, 61)  // 标题栏 #3c3c3d
	cSide    = types.ColorRef(37, 37, 38)  // 侧栏/面板 #252526
	cEditor  = types.ColorRef(30, 30, 30)  // 编辑区 #1e1e1e
	cStatus    = types.ColorRef(0, 122, 204) // 强调色：分隔条 hover/拖动高亮 #007acc
	cStatusBar = types.ColorRef(45, 45, 48)  // 状态栏底色（深灰、低调，不用刺眼蓝）
	cBorder    = types.ColorRef(45, 45, 45)  // 分隔线 #2d2d2d
	cText    = types.ColorFromRGB(204, 204, 204)
	cTextDim = types.ColorFromRGB(140, 140, 140)

	dropHintBg = &types.Color{R: 88, G: 166, B: 255, A: 55} // 拖拽目标区高亮半透明强调底
)

const (
	titleBarH = 36
	statusH   = 26
	toggleW   = 40 // 标题栏面板开关按钮宽
	winBtnW   = 46 // 标题栏窗口按钮宽
	dividerW  = 4  // 停靠分隔条粗细（细、低调；hover/拖动显蓝）

	appVersion = "v0.1.0" // 状态栏版本号
)

func main() {
	
	if os.Getenv("COMPANION_SHOT") != "" {
		renderShot()
		return
	}
	settingspanel.Load()          // 读安装目录 config/ 的全局设置（LLM 服务商/Key/模型 + 上次项目），供 buildProvider 用
	core.LoadLastProject()       // 恢复上次打开的项目根（文件树/终端/agent 统一用它）
	editorpanel.Editor.RestoreSession() // 恢复上次工作区里打开的文件标签（首帧即带上，读各文件磁盘当前内容）

	application = app.NewApplication()
	application.SetRootWidget(&Shell{})

	// 统一所有菜单（右键上下文菜单 / 编辑器右键 / 标题栏下拉）为 GitHub 深色，匹配应用主题。
	widget.SetMenuTheme(*ui.Bg, *ui.Fg, *ui.Accent, *ui.Border, *ui.FgMuted)
	// 对话框/抽屉也统一深色（设置面板、新建/重命名输入框、确认框等），否则白底面板与深色内容冲突。
	widget.SetDialogTheme(*ui.BgSubtle, *ui.Fg, *ui.FgMuted)
	// 下拉选择器（设置里的 服务商/模型 选择框）统一深色，否则 el 浅色白底与深色对话框冲突。
	widget.SetSelectTheme(*ui.Bg, *ui.Fg, *ui.Border, *ui.Bg, *ui.FgMuted)

	// CodeEditor 切暗色主题（VS Code Dark+），与窗口布局（cEditor=#1e1e1e 等）统一。
	widget.SetTheme(widget.DarkTheme())
	// 编辑器右键改用 companion 自定义菜单（editorContentMenu：撤销/剪贴/全选 + 结构化语言视图切换），
	// 不弹 CodeEditor 组件自带菜单。
	widget.SuppressEditorContextMenu = true

	// 光标位置变化时刷新状态栏（Ln X, Col Y 显示）。
	editorpanel.OnCursorMoved = func() {
		if shellStateRef != nil {
			shellStateRef.SetState()
		}
	}

	// Ctrl+S 保存当前编辑器标签（全局快捷键，优先于焦点 Widget）。VK_S=0x53。
	application.ShortcutManager.Register(0x53, event.ModCtrl, func() { editorpanel.Editor.Save() }, "Ctrl+S 保存")
	application.ShortcutManager.Register(0x46, event.ModCtrl, func() { chatpanel.TheState.ToggleSearch() }, "Ctrl+F 搜索对话") // VK_F
	application.ShortcutManager.Register(0xBC, event.ModCtrl, func() { settingspanel.OpenDialog() }, "Ctrl+, 打开设置")               // VK_OEM_COMMA

	// ── 工作区关闭回调：未保存提示 → 清除编辑器标签 + 隐藏面板 + 重建 shell → IDE 欢迎页 ──
	core.OnCloseProject = func() {
		editorpanel.Editor.ConfirmCloseAll(func() {
			editorpanel.MarkWorkspaceClosed()
			if shellStateRef != nil {
				shellStateRef.panels = state.IdeWelcomePanels()
				shellStateRef.SetState()
			}
		})
	}
	core.OnClearWorkspace = func() {
		editorpanel.Editor.ConfirmCloseAll(func() {
			editorpanel.MarkWorkspaceClosed()
			if shellStateRef != nil {
				shellStateRef.panels = state.IdeWelcomePanels()
				shellStateRef.SetState()
			}
		})
	}

	application.Ready = func() {
		// 标题栏命中区：顶部 titleBarH 高、右侧 6 个按钮（3 面板开关 + 3 窗口）宽除外 → 系统接管拖动/双击最大化。
		application.SetTitleBar(titleBarH, 3*toggleW+3*winBtnW)
		application.EnableWindowEffects() // DWM 阴影 + Win11 圆角

		// 窗口就绪后加载对话历史（此时 framework 已就绪，SetState 安全）
		chatpanel.TheState.LoadHistory()
	}

	cfg := app.Config{Title: "伴随式 CodeAgent", Width: 1200, Height: 760, Resizable: true, Borderless: true}
	if err := application.Run(cfg); err != nil {
		fmt.Println("运行失败:", err)
	}
}

// renderShot 无窗口渲染整个主窗壳到 PNG（环境变量 COMPANION_SHOT 触发），用于布局/视觉自检。
func renderShot() {
	shot("companion_shot.png", state.DefaultPanels())
}

func shot(name string, p *state.Panels) {
	const w, h = 1200, 760
	sk := canvas.NewSkiaCanvas(w, h)
	defer sk.Release()
	pipe := render.NewPipeline(w, h, sk)
	pipe.SetRootElement(widget.CreateElementFor(&Shell{initial: p}))
	if err := pipe.Render(); err != nil {
		fmt.Println("render:", err)
		return
	}
	if err := sk.SaveToPNG(name); err != nil {
		fmt.Println("save:", err)
		return
	}
	fmt.Println("✅", name, "已保存")
}

// ─── 主窗壳（有状态）：标题栏 + 主体（左|中|右）+ 状态栏 ───────

type Shell struct {
	widget.StatefulWidget
	initial *state.Panels // 可选初始布局（renderShot 注入不同状态自检；nil=默认全开）
}

func (sh *Shell) CreateState() widget.State {
	p := sh.initial
	if p == nil {
		if len(core.Folders) > 0 {
			p = state.DefaultPanels() // 已有工作区：显示所有面板
		} else {
			p = state.IdeWelcomePanels() // 无工作区：隐藏面板，仅中列展示 IDE 欢迎页
		}
	}
	s := &shellState{panels: p}
	shellStateRef = s
	return s
}

type shellState struct {
	widget.BaseState
	panels   *state.Panels // 停靠布局的唯一真相来源
	leftView string        // 左栏当前视图标签："files" / "git"（复刻参考左区多面板 tab 切换）

	// 面板拖拽停靠（Phase 2）：拖动手柄期间记录正在拖的面板组 + 起点/当前鼠标坐标（窗口坐标）。
	dragPanel              string
	dragSX, dragSY         float64 // 起点
	dragCX, dragCY         float64 // 当前
}

func (s *shellState) Build(ctx widget.BuildContext) widget.Widget {
	shell := widget.VBox(
		s.titleBar(),
		expand(s.body()),
		statusBar(),
	)
	if s.dragPanel == "" {
		return shell
	}
	// 拖拽中：用自绘叠层(PaintLayer)画「目标区高亮 + 跟随光标的半透明面板影」。
	// 关键：拖动中只重绘此叠层(onGrabMove→OnNeedsRepaint)，不重建整个 shell→跟手不卡。
	return widget.NewStack(
		shell,
		&widget.PaintLayer{OnPaint: s.paintDragOverlay},
	)
}

func dragPanelName(id string) string {
	if n := map[string]string{"files": "文件面板", "chat": "对话面板", "terminal": "终端面板"}[id]; n != "" {
		return n
	}
	return id
}

// zoneRect 估算某停靠区在窗口中的矩形(用于拖拽高亮)。
func (s *shellState) zoneRect(z state.Zone, w, h float64) (x, y, rw, rh float64) {
	p := s.panels
	top, bot := float64(titleBarH), h-statusH
	leftEdge := 0.0
	if p.Left {
		leftEdge = p.LeftW + dividerW
	}
	rightEdge := w
	if p.Right {
		rightEdge = w - p.RightW - dividerW
	}
	switch z {
	case state.ZoneLeft:
		return 0, top, p.LeftW, bot - top
	case state.ZoneRight:
		return w - p.RightW, top, p.RightW, bot - top
	case state.ZoneBottom:
		return leftEdge, bot - p.BottomH, rightEdge - leftEdge, p.BottomH
	}
	return 0, top, 0, 0
}

// paintDragOverlay 自绘拖拽叠层：目标区半透明高亮 + 跟随光标的半透明面板影。仅重绘、不重建。
func (s *shellState) paintDragOverlay(cvs canvas.Canvas, ox, oy, w, h float64) {
	if s.dragPanel == "" {
		return
	}
	// 目标区高亮
	if tz, ok := s.dragTargetZone(); ok {
		zx, zy, zw, zh := s.zoneRect(tz, w, h)
		if zw > 0 && zh > 0 {
			fp := paint.DefaultPaint()
			fp.Color = *dropHintBg
			cvs.DrawRect(ox+zx, oy+zy, zw, zh, fp)
			sp := paint.DefaultStrokePaint()
			sp.Color, sp.StrokeWidth = *cStatus, 2
			cvs.DrawRect(ox+zx, oy+zy, zw, zh, sp)
			f := canvas.DefaultFont()
			f.Size = 13
			canvas.DrawTextAligned(cvs, "停靠到此", types.Rect{X: ox + zx, Y: oy + zy, Width: zw, Height: zh}, f, *ui.White, canvas.HAlignCenter, canvas.VAlignMiddle)
		}
	}
	// 跟随光标的半透明面板影（居中于光标）
	const gw, gh = 210.0, 140.0
	gx, gy := s.dragCX-gw/2, s.dragCY-gh/2
	fp := paint.DefaultPaint()
	fp.Color = types.Color{R: 88, G: 166, B: 255, A: 70}
	cvs.DrawRoundedRect(gx, gy, gw, gh, 8, fp)
	sp := paint.DefaultStrokePaint()
	sp.Color, sp.StrokeWidth = *ui.Accent, 2
	cvs.DrawRoundedRect(gx, gy, gw, gh, 8, sp)
	widget.PaintLucide(cvs, "move", gx+gw/2-13, gy+gh/2-22, 26, 2, *ui.White)
	f := canvas.DefaultFont()
	f.Size = 13
	canvas.DrawTextAligned(cvs, "移动 "+dragPanelName(s.dragPanel), types.Rect{X: gx, Y: gy + gh/2 + 8, Width: gw, Height: 20}, f, *ui.White, canvas.HAlignCenter, canvas.VAlignMiddle)
}

// titleBar 自绘标题栏：左 logo+标题（拖动区）| 右 面板开关 ×3 + 窗口按钮 ×3。
func (s *shellState) titleBar() widget.Widget {
	p := s.panels
	// logo（固定宽，不撑开）+ 菜单（嵌在标题栏）+ 弹性拖动空白 + 面板开关 + 窗口按钮。
	// 菜单/按钮经 ClickTarget 命中判定为可点(HTCLIENT)，空白区交系统拖动(HTCAPTION，见 app.SetTitleBar)。
	kids := []widget.Widget{
		widget.Div( // app 图标徽标（仅图标、无文字，复刻参考 icon.svg 的 Pair 标志）
			widget.Style{Height: titleBarH, Padding: types.EdgeInsetsLTRB(10, 0, 4, 0),
				FlexDirection: "row", AlignItems: "center"},
			logo.Small(),
		),
	}
	kids = append(kids, s.titleMenus()...)
	// 居中标题：当前工作区名 + 应用名（让用户一眼看到打开的是哪个项目）；占位空白兼作拖动区。
	kids = append(kids, expand(widget.Div(widget.Style{})))
	kids = append(kids,
		widget.Lucide("folder", widget.IconSize(13), widget.IconColor(cText)),
		widget.Div(widget.Style{Width: 6}),
		label(filepath.Base(core.Root()), cText, 12),
		label("  —  Pair CodeAgent", cTextDim, 12),
	)
	kids = append(kids, expand(widget.Div(widget.Style{})))
	kids = append(kids,
		s.toggleBtn("panel-left", state.ZoneLeft, p.Left),
		s.toggleBtn("panel-bottom", state.ZoneBottom, p.Bottom),
		s.toggleBtn("panel-right", state.ZoneRight, p.Right),
		winButton("minus", func() { application.Minimize() }),
		winButton(maxRestoreIcon(), func() { application.ToggleMaximize() }),
		winButton("x", func() { application.Close() }, winCloseRed),
	)
	return widget.Div(
		widget.Style{Height: titleBarH, BackgroundColor: cTitle, FlexDirection: "row", AlignItems: "center"},
		kids,
	)
}

// toggleBtn 面板开关：图标亮=展开、暗=收起，点击翻转该停靠区。
func (s *shellState) toggleBtn(icon string, z state.Zone, on bool) widget.Widget {
	col := cTextDim
	if on {
		col = cText
	}
	return &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Lucide(icon, widget.IconSize(16), widget.IconColor(col))},
		OnClick:           func() { s.panels.Toggle(z); s.SetState() },
		Color:             *cTitle,
		MinWidth:          toggleW,
		MinHeight:         titleBarH,
	}
}

// winCloseRed 关闭键悬停红（复刻参考 TitleBar 的 #e81123）。
var winCloseRed = types.ColorFromHex("#e81123")

// winButton 标题栏窗口按钮；可选 hover 传悬停底色（关闭键传红，其余省略＝自动变暗）。
func winButton(icon string, onClick func(), hover ...types.Color) widget.Widget {
	b := &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Lucide(icon, widget.IconSize(15), widget.IconColor(cText))},
		OnClick:           onClick,
		Color:             *cTitle, // 同标题栏
		MinWidth:          winBtnW,
		MinHeight:         titleBarH,
	}
	if len(hover) > 0 {
		b.HoverColor = hover[0]
	}
	return b
}

// maxRestoreIcon 据最大化状态切「最大化(单方块)/还原(叠层方块)」图标（复刻参考）。
func maxRestoreIcon() string {
	if application != nil && application.IsMaximized() {
		return "minimize" // 还原（复刻参考用 lucide Minimize：角内收）
	}
	return "square" // 最大化：单方块
}

// titleMenus 标题栏内嵌菜单——**1:1 复刻参考源 TitleBar.tsx 的菜单结构**（文件/编辑/视图/终端/Agent/帮助，
// 项/顺序/分隔/快捷键/勾选照搬）。companion 暂无对应功能的项置灰 Disabled（诚实保真，非删项）。
// 嵌在标题栏内可点靠 ClickTarget 命中判定（见 app.SetTitleBar / win32 NCHITTEST）。
func (s *shellState) titleMenus() []widget.Widget {
	p := s.panels
	return []widget.Widget{
		menuBarBtn("文件", []widget.DropdownItem{
			{Label: "新建项目", Command: "file.newProject"},
			{Label: "新建文件", Shortcut: "Ctrl+N", Command: "file.new"},
			{Label: "打开文件", Shortcut: "Ctrl+O", Command: "file.open"},
			{Label: "打开文件夹", Shortcut: "Ctrl+K Ctrl+O", Command: "file.openFolder"},
			{Label: "添加文件夹到工作区", Shortcut: "Ctrl+Shift+O", Command: "file.addFolder"},
			{Label: "保存", Shortcut: "Ctrl+S", Command: "file.save", Divided: true},
			{Label: "保存工作区", Command: "file.saveWorkspace", Divided: true},
			{Label: "管理工作区...", Command: "file.manageWorkspace"},
			{Label: "关闭项目", Command: "file.closeProject", Divided: true},
			{Label: "关闭工作区", Command: "file.closeWorkspace"},
			{Label: "退出", Shortcut: "Alt+F4", Command: "file.quit", Divided: true},
		}, s.onFileMenu),
		menuBarBtn("编辑", []widget.DropdownItem{
			{Label: "撤销", Shortcut: "Ctrl+Z", Command: "edit.undo"},
			{Label: "重做", Shortcut: "Ctrl+Y", Command: "edit.redo"},
			{Label: "剪切", Shortcut: "Ctrl+X", Command: "edit.cut", Divided: true},
			{Label: "复制", Shortcut: "Ctrl+C", Command: "edit.copy"},
			{Label: "粘贴", Shortcut: "Ctrl+V", Command: "edit.paste"},
			{Label: "跨文件搜索", Shortcut: "Ctrl+Shift+F", Command: "edit.searchfiles", Divided: true},
			{Label: "对话内搜索", Shortcut: "Ctrl+F", Command: "edit.chatsearch"},
		}, s.onEditMenu),
		menuBarBtn("视图", []widget.DropdownItem{
			{Label: "专注模式", Shortcut: "Ctrl+K", Disabled: true},
			{Label: "文件树", Shortcut: "Ctrl+B", Command: "view.files", Checked: p.Left && s.leftView != "git", Divided: true},
			{Label: "搜索", Command: "view.search", Checked: p.Left && s.leftView == "search"},
			{Label: "Git", Command: "view.git", Checked: p.Left && s.leftView == "git"},
			{Label: "对话", Shortcut: "Ctrl+Shift+C", Command: "view.chat", Checked: p.Right},
			{Label: "终端", Shortcut: "Ctrl+J", Command: "view.terminal", Checked: p.Bottom},
			{Label: "放大", Shortcut: "Ctrl+=", Command: "view.zoomIn", Divided: true},
			{Label: "缩小", Shortcut: "Ctrl+-", Command: "view.zoomOut"},
			{Label: "切换 Minimap", Command: "view.toggleMinimap"},
			{Label: "导出当前对话", Command: "view.export"},
		}, s.onViewMenu),
		menuBarBtn("终端", termMenuItems(), s.onTerminalMenu),
		menuBarBtn("Agent", []widget.DropdownItem{
			{Label: "Agent 监控面板", Shortcut: "Ctrl+Shift+M", Command: "agent.monitor"},
			{Label: "性能监控", Command: "agent.perf", Divided: true},
			{Label: "性能测试", Command: "agent.perfdemo", Divided: true},
			{Label: "进化图（EvoMap）", Disabled: true},
			{Label: "探索项目知识库", Command: "agent.explore", Divided: true},
		}, s.onAgentMenu),
		menuBarBtn("帮助", []widget.DropdownItem{
			{Label: "扩展市场", Command: "help.marketplace"},
			{Label: "打开设置", Shortcut: "Ctrl+,", Command: "help.settings"},
			{Label: "更新日志", Command: "help.changelog", Divided: true},
			{Label: "关于", Command: "help.about", Divided: true},
			{Label: "开发者工具", Shortcut: "F12", Disabled: true},
		}, s.onHelpMenu),
	}
}

func (s *shellState) onFileMenu(cmd string) {
	switch cmd {
	case "file.newProject":
		core.NewProjectViaDialog()
	case "file.new":
		ctxmenupanel.NewEntryIn(core.Root(), false)
	case "file.open":
		ctxmenupanel.OpenFileViaDialog()
	case "file.openFolder":
		ctxmenupanel.OpenFolderViaDialog()
	case "file.addFolder":
		ctxmenupanel.AddFolderViaDialog()
	case "file.save":
		editorpanel.Editor.Save()
	case "file.saveWorkspace":
		core.SaveWorkspaceMenu()
	case "file.manageWorkspace":
		core.ShowManager()
	case "file.closeProject":
		core.CloseProjectMenu()
	case "file.closeWorkspace":
		core.CloseWorkspaceMenu()
	case "file.quit":
		application.Close()
	}
}

func (s *shellState) onEditMenu(cmd string) {
	switch cmd {
	case "edit.undo":
		widget.RunEditorCommand("undo")
	case "edit.redo":
		widget.RunEditorCommand("redo")
	case "edit.cut":
		widget.RunEditorCommand("cut")
	case "edit.copy":
		widget.RunEditorCommand("copy")
	case "edit.paste":
		widget.RunEditorCommand("paste")
	case "edit.chatsearch":
		chatpanel.TheState.ToggleSearch()
	case "edit.searchfiles":
		s.showLeft("search")
	}
}

func (s *shellState) onViewMenu(cmd string) {
	switch cmd {
	case "view.files":
		s.showLeft("files")
	case "view.search":
		s.showLeft("search")
	case "view.git":
		s.showLeft("git")
	case "view.chat":
		s.panels.Toggle(state.ZoneRight)
		s.SetState()
	case "view.terminal":
		s.panels.Toggle(state.ZoneBottom)
		s.SetState()
	case "view.export":
		chatpanel.TheState.ExportActive()
	case "view.zoomIn":
		menuactions.SetEditorFontSize(menuactions.EditorFontSize() + 1)
	case "view.zoomOut":
		menuactions.SetEditorFontSize(menuactions.EditorFontSize() - 1)
	case "view.toggleMinimap":
		core.Settings.HideMinimap = !core.Settings.HideMinimap
		core.Save()
		menuactions.Relayout()
	}
}

// termMenuItems 菜单栏「终端」下拉：列出 CMD/PowerShell/Git Bash，本机没探测到的灰显（Disabled）。
func termMenuItems() []widget.DropdownItem {
	detected := map[string]bool{}
	for _, sh := range termpanel.AvailableShells() {
		detected[sh.Code] = true
	}
	return []widget.DropdownItem{
		{Label: "新建 CMD", Command: "term.cmd", Disabled: !detected["cmd"]},
		{Label: "新建 PowerShell", Command: "term.powershell", Disabled: !detected["powershell"]},
		{Label: "新建 Git Bash", Command: "term.gitbash", Disabled: !detected["gitbash"]},
	}
}

func (s *shellState) onTerminalMenu(cmd string) {
	code := map[string]string{"term.cmd": "cmd", "term.powershell": "powershell", "term.gitbash": "gitbash"}[cmd]
	if code == "" {
		return
	}
	if !s.panels.Bottom { // 终端面板原来没显示：显示它，并把默认标签设成所选 shell（不凭空多一个 cmd 标签）
		s.panels.Toggle(state.ZoneBottom)
		termpanel.Mgr().SetActiveShell(code)
	} else { // 面板已开：新开一个该 shell 的标签
		termpanel.Mgr().NewTabWithShell(code)
	}
	s.SetState()
}

func (s *shellState) onAgentMenu(cmd string) {
	switch cmd {
	case "agent.monitor":
		menuactions.ShowAgentMonitor()
	case "agent.perf":
		menuactions.ShowPerfMonitor()
	case "agent.perfdemo":
		menuactions.ShowPerfDemo()
	case "agent.explore":
		chatpanel.TheState.ExploreKnowledgeBase()
	}
}

func (s *shellState) onHelpMenu(cmd string) {
	switch cmd {
	case "help.settings":
		settingspanel.OpenDialog()
	case "help.about":
		showAboutDialog()
	case "help.changelog":
		menuactions.ShowContentDialog("更新日志", 580,
			widget.NewScrollView(ui.TextC(menuactions.ChangelogText, *ui.Fg, 12)))
	case "help.marketplace":
		menuactions.ShowMarketplace()
	}
}

// showAboutDialog 关于对话框。
func showAboutDialog() {
	menuactions.ShowContentDialog("关于", 480,
		widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", Padding: types.EdgeInsets(8)},
			ui.TextC("Pair CodeAgent", *ui.Fg, 16),
			widget.Div(widget.Style{Height: 8}),
			ui.TextC("v0.1.0", *ui.FgMuted, 11),
			widget.Div(widget.Style{Height: 12}),
			ui.TextC(`用 goui（Go 自绘 UI 库、Skia GPU 渲染）全 Go 重写的 IDE 式 AI 编码助手。
复刻参考伴随式 CodeAgent（Electron + React + TypeScript）的 UI 与交互逻辑。`, *ui.Fg, 12),
			widget.Div(widget.Style{Height: 12}),
			ui.TextC("技术栈：Go + goui + SkiaSharp + Lucide 图标", *ui.FgMuted, 11),
		),
	)
}

func menuBarBtn(name string, items []widget.DropdownItem, onCmd func(string)) widget.Widget {
	// 用 Child=label（非 Text）：Button 对纯文本按钮(child==nil)会强加 64 最小宽 → 菜单项过宽；
	// 给 child 则按内容紧凑（label+padding，约 40px）。
	trigger := &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: label(name, cText, 12)},
		Color:             *cTitle,
		MinHeight:         titleBarH,
		Padding:           types.EdgeInsetsLTRB(9, 0, 9, 0),
	}
	return widget.NewDropdown(trigger, items...).WithOnCommand(onCmd).WithPlacement(widget.PlacementBottomStart)
}

// body 主体：左栏 | 中列（编辑区+底部）| 右栏。各区放哪个面板组由 Panels.*Panel 决定（可拖拽换位，
// 一区一组、移动=互换；编辑器恒居中）。分隔条可拖动调尺寸（兼作分隔线）。
func (s *shellState) body() widget.Widget {
	p := s.panels
	chatpanel.TheState.InputAreaH = p.BottomH // 对话输入区高 = 底部区高，使两者底部对齐
	cols := []widget.Widget{}
	if p.Left {
		cols = append(cols,
			widget.Div(
				widget.Style{Width: p.LeftW, BackgroundColor: cSide, FlexDirection: "column", AlignItems: "stretch"},
				expand(s.zoneInner(state.ZoneLeft)),
			),
			vDivide(func(d float64) {
				p.LeftW = state.Clamp(p.LeftW+d, state.MinSideW, state.MaxSideW)
				s.SetState()
			}),
		)
	}
	cols = append(cols, expand(s.midColumn()))
	if p.Right {
		rw := p.RightW
		if p.RightPanel == "chat" { // 对话在右栏：展开对话列表时整栏加宽
			rw = rightColW(p.RightW)
		}
		cols = append(cols,
			vDivide(func(d float64) {
				p.RightW = state.Clamp(p.RightW-d, state.MinSideW, state.MaxSideW) // 右栏左侧条：右拖变窄
				s.SetState()
			}),
			widget.Div(
				widget.Style{Width: rw, BackgroundColor: ui.Bg, FlexDirection: "column", AlignItems: "stretch"},
				expand(s.zoneInner(state.ZoneRight)),
			),
		)
	}
	return flexRow(cols...)
}

// midColumn 中间列：编辑区（撑满，恒居中）+ 底部区面板（按状态）。
func (s *shellState) midColumn() widget.Widget {
	p := s.panels
	rows := []widget.Widget{expand(editorpanel.MidContent())}
	if p.Bottom {
		rows = append(rows,
			hDivide(func(d float64) {
				p.BottomH = state.Clamp(p.BottomH-d, state.MinBotH, state.MaxBotH) // 底栏上侧条：下拖变矮
				s.SetState()
			}),
			widget.Div(
				widget.Style{Height: p.BottomH, BackgroundColor: cSide, FlexDirection: "column", AlignItems: "stretch"},
				expand(s.zoneInner(state.ZoneBottom)),
			),
		)
	}
	return flexCol(rows...)
}

// zoneInner 某区的内容：面板组 + 右上角拖拽手柄；拖拽中在目标区叠加「停靠到此」高亮。
func (s *shellState) zoneInner(z state.Zone) widget.Widget {
	id := s.panels.PanelIn(z)
	// 目标区高亮在 paintDragOverlay 里自绘（不放这里→拖动中不必重建 zone，跟手不卡）。
	return widget.NewStack(
		s.panelGroup(id),
		widget.NewPositioned(s.dragGrip(id)).WithTop(5).WithRight(7).WithZIndex(50),
	)
}

// panelGroup 据面板组 id 返回内容（files=文件/搜索/Git 标签组；chat=对话；其余走 panelBody）。
func (s *shellState) panelGroup(id string) widget.Widget {
	switch id {
	case "files":
		return s.filesGroup()
	case "chat":
		return chatpanel.Area()
	default:
		return panelBody(id) // terminal 等
	}
}

// dragGrip 面板右上角拖拽手柄：按住向 左/右/下 拖→停靠到对应区（与该区原面板互换）；
// 微小移动＝点击→循环换到下一区（保留 Phase 1 的便捷换位）。
func (s *shellState) dragGrip(panelID string) widget.Widget {
	return &widget.DragGrip{
		Icon: "move", Box: 22, IconSz: 14, Color: cText, Bg: *cTitle, // 醒目按钮：圆角底 + 亮图标
		OnStart: func(x, y float64) { s.onGrabStart(panelID, x, y) },
		OnMove:  func(x, y float64) { s.onGrabMove(x, y) },
		OnEnd:   func(x, y float64) { s.onGrabEnd(panelID, x, y) },
	}
}

func (s *shellState) onGrabStart(panelID string, x, y float64) {
	s.dragPanel = panelID
	s.dragSX, s.dragSY, s.dragCX, s.dragCY = x, y, x, y
	s.SetState()
}

func (s *shellState) onGrabMove(x, y float64) {
	s.dragCX, s.dragCY = x, y
	if widget.OnNeedsRepaint != nil {
		widget.OnNeedsRepaint() // 仅重绘叠层(PaintLayer)，不重建 shell→跟手流畅
	}
}

func (s *shellState) onGrabEnd(panelID string, x, y float64) {
	s.dragCX, s.dragCY = x, y
	tz, ok := s.dragTargetZone()
	s.dragPanel = ""
	switch {
	case ok:
		s.panels.Move(panelID, tz)
	default:
		if dx, dy := x-s.dragSX, y-s.dragSY; dx*dx+dy*dy < 36 { // <6px → 当点击：循环换位
			s.panels.Move(panelID, nextZone(s.panels.ZoneOf(panelID)))
		}
	}
	s.SetState()
}

// dragTargetZone 据拖动方向（起点→当前）判定目标区：水平为主→左/右，向下→底，向上/不足阈值→无。
func (s *shellState) dragTargetZone() (state.Zone, bool) {
	dx, dy := s.dragCX-s.dragSX, s.dragCY-s.dragSY
	const th = 45.0
	if dx*dx+dy*dy < th*th {
		return state.ZoneLeft, false
	}
	if math.Abs(dx) >= math.Abs(dy) {
		if dx > 0 {
			return state.ZoneRight, true
		}
		return state.ZoneLeft, true
	}
	if dy > 0 {
		return state.ZoneBottom, true
	}
	return state.ZoneLeft, false // 向上无目标区
}

func nextZone(z state.Zone) state.Zone {
	switch z {
	case state.ZoneLeft:
		return state.ZoneRight
	case state.ZoneRight:
		return state.ZoneBottom
	default:
		return state.ZoneLeft
	}
}

// dropHint 拖拽目标区高亮叠层（半透明强调底 + 「停靠到此」）。
// showLeft 显示左栏并切到 view；若已可见且正是该 view，则隐藏左栏（toggle）。
func (s *shellState) showLeft(view string) {
	cur := s.leftView
	if cur == "" {
		cur = "files"
	}
	if s.panels.Left && cur == view {
		s.panels.Left = false
	} else {
		s.panels.Left = true
		s.leftView = view
	}
	s.SetState()
}

// filesGroup 文件面板组：顶部视图标签条（文件/搜索/Git）+ 当前视图内容（宽/高由所在区包裹决定）。
// filesGroup 左栏：用 goui 的 Tabs 组件（深色+图标+紧凑模式），承载 文件/搜索/Git 三视图。
// 此前手搓标签条，效果差；改用真组件（带平滑滑块动画、受控切换）。
func (s *shellState) filesGroup() widget.Widget {
	views := []string{"files", "search", "git"}
	active := 0
	for i, v := range views {
		if v == s.leftView {
			active = i
		}
	}
	tabs := widget.NewTabs(
		widget.TabPane{Label: "文件", Icon: "folder", Content: &filetreepanel.FileTreePanel{}},
		widget.TabPane{Label: "搜索", Icon: "search", Content: &searchpanel.SearchPanel{}},
		widget.TabPane{Label: "Git", Icon: "git-branch", Content: &gitpanel.GitPanel{}},
	).WithActive(active).WithOnChange(func(i int) { s.leftView = views[i]; s.SetState() })
	tabs.Compact = true       // IDE 紧凑：矮条 + 内容紧贴填满
	tabs.ActiveColor = cText   // 深色主题配色
	tabs.InactiveColor = cTextDim
	tabs.LineColor = *cStatus // 强调蓝滑块/下划线
	tabs.BarColor = *cSide
	return tabs
}

func panelBody(id string) widget.Widget {
	switch id {
	case "files":
		return &filetreepanel.FileTreePanel{} // 左栏「文件」：真实文件树
	case "terminal":
		return termpanel.Area() // 中列底部「终端」：命令运行器
	}
	return widget.Div(
		widget.Style{Padding: types.EdgeInsets(12)},
		label("〔"+id+" 面板占位〕", cTextDim, 12),
	)
}

// statusBar 底部状态栏：左 agent 状态灯 + Git 分支，右 模型 + 编码（VS Code 风）。
// 读实时单例（运行态/分支/模型）；随 shell 重建刷新（面板开关等触发；agent 实时态对话面板已直显）。
func statusBar() widget.Widget {
	running := chatpanel.TheState != nil && chatpanel.TheState.Bridge != nil && chatpanel.TheState.Bridge.IsRunning()
	agentText, dotCol := "就绪", *ui.Success
	if running {
		agentText, dotCol = "运行中", *ui.Warning
	}
	branch := "—"
	if gitpanel.IsRepo() && gitpanel.Branch() != "" {
		branch = gitpanel.Branch()
	}
	model := "未配置模型"
	if core.Settings.Model != "" {
		model = core.Settings.Model
	} else if core.Settings.Provider != "" {
		model = core.Settings.Provider
	}
	return widget.Div(
		widget.Style{Height: statusH, BackgroundColor: cStatusBar, Padding: types.EdgeInsetsLTRB(10, 0, 10, 0),
			FlexDirection: "row", AlignItems: "center"},
		widget.Div(widget.Style{Width: 8, Height: 8, BackgroundColor: &dotCol, BorderRadius: 4}), // agent 状态灯
		widget.Div(widget.Style{Width: 6}),
		label(agentText, cText, 12),
		widget.Div(widget.Style{Width: 16}),
		statusItem("git-branch", branch, cText),
		widget.Div(widget.Style{Width: 16}),
		statusItem("", cursorPosText(), cTextDim),
		expand(widget.Div(widget.Style{})), // 中间撑满，把右侧项推到最右
		statusItem("", model, cText),
		widget.Div(widget.Style{Width: 16}),
		statusItem("", "UTF-8", cText),
		widget.Div(widget.Style{Width: 16}),
		statusItem("", appVersion, cTextDim),
	)
}

// statusItem 状态栏一项：可选图标 + 文本，水平排列、垂直居中。
func statusItem(icon, text string, c types.Color) widget.Widget {
	kids := []widget.Widget{}
	if icon != "" {
		kids = append(kids, widget.Lucide(icon, widget.IconSize(13), widget.IconColor(c)), widget.Div(widget.Style{Width: 5}))
	}
	kids = append(kids, label(text, c, 12))
	return rowCenter(kids...)
}

// cursorPosText 当前光标位置文本 "Ln X, Col Y"（无标签不显示）。
func cursorPosText() string {
	ln, col := editorpanel.Editor.CursorPosition()
	if ln == 0 && col == 0 {
		return ""
	}
	return fmt.Sprintf("Ln %d, Col %d", ln, col)
}

// rowCenter 行容器、交叉轴居中（接受动态子节点）。
func rowCenter(children ...widget.Widget) widget.Widget {
	args := make([]interface{}, 0, len(children)+1)
	args = append(args, widget.Style{FlexDirection: "row", AlignItems: "center"})
	for _, c := range children {
		args = append(args, c)
	}
	return widget.Div(args...)
}

// ─── 小工具 ───────────────────────────────────────────────

func expand(w widget.Widget) widget.Widget {
	return &widget.Expanded{SingleChildWidget: widget.SingleChildWidget{Child: w}, Flex: 1}
}

func label(s string, c types.Color, size float64) widget.Widget {
	t := widget.NewText(s, c)
	t.Font.Size = size
	return t
}

// label1 单行文本（过长自动省略号，不换行）——用于固定行高的列表行（提交历史等），避免换行撑破行高重叠。
func label1(s string, c types.Color, size float64) widget.Widget {
	t := widget.NewText(s, c)
	t.Font.Size = size
	t.MaxLines = 1
	return t
}

// rightColW 右栏宽度：展开对话列表时额外加宽（列表在对话右侧腾出，对话主区不变）。
func rightColW(base float64) float64 {
	if chatpanel.TheState.ShowThreads {
		return base + 190
	}
	return base
}

// flexRow / flexCol：带交叉轴拉伸的行/列容器，接受动态子节点列表。
// （Div 只收 ...interface{}，无法直接 spread []Widget，这里封一层省去手拼。）
func flexRow(children ...widget.Widget) widget.Widget { return flexBox("row", children) }
func flexCol(children ...widget.Widget) widget.Widget { return flexBox("column", children) }

func flexBox(dir string, children []widget.Widget) widget.Widget {
	args := make([]interface{}, 0, len(children)+1)
	args = append(args, widget.Style{FlexDirection: dir, AlignItems: "stretch"})
	for _, c := range children {
		args = append(args, c)
	}
	return widget.Div(args...)
}

// vDivide / hDivide：停靠分隔条。分隔条的样式（粗细 dividerW + 分隔色/高亮色）全部集中在此处
// 参数化——要改样式改这两行即可、无需动 goui 组件；组件只负责"能力"（hover 高亮 / 拖动 / 填充背景）。
func vDivide(onDrag func(float64)) widget.Widget {
	return widget.VResize(*cBorder, *cStatus, onDrag).WithThickness(dividerW)
}
func hDivide(onDrag func(float64)) widget.Widget {
	return widget.HResize(*cBorder, *cStatus, onDrag).WithThickness(dividerW)
}



