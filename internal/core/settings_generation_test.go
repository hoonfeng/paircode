package core

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestMigrateGenerationToProvider 生成参数的旧来源（settings 顶层旧字段 + 已移除的
// 「生成参数」注册段 pluginSettings.generation）一次性迁入「激活配置对应服务商」的
// 服务商级字段，并清空旧来源。
//
// ★ 2026-09-25 生成参数唯一来源 = 服务商配置（models.json 服务商级 + 模型级）；
// 设置面板「生成参数」页已移除（它与服务商配置重复且优先级最低）。
func TestMigrateGenerationToProvider(t *testing.T) {
	withTempInstall(t, func() {
		Settings = Default()
		Settings.Preset = "cfgA"
		Settings.Temperature = "0.9"
		Settings.ThinkingMode = "max"
		Settings.MaxTokens = 22222
		Settings.ContextMaxTokens = 333333
		Settings.PluginSettings = map[string]map[string]any{
			LegacyGenerationSettingsKey: {"temperature": "0.1", "maxTokens": float64(1234)},
		}
		ModelList = ModelListMap{"prov1": {BaseURL: "https://x/v1", Models: []string{"m1"}}}
		PresetList = AiPresets{"cfgA": {Provider: "prov1", ExecuteModel: "m1"}}
		defer func() { PresetList = nil }()

		MigrateGenerationToProvider()

		// ① 四项都落到激活配置对应服务商的服务商级字段
		//   （顶层旧字段更早即存在，优先于注册段旧值：温度取 0.9 而非注册段的 0.1）
		e := GetProviderEntry("prov1")
		if e.Temperature != "0.9" {
			t.Errorf("服务商级温度应迁入 0.9，得到 %q", e.Temperature)
		}
		if e.ThinkingMode != "max" {
			t.Errorf("服务商级思考档位应迁入 max，得到 %q", e.ThinkingMode)
		}
		if e.MaxTokens != 22222 {
			t.Errorf("服务商级最大输出应迁入 22222，得到 %d", e.MaxTokens)
		}
		if e.ContextMaxTokens != 333333 {
			t.Errorf("服务商级上下文窗口应迁入 333333，得到 %d", e.ContextMaxTokens)
		}

		// ② 旧来源清空（此后服务商配置是唯一来源）
		if Settings.Temperature != "" || Settings.ThinkingMode != "" ||
			Settings.MaxTokens != 0 || Settings.ContextMaxTokens != 0 || Settings.ModelParams != nil {
			t.Errorf("迁移后 settings 顶层旧字段应清空，得到 temp=%q think=%q max=%d ctx=%d mp=%v",
				Settings.Temperature, Settings.ThinkingMode, Settings.MaxTokens,
				Settings.ContextMaxTokens, Settings.ModelParams)
		}
		if _, ok := Settings.PluginSettings[LegacyGenerationSettingsKey]; ok {
			t.Errorf("迁移后注册段 pluginSettings.%s 应删除，得到 %v",
				LegacyGenerationSettingsKey, Settings.PluginSettings)
		}

		// ③ 落盘：models.json 携带服务商级值；settings.json 不再含该注册段
		data, err := os.ReadFile(ModelsPath())
		if err != nil {
			t.Fatalf("迁移结果未落盘 models.json: %v", err)
		}
		var top map[string]any
		if err := json.Unmarshal(data, &top); err != nil {
			t.Fatalf("models.json 解析失败: %v", err)
		}
		prov, _ := top["prov1"].(map[string]any)
		if prov["temperature"] != "0.9" || prov["thinkingMode"] != "max" {
			t.Errorf("models.json 服务商条目应含迁入的生成参数: %s", string(data))
		}
		sdata, err := os.ReadFile(SettingsPath())
		if err != nil {
			t.Fatalf("迁移结果未落盘 settings.json: %v", err)
		}
		if strings.Contains(string(sdata), LegacyGenerationSettingsKey) {
			t.Errorf("settings.json 不应再含 generation 注册段: %s", string(sdata))
		}

		// ④ 幂等：旧来源已清空 → 再跑不动（内存被清空也不回填）
		ModelList["prov1"] = ProviderEntry{BaseURL: "https://x/v1", Models: []string{"m1"}}
		MigrateGenerationToProvider()
		if got := GetProviderEntry("prov1").Temperature; got != "" {
			t.Errorf("迁移应幂等（旧来源已清空），得到 %q", got)
		}
	})
}

