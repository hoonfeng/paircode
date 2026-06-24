// 伴随式 CodeAgent —— GWui 重构版入口 + 主窗壳（标题栏 / 停靠区 / 状态栏）。
//
//go:build windows

package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hoonfeng/gwui"
	"github.com/hoonfeng/gwui/app"
	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"

	"github.com/hoonfeng/paircode/cmd/companion/bridge"
	"github.com/hoonfeng/paircode/cmd/companion/codetypes"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/uiapi"

	"github.com/hoonfeng/paircode/cmd/companion/ui"
	chatpanel "github.com/hoonfeng/paircode/cmd/companion/ui/chat"
	ctxmenupanel "github.com/hoonfeng/paircode/cmd/companion/ui/ctxmenu"
	editorpanel "github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	filetreepanel "github.com/hoonfeng/paircode/cmd/companion/ui/filetree"
	gitpanel "github.com/hoonfeng/paircode/cmd/companion/ui/git"
	menuactions "github.com/hoonfeng/paircode/cmd/companion/ui/menu"
	searchpanel "github.com/hoonfeng/paircode/cmd/companion/ui/search"
	settingspanel "github.com/hoonfeng/paircode/cmd/companion/ui/settings"
	"github.com/hoonfeng/paircode/cmd/companion/ui/state"
	termpanel "github.com/hoonfeng/paircode/cmd/companion/ui/terminal"
)

var theApp *app.App
var theDoc *dom.Document
var shellState *ui.ShellState

// pendingEvents 在 theApp 创建前缓冲事件注册（app.New 后统一 flush）。
var pendingEvents []func()

// addEvent 注册事件监听：theApp 已创建则直接注册，否则缓冲到 pendingEvents。
func addEvent(el *dom.Element, typ event.Type, fn event.Listener) {
	if theApp != nil {
		theApp.AddEventListener(el, typ, fn)
	} else {
		pendingEvents = append(pendingEvents, func() { theApp.AddEventListener(el, typ, fn) })
	}
}

// flushPendingEvents 在 app.New 之后执行所有缓冲的事件注册。
func flushPendingEvents() {
	for _, f := range pendingEvents {
		f()
	}
	pendingEvents = nil
}

