# Round3 遗留专项：定案与实施计划（t1 · analyst · 2026-09）

> 本文件为 **leftover-round3** 团队 t1（requirements）只读分析交付物。
> 定案依据：前两轮文档（docs/plugin-gaps.md、docs/plugin-round2-plan.md §6-§7、
> docs/plugin-round2-verification.md、docs/pluginization-gap-analysis.md）+ 本轮逐项实测复核。
> 本轮除本文件外**未改动任何源码/插件/配置**。每项遗留给出最终处置
> （删除 / 迁移 / 接线 / 实现）与实施顺序，供 t2 实施、t3 验证、t4 审查、t5 集成。

---

## 0. 本轮实测基线（2026-09 复核，全部重验）

| # | 遗留项 | 实测事实 | 证据 |
|---|---|---|---|
| ① | 死包 4 个 | `internal/jobs`（jobs.go+test）、`internal/permission`（permission.go+test）、`internal/provider`（5 源 3 测试）、`internal/vterm`（vterm.go+test）均在；全仓生产导入 = 0（仅 `scripts/relocate_imports.go:21-26` 字符串映射表提及） | glob + grep 实测 |
| ② | Go 旧名工具 | 旧名注册面：`registerCoreTools`（tools.go:401，read_file/write_file/edit_file/list_files/run_command + multi_edit/move_file/delete_file）、`registerSearchTools`（search.go:58，search_content/search_files）、别名层 `registerHarnessAliases`（harness_tools.go:61，read←read_file 等 6 条）；`RegisterDefaultTools→RegisterToolGroups→builtinPluginSpecs`（core/fs-search 两组）；测试引用 ≈92 处 / 27 文件（含纯历史消息 fixture） | grep 实测 |
| ③ | goal/subagent/workflow | 宿主 Loop = `Loop.Run`（loop.go:534，一次 Run=一个 turn，agentloop 双层状态）；无 goal 状态机；ctx 服务面 = fs/web/bash/sse/ws/logger/timer/kernel/market/mcp/skill/toolset/npm/plugins/process/**agents**/llm/http（jsplugin.go:3029-3037）——无 commands/goal/workflow/fork | read + grep |
| ④ | agent-teams fork/slash | `ctx.agents` 仅 start/followup/stop/running/status/list/lastText/ready（jsplugin_agents.go）；`spawnMember` 固定 `ctx.agents.start`（index.js:811-834）；无 ctx.commands 面；前端输入框 RightPanel.vue `onKeydown` 无 "/" 命令处理 | read + grep |
| ⑤ | ask_user 多问题 | 插件侧 schema 已有 `questions`（R2-7）；**Go 宿主侧未同步**：session_bridge.go:122 存档执行器 + session_manager.go:637 工具 + `askCh chan string` + SendAnswer（1148）+ WaitAnswer（1168）+ handleChatAnswer（web_server.go:2386）均为单问题；`Segment`（message_store.go:26-30）单 Question/Options/Answer；AskUserCard.vue 单卡片 | read + grep |
| ⑥ | P2 清理 | tools_concise.go:43-44 含 search_content/search_files 死条目；plugin.go:464-884 审批遗留（approvedClients/approvedGlobal/approvedProject + load/save 文件机制 + SetApprovedGlobalDir，JS 消费方仅旧 built 产物 web/assets）；config/ 模板 3 个（settings/models/mcp.template.json）；前端死件：OutputPanel/SubAgentBlock/TasksPanel 零导入、router.js 未挂载（main.js/ShellApp 无 router，文件自注「未使用 <router-view>」）、PlanView/TaskBoard 仅被死 router 引用、ui-state.js.bak 被跟踪、根 `web/` 旧壳 3 文件被跟踪 | grep + read |

**结论**：前两轮遗留清单全部属实，无新增漂移；Round2 已闭环项（43 插件/17 工具集条目/零 FATAL/243 工具零未落地）保持。

---

## 1. ① 死包最终处置 → **删除（4/4，不再保留）**

前两轮「待产品确认」在本轮定案为**删除**。共性依据：生产导入 0（实测）、功能已被替代实现完全继承、测试均为包内自测（删除后随包消失，无外部消费）。

