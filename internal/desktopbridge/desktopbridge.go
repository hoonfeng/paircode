// Package desktopbridge 提供桌面端的桥接初始化：加载核心配置、注册真实 API
// handler、注入 fetch 拦截（/api/* → 本地 Go handler）。desktop 主程序与
// dev/desktop_probe 探针共用，保证两者加载/启动方式完全一致。
package desktopbridge

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"wb-ui/bindings"
	"wb-ui/bridge"
	"wb-ui/jsc"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/agenttools"
	pairBridge "github.com/hoonfeng/paircode/internal/bridge"
	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/server/handler"
	mcppanel "github.com/hoonfeng/paircode/internal/ui/mcp"
	"github.com/hoonfeng/paircode/internal/ui/skills"
)

var bridgeRegistry = pairBridge.NewRegistry()
var bridgeSessionManager = agent.NewSessionManager()

// Init 初始化桌面端桥接：
//  1. 加载核心配置（core.Load + LoadLastProject），对齐 web 端
//  2. 初始化消息存储（JSONL + SQLite + persist worker）
//  3. 注册全部真实 API handler（internal/server/handler.RegisterAll，与 web 端共享逻辑）
//  4. 注入 JS：__DESKTOP_MODE__ / desktopBridge / fetch 拦截 / WebSocket stub
//  5. 启动 agent 事件推送（SubscribeAll → EvalJS onAgentEvent / onStatus）
func Init(wv *webkit.WebView) {
	rt := wv.JSInterpreter()
	if rt == nil {
		log.Printf("[Bridge] JSInterpreter 为 nil")
		return
	}
	log.Printf("[Bridge] 初始化桌面端桥接...")

	// ── 核心配置（对齐 cmd/companion InitCore） ──
	core.Load()
	core.LoadLastProject()
	if !core.Loaded {
		log.Println("[Bridge] 未发现已有配置，将使用默认设置。")
	}
	log.Printf("[Bridge] 工作区: %s (%d 个文件夹)", core.ProjectName(), len(core.Folders))
	log.Printf("[Bridge] API 已配置: %v", core.Configured())

	// ── 消息存储（会话/对话列表/消息记录真实数据） ──
	if root := core.Root(); root != "" {
		bridgeSessionManager.SetWorkspaceRoot(root)
		log.Printf("[Bridge] 消息存储已初始化于: %s", root)
	}

	registerHandlers()

	bridge.Register("/bridge/call", func(args []jsc.JSValue) (jsc.JSValue, error) {
		if len(args) < 3 {
			return jsc.StringValue(`{"status":400,"body":"{\"error\":\"param missing\"}"}`), nil
		}
		method := args[0].ToString()
		path := args[1].ToString()
		bodyJSON := args[2].ToString()
		paramsJSON := ""
		if len(args) > 3 {
			paramsJSON = args[3].ToString()
		}
		return jsc.StringValue(handleBridgeCall(method, path, bodyJSON, paramsJSON)), nil
	})

	bridge.InjectAll(rt)

	// ★ localStorage 文件持久化：前端 UI 状态（主题/打开文件等）重启不丢失。
	//   web 端浏览器 localStorage 天然持久；desktop 内存版重启丢状态，
	//   这里注入 JSON 文件后端（.pair/ui-state.json）对齐浏览器语义。
	if root := core.Root(); root != "" {
		bindings.SetLocalStoragePersist(newFileLocalStorage(filepath.Join(root, ".pair", "ui-state.json")))
		log.Printf("[Bridge] localStorage 持久化: %s", filepath.Join(root, ".pair", "ui-state.json"))
	}

	// ★ window 对象在 LoadHTML 内部（RegisterDOMBindings）才创建，
	//   因此 fetch 拦截等 JS 注入必须通过 BeforePageScripts 钩子
	//   （DOM bindings 已注册、页面 script 尚未执行时调用）。
	webkit.BeforePageScripts = func(rt2 *jsc.Interpreter) {
		injectJSBridge(rt2)
	}

	go forwardAgentEvents(wv)

	log.Printf("[Bridge] 完成, 已注册 %d 个处理器", len(bridgeRegistry.AllRoutes()))
}

