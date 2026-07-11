// Package ui 提供 GWui 版 companion 的全局应用上下文。
// 所有面板通过本包访问 app/doc 和其他面板引用。
//go:build windows && !webonly

package ui

import (
	"github.com/hoonfeng/gwui/app"
	"github.com/hoonfeng/gwui/dom"
)

// Ctx 是全局应用上下文（main 启动时设置）。
var Ctx = &AppCtx{}

// AppCtx 持有应用级引用和所有面板的指针。
type AppCtx struct {
	App *app.App
	Doc *dom.Document

	// 面板引用（由 main 创建后赋值）
	Chat     *ChatPanelRef
	Editor   *EditorPanelRef
	FileTree *FileTreePanelRef
	Terminal *TerminalPanelRef
	Git      *GitPanelRef
	Search   *SearchPanelRef

	// Shell 状态
	Shell *ShellState
}

// ShellState 持有 shell 的运行时状态（面板可见性/尺寸）。
type ShellState struct {
	LeftOpen   bool
	RightOpen  bool
	BottomOpen bool
	LeftView   string // "files" / "search" / "git"
	LeftW      float32
	RightW     float32
	BottomH    float32
	FocusMode  bool

	// DOM 引用
	LeftPanel   *dom.Element
	RightPanel  *dom.Element
	CenterPanel *dom.Element // 中间面板（编辑器+底栏），专注模式时隐藏
	BottomPanel *dom.Element
	LeftDivider *dom.Element
	RightDivider *dom.Element
	BottomDivider *dom.Element

	// Left panel view elements
	FileTreeEl *dom.Element
	SearchEl   *dom.Element
	GitEl      *dom.Element
}

// 面板引用类型（前向声明，实际类型在各面板包中定义）
type ChatPanelRef struct {
	Element *dom.Element
	Refresh func()
	Send    func(text string)
}
type EditorPanelRef struct {
	Element     *dom.Element
	Refresh     func()
	Open        func(path string)
	OpenAt      func(path string, line int)
	Save        func()
	CloseTab    func(idx int)
	CloseOthers func(idx int)
	CloseAll    func()
}
type FileTreePanelRef struct {
	Element *dom.Element
	Refresh func()
	RefreshPath func(path string)
	SelectPath  func(path string)
	RebuildRoots func()
}
type TerminalPanelRef struct {
	Element *dom.Element
	Refresh func()
}
type GitPanelRef struct {
	Element *dom.Element
	Refresh func()
}
type SearchPanelRef struct {
	Element *dom.Element
	Refresh func()
}

// MarkDirty 标记需要重绘。
func (c *AppCtx) MarkDirty() {
	if c.App != nil {
		c.App.MarkDirty()
	}
}
