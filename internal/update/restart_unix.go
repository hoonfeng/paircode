//go:build !windows

// restart_unix.go — Linux / macOS 下的脱离式脚本启动（新会话，父进程退出不牵连）。
package update

import (
	"os/exec"
	"syscall"
)

// StartDetached 以新会话（setsid）方式启动重启脚本。
func StartDetached(script string, cfg Config) error {
	cmd := exec.Command("/bin/sh", script)
	cmd.Dir = cfg.InstallDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
