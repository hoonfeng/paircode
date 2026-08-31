# Round4 ① Go 核心存量审计：internal/agent + cmd/companion + internal/server（t1 · analyst · 2026-09）

> 本文件为 **runtime-complete-round4** 团队 t1（requirements）只读分析交付物之一。
> 定案依据：前三轮文档（docs/plugin-gaps.md、plugin-round2-plan.md、plugin-round3-plan.md、
> go-core-capabilities.md、pluginization-gap-analysis.md）+ 本轮逐文件实测复核（grep/read）。
> 审计范围：`internal/agent`（非测试 138 文件）、`cmd/companion`（8 文件）、`internal/server`（10 文件）。
> 分类标准：**应插件化候选** = 逻辑是 agent 可见能力/领域能力，可经「能力在宿主、编排在插件」seam
> 迁出宿主直接注册面（或当前处于未装配孤儿态）；**应保留宿主基础设施** = 循环/会话/存储/传输/
> 插件宿主/服务面等，插件化会引入自锁或形态不匹配（沿用 go-core-capabilities.md §4 判断准则）。
> 本轮除本文件外**未改动任何源码/插件/配置**。

---

## 0. 审计结论速览（TL;DR）

| 类别 | 数量 | 处置建议 |
|---|---|---|
| **A. 应插件化候选（本轮新发现，未处理）** | 2 组孤儿 Go 工具 + 2 个死包 + 1 个半插件化面 | 归档/删除/封装（见 §1-§4） |
| **B. 应保留宿主基础设施** | ~120 文件 | 保留（见 §5 分类表） |
| **C. 已插件化但 Go 实现保留为存档/测试基座** | 20 组工具库 | 保留，风险=双轨漂移（见 §3） |

**核心结论**：前三轮插件化已把「agent 可见工具面」全部迁出宿主直接注册（宿主仅剩框架协议工具
`RegisterHostFrameworkTools` + 会话级 `RegisterMCPServers` + 自主模式 `RegisterPlanOnlyTools`）。
本轮审计未发现新的「大块未插件化逻辑」，但发现 **4 处零生产装配残留**（与 t1 报告 T1 缺口同型、
前三轮漏网）+ **1 处 MCP 工具面无插件归属**。JS 运行时缺口见配套文档 `docs/round4-js-runtime-upgrade.md`。

---

## 1. A1 孤儿 Go 工具实现（零生产注册、零插件声明、未存档）→ **本轮处置：归档或删除**

与 plugin-gaps.md T1 缺口（7 组孤儿工具）同型，但**不在** `legacy_host_tools.go` 的 7 组存档表内，
也未在 `.pair/plugins/tool-*` 任何插件声明——是前三轮漏网的未处理项。

| 文件 | 工具 | 实测证据 | 建议 |
|---|---|---|---|
| `internal/agent/findfiles.go`（152L） | `find_files_by_pattern` | `registerFindFilesByPatternTool` 定义于 :20，全仓生产调用 = 0（仅 `live_e2e_test.go:105` 历史消息 fixture 提及）；不在 builtinPluginSpecs / legacyToolGroups / embeddedToolRegistrars / tool_plugin_gen 白名单；`.pair/plugins` grep 零声明。Round3 已用 `glob`（registerCoreTools）替代其语义（tools_test.go:205 注释明示） | **删除**（功能已被 glob 完整继承，与 Round3 死包处置同标准）；或至少按 T1 模式存档（`ArchiveHostTool`），由 tool-core 插件 hostTool 承载——建议直接删除，glob 已覆盖 |
| `internal/agent/projectindex.go`（198L） | `project_index` | `registerProjectIndexTool` 定义于 :212，全仓生产调用 = 0（仅文件内自测引用）；同样不在任何装配表/插件声明 | **删除**（无消费者、无插件引用；project-info 知识库的 `project_info_*` 已由 tool-project-info 插件承载，路径索引语义被 `glob` + `project_info_explore` 覆盖） |

