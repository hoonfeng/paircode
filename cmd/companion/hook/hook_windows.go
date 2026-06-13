//go:build windows

package hook

import (
	"os/exec"
	"syscall"
)

func init() {
	hideWindow = func(cmd *exec.Cmd) {
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		cmd.SysProcAttr.HideWindow = true
	}
}
