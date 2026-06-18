// TaskProgressPanel —— 富交互任务进度面板（参考源 TaskProgressPanel）。
// 替换旧 planCard 静态列表，支持：编辑/删除/勾选完成/后端轮询/自动完成/进度条。
//
//go:build windows

package chatpanel

import (
	"fmt"
	"strings"

	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
)

// ─── 增强的任务数据模型 ────────────────────────────────────

// taskItem 一条任务的运行时表示（由 planStep + 交互状态组成）。
type taskItem struct {
	Step   string `json:"step"`
	Status string `json:"status"` // pending / in_progress / done / cancelled
	// 本地覆盖：用户交互（编辑/勾选）用此覆盖原始状态，同步到 bridge
	localStatus string
	localTitle  string
	deleted     bool
}

// progressStats 任务进度统计。
type progressStats struct {
	total     int
	completed int
	inFlight  int // 运行中（in_progress）
}

// taskProgressState 任务进度面板的交互状态（挂 ChatState 上）。
type taskProgressState struct {
	expanded bool       // 折叠/展开
	tasks    []taskItem // 当前任务列表（含本地覆盖）
	editIdx  int        // 正在编辑的任务索引（-1=无）
	editText string     // 编辑框文本
	_prevAllDone bool      // 上次是否全部完成（检测过渡态用）
}

// ─── ChatState 接口（挂载点）───────────────────────────────

// taskPS 返回任务进度交互状态（懒初始化）。
func (s *ChatState) taskPS() *taskProgressState {
	if s._taskPS == nil {
		s._taskPS = &taskProgressState{expanded: false, editIdx: -1}
	}
	return s._taskPS
}

// _taskPS 在 ChatState 里声名（见 chat.go 添加的字段）。
// 声明在 chat.go 里，这里只是使用。

// ─── 主构建函数 ────────────────────────────────────────────

// taskProgressPanel 构建任务进度面板（替代旧 planCard）。
// 顶部：任务数统计 + 进度条 + 折叠展开。
// 展开时：逐项显示任务，每项含状态圆圈 + 标题 + 操作按钮（编辑/删除）。
func (s *ChatState) taskProgressPanel() widget.Widget {
	tps := s.taskPS()
	if len(s.Plan) == 0 {
		return widget.Div(widget.Style{})
	}

	// 用当前 Plan 同步到 taskPS.tasks（保留本地覆盖）
	s.syncTasksFromPlan()

	stats := calcStats(tps.tasks)
// 过渡态自动折叠：只在刚完成时触发一次，不覆盖用户手动切换
allDone := !s.agentBusy() && stats.completed == stats.total && stats.total > 0
if s.AutoCollapse && allDone && !tps._prevAllDone && tps.expanded {
	tps.expanded = false
}
tps._prevAllDone = allDone

	kids := []widget.Widget{
		s.taskHeader(tps, stats),
	}
	if tps.expanded {
		for i := range tps.tasks {
			if tps.tasks[i].deleted {
				continue
			}
			kids = append(kids, s.taskRow(tps, i))
		}
	}

	return widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(8, 6, 8, 6)},
		widget.Div(
			widget.Style{
				Padding:         types.EdgeInsets(8),
				BackgroundColor: ui.BgSubtle,
				BorderRadius:    6,
				Shadow:          &paint.Shadow{Offset: types.Point{X: 0, Y: 2}, Blur: 8, Color: types.ColorFromRGBA(0, 0, 0, 25)},
				FlexDirection:   "column",
				AlignItems:      "stretch",
			},
			kids,
		),
	)
}

