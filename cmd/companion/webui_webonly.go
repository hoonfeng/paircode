// Web-only 版——仅注册平台回调 + 路由，所有 handler 逻辑在 web_server.go 统一实现。
//

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
	// ★ 接口插件化（2026-08-16）：chat/marketplace/memory 等 REST 接口已全部
	//   进内核路由表（kernel_register.go），由 core-api 磁盘插件装配。
	//   此处仅保留 WebSocket 传输端点（不属于 JSON API，宿主框架保留）：
	//   - /ws                全局事件流（agent 事件 → 前端）
	//   - /api/terminal/ws   PTY 终端桥
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		agent.ServeGlobalEventStreamWS(w, r, agentMgr)
	})
	mux.HandleFunc("/api/terminal/ws", agent.ServeTerminalWS)
}
