// Package ui 提供 GWui 版 companion 设计系统：CSS 主题 + 颜色常量 + 共享样式。
// 所有面板依赖本层；本层只依赖 GWui（dom/css），绝不反向依赖 main。
package ui

import "github.com/hoonfeng/gwui/dom"

// VS Code Dark+ 风格配色（IDE 深色）。
const (
	// 外壳
	TitleBarBg   = "#3c3c3d"
	SideBg       = "#252526"
	EditorBg     = "#1e1e1e"
	StatusBarBg  = "#2d2d30"
	Border       = "#2d2d2d"
	Accent       = "#0e639c"
	AccentHover  = "#1177bb"
	DropHintBg   = "rgba(88, 166, 255, 0.22)"

	// 文字
	Text     = "#cccccc"
	TextDim  = "#8c8c8c"
	TextMute = "#6e6e6e"

	// 面板
	PanelBg     = "#252526"
	PanelHeader = "#2d2d2d"
	InputBg     = "#3c3c3c"
	HoverBg     = "#2a2d2e"
	ActiveBg    = "#094771"
	SelectBg    = "#094771"

	// 语义
	Success = "#4ec9b0"
	Error   = "#f48771"
	Warning = "#dcdcaa"
	Info    = "#569cd6"

	// 对话
	ChatBg       = "#1e1e1e"
	UserBubble   = "#264f78"
	AssistantBg  = "#252526"
	ThinkingBg   = "#1a1a1a"
	ToolActivity = "#2d2d2d"

	// 编辑器
	LineNumBg  = "#1e1e1e"
	LineNumFg  = "#858585"
	Selection  = "#264f78"
	Highlight  = "#3d3d3d"

	// 尺寸
	TitleBarH = 36
	StatusH   = 30
	ToggleW   = 40
	WinBtnW   = 46
	DividerW  = 4
	MinSideW  = 160
	MaxSideW  = 1000 // 对齐 state/panels.go（侧栏可拖到编辑区只剩少量空间）
	MinChatW  = 300
	MinBotH   = 100
	MaxBotH   = 600
)

