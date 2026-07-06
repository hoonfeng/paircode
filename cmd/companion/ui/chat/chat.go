// 对话面板 —— GWui 版（DOM 重写）。
// 布局：工具栏 + 可折叠对话侧栏 + 用户/Agent 消息卡 + 输入区。
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

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/paircode/cmd/companion/ui/state"
)

// ─── 辅助类型 ───────────────────────────────────────────────────

// planStep Agent 任务计划的一步（update_plan 工具传入）。
type planStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

// pendingAsk agent 经 ask_user 工具发起的提问（带可选项）。
type pendingAsk struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Multi    bool     `json:"multiSelect"`
}

// donutCat 环形图分类项。
type donutCat struct {
	Label string // 分类名
	Value int    // 数值
	Color string // 弧段颜色 CSS
}

// AgentBridge 是 ChatState 与 Agent 引擎的接口（agent_bridge.go 实现）。
type AgentBridge interface {
	IsRunning() bool
	RunningThread() *state.Thread
	Start(task string)
	Stop()
	ResolveAsk(answer string)
}

// ─── 包级变量 ───────────────────────────────────────────────────

var (
	TheState          *ChatState
	NewBridge         func(cs *ChatState) AgentBridge
	ClipboardWrite    func(text string)
	OpenFileDialog    func(title, filter string) string
	OpenFolderDialog  func(title string) string
	OnChatContextMenu func(x, y float64) // 右键菜单回调
	OnChatInputContextMenu func(x, y float64, selectedText string) // 输入框右键菜单回调
)

func elTag(el *dom.Element) string {
	if el == nil {
		return "nil"
	}
	id := el.GetAttribute("id")
	if id != "" {
		return "#" + id
	}
	return el.TagName()
}

// ─── DOM 辅助构建函数 ──────────────────────────────────────────

// createDiv 创建带 style 和子元素的 <div>。
func createDiv(doc *dom.Document, style string, children ...*dom.Element) *dom.Element {
	el := doc.CreateElement("div")
	if style != "" {
		el.SetAttribute("style", style)
	}
	for _, c := range children {
		if c != nil {
			el.AppendChild(c)
		}
	}
	return el
}

// createText 创建带文本内容和样式的 <span>。
func createText(doc *dom.Document, text, style string) *dom.Element {
	span := doc.CreateElement("span")
	span.SetTextContent(text)
	if style != "" {
		span.SetAttribute("style", style)
	}
	return span
}

// createIcon 创建 data-icon 属性的图标 <span>。
func createIcon(doc *dom.Document, name string, size int) *dom.Element {
	span := doc.CreateElement("span")
	span.SetAttribute("data-icon", name)
	s := fmt.Sprintf("width:%dpx;height:%dpx;display:inline-flex;align-items:center;justify-content:center;flex-shrink:0;", size, size)
	span.SetAttribute("style", s)
	return span
}

// iconButton 创建带图标的可点击元素（通用小型图标按钮）。
func iconButton(doc *dom.Document, iconName string, size int, onClick func()) *dom.Element {
	el := doc.CreateElement("div")
	el.SetAttribute("style",
		fmt.Sprintf("width:%dpx;height:%dpx;display:inline-flex;align-items:center;justify-content:center;cursor:pointer;border-radius:4px;flex-shrink:0;", size+8, size+8))
	el.SetAttribute("hover-style", "background-color: rgba(255,255,255,0.08);")
	el.AppendChild(createIcon(doc, iconName, size))
	addClick(el, onClick)
	return el
}

// spacer 创建弹性空白（flex:1 占位）。
func spacer(doc *dom.Document) *dom.Element {
	el := doc.CreateElement("div")
	el.SetAttribute("style", "flex:1;")
	return el
}

// vLine 创建垂直分隔线（1px 竖线）。
func vLine(doc *dom.Document) *dom.Element {
	return createDiv(doc, "width:1px;background-color:"+ui.Border+";flex-shrink:0;")
}

// gapW 创建指定宽度的水平间隙。
func gapW(doc *dom.Document, w int) *dom.Element {
	return createDiv(doc, fmt.Sprintf("width:%dpx;flex-shrink:0;", w))
}

// gapH 创建指定高度的垂直间隙。
func gapH(doc *dom.Document, h int) *dom.Element {
	return createDiv(doc, fmt.Sprintf("height:%dpx;flex-shrink:0;", h))
}

// statusDot 创建状态灯（小圆点）。
func statusDot(doc *dom.Document, color string) *dom.Element {
	return createDiv(doc, fmt.Sprintf("width:7px;height:7px;border-radius:50%%;background-color:%s;flex-shrink:0;", color))
}

// addClick 为元素添加 click 事件监听。
func addClick(el *dom.Element, fn func()) {
	if ui.Ctx.App == nil || fn == nil {
		return
	}
	ui.Ctx.App.AddEventListener(el, event.Click, func(e event.Event) bool {
		fn()
		return true
	})
}

// addHover 添加 mouseenter/mouseleave 事件监听。
func addHover(el *dom.Element, onEnter, onLeave func()) {
	if ui.Ctx.App == nil {
		return
	}
	if onEnter != nil {
		ui.Ctx.App.AddEventListener(el, event.MouseEnter, func(e event.Event) bool {
			onEnter()
			return true
		})
	}
	if onLeave != nil {
		ui.Ctx.App.AddEventListener(el, event.MouseLeave, func(e event.Event) bool {
			onLeave()
			return true
		})
	}
}

