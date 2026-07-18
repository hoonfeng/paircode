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
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/api/terminal/ws", s.handleTerminalWS)
	mux.HandleFunc("/api/chat/stop", s.handleChatStop)
	mux.HandleFunc("/api/chat/answer", s.handleChatAnswer)
	mux.HandleFunc("/api/chat/approve", s.handleChatApprove)
	mux.HandleFunc("/api/chat/feedback", s.handleChatFeedback)
	mux.HandleFunc("/api/chat/rollback", s.handleChatRollback)
	mux.HandleFunc("/api/chat/compact", s.handleChatCompact)
	mux.HandleFunc("/api/marketplace/search", s.handleMarketplaceSearch)
	mux.HandleFunc("/api/marketplace/install", s.handleMarketplaceInstall)
	mux.HandleFunc("/api/marketplace/refresh", s.handleMarketplaceRefresh)
	mux.HandleFunc("/api/memory/search", s.handleMemorySearch)
	mux.HandleFunc("/api/memory/list", s.handleMemoryList)
	mux.HandleFunc("/api/memory/rebuild", s.handleMemoryRebuild)
}
