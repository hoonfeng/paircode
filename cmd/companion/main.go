// 伴随式 CodeAgent —— GWui 重构版入口 + 主窗壳。
// 使用 HTML 文件加载外壳布局（resources/html/shell.html），替代程序化 DOM 构建。
// 面板通过 getElementById 挂载到 HTML 容器，各自加载自己的 HTML 模板。
//
//go:build windows && !webonly

package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hoonfeng/gwui/app"
	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"
	"github.com/hoonfeng/gwui/uixml"

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
	marketplacepanel "github.com/hoonfeng/paircode/cmd/companion/ui/marketplace"
	menuactions "github.com/hoonfeng/paircode/cmd/companion/ui/menu"
	searchpanel "github.com/hoonfeng/paircode/cmd/companion/ui/search"
	settingspanel "github.com/hoonfeng/paircode/cmd/companion/ui/settings"
	"github.com/hoonfeng/paircode/cmd/companion/ui/state"
	statspanel "github.com/hoonfeng/paircode/cmd/companion/ui/stats"
	termpanel "github.com/hoonfeng/paircode/cmd/companion/ui/terminal"
)

var theApp *app.App
var theDoc *dom.Document
var shellState *ui.ShellState

func main() {
	// runtime.LockOSThread() 由 app.New() 内部调用

	// ── 恢复上次工作区 ──
	settingspanel.Load()
	core.LoadLastProject()

	// ── ShellState ──
	shellState = &ui.ShellState{
		LeftOpen:   true,
		RightOpen:  true,
		BottomOpen: len(core.Folders) > 0,
		LeftView:   "files",
		LeftW:      260,
		RightW:     55.0, // 百分比（55%）
		BottomH:    200,
	}

	// ── uixml 注册表：注册 HTML 中 onclick 对应的处理函数 ──
	reg := uixml.NewRegistry()
	registerActivityBarHandlers(reg)

	// ── 加载外壳 HTML ──
	doc, err := uixml.LoadFile(ui.ResourcePath("html/shell.html"), reg)
	if err != nil {
		log.Fatalf("加载 shell.html 失败: %v（资源目录=%s）", err, ui.ResourceDir())
	}
	theDoc = doc

	// ── 加载主题 CSS ──
	doc.AddStyleSheet(ui.ReadResourceString("css/theme.css"))

	// ── 获取外壳元素引用 ──
	shellState.LeftPanel = doc.GetElementByID("left-panel")
	shellState.RightPanel = doc.GetElementByID("right-panel")
	shellState.BottomPanel = doc.GetElementByID("bottom-panel")
	shellState.CenterPanel = doc.GetElementByID("center-panel")
	shellState.LeftDivider = doc.GetElementByID("left-divider")
	shellState.RightDivider = doc.GetElementByID("right-divider")
	shellState.BottomDivider = doc.GetElementByID("bottom-divider")

	// ── 字体路径 ──
	fontPath := findFont()

	// ── 创建 App ──
	a, err := app.New(doc, app.Config{
		Title:        "Pair CodeAgent",
		Width:        1400,
		Height:       900,
		FontPath:     fontPath,
		Resizable:    true,
		Centered:     true,
		MinWidth:     800,
		MinHeight:    500,
		UseWebRender: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	theApp = a
	ui.Ctx.App = a
	ui.Ctx.Doc = doc

	// ── 注册外壳事件（toggle/win 按钮、分隔条拖动）──
	registerShellEvents(doc)
	setupDividerDrag(doc, shellState.LeftDivider, "left")
	setupDividerDrag(doc, shellState.RightDivider, "right")
	setupDividerDrag(doc, shellState.BottomDivider, "bottom")

	// ── 构建标题栏下拉菜单 ──
	buildDropdowns(doc)

	// ── 初始可见性 ──
	updatePanelVisibility()

	// ── 初始化剪贴板 ──
	chatpanel.ClipboardWrite = ctxmenupanel.CopyToClipboard

	// ── 创建面板 ──
	chatView := chatpanel.New(doc)
	editorView := editorpanel.New(doc)
	fileTreeView := filetreepanel.New(doc)
	termView := termpanel.New(doc)
	gitView := gitpanel.New(doc)
	searchView := searchpanel.New(doc)

	// ── 装配面板到容器 ──
	viewContainer := doc.GetElementByID("view-container")
	editorArea := doc.GetElementByID("editor-area")
	bottomPanel := doc.GetElementByID("bottom-panel")
	mainPanel := doc.GetElementByID("main-panel")

	viewContainer.AppendChild(fileTreeView.Element())
	viewContainer.AppendChild(searchView.Element())
	viewContainer.AppendChild(gitView.Element())
	// 绝对定位：三个视图叠加，由 updateLeftView 切换显隐
	shellState.FileTreeEl = fileTreeView.Element()
	shellState.SearchEl = searchView.Element()
	shellState.GitEl = gitView.Element()
	updateLeftView()

	editorArea.AppendChild(editorView.Element())
	bottomPanel.AppendChild(termView.Element())
	mainPanel.AppendChild(chatView.Element())

	// 面板挂载到 DOM 树后执行首次初始化渲染（刷新消息列表 + 侧栏）
	chatView.PostInit()

	// ── 设置全局上下文 ──
	ui.Ctx.Shell = shellState
	ui.Ctx.Chat = &ui.ChatPanelRef{
		Element: chatView.Element(),
		Refresh: chatView.Refresh,
		Send:    chatView.Send,
	}
	ui.Ctx.Editor = &ui.EditorPanelRef{
		Element:     editorView.Element(),
		Refresh:     editorView.Refresh,
		Open:        editorView.Open,
		OpenAt:      editorView.OpenAt,
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

	// ── 启动状态栏监控 ──
	startStatusBarMonitor(doc)

	// ── 恢复编辑器会话 ──
	editorpanel.RestoreSession()

	// ── 刷新面板 ──
	fileTreeView.Refresh()
	editorView.Refresh()
	chatView.Refresh()
	gitView.Refresh()

	// ── 注册快捷键 ──
	registerShortcuts()

	// ── 初始化活动栏 active 状态 ──
	syncActivityBar()

	// ── 初始化命令面板事件 ──
	initCommandPalette(doc)

	// ── 触发重布局 ──
	a.MarkDirty()

	// ── 启动 Web UI 服务器（后台 goroutine，不阻塞桌面 GUI）──
	go startWebUI(9090)

	// ── 启动 ──
	a.Run()
}

// registerActivityBarHandlers 注册活动栏图标的 onclick 处理函数（在 HTML 加载前注册）。
func registerActivityBarHandlers(reg *uixml.Registry) {
	// 左面板视图切换：files / search / git
	leftViews := map[string]string{"files": "actFiles", "search": "actSearch", "git": "actGit"}
	for name, id := range leftViews {
		name, id := name, id
		reg.OnClick(id, func(ctx uixml.EventContext) bool {
			// 若已处于该视图且左面板打开，则收起左面板（VS Code 行为）
			if shellState.LeftView == name && shellState.LeftOpen {
				togglePanel("left")
				return true
			}
			shellState.LeftView = name
			if !shellState.LeftOpen {
				shellState.LeftOpen = true
				updatePanelVisibility()
			}
			updateLeftView()
			syncActivityBar()
			switch name {
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
	}
	// 设置
	reg.OnClick("actSettings", func(ctx uixml.EventContext) bool {
		settingspanel.OpenDialog()
		return true
	})
}

// syncActivityBar 同步活动栏图标的 active 状态与 shellState。
func syncActivityBar() {
	if theDoc == nil {
		return
	}
	// 左面板视图图标
	leftAct := map[string]string{"files": "act-files", "search": "act-search", "git": "act-git"}
	for name, id := range leftAct {
		if el := theDoc.GetElementByID(id); el != nil {
			if shellState.LeftOpen && shellState.LeftView == name {
				el.ClassList().Add("active")
			} else {
				el.ClassList().Remove("active")
			}
		}
	}
}

// registerShellEvents 注册标题栏 win 按钮事件（app.New 之后调用）。
func registerShellEvents(doc *dom.Document) {
	// 窗口按钮
	if btn := doc.GetElementByID("win-min"); btn != nil {
		theApp.AddEventListener(btn, event.Click, func(e event.Event) bool {
			if theApp != nil {
				theApp.Minimize()
			}
			return true
		})
	}
	if btn := doc.GetElementByID("win-max"); btn != nil {
		theApp.AddEventListener(btn, event.Click, func(e event.Event) bool {
			if theApp != nil {
				if theApp.IsFullScreen() {
					theApp.Restore()
				} else {
					theApp.Maximize()
				}
			}
			return true
		})
	}
	if btn := doc.GetElementByID("win-close"); btn != nil {
		theApp.AddEventListener(btn, event.Click, func(e event.Event) bool {
			if theApp != nil {
				theApp.Close()
			}
			return true
		})
	}
}

// buildDropdowns 构建标题栏下拉菜单并挂到 #menu-bar。
// 现代化的 AI IDE 菜单体系：每个菜单项都连接到实际功能或显示合理提示。
func buildDropdowns(doc *dom.Document) {
	menuBar := doc.GetElementByID("menu-bar")
	if menuBar == nil {
		return
	}
	menus := []struct {
		label string
		items []component.PopupMenuItem
	}{
		// ── 文件 ──
		{"文件", []component.PopupMenuItem{
			{Label: "新建文件   (Ctrl+N)", OnClick: func() { ctxmenupanel.NewEntryIn(core.Root(), false) }},
			{Label: "打开文件…   (Ctrl+O)", OnClick: func() { ctxmenupanel.OpenFileViaDialog() }},
			{Label: "打开文件夹…", OnClick: func() { ctxmenupanel.OpenFolderViaDialog() }},
			{Label: "添加文件夹到工作区…", OnClick: func() { ctxmenupanel.AddFolderViaDialog() }},
			{Divider: true},
			{Label: "保存   (Ctrl+S)", OnClick: func() {
				if ui.Ctx.Editor != nil {
					ui.Ctx.Editor.Save()
				}
			}},
			{Label: "全部保存", OnClick: func() {
				if ui.Ctx.Editor != nil {
					ui.Ctx.Editor.Save()
					uiapi.MessageSuccess("已保存")
				}
			}},
			{Divider: true},
			{Label: "保存工作区", OnClick: func() { core.SaveWorkspaceMenu() }},
			{Label: "管理工作区文件夹…", OnClick: func() { core.ShowManager() }},
			{Divider: true},
			{Label: "关闭项目", OnClick: func() { core.CloseProjectMenu() }},
			{Label: "关闭工作区", OnClick: func() { core.CloseWorkspaceMenu() }},
		}},
		// ── 编辑 ──
		{"编辑", []component.PopupMenuItem{
			{Label: "撤销   (Ctrl+Z)", OnClick: func() { editorpanel.Editor.Undo() }},
			{Label: "重做   (Ctrl+Shift+Z)", OnClick: func() { editorpanel.Editor.Redo() }},
			{Divider: true},
			{Label: "剪切", OnClick: func() { editorpanel.Editor.CutSelection() }},
			{Label: "复制", OnClick: func() { editorpanel.Editor.CopySelection() }},
			{Label: "粘贴", OnClick: func() { editorpanel.Editor.PasteText() }},
			{Divider: true},
			{Label: "查找对话   (Ctrl+F)", OnClick: func() { chatpanel.ToggleSearch() }},
			{Label: "跨文件搜索   (Ctrl+Shift+F)", OnClick: func() {
				shellState.LeftView = "search"
				if !shellState.LeftOpen {
					shellState.LeftOpen = true
					updatePanelVisibility()
				}
				updateLeftView()
				syncActivityBar()
				if ui.Ctx.Search != nil && ui.Ctx.Search.Refresh != nil {
					ui.Ctx.Search.Refresh()
				}
				if theApp != nil {
					theApp.MarkDirty()
				}
			}},
			{Label: "命令面板   (Ctrl+P)", OnClick: func() { openCommandPalette() }},
		}},
		// ── 视图 ──
		{"视图", []component.PopupMenuItem{
			{Label: "文件资源管理器", OnClick: func() {
				shellState.LeftView = "files"
				if !shellState.LeftOpen {
					togglePanel("left")
				} else {
					updateLeftView()
					syncActivityBar()
					if theApp != nil {
						theApp.MarkDirty()
					}
				}
			}},
			{Label: "搜索", OnClick: func() {
				shellState.LeftView = "search"
				if !shellState.LeftOpen {
					togglePanel("left")
				}
				shellState.LeftOpen = true
				updateLeftView()
				syncActivityBar()
				if ui.Ctx.Search != nil && ui.Ctx.Search.Refresh != nil {
					ui.Ctx.Search.Refresh()
				}
				if theApp != nil {
					theApp.MarkDirty()
				}
			}},
			{Label: "源代码管理", OnClick: func() {
				shellState.LeftView = "git"
				if !shellState.LeftOpen {
					togglePanel("left")
				}
				shellState.LeftOpen = true
				updateLeftView()
				syncActivityBar()
				if ui.Ctx.Git != nil && ui.Ctx.Git.Refresh != nil {
					ui.Ctx.Git.Refresh()
				}
				if theApp != nil {
					theApp.MarkDirty()
				}
			}},
			{Divider: true},
			{Label: "专注模式   (Ctrl+K)", OnClick: func() { toggleFocusMode() }},
			{Label: "切换侧边栏   (Ctrl+B)", OnClick: func() { togglePanel("left") }},
			{Label: "切换终端   (Ctrl+J)", OnClick: func() { togglePanel("bottom") }},
		}},
		// ── 终端 ──
		{"终端", []component.PopupMenuItem{
			{Label: "新建终端", OnClick: func() { termpanel.NewTerminal() }},
			{Divider: true},
			{Label: "清屏", OnClick: func() { termpanel.ClearActive() }},
		}},
		// ── Agent ｜ AI IDE 核心特色菜单 ──
		{"Agent", []component.PopupMenuItem{
			{Label: "启动新任务", OnClick: func() {
				if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
					ui.Ctx.Chat.Send("/task ")
				} else {
					uiapi.MessageInfo("请先在聊天面板中输入任务描述")
				}
			}},
			{Label: "停止 Agent", OnClick: func() {
				if chatpanel.IsRunning() {
					chatpanel.StopAgent()
				} else {
					uiapi.MessageInfo("Agent 当前未运行")
				}
			}},
			{Divider: true},
			{Label: "Agent 监控", OnClick: func() { menuactions.ShowAgentMonitor() }},
			{Divider: true},
			{Label: "MCP 市场…", OnClick: func() {
				marketplacepanel.OpenDialog()
			}},
			{Label: "技能市场…", OnClick: func() {
				marketplacepanel.OpenDialog()
			}},
			{Divider: true},
			{Label: "技能管理…", OnClick: func() {
				settingspanel.OpenDialog()
			}},
			{Label: "语言模型配置…", OnClick: func() {
				settingspanel.OpenDialog()
			}},
			{Divider: true},
			{Label: "Agent 设置…", OnClick: func() {
				settingspanel.OpenDialog()
			}},
			{Divider: true},
			{Label: "项目统计…", OnClick: func() {
				statspanel.ShowStatsDialog()
			}},
		}},
		// ── 帮助 ──
		{"帮助", []component.PopupMenuItem{
			{Label: "快捷键参考", OnClick: func() {
				uiapi.MessageInfo("快捷键：Ctrl+N 新建 | Ctrl+S 保存 | Ctrl+F 查找 | Ctrl+P 命令面板 | Ctrl+B 侧栏 | Ctrl+J 终端 | Ctrl+K 专注")
			}},
			{Label: "文档", OnClick: func() {
				uiapi.MessageInfo("Pair CodeAgent AI IDE 文档请访问项目 README")
			}},
			{Label: "检查更新", OnClick: func() { uiapi.MessageInfo("当前版本：v0.1.0（GWui）") }},
			{Divider: true},
			{Label: "报告问题", OnClick: func() { uiapi.MessageInfo("请提交 Issue 到项目仓库") }},
			{Divider: true},
			{Label: "关于", OnClick: func() { uiapi.MessageInfo("Pair CodeAgent v0.1.0\n基于 GWui 的现代化 AI IDE") }},
		}},
	}
	menuMaxWidth := float32(180)
	var activePopup *component.PopupMenu
	for _, m := range menus {
		btn := component.NewButton(doc, m.label)
		btn.SetHoverStyle("background:#2a2d2e;")
		btn.SetBaseStyle("flex:0 0 auto;display:inline-flex;align-items:center;justify-content:center;height:36px;padding:0 10px;font-size:13px;color:#cccccc;background:transparent;border:none;cursor:default;user-select:none;")
		btn.Element().ClassList().Add("menu-btn")
		popup := component.NewPopupMenu(doc, btn.Element(), m.items)
		popup.SetMatchWidth(false)
		popup.SetMaxWidth(menuMaxWidth)
		popup.SetPlacement(dom.PlacementBottomStart)
		btn.OnClick(func() {
			// 菜单条行为：点击不同按钮关闭前一个，点击同一按钮切换
			if activePopup != nil && activePopup != popup {
				activePopup.Close()
			}
			popup.Toggle()
			if popup.IsOpen() {
				activePopup = popup
			} else {
				activePopup = nil
			}
		})
		menuBar.AppendChild(btn.Element())
	}
}

// startStatusBarMonitor 启动状态栏定时刷新 goroutine。
// 仅当状态实际变化时才调用 MarkDirty，避免每 500ms 无谓重布局。
func startStatusBarMonitor(doc *dom.Document) {
	dot := doc.GetElementByID("status-dot")
	agentStatus := doc.GetElementByID("status-agent")
	gitText := doc.GetElementByID("status-git")
	cursorPos := doc.GetElementByID("status-cursor")
	langEl := doc.GetElementByID("status-lang")
	indentEl := doc.GetElementByID("status-indent")
	titleCenter := doc.GetElementByID("title-center")
	go func() {
		var lastPath, lastBranch, lastCursor, lastAgent, lastLang, lastIndent, lastTitle string
		var lastRunning bool
		for {
			time.Sleep(500 * time.Millisecond)
			changed := false

			running := chatpanel.IsRunning()
			agentStr := "就绪"
			if running {
				agentStr = "Agent 运行中…"
			}
			if running != lastRunning || agentStr != lastAgent {
				lastRunning = running
				lastAgent = agentStr
				if dot != nil {
					if running {
						dot.ClassList().Add("running")
					} else {
						dot.ClassList().Remove("running")
					}
				}
				if agentStatus != nil {
					agentStatus.SetTextContent(agentStr)
				}
				changed = true
			}

			branch := gitpanel.Branch()
			if branch != lastBranch {
				lastBranch = branch
				if gitText != nil {
					gitText.SetTextContent(branch)
				}
				changed = true
			}

			ln, col := editorpanel.CursorPosition()
			cursorStr := ""
			if ln > 0 {
				cursorStr = "Ln " + itoa(ln) + ", Col " + itoa(col)
			}
			if cursorStr != lastCursor {
				lastCursor = cursorStr
				if cursorPos != nil {
					cursorPos.SetTextContent(cursorStr)
				}
				changed = true
			}

			// 语言模式 + 缩进 + 标题栏（仅文件变化时更新）
			curPath := editorpanel.ActivePath()
			if curPath != lastPath {
				lastPath = curPath
				langStr := ""
				indentStr := ""
				titleStr := "Pair CodeAgent"
				if curPath != "" {
					langStr = languageOf(curPath)
					indentStr = "空格: 4"
					titleStr = filepath.Base(curPath) + " - Pair CodeAgent"
				}
				if titleStr != lastTitle {
					lastTitle = titleStr
					if titleCenter != nil {
						titleCenter.SetTextContent(titleStr)
					}
					changed = true
				}
				if langStr != lastLang {
					lastLang = langStr
					if langEl != nil {
						langEl.SetTextContent(langStr)
					}
					changed = true
				}
				if indentStr != lastIndent {
					lastIndent = indentStr
					if indentEl != nil {
						indentEl.SetTextContent(indentStr)
					}
					changed = true
				}
			}

			if changed && theApp != nil {
				theApp.MarkDirty()
			}
		}
	}()
}

// languageOf 根据文件扩展名返回语言模式名。
func languageOf(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "Go"
	case ".js":
		return "JavaScript"
	case ".ts":
		return "TypeScript"
	case ".jsx", ".tsx":
		return "React"
	case ".py":
		return "Python"
	case ".rs":
		return "Rust"
	case ".java":
		return "Java"
	case ".c", ".h":
		return "C"
	case ".cpp", ".hpp", ".cc":
		return "C++"
	case ".cs":
		return "C#"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".swift":
		return "Swift"
	case ".kt":
		return "Kotlin"
	case ".sh", ".bash":
		return "Shell"
	case ".bat", ".cmd":
		return "Batch"
	case ".ps1":
		return "PowerShell"
	case ".html", ".htm":
		return "HTML"
	case ".css":
		return "CSS"
	case ".scss", ".sass":
		return "SCSS"
	case ".less":
		return "LESS"
	case ".json":
		return "JSON"
	case ".xml":
		return "XML"
	case ".yaml", ".yml":
		return "YAML"
	case ".toml":
		return "TOML"
	case ".md":
		return "Markdown"
	case ".sql":
		return "SQL"
	case ".lua":
		return "Lua"
	case ".vue":
		return "Vue"
	case ".svelte":
		return "Svelte"
	default:
		if ext == "" {
			name := strings.ToLower(filepath.Base(path))
			switch name {
			case "makefile":
				return "Makefile"
			case "dockerfile":
				return "Dockerfile"
			}
			return "纯文本"
		}
		return strings.ToUpper(strings.TrimPrefix(ext, "."))
	}
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

// togglePanel 切换面板可见性。
func togglePanel(panel string) {
	switch panel {
	case "left":
		shellState.LeftOpen = !shellState.LeftOpen
	case "bottom":
		shellState.BottomOpen = !shellState.BottomOpen
	}
	updatePanelVisibility()
	// 同步活动栏的 active 状态
	syncActivityBar()
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

// updateLeftView 切换左面板活动视图。
func updateLeftView() {
	if shellState == nil {
		return
	}
	showStyle := "display:flex;flex-direction:column;flex:1;min-height:0;overflow:hidden;background:#252526;"
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
	// 更新左面板标题头
	if hdr := theDoc.GetElementByID("left-panel-header"); hdr != nil {
		switch shellState.LeftView {
		case "files":
			hdr.SetTextContent("资源管理器")
		case "search":
			hdr.SetTextContent("搜索")
		case "git":
			hdr.SetTextContent("源代码管理")
		}
	}
	if theApp != nil {
		theApp.MarkDirty()
	}
}

// updatePanelVisibility 根据状态更新面板可见性。
func updatePanelVisibility() {
	if shellState.FocusMode {
		// 专注模式：仅保留聊天面板，隐藏编辑区、左面板、底栏
		shellState.LeftPanel.ClassList().Add("hidden")
		shellState.LeftDivider.ClassList().Add("hidden")
		shellState.BottomPanel.ClassList().Add("hidden")
		shellState.BottomDivider.ClassList().Add("hidden")
		shellState.RightDivider.ClassList().Add("hidden")
		if shellState.CenterPanel != nil {
			shellState.CenterPanel.ClassList().Add("hidden")
		}
		// 右面板占满剩余宽度（活动栏除外）
		shellState.RightPanel.SetAttribute("style", "flex:1;display:flex;flex-direction:column;")
		return
	}
	// 非专注模式：恢复面板宽度
	if shellState.CenterPanel != nil {
		shellState.CenterPanel.ClassList().Remove("hidden")
	}
	// 右面板始终可见
	shellState.RightPanel.SetAttribute("style", "width:"+ftoa(shellState.RightW)+"%;min-width:620px;max-width:70%;flex-shrink:0;display:flex;flex-direction:column;")
	shellState.RightPanel.ClassList().Remove("hidden")
	shellState.RightDivider.ClassList().Remove("hidden")

	setVisible := func(el *dom.Element, show bool) {
		if show {
			el.ClassList().Remove("hidden")
		} else {
			el.ClassList().Add("hidden")
		}
	}
	setVisible(shellState.LeftPanel, shellState.LeftOpen)
	setVisible(shellState.LeftDivider, shellState.LeftOpen)
	setVisible(shellState.BottomPanel, shellState.BottomOpen)
	setVisible(shellState.BottomDivider, shellState.BottomOpen)
}

// setupDividerDrag 设置分隔条拖动。
// 浏览器标准模式：MouseDown 挂分隔条（启动拖拽），MouseMove/MouseUp 挂 doc.Body()（确保鼠标拖出 6px 后仍接收）。
func setupDividerDrag(doc *dom.Document, divider *dom.Element, side string) {
	if divider == nil || theApp == nil {
		return
	}
	docBody := doc.Body()
	var dragging bool
	var startX, startY float32
	var startW, startH float32
	var parentW float32 // 父容器宽度（右面板百分比计算用）

	theApp.AddEventListener(divider, event.MouseDown, func(e event.Event) bool {
		me := e.(*event.MouseEvent)
		dragging = true
		startX = me.X
		startY = me.Y
		switch side {
		case "left":
			startW = shellState.LeftW
		case "right":
			startW = shellState.RightW // 百分比
			vpW, _ := theApp.Size()
			parentW = float32(vpW) - 48 // 减去活动栏宽度
		case "bottom":
			startH = shellState.BottomH
		}
		divider.ClassList().Add("dragging")
		return true
	})

	theApp.AddEventListener(docBody, event.MouseMove, func(e event.Event) bool {
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
			// startW 是百分比，dx 是像素 → 转百分比变化
			dx := me.X - startX
			deltaPct := float32(0)
			if parentW > 0 {
				deltaPct = dx / parentW * 100
			}
			newPct := startW - deltaPct // 右拖→窄，左拖→宽
			if newPct < 20 {
				newPct = 20
			}
			if newPct > 70 {
				newPct = 70
			}
			shellState.RightW = newPct
			shellState.RightPanel.SetAttribute("style", "width:"+ftoa(newPct)+"%;min-width:620px;max-width:70%;flex-shrink:0;display:flex;flex-direction:column;")
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

	theApp.AddEventListener(docBody, event.MouseUp, func(e event.Event) bool {
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
		m.SetMaxWidth(480)
		m.SetMaxHeight(360)
		footer := m.Footer()
		confirmBtn := component.NewButton(theDoc, "确定")
		confirmBtn.OnClick(func() { m.Hide(); onConfirm() })
		footer.AppendChild(confirmBtn.Element())
		cancelBtn := component.NewButton(theDoc, "取消")
		cancelBtn.Element().ClassList().Add("btn-secondary")
		cancelBtn.OnClick(func() { m.Hide() })
		footer.AppendChild(cancelBtn.Element())
		m.Show()
	}
	// 自定义对话框
	uiapi.ShowDialogFunc = func(title string, width float32, content, footer interface{}) int {
		m := component.NewModal(theDoc)
		m.SetTitle(title)
		// 设尺寸上限防止内容撑满屏幕；width 参数生效，默认 600
		if width > 0 {
			m.SetMaxWidth(width)
		} else {
			m.SetMaxWidth(600)
		}
		m.SetMaxHeight(560)
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
		core.Settings.WorkspaceFolders = append([]string{}, core.Folders...)
		core.Settings.LastProject = core.Root()
		core.Loaded = true
		core.Save()
		if len(core.Folders) > 0 {
			shellState.LeftOpen = true
			shellState.BottomOpen = true
		} else {
			shellState.LeftOpen = false
			shellState.BottomOpen = false
		}
		invalidateCmdPaletteFiles()
		updatePanelVisibility()
		syncActivityBar()
		if theApp != nil {
			theApp.MarkDirty()
		}
	}
	core.OnCloseProject = func() {
		shellState.LeftOpen = false
		shellState.BottomOpen = false
		invalidateCmdPaletteFiles()
		updatePanelVisibility()
		syncActivityBar()
		if theApp != nil {
			theApp.MarkDirty()
		}
	}
	core.OnClearWorkspace = func() {
		shellState.LeftOpen = false
		shellState.BottomOpen = false
		invalidateCmdPaletteFiles()
		updatePanelVisibility()
		syncActivityBar()
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
		m.SetMaxWidth(480)
		m.SetMaxHeight(360)
		bodyEl := m.Content()
		bodyEl.SetTextContent("")
		inp := component.NewInput(theDoc, initial)
		bodyEl.AppendChild(inp.Element())
		footer := m.Footer()
		okBtn := component.NewButton(theDoc, "确定")
		okBtn.OnClick(func() { m.Hide(); onOk(inp.Value()) })
		footer.AppendChild(okBtn.Element())
		cancelBtn := component.NewButton(theDoc, "取消")
		cancelBtn.Element().ClassList().Add("btn-secondary")
		cancelBtn.OnClick(func() { m.Hide() })
		footer.AppendChild(cancelBtn.Element())
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
	chatpanel.OnChatContextMenu = ctxmenupanel.ChatMessageContextMenu
	chatpanel.OnChatInputContextMenu = ctxmenupanel.ChatInputContextMenu
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

	// Bridge 帧泵注入
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
		shellState.LeftView = "search"
		if !shellState.LeftOpen {
			togglePanel("left")
		}
	})
	theApp.RegisterShortcut("Ctrl+P", func() {
		openCommandPalette()
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
	n := int(f)
	return itoa(n)
}

// 确保未使用的 import 不报错
var _ = state.DefaultPanels
var _ = codetypes.CodeLoc{}
