// Package core 持有 companion 的跨面板共享态/服务 —— 分层地基:面板与工具依赖 core，不再反依赖 main。
// 本文件:工作区路径状态(多根文件夹 + 主根)。纯路径/持久化，不碰 UI/面板/设置(设置耦合的工作区逻辑
// 仍在 main，读 core.Folders/Root + theSettings)。
//
//go:build windows

package core

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

// Folders 工作区的所有文件夹（多根，VS Code 模型）。空=未打开工作区。
var Folders []string

// Root 主文件夹（工作区首个；agent/终端默认在此）。未打开工作区时返回空字符串。
func Root() string {
	if len(Folders) > 0 {
		return Folders[0]
	}
	return ""
}

// ProjectName 工作区显示名：未打开=「未打开文件夹」；单文件夹=文件夹名；多根=「工作区 (N)」。
func ProjectName() string {
	if len(Folders) == 0 {
		return "未打开文件夹"
	}
	if len(Folders) > 1 {
		return fmt.Sprintf("工作区 (%d 个文件夹)", len(Folders))
	}
	return filepath.Base(Root())
}

// IndexOfFolder 返回文件夹在工作区中的下标（不在=-1）。
func IndexOfFolder(p string) int {
	for i, f := range Folders {
		if f == p {
			return i
		}
	}
	return -1
}

// OpenInNewWindow 用新进程（独立新窗口）打开指定文件夹，不影响当前工作区。
func OpenInNewWindow(p string) {
	if p == "" {
		return
	}
	exe, _ := os.Executable()
	if exe == "" {
		exe = os.Args[0]
	}
	c := exec.Command(exe, p)
	// 隐藏子进程控制台窗口（无控制台父进程时 console 程序会自己弹窗）
	if runtime.GOOS == "windows" {
		c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	c.Dir = filepath.Dir(exe) // 与 exe 同目录启动，能找到 libSkiaSharp.dll / fonts
	_ = c.Start()
}

// SaveWorkspaceFile 把工作区文件夹列表存成可移植/可入库的 .pair/core.json（在主文件夹下）。
func SaveWorkspaceFile() error {
	root := Root()
	if root == "" {
		return nil
	}
	dir := filepath.Join(root, ".pair")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]any{"folders": Folders}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "core.json"), data, 0o644)
}
