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
	"sync/atomic"
	"time"

	"wb-ui/bindings"
	"wb-ui/bridge"
	"wb-ui/jsc"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/agent"
	pairBridge "github.com/hoonfeng/paircode/internal/bridge"
	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/pty"
	"github.com/hoonfeng/paircode/internal/server/handler"
	mcppanel "github.com/hoonfeng/paircode/internal/ui/mcp"
	"github.com/hoonfeng/paircode/internal/ui/skills"
)

var bridgeRegistry = pairBridge.NewRegistry()
var bridgeSessionManager = agent.NewSessionManager()

// isWsSwitchProbe 临时调试开关：打印每个 bridge_call 的耗时（工作区切换性能定位）。
// 通过环境变量 WS_SWITCH_TIMING=1 开启，默认关闭不影响正常运行。
var isWsSwitchProbe = os.Getenv("WS_SWITCH_TIMING") == "1"

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

	// ★ 启动时初始化参考工具注册表（对齐 web 端 startWebUI），
	// 供 /api/tools 查询工具列表与状态——桌面端此前缺失这三个路由，
	// 工具配置弹窗 GET /tools → 404 → 「加载失败，请重试」。
	if root := core.Root(); root != "" {
		initReg := agent.NewRegistry()
		agent.RegisterHostFrameworkTools(initReg, root)
		agent.RegisterCommitMessageTool(initReg)
		// 参考注册表也加载管理工具（多项目），保证 /api/tools 工具面板可见
		handler.SetToolsRegistry(initReg)
		log.Printf("[Bridge] 参考工具注册表已初始化（%d 个工具）", len(initReg.AllToolMeta()))
	}

	registerHandlers()

	bridge.InjectAll(rt)

	// ★ 终端 PTY 桥接：前端 TerminalPanel 的 WebSocket('/api/terminal/ws')
	// 经 stub 转接 → Go 侧真实 ConPTY 会话（复用 companion 的 internal/pty）。
	// 键盘输入/控制消息经 __desktopTerminalSend 进来，PTY 输出经
	// evalJSOnDesktop 推到 __desktopTerminals[id].onmessage。
	registerTerminalBridge(wv, rt)

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

// registerHandlers 注册全部真实 handler：
//  1. 注册到内部 bridgeRegistry（保留，供 web 模式 /api/* 与 debug 共用）
//  2. ★ 转注册到 wb-ui 原生 bridge（bridge.RegisterHTTP）：前端 fetch('/api/xxx')
//     被 wb-ui 的 page.RegisterFetch / SDK 拦截后**直接调 Go handler**——不再走
//     JS 拦截器 + go.bridge_call 统包 + 虚拟 HTTP 分发，实现真正两层交互。
func registerHandlers() {
	handler.AgentMgr = bridgeSessionManager
	handler.BuildLoopOpts = buildDesktopLoopOpts
	router := handler.NewRouter(nil, bridgeRegistry)
	handler.RegisterAll(router)

	// 转注册：method + pattern + http.HandlerFunc 直接注册到 wb-ui bridge。
	// wb-ui 侧 dispatchHTTP 会把 fetch 的 url/options 装配成 *http.Request 再
	// 调 handler，Go 侧无需任何自定义请求/响应解析。
	for _, r := range bridgeRegistry.AllRoutes() {
		h := http.HandlerFunc(r.Handler)
		if isWsSwitchProbe {
			// ★ WS_SWITCH_TIMING=1：打印每个耗时 >5ms 的 handler 调用，
			//   定位「启动 22 秒」里同步 bridge_call（/api/settings、/api/health、
			//   /api/conversations 等）的慢 handler。
			pattern := r.Pattern
			h = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				t0 := time.Now()
				r.Handler(w, req)
				if d := time.Since(t0); d > 5*time.Millisecond {
					log.Printf("[WS-TIMING] %s %s = %.2fms", req.Method, pattern, float64(d.Microseconds())/1000)
				}
			})
		}
		bridge.RegisterHTTP(r.Method, r.Pattern, h)
	}
	log.Printf("[Bridge] 已转注册 %d 条路由到 wb-ui bridge（两层直调）", len(bridgeRegistry.AllRoutes()))
}

