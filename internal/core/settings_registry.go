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
//
// ★ 配置本身无内置（2026-08-19）：所有配置项由插件经 ctx.registerSettings 注册，
//
//	前端设置面板纯 schema 驱动渲染。组件类型规范：
//	  text / password / number / checkbox / select / textarea / slider(0-100) /
//	  color(hex) / tags(逗号分隔数组) / roles(键值对文本区)
type SettingField struct {
	Name    string   `json:"name"`              // 字段名（key——值对象属性名）
	Label   string   `json:"label"`             // 显示名（名称）
	Type    string   `json:"type"`              // 组件类型：text|password|number|checkbox|select|textarea|slider|color|tags|roles
	Default any      `json:"default,omitempty"` // 默认值（前端合并；无则零值）
	Options []string `json:"options,omitempty"` // select 的可选项
	Hint    string   `json:"hint,omitempty"`    // 提示文字
	Group   string   `json:"group,omitempty"`   // 分组标题（同一 tab 内分组）
	Binding string   `json:"binding,omitempty"` // ★ AppSettings 顶层字段绑定（settings.json 的 json key）；
	//   空 = 插件命名空间（存 pluginSettings[key]，由注册插件自行消费）；
	//   非空 = 核心运行字段（存 AppSettings 顶层，宿主 Go 运行时读取）。
	Placeholder string `json:"placeholder,omitempty"` // 输入占位文字
	Min         *int   `json:"min,omitempty"`         // number/slider 最小值
	Max         *int   `json:"max,omitempty"`         // number/slider 最大值
	Step        *int   `json:"step,omitempty"`        // number/slider 步长
	// ★ 2026-08-19 动态数据源与联动（前端通用渲染，纯 schema 声明）：
	OptionsSource string   `json:"optionsSource,omitempty"` // select 动态选项源：'models'=按服务商模型列表 / 'providers'=服务商列表（经 /api/models）
	LinkField     string   `json:"linkField,omitempty"`     // select 变化时联动填充的字段名（如 provider→baseURL，经 providerBaseURLs）
	LinkFields    []string `json:"linkFields,omitempty"`    // ★ 多字段联动（如 provider→[baseURL, apiKey]；apiKey 经 providerKeys 填充）
	// ★ 2026-08-21 模型参数定义（provider-manager 专用）：声明服务商编辑表单内
	//   逐模型参数区的字段清单（温度/思考档位/输出上限/上下文窗口/多模态…），
	//   前端 ProviderManager 按此 schema 动态渲染，参数定义全部在配置注册里。
	ModelParamFields []ModelParamFieldDef `json:"modelParamFields,omitempty"`
	// ★ 2026-08-21 模型编辑器声明（provider-manager 专用）：{label, placeholder} 声明
	//   添加模型区组件配置，前端 ProviderManager 按此渲染模型编辑器（schema 驱动）。
	ModelEditor *ModelEditorDef `json:"modelEditor,omitempty"`
	// ★ 2026-09-01 AI 配置表单字段（preset-manager 专用）：声明「添加/编辑 AI 配置」
	//   弹窗内的字段清单（如 provider/apiKey），前端 PresetManager 按此 schema 动态渲染，
	//   与 ProviderManager 的 modelParamFields 同模式（配置字段全在插件注册里）。
	PresetFields []map[string]any `json:"presetFields,omitempty"`
}

// ModelEditorDef provider-manager 的模型编辑器声明（schema 驱动）。
type ModelEditorDef struct {
	Label       string `json:"label,omitempty"`       // 区域标题（缺省用组件默认）
	Placeholder string `json:"placeholder,omitempty"` // 输入框占位（缺省用组件默认）
}

// ModelParamFieldDef provider-manager 的单个模型参数定义（schema 驱动）。
type ModelParamFieldDef struct {
	Name    string   `json:"name"`              // 参数名（settings.modelParams[provider][model] 的属性名）
	Label   string   `json:"label"`             // 显示名
	Type    string   `json:"type"`              // checkbox|select|number|text
	Default any      `json:"default,omitempty"` // 默认值
	Options []string `json:"options,omitempty"` // select 可选项
	Hint    string   `json:"hint,omitempty"`    // 提示文字
	Min     *int     `json:"min,omitempty"`     // number 最小值
	Max     *int     `json:"max,omitempty"`     // number 最大值
	Step    *int     `json:"step,omitempty"`    // number 步长
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
