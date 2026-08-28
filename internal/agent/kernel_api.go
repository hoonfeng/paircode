// ═══════════════════════════════════════════════════════════════
// kernel_api.go — 内核 API 路由表：内置 HTTP 接口的能力注册
//
// 背景（2026-08-16）：内置 /api/* 接口（约 90 条）原本在 web_server.go
// mux.HandleFunc 硬编码挂载。接口插件化后：
//   - 内核实现（Go handler 函数）保留为本文件的内核路由表（能力层）；
//   - 路由的「挂载权」交给插件：core-api 磁盘插件在 apply 时调用
//     ctx.kernel.install(routes) 把清单逐条挂到插件 ext 路由表
//     （ExtRouteMiddleware 在宿主 mux 之前拦截，命中即执行）；
//   - 卸载（插件 stop/undefine）时经 ctx.kernel 登记的 disposer 自动
//     摘除路由——接口随插件生命周期生灭，宿主不再硬编码挂载。
//
// 对齐工具插件化（hostTool）：实现留内核（Go），注册权在插件（JS）——
// 本表即「接口的 hostTool」。插件用 ctx.kernel.routes() 可见全部清单。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// KernelRouteMeta 内核路由元数据（插件可见的清单条目）。
type KernelRouteMeta struct {
	Key    string `json:"key"`    // 唯一标识（插件 install 用）
	Method string `json:"method"` // HTTP 方法
	Path   string `json:"path"`   // 路由路径（以 /* 结尾=前缀匹配，对齐 ext 表）
	Desc   string `json:"desc"`   // 用途说明
}

// kernelRoute 一条内核路由（元数据 + 实现）。
type kernelRoute struct {
	meta KernelRouteMeta
	h    http.HandlerFunc
}

var (
	kernelAPIMu     sync.RWMutex
	kernelAPITable  = map[string]kernelRoute{} // key -> route（能力注册表）
	kernelInstalled = map[string]bool{}        // key -> 是否已挂到 ext 表（防重复）
)

// KernelAPIRegister 注册一条内核能力路由（key 唯一，重复注册报错）。
// method 支持逗号分隔多方法（如 "GET,POST"——同一 handler 内部按方法分支，
// 对应原 mux.HandleFunc 不区分方法、handler 自查的语义）。
// path 以 "/*" 结尾表示前缀匹配（对齐 ext 路由表：/api/conversations/*）。
func KernelAPIRegister(key, method, path, desc string, h http.HandlerFunc) error {
	if key == "" || method == "" || path == "" || h == nil {
		return fmt.Errorf("内核路由注册: key/method/path/handler 不能为空")
	}
	kernelAPIMu.Lock()
	defer kernelAPIMu.Unlock()
	if _, dup := kernelAPITable[key]; dup {
		return fmt.Errorf("内核路由 %q 重复注册（key 唯一）", key)
	}
	kernelAPITable[key] = kernelRoute{
		meta: KernelRouteMeta{Key: key, Method: method, Path: path, Desc: desc},
		h:    h,
	}
	return nil
}

// KernelAPIRoutes 返回全部内核路由元数据（按 key 排序；插件 routes() 查询）。
func KernelAPIRoutes() []KernelRouteMeta {
	kernelAPIMu.RLock()
	defer kernelAPIMu.RUnlock()
	out := make([]KernelRouteMeta, 0, len(kernelAPITable))
	for _, rt := range kernelAPITable {
		out = append(out, rt.meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// KernelAPIInstall 把一条内核路由挂到插件 ext 路由表（路由挂载权在插件）。
// method 逗号分隔时逐条注册（如 GET,POST 注册两条 ext 路由，同一 handler）。
// 返回 disposer（摘除全部相关路由）；重复安装同一 key 报错（装配层契约）。
// 插件侧（ctx.kernel.install）会登记 disposer，插件卸载时自动摘除。
func KernelAPIInstall(key string) (func(), error) {
	kernelAPIMu.Lock()
	defer kernelAPIMu.Unlock()
	rt, ok := kernelAPITable[key]
	if !ok {
		return nil, fmt.Errorf("内核路由 %q 不存在（可用 ctx.kernel.routes() 查看清单）", key)
	}
	if kernelInstalled[key] {
		return nil, fmt.Errorf("内核路由 %q 已安装（重复安装是装配错误）", key)
	}
	methods := strings.Split(rt.meta.Method, ",")
	disposers := make([]func(), 0, len(methods))
	for _, m := range methods {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		dispose, err := RegisterExtRoute(m, rt.meta.Path, rt.h)
		if err != nil {
			// 回滚已注册的部分
			for _, d := range disposers {
				d()
			}
			return nil, fmt.Errorf("内核路由 %q 安装失败: %w", key, err)
		}
		disposers = append(disposers, dispose)
	}
	if len(disposers) == 0 {
		return nil, fmt.Errorf("内核路由 %q method 无效: %q", key, rt.meta.Method)
	}
	kernelInstalled[key] = true
	return func() {
		for _, d := range disposers {
			d()
		}
		kernelAPIMu.Lock()
		delete(kernelInstalled, key)
		kernelAPIMu.Unlock()
	}, nil
}

// KernelAPIInstalledKeys 返回已安装的内核路由 key（插件 installed() 查询/自检）。
func KernelAPIInstalledKeys() []string {
	kernelAPIMu.RLock()
	defer kernelAPIMu.RUnlock()
	out := make([]string, 0, len(kernelInstalled))
	for k := range kernelInstalled {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// KernelAPITotal 内核表容量（status 等接口用）。
func KernelAPITotal() int {
	kernelAPIMu.RLock()
	defer kernelAPIMu.RUnlock()
	return len(kernelAPITable)
}
