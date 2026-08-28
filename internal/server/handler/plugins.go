// 插件管理与使用共享 handler：/api/plugins*（web 端与桌面端共用）。
//
// 提供：
//   - GET  /api/plugins             插件列表（不含 client 半源码，省流量）
//   - GET  /api/plugins/detail      单插件详情（?id= 插件名或 dyn id，含 client 半源码）
//   - POST /api/plugins/action      启停/删除（{id, action: start|stop|undefine}）
//   - POST /api/plugins/define      直接定义 JS 动态插件（{purpose, code, client?, language?}）
//   - POST /api/plugins/event       浏览器 client 半 → host 事件桥（{event, payload}）
//   - GET  /api/plugins/client-events  host → 浏览器轮询（?since=seq，返回增量事件）
//   - POST/GET /api/plugins/client-state 浏览器上报 client 半快照 / 宿主读取（inspect 数据源）
//
// PluginHost 由各端（web_server / desktop）在启动时经 SetPluginHost 注入；
// 未注入（无插件系统）时列表返回空、写操作明确报错。
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/hoonfeng/paircode/internal/agent"
)

// PluginHost 全局插件宿主（浏览器 UI 与 REST 共用）。
var (
	PluginHost   *agent.PluginHost
	PluginHostMu sync.RWMutex
)

// SetPluginHost 由入口注入全局插件宿主（创建于启动初始化时）。
func SetPluginHost(ph *agent.PluginHost) {
	PluginHostMu.Lock()
	PluginHost = ph
	PluginHostMu.Unlock()
}

// GetPluginHost 取全局插件宿主（未注入返回 nil）。
func GetPluginHost() *agent.PluginHost {
	PluginHostMu.RLock()
	defer PluginHostMu.RUnlock()
	return PluginHost
}

// getPluginHost 取全局插件宿主（nil 时返回 nil,false）。
func getPluginHost() (*agent.PluginHost, bool) {
	PluginHostMu.RLock()
	defer PluginHostMu.RUnlock()
	return PluginHost, PluginHost != nil
}

// HandlePlugins GET /api/plugins：插件列表（含工具/服务/状态/client 有无，不含源码）。
func HandlePlugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonResp(w, []any{})
		return
	}
	recs := ph.Inspect()
	// 列表接口省略 client 半源码（详情接口按需取）
	var reg *agent.Registry
	if ph.Context() != nil {
		reg = ph.Context().Tools
	}
	out := make([]map[string]any, 0, len(recs))
	for _, rec := range recs {
		out = append(out, pluginRecordSummary(rec, reg))
	}
	jsonResp(w, out)
}

// HandlePluginDetail GET /api/plugins/detail?id=：单插件详情（含 client 半源码）。
func HandlePluginDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		jsonErr(w, "缺少 id 参数")
		return
	}
	name, err := ph.ResolvePluginName(id)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	rec := ph.InspectDetail(name)
	if rec == nil {
		jsonErr(w, "插件不存在: "+name)
		return
	}
	var reg *agent.Registry
	if ph.Context() != nil {
		reg = ph.Context().Tools
	}
	jsonResp(w, pluginRecordSummary(*rec, reg))
}