| 包 | 现状 | 功能继承者 | 处置 | 风险 |
|---|---|---|---|---|
| internal/jobs | jobs.go 9.4KB + test | `internal/agent/shell.go` globalBG + tool-shell 插件（run_background/read_output/kill_process + job_output/job_list/job_kill 别名） | **git rm 删除** | 低（零导入） |
| internal/permission | permission.go 9.9KB + test | 审核门 = 工具 `RequiresApproval`/`DynamicApproval` 元数据 + Loop 审批门（loop.go:900、loop_parallel.go resolveApproval）+ `internal/hook` 信任门 | **git rm 删除** | 低 |
| internal/provider | provider.go/dotenv/effort/retry/schema 5 源 3 测试 | `internal/agent/provider.go` + `provider_impls.go`（S1 实现注册表，已全生产构造点路由） | **git rm 删除** | 低（Round2 已实测无 build tag 引用） |
| internal/vterm | vterm.go 11.9KB + test | wb-ui 前端 TerminalPanel（PTY 渲染由前端承担） | **git rm 删除** | 低 |

**连带动作**（同一次提交）：
1. `scripts/relocate_imports.go:21-26` 删除 4 条字符串映射（工具不再指向不存在的包）。
2. 删除前 `grep -r "internal/jobs|internal/permission|internal/provider|internal/vterm" --include=*.go` 复核零导入（本轮已验，t2 删除前再跑一次）。
3. docs 同步：plugin-gaps.md §遗留项、plugin-round2-plan.md §7.2 状态改为「已删除」。

**验收**：`go build ./...` + `go vet ./...` 通过；`TestToolLandingMatrix` 等工具面测试不涉及死包（无引用）。

---

## 2. ② Go 旧名工具清理 → **先迁移测试、再删注册/归档（4 阶段）**

**目标态**：Go 生产 + 测试注册面**零旧名**（read_file/write_file/edit_file/run_command/search_content/search_files/list_files）；Go 实现以**新 harness 名**保留为「测试/归档基座」（生产语义在 tool-harness JS 插件，Go 侧不重复生产注册）；**历史消息兼容分类保留**。

### 阶段 A —— 改名基座（新名 = 基座，别名层删除）

- `tools.go registerCoreTools`：`read_file→read`、`write_file→write`、`edit_file→edit`、`run_command→bash`、`list_files→glob`（与 tool-core 插件对齐的 multi_edit/move_file/delete_file 保留原名）；**参数形态保持现状（`path`）**——Go 侧仅测试/归档基座，DSH 参数语义（file_path/replace_all/timeoutMs）以 tool-harness JS 插件为准（R2-7 已对齐），不在 Go 侧重复。
- `search.go registerSearchTools`：`search_content→grep`、`search_files→glob`；与 core 组合并后删除本函数（glob/grep 落入 registerCoreTools）。
- `harness_tools.go`：**删除 `registerHarnessAliases` + `harnessAlias` 结构**（别名层失去来源即无意义）；`RegisterHarnessTools` = 新名基座（read/write/edit/bash/glob/grep/str_replace_editor/run_code）；头注释更新。
- `builtin_plugins.go builtinPluginSpecs`：core 组描述改新名；fs-search 组并入 core（或改名）；`RegisterDefaultTools/RegisterToolGroups` 语义不变。
- `tools_staging.go:29-34` 新旧名等价表**保留**（staging UI 对历史消息展示的映射文档），仅注释标注「历史映射」。
- `tools_concise.go:43-44`：删 search_content/search_files 条目（glob/grep 已有条目）。

### 阶段 B —— 迁移测试引用（≈92 处 / 27 文件，分类处理）

| 类别 | 文件 | 动作 |
|---|---|---|
| **必迁**（注册+执行 / 断言注册表 / 循环执行工具调用） | tools_test、search_test、binary_test、shell_test、multiproject2_test、loop_test、agentloop_test、jsloop_e2e_test、event_consistency_test、loop_service_test、livesnapshot_ws_test、harness_tools_test（别名映射表→基座断言、list_files→glob）、builtin_toolset_test、cache_diag_test、tools_staging_test、toolset_prune_test、node_bridge_bind_test | 工具名改新名（read_file→read、write_file→write、edit_file→edit、run_command→bash、search_content→grep、search_files/list_files→glob）；参数不变（path） |
| **保留**（纯历史消息 fixture，不参与注册/执行） | compress_test、history_condense_test、evaluator_test、event_alignment_test、kv_cache_test、llm_test、loop_factory_test、loop_approve_state_test、compact_offset_test、condense_verify_test、storm_breaker_test、circling_test、typed_tool_test、skill_loader_test（AllowedTools 字符串）、live_e2e_test、pluginization_fixes_test（review 配置迁移 fixture） | **不动**；注释标注「历史消息 fixture，非注册引用」 |
| **兼容保留**（生产历史分类） | history_condense.go:296-313、reviewer.go:88/129/168、storm_breaker.go:168、session_context.go:490-503（fileModifyTools/toolPathParams 旧名条目=历史消息兼容，R2 F1 已加新名） | **不动**（删旧名会破坏历史消息处理）；bridge_controller.go:257/277/281 record 标签为审计历史名，可顺手改 read/write（可选） |

