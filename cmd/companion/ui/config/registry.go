// Package config 是 companion 的「配置声明 UI」地基 —— 把 UI 声明(JSON 配置/Go ComponentSpec)
// 与逻辑(事件 handler)分离。设计目标(对应维护者要求):
//   - 窗口→容器→组件:UI 是嵌套的 ComponentSpec 树(组件 + props + children),不再手搓 widget 属性。
//   - 配置声明 UI:静态 UI 写成 JSON 配置文件(可喂设计器/热加载);有状态表单用 Go ComponentSpec 算 props。
//   - 代码只挂事件:组件 events 字段按名引用 handler,代码只在此注册处理函数(On)。
//   - 复用库地基:底层是 internal/widget 的 declarative 系统(LoadConfig/BuildFromSpec/RegisterComponent),
//     companion 各面板与表单组件 RegisterComponent 进去,即可在配置里按 type 引用。
//
//go:build windows

package config

import (
	"encoding/json"

	"github.com/hoonfeng/goui/pkg/widget"
)

// handlers 累积所有已注册的事件处理器(配置里 events 字段按名引用;代码挂事件的唯一入口)。
var handlers = widget.Handlers{}

// On 注册一个事件处理器。配置中 "events":{"click":"name"} 即按 name 引用。重复名后者覆盖。
func On(name string, fn widget.EventHandler) { handlers[name] = fn }

// OnClick 便捷注册无参点击 handler(最常见,免去包 EventContext)。
func OnClick(name string, fn func()) {
	handlers[name] = func(widget.EventContext) { fn() }
}

// OnValue 便捷注册「带值」handler(Input change/Select change/Switch change 等,值在 ctx.Data)。
func OnValue(name string, fn func(any)) {
	handlers[name] = func(ctx widget.EventContext) { fn(ctx.Data) }
}

// Build 用累积的 handlers 把 Go ComponentSpec 构建成 widget 树(有状态表单:调用方算好 props)。
func Build(spec widget.ComponentSpec) widget.Widget { return widget.BuildFromSpec(spec, handlers) }

// Load 用累积的 handlers 从 JSON 数据加载 UI(静态 UI / 设计器产物)。
func Load(data []byte) (widget.Widget, error) { return widget.LoadConfig(data, handlers) }

// LoadFile 从 JSON 文件加载 UI(外部配置热加载,免重编译)。
func LoadFile(path string) (widget.Widget, error) { return widget.LoadConfigFile(path, handlers) }

// Register 注册一个组件类型(薄封装 widget.RegisterComponent,便于 companion 只 import config)。
// 面板(ChatPanel/Editor…)与表单组件(Field/Toggle/FontPicker)注册后即可在配置里按 type 引用。
func Register(name string, factory widget.ComponentFactory) { widget.RegisterComponent(name, factory) }

// ParseSpec 把 JSON 配置解析成 ComponentSpec(不立即构建)。用于「配置缓存一次、每帧 config.Build 重建」
// 的有状态面板(避免每帧读文件/解析 JSON)；纯静态 UI 直接 Load/LoadFile 即可。
func ParseSpec(data []byte) (widget.ComponentSpec, error) {
	var spec widget.ComponentSpec
	err := json.Unmarshal(data, &spec)
	return spec, err
}
