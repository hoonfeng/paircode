// 终端面板 —— 真终端：ConPTY 伪终端(pty) + VT/ANSI 屏幕模型(vterm) + 可聚焦渲染/输入(TerminalView)。
// 多标签：每标签一个持久 PTY 会话喂 vterm；TerminalView 抓原始按键转 VT 写 PTY、自绘 vterm 网格。
// 线程模型：读协程读 PTY 原始字节进 pending(加锁)；帧泵每帧在 UI 线程把 pending 喂进 vterm(单线程更新网格)
// + 重绘；久无输出停泵省电。键盘/resize/cd 都在 UI 线程。
//
//go:build windows

package termpanel

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/user/gou-ide/cmd/companion/core"
	"github.com/user/gou-ide/cmd/companion/pty"
	"github.com/user/gou-ide/cmd/companion/ui"
	"github.com/user/gou-ide/cmd/companion/vterm"
	"github.com/user/goui/pkg/animation"
	"github.com/user/goui/pkg/canvas"
	"github.com/user/goui/pkg/event"
	"github.com/user/goui/pkg/types"
	"github.com/user/goui/pkg/widget"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const termIdleFrames = 180 // 连续 ~3s 无输出 → 停泵省电；窗口放宽以接住 ping/慢构建等稀疏输出。
// 注：停泵后纯后台输出（无按键）会延到下次交互才刷——根治需读协程跨线程唤醒 UI 循环(PostMessage)，列为后续。

// 多实例终端：theTermMgr 管多个标签，theTerminal 始终指向「当前活动标签」
// （外部 OpenDir/CopyAll/ClearScreen/shell 等照旧用 theTerminal，自动作用于活动标签）。
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

// ─── 多标签管理器 ───────────────────────────────────────────────
type termManager struct {
	widget.BaseState
	tabs   []*terminalState
	active int
}

func newTermManager() *termManager {
	m := &termManager{tabs: []*terminalState{NewState()}}
	theTerminal = m.tabs[0]
	return m
}

// NewTabWithShell 用指定 shell 新建标签（继承当前目录），切到它。供菜单「新建 X 终端」。
func (m *termManager) NewTabWithShell(code string) {
	t := NewState()
	t.shell = code
	if cur := m.tabs[m.active]; cur != nil {
		t.cwd = cur.cwd
	}
	m.tabs = append(m.tabs, t)
	m.active = len(m.tabs) - 1
	theTerminal = t
	m.SetState()
}

// SetActiveShell 把当前标签换成指定 shell：杀旧 PTY、下帧用新 shell 重启；重画标签栏（shell 标签变）。
func (m *termManager) SetActiveShell(code string) {
	t := m.tabs[m.active]
	if t == nil || t.shell == code {
		return
	}
	t.shell = code
	t.killPTY()
	m.SetState()
}

// switchTab 切到第 i 个标签。
func (m *termManager) switchTab(i int) {
	if i < 0 || i >= len(m.tabs) || i == m.active {
		return
	}
	m.active = i
	theTerminal = m.tabs[i]
	m.SetState()
}

// closeTab 关闭第 i 个标签（至少留一个），杀掉它的 PTY。
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
	m.SetState()
}

func (m *termManager) Build(ctx widget.BuildContext) widget.Widget {
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch", BackgroundColor: ui.ShellEditor},
		m.tabBar(),
		ui.Expand(&termInstance{st: m.tabs[m.active]}),
	)
}