### 阶段 C —— 删除旧名注册

- `tools.go`：删除 read_file/write_file/edit_file/run_command/list_files 五个旧名 Tool 注册（新名版本已在阶段 A 就位）。
- `search.go`：删除 search_content/search_files 注册（glob/grep 已并入 core）。
- 同步清理注释：tools.go:398（RegisterDefaultTools 头注）、tools_staging.go:23、builtin_plugins.go 头注、harness_tools.go 头注、plugin-gaps.md/plugin-round2-plan.md 相关段落。

### 阶段 D —— 归档核对

- `plugins-src/plugins/tool-*` Go 归档二进制经 `RegisterToolGroups` 走 builtinPluginSpecs——阶段 A 改名后自动新名，无需逐个改（t2 抽验 tool-harness/tool-core 归档源）。
- 最终 grep 断言：`"(read_file|write_file|edit_file|run_command|search_content|search_files|list_files)"` 在 Go 非测试生产文件仅剩「历史兼容分类」处（history_condense/reviewer/storm_breaker/session_context + 注释）。

**验收**：`go build ./...`、`go vet ./...`、`go test ./internal/agent/ -count=1 -timeout 10m`（环境依赖除外，清单沿用 round2 §6.13）全绿；`TestToolLandingMatrix` 通过；grep 零残留断言成立。**分阶段提交**（A/B/C 各一 commit），回滚=git revert。

---

## 3. ③ harness 机制插件化 → **宿主 Loop 机制级支持 + 插件化工具面**

原则：机制（状态机/会话编排）在宿主 Loop 层，工具面（schema/描述）在插件层（与 R2 一切皆插件同构）；无 goal/无插件时**零行为变化**。

### 3.1 goal（P1，全量实现）

- **宿主状态机**：`internal/agent/goal.go`（新）——`Goal{ID, Revision, Objective, Phase, Rounds, RoundLimit, BlockerReason, Armed}`；会话级（SessionManager 按 convID 持有）+ 持久化 `.pair/goals/<convID>.json`（防宿主重启丢目标）；`create_goal` 直接接收 objective（不做 LLM 推断，减少不确定面）。
- **Loop 接线（机制级）**：
  - `Loop.Run` 启动时若有活动 goal → 目标注入系统提示（`goal objective + phase + rounds` 段，对齐 DSH「同会话完成目标」语义）；
  - **自动续轮**：`SessionManager` 在 Run 返回后检查——goal Armed && !complete && Rounds < RoundLimit → 自动发起下一轮（continuation 消息），即「goal rounds」；pause 停止续轮、resume 重挂 Armed；同一阻塞条件连续 ≥3 轮 → 自动 blocked（blocked_reason 记录）。
- **插件**：`.pair/plugins/tool-goal`（新）——3 工具 `create_goal`/`get_goal`/`update_goal`，schema 对齐 DSH（create：objective/max_goal_rounds；get：返回 goal_id/revision/objective/phase/rounds/roundLimit/blockerReason/armed；update：goal_id/revision/action∈{edit,pause,resume,complete,blocked}+objective/max_goal_rounds/blocked_reason），execute → `ctx.hostTool.exec`。
- **宿主执行器**：`archiveGoalTools`（同 ask_user 会话桥模式，InitLoopHooks 同级接线）。
- **测试**：`TestGoalManager`（create/get/update 全 action + revision 冲突拒绝 + 持久化）、`TestGoalAutoContinue`（会话层续轮/轮次上限/blocked）。

### 3.2 subagent（P2，裁剪实现：通用工具 + fork）

