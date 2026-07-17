// PairCode IDE Web 服务器入口。
// 直接启动 Web 服务（不再有 goui 启动面板）。
//
//go:build windows

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// 编译版本号（由 packager 通过 -ldflags=-X main.version=<version> 注入）
var version = "v1.0.6"

func main() {
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
