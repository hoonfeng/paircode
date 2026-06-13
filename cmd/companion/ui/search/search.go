// 搜索面板 —— 左栏「搜索」内容：复刻参考 SearchPanel（搜索框 + 大小写/全词/正则开关 +
// 结果树：文件头可折叠 + 命中行 行号+高亮匹配）。跨文件内容搜索，点命中行跳编辑器。详见 AGENTS.md。
//
//go:build windows

package searchpanel

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

var searchHL = types.ColorFromRGB(87, 64, 0) // 命中高亮底（暗金）

const searchMaxMatches = 500

// searchMatch 一行命中：行号 + 行文本（已左 trim）+ 匹配字节区间（相对 trim 后文本）。
type searchMatch struct {
	line   int
	text   string
	ranges [][2]int
}

type searchFile struct {
	rel     string
	abspath string
	matches []searchMatch
}

// searchFlatItem 扁平化的可见项，供 VirtualList 按需渲染（取代 ScrollView 全量创建 Widget）。
type searchFlatItem struct {
	kind     byte // 0=fileHeader, 1=matchRow
	fileIdx  int  // index in s.files
	matchIdx int  // index in s.files[fileIdx].matches（仅 kind=1）
}

var theSearch = &searchState{collapsed: map[string]bool{}}

// SearchPanel 搜索面板组件。
type SearchPanel struct{ widget.StatefulWidget }

func (p *SearchPanel) CreateState() widget.State { return theSearch }

type searchState struct {
	widget.BaseState
	query                           string
	caseSensitive, wholeWord, regex bool
	showReplace                     bool
	replaceText                     string
	files                           []searchFile
	totalMatches                    int
	searched                        bool
	capped                          bool
	errMsg                          string
	collapsed                       map[string]bool
	previewRe                       *regexp.Regexp // 替换预览用（Build 时按 replaceText 编译；nil=不预览）

	flatItems []searchFlatItem // 扁平化可见项（Build 时重建，供 VirtualList 用）
}

// compile 据查询 + 选项构造正则。
func (s *searchState) compile() (*regexp.Regexp, error) {
	pat := s.query
	if !s.regex {
		pat = regexp.QuoteMeta(pat)
	}
	if s.wholeWord {
		pat = `\b` + pat + `\b`
	}
	if !s.caseSensitive {
		pat = `(?i)` + pat
	}
	return regexp.Compile(pat)
}

var searchSkipDir = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true, "build": true,
	".cache": true, ".idea": true, ".vscode": true, "bin": true, "obj": true,
}

// run 跨文件搜索（同步遍历工作区；跳过 .git/node_modules 等、>1MB、二进制；命中上限 500）。
func (s *searchState) run() {
	s.searched = true
	s.files, s.totalMatches, s.capped, s.errMsg = nil, 0, false, ""
	if strings.TrimSpace(s.query) == "" {
		s.SetState()
		return
	}
	re, err := s.compile()
	if err != nil {
		s.errMsg = "正则错误：" + err.Error()
		s.SetState()
		return
	}
	root := core.Root()
	filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return nil
		}
		if s.totalMatches >= searchMaxMatches {
			s.capped = true
			return filepath.SkipAll
		}
		if d.IsDir() {
			if searchSkipDir[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if info, e := d.Info(); e == nil && info.Size() > 1<<20 {
			return nil // >1MB 跳过
		}
		data, e := os.ReadFile(path)
		if e != nil || bytes.IndexByte(data, 0) >= 0 {
			return nil // 读失败 / 含 NUL → 视为二进制
		}
		var matches []searchMatch
		for i, line := range strings.Split(string(data), "\n") {
			locs := re.FindAllStringIndex(line, -1)
			if len(locs) == 0 {
				continue
			}
			trimmed := strings.TrimLeft(line, " \t")
			shift := len(line) - len(trimmed)
			var ranges [][2]int
			for _, l := range locs {
				a, b := l[0]-shift, l[1]-shift
				if a < 0 {
					a = 0
				}
				if b > len(trimmed) {
					b = len(trimmed)
				}
				if a < b {
					ranges = append(ranges, [2]int{a, b})
				}
			}
			matches = append(matches, searchMatch{line: i + 1, text: trimmed, ranges: ranges})
			s.totalMatches += len(locs)
			if s.totalMatches >= searchMaxMatches {
				break
			}
		}
		if len(matches) > 0 {
			rel, _ := filepath.Rel(root, path)
			s.files = append(s.files, searchFile{rel: filepath.ToSlash(rel), abspath: path, matches: matches})
		}
		return nil
	})
	s.SetState()
}

func (s *searchState) toggleFile(rel string) { s.collapsed[rel] = !s.collapsed[rel]; s.SetState() }

// searchInputStyle 已下沉到 ui.StyleInput。

