# 插件目录（.pair/plugins/）——插件 = 自包含包（源码 + 二进制 + 资源）

> 主程序（companion.exe）只保留框架：装载插件、调度工具、审批、对话。
> 插件的**实现源码 / 独立二进制 / 加载资源**全部位于本目录，用户可自行修改、
> 重新构建、替换——改完重启生效，无需重新编译主程序。

## 目录结构约定

| 形态 | 路径 | 说明 |
|---|---|---|
| 插件包（可装载） | `<name>/package.json + index.js[+client.js]` | 启动扫描装载（LoadGlobalPlugins）；index.js = host 半（api 声明 + 调度），client.js = 浏览器半（UI 渲染） |
| 源码包（不可装载） | `<name>-src/` | ★ 插件**源码**。装载器按 `-src` 后缀跳过；用户改源码 → 重新构建 → 产物进插件包/assets |
| 独立二进制 | `<name>/bin/<name>.exe` | ★ 依赖 Go 内核的工具独立成**单独二进制项目**（源码在 `plugins-src/plugins/<name>/`），产物放本插件目录 bin/——插件自包含，改源码重编译即更换实现 |
| 加载资源 | `<name>/assets/` | 二进制/JS 运行所需资源（字体、模板、索引数据等），随插件目录分发 |

## 二进制插件协议（宿主 ctx.binary 服务）

```text
JS 插件 execute → ctx.binary.exec(tool, args[, {timeout}]) → text
  宿主定位 <插件目录>/bin/<插件名>.exe（Windows 加 .exe）
  stdin  JSON {"tool":"binary_strings","args":{...},"root":"<工作区根>"}
  stdout JSON {"ok":true,"text":"..."} | {"ok":false,"error":"..."}
  exit 0（协议错误 exit 2）
```

