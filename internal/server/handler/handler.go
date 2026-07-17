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
}

// Handle 注册 handler 到所有可用路由。
func (r *Router) Handle(method, pattern string, handler http.HandlerFunc) {
	if r.Mux != nil {
		r.Mux.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
			if req.Method != method {
				jsonErr(w, "不允许的方法: "+req.Method)
				return
			}
			handler(w, req)
		})
	}
	if r.Reg != nil {
		r.Reg.Register(method, pattern, bridge.HandlerFunc(handler))
	}
}

// NewRouter 创建统一路由器。
func NewRouter(mux *http.ServeMux, reg *bridge.Registry) *Router {
	return &Router{Mux: mux, Reg: reg}
}
