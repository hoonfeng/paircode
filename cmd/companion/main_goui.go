// PairCode 启动面板 —— goui 桌面 GUI。
// 无边框窗口 + 自定义标题栏 + 系统托盘。
// 服务商配置：Select 选择 + 可视化新增 + 自动保存 models.json。
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

// defaultBaseURLs 已知服务商默认接口地址。
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
	port       int
	provider   string
	baseURL    string
	apiKey     string
	model      string
	modelOpts  []widget.SelectOption
	provOpts   []widget.SelectOption
	err        string
	msg        string
	msgOK      bool
	serverPort int
	// 新增服务商表单状态
	adding    bool
	newName   string
	newURL    string
	newModels string
}

func (s *panelState) InitState() {
	s.port = InitCore()
	s.provider = core.Settings.Provider
	s.baseURL = core.Settings.BaseURL
	s.apiKey = core.Settings.APIKey
	s.model = core.MainModel()
	s.serverPort = s.port
	s.refreshAll()
}

func (s *panelState) refreshAll() {
	providers := core.GetProviders()
	s.provOpts = make([]widget.SelectOption, 0, len(providers))
	for _, p := range providers {
		s.provOpts = append(s.provOpts, widget.SelectOption{Label: p, Value: p})
	}
	s.refreshModels()
}

func (s *panelState) refreshModels() {
	models := core.GetModels(s.provider)
	if models == nil {
		models = []string{}
	}
	s.modelOpts = make([]widget.SelectOption, 0, len(models))
	for _, m := range models {
		s.modelOpts = append(s.modelOpts, widget.SelectOption{Label: m, Value: m})
	}
}

func (s *panelState) onProviderChange(val string) {
	if val == "" || val == s.provider {
		return
	}
	s.provider = val
	if url, ok := defaultBaseURLs[val]; ok {
		s.baseURL = url
	} else {
		s.baseURL = ""
	}
	models := core.GetModels(val)
	if len(models) > 0 {
		s.model = models[0]
	} else {
		s.model = ""
	}
	s.refreshModels()
	s.SetState()
}

func (s *panelState) onModelChange(val string) {
	if val == "" {
		return
	}
	s.model = val
	s.SetState()
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
		s.msg = "已保存，但密钥或模型为空"
		s.msgOK = false
	}
	s.SetState()
}

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

// confirmAdd 确认新增服务商 → core.AddProvider + SaveModelList。
func (s *panelState) confirmAdd() {
	if s.newName == "" {
		s.msg, s.msgOK = "服务商名称不能为空", false
		s.SetState()
		return
	}
	// 解析模型：逗号分隔
	var models []string
	if s.newModels != "" {
		models = splitModels(s.newModels)
	}
	if err := core.AddProvider(s.newName, models); err != nil {
		s.msg, s.msgOK = "添加失败: "+err.Error(), false
		s.SetState()
		return
	}
	defaultBaseURLs[s.newName] = s.newURL
	s.provider = s.newName
	s.baseURL = s.newURL
	if len(models) > 0 {
		s.model = models[0]
	} else {
		s.model = ""
	}
	s.refreshAll()
	s.adding = false
	s.newName, s.newURL, s.newModels = "", "", ""
	s.msg, s.msgOK = fmt.Sprintf("服务商 %q 已添加", s.provider), true
	s.SetState()
}

// splitModels 按逗号拆分模型名。
func splitModels(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			t := s[start:i]
			if t != "" {
				out = append(out, t)
			}
			start = i + 1
		}
	}
	return out
}

// ── Build ──────────────────────────────────────────────

