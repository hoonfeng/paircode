// 微信 ClawBot 桥（wechat-bridge）——常驻进程入口。
//
// 与 PairCode 插件壳（.pair/plugins/wechat-bridge/index.js）通过 stdio 行 JSON 协议交互：
//   - stdout：事件流（hello/status/account/log/error），每行一条 JSON
//   - stdin ：命令流（status/stop/relogin/verify_code/qr.refresh/accounts.remove）
//
// 构建：go build ./plugins-src/plugins/wechat-bridge（产物 wxbridge-<os>-<arch>）。
// 自守护：stdin EOF（宿主退出/管道断开）→ 立即自行退出，防孤儿残留。
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/account"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/bridge"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/config"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/ctl"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/localhttp"
)

// version 构建版本（可经 -ldflags "-X main.version=..." 覆盖）。
var version = "1.0.0-dev"

var (
	flagDataDir   = flag.String("data-dir", "", "数据目录（必填；通常 <workspace>/.pair/wechat-bridge）")
	flagPaircode  = flag.String("paircode", "", "PairCode 地址（默认 http://127.0.0.1:9090）")
	flagPort      = flag.Int("port", 9097, "本地回环 HTTP 端口（0=自动）")
	flagToken     = flag.String("token", "", "本地 HTTP 写操作校验 token（空=不校验；插件壳启动时注入）")
	flagDump      = flag.String("dump", "", "调试命令（selftest|qr|qr-fetch|login|ws-probe），执行后退出")
	flagWorkspace = flag.String("workspace", "", "默认会话工作区（账号未单独指定时；空=自动推断）")
)

func main() {
	flag.Parse()
	if *flagDump != "" {
		os.Exit(runDump(*flagDump))
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "[bridge] 致命错误: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if *flagDataDir == "" {
		return fmt.Errorf("缺少 --data-dir 参数")
	}
	cfg := config.Default()
	cfg.DataDir = *flagDataDir
	if *flagPaircode != "" {
		cfg.PairCodeURL = *flagPaircode
	}
	cfg.Port = *flagPort
	cfg.WorkspaceRoot = *flagWorkspace

	bus := ctl.NewBus()

	// ── 单实例探测：数据目录锁 + /health 探活 ──
	// 热更新时序由插件壳保证（先 onDispose 停旧桥，再启动新桥）；此处仅兜底。
	if inst, ok := probeExistingInstance(cfg); ok {
		bus.Hello(version, os.Getpid(), inst.Port, true, inst.Accounts)
		return nil
	}

	// ── 账号总管 ──
	mgr := account.NewManager(cfg.DataDir)
	if err := mgr.Load(); err != nil {
		bus.Log("warn", fmt.Sprintf("账号注册表加载失败: %v", err))
	}
	mgr.SetOnChange(func() {
		evt := map[string]any{"evt": "status"}
		for k, v := range mgr.StatusMap() {
			evt[k] = v
		}
		bus.Emit(evt)
		updateLockAccounts(cfg, mgr.IDs())
	})

	// ── 桥总管（多账号桥循环 + 事件流）──
	// 登录成功（SetOnReady）→ 启动对应账号桥；移除账号（SetOnRemoved）→ 停桥。
	host := bridge.NewHost(cfg, mgr, func(format string, args ...any) {
		bus.Log("info", fmt.Sprintf(format, args...))
	})
	mgr.SetOnReady(host.StartAccount)
	mgr.SetOnRemoved(host.StopAccount)

	// ── 本地回环 HTTP ──
	srv := localhttp.New(localhttp.Options{
		Token: *flagToken,
		Login: func() localhttp.SessionLike {
			if s := mgr.CurrentLoginSession(); s != nil {
				return s
			}
			return nil
		},
		Status: mgr.StatusMap,
		Accounts: func() []map[string]any {
			sm := mgr.StatusMap()
			arr, _ := sm["accounts"].([]map[string]any)
			return arr
		},
		AddAcct:    mgr.Add,
		RemoveAcct: mgr.Remove,
		Relogin:    mgr.Relogin,
		Send:       host.SendTo,
		Contacts:   host.Contacts,
	})
	port, err := srv.Start(cfg.Port)
	if err != nil {
		return fmt.Errorf("本地 HTTP 启动失败: %w", err)
	}
	cfg.Port = port

	if err := writeLock(cfg); err != nil {
		bus.Log("warn", fmt.Sprintf("写实例锁失败（继续运行）: %v", err))
	}
	defer removeLock(cfg)

	bus.Hello(version, os.Getpid(), port, false, mgr.IDs())
	bus.Log("info", fmt.Sprintf("桥已启动（data-dir=%s，paircode=%s，port=%d）", cfg.DataDir, cfg.PairCodeURL, port))

	// ── 桥循环启动（在 hello 之后：保证壳先收握手事件；StartAll 内部按
	// 注册表对每个账号幂等启动——重复账号不会重复建桥）──
	host.Start()

	// ── 启动编排：无账号 → 自动建默认账号并进入扫码登录 ──
	mgr.Bootstrap()

	// ── 退出协调：信号 或 stdin EOF（宿主退出）──
	quit := make(chan string, 2)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-sigCh
		quit <- fmt.Sprintf("收到信号 %v", s)
	}()

	go ctl.ReadCommands(
		func(c ctl.Command) { handleCommand(bus, c, quit, mgr) },
		func() { quit <- "stdin EOF（宿主已退出）" },
	)

	reason := <-quit
	bus.Log("info", fmt.Sprintf("桥退出：%s", reason))
	host.StopAll()
	mgr.StopAll()
	srv.Stop()
	return nil
}

// handleCommand 处理壳命令。
func handleCommand(bus *ctl.Bus, c ctl.Command, quit chan<- string, mgr *account.Manager) {
	switch c.Cmd {
	case "stop":
		select {
		case quit <- "收到 stop 命令":
		default:
		}
	case "status":
		evt := map[string]any{"evt": "status"}
		for k, v := range mgr.StatusMap() {
			evt[k] = v
		}
		bus.Emit(evt)
	case "verify_code":
		if s := mgr.CurrentLoginSession(); s != nil && s.SubmitVerifyCode(c.Code) {
			bus.Log("info", "验证码已提交")
		} else {
			bus.Log("warn", "当前无等待验证码的登录会话")
		}
	case "qr.refresh":
		if s := mgr.CurrentLoginSession(); s != nil {
			s.RefreshQR()
			bus.Log("info", "已请求刷新二维码")
		} else {
			bus.Log("warn", "当前无登录会话")
		}
	case "relogin":
		if c.Account != "" {
			if err := mgr.Relogin(c.Account); err != nil {
				bus.Error(c.Account, err.Error(), false)
			}
		} else if s := mgr.CurrentLoginSession(); s != nil {
			s.RefreshQR()
		} else {
			bus.Log("warn", "未指定账号且无活动登录会话")
		}
	case "accounts.remove":
		if c.Account == "" {
			bus.Error("", "缺少 account 参数", false)
		} else if err := mgr.Remove(c.Account); err != nil {
			bus.Error(c.Account, err.Error(), false)
		} else {
			bus.Log("info", fmt.Sprintf("账号已移除: %s", c.Account))
		}
	default:
		bus.Log("warn", fmt.Sprintf("未知命令: %s", c.Cmd))
	}
}
