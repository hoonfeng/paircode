// 对话面板 —— 1:1 复刻参考源（GitHub 深色主题）：布局工具栏 + 可折叠对话侧栏 +
// 用户/Agent 消息卡 + 输入区（包裹盒 + textarea + 工具按钮栏）。详见 AGENTS.md。
//
//go:build windows

package chatpanel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
	"github.com/hoonfeng/paircode/cmd/companion/agent"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/paircode/cmd/companion/ui/state"
)

// 颜色令牌统一在 ui 包（ui.Bg/*ui.Fg/ui.AccentStrong/ui.White…）；本文件改读 ui，不再本地声明 gh*。

// planStep Agent 任务计划的一步（update_plan 工具传入；status: pending/in_progress/done）。
type planStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

// pendingAsk agent 经 ask_user 工具发起的提问（带可选项）；用户回答前对话阻塞等待。
type pendingAsk struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Multi    bool     `json:"multiSelect"`
}

// ChatState 是对话面板的持久状态（包级单例）。原因：companion 每次 relayout 会重跑
// shell.Build 重建整棵右栏（含本面板的 StatefulElement），若状态挂在 Element 上会随之
// 被重置（showThreads/会话/草稿全丢）。单例确保跨重建存活。companion 只有一个对话面板，安全。
type ChatState struct {
	widget.BaseState
	Store        *state.ChatStore
	ShowThreads  bool        // 对话侧栏展开
	ThreadLeft   bool        // 对话侧栏靠左（默认右），由换边按钮切换
	AutoReview   bool        // 自动/手动审核
	Autonomous   bool        // 全自主模式
	AutoCollapse bool        // 最后一条自动收缩
	SendSeq      int         // 递增 → 输入框清空 + 滚到底
	InputAreaH   float64     // 输入区固定高（由 main.go 设为终端面板高 BottomH，使两者等高对齐）
	Bridge       AgentBridge // Agent 引擎接入（懒建，见 agent_bridge.go）
	Plan         []planStep  // 当前 Agent 任务计划清单（update_plan 工具更新；置顶可视）
	Ask          *pendingAsk // 当前 agent 的提问（ask_user）；非空时显问答卡、对话阻塞等回答
	Attachments  []string    // 待发送的附件文件路径（回形针添加，发送时随任务作上下文给 agent）
	PerfTest     int         // 性能测试令牌（递增→填充 1000 条测试消息，验证 VirtualList 虚加载性能）

	HoveredMsg  int    // 当前 hover 的消息索引（-1=无）→ 揭示该消息的操作按钮
	ShowSearch  bool   // Ctrl+F 搜索栏开
	SearchQuery string // 搜索词（实时过滤消息）
	searchSeq   int    // 搜索框受控清空令牌

	// VirtualList 高度缓存：避免每次 Build 重算 ItemHeights 覆盖 Layout 自动回写的实际高度
	cachedHeights []float64 // 缓存的消息高度数组（len=消息数，nil 时重建）
	cacheMsgLen   int       // 缓存时的消息总数（用于检测消息变动后重建）

	// 节流保存：记录上次 SaveHistory 的时间，避免每帧完整序列化对话历史
	lastSaveTime time.Time

	// 【轮询模式】数据版本号，用于 UI 线程检测数据变化。
	// Agent goroutine 通过 drain 写入 state 后递增此版本号（不调 SetState），
	// UI 线程每帧通过 app.OnDataChange 检测版本号变化后自动触发重建。
	DataVersion atomic.Int64

	// _taskPS 任务进度面板交互状态（懒初始化）。
	_taskPS *taskProgressState
}

// loadTestData 性能测试：填充 1000 条测试消息（含用户+Agent 对话对），验证 VirtualList 虚加载性能。
// Agent 消息包含思考链、工具活动、Markdown 正文，模拟真实对话负载。
func (s *ChatState) loadTestData() {
	t := s.Store.Active()
	if t == nil {
		return
	}
	const N = 500 // 500 轮用户+Agent = 1000 条消息
	for i := 0; i < N; i++ {
		t.Messages = append(t.Messages, state.Message{
			Role: state.User,
			Text: fmt.Sprintf("这是第 %d 轮用户消息，用于性能压测。请帮我分析这段数据并给出建议。", i+1),
		})
		t.Messages = append(t.Messages, state.Message{
			Role:     state.Assistant,
			Text:     fmt.Sprintf("## 第 %d 轮分析报告\n\n已完成分析。以下是结果：\n\n1. **数据概况**：共处理 %d 条记录\n2. **关键发现**：\n   - 性能正常\n   - 无异常错误\n3. **建议**：继续监控\n\n```go\nfmt.Println(\"Hello, 性能测试!\")\n```\n\n| 指标 | 值 |\n|------|----|\n| 延迟 | 5ms |\n| 吞吐 | 1000/s |", i+1, i*100),
			Thinking: "让我分析这些数据...\n1. 系统状态正常\n2. 性能指标在预期范围内\n3. 建议继续观察",
			Activities: []state.Activity{
				{Tool: "read_file", Args: `{"path":"data.csv"}`, Result: "已读取 1000 条记录", Done: true},
				{Tool: "analyze", Args: `{"metric":"latency"}`, Result: "平均延迟 5ms", Done: true},
			},
			Notes:     []string{"上下文已压缩（保留最近分析）"},
			Collapsed: i < N-3, // 仅最后 3 条展开，其余折叠减少首次渲染压力
		})
	}
	s.SendSeq++           // 滚到底部
	s.cachedHeights = nil // 清空高度缓存，让新消息重新估算
	s.SetState()
}

