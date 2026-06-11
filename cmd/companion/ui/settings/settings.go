// Package settingspanel 是设置面板（模态对话框 + 顶部 tab）。设置数据层(core.AppSettings)在 core 包。
// 本文件含:设置面板 UI(settingsBodyState/各 tab) + 字体控件 + 字段绑定组件。
//
//go:build windows

package settingspanel

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/user/gou-ide/cmd/companion/agent"
	"github.com/user/gou-ide/cmd/companion/ui/config"
	"github.com/user/gou-ide/cmd/companion/ui/chat"
	"github.com/user/gou-ide/cmd/companion/core"
	"github.com/user/gou-ide/cmd/companion/ui/editor"
	"github.com/user/gou-ide/cmd/companion/ui/mcp"
	"github.com/user/gou-ide/cmd/companion/ui/mdview"
	"github.com/user/gou-ide/cmd/companion/ui/skills"
	"github.com/user/gou-ide/cmd/companion/ui/sysfont"
	"github.com/user/gou-ide/cmd/companion/ui/terminal"
	"github.com/user/gou-ide/cmd/companion/ui/theme"
	"github.com/user/gou-ide/cmd/companion/ui"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// ─── apply — 保存/启动时调用的应用逻辑（碰面板/ui）──

// ApplyAgentSettings 把持久化设置应用到运行态（启动时若有存档 + 保存后调用）。
func ApplyAgentSettings() {
	chatpanel.TheState.AutoReview = !core.Settings.RequireApproval
	chatpanel.TheState.Autonomous = core.Settings.Autonomous
	chatpanel.TheState.AutoCollapse = core.Settings.AutoCollapse
	if core.Settings.DefaultShell != "" {
		termpanel.Active().SetShell(core.Settings.DefaultShell)
	}
	agent.SetSearxngURL(core.Settings.SearxngURL)
	ApplyFontFamily()
}

// ApplyFontFamily 把外观设置的等宽字体族应用到自绘等宽面。
func ApplyFontFamily() {
	fam := core.FirstFontFamily(core.Settings.FontFamily)
	if fam == "" {
		fam = "Consolas"
	}
	termpanel.GridFont.Family = fam
	mdview.MonoFont.Family = fam
}

// Load 启动加载:先 core.Load() 读数据,再做 apply(碰面板/ui)。
func Load() {
	if core.Load() {
		ApplyAgentSettings()
	} else {
		agent.SetSearxngURL(core.Settings.SearxngURL)
		ApplyFontFamily()
	}
	widget.LoadCustomProviders(core.Settings.CustomProviders)
}

// ApplyIgnoreDirs 由 main 注入（调用 agent_bridge.go 的 applyIgnoreDirs）。
var ApplyIgnoreDirs func(root string)

// ─── 编辑缓冲 & 数据 ──

var (
	EditingSettings     core.AppSettings // 对话框编辑副本
	editingInstructions string
)

// 字段指针表（用于声明式设置绑定）。
var (
	BoolFields map[string]*bool
	IntFields  map[string]*int
	StrFields  map[string]*string
	ListFields map[string]*[]string
)

func InitFieldPtrs() {
	e := &EditingSettings
	BoolFields = map[string]*bool{
		"AutoIterate": &e.AutoIterate, "RequireApproval": &e.RequireApproval, "AIReview": &e.AIReview,
		"Benchmark": &e.Benchmark, "LuaTools": &e.LuaTools, "HideMinimap": &e.HideMinimap,
		"Autonomous": &e.Autonomous, "AutoCollapse": &e.AutoCollapse, "CompressEnabled": &e.CompressEnabled,
		"EditorFontBold": &e.EditorFontBold, "AutoConnectMCP": &e.AutoConnectMCP, "PhilosophyEnabled": &e.PhilosophyEnabled,
	}
	IntFields = map[string]*int{
		"MaxIterations": &e.MaxIterations, "MaxParallel": &e.MaxParallel, "ReviewRetries": &e.ReviewRetries,
		"TermFontSize": &e.TermFontSize, "EditorFontSize": &e.EditorFontSize, "MaxTokens": &e.MaxTokens,
		"ContextMaxTokens": &e.ContextMaxTokens,
	}
	StrFields = map[string]*string{
		"Theme": &e.Theme, "DefaultShell": &e.DefaultShell, "TermEncoding": &e.TermEncoding,
		"SearxngURL": &e.SearxngURL, "SystemInstructions": &e.SystemInstructions, "ThinkingMode": &e.ThinkingMode,
		"Provider": &e.Provider, "BaseURL": &e.BaseURL, "APIKey": &e.APIKey,
	}
	ListFields = map[string]*[]string{"IgnoreDirs": &e.IgnoreDirs}
}

// ─── 哲学数据（core.go 引用）──

var Philosophies = []struct{ ID, Name string }{
	{"tao-te-ching", "道德经"}, {"huangdi-yinfu-jing", "黄帝阴符经"}, {"sunzi-bingfa", "孙子兵法"},
}
var RoleEntries = []struct{ ID, Name string }{
	{"planner", "规划 Agent"}, {"implementer", "执行 Agent"}, {"explorer", "探索 Agent"},
	{"reviewer", "审核 Agent"}, {"verifier", "验证 Agent"}, {"debugger", "调试 Agent"}, {"general", "通用 Agent"},
}

func Contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func Toggle(ss []string, s string) []string {
	out := ss[:0:0]
	found := false
	for _, x := range ss {
		if x == s {
			found = true
			continue
		}
		out = append(out, x)
	}
	if !found {
		out = append(out, s)
	}
	return out
}

// PhilosophyPrompt 主执行 Agent 的哲学。
func PhilosophyPrompt() string {
	return classicsPhilosophy() + roleSpecific("general") + roleSpecific("implementer")
}

// ─── 角色提示（core.go 用）──

func RoleDisplayName(roleID string) string {
	for _, r := range RoleEntries {
		if r.ID == roleID {
			return r.Name
		}
	}
	return "通用 Agent"
}

func RolePhilosophy(roleID string) string {
	return classicsPhilosophy() + roleSpecific(roleID)
}

// ─── 设置对话框 ────────────────────────────────

var SettingsTabs = []struct{ ID, Label string }{
	{"model", "模型"}, {"agent", "Agent"}, {"instructions", "指令"}, {"appearance", "外观"},
	{"terminal", "终端"}, {"philosophy", "思想"}, {"mcp", "MCP"}, {"skills", "Skills"},
}

var theBody = &bodyState{tab: "model"}

type SettingsBody struct{ widget.StatefulWidget }
func (b *SettingsBody) CreateState() widget.State { return theBody }

type bodyState struct {
	widget.BaseState
	tab          string
	resetTok     int
	editingRole  string
	secCollapsed map[string]bool
}

func (b *bodyState) toggleSec(key string) {
	if b.secCollapsed == nil {
		b.secCollapsed = map[string]bool{}
	}
	b.secCollapsed[key] = !b.secCollapsed[key]
}

// OpenDialog 打开设置模态对话框。
func OpenDialog() {
	EditingSettings = core.Settings
	if EditingSettings.Provider == "" {
		EditingSettings.Provider = "deepseek"
	}
	if EditingSettings.BaseURL == "" {
		_, _, EditingSettings.BaseURL, _ = providerByID(EditingSettings.Provider)
	}
	if EditingSettings.Model == "" {
		EditingSettings.Model = defaultModelFor(EditingSettings.Provider)
	}
	editingInstructions = loadInstructions()
	theBody.tab = "model"
	theBody.resetTok++
	var id int
	dlg := widget.NewDialog("设置", &SettingsBody{}).WithWidth(580).WithTransition("fade").WithFooter(
		ui.Btn("取消", func() { widget.HideOverlay(id) }),
		ui.PrimaryBtn("保存", func() {
			core.Settings = EditingSettings
			core.Save()
			saveInstructions(editingInstructions)
			core.Loaded = true
			ApplyAgentSettings()
			theme.Apply(core.Settings.Theme)
			if ApplyIgnoreDirs != nil { ApplyIgnoreDirs(core.Root()) }
			editorpanel.Editor.BumpReload()
			editorpanel.Editor.SetState()
			widget.HideOverlay(id)
		}),
	)
	id = widget.ShowDialog(dlg)
}

func instructionsPath() string { return filepath.Join(core.Root(), ".pair", "rules.md") }
func loadInstructions() string { data, _ := os.ReadFile(instructionsPath()); return string(data) }
func saveInstructions(s string) {
	p := instructionsPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte(s), 0o644)
}

func (b *bodyState) Build(ctx widget.BuildContext) widget.Widget {
	tabs := make([]widget.Widget, 0, len(SettingsTabs))
	for _, t := range SettingsTabs {
		tabs = append(tabs, b.tabBtn(t.ID, t.Label))
	}
	return widget.Div(widget.Style{Width: 556, FlexDirection: "column", AlignItems: "stretch"},
		widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center", BackgroundColor: ui.BgSubtle,
			BorderColor: ui.Border, BorderWidth: 1, Padding: types.EdgeInsetsLTRB(0, 0, 0, 2)}, tabs),
		widget.Div(widget.Style{Height: 12}),
		widget.Div(widget.Style{Height: 420, FlexDirection: "column", AlignItems: "stretch",
			Padding: types.EdgeInsetsLTRB(10, 0, 10, 0)},
			ui.Expand(widget.NewScrollView(widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch",
				Padding: types.EdgeInsetsLTRB(0, 0, 14, 0)}, b.content())))))
}

func (b *bodyState) tabBtn(id, lbl string) widget.Widget {
	on := b.tab == id
	tc, bg := *ui.FgMuted, *ui.BgSubtle
	if on { tc, bg = *ui.Fg, *ui.BgMuted }
	return &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: ui.TextC(lbl, tc, 12)},
		OnClick: func() { b.tab = id; b.SetState() }, Color: bg, MinHeight: 26,
		Padding: types.EdgeInsetsLTRB(10, 0, 10, 0),
	}
}

func (b *bodyState) content() widget.Widget {
	switch b.tab {
	case "model": return b.modelTab()
	case "agent": return b.agentTab()
	case "instructions": return b.instructionsTab()
	case "appearance": return b.appearanceTab()
	case "terminal": return b.terminalTab()
	case "philosophy": return b.philosophyTab()
	case "mcp": return b.mcpTab()
	case "skills": return b.skillsTab()
	}
	return widget.Div(widget.Style{Height: 180, FlexDirection: "column", AlignItems: "center", JustifyContent: "center"},
		widget.Lucide("settings", widget.IconSize(26), widget.IconColor(*ui.FgMuted)),
		widget.Div(widget.Style{Height: 8}), ui.TextC("该设置项待接入", *ui.FgMuted, 12))
}