// syncTasksFromPlan 把 s.Plan 同步到 taskPS.tasks，保留本地覆盖。
func (s *ChatState) syncTasksFromPlan() {
	tps := s.taskPS()
	// 建立旧任务索引
	old := make(map[string]int) // step → index
	for i, t := range tps.tasks {
		if !t.deleted {
			old[t.Step] = i
		}
	}
	// 重建列表
	newTasks := make([]taskItem, 0, len(s.Plan))
	for _, p := range s.Plan {
		ti := taskItem{Step: p.Step, Status: p.Status}
		if idx, ok := old[ti.Step]; ok {
			prev := tps.tasks[idx]
			// 保留本地覆盖
			if prev.localStatus != "" {
				ti.localStatus = prev.localStatus
			}
			if prev.localTitle != "" {
				ti.localTitle = prev.localTitle
			}
		}
		newTasks = append(newTasks, ti)
	}
	tps.tasks = newTasks
}

// calcStats 计算任务统计。
func calcStats(tasks []taskItem) progressStats {
	var s progressStats
	for _, t := range tasks {
		if t.deleted {
			continue
		}
		status := t.localStatus
		if status == "" {
			status = t.Status
		}
		s.total++
		switch status {
		case "done":
			s.completed++
		case "in_progress":
			s.inFlight++
		}
	}
	return s
}

// effectiveStatus 返回任务的实际显示状态（优先本地覆盖）。
func (t *taskItem) effectiveStatus() string {
	if t.localStatus != "" {
		return t.localStatus
	}
	return t.Status
}

func (t *taskItem) effectiveTitle() string {
	if t.localTitle != "" {
		return t.localTitle
	}
	return t.Step
}

// ─── 头部：统计 + 进度条 + 折叠按钮 ──────────────────────

func (s *ChatState) taskHeader(tps *taskProgressState, stats progressStats) widget.Widget {
	// 更新：Agent 已完成 → 所有 in_progress 自动变 done
	if s.agentBusy() {
		for i := range tps.tasks {
			if tps.tasks[i].effectiveStatus() == "in_progress" && tps.tasks[i].localStatus == "" {
				tps.tasks[i].localStatus = "in_progress" // 冻结计划状态
			}
		}
	} else {
		// Agent 空闲 → 还在 in_progress 的标记为 done（对话结束时自动完成）
		changed := false
		for i := range tps.tasks {
			if tps.tasks[i].effectiveStatus() == "in_progress" && tps.tasks[i].localStatus == "" {
				tps.tasks[i].localStatus = "done"
				changed = true
			}
		}
		if changed {
			// 同步回 s.Plan
			for i := range s.Plan {
				if s.Plan[i].Status == "in_progress" {
					s.Plan[i].Status = "done"
				}
			}
		}
		// 重新统计
		stats = calcStats(tps.tasks)
	}

	pct := 0.0
	if stats.total > 0 {
		pct = float64(stats.completed) / float64(stats.total) * 100
	}

	// 图标 + 标题 + 计数
	icon := "list-checks"
	txt := "计划"
	if stats.inFlight > 0 {
		icon = "loader-circle"
	}

	// 进度条
	bar := widget.Div(
		widget.Style{
			Height:          3,
			BackgroundColor: ui.BgMuted,
			BorderRadius:    1.5,
			FlexDirection:   "row",
			AlignItems:      "stretch",
			Margin:          types.EdgeInsetsLTRB(0, 4, 0, 0),
		},
		widget.Div(widget.Style{
			Width:           pct,
			BackgroundColor: ui.Accent,
			BorderRadius:    1.5,
		}),
	)

	// 折叠展开按钮
	var toggleBtn widget.Widget
	toggleIcon := "chevron-down"
	if !tps.expanded {
		toggleIcon = "chevron-right"
	}
	toggleBtn = &widget.Button{
		Icon: toggleIcon, IconSize: 13, TextColor: *ui.FgMuted,
		OnClick: func() { tps.expanded = !tps.expanded; s.SetState() },
		Color:   *ui.BgSubtle,
		MinWidth: 20, MinHeight: 20,
	}

	return widget.Div(
		widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(0, 0, 0, 5)},
		widget.Lucide(icon, widget.IconSize(13), widget.IconColor(*ui.Accent)),
		widget.Div(widget.Style{Width: 4}),
		ui.TextC(txt, *ui.Fg, 12),
		widget.Div(widget.Style{Width: 4}),
		ui.TextC(fmt.Sprintf("%d/%d  %.0f%%", stats.completed, stats.total, pct), *ui.FgMuted, 10),
		ui.Expand(bar),
		toggleBtn,
	)
}

