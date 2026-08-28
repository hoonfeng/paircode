# Round2 插件化专项：三合一分析报告与执行计划（t1 · 2026-09）

> 本文件为 **plugin-round2-alignment** 团队 t1（requirements）分析交付物。
> 三合一只读分析：① 上轮遗留项处置建议；② repo 插件与 DSH 参考差异对照；
> ③ 全插件工具重复矩阵。除本文件外未改动任何源码/插件/配置。
>
> DSH 参考基线：
> - deepseek-harness 核心工具：`C:\Users\sqmen\AppData\Roaming\npm\node_modules\@deepseek-ai\dsh\node_modules\@deepseek-ai\dsh-tool-*`
> - dsh-agent-teams：`C:\Users\sqmen\.dsh\profiles\web\node_modules\@nanmicoder\dsh-agent-teams`（v0.1.14）

---

## 0. 现状基线（本轮实测）

- 插件包总数：**43 个**（`.pair/plugins/` 下含 package.json 的目录；任务书说 41，实测 43，
  差异：host-capability-probe / tool-asset / tool-bridge / tool-entryconfig / tool-evolution /
  tool-progress / tool-resource / tool-snapshot 等为 t1 新增与既有合计）。
- 磁盘插件工具注册：25 个 tool-* 插件 + agent-teams（13 工具）+ marketplace（2 工具）
  + agentloop（0 工具，装配器）+ host-capability-probe（0 工具，webServer 探测）。
- 宿主框架工具（RegisterHostFrameworkTools）：history_search/list/count、update_plan、
  tool_stats、update_tasks（tool-system 插件 hostTool 承载）。
- 工具集：`.pair/toolsets/default.json`（git-ignored 运行时数据）声明 agent 可见白名单，
  内含 **13 个 tool-* 插件的内嵌 code 拷贝** + builtin 条目（system/plugin-mgmt/toolset-mgmt/shell/web）。
- 工作树 git 完全干净（master @ 94063c6f）。

### 实测运行结论（go test，GOCACHE 重定向沙箱）

```text
TestToolGitJSNative        FAIL  git_status  → ctx.binary.exec: 插件二进制不存在 tool-git.exe
TestToolMemoryJSNative     FAIL  memory_write → ctx.binary.exec: 插件二进制不存在 tool-memory.exe
TestToolProjectInfoJSNative FAIL project_info_write → 二进制不存在 tool-project-info.exe
TestToolVerifyJSNative     FAIL  project_info_verify → 二进制不存在 tool-verify.exe
TestToolBugJSNative        FAIL  bug_analyze  → ctx.binary.exec: 插件二进制不存在 tool-bug.exe
TestToolLandingSpotCheck   FAIL  tool-project-info/project_info_list → 二进制不存在
TestToolOfficeJSNative     FAIL  csv_read 断言（内嵌内核可执行，仅 Markdown 表格空格断言陈旧）
TestToolBinaryJSNative     通过（内嵌内核承接）
TestToolLandingMatrix      通过（但模式表与实际 JS 漂移，见 §1.2 关键发现）
```

---

## 1. 遗留项处置建议

### 1.1 关键发现（先于清单的横切事实）

1. **「3 个 binary 插件」实为 5 个**：任务书/docs 记为 tool-project-info/tool-verify/tool-memory，
   实测 **tool-git、tool-memory、tool-project-info、tool-verify、tool-bug 共 5 个插件仍是
   binary 形态**（JS `execute: (args) => ctx.binary.exec(t.name, args)`，见各 index.js），
   且 `embedded_tools.go` 内嵌内核**未覆盖** git/memory/project-info/verify/bug 五组
   （`embeddedToolRegistrars` 只有 binary/codegraph/codegraph-extra/screenshot/web-debug/
   harness/debug/office 九组）。→ 处置范围应为 **5 个**，建议 t2 一并 JS 化。
2. **tool_landing_test.go 模式表与实际 JS 漂移**：`toolPluginModes` 把 tool-git/memory/
   project-info/verify/bug/web/vison 标为 `native`，但 5 个插件实际仍是 binary 形态——
   模式表是「计划态」而非「现状态」，`TestToolLandingMatrix` 因 native 分支无断言而
   **假绿**。JS 化后需同步模式表（或改为运行时探测）。
3. **office 的 csv_read 断言陈旧**：内嵌内核已承接 office（`registerOfficeTools` 在
   embeddedToolRegistrars 内），实测可执行，仅测试断言期望 `| name |` 而内核输出
   `| name   |`（Markdown 填充空格差异）。属测试断言修正，非插件问题。
4. **toolset 内嵌 code 与磁盘插件双份**：default.json 内嵌 13 个插件 code，与
   `.pair/plugins/` 磁盘包并存。实测 10 个字节级一致、**3 个不一致**（tool-harness
   16.6KB vs 17.0KB、tool-core 9.0KB vs 9.3KB、agent-teams 101KB vs 100KB 且 1882 行差异）。
   装载时序：installToolset（内嵌）→ LoadGlobalPlugins（磁盘，同名卸载重定义）——
   磁盘版最终生效，内嵌 code 是**冗余陈旧拷贝**（agent-teams 差异最大）。

### 1.2 11 个 internal 死包处置（三选一 + 依据）

| 包 | 现状（实测） | 处置 | 依据 |
|---|---|---|---|
| internal/jobs | jobs.go 9.4KB + test；生产零导入；shell.go globalBG 已承接后台进程 | **删除** | 功能已被 internal/agent/shell.go 的 globalBG 取代（tool-shell 插件 ctx.process）；无插件/内核引用（实测 refs=0） |
| internal/permission | permission.go 9.9KB + test；生产零导入；审核门已由 requiresApproval 元数据 + loop 审批实现 | **删除** | 功能被 agent 层审核机制取代；无引用 |
| internal/provider | provider.go/dotenv/effort/retry/schema 共 5 源 3 测试；生产零导入；agent 已有 ProviderParams/OpenAIProvider | **删除** | 被 internal/agent/provider.go + provider_impls.go 取代；无引用（注意：删除前确认无 build tag 引用，实测无） |
| internal/vterm | vterm.go 11.9KB + test；生产零导入；PTY 渲染已由 wb-ui 前端承担 | **删除** | 屏幕模型属历史终端方案，wb-ui 已接管；无引用 |
| internal/agenttools | tools.go 610B 单行转发壳（windows tag）；零导入 | **删除** | 薄壳转发到 agent.RegisterManagementTools，无调用方；转发目标自身即宿主框架 |
| internal/codetypes | codetypes.go 847B（CodeLoc 等）；零导入 | **删除** | 无消费者；若未来 UI 框架需要可随时重建（3 个 struct 的规模） |
| internal/event | **空目录**（0 文件） | **删除** | 空壳无内容 |
| internal/model | **空目录**（0 文件） | **删除** | 空壳无内容 |
| internal/store | **空目录**（0 文件） | **删除** | 空壳无内容 |
| internal/summary | **空目录**（0 文件） | **删除** | 空壳无内容 |
| internal/verify | **空目录**（0 文件） | **删除** | 空壳无内容（注意与 plugins-src tool-verify 无关） |

