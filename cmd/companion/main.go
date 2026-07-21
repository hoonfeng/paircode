// PairCode IDE Web 服务器入口。
// 直接启动 Web 服务（不再有 goui 启动面板）。
//
//go:build windows

package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

// 编译版本号（由 packager 通过 -ldflags=-X main.version=<version> 注入）
var version = "v1.1.1"

func main() {
	// ★ 全局 panic recovery — 捕获所有未捕获的 panic，防止进程静默崩溃
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[FATAL] 未捕获的异常: %v", r)
			// 输出完整堆栈
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			log.Printf("[FATAL] 堆栈:\n%s", buf[:n])
			// 确保日志刷盘后再退出
			log.Printf("[FATAL] 进程因未捕获异常终止")
			os.Exit(1)
		}
	}()

	port := InitCore()

	log.Printf("[main] 正在启动 Web 服务器 (端口 %d)...", port)
	if err := StartWebServer(port); err != nil {
		log.Fatalf("[main] 启动失败: %v", err)
	}
	log.Printf("[main] Web 服务器已启动，请打开 http://localhost:%d", port)

	// 等待退出信号
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("[main] 收到退出信号，正在关闭...")
	StopWebServer()
	log.Println("[main] 已退出。")
}
