package core

import (
	"encoding/json"
	"os"
	"testing"
)

// TestGetProviderProtocolDefault 服务商协议读取（models.json protocol 字段；空=默认）。
func TestGetProviderProtocolDefault(t *testing.T) {
	SetModelList(ModelListMap{
		"anthropic": {BaseURL: "https://api.anthropic.com/v1", Protocol: "anthropic-messages"},
		"deepseek":  {BaseURL: "https://api.deepseek.com/v1"}, // 无协议 → 默认 openai-completions
	})
	if got := GetProviderProtocol("anthropic"); got != "anthropic-messages" {
		t.Errorf("anthropic 协议 = %q，期望 anthropic-messages", got)
	}
	if got := GetProviderProtocol("deepseek"); got != "" {
		t.Errorf("deepseek 协议 = %q，期望空（默认 openai-completions）", got)
	}
	if got := GetProviderProtocol("不存在"); got != "" {
		t.Errorf("未知服务商协议 = %q，期望空", got)
	}
}

// TestGetProviderProtocols 服务商 → 协议映射完整性。
func TestGetProviderProtocols(t *testing.T) {
	SetModelList(ModelListMap{
		"anthropic": {Protocol: "anthropic-messages"},
		"openai":    {Protocol: "openai-responses"},
		"deepseek":  {},
	})
	m := GetProviderProtocols()
	if m["anthropic"] != "anthropic-messages" || m["openai"] != "openai-responses" {
		t.Errorf("GetProviderProtocols = %v", m)
	}
	if _, ok := m["deepseek"]; !ok {
		t.Errorf("GetProviderProtocols 应含 deepseek（空值也保留键）：%v", m)
	}
}

// TestLoadModelListFallback 无配置文件 → 内置默认含 anthropic 原生协议。
func TestLoadModelListFallback(t *testing.T) {
	// 当前测试 cwd 无 config/ → useDefaultModels 兜底
	ModelList = nil
	LoadModelList()
	if got := GetProviderProtocol("anthropic"); got != "anthropic-messages" {
		t.Errorf("内置默认 anthropic 协议 = %q，期望 anthropic-messages", got)
	}
	if got := GetProviderBaseURL("anthropic"); got != "https://api.anthropic.com/v1" {
		t.Errorf("内置默认 anthropic base = %q", got)
	}
}

