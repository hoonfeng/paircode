package agent

import (
	"testing"

	"github.com/hoonfeng/paircode/internal/core"
)

// TestContextWindowUsesAssembledValue 上下文窗口只消费装配结果。
// ★ 2026-09-20：核心零直读生成参数——settings 顶层 contextMaxTokens 已迁入服务商配置并清空
// （★ 2026-09-25 迁移目标 = 激活配置对应服务商的级字段，见 core/settings_generation.go）；
// 装配器未给值时回退机制兜底常量（运行保障，非配置面）。
func TestContextWindowUsesAssembledValue(t *testing.T) {
	if got := ContextWindow(ProviderParams{ContextMaxTokens: 1000000}); got != 1000000 {
		t.Fatalf("装配结果应优先：want 1000000, got %d", got)
	}
	if got := ContextWindow(ProviderParams{}); got != core.DefaultContextWindow {
		t.Fatalf("未装配窗口时应回退机制兜底常量：want %d, got %d", core.DefaultContextWindow, got)
	}
	if got := ContextWindow(ProviderParams{ContextMaxTokens: -1}); got != core.DefaultContextWindow {
		t.Fatalf("非法（<=0）装配结果应回退机制兜底常量：want %d, got %d", core.DefaultContextWindow, got)
	}
	// ★ 回归：核心字段不得参与取值（即使被外部写回）
	old := core.Settings.ContextMaxTokens
	defer func() { core.Settings.ContextMaxTokens = old }()
	core.Settings.ContextMaxTokens = 123456
	if got := ContextWindow(ProviderParams{}); got != core.DefaultContextWindow {
		t.Fatalf("核心 settings 字段不得参与窗口取值：want %d, got %d", core.DefaultContextWindow, got)
	}
}

// TestResolveProviderBaseHasNoConfigReads 裸基线不携带任何配置值（连接字段 + 生成参数）。
//
// ★ 2026-09-20 生成参数零直读：温度/思考档位/最大输出/上下文窗口/模型级参数表一律由
// 插件装配器决策（★ 2026-09-25 起唯一来源 = 服务商配置 models.json，模型级 > 服务商级；
// 原「全局默认」配置层随设置面板「生成参数」页移除，插件侧仅余机制常量 GEN_DEFAULTS 兜底）。
// ★ 2026-09-20 连接字段零直读：服务商/地址/Key/模型不再从 settings 顶层兜底——
// 旧顶层值已迁入 ai-presets.json 的一条 AI 配置并清空（core/settings_connection.go），
// 装配器按激活配置（core.Settings.Preset → ctx.aiPresets）经 ctx.models 展开。
func TestResolveProviderBaseHasNoConfigReads(t *testing.T) {
	oldSettings, oldList := core.Settings, core.ModelList
	defer func() { core.Settings, core.ModelList = oldSettings, oldList }()

	core.ModelList = core.ModelListMap{
		"prov-x": {
			BaseURL:          "https://x/v1",
			Models:           []string{"m1"},
			ContextMaxTokens: 1000000,
			Temperature:      "0.6",
			MaxTokens:        42000,
			ModelParams:      map[string]core.ModelParamEntry{"m1": {Temperature: "1.0"}},
		},
	}
	core.Settings = core.Default()
	core.Settings.Preset = "激活配置" // 装配上下文：必须透传给装配器
	// 即使旧连接字段与生成参数字段被写回（历史数据/外部写入），基线也不得携带它们
	core.Settings.Provider = "prov-x"
	core.Settings.BaseURL = "https://legacy.example/v1"
	core.Settings.APIKey = "LEGACY-KEY"
	core.Settings.ExecuteModel = "m-legacy"
	core.Settings.Model = "m-legacy-old"
	core.Settings.Temperature = "0.9"
	core.Settings.ThinkingMode = "max"
	core.Settings.MaxTokens = 99999
	core.Settings.ContextMaxTokens = 88888
	core.Settings.ModelParams = map[string]map[string]core.ModelParamEntry{
		"prov-x": {"m1": {Temperature: "0.1"}},
	}

	base := resolveProviderBase()
	if base.Preset != "激活配置" {
		t.Errorf("装配上下文（激活配置名）必须透传，得到 %q", base.Preset)
	}
	// ★ 连接字段零直读：基线这四项必须是「未配置」机制语义
	if base.Provider != "" {
		t.Errorf("基线服务商应为空（由装配器按激活配置展开），得到 %q", base.Provider)
	}
	if base.BaseURL != "" {
		t.Errorf("基线 API 地址应为空（唯一来源 AI 配置/models.json），得到 %q", base.BaseURL)
	}
	if base.APIKey != "" {
		t.Errorf("基线 API Key 应为空（唯一来源 AI 配置），得到 %q", base.APIKey)
	}
	if base.Model != "" {
		t.Errorf("基线模型应为空（唯一来源 AI 配置/会话选定），得到 %q", base.Model)
	}
	if base.ProviderContextMaxTokens != 0 {
		t.Errorf("基线不应预查服务商级上下文窗口（装配器按最终服务商经 ctx.models 取），得到 %d", base.ProviderContextMaxTokens)
	}
	// ★ 生成参数零直读：基线这五项必须是「未配置」机制语义
	if base.Temperature != -1 {
		t.Errorf("基线温度应为 -1（未配置=不下发），得到 %v", base.Temperature)
	}
	if base.MaxTokens != 0 {
		t.Errorf("基线最大输出应为 0（未配置），得到 %d", base.MaxTokens)
	}
	if base.ThinkingMode != "" {
		t.Errorf("基线思考档位应为空（未配置），得到 %q", base.ThinkingMode)
	}
	if base.ContextMaxTokens != 0 {
		t.Errorf("基线上下文窗口应为 0（未配置），得到 %d", base.ContextMaxTokens)
	}
	if base.ModelParams != nil {
		t.Errorf("基线不应携带模型级参数表（唯一来源 models.json），得到 %v", base.ModelParams)
	}
}