- **宿主**（复用现有 subagent_registry）：
  - **fork 机制**：`SubAgentSpec` 增 `ForkOf string`（源会话 convID）；`SubAgentSpawner` 增 `Fork func(spec SubAgentSpec, seed []Message) error`（web 层实现：新会话初始历史 = 源会话当前消息快照 + 任务，persona/模型沿用 spec）；
  - `ctx.agents` 增 `fork({task, forkFrom: convId, ...})` 与 `report(text)`（子会话报告写入 `SubAgentRecord.Report`，status/list 可读）；
  - `list_agents`/`interrupt_agent`/`send_message` 复用既有 list/stop/followup。
- **插件**：`.pair/plugins/tool-subagent`（新）——`subagent`（后台委托）、`subagent_fork`（fork 委托，run_in_background 默认 true）、`report`、`list_agents`、`interrupt_agent`、`send_message`（DSH 命名对齐；agent-teams 不受影响）。
- **测试**：`TestAgentsForkSpec`（spec 映射）、`TestJSPluginAgentsFork`（ctx.agents.fork 桥）。

### 3.3 workflow（P2，最小可用）

- **宿主**：`internal/agent/workflow.go`（新）——goja 执行 workflow 脚本（meta{name,description,phases} + script 体），钩子：`agent(prompt, opts)` → ctx.agents.start + 等待 `subagent/idle` 事件取结果（SubAgentLastText）；`pipeline(items, ...stages)`（逐项过阶段，无 barrier）；`parallel(thunks)`（并发，**上限 4**）；`phase(title)`/`log(msg)` → 进度事件；返回 JSON 结果。会话级运行态（running map + cancel）。
- **插件**：`.pair/plugins/tool-workflow`（新）——`workflow` 工具（script/meta/args 参数，execute → `ctx.hostTool.exec`）。
- **范围声明**：不做跨会话恢复/持久化队列（记录为后续演进）。
- **测试**：`TestWorkflowRunner`（agent 1 个 + pipeline 2 项 + 并发上限 + 取消）。

---

## 4. ④ agent-teams fork 能力面 + slash 命令 → **宿主两面 + 插件接线**

### 4.1 memberProvider fork

- **宿主面**：`ctx.agents.fork`（§3.2）。
- **插件**：`.pair/plugins/agent-teams/index.js`——`CFG.memberProvider`（`'spawn'|'fork'`，默认 `'spawn'` 行为不变；配置来源：插件配置/环境变量，读取失败回落 spawn）；`spawnMember` 按配置分支：fork 时 `ctx.agents.fork({task, forkFrom: captainSessionId, system: persona, model, provider, reasoningEffort, wsRoot, denyTools})`（fork 种子=队长会话历史快照，persona 经 system 覆盖）；README 增 memberProvider 说明。
- **验收**：`TestAgentTeamsDiskLoad` 通过；node --check；配置 fork 后 spawnMember 日志走 fork 分支。

### 4.2 slash 命令（宿主 ctx.commands 面）

- **宿主**：`internal/agent/commands.go`（新）——命令注册表 `{name, description, handler(goja)}` + `buildCommandsService(pc)`（jsplugin inject 增 `commands`；插件 `ctx.commands.register({name, description, handler})`，插件卸载自动注销）；HTTP 面 `GET /api/commands`（清单）+ `POST /api/commands/run`（{name, args} → 执行结果；web_server.go 或 kernel_register.go 注册）。
- **前端**：`RightPanel.vue` `onKeydown`——输入以 `/` 开头 → 拉 `/api/commands` 下拉提示；Enter 执行 `POST /api/commands/run`，结果以系统/用户消息注入对话（`api.js` 增两接口）；**无匹配命令时 "/" 输入原样发送（降级零破坏）**。
- **插件**：agent-teams 注册 `/agent-teams`（无参=团队面板/status 快照；子命令 `/agent-teams status|resume` 按需）。
- **验收**：输入 `/agent-teams` 出现菜单并可执行；插件卸载命令消失；无命令时斜杠输入原样发送。

---

## 5. ⑤ ask_user 多问题前端 UI → **前后端全链路实现（向后兼容）**

