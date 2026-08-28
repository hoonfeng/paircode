//go:build windows

package executil

import (
	"os/exec"
	"syscall"
)

// HideWindow 隐藏子进程控制台窗口（Windows：SysProcAttr.HideWindow；
// 无控制台父进程时 console 程序会自己弹窗）。
func HideWindow(c *exec.Cmd) {
	if c != nil {
		c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
}
