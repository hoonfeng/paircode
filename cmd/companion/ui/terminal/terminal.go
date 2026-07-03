// 终端面板 —— 真终端：ConPTY 伪终端(pty) + VT/ANSI 屏幕模型(vterm) + DOM 网格渲染。
// 多标签：每标签一个持久 PTY 会话喂 vterm；DOM 渲染 vterm 网格。
// 线程模型：读协程读 PTY 原始字节进 pending(加锁)；帧泵每帧在 UI 线程把 pending 喂进 vterm
// + 重绘 DOM 网格；久无输出停泵省电。键盘/resize/cd 都在 UI 线程。
//
// GWui 版：使用 dom.Document + app.SetInterval，不再依赖 goui。
//
//go:build windows

package termpanel

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"

	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/pty"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/paircode/cmd/companion/vterm"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const termIdleFrames = 180

var (
	theTermMgr  *termManager
	theTerminal *terminalState
)

func init() { theTermMgr = newTermManager() }

func NewState() *terminalState {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return &terminalState{cwd: cwd, shell: "cmd", vt: vterm.New(80, 24), cols: 80, rows: 24}
}

// ─── 多标签管理器 ───────────────────────────────────────────

type termManager struct {
	doc      *dom.Document
	rootEl   *dom.Element
	tabBarEl *dom.Element
	termEl   *dom.Element

	tabs    []*terminalState
	active  int
	focused bool
}

func newTermManager() *termManager {
	m := &termManager{tabs: []*terminalState{NewState()}}
	theTerminal = m.tabs[0]
	return m
}

