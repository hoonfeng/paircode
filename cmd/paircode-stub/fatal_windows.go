//go:build stub && windows

package main

import (
	"log"
	"syscall"
	"unsafe"
)

// fatal 在 Windows GUI 子系统下没有控制台，用 MessageBox 把错误直接呈现给用户，
// 否则双击启动失败时用户看不到任何信息。
func fatal(msg string) {
	log.Printf("[stub] %s", msg)
	const mbIconError = 0x00000010
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	title, _ := syscall.UTF16PtrFromString("PairCode 启动失败")
	body, _ := syscall.UTF16PtrFromString(msg)
	proc.Call(0, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(title)), mbIconError)
}
