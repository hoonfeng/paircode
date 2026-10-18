//go:build gui

// GUI 一体化入口（build tag: gui）。
//
// 构建方式（在 gou-ide 根目录，需 CGO 工具链与 goskia 原生库）：
//
//	CGO_ENABLED=1 GOWORK=go.work.gui go build -tags gui -o release/pair.exe ./cmd/companion
//
// 进程模型：Web 服务（本进程）与桌面窗口（wb-ui 引擎）同进程。启动顺序是
// 「先起 HTTP 服务、再开窗口」——窗口加载的页面、子资源、/api、/ws 全部由
// 本进程自身提供，不再拉起 pair.exe 子进程。
package main

import (
	"fmt"
	"log"

	"wb-ui/ideapp"
)

// runGUI 打开桌面窗口并阻塞到窗口关闭；返回 true 表示 GUI 模式已运行完毕
// （调用方应清理并退出进程）。端口固定用本进程实际监听的端口。
func runGUI(port int) bool {
	log.Printf("[gui] 启动桌面窗口（同进程，端口 %d）…", port)
	err := ideapp.Run(ideapp.Options{
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		Mode:    "hybrid", // HTTP 装配 + WebSocket 桥（引擎 WS 是 stub）
		Width:   1600,
		Height:  1000,
		Title:   "PairCode IDE",
		Logf:    log.Printf,
	})
	if err != nil {
		log.Printf("[gui] 窗口运行失败：%v", err)
	}
	log.Printf("[gui] 窗口已关闭")
	return true
}