func (m *termManager) NewTabWithShell(code string) {
	t := NewState()
	t.shell = code
	if cur := m.tabs[m.active]; cur != nil {
		t.cwd = cur.cwd
	}
	m.tabs = append(m.tabs, t)
	m.active = len(m.tabs) - 1
	theTerminal = t
	m.renderTabBar()
	m.renderActiveTerm()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (m *termManager) SetActiveShell(code string) {
	t := m.tabs[m.active]
	if t == nil || t.shell == code {
		return
	}
	t.shell = code
	t.killPTY()
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
	m.tabs[i].killPTY()
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

	// 加载 HTML 模板（资源目录 html/panels/terminal.html）
	ui.MustLoadPanelHTML(doc, "panels/terminal.html", nil)
	theTermMgr.rootEl = doc.GetElementByID("terminal-root")
	theTermMgr.tabBarEl = doc.GetElementByID("terminal-tabbar")
	theTermMgr.termEl = doc.GetElementByID("terminal-area")

	// 从临时父节点（body）中分离根元素
	ui.DetachRoot(theTermMgr.rootEl)

	theTermMgr.renderTabBar()
	theTermMgr.renderActiveTerm()

	// 注册可聚焦组件：使点击终端时能被 GWui 焦点系统识别
	doc.RegisterComponent(theTermMgr.termEl, &termFocusable{})

	// 焦点跟踪：用于控制光标显示
	on(theTermMgr.termEl, event.FocusIn, func(e event.Event) bool {
		theTermMgr.focused = true
		theTermMgr.Refresh()
		return true
	})
	on(theTermMgr.termEl, event.FocusOut, func(e event.Event) bool {
		theTermMgr.focused = false
		theTermMgr.Refresh()
		return true
	})

	// 键盘事件（KeyDown：控制键/功能键）
	on(theTermMgr.termEl, event.KeyDown, func(e event.Event) bool {
		ke, ok := e.(*event.KeyboardEvent)
		if !ok || theTerminal == nil {
			return false
		}
		theTerminal.handleKey(ke)
		return true
	})

	// 键盘事件（KeyPress：可打印字符——GWui 的 SetCharCallback 通过 KeyPress 分发）
	on(theTermMgr.termEl, event.KeyPress, func(e event.Event) bool {
		ke, ok := e.(*event.KeyboardEvent)
		if !ok || theTerminal == nil || ke.Char == 0 {
			return false
		}
		theTerminal.handleKey(ke)
		return true
	})

	on(theTermMgr.termEl, event.Wheel, func(e event.Event) bool {
		we, ok := e.(*event.WheelEvent)
		if !ok || theTerminal == nil {
			return false
		}
		theTerminal.handleWheel(float64(we.DeltaY))
		return true
	})

	on(theTermMgr.termEl, event.MouseDown, func(e event.Event) bool {
		me, ok := e.(*event.MouseEvent)
		if !ok || theTerminal == nil {
			return false
		}
		// 点击终端时请求焦点，确保后续键盘事件能路由到终端区域
		theTermMgr.termEl.Focus()
		theTerminal.onSelDown(float64(me.X), float64(me.Y))
		return true
	})

	on(theTermMgr.termEl, event.MouseMove, func(e event.Event) bool {
		me, ok := e.(*event.MouseEvent)
		if !ok || theTerminal == nil {
			return false
		}
		theTerminal.onSelDrag(float64(me.X), float64(me.Y))
		return true
	})

	on(theTermMgr.termEl, event.MouseUp, func(e event.Event) bool {
		if theTerminal == nil {
			return false
		}
		theTerminal.onSelUp(0, 0)
		return true
	})

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

// NewTerminal 新建默认终端。
func NewTerminal() {
	theTermMgr.NewTabWithShell("cmd")
}

// OpenActiveTerminalDir 在活跃终端中 cd 到指定目录。
// 若没有活跃终端则新建一个。
func OpenActiveTerminalDir(dir string) {
	if theTerminal == nil {
		NewTerminal()
	}
	if theTerminal != nil {
		theTerminal.OpenDir(dir)
	}
}

func (m *termManager) renderTabBar() {
	m.tabBarEl.ClearChildren()

	for i, t := range m.tabs {
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

		label := m.doc.CreateElement("div")
		label.SetAttribute("style", "color:"+tc+";font-size:11px;white-space:nowrap;")
		label.SetTextContent(shellLabel(t.shell))
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
	m.termEl.ClearChildren()
	if m.active < 0 || m.active >= len(m.tabs) {
		return
	}

	t := m.tabs[m.active]
	gridEl := m.doc.CreateElement("div")
	gridEl.SetAttribute("style", "flex:1;min-height:0;padding:4px;font-family:monospace;font-size:14px;line-height:1.2;color:"+colText+";background:"+colEditor+";overflow:hidden;white-space:pre;")

	t.gridEl = gridEl
	m.termEl.AppendChild(gridEl)

	// 创建光标覆盖层（绝对定位在 grid 内 + 闪烁动画）
	cursorEl := m.doc.CreateElement("div")
	cursorEl.SetAttribute("style", "position:absolute;width:9px;height:17px;background:"+colCursor+";display:none;"+
		"animation:termCursorBlink 1s step-end infinite;pointer-events:none;z-index:10;"+
		"transition:left 0.05s,top 0.05s;")
	// 注入 CSS keyframe（只做一次）
	if m.doc.GetElementByID("term-cursor-style") == nil {
		style := m.doc.CreateElement("style")
		style.SetAttribute("id", "term-cursor-style")
		style.SetTextContent("@keyframes termCursorBlink { 0%,100% { opacity:1; } 50% { opacity:0.15; } }")
		m.doc.Head().AppendChild(style)
	}
	m.termEl.AppendChild(cursorEl)
	t.cursorEl = cursorEl

	// 启动 PTY + vterm + pump 全链路
	t.renderGrid()
}

// ─── 单终端状态 ─────────────────────────────────────────────

type terminalState struct {
	gridEl   *dom.Element
	cursorEl *dom.Element

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
	decCarry   []byte

	selecting    bool
	hasSel       bool
	selAR, selAC int
	selCR, selCC int
	cellW, cellH float64
}

type vtSel struct {
	rowA, colA int
	rowB, colB int
}

func (t *terminalState) ensurePTY(cols, rows int) {
	t.mu.Lock()
	if t.alive || t.failed {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	sess, err := pty.Start(ptyShellFor(t.shell), t.cwd, cols, rows)
	if err != nil {
		t.mu.Lock()
		t.failed = true
		t.mu.Unlock()
		t.vt = vterm.New(cols, rows)
		t.vt.Write([]byte("[终端启动失败: " + err.Error() + "]\r\n"))
		t.renderGrid()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
		return
	}
	t.vt = vterm.New(cols, rows)
	t.mu.Lock()
	t.sess, t.alive, t.cols, t.rows = sess, true, cols, rows
	t.mu.Unlock()
	go t.reader(sess)
	t.startPump()
}

func (t *terminalState) reader(sess pty.PTY) {
	buf := make([]byte, 4096)
	for {
		n, err := sess.Read(buf)
		if n > 0 {
			os.Stderr.WriteString("[reader] read " + strconv.Itoa(n) + " bytes\n")
			t.mu.Lock()
			t.pending = append(t.pending, buf[:n]...)
			t.mu.Unlock()
		}
		if err != nil {
			os.Stderr.WriteString("[reader] err: " + err.Error() + "\n")
			t.mu.Lock()
			t.alive = false
			t.mu.Unlock()
			return
		}
	}
}

func (t *terminalState) drain() {
	t.mu.Lock()
	var data []byte
	if len(t.pending) > 0 {
		data = t.pending
		t.pending = nil
		t.idleFrames = 0
	} else {
		t.idleFrames++
	}
	idle := t.idleFrames > termIdleFrames
	t.mu.Unlock()

	if len(data) > 0 {
		logHex := func(prefix string, b []byte) {
			hexStr := make([]string, len(b))
			for i, v := range b {
				hexStr[i] = strconv.FormatUint(uint64(v), 16)
			}
			maxShow := 64
			if len(hexStr) > maxShow {
				hexStr = hexStr[:maxShow]
			}
			os.Stderr.WriteString(prefix + " " + strconv.Itoa(len(b)) + " bytes: [" + strings.Join(hexStr, " ") + "]\n")
		}
		logHex("[drain] raw", data)
		data = t.decodeOutput(data)
		logHex("[drain] decoded", data)
		before := t.vt.ScrollbackLen()
		t.vt.Write(data)
		if t.scrollOff > 0 {
			t.scrollOff += t.vt.ScrollbackLen() - before
			if mx := t.vt.ScrollbackLen(); t.scrollOff > mx {
				t.scrollOff = mx
			}
		}
		t.renderGrid()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
	}
	if idle {
		os.Stderr.WriteString("[drain] idle, stopping pump\n")
		t.stopPump()
	}
}

func (t *terminalState) renderGrid() {
	if t.gridEl == nil || t.vt == nil {
		return
	}

	cols, rows := t.vt.Size()
	t.ensurePTY(cols, rows)

	scrLen := t.vt.ScrollbackLen()
	// scrollOff=0 时显示当前屏幕（scrLen ~ scrLen+rows-1），scrollOff>0 时向上滚动
	startRow := scrLen - t.scrollOff
	if startRow < 0 {
		startRow = 0
	}
	endRow := startRow + rows
	if endRow > scrLen+rows {
		endRow = scrLen + rows
	}

	// 构建完整文本
	var b strings.Builder
	for r := startRow; r < endRow; r++ {
		if r > startRow {
			b.WriteByte('\n')
		}
		rowData := t.vt.RowAt(r)
		for c := 0; c < cols; c++ {
			ch := ' '
			if c < len(rowData) && rowData[c].Ch != 0 {
				ch = rowData[c].Ch
			}
			b.WriteRune(ch)
		}
	}
	text := b.String()

	// 调试日志
	os.Stderr.WriteString("[renderGrid] cols=" + strconv.Itoa(cols) + " rows=" + strconv.Itoa(rows) + " textLen=" + strconv.Itoa(len(text)) + " scrollLen=" + strconv.Itoa(scrLen) + "\n")
	if len(text) > 0 {
		preview := text
		if len(preview) > 120 {
			preview = preview[:120]
		}
		os.Stderr.WriteString("[renderGrid] preview=" + strings.ReplaceAll(preview, "\n", "↵") + "\n")
	}

	// 完全替换元素，强制重布局
	if theTermMgr != nil && theTermMgr.doc != nil && t.gridEl.Parent() != nil {
		parentEl, ok := t.gridEl.Parent().(*dom.Element)
		if ok {
			newEl := theTermMgr.doc.CreateElement("div")
			newEl.SetAttribute("style", "flex:1;min-height:0;padding:4px;font-family:monospace;font-size:14px;line-height:1.2;color:"+colText+";background:"+colEditor+";overflow:hidden;white-space:pre;")
			newEl.SetTextContent(text)
			parentEl.ReplaceChild(newEl, t.gridEl)
			t.gridEl = newEl
		} else {
			t.gridEl.SetTextContent(text)
		}
	} else {
		t.gridEl.SetTextContent(text)
	}

	// 验证文本是否设置成功
	actualText := t.gridEl.TextContent()
	os.Stderr.WriteString("[renderGrid] after SetTextContent, TextContent() len=" + strconv.Itoa(len(actualText)) + "\n")

	// ── 更新光标位置 ──
	t.updateCursor(cols, rows, scrLen, startRow)
}

func (t *terminalState) updateCursor(cols, rows, scrLen, startRow int) {
	if t.cursorEl == nil || theTermMgr == nil || theTermMgr.doc == nil {
		return
	}
	// 检查终端区域是否有焦点（用 data-focused 标记在 terminal-area 上）
	focused := theTermMgr.focused

	cx, cy := t.vt.Cursor()
	// 光标在组合缓冲中的位置：scrolled + cursor row
	cursorCompositeRow := scrLen + cy
	visRow := cursorCompositeRow - startRow
	if visRow < 0 || visRow >= rows {
		t.cursorEl.SetAttribute("style", "display:none;")
		return
	}

	// 计算单元格尺寸
	padding := 4 // 与 gridEl 的 padding 一致
	fontSize := 14.0
	cellW := fontSize * 0.6   // 约 0.6 em = monospace 字符宽
	cellH := fontSize * 1.2   // line-height
	left := float64(padding) + float64(cx)*cellW
	top := float64(padding) + float64(visRow)*cellH

	disp := "block"
	if !focused {
		disp = "none"
	}
	t.cursorEl.SetAttribute("style",
		"position:absolute;left:"+strconv.FormatFloat(left, 'f', 1, 64)+"px;"+
			"top:"+strconv.FormatFloat(top, 'f', 1, 64)+"px;"+
			"width:"+strconv.FormatFloat(cellW, 'f', 1, 64)+"px;"+
			"height:"+strconv.FormatFloat(cellH, 'f', 1, 64)+"px;"+
			"background:"+colCursor+";display:"+disp+";"+
			"animation:termCursorBlink 1s step-end infinite;"+
			"pointer-events:none;z-index:10;"+
			"transition:left 0.05s,top 0.05s;")
}

// ─── 按键 → VT ─────────────────────────────────────────────

func (t *terminalState) handleKey(ev *event.KeyboardEvent) {
	data := keyToVT(ev)
	if len(data) == 0 {
		return
	}
	t.scrollOff = 0

	// 用系统代码页编码转换：GBK 系统需把 UTF-8 输入转 GBK 后写入 ConPTY
	// decodeOutput 方向相反（GBK→UTF-8），保证 vterm 始终处理 UTF-8。
	enc := core.Settings.TermEncoding
	if enc == "auto" {
		enc = detectSystemEncoding()
	}
	if enc == "gbk" {
		encoder := simplifiedchinese.GBK.NewEncoder()
		converted, _, err := transform.Bytes(encoder, data)
		if err == nil && len(converted) > 0 {
			data = converted
		} else {
			os.Stderr.WriteString("[handleKey] gbk encode err: " + err.Error() + "; using raw utf8\n")
		}
	}

	logHex := func(prefix string, b []byte) {
		hexStr := make([]string, len(b))
		for i, v := range b {
			hexStr[i] = strconv.FormatUint(uint64(v), 16)
		}
		os.Stderr.WriteString(prefix + " " + strconv.Itoa(len(b)) + " bytes: [" + strings.Join(hexStr, " ") + "]\n")
	}
	logHex("[handleKey] writing", data)

	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	if sess != nil {
		n, err := sess.Write(data)
		if err != nil {
			os.Stderr.WriteString("[handleKey] write err: " + err.Error() + "\n")
		} else {
			os.Stderr.WriteString("[handleKey] write OK: " + strconv.Itoa(n) + " bytes\n")
		}
		t.startPump()
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
			// 用 UTF-8 编码 rune（支持中文等多字节字符；byte(ev.Char) 会截断 >255 的字符）
			return []byte(string(ev.Char))
		}
	}
	return nil
}

// ─── 滚轮回看 ──────────────────────────────────────────────

func (t *terminalState) handleWheel(deltaY float64) {
	off := t.scrollOff + int(deltaY)*3
	if mx := t.vt.ScrollbackLen(); off > mx {
		off = mx
	}
	if off < 0 {
		off = 0
	}
	if off != t.scrollOff {
		t.scrollOff = off
		t.renderGrid()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
	}
}

// ─── 鼠标选区 ──────────────────────────────────────────────

func (t *terminalState) cellAt(localX, localY float64) (row, col int) {
	if t.cellW <= 0 || t.cellH <= 0 {
		return 0, 0
	}
	col = int(localX / t.cellW)
	if col < 0 {
		col = 0
	}
	vr := int(localY / t.cellH)
	if vr < 0 {
		vr = 0
	}
	row = (t.vt.ScrollbackLen() - t.scrollOff) + vr
	return
}

func (t *terminalState) onSelDown(x, y float64) {
	t.selAR, t.selAC = t.cellAt(x, y)
	t.selCR, t.selCC = t.selAR, t.selAC
	t.selecting, t.hasSel = true, false
}

func (t *terminalState) onSelDrag(x, y float64) {
	if !t.selecting {
		return
	}
	t.selCR, t.selCC = t.cellAt(x, y)
	t.hasSel = t.selCR != t.selAR || t.selCC != t.selAC
}

func (t *terminalState) onSelUp(x, y float64) {
	t.selecting = false
	if t.hasSel {
		t.copySelection()
	}
}

func (t *terminalState) copySelection() string {
	s := t.normSel()
	if s == nil {
		return ""
	}
	cols, _ := t.vt.Size()
	var b strings.Builder
	for r := s.rowA; r <= s.rowB; r++ {
		row := t.vt.RowAt(r)
		c0, c1 := 0, cols
		if r == s.rowA {
			c0 = s.colA
		}
		if r == s.rowB {
			c1 = s.colB
		}
		var line []rune
		for c := c0; c < c1; c++ {
			ch := ' '
			if c < len(row) && row[c].Ch != 0 {
				ch = row[c].Ch
			}
			line = append(line, ch)
		}
		b.WriteString(strings.TrimRight(string(line), " "))
		if r < s.rowB {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (t *terminalState) normSel() *vtSel {
	if !t.hasSel {
		return nil
	}
	rA, cA, rB, cB := t.selAR, t.selAC, t.selCR, t.selCC
	if rA > rB || (rA == rB && cA > cB) {
		rA, cA, rB, cB = rB, cB, rA, cA
	}
	return &vtSel{rA, cA, rB, cB}
}

// ─── PTY ────────────────────────────────────────────────────

func (t *terminalState) resizeTo(cols, rows int) {
	t.mu.Lock()
	if !t.alive || (cols == t.cols && rows == t.rows) {
		t.mu.Unlock()
		return
	}
	t.cols, t.rows = cols, rows
	sess := t.sess
	t.mu.Unlock()
	t.vt.Resize(cols, rows)
	if sess != nil {
		sess.Resize(cols, rows)
	}
}

func (t *terminalState) startPump() {
	t.mu.Lock()
	t.idleFrames = 0
	if t.pumpID != 0 {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	if ui.Ctx.App == nil {
		os.Stderr.WriteString("[startPump] ui.Ctx.App is nil, CANNOT start pump\n")
		return
	}
	os.Stderr.WriteString("[startPump] registering SetInterval pump\n")
	t.pumpID = ui.Ctx.App.SetInterval(func() { t.drain() }, 30*time.Millisecond)
}

func (t *terminalState) stopPump() {
	t.mu.Lock()
	id := t.pumpID
	t.pumpID = 0
	t.mu.Unlock()
	if id != 0 && ui.Ctx.App != nil {
		ui.Ctx.App.ClearInterval(id)
	}
}

func (t *terminalState) killPTY() {
	t.mu.Lock()
	sess := t.sess
	t.sess, t.alive, t.failed = nil, false, false
	t.mu.Unlock()
	if sess != nil {
		sess.Close()
	}
	t.stopPump()
}

// ─── 操作 ────────────────────────────────────────────────────

func (t *terminalState) OpenDir(dir string) {
	t.cwd = dir
	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	if sess != nil {
		sess.Write([]byte("cd /d \"" + dir + "\"\r"))
		t.startPump()
	}
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (t *terminalState) CopyAll() string {
	cols, rows := t.vt.Size()
	var b strings.Builder
	for r := 0; r < rows; r++ {
		line := make([]rune, 0, cols)
		for c := 0; c < cols; c++ {
			if ch := t.vt.Cell(r, c).Ch; ch != 0 {
				line = append(line, ch)
			}
		}
		b.WriteString(strings.TrimRight(string(line), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

func (t *terminalState) PasteToInput() {
	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	if sess != nil {
		sess.Write([]byte(t.clipboardText()))
		t.startPump()
	}
}

func (t *terminalState) clipboardText() string {
	return ""
}

func (t *terminalState) ClearScreen() {
	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	cmd := "clear\r"
	if t.shell == "cmd" {
		cmd = "cls\r"
	}
	if sess != nil {
		sess.Write([]byte(cmd))
		t.startPump()
	} else {
		t.vt = vterm.New(t.cols, t.rows)
		t.renderGrid()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
	}
}

// ─── 编码 ────────────────────────────────────────────────────

func TermEncodingLabel() string {
	if core.Settings.TermEncoding == "gbk" {
		return "GBK"
	}
	return "UTF-8"
}

func (t *terminalState) decodeOutput(b []byte) []byte {
	enc := core.Settings.TermEncoding
	if enc == "auto" {
		enc = detectSystemEncoding()
	}
	if enc != "gbk" {
		return b
	}
	src := b
	if len(t.decCarry) > 0 {
		src = append(t.decCarry, b...)
		t.decCarry = nil
	}
	out, carry := gbkToUTF8(src)
	t.decCarry = carry
	return out
}

// detectSystemEncoding 检测 ConPTY 实际使用的输出编码。
// OEM 代码页（GetOEMCP）是控制台子系统的默认编码，ConPTY 输出使用此编码。
// 代码页 936（简体中文 GBK）→ 返回 "gbk"，否则返回 "utf-8"。
func detectSystemEncoding() string {
	mod := syscall.NewLazyDLL("kernel32.dll")
	// OEM 代码页是控制台/ConPTY 的原始输出编码（cmd.exe / PowerShell 用此编码输出）
	proc := mod.NewProc("GetOEMCP")
	r, _, _ := proc.Call()
	os.Stderr.WriteString("[encoding] GetOEMCP=" + strconv.FormatUint(uint64(r), 10) + "\n")
	if r == 936 {
		return "gbk"
	}
	// 回退：检查 ANSI 代码页（UTF-8 beta 开启时 GetACP=65001，否则 936）
	proc2 := mod.NewProc("GetACP")
	r2, _, _ := proc2.Call()
	os.Stderr.WriteString("[encoding] GetACP=" + strconv.FormatUint(uint64(r2), 10) + "\n")
	if r2 == 936 {
		return "gbk"
	}
	return "utf-8"
}

func gbkToUTF8(b []byte) (out, carry []byte) {
	dst := make([]byte, len(b)*2+16)
	nDst, nSrc, err := simplifiedchinese.GBK.NewDecoder().Transform(dst, b, false)
	out = append([]byte(nil), dst[:nDst]...)
	rest := b[nSrc:]
	switch {
	case err == transform.ErrShortSrc:
		carry = append([]byte(nil), rest...)
	case err != nil && len(rest) > 0:
		carry = append([]byte(nil), rest[1:]...)
	}
	return
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

// ─── 可聚焦组件 ─────────────────────────────────────────────

// termFocusable 实现 app.FocusableComponent 接口（HandleEvent + Focusable），
// 使 #terminal-area 能被 GWui 焦点系统识别并接收键盘事件。
type termFocusable struct{}

func (f *termFocusable) HandleEvent(e event.Event) bool { return false }
func (f *termFocusable) Focusable() bool                { return true }

// ─── 颜色常量 ────────────────────────────────────────────────
var (
	colText      = ui.Text
	colTextDim   = ui.TextDim
	colEditor    = ui.EditorBg
	colStatusBar = ui.StatusBarBg
	colHover     = ui.HoverBg
	colSide      = ui.SideBg
	colCursor    = "#cccccc"
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