func main() {
	// runtime.LockOSThread() 由 app.New() 内部调用

	// ── 恢复上次工作区 ──
	settingspanel.Load()
	core.LoadLastProject()

	// ── 创建文档 + 应用主题 ──
	doc := gwui.NewDocument()
	theDoc = doc
	ui.ApplyTheme(doc)

	// ── 创建 Shell 布局 ──
	shellState = &ui.ShellState{
		LeftOpen:   len(core.Folders) > 0,
		RightOpen:  true,
		BottomOpen: len(core.Folders) > 0,
		LeftView:   "files",
		LeftW:      260,
		RightW:     420,
		BottomH:    200,
	}

	body := doc.Body()
	body.SetAttribute("style", "display: flex; flex-direction: column; width: 100%; height: 100%; overflow: hidden;")

	// 标题栏
	titleBar := buildTitleBar(doc)
	body.AppendChild(titleBar)

	// 主体
	mainBody := doc.CreateElement("div")
	mainBody.ClassList().Add("body")
	body.AppendChild(mainBody)

	// 左面板
	leftPanel := doc.CreateElement("div")
	leftPanel.ClassList().Add("left-panel")
	leftPanel.SetAttribute("style", "width: 260px; position: relative;")
	shellState.LeftPanel = leftPanel
	mainBody.AppendChild(leftPanel)

	// 左分隔条
	leftDiv := doc.CreateElement("div")
	leftDiv.ClassList().Add("vdivider")
	shellState.LeftDivider = leftDiv
	mainBody.AppendChild(leftDiv)

	// 中间面板（编辑器 + 底栏）
	centerPanel := doc.CreateElement("div")
	centerPanel.ClassList().Add("center-panel")
	mainBody.AppendChild(centerPanel)

	// 编辑器区域
	editorArea := doc.CreateElement("div")
	editorArea.SetAttribute("style", "flex: 1; overflow: hidden; display: flex; flex-direction: column;")
	centerPanel.AppendChild(editorArea)

	// 底部分隔条
	bottomDiv := doc.CreateElement("div")
	bottomDiv.ClassList().Add("hdivider")
	shellState.BottomDivider = bottomDiv
	centerPanel.AppendChild(bottomDiv)

	// 底栏面板
	bottomPanel := doc.CreateElement("div")
	bottomPanel.ClassList().Add("bottom-panel")
	bottomPanel.SetAttribute("style", "height: 200px;")
	shellState.BottomPanel = bottomPanel
	centerPanel.AppendChild(bottomPanel)

	// 右分隔条
	rightDiv := doc.CreateElement("div")
	rightDiv.ClassList().Add("vdivider")
	shellState.RightDivider = rightDiv
	mainBody.AppendChild(rightDiv)

	// 右面板
	rightPanel := doc.CreateElement("div")
	rightPanel.ClassList().Add("right-panel")
	rightPanel.SetAttribute("style", "width: 420px; display:flex; flex-direction:column;")
	shellState.RightPanel = rightPanel
	mainPanel := doc.CreateElement("div")
		mainPanel.SetAttribute("style", "flex:1;display:flex;flex-direction:column;")
	rightPanel.AppendChild(mainPanel)
	shellState.RightPanel = rightPanel
	mainBody.AppendChild(rightPanel)

	// 状态栏
	statusBar := buildStatusBar(doc)
	body.AppendChild(statusBar)

	// ── 初始可见性 ──
	updatePanelVisibility()

	// ── 分隔条拖动 ──
	setupDividerDrag(doc, leftDiv, "left")
	setupDividerDrag(doc, rightDiv, "right")
	setupDividerDrag(doc, bottomDiv, "bottom")

	// ── 字体路径 ──
	fontPath := findFont()

	// ── 创建 App（theApp / ui.Ctx.App 在此之后可用）──
	a, err := app.New(doc, app.Config{
		Title:     "Pair CodeAgent",
		Width:     1400,
		Height:    900,
		FontPath:  fontPath,
		Resizable: true,
		Centered:  true,
		MinWidth:  800,
		MinHeight: 500,
	})
	if err != nil {
		log.Fatal(err)
	}
	theApp = a
	ui.Ctx.App = a

	// ── flush 缓冲的事件注册（buildTitleBar/setupDividerDrag 等在 app.New 前调用的）──
	flushPendingEvents()
	ui.Ctx.Doc = doc

	// ── 创建面板（此时 ui.Ctx.App 已可用，面板的 on() 能注册事件）──
	chatView := chatpanel.New(doc)
	editorView := editorpanel.New(doc)
	fileTreeView := filetreepanel.New(doc)
	termView := termpanel.New(doc)
	gitView := gitpanel.New(doc)
	searchView := searchpanel.New(doc)

	// 装配面板到容器
	// Tab switching bar at top of left panel
	tabBar := doc.CreateElement("div")
	tabBar.SetAttribute("style", "display:flex;flex-direction:row;height:30px;flex-shrink:0;background:#2d2d2d;border-bottom:1px solid #1e1e1e;")
	tabBtns := make(map[string]*dom.Element)
	for _, t := range []struct{name, icon, label string}{{"files","folder","Files"},{"search","search","Search"},{"git","git-branch","Git"}} {
		btn := doc.CreateElement("div")
		btn.SetAttribute("style", "flex:1;display:flex;align-items:center;justify-content:center;gap:4px;cursor:pointer;font-size:12px;color:#ccc;padding:4px 0;background:#2d2d2d;border-bottom:2px solid transparent;")
		btn.SetAttribute("hover-style", "color:#fff;background:#3c3c3c;")
		icon := doc.CreateElement("span")
		icon.SetAttribute("data-icon", t.icon)
		icon.SetAttribute("style", "width:14px;height:14px;")
		btn.AppendChild(icon)
		lbl := doc.CreateElement("span")
		lbl.SetTextContent(t.label)
		btn.AppendChild(lbl)
		btnN := t.name
		tabBtns[btnN] = btn
		addEvent(btn, event.Click, func(e event.Event) bool {
			if shellState.LeftView == btnN {
				return true
			}
			shellState.LeftView = btnN
			updateLeftView()
			for n, b := range tabBtns {
				if n == btnN {
					b.SetAttribute("style", "flex:1;display:flex;align-items:center;justify-content:center;gap:4px;cursor:pointer;font-size:12px;color:#fff;padding:4px 0;background:#1e1e1e;border-bottom:2px solid #007acc;")
				} else {
					b.SetAttribute("style", "flex:1;display:flex;align-items:center;justify-content:center;gap:4px;cursor:pointer;font-size:12px;color:#ccc;padding:4px 0;background:#2d2d2d;border-bottom:2px solid transparent;")
				}
			}
			switch btnN {
			case "search":
				if ui.Ctx.Search != nil && ui.Ctx.Search.Refresh != nil {
					ui.Ctx.Search.Refresh()
				}
			case "git":
				if ui.Ctx.Git != nil && ui.Ctx.Git.Refresh != nil {
					ui.Ctx.Git.Refresh()
				}
			}
			if theApp != nil {
				theApp.MarkDirty()
			}
			return true
		})
		tabBar.AppendChild(btn)
	}
	leftPanel.AppendChild(tabBar)

	leftPanel.AppendChild(fileTreeView.Element())
	leftPanel.AppendChild(searchView.Element())
	leftPanel.AppendChild(gitView.Element())
	shellState.FileTreeEl = fileTreeView.Element()
	shellState.SearchEl = searchView.Element()
	shellState.GitEl = gitView.Element()
	updateLeftView()
	// Stack views for tab switching
	for _, el := range []*dom.Element{fileTreeView.Element(), searchView.Element(), gitView.Element()} {
		el.SetAttribute("style", "position: absolute; top: 0; left: 0; right: 0; bottom: 0; overflow: auto;")
	}
	// Highlight files tab as active
	if btn, ok := tabBtns["files"]; ok {
		btn.SetAttribute("style", "flex:1;display:flex;align-items:center;justify-content:center;gap:4px;cursor:pointer;font-size:12px;color:#fff;padding:4px 0;background:#1e1e1e;border-bottom:2px solid #007acc;")
	}

	editorArea.AppendChild(editorView.Element())
	bottomPanel.AppendChild(termView.Element())
	mainPanel.AppendChild(chatView.Element())

	// ── 设置全局上下文 ──
	ui.Ctx.Shell = shellState
	ui.Ctx.Chat = &ui.ChatPanelRef{
		Element: chatView.Element(),
		Refresh: chatView.Refresh,
		Send:    chatView.Send,
	}
	ui.Ctx.Editor = &ui.EditorPanelRef{
		Element: editorView.Element(),
		Refresh: editorView.Refresh,
		Open:    editorView.Open,
		OpenAt:  editorView.OpenAt,
		Save:        editorView.Save,
		CloseTab:    editorView.CloseTab,
		CloseOthers: editorView.CloseOtherTabs,
		CloseAll:    editorView.CloseAllTabs,
	}
	ui.Ctx.FileTree = &ui.FileTreePanelRef{
		Element:      fileTreeView.Element(),
		Refresh:      fileTreeView.Refresh,
		RefreshPath:  fileTreeView.RefreshPath,
		SelectPath:   fileTreeView.SelectPath,
		RebuildRoots: fileTreeView.RebuildRoots,
	}
	ui.Ctx.Terminal = &ui.TerminalPanelRef{
		Element: termView.Element(),
		Refresh: termView.Refresh,
	}
	ui.Ctx.Git = &ui.GitPanelRef{
		Element: gitView.Element(),
		Refresh: gitView.Refresh,
	}
	ui.Ctx.Search = &ui.SearchPanelRef{
		Element: searchView.Element(),
		Refresh: searchView.Refresh,
	}

	// ── 注入 uiapi 实现 ──
	injectUIAPI()

	// ── 注入 core 回调 ──
	injectCoreCallbacks()

	// ── 注入面板回调 ──
	injectPanelCallbacks()

	// ── 恢复编辑器会话 ──
	editorpanel.RestoreSession()

	// ── 刷新面板 ──
	fileTreeView.Refresh()
	editorView.Refresh()
	chatView.Refresh()
	gitView.Refresh()

	// ── 注册快捷键 ──
	registerShortcuts()

	// ── 触发重布局（面板在 app.New 后才加入 DOM）──
	a.MarkDirty()

	// ── 启动 ──
	a.Run()
}

