package core

// settings_registry.go — 插件配置注册机制（2026-08-16）
//
// 让各插件通过 JS ctx.registerSettings(schema) 注册自己的配置段：
//   - schema 进全局注册表 PluginSettingSchemas（重启随插件装载重建）
//   - 值存 settings.json 的 pluginSettings[key]（按命名空间隔离）
//   - 前端设置面板按 schemas 动态渲染插件 tab，插件运行时用
//     ctx.getSettings(key) / ctx.setSettings(key, value) 读写。
//
// 内置核心字段（AppSettings 顶层）保持不变；插件配置不再写死进结构体。

// SettingField 一个可注册的配置字段（前端按 type 渲染控件）。
type SettingField struct {
	Name    string   `json:"name"`              // 字段名（key）
	Label   string   `json:"label"`             // 显示名
	Type    string   `json:"type"`              // text|password|number|checkbox|select|textarea
	Default any      `json:"default,omitempty"` // 默认值（前端合并；无则零值）
	Options []string `json:"options,omitempty"` // select 的可选项
	Hint    string   `json:"hint,omitempty"`    // 提示文字
	Group   string   `json:"group,omitempty"`   // 分组标题（同一 tab 内分组）
}

// SettingSchema 一个插件注册的配置段（前端渲染为一个 tab）。
type SettingSchema struct {
	Key    string         `json:"key"`    // 命名空间（settings.pluginSettings[key]；缺省=插件名）
	Title  string         `json:"title"`  // tab 标题（缺省=key）
	Fields []SettingField `json:"fields"` // 字段清单
}

// PluginSettingSchemas 全局插件配置注册表（插件装载时 registerSettings 填充；
// 进程重启后随插件重新装载重建，不持久化）。
var PluginSettingSchemas []SettingSchema

// RegisterPluginSettingSchema 注册/覆盖一个配置段（同名 key 覆盖）。
// 幂等：同一插件重复装载不会累积。
func RegisterPluginSettingSchema(s SettingSchema) {
	for i := range PluginSettingSchemas {
		if PluginSettingSchemas[i].Key == s.Key {
			PluginSettingSchemas[i] = s
			return
		}
	}
	PluginSettingSchemas = append(PluginSettingSchemas, s)
}

// PluginSettingValue 读某命名空间的当前值（未设置 → 空 map）。
func (s *AppSettings) PluginSettingValue(key string) map[string]any {
	if s.PluginSettings == nil {
		return map[string]any{}
	}
	v, ok := s.PluginSettings[key]
	if !ok || v == nil {
		return map[string]any{}
	}
	return v
}

// PluginSettingDefaults 汇总某命名空间所有已注册字段的默认值（前端合并用）。
func PluginSettingDefaults(key string) map[string]any {
	out := map[string]any{}
	for _, sch := range PluginSettingSchemas {
		if sch.Key != key {
			continue
		}
		for _, f := range sch.Fields {
			if f.Default != nil {
				out[f.Name] = f.Default
			}
		}
	}
	return out
}