// markDirty 标记 UI 需要重绘。
func markDirty() {
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (s *ChatState) SetState() {
	markDirty()
}

func (s *ChatState) SaveHistory() {
	s.saveHistory()
}

// ─── ChatState ──────────────────────────────────────────────────

// ChatState 是对话面板的持久状态（包级单例）。
type ChatState struct {
	doc  *dom.Document
	root *dom.Element // 面板根元素

	Store        *state.ChatStore
	ShowThreads  bool
	AutoReview   bool
	Autonomous   bool
	AutoCollapse bool
	SendSeq      int
	DraftVersion int
	InputAreaH   float64
	Bridge       AgentBridge
	Plan         []planStep
	Ask          *pendingAsk
	Attachments  []string
	DraftRefs    []string
	PerfTest     int

	HoveredMsg  int
	ShowSearch  bool
	SearchQuery string
	searchSeq   int

	// DOM 元素引用（用于动态更新）
	toolbarEl    *dom.Element
	searchBarEl  *dom.Element
	msgAreaEl    *dom.Element // 消息区容器（overflow-y:auto，所有消息直接渲染）
	inputAreaEl  *dom.Element
	askCardEl    *dom.Element        // agent 提问卡占位（动态显示/隐藏）
	taskProgEl   *dom.Element        // 任务进度面板占位
	sidebarEl    *dom.Element        // 对话侧栏（固定 300px）
	mainColEl    *dom.Element        // 主列
	inputComp    *component.Input    // 搜索栏输入框
	textAreaComp *component.TextArea // 主输入框

	lastSaveTime time.Time

	// 轮询模式数据版本号（Agent goroutine 写入后递增，UI 线程检测后触发重建）
	DataVersion atomic.Int64

	// ─── 输入历史（上下键切换历史消息） ───
	msgHistory   []string // 已发送的消息记录（最新在末尾）
	historyIdx   int      // 当前浏览位置（-1=未浏览，0=最早）
	historyDraft string   // 进入历史浏览前保存的当前草稿

	_taskPS *taskProgressState
}

// taskProgressState 任务进度面板交互状态。
type taskProgressState struct{}

// ─── New ────────────────────────────────────────────────────────

// New 创建对话面板。
func New(doc *dom.Document) *ChatState {
	s := &ChatState{
		doc:        doc,
		Store:      state.NewChatStore(),
		ShowSearch: false,
		HoveredMsg: -1,
	}

	// ── 加载已有对话历史（.pair/conversations/history.json）──
	s.Store.Load(core.Root())

	// ── 加载 HTML 模板（资源目录 html/panels/chat.html）──
	ui.MustLoadPanelHTML(doc, "panels/chat.html", nil)
	root := doc.GetElementByID("chat-root")

	// 1. 工具栏：填充 HTML 提供的 #chat-toolbar 容器
	toolbar := doc.GetElementByID("chat-toolbar")
	s.buildToolbarInto(toolbar)

	// 2. 搜索栏：填充 HTML 提供的 #chat-searchbar 容器（初始隐藏）
	searchBar := doc.GetElementByID("chat-searchbar")
	s.buildSearchBarInto(searchBar)

	// 3. 消息区：使用 HTML 提供的 #chat-msg-area 容器（overflow-y:auto）
	msgArea := doc.GetElementByID("chat-msg-area")
	s.msgAreaEl = msgArea
	if OnChatContextMenu != nil {
		ui.Ctx.App.AddEventListener(msgArea, event.ContextMenu, func(e event.Event) bool {
			e.StopPropagation()
			if me, ok := e.(*event.MouseEvent); ok {
				OnChatContextMenu(float64(me.X), float64(me.Y))
			}
			return true
		})
	}

	// 4. Agent 提问卡占位：使用 HTML 提供的 #chat-ask-card
	s.askCardEl = doc.GetElementByID("chat-ask-card")

	// 5. 任务进度面板占位：使用 HTML 提供的 #chat-task-progress
	s.taskProgEl = doc.GetElementByID("chat-task-progress")

	// 6. 输入区：填充 HTML 提供的 #chat-input-area 容器
	inputArea := doc.GetElementByID("chat-input-area")
	s.buildInputAreaInto(inputArea)

	// 7. 侧栏容器：从 HTML 获取 #chat-sidebar
	s.sidebarEl = doc.GetElementByID("chat-sidebar")
	// 8. 主列（#chat-main）
	s.mainColEl = doc.GetElementByID("chat-main")

	s.root = root

	// 默认显示对话列表侧栏
	s.ShowThreads = true

	// 从临时父节点（body）中分离根元素
	ui.DetachRoot(root)

	TheState = s

	// 初始化消息列表 + 侧栏（含分隔条显隐+两栏重排）
	s.refreshMessageList()
	s.refreshSidebar()

	return s
}

// Element 返回面板根元素。
func (s *ChatState) Element() *dom.Element { return s.root }

// Refresh 刷新消息列表。
func (s *ChatState) Refresh() {
	s.refreshMessageList()
	s.refreshSidebar()
	markDirty()
}

// ─── 布局构建助手 ─────────────────────────────────────────────

// buildToolbarInto 将工具栏内容填充到 HTML 提供的容器中。
func (s *ChatState) buildToolbarInto(bar *dom.Element) {
	doc := s.doc

	// 标题
	title := doc.CreateElement("div")
	title.SetAttribute("style", "font-size:11px;color:#cccccc;text-transform:uppercase;letter-spacing:0.5px;padding-left:8px;white-space:nowrap;")
	title.SetTextContent("AI 对话")
	bar.AppendChild(title)

	bar.AppendChild(spacer(doc))

	// 搜索按钮
	searchBtn := iconButton(doc, "search", 14, func() {
		s.ToggleSearch()
	})
	bar.AppendChild(searchBtn)

	// 列表切换按钮
	listBtn := iconButton(doc, "list", 14, func() {
		s.ShowThreads = !s.ShowThreads
		s.refreshSidebar()
		markDirty()
	})
	bar.AppendChild(listBtn)

	s.toolbarEl = bar
}

// buildSearchBarInto 将搜索栏内容填充到 HTML 提供的容器中。
func (s *ChatState) buildSearchBarInto(sb *dom.Element) {
	doc := s.doc

	// 放大镜图标
	sb.AppendChild(createIcon(doc, "search", 13))

	sb.AppendChild(gapW(doc, 8))

	// 搜索输入框
	inp := component.NewInput(doc, "搜索对话内容…")
	inp.SetBaseStyle(
		"flex:1;background-color:" + ui.InputBg + ";color:" + ui.Text + ";" +
			"border:1px solid " + ui.InputBg + ";padding:2px 6px;font-size:13px;")
	inp.OnChange(func(q string) {
		s.SearchQuery = q
		s.refreshMessageList()
	})
	s.inputComp = inp
	sb.AppendChild(inp.Element())

	// 匹配计数
	sb.AppendChild(gapW(doc, 6))

	// 关闭按钮
	closeBtn := iconButton(doc, "x", 12, func() {
		s.ToggleSearch()
	})
	sb.AppendChild(closeBtn)

	s.searchBarEl = sb
}

// refreshMessageList 重建消息列表（直接渲染到 #chat-msg-area）。
func (s *ChatState) refreshMessageList() {
	t := s.Store.Active()
	if t == nil || s.msgAreaEl == nil {
		return
	}

	q := strings.ToLower(strings.TrimSpace(s.SearchQuery))

	// 清空并重建消息列表
	s.msgAreaEl.SetTextContent("")

	// 存放所有消息卡片的容器
	for i, m := range t.Messages {
		if q != "" && !msgMatches(m, q) {
			continue
		}
		el := s.renderMessage(t, i)
		if el != nil {
			s.msgAreaEl.AppendChild(el)
		}
	}

	// 滚动到底部（通过设置 scroll-y 属性）
	s.msgAreaEl.SetAttribute("scroll-y", "999999")
}

// buildVirtualItems 构建完整的 VirtualItem 列表。
// saveToMsgHistory 保存发送的消息到输入历史（上下键切换用）。
// 不保存空白、不保存重复（与最新一条相同），最多保留 100 条。
func (s *ChatState) saveToMsgHistory(text string) {
	if text == "" {
		return
	}
	if len(s.msgHistory) > 0 && s.msgHistory[len(s.msgHistory)-1] == text {
		return
	}
	const maxHistory = 100
	if len(s.msgHistory) >= maxHistory {
		s.msgHistory = s.msgHistory[1:]
	}
	s.msgHistory = append(s.msgHistory, text)
}

// ─── 消息渲染 ──────────────────────────────────────────────────

// renderMessage 渲染单条消息。
func (s *ChatState) renderMessage(t *state.Thread, i int) *dom.Element {
	m := t.Messages[i]
	doc := s.doc

	var card *dom.Element
	if m.Role == state.User {
		card = userCard(doc, m)
	} else {
		card = agentMessageCard(doc, m,
			func() { t.Messages[i].Collapsed = !t.Messages[i].Collapsed; s.refreshMessageList(); markDirty() },
			func() {
				t.Messages[i].ThinkingExpanded = !t.Messages[i].ThinkingExpanded
				s.refreshMessageList()
				markDirty()
			},
			func(ti int) {
				if ti >= 0 && ti < len(t.Messages[i].Timeline) && t.Messages[i].Timeline[ti].Kind == "tool" {
					t.Messages[i].Timeline[ti].Expanded = !t.Messages[i].Timeline[ti].Expanded
					cid := t.Messages[i].Timeline[ti].CallID
					for ai := range t.Messages[i].Activities {
						if t.Messages[i].Activities[ai].CallID == cid {
							t.Messages[i].Activities[ai].Expanded = t.Messages[i].Timeline[ti].Expanded
							break
						}
					}
					s.refreshMessageList()
					markDirty()
				} else if ti >= 0 && ti < len(t.Messages[i].Activities) {
					t.Messages[i].Activities[ti].Expanded = !t.Messages[i].Activities[ti].Expanded
					s.refreshMessageList()
					markDirty()
				}
			},
		)
	}

	// 消息外层容器（带 padding：左右 16px 避免消息贴边）
	outer := createDiv(doc, "padding:10px 16px;flex-direction:column;", card)

	// 操作按钮（hover 显示）
	if !m.Streaming {
		actions := s.buildMessageActions(t, i)
		if actions != nil {
			if m.Role == state.User {
				// 用户消息：按钮在气泡下方，右对齐，不覆盖文字
				actions.SetAttribute("style",
					"display:none;flex-direction:row;align-items:center;justify-content:flex-end;gap:4px;"+
						"padding:4px 0 0 0;")
				// 外层：卡片在上，按钮在下
				wrapper := doc.CreateElement("div")
				wrapper.SetAttribute("style", "display:flex;flex-direction:column;")
				wrapper.AppendChild(card)
				wrapper.AppendChild(actions)
				addHover(wrapper,
					func() {
						s.HoveredMsg = i
						actions.SetAttribute("style",
							"display:flex;flex-direction:row;align-items:center;justify-content:flex-end;gap:4px;"+
								"padding:4px 0 0 0;")
					},
					func() {
						if s.HoveredMsg == i {
							s.HoveredMsg = -1
						}
						actions.SetAttribute("style",
							"display:none;flex-direction:row;align-items:center;justify-content:flex-end;gap:4px;"+
								"padding:4px 0 0 0;")
					},
				)
				outer = createDiv(doc, "padding:10px 16px;flex-direction:column;", wrapper)
			} else {
				// Assistant 消息：保留右上角浮层按钮（不遮挡文字）
				actionsStyle := "position:absolute;top:8px;right:8px;"
				actions.SetAttribute("style", actionsStyle+";display:none;flex-direction:row;align-items:center;gap:4px;"+
					"background-color:"+ui.SideBg+";border:1px solid "+ui.Border+";border-radius:8px;padding:3px;z-index:10;")
				wrapper := doc.CreateElement("div")
				wrapper.SetAttribute("style", "position:relative;")
				wrapper.AppendChild(card)
				wrapper.AppendChild(actions)
				addHover(wrapper,
					func() {
						s.HoveredMsg = i
						actions.SetAttribute("style",
							actionsStyle+"display:flex;flex-direction:row;align-items:center;gap:4px;"+
								"background-color:"+ui.SideBg+";border:1px solid "+ui.Border+";border-radius:8px;padding:3px;z-index:10;")
					},
					func() {
						if s.HoveredMsg == i {
							s.HoveredMsg = -1
						}
						actions.SetAttribute("style",
							actionsStyle+"display:none;flex-direction:row;align-items:center;gap:4px;"+
								"background-color:"+ui.SideBg+";border:1px solid "+ui.Border+";border-radius:8px;padding:3px;z-index:10;")
					},
				)
				outer = createDiv(doc, "padding:10px 16px;flex-direction:column;", wrapper)
			}
		}
	}

	// 消息间间距
	container := createDiv(doc, "flex-direction:column;", outer, gapH(doc, 8))
	return container
}

// buildMessageActions 建浮层操作按钮（复制/重新生成/删除）。
func (s *ChatState) buildMessageActions(t *state.Thread, i int) *dom.Element {
	m := t.Messages[i]
	if m.Streaming {
		return nil
	}
	doc := s.doc

	actions := doc.CreateElement("div")
	actions.SetAttribute("style",
		"display:none;flex-direction:row;align-items:center;gap:4px;"+
			"background-color:"+ui.SideBg+";border:1px solid "+ui.Border+";border-radius:8px;padding:3px;z-index:10;")

	// 复制按钮：所有消息都显示（文本非空时）
	if m.Text != "" {
		btn := iconButton(doc, "copy", 13, func() {
			if ClipboardWrite != nil {
				ClipboardWrite(m.Text)
			}
		})
		actions.AppendChild(btn)
	}

	// 重试按钮：仅用户消息的最后一条
	if m.Role == state.User && i == len(t.Messages)-1 && !s.agentBusy() {
		btn := iconButton(doc, "refresh-cw", 13, func() { s.Regenerate(t, i) })
		actions.AppendChild(btn)
	}

	// 删除按钮：仅用户消息
	if m.Role == state.User {
		btn := iconButton(doc, "trash-2", 13, func() { s.DeleteMessage(t, i) })
		actions.AppendChild(btn)
	}

	return actions
}

// agentMessageCard 渲染 Agent 消息卡片。
func agentMessageCard(doc *dom.Document, m state.Message, toggleCollapse, toggleThinking func(), toggleActivity func(int)) *dom.Element {
	if m.Collapsed && !m.Streaming {
		return buildCollapsedCard(doc, m, toggleCollapse)
	}
	return buildExpandedCard(doc, m, toggleCollapse, toggleThinking, toggleActivity)
}

// buildCollapsedCard 折叠态消息卡（摘要行）。
func buildCollapsedCard(doc *dom.Document, m state.Message, toggleCollapse func()) *dom.Element {
	card := doc.CreateElement("div")
	card.SetAttribute("style",
		"display:flex;flex-direction:row;align-items:center;height:36px;padding:0 12px;cursor:pointer;border-radius:6px;"+
			"background-color:"+ui.AssistantBg+";border:1px solid "+ui.Border+";")
	card.SetAttribute("hover-style", "background-color:"+ui.HoverBg+";")

	// 展开箭头
	arrow := createIcon(doc, "chevron-right", 14)
	arrow.SetAttribute("style", arrow.GetAttribute("style")+";color:"+ui.TextMute+";")
	card.AppendChild(arrow)
	card.AppendChild(gapW(doc, 6))

	// 角色标签
	role := createText(doc, "Agent", "color:"+ui.Accent+";font-size:12px;font-weight:bold;flex-shrink:0;")
	card.AppendChild(role)
	card.AppendChild(gapW(doc, 8))

	// 摘要文本
	summary := m.Text
	if len([]rune(summary)) > 60 {
		summary = string([]rune(summary)[:60]) + "…"
	}
	summary = strings.ReplaceAll(summary, "\n", " ")
	summaryEl := createText(doc, summary, "color:"+ui.TextMute+";font-size:12px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;flex:1;")
	card.AppendChild(summaryEl)

	addClick(card, toggleCollapse)
	return card
}

// buildExpandedCard 展开态消息卡。
func buildExpandedCard(doc *dom.Document, m state.Message, toggleCollapse, toggleThinking func(), toggleActivity func(int)) *dom.Element {
	card := doc.CreateElement("div")

	// ★ 流式状态：添加 accent 色左边框（伴随式codeagent风格）
	borderColor := ui.Border
	if m.Streaming {
		borderColor = ui.Accent
	}
	card.SetAttribute("style",
		"display:flex;flex-direction:column;padding:8px 12px;border-radius:6px;"+
			"background-color:"+ui.AssistantBg+";border:1px solid "+borderColor+";")

	// ── header 行 ──
	header := doc.CreateElement("div")
	header.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;margin-bottom:4px;")

	// 折叠按钮
	collapseBtn := createIcon(doc, "chevron-down", 14)
	collapseBtn.SetAttribute("style", collapseBtn.GetAttribute("style")+";cursor:pointer;color:"+ui.TextMute+";")
	collapseBtn.SetAttribute("hover-style", "color:"+ui.Text+";")
	addClick(collapseBtn, toggleCollapse)
	header.AppendChild(collapseBtn)
	header.AppendChild(gapW(doc, 6))

	role := createText(doc, "Agent", "color:"+ui.Accent+";font-size:12px;font-weight:bold;")
	header.AppendChild(role)
	header.AppendChild(spacer(doc))

	if m.Streaming {
		// ★ 流式光标（伴随式codeagent风格：闪烁的点）
		cursor := createDiv(doc,
			"width:6px;height:6px;border-radius:50%;background-color:"+ui.Accent+";flex-shrink:0;"+
				"animation:chat-cursor-blink 1.2s ease-in-out infinite;margin-right:4px;")
		header.AppendChild(cursor)
		spin := createText(doc, "思考中…", "color:"+ui.Warning+";font-size:11px;")
		header.AppendChild(spin)
	}
	card.AppendChild(header)

	// ── Thinking 块 ──
	if strings.TrimSpace(m.Thinking) != "" {
		thinkingSec := buildThinkingSection(doc, m.Thinking, m.ThinkingExpanded || m.Streaming, toggleThinking)
		card.AppendChild(thinkingSec)
	}

	// ── Timeline / Activities ──
	if len(m.Timeline) > 0 {
		// 按 Timeline 事件流顺序渲染
		for ti, entry := range m.Timeline {
			switch entry.Kind {
			case "thinking":
				if strings.TrimSpace(entry.Content) != "" {
					expanded := m.ThinkingExpanded || m.Streaming
					ts := buildTimelineThinking(doc, entry.Content, expanded, toggleThinking)
					card.AppendChild(ts)
				}
			case "content":
				if txt := strings.TrimSpace(entry.Content); txt != "" {
					contentEl := buildContentBlock(doc, txt)
					card.AppendChild(contentEl)
				}
			case "tool":
				ta := entry.TimelineEntryAsActivity()
				taEl := buildActivityItem(doc, ta, func() { toggleActivity(ti) })
				card.AppendChild(taEl)
			}
		}
	} else {
		// 向后兼容：无 Timeline 时使用旧版渲染
		if strings.TrimSpace(m.Thinking) != "" && len(m.Timeline) == 0 {
			// 已在上方渲染，避免重复
		}
		for ai := range m.Activities {
			act := m.Activities[ai]
			idx := ai
			actEl := buildActivityItem(doc, act, func() { toggleActivity(idx) })
			card.AppendChild(actEl)
		}
	}

	// ── 正文 ──
	if txt := strings.TrimSpace(m.Text); txt != "" && len(m.Timeline) == 0 {
		contentEl := buildContentBlock(doc, txt)
		card.AppendChild(contentEl)
	}

	// ── Notes ──
	for _, note := range m.Notes {
		noteEl := createText(doc, note, "display:block;color:"+ui.TextMute+";font-size:11px;padding:2px 0;")
		card.AppendChild(noteEl)
	}

	// ── Eval ──
	if m.Eval != nil {
		evalEl := buildEvalBlock(doc, m.Eval)
		card.AppendChild(evalEl)
	}

	return card
}

// buildThinkingSection 构建思考链折叠块。
func buildThinkingSection(doc *dom.Document, thinking string, expanded bool, toggle func()) *dom.Element {
	sec := doc.CreateElement("div")
	sec.SetAttribute("style",
		"display:flex;flex-direction:column;margin:4px 0;padding:6px 8px;background-color:"+ui.ThinkingBg+";border-radius:4px;border-left:2px solid "+ui.TextMute+";")

	header := doc.CreateElement("div")
	header.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;cursor:pointer;")
	addClick(header, toggle)

	iconName := "chevron-right"
	if expanded {
		iconName = "chevron-down"
	}
	arrow := createIcon(doc, iconName, 12)
	arrow.SetAttribute("style", arrow.GetAttribute("style")+";color:"+ui.TextMute+";")
	header.AppendChild(arrow)
	header.AppendChild(gapW(doc, 4))

	label := createText(doc, "思考", "color:"+ui.TextDim+";font-size:11px;font-weight:bold;")
	header.AppendChild(label)

	// 折叠时显示第一行摘要
	if !expanded {
		summaryText := firstLine(thinking)
		summary := createText(doc, "  "+summaryText, "color:"+ui.TextDim+";font-size:11px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;flex:1;")
		header.AppendChild(summary)
	}
	sec.AppendChild(header)

	if expanded {
		// 展开时也只显示前 6 行（简化展示）
		simplified := simplifyThinking(thinking)
		content := createText(doc, simplified, "display:block;color:"+ui.TextDim+";font-size:12px;margin-top:4px;white-space:pre-wrap;word-break:break-word;line-height:1.5;")
		sec.AppendChild(content)
	}
	return sec
}

// firstLine 返回文本的第一行（去空）。

// buildTimelineThinking 基于 Timeline 渲染思考块。
func buildTimelineThinking(doc *dom.Document, content string, expanded bool, toggle func()) *dom.Element {
	return buildThinkingSection(doc, content, expanded, toggle)
}

// buildContentBlock 渲染消息正文段落。
func buildContentBlock(doc *dom.Document, text string) *dom.Element {
	return createText(doc, text,
		"display:block;color:"+ui.Text+";font-size:13px;margin:4px 0;white-space:pre-wrap;word-break:break-word;line-height:1.5;")
}

// toolIconName 根据工具名返回合适的图标名称（参照伴随式codeagent的图标分类）。
func toolIconName(tool string) string {
	switch {
	case strings.HasPrefix(tool, "read_file") || strings.HasPrefix(tool, "read_"):
		return "file-text"
	case strings.HasPrefix(tool, "edit_file") || strings.HasPrefix(tool, "write_file") ||
		strings.HasPrefix(tool, "create_file") || tool == "file":
		return "code"
	case strings.HasPrefix(tool, "shell") || strings.HasPrefix(tool, "bash") ||
		strings.HasPrefix(tool, "powershell") || strings.HasPrefix(tool, "run_"):
		return "terminal"
	case strings.HasPrefix(tool, "search") || strings.HasPrefix(tool, "grep"):
		return "search"
	case strings.HasPrefix(tool, "web_fetch") || strings.HasPrefix(tool, "web_search"):
		return "globe"
	case strings.HasPrefix(tool, "mcp_"):
		return "box"
	case strings.HasPrefix(tool, "skill_"):
		return "layers"
	case strings.HasPrefix(tool, "ask_user"):
		return "help-circle"
	case strings.HasPrefix(tool, "memory_"):
		return "database"
	case strings.HasPrefix(tool, "git_"):
		return "git-branch"
	default:
		return "terminal"
	}
}
func buildActivityItem(doc *dom.Document, a state.Activity, toggle func()) *dom.Element {
	item := doc.CreateElement("div")
	item.SetAttribute("style",
		"display:flex;flex-direction:column;margin:2px 0;padding:4px 8px;background-color:"+ui.ToolActivity+";border-radius:4px;cursor:pointer;")
	item.SetAttribute("hover-style", "background-color:"+ui.HoverBg+";")
	addClick(item, toggle)

	// 首行：工具名 + 状态
	row1 := doc.CreateElement("div")
	row1.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;")

	// ★ 参考伴随式codeagent：按工具类型选择图标
	iconName := toolIconName(a.Tool)
	icon := createIcon(doc, iconName, 11)
	icon.SetAttribute("style", icon.GetAttribute("style")+";color:"+ui.TextMute+";")
	row1.AppendChild(icon)
	row1.AppendChild(gapW(doc, 4))

	name := createText(doc, a.Tool, "color:"+ui.TextDim+";font-size:11px;flex:1;")
	row1.AppendChild(name)

	if a.Done {
		doneIcon := createIcon(doc, "check", 10)
		doneIcon.SetAttribute("style", doneIcon.GetAttribute("style")+";color:"+ui.Success+";")
		row1.AppendChild(doneIcon)
	} else if a.AwaitingApproval {
		waitIcon := createText(doc, "?", "color:"+ui.Warning+";font-size:11px;")
		row1.AppendChild(waitIcon)
	} else {
		spin := createText(doc, "…", "color:"+ui.Warning+";font-size:11px;")
		row1.AppendChild(spin)
	}
	item.AppendChild(row1)

	// 参数预览
	if a.Expanded && a.Args != "" {
		args := createText(doc, ArgPreview(a.Args), "display:block;color:"+ui.TextMute+";font-size:10px;margin-top:2px;white-space:pre-wrap;font-family:monospace;")
		item.AppendChild(args)
	}

	// 结果预览
	if a.Expanded && a.Result != "" {
		result := createText(doc, a.Result, "display:block;color:"+ui.TextDim+";font-size:11px;margin-top:2px;white-space:pre-wrap;")
		item.AppendChild(result)
	}

	return item
}

// buildEvalBlock 渲染评测结果。
func buildEvalBlock(doc *dom.Document, e *state.Eval) *dom.Element {
	return createDiv(doc,
		"margin:4px 0;padding:6px 8px;background-color:"+ui.ThinkingBg+";border-radius:4px;border-left:2px solid "+ui.Accent+";flex-direction:column;",
		createText(doc, fmt.Sprintf("评测: %d/100 (完成%d 正确%d 深度%d 效率%d)",
			e.Total, e.Completion, e.Correctness, e.Depth, e.Efficiency),
			"color:"+ui.Text+";font-size:11px;"),
	)
}

// userCard 渲染用户消息卡片（右对齐气泡风格，参照伴随式codeagent）。
func userCard(doc *dom.Document, m state.Message) *dom.Element {
	return createDiv(doc,
		"display:flex;flex-direction:row;justify-content:flex-end;padding:0;", // 水平间距由外层容器统一提供
		createDiv(doc,
			"display:inline-flex;flex-direction:column;padding:8px 14px;border-radius:10px 10px 2px 10px;"+
				"background-color:"+ui.UserBubble+";"+
				"max-width:100%;word-break:break-word;",
			createText(doc, m.Text,
				"display:block;color:#e0e0e0;font-size:13px;white-space:pre-wrap;word-break:break-word;line-height:1.5;font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;"),
		),
	)
}

// ─── 输入区 ─────────────────────────────────────────────────────

// buildInputAreaInto 将输入区内容填充到 HTML 提供的容器中。
func (s *ChatState) buildInputAreaInto(box *dom.Element) {
	doc := s.doc

	// 引用 chips 行（DraftRefs）
	refsRow := doc.CreateElement("div")
	refsRow.SetAttribute("style", "display:none;flex-direction:row;flex-wrap:wrap;align-items:center;padding:4px 8px 0 8px;gap:4px;")
	box.AppendChild(refsRow)

	// 附件 chips 行
	attRow := doc.CreateElement("div")
	attRow.SetAttribute("style", "display:none;flex-direction:row;flex-wrap:wrap;align-items:center;padding:4px 8px 0 8px;gap:4px;")
	box.AppendChild(attRow)

	// 输入框包裹（圆角边框）
	inputBox := doc.CreateElement("div")
	inputBox.SetAttribute("style",
		"display:flex;flex-direction:column;margin:8px;padding:0;border:1px solid "+ui.Border+";border-radius:8px;background-color:"+ui.ChatBg+";flex:1;")

	// TextArea
	ta := component.NewTextArea(doc, "", 300, 100)
	if s.Store.Draft != "" {
		ta.SetText(s.Store.Draft)
	}
	ta.SetBaseStyle(
		"background-color:transparent;color:" + ui.Text + ";border:none;padding:8px;font-size:13px;outline:none;width:100%;min-height:160px;resize:none;")
	ta.SetHoverStyle("background-color:transparent;")
	ta.SetFocusStyle("background-color:transparent;")
	ta.OnChange(func(text string) {
		s.Store.Draft = text
	})

	// 右键菜单
	if OnChatInputContextMenu != nil {
		ta.OnContextMenu(OnChatInputContextMenu)
	}

	// 键盘事件：Enter 提交 + 上下键历史消息
	ui.Ctx.App.AddEventListener(ta.Element(), event.KeyDown, func(e event.Event) bool {
		ke, ok := e.(*event.KeyboardEvent)
		if !ok {
			return false
		}
		// Enter 提交（Shift+Enter 保留为换行）
		if ke.Key == event.CodeEnter && !ke.Shift {
			s.Send("")
			return true
		}
		// 上键：浏览历史消息（向上）
		if ke.Key == event.CodeUp && !ke.Shift && !ke.Ctrl {
			if len(s.msgHistory) == 0 {
				return true
			}
			if s.historyIdx == -1 {
				// 首次进入历史：保存当前草稿
				s.historyDraft = ta.Text()
				s.historyIdx = len(s.msgHistory) - 1
			} else if s.historyIdx > 0 {
				s.historyIdx--
			}
			text := s.msgHistory[s.historyIdx]
			ta.SetText(text)
			ta.SetCursorPos(len([]rune(text)))
			return true
		}
		// 下键：浏览历史消息（向下）
		if ke.Key == event.CodeDown && !ke.Shift && !ke.Ctrl {
			if s.historyIdx == -1 {
				return true
			}
			if s.historyIdx == len(s.msgHistory)-1 {
				// 最新位置：恢复保存的草稿
				ta.SetText(s.historyDraft)
				ta.SetCursorPos(len([]rune(s.historyDraft)))
				s.historyIdx = -1
				s.historyDraft = ""
			} else {
				s.historyIdx++
				text := s.msgHistory[s.historyIdx]
				ta.SetText(text)
				ta.SetCursorPos(len([]rune(text)))
			}
			return true
		}
		return false
	})

	inputBox.AppendChild(ta.Element())

	// 底部工具按钮栏
	toolbar := doc.CreateElement("div")
	toolbar.SetAttribute("style",
		"display:flex;flex-direction:row;align-items:center;padding:4px 8px 8px 8px;")

	// 附件按钮
	attachBtn := iconButton(doc, "paperclip", 13, func() { s.addAttachment() })
	toolbar.AppendChild(attachBtn)
	toolbar.AppendChild(gapW(doc, 4))

	// 文件夹按钮
	folderBtn := iconButton(doc, "folder", 13, func() { s.addAttachmentDir() })
	toolbar.AppendChild(folderBtn)
	toolbar.AppendChild(spacer(doc))

	// 性能测试按钮（⚡）
	zapBtn := iconButton(doc, "zap", 13, func() { s.loadTestData() })
	toolbar.AppendChild(zapBtn)
	toolbar.AppendChild(gapW(doc, 5))

	// 审核开关
	reviewBtn := s.buildReviewToggle()
	toolbar.AppendChild(reviewBtn)
	toolbar.AppendChild(gapW(doc, 5))

	// 自主模式开关
	autoBtn := s.buildToggleBtn("refresh-cw", "自主", s.Autonomous, ui.Info, func() {
		s.Autonomous = !s.Autonomous
		s.persistAgentToggles()
		refreshInputArea(s)
	})
	toolbar.AppendChild(autoBtn)
	toolbar.AppendChild(gapW(doc, 8))

	// 发送/停止按钮
	sendBtn := s.buildSendOrStop()
	toolbar.AppendChild(sendBtn)

	inputBox.AppendChild(toolbar)
	box.AppendChild(inputBox)

	s.textAreaComp = ta
	s.inputAreaEl = box
}

// buildReviewToggle 构建审核模式切换按钮。
func (s *ChatState) buildReviewToggle() *dom.Element {
	doc := s.doc
	iconName, text, clr := "shield", "自动", ui.Success
	if !s.AutoReview {
		iconName, text, clr = "shield-off", "手动", ui.Warning
	}

	btn := doc.CreateElement("div")
	btn.SetAttribute("style",
		"display:inline-flex;flex-direction:row;align-items:center;gap:4px;padding:3px 9px 3px 10px;height:24px;"+
			"cursor:pointer;border:1px solid "+clr+";border-radius:5px;background-color:"+tintCSS(clr, 30)+";font-size:11px;color:"+clr+";flex-shrink:0;")
	btn.SetAttribute("hover-style", "opacity:0.9;")

	iconEl := createIcon(doc, iconName, 13)
	btn.AppendChild(iconEl)

	textEl := createText(doc, text, "color:"+clr+";font-size:11px;")
	btn.AppendChild(textEl)

	addClick(btn, func() {
		s.AutoReview = !s.AutoReview
		s.persistAgentToggles()
		// 重建输入区
		refreshInputArea(s)
	})

	return btn
}

// buildToggleBtn 构建开关按钮（带图标+文字）。
func (s *ChatState) buildToggleBtn(icon, text string, on bool, onColor string, onClick func()) *dom.Element {
	doc := s.doc
	fg, border, bg := ui.TextMute, ui.Border, "transparent"
	if on {
		fg = onColor
		border = onColor
		bg = tintCSS(onColor, 30)
	}
	btn := doc.CreateElement("div")
	btn.SetAttribute("style",
		"display:inline-flex;flex-direction:row;align-items:center;gap:4px;padding:3px 9px 3px 10px;height:24px;"+
			"cursor:pointer;border:1px solid "+border+";border-radius:5px;background-color:"+bg+";font-size:11px;color:"+fg+";flex-shrink:0;")
	btn.SetAttribute("hover-style", "opacity:0.9;")

	iconEl := createIcon(doc, icon, 13)
	btn.AppendChild(iconEl)

	textEl := createText(doc, text, "color:"+fg+";font-size:11px;")
	btn.AppendChild(textEl)

	addClick(btn, onClick)
	return btn
}

// buildSendOrStop 构建发送/停止按钮。
func (s *ChatState) buildSendOrStop() *dom.Element {
	doc := s.doc
	if s.Bridge != nil && s.Bridge.IsRunning() {
		btn := doc.CreateElement("div")
		btn.SetAttribute("style",
			"display:inline-flex;flex-direction:row;align-items:center;gap:6px;padding:3px 12px 3px 13px;height:24px;"+
				"cursor:pointer;border-radius:5px;background-color:"+ui.Error+";color:#fff;font-size:12px;flex-shrink:0;")
		btn.SetAttribute("hover-style", "opacity:0.9;")

		iconEl := createIcon(doc, "square", 12)
		iconEl.SetAttribute("style", iconEl.GetAttribute("style")+";color:#fff;")
		btn.AppendChild(iconEl)

		textEl := createText(doc, "停止", "color:#fff;font-size:12px;")
		btn.AppendChild(textEl)

		addClick(btn, func() {
			if s.Bridge != nil {
				s.Bridge.Stop()
			}
		})
		return btn
	}
	return s.buildPrimaryBtn("发送", func() { s.Send("") })
}

// buildPrimaryBtn 构建主发送按钮。
func (s *ChatState) buildPrimaryBtn(text string, onClick func()) *dom.Element {
	doc := s.doc
	btn := doc.CreateElement("div")
	btn.SetAttribute("style",
		"display:inline-flex;flex-direction:row;align-items:center;gap:6px;padding:3px 12px 3px 13px;height:24px;"+
			"cursor:pointer;border-radius:5px;background-color:"+ui.Accent+";color:#fff;font-size:12px;flex-shrink:0;")
	btn.SetAttribute("hover-style", "background-color:"+ui.AccentHover+";")

	iconEl := createIcon(doc, "send", 13)
	iconEl.SetAttribute("style", iconEl.GetAttribute("style")+";color:#fff;")
	btn.AppendChild(iconEl)

	textEl := createText(doc, text, "color:#fff;font-size:12px;")
	btn.AppendChild(textEl)

	addClick(btn, onClick)
	return btn
}

// refreshInputArea 重建输入区 DOM（审核/自主开关变化时）。
// 输入区容器 (#chat-input-area) 来自 HTML 模板，持久存在；此处清空并重建其内容。
func refreshInputArea(s *ChatState) {
	if s.inputAreaEl != nil {
		s.inputAreaEl.ClearChildren()
		s.buildInputAreaInto(s.inputAreaEl)
	}
	markDirty()
}

// ─── 附件 ──────────────────────────────────────────────────────

func (s *ChatState) addAttachment() {
	if OpenFileDialog == nil {
		return
	}
	if p := OpenFileDialog("添加附件", "所有文件|*.*"); p != "" {
		s.Attachments = append(s.Attachments, p)
		markDirty()
	}
}

func (s *ChatState) addAttachmentDir() {
	if OpenFolderDialog == nil {
		return
	}
	if p := OpenFolderDialog("添加项目文件夹到附件"); p != "" {
		s.Attachments = append(s.Attachments, p)
		markDirty()
	}
}

func (s *ChatState) removeAttachment(i int) {
	if i >= 0 && i < len(s.Attachments) {
		s.Attachments = append(s.Attachments[:i], s.Attachments[i+1:]...)
		markDirty()
	}
}

func (s *ChatState) AddRef(text string) {
	s.DraftRefs = append(s.DraftRefs, text)
	markDirty()
}

func (s *ChatState) removeRef(i int) {
	if i >= 0 && i < len(s.DraftRefs) {
		s.DraftRefs = append(s.DraftRefs[:i], s.DraftRefs[i+1:]...)
		markDirty()
	}
}

// attachmentChips 构建附件 chips 行。
func (s *ChatState) attachmentChips() *dom.Element {
	doc := s.doc
	row := doc.CreateElement("div")
	row.SetAttribute("style", "display:flex;flex-direction:row;flex-wrap:wrap;align-items:center;padding:0 8px;gap:6px;")

	for i, p := range s.Attachments {
		idx := i
		chip := buildChip(doc, p, func() { s.removeAttachment(idx) })
		row.AppendChild(chip)
	}
	return row
}

// refChips 构建引用 chips 行。
func (s *ChatState) refChips() *dom.Element {
	doc := s.doc
	row := doc.CreateElement("div")
	row.SetAttribute("style", "display:flex;flex-direction:row;flex-wrap:wrap;align-items:center;padding:0 8px;gap:6px;")

	for i, ref := range s.DraftRefs {
		idx := i
		label := refLabel(ref)
		chip := buildRefChip(doc, label, ref, func() { s.removeRef(idx) })
		row.AppendChild(chip)
	}
	return row
}

// buildChip 构建单个附件 chip。
func buildChip(doc *dom.Document, path string, onRemove func()) *dom.Element {
	iconName := "file-text"
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		iconName = "folder"
	}
	chip := doc.CreateElement("div")
	chip.SetAttribute("style",
		"display:inline-flex;flex-direction:row;align-items:center;background-color:"+ui.PanelHeader+";border-radius:4px;padding:2px 6px 2px 4px;gap:3px;")

	iconEl := createIcon(doc, iconName, 11)
	iconEl.SetAttribute("style", iconEl.GetAttribute("style")+";color:"+ui.TextMute+";")
	chip.AppendChild(iconEl)

	label := createText(doc, filepath.Base(path), "color:"+ui.Text+";font-size:11px;")
	chip.AppendChild(label)

	closeBtn := createIcon(doc, "x", 10)
	closeBtn.SetAttribute("style", closeBtn.GetAttribute("style")+";cursor:pointer;color:"+ui.TextMute+";")
	closeBtn.SetAttribute("hover-style", "color:"+ui.Text+";")
	addClick(closeBtn, onRemove)
	chip.AppendChild(closeBtn)

	return chip
}

