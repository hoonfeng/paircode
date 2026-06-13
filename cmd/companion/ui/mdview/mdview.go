// 对话面板的轻量 Markdown 渲染。块级：标题/无序·有序列表/任务列表/引用/分隔线/
// 表格/代码块(等宽+高亮+复制)/段落。行内 **bold**/*italic*/`code` 用不同颜色区分，
// 链接 [text](url)用强调色+下划线。
//
//go:build windows

package mdview

import (
	"regexp"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

var (
	mdBoldRe    = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	mdItalicRe  = regexp.MustCompile(`\*([^*]+)\*`)
	mdCodeInlRe = regexp.MustCompile("`([^`]+)`")
	mdLinkRe    = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	mdTaskDone  = regexp.MustCompile(`^-\s+\[x\]\s+(.*)`)
	mdTaskTodo  = regexp.MustCompile(`^-\s+\[\s*\]\s+(.*)`)
	mdTableRow  = regexp.MustCompile(`^\|.+?\|$`)
	mdOrderedRe = regexp.MustCompile(`^(\d+)\.\s+(.*)`)
	MonoFont    = canvas.Font{Family: "Consolas", Size: 12.5}
)

// ExpandTabs 把 tab 展开为 4 空格（等宽字体下 tab 会渲染成缺字框，且 Go 代码多 tab 缩进）。
func ExpandTabs(s string) string { return strings.ReplaceAll(s, "\t", "    ") }

// Render 把 markdown 文本渲染为块级 widget（标题按字号分级 / 列表 / 任务列表 / 引用 / 分隔线 /
// 代码块(等宽+高亮+复制) / 表格 / 段落）。行内标记用颜色区分显示。
func Render(text string) widget.Widget {
	var blocks []widget.Widget
	inCode := false
	var code []string
	var codeLang string
	inTable := false
	var tableRows []string

	for _, raw := range strings.Split(text, "\n") {
		t := strings.TrimSpace(raw)

		// 代码块起止
		if strings.HasPrefix(t, "```") {
			if inCode {
				blocks = append(blocks, mdCodeBlock(code, codeLang))
				code, codeLang, inCode = nil, "", false
			} else {
				inCode = true
				codeLang = strings.TrimSpace(t[3:])
			}
			continue
		}
		if inCode {
			code = append(code, raw)
			continue
		}

		// 表格块结束（空行或非表格行）
		if inTable && (t == "" || !mdTableRow.MatchString(t)) {
			blocks = append(blocks, mdTable(tableRows))
			tableRows, inTable = nil, false
			if t == "" {
				continue
			}
		}

		if t == "" {
			continue
		}

		switch {
		case t == "---" || t == "***":
			blocks = append(blocks, widget.Div(widget.Style{Height: 1, BackgroundColor: ui.Border}))

		case strings.HasPrefix(t, "### "):
			blocks = append(blocks, inlineRow(mdRenderInline(t[4:]), 13.5))

		case strings.HasPrefix(t, "## "):
			blocks = append(blocks, inlineRow(mdRenderInline(t[3:]), 15))

		case strings.HasPrefix(t, "# "):
			blocks = append(blocks, inlineRow(mdRenderInline(t[2:]), 17))

		case strings.HasPrefix(t, "> "):
			blocks = append(blocks, mdQuote(mdRenderInline(t[2:])))

		case mdTaskDone.MatchString(t) || mdTaskTodo.MatchString(t):
			blocks = append(blocks, mdTaskItem(t))

		case strings.HasPrefix(t, "- "), strings.HasPrefix(t, "* "):
			blocks = append(blocks, mdListItem("•", mdRenderInline(t[2:])))

		case mdTableRow.MatchString(t):
			// 分隔行（| --- | --- |）跳过
			if strings.Count(t, "-") > len(t)/2 {
				continue
			}
			tableRows = append(tableRows, t)
			inTable = true

		default:
			if m := mdOrderedRe.FindStringSubmatch(t); m != nil {
				blocks = append(blocks, mdListItem(m[1]+".", mdRenderInline(m[2])))
			} else {
				blocks = append(blocks, inlineRow(mdRenderInline(t)))
			}
		}
	}

	// 兜底闭合
	if inCode && len(code) > 0 {
		blocks = append(blocks, mdCodeBlock(code, codeLang))
	}
	if inTable && len(tableRows) > 0 {
		blocks = append(blocks, mdTable(tableRows))
	}

	if len(blocks) == 0 {
		return widget.Div(widget.Style{})
	}
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", Gap: 6}, blocks)
}

// inlineRow 把一组行内 span 包成一行（非段落块）。fontSize 为 0 时使用 span 自身字号。
func inlineRow(spans []span, fontSize ...float64) widget.Widget {
	if len(spans) == 0 {
		return widget.Div(widget.Style{})
	}
	fs := 13.0
	if len(fontSize) > 0 && fontSize[0] > 0 {
		fs = fontSize[0]
	}
	kids := make([]widget.Widget, len(spans))
	for i, s := range spans {
		fnt := s.Font
		if fnt.Size == 0 {
			fnt.Size = fs
		}
		t := widget.NewText(s.Text, s.Color)
		t.Font = fnt
		kids[i] = t
	}
	return widget.Div(widget.Style{FlexDirection: "row", FlexWrap: "wrap"}, kids)
}

// span 一个行内文本片段（带颜色和字体）。
type span struct {
	Text  string
	Color types.Color
	Font  canvas.Font
}

// mdRenderInline 解析行内标记（加粗/斜体/代码/链接），返回 span 列表。
// 用不同颜色区分：加粗=Fg(更亮)、斜体=FgSubtle、行内代码=Accent、链接=Accent+下划线文本。
func mdRenderInline(s string) []span {
	// 第一步：用链接正则替换 [text](url) → 占位标记
	type linkMatch struct {
		text, url string
		start, end int
	}
	var links []linkMatch
	// 用链接正则找出所有 [text](url)
	s = mdLinkRe.ReplaceAllStringFunc(s, func(m string) string {
		parts := mdLinkRe.FindStringSubmatch(m)
		if len(parts) == 3 {
			idx := len(links)
			links = append(links, linkMatch{text: parts[1], url: parts[2]})
			return "\x00LINK" + itoa(idx) + "\x00"
		}
		return m
	})

	// 第二步：解析加粗/斜体/代码标记
	type token struct {
		text  string
		style string
	}
	var tokens []token

	// 按顺序处理加粗、斜体、行内代码
	// 用最简单的逐字符扫描法
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		// 检查行内代码 `…`
		if runes[i] == '`' {
			j := i + 1
			for j < len(runes) && runes[j] != '`' {
				j++
			}
			if j < len(runes) {
				tokens = append(tokens, token{text: string(runes[i+1:j]), style: "code"})
				i = j + 1
				continue
			}
		}
		// 检查加粗 **…**
		if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
			j := i + 2
			for j+1 < len(runes) && !(runes[j] == '*' && runes[j+1] == '*') {
				j++
			}
			if j+1 < len(runes) {
				tokens = append(tokens, token{text: string(runes[i+2:j]), style: "bold"})
				i = j + 2
				continue
			}
		}
		// 检查斜体 *…*（但排除 ** 开头）
		if runes[i] == '*' && (i+1 >= len(runes) || runes[i+1] != '*') {
			j := i + 1
			for j < len(runes) && runes[j] != '*' {
				j++
			}
			if j < len(runes) && (j+1 >= len(runes) || runes[j+1] != '*') {
				tokens = append(tokens, token{text: string(runes[i+1:j]), style: "italic"})
				i = j + 1
				continue
			}
		}
		// 检查 LINK 占位
		if runes[i] == '\x00' {
			rest := string(runes[i:])
			for li, lm := range links {
				marker := "\x00LINK" + itoa(li) + "\x00"
				if strings.HasPrefix(rest, marker) {
					tokens = append(tokens, token{text: lm.text + " → " + lm.url, style: "link"})
					i += len([]rune(marker))
					goto next
				}
			}
		}
		// 普通文本
		tokens = append(tokens, token{text: string(runes[i]), style: ""})
		i++
	next:
	}

	// 第三步：转成 span 列表
	var spans []span
	for _, tok := range tokens {
		if tok.text == "" {
			continue
		}
		col := *ui.Fg
		fnt := canvas.Font{Family: "", Size: 13}
		switch tok.style {
		case "bold":
			col = *ui.Fg // bold 用主色（更突出）
		case "italic":
			col = *ui.FgSubtle
		case "code":
			col = *ui.Accent
			fnt = MonoFont
			fnt.Size = 13
		case "link":
			col = *ui.Accent
		}
		spans = append(spans, span{Text: tok.text, Color: col, Font: fnt})
	}
	return spans
}