// confirmReplaceAll 先确认（替换不可撤销）。
func (s *searchState) confirmReplaceAll() {
	if strings.TrimSpace(s.query) == "" || len(s.files) == 0 {
		return
	}
	msg := fmt.Sprintf("将在 %d 个文件中把匹配替换为「%s」，此操作不可撤销。确定？", len(s.files), s.replaceText)
	widget.ShowConfirm("全部替换", msg, widget.MsgWarning, func() { s.replaceAll() }, nil)
}

// replaceAll 全部替换：写回 + 重搜 + 提示。
func (s *searchState) replaceAll() {
	n := s.doReplace()
	s.run() // 重搜：结果应减少/清空
	widget.MessageSuccess(fmt.Sprintf("已在 %d 个文件中替换。", n))
}

// doReplace 对所有命中文件做整文件正则替换并写回（非正则=字面替换；正则=支持 $1）。返回改动文件数。
func (s *searchState) doReplace() int {
	re, err := s.compile()
	if err != nil {
		return 0
	}
	n := 0
	for _, f := range s.files {
		data, e := os.ReadFile(f.abspath)
		if e != nil {
			continue
		}
		var out string
		if s.regex {
			out = re.ReplaceAllString(string(data), s.replaceText)
		} else {
			out = re.ReplaceAllLiteralString(string(data), s.replaceText)
		}
		if out != string(data) {
			if os.WriteFile(f.abspath, []byte(out), 0o644) == nil {
				n++
				editorpanel.Editor.ReloadIfOpen(f.abspath) // 已打开则刷新编辑器
			}
		}
	}
	return n
}

// replaceFile 只替换某个文件（增强：参考仅有全部替换）。改动后重搜。
func (s *searchState) replaceFile(f searchFile) {
	if s.doReplaceFile(f) {
		s.run()
	}
}

// doReplaceFile 对单个文件做整文件正则替换并写回；返回是否改动。
func (s *searchState) doReplaceFile(f searchFile) bool {
	re, err := s.compile()
	if err != nil {
		return false
	}
	data, e := os.ReadFile(f.abspath)
	if e != nil {
		return false
	}
	var out string
	if s.regex {
		out = re.ReplaceAllString(string(data), s.replaceText)
	} else {
		out = re.ReplaceAllLiteralString(string(data), s.replaceText)
	}
	if out == string(data) {
		return false
	}
	if os.WriteFile(f.abspath, []byte(out), 0o644) != nil {
		return false
	}
	editorpanel.Editor.ReloadIfOpen(f.abspath)
	return true
}

// ─── UI ───────────────────────────────────────────────────────

func (s *searchState) Build(ctx widget.BuildContext) widget.Widget {
	s.previewRe = nil // 替换预览：替换模式且有替换文本时编译一次，供 matchRow 复用
	if s.showReplace && s.replaceText != "" {
		if re, err := s.compile(); err == nil {
			s.previewRe = re
		}
	}
	rows := []widget.Widget{s.searchBar()}
	switch {
	case s.errMsg != "":
		rows = append(rows, widget.Div(widget.Style{Padding: types.EdgeInsets(10)}, ui.TextC(s.errMsg, *ui.Danger, 11)))
	case s.searched && len(s.files) == 0:
		rows = append(rows, ui.Expand(ui.EmptyState("search", "无结果", "没有匹配的内容")))
	case s.searched:
		rows = append(rows, s.stats())
		s.flatItems = s.buildFlatItems()
		if len(s.flatItems) == 0 {
			rows = append(rows, ui.Expand(ui.EmptyState("search", "无结果", "没有匹配的内容")))
		} else {
			rows = append(rows, ui.Expand(&widget.VirtualList{
				ItemCount:  len(s.flatItems),
				ItemHeight: 24,
				RenderItem: s.renderFlatItem,
			}))
		}
	}
	return widget.Div(
		widget.Style{BackgroundColor: ui.ShellSide, FlexDirection: "column", AlignItems: "stretch"},
		rows,
	)
}

// buildFlatItems 据搜索结果和折叠态构建扁平可见项列表。
func (s *searchState) buildFlatItems() []searchFlatItem {
	var out []searchFlatItem
	for fi, f := range s.files {
		out = append(out, searchFlatItem{kind: 0, fileIdx: fi})
		if s.collapsed[f.rel] {
			continue
		}
		for mi := range f.matches {
			out = append(out, searchFlatItem{kind: 1, fileIdx: fi, matchIdx: mi})
		}
	}
	return out
}

