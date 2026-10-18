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
	"os"
	"strconv"

	"wb-ui/ideapp"
)

// runGUI 打开桌面窗口并阻塞到窗口关闭；返回 true 表示 GUI 模式已运行完毕
// （调用方应清理并退出进程）。端口固定用本进程实际监听的端口。
func runGUI(port int) bool {
	// ★ CDP 调试端口（与 Chrome DevTools 同协议）：默认关闭。设置
	//   GOUIDE_DEBUG_PORT=9223 后，外部工具（Playwright/Puppeteer/任意 WS
	//   客户端）即可查询 DOM、computed style、注入输入、截图；日志会打印
	//   "DevTools listening on ws://…" 供发现。
	debugPort := 0
	if v := os.Getenv("GOUIDE_DEBUG_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < 65536 {
			debugPort = n
		} else {
			log.Printf("[gui] GOUIDE_DEBUG_PORT=%q 无效（需 1-65535），调试端口关闭", v)
		}
	}
	log.Printf("[gui] 启动桌面窗口（同进程，端口 %d）…", port)
	if debugPort > 0 {
		log.Printf("[gui] CDP 调试端口：127.0.0.1:%d", debugPort)
	}
	err := ideapp.Run(ideapp.Options{
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		Mode:    "hybrid", // HTTP 装配 + WebSocket 桥（引擎 WS 是 stub）
		Width:   1600,
		Height:  1000,
		Title:   "PairCode IDE",
		DebugPort: debugPort,
		Logf:    log.Printf,
	})
	if err != nil {
		log.Printf("[gui] 窗口运行失败：%v", err)
	}
	log.Printf("[gui] 窗口已关闭")
	return true
}
