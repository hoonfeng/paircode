# Runtime 升级实施记录（Round4 ②：JS 插件运行时完整化）

> 本文件为 **runtime-complete-round4** 团队 t2（implementation）实施记录。
> 定案依据：`docs/round4-js-runtime-upgrade.md`（候选对比与方案 C 定案）。
> 状态：**已实施（2026-09）**；验证/审查/集成结论由 t3/t4/t5 回填。

---

## 1. 方案回顾（候选 C：混合按需路由）

| 轨 | 触发条件 | 运行时 | 本轮改动 |
|---|---|---|---|
| goja 沙箱 | 零 npm 依赖、无 cordis4 peer | goja（Node API 禁制，沙箱纪律保持） | **不动**（零回归） |
| Node 桥 cordis3 | dependencies 非空 / peer `cordis`/`@cordisjs/core` ^4 | `@cordisjs/core` Context | 协议向后兼容扩展（不破坏） |
| **Node 桥 DSH（本轮新增）** | peer `@deepseek-ai/cordis` ^4 | `@deepseek-ai/cordis` Context + DSH 门面 | bridge_node.js / node_bridge.go / node_plugins.go |

候选 A（goja+goja_nodejs 借用）经 t1 评估**不采纳**（缺 cordis4+DSH 服务面、
与沙箱纪律冲突、fork import 改写风险），降级 P2——本轮未引入任何 go.mod 新依赖
（go.mod / go.sum **零变化**）。

---

## 2. 工作包实施明细（按 t1 §3 计划 1-6）

### 工作包 1：桥协议扩展（subscribe/event/cmdrun + DSH 服务面路由）

**`internal/agent/bridge_node.js`**（302L → 700L）：
- 协议新增：
  - Node → Go：`{"t":"subscribe","plugin","events":[...]}`（插件 ctx.on 订阅白名单）
  - Go → Node：`{"t":"event","name","payload"}`（host 事件按订阅转发）
  - Go → Node：`{"t":"cmdrun","id","name","args"}`（宿主执行插件命令 handler）
- `svc()` 消息携带 `plugin` 字段（命令归属用）。
- `invoke` 载荷新增 `convId`/`wsRoot` → Node 侧构造 `exec.agent` 句柄
  （DSH 工具 `requireCaptain(exec)` 消费；`agent.session.header.cwd` =
  调用会话工作区根）。

**`internal/agent/node_bridge.go`**（670L → 1045L）：
- readLoop 协议解析：`subscribe` 消息 → `handleSubscribeMsg`（白名单表 `subs`）。
- `emitBridgeEvent(name, payload)`：白名单外零开销（锁 + map 查询），
  防事件风暴（t1 风险 4）。
- `dshService(svcName, method, args, plugin)`：新服务面（见工作包 4），
  handled=false 时回落既有 `mapBridgeService`（fs/web/bash/store/loop 不动）。
- `runNodeCommand`：cmdrun 消息 + pending 通道（90s 超时）。

### 工作包 2：cordis4 装载分支 + 路由判定

**`internal/agent/bridge_node.js`**：
- plugins.json 条目支持 `{spec, runtime}`（旧字符串兼容）；`runtime=="dsh"` →
  `import('@deepseek-ai/cordis')` → `new Context()` → `decorateDshCtx` →
  `applyFn(ctx, {})`；其余走 cordis3 既有路径（双 Context 独立 import 共存）。
- `decorateDshCtx`（DSH 门面，见工作包 4）+ `ctx.inject`（依赖全就绪即同步回调，
  防 cordis4 原生 inject 因服务缺失而 pend fiber）。

**`internal/agent/node_plugins.go`**：
- `nodePluginNeedsNode` 增 `@deepseek-ai/cordis` ^4 peer 判定（t1 首个失败点）。
- 新增 `nodePluginRuntime(manifest)` → `"dsh" | "node" | ""`（dsh 优先）。
- plugins.json 新格式 `nodePluginEntry{Spec, Runtime}` + 旧格式 Unmarshal 兼容；
  `Specs()` 兼容旧消费者（bridgeLoadedPlugins）。