// renderFlatItem VirtualList 回调：按 index 渲染一个扁平项。
func (s *searchState) renderFlatItem(i int) widget.Widget {
	if i < 0 || i >= len(s.flatItems) {
		return nil
	}
	fi := s.flatItems[i]
	switch fi.kind {
	case 0: // fileHeader
		f := s.files[fi.fileIdx]
		return s.flatFileHeader(f.rel, len(f.matches), f.abspath)
	case 1: // matchRow
		f := s.files[fi.fileIdx]
		return s.matchRow(f.abspath, f.matches[fi.matchIdx])
	}
	return nil
}

// flatFileHeader 从扁平项渲染搜索文件头（可折叠，含替换按钮）。
func (s *searchState) flatFileHeader(rel string, count int, abspath string) widget.Widget {
	chev := "chevron-down"
	if s.collapsed[rel] {
		chev = "chevron-right"
	}
	header := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(6, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Lucide(chev, widget.IconSize(13), widget.IconColor(*ui.ShellTextDim)),
			widget.Div(widget.Style{Width: 3}),
			widget.Lucide("file-text", widget.IconSize(13), widget.IconColor(*ui.ShellTextDim)),
			widget.Div(widget.Style{Width: 5}),
			ui.Expand(ui.TextC(rel, *ui.ShellText, 12)),
			ui.TextC(ui.Itoa(count), *ui.ShellTextDim, 10),
		)},
		OnClick:    func() { s.toggleFile(rel) },
		HoverColor: *ui.FtHover,
	}
	// 替换模式：文件头右侧加「替换此文件」
	if s.previewRe != nil {
		r := rel // capture
		return widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center", BackgroundColor: ui.ShellSide},
			ui.Expand(header),
			&widget.Button{
				SingleChildWidget: widget.SingleChildWidget{Child: ui.TextC("替换", *ui.White, 10)},
				OnClick:           func() { s.replaceOneFile(r) },
				Color:             *ui.Warning, MinHeight: 20, Padding: types.EdgeInsetsLTRB(7, 0, 7, 0),
			},
			widget.Div(widget.Style{Width: 6}),
		)
	}
	return header
}

// replaceOneFile 替换单个文件（按 rel 路径在 s.files 中查找并替换）。
func (s *searchState) replaceOneFile(rel string) {
	for _, f := range s.files {
		if f.rel == rel {
			s.replaceFile(f)
			return
		}
	}
}

func (s *searchState) searchBar() widget.Widget {
	in := widget.NewInput("", func(t string) { s.query = t }).WithPlaceholder("搜索").WithOnSubmit(func(string) { s.run() })
	ui.StyleInput(in)
	rows := []widget.Widget{
		widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			ui.Expand(in),
			widget.Div(widget.Style{Width: 4}),
			ui.ShellIconBtn("search", s.run),
		),
		widget.Div(widget.Style{Height: 6}),
		widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			searchToggle("Aa", "区分大小写", s.caseSensitive, func() { s.caseSensitive = !s.caseSensitive; s.run() }),
			widget.Div(widget.Style{Width: 4}),
			searchToggle("全词", "全字匹配", s.wholeWord, func() { s.wholeWord = !s.wholeWord; s.run() }),
			widget.Div(widget.Style{Width: 4}),
			searchToggle(".*", "正则", s.regex, func() { s.regex = !s.regex; s.run() }),
			widget.Div(widget.Style{Width: 4}),
			searchToggle("替换", "替换模式", s.showReplace, func() { s.showReplace = !s.showReplace; s.SetState() }),
		),
	}
	if s.showReplace {
		rin := widget.NewInput("替换为...", func(t string) { s.replaceText = t; s.SetState() }).WithOnSubmit(func(string) { s.confirmReplaceAll() })
		ui.StyleInput(rin)
		rows = append(rows,
			widget.Div(widget.Style{Height: 6}),
			widget.Div(
				widget.Style{FlexDirection: "row", AlignItems: "center"},
				ui.Expand(rin),
				widget.Div(widget.Style{Width: 4}),
				ui.SolidDangerBtnX("全部替换", s.confirmReplaceAll, ui.BtnOpts{Size: ui.SizeSm}),
			),
		)
	}
	return widget.Div(
		widget.Style{Padding: types.EdgeInsets(6), FlexDirection: "column", AlignItems: "stretch",
			BackgroundColor: ui.ShellSide, BorderColor: ui.ShellBorder, BorderWidth: 1},
		rows,
	)
}