// ─── Tab 内容 ──

func (b *bodyState) agentTab() widget.Widget {
	return config.Build(widget.ComponentSpec{Type: "Column", Style: &widget.StyleSpec{AlignItems: "stretch"},
		Children: []widget.ComponentSpec{
			{Type: "Muted", Text: "Agent 行为"},
			{Type: "SettingsSlider", Props: map[string]any{"field": "MaxIterations", "label": "最大迭代次数", "min": 5, "max": 200, "step": 1}},
			{Type: "SettingsSlider", Props: map[string]any{"field": "MaxParallel", "label": "并行 Agent 数", "min": 1, "max": 8, "step": 1}},
			{Type: "SettingsSlider", Props: map[string]any{"field": "ReviewRetries", "label": "自动迭代上限", "min": 0, "max": 10, "step": 1}},
			{Type: "SettingsToggle", Props: map[string]any{"field": "AutoIterate", "label": "评分不足自动迭代改进"}},
			{Type: "SettingsToggle", Props: map[string]any{"field": "RequireApproval", "label": "破坏性操作需人工确认"}},
			{Type: "SettingsToggle", Props: map[string]any{"field": "AIReview", "label": "AI 审核写操作"}},
			{Type: "Muted", Text: "AI 审核模式：写/改/删/命令交审核模型自动裁决，驳回回灌建议改道。"},
			{Type: "SettingsToggle", Props: map[string]any{"field": "Benchmark", "label": "任务完成后自动评测评分"}},
			{Type: "SettingsToggle", Props: map[string]any{"field": "LuaTools", "label": "Lua 自定义工具"}},
		}})
}

func settingsTextarea(placeholder, val string, rows, tok int, onChange func(string)) widget.Widget {
	return ui.Textarea(placeholder, val, rows, tok, onChange)
}
func sectionHeader(icon, title, sub string) widget.Widget { return ui.SectionHeader(icon, title, sub) }

func (b *bodyState) instructionsTab() widget.Widget {
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		sectionHeader("file-text", "系统级指令", "（对所有项目生效，存设置中）"),
		widget.Div(widget.Style{Height: 6}),
		settingsTextarea("通用编码规范、安全规则、工作流程…", EditingSettings.SystemInstructions, 7, b.resetTok, func(t string) { EditingSettings.SystemInstructions = t }),
		widget.Div(widget.Style{Height: 14}),
		sectionHeader("book-open", "项目级指令", "（仅当前项目，存 .pair/rules.md）"),
		widget.Div(widget.Style{Height: 6}),
		settingsTextarea("项目特定的需求说明、技术栈约束…", editingInstructions, 7, b.resetTok, func(t string) { editingInstructions = t }),
		widget.Div(widget.Style{Height: 4}),
		ui.TextC("保存设置即写入；项目级与项目根的 AGENTS.md/CLAUDE.md 一并注入系统提示。", *ui.FgMuted, 10),
	)
}

func (b *bodyState) appearanceTab() widget.Widget {
	return config.Build(widget.ComponentSpec{Type: "Column", Style: &widget.StyleSpec{AlignItems: "stretch"},
		Children: []widget.ComponentSpec{
			{Type: "Muted", Text: "外观"},
			{Type: "SettingsSelect", Props: map[string]any{"field": "Theme", "label": "主题", "options": themeOptions()}},
			{Type: "VGap", Props: map[string]any{"size": 8}},
			{Type: "Muted", Text: "编辑器字体（字体 / 字号 / 粗 B / 斜 I / 下划线 U）"},
			{Type: "VGap", Props: map[string]any{"size": 4}},
			{Type: "EditorFontPicker"},
			{Type: "VGap", Props: map[string]any{"size": 12}},
			{Type: "Muted", Text: "界面字体（全部已装字体；字号各处自定）"},
			{Type: "VGap", Props: map[string]any{"size": 4}},
			{Type: "UIFontPicker"},
			{Type: "VGap", Props: map[string]any{"size": 12}},
			{Type: "SettingsToggle", Props: map[string]any{"field": "HideMinimap", "label": "启用 Minimap", "invert": true}},
		}})
}

