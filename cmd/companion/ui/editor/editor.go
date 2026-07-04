// 编辑器面板 —— 中列编辑区：多标签页 + 编辑 + 保存（Ctrl+S）。
// 点文件树文件 → 新标签（或切到已打开标签）；改动标 dirty(●)；Ctrl+S 写盘。
//
//go:build windows

package editorpanel

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"
	"github.com/hoonfeng/gwui/svg"

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

// sessionFilePath 保存编辑器会话的文件路径。
var sessionFilePath string

func init() {
	// 注册编辑器专用 SVG 图标
	svg.RegisterIcon("columns", &svg.Icon{
		PathData: "M3 19V5h7v14H3zm11 0V5h7v14h-7z",
		ViewBoxW: 24, ViewBoxH: 24,
	})
	svg.RegisterIcon("single-panel", &svg.Icon{
		PathData: "M3 19V5h18v14H3z",
		ViewBoxW: 24, ViewBoxH: 24,
	})
}

// Editor 编辑器状态（包级单例）。
var Editor = &editorState{}

type editorTab struct {
	path    string
	content string
	lang    string
	dirty   bool
}

// editorSession 用于 JSON 序列化的会话数据。
type editorSession struct {
	Paths  []string `json:"paths"`
	Active int      `json:"active"`
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

	// ══ 缓存优化 ════════════════════════════════════════════
	// 避免每次 switchTo/Open/buildEditor 重新创建 CodeEditor（全量 DOM 重建）。
	// path → CodeEditor 实例
	editorCache map[string]*component.CodeEditor
	// path → 编辑器容器（dom.Element，flex:1 的外部包装容器）
	containerCache map[string]*dom.Element
}

// perfLog 打印编辑器性能日志（GWUI_DEBUG_PERF=1 时启用）。
func perfLog(label string, start time.Time) {
	if os.Getenv("GWUI_DEBUG_PERF") != "" {
		log.Printf("[perf:editor] %s: %v", label, time.Since(start))
	}
}

// ─── 公开方法 ────────────────────────────────────────────────

// New 创建编辑器面板（由 main.go 调用）。
func New(doc *dom.Document) *editorState {
	defer perfLog("New", time.Now())

	Editor.doc = doc
	Editor.editorCache = make(map[string]*component.CodeEditor)
	Editor.containerCache = make(map[string]*dom.Element)

	// 加载 HTML 模板（资源目录 html/panels/editor.html）
	ui.MustLoadPanelHTML(doc, "panels/editor.html", nil)
	Editor.rootEl = doc.GetElementByID("editor-root")
	Editor.contentArea = doc.GetElementByID("editor-content")

	// 从临时父节点（body）中分离根元素
	ui.DetachRoot(Editor.rootEl)

	// 欢迎页（默认）
	Editor.renderWelcome()
	return Editor
}

func (e *editorState) Element() *dom.Element { return e.rootEl }

func (e *editorState) Refresh() {
	e.renderAll()
}

// RestoreSession 恢复编辑器会话（main.go 启动时调用）。
// 从 ~/.paircode/session.json 读取上次打开的文件路径并重新打开。
func RestoreSession() {
	defer perfLog("RestoreSession", time.Now())

	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[editor] session: 无法获取 home 目录: %v", err)
		return
	}
	sessionFilePath = filepath.Join(home, ".paircode", "session.json")

	data, err := os.ReadFile(sessionFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[editor] session: 读取失败: %v", err)
		}
		return
	}

	var sess editorSession
	if err := json.Unmarshal(data, &sess); err != nil {
		log.Printf("[editor] session: JSON 解析失败: %v", err)
		return
	}
	if len(sess.Paths) == 0 {
		return
	}

	// 延迟打开：确保渲染就绪后执行
	for _, p := range sess.Paths {
		if _, err := os.Stat(p); err == nil {
			Editor.Open(p)
		}
	}
	if sess.Active >= 0 && sess.Active < len(sess.Paths) {
		Editor.switchTo(sess.Active)
	}
	if len(Editor.tabs) > 0 {
		log.Printf("[editor] session: 恢复了 %d 个标签", len(Editor.tabs))
	}
}

