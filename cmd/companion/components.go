// 声明式装配 —— 把 companion 的面板与自定义组件注册进 declarative 系统、挂接事件 handler，
// 让 UI 能用配置(JSON / Go ComponentSpec)声明、代码只在此挂事件(对应维护者:窗口→容器→组件 + 配置化)。
//
//go:build windows

package main

import (
	"github.com/hoonfeng/paircode/cmd/companion/ui/config"
	"github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	"github.com/hoonfeng/paircode/cmd/companion/ui/filetree"
	"github.com/hoonfeng/paircode/cmd/companion/ui/git"
	"github.com/hoonfeng/paircode/cmd/companion/ui/logo"
	"github.com/hoonfeng/paircode/cmd/companion/ui/state"
	"github.com/hoonfeng/paircode/cmd/companion/ui/terminal"
	"github.com/hoonfeng/goui/pkg/widget"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui/chat"
	"github.com/hoonfeng/paircode/cmd/companion/bridge"
	"github.com/hoonfeng/paircode/cmd/companion/ui/ctxmenu"
	"github.com/hoonfeng/paircode/cmd/companion/ui/menu"
	"github.com/hoonfeng/paircode/cmd/companion/ui/settings"
)

func init() {
	// ── 面板注册为配置可引用的组件(内部仍 Go 有状态，配置树里是一个节点 {"type":"ChatPanel"}) ──
	config.Register("ChatPanel", func(widget.DeclarativeContext) widget.Widget { return chatpanel.Area() })
	config.Register("FileTreePanel", func(widget.DeclarativeContext) widget.Widget { return &filetreepanel.FileTreePanel{} })
	// 文件树的右键菜单(耦合 chat/工作区)、对话框、根拖拽持久化由 main 注入(同 editorpanel.On* 模式)。
	filetreepanel.OnNodeMenu = func(x, y float64, n *filetreepanel.FileNode) { ctxmenupanel.FileNodeMenu(x, y, n) }
	filetreepanel.OnEmptyMenu = ctxmenupanel.FileTreeEmptyMenu
	filetreepanel.OnRootMenu = ctxmenupanel.WorkspaceRootMenu
	filetreepanel.OnOpenFolder = ctxmenupanel.OpenFolderViaDialog
	filetreepanel.OnAddFolder = ctxmenupanel.AddFolderViaDialog
	filetreepanel.OnWorkspaceChanged = func(primaryChanged bool) {
		filetreepanel.FileTree.RebuildRoots()
		termpanel.Active().OpenDir(core.Root())
		if primaryChanged && chatpanel.TheState.Bridge != nil { chatpanel.TheState.Bridge.ResetForNewRoot() }
		core.Settings.WorkspaceFolders = append([]string{}, core.Folders...)
		core.Settings.LastProject = core.Root()
		core.Loaded = true
		core.Save()
		}
	config.Register("EditorPanel", func(widget.DeclarativeContext) widget.Widget { return &editorpanel.EditorPanel{} })
	// 编辑器的右键菜单(耦合 ctxmenu/chat)、转到引用/大纲(耦合面板 UI)由 main 注入(同 termpanel 模式)。
	editorpanel.OnContentMenu = ctxmenupanel.EditorContentMenu
	editorpanel.OnTabMenu = ctxmenupanel.EditorTabMenu
	editorpanel.OnReferences = menuactions.EditorReferences
	editorpanel.OnSymbols = menuactions.EditorSymbols
	config.Register("TerminalPanel", func(widget.DeclarativeContext) widget.Widget { return &termpanel.Panel{} })
	termpanel.OnContextMenu = ctxmenupanel.TerminalMenu                               // 终端右键菜单(含「添加到对话」耦合 chat)由 main 注入
	gitpanel.OnTreeRefresh = func() { filetreepanel.FileTree.Refresh() } // git 动作改文件后刷新文件树(破 git↔filetree 循环)
	config.Register("SettingsBody", func(widget.DeclarativeContext) widget.Widget { return &settingspanel.SettingsBody{} })

	// ── 自定义叶子组件 ──
	config.Register("PairLogo", func(widget.DeclarativeContext) widget.Widget { return logo.Big() })

	// ── 事件 handler(配置里 events 按名引用；代码只在此挂事件) ──
	config.OnClick("openFile", ctxmenupanel.OpenFileViaDialog)

	// ── 欢迎页按钮回调注入 ──
	// 代码欢迎页（已打开工作区但无文件标签时）：打开文件、新建文件
	editorpanel.OnOpenFile = ctxmenupanel.OpenFileViaDialog
	editorpanel.OnNewFile = func() { ctxmenupanel.NewEntryIn(core.Root(), false) }
	// IDE 欢迎页（未打开工作区时）：打开文件夹作为工作区、新建项目、打开最近项目
	editorpanel.OnOpenFolder = ctxmenupanel.OpenFolderViaDialog
	editorpanel.OnNewProject = core.NewProjectViaDialog
	editorpanel.OnOpenRecent = core.OpenProject

	// Agent bridge callback injected for lazy bridge creation
	chatpanel.NewBridge = func(cs *chatpanel.ChatState) chatpanel.AgentBridge {
		return &bridge.AgentBridge{Cs: cs}
	}

	ctxmenupanel.Application = nil
// Inject workspace callback for core.SetProject/CloseProject etc.
	core.OnSyncWorkspace = func(primaryChanged bool) {
		filetreepanel.FileTree.RebuildRoots()
		termpanel.Active().OpenDir(core.Root())
		if primaryChanged && chatpanel.TheState.Bridge != nil {
			chatpanel.TheState.Bridge.ResetForNewRoot()
		}
		core.Settings.WorkspaceFolders = append([]string{}, core.Folders...)
		core.Settings.LastProject = core.Root()
		core.Loaded = true
		core.Save()
		// 工作区变更后触发 shell 重建：打开项目 → 显示面板 + 切编辑器；关闭项目 → 显示 IDE 欢迎页
		if shellStateRef != nil {
			if len(core.Folders) > 0 {
				shellStateRef.panels = state.DefaultPanels()
			} else {
				shellStateRef.panels = state.IdeWelcomePanels()
			}
			shellStateRef.SetState()
		}
	}
	settingspanel.RegisterSettingsUI()
		// Inject bridge.ApplyIgnoreDirs callback
		settingspanel.ApplyIgnoreDirs = bridge.ApplyIgnoreDirs
}

