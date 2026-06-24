// 编辑器面板 —— 中列编辑区：多标签页 + 编辑 + 保存（Ctrl+S）。
// 点文件树文件 → 新标签（或切到已打开标签）；改动标 dirty(●)；Ctrl+S 写盘。
//
// GWui 版：使用 dom.Document 创建动态 UI，不再依赖 goui。
//
//go:build windows

package editorpanel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"

	"github.com/hoonfeng/paircode/cmd/companion/ui"
)

// ─── 颜色常量 ────────────────────────────────────────────────
var (
	colText     = ui.Text        // "#cccccc"
	colTextDim  = ui.TextDim     // "#8c8c8c"
	colSide     = ui.SideBg      // "#252526"
	colEditor   = ui.EditorBg    // "#1e1e1e"
	colBorder   = ui.Border      // "#2d2d2d"
	colAccent   = ui.Accent      // "#0e639c"
	colHover    = ui.HoverBg     // "#2a2d2e"
	colSelected = ui.ActiveBg    // "#094771"
	colWhite    = "#ffffff"
	colWarning  = ui.Warning     // "#dcdcaa"
	colStatus   = ui.StatusBarBg // "#2d2d30"
)

// Editor 编辑器状态（包级单例）。
var Editor = &editorState{}

type editorTab struct {
	path    string
	content string
	lang    string
	dirty   bool
}

type editorState struct {
	doc     *dom.Document
	rootEl  *dom.Element
	tabBar  *dom.Element
	contentArea *dom.Element

	tabs     []*editorTab
	active   int
	split    bool
	rightTab int

	cursorLine int
	cursorCol  int
	gotoLine   int
}

// New 创建编辑器面板（由 main.go 调用）。
func New(doc *dom.Document) *editorState {
	Editor.doc = doc
	Editor.rootEl = doc.CreateElement("div")
	Editor.rootEl.SetAttribute("style", "display:flex;flex-direction:column;flex:1;overflow:hidden;")

	// 标题栏
	title := doc.CreateElement("div")
	title.ClassList().Add("panel-header")
	title.SetTextContent("EDITOR")
	Editor.rootEl.AppendChild(title)

	// 欢迎页（默认）
	Editor.renderWelcome()
	return Editor
}

func (e *editorState) Element() *dom.Element { return e.rootEl }

func (e *editorState) Refresh() {
	e.renderAll()
}

// RestoreSession 恢复编辑器会话（main.go 启动时调用）。
func RestoreSession() {
	// 会话恢复暂不实现
}

// CursorPosition 返回当前光标行/列（1 基），供状态栏显示。
func CursorPosition() (line, col int) {
	return Editor.cursorLine, Editor.cursorCol
}

// ─── 公开方法（外部通过 ui.Ctx.Editor 调用）────────────────

