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
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

// RegisterCordisTools 注册 cordis_* 动态插件管理工具。
// 由 AgentBase.Init 调用；host 为插件宿主，root 为工作区根（dir 参数解析基准）。
func RegisterCordisTools(registry *Registry, host *PluginHost, root string) {
	registry.Register(&Tool{
		Name:        "cordis_inspect",
		Description: "查看当前进程的插件运行时（三层自检，对齐 harness cordis_inspect_self）：① 无 id → 摘要（插件/动态包列表，含版本数与 waiting 提示）；② id=pluginId 或 dyn id → 版本链（当前活动版本 + 各版本状态）；③ id + version=vN → 指定版本源码与完整运行诊断（diag/lastError）。插件 = { name, apply(ctx) }，JS 动态插件用 cordis_define 定义、cordis_run 装载。",
		Category:    "system",
		ReadOnly:    true,
		Parameters: objSchema(map[string]any{
			"id":      strProp("可选：精确插件名、dyn id（如 dyn-1）或 pluginId（稳定身份）。省略则报告全部（L1 摘要）。"),
			"version": strProp("可选：配合 id 指定版本号（如 v2）查看该版本源码+诊断（L3）。缺省=版本链概览（L2）。"),
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return cordisInspectReport(host, argStr(args, "id"), argStr(args, "version"))
		},
	})

	registry.Register(&Tool{
		Name:        "cordis_define",
		Description: "登记一个 JS/TS 动态插件定义（语法预检，不运行）。★ 创建前建议 cordis_inspect 检查已有插件/工具（同名插件或工具会导致冲突——define 后自动检测并提示）。★ 登记后自动同步（重启自动装配、前端插件面板可见可管理），cordis_run 装载后工具对 agent 可用。★ 作用域 scope：含 client 半的 UI 类插件默认 global——全局插件跨工作区生效，存 <安装目录>/.pair/plugins/dynamic.json（★ 独立于工具集，不属于任何工具集）；纯 host 工具插件默认 project（工作区 .pair/toolsets/dynamic.json 工具集，按项目加载）。code 是 async 函数体（host 半，宿主进程内运行），支持两种形态：① 对象形态 return { name, apply(ctx, config), inject? }；② 函数形态 return (ctx, config) => void（cordis 生态惯例，函数名作插件名）。apply 中可用 ctx.tools.register 注册工具、ctx.systemPrompt.section 贡献提示、ctx.on 监听事件、ctx.provide 提供服务；inject: ['fs','web','bash','logger','timer',...] 声明硬依赖（宿主缺失时插件进入 waiting，服务出现后自动激活；可选服务用 ctx.get(name) 判 undefined）。可选 client 参数提供浏览器半代码（UI 侧运行，web 界面插件面板装载）：形态 (ui) => void，ui 提供 on/emit/registerPanel/http 等浏览器侧服务。TS 源码（含 interface/type 注解）由内置编译器自动转译。★ 版本化：pluginId 非空时向已有插件追加新版本（对齐 harness define existing append）；缺省=新建插件。返回 dyn id（精确版本）供 cordis_run/stop/undefine 使用。",
		Category:    "system",
		Parameters: objSchema(map[string]any{
			"code":     strProp("插件 host 半代码（JS 或 TS，async 函数体，return { name, apply(ctx, config), inject? } 或 return (ctx, config) => void）。可访问全局：ctx/harness/console/btoa/atob/TextEncoder/TextDecoder/CordisApi（内置真 cordis 运行时，new CordisApi.api.Context() 建 cordis app 跑生态插件协作）；inject 声明后 ctx.fs/web/bash/logger/timer 可用。"),
			"client":   strProp("可选：插件 client 半代码（浏览器端执行，web 界面插件面板装载）。形态 (ui) => void：ui.on(event, fn) 收 host 事件（ui:/client: 前缀）、ui.emit(event, payload) 发事件回 host（host: 前缀给 host 插件消费）、ui.invoke(plugin, method, args?) 远程调用 host 半 ctx.registerClientMethod 注册的方法（invoke RPC）、ui.reportFailure(phase, message) 失败上报（render/guard/boot，Agent inspect 可查）、ui.registerPanel({id,title,icon,render,props}) 注册自定义面板（render(el, ui)，el 为容器 DOM，ui 为当前沙箱对象）、ui.http.get/post 调后端 API。★ 含 client 半 = UI 类插件，自动 global 作用域（跨工作区生效）。"),
			"language": strProp("可选：源码语言 \"js\" | \"ts\"，默认自动探测（含 interface/type 注解/类型标注视为 ts）。"),
			"purpose":  strProp("可选：插件用途说明。"),
			"pluginId": strProp("可选：已有插件的稳定 id（cordis_define 首次返回的 dyn-<n> 即稳定身份）。非空=向该插件追加新版本（existing append）；缺省=新建插件。追加版本后 cordis_run 传 pluginId 装载最新版。"),
			"dir":      strProp("可选：源码目录（解析相对 import 的多文件插件）。缺省=单文件模式（不解析 import）。"),
			"scope":    strProp("可选：生效作用域 \"global\"=全局（跨工作区，UI 类插件默认）|\"project\"=项目（默认，纯工具插件）。含 client 半时自动 global。"),
		}, "code"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			code := argStr(args, "code")
			if strings.TrimSpace(code) == "" {
				return "", fmt.Errorf("code 不能为空")
			}
			purpose := argStr(args, "purpose")
			language := argStr(args, "language")
			clientCode := argStr(args, "client")
			pluginId := strings.TrimSpace(argStr(args, "pluginId"))
			scope := strings.TrimSpace(argStr(args, "scope"))
			dir := ""
			if d := strings.TrimSpace(argStr(args, "dir")); d != "" {
				resolved, err := resolvePathFor(root, args, d)
				if err != nil {
					return "", err
				}
				dir = resolved
			}
			id, err := host.DefineJSCodeVersioned(code, language, purpose, dir, clientCode, pluginId)
			if err != nil {
				return "", err
			}
			// ★ 创建工具插件前考虑已有插件/工具（需求）：define 成功后检测同名冲突，
			//   返回信息附带 warning 引导 agent 决策；并实时同步到工作区工具集。
			def, _ := host.GetJSDef(id)
			var warnings []string
			pname := extractJSPluginName(def.code)
			if pname != "" {
				for _, other := range host.JSDefs() {
					if other.id == id {
						continue
					}
					if other.Name() == pname {
						warnings = append(warnings, fmt.Sprintf("宿主已有同名插件 %q（%s，v%s）——若两者都运行将冲突；建议 cordis_inspect 查看后决定是否换名或停旧装新",
							pname, other.id, other.version))
						break
					}
				}
				if registry != nil {
					if t, ok := registry.Get(pname); ok {
						who := "宿主内置工具"
						if host.HasPluginTool(pname) {
							who = "其他插件工具"
						}
						_ = t
						warnings = append(warnings, fmt.Sprintf("工具名 %q 已被 %s 占用——注册同名工具会被拒绝（claimTool 冲突）；如需同名请先禁用/卸载占用方，或换工具名",
							pname, who))
					}
				}
			}
			// 动态插件实时固化到插件工具集（跨重启存续；cordis_define 只登记不装载，
			// cordis_run 后工具可用，重启自动装配）。★ 作用域：含 client 半的 UI 类
			// 插件自动 global（全局生效，不进项目工具集）；scope 显式指定 global/project。
			syncMsg := ""
			if dir == "" { // 多文件 bundle 插件不固化（代码依赖 dir 相对 import）
				if scope == "" && strings.TrimSpace(clientCode) != "" {
					scope = "global" // UI 类插件默认全局
				}
				if msg, serr := syncDynamicPluginToToolset(root, def, pname, scope); serr != nil {
					log.Printf("[cordis] 同步动态插件到工具集失败: %v", serr)
				} else {
					syncMsg = msg
				}
			}
			extra := ""
			if dir != "" {
				extra = "，多文件 bundle（dir=" + dir + "）"
			}
			if strings.TrimSpace(clientCode) != "" {
				extra += "，含 client 半（浏览器 UI）"
			}
			mode := "新建插件"
			if pluginId != "" {
				mode = "向 " + pluginId + " 追加版本"
			}
			msg := fmt.Sprintf("已登记 %s（%s，version=%s 语言 %s，purpose: %s%s）。用 cordis_run id=%s 或 id=%s 装载。",
				id, mode, def.version, detectPluginLanguage(code, language), purpose, extra, id, def.pluginId)
			if len(warnings) > 0 {
				msg += "\n\n⚠️ " + strings.Join(warnings, "\n⚠️ ")
			}
			if syncMsg != "" {
				msg += "\n" + syncMsg
			}
			return msg, nil
		},
	})

	registry.Register(&Tool{
		Name:        "cordis_run",
		Description: "装载一个已登记的 JS 动态插件（cordis_define 的 id 或 pluginId）：在 goja 沙箱中求值并执行 apply(ctx, config)。可选 config 透传为 apply 第二参（插件配置）。id 可传精确 dyn id（指定版本）或 pluginId（最新版本）；已运行的插件重复 run 会先卸载旧实例再装载新版本（restart 语义，对齐 harness run mode=run）。inject 声明服务缺失时插件进入 waiting（服务出现后自动激活，可用 cordis_inspect 查看）。",
		Category:    "system",
		Parameters: objSchema(map[string]any{
			"id":     strProp("cordis_define 返回的 dyn id（如 dyn-1，精确版本）或 pluginId（稳定身份=首次 dyn id，装载最新版本）。"),
			"config": strProp("可选：插件配置 JSON 对象（透传给 apply(ctx, config) 第二参）。"),
		}, "id"),
		// ★ 2026-08-19：client 半激活审批机制整体取消（参考项目 deepseek-harness
		//   无此机制）→ 恒 false：装载带 client 半的插件不再触发审批门，浏览器
		//   直接装载（IsClientApproved 恒 true）。
		DynamicApproval: func(tc ToolCall) bool {
			return false
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			id := argStr(args, "id")
			def, err := host.resolveJSDef(id)
			if err != nil {
				return "", err
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
			// ★ 2026-08-19：client 半激活审批机制整体取消（参考项目无此机制），
			//   不再 MarkClientApproved；浏览器直接装载全部 client 半。
			// 等待语义：装载成功但插件进入 waiting（inject 缺服务）
			if def.status == PluginWaiting {
				msg := fmt.Sprintf("插件 %s (%s v%s) 已进入 waiting：inject 声明 %v 中宿主未提供 %v。服务出现后将自动激活；可用 cordis_inspect id=%s 查看。",
					def.name, def.id, def.version, def.inject, def.waitingFor, def.pluginId)
				if c := def.ConsoleText(); c != "" {
					msg += "\n\n插件 console 输出：\n" + truncRunesAgent(c, 2000)
				}
				return msg, nil
			}
			msg := fmt.Sprintf("插件 %s (%s v%s) 已装载并运行。可用 cordis_inspect id=%s 查看。", def.name, def.id, def.version, def.pluginId)
			if c := def.ConsoleText(); c != "" {
				msg += "\n\n插件 console 输出：\n" + truncRunesAgent(c, 2000)
			}
			return msg, nil
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
		Name:        "cordis_service_list",
		Description: "列出宿主可用服务及其方法签名（写插件时先查询：inject 声明硬依赖 / ctx.get(name) 读可选服务）。静态服务（fs/web/bash/logger/timer/tools/events）声明后按 ctx.xxx 属性访问；动态服务（ctx.provide）用 ctx.get 读取。",
		Category:    "system",
		ReadOnly:    true,
		Parameters:  objSchema(map[string]any{}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return cordisServiceList(host), nil
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

	registry.Register(&Tool{
		Name:        "cordis_inspect_query",
		Description: "按精确协议查询插件运行时/宿主目录（对齐 harness cordis_inspect_query 的简化实现）。platform=host 时由宿主本地执行只读查询，不修改运行时。provider 决定查询对象：service（服务契约）/ tool（工具 schema）/ event（事件模式）/ plugin（插件记录）。method 由 provider 决定：service 支持 listService（无 input 列签名目录；input={name} 取精确契约）与 getService；tool 支持 listTool 与 getTool（input={name}）；event 支持 listEvent 与 getEvent（input={name}）；plugin 支持 listPlugin 与 getPlugin（input={name}）。写插件前先查精确签名，不要臆测。",
		Category:    "system",
		ReadOnly:    true,
		Parameters: objSchema(map[string]any{
			"platform": strProp("运行时平台：\"host\"（宿主进程，本地执行）| \"client\"（浏览器；当前返回 client 半装载状态摘要）。"),
			"provider": strProp("查询对象：service | tool | event | plugin。"),
			"method":   strProp("provider 的方法：listService/getService、listTool/getTool、listEvent/getEvent、listPlugin/getPlugin。"),
			"input":    strProp("可选：查询输入 JSON 对象（如 {name:\"fs\"}）。"),
		}, "platform", "provider", "method"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			platform := argStr(args, "platform")
			provider := argStr(args, "provider")
			method := argStr(args, "method")
			input := map[string]any{}
			if is := strings.TrimSpace(argStr(args, "input")); is != "" {
				if err := json.Unmarshal([]byte(is), &input); err != nil {
					return "", fmt.Errorf("input 不是合法 JSON 对象: %v", err)
				}
			}
			return cordisInspectQuery(host, platform, provider, method, input)
		},
	})
}

// cordisInspectQuery 执行精确协议查询（cordis_inspect_query 实现）。
// platform=client 时读取浏览器 plugin-runtime 周期上报的快照（真实页面状态），
// 未上报/过期视为离线并给宿主侧装载清单兜底。
func cordisInspectQuery(host *PluginHost, platform, provider, method string, input map[string]any) (string, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	provider = strings.ToLower(strings.TrimSpace(provider))
	method = strings.ToLower(strings.TrimSpace(method))
	if host == nil {
		return "", fmt.Errorf("插件宿主未初始化")
	}
	p := host.InspectProviderLookup(platform, provider)
	if p == nil {
		known := host.InspectProviders(platform)
		if len(known) == 0 {
			return "", fmt.Errorf("platform 必须是 host 或 client，收到 %q", platform)
		}
		return "", fmt.Errorf("provider 必须是 %s，收到 %q", strings.Join(known, "|"), provider)
	}
	mth, ok := p.Methods[method]
	if !ok {
		var names []string
		for n := range p.Methods {
			names = append(names, n)
		}
		sort.Strings(names)
		return "", fmt.Errorf("%s 平台方法必须是 %s，收到 %q", provider, strings.Join(names, "|"), method)
	}
	return mth.Query(host, input)
}

// registerBuiltinInspectProviders 注册内置 inspect provider（对齐 harness 的
// hostInspectProviders/clientInspectProviders；NewPluginHost 调用）。
// host 平台：service/tool/event/plugin（宿主侧真实状态）；
// client 平台：plugin/event/service/tool（浏览器 plugin-runtime 上报快照）。
func registerBuiltinInspectProviders(h *PluginHost) {
	// ── host/service：服务目录 + 精确契约 ──
	h.RegisterInspectProvider("host", &InspectProvider{ID: "service", Description: "Host 服务目录与精确契约", Methods: map[string]InspectMethod{
		"listservice": {Name: "listService", Description: "服务目录（签名一览）", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			var sb strings.Builder
			sb.WriteString("## Service 目录（签名一览；getService {name} 取精确契约）\n")
			static := []struct{ name, api string }{
				{"fs", "readFile/writeFile/appendFile/exists/readdir/stat/mkdir/rm（工作区受限）"},
				{"web", "fetch(url)→{ok,status,text}（GET，60s 超时，4MB 上限）"},
				{"bash", "exec(cmd,cwd?)→{output,error}（120s 超时）"},
				{"logger", "logger(scope)→{log,info,warn,debug,error}"},
				{"timer", "timeout(fn,ms)/interval(fn,ms)→cancel"},
				{"tools", "register({name,description,parameters,execute})/list()"},
				{"events", "on(name,fn)/emit(name,payload)"},
				{"app", "workspaceRoot"},
				{"workspaceRoot", "工作区根路径"},
				{"store", "会话存储"},
			}
			for _, s := range static {
				fmt.Fprintf(&sb, "- **%s**：%s\n", s.name, s.api)
			}
			var dyn []string
			h.ctx.servicesMu.RLock()
			for n := range h.ctx.services {
				dyn = append(dyn, n)
			}
			h.ctx.servicesMu.RUnlock()
			sort.Strings(dyn)
			if len(dyn) > 0 {
				sb.WriteString("\n## Dynamic services（ctx.provide/ctx.get）\n")
				for _, n := range dyn {
					fmt.Fprintf(&sb, "- %s\n", n)
				}
			}
			return sb.String(), nil
		}},
		"list": {Name: "list", Description: "服务目录别名", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			p := h.InspectProviderLookup("host", "service")
			return p.Methods["listservice"].Query(h, nil)
		}},
		"getservice": {Name: "getService", Description: "单个服务精确契约", Query: func(h *PluginHost, input map[string]any) (string, error) {
			name := ""
			if v, ok := input["name"].(string); ok {
				name = v
			}
			if name == "" {
				return "", fmt.Errorf("getService 需要 input={name}（如 {name:\"fs\"}）")
			}
			return cordisServiceContract(h, name), nil
		}},
		"get": {Name: "get", Description: "getService 别名", Query: func(h *PluginHost, input map[string]any) (string, error) {
			p := h.InspectProviderLookup("host", "service")
			return p.Methods["getservice"].Query(h, input)
		}},
	}})
	// ── host/tool：工具注册表 + 完整 schema ──
	h.RegisterInspectProvider("host", &InspectProvider{ID: "tool", Description: "Host 工具注册表", Methods: map[string]InspectMethod{
		"listtool": {Name: "listTool", Description: "工具列表", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			var sb strings.Builder
			sb.WriteString("## Tools（host 注册表；getTool {name} 取完整 schema）\n")
			for _, m := range h.ctx.Tools.AllToolMeta() {
				fmt.Fprintf(&sb, "- **%s**：%s\n", m.Name, firstLine(m.Description))
			}
			return sb.String(), nil
		}},
		"list": {Name: "list", Description: "listTool 别名", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			p := h.InspectProviderLookup("host", "tool")
			return p.Methods["listtool"].Query(h, nil)
		}},
		"gettool": {Name: "getTool", Description: "单个工具完整 schema", Query: func(h *PluginHost, input map[string]any) (string, error) {
			name := ""
			if v, ok := input["name"].(string); ok {
				name = v
			}
			if name == "" {
				return "", fmt.Errorf("getTool 需要 input={name}")
			}
			for _, m := range h.ctx.Tools.AllToolMeta() {
				if m.Name == name {
					b, _ := json.MarshalIndent(m, "", "  ")
					return string(b), nil
				}
			}
			return "", fmt.Errorf("工具 %q 不存在", name)
		}},
		"get": {Name: "get", Description: "getTool 别名", Query: func(h *PluginHost, input map[string]any) (string, error) {
			p := h.InspectProviderLookup("host", "tool")
			return p.Methods["gettool"].Query(h, input)
		}},
	}})
	// ── host/event：事件目录 + 详情 ──
	h.RegisterInspectProvider("host", &InspectProvider{ID: "event", Description: "Host 事件目录", Methods: map[string]InspectMethod{
		"listevent": {Name: "listEvent", Description: "事件列表", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			var sb strings.Builder
			names := h.EventBus().EventNames()
			sb.WriteString("## Events（当前已注册事件；getEvent {name} 取详情）\n")
			if len(names) == 0 {
				sb.WriteString("（无事件）\n")
			}
			for _, n := range names {
				fmt.Fprintf(&sb, "- %s（%d 监听器）\n", n, h.EventBus().ListenerCount(n))
			}
			return sb.String(), nil
		}},
		"list": {Name: "list", Description: "listEvent 别名", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			p := h.InspectProviderLookup("host", "event")
			return p.Methods["listevent"].Query(h, nil)
		}},
		"getevent": {Name: "getEvent", Description: "单个事件详情", Query: func(h *PluginHost, input map[string]any) (string, error) {
			name := ""
			if v, ok := input["name"].(string); ok {
				name = v
			}
			if name == "" {
				return "", fmt.Errorf("getEvent 需要 input={name}")
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "## Event %s\n- 监听器数: %d\n", name, h.EventBus().ListenerCount(name))
			sb.WriteString("- 约定：ui:/client: 前缀事件会转发浏览器 client 半；host: 前缀由浏览器发回宿主。\n")
			return sb.String(), nil
		}},
		"get": {Name: "get", Description: "getEvent 别名", Query: func(h *PluginHost, input map[string]any) (string, error) {
			p := h.InspectProviderLookup("host", "event")
			return p.Methods["getevent"].Query(h, input)
		}},
	}})
	// ── host/plugin：插件目录 + 详情 ──
	h.RegisterInspectProvider("host", &InspectProvider{ID: "plugin", Description: "Host 插件目录", Methods: map[string]InspectMethod{
		"listplugin": {Name: "listPlugin", Description: "插件列表", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			var sb strings.Builder
			sb.WriteString("## Plugins\n")
			for _, r := range h.Inspect() {
				fmt.Fprintf(&sb, "- %s [%s] %s", r.Name, r.Source, r.State)
				if len(r.Tools) > 0 {
					fmt.Fprintf(&sb, " tools=%s", strings.Join(r.Tools, ","))
				}
				if r.HasClient {
					sb.WriteString(" client=yes")
				}
				sb.WriteString("\n")
			}
			return sb.String(), nil
		}},
		"list": {Name: "list", Description: "listPlugin 别名", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			p := h.InspectProviderLookup("host", "plugin")
			return p.Methods["listplugin"].Query(h, nil)
		}},
		"getplugin": {Name: "getPlugin", Description: "单个插件详情", Query: func(h *PluginHost, input map[string]any) (string, error) {
			name := ""
			if v, ok := input["name"].(string); ok {
				name = v
			}
			if name == "" {
				return "", fmt.Errorf("getPlugin 需要 input={name}")
			}
			rec := h.InspectDetail(name)
			if rec == nil {
				return "", fmt.Errorf("插件 %q 不存在", name)
			}
			return renderPluginRecord(*rec), nil
		}},
		"get": {Name: "get", Description: "getPlugin 别名", Query: func(h *PluginHost, input map[string]any) (string, error) {
			p := h.InspectProviderLookup("host", "plugin")
			return p.Methods["getplugin"].Query(h, input)
		}},
	}})
	// ── client 平台：浏览器 client 半实时状态（plugin-runtime 上报快照）──
	registerClientInspectProviders(h)
}

