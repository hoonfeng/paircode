// Web-only 版——仅注册平台回调 + 路由，所有 handler 逻辑在 web_server.go 统一实现。
//
//go:build windows

package main

import (
	"net/http"

	"github.com/hoonfeng/paircode/internal/agent"
)

func init() {
	// Web-only：规则式压缩（nil Compressor）
	webCompressor = func() agent.Compressor { return nil }
}

func registerExtraHandlers(mux *http.ServeMux, s *webServer) {
	mux.HandleFunc("/api/chat/send", s.handleChatSend)
	// WebSocket 端点由框架层处理（internal/agent）：/ws 全局事件流、
	// /api/terminal/ws PTY 终端桥——main 只保留注册（帧实现/连接管理
	// 全部下沉 internal/agent/wsconn.go + event_ws.go + terminal_ws.go）。
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		agent.ServeGlobalEventStreamWS(w, r, agentMgr)
	})
	mux.HandleFunc("/api/terminal/ws", agent.ServeTerminalWS)
	mux.HandleFunc("/api/chat/stop", s.handleChatStop)
	mux.HandleFunc("/api/chat/answer", s.handleChatAnswer)
	mux.HandleFunc("/api/chat/approve", s.handleChatApprove)
	mux.HandleFunc("/api/chat/feedback", s.handleChatFeedback)
	mux.HandleFunc("/api/chat/rollback", s.handleChatRollback)
	mux.HandleFunc("/api/chat/compact", s.handleChatCompact)
	mux.HandleFunc("/api/marketplace/search", s.handleMarketplaceSearch)
	mux.HandleFunc("/api/marketplace/install", s.handleMarketplaceInstall)
	mux.HandleFunc("/api/marketplace/uninstall", s.handleMarketplaceUninstall)
	mux.HandleFunc("/api/marketplace/refresh", s.handleMarketplaceRefresh)
	mux.HandleFunc("/api/memory/search", s.handleMemorySearch)
	mux.HandleFunc("/api/memory/list", s.handleMemoryList)
	mux.HandleFunc("/api/memory/rebuild", s.handleMemoryRebuild)
}
