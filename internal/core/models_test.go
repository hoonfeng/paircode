package core

import "testing"

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
