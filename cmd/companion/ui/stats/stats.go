// 项目级 Token/上下文统计面板 —— 以 Modal 对话框展示持久化的项目使用统计。
// 数据来源：core.ProjectStats（存储于安装目录 .pair/stats.json，独立于对话记录）。
//
//go:build windows

package stats

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"

	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/paircode/cmd/companion/uiapi"
)

// ─── 辅助函数 ─────────────────────────────────────────────

func comma(n int) string {
	if n == 0 {
		return "0"
	}
	s := fmt.Sprintf("%d", n)
	parts := make([]string, 0, len(s)/3+1)
	for i := len(s); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		parts = append([]string{s[start:i]}, parts...)
	}
	return strings.Join(parts, ",")
}

func fmtDuration(start, last time.Time) string {
	if start.IsZero() || last.IsZero() {
		return "—"
	}
	days := int(last.Sub(start).Hours() / 24)
	if days < 1 {
		return "今天"
	}
	if days == 1 {
		return "1 天"
	}
	return fmt.Sprintf("%d 天", days)
}

// ─── DOM 辅助 ──────────────────────────────────────────────

func txt(doc dom.Document, text, style string) *dom.Element {
	el := doc.CreateElement("div")
	el.SetAttribute("style", style)
	el.SetText(text)
	return el
}

func header(doc dom.Document, text string) *dom.Element {
	return txt(doc, text, "font-size:14px;font-weight:bold;color:"+ui.Text+";margin-top:16px;margin-bottom:8px;")
}

func div(doc dom.Document, style string) *dom.Element {
	el := doc.CreateElement("div")
	el.SetAttribute("style", style)
	return el
}

func addClick(el *dom.Element, fn func()) {
	if ui.Ctx.App == nil || fn == nil {
		return
	}
	ui.Ctx.App.AddEventListener(el, event.Click, func(e event.Event) bool {
		fn()
		return true
	})
}

// ─── 环形图 ────────────────────────────────────────────────

type donutCat struct {
	Label string
	Value int
	Color string
}

func donutSection(doc dom.Document, label string, used, max int, cats []donutCat) *dom.Element {
	wrapper := div(doc, "display:flex;align-items:center;gap:16px;margin:8px 0;")

	// SVG 环形图
	svg := doc.CreateElement("svg")
	svg.SetAttribute("width", "80")
	svg.SetAttribute("height", "80")
	svg.SetAttribute("viewBox", "0 0 80 80")

	bgCircle := doc.CreateElement("circle")
	bgCircle.SetAttribute("cx", "40")
	bgCircle.SetAttribute("cy", "40")
	bgCircle.SetAttribute("r", "32")
	bgCircle.SetAttribute("fill", "none")
	bgCircle.SetAttribute("stroke", ui.HoverBg)
	bgCircle.SetAttribute("stroke-width", "6")
	svg.AppendChild(bgCircle)

	const r, cx, cy = 32, 40, 40
	total := 0
	for _, c := range cats {
		total += c.Value
	}
	if total > 0 {
		var startAngle float64 = -90
		for _, c := range cats {
			if c.Value <= 0 {
				continue
			}
			angle := float64(c.Value) / float64(total) * 360
			endAngle := startAngle + angle
			large := 0
			if angle > 180 {
				large = 1
			}
			sx := cx + r*math.Cos(startAngle*math.Pi/180)
			sy := cy + r*math.Sin(startAngle*math.Pi/180)
			ex := cx + r*math.Cos(endAngle*math.Pi/180)
			ey := cy + r*math.Sin(endAngle*math.Pi/180)

			arc := doc.CreateElement("path")
			d := fmt.Sprintf("M %.1f %.1f A %d %d 0 %d 1 %.1f %.1f", sx, sy, r, r, large, ex, ey)
			arc.SetAttribute("d", d)
			arc.SetAttribute("fill", "none")
			arc.SetAttribute("stroke", c.Color)
			arc.SetAttribute("stroke-width", "6")
			svg.AppendChild(arc)

			startAngle = endAngle
		}
	}

	pct := 0
	if max > 0 {
		pct = used * 100 / max
		if pct > 100 {
			pct = 100
		}
	}
	centerText := doc.CreateElement("text")
	centerText.SetAttribute("x", "40")
	centerText.SetAttribute("y", "42")
	centerText.SetAttribute("text-anchor", "middle")
	centerText.SetAttribute("fill", ui.Text)
	centerText.SetAttribute("font-size", "14")
	centerText.SetAttribute("font-weight", "bold")
	centerText.SetText(fmt.Sprintf("%d%%", pct))
	svg.AppendChild(centerText)

	wrapper.AppendChild(svg)

	info := div(doc, "display:flex;flex-direction:column;gap:2px;")
	info.AppendChild(txt(doc, label, "font-size:12px;color:"+ui.TextDim+";"))
	info.AppendChild(txt(doc, fmt.Sprintf("%s / %s", comma(used), comma(max)), "font-size:13px;color:"+ui.Text+";font-weight:bold;"))
	info.AppendChild(txt(doc, fmt.Sprintf("已用 %d%%", pct), "font-size:11px;color:"+ui.TextMute+";"))

	legend := div(doc, "display:flex;flex-wrap:wrap;gap:4px 12px;margin-top:4px;")
	for _, c := range cats {
		if c.Value <= 0 {
			continue
		}
		item := div(doc, "display:flex;align-items:center;gap:4px;")
		dot := div(doc, fmt.Sprintf("width:8px;height:8px;border-radius:50%%;background:%s;flex-shrink:0;", c.Color))
		item.AppendChild(dot)
		item.AppendChild(txt(doc, fmt.Sprintf("%s %s", c.Label, comma(c.Value)), "font-size:11px;color:"+ui.TextDim+";"))
		legend.AppendChild(item)
	}
	info.AppendChild(legend)
	wrapper.AppendChild(info)

	return wrapper
}

