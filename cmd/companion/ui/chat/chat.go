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
	"math"
	"sync/atomic"
	"time"

	"github.com/hoonfeng/goui/pkg/canvas"
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
	DraftVersion int         // 递增 → 外部设置草稿时刷新输入框（右键添加到对话等）
	InputAreaH   float64     // 输入区固定高（由 main.go 设为终端面板高 BottomH，使两者等高对齐）
	Bridge       AgentBridge // Agent 引擎接入（懒建，见 agent_bridge.go）
	Plan         []planStep  // 当前 Agent 任务计划清单（update_plan 工具更新；置顶可视）
	Ask          *pendingAsk // 当前 agent 的提问（ask_user）；非空时显问答卡、对话阻塞等回答
	Attachments  []string    // 待发送的附件文件路径（回形针添加，发送时随任务作上下文给 agent）
	DraftRefs    []string    // 右键菜单「添加到对话」的引用文本（以 chips 展示在输入框上方，发送时拼入上下文）
	PerfTest     int         // 性能测试令牌（递增→填充 1000 条测试消息，验证 VirtualList 虚加载性能）

	HoveredMsg  int    // 当前 hover 的消息索引（-1=无）→ 揭示该消息的操作按钮
	ShowSearch  bool   // Ctrl+F 搜索栏开
	SearchQuery string // 搜索词（实时过滤消息）
	searchSeq   int    // 搜索框受控清空令牌

	// VirtualList 高度缓存：避免每次 Build 重算 ItemHeights 覆盖 Layout 自动回写的实际高度
	cachedHeights      []float64 // 缓存的消息高度数组（len=消息数，nil 时重建）
	cacheMsgLen        int       // 缓存时的消息总数（用于检测消息变动后重建）
	cachedScrollOffset float64   // 缓存滚动偏移（OnScroll 保存→下次 Build 回传 VirtualList.ScrollOffset，防止鼠标移动/点击后跳回顶部）

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
	mainKids = append(mainKids, ui.Expand(s.scrollMessages()))
	if s.Ask != nil { // agent 提问中：问答卡置于输入区上方
		mainKids = append(mainKids, s.askCard())
	}
	mainKids = append(mainKids, s.taskProgressPanel())
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
	// 使用 VirtualList 虚加载：只渲染可视区内的消息，支撑大量消息流畅滚动
	return &widget.VirtualList{
		ItemCount:           len(filteredIndices),
		ItemHeight:          80,
		ItemHeights:         itemHeights,
		ScrollToBottomToken: s.SendSeq, // 新消息 → 自动滚到底
		ScrollOffset:        s.cachedScrollOffset,
		OnScroll: func(so float64) {
			s.cachedScrollOffset = so
		},
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

// sidebarContent 侧栏内容（270px）：上部分对话列表 50% + 下部分均分给环图区和 Token 统计（各 25%）
func (s *ChatState) sidebarContent() widget.Widget {
	threads := s.Store.Threads
	itemH := 36.0
	// 计算整个项目的 Token 用量和分类占比（遍历所有会话）
	ctxUsed, ctxMax, cats, tokenStats := s.projectTokenStats()
	return widget.Div(
		widget.Style{Width: 270, BackgroundColor: ui.BgSubtle, BorderColor: ui.Border, BorderWidth: 1,
			FlexDirection: "column", AlignItems: "stretch"},
		// 头部（对话列表在上方，方便拖拽）
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(10, 8, 8, 8), BorderColor: ui.Border, BorderWidth: 1,
				FlexDirection: "row", AlignItems: "center"},
			ui.Expand(ui.TextC("对话", *ui.FgSubtle, 12)),
			toolBtn("arrow-left-right", s.ThreadLeft, func() { s.ThreadLeft = !s.ThreadLeft; s.SetState() }),
			toolBtn("download", false, s.ExportActive),
			toolBtn("plus", false, func() { s.Store.NewThread(); s.saveHistory(); s.SetState() }),
		),
		// 对话列表：撑满上半部分 50%（Expanded Flex=1，另一半分给底部区域）
		&widget.Expanded{
			SingleChildWidget: widget.SingleChildWidget{
				Child: &widget.VirtualList{
					ItemCount:  len(threads),
					ItemHeight: itemH,
					Overscan:   5,
					RenderItem: func(i int) widget.Widget {
						return s.threadItem(threads[i])
					},
				},
			},
			Flex: 1,
		},
		// 明显的水平分割线（2px 粗，与对话列表明确分隔）
		widget.Div(widget.Style{Height: 3, BackgroundColor: ui.Border}),
		// 下半部分：环图卡片 + Token 统计卡片，用 Padding 包裹
		&widget.Expanded{
			SingleChildWidget: widget.SingleChildWidget{
				Child: widget.Div(
					widget.Style{FlexDirection: "column", AlignItems: "stretch",
						Padding: types.EdgeInsetsLTRB(8, 8, 8, 8)},
					// 卡片1：环图区（占一半）
					&widget.Expanded{
						SingleChildWidget: widget.SingleChildWidget{
							Child: widget.Div(
								widget.Style{
									BackgroundColor: ui.Bg,
									BorderRadius:    6,
									BorderColor:     ui.Border,
									BorderWidth:     1,
									FlexDirection:   "column",
									AlignItems:      "stretch",
								},
								s.donutChartSection(ctxUsed, ctxMax, cats),
							),
						},
						Flex: 1,
					},
					widget.Div(widget.Style{Height: 8}), // 卡片间距
					// 卡片2：Token 统计（占一半）
					&widget.Expanded{
						SingleChildWidget: widget.SingleChildWidget{
							Child: widget.Div(
								widget.Style{
									BackgroundColor: ui.Bg,
									BorderRadius:    6,
									BorderColor:     ui.Border,
									BorderWidth:     1,
									FlexDirection:   "column",
									AlignItems:      "stretch",
								},
								s.tokenStatsSection(tokenStats),
							),
						},
						Flex: 1,
					},
				),
			},
			Flex: 1,
		},
	)
}