**处置动作**（t2 inScope）：
1. `git rm internal/agent/findfiles.go internal/agent/projectindex.go`（删除前再跑一次 grep 复核零引用，纯测试 fixture 一并随文件删除——live_e2e_test 中该工具名属于「历史消息 fixture」语义，工具调用不会真执行，可保留字符串不动）；
2. `tool_landing_test.go` 若含这两个名字的断言（当前无），随删除清理；
3. docs 同步：本文件 §1 + plugin-gaps.md「未实施项」追加。

**验收**：`go build ./...` + `go vet ./...` 通过；grep `registerFindFilesByPatternTool|registerProjectIndexTool|find_files_by_pattern|project_index` 生产代码零残留。

---

## 2. A2 死代码包（零生产调用，仅测试自证）→ **本轮处置：与 Round3 ① 同标准评估删除**

| 文件 | 内容 | 实测证据 | 建议 |
|---|---|---|---|
| `internal/agent/evalstore.go`（459L） | 评分记录持久化（`GetEvalStore`） | `GetEvalStore` 生产调用 = 0（仅 evalstore_test.go）；无 HTTP 路由、无工具注册引用 | **待产品确认**：评分系统（LLM-as-Judge 存档）在外部实现是 bench 子系统，pair 生产无装配点。倾向删除（与 jobs/permission/provider/vterm 同型：零导入零装配） |
| `internal/agent/evaluator.go`（129L） | LLM-as-Judge 评测 Agent（`Evaluator.Evaluate`，复刻参考 bench/evaluator.ts） | `NewEvaluator`/`Evaluator` 生产调用 = 0（仅 evaluator_test.go）；`DefaultJudgePrompt` 被 role_prompts.go 磁盘优先 loader 引用（保留 loader 即可） | 同上。若保留，**至少**把 `DefaultJudgePrompt` 之外的主体标记「bench 归档、生产零装配」（与 Go 旧名工具「测试/归档专用」同款注释纪律） |

**注意**：`DefaultJudgePrompt`（evaluator.go:41）被 `role_prompts.go` 的磁盘优先 loader 引用（plugin-round3 C1 闭环），删除 evaluator.go 时需把 `judgeSystemPrompt`/`DefaultJudgePrompt` 常量迁移到 role_prompts.go，再删 `Evaluator` 本体。t2 需先 grep 复核。

---

## 3. C 类：已插件化组的 Go 实现库（存档/测试基座，**保留**，标注双轨漂移风险）

以下 Go 工具实现已全部迁出宿主直接注册面：宿主只注册框架协议工具
（`RegisterHostFrameworkTools`，builtin_plugins.go:105-110），agent 可见工具面由 `.pair/plugins/tool-*`
JS 插件（28 个）注册；Go 实现经三条路径保留为「宿主能力库」：
- `claimTool` 自动存档（plugin.go:256-274，插件同名接管时 `ArchiveHostTool`）；
- `legacy_host_tools.go` 显式存档 7 组孤儿工具（asset/bridge/entryconfig/evolution/progress/resource/snapshot）；
- `embedded_tools.go` 内嵌内核注册表（`InitEmbeddedToolRegistry`，jsplugin.go:654 ctx.binary 回退）；
- `builtinPluginSpecs`（builtin_plugins.go:37-76）+ `RegisterToolGroups` 仅作 plugins-src 独立二进制
  （已废弃构建）与测试基座。

