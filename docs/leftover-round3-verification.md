# Round3 遗留专项：独立验证报告（t3 · verifier）

> 独立验证 t2 实施结果（docs/plugin-round3-plan.md §7-§10，12 工作包 + 14 commit Round3-1…Round3-14）。
> 验证人：verifier。验证范围：构建/测试、范围合规、装载冒烟（临时端口）、新机制功能抽查、
> 旧名清理完整性。**本报告不修改任何实现代码**（唯一改动文件为本报告）。
> 基线：HEAD @ 7c02c637（Round3-14）；go1.26.4 windows/amd64；node v22.17.0；工作目录 F:\syproject\gou-ide。

---

## 1. 命令结果表（逐条运行 t2 verify 命令）

| # | 命令 | 退出码 | 结果 | 说明 |
|---|---|---|---|---|
| 1 | `go build ./...`（GOCACHE/GOMODCACHE 重定向 `.verify-tmp/`，CGO_ENABLED=1） | 0 | ✅ 通过 | stderr 仅 go telemetry upload token 沙箱噪音（Access denied，不影响构建） |
| 2 | `go vet ./...`（同上） | 0 | ✅ 通过 | 同上 |
| 3 | `go test ./internal/agent/ -count=1 -timeout 10m`（复跑 2 次） | 1 | ⚠️ 环境依赖失败（非回归） | 14 个失败全部为沙箱环境限制，与 round2 §6.13 清单同根因（见 §1.1） |
| 4 | `go test ./internal/core/ ./internal/server/... -count=1 -timeout 10m` | 0 | ✅ 通过 | core + server/handler 全绿 |
| 5 | `go test ./internal/hook/ ./pkg/summary/ -count=1` | 0 | ✅ 通过 | hook 全绿；summary 无测试文件 |
| 6 | Round3 新增测试集（-run 过滤，见 §1.2） | 0 | ✅ 通过 | 21 项全绿 1.2s |
| 7 | `node --check` 全部插件 JS（45 插件目录全量 *.js） | 0 | ✅ 通过 | 零语法错误 |
| 8 | 装载冒烟（WEB_PORT=9547 临时端口，避开 9090/9479/9323） | 0 | ✅ 通过 | 见 §3 |
| 9 | 前端构建 `cd cmd/companion/web-ui && npm run build` | 1 | ⚠️ 环境依赖（非回归） | vite/esbuild `spawn EPERM`（DSH 沙箱文档化边界；t2 §10.1 同注），需队长全权限通道复跑 |

### 1.1 全量测试环境依赖失败清单（14 项，全部非回归）

| 失败测试 | 根因 | 判定 |
|---|---|---|
| TestToolRunCommand / TestRunShellWithBash / TestRunShellBacktick / TestRunBackground / TestRunCommandInLoop / TestDetectBash / TestJSPluginCtxBashLogger / TestToolCoreJSNative / TestToolHarnessJSNative / TestHarnessAlias_GlobGrepBash / TestRunCode_Go / TestBGCrossRegistry / TestRunCodeNested | msys bash `couldn't create signal pipe, Win32 error 5`（受限 token 无法建命名管道，DSH 沙箱文档化边界；`exit status 0xc0000142`） | 环境依赖 |
| TestWebDebugTimeoutMs | rod/Chromium 下载写 `%APPDATA%\rod` 被拒（`mkdir ... : Access is denied`） | 环境依赖 |

> 与 round2 §6.13 清单对比：round2 的 13 项全部复现；新增 TestBGCrossRegistry、TestRunCode_Go
> 2 项（失败签名同为 msys 信号管道，非断言语义错误）；round2 的 TestNodeBridgeE2EHelloBridge
> 本轮未失败（local-test-plugin 环境就绪）。**无新增断言类失败**。

### 1.2 Round3 新增/关键测试（独立复跑全绿）

`go test ./internal/agent/ -run 'TestNoOldNameRegistration|TestRegisterHarnessTools_Base|TestGoalManager|TestGoalAutoContinue|TestGoalArchiveExecutors|TestAgentsForkSpec|TestJSPluginAgentsFork|TestWorkflowRunner|TestCommandsRegistry|TestJSPluginCommandsBridge|TestAskUserMultiQuestion|TestAskUserNoHostTimeout|TestToolLanding' -count=1 -v` → **ok（1.2s）**

