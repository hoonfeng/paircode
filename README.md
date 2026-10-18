# PairCode

本地优先的 AI 编程助手与 Agent 平台——把你的开发工作区变成可编程、可扩展、可托管的 Agent 环境。

PairCode 运行在本地（Windows / Linux / macOS），提供 Web IDE 界面（`http://localhost:9090`），
核心是一个 **Go 实现的 Agent 运行时**：双层循环（turn/step）驱动模型，**一切皆插件**
（磁盘优先、不重编译改 UI），内置代码图谱、语义搜索、持久记忆与知识库、任务 DAG 团队编排、
MCP 支持与 HTTP 接口插件化。

> Web 界面由「壳 + 区域插件」构成：`web-ui`（壳）只保留骨架，全部 UI 区域
> （标题栏/活动栏/侧边栏/编辑器/右面板/状态栏/弹窗）都是独立磁盘插件包，可单独构建与替换。

## 特性

- **Agent 核心（Go）**：turn/step 双层循环状态模型，支持 max-tokens 粘滞、内容循环兜底、
  token 压力触发历史精简、绕圈/重复检测、会话级审核（review）与断线重连消息重同步。
  ★ 工具调用一律串行执行（不并行调用工具）；子 Agent 默认关闭，需 `PAIR_ALLOW_SUBAGENTS=1` 显式开启。
- **段预算双闸门 + 自动分段续跑**：单段同时受**步数预算**与**工具调用预算**约束，任一达上限
  即结束本段并自动续跑（同会话历史保留），续跑段数有上限；三项（`stepBudget` /
  `toolCallBudget` / `maxToolBudgetSegments`，默认 120 / 120 / 20）在「设置 → Agent」实时可配，
  长任务不再中断。
- **会话交接（提交消息）**：续跑与新提交对话两个时机自动整理上下文——超阈值时生成
  「会话交接·提交消息」（相关性 + 目标/已完成/当前状态/下一步），未达阈值逐字节原样注入
  （缓存前缀零影响）；复用期内轻量语义复检、相关性变化驱动保留深度。
- **一切皆插件**：插件 = 磁盘目录（`<workspace>/.pair/plugins/<id>/`），无需重编译；
  `cordis(op=define)` 运行时定义（goja 沙箱），版本化 package 模型，插件可常驻/按需激活，
  装载状态与诊断可查询。
- **webServer 接口插件化**：`ctx.webServer` 路由注册，HTTP 端点由插件声明；
  事件流（`/ws`）由插件推送，Web 前端与外部客户端可订阅。
- **UI 区域插件化**：`dsh.ui.slot` 槽位注册表（跨副本共享），壳 + 7 大区域包；
  client 半（浏览器端）与 host 半分离，`boot()` 单入口两源合并装载。
- **创作域插件（独立发布）**：六大创作域——画板 / UI 设计 / 3D 模型 / 音乐 / 2D 角色 / 人声，
  「一域一包」（工具面 + 面板同包，`@paircode/tool-*`）；真源在 `plugins-dist/`（与
  `.pair/plugins` 同权扫描、开发期 junction 挂载、**不进 IDE 发布包**），随市场独立分发，
  不随 IDE 发版；主内容区视图（`ui.registerView`）与「并排对话」布局由面板承载。
- **工具体系**：内置工具（文件读写 / 内容搜索 / `apply_patch` 补丁编辑 / 命令执行 / 代码图谱 / 记忆 / 知识库）
  + 磁盘插件工具 + Node 桥插件（`@deepseek-ai/*` cordis 生态），同名工具并存可切换生效方。
- **代码图谱**：结构化符号索引（function/struct/interface/call_site/import…），
  支持影响分析、符号定位、跨文件调用链。
- **语义搜索**：本地 ONNX 向量模型（`config/models/bge-small-zh-v1.5`）离线嵌入，
  无网络依赖（CGO 不可用时自动降级关键词搜索）。
