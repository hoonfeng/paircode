// provider_factory.go — LLM Provider 参数装配点（配置消费插件化）
//
// ★ 2026-08-19：配置消费从 Go 内核搬往插件。
//   装配链：存储基线（core.Settings，存储层） → ProviderFactoryNow().Apply()（插件可覆盖）。
//   业务层（web_server / stub / desktopbridge / llm_analyze）统一经 ResolveProviderParams()
//   获取最终 Provider 参数，不再直接读 core.Settings 的 AI 业务字段——配置决策在插件
//   （agentloop 的 provider 装配器读 ai 组配置返回 overrides），Go 内核只留装配点与兜底基线。

package agent

import (
	"sync"

	"github.com/hoonfeng/paircode/internal/core"
)

// ProviderParams LLM Provider 可装配参数（基线 + 插件覆盖合并后的最终值）。
type ProviderParams struct {
	Provider         string                               // 当前服务商（装配器按服务商取模型级参数）
	BaseURL          string                               // API 端点（不含 /chat/completions）
	APIKey           string                               // 服务商密钥
	Model            string                               // 主执行模型
	Temperature      float64                              // 随机性（-1=不传）
	MaxTokens        int                                  // 最大输出 token（0=不传）
	ThinkingMode     string                               // non-thinking/thinking/thinking_max；空=不下发
	PlanModel        string                               // 规划模型（自主模式分解任务用）
	ReviewModel      string                               // 审核模型（AI 审核用）
	ContextMaxTokens int                                  // ★ 模型级上下文窗口（0=不传）
	ModelParams      map[string]map[string]core.ModelParamEntry // ★ 模型级参数表（服务商 → 模型 → 参数），供装配器按当前模型取
}

// ProviderFactory 装配 LLM Provider 参数的工厂接口（对齐 LoopFactory 单槽位语义）。
type ProviderFactory interface {
	// Apply 接收当前参数快照，返回覆盖合并后的最终参数（JS 装配器返回 overrides）。
	Apply(current ProviderParams) ProviderParams
}

// goProviderFactory 默认工厂：原样返回（无插件装配时用存储基线）。
type goProviderFactory struct{}

func (goProviderFactory) Apply(cur ProviderParams) ProviderParams { return cur }

var (
	providerMu         sync.RWMutex
	providerFactoryVal ProviderFactory = goProviderFactory{}
)

// ReplaceProviderFactory 替换全局 Provider 装配器（单槽位，后注册覆盖先注册），
// 返回还原函数（插件卸载时调用自动还原默认工厂）。
func ReplaceProviderFactory(f ProviderFactory) (restore func()) {
	providerMu.Lock()
	old := providerFactoryVal
	providerFactoryVal = f
	providerMu.Unlock()
	return func() {
		providerMu.Lock()
		providerFactoryVal = old
		providerMu.Unlock()
	}
}

// ProviderFactoryNow 返回当前装配器。
func ProviderFactoryNow() ProviderFactory {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return providerFactoryVal
}

// ResolveProviderParams 解析最终 Provider 参数：存储基线 → 装配器覆盖。
// ★ 业务层统一入口：Go 内核不再直接读 core.Settings 的 AI 业务字段。
func ResolveProviderParams() ProviderParams {
	provider := core.Settings.Provider
	// ★ 2026-08-20 服务商独立 Key/BaseURL 优先（models.json 每服务商保存），
	//   缺省回退全局 settings 字段（兼容旧配置）。
	baseURL := core.Settings.BaseURL
	apiKey := core.Settings.APIKey
	if p := core.GetProviderBaseURL(provider); p != "" {
		baseURL = p
	}
	if k := core.GetProviderAPIKey(provider); k != "" {
		apiKey = k
	}
	cur := ProviderParams{
		Provider:         provider,
		BaseURL:          baseURL,
		APIKey:           apiKey,
		Model:            core.MainModel(),
		Temperature:      core.Temperature(),
		MaxTokens:        core.Settings.MaxTokens,
		ThinkingMode:     core.Settings.ThinkingMode,
		ContextMaxTokens: core.Settings.ContextMaxTokens,
		PlanModel:        core.Settings.PlanModel,
		ReviewModel:      core.Settings.ReviewModel,
		ModelParams:      core.Settings.ModelParams,
	}
	return ProviderFactoryNow().Apply(cur)
}

// ConfiguredProvider 是否已配好可用 Provider（业务层用，替代 core.Configured 的 AI 检查）。
func ConfiguredProvider() bool {
	p := ResolveProviderParams()
	return p.APIKey != "" && p.BaseURL != "" && p.Model != ""
}
