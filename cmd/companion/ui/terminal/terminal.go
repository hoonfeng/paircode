// 终端面板 —— 基于 TextArea 的终端。
//
// 架构分层：
//   TextArea（仅显示层）— 文本显示、等宽字体 / CJK 支持
//   TerminalWidget（逻辑层）— 事件全部自己处理、不委托给 TextArea
//   vterm — ANSI/VT 字节流 → 单元格网格
//   PTY — ConPTY 子进程（cmd/powershell）的 I/O
//
// 关键约定：TerminalWidget 完整接管 TextArea 元素的组件注册，所有
// HandleEvent 不调用 TextArea.HandleEvent。显示直接用 DOM 属性操作
// （SetTextContent + 设 data-cursor-pos / data-sel-*），绕过 TextController。
//
//go:build windows

package termpanel

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"

	"github.com/hoonfeng/paircode/cmd/companion/pty"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/paircode/cmd/companion/vterm"
)


var (
	theTermMgr  *termManager
	theTerminal *TerminalWidget
)

func init() { theTermMgr = newTermManager() }

// ─── 多标签管理器 ───────────────────────────────────────────

type termManager struct {
	doc      *dom.Document
	rootEl   *dom.Element
	tabBarEl *dom.Element
	termEl   *dom.Element

	tabs   []*TerminalWidget
	active int
}

func newTermManager() *termManager {
	return &termManager{tabs: []*TerminalWidget{nil}}
}