// tabBar 顶部标签栏：每标签 shell 徽标 + 目录名 + 关闭×；末尾「+」新建。
func (m *termManager) tabBar() widget.Widget {
	kids := make([]widget.Widget, 0, len(m.tabs)+1)
	for i, t := range m.tabs {
		idx := i
		bg, tc := ui.ShellStatusBar, *ui.ShellTextDim
		if i == m.active {
			bg, tc = ui.ShellEditor, *ui.ShellText
		}
		kids = append(kids, &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
				widget.Style{BackgroundColor: bg, Padding: types.EdgeInsetsLTRB(10, 5, 10, 5)},
				ui.TextC(shellLabel(t.shell), tc, 11),
			)},
			OnClick: func() { m.switchTab(idx) },
		})
		if len(m.tabs) > 1 {
			kids = append(kids, &widget.Clickable{
				SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
					widget.Style{BackgroundColor: bg, Padding: types.EdgeInsetsLTRB(0, 5, 8, 5)},
					widget.Lucide("x", widget.IconSize(11), widget.IconColor(*ui.ShellTextDim)),
				)},
				OnClick: func() { m.closeTab(idx) },
			})
		}
	}
	// 「+」改成下拉：列出本机探测到的 shell，选哪个就用哪个 shell 新建标签（之前只能 cmd、不能选）。
	plusItems := make([]widget.DropdownItem, 0, 3)
	for _, sh := range AvailableShells() {
		plusItems = append(plusItems, widget.DropdownItem{Label: "新建 " + sh.Label, Command: sh.Code})
	}
	// 触发器用 Div（非 Button）：Button 默认强加 minHeight=32 > 标签栏 28，会把整行顶到 32、加号偏下；
	// Div 跟标签同结构、随 stretch 填满并居中图标（Dropdown 自身拦截点击，触发器无需可点）。
	plusTrigger := widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(8, 5, 8, 5)},
		widget.Lucide("plus", widget.IconSize(13), widget.IconColor(*ui.ShellStatus)))
	kids = append(kids, widget.NewDropdown(plusTrigger, plusItems...).
		WithOnCommand(func(code string) { m.NewTabWithShell(code) }).
		WithPlacement(widget.PlacementBottomStart))
	// AlignItems:stretch —— 让每个标签与它的关闭× 都填满标签栏高(28)，两者等高（之前 center 下
	// 标签名 Div 24px、× Div 21px 高度不一致）。
	return widget.Div(
		widget.Style{FlexDirection: "row", AlignItems: "stretch", BackgroundColor: ui.ShellStatusBar, Height: 28},
		kids,
	)
}

// termInstance 把某个终端标签作为子 StatefulWidget 挂载（活动标签才挂载渲染；
// 切换=挂另一标签；后台标签的 vt/读协程仍在其 *terminalState 里活着，切回即见）。
type termInstance struct {
	widget.StatefulWidget
	st *terminalState
}

func (w *termInstance) CreateState() widget.State { return w.st }

// shellLabel 当前 shell 的标签。
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

// ptyShellFor 把内部 shell 码（cmd/powershell/gitbash）映射到探测到的 pty.Shell。
func ptyShellFor(code string) pty.Shell {
	switch code {
	case "cmd":
		return pty.ShellByName("CMD")
	case "powershell":
		if s := pty.ShellByName("PowerShell"); s.Name == "PowerShell" {
			return s // Windows PowerShell
		}
		return pty.ShellByName("PowerShell 7") // 仅装了 pwsh 时退回它
	case "gitbash":
		return pty.ShellByName("Git Bash")
	}
	return pty.DefaultShell()
}

// ShellOpt 一个可选 shell（菜单用；字段导出供 main 的「+」下拉读）。
type ShellOpt struct{ Code, Label string }

// AvailableShells 本机探测到的、受支持的 shell（CMD/PowerShell/Bash），去重，供菜单列出。
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

// ─── 单个终端标签（PTY + vterm）───────────────────────────────────
type terminalState struct {
	widget.BaseState
	mu         sync.Mutex            // 护 pending / alive / sess / idleFrames / pump
	vt         *vterm.Terminal       // 屏幕模型（仅 UI 线程读写）
	sess       pty.PTY               // 伪终端会话
	pending    []byte                // 读协程写、帧泵取（原始 PTY 字节）
	alive      bool                  // PTY 在跑
	failed     bool                  // 启动失败（不再反复重试）
	shell      string                // cmd / powershell / gitbash
	cwd        string                // 初始/cd 目录
	cols, rows int                   // 当前终端列/行
	idleFrames int                   // 连续无输出帧数
	scrollOff  int                   // 回看偏移（0=贴底/实时，>0=向上看历史；仅 UI 线程）
	pump       *animation.Controller // 帧泵
	decCarry   []byte                // GBK 流式解码残留的不完整尾字节（仅 UI 线程/drain）
	// 鼠标拖拽选区（仅 UI 线程）：锚点/游标用组合缓冲绝对行 + 列；cellW/H 为最近一帧格宽/行高（供坐标映射）。
	selecting    bool
	hasSel       bool
	selAR, selAC int
	selCR, selCC int
	cellW, cellH float64
}

// Panel 终端面板组件。
type Panel struct{ widget.StatefulWidget }

func (t *Panel) CreateState() widget.State { return theTermMgr }

