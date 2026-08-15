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
	"sort"
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
		Description: "登记一个 JS/TS 动态插件定义（语法预检，不运行）。code 是 async 函数体（host 半，宿主进程内运行），支持两种形态：① 对象形态 return { name, apply(ctx, config), inject? }；② 函数形态 return (ctx, config) => void（cordis 生态惯例，函数名作插件名）。apply 中可用 ctx.tools.register 注册工具、ctx.systemPrompt.section 贡献提示、ctx.on 监听事件、ctx.provide 提供服务；inject: ['fs','web','bash','logger','timer',...] 声明硬依赖（宿主缺失会明确报错）。可选 client 参数提供浏览器半代码（UI 侧运行，web 界面插件面板装载）：形态 (ui) => void，ui 提供 on/emit/registerPanel/http 等浏览器侧服务。TS 源码（含 interface/type 注解）由内置编译器自动转译。返回 dyn id 供 cordis_run/stop/undefine 使用。",
		Category:    "system",
		Parameters: objSchema(map[string]any{
			"code":     strProp("插件 host 半代码（JS 或 TS，async 函数体，return { name, apply(ctx, config), inject? } 或 return (ctx, config) => void）。可访问全局：ctx/harness/console/btoa/atob/TextEncoder/TextDecoder；inject 声明后 ctx.fs/web/bash/logger/timer 可用。"),
			"client":   strProp("可选：插件 client 半代码（浏览器端执行，web 界面插件面板装载）。形态 (ui) => void：ui.on(event, fn) 收 host 事件（ui:/client: 前缀）、ui.emit(event, payload) 发事件回 host（host: 前缀给 host 插件消费）、ui.registerPanel({id,title,icon,render}) 注册自定义面板、ui.http.get/post 调后端 API。缺省=纯 host 插件。"),
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
			clientCode := argStr(args, "client")
			dir := ""
			if d := strings.TrimSpace(argStr(args, "dir")); d != "" {
				resolved, err := resolvePathFor(root, args, d)
				if err != nil {
					return "", err
				}
				dir = resolved
			}
			id, err := host.DefineJSCodeFull(code, language, purpose, dir, clientCode)
			if err != nil {
				return "", err
			}
			extra := ""
			if dir != "" {
				extra = "，多文件 bundle（dir=" + dir + "）"
			}
			if strings.TrimSpace(clientCode) != "" {
				extra += "，含 client 半（浏览器 UI）"
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
	switch platform {
	case "host":
		return cordisHostQuery(host, provider, method, input)
	case "client":
		return cordisClientQuery(host, provider, method, input)
	default:
		return "", fmt.Errorf("platform 必须是 host 或 client，收到 %q", platform)
	}
}

// cordisClientQuery client 平台查询（数据源：浏览器 plugin-runtime 上报快照）。
func cordisClientQuery(host *PluginHost, provider, method string, input map[string]any) (string, error) {
	snap := host.ClientState()
	var sb strings.Builder
	offline := !snap.Connected
	header := "## Client runtime（浏览器插件 client 半，实时上报）\n"
	if offline {
		header += "⚠️ 浏览器未连接（页面未打开或 30s 内未上报）——以下为宿主侧装载清单兜底：\n"
	}
	sb.WriteString(header)
	name := ""
	if v, ok := input["name"].(string); ok {
		name = strings.TrimSpace(v)
	}
	now := time.Now().Unix()
	if snap.Connected && now-snap.ReportedAt <= clientStateTTL {
		fmt.Fprintf(&sb, "- 上报时间：%s 前（%d 秒前）\n", humanDur(now-snap.ReportedAt), now-snap.ReportedAt)
	}

	switch provider {
	case "plugin":
		switch method {
		case "listplugin", "list":
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
			recs := host.Inspect()
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
		case "getplugin", "get":
			if name == "" {
				return "", fmt.Errorf("getPlugin 需要 input={name}（client 半插件名）")
			}
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
			return "", fmt.Errorf("浏览器 client 半中未找到 %q（离线则参考宿主侧：%v）", name, hostHasClientHalf(host, name))
		default:
			return "", fmt.Errorf("plugin 平台方法必须是 listPlugin|getPlugin，收到 %q", method)
		}
	case "event":
		switch method {
		case "listevent", "list":
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
		default:
			return "", fmt.Errorf("event 平台方法必须是 listEvent，收到 %q", method)
		}
	case "service":
		// client 侧服务（http 路由等）暂未上报——给摘要
		sb.WriteString("client 半服务（ui.http 路由/数据源）尚未纳入上报协议；可用 event/plugin 平台查询面板与事件。\n")
		return sb.String(), nil
	case "tool":
		sb.WriteString("client 半工具（浏览器侧能力）尚未纳入上报协议；插件工具统一经 host 侧注册。\n")
		return sb.String(), nil
	default:
		return "", fmt.Errorf("provider 必须是 service|tool|event|plugin，收到 %q", provider)
	}
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

// cordisHostQuery host 平台查询。
func cordisHostQuery(host *PluginHost, provider, method string, input map[string]any) (string, error) {
	var sb strings.Builder
	switch provider {
	case "service":
		switch method {
		case "listservice", "list":
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
			host.ctx.servicesMu.RLock()
			for n := range host.ctx.services {
				dyn = append(dyn, n)
			}
			host.ctx.servicesMu.RUnlock()
			sort.Strings(dyn)
			if len(dyn) > 0 {
				sb.WriteString("\n## Dynamic services（ctx.provide/ctx.get）\n")
				for _, n := range dyn {
					fmt.Fprintf(&sb, "- %s\n", n)
				}
			}
			return sb.String(), nil
		case "getservice", "get":
			name := ""
			if v, ok := input["name"].(string); ok {
				name = v
			}
			if name == "" {
				return "", fmt.Errorf("getService 需要 input={name}（如 {name:\"fs\"}）")
			}
			return cordisServiceContract(host, name), nil
		default:
			return "", fmt.Errorf("service 平台方法必须是 listService|getService，收到 %q", method)
		}
	case "tool":
		meta := host.ctx.Tools.AllToolMeta()
		switch method {
		case "listtool", "list":
			sb.WriteString("## Tools（host 注册表；getTool {name} 取完整 schema）\n")
			for _, m := range meta {
				fmt.Fprintf(&sb, "- **%s**：%s\n", m.Name, firstLine(m.Description))
			}
			return sb.String(), nil
		case "gettool", "get":
			name := ""
			if v, ok := input["name"].(string); ok {
				name = v
			}
			if name == "" {
				return "", fmt.Errorf("getTool 需要 input={name}")
			}
			for _, m := range meta {
				if m.Name == name {
					b, _ := json.MarshalIndent(m, "", "  ")
					return string(b), nil
				}
			}
			return "", fmt.Errorf("工具 %q 不存在", name)
		default:
			return "", fmt.Errorf("tool 平台方法必须是 listTool|getTool，收到 %q", method)
		}
	case "event":
		switch method {
		case "listevent", "list":
			names := host.EventBus().EventNames()
			sb.WriteString("## Events（当前已注册事件；getEvent {name} 取详情）\n")
			if len(names) == 0 {
				sb.WriteString("（无事件）\n")
			}
			for _, n := range names {
				fmt.Fprintf(&sb, "- %s（%d 监听器）\n", n, host.EventBus().ListenerCount(n))
			}
			return sb.String(), nil
		case "getevent", "get":
			name := ""
			if v, ok := input["name"].(string); ok {
				name = v
			}
			if name == "" {
				return "", fmt.Errorf("getEvent 需要 input={name}")
			}
			n := host.EventBus().ListenerCount(name)
			sb.WriteString(fmt.Sprintf("## Event %s\n- 监听器数: %d\n", name, n))
			sb.WriteString("- 约定：ui:/client: 前缀事件会转发浏览器 client 半；host: 前缀由浏览器发回宿主。\n")
			return sb.String(), nil
		default:
			return "", fmt.Errorf("event 平台方法必须是 listEvent|getEvent，收到 %q", method)
		}
	case "plugin":
		switch method {
		case "listplugin", "list":
			sb.WriteString("## Plugins\n")
			for _, r := range host.Inspect() {
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
		case "getplugin", "get":
			name := ""
			if v, ok := input["name"].(string); ok {
				name = v
			}
			if name == "" {
				return "", fmt.Errorf("getPlugin 需要 input={name}")
			}
			rec := host.InspectDetail(name)
			if rec == nil {
				return "", fmt.Errorf("插件 %q 不存在", name)
			}
			return renderPluginRecord(*rec), nil
		default:
			return "", fmt.Errorf("plugin 平台方法必须是 listPlugin|getPlugin，收到 %q", method)
		}
	default:
		return "", fmt.Errorf("provider 必须是 service|tool|event|plugin，收到 %q", provider)
	}
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

