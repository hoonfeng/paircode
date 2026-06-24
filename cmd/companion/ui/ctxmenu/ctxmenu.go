//go:build windows

// Package ctxmenu 提供右键菜单和文件对话框（GWui 版）。
// 所有 UI 元素均为运行时动态创建，无静态面板结构。
// 如需将对话框改造为 uixml 模板，需解决跨文档组件注册（ComponentAtNode）和焦点委托问题。
package ctxmenu

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"

	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/paircode/cmd/companion/uiapi"
	filetreepanel "github.com/hoonfeng/paircode/cmd/companion/ui/filetree"
)

// ── 文件对话框（简化为 Modal 输入路径）──

// OpenFileViaDialog 打开文件对话框（Modal 输入路径）。
func OpenFileViaDialog() {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	showPathDialog(doc, "打开文件", "输入文件路径", "", func(path string) {
		if path == "" {
			return
		}
		if _, err := os.Stat(path); err != nil {
			uiapi.MessageError("文件不存在：" + path)
			return
		}
		if ui.Ctx.Editor != nil {
			ui.Ctx.Editor.Open(path)
		}
	})
}

// PickFolder 选择文件夹对话框（Modal 输入路径）。
// 注意：GWui Modal 为异步，此函数立即返回空字符串。
// 路径选择通过 Modal 的确认回调完成，调用方需适配异步模式。
func PickFolder(title string) string {
	doc := ui.Ctx.Doc
	if doc == nil {
		return ""
	}
	showPathDialog(doc, title, "输入文件夹路径", core.Root(), func(path string) {
		// 异步回调：路径无法同步返回，仅做验证
		if path != "" {
			if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
				uiapi.MessageError("文件夹不存在：" + path)
			}
		}
	})
	return "" // 简化：异步 Modal 无法同步返回
}

// OpenFolderViaDialog 打开文件夹对话框。
func OpenFolderViaDialog() {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	showPathDialog(doc, "打开文件夹", "输入文件夹路径", core.Root(), func(path string) {
		if path == "" {
			return
		}
		if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
			uiapi.MessageError("文件夹不存在：" + path)
			return
		}
		core.SetProject(path)
		uiapi.MessageSuccess("已打开文件夹：" + filepath.Base(path))
	})
}

// AddFolderViaDialog 添加文件夹到工作区。
func AddFolderViaDialog() {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	showPathDialog(doc, "添加文件夹到工作区", "输入文件夹路径", core.Root(), func(path string) {
		if path == "" {
			return
		}
		if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
			uiapi.MessageError("文件夹不存在：" + path)
			return
		}
		core.AddFolder(path)
		uiapi.MessageSuccess("已添加文件夹：" + filepath.Base(path))
	})
}

// NewEntryIn 在 root 下新建文件或文件夹。
func NewEntryIn(root string, isDir bool) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	title := "新建文件"
	if isDir {
		title = "新建文件夹"
	}
	showPathDialog(doc, title, "输入名称", "", func(name string) {
		if name == "" {
			return
		}
		full := filepath.Join(root, name)
		var err error
		if isDir {
			err = os.MkdirAll(full, 0o755)
		} else {
			err = os.WriteFile(full, []byte{}, 0o644)
		}
		if err != nil {
			uiapi.MessageError("创建失败：" + err.Error())
			return
		}
		if ui.Ctx.FileTree != nil {
			ui.Ctx.FileTree.RefreshPath(root)
		}
		if !isDir && ui.Ctx.Editor != nil {
			ui.Ctx.Editor.Open(full)
		}
		uiapi.MessageSuccess("已创建：" + name)
	})
}

// showPathDialog 显示路径输入对话框（Modal + Input + OK/Cancel）。
// 由于对话框内容完全动态（标题、占位符、初始值、回调均每次不同），
// 且 Input/Button 需要主文档的 ComponentAtNode / Focus 支持，
// 此处使用动态组件创建而非 uixml XML 模板。
func showPathDialog(doc *dom.Document, title, placeholder, initial string, onOk func(path string)) {
	modal := component.NewModal(doc)
	modal.SetTitle(title)

	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()
	body.SetAttribute("style",
		"display: flex; flex-direction: column; gap: 12px; min-width: 360px;")

	input := component.NewInput(doc, placeholder)
	if initial != "" {
		input.SetValue(initial)
	}
	body.AppendChild(input.Element())

	btnRow := doc.CreateElement("div")
	btnRow.SetAttribute("style",
		"display: flex; flex-direction: row; gap: 12px; justify-content: center; margin-top: 8px;")

	okBtn := component.NewButton(doc, "确定")
	okBtn.OnClick(func() {
		path := input.Value()
		modal.Hide()
		onOk(path)
	})
	btnRow.AppendChild(okBtn.Element())

	cancelBtn := component.NewButton(doc, "取消")
	cancelBtn.OnClick(func() {
		modal.Hide()
	})
	btnRow.AppendChild(cancelBtn.Element())

	body.AppendChild(btnRow)

	modal.Show()
}

// ── 右键菜单 ──

