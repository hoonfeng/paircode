// PairCode 启动面板 —— goui 桌面 GUI。
// 无边框窗口 + 自定义标题栏 + 系统托盘。
// 与 Web 服务器同进程，包含服务商配置和服务器启停。
//
//go:build windows

package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"

	"github.com/hoonfeng/goui/pkg/app"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
	"github.com/hoonfeng/paircode/cmd/companion/core"

	_ "github.com/hoonfeng/goui/pkg/platform"
)

// ── 有状态根组件 ───────────────────────────────────────

type PanelRoot struct{ widget.StatefulWidget }

func (p *PanelRoot) CreateState() widget.State { return &panelState{} }

type panelState struct {
	widget.BaseState
	port     int
	provider string
	baseURL  string
	apiKey   string
	model    string
	err      string
	msg      string
	msgOK    bool
}

func (s *panelState) InitState() {
	s.port = InitCore()
	s.provider = core.Settings.Provider
	s.baseURL = core.Settings.BaseURL
	s.apiKey = core.Settings.APIKey
	s.model = core.MainModel()
}

func (s *panelState) toggleServer() {
	if IsWebServerRunning() {
		StopWebServer()
		s.err = ""
		s.SetState()
		return
	}
	if s.port <= 0 || s.port > 65535 {
		s.err = "端口号无效"
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
	openBrowser(s.port)
}

func (s *panelState) saveConfig() {
	core.Settings.Provider = s.provider
	core.Settings.BaseURL = s.baseURL
	core.Settings.APIKey = s.apiKey
	core.Settings.ExecuteModel = s.model
	core.Settings.Model = s.model
	core.Save()
	if core.Configured() {
		s.msg = "配置已保存"
		s.msgOK = true
	} else {
		s.msg = "已保存，但 API 密钥或模型为空，请补充完整"
		s.msgOK = false
	}
	s.SetState()
}

func (s *panelState) Build(ctx widget.BuildContext) widget.Widget {
	running := IsWebServerRunning()
	configured := core.Configured()

	// ── 状态栏 ──
	statusColor := types.ColorFromRGB(82, 196, 26)
	statusText := "● 运行中"
	if !running {
		statusColor = types.ColorFromRGB(160, 160, 160)
		statusText = "● 已停止"
	}
	statusLine := widget.NewText(statusText, statusColor)

	// ── 配置状态 ──
	var cfgLine widget.Widget
	if configured {
		cfgLine = widget.HBox(
			widget.NewText("✓ 已配置 ", types.ColorFromRGB(82, 196, 26)),
			widget.NewText(s.provider+"/"+s.model, types.ColorFromRGB(96, 98, 102)),
		)
	} else {
		cfgLine = widget.NewText("⚠ 未配置 API — 填写下方信息后保存", types.ColorFromRGB(234, 67, 53))
	}

	// ── 输入控件 ──
	providerInput := widget.NewInput("deepseek", func(t string) { s.provider = t })
	providerInput.Text = s.provider
	baseURLInput := widget.NewInput("https://api.deepseek.com/v1", func(t string) { s.baseURL = t })
	baseURLInput.Text = s.baseURL
	apiKeyInput := widget.NewInput("sk-...", func(t string) { s.apiKey = t })
	apiKeyInput.Text = s.apiKey
	modelInput := widget.NewInput("deepseek-v4-flash", func(t string) { s.model = t })
	modelInput.Text = s.model
	portInput := widget.NewInput("9090", func(t string) {
		if t != "" {
			fmt.Sscanf(t, "%d", &s.port)
		}
	})
	portInput.Text = fmt.Sprintf("%d", s.port)

	// ── 提示消息 ──
	var msgWidget widget.Widget
	if s.msg != "" {
		c := types.ColorFromRGB(82, 196, 26)
		if !s.msgOK {
			c = types.ColorFromRGB(234, 67, 53)
		}
		msgWidget = widget.NewText(s.msg, c)
	}
	var errWidget widget.Widget
	if s.err != "" {
		errWidget = widget.NewText("错误: "+s.err, types.ColorFromRGB(234, 67, 53))
	}

	// ── label 辅助 ──
	label := func(t string) widget.Widget {
		return widget.NewText(t, types.ColorFromRGB(96, 98, 102))
	}
	// 输入行：左侧标签 + 右侧固定 200px 输入框
	inputRow := func(lbl string, inp widget.Widget) widget.Widget {
		return widget.HBox(
			label(lbl),
			widget.SpacerDiv(),
			widget.Div(widget.Style{Width: 200}, inp),
		)
	}

	// ── 配置区块 ──
	configSection := widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(0, 2, 0, 2)},
		widget.NewText("── 服务商配置 ──", types.ColorFromRGB(144, 147, 153)),
		inputRow("提供者:", providerInput),
		inputRow("接口地址:", baseURLInput),
		inputRow("API密钥:", apiKeyInput),
		inputRow("模型:", modelInput),
		widget.HBox(
			widget.SpacerDiv(),
			widget.NewButton("保存配置", s.saveConfig).
				WithColor(types.ColorFromRGB(64, 158, 255)).
				WithMinWidth(120),
		),
		msgWidget,
	)

	// ── 服务器区块 ──
	serverSection := widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(0, 2, 0, 2)},
		widget.NewText("── 服务器 ──", types.ColorFromRGB(144, 147, 153)),
		inputRow("监听端口:", portInput),
		widget.NewButton(func() string {
			if running {
				return "停止服务器"
			}
			return "启动服务器"
		}(), s.toggleServer).
			WithColor(func() types.Color {
				if running {
					return types.ColorFromRGB(234, 67, 53)
				}
				return types.ColorFromRGB(64, 158, 255)
			}()).
			WithMinWidth(140),
		errWidget,
	)

	// ── 拼装总面板（不含标题栏——标题栏在 Build 外由 titleBar() 返回） ──
	body := widget.Div(
		widget.Style{
			Padding:       types.EdgeInsets(16),
			FlexDirection: "column",
			Gap:           4,
		},
		statusLine,
		cfgLine,
		configSection,
		serverSection,
		widget.SpacerDiv(),
		widget.NewText(
			fmt.Sprintf("Web IDE — http://localhost:%d", s.port),
			types.ColorFromRGB(144, 147, 153),
		),
	)

	// 返回垂直布局：标题栏 + 分隔线 + 内容体
	return widget.VBox(
		titleBar(),
		&widget.Container{Height: 1, Background: &widget.PaintWidget{Color: ptrColor(220, 224, 228)}},
		&widget.Expanded{SingleChildWidget: widget.SingleChildWidget{Child: body}, Flex: 1},
	)
}

