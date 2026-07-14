// PairCode Web 启动面板 —— Gio 版系统托盘启动器
// 管理 companion web-only 服务器的启动/停止，提供系统托盘和设置界面。
//
//go:build windows

package main

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var (
	cfg       *Config
	serverMgr *ServerManager
	trayMgr   *TrayManager

	// Gio 主题
	theme *material.Theme

	// 设置窗口
	settingsWin *app.Window
	showWin     bool

	// 控件
	portEditor   widget.Editor
	autoStartCB  widget.Bool
	saveBtn      widget.Clickable
	toggleBtn    widget.Clickable

	// 状态
	statusText string
	mu         sync.Mutex
)

func main() {
	log.SetPrefix("[tray] ")
	log.Println("PairCode 启动面板启动中...")

	// 初始化配置
	cfg = loadConfig()
	portEditor.SetText(fmt.Sprintf("%d", cfg.Port))
	autoStartCB.Value = cfg.AutoStart

	// 服务器管理器
	serverMgr = NewServerManager(cfg.Port)
	statusText = "● 已停止"

	// Gio 主题
	theme = material.NewTheme()
	theme.Palette = material.Palette{
		Bg:         color.NRGBA{R: 0x0d, G: 0x11, B: 0x17, A: 255},
		Fg:         color.NRGBA{R: 0xe6, G: 0xed, B: 0xf3, A: 255},
		ContrastBg: color.NRGBA{R: 0x58, G: 0xa6, B: 0xff, A: 255},
		ContrastFg: color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 255},
	}

	// 系统托盘（在独立 goroutine 中，锁定系统线程）
	trayMgr = NewTrayManager()
	go trayMgr.Run()

	// Gio 设置窗口
	go func() {
		settingsWin = new(app.Window)
		settingsWin.Option(
			app.Title("PairCode 启动面板"),
			app.Size(unit.Dp(480), unit.Dp(560)),
		)
		if err := runSettings(settingsWin); err != nil {
			log.Printf("设置窗口退出: %v", err)
		}
	}()

	// 处理托盘事件
	go handleTrayEvents()

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("正在关闭...")
	serverMgr.Stop()
	trayMgr.Destroy()
}

func handleTrayEvents() {
	for evt := range trayMgr.Events() {
		switch evt {
		case EventToggleServer:
			mu.Lock()
			toggleServer()
			mu.Unlock()
			settingsWin.Invalidate()

		case EventOpenSettings:
			showWin = true
			settingsWin.Invalidate()

		case EventQuit:
			log.Println("退出...")
			serverMgr.Stop()
			trayMgr.Destroy()
			os.Exit(0)
		}
	}
}

func toggleServer() {
	if serverMgr.IsRunning() {
		serverMgr.Stop()
		statusText = "● 已停止"
	} else {
		portStr := portEditor.Text()
		port := 9090
		if portStr != "" {
			fmt.Sscanf(portStr, "%d", &port)
		}
		serverMgr.Port = port
		if err := serverMgr.Start(); err != nil {
			statusText = "启动失败: " + err.Error()
		} else {
			statusText = fmt.Sprintf("● 运行中 (端口 %d)", port)
		}
	}
}

func runSettings(w *app.Window) error {
	var ops op.Ops
	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			renderSettings(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

// renderSettings 绘制设置窗口
func renderSettings(gtx layout.Context) layout.Dimensions {
	mu.Lock()
	defer mu.Unlock()

	// 处理按钮点击（必须在 gtx 上下文中调用）
	if toggleBtn.Clicked(gtx) {
		toggleServer()
	}
	if saveBtn.Clicked(gtx) {
		onSave()
	}

	th := theme
	inset := layout.UniformInset(unit.Dp(16))

	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:    layout.Vertical,
			Spacing: layout.SpaceBetween,
		}.Layout(gtx,
			// 标题
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.H5(th, "PairCode 启动面板").Layout(gtx)
			}),

			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),

			// 状态
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Body1(th, statusText).Layout(gtx)
			}),

			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),

			// 端口
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Body2(th, "监听端口:").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ed := material.Editor(th, &portEditor, "9090")
				ed.TextSize = unit.Sp(14)
				return ed.Layout(gtx)
			}),

			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),

			// 开机自启
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.CheckBox(th, &autoStartCB, "开机自动启动").Layout(gtx)
			}),

			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),

			// 模型列表信息
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Body2(th, fmt.Sprintf("已配置 %d 个模型提供者（在 config/models.json 中管理）", len(cfg.Models))).Layout(gtx)
			}),

			// 模型简要列表
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(cfg.Models) == 0 {
					return material.Body2(th, "暂无模型配置").Layout(gtx)
				}

				var providers []string
				for prov := range cfg.Models {
					providers = append(providers, prov)
				}

				list := layout.List{Axis: layout.Vertical}
				return list.Layout(gtx, len(providers), func(gtx layout.Context, i int) layout.Dimensions {
					prov := providers[i]
					models := cfg.Models[prov]
					return material.Body2(th, fmt.Sprintf("  %s (%d 个模型)", prov, len(models))).Layout(gtx)
				})
			}),

			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),

			// 按钮
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btnText := "启动服务器"
				if serverMgr.IsRunning() {
					btnText = "停止服务器"
				}
				return layout.Flex{
					Axis:      layout.Horizontal,
					Spacing:   layout.SpaceBetween,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Button(th, &toggleBtn, btnText).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Button(th, &saveBtn, "保存设置").Layout(gtx)
					}),
				)
			}),
		)
	})
}

func onSave() {
	portStr := portEditor.Text()
	if portStr != "" {
		fmt.Sscanf(portStr, "%d", &cfg.Port)
	}
	cfg.AutoStart = autoStartCB.Value

	if err := cfg.Save(); err != nil {
		statusText = "保存失败: " + err.Error()
	} else {
		statusText = "设置已保存"
		if cfg.AutoStart {
			enableAutoStart()
		} else {
			disableAutoStart()
		}
		if trayMgr != nil {
			trayMgr.UpdatePort(cfg.Port)
		}
	}
}
