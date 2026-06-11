// 对话面板的轻量 Markdown 渲染（GitHub 深色主题）。块级：标题/无序·有序列表/引用/分隔线/
// 代码块(等宽+复制)/段落。行内 **bold**/*italic*/`code` 简化为去标记纯文本（goui Text 单一样式，
// 行内混排样式属进阶）。参考 internal/widget/markdown.go 的块级解析，但配深色 + 代码块复制按钮。
//
//go:build windows

package mdview

import (
	"regexp"
	"strings"

	"github.com/user/gou-ide/cmd/companion/ui"
	"github.com/user/goui/pkg/canvas"
	"github.com/user/goui/pkg/types"
	"github.com/user/goui/pkg/widget"
)

var (
	mdBoldRe    = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	mdItalicRe  = regexp.MustCompile(`\*([^*]+)\*`)
	mdCodeInlRe = regexp.MustCompile("`([^`]+)`")
	mdOrderedRe = regexp.MustCompile(`^(\d+)\.\s+(.*)`)
	MonoFont    = canvas.Font{Family: "Consolas", Size: 12.5}
)

// ExpandTabs 把 tab 展开为 4 空格（等宽字体下 tab 会渲染成缺字框，且 Go 代码多 tab 缩进）。
func ExpandTabs(s string) string { return strings.ReplaceAll(s, "\t", "    ") }

// mdStripInline 去行内标记（粗体/斜体/行内代码）→纯文本。
func mdStripInline(s string) string {
	s = mdBoldRe.ReplaceAllString(s, "$1")
	s = mdItalicRe.ReplaceAllString(s, "$1")
	s = mdCodeInlRe.ReplaceAllString(s, "$1")
	return s
}

// Render 把 markdown 文本渲染为深色块级 widget（标题按字号分级 / 列表 / 引用 / 分隔线 / 代码块 / 段落）。
func Render(text string) widget.Widget {
	var blocks []widget.Widget
	inCode := false
	var code []string
	var codeLang string
	for _, raw := range strings.Split(text, "\n") {
		t := strings.TrimSpace(raw)
		if strings.HasPrefix(t, "```") {
			if inCode {
				blocks = append(blocks, mdCodeBlock(code, codeLang))
				code, codeLang, inCode = nil, "", false
			} else {
				inCode = true
				codeLang = strings.TrimSpace(t[3:]) // ```go → "go"
			}
			continue
		}
		if inCode {
			code = append(code, raw)
			continue
		}
		if t == "" {
			continue
		}
		switch {
		case t == "---" || t == "***":
			blocks = append(blocks, widget.Div(widget.Style{Height: 1, BackgroundColor: ui.Border}))
		case strings.HasPrefix(t, "### "):
			blocks = append(blocks, ui.TextC(mdStripInline(t[4:]), *ui.Fg, 13.5))
		case strings.HasPrefix(t, "## "):
			blocks = append(blocks, ui.TextC(mdStripInline(t[3:]), *ui.Fg, 15))
		case strings.HasPrefix(t, "# "):
			blocks = append(blocks, ui.TextC(mdStripInline(t[2:]), *ui.Fg, 17))
		case strings.HasPrefix(t, "> "):
			blocks = append(blocks, mdQuote(mdStripInline(t[2:])))
		case strings.HasPrefix(t, "- "), strings.HasPrefix(t, "* "):
			blocks = append(blocks, mdListItem("•", mdStripInline(t[2:])))
		default:
			if m := mdOrderedRe.FindStringSubmatch(t); m != nil {
				blocks = append(blocks, mdListItem(m[1]+".", mdStripInline(m[2])))
			} else {
				blocks = append(blocks, ui.TextC(mdStripInline(t), *ui.Fg, 13))
			}
		}
	}
	if inCode && len(code) > 0 { // 未闭合代码块兜底
		blocks = append(blocks, mdCodeBlock(code, codeLang))
	}
	if len(blocks) == 0 {
		return widget.Div(widget.Style{})
	}
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", Gap: 6}, blocks)
}

func mdQuote(text string) widget.Widget {
	return widget.Div(widget.Style{FlexDirection: "row", AlignItems: "stretch", Gap: 8},
		widget.Div(widget.Style{Width: 3, BackgroundColor: ui.Border, BorderRadius: 1.5}),
		ui.Expand(ui.TextC(text, *ui.FgSubtle, 13)),
	)
}

func mdListItem(bullet, text string) widget.Widget {
	return widget.Div(widget.Style{FlexDirection: "row", Gap: 8},
		ui.TextC(bullet, *ui.FgSubtle, 13),
		ui.Expand(ui.TextC(text, *ui.Fg, 13)),
	)
}

// mdCodeBlock 代码块：语法高亮逐行（等宽、保留缩进）+ 深底框 + 头(语言标签 + 复制按钮)。
// 高亮复用 widget.HighlightCode（编辑器同款词法内核 + 主题色）。
func mdCodeBlock(lines []string, lang string) widget.Widget {
	joined := strings.Join(lines, "\n")                    // 复制用原文（保留 tab）
	rows := widget.HighlightCode(ExpandTabs(joined), lang) // 显示用：tab→空格（等宽字体下 tab 渲染成缺字框）
	body := make([]widget.Widget, 0, len(rows))
	for _, spans := range rows {
		body = append(body, mdCodeLine(spans))
	}
	bodyDiv := widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"}, body)

	kids := []widget.Widget{}
	if lang != "" || widget.ClipboardWrite != nil { // 头：语言标签 + 复制（剪贴板可用才放，避免空按钮）
		hdr := []widget.Widget{}
		if lang != "" {
			hdr = append(hdr, ui.TextC(lang, *ui.FgMuted, 10))
		}
		hdr = append(hdr, ui.Expand(widget.Div(widget.Style{})))
		if widget.ClipboardWrite != nil {
			hdr = append(hdr, &widget.Button{
				Icon: "copy", IconSize: 12, TextColor: *ui.FgMuted,
				OnClick:  func() { widget.ClipboardWrite(joined) },
				Color:    *ui.BgMuted,
				MinWidth: 22, MinHeight: 20,
			})
		}
		kids = append(kids, widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"}, hdr))
	}
	kids = append(kids, bodyDiv)
	return widget.Div(
		widget.Style{BackgroundColor: ui.Bg, BorderColor: ui.Border, BorderWidth: 1, BorderRadius: 4,
			Padding: types.EdgeInsets(8), FlexDirection: "column", AlignItems: "stretch", Gap: 4},
		kids,
	)
}

// mdCodeLine 一行高亮代码：着色 span 横排（等宽，保留缩进）；空行给最小高保留间距。
func mdCodeLine(spans []widget.HighlightSpan) widget.Widget {
	if len(spans) == 0 {
		return widget.Div(widget.Style{Height: 8})
	}
	kids := make([]widget.Widget, 0, len(spans))
	for _, sp := range spans {
		t := widget.NewText(sp.Text, sp.Color)
		t.Font = MonoFont
		kids = append(kids, t)
	}
	return widget.Div(widget.Style{FlexDirection: "row"}, kids)
}
