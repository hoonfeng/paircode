// 服务器管理 —— 管理 companion web-only 子进程
//
//go:build windows

package main

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
)

// ServerManager 管理 companion web-only 进程
type ServerManager struct {
	Port    int
	cmd     *exec.Cmd
	running bool
	mu      sync.Mutex
}

func NewServerManager(port int) *ServerManager {
	return &ServerManager{Port: port}
}

// Start 启动 companion 服务器
func (sm *ServerManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("服务器已在运行中")
	}

	exePath := GetCompanionExePath()
	workDir := GetCompanionWorkDir()

	sm.cmd = exec.Command(exePath)
	sm.cmd.Dir = workDir
	sm.cmd.Env = append(sm.cmd.Env, fmt.Sprintf("WEB_PORT=%d", sm.Port))

	// 捕获输出
	sm.cmd.Stdout = log.Writer()
	sm.cmd.Stderr = log.Writer()

	if err := sm.cmd.Start(); err != nil {
		return fmt.Errorf("启动服务器失败: %w", err)
	}

	sm.running = true
	log.Printf("服务器已启动 (PID: %d, 端口: %d)", sm.cmd.Process.Pid, sm.Port)

	// 监控进程退出
	go func() {
		err := sm.cmd.Wait()
		sm.mu.Lock()
		sm.running = false
		sm.mu.Unlock()
		if err != nil {
			log.Printf("服务器进程退出: %v", err)
		} else {
			log.Println("服务器进程正常退出")
		}
	}()

	return nil
}

// Stop 停止服务器
func (sm *ServerManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running || sm.cmd == nil {
		return nil
	}

	if err := sm.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("停止服务器失败: %w", err)
	}

	sm.running = false
	log.Println("服务器已停止")
	return nil
}

// IsRunning 检查服务器是否运行中
func (sm *ServerManager) IsRunning() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.running
}
