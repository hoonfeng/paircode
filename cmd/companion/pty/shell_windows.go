//go:build windows

package pty

import "os/exec"

// DetectShells 探测 Windows 可用解释器：CMD 必有；PowerShell / PowerShell 7(pwsh) / Git Bash 视安装而定。
func DetectShells() []Shell {
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
