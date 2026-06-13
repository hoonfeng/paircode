// Agent 消息卡富渲染 —— 从 agent_bridge.go 提取。
//
//go:build windows

package chatpanel

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/ui/mdview"
	"github.com/hoonfeng/paircode/cmd/companion/ui/state"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// statusShadow 根据 Agent 消息状态返回带状态色的阴影（代替左边 3px 竖条）。
// 运行中=黄色晕、出错=红色晕、完成=默认暗晕。
func statusShadow(m state.Message) *paint.Shadow {
	col := types.ColorFromRGBA(0, 0, 0, 25) // 默认
	switch {
	case m.Streaming:
		col = types.ColorFromRGBA(234, 179, 8, 30) // 运行中→暖黄晕
	case strings.Contains(m.Text, "[错误]"):
		col = types.ColorFromRGBA(239, 68, 68, 35) // 出错→红晕
	}
	return &paint.Shadow{
		Offset: types.Point{X: 0, Y: 3},
		Blur:   10,
		Color:  col,
	}
}

// cardStyle 卡片基础样式（带状态阴影），复用避免重复
func cardStyle(bg *types.Color, radius float64, m state.Message) widget.Style {
	return widget.Style{
		BackgroundColor: bg,
		BorderRadius:    radius,
		Shadow:          statusShadow(m),
		FlexDirection:   "column",
		AlignItems:      "stretch",
	}
}

// ─── Agent 消息卡——按事件流顺序独立展示 ─────────────────────
//
// 思考/工具/正文按收到事件的顺序分别独立展示，不再合并为一段 Markdown。
// 思考链用 thinkingInline（可折叠文本），工具活动用 activityInline
//（带结果展开/折叠），正文用 mdview.Render（完整 Markdown 渲染）。
// 三者保持事件流的独立性——LLM 输出时 thinking/content/tool_call
// 是分别独立发送的事件，展示时也保持这种独立性。

func agentMessageCard(m state.Message, onToggleCollapse, onToggleThinking func(), onToggleActivity func(int)) widget.Widget {
	collapsed := !m.Streaming && m.Collapsed
	kids := []widget.Widget{agentHeaderCollapsible(m, onToggleCollapse)}
	if !collapsed {
		hasContent := false
		// 1) 思考链——独立展示（按事件流顺序，与工具/正文不合并）
		if t := strings.TrimSpace(m.Thinking); t != "" {
			kids = append(kids, vgap(6), thinkingInline(m, onToggleThinking))
			hasContent = true
		}
		// 2) 工具活动——各自独立展示（每个活动独立渲染，不合并到Markdown文本）
		for i, a := range m.Activities {
			kids = append(kids, vgap(4), activityInline(a, func() { onToggleActivity(i) }))
			hasContent = true
		}
		// 3) 正文——独立 Markdown 渲染（保持完整 Markdown 支持）
		if t := strings.TrimSpace(m.Text); t != "" {
			kids = append(kids, vgap(6), mdview.Render(t))
			hasContent = true
		}
		if !hasContent && m.Streaming {
			kids = append(kids, vgap(4), ui.TextC("思考中…", *ui.FgMuted, 12))
		}
		// 系统提示（保持简洁文本行，不属于 LLM 正文）
		for _, n := range m.Notes {
			kids = append(kids, vgap(4), systemNoteInline(n))
		}
		// 任务评分卡
		if m.Eval != nil {
			kids = append(kids, vgap(8), evalCard(m.Eval))
		}
	}
	style := cardStyle(ui.BgMuted, 6, m)
	style.Padding = types.EdgeInsetsLTRB(14, 10, 14, 10)
	return widget.Div(style, kids)
}

// evalCard 任务评分卡：星标 + 总分(配色) + 4 维度 + 一句话总评。
func evalCard(e *state.Eval) widget.Widget {
	scoreCol := *ui.Success // ≥80 绿 / ≥60 黄 / 否则红
	switch {
	case e.Total < 60:
		scoreCol = *ui.Danger
	case e.Total < 80:
		scoreCol = *ui.Warning
	}
	kids := []widget.Widget{
		widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"},
			widget.Lucide("star", widget.IconSize(13), widget.IconColor(*ui.Accent)),
			hgap(6),
			ui.TextC("任务评分", *ui.FgSubtle, 11),
			ui.Expand(widget.Div(widget.Style{})),
			ui.TextC(ui.Itoa(e.Total), scoreCol, 16),
			ui.TextC("/100", *ui.FgMuted, 11),
		),
		vgap(6),
		widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"},
			dimLabel("完成度", e.Completion, 40), dimLabel("正确性", e.Correctness, 30),
			dimLabel("深度", e.Depth, 20), dimLabel("效率", e.Efficiency, 10),
		),
	}
	if strings.TrimSpace(e.Feedback) != "" {
		kids = append(kids, vgap(5), ui.TextC(e.Feedback, *ui.FgSubtle, 11))
	}
	style := widget.Style{
		BackgroundColor: ui.Bg,
		BorderRadius:    5,
		FlexDirection:   "column",
		AlignItems:      "stretch",
		Padding:         types.EdgeInsetsLTRB(10, 8, 10, 8),
	}
	return widget.Div(style, kids)
}

