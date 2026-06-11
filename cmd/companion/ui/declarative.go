// ui 组件的声明式注册 —— 让每个 ui 组件既能 imperative 调用(ui.PrimaryBtn(...))，
// 又能在配置里按 type 声明({"type":"PrimaryBtn","text":"保存","events":{"click":"onSave"}})，
// 同一套观感。库已占用 Row/Col/Card/Text/Button/Input/Select 等通用名(走 widget 内置 + applyTheme
// 对齐主题)；这里只注册 companion 专有的语义组件名，避免重复注册 panic。
//
//go:build windows

package ui

import "github.com/user/goui/pkg/widget"

func init() { registerComponents() }

// registerComponents 把 companion 专有 ui 组件注册进 declarative 注册表。
func registerComponents() {
	// ── 按钮族(text + 可选 props.icon + click 事件) ──
	reg("PrimaryBtn", btnFactory(PrimaryBtnX))
	reg("Btn", btnFactory(BtnX))
	reg("GhostBtn", btnFactory(GhostBtnX))
	reg("DangerBtn", btnFactory(DangerBtnX))
	reg("SuccessBtn", btnFactory(SuccessBtnX))
	reg("SolidDangerBtn", btnFactory(SolidDangerBtnX))

	// ── 图标按钮(props.icon + click) ──
	reg("IconBtn", func(ctx widget.DeclarativeContext) widget.Widget {
		return IconBtn(propStr(ctx, "icon"), clickOf(ctx))
	})

	// ── 文字族(text) ──
	reg("Muted", textFactory(Muted))
	reg("Subtle", textFactory(Subtle))
	reg("Title", textFactory(Title))

	// ── 开关 / 药丸(props.on + change 事件) ──
	reg("Toggle", func(ctx widget.DeclarativeContext) widget.Widget {
		return Toggle(ctx.Spec.Text, propBool(ctx, "on"), changeOf(ctx))
	})
	reg("Pill", func(ctx widget.DeclarativeContext) widget.Widget {
		return Pill(propStrOr(ctx, "onText", "启用"), propStrOr(ctx, "offText", "禁用"), propBool(ctx, "on"), changeOf(ctx))
	})
	reg("AccentPill", func(ctx widget.DeclarativeContext) widget.Widget {
		return AccentPill(propStrOr(ctx, "onText", "已选"), propStrOr(ctx, "offText", "未选"), propBool(ctx, "on"), changeOf(ctx))
	})

	// ── 间距 / 分隔 ──
	reg("VGap", func(ctx widget.DeclarativeContext) widget.Widget { return VGap(propNum(ctx, "size")) })
	reg("HGap", func(ctx widget.DeclarativeContext) widget.Widget { return HGap(propNum(ctx, "size")) })
	reg("UIDivider", func(ctx widget.DeclarativeContext) widget.Widget { return Divider() })

	// ── 小节标题(props.icon/title/sub) ──
	reg("SectionHeader", func(ctx widget.DeclarativeContext) widget.Widget {
		return SectionHeader(propStr(ctx, "icon"), propStrOr(ctx, "title", ctx.Spec.Text), propStr(ctx, "sub"))
	})

	// ── 表单字段(text=标签 + child=控件) ──
	reg("Field", func(ctx widget.DeclarativeContext) widget.Widget {
		var control widget.Widget
		if len(ctx.Children) > 0 {
			control = ctx.Children[0]
		}
		return Field(ctx.Spec.Text, control)
	})

	// ── 卡片 / 面板容器(children) ──
	reg("UICard", func(ctx widget.DeclarativeContext) widget.Widget { return Card(ctx.Children...) })
	reg("UIPanel", func(ctx widget.DeclarativeContext) widget.Widget { return Col(ctx.Children...) })
}

// reg 注册一个 companion 专有组件类型(薄封装，集中处理)。
func reg(name string, f widget.ComponentFactory) { widget.RegisterComponent(name, f) }

// btnFactory 把「label+onClick+opts→Widget」按钮 helper 包成工厂(读 text + 可选 props.icon + click 事件)。
func btnFactory(fn func(string, func(), BtnOpts) widget.Widget) widget.ComponentFactory {
	return func(ctx widget.DeclarativeContext) widget.Widget {
		return fn(ctx.Spec.Text, clickOf(ctx), BtnOpts{Icon: propStr(ctx, "icon")})
	}
}

// textFactory 把「string→*Text」文字 helper 包成工厂。
func textFactory(fn func(string) *widget.Text) widget.ComponentFactory {
	return func(ctx widget.DeclarativeContext) widget.Widget { return fn(ctx.Spec.Text) }
}

// ─── 事件 / props 取值小助手 ───

// clickOf 取 click 事件对应的处理器(无则空函数)。
func clickOf(ctx widget.DeclarativeContext) func() {
	if name, ok := ctx.Spec.Events["click"]; ok {
		if h, ok := ctx.Handlers[name]; ok {
			return func() { h(widget.EventContext{Name: "click"}) }
		}
	}
	return func() {}
}

// changeOf 取 change 事件对应的处理器(开关/药丸切换用，无则空函数)。
func changeOf(ctx widget.DeclarativeContext) func() {
	if name, ok := ctx.Spec.Events["change"]; ok {
		if h, ok := ctx.Handlers[name]; ok {
			return func() { h(widget.EventContext{Name: "change"}) }
		}
	}
	return func() {}
}

func propStr(ctx widget.DeclarativeContext, key string) string {
	if v, ok := ctx.Spec.Props[key].(string); ok {
		return v
	}
	return ""
}
func propStrOr(ctx widget.DeclarativeContext, key, def string) string {
	if s := propStr(ctx, key); s != "" {
		return s
	}
	return def
}
func propBool(ctx widget.DeclarativeContext, key string) bool {
	b, _ := ctx.Spec.Props[key].(bool)
	return b
}
func propNum(ctx widget.DeclarativeContext, key string) float64 {
	switch v := ctx.Spec.Props[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
}
