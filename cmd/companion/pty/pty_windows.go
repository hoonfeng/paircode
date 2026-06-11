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
func Start(sh Shell, dir string, cols, rows int) (PTY, error) {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	// 用 raw CreatePipe（同步、字节模式、可继承句柄）——os.Pipe 的句柄能读到初始序列，
	// 但 conhost 绑不上 shell 的 stdout（已知可用 Go ConPTY 实现都用 CreatePipe + 可继承）。
	var sa syscall.SecurityAttributes
	sa.Length = uint32(unsafe.Sizeof(sa))
	sa.InheritHandle = 1
	var conIn, ptyIn, ptyOut, conOut syscall.Handle
	if err := syscall.CreatePipe(&conIn, &ptyIn, &sa, 0); err != nil { // 输入：conIn(读,给 ConPTY) / ptyIn(写,留)
		return nil, err
	}
	if err := syscall.CreatePipe(&ptyOut, &conOut, &sa, 0); err != nil { // 输出：ptyOut(读,留) / conOut(写,给 ConPTY)
		syscall.CloseHandle(conIn)
		syscall.CloseHandle(ptyIn)
		return nil, err
	}

	var hpc uintptr
	r1, _, _ := procCreatePseudoConsole.Call(coordSize(cols, rows), uintptr(conIn), uintptr(conOut), 0, uintptr(unsafe.Pointer(&hpc)))
	syscall.CloseHandle(conIn) // 伪控制台已复制 console 侧句柄，关掉本端副本
	syscall.CloseHandle(conOut)
	if r1 != 0 {
		syscall.CloseHandle(ptyIn)
		syscall.CloseHandle(ptyOut)
		return nil, fmt.Errorf("CreatePseudoConsole 失败: 0x%x", r1)
	}
	inW := os.NewFile(uintptr(ptyIn), "conpty-in")   // 写键盘
	outR := os.NewFile(uintptr(ptyOut), "conpty-out") // 读输出

	// 进程线程属性列表（带 PSEUDOCONSOLE）：先问大小，再初始化，再 Update 挂上 hpc。
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

	var si startupInfoEx
	si.StartupInfo.Cb = uint32(unsafe.Sizeof(si))
	// 关键：置 STARTF_USESTDHANDLES（std 句柄留空）→ 子进程不继承父控制台的 std 句柄，
	// 伪控制台属性才能接管子 shell 的 I/O（否则 shell 输出跑去父控制台——之前的 bug 根因）。
	si.StartupInfo.Flags = startfUseStdHandles
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
