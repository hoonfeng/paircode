// editorpanel 对外接口 —— 编辑器面板。外部经 Editor 单例(Open/OpenAt/Save/…)用;
// 菜单(耦合 ctxmenu/chat)与欢迎页配置由 main 注入(同 termpanel.OnContextMenu 模式)。
//
// 设计区分：
//   - 代码欢迎页（Code Welcome）：已打开工作区但未打开文件时，编辑区空状态展示
//   - IDE 欢迎页（IDE Welcome）：未打开工作区/项目时，中列整体展示
//
//go:build windows

package editorpanel

import (
	"github.com/user/gou-ide/cmd/companion/core"
	"github.com/user/goui/internal/widget"
)

// 注入回调/配置(菜单耦合 ctxmenu/chat、欢迎页按钮由 main 注入，故注入而非本包持有)。
var (
	OnContentMenu func(x, y float64)        // 编辑器内容区右键菜单(main 注入 editorContentMenu)
	OnTabMenu     func(x, y float64, i int) // 编辑器标签右键菜单(带标签索引；main 注入 editorTabMenu)

	// ── 代码欢迎页回调（工作区已打开，但无文件标签时）──
	OnOpenFile func() // 「打开文件」按钮(main 注入 ctxmenupanel.OpenFileViaDialog)
	OnNewFile  func() // 「新建文件」按钮(main 注入 NewEntryIn)

	// ── IDE 欢迎页回调（未打开工作区时）──
	OnOpenFolder func() // 「打开文件夹作为工作区」(main 注入 ctxmenupanel.OpenFolderViaDialog)
	OnNewProject func() // 「新建项目」(main 注入 core.NewProjectViaDialog)
	OnOpenRecent func(p string) // 「打开最近项目」(main 注入 core.OpenProject)

	OnReferences func(refs []widget.CodeLoc) // 转到引用结果回调(main 注入 editorReferences，结果列进面板)
	OnSymbols    func(syms []widget.CodeSym) // 文档大纲结果回调(main 注入 editorSymbols)

	// OnCursorMoved 光标位置变化通知回调(main 注入，触发状态栏 Ln/Col 刷新)。
	OnCursorMoved func()
)

// workspaceClosed 标记：关闭项目/工作区后，强制切 IDE 欢迎页（防止重建时序问题）。
var workspaceClosed bool

// MarkWorkspaceClosed 标记工作区已关闭，下一次 MidContent() 强制返回 IDE 欢迎页。
func MarkWorkspaceClosed() { workspaceClosed = true }

// MidContent 中列内容分发：未打开工作区 → IDE 欢迎页；已打开 → 编辑器面板。
// 供 main.go 的 midColumn() 调用。
func MidContent() widget.Widget {
	if workspaceClosed {
		workspaceClosed = false
		return buildIdeWelcome()
	}
	if len(core.Folders) == 0 {
		return buildIdeWelcome()
	}
	return &EditorPanel{}
}

// Area 中列编辑区入口(main 的 midColumn 调用，已由 MidContent 替代，保留兼容）。
func Area() widget.Widget { return &EditorPanel{} }

// Reset 复位编辑器单例(测试用)。
func Reset() { Editor = &editorState{} }

// NewTabForTest 造一个仅含路径的标签(测试用，不读盘)。
func NewTabForTest(path string) *editorTab { return &editorTab{path: path} }

// ─── editorState 外部访问 ───

// Tabs 当前打开的标签（外部只读遍历）。
func (e *editorState) Tabs() []*editorTab { return e.tabs }

// SetTabs 直接设标签集（测试用；变参以便外部传 NewTabForTest 的返回值，无需命名内部类型）。
func (e *editorState) SetTabs(ts ...*editorTab) { e.tabs = ts }

// BumpReload 重载令牌 +1（设置面板改字体/字号后，令各标签编辑器按令牌重建）。
func (e *editorState) BumpReload() { e.reload++ }

// ─── editorTab 字段访问(ctxmenu / 测试用，避免导出内部字段名) ───

func (t *editorTab) Path() string    { return t.path }
func (t *editorTab) Lang() string    { return t.lang }
func (t *editorTab) Content() string { return t.content }
func (t *editorTab) Dirty() bool     { return t.dirty }
func (t *editorTab) SetDirty(b bool) { t.dirty = b }
