// Web-only 模式服务器生命周期管理。
//

package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/hoonfeng/paircode/internal/core"
)

// serverRunning 原子标志，标记 Web 服务器是否已启动。
var serverRunning atomic.Bool

// InitCore 初始化核心配置，返回默认端口。
// 被 GUI 启动面板或旧版 web-only main 调用。
func InitCore() int {
	log.Printf("[core] 加载配置… (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	core.Load()
	core.LoadLastProject()

	if !core.Loaded {
		log.Println("[core] 未发现已有配置，将使用默认设置。")
	}

	projectName := core.ProjectName()
	folders := core.Folders
	log.Printf("[core] 工作区: %s (%d 个文件夹)", projectName, len(folders))
	for i, f := range folders {
		log.Printf("[core]   文件夹[%d]: %s", i, f)
	}
	log.Printf("[core] API 已配置: %v", core.Configured())

	port := 9090
	if p := os.Getenv("WEB_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
		log.Printf("[core] 端口通过 WEB_PORT 环境变量覆盖: %d", port)
	} else {
		log.Printf("[core] 使用默认端口: %d", port)
	}

	log.Printf("[core] 初始化完成")
	return port
}

// StartWebServer 启动 Web 服务器（非阻塞，goroutine 中运行）。
// 返回 error 表示启动失败（参数错误等）；nil 表示 goroutine 已启动。
// 可通过 IsWebServerRunning() 查询运行状态。
func StartWebServer(port int) error {
	if serverRunning.Load() {
		return fmt.Errorf("服务器已在运行中")
	}
	log.Printf("[server] 正在初始化 Web 服务 (端口 %d)…", port)
	startWebUI(port)
	serverRunning.Store(true)
	log.Printf("[server] Web 服务已启动 (端口 %d)", port)
	return nil
}

// StopWebServer 停止 Web 服务器。
func StopWebServer() {
	if !serverRunning.Load() {
		return
	}
	log.Println("[server] 正在关闭 Web 服务…")
	stopWebUI()
	serverRunning.Store(false)
	log.Println("[server] Web 服务已停止。")
}

// IsWebServerRunning 查询 Web 服务器是否运行中。
func IsWebServerRunning() bool {
	return serverRunning.Load()
}