// buildRefChip 构建单个引用 chip。
func buildRefChip(doc *dom.Document, label, ref string, onRemove func()) *dom.Element {
	iconName := "message-square"
	if strings.HasPrefix(ref, "参考文件：") {
		iconName = "file-text"
	} else if strings.HasPrefix(ref, "参考目录：") {
		iconName = "folder"
	} else if strings.HasPrefix(ref, "```") {
		iconName = "braces"
	}
	chip := doc.CreateElement("div")
	chip.SetAttribute("style",
		"display:inline-flex;flex-direction:row;align-items:center;background-color:"+ui.PanelHeader+";border-radius:4px;padding:2px 6px 2px 4px;gap:3px;")

	iconEl := createIcon(doc, iconName, 11)
	iconEl.SetAttribute("style", iconEl.GetAttribute("style")+";color:"+ui.TextMute+";")
	chip.AppendChild(iconEl)

	labelEl := createText(doc, label, "color:"+ui.Text+";font-size:11px;")
	chip.AppendChild(labelEl)

	closeBtn := createIcon(doc, "x", 10)
	closeBtn.SetAttribute("style", closeBtn.GetAttribute("style")+";cursor:pointer;color:"+ui.TextMute+";")
	closeBtn.SetAttribute("hover-style", "color:"+ui.Text+";")
	addClick(closeBtn, onRemove)
	chip.AppendChild(closeBtn)

	return chip
}