func (t *terminalState) Build(ctx widget.BuildContext) widget.Widget {
	tv := &widget.TerminalView{
		OnPaint: func(cvs canvas.Canvas, x, y, w, h float64) {
			font := termGridFontNow()
			cw, ch := termCellSize(cvs, font)
			t.cellW, t.cellH = cw, ch // 存格宽/行高，供鼠标坐标→单元映射
			cols, rows := 1, 1
			if cw > 0 {
				cols = int(w / cw)
			}
			if ch > 0 {
				rows = int(h / ch)
			}
			if cols < 1 {
				cols = 1
			}
			if rows < 1 {
				rows = 1
			}
			t.ensurePTY(cols, rows)
			t.resizeTo(cols, rows)
			paintVTGrid(cvs, x, y, w, h, t.vt, font, t.scrollOff, t.normSel())
		},
		OnKey:       t.handleKey,
		OnWheel:     t.handleWheel,
		OnMouseDown: t.onSelDown,
		OnMouseDrag: t.onSelDrag,
		OnMouseUp:   t.onSelUp,
	}
	return &widget.ContextArea{ // 右键：终端菜单（复制全部/粘贴/添加到对话/清屏/切 shell）
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{BackgroundColor: ui.ShellEditor, FlexDirection: "column", AlignItems: "stretch"},
			ui.Expand(tv),
		)},
		OnContextMenu: func(x, y float64) {
			if OnContextMenu != nil { // 右键菜单由 main 注入(菜单含「添加到对话」耦合 chat，故不在本包)
				OnContextMenu(x, y)
			}
		},
	}
}

// ─── 鼠标拖拽选区（仅 UI 线程）──────────────────────────────────

// cellAt 把终端内局部坐标(px)换算成「组合缓冲绝对行 + 列」。
func (t *terminalState) cellAt(localX, localY float64) (row, col int) {
	if t.cellW <= 0 || t.cellH <= 0 {
		return 0, 0
	}
	col = int(localX / t.cellW)
	if col < 0 {
		col = 0
	}
	vr := int(localY / t.cellH) // 视窗内可见行
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
	widget.OnNeedsRepaint()
}

func (t *terminalState) onSelDrag(x, y float64) {
	if !t.selecting {
		return
	}
	t.selCR, t.selCC = t.cellAt(x, y)
	t.hasSel = t.selCR != t.selAR || t.selCC != t.selAC
	widget.OnNeedsRepaint()
}

func (t *terminalState) onSelUp(x, y float64) {
	t.selecting = false
	if t.hasSel { // 拖选完即复制（PuTTY/xterm 风格：选 = 复制；右键可粘贴）
		if widget.ClipboardWrite != nil {
			widget.ClipboardWrite(t.copySelection())
		}
	}
	widget.OnNeedsRepaint()
}

// normSel 当前选区归一化成 (rowA,colA)≤(rowB,colB)；无选区返回 nil。
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

// copySelection 提取选区文本（首行从 colA、末行到 colB、中间整行；去行尾空格、行间换行）。
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

// ensurePTY 懒启动伪终端（首帧拿到真实尺寸时）。
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
		widget.OnNeedsRepaint()
		return
	}
	t.vt = vterm.New(cols, rows)
	t.mu.Lock()
	t.sess, t.alive, t.cols, t.rows = sess, true, cols, rows
	t.mu.Unlock()
	go t.reader(sess)
	t.startPump()
}

// reader 持续读 PTY 原始字节进 pending（读协程，会话存活期间常驻）。
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

// drain 每帧把 pending 喂进 vterm（UI 线程）+ 重绘；久无输出停泵。
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
		data = t.decodeOutput(data) // 按编码设置转码（gbk→utf-8；auto/utf-8 原样）
		before := t.vt.ScrollbackLen()
		t.vt.Write(data)     // UI 线程更新网格
		if t.scrollOff > 0 { // 用户在看历史：随新增滚回行上移，保持视图内容稳定（不被新输出顶走）
			t.scrollOff += t.vt.ScrollbackLen() - before
			if mx := t.vt.ScrollbackLen(); t.scrollOff > mx {
				t.scrollOff = mx
			}
		}
		if widget.OnNeedsRepaint != nil {
			widget.OnNeedsRepaint() // 仅重绘（网格尺寸未变，无需 relayout）
		}
	}
	if idle {
		t.stopPump()
	}
}

// handleWheel 滚轮回看：上滚（deltaY>0）看历史、下滚回贴底。每格滚 3 行。
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
		if widget.OnNeedsRepaint != nil {
			widget.OnNeedsRepaint()
		}
	}
}

// handleKey 按键 → VT 字节 → 写 PTY（起泵接住 shell 响应）。
func (t *terminalState) handleKey(ev *event.KeyEvent) {
	data := keyToVT(ev)
	if len(data) == 0 {
		return
	}
	t.scrollOff = 0 // 输入 → 回到贴底/实时
	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	if sess != nil {
		sess.Write(data)
		t.startPump()
	}
}