func (s *ChatState) Build(ctx widget.BuildContext) widget.Widget {
	mainKids := []widget.Widget{s.layoutToolbar()}
	if s.ShowSearch {
		mainKids = append(mainKids, s.searchBar())
	}
	mainKids = append(mainKids, s.taskProgressPanel())
	mainKids = append(mainKids, ui.Expand(s.scrollMessages()))
	if s.Ask != nil { // agent 提问中：问答卡置于输入区上方
		mainKids = append(mainKids, s.askCard())
	}
	mainKids = append(mainKids, s.inputArea())
	main := widget.Div(
		widget.Style{BackgroundColor: ui.Bg, FlexDirection: "column", AlignItems: "stretch"},
		mainKids,
	)
	if !s.ShowThreads {
		return main
	}
	// 对话列表：与主区并排分隔（不覆盖）。右栏整体会相应加宽（见 main.go rightColW），
	// 列表独立腾出；内容主区用 expandMin 保证 ≥MinChatW，不被列表挤窄（专注模式整壳填充时尤为关键）。
	if s.ThreadLeft {
		return ui.FlexRow(s.sidebarContent(), ui.VLine(), ui.ExpandMin(main, float64(state.MinChatW)))
	}
	return ui.FlexRow(ui.ExpandMin(main, float64(state.MinChatW)), ui.VLine(), s.sidebarContent())
}

func (s *ChatState) scrollMessages() widget.Widget {
	t := s.Store.Active()
	if t == nil {
		return widget.Div(widget.Style{})
	}
	q := strings.ToLower(strings.TrimSpace(s.SearchQuery))

	// 建立过滤后的消息索引（Ctrl+F 搜索过滤）
	filteredIndices := make([]int, 0, len(t.Messages))
	for i := range t.Messages {
		if q != "" && !msgMatches(t.Messages[i], q) {
			continue
		}
		filteredIndices = append(filteredIndices, i)
	}

	if q != "" && len(filteredIndices) == 0 {
		return widget.Div(
			widget.Style{Padding: types.EdgeInsets(10)},
			ui.TextC("无匹配消息", *ui.FgMuted, 12),
		)
	}

	// 使用缓存的 ItemHeights（避免每次 Build 重设覆盖 Layout 自动回写的实际高度）
	itemHeights := s.cachedHeights
	msgLen := len(t.Messages)
	if itemHeights == nil || len(itemHeights) != len(filteredIndices) || msgLen != s.cacheMsgLen || s.cacheMsgLen == 0 {
		// 缓存失效（消息增删/搜索过滤变化）：重新估算初始高度
		itemHeights = make([]float64, len(filteredIndices))
		for i, msgIdx := range filteredIndices {
			itemHeights[i] = estimateMessageHeight(t.Messages[msgIdx])
		}
		s.cachedHeights = itemHeights
		s.cacheMsgLen = msgLen
	}

	// 使用 VirtualList 虚加载：只渲染可视区内的消息，支撑大量消息流畅滚动
	return &widget.VirtualList{
		ItemCount:           len(filteredIndices),
		ItemHeight:          80,
		ItemHeights:         itemHeights,
		ScrollToBottomToken: s.SendSeq, // 新消息 → 自动滚到底
		Overscan:            5,
		RenderItem: func(i int) widget.Widget {
			return widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(10, 0, 10, 0), FlexDirection: "column"},
				s.renderMessage(t, filteredIndices[i]),
				widget.Div(widget.Style{Height: 8}), // 消息间间距
			)
		},
	}
}

// estimateMessageHeight 估算单条消息的初始渲染高度（VirtualList ItemHeights 初值，支持可变高度）。
// 折叠态≈56px；展开态按思考/活动/正文行数累加估算；Layout 后会通过 VirtualList 自动回写实际高度修正。
func estimateMessageHeight(m state.Message) float64 {
	if m.Collapsed && !m.Streaming {
		return 56 // 折叠态：header + padding + gap
	}
	h := 56.0 // 展开态基础：header + padding

	if len(m.Timeline) > 0 {
		// Timeline 渲染：按事件流顺序逐条估算
		for _, entry := range m.Timeline {
			switch entry.Kind {
			case "thinking":
				if m.ThinkingExpanded || m.Streaming {
					h += 200
				} else {
					h += 36
				}
			case "content":
				if t := strings.TrimSpace(entry.Content); t != "" {
					lines := strings.Count(t, "\n") + 1
					h += float64(lines)*19.0 + 4
				}
			case "tool":
				ah := 28.0
				if entry.Expanded {
					ah = 80
				}
				h += ah + 4
			}
		}
	} else {
		// 向后兼容：无 Timeline 时使用旧版估算
		if strings.TrimSpace(m.Thinking) != "" {
			if m.ThinkingExpanded || m.Streaming {
				h += 200
			} else {
				h += 36
			}
		}
		for _, a := range m.Activities {
			ah := 28.0
			if a.Expanded {
				ah = 80
			}
			h += ah
		}
		if txt := strings.TrimSpace(m.Text); txt != "" {
			lines := strings.Count(txt, "\n") + 1
			h += float64(lines) * 19.0
		}
	}
	if m.Eval != nil {
		h += 80
	}
	h += float64(len(m.Notes)) * 24
	return h
}