// itoa 数字转字符串（简易版，避免 strconv 导入）
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	d := ""
	for n > 0 {
		d = string(rune('0'+n%10)) + d
		n /= 10
	}
	return d
}

func mdQuote(spans []span) widget.Widget {
	return widget.Div(widget.Style{FlexDirection: "row", AlignItems: "stretch", Gap: 8},
		widget.Div(widget.Style{Width: 3, BackgroundColor: ui.Border, BorderRadius: 1.5}),
		ui.Expand(inlineRow(spans)),
	)
}

// mdTaskItem 任务列表项：- [x] 完成 / - [ ] 待办，用 checkbox 图标。
func mdTaskItem(line string) widget.Widget {
	done := mdTaskDone.FindStringSubmatch(line)
	todo := mdTaskTodo.FindStringSubmatch(line)
	var icon string
	var text string
	if len(done) == 2 {
		icon = "check-square"
		text = done[1]
	} else if len(todo) == 2 {
		icon = "square"
		text = todo[1]
	}
	col := *ui.Fg
	if icon == "check-square" {
		col = *ui.Success
	}
	return widget.Div(widget.Style{FlexDirection: "row", Gap: 8, AlignItems: "center"},
		widget.Lucide(icon, widget.IconSize(12), widget.IconColor(col)),
		ui.Expand(inlineRow(mdRenderInline(text))),
	)
}

