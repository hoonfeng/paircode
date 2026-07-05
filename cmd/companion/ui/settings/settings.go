//go:build windows

// Package settings 提供设置面板的加载/保存/UI（GWui 版）。
// 使用 uixml 声明式 UI 构建设置对话框布局，保留 Go 逻辑处理交互与数据。
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"
	"github.com/hoonfeng/gwui/uixml"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/paircode/cmd/companion/uiapi"
)

// ApplyIgnoreDirs 由 bridge 注入：应用忽略目录设置到搜索/知识库。
var ApplyIgnoreDirs func(root string)

// ─── 哲学数据（roleprompts 包引用）──

// PhilosophyEntry 一部经典哲学（思想 tab 可选）。
type PhilosophyEntry struct {
	ID   string
	Name string
}

// RoleEntry 一个角色（哲学小节标题用）。
type RoleEntry struct {
	ID   string
	Name string
}

// Philosophies 可选的经典哲学列表（ID 与 settings.PhilosophySelected 对应）。
var Philosophies = []PhilosophyEntry{
	{"tao-te-ching", "《道德经》"},
	{"huangdi-yinfu-jing", "《黄帝阴符经》"},
	{"sunzi-bingfa", "《孙子兵法》"},
	{"lunyu", "《论语》"},
	{"yijing", "《易经》"},
	{"zhongyong", "《中庸》"},
	{"daxue", "《大学》"},
}

// RoleEntries 角色列表（哲学小节标题用）。
var RoleEntries = []RoleEntry{
	{"planner", "规划"},
	{"reviewer", "审核"},
	{"judge", "评测"},
	{"explorer", "探索"},
	{"verifier", "验证"},
	{"debugger", "调试"},
	{"executor", "执行"},
}

// Contains 检查字符串是否在切片中（roleprompts 包用）。
func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// EditingSettings 是设置面板的编辑缓冲（OpenDialog 时从 core.Settings 复制，
// Save 时写回 core.Settings）。
var EditingSettings core.AppSettings

// Load 从 config/settings.json 加载设置到 core.Settings，并同步到编辑缓冲。
func Load() {
	core.Load()
	EditingSettings = core.Settings
}

// Save 把编辑缓冲写回 core.Settings 并保存到 config/settings.json。
func Save() {
	core.Settings = EditingSettings
	core.Save()
}

// ── 输入框工厂 ──

const inputBase = "background-color: " + ui.InputBg + "; color: " + ui.Text + "; border: 1px solid " + ui.Border + "; padding: 4px 8px; font-size: 13px; width: 100%; box-sizing: border-box; outline: none;"
const labelStyle = "color: " + ui.Text + "; font-size: 13px; width: 100px; flex-shrink: 0;"

func newInput(doc *dom.Document, ph string, val string) *component.Input {
	inp := component.NewInput(doc, ph)
	inp.SetValue(val)
	inp.SetBaseStyle(inputBase)
	return inp
}

func newNumberInput(doc *dom.Document, ph string, val int) *component.Input {
	inp := component.NewInput(doc, ph)
	if val > 0 {
		inp.SetValue(strconv.Itoa(val))
	}
	inp.SetBaseStyle(inputBase)
	return inp
}

func newSelect(doc *dom.Document, opts []string, val string) *component.Select {
	selIdx := 0
	for i, o := range opts {
		if o == val {
			selIdx = i
			break
		}
	}
	sel := component.NewSelect(doc, opts, selIdx)
	sel.SetBaseStyle(inputBase)
	return sel
}

func newCheckbox(doc *dom.Document, label string, checked bool) *component.Checkbox {
	cb := component.NewCheckbox(doc, label, checked)
	cb.SetBaseStyle("color: " + ui.Text + "; font-size: 13px; display: flex; align-items: center; gap: 8px; cursor: pointer;")
	return cb
}

// ── 占位替换 ──

func replaceInput(doc *dom.Document, id string, inp *component.Input) {
	el := inp.Element()
	ui.ReplaceChildByID(doc, id, el)
	ensureFlex1(el)
}

func replaceSelect(doc *dom.Document, id string, sel *component.Select) {
	el := sel.Element()
	ui.ReplaceChildByID(doc, id, el)
	ensureFlex1(el)
}