// refreshSidebar 更新侧栏显示/隐藏和内容，同时控制分隔条显隐和两栏顺序。
func (s *ChatState) refreshSidebar() {
	if s.sidebarEl == nil {
		return
	}
	if s.ShowThreads {
		// 侧栏显示（固定 300px，右边框分隔，始终在右）
		s.sidebarEl.SetAttribute("style",
			"display:flex;flex-direction:column;width:300px;flex-shrink:0;border-left:1px solid "+ui.Border+";background:"+ui.SideBg+";overflow:hidden;")
		s.sidebarEl.ClearChildren()
		content := s.sidebarContent()
		s.sidebarEl.AppendChild(content)
	} else {
		s.sidebarEl.SetAttribute("style", "display:none;")
	}
}

// ─── 对话侧栏 ──────────────────────────────────────────────────

// sidebarContent 侧栏内容（300px）。
func (s *ChatState) sidebarContent() *dom.Element {
	doc := s.doc
	threads := s.Store.Threads

	// Token 统计
	ctxUsed, ctxMax, cats, tokenStats := s.projectTokenStats()

	sidebar := doc.CreateElement("div")
	sidebar.SetAttribute("style",
		"width:100%;display:flex;flex-direction:column;background-color:"+ui.SideBg+";height:100%;overflow:hidden;")

	// 头部
	head := doc.CreateElement("div")
	head.SetAttribute("style",
		"display:flex;flex-direction:row;align-items:center;padding:8px 10px 8px 8px;border-bottom:1px solid "+ui.Border+";")
	titleEl := createText(doc, "对话", "color:"+ui.TextMute+";font-size:12px;flex:1;")
	head.AppendChild(titleEl)

	// 新建会话按钮
	exportBtn := iconButton(doc, "download", 13, s.ExportActive)
	head.AppendChild(exportBtn)

	// 新建会话按钮
	newBtn := iconButton(doc, "plus", 13, func() {
		s.Store.NewThread()
		s.saveHistory()
		s.refreshMessageList()
		s.refreshSidebar()
		markDirty()
	})
	head.AppendChild(newBtn)
	sidebar.AppendChild(head)

	// 对话列表
	listArea := doc.CreateElement("div")
	listArea.SetAttribute("style", "flex:1;overflow-y:auto;padding:2px 0;")
	for _, t := range threads {
		item := s.buildThreadItem(t)
		listArea.AppendChild(item)
	}
	sidebar.AppendChild(listArea)

	// 分割线
	divider := doc.CreateElement("div")
	divider.SetAttribute("style", "height:3px;background-color:"+ui.Border+";flex-shrink:0;")
	sidebar.AppendChild(divider)

	// 底部：环图 + Token 统计
	bottomArea := doc.CreateElement("div")
	bottomArea.SetAttribute("style", "flex-shrink:0;display:flex;flex-direction:column;padding:8px;overflow-y:auto;")

	// 环图卡片
	donutCard := doc.CreateElement("div")
	donutCard.SetAttribute("style",
		"display:flex;flex-direction:column;background-color:"+ui.ChatBg+";border:1px solid "+ui.Border+";border-radius:6px;padding:4px;")
	donutCard.AppendChild(s.donutChartSection(ctxUsed, ctxMax, cats))
	bottomArea.AppendChild(donutCard)

	bottomArea.AppendChild(gapH(doc, 8))

	// Token 统计卡片
	statsCard := doc.CreateElement("div")
	statsCard.SetAttribute("style",
		"display:flex;flex-direction:column;background-color:"+ui.ChatBg+";border:1px solid "+ui.Border+";border-radius:6px;padding:4px;")
	statsCard.AppendChild(s.tokenStatsSection(tokenStats))
	bottomArea.AppendChild(statsCard)

	sidebar.AppendChild(bottomArea)

	return sidebar
}

