package core

import (
	"log"
	"strings"
)

// settings_generation.go — 生成参数「旧来源 → 服务商配置」的一次性迁移（★ 2026-09-25）。
//
// 唯一来源纪律：生成参数（温度 / 思考档位 / 最大输出 / 上下文窗口）的唯一来源 = **服务商配置**
// （config/models.json：模型级 ProviderEntry.ModelParams[模型] > 服务商级 ProviderEntry 字段）。
//
// 历史沿革（本文件为何只剩迁移与兜底常量）：
//   - 起初四个参数存 AppSettings 顶层 + settings.modelParams，且 Go 内核有运行期直读
//     （resolveProviderBase / ContextWindow）；
//   - 2026-09-19 收敛到 models.json（服务商级 / 模型级）——见 MigrateParamSettingsToModels；
//   - 2026-09-20 又加了「全局默认」层：插件 agentloop 注册 generation 段（值存
//     pluginSettings.generation），内核零直读、取值全在装配器；
//   - 2026-09-25 该「全局默认」层**整体移除**：它与服务商配置字段完全重复，却处于装配链最低
//     优先级（服务商级一配即被覆盖）——用户在设置面板改温度/输出/窗口「改了不生效」；
//     同时服务商级缺「思考档位」字段，反而迫使全局段成为思考档位的唯一全局入口。
//     现补服务商级思考档位（ProviderEntry.ThinkingMode）后删除该注册段：生成参数只剩
//     「服务商配置」一个来源（模型级 > 服务商级），设置面不再有重复入口。
//
// 本文件职责：① 机制兜底常量（未配置时的运行保障，不是配置面）；
//            ② 旧来源（settings 顶层旧字段 + pluginSettings.generation）→ 服务商级字段的迁移。

// LegacyGenerationSettingsKey 设置面板「生成参数」段曾用的注册 key（★ 2026-09-25 该注册段已移除）。
// 仅用于读取历史值并迁入服务商配置；core 与插件均不再注册此段。
const LegacyGenerationSettingsKey = "generation"

// DefaultContextWindow 上下文窗口的机制兜底值（token）。
//
// ★ 这是「运行保障」而非配置面：正常路径下窗口由装配器按
// 模型级（models.json modelParams）> 服务商级（models.json）> 配置级（ai-presets.json）
// 算出正数；仅当插件未装载/全部未配置时，本常量保证 Loop 精简与压缩有可用窗口
// （否则窗口 0 会导致不压缩/无限上下文）。
// 要改默认值：服务商面板的「上下文大小」字段，或插件 agentloop 的 GEN_DEFAULTS（机制兜底）。
const DefaultContextWindow = 64000

// anyToInt 宽松取整（number / 数字字符串 / 空 → 0）；迁移读取旧值时用。
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

// MigrateGenerationToProvider 把生成参数的旧来源一次性迁进服务商配置，并清空旧来源。
//
// 旧来源（同源、先后出现过，值可能都存在）：
//   - settings.json 顶层 temperature/thinkingMode/maxTokens/contextMaxTokens（★ 2026-09-20 前的旧字段）；
//   - pluginSettings.generation（设置面板「生成参数」段的注册值，★ 2026-09-25 该段已移除）；
//   - settings.modelParams（模型级旧字段；正常已由 MigrateParamSettingsToModels 搬走，
//     这里只做「确保清空」——它已无任何运行期消费者）。
//
// 目标：激活配置（settings.preset）所指向服务商的**服务商级**字段
// （models.json：Temperature/ThinkingMode/MaxTokens/ContextMaxTokens），**只补空、不覆盖已有值**
// ——服务商配置已是唯一来源，用户后来在服务商面板里设的值优先于这些历史默认。
//
// 幂等：旧来源清空后重跑直接返回。无法定位目标服务商（无激活配置 / 服务商不在 models.json /
// 落盘失败）时**保留旧值**并打日志（下次启动重试），避免「值被清掉又没落到任何地方」。
// ★ 调用时机：core.Load 内、MigrateParamSettingsToModels 之后（模型级参数先搬进 models.json）。
func MigrateGenerationToProvider() {
	legacy := map[string]any{}
	if s := strings.TrimSpace(Settings.Temperature); s != "" {
		legacy["temperature"] = s
	}
	if s := strings.TrimSpace(Settings.ThinkingMode); s != "" {
		legacy["thinkingMode"] = s
	}
	if Settings.MaxTokens > 0 {
		legacy["maxTokens"] = Settings.MaxTokens
	}
	if Settings.ContextMaxTokens > 0 {
		legacy["contextMaxTokens"] = Settings.ContextMaxTokens
	}
	// 注册段旧值补齐（顶层旧字段更早、值等价——已有则不再取注册段，避免覆盖语义模糊）
	if gen := Settings.PluginSettingValue(LegacyGenerationSettingsKey); len(gen) > 0 {
		for _, k := range []string{"temperature", "thinkingMode", "maxTokens", "contextMaxTokens"} {
			if _, exists := legacy[k]; exists {
				continue
			}
			if v, ok := gen[k]; ok {
				legacy[k] = v
			}
		}
	}
	hasModelParams := len(Settings.ModelParams) > 0
	if len(legacy) == 0 && !hasModelParams {
		return // 无遗留值（已迁移或本就未配置）
	}

	provider := strings.TrimSpace(GetPreset(Settings.Preset).Provider)
	entry, ok := ModelList[provider]
	if provider == "" || !ok {
		log.Printf("[config] 生成参数旧值暂未能迁入服务商配置（激活配置 %q 无可用服务商）——保留待下次启动重试",
			Settings.Preset)
		return
	}
	changed := false
	if entry.Temperature == "" {
		if v, isStr := legacy["temperature"].(string); isStr && strings.TrimSpace(v) != "" {
			entry.Temperature = strings.TrimSpace(v)
			changed = true
		}
	}
	if entry.ThinkingMode == "" {
		if v, isStr := legacy["thinkingMode"].(string); isStr && strings.TrimSpace(v) != "" {
			entry.ThinkingMode = strings.TrimSpace(v)
			changed = true
		}
	}
	if entry.MaxTokens == 0 {
		if n := anyToInt(legacy["maxTokens"]); n > 0 {
			entry.MaxTokens = n
			changed = true
		}
	}
	if entry.ContextMaxTokens == 0 {
		if n := anyToInt(legacy["contextMaxTokens"]); n > 0 {
			entry.ContextMaxTokens = n
			changed = true
		}
	}
	if changed {
		ModelList[provider] = entry
		if err := SaveModelList(); err != nil {
			log.Printf("[config] 生成参数迁入服务商 %q 失败（保留旧值待重试）: %v", provider, err)
			return
		}
		log.Printf("[config] 已把生成参数旧值迁入服务商 %q 的服务商级配置（此后生成参数唯一来源 = 服务商配置）", provider)
	}

	// 清空旧来源：生成参数唯一来源 = 服务商配置，旧字段不再有任何运行期消费者。
	Settings.Temperature = ""
	Settings.ThinkingMode = ""
	Settings.MaxTokens = 0
	Settings.ContextMaxTokens = 0
	Settings.ModelParams = nil
	if Settings.PluginSettings != nil {
		delete(Settings.PluginSettings, LegacyGenerationSettingsKey)
	}
	Save()
}
