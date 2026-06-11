//go:build windows

package ui

import (
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// 表单控件 —— 主题化的 Field / Toggle / Pill / Input / Select / Slider。
// 全部读令牌，换肤即时；替代 settings.go 散落的 settingsField/settingsToggle/... 私有 helper。

// Field 一行表单项：标签 + 控件（竖排，顶部留距）。
func Field(label string, control widget.Widget) widget.Widget {
	return Box(widget.Style{FlexDirection: "column", AlignItems: "stretch", Padding: types.EdgeInsetsLTRB(0, 12, 0, 0)},
		Label(label), VGap(5), control,
	)
}

// Toggle 一行开关：左标签弹性 + 右「开 / 关」按钮（开 = 强调实心）。
func Toggle(label string, on bool, toggle func()) widget.Widget {
	state, fg, bg := "关", *Fg, *BgMuted
	if on {
		state, fg, bg = "开", *OnAccent, *AccentStrong
	}
	return Box(widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(0, 12, 0, 0)},
		Expand(Label(label)),
		btn(state, toggle, bg, fg, *BgHover, BtnOpts{Size: SizeSm, MinW: 44}),
	)
}

// Pill 启用态小药丸（on = 成功绿底白字 / off = 三级面弱字），可点切换。用于启用/禁用资源（MCP/Skills）。
func Pill(onText, offText string, on bool, toggle func()) widget.Widget {
	t, fg, bg := offText, *FgMuted, *BgMuted
	if on {
		t, fg, bg = onText, *OnAccent, *Success
	}
	return btn(t, toggle, bg, fg, bg, BtnOpts{Size: SizeXs, MinW: 40})
}

// AccentPill 选择态小药丸（on = 强调蓝底白字 / off = 三级面弱字），可点切换。用于「已选/未选」类选择（哲学经典）。
func AccentPill(onText, offText string, on bool, toggle func()) widget.Widget {
	t, fg, bg := offText, *FgMuted, *BgMuted
	if on {
		t, fg, bg = onText, *OnAccent, *AccentStrong
	}
	return btn(t, toggle, bg, fg, bg, BtnOpts{Size: SizeXs, MinW: 44})
}

// Input 主题化单行输入框（深色面 + 焦点描强调边）。
func Input(placeholder, value string, resetTok int, onChange func(string)) *widget.Input {
	in := widget.NewInput(placeholder, onChange)
	in.Text = value
	in.ResetToken = resetTok
	in.Color = *Fg
	in.CursorColor = *Fg
	in.PlaceholderColor = *FgMuted
	in.BGColor = *Bg
	in.BorderColor = *Border
	in.FocusBorderColor = *Accent
	in.HoverBorderColor = *Border
	return in
}

// Textarea 主题化多行文本框（自动换行；底层亦为 Input，rows 控制行数）。
func Textarea(placeholder, value string, rows, resetTok int, onChange func(string)) *widget.Input {
	ta := widget.NewTextarea(placeholder, rows, onChange)
	ta.Text = value
	ta.ResetToken = resetTok
	ta.Wrap = true
	ta.Font.Size = 13
	ta.Color = *Fg
	ta.CursorColor = *Fg
	ta.PlaceholderColor = *FgMuted
	ta.BGColor = *Bg
	ta.BorderColor = *Border
	ta.FocusBorderColor = *Accent
	ta.HoverBorderColor = *Border
	return ta
}

// Select 主题化下拉（深色由包级 SetSelectTheme 统一，下拉已封顶+可滚动）。
func Select(value string, opts []widget.SelectOption, width float64, onChange func(string)) *widget.Select {
	return widget.NewSelect(opts).WithValue(value).WithWidth(width).WithOnChanged(onChange)
}

// Slider 主题化滑块（强调色轨 / 滑块）。
func Slider(value, min, max, step float64, onChange func(float64)) widget.Widget {
	sl := widget.NewSlider(value, onChange).WithRange(min, max).WithStep(step)
	sl.ActiveColor, sl.ThumbColor = *AccentStrong, *AccentStrong
	sl.InactiveColor = *BgMuted
	sl.LabelColor = *Fg
	return sl
}

// SliderField 滑块 + 标签（标签含实时值）一体。
func SliderField(label string, value, min, max, step float64, onChange func(float64)) widget.Widget {
	return Field(label, Slider(value, min, max, step, onChange))
}