> 共性依据：11 包生产导入全为 0（实测 `paircode/internal/<pkg>` 引用扫描）；
> event/model/store/summary/verify 为空目录。`internal/hook` 已在 t2 接线（loop_hooks.go，
> 不属死包）；`internal/bridge/pty/ui/uiapi/server/core` 有生产引用（bridge×2、pty×3、
> ui×2、uiapi×1、server×10、core×26），不在删除清单。
> ⚠️ 删除属「产品确认」类：建议 t2 先删 5 个空目录 + agenttools/codetypes（零风险），
> jobs/permission/provider/vterm 需产品确认后删（有实现有测试，保留可作参考移植源）。

### 1.3 5 个 binary 形态插件 JS 化方案（对齐 tool-core 模式）

**范围修正**：任务书 3 个 → 实测 5 个（tool-git/tool-memory/tool-project-info/tool-verify/tool-bug）。

| 插件 | 工具数 | 现有 Go 实现 | JS 原生迁移工作量 | 备选：内嵌内核补齐 |
|---|---|---|---|---|
| tool-project-info | 7 | projectinfo.go 248 行起 | 中（write/read/list/tree/search/delete/explore，ctx.fs 直读 .pair/project-info） | 低（embeddedToolRegistrars 加 registerProjectInfoTools 一行） |
| tool-verify | 2 | verify_tools.go | 低（memory_verify/project_info_verify，扫描引用有效性） | 低 |
| tool-memory | 5 | memory.go 240 行起 | 中（write/delete/read/list/search，ctx.fs 读写 .pair/memory） | 低 |
| tool-git | 10 | git.go | **高**（10 个 git CLI 编排，建议 ctx.process 或 ctx.bash 封装，或保留 hostTool 存档） | 低（加 registerGitTools） |
| tool-bug | 3 | bugfix.go | 中（bug_detect 需 go vet/build/test 编排，可用 ctx.bash） | 低（加 RegisterBugTools） |

**建议**：t2 采用「**JS 原生迁移为主、内嵌内核补齐兜底**」组合：
- tool-project-info / tool-verify / tool-memory：JS 原生（对齐 tool-core，ctx.fs 实现），
  工作量可控且无外部进程依赖；
- tool-git / tool-bug：优先 JS 原生（git 用 ctx.process/bash 封装 10 工具；bug 用 ctx.bash 编排
  vet/build/test）；若 t2 时间不足，**最低成本方案 = embeddedToolRegistrars 补 5 组注册函数**
  （Go 实现全在，每行一个注册函数），JS 无需改动，立即消除「二进制不存在」故障。
- 同步修复：tool_landing_test.go 模式表改为实际态；office csv_read 断言空格修正。

### 1.4 P2 清理项 now/defer 分类

| ID | 项 | 分类 | 理由/动作 |
|---|---|---|---|
| T3 | tools_concise.go 文言文精简硬编码 | **defer（保留宿主框架能力）** | ApplyConciseToolDescriptions 属 harness 对齐框架行为（t1 已认可）；但需注意与插件 description 双份漂移风险（同工具两处描述），建议后续统一为「插件描述 + 精简映射表配置化」 |
| S2 | web_server.go 遗留 ~25 个未注册 handler | **now（t2 低风险）** | 实测 handler 注册仅 2 处（/plugins-assets/、/），handleFS*/handleGit* 等为未注册死实现（gap 分析 L171-173 有清单）；git-api/fs-api 插件已接管。删除前需 grep 引用（见 §3 影响面） |
| F2/F3 | 前端死组件（OutputPanel/SubAgentBlock/TasksPanel/router.js）+ 前端产物堆积 | **defer** | 属前端专项（超出 t2 inScope）；需 plugins-src/ui-app 专项清理 |
| L4(旧) | 审批遗留字段 | **defer** | 纯兼容死代码，无行为影响 |
| G2/G3/G4 | 配置模板漂移/APIKey 明文/无校验 | **defer（G3 部分 now）** | packager stripSecrets 已发布侧脱敏；G3 APIKey 明文建议 t2 顺手在设置面板做掩码展示（低风险）；G2/G4 需产品决策 |
| L3(旧) | AgentBase 未用 + 初始化三份 | **defer** | 架构级统一，风险高，超出本轮 |

### 1.5 t4 低危 findings（L1–L5）处置

| ID | 现状（实测） | 处置 |
|---|---|---|
| L1 bridge_release→bridge_lockdown 描述漂移 | **仍存在**：legacy_host_tools.go:30 描述「bridge_status/takeover/release/exec/…」，tool-bridge/index.js 头注释/purpose 亦写 bridge_release；实际工具是 bridge_lockdown（bridge_tools.go:71） | **now**：3 处字符串改 bridge_lockdown（legacy_host_tools.go:30、tool-bridge/index.js 头注释+purpose、docs/plugin-gaps.md:30） |
| L2 钩子 payload.turn 恒 0 | **仍存在**：loop_hooks.go:205/238 两处 hookPayloadOf 传 0 | **now**：从 Loop 状态透传真实轮次（openTurn 后计数），或文档化未接线；成本低 |
| L3 G1 注释与实现不一致 | **部分修复**：toolconfig.go 头注释已改为「settings 同字段为空/默认时采用」，迁移实现（76-87）仍无条件以遗留覆盖 settings 白/黑名单 | **now**：统一注释与实际覆盖语义（黑/白名单仅在 settings 为空时迁移）或按注释改实现；同时记录「per-workspace→全局」语义变化 |
| L4 RegisterProviderImpl 还原顺序敏感 | **仍存在**：provider_impls.go:33 同名二次注册后，先注册者 restore() 覆盖后注册者 | **defer/文档化**：单插件场景无影响；改为栈式还原成本中等，建议文档化 + 测试标注 |
| L5 config/philosophy 残留 | **仍存在**：config/philosophy/ 10 个 txt（含 debugger/explorer/general 等）仍在开发树 | **now**：删除目录（已从 packager 移除，零消费者；删除前 grep 确认） |

---

## 2. DSH 参考对照

### 2.1 agent-teams 插件 vs 参考 @nanmicoder/dsh-agent-teams v0.1.14

**工具面：13 工具名完全一致**（create/edit_plan/approve/add_member/remove_member/create_task/
reassign_task/claim_task/update_task/send_message/status/resume/delete）——已对齐，无需改名。

**实现差异清单（应同步项）**：