func (s *searchState) stats() widget.Widget {
	txt := fmtMatchStats(s.totalMatches, len(s.files))
	if s.capped {
		txt += "（已截断 " + ui.Itoa(searchMaxMatches) + "）"
	}
	return widget.Div(
		widget.Style{Height: 22, Padding: types.EdgeInsetsLTRB(8, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
		ui.TextC(txt, *ui.ShellTextDim, 10),
	)
}

// fileBlock 一个文件的结果：文件头（可折叠）+ 命中行。
func (s *searchState) fileBlock(out *[]widget.Widget, f searchFile) {
	chev := "chevron-down"
	if s.collapsed[f.rel] {
		chev = "chevron-right"
	}
	header := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(6, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Lucide(chev, widget.IconSize(13), widget.IconColor(*ui.ShellTextDim)),
			widget.Div(widget.Style{Width: 3}),
			widget.Lucide("file-text", widget.IconSize(13), widget.IconColor(*ui.ShellTextDim)),
			widget.Div(widget.Style{Width: 5}),
			ui.Expand(ui.TextC(f.rel, *ui.ShellText, 12)),
			ui.TextC(ui.Itoa(len(f.matches)), *ui.ShellTextDim, 10),
		)},
		OnClick:    func() { s.toggleFile(f.rel) },
		HoverColor: *ui.FtHover,
	}
	if s.previewRe != nil { // 替换模式：文件头右侧加「替换此文件」（在 Clickable 外，避免点按钮误折叠）
		ff := f
		*out = append(*out, widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center", BackgroundColor: ui.ShellSide},
			ui.Expand(header),
			&widget.Button{
				SingleChildWidget: widget.SingleChildWidget{Child: ui.TextC("替换", *ui.White, 10)},
				OnClick:           func() { s.replaceFile(ff) },
				Color:             *ui.Warning, MinHeight: 20, Padding: types.EdgeInsetsLTRB(7, 0, 7, 0),
			},
			widget.Div(widget.Style{Width: 6}),
		))
	} else {
		*out = append(*out, header)
	}
	if s.collapsed[f.rel] {
		return
	}
	for _, m := range f.matches {
		*out = append(*out, s.matchRow(f.abspath, m))
	}
}

func (s *searchState) matchRow(abspath string, m searchMatch) widget.Widget {
	seg := []widget.Widget{
		widget.Div(widget.Style{Width: 34, FlexDirection: "row", AlignItems: "center"}, ui.TextC(ui.Itoa(m.line), *ui.ShellTextDim, 10)),
	}
	seg = append(seg, highlightedLine(m.text, m.ranges)...)
	mainRow := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 22, Padding: types.EdgeInsetsLTRB(20, 0, 6, 0), FlexDirection: "row", AlignItems: "center"},
			seg,
		)},
		OnClick:    func() { editorpanel.Editor.OpenAt(abspath, m.line) },
		HoverColor: *ui.FtHover,
	}
	if s.previewRe == nil {
		return mainRow
	}
	var repl string // 替换预览：该行替换后的样子（绿色）
	if s.regex {
		repl = s.previewRe.ReplaceAllString(m.text, s.replaceText)
	} else {
		repl = s.previewRe.ReplaceAllLiteralString(m.text, s.replaceText)
	}
	if repl == m.text {
		return mainRow // 该行无变化（不应发生，保险）
	}
	if len(repl) > 200 {
		repl = repl[:200]
	}
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		mainRow,
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(40, 0, 6, 2), FlexDirection: "row", AlignItems: "center"},
			ui.TextC(repl, *ui.Success, 11.5),
		),
	)
}

// highlightedLine 把行按匹配区间切成 普通/高亮 段。
func highlightedLine(text string, ranges [][2]int) []widget.Widget {
	if len(text) > 200 {
		text = text[:200] // 超长行截断，避免行过宽
	}
	var segs []widget.Widget
	prev := 0
	for _, r := range ranges {
		a, b := r[0], r[1]
		if a >= len(text) || a < prev {
			continue
		}
		if b > len(text) {
			b = len(text)
		}
		if a > prev {
			segs = append(segs, ui.TextC(text[prev:a], *ui.ShellTextDim, 11.5))
		}
		segs = append(segs, widget.Div(
			widget.Style{BackgroundColor: &searchHL, Padding: types.EdgeInsetsLTRB(1, 0, 1, 0)},
			ui.TextC(text[a:b], *ui.ShellText, 11.5),
		))
		prev = b
	}
	if prev < len(text) {
		segs = append(segs, ui.TextC(text[prev:], *ui.ShellTextDim, 11.5))
	}
	return segs
}

// searchToggle 选项开关（激活高亮）。
func searchToggle(text, _ string, on bool, onClick func()) widget.Widget {
	bg, tc := *ui.ShellTitle, *ui.ShellTextDim
	if on {
		bg, tc = *ui.AccentStrong, *ui.White
	}
	return &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: ui.TextC(text, tc, 11)},
		OnClick:           onClick,
		Color:             bg,
		MinWidth:          30,
		MinHeight:         22,
		Padding:           types.EdgeInsetsLTRB(6, 0, 6, 0),
	}
}

func fmtMatchStats(matches, files int) string {
	return ui.Itoa(matches) + " 处匹配 · " + ui.Itoa(files) + " 个文件"
}

// itoa 已下沉到 ui.Itoa。