21 项全 PASS：TestNoOldNameRegistration / TestRegisterHarnessTools_Base / TestGoalManager /
TestGoalAutoContinue / TestGoalArchiveExecutors / TestAgentsForkSpec / TestJSPluginAgentsFork /
TestWorkflowRunner_Agent / _Pipeline / _Parallel / _PhaseLogArgs / _Cancel / _Errors /
TestCommandsRegistry / TestJSPluginCommandsBridge / TestAskUserMultiQuestion_Parse / _Routing /
_Executor / TestAskUserNoHostTimeout（单问题回归）/ TestToolLandingMatrix / TestToolLandingSpotCheck。
另 `go test ./internal/core/ -run TestTemplateSchemaAligned` → **ok**（settings/models/mcp 三模板键集⊆结构体 + 默认值一致）。

---

## 2. 范围核对（git status/diff 全量）

- **工作树完全干净**：`git status --porcelain` 空（含 untracked）；14 个 commit 全部落库。
- 改动范围（Round3-1…Round3-14，`git log e1e6ca2a..HEAD --stat` 全量核对）：
  - `internal/{jobs,permission,provider,vterm}` 4 包 14 文件删除 + `scripts/relocate_imports.go` 4 条映射 → 工作包 1 ✅
  - `internal/agent/`（tools.go/search.go/harness_tools.go/builtin_plugins.go/tools_concise.go/tools_staging.go
    改名基座；19 个测试文件迁移；goal.go/workflow.go/commands.go/subagent_registry.go/jsplugin_agents.go/
    session_bridge.go/session_manager.go/message_store.go/plugin.go 新机制与清理；no_old_name_test.go 等新测试）→ 工作包 2-12 ✅
  - `cmd/companion/`（web_server.go ask_user answers 双兼容 + /api/commands 两接口；subagent_spawn.go ForkSeed；kernel_register.go）✅
  - `internal/core/`（settings.go/models.go 未知键告警 + template_aligned_test.go）✅
  - `internal/server/handler/plugins.go`（/api/plugins 去 clientApproved 输出）✅
  - `internal/hook/hook.go`、`pkg/summary/summary.go`（改名同步注释/兼容）✅
  - `.pair/plugins/`（tool-goal/tool-subagent/tool-workflow 新建；agent-teams fork/slash；tool-system answers 对齐）✅
  - `plugins-src/ui-app/`（api.js/RightPanel.vue/AskUserCard.vue/agent-events.js；删 5 死组件 + router.js + vue-router + ui-state.js.bak）✅
  - `plugins-src/plugins/`（无改动；归档源仅抽验）✅
  - 根 `web/` 旧壳 3 文件删除、`scripts/build-ui.mjs` 预清理、docs 回填 ✅
