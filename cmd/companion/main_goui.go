// PairCode 启动面板 —— goui 桌面 GUI。
// 深色主题 + 自定义标题栏 + 系统托盘。
// 服务商配置（Select 选择 + 新增）+ 环境检测 + 端口检测。
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

// ── 深色主题 ──

func init() {
	th := widget.DefaultTheme()
	th.BGColor = types.ColorFromRGB(26, 27, 30)         // #1a1b1e
	th.SurfaceColor = types.ColorFromRGB(37, 38, 43)    // #25262b
	th.TextColor = types.ColorFromRGB(228, 228, 231)    // #e4e4e7
	th.SecondaryText = types.ColorFromRGB(161, 161, 170) // #a1a1aa
	th.PrimaryColor = types.ColorFromRGB(99, 102, 241)   // #6366f1 紫蓝
	th.SuccessColor = types.ColorFromRGB(52, 211, 153)   // #34d399 翠绿
	th.ErrorColor = types.ColorFromRGB(239, 68, 68)      // #ef4444
	th.WarningColor = types.ColorFromRGB(251, 191, 36)   // #fbbf24
	th.BorderColor = types.ColorFromRGB(53, 55, 64)      // #353740
	th.DividerColor = types.ColorFromRGB(39, 39, 42)     // #27272a
	th.FillColor = types.ColorFromRGB(37, 38, 43)
	th.TextRegular = types.ColorFromRGB(161, 161, 170)
	th.PlaceholderColor = types.ColorFromRGB(113, 113, 122)
	th.Input.BGColor = types.ColorFromRGB(30, 31, 36)
	th.Input.TextColor = types.ColorFromRGB(228, 228, 231)
	th.Input.BorderColor = types.ColorFromRGB(53, 55, 64)
	th.Input.FocusBorderColor = types.ColorFromRGB(99, 102, 241)
	th.Input.PlaceholderColor = types.ColorFromRGB(113, 113, 122)
	th.Button.DefaultColor = types.ColorFromRGB(99, 102, 241)
	th.Button.TextColor = types.ColorFromRGB(255, 255, 255)
	widget.SetTheme(th)

	widget.SetSelectTheme(
		types.ColorFromRGB(30, 31, 36),
		types.ColorFromRGB(228, 228, 231),
		types.ColorFromRGB(53, 55, 64),
		types.ColorFromRGB(39, 39, 42),
		types.ColorFromRGB(113, 113, 122),
	)
	widget.SetMenuTheme(
		types.ColorFromRGB(37, 38, 43),
		types.ColorFromRGB(228, 228, 231),
		types.ColorFromRGB(39, 39, 42),
		types.ColorFromRGB(53, 55, 64),
		types.ColorFromRGB(113, 113, 122),
	)
	widget.SetDialogTheme(
		types.ColorFromRGB(37, 38, 43),
		types.ColorFromRGB(228, 228, 231),
		types.ColorFromRGB(161, 161, 170),
	)
}

// 已知服务商默认接口地址。
var defaultBaseURLs = map[string]string{
	"deepseek":          "https://api.deepseek.com/v1",
	"openai":            "https://api.openai.com/v1",
	"anthropic":         "https://api.anthropic.com/v1",
	"openai-compatible": "",
}

// 环境检测工具列表。
var envTools = []struct {
	name string
	cmd  string
}{
	{"Git", "git"}, {"Go", "go"}, {"Node.js", "node"},
	{"npm", "npm"}, {"Python", "python"}, {"uv", "uv"},
	{"GCC", "gcc"}, {"Rust", "rustc"},
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
	envResults []envResult
	portStatus string
	portOK     bool
	err        string
}