// ensureFlex1 确保元素的内联 style 包含 flex:1;min-width:0。
// flex:1 让元素在 settings-row 的 flex 布局中填满剩余空间。
func ensureFlex1(el *dom.Element) {
	cur := el.GetAttribute("style")
	if cur != "" && strings.Contains(cur, "flex:") {
		return
	}
	if cur != "" {
		el.SetAttribute("style", cur+";flex:1;min-width:0;")
	} else {
		el.SetAttribute("style", "flex:1;min-width:0;")
	}
}

func replaceCheckbox(doc *dom.Document, id string, cb *component.Checkbox) {
	// Checkbox 自带 label，本身就是完整行
	container := doc.GetElementByID(id)
	if container == nil {
		return
	}
	// 清空容器并放 checkbox 元素
	for {
		c := container.FirstChild()
		if c == nil {
			break
		}
		container.RemoveChild(c)
	}
	container.AppendChild(cb.Element())
	ensureFlex1(cb.Element())
}

// ── OpenDialog ──

// clickHandler 为 GWui 组件注册系统提供点击事件处理。
type clickHandler struct {
	fn func()
}

func (h *clickHandler) HandleEvent(e event.Event) bool {
	if e.Type() == event.Click {
		h.fn()
		return true
	}
	return false
}
func (h *clickHandler) Focusable() bool { return false }

// OpenDialog 打开设置对话框（Modal）。
func OpenDialog() {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	EditingSettings = core.Settings

	modal := component.NewModal(doc)
	modal.SetTitle("设置")
	modal.SetMaxWidth(600)
	modal.SetMaxHeight(560)

	// ── 在 header 右上角添加操作按钮 ──
	headerActions := modal.HeaderActions()

	saveBtn := doc.CreateElement("div")
	saveBtn.SetAttribute("style", "padding:2px 12px;font-size:12px;cursor:pointer;color:#fff;background:#0e639c;border:none;border-radius:3px;user-select:none;")
	saveBtn.SetTextContent("保存")
	headerActions.AppendChild(saveBtn)

	exportBtn := doc.CreateElement("div")
	exportBtn.SetAttribute("style", "padding:2px 10px;font-size:12px;cursor:pointer;color:#cccccc;border:1px solid #454545;border-radius:3px;user-select:none;")
	exportBtn.SetTextContent("导出")
	headerActions.AppendChild(exportBtn)

	importBtn := doc.CreateElement("div")
	importBtn.SetAttribute("style", "padding:2px 10px;font-size:12px;cursor:pointer;color:#cccccc;border:1px solid #454545;border-radius:3px;user-select:none;")
	importBtn.SetTextContent("导入")
	headerActions.AppendChild(importBtn)

	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()

	// ── 加载 HTML 模板 ──
	reg := uixml.NewRegistry()
	// 注册 tab onclick 处理器（使 HTML 中 onclick="selectSettingsTab(n)" 生效）
	for i := 0; i <= 6; i++ {
		idx := i
		reg.OnClick(fmt.Sprintf("selectSettingsTab(%d)", i), func(ctx uixml.EventContext) bool {
			selectTab(doc, idx)
			return true
		})
	}

	ui.MustLoadPanelHTML(doc, "panels/settings.html", reg)
	root := doc.GetElementByID("settings-root")

	// ── 创建所有控件并替换占位 ──
	createLLMTab(doc)
	createCompressTab(doc)
	createAgentTab(doc)
	createTerminalTab(doc)
	createAppearanceTab(doc)
	createPhilosophyTab(doc)
	createMCPTab(doc)

	// 转移组件 + 移入 Modal
	ui.TransferComponents(doc, doc, root)
	ui.DetachRoot(root)
	body.AppendChild(root)

	// ── 注册按钮点击事件 ──
	// 保存
	doc.RegisterComponent(saveBtn, &clickHandler{
		fn: func() {
			saveAll(doc)
			Save()
			ApplyAgentSettings()
			ApplyFontFamily()
			if ApplyIgnoreDirs != nil {
				ApplyIgnoreDirs(core.Root())
			}
			modal.Hide()
			uiapi.MessageSuccess("设置已保存")
		},
	})

	// 导出
	doc.RegisterComponent(exportBtn, &clickHandler{
		fn: func() {
			path := ui.SaveFileDialog("导出配置", "JSON 文件 (*.json)|*.json", "settings.json")
			if path == "" {
				return
			}
			data, err := json.MarshalIndent(EditingSettings, "", "  ")
			if err != nil {
				uiapi.MessageError("序列化配置失败: " + err.Error())
				return
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				uiapi.MessageError("写入文件失败: " + err.Error())
				return
			}
			uiapi.MessageSuccess("配置已导出: " + path)
		},
	})

	// 导入
	doc.RegisterComponent(importBtn, &clickHandler{
		fn: func() {
			path := ui.OpenFileDialog("导入配置", "JSON 文件 (*.json)|*.json")
			if path == "" {
				return
			}
			data, err := os.ReadFile(path)
			if err != nil {
				uiapi.MessageError("读取文件失败: " + err.Error())
				return
			}
			var imported core.AppSettings
			if err := json.Unmarshal(data, &imported); err != nil {
				uiapi.MessageError("解析配置文件失败: " + err.Error())
				return
			}
			// 应用导入配置
			core.Settings = imported
			core.Save()
			EditingSettings = core.Settings

			// 重建 UI
			body.ClearChildren()

			newReg := uixml.NewRegistry()
			ui.MustLoadPanelHTML(doc, "panels/settings.html", newReg)
			newRoot := doc.GetElementByID("settings-root")

			createLLMTab(doc)
			createCompressTab(doc)
			createAgentTab(doc)
			createTerminalTab(doc)
			createAppearanceTab(doc)
			createPhilosophyTab(doc)
			createMCPTab(doc)

			ui.TransferComponents(doc, doc, newRoot)
			ui.DetachRoot(newRoot)
			body.AppendChild(newRoot)

			selectTab(doc, 0)
			uiapi.MessageSuccess("配置已导入")
		},
	})

	modal.Show()
	selectTab(doc, 0) // 默认选中第一个 tab
}

