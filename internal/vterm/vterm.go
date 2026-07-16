// Package vterm 终端模拟器的「屏幕模型」：把 PTY 的 VT/ANSI 字节流解析成 cell 网格
// （行×列，每格 字符+前景/背景/粗体），供上层渲染。纯 Go 逻辑、跨平台、可单测。
//
// 覆盖常见序列：可打印字符、\r \n \b \t、CSI 光标移动(A/B/C/D/H/f/G)、擦除(J/K)、
// SGR 颜色与粗体(0/1/22/30-37/40-47/90-97/100-107/39/49/38;5/48;5/38;2/48;2)。
// 未识别序列吃掉不崩。全屏交互(vim 的备用屏/复杂光标)后续再扩。
package vterm

// Color 终端颜色：RGB；Default=true 表示用终端默认前景/背景（渲染方决定具体色）。
type Color struct {
	R, G, B uint8
	Default bool
}

// DefaultColor 默认前景/背景占位。
func DefaultColor() Color { return Color{Default: true} }

// ansi16 标准 16 色调色板（偏 VS Code 暗色风）。
var ansi16 = [16]Color{
	{R: 30, G: 30, B: 30}, {R: 205, G: 49, B: 49}, {R: 13, G: 188, B: 121}, {R: 229, G: 229, B: 16},
	{R: 36, G: 114, B: 200}, {R: 188, G: 63, B: 188}, {R: 17, G: 168, B: 205}, {R: 229, G: 229, B: 229},
	{R: 102, G: 102, B: 102}, {R: 241, G: 76, B: 76}, {R: 35, G: 209, B: 139}, {R: 245, G: 245, B: 67},
	{R: 59, G: 142, B: 234}, {R: 214, G: 112, B: 214}, {R: 41, G: 184, B: 219}, {R: 255, G: 255, B: 255},
}

// Cell 网格一格。
type Cell struct {
	Ch   rune
	FG   Color
	BG   Color
	Bold bool
}

func blankCell() Cell { return Cell{Ch: ' ', FG: DefaultColor(), BG: DefaultColor()} }

func blankRow(cols int) []Cell {
	row := make([]Cell, cols)
	for i := range row {
		row[i] = blankCell()
	}
	return row
}

type parserState int

const (
	stateGround    parserState = iota
	stateEsc                   // 收到 ESC，等后续
	stateCSI                   // ESC [ … 控制序列
	stateString                // OSC/DCS/SOS/PM/APC 字符串型（如设标题），吃到 BEL 或 ST
	stateStringEsc             // 字符串中遇 ESC，等 ST 的 '\'
	stateEscInter              // ESC 中间字节（字符集设计 ( ) * +），再吃一个最终字符
)

// Terminal 屏幕模型 + VT 解析状态机。非并发安全（调用方在单线程喂字节、读网格）。
type Terminal struct {
	cols, rows int
	grid       [][]Cell // [row][col]，当前可视屏
	scroll     [][]Cell // 滚出顶部的历史行
	maxScroll  int
	cx, cy     int // 光标列/行
	fg, bg     Color
	bold       bool

	state    parserState
	params   []int
	curParam int
	priv     bool // CSI 私有前缀 '?'（忽略）
}

// New 建一个 cols×rows 的终端。
func New(cols, rows int) *Terminal {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	t := &Terminal{cols: cols, rows: rows, fg: DefaultColor(), bg: DefaultColor(), maxScroll: 5000}
	t.grid = makeGrid(rows, cols)
	return t
}

func makeGrid(rows, cols int) [][]Cell {
	g := make([][]Cell, rows)
	for r := range g {
		g[r] = blankRow(cols)
	}
	return g
}

// Size 返回列、行。
func (t *Terminal) Size() (cols, rows int) { return t.cols, t.rows }

// Cursor 返回光标列、行。
func (t *Terminal) Cursor() (col, row int) { return t.cx, t.cy }

// Cell 取某格（越界返回空格）。
func (t *Terminal) Cell(row, col int) Cell {
	if row < 0 || row >= t.rows || col < 0 || col >= t.cols {
		return blankCell()
	}
	return t.grid[row][col]
}