// buildThreadItem 渲染单个会话项。
func (s *ChatState) buildThreadItem(t *state.Thread) *dom.Element {
	doc := s.doc
	active := t.ID == s.Store.ActiveID
	txtClr := ui.TextMute
	bgClr := "transparent"
	hoverClr := ui.HoverBg
	if active {
		txtClr = ui.Text
		bgClr = ui.ActiveBg
		hoverClr = ui.ActiveBg
	}
	dotClr := ui.Success
	if s.threadRunning(t) {
		dotClr = ui.Warning
	}

	item := doc.CreateElement("div")
	item.SetAttribute("style",
		"display:flex;flex-direction:row;align-items:center;height:36px;padding:0 6px 0 0;cursor:pointer;background-color:"+bgClr+";")
	item.SetAttribute("hover-style", "background-color:"+hoverClr+";")
	// 直接注册事件监听器，切换对话
	clickT := t
	ui.Ctx.App.AddEventListener(item, event.Click, func(e event.Event) bool {
		e.StopPropagation()
		s.Store.Switch(clickT.ID)
		s.refreshMessageList()
		s.refreshSidebar()
		markDirty()
		return true
	})

	// 左侧强调条（当前会话 3px 强调）
	var bar *dom.Element
	if active {
		bar = createDiv(doc, "width:3px;height:18px;background-color:"+ui.Accent+";border-radius:1.5px;flex-shrink:0;")
	} else {
		bar = createDiv(doc, "width:3px;flex-shrink:0;")
	}
	item.AppendChild(bar)
	item.AppendChild(gapW(doc, 7))
	item.AppendChild(statusDot(doc, dotClr))
	item.AppendChild(gapW(doc, 8))

	titleEl := createText(doc, t.Title, "color:"+txtClr+";font-size:12px;flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;")
	item.AppendChild(titleEl)

	// 关闭按钮
	closeBtn := createIcon(doc, "x", 11)
	closeBtn.SetAttribute("style", closeBtn.GetAttribute("style")+";cursor:pointer;color:"+ui.TextMute+";margin-left:4px;")
	closeBtn.SetAttribute("hover-style", "color:"+ui.Text+";")
	tt := t
	// 直接注册事件监听器，调用 StopPropagation() 防止冒泡到父级 item
	ui.Ctx.App.AddEventListener(closeBtn, event.Click, func(e event.Event) bool {
		e.StopPropagation()
		s.Store.Delete(tt.ID)
		s.saveHistory()
		s.refreshMessageList()
		s.refreshSidebar()
		markDirty()
		return true
	})
	item.AppendChild(closeBtn)

	return item
}