var providerPresets = []struct {
	name, label, base string
	models            []struct{ id, name string }
}{
	{"deepseek", "DeepSeek", "https://api.deepseek.com/v1", []struct{ id, name string }{{"deepseek-v4-flash", "DeepSeek V4 Flash"}, {"deepseek-v4-pro", "DeepSeek V4 Pro"}}},
	{"openai", "OpenAI", "https://api.openai.com/v1", []struct{ id, name string }{
		{"gpt-5.5", "GPT-5.5（旗舰）"}, {"gpt-5.4", "GPT-5.4（编码）"}, {"gpt-5.4-mini", "GPT-5.4 Mini"},
		{"gpt-4.1", "GPT-4.1"}, {"gpt-4.1-mini", "GPT-4.1 Mini"}, {"gpt-4o", "GPT-4o（多模态）"},
		{"o3", "o3（推理）"}, {"o3-pro", "o3 Pro（深度推理）"}, {"o4-mini", "o4 Mini（快速推理）"}, {"o4-mini-deep-research", "o4 Mini Deep Research"},
	}},
	{"qwen", "通义千问 (Qwen)", "https://dashscope.aliyuncs.com/compatible-mode/v1", []struct{ id, name string }{
		{"qwen3.7-max", "Qwen3.7 Max（旗舰）"}, {"qwen3.6-plus", "Qwen3.6 Plus（增强）"}, {"qwen3.6-flash", "Qwen3.6 Flash（快速）"},
		{"qwen3-235b-a22b", "Qwen3 235B-A22B（MoE）"}, {"qwen-turbo-latest", "Qwen Turbo（轻量）"},
	}},
	{"zhipu", "智谱 (GLM)", "https://open.bigmodel.cn/api/paas/v4", []struct{ id, name string }{
		{"glm-5.1", "GLM-5.1（旗舰）"}, {"glm-5", "GLM-5（高智能）"}, {"glm-5-turbo", "GLM-5 Turbo（增强）"},
		{"glm-4.7", "GLM-4.7（高智能）"}, {"glm-4.7-flashx", "GLM-4.7 FlashX（高速）"}, {"glm-4.7-flash", "GLM-4.7 Flash（免费）"},
		{"glm-4.6", "GLM-4.6（超强性能）"}, {"glm-4.5-air", "GLM-4.5 Air（高性价比）"}, {"glm-4-long", "GLM-4 Long（超长上下文）"},
		{"glm-5v-turbo", "GLM-5V Turbo（多模态）"},
	}},
	{"moonshot", "月之暗面 (Kimi)", "https://api.moonshot.cn/v1", []struct{ id, name string }{
		{"kimi-k2.6", "Kimi K2.6（旗舰）"}, {"kimi-k2.5", "Kimi K2.5（增强）"}, {"kimi-k2", "Kimi K2（基础）"},
		{"kimi-k2-thinking", "Kimi K2 Thinking（深度思考）"}, {"moonshot-v1-128k", "Moonshot v1 128K"},
	}},
	{"ocat", "Ocat.run", "https://ocat.run/v1", []struct{ id, name string }{{"ocat-default", "Ocat 模型（动态加载）"}}},
	{"custom", "自定义 (OpenAI 兼容)", "http://localhost:11434/v1", nil},
}

func providerByID(id string) (name, label, base string, models []struct{ id, name string }) {
	for _, p := range providerPresets {
		if p.name == id { return p.name, p.label, p.base, p.models }
	}
	last := providerPresets[len(providerPresets)-1]
	return last.name, last.label, last.base, last.models
}

func defaultModelFor(id string) string {
	if _, _, _, models := providerByID(id); len(models) > 0 { return models[0].id }
	return ""
}

func themeOptions() []any {
	mk := func(label, value string) any { return map[string]any{"label": label, "value": value} }
	return []any{mk("暗色 (GitHub Dark)", "dark"), mk("亮色 (GitHub Light)", "light"),
		mk("高对比度", "high-contrast"), mk("暖阳 (Solarized Light)", "solarized-light"), mk("暗紫 (Dracula)", "dracula")}
}

func (b *bodyState) terminalTab() widget.Widget {
	return config.Build(widget.ComponentSpec{Type: "Column", Style: &widget.StyleSpec{AlignItems: "stretch"},
		Children: []widget.ComponentSpec{
			{Type: "Muted", Text: "终端"},
			{Type: "SettingsSelect", Props: map[string]any{"field": "DefaultShell", "label": "默认 Shell", "options": optionList("自动检测", "auto", "CMD", "cmd", "PowerShell", "powershell", "Git Bash", "gitbash")}},
			{Type: "SettingsSlider", Props: map[string]any{"field": "TermFontSize", "label": "字号", "suffix": "px", "min": 10, "max": 20, "step": 1}},
			{Type: "SettingsSelect", Props: map[string]any{"field": "TermEncoding", "label": "编码", "options": optionList("自动检测", "auto", "UTF-8", "utf-8", "GBK", "gbk")}},
		}})
}

func optionList(labelValuePairs ...string) []any {
	out := make([]any, 0, len(labelValuePairs)/2)
	for i := 0; i+1 < len(labelValuePairs); i += 2 {
		out = append(out, map[string]any{"label": labelValuePairs[i], "value": labelValuePairs[i+1]})
	}
	return out
}

func classicsPhilosophy() string {
	if !core.Settings.PhilosophyEnabled { return "" }
	var names []string
	for _, c := range Philosophies {
		if Contains(core.Settings.PhilosophySelected, c.ID) { names = append(names, c.Name) }
	}
	if len(names) == 0 { return "" }
	return "\n\n# 指导思想\n" + strings.Join(names, "、") + "。"
}

func roleSpecific(roleID string) string {
	if !core.Settings.PhilosophyEnabled { return "" }
	c := roleCustomPhilosophy(roleID)
	if c == "" { return "" }
	return "\n\n## " + RoleDisplayName(roleID) + "哲学\n" + c
}

func roleCustomPhilosophy(roleID string) string {
	var parts []string
	if data, err := os.ReadFile(filepath.Join(core.ConfigDir(), "philosophy", roleID+".txt")); err == nil {
		if s := strings.TrimSpace(string(data)); s != "" { parts = append(parts, s) }
	}
	if c := strings.TrimSpace(core.Settings.PhilosophyRoles[roleID]); c != "" { parts = append(parts, c) }
	return strings.Join(parts, "\n\n")
}