// registerHandlers 注册全部真实 handler 到 bridgeRegistry。
// 与 web 端 cmd/companion 共用 internal/server/handler 的实现（fs/workspace/git/chat/conversations/tokens...）。
func registerHandlers() {
	handler.AgentMgr = bridgeSessionManager
	handler.BuildLoopOpts = buildDesktopLoopOpts
	router := handler.NewRouter(nil, bridgeRegistry)
	handler.RegisterAll(router)
}

// injectJSBridge 注入桌面端 JS 环境：
//   - window.__DESKTOP_MODE__ = true（前端 SDK 检测开关）
//   - window.desktopBridge（sdk.js 直调通道，兼容保留）
//   - window.fetch 拦截：/api/* 请求转发到 go.bridge_call（本地 Go handler），其余放行
//   - window.WebSocket stub：/ws 连接不实际建连，消息由 Go 端 EvalJS 推送
//
// ★ panel-only（只加载右侧面板）不再在此注入：那是独立测试程序
//   （dev/desktop_probe/folded_probe.go 等）的需求，由调用方经
//   webkit.BeforePageScripts 自行注入 window.__DESKTOP_PANEL_MODE__ = true；
//   cmd/desktop 主程序保持完整 IDE 布局。
func injectJSBridge(rt *jsc.Interpreter) {
	rt.RunJS(`(function(){
		window.__DESKTOP_MODE__ = true;
		window.desktopBridge = {
			call: function(method, path, bodyJSON, paramsJSON) {
				try {
					var r = go.bridge_call(method, path||'', bodyJSON||'', paramsJSON||'');
					return Promise.resolve(r);
				} catch(e) {
					return Promise.reject('[Bridge] ' + (e.message||e));
				}
			},
			onAgentEvent: null,
			onStatus: null
		};

		// ── fetch 拦截：/api/* → 本地 Go handler（desktopBridge 语义） ──
		var _fetch = window.fetch;

		function desktopResponse(body, status) {
			var st = status || 200;
			return {
				ok: st >= 200 && st < 300,
				status: st,
				statusText: st === 200 ? 'OK' : ('HTTP ' + st),
				url: '',
				json: function() {
					return Promise.resolve().then(function() {
						if (!body) return null;
						return JSON.parse(body);
					});
				},
				text: function() {
					return Promise.resolve(body || '');
				},
				headers: {}
			};
		}

		window.fetch = function(url, options) {
			var u = (typeof url === 'string') ? url : (url && url.url) || String(url);
			// ★ Normalize absolute URLs (api.js builds full URLs via
			// new URL('/api/...', location.origin)): strip the origin so
			// /api/* requests are intercepted locally instead of going over
			// the network (goja's native fetch truncates/loses long JSON).
			var nu = u;
			if (nu.indexOf('://') >= 0) {
				try {
					var _u = new URL(nu);
					nu = _u.pathname + (nu.indexOf('?') >= 0 ? nu.substring(nu.indexOf('?')) : '');
				} catch (e) {}
			}
			var isApi = nu === '/api' || nu.indexOf('/api/') === 0;
			if (!isApi) {
				if (_fetch) return _fetch(url, options);
				return Promise.reject(new Error('desktop: fetch not available for ' + u));
			}
			var method = (options && options.method) || 'GET';
			var body = (options && options.body) || '';
			var qIdx = nu.indexOf('?');
			var path = qIdx >= 0 ? nu.substring(0, qIdx) : nu;
			var params = {};
			if (qIdx >= 0) {
				var qs = nu.substring(qIdx + 1);
				qs.split('&').forEach(function(pair) {
					var kv = pair.split('=');
					if (kv[0]) {
						var k = kv[0], v = kv.length > 1 ? kv[1] : '';
						try { k = decodeURIComponent(k); } catch(e) {}
						try { v = decodeURIComponent(v); } catch(e) {}
						params[k] = v;
					}
				});
			}
			try {
				var r = go.bridge_call(method, path, body || '', JSON.stringify(params));
				var parsed = JSON.parse(r); // {status, body}
				return Promise.resolve(desktopResponse(parsed.body, parsed.status));
			} catch(e) {
				return Promise.reject(new Error('[Bridge] ' + (e.message || e)));
			}
		};

		// ── WebSocket stub：不实际建连，onmessage 由 Go 端 EvalJS 推送 ──
		window.__desktopWS = null;
		window.WebSocket = function(url) {
			this.url = url;
			this.readyState = 0; // CONNECTING
			this.bufferedAmount = 0;
			this.extensions = '';
			this.protocol = '';
			this.binaryType = 'blob';
			var self = this;
			window.__desktopWS = this;
			setTimeout(function() {
				self.readyState = 1; // OPEN
				if (typeof self.onopen === 'function') self.onopen({});
			}, 0);
		};
		window.WebSocket.CONNECTING = 0;
		window.WebSocket.OPEN = 1;
		window.WebSocket.CLOSING = 2;
		window.WebSocket.CLOSED = 3;
		window.WebSocket.prototype.send = function(data) {
			// 桌面模式下忽略 send（前端心跳等），Go 端不接收
		};
		window.WebSocket.prototype.close = function() {
			var self = this;
			if (self.readyState === 3) return;
			self.readyState = 3;
			if (typeof self.onclose === 'function') self.onclose({code: 1000, reason: 'desktop'});
		};
		window.WebSocket.prototype.dispatchMessage = function(data) {
			var self = this;
			if (typeof self.onmessage === 'function') {
				self.onmessage({data: data});
			}
		};
		window.WebSocket.prototype.dispatchStatus = function(payload) {
			var self = this;
			if (typeof self.onmessage === 'function') {
				self.onmessage({data: JSON.stringify({type: 'status', runningConvs: payload.runningConvs, runningByWorkspace: payload.runningByWorkspace})});
			}
		};
	})()`)
}