// findFont 查找字体文件路径。
func findFont() string {
	candidates := []string{
		"fonts/AlibabaPuHuiTi-3-55-Regular.ttf",
		"../GWui/fonts/AlibabaPuHuiTi-3-55-Regular.ttf",
		"F:/syproject/GWui/fonts/AlibabaPuHuiTi-3-55-Regular.ttf",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

// buildTitleBar 构建标题栏。
func buildTitleBar(doc *dom.Document) *dom.Element {
	bar := doc.CreateElement("div")
	bar.ClassList().Add("titlebar")

	// Logo
	logo := doc.CreateElement("div")
	logo.ClassList().Add("logo")
	logo.SetTextContent("P")
	bar.AppendChild(logo)

	// 菜单栏
	menuBar := doc.CreateElement("div")
	menuBar.ClassList().Add("menu-bar")
	bar.AppendChild(menuBar)

	// 菜单项
	menus := []struct {
		label string
		items []component.DropdownItem
	}{
		{"文件", []component.DropdownItem{
			{Label: "新建文件", OnClick: func() { ctxmenupanel.NewEntryIn(core.Root(), false) }},
			{Label: "打开文件…", OnClick: func() { ctxmenupanel.OpenFileViaDialog() }},
			{Label: "打开文件夹…", OnClick: func() { ctxmenupanel.OpenFolderViaDialog() }},
			{Label: "添加文件夹到工作区…", OnClick: func() { ctxmenupanel.AddFolderViaDialog() }},
			{Label: "保存", OnClick: func() {
				if ui.Ctx.Editor != nil {
					ui.Ctx.Editor.Save()
				}
			}},
			{Label: "关闭项目", OnClick: func() { core.CloseProjectMenu() }},
			{Label: "关闭工作区", OnClick: func() { core.CloseWorkspaceMenu() }},
		}},
		{"编辑", []component.DropdownItem{
			{Label: "查找对话", OnClick: func() { chatpanel.ToggleSearch() }},
		}},
		{"视图", []component.DropdownItem{
			{Label: "文件树", OnClick: func() { togglePanel("left") }},
			{Label: "对话", OnClick: func() { togglePanel("right") }},
			{Label: "终端", OnClick: func() { togglePanel("bottom") }},
			{Label: "专注模式", OnClick: func() { toggleFocusMode() }},
		}},
		{"终端", []component.DropdownItem{
			{Label: "新建终端", OnClick: func() { termpanel.NewTerminal() }},
		}},
		{"Agent", []component.DropdownItem{
			{Label: "Agent 监控", OnClick: func() { menuactions.ShowAgentMonitor() }},
		}},
		{"工具", []component.DropdownItem{
			{Label: "设置…", OnClick: func() { settingspanel.OpenDialog() }},
		}},
		{"帮助", []component.DropdownItem{
			{Label: "关于", OnClick: func() { uiapi.MessageInfo("Pair CodeAgent v0.1.0 (GWui)") }},
		}},
	}

	for _, m := range menus {
		m := m
		// 使用 Dropdown 组件（自带 trigger，无需单独创建 menu-item）
		dd := component.NewDropdown(doc, m.label, m.items)
		menuBar.AppendChild(dd.Element())
	}

	// 居中标题
	titleCenter := doc.CreateElement("div")
	titleCenter.ClassList().Add("title-center")
	titleCenter.SetTextContent("Pair CodeAgent")
	bar.AppendChild(titleCenter)

	// 面板开关
	toggles := doc.CreateElement("div")
	toggles.ClassList().Add("panel-toggles")
	bar.AppendChild(toggles)

	toggleLeft := doc.CreateElement("div")
	toggleLeft.ClassList().Add("toggle-btn")
	toggleLeft.SetAttribute("title", "文件树")
	toggleLeft.AppendChild(createIconSpan(doc, "menu"))
	if shellState.LeftOpen {
		toggleLeft.ClassList().Add("active")
	}
	addEvent(toggleLeft, event.Click, func(e event.Event) bool {
		togglePanel("left")
		return true
	})
	toggles.AppendChild(toggleLeft)

	toggleBottom := doc.CreateElement("div")
	toggleBottom.ClassList().Add("toggle-btn")
	toggleBottom.SetAttribute("title", "终端")
	toggleBottom.AppendChild(createIconSpan(doc, "terminal"))
	if shellState.BottomOpen {
		toggleBottom.ClassList().Add("active")
	}
	addEvent(toggleBottom, event.Click, func(e event.Event) bool {
		togglePanel("bottom")
		return true
	})
	toggles.AppendChild(toggleBottom)

	toggleRight := doc.CreateElement("div")
	toggleRight.ClassList().Add("toggle-btn")
	toggleRight.SetAttribute("title", "对话")
	toggleRight.AppendChild(createIconSpan(doc, "chat"))
	if shellState.RightOpen {
		toggleRight.ClassList().Add("active")
	}
	addEvent(toggleRight, event.Click, func(e event.Event) bool {
		togglePanel("right")
		return true
	})
	toggles.AppendChild(toggleRight)

	// 窗口按钮
	winBtns := doc.CreateElement("div")
	winBtns.ClassList().Add("win-btns")
	bar.AppendChild(winBtns)

	minBtn := doc.CreateElement("div")
	minBtn.ClassList().Add("win-btn")
	minBtn.AppendChild(createIconSpan(doc, "minus"))
	addEvent(minBtn, event.Click, func(e event.Event) bool {
		if theApp != nil {
			theApp.Minimize()
		}
		return true
	})
	winBtns.AppendChild(minBtn)

	maxBtn := doc.CreateElement("div")
	maxBtn.ClassList().Add("win-btn")
	maxBtn.AppendChild(createIconSpan(doc, "checkbox-unchecked"))
	addEvent(maxBtn, event.Click, func(e event.Event) bool {
		if theApp != nil {
			if theApp.IsFullScreen() {
				theApp.Restore()
			} else {
				theApp.Maximize()
			}
		}
		return true
	})
	winBtns.AppendChild(maxBtn)

	closeBtn := doc.CreateElement("div")
	closeBtn.ClassList().Add("win-btn", "close")
	closeBtn.AppendChild(createIconSpan(doc, "close"))
	addEvent(closeBtn, event.Click, func(e event.Event) bool {
		if theApp != nil {
			theApp.Close()
		}
		return true
	})
	winBtns.AppendChild(closeBtn)

	return bar
}

// createIconSpan 创建一个带 data-icon 属性的 span 元素用于渲染 SVG 图标。
// 图标尺寸固定 16x16，颜色继承父元素。
func createIconSpan(doc *dom.Document, iconName string) *dom.Element {
	span := doc.CreateElement("span")
	span.SetAttribute("data-icon", iconName)
	span.SetAttribute("style", "width: 16px; height: 16px; display: inline-flex; align-items: center; justify-content: center;")
	return span
}

// buildStatusBar 构建状态栏。
func buildStatusBar(doc *dom.Document) *dom.Element {
	bar := doc.CreateElement("div")
	bar.ClassList().Add("statusbar")

	left := doc.CreateElement("div")
	left.ClassList().Add("status-left")
	bar.AppendChild(left)

	// Agent 状态灯
	dot := doc.CreateElement("div")
	dot.ClassList().Add("status-dot")
	left.AppendChild(dot)

	agentStatus := doc.CreateElement("span")
	agentStatus.SetTextContent("就绪")
	left.AppendChild(agentStatus)

	// Git 分支
	gitInfo := doc.CreateElement("span")
	gitInfo.ClassList().Add("status-item")
	gitIcon := createIconSpan(doc, "git-branch")
	gitIcon.SetAttribute("style", "width: 14px; height: 14px; display: inline-flex; align-items: center; justify-content: center;")
	gitText := doc.CreateElement("span")
	gitInfo.AppendChild(gitIcon)
	gitInfo.AppendChild(gitText)
	left.AppendChild(gitInfo)

	// 光标位置
	cursorPos := doc.CreateElement("span")
	cursorPos.ClassList().Add("status-item")
	cursorPos.SetTextContent("Ln 1, Col 1")
	left.AppendChild(cursorPos)

	// 右侧
	right := doc.CreateElement("div")
	right.ClassList().Add("status-right")
	bar.AppendChild(right)

	model := doc.CreateElement("span")
	model.SetTextContent("deepseek-chat")
	right.AppendChild(model)

	encoding := doc.CreateElement("span")
	encoding.SetTextContent("UTF-8")
	right.AppendChild(encoding)

	version := doc.CreateElement("span")
	version.SetTextContent("v0.1.0")
	right.AppendChild(version)

	// 定时刷新状态栏
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			if chatpanel.IsRunning() {
				dot.ClassList().Add("running")
				agentStatus.SetTextContent("Agent 运行中…")
			} else {
				dot.ClassList().Remove("running")
				agentStatus.SetTextContent("就绪")
			}
			if branch := gitpanel.Branch(); branch != "" {
				gitText.SetTextContent(branch)
			} else {
				gitText.SetTextContent("")
			}
			if ln, col := editorpanel.CursorPosition(); ln > 0 {
				cursorPos.SetTextContent("Ln " + itoa(ln) + ", Col " + itoa(col))
			}
			if theApp != nil {
				theApp.MarkDirty()
			}
		}
	}()

	return bar
}

