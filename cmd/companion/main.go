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
	"time"
)

// 编译版本号（由 packager 通过 -ldflags=-X main.version=<version> 注入）
var version = "v1.1.2"

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
	// ★ 安全处理：收到信号时不立即退出，先检查是否有运行中的 agent 会话。
	// 避免因 run_command/run_background 超时触发的 CTRL_BREAK_EVENT（Windows 上映射为 SIGTERM）
	// 或 agent 执行过程中的误信号导致服务器意外关闭。
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// 首次信号：检查是否有 agent 运行中
	s := <-sig
	log.Printf("[main] 收到退出信号 (%v)，检查运行中的 agent 会话...", s)

	if agentMgr != nil {
		running := agentMgr.ListRunning()
		if len(running) > 0 {
			log.Printf("[main] 有 %d 个 agent 会话仍在运行，不退出（防止误信号导致服务器关闭）", len(running))
			log.Printf("[main] 再发一次 %v 可强制关闭", s)
			// 二次信号才真正退出
			select {
			case <-sig:
				log.Println("[main] 收到二次退出信号，强制关闭...")
			case <-waitForAgentStop(running):
				log.Println("[main] agent 会话已全部结束，正在关闭...")
			}
		}
	}

	log.Println("[main] 收到退出信号，正在关闭...")
	StopWebServer()
	log.Println("[main] 已退出。")
}

// waitForAgentStop 等待指定 agent 会话全部结束。返回一个 closed channel 表示完成。
// 最多等待 30 秒超时，超时后强制返回（不阻塞服务器关闭流程）。
func waitForAgentStop(initialRunning []string) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		timeout := time.After(30 * time.Second)
		for {
			select {
			case <-ticker.C:
				running := agentMgr.ListRunning()
				// 检查 initialRunning 中的会话是否都已结束
				allDone := true
				for _, id := range initialRunning {
					for _, r := range running {
						if r == id {
							allDone = false // 此会话仍在运行
							break
						}
					}
					if !allDone {
						break
					}
				}
				if allDone {
					return
				}
			case <-timeout:
				log.Printf("[main] 等待 agent 会话结束超时（30s），强制关闭")
				return
			}
		}
	}()
	return done
}