// forwardAgentEvents 订阅全部 agent 会话事件并推送到前端：
//   - 每条 GlobalEvent → window.desktopBridge.onAgentEvent(convId, dataJSON)
//   - done/error 事件后追加 status 推送（对齐 web 端 WS 行为）
// 前端无 onAgentEvent 订阅时降级为通过 WebSocket stub 的 dispatchMessage 推送
// （兼容 companion 前端 initWebSocket 的 onmessage 通道）。
func forwardAgentEvents(wv *webkit.WebView) {
	if bridgeSessionManager == nil {
		return
	}
	ch := bridgeSessionManager.SubscribeAll()
	defer bridgeSessionManager.UnsubscribeAll(ch)

	pushStatus := func() {
		running := bridgeSessionManager.ListRunning()
		counts := make(map[string]int, 8)
		for _, id := range running {
			ws := bridgeSessionManager.GetWorkspaceRoot(id)
			if ws == "" {
				continue
			}
			counts[ws]++
		}
		payload, _ := json.Marshal(map[string]any{
			"runningConvs":       running,
			"runningByWorkspace": counts,
		})
		evalJSOnDesktop(wv, `(function(){
			var p = `+string(payload)+`;
			if (window.desktopBridge && typeof window.desktopBridge.onStatus === 'function') {
				window.desktopBridge.onStatus(JSON.stringify(p));
			}
			if (window.__desktopWS && typeof window.__desktopWS.dispatchStatus === 'function') {
				window.__desktopWS.dispatchStatus(p);
			}
		})()`)
	}

	for ge := range ch {
		data, _ := json.Marshal(ge.Event)
		convID := ge.ConvID
		evalJSOnDesktop(wv, `(function(){
			var convId = `+convJSON(convID)+`;
			var data = `+string(data)+`;
			if (window.desktopBridge && typeof window.desktopBridge.onAgentEvent === 'function') {
				try { window.desktopBridge.onAgentEvent(convId, JSON.stringify(data)); } catch(e) {}
			}
			if (window.__desktopWS && typeof window.__desktopWS.dispatchMessage === 'function') {
				try {
					window.__desktopWS.dispatchMessage(JSON.stringify({convId: convId, type: data.type, content: data.content, tool: data.tool, args: data.args, callId: data.callId, doneReason: data.doneReason}));
				} catch(e) {}
			}
		})()`)
		if ge.Event.Type == agent.EventDone || ge.Event.Type == agent.EventError {
			time.Sleep(50 * time.Millisecond)
			pushStatus()
		}
	}
}

func convJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// evalJSOnDesktop 在 WebView 的 JS 解释器上执行 JS。
// wv 为 nil 时静默跳过（窗口尚未创建时的安全兜底）。
func evalJSOnDesktop(wv *webkit.WebView, js string) {
	if wv == nil {
		return
	}
	rt := wv.JSInterpreter()
	if rt == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Bridge] EvalJS recover: %v", r)
		}
	}()
	rt.RunJS(js)
}