| 能力 | 参考 v0.1.14 | repo 移植（.pair/plugins/agent-teams） | 应同步 |
|---|---|---|---|
| 两阶段计划 | staged 全流程（planReviewState/awaiting_review/approved/discarded/returned） | ✅ 同（index.js L978、L1667 路由 approve/discard/continue/halt） | — |
| 质量门禁 | quality-gates.js：requirements/implementation/verification/review/repair/integration 契约 + verdict=pass 门 + repair/复审循环 | ✅ 同（TASK_KINDS/evaluateQualityCompletion/planQualityFollowUp/resolveReviewPolicy） | — |
| profiles（命名团队模板） | profiles.js：`profiles` 配置、`profile=` 参数、seed tasks、captain-planning、MAX_TEAM_PROFILES=16 | ❌ **缺失**（create 无 profile 参数、无 profiles 配置 schema） | **建议同步**（高价值：固定阵容一键建队） |
| memberProvider（spawn/fork） | 配置 `memberProvider: spawn\|fork`（cordis.patch.yml） | ❌ 仅 spawn（spawnMember 固定 ctx.agents.start） | 建议同步 fork 选项 |
| slash 命令 /agent-teams | command.js registerAgentTeamsCommand + gesture boundary | ❌ **缺失**（client.js 只有标题栏按钮，无宿主命令注册） | 建议同步（若宿主支持 ctx.commands） |
| 活动面板 | ActivityPanel/StagingPlanEditor/activity-model/panel-geometry（195KB client） | ⚠️ 简化版（client.js 19KB：3s 轮询 + 成员卡片 + 任务 DAG + staged 操作） | 按宿主 UI 能力选择性同步（面板几何/折叠态/模型标注） |
| 模型路由 | 原生 provider/model/reasoning 目录（plan editor 用 Harness 目录） | ⚠️ 快照队长当前路由（ctx.llm.current），add_member 支持 provider/model 参数 | 建议同步 reasoning_effort 透传 |
| 事件 | event-types.js/events.js（agent-teams/* 事件发射） | ❌ 无事件发射（仅 HTTP 轮询） | 视宿主事件总线能力 |
| snapshot | snapshot.js collectArchivedTeamsActivity | ⚠️ 仅 archiveTeamDir 归档 | 低优先 |
| 状态层 | <ws>/<stateDir>/<teamId>/ team.json + inbox/*.jsonl | ✅ 同 | — |
| 成员会话 | subagents 可续聊 + parked attempt + 冷恢复 + 退役黑名单 | ✅ 同（ctx.agents.start/followup + retired-members.json） | — |

**建议 t2 同步优先级**：① profiles（配置 schema + create profile 参数 + 提示词列出）；
② memberProvider fork；③ slash 命令（若宿主支持）；④ reasoning_effort 透传。
工具名/状态机/质量门禁已对齐，无需动。

### 2.2 harness 工具语义/命名差异（repo vs deepseek-harness 核心）

| 工具 | DSH 参考语义 | repo 现状 | 差异 | 对齐建议 |
|---|---|---|---|---|
| read | 参数 `file_path`；输出带行号（lines[{number,text}]+totalLines） | tool-harness `read`：参数 `path`；输出纯文本（无行号） | 参数名 + 行号输出 | **建议对齐**：加行号输出（前端/测试消费方见 §3）；参数名改 file_path 需评估兼容（大量调用方）→ 可加别名 |
| write | `file_path` + content；输出 {path,before,after} | `path` + content；文本输出 | 参数名 | 同上（别名策略） |
| edit | `file_path`/old_string/new_string/replace_all；输出 before/after | `path` + old/new + line_start/line_end；无 replace_all | replace_all 缺失 | **建议补 replace_all**（语义对齐）；参数名别名 |
| glob/grep | pattern + path 等（dsh-tool-fs-search） | tool-harness glob/grep（ctx.fs.glob/grep） | 基本对齐（pattern/path/language/case_insensitive） | 保持 |
| bash | command + description(必填,5-10词) + timeoutMs + workdir + run_in_background + sandbox_permissions | `bash`：command + cwd + project；无 description/timeoutMs 参数 | description 必填缺失、timeoutMs 参数缺失、无 background 开关（run_background 独立工具） | **建议**：bash 加 description 可选参数 + timeoutMs（对齐 UI 展示）；run_in_background 保持独立工具（repo 已有 run_background，不需合并） |
| pwsh | dsh-tool-pwsh（Windows 主 shell） | 无 pwsh 工具（bash 走 git-bash/ConPTY） | **缺失 pwsh** | 建议评估：Windows 宿主加 tool-pwsh 或文档化 bash 等价 |
| web_search/web_fetch | dsh-tool-web 同名 | tool-web 同名 | ✅ 一致 | 保持 |
| ask_user | dsh-tool-ask-user：`ask_user_question`（questions 数组+id/options） | tool-system `ask_user`（单问题字符串，hostTool 会话桥） | 命名 + 参数形态 | **建议**：ask_user 支持 questions 数组（多问题/选项），命名保持 ask_user（harness_aligned 已含）或加别名 ask_user_question |
| skill | dsh-tool-skill：单 `skill` 工具（按名加载 SKILL.md） | tool-system skill_list/load_skill/load_skill_resource/skill_write/skill_delete | 拆分更细 | 保持（repo 更完整）；若需严格对齐可加 skill 别名 |
| jobs | dsh-tool-jobs：job_output/job_list/job_kill（后台任务） | tool-shell run_background/read_output/kill_process | 命名不同、语义等价 | **建议**：加 job_output/job_list/job_kill 别名（对齐命名），保留原名（消费方兼容） |
| todo | dsh-tool-todo：todo_write（全量替换清单） | tool-system update_tasks（全量替换任务清单） | 命名不同、语义等价 | **建议**：加 todo_write 别名 |
| goal | create_goal/get_goal/update_goal（同会话长目标） | **缺失** | 无 goal 机制 | 见 §2.3 |
| run_code | dsh-code-runtime | tool-harness run_code（ctx.binary → 内嵌内核 registerRunCode） | ✅ 一致 | 保持 |
| str_replace_editor | dsh-tool-str-replace-editor | tool-harness 同名 | ✅ 一致 | 保持 |

### 2.3 harness 特有机制中 repo 尚未插件化的部分

| 机制 | 参考 | repo 现状 | 建议 |
|---|---|---|---|
| goal 工具（create_goal/get_goal/update_goal） | dsh-goal + dsh-tool-goal | **无**（Loop 无 goal 状态机） | **P1 建议**：新增 goal 插件（宿主 Loop 加 goal 字段 + 3 工具），或先文档化暂不做 |
| subagent 工具（subagent/subagent_control/report） | dsh-subagent + dsh-tool-subagent* | 仅 agent-teams 内部 ctx.agents 编排，无通用 subagent 工具 | **P2**：可做通用 subagent 工具（含 agent-teams 已用能力） |
| workflow 工具 | dsh-workflow + dsh-tool-workflow | **无**（agent-teams 是团队编排，非 workflow 引擎） | **P2**：如需 workflow 引擎再移植 |
| ralph 工具 | dsh-tool-ralph | **无** | **P3**：按需 |
| tool-cordis | dsh-tool-cordis（宿主命令注册） | repo 有 cordis_* 框架工具（内置） | 保持（宿主内建） |
| skill 资源披露 | dsh-skill-filesystem | tool-system skill_* 已覆盖 | 保持 |

---

## 3. 工具重复矩阵（43 插件 + 内核注册）

### 3.1 全量工具注册清单（按插件）

| 插件 | 工具（数量） | 执行模式（实测） |
|---|---|---|
| tool-harness | read/write/edit/glob/grep/bash/str_replace_editor/run_code（8） | JS 原生（ctx.fs/bash）+ run_code 内嵌内核 |
| tool-core | multi_edit/move_file/delete_file（3） | JS 原生 |
| tool-shell | run_background/read_output/kill_process（3） | JS 原生（ctx.process） |
| tool-system | skill_list/load_skill/load_skill_resource/skill_write/skill_delete/mcp_list/mcp_add/mcp_remove/history_search/history_list/history_count/update_plan/tool_stats/update_tasks/ask_user/task_create（16） | hostTool（宿主 Go 存档） |
| tool-web | web_fetch/web_search（2） | JS 原生 |
| tool-git | git_status/diff/log/show/blame/add/commit/branch/checkout/stash（10） | **binary（故障）** |
| tool-memory | memory_write/delete/read/list/search（5） | **binary（故障）** |
| tool-project-info | project_info_write/read/list/tree/search/delete/explore（7） | **binary（故障）** |
| tool-verify | memory_verify/project_info_verify（2） | **binary（故障）** |
| tool-bug | bug_analyze/detect/fix（3） | **binary（故障）** |
| tool-office | csv_read/csv_write/json_to_table/table_stats/text_report/word_read/word_write/read_xlsx/write_xlsx/read_pdf/markdown_to_html（11） | 内嵌内核（ok） |
| tool-debug | debug_inject_log/run_capture/analyze_output/parse_stack/cleanup_logs/watch/evaluate_session（7） | 内嵌内核（ok） |
| tool-binary | inspect_binary/write_binary/binary_strings/find/patch/info/hash/entropy（8） | 内嵌内核（ok） |
| tool-codegraph | codegraph_build/stats/file_structure/function/class/callers/callees/impact/search/git_history/entity_history/get_edit_context/find_related_tests/analyze_complexity/search_by_pattern/trace_call_chain/find_dead_code/module_architecture（18） | 内嵌内核（ok） |
| tool-codegraph-extra | find_entry_points/find_hot_paths/find_by_imports/get_detailed_symbol/find_dead_imports/search_by_error/index_markdown/search_docs/verify_design/pr_context/find_by_signature/semantic_search/explore（13） | 内嵌内核（ok） |
| tool-screenshot | screenshot_desktop/window/area（3） | 内嵌内核（ok） |
| tool-web-debug | web_debug（1） | 内嵌内核（ok） |
| tool-vision | submit_image（1） | JS 原生（ctx.fs.stat 校验 + 标记行） |
| tool-asset | asset_list/search/delete（3） | hostTool（存档） |
| tool-bridge | bridge_status/takeover/lockdown/exec/register_system_tool（5） | hostTool（存档） |
| tool-entryconfig | find_entry_points/find_config_files（2） | hostTool（存档） |
| tool-evolution | evolution_save_capsule/search_capsules/save_gene/status（4） | hostTool（存档） |
| tool-progress | progress_checker（1） | hostTool（存档） |
| tool-resource | resource_list/search/stats（3） | hostTool（存档） |
| tool-snapshot | restore_snapshot/list_snapshots（2） | hostTool（存档） |
| agent-teams | agent_teams_* ×13 | JS 原生（ctx.agents） |
| marketplace | marketplace_search/install（2） | JS 原生（ctx.web） |
| host-capability-probe | （0 工具） | webServer /api/ext/probe |
| agentloop | （0 工具，装配器） | loopFactory/providerFactory |
| core-api/fs-api/git-api/web-api/ui-* | （0 agent 工具） | HTTP 路由/UI 半 |

### 3.2 重复矩阵（同语义/同名单工具）

| 工具名 | 出现位置 | 功能重叠 | 保留版本（最优判定） | 动作 | 影响面（消费方） |
|---|---|---|---|---|---|
| run_background/read_output/kill_process | ① tool-shell 插件（JS 原生）；② default.json `builtin:shell` 条目（同名使能）；③ tools_concise.go 有描述 | ①③ 同一工具（builtin 条目仅 SetToolEnabled 引用已注册名，非重复注册） | **tool-shell 插件**（实现载体） | **保留 tool-shell；清理 default.json builtin:shell 冗余条目**（与 tool-shell 插件条目重复声明） | 消费方：default.json 工具集白名单、前端工具面板、提示词（run_background 等）；builtin:shell 移除后 tool-shell 插件条目仍在 → 白名单不变 |
| web_fetch/web_search | ① tool-web 插件（JS 原生）；② default.json `builtin:web` 条目；③ harness_aligned 清单 | 同②机制 | **tool-web 插件** | 同上：清理 builtin:web 冗余（保留 tool-web 磁盘插件，default.json 由 toolset 管理） | 消费方：前端 RightPanel（/^web_fetch/ 图标）、提示词 |
| read/write/edit/glob/grep/bash/run_code/str_replace_editor | ① tool-harness 插件；② default.json `builtin:system` 条目；③ 内嵌内核（run_code/str_replace_editor）；④ Go registerHarnessAliases（已归档） | 同一工具多来源使能/实现 | **tool-harness 插件**（实现）+ builtin:system 保留（白名单语义） | 保留；**移除 Go 侧 harnessAliases 死代码**（harness_tools.go registerHarnessAliases 无生产调用，实测 RegisterHarnessTools 仅 embeddedToolRegistrars 引用其 run_code/str_replace_editor 部分） | 消费方：内置工具集 builtin:system、提示词、前端（RightPanel read_file/run_command 图标为旧名——工具已改名 read/bash，前端展示壳需同步或保持旧名映射兼容） |
| read_file/write_file/edit_file/run_command/search_files/search_content | Go registerCoreTools/registerSearchTools（tools.go:401/搜索组） | 被 read/write/edit/bash/glob/grep 取代 | — | **删除 Go 旧名工具注册**（零生产注册点；tools.go registerCoreTools 仅 builtinPluginSpecs 引用，宿主不调用） | 消费方：config/roles/*.md 提示仍写 read_file/run_command（reviewer.md:21、verifier.md:18、explorer.md:18、planner.md:23）→ **需同步改提示为 read/bash**；前端 RightPanel.vue 图标映射（read_file/run_command 分支）→ 加 read/bash 分支或保留兼容 |
| history_search/list/count、update_plan/tool_stats/update_tasks | ① 宿主 RegisterHostFrameworkTools；② tool-system 插件 hostTool 声明 | 同一工具双入口（宿主注册 + 插件声明，插件 claimTool 时 ArchiveHostTool 存档） | 宿主注册为真、插件为编排壳 | 保留（设计如此：tool-system 是 SystemTool 外置壳） | 消费方：提示词、前端、工具集 builtin:system |
| ask_user | ① tool-system（hostTool）；② session_bridge 会话桥 | 同一 | 保留 | 保留 + 加 questions 数组能力（§2.2） | 前端 RightPanel（ask_user 图标/未答折叠）、提示词 |
| fs HTTP 面 | ① fs-api 插件 /api/fs/*；② web-api 插件 /api/ext/fs/{read,exists,list} | /api/ext/fs/read|exists|list 与 /api/fs/read|list 重复 | **fs-api**（完整 drives/list/read/write/rename/delete/mkdir/search/file-info/hex） | **删 web-api 的 /api/ext/fs/{read,exists,list} 三个路由**（web-api 保留 status/fetch/routes/async/echo 演示面） | 消费方：前端只用 /api/fs/image（实测 ImageViewer/RightPanel）；/api/ext/fs 无前端消费者 → 零影响 |
| git 面 | ① tool-git 插件（agent 工具）；② git-api 插件（HTTP 17 条 + UI 面板） | 不同面（agent 工具 vs HTTP API），非重复 | 两者都保留 | 无去重动作 | — |
| marketplace_search/install | marketplace 插件 | 唯一 | 保留 | — | — |
| toolset 内嵌 code vs 磁盘插件（13 个） | default.json 内嵌 + .pair/plugins 磁盘 | 同一插件双份 code | **磁盘插件**（装载时序最后生效） | **清理 default.json 内嵌 code**（13 个插件条目改 name-only 引用磁盘包，或由启动逻辑自动同步）；agent-teams 差异最大需重点处理 | 消费方：工具集白名单（条目保留）、toolset_export（内嵌变引用后导出含磁盘包说明）、工具面板 |

### 3.3 消费方影响检查汇总（删除/改名前置条件）

1. **提示词/角色**：config/roles/*.md（现为磁盘优先生效，C1）仍引用旧工具名 read_file/
   run_command/search_content/search_files（explorer/planner/reviewer/verifier 四文件）——
   若删 Go 旧名工具或改名，必须同步这 4 个 md（t2 顺带）。
2. **工具集**：default.json 的 builtin:shell/builtin:web 与插件条目重复（§3.2 ①②）；
   tool-core 条目的 DisabledTools（read_file/write_file/edit_file/run_command）是**陈旧引用**
   （tool-core 插件已只注册 multi_edit/move_file/delete_file）——清理时确认无副作用
   （SetToolEnabled 对不存在工具是 no-op，已实测安全）。
3. **前端**：RightPanel.vue 图标映射 read_file/run_command/web_fetch/git_status/
   screenshot_*/ask_user 等旧名分支——去重改名需同步前端映射（或保留旧名分支作兼容壳）。
