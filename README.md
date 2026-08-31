# PairCode

本地优先的 AI 编程助手与 Agent 平台——把你的开发工作区变成可编程、可扩展、可托管的 Agent 环境。

PairCode 运行在本地（Windows / Linux / macOS），提供 Web IDE 界面（`http://localhost:9090`），
核心是一个 **Go 实现的 Agent 运行时**：双层循环（turn/step）驱动模型，**一切皆插件**
（磁盘优先、不重编译改 UI），内置代码图谱、语义搜索、持久记忆与知识库、子 Agent 编排、
MCP 支持与 HTTP 接口插件化。

> Web 界面由「壳 + 区域插件」构成：`web-ui`（壳）只保留骨架，全部 UI 区域
> （标题栏/活动栏/侧边栏/编辑器/右面板/状态栏/弹窗）都是独立磁盘插件包，可单独构建与替换。

## 特性

- **Agent 核心（Go）**：turn/step 双层循环状态模型，支持 max-tokens 粘滞、内容循环兜底、
  token 压力触发历史精简、会话级审核（review）与断线重连消息重同步。
- **一切皆插件**：插件 = 磁盘目录（`<workspace>/.pair/plugins/<id>/`），无需重编译；
  `cordis_define` 运行时定义（goja 沙箱），版本化 package 模型，插件可常驻/按需激活，
  装载状态与诊断可查询。
- **webServer 接口插件化**：`ctx.webServer` 路由注册，HTTP 端点由插件声明；
  事件流（`/ws`）由插件推送，Web 前端与外部客户端可订阅。
- **UI 区域插件化**：`dsh.ui.slot` 槽位注册表（跨副本共享），壳 + 7 大区域包；
  client 半（浏览器端）与 host 半分离，`boot()` 单入口两源合并装载。
- **工具体系**：内置工具（文件/搜索/编辑/执行/验证/代码图谱/记忆/knowledge）
  + 磁盘插件工具 + Node 桥插件（`@deepseek-ai/*` cordis 生态），同名工具并存可切换生效方。
- **代码图谱**：结构化符号索引（function/struct/interface/call_site/import…），
  支持影响分析、符号定位、跨文件调用链。
- **语义搜索**：本地 ONNX 向量模型（`config/models/bge-small-zh-v1.5`）离线嵌入，
  无网络依赖（CGO 不可用时自动降级关键词搜索）。
- **多项目工作区**：一个进程管理多个项目文件夹，工具/上下文按项目隔离。
- **子 Agent 与团队**：队长会话派生可续聊成员会话、任务 DAG 自动调度、质量门禁修复循环。
- **MCP 支持**：接入外部 MCP 服务器（scope: user/project）。
- **跨平台打包**：`packager` 一键产出 windows/linux/darwin 三平台发布包。

## 快速开始

```bash
# Windows（CGO 开启，前端产物已随仓库提供时可不构建 web-ui）
set CGO_ENABLED=1
go build -o companion.exe ./cmd/companion
./companion.exe
# 浏览器打开 http://localhost:9090（端口可用 WEB_PORT 环境变量覆盖）
```

工作区 = 启动目录（`InitCore` 读取当前目录/配置），`config/` 存放模型与偏好配置，
`logs/` 落盘运行日志。

## 架构速览

| 层 | 位置 | 说明 |
|---|---|---|
| 入口/Web | `cmd/companion` | HTTP API、WS 事件流、静态资源、插件装配 |
| Agent 核心 | `internal/agent` | 双层循环、工具执行器、插件宿主（goja 沙箱 + 磁盘插件）、Node 桥 |
| 服务层 | `internal/server` | HTTP handler、UI boot、插件 API |
| 桌面壳 | `cmd/desktop` | wb-ui 桌面窗口（goskia + webview，可选） |
| 工具库 | `pkg/` | codegraph（tsit 语法树）、db（sqlite）、executil、memory、summary、verify |
| 前端壳 | `web-ui` | 薄壳（唯一入口 `index.html` + `__PAIRCODE_CORE` 共享核心） |
| UI 区域插件源 | `plugins-src/ui-app` | 各区域 Vue 组件源，`build-ui.mjs` 逐插件构建 |
| 磁盘插件 | `<workspace>/.pair/plugins/<id>/` | 运行时插件（host 半 + client 半 + assets） |
| 运行时资源 | `.pair/assets/runtime/` | cordis bundle / bridge_node.js / web 前端产物（外部优先 + embed 兜底） |

## 开发

```bash
# Go 测试（需要 CGO）
set CGO_ENABLED=1
go test ./internal/agent/ ./pkg/...

# Web 壳构建（web-ui，产出 cmd/companion/web-ui/dist）
cd web-ui && npm install && npm run build

# UI 区域插件构建（plugins-src/ui-app → .pair/plugins/ui-*/assets）
cd plugins-src/ui-app && npm install && npm run build:ui

# 工具插件生成（幂等：已有插件不覆盖，Go 侧变更重跑同步）
go run -tags toolsgen ./dev/tool_plugin_gen

# 三平台发布包
go build -o packager.exe ./scripts/packager && ./packager.exe
```

插件开发见 `docs/plugin-development.md`（磁盘插件形态、`cordis_define` 用法、
工具注册/系统提示贡献/事件订阅、client 半与 UI 面板）。

## 许可

- 本项目：MIT（见 [LICENSE](LICENSE)）
- 第三方组件与许可清单：见 [THIRD_PARTY_NOTICES.md](docs/THIRD_PARTY_NOTICES.md)
