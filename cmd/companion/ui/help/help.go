//go:build windows && !webonly

// Package help 提供帮助文档对话框（GWui 版）。
// 每个 Show* 函数打开一个 Modal，加载对应 HTML 模板显示格式化的文档内容。
package help

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/uixml"

	"github.com/hoonfeng/paircode/cmd/companion/ui"
)

// showHelp 通用帮助文档显示函数。
// htmlFile 为 resources/html/help/ 下的 HTML 文件名（不含路径分隔符）。
func showHelp(doc *dom.Document, title, htmlFile string, width, height float32) {
	if doc == nil {
		log.Println("[help] doc 为空，无法显示帮助")
		return
	}

	// ── 安全检查：防止路径遍历 ──
	if !validateHelpPath(htmlFile) {
		log.Printf("[help] 路径校验失败：%q", htmlFile)
		return
	}

	modal := component.NewModal(doc)
	modal.SetTitle(title)
	modal.SetMaxWidth(width)
	modal.SetMaxHeight(height)

	body := modal.Content()
	if body == nil {
		log.Println("[help] modal body 为空")
		return
	}
	body.ClearChildren()
	body.SetAttribute("style",
		"display: flex; flex-direction: column; gap: 4px; "+
			"min-width: "+fmt.Sprintf("%.0fpx", width-40)+"; "+
			"max-height: "+fmt.Sprintf("%.0fpx", height-60)+"; "+
			"overflow-y: auto;")

	reg := uixml.NewRegistry()
	reg.OnClick("closeHelp", func(ctx uixml.EventContext) bool {
		modal.Hide()
		return true
	})
	reg.OnClick("exportHelp", func(ctx uixml.EventContext) bool {
		exportHelpDoc(htmlFile, title)
		return true
	})

	relPath := "help/" + htmlFile
	fullPath := ui.ResourcePath("html/" + relPath)

	// 再次校验：确保最终路径在 resources/html/help/ 下
	absHelpDir, _ := filepath.Abs(ui.ResourcePath("html/help"))
	absTarget, err := filepath.Abs(fullPath)
	if err != nil || !strings.HasPrefix(absTarget, absHelpDir) {
		log.Printf("[help] 路径越界：%s → %s", relPath, absTarget)
		return
	}

	err = ui.LoadPanelHTML(doc, relPath, reg)
	if err != nil {
		log.Printf("[help] 加载模板失败 %s: %v", relPath, err)
		// 显示错误占位
		errEl := doc.CreateElement("div")
		errEl.SetAttribute("style",
			"padding: 24px; text-align: center; font-size: 16px; color: #f48771;")
		errEl.SetTextContent("无法加载文档：" + err.Error())
		body.AppendChild(errEl)
		modal.Show()
		return
	}

	ui.AdoptBodyChildren(doc, body)
	log.Printf("[help] 已打开：%s (%s)", title, relPath)
	modal.Show()
}

// exportHelpDoc 将帮助文档导出为 Markdown 文件。
func exportHelpDoc(htmlFile, title string) {
	// 读取原始 HTML 文件
	data, err := ui.ReadResource("html/help/" + htmlFile)
	if err != nil {
		log.Printf("[help] 导出失败：读取文件错误 %v", err)
		return
	}

	// 转换为 Markdown
	md := helpHTMLToMarkdown(string(data), title)

	// 弹出保存对话框
	mdName := strings.TrimSuffix(htmlFile, ".html") + ".md"
	savePath := ui.SaveFileDialog("导出帮助文档", "Markdown 文件 (*.md)|*.md", mdName)
	if savePath == "" {
		return // 用户取消了
	}

	if err := os.WriteFile(savePath, []byte(md), 0o644); err != nil {
		log.Printf("[help] 导出失败：写入文件错误 %v", err)
		return
	}
	log.Printf("[help] 已导出：%s → %s", title, savePath)
}

