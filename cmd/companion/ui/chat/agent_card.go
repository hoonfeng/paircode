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

// ─── Agent 消息卡富渲染（头 + 思考 + 工具活动 + 正文，按时间顺序混排）──────────
//
// 不再分为三个固定区域（思考块 / 工具活动 / 正文），而是按 Agent 执行的时间顺序
// 流畅展示：header → 思考 → 工具活动 → 正文。思考块不再带独立竖线和灰底背景，
// 工具活动行不再带独立阴影，所有内容在统一卡片内用图标标签区分类型。

func agentMessageCard(m state.Message, onToggleCollapse, onToggleThinking func(), onToggleActivity func(int)) widget.Widget {
	collapsed := !m.Streaming && m.Collapsed
	kids := []widget.Widget{agentHeaderCollapsible(m, onToggleCollapse)}
	if !collapsed {
		// 1) 思考链（按时间顺序最先）
		if strings.TrimSpace(m.Thinking) != "" {
			kids = append(kids, vgap(6), thinkingBlock(m, onToggleThinking))
		}
		// 2) 系统提示（如上下文压缩等）
		for _, n := range m.Notes {
			kids = append(kids, vgap(5), systemNote(n))
		}
		// 3) 工具活动（按调用顺序）
		for i, a := range m.Activities {
			ai := i
			kids = append(kids, vgap(6), activityRow(a, func() { onToggleActivity(ai) }))
		}
		// 4) 正文（最终输出，时间上最后）
		if txt := strings.TrimSpace(m.Text); txt != "" {
			kids = append(kids, vgap(8), mdview.Render(m.Text))
		} else if m.Streaming && len(m.Activities) == 0 && strings.TrimSpace(m.Thinking) == "" {
			kids = append(kids, vgap(6), ui.TextC("思考中…", *ui.FgMuted, 12))
		}
		// 5) 任务评分（完成后）
		if m.Eval != nil {
			kids = append(kids, vgap(8), evalCard(m.Eval))
		}
	}
	style := cardStyle(ui.BgMuted, 6, m)
	style.Padding = types.EdgeInsetsLTRB(14, 10, 14, 10)
	// 状态通过阴影颜色表达（statusShadow），不再使用左边 3px 竖条
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

// systemNote 系统提示（上下文压缩 / 自动迭代 / 探索 / 验证等）：素色小字一行 + 贴切图标（非 LLM 正文）。
func systemNote(text string) widget.Widget {
	icon := "minimize" // 压缩=还原图标（默认）
	switch {
	case strings.Contains(text, "重复") || strings.Contains(text, "绕圈"):
		icon = "shield-alert" // 绕圈检测=警示
	case strings.Contains(text, "迭代"):
		icon = "refresh-cw" // 自动迭代=循环
	case strings.Contains(text, "探索"):
		icon = "search" // 探索阶段=放大镜
	case strings.Contains(text, "验证"):
		icon = "circle-check" // 验证阶段=对勾
	}
	return widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"},
		widget.Lucide(icon, widget.IconSize(11), widget.IconColor(*ui.FgMuted)),
		hgap(5),
		ui.Expand(ui.TextC(text, *ui.FgMuted, 10.5)),
	)
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

// thinkingBlock 思考块：可折叠头（思考图标 + "思考"标签）；展开看全文、折叠看首行。
// 不再使用左竖线和灰底背景，改为行内标签标记，与工具活动、正文统一风格按时间顺序展示。
func thinkingBlock(m state.Message, onToggle func()) widget.Widget {
	expanded := m.Streaming || m.ThinkingExpanded
	ic := "chevron-right"
	if expanded {
		ic = "chevron-down"
	}
	header := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			widget.Lucide(ic, widget.IconSize(11), widget.IconColor(*ui.FgMuted)),
			hgap(4),
			widget.Lucide("brain", widget.IconSize(12), widget.IconColor(*ui.FgMuted)),
			hgap(4),
			ui.TextC("思考", *ui.FgMuted, 10),
		)},
		OnClick: onToggle,
	}
	kids := []widget.Widget{header}
	if expanded {
		kids = append(kids, vgap(4), ui.TextC(strings.TrimSpace(m.Thinking), *ui.FgSubtle, 11))
	} else {
		kids = append(kids, vgap(3), ui.TextC(truncRunes(firstLine(m.Thinking), 48), *ui.FgMuted, 10.5))
	}
	return widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(8, 4, 0, 4),
			FlexDirection: "column", AlignItems: "stretch"},
		kids,
	)
}

// activityRow 一次工具调用：[chevron] 工具图标(进行蓝/完成绿/待批准黄) + 名 + 参数预览；
// 待批准时附「允许/拒绝」按钮条；有结果时头可点折叠——折叠看首行预览、展开看全量(等宽)。
// 不再使用独立阴影，统一由外层 Agent 卡片的阴影表达状态。
func activityRow(a state.Activity, onToggle func()) widget.Widget {
	iconCol := *ui.Accent
	switch {
	case a.AwaitingApproval:
		iconCol = *ui.Warning
	case a.Done:
		iconCol = *ui.Success
	}
	hasResult := a.Done && strings.TrimSpace(a.Result) != ""
	var headRow []widget.Widget
	if hasResult {
		ic := "chevron-right"
		if a.Expanded {
			ic = "chevron-down"
		}
		headRow = append(headRow, widget.Lucide(ic, widget.IconSize(11), widget.IconColor(*ui.FgMuted)), hgap(4))
	} else {
		headRow = append(headRow, widget.Div(widget.Style{Width: 15})) // 无 chevron 时占位对齐
	}
	headRow = append(headRow,
		widget.Lucide(iconForTool(a.Tool), widget.IconSize(12), widget.IconColor(iconCol)),
		hgap(6),
		ui.TextC(a.Tool, *ui.Fg, 11),
		hgap(6),
		ui.Expand(ui.TextC(ArgPreview(a.Args), *ui.FgMuted, 11)),
	)
	headDiv := widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"}, headRow)
	var head widget.Widget = headDiv
	if hasResult { // 头可点折叠/展开
		head = &widget.Clickable{SingleChildWidget: widget.SingleChildWidget{Child: headDiv}, OnClick: onToggle}
	}

	kids := []widget.Widget{head}
	switch {
	case a.AwaitingApproval:
		kids = append(kids, vgap(6), approvalBar(a.CallID))
	case hasResult && a.Expanded:
		kids = append(kids, vgap(4), activityResultBody(a.Result))
	case hasResult:
		kids = append(kids, vgap(3), ui.TextC(truncRunes(mdview.ExpandTabs(firstLine(a.Result)), 88), *ui.FgSubtle, 10))
	}
	border := ui.Border
	if a.AwaitingApproval {
		border = ui.Warning
	}
	// 工具活动行使用边框 + 浅底（无独立阴影，状态由外层卡片阴影传达）
	style := widget.Style{
		BackgroundColor: ui.Bg,
		BorderColor:     border,
		BorderWidth:     1,
		BorderRadius:    4,
		Padding:         types.EdgeInsetsLTRB(8, 6, 8, 6),
		FlexDirection:   "column",
		AlignItems:      "stretch",
	}
	return widget.Div(style, kids)
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