// clientRuntimeHeader client 平台查询的统一头（含离线提示与上报时间）。
func clientRuntimeHeader(snap ClientRuntimeSnapshot) string {
	var sb strings.Builder
	sb.WriteString("## Client runtime（浏览器插件 client 半，实时上报）\n")
	if !snap.Connected {
		sb.WriteString("⚠️ 浏览器未连接（页面未打开或 30s 内未上报）——以下为宿主侧装载清单兜底：\n")
	} else {
		now := time.Now().Unix()
		if now-snap.ReportedAt <= clientStateTTL {
			fmt.Fprintf(&sb, "- 上报时间：%s 前（%d 秒前）\n", humanDur(now-snap.ReportedAt), now-snap.ReportedAt)
		}
	}
	return sb.String()
}

// clientInputName 提取 input.name（getPlugin/getService 等公共参数）。
func clientInputName(input map[string]any) string {
	if v, ok := input["name"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// registerClientInspectProviders client 平台内置 provider（数据源：浏览器上报快照）。
func registerClientInspectProviders(h *PluginHost) {
	// ── client/plugin：浏览器 client 半插件清单（离线给宿主侧兜底）──
	h.RegisterInspectProvider("client", &InspectProvider{ID: "plugin", Description: "Client 半插件清单", Methods: map[string]InspectMethod{
		"listplugin": {Name: "listPlugin", Description: "浏览器装载的 client 半列表", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			snap := h.ClientState()
			var sb strings.Builder
			sb.WriteString(clientRuntimeHeader(snap))
			if snap.Connected {
				if len(snap.Plugins) == 0 {
					sb.WriteString("（浏览器已连接，但无 client 半装载）\n")
				}
				for _, p := range snap.Plugins {
					mark := "loaded"
					if p.Status == "error" {
						mark = "error: " + p.Error
					}
					fmt.Fprintf(&sb, "- **%s** [%s] panels=%d events=%d version=%s\n",
						p.Name, mark, len(p.Panels), len(p.Events), p.Version)
				}
				return sb.String(), nil
			}
			// 离线兜底：宿主侧含 client 半的插件清单
			recs := h.Inspect()
			n := 0
			for _, r := range recs {
				if r.HasClient {
					fmt.Fprintf(&sb, "- %s [%s] %s client=%s\n", r.Name, r.Source, r.State, r.DefID)
					n++
				}
			}
			if n == 0 {
				sb.WriteString("（无含 client 半的插件）\n")
			}
			return sb.String(), nil
		}},
		"list": {Name: "list", Description: "listPlugin 别名", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			p := h.InspectProviderLookup("client", "plugin")
			return p.Methods["listplugin"].Query(h, nil)
		}},
		"getplugin": {Name: "getPlugin", Description: "单个 client 半详情", Query: func(h *PluginHost, input map[string]any) (string, error) {
			snap := h.ClientState()
			name := clientInputName(input)
			if name == "" {
				return "", fmt.Errorf("getPlugin 需要 input={name}（client 半插件名）")
			}
			var sb strings.Builder
			sb.WriteString(clientRuntimeHeader(snap))
			for _, p := range snap.Plugins {
				if p.Name == name {
					status := "loaded"
					if p.Status == "error" {
						status = "error: " + p.Error
					}
					fmt.Fprintf(&sb, "- **%s** [%s] version=%s\n", p.Name, status, p.Version)
					if len(p.Panels) > 0 {
						sb.WriteString("  - 面板：\n")
						for _, pid := range p.Panels {
							fmt.Fprintf(&sb, "    - %s\n", pid)
						}
					}
					if len(p.Events) > 0 {
						sb.WriteString("  - 监听事件：\n")
						for _, ev := range p.Events {
							fmt.Fprintf(&sb, "    - %s\n", ev)
						}
					}
					return sb.String(), nil
				}
			}
			return "", fmt.Errorf("浏览器 client 半中未找到 %q（离线则参考宿主侧：%v）", name, hostHasClientHalf(h, name))
		}},
		"get": {Name: "get", Description: "getPlugin 别名", Query: func(h *PluginHost, input map[string]any) (string, error) {
			p := h.InspectProviderLookup("client", "plugin")
			return p.Methods["getplugin"].Query(h, input)
		}},
	}})
	// ── client/event：浏览器侧事件监听目录 ──
	h.RegisterInspectProvider("client", &InspectProvider{ID: "event", Description: "Client 半事件监听目录", Methods: map[string]InspectMethod{
		"listevent": {Name: "listEvent", Description: "client 半注册的事件监听", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			snap := h.ClientState()
			var sb strings.Builder
			sb.WriteString(clientRuntimeHeader(snap))
			if !snap.Connected || len(snap.Plugins) == 0 {
				sb.WriteString("（浏览器未连接或无 client 半——无 client 侧事件监听可查）\n")
				return sb.String(), nil
			}
			seen := map[string]bool{}
			for _, p := range snap.Plugins {
				for _, ev := range p.Events {
					if !seen[ev] {
						fmt.Fprintf(&sb, "- %s（%s）\n", ev, p.Name)
						seen[ev] = true
					}
				}
			}
			if len(seen) == 0 {
				sb.WriteString("（client 半均未注册事件监听）\n")
			}
			return sb.String(), nil
		}},
		"list": {Name: "list", Description: "listEvent 别名", Query: func(h *PluginHost, _ map[string]any) (string, error) {
			p := h.InspectProviderLookup("client", "event")
			return p.Methods["listevent"].Query(h, nil)
		}},
	}})
	// ── client/service / client/tool：暂未纳入上报协议（提示性占位）──
	h.RegisterInspectProvider("client", &InspectProvider{ID: "service", Description: "Client 半服务（暂未上报）", Methods: map[string]InspectMethod{
		"listservice": {Name: "listService", Description: "client 半服务目录", Query: func(_ *PluginHost, _ map[string]any) (string, error) {
			return "client 半服务（ui.http 路由/数据源）尚未纳入上报协议；可用 event/plugin 平台查询面板与事件。\n", nil
		}},
		"list": {Name: "list", Description: "listService 别名", Query: func(_ *PluginHost, _ map[string]any) (string, error) {
			return "client 半服务（ui.http 路由/数据源）尚未纳入上报协议；可用 event/plugin 平台查询面板与事件。\n", nil
		}},
	}})
	h.RegisterInspectProvider("client", &InspectProvider{ID: "tool", Description: "Client 半工具（暂未上报）", Methods: map[string]InspectMethod{
		"listtool": {Name: "listTool", Description: "client 半工具列表", Query: func(_ *PluginHost, _ map[string]any) (string, error) {
			return "client 半工具（浏览器侧能力）尚未纳入上报协议；插件工具统一经 host 侧注册。\n", nil
		}},
		"list": {Name: "list", Description: "listTool 别名", Query: func(_ *PluginHost, _ map[string]any) (string, error) {
			return "client 半工具（浏览器侧能力）尚未纳入上报协议；插件工具统一经 host 侧注册。\n", nil
		}},
	}})
}

