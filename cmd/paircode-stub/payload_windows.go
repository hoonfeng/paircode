//go:build stub && windows

package main

// Windows：PE 导入表在进程启动前解析 libSkiaSharp.dll，二者必须同目录；
// stub 的 payload 中主程序名为 pair.exe、Skia 库为 libSkiaSharp.dll。
func mainExeName() string { return "pair.exe" }
