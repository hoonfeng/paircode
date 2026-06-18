// Application 由 main 注入（用于获取窗口句柄）。

// 右键上下文菜单（复刻参考源：文件树 / 编辑器标签 / 终端）。复用 goui 的 widget.ShowContextMenu
// （Overlay 菜单）+ widget.ContextArea（右键触发、布局透传）。新建/重命名走 showPrompt 输入框，
// 删除走 widget.ShowConfirm。详见 AGENTS.md。
//
//go:build windows


package ctxmenupanel



import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/ui/chat"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	"github.com/hoonfeng/paircode/cmd/companion/ui/filetree"
	"github.com/hoonfeng/paircode/cmd/companion/ui/terminal"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
	"github.com/hoonfeng/goui/pkg/window"
)

var Application *struct{ Window interface{ NativeHandle() uintptr } }

// openFileViaDialog 弹系统「打开文件」对话框，选中即在编辑器打开（标题栏「文件→打开文件」Ctrl+O）。
func OpenFileViaDialog() {
	if window.OpenFileDialog == nil {
		return
	}
	var h uintptr
	if Application != nil && Application.Window != nil {
		h = Application.Window.NativeHandle()
	}
	if p := window.OpenFileDialog(h, "打开文件", ""); p != "" {
		editorpanel.Editor.Open(p)
		filetreepanel.FileTree.SelectPath(p)
	}
}

// pickFolder 弹系统“选择文件夹”对话框，返回选中路径（取消=空）。
func PickFolder(title string) string {
	if window.OpenFolderDialog == nil {
		return ""
	}
	var h uintptr
	if Application != nil && Application.Window != nil {
		h = Application.Window.NativeHandle()
	}
	return window.OpenFolderDialog(h, title)
}

// openFolderViaDialog 选文件夹后打开：无打开的工作区→本窗口替换；已有工作区→开新窗口（见 openProject）。
func OpenFolderViaDialog() {
	if p := PickFolder("打开文件夹"); p != "" {
		core.OpenProject(p)
	}
}

// addFolderViaDialog 添加文件夹到工作区：把选中文件夹加进来，变多根工作区（VS Code「Add Folder to Workspace」）。
func AddFolderViaDialog() {
	if p := PickFolder("添加文件夹到工作区"); p != "" {
		core.AddFolder(p)
	}
}

// workspaceRootMenu 多根工作区里，右键某个根文件夹的菜单：排序（首个=Agent 主文件夹）+ 移除。
func WorkspaceRootMenu(x, y float64, path string) {
	i := core.IndexOfFolder(path)
	items := []widget.MenuItem{
		mi("plus", "新建文件", func() { NewEntryIn(path, false) }),
		mi("folder-plus", "新建文件夹", func() { NewEntryIn(path, true) }),
		sep(),
		mi("terminal", "在终端打开", func() { termpanel.Active().OpenDir(path) }),
		mi("folder", "在资源管理器中显示", func() { RevealInExplorer(path, true) }),
	}
	if i > 0 { // 非首位 → 可一键设为首选（拖拽手柄也能排序）
		items = append(items, sep(), mi("star", "设为首选项目（Agent 主文件夹）", func() { core.SetPrimaryFolder(path) }))
	}
	items = append(items, sep(), miD("circle-x", "从工作区移除文件夹", func() { core.RemoveFolder(path) }))
	ShowMenu(x, y, items)
}

// mi 菜单项（icon=Lucide 图标名，空=无图标）。
func mi(icon, label string, onClick func()) widget.MenuItem {
	return widget.MenuItem{Icon: icon, Label: label, OnClick: onClick, Enabled: true}
}

// miD 危险菜单项（红字/红底，如删除）。
func miD(icon, label string, onClick func()) widget.MenuItem {
	return widget.MenuItem{Icon: icon, Label: label, OnClick: onClick, Enabled: true, Danger: true}
}

// sep 分组分隔线。
func sep() widget.MenuItem { return widget.MenuItem{Separator: true} }

// showMenu 用 GitHub 深色配色弹出右键菜单（统一各处外观，匹配参考源深色上下文菜单）。
var currentMenuID int // 当前显示的右键菜单 overlay id，关闭旧菜单防叠加

func ShowMenu(x, y float64, items []widget.MenuItem) {
	// 关闭上一次的右键菜单（防止多个菜单叠加）
	if currentMenuID > 0 {
		widget.HideOverlay(currentMenuID)
	}
	currentMenuID = widget.ShowContextMenuStyled(x, y, items, *ui.BgMuted, *ui.Fg, *ui.AccentStrong, *ui.Border)
}

