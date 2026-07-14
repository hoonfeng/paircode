// PairCode 启动面板 —— goui 桌面 GUI。
// 无边框窗口 + 自定义标题栏 + 系统托盘 + 服务商配置选择器。
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

// ── 已知服务商默认 BaseURL ──
var defaultBaseURLs = map[string]string{
	"deepseek":          "https://api.deepseek.com/v1",
	"openai":            "https://api.openai.com/v1",
	"anthropic":         "https://api.anthropic.com/v1",
	"openai-compatible": "",
	"custom":            "",
}

// ── 有状态根组件 ──

type PanelRoot struct{ widget.StatefulWidget }

func (p *PanelRoot) CreateState() widget.State { return &panelState{} }

type panelState struct {
	widget.BaseState
	port        int
	provider    string
	baseURL     string
	apiKey      string
	model       string
	modelOpts   []widget.SelectOption // 当前服务商下的模型选项
	err         string
	msg         string
	msgOK       bool
	serverPort  int
}

func (s *panelState) InitState() {
	s.port = InitCore()
	s.provider = core.Settings.Provider
	s.baseURL = core.Settings.BaseURL
	s.apiKey = core.Settings.APIKey
	s.model = core.MainModel()
	s.serverPort = s.port
	s.refreshModelOptions()
}

// refreshModelOptions 根据当前 provider 刷新模型选项列表。
func (s *panelState) refreshModelOptions() {
	models := core.GetModels(s.provider)
	if models == nil {
		models = []string{}
	}
	s.modelOpts = make([]widget.SelectOption, 0, len(models))
	for _, m := range models {
		s.modelOpts = append(s.modelOpts, widget.SelectOption{
			Label: m,
			Value: m,
		})
	}
}

// onProviderChange 服务商选择变化时自动填充默认 BaseURL 并刷新模型列表。
func (s *panelState) onProviderChange(val string) {
	if val == "" || val == s.provider {
		return
	}
	s.provider = val
	// 自动填充默认 BaseURL
	if url, ok := defaultBaseURLs[val]; ok {
		s.baseURL = url
	}
	// 重置模型：取默认列表第一个
	models := core.GetModels(val)
	if len(models) > 0 {
		s.model = models[0]
	} else {
		s.model = ""
	}
	s.refreshModelOptions()
	s.SetState()
}

// onModelChange 模型选择变化。
func (s *panelState) onModelChange(val string) {
	if val == "" {
		return
	}
	s.model = val
	s.SetState()
}

// toggleServer 切换服务器启动/停止。
func (s *panelState) toggleServer() {
	if IsWebServerRunning() {
		StopWebServer()
		s.err = ""
		s.SetState()
		return
	}
	if s.serverPort <= 0 || s.serverPort > 65535 {
		s.err = "端口号无效"
		s.SetState()
		return
	}
	if err := StartWebServer(s.serverPort); err != nil {
		s.err = err.Error()
		s.SetState()
		return
	}
	s.err = ""
	s.SetState()
	openBrowser(s.serverPort)
}

// saveConfig 保存配置到 core.Settings。
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