| Go 实现文件 | 对应插件 | 现状 |
|---|---|---|
| tools.go（registerCoreTools：read/write/edit/multi_edit/bash/move_file/delete_file/glob/grep + codegraph 注入） | tool-core / tool-harness | 测试/归档基座（Round3 改名后新名）；宿主不注册 |
| search.go、harness_tools.go（RegisterHarnessTools/run_code） | tool-harness | 同上（含 goja 嵌套工具调度） |
| git.go、web.go、shell.go（globalBG 服务）、memory.go、verify_tools.go | tool-git / tool-web / tool-shell / tool-memory / tool-verify | 同上 |
| officetools.go（1890L）、debug_tools.go（1425L）、screenshot_tool.go(+other)、webdebug.go、bugdetect.go+bugfix.go、codegraph_tools.go+codegraph_extra.go、binary.go+binary_re.go、projectinfo.go | tool-office / tool-debug / tool-screenshot / tool-web-debug / tool-bug / tool-codegraph(+extra) / tool-binary / tool-project-info | 同上；其中 office/debug/binary/codegraph/screenshot/web-debug 同时在 embedded_tool_registry（ctx.binary 回退） |
| asset_manager.go+asset_store.go+asset_tools.go、evolution.go+evolution_tools.go、bridge_controller.go+bridge_tools.go、entryconfig.go、progress_checker.go、resource_manager.go+resource_tools.go、snapshot.go（RegisterSnapshotTools） | tool-asset / tool-evolution / tool-bridge / tool-entryconfig / tool-progress / tool-resource / tool-snapshot | legacy_host_tools.go 存档（T1 闭环） |
| task_tools.go（update_tasks）、plan.go（update_plan）、tool_stats.go、management_tools.go（history_*） | tool-system（hostTool 承载） | 宿主框架协议注册（RegisterHostFrameworkTools），保留 |

**风险标注（非阻塞，记录不修）**：Go 实现与 JS 插件实现并存 = 双轨漂移风险（行为/描述不一致）。
已缓解机制：`tools_concise.go`（描述精简宿主表）、`tool_plugin_gen.go`（生成器白名单对齐）、
`TestToolLandingMatrix`（295 工具零未落地断言）、`TestNoOldNameRegistration`（旧名零注册断言）。
**本轮不扩大迁移**（JS 化已完成，Go 侧作为能力库是刻意设计，见 go-core-capabilities.md §3.2）。

---

## 4. A4 MCP 工具面无插件归属 → **记录 + 可选封装（不阻塞）**

- `RegisterMCPServers`（mcp.go:279）在**生产路径直接注册** `mcp__<server>__*` 工具到会话 Registry：
  - cmd/companion/web_server.go:2091（buildWebLoopOpts 内）
  - internal/desktopbridge/desktopbridge.go:505（桌面端）
- 无对应 tool-mcp 插件；`ctx.mcp` 服务（jsplugin.go:1512）只提供 MCP 配置管理面（mcp_list/save），
  工具面注册绕过插件宿主（不经 claimTool → 无插件归属 → 插件面板不可见、不可按插件启停）。
- **判断**：MCP 是宿主协议层（go-sdk stdio 连接），协议连接应留宿主；但「工具面挂载」可插件化——
  建议**记录为 P2**（封装 tool-mcp 插件：插件声明工具、execute 走 `ctx.hostTool.exec` 复用
  `RegisterMCPServers` 注册的 Go 执行器，或暴露 `ctx.mcpTools` 服务面）。**本轮不动**（行为稳定、
  风险低，且 MCP 工具注册数与配置相关，封装需定义执行器存档路径）。

---

## 5. B 类：应保留的宿主基础设施（完整清单 + 分类依据）

### B1 循环 / 会话 / 存储核心（不可插件化：宿主编排本质）
`loop.go`（1545L，TAOR 主循环+快照同步+审批门）、`agentloop.go`（双层循环状态模型）、
`agent.go`（AgentBase）、`session_manager.go`（1437L，会话生命周期+自动续轮）、
`session_context.go`（1025L，上下文构建）、`message_store.go`（1806L，JSONL 持久化）、
`store.go`（存储接口）、`build_history.go`、`compress.go`（上下文压缩）、`history_condense.go`、
`storm_breaker.go`、`circling.go`（绕圈检测）、`loop_factory.go`、`loop_parallel.go`、
`loop_approve_state.go`、`loop_hooks.go`（L2 闭环）、`loop_service.go`（ctx.get('loop') 服务面）、
`jsloop_registry.go/jsloop_run.go/jsloop_runner.go`（JS loop 插件运行时——**机制在宿主**，与 goal 同构）。
依据：这些是「一次 Run=一个 turn」的状态机与消息流，插件化会破坏会话一致性（Round3 §3 已定案机制留宿主）。