- **outOfScope 零改动**（commit 范围 `--name-only` 过滤实测）：
  - `.agent-teams/` ✅ 零条目；`release/` ✅ 零条目；`E:\` ✅ 无任何指向改动
  - 敏感配置（`config/settings.json`、`config/ai-presets.json`、`config/models.json` 运行时）✅ 零改动
  - 未跟踪文件：仅 `.verify-tmp/`（构建缓存，git-ignored）与 `docs/leftover-round3-verification.md`（本报告）
- 死包删除复核：`internal/{jobs,permission,provider,vterm}` 4 包在生产导入面零残留（grep 实测），
  `scripts/relocate_imports.go` 映射表已删 4 条（commit Round3-1 stat 核实）。

---

## 3. 装载冒烟（临时端口 9547，避开 9090/9479/9323）

方法：`go build -o companion_r3.exe ./cmd/companion`（67MB）置于仓库根（InstallDir 语义 → 命中
`<InstallDir>/.pair/plugins`），`WEB_PORT=9547` 启动 → 探活 → 终止并清理二进制。

| 探活项 | 结果 |
|---|---|
| `/api/health` | ✅ 200 `{"status":"ok","workspace":"F:\\syproject\\gou-ide","folders":[...]}` |
| `/api/plugins` | ✅ 200，**46 插件**全装载（含 tool-goal / tool-subagent / tool-workflow / agent-teams） |
| `/api/tools` | ✅ 200，**184 工具**；goal 3 工具（create_goal/get_goal/update_goal）、subagent 6 工具（subagent/subagent_fork/report/list_agents/interrupt_agent/send_message）、workflow 1 工具（workflow）全部注册且描述完整 |
| `/api/commands` | ✅ 200 `{ok:true, commands:[{name:"agent-teams", description:"团队面板/状态快照（/agent-teams [status]）", owner:"agent-teams"}]}` |
| `POST /api/commands/run` | ✅ 200 执行 `/agent-teams` 返回团队快照 JSON（leftover-round3 running / 4 成员 / 5 任务） |
| Web 根 `/` | ✅ 200（index.html 9.5KB → assets/index-BY0sezMm.js） |
| 日志扫描（logs/paircode.log 全量） | ✅ `全局插件宿主已初始化（46 个插件）` ×N、`已装载 47 个工具集（17 个插件）`（round2 44/17 → +3 新插件）；**FATAL/panic/重复/冲突/duplicate = 0 处** |

> 探活后进程已终止，`companion_r3.exe` 与冒烟临时文件已清理（git-ignored，工作树保持干净）。

---

## 4. 新机制功能抽查

| 机制 | 抽查方式 | 结果 |
|---|---|---|
| **goal 状态机 + 持久化** | TestGoalManager：create（默认轮次上限 3）→ 重复 create 拒绝 → get → update edit（revision 乐观锁 1→2）→ 过期 revision 拒绝 → pause（停续轮）→ resume（重挂）→ complete（终态不再续轮）→ 终态 resume 拒绝 → blocked（blockerReason 记录）→ **跨管理器实例读盘恢复**（新 GoalManager 从 `<wsRoot>/.pair/goals/<convID>.json` 恢复 blocked 状态 + 文件存在断言） | ✅ 全 PASS |
| **goal 自动续轮** | TestGoalAutoContinue：轮次上限 / pause / resume / 同一阻塞条件连续 ≥3 轮自动 blocked / 系统提示注入段（goalSystemMarker 幂等） | ✅ 全 PASS |
| **goal 工具面** | TestGoalArchiveExecutors（宿主执行器存档接线）+ 冒烟注册 3 工具 + tool-goal 插件 schema 对齐 DSH（create_goal objective 必填 / update_goal goal_id+revision+action 必填） | ✅ PASS |
| **subagent 后台执行与结果回收** | TestWorkflowRunner_Agent 真实 spawn 子 Agent 并等待 `subagent/idle` 回收 LastText（测试日志实测 `[subagent] 已启动成员会话 … label=wf-analyst` → `[workflow] agent 完成 … text=42 字符`） | ✅ PASS（真机路径） |
| **subagent fork** | TestAgentsForkSpec（SubAgentSpec.ForkOf 映射）+ TestJSPluginAgentsFork（ctx.agents.fork 桥）；subagent_spawn.go ForkSeed 源会话快照 ≤60 条 + 任务；ctx.agents.report 写入 SubAgentRecord.Report | ✅ 全 PASS |
| **workflow 编排** | TestWorkflowRunner_Agent/Pipeline/Parallel/PhaseLogArgs/Cancel/Errors 6 项（pipeline 逐项过阶段失败项 null、parallel barrier、phase/log 进度、args 输入、ctx 取消） | ✅ 全 PASS |
| **agent-teams fork 面** | agent-teams/index.js 实测：CFG.memberProvider = 设置 → env `AGENT_TEAMS_MEMBER_PROVIDER` → 默认 `spawn`，非法值回落 spawn；spawnMember fork 分支 `ctxRef.agents.fork({...spec, forkFrom: captain})`，失败 try/catch 回落 spawn；3 处调用点传 captainSessionId（approve/add_member/spawn 路径）；README 已更新 | ✅ 源码面核实 |
| **slash 命令** | 宿主 commands.go 注册表（owner 归属 + 卸载自动注销 + `/` 归一化）；冒烟 GET 清单 + POST 执行真实返回团队快照；RightPanel.vue 实测：`/` 前缀下拉（≤8 项）/↑↓/Enter 执行/无匹配 `sendMessage()` 原样发送降级；api.js listCommands/runCommand 两接口；agent-teams 注册 handler 读 `args.args` 与前端传参形状一致 | ✅ 运行 + 源码双核实 |
| **ask_user 多问题** | TestAskUserMultiQuestion_Parse/Routing/Executor（questions 参数编解码 + answers 数组回灌 + 执行器路由）；AskUserCard.vue 实测：questions prop 多问题渲染（text/single/multi 独立控件 + multiFormValid 全答校验 + 一次提交 answers 数组）；单问题路径原样保留；TestAskUserNoHostTimeout 单问题回归 PASS；Segment.Questions/Answers 新字段 + 旧字段序列化兼容（message_store.go 源码核实） | ✅ 全 PASS |

---

## 5. 旧名清理完整性（② 阶段 A-D）

| 核对项 | 结果 |
|---|---|
| 生产注册面零旧名 | ✅ TestNoOldNameRegistration PASS（read_file/write_file/edit_file/run_command/search_content/search_files/list_files 在默认注册面零命中；新名 read/write/edit/bash/glob/grep 存在且 handler 非空）；TestRegisterHarnessTools_Base 同断言 PASS |
| 新名基座 | ✅ registerCoreTools 注册 read/write/edit/bash/glob/grep（list_files 并入 glob 双模式、search_content→grep）；registerHarnessAliases 别名层删除（grep 零代码残留，仅注释提及） |
| Go 非测试生产文件旧名残留 | ✅ 仅「历史兼容分类」+ 注释（grep 全树分类核实）：history_condense.go:296-313、reviewer.go:88/129/168、storm_breaker.go:168、session_context.go:490-503、pkg/summary/summary.go:79-83、tools_staging.go:31-36（历史映射表保留）；tools.go/search.go/builtin_plugins.go/tools_concise.go/node_bridge.go 等仅改名为主题的注释 |
| 测试引用迁移 | ✅ Round3-3 迁移 19 个测试文件（17 必迁 + harness_filter 修正 + no_old_name 新断言）；残留旧名的测试文件全部为「纯历史消息 fixture / 模式匹配样例 / 任务提示 / 断言清单」（compress/condense_verify/evaluator/event_alignment/kv_cache/llm/loop_factory/loop_approve_state/storm_breaker/typed_tool/skill_loader/live_*/circling/compact_offset/history_condense/pluginization_fixes/debugtools/hook_test），与定案 §2 保留分类一致 |
| 归档核对 | ✅ plugins-src/plugins/tool-* Go 归档源（build-plugin-bins.bat 废弃不跑、运行时全 JS 插件）内旧名仅存于 UsageGuide 对比性描述文本（「比 run_command 更…」），非注册面；tool-harness 归档源抽验仅 run_code |
| 前端 | ✅ 生产显示映射保留旧名分支为「历史消息兼容壳」（chat-utils.js 新旧名双分支；RightPanel toolMeta/ApprovalBar/agent-events fileTools 仅旧名分支 → 新名调用走通用图标/不触发文件树刷新，显示级影响，见 F2）；config/ 角色文件零旧名（grep 实测） |
| 全量测试面 | ✅ 除 §1.1 环境依赖 14 项外全绿，无旧名相关断言失败 |

---

## 6. 发现（低危 / 信息，不阻塞）

| ID | 级别 | 内容 | 建议 |
|---|---|---|---|
| F1 | info | 前端构建在沙箱内失败（vite/esbuild `spawn EPERM`，DSH 文档化边界），与 t2 §10.1 偏差记录一致；**当前 `.pair/assets/runtime/web` 服务的是 round2 期 bundle**（实测不含 `/commands/run`、`multi_select`、`answers` 标记）——slash 菜单与 ask_user 多问题 UI 在重建前不会出现在运行界面 | 队长全权限通道（danger-full-access 已验证可用）复跑 `cd cmd/companion/web-ui && npm run build` + sync-web-dist.mjs，重建后冒烟复核前端 |
| F2 | low | `plugins-src/ui-app/src/components/RightPanel.vue` toolMeta（1069-1076）与 `ApprovalBar.vue`（61-66）显示映射、`agent-events.js:390` fileTools 刷新清单仅旧名分支——新名（read/write/edit/bash/glob/grep）调用显示通用图标/不触发文件树刷新；round2 起即存在（tool-harness 早已新名注册），非 Round3 回归 | 后续随手补新名分支（与 chat-utils.js 双分支对齐） |
| F3 | info | `internal/hook/hook_test.go` 以 read_file 等作模式匹配样例字符串（纯 fixture，非注册引用），与定案「纯 fixture 保留」分类一致 | 可不动 |
| F4 | info | go telemetry upload token stderr 噪音（沙箱限制，不影响构建/测试） | 可忽略 |

---

## 7. 结论

- **可构建性**：`go build ./...`、`go vet ./...` 全绿；companion exe 构建成功（67MB）。
- **测试**：非环境依赖测试全绿；14 项失败与 round2 §6.13 同根因（msys bash 信号管道 13 + rod 下载 1），
  新增 2 项同为信号管道签名，**无断言类回归**；Round3 新增 21 项测试全 PASS。
- **范围合规**：工作树干净；Round3-1…14 全部改动落在 t2 inScope；`.agent-teams/`、`release/`、
  敏感配置、`E:\` 安装目录零改动。
- **装载健康**：46 插件 / 47 工具集（17 插件）/ 184 工具（goal 3 + subagent 6 + workflow 1 全在）/
  /api/health 200 / /api/commands 1 命令且可执行 / 日志零 FATAL/panic/重复冲突。
- **新机制**：goal（状态机+持久化+自动续轮+3 工具）、subagent（后台执行+结果回收+fork）、
  workflow（6 项运行器测试）、agent-teams fork/slash（源码面 + 运行面双核实）、ask_user 多问题
  （宿主协议 3 测试 + 前端源码核实 + 单问题回归）全部达标。
- **旧名清理**：生产注册面零旧名（双测试断言 + grep 分类核实）；迁移后测试面全绿；
  残留旧名全部落在定案保留分类（历史兼容/纯 fixture/归档描述/前端兼容壳）。
- **待办**：前端重建（F1，需队长全权限通道）——t2 已声明、非本沙箱可执行项。

**总体判定：PASS（可进入 t4 代码审查）**，附 1 个信息项（F1 前端重建待队长通道执行）与 3 个低危/信息发现。

---

## 8. t5 集成复核（2026-09 · engineer 回填）

> t4 verdict=pass（7 个非阻塞 findings），t5 集成处置后复跑全量验证。基线：Round3-16（本报告追加时）。

| 项 | 结果 | 证据 |
|---|---|---|
| t4 findings 处置 | F1-F6 全部修复（relocate_imports vterm 残留、plugins-src 归档文案清扫、reviewer.go 新名补审、workflow CPU 看门狗、forkSeed 内存截断、answers ID 校验）；F7 随 F2 覆盖 | docs/plugin-round3-plan.md §11.1 |
| 前端构建 | 沙箱 `spawn EPERM`（web-ui npm run build 与 build-ui.mjs 均实测）→ 待队长 danger-full-access 通道 | 本报告 §6 F1 同根因 |
| CGO 构建 | `CGO_ENABLED=1 go build -o pair.exe ./cmd/companion` ✅ exit 0（67.2MB；pair.exe git-ignored） | 构建日志 |
| 启动冒烟（pair.exe） | /api/health 200；46 插件全 running；184 工具零缺失（goal 3/subagent 6/workflow 1）；/api/commands 返回 /agent-teams；零 FATAL/panic/重复注册 | 冒烟探活 |
| 测试 | 受 findings 修复影响的测试（TestAskUserMultiQuestion×3/TestValidateAskAnswers/TestAskUserNoHostTimeout）全 PASS；全量 agent 测试 14 项环境依赖失败同 §1.1 清单 | go test |
| git 收尾 | 工作树干净（pair.exe 忽略）；Round3 全量改动落 t2 inScope；.agent-teams/release/敏感配置/E:\ 零改动 | git status |

**结论：Round3 可交付。唯一待办 = 前端重建（队长全权限通道复跑 web-ui npm run build + sync-web-dist.mjs）。**