// StyleSheet 返回全局 CSS 样式表（VS Code Dark+ 风格）。
func StyleSheet() string {
	return `
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-size: 15px; color: ` + Text + `; background: ` + EditorBg + `; overflow: hidden; }
.titlebar { display: flex; flex-direction: row; height: ` + itoa(TitleBarH) + `px; background: ` + TitleBarBg + `; align-items: center; user-select: none; overflow: hidden; }
.titlebar .logo { width: 36px; height: 36px; display: flex; align-items: center; justify-content: center; font-size: 18px; color: ` + Accent + `; }
.titlebar .menu-bar { display: flex; flex-direction: row; flex: 1; }
.titlebar .menu-item { padding: 0 8px; height: 36px; display: flex; align-items: center; cursor: pointer; font-size: 15px; color: ` + Text + `; }
.titlebar .menu-item:hover { background: ` + HoverBg + `; }
.titlebar .title-center { flex: 1; text-align: center; font-size: 14px; color: ` + TextDim + `; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.titlebar .panel-toggles { display: flex; flex-direction: row; }
.titlebar .toggle-btn { width: ` + itoa(ToggleW) + `px; height: 36px; display: flex; align-items: center; justify-content: center; cursor: pointer; color: ` + TextDim + `; }
.titlebar .toggle-btn:hover { background: ` + HoverBg + `; }
.titlebar .toggle-btn.active { color: ` + Text + `; background: ` + HoverBg + `; }
.titlebar .win-btns { display: flex; flex-direction: row; }
.titlebar .win-btn { width: ` + itoa(WinBtnW) + `px; height: 36px; display: flex; align-items: center; justify-content: center; cursor: pointer; color: ` + Text + `; }
.titlebar .win-btn:hover { background: ` + HoverBg + `; }
.titlebar .win-btn.close:hover { background: #e81123; color: #fff; }

/* 标题栏下拉菜单触发器（替代 .menu-item） */
.titlebar .dropdown { display: flex; flex-direction: row; }
.titlebar .dropdown-trigger { padding: 0 8px; height: 36px; display: flex; flex-direction: row; align-items: center; cursor: pointer; font-size: 15px; color: ` + Text + `; user-select: none; }
.titlebar .dropdown-trigger:hover { background: ` + HoverBg + `; }
.titlebar .dropdown-label { white-space: nowrap; }

/* 弹出菜单（VS Code Dark+ 风格） */
.popup-menu { display: flex; flex-direction: column; min-width: 200px; background: ` + SideBg + `; border: 1px solid #454545; padding: 4px 0; font-size: 15px; color: ` + Text + `; user-select: none; box-shadow: 0 2px 8px rgba(0,0,0,0.36); }
.popup-menu-item { padding: 4px 28px 4px 24px; height: 26px; display: flex; flex-direction: row; align-items: center; cursor: pointer; white-space: nowrap; color: ` + Text + `; }
.popup-menu-item:hover { background: ` + ActiveBg + `; }
.popup-menu-item.selected { background: ` + ActiveBg + `; }
.popup-menu-item.disabled { color: ` + TextMute + `; cursor: default; }
.popup-menu-item.disabled:hover { background: transparent; }
.popup-menu-divider { height: 1px; background: #454545; margin: 4px 0; }

/* Select 下拉选择组件 */
.select { display: block; font-size: 15px; color: ` + Text + `; }
.select-trigger { display: flex; flex-direction: row; align-items: center; justify-content: space-between; padding: 4px 8px; background: ` + InputBg + `; border: 1px solid ` + Border + `; cursor: pointer; color: ` + Text + `; user-select: none; }
.select-trigger:hover { border-color: ` + Accent + `; }
.select-label { flex: 1; }
.select-arrow { width: 16px; height: 16px; flex-shrink: 0; margin-left: 8px; display: inline-flex; align-items: center; justify-content: center; color: ` + TextDim + `; }

.body { display: flex; flex-direction: row; flex: 1; overflow: hidden; }
.left-panel { display: flex; flex-direction: column; background: ` + SideBg + `; overflow: hidden; }
.center-panel { display: flex; flex-direction: column; flex: 1; background: ` + EditorBg + `; overflow: hidden; }
.right-panel { display: flex; flex-direction: column; background: ` + ChatBg + `; overflow: hidden; }
.bottom-panel { display: flex; flex-direction: column; background: ` + SideBg + `; overflow: hidden; }

.vdivider { width: ` + itoa(DividerW) + `px; background: ` + Border + `; cursor: col-resize; flex-shrink: 0; }
.vdivider:hover, .vdivider.dragging { background: ` + Accent + `; }
.hdivider { height: ` + itoa(DividerW) + `px; background: ` + Border + `; cursor: row-resize; flex-shrink: 0; }
.hdivider:hover, .hdivider.dragging { background: ` + Accent + `; }

/* 状态栏 */
.statusbar { display: flex; flex-direction: row; height: ` + itoa(StatusH) + `px; background: ` + StatusBarBg + `; align-items: center; justify-content: space-between; padding: 0 8px; font-size: 14px; color: ` + TextDim + `; user-select: none; }
.statusbar .status-left { display: flex; flex-direction: row; gap: 12px; align-items: center; }
.statusbar .status-right { display: flex; flex-direction: row; gap: 12px; align-items: center; }
.statusbar .status-item { display: flex; flex-direction: row; align-items: center; gap: 4px; }
.statusbar .status-dot { width: 8px; height: 8px; border-radius: 50%; background: ` + TextMute + `; }
.statusbar .status-dot.running { background: ` + Warning + `; }

/* 通用 */
.panel-header { display: flex; flex-direction: row; height: 35px; align-items: center; padding: 0 8px; background: ` + PanelHeader + `; font-size: 13px; font-weight: bold; text-transform: uppercase; color: ` + TextDim + `; white-space: nowrap; line-height: 1; }
.panel-content { flex: 1; overflow: auto; }
.btn { display: inline-flex; align-items: center; justify-content: center; padding: 4px 12px; background: ` + Accent + `; color: #fff; border: none; cursor: pointer; font-size: 15px; }
.btn:hover { background: ` + AccentHover + `; }
.btn-ghost { background: transparent; color: ` + Text + `; }
.btn-ghost:hover { background: ` + HoverBg + `; }
.btn-danger { background: #5a1d1d; color: ` + Error + `; }
.btn-danger:hover { background: #7a2424; }
.input { background: ` + InputBg + `; color: ` + Text + `; border: 1px solid ` + Border + `; padding: 4px 8px; font-size: 15px; }
.input:focus { border-color: ` + Accent + `; }
.muted { color: ` + TextDim + `; }
.row { display: flex; flex-direction: row; align-items: center; }
.col { display: flex; flex-direction: column; }
.gap4 { gap: 4px; } .gap8 { gap: 8px; } .gap12 { gap: 12px; }
.flex1 { flex: 1; }
.hidden { display: none !important; }

/* 文件树 */
.file-tree { font-size: 15px; }
.file-node { display: flex; flex-direction: row; align-items: center; height: 26px; padding: 0 4px; cursor: pointer; white-space: nowrap; }
.file-node:hover { background: ` + HoverBg + `; }
.file-node.selected { background: ` + ActiveBg + `; }
.file-node .indent { flex-shrink: 0; }
.file-node .icon { width: 16px; flex-shrink: 0; text-align: center; }
.file-node .name { overflow: hidden; text-overflow: ellipsis; }

/* 编辑器标签 */
.tab-bar { display: flex; flex-direction: row; background: ` + TitleBarBg + `; overflow: hidden; }
.tab { display: flex; flex-direction: row; align-items: center; padding: 0 12px; height: 35px; cursor: pointer; font-size: 15px; color: ` + TextDim + `; border-right: 1px solid ` + Border + `; }
.tab.active { background: ` + EditorBg + `; color: ` + Text + `; }
.tab:hover:not(.active) { background: ` + HoverBg + `; }
.tab .close { margin-left: 6px; opacity: 0.6; }
.tab .close:hover { opacity: 1; }

/* 对话面板 */
.chat-msgs { flex: 1; overflow-y: auto; padding: 8px; }
.chat-msg { margin-bottom: 12px; }
.chat-msg.user { background: ` + UserBubble + `; padding: 8px 12px; border-radius: 6px; }
.chat-msg.assistant { background: ` + Border + `; padding: 8px 12px; border-radius: 6px; }
.chat-msg .role { font-size: 13px; color: ` + TextDim + `; margin-bottom: 4px; }
.chat-msg .thinking { background: #2a2a2a; padding: 6px 8px; margin: 4px 0; font-size: 14px; color: ` + TextDim + `; border-left: 2px solid ` + TextMute + `; }
.chat-msg .activity { background: #2a2a2a; padding: 4px 8px; margin: 2px 0; font-size: 14px; color: ` + TextDim + `; }
.chat-input-area { padding: 8px; border-top: 1px solid ` + Border + `; }
.chat-input { width: 100%; background: ` + InputBg + `; color: ` + Text + `; border: 1px solid ` + Border + `; padding: 8px; font-size: 15px; min-height: 60px; }
.chat-send-btn { margin-top: 4px; }

/* 终端 */
.terminal { background: ` + EditorBg + `; color: ` + Text + `; font-family: monospace; font-size: 15px; padding: 4px; overflow: auto; white-space: pre; }

/* Git 面板 */
.git-file { display: flex; flex-direction: row; align-items: center; height: 26px; padding: 0 8px; cursor: pointer; font-size: 15px; }
.git-file:hover { background: ` + HoverBg + `; }
.git-file .status { width: 16px; font-weight: bold; }
.git-file .path { overflow: hidden; text-overflow: ellipsis; }

/* 搜索面板 */
.search-result { padding: 4px 8px; cursor: pointer; font-size: 15px; }
.search-result:hover { background: ` + HoverBg + `; }
.search-result .file { color: ` + Info + `; }
.search-result .line { color: ` + TextDim + `; font-size: 14px; }

/* 设置面板 */
.settings-row { display: flex; flex-direction: row; align-items: center; padding: 8px; gap: 12px; }
.settings-label { width: 200px; font-size: 15px; color: ` + Text + `; }
.settings-field { flex: 1; }

/* 欢迎页 */
.welcome { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; padding: 40px; }
.welcome h1 { font-size: 32px; color: ` + Text + `; margin-bottom: 16px; }
.welcome p { font-size: 16px; color: ` + TextDim + `; margin-bottom: 24px; }
.welcome .actions { display: flex; flex-direction: row; gap: 12px; }
.welcome .action { padding: 12px 24px; background: ` + Accent + `; color: #fff; cursor: pointer; font-size: 16px; }
.welcome .action:hover { background: ` + AccentHover + `; }

/* 对话流式光标闪烁动画 */
@keyframes chat-cursor-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}
`
}

// ApplyTheme 把全局样式表注入文档。
func ApplyTheme(doc *dom.Document) {
	doc.AddStyleSheet(StyleSheet())
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
