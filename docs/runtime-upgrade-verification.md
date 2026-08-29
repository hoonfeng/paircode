# Round4 运行时升级独立验证报告（t3）

> 验证人：verifier（独立 QA，未参与 t2 实现）。
> 验证对象：t2「实施 Round4：运行时完整化 + DSH 插件直跑 + Go 核心处置」（commit 8f17896b 工作树 + t2 改动）。
> 结论：**FAILED（阻塞）** —— 发现 1 项**高严重度未覆盖缺陷（F1a：运行时资源外置桥脚本未同步）**，
> 导致真实应用（exe 置于仓库根、加载 `.pair/assets/runtime` 外置资源）下 DSH 插件装载失败；
> 另独立复现 t4 审查已报的 F1（跨重启装载丢失，t6 修复中）。
> 环境：Windows、Go 1.26.4、Node v22.17.0、npm 11.18.0、registry.npmjs.org 可达。
> 沙箱说明：`go build/vet/test` 需工作区 GOCACHE（`C:\Users\...\go-build` 写入被沙箱拒）；
> 模块缓存 `F:\MyGolangPrograms\pkg\mod` 只读（stat cache 写失败为非致命回退，不影响结果）。

---

## 1. 命令结果表（t2 verify 逐条复跑）

| # | 命令 | 结果 | 退出码 | 证据 |
|---|---|---|---|---|
| 1 | `go build ./...`（CGO_ENABLED=1 + 工作区 GOCACHE） | ✅ 通过 | 0 | 全包编译成功（`.verify-tmp/gocache`；仅 telemetry/stat-cache 非致命告警） |
| 2 | `go vet ./...` | ✅ 通过 | 0 | 全包 vet 无告警 |
| 3 | `go test ./internal/agent/ -count=1 -timeout 10m` | ✅ 通过（除 14 项文档登记环境依赖失败） | 1* | 132.1s；**恰好 14 项失败 = 文档登记清单**（13 msys bash 信号管道 `couldn't create signal pipe, Win32 error 5` + 1 rod `TestWebDebugTimeoutMs` AppData 写拒），**零新增**；`TestToolLandingMatrix`（295 断言）PASS |
| 4 | `node --check internal/agent/bridge_node.js` | ✅ 通过 | 0 | 升级后桥脚本语法合法 |
| 5 | `node --check .pair/cordis/node/bridge.js`（部署态） | ✅ 通过 | 0 | 语法合法（内容为旧版，见 F1a） |
| 6 | `go test -v -run 'TestNodeBridgeDSHPluginE2E\|TestDSH*\|TestNodeBridge*\|TestNodePlugin*\|TestToolLandingMatrix\|TestNoOldNameRegistration'` | ✅ 全部 PASS（1 SKIP） | 0 | 见 §4 清单；`TestNodeBridgeE2EHelloBridge` SKIP（需本地 npm 夹具，非本轮回归） |

*退出码 1 仅由 14 项环境依赖失败引起，与 round2 §6.13 / plugin-gaps.md「环境依赖测试说明」清单逐项吻合。

14 项环境依赖失败清单（与 docs/plugin-gaps.md L133-148 文档登记一致）：
`TestToolHarnessJSNative / TestHarnessAlias_GlobGrepBash / TestRunCode_Go / TestBGCrossRegistry /
TestJSPluginCtxBashLogger / TestRunCodeNested / TestDetectBash / TestRunShellWithBash /
TestRunShellBacktick / TestRunBackground / TestRunCommandInLoop / TestToolCoreJSNative /
TestToolRunCommand`（全部 msys bash 信号管道，沙箱禁 named-pipe 所致，DSH 文档化边界）
+ `TestWebDebugTimeoutMs`（rod Chromium 下载写 AppData 被拒）。

---

## 2. 范围核对

