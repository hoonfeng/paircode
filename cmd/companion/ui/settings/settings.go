//go:build windows

// Package settings 提供设置面板的加载/保存/UI（GWui 版）。
// 使用 uixml 声明式 UI 构建设置对话框布局，保留 Go 逻辑处理交互与数据。
package settings

import (
	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/uixml"

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

// ── uixml 辅助 ──

// OpenDialog 打开设置对话框（Modal）。
func OpenDialog() {
	doc := ui.Ctx.Doc
	if doc == nil {
		return
	}
	// 从生效设置复制到编辑缓冲
	EditingSettings = core.Settings

	modal := component.NewModal(doc)
	modal.SetTitle("设置")

	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()

	// 在主文档上创建真实交互组件，保持完全可控的事件/值访问
	providerInput := component.NewInput(doc, "Provider（如 deepseek）")
	providerInput.SetValue(EditingSettings.Provider)
	providerInput.SetBaseStyle(
		"background-color: " + ui.InputBg + "; color: " + ui.Text + "; " +
			"border: 1px solid " + ui.Border + "; padding: 4px 8px; font-size: 13px; width: 100%;")

	apiKeyInput := component.NewInput(doc, "API Key")
	apiKeyInput.SetValue(EditingSettings.APIKey)
	apiKeyInput.SetBaseStyle(
		"background-color: " + ui.InputBg + "; color: " + ui.Text + "; " +
			"border: 1px solid " + ui.Border + "; padding: 4px 8px; font-size: 13px; width: 100%;")

	modelInput := component.NewInput(doc, "模型名（如 deepseek-v4-flash）")
	modelInput.SetValue(EditingSettings.ExecuteModel)
	modelInput.SetBaseStyle(
		"background-color: " + ui.InputBg + "; color: " + ui.Text + "; " +
			"border: 1px solid " + ui.Border + "; padding: 4px 8px; font-size: 13px; width: 100%;")

	autoReviewCb := component.NewCheckbox(doc, "自动审核（Auto Review）", EditingSettings.AutoReview)
	autonomousCb := component.NewCheckbox(doc, "自主模式（Autonomous）", EditingSettings.Autonomous)

	reg := uixml.NewRegistry()
	reg.OnClick("saveSettings", func(ctx uixml.EventContext) bool {
		// 从主文档组件直接读取最新值
		EditingSettings.Provider = providerInput.Value()
		EditingSettings.APIKey = apiKeyInput.Value()
		EditingSettings.ExecuteModel = modelInput.Value()
		EditingSettings.AutoReview = autoReviewCb.Checked()
		EditingSettings.Autonomous = autonomousCb.Checked()

		Save()
		ApplyAgentSettings()
		ApplyFontFamily()
		if ApplyIgnoreDirs != nil {
			ApplyIgnoreDirs(core.Root())
		}
		modal.Hide()
		uiapi.MessageSuccess("设置已保存")
		return true
	})
	reg.OnClick("cancelSettings", func(ctx uixml.EventContext) bool {
		modal.Hide()
		return true
	})

	// 加载 HTML 模板（资源目录 html/panels/settings.html）
	ui.MustLoadPanelHTML(doc, "panels/settings.html", reg)
	root := doc.GetElementByID("settings-root")

	// 用主文档上的真实组件替换占位元素
	ui.ReplaceChildByID(doc, "settings-provider-ph", providerInput.Element())
	ui.ReplaceChildByID(doc, "settings-apikey-ph", apiKeyInput.Element())
	ui.ReplaceChildByID(doc, "settings-model-ph", modelInput.Element())
	ui.ReplaceChildByID(doc, "settings-autoreview-ph", autoReviewCb.Element())
	ui.ReplaceChildByID(doc, "settings-autonomous-ph", autonomousCb.Element())

	// 转移组件注册 + 将布局内容移至 Modal body
	ui.TransferComponents(doc, doc, root)
	ui.DetachRoot(root)
	body.AppendChild(root)

	modal.Show()
}

// RegisterSettingsUI 注册设置 UI（空实现，预留扩展）。
func RegisterSettingsUI() {
	// 预留：可在此注册设置项的动态 UI
}

// ApplyAgentSettings 应用 Agent 行为设置到运行时。
func ApplyAgentSettings() {
	uiapi.MarkDirty()
}

// ApplyFontFamily 应用字体设置到运行时。
func ApplyFontFamily() {
	uiapi.MarkDirty()
}
