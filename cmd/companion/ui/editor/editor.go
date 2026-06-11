// 编辑器面板 —— 中列编辑区：多标签页 + 语法高亮 + 编辑 + 保存（Ctrl+S）。
// 点文件树文件 → 新标签（或切到已打开标签）；改动标 dirty(●)；Ctrl+S 写盘。
// 复用 goui 对标 Monaco 的 CodeEditor，切标签靠 ReloadToken 受控重载。详见 AGENTS.md。
//
//go:build windows

package editorpanel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/gou-ide/cmd/companion/core"
	"github.com/user/gou-ide/cmd/companion/langsrv"
	"github.com/user/gou-ide/cmd/companion/ui"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// Editor 编辑器状态（包级单例，StatefulWidget State，跨 relayout 存活）。
var Editor = &editorState{}

type editorTab struct {
	path    string
	content string
	lang    string // 扩展名去点（交 ceLangFor / HasProvider）
	dirty   bool   // 有未保存改动
}

type editorState struct {
	widget.BaseState
	tabs     []*editorTab
	active   int
	reload   int // 受控重载令牌：切换/打开/关闭文件时自增 → CodeEditor 重置为新标签内容
	reveal   int // 跳转令牌：openAt 自增 → CodeEditor 跳到 gotoLine
	gotoLine int // 待跳转行(1 基)，供 openAt 跳转用

	split    bool // 分栏对比：左 active + 右 rightTab 两栏并排
	rightTab int  // 右栏标签索引
	reloadR  int  // 右栏独立重载令牌

	cursorLine int // 当前活动标签光标行（1 基），供状态栏 Ln 显示
	cursorCol  int // 当前活动标签光标列（1 基），供状态栏 Col 显示
}

// EditorPanel 编辑器面板组件。
type EditorPanel struct{ widget.StatefulWidget }

func (e *EditorPanel) CreateState() widget.State { return Editor }

func (e *editorState) ActiveTab() *editorTab {
	if e.active >= 0 && e.active < len(e.tabs) {
		return e.tabs[e.active]
	}
	return nil
}

// open 打开文件：已打开→切到该标签（无未保存改动时刷新为磁盘最新内容）；否则新建标签读内容。
func (e *editorState) Open(path string) {
	for i, t := range e.tabs {
		if t.path == path {
			if !t.dirty {
				loadTabContent(t) // 重新打开已开文件→刷新为磁盘当前内容（无未保存改动时）
			}
			e.switchTo(i)
			return
		}
	}
	t := &editorTab{path: path}
	loadTabContent(t)
	langsrv.MaybeOfferInstall(t.lang) // 该语言有服务器但没装→弹一次「自动安装」确认
	e.tabs = append(e.tabs, t)
	e.active = len(e.tabs) - 1
	e.reload++
	e.persistSession()
	e.SetState()
}

// openAt 打开文件并跳转到第 line 行（1 基）。供搜索/Git 点击结果定位用。
func (e *editorState) OpenAt(path string, line int) {
	e.gotoLine = line
	e.reveal++
	e.Open(path) // open 内部 reload++ + SetState，本次渲染即带上 gotoLine/reveal
}

// reloadIfOpen 若文件已在某标签打开且无未保存改动，则从磁盘重载内容（Agent 改文件后保持编辑器同步）。
// 有未保存改动(dirty)时不覆盖用户编辑。返回是否实际重载。
func (e *editorState) ReloadIfOpen(path string) bool {
	for _, t := range e.tabs {
		if t.path == path {
			if t.dirty {
				return false // 有未保存改动，不覆盖
			}
			loadTabContent(t)
			e.reload++
			e.SetState()
			return true
		}
	}
	return false
}

func (e *editorState) switchTo(i int) {
	if i < 0 || i >= len(e.tabs) {
		return
	}
	e.active = i
	e.reload++
	e.persistSession()
	e.SetState()
}

