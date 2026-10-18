//go:build stub && linux

package main

// Linux：goskia 以 -Wl,-rpath,$ORIGIN 链接，so 与主程序同目录即可被加载；
// payload 中主程序名为 pair、Skia 库为 libSkiaSharp.so。
func mainExeName() string { return "pair" }