// Build 构建面板 UI。
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

	// ── 配置状态行 ──
	var cfgLine widget.Widget
	if configured {
		cfgLine = widget.NewText(
			fmt.Sprintf("✓ %s / %s", s.provider, s.model),
			types.ColorFromRGB(82, 196, 26),
		)
	} else {
		cfgLine = widget.NewText("⚠ 未配置 — 选择服务商、填写密钥后保存", types.ColorFromRGB(234, 67, 53))
	}

	// ── 服务商 Select ──
	providers := core.GetProviders()
	provOpts := make([]widget.SelectOption, 0, len(providers))
	for _, p := range providers {
		provOpts = append(provOpts, widget.SelectOption{Label: p, Value: p})
	}
	providerSel := widget.NewSelect(provOpts).
		WithPlaceholder("选择服务商").
		WithValue(s.provider).
		WithOnChanged(s.onProviderChange).
		WithWidth(200)

	// ── 模型 Select ──
	modelSel := widget.NewSelect(s.modelOpts).
		WithPlaceholder("选择模型").
		WithValue(s.model).
		WithOnChanged(s.onModelChange).
		WithWidth(200)

	// ── 文本输入 ──
	baseURLInput := widget.NewInput("https://...", func(t string) { s.baseURL = t })
	baseURLInput.Text = s.baseURL

	apiKeyInput := widget.NewInput("sk-...", func(t string) { s.apiKey = t })
	apiKeyInput.Text = s.apiKey

	portInput := widget.NewInput("9090", func(t string) {
		if t != "" {
			fmt.Sscanf(t, "%d", &s.serverPort)
		}
	})
	portInput.Text = fmt.Sprintf("%d", s.serverPort)

	// ── 提示消息 ──
	var msgEl widget.Widget
	if s.msg != "" {
		c := types.ColorFromRGB(82, 196, 26)
		if !s.msgOK {
			c = types.ColorFromRGB(234, 67, 53)
		}
		msgEl = widget.NewText(s.msg, c)
	}
	var errEl widget.Widget
	if s.err != "" {
		errEl = widget.NewText("错误: "+s.err, types.ColorFromRGB(234, 67, 53))
	}

	// ── label 辅助 ──
	lbl := func(t string) widget.Widget {
		return widget.NewText(t, types.ColorFromRGB(96, 98, 102))
	}
	// 输入行：标签 + 固定 200px 控件
	rowF := func(label widget.Widget, ctrl widget.Widget) widget.Widget {
		return widget.HBox(label, widget.SpacerDiv(), ctrl)
	}
	w200 := func(w widget.Widget) widget.Widget {
		return widget.Div(widget.Style{Width: 200}, w)
	}

	// ── 服务商配置区块 ──
	configBody := widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(0, 2, 18, 2), FlexDirection: "column", Gap: 6},
		widget.NewText("── 服务商 ──", types.ColorFromRGB(144, 147, 153)),
		rowF(lbl("服务商:"), providerSel),
		rowF(lbl("接口地址:"), w200(baseURLInput)),
		rowF(lbl("API 密钥:"), w200(apiKeyInput)),
		rowF(lbl("模型:"), modelSel),
		widget.HBox(
			widget.SpacerDiv(),
			widget.NewButton("保存配置", s.saveConfig).
				WithColor(types.ColorFromRGB(64, 158, 255)).
				WithMinWidth(120),
		),
		msgEl,
	)

	// ── 服务器区块 ──
	serverBody := widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(0, 2, 2, 2), FlexDirection: "column", Gap: 6},
		widget.NewText("── 服务器 ──", types.ColorFromRGB(144, 147, 153)),
		rowF(lbl("监听端口:"), w200(portInput)),
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
		errEl,
	)

	// ── 内容体 ──
	body := widget.Div(
		widget.Style{Padding: types.EdgeInsets(16), FlexDirection: "column", Gap: 4},
		widget.HBox(
			widget.NewText(statusText, statusColor),
			widget.SpacerDiv(),
			cfgLine,
		),
		configBody,
		serverBody,
		widget.SpacerDiv(),
		widget.NewText(
			fmt.Sprintf("Web IDE — http://localhost:%d", s.serverPort),
			types.ColorFromRGB(144, 147, 153),
		),
	)

	return widget.VBox(
		titleBar(),
		&widget.Container{Height: 1, Background: &widget.PaintWidget{Color: ptrColor(220, 224, 228)}},
		&widget.Expanded{SingleChildWidget: widget.SingleChildWidget{Child: body}, Flex: 1},
	)
}

// ── 自定义标题栏 ──

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
			tbBtn("—", func() {
				if application != nil && application.Window != nil {
					application.Window.Hide()
				}
			}),
			tbBtn("✕", func() { application.Quit() }),
		),
	)
}

func tbBtn(label string, onClick func()) widget.Widget {
	return &widget.Button{
		Text: label, OnClick: onClick,
		Color: types.ColorFromRGB(48, 52, 64), TextColor: types.ColorFromRGB(235, 238, 245),
		MinWidth: 46, MinHeight: 36,
	}
}

// ── 辅助 ──

func ptrColor(r, g, b uint8) *types.Color {
	c := types.ColorFromRGB(r, g, b)
	return &c
}

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
	cfg.Height = 560
	cfg.Resizable = true
	cfg.Borderless = true

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
					if application.Window != nil {
						application.Window.Show()
					}
				case 9:
					application.Quit()
				}
			})
		}
		// 自绘标题栏命中区：36px 高，右侧 2×46=92px 留给按钮
		application.SetTitleBar(36, 92)
		application.EnableWindowEffects()
	}

	if err := application.Run(cfg); err != nil {
		log.Fatalf("启动面板退出: %v", err)
	}

	StopWebServer()
	log.Println("PairCode 启动面板已退出。")
}