// togglePanel 切换面板可见性。
func togglePanel(panel string) {
	switch panel {
	case "left":
		shellState.LeftOpen = !shellState.LeftOpen
	case "right":
		shellState.RightOpen = !shellState.RightOpen
	case "bottom":
		shellState.BottomOpen = !shellState.BottomOpen
	}
	updatePanelVisibility()
	if theApp != nil {
		theApp.MarkDirty()
	}
}

// toggleFocusMode 专注模式：隐藏所有侧栏。
func toggleFocusMode() {
	shellState.FocusMode = !shellState.FocusMode
	updatePanelVisibility()
	if theApp != nil {
		theApp.MarkDirty()
	}
}


// updateLeftView switches the active view in the left panel
func updateLeftView() {
	if shellState == nil {
		return
	}
	showStyle := "position:absolute;top:0;left:0;right:0;bottom:0;display:flex;flex-direction:column;overflow:hidden;"
	hideStyle := "display:none;"
	if shellState.FileTreeEl != nil {
		if shellState.LeftView == "files" {
			shellState.FileTreeEl.SetAttribute("style", showStyle)
		} else {
			shellState.FileTreeEl.SetAttribute("style", hideStyle)
		}
	}
	if shellState.SearchEl != nil {
		if shellState.LeftView == "search" {
			shellState.SearchEl.SetAttribute("style", showStyle)
		} else {
			shellState.SearchEl.SetAttribute("style", hideStyle)
		}
	}
	if shellState.GitEl != nil {
		if shellState.LeftView == "git" {
			shellState.GitEl.SetAttribute("style", showStyle)
		} else {
			shellState.GitEl.SetAttribute("style", hideStyle)
		}
	}
	if theApp != nil {
		theApp.MarkDirty()
	}
}