// FileNodeMenu 文件节点右键菜单。
func FileNodeMenu(x, y float64, n *filetreepanel.FileNode) {
	doc := ui.Ctx.Doc
	if doc == nil || n == nil {
		return
	}
	items := []component.ContextMenuItem{
		{Label: "打开", OnClick: func() {
			if !n.IsDir && ui.Ctx.Editor != nil {
				ui.Ctx.Editor.Open(n.Path)
			}
		}},
		{Label: "复制路径", OnClick: func() {
			CopyToClipboard(n.Path)
			uiapi.MessageInfo("已复制路径")
		}},
		{Label: "重命名", OnClick: func() {
			showPathDialog(doc, "重命名", "输入新名称", n.Name, func(newName string) {
				if newName == "" || newName == n.Name {
					return
				}
				newPath := filepath.Join(filepath.Dir(n.Path), newName)
				if err := os.Rename(n.Path, newPath); err != nil {
					uiapi.MessageError("重命名失败：" + err.Error())
					return
				}
				if ui.Ctx.FileTree != nil {
					ui.Ctx.FileTree.Refresh()
				}
				uiapi.MessageSuccess("已重命名")
			})
		}},
		{Divider: true},
		{Label: "删除", OnClick: func() {
			uiapi.ShowConfirm("删除", "确定删除「"+n.Name+"」？", uiapi.KindWarning, func() {
				var err error
				if n.IsDir {
					err = os.RemoveAll(n.Path)
				} else {
					err = os.Remove(n.Path)
				}
				if err != nil {
					uiapi.MessageError("删除失败：" + err.Error())
					return
				}
				if ui.Ctx.FileTree != nil {
					ui.Ctx.FileTree.Refresh()
				}
				uiapi.MessageSuccess("已删除")
			})
		}},
	}
	showContextMenu(doc, x, y, items)
}

// FileTreeEmptyMenu 文件树空白区右键菜单。
func FileTreeEmptyMenu(x, y float64) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	root := core.Root()
	items := []component.ContextMenuItem{
		{Label: "新建文件", OnClick: func() { NewEntryIn(root, false) }},
		{Label: "新建文件夹", OnClick: func() { NewEntryIn(root, true) }},
		{Divider: true},
		{Label: "打开文件夹…", OnClick: func() { OpenFolderViaDialog() }},
		{Label: "添加文件夹到工作区…", OnClick: func() { AddFolderViaDialog() }},
	}
	showContextMenu(doc, x, y, items)
}

// WorkspaceRootMenu 工作区根右键菜单。
func WorkspaceRootMenu(x, y float64, path string) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	items := []component.ContextMenuItem{
		{Label: "新建文件", OnClick: func() { NewEntryIn(path, false) }},
		{Label: "新建文件夹", OnClick: func() { NewEntryIn(path, true) }},
		{Divider: true},
		{Label: "复制路径", OnClick: func() {
			CopyToClipboard(path)
			uiapi.MessageInfo("已复制路径")
		}},
		{Label: "在资源管理器中打开", OnClick: func() {
			openInExplorer(path)
		}},
		{Divider: true},
		{Label: "从工作区移除", OnClick: func() {
			core.RemoveFolder(path)
			uiapi.MessageInfo("已从工作区移除")
		}},
	}
	showContextMenu(doc, x, y, items)
}

// EditorContentMenu 编辑器内容右键菜单。
func EditorContentMenu(x, y float64) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	items := []component.ContextMenuItem{
		{Label: "剪切", OnClick: func() {}},
		{Label: "复制", OnClick: func() {}},
		{Label: "粘贴", OnClick: func() {}},
		{Divider: true},
		{Label: "全选", OnClick: func() {}},
		{Label: "查找", OnClick: func() {}},
	}
	showContextMenu(doc, x, y, items)
}

// EditorTabMenu 编辑器标签右键菜单。
func EditorTabMenu(x, y float64, i int) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	_ = i
	items := []component.ContextMenuItem{
		{Label: "关闭", OnClick: func() {
			if ui.Ctx.Editor != nil && ui.Ctx.Editor.CloseTab != nil {
				ui.Ctx.Editor.CloseTab(i)
			}
		}},
		{Label: "关闭其他", OnClick: func() {
			if ui.Ctx.Editor != nil && ui.Ctx.Editor.CloseOthers != nil {
				ui.Ctx.Editor.CloseOthers(i)
			}
		}},
		{Label: "关闭全部", OnClick: func() {
			if ui.Ctx.Editor != nil && ui.Ctx.Editor.CloseAll != nil {
				ui.Ctx.Editor.CloseAll()
			}
		}},
	}
	showContextMenu(doc, x, y, items)
}

// TerminalMenu 终端右键菜单。
func TerminalMenu(x, y float64) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	items := []component.ContextMenuItem{
		{Label: "复制", OnClick: func() {}},
		{Label: "粘贴", OnClick: func() {}},
		{Divider: true},
		{Label: "清屏", OnClick: func() {
			uiapi.MarkDirty()
		}},
	}
	showContextMenu(doc, x, y, items)
}

// showContextMenu 创建并显示上下文菜单。
func showContextMenu(doc *dom.Document, x, y float64, items []component.ContextMenuItem) {
	cm := component.NewContextMenu(doc, items)
	cm.ShowAt(float32(x), float32(y))
}

// openInExplorer 在 Windows 资源管理器中打开。
func openInExplorer(path string) {
	_ = exec.Command("explorer.exe", path).Start()
}

// ── 剪贴板（Windows API）──

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

// CopyToClipboard 把文本复制到系统剪贴板（Windows UTF-16）。
func CopyToClipboard(text string) {
	w := utf16.Encode([]rune(text))
	w = append(w, 0) // null terminator
	size := len(w) * 2

	h, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(size))
	if h == 0 {
		return
	}
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return
	}
	// 通过 reflect.SliceHeader 创建指向全局内存的切片（避免 uintptr→Pointer 转换）
	var dst []uint16
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&dst))
	sh.Data = ptr
	sh.Len = len(w)
	sh.Cap = len(w)
	copy(dst, w)
	procGlobalUnlock.Call(h)

	procOpenClipboard.Call(0)
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	procSetClipboardData.Call(cfUnicodeText, h)
}
