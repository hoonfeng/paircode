# PairCode IDE 桌面端

## 架构

```
cmd/desktop/
├── main.go              ← 桌面端入口（wb-ui WebView + 桥接）
├── desktop_bridge.go    ← ★ 桥接核心：fetch 拦截 + 真实 handler 注册 + agent 事件推送
└── web-ui/              ← 前端工程（构建脚本指向 cmd/companion/web-ui 真实源码）
    ├── vite.config.desktop.js  ← iife 构建 + libs 注入 + module→classic 后处理
    ├── sdk.js           ← 双模式通信 SDK（Web: HTTP+WS / 桌面: desktopBridge 直调）
    └── dist/            ← 构建产物（iife 自包含 HTML，wb-ui jsc 不支持 ES module）

internal/bridge/
└── registry.go          ← Handler 注册表（method + path → HandlerFunc）

internal/server/handler/ ← ★ 共享真实 handler（与 web 端同源）
└── register.go          ← RegisterAll() 注册 65+ 路由（fs/git/chat/conversations/tokens/...）
```

## 通信模式（fetch 拦截方案）

桌面端**不修改 companion 前端源码**，通过注入 `window.fetch` 拦截实现：

```
前端 Vue App（companion 源码原样）
  → api.js: fetch('/api/fs/list?path=...')
    → [注入的 fetch 拦截] 识别 /api/*（任意 method + query）
      → go.bridge_call(method, path, body, params)
        → bridge.Registry.Dispatch → 真实 handler（internal/server/handler）
          → 返回 {status, body} → Response 对象
  → 非 /api/* 请求放行（原生 fetch，桌面无网络时 reject）
```

WebSocket `/ws` 不实际建连：注入 WebSocket stub（不崩溃、不重连风暴），
agent 事件由 Go 端 `forwardAgentEvents`（SubscribeAll）通过
`desktopBridge.onAgentEvent / onStatus` 或 stub 的 dispatchMessage 推送。

**无 HTTP 服务、无 TCP 开销、无端口占用，数据全部来自本地真实逻辑。**

## 初始化顺序（关键）

```
main:
  1. webkit.NewWebView() + setupLoaders（dist 资源加载器）
  2. InitDesktopBridge(wv)：
       - core.Load() + LoadLastProject()       ← 真实配置/工作区
       - bridgeSessionManager.SetWorkspaceRoot ← 消息存储（JSONL + SQLite）
       - handler.RegisterAll → 65+ 真实 handler
       - bridge.Register("/bridge/call") → window.go.bridge_call
       - webkit.BeforePageScripts = injectJSBridge（fetch 拦截/desktopBridge/WS stub）
       - go forwardAgentEvents(wv)             ← agent 事件 → 前端
  3. wv.LoadHTML(index.html)：
       - 内部：RegisterFetch → bridge.InjectAll → InjectSDK → RegisterDOMBindings
       - ★ BeforePageScripts 钩子（window 已存在、页面 script 未执行）→ 注入拦截
       - 执行页面 script（vue → pinia → vue-router → app bundle）
```

## wb-ui 依赖的引擎能力

- **URL / URLSearchParams**：companion 前端 apiURL() 依赖 `u.searchParams.set`
- **BeforePageScripts 钩子**：DOM bindings 后、页面 script 前注入桌面环境
- **fetch / XMLHttpRequest**：page.RegisterFetch（bridge 路由优先 + HTTP 兜底）
- **window / location / document**：bindings.RegisterDOMBindings

## 构建

```bash
# 1. 构建前端（iife 版，源码来自 companion）
cd cmd/desktop/web-ui
node ..\..\companion\web-ui\node_modules\vite\bin\vite.js build --config vite.config.desktop.js

# 2. 构建桌面端（需要 CGO）
cd F:\syproject\gou-ide
set CGO_ENABLED=1
go build -tags=cgo ./cmd/desktop

# 3. 运行
go run -tags=cgo ./cmd/desktop
```

## 关键约束

1. **构建标签**：`cmd/desktop/` 只在 Windows + CGO 下编译（GLFW + Skia）
2. **iife 前端**：wb-ui jsc 不支持 ES module；vite.config.desktop.js 输出
   iife + external(vue/pinia/vue-router) + index.html 后处理为 classic script
3. **注入时机**：fetch 拦截必须经 BeforePageScripts（LoadHTML 内）注入——
   window 对象在 RegisterDOMBindings 时才创建，LoadHTML 前注入会 ReferenceError
4. **handler 共享**：internal/server/handler 与 web 端同源，新增 API 只需
   在 register.go 注册一行即桌面/共享生效