// HandlePluginAction POST /api/plugins/action：启停/删除插件。
func HandlePluginAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	var req struct {
		ID     string `json:"id"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		jsonErr(w, "缺少 id")
		return
	}
	name, err := ph.ResolvePluginName(req.ID)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	switch req.Action {
	case "start":
		if err := ph.Reload(name); err != nil {
			jsonErr(w, err.Error())
			return
		}
		// ★ 2026-08-2x：启动联动——有工具的磁盘插件加回工作区工具集
		//   （与 stop 移除对称；纯 UI 无工具插件不加，避免占位工具集）
		if root := pickWorkspaceRoot(r, ""); root != "" {
			if len(ph.PluginToolsByPlugin()[name]) > 0 {
				if n := agent.RestorePluginToToolsetsPublic(root, name); n > 0 {
					log.Printf("[plugin] 启动 %s 时已加回 %d 个工具集条目", name, n)
				}
			}
			// 重算可见性：新声明工具对 agent 可见（幂等）
			if ph.Context() != nil && ph.Context().Tools != nil {
				agent.ApplyToolsetVisibilityFilter(ph.Context().Tools, ph, root)
			}
		}
		jsonResp(w, map[string]any{"ok": true, "name": name, "state": "running"})
	case "stop":
		if err := ph.Unload(name); err != nil {
			jsonErr(w, err.Error())
			return
		}
		// ★ 2026-08-2x：停止插件联动工作区工具集——同名内嵌 code 条目移除并保存
		//   （未启用插件不再留在工具集；重启 installToolset 不会复活其声明）
		if root := pickWorkspaceRoot(r, ""); root != "" {
			if n := agent.RemovePluginFromToolsetsPublic(root, name); n > 0 {
				log.Printf("[plugin] 停止 %s 时已从 %d 个工具集移除内嵌条目", name, n)
			}
		}
		jsonResp(w, map[string]any{"ok": true, "name": name, "state": "stopped"})
	case "undefine":
		// ★ UndefinePermanent = 删内存定义 + 删磁盘插件包目录（.pair/plugins/<name>/），
		//   防重启 LoadGlobalPlugins 从磁盘包重新装配「复活」。
		if err := ph.UndefinePermanent(name); err != nil {
			jsonErr(w, err.Error())
			return
		}
		// ★ 联动：工具集中若有内嵌 code 的同名条目一并移除（防重启 installToolset 复活）
		if root := pickWorkspaceRoot(r, ""); root != "" {
			if n := agent.RemovePluginFromToolsetsPublic(root, name); n > 0 {
				log.Printf("[plugin] 删除定义 %s 时已从 %d 个工具集移除内嵌条目", name, n)
			}
		}
		jsonResp(w, map[string]any{"ok": true, "name": name, "state": "removed"})
	default:
		jsonErr(w, "action 必须是 start|stop|undefine，收到 "+req.Action)
	}
}

// HandlePluginDefine POST /api/plugins/define：直接定义 JS 动态插件（浏览器新建）。
// body: { purpose, code, client?, language?, run? }
//   - purpose: 用途说明（必填）
//   - code: host 半代码（必填）
//   - client: client 半代码（可选）
//   - language: "js"|"ts"（可选，自动探测）
//   - run: 定义后立即装载（可选，默认 true）
func HandlePluginDefine(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	var req struct {
		Purpose  string `json:"purpose"`
		Code     string `json:"code"`
		Client   string `json:"client"`
		Language string `json:"language"`
		Run      *bool  `json:"run"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "请求体解析失败: "+err.Error())
		return
	}
	id, err := ph.DefineJSCodeFull(req.Code, req.Language, req.Purpose, "", req.Client)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	run := true
	if req.Run != nil {
		run = *req.Run
	}
	state := "defined"
	pluginName := ""
	if run {
		def, _ := ph.GetJSDef(id)
		if err := ph.LoadJSDynamic(def); err != nil {
			jsonErr(w, "定义成功但装载失败: "+err.Error())
			return
		}
		state = "running"
		pluginName = def.Name()
	}
	jsonResp(w, map[string]any{"ok": true, "id": id, "name": pluginName, "state": state})
}

