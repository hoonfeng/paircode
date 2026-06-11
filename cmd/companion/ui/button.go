//go:build windows

package ui

import (
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// 按钮 —— 语义变体（主 / 次 / 幽灵 / 危险），读令牌着色。
// 替代 companion 满屏的 &widget.Button{SingleChildWidget:{Child:label(...)}, Color:..., ...}。
// 默认用直函数（PrimaryBtn/Btn/…）；要图标 / 尺寸 / 最小宽时用 *X 变体传 BtnOpts。

// BtnSize 按钮尺寸档。
type BtnSize int

const (
	SizeMd BtnSize = iota // 默认（高 26）
	SizeSm                // 小（高 22）
	SizeXs                // 极小（高 20）
)

// BtnOpts 按钮可选项（图标 / 尺寸 / 最小宽）。
type BtnOpts struct {
	Icon string
	Size BtnSize
	MinW float64
}

func btn(label string, onClick func(), bg, fg, hover types.Color, o BtnOpts) widget.Widget {
	h, fs, padX := 26.0, 12.0, 12.0
	switch o.Size {
	case SizeSm:
		h, fs, padX = 22, 11, 10
	case SizeXs:
		h, fs, padX = 20, 10, 8
	}
	b := &widget.Button{
		OnClick: onClick, Color: bg, HoverColor: hover, MinHeight: h, MinWidth: o.MinW,
		Padding: types.EdgeInsetsLTRB(padX, 0, padX, 0), BorderRadius: Radius - 2,
	}
	switch {
	case o.Icon != "" && label == "": // 纯图标按钮
		b.Icon, b.IconSize, b.IconColor = o.Icon, fs+2, &fg
	case o.Icon != "": // 图标 + 文字（走 Button 原生组合）
		b.Icon, b.IconSize, b.IconColor = o.Icon, fs+1, &fg
		b.Text, b.TextColor, b.FontSize = label, fg, fs
	default: // 纯文字：用子 Text，使界面字体钩子（applyUIFont）也作用于按钮文字
		b.Child = mkText(label, fg, fs, 0)
	}
	return b
}

// PrimaryBtn 主按钮（强调实心底 + 白字）。
func PrimaryBtn(label string, onClick func()) widget.Widget {
	return btn(label, onClick, *AccentStrong, *OnAccent, *Accent, BtnOpts{})
}

// Btn 次级按钮（三级面底 + 主文字）。
func Btn(label string, onClick func()) widget.Widget {
	return btn(label, onClick, *BgMuted, *Fg, *BgHover, BtnOpts{})
}

// GhostBtn 幽灵按钮（透明底 + 次文字，悬停显面）。
func GhostBtn(label string, onClick func()) widget.Widget {
	return btn(label, onClick, types.Color{}, *FgSubtle, *BgHover, BtnOpts{})
}

// DangerBtn 危险按钮（三级面底 + 危险色字）。
func DangerBtn(label string, onClick func()) widget.Widget {
	return btn(label, onClick, *BgMuted, *Danger, *BgHover, BtnOpts{})
}

// SuccessBtn 成功实心按钮（成功绿底 + 白字）—— 用于「允许/确认放行」类正向动作。
func SuccessBtn(label string, onClick func()) widget.Widget {
	return btn(label, onClick, *Success, *OnAccent, *Success, BtnOpts{})
}

// SolidDangerBtn 危险实心按钮（危险红底 + 白字）—— 用于「拒绝/停止」类高危动作。
func SolidDangerBtn(label string, onClick func()) widget.Widget {
	return btn(label, onClick, *Danger, *OnAccent, *Danger, BtnOpts{})
}

// *X 变体：带可选项（图标 / 尺寸 / 最小宽）。
func PrimaryBtnX(label string, onClick func(), o BtnOpts) widget.Widget {
	return btn(label, onClick, *AccentStrong, *OnAccent, *Accent, o)
}
func BtnX(label string, onClick func(), o BtnOpts) widget.Widget {
	return btn(label, onClick, *BgMuted, *Fg, *BgHover, o)
}
func GhostBtnX(label string, onClick func(), o BtnOpts) widget.Widget {
	return btn(label, onClick, types.Color{}, *FgSubtle, *BgHover, o)
}
func DangerBtnX(label string, onClick func(), o BtnOpts) widget.Widget {
	return btn(label, onClick, *BgMuted, *Danger, *BgHover, o)
}
func SuccessBtnX(label string, onClick func(), o BtnOpts) widget.Widget {
	return btn(label, onClick, *Success, *OnAccent, *Success, o)
}
func SolidDangerBtnX(label string, onClick func(), o BtnOpts) widget.Widget {
	return btn(label, onClick, *Danger, *OnAccent, *Danger, o)
}

// IconBtn 纯图标幽灵按钮（工具栏 / 关闭 ×）。
func IconBtn(icon string, onClick func()) widget.Widget {
	return btn("", onClick, types.Color{}, *FgSubtle, *BgHover, BtnOpts{Icon: icon, Size: SizeSm})
}
