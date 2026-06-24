//go:build windows

// Package menu 提供菜单动作（Agent 监控、引用/符号对话框等）（GWui 版）。
// 使用 uixml 声明式 UI 构建对话框内容。
package menu

import (
	"fmt"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"
	"github.com/hoonfeng/gwui/uixml"

	"github.com/hoonfeng/paircode/cmd/companion/codetypes"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/paircode/cmd/companion/uiapi"
)

// ── uixml 辅助 ──

// transferComponents 递归地将 srcDoc 中的组件注册转移到 dstDoc，
// 使 uixml 加载的元素能在主文档中正确处理事件。
func transferComponents(srcDoc, dstDoc *dom.Document, el *dom.Element) {
	if comp := srcDoc.ComponentAtNode(el); comp != nil {
		dstDoc.RegisterComponent(el, comp)
	}
	for _, child := range el.Children() {
		if e, ok := child.(*dom.Element); ok {
			transferComponents(srcDoc, dstDoc, e)
		}
	}
}

// adoptContent 将 xml 经 uixml 加载后的内容转移到 parent 下，
// 同时把组件注册从临时文档迁移到 dstDoc。
func adoptContent(dstDoc *dom.Document, parent *dom.Element, xml string, reg *uixml.Registry) {
	uixml.MustLoadStringInto(ui.Ctx.Doc, xml, reg)
	body := ui.Ctx.Doc.Body()
	for {
		c := body.FirstChild()
		if c == nil {
			break
		}
		body.RemoveChild(c)
		if el, ok := c.(*dom.Element); ok {
			transferComponents(ui.Ctx.Doc, dstDoc, el)
		}
		parent.AppendChild(c)
	}
}

// ── 导出函数 ──

// ShowAgentMonitor 显示 Agent 监控面板。
func ShowAgentMonitor() {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	modal := component.NewModal(doc)
	modal.SetTitle("Agent 监控")

	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()
	body.SetAttribute("style",
		"display: flex; flex-direction: column; gap: 8px; "+
			"min-width: 400px; min-height: 200px;")

	reg := uixml.NewRegistry()
	reg.OnClick("closeMonitor", func(ctx uixml.EventContext) bool {
		modal.Hide()
		return true
	})

	const xmlUI = `<div style="font-size:16px;color:#616161;padding:16px 0;text-align:center">Agent 监控（暂无运行中的 Agent）</div>
<button label="关闭" onclick="closeMonitor" style="align-self:flex-end"/>`

	adoptContent(doc, body, xmlUI, reg)
	modal.Show()
}

// ShowContentDialog 显示自定义内容对话框。
func ShowContentDialog(title string, width float32, content *dom.Element) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	modal := component.NewModal(doc)
	modal.SetTitle(title)

	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()

	widthStr := fmt.Sprintf("%.0fpx", width)
	body.SetAttribute("style",
		"display: flex; flex-direction: column; "+
			"min-width: "+widthStr+"; max-width: "+widthStr+";")

	// 将外部传入的内容元素直接追加（已在主文档中）
	if content != nil {
		body.AppendChild(content)
	}

	modal.Show()
}

