//go:build stub && darwin

package main

// macOS：goskia 以 -Wl,-rpath,@loader_path 链接，dylib 与主程序同目录即可加载；
// payload 中主程序名为 pair、Skia 库为 libSkiaSharp.dylib。
func mainExeName() string { return "pair" }
