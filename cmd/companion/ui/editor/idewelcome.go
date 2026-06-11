// idewelcome IDE 欢迎页 —— 未打开工作区/项目时，中列整体展示。
//
// 设计意图：
//   - 与代码欢迎页（welcome.go）区分：本页在「无工作区」时展示，属 IDE 级空状态
//   - 提供「打开文件夹」、「新建项目」入口
//   - 显示最近项目列表，支持快速打开
//   - 参考 source: 伴随式codeagent WelcomePage.tsx
//
//go:build windows

package editorpanel

import (
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/paircode/cmd/companion/ui/logo"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// buildIdeWelcome 构建 IDE 欢迎页。直接读取 ui 主题令牌，跟随主题切换。
func buildIdeWelcome() widget.Widget {
	return widget.Div(
		widget.Style{
			FlexDirection:   "column",
			AlignItems:      "center",
			BackgroundColor: ui.ShellEditor,
		},
		// 顶部弹性留白
		spacerF(1),
		// Logo
		logo.Big(),
		ui.VGap(16),
		// 标题：Pair（用强调色突出，近似参考渐变效果）
		ui.TextC("Pair", *ui.Accent, 28),
		ui.VGap(6),
		// 副标题
		ui.TextC("AI-powered coding agent — 智能多 Agent 协作编程", *ui.FgSubtle, 12),
		ui.VGap(28),
		// 操作按钮行
		ui.RowG(12,
			ideActionBtn("打开文件夹", OnOpenFolder, "folder-open"),
			ideActionBtn("新建项目", OnNewProject, "sparkles"),
		),
		ui.VGap(32),
		// 最近项目
		ideRecentSection(),
		// 快捷键提示
		ui.VGap(24),
		ui.TextC("Ctrl+O 打开文件  ·  Ctrl+K Ctrl+O 打开文件夹  ·  Ctrl+N 新建文件", *ui.FgMuted, 11),
		// 底部弹性留白
		spacerF(2),
	)
}

// ── 操作按钮 ──

// ideActionBtn IDE 欢迎页的操作按钮（带图标，紧凑尺寸，使用主题标准样式）。
func ideActionBtn(label string, onClick func(), icon string) widget.Widget {
	if onClick == nil {
		onClick = func() {}
	}
	th := widget.CurrentTheme()
	return &widget.Button{
		Text:       label,
		Icon:       icon,
		IconSize:   14,
		IconColor:  &th.Button.TextColor,
		Color:      th.Button.DefaultColor,
		HoverColor: th.Button.HoverColor,
		TextColor:  th.Button.TextColor,
		FontSize:   12,
		Padding:    types.EdgeInsetsLTRB(16, 5, 16, 5),
		MinWidth:   120,
		MinHeight:  28,
		OnClick:    onClick,
	}
}

// ── 最近项目 ──

// ideRecentSection 最近项目列表（仅在有记录时显示）。
func ideRecentSection() widget.Widget {
	recents := core.Settings.RecentProjects
	nonEmpty := make([]string, 0, len(recents))
	for _, p := range recents {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	if len(nonEmpty) == 0 {
		return ui.VGap(8) // 无最近项目 -> 仅返回小间距
	}

	// 最多显示 6 条
	maxShow := 6
	if len(nonEmpty) > maxShow {
		nonEmpty = nonEmpty[:maxShow]
	}

	rows := []widget.Widget{
		widget.Div(widget.Style{Height: 1, BackgroundColor: ui.ShellBorder}),
		ui.VGap(12),
		ui.TextC("最近项目", *ui.FgSubtle, 13),
		ui.VGap(8),
	}
	for _, p := range nonEmpty {
		rows = append(rows, ideRecentRow(p))
	}
	rows = append(rows, ui.VGap(12))

	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch", Padding: types.EdgeInsetsLTRB(48, 0, 48, 0)},
		rows,
	)
}

// ideRecentRow 单个最近项目行：图标 + 路径 + 打开按钮。
func ideRecentRow(p string) widget.Widget {
	// 如果路径太长，截断显示
	display := p
	if len(display) > 60 {
		display = "..." + display[len(display)-57:]
	}

	// 路径文本使用单行省略
	pathText := widget.NewText(display, *ui.FgSubtle)
	pathText.Font.Size = 11
	pathText.MaxLines = 1

	openBtn := &widget.Button{
		Text:       "打开",
		FontSize:   11,
		TextColor:  *ui.Accent,
		Color:      types.Color{}, // 透明底
		HoverColor: *ui.BgHover,
		Padding:    types.EdgeInsetsLTRB(8, 2, 8, 2),
		MinHeight:  22,
		OnClick: func() {
			if OnOpenRecent != nil {
				OnOpenRecent(p)
			}
		},
	}

	return widget.Div(
		widget.Style{
			FlexDirection: "row",
			AlignItems:    "center",
			Padding:       types.EdgeInsetsLTRB(8, 4, 8, 4),
		},
		widget.Lucide("folder", widget.IconSize(13), widget.IconColor(*ui.FgSubtle)),
		ui.HGap(8),
		&widget.Expanded{
			SingleChildWidget: widget.SingleChildWidget{Child: pathText},
			Flex:              1,
		},
		ui.HGap(8),
		openBtn,
	)
}

// ── 工具 ──

// spacerF 弹性空白（flex:n）。
func spacerF(n int) widget.Widget {
	return &widget.Expanded{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(widget.Style{})},
		Flex: n,
	}
}