// ─── 环形图 + Token 统计（侧栏上半部分） ──────────────────────

// donutCat 环形图分类项。
type donutCat struct {
	Label string       // 分类名
	Value int          // 数值
	Color *types.Color // 弧段颜色
}

// projectTokenStats 计算整个项目的 Token 统计（遍历所有会话）：
//   - contextUsed：所有会话的 prompt tokens 总和（用于进度环）
//   - contextMax：上下文上限 1M（1,048,576）
//   - cats：按消息内容拆分为上下文/工具/提示词/其他四类
//   - tokenStats：总/缓存命中/未命中/输出
func (s *ChatState) projectTokenStats() (contextUsed, contextMax int, cats []donutCat, tokenStats struct{ Total, CacheHit, CacheMiss, Output int }) {
	contextMax = 1048576 // 1M
	totalPrompt := 0
	totalCompletion := 0
	toolTokens := 0
	skillsTokens := 0
	mcpTokens := 0
	promptTokens := 0
	otherTokens := 0
	cacheHitTotal := 0   // 所有会话的 API 返回缓存命中 tokens 总和
	cacheMissTotal := 0  // 所有会话的 API 返回缓存未命中 tokens 总和

	for _, t := range s.Store.Threads {
		usage := t.CalculateTokenUsage()
		totalPrompt += usage.PromptTokens
		totalCompletion += usage.CompletionTokens

		// 累加 API 返回的真实缓存命中/未命中数据
		cacheHitTotal += t.TokenUsage.PromptCacheHitTokens
		cacheMissTotal += t.TokenUsage.PromptCacheMissTokens

		// 按消息内容分类 completion tokens
		for _, m := range t.Messages {
			if m.Role == state.Assistant {
				for _, a := range m.Activities {
					tn := a.Tool
					ch := len([]rune(tn)) + len([]rune(a.Args)) + len([]rune(a.Result))
					if strings.HasPrefix(tn, "skill_") || strings.HasPrefix(tn, "skills_") {
						skillsTokens += ch
					} else if strings.HasPrefix(tn, "mcp_") {
						mcpTokens += ch
					} else {
						toolTokens += ch
					}
				}
				for _, e := range m.Timeline {
					if e.Kind == "tool" {
						tn := e.Tool
						ch := len([]rune(tn)) + len([]rune(e.Args))
						if strings.HasPrefix(tn, "skill_") || strings.HasPrefix(tn, "skills_") {
							skillsTokens += ch
						} else if strings.HasPrefix(tn, "mcp_") {
							mcpTokens += ch
						} else {
							toolTokens += ch
						}
					}
				}
				promptTokens += len([]rune(m.Text)) + len([]rune(m.Thinking))
			}
		}
	}

	// 无数据时给默认占位
	// 无数据时给默认占位（不包含"上下文"——由进度环单独表示）
	if totalPrompt+totalCompletion == 0 {
		cats = []donutCat{
			{Label: "工具", Value: 0, Color: ui.Success},
			{Label: "Skills", Value: 0, Color: ui.Blue},
			{Label: "MCP", Value: 0, Color: ui.Purple},
			{Label: "提示词", Value: 0, Color: ui.Warning},
			{Label: "其他", Value: 0, Color: ui.FgMuted},
		}
		return
	}

	// 归一化分类值到 completionTokens 范围内
	allTokens := toolTokens + skillsTokens + mcpTokens + promptTokens + otherTokens
	if totalCompletion > 0 && allTokens > 0 {
		sum := float64(allTokens)
		toolTokens = int(float64(toolTokens) / sum * float64(totalCompletion))
		skillsTokens = int(float64(skillsTokens) / sum * float64(totalCompletion))
		mcpTokens = int(float64(mcpTokens) / sum * float64(totalCompletion))
		promptTokens = int(float64(promptTokens) / sum * float64(totalCompletion))
		otherTokens = totalCompletion - toolTokens - skillsTokens - mcpTokens - promptTokens
		if otherTokens < 0 {
			otherTokens = 0
		}
	}
	if toolTokens < 1 && totalCompletion > 0 && allTokens == 0 {
		toolTokens = 1
	}

	contextUsed = totalPrompt
	if contextUsed > contextMax {
		contextUsed = contextMax
	}

	// 分类明细不包含"上下文"——由进度环单独表示
	cats = []donutCat{
		{Label: "工具", Value: toolTokens, Color: ui.Success},
		{Label: "Skills", Value: skillsTokens, Color: ui.Blue},
		{Label: "MCP", Value: mcpTokens, Color: ui.Purple},
		{Label: "提示词", Value: promptTokens, Color: ui.Warning},
		{Label: "其他", Value: otherTokens, Color: ui.FgMuted},
	}

	total := totalPrompt + totalCompletion
	tokenStats.Total = total
	tokenStats.Output = totalCompletion
	if cacheHitTotal > 0 || cacheMissTotal > 0 {
		// 有 API 返回的真实缓存数据
		tokenStats.CacheHit = cacheHitTotal
		tokenStats.CacheMiss = cacheMissTotal
	} else {
		// 无真实数据时回退启发式估算（缓存命中约 60%）
		tokenStats.CacheHit = int(float64(totalPrompt) * 0.6)
		tokenStats.CacheMiss = totalPrompt - tokenStats.CacheHit
		if tokenStats.CacheMiss < 0 {
			tokenStats.CacheMiss = 0
		}
	}
	return
}

