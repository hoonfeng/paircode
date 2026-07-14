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
	"github.com/hoonfeng/goui/pkg/window"

	_ "github.com/hoonfeng/goui/pkg/platform" // 注册 Win32 窗口后端
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
	trayID    int    // 托盘图标 ID，0=未创建
}

func (s *panelState) InitState() {
	s.port = InitCore() // 读取 core 配置取得默认端口
	s.trayID = 0
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

	// 标题 + 状态
	titleBar := widget.HBox(
		widget.NewText("PairCode 启动面板", types.ColorFromRGB(48, 49, 51)),
		widget.SpacerDiv(),
		widget.NewText(statusText, statusColor),
	)

	// 端口行
	portInput := widget.NewInput("9090", func(text string) {
		if text != "" {
			fmt.Sscanf(text, "%d", &s.port)
		}
	})
	if s.port > 0 {
		portInput.Text = fmt.Sprintf("%d", s.port)
	}

	portRow := widget.HBox(
		widget.NewText("监听端口:", types.ColorFromRGB(96, 98, 102)),
		widget.SpacerDiv(),
		portInput,
	)

	// 自动启动行
	autoRow := widget.HBox(
		widget.NewText("开机自动启动:", types.ColorFromRGB(96, 98, 102)),
		widget.SpacerDiv(),
		widget.NewSwitch(s.autoStart, func(v bool) { s.autoStart = v; s.SetState() }),
	)

	// 按钮
	btnText := "启动服务器"
	btnColor := types.ColorFromRGB(64, 158, 255)
	if running {
		btnText = "停止服务器"
		btnColor = types.ColorFromRGB(234, 67, 53)
	}

	// 提示文字
	hintText := fmt.Sprintf("Web IDE 模式 — 在浏览器中访问 http://localhost:%d 使用", s.port)

	// 分隔线
	sep := &widget.Container{
		Height:      1,
		Margin:      types.EdgeInsetsLTRB(0, 4, 0, 8),
		Background:  &widget.PaintWidget{Color: ptrColor(220, 224, 228)},
	}

	return widget.Div(
		widget.Style{
			Padding:       types.EdgeInsets(24),
			FlexDirection: "column",
			Gap:           8,
		},
		titleBar,
		sep,
		portRow,
		autoRow,
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(0, 16, 0, 8)},
			widget.NewButton(btnText, s.toggleServer).
				WithColor(btnColor).
				WithMinWidth(140),
		),
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 0)},
			widget.NewText(hintText, types.ColorFromRGB(144, 147, 153)),
		),
	)
}

// ptrColor 创建 *types.Color。
func ptrColor(r, g, b uint8) *types.Color {
	c := types.ColorFromRGB(r, g, b)
	return &c
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

func main() {
	runtime.LockOSThread()

	// 初始化面板
	application := app.NewApplication()
	application.SetRootWidget(&PanelRoot{})

	// 应用配置
	cfg := app.DefaultConfig()
	cfg.Title = "PairCode 启动面板"
	cfg.Width = 480
	cfg.Height = 360
	cfg.Resizable = false

	// 设置 Ready 回调：窗口显示后添加系统托盘
	application.Ready = func() {
		hwnd := application.Window.NativeHandle()
		if hwnd == 0 {
			return
		}
		// 使用 assets/icon64.png 作为托盘图标
		iconPath := "assets/icon64.png"
		if _, err := os.Stat(iconPath); err != nil {
			iconPath = ""
		}
		trayID := window.AddTrayIcon(hwnd, "PairCode Web IDE", iconPath, func() {
			port := 9090
			openBrowser(port)
		})
		if trayID > 0 {
			window.SetTrayMenu(trayID, []window.TrayMenuItem{
				{ID: 1, Label: "在浏览器中打开"},
				{Separator: true},
				{ID: 2, Label: "启动/停止服务器"},
				{ID: 3, Label: "显示窗口"},
				{Separator: true},
				{ID: 4, Label: "退出"},
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
				case 4:
					application.Quit()
				}
			})
		}
	}

	// 运行应用（阻塞直到窗口关闭）
	if err := application.Run(cfg); err != nil {
		log.Fatalf("启动面板退出: %v", err)
	}

	// 退出前清理
	StopWebServer()
	log.Println("PairCode 启动面板已退出。")
}
