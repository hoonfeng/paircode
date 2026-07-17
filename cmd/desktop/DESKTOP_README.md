# PairCode IDE 桌面端

## 架构

```
cmd/desktop/
├── main.go          ← 桌面端入口（直接使用 wb-ui SDK）
└── web-ui/          ← 前端工程（从 cmd/companion/web-ui 复制）
    ├── src/
    │   ├── sdk.js   ← 双模式通信 SDK（Web: HTTP+WS / 桌面: desktopBridge 直调）
    │   └── api.js   ← 向后兼容包装层
    └── ...

internal/bridge/
└── registry.go      ← Handler 注册表（method + path → HandlerFunc，纯注册无 UI 依赖）
```

## 通信模式

| 模式 | SDK 层 | 传输方式 | 窗口引擎 |
|------|--------|---------|---------|
| **Web 开发** | `sdk.js` | HTTP fetch + WebSocket | 浏览器 |
| **桌面端** | `sdk.js` 检测 `__DESKTOP_MODE__` | `window.desktopBridge.call()` → Go 直调 | wb-ui (GLFW + Skia) |

数据流：
```
前端 Vue App
  → sdk.js: apiGet('/api/fs/list', {path: '...'})
    → window.desktopBridge.call('GET', '/api/fs/list', '', '{"path":"..."}')
      → wb-ui jsc 引擎 → Go jsc.NewNativeFunction 回调
        → bridge.Registry.Dispatch('GET', '/api/fs/list', ...)
          → handleFSList(w, r)
            → 返回 JSON 结果
```

**无 HTTP 请求、无 TCP 开销、无端口占用。**

## wb-ui 依赖

wb-ui 项目位于 `F:\syproject\wb-ui`，是纯 Go 的桌面 UI SDK：
- **GLFW** → 窗口管理 + 输入事件
- **Skia** (goskia) → GPU 加速 2D 渲染
- **JavaScriptCore Port** → JS 引擎（支持 Proxy/Promise/ES Module）
- **WebKit Port** → HTML/CSS 解析 + 布局 + 渲染管线
- **bindings** → Go↔JS 桥接（注册 Go 函数为 JS 全局对象）

## 构建

```bash
# 需要 CGO（wb-ui 依赖 GLFW + Skia）
set CGO_ENABLED=1

# 构建桌面端
go build -tags=cgo ./cmd/desktop

# 构建前需要先构建前端
cd cmd/desktop/web-ui
npm install
npm run build    # 输出到 dist/
```

## 关键约束

1. **构建标签**：`cmd/desktop/main.go` 使用 `//go:build windows && cgo`，只在 Windows + CGO 环境下编译
2. **前端加载**：先读取 `web-ui/dist/index.html`（Vite 构建产物），不存在时退回到内嵌测试页
3. **Handler 迁移**：所有 Handler 桩需从 `cmd/companion/web_server.go` 逐步迁移到共享包