- `ctx.binary.dir()` → 插件目录绝对路径（JS 可拼接 assets/ 资源路径）
- `ctx.http.register(method, path, fn)` → 注册自定义 HTTP API 路由（接口插件化）：
  - path 以 `/*` 结尾=前缀匹配（如 `/api/ext/*` 匹配其下任意路径）
  - `fn(req) → resp`：req=`{method, path, query, headers, body}`；
    resp=`{status, body, headers}` 或字符串
  - 返回 unregister 函数；插件卸载自动注销；重复 (method, path) 注册报错
  - 插件路由在宿主 mux 之前拦截：命中走插件、未命中走内置 /api/* 与静态文件
  - 实现：internal/agent/ext_routes.go（ExtRouteMiddleware）+ jsplugin.go ctx.http
  - ★ 落地用例（2026-08-16）：磁盘插件 `web-api/` 用本能力注册 `/api/ext/*`
    6 条路由（status / fetch 同源代理 / fs-read / fs-exists / fs-list / routes），
    curl 直接消费，验证「插件扩展 HTTP 接口」链路全通（见 web-api/index.js）
- `ctx.kernel.*` → 内置 HTTP 接口装配（★ 接口插件化：Go 硬编码清零）：  - 内置 /api/* 接口实现保留 Go 内核路由表（internal/agent/kernel_api.go，
    由 cmd/companion/kernel_register.go 注册 82 条），**挂载权在插件**；
  - `ctx.kernel.routes()` → 全部内核接口清单 `[{key,method,path,desc}]`
  - `ctx.kernel.install(list)` → 把清单中指定 key 挂到插件 ext 路由表
    （返回 `{installed, missing, total}`；重复安装报错=装配层契约）
  - `ctx.kernel.installed()` / `ctx.kernel.total()` → 已安装 key / 表容量
  - 插件卸载自动摘除路由（接口随插件生命周期生灭）
  - ★ 落地用例（2026-08-16）：磁盘插件 `core-api/` 持有 82 条内置接口清单
    并在 apply 时全量安装（改 core-api/index.js 的 ROUTES 数组可增删接口）——
    web_server.go mux 不再硬编码任何 /api/* 路由（只剩 /ws、/api/terminal/ws
    WebSocket 端点与 /plugins-assets/、静态文件框架路由）
  - ⚠️ 管理面自锁：/api/plugins* 与 /api/toolsets* 也由 core-api 装配，
    停用 core-api 会导致插件面板接口消失（重启恢复，与停用 ui-app 同理）
- `ctx.sse.register(path, fn)` → 注册 SSE 实时推送端点（事件通道插件化）：
  - `fn(emit, params) → cleanup`：连接建立时在 VM 锁内调用一次（可 await）；
    `emit(event, payload)` 推送事件（payload JSON 序列化，跨调用可保存复用，
    连接断开后抛错）；返回值（函数）为 cleanup，连接断开时调用
  - 返回 unregister 函数；插件卸载自动注销；重复 path 注册报错
  - 实现：internal/agent/ext_sse.go（ExtSSEMiddleware）+ jsplugin.go ctx.sse；
    浏览器/外部用 EventSource/curl 消费（text/event-stream，逐事件 Flush）
- `ctx.ws.register(path, fn)` → 注册双向 WebSocket 端点（实时双向通道插件化）：
  - `fn(conn, params) → cleanup`：连接建立后于 VM 锁内调用一次（可 await）；
    `conn.send(payload)` 发送（string 直传/其他 JSON 序列化，断开后抛错）、
    `conn.onMessage(fn)` 注册消息回调（Go 读循环 → VM 锁内调 JS，文本帧
    尝试 JSON.parse）、`conn.close()` 主动关闭；handler 返回 cleanup（断开时调用）
  - 返回 unregister 函数；插件卸载自动注销；重复 path 注册报错
  - 实现：internal/agent/ext_ws.go（ExtWSMiddleware）+ wsconn.go（RFC6455
    帧实现，从 cmd/companion 下沉）+ jsplugin.go ctx.ws
  - 宿主端点同样由框架层处理：/ws 全局事件流（event_ws.go）、
    /api/terminal/ws PTY 桥（terminal_ws.go）——main 只保留路由注册

## 会话状态协议（tool-system 保持 hostTool 的依据）

框架协议工具（update_tasks / ask_user / tool_stats /
cordis_* / toolset_* / history_* 等）**保持宿主实现**——它们绑定会话内存态
（Loop 计划步骤、审核门、UI 任务面板/进度、对话压缩引用），非纯磁盘编码能力。

## 框架能力清单（宿主固有，不占插件位）

> 除了磁盘插件（可分离/可扩展）与 hostTool（宿主注册表工具）外，还有一类
> **框架能力**——宿主自我暴露或宿主核心数据，分离无意义。它们**内联在宿主
> 构造处（NewPluginHost）直接提供，不以插件形态存在**：不可启停、不出现在
> 插件列表/Inspect 中——插件列表只含真实可管理的插件。
>
> | 能力 | 位置 | 为什么不可分离 |
> |------|------|----------------|
> | workspaceRoot 服务（原 sysinfo） | `NewPluginHost`（plugin.go：`h.ctx.Provide("workspaceRoot", root)`） | 向插件生态暴露**宿主自己**的工作区根：JS 插件走宿主直接注入（app.workspaceRoot）不消费服务；Go/JS 插件经 `ctx.Get("workspaceRoot")` 取——提供者即宿主，分离无意义 |
> | 内置工具集模板（原 toolset-tpl-core） | `NewPluginHost`（plugin.go：`registerBuiltinTemplates(h)`，toolset_templates.go） | toolset_build 的动态组合数据源（项目助手/Git 流/Web 开发模板），Generate 逻辑内嵌宿主；toolset_build 本身是宿主框架能力，数据源随宿主合理（市场/用户可 RegisterTemplate 追加专属模板） |
> | SystemTool 组（update_tasks/tool_stats/history_*） | `RegisterHostFrameworkTools` | 会话绑定内存态（Loop 计划/审核门/UI 任务面板/对话压缩），见上节「会话状态协议」 |
> | harness 7 工具（read/write/edit/glob/grep/bash/str_replace_editor） | 宿主注册表 | agent 自身能力面（框架协议），JS 插件同名接管时 ArchiveHostTool 存档供 ctx.hostTool 兜底 |
>
> 判别标准：**提供者是否就是宿主自身、数据是否绑定宿主内存态**——两者取一即
> 不可分离，归为框架能力**内联宿主**（不占插件位）；其余（业务实现）一律磁盘
> 插件（JS 原生化或独立二进制）。

磁盘侧协议（供未来二进制化/外部消费参考）：

| 数据 | 位置 | 格式 |
|------|------|------|
| 任务列表 | `.pair/tasks/<id>.json` | 每任务一文件：`{id, subject, description, status, dependencies[], planStepIndex?, convId?, created_at, updated_at}`；全量替换=清旧写新 |
| 工具统计 | 宿主内存（GetToolStats） | ToolStatsSummary（调用次数/时长/成功失败） |
| 对话历史 | `.pair/conversations/*.jsonl + index.json` | JSONL 追加（只写不读，压缩时读） |
| 插件定义 | `.pair/plugins/dynamic.json` / `.pair/toolsets/*.json` | 磁盘插件包 + 工具集固化 |

> 若未来要二进制化 update_tasks：需宿主 TaskManager 改为「磁盘为准 + 每次操作
> 后内存同步」（mtime 感知），并保证 UI（/api/tasks 读宿主内存）与 agent 视角一致
> ——改动面大且破坏循环契约，当前不采纳。
- `ctx.binary.exec(tool, args, {bin})` → opts.bin 可指定**其它插件目录的二进制**
  （历史形态：跨插件共用统一二进制；第四轮已演进为每插件独立 exe，各工具组
  JS 缺省 bin=插件名，无跨插件引用）
- 二进制内可用 `os.Executable()` 定位自身目录 → 上级即插件目录（读 assets/）
- 超时默认 60s（opts.timeout 毫秒可覆盖）
- 示例：`.pair/plugins/tool-binary/`（8 个二进制工具，含逆向分析）——协议实现
  参考 `plugins-src/plugins/tool-binary/main.go`

## 独立插件二进制（每插件一个 exe，2026-08-16 第四轮）

依赖 Go 实现的工具组（git/memory/verify/project-info/binary（含逆向）/
debug/vision/screenshot/web-debug/bug/office/lsp/codegraph/codegraph-extra/
search/web 等 17 组）**各自一个独立二进制**承载实现：

- 源码：`plugins-src/plugins/<name>/main.go` + `plugins-src/plugins/<name>/impl/`（**自持本组
  实现，不 import internal/agent**）→ 产物 `<插件目录>/bin/<name>.exe`
- 各组插件 JS 的 execute 统一 `ctx.binary.exec(t.name, args)`（缺省 bin=插件名
  → 定位本插件目录 bin/ 下的 exe；`opts.bin` 跨插件共用仅历史形态，已无引用）
- **改实现**：改 `plugins-src/plugins/<name>/impl/*.go` → `go build -o
  .pair/plugins/<name>/bin/<name>.exe ./plugins-src/plugins/<name>` → 重启 → 本组生效
  （主程序 exe 无需重编译）
- tool-binary 只是其中普通一员（承载 binary 组 inspect_binary/write_binary/
  binary_strings/find/patch/info/hash/entropy 8 个工具，2026-08-16 并入
  原 tool-binary-re 逆向 6 工具），**不是**统一容器——各插件实现互相独立
- hostTool 形态（工具-system）：SystemTool（update_tasks/tool_stats/
  history_*）+ Skills/MCP/市场/提交信息——依赖宿主会话状态，execute 走
  `ctx.hostTool.exec`；ask_user/task_create 经**会话桥**（session_bridge.go）
  按 _convID 路由回会话（见下方 tool-system 条目）

当前 execute 形态分布（生成器 tool_plugin_gen.go 的 binary 字段控制）：
- `self`（独立二进制）：tool-binary、tool-bug、tool-codegraph、
  tool-codegraph-extra、tool-debug、tool-git、tool-harness、tool-lsp、
  tool-memory、tool-office、tool-project-info、tool-screenshot、tool-search、
  tool-verify、tool-vision、tool-web、tool-web-debug（17 组，各自 bin/ 下的 exe）
- `""`（hostTool）：tool-system（会话绑定 SystemTool + Skills/MCP/市场/提交）
- 手工迁移（不在此列）：tool-core（JS 原生 impls）、tool-shell、tool-web（webFetch 直连）

## 三层工具实现（从易到难，用户可改程度递增）

1. **api 壳（hostTool 代理）**：`index.js` 只声明 schema，execute 调
   `ctx.hostTool.exec` 复用主程序内 Go 执行器——只能改描述/参数。
2. **JS 原生实现**：`index.js` 的 execute 直接写 JS（jsToolToGo 支持
   `execute: (args) => result`），用 `ctx.fs` / `ctx.bash` / `ctx.web` 等
   宿主服务实现逻辑——纯 JS 可改，无需重新编译。
3. **独立二进制**：依赖 Go 内核/系统能力（PE 解析、哈希、文档转换、索引等）
   → 独立 Go 项目 `plugins-src/plugins/<name>/` → 编译产物进 `<插件包>/bin/`
   → JS 壳 `ctx.binary.exec` 调度。改二进制源码 → `go build` → 重启生效。

## 修改指南（用户自助）

### 改工具行为（示例：tool-binary）
```bash
# ① 改实现（JS 壳 or 独立二进制源码）
vim .pair/plugins/tool-binary/index.js              # 改 api/调度
vim plugins-src/plugins/tool-binary/impl/binary.go          # 改真实实现（读/写）
vim plugins-src/plugins/tool-binary/impl/binary_re.go       # 改真实实现（逆向分析）
# ② 重新编译二进制（改了 impl 时）
go build -o .pair/plugins/tool-binary/bin/tool-binary.exe ./plugins-src/plugins/tool-binary
# ③ 重启 companion → 生效
```

### 改前端 UI
```bash
# 源码位于项目根 plugins-src/ui-app/（2026-08-17 迁移：不再放 .pair 内；Vue3 + Vite；node_modules 为 junction
# 指向 cmd/companion/web-ui/node_modules，勿删）
# ① 改组件/壳
vim plugins-src/ui-app/src/components/*.vue
# ② 构建 7 个 UI 区域插件包（产物 → .pair/plugins/ui-*/assets/）
node scripts/build-ui.mjs
# ③ 构建壳（产物 → .pair/assets/runtime/web/，宿主外部优先加载）
cd plugins-src/ui-app && node_modules/.bin/vite build
# ④ 重启 companion → 生效
```

### 通用工具组迁移（内置 Go 组 → 磁盘插件）
```bash
go run -tags toolsgen ./dev/tool_plugin_gen   # 幂等：已有插件不覆盖，Go 侧变更重跑同步
```

## 其他目录

- `.pair/assets/runtime/` — 运行时资源（cordis.bundle.js/bridge_node.js/
  ide_ref*/web 前端产物），外部优先 + embed 兜底（见其 README）
- `.pair/toolsets/` — 工具集（插件组合包）

## 插件包 config 通道（apply(ctx, config)）

磁盘插件包的 `package.json` 可带 `"config": {...}` 字段，装载时透传给
`apply(ctx, config)` 第二参（GlobalPluginPackage.Config → def.config），
改 config 重启生效、无需重编译。用例：

- **`agentloop/`**（2026-08-16）：Agent 循环装配器（LoopFactory 单槽位 JS
  装配器）。Loop 核心在 Go（会话/持久化深度耦合），本插件做「参数级装配」：
  apply 时 `ctx.loopFactory.register(apply)`，每次创建循环（会话 Start /
  自闭环 Run 统一走 CreateLoop）时收到装配快照、返回非空字段覆盖。config
  支持 systemAppend（追加系统提示词）/maxIterations/maxContextTokens/
  autonomous/maxAutonomousMinutes/checkpointInterval/reviewMode/
  reviewBlacklist/reviewWhitelist；停用插件自动还原默认工厂（Loop 不受影响）。
  （快照字段实现：internal/agent/jsplugin_loopfactory.go）
- **`tool-system/`**（2026-08-16 扩容）：系统内部工具 16 个——SystemTool 组
  （update_tasks/tool_stats/history_search/history_list/history_count）
  + Skills（skill_list/load_skill/load_skill_resource/skill_write/skill_delete）
  + MCP（mcp_list/mcp_add/mcp_remove）+ 市场（marketplace_search/
  marketplace_install）。execute 全走
  `ctx.hostTool.exec`（宿主 Go 执行器：编排在插件、能力在宿主）。生成器
  tool_plugin_gen.go 的 tool-system 组白名单同步维护（含 11 个新工具），
  `go run -tags toolsgen ./dev/tool_plugin_gen` 重跑不丢失。**ask_user/
  task_create 已插件化**（2026-08-16 会话桥机制）：Loop ctx 链携带 convID
  （SessionManager.Start runCtx 注入）→ JS 工具包装（jsToolToGo）复制 args
  注入 `_convID` 内部键 → 插件 execute 经 ctx.hostTool.exec 路由回宿主 →
  hostTool 路由执行器（session_bridge.go archiveSessionTools）→ SessionBridge
  （web 层注入，WaitAnswer 读会话 askCh / GetWorkspaceRoot）→ 会话按 convID
  精确路由，多会话并发不串。SessionManager.Start 检测 reg 已存在同名工具时
  不再注册会话级版本（插件优先、宿主兜底）。ask_user 提问卡片/回答交互
  （message_store ask_user segment + /api/answer → SendAnswer）不变。

---

## 版本发布（2026-08-17 新增）

插件体系支持**整体打包发布**（版本对齐 UI 1.3.0）：

### UI 代码专门目录

- **`plugins-src/ui-app/`**（项目根，2026-08-17 从 .pair 迁出）—— UI **源码专门目录**（壳 ShellApp + 7 个区域入口
  ui-main-*.js + 60+ Vue 组件 + docs 文档）。Vite 工程（vite.config.js +
  package.json），`npm run build` 产壳到 `.pair/assets/runtime/web/`；
  `node scripts/build-ui.mjs` 产 7 个区域包到 `ui-<region>/assets/`。
- **`ui-*/`** —— UI 区域插件包（titlebar/activitybar/sidebar/editor/
  right-panel/statusbar/modals + statusbar-conn），client.js 为浏览器半，
  assets/ 为构建产物，随发布包分发。

### 发布工程（package.json）

`.pair/plugins/package.json` 是插件体系发布级配置：

```bash
# 构建 7 个区域 UI 插件包（ui-<region>/assets/）
npm run build
# 构建壳（plugins-src/ui-app/ → .pair/assets/runtime/web/）
npm run build:shell
# 打包发布（npm pack → release/PairCode-plugins-<version>.tgz）
npm run pack
# 版本号升级（如 1.3.0 → 1.4.0）
npm run version:bump -- 1.4.0
```

- **files 字段**：全部 33 个插件目录（agentloop/core-api/tool-*/ui-*/web-api）
  + UI 源码工程（plugins-src/ui-app/）+ 模型依赖清单（config/models.json +
  config/settings.template.json）+ README.md —— 即「插件 + 插件代码 + 依赖
  模型」整体入包。
- **依赖声明**：dependencies 覆盖 UI 运行时依赖（vue/codemirror/xterm/
  marked/mermaid/pinia/vue-router），devDependencies 为构建依赖（vite/
  @vitejs/plugin-vue）；`node_modules`（junction）经 .npmignore 排除，包体
  只含源码与插件自包含资源。

### 依赖的模型（随包分发）

发布包 **config/** 目录携带模型依赖：

| 文件 | 内容 |
|------|------|
| `config/models.json` | 依赖的服务商 + 模型清单：deepseek（deepseek-r1/v4-pro/v4-flash）、anthropic（claude 系列）、kimi（kimi-k3）、openai-compatible/custom |
| `config/settings.template.json` | 配置模板（默认主模型 deepseek-v4-flash、规划/执行/审核模型、压缩模型等） |

宿主启动时 `EnsureModelList()` 确保 models.json 存在（缺失自动写入默认），
新环境部署发布包时把 config/models.json 复制到宿主 config/ 即恢复完整模型
列表。