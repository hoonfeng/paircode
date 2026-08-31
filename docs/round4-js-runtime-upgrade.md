# Round4 ② JS 插件运行时升级：定案与实施计划（t1 · analyst · 2026-09）

> 本文件为 **runtime-complete-round4** 团队 t1（requirements）只读分析交付物之二。
> 目标（团队目标原文）：goja 非完整运行时（缺 require/fs/path/process/事件循环等 Node 能力）
> 导致 外部插件（Node 形态）不能直接使用；评估并实施完整运行时方案（候选：goja+goja_nodejs
> 借用代码、全 Node 桥运行、混合按需路由）；外部参考插件（dsh-agent-teams 等）可在升级后
> 运行时直接装载运行。
> 证据来源：本仓源码（internal/agent/jsplugin.go、node_bridge.go、bridge_node.js、node_plugins.go、
> npm_plugin.go、tscompile.go、builtin_stdlib.go）+ 外部仓库（外部实现的
> packages/extensions/cordis-host-runner、packages/boot/app-boot + dsh-agent-teams v0.1.14）
> 逐文件实测复核。

---

## 1. 现状：双轨运行时与各自边界（实测）

### 1.1 goja 沙箱轨（默认，`.pair/plugins/*/index.js` + 工具集 + cordis.patch 代码插件）

- 装载：`LoadGlobalPlugins`（toolset.go:562）→ `applyGlobalPluginDir`（:594）→ `DefineJSCodeFull`
  → `LoadJSDynamic`（jsplugin.go:2853）→ `evalJSPlugin`（jsplugin.go:2809，`(async () => {...})()` 包装）。
- 沙箱纪律（jsplugin.go:1-19 头注 + :2738-2754）：**与 外部动态插件沙箱逐条对齐**——
  `require/setTimeout/setInterval/setImmediate/clearTimeout/clearInterval/fetch` 全部 trap 报教学错误；
  定时能力经 `ctx.timeout/ctx.interval`（jsplugin.go:490-491）；网络经 `ctx.web`；文件经 `ctx.fs`。
  外部实现 `cordis-host-runner/src/sandbox.ts:96-119` 的 `NODE_API_REDIRECTS` 与此一一对应
  （gou-ide 是 2026-08 主动对齐移植）。
- 模块解析：esbuild 内联（tscompile.go）——相对导入 bundle，非相对包导入 mock 空模块
  （mockPackageOnResolve :128-149），内置库仅 `path/events/util` 三个纯 JS mini 实现
  （builtin_stdlib.go:20-205）。
- 运行期：goja 非线程安全 → `vm.Lock/Unlock` 协作锁（goja fork 补丁 cb1a573，
  2026-08-29 已收编进本仓 `goja/`，module `github.com/hoonfeng/paircode/goja`）；
  `vm.Interrupt` 超时中断（runJSWithTimeout，jsplugin.go:100-116）。

### 1.2 Node 桥轨（`.pair/cordis/node/`，npm 插件）

- 触发：`nodePluginNeedsNode`（node_plugins.go:73-84）——manifest `dependencies` 非空
  或 peer `@cordisjs/core` 主版本 ^4 → Node 桥；或 cordis.patch.json 条目 `config.runtime=="node"`。
- 形态：spawn 真实 `node bridge.js`（bridge_node.js，302L，embed + 外置运行时资源），
  stdin/stdout JSON Lines 协议（node_bridge.go:42-670）；崩溃自动重启 ≤3 次。
- Node 侧装载（bridge_node.js:225-267）：`import('@cordisjs/core')`（**cordis 3**）→ `new Context()`
  → `decorateCtx`（:150-189，注入 `ctx.tools/fs/web/bash/store/loop/workspaceRoot` + 拦截 `ctx.provide`
  把服务方法暴露成 `<service>_<method>` 工具）→ `applyFn(ctx)`。
- ctx 服务转发（node_bridge.go:430-618 `mapBridgeService`）：fs(read/write/list/exists/mkdir/remove)、
  web(fetch/search)、bash(exec)、store(read/append)、loop(info/snapshot)。