// resizeTo 面板尺寸变 → 同步 vterm + PTY（伪控制台据此重排）。
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
	if t.pump != nil {
		t.mu.Unlock()
		return
	}
	p := animation.NewController(time.Second, animation.Linear)
	p.Repeat = true
	p.OnUpdate = func(float64) { t.drain() }
	t.pump = p
	t.mu.Unlock()
	p.Start()
}

func (t *terminalState) stopPump() {
	t.mu.Lock()
	p := t.pump
	t.pump = nil
	t.mu.Unlock()
	if p != nil {
		p.Stop()
	}
}

// killPTY 杀掉本标签的 PTY 会话（关闭标签 / 切 shell 时）。
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

// ─── 右键菜单动作（UI 线程）────────────────────────────────

// OpenDir 把终端切到某目录（文件树「在终端打开」）：cd 当前 shell；未起则记为初始目录。
func (t *terminalState) OpenDir(dir string) {
	t.cwd = dir
	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	if sess != nil {
		sess.Write([]byte("cd /d \"" + dir + "\"\r")) // cmd 用 cd /d；其它 shell 多余的 /d 会忽略或报小错
		t.startPump()
	}
	t.SetState()
}

// CopyAll 取屏幕全部文本（按行，去尾空白）。
func (t *terminalState) CopyAll() string {
	cols, rows := t.vt.Size()
	var b strings.Builder
	for r := 0; r < rows; r++ {
		line := make([]rune, 0, cols)
		for c := 0; c < cols; c++ {
			if ch := t.vt.Cell(r, c).Ch; ch != 0 { // 跳过宽字符续格占位
				line = append(line, ch)
			}
		}
		b.WriteString(strings.TrimRight(string(line), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

// PasteToInput 粘贴：把剪贴板内容写进 PTY（真终端=直接送给 shell）。
func (t *terminalState) PasteToInput() {
	if widget.ClipboardRead == nil {
		return
	}
	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	if sess != nil {
		sess.Write([]byte(widget.ClipboardRead()))
		t.startPump()
	}
}

// ClearScreen 清屏：发 shell 的清屏命令（cmd→cls / 其它→clear）。
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
		if widget.OnNeedsRepaint != nil {
			widget.OnNeedsRepaint()
		}
	}
}

// ─── 输出编码（设置→终端→编码）─────────────────────────────

// TermEncodingLabel 状态栏显示的终端编码（auto→UTF-8，ConPTY 默认输出 UTF-8）。
func TermEncodingLabel() string {
	if core.Settings.TermEncoding == "gbk" {
		return "GBK"
	}
	return "UTF-8"
}

// decodeOutput 按编码设置把 PTY 原始字节转为 UTF-8 供 vterm（仅 UI 线程/drain）。
// auto/utf-8：原样（ConPTY 默认 UTF-8）；gbk：GBK→UTF-8 流式解码，末尾不完整多字节序列留待下帧补全。
func (t *terminalState) decodeOutput(b []byte) []byte {
	if core.Settings.TermEncoding != "gbk" {
		return b
	}
	src := b
	if len(t.decCarry) > 0 {
		src = append(t.decCarry, b...) // 接上次残留的不完整尾字节
		t.decCarry = nil
	}
	out, carry := gbkToUTF8(src)
	t.decCarry = carry
	return out
}

// gbkToUTF8 把 GBK 字节解码为 UTF-8；返回已解码部分 + 末尾不完整序列（留待下次补全）。
// 非法字节：跳过 1 字节后续帧再试（防卡死）。dst 取 2x+16 足够（GBK 双字节→UTF-8 至多三字节）。
func gbkToUTF8(b []byte) (out, carry []byte) {
	dst := make([]byte, len(b)*2+16)
	nDst, nSrc, err := simplifiedchinese.GBK.NewDecoder().Transform(dst, b, false)
	out = append([]byte(nil), dst[:nDst]...)
	rest := b[nSrc:]
	switch {
	case err == transform.ErrShortSrc: // 尾部多字节不完整 → 整段留作残留
		carry = append([]byte(nil), rest...)
	case err != nil && len(rest) > 0: // 非法字节 → 跳过 1 字节，其余下帧再试
		carry = append([]byte(nil), rest[1:]...)
	}
	return
}

// Area 终端面板入口（panelBody 调用）。
func Area() widget.Widget { return &Panel{} }
