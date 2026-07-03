// 终端面板 —— 基于 TextArea 的终端。
//
// 架构分层：
//   TextArea（嵌入）— 文本显示、鼠标选中、光标闪烁
//   vterm — ANSI/VT 字节流 → 单元格网格
//   PTY — ConPTY 子进程（cmd/powershell）的 I/O
//
// 数据流：
//   PTY 输出 → reader goroutine → pending(mutex) → pump(drain) → vterm.Write() → 提取网格文本 → TextArea.SetText()
//   键盘输入 → UI 线程 handleKey → keyToVT() 转换 → PTY.Write()
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

const termIdleFrames = 180

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

	// 键盘事件路由到当前活跃的终端
	on(theTermMgr.termEl, event.KeyDown, func(e event.Event) bool {
		tw := theTerminal
		if tw == nil {
			return false
		}
		ke, ok := e.(*event.KeyboardEvent)
		if !ok {
			return false
		}
		tw.handleKey(ke)
		return true
	})

	on(theTermMgr.termEl, event.KeyPress, func(e event.Event) bool {
		tw := theTerminal
		if tw == nil {
			return false
		}
		ke, ok := e.(*event.KeyboardEvent)
		if !ok || ke.Char == 0 {
			return false
		}
		tw.handleKey(ke)
		return true
	})

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

	// 焦点跟踪
	on(theTermMgr.termEl, event.FocusIn, func(e event.Event) bool {
		return true
	})
	on(theTermMgr.termEl, event.FocusOut, func(e event.Event) bool {
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
		// 首次：创建第一个终端
		tw = newTerminalWidget("cmd", m)
		m.tabs[m.active] = tw
		theTerminal = tw
	}

	// 将 TextArea 元素放入 #terminal-area
	m.termEl.AppendChild(tw.textArea.Element())

	// 根据面板尺寸 resize
	tw.resizeFromPanel(m.termEl)

	// 聚焦终端区域（使键盘事件能到达）
	m.termEl.Focus()

	// 启动 PTY（如有 pending 数据则保持运行）
	tw.ensurePTY()
	_ = tw // 启动 pump
}

// ─── 单终端：TerminalWidget ─────────────────────────────────

// TerminalWidget 基于 TextArea 的终端组件。
// TextArea 处理：文本显示（等宽/无 CJK 问题）、鼠标选中/拖拽、光标闪烁。
// 本组件处理：键盘 → PTY、PTY→vterm→TextArea 显示、回滚。
type TerminalWidget struct {
	textArea *component.TextArea // 显示层
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
	idleFrames int
	scrollOff  int
	pumpID     int

	// 单元格尺寸缓存（用于 resize/滚动计算）
	cellW, cellH float64

	// 光标跟踪
	focused bool
}

func newTerminalWidget(shell string, mgr *termManager) *TerminalWidget {
	cwd, _ := os.Getwd()

	ta := component.NewTextArea(mgr.doc, "", 0, 0)

	// 设置终端样式
	ta.SetBaseStyle(
		"flex:1;min-height:0;"+
			"font-family:monospace;font-size:14px;line-height:1.2;"+
			"color:"+colText+";background:"+colEditor+";"+
			"padding:4px;"+
			"overflow:hidden;"+
			"white-space:pre;"+
			"border:none;outline:none;"+
			"box-shadow:none;")

	// 移除 contenteditable（终端不自编辑文本，输入路由到 PTY）
	ta.Element().RemoveAttribute("contenteditable")
	// 设置 tabindex=0 使元素可聚焦
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

	// 注册 TerminalWidget 作为 TextArea 元素的组件，覆盖 TextArea 自身的注册
	mgr.doc.RegisterComponent(ta.Element(), tw)

	// 点击时聚焦
	on(ta.Element(), event.MouseDown, func(e event.Event) bool {
		ta.Element().Focus()
		return false
	})

	return tw
}

// HandleEvent 实现 ComponentHandler 接口。
// 键盘事件 → PTY；鼠标事件 → TextArea（选中/拖拽/复制）；其他 → 回退。
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
		return tw.textArea.HandleEvent(e)

	case event.MouseMove:
		return tw.textArea.HandleEvent(e)

	case event.MouseUp:
		tw.textArea.HandleEvent(e)
		// 选中结束时自动复制到剪贴板
		if tw.textArea.HasSelection() {
			text := tw.textArea.SelectedText()
			if text != "" {
				component.CopyToClipboard(text)
				os.Stderr.WriteString("[terminal] auto-copied " + strconv.Itoa(len(text)) + " chars\n")
			}
		}
		return true

	case event.FocusIn:
		tw.focused = true
		return tw.textArea.HandleEvent(e)

	case event.FocusOut:
		tw.focused = false
		return tw.textArea.HandleEvent(e)

	default:
		return false
	}
}

