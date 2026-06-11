// Package ui 是 companion 的设计系统层 —— 类 web 的「设计令牌 + 组件库」，也是**唯一**的调色板。
//
// 设计意图（对应维护者要求）：
//   - 单一真相源：内容色 + VS Code 外壳色 + 语义色全在此一套令牌，不再有 gh*/c* 多套并存。
//   - 配置化换肤：Apply 一次换全部令牌；令牌是稳定指针，就地改 *Bg 等，持指针的样式/类下次绘制即见新色
//     （沿用 goui ColorRef「改色不重建」的性能设计）。
//   - 类 web 写法：组件 helper（Row/Col/Card/Button/Field/Toggle…）读令牌构建，调用方只描述结构。
//   - 模块分层：companion 各面板依赖本层；本层只依赖 internal/widget + types，绝不反向依赖 main。
//
//go:build windows

package ui

import "github.com/user/goui/internal/types"

// Tokens 是 Apply 的输入 DTO：一套主题的全部语义色（值）。companion theme.go 据 appTheme 构造它。
type Tokens struct {
	// ── 内容面（GitHub 风：对话/设置/搜索/Git 等多数 UI）──
	Bg           types.Color // 主背景
	BgSubtle     types.Color // 次级面（卡片/侧栏/输入框底）
	BgMuted      types.Color // 三级面（次级按钮/标签/agent 卡）
	BgHover      types.Color // 悬停面
	BgActive     types.Color // 激活/选中面
	Border       types.Color // 边框/分隔线
	Accent       types.Color // 强调（链接/焦点环/选中文字）
	AccentStrong types.Color // 强调实心（主按钮底）
	Blue         types.Color // 自主模式蓝（刻意区别于 Accent）
	OnAccent     types.Color // 强调底之上的文字（白）
	Text         types.Color // 主文字
	TextSubtle   types.Color // 次文字
	TextMuted    types.Color // 弱文字/占位
	Success      types.Color
	Warning      types.Color
	Danger       types.Color
	UserBg       types.Color // 用户卡黄底
	UserBorder   types.Color // 用户卡边

	// ── VS Code 外壳面（标题栏/侧栏/编辑区/状态栏）──
	ShellTitle     types.Color
	ShellSide      types.Color
	ShellEditor    types.Color
	ShellStatus    types.Color // 强调（分隔条 hover/拖动高亮）
	ShellStatusBar types.Color
	ShellBorder    types.Color
	ShellText      types.Color
	ShellTextDim   types.Color

	// ── 文件树行 ──
	FtHover    types.Color // 行悬停
	FtSelected types.Color // 行选中

	Radius float64 // 默认圆角
}

// 稳定指针调色板：companion 各处直接用 ui.Bg 作 *Color、用 *ui.Text 取值；Apply 就地改这些指针的目标。
var (
	Bg, BgSubtle, BgMuted, BgHover, BgActive, Border = nc(), nc(), nc(), nc(), nc(), nc()
	Accent, AccentStrong, Blue, OnAccent, White      = nc(), nc(), nc(), nc(), nc()
	Fg, FgSubtle, FgMuted                            = nc(), nc(), nc() // 前景文字色（名避开 Text/Subtle/Muted 文字 helper）
	Success, Warning, Danger                         = nc(), nc(), nc()
	UserBg, UserBorder                               = nc(), nc()

	ShellTitle, ShellSide, ShellEditor       = nc(), nc(), nc()
	ShellStatus, ShellStatusBar, ShellBorder = nc(), nc(), nc()
	ShellText, ShellTextDim                  = nc(), nc()

	FtHover, FtSelected = nc(), nc()

	Radius = 6.0
)

// nc 新建一个零色指针（调色板槽位）。
func nc() *types.Color { return &types.Color{} }

func init() { Apply(darkTokens()) } // 启动即铺默认深色（applyTheme 之前用 ui 也安全）

// darkTokens 默认深色令牌（= companion 原 GitHub 深色 + VS Code 外壳灰，换肤回 dark 零差异）。
func darkTokens() Tokens {
	h := types.ColorFromHex
	rgb := types.ColorFromRGB
	return Tokens{
		Bg: h("#0d1117"), BgSubtle: h("#161b22"), BgMuted: h("#21262d"),
		BgHover: h("#30363d"), BgActive: rgb(28, 45, 74), Border: h("#30363d"),
		Accent: h("#58a6ff"), AccentStrong: h("#1f6feb"), Blue: rgb(59, 130, 246), OnAccent: rgb(255, 255, 255),
		Text: h("#e6edf3"), TextSubtle: h("#8b949e"), TextMuted: h("#6e7681"),
		Success: h("#3fb950"), Warning: h("#d29922"), Danger: h("#f85149"),
		UserBg: rgb(24, 23, 18), UserBorder: rgb(38, 35, 23),
		ShellTitle: rgb(60, 60, 61), ShellSide: rgb(37, 37, 38), ShellEditor: rgb(30, 30, 30),
		ShellStatus: rgb(0, 122, 204), ShellStatusBar: rgb(45, 45, 48), ShellBorder: rgb(45, 45, 45),
		ShellText: rgb(204, 204, 204), ShellTextDim: rgb(140, 140, 140),
		FtHover: rgb(42, 45, 46), FtSelected: rgb(9, 71, 113),
		Radius: 6,
	}
}

// Apply 切换当前调色板：就地改全部令牌指针 + 刷新样式类（换肤总入口；companion applyTheme 调用）。
func Apply(t Tokens) {
	if t.Radius == 0 {
		t.Radius = 6
	}
	if t.OnAccent.A == 0 {
		t.OnAccent = types.ColorFromRGB(255, 255, 255)
	}
	*Bg, *BgSubtle, *BgMuted = t.Bg, t.BgSubtle, t.BgMuted
	*BgHover, *BgActive, *Border = t.BgHover, t.BgActive, t.Border
	*Accent, *AccentStrong, *Blue, *OnAccent = t.Accent, t.AccentStrong, t.Blue, t.OnAccent
	*White = types.ColorFromRGB(255, 255, 255)
	*Fg, *FgSubtle, *FgMuted = t.Text, t.TextSubtle, t.TextMuted
	*Success, *Warning, *Danger = t.Success, t.Warning, t.Danger
	*UserBg, *UserBorder = t.UserBg, t.UserBorder
	*ShellTitle, *ShellSide, *ShellEditor = t.ShellTitle, t.ShellSide, t.ShellEditor
	*ShellStatus, *ShellStatusBar, *ShellBorder = t.ShellStatus, t.ShellStatusBar, t.ShellBorder
	*ShellText, *ShellTextDim = t.ShellText, t.ShellTextDim
	*FtHover, *FtSelected = t.FtHover, t.FtSelected
	Radius = t.Radius
	defineClasses() // 用新令牌重注册 CSS 样式类（类持稳定指针，其实改色已自动；保留以防类引用值快照）
}