// hostHasClientHalf 宿主侧是否含指定 client 半插件（离线兜底提示）。
func hostHasClientHalf(host *PluginHost, name string) bool {
	for _, r := range host.Inspect() {
		if r.HasClient && r.Name == name {
			return true
		}
	}
	return false
}

// humanDur 秒 → 人类可读时长（如 "3 秒"、"1 分钟 5 秒"）。
func humanDur(sec int64) string {
	if sec < 60 {
		return fmt.Sprintf("%d 秒", sec)
	}
	return fmt.Sprintf("%d 分钟 %d 秒", sec/60, sec%60)
}

// cordisServiceContract 单个服务精确契约。
func cordisServiceContract(host *PluginHost, name string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## Service %s\n", name)
	switch name {
	case "fs":
		sb.WriteString("readFile(path)→string | writeFile(path, content) | appendFile(path, content) | exists(path)→bool | readdir(path)→[]string | stat(path)→{name,size,isDir,mtime} | mkdir(path, recursive?) | rm(path, recursive?)。所有路径相对工作区根，越界拦截。\n")
	case "web":
		sb.WriteString("fetch(url)→{ok, status, text}（GET，60s 超时，4MB 上限）。\n")
	case "bash":
		sb.WriteString("exec(cmd, cwd?)→{output, error}（120s 超时；cwd 相对工作区根）。\n")
	case "logger":
		sb.WriteString("logger(scope)→{log, info, warn, debug, error}（带插件标签写透宿主 stdout）。\n")
	case "timer":
		sb.WriteString("timeout(fn, ms)→cancel | interval(fn, ms)→cancel（卸载自动清理）。\n")
	case "tools":
		sb.WriteString("register({name, description, parameters, execute})→void | list()→[]。\n")
	case "events":
		sb.WriteString("on(name, fn)→cancel | emit(name, payload)。ui:/client: 前缀事件转发浏览器。\n")
	case "app":
		sb.WriteString("{ workspaceRoot }。\n")
	default:
		// 动态服务
		host.ctx.servicesMu.RLock()
		v, ok := host.ctx.services[name]
		host.ctx.servicesMu.RUnlock()
		if ok {
			fmt.Fprintf(&sb, "动态服务（ctx.provide 注册）：%T\n", v)
		} else {
			sb.WriteString("未知服务；用 listService 查看目录。\n")
		}
	}
	return sb.String()
}

