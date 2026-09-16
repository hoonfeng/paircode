// PairCode IDE Web 服务器入口。
// 直接启动 Web 服务（不再有 goui 启动面板）。
//

package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
)

// portInUseHoldSeconds 端口被占用时，提示信息在控制台停留的秒数
// （双击启动的窗口不会一闪而过，用户能看清原因）。
const portInUseHoldSeconds = 10

// 编译版本号（由 packager 通过 -ldflags=-X main.version=<version> 注入）
// 也用于 /api/system/info 返回给前端 About 弹窗展示。
// ★ 缺省值 = 当前发布版本（packager.json 注入覆盖；无注入时兜底显示）。
var version = "v1.6.1"

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

	// ★ 日志落盘（<安装目录>/logs/paircode.log）：最先初始化，确保启动日志也进文件
	initLogFile()

	port := InitCore()

	log.Printf("[main] PairCode IDE %s 启动中 (%s, %s/%s)", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	log.Printf("[main] 工作目录: %s", getCwd())

	log.Printf("[main] 正在启动 Web 服务器 (端口 %d)...", port)
	if err := StartWebServer(port); err != nil {
		// ★ 2026-09-16：端口占用 = 启动失败。明确提示 + 停留后退出，
		//   不再留下「有窗口、无 WebUI」的僵尸实例。
		var pie *PortInUseError
		if errors.As(err, &pie) {
			if pie.ServingPairCode() {
				log.Printf("[main] 启动中止：端口 %d 已被另一个 PairCode 实例占用（本机已有实例在运行）。", pie.Port)
				log.Printf("[main] 请直接打开 http://localhost:%d 使用现有实例；若确实需要并行运行，请设环境变量 WEB_PORT 指定其它端口。", pie.Port)
			} else {
				log.Printf("[main] 启动中止：端口 %d 已被其它程序占用（%v）。", pie.Port, pie.Err)
				log.Printf("[main] 可设环境变量 WEB_PORT 指定其它端口后重试。")
			}
			log.Printf("[main] 本窗口将在 %d 秒后自动关闭。", portInUseHoldSeconds)
			time.Sleep(portInUseHoldSeconds * time.Second)
			os.Exit(1)
		}
		log.Fatalf("[main] 启动失败: %v", err)
	}
	log.Printf("[main] 已启动，请打开 http://0.0.0.0:%d（本机浏览器可用 http://localhost:%d，局域网设备用本机 IP）", port, port)

	// ★ 2026-09-15：退出信号钩子——Ctrl+C / SIGTERM 时清理 MCP 连接池中的子进程，
	//   避免 MCP 服务器进程孤儿化残留。
	// ★ 2026-09-16：追加 KillAllBackgroundProcesses——终止全部后台进程
	//   （run_background/exec_command 启动的 dev server、插件后台进程如
	//   微信桥 wxbridge.exe），堵住「子进程孤儿残留」同类缺口；与插件的
	//   stdin-EOF 自守护（如微信桥）形成双保险。
	//   注意：Windows 强杀（任务管理器终止进程）不触发任何清理（OS 限制），
	//   由空闲回收兜底（PAIR_MCP_IDLE_TTL_SEC，默认 10 分钟）。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("[main] 收到退出信号，清理 MCP 连接与后台进程…")
		agent.CloseAllMCPConnections()
		agent.KillAllBackgroundProcesses()
		os.Exit(0)
	}()

	// 永久阻塞，直到用户关闭命令窗口或 kill 进程
	select {}
}
