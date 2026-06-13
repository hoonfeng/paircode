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
	out := []Shell{{Name: "CMD", Path: "cmd", Args: nil}} // cmd 总在
	if p, err := exec.LookPath("powershell"); err == nil {
		out = append(out, Shell{Name: "PowerShell", Path: p, Args: []string{"-NoLogo", "-NoProfile"}})
	}
	if p, err := exec.LookPath("pwsh"); err == nil { // PowerShell 7+
		out = append(out, Shell{Name: "PowerShell 7", Path: p, Args: []string{"-NoLogo", "-NoProfile"}})
	}
	if p, err := exec.LookPath("bash"); err == nil { // Git Bash / WSL bash
		out = append(out, Shell{Name: "Git Bash", Path: p, Args: []string{"-i"}})
	}
	return out
}

func fallbackShell() Shell { return Shell{Name: "CMD", Path: "cmd"} }