// Scrollback 返回滚出顶部的历史行（供上层做回看渲染）。
func (t *Terminal) Scrollback() [][]Cell { return t.scroll }

// ScrollbackLen 滚回历史行数。
func (t *Terminal) ScrollbackLen() int { return len(t.scroll) }

// RowAt 取「组合缓冲」（滚回历史在前、当前屏在后）的第 i 行（0 起）。越界返回 nil。
// 用于回看渲染：i ∈ [0, ScrollbackLen()+rows)。
func (t *Terminal) RowAt(i int) []Cell {
	if i < 0 {
		return nil
	}
	if i < len(t.scroll) {
		return t.scroll[i]
	}
	if gi := i - len(t.scroll); gi < t.rows {
		return t.grid[gi]
	}
	return nil
}

// Write 喂 PTY 输出字节，推进解析状态机更新网格。
func (t *Terminal) Write(p []byte) (int, error) {
	for _, r := range string(p) { // 按 rune（UTF-8 解码）；控制字节都是 ASCII 单 rune
		t.feed(r)
	}
	return len(p), nil
}

func (t *Terminal) feed(r rune) {
	switch t.state {
	case stateGround:
		t.feedGround(r)
	case stateEsc:
		t.feedEsc(r)
	case stateCSI:
		t.feedCSI(r)
	case stateString: // OSC/DCS 等：吃内容直到 BEL 或 ST（ESC \）
		switch r {
		case 0x07:
			t.state = stateGround
		case 0x1b:
			t.state = stateStringEsc
		}
	case stateStringEsc:
		t.state = stateGround // ESC \ = ST，串结束（任何字符都收尾）
	case stateEscInter:
		t.state = stateGround // 字符集设计的最终字符，吃掉
	}
}

// feedEsc 处理 ESC 之后的字节，分派到 CSI / 字符串(OSC/DCS) / 字符集 / 单字符忽略。
func (t *Terminal) feedEsc(r rune) {
	switch r {
	case '[':
		t.state = stateCSI
		t.params = t.params[:0]
		t.curParam = 0
		t.priv = false
	case ']', 'P', 'X', '^', '_': // OSC(设标题等) / DCS / SOS / PM / APC：字符串型
		t.state = stateString
	case '(', ')', '*', '+', '-', '.', '/': // 字符集设计：再吃一个最终字符
		t.state = stateEscInter
	default:
		t.state = stateGround // ESC = / > / \ / 7 / 8 / c 等，单字符，忽略
	}
}

func (t *Terminal) feedGround(r rune) {
	switch r {
	case 0x1b: // ESC
		t.state = stateEsc
	case '\r':
		t.cx = 0
	case '\n', 0x0b, 0x0c: // LF / VT / FF
		t.lineFeed()
	case '\b':
		if t.cx > 0 {
			t.cx--
		}
	case '\t':
		t.cx = ((t.cx / 8) + 1) * 8
		if t.cx > t.cols-1 {
			t.cx = t.cols - 1
		}
	case 0x07: // 响铃，忽略
	default:
		if r < 0x20 {
			return // 其它控制字符忽略
		}
		t.putChar(r)
	}
}

func (t *Terminal) putChar(r rune) {
	w := 1
	if isWide(r) { // CJK 等全角字符占 2 格（东亚宽度）
		w = 2
	}
	if t.cx+w > t.cols { // 放不下 → 折行
		t.cx = 0
		t.lineFeed()
	}
	t.grid[t.cy][t.cx] = Cell{Ch: r, FG: t.fg, BG: t.bg, Bold: t.bold}
	t.cx++
	if w == 2 && t.cx < t.cols {
		// 宽字符续格：占位、配同背景、不画字（渲染时 Ch==0 跳过），防后续字符叠上来。
		t.grid[t.cy][t.cx] = Cell{Ch: 0, FG: t.fg, BG: t.bg, Bold: t.bold}
		t.cx++
	}
}