### 1.3 缺口（为什么 dsh-agent-teams 现在不能直接装载）

外部参考插件 `@nanmicoder/dsh-agent-teams` v0.1.14（本地安装）实测：

| 维度 | dsh-agent-teams 要求 | 当前宿主 | 缺口 |
|---|---|---|---|
| 模块形态 | npm 包（`main: lib/index.js`，ESM），peerDeps 含 `@deepseek-ai/cordis ^4` + 11 个 `@deepseek-ai/dsh-*` | Node 桥 npm install 后可解析（npm 7+ 自动装 peer），但桥用 cordis3 `@cordisjs/core` Context | **cordis 4 API 面**（ctx.inject/inject 数组/logger 等 cordis4 语义） |
| inject 服务 | `['tools','llm','subagents','systemPrompt','agents']`（index.ts:64）+ 可选 `commands`（:218）+ `ctx.get(webServer/workspaceRegistry)`（:232-233）+ `ctx.logger`（tools.ts:668） | 桥 ctx 仅 tools/fs/web/bash/store/loop/workspaceRoot；goja 侧有 agents/llm/commands（jsplugin.go:3029-3037）但 **Node 桥无这些门面** | **外部服务面门面缺失**：agents/subagents/llm/systemPrompt/commands/logger/webServer/workspaceRegistry |
| 事件 | `ctx.on('agent/pre-step')`（command.ts:124）、`ctx.on('agent/status')`（scheduler.ts:488）、`ctx.on('agent/request-error')`（members.ts:316）、`ctx.on('internal/service')`（index.ts:520） | 桥协议无事件订阅通道（node_bridge.go 协议仅 invoke/service/result/log/ready/tool/ping） | **agent 事件桥缺失** |
| Agent 句柄 | `agent.inject(createUserMessage(...))`（tools.ts:660）、`agent.steer`、`childCtx.on` | 桥无 agent 句柄方法映射；goja 侧 ctx.agents.start/followup/stop/status/list/lastText/ready/fork/report（jsplugin_agents.go） | **inject/steer/followup 映射** |
| 客户端半 | `dsh.client.inject` 7 个 @deepseek-ai/dsh-client-*（UI 面板） | 无 client 半渲染（gou-ide UI 槽位由 ui-* 插件承载） | **可裁剪**（host 功能不依赖 client 半；面板可复用现有 agent-teams 移植版 client.js 或后续专项） |

