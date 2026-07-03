//go:build windows

package pty

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var (
	modkernel32                           = syscall.NewLazyDLL("kernel32.dll")
	procCreatePseudoConsole               = modkernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole               = modkernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole                = modkernel32.NewProc("ClosePseudoConsole")
	procInitializeProcThreadAttributeList = modkernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute         = modkernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttributeList     = modkernel32.NewProc("DeleteProcThreadAttributeList")
	procCreateProcessW                    = modkernel32.NewProc("CreateProcessW")
)

const (
	procThreadAttributePseudoConsole = 0x00020016
	extendedStartupInfoPresent       = 0x00080000
	startfUseStdHandles              = 0x00000100
	infiniteWait                     = 0xFFFFFFFF
)

// startupInfoEx = STARTUPINFOEX：标准 STARTUPINFO + 属性列表指针。
type startupInfoEx struct {
	syscall.StartupInfo
	AttributeList *byte
}

type windowsPty struct {
	hpc      uintptr // HPCON 伪控制台句柄
	inW      *os.File
	outR     *os.File
	hProcess syscall.Handle
	hThread  syscall.Handle
}

func coordSize(cols, rows int) uintptr {
	return uintptr(uint32(uint16(int16(cols))) | uint32(uint16(int16(rows)))<<16)
}

