//go:build windows

package ui

import (
	"strconv"

	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// FontPickerOpts 字体选择器配置（全配置化：候选字体与回调都由调用方提供，ui 不碰平台字体枚举）。
type FontPickerOpts struct {
	Fonts        []string // 候选字体族（编辑器=等宽候选 / 界面=全部已装；调用方决定）
	Current      string   // 当前字体族（空 + DefaultLabel 非空 → 选中「默认」项）
	DefaultLabel string   // 非空：下拉首项为该「默认」标签（值 ""），供「跟随默认」
	Width        float64  // 下拉宽（0=180）
	ResetTok     int      // 预设刷新令牌（保留，未来受控刷新用）

	Size      int // 当前字号（OnSize!=nil 才显步进器；<=0 显示为默认 14）
	Bold      bool
	Italic    bool
	Underline bool

	OnFont      func(string)
	OnSize      func(int) // nil=不显字号步进（界面字体字号各处自定）
	OnBold      func()
	OnItalic    func()
	OnUnderline func()
}

// FontPicker 封装的字体设置组件：字体族下拉(可搜索·已封顶滚动) + 字号步进 + 粗/斜/下划线开关。
// 替代旧写法（直接甩个无限高 select 撑爆窗口 + 散在面板里的 fontControl）。
func FontPicker(o FontPickerOpts) widget.Widget {
	w := o.Width
	if w <= 0 {
		w = 180
	}
	opts := make([]widget.SelectOption, 0, len(o.Fonts)+1)
	if o.DefaultLabel != "" {
		opts = append(opts, widget.SelectOption{Label: o.DefaultLabel, Value: ""})
	}
	for _, f := range o.Fonts {
		opts = append(opts, widget.SelectOption{Label: f, Value: f})
	}
	sel := widget.NewSelect(opts).WithValue(o.Current).WithWidth(w).WithFilterable(true).WithOnChanged(o.OnFont)

	kids := []widget.Widget{Expand(sel)}
	if o.OnSize != nil {
		kids = append(kids, HGap(8), sizeStepper(o.Size, o.OnSize))
	}
	kids = append(kids,
		HGap(8), styleToggle("B", o.Bold, o.OnBold),
		HGap(5), styleToggle("I", o.Italic, o.OnItalic),
		HGap(5), styleToggle("U", o.Underline, o.OnUnderline),
	)
	return Row(kids...)
}

// sizeStepper 字号步进器：[−] N px [+]，夹在 8~40（<=0 显示默认 14）。
func sizeStepper(cur int, set func(int)) widget.Widget {
	if cur <= 0 {
		cur = 14
	}
	clampSize := func(v int) int {
		if v < 8 {
			return 8
		}
		if v > 40 {
			return 40
		}
		return v
	}
	return Box(widget.Style{FlexDirection: "row", AlignItems: "center", BackgroundColor: BgMuted,
		BorderColor: Border, BorderWidth: 1, BorderRadius: Radius - 1},
		stepBtn("−", func() { set(clampSize(cur - 1)) }),
		Box(widget.Style{Width: 42, FlexDirection: "row", AlignItems: "center"}, Expand(TextC("  "+strconv.Itoa(cur)+"px", *Fg, 12))),
		stepBtn("+", func() { set(clampSize(cur + 1)) }),
	)
}

// stepBtn 步进器加减按钮（字号略大便于点）。用子 Text，避免无 child 文本按钮的 64 默认最小宽撑宽。
func stepBtn(text string, onClick func()) widget.Widget {
	return &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: mkText(text, *Fg, 15, 0)},
		OnClick:           onClick,
		Color:             *BgMuted, HoverColor: *BgHover, MinHeight: 28, Padding: types.EdgeInsetsLTRB(10, 0, 10, 0),
	}
}

// styleToggle 粗 / 斜 / 下划线开关按钮：激活 = 强调实心底白字。用子 Text 保持紧凑（同 stepBtn 理由）。
func styleToggle(text string, active bool, onClick func()) widget.Widget {
	bg, fg := *BgMuted, *Fg
	if active {
		bg, fg = *AccentStrong, *OnAccent
	}
	return &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: mkText(text, fg, 12, 0)},
		OnClick:           onClick,
		Color:             bg, HoverColor: *BgHover, MinHeight: 28, Padding: types.EdgeInsetsLTRB(11, 0, 11, 0),
	}
}