4. **插件间调用**：tool-verify 依赖 memory/project-info 的磁盘结构（非工具调用）；
   marketplace 依赖 ctx.web；agent-teams 依赖 ctx.agents/llm——无跨插件工具名耦合。
5. **测试**：tool_landing_test.go 模式表（§1.1 ②）、diskplugin_jsnative_test.go（5 个 binary
   测试当前失败）、office csv_read 断言（§1.1 ③）、embedded_tools_test.go 覆盖清单。
6. **reviewBlacklist**：无默认黑名单引用旧名（loop.go 仅注释）；settings.json 用户数据不受影响。

---

## 4. Round2 执行计划（按优先级）

| # | 动作 | 类型 | t2 实施 | 风险 | 回滚 |
|---|---|---|---|---|---|
| R2-1 | 5 个 binary 插件 JS 化（project-info/verify/memory 原生；git/bug 原生或内嵌内核补齐） | 实施 | ✅ | 中（git 10 工具编排） | git/bug 走内嵌内核时零 JS 改动、可即时回退；原生迁移保留旧 index.js git 历史 |
| R2-2 | 修复 tool_landing_test.go 模式表 + office csv_read 断言 | 测试 | ✅ | 低 | 纯测试 |
| R2-3 | t4 L1/L2/L3/L5 修复（bridge 描述、turn 透传、G1 语义统一、philosophy 删除） | 实施 | ✅ | 低 | 小改动，git revert |
| R2-4 | P2 S2：web_server.go 未注册 handler 删除（先 grep 引用） | 清理 | ✅ | 低-中 | 删除前确认零注册点（实测仅 2 处注册） |
| R2-5 | 死包删除（5 空目录 + agenttools/codetypes 直接删；jobs/permission/provider/vterm 待产品确认） | 清理 | ⚠️ 部分（空目录+薄壳先删，其余待确认） | 低 | git revert |
| R2-6 | agent-teams 对齐：profiles + memberProvider(fork) + reasoning_effort 透传（+slash 若宿主支持） | 实施 | ✅（按宿主能力裁剪） | 中（profiles 状态机） | 配置默认关闭，无行为变化 |
| R2-7 | harness 语义对齐：read 行号输出、edit replace_all、bash description/timeoutMs、ask_user questions 数组、jobs/todo 别名 | 实施 | ✅（分批） | 中（read 输出形态变化影响前端/测试） | 别名方式（原名保留）零破坏 |
| R2-8 | 工具集去重：default.json builtin:shell/builtin:web 冗余条目 + 13 插件内嵌 code 清理（改引用磁盘包） | 清理 | ✅（需谨慎：default.json 是运行时数据，改装载逻辑而非手改文件） | 中（影响白名单装载） | 装载逻辑加开关 |
| R2-9 | Go 旧名工具删除（read_file 等）+ 提示词/前端同步 | 清理 | ✅（与 R2-7 联动） | 中 | 分步提交 |
| R2-10 | web-api /api/ext/fs/* 三路由删除 | 清理 | ✅ | 低 | 零消费者（实测） |
| R2-11 | goal/subagent/workflow 机制评估 | 需求 | ❌（t3+ 或文档化） | — | — |
| R2-12 | 前端死组件/产物（F2/F3）、配置模板（G2/G4）、审批遗留字段 | 清理 | ❌（defer） | — | — |

**依赖关系**：R2-1 → R2-2（模式表反映新态）；R2-7 ↔ R2-9（改名与旧名删除联动，消费方同步）；
R2-8 独立但需与 R2-6/2-7 顺序隔离（工具集装载逻辑改动后跑全量冒烟）。
**验证命令**：`go build ./...`、`go vet ./...`、`go test ./internal/agent/ -count=1 -timeout 10m`
（GOCACHE 重定向 `.verify-tmp/gocache`，CGO_ENABLED=1）；前端 `cd cmd/companion/web-ui && npm run build`（沙箱外）。

---

## 5. 证据索引（文件级）

- 死包：`internal/{jobs,permission,provider,vterm,agenttools,codetypes,event,model,store,summary,verify}/`（实测导入 0）
- binary 插件：`.pair/plugins/tool-{git,memory,project-info,verify,bug}/index.js`（execute→ctx.binary.exec）；
  `internal/agent/embedded_tools.go`（embeddedToolRegistrars 九组，缺五组）
- 模式表漂移：`internal/agent/tool_landing_test.go:34-62`（toolPluginModes）
- office 断言：`internal/agent/diskplugin_jsnative_test.go:216-221`（csv_read 空格）
- toolset 内嵌 vs 磁盘：`.pair/toolsets/default.json`（git-ignored）vs `.pair/plugins/*/index.js`
  （tool-harness/tool-core/agent-teams 字节不一致）
- 重复声明：default.json `builtin:shell`/`builtin:web` 与 tool-shell/tool-web 插件条目
- t4 低危：`internal/agent/legacy_host_tools.go:30`、`internal/agent/loop_hooks.go:205,238`、
  `internal/agent/toolconfig.go:41-87`、`internal/agent/provider_impls.go:33`、`config/philosophy/`
- 提示词旧名：`config/roles/{explorer,planner,reviewer,verifier}.md`
- 参考对照：`C:\Users\sqmen\.dsh\profiles\web\node_modules\@nanmicoder\dsh-agent-teams\`
  （package.json v0.1.14、lib/profiles.js、lib/quality-gates.js、README_ZH.md）
- harness 工具参考：`…\dsh\node_modules\@deepseek-ai\dsh-tool-{fs,fs-search,bash,web,skill,jobs,todo,goal,ask-user,str-replace-editor}\lib\index.js`
## 6. Round2 实施记录（t2 · 2026-09）

> 本节由 t2（implementation）回填：每项标注「已实施 / 未实施（原因）」，附验收点与验证证据。
> 验证基线：`go build ./...` ✅、`go vet ./...` ✅、`go test ./internal/agent/ -count=1 -timeout 10m`
> 环境依赖失败除外（见 §6.13）；插件装载冒烟 ✅（43 插件装载、工具集 17 条目、无重复注册冲突、无 FATAL）。

### 6.1 R2-1 5 个 binary 插件 JS 化 → **已实施（5/5，超出任务书 3 个的实测范围）**

| 插件 | 迁移方式 | 验收点 | 证据 |
|---|---|---|---|
| tool-memory（5 工具） | JS 原生（ctx.fs 读写 .pair/memory/，复刻 memory.go：frontmatter/MEMORY.md 索引/碎片化提醒/多项目路由） | `TestToolMemoryJSNative` 通过；不再调 ctx.binary | ✅ `go test -run TestToolMemoryJSNative` |
| tool-project-info（7 工具） | JS 原生（ctx.fs：树形路径分级/notes 前缀镜像/explore/渐进式披露） | `TestToolProjectInfoJSNative` 通过 | ✅ 同上 |
| tool-verify（2 工具） | JS 原生（引用正则提取 + 跨工作区根存在性校验，复刻 pkg/verify） | `TestToolVerifyJSNative` 通过 | ✅ 同上 |
| tool-git（10 工具） | JS 原生（ctx.process.exec argv 直连 git CLI，30s 超时；复刻 git.go 参数组装） | `TestToolGitJSNative` + `TestToolGitJSNativeMultiProject` 通过 | ✅ 同上 |
| tool-bug（3 工具） | JS 原生（bug_analyze 纯文本解析 build/test/run；bug_detect/fix 经 ctx.bash 编排 vet/build/test） | `TestToolBugJSNative` 通过 | ✅ 同上 |

- 全部插件 `inject` 声明（fs/bash/process），`execute` 不再依赖 `ctx.binary`；模式表 tool_landing_test.go 原「计划态 native」现为实际态（无需改表，表与实现一致）。
- `plugins-src/plugins/tool-*` Go 二进制源码保留作归档参考（不参与装载）。

### 6.2 R2-2 测试修复 → **已实施**

- `internal/agent/diskplugin_jsnative_test.go`：office csv_read 断言 `| name |` → 对列宽不敏感（`| name` + `---` + 数据行）；word_read 断言 `# 标题一` → `标题一`（内核 Heading1 渲染为纯文本）。
- tool_landing_test.go 模式表与实现核对一致（native×7 组现为真实态）；`TestToolLandingMatrix` 通过（243 工具，hostTool 135/binary 43/mixed 26/native 39，零未落地）。