// Start 起一个 ConPTY 会话跑 shell：伪控制台给 shell 真 tty → 输出行缓冲可流式 + 持久会话 + 交互式程序。
//
// 遵循微软官方 ConPTY 示例模式：
// 1) 创建两条管道（stdin + stdout）
// 2) 用管道端创建伪控制台
// 3) 在 STARTUPINFOEX 中设置 PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE
// 4) 子进程的 std 句柄设到 pipe 端（STARTF_USESTDHANDLES），ConPTY 通过它们重定向 I/O
func Start(sh Shell, dir string, cols, rows int) (PTY, error) {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	var sa syscall.SecurityAttributes
	sa.Length = uint32(unsafe.Sizeof(sa))
	sa.InheritHandle = 1

	// stdin 管道：conIn（读端→给 ConPTY 和子进程）/ ptyIn（写端→父进程写键盘）
	var conIn, ptyIn syscall.Handle
	if err := syscall.CreatePipe(&conIn, &ptyIn, &sa, 0); err != nil {
		return nil, err
	}
	// stdout 管道：ptyOut（读端→父进程读输出）/ conOut（写端→给 ConPTY 和子进程）
	var ptyOut, conOut syscall.Handle
	if err := syscall.CreatePipe(&ptyOut, &conOut, &sa, 0); err != nil {
		syscall.CloseHandle(conIn)
		syscall.CloseHandle(ptyIn)
		return nil, err
	}

	// ── 创建伪控制台 ──
	var hpc uintptr
	r1, _, _ := procCreatePseudoConsole.Call(coordSize(cols, rows), uintptr(conIn), uintptr(conOut), 0, uintptr(unsafe.Pointer(&hpc)))
	if r1 != 0 {
		syscall.CloseHandle(conIn)
		syscall.CloseHandle(conOut)
		syscall.CloseHandle(ptyIn)
		syscall.CloseHandle(ptyOut)
		return nil, fmt.Errorf("CreatePseudoConsole 失败: 0x%x", r1)
	}
	// 不关闭 conIn/conOut——它们将作为子进程的 std 句柄传给 CreateProcess。
	// 父进程侧使用的句柄是 ptyIn（写键盘输入）和 ptyOut（读 shell 输出）。
	inW := os.NewFile(uintptr(ptyIn), "conpty-in")
	outR := os.NewFile(uintptr(ptyOut), "conpty-out")

	// ── 属性列表（挂上 PSEUDOCONSOLE） ──
	var attrSize uintptr
	procInitializeProcThreadAttributeList.Call(0, 1, 0, uintptr(unsafe.Pointer(&attrSize)))
	attrList := make([]byte, attrSize)
	if r, _, e := procInitializeProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&attrList[0])), 1, 0, uintptr(unsafe.Pointer(&attrSize))); r == 0 {
		procClosePseudoConsole.Call(hpc)
		inW.Close()
		outR.Close()
		return nil, fmt.Errorf("InitializeProcThreadAttributeList: %v", e)
	}
	if r, _, e := procUpdateProcThreadAttribute.Call(uintptr(unsafe.Pointer(&attrList[0])), 0, procThreadAttributePseudoConsole, hpc, unsafe.Sizeof(hpc), 0, 0); r == 0 {
		procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&attrList[0])))
		procClosePseudoConsole.Call(hpc)
		inW.Close()
		outR.Close()
		return nil, fmt.Errorf("UpdateProcThreadAttribute: %v", e)
	}

	// ── 子进程 STARTUPINFO ──
	// 微软 ConPTY 示例模式：
	// - 子进程的 stdin/stdout 指向与 ConPTY 共享的同一管道端
	// - 设置 STARTF_USESTDHANDLES，显式指定句柄
	// - bInheritHandles=FALSE（句柄通过 STARTUPINFO 显式传递，非继承）
	var si startupInfoEx
	si.StartupInfo.Cb = uint32(unsafe.Sizeof(si))
	si.StartupInfo.Flags = startfUseStdHandles
	si.StartupInfo.StdInput = conIn   // 读端：子进程读键盘输入
	si.StartupInfo.StdOutput = conOut // 写端：子进程写终端输出
	si.StartupInfo.StdErr = conOut
	si.AttributeList = &attrList[0]

	cmdline := quoteCmd(sh.Path)
	if len(sh.Args) > 0 {
		cmdline += " " + strings.Join(sh.Args, " ")
	}
	cmdlinePtr, _ := syscall.UTF16PtrFromString(cmdline)
	var dirPtr *uint16
	if dir != "" {
		dirPtr, _ = syscall.UTF16PtrFromString(dir)
	}

	var pi syscall.ProcessInformation
	r4, _, e4 := procCreateProcessW.Call(
		0, uintptr(unsafe.Pointer(cmdlinePtr)), 0, 0, 0,
		extendedStartupInfoPresent, 0, uintptr(unsafe.Pointer(dirPtr)),
		uintptr(unsafe.Pointer(&si)), uintptr(unsafe.Pointer(&pi)))
	procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&attrList[0])))
	// CreateProcess 已复制子进程侧的句柄，父进程侧可以安全关闭
	syscall.CloseHandle(conIn)
	syscall.CloseHandle(conOut)
	if r4 == 0 {
		procClosePseudoConsole.Call(hpc)
		inW.Close()
		outR.Close()
		return nil, fmt.Errorf("CreateProcess: %v", e4)
	}

	return &windowsPty{hpc: hpc, inW: inW, outR: outR, hProcess: pi.Process, hThread: pi.Thread}, nil
}

func (p *windowsPty) Read(b []byte) (int, error)  { return p.outR.Read(b) }
func (p *windowsPty) Write(b []byte) (int, error) { return p.inW.Write(b) }

func (p *windowsPty) Resize(cols, rows int) error {
	procResizePseudoConsole.Call(p.hpc, coordSize(cols, rows))
	return nil
}

func (p *windowsPty) Wait() error {
	if p.hProcess != 0 {
		syscall.WaitForSingleObject(p.hProcess, infiniteWait)
	}
	return nil
}

func (p *windowsPty) Close() error {
	if p.hpc != 0 {
		procClosePseudoConsole.Call(p.hpc) // 关伪控制台→shell 收到 EOF/退出
		p.hpc = 0
	}
	p.inW.Close()
	p.outR.Close()
	if p.hProcess != 0 {
		syscall.TerminateProcess(p.hProcess, 0)
		syscall.CloseHandle(p.hProcess)
		p.hProcess = 0
	}
	if p.hThread != 0 {
		syscall.CloseHandle(p.hThread)
		p.hThread = 0
	}
	return nil
}

func quoteCmd(s string) string {
	if strings.ContainsAny(s, " \t") {
		return `"` + s + `"`
	}
	return s
}