### 工作包 3：DSH 插件安装路径

**`internal/agent/node_plugins.go`**：
- `marketInstallNPMPluginNode`：runtime 判定 → plugins.json 条目带 runtime、
  patch 记录 `config.runtime="dsh"`、安装文案标注运行时。
- `dshPeerSpecs` / `npmInstallDshPeers`：dsh 插件 peer 全 optional（npm 7+ 不自动装）
  → 显式 `npm install` 非 client 的 `@deepseek-ai/*`（cordis + dsh-* + schemastery；
  client 半与 react 由宿主 UI 槽位承载不装入）。
- `uninstallNPMPlugin` runtime 判断扩展 `node || dsh`。

### 工作包 4：Go 侧 DSH 服务面 + agent 事件桥

**`internal/agent/node_bridge.go` `dshService`**（全部映射现有 Go 实现，
与 goja 轨 jsplugin_agents.go 同源）：

| 服务 | 方法 | Go 后端 |
|---|---|---|
| `agents` | get/status/list/running/ready/start/fork/followup/inject/steer/cancel/stop/report/lastText | SubAgentRegistry（SpawnSubAgent/FollowupSubAgent/StopSubAgent/ListSubAgents/ReportSubAgent…） |
| `subagents` | getProvider/list/startContinuable/followup/interrupt | 同上（spawn provider 描述） |
| `llm` | models/current/listModels/resolveCallConfig | SubAgentModels/SubAgentCurrentModel |
| `systemPrompt` | section | PluginHost.AddSystemPromptSection |
| `commands` | register/unregister/list/run | RegisterHostCommand（handler → runNodeCommand 回 Node）/RunHostCommand |
| `logger` | （Node 侧 console 通道，无往返） | — |

**`internal/agent/subagent_registry.go`**：`emitAgentStatusEvent`（载荷
`{agent:{id,status,session.header.cwd}, status}` 对齐 DSH scheduler 消费面）在
SpawnSubAgent（running）/ FollowupSubAgent（running）/ StopSubAgent（stopped）/
轮次结束 idle 边 四处置出 → `emitBridgeEvent("agent/status", ...)`。

**agent/pre-step / agent/request-error**：桥协议与订阅面已就绪（插件可注册
handler 不报错），但宿主循环 pre-step 拦截链与请求错误事件源为**宿主循环语义
改造**（涉及 next() 链），本轮记录为 P2 缺口（不影响装载运行与工具实测——
gesture boundary 仅在有 slash 命令触发时才有意义，工具路径完全可用）。

### 工作包 5：端到端验证（t2 侧）

- `internal/agent/dsh_bridge_test.go`（新增）：
  - `TestDSHBridgeServices`：dshService 映射与错误路径
  - `TestDSHBridgeSystemPromptCommand`：systemPrompt 段注册 + 命令表注册/卸载
  - `TestBridgeEventSubscription`：订阅白名单门控（未订阅零转发）
  - `TestNodeBridgeAgentStatusEvent`：spawn → agent/status 事件载荷（os.Pipe 捕获）
  - `TestNodeBridgeDSHPluginE2E`：npm 安装 dsh-agent-teams@0.1.14 + DSH peers →
    cordis4 装载 → **13 个 agent_teams_* 工具注册** → create（staged）→ status →
    delete 冒烟 + `.agent-teams/<id>/team.json` 落盘校验；环境不满足自动 Skip
- `internal/agent/node_bridge_test.go`：`TestNodePluginRuntime`（dsh/node/goja 判定）、
  `TestNodePluginsFile`（新格式 + 旧格式兼容）、`TestNodePluginNeedsNode` 增 dsh 用例

### 工作包 6：文档/许可证