### 6.3 R2-3 t4 L1/L2/L3/L5 修复 → **已实施（L4 按计划 defer/文档化）**

- **L1** bridge 描述漂移：`legacy_host_tools.go:30`、`tool_plugin_gen.go:89`、`.pair/plugins/tool-bridge/index.js`（头注释+purpose）、`docs/pluginization-gap-analysis.md:47` 四处 `bridge_release` → `bridge_lockdown`。
- **L2** 钩子 turn 恒 0：`loop_hooks.go` 新增 `loopHookCurrentTurn`（atomic），`hookPayloadOf` 在 turn<=0 时取当前活动 Loop 轮次；`agentloop.go` `openTurn/endTurn` 维护（openTurn 置 TurnNo、endTurn 清零）。`TestLoopHookConfigBlock` 等钩子测试通过。
- **L3** G1 注释与实现不一致：`toolconfig.go` 迁移合并语义统一为「settings 同字段为空/默认时采用遗留值」（reviewMode 非默认 auto、列表为空才迁移），文件头注释与实现一致；`TestReviewConfigSingleSourceMigration` 通过。
- **L4** RegisterProviderImpl 还原顺序敏感：**未实施（按 t1 建议 defer/文档化）**——单插件场景无影响；栈式还原成本中等，记为遗留（§7.2）。
- **L5** config/philosophy 残留：目录已删除（10 个 txt，grep 确认零消费者：Go/packager 零引用；仅 docs/dev probes 提及历史）。