// planCard 保留旧接口别名（TaskProgressPanel 已替换它）。
func (s *ChatState) planCard() widget.Widget { return s.taskProgressPanel() }

// askCard agent 提问卡（ask_user）：问题 + 选项按钮 + 自由输入；回答前对话阻塞等待。
func (s *ChatState) askCard() widget.Widget {
	pa := s.Ask
	rows := []widget.Widget{
		widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(0, 0, 0, 5)},
			widget.Lucide("circle-help", widget.IconSize(14), widget.IconColor(*ui.Accent)),
			widget.Div(widget.Style{Width: 6}),
			ui.TextC("Agent 需要你的回答", *ui.Fg, 12),
		),
		ui.TextC(pa.Question, *ui.Fg, 13),
	}
	for _, o := range pa.Options {
		oo := o
		rows = append(rows, widget.Div(widget.Style{Height: 6}),
			ui.Btn(oo, func() { resolveAskUI(oo) }))
	}
	in := widget.NewInput("", func(string) {}).
		WithPlaceholder("或输入你的回答，回车提交…").
		WithOnSubmit(func(t string) {
			if strings.TrimSpace(t) != "" {
				resolveAskUI(t)
			}
		})
	ui.StyleInput(in)
	rows = append(rows, widget.Div(widget.Style{Height: 8}), in)
	askCardStyle := widget.Style{
		Padding:         types.EdgeInsets(10),
		BackgroundColor: ui.BgSubtle,
		BorderColor:     ui.Accent,
		BorderWidth:     1,
		BorderRadius:    6,
		Shadow:          &paint.Shadow{Offset: types.Point{X: 0, Y: 2}, Blur: 8, Color: types.ColorFromRGBA(0, 0, 0, 25)},
		FlexDirection:   "column",
		AlignItems:      "stretch",
	}
	return widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(8, 6, 8, 6)},
		widget.Div(askCardStyle, rows),
	)
}

// resolveAskUI 把问答卡的回答路由到 bridge。
func resolveAskUI(answer string) {
	if TheState != nil && TheState.Bridge != nil {
		TheState.Bridge.ResolveAsk(answer)
	}
}

// ─── 顶部布局工具栏（List 显隐侧栏，靠右）─────────────────
func (s *ChatState) layoutToolbar() widget.Widget {
	toggle := toolBtn("list", s.ShowThreads, func() { s.ShowThreads = !s.ShowThreads; s.SetState() })
	st := widget.Style{BackgroundColor: ui.Bg, BorderColor: ui.Border, BorderWidth: 1,
		FlexDirection: "row", AlignItems: "center"}
	// 区角「移动」手柄放在对话列表对侧，工具栏在那侧留 30px 给它，列表切换按钮顶到另一侧。
	if s.ThreadLeft { // 列表在左→手柄在右→右留 30、☰ 靠左
		st.Padding = types.EdgeInsetsLTRB(6, 3, 30, 3)
		return widget.Div(st, toggle, ui.Expand(widget.Div(widget.Style{})))
	}
	// 列表在右(默认)→手柄在左→左留 30、☰ 靠右
	st.Padding = types.EdgeInsetsLTRB(30, 3, 6, 3)
	return widget.Div(st, ui.Expand(widget.Div(widget.Style{})), toggle)
}

// ─── 对话侧栏 ─────────────────────────────────────────────

// sidebarContent 侧栏内容（180px）：头部（标题 + 切换/导出/新建）+ 会话列表（VirtualList 虚加载）。
func (s *ChatState) sidebarContent() widget.Widget {
	threads := s.Store.Threads
	itemH := 36.0
	return widget.Div(
		widget.Style{Width: 180, BackgroundColor: ui.BgSubtle, BorderColor: ui.Border, BorderWidth: 1,
			FlexDirection: "column", AlignItems: "stretch"},
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(10, 8, 8, 8), BorderColor: ui.Border, BorderWidth: 1,
				FlexDirection: "row", AlignItems: "center"},
			ui.Expand(ui.TextC("对话", *ui.FgSubtle, 12)),
			toolBtn("arrow-left-right", s.ThreadLeft, func() { s.ThreadLeft = !s.ThreadLeft; s.SetState() }),
			toolBtn("download", false, s.ExportActive),
			toolBtn("plus", false, func() { s.Store.NewThread(); s.saveHistory(); s.SetState() }),
		),
		ui.Expand(&widget.VirtualList{
			ItemCount:  len(threads),
			ItemHeight: itemH,
			Overscan:   5,
			RenderItem: func(i int) widget.Widget {
				return s.threadItem(threads[i])
			},
		}),
	)
}