type envResult struct {
	name string
	ok   bool
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
	s.envResults = make([]envResult, 0, len(envTools))
	for _, t := range envTools {
		_, err := exec.LookPath(t.cmd)
		s.envResults = append(s.envResults, envResult{name: t.name, ok: err == nil})
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
		s.msg = "已保存，但密钥或接口地址为空"
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
		s.msg, s.msgOK = "服务商名称不能为空", false
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

	primary := types.ColorFromRGB(99, 102, 241)
	green := types.ColorFromRGB(52, 211, 153)
	red := types.ColorFromRGB(239, 68, 68)
	textCol := types.ColorFromRGB(228, 228, 231)
	muted := types.ColorFromRGB(161, 161, 170)
	warning := types.ColorFromRGB(251, 191, 36)
	inputBg := types.ColorFromRGB(30, 31, 36)

	// 状态栏
	statusColor := green
	statusText := "● Running"
	if !running {
		statusColor = muted
		statusText = "● Stopped"
	}
	var cfgLine widget.Widget
	if configured {
		cfgLine = widget.NewText(fmt.Sprintf("%s · %s", s.provider, core.MainModel()), green)
	} else {
		cfgLine = widget.NewText("未配置 API Key", warning)
	}

	// ── 控件 ──
	providerSel := widget.NewSelect(s.provOpts).
		WithPlaceholder("选择服务商").
		WithValue(s.provider).
		WithOnChanged(s.onProviderChange).
		WithWidth(220)

	baseURLInput := widget.NewInput("https://api.xxx.com/v1", func(t string) { s.baseURL = t })
	baseURLInput.Text = s.baseURL
	baseURLInput.BGColor = inputBg

	apiKeyInput := widget.NewInput("sk-...", func(t string) { s.apiKey = t })
	apiKeyInput.Text = s.apiKey
	apiKeyInput.BGColor = inputBg

	portInput := widget.NewInput("9090", func(t string) {
		if t != "" { fmt.Sscanf(t, "%d", &s.serverPort) }
	})
	portInput.Text = fmt.Sprintf("%d", s.serverPort)
	portInput.BGColor = inputBg

	// ── 提示 ──
	var msgEl widget.Widget
	if s.msg != "" {
		c := green
		if !s.msgOK { c = red }
		msgEl = widget.NewText(s.msg, c)
	}
	var errEl widget.Widget
	if s.err != "" {
		errEl = widget.NewText("错误: "+s.err, red)
	}

	// ── 辅助 ──
	label := func(t string) widget.Widget {
		return widget.NewText(t, muted)
	}
	w220 := func(w widget.Widget) widget.Widget {
		return widget.Div(widget.Style{Width: 220}, w)
	}
	row := func(l, c widget.Widget) widget.Widget {
		return widget.HBox(l, widget.SpacerDiv(), c)
	}

	// ── 环境检测行 ──
	var envLine widget.Widget
	{
		var items []widget.Widget
		for _, r := range s.envResults {
			ico, col := "✗", red
			if r.ok { ico, col = "✓", green }
			items = append(items, widget.NewText(ico+" "+r.name, col))
		}
		n := len(items)
		l1 := widget.HBox(items[:imin(4, n)]...)
		var l2 widget.Widget
		if n > 4 { l2 = widget.HBox(items[4:imin(8, n)]...) }
		envLine = widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4), FlexDirection: "column", Gap: 4},
			l1, l2,
		)
	}

	// ── 端口状态 ──
	psWidget := widget.NewText(s.portStatus, green)
	if !s.portOK && s.portStatus != "" {
		psWidget = widget.NewText(s.portStatus, red)
	}

	// ── 区块标题 ──
	sectionTitle := func(t string) widget.Widget {
		return widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 2)},
			widget.NewText(t, muted),
		)
	}
	cardDiv := func(children ...widget.Widget) widget.Widget {
		args := []interface{}{widget.Style{
			BackgroundColor: types.ColorRef(30, 31, 36),
			BorderRadius:    8,
			BorderColor:     types.ColorRef(53, 55, 64),
			BorderWidth:     1,
			Padding:         types.EdgeInsets(12),
			FlexDirection:   "column", Gap: 8,
		}}
		for _, c := range children { args = append(args, c) }
		return widget.Div(args...)
	}

	// ── 新增服务商 ──
	var addPanel widget.Widget
	if s.adding {
		nIn := widget.NewInput("deepseek-vllm", func(t string) { s.newName = t })
		nIn.Text = s.newName; nIn.BGColor = inputBg
		uIn := widget.NewInput("https://api.example.com/v1", func(t string) { s.newURL = t })
		uIn.Text = s.newURL; uIn.BGColor = inputBg
		tagIn := widget.NewInputTag(s.newModels...)
		tagIn.Placeholder = "输入模型名，回车添加"
		tagIn.OnChange = func(t []string) { s.newModels = t }
		addArgs := []interface{}{widget.Style{
			BackgroundColor: types.ColorRef(37, 38, 43),
			BorderRadius:    8, Padding: types.EdgeInsets(12),
			FlexDirection: "column", Gap: 8,
		},
			widget.NewText("新增服务商", textCol),
			row(label("名称"), w220(nIn)),
			row(label("API URL"), w220(uIn)),
			row(label("模型"), tagIn),
			widget.HBox(widget.SpacerDiv(),
				widget.NewButton("取消", func() { s.adding = false; s.SetState() }).
					WithColor(types.ColorFromRGB(53, 55, 64)).WithTextColor(textCol),
				widget.Div(widget.Style{Width: 8}),
				widget.NewButton("添加", s.confirmAdd).WithColor(primary).WithMinWidth(100),
			),
		}
		addPanel = widget.Div(addArgs...)
	}

	// ── 服务商卡片 ──
	provCard := cardDiv(
		row(label("服务商"), providerSel),
		row(label("接口地址"), w220(baseURLInput)),
		row(label("API 密钥"), w220(apiKeyInput)),
		widget.HBox(widget.SpacerDiv(),
			widget.NewLink("+ 新增服务商", func() { s.adding = true; s.SetState() }),
			widget.Div(widget.Style{Width: 8}),
			widget.NewButton("保存", s.saveConfig).WithColor(primary).WithMinWidth(80),
		),
		msgEl,
	)

	// ── 服务器卡片 ──
	serverCard := cardDiv(
		row(label("端口"), w220(portInput)),
		widget.HBox(widget.SpacerDiv(),
			widget.NewButton("检测端口", s.checkPort).
				WithColor(types.ColorFromRGB(53, 55, 64)).WithTextColor(primary),
		),
		psWidget,
		widget.NewButton(func() string {
			if running { return "停止服务器" }
			return "启动服务器"
		}(), s.toggleServer).
			WithColor(func() types.Color {
				if running { return red }
				return primary
			}()).
			WithMinWidth(140),
		errEl,
	)

	// ── 完整布局 ──
	body := widget.Div(
		widget.Style{Padding: types.EdgeInsets(16), FlexDirection: "column", Gap: 10},
		widget.HBox(widget.NewText(statusText, statusColor), widget.SpacerDiv(), cfgLine),
		sectionTitle("服务商"),
		provCard,
		addPanel,
		sectionTitle("环境"),
		cardDiv(envLine),
		sectionTitle("服务器"),
		serverCard,
		widget.SpacerDiv(),
		widget.NewText(fmt.Sprintf("http://localhost:%d  —  PairCode IDE", s.serverPort), muted),
	)

	return widget.VBox(
		titleBar(),
		&widget.Container{
			Height:     1,
			Background: &widget.PaintWidget{Color: ptrColor(53, 55, 64)},
		},
		&widget.Expanded{
			SingleChildWidget: widget.SingleChildWidget{Child: body},
			Flex:              1,
		},
	)
}

// ── 自定义标题栏 ──

func titleBar() widget.Widget {
	primary := types.ColorFromRGB(99, 102, 241)
	tc := types.ColorFromRGB(228, 228, 231)

	return widget.Div(
		widget.Style{Height: 36, BackgroundColor: types.ColorRef(26, 27, 30)},
		widget.HBox(
			&widget.Expanded{
				SingleChildWidget: widget.SingleChildWidget{
					Child: widget.Div(
						widget.Style{Padding: types.EdgeInsetsLTRB(14, 0, 0, 0), Height: 36},
						widget.HBox(
							widget.NewText("◆", primary),
							widget.Div(widget.Style{Width: 8}),
							widget.NewText("PairCode", tc),
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
		Color: types.ColorFromRGB(26, 27, 30), TextColor: types.ColorFromRGB(161, 161, 170),
		MinWidth: 46, MinHeight: 36,
	}
}

func ptrColor(r, g, b uint8) *types.Color {
	c := types.ColorFromRGB(r, g, b)
	return &c
}

func imin(a, b int) int {
	if a < b { return a }
	return b
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
	cfg.Height = 640
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