### 6.4 R2-4 P2 S2：web_server.go 未注册 handler 删除 → **已实施**

- `cmd/companion/web_server.go` 删除 **1155 行**死代码：FS 死 handler（handleFSDrives/List/Read/FileInfo/Hex/Write/Rename/Delete/Mkdir/Search）+ 整个 Git API 区（withGitDir/runGitInternal/gitStatusResult + handleGitStatus/Init/Diff/Add/Reset/Commit/Log/Branch/Checkout/Stash/StashList/Ignore/Discard/Push/Pull/Remote 16 条）。
- 前置 grep：全部引用仅限本文件（零外部引用）；fs-api 插件 10 条 `/api/fs/*` + git-api 插件 16 条 `/api/git/*` 已接管；`/api/fs/image` 保留内核（kernel_register.go 注册）。
- 修复删除导致的未使用导入（bufio/strconv）。`go build ./...` 通过；冒烟 /api/health 200。

### 6.5 R2-5 死包删除 → **已实施（部分：7/11，余 4 待产品确认）**

- 已删：`internal/event|model|store|summary|verify`（空目录）+ `internal/agenttools`（610B 薄壳）+ `internal/codetypes`（847B，零导入）——删除前 grep 确认零导入（仅 scripts/ 迁移工具的字符串映射提及）。
- 保留（t1 判定「需产品确认」，有实现有测试可作参考移植源）：`internal/jobs|permission|provider|vterm`——记录于 §7.2。

### 6.6 R2-6 agent-teams 对齐（profiles + reasoning_effort + fork 评估）→ **已实施（裁剪版）**

- **profiles**（对齐参考 lib/profiles.js）：`.pair/plugins/agent-teams/index.js` 新增 `TEAM_PROFILES`（默认 16 上限 MAX_TEAM_PROFILES）+ `create(profile=…)` 参数（提示词列出可用模板：default/captain-planning）——profile 一键建队（成员阵容 + seed 任务，队长可 edit_plan 调整）。内置 default（四人研发质量流水线：analyst/engineer/verifier/reviewer + 4 条 seed 任务）与 captain-planning（仅阵容）。
- **reasoning_effort 透传**：`SubAgentSpec.ReasoningEffort`（subagent_registry.go）→ `ctx.agents.start` 参数映射（jsplugin_agents.go）→ `buildSubAgentProvider` 仅档位时也构造 Provider（沿用默认路由、覆盖 ThinkingMode，经 applyThinking 下发 reasoning_effort）；插件侧 add_member/edit_plan/spawnMember/快照全链路透传成员档位。
- **memberProvider fork**：**未实施（按宿主能力裁剪）**——宿主 ctx.agents 无 fork 能力面（仅 start/followup/stop/status/list），需宿主先支持 subagent_fork 语义；记为遗留（§7.2）。
- **slash 命令**：宿主无 ctx.commands 能力面，未实施（参考实现 command.js 需 cordis 命令注册，宿主为 Web/桌面 GUI 无 CLI 命令面）。
- 验收：`TestAgentTeamsDiskLoad` 通过（13 工具完整注册）；冒烟装载日志正常。

### 6.7 R2-7 harness 语义对齐 → **已实施（分批，别名方式零破坏）**