func (e *editorState) Close(i int) {
	if i < 0 || i >= len(e.tabs) {
		return
	}
	if t := e.tabs[i]; t.dirty {
		e.confirmCloseSingle(i, t)
		return
	}
	e.doClose(i)
}

func (e *editorState) doClose(i int) {
	e.tabs = append(e.tabs[:i], e.tabs[i+1:]...)
	if e.active >= len(e.tabs) {
		e.active = len(e.tabs) - 1
	}
	e.reload++
	e.persistSession()
	e.SetState()
}

// closeOthers 关闭除第 i 个外的所有标签（右键菜单）。
func (e *editorState) CloseOthers(i int) {
	if i < 0 || i >= len(e.tabs) {
		return
	}
	dirtyCount := 0
	for j, t := range e.tabs {
		if j != i && t.dirty {
			dirtyCount++
		}
	}
	if dirtyCount > 0 {
		e.confirmCloseOthers(i, dirtyCount)
		return
	}
	e.tabs = []*editorTab{e.tabs[i]}
	e.active = 0
	e.reload++
	e.persistSession()
	e.SetState()
}

// closeAll 关闭所有标签。
func (e *editorState) CloseAll() {
	if e.HasDirty() {
		e.confirmCloseAll()
		return
	}
	e.doCloseAll()
}

func (e *editorState) doCloseAll() {
	e.tabs = nil
	e.active = 0
	e.reload++
	e.persistSession()
	e.SetState()
}

// HasDirty 是否有未保存的标签。
func (e *editorState) HasDirty() bool {
	for _, t := range e.tabs {
		if t.dirty {
			return true
		}
	}
	return false
}

// ConfirmCloseAll 关闭所有标签（带未保存提示），用户确认后回调 onConfirmed。
func (e *editorState) ConfirmCloseAll(onConfirmed func()) {
	if !e.HasDirty() {
		e.doCloseAll()
		if onConfirmed != nil {
			onConfirmed()
		}
		return
	}
	dirtyCount := 0
	var names []string
	for _, t := range e.tabs {
		if t.dirty {
			dirtyCount++
			if len(names) < 3 {
				names = append(names, filepath.Base(t.path))
			}
		}
	}
	label := fmt.Sprintf("有 %d 个文件未保存", dirtyCount)
	detail := strings.Join(names, "、")
	if dirtyCount > len(names) {
		detail += fmt.Sprintf(" 等 %d 个", dirtyCount)
	}
	body := widget.Div(widget.Style{Padding: types.EdgeInsets(4)},
		ui.TextLine(fmt.Sprintf("%s：%s\n\n是否保存后再关闭？", label, detail), *ui.Fg, 12),
	)
	var id int
	dlg := widget.NewDialog("未保存的更改", body).WithWidth(400).WithFooter(
		ui.Btn("不保存", func() {
			widget.HideOverlay(id)
			e.doCloseAll()
			if onConfirmed != nil {
				onConfirmed()
			}
		}),
		ui.Btn("取消", func() {
			widget.HideOverlay(id)
		}),
		ui.PrimaryBtn("全部保存", func() {
			widget.HideOverlay(id)
			e.saveAllDirty()
			e.doCloseAll()
			if onConfirmed != nil {
				onConfirmed()
			}
		}),
	)
	id = widget.ShowDialog(dlg)
}

// ─── 私有的未保存确认 ───

// confirmCloseSingle 关闭单个未保存标签的确认对话框。
func (e *editorState) confirmCloseSingle(i int, t *editorTab) {
	name := filepath.Base(t.path)
	body := widget.Div(widget.Style{Padding: types.EdgeInsets(4)},
		ui.TextLine(fmt.Sprintf("「%s」有未保存的更改。\n是否保存后再关闭？", name), *ui.Fg, 12),
	)
	var id int
	dlg := widget.NewDialog("未保存的更改", body).WithWidth(380).WithFooter(
		ui.Btn("不保存", func() {
			widget.HideOverlay(id)
			e.doClose(i)
		}),
		ui.Btn("取消", func() {
			widget.HideOverlay(id)
		}),
		ui.PrimaryBtn("保存", func() {
			widget.HideOverlay(id)
			e.saveTab(t)
			e.doClose(i)
		}),
	)
	id = widget.ShowDialog(dlg)
}