func (b *bodyState) philosophyTab() widget.Widget {
	kids := []widget.Widget{
		ui.TextC("指导思想", *ui.FgMuted, 11),
		settingsToggle("启用主 Agent 哲学指导", EditingSettings.PhilosophyEnabled, func() {
			EditingSettings.PhilosophyEnabled = !EditingSettings.PhilosophyEnabled; b.SetState()
		}),
	}
	if EditingSettings.PhilosophyEnabled {
		for _, c := range Philosophies {
			cc := c
			on := Contains(EditingSettings.PhilosophySelected, cc.ID)
			kids = append(kids, widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(16, 8, 0, 0)},
				ui.Expand(ui.TextC("· "+cc.Name, *ui.Fg, 12)),
				ui.AccentPill("已选", "未选", on, func() {
					EditingSettings.PhilosophySelected = Toggle(EditingSettings.PhilosophySelected, cc.ID); b.SetState()
				}),
			))
		}
	}
	kids = append(kids, widget.Div(widget.Style{Height: 14}), ui.TextC("子 Agent 哲学（每个角色独立配置）", *ui.FgMuted, 11), widget.Div(widget.Style{Height: 4}))
	for _, r := range RoleEntries {
		kids = append(kids, b.roleCard(r.ID, r.Name))
	}
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"}, kids)
}

func (b *bodyState) roleCard(id, name string) widget.Widget {
	expanded := b.editingRole == id
	hbg := *ui.BgSubtle; hint := "编辑"
	if expanded { hbg = *ui.BgMuted; hint = "收起" }
	header := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(widget.Style{Height: 30, FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(8, 0, 8, 0), BackgroundColor: &hbg},
			ui.Expand(ui.TextC(name, *ui.Fg, 12)), ui.TextC(hint, *ui.FgMuted, 11))},
		OnClick: func() {
			if b.editingRole == id { b.editingRole = "" } else { b.editingRole = id }; b.SetState()
		},
		HoverColor: *ui.BgMuted,
	}
	if !expanded { return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", Padding: types.EdgeInsetsLTRB(0, 4, 0, 0)}, header) }
	ta := settingsTextarea("该角色专属哲学内容（留空=用内置默认）…", EditingSettings.PhilosophyRoles[id], 10, b.resetTok, func(t string) {
		if EditingSettings.PhilosophyRoles == nil { EditingSettings.PhilosophyRoles = map[string]string{} }
		EditingSettings.PhilosophyRoles[id] = t
	})
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", Padding: types.EdgeInsetsLTRB(0, 4, 0, 0)},
		header, widget.Div(widget.Style{Height: 6}), ta, widget.Div(widget.Style{Height: 6}),
		ui.Row(ui.PrimaryBtn("保存", func() { widget.MessageSuccess("已保存") }), ui.HGap(8),
			ui.Btn("恢复默认", func() { delete(EditingSettings.PhilosophyRoles, id); b.resetTok++; b.SetState() })))
}

func (b *bodyState) mcpTab() widget.Widget {
	kids := []widget.Widget{
		settingsToggle("启动对话时自动连接 MCP", EditingSettings.AutoConnectMCP, func() { EditingSettings.AutoConnectMCP = !EditingSettings.AutoConnectMCP; b.SetState() }),
		widget.Div(widget.Style{Height: 8}),
	}
	for _, lv := range mcppanel.Levels {
		kids = append(kids, b.mcpLevelSection(lv.ID, lv.Name))
	}
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"}, kids)
}

func (b *bodyState) sectionHead(key, title string, n int, action widget.Widget) widget.Widget {
	expanded := !b.secCollapsed[key]
	chev := "chevron-right"; if expanded { chev = "chevron-down" }
	return widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"},
		ui.Expand(&widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center", Height: 26},
				widget.Lucide(chev, widget.IconSize(14), widget.IconColor(*ui.FgMuted)), widget.Div(widget.Style{Width: 6}),
				ui.TextC(title+"（"+ui.Itoa(n)+"）", *ui.Fg, 12))},
			OnClick: func() { b.toggleSec(key); b.SetState() }, HoverColor: *ui.BgMuted,
		}), action)
}

func (b *bodyState) mcpLevelSection(level, name string) widget.Widget {
	servers := mcppanel.ReadLevel(level)
	editable := level != mcppanel.LevelSystem
	key := "mcp:" + level
	var add widget.Widget
	if editable { lv := level; add = ui.PrimaryBtnX("+ 添加", func() { mcppanel.OpenEditor(lv, mcppanel.Entry{}, b.SetState) }, ui.BtnOpts{Size: ui.SizeSm}) }
	out := []widget.Widget{widget.Div(widget.Style{Height: 8}), b.sectionHead(key, name, len(servers), add)}
	if !b.secCollapsed[key] {
		if len(servers) == 0 { out = append(out, widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(20, 4, 0, 0)}, ui.TextC("（无）", *ui.FgMuted, 10))) }
		for _, s := range servers { out = append(out, widget.Div(widget.Style{Height: 4}), b.mcpCard(level, s, editable)) }
	}
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"}, out)
}

