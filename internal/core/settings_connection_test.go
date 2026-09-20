package core

import (
	"encoding/json"
	"os"
	"testing"
)

// TestMigrateLegacyConnectionToPreset settings 顶层旧连接字段一次性迁入
// ai-presets.json 的一条 AI 配置，并把 settings.preset 指向它、清空顶层旧字段
// ——此后连接信息唯一来源 = AI 配置（核心零直读连接字段）。
func TestMigrateLegacyConnectionToPreset(t *testing.T) {
	withTempInstall(t, func() {
		oldPresets := PresetList
		defer func() { PresetList = oldPresets }()

		Settings = Default()
		Settings.Provider = "prov-legacy"
		Settings.BaseURL = "https://legacy.example/v1"
		Settings.APIKey = "LEGACY-KEY"
		Settings.ExecuteModel = "m-legacy"

		MigrateLegacyConnectionToPreset()

		// ① 生成一条配置（preset 为空 → 用默认名），四个连接字段完整迁入
		if Settings.Preset != LegacyConnectionPresetName {
			t.Errorf("settings.preset 应指向迁移生成的配置 %q，得到 %q", LegacyConnectionPresetName, Settings.Preset)
		}
		p := GetPreset(LegacyConnectionPresetName)
		if p.Provider != "prov-legacy" || p.BaseURL != "https://legacy.example/v1" ||
			p.APIKey != "LEGACY-KEY" || p.ExecuteModel != "m-legacy" {
			t.Errorf("连接字段应完整迁入配置，得到 %+v", p)
		}

		// ② 顶层旧字段清空（核心零直读）
		if Settings.Provider != "" || Settings.BaseURL != "" || Settings.APIKey != "" ||
			Settings.Model != "" || Settings.ExecuteModel != "" ||
			Settings.PlanModel != "" || Settings.ReviewModel != "" {
			t.Errorf("迁移后顶层连接字段应清空，得到 provider=%q baseURL=%q apiKey=%q model=%q execute=%q",
				Settings.Provider, Settings.BaseURL, Settings.APIKey, Settings.Model, Settings.ExecuteModel)
		}

		// ③ 落盘：settings.json 顶层不再含连接键（omitempty）；ai-presets.json 含该配置
		data, err := os.ReadFile(SettingsPath())
		if err != nil {
			t.Fatalf("迁移结果未落盘 settings.json: %v", err)
		}
		var top map[string]any
		if err := json.Unmarshal(data, &top); err != nil {
			t.Fatalf("settings.json 解析失败: %v", err)
		}
		for _, k := range []string{"provider", "baseURL", "apiKey", "model", "executeModel", "planModel", "reviewModel"} {
			if _, ok := top[k]; ok {
				t.Errorf("迁移后 settings.json 顶层不应再含连接键 %q：%s", k, string(data))
			}
		}
		if top["preset"] != LegacyConnectionPresetName {
			t.Errorf("settings.json 的 preset 应为迁移配置名，得到 %v", top["preset"])
		}
		pdata, err := os.ReadFile(AiPresetsPath())
		if err != nil {
			t.Fatalf("迁移结果未落盘 ai-presets.json: %v", err)
		}
		var saved AiPresets
		if err := json.Unmarshal(pdata, &saved); err != nil {
			t.Fatalf("ai-presets.json 解析失败: %v", err)
		}
		if saved[LegacyConnectionPresetName].APIKey != "LEGACY-KEY" {
			t.Errorf("ai-presets.json 应含迁入的连接配置，得到 %s", string(pdata))
		}

		// ④ 幂等：重复调用不改变结果（顶层已清空 → 直接返回）
		MigrateLegacyConnectionToPreset()
		if Settings.Preset != LegacyConnectionPresetName {
			t.Errorf("迁移应幂等（preset 不变），得到 %q", Settings.Preset)
		}
		if got := GetPreset(LegacyConnectionPresetName).APIKey; got != "LEGACY-KEY" {
			t.Errorf("迁移应幂等（配置不变），得到 %q", got)
		}
	})
}

// TestMigrateLegacyConnectionDoesNotOverwritePreset 激活配置已有的字段优先，
// 旧顶层值只补该配置的空字段（配置是唯一来源，不得被旧存储覆盖）。
func TestMigrateLegacyConnectionDoesNotOverwritePreset(t *testing.T) {
	withTempInstall(t, func() {
		oldPresets := PresetList
		defer func() { PresetList = oldPresets }()

		SetAiPresets(AiPresets{
			"工作配置": {Provider: "p-new", APIKey: "NEW-KEY", ExecuteModel: "m-new"},
		})
		Settings = Default()
		Settings.Preset = "工作配置"
		Settings.Provider = "p-old"
		Settings.BaseURL = "https://old.example/v1"
		Settings.APIKey = "OLD-KEY"
		Settings.ExecuteModel = "m-old"

		MigrateLegacyConnectionToPreset()

		p := GetPreset("工作配置")
		if p.Provider != "p-new" || p.APIKey != "NEW-KEY" || p.ExecuteModel != "m-new" {
			t.Errorf("配置已有字段不得被旧顶层值覆盖，得到 %+v", p)
		}
		if p.BaseURL != "https://old.example/v1" {
			t.Errorf("配置空缺字段应由旧顶层值补齐，得到 %q", p.BaseURL)
		}
		if Settings.Preset != "工作配置" {
			t.Errorf("激活配置名应保持不变，得到 %q", Settings.Preset)
		}
		if Settings.APIKey != "" || Settings.Provider != "" {
			t.Errorf("迁移后顶层连接字段应清空，得到 provider=%q apiKey=%q", Settings.Provider, Settings.APIKey)
		}
	})
}