// RegisterSettingsUI 注册设置 UI（空实现，预留扩展）。
func RegisterSettingsUI() {}

// ApplyAgentSettings 应用 Agent 行为设置到运行时。
func ApplyAgentSettings() {
	uiapi.MarkDirty()
}

// ApplyFontFamily 应用字体设置到运行时。
func ApplyFontFamily() {
	uiapi.MarkDirty()
}

// ── Tab 切换 ──

func selectTab(doc *dom.Document, idx int) {
	for i := 0; i <= 6; i++ {
		tabEl := doc.GetElementByID(fmt.Sprintf("settings-tab-%d", i))
		paneEl := doc.GetElementByID(fmt.Sprintf("settings-pane-%d", i))
		if tabEl == nil || paneEl == nil {
			continue
		}
		if i == idx {
			tabEl.SetAttribute("style", "padding:8px 16px;cursor:pointer;font-size:13px;color:"+ui.Text+";border-bottom:2px solid "+ui.Accent+";user-select:none;")
			paneEl.SetAttribute("style", "display:flex;flex-direction:column;gap:10px;")
		} else {
			tabEl.SetAttribute("style", "padding:8px 16px;cursor:pointer;font-size:13px;color:"+ui.TextDim+";border-bottom:2px solid transparent;user-select:none;")
			paneEl.SetAttribute("style", "display:none;")
		}
	}
}

// ── 各 Tab 控件创建 ──

// 服务商默认 API 地址映射（选择服务商时自动填充 API 地址 / 模型列表）。
var providerBaseURLs = map[string]string{
	"deepseek":          "https://api.deepseek.com/v1",
	"openai":            "https://api.openai.com/v1",
	"anthropic":         "https://api.anthropic.com",
	"openai-compatible": "",
	"custom":            "",
}