func (m *termManager) NewTabWithShell(code string) {
	tw := newTerminalWidget(code, m)
	m.tabs = append(m.tabs, tw)
	m.active = len(m.tabs) - 1
	theTerminal = tw
	m.renderTabBar()
	m.renderActiveTerm()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (m *termManager) SetActiveShell(code string) {
	tw := m.tabs[m.active]
	if tw == nil || tw.shell == code {
		return
	}
	tw.shell = code
	tw.killPTY()
	m.renderTabBar()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (m *termManager) switchTab(i int) {
	if i < 0 || i >= len(m.tabs) || i == m.active {
		return
	}
	m.active = i
	theTerminal = m.tabs[i]
	m.renderTabBar()
	m.renderActiveTerm()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (m *termManager) closeTab(i int) {
	if i < 0 || i >= len(m.tabs) || len(m.tabs) == 1 {
		return
	}
	if tw := m.tabs[i]; tw != nil {
		tw.killPTY()
	}
	m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
	if m.active >= len(m.tabs) {
		m.active = len(m.tabs) - 1
	} else if m.active > i {
		m.active--
	}
	theTerminal = m.tabs[m.active]
	m.renderTabBar()
	m.renderActiveTerm()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

// ─── 面板 ──────────────────────────────────────────────────

// New 创建终端面板。
func New(doc *dom.Document) *termManager {
	theTermMgr.doc = doc

	ui.MustLoadPanelHTML(doc, "panels/terminal.html", nil)
	theTermMgr.rootEl = doc.GetElementByID("terminal-root")
	theTermMgr.tabBarEl = doc.GetElementByID("terminal-tabbar")
	theTermMgr.termEl = doc.GetElementByID("terminal-area")

	ui.DetachRoot(theTermMgr.rootEl)

	// 终端 Wheel 事件（在 #terminal-area 上监听，捕获可冒泡至此的 wheel）
	on(theTermMgr.termEl, event.Wheel, func(e event.Event) bool {
		tw := theTerminal
		if tw == nil {
			return false
		}
		we, ok := e.(*event.WheelEvent)
		if !ok {
			return false
		}
		tw.handleWheel(float64(we.DeltaY))
		return true
	})

	theTermMgr.renderTabBar()
	theTermMgr.renderActiveTerm()

	return theTermMgr
}

func (m *termManager) Element() *dom.Element { return m.rootEl }

func (m *termManager) Refresh() {
	m.renderTabBar()
	m.renderActiveTerm()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

// NewTerminal 新建默认终端（供外部调用）。
func NewTerminal() {
	theTermMgr.NewTabWithShell("cmd")
}

// OpenActiveTerminalDir 在活跃终端中 cd 到指定目录。
func OpenActiveTerminalDir(dir string) {
	if theTerminal == nil {
		NewTerminal()
	}
	if theTerminal != nil {
		theTerminal.OpenDir(dir)
	}
}

// ClearActive 清空活跃终端的屏幕。
func ClearActive() {
	if theTerminal != nil {
		theTerminal.ClearScreen()
	}
}

// CopyActiveAll 返回活跃终端的可见屏文本。
func CopyActiveAll() string {
	if theTerminal != nil {
		return theTerminal.CopyAll()
	}
	return ""
}

// PasteToActive 向活跃终端粘贴文本（发送到 PTY）。
func PasteToActive(text string) {
	if theTerminal == nil || text == "" {
		return
	}
	theTerminal.mu.Lock()
	sess := theTerminal.sess
	theTerminal.mu.Unlock()
	if sess != nil {
		sess.Write([]byte(text))
		theTerminal.startPump()
	}
}

func (m *termManager) renderTabBar() {
	if m.tabBarEl == nil {
		return
	}
	m.tabBarEl.ClearChildren()

	for i, tw := range m.tabs {
		idx := i
		bg := colStatusBar
		tc := colTextDim
		if i == m.active {
			bg = colEditor
			tc = colText
		}

		tab := m.doc.CreateElement("div")
		tab.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;padding:0 10px;height:28px;cursor:pointer;background:"+bg+";flex-shrink:0;")
		tab.SetAttribute("hover-style", "background:"+colHover+";")

		shellCode := "cmd"
		if tw != nil {
			shellCode = tw.shell
		}
		label := m.doc.CreateElement("div")
		label.SetAttribute("style", "color:"+tc+";font-size:11px;white-space:nowrap;")
		label.SetTextContent(shellLabel(shellCode))
		tab.AppendChild(label)

		on(tab, event.Click, func(e event.Event) bool {
			m.switchTab(idx)
			return true
		})

		m.tabBarEl.AppendChild(tab)

		if len(m.tabs) > 1 {
			closeBtn := m.doc.CreateElement("div")
			closeBtn.SetAttribute("style", "display:flex;align-items:center;justify-content:center;padding:0 8px;height:28px;cursor:pointer;background:"+bg+";flex-shrink:0;")
			closeBtn.SetAttribute("hover-style", "background:"+colHover+";")
			closeIc := m.doc.CreateElement("span")
			closeIc.SetAttribute("data-icon", "x")
			closeIc.SetAttribute("style", "width:11px;height:11px;color:"+colTextDim+";")
			closeBtn.AppendChild(closeIc)
			on(closeBtn, event.Click, func(e event.Event) bool {
				e.StopPropagation()
				m.closeTab(idx)
				return true
			})
			m.tabBarEl.AppendChild(closeBtn)
		}
	}

	// "+" 新建按钮
	plusBtn := m.doc.CreateElement("div")
	plusBtn.SetAttribute("style", "display:flex;align-items:center;justify-content:center;padding:0 8px;height:28px;cursor:pointer;flex-shrink:0;")
	plusBtn.SetAttribute("hover-style", "background:"+colHover+";")
	plusIc := m.doc.CreateElement("span")
	plusIc.SetAttribute("data-icon", "plus")
	plusIc.SetAttribute("style", "width:13px;height:13px;color:"+colTextDim+";")
	plusBtn.AppendChild(plusIc)
	on(plusBtn, event.Click, func(e event.Event) bool {
		m.NewTabWithShell("cmd")
		return true
	})
	m.tabBarEl.AppendChild(plusBtn)

	filler := m.doc.CreateElement("div")
	filler.SetAttribute("style", "flex:1;")
	m.tabBarEl.AppendChild(filler)
}

func (m *termManager) renderActiveTerm() {
	if m.termEl == nil {
		return
	}
	m.termEl.ClearChildren()
	if m.active < 0 || m.active >= len(m.tabs) {
		return
	}

	tw := m.tabs[m.active]
	if tw == nil {
		tw = newTerminalWidget("cmd", m)
		m.tabs[m.active] = tw
		theTerminal = tw
	}

	// 将 TextArea 元素放入 #terminal-area
	m.termEl.AppendChild(tw.textArea.Element())

	// 保存面板元素引用，供 resize 检测
	tw.panelEl = m.termEl

	// 根据面板尺寸 resize
	tw.resizeFromPanel(m.termEl)

	// 聚焦到 TextArea 元素上（键盘事件路由到 TerminalWidget.HandleEvent）
	tw.textArea.Element().Focus()

	// 启动 PTY
	tw.ensurePTY()
}

// ─── 单终端：TerminalWidget ─────────────────────────────────

// TerminalWidget 基于 TextArea 的终端组件。
//
// 事件全部自己处理，不委托给 TextArea.HandleEvent（避免与 TextController
// 内部的 cursorPos/selStart 状态冲突）。
//
// 显示流程（同步，主线程）：
//   drain() → vt.Write(data) → syncDisplay() →
//     ta.el.SetTextContent(text)          ← 不入 TextController
//     ta.el.SetAttribute("data-cursor-pos", …) ← 直接 DOM
//
// 鼠标选中（全权拥有）：
//   HandleEvent(MouseDown/Move/Up) → 自己维护 selStart/selEnd
//   → 设 data-sel-start/end → 选中复制
//
type TerminalWidget struct {
	textArea *component.TextArea // 仅用于显示层（元素 + 文本节点）
	doc      *dom.Document
	tabIdx   int

	// PTY + vterm
	mu         sync.Mutex
	vt         *vterm.Terminal
	sess       pty.PTY
	pending    []byte
	alive      bool
	failed     bool
	shell      string
	cwd        string
	cols, rows int
	scrollOff  int
	pumpID     int

	// 单元格尺寸
	cellW, cellH float64
	focused      bool
	panelEl      *dom.Element

	// 显示文本缓存（由 syncDisplay 更新）
	displayText string
	// 指向 TextArea 元素中的 Text 节点（供 SetText 直接更新内容，
	// 避免 SetTextContent 清空/重建 children 带来的不一致）
	textNode     *dom.Text
	// 每行信息缓存（鼠标坐标转换用）
	// vtermRows[i] = 第 i 行在 displayText 中的 flat 文本（不含行间 \n）
	vtermRows    []string
	vtermColFlat []map[int]int // vtermRows[i] 的列 → rune 索引

	// 鼠标选中（TerminalWidget 自管，不入 TextController）
	selDown  bool
	selStart int
	selEnd   int
}

// newTerminalWidget 创建新终端控件。
//
// 注意：TextArea 的 NewTextArea 在内部注册了它自己为组件。
// 调用后立即用 RegisterComponent 覆盖为 TerminalWidget，使所有事件
// 路由到 TerminalWidget.HandleEvent，而不是 TextArea.HandleEvent。
func newTerminalWidget(shell string, mgr *termManager) *TerminalWidget {
	cwd, _ := os.Getwd()

	ta := component.NewTextArea(mgr.doc, "", 0, 0)

	// 终端样式：必须含 width:auto 覆盖 NewTextArea 的 width:0px
	ta.SetBaseStyle(
		"flex:1;min-height:0;width:auto;"+
			"font-family:monospace;font-size:14px;line-height:1.2;"+
			"color:"+colText+";background:"+colEditor+";"+
			"padding:4px;"+
			"overflow:hidden;"+
			"white-space:pre;"+
			"border:none;outline:none;"+
			"box-shadow:none;")

	// 移除 contenteditable（终端不自编辑文本，输入路由到 PTY）
	ta.Element().RemoveAttribute("contenteditable")
	ta.Element().SetAttribute("tabindex", "0")

	tw := &TerminalWidget{
		textArea: ta,
		doc:      mgr.doc,
		vt:       vterm.New(80, 24),
		shell:    shell,
		cwd:      cwd,
		cols:     80,
		rows:     24,
		cellW:    14.0 * 0.6,
		cellH:    14.0 * 1.2,
		tabIdx:   len(mgr.tabs),
	}

	// 获取内置 Text 节点引用（NewTextArea 初始化时创建了一个 " " 文本节点）
	children := ta.Element().Children()
	if len(children) > 0 {
		if tn, ok := children[0].(*dom.Text); ok {
			tw.textNode = tn
		}
	}

	// 【关键】覆盖 TextArea 的组件注册，TerminalWidget 成为元素的事件处理器
	mgr.doc.RegisterComponent(ta.Element(), tw)

	return tw
}

// HandleEvent 实现 ComponentHandler 接口。
//
// 【原则】绝不委托给 tw.textArea.HandleEvent。所有事件自处理。
// 键盘 → PTY；鼠标 → 自己的选中逻辑；焦点 → 记状态。
func (tw *TerminalWidget) HandleEvent(e event.Event) bool {
	switch e.Type() {
	case event.KeyDown, event.KeyPress:
		ke, ok := e.(*event.KeyboardEvent)
		if !ok {
			return false
		}
		tw.handleKey(ke)
		return true

	case event.MouseDown:
		me, ok := e.(*event.MouseEvent)
		if !ok {
			return false
		}
		tw.handleMouseDown(me)
		return true

	case event.MouseMove:
		if !tw.selDown {
			return false
		}
		me, ok := e.(*event.MouseEvent)
		if !ok {
			return false
		}
		tw.handleMouseMove(me)
		return true

	case event.MouseUp:
		if !tw.selDown {
			return false
		}
		me, ok := e.(*event.MouseEvent)
		if !ok {
			return false
		}
		tw.handleMouseUp(me)
		return true

	case event.FocusIn:
		tw.focused = true
		return false

	case event.FocusOut:
		tw.focused = false
		// 失焦时隐藏光标
		tw.textArea.Element().RemoveAttribute("data-cursor-pos")
		return false

	default:
		return false
	}
}

func (tw *TerminalWidget) Focusable() bool { return true }

// Element 返回终端组件的根 DOM 元素（即 TextArea 的元素）。
func (tw *TerminalWidget) Element() *dom.Element { return tw.textArea.Element() }

// ─── 显示同步（核心）─────────────────────────────────────────

// syncDisplay 从 vterm 网格提取文本，直接用 DOM 操作显示。
//
// 关键：不使用 TextArea.SetText（它会重置 TextController.cursorPos）。
// 改用 ta.el.SetTextContent + 直接设 data-cursor-pos。
func (tw *TerminalWidget) syncDisplay() {
	if tw.vt == nil {
		return
	}

	cols, rows := tw.vt.Size()
	scrLen := tw.vt.ScrollbackLen()
	startRow := scrLen - tw.scrollOff
	if startRow < 0 {
		startRow = 0
	}
	endRow := startRow + rows

	// 提取网格文本，缓存行信息
	type rowInfo struct {
		text      string
		colToFlat map[int]int
	}
	var info []rowInfo

	for r := startRow; r < endRow && r < scrLen+rows; r++ {
		rowData := tw.vt.RowAt(r)
		runes := make([]rune, 0, cols)
		colToFlat := make(map[int]int, cols)
		contentful := false

		for c := 0; c < cols; c++ {
			if c < len(rowData) && rowData[c].Ch == 0 {
				continue // 宽字符续格跳过
			}
			var ch rune = ' '
			if c < len(rowData) && rowData[c].Ch != 0 {
				ch = rowData[c].Ch
			}
			if ch != ' ' {
				contentful = true
			}
			colToFlat[c] = len(runes)
			runes = append(runes, ch)
		}

		if contentful {
			info = append(info, rowInfo{text: string(runes), colToFlat: colToFlat})
		} else {
			info = append(info, rowInfo{text: "", colToFlat: colToFlat})
		}
	}

	// 裁剪尾随空行
	lastNonEmpty := len(info) - 1
	for lastNonEmpty >= 0 && info[lastNonEmpty].text == "" {
		lastNonEmpty--
	}

	// 拼接文本
	var b strings.Builder
	for i := 0; i <= lastNonEmpty; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(info[i].text)
	}
	text := b.String()

	// 【关键】用 Text.SetText 而非 Element.SetTextContent。
	// SetTextContent 会清空 e.children 再重建 Text 节点，频繁调用可能导致
	// 布局引擎读到不一致的 children 状态。SetText 只改节点内的字符串，
	// 保留 children 结构稳定。
	if tw.textNode != nil {
		tw.textNode.SetText(text)
	} else {
		el := tw.textArea.Element()
		el.SetTextContent(text)
	}

	// 缓存显示文本和行信息（供鼠标坐标转换）
	tw.displayText = text
	tw.vtermRows = make([]string, lastNonEmpty+1)
	tw.vtermColFlat = make([]map[int]int, lastNonEmpty+1)
	for i := 0; i <= lastNonEmpty; i++ {
		tw.vtermRows[i] = info[i].text
		tw.vtermColFlat[i] = info[i].colToFlat
	}

	// 同步光标
	cursorPos := tw.calcCursorPos()
	if cursorPos >= 0 && tw.focused {
		tw.textArea.Element().SetAttribute("data-cursor-pos", strconv.Itoa(cursorPos))
	}

	// 日志
	logText := text
	if len(logText) > 80 {
		logText = logText[:80]
	}
	os.Stderr.WriteString("[syncDisplay] rows=" + strconv.Itoa(rows) +
		" textLen=" + strconv.Itoa(len(text)) +
		" cursor=" + strconv.Itoa(cursorPos) +
		" text=" + strings.ReplaceAll(logText, "\n", "↵") + "\n")
}

// calcCursorPos 返回 vterm 光标在 flat 文本中的索引。
// scrollOff>0（回看）时返回 -1 表示不显示光标。
func (tw *TerminalWidget) calcCursorPos() int {
	if tw.vt == nil || tw.scrollOff > 0 {
		return -1
	}
	cx, cy := tw.vt.Cursor()

	// 使用缓存的行信息
	if cy < len(tw.vtermColFlat) {
		pos := 0
		for i := 0; i < cy; i++ {
			if i > 0 {
				pos++
			}
			pos += len([]rune(tw.vtermRows[i]))
		}
		if cy > 0 {
			pos++ // 行间 \n
		}
		if flat, ok := tw.vtermColFlat[cy][cx]; ok {
			pos += flat
		} else {
			// cx 对应续格，放在行尾
			pos += len([]rune(tw.vtermRows[cy]))
		}
		return pos
	}
	return 0
}

// ─── 鼠标选中（全权拥有）────────────────────────────────────

// handleMouseDown 处理鼠标点击：定位光标位置，开始选中。
func (tw *TerminalWidget) handleMouseDown(me *event.MouseEvent) {
	el := tw.textArea.Element()
	el.Focus()

	pos := tw.mousePosToFlat(me.OffsetX, me.OffsetY)
	tw.selDown = true
	tw.selStart = pos
	tw.selEnd = pos

	// 清除选中显示
	el.RemoveAttribute("data-sel-start")
	el.RemoveAttribute("data-sel-end")
}

// handleMouseMove 处理鼠标拖拽：更新选中范围。
func (tw *TerminalWidget) handleMouseMove(me *event.MouseEvent) {
	pos := tw.mousePosToFlat(me.OffsetX, me.OffsetY)
	tw.selEnd = pos

	el := tw.textArea.Element()
	if tw.selStart < tw.selEnd {
		el.SetAttribute("data-sel-start", strconv.Itoa(tw.selStart))
		el.SetAttribute("data-sel-end", strconv.Itoa(tw.selEnd))
	} else if tw.selEnd < tw.selStart {
		el.SetAttribute("data-sel-start", strconv.Itoa(tw.selEnd))
		el.SetAttribute("data-sel-end", strconv.Itoa(tw.selStart))
	} else {
		el.RemoveAttribute("data-sel-start")
		el.RemoveAttribute("data-sel-end")
	}
}

// handleMouseUp 处理鼠标释放：复制选中文本到剪贴板。
func (tw *TerminalWidget) handleMouseUp(me *event.MouseEvent) {
	tw.selDown = false

	if tw.selStart == tw.selEnd {
		return // 未选中
	}

	start, end := tw.selStart, tw.selEnd
	if start > end {
		start, end = end, start
	}

	runes := []rune(tw.displayText)
	if start < len(runes) && end <= len(runes) && start < end {
		selected := string(runes[start:end])
		if selected != "" {
			component.CopyToClipboard(selected)
			os.Stderr.WriteString("[terminal] auto-copied " + strconv.Itoa(len(selected)) + " chars\n")
		}
	}

	// 保留选中高亮
	el := tw.textArea.Element()
	if start < end {
		el.SetAttribute("data-sel-start", strconv.Itoa(start))
		el.SetAttribute("data-sel-end", strconv.Itoa(end))
	}
}

// mousePosToFlat 将鼠标偏移坐标（OffsetX/Y，相对于元素内容区）转换为
// displayText 中的 flat 索引。用缓存行信息定位。
func (tw *TerminalWidget) mousePosToFlat(ox, oy float32) int {
	if len(tw.vtermRows) == 0 {
		return 0
	}

	// OffsetX/OffsetY 是相对于元素内容区（padding inner edge）的坐标
	// 终端 padding:4px，所以文本网格从 (4,4) 开始
	padding := float32(4.0)
	relX := ox - padding
	relY := oy - padding
	if relX < 0 {
		relX = 0
	}
	if relY < 0 {
		relY = 0
	}

	cellW := float32(tw.cellW)
	cellH := float32(tw.cellH)
	if cellW < 1 {
		cellW = 14.0 * 0.6
	}
	if cellH < 1 {
		cellH = 14.0 * 1.2
	}

	col := int(relX / cellW)
	row := int(relY / cellH)

	// 钳位到有效范围
	if row >= len(tw.vtermRows) {
		row = len(tw.vtermRows) - 1
	}
	if row < 0 {
		row = 0
	}
	if col < 0 {
		col = 0
	}

	// 计算 flat 位置
	pos := 0
	for i := 0; i < row; i++ {
		if i > 0 {
			pos++
		}
		pos += len([]rune(tw.vtermRows[i]))
	}
	if row > 0 {
		pos++
	}

	// 列→flat 映射
	if row < len(tw.vtermColFlat) {
		if flat, ok := tw.vtermColFlat[row][col]; ok {
			pos += flat
		} else {
			// 超出已有列映射，放行尾
			if row < len(tw.vtermRows) {
				pos += len([]rune(tw.vtermRows[row]))
			}
		}
	}

	// 钳位到 total length
	totalLen := len([]rune(tw.displayText))
	if pos > totalLen {
		pos = totalLen
	}
	return pos
}

// ─── PTY 管理 ─────────────────────────────────────────────────

func (tw *TerminalWidget) ensurePTY() {
	tw.mu.Lock()
	if tw.alive || tw.failed {
		tw.mu.Unlock()
		return
	}
	tw.mu.Unlock()

	cols := tw.cols
	if cols < 10 {
		cols = 80
	}
	rows := tw.rows
	if rows < 3 {
		rows = 24
	}

	sess, err := pty.Start(ptyShellFor(tw.shell), tw.cwd, cols, rows)
	if err != nil {
		tw.mu.Lock()
		tw.failed = true
		tw.mu.Unlock()
		tw.vt = vterm.New(cols, rows)
		tw.vt.Write([]byte("[终端启动失败: " + err.Error() + "]\r\n"))
		tw.syncDisplay()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
		return
	}
	tw.vt = vterm.New(cols, rows)
	tw.mu.Lock()
	tw.sess, tw.alive, tw.cols, tw.rows = sess, true, cols, rows
	tw.mu.Unlock()
	go tw.reader(sess)
	tw.startPump()
}

func (tw *TerminalWidget) reader(sess pty.PTY) {
	buf := make([]byte, 4096)
	for {
		n, err := sess.Read(buf)
		if n > 0 {
			preview := string(buf[:n])
			previewStr := strconv.Quote(preview)
			if len(previewStr) > 120 {
				previewStr = previewStr[:120] + "..."
			}
			os.Stderr.WriteString("[reader] " + strconv.Itoa(n) + " bytes: " + previewStr + "\n")
			tw.mu.Lock()
			tw.pending = append(tw.pending, buf[:n]...)
			tw.mu.Unlock()
		}
		if err != nil {
			os.Stderr.WriteString("[reader] err: " + err.Error() + "\n")
			tw.mu.Lock()
			tw.alive = false
			tw.mu.Unlock()
			return
		}
	}
}

func (tw *TerminalWidget) startPump() {
	if ui.Ctx.App == nil {
		return
	}
	tw.mu.Lock()
	if tw.pumpID != 0 {
		tw.mu.Unlock()
		return
	}
	tw.pumpID = ui.Ctx.App.SetInterval(func() { tw.drain() }, 30*time.Millisecond)
	tw.mu.Unlock()
}

func (tw *TerminalWidget) killPTY() {
	tw.mu.Lock()
	sess := tw.sess
	tw.sess, tw.alive, tw.failed = nil, false, false
	tw.mu.Unlock()
	if sess != nil {
		sess.Close()
	}
}

func (tw *TerminalWidget) drain() {
	// 【关键】无数据时也要检查尺寸变化（最大化/最小化/拖拽改宽度）
	resized := tw.checkResize()

	tw.mu.Lock()
	var data []byte
	if len(tw.pending) > 0 {
		data = tw.pending
		tw.pending = nil
	}
	tw.mu.Unlock()

	if len(data) > 0 {
		os.Stderr.WriteString("[drain] processing " + strconv.Itoa(len(data)) + " bytes\n")
		tw.ensurePTY()
		tw.vt.Write(data)
		os.Stderr.WriteString("[drain] after vt.Write, calling syncDisplay\n")
		tw.syncDisplay()
		os.Stderr.WriteString("[drain] syncDisplay done\n")
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
	} else if resized {
		// 无数据但尺寸变了，更新显示
		tw.syncDisplay()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
	}
}

// ─── Resize ──────────────────────────────────────────────────

// checkResize 检查面板尺寸是否变化，变化时调用 resizeTo 并返回 true。
func (tw *TerminalWidget) checkResize() bool {
	if tw.panelEl == nil {
		return false
	}
	l, t_, r, b := tw.panelEl.GetBoundingClientRect()
	panelW := float64(r - l)
	panelH := float64(b - t_)
	if panelW < 10 || panelH < 10 {
		return false
	}
	padding := 4.0 * 2
	fontSize := 14.0
	cellW := fontSize * 0.6
	cellH := fontSize * 1.2
	newCols := int((panelW - padding) / cellW)
	newRows := int((panelH - padding) / cellH)
	if newCols < 10 {
		newCols = 10
	}
	if newRows < 3 {
		newRows = 3
	}
	if newCols != tw.cols || newRows != tw.rows {
		tw.resizeTo(newCols, newRows)
		return true
	}
	return false
}

func (tw *TerminalWidget) resizeFromPanel(panelEl *dom.Element) {
	l, t_, r, b := panelEl.GetBoundingClientRect()
	panelW := r - l
	panelH := b - t_
	if panelW < 10 || panelH < 10 {
		if ui.Ctx.App != nil {
			ui.Ctx.App.SetTimeout(func() {
				tw.resizeFromPanel(panelEl)
				ui.Ctx.App.MarkDirty()
			}, 100*time.Millisecond)
		}
		return
	}
	padding := 4.0 * 2
	fontSize := 14.0
	cellW := fontSize * 0.6
	cellH := fontSize * 1.2
	newCols := int((float64(panelW) - padding) / cellW)
	newRows := int((float64(panelH) - padding) / cellH)
	if newCols < 10 {
		newCols = 10
	}
	if newRows < 3 {
		newRows = 3
	}
	tw.cellW, tw.cellH = cellW, cellH
	tw.resizeTo(newCols, newRows)
}

func (tw *TerminalWidget) resizeTo(cols, rows int) {
	tw.mu.Lock()
	if cols == tw.cols && rows == tw.rows {
		tw.mu.Unlock()
		return
	}
	tw.cols, tw.rows = cols, rows
	sess := tw.sess
	alive := tw.alive
	tw.mu.Unlock()
	tw.vt.Resize(cols, rows)
	if alive && sess != nil {
		sess.Resize(cols, rows)
	}
}

// ─── 键盘 → VT ─────────────────────────────────────────────

func (tw *TerminalWidget) handleKey(ev *event.KeyboardEvent) {
	// KeyPress 控制字符（<=0x1F）由 KeyDown 已处理，跳过避免重复发送到 PTY
	if ev.Type() == event.KeyPress && ev.Char <= 0x1F {
		return
	}
	data := keyToVT(ev)
	if len(data) == 0 {
		return
	}
	tw.scrollOff = 0 // 用户键入时回到当前屏

	dataStr := strconv.Quote(string(data))
	os.Stderr.WriteString("[handleKey] key=" + strconv.Itoa(int(ev.Key)) +
		" char=" + strconv.Itoa(int(ev.Char)) +
		" ctrl=" + strconv.FormatBool(ev.Ctrl) +
		" data=" + dataStr + "\n")

	tw.mu.Lock()
	sess := tw.sess
	tw.mu.Unlock()
	if sess != nil {
		sess.Write(data)
		tw.startPump()
	}
}

func keyToVT(ev *event.KeyboardEvent) []byte {
	switch ev.Key {
	case event.CodeEnter:
		return []byte{'\r'}
	case event.CodeTab:
		return []byte{'\t'}
	case event.CodeBackspace:
		return []byte{'\x7f'}
	case event.CodeEscape:
		return []byte{27}
	case event.CodeUp:
		return []byte{27, '[', 'A'}
	case event.CodeDown:
		return []byte{27, '[', 'B'}
	case event.CodeRight:
		return []byte{27, '[', 'C'}
	case event.CodeLeft:
		return []byte{27, '[', 'D'}
	case event.CodeHome:
		return []byte{27, '[', 'H'}
	case event.CodeEnd:
		return []byte{27, '[', 'F'}
	case event.CodeDelete:
		return []byte{27, '[', '3', '~'}
	case event.CodeInsert:
		return []byte{27, '[', '2', '~'}
	case event.CodePageUp:
		return []byte{27, '[', '5', '~'}
	case event.CodePageDown:
		return []byte{27, '[', '6', '~'}
	case event.CodeF1:
		return []byte{27, 'O', 'P'}
	case event.CodeF2:
		return []byte{27, 'O', 'Q'}
	case event.CodeF3:
		return []byte{27, 'O', 'R'}
	case event.CodeF4:
		return []byte{27, 'O', 'S'}
	case event.CodeF5:
		return []byte{27, '[', '1', '5', '~'}
	case event.CodeF6:
		return []byte{27, '[', '1', '7', '~'}
	case event.CodeF7:
		return []byte{27, '[', '1', '8', '~'}
	case event.CodeF8:
		return []byte{27, '[', '1', '9', '~'}
	case event.CodeF9:
		return []byte{27, '[', '2', '0', '~'}
	case event.CodeF10:
		return []byte{27, '[', '2', '1', '~'}
	case event.CodeF11:
		return []byte{27, '[', '2', '3', '~'}
	case event.CodeF12:
		return []byte{27, '[', '2', '4', '~'}
	default:
		if ev.Char != 0 {
			if ev.Ctrl {
				ch := byte(ev.Char)
				if ch >= 'a' && ch <= 'z' {
					return []byte{ch - 'a' + 1}
				}
				if ch >= 'A' && ch <= 'Z' {
					return []byte{ch - 'A' + 1}
				}
			}
			return []byte(string(ev.Char))
		}
	}
	return nil
}

// ─── 滚轮回看 ──────────────────────────────────────────────

func (tw *TerminalWidget) handleWheel(deltaY float64) {
	off := tw.scrollOff + int(deltaY)*3
	if mx := tw.vt.ScrollbackLen(); off > mx {
		off = mx
	}
	if off < 0 {
		off = 0
	}
	if off != tw.scrollOff {
		tw.scrollOff = off
		tw.syncDisplay()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
	}
}

// ─── 操作 ────────────────────────────────────────────────────

func (tw *TerminalWidget) OpenDir(dir string) {
	tw.cwd = dir
	tw.mu.Lock()
	sess := tw.sess
	tw.mu.Unlock()
	if sess != nil {
		sess.Write([]byte("cd /d \"" + dir + "\"\r"))
		tw.startPump()
	}
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (tw *TerminalWidget) ClearScreen() {
	tw.mu.Lock()
	sess := tw.sess
	tw.mu.Unlock()
	cmd := "clear\r"
	if tw.shell == "cmd" {
		cmd = "cls\r"
	}
	if sess != nil {
		sess.Write([]byte(cmd))
		tw.startPump()
	} else {
		tw.vt = vterm.New(tw.cols, tw.rows)
		tw.syncDisplay()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
	}
}

func (tw *TerminalWidget) PasteToInput() {
	tw.mu.Lock()
	sess := tw.sess
	tw.mu.Unlock()
	if sess != nil {
		// TODO: 从剪贴板读取文本
		tw.startPump()
	}
}

// CopyAll 返回当前屏可见文本（用于菜单"全选复制"）。
func (tw *TerminalWidget) CopyAll() string {
	if tw.vt == nil {
		return ""
	}
	cols, rows := tw.vt.Size()
	var b strings.Builder
	for r := 0; r < rows; r++ {
		line := make([]rune, 0, cols)
		for c := 0; c < cols; c++ {
			if ch := tw.vt.Cell(r, c).Ch; ch != 0 {
				line = append(line, ch)
			}
		}
		b.WriteString(strings.TrimRight(string(line), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

// ─── 编码（桩）─────────────────────────────────────────────

func TermEncodingLabel() string {
	return "UTF-8"
}

// ─── Shell ──────────────────────────────────────────────────

type ShellOpt struct{ Code, Label string }

func AvailableShells() []ShellOpt {
	var out []ShellOpt
	seen := map[string]bool{}
	add := func(code string) {
		if !seen[code] {
			seen[code] = true
			out = append(out, ShellOpt{code, shellLabel(code)})
		}
	}
	for _, s := range pty.DetectShells() {
		switch s.Name {
		case "CMD":
			add("cmd")
		case "PowerShell", "PowerShell 7":
			add("powershell")
		case "Git Bash":
			add("gitbash")
		}
	}
	return out
}

func shellLabel(shell string) string {
	switch shell {
	case "powershell":
		return "PowerShell"
	case "gitbash":
		return "Bash"
	default:
		return "CMD"
	}
}

func ptyShellFor(code string) pty.Shell {
	switch code {
	case "cmd":
		return pty.ShellByName("CMD")
	case "powershell":
		if s := pty.ShellByName("PowerShell"); s.Name == "PowerShell" {
			return s
		}
		return pty.ShellByName("PowerShell 7")
	case "gitbash":
		return pty.ShellByName("Git Bash")
	}
	return pty.DefaultShell()
}

// ─── 颜色常量 ────────────────────────────────────────────────

var (
	colText      = ui.Text
	colTextDim   = ui.TextDim
	colEditor    = ui.EditorBg
	colStatusBar = ui.StatusBarBg
	colHover     = ui.HoverBg
	colSide      = ui.SideBg
)

// ─── 事件辅助 ──────────────────────────────────────────────

func on(el *dom.Element, typ event.Type, fn func(event.Event) bool) {
	if ui.Ctx.App != nil {
		ui.Ctx.App.AddEventListener(el, typ, fn)
	}
}

// OnContextMenu 由 main 注入。
var OnContextMenu func(x, y float64)

// Area 保留兼容。
func Area() *dom.Element { return theTermMgr.rootEl }
