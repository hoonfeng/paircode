package agent

import (
	"testing"

	"github.com/hoonfeng/paircode/goja"
)

// TestApplyOverridesProvider 装配器决策出的最终服务商必须回写到 ProviderParams。
//
// ★ 2026-09-20 缺陷修复：`provider` 键此前被 applyOverrides 丢弃——JS 装配器算出的
// 服务商（激活配置展开 / 会话选定 convProvider）只用于其内部查 ctx.models，
// 而 ProviderParams.Provider 始终停留在基线值。旧基线（core.Settings.Provider，
// 从 settings 顶层读取）有兜底值所以未暴露；连接字段零直读后表现为 provider 恒空
// （服务商实现路由 LookupProviderImpl 与装配日志失真）。
func TestApplyOverridesProvider(t *testing.T) {
	vm := goja.New()
	b := &jsProviderFactoryBridge{vm: vm}

	obj := vm.ToValue(map[string]any{
		"provider": "prov-x",
		"baseURL":  "https://x.example/v1",
		"apiKey":   "K1",
		"model":    "m1",
		"protocol": "anthropic-messages",
	}).ToObject(vm)
	out := b.applyOverrides(ProviderParams{Preset: "激活配置"}, obj)
	if out.Provider != "prov-x" {
		t.Errorf("服务商覆盖必须回写，得到 %q", out.Provider)
	}
	if out.BaseURL != "https://x.example/v1" || out.APIKey != "K1" || out.Model != "m1" || out.Protocol != "anthropic-messages" {
		t.Errorf("其余覆盖字段应一并生效，得到 %+v", out)
	}
	if out.Preset != "激活配置" {
		t.Errorf("装配上下文（Preset）应保留，得到 %q", out.Preset)
	}

	// 空值/未定义不覆盖（机制语义：装配器没给值就维持基线）
	for _, over := range []map[string]any{{"provider": ""}, {"provider": nil}, {}} {
		o := vm.ToValue(over).ToObject(vm)
		got := b.applyOverrides(ProviderParams{Provider: "base-prov"}, o)
		if got.Provider != "base-prov" {
			t.Errorf("覆盖 %v 不得改动服务商，得到 %q", over, got.Provider)
		}
	}
}

// TestApplyOverridesGenerationParams 生成参数覆盖语义（≥0 才覆盖；0/缺失不覆盖）——
// 与 provider 覆盖同批验证，防止桥接层漏字段。
func TestApplyOverridesGenerationParams(t *testing.T) {
	vm := goja.New()
	b := &jsProviderFactoryBridge{vm: vm}

	obj := vm.ToValue(map[string]any{
		"temperature":      0.55,
		"thinkingMode":     "xhigh",
		"maxTokens":        12345,
		"contextMaxTokens": 67890,
		"multimodal":       true,
	}).ToObject(vm)
	out := b.applyOverrides(ProviderParams{Temperature: -1}, obj)
	if out.Temperature != 0.55 || out.ThinkingMode != "xhigh" || out.MaxTokens != 12345 ||
		out.ContextMaxTokens != 67890 || !out.Multimodal {
		t.Errorf("生成参数覆盖应全部生效，得到 %+v", out)
	}

	// 负温度 / 零值不覆盖（机制语义：未配置）
	obj2 := vm.ToValue(map[string]any{"temperature": -1.0, "maxTokens": 0, "contextMaxTokens": 0}).ToObject(vm)
	out2 := b.applyOverrides(ProviderParams{Temperature: 0.3, MaxTokens: 111, ContextMaxTokens: 222}, obj2)
	if out2.Temperature != 0.3 || out2.MaxTokens != 111 || out2.ContextMaxTokens != 222 {
		t.Errorf("非正值不得覆盖生成参数，得到 %+v", out2)
	}
}