// dimLabel 评分维度小标：「名 得/满」。
func dimLabel(name string, got, max int) widget.Widget {
	return widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(0, 0, 14, 0)},
		ui.TextC(name+" ", *ui.FgMuted, 10.5),
		ui.TextC(ui.Itoa(got)+"/"+ui.Itoa(max), *ui.Fg, 10.5),
	)
}

// systemNoteInline 系统提示——极简文本行，无图标，与正文、思考、工具活动
// 混在同一个连续流中。
func systemNoteInline(text string) widget.Widget {
	return ui.TextC(text, *ui.FgMuted, 10.5)
}

// agentStatusColor Agent 卡状态色（阴影用）：运行中黄 / 出错红 / 完成绿。
func agentStatusColor(m state.Message) *types.Color {
	switch {
	case m.Streaming:
		return ui.Warning
	case strings.Contains(m.Text, "[错误]"):
		return ui.Danger
	default:
		return ui.Success
	}
}

// agentHeaderCollapsible Agent 卡头：bot + Agent + 运行中点；完成后变可点折叠头（chevron + 折叠时摘要）。
func agentHeaderCollapsible(m state.Message, onToggle func()) widget.Widget {
	done := !m.Streaming
	var row []widget.Widget
	if done {
		ic := "chevron-down"
		if m.Collapsed {
			ic = "chevron-right"
		}
		row = append(row, widget.Lucide(ic, widget.IconSize(12), widget.IconColor(*ui.FgMuted)), hgap(4))
	}
	row = append(row,
		widget.Lucide("bot", widget.IconSize(13), widget.IconColor(*ui.Accent)),
		hgap(6),
		ui.TextC("Agent", *ui.FgSubtle, 11),
	)
	switch {
	case m.Streaming:
		row = append(row, hgap(8), statusDot(ui.Warning), hgap(4), ui.TextC("运行中", *ui.FgMuted, 10))
	case m.Collapsed:
		row = append(row, hgap(8), ui.Expand(ui.TextC(collapseSummary(m), *ui.FgMuted, 11)))
	}
	header := widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"}, row)
	if !done {
		return header
	}
	return &widget.Clickable{SingleChildWidget: widget.SingleChildWidget{Child: header}, OnClick: onToggle}
}

// collapseSummary 折叠态摘要：正文首行（去标题号）；无正文则按活动数。
func collapseSummary(m state.Message) string {
	if s := strings.TrimSpace(strings.TrimLeft(firstLine(m.Text), "# ")); s != "" {
		return truncRunes(s, 56)
	}
	if n := len(m.Activities); n > 0 {
		return "已完成 · " + strconv.Itoa(n) + " 步"
	}
	return "已完成"
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// thinkingInline 思考内容——以自然文本输出，不做独立"思考"标签和图标区分，
// 仅保留可折叠功能（流式时自动展开，完成后可折叠看首行），与正文、工具活动
// 混在同一个连续输出流中，像普通 LLM 回复一样自然。
func thinkingInline(m state.Message, onToggle func()) widget.Widget {
	expanded := m.Streaming || m.ThinkingExpanded
	col := *ui.FgSubtle // 思考内容用浅色，与正文略微区分但不做区块分隔
	kids := []widget.Widget{}
	if expanded {
		kids = append(kids, ui.TextC(strings.TrimSpace(m.Thinking), col, 11))
	} else {
		// 折叠时：仅显示首行摘要，可点击展开
		head := widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"},
			widget.Lucide("chevron-right", widget.IconSize(11), widget.IconColor(*ui.FgMuted)),
			hgap(4),
			ui.TextC(truncRunes(firstLine(m.Thinking), 48), *ui.FgMuted, 10.5),
		)
		kids = append(kids, &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: head},
			OnClick:           onToggle,
		})
	}
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"}, kids)
}