**路由判定缺口（第一个失败点，实测）**：`npmMarketInstall`（npm_plugin.go:307）按
`nodePluginNeedsNode(manifest)`（node_plugins.go:73-84）分流 goja/Node 桥——该函数只查
`dependencies` 非空或 peer key `cordis`/`@cordisjs/core` ^4。而 dsh-agent-teams 的 package.json
**无 dependencies 字段**、peer 键是 **`@deepseek-ai/cordis`**（^4.0.1-rc.1）→ 判定返回 false →
**错误路由到 goja 轨**（npmPackageMain 取 `lib/index.js` → esbuild bundle + @deepseek-ai/* mock 空模块
→ 运行期 cordis4 API 全 undefined → 装载失败）。即使补上路由，Node 桥也装不上 cordis4 插件（§1.3
缺口 1-4）。因此本轮须同时：① `nodePluginNeedsNode` 增加 `@deepseek-ai/cordis` ^4 peer 判定
（路由到 dsh 分支）；② 桥实现 外部门面。

补充实测：goja 轨不可能直跑该包——其 `lib/index.js` 是 ESM bundle，imports `@deepseek-ai/cordis`
等 12 个包；即便 esbuild 全部内联，cordis4 Context + dsh-* 服务面在 goja 内也不存在。

---

## 2. 候选方案对比与定案

### 候选 A：goja + goja_nodejs 借用代码（github.com/dop251/goja_nodejs，MIT）

- **能力增量**：`require` 注册表 + `fs/path/process/os/buffer/console/url/util/events` + eventloop
  （setTimeout/setInterval/setImmediate + 微任务/macrotask 队列）。`fs` 走真实 Go fs。
- **实测局限（对本目标致命）**：
  1. **不能直跑 dsh-agent-teams**——缺 cordis4 + dsh-* 服务面（§1.3 缺口 2-4 与运行时无关）；
     即使装了 goja_nodejs，插件 import 的 `@deepseek-ai/cordis` 仍无法解析/运行（cordis4 依赖
     Fiber/Inject 等宿主组合语义）。
  2. 与沙箱纪律冲突：外部动态插件沙箱**故意**禁 Node API（sandbox.ts:96-119），gou-ide 已对齐
     （jsplugin.go:2738）；引入 require/fs 等于破坏「插件只能经 ctx 服务触达能力」的审计面。
  3. fork 兼容风险：goja 是仓库自有 fork（`goja/`，module `github.com/hoonfeng/paircode/goja`，
     非 `github.com/dop251/goja`）；goja_nodejs 直接 import 上游路径，需要 import 改写 + API 对齐验证
     （fork 保留了 Runtime/Object/Interrupt 等，但需编译实测）。
  4. 借用规模：goja_nodejs 的 fs（含 watcher/stream 面）约 6-7k 行，长期随上游维护成本高。
- **结论**：**不采纳为本轮主方案**。保留为 P2 可选加分项（仅当出现「纯 Node stdlib 形态、无
  cordis 依赖」的第三方插件时再评估按需子集借用：path/events/util 已有 mini 实现，可补 fs/process
  mini 面），许可证 MIT 可借用但需在 THIRD_PARTY 声明。

### 候选 B：全 Node 桥运行（扩展现有 node_bridge，推荐主体）

- **能力增量**：完整 Node（require/fs/path/process/事件循环/npm 生态），桥已存在、协议已稳、
  崩溃重启/工作区隔离/工具注册面全通（node_bridge.go）。
- **需补 4 块**（对应 §1.3 缺口）：
  1. **cordis4 装载**：bridge_node.js 的 loadPlugins 增加 `runtime=="dsh"` 分支——`import('@deepseek-ai/cordis')`
     `new Context()`（cordis4），插件对象形态兼容（candidate.apply/pluginObj 逻辑复用）；
  2. **外部服务面门面**（decorateCtx 扩展）：`ctx.agents`（start/followup/stop/status/list/lastText/
     fork/report → 已有 Go 服务）、`ctx.subagents`（spawn/fork/list → 映射 agents）、`ctx.llm`
     （chat/models/current → 已有 llm 服务）、`ctx.systemPrompt`（section → 已有）、`ctx.commands`
     （register/list/run → 已有）、`ctx.logger`（info/warn/error → log 消息）、
     `ctx.get('webServer')`（register → ctx.http 宿主路由）、`ctx.get('workspaceRegistry')`（get →
     ctx.app 工作区信息）；服务调用走现有 `svc()` 协议（Go 侧 mapBridgeService 扩展新 svc 名）；
  3. **agent 事件桥**：协议新增 `subscribe`（插件声明要监听的事件名）与 `event`（Go → Node 推送），
     Go 侧把 `agent/pre-step`（loop_hooks/loop.go pre-step 点）、`agent/status`（subagent_registry
     状态变更）、`agent/request-error`、`internal/service` 等事件转发；Node 侧 `ctx.on` 接线；
  4. **agent 句柄方法**：`ctx.agents` 返回的 agent 对象增加 `inject(message)`（→ Go followup/inject
     服务方法）、`steer`、`on(event, fn)`（子上下文事件订阅，走订阅桥）。
- **安装路径**：复用 `marketInstallNPMPluginNode`（node_plugins.go:131-207，npm install + plugins.json
  + patch 记录 + 重启桥）；plugin spec 记录加 `runtime:"dsh"` 标记；peer deps（@deepseek-ai/cordis、
  dsh-* 11 包）由 npm 7+ 自动装入桥 node_modules（@cordisjs/core 保留供 cordis3 插件共存）。
- **风险**：npm 安装体积（dsh-* 依赖树 ~数十 MB，一次性）；Node 版本（包 engines `^22.19.0||>=24`，
  本机 v22.17.0 差一档，npm 默认仅警告，需实测装载；失败时降级提示）；插件崩溃隔离（已有重启机制）；
  cordis3/cordis4 双 Context 共存（各自独立 import，无冲突）。

### 候选 C：混合按需路由（推荐 = B 主体 + 路由规则）

- 路由规则（与现有 `nodePluginNeedsNode` 同构扩展）：
  - 无 dependencies + 无 cordis4 peer → **goja 沙箱**（现状不变，沙箱纪律保持）；
  - dependencies 非空 / peer `cordis`/`@cordisjs/core` ^4 / `runtime:"node"` → **Node 桥 cordis3**（现状不变）；
  - peer `@deepseek-ai/cordis` ^4 或 `runtime:"dsh"` → **Node 桥 cordis4 + 外部门面**（本轮新增）；
- 现状插件零行为变化（agent-teams 移植版、tool-* 28 插件仍走 goja；npm cordis3 插件仍走旧桥）。

### 定案

> **主方案 = 候选 C（混合按需路由）：在现有 Node 桥之上实施 B 的 4 块扩展，goja 轨保持不动；
> 候选 A（goja_nodejs 借用）降级为 P2 可选加分项，本轮不实施。**
>
> 理由（文件级证据）：① dsh-agent-teams 实测为 cordis4 + dsh-* 服务面依赖的 npm bundle
> （§1.3 表），goja_nodejs 只能补 Node stdlib、补不了服务面——B/C 是**唯一能达成「直接装载运行」**
> 的路径；② Node 桥基础设施已存在且生产验证过（node_bridge.go 协议/重启/隔离），扩展成本
> 远低于重造；③ goja 轨的 Node-API 禁制是对齐 外部自身沙箱纪律的刻意设计（sandbox.ts:96-119），
> 不应为兼容而破坏；④ 路由判定已存在（nodePluginNeedsNode），加一档 runtime 分支是自然演化。

---

## 3. 实施计划（t2 inScope 建议，按序）

| 序 | 工作包 | 改动文件 | 验收 |
|---|---|---|---|
| 1 | **桥协议扩展**：`subscribe`/`event`/`agentSvc` 消息 + 新 svc 名（agents/subagents/llm/systemPrompt/commands/logger/workspace）路由 | `internal/agent/node_bridge.go`（协议解析 + mapBridgeService 扩展 + EventBus→桥事件转发）、`internal/agent/bridge_node.js`（decorateCtx 门面 + 订阅表 + ctx.on 接线） | `TestNodeBridgeProtocolPing` 回归 + 新增 `TestNodeBridgeDSHFacades`（订阅/事件/服务映射单测） |
| 2 | **cordis4 装载分支**：`runtime=="dsh"` → `@deepseek-ai/cordis` Context | `internal/agent/bridge_node.js` loadPlugins 分支；`internal/agent/node_plugins.go`（spec 记录 runtime 字段 + plugins.json 结构兼容 + **`nodePluginNeedsNode` 增 `@deepseek-ai/cordis` ^4 peer 判定**） | `node --check bridge_node.js`；桥装载日志显示 cordis4 Context 分支；`TestNodePluginNeedsNode` 新增 dsh 包判定用例 |
| 3 | **外部插件安装路径**：`marketInstallNPMPluginNode` 支持 dsh 标记（peer 安装提示、runtime 透传）；patch 记录 `runtime:"dsh"` | `internal/agent/node_plugins.go`、`internal/agent/npm_plugin.go`（安装输出文案） | 装 `@nanmicoder/dsh-agent-teams@0.1.14` 成功、plugins.json 记录、桥重启装载 |
| 4 | **Go 侧服务实现对接**：agents/subagents/llm/systemPrompt/commands/logger 服务方法 → 现有 Go 实现（jsplugin_agents.go/llm/commands/loop_hooks 复用）；agent 句柄 inject/steer 映射（subagent_registry followup + message 注入）；agent/status、agent/pre-step 事件源接线 | `internal/agent/node_bridge.go`（direct 处理器）、`internal/agent/subagent_registry.go`/`loop.go`/`loop_hooks.go`（事件点）、`cmd/companion/web_server.go`（如需要 SetNodeBridgeManager 同款注入） | 新增 `TestNodeBridgeAgentEvents`（事件订阅收到 agent/status 变更） |
| 5 | **端到端验证**：装 dsh-agent-teams → 桥装载 → `agent_teams_*` 13 工具注册进 Registry → 冒烟调用（agent_teams_status/create 两阶段）→ 卸载还原 | 验证脚本/测试 `npm_plugin_e2e_test.go` 同型 | `TestNodeBridgeDSHPluginE2E`（或 t3 手工冒烟）：工具清单含 13 个 agent_teams_*，零 FATAL，卸载后工具消失 |
| 6 | **文档/许可证**：THIRD_PARTY 声明（@deepseek-ai/cordis、dsh-*、@nanmicoder/dsh-agent-teams 均 MIT）；docs 更新 | docs/round4-js-runtime-upgrade.md 回填实施记录 | 许可证清单齐全（§5） |

**范围声明**：
- client 半（Web 面板）**不在本轮**——host 工具面完整即可「装载运行」；UI 复用现有移植版
  agent-teams 的 client.js（/api/agent-teams/teams 已存在）或后续前端专项。
- 不做跨进程插件状态持久化（团队状态本就在磁盘 .agent-teams/，桥重启后插件重新 apply 可恢复；
  与 cordis3 插件现状一致）。
- goja_nodejs 借用（候选 A）本轮不实施，记录为 P2。

**验证基线**（t3 沿用）：`go build ./...` + `go vet ./...`；`go test ./internal/agent/` 环境依赖清单
沿用 round3 §10.1；`node --check bridge_node.js`；npm 安装需网络（沙箱外/队长全权限通道执行
`npm install` 步骤；桥启动本身无需网络）；`TestToolLandingMatrix` 回归（新增 agent_teams_* 工具
若加入工具集，需同步 landing 表——建议保持默认不进工作区工具集，由插件面板管理，避免 295 断言漂移）。

---

## 4. 风险与回滚

1. **npm 依赖树**：@deepseek-ai/cordis + 11 个 dsh-* peer（含 react 等 optional）——npm 7+ 自动装
   non-optional peer；react 等 optional peer 缺失不阻断（插件源码 import 的包须在 node_modules，
   否则 cordis4 Context 装载失败 → 桥 log 明确报错、插件跳过，**不影响其他插件**——loadPlugins
   逐条 try/catch 已隔离）。回滚 = 从 plugins.json 移除 spec + 重启桥（uninstallNodePlugin 已实现）。
2. **Node 版本**：engines ^22.19.0 vs 本机 22.17.0——npm 仅警告；若运行时 API 差异导致装载失败，
   升级本机 Node 或记录降级提示（错误信息明确）。回滚同上。
3. **协议兼容**：新增消息类型向后兼容（旧桥脚本不识别 subscribe/event 时忽略未知字段——
   需在 bridge_node.js 保持 JSON 解析容错）；Go 侧 mapBridgeService 未知 svc 已返回明确错误。
4. **事件风暴**：agent/status 事件频率高——订阅桥按插件声明过滤（subscribe 白名单），
   Go 侧只转发已订阅事件，防协议撑爆（参考 invokeTool 3 分钟超时 + 4MB 行缓冲既有防护）。
5. **双 cordis 共存**：cordis3（@cordisjs/core）与 cordis4（@deepseek-ai/cordis）不同包名，
   node_modules 平铺共存无冲突；装载分支按 runtime 选择 Context，不做混用。

---

## 5. 许可证核查表（借用/依赖代码）

| 组件 | 版本 | 许可证 | 用途 | 声明要求 |
|---|---|---|---|---|
| goja（gou-ide 自有 fork） | fork of dop251/goja | MIT（goja/LICENSE，Copyright 2016 Dmitry Panov / 2012 Robert Krimen） | 沙箱运行时（已在用；2026-08-29 收编进本仓 `goja/`，module `github.com/hoonfeng/paircode/goja`） | 保留 LICENSE（已随包） |
| goja_nodejs（候选 A，本轮不实施） | latest | MIT | 若实施需 fs/path/process 等借用 | THIRD_PARTY 声明 + 保留头注 |
| @cordisjs/core（cordis3） | 3.18.1（bridge 安装） | MIT（cordis 项目） | 现有 Node 桥 Context | npm 安装自带 license 字段；桥 package.json 记录 |
| @deepseek-ai/cordis（cordis4） | ^4.0.1 | MIT（外部实现 LICENSE，2026 DeepSeek） | 本轮 外部装载 Context | 同上 |
| @deepseek-ai/dsh-*（11 包） | 0.1.0-rc.x | MIT | 外部服务面 peer 依赖（npm 自动装） | 同上 |
| @nanmicoder/dsh-agent-teams | 0.1.14 | MIT（LICENSE，2026 程序员阿江） | 参考插件直跑验证 | THIRD_PARTY 记录 |
| cordis.bundle.js（embed） | @cordisjs/core 3.18.1 bundle | MIT | goja 内 cordis 运行时（已在用） | 保留源注释（已在 assets 头注） |

> 全部组件均为 MIT，无传染性义务；借用代码需保留版权头注并在 THIRD_PARTY_NOTICES 类文档登记
> （建议 t2 在仓库根新增 `THIRD_PARTY_NOTICES.md` 汇总以上条目，含 goja/goja_nodejs 两个借用源）。

---

## 6. 交付物

- 本文件：候选对比 + 定案 + 实施计划（t2 依据）。
- `docs/round4-go-core-audit.md`：Go 核心存量审计（配套 ①）。
- t2 实施后回填：工作包 1-6 实施记录 + t3 验证 + t4 审查 + t5 集成结论（沿用 plugin-round3-plan.md §10-§11 格式）。

---

## 7. t2 实施回填（2026-09 · engineer）

**状态：已实施（候选 C 全部 6 个工作包）**，实施记录见 `docs/runtime-upgrade-plan.md`；
验证测试见 `internal/agent/dsh_bridge_test.go`（外部服务面/事件桥/E2E）与
`internal/agent/node_bridge_test.go`（runtime 判定/plugins.json 兼容）。

要点：
- ① 路由判定：`nodePluginNeedsNode` 增 `@deepseek-ai/cordis` ^4 peer；
  新增 `nodePluginRuntime` → `dsh|node|""`（dsh 优先）。
- ② cordis4 装载：`bridge_node.js` `runtime=="dsh"` → `@deepseek-ai/cordis`
  `new Context()` + `decorateDshCtx`（agents/subagents/llm/systemPrompt/commands/
  logger/effect/on/inject/get 门面）；cordis3 轨不变。
- ③ 安装路径：plugins.json 条目 `{spec, runtime}`（旧字符串兼容）；
  `npmInstallDshPeers` 显式安装非 client 的 `@deepseek-ai/*` peer（全 optional
  npm 不自动装）；patch 记录 `runtime:"dsh"`。
- ④ 服务对接：`node_bridge.go` `dshService`（agents→SubAgentRegistry、
  llm→模型目录、systemPrompt→PluginHost 段、commands→命令表+cmdrun 回 Node）；
  `subagent_registry.go` agent/status 事件源（running/idle/stopped）。
- ⑤ E2E：`TestNodeBridgeDSHPluginE2E`（npm 安装 dsh-agent-teams@0.1.14 →
  13 个 agent_teams_* 工具 → create/status/delete 冒烟 + 状态落盘；
  npm 网络环境不满足自动 Skip）。
- ⑥ 许可证：`THIRD_PARTY_NOTICES.md` 新建（全部 MIT）。
- **P2 遗留**：agent/pre-step 拦截链与 agent/request-error 事件源（宿主循环语义
  改造，记录不阻塞）；webServer/workspaceRegistry 门面（宿主 UI 槽位承载）；
  goja_nodejs 借用（候选 A）未实施。
- t3 验证 / t4 审查 / t5 集成结论：由对应成员回填（沿用 §10-§11 格式）。