// isWide 东亚宽度：CJK/假名/谚文/全角符号等占 2 格。
func isWide(r rune) bool {
	return (r >= 0x1100 && r <= 0x115F) || // 谚文字母
		(r >= 0x2E80 && r <= 0x303E) || // CJK 部首/康熙/CJK 标点
		(r >= 0x3041 && r <= 0x33FF) || // 假名/CJK 符号/带圈
		(r >= 0x3400 && r <= 0x4DBF) || // CJK 扩展 A
		(r >= 0x4E00 && r <= 0x9FFF) || // CJK 统一表意
		(r >= 0xA000 && r <= 0xA4CF) || // 彝文
		(r >= 0xAC00 && r <= 0xD7A3) || // 谚文音节
		(r >= 0xF900 && r <= 0xFAFF) || // CJK 兼容
		(r >= 0xFE30 && r <= 0xFE4F) || // CJK 兼容形式
		(r >= 0xFF00 && r <= 0xFF60) || // 全角 ASCII
		(r >= 0xFFE0 && r <= 0xFFE6) || // 全角符号
		(r >= 0x20000 && r <= 0x3FFFD) // CJK 扩展 B+
}

func (t *Terminal) lineFeed() {
	t.cy++
	if t.cy >= t.rows {
		t.cy = t.rows - 1
		t.scrollUp()
	}
}

func (t *Terminal) scrollUp() {
	t.scroll = append(t.scroll, t.grid[0])
	if len(t.scroll) > t.maxScroll {
		t.scroll = t.scroll[len(t.scroll)-t.maxScroll:]
	}
	copy(t.grid, t.grid[1:])
	t.grid[t.rows-1] = blankRow(t.cols)
}

func (t *Terminal) feedCSI(r rune) {
	switch {
	case r >= '0' && r <= '9':
		t.curParam = t.curParam*10 + int(r-'0')
	case r == ';':
		t.params = append(t.params, t.curParam)
		t.curParam = 0
	case r == '?':
		t.priv = true
	case r >= '@' && r <= '~': // 终止字节
		t.params = append(t.params, t.curParam)
		t.curParam = 0
		t.dispatchCSI(r)
		t.state = stateGround
	default:
		t.state = stateGround // 异常，回地面
	}
}

// param 第 i 个参数，缺省/为 0 时用 def（光标移动/定位语义：0 视为默认）。
func (t *Terminal) param(i, def int) int {
	if i < len(t.params) && t.params[i] > 0 {
		return t.params[i]
	}
	return def
}

// raw 第 i 个参数原值（缺省 0），用于 J/K 这类 0 有意义的命令。
func (t *Terminal) raw(i int) int {
	if i < len(t.params) {
		return t.params[i]
	}
	return 0
}

func (t *Terminal) dispatchCSI(cmd rune) {
	if t.priv { // 私有模式（如 ?25h 光标显隐、?1049h 备用屏）暂忽略
		return
	}
	switch cmd {
	case 'm':
		t.sgr()
	case 'H', 'f':
		t.cy = clamp(t.param(0, 1)-1, 0, t.rows-1)
		t.cx = clamp(t.param(1, 1)-1, 0, t.cols-1)
	case 'A':
		t.cy = clamp(t.cy-t.param(0, 1), 0, t.rows-1)
	case 'B':
		t.cy = clamp(t.cy+t.param(0, 1), 0, t.rows-1)
	case 'C':
		t.cx = clamp(t.cx+t.param(0, 1), 0, t.cols-1)
	case 'D':
		t.cx = clamp(t.cx-t.param(0, 1), 0, t.cols-1)
	case 'G':
		t.cx = clamp(t.param(0, 1)-1, 0, t.cols-1)
	case 'd':
		t.cy = clamp(t.param(0, 1)-1, 0, t.rows-1)
	case 'J':
		t.eraseDisplay(t.raw(0))
	case 'K':
		t.eraseLine(t.raw(0))
	case 'X':
		// Erase Characters (ECH)：从光标处擦除 n 个字符，不移动光标
		n := t.param(0, 1)
		row := t.grid[t.cy]
		for c := t.cx; c < t.cx+n && c < t.cols; c++ {
			row[c] = blankCell()
		}
	}
}