// activityInline 一次工具调用——以自然文本行输出，无图标/无边框/无背景色，
// 和思考内容、正文混在同一个流中，像普通 LLM 回复中自然出现的工具描述。
// 保留折叠（有结果时可展开查看详情）和待批准按钮。
func activityInline(a state.Activity, onToggle func()) widget.Widget {
	// 状态色标
	statusLabel := ""
	statusCol := *ui.FgMuted
	switch {
	case a.AwaitingApproval:
		statusLabel = " [待批准]"
		statusCol = *ui.Warning
	case a.Done:
		statusCol = *ui.Success
	default:
		statusLabel = " …"
	}
	// 工具调用行：`toolName(args…)` 样式，像普通文本描述
	argText := ArgPreview(a.Args)
	lineText := a.Tool + "(" + argText + ")" + statusLabel

	hasResult := a.Done && strings.TrimSpace(a.Result) != ""
	kids := []widget.Widget{}

	if hasResult && a.Expanded {
		// 展开态：可折叠头 + 结果全文
		head := widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"},
			widget.Lucide("chevron-down", widget.IconSize(11), widget.IconColor(*ui.FgMuted)),
			hgap(4),
			ui.TextC(lineText, statusCol, 10.5),
		)
		kids = append(kids, &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: head},
			OnClick:           onToggle,
		})
		kids = append(kids, vgap(3), activityResultBody(a.Result))
	} else if hasResult {
		// 折叠态：可点首行 + 结果首行预览
		head := widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"},
			widget.Lucide("chevron-right", widget.IconSize(11), widget.IconColor(*ui.FgMuted)),
			hgap(4),
			ui.TextC(lineText, statusCol, 10.5),
		)
		kids = append(kids, &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: head},
			OnClick:           onToggle,
		})
		kids = append(kids, vgap(2),
			ui.TextC(truncRunes(mdview.ExpandTabs(firstLine(a.Result)), 88), *ui.FgMuted, 10))
	} else {
		// 无结果（进行中或无结果）：纯文本行
		kids = append(kids, ui.TextC(lineText, statusCol, 10.5))
	}

	// 待批准时追加按钮条
	if a.AwaitingApproval {
		kids = append(kids, vgap(4), approvalBar(a.CallID))
	}

	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"}, kids)
}

// activityResultBody 展开态工具结果：等宽多行（tab→空格、截断 4000 防撑爆）。
func activityResultBody(result string) widget.Widget {
	t := widget.NewText(mdview.ExpandTabs(truncRunes(strings.TrimSpace(result), 4000)), *ui.FgSubtle)
	t.Font = mdview.MonoFont
	return t
}

// approvalBar 待批准操作的提示 + 允许/拒绝按钮（手动审核模式）。点击经单例 bridge 把裁决送回阻塞的 loop。
func approvalBar(callID string) widget.Widget {
	return widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"},
		widget.Lucide("shield-alert", widget.IconSize(12), widget.IconColor(*ui.Warning)),
		hgap(6),
		ui.TextC("等待批准", *ui.Warning, 11),
		ui.Expand(widget.Div(widget.Style{})),
		ui.SuccessBtnX("允许", func() { resolveApprovalUI(callID, true) }, ui.BtnOpts{Icon: "check", Size: ui.SizeSm}),
		hgap(6),
		ui.SolidDangerBtnX("拒绝", func() { resolveApprovalUI(callID, false) }, ui.BtnOpts{Icon: "x", Size: ui.SizeSm}),
	)
}

// resolveApprovalUI 把按钮点击路由到单例对话面板的 bridge（与 theChatState 单例约定一致）。
func resolveApprovalUI(callID string, ok bool) {
	if TheState != nil && TheState.Bridge != nil {
		TheState.Bridge.ResolveApproval(callID, ok)
	}
}

func iconForTool(name string) string {
	switch { // 前缀族：git_* / memory_* / mcp.*
	case strings.HasPrefix(name, "git_"):
		return "git-branch"
	case strings.HasPrefix(name, "memory_"):
		return "file-text"
	}
	switch name {
	case "read_file", "write_file":
		return "file-text"
	case "edit_file", "multi_edit":
		return "file-code"
	case "list_files":
		return "folder"
	case "search_content", "search_files":
		return "search"
	case "run_command", "run_background", "read_output":
		return "terminal"
	case "kill_process":
		return "circle-x"
	case "web_fetch", "web_search":
		return "globe"
	case "move_file":
		return "square-pen"
	case "delete_file":
		return "trash-2"
	}
	return "braces"
}

// argPreview 取关键参数（path/command/pattern）作预览，截断。
func argPreview(argsJSON string) string {
	var m map[string]any
	if json.Unmarshal([]byte(argsJSON), &m) == nil {
		for _, k := range []string{"path", "command", "pattern", "old_string"} {
			if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
				return truncRunes(strings.TrimSpace(v), 64)
			}
		}
	}
	return truncRunes(strings.TrimSpace(argsJSON), 64)
}

func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func vgap(h float64) widget.Widget { return widget.Div(widget.Style{Height: h}) }
func hgap(w float64) widget.Widget { return widget.Div(widget.Style{Width: w}) }