// TestRenameProvider 服务商改名：models.json 键迁移（条目内容完整保留）+ AI 配置引用同步 + 拒绝分支。
//
// ★ 2026-09-20 接线后锁定语义：服务商名既是 models.json 的键，也是 ai-presets.json 里
// AI 配置的连接引用（连接信息唯一来源）。改名只改一处 = 配置指向不存在的服务商。
func TestRenameProvider(t *testing.T) {
	withTempInstall(t, func() {
		oldPresets := PresetList
		defer func() { PresetList = oldPresets }()

		SetModelList(ModelListMap{
			"旧服务商": {
				BaseURL: "https://old.example.com/v1", Protocol: "anthropic-messages",
				Models: []string{"m1", "m2"}, Temperature: "0.5", MaxTokens: 8192,
				ContextMaxTokens: 200000,
				ModelParams:      map[string]ModelParamEntry{"m1": {Temperature: "0.9", Multimodal: true}},
			},
			"别的": {BaseURL: "https://other.example.com/v1"},
		})
		SetAiPresets(AiPresets{
			"配置A": {Provider: "旧服务商", APIKey: "KA", ExecuteModel: "m1"},
			"配置B": {Provider: "别的", APIKey: "KB"},
			"配置C": {Provider: "旧服务商", APIKey: "KC"},
		})

		// 受影响配置清单（改名/删除前给用户的提示依据）
		if got := PresetsReferencing("旧服务商"); len(got) != 2 || got[0] != "配置A" || got[1] != "配置C" {
			t.Errorf("PresetsReferencing = %v，期望 [配置A 配置C]", got)
		}
		if got := PresetsReferencing("别的"); len(got) != 1 || got[0] != "配置B" {
			t.Errorf("PresetsReferencing(别的) = %v", got)
		}
		if got := PresetsReferencing(""); len(got) != 0 {
			t.Errorf("空服务商名应返回空清单，得到 %v", got)
		}

		if err := RenameProvider("旧服务商", "新服务商"); err != nil {
			t.Fatalf("改名失败: %v", err)
		}

		// ① models.json 键迁移，条目内容一个不少（地址/协议/模型/服务商级与模型级参数）
		if _, ok := ModelList["旧服务商"]; ok {
			t.Error("改名后旧键不应残留")
		}
		e := GetProviderEntry("新服务商")
		if e.BaseURL != "https://old.example.com/v1" || e.Protocol != "anthropic-messages" {
			t.Errorf("改名后连接信息丢失: %+v", e)
		}
		if len(e.Models) != 2 || e.Models[0] != "m1" {
			t.Errorf("改名后模型列表丢失: %v", e.Models)
		}
		if e.Temperature != "0.5" || e.MaxTokens != 8192 || e.ContextMaxTokens != 200000 {
			t.Errorf("改名后服务商级参数丢失: %+v", e)
		}
		if mp := e.ModelParams["m1"]; mp.Temperature != "0.9" || !mp.Multimodal {
			t.Errorf("改名后模型级参数丢失: %+v", mp)
		}
		// 其他服务商不受影响
		if GetProviderBaseURL("别的") != "https://other.example.com/v1" {
			t.Error("改名不应影响其他服务商")
		}

		// ② AI 配置引用同步（内存 + 落盘）
		if got := GetPreset("配置A").Provider; got != "新服务商" {
			t.Errorf("配置A 的 provider 应同步为新名，得到 %q", got)
		}
		if got := GetPreset("配置C").Provider; got != "新服务商" {
			t.Errorf("配置C 的 provider 应同步为新名，得到 %q", got)
		}
		if got := GetPreset("配置B").Provider; got != "别的" {
			t.Errorf("不匹配的配置不得改动，得到 %q", got)
		}

		data, err := os.ReadFile(ModelsPath())
		if err != nil {
			t.Fatalf("读取 models.json 失败: %v", err)
		}
		var saved ModelListMap
		if err := json.Unmarshal(data, &saved); err != nil {
			t.Fatalf("models.json 解析失败: %v", err)
		}
		if _, ok := saved["旧服务商"]; ok {
			t.Errorf("旧键不应落盘: %s", string(data))
		}
		if saved["新服务商"].BaseURL != "https://old.example.com/v1" {
			t.Errorf("新键未落盘或内容缺失: %s", string(data))
		}

		pdata, err := os.ReadFile(AiPresetsPath())
		if err != nil {
			t.Fatalf("读取 ai-presets.json 失败: %v", err)
		}
		var savedPresets AiPresets
		if err := json.Unmarshal(pdata, &savedPresets); err != nil {
			t.Fatalf("ai-presets.json 解析失败: %v", err)
		}
		if savedPresets["配置A"].Provider != "新服务商" {
			t.Errorf("配置引用未落盘: %s", string(pdata))
		}

		// ③ 拒绝分支：不存在 / 新名重复 / 空新名 —— 且失败不得破坏数据
		if err := RenameProvider("不存在", "x"); err == nil {
			t.Error("对不存在的服务商改名应报错")
		}
		if err := RenameProvider("新服务商", "别的"); err == nil {
			t.Error("新名与既有服务商重复应报错")
		}
		if err := RenameProvider("新服务商", ""); err == nil {
			t.Error("空新名应报错")
		}
		if _, ok := ModelList["新服务商"]; !ok {
			t.Error("失败的改名不得删除原条目")
		}
		if len(ModelList) != 2 {
			t.Errorf("失败的改名不得增删服务商，现有 %d 个", len(ModelList))
		}
	})
}