// confirmCloseOthers 关闭其他标签（含未保存）的确认对话框。
func (e *editorState) confirmCloseOthers(keep int, dirtyCount int) {
	label := fmt.Sprintf("有 %d 个其他标签未保存", dirtyCount)
	body := widget.Div(widget.Style{Padding: types.EdgeInsets(4)},
		ui.TextLine(fmt.Sprintf("%s。\n是否保存后再关闭？", label), *ui.Fg, 12),
	)
	var id int
	dlg := widget.NewDialog("未保存的更改", body).WithWidth(380).WithFooter(
		ui.Btn("不保存", func() {
			widget.HideOverlay(id)
			e.tabs = []*editorTab{e.tabs[keep]}
			e.active = 0
			e.reload++
			e.persistSession()
			e.SetState()
		}),
		ui.Btn("取消", func() {
			widget.HideOverlay(id)
		}),
		ui.PrimaryBtn("全部保存", func() {
			widget.HideOverlay(id)
			for j, t := range e.tabs {
				if j != keep && t.dirty {
					e.saveTab(t)
				}
			}
			e.tabs = []*editorTab{e.tabs[keep]}
			e.active = 0
			e.reload++
			e.persistSession()
			e.SetState()
		}),
	)
	id = widget.ShowDialog(dlg)
}

// confirmCloseAll （无回调版）关闭所有标签的确认对话框——用于标签栏 × 等不需要后续处理动作的场景。
func (e *editorState) confirmCloseAll() {
	e.ConfirmCloseAll(nil)
}

// saveTab 保存单个标签到磁盘。
func (e *editorState) saveTab(t *editorTab) {
	if t == nil || !t.dirty {
		return
	}
	if err := os.WriteFile(t.path, []byte(t.content), 0o644); err == nil {
		t.dirty = false
	}
}

// saveAllDirty 保存所有未保存标签到磁盘。
func (e *editorState) saveAllDirty() {
	for _, t := range e.tabs {
		e.saveTab(t)
	}
}

// tabAt 取第 i 个标签（越界 nil）。
func (e *editorState) TabAt(i int) *editorTab {
	if i < 0 || i >= len(e.tabs) {
		return nil
	}
	return e.tabs[i]
}

// onEdit CodeEditor 内容变化回调：同步到当前标签 + 首次改动标 dirty。
func (e *editorState) onEdit(content string) {
	t := e.ActiveTab()
	if t == nil {
		return
	}
	t.content = content
	if !t.dirty {
		t.dirty = true
		e.SetState() // 仅首次编辑触发一次 relayout 以显示 ● dirty 标记
	}
}

// onCursorUpdate 光标位置变化回调：CodeEditor 每帧/每次移动光标时触发。
func (e *editorState) onCursorUpdate(line, col int) {
	if e.cursorLine != line+1 || e.cursorCol != col+1 {
		e.cursorLine = line + 1 // 0 基→1 基
		e.cursorCol = col + 1
		if OnCursorMoved != nil {
			OnCursorMoved()
		}
	}
}

// CursorPosition 返回当前光标行/列（1 基），供状态栏显示。
func (e *editorState) CursorPosition() (line, col int) {
	return e.cursorLine, e.cursorCol
}

// save 保存当前标签到磁盘（Ctrl+S）。
func (e *editorState) Save() {
	t := e.ActiveTab()
	if t == nil || !t.dirty {
		return
	}
	if err := os.WriteFile(t.path, []byte(t.content), 0o644); err == nil {
		t.dirty = false
		e.SetState()
	}
}