- **多项目工作区**：一个进程管理多个项目文件夹，工具/上下文按项目隔离。
- **任务 DAG 团队编排**：任务依赖门禁、质量门禁（契约 / verdict / findings，修复 + 复审自动派生）、
  两阶段批准（staged → approve → running），由队长会话单 Agent 多步执行完成。
- **创造模式（`/创造 <需求>`）**：按一句需求自主创建「场景」工具集——能力盘点 → 组合现有插件 →
  固化到 `.pair/toolsets/<场景名>.json`，对话面板工具集选择器即时可切换。
- **LLM 追踪**：`llm-trace` 插件逐请求落盘前缀命中分析（JSONL），定位缓存前缀断裂与成本热点。
- **MCP 支持**：接入外部 MCP 服务器（scope: user/project）。
- **跨平台打包**：`packager` 一键产出 windows/linux/darwin 三平台发布包。

## 从 Release 运行（推荐）

发布包三平台统一形态：**主体只有一个可执行文件，双击即用，无需解压**。
`pair-server[.exe]` 是可选的服务版（无窗口，浏览器访问 `http://localhost:9090`）。

| 平台 | 发布包 | 启动文件 | 说明 |
|---|---|---|---|
| Windows | `PairCode-<版本>.zip` | `PairCode.exe` | Release 同时提供**裸 `PairCode.exe`**，下载后直接双击即可（运行时库 / 前端 / 插件均已嵌入） |
| Linux | `PairCode-linux-<版本>.zip` | `PairCode-linux-amd64` | `chmod +x` 后执行；同样单文件自足（Skia 运行库内嵌解出） |
| macOS | `PairCode-darwin-<版本>.zip` | `PairCode-darwin-arm64` | 同上；首次运行需在「系统设置 → 隐私与安全性」中允许 |

- 工作区 = 启动文件所在目录；`config/`、`logs/`、`.pair/` 会就地生成。
- 端口默认 9090，可用环境变量 `WEB_PORT` 覆盖（桌面版窗口与后端同进程）。

### Linux / macOS 中文字体（v1.6.22 起）

GUI 引擎（wb-ui `fontmgr`）早期只按 Windows 字体族名找字体，Linux/macOS 上中文会渲染成方块。
v1.6.22 起的行为：

- **白名单按平台分表**（`runtime.GOOS`）：Linux 覆盖 `Noto Sans CJK SC/JP`、`Source Han Sans`、
  `WenQuanYi Zen Hei/Micro Hei`、`Droid Sans Fallback`；macOS 覆盖 `PingFang SC`、
  `Hiragino Sans GB`、`Heiti SC`、`STHeiti`、`Songti SC` 等。Windows 表保持原样。
- **TTC 逐 face 尝试**：`NotoSansCJK-Regular.ttc` 这类大集合按 face 索引逐个加载，
  至少保证拿到 SC face（不会停在 JP/KR face 上）。
- **平台默认 CJK 族 + 分类修正**：找不到对应西文族时按平台选默认 CJK 族；`classifyFont`
  把上述族标为 cjk，等宽/无衬线回退链不再断在方块上。
- **可观测**：启动日志打印 `[fontmgr] cjk=<族名>`（无可用字体为 `(none)`），
  release 流水线对该行做硬断言，字体回归会直接把发布打红。

Linux 上需要系统装有 CJK 字体（Ubuntu：`fonts-noto-cjk`、`fonts-wqy-zenhei`）；
macOS 自带 PingFang，无需额外安装。逐版本核验记录见 `docs/releases/`。

## 快速开始

```bash
# Windows（CGO 开启；前端产物已随仓库提供，无需重建）
set CGO_ENABLED=1
go build -o companion.exe ./cmd/companion
./companion.exe
# 浏览器打开 http://localhost:9090（端口可用 WEB_PORT 环境变量覆盖）

# Linux / macOS
export CGO_ENABLED=1
go build -o companion ./cmd/companion
./companion
# 浏览器打开 http://localhost:9090
```