// donutChartSection 上下文进度环 + 分类明细（上下结构：环在上、文字在下）。
// 侧栏中间部分，已不包含 Token 统计（tokenStatsSection 独立展示）。
func (s *ChatState) donutChartSection(contextUsed, contextMax int, cats []donutCat) widget.Widget {
	ringSize := 90.0
	pct := 0.0
	if contextMax > 0 {
		pct = float64(contextUsed) * 100.0 / float64(contextMax)
	}
	centerText := fmt.Sprintf("%.0f%%", pct)

	// 分类明细——各分类值占总上下文的百分比
	ctxBase := float64(contextUsed)
	if ctxBase <= 0 {
		ctxBase = 1
	}
	// 将5个分类按2个一组放入行，用 Expanded 平分空间
	var detailRows []widget.Widget
	for i := 0; i < len(cats); i += 2 {
		rowKids := []widget.Widget{}
		for j := i; j < i+2 && j < len(cats); j++ {
			cc := cats[j]
			pct2 := float64(cc.Value) * 100.0 / ctxBase
			item := widget.Div(
				widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(2, 1, 2, 1)},
				widget.Div(widget.Style{Width: 8, Height: 8, BackgroundColor: cc.Color, BorderRadius: 4}),
				widget.Div(widget.Style{Width: 3}),
				ui.TextC(cc.Label, *ui.FgSubtle, 11),
				ui.Expand(widget.Div(widget.Style{})),
				ui.TextC(fmt.Sprintf("%.0f%%", pct2), *ui.Fg, 11),
			)
			rowKids = append(rowKids, &widget.Expanded{
				SingleChildWidget: widget.SingleChildWidget{Child: item},
				Flex: 1,
			})
		}
		detailRows = append(detailRows,
			widget.Div(
				widget.Style{FlexDirection: "row", AlignItems: "center"},
				rowKids,
			),
		)
	}

	// 上下结构：环在上，分类明细在下
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "center", Padding: types.EdgeInsetsLTRB(6, 6, 8, 6)},
		// 环居中（PaintLayer自绘环+百分比文字，MeasureText精确居中）
		widget.Div(
			widget.Style{Width: ringSize, Height: ringSize},
			&widget.PaintLayer{
				OnPaint: func(cvs2 canvas.Canvas, ox, oy, w, h float64) {
					// 先画环形
					paintProgressRing(cvs2, ox, oy, w, h, contextUsed, contextMax)
					// 再画中心百分比文字（精确居中）
					font := canvas.Font{Size: 13, Weight: canvas.FontWeightBold}
					tm := cvs2.MeasureText(centerText, font)
					inkH := tm.InkBottom - tm.InkTop
					if inkH <= 0 {
						inkH = font.Size * 1.2
					}
					tx := ox + (w-tm.Width)/2
					ty := oy + (h+inkH)/2 - tm.InkBottom
					tp := paint.DefaultPaint()
					tp.Color = *ui.Fg
					cvs2.DrawText(centerText, tx, ty, font, tp)
				},
			},
		),
		// 分类明细：3行2列网格，每项只显示白色百分比
		widget.Div(
			widget.Style{FlexDirection: "column", AlignItems: "stretch", Width: 240},
			detailRows,
		),
	)
}