// cordisServiceList 渲染宿主可用服务目录（cordis_service_list）。
func cordisServiceList(host *PluginHost) string {
	var sb strings.Builder
	sb.WriteString("## Services（inject 声明后按 ctx.xxx 访问；可选服务用 ctx.get(name) 判 undefined）\n")
	static := []struct {
		name string
		api  string
	}{
		{"fs", "工作区受限文件服务（越界拦截）：readFile(path)→string / writeFile(path, content) / appendFile(path, content) / exists(path)→bool / readdir(path)→[]string / stat(path)→{name,size,isDir,mtime} / mkdir(path, recursive?) / rm(path, recursive?)"},
		{"web", "HTTP 服务：fetch(url)→{ok, status, text}（GET，60s 超时，4MB 上限）"},
		{"bash", "shell 命令：exec(cmd, cwd?)→{output, error}（120s 超时，cwd 相对工作区根）"},
		{"logger", "日志：logger(scope)→{log, info, warn, debug, error}（带插件标签写透宿主 stdout）"},
		{"timer", "定时器：timeout(fn, ms) / interval(fn, ms)→取消函数（卸载自动清理）"},
		{"tools", "工具注册：register({name, description, parameters, execute}) / list()"},
		{"events", "事件：on(name, fn) / emit(name, payload)"},
		{"app", "宿主信息：workspaceRoot"},
		{"workspaceRoot", "工作区根路径（字符串）"},
		{"store", "会话存储（ConversationStore）"},
	}
	for _, s := range static {
		fmt.Fprintf(&sb, "- **%s**：%s\n", s.name, s.api)
	}
	// 动态服务（ctx.provide / ctx.get）
	var dyn []string
	host.ctx.servicesMu.RLock()
	for n := range host.ctx.services {
		dyn = append(dyn, n)
	}
	host.ctx.servicesMu.RUnlock()
	sort.Strings(dyn)
	if len(dyn) > 0 {
		sb.WriteString("\n## Dynamic services（ctx.provide 注册，ctx.get(name) 读取）\n")
		for _, n := range dyn {
			fmt.Fprintf(&sb, "- %s\n", n)
		}
	} else {
		sb.WriteString("\n（当前无动态服务）\n")
	}
	return sb.String()
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

// cordisInspectReport 生成插件运行时报告（三层自检，对齐 harness cordis_inspect_self）：
//
//	L1 摘要：filter 为空 → 插件列表 + 动态包列表（含版本数）+ waiting 提示
//	L2 版本：filter=pluginId/dyn id/插件名（无 version）→ 该插件版本链 + 运行状态 + 诊断摘要
//	L3 源码：filter 指向定义且带 version → 指定版本源码 + 完整诊断
func cordisInspectReport(host *PluginHost, filter, version string) (string, error) {
	var sb strings.Builder
	recs := host.Inspect()
	defs := host.JSDefs()

	if filter != "" {
		// ── 精确查询：插件或 JS 定义（L2/L3）──
		for _, r := range recs {
			if r.Name == filter {
				sb.WriteString(renderPluginRecord(r))
				// 若该插件有 JS 定义 → 附版本链（L2）
				if d := host.DefByPluginOrName(filter); d != nil {
					sb.WriteString("\n" + renderVersionChain(host, d.pluginId, version))
				}
				return sb.String(), nil
			}
		}
		for _, d := range defs {
			if d.id == filter || d.pluginId == filter {
				if version != "" {
					// L3：指定版本源码 + 完整诊断
					for _, dv := range host.PluginVersions(d.pluginId) {
						if dv.version == version {
							sb.WriteString(renderJSDefDetail(dv))
							return sb.String(), nil
						}
					}
					return "", fmt.Errorf("插件 %s 没有版本 %s（已有: %s）", filter, version, host.PluginVersionNames(d.pluginId))
				}
				sb.WriteString(renderVersionChain(host, d.pluginId, ""))
				return sb.String(), nil
			}
		}
		return "", fmt.Errorf("未找到插件/定义 %q（定义只活在进程内存；插件名或 dyn id 均可）", filter)
	}

	// ── L1 摘要 ──
	sb.WriteString("## Plugins（运行中）\n")
	n := 0
	for _, r := range recs {
		if r.State != "running" {
			continue
		}
		sb.WriteString(fmt.Sprintf("- %s [%s]", r.Name, r.Source))
		if len(r.Tools) > 0 {
			sb.WriteString(fmt.Sprintf(" tools=%s", strings.Join(r.Tools, ",")))
		}
		if len(r.Provides) > 0 {
			sb.WriteString(fmt.Sprintf(" provides=%s", strings.Join(r.Provides, ",")))
		}
		sb.WriteString("\n")
		n++
	}
	if n == 0 {
		sb.WriteString("（无运行中插件）\n")
	}

	sb.WriteString("\n## Dynamic Packages（JS 插件定义，按 pluginId 分组）\n")
	if len(defs) == 0 {
		sb.WriteString("（无 JS 动态插件定义）\n")
	}
	// 按 pluginId 分组（去重）展示
	seen := map[string]bool{}
	for _, d := range defs {
		if seen[d.pluginId] {
			continue
		}
		seen[d.pluginId] = true
		state := d.status.String()
		chain := host.PluginVersions(d.pluginId)
		verNote := ""
		if len(chain) > 1 {
			verNote = fmt.Sprintf("（%d 版本，最新 %s）", len(chain), chain[len(chain)-1].version)
		}
		extra := ""
		if d.status == PluginWaiting && len(d.waitingFor) > 0 {
			extra = fmt.Sprintf(" ⏳ waiting 缺服务: %v", d.waitingFor)
		}
		if d.status == PluginFailed || d.status == PluginRejected {
			extra = fmt.Sprintf(" ❌ %s: %s", d.status, truncateStr(d.lastError, 80))
		}
		sb.WriteString(fmt.Sprintf("- %s %s [%s] %s%s%s\n", d.pluginId, d.name, state, d.version, verNote, extra))
	}

	// waiting 汇总提示
	if wd := host.waitingDefs(); len(wd) > 0 {
		sb.WriteString("\n## Waiting（inject 服务未就绪，自动激活中）\n")
		for _, d := range wd {
			sb.WriteString(fmt.Sprintf("- %s (%s) 缺服务: %v\n", d.pluginId, d.name, d.waitingFor))
		}
	}
	sb.WriteString("\n详情：cordis_inspect id=<pluginId 或 dyn id>；源码：cordis_inspect id=<pluginId> version=<版本号>。\n")
	return sb.String(), nil
}

// DefByPluginOrName 按插件名或稳定 id 找 JS 定义（Inspect 补充用）。
func (h *PluginHost) DefByPluginOrName(name string) *jsPluginDef {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, d := range h.defs {
		if d.pluginId == name || d.name == name {
			return d
		}
	}
	return nil
}

// PluginVersionNames 某 pluginId 的版本号列表（逗号分隔）。
func (h *PluginHost) PluginVersionNames(pluginId string) string {
	chain := h.PluginVersions(pluginId)
	names := make([]string, 0, len(chain))
	for _, d := range chain {
		names = append(names, d.version)
	}
	return strings.Join(names, ", ")
}

// renderVersionChain 渲染版本链（L2：版本指针 + package 摘要）。
func renderVersionChain(host *PluginHost, pluginId, wantVersion string) string {
	chain := host.PluginVersions(pluginId)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s（%d 版本）\n", pluginId, len(chain)))
	// 当前活动版本（运行中的）
	active := ""
	for _, d := range chain {
		if d.status == PluginRunning {
			active = d.version
			break
		}
	}
	sb.WriteString(fmt.Sprintf("- pluginId: %s\n", pluginId))
	if active != "" {
		sb.WriteString(fmt.Sprintf("- current: %s（运行中）\n", active))
	} else {
		sb.WriteString("- current: （无运行中版本）\n")
	}
	sb.WriteString("- versions:\n")
	for _, d := range chain {
		mark := " "
		if d.version == active {
			mark = "▶"
		}
		extra := ""
		if d.status == PluginWaiting {
			extra = fmt.Sprintf(" waiting 缺服务: %v", d.waitingFor)
		} else if d.status == PluginFailed || d.status == PluginRejected {
			extra = fmt.Sprintf(" %s: %s", d.status, truncateStr(d.lastError, 60))
		}
		fmt.Fprintf(&sb, "  %s %s %s [%s] %s%s\n", mark, d.version, d.id, d.status, d.createdAt.Format("01-02 15:04"), extra)
	}
	if wantVersion == "" {
		sb.WriteString("\n源码与诊断：cordis_inspect id=" + pluginId + " version=<版本号>\n")
	}
	return sb.String()
}

// renderPluginRecord 渲染单个插件详情。
func renderPluginRecord(r PluginRecord) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s\n", r.Name))
	sb.WriteString(fmt.Sprintf("- source: %s\n- state: %s\n", r.Source, r.State))
	if r.PluginID != "" {
		sb.WriteString(fmt.Sprintf("- pluginId: %s（dyn id: %s, pkg: %s）\n", r.PluginID, r.DefID, r.PkgID))
	}
	if r.Version != "" {
		sb.WriteString(fmt.Sprintf("- version: %s\n", r.Version))
	}
	if r.Versions > 1 {
		sb.WriteString(fmt.Sprintf("- 累计版本: %d\n", r.Versions))
	}
	if len(r.WaitingFor) > 0 {
		sb.WriteString(fmt.Sprintf("- waitingFor: %s\n", strings.Join(r.WaitingFor, ", ")))
	}
	if r.LastError != "" {
		sb.WriteString(fmt.Sprintf("- lastError: %s\n", truncateStr(r.LastError, 200)))
	}
	if len(r.Diag) > 0 {
		sb.WriteString("- diag:\n")
		for _, line := range r.Diag {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
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

// renderJSDefDetail 渲染 JS 定义详情（L3 源码层：含代码预览与完整诊断）。
func renderJSDefDetail(d *jsPluginDef) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s（%s）\n", d.id, d.version))
	form := "对象形态"
	if d.isFunc {
		form = "函数形态"
	}
	sb.WriteString(fmt.Sprintf("- name: %s\n- pluginId: %s\n- pkg: %s\n- purpose: %s\n- status: %s\n- form: %s\n- createdAt: %s\n",
		d.name, d.pluginId, d.packageId, d.purpose, d.status, form, d.createdAt.Format(time.RFC3339)))
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
	if len(d.waitingFor) > 0 {
		sb.WriteString(fmt.Sprintf("- waitingFor: %s\n", strings.Join(d.waitingFor, ", ")))
	}
	if d.lastError != "" {
		sb.WriteString(fmt.Sprintf("- lastError: %s\n", d.lastError))
	}
	if len(d.diag) > 0 {
		sb.WriteString("- diag:\n")
		for _, line := range d.diag {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}
	preview := d.code
	if len(preview) > 1200 {
		preview = preview[:1200] + "\n…（截断）"
	}
	sb.WriteString("\n```js\n" + preview + "\n```\n")
	return sb.String()
}

// ─── 动态插件 → 工作区工具集同步 ───────────────────────────

// extractJSPluginName 从转译后的插件代码静态提取插件名：
//   - 对象形态 return { name: 'xxx', apply(ctx) {...} } → name 字符串字面量
//   - 函数形态 return (ctx, config) => {...} / function name(ctx) → 函数名（匿名返回空）
//
// 提取失败返回空串（调用方 fallback 到 dyn id）。
func extractJSPluginName(jsCode string) string {
	s := strings.TrimSpace(jsCode)
	// 对象形态：name: 'xxx' / name: "xxx"
	if i := strings.Index(s, "name"); i >= 0 {
		rest := s[i+4:]
		rest = strings.TrimSpace(rest)
		if strings.HasPrefix(rest, ":") {
			rest = strings.TrimSpace(rest[1:])
			if len(rest) > 1 && (rest[0] == '\'' || rest[0] == '"') {
				q := rest[0]
				for j := 1; j < len(rest); j++ {
					if rest[j] == q {
						name := rest[1:j]
						if name != "" {
							return name
						}
						break
					}
				}
			}
		}
	}
	// 函数形态：return function name(...) 或 return (ctx, config) =>（匿名）
	if i := strings.Index(s, "return function"); i >= 0 {
		rest := strings.TrimSpace(s[i+len("return function"):])
		j := 0
		for j < len(rest) && (rest[j] == '_' || rest[j] == '$' || rest[j] >= 'a' && rest[j] <= 'z' || rest[j] >= 'A' && rest[j] <= 'Z' || j > 0 && rest[j] >= '0' && rest[j] <= '9') {
			j++
		}
		if j > 0 {
			return rest[:j]
		}
	}
	return ""
}

// ★ cordis 动态插件一律固化到程序目录 <InstallDir>/.pair/plugins/dynamic.json
//
//	（插件是程序的扩展，不属于工作区；scope 仅记录生效作用域：global=UI 类
//	跨工作区 / project=工具插件）。
const dynamicToolsetName = "dynamic"

// syncDynamicPluginToToolset 把 cordis_define 登记的动态插件固化到全局插件文件
// <InstallDir>/.pair/plugins/dynamic.json（★ 插件是程序的扩展——一律安装在程序
// 所在目录，不存工作区；scope 仅记录生效作用域：global=UI 类跨工作区 /
// project=工具插件。工具集（.pair/toolsets/）只放 toolset_build 构建的命名工具集）。
// 只固化不装载（cordis_define 语义：登记）；cordis_run 装载后工具可用，重启自动装配。
func syncDynamicPluginToToolset(root string, def *jsPluginDef, name, scope string) (string, error) {
	if def == nil || root == "" {
		return "", os.ErrInvalid
	}
	entryName := name
	if entryName == "" {
		entryName = def.id // 匿名插件：以 dyn id 为条目名
	}
	s := scope
	if s == "" {
		s = "project"
	}
	if err := syncGlobalPlugin(ToolsetPlugin{
		Name:    entryName,
		Purpose: def.purpose,
		Code:    def.code,
		Client:  def.clientCode,
		Scope:   s,
	}); err != nil {
		return "", err
	}
	return fmt.Sprintf("已同步到全局插件包（程序目录 %s/%s/——package.json + index.js[+client.js]，重启自动装配，cordis_inspect 查看。★ 插件一律以插件包形式装在程序所在目录，不属于工作区；scope=%s）", GlobalPluginsPath(), entryName, s), nil
}
