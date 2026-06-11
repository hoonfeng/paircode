// Package state 持有 companion 的应用状态（Go 结构体 + goui SetState，见 ../AGENTS.md §5）。
// 不照搬 Zustand：状态结构体是布局的唯一真相来源，变更后由持有它的 State 调 SetState 触发重建。
package state

// Panels 三个停靠区的可见性、尺寸、各区所放面板——窗壳布局的唯一真相来源。
// 三个可停靠面板组 files(文件/搜索/Git)/chat/terminal 在三区间排列（一区一组，移动=两区互换）；
// 编辑器恒居中。
type Panels struct {
	Left, Right, Bottom              bool    // 各停靠区是否展开
	LeftW, RightW, BottomH           float64 // 各停靠区尺寸（像素）
	LeftPanel, RightPanel, BotPanel string  // 各区所放面板组："files"/"chat"/"terminal"
}

// DefaultPanels 默认三区全开，IDE 常用尺寸 + 默认排布（左 files / 右 chat / 底 terminal）。
func DefaultPanels() *Panels {
	return &Panels{
		Left: true, Right: true, Bottom: true,
		LeftW: 260, RightW: 400, BottomH: 200,
		LeftPanel: "files", RightPanel: "chat", BotPanel: "terminal",
	}
}

// IdeWelcomePanels IDE 欢迎页面板布局：所有停靠区隐藏，仅中列展示欢迎页。
// 在未打开工作区/项目时使用，让窗口干净地呈现欢迎入口。
func IdeWelcomePanels() *Panels {
	return &Panels{
		Left: false, Right: false, Bottom: false,
		LeftW: 260, RightW: 400, BottomH: 200,
		LeftPanel: "files", RightPanel: "chat", BotPanel: "terminal",
	}
}

// PanelIn 返回某区当前所放的面板组 id。
func (p *Panels) PanelIn(z Zone) string {
	switch z {
	case ZoneLeft:
		return p.LeftPanel
	case ZoneRight:
		return p.RightPanel
	default:
		return p.BotPanel
	}
}

func (p *Panels) setPanel(z Zone, panel string) {
	switch z {
	case ZoneLeft:
		p.LeftPanel = panel
	case ZoneRight:
		p.RightPanel = panel
	case ZoneBottom:
		p.BotPanel = panel
	}
}

// ZoneOf 返回某面板组当前所在的区。
func (p *Panels) ZoneOf(panel string) Zone {
	switch panel {
	case p.RightPanel:
		return ZoneRight
	case p.BotPanel:
		return ZoneBottom
	default:
		return ZoneLeft
	}
}

// Move 把面板组移到目标区：与目标区原面板**互换**（保持一区一组），并确保目标区可见。
func (p *Panels) Move(panel string, to Zone) {
	from := p.ZoneOf(panel)
	if from == to {
		return
	}
	other := p.PanelIn(to)
	p.setPanel(to, panel)
	p.setPanel(from, other)
	p.show(to)
}

func (p *Panels) show(z Zone) {
	switch z {
	case ZoneLeft:
		p.Left = true
	case ZoneRight:
		p.Right = true
	case ZoneBottom:
		p.Bottom = true
	}
}

// 尺寸约束（拖动 resize 时夹紧，避免拖没/拖爆）。
const (
	MinSideW = 160
	MaxSideW = 1000 // 侧栏(含对话)可拖到的最大宽（编辑区只留少量即可，不必大块预留）
	MinChatW = 360  // 对话主区最小宽（仅内容，不含对话列表；列表展开时整栏在此之上再加列表宽）
	ThreadW  = 190  // 对话列表展开占的额外栏宽（侧栏 180 + 分隔）
	MinBotH  = 120
	MaxBotH  = 600
)

// Toggle 翻转某停靠区的展开状态。
func (p *Panels) Toggle(z Zone) {
	switch z {
	case ZoneLeft:
		p.Left = !p.Left
	case ZoneRight:
		p.Right = !p.Right
	case ZoneBottom:
		p.Bottom = !p.Bottom
	}
}

// Zone 标识一个停靠区。
type Zone int

const (
	ZoneLeft Zone = iota
	ZoneRight
	ZoneBottom
)

// Clamp 把 v 夹到 [lo, hi]。
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