func (b *bodyState) mcpCard(level string, s mcppanel.Entry, editable bool) widget.Widget {
	cmd := s.Command
	for _, a := range s.Args { cmd += " " + a }
	lv := level
	on := mcppanel.Enabled(level, s.Name)
	row := []widget.Widget{enablePill(on, func() { mcppanel.SetEnabled(lv, s.Name, !on); b.SetState() }), widget.Div(widget.Style{Width: 8}), ui.Expand(ui.TextC(s.Name, *ui.Fg, 12))}
	if editable {
		row = append(row, ui.BtnX("编辑", func() { mcppanel.OpenEditor(lv, s, b.SetState) }, ui.BtnOpts{Size: ui.SizeSm}), ui.HGap(6),
			ui.DangerBtnX("删除", func() { _ = mcppanel.Delete(lv, s.Name); b.SetState() }, ui.BtnOpts{Size: ui.SizeSm}))
	}
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", BackgroundColor: ui.Bg,
		BorderColor: ui.Border, BorderWidth: 1, BorderRadius: 5, Padding: types.EdgeInsets(8), Margin: types.EdgeInsetsLTRB(16, 0, 0, 0)},
		widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"}, row), widget.Div(widget.Style{Height: 4}), ui.Mono(cmd, *ui.FgMuted, 10))
}

func enablePill(on bool, toggle func()) widget.Widget { return ui.Pill("启用", "禁用", on, toggle) }
func orStr(s, def string) string { if strings.TrimSpace(s) == "" { return def }; return s }

func (b *bodyState) skillsTab() widget.Widget {
	kids := []widget.Widget{ui.TextC("SKILL.md 放入对应层级 skills/<名>/ 即可加载", *ui.FgMuted, 10)}
	for _, lv := range skillspanel.Levels {
		kids = append(kids, b.skillLevelSection(lv.ID, lv.Name))
	}
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"}, kids)
}

func (b *bodyState) skillLevelSection(level, name string) widget.Widget {
	skills := skillspanel.ReadLevel(level)
	editable := level != mcppanel.LevelSystem
	key := "skill:" + level
	var add widget.Widget
	if editable { lv := level; add = ui.PrimaryBtnX("+ 添加", func() { skillspanel.OpenEditor(lv, skillspanel.Entry{}, b.SetState) }, ui.BtnOpts{Size: ui.SizeSm}) }
	out := []widget.Widget{widget.Div(widget.Style{Height: 8}), b.sectionHead(key, name, len(skills), add)}
	if !b.secCollapsed[key] {
		if len(skills) == 0 { out = append(out, widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(20, 4, 0, 0)}, ui.TextC("（无）", *ui.FgMuted, 10))) }
		for _, s := range skills { out = append(out, widget.Div(widget.Style{Height: 4}), b.skillCard(level, s, editable)) }
	}
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"}, out)
}

func (b *bodyState) skillCard(level string, s skillspanel.Entry, editable bool) widget.Widget {
	lv, ss := level, s
	on := skillspanel.Enabled(level, s.Name)
	row := []widget.Widget{enablePill(on, func() { skillspanel.SetEnabled(lv, ss.Name, !on); b.SetState() }), widget.Div(widget.Style{Width: 8}), ui.Expand(ui.TextC(ss.Name, *ui.Fg, 12)), ui.TextC(skillspanel.ModeLabel(ss.Mode), *ui.FgMuted, 10)}
	if editable {
		row = append(row, ui.HGap(8), ui.BtnX("编辑", func() { skillspanel.OpenEditor(lv, ss, b.SetState) }, ui.BtnOpts{Size: ui.SizeSm}), ui.HGap(6),
			ui.DangerBtnX("删除", func() { _ = skillspanel.Delete(lv, ss.Name); b.SetState() }, ui.BtnOpts{Size: ui.SizeSm}))
	}
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", BackgroundColor: ui.Bg,
		BorderColor: ui.Border, BorderWidth: 1, BorderRadius: 5, Padding: types.EdgeInsets(8), Margin: types.EdgeInsetsLTRB(16, 0, 0, 0)},
		widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"}, row), widget.Div(widget.Style{Height: 4}),
		ui.TextC(orStr(ss.Description, "（无描述）"), *ui.FgMuted, 10))
}

func settingsToggle(lbl string, on bool, toggle func()) widget.Widget { return ui.Toggle(lbl, on, toggle) }

const settingsCtlW = 522

func providerSelect(value string, onPick func(string)) widget.Widget {
	opts := make([]widget.SelectOption, 0, len(providerPresets))
	for _, p := range providerPresets { opts = append(opts, widget.SelectOption{Label: p.label, Value: p.name}) }
	return widget.NewSelect(opts).WithValue(value).WithWidth(settingsCtlW).WithOnChanged(onPick)
}