- **read 行号输出**（tool-harness）：对齐 dsh-tool-fs `formatReadOutput`——`<path>/<type>/<content>` 块内每行 `N: text` + footer（`(End of file - total N lines)` / `(Showing lines X-Y of N. Use offset=… to continue.)`）；参数 `file_path`（DSH 名，兼容旧 `path`，file_path 优先）；offset 默认 1、limit 默认 2000。测试断言同步（harness_test.go）。
- **edit replace_all**（tool-harness）：新增 `replace_all` 布尔参数（默认 false 保持「须唯一」安全语义；true 时从后往前全量替换）；多处出现时错误提示引导用 replace_all。
- **bash description/timeoutMs**（tool-harness）：新增可选 `description`（参考实现必填，repo 兼容旧调用方保持可选——文档化差异）与 `timeoutMs`（>0 覆盖默认 120s 传给 ctx.bash.exec；0=不超时）。
- **write file_path 别名**（tool-harness）：write 同 read 双参数名。
- **ask_user questions 数组**：tool-system **插件侧 schema** 增 `questions`（DSH ask_user_question 形态 [{id, question, options?, multi_select?}]）；★ t4 F5 修正（2026-09 t5）：**Go 宿主侧 schema（session_bridge.go/session_manager.go）未同步**——Go 侧仍为 question/askType/options，questions 参数被忽略、按 question 渲染（插件描述已注明降级：「传 questions 时请同时给 question 合并文本」）；宿主前端当前按单问题渲染（built 产物，前端专项 F2/F3 外），多问题 UI 记遗留（§7.2）。
- **jobs 别名**（tool-shell）：新增 `job_output`/`job_list`/`job_kill`（DSH 命名，原名保留）——job_list 需新能力：`bgRegistry.list()`（shell.go）+ `ctx.process.list`（jsplugin.go）返回 [{id,status,error?}]。
- **todo_write 别名**（tool-system）：新增 `todo_write`（语义同 update_tasks，execute 经 hostExec 映射）；宿主侧 `registerTaskTools` 提取共享 handler 并 `ArchiveHostTool` 存档 todo_write（landing 矩阵 hostTool 断言通过）。
- 验收：`TestToolHarnessJSNative` 的 read/edit/replace_all/file_path 断言通过（bash 步为环境依赖，见 §6.13）；`TestToolLandingMatrix` 243 工具零未落地；node `--check` 全部插件通过。

### 6.8 R2-8 工具集去重 → **已实施（改装载逻辑 + 运行时数据清理）**

- **装载逻辑（持久生效）**：`applyToolsetPlugin` 在磁盘插件包存在且 main 源码有效时跳过内嵌 code 装载（`diskPluginCodeAvailable` 判断，缺失回退内嵌）——13 个插件条目降级为 name-only 白名单声明，工具由 `LoadGlobalPlugins` 磁盘包注册；DisabledTools 经可见性收敛白名单（−DisabledTools）同样生效。
- **运行时数据**：`.pair/toolsets/default.json`（git-ignored）清理——13 条内嵌 code 置空（agent-teams 101KB 陈旧拷贝等）、删除冗余 `builtin:shell`/`builtin:web` 条目（与 tool-shell/tool-web 插件条目重复声明；注意 tool-web 原本无插件条目——补 name-only `tool-web` 条目保住白名单）、清 tool-core 陈旧 DisabledTools（read_file 等旧名）。
- 验收：冒烟「已装载 44 个工具集（**17** 个插件）」（基线 18 → 17：−2 builtin +1 tool-web）；web 工具白名单不丢（tool-web 条目替代 builtin:web）；`TestToolLandingMatrix`/`TestApplyWorkspaceToolsetWhitelist` 通过。

### 6.9 R2-9 Go 旧名工具删除 + 消费方同步 → **已实施（生产注册面已零旧名；Go 实现保留为测试/归档专用）**

- **核实**：生产注册路径（RegisterHostFrameworkTools + 磁盘插件 + embeddedToolRegistrars）均不注册 read_file/write_file/edit_file/run_command/search_content/search_files——旧名 Go 实现（registerCoreTools/registerSearchTools）仅被 builtinPluginSpecs（归档二进制）+ **89 处测试** RegisterDefaultTools 引用。删除将破坏测试面 → 保留为「测试/归档专用」并记录风险清单（§7.2），生产 agent 面已零旧名。
- **消费方同步**：
  - `node_bridge.go` 服务映射旧名 → 新名（fs.read→read、fs.write→write、bash.exec→bash；fs.list 无对应工具改直连 os.ReadDir）——修复 npm 插件 ctx.fs 在生产 registry 下旧名查不到的潜在断链；`node_bridge_test.go` 断言同步。
  - `config/roles/{explorer,verifier,planner,reviewer}.md` 旧工具名 → 新名（read/bash/glob/grep）；Go 内置提示（reviewer.go/planner.go）同步（`TestRolePromptDiskDefaultSync` 通过）。
  - reviewer.go 的 run_command 审核启发式保留（历史消息兼容 + 新名工具 bash 不在写类名单——bash 写类语义已由 requiresApproval 元数据覆盖）。
  - 前端 RightPanel 图标映射：前端为 built 产物（web/assets），旧名分支保留作兼容（F2/F3 前端专项）。
- harness 别名（registerHarnessAliases）：保留（测试专用；生产 embedded 注册表无旧工具时别名自动跳过）。

### 6.10 R2-10 web-api /api/ext/fs/* 三路由删除 → **已实施**

- `.pair/plugins/web-api/index.js` 删除 `/api/ext/fs/{read,exists,list}` 三条路由（与 fs-api 重复，实测零消费者：grep 仅 docs 提及），保留 status/fetch/routes/async/echo 演示面；顺带移除不再使用的 fs inject 与 fsSvc。web-api 路由 8 → 5。

### 6.11 R2-11 goal/subagent/workflow 机制 → **未实施（按计划 t3+ 或文档化）**

- 需求评估结论不变：goal/subagent/workflow 属宿主 Loop 机制级新增（需宿主 Loop 状态机支持），非插件侧可独立完成；记为 §7.2 遗留，建议后续专项（P1：goal；P2：subagent/workflow）。

### 6.12 R2-12 前端死组件/产物（F2/F3）、G2/G4、审批遗留字段 → **未实施（defer，超 t2 inScope）**

- 前端专项（OutputPanel/SubAgentBlock/TasksPanel/router.js + 产物堆积）需 plugins-src/ui-app 专项清理；G2/G4 需产品决策；L4(旧) 审批遗留字段为纯兼容死代码。

### 6.13 测试结果与环境依赖说明

- `go build ./...` ✅、`go vet ./...` ✅（GOCACHE/GOMODCACHE 重定向 `.verify-tmp/`，CGO_ENABLED=1）。
- `go test ./internal/agent/ -count=1 -timeout 10m`：**非环境依赖测试全部通过**；环境依赖失败清单（本 DSH 沙箱限制，非代码回归）：
  - **msys bash 信号管道**（`bash.exe: couldn't create signal pipe / CreateFileMapping … Win32 error 5`——受限 token 下 msys bash 无法创建命名管道，DSH 沙箱文档化边界）：TestToolRunCommand、TestRunShellWithBash、TestRunShellBacktick、TestRunBackground、TestRunCommandInLoop、TestBGCrossRegistry、TestDetectBash、TestJSPluginCtxBashLogger、TestToolCoreJSNative（run_command 步）、TestToolHarnessJSNative（bash ⑥ 步，R2-7 断言步均过）、TestHarnessAlias_GlobGrepBash（tool-git 已改 ctx.process.exec 直连后**通过**）。
  - **Chromium/rod 下载**（写 `%APPDATA%\rod` 被沙箱拒绝）：TestWebDebugTimeoutMs。
  - **go/node 子进程环境**：TestRunCode_Go（go 环境）、TestRunCodeNested（node/go 子进程）。
  - 上述测试在正常开发环境（t1 基线环境）可执行；t1 报告的 5 个 binary 插件失败 + office csv_read/word_read 断言失败本轮已全部修复通过。