// updatePanelVisibility 根据状态更新面板可见性。
func updatePanelVisibility() {
	if shellState.FocusMode {
		shellState.LeftPanel.ClassList().Add("hidden")
		shellState.RightPanel.ClassList().Add("hidden")
		shellState.BottomPanel.ClassList().Add("hidden")
		shellState.LeftDivider.ClassList().Add("hidden")
		shellState.RightDivider.ClassList().Add("hidden")
		shellState.BottomDivider.ClassList().Add("hidden")
		return
	}
	setVisible := func(el *dom.Element, show bool) {
		if show {
			el.ClassList().Remove("hidden")
		} else {
			el.ClassList().Add("hidden")
		}
	}
	setVisible(shellState.LeftPanel, shellState.LeftOpen)
	setVisible(shellState.LeftDivider, shellState.LeftOpen)
	setVisible(shellState.RightPanel, shellState.RightOpen)
	setVisible(shellState.RightDivider, shellState.RightOpen)
	setVisible(shellState.BottomPanel, shellState.BottomOpen)
	setVisible(shellState.BottomDivider, shellState.BottomOpen)
}

// setupDividerDrag 设置分隔条拖动。
func setupDividerDrag(doc *dom.Document, divider *dom.Element, side string) {
	var dragging bool
	var startX, startY float32
	var startW, startH float32

	addEvent(divider, event.MouseDown, func(e event.Event) bool {
		me := e.(*event.MouseEvent)
		dragging = true
		startX = me.X
		startY = me.Y
		switch side {
		case "left":
			startW = shellState.LeftW
		case "right":
			startW = shellState.RightW
		case "bottom":
			startH = shellState.BottomH
		}
		divider.ClassList().Add("dragging")
		return true
	})

	addEvent(divider, event.MouseMove, func(e event.Event) bool {
		if !dragging {
			return false
		}
		me := e.(*event.MouseEvent)
		switch side {
		case "left":
			newW := startW + (me.X - startX)
			if newW < ui.MinSideW {
				newW = ui.MinSideW
			}
			if newW > ui.MaxSideW {
				newW = ui.MaxSideW
			}
			shellState.LeftW = newW
			shellState.LeftPanel.SetAttribute("style", "width: "+ftoa(newW)+"px;")
		case "right":
			newW := startW - (me.X - startX)
			if newW < ui.MinChatW {
				newW = ui.MinChatW
			}
			if newW > ui.MaxSideW {
				newW = ui.MaxSideW
			}
			shellState.RightW = newW
			shellState.RightPanel.SetAttribute("style", "width: "+ftoa(newW)+"px;")
		case "bottom":
			newH := startH - (me.Y - startY)
			if newH < ui.MinBotH {
				newH = ui.MinBotH
			}
			if newH > ui.MaxBotH {
				newH = ui.MaxBotH
			}
			shellState.BottomH = newH
			shellState.BottomPanel.SetAttribute("style", "height: "+ftoa(newH)+"px;")
		}
		if theApp != nil {
			theApp.MarkDirty()
		}
		return true
	})

	addEvent(divider, event.MouseUp, func(e event.Event) bool {
		dragging = false
		divider.ClassList().Remove("dragging")
		return true
	})
}