// threadItem 会话项：左强调条(当前) + 状态灯(运行黄/就绪绿) + 标题 + 关闭×。整行可点、悬停高亮。
func (s *ChatState) threadItem(t *state.Thread) widget.Widget {
	tt := t
	active := t.ID == s.Store.ActiveID
	txt := *ui.FgMuted
	bg := types.Color{} // 透明
	hover := *ui.BgHover
	if active {
		txt = *ui.Fg
		bg = *ui.BgActive
		hover = *ui.BgActive // 选中态悬停不再变色
	}
	// 状态灯：该会话正被 Agent 运行→运行黄，否则就绪绿（复刻 thread.status 着色）。
	dot := ui.Success
	if s.threadRunning(tt) {
		dot = ui.Warning
	}
	// 左侧 3px 强调条（仅当前会话），复刻参考 borderLeft:3px accent；非当前留占位保持对齐。
	var bar widget.Widget
	if active {
		bar = widget.Div(widget.Style{Width: 3, Height: 18, BackgroundColor: ui.Accent, BorderRadius: 1.5})
	} else {
		bar = widget.Div(widget.Style{Width: 3})
	}
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 36, FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(0, 0, 6, 0)},
			bar,
			widget.Div(widget.Style{Width: 7}),
			statusDot(dot),
			widget.Div(widget.Style{Width: 8}),
			ui.Expand(ui.TextC(tt.Title, txt, 12)),
			s.threadCloseBtn(tt),
		)},
		OnClick:    func() { s.Store.Switch(tt.ID); s.SetState() },
		Color:      bg,
		HoverColor: hover,
	}
}

// threadCloseBtn 会话关闭×：StopPropagation 使点×只删会话、不触发外层切换。
func (s *ChatState) threadCloseBtn(t *state.Thread) widget.Widget {
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Padding: types.EdgeInsets(4)},
			widget.Lucide("x", widget.IconSize(13), widget.IconColor(*ui.FgMuted)),
		)},
		OnClick:         func() { s.Store.Delete(t.ID); s.saveHistory(); s.SetState() },
		StopPropagation: true,
		HoverColor:      *ui.BgHover,
	}
}

// threadRunning 该会话是否正被 Agent 引擎运行（状态灯着色用）。
func (s *ChatState) threadRunning(t *state.Thread) bool {
	return s.Bridge != nil && s.Bridge.RunningThread() == t
}

// ExportActive 把当前会话导出为 Markdown 写到工作区根，并在对话里回执路径（复刻 download/导出）。
func (s *ChatState) ExportActive() {
	th := s.Store.Active()
	if th == nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", th.Title)
	for _, m := range th.Messages {
		who := "用户"
		if m.Role == state.Assistant {
			who = "Agent"
		}
		fmt.Fprintf(&b, "## %s\n\n", who)
		if t := strings.TrimSpace(m.Thinking); t != "" {
			fmt.Fprintf(&b, "> 思考：%s\n\n", t)
		}
		for _, a := range m.Activities {
			fmt.Fprintf(&b, "- `%s` %s\n", a.Tool, ArgPreview(a.Args))
		}
		if len(m.Activities) > 0 {
			b.WriteString("\n")
		}
		if t := strings.TrimSpace(m.Text); t != "" {
			b.WriteString(t + "\n\n")
		}
	}
	wd, _ := os.Getwd()
	path := filepath.Join(wd, "对话导出-"+th.ID+".md")
	note := "已导出对话到 " + path
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		note = "导出失败：" + err.Error()
	}
	th.Messages = append(th.Messages, state.Message{Role: state.Assistant, Text: note})
	s.SendSeq++
	s.SetState()
}

func statusDot(c *types.Color) widget.Widget {
	return widget.Div(widget.Style{Width: 7, Height: 7, BackgroundColor: c, BorderRadius: 3.5})
}

// ─── 消息列表 ─────────────────────────────────────────────

// renderMessage 渲染单条消息：用户黄卡 / Agent 卡（折叠回调按索引改 store）；hover 揭示操作按钮（绝对定位叠加）。
func (s *ChatState) renderMessage(t *state.Thread, i int) widget.Widget {
	m := t.Messages[i]
	var card widget.Widget
	if m.Role == state.User {
		card = userCard(m)
	} else {
		card = agentMessageCard(m,
			func() { t.Messages[i].Collapsed = !t.Messages[i].Collapsed; s.SetState() },
			func() { t.Messages[i].ThinkingExpanded = !t.Messages[i].ThinkingExpanded; s.SetState() },
			func(ti int) {
				if ti >= 0 && ti < len(t.Messages[i].Timeline) && t.Messages[i].Timeline[ti].Kind == "tool" {
					t.Messages[i].Timeline[ti].Expanded = !t.Messages[i].Timeline[ti].Expanded
					// 同步 Activities 的展开状态（向后兼容）
					cid := t.Messages[i].Timeline[ti].CallID
					for ai := range t.Messages[i].Activities {
						if t.Messages[i].Activities[ai].CallID == cid {
							t.Messages[i].Activities[ai].Expanded = t.Messages[i].Timeline[ti].Expanded
							break
						}
					}
					s.SetState()
				} else if ti >= 0 && ti < len(t.Messages[i].Activities) {
					// 无 Timeline 时回退到 Activities 索引
					t.Messages[i].Activities[ti].Expanded = !t.Messages[i].Activities[ti].Expanded
					s.SetState()
				}
			},
		)
	}
	content := card
	if actions := s.messageActions(t, i); actions != nil { // hover 时叠加操作按钮（绝对定位，不挤布局）
		content = widget.NewStack(card, widget.NewPositioned(actions).WithTop(6).WithRight(8))
	}
	// hover 揭示：包一层只检测 hover 的 Clickable（OnClick=nil→无手型）；goui 子树 hover 语义保证移到按钮上仍算 hover。
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: content},
		OnHoverChange: func(h bool) {
			switch {
			case h:
				s.HoveredMsg = i
			case s.HoveredMsg == i:
				s.HoveredMsg = -1
			default:
				return
			}
			s.SetState()
		},
	}
}