- **宿主协议**：
  - `session_bridge.go` 存档执行器（:122）：参数支持 `questions: [{id, question, options?, multi_select?}]`（与 `question` 并存：questions 优先，缺省回落单问题路径）；
  - `session_manager.go`：`askCh chan string` → `askCh chan []AskAnswer`（AskAnswer{ID, Answer}）；:637 注册工具、SendAnswer（:1148）、WaitAnswer（:1168）同步结构化；单问题路径编为单元素数组（兼容）；
  - `message_store.go Segment`：增 `Questions []SegmentQuestion{ID, Question, Options, MultiSelect}` + `Answers []SegmentAnswer{ID, Answer}`（旧 Question/Options/Answer 字段保留序列化兼容）；segment 组装（:200-243）多问题分支；
  - `web_server.go handleChatAnswer`（:2386）：收 `{convID, callID, answers:[{id, answer}]}`（兼容旧 `{answer}`）。
- **前端**：`AskUserCard.vue`——`questions` prop 渲染多问题列表（每问题按 text/single/multi 独立控件），一次「发送」提交 `answers` 数组；单问题渲染路径不变。
- **插件**：tool-system schema 已就绪（R2-7），仅对齐 answers 语义与描述（「传 questions 时请同时给 question 合并文本」的降级说明撤销）。
- **验收**：`TestAskUserMultiQuestion`（宿主路由多问题编解码 + 回答回灌）；前端多问题卡片渲染+提交（沙箱外构建验证）；`TestAskUserNoHostTimeout` 单问题回归通过。

---

## 6. ⑥ P2 清理 → **按四项分别处置**

### 6.1 tools_concise 描述
- 删 `search_content`/`search_files` 死条目（随 ② 阶段 A）；
- `ask_user` 条目补 questions 语义；
- **保留宿主框架能力定位**（ApplyConciseToolDescriptions 属 harness 对齐行为，t1 已认可）；配置化（config/tools-concise.json 覆盖表）列为可选加分项，不阻塞主线。

### 6.2 审批遗留字段（plugin.go L4-旧）
- **删除**：`approvedClients/approvedGlobal/approvedProject` 三个 map 字段 + `loadApprovedClients/saveApprovedClients/loadApprovedFile/saveApprovedFile/approvedFilePath` 全套 + `SetApprovedGlobalDir`（测试隔离专用，同步清理其测试调用）；
- `MarkClientApproved`：先 grep 调用点，零调用则删除（plugin_tools.go:175 注释已声明不再调用）；
- `IsClientApproved` + 记录字段 `ClientApproved`：JS 源码消费者 = 0（仅旧 built 产物 web/assets，随 6.4 删除）→ **一并删除字段与函数**，`/api/plugins` 记录不再输出 clientApproved（dist 重建后前端零引用）；
- **验收**：`go build` + `TestPluginHost*` 相关测试全绿；grep `approvedGlobal|approvedProject|approvedClients|ClientApproved|IsClientApproved|MarkClientApproved` 仅剩文档历史提及。

### 6.3 配置模板漂移/校验（G2/G4 裁剪版）
- **模板对齐**：`config/settings.template.json` 键集 ⊆ `AppSettings`（internal/core/settings.go:16）键集 + 默认值 == `Default()`（现模板 26 键与 AppSettings 基本一致，差异以测试固定：AI 业务字段 omitempty 不出现在模板属预期）；
- **新增测试** `TestTemplateSchemaAligned`：解析 3 个 template（settings/models/mcp），断言键集 ⊆ 对应加载结构体键集、默认值一致（models/mcp 对齐 internal/core/models.go 与 mcp 加载器 schema，t2 先 diff 再修模板或加载器，**以加载器为准**）；
- **启动校验**：`core.Load` 增未知键/类型告警（json.Unmarshal 后 diff raw keys，log.Warn 不阻断）；models.json/mcp.json 同理（packager stripSecrets 已处理发布脱敏，不动）。
- **验收**：模板测试通过；`TestReviewConfigSingleSourceMigration` 等配置测试回归。