// threadRunning 该会话是否正被 Agent 引擎运行。
func (s *ChatState) threadRunning(t *state.Thread) bool {
	return s.Bridge != nil && s.Bridge.RunningThread() == t
}

// ─── 环形图 + Token 统计 ──────────────────────────────────────

// projectTokenStats 计算整个项目的 Token 统计。
func (s *ChatState) projectTokenStats() (contextUsed, contextMax int, cats []donutCat, tokenStats struct{ Total, CacheHit, CacheMiss, Output int }) {
	contextMax = 1048576 // 1M
	totalPrompt := 0
	totalCompletion := 0
	toolTokens := 0
	skillsTokens := 0
	mcpTokens := 0
	promptTokens := 0
	otherTokens := 0
	cacheHitTotal := 0
	cacheMissTotal := 0

	for _, t := range s.Store.Threads {
		usage := t.CalculateTokenUsage()
		totalPrompt += usage.PromptTokens
		totalCompletion += usage.CompletionTokens

		cacheHitTotal += t.TokenUsage.PromptCacheHitTokens
		cacheMissTotal += t.TokenUsage.PromptCacheMissTokens

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

	if totalPrompt+totalCompletion == 0 {
		cats = []donutCat{
			{Label: "工具", Value: 0, Color: ui.Success},
			{Label: "Skills", Value: 0, Color: ui.Info},
			{Label: "MCP", Value: 0, Color: ui.Accent},
			{Label: "提示词", Value: 0, Color: ui.Warning},
			{Label: "其他", Value: 0, Color: ui.TextMute},
		}
		return
	}

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

	cats = []donutCat{
		{Label: "工具", Value: toolTokens, Color: ui.Success},
		{Label: "Skills", Value: skillsTokens, Color: ui.Info},
		{Label: "MCP", Value: mcpTokens, Color: ui.Accent},
		{Label: "提示词", Value: promptTokens, Color: ui.Warning},
		{Label: "其他", Value: otherTokens, Color: ui.TextMute},
	}

	total := totalPrompt + totalCompletion
	tokenStats.Total = total
	tokenStats.Output = totalCompletion
	if cacheHitTotal > 0 || cacheMissTotal > 0 {
		tokenStats.CacheHit = cacheHitTotal
		tokenStats.CacheMiss = cacheMissTotal
	} else {
		tokenStats.CacheHit = int(float64(totalPrompt) * 0.6)
		tokenStats.CacheMiss = totalPrompt - tokenStats.CacheHit
		if tokenStats.CacheMiss < 0 {
			tokenStats.CacheMiss = 0
		}
	}
	return
}

// donutChartSection 上下文进度环 + 分类明细。
func (s *ChatState) donutChartSection(contextUsed, contextMax int, cats []donutCat) *dom.Element {
	doc := s.doc
	pct := 0.0
	if contextMax > 0 {
		pct = float64(contextUsed) * 100.0 / float64(contextMax)
	}
	centerText := fmt.Sprintf("%.0f%%", pct)

	ringSize := 90.0

	// 进度环颜色
	ringColor := ui.Success
	if pct > 70 {
		ringColor = ui.Warning
	}
	if pct > 90 {
		ringColor = ui.Error
	}

	// 用 CSS conic-gradient 画环形
	ring := doc.CreateElement("div")
	ring.SetAttribute("style",
		fmt.Sprintf("width:%fpx;height:%fpx;border-radius:50%%;position:relative;flex-shrink:0;"+
			"background:conic-gradient(%s 0%% %f%%, %s %f%% 100%%);",
			ringSize, ringSize, ringColor, pct, ui.PanelHeader, pct))

	// 中心镂空
	hole := doc.CreateElement("div")
	hole.SetAttribute("style",
		fmt.Sprintf("position:absolute;top:50%%;left:50%%;transform:translate(-50%%,-50%%);"+
			"width:%fpx;height:%fpx;border-radius:50%%;background-color:%s;"+
			"display:flex;align-items:center;justify-content:center;",
			ringSize*0.55, ringSize*0.55, ui.SideBg))
	center := createText(doc, centerText, "color:"+ui.Text+";font-size:13px;font-weight:bold;")
	hole.AppendChild(center)
	ring.AppendChild(hole)

	// 分类明细
	ctxBase := float64(contextUsed)
	if ctxBase <= 0 {
		ctxBase = 1
	}

	detailRows := doc.CreateElement("div")
	detailRows.SetAttribute("style", "display:flex;flex-direction:column;width:240px;margin-top:8px;")

	for i := 0; i < len(cats); i += 2 {
		row := doc.CreateElement("div")
		row.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;")

		for j := i; j < i+2 && j < len(cats); j++ {
			cc := cats[j]
			pct2 := float64(cc.Value) * 100.0 / ctxBase
			item := doc.CreateElement("div")
			item.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;padding:1px 2px;flex:1;")

			dot := createDiv(doc, fmt.Sprintf("width:8px;height:8px;border-radius:50%%;background-color:%s;flex-shrink:0;", cc.Color))
			item.AppendChild(dot)
			item.AppendChild(gapW(doc, 3))

			label := createText(doc, cc.Label, "color:"+ui.TextMute+";font-size:11px;flex:1;")
			item.AppendChild(label)

			val := createText(doc, fmt.Sprintf("%.0f%%", pct2), "color:"+ui.Text+";font-size:11px;")
			item.AppendChild(val)

			row.AppendChild(item)
		}
		detailRows.AppendChild(row)
	}

	container := createDiv(doc, "flex-direction:column;align-items:center;",
		ring,
		detailRows,
	)
	return container
}

// tokenStatsSection Token 统计四行。
func (s *ChatState) tokenStatsSection(ts struct{ Total, CacheHit, CacheMiss, Output int }) *dom.Element {
	doc := s.doc
	row := func(label string, value string, valColor string) *dom.Element {
		return createDiv(doc,
			"flex-direction:row;align-items:center;padding:0 0 6px 0;",
			createText(doc, label, "color:"+ui.TextMute+";font-size:13px;flex:1;"),
			createText(doc, value, "color:"+valColor+";font-size:13px;"),
		)
	}

	return createDiv(doc,
		"flex-direction:column;padding:6px;",
		row("总", shortToken(ts.Total), ui.Text),
		row("缓存命中", shortToken(ts.CacheHit), ui.Success),
		row("未命中", shortToken(ts.CacheMiss), ui.Warning),
		row("输出", shortToken(ts.Output), ui.Accent),
	)
}

// ─── 任务进度面板 ─────────────────────────────────────────────

// taskProgressPanel 构建任务进度面板（计划清单）。
func (s *ChatState) taskProgressPanel() *dom.Element {
	if len(s.Plan) == 0 {
		panel := s.doc.CreateElement("div")
		panel.SetAttribute("style", "display:none;")
		return panel
	}
	doc := s.doc
	panel := doc.CreateElement("div")
	panel.SetAttribute("style",
		"display:flex;flex-direction:column;padding:6px 12px;border-top:1px solid "+ui.Border+";background-color:"+ui.ChatBg+";")

	title := createText(doc, "任务计划", "color:"+ui.TextDim+";font-size:11px;font-weight:bold;margin-bottom:4px;")
	panel.AppendChild(title)

	for _, step := range s.Plan {
		row := doc.CreateElement("div")
		row.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;padding:2px 0;")

		var dotColor string
		switch step.Status {
		case "done":
			dotColor = ui.Success
		case "in_progress":
			dotColor = ui.Warning
		default:
			dotColor = ui.TextMute
		}
		dot := statusDot(doc, dotColor)
		row.AppendChild(dot)
		row.AppendChild(gapW(doc, 6))

		stepText := createText(doc, step.Step, "color:"+ui.Text+";font-size:12px;")
		row.AppendChild(stepText)
		panel.AppendChild(row)
	}
	return panel
}

// ─── 包级函数 ──────────────────────────────────────────────────

// ToggleSearch 切换搜索栏显隐。
func ToggleSearch() {
	if TheState != nil {
		TheState.ToggleSearch()
	}
}

// IsRunning 返回 Agent 是否正在运行。
func IsRunning() bool {
	return TheState != nil && TheState.Bridge != nil && TheState.Bridge.IsRunning()
}

// StopAgent 停止正在运行的 Agent。
func StopAgent() {
	if TheState != nil && TheState.Bridge != nil {
		TheState.Bridge.Stop()
	}
}

// resolveAskUI 把问答卡的回答路由到 bridge。
func resolveAskUI(answer string) {
	if TheState != nil && TheState.Bridge != nil {
		TheState.Bridge.ResolveAsk(answer)
	}
}

// ─── ChatState 方法 ───────────────────────────────────────────

// loadTestData 性能测试：填充 1000 条测试消息。
func (s *ChatState) loadTestData() {
	t := s.Store.Active()
	if t == nil {
		return
	}
	const N = 500
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
			Collapsed: i < N-3,
		})
	}
	s.SendSeq++
	s.refreshMessageList()
	markDirty()
}