// injectUIAPI 注入 uiapi 实现（Toast / Modal / Overlay）。
func injectUIAPI() {
	// Toast 通知
	var globalToast *component.Toast
	uiapi.MessageFunc = func(text string, kind uiapi.MessageKind) {
		if globalToast == nil {
			globalToast = component.NewToast(theDoc)
		}
		switch kind {
		case uiapi.KindError:
			globalToast.ShowError(text)
		case uiapi.KindWarning:
			globalToast.ShowWarning(text)
		case uiapi.KindSuccess:
			globalToast.ShowSuccess(text)
		default:
			globalToast.ShowInfo(text)
		}
	}
	// 确认对话框
	uiapi.ShowConfirmFunc = func(title, body string, _ uiapi.MessageKind, onConfirm func()) {
		m := component.NewModal(theDoc)
		m.SetTitle(title)
		m.SetBody(body)
		bodyEl := m.Content()
		btnRow := theDoc.CreateElement("div")
		btnRow.SetAttribute("style", "display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px;")
		confirmBtn := component.NewButton(theDoc, "确定")
		confirmBtn.OnClick(func() { m.Hide(); onConfirm() })
		btnRow.AppendChild(confirmBtn.Element())
		cancelBtn := component.NewButton(theDoc, "取消")
		cancelBtn.SetStyle("background-color: #9e9e9e;")
		cancelBtn.OnClick(func() { m.Hide() })
		btnRow.AppendChild(cancelBtn.Element())
		bodyEl.AppendChild(btnRow)
		m.Show()
	}
	// 自定义对话框
	uiapi.ShowDialogFunc = func(title string, _ float32, content, footer interface{}) int {
		m := component.NewModal(theDoc)
		m.SetTitle(title)
		if contentEl, ok := content.(*dom.Element); ok {
			ce := m.Content()
			ce.SetTextContent("")
			ce.AppendChild(contentEl)
		}
		if footerEl, ok := footer.(*dom.Element); ok {
			m.Content().AppendChild(footerEl)
		}
		m.Show()
	return 0
	}
	// 隐藏浮层
	uiapi.HideOverlayFunc = func(id int) {
		dom.HideOverlay(id)
	}
	uiapi.MarkDirtyFunc = func() {
		if theApp != nil {
			theApp.MarkDirty()
		}
	}
	uiapi.RequestFrameFunc = func() {
		if theApp != nil {
			theApp.RequestFrame()
		}
	}
}