### 6.4 前端死组件/产物（F2/F3）
- **删除源码死件**：`plugins-src/ui-app/src/components/OutputPanel.vue`、`SubAgentBlock.vue`、`TasksPanel.vue`、`PlanView.vue`、`TaskBoard.vue`（零导入实测；PlanView/TaskBoard 仅被未挂载的 router 引用）、`src/router.js`；`package.json` 移除 vue-router 依赖（先 grep 全 src 确认无动态引用）；
- **删除跟踪产物**：`plugins-src/ui-app/src/ui-state.js.bak`、根 `web/` 旧壳（`web/index.html` + `web/favicon.svg` + `web/assets/index-CrzpNXyK.js`，仅 docs 提及，无 Go/packager 引用）；
- **构建流水线防堆积**：`scripts/build-ui.mjs` 构建前清理 `cmd/companion/web-ui/dist` 旧 bundle（14 个 index-*.js 历史产物）；`.pair/assets/runtime/web` 由 sync-web-dist.mjs 同步覆盖（dist/runtime 均 git-ignored，本地清理 + 流水线保证）；
- **验证**：`cd cmd/companion/web-ui && npm run build`（沙箱 EPERM → 队长全权限通道执行）；dist 无旧 bundle；grep `OutputPanel|SubAgentBlock|TasksPanel|PlanView|TaskBoard|vue-router` 源码零残留。

---

## 7. 实施顺序与任务划分（t2 建议 inScope）

| 序 | 工作包 | 依赖 | 提交建议 |
|---|---|---|---|
| 1 | ① 死包删除（4 包 + relocate_imports 表 + docs） | 无 | 独立 commit |
| 2 | ② 阶段 A 改名基座 + 阶段 C 旧名删除（同一批编译原子） | 无 | commit A/C 合并或相邻 |
| 3 | ② 阶段 B 测试迁移（≈17 文件必迁） | 2 | 独立 commit（最大面，先于 4） |
| 4 | ② 阶段 D 归档核对 + grep 断言 + tools_concise 死条目 | 2、3 | 独立 commit |
| 5 | ⑥.2 审批遗留字段 + ⑥.3 配置模板/校验 | 无（可与 2-4 并行） | 独立 commit |
| 6 | ③.1 goal（宿主状态机 + Loop 接线 + tool-goal 插件 + 测试） | 无 | 独立 commit |
| 7 | ③.2 subagent fork（宿主 Fork + ctx.agents.fork/report + tool-subagent 插件） | 无（可并行 6） | 独立 commit |
| 8 | ③.3 workflow 运行器 + tool-workflow 插件 | 7（agent 钩子复用） | 独立 commit |
| 9 | ④.1 agent-teams memberProvider fork | 7 | 独立 commit |
| 10 | ④.2 ctx.commands + /api/commands + 前端 slash 菜单 + agent-teams /agent-teams | 无 | commit（前后端同批，前端构建需沙箱外） |
| 11 | ⑤ ask_user 多问题（宿主协议 + Segment + 前端卡片） | 无 | commit（同批前后端） |
| 12 | ⑥.4 前端死件/产物 + 构建流水线清理 | 无（最后做，需沙箱外构建） | 独立 commit |

**验证基线**（t3 沿用）：`go build ./...`、`go vet ./...`、`go test ./internal/agent/ -count=1 -timeout 10m`（环境依赖清单沿用 round2 §6.13：msys bash 信号管道 / rod 下载 / go-node 子进程）；插件 `node --check` 全量；冒烟（临时 WEB_PORT）43 插件 / 17 工具集条目 / `/api/health` 200 / 零 FATAL；前端 `cd cmd/companion/web-ui && npm run build`（沙箱外）。

---

## 8. 风险与回滚

1. **② 迁移面最大**：分阶段提交（A/C → B → D），任一阶段可独立 git revert；纯 fixture 字符串不动，避免无谓 churn；必迁文件清单（§2 阶段 B 表）作为 t2 核对表。
2. **③ goal 自动续轮**：默认关闭（无 create_goal 时零行为变化）；续轮受 RoundLimit 与 pause 控制；blocked 判定需同一阻塞条件 ≥3 轮（对齐 DSH 语义），防误判。
3. **③.3 workflow 并发**：并行上限 4，防成员风暴；运行态可取消。
4. **④ slash 降级**：无匹配命令时 "/" 原样发送；命令注册表随插件卸载自动注销，无悬挂。
5. **⑤ 协议变化**：answers 数组与旧 answer 字段双兼容；单问题路径行为不变。
6. **⑥.4 前端构建**：沙箱 EPERM 已知（vite/esbuild spawn），需队长全权限通道复跑 `npm run build`；删除 router/vue-router 前 grep 动态引用。
7. **敏感配置零触碰**：config/settings.json、config/ai-presets.json、config/models.json（运行时）均 git-ignored，任何工作包不得修改。
8. **API 形状变化**（/api/plugins 去 clientApproved、/api/chat/answer 收 answers）：dist 随前端重建同步；E:\ 旧安装目录 UI 由发布流程覆盖，不单独兼容。