// ─── 单行任务 ──────────────────────────────────────────────

func (s *ChatState) taskRow(tps *taskProgressState, i int) widget.Widget {
	t := tps.tasks[i]
	status := t.effectiveStatus()
	title := t.effectiveTitle()

	// 状态圆圈图标
	icon, col := "circle", *ui.FgMuted
	switch status {
	case "done":
		icon, col = "circle-check", *ui.Success
	case "in_progress":
		icon, col = "loader-circle", *ui.Warning
	}
	txtCol := *ui.Fg
	if status == "done" {
		txtCol = *ui.FgMuted
	}

	// 状态切换按钮（点击圆圈切换完成/待执行）
	checkbox := &widget.Button{
		Icon: icon, IconSize: 13, TextColor: col,
		OnClick: func() {
			if t.localStatus == "done" {
				t.localStatus = "pending"
			} else {
				t.localStatus = "done"
			}
			s.SetState()
		},
		Color: *ui.BgSubtle, MinWidth: 20, MinHeight: 20,
	}

	// 标题
	var titleWidget widget.Widget
	if tps.editIdx == i {
		// 编辑模式：输入框 + 保存/取消按钮
		in := widget.NewInput("", func(string) {})
		in.Text = tps.editText
		in.Color = *ui.Fg
		in.CursorColor = *ui.Fg
		in.PlaceholderColor = *ui.FgMuted
		in.BGColor = *ui.Bg
		in.BorderColor = *ui.Border
		in.Text = tps.editText
		in.OnSubmit = func(txt string) {
			if strings.TrimSpace(txt) != "" {
				t.localTitle = strings.TrimSpace(txt)
			}
			tps.editIdx = -1
			s.SetState()
		}
		// 按 Esc 取消
		titleWidget = widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center", Gap: 4},
			ui.Expand(in),
			&widget.Button{
				Icon: "check", IconSize: 11, TextColor: *ui.Success,
				OnClick: func() {
					if strings.TrimSpace(tps.editText) != "" {
						t.localTitle = strings.TrimSpace(tps.editText)
					}
					tps.editIdx = -1
					s.SetState()
				},
				Color: *ui.BgSubtle, MinWidth: 18, MinHeight: 18,
			},
			&widget.Button{
				Icon: "x", IconSize: 11, TextColor: *ui.FgMuted,
				OnClick: func() { tps.editIdx = -1; s.SetState() },
				Color:   *ui.BgSubtle, MinWidth: 18, MinHeight: 18,
			},
		)
	} else {
		titleWidget = &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: ui.Expand(ui.TextC(title, txtCol, 11.5))},
			OnClick: func() {
				tps.editIdx = i
				tps.editText = title
				s.SetState()
			},
		}
	}

	// 操作按钮（仅 hover 显示，在 agent 消息卡里 hover 已有机制；这里简化成一直显示）
	actions := widget.Div(
		widget.Style{FlexDirection: "row", AlignItems: "center", Gap: 0},
		&widget.Button{
			Icon: "pencil", IconSize: 11, TextColor: *ui.FgMuted,
			OnClick: func() { tps.editIdx = i; tps.editText = title; s.SetState() },
			Color:   *ui.BgSubtle, MinWidth: 18, MinHeight: 18,
		},
		&widget.Button{
			Icon: "trash-2", IconSize: 11, TextColor: *ui.Danger,
			OnClick: func() { t.deleted = true; s.SetState() },
			Color:   *ui.BgSubtle, MinWidth: 18, MinHeight: 18,
		},
	)

	return widget.Div(
		widget.Style{Height: 22, FlexDirection: "row", AlignItems: "center", Gap: 4},
		checkbox,
		widget.Div(widget.Style{Width: 2}),
		ui.Expand(titleWidget),
		actions,
	)
}