// saveSession 将当前打开文件的路径列表保存到 session.json。
func saveSession() {
	if sessionFilePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		sessionFilePath = filepath.Join(home, ".paircode", "session.json")
	}
	paths := make([]string, len(Editor.tabs))
	for i, t := range Editor.tabs {
		paths[i] = t.path
	}
	sess := editorSession{Paths: paths, Active: Editor.active}
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return
	}
	dir := filepath.Dir(sessionFilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	if err := os.WriteFile(sessionFilePath, data, 0o644); err != nil {
		log.Printf("[editor] session: 保存失败: %v", err)
	}
}

// CursorPosition 返回当前光标行/列（1 基），供状态栏显示。
func CursorPosition() (line, col int) {
	return Editor.cursorLine, Editor.cursorCol
}

// ActivePath 返回当前活动标签的文件路径（无打开标签返回空串）。
func ActivePath() string {
	if t := Editor.ActiveTab(); t != nil {
		return t.path
	}
	return ""
}

// ─── 公开方法（外部通过 ui.Ctx.Editor 调用）────────────────

func (e *editorState) Open(path string) {
	defer perfLog("Open("+filepath.Base(path)+")", time.Now())

	for i, t := range e.tabs {
		if t.path == path {
			if !t.dirty {
				loadTabContent(t)
				// 更新缓存中的编辑器内容
				if ce, ok := e.editorCache[path]; ok && !t.dirty {
					ce.SetText(t.content)
				}
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
	saveSession()
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
		// 更新脏标记：不重建整条 tab 栏，只修改对应标签的样式
		e.updateTabDirtyDot(e.active)
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
	saveSession()
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

// ── 剪贴板与编辑操作（供菜单/快捷键调用）────────────────

// activeEditor 返回当前活动标签的 CodeEditor 实例。
func (e *editorState) activeEditor() *component.CodeEditor {
	if e.active < 0 || e.active >= len(e.tabs) {
		return nil
	}
	path := e.tabs[e.active].path
	if ce, ok := e.editorCache[path]; ok {
		return ce
	}
	return nil
}

// Undo 撤销当前编辑器中的操作。
func (e *editorState) Undo() {
	if ce := e.activeEditor(); ce != nil {
		ce.Undo()
	}
}

// Redo 重做当前编辑器中的操作。
func (e *editorState) Redo() {
	if ce := e.activeEditor(); ce != nil {
		ce.Redo()
	}
}

// CopySelection 复制选中文本到剪贴板。
func (e *editorState) CopySelection() {
	if ce := e.activeEditor(); ce != nil {
		ce.CopySelection()
	}
}

// CutSelection 剪切选中文本到剪贴板。
func (e *editorState) CutSelection() {
	if ce := e.activeEditor(); ce != nil {
		ce.CutSelection()
	}
}

// PasteText 从剪贴板粘贴文本到编辑器。
func (e *editorState) PasteText() {
	if ce := e.activeEditor(); ce != nil {
		ce.PasteText()
	}
}

// ActiveCodeEditor 返回当前活动标签的 CodeEditor 实例（供右键菜单等外部使用）。
func (e *editorState) ActiveCodeEditor() *component.CodeEditor {
	return e.activeEditor()
}

func (e *editorState) ReloadIfOpen(path string) bool {
	for _, t := range e.tabs {
		if t.path == path {
			if t.dirty {
				return false
			}
			loadTabContent(t)
			// 更新缓存
			if ce, ok := e.editorCache[path]; ok {
				ce.SetText(t.content)
			}
			// 只需更新当前可见编辑器的内容
			if e.active == tabIndexByPath(path) && ui.Ctx.App != nil {
				ui.Ctx.App.MarkDirty()
			}
			return true
		}
	}
	return false
}

func tabIndexByPath(path string) int {
	for i, t := range Editor.tabs {
		if t.path == path {
			return i
		}
	}
	return -1
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
	defer perfLog("renderAll(tabs="+fmt.Sprint(len(e.tabs))+")", time.Now())

	e.contentArea.ClearChildren()

	if len(e.tabs) == 0 {
		e.renderWelcome()
		return
	}

	// 标签栏
	e.tabBar = e.doc.CreateElement("div")
	e.tabBar.SetAttribute("style", "display:flex;flex-direction:row;align-items:stretch;background:"+colSide+";height:34px;flex-shrink:0;overflow:hidden;")
	e.buildTabBar()
	e.contentArea.AppendChild(e.tabBar)

	// 编辑区容器（填充剩余空间）
	editorWrap := e.doc.CreateElement("div")
	editorWrap.SetAttribute("style", "flex:1;display:flex;flex-direction:column;overflow:hidden;")

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

		editorWrap.AppendChild(splitContainer)
	} else {
		editorEl := e.buildEditorWrapper(e.ActiveTab(), e.active)
		if editorEl != nil {
			editorWrap.AppendChild(editorEl)
		}
	}
	e.contentArea.AppendChild(editorWrap)
}

// buildTabBar 构建标签栏（全量）。
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
		dot.SetAttribute("id", fmt.Sprintf("editor-dirty-dot-%d", i))
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
	on(tab, event.ContextMenu, func(e event.Event) bool {
		e.StopPropagation()
		if me, ok := e.(*event.MouseEvent); ok && OnTabMenu != nil {
			OnTabMenu(float64(me.X), float64(me.Y), i)
		}
		return true
	})

	return tab
}

// updateTabDirtyDot 只修改第 i 个标签的脏点样式，不重建整个标签栏。
// 这是 OnChange 按键时的关键优化路径——避免 DOM 重建。
func (e *editorState) updateTabDirtyDot(i int) {
	if e.tabBar == nil || i < 0 || i >= len(e.tabs) || len(e.tabs) == 0 {
		// 退回到 buildTabBar
		e.buildTabBar()
		return
	}

	t := e.tabs[i]

	// 找到第 i 个标签对应的 tabBar 子元素
	// tabBar 结构：sep? tab[0] sep tab[1] ... filler splitToggle?
	childIdx := 0
	for j := 0; j < i; j++ {
		if j == 0 {
			childIdx++ // 第一个 tab 无前导分隔符
		} else {
			childIdx += 2 // 分隔符 + tab
		}
		childIdx++ // tab 本身
	}
	// 偏移到第 i 个 tab 的子节点
	tabIdx := 0
	for j := 0; j < i; j++ {
		if j > 0 {
			tabIdx++ // 分隔符
		}
		tabIdx++ // tab
	}

	var tabEl *dom.Element
	tabChildren := e.tabBar.Children()
	if tabIdx < len(tabChildren) {
		tabEl = tabChildren[tabIdx].(*dom.Element)
	}
	if tabEl == nil {
		e.buildTabBar()
		return
	}

	// 找到 dirty 或 close 按钮在 tabEl 的子节点中的位置：
	// icon(0) + spacer(1) + name(2) + spacer(3) + dirty/close(4)
	subChildren := tabEl.Children()
	// 备份或清除旧第 5 个子节点
	if len(subChildren) >= 5 {
		// 替换第 5 个（索引 4）子节点
		old := subChildren[4]
		tabEl.RemoveChild(old)
	}

	if t.dirty {
		dot := e.doc.CreateElement("div")
		dot.SetAttribute("style", "width:8px;height:8px;border-radius:4px;background:"+colText+";flex-shrink:0;")
		tabEl.AppendChild(dot)
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
		tabEl.AppendChild(closeBtn)
	}
}

func (e *editorState) splitToggle() *dom.Element {
	iconName := "single-panel"
	col := colTextDim
	if e.split {
		iconName = "columns"
		col = colText
	}
	btn := e.doc.CreateElement("div")
	btn.SetAttribute("style", "display:flex;align-items:center;justify-content:center;height:34px;padding:0 10px;cursor:pointer;flex-shrink:0;")
	btn.SetAttribute("hover-style", "background:"+colHover+";")
	icon := e.doc.CreateElement("span")
	icon.SetAttribute("data-icon", iconName)
	icon.SetAttribute("style", "width:15px;height:15px;display:inline-flex;align-items:center;justify-content:center;color:"+col+";")
	btn.AppendChild(icon)
	on(btn, event.Click, func(e event.Event) bool {
		Editor.toggleSplit()
		return true
	})
	return btn
}

// buildEditorWrapper 创建或复用编辑器的容器 + CodeEditor。
// 按文件路径缓存 CodeEditor 实例，避免每次 switchTo/Open 重建 DOM。
func (e *editorState) buildEditorWrapper(t *editorTab, tabIdx int) *dom.Element {
	defer perfLog("buildEditorWrapper("+filepath.Base(t.path)+")", time.Now())

	if t == nil {
		el := e.doc.CreateElement("div")
		el.SetAttribute("style", "flex:1;background:"+colEditor+";")
		return el
	}

	// 从缓存取已有的 editor 容器
	path := t.path
	if container, ok := e.containerCache[path]; ok {
		// 更新内容（如果已变更）
		if ce, ok := e.editorCache[path]; ok {
			ce.SetText(t.content)
		}
		// 确保 visible（可能在非活动时被隐藏）
		container.SetAttribute("style", "flex:1;display:flex;flex-direction:column;background:"+colEditor+";overflow:hidden;")
		return container
	}

	// 缓存未命中 → 创建新的 CodeEditor
	container := e.doc.CreateElement("div")
	container.SetAttribute("style", "flex:1;display:flex;flex-direction:column;background:"+colEditor+";overflow:hidden;")

	ce := component.NewCodeEditor(e.doc, t.lang, t.content, 1, 1)
	ce.SetFontSize(14)
	ce.SetFontFamily("monospace")
	ce.SetBaseStyle("flex:1;width:auto;height:auto;border:none;outline:none;padding:0;")

	// ── OnChange：修改时更新 content + 脏标记 ──
	// 注意：OnChange 回调中 tabIdxCapture 捕获的是创建时的索引。
	// 但切换标签后索引会变化。因此用路径来查找可靠的 tab。
	pathCapture := t.path
	ce.OnChange(func(v string) {
		// 通过路径查找当前 tab（不受索引变化影响）
		tabIdx := -1
		for j, tt := range Editor.tabs {
			if tt.path == pathCapture {
				tabIdx = j
				break
			}
		}
		if tabIdx < 0 {
			return
		}
		tt := Editor.tabs[tabIdx]
		tt.content = v
		if !tt.dirty {
			tt.dirty = true
			// 只更新脏点样式，不重建 tab 栏（核心性能优化）
			Editor.updateTabDirtyDot(tabIdx)
			if ui.Ctx.App != nil {
				ui.Ctx.App.MarkDirty()
			}
		}
	})

	// 右键菜单：编辑器内容区
	on(ce.Element(), event.ContextMenu, func(e event.Event) bool {
		e.StopPropagation()
		if me, ok := e.(*event.MouseEvent); ok && OnContentMenu != nil {
			OnContentMenu(float64(me.X), float64(me.Y))
		}
		return true
	})

	container.AppendChild(ce.Element())

	// 存入缓存
	e.editorCache[path] = ce
	e.containerCache[path] = container

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

	editorEl := e.buildEditorWrapper(e.TabAt(idx), idx)
	if editorEl != nil {
		col.AppendChild(editorEl)
	}
	return col
}

// ─── 欢迎页 ──────────────────────────────────────────────────

func (e *editorState) renderWelcome() {
	e.contentArea.ClearChildren()
	welcome := e.doc.CreateElement("div")
	welcome.SetAttribute("style", "display:flex;flex-direction:column;align-items:center;justify-content:center;height:100%;padding:40px;")

	h1 := e.doc.CreateElement("div")
	h1.SetAttribute("style", "font-size:28px;color:"+colText+";margin-bottom:16px;")
	h1.SetTextContent("Pair CodeAgent")
	welcome.AppendChild(h1)

	p := e.doc.CreateElement("div")
	p.SetAttribute("style", "font-size:13px;color:"+colTextDim+";margin-bottom:24px;")
	p.SetTextContent("打开文件开始编辑，或输入任务与 Agent 对话")
	welcome.AppendChild(p)

	actions := e.doc.CreateElement("div")
	actions.SetAttribute("style", "display:flex;flex-direction:row;gap:12px;")

	openBtn := e.doc.CreateElement("div")
	openBtn.SetAttribute("style", "padding:10px 20px;background:"+colAccent+";color:#fff;cursor:pointer;font-size:13px;border-radius:4px;")
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
	newBtn.SetAttribute("style", "padding:10px 20px;background:"+colAccent+";color:#fff;cursor:pointer;font-size:13px;border-radius:4px;")
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
	e.contentArea.AppendChild(welcome)
}

// ─── 内部逻辑 ────────────────────────────────────────────────

func (e *editorState) switchTo(i int) {
	defer perfLog("switchTo("+fmt.Sprint(i)+")", time.Now())

	if i < 0 || i >= len(e.tabs) {
		return
	}

	if e.active == i {
		return
	}

	prevActive := e.active
	e.active = i

	// 避免全量 renderAll，改为增量更新：
	// 1. 只更新 tab 样式（激活/非激活状态）
	// 2. 显隐编辑器容器
	// 3. 更新 contentArea 的子元素

	// 如果 tabBar 存在，更新每个 tab 的背景色
	if e.tabBar != nil {
		e.updateTabActiveStyles(prevActive, i)
	}

	// 更新编辑器区域内可见的编辑器
	e.updateVisibleEditor()

	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

// updateTabActiveStyles 只更新 tab 的背景色、文字颜色（不重建 tab 元素）。
func (e *editorState) updateTabActiveStyles(prev, cur int) {
	// 更新所有 tab 样式：遍历 tabBar 子节点中的 tab 元素
	childIdx := 0
	for j := 0; j < len(e.tabs); j++ {
		if j > 0 {
			childIdx++ // 分隔符
		}
		tabChildren := e.tabBar.Children()
		if childIdx >= len(tabChildren) {
			break
		}
		tabEl := tabChildren[childIdx].(*dom.Element)

		if j == cur {
			tabEl.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;height:34px;padding:0 8px 0 12px;cursor:pointer;background:"+colEditor+";flex-shrink:0;")
		} else if j == prev {
			tabEl.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;height:34px;padding:0 8px 0 12px;cursor:pointer;background:"+colSide+";flex-shrink:0;")
		}

		// 更新文件名字体颜色
		ch := tabEl.Children()
		if len(ch) >= 3 {
			nameEl := ch[2].(*dom.Element)
			if j == cur {
				nameEl.SetAttribute("style", "color:"+colText+";font-size:12px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:120px;")
			} else if j == prev {
				nameEl.SetAttribute("style", "color:"+colTextDim+";font-size:12px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:120px;")
			}
		}
		childIdx++ // tab
	}
}

// updateVisibleEditor 切换编辑区内的可见编辑器。
// 在非分屏模式下，用缓存中的编辑器替换显示。
func (e *editorState) updateVisibleEditor() {
	// 简单实现：如果内容区已经有编辑器子节点，直接替换
	if e.split {
		// 分屏模式下不进行增量优化，fallback 到 renderAll
		e.renderAll()
		return
	}

	t := e.ActiveTab()
	if t == nil {
		e.renderWelcome()
		return
	}

	// 获取编辑区子节点
	// contentArea 的子节点结构：tabBar(0) + editorWrap(1)
	children := e.contentArea.Children()
	if len(children) < 2 {
		// 结构意外，fallback
		e.renderAll()
		return
	}

	// 获取 editorWrap（第二个子节点）
	editorWrap := children[1].(*dom.Element)
	// 清空 editorWrap 的子节点
	editorWrap.ClearChildren()

	editorEl := e.buildEditorWrapper(t, e.active)
	if editorEl != nil {
		editorWrap.AppendChild(editorEl)
	}
}

// updateTabDirtyMarker 保留原接口（外部调用）——直接更新脏点。
// 避免重建整个标签栏。
func (e *editorState) updateTabDirtyMarker(i int) {
	if e.tabBar == nil || len(e.tabs) == 0 {
		return
	}
	e.updateTabDirtyDot(i)
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (e *editorState) doClose(i int) {
	// 清理缓存
	path := e.tabs[i].path
	delete(e.editorCache, path)
	delete(e.containerCache, path)

	e.tabs = append(e.tabs[:i], e.tabs[i+1:]...)
	if e.active >= len(e.tabs) {
		e.active = len(e.tabs) - 1
	}
	e.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
	saveSession()
}

func (e *editorState) doCloseAll() {
	// 清理所有缓存
	e.editorCache = make(map[string]*component.CodeEditor)
	e.containerCache = make(map[string]*dom.Element)
	e.tabs = nil
	e.active = 0
	e.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
	saveSession()
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
	t.content = strings.ReplaceAll(string(data), "\t", "    ")
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
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go":
		return "file-code", "#00ADD8"
	case ".js", ".ts", ".jsx", ".tsx":
		return "file-code", "#3178C6"
	case ".py":
		return "file-code", "#3572A5"
	case ".rs":
		return "file-code", "#DEA584"
	case ".html", ".htm":
		return "file-code", "#E44D26"
	case ".css", ".scss", ".less":
		return "file-code", "#563D7C"
	case ".json":
		return "file-code", "#5B5B5B"
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
