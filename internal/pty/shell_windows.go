//go:build windows

package pty

import (
	"os/exec"
	"sync"
)

var (
	detectShellsOnce sync.Once
	detectedShells   []Shell
)

// DetectShells 探测 Windows 可用解释器：CMD 必有；PowerShell / PowerShell 7(pwsh) / Git Bash 视安装而定。
// 结果缓存，仅首次调用时探测（shell 列表运行时不会变化）。
func DetectShells() []Shell {
	detectShellsOnce.Do(func() {
		detectedShells = detectShellsUncached()
	})
	return detectedShells
}

func detectShellsUncached() []Shell {
	// ★ UTF-8 代码页：Windows cmd 默认 GBK(cp936) 输出，xterm 按 UTF-8 解码
	// → 中文乱码。启动时注入 UTF-8（浏览器终端/Windows Terminal 标准）：
	//   - CMD:  /K chcp 65001>nul（>nul 吞掉 chcp 回显，不污染首行）
	//   - PowerShell: [Console]::OutputEncoding=UTF8（PS5.1 控制台输出编码）
	//   - Git Bash: 原生 UTF-8，无需处理
	// ★ banner 补充：cmd 的 stdout 是 ConPTY 匿名管道（非控制台句柄）时，
	//   启动 banner（Microsoft Windows [版本...] + 版权行）不输出（cmd 的
	//   banner 走控制台专用路径，管道下静默丢弃，实测真实终端/浏览器参照
	//   均如此）。为对齐真实 cmd 体验，用 ver（动态输出版本行）+ echo 版权行
	//   在 chcp 之后手动补打（chcp 在 banner 前执行避免切换代码页清屏干扰）。
	out := []Shell{{Name: "CMD", Path: "cmd",
		Args: []string{"/q", "/d", "/K", "chcp 65001>nul & echo. & ver & echo (c) Microsoft Corporation。保留所有权利。 & echo."}}} // cmd 总在
	if p, err := exec.LookPath("powershell"); err == nil {
		out = append(out, Shell{Name: "PowerShell", Path: p,
			Args: []string{"-NoLogo", "-NoProfile", "-NoExit", "-Command", "[Console]::OutputEncoding=[System.Text.Encoding]::UTF8"}})
	}
	if p, err := exec.LookPath("pwsh"); err == nil { // PowerShell 7+
		out = append(out, Shell{Name: "PowerShell 7", Path: p,
			Args: []string{"-NoLogo", "-NoProfile", "-NoExit", "-Command", "[Console]::OutputEncoding=[System.Text.Encoding]::UTF8"}})
	}
	if p, err := exec.LookPath("bash"); err == nil { // Git Bash / WSL bash
		out = append(out, Shell{Name: "Git Bash", Path: p, Args: []string{"-i"}})
	}
	return out
}

func fallbackShell() Shell { return Shell{Name: "CMD", Path: "cmd"} }
