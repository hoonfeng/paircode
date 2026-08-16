// ═══════════════════════════════════════════════════════════════
// ext_ws.go — 双向 WebSocket 通道插件化：外部 WS 路由注册表
//
// 背景（2026-08-16）：SSE（ext_sse.go）补齐了**单向推送**通道；本文件
// 补齐**双向实时**通道——插件可注册 WebSocket 端点，与浏览器/外部客户端
// 双向收发消息（终端桥、实时协作、自定义协议等）。ws 帧实现由
// wsconn.go 提供（RFC6455 最小实现，从 cmd/companion 抽离）。
//
// 与 ext_sse / ext_routes 对齐：
//   - 精确 path 或前缀 path + "/*"；重复注册报错；注册返回 disposer
//   - ExtWSMiddleware 在 mux 之前拦截（仅处理 Upgrade 请求，其余放行）
//   - 宿主自身端点（/ws 事件流、/api/terminal/ws PTY）经 mux 照常工作
//
// JS 侧桥：ctx.ws.register(path, handler)（见 jsplugin.go buildWSService）。
// handler(conn, params) 在连接建立后调用；conn.send(payload) 双向发送、
// conn.onMessage(fn) 注册消息回调（Go 读循环 → VM 锁内调 JS）、
// conn.close() 主动关闭；handler 返回 cleanup（断开时调用）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// WSHostHandler WebSocket 端点处理器：连接建立后调用（在 ServeExtWS 的
// goroutine 中）。
//   - conn：已升级的 WSConn（send/close 并发安全）
//   - params：URL 查询参数（map[string]string）
//   - done：连接生命周期信号——**handler 负责在连接结束时 close(done)**
//     （读循环错误/客户端 close 帧/服务端主动关闭时）
//
// 返回值：cleanup（连接断开时调用；可返回 nil）。
type WSHostHandler func(conn *WSConn, params map[string]string, done chan struct{}) func()

// extWSRoute 一条注册的 WS 路由。
type extWSRoute struct {
	path   string // 绝对路径（无尾斜杠；以 "/*" 结尾=前缀匹配）
	prefix bool
	h      WSHostHandler
}

var (
	extWSMu sync.RWMutex
	extWS   = map[string]extWSRoute{} // key: path
)

// RegisterExtWS 注册一条外部 WebSocket 路由。重复 path 注册报错。
// path 以 "/*" 结尾表示前缀匹配（匹配 path 与 path/<anything>）。
// 返回 disposer（卸载路由）。
func RegisterExtWS(path string, h WSHostHandler) (func(), error) {
	if path == "" || h == nil {
		return nil, fmt.Errorf("WS 路由注册: path/handler 不能为空")
	}
	prefix := false
	key := path
	if strings.HasSuffix(path, "/*") {
		prefix = true
		key = strings.TrimSuffix(path, "/*")
		key = strings.TrimSuffix(key, "/")
	}
	extWSMu.Lock()
	defer extWSMu.Unlock()
	if _, dup := extWS[key]; dup {
		return nil, fmt.Errorf("WS 路由 %s 重复注册（路由是装配层契约，冲突即配置错误）", key)
	}
	extWS[key] = extWSRoute{path: key, prefix: prefix, h: h}
	return func() {
		extWSMu.Lock()
		defer extWSMu.Unlock()
		delete(extWS, key)
	}, nil
}

// ServeExtWS 尝试用 WS 路由处理请求；命中返回 true（已接管连接）。
// 仅处理 Upgrade: websocket 请求；未命中或非 Upgrade 请求返回 false。
// 供 ExtWSMiddleware 与测试直接调用。
func ServeExtWS(w http.ResponseWriter, r *http.Request) bool {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return false
	}
	p := r.URL.Path
	extWSMu.RLock()
	hit, ok := extWS[p]
	if !ok {
		for _, rt := range extWS {
			if rt.prefix && (p == rt.path || strings.HasPrefix(p, rt.path+"/")) {
				hit, ok = rt, true
				break
			}
		}
	}
	h := hit.h
	extWSMu.RUnlock()
	if !ok || h == nil {
		return false
	}

	conn, err := UpgradeWS(w, r)
	if err != nil {
		// 命中插件路由但升级失败：upgradeWebSocket 已写 HTTP 错误响应
		return true
	}

	done := make(chan struct{})
	var cleanup func()
	if h != nil {
		cleanup = h(conn, queryParams(r), done)
	}
	if cleanup != nil {
		defer cleanup()
	}
	<-done
	return true
}

// ExtWSMiddleware 顶层处理器：WebSocket 端点优先（仅 Upgrade 请求），
// 未命中交给 next（SSE/HTTP 路由链）。
func ExtWSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ServeExtWS(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}