// buildDesktopLoopOpts 构建 agent.LoopOpts（桌面版）。
// 与 web 端 buildWebLoopOpts 对齐：Provider + 工具注册表 + 系统提示词 + 历史加载。
func buildDesktopLoopOpts(convID, message string, autonomous bool) agent.LoopOpts {
	prov := buildDesktopProvider()

	root := core.Root()
	agent.WorkspaceRoots = core.Folders
	if root != "" {
		agent.SkillProjectDir = filepath.Join(root, ".pair", "skills")
	}
	if sysDir := filepath.Join(core.ConfigDir(), "skills"); sysDir != "" {
		agent.SkillSystemDir = sysDir
	}
	agent.SkillEnabled = core.Settings.SkillEnabledOverrides
	agent.SkillStatusOverride = core.Settings.SkillStatusOverrides
	agent.MCPUserConfigPath = filepath.Join(core.ConfigDir(), "mcp.json")
	if root != "" {
		agent.MCPProjectConfigPath = filepath.Join(root, ".pair", "mcp.json")
	}

	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	agent.RegisterCommitMessageTool(reg)
	agenttools.RegisterManagementTools(reg, root)
	if cfgs := mcppanel.LoadConfigs(); len(cfgs) > 0 {
		agentCfgs := make([]agent.MCPServerConfig, len(cfgs))
		for i, c := range cfgs {
			agentCfgs[i] = agent.MCPServerConfig{Name: c.Name, Command: c.Command, Args: c.Args, Env: c.Env}
		}
		agent.RegisterMCPServers(reg, agentCfgs)
	}
	agent.LoadLuaTools(reg, filepath.Join(root, ".pair", "tools"))
	agent.SetCodeGraphDB(bridgeSessionManager.RawDB())
	agent.InitDebugLogger(root, 50)

	sys := agent.ComposeSystemPrompt(
		buildDesktopSystemStatic(),
		buildDesktopSystemDynamic(root),
	)
	if guide := reg.UsageGuideText(); guide != "" {
		sys += "\n\n" + guide
	}

	var history []agent.Message
	var summaries []string
	if convID != "" {
		if store := bridgeSessionManager.Store(); store != nil {
			if raw, err := store.LoadAll(convID); err == nil && raw != nil {
				history = make([]agent.Message, len(raw))
				copy(history, raw)
			}
			summaries, _ = store.LoadCompressedSummaries(convID)
		}
	}
	history = agent.TrimInterruptedHistory(history)
	originalHistory := make([]agent.Message, len(history))
	copy(originalHistory, history)

	if resumeCtx := agent.BuildResumeContext(convID, message, history, bridgeSessionManager.Store(), core.Folders); resumeCtx != "" {
		sys += "\n\n" + resumeCtx
	}
	history = agent.CondenseHistory(history)

	maxIter := core.Settings.MaxIterations
	if autonomous {
		if maxIter <= 0 {
			maxIter = 60
		} else {
			maxIter *= 2
		}
	}

	return agent.LoopOpts{
		Provider:            prov,
		Registry:            reg,
		System:              sys,
		MaxIterations:       maxIter,
		MaxContextTokens:    core.Settings.ContextMaxTokens,
		Compressor:          nil,
		History:             history,
		HistoryOriginal:     originalHistory,
		CompressedSummaries: summaries,
		Autonomous:          autonomous,
	}
}

func buildDesktopProvider() agent.Provider {
	s := core.Settings
	if s.APIKey == "" || s.BaseURL == "" {
		return nil
	}
	return &agent.OpenAIProvider{
		BaseURL:      s.BaseURL,
		APIKey:       s.APIKey,
		Model:        core.MainModel(),
		Temperature:  core.Temperature(),
		MaxTokens:    s.MaxTokens,
		ThinkingMode: s.ThinkingMode,
	}
}

func buildDesktopSystemStatic() string {
	var b strings.Builder
	b.WriteString("你是 PairCode IDE 桌面端的内置 Agent，运行在本地桌面环境中。")
	b.WriteString("你的任务是帮助用户完成代码编辑、文件操作、Git 管理、问题排查等开发工作。")
	b.WriteString("优先使用专用工具而非 run_command。写类操作需谨慎。")
	b.WriteString(skills.Prompt())
	return b.String()
}

