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
	Provider                 string                                     // 当前服务商（装配器按服务商取模型级参数）
	BaseURL                  string                                     // ★ API 完整请求端点（含 /chat/completions），直接作为请求 URL，不再拼接
	APIKey                   string                                     // 服务商密钥
	Model                    string                                     // 主执行模型
	Temperature              float64                                    // 随机性（-1=不传）
	MaxTokens                int                                        // 最大输出 token（0=不传）
	ThinkingMode             string                                     // non-thinking/thinking/thinking_max；空=不下发
	Multimodal               bool                                       // ★ 2026-08-21 多模态：当前模型支持图片输入（装配器按模型级参数标记）
	PlanModel                string                                     // 规划模型（自主模式分解任务用）
	ReviewModel              string                                     // 审核模型（AI 审核用）
	ContextMaxTokens         int                                        // ★ 模型级上下文窗口（0=不传）
	ProviderContextMaxTokens int                                        // ★ 服务商级默认上下文窗口（models.json 每服务商配置；0=未配置，供装配器兜底）
	ModelParams              map[string]map[string]core.ModelParamEntry // ★ 模型级参数表（服务商 → 模型 → 参数），供装配器按当前模型取
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

// ResolveProviderParams 解析最终 Provider 参数：激活预设 → 存储基线 → 装配器覆盖。
// ★ 业务层统一入口：Go 内核不再直接读 core.Settings 的 AI 业务字段。
// ★ 2026-08-21 配置来源收敛：ai-presets.json 是 AI 配置（key/模型/参数）的唯一来源——
//
//	settings 只存 preset（当前激活预设名），装配时按 preset 从 ai-presets.json 展开整套配置；
//	settings 顶层字段仅兜底（兼容无预设的旧配置），models.json 服务商 key/baseURL 再兜底。
//
// ★ 统一模型：不再拆分 规划/审核 模型，PlanModel/ReviewModel 一律等于执行模型。
func ResolveProviderParams() ProviderParams {
	provider := core.Settings.Provider
	baseURL := core.Settings.BaseURL
	apiKey := core.Settings.APIKey
	model := core.MainModel()
	temperature := core.Temperature()
	thinking := core.Settings.ThinkingMode
	maxTokens := core.Settings.MaxTokens
	context := core.Settings.ContextMaxTokens

	// ① 激活预设展开（settings.preset → ai-presets.json，整套覆盖）
	if name := core.Settings.Preset; name != "" {
		if p := core.GetPreset(name); p.Provider != "" || p.ExecuteModel != "" {
			if p.Provider != "" {
				provider = p.Provider
			}
			if p.BaseURL != "" {
				baseURL = p.BaseURL
			}
			if p.APIKey != "" {
				apiKey = p.APIKey
			}
			if p.ExecuteModel != "" {
				model = p.ExecuteModel
			}
			if p.Temperature != "" {
				temperature = core.ParseTempOr(p.Temperature, -1)
			}
			if p.ThinkingMode != "" {
				thinking = p.ThinkingMode
			}
			if p.MaxTokens > 0 {
				maxTokens = p.MaxTokens
			}
			if p.ContextMaxTokens > 0 {
				context = p.ContextMaxTokens
			}
		}
	}
	// ② settings 顶层兜底（兼容旧配置）
	// ③ models.json 服务商 key/baseURL 兜底（无 preset 且 settings 顶层为空时）
	if baseURL == "" {
		if p := core.GetProviderBaseURL(provider); p != "" {
			baseURL = p
		}
	}
	if apiKey == "" {
		if k := core.GetProviderAPIKey(provider); k != "" {
			apiKey = k
		}
	}
	// ★ 统一模型：规划/审核 一律用执行模型（不拆分）
	planModel := model
	reviewModel := model
	cur := ProviderParams{
		Provider:                 provider,
		BaseURL:                  baseURL,
		APIKey:                   apiKey,
		Model:                    model,
		Temperature:              temperature,
		MaxTokens:                maxTokens,
		ThinkingMode:             thinking,
		ContextMaxTokens:         context,
		ProviderContextMaxTokens: core.GetProviderContextMaxToken(provider), // ★ 服务商级默认上下文（模型级未配置时兜底）
		PlanModel:                planModel,
		ReviewModel:              reviewModel,
		ModelParams:              core.Settings.ModelParams,
	}
	return ProviderFactoryNow().Apply(cur)
}

// ConfiguredProvider 是否已配好可用 Provider（业务层用，替代 core.Configured 的 AI 检查）。
func ConfiguredProvider() bool {
	p := ResolveProviderParams()
	return p.APIKey != "" && p.BaseURL != "" && p.Model != ""
}