// helpHTMLToMarkdown 将帮助文档的 HTML 内容简单转换为 Markdown 格式。
func helpHTMLToMarkdown(html, title string) string {
	var b strings.Builder
	b.WriteString("# " + title + "\n\n")

	lines := strings.Split(html, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 跳过 HTML 注释和包装 div
		if strings.HasPrefix(line, "<!--") || strings.HasPrefix(line, "<div") || line == "</div>" {
			continue
		}
		// 跳过关闭按钮和导出按钮
		if strings.Contains(line, "closeHelp") || strings.Contains(line, "exportHelp") || strings.Contains(line, "</button>") {
			continue
		}

		// h2 标题
		if strings.HasPrefix(line, "<h2") {
			start := strings.Index(line, ">")
			end := strings.LastIndex(line, "</h2>")
			if start != -1 && end != -1 && end > start {
				text := strings.TrimSpace(line[start+1 : end])
				b.WriteString("## " + text + "\n\n")
			}
			continue
		}

		// h3 标题
		if strings.HasPrefix(line, "<h3") {
			start := strings.Index(line, ">")
			end := strings.LastIndex(line, "</h3>")
			if start != -1 && end != -1 && end > start {
				text := strings.TrimSpace(line[start+1 : end])
				b.WriteString("### " + text + "\n\n")
			}
			continue
		}

		// 段落 <p>
		if strings.HasPrefix(line, "<p") {
			start := strings.Index(line, ">")
			end := strings.LastIndex(line, "</p>")
			if start != -1 && end != -1 && end > start {
				text := stripTags(line[start+1 : end])
				text = strings.TrimSpace(text)
				if text != "" {
					b.WriteString(text + "\n\n")
				}
			}
			continue
		}

		// pre/code 块
		if strings.HasPrefix(line, "<pre") {
			start := strings.Index(line, ">")
			end := strings.LastIndex(line, "</pre>")
			if start != -1 && end != -1 && end > start {
				code := line[start+1 : end]
				code = strings.ReplaceAll(code, "&lt;", "<")
				code = strings.ReplaceAll(code, "&gt;", ">")
				code = strings.ReplaceAll(code, "&amp;", "&")
				b.WriteString("```\n" + code + "\n```\n\n")
			}
			continue
		}

		// 表格行
		if strings.HasPrefix(line, "<tr") {
			continue
		}
		if strings.HasPrefix(line, "<th") || strings.HasPrefix(line, "<td") {
			text := stripTags(line)
			text = strings.TrimSpace(text)
			if text != "" {
				if strings.HasPrefix(line, "<th") {
					b.WriteString("| **" + text + "** ")
				} else if strings.HasPrefix(line, "<td") {
					b.WriteString("| " + text + " ")
				}
			}
			continue
		}
		if line == "</tr>" || line == "</table>" || line == "</thead>" || strings.HasPrefix(line, "<table") {
			if line == "</tr>" {
				b.WriteString("|\n")
			}
			if line == "</table>" || line == "</thead>" {
				b.WriteString("\n")
			}
			continue
		}

		// 无序列表
		if strings.HasPrefix(line, "<li") {
			start := strings.Index(line, ">")
			end := strings.LastIndex(line, "</li>")
			if start != -1 && end != -1 && end > start {
				text := stripTags(line[start+1 : end])
				b.WriteString("- " + strings.TrimSpace(text) + "\n")
			}
			continue
		}

		// code 内联
		if strings.Contains(line, "<code") {
			text := stripTags(line)
			b.WriteString(strings.TrimSpace(text) + "\n\n")
			continue
		}

		// 分隔线
		if strings.Contains(line, "<hr") || strings.Contains(line, "height:1px") {
			b.WriteString("---\n\n")
			continue
		}
	}

	return b.String()
}

// stripTags 移除简单的 HTML 标签。
func stripTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	// 解码 HTML 实体
	text := result.String()
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	return text
}

// validateHelpPath 校验 htmlFile 是否合法：禁止空、包含路径分隔符、路径遍历。
func validateHelpPath(htmlFile string) bool {
	if htmlFile == "" {
		return false
	}
	// 禁止包含路径分隔符
	if strings.ContainsAny(htmlFile, `/\`) {
		return false
	}
	// 禁止包含路径遍历模式
	if strings.Contains(htmlFile, "..") {
		return false
	}
	// 只允许 .html 扩展名
	if !strings.HasSuffix(htmlFile, ".html") {
		return false
	}
	// 禁止隐藏文件或特殊字符
	if strings.HasPrefix(htmlFile, ".") {
		return false
	}
	return true
}

// ShowGettingStarted 显示「快速开始」文档。
func ShowGettingStarted() {
	showHelp(ui.Ctx.Doc, "快速开始 — PairCode IDE", "getting_started.html", 920, 660)
}

// ShowFeatures 显示「功能介绍」文档。
func ShowFeatures() {
	showHelp(ui.Ctx.Doc, "功能介绍 — PairCode IDE", "features.html", 940, 670)
}

// ShowAPIDocs 显示「API 文档」。
func ShowAPIDocs() {
	showHelp(ui.Ctx.Doc, "API 文档 — PairCode IDE", "api_docs.html", 940, 670)
}

// ShowToolsDocs 显示「工具文档」—— AI 可用的所有工具分类说明。
func ShowToolsDocs() {
	showHelp(ui.Ctx.Doc, "工具文档 — PairCode IDE", "tools_docs.html", 960, 680)
}

// ShowFAQ 显示「常见问题」文档。
func ShowFAQ() {
	showHelp(ui.Ctx.Doc, "常见问题 — PairCode IDE", "faq.html", 880, 650)
}

// ShowShortcuts 显示「快捷键参考」文档。
func ShowShortcuts() {
	showHelp(ui.Ctx.Doc, "快捷键参考 — PairCode IDE", "shortcuts.html", 860, 640)
}