// copyToClipboard 写剪贴板（无实现则忽略）。
func CopyToClipboard(s string) {
	if widget.ClipboardWrite != nil {
		widget.ClipboardWrite(s)
	}
}

// addToChat 把引用文本加入 DraftRefs 列表，以 chip 组件展示在输入框上方（不再塞入输入框草稿）。
func AddToChat(text string) {
	chatpanel.TheState.AddRef(text)
}

// revealInExplorer 在资源管理器中定位文件 / 打开目录（Windows）。explorer 退出码常非 0，忽略。
func RevealInExplorer(path string, isDir bool) {
	if isDir {
		_ = exec.Command("explorer", path).Start()
	} else {
		_ = exec.Command("explorer", "/select,"+path).Start()
	}
}

// showPrompt 输入对话框（标题 + 单行输入 + 取消/确定）；确定且非空回调输入值。
func ShowPrompt(title, initial string, onOk func(string)) {
	val := initial
	in := widget.NewInput("", nil)
	in.Text = initial
	in.OnTextChanged = func(s string) { val = s }
	in.Color = *ui.Fg
	in.CursorColor = *ui.Fg
	in.BGColor = *ui.Bg
	in.BorderColor = *ui.Border
	in.FocusBorderColor = *ui.Accent
	in.HoverBorderColor = *ui.Border
	var id int
	dlg := widget.NewDialog(title, widget.Div(widget.Style{Padding: types.EdgeInsets(4)}, in)).WithWidth(360).WithFooter(
		ui.Btn("取消", func() { widget.HideOverlay(id) }),
		ui.PrimaryBtn("确定", func() {
			widget.HideOverlay(id)
			if v := strings.TrimSpace(val); v != "" {
				onOk(v)
			}
		}),
	)
	id = widget.ShowDialog(dlg)
}

// ─── 文件树菜单 ───────────────────────────────────────────

func dirOf(n *filetreepanel.FileNode) string {
	if n.IsDir() {
		return n.Path()
	}
	return filepath.Dir(n.Path())
}

