// PairCode IDE 启动面板 — goui 桌面 GUI。
// 深色科技风 + 自定义标题栏 + 系统托盘。
// 仅维护 models.json（服务商配置），不设置当前模型。
// 环境检测：git / python / uv / npm。
//
//go:build windows

package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"
	"time"

	"github.com/hoonfeng/goui/pkg/app"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
	"github.com/hoonfeng/paircode/cmd/companion/core"

	_ "github.com/hoonfeng/goui/pkg/platform"
)

// ── 科技风深色主题 ──

func init() {
	th := widget.DefaultTheme()
	th.BGColor = types.ColorFromRGB(13, 17, 23)
	th.SurfaceColor = types.ColorFromRGB(22, 27, 34)
	th.TextColor = types.ColorFromRGB(240, 246, 252)
	th.SecondaryText = types.ColorFromRGB(139, 148, 158)
	th.PrimaryColor = types.ColorFromRGB(88, 166, 255)
	th.SuccessColor = types.ColorFromRGB(63, 185, 80)
	th.ErrorColor = types.ColorFromRGB(248, 81, 73)
	th.WarningColor = types.ColorFromRGB(210, 153, 34)
	th.BorderColor = types.ColorFromRGB(48, 54, 61)
	th.DividerColor = types.ColorFromRGB(33, 38, 45)
	th.FillColor = types.ColorFromRGB(22, 27, 34)
	th.TextRegular = types.ColorFromRGB(139, 148, 158)
	th.PlaceholderColor = types.ColorFromRGB(113, 120, 128)
	th.Input.BGColor = types.ColorFromRGB(13, 17, 23)
	th.Input.TextColor = types.ColorFromRGB(240, 246, 252)
	th.Input.BorderColor = types.ColorFromRGB(48, 54, 61)
	th.Input.FocusBorderColor = types.ColorFromRGB(88, 166, 255)
	th.Input.PlaceholderColor = types.ColorFromRGB(113, 120, 128)
	th.Button.DefaultColor = types.ColorFromRGB(88, 166, 255)
	th.Button.TextColor = types.ColorFromRGB(255, 255, 255)
	widget.SetTheme(th)

	widget.SetSelectTheme(
		types.ColorFromRGB(13, 17, 23),
		types.ColorFromRGB(240, 246, 252),
		types.ColorFromRGB(48, 54, 61),
		types.ColorFromRGB(33, 38, 45),
		types.ColorFromRGB(113, 120, 128),
	)
	widget.SetMenuTheme(
		types.ColorFromRGB(22, 27, 34),
		types.ColorFromRGB(240, 246, 252),
		types.ColorFromRGB(33, 38, 45),
		types.ColorFromRGB(48, 54, 61),
		types.ColorFromRGB(113, 120, 128),
	)
	widget.SetDialogTheme(
		types.ColorFromRGB(22, 27, 34),
		types.ColorFromRGB(240, 246, 252),
		types.ColorFromRGB(139, 148, 158),
	)
}

var defaultBaseURLs = map[string]string{
	"deepseek":          "https://api.deepseek.com/v1",
	"openai":            "https://api.openai.com/v1",
	"anthropic":         "https://api.anthropic.com/v1",
	"openai-compatible": "",
}

var envChecks = []struct {
	name string
	cmd  string
}{
	{"Git", "git"},
	{"Python", "python"},
	{"uv", "uv"},
	{"npm", "npm"},
}

// ── 有状态根组件 ──

type PanelRoot struct{ widget.StatefulWidget }

func (p *PanelRoot) CreateState() widget.State { return &panelState{} }

type panelState struct {
	widget.BaseState
	provider   string
	baseURL    string
	apiKey     string
	provOpts   []widget.SelectOption
	serverPort int
	msg        string
	msgOK      bool
	adding     bool
	newName    string
	newURL     string
	newModels  []string
	envOK      map[string]bool
	portStatus string
	portOK     bool
	err        string
}

func (s *panelState) InitState() {
	_ = InitCore()
	s.provider = core.Settings.Provider
	s.baseURL = core.Settings.BaseURL
	s.apiKey = core.Settings.APIKey
	s.serverPort = 9090
	s.refreshProviders()
	s.checkEnv()
}

func (s *panelState) refreshProviders() {
	providers := core.GetProviders()
	s.provOpts = make([]widget.SelectOption, 0, len(providers))
	for _, p := range providers {
		s.provOpts = append(s.provOpts, widget.SelectOption{Label: p, Value: p})
	}
}

func (s *panelState) checkEnv() {
	s.envOK = make(map[string]bool, len(envChecks))
	for _, e := range envChecks {
		_, err := exec.LookPath(e.cmd)
		s.envOK[e.name] = err == nil
	}
}