| 核对项 | 结果 | 证据 |
|---|---|---|
| git status/diff 全在 inScope | ✅ | 改动 15 个文件（internal/agent 14 + docs/plugin-gaps.md 1）+ 未跟踪 5 个（docs/THIRD_PARTY_NOTICES.md、docs/round4-go-core-audit.md、docs/round4-js-runtime-upgrade.md、docs/runtime-upgrade-plan.md、internal/agent/dsh_bridge_test.go）；outOfScope 过滤 = **零条目** |
| outOfScope 零改动 | ✅ | `.agent-teams/`、`release/`、`E:\`、config 敏感文件、cmd/companion、pkg、internal/server/handler、plugins-src、scripts 零改动 |
| go.mod / go.sum | ✅ **零变化** | `git diff go.mod go.sum` 空（候选 A 未实施 → 无新 Go 依赖；npm 依赖经桥目录 install，不进 go.mod） |
| 删除文件引用完整性（A1/A2） | ✅ | `find_files_by_pattern`/`project_index` 生产面零残留（仅 live_e2e 夹具字符串/注释，t6 修复中）；`DefaultJudgePrompt/judgeSystemPrompt` 已迁 role_prompts.go 且 `TestRolePromptDiskDefaultSync` 通过；`extLangMap` 迁 search.go 并被 glob 引用（search.go:252） |
| 许可证合规 | ✅ | docs/THIRD_PARTY_NOTICES.md 齐全（goja/goja_nodejs(未引入)/@cordisjs/core/@deepseek-ai/cordis/dsh-*/schemastery/@nanmicoder/dsh-agent-teams 全 MIT + 维护约定）；go.mod 零新增 |

> 注：t6（repair，t4 审查触发）在验证期间并行修改了 `internal/agent/jsplugin.go`（F1 修复）、
> `internal/agent/plugin_test.go`（新增 TestLoadCordisPatchDSHRuntime）、`internal/agent/live_e2e_test.go`
> （F4 夹具修正）——与本报告 F1 结论一致，非本验证改动。

---

## 3. 装载冒烟（临时端口 9547，避开 9090；exe 置于仓库根 = InstallDir 语义）

### Phase 1 —— 既有状态基线（patch 空、plugins.json 两条旧 cordis3 条目、未装 DSH 插件）

| 探活项 | 结果 |
|---|---|
| `/api/health` | ✅ 200 `{"status":"ok","workspace":"F:\\syproject\\gou-ide","folders":[3]}` |
| `/api/plugins` | ✅ 46 插件全装载（与 round3 基线一致） |
| `/api/tools` | ✅ 184 工具（与 round3 基线一致） |
| `/api/commands` | ✅ `agent-teams`（owner=agent-teams，repo 移植版） |
| `POST /api/commands/run`（/agent-teams） | ✅ 返回实时团队快照（runtime-complete-round4 / 4 成员 / 5 任务） |
| 日志扫描 | ✅ FATAL/panic/duplicate/冲突 = 0；Node 桥未启动（patch 空，符合既有行为） |

### Phase 2-A —— 模拟 DSH 安装后状态（plugins.json 增 `{spec:"@nanmicoder/dsh-agent-teams@0.1.14",runtime:"dsh"}`；patch 增 runtime="dsh" 条目；npm 实装插件+peers 成功）

| 探活项 | 结果 |
|---|---|
| Node 桥启动 | ❌ **未启动**（bridge_err.log 零新行、应用日志零 `[node-bridge]` 行） |
| DSH 插件装载 | ❌ 未装载（/api/commands 仍 owner=agent-teams，无第二条命令） |
| 工具面 | ⚠️ 13 个 `agent_teams_*` 全部来自 repo 移植版（同名工具无法区分，需命令/日志判别） |

→ **复现 t4-F1（跨重启装载丢失）**：patch runtime="dsh" 不触发启动装配 `LoadCordisPatch` 的
`needNodeBridge`（jsplugin.go:4003 只认 "node"）。t6 修复已在途（工作树可见 jsplugin.go 改为
`rt == "node" || rt == "dsh"` + 新测试）。

### Phase 2-B —— 真实应用 + 现工作树资源（patch runtime="node" 触发桥启动）

| 探活项 | 结果 |
|---|---|
| 桥启动 | ✅ 启动（`[node-bridge] 就绪`） |
| 旧 cordis3 条目 | ✅ `已装载插件 @deepseek-ai/dsh-fs-local / dsh-subprocess-local`（cordis3 轨兼容） |
| **DSH 条目** | ❌ **`插件 [object Object] 装载失败: ERR_MODULE_NOT_FOUND: Cannot find package '[object Object]'`** —— 部署桥脚本把对象条目 `String(spec)` 成 `[object Object]` |

→ **发现 F1a（高，阻塞，t2 未覆盖）**：桥启动时 `bridgeNodeSource()`（node_bridge.go:47-48）**优先读
外置资源 `<exe 目录>/.pair/assets/runtime/bridge_node.js`**（runtime_assets.go:38-44 外部优先、embed
兜底），而该外置资源仍为 **12,329B 旧版**（commit 2e0f36ab「运行时资源外置」时代产物，无
readPluginSpecs/无 cordis4-DSH 分支；无任何同步脚本——仅 scripts/sync-web-dist.mjs 管 web dist），
内嵌新版（bridge_node.js 29,834B）被遮蔽。**E2E 测试直接写 `bridgeNodeJS` 内嵌变量，绕过外置资源，
因此测试全绿而真实应用装载失败。**

### Phase 2-C —— 临时部署新版桥脚本为外置资源（验证后已还原） + patch runtime="dsh"

| 探活项 | 结果 |
|---|---|
| 桥启动 | ❌ 仍未启动 → 独立确认 F1b（= t4 F1）：即使脚本为新版，runtime="dsh" 仍不触发启动装配 |

### Phase 2-D —— 新版桥脚本 + patch runtime="node"（隔离验证装载/执行路径本身）

| 探活项 | 结果 |
|---|---|
| DSH 插件装载 | ✅ `已装载插件 @nanmicoder/dsh-agent-teams@0.1.14（runtime=dsh）`（cordis4 分支） |
| DSH 服务面实测 | ✅ 装载期真实调用：`llm.current`、`agents.list`、`systemPrompt.section`、`commands.register`（`[node-bridge:diag]` 日志） |
| 事件订阅 | ✅ `订阅事件 [agent/status agent/pre-step internal/service]`（subscribe 协议 + 白名单） |
| 命令注册 | ✅ `/agent-teams` owner 变为 `node-bridge:@nanmicoder/dsh-agent-teams`（同名覆盖语义，commands.go:37-38 文档化） |
| `POST /api/commands/run`（/agent-teams） | ✅ **端到端 cmdrun 往返成功**：HTTP → Go 宿主命令表 → 桥 cmdrun → DSH 插件 handler → 返回 `{"kind":"error","text":"Usage: /agent-teams [--profile <name>] <goal>"}`（DSH 侧用法提示，证明 DSH 命令执行链路全通） |
| 工具注册 | ⚠️ **13 个 `agent_teams_*` 全部注册失败**：`工具 "agent_teams_*" 已被插件 agent-teams 注册，插件  不能覆盖`（claimTool 冲突；repo 移植版与 DSH 插件工具同名，二者为替代关系非并存关系；错误信息明确含处理建议，非静默） |
| 日志扫描 | ✅ 无 FATAL/panic |

> 验证后已还原：外置资源（12329B 原样）、plugins.json、cordis.patch.json 均恢复备份；
> git status 与验证前一致（仅 t2+t6 文件）；9547 无残留监听、无残留进程。
> 桥目录 node_modules 遗留 @nanmicoder/dsh-agent-teams + dsh-* rc.8（git-ignored 运行时状态，
> 与真实安装路径一致；plugins.json 已还原故不会被装载）。

---

## 4. DSH 插件实跑验证（t1 验收标准对照）

| t1 验收标准 | 结果 | 证据 |
|---|---|---|
| `agent_teams_*` 13 工具注册进 Registry | ✅（隔离桥路径）/ ❌（真实应用并存 repo 移植版时，见 §3 Phase 2-D 冲突） | `TestNodeBridgeDSHPluginE2E` PASS：真实 npm 安装 dsh-agent-teams@0.1.14 + DSH peers（cordis 4.0.1 / dsh-* rc.8 / schemastery）→ cordis4 装载 → **13 个 agent_teams_* 工具注册**（参考插件源码逐名核对 13 个：create/add_member/remove_member/reassign_task/create_task/resume/delete/claim_task/update_task/send_message/status/approve/edit_plan） |
| 冒烟调用 create/status/delete 两阶段 | ✅ | E2E：create（approval=required → staged）→ `.agent-teams/<id>/team.json` 落盘断言 → status（team_id/phase 一致）→ delete（deleted:true） |
| `/agent-teams` 命令注册 | ✅（隔离）/ ✅（真实应用 cmdrun 实跑） | 单元 `TestDSHBridgeSystemPromptCommand`（commands.register 进宿主表 + unregister 清理）PASS；真实应用 Phase 2-D `/api/commands/run` 实跑 DSH handler 成功 |
| 事件桥（agent/status 等订阅面） | ✅ | `TestBridgeEventSubscription`（白名单门控：未订阅零转发）PASS；`TestNodeBridgeAgentStatusEvent`（spawn → running 事件，载荷含 agent.id/session.header.cwd 对齐 DSH scheduler）PASS；Phase 2-D 实装载订阅日志 |
| DSH 服务面（llm/agents/systemPrompt/commands） | ✅ | `TestDSHBridgeServices`（映射/错误路径/回落 mapBridgeService）PASS；Phase 2-D 实装载 diag 日志四服务真实调用 |

**其余 Round4 验证测试**：`TestDSHBridgeSystemPromptCommand` PASS、`TestNodePluginRuntime` PASS、
`TestNodePluginsFile`（新旧 plugins.json 格式兼容）PASS、`TestNodePluginNeedsNode`（+dsh 用例）PASS、
`TestNodeBridgeProtocolPing` PASS、`TestNoOldNameRegistration` PASS、`TestToolLandingMatrix`（295 断言）PASS。

---

## 5. 既有插件兼容性回归抽查

| 抽查项 | 结果 | 证据 |
|---|---|---|
| 46 插件装载（tool-*/ui-*/agentloop/agent-teams/core-api/fs-api/git-api/marketplace/web-api/host-capability-probe） | ✅ 与升级前一致 | Phase 1 `/api/plugins` = 46；日志 `全局插件宿主已初始化（46 个插件）`、`已装载 47 个工具集（17 个插件）`（= round3） |
| 184 工具注册 | ✅ 与升级前一致 | Phase 1 `/api/tools` = 184（= round3）；TestToolLandingMatrix 295 断言零未落地 |
| agent-teams（repo 版）关键路径 | ✅ 升级前行为不变 | Phase 1 `/api/commands` + `/api/commands/run` 返回实时团队快照；卸载 DSH 状态还原后命令 owner 恢复 agent-teams |
| agentloop | ✅ | 装载日志 `registerLoop: 已注册 JS 循环实现 "agentloop"`（= round3 基线） |
| cordis3 Node 桥插件（旧 npm 轨） | ✅ 无回归 | Phase 2-B 两条旧 plugins.json 字符串条目经 cordis3 分支照常装载 |
| goja 轨 | ✅ 零改动 | t2 未触碰 goja 轨（jsplugin.go 仅 t6 修复 F1 时改动）；46 插件全走 goja 装载成功 |
| 工具名冲突语义（DSH vs repo 移植版并存） | ⚠️ 记录不阻塞 | 同名 13 工具被 claimTool 拒绝（明确报错+处理建议）；`/agent-teams` 命令按文档化语义被 DSH 覆盖——**两个插件为替代关系**；t1 范围声明（repo 移植版保持 goja）下不构成对既有插件自身的回归 |

---

## 6. 发现

| ID | 级别 | 问题 | 建议处置 |
|---|---|---|---|
| **F1a** | **高（阻塞）** | 运行时资源外置优先（`LoadRuntimeAssetString` 外部优先 → embed 兜底），而 `.pair/assets/runtime/bridge_node.js`（12,329B，commit 2e0f36ab 起未更新）**无 Round4 改动**；真实应用桥脚本为旧版：plugins.json 对象条目解析为 `[object Object]`（`ERR_MODULE_NOT_FOUND`），cordis4/DSH 分支、subscribe/event/cmdrun 协议均不存在 → **DSH 插件在真实应用装载必然失败**；E2E 测试直写内嵌 `bridgeNodeJS` 绕过资源层，未覆盖此路径。t2 changedPaths 不含 .pair/assets（inScope 未列） | **阻塞修复**：将新版 bridge_node.js 同步到 `.pair/assets/runtime/bridge_node.js`（并核查 packager/release 资源同步链）；补一条「外置资源优先」场景的测试或文档化资源同步步骤（t6 范围外，需队长纳入 t6 或新修复任务） |
| F1b | 高（阻塞，与 t4 F1 同） | patch runtime="dsh" 不触发 `LoadCordisPatch` 桥启动（jsplugin.go:4003 只认 "node"）→ DSH 插件跨重启丢失 | t6 修复已在途（工作树 jsplugin.go `rt=="node" \|\| rt=="dsh"` + TestLoadCordisPatchDSHRuntime）；修复后需重验重启场景 |
| F2 | 中（信息） | DSH 插件与 repo agent-teams 移植版工具同名 13 项 → 并存时 DSH 工具全被拒（claimTool 明确报错，非静默）；`/agent-teams` 命令被覆盖（文档化语义） | 记录：两者为替代关系（安装 DSH 插件前先停 repo 移植版）；错误信息中插件名空白（`插件  不能覆盖`，bridge 工具归属名未填）可顺手补 |
| F3 | 低（信息） | `runtime-upgrade-plan.md` §3/§6 称 THIRD_PARTY_NOTICES.md 在根目录，实际在 docs/（与 t4 F5 同） | t6 顺手修正 |

---

## 7. 结论

- **通过项**：构建/静态检查/全量测试（14 项环境依赖失败与文档登记逐项吻合，零新增）；范围与依赖合规
  （outOfScope 零改动、go.mod/go.sum 零变化）；既有插件基线（46 插件/184 工具/命令）与升级前一致；
  隔离桥路径下 DSH 插件实跑（13 工具 + create/status/delete + 落盘 + 事件桥 + 服务面）全通。
- **阻塞项**：**F1a**（外置桥资源未同步 → 真实应用 DSH 装载失败，t2/t4/E2E 均未覆盖）与 **F1b**
  （跨重启装载丢失，t4 F1 同源，t6 修复中）未解决前，验收标准「装载冒烟：DSH 插件装载成功」与
  「DSH 插件实跑」在**真实应用路径**不成立。
- 验证环境证据留档：`.verify-tmp/agent-tests.log`（全量）、`.verify-tmp/round4-tests-v.log`（专项）、
  `.verify-tmp/plugins.json.bak / cordis.patch.json.bak / bridge_node.js.asset.bak`（状态还原备份）。
- 建议：t6 纳入 F1a 同步修复后，重跑本报告 §3 Phase 2-A/D 冒烟复核（无需重跑全量测试）。

---

## 8. 复验（attempt 2，F1a/F1b 修复后）——结论：**通过**

> 修复依据：F1a 由 commit `ae73dce4`（外置桥资源同步 + `TestBridgeNodeResourceSync` 防漂移守卫，
> runtime_assets.go/packager.json 配套）；F1b 由 t6 repair（jsplugin.go `rt=="node" || rt=="dsh"`
> + `TestLoadCordisPatchDSHRuntime`）。t7（review r2）PASS。

### 8.1 构建/测试复跑（当前工作树 = t2 + t6 + F1a）

| # | 命令 | 结果 | 退出码 | 证据 |
|---|---|---|---|---|
| 1 | `go build ./...`（CGO_ENABLED=1 + 工作区 GOCACHE） | ✅ | 0 | 含 F1a/t6 改动的全树编译通过 |
| 2 | `go vet ./...` | ✅ | 0 | 无告警 |
| 3 | `go test ./internal/agent/ -count=1 -timeout 10m` | ✅（14 项环境依赖除外） | 1* | 110.6s；失败清单与 round1/文档登记**逐项一致**（13 msys bash 信号管道 + 1 rod），零新增 |
| 4 | 专项 -v 批（守卫 + E2E + Round4 单元 + 回归） | 见 8.3 | 0 | `TestBridgeNodeResourceSync` / `TestLoadCordisPatchDSHRuntime` / `TestNodeBridgeDSHPluginE2E` 等 |

### 8.2 装载冒烟（临时端口 9548，模拟真实安装后状态：plugins.json DSH 条目 runtime=dsh + patch runtime=dsh）

| 探活项 | 结果 |
|---|---|
| `/api/health` | ✅ 200 |
| **桥自动启动（F1b 复验）** | ✅ patch runtime="dsh" 即触发 `LoadCordisPatch` → Node 桥启动（旧版行为：只认 "node"，不启动） |
| **DSH 插件装载（F1a 复验）** | ✅ `已装载插件 @nanmicoder/dsh-agent-teams@0.1.14（runtime=dsh）`（cordis4 分支）；**零 `[object Object]` / ERR_MODULE_NOT_FOUND** |
| DSH 服务面 | ✅ llm.current / agents.list / systemPrompt.section / commands.register 真实调用（diag 日志） |
| 事件订阅 | ✅ [agent/status] → [agent/status agent/pre-step internal/service] |
| 命令注册 | ✅ `/agent-teams` owner = `node-bridge:@nanmicoder/dsh-agent-teams` |
| **cmdrun 往返（复验）** | ✅ `POST /api/commands/run` → Go 宿主命令表 → 桥 cmdrun → DSH handler → `{"kind":"error","text":"Usage: /agent-teams [--profile <name>] <goal>"}` |
| 既有插件 | ✅ 46 插件 / 184 工具（=round3 基线）；2 条旧 cordis3 条目照常装载；13 个 DSH 工具与 repo 移植版同名被 claimTool 明确拒绝（F2 既有记录，替代关系，非静默、非 FATAL） |
| 日志扫描 | ✅ **FATAL/panic/`[object Object]`/ERR_MODULE_NOT_FOUND = 0 处** |
| 状态还原 | ✅ plugins.json/patch 恢复备份；9548 无残留监听/进程；git status 与复验前一致 |

### 8.3 专项测试 -v 证据（round4-tests-v-r2.log）

`TestBridgeNodeResourceSync`（F1a 守卫：内嵌含 DSH 标记 + 外置与内嵌字节一致）✅ PASS、
`TestLoadCordisPatchDSHRuntime`（F1b 守卫：dsh/node 条目触发桥、纯代码条目不触发）✅ PASS、
`TestNodeBridgeDSHPluginE2E`（13 工具 + create/status/delete + 落盘）✅ PASS、
`TestDSHBridgeServices` / `TestDSHBridgeSystemPromptCommand` / `TestBridgeEventSubscription` /
`TestNodeBridgeAgentStatusEvent` / `TestNodePluginRuntime` / `TestNodePluginsFile` /
`TestNodePluginNeedsNode` / `TestToolLandingMatrix`（295 断言）/ `TestNoOldNameRegistration` ✅ 全 PASS。

### 8.4 复验结论

round1 的阻塞项 F1a（外置桥资源未同步）与 F1b（runtime=dsh 不触发桥启动）**均已修复并有守卫测试锁定**；
真实应用（exe 仓库根）在真实安装状态（patch runtime=dsh）下：桥自动启动 → DSH 插件 cordis4 分支装载成功 →
服务面/事件订阅/命令注册全部生效 → cmdrun 往返执行 DSH handler 成功，全程零 FATAL。
验收标准逐条满足（见 t3 attempt 2 的 acceptanceResults）。遗留记录（非阻塞）：F2 工具同名替代关系（含
错误信息中插件名空白 cosmetic）、F3 文档路径小瑕、agent/pre-step 拦截链 P2、候选 A（goja_nodejs）P2。

---

## 9. t5 端到端集成结论（2026-09 · engineer）

t5 集成（全量构建 + 真实二进制冒烟 + 文档收尾 + 提交）完成，t3 复验场景在**集成态**下复现通过：

| 验收项 | 结果 |
|---|---|
| 前端构建 | ⚠️ DSH 沙箱 `spawn EPERM` 阻断（vite/esbuild 子进程边界，round3 §10.1 同款）；**本轮前端零改动**，dist 为既有产物；沙箱外同一命令可执行（上报队长全权限通道） |
| `CGO_ENABLED=1 go build -o pair.exe ./cmd/companion` | ✅ exit 0（67.3MB） |
| 启动冒烟（临时端口 9423） | ✅ health 200；46 插件 / 47 工具集；桥自启（patch runtime=dsh → LoadCordisPatch → ensureNodeBridge）；dsh-agent-teams cordis4 分支装载；事件订阅/服务面生效；FATAL/panic = 0 |
| 工具注册 | ⚠️ 13 个 `agent_teams_*` 与 repo 移植版同名被 claimTool 拒绝（F2 替代关系定案）——t5 已补**归属名**（`node-bridge:@nanmicoder/dsh-agent-teams`），报错含完整处理建议，非静默、非 FATAL |
| docs 收尾 | ✅ runtime-upgrade-plan.md §6（t5 集成记录）、本文件 §9、plugin-gaps.md Round4 状态（含 F2 替代关系同步） |
| git | ✅ 集成提交后工作树干净；`go.sum` 零变化（零新增 Go 依赖，无需变更） |

**t5 顺手修复（集成暴露，均已回归）**：
1. `resolvePluginApply`（bridge_node.js）：类/服务库导出形态（如 dsh-fs-local 的
   `LocalFileSystem` 类）不再以 `Function.prototype.apply` TypeError 崩溃，友好跳过；
2. 桥工具归属名（node_bridge.go `forPlugin("node-bridge:"+plugin)`）——F2 cosmetic 落实；
3. 工作区桥 plugins.json 清理 2 条误装服务库条目（非插件形态，cordis3 轨假装载）；
4. 外置桥资源随修复再同步（`.pair/assets/runtime/bridge_node.js`，守卫测试 PASS）。

**结论**：Round4 核心目标「DSH 参考插件在升级后运行时直接装载运行」在真实应用
（pair.exe 仓库根启动）集成态下成立；与 repo 移植版并存时为文档化替代关系
（先停移植版再装 DSH 插件即可全量接管 13 工具）。可交付。
