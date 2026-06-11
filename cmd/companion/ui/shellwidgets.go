//go:build windows

package ui

import (
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// 跨面板共用的小部件 —— 原散在 filetree/git/search 里的泛用 UI helper，下沉到 ui 供各面板共用。

// ShellIconBtn 外壳侧栏的小图标按钮（文件树 / 搜索工具条用）。
func ShellIconBtn(icon string, onClick func()) widget.Widget {
	return &widget.Button{
		Icon: icon, IconSize: 14, TextColor: *ShellTextDim,
		OnClick: onClick, Color: *ShellSide,
		MinWidth: 24, MinHeight: 22,
	}
}

// EmptyState 居中空状态（大图标 + 标题 + 副标题），面板无内容时用。
func EmptyState(icon, title, sub string) widget.Widget {
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "center", JustifyContent: "center", Padding: types.EdgeInsets(20)},
		widget.Lucide(icon, widget.IconSize(28), widget.IconColor(*ShellTextDim)),
		widget.Div(widget.Style{Height: 8}),
		TextC(title, *ShellText, 12),
		widget.Div(widget.Style{Height: 3}),
		TextC(sub, *ShellTextDim, 11),
	)
}

// StyleInput 给输入框套上主题化的搜索框样式（搜索面板 / 对话搜索框共用）。
func StyleInput(in *widget.Input) {
	in.Color = *Fg
	in.CursorColor = *Fg
	in.BGColor = *Bg
	in.BorderColor = *Border
	in.FocusBorderColor = *Accent
	in.HoverBorderColor = *Border
}
