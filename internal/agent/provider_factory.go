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
// ★ 2026-08-21 配置来源收敛：ai-presets.json 是 AI 配置（模型/参数）的来源——
//
//	settings 只存 preset（当前激活预设名），装配时按 preset 从 ai-presets.json 展开整套配置；
//	settings 顶层字段仅兜底（兼容无预设的旧配置）。
//
// ★ 2026-09-01 Key 回归 AI 配置：API Key 以激活预设携带的 Key 为准（预设 Key 优先），
//   服务商级 Key（models.json）仅当预设未填 Key 时兜底（旧数据迁移兼容）。
// ★ 统一模型：不再拆分 规划/审核 模型，PlanModel/ReviewModel 一律等于执行模型。
func ResolveProviderParams() ProviderParams {
	return ProviderFactoryNow().Apply(resolveProviderBase())
}

// resolveProviderBase 解析装配器之前的基线参数（会话级覆盖在此之后叠加）。
func resolveProviderBase() ProviderParams {

	provider := core.Settings.Provider
	baseURL := core.Settings.BaseURL
	apiKey := core.Settings.APIKey
	model := core.MainModel()
	temperature := core.Temperature()
	thinking := core.Settings.ThinkingMode
	maxTokens := core.Settings.MaxTokens
	context := core.Settings.ContextMaxTokens

	// ① 激活预设展开（settings.preset → ai-presets.json，整套覆盖）
	// ★ 2026-09-01 Key 回归 AI 配置：预设携带 API Key（AiPreset.APIKey），
	//   装配时预设 Key 优先；旧预设里无 Key 的走服务商级兜底。
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
	// ③ Key 来源：预设（AI 配置）已在 ① 中优先应用；服务商级 Key 仅当预设
	//    未填 Key 时兜底（旧数据迁移，models.json 里可能还存有旧服务商 Key）。
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
	return cur

}

// ConfiguredProvider 是否已配好可用 Provider（业务层用，替代 core.Configured 的 AI 检查）。
func ConfiguredProvider() bool {
	p := ResolveProviderParams()
	return p.APIKey != "" && p.BaseURL != "" && p.Model != ""
}

// ─── 会话级模型路由（★ 2026-08-31） ────────────────────────────────
//
// 问题：此前对话面板切换模型 = 写全局 settings（preset/executeModel），
// 所有会话（含已开始的历史对话）的模型一起被改。
// 现在：会话元数据记录 provider+model（ConversationMeta.Provider/Model），
// 切换只写本会话；未设置的会话沿用全局默认配置。
//
// convModelLookup 由 web 层注入（按会话根路由 store），agent 包不直接依赖
// SessionManager，保持解耦；未注入时会话级解析退化为全局默认。
var (
	convModelMu     sync.RWMutex
	convModelLookup func(convID, wsRoot string) (provider, model string)
)

// SetConvModelLookup 注入会话级模型查询钩子（web 层启动时调用）。
func SetConvModelLookup(fn func(convID, wsRoot string) (provider, model string)) {
	convModelMu.Lock()
	convModelLookup = fn
	convModelMu.Unlock()
}

// LookupConvModel 查询会话级模型（provider, model；均空=未设置）。
func LookupConvModel(convID, wsRoot string) (string, string) {
	if convID == "" {
		return "", ""
	}
	convModelMu.RLock()
	fn := convModelLookup
	convModelMu.RUnlock()
	if fn == nil {
		return "", ""
	}
	return fn(convID, wsRoot)
}

// ResolveProviderParamsForConv 解析会话级 Provider 参数：
// 全局默认（ResolveProviderParams）→ 会话选定的 服务商/模型 覆盖 → 装配器再覆盖。
// 会话未设置模型时与 ResolveProviderParams 完全一致（零行为变化）。
func ResolveProviderParamsForConv(convID, wsRoot string) ProviderParams {
	provider, model := LookupConvModel(convID, wsRoot)
	if provider == "" && model == "" {
		return ResolveProviderParams()
	}
	cur := resolveProviderBase()
	if provider != "" {
		cur.Provider = provider
		// 服务商切换 → BaseURL 取该服务商配置；Key 优先取该服务商预设中的 Key（AI 配置），服务商级兜底
		if u := core.GetProviderBaseURL(provider); u != "" {
			cur.BaseURL = u
		}
		if k := core.GetPresetAPIKeyForProvider(provider); k != "" {
			cur.APIKey = k
		} else if k := core.GetProviderAPIKey(provider); k != "" {
			cur.APIKey = k
		}
		cur.ProviderContextMaxTokens = core.GetProviderContextMaxToken(provider)
	}
	if model != "" {
		cur.Model = model
		cur.PlanModel = model
		cur.ReviewModel = model
	}
	return ProviderFactoryNow().Apply(cur)
}
