//go:build windows

package ui

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// 排版 helper —— 语义化字号 / 颜色，读 T 令牌。替代 companion 的 label(s, color, size)：
// 调用方说「这是标题 / 弱提示」，而非每处重复颜色与字号常量。

// fontHook 可选界面字体后处理（companion 注入「界面字体」族 + 粗 / 斜 / 下划线）。
var fontHook func(*canvas.Font)

// SetFontHook 注册界面字体钩子（companion 在此接入 UIFont 设置；保持 ui 与 main 解耦）。
func SetFontHook(fn func(*canvas.Font)) { fontHook = fn }

func mkText(s string, c types.Color, size float64, weight canvas.FontWeight) *widget.Text {
	t := widget.NewText(s, c)
	t.Font.Size = size
	if weight != 0 {
		t.Font.Weight = weight
	}
	if fontHook != nil {
		fontHook(&t.Font)
	}
	return t
}

// Text 主文字（13px，Fg 令牌色）。
func Text(s string) *widget.Text { return mkText(s, *Fg, 13, 0) }

// TextC 指定颜色 / 字号的文字 —— 迁移期可直替 companion 的 label(s, color, size)。
func TextC(s string, c types.Color, size float64) *widget.Text { return mkText(s, c, size, 0) }

// Muted 弱文字（11px，FgMuted）—— 说明 / 提示文案常用。
func Muted(s string) *widget.Text { return mkText(s, *FgMuted, 11, 0) }

// Subtle 次文字（12px，FgSubtle）。
func Subtle(s string) *widget.Text { return mkText(s, *FgSubtle, 12, 0) }

// Label 字段标签（12px，Fg 色）。
func Label(s string) *widget.Text { return mkText(s, *Fg, 12, 0) }

// Title 小节标题（12px 加粗，Fg 色）。
func Title(s string) *widget.Text { return mkText(s, *Fg, 12, canvas.FontWeightBold) }

// TextLine 单行文本（过长省略号，MaxLines=1）—— 固定行高列表用；迁移期直替 companion 的 label1(s,color,size)。
func TextLine(s string, c types.Color, size float64) *widget.Text {
	t := mkText(s, c, size, 0)
	t.MaxLines = 1
	return t
}

// Mono 等宽文本（固定 Consolas，不走界面字体钩子）—— diff / 代码对齐用；迁移期直替 monoLabel。
func Mono(s string, c types.Color, size float64) *widget.Text {
	t := widget.NewText(s, c)
	t.Font = canvas.Font{Family: "Consolas", Size: size}
	return t
}