var (
	providerSel         *component.Select
	baseURLInput        *component.Input
	apiKeyInput         *component.Input
	execModelSel        *component.Select
	planModelSel        *component.Select
	reviewModelSel      *component.Select
	tempSlider          *component.Slider
	tempLabel           *dom.Element
	thinkingSelect      *component.Select
	maxTokensInput      *component.Input
	ctxMaxTokensInput   *component.Input
	compressCb          *component.Checkbox
	compressProviderSel *component.Select
	compressAPIKeyInp   *component.Input
	compressBaseURLInp  *component.Input
	compressModelSel    *component.Select
	compressThinkSel    *component.Select
	autoCollapseCb      *component.Checkbox
	autoIterateCb       *component.Checkbox
	requireApprovalCb   *component.Checkbox
	luaToolsCb          *component.Checkbox
	benchmarkCb         *component.Checkbox
	maxIterationsInp    *component.Input
	maxParallelInp      *component.Input
	reviewRetriesInp    *component.Input
	searxngInput        *component.Input
	ignoreDirsInput     *component.Input
	sysInstructionsInp  *component.Input
	shellSelect         *component.Select
	termFontSizeInp     *component.Input
	encodingSelect      *component.Select
	themeSelect         *component.Select
	fontFamilyInput     *component.Input
	editorFontSizeInp   *component.Input
	editorFontBoldCb    *component.Checkbox
	editorFontItalicCb  *component.Checkbox
	editorFontULCb      *component.Checkbox
	uiFontFamilyInp     *component.Input
	uiFontBoldCb        *component.Checkbox
	uiFontItalicCb      *component.Checkbox
	uiFontULCb          *component.Checkbox
	hideMinimapCb       *component.Checkbox
	philosophyCb        *component.Checkbox
	philosophyCbs       []*component.Checkbox // 每个经典一个 checkbox
	autoConnectMCPCb    *component.Checkbox
	skillCbs            []*component.Checkbox // 技能开关列表
)

func createLLMTab(doc *dom.Document) {
	s := &EditingSettings
	modelOpts := core.GetModels(s.Provider)
	providerSel = newSelect(doc, core.GetProviders(), s.Provider)
	providerSel.OnChange(func(idx int, val string) {
		// 切换服务商时：自动填充 baseURL + 刷新模型列表
		if url, ok := providerBaseURLs[val]; ok {
			baseURLInput.SetValue(url)
		}
		models := core.GetModels(val)
		execModelSel.SetOptions(models)
		planModelSel.SetOptions(models)
		reviewModelSel.SetOptions(models)
	})
	replaceSelect(doc, "s-provider", providerSel)

	baseURLInput = newInput(doc, "API 地址", s.BaseURL)
	replaceInput(doc, "s-baseurl", baseURLInput)

	apiKeyInput = newInput(doc, "API Key", s.APIKey)
	replaceInput(doc, "s-apikey", apiKeyInput)

	execModelSel = newSelect(doc, modelOpts, s.ExecuteModel)
	replaceSelect(doc, "s-execmodel", execModelSel)

	planModelSel = newSelect(doc, modelOpts, s.PlanModel)
	replaceSelect(doc, "s-planmodel", planModelSel)

	reviewModelSel = newSelect(doc, modelOpts, s.ReviewModel)
	replaceSelect(doc, "s-reviewmodel", reviewModelSel)

	tempVal := parseFloat(s.Temperature, 0.7)
	tempSlider = component.NewSlider(doc, 0, 2, tempVal)
	tempSlider.SetStep(0.1)
	tempSlider.SetBaseStyle("flex:1;min-width:0;padding:0 4px;")
	tempLabel = doc.CreateElement("span")
	tempLabel.SetAttribute("style", "color: "+ui.Text+"; font-size: 13px; min-width: 32px; text-align: right;")
	tempLabel.SetTextContent(fmt.Sprintf("%.1f", tempVal))
	tempSlider.OnChange(func(v float32) {
		tempLabel.SetTextContent(fmt.Sprintf("%.1f", v))
	})
	container := doc.GetElementByID("s-temperature")
	if container != nil {
		container.ClearChildren()
		container.AppendChild(tempSlider.Element())
		container.AppendChild(tempLabel)
	}

	thinkingSelect = newSelect(doc, []string{"non-thinking", "thinking", "thinking_max"}, s.ThinkingMode)
	replaceSelect(doc, "s-thinkingmode", thinkingSelect)

	maxTokensInput = newNumberInput(doc, "最大 Token 数", s.MaxTokens)
	replaceInput(doc, "s-maxtokens", maxTokensInput)

	ctxMaxTokensInput = newNumberInput(doc, "上下文窗口上限", s.ContextMaxTokens)
	replaceInput(doc, "s-ctxmaxtokens", ctxMaxTokensInput)
}

