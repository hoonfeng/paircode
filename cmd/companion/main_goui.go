// PairCode 启动面板 —— goui 桌面 GUI。
// 与 Web 服务器同进程，提供系统托盘 + 启动配置界面。
//
//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/hoonfeng/goui/pkg/app"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"

	_ "github.com/hoonfeng/goui/pkg/platform"
)

// ── 启动面板 StatefulWidget ─────────────────────────────

// PanelRoot 启动面板根组件。
type PanelRoot struct {
	widget.StatefulWidget
}

func (p *PanelRoot) CreateState() widget.State { return &panelState{} }

// panelState 启动面板运行时状态。
type panelState struct {
	widget.BaseState
	port      int    // 端口号
	err       string // 当前错误提示
	autoStart bool   // 开机自启
}

func (s *panelState) InitState() {
	s.port = InitCore() // 读取 core 配置取得默认端口
}

// toggleServer 切换服务器启动/停止。
func (s *panelState) toggleServer() {
	if IsWebServerRunning() {
		StopWebServer()
		s.err = ""
		s.SetState()
		return
	}
	if s.port <= 0 || s.port > 65535 {
		s.err = "端口号无效（1-65535）"
		s.SetState()
		return
	}
	if err := StartWebServer(s.port); err != nil {
		s.err = err.Error()
		s.SetState()
		return
	}
	s.err = ""
	s.SetState()
	// 自动打开浏览器
	openBrowser(s.port)
}

// Build 构建启动面板 UI。
func (s *panelState) Build(ctx widget.BuildContext) widget.Widget {
	running := IsWebServerRunning()

	// 状态颜色与文字
	statusColor := types.ColorFromRGB(82, 196, 26) // 绿色
	statusText := fmt.Sprintf("● 运行中 (端口 %d)", s.port)
	if !running {
		statusColor = types.ColorFromRGB(160, 160, 160) // 灰色
		statusText = "● 已停止"
	}

	return widget.Div(
		widget.Style{
			Padding:       types.EdgeInsets(24),
			FlexDirection: "column",
			Gap:           12,
		},
		// 标题 + 状态
		widget.HBox(
			widget.NewText("PairCode 启动面板", types.ColorFromRGB(48, 49, 51)),
			widget.SpacerDiv(),
			widget.NewText(statusText, statusColor),
		),
		// 端口行
		widget.HBox(
			widget.NewText("监听端口:", types.ColorFromRGB(96, 98, 102)),
			widget.SpacerDiv(),
			widget.NewInput("9090", func(text string) {
				if text != "" {
					fmt.Sscanf(text, "%d", &s.port)
				}
			}),
		),
		// 开机自动启动
		widget.HBox(
			widget.NewText("开机自动启动:", types.ColorFromRGB(96, 98, 102)),
			widget.SpacerDiv(),
			widget.NewSwitch(s.autoStart, func(v bool) { s.autoStart = v; s.SetState() }),
		),
		// 按钮
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(0, 16, 0, 8)},
			widget.NewButton(func() string {
				if running {
					return "停止服务器"
				}
				return "启动服务器"
			}(), s.toggleServer).
				WithColor(func() types.Color {
					if running { return types.ColorFromRGB(234, 67, 53) }
					return types.ColorFromRGB(64, 158, 255)
				}()).
				WithMinWidth(140),
		),
		// 提示文字
		widget.NewText(
			fmt.Sprintf("Web IDE 模式 — 在浏览器中访问 http://localhost:%d 使用", s.port),
			types.ColorFromRGB(144, 147, 153),
		),
	)
}

// openBrowser 在默认浏览器中打开 URL。
func openBrowser(port int) {
	url := fmt.Sprintf("http://localhost:%d", port)
	go func() {
		proc, err := os.StartProcess("rundll32",
			[]string{"rundll32", "url.dll,FileProtocolHandler", url},
			&os.ProcAttr{Files: []*os.File{nil, nil, nil}})
		if err != nil {
			log.Printf("打开浏览器失败: %v", err)
		} else {
			proc.Release()
		}
	}()
}

// ── main ──────────────────────────────────────────────

var application *app.Application

func main() {
	runtime.LockOSThread()

	application = app.NewApplication()
	application.SetRootWidget(&PanelRoot{})

	cfg := app.DefaultConfig()
	cfg.Title = "PairCode 启动面板"
	cfg.Width = 480
	cfg.Height = 360
	cfg.Resizable = false

	// ---- Ready：首帧后注册系统托盘 ----
	application.Ready = func() {
		trayID := application.AddTray("PairCode Web IDE", "", func() {
			openBrowser(9090)
		})
		if trayID > 0 {
			application.SetTrayMenu(trayID, []app.TrayMenuItem{
				{ID: 1, Label: "在浏览器中打开"},
				{Separator: true},
				{ID: 2, Label: "启动/停止服务器"},
				{ID: 3, Label: "显示窗口"},
				{Separator: true},
				{ID: 9, Label: "退出"},
			}, func(id int) {
				switch id {
				case 1:
					openBrowser(9090)
				case 2:
					if IsWebServerRunning() {
						StopWebServer()
					} else {
						StartWebServer(9090)
					}
					if application.Pipeline != nil {
						application.Pipeline.MarkNeedsLayout()
					}
				case 3:
					application.Window.Show()
				case 9:
					application.Quit()
				}
			})
		}
	}

	// ---- 运行 ----
	if err := application.Run(cfg); err != nil {
		log.Fatalf("启动面板退出: %v", err)
	}

	// 退出前清理
	StopWebServer()
	log.Println("PairCode 启动面板已退出。")
}