func buildDesktopSystemDynamic(root string) string {
	var b strings.Builder
	b.WriteString(agent.LongTermMemoryPrompt())
	b.WriteString(agent.ProjectRules(root))
	if root != "" {
		b.WriteString(agent.ProjectKnowledge(root, 2500))
	}
	if len(core.Folders) > 0 {
		b.WriteString("\n\n# 项目环境")
		for i, f := range core.Folders {
			projName := filepath.Base(f)
			if i == 0 {
				b.WriteString(fmt.Sprintf("\n\n### %s（主项目）\n", projName))
			} else {
				b.WriteString(fmt.Sprintf("\n\n### %s\n", projName))
			}
			b.WriteString(fmt.Sprintf("> 路径: %s\n", f))
			projEnv := agent.ReadProjectEnv(f)
			if projEnv != "" {
				lines := strings.SplitN(projEnv, "\n", 2)
				if len(lines) > 1 && strings.HasPrefix(strings.TrimSpace(lines[0]), "# ") {
					b.WriteString(strings.TrimSpace(lines[1]) + "\n")
				} else {
					b.WriteString(projEnv + "\n")
				}
			} else {
				b.WriteString("（无环境配置）\n")
			}
		}
	}
	return b.String()
}

// ─── bridge call dispatch ──────────────────────────────────

func handleBridgeCall(method, path, bodyJSON, paramsJSON string) string {
	bodyReader := strings.NewReader(bodyJSON)
	httpReq, err := http.NewRequest(method, path, bodyReader)
	if err != nil {
		return errResp(400, "req failed: "+err.Error())
	}
	if bodyJSON != "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if paramsJSON != "" {
		var params map[string]string
		if json.Unmarshal([]byte(paramsJSON), &params) == nil {
			q := httpReq.URL.Query()
			for k, v := range params {
				q.Set(k, v)
			}
			httpReq.URL.RawQuery = q.Encode()
		}
	}
	vw := &virtRW{headers: http.Header{}, body: strings.Builder{}, status: 200}
	// ★ Dispatch 用纯路径匹配（路由注册无 query）——前端 go.bridge_call 可能
	// 传带 query 的 path（如 /api/conversations?workspace=...），须先剥离。
	dispatchPath := path
	if qIdx := strings.IndexByte(dispatchPath, '?'); qIdx >= 0 {
		dispatchPath = dispatchPath[:qIdx]
	}
	if !bridgeRegistry.Dispatch(method, dispatchPath, vw, httpReq) {
		return errResp(404, "no route: "+method+" "+dispatchPath)
	}
	return okResp(vw.status, vw.body.String())
}

type virtRW struct {
	headers http.Header
	body    strings.Builder
	status  int
}

func (v *virtRW) Header() http.Header         { return v.headers }
func (v *virtRW) Write(b []byte) (int, error) { return v.body.Write(b) }
func (v *virtRW) WriteHeader(s int)            { v.status = s }

func errResp(status int, msg string) string {
	b, _ := json.Marshal(map[string]interface{}{"status": status, "body": `{"error":"` + msg + `"}`})
	return string(b)
}
func okResp(status int, body string) string {
	b, _ := json.Marshal(map[string]interface{}{"status": status, "body": body})
	return string(b)
}

// ─── localStorage 文件持久化后端 ───────────────────────────

// fileLocalStorage 实现 bindings.LocalStoragePersist：JSON 文件后端。
// 文件格式：{"key":"value",...}；启动时全量载入，写操作同步落盘。
type fileLocalStorage struct {
	path  string
	mu    sync.Mutex
	cache map[string]string
}

func newFileLocalStorage(path string) *fileLocalStorage {
	fl := &fileLocalStorage{path: path}
	if data, err := os.ReadFile(path); err == nil {
		var m map[string]string
		if json.Unmarshal(data, &m) == nil {
			fl.cache = m
			return fl
		}
	}
	fl.cache = map[string]string{}
	return fl
}

func (f *fileLocalStorage) Load() map[string]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]string, len(f.cache))
	for k, v := range f.cache {
		out[k] = v
	}
	return out
}

func (f *fileLocalStorage) Save(key, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if value == "" {
		delete(f.cache, key)
	} else {
		f.cache[key] = value
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return
	}
	if data, err := json.Marshal(f.cache); err == nil {
		os.WriteFile(f.path, data, 0o644)
	}
}
