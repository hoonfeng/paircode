// provider_factory.go — LLM Provider 参数装配点（配置消费插件化）
//
// ★ 2026-08-19：配置消费从 Go 内核搬往插件。
//   装配链：存储基线（core.Settings，存储层） → ProviderFactory().Apply()（插件可覆盖）。
//   业务层（web_server / stub / desktopbridge / llm_analyze）统一经 ResolveProviderParams()
//   获取最终 Provider 参数，不再直接读 core.Settings 的 AI 业务字段。
//
// ★ 2026-09-03 决策全量迁插件（本轮）：Go 只留「机制 + 数据面」——
//   · 裸基线 resolveProviderBase：只注入装配上下文（零决策；★ 2026-09-20 起连接字段
//     也不再读，见下）；
//   · 装配上下文注入：Preset（当前生效配置名：会话级 > 全局激活）与 Conv*（会话选定值）
//     透传给装配器（JS 经 ctx.aiPresets / ctx.models 查数据面表）；
//   · 决策（配置整套展开、服务商数据兜底、Key 选择、统一模型同步、参数级覆盖、
//     上下文窗口层级）全部在插件装配器（agentloop 的 provider 装配器）内实现；
//   · 会话三元组查询（LookupConvModel）由 web 层注入钩子提供（数据面，agent 包不依赖 SessionManager）。
//   无插件装载时装配器为原样工厂（goProviderFactory），返回裸基线（回退行为，非决策）。
//
// ★ 2026-09-20 生成参数零直读；★ 2026-09-25 唯一来源收敛为「服务商配置」：
//   温度 / 思考档位 / 最大输出 / 上下文窗口不再由核心读取（此前读 core.Temperature() 与
//   core.Settings.{MaxTokens,ThinkingMode,ContextMaxTokens}，与「配置项一律插件 ctx.registerSettings
//   注册」的设计纪律不符）。现取值来源全在插件侧：
//   · 服务商级 / 模型级 → models.json（经 ctx.models）——**生成参数唯一来源**（模型级 > 服务商级）；
//   · 配置级（AI 配置预设）→ ai-presets.json（经 ctx.aiPresets）：连接信息；生成参数快照亦取激活配置；
//   · 机制兜底 → 插件内常量 GEN_DEFAULTS（非配置面，纯运行保障）。
//   设置面板「生成参数」注册段（settings.json → pluginSettings.generation）已于 2026-09-25 整体移除
//   （它与服务商配置字段重复且位于装配链最低优先级）。核心只消费装配结果（ProviderParams.ContextMaxTokens 等）。
//
// ★ 2026-09-20 连接字段零直读（同批清理）：服务商/地址/Key/模型此前从 settings 顶层
//   兜底读取（resolveProviderBase 的 Provider/BaseURL/APIKey/MainModel + 装配器 ③ 段的
//   s.provider/s.executeModel）。settings 顶层旧值已由 core 一次性迁入 ai-presets.json
//   的一条 AI 配置并清空（core/settings_connection.go）——现基线只传「装配上下文」
//   （Preset + 会话三元组），连接信息一律由装配器按激活配置经 ctx.aiPresets/ctx.models 展开。

package agent

import (
	"fmt"
	"log"
	"sync"

	"github.com/hoonfeng/paircode/internal/core"
)