// titleBar 自定义标题栏：左拖动区 + 右 — ✕ 按钮。
func titleBar() widget.Widget {
	return widget.Div(
		widget.Style{Height: 36, BackgroundColor: types.ColorRef(48, 52, 64)},
		widget.HBox(
			&widget.Expanded{
				SingleChildWidget: widget.SingleChildWidget{
					Child: widget.Div(
						widget.Style{Padding: types.EdgeInsetsLTRB(14, 0, 0, 0), Height: 36},
						widget.NewText("PairCode 启动面板", types.ColorFromRGB(235, 238, 245)),
					),
				},
				Flex: 1,
			},
			trayButton("—", func() {
				if application != nil && application.Window != nil {
					application.Window.Hide()
				}
			}),
			trayButton("✕", func() {
				if application != nil {
					application.Quit()
				}
			}),
		),
	)
}

// trayButton 标题栏按钮。
func trayButton(label string, onClick func()) widget.Widget {
	return &widget.Button{
		Text:      label,
		OnClick:   onClick,
		Color:     types.ColorFromRGB(48, 52, 64),
		TextColor: types.ColorFromRGB(235, 238, 245),
		MinWidth:  46,
		MinHeight: 36,
	}
}

// ptrColor 辅助：创建 *types.Color。
func ptrColor(r, g, b uint8) *types.Color {
	c := types.ColorFromRGB(r, g, b)
	return &c
}

// openBrowser 在默认浏览器中打开 URL。
func openBrowser(port int) {
	url := fmt.Sprintf("http://localhost:%d", port)
	go func() {
		if err := exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start(); err != nil {
			log.Printf("打开浏览器失败: %v", err)
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
	cfg.Height = 520
	cfg.Resizable = true
	cfg.Borderless = true

	application.Ready = func() {
		// 系统托盘
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
					if application.Window != nil {
						application.Window.Show()
					}
				case 9:
					application.Quit()
				}
			})
		}
		// 声明标题栏拖拽区（36px 高，右侧 92px 留给按钮）
		application.SetTitleBar(36, 92)
		application.EnableWindowEffects()
	}

	if err := application.Run(cfg); err != nil {
		log.Fatalf("启动面板退出: %v", err)
	}

	StopWebServer()
	log.Println("PairCode 启动面板已退出。")
}
