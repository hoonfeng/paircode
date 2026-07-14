// Package marketplace — 市场面板 UI（GWui 版）。
// 提供对话框形式的市场浏览/搜索/安装功能。
//
//go:build windows && !webonly

package marketplace

import (
	"fmt"
	"strings"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"
	"github.com/hoonfeng/gwui/uixml"

	"github.com/hoonfeng/paircode/cmd/companion/ui"
	mcppanel "github.com/hoonfeng/paircode/cmd/companion/ui/mcp"
	skillspanel "github.com/hoonfeng/paircode/cmd/companion/ui/skills"
)

// ─── 数据状态 ──

type panelState struct {
	tabIndex   int    // 0=全部 1=MCP 2=技能 3=已安装
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
	state = panelState{filterKind: "", tabIndex: 0}

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
		state.tabIndex = 0
		refreshList(doc, modal)
		updateTabActive(doc, 0)
		return true
	})
	reg.OnClick("selectMarketTab(1)", func(ctx uixml.EventContext) bool {
		state.filterKind = "mcp"
		state.tabIndex = 1
		refreshList(doc, modal)
		updateTabActive(doc, 1)
		return true
	})
	reg.OnClick("selectMarketTab(2)", func(ctx uixml.EventContext) bool {
		state.filterKind = "skill"
		state.tabIndex = 2
		refreshList(doc, modal)
		updateTabActive(doc, 2)
		return true
	})
	reg.OnClick("selectMarketTab(3)", func(ctx uixml.EventContext) bool {
		state.tabIndex = 3
		refreshList(doc, modal)
		updateTabActive(doc, 3)
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

	// 已安装 tab 走独立渲染
	if state.tabIndex == 3 {
		refreshInstalled(doc, modal, listEl)
		return
	}

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
	tabIDs := []string{"marketplace-tab-all", "marketplace-tab-mcp", "marketplace-tab-skill", "marketplace-tab-installed"}
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

// refreshInstalled 渲染「已安装」标签页的内容。
func refreshInstalled(doc *dom.Document, modal *component.Modal, listEl *dom.Element) {
	listEl.ClearChildren()

	// 收集已安装的 MCP 和技能
	type installedItem struct {
		kind  string // "mcp" / "skill"
		name  string
		desc  string
		level string // "user" / "project"
	}
	var items []installedItem

	for _, e := range mcppanel.ReadLevel(mcppanel.LevelUser) {
		items = append(items, installedItem{kind: "mcp", name: e.Name, desc: "MCP 服务器（用户级）", level: "user"})
	}
	for _, e := range mcppanel.ReadLevel(mcppanel.LevelProject) {
		items = append(items, installedItem{kind: "mcp", name: e.Name, desc: "MCP 服务器（工作区级）", level: "project"})
	}
	for _, e := range skillspanel.ReadLevel(skillspanel.LevelUser) {
		desc := e.Description
		if desc == "" {
			desc = "技能（用户级）"
		}
		items = append(items, installedItem{kind: "skill", name: e.Name, desc: desc, level: "user"})
	}
	for _, e := range skillspanel.ReadLevel(skillspanel.LevelProject) {
		desc := e.Description
		if desc == "" {
			desc = "技能（工作区级）"
		}
		items = append(items, installedItem{kind: "skill", name: e.Name, desc: desc, level: "project"})
	}

	// 更新计数
	countEl := doc.GetElementByID("marketplace-count")
	if countEl != nil {
		countEl.SetTextContent(fmt.Sprintf("%d", len(items)))
	}

	if len(items) == 0 {
		empty := doc.CreateElement("div")
		empty.SetAttribute("style", "padding: 24px; text-align: center; font-size: 13px; color: #6e6e6e;")
		empty.SetTextContent("暂无已安装的 MCP 服务器或技能")
		listEl.AppendChild(empty)
		return
	}

	for _, it := range items {
		it := it
		item := doc.CreateElement("div")
		item.ClassList().Add("marketplace-item")

		// 图标
		icon := doc.CreateElement("div")
		icon.ClassList().Add("marketplace-item-icon")
		if it.kind == "mcp" {
			icon.ClassList().Add("mcp")
		} else {
			icon.ClassList().Add("skill")
		}
		icon.SetTextContent(abbrev(it.kind))
		item.AppendChild(icon)

		// 主体
		bodyDiv := doc.CreateElement("div")
		bodyDiv.ClassList().Add("marketplace-item-body")

		nameEl := doc.CreateElement("div")
		nameEl.ClassList().Add("marketplace-item-name")
		nameEl.SetTextContent(it.name)
		bodyDiv.AppendChild(nameEl)

		descEl := doc.CreateElement("div")
		descEl.ClassList().Add("marketplace-item-desc")
		descEl.SetTextContent(it.desc)
		bodyDiv.AppendChild(descEl)

		item.AppendChild(bodyDiv)

		// 卸载按钮
		btn := doc.CreateElement("div")
		btn.ClassList().Add("marketplace-install-btn")
		btn.SetAttribute("style", "background:#5a1d1d;color:#f48771;")
		btn.SetTextContent("卸载")
		if ui.Ctx.App != nil {
			ui.Ctx.App.AddEventListener(btn, event.Click, func(e event.Event) bool {
				uninstallItem(it.kind, it.name)
				refreshInstalled(doc, modal, listEl)
				return true
			})
		}
		item.AppendChild(btn)

		listEl.AppendChild(item)
	}
}

// uninstallItem 卸载已安装的 MCP 或技能。
func uninstallItem(kind, name string) {
	switch kind {
	case "mcp":
		// MCP 可能安装在用户级或项目级，先尝试用户级再尝试项目级
		if err := mcppanel.Delete(mcppanel.LevelUser, name); err != nil {
			mcppanel.Delete(mcppanel.LevelProject, name)
		}
	case "skill":
		skillspanel.Delete(skillspanel.LevelProject, name)
	}
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