- `docs/THIRD_PARTY_NOTICES.md`（新增；t2 初稿在仓库根，t2 完成时移入 docs/ 保持 inScope）：
  t1 §5 许可证核查表落实（全 MIT）。
- 本文件（实施记录）+ `docs/round4-js-runtime-upgrade.md` §6 回填。
- `docs/plugin-gaps.md`：Round4 状态追加（见其「Round4」节）。

---

## 3. 变更文件清单

| 文件 | 变更 |
|---|---|
| `internal/agent/bridge_node.js` | cordis4 装载分支 + DSH 门面 + 协议扩展（subscribe/event/cmdrun/convId） |
| `internal/agent/node_bridge.go` | 订阅表/事件转发/dshService 服务面/runNodeCommand/invoke convId |
| `internal/agent/node_plugins.go` | @deepseek-ai/cordis 判定 + nodePluginRuntime + plugins.json 新格式 + DSH peer 安装 |
| `internal/agent/npm_plugin.go` | uninstall 运行时判断扩展（node/dsh） |
| `internal/agent/subagent_registry.go` | agent/status 事件源（running/idle/stopped） |
| `internal/agent/role_prompts.go` | DefaultJudgePrompt/judgeSystemPrompt 迁入（A2 处置） |
| `internal/agent/search.go` | extLangMap 迁入（findfiles.go 删除后 glob 生产依赖保留） |
| `internal/agent/dsh_bridge_test.go` | 新增：DSH 服务面/事件桥/E2E 测试 |
| `internal/agent/node_bridge_test.go` | 新增：runtime 判定/plugins.json 兼容用例 |
| `internal/agent/findfiles.go` `projectindex.go` `evalstore.go` `evaluator.go`（+2 测试文件） | **删除**（Go 核心审计 A1/A2 处置） |
| `docs/THIRD_PARTY_NOTICES.md` | 新增：许可证核查表（docs/ 下，保持 inScope） |
| `docs/runtime-upgrade-plan.md` | 本文件 |
| `docs/plugin-gaps.md` | Round4 状态节 |

**Round4 repair（t6）追加**：

| 文件 | 变更 |
|---|---|
| `internal/agent/jsplugin.go` | LoadCordisPatch runtime 判定 `"node" \|\| "dsh"`（F1b） |
| `internal/agent/node_bridge.go` | epoch 代数（Close 递增 / start 快照校验 / 成功清零 restarts）（F2）；handleServiceMsg 锁内 ph 快照 + dshService 签名（F3） |
| `internal/agent/plugin_test.go` | 新增 TestLoadCordisPatchDSHRuntime（F1b 用例） |
| `internal/agent/dsh_bridge_test.go` | dshService 新签名调用点同步（F3） |
| `internal/agent/live_e2e_test.go` | 5.3/5.4 改用 edit/read/glob 新工具名（F4） |
| `.pair/assets/runtime/bridge_node.js` | **F1a 同步**：新版内嵌脚本同步到外置资源（与内嵌 SHA 一致，跟踪文件） |
| `packager.json` | **F1a 同步链**：dist.include 补 `bridge_node.js` + `cordis.bundle.js`（发布包携带外置运行时脚本） |
| `internal/agent/runtime_assets.go` | **F1a**：`runtimeAssetDirOverride` 测试钩子（生产 nil） |
| `internal/agent/node_bridge_test.go` | **F1a**：新增 TestBridgeNodeSourceExternalPriority（外置优先/回退/同步回归三重断言） |
| `docs/runtime-upgrade-plan.md` | §4 验证结论 + §4.1 repair 记录（F1a-F5） |

## 4. 验收对照

