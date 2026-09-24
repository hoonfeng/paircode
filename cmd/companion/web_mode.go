// Web-only 模式服务器生命周期管理。
//

package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

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
	// ★ 2026-09-20：不再输出「是否已配置」——连接信息唯一来源是 AI 配置
	//   （ai-presets.json，settings 顶层连接字段已退出核心），且启动时插件尚未装载
	//   （装配器未注册 → 装配结果为空），此处判定必然失真。就绪判定请以运行期
	//   装配结果为准（agent.ConfiguredProvider，缺失项含 API Key/地址/模型）。
	log.Printf("[core] 当前激活 AI 配置: %q（连接信息以 config/ai-presets.json 为准）", core.Settings.Preset)

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
	// ★ 2026-09-16 端口预检（快速失败）：抢不到端口立即返回错误。
	//   本机已有 PairCode 实例时（双击第二个 pair.exe 的典型场景），旧实现要等
	//   插件/工具集装载几十秒后才在日志里冒一行「服务器错误」，且进程继续运行，
	//   留下「有窗口、无 WebUI」的僵尸实例（白占一整份内存）。现由 main 明确
	//   提示「已有实例在运行」并退出。
	if err := preflightPort(port); err != nil {
		return err
	}
	log.Printf("[server] 正在初始化 Web 服务 (端口 %d)…", port)
	if err := startWebUI(port); err != nil {
		return err
	}
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

// ─── 端口占用诊断（2026-09-16）────────────────────────────────

// PortInUseError 表示 Web 服务端口无法绑定（启动失败）。
// 最常见成因是「本机已有 PairCode 实例在运行」。
type PortInUseError struct {
	Port int
	Err  error
}

func (e *PortInUseError) Error() string {
	return fmt.Sprintf("端口 %d 已被占用: %v", e.Port, e.Err)
}

// Unwrap 保留底层错误（net.OpError / WSAEADDRINUSE），供 errors.Is 判定。
func (e *PortInUseError) Unwrap() error { return e.Err }

// ServingPairCode 探测占用该端口的服务是否为另一个 PairCode 实例：
// GET http://127.0.0.1:<port>/ 返回页面含 "PairCode" 特征即认定。
// 仅用于把用户提示写准（PairCode 实例 vs 其它程序），探测失败一律按后者处理。
func (e *PortInUseError) ServingPairCode() bool {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", e.Port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return false
	}
	return strings.Contains(string(body), "PairCode")
}

// preflightPort 启动前端口预检：试绑通配地址成功即立即释放。
// Go 对通配符地址（0.0.0.0）会创建双栈 socket（IPv4/IPv6 一并占用），
// 因此这里失败即等价于「两个地址族都抢不到该端口」。
func preflightPort(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return &PortInUseError{Port: port, Err: err}
	}
	_ = ln.Close()
	return nil
}