func loadTabContent(t *editorTab) {
	if isBinaryExt(filepath.Ext(t.path)) {
		t.content, t.lang = "〔二进制文件，不预览〕", ""
		return
	}
	if fi, err := os.Stat(t.path); err == nil && fi.Size() > 2*1024*1024 {
		t.content, t.lang = "〔文件过大（>2MB），不预览〕", ""
		return
	}
	data, err := os.ReadFile(t.path)
	if err != nil {
		t.content, t.lang = "// 读取失败: "+err.Error(), ""
		return
	}
	t.content = string(data)
	t.lang = strings.TrimPrefix(strings.ToLower(filepath.Ext(t.path)), ".")
}

func isBinaryExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".exe", ".dll", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".bmp", ".webp", ".pdf",
		".zip", ".gz", ".tar", ".7z", ".rar", ".ttf", ".otf", ".woff", ".woff2",
		".so", ".a", ".o", ".bin", ".dat", ".db", ".class", ".wasm", ".mp3", ".mp4", ".wav":
		return true
	}
	return false
}

func (e *editorState) Build(ctx widget.BuildContext) widget.Widget {
	if len(e.tabs) == 0 {
		// 欢迎页（空状态）由 Go 代码直接构建，不依赖 JSON 配置 / embed 硬编码。
		// 直接读取 ui 包主题指针，跟随主题切换。
		return buildWelcome()
	}
	editorBg := widget.Style{BackgroundColor: ui.ShellEditor, FlexDirection: "column", AlignItems: "stretch"}
	if e.split && len(e.tabs) >= 2 { // 分栏对比：左 active + 右 rightTab，各自独立编辑
		if e.rightTab < 0 || e.rightTab >= len(e.tabs) || e.rightTab == e.active {
			e.rightTab = pickOtherTab(e.active, len(e.tabs))
		}
		li, ri := e.active, e.rightTab
		left := e.tabEditorWidget(e.tabs[li], e.reload, e.reveal, e.gotoLine, func(c string) { e.onEditTab(li, c) })
		right := e.tabEditorWidget(e.tabs[ri], e.reloadR, 0, 0, func(c string) { e.onEditTab(ri, c) })
		cols := ui.FlexRow(
			ui.Expand(widget.VBox(e.colHeader(li), ui.Expand(left))),
			ui.VLine(),
			ui.Expand(widget.VBox(e.colHeader(ri), ui.Expand(right))),
		)
		return widget.Div(editorBg, e.tabBar(), ui.Expand(cols))
	}
	content := e.tabEditorWidget(e.ActiveTab(), e.reload, e.reveal, e.gotoLine, e.onEdit)
	return widget.Div(editorBg, e.tabBar(), ui.Expand(content))
}