// HandlePluginEvent POST /api/plugins/event：浏览器 client 半 → host 事件桥。
// body: { event, payload }；事件经 EventBus 广播（host 插件用 ctx.on('host:xxx') 消费）。
func HandlePluginEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	var req struct {
		Event   string `json:"event"`
		Payload any    `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Event) == "" {
		jsonErr(w, "缺少 event")
		return
	}
	ph.EmitHostEvent(req.Event, req.Payload)
	jsonResp(w, map[string]any{"ok": true})
}

// HandlePluginClientEvents GET /api/plugins/client-events?since=seq：
// host → 浏览器事件轮询（client 半消费 ui:/client: 前缀事件）。
func HandlePluginClientEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonResp(w, map[string]any{"events": []any{}, "lastSeq": 0})
		return
	}
	var since int64
	if s := r.URL.Query().Get("since"); s != "" {
		since, _ = strconv.ParseInt(s, 10, 64)
	}
	events, lastSeq := ph.ClientEventsSince(since)
	jsonResp(w, map[string]any{"events": events, "lastSeq": lastSeq})
}

// HandlePluginClientState POST /api/plugins/client-state：浏览器 plugin-runtime
// 周期上报 client 半运行快照（client inspect provider 的数据源）。
// GET：宿主/agent 侧读取最新快照（调试用）。
func HandlePluginClientState(w http.ResponseWriter, r *http.Request) {
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	if r.Method == "POST" {
		var snap agent.ClientRuntimeSnapshot
		if err := json.NewDecoder(r.Body).Decode(&snap); err != nil {
			jsonErr(w, "快照解析失败: "+err.Error())
			return
		}
		if snap.Plugins == nil {
			snap.Plugins = []agent.ClientPluginSnapshot{}
		}
		if snap.Panels == nil {
			snap.Panels = []string{}
		}
		ph.SetClientState(snap)
		jsonResp(w, map[string]any{"ok": true})
		return
	}
	if r.Method == "GET" {
		jsonResp(w, ph.ClientState())
		return
	}
	jsonErr(w, "仅 POST/GET")
}

// HandlePluginInvoke POST /api/plugins/invoke：浏览器 client 半远程调用 host 半
// 注册的方法（D11 invoke RPC；对齐 harness @Remote('invoke')）。
// body: { plugin, method, args }；返回 { ok, value? } 或 { ok:false, error }。
func HandlePluginInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	var req struct {
		Plugin string `json:"plugin"`
		Method string `json:"method"`
		Args   any    `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Plugin) == "" || strings.TrimSpace(req.Method) == "" {
		jsonErr(w, "缺少 plugin/method")
		return
	}
	value, err := ph.InvokeClientMethod(req.Plugin, req.Method, req.Args)
	if err != nil {
		jsonResp(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	jsonResp(w, map[string]any{"ok": true, "value": value})
}

// HandlePluginClientFailure POST /api/plugins/client-failure：浏览器 client 半
// 失败上报（渲染/守卫/启动阶段；对齐 harness reportRenderFailure/
// reportClientGuardFailure）。记入定义诊断，Agent 经 cordis_inspect 发现修复。
// body: { plugin, phase: render|guard|boot, message }。
func HandlePluginClientFailure(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	var req struct {
		Plugin  string `json:"plugin"`
		Phase   string `json:"phase"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Plugin) == "" {
		jsonErr(w, "缺少 plugin")
		return
	}
	if err := ph.ReportClientFailure(req.Plugin, req.Phase, req.Message); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

// pluginRecordSummary 插件记录 → 前端摘要（隐藏超长 client 源码，仅保留有无标记）。
// reg 用于查询每个工具的启用状态（agent 可见性）；nil 时工具视为默认启用。
func pluginRecordSummary(rec agent.PluginRecord, reg *agent.Registry) map[string]any {
	toolStates := map[string]bool{}
	for _, t := range rec.Tools {
		if reg != nil {
			toolStates[t] = reg.IsEnabled(t)
		} else {
			toolStates[t] = true
		}
	}
	// ★ 对勾语义（2026-08-19）：toolCordisVisible 单独返回插件面板对勾状态
	//   （对 cordis 可见性），与 toolStates（agent 可见性/工具集）解耦。
	cordisVisible := map[string]bool{}
	if ph := GetPluginHost(); ph != nil {
		for _, t := range rec.Tools {
			cordisVisible[t] = ph.IsToolCordisVisible(t)
		}
	}
	return map[string]any{
		"name":              rec.Name,
		"source":            rec.Source,
		"scope":             rec.Scope,
		"state":             rec.State,
		"provides":          rec.Provides,
		"tools":             rec.Tools,
		"toolStates":        toolStates,
		"toolCordisVisible": cordisVisible,
		"sections":          rec.Sections,
		"version":           rec.Version,
		"purpose":           rec.Purpose,
		"hasClient":         rec.HasClient,
		"clientCode":        rec.ClientCode,
		"clientApproved":    rec.ClientApproved,
		"defId":             rec.DefID,
	}
}

// HandlePluginToolToggle POST /api/plugins/tool：通用工具级开关（任意已注册工具，
// 含内置工具与插件工具）。body: { tool, enabled:true|false } →
// Registry.SetToolEnabled（agent 可见性，运行时生效）。插件详情工具列表的
// 单个工具开关走此接口（区别于 /api/plugins/builtin 的内置工具包持久化开关）。
func HandlePluginToolToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	if ph.Context() == nil {
		jsonErr(w, "工具注册表未就绪")
		return
	}
	var req struct {
		Tool    string `json:"tool"`
		Enabled *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "请求体解析失败: "+err.Error())
		return
	}
	if req.Tool == "" || req.Enabled == nil {
		jsonErr(w, "需要 tool + enabled")
		return
	}
	reg := ph.Context().Tools
	if _, ok := reg.Get(req.Tool); !ok {
		jsonErr(w, "工具不存在: "+req.Tool)
		return
	}
	// ★ 对勾语义（2026-08-19）：/api/plugins/tool 控制「对 cordis 可见性」
	//   （JS 插件运行时 ctx.tools.list 能否看到该工具）；agent 可见性由
	//   工作区工具集（toolset_edit/工具集面板）独立决定，此处不再触碰 Enabled。
	ph.SetToolCordisVisible(req.Tool, *req.Enabled)
	state := "对 cordis 可见"
	if !*req.Enabled {
		state = "对 cordis 隐藏"
	}
	jsonResp(w, map[string]any{"ok": true, "message": state + " " + req.Tool})
}

// HandleBuiltinPlugins GET/POST /api/plugins/builtin：内置工具包（被过滤的 pair
// 独有工具按内置插件组管理——「放进插件面板」的载体）。
//   - GET：返回内置工具包完整信息（分组 + 工具 + 启用状态 + 已加入分组 + 强制全部后状态）
//   - POST：工具级开关 {tool, enabled:true|false}（手动添加/移除指定工具），
//     或分组开关 {group, enabled:true|false}（加入工作区/移出），
//     或强制全部加入 {forceAll:true}（所有内置组一次性启用）
func HandleBuiltinPlugins(w http.ResponseWriter, r *http.Request) {
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	root := pickWorkspaceRoot(r, "")
	if root == "" {
		jsonErr(w, "工作区未就绪")
		return
	}
	reg := (*agent.Registry)(nil)
	if ph.Context() != nil {
		reg = ph.Context().Tools
	}
	if r.Method == "GET" {
		// ★ 无工具集配置 → 先自动创建基础工具集（极简核心 + 框架本身提供的
		//   工具），管理弹窗/工具集面板才有据可依；有配置保持不动。
		if err := agent.EnsureWorkspaceToolsetPublic(ph, root); err != nil {
			log.Printf("[builtin] 自动生成基础工具集失败: %v", err)
		}
		jsonResp(w, agent.BuiltinToolsetInfoPublic(reg, ph, root))
		return
	}
	if r.Method != "POST" {
		jsonErr(w, "仅 GET/POST")
		return
	}
	var req struct {
		Group         string `json:"group"`
		Tool          string `json:"tool"`
		Enabled       *bool  `json:"enabled"`
		ForceAll      bool   `json:"forceAll"`
		WorkspaceRoot string `json:"workspaceRoot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "请求体解析失败: "+err.Error())
		return
	}
	if req.WorkspaceRoot != "" {
		root = req.WorkspaceRoot
	}
	if req.ForceAll {
		msg, err := agent.EnableAllBuiltinPublic(ph, root)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "message": msg})
		return
	}
	if req.Tool != "" {
		if req.Enabled == nil {
			jsonErr(w, "需要 tool + enabled")
			return
		}
		msg, err := agent.SetBuiltinToolEnabledPublic(ph, root, req.Tool, *req.Enabled)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "message": msg})
		return
	}
	if req.Group == "" || req.Enabled == nil {
		jsonErr(w, "需要 tool/group + enabled，或 forceAll=true")
		return
	}
	msg, err := agent.SetBuiltinGroupEnabledPublic(ph, root, req.Group, *req.Enabled)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "message": msg})
}