// messageActions hover 揭示的操作按钮行：复制 / 重新生成（仅末条助手且空闲）/ 删除。未 hover 或流式中返回 nil。
func (s *ChatState) messageActions(t *state.Thread, i int) widget.Widget {
	if i < 0 || i >= len(t.Messages) || s.HoveredMsg != i {
		return nil
	}
	m := t.Messages[i]
	if m.Streaming {
		return nil
	}
	var kids []widget.Widget
	if widget.ClipboardWrite != nil {
		kids = append(kids, msgActionBtn("copy", func() { widget.ClipboardWrite(m.Text) }))
	}
	if m.Role == state.Assistant && i == len(t.Messages)-1 && !s.agentBusy() {
		kids = append(kids, msgActionBtn("refresh-cw", func() { s.Regenerate(t, i) }))
	}
	kids = append(kids, msgActionBtn("trash-2", func() { s.DeleteMessage(t, i) }))
	return widget.Div(
		widget.Style{BackgroundColor: ui.BgSubtle, BorderColor: ui.Border, BorderWidth: 1, BorderRadius: 5,
			Padding: types.EdgeInsets(2), FlexDirection: "row", AlignItems: "center", Gap: 1},
		kids,
	)
}

// msgActionBtn 消息操作小图标钮（透明底、悬停高亮）。
func msgActionBtn(icon string, onClick func()) widget.Widget {
	return &widget.Button{
		Icon: icon, IconSize: 12, TextColor: *ui.FgMuted,
		OnClick: onClick, Color: *ui.BgSubtle, HoverColor: *ui.BgHover,
		BorderRadius: 4, MinWidth: 22, MinHeight: 20,
	}
}

func (s *ChatState) agentBusy() bool { return s.Bridge != nil && s.Bridge.IsRunning() }

// persistAgentToggles 把输入栏的审核/自主/收缩开关写回设置并落盘（与设置面板「Agent」tab 同源，重启保持）。
func (s *ChatState) persistAgentToggles() {
	core.Settings.RequireApproval = !s.AutoReview // 自动审核 ↔ 破坏性操作需人工确认
	core.Settings.Autonomous = s.Autonomous
	core.Settings.AutoCollapse = s.AutoCollapse
	core.Save()
}

// deleteMessage 删除某条消息。
func (s *ChatState) DeleteMessage(t *state.Thread, i int) {
	if i < 0 || i >= len(t.Messages) {
		return
	}
	t.Messages = append(t.Messages[:i], t.Messages[i+1:]...)
	s.HoveredMsg = -1
	s.saveHistory()
	s.SetState()
}

// regenerate 重新生成末条助手回复：删掉它（末条复位为 user task）再跑一轮。
func (s *ChatState) Regenerate(t *state.Thread, i int) {
	if s.agentBusy() || i <= 0 || i != len(t.Messages)-1 {
		return
	}
	user := t.Messages[i-1]
	if user.Role != state.User {
		return
	}
	t.Messages = t.Messages[:i] // 删本条助手回复 → 末条变回 user task（start 的 history 排除末条）
	if s.Bridge == nil {
		if NewBridge != nil {
			s.Bridge = NewBridge(s)
		}
	}
	if s.Bridge == nil {
		return
	}
	s.HoveredMsg = -1
	s.SendSeq++
	s.saveHistory()
	s.Bridge.Start(user.Text)
	s.SetState()
}

// userCard 用户消息：黄底卡片、pre-wrap 文本（复刻 .cc-user）。
func userCard(m state.Message) widget.Widget {
	return widget.Div(
		widget.Style{
			BackgroundColor: ui.UserBg,
			BorderColor:     ui.UserBorder,
			BorderWidth:     1,
			BorderRadius:    6,
			Padding:         types.EdgeInsetsLTRB(12, 8, 12, 8),
			Shadow:          &paint.Shadow{Offset: types.Point{X: 0, Y: 2}, Blur: 8, Color: types.ColorFromRGBA(0, 0, 0, 25)},
			FlexDirection:   "column",
			AlignItems:      "stretch",
		},
		ui.TextC(m.Text, *ui.Fg, 13),
	)
}

// ─── 输入区（复刻 ChatInput）──────────────────────────────

