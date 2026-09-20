package core

import (
	"log"
	"strings"
)

// settings_generation.go — 生成参数的「插件注册配置」域（★ 2026-09-20）。
//
// ★ 设计纪律（docs/pluginization-gap-analysis.md、changelog「配置项注册纪律」）：
//
//	「配置项一律由插件经 ctx.registerSettings 注册，值存 settings.json 的
//	 pluginSettings.<注册 key>；不再新增全局设置顶层字段，避免设置面膨胀」。
//
// 生成参数（温度 / 思考档位 / 最大输出 / 上下文窗口）此前存在 AppSettings 顶层
// （temperature/thinkingMode/maxTokens/contextMaxTokens + modelParams），且 Go 内核对它们
// 有运行期直读（resolveProviderBase / ContextWindow）——与上述纪律不符。现收敛为：
//
//   - 全局默认：插件 agentloop 注册的 GenerationSettingsKey 段（值存 pluginSettings.generation）；
//   - 服务商级 / 模型级：models.json（插件数据面，经 ctx.models 读写）；
//   - 配置级（AI 配置预设）：ai-presets.json（插件数据面，经 ctx.aiPresets 读写）。
//
// Go 内核**不再直读任何生成参数**：取值全部由装配器（agentloop 的 provider 装配器）
// 经 ctx.getSettings(GenerationSettingsKey) / ctx.models / ctx.aiPresets 决策，
// 内核只消费装配结果（agent.ResolveProviderParams → ProviderParams）。
//
// 本文件只保留两件事：① 插件未装载/未配置时的**机制兜底常量**（不是配置面）；
// ② settings.json 旧顶层字段 → 插件注册域的一次性迁移（迁移后旧字段清空、不再写回）。

// GenerationSettingsKey 生成参数配置段的注册 key——由插件 agentloop
// ctx.registerSettings({key: 'generation', title: '生成参数', fields: [...]}) 注册。
// ★ 插件侧改注册 key 时需同步本常量（core 与插件共享同一命名空间名）。
const GenerationSettingsKey = "generation"

// DefaultContextWindow 上下文窗口的机制兜底值（token）。
//
// ★ 这是「运行保障」而非配置面：正常路径下窗口由装配器按
// 模型级（models.json modelParams）> 服务商级（models.json）> 配置级（ai-presets.json）
// > 全局（插件注册 generation 段）算出正数；仅当插件未装载/全部未配置时，
// 本常量保证 Loop 精简与压缩有可用窗口（否则窗口 0 会导致不压缩/无限上下文）。
// 配置默认值请改插件 registerSettings 的 default（agentloop 的 GEN_DEFAULTS）。
const DefaultContextWindow = 64000

// GenerationDefaultValue 读插件注册的生成参数值（未注册/未设置 → 返回 nil）。
// 仅用于「保存 AI 配置」等需要抓当前全局默认快照的场景（server/handler）。
func GenerationDefaultValue(name string) any {
	v := Settings.PluginSettingValue(GenerationSettingsKey)[name]
	return v
}

// GenerationTemperature 读插件注册域的温度（字符串，"0"~"2.0"；空=未配置）。
func GenerationTemperature() string {
	s, _ := GenerationDefaultValue("temperature").(string)
	return strings.TrimSpace(s)
}

// GenerationThinkingMode 读插件注册域的思考档位（空=未配置）。
func GenerationThinkingMode() string {
	s, _ := GenerationDefaultValue("thinkingMode").(string)
	return strings.TrimSpace(s)
}

// GenerationMaxTokens 读插件注册域的最大输出 token（0=未配置）。
// 兼容 JSON 反序列化后的 float64 与已存的字符串形态。
func GenerationMaxTokens() int {
	return anyToInt(GenerationDefaultValue("maxTokens"))
}

// GenerationContextMaxTokens 读插件注册域的上下文窗口（0=未配置）。
func GenerationContextMaxTokens() int {
	return anyToInt(GenerationDefaultValue("contextMaxTokens"))
}

// anyToInt 宽松取整（number / 数字字符串 / 空 → 0）。
func anyToInt(v any) int {
	switch n := v.(type) {
	case float64:
		if n > 0 {
			return int(n)
		}
	case int:
		if n > 0 {
			return n
		}
	case int64:
		if n > 0 {
			return int(n)
		}
	case string:
		if i := ParseTempInt(n); i > 0 {
			return i
		}
	}
	return 0
}

// ParseTempInt 解析正整数（失败/非正 → 0）。
func ParseTempInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
		if n > 1<<40 {
			return 0
		}
	}
	return n
}

// MigrateGenerationSettingsFromLegacy 把 settings.json 顶层的生成参数旧字段
// （temperature/thinkingMode/maxTokens/contextMaxTokens/modelParams）一次性迁入
// 插件注册域，并清空旧字段（此后 Go 内核零直读；Save 时 omitempty 不再写回）。
//
// 幂等：写入只在目标键未配置时发生（不覆盖插件注册域已有值）；旧字段清空后重跑直接返回。
// ★ 调用时机：core.Load 内、MigrateParamSettingsToModels 之后（模型级参数先搬进 models.json）。
func MigrateGenerationSettingsFromLegacy() {
	if Settings.Temperature == "" && Settings.ThinkingMode == "" &&
		Settings.MaxTokens == 0 && Settings.ContextMaxTokens == 0 && len(Settings.ModelParams) == 0 {
		return // 无遗留值（已迁移或本就未配置）
	}
	if Settings.PluginSettings == nil {
		Settings.PluginSettings = map[string]map[string]any{}
	}
	gen := Settings.PluginSettings[GenerationSettingsKey]
	if gen == nil {
		gen = map[string]any{}
	}
	// 只在目标键未配置时写入：插件注册域（用户在新「生成参数」面板设置的值）优先
	put := func(k string, v any) {
		if _, exists := gen[k]; exists {
			return
		}
		gen[k] = v
	}
	if s := strings.TrimSpace(Settings.Temperature); s != "" {
		put("temperature", s)
	}
	if s := strings.TrimSpace(Settings.ThinkingMode); s != "" {
		put("thinkingMode", s)
	}
	if Settings.MaxTokens > 0 {
		put("maxTokens", Settings.MaxTokens)
	}
	if Settings.ContextMaxTokens > 0 {
		put("contextMaxTokens", Settings.ContextMaxTokens)
	}
	Settings.PluginSettings[GenerationSettingsKey] = gen

	// 清空旧字段：模型级参数已由 MigrateParamSettingsToModels 搬进 models.json
	// （见 models.go），全局默认已进插件注册域 → 旧字段不再有任何运行期消费者。
	Settings.Temperature = ""
	Settings.ThinkingMode = ""
	Settings.MaxTokens = 0
	Settings.ContextMaxTokens = 0
	Settings.ModelParams = nil
	Save()
	log.Printf("[config] 已把 settings.json 的生成参数迁入插件注册域 pluginSettings.%s（此后全局默认由插件注册配置为准）",
		GenerationSettingsKey)
}
