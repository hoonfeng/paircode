// 终端网格渲染：把 vterm 的 cell 网格自绘成 goui widget（PaintLayer，事件透传）。
// 等宽字体逐格画 背景块 + 字符，按 SGR 配色；块状半透明光标。
// 标准终端集成的「显示侧」——进程侧(pty/ConPTY) + 屏幕模型(vterm) 已就绪，这里把网格画出来。

//go:build windows

package termpanel

import (
	"github.com/user/gou-ide/cmd/companion/core"
	"github.com/user/gou-ide/cmd/companion/ui"
	"github.com/user/gou-ide/cmd/companion/vterm"
	"github.com/user/goui/pkg/canvas"
	"github.com/user/goui/pkg/event"
	"github.com/user/goui/pkg/paint"
	"github.com/user/goui/pkg/types"
)

// keyToVT 把按键事件转成写进 PTY 的 VT 字节：KeyChar→可打印字符；
// KeyDown→Ctrl 组合（控制字符）/ 回车/退格/Tab/Esc/方向键/Home/End/Page 等的 VT 序列。
func keyToVT(ev *event.KeyEvent) []byte {
	if ev.Type() == event.TypeKeyChar {
		if ev.Char >= 0x20 { // 可打印字符（含 Shift 后的）
			return []byte(string(ev.Char))
		}
		return nil
	}
	// KeyDown：Ctrl+字母 → 控制字符（Ctrl+C=0x03 等）
	if ev.Mods&event.ModCtrl != 0 && len(ev.Key) == 1 {
		c := ev.Key[0]
		switch {
		case c >= 'A' && c <= 'Z':
			return []byte{c - 'A' + 1}
		case c >= 'a' && c <= 'z':
			return []byte{c - 'a' + 1}
		}
	}
	switch ev.Key {
	case "Enter", "Return":
		return []byte("\r")
	case "Backspace":
		return []byte{0x7f}
	case "Tab":
		return []byte("\t")
	case "Escape":
		return []byte{0x1b}
	case "Delete":
		return []byte("\x1b[3~")
	case "ArrowUp":
		return []byte("\x1b[A")
	case "ArrowDown":
		return []byte("\x1b[B")
	case "ArrowRight":
		return []byte("\x1b[C")
	case "ArrowLeft":
		return []byte("\x1b[D")
	case "Home":
		return []byte("\x1b[H")
	case "End":
		return []byte("\x1b[F")
	case "PageUp":
		return []byte("\x1b[5~")
	case "PageDown":
		return []byte("\x1b[6~")
	}
	return nil
}

var GridFont = canvas.Font{Family: "Consolas", Size: 13}

// vtColor 把 vterm 颜色转 goui 颜色；默认色用传入的 def（终端前景/背景）。
func vtColor(c vterm.Color, def types.Color) types.Color {
	if c.Default {
		return def
	}
	return types.ColorFromRGB(c.R, c.G, c.B)
}

// termCellSize 当前终端字体下的格宽/行高（等宽）。cvs 可测量则用真实字宽，否则估算。
func termCellSize(cvs canvas.Canvas, font canvas.Font) (cw, ch float64) {
	cw = font.Size * 0.6
	if cvs != nil {
		if w := cvs.MeasureText("M", font).Width; w > 0 {
			cw = w
		}
	}
	return cw, font.Size * 1.45
}

// termGridFontNow 据终端字号设置取等宽字体。
func termGridFontNow() canvas.Font {
	f := GridFont
	if core.Settings.TermFontSize > 0 {
		f.Size = float64(core.Settings.TermFontSize)
	}
	return f
}

// vtSel 选区，组合缓冲坐标（行=scrollback+grid 的绝对行下标，列=列号）；已归一化 (rowA,colA)≤(rowB,colB)。
type vtSel struct{ rowA, colA, rowB, colB int }

// cellInSel 单元 (r,c) 是否落在选区内（标准文本选区：首行从 colA 起、末行到 colB 止、中间整行）。
func cellInSel(r, c int, s *vtSel) bool {
	if r < s.rowA || r > s.rowB {
		return false
	}
	if s.rowA == s.rowB {
		return c >= s.colA && c < s.colB
	}
	if r == s.rowA {
		return c >= s.colA
	}
	if r == s.rowB {
		return c < s.colB
	}
	return true
}

// paintVTGrid 在画布上画 vterm 网格：等宽逐格 背景块 + 选区高亮 + 字符（SGR 配色/粗体）+ 块状半透明光标。
// scrollOff 为回看偏移（0=贴底/实时，>0=向上看历史）；sel 非空则画选区蓝底。
func paintVTGrid(cvs canvas.Canvas, x, y, w, h float64, vt *vterm.Terminal, font canvas.Font, scrollOff int, sel *vtSel) {
	cw, ch := termCellSize(cvs, font)
	cols, rows := vt.Size()
	start := vt.ScrollbackLen() - scrollOff // 视窗顶行在组合缓冲中的下标
	for vr := 0; vr < rows; vr++ {
		cy := y + float64(vr)*ch
		if cy > y+h {
			break
		}
		row := vt.RowAt(start + vr) // 可能为 nil（越界）→ 该行空
		// 第一遍：底色 + 选区高亮先全铺。否则后一个单元的矩形会画在前一个宽字符（CJK 占 2 格）
		// 之上、盖住其右半 → 选中时中文只显示半个。分两遍后字符永远在所有矩形之上。
		for c := 0; c < cols; c++ {
			cx := x + float64(c)*cw
			cell := vterm.Cell{Ch: ' ', FG: vterm.DefaultColor(), BG: vterm.DefaultColor()}
			if c < len(row) {
				cell = row[c]
			}
			if !cell.BG.Default { // 背景块
				bp := paint.DefaultPaint()
				bp.Color = vtColor(cell.BG, *ui.ShellEditor)
				cvs.DrawRect(cx, cy, cw+0.5, ch, bp)
			}
			if sel != nil && cellInSel(start+vr, c, sel) { // 选区高亮底
				sp := paint.DefaultPaint()
				sp.Color = types.ColorFromRGB(38, 79, 120) // 选区蓝
				cvs.DrawRect(cx, cy, cw+0.5, ch, sp)
			}
		}
		// 第二遍：字符（画在所有底/选之上 → 宽字符不被相邻单元的矩形截半）
		for c := 0; c < cols && c < len(row); c++ {
			cell := row[c]
			if cell.Ch == ' ' || cell.Ch == 0 {
				continue
			}
			cx := x + float64(c)*cw
			tp := paint.DefaultPaint()
			tp.Color = vtColor(cell.FG, *ui.ShellText)
			f := font
			if cell.Bold {
				f.Weight = canvas.FontWeightBold
			}
			cvs.DrawText(string(cell.Ch), cx, canvas.BaselineFor(cy, ch, f.Size, canvas.VAlignMiddle), f, tp)
		}
	}
	if scrollOff == 0 { // 仅贴底/实时时画光标
		ccx, ccy := vt.Cursor()
		cur := paint.DefaultPaint()
		sc := *ui.ShellText
		cur.Color = types.ColorFromRGBA(sc.R, sc.G, sc.B, 150)
		cvs.DrawRect(x+float64(ccx)*cw, y+float64(ccy)*ch, cw, ch, cur)
	}
}