// injectCoreCallbacks 注入 core 回调。
func injectCoreCallbacks() {
	core.OnSyncWorkspace = func(primaryChanged bool) {
		if ui.Ctx.FileTree != nil {
			ui.Ctx.FileTree.RebuildRoots()
		}
		if ui.Ctx.Terminal != nil {
			// termpanel.Active().OpenDir(core.Root())
		}
		core.Settings.WorkspaceFolders = append([]string{}, core.Folders...)
		core.Settings.LastProject = core.Root()
		core.Loaded = true
		core.Save()
		// 工作区变更后更新面板可见性
		if len(core.Folders) > 0 {
			shellState.LeftOpen = true
			shellState.BottomOpen = true
		} else {
			shellState.LeftOpen = false
			shellState.BottomOpen = false
		}
		updatePanelVisibility()
		if theApp != nil {
			theApp.MarkDirty()
		}
	}
	core.OnCloseProject = func() {
		shellState.LeftOpen = false
		shellState.BottomOpen = false
		updatePanelVisibility()
		if theApp != nil {
			theApp.MarkDirty()
		}
	}
	core.OnClearWorkspace = func() {
		shellState.LeftOpen = false
		shellState.BottomOpen = false
		updatePanelVisibility()
		if theApp != nil {
			theApp.MarkDirty()
		}
	}
	core.OnShowManager = func() {
		uiapi.MessageInfo("请使用“文件 → 打开文件夹”或拖拽文件夹到窗口")
	}
	core.OnPickFolder = func(title string) string {
		return ctxmenupanel.PickFolder(title)
	}
	core.OnShowPrompt = func(title, initial string, onOk func(string)) {
		m := component.NewModal(theDoc)
		m.SetTitle(title)
		bodyEl := m.Content()
		bodyEl.SetTextContent("")
		inp := component.NewInput(theDoc, initial)
		bodyEl.AppendChild(inp.Element())
		btnRow := theDoc.CreateElement("div")
		btnRow.SetAttribute("style", "display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px;")
		okBtn := component.NewButton(theDoc, "确定")
		okBtn.OnClick(func() { m.Hide(); onOk(inp.Value()) })
		btnRow.AppendChild(okBtn.Element())
		cancelBtn := component.NewButton(theDoc, "取消")
		cancelBtn.SetStyle("background-color: #9e9e9e;")
		cancelBtn.OnClick(func() { m.Hide() })
		btnRow.AppendChild(cancelBtn.Element())
		bodyEl.AppendChild(btnRow)
		m.Show()
	}
}

