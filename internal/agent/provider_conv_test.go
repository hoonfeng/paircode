package agent

// 会话级配置装配测试（★ 2026-09-03）：会话记录配置名（preset）时按该配置整套展开，
// 含该配置的 Key——修复同服务商多配置时 GetPresetAPIKeyForProvider 按 map 无序
// 遍历猜 Key（取错配置 Key）的问题。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hoonfeng/paircode/internal/core"
)

// TestSetConvModel_PersistsPreset 落盘验证：SetConvModel 的 preset 持久化到 index.json。
func TestSetConvModel_PersistsPreset(t *testing.T) {
	dir := t.TempDir()
	s := NewMessageStore(dir)
	if err := s.CreateConversation("conv-p", "t", dir); err != nil {
		t.Fatal(err)
	}
	if err := s.SetConvModel("conv-p", "硅基流动", "m1", "ddd"); err != nil {
		t.Fatal(err)
	}
	// 直接读盘确认（绕开内存）
	raw, err := os.ReadFile(filepath.Join(dir, ".pair", "conversations", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var metas []ConversationMeta
	if err := json.Unmarshal(raw, &metas); err != nil {
		t.Fatal(err)
	}
	for _, m := range metas {
		if m.ID == "conv-p" {
			if m.Preset != "ddd" || m.Provider != "硅基流动" || m.Model != "m1" {
				t.Fatalf("落盘元数据不符：preset=%q provider=%q model=%q", m.Preset, m.Provider, m.Model)
			}
			return
		}
	}
	t.Fatal("index.json 中未找到 conv-p")
}

// seedConvTestPresets 布设测试配置：
//
//	激活预设（全局）：deepseek / DKEY / d-model（settings.preset 指向它）
//	ddd（错 Key）：  硅基流动 / BAD-KEY / ddd-model
//	硅基flash（对）: 硅基流动 / GOOD-KEY / flash-model
func seedConvTestPresets(t *testing.T) {
	t.Helper()
	core.SetAiPresets(map[string]core.AiPreset{
		"激活预设": {Provider: "deepseek", BaseURL: "https://ds.example", APIKey: "DKEY", ExecuteModel: "d-model"},
		"ddd":    {Provider: "硅基流动", BaseURL: "https://s.example", APIKey: "BAD-KEY", ExecuteModel: "ddd-model"},
		"硅基flash": {Provider: "硅基流动", BaseURL: "https://s.example", APIKey: "GOOD-KEY", ExecuteModel: "flash-model"},
	})
	core.Settings.Preset = "激活预设"
}

// hookConvLookup 注入会话级查询钩子（模拟 web 层 SetConvModelLookup）。
func hookConvLookup(t *testing.T, fn func(convID, wsRoot string) (string, string, string)) {
	t.Helper()
	SetConvModelLookup(fn)
}

// TestConvAssembly_ByPreset 会话选了配置名 → 整套展开（含该配置 Key，优先于全局激活预设）。
func TestConvAssembly_ByPreset(t *testing.T) {
	seedConvTestPresets(t)
	hookConvLookup(t, func(convID, wsRoot string) (string, string, string) {
		return "硅基流动", "ddd-model", "ddd"
	})
	p := ResolveProviderParamsForConv("conv-x", "")
	// 会话配置 Key 优先：ddd 的 BAD-KEY（修复前按服务商猜，可能取到 GOOD-KEY）
	if p.APIKey != "BAD-KEY" {
		t.Fatalf("会话配置 Key 未生效：got %q want BAD-KEY", p.APIKey)
	}
	if p.Provider != "硅基流动" {
		t.Fatalf("会话配置服务商未生效：got %q", p.Provider)
	}
	if p.Model != "ddd-model" {
		t.Fatalf("会话配置模型未生效：got %q", p.Model)
	}
	if p.BaseURL != "https://s.example" {
		t.Fatalf("会话配置 BaseURL 未生效：got %q", p.BaseURL)
	}
}

// TestConvAssembly_ByPresetWithGlobal 会话配置与全局激活预设并存：会话配置赢。
func TestConvAssembly_ByPresetWithGlobal(t *testing.T) {
	seedConvTestPresets(t)
	core.Settings.Preset = "激活预设" // 全局激活 deepseek
	hookConvLookup(t, func(convID, wsRoot string) (string, string, string) {
		return "硅基流动", "flash-model", "硅基flash"
	})
	p := ResolveProviderParamsForConv("conv-y", "")
	if p.Provider != "硅基流动" || p.APIKey != "GOOD-KEY" || p.Model != "flash-model" {
		t.Fatalf("会话配置应覆盖全局激活预设：got provider=%s key=%s model=%s", p.Provider, p.APIKey, p.Model)
	}
}

// TestConvAssembly_NoConv 会话未设模型 → 与全局完全一致（激活预设展开）。
func TestConvAssembly_NoConv(t *testing.T) {
	seedConvTestPresets(t)
	hookConvLookup(t, func(convID, wsRoot string) (string, string, string) {
		return "", "", ""
	})
	p := ResolveProviderParamsForConv("conv-z", "")
	if p.Provider != "deepseek" || p.APIKey != "DKEY" || p.Model != "d-model" {
		t.Fatalf("未设会话模型应回落全局激活预设：got provider=%s key=%s model=%s", p.Provider, p.APIKey, p.Model)
	}
}

// TestConvAssembly_ByProviderLegacy 历史会话（只有 provider/model，无配置名）：
// 保持旧链路——按服务商匹配 Key（任一该服务商配置的 Key，非确定但兼容现状；断言非空）。
func TestConvAssembly_ByProviderLegacy(t *testing.T) {
	seedConvTestPresets(t)
	hookConvLookup(t, func(convID, wsRoot string) (string, string, string) {
		return "硅基流动", "ddd-model", "" // 旧数据无 preset
	})
	p := ResolveProviderParamsForConv("conv-legacy", "")
	if p.Provider != "硅基流动" || p.Model != "ddd-model" {
		t.Fatalf("旧链路 provider/model 未生效：got %s / %s", p.Provider, p.Model)
	}
	if p.APIKey == "" {
		t.Fatal("旧链路应兜底到该服务商 Key（非空）")
	}
}

// TestConvAssembly_DeletedPreset 会话配置名已删 → 回落按服务商匹配（不崩溃、Key 非空）。
func TestConvAssembly_DeletedPreset(t *testing.T) {
	seedConvTestPresets(t)
	hookConvLookup(t, func(convID, wsRoot string) (string, string, string) {
		return "硅基流动", "ddd-model", "已删除的配置"
	})
	p := ResolveProviderParamsForConv("conv-del", "")
	if p.Provider != "硅基流动" || p.Model != "ddd-model" {
		t.Fatalf("配置删除后应按服务商回落：got %s / %s", p.Provider, p.Model)
	}
	if p.APIKey == "" {
		t.Fatal("配置删除后 Key 应兜底非空")
	}
}