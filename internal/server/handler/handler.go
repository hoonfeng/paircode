// Package handler — 共享 handler 实现
// 所有 handler 可作为独立函数注册到 http.ServeMux 或 bridge.Registry。
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/hoonfeng/paircode/internal/bridge"
	"github.com/hoonfeng/paircode/internal/agent"
)

// ─── 辅助函数 ──────────────────────────────────────────────

func jsonResp(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonErr(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ─── 全局依赖注入 ──────────────────────────────────────────
// 由入口（cmd/desktop/main.go）在运行前设置。
// 避免每个 handler 都通过 Context 传递，保持签名简洁。

// AgentMgr 全局会话管理器，用于 Chat handler。
var AgentMgr *agent.SessionManager

// BuildLoopOpts 构建 agent 循环参数的函数，各平台自行注入。
var BuildLoopOpts func(convID, message string, autonomous bool) agent.LoopOpts

// ─── 注册接口 ──────────────────────────────────────────────

// Router 是统一注册接口，同时注册到 http.ServeMux 和 bridge.Registry。
type Router struct {
	Mux *http.ServeMux   // Web 模式用
	Reg *bridge.Registry // 桌面模式用

	// muxHandlers 分组缓存：pattern -> method -> handler。
	// http.ServeMux 不允许同一 pattern 重复注册（会 panic），同 pattern 多方法
	// 必须合并为一次注册 + 按方法分发（对齐 bridge.Registry 的 path->method 索引）。
	muxHandlers map[string]map[string]http.HandlerFunc
}

// Handle 注册 handler 到所有可用路由。
// 同一 pattern 可多次调用注册不同方法（如 GET/PUT/DELETE 共享 handler 内部分支）。
func (r *Router) Handle(method, pattern string, handler http.HandlerFunc) {
	if r.Mux != nil {
		if r.muxHandlers == nil {
			r.muxHandlers = map[string]map[string]http.HandlerFunc{}
		}
		methods := r.muxHandlers[pattern]
		if methods == nil {
			methods = map[string]http.HandlerFunc{}
			r.muxHandlers[pattern] = methods
			// 首次注册该 pattern：挂一次分发器，后续 Handle 仅追加方法映射。
			r.Mux.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
				h, ok := r.muxHandlers[pattern][req.Method]
				if !ok {
					jsonErr(w, "不允许的方法: "+req.Method)
					return
				}
				h(w, req)
			})
		}
		methods[method] = handler
	}
	if r.Reg != nil {
		r.Reg.Register(method, pattern, bridge.HandlerFunc(handler))
	}
}

// NewRouter 创建统一路由器。
func NewRouter(mux *http.ServeMux, reg *bridge.Registry) *Router {
	return &Router{Mux: mux, Reg: reg}
}
