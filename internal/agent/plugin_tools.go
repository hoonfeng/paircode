// ═══════════════════════════════════════════════════════════════
// plugin_tools.go — cordis_* 动态插件工具（模型可用）
//
// 对齐 deepseek-harness 的 tool-cordis 工具集（简化为无浏览器半）：
//   - cordis_inspect  插件运行时只读报告（插件列表 + JS 定义 + 服务/工具）
//   - cordis_define   登记一个 JS 动态插件（语法预检，不运行）
//   - cordis_run      在 goja 沙箱中求值并装载（apply 注册工具/片段等）
//   - cordis_stop     停止插件并回收其贡献
//   - cordis_undefine 先停止，再忘掉定义
//
// 动态插件只存在于进程内存：不落盘、跨重启不存续（对齐 harness 立场）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RegisterCordisTools 注册 cordis_* 动态插件管理工具。
// 由 AgentBase.Init 调用；host 为插件宿主，root 为工作区根（dir 参数解析基准）。
func RegisterCordisTools(registry *Registry, host *PluginHost, root string) {
	registry.Register(&Tool{
		Name:        "cordis_inspect",
		Description: "查看当前进程的插件运行时：全部插件/JS 动态插件定义及其状态、贡献的工具、提供的服务。插件 = { name, apply(ctx) }，JS 动态插件用 cordis_define 定义、cordis_run 装载。",
		Category:    "system",
		ReadOnly:    true,
		Parameters: objSchema(map[string]any{
			"id": strProp("可选：精确插件名或 dyn id（如 dyn-1）。省略则报告全部。"),
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return cordisInspectReport(host, argStr(args, "id"))
		},
	})

	registry.Register(&Tool{
		Name:        "cordis_define",
		Description: "登记一个 JS/TS 动态插件定义（语法预检，不运行）。code 是 async 函数体，支持两种形态：① 对象形态 return { name, apply(ctx, config), inject? }；② 函数形态 return (ctx, config) => void（cordis 生态惯例，函数名作插件名）。apply 中可用 ctx.tools.register 注册工具、ctx.systemPrompt.section 贡献提示、ctx.on 监听事件、ctx.provide 提供服务；inject: ['fs','web','bash','logger','timer',...] 声明硬依赖（宿主缺失会明确报错）。TS 源码（含 interface/type 注解）由内置编译器自动转译。返回 dyn id 供 cordis_run/stop/undefine 使用。",
		Category:    "system",
		Parameters: objSchema(map[string]any{
			"code":     strProp("插件 host 半代码（JS 或 TS，async 函数体，return { name, apply(ctx, config), inject? } 或 return (ctx, config) => void）。可访问全局：ctx/harness/console/btoa/atob/TextEncoder/TextDecoder；inject 声明后 ctx.fs/web/bash/logger/timer 可用。"),
			"language": strProp("可选：源码语言 \"js\" | \"ts\"，默认自动探测（含 interface/type 注解/类型标注视为 ts）。"),
			"purpose":  strProp("可选：插件用途说明。"),
			"dir":      strProp("可选：源码目录（解析相对 import 的多文件插件）。缺省=单文件模式（不解析 import）。"),
		}, "code"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			code := argStr(args, "code")
			if strings.TrimSpace(code) == "" {
				return "", fmt.Errorf("code 不能为空")
			}
			purpose := argStr(args, "purpose")
			language := argStr(args, "language")
			dir := ""
			if d := strings.TrimSpace(argStr(args, "dir")); d != "" {
				resolved, err := resolvePathFor(root, args, d)
				if err != nil {
					return "", err
				}
				dir = resolved
			}
			id, err := host.DefineJSCodeDir(code, language, purpose, dir)
			if err != nil {
				return "", err
			}
			extra := ""
			if dir != "" {
				extra = "，多文件 bundle（dir=" + dir + "）"
			}
			return fmt.Sprintf("已登记 %s（语言 %s，purpose: %s%s）。用 cordis_run id=%s 装载。", id, detectPluginLanguage(code, language), purpose, extra, id), nil
		},
	})

	registry.Register(&Tool{
		Name:        "cordis_run",
		Description: "装载一个已登记的 JS 动态插件（cordis_define 的 id）：在 goja 沙箱中求值并执行 apply(ctx, config)。可选 config 透传为 apply 第二参（插件配置）。正在运行的插件重复 run 会重新装载（no-op 或重放）。",
		Category:    "system",
		Parameters: objSchema(map[string]any{
			"id":     strProp("cordis_define 返回的 dyn id（如 dyn-1）。"),
			"config": strProp("可选：插件配置 JSON 对象（透传给 apply(ctx, config) 第二参）。"),
		}, "id"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			id := argStr(args, "id")
			def, ok := host.GetJSDef(id)
			if !ok {
				return "", fmt.Errorf("插件定义不存在: %s（定义只活在进程内存，跨重启不存续）", id)
			}
			if cfgStr := strings.TrimSpace(argStr(args, "config")); cfgStr != "" {
				var cfg map[string]any
				if err := json.Unmarshal([]byte(cfgStr), &cfg); err != nil {
					return "", fmt.Errorf("config 不是合法 JSON 对象: %v", err)
				}
				def.config = cfg
			}
			if err := host.LoadJSDynamic(def); err != nil {
				return "", err
			}
			return fmt.Sprintf("插件 %s (%s) 已装载并运行。可用 cordis_inspect id=%s 查看。", def.name, id, id), nil
		},
	})

	registry.Register(&Tool{
		Name:        "cordis_stop",
		Description: "停止一个运行中的插件（JS 动态插件或 Go 插件），回收其注册的工具/系统提示/事件监听；定义保留，可再次 cordis_run。",
		Category:    "system",
		Parameters: objSchema(map[string]any{
			"id": strProp("插件名或 dyn id。"),
		}, "id"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name, err := host.resolvePluginName(argStr(args, "id"))
			if err != nil {
				return "", err
			}
			if err := host.Unload(name); err != nil {
				return "", err
			}
			return fmt.Sprintf("插件 %s 已停止，贡献已回收。", name), nil
		},
	})

	registry.Register(&Tool{
		Name:        "cordis_undefine",
		Description: "删除一个 JS 动态插件定义：先停止（若在运行），再忘掉它。定义消失后 cordis_run 不再可用。",
		Category:    "system",
		Parameters: objSchema(map[string]any{
			"id": strProp("cordis_define 返回的 dyn id。"),
		}, "id"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			id := argStr(args, "id")
			if err := host.RemoveJSDef(id); err != nil {
				return "", err
			}
			return fmt.Sprintf("已删除插件定义 %s。", id), nil
		},
	})
}

// resolvePluginName 把 dyn id 或插件名解析为插件名。
func (h *PluginHost) resolvePluginName(idOrName string) (string, error) {
	if idOrName == "" {
		return "", fmt.Errorf("id 不能为空")
	}
	if def, ok := h.GetJSDef(idOrName); ok && def.name != "" {
		return def.name, nil
	}
	if _, ok := h.Get(idOrName); ok {
		return idOrName, nil
	}
	return "", fmt.Errorf("未找到插件 %q", idOrName)
}

// cordisInspectReport 生成插件运行时报告。
func cordisInspectReport(host *PluginHost, filter string) (string, error) {
	var sb strings.Builder
	recs := host.Inspect()
	defs := host.JSDefs()

	if filter != "" {
		// 精确查询：插件或 JS 定义
		for _, r := range recs {
			if r.Name == filter {
				sb.WriteString(renderPluginRecord(r))
				return sb.String(), nil
			}
		}
		for _, d := range defs {
			if d.id == filter {
				sb.WriteString(renderJSDefDetail(d))
				return sb.String(), nil
			}
		}
		return "", fmt.Errorf("未找到插件/定义 %q（定义只活在进程内存）", filter)
	}

	// 宽泛报告
	sb.WriteString("## Plugins\n")
	if len(recs) == 0 {
		sb.WriteString("（无插件）\n")
	}
	for _, r := range recs {
		sb.WriteString(fmt.Sprintf("- %s [%s] %s", r.Name, r.Source, r.State))
		if len(r.Tools) > 0 {
			sb.WriteString(fmt.Sprintf(" tools=%s", strings.Join(r.Tools, ",")))
		}
		if len(r.Provides) > 0 {
			sb.WriteString(fmt.Sprintf(" provides=%s", strings.Join(r.Provides, ",")))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n## Dynamic Packages\n")
	if len(defs) == 0 {
		sb.WriteString("（无 JS 动态插件定义）\n")
	}
	for _, d := range defs {
		state := "defined"
		if _, ok := host.Get(d.name); ok && host.State(d.name) == PluginRunning {
			state = "running"
		}
		sb.WriteString(fmt.Sprintf("- %s %s (%s) %s purpose=%s\n", d.id, d.name, state, d.version, d.purpose))
	}
	return sb.String(), nil
}

// renderPluginRecord 渲染单个插件详情。
func renderPluginRecord(r PluginRecord) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s\n", r.Name))
	sb.WriteString(fmt.Sprintf("- source: %s\n- state: %s\n", r.Source, r.State))
	if r.Version != "" {
		sb.WriteString(fmt.Sprintf("- version: %s\n", r.Version))
	}
	if len(r.Provides) > 0 {
		sb.WriteString(fmt.Sprintf("- provides: %s\n", strings.Join(r.Provides, ", ")))
	}
	if len(r.Tools) > 0 {
		sb.WriteString(fmt.Sprintf("- tools: %s\n", strings.Join(r.Tools, ", ")))
	}
	if len(r.Sections) > 0 {
		sb.WriteString(fmt.Sprintf("- systemPrompt sections: %s\n", strings.Join(r.Sections, ", ")))
	}
	return sb.String()
}

// renderJSDefDetail 渲染 JS 定义详情（含代码预览）。
func renderJSDefDetail(d *jsPluginDef) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s\n", d.id))
	form := "对象形态"
	if d.isFunc {
		form = "函数形态"
	}
	sb.WriteString(fmt.Sprintf("- name: %s\n- purpose: %s\n- version: %s\n- form: %s\n- createdAt: %s\n",
		d.name, d.purpose, d.version, form, d.createdAt.Format(time.RFC3339)))
	if len(d.inject) > 0 {
		sb.WriteString(fmt.Sprintf("- inject: %s\n", strings.Join(d.inject, ", ")))
	}
	if len(d.config) > 0 {
		if b, err := json.Marshal(d.config); err == nil {
			sb.WriteString(fmt.Sprintf("- config: %s\n", string(b)))
		}
	}
	if len(d.provides) > 0 {
		sb.WriteString(fmt.Sprintf("- provides: %s\n", strings.Join(d.provides, ", ")))
	}
	preview := d.code
	if len(preview) > 800 {
		preview = preview[:800] + "\n…（截断）"
	}
	sb.WriteString("\n```js\n" + preview + "\n```\n")
	return sb.String()
}