// tokenStatsSection Token 统计
// tokenStatsSection Token 统计：四行分别显示总/缓存命中/未命中/输出，每行标签+值左对齐。
func (s *ChatState) tokenStatsSection(ts struct{ Total, CacheHit, CacheMiss, Output int }) widget.Widget {
	statRow := func(label string, value string, valColor types.Color) widget.Widget {
		return widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(0, 0, 0, 6)},
			ui.TextC(label, *ui.FgSubtle, 13),
			ui.Expand(widget.Div(widget.Style{})),
			ui.TextC(value, valColor, 14),
		)
	}
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch", Padding: types.EdgeInsetsLTRB(6, 6, 8, 6)},
		statRow("总", shortToken(ts.Total), *ui.Fg),
		statRow("缓存命中", shortToken(ts.CacheHit), *ui.Success),
		statRow("未命中", shortToken(ts.CacheMiss), *ui.Warning),
		statRow("输出", shortToken(ts.Output), *ui.Accent),
	)
}

// shortToken 把 token 数格式化为可读短字符串，如 1200→"1.2k"、1050000→"1.0M"。
func shortToken(n int) string {
	switch {
	case n >= 1000000:
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// paintProgressRing 绘制圆形进度环（单弧段）：灰色背景环 + 强调色进度弧（从12点顺时针），
// 低用量绿、中用量黄、高用量红。
func paintProgressRing(cvs canvas.Canvas, ox, oy, w, h float64, contextUsed, contextMax int) {
	cx := ox + w/2
	cy := oy + h/2
	outerR := w * 0.38
	innerR := outerR * 0.55
	degToRad := func(d float64) float64 { return d * math.Pi / 180.0 }

	// 计算进度比例
	progress := 0.0
	if contextMax > 0 {
		progress = float64(contextUsed) / float64(contextMax)
	}
	if progress > 1.0 {
		progress = 1.0
	}

	// ---- 1. 画灰色背景环（不填充中心镂空——留给调用方画文字） ----
	bgPath := &canvas.Path{}
	bgPath.MoveTo(cx, cy)
	bgPath.LineTo(cx+outerR, cy)
	bgPath.Arc(cx, cy, outerR, 0, 360, false)
	bgPath.LineTo(cx, cy)
	bgPath.Close()

	cvs.Save()
	cvs.Clip(bgPath)
	bgFill := paint.DefaultPaint()
	bgFill.Color = *ui.BgMuted
	bgFill.AntiAlias = true
	cvs.DrawCircle(cx, cy, outerR, bgFill)
	cvs.Restore()

	// 镂空中心（用BgSubtle圆形覆盖）
	holeFill := paint.DefaultPaint()
	holeFill.Color = *ui.BgSubtle
	holeFill.AntiAlias = true
	cvs.DrawCircle(cx, cy, innerR, holeFill)

	// ---- 2. 画进度弧段 ----
	if progress > 0.005 {
		span := 360.0 * progress
		if span < 1.0 {
			span = 1.0
		}
		startAngle := -90.0
		sRad := degToRad(startAngle)
		sx := cx + outerR*math.Cos(sRad)
		sy := cy + outerR*math.Sin(sRad)

		arcPath := &canvas.Path{}
		arcPath.MoveTo(cx, cy)
		arcPath.LineTo(sx, sy)
		arcPath.Arc(cx, cy, outerR, startAngle, startAngle+span, false)
		arcPath.LineTo(cx, cy)
		arcPath.Close()

		// 进度颜色：低绿、中黄、高红
		arcColor := *ui.Success
		if progress > 0.7 {
			arcColor = *ui.Warning
		}
		if progress > 0.9 {
			arcColor = *ui.Danger
		}

		cvs.Save()
		cvs.Clip(arcPath)
		arcFill := paint.DefaultPaint()
		arcFill.Color = arcColor
		arcFill.AntiAlias = true
		cvs.DrawCircle(cx, cy, outerR, arcFill)
		cvs.Restore()
	}

	// ---- 3. 外圈描边 ----
	sp := paint.DefaultStrokePaint()
	sp.Color = *ui.Border
	sp.StrokeWidth = 0.5
	cvs.DrawCircle(cx, cy, outerR, sp)
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
	// å¦æ user æ¶æ¯æé¿ææ¬æä»¶ï¼ä»æä»¶æ¢å¤åæï¼Text æ­¤æ¶ä»ä¸ºæä»¶å¼ç¨ï¼
	userText := user.Text
	if user.LongTextFile != "" {
		if data, err := os.ReadFile(user.LongTextFile); err == nil {
			userText = string(data)
		}
	}
	s.saveHistory()
	s.Bridge.Start(userText)
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
	ta.SubmitOnEnter = true // Enter 发送 / Shift+Enter 换行
	ta.OnSubmit = func(string) { s.Send() }
	ta.Wrap = true          // 长输入按宽折行，不横向滚动（用户：输入框要自动换行）
	ta.Text = s.Store.Draft // 重建时回填草稿（发送后随 SendSeq 复位为空）
	ta.ResetToken = s.SendSeq + s.DraftVersion
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
	if len(s.DraftRefs) > 0 {
		boxKids = append(boxKids, s.refChips())
	}
	if len(s.Attachments) > 0 {
		boxKids = append(boxKids, s.attachmentChips())
	}
	boxKids = append(boxKids,
		ui.Expand(ta), // textarea 撑满包裹盒减按钮栏的剩余高度
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(8, 4, 8, 8), FlexDirection: "row", AlignItems: "center"},
			iconGhost("paperclip", s.addAttachment),
			widget.Div(widget.Style{Width: 4}),
			iconGhost("folder", s.addAttachmentDir),
			ui.Expand(widget.Div(widget.Style{})),
			iconGhost("zap", func() { s.loadTestData() }), // ⚡性能测试：填充1000条消息验证虚加载
			widget.Div(widget.Style{Width: 5}),
			s.reviewToggle(),
			widget.Div(widget.Style{Width: 5}),
			toggleBtn("refresh-cw", "自主", s.Autonomous, ui.Blue, func() { s.Autonomous = !s.Autonomous; s.persistAgentToggles(); s.SetState() }),

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

// addAttachmentDir 目录附件：选择项目文件夹，递归添加到附件列表。
func (s *ChatState) addAttachmentDir() {
	if widget.OpenFolderDialog == nil {
		return
	}
	if p := widget.OpenFolderDialog("添加项目文件夹到附件"); p != "" {
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
				// 目录显示 folder 图标，文件显示 file-text 图标
				func() widget.Widget {
					iconName := "file-text"
					if info, err := os.Stat(p); err == nil && info.IsDir() {
						iconName = "folder"
					}
					return widget.Lucide(iconName, widget.IconSize(11), widget.IconColor(*ui.FgMuted))
				}(),
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
	totalLimit := 50000 // 所有附件总内容上限
	for _, p := range atts {
		if totalLimit <= 0 {
			b.WriteString("\n…（附件内容已达上限，剩余已忽略）")
			break
		}
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			// 目录：递归遍历，收集文件内容
			dirSize := 0
			filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil // skip dirs
				}
				// 跳过常见噪音目录
				rel, _ := filepath.Rel(p, path)
				if rel != "" {
					parts := strings.Split(rel, string(filepath.Separator))
					for _, seg := range parts {
						switch seg {
						case "node_modules", ".git", ".svn", "__pycache__", ".next",
							"target", "bin", "obj", "dist", "build", "vendor":
							return filepath.SkipDir
						}
					}
				}
				data, rerr := os.ReadFile(path)
				if rerr != nil {
					return nil
				}
				c := string(data)
				if len(c) > 20000 {
					c = c[:20000] + "\n…（已截断）"
				}
				if len(c)+dirSize > totalLimit {
					c = c[:totalLimit-dirSize]
					fmt.Fprintf(&b, "\n\n## %s\n%s\n…（目录内容已达上限）", path, c)
					totalLimit = 0
					return nil
				}
				dirSize += len(c)
				fmt.Fprintf(&b, "\n\n## %s\n%s", path, c)
				return nil
			})
			totalLimit -= dirSize
		} else {
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			content := string(data)
			if len(content) > 20000 {
				content = content[:20000] + "\n…（已截断）"
			}
			if len(content) > totalLimit {
				content = content[:totalLimit] + "\n…（已达附件总上限）"
			}
			fmt.Fprintf(&b, "\n\n## %s\n%s", p, content)
			totalLimit -= len(content)
		}
	}
	return b.String()
}

