// 插件管理与使用共享 handler：/api/plugins*（web 端与桌面端共用）。
//
// 提供：
//   - GET  /api/plugins             插件列表（不含 client 半源码，省流量）
//   - GET  /api/plugins/detail      单插件详情（?id= 插件名或 dyn id，含 client 半源码）
//   - POST /api/plugins/action      启停/删除（{id, action: start|stop|undefine}）
//   - POST /api/plugins/define      直接定义 JS 动态插件（{purpose, code, client?, language?}）
//   - POST /api/plugins/event       浏览器 client 半 → host 事件桥（{event, payload}）
//   - GET  /api/plugins/client-events  host → 浏览器轮询（?since=seq，返回增量事件）
//
// PluginHost 由各端（web_server / desktop）在启动时经 SetPluginHost 注入；
// 未注入（无插件系统）时列表返回空、写操作明确报错。
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/hoonfeng/paircode/internal/agent"
)

// PluginHost 全局插件宿主（浏览器 UI 与 REST 共用）。
var (
	PluginHost     *agent.PluginHost
	PluginHostMu   sync.RWMutex
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
	out := make([]map[string]any, 0, len(recs))
	for _, rec := range recs {
		out = append(out, pluginRecordSummary(rec))
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
	jsonResp(w, pluginRecordSummary(*rec))
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
		jsonResp(w, map[string]any{"ok": true, "name": name, "state": "running"})
	case "stop":
		if err := ph.Unload(name); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "name": name, "state": "stopped"})
	case "undefine":
		if err := ph.Undefine(name); err != nil {
			jsonErr(w, err.Error())
			return
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

// pluginRecordSummary 插件记录 → 前端摘要（隐藏超长 client 源码，仅保留有无标记）。
func pluginRecordSummary(rec agent.PluginRecord) map[string]any {
	return map[string]any{
		"name":      rec.Name,
		"source":    rec.Source,
		"state":     rec.State,
		"provides":  rec.Provides,
		"tools":     rec.Tools,
		"sections":  rec.Sections,
		"version":   rec.Version,
		"purpose":   rec.Purpose,
		"hasClient": rec.HasClient,
		"clientCode": rec.ClientCode,
		"defId":     rec.DefID,
	}
}
