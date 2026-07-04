// Package marketplace — 市场面板 UI（GWui 版）。
// 提供对话框形式的市场浏览/搜索/安装功能。
//
//go:build windows

package marketplace

import (
	"fmt"
	"strings"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"
	"github.com/hoonfeng/gwui/uixml"

	"github.com/hoonfeng/paircode/cmd/companion/ui"
)

// ─── 数据状态 ──

type panelState struct {
	filterKind string // "" / "mcp" / "skill"
	searchText string
}

var state panelState

// ─── 打开市场面板 ──

// OpenDialog 打开市场面板对话框。
func OpenDialog() {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	state = panelState{filterKind: ""}

	modal := component.NewModal(doc)
	modal.SetTitle("市场 — MCP 服务器 & 技能")
	modal.SetMaxWidth(560)
	modal.SetMaxHeight(560)

	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()

	// 加载 HTML
	reg := uixml.NewRegistry()
	ui.MustLoadPanelHTML(doc, "panels/marketplace.html", reg)

	// Tab 切换：使用 onclick 字符串注册
	reg.OnClick("selectMarketTab(0)", func(ctx uixml.EventContext) bool {
		state.filterKind = ""
		refreshList(doc, modal)
		updateTabActive(doc, 0)
		return true
	})
	reg.OnClick("selectMarketTab(1)", func(ctx uixml.EventContext) bool {
		state.filterKind = "mcp"
		refreshList(doc, modal)
		updateTabActive(doc, 1)
		return true
	})
	reg.OnClick("selectMarketTab(2)", func(ctx uixml.EventContext) bool {
		state.filterKind = "skill"
		refreshList(doc, modal)
		updateTabActive(doc, 2)
		return true
	})

	root := doc.GetElementByID("marketplace-root")
	ui.TransferComponents(doc, doc, root)

	body.AppendChild(root)

	// 搜索输入框回车触发
	searchInput := doc.GetElementByID("marketplace-search-input")
		if ui.Ctx.App != nil {
		ui.Ctx.App.AddEventListener(searchInput, event.KeyDown, func(e event.Event) bool {
			ke := e.(*event.KeyboardEvent)
			if ke.Key == event.CodeEnter {
				el := doc.GetElementByID("marketplace-search-input")
				if el != nil {
					state.searchText = el.TextContent()
				}
				refreshList(doc, modal)
			}
			return true
		})
	}

	// 初始渲染
	refreshList(doc, modal)
	updateTabActive(doc, 0)

	modal.Show()
}

// ─── 刷新列表 ──

func refreshList(doc *dom.Document, modal *component.Modal) {
	listEl := doc.GetElementByID("marketplace-list")
	if listEl == nil {
		return
	}
	listEl.ClearChildren()

	results := Search(state.searchText, state.filterKind)

	// 更新计数
	countEl := doc.GetElementByID("marketplace-count")
	if countEl != nil {
		countEl.SetTextContent(fmt.Sprintf("%d", len(results)))
	}

	if len(results) == 0 {
		empty := doc.CreateElement("div")
		empty.SetAttribute("style", "padding: 24px; text-align: center; font-size: 13px; color: #6e6e6e;")
		if state.searchText != "" {
			empty.SetTextContent(fmt.Sprintf("未找到匹配「%s」的条目", state.searchText))
		} else {
			empty.SetTextContent("暂无可用条目")
		}
		listEl.AppendChild(empty)
		return
	}

	for _, entry := range results {
		entry := entry // capture
		item := doc.CreateElement("div")
		item.ClassList().Add("marketplace-item")

		// 图标
		icon := doc.CreateElement("div")
		icon.ClassList().Add("marketplace-item-icon")
		if entry.Kind == "mcp" {
			icon.ClassList().Add("mcp")
		} else {
			icon.ClassList().Add("skill")
		}
		icon.SetTextContent(abbrev(entry.Kind))
		item.AppendChild(icon)

		// 主体
		bodyDiv := doc.CreateElement("div")
		bodyDiv.ClassList().Add("marketplace-item-body")

		nameEl := doc.CreateElement("div")
		nameEl.ClassList().Add("marketplace-item-name")
		nameEl.SetTextContent(entry.Name)
		bodyDiv.AppendChild(nameEl)

		descEl := doc.CreateElement("div")
		descEl.ClassList().Add("marketplace-item-desc")
		descEl.SetTextContent(entry.Description)
		bodyDiv.AppendChild(descEl)

		// 标签行
		tagRow := doc.CreateElement("div")
		tagRow.SetAttribute("style", "display:flex;flex-direction:row;gap:4px;margin-top:2px;")

		if entry.Kind == "mcp" {
			tag := doc.CreateElement("span")
			tag.ClassList().Add("marketplace-item-tag")
			tag.SetTextContent(entry.Command)
			tagRow.AppendChild(tag)
		} else if entry.Activation != "" && entry.Activation != "auto" {
			tag := doc.CreateElement("span")
			tag.ClassList().Add("marketplace-item-tag")
			tag.SetTextContent(entry.Activation)
			tagRow.AppendChild(tag)
		}
		if entry.Kind == "skill" {
			tag := doc.CreateElement("span")
			tag.ClassList().Add("marketplace-item-tag")
			tag.SetTextContent("skill")
			tagRow.AppendChild(tag)
		}

		bodyDiv.AppendChild(tagRow)
		item.AppendChild(bodyDiv)

		// 安装状态 & 按钮
		installed := IsInstalled(entry.ID)
		if installed {
			tag := doc.CreateElement("span")
			tag.ClassList().Add("marketplace-item-installed-tag")
			tag.SetTextContent("已安装")
			item.AppendChild(tag)
		} else {
			btn := doc.CreateElement("div")
			btn.ClassList().Add("marketplace-install-btn")
			btn.SetTextContent("安装")
			if ui.Ctx.App != nil {
				ui.Ctx.App.AddEventListener(btn, event.Click, func(e event.Event) bool {
					InstallAndNotify(entry.ID)
					refreshList(doc, modal)
					return true
				})
			}
			item.AppendChild(btn)
		}

		listEl.AppendChild(item)
	}
}

func updateTabActive(doc *dom.Document, activeIdx int) {
	tabIDs := []string{"marketplace-tab-all", "marketplace-tab-mcp", "marketplace-tab-skill"}
	for i, id := range tabIDs {
		el := doc.GetElementByID(id)
		if el == nil {
			continue
		}
		if i == activeIdx {
			el.ClassList().Add("active")
		} else {
			el.ClassList().Remove("active")
		}
	}
}

func abbrev(kind string) string {
	switch kind {
	case "mcp":
		return "M"
	case "skill":
		return "S"
	}
	return "?"
}

// ─── URL / 说明文本 ──

// InstallHelp 返回安装帮助文本（给 agenttools 用）。
func InstallHelp() string {
	var b strings.Builder
	b.WriteString("## 市场安装帮助\n\n")
	b.WriteString("使用 marketplace_search [query] [kind] 浏览市场。\n")
	b.WriteString("使用 marketplace_install <id> 安装。\n\n")
	b.WriteString("目前可用条目：\n")
	for _, e := range Registry {
		status := ""
		if IsInstalled(e.ID) {
			status = " [已安装]"
		}
		fmt.Fprintf(&b, "- [%s] %s（%s）：%s%s\n", e.Kind, e.Name, e.ID, e.Description, status)
	}
	return b.String()
}