// ─── 面板入口 ──────────────────────────────────────────────

// ShowStatsDialog 打开项目统计面板 Modal。
func ShowStatsDialog() {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}

	modal := component.NewModal(doc)
	modal.SetTitle("项目统计")
	modal.SetMaxWidth(480)
	modal.SetMaxHeight(560)

	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()

	ps := core.GetProjectStatsSnapshot()

	container := doc.CreateElement("div")
	container.SetAttribute("style", "padding:12px 16px;overflow-y:auto;max-height:480px;")

	// ── 概览卡片 ──
	overview := div(doc, "display:grid;grid-template-columns:1fr 1fr 1fr;gap:8px;margin-bottom:12px;")
	addCard := func(label, value string) {
		card := div(doc, fmt.Sprintf("background:%s;border-radius:4px;padding:8px 12px;text-align:center;", ui.InputBg))
		card.AppendChild(txt(doc, value, "font-size:18px;font-weight:bold;color:"+ui.Text+";"))
		card.AppendChild(txt(doc, label, "font-size:11px;color:"+ui.TextMute+";margin-top:2px;"))
		overview.AppendChild(card)
	}
	addCard("总 Token", comma(ps.TotalTokens))
	addCard("LLM 调用", comma(ps.TotalLLMCalls))
	addCard("总轮次", comma(ps.TotalTurns))
	addCard("工具调用", comma(ps.TotalToolCalls))
	addCard("输入", comma(ps.TotalPromptTokens))
	addCard("输出", comma(ps.TotalCompletionTokens))
	container.AppendChild(overview)

	// ── Token 分布 ──
	container.AppendChild(header(doc, "Token 分布"))
	totalTokens := ps.TotalTokens
	if totalTokens > 0 {
		distCats := []donutCat{
			{Label: "输入(Prompt)", Value: ps.TotalPromptTokens, Color: ui.Info},
			{Label: "输出(Completion)", Value: ps.TotalCompletionTokens, Color: ui.Success},
		}
		container.AppendChild(donutSection(doc, "Token 分布", totalTokens, totalTokens+1000, distCats))
	} else {
		container.AppendChild(txt(doc, "尚无数据，开始使用 LLM 后将自动统计", "font-size:12px;color:"+ui.TextMute+";"))
	}

	// ── 缓存统计 ──
	container.AppendChild(header(doc, "缓存效率"))
	totalCache := ps.TotalCacheHitTokens + ps.TotalCacheMissTokens
	if totalCache > 0 {
		hitRate := float64(ps.TotalCacheHitTokens) / float64(totalCache) * 100
		row := div(doc, "display:flex;gap:16px;align-items:center;")
		row.AppendChild(txt(doc, fmt.Sprintf("命中率: %.1f%%", hitRate), "font-size:14px;font-weight:bold;color:"+ui.Success+";"))
		row.AppendChild(txt(doc, fmt.Sprintf("命中: %s", comma(ps.TotalCacheHitTokens)), "font-size:12px;color:"+ui.TextDim+";"))
		row.AppendChild(txt(doc, fmt.Sprintf("未命中: %s", comma(ps.TotalCacheMissTokens)), "font-size:12px;color:"+ui.TextDim+";"))
		container.AppendChild(row)
	} else {
		container.AppendChild(txt(doc, "缓存数据将在下次 LLM 调用后显示", "font-size:12px;color:"+ui.TextMute+";"))
	}

	// ── 按模型细分 ──
	if len(ps.PerModel) > 0 {
		container.AppendChild(header(doc, "按模型统计"))
		tbl := div(doc, "display:flex;flex-direction:column;gap:3px;")
		hdr := div(doc, fmt.Sprintf("display:flex;gap:8px;padding:4px 8px;background:%s;border-radius:2px;font-size:11px;color:%s;", ui.PanelHeader, ui.TextMute))
		hdr.AppendChild(txt(doc, "模型", "flex:1;"))
		hdr.AppendChild(txt(doc, "调用", "width:45px;text-align:right;"))
		hdr.AppendChild(txt(doc, "Prompt", "width:65px;text-align:right;"))
		hdr.AppendChild(txt(doc, "Completion", "width:65px;text-align:right;"))
		hdr.AppendChild(txt(doc, "总计", "width:65px;text-align:right;"))
		tbl.AppendChild(hdr)

		for name, ms := range ps.PerModel {
			if ms == nil {
				continue
			}
			row := div(doc, fmt.Sprintf("display:flex;gap:8px;padding:4px 8px;font-size:12px;color:%s;border-radius:2px;", ui.Text))
			row.SetAttribute("hover-style", "background:"+ui.HoverBg+";")
			row.AppendChild(txt(doc, name, "flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;"))
			row.AppendChild(txt(doc, comma(ms.LLMCalls), "width:45px;text-align:right;"))
			row.AppendChild(txt(doc, comma(ms.PromptTokens), "width:65px;text-align:right;"))
			row.AppendChild(txt(doc, comma(ms.CompletionTokens), "width:65px;text-align:right;"))
			row.AppendChild(txt(doc, comma(ms.TotalTokens), "width:65px;text-align:right;"))
			tbl.AppendChild(row)
		}
		container.AppendChild(tbl)
	}

	// ── 每日趋势 ──
	if len(ps.PerDay) > 0 {
		container.AppendChild(header(doc, "每日使用（最近 7 天）"))
		tbl := div(doc, "display:flex;flex-direction:column;gap:3px;")
		hdr := div(doc, fmt.Sprintf("display:flex;gap:8px;padding:4px 8px;background:%s;border-radius:2px;font-size:11px;color:%s;", ui.PanelHeader, ui.TextMute))
		hdr.AppendChild(txt(doc, "日期", "flex:1;"))
		hdr.AppendChild(txt(doc, "调用", "width:40px;text-align:right;"))
		hdr.AppendChild(txt(doc, "Token", "width:65px;text-align:right;"))
		hdr.AppendChild(txt(doc, "轮次", "width:40px;text-align:right;"))
		hdr.AppendChild(txt(doc, "工具", "width:40px;text-align:right;"))
		tbl.AppendChild(hdr)

		days := make([]string, 0, len(ps.PerDay))
		for d := range ps.PerDay {
			days = append(days, d)
		}
		// 冒泡排序倒序
		for i := 0; i < len(days)-1; i++ {
			for j := i + 1; j < len(days); j++ {
				if days[j] > days[i] {
					days[i], days[j] = days[j], days[i]
				}
			}
		}
		if len(days) > 7 {
			days = days[:7]
		}
		for _, date := range days {
			ds := ps.PerDay[date]
			if ds == nil {
				continue
			}
			row := div(doc, fmt.Sprintf("display:flex;gap:8px;padding:4px 8px;font-size:12px;color:%s;", ui.Text))
			row.AppendChild(txt(doc, date, "flex:1;"))
			row.AppendChild(txt(doc, comma(ds.LLMCalls), "width:40px;text-align:right;"))
			row.AppendChild(txt(doc, comma(ds.TotalTokens), "width:65px;text-align:right;"))
			row.AppendChild(txt(doc, comma(ds.Turns), "width:40px;text-align:right;"))
			row.AppendChild(txt(doc, comma(ds.ToolCalls), "width:40px;text-align:right;"))
			tbl.AppendChild(row)
		}
		container.AppendChild(tbl)
	}

	// ── 统计周期 ──
	footer := div(doc, "margin-top:16px;padding-top:8px;border-top:1px solid "+ui.Border+";font-size:11px;color:"+ui.TextMute+";display:flex;justify-content:space-between;")
	footer.AppendChild(txt(doc, "数据存储于 .pair/stats.json，删除对话不影响统计", ""))
	footer.AppendChild(txt(doc, fmt.Sprintf("持续 %s", fmtDuration(ps.FirstRecord, ps.LastUpdate)), ""))
	container.AppendChild(footer)

	// ── 重置按钮 ──
	resetBtn := doc.CreateElement("div")
	resetBtn.SetAttribute("style", fmt.Sprintf("margin-top:12px;padding:6px 12px;background:%s;color:%s;border-radius:4px;font-size:12px;cursor:pointer;text-align:center;", ui.Error, ui.Text))
	resetBtn.SetText("重置统计数据")
	resetBtn.SetAttribute("hover-style", "opacity:0.8;")
	container.AppendChild(resetBtn)

	clickCount := 0
	addClick(resetBtn, func() {
		clickCount++
		if clickCount == 1 {
			resetBtn.SetText("再次点击确认重置")
			resetBtn.SetAttribute("style", fmt.Sprintf("margin-top:12px;padding:6px 12px;background:%s;color:%s;border-radius:4px;font-size:12px;cursor:pointer;text-align:center;opacity:0.8;", ui.Warning, ui.Text))
		} else {
			core.ResetProjectStats()
			modal.Hide()
			uiapi.MessageSuccess("统计数据已重置")
		}
	})

	body.AppendChild(container)
	modal.Show()
}
