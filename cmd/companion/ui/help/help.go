//go:build windows && !webonly

// Package help 提供帮助文档对话框（GWui 版）。
// 每个 Show* 函数打开一个 Modal，加载对应 HTML 模板显示格式化的文档内容。
package help

import (
	"fmt"
	"log"
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
	showHelp(ui.Ctx.Doc, "快速开始 — PairCode IDE", "getting_started.html", 760, 560)
}

// ShowFeatures 显示「功能介绍」文档。
func ShowFeatures() {
	showHelp(ui.Ctx.Doc, "功能介绍 — PairCode IDE", "features.html", 780, 580)
}

// ShowAPIDocs 显示「API 文档」。
func ShowAPIDocs() {
	showHelp(ui.Ctx.Doc, "API 文档 — PairCode IDE", "api_docs.html", 780, 580)
}

// ShowToolsDocs 显示「工具文档」—— AI 可用的所有工具分类说明。
func ShowToolsDocs() {
	showHelp(ui.Ctx.Doc, "工具文档 — PairCode IDE", "tools_docs.html", 800, 600)
}

// ShowFAQ 显示「常见问题」文档。
func ShowFAQ() {
	showHelp(ui.Ctx.Doc, "常见问题 — PairCode IDE", "faq.html", 720, 540)
}

// ShowShortcuts 显示「快捷键参考」文档。
func ShowShortcuts() {
	showHelp(ui.Ctx.Doc, "快捷键参考 — PairCode IDE", "shortcuts.html", 700, 520)
}
