// termpanel 对外接口 —— 终端面板自包含(只依赖 core/pty/vterm/ui)，外部经这几个入口用:
//   - Area() 建面板 widget;Active() 拿当前活动标签(调 OpenDir/CopyAll/...);Mgr() 多标签管理;
//   - GridFont(等宽字体，设置面板改);Active().SetShell(设默认 shell)。
//
//go:build windows

package termpanel

// OnContextMenu 终端右键菜单回调，由 main 注入(菜单含「添加到对话」等耦合 chat/clipboard，故菜单构建留 main)。
var OnContextMenu func(x, y float64)

// Active 返回当前活动终端标签状态（外部经它调 OpenDir/CopyAll/PasteToInput/ClearScreen/SetShell）。
func Active() *terminalState { return theTerminal }

// Mgr 返回多标签终端管理器（外部新建/切 shell：NewTabWithShell/SetActiveShell）。
func Mgr() *termManager { return theTermMgr }

// SetShell 设当前终端的 shell（设置面板「默认 Shell」用）。
func (t *terminalState) SetShell(s string) { t.shell = s }
