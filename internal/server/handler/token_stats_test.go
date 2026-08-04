package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hoonfeng/paircode/internal/agent"
)

// 验证 desktop 共享 handler 的 token 统计真实数据（修复桩回归）。
func TestTokensStatsRealData(t *testing.T) {
	// 工作区级：读真实磁盘 .pair/token-stats.json（gou-ide 根目录）
	r := httptest.NewRequest(http.MethodGet, "/api/tokens/stats?workspaceRoot=F:/syproject/gou-ide", nil)
	w := httptest.NewRecorder()
	HandleTokensStats(w, r)
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}
	body := w.Body.String()
	if !contains(body, `"promptTokens":`) {
		t.Fatalf("缺少 promptTokens 字段: %s", body)
	}
	if !contains(body, `"totalTokens":`) {
		t.Fatalf("缺少 totalTokens 字段: %s", body)
	}
	t.Logf("tokens/stats 返回: %s", body)

	// 对话级：初始化 AgentMgr + store 后请求 token-stats
	mgr := agent.NewSessionManager()
	mgr.SetWorkspaceRoot("F:/syproject/gou-ide")
	AgentMgr = mgr
	defer func() { AgentMgr = nil }()

	r2 := httptest.NewRequest(http.MethodGet, "/api/conversations/conv_nonexist/token-stats", nil)
	w2 := httptest.NewRecorder()
	HandleConversationByID(w2, r2)
	if w2.Code != 200 {
		t.Fatalf("token-stats status=%d body=%s", w2.Code, w2.Body.String())
	}
	body2 := w2.Body.String()
	// 不存在的对话 → 返回零值 JSON（promptTokens 字段存在）
	if !contains(body2, `"promptTokens"`) {
		t.Fatalf("token-stats 缺少 promptTokens 字段: %s", body2)
	}
	t.Logf("conversations/{id}/token-stats 返回: %s", body2)

	// AgentMgr 为 nil 时不应 panic（健壮性）
	AgentMgr = nil
	r3 := httptest.NewRequest(http.MethodGet, "/api/conversations/conv_x/token-stats", nil)
	w3 := httptest.NewRecorder()
	HandleConversationByID(w3, r3)
	t.Logf("AgentMgr=nil 时返回: %d %s", w3.Code, w3.Body.String())
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