// ProviderParams LLM Provider 可装配参数（基线 + 插件覆盖合并后的最终值）。
type ProviderParams struct {
	Provider                 string                                     // 当前服务商（装配器按服务商取模型级参数）
	Preset                   string                                     // ★ 2026-09-03 当前生效配置名（会话级 > 全局激活；装配器按名整套展开）
	ConvProvider             string                                     // ★ 2026-09-03 会话选定服务商（空=会话未设置；装配器决策覆盖用）
	ConvModel                string                                     // ★ 2026-09-03 会话选定模型（空=会话未设置）
	ConvPreset               string                                     // ★ 2026-09-03 会话选定配置名（空=会话未选配置）
	BaseURL                  string                                     // ★ API 基础地址（不含协议路径；完整端点由 ResolveEndpointURL 按 Protocol 拼接）
	Protocol                 string                                     // ★ 2026-09-02 LLM 协议（openai-completions/openai-responses/anthropic-messages；空=默认 openai-completions）
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

// logResolvedParams 装配结果诊断日志（用户可见：确认最终传入 agentloop/Provider 的配置）。
// tag=调用场景（global=全局装配 / conv=会话级装配）；convID 非空时附带会话级上下文。
func logResolvedParams(tag, convID string, p ProviderParams) {
	key := ""
	if p.APIKey != "" {
		key = p.APIKey
		if len(key) > 8 {
			key = key[:4] + "…" + key[len(key)-4:]
		}
	}
	extra := ""
	if convID != "" {
		extra = fmt.Sprintf(" conv=%s (会话选定 provider=%s model=%s preset=%s)", convID, p.ConvProvider, p.ConvModel, p.ConvPreset)
	}
	// ★ 2026-09-19：一并打印生成参数（温度/最大输出/上下文窗口）——便于确认取值来源
	//   （一律以 models.json 服务商配置为准，模型级 > 服务商级）。
	// ★ 2026-09-20 补打 thinkingMode；★ 2026-09-25 生成参数唯一来源 = 服务商配置
	//   （设置面板「生成参数」段已移除），本行输出即可判定四项生成参数的实际来源。
	log.Printf("[provider] %s 装配结果: provider=%s model=%s preset=%s baseURL=%s protocol=%s apiKey=%s | temperature=%.2f thinkingMode=%s maxTokens=%d contextMaxTokens=%d%s",
		tag, p.Provider, p.Model, p.Preset, p.BaseURL, p.Protocol, key, p.Temperature, p.ThinkingMode, p.MaxTokens, p.ContextMaxTokens, extra)
}

// ResolveProviderParams 解析最终 Provider 参数：存储基线 → 装配器覆盖。
// ★ 业务层统一入口：Go 内核不再直接读 core.Settings 的 AI 业务字段。
// ★ 2026-08-21 配置来源收敛：ai-presets.json 是 AI 配置（模型/参数）的来源——
//
//	settings 只存 preset（当前激活预设名），装配时按 preset 从 ai-presets.json 展开整套配置；
//	settings 顶层字段仅兜底（兼容无预设的旧配置）。
//
// ★ 2026-09-01 Key 回归 AI 配置：API Key 以激活预设携带的 Key 为准（预设 Key 优先），
//
//	服务商级 Key（models.json）仅当预设未填 Key 时兜底（旧数据迁移兼容）。
//
// ★ 统一模型：不再拆分 规划/审核 模型，PlanModel/ReviewModel 一律跟随执行模型。
// ★ 2026-09-03 决策迁插件：上述展开规则全部由装配器（agentloop）实现；Go 只传
//
//	裸基线 + Preset（全局激活配置名），不重复任何决策。
func ResolveProviderParams() ProviderParams {
	p := ProviderFactoryNow().Apply(resolveProviderBase())
	logResolvedParams("global", "", p)
	return p
}

// resolveProviderBase 解析装配器之前的裸基线（零决策）：
// 只注入装配上下文（Preset=全局激活配置名；Conv* 由 ForConv 注入）。
// 服务商默认数据（BaseURL/Key/协议/上下文）与配置展开由装配器经 ctx.models/ctx.aiPresets
// 决策（★ 2026-09-03 决策迁插件——Go 不再预填任何 AI 业务派生值）。
//
// ★ 2026-09-20 连接字段零直读：此前基线还从 core.Settings 抓 Provider/BaseURL/APIKey/
// Model（旧连接字段兜底），装配器再兜一层 s.provider/s.executeModel——同一份连接信息
// 在 settings 顶层与 ai-presets.json 两处存储、易漂移。现 settings 顶层旧值已一次性迁入
// AI 配置并清空（core/settings_connection.go），基线只留装配上下文；服务商/地址/Key/模型
// 由装配器按激活配置展开（无激活配置 → 未配置，运行期由 ConfiguredProvider 报缺失）。
func resolveProviderBase() ProviderParams {
	return ProviderParams{
		// ★ 生成参数不再由核心直读（见文件头）：基线一律留「未配置」，
		//   由装配器按服务商配置（models.json，模型级 > 服务商级）决策；
		//   无任何配置面时由插件机制常量 GEN_DEFAULTS 兜底（非配置面）。
		//   Temperature=-1 是「不下发温度」的机制语义（Provider 不传 temperature 字段）。
		Temperature: -1,
		// 装配上下文：全局激活配置名（装配器据此经 ctx.aiPresets 整套展开）。
		// ★ 服务商级默认上下文窗口（ProviderContextMaxTokens）不再预查——装配器按
		//   「最终服务商」经 ctx.models 取（基线预查会按旧服务商算，可能不是最终值）。
		Preset: core.Settings.Preset,
	}
}

// ConfiguredProvider 是否已配好可用 Provider（业务层用，替代 core.Configured 的 AI 检查）。
func ConfiguredProvider() bool {
	p := ResolveProviderParams()
	return p.APIKey != "" && p.BaseURL != "" && p.Model != ""
}

// ContextWindow 返回生效的上下文窗口（token）——一律以服务商配置（models.json）为准：
// 装配器已按「模型级（models.json modelParams）> 服务商级（models.json）」算出 ContextMaxTokens；
// ★ 2026-09-25：生成参数唯一来源 = 服务商配置；原「全局（插件注册 generation 段）」层已随
// 设置面板该页移除。故本函数只在插件未装载/全部未配置时回退机制兜底常量（运行保障，非配置面）。
//
// ★ 2026-09-19 缺陷修复：此前 Loop 装配 / 历史精简 / 会话交接 / 精简策略都直接读
// core.Settings.ContextMaxTokens，models.json 的服务商级上下文窗口不生效（装配结果无人消费）。
// ★ 2026-09-20：核心不再直读任何生成参数——移除对 core.Settings.ContextMaxTokens 的兜底
// （旧字段已迁入服务商配置并清空），改回机制常量 core.DefaultContextWindow。
func ContextWindow(p ProviderParams) int {
	if p.ContextMaxTokens > 0 {
		return p.ContextMaxTokens
	}
	return core.DefaultContextWindow
}

// ConfiguredProviderForConv 会话感知的 Provider 就绪检查（★ 2026-09-04 同步段校验修复）：
// 同步段（handleChatSend 等）此前只用全局装配判定（ConfiguredProvider）——用户已在
// 会话里选定 服务商/模型/配置（会话三元组非空）时，全局激活配置可能尚未展开完整
// （如激活配置未指定模型），导致误报「未配置 API key」。本函数按会话装配判定，
// 会话未设置时退化为全局判定（与 ConfiguredProvider 完全一致）。
// 返回 (是否就绪, 缺失项描述)；描述区分「API Key / API 地址 / 模型」三者，供前端准确提示。
func ConfiguredProviderForConv(convID, wsRoot string) (bool, string) {
	provider, model, preset := LookupConvModel(convID, wsRoot)
	var p ProviderParams
	if provider == "" && model == "" && preset == "" {
		p = ResolveProviderParams()
	} else {
		p = ResolveProviderParamsForConv(convID, wsRoot)
	}
	if p.APIKey == "" {
		return false, "API Key 为空"
	}
	if p.BaseURL == "" {
		return false, "API 地址为空"
	}
	if p.Model == "" {
		return false, "模型为空"
	}
	return true, ""
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
// ★ 2026-09-03 三元组：provider/model/preset（配置名——装配按配置整套展开）。
var (
	convModelMu     sync.RWMutex
	convModelLookup func(convID, wsRoot string) (provider, model, preset string)
)

// SetConvModelLookup 注入会话级模型查询钩子（web 层启动时调用）。
func SetConvModelLookup(fn func(convID, wsRoot string) (provider, model, preset string)) {
	convModelMu.Lock()
	convModelLookup = fn
	convModelMu.Unlock()
}

// LookupConvModel 查询会话级模型（provider, model, preset；全空=未设置）。
func LookupConvModel(convID, wsRoot string) (string, string, string) {
	if convID == "" {
		return "", "", ""
	}
	convModelMu.RLock()
	fn := convModelLookup
	convModelMu.RUnlock()
	if fn == nil {
		return "", "", ""
	}
	return fn(convID, wsRoot)
}

// ResolveProviderParamsForConv 解析会话级 Provider 参数（★ 2026-09-03 机制收敛）：
//
//	Go 只做三件事：① 查会话三元组（LookupConvModel，web 层注入的数据面钩子）；
//	② 注入装配上下文（Preset=会话配置>全局激活；Conv* 透传会话选定值）；③ 委托装配器决策。
//	配置整套展开（含 Key/协议/参数）与旧服务商匹配链路全部在插件装配器内实现。
//
// 会话未设置模型时与 ResolveProviderParams 完全一致（零行为变化）。
func ResolveProviderParamsForConv(convID, wsRoot string) ProviderParams {
	provider, model, preset := LookupConvModel(convID, wsRoot)
	if provider == "" && model == "" && preset == "" {
		return ResolveProviderParams()
	}
	cur := resolveProviderBase()
	// 装配上下文注入：会话选定值（装配器决策的依据，见 agentloop provider 装配器）
	cur.ConvProvider, cur.ConvModel, cur.ConvPreset = provider, model, preset
	if provider != "" {
		cur.Provider = provider
	}
	if model != "" {
		cur.Model = model
	}
	if preset != "" {
		cur.Preset = preset
	}
	p := ProviderFactoryNow().Apply(cur)
	logResolvedParams("conv", convID, p)
	return p
}
