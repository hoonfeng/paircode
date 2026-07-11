//go:build windows && !webonly

package ui

import (
	"fmt"
	"os/exec"
	"strings"
)

// OpenFileDialog 弹出 Windows 文件打开对话框，返回选择的文件路径。
// 用户取消时返回空字符串。
func OpenFileDialog(title, filter string) string {
	return runFileDialog(true, title, filter, "")
}

// SaveFileDialog 弹出 Windows 文件保存对话框，返回选择的文件路径。
// 用户取消时返回空字符串。
func SaveFileDialog(title, filter, defaultName string) string {
	return runFileDialog(false, title, filter, defaultName)
}

func runFileDialog(open bool, title, filter, defaultName string) string {
	// 安全转义 PowerShell 字符串中的单引号
	esc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }

	var script string
	if open {
		script = fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms
$f = New-Object System.Windows.Forms.OpenFileDialog
$f.Title = '%s'
$f.Filter = '%s'
$f.CheckFileExists = $true
$r = $f.ShowDialog()
if ($r -eq [System.Windows.Forms.DialogResult]::OK) { $f.FileName } else { '' }`, esc(title), esc(filter))
	} else {
		script = fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms
$f = New-Object System.Windows.Forms.SaveFileDialog
$f.Title = '%s'
$f.Filter = '%s'
$f.FileName = '%s'
$f.OverwritePrompt = $true
$r = $f.ShowDialog()
if ($r -eq [System.Windows.Forms.DialogResult]::OK) { $f.FileName } else { '' }`, esc(title), esc(filter), esc(defaultName))
	}

	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
