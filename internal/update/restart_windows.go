//go:build windows

// restart_windows.go — Windows 下的脱离式脚本启动（父进程退出后脚本继续运行）。
package update

import (
	"os/exec"
	"syscall"
)

// StartDetached 以脱离进程组的方式启动重启脚本。
//
// DETACHED_PROCESS：新进程无控制台，父进程退出不影响；
// CREATE_NEW_PROCESS_GROUP：新进程组，避免被父进程的 Ctrl+C / 控制台事件波及。
func StartDetached(script string, cfg Config) error {
	cmd := exec.Command("cmd", "/c", script)
	cmd.Dir = cfg.InstallDir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000008 | 0x00000200, // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
		HideWindow:    true,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