func mdListItem(bullet string, spans []span) widget.Widget {
	return widget.Div(widget.Style{FlexDirection: "row", Gap: 8},
		ui.TextC(bullet, *ui.FgSubtle, 13),
		ui.Expand(inlineRow(spans)),
	)
}

// mdTable 渲染 markdown 表格（| col1 | col2 | 格式）。
func mdTable(rows []string) widget.Widget {
	if len(rows) == 0 {
		return widget.Div(widget.Style{})
	}

	// 解析所有行为单元格列表
	var parsed [][]string
	for _, row := range rows {
		r := strings.TrimSpace(row)
		r = strings.TrimPrefix(r, "|")
		r = strings.TrimSuffix(r, "|")
		cells := strings.Split(r, "|")
		parsed = append(parsed, cells)
	}

	// 计算最大列数
	maxCols := 0
	for _, cells := range parsed {
		if len(cells) > maxCols {
			maxCols = len(cells)
		}
	}
	if maxCols == 0 {
		return widget.Div(widget.Style{})
	}

	var body []widget.Widget
	for ri, cells := range parsed {
		// 补齐到 maxCols
		for len(cells) < maxCols {
			cells = append(cells, "")
		}
		var rowKids []widget.Widget
		for ci, cell := range cells {
			cellText := strings.TrimSpace(cell)
			fg := *ui.Fg
			bg := types.Color{} // 透明
			if ri == 0 {
				fg = *ui.Fg    // 表头用主色
			}
			cellStyle := widget.Style{
				Padding:  types.EdgeInsetsLTRB(6, 3, 6, 3),
			}
			rowKids = append(rowKids,
				widget.Div(cellStyle,
					inlineRow(mdRenderInline(cellText)),
				),
			)
		}
		// 用 FlexRow 的 Gap 模拟列间距；给 border-bottom 区分行
		rowStyle := widget.Style{
			FlexDirection: "row",
			AlignItems:    "center",
			BorderColor:   ui.Border,
			BorderWidth:   1,
		}
		if ri == 0 {
			// 表头加底部强调线
			rowStyle.BorderWidth = 0
			body = append(body, widget.Div(widget.Style{
				FlexDirection: "column", AlignItems: "stretch",
				BorderColor: ui.Border, BorderWidth: 1, BorderRadius: 4,
				BackgroundColor: ui.BgSubtle,
			},
				widget.Div(rowStyle, rowKids),
				widget.Div(widget.Style{Height: 1, BackgroundColor: ui.Border}),
			))
		} else {
			body = append(body, widget.Div(rowStyle, rowKids))
		}
	}

	return widget.Div(widget.Style{
		FlexDirection:  "column",
		AlignItems:     "stretch",
		BorderColor:    ui.Border,
		BorderWidth:    1,
		BorderRadius:   4,
		Margin:         types.EdgeInsetsLTRB(0, 0, 0, 0),
	}, body)
}

// mdCodeBlock 代码块：语法高亮逐行（等宽、保留缩进）+ 深底框 + 头(语言标签 + 复制按钮)。
func mdCodeBlock(lines []string, lang string) widget.Widget {
	joined := strings.Join(lines, "\n")                    // 复制用原文（保留 tab）
	rows := widget.HighlightCode(ExpandTabs(joined), lang) // 显示用：tab→空格
	body := make([]widget.Widget, 0, len(rows))
	for _, spans := range rows {
		body = append(body, mdCodeLine(spans))
	}
	bodyDiv := widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"}, body)

	kids := []widget.Widget{}
	if lang != "" || widget.ClipboardWrite != nil {
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