func createCompressTab(doc *dom.Document) {
	s := &EditingSettings
	compressCb = newCheckbox(doc, "启用上下文压缩", s.CompressEnabled)
	replaceCheckbox(doc, "s-compress-enabled", compressCb)

	compressProviderSel = newSelect(doc, core.GetProviders(), s.CompressProvider)
	compressProviderSel.OnChange(func(idx int, val string) {
		// 切换压缩服务商时：自动填充 baseURL + 刷新模型列表
		if url, ok := providerBaseURLs[val]; ok {
			compressBaseURLInp.SetValue(url)
		}
		compressModelSel.SetOptions(core.GetModels(val))
	})
	replaceSelect(doc, "s-compress-provider", compressProviderSel)

	compressAPIKeyInp = newInput(doc, "API Key", s.CompressAPIKey)
	replaceInput(doc, "s-compress-apikey", compressAPIKeyInp)

	compressBaseURLInp = newInput(doc, "API 地址", s.CompressBaseURL)
	replaceInput(doc, "s-compress-baseurl", compressBaseURLInp)

	compressModelSel = newSelect(doc, core.GetModels(s.CompressProvider), s.CompressModel)
	replaceSelect(doc, "s-compress-model", compressModelSel)

	compressThinkSel = newSelect(doc, []string{"non-thinking", "thinking", "thinking_max"}, s.CompressThinkingMode)
	replaceSelect(doc, "s-compress-thinking", compressThinkSel)
}

func createAgentTab(doc *dom.Document) {
	s := &EditingSettings

	autoCollapseCb = newCheckbox(doc, "自动折叠工具调用输出", s.AutoCollapse)
	replaceCheckbox(doc, "s-autocollapse", autoCollapseCb)

	autoIterateCb = newCheckbox(doc, "驳回后自动迭代改进", s.AutoIterate)
	replaceCheckbox(doc, "s-autoiterate", autoIterateCb)

	requireApprovalCb = newCheckbox(doc, "破坏性操作需人类审批", s.RequireApproval)
	replaceCheckbox(doc, "s-requireapproval", requireApprovalCb)

	luaToolsCb = newCheckbox(doc, "启用 Lua 自定义工具", s.LuaTools)
	replaceCheckbox(doc, "s-luatools", luaToolsCb)

	benchmarkCb = newCheckbox(doc, "启用基准测试（Benchmark）", s.Benchmark)
	replaceCheckbox(doc, "s-benchmark", benchmarkCb)

	maxIterationsInp = newNumberInput(doc, "最大迭代步数", s.MaxIterations)
	replaceInput(doc, "s-maxiterations", maxIterationsInp)

	maxParallelInp = newNumberInput(doc, "最大并行 Agent 数", s.MaxParallel)
	replaceInput(doc, "s-maxparallel", maxParallelInp)

	reviewRetriesInp = newNumberInput(doc, "审核重试次数", s.ReviewRetries)
	replaceInput(doc, "s-reviewretries", reviewRetriesInp)

	searxngInput = newInput(doc, "SearXNG 地址（空=用 DuckDuckGo）", s.SearxngURL)
	replaceInput(doc, "s-searxng", searxngInput)

	ignoreDirsInput = newInput(doc, "逗号分隔的目录名", strings.Join(s.IgnoreDirs, ", "))
	replaceInput(doc, "s-ignoredirs", ignoreDirsInput)

	sysInstructionsInp = newInput(doc, "系统级指令（多行用 \\n）", s.SystemInstructions)
	replaceInput(doc, "s-systeminstructions", sysInstructionsInp)
}

func createTerminalTab(doc *dom.Document) {
	s := &EditingSettings
	shellSelect = newSelect(doc, []string{"auto", "cmd", "powershell", "git-bash"}, s.DefaultShell)
	replaceSelect(doc, "s-defaultshell", shellSelect)

	termFontSizeInp = newNumberInput(doc, "终端字号", s.TermFontSize)
	replaceInput(doc, "s-termfontsize", termFontSizeInp)

	encodingSelect = newSelect(doc, []string{"auto", "utf-8", "gbk"}, s.TermEncoding)
	replaceSelect(doc, "s-termencoding", encodingSelect)
}