- ✅ 运行时升级按 t1 方案（候选 C）实施，分工作包 1-6，每包有验收测试
- ✅ Node API 完整可用（真实 Node 进程 = require/fs/path/process/事件循环原生能力）
- ✅ 借用代码许可证：本轮**零借用**（候选 A 未实施），npm 依赖全 MIT 已登记
- ✅ dsh-agent-teams 直跑：E2E 测试 13 工具注册 + create/status/delete 冒烟
- ✅ 零回归：goja 轨与 cordis3 轨代码路径不变（协议向后兼容）
- ✅ Go 核心处置：A1 孤儿工具删除（extLangMap 保留）、A2 死代码删除（judge 提示迁移）
- ✅ 验证结论（t2 + t6 repair 复跑）：`go build ./...` exit 0；`go vet ./...` exit 0；
  `go test ./internal/agent/ -count=1 -timeout 10m` 仅 14 项文档登记环境依赖失败
  （msys bash 信号管道组 + rod 下载，与 round3 §10.1 清单一致）**零新增**；
  `TestNodeBridgeDSHPluginE2E` 实跑 PASS（dsh-agent-teams@0.1.14：13 个 agent_teams_* 工具注册、
  create（staged）→ status → delete 冒烟、`.agent-teams/<id>/team.json` 状态落盘）。

---

## 4.1 Round4 repair（t6 · 2026-09）实施记录

t4 审查 findings 处置（F1-F5，全部修复）+ t3 验证新增 F1a（高严重度，纳入本 repair）：

| # | 审查项 | 处置 | 证据 |
|---|---|---|---|
| F1a | **外置资源遮蔽（高）**：`bridgeNodeSource()`（node_bridge.go:47-48）外部资源优先——exe 在仓库根（开发/冒烟）时加载 `.pair/assets/runtime/bridge_node.js`（12,329B，commit 2e0f36ab 起未更新，无 Round4 改动），内嵌新版（29,834B）被遮蔽；真实装载 DSH 条目报 `Cannot find package '[object Object]'`（旧脚本把 {spec,runtime} 对象 String() 化） | **已修复**：① 新版 `internal/agent/bridge_node.js` 同步到 `.pair/assets/runtime/bridge_node.js`（跟踪文件，与内嵌 SHA 一致，commit ae73dce4）；② 同步链核查：`scripts/build-ui.mjs`/`sync-web-dist.mjs` 仅管 web 壳，`packager.json` dist.include 原仅含 `.pair/assets/runtime/web`——**已补 `bridge_node.js` + `cordis.bundle.js` 两条目**（发布包也携带外置运行时脚本，字段可独立热更）；③ 双守卫测试：`TestBridgeNodeResourceSync`（ae73dce4，外置==内嵌逐字节）+ `TestBridgeNodeSourceExternalPriority`（本 repair，外置优先/回退内嵌/归一化同步三重断言）；`runtime_assets.go` 增 `runtimeAssetDirOverride` 测试钩子 | `.pair/assets/runtime/bridge_node.js`（同步，ae73dce4）、`internal/agent/bridge_resource_sync_test.go`（ae73dce4）、`packager.json`（dist.include +2 条目）、`internal/agent/node_bridge_test.go`、`internal/agent/runtime_assets.go` |
| F1b | `LoadCordisPatch` runtime 判定只认 `"node"`，`runtime="dsh"` patch 条目被静默跳过 | **已修复**：判定改 `rt == "node" \|\| rt == "dsh"`；`plugin_test.go` 新增 `TestLoadCordisPatchDSHRuntime`（dsh+node 双条目 → 断言桥就绪；纯代码条目 → 断言不触发） | `internal/agent/jsplugin.go`（LoadCordisPatch）、`internal/agent/plugin_test.go` |
| F2 | 桥 Close/start 竞态：启动期间被 Close/替换可能遗留僵尸进程、过期启动覆盖 globalNodeBridge | **已修复**：`nodeBridge.epoch` 代数计数——`Close()` 递增；`start()` 启动前快照 epoch，就绪等待循环内 `isStale` 校验（过期 → kill 进程并返回「已取消」错误）；`start()` 成功后 `restarts` 清零（连续崩溃计数只统计就绪前段） | `internal/agent/node_bridge.go`（epoch/isStale/start/Close） |
| F3 | `handleServiceMsg` 内 `b.ph` 无锁读取（与 `bindHost` 写并发数据竞争），dshService/subAgentSpecFromArgs 各自取 ph 可能不一致 | **已修复**：`handleServiceMsg` 顶部 `b.mu` 锁内取 ph 快照，传入 `dshService(…, ph)` 与 Execute 回落路径共用同一快照；`dshService` 签名增 ph 参数（测试调用点同步更新） | `internal/agent/node_bridge.go`（handleServiceMsg/dshService/subAgentSpecFromArgs）、`internal/agent/dsh_bridge_test.go` |
| F4 | `live_e2e_test.go` 任务文本/断言仍引用已删除的 `find_files_by_pattern` | **已修复**：5.4 改 `glob`（pattern `**/*.go`，本意即验证 glob ** 语义）；5.3 同步 `edit_file/read_file` → `edit/read`（词边界断言防 multi_edit 误判） | `internal/agent/live_e2e_test.go` |
| F5 | 实施记录缺实际路径与验证结论 | **已修复**：本文件 §3 变更清单（实际路径）+ §4 验证结论（命令输出与 E2E 结果） | `docs/runtime-upgrade-plan.md` |