### B2 LLM / Provider
`llm.go`（546L，HTTP 客户端）、`provider_factory.go`（配置装配点）、`provider_impls.go`（S1 实现槽位，
**已插件化**：`ctx.provider.register`）、`llm_analyze.go`（工具集 LLM 意图分析）、`embedding*.go`+`bpetokenizer.go`（向量/分词，onnx 内核）。

### B3 插件宿主 / 运行时（本轮 ② 的主战场）
`plugin.go`（1303L，PluginHost/claimTool/生命周期）、`plugin_tools.go`（1008L，cordis 工具面）、
`jsplugin.go`（3888L，goja 沙箱+ctx 服务面）、`jsplugin_agents.go`/`jsplugin_jsloop.go`/
`jsplugin_loopfactory.go`/`jsplugin_providerfactory.go`/`jsplugin_providerimpl.go`、
`tscompile.go`（esbuild 编译）、`builtin_stdlib.go`（path/events/util 内联库）、
`node_bridge.go`（638L，Node 桥进程管理）、`node_plugins.go`（npm 插件安装路径）、
`npm_plugin.go`（npm 市场）、`embedded_tools.go`、`host_executors.go`、`legacy_host_tools.go`、
`builtin_plugins.go`、`tool_plugin_gen.go`（生成器）、`runtime_assets.go`、`assets/cordis.bundle.js`。

### B4 工具集 / 工具面框架
`toolset.go`（1549L，含 LoadGlobalPlugins/applyGlobalPluginDir 磁盘插件装配）、`toolset_tools.go`、
`toolset_templates.go`、`builtin_toolset.go`、`tools.go`（Registry 执行器+核心工具库）、
`tools_staging.go`、`tools_concise.go`、`typed_tool.go`、`tool_error.go`、`toolconfig.go`、
`harness_filter.go`、`harness_tools.go`、`glob.go`、`edit_matcher.go`、`encoding_detect.go`。

### B5 会话安全 / 快照 / 回滚
`snapshot.go`（快照系统，RegisterSnapshotTools 已存档）、`rollback.go`（240L 回滚引擎，HTTP 面 /api/chat/rollback）。

### B6 规划 / 任务 / 状态 / 统计
`plan.go`+`plan_manager.go`（update_plan 框架工具）、`task_manager.go`+`task_tools.go`（update_tasks 框架工具）、
`execution_plan.go`、`exec_state.go`、`execution_log.go`、`token_stats.go`（/api/tokens/stats）、
`tool_stats.go`（tool_stats 框架工具）、`cache_shape.go`、`build_history.go`。

### B7 能力管理（Skills / MCP / 记忆 / 资源）
`skill_loader.go`（502L，UI 与 agent 共用）、`mcp.go`+`mcp_config.go`（协议层 + 配置）、
`market_source.go`（市场源注册表，插件化扩展点）、`resource_manager.go`+`resource_tools.go`（存档）、
`management_tools.go`（history_search/list/count 框架工具）、`memory.go`（记忆库，工具已插件化）。

### B8 多 Agent 编排（机制留宿主，工具面已插件化）
`subagent_registry.go`（529L，可续聊子 Agent 注册表）、`subagent_sink.go`、`session_bridge.go`（ask_user 会话桥，已存档执行器）、
`planner.go`/`reviewer.go`（多 Agent 角色引擎，loop.go:512 生产调用）、`goal.go`（Round3 状态机）、
`workflow.go`（Round3 goja 运行器）、`commands.go`（Round3 命令注册表）、`autonomous_controller.go`（RegisterPlanOnlyTools，session_manager.go:650 生产调用）。

### B9 传输 / 路由层
`event_ws.go`、`wsconn.go`、`terminal_ws.go`（PTY WebSocket）、`ext_routes.go`/`ext_sse.go`/`ext_ws.go`（插件扩展路由）、
`kernel_api.go`（~90 条内核路由表）。

