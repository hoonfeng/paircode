# PairCode IDE 桌面端

## 目录结构

```
cmd/desktop/
├── main.go          ← 桌面端入口（初始化 → 注册 Handler → 启动窗口）
└── web-ui/          ← 前端工程（从 cmd/companion/web-ui 复制）
    ├── package.json
    ├── index.html
    ├── vite.config.js     ← 桌面端 Vite 配置（注入 __DESKTOP_MODE__ 标志）
    └── src/
        ├── sdk.js         ← 双模式 SDK（核心！Web 模式 HTTP+WS，桌面模式 Go 直调）
        ├── api.js         ← 向后兼容包装层（re-export from sdk.js）
        ├── agent-events.js
        ├── main.js
        ├── App.vue
        └── components/    ← Vue 组件

internal/bridge/
├── registry.go       ← Handler 注册表（method + path → HandlerFunc）
└── desktop.go        ← 桌面桥接（WebView 窗口 + JS 注入 + 事件推送）
```

## 通信模式

| 模式 | API 调用 | 事件推送 |
|------|---------|---------|
| **Web 开发** | HTTP fetch → Go Server | WebSocket |
| **桌面端** | desktopBridge.call() → Go Handler 直调 | Go → JS 回调 |

前端代码完全不变，仅通过 `window.__DESKTOP_MODE__` 标志切换通信方式。

## 构建与运行

### 前端构建
```bash
cd cmd/desktop/web-ui
npm install
npm run build    # 输出到 dist/
```

### Go 构建
```bash
cd F:\syproject\gou-ide
go build ./cmd/desktop
```

### 开发模式（前端 HMR）
```bash
# 终端1：启动前端 dev server
cd cmd/desktop/web-ui
npm run dev      # 端口 5174

# 终端2：运行桌面端（前端页面通过 WebView 加载 dev server 地址）
go run ./cmd/desktop
```

## 集成 WebView

桌面窗口目前是桩实现，需要集成 WebView 库。推荐方案：

### 方案 A：github.com/webview/webview（推荐）
```go
import "github.com/webview/webview"

w := webview.New(false)
defer w.Destroy()
w.SetTitle("PairCode IDE")
w.SetSize(1280, 800, webview.HintNone)
w.Bind("bridgeCall", bridgeInstance.BridgeCall)  // 绑定 Go 函数为 JS 全局函数
w.Init(BridgeInjectionScript())                   // 注入 desktopBridge JS 对象
w.Navigate("http://localhost:5174")                // 开发模式
// 或 w.Navigate("file:///path/to/dist/index.html") // 生产模式
w.Run()
```

### 方案 B：WebView2（Windows 专用）
使用 `github.com/jchv/go-webview2` 或 Microsoft 官方 WebView2。