**repair 验证**：`go build ./...` ✅、`go vet ./...` ✅、`go test ./internal/agent/ -count=1 -timeout 10m` ✅
（新增用例 `TestLoadCordisPatchDSHRuntime`、`TestBridgeNodeSourceExternalPriority` 通过；
14 项环境依赖失败清单不变，零新增）。F1a 实测闭环：修复后真实装载（exe 在仓库根冒烟）
`bridgeNodeSource()` 返回新版脚本，DSH 条目按 `{spec,runtime:"dsh"}` 正确装载（E2E 复跑 PASS）。

## 5. 遗留（P2，记录不阻塞）

1. agent/pre-step 拦截链、agent/request-error 事件源（宿主循环语义改造，见工作包 4）。
2. webServer/workspaceRegistry 门面（Web 面板数据路由）：宿主 UI 槽位由既有
   agent-teams 移植版 client.js 承载；headless 下插件保持 tool-only（与 DSH
   headless profile 语义一致）。
3. ~~goja_nodejs 借用（候选 A）~~ —— **✅ 已实施（2026-08-29「创造需求」落地）**：
   无外部需求时从仓库自有插件创造需求（`tool-project-info` 等手工 `d+'/'+name`
   路径拼接在 Windows 分隔符下有隐患、无 base64/hex 编解码）。落地为
   `internal/agent/nodeapi_mini.go` 原创实现（require + fs（工作区根受限同步面）/
   path/buffer/events/util + 相对文件模块），**未引入 goja_nodejs 依赖**（避免 fork
   模块身份问题）；`tool-project-info` 已重构使用 require('path')。测试：
   TestNodeAPIMini{FS,PathAndBuffer,RelativeRequire,InSandbox}。剩余：全量 Node fs
   （watcher/stream 面）与 npm 模块解析仍走 Node 桥（设计使然）。

---

## 6. t5 端到端集成记录（2026-09 · engineer）

### 6.1 集成冒烟（真实 pair.exe，临时端口 9423）

前置：工作区桥（`.pair/cordis/node/`，gitignored 运行时数据）装配
`@nanmicoder/dsh-agent-teams@0.1.14` + DSH peers（cordis 4.0.1 / dsh-* rc.8 /
schemastery）；`plugins.json` 写入 `{"spec":"@nanmicoder/dsh-agent-teams@0.1.14","runtime":"dsh"}`
对象条目；`.pair/cordis.patch.json` 写入同款 patch 条目（runtime=dsh）——
使 `LoadCordisPatch` 在**应用启动时**自动拉起 Node 桥（t6 F1b 修复的真实闭环）。

冒烟结果（`CGO_ENABLED=1 go build -o pair.exe ./cmd/companion`，67.3MB）：