### B10 HTTP 服务层（cmd/companion + internal/server）
`cmd/companion/main.go`、`web_server.go`（2566L：startWebUI + 内核表 handler 实现）、`web_mode.go`、`webui_webonly.go`、
`kernel_register.go`（内核表注册）、`subagent_spawn.go`（成员会话启动器注入）、`log_file.go`、`event_ring.go`；
`internal/server/handler/` 10 文件（git.go/health_fs.go/plugins.go/register.go/stub.go/tools.go/toolsets.go/
uiassembly.go/workspace.go/handler.go——web/desktop 共享 handler 实现，Round2 S2 已删除未注册死 handler 1155 行）。
依据：HTTP 传输层是宿主 mux 基础设施，内核路由表挂载权已在 core-api 插件（go-core-capabilities.md §2）。

### B11 提示词 / 角色
`prompt_registry.go`（系统提示组装，对齐 外部 system-prompt）、`role_prompts.go`（磁盘优先角色 loader，C1 闭环）、
`selfmanagement_prompt.go`。

### B12 图像 / 编码 / 其他工具库
`image_submit.go`（submit_image 协议解析，loop.go 生产调用）、`projectenv.go`、`multiproject.go`、
`parallel_tools.go`（4L 废弃声明）、`binary.go`/`binary_re.go` 等（并入 C 类）。

---

## 6. 处置建议汇总（按优先级）

| 优先级 | 项 | 处置 | 工作量 |
|---|---|---|---|
| P0（本轮） | A1：findfiles.go / projectindex.go 孤儿工具 | 删除（glob 已覆盖语义） | S |
| P1（本轮） | A2：evalstore.go / evaluator.go 死代码 | 评估删除；保留则标注归档（DefaultJudgePrompt 常量迁 role_prompts.go） | S |
| P2（记录） | A4：MCP 工具面插件封装 | 可选 tool-mcp 插件（hostTool 存档），本轮不动 | M |
| 保留 | C 类 20 组 Go 工具库 | 维持现状（测试/归档基座 + 宿主能力库），双轨漂移风险已缓解 | — |
| 保留 | B 类 ~150 文件 | 宿主基础设施，不迁移 | — |

**验证基线**（t3 沿用）：`go build ./...`、`go vet ./...`、`go test ./internal/agent/ -count=1 -timeout 10m`
（环境依赖清单沿用 round3 §10.1：msys bash 信号管道组 + rod 下载）；`TestToolLandingMatrix` 通过；
grep 断言（§1 验收）零残留。

---

## 7. 证据索引（文件级）

- 装配面：`builtin_plugins.go:37-113`（宿主仅 RegisterHostFrameworkTools）、`legacy_host_tools.go:23-63`
  （7 组存档表，无 findfiles/projectindex）、`embedded_tools.go:21-31`（内嵌注册表 9 组，无 findfiles/projectindex）、
  `tool_plugin_gen.go:53-63`（生成器白名单，无 findfiles/projectindex）、`plugin.go:256-274`（claimTool 存档）、
  `host_executors.go:36-78`（执行器索引）。
- 孤儿证据：`findfiles.go:20`、`projectindex.go:212` 定义处 + 全仓 grep 生产调用 0；
  `tools_test.go:205`（find_files_by_pattern → glob 注释）；`.pair/plugins` grep 零声明。
- 死代码证据：`evalstore.go:73`、`evaluator.go:64` + grep 生产调用 0；`role_prompts.go`（Default*Prompt 引用）。
- MCP 证据：`mcp.go:279`、`web_server.go:2091`、`desktopbridge.go:505`、`jsplugin.go:1512`（ctx.mcp 仅配置面）。
- 双轨证据：`.pair/plugins/tool-*` 28 个 JS 插件 + 对应 Go 实现（§3 表）。
- JS 运行时缺口（② 配套）：`jsplugin.go:2738-2754`（Node API trap）、`builtin_stdlib.go:20-205`（仅 path/events/util）、
  `node_bridge.go:42-670`（桥协议）、`bridge_node.js:225-267`（@cordisjs/core cordis3 装载）、
  `node_plugins.go:67-84`（nodePluginNeedsNode 路由判定）。