func createAppearanceTab(doc *dom.Document) {
	s := &EditingSettings
	themeSelect = newSelect(doc, []string{"dark", "light", "high-contrast", "solarized-light", "dracula"}, s.Theme)
	replaceSelect(doc, "s-theme", themeSelect)

	fontFamilyInput = newInput(doc, "编辑器字体族", s.FontFamily)
	replaceInput(doc, "s-fontfamily", fontFamilyInput)

	editorFontSizeInp = newNumberInput(doc, "编辑器字号", s.EditorFontSize)
	replaceInput(doc, "s-editorfontsize", editorFontSizeInp)

	editorFontBoldCb = newCheckbox(doc, "编辑器字体加粗", s.EditorFontBold)
	replaceCheckbox(doc, "s-editorfontbold", editorFontBoldCb)

	editorFontItalicCb = newCheckbox(doc, "编辑器字体斜体", s.EditorFontItalic)
	replaceCheckbox(doc, "s-editorfontitalic", editorFontItalicCb)

	editorFontULCb = newCheckbox(doc, "编辑器字体下划线", s.EditorFontUnderline)
	replaceCheckbox(doc, "s-editorfontunderline", editorFontULCb)

	uiFontFamilyInp = newInput(doc, "界面字体族", s.UIFontFamily)
	replaceInput(doc, "s-uifontfamily", uiFontFamilyInp)

	uiFontBoldCb = newCheckbox(doc, "界面字体加粗", s.UIFontBold)
	replaceCheckbox(doc, "s-uifontbold", uiFontBoldCb)

	uiFontItalicCb = newCheckbox(doc, "界面字体斜体", s.UIFontItalic)
	replaceCheckbox(doc, "s-uifontitalic", uiFontItalicCb)

	uiFontULCb = newCheckbox(doc, "界面字体下划线", s.UIFontUnderline)
	replaceCheckbox(doc, "s-uifontunderline", uiFontULCb)

	hideMinimapCb = newCheckbox(doc, "隐藏编辑器 Minimap", s.HideMinimap)
	replaceCheckbox(doc, "s-hideminimap", hideMinimapCb)
}

func createPhilosophyTab(doc *dom.Document) {
	s := &EditingSettings
	philosophyCb = newCheckbox(doc, "启用思想注入（Philosophy）", s.PhilosophyEnabled)
	replaceCheckbox(doc, "s-philosophy-enabled", philosophyCb)

	// 哲学经典列表
	philosophyCbs = nil
	listContainer := doc.GetElementByID("s-philosophy-list")
	if listContainer != nil {
		for _, p := range Philosophies {
			selected := false
			for _, sel := range s.PhilosophySelected {
				if sel == p.ID {
					selected = true
					break
				}
			}
			cb := newCheckbox(doc, p.Name, selected)
			philosophyCbs = append(philosophyCbs, cb)
			row := doc.CreateElement("div")
			row.SetAttribute("style", "padding: 2px 0;")
			row.AppendChild(cb.Element())
			listContainer.AppendChild(row)
		}
	}
}

func createMCPTab(doc *dom.Document) {
	s := &EditingSettings
	autoConnectMCPCb = newCheckbox(doc, "启动时自动连接 MCP 服务器", s.AutoConnectMCP)
	replaceCheckbox(doc, "s-autoconnectmcp", autoConnectMCPCb)

	// ── 技能管理列表 ──
	skillCbs = nil
	listContainer := doc.GetElementByID("s-skills-list")
	if listContainer == nil {
		return
	}
	listContainer.ClearChildren()

	skills := agent.LoadAllSkills()
	for _, sk := range skills {
		sk := sk // capture
		// 检查当前启用态
		key := string(sk.Level) + "::" + sk.Name
		enabled := true
		if v, ok := s.SkillEnabledOverrides[key]; ok {
			enabled = v
		}
		// 显示标签：层级前缀 + 名 + 描述
		levelTag := "system"
		if sk.Level == agent.LevelProject {
			levelTag = "project"
		}
		label := "[" + levelTag + "] " + sk.Name
		if sk.Description != "" {
			label += " — " + sk.Description
		}
		cb := newCheckbox(doc, label, enabled)
		skillCbs = append(skillCbs, cb)

		row := doc.CreateElement("div")
		row.SetAttribute("style", "padding: 2px 0;")
		row.AppendChild(cb.Element())
		listContainer.AppendChild(row)
	}
}

