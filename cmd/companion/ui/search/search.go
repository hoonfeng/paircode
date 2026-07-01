//go:build windows

// Package search 提供 GWui 版搜索面板。
// 使用 uixml 声明式 UI 构建面板布局，保留 Go 逻辑处理搜索/渲染。
package search

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"
	"github.com/hoonfeng/gwui/uixml"

	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
)

// SearchResult 搜索结果。
type SearchResult struct {
	Path string
	Line int
	Text string
}

// SearchPanel 搜索面板。
type SearchPanel struct {
	doc       *dom.Document
	root      *dom.Element
	inputComp *component.Input
	content   *dom.Element
	results   []SearchResult
	keyword   string
	lastInput string
}

// New 创建搜索面板。
func New(doc *dom.Document) *SearchPanel {
	p := &SearchPanel{doc: doc}

	// 创建真实 Input 组件（主文档上创建，支持 Enter 搜索）
	inputComp := component.NewInput(doc, "搜索...")
	inputComp.SetBaseStyle(
		"background-color: " + ui.InputBg + "; color: " + ui.Text + "; " +
			"border: 1px solid " + ui.Border + "; padding: 4px 8px; font-size: 13px; flex: 1;")
	inputComp.OnChange(func(v string) {
		if v == p.lastInput {
			if v != "" {
				p.doSearch(v)
			}
		} else {
			p.lastInput = v
		}
	})

	// uixml 注册表：搜索按钮点击
	reg := uixml.NewRegistry()
	reg.OnClick("onSearch", func(ctx uixml.EventContext) bool {
		p.doSearch(inputComp.Value())
		return true
	})

	// 加载 HTML 模板（资源目录 html/panels/search.html）
	ui.MustLoadPanelHTML(doc, "panels/search.html", reg)
	p.root = doc.GetElementByID("search-root")
	p.content = doc.GetElementByID("search-result-container")

	// 用真实 Input 替换占位 div
	ui.ReplaceChildByID(doc, "search-input-placeholder", inputComp.Element())

	// 转移组件注册到主文档（HTML 加载可能在临时文档上下文，确保组件归属正确）
	if p.root != nil {
		ui.TransferComponents(doc, doc, p.root)
	}

	// 从临时父节点（body）中分离根元素
	ui.DetachRoot(p.root)

	p.inputComp = inputComp
	p.renderResults()
	return p
}

// Element 返回面板根元素。
func (p *SearchPanel) Element() *dom.Element { return p.root }

// Refresh 重新渲染结果列表。
func (p *SearchPanel) Refresh() {
	p.renderResults()
}

// doSearch 执行搜索。
func (p *SearchPanel) doSearch(keyword string) {
	p.keyword = keyword
	p.results = nil

	if keyword == "" {
		p.renderResults()
		return
	}

	// 遍历工作区所有文件
	for _, folder := range core.Folders {
		stopped := false
		filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			if stopped {
				return filepath.SkipDir
			}
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if info.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			// 只搜索文本文件，跳过大文件
			if !isTextFile(path) || info.Size() > 1024*1024 {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				if strings.Contains(line, keyword) {
					p.results = append(p.results, SearchResult{
						Path: path,
						Line: i + 1,
						Text: strings.TrimSpace(line),
					})
					if len(p.results) >= 200 {
						stopped = true
						return filepath.SkipDir
					}
				}
			}
			return nil
		})
	}

	p.renderResults()
}

// renderResults 渲染搜索结果。
func (p *SearchPanel) renderResults() {
	p.content.ClearChildren()

	if p.keyword == "" {
		hint := p.doc.CreateElement("div")
		hint.SetAttribute("style", "padding: 12px; color: "+ui.TextDim+"; font-size: 13px;")
		hint.SetTextContent("输入搜索关键词")
		p.content.AppendChild(hint)
	}
	if len(p.results) == 0 && p.lastInput != "" {
		hint := p.doc.CreateElement("div")
		hint.SetAttribute("style", "padding: 12px; color: "+ui.TextDim+"; font-size: 13px;")
		hint.SetTextContent("无结果")
		p.content.AppendChild(hint)
		return
	}

	// 结果计数
	count := p.doc.CreateElement("div")
	count.SetAttribute("style",
		"padding: 4px 12px; color: "+ui.TextDim+"; font-size: 13px; "+
			"border-bottom: 1px solid "+ui.Border+";")
	count.SetTextContent(strconv.Itoa(len(p.results)) + " 个结果")
	p.content.AppendChild(count)

	for _, r := range p.results {
		row := p.doc.CreateElement("div")
		row.ClassList().Add("search-result")

		// 文件名:行号
		fileEl := p.doc.CreateElement("div")
		fileEl.ClassList().Add("file")
		fileEl.SetTextContent(r.Path + ":" + strconv.Itoa(r.Line))
		row.AppendChild(fileEl)

		// 匹配内容
		textEl := p.doc.CreateElement("div")
		textEl.ClassList().Add("line")
		textEl.SetTextContent(r.Text)
		row.AppendChild(textEl)

		// 点击跳转到文件
		path := r.Path
		line := r.Line
		on(row, event.Click, func(e event.Event) bool {
			if ui.Ctx.Editor != nil {
				ui.Ctx.Editor.OpenAt(path, line)
			}
			return true
		})

		p.content.AppendChild(row)
	}
}

// isTextFile 判断文件是否是文本文件（按扩展名）。
func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".js", ".ts", ".tsx", ".jsx", ".py", ".java", ".c", ".cpp", ".h", ".hpp",
		".cs", ".rs", ".rb", ".php", ".swift", ".kt", ".scala", ".sh", ".bat", ".ps1",
		".json", ".xml", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".conf", ".env",
		".md", ".txt", ".html", ".htm", ".css", ".scss", ".less", ".sql", ".lua",
		".vue", ".svelte", ".mod", ".sum":
		return true
	}
	// 无扩展名的已知文本文件
	name := strings.ToLower(filepath.Base(path))
	switch name {
	case "makefile", "dockerfile", "readme", "license", ".gitignore", ".env":
		return true
	}
	return false
}

// on 注册事件监听器（通过全局 App）。
func on(el *dom.Element, typ event.Type, fn func(event.Event) bool) {
	if ui.Ctx.App != nil {
		ui.Ctx.App.AddEventListener(el, typ, fn)
	}
}