func (s *panelState) Build(ctx widget.BuildContext) widget.Widget {
	running := IsWebServerRunning()
	configured := core.Configured()

	// 状态栏
	statusColor := types.ColorFromRGB(82, 196, 26)
	statusText := "● 运行中"
	if !running {
		statusColor = types.ColorFromRGB(160, 160, 160)
		statusText = "● 已停止"
	}
	var cfgLine widget.Widget
	if configured {
		cfgLine = widget.NewText(fmt.Sprintf("✓ %s / %s", s.provider, s.model), types.ColorFromRGB(82, 196, 26))
	} else {
		cfgLine = widget.NewText("⚠ 未配置", types.ColorFromRGB(234, 67, 53))
	}

	// ── 控件 ──
	providerSel := widget.NewSelect(s.provOpts).
		WithPlaceholder("选择服务商").
		WithValue(s.provider).
		WithOnChanged(s.onProviderChange).
		WithWidth(200)
	modelSel := widget.NewSelect(s.modelOpts).
		WithPlaceholder("选择模型").
		WithValue(s.model).
		WithOnChanged(s.onModelChange).
		WithWidth(200)
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

	// 提示
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

	// 辅助
	lbl := func(t string) widget.Widget {
		return widget.NewText(t, types.ColorFromRGB(96, 98, 102))
	}
	w200 := func(w widget.Widget) widget.Widget {
		return widget.Div(widget.Style{Width: 200}, w)
	}
	row := func(l, c widget.Widget) widget.Widget {
		return widget.HBox(l, widget.SpacerDiv(), c)
	}

	// ── 新增服务商表单 ──
	var addForm widget.Widget
	if s.adding {
		nameInput := widget.NewInput("deepseek-vllm", func(t string) { s.newName = t })
		nameInput.Text = s.newName
		urlInput := widget.NewInput("https://api.example.com/v1", func(t string) { s.newURL = t })
		urlInput.Text = s.newURL
		modelsInput := widget.NewInput("model-a,model-b", func(t string) { s.newModels = t })
		modelsInput.Text = s.newModels

		addForm = widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(8, 6, 8, 8),
				BackgroundColor: types.ColorRef(245, 247, 250),
				BorderRadius:    6,
				FlexDirection:   "column", Gap: 4},
			widget.NewText("新增服务商", types.ColorFromRGB(48, 49, 51)),
			row(lbl("名称:"), w200(nameInput)),
			row(lbl("接口地址:"), w200(urlInput)),
			row(lbl("模型(逗号分隔):"), w200(modelsInput)),
			widget.HBox(
				widget.SpacerDiv(),
				widget.NewButton("取消", func() { s.adding = false; s.SetState() }).
					WithColor(types.ColorFromRGB(244, 244, 245)).
					WithTextColor(types.ColorFromRGB(96, 98, 102)),
				widget.Div(widget.Style{Width: 8}),
				widget.NewButton("确认添加", s.confirmAdd).
					WithColor(types.ColorFromRGB(64, 158, 255)).
					WithMinWidth(100),
			),
		)
	}

	// ── 服务商区块 ──
	configBody := widget.Div(
		widget.Style{FlexDirection: "column", Gap: 6},
		row(lbl("服务商:"), providerSel),
		widget.HBox(
			widget.SpacerDiv(),
			widget.NewLink("+ 新增", func() { s.adding = true; s.SetState() }).WithType(widget.LinkPrimary),
		),
		addForm,
		row(lbl("接口地址:"), w200(baseURLInput)),
		row(lbl("API 密钥:"), w200(apiKeyInput)),
		row(lbl("模型:"), modelSel),
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
		widget.Style{FlexDirection: "column", Gap: 6},
		row(lbl("监听端口:"), w200(portInput)),
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

	// ── 完整布局 ──
	body := widget.Div(
		widget.Style{Padding: types.EdgeInsets(16), FlexDirection: "column", Gap: 6},
		widget.HBox(widget.NewText(statusText, statusColor), widget.SpacerDiv(), cfgLine),
		widget.NewText("── 服务商 ──", types.ColorFromRGB(144, 147, 153)),
		configBody,
		widget.NewText("── 服务器 ──", types.ColorFromRGB(144, 147, 153)),
		serverBody,
		widget.SpacerDiv(),
		widget.NewText(fmt.Sprintf("Web IDE — http://localhost:%d", s.serverPort), types.ColorFromRGB(144, 147, 153)),
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
	cfg.Height = 620
	cfg.Resizable = true
	cfg.Borderless = true

	application.Ready = func() {
		trayID := application.AddTray("PairCode Web IDE", "", func() { openBrowser(9090) })
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
		application.SetTitleBar(36, 92)
		application.EnableWindowEffects()
	}

	if err := application.Run(cfg); err != nil {
		log.Fatalf("启动面板退出: %v", err)
	}
	StopWebServer()
	log.Println("PairCode 启动面板已退出。")
}