// TestMigrateLegacyConnectionNoop 顶层无连接字段时不动作（不产生空配置、不改 preset）。
func TestMigrateLegacyConnectionNoop(t *testing.T) {
	withTempInstall(t, func() {
		oldPresets := PresetList
		defer func() { PresetList = oldPresets }()

		SetAiPresets(AiPresets{})
		Settings = Default()

		MigrateLegacyConnectionToPreset()

		if Settings.Preset != "" {
			t.Errorf("无遗留值不应改动 preset，得到 %q", Settings.Preset)
		}
		if len(GetAiPresets()) != 0 {
			t.Errorf("无遗留值不应生成配置，得到 %v", GetAiPresets())
		}
	})
}

// TestAiPresetFromSettingsUsesActivePreset 保存配置时的快照来源 = 激活配置
// （连接信息）+ 插件注册域（生成参数）——核心字段零参与。
func TestAiPresetFromSettingsUsesActivePreset(t *testing.T) {
	oldPresets, oldSettings := PresetList, Settings
	defer func() { PresetList, Settings = oldPresets, oldSettings }()

	SetAiPresets(AiPresets{
		"激活": {Provider: "p1", BaseURL: "https://p1/v1", APIKey: "K1", ExecuteModel: "m1",
			Protocol: "anthropic-messages"},
	})
	Settings = Default()
	Settings.Preset = "激活"
	Settings.PluginSettings = map[string]map[string]any{
		GenerationSettingsKey: {"temperature": "0.7", "thinkingMode": "low", "maxTokens": float64(4096)},
	}
	// 顶层旧字段即使被写回也不参与快照
	Settings.Provider = "p-legacy"
	Settings.APIKey = "LEGACY"
	Settings.ExecuteModel = "m-legacy"

	p := AiPresetFromSettings()
	if p.Provider != "p1" || p.APIKey != "K1" || p.ExecuteModel != "m1" || p.Protocol != "anthropic-messages" {
		t.Errorf("快照应取激活配置的连接信息，得到 %+v", p)
	}
	if p.PlanModel != "m1" || p.ReviewModel != "m1" {
		t.Errorf("统一模型：plan/review 应跟随执行模型，得到 plan=%q review=%q", p.PlanModel, p.ReviewModel)
	}
	if p.Temperature != "0.7" || p.ThinkingMode != "low" || p.MaxTokens != 4096 {
		t.Errorf("快照的生成参数应取插件注册域，得到 %+v", p)
	}
}

// TestRenamePresetProvider 服务商改名 → AI 配置中引用旧名的条目同步更新
// （连接信息唯一来源是配置，不改配置等于改名后配置指向不存在的服务商）。
func TestRenamePresetProvider(t *testing.T) {
	withTempInstall(t, func() {
		oldPresets := PresetList
		defer func() { PresetList = oldPresets }()

		SetAiPresets(AiPresets{
			"a": {Provider: "旧名", APIKey: "KA", ExecuteModel: "ma"},
			"b": {Provider: "别的", APIKey: "KB", ExecuteModel: "mb"},
			"c": {Provider: "旧名", APIKey: "KC", ExecuteModel: "mc"},
		})
		if n := RenamePresetProvider("旧名", "新名"); n != 2 {
			t.Fatalf("应更新 2 条配置，得到 %d", n)
		}
		if got := GetPreset("a").Provider; got != "新名" {
			t.Errorf("配置 a 的服务商应更新为新名，得到 %q", got)
		}
		if got := GetPreset("c").Provider; got != "新名" {
			t.Errorf("配置 c 的服务商应更新为新名，得到 %q", got)
		}
		if got := GetPreset("b").Provider; got != "别的" {
			t.Errorf("不匹配的配置不得被改动，得到 %q", got)
		}
		// 落盘
		if data, err := os.ReadFile(AiPresetsPath()); err == nil {
			var saved AiPresets
			_ = json.Unmarshal(data, &saved)
			if saved["a"].Provider != "新名" {
				t.Errorf("改名结果应落盘 ai-presets.json，得到 %s", string(data))
			}
		} else {
			t.Errorf("读取 ai-presets.json 失败: %v", err)
		}
		// 无匹配 → 不动作
		if n := RenamePresetProvider("不存在", "x"); n != 0 {
			t.Errorf("无匹配时应返回 0，得到 %d", n)
		}
	})
}