func (b *bodyState) modelTab() widget.Widget {
	prov := EditingSettings.Provider
	temp := core.ParseTempOr(EditingSettings.Temperature, 1.0)
	maxTokStr := ""
	if EditingSettings.MaxTokens > 0 { maxTokStr = ui.Itoa(EditingSettings.MaxTokens) }
	provSel := providerSelect(prov, func(v string) {
		EditingSettings.Provider = v
		if v != "custom" { _, _, base, _ := providerByID(v); dm := defaultModelFor(v); EditingSettings.BaseURL = base; EditingSettings.PlanModel, EditingSettings.ExecuteModel, EditingSettings.ReviewModel = dm, dm, dm }
		b.resetTok++; b.SetState()
	})
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		ui.TextC("模型配置", *ui.FgMuted, 11),
		settingsField("服务提供商", provSel),
		settingsField("API Key", settingsInput("sk-...", EditingSettings.APIKey, b.resetTok, func(t string) { EditingSettings.APIKey = t })),
		settingsField("API 地址", settingsInput("https://...", EditingSettings.BaseURL, b.resetTok, func(t string) { EditingSettings.BaseURL = t })),
		settingsField("规划模型", modelSelectFor(prov, EditingSettings.PlanModel, b.resetTok, func(v string) { EditingSettings.PlanModel = v; b.SetState() })),
		settingsField("执行模型", modelSelectFor(prov, EditingSettings.ExecuteModel, b.resetTok, func(v string) { EditingSettings.ExecuteModel = v; b.SetState() })),
		settingsField("审核模型", modelSelectFor(prov, EditingSettings.ReviewModel, b.resetTok, func(v string) { EditingSettings.ReviewModel = v; b.SetState() })),
		settingsSlider("温度: "+strconv.FormatFloat(temp, 'f', 1, 64), temp, 0, 2, 0.1, func(v float64) { EditingSettings.Temperature = strconv.FormatFloat(v, 'f', 1, 64); b.SetState() }),
		settingsField("思考模式", thinkingSelect(EditingSettings.ThinkingMode, func(v string) { EditingSettings.ThinkingMode = v; b.SetState() })),
		settingsField("最大 Token", settingsInput("131072", maxTokStr, b.resetTok, func(t string) { EditingSettings.MaxTokens, _ = strconv.Atoi(strings.TrimSpace(t)) })),
		settingsSlider("上下文窗口: "+fmtContext(EditingSettings.ContextMaxTokens), float64(EditingSettings.ContextMaxTokens), 32000, 1000000, 32000, func(v float64) { EditingSettings.ContextMaxTokens = int(v); b.SetState() }),
		widget.Div(widget.Style{Height: 16}),
		settingsToggle("启用压缩模型", EditingSettings.CompressEnabled, func() { EditingSettings.CompressEnabled = !EditingSettings.CompressEnabled; b.SetState() }),
		b.compressSection(),
	)
}

func (b *bodyState) compressSection() widget.Widget {
	if !EditingSettings.CompressEnabled { return widget.Div(widget.Style{}) }
	cp := EditingSettings.CompressProvider
	if cp == "" { cp = "deepseek" }
	provSel := providerSelect(cp, func(v string) {
		EditingSettings.CompressProvider = v
		if v != "custom" { _, _, base, _ := providerByID(v); EditingSettings.CompressBaseURL = base; EditingSettings.CompressModel = defaultModelFor(v) }
		b.resetTok++; b.SetState()
	})
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		settingsField("服务提供商", provSel),
		settingsField("API Key", settingsInput("留空则复用主模型 Key", EditingSettings.CompressAPIKey, b.resetTok, func(t string) { EditingSettings.CompressAPIKey = t })),
		settingsField("API 地址", settingsInput("https://...", EditingSettings.CompressBaseURL, b.resetTok, func(t string) { EditingSettings.CompressBaseURL = t })),
		settingsField("模型", modelSelectFor(cp, EditingSettings.CompressModel, b.resetTok, func(v string) { EditingSettings.CompressModel = v; b.SetState() })),
		settingsField("思考模式", thinkingSelect(EditingSettings.CompressThinkingMode, func(v string) { EditingSettings.CompressThinkingMode = v; b.SetState() })),
	)
}

func settingsField(lbl string, in widget.Widget) widget.Widget { return ui.Field(lbl, in) }
func settingsInput(placeholder, val string, tok int, onChanged func(string)) widget.Widget { return ui.Input(placeholder, val, tok, onChanged) }
func settingsSlider(lbl string, val, min, max, step float64, onChange func(float64)) widget.Widget { return ui.SliderField(lbl, val, min, max, step, onChange) }

func modelSelectFor(provider, value string, tok int, onChange func(string)) widget.Widget {
	if provider == "custom" { return settingsInput("自定义模型 ID", value, tok, onChange) }
	_, _, _, models := providerByID(provider)
	opts := make([]widget.SelectOption, 0, len(models)+1)
	inList := false
	for _, m := range models { opts = append(opts, widget.SelectOption{Label: m.name, Value: m.id}); if m.id == value { inList = true } }
	if value != "" && !inList { opts = append([]widget.SelectOption{{Label: value, Value: value}}, opts...) }
	return widget.NewSelect(opts).WithValue(value).WithWidth(settingsCtlW).WithOnChanged(onChange)
}

func thinkingSelect(value string, onChange func(string)) widget.Widget {
	return settingsSelect(value, []widget.SelectOption{{Label: "关闭", Value: "non-thinking"}, {Label: "开启（推荐）", Value: "thinking"}, {Label: "最大推理深度", Value: "thinking_max"}}, onChange)
}

func settingsSelect(value string, opts []widget.SelectOption, onChange func(string)) widget.Widget {
	return ui.Select(value, opts, settingsCtlW, onChange)
}

func fmtContext(n int) string {
	if n >= 1000000 { return ui.Itoa(n/1000000) + "M" }
	if n >= 1000 { return ui.Itoa(n/1000) + "K" }
	return ui.Itoa(n)
}

// ─── 字体控件 ──

