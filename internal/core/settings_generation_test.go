package core

import (
	"encoding/json"
	"os"
	"testing"
)

// TestMigrateGenerationSettingsFromLegacy settings 顶层生成参数一次性迁入插件注册域
// （pluginSettings.generation）并清空旧字段——此后 Go 内核零直读生成参数。
// ★ 2026-09-20：配置项一律由插件 ctx.registerSettings 注册（值存 pluginSettings.<key>）。
func TestMigrateGenerationSettingsFromLegacy(t *testing.T) {
	withTempInstall(t, func() {
		Settings = Default()
		Settings.Temperature = "0.9"
		Settings.ThinkingMode = "max"
		Settings.MaxTokens = 22222
		Settings.ContextMaxTokens = 333333
		Settings.ModelParams = map[string]map[string]ModelParamEntry{
			"p": {"m": {Temperature: "1.0", MaxTokens: 13000}},
		}

		MigrateGenerationSettingsFromLegacy()

		// ① 四项全局默认进插件注册域
		if got := GenerationTemperature(); got != "0.9" {
			t.Errorf("温度应迁入插件注册域，得到 %q", got)
		}
		if got := GenerationThinkingMode(); got != "max" {
			t.Errorf("思考档位应迁入插件注册域，得到 %q", got)
		}
		if got := GenerationMaxTokens(); got != 22222 {
			t.Errorf("最大输出应迁入插件注册域，得到 %d", got)
		}
		if got := GenerationContextMaxTokens(); got != 333333 {
			t.Errorf("上下文窗口应迁入插件注册域，得到 %d", got)
		}

		// ② 旧字段清空（模型级参数已由 MigrateParamSettingsToModels 搬进 models.json）
		if Settings.Temperature != "" || Settings.ThinkingMode != "" ||
			Settings.MaxTokens != 0 || Settings.ContextMaxTokens != 0 || Settings.ModelParams != nil {
			t.Errorf("迁移后旧字段应清空（核心零直读），得到 temp=%q think=%q max=%d ctx=%d mp=%v",
				Settings.Temperature, Settings.ThinkingMode, Settings.MaxTokens,
				Settings.ContextMaxTokens, Settings.ModelParams)
		}

		// ③ 落盘 settings.json 顶层不再含旧键（omitempty），值已转到插件注册域
		data, err := os.ReadFile(SettingsPath())
		if err != nil {
			t.Fatalf("迁移结果未落盘 settings.json: %v", err)
		}
		var top map[string]any
		if err := json.Unmarshal(data, &top); err != nil {
			t.Fatalf("settings.json 解析失败: %v", err)
		}
		for _, k := range []string{"temperature", "thinkingMode", "maxTokens", "contextMaxTokens", "modelParams"} {
			if _, ok := top[k]; ok {
				t.Errorf("迁移后 settings.json 顶层不应再含旧键 %q：%s", k, string(data))
			}
		}
		ps, _ := top["pluginSettings"].(map[string]any)
		gen, _ := ps[GenerationSettingsKey].(map[string]any)
		if gen["temperature"] != "0.9" || gen["thinkingMode"] != "max" {
			t.Errorf("插件注册域应含迁入的生成参数，得到 %v", ps)
		}

		// ④ 幂等：重复调用不改变插件注册域值
		MigrateGenerationSettingsFromLegacy()
		if got := GenerationTemperature(); got != "0.9" {
			t.Errorf("迁移应幂等，得到 %q", got)
		}
		if got := GenerationContextMaxTokens(); got != 333333 {
			t.Errorf("迁移应幂等，得到 %d", got)
		}
	})
}

// TestMigrateGenerationDoesNotOverwritePluginConfig 插件注册域已有值优先，
// 旧顶层字段不得覆盖用户在「生成参数」面板里设过的值。
func TestMigrateGenerationDoesNotOverwritePluginConfig(t *testing.T) {
	withTempInstall(t, func() {
		Settings = Default()
		Settings.PluginSettings = map[string]map[string]any{
			GenerationSettingsKey: {"temperature": "0.1", "maxTokens": float64(1234)},
		}
		Settings.Temperature = "0.9"
		Settings.ThinkingMode = "max"
		Settings.MaxTokens = 22222

		MigrateGenerationSettingsFromLegacy()

		if got := GenerationTemperature(); got != "0.1" {
			t.Errorf("插件注册域已有的温度不得被旧字段覆盖，得到 %q", got)
		}
		if got := GenerationMaxTokens(); got != 1234 {
			t.Errorf("插件注册域已有的最大输出不得被旧字段覆盖，得到 %d", got)
		}
		// 未配置的键仍照迁（thinkingMode 缺失 → 从旧字段补）
		if got := GenerationThinkingMode(); got != "max" {
			t.Errorf("未配置的键应从旧字段补迁，得到 %q", got)
		}
	})
}

// TestGenerationDefaultValueMissing 无插件配置（插件未装载/未设置）时读取返回零值，
// 装配器据此走 GEN_DEFAULTS 兜底（核心不持有生成参数默认值）。
func TestGenerationDefaultValueMissing(t *testing.T) {
	old := Settings
	defer func() { Settings = old }()
	Settings = Default()

	if got := GenerationTemperature(); got != "" {
		t.Errorf("未配置温度应为空，得到 %q", got)
	}
	if got := GenerationMaxTokens(); got != 0 {
		t.Errorf("未配置最大输出应为 0，得到 %d", got)
	}
	if got := GenerationContextMaxTokens(); got != 0 {
		t.Errorf("未配置上下文窗口应为 0，得到 %d", got)
	}
	if got := DefaultContextWindow; got <= 0 {
		t.Errorf("机制兜底常量必须为正（否则 Loop 精简无窗口），得到 %d", got)
	}
}
