// PairCode IDE Web 服务器入口。
// 直接启动 Web 服务（不再有 goui 启动面板）。
//
//go:build windows

package main

import (
	"log"
	"os"
	"runtime"
)

// 编译版本号（由 packager 通过 -ldflags=-X main.version=<version> 注入）
// 也用于 /api/system/info 返回给前端 About 弹窗展示。
var version = "v1.1.6"

// getCwd 返回当前工作目录，失败时返回 "?"。
func getCwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "?"
	}
	return d
}

func main() {
	// ★ 全局 panic recovery — 捕获所有未捕获的 panic，防止进程静默崩溃
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[FATAL] 未捕获的异常: %v", r)
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			log.Printf("[FATAL] 堆栈:\n%s", buf[:n])
			log.Printf("[FATAL] 进程因未捕获异常终止")
			os.Exit(1)
		}
	}()

	port := InitCore()

	log.Printf("[main] PairCode IDE %s 启动中 (%s, %s/%s)", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	log.Printf("[main] 工作目录: %s", getCwd())

	log.Printf("[main] 正在启动 Web 服务器 (端口 %d)...", port)
	if err := StartWebServer(port); err != nil {
		log.Fatalf("[main] 启动失败: %v", err)
	}
	log.Printf("[main] 已启动，请打开 http://localhost:%d", port)
	// 永久阻塞，直到用户关闭命令窗口或 kill 进程
	select {}
}