// EditorReferences 显示引用结果，点击跳转到编辑器。
func EditorReferences(refs []codetypes.CodeLoc) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	modal := component.NewModal(doc)
	modal.SetTitle(fmt.Sprintf("引用 (%d)", len(refs)))

	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()
	body.SetAttribute("style",
		"display: flex; flex-direction: column; gap: 4px; "+
			"min-width: 500px; max-height: 400px; overflow-y: auto;")

	var containerEl *dom.Element

	reg := uixml.NewRegistry()
	reg.OnClick("closeRefs", func(ctx uixml.EventContext) bool {
		modal.Hide()
		return true
	})

	const xmlShell = `<div id="refsContainer" style="display:flex;flex-direction:column;gap:4px"></div>
<button label="关闭" onclick="closeRefs" style="margin-top:8px;align-self:flex-end"/>`

	// 加载 XML，捕获容器引用，然后迁移到主文档
	uixml.MustLoadStringInto(ui.Ctx.Doc, xmlShell, reg)
	containerEl = ui.Ctx.Doc.GetElementByID("refsContainer")

	srcBody := ui.Ctx.Doc.Body()
	for {
		c := srcBody.FirstChild()
		if c == nil {
			break
		}
		srcBody.RemoveChild(c)
		if el, ok := c.(*dom.Element); ok {
			transferComponents(ui.Ctx.Doc, doc, el)
		}
		body.AppendChild(c)
	}

	// 动态构建条目
	if len(refs) == 0 {
		emptyEl := doc.CreateElement("div")
		emptyEl.SetAttribute("style",
			"font-size: 16px; color: #9e9e9e; padding: 16px 0; text-align: center;")
		emptyEl.SetTextContent("未找到引用")
		containerEl.AppendChild(emptyEl)
	} else {
		for _, ref := range refs {
			ref := ref
			item := doc.CreateElement("div")
			item.SetAttribute("style",
				"display: flex; flex-direction: row; gap: 8px; "+
					"padding: 6px 8px; cursor: pointer; border-radius: 4px; "+
					"font-size: 15px; color: #212121;")
			item.SetAttribute("hover-style", "background-color: #f5f5f5;")
			item.SetTextContent(fmt.Sprintf("%s:%d:%d", ref.File, ref.Line, ref.Col))
			if ui.Ctx.App != nil {
				path, line := ref.File, ref.Line
				ui.Ctx.App.AddEventListener(item, event.Click, func(e event.Event) bool {
					modal.Hide()
					if ui.Ctx.Editor != nil {
						ui.Ctx.Editor.OpenAt(path, line)
					}
					return true
				})
			}
			containerEl.AppendChild(item)
		}
	}

	modal.Show()
}

// EditorSymbols 显示符号大纲，点击跳转到编辑器。
func EditorSymbols(syms []codetypes.CodeSym) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	modal := component.NewModal(doc)
	modal.SetTitle(fmt.Sprintf("符号 (%d)", len(syms)))

	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()
	body.SetAttribute("style",
		"display: flex; flex-direction: column; gap: 4px; "+
			"min-width: 400px; max-height: 400px; overflow-y: auto;")

	var containerEl *dom.Element

	reg := uixml.NewRegistry()
	reg.OnClick("closeSyms", func(ctx uixml.EventContext) bool {
		modal.Hide()
		return true
	})

	const xmlShell = `<div id="symsContainer" style="display:flex;flex-direction:column;gap:4px"></div>
<button label="关闭" onclick="closeSyms" style="margin-top:8px;align-self:flex-end"/>`

	uixml.MustLoadStringInto(ui.Ctx.Doc, xmlShell, reg)
	containerEl = ui.Ctx.Doc.GetElementByID("symsContainer")

	srcBody := ui.Ctx.Doc.Body()
	for {
		c := srcBody.FirstChild()
		if c == nil {
			break
		}
		srcBody.RemoveChild(c)
		if el, ok := c.(*dom.Element); ok {
			transferComponents(ui.Ctx.Doc, doc, el)
		}
		body.AppendChild(c)
	}

	if len(syms) == 0 {
		emptyEl := doc.CreateElement("div")
		emptyEl.SetAttribute("style",
			"font-size: 16px; color: #9e9e9e; padding: 16px 0; text-align: center;")
		emptyEl.SetTextContent("未找到符号")
		containerEl.AppendChild(emptyEl)
	} else {
		for _, sym := range syms {
			sym := sym
			item := doc.CreateElement("div")
			indent := sym.Depth * 16
			item.SetAttribute("style",
				fmt.Sprintf("padding: 4px 8px 4px %dpx; cursor: pointer; "+
					"font-size: 15px; color: #212121; border-radius: 4px;", indent+8))
			item.SetAttribute("hover-style", "background-color: #f5f5f5;")
			item.SetTextContent(fmt.Sprintf("%s (行 %d)", sym.Name, sym.Line))
			if ui.Ctx.App != nil {
				ui.Ctx.App.AddEventListener(item, event.Click, func(e event.Event) bool {
					modal.Hide()
					return true
				})
			}
			containerEl.AppendChild(item)
		}
	}

	modal.Show()
}

// EditorFontSize 获取编辑器字号（0=默认 14）。
func EditorFontSize() int {
	if core.Settings.EditorFontSize <= 0 {
		return 14
	}
	return core.Settings.EditorFontSize
}

// SetEditorFontSize 设置编辑器字号并保存。
func SetEditorFontSize(sz int) {
	core.Settings.EditorFontSize = sz
	core.Save()
	uiapi.MarkDirty()
}

// Relayout 触发重布局。
func Relayout() {
	uiapi.MarkDirty()
}