工作区 = 启动目录（`InitCore` 读取当前目录/配置），`config/` 存放模型与偏好配置，
`logs/` 落盘运行日志。

### 平台能力说明

| 能力 | Windows | Linux | macOS | 依赖 |
|---|---|---|---|---|
| Web IDE / Agent / 插件 | ✅ | ✅ | ✅ | — |
| 内置终端（PTY） | ✅ ConPTY | ✅ pts（creack/pty） | ✅ pts（creack/pty） | 至少一个 shell（bash/zsh/sh） |
| 桌面/区域/窗口截图 | ✅ GDI | ✅ 全屏 + 标准库裁剪 | ✅ screencapture | Linux 需 grim / scrot / ImageMagick `import` / gnome-screenshot / spectacle 之一；窗口截图另需 xdotool(X11) |
| 「在资源管理器中显示」 | ✅ explorer 选中 | ✅ xdg-open | ✅ Finder `open -R` | — |
| 在线更新 | ✅ .bat 重启脚本 | ✅ sh 重启脚本 | ✅ sh 重启脚本 | 安装目录可写 |
| 语义搜索（ONNX） | ✅ | ✅ | ✅ | 需对应平台的 onnxruntime 动态库（打包配置已分流） |

## 架构速览

| 层 | 位置 | 说明 |
|---|---|---|
| 入口/Web | `cmd/companion` | HTTP API、WS 事件流、静态资源、插件装配 |
| Agent 核心 | `internal/agent` | 双层循环、工具执行器、插件宿主（goja 沙箱 + 磁盘插件）、Node 桥 |
| 服务层 | `internal/server` | HTTP handler、UI boot、插件 API |
| 工具库 | `pkg/` | codegraph（tsit 语法树）、db（sqlite）、executil、memory、summary、verify |
| Web 前端源 | `plugins-src/ui-app` | 壳（vite 构建）+ 各区域 Vue 组件源（`build-ui.mjs` 逐插件构建） |
| 磁盘插件 | `<workspace>/.pair/plugins/<id>/` | 运行时插件（host 半 + client 半 + assets） |
| 独立发布插件 | `plugins-dist/<id>/` | 创作域插件真源（同权扫描；不进 IDE 发布包，开发期 junction 挂载到 `.pair/plugins`） |
| 运行时资源 | `.pair/assets/runtime/` | cordis bundle / bridge_node.js / web 前端产物（外部优先 + embed 兜底） |

## 开发

```bash
# Go 测试（需要 CGO）
set CGO_ENABLED=1
go test ./internal/agent/ ./pkg/...

# Web 前端构建（plugins-src/ui-app → .pair/assets/runtime/web + cmd/companion/web-ui/dist embed 兜底）
cd plugins-src/ui-app && npm install && npm run build

# UI 区域插件构建（plugins-src/ui-app → .pair/plugins/ui-*/assets）
cd plugins-src/ui-app && npm install && npm run build:ui

# 工具插件生成（幂等：已有插件不覆盖，Go 侧变更重跑同步）
go run -tags toolsgen ./dev/tool_plugin_gen

# 创作域插件（独立发布）打包 / 发布（真源 plugins-dist/，不进 IDE 发布包）
node scripts/publish-official-plugins.mjs                                    # 打包 + 校验（默认不发布）
node scripts/publish-official-plugins.mjs --publish --only tool-art          # 真实发布（2FA 时加 --otp）
node scripts/verify-dist-isolation.mjs                                       # 护栏：独立发布包 ↔ 打包排除项一致

# 三平台发布包
go build -o packager.exe ./scripts/packager && ./packager.exe
```

插件开发见 `docs/plugin-development.md`（磁盘插件形态、`cordis(op=define)` 用法、
工具注册/系统提示贡献/事件订阅、client 半与 UI 面板）。

## 许可

- 本项目：MIT（见 [LICENSE](LICENSE)）
- 第三方组件与许可清单：见 [THIRD_PARTY_NOTICES.md](docs/THIRD_PARTY_NOTICES.md)
