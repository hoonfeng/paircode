//go:build windows

// Package ctxmenu 提供右键菜单和文件对话框（GWui 版）。
// 所有 UI 元素均为运行时动态创建，无静态面板结构。
// 如需将对话框改造为 uixml 模板，需解决跨文档组件注册（ComponentAtNode）和焦点委托问题。
package ctxmenu

import (
	"fmt"
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
	chatpanel "github.com/hoonfeng/paircode/cmd/companion/ui/chat"
	editorpanel "github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	filetreepanel "github.com/hoonfeng/paircode/cmd/companion/ui/filetree"
	termpanel "github.com/hoonfeng/paircode/cmd/companion/ui/terminal"
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

// FileNodeMenu 文件节点右键菜单（根据文件/目录类型动态构建）。
func FileNodeMenu(x, y float64, n *filetreepanel.FileNode) {
	doc := ui.Ctx.Doc
	if doc == nil || n == nil {
		return
	}
	if n.IsDir {
		dirNodeMenu(doc, x, y, n)
	} else {
		fileNodeMenu(doc, x, y, n)
	}
}

// fileNodeMenu 文件专属右键菜单。
func fileNodeMenu(doc *dom.Document, x, y float64, n *filetreepanel.FileNode) {
	items := []component.ContextMenuItem{
		// ── 打开 ──
		{Label: "打开", OnClick: func() {
			if ui.Ctx.Editor != nil {
				ui.Ctx.Editor.Open(n.Path)
			}
		}},
		{Label: "打开到侧边", OnClick: func() {
			uiapi.MessageInfo("打开到侧边功能开发中，现在以正常模式打开文件")
			if ui.Ctx.Editor != nil {
				ui.Ctx.Editor.Open(n.Path)
			}
		}},

		// ── 剪贴板 ──
		{Divider: true},
		{Label: "剪切", OnClick: func() {
			CopyToClipboard(n.Path)
			uiapi.MessageInfo("已复制文件路径（剪切操作待实现完整流程）")
		}},
		{Label: "复制", OnClick: func() {
			CopyToClipboard(n.Path)
			uiapi.MessageInfo("已复制文件路径")
		}},
		{Label: "粘贴", OnClick: func() {
			uiapi.MessageInfo("请在目标目录下使用粘贴（功能待实现）")
		}},

		// ── 操作 ──
		{Divider: true},
		{Label: "重命名", OnClick: func() {
			showRenameDialog(doc, n)
		}},
		{Label: "删除", OnClick: func() {
			showDeleteConfirm(n)
		}},

		// ── 路径 ──
		{Divider: true},
		{Label: "复制路径", OnClick: func() {
			CopyToClipboard(n.Path)
			uiapi.MessageInfo("已复制完整路径")
		}},
		{Label: "复制相对路径", OnClick: func() {
			rel := relativePath(n.Path)
			CopyToClipboard(rel)
			uiapi.MessageInfo("已复制相对路径：" + rel)
		}},
		{Label: "复制文件名", OnClick: func() {
			CopyToClipboard(n.Name)
			uiapi.MessageInfo("已复制文件名：" + n.Name)
		}},

		// ── 终端/系统 ──
		{Divider: true},
		{Label: "在终端中打开", OnClick: func() {
			termpanel.OpenActiveTerminalDir(filepath.Dir(n.Path))
			uiapi.MessageInfo("已在终端中打开所在目录")
		}},
		{Label: "在资源管理器中显示", OnClick: func() {
			selectInExplorer(n.Path)
		}},

		// ── AI 操作 ──
		{Divider: true},
		{Label: "AI: 解释此文件", OnClick: func() {
			if ui.Ctx.Editor != nil {
				ui.Ctx.Editor.Open(n.Path)
			}
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/explain " + n.Path)
			} else {
				uiapi.MessageInfo("AI 解释功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 优化此文件", OnClick: func() {
			if ui.Ctx.Editor != nil {
				ui.Ctx.Editor.Open(n.Path)
			}
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/optimize " + n.Path)
			} else {
				uiapi.MessageInfo("AI 优化功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 审查代码", OnClick: func() {
			if ui.Ctx.Editor != nil {
				ui.Ctx.Editor.Open(n.Path)
			}
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/review " + n.Path)
			} else {
				uiapi.MessageInfo("AI 审查功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 生成单元测试", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/unittest " + n.Path)
			} else {
				uiapi.MessageInfo("AI 单元测试生成功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 重构此文件", OnClick: func() {
			if ui.Ctx.Editor != nil {
				ui.Ctx.Editor.Open(n.Path)
			}
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/refactor " + n.Path)
			} else {
				uiapi.MessageInfo("AI 重构功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 查找引用", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/findrefs " + n.Path)
			} else {
				uiapi.MessageInfo("查找引用功能开发中，将在后续版本支持")
			}
		}},
		{Label: "AI: 翻译注释", OnClick: func() {
			if ui.Ctx.Editor != nil {
				ui.Ctx.Editor.Open(n.Path)
			}
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/translate " + n.Path)
			} else {
				uiapi.MessageInfo("AI 翻译功能：请确保聊天面板可用")
			}
		}},
	}
	showContextMenu(doc, x, y, items)
}

// dirNodeMenu 目录专属右键菜单。
func dirNodeMenu(doc *dom.Document, x, y float64, n *filetreepanel.FileNode) {
	items := []component.ContextMenuItem{
		// ── 展开/折叠 ──
		{Label: "展开/折叠", OnClick: func() {
			if ui.Ctx.FileTree != nil {
				filetreepanel.FileTree.Toggle(n)
			}
		}},

		// ── 新建 ──
		{Divider: true},
		{Label: "新建文件", OnClick: func() { NewEntryIn(n.Path, false) }},
		{Label: "新建文件夹", OnClick: func() { NewEntryIn(n.Path, true) }},

		// ── 剪贴板 ──
		{Divider: true},
		{Label: "剪切", OnClick: func() {
			CopyToClipboard(n.Path)
			uiapi.MessageInfo("已复制文件夹路径（剪切操作待实现）")
		}},
		{Label: "复制", OnClick: func() {
			CopyToClipboard(n.Path)
			uiapi.MessageInfo("已复制文件夹路径")
		}},
		{Label: "粘贴", OnClick: func() {
			uiapi.MessageInfo("请在目标目录下使用粘贴（功能待实现）")
		}},

		// ── 操作 ──
		{Divider: true},
		{Label: "重命名", OnClick: func() {
			showRenameDialog(doc, n)
		}},
		{Label: "删除", OnClick: func() {
			showDeleteConfirm(n)
		}},

		// ── 路径 ──
		{Divider: true},
		{Label: "复制路径", OnClick: func() {
			CopyToClipboard(n.Path)
			uiapi.MessageInfo("已复制完整路径")
		}},
		{Label: "复制相对路径", OnClick: func() {
			rel := relativePath(n.Path)
			CopyToClipboard(rel)
			uiapi.MessageInfo("已复制相对路径：" + rel)
		}},

		// ── 终端/系统 ──
		{Divider: true},
		{Label: "在终端中打开", OnClick: func() {
			termpanel.OpenActiveTerminalDir(n.Path)
			uiapi.MessageInfo("已在终端中打开此目录")
		}},
		{Label: "在资源管理器中显示", OnClick: func() {
			openInExplorer(n.Path)
		}},

		// ── AI 操作 ──
		{Divider: true},
		{Label: "AI: 分析目录结构", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/tree " + n.Path)
			} else {
				uiapi.MessageInfo("AI 目录分析功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 递归解释目录", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/explain-dir " + n.Path)
			} else {
				uiapi.MessageInfo("AI 目录解释功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 查找引用", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/findrefs-dir " + n.Path)
			} else {
				uiapi.MessageInfo("查找引用功能开发中")
			}
		}},
		{Label: "AI: 生成目录概述", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/summarize-dir " + n.Path)
			} else {
				uiapi.MessageInfo("AI 概述功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 审查目录代码", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/review-dir " + n.Path)
			} else {
				uiapi.MessageInfo("AI 代码审查功能：请确保聊天面板可用")
			}
		}},

		// ── 工作区操作 ──
		{Divider: true},
		{Label: "添加到工作区", OnClick: func() {
			core.AddFolder(n.Path)
			uiapi.MessageInfo("已添加文件夹到工作区：" + n.Name)
		}},
	}
	showContextMenu(doc, x, y, items)
}

// showRenameDialog 显示重命名对话框。
func showRenameDialog(doc *dom.Document, n *filetreepanel.FileNode) {
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
		if n.IsDir {
			uiapi.MessageSuccess("文件夹已重命名为：" + newName)
		} else {
			uiapi.MessageSuccess("文件已重命名为：" + newName)
		}
	})
}

// showDeleteConfirm 显示删除确认对话框。
func showDeleteConfirm(n *filetreepanel.FileNode) {
	name := n.Name
	kind := "文件"
	if n.IsDir {
		kind = "文件夹"
	}
	uiapi.ShowConfirm("删除", fmt.Sprintf("确定删除%s「%s」？\n此操作不可恢复。", kind, name), uiapi.KindWarning, func() {
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
		uiapi.MessageSuccess("已删除：" + name)
	})
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
		{Label: "粘贴", OnClick: func() {
			uiapi.MessageInfo("粘贴功能待实现（请使用系统快捷键 Ctrl+V）")
		}},
		{Divider: true},
		{Label: "打开文件夹…", OnClick: func() { OpenFolderViaDialog() }},
		{Label: "添加文件夹到工作区…", OnClick: func() { AddFolderViaDialog() }},
		{Divider: true},
		{Label: "在终端中打开", OnClick: func() {
			termpanel.OpenActiveTerminalDir(root)
			uiapi.MessageInfo("已在终端中打开工作区目录")
		}},
	}
	showContextMenu(doc, x, y, items)
}

// WorkspaceRootMenu 工作区根右键菜单。
func WorkspaceRootMenu(x, y float64, path string) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	isSingleRoot := len(core.Folders) <= 1
	rootName := filepath.Base(path)
	items := []component.ContextMenuItem{
		{Label: "新建文件", OnClick: func() { NewEntryIn(path, false) }},
		{Label: "新建文件夹", OnClick: func() { NewEntryIn(path, true) }},
		{Divider: true},
		{Label: "复制路径", OnClick: func() {
			CopyToClipboard(path)
			uiapi.MessageInfo("已复制路径：" + path)
		}},
		{Label: "在资源管理器中打开", OnClick: func() {
			openInExplorer(path)
		}},
		{Label: "在终端中打开", OnClick: func() {
			termpanel.OpenActiveTerminalDir(path)
			uiapi.MessageInfo("已在终端中打开此目录")
		}},

		// ── AI 项目操作 ──
		{Divider: true},
		{Label: "AI: 项目概览", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/project-overview")
			} else {
				uiapi.MessageInfo("AI 项目概览功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 代码审查", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/review-all")
			} else {
				uiapi.MessageInfo("AI 代码审查功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 生成项目文档", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/generate-docs")
			} else {
				uiapi.MessageInfo("AI 文档生成功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 重构分析", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/refactor-analysis")
			} else {
				uiapi.MessageInfo("AI 重构分析功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 技术债务分析", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/tech-debt")
			} else {
				uiapi.MessageInfo("AI 技术债务分析功能：请确保聊天面板可用")
			}
		}},
		{Label: "AI: 代码规范检查", OnClick: func() {
			if ui.Ctx.Chat != nil && ui.Ctx.Chat.Send != nil {
				ui.Ctx.Chat.Send("/lint-all")
			} else {
				uiapi.MessageInfo("AI 规范检查功能：请确保聊天面板可用")
			}
		}},

		{Divider: true},
		{Label: "重命名", OnClick: func() {
			showPathDialog(doc, "重命名文件夹", "输入新名称", rootName, func(newName string) {
				if newName == "" || newName == rootName {
					return
				}
				newPath := filepath.Join(filepath.Dir(path), newName)
				if err := os.Rename(path, newPath); err != nil {
					uiapi.MessageError("重命名失败：" + err.Error())
					return
				}
				// 更新工作区中的路径
				core.RemoveFolder(path)
				core.AddFolder(newPath)
				if ui.Ctx.FileTree != nil {
					ui.Ctx.FileTree.RebuildRoots()
				}
				uiapi.MessageSuccess("文件夹已重命名为：" + newName)
			})
		}},
		{Label: "删除文件夹", OnClick: func() {
			uiapi.ShowConfirm("删除文件夹", fmt.Sprintf("确定永久删除「%s」？\n此操作不可恢复。", rootName), uiapi.KindWarning, func() {
				if err := os.RemoveAll(path); err != nil {
					uiapi.MessageError("删除失败：" + err.Error())
					return
				}
				core.RemoveFolder(path)
				uiapi.MessageSuccess("已删除：" + rootName)
			})
		}},
		{Divider: true},
		{Label: "在新建窗口中打开", OnClick: func() {
			core.OpenInNewWindow(path)
		}},
	}
	if !isSingleRoot {
		items = append(items,
			component.ContextMenuItem{Divider: true},
			component.ContextMenuItem{
				Label: "从工作区移除",
				OnClick: func() {
					core.RemoveFolder(path)
					uiapi.MessageInfo("已从工作区移除：" + rootName)
				},
			},
		)
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
		{Label: "撤销", OnClick: func() { editorpanel.Editor.Undo() }},
		{Label: "重做", OnClick: func() { editorpanel.Editor.Redo() }},
		{Divider: true},
		{Label: "剪切", OnClick: func() { editorpanel.Editor.CutSelection() }},
		{Label: "复制", OnClick: func() { editorpanel.Editor.CopySelection() }},
		{Label: "粘贴", OnClick: func() { editorpanel.Editor.PasteText() }},
		{Divider: true},
		{Label: "全选", OnClick: func() {
			if ce := editorpanel.Editor.ActiveCodeEditor(); ce != nil {
				ce.SelectAll()
			}
		}},
	}
	showContextMenu(doc, x, y, items)
}

// EditorTabMenu 编辑器标签右键菜单。
func EditorTabMenu(x, y float64, i int) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
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
		{Divider: true},
		{Label: "复制路径", OnClick: func() {
			path := editorpanel.ActivePath()
			if path != "" {
				CopyToClipboard(path)
				uiapi.MessageSuccess("已复制路径：" + path)
			} else {
				uiapi.MessageInfo("没有打开的文件")
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
		{Label: "复制全部", OnClick: func() {
			text := termpanel.CopyActiveAll()
			if text != "" {
				CopyToClipboard(text)
				uiapi.MessageSuccess("已复制终端内容")
			}
		}},
		{Label: "粘贴", OnClick: func() {
			text := component.PasteFromClipboard()
			if text != "" {
				termpanel.PasteToActive(text)
			}
		}},
		{Divider: true},
		{Label: "清屏", OnClick: func() {
			termpanel.ClearActive()
		}},
	}
	showContextMenu(doc, x, y, items)
}

// ── 对话（Chat）右键菜单 ──

// ChatMessageContextMenu 对话面板消息区右键菜单。
func ChatMessageContextMenu(x, y float64) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	items := []component.ContextMenuItem{
		{Label: "复制选中文本", OnClick: func() {
			CopyToClipboard(getSelectedText())
			uiapi.MessageInfo("已复制选中文本")
		}},
		{Label: "复制整个对话", OnClick: func() {
			if chatpanel.TheState != nil {
				chatpanel.TheState.ExportActive()
			} else {
				uiapi.MessageInfo("对话面板未初始化")
			}
		}},
		{Label: "以当前消息为引用发送", OnClick: func() {
			// 获取当前活跃会话的最后一条助手消息
			if chatpanel.TheState != nil && chatpanel.TheState.Store.Active() != nil {
				t := chatpanel.TheState.Store.Active()
				if len(t.Messages) > 0 {
					last := t.Messages[len(t.Messages)-1]
					if last.Text != "" {
						chatpanel.TheState.AddRef(last.Text)
						uiapi.MessageInfo("已添加引用到输入框")
					}
				}
			}
		}},
		{Divider: true},
		{Label: "AI: 分析选中内容", OnClick: func() {
			if chatpanel.TheState != nil {
				chatpanel.TheState.Send("/analyze " + getSelectedText())
			}
		}},
		{Label: "AI: 继续生成", OnClick: func() {
			if chatpanel.TheState != nil {
				chatpanel.TheState.Send("/continue")
			}
		}},
		{Label: "AI: 优化提示词", OnClick: func() {
			if chatpanel.TheState != nil {
				chatpanel.TheState.Send("/improve " + getSelectedText())
			}
		}},
		{Divider: true},
		{Label: "清空对话", OnClick: func() {
			if chatpanel.TheState != nil {
				t := chatpanel.TheState.Store.Active()
				if t != nil {
					t.Messages = nil
					chatpanel.TheState.Refresh()
					uiapi.MessageInfo("已清空当前对话")
				}
			}
		}},
		{Label: "导出对话(Markdown)", OnClick: func() {
			if chatpanel.TheState != nil {
				chatpanel.TheState.ExportActive()
			}
		}},
	}
	showContextMenu(doc, x, y, items)
}

// getSelectedText 获取当前选中文本（简化实现）。
func getSelectedText() string {
	// 从聊天输入框获取内容作为上下文
	if chatpanel.TheState != nil {
		draft := chatpanel.TheState.Store.Draft
		if draft != "" {
			return draft
		}
		t := chatpanel.TheState.Store.Active()
		if t != nil && len(t.Messages) > 0 {
			last := t.Messages[len(t.Messages)-1]
			if last.Text != "" {
				// 取最后 200 个字符作为上下文
				runes := []rune(last.Text)
				if len(runes) > 200 {
					return string(runes[len(runes)-200:])
				}
				return last.Text
			}
		}
	}
	return ""
}

// showContextMenu 创建并显示上下文菜单。
// 使用包级变量 currentCM 跟踪当前菜单实例，显示前关闭旧菜单，防止多次右键叠加。
var currentCM *component.ContextMenu

func showContextMenu(doc *dom.Document, x, y float64, items []component.ContextMenuItem) {
	if currentCM != nil {
		currentCM.Hide()
	}
	cm := component.NewContextMenu(doc, items)
	cm.SetOnClose(func() {
		if currentCM == cm {
			currentCM = nil
		}
	})
	cm.ShowAt(float32(x), float32(y))
	currentCM = cm
}

// ChatInputContextMenu 聊天输入框右键菜单。
// selectedText 为输入框中当前选中的文本（可为空）。
func ChatInputContextMenu(x, y float64, selectedText string) {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	items := []component.ContextMenuItem{
		{Label: "剪切", OnClick: func() {
			if selectedText != "" {
				CopyToClipboard(selectedText)
			}
			// 实际剪切操作由键盘 Ctrl+X 处理；此处复制后清空由输入框自身处理
		}},
		{Label: "复制", OnClick: func() {
			if selectedText != "" {
				CopyToClipboard(selectedText)
			}
		}},
		{Label: "粘贴", OnClick: func() {
			// 粘贴由 TextArea 的 Ctrl+V 处理；此处仅占位
			// 可通过全局粘贴函数触发
		}},
		{Divider: true},
		{Label: "全选", OnClick: func() {
			// 全选由 TextArea 的 Ctrl+A 处理
		}},
	}
	showContextMenu(doc, x, y, items)
}

// ── 工具函数 ──

// relativePath 返回相对于工作区根的路径。若不在工作区内则返回完整路径。
func relativePath(absPath string) string {
	root := core.Root()
	if root == "" {
		return absPath
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return absPath
	}
	return rel
}

// openInExplorer 在 Windows 资源管理器中打开目录。
func openInExplorer(path string) {
	_ = exec.Command("explorer.exe", path).Start()
}

// selectInExplorer 在 Windows 资源管理器中选中文件。
func selectInExplorer(path string) {
	_ = exec.Command("explorer.exe", "/select,", path).Start()
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
