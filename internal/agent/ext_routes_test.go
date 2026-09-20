package agent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestExtRouteJSPlugin 验证 HTTP 接口插件化全链路：JS 插件经 ctx.http.register
// 注册自定义 API 路由（精确 + 前缀）→ ExtRouteMiddleware 拦截 → 请求转 JS
// → 响应写回；插件卸载自动注销；重复注册报错；宿主 mux 路由不受影响。
func TestExtRouteJSPlugin(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, "F:\\syproject\\gou-ide")

	const code = `return {
	  name: 'ext-route-demo',
	  apply(ctx) {
	    ctx.http.register('GET', '/api/ext/hello', (req) => {
	      return { status: 200, body: 'hello:' + (req.query || ''), headers: {'X-Ext': '1'} }
	    })
	    ctx.http.register('POST', '/api/ext/echo', (req) => {
	      return { status: 201, body: 'echo:' + req.body }
	    })
	    ctx.http.register('GET', '/api/ext/prefix/*', (req) => {
	      return 'prefix:' + req.path
	    })
	  },
	}`
	id, err := host.DefineJSCodeFull(code, "js", "接口插件化测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}
	defer host.Unload(def.name)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("host-ok"))
	})
	top := ExtRouteMiddleware(mux)

	// ① 精确 GET + query
	rec := httptest.NewRecorder()
	top.ServeHTTP(rec, httptest.NewRequest("GET", "/api/ext/hello?x=1", nil))
	if rec.Code != 200 || rec.Body.String() != "hello:x=1" || rec.Header().Get("X-Ext") != "1" {
		t.Fatalf("GET /api/ext/hello 异常: code=%d body=%q x-ext=%q", rec.Code, rec.Body.String(), rec.Header().Get("X-Ext"))
	}

	// ② POST + body
	rec2 := httptest.NewRecorder()
	top.ServeHTTP(rec2, httptest.NewRequest("POST", "/api/ext/echo", strings.NewReader("ping")))
	if rec2.Code != 201 || rec2.Body.String() != "echo:ping" {
		t.Fatalf("POST /api/ext/echo 异常: code=%d body=%q", rec2.Code, rec2.Body.String())
	}

	// ③ 前缀匹配
	rec3 := httptest.NewRecorder()
	top.ServeHTTP(rec3, httptest.NewRequest("GET", "/api/ext/prefix/a/b", nil))
	if rec3.Body.String() != "prefix:/api/ext/prefix/a/b" {
		t.Fatalf("前缀路由异常: %q", rec3.Body.String())
	}

	// ④ 宿主 mux 路由不受影响（未命中插件路由走 mux）
	rec4 := httptest.NewRecorder()
	top.ServeHTTP(rec4, httptest.NewRequest("GET", "/api/health", nil))
	if rec4.Body.String() != "host-ok" {
		t.Fatalf("宿主路由被拦截: %q", rec4.Body.String())
	}

	// ⑤ 未注册路径 → 404（mux 默认）
	rec5 := httptest.NewRecorder()
	top.ServeHTTP(rec5, httptest.NewRequest("GET", "/api/nonexistent", nil))
	if rec5.Code != 404 {
		t.Fatalf("未注册路径应 404: code=%d", rec5.Code)
	}

	// ⑥ 插件卸载自动注销
	if err := host.Unload(def.name); err != nil {
		t.Fatalf("卸载失败: %v", err)
	}
	rec6 := httptest.NewRecorder()
	top.ServeHTTP(rec6, httptest.NewRequest("GET", "/api/ext/hello", nil))
	if rec6.Code != 404 {
		t.Fatalf("卸载后插件路由应 404: code=%d", rec6.Code)
	}

	// ⑦ 重复注册报错
	unregDup, err := RegisterExtRoute("GET", "/api/ext/dup", func(w http.ResponseWriter, r *http.Request) {})
	if err != nil {
		t.Fatalf("首次注册应成功: %v", err)
	}
	// ★ 这是直接写全局路由表（非插件路由，插件卸载不会自动注销）：必须显式注销，
	//   否则同包重复运行（-count=2 等）时第二轮这条「首次注册」会因第一轮残留条目
	//   报「重复注册」→ 假失败（实测 -count=2 必现）。
	t.Cleanup(unregDup)
	if _, err = RegisterExtRoute("GET", "/api/ext/dup", func(w http.ResponseWriter, r *http.Request) {}); err == nil {
		t.Fatal("重复注册应报错")
	}
}
