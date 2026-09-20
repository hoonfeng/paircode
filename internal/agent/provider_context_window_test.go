package agent

import (
	"testing"

	"github.com/hoonfeng/paircode/internal/core"
)

// TestContextWindowPrefersProviderConfig 上下文窗口一律以服务商配置（models.json，经装配器）为准；
// 仅在服务商/模型都未配置时才回退 settings 顶层值（★ 2026-09-19 缺陷修复：
// 此前 web 层直读 core.Settings.ContextMaxTokens，models.json 的服务商级配置不生效）。
func TestContextWindowPrefersProviderConfig(t *testing.T) {
	old := core.Settings.ContextMaxTokens
	defer func() { core.Settings.ContextMaxTokens = old }()
	core.Settings.ContextMaxTokens = 64000

	if got := ContextWindow(ProviderParams{ContextMaxTokens: 1000000}); got != 1000000 {
		t.Fatalf("服务商配置应优先：want 1000000, got %d", got)
	}
	if got := ContextWindow(ProviderParams{}); got != 64000 {
		t.Fatalf("未配置服务商窗口时应回退 settings：want 64000, got %d", got)
	}
	if got := ContextWindow(ProviderParams{ContextMaxTokens: -1}); got != 64000 {
		t.Fatalf("非法（<=0）服务商窗口应回退 settings：want 64000, got %d", got)
	}
}

// TestResolveProviderBaseCarriesProviderContext 裸基线携带 models.json 的服务商级上下文窗口
// （供装配器决策；模型级参数表一并透传）。
func TestResolveProviderBaseCarriesProviderContext(t *testing.T) {
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

	base := resolveProviderBase()
	if base.ProviderContextMaxTokens != 1000000 {
		t.Errorf("服务商级上下文窗口应透传，得到 %d", base.ProviderContextMaxTokens)
	}
	if base.Provider != "prov-x" {
		t.Errorf("服务商应为 prov-x，得到 %q", base.Provider)
	}
}