// addRef 右键菜单「添加到对话」：把引用文本加入 DraftRefs 列表，以 chip 组件展示。
func (s *ChatState) AddRef(text string) {
	s.DraftRefs = append(s.DraftRefs, text)
	s.SetState()
}

// removeRef 移除指定索引的引用 chip。
func (s *ChatState) removeRef(i int) {
	if i >= 0 && i < len(s.DraftRefs) {
		s.DraftRefs = append(s.DraftRefs[:i], s.DraftRefs[i+1:]...)
		s.SetState()
	}
}

// refLabel 从引用文本中提取简短 chip 标签（用于展示）。
func refLabel(text string) string {
	if strings.HasPrefix(text, "参考文件：") {
		return strings.TrimPrefix(text, "参考文件：")
	}
	if strings.HasPrefix(text, "参考目录：") {
		return strings.TrimPrefix(text, "参考目录：")
	}
	if strings.HasPrefix(text, "\x60\x60\x60\\n") {
		lines := strings.SplitN(text, "\\n", 3)
		if len(lines) > 1 && len(lines[0]) > 3 {
			return lines[0][3:] + " 代码片段"
		}
		return "代码片段"
	}
	if len(text) > 30 {
		return text[:30] + "\u2026"
	}
	return text
}

// refChips 待发送引用 chips（标签 + 移除按钮），类似 attachmentChips 风格。
func (s *ChatState) refChips() widget.Widget {
	var chips []widget.Widget
	for i, ref := range s.DraftRefs {
		idx := i
		label := refLabel(ref)
		chips = append(chips,
			widget.Div(
				widget.Style{FlexDirection: "row", AlignItems: "center", BackgroundColor: ui.BgMuted,
					BorderRadius: 4, Padding: types.EdgeInsetsLTRB(6, 3, 4, 3)},
				// 根据引用类型选择图标
				func() widget.Widget {
					iconName := "message-square"
					if strings.HasPrefix(ref, "参考文件：") {
						iconName = "file-text"
					} else if strings.HasPrefix(ref, "参考目录：") {
						iconName = "folder"
					} else if strings.HasPrefix(ref, "\x60\x60\x60") {
						iconName = "braces"
					}
					return widget.Lucide(iconName, widget.IconSize(11), widget.IconColor(*ui.FgMuted))
				}(),
				widget.Div(widget.Style{Width: 4}),
				ui.TextC(label, *ui.Fg, 11),
				widget.Div(widget.Style{Width: 3}),
				&widget.Clickable{
					SingleChildWidget: widget.SingleChildWidget{Child: widget.Lucide("x", widget.IconSize(11), widget.IconColor(*ui.FgMuted))},
					OnClick:           func() { s.removeRef(idx) },
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
		BorderColor: ui.Border, BorderWidth: 1, BorderRadius: 4,
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
	s.DraftRefs = nil
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

	// 长文本检测：超过 2000 字节 → 写入临时文件，chat 只显引用
	// 避免 VirtualList 中大量文本导致滚动跳动、查看不便。
	const longMsgThreshold = 2000
	var longTextFile string
	if len(draft) > longMsgThreshold {
		longDir := filepath.Join(".pair", "chat-long")
		_ = os.MkdirAll(longDir, 0755)
		fname := fmt.Sprintf("long-msg-%d.txt", time.Now().UnixNano())
		longPath := filepath.Join(longDir, fname)
		_ = os.WriteFile(longPath, []byte(draft), 0644)
		longTextFile = longPath
		display = fmt.Sprintf("[长消息 · 已保存至 %s，内容不在此展示以免滚动跳动]", longPath)
	}

	if len(atts) > 0 {
		if display != "" {
			display += "\n"
		}
		display += "[附件：" + attachmentNames(atts) + "]"
	}
	if !s.Store.Send(display) { // 只加 user 消息（Send 内部 trim+空判）
		return
	}
	// 如果有长文本临时文件路径，回写到最后一条消息记录（供 Regenerate 恢复原文）
	if longTextFile != "" {
		t := s.Store.Active()
		if t != nil && len(t.Messages) > 0 {
			t.Messages[len(t.Messages)-1].LongTextFile = longTextFile
		}
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
	refsText := ""
	if len(s.DraftRefs) > 0 {
		refsText = "\n\n# 用户引用\n" + strings.Join(s.DraftRefs, "\n\n")
	}
	s.Bridge.Start(draft + refsText + attachmentContext(atts)) // agent 任务含附件内容（内容只给 LLM、不污染显示）
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
