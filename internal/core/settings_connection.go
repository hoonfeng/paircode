package core

import (
	"log"
	"strings"
)

// settings_connection.go — AI 连接字段（服务商/地址/Key/模型）的插件数据面收敛（★ 2026-09-20）。
//
// ★ 设计纪律（同 settings_generation.go）：配置项的唯一来源是插件数据面——
//   - AI 连接配置（服务商/地址/Key/模型/参数）：ai-presets.json
//     （插件经 ctx.aiPresets 读写；前端「AI 配置」面板维护，应用后只写 settings.preset）；
//   - 服务商数据（地址/协议/模型列表/服务商级参数）：models.json（插件经 ctx.models 读写）。
//
// 而 AppSettings 顶层的 provider/baseURL/apiKey/model/executeModel/planModel/reviewModel
// 是 2026-08-21 之前的旧存储形态，长期只作兼容兜底——装配器要多兜一层
// （current.provider || s.provider），且「应用配置」只写 preset 后顶层值会与激活配置漂移。
// 本文件把这条旧形态一次性收敛：
//
//   - 迁移（MigrateLegacyConnectionToPreset）：顶层连接字段 → 一条 AI 配置
//     （仅补该配置的空字段，不覆盖用户已保存的值），并把 settings.preset 指向它；
//     随后清空顶层旧字段（omitempty → 此后不再写回 settings.json）；
//   - 此后 Go 内核零直读连接字段：resolveProviderBase 只传装配上下文（Preset 与会话三元组），
//     连接信息一律由装配器按激活配置经 ctx.aiPresets/ctx.models 展开。
//
// ★ settings.preset 保留在核心：它是「当前激活配置名」（UI 状态 + 装配上下文），
//   不是配置值本身——配置值全部在 ai-presets.json。

// LegacyConnectionPresetName 旧顶层连接字段迁移生成的配置名。
// （仅当 settings.preset 为空时使用；用户已有同名配置时沿用其字段。）
const LegacyConnectionPresetName = "默认配置"

// MigrateLegacyConnectionToPreset 把 settings.json 顶层的 AI 连接字段一次性迁入
// ai-presets.json（一条命名配置），并清空顶层旧字段。
//
// 语义：
//   - 目标配置名 = settings.preset（非空时沿用，保持用户当前激活项），否则 LegacyConnectionPresetName；
//   - 只补目标配置的空字段（配置是唯一来源，用户已保存的值优先，不覆盖）；
//   - 迁移后 settings.preset 指向该配置 → 装配器按配置整套展开，与旧顶层兜底行为一致；
//   - 幂等：仅当顶层仍存在非空连接字段时才动作（清空后重跑直接返回）。
//
// ★ 调用时机：core.Load 内、MigrateParamSettingsToModels 之后——params 迁移仍要读
// Settings.Provider 定位「当前服务商」，故连接字段的清理必须排在它后面。
func MigrateLegacyConnectionToPreset() {
	provider := strings.TrimSpace(Settings.Provider)
	baseURL := strings.TrimSpace(Settings.BaseURL)
	apiKey := strings.TrimSpace(Settings.APIKey)
	model := strings.TrimSpace(Settings.ExecuteModel)
	if model == "" {
		model = strings.TrimSpace(Settings.Model) // 旧单模型字段（Load 内已回填，此处双保险）
	}
	if provider == "" && baseURL == "" && apiKey == "" && model == "" {
		return // 无遗留值（已迁移，或本就由 AI 配置承载）
	}

	name := strings.TrimSpace(Settings.Preset)
	if name == "" {
		name = LegacyConnectionPresetName
	}
	g := GetAiPresets() // 懒加载（Load 内已 EnsureAiPresets）
	if g == nil {
		g = AiPresets{}
	}
	p := g[name]
	filled := false
	if p.Provider == "" && provider != "" {
		p.Provider, filled = provider, true
	}
	if p.BaseURL == "" && baseURL != "" {
		p.BaseURL, filled = baseURL, true
	}
	if p.APIKey == "" && apiKey != "" {
		p.APIKey, filled = apiKey, true
	}
	if p.ExecuteModel == "" && model != "" {
		p.ExecuteModel, filled = model, true
	}
	g[name] = p
	PresetList = g
	if filled {
		if err := SaveAiPresets(); err != nil {
			// 落盘失败则保留顶层旧字段（下次启动重试），避免配置丢失
			log.Printf("[config] 旧连接配置写入 ai-presets.json 失败: %v（暂保留 settings 顶层字段）", err)
			return
		}
	}

	Settings.Preset = name
	// 清空顶层旧字段：此后连接信息唯一来源 = ai-presets.json（激活配置）
	Settings.Provider = ""
	Settings.BaseURL = ""
	Settings.APIKey = ""
	Settings.Model = ""
	Settings.ExecuteModel = ""
	Settings.PlanModel = ""
	Settings.ReviewModel = ""
	Save()
	log.Printf("[config] 已把 settings.json 的连接配置迁入 AI 配置 %q（此后连接信息以 ai-presets.json 为准）", name)
}
