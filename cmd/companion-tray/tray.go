// Windows 系统托盘实现 —— 使用 Win32 API Shell_NotifyIcon
//
//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Win32 常量
const (
	WM_USER          = 0x0400
	WM_TRAYICON      = WM_USER + 1
	WM_DESTROY       = 0x0002
	WM_COMMAND       = 0x0111
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP     = 0x0205

	NIM_ADD    = 0
	NIM_MODIFY = 1
	NIM_DELETE = 2

	NIF_MESSAGE = 1
	NIF_ICON    = 2
	NIF_TIP     = 4

	// 菜单项 ID
	CMD_OPEN        = 1001
	CMD_START_STOP  = 1002
	CMD_OPEN_BROWSER = 1003
	CMD_SETTINGS    = 1004
	CMD_AUTOSTART   = 1005
	CMD_QUIT        = 1006

	MF_STRING    = 0
	MF_CHECKED   = 8
	MF_SEPARATOR = 0x800

	TPM_RIGHTBUTTON = 2
	TPM_BOTTOMALIGN = 0x20
	TPM_RETURNCMD   = 0x0100

	SW_SHOW = 5
	SW_HIDE = 0

	IDI_APPLICATION = 32512
)

// NOTIFYICONDATA 的 Windows 结构（x64）
type NOTIFYICONDATA struct {
	CbSize           uint32
	HWnd             windows.HWND
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            windows.Handle
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     windows.Handle
}

// TrayManager 系统托盘管理器
type TrayManager struct {
	hwnd      windows.HWND
	hinst     windows.Handle
	hIcon     windows.Handle
	nid       *NOTIFYICONDATA
	port      int
	running   bool
	destroyed bool
	mu        sync.Mutex
	events    chan TrayEvent
}

type TrayEvent int

const (
	EventToggleServer TrayEvent = iota
	EventOpenSettings
	EventQuit
)

var (
	user32         = windows.NewLazySystemDLL("user32.dll")
	shell32        = windows.NewLazySystemDLL("shell32.dll")
	kernel32       = windows.NewLazySystemDLL("kernel32.dll")
	advapi32       = windows.NewLazySystemDLL("advapi32.dll")

	procCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	procDestroyMenu            = user32.NewProc("DestroyMenu")
	procAppendMenuW            = user32.NewProc("AppendMenuW")
	procTrackPopupMenu         = user32.NewProc("TrackPopupMenu")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procGetCursorPos           = user32.NewProc("GetCursorPos")
	procLoadIconW              = user32.NewProc("LoadIconW")
	procDestroyIcon            = user32.NewProc("DestroyIcon")
	procShellNotifyIconW       = shell32.NewProc("Shell_NotifyIconW")
	procExtractIconW           = shell32.NewProc("ExtractIconW")
	procRegOpenKeyExW          = advapi32.NewProc("RegOpenKeyExW")
	procRegSetValueExW         = advapi32.NewProc("RegSetValueExW")
	procRegDeleteValueW        = advapi32.NewProc("RegDeleteValueW")
	procRegCloseKey            = advapi32.NewProc("RegCloseKey")
)

// POINT 结构
type POINT struct {
	X, Y int32
}

func NewTrayManager() *TrayManager {
	return &TrayManager{
		port:    cfg.Port,
		events:  make(chan TrayEvent, 10),
		running: true,
	}
}

