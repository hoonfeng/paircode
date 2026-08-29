// jsplugin_providerfactory.go — JS Provider 装配器桥（ctx.providerFactory.register）
//
// 把 JS 装配器（插件注册的 apply 函数）接到 ProviderFactory 接口：Apply 时把当前
// 参数快照传给 JS，合并返回的 overrides 作为最终 Provider 参数。
//
// 覆盖语义（对齐 loopFactory 单槽位）：
//   - 字符串字段：非空才覆盖
//   - temperature：≥0 才覆盖（-1/缺省 = 不覆盖，维持基线）
//   - maxTokens：>0 才覆盖（0/缺省 = 不覆盖）
//   - 返回 null/undefined = 不改动
//   - 装配器执行失败：回退基线（不阻断业务）

package agent

import (
	"github.com/hoonfeng/paircode/goja"
)

// jsProviderFactoryBridge 把 JS 装配器接到 ProviderFactory 接口。
type jsProviderFactoryBridge struct {
	vm     *goja.Runtime
	apply  goja.Callable // apply(current) → overrides | null
	plugin *jsPluginAdapter
}

// Apply 实现 ProviderFactory：JS 装配 → 参数合并。
func (b *jsProviderFactoryBridge) Apply(current ProviderParams) ProviderParams {
	snap := map[string]any{
		"provider":                 current.Provider,
		"baseURL":                  current.BaseURL,
		"apiKey":                   current.APIKey,
		"model":                    current.Model,
		"temperature":              current.Temperature,
		"maxTokens":                current.MaxTokens,
		"thinkingMode":             current.ThinkingMode,
		"planModel":                current.PlanModel,
		"reviewModel":              current.ReviewModel,
		"contextMaxTokens":         current.ContextMaxTokens,
		"providerContextMaxTokens": current.ProviderContextMaxTokens, // ★ 服务商级默认上下文（模型级未配置时兜底）
		"modelParams":              current.ModelParams,
	}
	var (
		ret     goja.Value
		callErr error
	)
	// goja 非并发安全：JS apply 必须持 VM 锁执行（可能来自任意 goroutine）。
	b.plugin.withLock(func() {
		v, err := b.apply(goja.Undefined(), b.vm.ToValue(snap))
		ret, callErr = v, err
	})
	if callErr != nil {
		// 装配失败回退基线（不阻断业务；错误已在 VM 侧记录）
		return current
	}
	if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
		return current
	}
	return b.applyOverrides(current, ret.ToObject(b.vm))
}

// applyOverrides 从 JS 返回对象读取字段并覆盖到 params（非空才覆盖）。
func (b *jsProviderFactoryBridge) applyOverrides(cur ProviderParams, obj *goja.Object) ProviderParams {
	out := cur
	get := func(key string) goja.Value { return obj.Get(key) }

	if v := get("baseURL"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) && v.String() != "" {
		out.BaseURL = v.String()
	}
	if v := get("apiKey"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) && v.String() != "" {
		out.APIKey = v.String()
	}
	if v := get("model"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) && v.String() != "" {
		out.Model = v.String()
	}
	if v := get("contextMaxTokens"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		if n := int(v.ToInteger()); n > 0 {
			out.ContextMaxTokens = n
		}
	}
	if v := get("temperature"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		if f := v.ToFloat(); f >= 0 {
			out.Temperature = f
		}
	}
	if v := get("maxTokens"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		if n := int(v.ToInteger()); n > 0 {
			out.MaxTokens = n
		}
	}
	if v := get("thinkingMode"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) && v.String() != "" {
		out.ThinkingMode = v.String()
	}
	// ★ 2026-08-21 多模态：装配器按模型级参数标记（agentloop 读 modelParams[provider][model].multimodal）
	if v := get("multimodal"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		out.Multimodal = v.ToBoolean()
	}
	if v := get("planModel"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) && v.String() != "" {
		out.PlanModel = v.String()
	}
	if v := get("reviewModel"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) && v.String() != "" {
		out.ReviewModel = v.String()
	}
	return out
}