func (tw *TerminalWidget) Focusable() bool { return true }

// Element 返回终端组件的根 DOM 元素（即 TextArea 的元素）。
func (tw *TerminalWidget) Element() *dom.Element { return tw.textArea.Element() }

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
	tw.mu.Lock()
	tw.idleFrames = 0
	if tw.pumpID != 0 {
		tw.mu.Unlock()
		return
	}
	tw.mu.Unlock()

	if ui.Ctx.App == nil {
		return
	}
	tw.pumpID = ui.Ctx.App.SetInterval(func() { tw.drain() }, 30*time.Millisecond)
}

func (tw *TerminalWidget) stopPump() {
	tw.mu.Lock()
	id := tw.pumpID
	tw.pumpID = 0
	tw.mu.Unlock()
	if id != 0 && ui.Ctx.App != nil {
		ui.Ctx.App.ClearInterval(id)
	}
}

func (tw *TerminalWidget) killPTY() {
	tw.mu.Lock()
	sess := tw.sess
	tw.sess, tw.alive, tw.failed = nil, false, false
	tw.mu.Unlock()
	if sess != nil {
		sess.Close()
	}
	tw.stopPump()
}

func (tw *TerminalWidget) drain() {
	tw.mu.Lock()
	var data []byte
	if len(tw.pending) > 0 {
		data = tw.pending
		tw.pending = nil
		tw.idleFrames = 0
	} else {
		tw.idleFrames++
	}
	idle := tw.idleFrames > termIdleFrames
	tw.mu.Unlock()

	if len(data) > 0 {
		tw.ensurePTY()
		tw.vt.Write(data)
		tw.syncDisplay()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
	}
	if idle {
		tw.stopPump()
	}
}

// syncDisplay 提取 vterm 网格文本显示到 TextArea。
func (tw *TerminalWidget) syncDisplay() {
	if tw.vt == nil {
		return
	}

	cols, rows := tw.vt.Size()
	scrLen := tw.vt.ScrollbackLen()

	// scrollOff=0 显示当前屏，>0 显示回滚历史
	startRow := scrLen - tw.scrollOff
	if startRow < 0 {
		startRow = 0
	}
	endRow := startRow + rows

	var b strings.Builder
	for r := startRow; r < endRow && r < scrLen+rows; r++ {
		if r > startRow {
			b.WriteByte('\n')
		}
		rowData := tw.vt.RowAt(r)
		// 找最后一个非空字符
		lastNonSpace := -1
		for c := 0; c < cols; c++ {
			var ch rune = ' '
			if c < len(rowData) && rowData[c].Ch != 0 {
				ch = rowData[c].Ch
			}
			if ch != ' ' {
				lastNonSpace = c
			}
		}
		// 写字符（跳过 Ch==0 续格和尾随空格）
		for c := 0; c <= lastNonSpace; c++ {
			if c < len(rowData) && rowData[c].Ch == 0 {
				continue // 跳过宽字符续格
			}
			var ch rune = ' '
			if c < len(rowData) && rowData[c].Ch != 0 {
				ch = rowData[c].Ch
			}
			b.WriteRune(ch)
		}
	}
	text := b.String()

	// 设置 TextArea 文本（这会触发 onRender 更新 DOM 和 data-cursor-pos）
	tw.textArea.SetText(text)

	// 调试日志
	if len(text) > 0 {
		preview := text
		if len(preview) > 80 {
			preview = preview[:80]
		}
		os.Stderr.WriteString("[syncDisplay] rows=" + strconv.Itoa(rows) + " text=" + strings.ReplaceAll(preview, "\n", "↵") + "\n")
	}
}

// ─── Resize ──────────────────────────────────────────────────

func (tw *TerminalWidget) resizeFromPanel(panelEl *dom.Element) {
	l, t_, r, b := panelEl.GetBoundingClientRect()
	panelW := r - l
	panelH := b - t_
	if panelW < 10 || panelH < 10 {
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
	data := keyToVT(ev)
	if len(data) == 0 {
		return
	}
	tw.scrollOff = 0 // 用户键入时回到当前屏

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
		return []byte{'\b'}
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