---

## 9. 交付物清单（t2 实施后回填）

- 本文件 §1-§8 各项处置的「已实施 / 未实施（原因）」回填（t2 实施记录 + t3 验证报告 + t4 审查 + t5 集成结论，格式沿用 plugin-round2-plan.md §6-§8）。
- 新增测试：TestGoalManager / TestGoalAutoContinue / TestAgentsForkSpec / TestJSPluginAgentsFork / TestWorkflowRunner / TestCommandsRegistry / TestAskUserMultiQuestion / TestTemplateSchemaAligned / TestNoOldNameRegistration（grep 断言）。

---

## 10. t2 实施记录（2026-09 · engineer 回填）

> 12 个工作包全部按定案实施，11 个独立 commit（Round3-1 … Round3-12）。
> 验证基线：`go build ./...` ✅、`go vet ./...` ✅（GOCACHE/GOMODCACHE 重定向 .verify-tmp/，CGO_ENABLED=1）；
> `go test ./internal/agent/ -count=1 -timeout 10m` 仅环境依赖失败（msys bash 信号管道组 + TestWebDebugTimeoutMs rod，
> 与 round2 §6.13 清单一致；TestToolLandingMatrix 295 工具零未落地）；`go test ./internal/core/ ./internal/server/...` ✅。
> 插件冒烟（临时 WEB_PORT，exe 置于仓库根以命中 <InstallDir>/.pair/plugins）：46 插件全装载、/api/health 200、
> /api/commands 返回 agent-teams 命令、184 工具注册（goal 3 + subagent 6 + workflow 1 全在）、零 FATAL/重复注册。

