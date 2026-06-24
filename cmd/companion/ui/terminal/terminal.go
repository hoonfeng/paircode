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
	"strings"
	"sync"
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

	tabs   []*terminalState
	active int
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
	theTermMgr.rootEl = doc.CreateElement("div")
	theTermMgr.rootEl.SetAttribute("style", "display:flex;flex-direction:column;flex:1;background:"+colEditor+";overflow:hidden;")

	title := doc.CreateElement("div")
	title.ClassList().Add("panel-header")
	title.SetTextContent("TERMINAL")
	theTermMgr.rootEl.AppendChild(title)

	// 标签栏
	theTermMgr.tabBarEl = doc.CreateElement("div")
	theTermMgr.tabBarEl.SetAttribute("style", "display:flex;flex-direction:row;align-items:stretch;background:"+colStatusBar+";height:28px;flex-shrink:0;overflow:hidden;")
	theTermMgr.rootEl.AppendChild(theTermMgr.tabBarEl)

	// 终端区（可聚焦，接收键盘输入）
	theTermMgr.termEl = doc.CreateElement("div")
	theTermMgr.termEl.SetAttribute("tabindex", "0")
	theTermMgr.termEl.SetAttribute("style", "flex:1;background:"+colEditor+";overflow:hidden;position:relative;outline:none;")
	theTermMgr.rootEl.AppendChild(theTermMgr.termEl)

	theTermMgr.renderTabBar()
	theTermMgr.renderActiveTerm()

	// 键盘事件
	on(theTermMgr.termEl, event.KeyDown, func(e event.Event) bool {
		ke, ok := e.(*event.KeyboardEvent)
		if !ok || theTerminal == nil {
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
	gridEl.SetAttribute("style", "position:absolute;top:0;left:0;right:0;bottom:0;padding:4px;font-family:Consolas,'Courier New',monospace;font-size:14px;line-height:1.2;color:"+colText+";background:"+colEditor+";overflow:hidden;white-space:pre;")

	t.gridEl = gridEl
	t.renderGrid()

	m.termEl.AppendChild(gridEl)
}

// ─── 单终端状态 ─────────────────────────────────────────────

type terminalState struct {
	gridEl *dom.Element

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
			t.mu.Lock()
			t.pending = append(t.pending, buf[:n]...)
			t.mu.Unlock()
		}
		if err != nil {
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
		data = t.decodeOutput(data)
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
	startRow := scrLen - rows - t.scrollOff
	if startRow < 0 {
		startRow = 0
	}

	var b strings.Builder
	for r := startRow; r < startRow+rows && r < scrLen; r++ {
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
	t.gridEl.SetTextContent(b.String())
}

// ─── 按键 → VT ─────────────────────────────────────────────

func (t *terminalState) handleKey(ev *event.KeyboardEvent) {
	data := keyToVT(ev)
	if len(data) == 0 {
		return
	}
	t.scrollOff = 0
	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	if sess != nil {
		sess.Write(data)
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
			ch := byte(ev.Char)
			if ev.Ctrl {
				if ch >= 'a' && ch <= 'z' {
					return []byte{ch - 'a' + 1}
				}
				if ch >= 'A' && ch <= 'Z' {
					return []byte{ch - 'A' + 1}
				}
			}
			return []byte{ch}
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
		return
	}
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
	if core.Settings.TermEncoding != "gbk" {
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
