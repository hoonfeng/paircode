// Package workspace 是工作区管理（项目打开/切换/多根管理）。
// 数据层(Folders/Root/…)在 core 包；本文件封装工作区操作（读 Folders + 面板回调）。
//
//go:build windows

package core

import (
	"os"
	"path/filepath"

	"github.com/hoonfeng/paircode/cmd/companion/uiapi"
)

// ─── 注入回调（UI 面板操作，由 main 注入）──

var (
	OnSyncWorkspace func(primaryChanged bool)
	OnPickFolder    func(title string) string
	OnShowPrompt    func(title, initial string, onOk func(string))
	OnOpenFolder    func() // 注入 ctxmenupanel.OpenFolderViaDialog
	OnAddFolder     func() // 注入 ctxmenupanel.AddFolderViaDialog
	OnOpenProject   func(p string)
	// OnCloseProject  关闭当前项目后回调（清除编辑器标签 + 刷新界面）
	OnCloseProject func()
	// OnClearWorkspace 清空工作区后回调（清除编辑器标签 + 刷新界面 → IDE 欢迎页）
	OnClearWorkspace func()
	// OnShowManager 显示管理工作区文件夹对话框（UI 层注入）
	OnShowManager func()
)

// LoadLastProject 启动恢复工作区。
func LoadLastProject() {
	folders := Settings.WorkspaceFolders
	if len(folders) == 0 && Settings.LastProject != "" {
		folders = []string{Settings.LastProject}
	}
	for _, p := range folders {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			Folders = append(Folders, p)
		}
	}
	if len(Folders) > 0 {
		AddRecent(Folders[0])
	}
}

// AddRecent 记入最近项目。
func AddRecent(p string) {
	if p == "" {
		return
	}
	out := []string{p}
	for _, r := range Settings.RecentProjects {
		if r != p && len(out) < 12 {
			out = append(out, r)
		}
	}
	Settings.RecentProjects = out
}

// OpenProject 打开某文件夹。
func OpenProject(p string) {
	if p == "" {
		return
	}
	if len(Folders) == 0 {
		SetProject(p)
	} else {
		OpenInNewWindow(p)
	}
}

// PersistWorkspace 把当前工作区写进设置。
func PersistWorkspace() {
	if len(Folders) == 0 {
		return
	}
	Settings.WorkspaceFolders = append([]string{}, Folders...)
	Settings.LastProject = Root()
	Loaded = true
	Save()
}

// SetProject 打开文件夹替换整个工作区。
func SetProject(p string) {
	if p == "" {
		return
	}
	Folders = []string{p}
	AddRecent(p)
	if OnSyncWorkspace != nil {
		OnSyncWorkspace(true)
	}
}

// AddFolder 添加文件夹到工作区。
func AddFolder(p string) {
	if p == "" {
		return
	}
	for _, f := range Folders {
		if f == p {
			return
		}
	}
	if len(Folders) == 0 {
		Folders = append(Folders, Root())
		if Folders[0] == p {
			if OnSyncWorkspace != nil {
				OnSyncWorkspace(false)
			}
			return
		}
	}
	Folders = append(Folders, p)
	if OnSyncWorkspace != nil {
		OnSyncWorkspace(false)
	}
}

// SetPrimaryFolder 把文件夹移到首位。
func SetPrimaryFolder(p string) {
	i := IndexOfFolder(p)
	if i <= 0 {
		return
	}
	Folders = append(Folders[:i], Folders[i+1:]...)
	Folders = append([]string{p}, Folders...)
	if OnSyncWorkspace != nil {
		OnSyncWorkspace(true)
	}
}

// RemoveFolder 从工作区移除文件夹。
func RemoveFolder(p string) {
	out := Folders[:0:0]
	for _, f := range Folders {
		if f != p {
			out = append(out, f)
		}
	}
	primaryChanged := len(out) == 0 || out[0] != Root()
	Folders = out
	if OnSyncWorkspace != nil {
		OnSyncWorkspace(primaryChanged)
	}
}

// ClearWorkspace 关闭整个工作区。
func ClearWorkspace() {
	Folders = nil
	if OnSyncWorkspace != nil {
		OnSyncWorkspace(true)
	}
}

// CloseProject 关闭当前项目。
func CloseProject() {
	if len(Folders) == 0 {
		return
	}
	RemoveFolder(Folders[0])
}

// ─── 工作区菜单 ──

// NewProjectViaDialog 新建项目。
func NewProjectViaDialog() {
	if OnPickFolder == nil {
		return
	}
	parent := OnPickFolder("选择新项目的父目录")
	if parent == "" {
		return
	}
	if OnShowPrompt == nil {
		return
	}
	OnShowPrompt("新建项目", "", func(name string) {
		dir := filepath.Join(parent, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			uiapi.MessageError("创建失败：" + err.Error())
			return
		}
		SetProject(dir)
		uiapi.MessageSuccess("已创建并打开项目「" + name + "」")
	})
}

// SaveWorkspaceMenu 保存工作区。
func SaveWorkspaceMenu() {
	if err := SaveWorkspaceFile(); err != nil {
		uiapi.MessageError("保存工作区失败：" + err.Error())
		return
	}
	uiapi.MessageSuccess("工作区已保存到 .pair/json")
}

// CloseProjectMenu 关闭当前项目。
func CloseProjectMenu() {
	if len(Folders) == 0 {
		uiapi.MessageInfo("当前没有打开的项目")
		return
	}
	CloseProject()
	uiapi.MessageInfo("已关闭项目")
	if OnCloseProject != nil {
		OnCloseProject()
	}
}

// CloseWorkspaceMenu 关闭整个工作区。
func CloseWorkspaceMenu() {
	if len(Folders) == 0 {
		uiapi.MessageInfo("工作区已是空的")
		return
	}
	uiapi.ShowConfirm("关闭工作区", "确定关闭整个工作区？", uiapi.KindWarning,
		func() {
			ClearWorkspace()
			uiapi.MessageInfo("已关闭工作区")
			if OnClearWorkspace != nil {
				OnClearWorkspace()
			}
		})
}

// ShowManager 显示管理工作区文件夹对话框（由 UI 层实现）。
func ShowManager() {
	if OnShowManager != nil {
		OnShowManager()
	}
}