func (t *Terminal) eraseDisplay(mode int) {
	switch mode {
	case 0: // 光标到屏末
		t.eraseInLine(0)
		for r := t.cy + 1; r < t.rows; r++ {
			t.grid[r] = blankRow(t.cols)
		}
	case 1: // 屏首到光标
		for r := 0; r < t.cy; r++ {
			t.grid[r] = blankRow(t.cols)
		}
		t.eraseInLine(1)
	case 2, 3: // 整屏
		for r := 0; r < t.rows; r++ {
			t.grid[r] = blankRow(t.cols)
		}
	}
}

func (t *Terminal) eraseLine(mode int) { t.eraseInLine(mode) }

func (t *Terminal) eraseInLine(mode int) {
	row := t.grid[t.cy]
	switch mode {
	case 0: // 光标到行末
		for c := t.cx; c < t.cols; c++ {
			row[c] = blankCell()
		}
	case 1: // 行首到光标
		for c := 0; c <= t.cx && c < t.cols; c++ {
			row[c] = blankCell()
		}
	case 2: // 整行
		for c := 0; c < t.cols; c++ {
			row[c] = blankCell()
		}
	}
}

func (t *Terminal) sgr() {
	for i := 0; i < len(t.params); i++ {
		n := t.params[i]
		switch {
		case n == 0:
			t.fg, t.bg, t.bold = DefaultColor(), DefaultColor(), false
		case n == 1:
			t.bold = true
		case n == 22:
			t.bold = false
		case n >= 30 && n <= 37:
			t.fg = ansi16[n-30]
		case n >= 90 && n <= 97:
			t.fg = ansi16[n-90+8]
		case n == 39:
			t.fg = DefaultColor()
		case n >= 40 && n <= 47:
			t.bg = ansi16[n-40]
		case n >= 100 && n <= 107:
			t.bg = ansi16[n-100+8]
		case n == 49:
			t.bg = DefaultColor()
		case n == 38:
			if c, adv, ok := parseExtColor(t.params[i:]); ok {
				t.fg = c
				i += adv
			}
		case n == 48:
			if c, adv, ok := parseExtColor(t.params[i:]); ok {
				t.bg = c
				i += adv
			}
		}
	}
}

// parseExtColor 解析 38/48 的扩展色：;5;N（256 色）或 ;2;R;G;B（真彩）。返回色、消耗的额外参数数、是否成功。
func parseExtColor(p []int) (Color, int, bool) {
	if len(p) >= 3 && p[1] == 5 {
		return color256(p[2]), 2, true
	}
	if len(p) >= 5 && p[1] == 2 {
		return Color{R: uint8(p[2]), G: uint8(p[3]), B: uint8(p[4])}, 4, true
	}
	return Color{}, 0, false
}

func color256(n int) Color {
	switch {
	case n < 16:
		return ansi16[n&15]
	case n >= 232:
		g := uint8(8 + (n-232)*10)
		return Color{R: g, G: g, B: g}
	default:
		n -= 16
		conv := func(v int) uint8 {
			if v == 0 {
				return 0
			}
			return uint8(55 + v*40)
		}
		return Color{R: conv((n / 36) % 6), G: conv((n / 6) % 6), B: conv(n % 6)}
	}
}

// Resize 改尺寸：新建网格，左上对齐拷贝旧内容，光标钳进范围（不做回流，够用）。
func (t *Terminal) Resize(cols, rows int) {
	if cols < 1 || rows < 1 || (cols == t.cols && rows == t.rows) {
		return
	}
	ng := makeGrid(rows, cols)
	for r := 0; r < rows && r < t.rows; r++ {
		for c := 0; c < cols && c < t.cols; c++ {
			ng[r][c] = t.grid[r][c]
		}
	}
	t.grid = ng
	t.cols, t.rows = cols, rows
	t.cx = clamp(t.cx, 0, cols-1)
	t.cy = clamp(t.cy, 0, rows-1)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