// ToggleSearch Ctrl+F 开关搜索栏。
func (s *ChatState) ToggleSearch() {
	s.ShowSearch = !s.ShowSearch
	if !s.ShowSearch {
		s.SearchQuery = ""
		s.searchSeq++
	}
	// 更新搜索栏显隐
	if s.searchBarEl != nil {
		if s.ShowSearch {
			s.searchBarEl.SetAttribute("style",
				"display:flex;height:34px;padding:0 10px;background-color:"+ui.PanelHeader+";border-bottom:1px solid "+ui.Border+";flex-direction:row;align-items:center;")
		} else {
			s.searchBarEl.SetAttribute("style", "display:none;")
		}
	}
	if !s.ShowSearch {
		s.refreshMessageList()
	}
	markDirty()
}

// SearchMatchCount 当前会话命中搜索词的消息数。
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

// agentBusy 返回 Agent 是否忙碌。
func (s *ChatState) agentBusy() bool { return s.Bridge != nil && s.Bridge.IsRunning() }

// persistAgentToggles 持久化审核/自主开关。
func (s *ChatState) persistAgentToggles() {
	core.Settings.RequireApproval = !s.AutoReview
	core.Settings.Autonomous = s.Autonomous
	core.Save()
}

// saveHistory 保存对话历史。
func (s *ChatState) saveHistory() {
	s.Store.Save(core.Root())
}

