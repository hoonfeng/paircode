// ═══════════════════════════════════════════════════════════════
// ext_sse.go — SSE 事件推送通道插件化：外部 SSE 路由注册表
//
// 背景（2026-08-16）：接口插件化（ext_routes.go）补齐了 HTTP 请求-响应
// 路由注册表；本文件补齐**实时推送**通道——插件可注册 SSE（Server-Sent
// Events）端点，向浏览器/外部客户端推送事件流（进度、日志、通知等），
// 宿主统一管理连接生命周期（SSE 头/Flush/断连检测/cleanup）。
//
// 与 HTTP 注册表（ext_routes）对齐：
//   - 精确 path 或前缀 path + "/*"
//   - 重复注册报错（路由是装配层契约）
//   - 注册返回 disposer（插件卸载自动回收）
//   - ExtSSEMiddleware 在 mux 之前拦截（SSE 端点优先，未命中走普通路由）
//
// JS 侧桥：ctx.sse.register(path, handler)（见 jsplugin.go buildSSEService）。
// handler(emit, params) 在连接建立时于 VM 锁内调用一次；emit(event, payload)
// 可跨调用推送（连接断开后返回错误）；handler 返回 cleanup（断开时调用）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// SSEHandler SSE 端点处理器：连接建立时调用一次。
//   - params：URL 查询参数（map[string]string）
//   - emit：推送函数（event 名 + payload，payload 会被 JSON 序列化）；
//     可在 handler 内或跨调用使用；连接断开后返回错误
//   - done：连接断开通道（客户端断开/服务器关闭时 close）
//
// 返回值：cleanup（连接断开时调用；可返回 nil）。
type SSEHandler func(params map[string]string, emit func(event string, payload any) error, done <-chan struct{}) func()

// extSSERoute 一条注册的 SSE 路由。
type extSSERoute struct {
	path   string // 绝对路径（无尾斜杠；以 "/*" 结尾=前缀匹配）
	prefix bool
	h      SSEHandler
}

var (
	extSSEMu sync.RWMutex
	extSSE   = map[string]extSSERoute{} // key: path
)

// RegisterExtSSE 注册一条外部 SSE 路由。重复 path 注册报错。
// path 以 "/*" 结尾表示前缀匹配（匹配 path 与 path/<anything>）。
// 返回 disposer（卸载路由）。
func RegisterExtSSE(path string, h SSEHandler) (func(), error) {
	if path == "" || h == nil {
		return nil, fmt.Errorf("SSE 路由注册: path/handler 不能为空")
	}
	prefix := false
	key := path
	if strings.HasSuffix(path, "/*") {
		prefix = true
		key = strings.TrimSuffix(path, "/*")
		key = strings.TrimSuffix(key, "/")
	}
	extSSEMu.Lock()
	defer extSSEMu.Unlock()
	if _, dup := extSSE[key]; dup {
		return nil, fmt.Errorf("SSE 路由 %s 重复注册（路由是装配层契约，冲突即配置错误）", key)
	}
	extSSE[key] = extSSERoute{path: key, prefix: prefix, h: h}
	return func() {
		extSSEMu.Lock()
		defer extSSEMu.Unlock()
		delete(extSSE, key)
	}, nil
}

// ServeExtSSE 尝试用 SSE 路由处理请求；命中返回 true（已写响应流）。
// 供 ExtSSEMiddleware 与测试直接调用。
func ServeExtSSE(w http.ResponseWriter, r *http.Request) bool {
	p := r.URL.Path
	extSSEMu.RLock()
	hit, ok := extSSE[p]
	if !ok {
		for _, rt := range extSSE {
			if rt.prefix && (p == rt.path || strings.HasPrefix(p, rt.path+"/")) {
				hit, ok = rt, true
				break
			}
		}
	}
	h := hit.h
	extSSEMu.RUnlock()
	if !ok || h == nil {
		return false
	}

	// SSE 响应头（禁缓冲，逐事件 flush）
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, canFlush := w.(http.Flusher)
	ctx := r.Context()

	// emit：写一条 SSE 事件并 flush；连接断开后返回错误
	emit := func(event string, payload any) error {
		select {
		case <-ctx.Done():
			return fmt.Errorf("SSE 连接已断开（%s）", ctx.Err())
		default:
		}
		var data string
		if payload == nil {
			data = ""
		} else if s, ok := payload.(string); ok {
			data = s
		} else if b, err := json.Marshal(payload); err == nil {
			data = string(b)
		} else {
			data = fmt.Sprint(payload)
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, strings.ReplaceAll(data, "\n", "\ndata: ")); err != nil {
			return err
		}
		if canFlush {
			flusher.Flush()
		}
		return nil
	}

	// 连接断开通道（客户端断开/服务器关闭）
	done := ctx.Done()

	// 调用处理器（宿主 Go 侧直接调；JS 桥在 VM 锁内调）
	var cleanup func()
	if h != nil {
		cleanup = h(queryParams(r), emit, done)
	}
	if cleanup != nil {
		defer cleanup()
	}
	<-done
	return true
}

// queryParams 提取 URL 查询参数（map[string]string，重复键取首个）。
func queryParams(r *http.Request) map[string]string {
	q := r.URL.Query()
	m := make(map[string]string, len(q))
	for k, vs := range q {
		if len(vs) > 0 {
			m[k] = vs[0]
		}
	}
	return m
}

// ExtSSEMiddleware 顶层处理器：SSE 端点优先，未命中交给 next（普通路由链）。
func ExtSSEMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ServeExtSSE(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}
