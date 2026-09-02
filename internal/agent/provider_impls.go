// provider_impls.go — LLM Provider 实现级插件槽位（t1 报告 S1 缺口闭环）
//
// ★ 背景（2026-09）：Provider 只有「参数可装配」（ProviderFactory/装配器），
//   实现本身（Chat/Name）不可插拔——新协议（Anthropic 原生、本地推理等）
//   必须改 Go 内核。本文件提供实现级注册面：
//
//	RegisterProviderImpl(name, factory)  Go 侧注册（同名覆盖；返回还原函数）
//	ctx.provider.register(name, impl)     JS 插件注册（jsplugin_providerimpl.go）
//	CreateProvider(params)                业务层统一入口：按服务商名查注册表，
//	                                     未注册回退 OpenAI 兼容实现
//
// 路由语义：params.Provider（装配后服务商名，如 "deepseek"）→ 注册表查找
// （大小写不敏感）→ 命中用插件实现；未命中回退 OpenAIProvider（行为不变）。
// 插件卸载自动还原（注册时登记 cleanup，还原后回退 OpenAI 实现）。

package agent

import (
	"strings"
	"sync"
)

// ProviderImplFactory Provider 实现工厂：由最终参数构造 Provider 实例。
type ProviderImplFactory func(params ProviderParams) Provider

var (
	providerImplMu sync.RWMutex
	providerImpls  = map[string]ProviderImplFactory{}
)

// RegisterProviderImpl 注册 Provider 实现工厂（服务商名 → 工厂；同名覆盖，
// 后注册覆盖先注册）。返回还原函数（插件卸载时调用自动回退默认/前一实现）。
func RegisterProviderImpl(name string, factory ProviderImplFactory) (restore func()) {
	providerImplMu.Lock()
	old, existed := providerImpls[strings.ToLower(strings.TrimSpace(name))]
	providerImpls[strings.ToLower(strings.TrimSpace(name))] = factory
	providerImplMu.Unlock()
	if !existed {
		old = nil
	}
	return func() {
		providerImplMu.Lock()
		if existed {
			providerImpls[strings.ToLower(strings.TrimSpace(name))] = old
		} else {
			delete(providerImpls, strings.ToLower(strings.TrimSpace(name)))
		}
		providerImplMu.Unlock()
	}
}

// LookupProviderImpl 查询服务商名对应的实现工厂（未注册返回 nil）。
func LookupProviderImpl(name string) ProviderImplFactory {
	providerImplMu.RLock()
	defer providerImplMu.RUnlock()
	return providerImpls[strings.ToLower(strings.TrimSpace(name))]
}

// ProviderImplNames 列出已注册的实现级 Provider 服务商名（诊断/插件面板用）。
func ProviderImplNames() []string {
	providerImplMu.RLock()
	defer providerImplMu.RUnlock()
	names := make([]string, 0, len(providerImpls))
	for n := range providerImpls {
		names = append(names, n)
	}
	return names
}

// CreateProvider 按最终参数创建 Provider 实现（业务层统一入口）：
//  1. 服务商名在实现注册表有插件实现 → 用之（新协议无需改 Go 内核）；
//  2. 否则按协议路由内置实现：anthropic-messages → AnthropicProvider；
//     openai-completions / openai-responses（responses 适配器见 provider_responses.go）→
//     OpenAI 兼容实现（行为与历史一致，仅 URL 按协议拼接）。
//
// ★ 业务层（buildWebProvider / buildDesktopProvider / 子 Agent / 审核/规划
//
//	Provider / LLM 分析）统一经此创建，不再直接 new OpenAIProvider——
//	插件注册的实现对所有消费方生效。
func CreateProvider(params ProviderParams) Provider {
	if f := LookupProviderImpl(params.Provider); f != nil {
		return f(params)
	}
	switch normalizeProtocol(params.Protocol) {
	case ProtocolAnthropicMessages:
		return &AnthropicProvider{
			BaseURL:      params.BaseURL,
			Protocol:     params.Protocol,
			APIKey:       params.APIKey,
			Model:        params.Model,
			Temperature:  params.Temperature,
			MaxTokens:    params.MaxTokens,
			ThinkingMode: params.ThinkingMode,
			Multimodal:   params.Multimodal,
		}
	case ProtocolOpenAIResponses:
		return &ResponsesProvider{
			BaseURL:      params.BaseURL,
			Protocol:     params.Protocol,
			APIKey:       params.APIKey,
			Model:        params.Model,
			Temperature:  params.Temperature,
			MaxTokens:    params.MaxTokens,
			ThinkingMode: params.ThinkingMode,
			Multimodal:   params.Multimodal,
		}
	default:
		return &OpenAIProvider{
			BaseURL:      params.BaseURL,
			Protocol:     params.Protocol,
			APIKey:       params.APIKey,
			Model:        params.Model,
			Temperature:  params.Temperature,
			MaxTokens:    params.MaxTokens,
			ThinkingMode: params.ThinkingMode,
			Multimodal:   params.Multimodal,
		}
	}
}

// normalizeProtocol 协议归一化（空 → 默认 OpenAI 兼容）。
func normalizeProtocol(p string) string {
	if p == "" {
		return DefaultProtocol
	}
	return p
}