// relForFileTree 取相对所属工作区文件夹的 slash 路径（多根：找包含它的那个根）。
func relForFileTree(p string) string {
	for _, r := range filetreepanel.FileTree.Roots() {
		if rel, err := filepath.Rel(r.Path(), p); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	if rel, err := filepath.Rel(core.Root(), p); err == nil {
		return filepath.ToSlash(rel)
	}
	return p
}

// fileNodeMenuItems 文件/文件夹节点右键菜单项（拆出便于测试）。
func FileNodeMenuItems(n *filetreepanel.FileNode) []widget.MenuItem {
	var items []widget.MenuItem
	if n.IsDir() {
		ic, lbl := "chevron-right", "展开"
		if n.Expanded() {
			ic, lbl = "chevron-down", "折叠"
		}
		items = append(items, mi(ic, lbl, func() { filetreepanel.FileTree.Toggle(n) }))
	} else {
		items = append(items, mi("file-code", "打开", func() { editorpanel.Editor.Open(n.Path()); filetreepanel.FileTree.SelectPath(n.Path()) }))
	}
	return append(items,
		sep(),
		mi("plus", "新建文件", func() { NewEntryIn(dirOf(n), false) }),
		mi("folder-plus", "新建文件夹", func() { NewEntryIn(dirOf(n), true) }),
		sep(),
		mi("square-pen", "重命名", func() { renameEntry(n) }),
		miD("trash-2", "删除", func() { deleteEntry(n) }),
		sep(),
		mi("copy", "复制名称", func() { CopyToClipboard(n.Name()) }),
		mi("copy", "复制相对路径", func() { CopyToClipboard(relForFileTree(n.Path())) }),
		mi("copy", "复制绝对路径", func() { CopyToClipboard(n.Path()) }),
		sep(),
		mi("message-square", "添加到对话", func() {
			pfx := "参考文件："
			if n.IsDir() {
				pfx = "参考目录："
			}
			AddToChat(pfx + relForFileTree(n.Path()))
		}),
		mi("terminal", "在终端打开", func() { termpanel.Active().OpenDir(dirOf(n)) }),
		mi("folder", "在资源管理器中打开", func() { RevealInExplorer(n.Path(), n.IsDir()) }),
		mi("refresh-cw", "刷新", func() { filetreepanel.FileTree.Refresh() }),
	)
}

func FileNodeMenu(x, y float64, n *filetreepanel.FileNode) { ShowMenu(x, y, FileNodeMenuItems(n)) }

// fileTreeEmptyItems 文件树空白处右键菜单项（根目录操作）。
func fileTreeEmptyItems() []widget.MenuItem {
	root := core.Root()
	return []widget.MenuItem{
		mi("plus", "新建文件", func() { NewEntryIn(root, false) }),
		mi("folder-plus", "新建文件夹", func() { NewEntryIn(root, true) }),
		sep(),
		mi("refresh-cw", "刷新", func() { filetreepanel.FileTree.Refresh() }),
		mi("folder", "在资源管理器中打开", func() { RevealInExplorer(root, true) }),
		mi("terminal", "在终端打开", func() { termpanel.Active().OpenDir(root) }),
	}
}

func FileTreeEmptyMenu(x, y float64) { ShowMenu(x, y, fileTreeEmptyItems()) }

// newEntryIn 在 dir 下新建文件/文件夹（弹输入框取名）。
func NewEntryIn(dir string, isDir bool) {
	title := "新建文件"
	if isDir {
		title = "新建文件夹"
	}
	ShowPrompt(title, "", func(name string) {
		p := filepath.Join(dir, name)
		var err error
		if isDir {
			err = os.MkdirAll(p, 0o755)
		} else {
			err = os.WriteFile(p, nil, 0o644)
		}
		if err != nil {
			widget.ShowAlert("出错", err.Error(), widget.MsgError, nil)
			return
		}
		filetreepanel.FileTree.Refresh()
		if !isDir {
			editorpanel.Editor.Open(p)
		}
	})
}

// renameEntry 重命名节点（弹输入框）。
func renameEntry(n *filetreepanel.FileNode) {
	ShowPrompt("重命名", n.Name(), func(name string) {
		np := filepath.Join(filepath.Dir(n.Path()), name)
		if err := os.Rename(n.Path(), np); err != nil {
			widget.ShowAlert("出错", err.Error(), widget.MsgError, nil)
			return
		}
		filetreepanel.FileTree.Refresh()
	})
}

// deleteEntry 删除节点（先确认）。
func deleteEntry(n *filetreepanel.FileNode) {
	kind := "文件"
	if n.IsDir() {
		kind = "文件夹"
	}
	widget.ShowConfirm("删除"+kind, "确定删除"+kind+"「"+n.Name()+"」？此操作不可撤销。", widget.MsgError,
		func() {
			if err := os.RemoveAll(n.Path()); err != nil {
				widget.ShowAlert("出错", err.Error(), widget.MsgError, nil)
				return
			}
			filetreepanel.FileTree.Refresh()
		}, nil)
}

// ─── 编辑器标签菜单 ───────────────────────────────────────

func EditorTabItems(i int) []widget.MenuItem {
	return []widget.MenuItem{
		mi("x", "关闭", func() { editorpanel.Editor.Close(i) }),
		mi("", "关闭其他", func() { editorpanel.Editor.CloseOthers(i) }),
		mi("", "关闭所有", func() { editorpanel.Editor.CloseAll() }),
		sep(),
		mi("copy", "复制路径", func() {
			if t := editorpanel.Editor.TabAt(i); t != nil {
				CopyToClipboard(t.Path())
			}
		}),
		mi("copy", "复制目录路径", func() {
			if t := editorpanel.Editor.TabAt(i); t != nil {
				CopyToClipboard(filepath.Dir(t.Path()))
			}
		}),
		sep(),
		mi("message-square", "添加到对话", func() {
			if t := editorpanel.Editor.TabAt(i); t != nil {
				AddToChat("参考文件：" + relForFileTree(t.Path()))
			}
		}),
	}
}

func EditorTabMenu(x, y float64, i int) { ShowMenu(x, y, EditorTabItems(i)) }

// ─── 终端菜单 ─────────────────────────────────────────────

// terminalItems 终端右键菜单：只放与当前内容相关的操作（复制/粘贴/清屏/加到对话）。
// 新建/切换 shell 放在「+ 下拉」与菜单栏「终端」菜单里，不塞进右键菜单。
func TerminalItems() []widget.MenuItem {
	return []widget.MenuItem{
		mi("copy", "复制全部", func() { CopyToClipboard(termpanel.Active().CopyAll()) }),
		mi("clipboard", "粘贴", func() { termpanel.Active().PasteToInput() }),
		mi("eraser", "清屏", func() { termpanel.Active().ClearScreen() }),
		sep(),
		mi("message-square", "添加到对话", func() { AddToChat("```\n" + termpanel.Active().CopyAll() + "```") }),
	}
}

func TerminalMenu(x, y float64) { ShowMenu(x, y, TerminalItems()) }

// ─── 编辑器内容菜单（代码编辑器/结构化工作台的右键自定义菜单，取代组件自带）─────────

func EditorContentItems() []widget.MenuItem {
	codeEd := widget.HasFocusedEditor()        // 代码视图：聚焦 CodeEditor
	tableEd := widget.HasFocusedStructEditor() // 表格视图：聚焦 StructEditor
	en := codeEd || tableEd
	// 复制/剪切/粘贴：代码视图走 RunEditorCommand，表格视图走选中单元格操作。
	copyFn := func() {
		if codeEd {
			widget.RunEditorCommand("copy")
		} else if s := widget.StructEditorCopyCell(); s != "" {
			CopyToClipboard(s)
		}
	}
	cutFn := func() {
		if codeEd {
			widget.RunEditorCommand("cut")
		} else if s := widget.StructEditorCopyCell(); s != "" {
			CopyToClipboard(s)
			widget.StructEditorPasteCell("")
		}
	}
	pasteFn := func() {
		if codeEd {
			widget.RunEditorCommand("paste")
		} else if widget.ClipboardRead != nil {
			widget.StructEditorPasteCell(widget.ClipboardRead())
		}
	}
	items := []widget.MenuItem{
		{Icon: "undo-2", Label: "撤销", Enabled: codeEd, OnClick: func() { widget.RunEditorCommand("undo") }, Shortcut: "Ctrl+Z"},
		{Icon: "redo-2", Label: "重做", Enabled: codeEd, OnClick: func() { widget.RunEditorCommand("redo") }, Shortcut: "Ctrl+Y"},
		sep(),
		{Icon: "scissors", Label: "剪切", Enabled: en, OnClick: cutFn, Shortcut: "Ctrl+X"},
		{Icon: "copy", Label: "复制", Enabled: en, OnClick: copyFn, Shortcut: "Ctrl+C"},
		{Icon: "clipboard", Label: "粘贴", Enabled: en, OnClick: pasteFn, Shortcut: "Ctrl+V"},
		sep(),
		{Icon: "list", Label: "全选", Enabled: codeEd, OnClick: func() { widget.RunEditorCommand("selectAll") }, Shortcut: "Ctrl+A"},
		{Icon: "braces", Label: "格式化文档", Enabled: codeEd, OnClick: func() { widget.RunEditorCommand("format") }},
	}
	// 自动换行：全局开关（不依赖聚焦哪个编辑器；右键菜单覆盖层会令编辑器失焦，故必须全局）。开启时显示对勾。
	wrapIcon := ""
	if widget.WordWrapEnabled() {
		wrapIcon = "check"
	}
	items = append(items, sep(), widget.MenuItem{Icon: wrapIcon, Label: "自动换行", Enabled: true, OnClick: widget.ToggleWordWrap})
	// 文件操作（对照参考补齐：在资源管理器中显示 / 复制路径 / 复制文件名）
	if t := editorpanel.Editor.ActiveTab(); t != nil {
		items = append(items, sep(),
			mi("folder", "在资源管理器中显示", func() { RevealInExplorer(t.Path(), false) }),
			mi("copy", "复制路径", func() { CopyToClipboard(t.Path()) }),
			mi("file", "复制文件名", func() { CopyToClipboard(filepath.Base(t.Path())) }),
		)
	}
	// 添加到对话：有选中代码→加代码块；否则加文件引用。
	items = append(items, sep(), mi("message-square", "添加到对话", func() {
		if sel := widget.FocusedEditorSelection(); sel != "" {
			AddToChat("```\n" + sel + "\n```")
		} else if t := editorpanel.Editor.ActiveTab(); t != nil {
			AddToChat("参考文件：" + relForFileTree(t.Path()))
		}
	}))
	// 结构化语言（有 LanguageProvider）→ 追加「代码⇄表格」视图切换。
	if t := editorpanel.Editor.ActiveTab(); t != nil && widget.HasProvider(t.Lang()) {
		icon, label := "code", "切换到代码视图"
		if widget.WorkbenchModeIsText() {
			icon, label = "table-2", "切换到表格视图"
		}
		items = append(items, sep(), mi(icon, label, widget.ToggleWorkbenchView))
	}
	return items
}

func EditorContentMenu(x, y float64) { ShowMenu(x, y, EditorContentItems()) }