// tabEditorWidget 构建某标签的编辑器视图（CodeWorkbench 或 CodeEditor）+ 右键菜单包裹。
// reload/reveal/gotoLine 为该栏独立令牌；onEdit 路由到对应标签（分栏时左右各一份）。
func (e *editorState) tabEditorWidget(t *editorTab, reloadTok, revealTok, gotoLineVal int, onEdit func(string)) widget.Widget {
	if t == nil {
		return widget.Div(widget.Style{BackgroundColor: ui.ShellEditor})
	}
	fam := core.FirstFontFamily(core.Settings.FontFamily) // 外观设置：等宽字体族（空=组件默认 Consolas）
	srv, srvArgs, langID, hasLSP := langsrv.For(t.lang)   // 该语言有可用语言服务器吗（gopls/tsserver/pyright/…）
	hasLSP = hasLSP && t.path != ""                       // 需有真实文件路径才能喂 LSP
	var editor widget.Widget
	// 有 LanguageProvider 的语言用 CodeWorkbench（结构化/代码视图可切换）；无 provider 的用 CodeEditor。
	if widget.HasProvider(t.lang) {
		wb := widget.NewCodeWorkbench(t.content).WithSize(9000, 9000).WithLang(t.lang).WithFontFamily(fam).WithFontSize(float64(core.Settings.EditorFontSize)).WithFontStyle(core.Settings.EditorFontBold, core.Settings.EditorFontItalic, core.Settings.EditorFontUnderline)
		wb.ReloadToken = reloadTok
		wb.OnChange = onEdit
		if hasLSP { // 代码视图接语言服务器（补全/诊断/转到定义/查找引用/大纲/悬停）
			wb.LSPServer, wb.LSPArgs, wb.LSPWorkspace, wb.LSPFile, wb.LSPLangID = srv, srvArgs, pathToURI(core.Root()), pathToURI(t.path), langID
			wb.OnGoToDefinition, wb.OnReferences, wb.OnDocumentSymbols = editorGoToDef, OnReferences, OnSymbols
		}
		editor = wb
	} else {
		ed := widget.NewCodeEditor(t.lang, t.content).WithSize(9000, 9000).WithFontFamily(fam)
		ed.ReloadToken = reloadTok
		ed.RevealLine = gotoLineVal
		ed.RevealToken = revealTok
		ed.OnChange = onEdit
		ed.OnCursorMove = e.onCursorUpdate
		ed.FontSize = float64(core.Settings.EditorFontSize) // 外观设置：字号（0=默认 14）
		ed.FontBold, ed.FontItalic, ed.FontUnderline = core.Settings.EditorFontBold, core.Settings.EditorFontItalic, core.Settings.EditorFontUnderline
		ed.Minimap = !core.Settings.HideMinimap // 外观设置：minimap（默认开）
		if hasLSP {                             // 非 Go 语言也接 LSP（tsserver/pyright/clangd/rust-analyzer…）
			ed.LSPServer, ed.LSPArgs, ed.LSPWorkspace, ed.LSPFile, ed.LSPLangID = srv, srvArgs, pathToURI(core.Root()), pathToURI(t.path), langID
			ed.OnGoToDefinition, ed.OnReferences, ed.OnDocumentSymbols = editorGoToDef, OnReferences, OnSymbols
		}
		editor = ed
	}
	// 包一层 ContextArea：右键弹 companion 自定义菜单（撤销/剪贴/全选 + 结构化语言视图切换）。
	return &widget.ContextArea{
		SingleChildWidget: widget.SingleChildWidget{Child: editor},
		OnContextMenu: func(x, y float64) {
			if OnContentMenu != nil {
				OnContentMenu(x, y)
			}
		},
	}
}

// colHeader 分栏每栏顶部的文件名条。
func (e *editorState) colHeader(idx int) widget.Widget {
	name := "—"
	if idx >= 0 && idx < len(e.tabs) {
		name = filepath.Base(e.tabs[idx].path)
		if e.tabs[idx].dirty {
			name = "● " + name
		}
	}
	return widget.Div(
		widget.Style{Height: 24, BackgroundColor: ui.ShellSide, FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(10, 0, 10, 0)},
		ui.TextC(name, *ui.ShellTextDim, 11),
	)
}

// onEditTab 改某标签内容（分栏时按栏路由）。
func (e *editorState) onEditTab(idx int, content string) {
	if idx < 0 || idx >= len(e.tabs) {
		return
	}
	t := e.tabs[idx]
	t.content = content
	if !t.dirty {
		t.dirty = true
		e.SetState()
	}
}

// toggleSplit 切换分栏对比视图（右栏取另一个标签）。
func (e *editorState) toggleSplit() {
	e.split = !e.split
	if e.split {
		e.rightTab = pickOtherTab(e.active, len(e.tabs))
		e.reloadR++
	}
	e.SetState()
}

func pickOtherTab(active, n int) int {
	if n < 2 {
		return active
	}
	if active > 0 {
		return active - 1
	}
	return 1
}

// pathToURI 本地路径 → file:// URI（喂 gopls）。
func pathToURI(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	abs = filepath.ToSlash(abs)
	if !strings.HasPrefix(abs, "/") {
		abs = "/" + abs // Windows C:/... → /C:/...
	}
	return "file://" + abs
}

// editorGoToDef 转到定义回调：打开目标文件并跳到行（目标可能在工作区外，如 stdlib/依赖）。
func editorGoToDef(file string, line, col int) {
	if line < 1 {
		line = 1
	}
	Editor.OpenAt(file, line)
}

