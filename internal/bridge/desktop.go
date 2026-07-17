// Package bridge 提供 Handler 注册表与双模式调度能力。
// 本文件实现桌面桥接：通过系统 WebView 加载前端页面，注入 desktopBridge 对象。
package bridge

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// DesktopBridge 管理桌面端桥接：窗口生命周期 + 前端 ↔ Go 双向调用。
//
// 工作方式：
//  1. 创建原生窗口（使用系统 WebView2 on Windows）
//  2. 加载 web-ui 构建产物（dist/index.html）
//  3. 页面加载前注入 window.desktopBridge JS 对象
//  4. 前端 sdk.js 通过 desktopBridge.call() 调 Go Handler
//  5. Go 通过 desktopBridge.onAgentEvent / onStatus 推送事件到前端
type DesktopBridge struct {
	reg    *Registry
	window interface { // 抽象的窗口接口，避免外部依赖
		Run() error
		Terminate()
		SetTitle(title string)
		Bind(name string, fn any) error
		Eval(js string) error
		Navigate(url string) error
	}

	mu         sync.Mutex
	running    bool
	eventCbs   map[string]func(string, string)
}

// NewDesktopBridge 创建桌面桥接实例。
// reg 为已注册好所有 Handler 的 Registry。
func NewDesktopBridge(reg *Registry) *DesktopBridge {
	return &DesktopBridge{
		reg:      reg,
		eventCbs: make(map[string]func(string, string)),
	}
}

// StartWindow 创建并显示桌面窗口。
// width/height 为 CSS 像素尺寸，title 为窗口标题。
// 内部启动系统 WebView，加载 web-ui 前端页面。
func (db *DesktopBridge) StartWindow(width, height int, title string) error {
	// TODO: 实际创建 WebView2 窗口
	// 当前为桩实现，返回错误提示用户尚未集成 WebView 库。
	// 
	// 集成方式（二选一）：
	//
	// 方案 A: github.com/webview/webview（轻量，跨平台）
	//   w := webview.New(false)  // false = 不启用 DevTools
	//   defer w.Destroy()
	//   w.SetTitle(title)
	//   w.SetSize(width, height, webview.HintNone)
	//   w.Bind("desktopBridge", db)  // 绑定 bridge 为 JS 全局对象
	//   w.Navigate("file:///path/to/web-ui/dist/index.html")
	//   w.Run()
	//
	// 方案 B: 内部 HTTP + WebView2（更可控）
	//   在随机端口启动本地 HTTP 服务器，提供 web-ui 静态文件
	//   + /api/bridge 端点处理桌面桥接调用
	//   WebView2 导航到 http://localhost:{port}
	//   注入 JS 脚本定义 desktopBridge.call = async (method, path, body, params) => {
	//     const r = await fetch('/api/bridge', { method: 'POST', body: JSON.stringify({...}) })
	//     return r.text()
	//   }

	return fmt.Errorf("DesktopBridge.StartWindow: WebView 集成尚未实现。请参考 internal/bridge/desktop.go 中的集成方案")
}

// Stop 关闭桌面窗口并释放资源。
func (db *DesktopBridge) Stop() {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.window != nil {
		db.window.Terminate()
	}
	db.running = false
	log.Println("[DesktopBridge] 已关闭")
}

// IsRunning 返回窗口是否运行中。
func (db *DesktopBridge) IsRunning() bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.running
}

// BridgeCall 是桌面桥接的核心调用入口（被 JS 端 desktopBridge.call 调用）。
// 接收 method/path/bodyJSON/paramsJSON，通过 Registry 分发到对应 Handler，
// 返回序列化的 BridgeCallResponse JSON。
//
// 方法签名设计为 Go 可被 webview.Bind 直接导出为 JS 函数：
//
//	func(method, path, bodyJSON, paramsJSON string) string
func (db *DesktopBridge) BridgeCall(method, path, bodyJSON, paramsJSON string) string {
	return db.reg.HandleBridgeCall(fmt.Sprintf(
		`{"method":"%s","path":"%s","body":%s,"params":%s}`,
		method, path,
		quoteJSON(bodyJSON),
		quoteJSON(paramsJSON),
	))
}

// PushAgentEvent 向 JS 端推送 agent 事件。
// convId: 对话 ID；data: 事件数据（将被 JSON 序列化）。
// 对应 sdk.js 中 bridge.onAgentEvent(convId, dataJSON) 回调。
func (db *DesktopBridge) PushAgentEvent(convId string, data any) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("[DesktopBridge] PushAgentEvent 序列化失败: %v", err)
		return
	}

	js := fmt.Sprintf(
		`if(window.desktopBridge && window.desktopBridge.onAgentEvent) window.desktopBridge.onAgentEvent(%s, %s)`,
		quoteJSON(convId), string(dataJSON),
	)

	db.evalJS(js)
}

// PushStatus 向 JS 端推送运行状态。
// 对应 sdk.js 中 bridge.onStatus(payloadJSON) 回调。
func (db *DesktopBridge) PushStatus(payload any) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[DesktopBridge] PushStatus 序列化失败: %v", err)
		return
	}

	js := fmt.Sprintf(
		`if(window.desktopBridge && window.desktopBridge.onStatus) window.desktopBridge.onStatus(%s)`,
		string(payloadJSON),
	)

	db.evalJS(js)
}

// evalJS 在 WebView 中执行 JS 代码。
func (db *DesktopBridge) evalJS(js string) {
	db.mu.Lock()
	win := db.window
	db.mu.Unlock()

	if win != nil {
		win.Eval(js)
	}
}

// ─── JS 桥接注入脚本 ──────────────────────────────────────

// BridgeInjectionScript 返回需要注入到前端页面的 JS 代码。
// 这段代码定义了 window.desktopBridge 对象及其 call 方法。
func BridgeInjectionScript() string {
	return `window.__DESKTOP_MODE__ = true;
window.desktopBridge = {
  // call 调用 Go 端的 BridgeCall (method, path, bodyJSON, paramsJSON) -> string
  call: function(method, path, bodyJSON, paramsJSON) {
    // 在 webview.Bind 方案中，bridgeCall 是 Go 函数直接绑到 JS 全局
    // 在 HTTP+WebView2 方案中，通过 fetch 请求 /api/bridge
    if(typeof bridgeCall === 'function') {
      return bridgeCall(method, path, bodyJSON || '', paramsJSON || '');
    }
    return JSON.stringify({status:503, body:'{"error":"bridgeCall 不可用"}'});
  },
  // 以下回调由 Go 端调用设置
  onAgentEvent: null,
  onStatus: null,
};`
}

// quoteJSON 给 JSON 字符串加引号（用于嵌入 JS 字符串字面量）。
func quoteJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