func (s *panelState) checkPort() {
	p := s.serverPort
	if p <= 0 || p > 65535 {
		s.portStatus = "端口号无效"
		s.portOK = false
		s.SetState()
		return
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", p), 2*time.Second)
	if err == nil {
		conn.Close()
		s.portStatus = fmt.Sprintf("端口 %d 已被占用", p)
		s.portOK = false
	} else {
		s.portStatus = fmt.Sprintf("端口 %d 可用", p)
		s.portOK = true
	}
	s.SetState()
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
	s.SetState()
}

func (s *panelState) saveConfig() {
	core.Settings.Provider = s.provider
	core.Settings.BaseURL = s.baseURL
	core.Settings.APIKey = s.apiKey
	core.Save()
	if core.Configured() {
		s.msg = "配置已保存"
		s.msgOK = true
	} else {
		s.msg = "已保存，但密钥或地址为空"
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

func (s *panelState) confirmAdd() {
	if s.newName == "" {
		s.msg, s.msgOK = "名称不能为空", false
		s.SetState()
		return
	}
	if err := core.AddProvider(s.newName, s.newModels); err != nil {
		s.msg, s.msgOK = "添加失败: "+err.Error(), false
		s.SetState()
		return
	}
	defaultBaseURLs[s.newName] = s.newURL
	s.provider = s.newName
	s.baseURL = s.newURL
	s.refreshProviders()
	s.adding = false
	s.newName, s.newURL = "", ""
	s.newModels = nil
	s.msg, s.msgOK = fmt.Sprintf("服务商 %q 已添加", s.provider), true
	s.SetState()
}

// ── Build ──

func (s *panelState) Build(ctx widget.BuildContext) widget.Widget {
	running := IsWebServerRunning()
	configured := core.Configured()

	blue := types.ColorFromRGB(88, 166, 255)
	green := types.ColorFromRGB(63, 185, 80)
	red := types.ColorFromRGB(248, 81, 73)
	text := types.ColorFromRGB(240, 246, 252)
	muted := types.ColorFromRGB(139, 148, 158)
	warn := types.ColorFromRGB(210, 153, 34)
	cardBg := types.ColorRef(22, 27, 34)
	borderClr := ptrColor(48, 54, 61)
	btnDim := types.ColorFromRGB(48, 54, 61)

	// 状态栏
	sc := green
	st := "● Running"
	if !running {
		sc = muted
		st = "● Stopped"
	}

	// ── 控件 ──
	providerSel := widget.NewSelect(s.provOpts).
		WithPlaceholder("选择服务商").
		WithValue(s.provider).
		WithOnChanged(s.onProviderChange).
		WithWidth(220)
	baseURLInput := widget.NewInput("https://api.xxx.com/v1", func(t string) { s.baseURL = t })
	baseURLInput.Text = s.baseURL
	apiKeyInput := widget.NewInput("sk-...", func(t string) { s.apiKey = t })
	apiKeyInput.Text = s.apiKey
	portInput := widget.NewInput("9090", func(t string) {
		if t != "" { fmt.Sscanf(t, "%d", &s.serverPort) }
	})
	portInput.Text = fmt.Sprintf("%d", s.serverPort)

	var msgEl widget.Widget
	if s.msg != "" {
		c := green
		if !s.msgOK { c = red }
		msgEl = widget.NewText(s.msg, c)
	}
	var errEl widget.Widget
	if s.err != "" { errEl = widget.NewText(s.err, red) }

	// 辅助
	lb := func(t string) widget.Widget { return widget.NewText(t, muted) }
	w220 := func(w widget.Widget) widget.Widget { return widget.Div(widget.Style{Width: 220}, w) }
	rw := func(l, c widget.Widget) widget.Widget { return widget.HBox(l, widget.SpacerDiv(), c) }

	// card 工厂
	card := func(ch ...widget.Widget) widget.Widget {
		args := []interface{}{
			widget.Style{
				BackgroundColor: cardBg,
				BorderRadius:    8,
				BorderColor:     borderClr,
				BorderWidth:     1,
				Padding:         types.EdgeInsets(12),
				FlexDirection:   "column", Gap: 8,
			},
		}
		for _, c2 := range ch { args = append(args, c2) }
		return widget.Div(args...)
	}

	// ── 新增面板 ──
	var addPanel widget.Widget
	if s.adding {
		nIn := widget.NewInput("my-provider", func(t string) { s.newName = t })
		nIn.Text = s.newName
		uIn := widget.NewInput("https://api.example.com/v1", func(t string) { s.newURL = t })
		uIn.Text = s.newURL
		tagIn := widget.NewInputTag(s.newModels...)
		tagIn.Placeholder = "输入模型名，回车添加"
		tagIn.OnChange = func(t []string) { s.newModels = t }

		addPanel = widget.Div(
			widget.Style{
				BackgroundColor: types.ColorRef(13, 17, 23),
				BorderRadius:    8,
				BorderColor:     borderClr,
				BorderWidth:     1,
				Padding:         types.EdgeInsets(12),
				FlexDirection:   "column", Gap: 8,
			},
			widget.NewText("新增服务商", text),
			rw(lb("名称"), w220(nIn)),
			rw(lb("API地址"), w220(uIn)),
			tagIn,
			widget.HBox(widget.SpacerDiv(),
				widget.NewButton("取消", func() { s.adding = false; s.SetState() }).
					WithColor(btnDim).WithTextColor(text),
				widget.Div(widget.Style{Width: 8}),
				widget.NewButton("添加", s.confirmAdd).
					WithColor(blue).WithMinWidth(100),
			),
		)
	}

	// ── 环境检测 ──
	var envItems []widget.Widget
	for _, e := range envChecks {
		ico, col := "✗", red
		if s.envOK[e.name] { ico, col = "✓", green }
		envItems = append(envItems, widget.NewText(ico+" "+e.name, col))
	}
	envLine := widget.HBox(envItems...)

	// ── 端口状态 ──
	psW := widget.NewText(s.portStatus, green)
	if !s.portOK && s.portStatus != "" { psW = widget.NewText(s.portStatus, red) }

	// ── 区域 ──
	sectionTitle := func(t string) widget.Widget {
		return widget.NewText(t, muted)
	}
	

	body := widget.Div(
		widget.Style{Padding: types.EdgeInsets(16), FlexDirection: "column", Gap: 10},
		widget.HBox(widget.NewText(st, sc), widget.SpacerDiv(),
			func() widget.Widget {
				if configured { return widget.NewText(s.provider, green) }
				return widget.NewText("未配置 API Key", warn)
			}(),
		),
		// SERVICE PROVIDER
		sectionTitle("SERVICE PROVIDER"),
		card(
			rw(lb("Provider"), providerSel),
			rw(lb("Base URL"), w220(baseURLInput)),
			rw(lb("API Key"), w220(apiKeyInput)),
			widget.HBox(widget.SpacerDiv(),
				widget.NewButton("+ New", func() { s.adding = true; s.SetState() }).
					WithColor(btnDim).WithTextColor(blue),
				widget.Div(widget.Style{Width: 8}),
				widget.NewButton("Save", s.saveConfig).WithColor(blue).WithMinWidth(80),
			),
			msgEl,
		),
		addPanel,
		// ENVIRONMENT
		sectionTitle("ENVIRONMENT"),
		card(envLine),
		// SERVER
		sectionTitle("SERVER"),
		card(
			rw(lb("Port"), w220(portInput)),
			widget.HBox(widget.SpacerDiv(),
				widget.NewButton("Check Port", s.checkPort).
					WithColor(btnDim).WithTextColor(blue),
			),
			psW,
			widget.NewButton(func() string {
				if running { return "Stop Server" }
				return "Start Server"
			}(), s.toggleServer).
				WithColor(func() types.Color {
					if running { return red }
					return blue
				}()).
				WithMinWidth(140),
			errEl,
		),
		widget.SpacerDiv(),
		widget.NewText(fmt.Sprintf("http://localhost:%d  —  PairCode IDE", s.serverPort), muted),
	)

	return widget.VBox(
		titleBar(),
		&widget.Container{Height: 1, Background: &widget.PaintWidget{Color: ptrColor(48, 54, 61)}},
		&widget.Expanded{SingleChildWidget: widget.SingleChildWidget{Child: body}, Flex: 1},
	)
}

// ── 标题栏 ──

func titleBar() widget.Widget {
	blue := types.ColorFromRGB(88, 166, 255)
	tc := types.ColorFromRGB(240, 246, 252)
	return widget.Div(
		widget.Style{Height: 36, BackgroundColor: types.ColorRef(13, 17, 23)},
		widget.HBox(
			&widget.Expanded{
				SingleChildWidget: widget.SingleChildWidget{
					Child: widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(14, 0, 0, 0), Height: 36},
						widget.HBox(
							widget.NewText("◈", blue),
							widget.Div(widget.Style{Width: 8}),
							widget.NewText("PairCode IDE", tc),
						),
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
		Color: types.ColorFromRGB(13, 17, 23), TextColor: types.ColorFromRGB(139, 148, 158),
		MinWidth: 46, MinHeight: 36,
	}
}

func ptrColor(r, g, b uint8) *types.Color {
	c := types.ColorFromRGB(r, g, b)
	return &c
}

func openBrowser(port int) {
	url := fmt.Sprintf("http://localhost:%d", port)
	go func() { exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start() }()
}

// ── main ──

var application *app.Application

func main() {
	runtime.LockOSThread()
	application = app.NewApplication()
	application.SetRootWidget(&PanelRoot{})

	cfg := app.DefaultConfig()
	cfg.Title = "PairCode IDE"
	cfg.Width = 480
	cfg.Height = 620
	cfg.Resizable = true
	cfg.Borderless = true

	application.Ready = func() {
		trayID := application.AddTray("PairCode IDE", "", func() { openBrowser(9090) })
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
				case 1: openBrowser(9090)
				case 2:
					if IsWebServerRunning() { StopWebServer() } else { StartWebServer(9090) }
					if application.Pipeline != nil { application.Pipeline.MarkNeedsLayout() }
				case 3:
					if application.Window != nil { application.Window.Show() }
				case 9: application.Quit()
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
	log.Println("PairCode IDE 已退出。")
}