- **插件装载冒烟**：`companion_r2.exe`（临时构建，已清理）WEB_PORT=9322 启动 → `/api/health` 200、`/api/plugins` 200（43 插件）、「工具集 44 个（17 个插件）」、启动日志无 FATAL/panic/重复注册冲突 → 探活后终止。
- **前端构建**：未执行（npm/vite esbuild spawn 在本沙箱被 EPERM 阻断——队长确认有全权限通道，按需沙箱外执行；本轮无前端产物改动）。

---

## 7. 遗留项与风险清单（t2 结项）

### 7.1 本轮未实施（按计划 defer / 超范围）

| 项 | 状态 | 原因/去向 |
|---|---|---|
| R2-11 goal/subagent/workflow 机制 | defer | 需宿主 Loop 机制级支持（P1 goal / P2 subagent/workflow） |
| R2-12 前端死组件 F2/F3、G2/G4、审批遗留字段 | defer | 前端专项 / 产品决策 |
| R2-6 memberProvider fork | defer | 宿主 ctx.agents 无 fork 能力面，需宿主先支持 subagent_fork |
| R2-6 slash 命令 | defer | 宿主无 ctx.commands 命令面（Web/桌面 GUI 形态） |
| R2-7 ask_user 多问题 UI | defer | 前端为 built 产物，多问题渲染属前端专项；schema 已对齐 |

### 7.2 风险清单（被删/保留工具的消费方记录）

1. **Go 旧名工具实现保留（测试/归档专用）**：registerCoreTools/registerSearchTools + builtinPluginSpecs 的 core/fs-search 条目被 89 处测试的 RegisterDefaultTools 与 plugins-src 归档二进制引用——本轮不删（删除需先迁移测试面），生产注册面已零旧名（R2-9 已核实）。消费方引用已同步：node_bridge（→read/write/bash）、config/roles 4 个 md（→read/bash/glob/grep）、前端 built 产物保留旧名图标分支作兼容。
2. **jobs/permission/provider/vterm 死包**：保留待产品确认（t1 判定）；删除前需确认无 build tag 引用（实测无）。
3. **t4 L4 RegisterProviderImpl 还原顺序**：文档化（单插件场景无影响），栈式还原留后续。
4. **default.json 运行时数据**（git-ignored）：本轮清理仅作用于本机运行时态；装载逻辑（applyToolsetPlugin 磁盘优先）为持久生效项；其他工作区/安装目录的 default.json 若仍带内嵌 code，装载时同样被跳过（磁盘优先）。
5. **read 输出形态变化**：tool-harness read 由纯文本 → DSH 行号块；消费方为 LLM 与通用消息流（无格式断言），前端渲染为文本展示无破坏；harness_test.go 断言已同步。
6. **node_bridge fs.read 输出形态**（t4 F3，2026-09 t5 文档化）：npm 插件 ctx.fs.read 经桥映射到 tool-harness read，返回 DSH 行号块而非原始文本；当前仓库零消费者，若第三方解析型插件出现需在桥接层剥离包装（前端专项）。

### 7.3 t4 审查 findings 处置记录（2026-09 t5 集成收尾）

t4 verdict=pass，5 个非阻塞 findings 全部在 t5 处置：

| ID | 严重度 | 处置 | 证据 |
|---|---|---|---|
| F4 | medium | **已修复**：agent-teams profile=default 种子任务按序接线——TEAM_PROFILES 任务增 ref/deps/reviewedTaskRef，seed 两遍建任务（先登记 ref→id 再静态接线：implementation←requirements、verification←implementation、review←implementation 且 reviewedTaskId=实施任务 id） | `.pair/plugins/agent-teams/index.js`（TEAM_PROFILES + create seed 块）；node --check 通过 |
| F1 | medium | **已修复**：session_context.go `fileModifyTools`/`toolPathParams` 增 edit/write（file_path/path 双参数名）；storm_breaker.go `repeatSuccessSignature` 增 write/edit（旧名保留兼容历史消息） | `internal/agent/session_context.go`、`internal/agent/storm_breaker.go`；go build 通过 |
| F2 | low | **已修复（全插件清扫）**：14 个 tool-* 插件 index.js 的 LLM 面向文案旧名替换（read_file→read、write_file→write、edit_file→edit、run_command→bash、search_content→grep、search_files→glob、list_files→glob）；tool-core purpose/头注释更正为实际 3 工具；grep 零残留（built 资产 ui-modals/ui-right-panel 除外，前端专项） | 全插件 node --check 通过；grep 验证 |
| F3 | low | **已文档化**：node_bridge.go fs.read 映射处注释说明输出形态变化（DSH 行号块）与桥接层剥离方案（当前零消费者） | `internal/agent/node_bridge.go` 注释 |
| F5 | low | **已修正**：§6.7 ask_user 表述改为「仅插件侧 schema 增 questions；Go 宿主侧未同步（questions 被忽略，已文档化降级）」 | 本文件 §6.7 |

---

## 8. 集成收尾（t5 · 2026-09）

> t5（integration）执行记录：t4 findings 处置、全量构建、启动冒烟、docs 收尾、git 提交。

### 8.1 构建与冒烟

| # | 命令 | 退出码 | 结果 |
|---|---|---|---|
| 1 | `cd cmd/companion/web-ui && npm run build`（build-ui.mjs → plugins-src/ui-app vite → sync-web-dist.mjs） | 0（脚本不传播失败） | ⚠️ **沙箱 EPERM**（`[commonjs--resolver] spawn EPERM`，vite/esbuild 子进程 spawn 被 DSH 沙箱拦截）——已报队长走全权限通道复跑；本轮**无前端源码改动**，dist 为既有产物，无需重建 |
| 2 | `CGO_ENABLED=1 go build -o pair.exe ./cmd/companion` | 0 | ✅ pair.exe 67.5MB（git-ignored） |
| 3 | 启动冒烟 WEB_PORT=9323 | 0 | ✅ /api/health 200、/api/plugins 200（43 插件）、/api/tools 200；日志「43 个插件」「44 个工具集（17 个插件）」；FATAL/panic/重复/冲突=0；探活后终止清理 |

### 8.2 收尾与提交

- **t4 findings**：F1/F2/F4 修复、F3 文档化、F5 文档修正（详见 §7.3）；t3 复核发现的 tool-bridge package.json purpose 一并同步 lockdown。
- **docs**：本文件（§7.3/§8）、`docs/plugin-round2-verification.md`（t3 报告 + §8 t5 集成结论）、`docs/plugin-gaps.md`（遗留状态）全部收尾。
- **git**：Round2 全部改动 + docs 提交（提交信息见 t5 output）；`pair.exe`、运行时数据（.pair/toolsets/default.json、logs/）git-ignored，`git status` 仅预期产物。
- **交付物**：pair.exe（可交付 Round2 版本）+ 实施/验证/审查/集成四份文档 + 全部源码改动。