// injectPanelCallbacks 注入面板间回调。
func injectPanelCallbacks() {
	// 文件树回调
	filetreepanel.OnNodeMenu = func(x, y float64, n *filetreepanel.FileNode) {
		ctxmenupanel.FileNodeMenu(x, y, n)
	}
	filetreepanel.OnEmptyMenu = ctxmenupanel.FileTreeEmptyMenu
	filetreepanel.OnRootMenu = ctxmenupanel.WorkspaceRootMenu
	filetreepanel.OnOpenFolder = ctxmenupanel.OpenFolderViaDialog
	filetreepanel.OnAddFolder = ctxmenupanel.AddFolderViaDialog
	filetreepanel.OnWorkspaceChanged = func(primaryChanged bool) {
		if ui.Ctx.FileTree != nil {
			ui.Ctx.FileTree.RebuildRoots()
		}
		core.Settings.WorkspaceFolders = append([]string{}, core.Folders...)
		core.Settings.LastProject = core.Root()
		core.Loaded = true
		core.Save()
	}

	// 编辑器回调
	editorpanel.OnContentMenu = ctxmenupanel.EditorContentMenu
	editorpanel.OnTabMenu = ctxmenupanel.EditorTabMenu
	editorpanel.OnOpenFile = ctxmenupanel.OpenFileViaDialog
	editorpanel.OnNewFile = func() { ctxmenupanel.NewEntryIn(core.Root(), false) }
	editorpanel.OnOpenFolder = ctxmenupanel.OpenFolderViaDialog
	editorpanel.OnNewProject = core.NewProjectViaDialog
	editorpanel.OnOpenRecent = core.OpenProject
	editorpanel.OnReferences = func() { menuactions.EditorReferences(nil) }
	editorpanel.OnSymbols = func() { menuactions.EditorSymbols(nil) }
	editorpanel.OnCursorMoved = func() {
		if theApp != nil {
			theApp.MarkDirty()
		}
	}

	// 终端回调
	termpanel.OnContextMenu = ctxmenupanel.TerminalMenu

	// Git 回调
	gitpanel.OnTreeRefresh = func() {
		if ui.Ctx.FileTree != nil {
			ui.Ctx.FileTree.Refresh()
		}
	}

	// Bridge 帧泵注入：用 app.SetInterval/ClearInterval 替代 goui animation.Controller
	bridge.SetIntervalFunc = func(interval time.Duration, fn func()) int {
		if theApp == nil {
			return 0
		}
		return theApp.SetInterval(fn, interval)
	}
	bridge.ClearIntervalFunc = func(id int) {
		if theApp != nil {
			theApp.ClearInterval(id)
		}
	}

	// Chat bridge 回调
	chatpanel.NewBridge = func(cs *chatpanel.ChatState) chatpanel.AgentBridge {
		return bridge.NewAgentBridge(cs)
	}

	// Settings 回调
	settingspanel.ApplyIgnoreDirs = bridge.ApplyIgnoreDirs
}

// registerShortcuts 注册全局快捷键。
func registerShortcuts() {
	if theApp == nil {
		return
	}
	theApp.RegisterShortcut("Ctrl+S", func() {
		if ui.Ctx.Editor != nil {
			ui.Ctx.Editor.Save()
		}
	})
	theApp.RegisterShortcut("Ctrl+F", func() {
		chatpanel.ToggleSearch()
	})
	theApp.RegisterShortcut("Ctrl+B", func() {
		togglePanel("left")
	})
	theApp.RegisterShortcut("Ctrl+J", func() {
		togglePanel("bottom")
	})
	theApp.RegisterShortcut("Ctrl+K", func() {
		toggleFocusMode()
	})
	theApp.RegisterShortcut("Ctrl+N", func() {
		ctxmenupanel.NewEntryIn(core.Root(), false)
	})
	theApp.RegisterShortcut("Ctrl+O", func() {
		ctxmenupanel.OpenFileViaDialog()
	})
	theApp.RegisterShortcut("Ctrl+Shift+O", func() {
		ctxmenupanel.AddFolderViaDialog()
	})
	theApp.RegisterShortcut("Ctrl+Shift+F", func() {
		// 跨文件搜索
		shellState.LeftView = "search"
		if !shellState.LeftOpen {
			togglePanel("left")
		}
	})
	theApp.RegisterShortcut("Ctrl+Shift+C", func() {
		togglePanel("right")
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func ftoa(f float32) string {
	// 简单实现：整数部分
	n := int(f)
	return itoa(n)
}

// 确保未使用的 import 不报错
var _ = state.DefaultPanels
var _ = codetypes.CodeLoc{}
