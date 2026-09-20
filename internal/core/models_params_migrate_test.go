package core

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestProviderEntryParamFieldsJSON 服务商生成参数字段（温度/最大输出/模型级参数）JSON 往返。
// ★ 2026-09-19：这三个参数的唯一来源是 models.json（服务商条目），必须能正确序列化/反序列化。
func TestProviderEntryParamFieldsJSON(t *testing.T) {
	in := ProviderEntry{
		BaseURL:     "https://api.example.com/v1",
		Models:      []string{"m1", "m2"},
		Temperature: "0.7",
		MaxTokens:   32000,
		ModelParams: map[string]ModelParamEntry{
			"m1": {Temperature: "1.0", MaxTokens: 13000, ContextMaxTokens: 128000, Multimodal: true},
		},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out ProviderEntry
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Temperature != "0.7" || out.MaxTokens != 32000 {
		t.Errorf("服务商级参数往返失败: temp=%q max=%d", out.Temperature, out.MaxTokens)
	}
	mp := out.ModelParams["m1"]
	if mp.Temperature != "1.0" || mp.MaxTokens != 13000 || mp.ContextMaxTokens != 128000 || !mp.Multimodal {
		t.Errorf("模型级参数往返失败: %+v", mp)
	}
	// 未配置时不落盘（omitempty，保持 models.json 干净）
	empty, _ := json.Marshal(ProviderEntry{BaseURL: "x"})
	for _, k := range []string{"temperature", "maxTokens", "modelParams"} {
		if strings.Contains(string(empty), k) {
			t.Errorf("空字段 %q 不应写入 models.json: %s", k, string(empty))
		}
	}
}

// withTempInstall 在临时目录中运行 fn（ConfigDir/InstallDir 回退 wd/config），并恢复全局状态。
func withTempInstall(t *testing.T, fn func()) {
	t.Helper()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	oldSettings, oldList := Settings, ModelList
	defer func() {
		Settings, ModelList = oldSettings, oldList
		_ = os.Chdir(oldWD)
	}()
	fn()
}

// TestMigrateParamSettingsToModels settings 的生成参数一次性迁入 models.json（模型级 + 服务商级），且幂等。
func TestMigrateParamSettingsToModels(t *testing.T) {
	withTempInstall(t, func() {
		Settings = Default()
		Settings.Provider = "prov1"
		Settings.Temperature = "0.9"
		Settings.MaxTokens = 22222
		Settings.ContextMaxTokens = 333333
		Settings.ModelParams = map[string]map[string]ModelParamEntry{
			"prov1": {"m1": {Temperature: "1.0", MaxTokens: 13000, Multimodal: true}},
		}
		ModelList = ModelListMap{"prov1": {BaseURL: "https://x/v1", Models: []string{"m1"}}}

		MigrateParamSettingsToModels()

		e := GetProviderEntry("prov1")
		if e.Temperature != "0.9" {
			t.Errorf("服务商级温度应迁移 0.9，得到 %q", e.Temperature)
		}
		if e.MaxTokens != 22222 {
			t.Errorf("服务商级最大输出应迁移 22222，得到 %d", e.MaxTokens)
		}
		if e.ContextMaxTokens != 333333 {
			t.Errorf("服务商级上下文窗口应迁移 333333，得到 %d", e.ContextMaxTokens)
		}
		mp := e.ModelParams["m1"]
		if mp.Temperature != "1.0" || mp.MaxTokens != 13000 || !mp.Multimodal {
			t.Errorf("模型级参数应迁移完整，得到 %+v", mp)
		}
		if _, err := os.Stat(ModelsPath()); err != nil {
			t.Fatalf("迁移结果未落盘 models.json: %v", err)
		}

		// 幂等：标记文件已写 → 再次迁移不动（即使内存被清空）
		ModelList = ModelListMap{"prov1": {BaseURL: "https://x/v1", Models: []string{"m1"}}}
		MigrateParamSettingsToModels()
		if got := GetProviderEntry("prov1").Temperature; got != "" {
			t.Errorf("迁移应幂等（标记文件生效），得到 %q", got)
		}
		if got := GetProviderEntry("prov1").ContextMaxTokens; got != 0 {
			t.Errorf("迁移应幂等，得到 contextMaxTokens=%d", got)
		}
	})
}

// TestMigrateParamSettingsDoesNotOverwriteModels 已有 models.json 配置优先，迁移不得覆盖。
func TestMigrateParamSettingsDoesNotOverwriteModels(t *testing.T) {
	withTempInstall(t, func() {
		Settings = Default()
		Settings.Provider = "prov1"
		Settings.Temperature = "0.9"
		Settings.MaxTokens = 22222
		Settings.ContextMaxTokens = 333333
		Settings.ModelParams = map[string]map[string]ModelParamEntry{
			"prov1": {"m1": {Temperature: "1.0"}},
		}
		ModelList = ModelListMap{"prov1": {
			BaseURL:          "https://x/v1",
			Models:           []string{"m1"},
			Temperature:      "0.1",
			MaxTokens:        999,
			ContextMaxTokens: 555,
			ModelParams:      map[string]ModelParamEntry{"m1": {Temperature: "0.2"}},
		}}

		MigrateParamSettingsToModels()

		e := GetProviderEntry("prov1")
		if e.Temperature != "0.1" || e.MaxTokens != 999 || e.ContextMaxTokens != 555 {
			t.Errorf("models.json 已有服务商级配置不得被 settings 覆盖: %+v", e)
		}
		if got := e.ModelParams["m1"].Temperature; got != "0.2" {
			t.Errorf("models.json 已有模型级配置不得被覆盖，得到 %q", got)
		}
	})
}

// TestProviderParamGetters 三类查询（温度/最大输出/模型级参数）读取 models.json 数据面。
func TestProviderParamGetters(t *testing.T) {
	withTempInstall(t, func() {
		ModelList = ModelListMap{
			"a": {Temperature: "0.5", MaxTokens: 111, ModelParams: map[string]ModelParamEntry{"m": {Multimodal: true}}},
			"b": {BaseURL: "https://b"},
		}
		temps := GetProviderTemperatures()
		if temps["a"] != "0.5" || temps["b"] != "" {
			t.Errorf("GetProviderTemperatures 异常: %+v", temps)
		}
		maxes := GetProviderMaxTokens()
		if maxes["a"] != 111 || maxes["b"] != 0 {
			t.Errorf("GetProviderMaxTokens 异常: %+v", maxes)
		}
		mps := GetProviderModelParams()
		if len(mps["a"]) != 1 || !mps["a"]["m"].Multimodal {
			t.Errorf("GetProviderModelParams 异常: %+v", mps)
		}
		if _, ok := mps["b"]; ok {
			t.Errorf("无模型参数的服务商不应出现在映射中: %+v", mps)
		}
	})
}
