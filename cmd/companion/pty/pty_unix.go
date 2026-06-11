//go:build linux || darwin

package pty

import "fmt"

// Start（Unix）：TODO 用 openpty/forkpty 实现真伪终端（pts 即 tty，无缓冲问题）。
// 当前为桩——解释器探测（DetectShells）已就绪，PTY 启动待实现。
func Start(sh Shell, dir string, cols, rows int) (PTY, error) {
	return nil, fmt.Errorf("pty: Unix forkpty 尚未实现（Windows ConPTY 已就绪，Unix 待接）")
}
