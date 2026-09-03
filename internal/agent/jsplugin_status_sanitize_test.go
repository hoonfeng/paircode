package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestExtRouteIllegalStatus 验证 JS 插件桥非法 HTTP 状态码防御（2026-09-03）：
// ctx.http 返回 {status:0}/{status:undefined} 或 webServer res.writeHead(0) 时，
// 不得触发 net/http panic（invalid WriteHeader code 0——LLM 流中断错误路径触发过），
// 应回退 200 且 body 保留（插件返回内容仍可达前端，便于其侧排障）。
func TestExtRouteIllegalStatus(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, "F:\\syproject\\gou-ide")

	const code = `return {
	  name: 'ext-status-demo',
	  apply(ctx) {
	    ctx.http.register('GET', '/api/ext/status0', () => ({ status: 0, body: 'body-zero' }))
	    ctx.http.register('GET', '/api/ext/status-undef', () => ({ status: undefined, body: 'body-undef' }))
	    ctx.http.register('GET', '/api/ext/status-throw', () => { throw new Error('boom') })
	    ctx.webServer.register({ kind: 'exact', path: '/api/ext/ws-writehead0', handler: (req, res) => {
	      res.writeHead(0)
	      res.end('ws-body')
	    }})
	  },
	}`
	id, err := host.DefineJSCodeFull(code, "js", "状态码防御测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}
	defer host.Unload(def.name)

	top := ExtRouteMiddleware(http.NewServeMux())

	// ① ctx.http 返回 status:0 → 回退 200，body 保留，不 panic
	rec := httptest.NewRecorder()
	top.ServeHTTP(rec, httptest.NewRequest("GET", "/api/ext/status0", nil))
	if rec.Code != 200 || rec.Body.String() != "body-zero" {
		t.Fatalf("status:0 应回退 200 且 body 保留: code=%d body=%q", rec.Code, rec.Body.String())
	}

	// ② status: undefined（ToInteger→0）→ 同样回退 200
	rec2 := httptest.NewRecorder()
	top.ServeHTTP(rec2, httptest.NewRequest("GET", "/api/ext/status-undef", nil))
	if rec2.Code != 200 || rec2.Body.String() != "body-undef" {
		t.Fatalf("status:undefined 应回退 200 且 body 保留: code=%d body=%q", rec2.Code, rec2.Body.String())
	}

	// ③ handler 抛错 → 500（callErr 路径；recover 兜底不炸 net/http）
	rec3 := httptest.NewRecorder()
	top.ServeHTTP(rec3, httptest.NewRequest("GET", "/api/ext/status-throw", nil))
	if rec3.Code != 500 {
		t.Fatalf("handler 抛错应 500: code=%d body=%q", rec3.Code, rec3.Body.String())
	}

	// ④ webServer res.writeHead(0) → 回退 200，body 保留
	rec4 := httptest.NewRecorder()
	top.ServeHTTP(rec4, httptest.NewRequest("GET", "/api/ext/ws-writehead0", nil))
	if rec4.Code != 200 || rec4.Body.String() != "ws-body" {
		t.Fatalf("writeHead(0) 应回退 200 且 body 保留: code=%d body=%q", rec4.Code, rec4.Body.String())
	}
}