func (b *bodyState) FontControl(famPtr *string, defLabel string, monoOnly bool, sizePtr *int, boldPtr, italicPtr, underlinePtr *bool) widget.Widget {
	cur := core.FirstFontFamily(*famPtr)
	var fonts []string
	if monoOnly {
		if cur == "" { cur = "Consolas" }
		fonts = sysfont.Available(cur)
	} else { fonts = sysfont.System() }
	o := ui.FontPickerOpts{
		Fonts: fonts, Current: cur, DefaultLabel: defLabel, Width: 196, ResetTok: b.resetTok,
		Bold: *boldPtr, Italic: *italicPtr, Underline: *underlinePtr,
		OnFont: func(v string) { *famPtr = v; b.SetState() },
		OnBold: func() { *boldPtr = !*boldPtr; b.SetState() }, OnItalic: func() { *italicPtr = !*italicPtr; b.SetState() }, OnUnderline: func() { *underlinePtr = !*underlinePtr; b.SetState() },
	}
	if sizePtr != nil { o.Size = *sizePtr; o.OnSize = func(v int) { *sizePtr = v; b.SetState() } }
	return ui.FontPicker(o)
}

// ─── 字段绑定组件注册 ──

func RegisterSettingsUI() {
	InitFieldPtrs()
	RegisterBoundComponents()
	config.Register("EditorFontPicker", func(ctx widget.DeclarativeContext) widget.Widget {
		return theBody.FontControl(&EditingSettings.FontFamily, "", true,
			&EditingSettings.EditorFontSize, &EditingSettings.EditorFontBold,
			&EditingSettings.EditorFontItalic, &EditingSettings.EditorFontUnderline)
	})
	config.Register("UIFontPicker", func(ctx widget.DeclarativeContext) widget.Widget {
		return theBody.FontControl(&EditingSettings.UIFontFamily, "默认", false, nil,
			&EditingSettings.UIFontBold, &EditingSettings.UIFontItalic, &EditingSettings.UIFontUnderline)
	})
}

func RegisterBoundComponents() {
	config.Register("SettingsSlider", func(ctx widget.DeclarativeContext) widget.Widget {
		ptr := IntFields[sProp(ctx, "field")]; if ptr == nil { return ui.Muted("未知字段") }
		return ui.SliderField(sProp(ctx, "label")+": "+strconv.Itoa(*ptr)+sProp(ctx, "suffix"), float64(*ptr), fProp(ctx, "min"), fProp(ctx, "max"), fPropOr(ctx, "step", 1), func(v float64) { *ptr = int(v); theBody.SetState() })
	})
	config.Register("SettingsToggle", func(ctx widget.DeclarativeContext) widget.Widget {
		ptr := BoolFields[sProp(ctx, "field")]; if ptr == nil { return ui.Muted("⟨未知字段⟩") }
		on := *ptr; if bProp(ctx, "invert") { on = !on }
		return ui.Toggle(sProp(ctx, "label"), on, func() { *ptr = !*ptr; theBody.SetState() })
	})
	config.Register("SettingsSelect", func(ctx widget.DeclarativeContext) widget.Widget {
		ptr := StrFields[sProp(ctx, "field")]; if ptr == nil { return ui.Muted("⟨未知字段⟩") }
		w := fPropOr(ctx, "width", settingsCtlW)
		sel := ui.Select(*ptr, toSelectOptions(ctx.Spec.Props["options"]), w, func(v string) { *ptr = v; theBody.SetState() })
		if lbl := sProp(ctx, "label"); lbl != "" { return ui.Field(lbl, sel) }
		return sel
	})
	config.Register("SettingsInput", func(ctx widget.DeclarativeContext) widget.Widget {
		field, ph, lbl := sProp(ctx, "field"), sProp(ctx, "placeholder"), sProp(ctx, "label")
		if bProp(ctx, "list") {
			ptr := ListFields[field]; if ptr == nil { return ui.Muted("⟨未知列表字段⟩") }
			in := ui.Input(ph, strings.Join(*ptr, ", "), theBody.resetTok, func(t string) { *ptr = splitCommaList(t) })
			return ui.Field(lbl, in)
		}
		ptr := StrFields[field]; if ptr == nil { return ui.Muted("⟨未知字段⟩") }
		in := ui.Input(ph, *ptr, theBody.resetTok, func(t string) { *ptr = t })
		return ui.Field(lbl, in)
	})
}

func splitCommaList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") { if p = strings.TrimSpace(p); p != "" { out = append(out, p) } }
	return out
}

func sProp(ctx widget.DeclarativeContext, k string) string { s, _ := ctx.Spec.Props[k].(string); return s }
func bProp(ctx widget.DeclarativeContext, k string) bool { b, _ := ctx.Spec.Props[k].(bool); return b }
func fProp(ctx widget.DeclarativeContext, k string) float64 { return fPropOr(ctx, k, 0) }
func fPropOr(ctx widget.DeclarativeContext, k string, def float64) float64 {
	switch v := ctx.Spec.Props[k].(type) { case float64: return v; case int: return float64(v) }
	return def
}

func toSelectOptions(v any) []widget.SelectOption {
	arr, ok := v.([]any); if !ok { return nil }
	out := make([]widget.SelectOption, 0, len(arr))
	for _, it := range arr { if m, ok := it.(map[string]any); ok { l, _ := m["label"].(string); val, _ := m["value"].(string); out = append(out, widget.SelectOption{Label: l, Value: val}) } }
	return out
}
