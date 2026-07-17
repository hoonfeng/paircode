// Package bridge 提供 Handler 注册表与双模式调度能力。
// Web 模式下注册表生成 http.ServeMux，桌面模式下导出为 JS 全局函数供 wb-ui bindings 调用。
package bridge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// HandlerFunc 统一业务处理函数签名（与 http.HandlerFunc 完全兼容）。
type HandlerFunc func(w http.ResponseWriter, r *http.Request)

// Route 描述一条路由注册条目。
type Route struct {
	Method  string       // GET / POST / PUT / DELETE
	Pattern string       // 路由模式，如 "/api/fs/list"
	Handler HandlerFunc
}

// Registry 是线程安全的 Handler 注册表。
// 相同 path 下不同 method 可注册不同的 Handler。
type Registry struct {
	mu     sync.RWMutex
	routes []Route
	// path -> method -> handler 快速索引
	index map[string]map[string]HandlerFunc
}

// NewRegistry 创建空的注册表。
func NewRegistry() *Registry {
	return &Registry{
		index: make(map[string]map[string]HandlerFunc),
	}
}

// Register 注册一个 Handler。
// method 为 "GET"/"POST"/"PUT"/"DELETE" 等 HTTP 方法。
// pattern 为路由路径，如 "/api/fs/list"。
func (r *Registry) Register(method, pattern string, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes = append(r.routes, Route{
		Method:  strings.ToUpper(method),
		Pattern: pattern,
		Handler: handler,
	})

	if r.index[pattern] == nil {
		r.index[pattern] = make(map[string]HandlerFunc)
	}
	r.index[pattern][strings.ToUpper(method)] = handler
}

// Dispatch 根据 method + path 匹配并执行 Handler。
// 匹配策略：先精确匹配 path，再尝试最长前缀匹配。
// 返回 true 表示已处理；false 表示无匹配 handler。
func (r *Registry) Dispatch(method, path string, w http.ResponseWriter, req *http.Request) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. 精确匹配
	if handlers, ok := r.index[path]; ok {
		if h, ok := handlers[strings.ToUpper(method)]; ok {
			h(w, req)
			return true
		}
	}

	// 2. 前缀匹配（处理 /api/conversations/{id}/messages 等带参数的路径）
	//    从最长路径开始匹配
	for pattern, handlers := range r.index {
		if strings.HasPrefix(path, pattern) || strings.HasPrefix(pattern, path) {
			// 较长的 pattern 优先匹配
			continue
		}
		if strings.HasPrefix(path, pattern) {
			if h, ok := handlers[strings.ToUpper(method)]; ok {
				h(w, req)
				return true
			}
		}
	}

	return false
}

// AllRoutes 返回所有已注册的路由副本（线程安全）。
func (r *Registry) AllRoutes() []Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make([]Route, len(r.routes))
	copy(routes, r.routes)
	return routes
}

// ServeHTTPMux 将注册表转换为 http.ServeMux。
// 适用于 Web 模式：所有 Handler 注册到标准 HTTP 路由。
// 注意：相同 path 不同 method 将注册到同一个 HandleFunc，由 Handler 内部判断。
func (r *Registry) ServeHTTPMux() *http.ServeMux {
	mux := http.NewServeMux()

	// 按 path 分组后注册
	grouped := make(map[string]map[string]HandlerFunc)
	for _, route := range r.AllRoutes() {
		if grouped[route.Pattern] == nil {
			grouped[route.Pattern] = make(map[string]HandlerFunc)
		}
		grouped[route.Pattern][route.Method] = route.Handler
	}

	for pattern, handlers := range grouped {
		pattern := pattern // capture
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			if h, ok := handlers[strings.ToUpper(r.Method)]; ok {
				h(w, r)
				return
			}
			http.Error(w, fmt.Sprintf("不允许的方法: %s (路径: %s)", r.Method, pattern), http.StatusMethodNotAllowed)
		})
	}

	return mux
}

// ─── 桌面桥接导出 ──────────────────────────────────────────

// BridgeCallRequest 是桌面模式前端传过来的调用请求结构。
// 对应 sdk.js 中 desktopBridge.call(method, path, bodyJSON, paramsJSON) 的序列化参数。
type BridgeCallRequest struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Body   string `json:"body"`   // JSON 字符串，空串表示无 body
	Params string `json:"params"` // JSON 字符串，空串表示无 params
}

// BridgeCallResponse 是桌面模式调用后的响应结构。
type BridgeCallResponse struct {
	Status  int             `json:"status"`
	Body    string          `json:"body"` // JSON 字符串
	Headers map[string]string `json:"headers,omitempty"`
}

// HandleBridgeCall 是桌面桥接的统一调用入口。
// 接收 BridgeCallRequest，构造虚拟的 http.Request 和 http.ResponseWriter，分发到对应 Handler。
// 返回 BridgeCallResponse 的 JSON 序列化字符串。
func (r *Registry) HandleBridgeCall(callReqJSON string) string {
	var req BridgeCallRequest
	if err := json.Unmarshal([]byte(callReqJSON), &req); err != nil {
		return r.errorResponse(400, "非法请求: "+err.Error())
	}

	// 构造虚拟 HTTP 请求
	bodyReader := strings.NewReader(req.Body)
	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return r.errorResponse(400, "构造请求失败: "+err.Error())
	}
	if req.Body != "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// 解析 params 作为 Query 参数
	if req.Params != "" {
		var params map[string]string
		if err := json.Unmarshal([]byte(req.Params), &params); err == nil {
			q := httpReq.URL.Query()
			for k, v := range params {
				q.Set(k, v)
			}
			httpReq.URL.RawQuery = q.Encode()
		}
	}

	// 构造虚拟 ResponseWriter
	vw := &virtualResponse{
		header: http.Header{},
		buf:    strings.Builder{},
		status: 200,
	}

	// 分发
	if !r.Dispatch(req.Method, req.Path, vw, httpReq) {
		return r.errorResponse(404, fmt.Sprintf("无匹配路由: %s %s", req.Method, req.Path))
	}

	return r.successResponse(vw.status, vw.buf.String())
}

// ─── 虚拟 ResponseWriter ────────────────────────────────────

type virtualResponse struct {
	header http.Header
	buf    strings.Builder
	status int
}

func (v *virtualResponse) Header() http.Header         { return v.header }
func (v *virtualResponse) Write(b []byte) (int, error) { return v.buf.Write(b) }
func (v *virtualResponse) WriteHeader(statusCode int)   { v.status = statusCode }

// ─── 辅助 ──────────────────────────────────────────────────

func (r *Registry) errorResponse(status int, msg string) string {
	resp := BridgeCallResponse{
		Status: status,
		Body:   fmt.Sprintf(`{"error":"%s"}`, msg),
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func (r *Registry) successResponse(status int, body string) string {
	resp := BridgeCallResponse{
		Status: status,
		Body:   body,
	}
	b, _ := json.Marshal(resp)
	return string(b)
}
