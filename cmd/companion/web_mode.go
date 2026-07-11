// Web-only 模式启动入口：跳过 GWui 桌面 GUI，只启动 Web IDE 管理服务器。
// 用于自举开发（用浏览器管理项目、编辑代码、运行 Agent）。
//
//go:build webonly

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hoonfeng/paircode/cmd/companion/core"
)

func main() {
	// ── 初始化核心 ──
	core.Load()
	core.LoadLastProject()

	if !core.Loaded {
		log.Println("[WebMode] 未发现已有配置，将使用默认设置。")
	}

	log.Printf("[WebMode] 工作区: %s", core.Root())
	log.Printf("[WebMode] 文件夹: %v", core.Folders)
	log.Printf("[WebMode] 已配置 API: %v", core.Configured())

	// ── 启动 Web UI 服务器 ──
	port := 9090
	if p := os.Getenv("WEB_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}
	startWebUI(port)

	// ── 等待退出信号 ──
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("[WebMode] 正在关闭 Web 服务器…")
	stopWebUI()
	log.Println("[WebMode] 已退出。")
}