| 项 | 结果 |
|---|---|
| `/api/health` | ✅ 200（workspace 正常） |
| 全局插件宿主 | ✅ 46 个插件（= round3 基线 + 既有） |
| 工具集 | ✅ 47 个（17 个插件） |
| 桥自动启动 | ✅ `[node-bridge] 就绪（插件 [@nanmicoder/dsh-agent-teams@0.1.14]，工具 ）` |
| DSH 插件装载 | ✅ `已装载插件 @nanmicoder/dsh-agent-teams@0.1.14（runtime=dsh）`（cordis4 分支） |
| 事件订阅 | ✅ `[agent/status]` → `[agent/status agent/pre-step internal/service]` |
| DSH 服务面 | ✅ llm.current / agents.list / systemPrompt.section（diag 日志） |
| 工具注册 | ⚠️ 13 个 `agent_teams_*` 与 repo 移植版（`.pair/plugins/agent-teams`）同名被
  claimTool 拒绝——**替代关系定案**（t3 F2 记录）：报错含完整处理建议
  `工具 "agent_teams_create" 已被插件 agent-teams 注册，插件 node-bridge:@nanmicoder/dsh-agent-teams 不能覆盖。
  请换工具名，或先 cordis_stop agent-teams 再注册`（t5 已补归属名，非静默、非 FATAL） |
| 日志扫描 | ✅ FATAL/panic = 0 |

### 6.2 t5 顺手修复（集成暴露）

| 项 | 说明 |
|---|---|
| **resolvePluginApply**（bridge_node.js） | 导出形态健壮化：类/服务库（如 `@deepseek-ai/dsh-fs-local`
  导出 `LocalFileSystem` 类）此前经原型链取到 `Function.prototype.apply` 以
  `TypeError` 崩溃；现仅接受非 class 函数 / own-property `apply`/`default`，
  其余友好跳过（`无 apply 函数（导出形态不支持：类/服务库/空导出），跳过`） |
| **桥工具归属名**（node_bridge.go） | `handleToolMsg`/`registerToolsTo` 经
  `forPlugin("node-bridge:"+plugin)` 注册——claimTool 冲突报错信息完整
  （verifier F2 cosmetic 项落实）；无冲突时正确登记归属 |
| **plugins.json 清理** | 工作区桥原含 2 条误装服务库条目
  （`@deepseek-ai/dsh-fs-local`/`dsh-subprocess-local`——非 cordis 插件，
  cordis3 轨下「装载成功」为 ctx.plugin 延迟激活假象、工具恒为空）已移除并记录 |
| **外置资源再同步** | bridge_node.js 修复后重新同步
  `.pair/assets/runtime/bridge_node.js`（跟踪文件）；`TestBridgeNodeResourceSync`
  守卫复跑 PASS（内嵌==外置逐字节） |

### 6.3 验证复跑（t5 侧）

`go build ./...` ✅、`go vet ./...` ✅、关键测试全 PASS：
`TestNodeBridgeDSHPluginE2E`（36.6s）、`TestBridgeNodeResourceSync`、
`TestLoadCordisPatchDSHRuntime`、`TestDSHBridge*`、`TestBridgeEvent*`、
`TestNodeBridgeAgentStatusEvent`、`TestNodePluginRuntime`、`TestNodePluginsFile`、
`TestToolLandingMatrix`（295 断言）。

前端构建：`cd cmd/companion/web-ui && npm run build` 在 DSH 沙箱内被
`spawn EPERM` 阻断（vite/esbuild 子进程 spawn 边界，round3 §10.1 同款记录）；
**本轮前端源码零改动**，dist 为既有构建结果；沙箱外同一命令可执行
（已上报队长走全权限通道）。

### 6.4 提交

集成提交涵盖：t2 运行时升级 + t6 repair + t5 集成修复与文档（见 §3/§4.1/§6），
`go.sum` 无变化（本轮零新增 Go 依赖），提交后 `git status` 干净。
