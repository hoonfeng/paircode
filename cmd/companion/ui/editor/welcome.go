// welcome 代码欢迎页 —— 已打开工作区但未打开文件时，编辑区空状态展示。
//
// 设计意图：
//   - 与 IDE 欢迎页（idewelcome.go）区分：本页只管「工作区已打开、编辑器无标签」的场景
//   - 直接使用 ui 包的主题指针（*types.Color），主题切换时指针指向的颜色就地更新
//   - 显示 Pair 标题、打开文件提示及入口
//   - 参考 source: 伴随式codeagent EditorPanel.tsx 空状态
//
//go:build windows

package editorpanel

import (
	"github.com/user/gou-ide/cmd/companion/ui"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// buildWelcome 构建代码欢迎页（编辑器空状态），直接读取 ui 主题令牌，跟随主题切换。
// 参考 source: 伴随式codeagent EditorPanel.tsx 空状态（Code2 图标 + Pair 标题 + 打开文件按钮）。
func buildWelcome() widget.Widget {
	// code-2 图标色：FgSubtle 30% 半透明
	codeIconColor := *ui.FgSubtle
	codeIconColor.A = 77

	return widget.Div(
		widget.Style{
			FlexDirection:   "column",
			AlignItems:      "center",
			JustifyContent:  "center",
			BackgroundColor: ui.ShellEditor, // 指针 => 主题切换时自动更新
		},
		// code-2 图标（64px，半透明装饰）
		widget.Lucide("code-2", widget.IconSize(64), widget.IconColor(codeIconColor)),
		widget.Div(widget.Style{Height: 16}),
		// 标题
		ui.TextC("Pair", *ui.Fg, 16),
		widget.Div(widget.Style{Height: 6}),
		// 提示文字
		ui.TextC("打开文件开始编辑", *ui.FgSubtle, 12),
		widget.Div(widget.Style{Height: 4}),
		ui.TextC("从左侧资源管理选择 · 拖拽文件 · Ctrl+O", *ui.FgMuted, 11),
		widget.Div(widget.Style{Height: 16}),
		// 打开文件按钮（简洁样式，不带边框图标，对齐参考）
		codeWelcomeBtn("打开文件", OnOpenFile),
	)
}

// codeWelcomeBtn 代码欢迎页的打开文件按钮。
// 使用主题标准按钮样式（DefaultColor/TextColor/HoverColor），确保文字与背景对比度足够；
// 不加图标以保持编辑器空状态的极简风格，尺寸紧凑对齐主题。
func codeWelcomeBtn(label string, onClick func()) widget.Widget {
	if onClick == nil {
		onClick = func() {} // 回调未注入时安全空操作
	}
	th := widget.CurrentTheme()
	return &widget.Button{
		Text:       label,
		FontSize:   12,
		TextColor:  th.Button.TextColor,
		Color:      th.Button.DefaultColor,
		HoverColor: th.Button.HoverColor,
		Padding:    types.EdgeInsetsLTRB(14, 4, 14, 4),
		MinWidth:   80,
		MinHeight:  26,
		OnClick:    onClick,
	}
}