// ─── 发送 ──────────────────────────────────────────────────────

// SetInputText 设置聊天输入框文本（供外部使用，如右键"添加到对话"）。
func (s *ChatState) SetInputText(text string) {
	s.Store.Draft = text
	if s.textAreaComp != nil {
		s.textAreaComp.SetText(text)
	}
}

// Send 发送输入框草稿消息。
func (s *ChatState) Send(text string) {
	if s.Bridge != nil && s.Bridge.IsRunning() {
		return
	}
	draft := text
	if draft == "" {
		draft = s.Store.Draft
	}
	// 保存原始输入到历史（不含附件/长文本后缀）
	s.saveToMsgHistory(draft)
	atts := s.Attachments
	display := draft

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
	if !s.Store.Send(display) {
		return
	}
	if longTextFile != "" {
		t := s.Store.Active()
		if t != nil && len(t.Messages) > 0 {
			t.Messages[len(t.Messages)-1].LongTextFile = longTextFile
		}
	}
	s.SendSeq++
	s.Plan = nil
	s.Attachments = nil
	s.historyIdx = -1
	s.historyDraft = ""
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
	s.Bridge.Start(draft + refsText + attachmentContext(atts))
	s.saveHistory()
	s.refreshMessageList()
	// 清空输入框
	if s.textAreaComp != nil {
		s.textAreaComp.SetText("")
	}
	markDirty()
}

// sendTask 以指定任务发起一轮。
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
	s.refreshMessageList()
	markDirty()
}

// ExploreKnowledgeBase 探索项目知识库。
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

// ─── 消息操作 ──────────────────────────────────────────────────

// DeleteMessage 删除某条消息。
func (s *ChatState) DeleteMessage(t *state.Thread, i int) {
	if i < 0 || i >= len(t.Messages) {
		return
	}
	t.Messages = append(t.Messages[:i], t.Messages[i+1:]...)
	s.HoveredMsg = -1
	s.saveHistory()
	s.refreshMessageList()
	markDirty()
}

// Regenerate 重新生成末条助手回复。
func (s *ChatState) Regenerate(t *state.Thread, i int) {
	if s.agentBusy() || i <= 0 || i != len(t.Messages)-1 {
		return
	}
	user := t.Messages[i-1]
	if user.Role != state.User {
		return
	}
	t.Messages = t.Messages[:i]
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

	userText := user.Text
	if user.LongTextFile != "" {
		if data, err := os.ReadFile(user.LongTextFile); err == nil {
			userText = string(data)
		}
	}
	s.saveHistory()
	s.Bridge.Start(userText)
	s.refreshMessageList()
	markDirty()
}

// ExportActive 把当前会话导出为 Markdown。
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
	s.refreshMessageList()
	markDirty()
}

// ─── 输入区动态重建 ──────────────────────────────────────────

// 重建输入区的方法现在在 Send 等操作后调用 refreshInputArea。

// ─── 实用工具 ──────────────────────────────────────────────────

// estimateMessageHeight 估算单条消息的初始渲染高度。
func estimateMessageHeight(m state.Message) float64 {
	if m.Collapsed && !m.Streaming {
		return 56
	}
	h := 56.0

	if len(m.Timeline) > 0 {
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

// msgMatches 消息是否命中搜索词。
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

// shortToken 格式化 token 数为可读短字符串。
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

// ArgPreview 返回工具参数的简短预览。
func ArgPreview(args string) string {
	if len([]rune(args)) > 80 {
		return string([]rune(args)[:80]) + "…"
	}
	return args
}

// refLabel 从引用文本中提取简短 chip 标签。
func refLabel(text string) string {
	if strings.HasPrefix(text, "参考文件：") {
		return strings.TrimPrefix(text, "参考文件：")
	}
	if strings.HasPrefix(text, "参考目录：") {
		return strings.TrimPrefix(text, "参考目录：")
	}
	if strings.HasPrefix(text, "```\n") {
		lines := strings.SplitN(text, "\n", 3)
		if len(lines) > 1 && len(lines[0]) > 3 {
			return lines[0][3:] + " 代码片段"
		}
		return "代码片段"
	}
	if len(text) > 30 {
		return text[:30] + "…"
	}
	return text
}

// attachmentContext 把附件内容拼成给 agent 的上下文段。
func attachmentContext(atts []string) string {
	if len(atts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n# 用户附件")
	totalLimit := 50000
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
			dirSize := 0
			filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
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

// attachmentNames 附件文件名（逗号分隔）。
func attachmentNames(atts []string) string {
	names := make([]string, len(atts))
	for i, p := range atts {
		names[i] = filepath.Base(p)
	}
	return strings.Join(names, ", ")
}

// tintCSS 把 CSS 六位十六进制颜色 `#RRGGBB` 按 alpha 透明度转为半透明 rgba（用于按钮开启态的低透明填充）。
// 示例：tintCSS("#4ec9b0", 30) → "rgba(78,201,176,0.1176)"
func tintCSS(c string, alpha uint8) string {
	if len(c) < 7 || c[0] != '#' {
		fmt.Fprintf(os.Stderr, "[tintCSS] INVALID input: %q (len=%d, prefix=%c)\n", c, len(c), c[0])
		return c
	}
	// 解析 #RRGGBB
	r := hexPair(c[1:3])
	g := hexPair(c[3:5])
	b := hexPair(c[5:7])
	a := float64(alpha) / 255.0
	res := fmt.Sprintf("rgba(%d,%d,%d,%.4f)", r, g, b, a)
	fmt.Fprintf(os.Stderr, "[tintCSS] %q alpha=%d → %s\n", c, alpha, res)
	return res
}

// hexPair 把两位十六进制字符串解析为数值（如 "4e" → 78）。
func hexPair(s string) int {
	if len(s) < 2 {
		return 0
	}
	return hexVal(s[0])*16 + hexVal(s[1])
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	default:
		return 0
	}
}

// firstLine 返回文本的首行（去空，截断到 60 字符）。
func firstLine(text string) string {
	idx := strings.Index(text, "\n")
	if idx > 0 {
		text = text[:idx]
	}
	text = strings.TrimSpace(text)
	if len([]rune(text)) > 60 {
		return string([]rune(text)[:60]) + "…"
	}
	return text
}

// simplifyThinking 简化思考内容：只显示前 6 行或前 300 字符。
func simplifyThinking(text string) string {
	lines := strings.SplitN(text, "\n", 8)
	if len(lines) > 6 {
		lines = lines[:6]
		lines = append(lines, "…（思考内容已省略）")
	}
	simplified := strings.Join(lines, "\n")
	if len([]rune(simplified)) > 300 {
		return string([]rune(simplified)[:300]) + "\n…（思考内容已省略）"
	}
	return simplified
}

// planCard 保留旧接口别名。
func (s *ChatState) planCard() *dom.Element { return s.taskProgressPanel() }