| 序 | 工作包 | 状态 | 证据 |
|---|---|---|---|
| 1 | ① 死包删除 | ✅ | `git rm` 4 包 14 文件（删除前 grep 复核零导入）；relocate_imports.go 删 4 条映射；commit Round3-1 |
| 2 | ② 阶段 A/C 改名基座 | ✅ | registerCoreTools：read/write/edit/bash/glob（list_files 并入 glob 双模式：有 pattern 递归查找/无 pattern 目录列举）+ grep（原 search_content）；删 registerHarnessAliases 别名层与 fs-search 组（并入 core）；tools_concise 删死条目；commit Round3-2 |
| 3 | ② 阶段 B 测试迁移 | ✅ | 17 文件必迁批量改名（含 harness_filter_test pair 清单修正，Round3-12）；纯历史 fixture 17 文件保留不动；commit Round3-3 |
| 4 | ② 阶段 D 归档核对 | ✅ | tool-harness 归档源抽验（仅 run_code 无旧名）；TestNoOldNameRegistration 零旧名断言；tool_landing 全量矩阵通过 |
| 5 | ⑥.2 审批遗留字段 | ✅ | plugin.go 删 approvedClients/approvedGlobal/approvedProject + load/save 全套 + SetApprovedGlobalDir + IsClientApproved/MarkClientApproved/ClientApproved（/api/plugins 输出同步删）；commit Round3-4 |
| 6 | ⑥.3 配置模板/校验 | ✅ | TestTemplateSchemaAligned（settings/models/mcp 键集⊆结构体 + 默认值一致，ignoreDirs 为模板增强默认以测试固定）；core.Load/LoadModelList 未知键告警；commit Round3-5 |
| 7 | ③.1 goal | ✅ | goal.go：Goal 状态机 + GoalManager（.pair/goals/<convID>.json 持久化、revision 乐观锁、edit/pause/resume/complete/blocked、同一阻塞条件连续 ≥3 轮自动 blocked）；SessionManager 自动续轮 + 系统提示注入；tool-goal 插件（3 工具）；TestGoalManager/TestGoalAutoContinue/TestGoalArchiveExecutors；commit Round3-6 |
| 8 | ③.2 subagent fork | ✅ | SubAgentSpec.ForkOf + Spawner.Fork/ForkSeed（web 层 subagent_spawn.go：源会话快照≤60 条 + 任务）+ ctx.agents.fork/report（SubAgentRecord.Report/ForkOf，status/list 可读）；tool-subagent 插件 6 工具；TestAgentsForkSpec/TestJSPluginAgentsFork；commit Round3-7 |
| 9 | ③.3 workflow | ✅ | workflow.go：goja 运行器（agent 同步等待子 Agent 完成取 LastText/pipeline 逐项过阶段失败项 null/parallel barrier（goja 单线程顺序调度，并发由宿主子 Agent 承载，天然满足上限 4）/phase/log/args + ctx 取消）；tool-workflow 插件；TestWorkflowRunner 5 项；commit Round3-8 |
| 10 | ④.1 memberProvider fork | ✅ | agent-teams CFG.memberProvider（设置→env AGENT_TEAMS_MEMBER_PROVIDER→默认 spawn；非法回落 spawn）；spawnMember fork 分支（forkFrom=队长会话，失败回落 spawn）；3 处调用点传 captainSessionId；README 增说明；commit Round3-9 |
| 11 | ④.2 ctx.commands + slash | ✅ | commands.go 注册表（owner 归属、卸载自动注销、/名归一化）+ ctx.commands 服务（withLock 跨 goroutine）+ GET/POST /api/commands（结果系统消息注入会话）+ core-api 清单 + RightPanel.vue "/" 菜单（下拉/↑↓/Enter 执行/无匹配原样发送）+ api.js 2 接口 + agent-teams 注册 /agent-teams；TestCommandsRegistry/TestJSPluginCommandsBridge；commit Round3-9 |
| 12 | ⑤ ask_user 多问题 | ✅ | askCh chan []AskAnswer（单问题=单元素数组）；questions 参数/answers 回灌（JSON）；Segment.Questions/Answers（旧字段保留兼容）；/api/chat/answer 收 answers 双兼容；AskUserCard 多问题渲染（text/single/multi 控件 + 一次提交）；agent-events.js 实时分支同步；tool-system schema 对齐（撤销「多问题 UI 属前端专项」降级说明）；TestAskUserMultiQuestion（解析/路由回环/执行器）+ 单问题回归；commit Round3-10 |
| 13 | ⑥.4 前端死件/产物 | ✅ | 删 OutputPanel/SubAgentBlock/TasksPanel/PlanView/TaskBoard/router.js + vue-router 依赖 + ui-state.js.bak + 根 web/ 旧壳 3 文件（删除前 grep 零导入/零 Go 引用）；build-ui.mjs 预清理 dist 历史 bundle（sync-web-dist.mjs 双保险）；commit Round3-11 |

### 10.1 未实施/偏差说明

- **前端构建（`cd cmd/companion/web-ui && npm run build`）**：沙箱内未执行（vite/esbuild spawn 已知 EPERM 边界）；
  前端源码改动（api.js/RightPanel.vue/AskUserCard.vue/agent-events.js）**待队长全权限通道构建验证**（danger-full-access 已验证可用）。
- **workflow parallel 并发**：goja 单线程模型下 JS 侧顺序调度（barrier 语义保持），并发性由宿主子 Agent 承载；
  已在代码注释与本文档 §3.3 声明为设计取舍。
- **TestHarnessAlias_GlobGrepBash**：glob/grep 断言全过，bash 步为 msys 信号管道环境依赖（round2 §6.13 同列）。
- **归档二进制**（plugins-src/plugins/tool-*）：按定案抽验 tool-harness（仅 run_code）；其余组经
  RegisterToolGroups 自动新名注册，无需逐个改；build-plugin-bins.bat 仍废弃不跑。

### 10.2 新增测试清单（全部通过）

TestNoOldNameRegistration / TestRegisterHarnessTools_Base / TestGoalManager / TestGoalAutoContinue /
TestGoalArchiveExecutors / TestAgentsForkSpec / TestJSPluginAgentsFork / TestWorkflowRunner_Agent /
TestWorkflowRunner_Pipeline / TestWorkflowRunner_Parallel / TestWorkflowRunner_PhaseLogArgs /
TestWorkflowRunner_Cancel / TestWorkflowRunner_Errors / TestCommandsRegistry / TestJSPluginCommandsBridge /
TestAskUserMultiQuestion_Parse / TestAskUserMultiQuestion_Routing / TestAskUserMultiQuestion_Executor /
TestTemplateSchemaAligned / TestAskUserNoHostTimeout（单问题回归）。