// Run 托盘消息循环（必须在独立线程运行）
func (tm *TrayManager) Run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 模块句柄
	mod, err := getModuleHandle()
	if err != nil {
		log.Printf("获取模块句柄失败: %v", err)
		return
	}
	tm.hinst = mod

	// 注册窗口类
	className := []uint16{'P', 'a', 'i', 'r', 'C', 'o', 'd', 'e', 'T', 'r', 'a', 'y', 0}
	wc := struct {
		style         uint32
		lpfnWndProc   uintptr
		cbClsExtra    int32
		cbWndExtra    int32
		hInstance     windows.Handle
		hIcon         windows.Handle
		hCursor       windows.Handle
		hbrBackground windows.Handle
		lpszMenuName  *uint16
		lpszClassName *uint16
	}{
		lpfnWndProc:   windows.NewCallback(tm.wndProc),
		hInstance:     tm.hinst,
		lpszClassName: &className[0],
	}

	regClass := user32.NewProc("RegisterClassW")
	ret, _, _ := regClass.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		log.Println("注册窗口类失败")
		return
	}

	// 创建隐藏窗口
	create := user32.NewProc("CreateWindowExW")
	hwnd, _, _ := create.Call(
		0,
		uintptr(unsafe.Pointer(&className[0])),
		uintptr(unsafe.Pointer(&[]uint16{'T', 'r', 'a', 'y', 0}[0])),
		0, 0, 0, 0, 0,
		0, 0, uintptr(tm.hinst), 0,
	)
	if hwnd == 0 {
		log.Println("创建隐藏窗口失败")
		return
	}
	tm.hwnd = windows.HWND(hwnd)

	// 加载图标
	tm.loadIcon()

	// 创建 NOTIFYICONDATA
	nid := &NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		HWnd:             tm.hwnd,
		UID:              1,
		UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		UCallbackMessage: WM_TRAYICON,
		HIcon:            tm.hIcon,
	}
	copy(nid.SzTip[:], syscall.StringToUTF16("PairCode Web IDE"))
	tm.nid = nid

	// 添加图标
	if tm.callShellNotify(NIM_ADD) != 0 {
		log.Println("托盘图标添加成功")
	} else {
		log.Println("托盘图标添加失败（可能已有）")
	}

	// 消息循环
	var msg struct {
		hwnd    uintptr
		message uint32
		wParam  uintptr
		lParam  uintptr
		time    uint32
		pt      POINT
	}
	getMsg := user32.NewProc("GetMessageW")

	for tm.running {
		ret, _, _ := getMsg.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 { // WM_QUIT
			break
		}
		user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&msg)))
		user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&msg)))
	}

	tm.Destroy()
}

func (tm *TrayManager) wndProc(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		switch lParam {
		case WM_LBUTTONDBLCLK:
			openBrowser(tm.port)
		case WM_RBUTTONUP:
			tm.showMenu()
		}
		return 0

	case WM_COMMAND:
		switch wParam {
		case CMD_OPEN:
			openBrowser(tm.port)
		case CMD_START_STOP:
			tm.events <- EventToggleServer
		case CMD_OPEN_BROWSER:
			openBrowser(tm.port)
		case CMD_SETTINGS:
			tm.events <- EventOpenSettings
		case CMD_AUTOSTART:
			toggleAutoStart()
		case CMD_QUIT:
			tm.events <- EventQuit
		}
		return 0

	case WM_DESTROY:
		postQuit := user32.NewProc("PostQuitMessage")
		postQuit.Call(0)
		return 0
	}

	defWnd := user32.NewProc("DefWindowProcW")
	ret, _, _ := defWnd.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func (tm *TrayManager) showMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	items := []struct {
		id   uint32
		text string
		flags uint32
		checked bool
	}{
		{CMD_OPEN, "打开浏览器(&O)", MF_STRING, false},
		{0, "", MF_SEPARATOR, false},
		{CMD_START_STOP, "启动/停止服务器(&S)", MF_STRING, false},
		{CMD_OPEN_BROWSER, "在浏览器中打开(&B)", MF_STRING, false},
		{0, "", MF_SEPARATOR, false},
		{CMD_SETTINGS, "设置(&C)", MF_STRING, false},
		{CMD_AUTOSTART, "开机自启(&A)", MF_STRING, cfg.AutoStart},
		{0, "", MF_SEPARATOR, false},
		{CMD_QUIT, "退出(&Q)", MF_STRING, false},
	}

	for _, it := range items {
		if it.text == "" {
			procAppendMenuW.Call(menu, MF_SEPARATOR, 0, 0)
			continue
		}
		flags := uintptr(it.flags)
		if it.checked {
			flags |= MF_CHECKED
		}
		u16 := syscall.StringToUTF16Ptr(it.text)
		procAppendMenuW.Call(menu, flags, uintptr(it.id), uintptr(unsafe.Pointer(u16)))
	}

	// 位置
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(uintptr(tm.hwnd))
	procTrackPopupMenu.Call(menu, TPM_RIGHTBUTTON|TPM_BOTTOMALIGN|TPM_RETURNCMD,
		uintptr(pt.X), uintptr(pt.Y), 0, uintptr(tm.hwnd), 0)
}