func (e *editorState) Open(path string) {
	for i, t := range e.tabs {
		if t.path == path {
			if !t.dirty {
				loadTabContent(t)
			}
			e.switchTo(i)
			return
		}
	}
	t := &editorTab{path: path}
	loadTabContent(t)
	e.tabs = append(e.tabs, t)
	e.active = len(e.tabs) - 1
	e.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (e *editorState) OpenAt(path string, line int) {
	e.gotoLine = line
	e.Open(path)
}

func (e *editorState) Save() {
	t := e.ActiveTab()
	if t == nil || !t.dirty {
		return
	}
	if err := os.WriteFile(t.path, []byte(t.content), 0o644); err == nil {
		t.dirty = false
		e.renderAll()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
	}
}

func (e *editorState) CloseTab(idx int) {
	if idx < 0 || idx >= len(e.tabs) {
		return
	}
	if t := e.tabs[idx]; t.dirty {
		e.confirmCloseSingle(idx, t)
		return
	}
	e.doClose(idx)
}

func (e *editorState) CloseOtherTabs(idx int) {
	if idx < 0 || idx >= len(e.tabs) {
		return
	}
	dirtyCount := 0
	for j, t := range e.tabs {
		if j != idx && t.dirty {
			dirtyCount++
		}
	}
	if dirtyCount > 0 {
		// 简化：先保存所有再关
		for j, t := range e.tabs {
			if j != idx && t.dirty {
				e.saveTab(t)
			}
		}
	}
	e.tabs = []*editorTab{e.tabs[idx]}
	e.active = 0
	e.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (e *editorState) CloseAllTabs() {
	if e.HasDirty() {
		for _, t := range e.tabs {
			if t.dirty {
				e.saveTab(t)
			}
		}
	}
	e.doCloseAll()
}

func (e *editorState) ReloadIfOpen(path string) bool {
	for _, t := range e.tabs {
		if t.path == path {
			if t.dirty {
				return false
			}
			loadTabContent(t)
			e.renderAll()
			if ui.Ctx.App != nil {
				ui.Ctx.App.MarkDirty()
			}
			return true
		}
	}
	return false
}

// ActiveTab 返回当前活动标签。
func (e *editorState) ActiveTab() *editorTab {
	if e.active >= 0 && e.active < len(e.tabs) {
		return e.tabs[e.active]
	}
	return nil
}

// TabAt 返回第 i 个标签。
func (e *editorState) TabAt(i int) *editorTab {
	if i < 0 || i >= len(e.tabs) {
		return nil
	}
	return e.tabs[i]
}

func (e *editorState) HasDirty() bool {
	for _, t := range e.tabs {
		if t.dirty {
			return true
		}
	}
	return false
}

// ─── 内部渲染 ────────────────────────────────────────────────

func (e *editorState) renderAll() {
	e.rootEl.ClearChildren()

	if len(e.tabs) == 0 {
		e.renderWelcome()
		return
	}

	// 标签栏
	e.tabBar = e.doc.CreateElement("div")
	e.tabBar.SetAttribute("style", "display:flex;flex-direction:row;align-items:stretch;background:"+colSide+";height:34px;flex-shrink:0;overflow:hidden;")
	e.buildTabBar()
	e.rootEl.AppendChild(e.tabBar)

	// 编辑区
	e.contentArea = e.doc.CreateElement("div")
	e.contentArea.SetAttribute("style", "flex:1;display:flex;flex-direction:column;overflow:hidden;")

	if e.split && len(e.tabs) >= 2 {
		if e.rightTab < 0 || e.rightTab >= len(e.tabs) || e.rightTab == e.active {
			e.rightTab = pickOtherTab(e.active, len(e.tabs))
		}
		splitContainer := e.doc.CreateElement("div")
		splitContainer.SetAttribute("style", "display:flex;flex-direction:row;flex:1;overflow:hidden;")

		leftCol := e.buildEditorColumn(e.active)
		splitContainer.AppendChild(leftCol)

		divider := e.doc.CreateElement("div")
		divider.SetAttribute("style", "width:1px;background:"+colBorder+";flex-shrink:0;")
		splitContainer.AppendChild(divider)

		rightCol := e.buildEditorColumn(e.rightTab)
		splitContainer.AppendChild(rightCol)

		e.contentArea.AppendChild(splitContainer)
	} else {
		editorEl := e.buildEditor(e.ActiveTab(), e.active)
		if editorEl != nil {
			e.contentArea.AppendChild(editorEl)
		}
	}
	e.rootEl.AppendChild(e.contentArea)
}

func (e *editorState) buildTabBar() {
	e.tabBar.ClearChildren()

	for i, t := range e.tabs {
		if i > 0 {
			sep := e.doc.CreateElement("div")
			sep.SetAttribute("style", "width:1px;background:"+colBorder+";flex-shrink:0;")
			e.tabBar.AppendChild(sep)
		}
		e.tabBar.AppendChild(e.tabItem(i, t))
	}

	// 弹性占位
	filler := e.doc.CreateElement("div")
	filler.SetAttribute("style", "flex:1;")
	e.tabBar.AppendChild(filler)

	if len(e.tabs) >= 2 {
		e.tabBar.AppendChild(e.splitToggle())
	}
}

func (e *editorState) tabItem(i int, t *editorTab) *dom.Element {
	active := i == e.active
	bg := colSide
	txtCol := colTextDim
	if active {
		bg = colEditor
		txtCol = colText
	}

	tab := e.doc.CreateElement("div")
	tab.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;height:34px;padding:0 8px 0 12px;cursor:pointer;background:"+bg+";flex-shrink:0;")
	tab.SetAttribute("hover-style", "background:"+colHover+";")

	// 图标
	icon, iconCol := fileIcon(t.path)
	ic := e.doc.CreateElement("span")
	ic.SetAttribute("data-icon", icon)
	ic.SetAttribute("style", fmt.Sprintf("width:13px;height:13px;color:%s;flex-shrink:0;", iconCol))
	tab.AppendChild(ic)

	tab.AppendChild(spacer(e.doc, 6))

	// 文件名
	nameEl := e.doc.CreateElement("div")
	nameEl.SetAttribute("style", "color:"+txtCol+";font-size:12px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:120px;")
	nameEl.SetTextContent(filepath.Base(t.path))
	tab.AppendChild(nameEl)

	tab.AppendChild(spacer(e.doc, 8))

	// dirty 点 / 关闭按钮
	if t.dirty {
		dot := e.doc.CreateElement("div")
		dot.SetAttribute("style", "width:8px;height:8px;border-radius:4px;background:"+colText+";flex-shrink:0;")
		tab.AppendChild(dot)
	} else {
		closeBtn := e.doc.CreateElement("div")
		closeBtn.SetAttribute("style", "display:flex;align-items:center;justify-content:center;width:18px;height:18px;border-radius:3px;cursor:pointer;flex-shrink:0;")
		closeBtn.SetAttribute("hover-style", "background:"+colHover+";")
		closeIc := e.doc.CreateElement("span")
		closeIc.SetAttribute("data-icon", "x")
		closeIc.SetAttribute("style", "width:12px;height:12px;color:"+colTextDim+";")
		closeBtn.AppendChild(closeIc)
		closeIdx := i
		on(closeBtn, event.Click, func(e event.Event) bool {
			e.StopPropagation()
			Editor.CloseTab(closeIdx)
			return true
		})
		tab.AppendChild(closeBtn)
	}

	on(tab, event.Click, func(e event.Event) bool {
		e.StopPropagation()
		Editor.switchTo(i)
		return true
	})

	return tab
}

func (e *editorState) splitToggle() *dom.Element {
	txt := "分栏"
	col := colTextDim
	if e.split {
		txt = "单栏"
		col = colText
	}
	btn := e.doc.CreateElement("div")
	btn.SetAttribute("style", "display:flex;align-items:center;justify-content:center;height:34px;padding:0 10px;cursor:pointer;font-size:11px;color:"+col+";flex-shrink:0;")
	btn.SetAttribute("hover-style", "background:"+colHover+";")
	btn.SetTextContent(txt)
	on(btn, event.Click, func(e event.Event) bool {
		Editor.toggleSplit()
		return true
	})
	return btn
}

func (e *editorState) buildEditor(t *editorTab, tabIdx int) *dom.Element {
	if t == nil {
		el := e.doc.CreateElement("div")
		el.SetAttribute("style", "flex:1;background:"+colEditor+";")
		return el
	}

	container := e.doc.CreateElement("div")
	container.SetAttribute("style", "flex:1;display:flex;flex-direction:column;background:"+colEditor+";overflow:hidden;")

	// 使用 textarea 作为编辑器（临时方案，后续可替换为 CodeEditor 组件）
	ta := component.NewInput(e.doc, "")
	ta.SetBaseStyle("flex:1;background:" + colEditor + ";color:" + colText + ";border:none;padding:8px;font-size:14px;font-family:monospace;resize:none;outline:none;overflow:auto;")
	ta.SetValue(t.content)

	tabIdxCapture := tabIdx
	ta.OnChange(func(v string) {
		if tabIdxCapture < 0 || tabIdxCapture >= len(Editor.tabs) {
			return
		}
		tt := Editor.tabs[tabIdxCapture]
		tt.content = v
		if !tt.dirty {
			tt.dirty = true
			Editor.renderAll()
			if ui.Ctx.App != nil {
				ui.Ctx.App.MarkDirty()
			}
		}
	})

	container.AppendChild(ta.Element())
	return container
}

func (e *editorState) buildEditorColumn(idx int) *dom.Element {
	col := e.doc.CreateElement("div")
	col.SetAttribute("style", "flex:1;display:flex;flex-direction:column;overflow:hidden;")

	// 栏头
	if idx >= 0 && idx < len(e.tabs) {
		name := filepath.Base(e.tabs[idx].path)
		if e.tabs[idx].dirty {
			name = "● " + name
		}
		header := e.doc.CreateElement("div")
		header.SetAttribute("style", "height:24px;display:flex;flex-direction:row;align-items:center;padding:0 10px;background:"+colSide+";flex-shrink:0;")
		hText := e.doc.CreateElement("div")
		hText.SetAttribute("style", "color:"+colTextDim+";font-size:11px;")
		hText.SetTextContent(name)
		header.AppendChild(hText)
		col.AppendChild(header)
	}

	editorEl := e.buildEditor(e.TabAt(idx), idx)
	if editorEl != nil {
		col.AppendChild(editorEl)
	}
	return col
}

// ─── 欢迎页 ──────────────────────────────────────────────────

func (e *editorState) renderWelcome() {
	e.rootEl.ClearChildren()
	welcome := e.doc.CreateElement("div")
	welcome.SetAttribute("style", "display:flex;flex-direction:column;align-items:center;justify-content:center;height:100%;padding:40px;")

	h1 := e.doc.CreateElement("div")
	h1.SetAttribute("style", "font-size:32px;color:"+colText+";margin-bottom:16px;")
	h1.SetTextContent("Pair CodeAgent")
	welcome.AppendChild(h1)

	p := e.doc.CreateElement("div")
	p.SetAttribute("style", "font-size:16px;color:"+colTextDim+";margin-bottom:24px;")
	p.SetTextContent("打开文件开始编辑，或输入任务与 Agent 对话")
	welcome.AppendChild(p)

	actions := e.doc.CreateElement("div")
	actions.SetAttribute("style", "display:flex;flex-direction:row;gap:12px;")

	openBtn := e.doc.CreateElement("div")
	openBtn.SetAttribute("style", "padding:12px 24px;background:"+colAccent+";color:#fff;cursor:pointer;font-size:16px;border-radius:4px;")
	openBtn.SetAttribute("hover-style", "background:#1177bb;")
	openBtn.SetTextContent("打开文件")
	on(openBtn, event.Click, func(e event.Event) bool {
		if OnOpenFile != nil {
			OnOpenFile()
		}
		return true
	})
	actions.AppendChild(openBtn)

	newBtn := e.doc.CreateElement("div")
	newBtn.SetAttribute("style", "padding:12px 24px;background:"+colAccent+";color:#fff;cursor:pointer;font-size:16px;border-radius:4px;")
	newBtn.SetAttribute("hover-style", "background:#1177bb;")
	newBtn.SetTextContent("新建文件")
	on(newBtn, event.Click, func(e event.Event) bool {
		if OnNewFile != nil {
			OnNewFile()
		}
		return true
	})
	actions.AppendChild(newBtn)

	welcome.AppendChild(actions)
	e.rootEl.AppendChild(welcome)
}

// ─── 内部逻辑 ────────────────────────────────────────────────

func (e *editorState) switchTo(i int) {
	if i < 0 || i >= len(e.tabs) {
		return
	}
	e.active = i
	e.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (e *editorState) doClose(i int) {
	e.tabs = append(e.tabs[:i], e.tabs[i+1:]...)
	if e.active >= len(e.tabs) {
		e.active = len(e.tabs) - 1
	}
	e.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (e *editorState) doCloseAll() {
	e.tabs = nil
	e.active = 0
	e.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (e *editorState) toggleSplit() {
	e.split = !e.split
	if e.split {
		e.rightTab = pickOtherTab(e.active, len(e.tabs))
	}
	e.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
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

func (e *editorState) saveTab(t *editorTab) {
	if t == nil || !t.dirty {
		return
	}
	if err := os.WriteFile(t.path, []byte(t.content), 0o644); err == nil {
		t.dirty = false
	}
}

// ─── 确认关闭对话框 ────────────────────────────────────────

func (e *editorState) confirmCloseSingle(i int, t *editorTab) {
	name := filepath.Base(t.path)
	doc := e.doc
	modal := component.NewModal(doc)
	modal.SetTitle("未保存的更改")
	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()
	body.SetAttribute("style", "display:flex;flex-direction:column;gap:12px;min-width:300px;")
	msg := doc.CreateElement("div")
	msg.SetAttribute("style", "color:"+colText+";font-size:12px;")
	msg.SetTextContent(fmt.Sprintf("「%s」有未保存的更改。\n是否保存后再关闭？", name))
	body.AppendChild(msg)

	btnRow := doc.CreateElement("div")
	btnRow.SetAttribute("style", "display:flex;flex-direction:row;gap:8px;justify-content:flex-end;")
	// 不保存
	unsaveBtn := component.NewButton(doc, "不保存")
	unsaveBtn.OnClick(func() { modal.Hide(); e.doClose(i) })
	btnRow.AppendChild(unsaveBtn.Element())
	// 取消
	cancelBtn := component.NewButton(doc, "取消")
	cancelBtn.SetStyle("background-color:#9e9e9e;color:#fff;padding:4px 12px;")
	cancelBtn.OnClick(func() { modal.Hide() })
	btnRow.AppendChild(cancelBtn.Element())
	// 保存
	saveBtn := component.NewButton(doc, "保存")
	saveBtn.SetStyle("background-color:" + colAccent + ";color:#fff;padding:4px 12px;")
	saveBtn.OnClick(func() { modal.Hide(); e.saveTab(t); e.doClose(i) })
	btnRow.AppendChild(saveBtn.Element())
	body.AppendChild(btnRow)
	modal.Show()
}

// ─── 文件工具 ────────────────────────────────────────────────

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

func fileIcon(name string) (string, string) {
	// 按扩展名返回图标名和颜色
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go":
		return "file-code", "#00ADD8" // Go blue
	case ".js", ".ts", ".jsx", ".tsx":
		return "file-code", "#3178C6" // TypeScript blue
	case ".py":
		return "file-code", "#3572A5" // Python blue
	case ".rs":
		return "file-code", "#DEA584" // Rust orange
	case ".html", ".htm":
		return "file-code", "#E44D26" // HTML orange
	case ".css", ".scss", ".less":
		return "file-code", "#563D7C" // CSS purple
	case ".json":
		return "file-code", "#5B5B5B" // JSON gray
	case ".md":
		return "file-text", colAccent
	case ".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg":
		return "file-image", "#CBBE6E"
	case ".zip", ".tar", ".gz", ".7z", ".rar":
		return "file-archive", "#CBCB6E"
	case ".exe", ".dll", ".bin":
		return "file-binary", "#CBCB6E"
	default:
		if ext == "" || len(ext) > 6 {
			return "file", colTextDim
		}
		return "file", colTextDim
	}
}

func spacer(doc *dom.Document, w float64) *dom.Element {
	s := doc.CreateElement("div")
	s.SetAttribute("style", fmt.Sprintf("width:%.0fpx;flex-shrink:0;", w))
	return s
}

func on(el *dom.Element, typ event.Type, fn func(event.Event) bool) {
	if ui.Ctx.App != nil {
		ui.Ctx.App.AddEventListener(el, typ, fn)
	}
}

// ─── 外部回调 ────────────────────────────────────────────────

var (
	OnContentMenu func(x, y float64)
	OnTabMenu     func(x, y float64, i int)
	OnOpenFile    func()
	OnNewFile     func()
	OnOpenFolder  func()
	OnNewProject  func()
	OnOpenRecent  func(path string)
	OnReferences  func()
	OnSymbols     func()
	OnCursorMoved func()
)
