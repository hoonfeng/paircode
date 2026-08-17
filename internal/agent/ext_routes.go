// ═══════════════════════════════════════════════════════════════
// ext_routes.go — HTTP 接口插件化：外部路由注册表（对齐 harness webServer 服务）
//
// 背景（2026-08-16）：工具已全部磁盘插件化、agentloop 已工厂化（LoopFactory），
// 但 HTTP 接口（/api/* 约 60 路由）仍宿主硬编码（web_server.go mux.HandleFunc）。
// 本注册表提供**接口扩展点**：插件（JS ctx.http.register 或宿主 Go 代码）注册
// 自定义路由，ExtRouteMiddleware 在 mux 之前拦截——命中插件路由则执行，否则
// 交给现有 mux（宿主内置路由 + 静态文件不受影响）。
//
// 对齐 harness（ref/deepseek-harness/packages/host/webserver）：
//   - exact 精确路径（pathname 逐字匹配）
//   - prefix 前缀路径（path + "/*" 匹配 path 与 path/<anything>）
//   - 重复 (method, path) 注册报错（路由是装配层契约，冲突即配置错误）
//   - 注册返回 disposer（插件卸载自动回收）
//
// 与 harness 差异：harness 的 webServer 是 cordis 服务（ctx.webServer.register）；
// 本实现为全局注册表 + JS ctx.http 桥（goja 插件沙箱内无法构造 http.ResponseWriter，
// 桥把请求转成 JSON 对象、把 JS 返回值转写为 HTTP 响应）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// ExtRouteHandler 外部路由处理器（宿主 Go 侧形态，标准 http handler）。
type ExtRouteHandler = http.HandlerFunc

// extRoute 一条注册路由。
type extRoute struct {
	method string // HTTP 方法（GET/POST/…）
	path   string // 绝对路径（无尾斜杠；以 "/*" 结尾=前缀匹配）
	prefix bool   // 是否为前缀匹配
	h      ExtRouteHandler
}

var (
	extRoutesMu sync.RWMutex
	extRoutes   = map[string]extRoute{} // key: METHOD + " " + path
)

// RegisterExtRoute 注册一条外部 HTTP 路由。重复 (method, path) 注册报错。
// path 以 "/*" 结尾表示前缀匹配（匹配 path 与 path/<anything>，如 "/api/ext/*"）。
// 返回 disposer（卸载路由）。
// ★ key 用原始 path（含 "/*"）——prefix 路由与同路径精确路由可共存
//   （如 "/api/conversations" 精确 + "/api/conversations/*" 前缀）。
func RegisterExtRoute(method, path string, h ExtRouteHandler) (func(), error) {
	if method == "" || path == "" || h == nil {
		return nil, fmt.Errorf("外部路由注册: method/path/handler 不能为空")
	}
	prefix := false
	key := path
	if strings.HasSuffix(path, "/*") {
		prefix = true
		key = strings.TrimSuffix(path, "/*")
		key = strings.TrimSuffix(key, "/")
	}
	full := method + " " + path
	extRoutesMu.Lock()
	defer extRoutesMu.Unlock()
	if _, dup := extRoutes[full]; dup {
		return nil, fmt.Errorf("外部路由 %s 重复注册（路由是装配层契约，冲突即配置错误）", full)
	}
	extRoutes[full] = extRoute{method: method, path: key, prefix: prefix, h: h}
	return func() {
		extRoutesMu.Lock()
		defer extRoutesMu.Unlock()
		delete(extRoutes, full)
	}, nil
}

// ExtRouteInfo 已注册外部路由的信息（ctx.http.list() 查询）。
type ExtRouteInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Prefix bool   `json:"prefix"`
}

// RegisteredExtRoutes 返回全部已注册的外部路由（按 method+path 排序；ctx.http.list 用）。
func RegisteredExtRoutes() []ExtRouteInfo {
	extRoutesMu.RLock()
	defer extRoutesMu.RUnlock()
	out := make([]ExtRouteInfo, 0, len(extRoutes))
	for _, rt := range extRoutes {
		path := rt.path
		if rt.prefix {
			path += "/*"
		}
		out = append(out, ExtRouteInfo{Method: rt.method, Path: path, Prefix: rt.prefix})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Method != out[j].Method {
			return out[i].Method < out[j].Method
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// RegisterExtRouteAny 注册一条不区分 HTTP 方法的路由（method 通配 "*"）。
// 对齐 harness webServer：route 只有 kind/path/handler，不携带 method——
// handler 自行判断 req.method（Node 风格）。精确 method 路由优先于 "*"。
func RegisterExtRouteAny(path string, h ExtRouteHandler) (func(), error) {
	return RegisterExtRoute("*", path, h)
}

// ServeExtRoute 尝试用插件路由处理请求；命中返回 true（已写响应）。
// 供 ExtRouteMiddleware 与测试直接调用。
func ServeExtRoute(w http.ResponseWriter, r *http.Request) bool {
	p := r.URL.Path
	// 精确匹配优先（先同方法，再 "*" 通配——webServer 路由不区分方法）
	extRoutesMu.RLock()
	hit, ok := extRoutes[r.Method+" "+p]
	if !ok {
		hit, ok = extRoutes["*"+" "+p]
	}
	if !ok {
		// 前缀匹配：逐条检查（注册量小，线性可接受；method 同方法或通配）
		for _, rt := range extRoutes {
			if rt.prefix && (rt.method == r.Method || rt.method == "*") && (p == rt.path || strings.HasPrefix(p, rt.path+"/")) {
				hit, ok = rt, true
				break
			}
		}
	}
	h := hit.h
	extRoutesMu.RUnlock()
	if !ok || h == nil {
		return false
	}
	h(w, r)
	return true
}

// ExtRouteMiddleware 顶层处理器：插件路由优先，未命中交给 next（宿主 mux）。
func ExtRouteMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ServeExtRoute(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}