func (tm *TrayManager) callShellNotify(action uint32) uintptr {
	ret, _, _ := procShellNotifyIconW.Call(
		uintptr(action),
		uintptr(unsafe.Pointer(tm.nid)),
	)
	return ret
}

func (tm *TrayManager) loadIcon() {
	// 加载系统默认图标
	icon, _, _ := procLoadIconW.Call(0, IDI_APPLICATION)
	if icon != 0 {
		tm.hIcon = windows.Handle(icon)
		return
	}
	// 回退：从 exe 提取
	exe, _ := os.Executable()
	u16 := syscall.StringToUTF16Ptr(exe)
	hicon, _, _ := procExtractIconW.Call(uintptr(tm.hinst), uintptr(unsafe.Pointer(u16)), 0)
	if hicon != 0 {
		tm.hIcon = windows.Handle(hicon)
	}
}

// Destroy 公开的清理方法
func (tm *TrayManager) Destroy() {
	tm.mu.Lock()
	if tm.destroyed {
		tm.mu.Unlock()
		return
	}
	tm.destroyed = true
	tm.mu.Unlock()

	if tm.nid != nil {
		tm.callShellNotify(NIM_DELETE)
	}
	if tm.hIcon != 0 {
		procDestroyIcon.Call(uintptr(tm.hIcon))
	}
	if tm.hwnd != 0 {
		user32.NewProc("DestroyWindow").Call(uintptr(tm.hwnd))
	}
}

func (tm *TrayManager) UpdatePort(port int) {
	tm.port = port
}

func (tm *TrayManager) Events() <-chan TrayEvent {
	return tm.events
}

// --- 辅助函数 ---

func getModuleHandle() (windows.Handle, error) {
	mod := kernel32.NewProc("GetModuleHandleW")
	ret, _, _ := mod.Call(0)
	if ret == 0 {
		return 0, fmt.Errorf("GetModuleHandleW failed")
	}
	return windows.Handle(ret), nil
}

func openBrowser(port int) {
	url := fmt.Sprintf("http://localhost:%d", port)
	exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func toggleAutoStart() {
	if cfg.AutoStart {
		disableAutoStart()
		cfg.AutoStart = false
	} else {
		enableAutoStart()
		cfg.AutoStart = true
	}
	cfg.Save()
}

func enableAutoStart() {
	keyPath := syscall.StringToUTF16Ptr(`Software\Microsoft\Windows\CurrentVersion\Run`)
	var hKey windows.Handle
	ret, _, _ := procRegOpenKeyExW.Call(
		windows.HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(keyPath)),
		0, windows.KEY_SET_VALUE,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if ret != 0 {
		log.Printf("打开注册表失败: error=%d", ret)
		return
	}
	defer procRegCloseKey.Call(uintptr(hKey))

	exe, _ := os.Executable()
	absPath, _ := filepath.Abs(exe)
	value := syscall.StringToUTF16Ptr(absPath)
	// 计算 UTF16 字节数（含终止 null）
	byteLen := (len(absPath) + 1) * 2

	procRegSetValueExW.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("PairCodeTray"))),
		0, windows.REG_SZ,
		uintptr(unsafe.Pointer(value)), uintptr(byteLen),
	)
	log.Println("开机自启已启用")
}

func disableAutoStart() {
	keyPath := syscall.StringToUTF16Ptr(`Software\Microsoft\Windows\CurrentVersion\Run`)
	var hKey windows.Handle
	ret, _, _ := procRegOpenKeyExW.Call(
		windows.HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(keyPath)),
		0, windows.KEY_SET_VALUE,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if ret != 0 {
		log.Printf("打开注册表失败: error=%d", ret)
		return
	}
	defer procRegCloseKey.Call(uintptr(hKey))

	procRegDeleteValueW.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("PairCodeTray"))),
	)
	log.Println("开机自启已禁用")
}
