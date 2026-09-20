package agent

import (
	"testing"

	"github.com/hoonfeng/paircode/internal/core"
)

// TestContextWindowUsesAssembledValue 上下文窗口只消费装配结果。
// ★ 2026-09-20：核心零直读生成参数——settings 顶层 contextMaxTokens 已迁入插件注册域
// （pluginSettings.generation）并清空；装配器未给值时回退机制兜底常量（运行保障，非配置面）。
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

// TestResolveProviderBaseHasNoGenerationParams 裸基线不携带任何生成参数
// （★ 2026-09-20：温度/思考档位/最大输出/上下文窗口/模型级参数表一律由插件装配器决策——
// 全局默认取自插件注册配置域 pluginSettings.generation，模型/服务商级取自 models.json）。
func TestResolveProviderBaseHasNoGenerationParams(t *testing.T) {
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
	core.Settings.Provider = "prov-x"
	// 即使旧字段被写回（历史数据/外部写入），基线也不得携带它们
	core.Settings.Temperature = "0.9"
	core.Settings.ThinkingMode = "max"
	core.Settings.MaxTokens = 99999
	core.Settings.ContextMaxTokens = 88888
	core.Settings.ModelParams = map[string]map[string]core.ModelParamEntry{
		"prov-x": {"m1": {Temperature: "0.1"}},
	}

	base := resolveProviderBase()
	if base.ProviderContextMaxTokens != 1000000 {
		t.Errorf("服务商级上下文窗口应透传（数据面预查，供装配器决策），得到 %d", base.ProviderContextMaxTokens)
	}
	if base.Provider != "prov-x" {
		t.Errorf("服务商应为 prov-x，得到 %q", base.Provider)
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