// ── 保存 ──

func saveAll(doc *dom.Document) {
	s := &EditingSettings

	// LLM
	s.Provider = providerSel.Value()
	s.BaseURL = baseURLInput.Value()
	s.APIKey = apiKeyInput.Value()
	s.ExecuteModel = execModelSel.Value()
	s.PlanModel = planModelSel.Value()
	s.ReviewModel = reviewModelSel.Value()
	s.Temperature = fmt.Sprintf("%.1f", tempSlider.Value())
	s.ThinkingMode = thinkingSelect.Value()
	s.MaxTokens = parseInt(maxTokensInput.Value())
	s.ContextMaxTokens = parseInt(ctxMaxTokensInput.Value())

	// 压缩
	s.CompressEnabled = compressCb.Checked()
	s.CompressProvider = compressProviderSel.Value()
	s.CompressAPIKey = compressAPIKeyInp.Value()
	s.CompressBaseURL = compressBaseURLInp.Value()
	s.CompressModel = compressModelSel.Value()
	s.CompressThinkingMode = compressThinkSel.Value()

	// Agent
	s.AutoCollapse = autoCollapseCb.Checked()
	s.AutoIterate = autoIterateCb.Checked()
	s.RequireApproval = requireApprovalCb.Checked()
	s.LuaTools = luaToolsCb.Checked()
	s.Benchmark = benchmarkCb.Checked()
	s.MaxIterations = parseInt(maxIterationsInp.Value())
	s.MaxParallel = parseInt(maxParallelInp.Value())
	s.ReviewRetries = parseInt(reviewRetriesInp.Value())
	s.SearxngURL = searxngInput.Value()
	s.IgnoreDirs = splitComma(ignoreDirsInput.Value())
	s.SystemInstructions = sysInstructionsInp.Value()

	// 终端
	s.DefaultShell = shellSelect.Value()
	s.TermFontSize = parseInt(termFontSizeInp.Value())
	s.TermEncoding = encodingSelect.Value()

	// 外观
	s.Theme = themeSelect.Value()
	s.FontFamily = fontFamilyInput.Value()
	s.EditorFontSize = parseInt(editorFontSizeInp.Value())
	s.EditorFontBold = editorFontBoldCb.Checked()
	s.EditorFontItalic = editorFontItalicCb.Checked()
	s.EditorFontUnderline = editorFontULCb.Checked()
	s.UIFontFamily = uiFontFamilyInp.Value()
	s.UIFontBold = uiFontBoldCb.Checked()
	s.UIFontItalic = uiFontItalicCb.Checked()
	s.UIFontUnderline = uiFontULCb.Checked()
	s.HideMinimap = hideMinimapCb.Checked()

	// 思想
	s.PhilosophyEnabled = philosophyCb.Checked()
	s.PhilosophySelected = nil
	for i, cb := range philosophyCbs {
		if cb.Checked() && i < len(Philosophies) {
			s.PhilosophySelected = append(s.PhilosophySelected, Philosophies[i].ID)
		}
	}

	// MCP
	s.AutoConnectMCP = autoConnectMCPCb.Checked()

	// 技能管理
	skills := agent.LoadAllSkills()
	overrides := make(map[string]bool)
	for i, cb := range skillCbs {
		if i >= len(skills) {
			break
		}
		key := string(skills[i].Level) + "::" + skills[i].Name
		// 只存与实际默认态不同的项：true=显式启用（当默认禁用时），false=显式禁用（当默认启用时）
		overrides[key] = cb.Checked()
	}
	s.SkillEnabledOverrides = overrides
}

func parseFloat(s string, fallback float32) float32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return fallback
	}
	return float32(v)
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func splitComma(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
