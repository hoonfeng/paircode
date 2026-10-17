//go:build linux || darwin

package pty

// pty_unix.go — Linux / macOS 伪终端实现。
//
// 2026-10 之前这里是桩（Start 直接返回 "Unix forkpty 尚未实现"），
// 导致 Linux/macOS 上内置终端完全不可用（WebSocket 连接后立即 error）。
// 现用 creack/pty 实现真正的 pts：openpty 语义 + setsid/setctty，
// 与 Windows ConPTY 路径同构（读写 master 端、resize、等待退出、关闭时
// 终止整个前台进程组）。
//
// 为什么不是裸管道：见 pty.go 包注释——裸管道下 shell 输出全缓冲，
// 交互式程序（vim/htop）无法工作，且没有 tty 时很多命令行为不同。

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// unixPty 一条 Unix 伪终端会话：master 端是文件描述符，slave 端交给 shell。
type unixPty struct {
	f         *os.File
	cmd       *exec.Cmd
	closeOnce sync.Once
	closeErr  error
}

// Start 在 dir 下以默认尺寸启动 sh，返回可读写/resize 的伪终端。
func Start(sh Shell, dir string, cols, rows int) (PTY, error) {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	if sh.Path == "" {
		sh = fallbackShell()
	}

	cmd := exec.Command(sh.Path, sh.Args...)
	if dir != "" {
		cmd.Dir = dir
	}
	// TERM 决定 shell 里程序是否输出颜色/光标控制序列。xterm.js 兼容
	// xterm-256color；若用户在外部已设 TERM 则尊重之（tmux/screen 场景）。
	cmd.Env = os.Environ()
	if os.Getenv("TERM") == "" {
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	}
	// 交互式 shell 的提示符/行为更接近真实终端。
	if len(sh.Args) == 0 {
		cmd.Args = append(cmd.Args, "-i")
	}

	f, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
	if err != nil {
		return nil, fmt.Errorf("pty: 启动 %s 失败: %w", strings.Join(cmd.Args, " "), err)
	}
	return &unixPty{f: f, cmd: cmd}, nil
}

func (p *unixPty) Read(b []byte) (int, error)  { return p.f.Read(b) }
func (p *unixPty) Write(b []byte) (int, error) { return p.f.Write(b) }

// Resize 调整 pts 窗口尺寸（TIOCSWINSZ），shell/前台程序收到 SIGWINCH。
func (p *unixPty) Resize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	return pty.Setsize(p.f, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

// Wait 阻塞至 shell 退出。
func (p *unixPty) Wait() error { return p.cmd.Wait() }

// Close 终止 shell 及其前台进程组，然后关闭 master。
//
// creack/pty 以 Setsid 启动子进程 → 子进程是新会话/进程组的组长，
// 因此 -pid 即整个 shell 进程组。只 Kill 主进程会留下子进程（如 vim、
// 长时间运行的命令）占着 pts；先对进程组发 SIGHUP（终端关闭语义），
// 再 Kill 兜底。
func (p *unixPty) Close() error {
	p.closeOnce.Do(func() {
		if p.cmd.Process != nil {
			pid := p.cmd.Process.Pid
			_ = syscall.Kill(-pid, syscall.SIGHUP)
			_ = p.cmd.Process.Kill()
		}
		p.closeErr = p.f.Close()
	})
	return p.closeErr
}
