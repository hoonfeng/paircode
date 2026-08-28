//go:build !windows

package executil

import "os/exec"

// HideWindow 非 Windows 平台：无控制台窗口概念，空操作。
func HideWindow(c *exec.Cmd) {}