func (s *ChatState) inputArea() widget.Widget {
	placeholder := "输入任务... Enter 发送"
	if s.Bridge != nil && s.Bridge.IsRunning() {
		placeholder = "Agent 运行中…（点停止后再发）"
	}
	ta := widget.NewTextarea(placeholder, 3, func(t string) { s.Store.Draft = t })
	ta.Wrap = true          // 长输入按宽折行，不横向滚动（用户：输入框要自动换行）
	ta.Text = s.Store.Draft // 重建时回填草稿（发送后随 SendSeq 复位为空）
	ta.ResetToken = s.SendSeq
	ta.BGColor = types.Color{} // 透明：让外层圆角包裹盒的背景+圆角透出（否则 textarea 方角盖住上圆角）
	ta.Color = *ui.Fg
	ta.CursorColor = *ui.Fg // 亮光标，深背景可见
	ta.PlaceholderColor = *ui.FgMuted
	// 三态边框都设为包裹盒底色 → textarea 自身无边框（边框由外层圆角盒提供）。
	// 关键：HoverBorderColor 不设会默认 el #c0c4cc 浅灰，导致悬停时框内冒出一圈灰边框。
	ta.BorderColor = *ui.Bg
	ta.FocusBorderColor = *ui.Bg
	ta.HoverBorderColor = *ui.Bg

	// 输入框包裹盒：圆角 8、1px 边框，含 [附件 chips] + textarea（撑满剩余高）+ 底部工具按钮栏。
	boxKids := []widget.Widget{}
	if len(s.Attachments) > 0 {
		boxKids = append(boxKids, s.attachmentChips())
	}
	boxKids = append(boxKids,
		ui.Expand(ta), // textarea 撑满包裹盒减按钮栏的剩余高度
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(8, 4, 8, 8), FlexDirection: "row", AlignItems: "center"},
			iconGhost("paperclip", s.addAttachment),
			ui.Expand(widget.Div(widget.Style{})),
			iconGhost("zap", func() { s.loadTestData() }), // ⚡性能测试：填充1000条消息验证虚加载
			widget.Div(widget.Style{Width: 5}),
			s.reviewToggle(),
			widget.Div(widget.Style{Width: 5}),
			toggleBtn("refresh-cw", "自主", s.Autonomous, ui.Blue, func() { s.Autonomous = !s.Autonomous; s.persistAgentToggles(); s.SetState() }),
			widget.Div(widget.Style{Width: 5}),
			toggleBtn("chevron-down", "收缩", s.AutoCollapse, ui.Accent, func() { s.AutoCollapse = !s.AutoCollapse; s.persistAgentToggles(); s.SetState() }),
			widget.Div(widget.Style{Width: 8}),
			s.sendOrStop(),
		),
	)
	box := widget.Div(
		widget.Style{BackgroundColor: ui.Bg, BorderColor: ui.Border, BorderWidth: 1, BorderRadius: 8,
			FlexDirection: "column", AlignItems: "stretch"},
		boxKids,
	)
	// 输入区固定高 = 终端面板高（inputAreaH），与中列底部终端面板等高对齐；包裹盒撑满。
	inputH := s.InputAreaH
	if inputH < 100 {
		inputH = 200 // 兜底（= 终端面板默认高）
	}
	return widget.Div(
		widget.Style{Height: inputH, Padding: types.EdgeInsetsLTRB(12, 8, 12, 10), BorderColor: ui.Border, BorderWidth: 1,
			BackgroundColor: ui.Bg, FlexDirection: "column", AlignItems: "stretch"},
		ui.Expand(box),
	)
}

// addAttachment 回形针：选文件加入待发送附件。
func (s *ChatState) addAttachment() {
	if widget.OpenFileDialog == nil {
		return
	}
	if p := widget.OpenFileDialog("添加附件", "所有文件|*.*"); p != "" {
		s.Attachments = append(s.Attachments, p)
		s.SetState()
	}
}

func (s *ChatState) removeAttachment(i int) {
	if i >= 0 && i < len(s.Attachments) {
		s.Attachments = append(s.Attachments[:i], s.Attachments[i+1:]...)
		s.SetState()
	}
}

// attachmentChips 待发送附件 chips（文件名 + 移除按钮）。
func (s *ChatState) attachmentChips() widget.Widget {
	var chips []widget.Widget
	for i, p := range s.Attachments {
		idx := i
		chips = append(chips,
			widget.Div(
				widget.Style{FlexDirection: "row", AlignItems: "center", BackgroundColor: ui.BgMuted,
					BorderRadius: 4, Padding: types.EdgeInsetsLTRB(6, 3, 4, 3)},
				widget.Lucide("file-text", widget.IconSize(11), widget.IconColor(*ui.FgMuted)),
				widget.Div(widget.Style{Width: 4}),
				ui.TextC(filepath.Base(p), *ui.Fg, 11),
				widget.Div(widget.Style{Width: 3}),
				&widget.Clickable{
					SingleChildWidget: widget.SingleChildWidget{Child: widget.Lucide("x", widget.IconSize(11), widget.IconColor(*ui.FgMuted))},
					OnClick:           func() { s.removeAttachment(idx) },
				},
			),
			widget.Div(widget.Style{Width: 6}),
		)
	}
	return widget.Div(
		widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(8, 8, 8, 0)},
		chips,
	)
}