// tabBar 编辑区顶部标签栏：每个打开的文件一个标签（类型图标+名+dirty●/关闭×），点切换、×关闭。
func (e *editorState) tabBar() widget.Widget {
	items := make([]widget.Widget, 0, len(e.tabs)*2+2)
	for i, t := range e.tabs {
		if i > 0 { // 相邻标签间 1px 竖分隔条（stretch 撑满高）
			items = append(items, widget.Div(widget.Style{Width: 1, BackgroundColor: ui.ShellBorder}))
		}
		items = append(items, e.tabItem(i, t))
	}
	items = append(items, ui.Expand(widget.Div(widget.Style{}))) // 弹性占位把分栏按钮推到最右
	if len(e.tabs) >= 2 {                                        // 至少 2 个标签才提供分栏对比
		items = append(items, e.splitToggle())
	}
	return widget.Div(
		widget.Style{Height: 34, BackgroundColor: ui.ShellSide, FlexDirection: "row", AlignItems: "stretch"},
		items,
	)
}

// splitToggle 标签栏右侧「分栏/单栏」对比视图开关。
func (e *editorState) splitToggle() widget.Widget {
	txt, col := "分栏", *ui.FgSubtle
	if e.split {
		txt, col = "单栏", *ui.Fg
	}
	return &widget.Button{
		Text: txt, FontSize: 11, TextColor: col, Color: *ui.ShellSide, HoverColor: *ui.BgMuted,
		Padding: types.EdgeInsetsLTRB(10, 0, 10, 0),
		OnClick: e.toggleSplit,
	}
}

func (e *editorState) tabItem(i int, t *editorTab) widget.Widget {
	active := i == e.active
	bg := *ui.ShellSide
	txtCol := *ui.ShellTextDim
	if active {
		bg = *ui.ShellEditor
		txtCol = *ui.ShellText
	}
	icon, iconCol := ui.FileIcon(filepath.Base(t.path), false, false)
	row := []widget.Widget{
		widget.Lucide(icon, widget.IconSize(13), widget.IconColor(iconCol)),
		widget.Div(widget.Style{Width: 6}),
		ui.TextC(filepath.Base(t.path), txtCol, 12),
		widget.Div(widget.Style{Width: 8}),
	}
	if t.dirty {
		row = append(row, widget.Div(widget.Style{Width: 8, Height: 8, BackgroundColor: ui.ShellText, BorderRadius: 4}))
	} else {
		row = append(row, e.closeBtn(i))
	}
	tab := &widget.Clickable{
		// 显式 Height 让标签稳定填满标签栏高度并使内容垂直居中（不依赖 stretch 链路，单/多文件一致）。
		// 不再用整框边框（之前因标签按内容高、边框只有文字大小）；相邻标签靠 tabBar 里的分隔条区分。
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 34, FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(12, 0, 8, 0)},
			row,
		)},
		OnClick:    func() { e.switchTo(i) },
		Color:      bg,
		HoverColor: *ui.FtHover,
	}
	return &widget.ContextArea{ // 右键：标签菜单（关闭/关闭其他/复制路径/添加到对话）
		SingleChildWidget: widget.SingleChildWidget{Child: tab},
		OnContextMenu: func(x, y float64) {
			if OnTabMenu != nil {
				OnTabMenu(x, y, i)
			}
		},
	}
}

// closeBtn 标签关闭×：StopPropagation 使点×只关闭、不触发外层标签切换。
func (e *editorState) closeBtn(i int) widget.Widget {
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Padding: types.EdgeInsets(3)},
			widget.Lucide("x", widget.IconSize(12), widget.IconColor(*ui.ShellTextDim)),
		)},
		OnClick:         func() { e.Close(i) },
		StopPropagation: true,
		HoverColor:      *ui.FtHover,
	}
}

// 中列编辑区入口 editorArea() 已移到 api.go 的 Area()。