// TestMigrateGenerationToProviderDoesNotOverwrite 服务商配置里已有的值优先——
// 旧来源（含设置面板曾设过的全局默认）不得覆盖它，但仍一并清空旧来源。
func TestMigrateGenerationToProviderDoesNotOverwrite(t *testing.T) {
	withTempInstall(t, func() {
		Settings = Default()
		Settings.Preset = "cfgA"
		Settings.Temperature = "0.9"
		Settings.ThinkingMode = "max"
		Settings.PluginSettings = map[string]map[string]any{
			LegacyGenerationSettingsKey: {"temperature": "0.1"},
		}
		ModelList = ModelListMap{"prov1": {
			BaseURL: "https://x/v1", Models: []string{"m1"},
			Temperature: "0.1", ThinkingMode: "low",
		}}
		PresetList = AiPresets{"cfgA": {Provider: "prov1"}}
		defer func() { PresetList = nil }()

		MigrateGenerationToProvider()

		e := GetProviderEntry("prov1")
		if e.Temperature != "0.1" || e.ThinkingMode != "low" {
			t.Errorf("服务商配置已有值不得被旧值覆盖: %+v", e)
		}
		if Settings.Temperature != "" || Settings.ThinkingMode != "" {
			t.Errorf("旧来源仍应清空（服务商配置为唯一来源），得到 temp=%q think=%q",
				Settings.Temperature, Settings.ThinkingMode)
		}
	})
}

// TestMigrateGenerationToProviderKeepsLegacyWithoutProvider 无法定位目标服务商时
// **保留旧值**（下次启动重试），不得「清掉了又没落到任何地方」。
func TestMigrateGenerationToProviderKeepsLegacyWithoutProvider(t *testing.T) {
	withTempInstall(t, func() {
		Settings = Default()
		Settings.Preset = "" // 无激活配置 → 无目标服务商
		Settings.Temperature = "0.9"
		Settings.PluginSettings = map[string]map[string]any{
			LegacyGenerationSettingsKey: {"thinkingMode": "max"},
		}
		ModelList = ModelListMap{"prov1": {BaseURL: "https://x/v1"}}

		MigrateGenerationToProvider()

		if Settings.Temperature != "0.9" {
			t.Errorf("无激活服务商时应保留顶层旧值待重试，得到 %q", Settings.Temperature)
		}
		if _, ok := Settings.PluginSettings[LegacyGenerationSettingsKey]; !ok {
			t.Errorf("无激活服务商时不得清空注册段旧值: %v", Settings.PluginSettings)
		}
	})
}

// TestGetProviderThinkingModes 服务商级思考档位查询（前端「服务商」面板数据源）。
func TestGetProviderThinkingModes(t *testing.T) {
	withTempInstall(t, func() {
		ModelList = ModelListMap{
			"a": {ThinkingMode: "high"},
			"b": {BaseURL: "https://b"},
		}
		modes := GetProviderThinkingModes()
		if modes["a"] != "high" || modes["b"] != "" {
			t.Errorf("GetProviderThinkingModes 异常: %+v", modes)
		}
	})
}

// TestGenerationMechanismFallbackConstant 机制兜底常量必须为正（核心不持有生成参数默认值，
// 装配器侧用插件 GEN_DEFAULTS，宿主侧靠本常量保证 Loop 精简有窗口）。
func TestGenerationMechanismFallbackConstant(t *testing.T) {
	if DefaultContextWindow <= 0 {
		t.Errorf("机制兜底常量必须为正（否则 Loop 精简无窗口），得到 %d", DefaultContextWindow)
	}
}
