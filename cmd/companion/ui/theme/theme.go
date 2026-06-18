// 多主题渲染 —— 复刻参考源 styles/global.css 的 5 套主题核心变量（dark/light/high-contrast/
// solarized-light/dracula）。每套主题是一个 appTheme（核心色 + VS Code 外壳色），Apply 经
// uiTokens 映射成 ui 包的**单套**设计令牌（ui.Bg/ui.ShellSide/…，内容色与外壳色统一一处，不再有
// gh*/c* 多套），再灌 goui 组件主题（菜单/对话框/下拉/编辑器），运行时切换强制重建整树重读颜色。
//
// 注：参考的 solarized/dracula 另有大量 per-component CSS 覆盖（companion 无对应类），只复刻核心变量。
//
//go:build windows

package theme

import (
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// uiTokens 把一套 appTheme 映射成 ui 设计令牌（ui 包是唯一调色板）。内容 + VS Code 外壳 + 文件树行全覆盖。
func uiTokens(t appTheme) ui.Tokens {
	return ui.Tokens{
		Bg: t.bgPrimary, BgSubtle: t.bgSecondary, BgMuted: t.bgTertiary,
		BgHover: t.bgHover, BgActive: t.bgActive, Border: t.border,
		Accent: t.accent, AccentStrong: t.accentEmph, Blue: types.ColorFromRGB(59, 130, 246), Purple: types.ColorFromRGB(168, 130, 255), OnAccent: types.ColorFromRGB(255, 255, 255),
		Text: t.text, TextSubtle: t.textSec, TextMuted: t.textMuted,
		Success: t.success, Warning: t.warning, Danger: t.danger,
		UserBg: t.userBg, UserBorder: t.userBorder,
		ShellTitle: t.chTitle, ShellSide: t.chSide, ShellEditor: t.chEditor,
		ShellStatus: t.chStatus, ShellStatusBar: t.chStatusBar, ShellBorder: t.chBorder,
		ShellText: t.chText, ShellTextDim: t.chTextDim,
		FtHover: t.bgHover, FtSelected: thAlpha(t.accentEmph, 96),
		Radius: 6,
	}
}

// appTheme 一套主题的全部色板（companion 语义令牌 + VS Code 外壳令牌）。
type appTheme struct {
	dark bool // 暗色主题（决定 CodeEditor 用 Dark / Default(浅) 组件主题）

	bgPrimary, bgSecondary, bgTertiary, bgHover, bgActive, border types.Color // 背景/边框
	accent, accentEmph                                            types.Color // 强调
	text, textSec, textMuted                                      types.Color // 文字
	success, warning, danger                                      types.Color // 语义
	userBg, userBorder                                            types.Color // 用户卡（companion 用 warning 染色）

	chTitle, chSide, chEditor, chStatus, chStatusBar, chBorder types.Color // VS Code 外壳
	chText, chTextDim                                          types.Color
}

func thHex(s string) types.Color { return types.ColorFromHex(s) }

// gouiEditorTheme 据当前 appTheme 给 goui 编辑器配色：chrome（底色/行号/当前行/选区/缩略图）
// 与语法主色（关键字=强调、字符串=成功色、数字=警告色、注释/正文=对应令牌）全跟随主题；
// 类型/函数名沿用 dark/light 基底（区分度好），StructEditor（Go 结构化视图）整套跟随。
func gouiEditorTheme(t appTheme, base widget.Theme) widget.Theme {
	ce := base.CodeEditor
	ce.Background, ce.GutterBg = t.chEditor, t.chEditor
	ce.GutterText, ce.GutterActiveBg = t.textMuted, thAlpha(t.text, 10)
	ce.CurrentLineBg = thAlpha(t.text, 12)
	ce.Selection, ce.MinimapBg = thAlpha(t.accent, 64), t.bgSecondary
	ce.Keyword, ce.String, ce.Number = t.accent, t.success, t.warning
	ce.Comment, ce.Text = t.textMuted, t.text
	base.CodeEditor = ce
	base.StructEditor = widget.StructEditorTheme{
		Background: t.chEditor, CellBg: t.bgSecondary, LineColor: t.border,
		HeaderBg: t.bgTertiary, GutterBg: t.chEditor, GutterText: t.textMuted,
		TextColor: t.text, SelectionBg: thAlpha(t.accent, 64), HeaderText: t.textSec,
		FuncRowBg: thAlpha(t.accent, 28), MinimapBg: t.bgSecondary,
	}
	// 通用色 + 内置组件主题随 ui 令牌（让「配置驱动」的库组件 Button/Input/Slider/Switch/Select 等与
	// 手建 ui 组件同观感；否则它们仍是库默认 el 蓝/浅色）。
	base.PrimaryColor, base.TextColor, base.SecondaryText = t.accent, t.text, t.textSec
	base.BGColor, base.SurfaceColor, base.BorderColor, base.DividerColor = t.bgPrimary, t.bgSecondary, t.border, t.border
	base.ErrorColor, base.SuccessColor, base.WarningColor, base.InfoColor = t.danger, t.success, t.warning, t.textMuted
	base.TextRegular, base.PlaceholderColor = t.text, t.textMuted
	base.FillColor, base.BorderLight, base.BorderLighter = t.bgTertiary, t.border, t.border
	base.Button = widget.ButtonTheme{
		DefaultColor: t.bgTertiary, PrimaryColor: t.accentEmph, SuccessColor: t.success, DangerColor: t.danger,
		TextColor: t.text, HoverColor: t.bgHover, DisabledColor: t.textMuted, MinWidth: 64, MinHeight: 26, BorderRadius: 5,
	}
	base.Input = widget.InputTheme{
		TextColor: t.text, BGColor: t.bgPrimary, BorderColor: t.border, FocusBorderColor: t.accent,
		PlaceholderColor: t.textMuted, CursorColor: t.text, CursorWidth: 1.5, BorderRadius: 5,
	}
	base.Slider = widget.SliderTheme{
		ActiveColor: t.accentEmph, InactiveColor: t.bgTertiary, ThumbColor: t.accentEmph, LabelColor: t.text,
		ThumbRadius: 8, TrackHeight: 4,
	}
	base.Switch.ActiveColor, base.Switch.InactiveColor = t.accentEmph, t.bgTertiary
	base.Checkbox.ActiveColor, base.Checkbox.BorderColor, base.Checkbox.LabelColor = t.accentEmph, t.border, t.text
	base.Checkbox.InactiveBgColor = t.bgPrimary
	return base
}

// thAlpha 同色改透明度（用户卡/选中态等半透明令牌）。
func thAlpha(c types.Color, a uint8) types.Color { c.A = a; return c }

// mkTheme 从参考核心 CSS 变量构造主题（外壳映射到 bg 令牌、用户卡用 warning 低透染色）。
// 4 个非 dark 主题用它；dark 用下方显式字面量保持现有精确观感不变。
func mkTheme(dark bool, bgP, bgS, bgT, bgH, bd, acc, accE, txt, txtS, txtM, ok, warn, dang string, bgActiveAlpha uint8) appTheme {
	w, ae := thHex(warn), thHex(accE)
	return appTheme{
		dark:      dark,
		bgPrimary: thHex(bgP), bgSecondary: thHex(bgS), bgTertiary: thHex(bgT), bgHover: thHex(bgH),
		bgActive: thAlpha(ae, bgActiveAlpha), border: thHex(bd),
		accent: thHex(acc), accentEmph: ae,
		text: thHex(txt), textSec: thHex(txtS), textMuted: thHex(txtM),
		success: thHex(ok), warning: w, danger: thHex(dang),
		userBg: thAlpha(w, 28), userBorder: thAlpha(w, 64),
		// 外壳映射到核心令牌（整个 app 随主题统一）
		chTitle: thHex(bgS), chSide: thHex(bgS), chEditor: thHex(bgP), chStatus: thHex(acc),
		chStatusBar: thHex(bgS), chBorder: thHex(bd), chText: thHex(txt), chTextDim: thHex(txtM),
	}
}

// themeDark 暗色 —— 现有精确值（GitHub 深色内容色 + VS Code 外壳灰），切回 dark 零改动。
var themeDark = appTheme{
	dark:      true,
	bgPrimary: thHex("#0d1117"), bgSecondary: thHex("#161b22"), bgTertiary: thHex("#21262d"), bgHover: thHex("#30363d"),
	bgActive: types.ColorFromRGB(28, 45, 74), border: thHex("#30363d"),
	accent: thHex("#58a6ff"), accentEmph: thHex("#1f6feb"),
	text: thHex("#e6edf3"), textSec: thHex("#8b949e"), textMuted: thHex("#6e7681"),
	success: thHex("#3fb950"), warning: thHex("#d29922"), danger: thHex("#f85149"),
	userBg: types.ColorFromRGB(24, 23, 18), userBorder: types.ColorFromRGB(38, 35, 23),
	chTitle: types.ColorFromRGB(60, 60, 61), chSide: types.ColorFromRGB(37, 37, 38), chEditor: types.ColorFromRGB(30, 30, 30),
	chStatus: types.ColorFromRGB(0, 122, 204), chStatusBar: types.ColorFromRGB(45, 45, 48), chBorder: types.ColorFromRGB(45, 45, 45),
	chText: types.ColorFromRGB(204, 204, 204), chTextDim: types.ColorFromRGB(140, 140, 140),
}

// themes 五套主题（核心变量 1:1 照搬参考 global.css）。
var themes = map[string]appTheme{
	"dark":            themeDark,
	"light":           mkTheme(false, "#ffffff", "#f6f8fa", "#edeff2", "#d0d7de", "#d0d7de", "#0969da", "#0550ae", "#1f2328", "#4b5563", "#6b7280", "#1a7f37", "#9a6700", "#cf222e", 24),
	"high-contrast":   mkTheme(true, "#000000", "#0a0a0a", "#1a1a1a", "#2a2a2a", "#888888", "#79c0ff", "#58a6ff", "#ffffff", "#f0f0f0", "#e0e0e0", "#56d364", "#e3b341", "#ff7b72", 85),
	"solarized-light": mkTheme(false, "#fdf6e3", "#eee8d5", "#e4ddc7", "#d5ccb3", "#cdc4a8", "#268bd2", "#1a6ea0", "#073642", "#586e75", "#5d737a", "#859900", "#b58900", "#dc322f", 34),
	"dracula":         mkTheme(true, "#282a36", "#1e1f29", "#343746", "#44475a", "#44475a", "#bd93f9", "#9876e0", "#f8f8f2", "#c0c0d0", "#9090a0", "#50fa7b", "#f1fa8c", "#ff5555", 68),
}

// Apply 把指定主题灌进 ui 单套设计令牌 + goui 组件主题。未知名回退 dark。
// 令牌全在 ui 包（稳定指针，就地改色）——companion 各处读 ui.Bg/*ui.Fg/ui.ShellSide…，不再有 gh*/c* 多套。
func Apply(name string) {
	t, ok := themes[name]
	if !ok {
		t = themeDark
	}
	// 唯一调色板：把这套主题灌进 ui 令牌（就地改全部指针 + 重注册样式类，整 app 随之换肤）。
	ui.SetFontHook(applyUIFont) // 界面字体（族/粗斜）钩子，幂等设置
	ui.Apply(uiTokens(t))
	// goui 组件主题随之（菜单/对话框/下拉），否则浮层仍是旧色
	widget.SetMenuTheme(t.bgTertiary, t.text, t.accentEmph, t.border, t.textMuted)
	widget.SetDialogTheme(t.bgSecondary, t.text, t.textMuted)
	widget.SetSelectTheme(t.bgPrimary, t.text, t.border, t.bgTertiary, t.textMuted)
	// goui CodeEditor/StructEditor 配色随主题：之前只灌 Dark/Default 通用主题，
	// 故 dracula/solarized/high-contrast 的编辑器底色不匹配、各暗色主题编辑器长一样。
	base := widget.DefaultTheme()
	if t.dark {
		base = widget.DarkTheme()
	}
	widget.SetTheme(gouiEditorTheme(t, base))
	if widget.OnNeedsLayout != nil { // 运行时切换 → 强制重建整树重读颜色（启动时为 nil，首帧自然读）
		// 必须 BumpRebuild：否则配置未变的组件（菜单/标签/标题栏项目名等）命中 Build 缓存、
		// 不重读颜色令牌，要等下次自身 SetState 才变色（用户：很多组件点一下才换色）。
		widget.BumpRebuild()
		widget.OnNeedsLayout()
	}
}