// injectJSBridge 注入桌面端 JS 环境：
//   - window.__DESKTOP_MODE__ = true（前端 SDK 检测开关）
//   - window.desktopBridge（sdk.js 直调通道，兼容保留）
//   - window.WebSocket stub：/ws 连接不实际建连，消息由 Go 端 EvalJS 推送
//
// ★ fetch 拦截**不再由这里注入**：/api/* → 本地 Go handler 由 wb-ui 原生
//   bridge（page.RegisterFetch / bridge.RegisterHTTP 两层直调）承担，前端
//   fetch('/api/xxx') 直接被引擎拦截并调用注册的 Go handler，无需任何 JS
//   包装层。这里只保留前端运行必需的环境标记与 WebSocket stub。
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
				// 兼容保留：直接调用 wb-ui 原生 fetch 拦截路径。
				try {
					var u = '/api' + (path.indexOf('/') === 0 ? path : '/' + path);
					var opts = { method: method || 'GET' };
					if (bodyJSON) { opts.body = bodyJSON; }
					return fetch(u, opts).then(function(r){ return r.text(); });
				} catch(e) {
					return Promise.reject('[Bridge] ' + (e.message||e));
				}
			},
			onAgentEvent: null,
			onStatus: null
		};

		// ── WebSocket stub：不实际建连，onmessage 由 Go 端 EvalJS 推送 ──
		window.__desktopWS = null;      // 兼容别名（历史）
		window.__desktopAgentWS = null; // ★ agent 事件专用通道（固定实例，
		                                //   不受终端 WS 影响——终端 WS 在
		                                //   agent WS 之后创建会覆盖旧通道）
		window.__desktopTerminals = {}; // 终端会话表：id → WebSocket 实例
		window.WebSocket = function(url) {
			this.url = url;
			this.readyState = 0; // CONNECTING
			this.bufferedAmount = 0;
			this.extensions = '';
			this.protocol = '';
			this.binaryType = 'blob';
			var self = this;
			this.__isTerminal = url.indexOf('/api/terminal/ws') >= 0;
			if (this.__isTerminal) {
				// ── 终端连接：Go 侧建立真实 PTY 桥接会话 ──
				// 会话在 __desktopTerminalOpen 中创建（等 init 消息后启动
				// shell），输出经 __desktopTerminals[id].onmessage 推送。
				this.__termId = window.__desktopTerminalOpen(url);
				window.__desktopTerminals[this.__termId] = this;
				setTimeout(function() {
					self.readyState = 1; // OPEN
					if (typeof self.onopen === 'function') self.onopen({});
				}, 0);
				return;
			}
			// ── 普通 WS（agent 事件等）：stub 自动 OPEN，注册为 agent 通道 ──
			window.__desktopWS = this;
			window.__desktopAgentWS = this;
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
			if (this.__isTerminal && typeof window.__desktopTerminalSend === 'function') {
				// 终端输入：Uint8Array（TextEncoder 键盘输入）→ 字节字符串；
				// 或 JSON 字符串（init/resize/close 控制消息）
				var s;
				if (data && typeof data === 'object' && typeof data.length === 'number' && !(typeof data === 'string')) {
					s = '';
					for (var i = 0; i < data.length; i++) s += String.fromCharCode(data[i]);
				} else {
					s = String(data);
				}
				window.__desktopTerminalSend(this.__termId, s);
				return;
			}
			// 桌面模式忽略非终端 send（心跳等）
		};
		window.WebSocket.prototype.close = function() {
			var self = this;
			if (self.readyState === 3) return;
			self.readyState = 3;
			if (self.__isTerminal && typeof window.__desktopTerminalSend === 'function') {
				window.__desktopTerminalSend(self.__termId, JSON.stringify({type:'close'}));
			}
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

		// ── Go 主动调 JS 的演示/接入点（wb-ui CallFunction）──
		// window.__desktopNotify(title, msg)：宿主（Go）侧经
		// wv.CallFunction("__desktopNotify", ...) 直调前端，弹一个纯 DOM
		// toast（不依赖 Vue，顺带验证引擎的 DOM 增删 + 定时器能力）。
		// 业务接入可按同一模式暴露任意全局函数（如通知中心、状态栏刷新、
		// 进度上报），Go 侧在事件点 CallFunction 驱动。
		window.__desktopNotify = function(title, msg) {
			try {
				var el = document.createElement('div');
				el.setAttribute('data-notify', '1');
				el.style.cssText = 'position:fixed;right:16px;bottom:16px;background:rgba(30,30,30,0.95);color:#fff;padding:10px 14px;border-radius:8px;font-size:12px;z-index:9999;max-width:340px;box-shadow:0 4px 16px rgba(0,0,0,.35);font-family:sans-serif;line-height:1.5';
				el.textContent = title + ': ' + msg;
				document.body.appendChild(el);
				setTimeout(function(){ if (el.parentNode) el.parentNode.removeChild(el); }, 4000);
				return 'notified';
			} catch(e) {
				return 'notify-failed: ' + (e && e.message || e);
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
			var ag = window.__desktopAgentWS || window.__desktopWS;
			if (ag && typeof ag.dispatchStatus === 'function') {
				ag.dispatchStatus(p);
			}
		})()`)
	}

	for ge := range ch {
		data, _ := json.Marshal(ge.Event)
		convID := ge.ConvID
		// ★ 主线程执行（goja 非线程安全）：投递队列，Host.OnFrame drain。
		enqueueJS(`(function(){
			var convId = `+convJSON(convID)+`;
			var data = `+string(data)+`;
			if (window.desktopBridge && typeof window.desktopBridge.onAgentEvent === 'function') {
				try { window.desktopBridge.onAgentEvent(convId, JSON.stringify(data)); } catch(e) {}
			}
			var ag = window.__desktopAgentWS || window.__desktopWS;
			if (ag && typeof ag.dispatchMessage === 'function') {
				try {
					ag.dispatchMessage(JSON.stringify({convId: convId, type: data.type, content: data.content, tool: data.tool, args: data.args, callId: data.callId, doneReason: data.doneReason}));
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

// ─── 主线程 JS 推送队列 ─────────────────────────────────────
// goja 非线程安全：任何 goroutine（PTY 读循环、agent 事件推送）都不能
// 直接 RunJS，必须把 JS 片段投递到 mainQueue，由 Host 主循环每帧
// （OnFrame → DrainMainQueue）在主线程执行。队列满时丢弃（避免阻塞
// PTY 读循环，输出流式本来允许丢帧）。
var mainQueue = make(chan string, 4096)

func enqueueJS(js string) {
	select {
	case mainQueue <- js:
	default:
	}
}

// PushMainJS 把一段 JS 投递到主循环队列，由 Host.OnFrame →
// DrainMainQueue 在主线程（goja 安全）执行。供 desktop 诊断/自动化
// probe（如 --probe-editor）跨 goroutine 安全注入脚本。
func PushMainJS(js string) {
	enqueueJS(js)
}

// DrainMainQueue 在主线程执行所有待推送 JS 片段（Host.OnFrame 调用）。
func DrainMainQueue(wv *webkit.WebView) {
	for {
		select {
		case js := <-mainQueue:
			evalJSOnDesktop(wv, js)
		default:
			return
		}
	}
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
	agent.RegisterHostFrameworkTools(reg, root)
	agent.RegisterCommitMessageTool(reg)
	if cfgs := mcppanel.LoadConfigs(); len(cfgs) > 0 {
		agentCfgs := make([]agent.MCPServerConfig, len(cfgs))
		for i, c := range cfgs {
			agentCfgs[i] = agent.MCPServerConfig{Name: c.Name, Command: c.Command, Args: c.Args, Env: c.Env}
		}
		agent.RegisterMCPServers(reg, agentCfgs)
	}
	agent.SetCodeGraphDB(bridgeSessionManager.RawDB())
	agent.InitDebugLogger(root, 50)
	// ★ 保存注册表引用，供 /api/tools 查询工具列表与状态（桌面端 tools API）
	handler.SetToolsRegistry(reg)

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
	// ★ 配置消费插件化：Provider 参数统一经装配点解析（存储基线 → 插件装配器覆盖）。
	cur := agent.ResolveProviderParams()
	if cur.APIKey == "" || cur.BaseURL == "" {
		return nil
	}
	return &agent.OpenAIProvider{
		BaseURL:      cur.BaseURL,
		APIKey:       cur.APIKey,
		Model:        cur.Model,
		Temperature:  cur.Temperature,
		MaxTokens:    cur.MaxTokens,
		ThinkingMode: cur.ThinkingMode,
	}
}

func buildDesktopSystemStatic() string {
	var b strings.Builder
	b.WriteString("你是 PairCode IDE 桌面端的内置 Agent，运行在本地桌面环境中。")
	b.WriteString("你的任务是帮助用户完成代码编辑、文件操作、Git 管理、问题排查等开发工作。")
	b.WriteString("优先使用专用工具完成操作，写类操作需谨慎。")
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

// ─── 桌面终端 PTY 桥接 ─────────────────────────────────────────
// 前端 TerminalPanel 用 new WebSocket('/api/terminal/ws') 连接后端 PTY。
// 桌面端 WebSocket 是 stub（不真实建连），这里把终端 WS 桥接到真实
// ConPTY 会话：
//
//	前端 stub（WebSocket 构造器） URL 含 /api/terminal/ws →
//	    __desktopTerminalOpen(url) → Go 创建会话占位（返回 termId）
//	前端 socket.send(JSON init) → __desktopTerminalSend(termId, data)
//	    → Go 解析 init{shell,cwd,cols,rows} → pty.Start 启动 shell
//	PTY 输出（VT 字节流）→ Go 读循环 → evalJSOnDesktop 推
//	    __desktopTerminals[termId].onmessage({data: 字符串})
//	键盘输入（Uint8Array→字节字符串）→ __desktopTerminalSend → pty.Write
//	前端 socket.send(JSON resize/close) → pty.Resize / pty.Close

var (
	termSeq    int64
	termSessMu sync.Mutex
	termSess   = map[string]*desktopTerm{}
)

type desktopTerm struct {
	termID string
	wv     *webkit.WebView
	mu     sync.Mutex
	p      pty.PTY
	closed bool
}

// registerTerminalBridge 注册 Go native 函数（__desktopTerminalOpen /
// __desktopTerminalSend），供前端 WebSocket stub 转接终端 I/O。
func registerTerminalBridge(wv *webkit.WebView, rt *jsc.Interpreter) {
	if rt == nil {
		return
	}
	gobj := rt.GlobalObject()
	gobj.Set("__desktopTerminalOpen", jsc.FunctionValue(jsc.NewNativeFunction("__desktopTerminalOpen",
		func(in *jsc.Interpreter, thisVal jsc.JSValue, args []jsc.JSValue) jsc.JSValue {
			id := fmt.Sprintf("t%d", atomic.AddInt64(&termSeq, 1))
			termSessMu.Lock()
			termSess[id] = &desktopTerm{termID: id, wv: wv}
			termSessMu.Unlock()
			return jsc.StringValue(id)
		}, 1)))
	gobj.Set("__desktopTerminalSend", jsc.FunctionValue(jsc.NewNativeFunction("__desktopTerminalSend",
		func(in *jsc.Interpreter, thisVal jsc.JSValue, args []jsc.JSValue) jsc.JSValue {
			if len(args) < 2 {
				return jsc.Undefined()
			}
			id := args[0].ToString()
			data := args[1].ToString()
			handleTerminalMessage(id, data)
			return jsc.Undefined()
		}, 2)))
}

// handleTerminalMessage 处理来自前端 stub 的终端消息：JSON 控制消息
// （init/resize/close）或原始键盘输入（字节字符串，每个 rune ∈ 0..255）。
func handleTerminalMessage(id, data string) {
	termSessMu.Lock()
	t := termSess[id]
	termSessMu.Unlock()
	if t == nil {
		return
	}
	// ★ JSON 控制消息判断用 trim 后的字符串；键盘输入必须保留原始
	// 字节（含 \r\n 回车等控制字符），不能 trim 后写入。
	trimmed := strings.TrimSpace(data)
	if strings.HasPrefix(trimmed, "{") {
		var msg struct {
			Type  string `json:"type"`
			Shell string `json:"shell"`
			Cwd   string `json:"cwd"`
			Cols  int    `json:"cols"`
			Rows  int    `json:"rows"`
		}
		if json.Unmarshal([]byte(trimmed), &msg) == nil {
			switch msg.Type {
			case "init":
				t.start(msg.Shell, msg.Cwd, msg.Cols, msg.Rows)
			case "resize":
				t.resize(msg.Cols, msg.Rows)
			case "close":
				t.close()
			}
			return
		}
	}
	// 键盘/原始输入：原样写入（含 \r\n 回车等控制字节）。
	t.write([]byte(data))
}

// start 按 init 消息启动 PTY 会话，并启动输出读循环。
func (t *desktopTerm) start(shellName, cwd string, cols, rows int) {
	t.mu.Lock()
	if t.closed || t.p != nil {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	sh := pty.ShellByName(shellName)
	if cols < 2 {
		cols = 80
	}
	if rows < 2 {
		rows = 24
	}
	p, err := pty.Start(sh, cwd, cols, rows)
	if err != nil {
		log.Printf("[Terminal] PTY 启动失败: %v", err)
		evalJSOnDesktop(t.wv, `(function(){
			var ws = window.__desktopTerminals && window.__desktopTerminals[`+convJSON(t.termID)+`];
			if (ws && typeof ws.onerror === 'function') ws.onerror({message: 'PTY 启动失败'});
		})()`)
		return
	}
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		p.Close()
		return
	}
	t.p = p
	t.mu.Unlock()

	go t.readLoop(p)
}

// readLoop 持续读取 PTY 输出并推送到前端对应终端实例的 onmessage。
func (t *desktopTerm) readLoop(p pty.PTY) {
	buf := make([]byte, 4096)
	for {
		n, err := p.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			if os.Getenv("WB_TERM_DEBUG") != "" {
				log.Printf("[term-io] OUT %d bytes: %x", n, buf[:n])
			}
			// ★ 主线程执行（goja 非线程安全）：投递到 mainQueue，
			// Host.OnFrame → DrainMainQueue 在主循环 RunJS。
			enqueueJS(`(function(){
				var ws = window.__desktopTerminals && window.__desktopTerminals[` + convJSON(t.termID) + `];
				if (ws && typeof ws.onmessage === 'function') ws.onmessage({data: ` + convJSON(chunk) + `});
			})()`)
		}
		if err != nil {
			break
		}
	}
	t.close()
}

func (t *desktopTerm) write(data []byte) {
	t.mu.Lock()
	p := t.p
	t.mu.Unlock()
	if p == nil {
		return
	}
	_, _ = p.Write(data)
}

func (t *desktopTerm) resize(cols, rows int) {
	t.mu.Lock()
	p := t.p
	t.mu.Unlock()
	if p == nil || cols < 2 || rows < 2 {
		return
	}
	_ = p.Resize(cols, rows)
}

func (t *desktopTerm) close() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	p := t.p
	t.p = nil
	t.mu.Unlock()

	termSessMu.Lock()
	delete(termSess, t.termID)
	termSessMu.Unlock()

	if p != nil {
		_ = p.Close()
	}
}