// attachmentContext 把附件内容拼成给 agent 的上下文段（各截 20k）。
func attachmentContext(atts []string) string {
	if len(atts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n# 用户附件")
	for _, p := range atts {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := string(data)
		if len(content) > 20000 {
			content = content[:20000] + "\n…（已截断）"
		}
		fmt.Fprintf(&b, "\n\n## %s\n%s", p, content)
	}
	return b.String()
}

// attachmentNames 附件文件名（逗号分隔，用于显示）。
func attachmentNames(atts []string) string {
	names := make([]string, len(atts))
	for i, p := range atts {
		names[i] = filepath.Base(p)
	}
	return strings.Join(names, ", ")
}

// ─── Ctrl+F 搜索 ──────────────────────────────────────────

// ToggleSearch Ctrl+F 开关搜索栏（关闭时清空查询）。
func (s *ChatState) ToggleSearch() {
	s.ShowSearch = !s.ShowSearch
	if !s.ShowSearch {
		s.SearchQuery = ""
		s.searchSeq++
	}
	s.SetState()
}

// searchBar 搜索栏：放大镜 + 输入(实时过滤) + 匹配计数 + 关闭×。
func (s *ChatState) searchBar() widget.Widget {
	in := widget.NewInput("搜索对话内容…", func(q string) { s.SearchQuery = q; s.SetState() })
	in.Text = s.SearchQuery
	in.ResetToken = s.searchSeq
	in.Color = *ui.Fg
	in.CursorColor = *ui.Fg
	in.PlaceholderColor = *ui.FgMuted
	in.BGColor = *ui.BgSubtle
	in.BorderColor = *ui.BgSubtle
	in.FocusBorderColor = *ui.BgSubtle
	in.HoverBorderColor = *ui.BgSubtle
	var count widget.Widget = widget.Div(widget.Style{})
	if strings.TrimSpace(s.SearchQuery) != "" {
		count = ui.TextC(fmt.Sprintf("%d 条", s.SearchMatchCount()), *ui.FgMuted, 11)
	}
	return widget.Div(
		widget.Style{Height: 34, Padding: types.EdgeInsetsLTRB(10, 0, 8, 0), BackgroundColor: ui.BgSubtle,
			BorderColor: ui.Border, BorderWidth: 1, FlexDirection: "row", AlignItems: "center"},
		widget.Lucide("search", widget.IconSize(13), widget.IconColor(*ui.FgMuted)),
		widget.Div(widget.Style{Width: 8}),
		ui.Expand(in),
		count,
		widget.Div(widget.Style{Width: 6}),
		toolBtn("x", false, s.ToggleSearch),
	)
}

// msgMatches 消息是否命中搜索词（正文/思考/工具名·参数·结果；q 须已小写）。
func msgMatches(m state.Message, q string) bool {
	if strings.Contains(strings.ToLower(m.Text), q) || strings.Contains(strings.ToLower(m.Thinking), q) {
		return true
	}
	for _, a := range m.Activities {
		if strings.Contains(strings.ToLower(a.Tool), q) ||
			strings.Contains(strings.ToLower(a.Args), q) ||
			strings.Contains(strings.ToLower(a.Result), q) {
			return true
		}
	}
	return false
}

// searchMatchCount 当前会话命中搜索词的消息数。
func (s *ChatState) SearchMatchCount() int {
	t := s.Store.Active()
	q := strings.ToLower(strings.TrimSpace(s.SearchQuery))
	if t == nil || q == "" {
		return 0
	}
	n := 0
	for _, m := range t.Messages {
		if msgMatches(m, q) {
			n++
		}
	}
	return n
}

// ─── 小部件 ───────────────────────────────────────────────

// toolBtn 工具栏图标按钮（26×26，active 高亮）。用 Button 原生 Icon。
func toolBtn(icon string, active bool, onClick func()) widget.Widget {
	col := *ui.FgSubtle
	bg := *ui.Bg
	if active {
		col = *ui.Fg
		bg = *ui.BgMuted
	}
	return &widget.Button{
		Icon: icon, IconSize: 14, TextColor: col,
		OnClick: onClick, Color: bg,
		MinWidth: 26, MinHeight: 26,
	}
}

// iconGhost 透明底图标按钮（输入栏附件等）。用 Button 原生 Icon。
func iconGhost(icon string, onClick func()) widget.Widget {
	return &widget.Button{
		Icon: icon, IconSize: 14, TextColor: *ui.FgMuted,
		OnClick: onClick, Color: *ui.Bg,
		MinWidth: 24, MinHeight: 24,
	}
}

// tint 把强调色按 alpha 淡化为按钮底色（开启态的低透明填充）。
func tint(c *types.Color, a uint8) types.Color {
	return types.Color{R: c.R, G: c.G, B: c.B, A: a}
}

// reviewToggle 审核模式：双态彩色切换——自动(绿盾) ↔ 手动(黄盾)，永不灰显（复刻参考源）。
// 返回的就是一个 widget.Button（原生 Icon+Text），无任何中间封装。
func (s *ChatState) reviewToggle() widget.Widget {
	icon, text, c := "shield", "自动", ui.Success
	if !s.AutoReview {
		icon, text, c = "shield-off", "手动", ui.Warning
	}
	return &widget.Button{
		Icon: icon, IconSize: 13, IconGap: 4, Text: text, TextColor: *c, FontSize: 11,
		OnClick:      func() { s.AutoReview = !s.AutoReview; s.persistAgentToggles(); s.SetState() },
		Color:        tint(c, 30),
		BorderColor:  c,
		BorderWidth:  1,
		BorderRadius: 5,
		Padding:      types.EdgeInsetsLTRB(9, 3, 10, 3),
		MinHeight:    12,
	}
}

// toggleBtn 输入栏开关按钮（图标 + 文字）：开启时染强调色，关闭时低调灰。就是一个 widget.Button。
func toggleBtn(icon, text string, on bool, onColor *types.Color, onClick func()) widget.Widget {
	fg, border, bg := *ui.FgMuted, ui.Border, *ui.Bg
	if on {
		fg, border, bg = *onColor, onColor, tint(onColor, 30)
	}
	return &widget.Button{
		Icon: icon, IconSize: 13, IconGap: 4, Text: text, TextColor: fg, FontSize: 11,
		OnClick:      onClick,
		Color:        bg,
		BorderColor:  border,
		BorderWidth:  1,
		BorderRadius: 5,
		Padding:      types.EdgeInsetsLTRB(9, 3, 10, 3),
		MinHeight:    12,
	}
}

// primaryBtn 主按钮（发送）：实心强调底 + 送出图标 + 文字。用 Button 原生 Icon。
func primaryBtn(text string, onClick func()) widget.Widget {
	return &widget.Button{
		Icon: "send", IconSize: 13, IconGap: 6,
		Text: text, TextColor: *ui.White, FontSize: 12,
		OnClick:      onClick,
		Color:        *ui.AccentStrong,
		BorderRadius: 5,
		Padding:      types.EdgeInsetsLTRB(12, 3, 13, 3),
		MinHeight:    12,
	}
}

// sendOrStop 运行中显示红「停止」(取消 loop)，空闲显示蓝「发送」。
func (s *ChatState) sendOrStop() widget.Widget {
	if s.Bridge != nil && s.Bridge.IsRunning() {
		return &widget.Button{
			Icon: "square", IconSize: 12, IconGap: 6,
			Text: "停止", TextColor: *ui.White, FontSize: 12,
			OnClick:      func() { s.Bridge.Stop() },
			Color:        *ui.Danger,
			BorderRadius: 5,
			Padding:      types.EdgeInsetsLTRB(12, 3, 13, 3),
			MinHeight:    12,
		}
	}
	return primaryBtn("发送", s.Send)
}

// sendTask 以指定任务发起一轮（菜单/按钮触发，如「探索项目知识库」），不经输入框草稿。
func (s *ChatState) sendTask(task string) {
	if s.Bridge != nil && s.Bridge.IsRunning() {
		return
	}
	if !s.Store.Send(task) {
		return
	}
	s.SendSeq++
	s.Plan = nil
	s.Attachments = nil
	if s.Bridge == nil {
		if NewBridge != nil {
			s.Bridge = NewBridge(s)
		}
	}
	if s.Bridge == nil {
		return
	}
	s.Bridge.Start(task)
	s.saveHistory()
	s.SetState()
}

func (s *ChatState) Send() {
	if s.Bridge != nil && s.Bridge.IsRunning() {
		return // 上一轮还在跑，不重复发
	}
	draft := s.Store.Draft
	atts := s.Attachments
	display := draft // 显示给用户的消息：正文 + 附件名（不含内容）
	if len(atts) > 0 {
		if display != "" {
			display += "\n"
		}
		display += "[附件：" + attachmentNames(atts) + "]"
	}
	if !s.Store.Send(display) { // 只加 user 消息（Send 内部 trim+空判）
		return
	}
	s.SendSeq++  // 清输入框 + 滚到底
	s.Plan = nil // 新任务 → 清旧计划清单（Agent 会用 update_plan 重列）
	s.Attachments = nil
	if s.Bridge == nil {
		if NewBridge != nil {
			s.Bridge = NewBridge(s)
		}
	}
	if s.Bridge == nil {
		return
	}
	s.Bridge.Start(draft + attachmentContext(atts)) // agent 任务含附件内容（内容只给 LLM、不污染显示）
	s.saveHistory()
	s.SetState()
}

// ExploreKnowledgeBase 探索项目知识库（菜单动作）。
func (s *ChatState) ExploreKnowledgeBase() { s.sendTask(agent.ExploreKnowledgeTask()) }

// ApplyPlan 从 update_plan 工具参数解析并设置计划清单。
func (s *ChatState) ApplyPlan(argsJSON string) {
	var p struct {
		Plan []planStep `json:"plan"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &p); err == nil {
		s.Plan = p.Plan
	}
}

// ApplyAsk 从 ask_user 工具参数解析并设置提问。
func (s *ChatState) ApplyAsk(argsJSON string) {
	var pa pendingAsk
	if err := json.Unmarshal([]byte(argsJSON), &pa); err == nil && pa.Question != "" {
		s.Ask = &pa
	}
}

// ClearAsk 清空当前提问。
func (s *ChatState) ClearAsk() { s.Ask = nil }
