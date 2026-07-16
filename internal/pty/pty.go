// Package pty 跨平台伪终端（真终端模拟器的进程侧）。
//
// 为什么要伪终端而不是裸管道：cmd.exe 等控制台程序在 stdout 是「管道」时会全缓冲输出，
// 持久 shell 不退出→永不刷新→无输出流（裸管道版已证伪）。伪终端给 shell 一个真「console/tty」，
// 程序按行缓冲→可流式，且支持交互式程序（vim/htop）、cd/env 持久、resize。
//
// 平台实现（各自 build-tag）：
//   - Windows：ConPTY（CreatePseudoConsole + 带 PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE 的 CreateProcess）
//   - Linux/macOS：Unix PTY（openpty/forkpty，pts 即 tty，无缓冲问题）
//
// 上层（companion 终端）拿到 PTY 后：写命令/按键到它、持续读它的 VT 字节流 → 经 VT 解析器更新
// cell 网格 → 在 goui 里渲染。多实例 = 每个标签一个 PTY 会话。
package pty

import "io"

// PTY 一个伪终端会话：可读写（VT 字节流）、可改尺寸、可等退出、可关闭。
type PTY interface {
	io.ReadWriteCloser
	// Resize 通知伪终端新的列/行数（窗口尺寸变化时调，程序据此重排）。
	Resize(cols, rows int) error
	// Wait 阻塞至 shell 进程退出。
	Wait() error
}

// Shell 一个可用的解释器/shell（探测得到）。
type Shell struct {
	Name string   // 显示名：CMD / PowerShell / bash / zsh …
	Path string   // 可执行文件路径或命令名
	Args []string // 启动参数（交互式：通常 -i 或 -NoProfile 等）
}

// DefaultShell 本机首选 shell（DetectShells 的第一个）。
func DefaultShell() Shell {
	if s := DetectShells(); len(s) > 0 {
		return s[0]
	}
	return fallbackShell() // 各平台兜底（不应到达）
}

// DetectShells（各平台实现）探测本机可用的解释器，按优先级返回。
// ShellByName 据名字找已探测到的 shell（找不到→默认）。
func ShellByName(name string) Shell {
	for _, s := range DetectShells() {
		if s.Name == name {
			return s
		}
	}
	return DefaultShell()
}
