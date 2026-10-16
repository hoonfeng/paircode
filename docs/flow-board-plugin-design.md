# 工程流程与看板 · 插件化设计方案

> 状态：设计草案（待决策确认后实施）
> 目标：对话中提供「工程进度看板 + 工程流程」，按项目类型预设有初始流程，用户可自定义流程与看板。
> 硬约束：**框架核心零改动**；新增业务 = 新增数据文件 + 新插件包，不改 `internal/`。

---

## 0. 结论先行

本需求可以**完全用现有插件机制实现，框架层零改动**：

| 要素 | 承载方式 | 是否改核心 |
|---|---|---|
| 流程定义（预设 / 用户自定义） | 声明式 JSON：插件包 `assets/flows/*.json`（预设） + 工作区 `.pair/flows/*.json`（用户/项目） | ❌ 零改动 |
| 流程实例（本次对话跑哪个流程、到哪一步） | 工作区 `.pair/flow-runs/*.json`（按 `convId` 绑定） | ❌ 零改动 |
| 进度数据真相（具体任务条目） | **复用现有** `.pair/tasks/*.json`（`update_tasks`）——不另造任务系统 | ❌ 零改动 |
| 看板 UI | 插件 client 半：`ui.registerView`（主区 tab 看板）+ `ui.registerSlot`（常驻徽标） | ❌ 零改动 |
| 对话内「有流程」（agent 每轮知道处在哪一阶段） | `ctx.systemPrompt.section` 注入当前阶段/门禁 + `ctx.hooks` 门禁拦截 | ❌ 零改动 |
| 项目类型识别 | 插件 host 半读工作区特征文件（`go.mod`/`package.json`/…）自判 | ❌ 零改动 |
| 看板数据接口 | 插件 host 半 `ctx.webServer.register` 自建 `/api/flow/*` | ❌ 零改动 |

**唯一薄弱接线点**是与既有 `update_tasks` 的复用方式（详见 §9），已给出 4 条路径、首选路径的数据契约已取证确证。

### 0.1 已定决策（2026-09，用户确认）

- **D1 流程强度 = 重**：阶段 + 可执行门禁 + **审批卡点** + **证据留档** + **失败回滚建议**（§10 按此定稿）。
- **D2 看板位置 = 主内容区 tab**（可并排对话）；状态栏徽标**不作为默认挂载**，保留为后续可选增强（§6）。
- **D5 推进方式 = 先谈定设计，暂不动手实现**（本文档即本轮交付物；实施计划见 §11，未启动）。

> 结论不变：**框架核心零改动**。「重」强度所需的审批、证据、回滚建议全部由插件 host 半 + 工作区数据文件实现，不触碰 `internal/`。
>
> ★ 扩展需求（流程导出/导入 · 全局级与项目级 · 项目随时接入 · 流程管理面 · 交互完备与三感知）见 **§14**。

---

## 1. 需求拆解

- **R1 对话中有工程进度看板**：常驻可见（不抢主区）+ 可展开详情（阶段/关卡/证据）。
- **R2 按项目类型预设流程**：Go 服务 / Web 前端 / Python / 文档办公 / 库与 CLI / 调试排错 / 插件开发 …
- **R3 用户可自定义流程与看板**：改数据文件、对话生成、或表单；预设可被覆盖且不被覆盖回冲。
- **R4 不修改框架核心**：新增一种业务（如「Rust 项目流程」）只需新增 JSON，不动 `internal/`。

---

## 2. 现状取证（全部为本次实测）

| # | 事实 | 证据 |
|---|---|---|
| F1 | 插件双半 + 磁盘包结构（`package.json`/`index.js`/`client.js`/`assets/`），启动自动装载 | `docs/plugin-development.md` §12/§13；`.pair/plugins/*/` |
| F2 | client 半三种 UI 挂载点：`registerView`（主区 tab 工作台，可并排对话、懒挂载+保持）/ `registerPanel`（插件面板内）/ `registerSlot`（预定义槽位） | `plugins-src/ui-app/src/plugin-runtime.js` 中 `registerView`/`registerPanel`/`registerSlot` 实现；`docs/plugin-development.md` §12.1 |
| F3 | 预定义槽位：single 型 `titlebar`/`sidebar`/`right-panel`/`statusbar`/`editor`/`chat`；list 型 `overlay`/`titlebar-right`/`activitybar`/`editor-toolbar`/`chat-tools`/`statusbar-items` | `docs/plugin-development.md` §12（槽位清单） |
| F4 | **现成看板范例（双挂载）**：autopilot 徽标（`registerSlot`）+ `registerView('autopilot-board')` 主区看板，数据走 `/autopilot/rounds`，2s 轮询 | `.pair/plugins/autopilot/client.js`（`registerSlot` ~L203、`registerView` ~L251、`setInterval` ~L385） |
| F5 | client 半可直连宿主 HTTP：`ui.http.get/post` 即通用 `apiGet/apiPost`，**该层无白名单** | `plugin-runtime.js:595-598`；`plugins-src/ui-app/src/api.js:35` |
| F6 | host 半能力：`ctx.webServer.register`（自建接口）/`ctx.tools.register`/`ctx.commands.register`（slash 命令，需 `inject:['commands']`）/`ctx.systemPrompt.section`/`ctx.registerSettings`/`ctx.sse.register`/`ctx.hooks.register`/`ctx.registerClientMethod`（RPC） | `cordis(op=services)` 输出；`.pair/plugins/git-api/index.js` 用 `ctx.webServer.register` 范例 |
| F7 | host→浏览器事件桥：`/api/plugins/client-events` 2s 轮询增量；反向 `/api/plugins/event` | `docs/plugin-development.md` §12 |
| F8 | 任务清单真相：工作区 `.pair/tasks/<id>.json`，字段 `{id, subject, description, status(pending/in_progress/completed/cancelled), dependencies[], convId?, created_at, updated_at}`；对外 `GET /api/tasks?convId=`；前端右栏已有「任务进度」面板消费 | `internal/agent/task_manager.go:31-40`；`cmd/companion/kernel_register.go:67`；`plugins-src/ui-app/src/agent-events.js:89` |
| F9 | 执行计划：`.pair/plans/*.json`（`ExecutionManager`，Step 带 status）；但 plan 体系 UI 侧**已移除**（`/api/taskplan` 于 2026-08-31 移除） | `internal/agent/execution_plan.go`；`cmd/companion/web_server.go:1580` |
| F10 | agent-teams 插件：DAG 计划 + 审批 + `.agent-teams/` 磁盘真相 + `client.js` 计划看板（可借造型，但语义是「团队任务 DAG」，非工程流程阶段） | `.pair/plugins/agent-teams/{index.js,client.js}` |
| F11 | `tool-workflow` 是**脚本编排**（`pipeline/parallel/phase/log`，宿主 goja 运行器），不是流程模板/阶段模型——本方案不重复它 | `.pair/plugins/tool-workflow/index.js` |
| F12 | ❗Go 侧**无项目类型识别**实现（`grep 项目类型\|projectType\|detectProject` 零命中） | 本次 grep 结果 |
| F13 | ❗**无流程/阶段模型、无流程看板**（这两者即本方案要补的缺口） | 综合 F8–F12 |
| F14 | 预设机制先例：工具集预设（`presetModes`，中文名 + 播种标记 + 阶梯迁移，用户改动优先不被回冲）可作为「预设流程分发」的成熟范式 | `internal/agent/preset_toolsets.go` |

---

## 3. 分层设计

```
L3 分发与自定义：预设包（插件 assets/flows/*.json） + 用户/项目覆盖（工作区 .pair/flows/*.json）
                    ↑ 同名覆盖；预设只在缺失时播种（对齐 preset_toolsets 播种范式 F14）
L2 看板 UI（client 半，零核心改动）
     ├── ui.registerSlot({slotId:'statusbar-items'})  常驻徽标「阶段 3/7 · 门禁 1 待过」
     ├── ui.registerView({id:'flow-board'})           主区 tab 完整看板（可并排对话）
     └── （可选）ui.registerSlot({slotId:'chat-tools'}) 对话工具栏「流程」快切
L1 流程引擎插件（host 半，新插件包 tool-flow）
     ├── 工具面：flow_list / flow_start / flow_status / flow_advance / flow_define / flow_recommend
     ├── 项目类型识别（读工作区特征文件 → .pair/flows/_detected.json 缓存）
     ├── 接口面：ctx.webServer.register('/api/flow/...')（给看板只读拉数）+ 写操作走 RPC
     ├── 提示注入：ctx.systemPrompt.section（当前阶段/门禁 → agent 每轮可见）
     ├── 门禁执行：ctx.hooks.register（Stop/PreToolUse 拦截未过门禁的收尾）
     ├── 命令面：ctx.commands.register('/flow'、'/flow-board')
     └── 配置面：ctx.registerSettings（默认流程/自动推荐/门禁强度）
L0 数据层（磁盘，声明式 JSON）
     ├── 流程定义  .pair/flows/<flowId>.json          （用户/项目；预设来自插件包）
     ├── 流程实例  .pair/flow-runs/<runId>.json        （按 convId 绑定）
     ├── 任务条目  .pair/tasks/*.json                  ← 复用现有（update_tasks 真相，F8）
     └── 类型缓存  .pair/flows/_detected.json
```

要点：
- **L0 是唯一真相**（刷新/重启/换标签页都自洽），L1 只做状态跃迁与校验，L2 只做渲染——对齐 `docs/plugin-development.md` §12.2「事实数据走持久化接口，运行态只做视觉提示」的既有纪律。
- **新增项目类型流程 = 新增一份 JSON**（放插件包 assets 或工作区 `.pair/flows/`），不改任何代码 —— 满足 R4。

---

## 4. 扩展点映射（需求 → 机制，逐条零核心改动）

| 需求 | 采用机制 | 依据 |
|---|---|---|
| 看板常驻可发现 | `ui.registerSlot({slotId:'statusbar-items'})` 徽标 | F3、F4（autopilot 徽标先例） |
| 看板详情 | `ui.registerView({id:'flow-board'})` + `layout.openViewTab('flow-board')` | F2、F4 |
| 对话中看到进度 | 现有右栏 TaskPanel（`.pair/tasks`）+ 插件徽标 | F8 |
| 看板读数据 | 插件 host 半 `/api/flow/*`（`ctx.webServer.register`）+ client 半 `ui.http.get` | F5、F6 |
| 看板写操作（推进阶段/勾选门禁） | `ctx.registerClientMethod` + `ui.invoke`（RPC） | F6 |
| 实时刷新 | 事件桥 `ctx.emit → ui.on`（2s 轮询）或 `ctx.sse.register` | F7 |
| agent 每轮知道流程位置 | `ctx.systemPrompt.section({name,order,text})` 注入阶段与未过门禁 | F6 |
| 阶段推进由 agent 驱动 | 工具面 `flow_advance`（`ctx.tools.register`） | F6 |
| 门禁强制 | `ctx.hooks.register('Stop'|'PreToolUse')` 返回 `{block,feedback}` | F6 |
| 用户一键切流程 | slash 命令 `/flow`（`ctx.commands.register`，`inject:['commands']`） | F6 |
| 流程/看板预设 | 插件包 `assets/flows/*.json` + 播种策略（对齐 preset_toolsets） | F1、F14 |
| 用户自定义 | 工作区 `.pair/flows/*.json`（手写 / agent 经 `flow_define` 生成） | F1、F14 |
| 项目类型识别 | host 半读特征文件（`ctx.fs`） | F6、F12（现状无此能力，故补在插件侧） |
| 进度条目同步 | 复用 `.pair/tasks/*.json` | F8、§9 |

**核心改动 = 0**。所有业务规则落在 JSON 与插件包内，框架层不出现「流程/阶段/看板」任何字样。

---

## 5. 数据契约

### 5.1 流程定义 `FlowDef`（`.pair/flows/<flowId>.json` 或插件 `assets/flows/<flowId>.json`）

```jsonc
{
  "id": "go-backend-feature",
  "title": "Go 后端功能开发",
  "version": 1,
  "source": "preset",                     // preset | user（由落盘位置决定，勿手改）
  "appliesTo": {                           // 项目类型匹配（§7）
    "types": ["go"],
    "signals": ["go.mod", "go.work"]       // 任一命中即适用
  },
  "stages": [
    {
      "id": "recon",
      "title": "调研与定位",
      "goal": "搞清代码结构与调用链，确定改动面",
      "tasks": ["搜索关键符号", "读关键文件", "列出受影响文件清单"],
      "gates": [
        { "id": "g-recon-1", "kind": "check",   "text": "已列出受影响文件清单", "required": true },
        { "id": "g-recon-2", "kind": "command", "text": "基线编译通过", "cmd": "go build ./...", "required": true }
      ]
    },
    { "id": "impl", "title": "实施修改", "goal": "完成代码改动", "tasks": ["按方案改代码"], "gates": [] },
    { "id": "verify", "title": "验证", "goal": "编译+运行验证", "tasks": ["编译", "跑测试"],
      "gates": [{ "id": "g-ver-1", "kind": "command", "text": "go test ./... 通过", "cmd": "go test ./...", "required": true }] }
  ],
  "board": {                               // ↓ 看板自定义（R3）
    "kind": "kanban",                      // kanban | checklist | timeline | swimlane
    "columns": ["stage", "status"],
    "fields": ["goal", "gates", "evidence"],
    "groupBy": "stage"
  }
}
```

**门禁 kind**：`check`（自评/人工确认）/ `command`（插件可代跑并记证据）/ `artifact`（产物存在性，如文件、截图路径）。
**门禁 kind（按 D1「重」定稿）**：

- `check`：agent 自评 + 证据文本；
- `command`：插件代跑（`cmd`），输出留存为证据文件；
- `artifact`：产物存在性（文件 / 截图 / 报告路径）；
- `approval`：**审批卡点** —— 必须由用户显式批准（`ask_user` 回答，或看板按钮经 RPC 批准），插件**不得自动通过**。

**门禁字段**：`{id, kind, text, cmd?, artifact?, required?, blocking?, onFail?}`；
`onFail` = `advise`（仅给建议）| `rollback`（触发回滚建议流程，见 §10.3）。

### 5.2 流程实例 `FlowRun`（`.pair/flow-runs/<runId>.json`）

```jsonc
{
  "runId": "run-1730000000000000000",
  "convId": "conv_xxx",                    // ★ 会话绑定（多会话并行必需）
  "flowId": "go-backend-feature",
  "flowVersion": 1,
  "status": "active",                      // active | completed | cancelled
  "currentStage": "impl",
  "stages": [
    { "id": "recon", "status": "completed", "startedAt": "…", "completedAt": "…",
      "taskIds": ["1730000000000000001"],   // → .pair/tasks/<id>.json
      "gateResults": [ { "id": "g-recon-2", "status": "pass", "at": "…", "evidence": "go build ok" } ] },
    { "id": "impl", "status": "in_progress", "taskIds": [], "gateResults": [] }
  ],
  "createdAt": "…", "updatedAt": "…",

  "gitHead": "abc1234",                          // 流程启动时的 git HEAD（回滚建议基线，§10.3）
  "modifiedFiles": ["internal/agent/loop.go"],   // run 期间经 PostToolUse 钩子记录的改动文件
  "evidenceDir": ".pair/flow-runs/evidence/run-1730000000000000000/"  // 证据留档目录（§10.2）
}
```

### 5.3 与既有任务机制的对接（关键）

阶段任务**不另造**，写进现有任务清单，从而自动获得：右栏任务面板、`GET /api/tasks?convId=`、`progress_checker`、会话历史压缩识别（`history_condense.go` 已特判 `update_tasks`）。

任务条目格式（已确证，`task_manager.go:31-40`）：

```jsonc
{ "id": "1730000000000000001", "subject": "搜索关键符号", "description": "阶段 recon",
  "status": "pending", "dependencies": [], "convId": "conv_xxx",
  "created_at": "…", "updated_at": "…" }
```

> ⚠️ `.pair/tasks/` 是**工作区级共享目录**（各会话任务混放）——历史缺陷即「无 convId 的任务被按会话过滤掉导致前端面板空白」，见 `internal/agent/task_conv_binding_test.go`。流程插件写任务**必须带 `convId`**。

---

## 6. 看板 UI 设计

| 项 | 选型 | 理由 |
|---|---|---|
| 主看板（**唯一默认挂载**，D2 决策） | `ui.registerView({id:'flow-board', title:'流程', order:20, open:false})` | 完整工作台、可**并排对话**（边看板边干活的真实场景） |
| ~~常驻徽标~~（D2 未采纳，留作后续可选增强） | `ui.registerSlot({slotId:'statusbar-items'})`：`流程 实施 3/7 · 门禁 1 待过` | 若后续反馈「找不到看板」再加；点击 `layout.openViewTab('flow-board')` |
| 可发现性替代方案（无 UI 占用） | 对话内提示（§4 的 `systemPrompt.section` 注入当前阶段）+ `/flow`、`/flow-board` 命令 | 让 agent 与用户都「知道当前在哪一步」，不占任何界面空间 |
| 对话区快捷 | （可选后续）`ui.registerSlot({slotId:'chat-tools'})` 阶段胶囊 | 对话中零跳转可见，同样非默认 |
| 看板形态 | 由 `FlowDef.board.kind` 决定渲染（kanban / checklist / timeline / swimlane） | 看板自定义（R3）落在数据层，不在代码 |
| 数据刷新 | 事实数据每次 `ui.invoke`/`ui.http.get` 拉持久化接口；运行态用 `ui.on` 事件（2s 桥）做「进行中」视觉 | §12.2 既有纪律 |
| 渲染纪律 | 一切 agent/用户文本走 `textContent`（禁用 `innerHTML` 渲模型产出） | `docs/plugin-development.md` §12 渲染纪律 |

---

## 7. 预设流程（按项目类型）

识别信号由插件 host 半扫描工作区根（`ctx.fs`）：

| 项目类型 | 识别信号 | 预设流程（阶段） | 默认看板 |
|---|---|---|---|
| Go 服务/后端 | `go.mod`、`go.work` | 调研 → 实施 → 编译 → 测试 → 交付 | kanban |
| Web 前端 | `package.json`+`vite.config.*`/`next.config.*` | 需求 → 组件设计 → 实现 → 构建 → 浏览器验证（截图） | kanban |
| Python | `requirements.txt`/`pyproject.toml` | 调研 → 实现 → 运行验证 → 测试 | checklist |
| 文档/办公 | `.docx`/`.xlsx`/`doc/**` 为主 | 提纲 → 撰写 → 校对 → 导出 | checklist |
| 库/CLI | `Cargo.toml`/`pyproject.toml`+`cmd/`、`bin/` | 接口设计 → 实现 → 单测 → 文档 | timeline |
| 插件开发 | `.pair/plugins/*/package.json`、`plugins-dist/*` | 契约确认 → 实现 host 半 → 实现 client 半 → 装载验证 → 收尾 | kanban |
| 调试排错 | 无（按任务语义） | 复现 → 定位 → 修复 → 回归 | checklist |
| 通用兜底 | 无信号 | 理解任务 → 方案 → 执行 → 验证 | checklist |

预设**随插件包发布**（`assets/flows/*.json`），首次使用时播种到工作区 `.pair/flows/`（缺失才播种、用户改动优先、不静默回冲 —— 对齐 `preset_toolsets.go` 的播种+版本阶梯迁移范式 F14）。

---

## 8. 用户自定义（R3）

三条路径，按成本从低到高：

1. **直接编辑**工作区 `.pair/flows/<id>.json`（手写或复制预设改）。`flow_define` 工具负责 schema 校验 + 落盘 + 报错定位。
2. **对话生成**：用户说「给我一个 Rust 项目流程：调研→实现→clippy→测试」→ agent 调 `flow_define` 生成 JSON 并落盘。
3. **表单**（可选，后续）：插件面板内表单编辑（`ctx.registerSettings` 或 `registerPanel`），MVP 不做。

看板自定义同样在数据层：`board.kind/columns/fields/groupBy`，插件按声明渲染；未声明的字段走默认。

---

## 9. 唯一薄弱接线点：与 `update_tasks` 的关系

现状：`update_tasks` 生产侧由 **tool-system 插件**注册（`.pair/plugins/tool-system/index.js:235`），宿主侧实现 `registerTaskTools` 保留但生产不注册（`internal/agent/tool_plugin_gen.go:52,94` 注释）；`ctx.hostTool.exec` 只执行**宿主存档工具**（`internal/agent/jsplugin.go:594`，存档清单见 `legacy_host_tools.go`，其中**不含** task 类）。

因此流程插件与任务清单的对接按优先级：

| 路径 | 做法 | 评价 |
|---|---|---|
| **A（首选）** | 插件直接写 `.pair/tasks/<id>.json`（格式已确证 §5.3，TaskManager 无状态、只读目录） | 零核心改动、可控；需**格式守卫测试**防漂移 |
| **B** | 探测 `ctx.hostTool.names()` 是否含 `update_tasks`，有则 `ctx.hostTool.exec('update_tasks', …)` 复用 | 优雅但**未验证**（POC 第一项） |
| **C** | 提示注入 + 约定：阶段推进时由 agent 自己调 `update_tasks` 同步 | 零耦合，但依赖 agent 守规矩（可配合门禁检查兜底） |
| **D（不推荐）** | 宿主新增写接口 | 属框架层加业务接口 → 违背硬约束，仅在 A/B/C 全不可行时评估 |

> 建议落地顺序：先 POC B（1 次探测即可定论）；不行则用 A，并加「插件写出的任务文件能被宿主 `/api/tasks` 正确读出」的集成测试。

---

## 10. 门禁执行（「重」强度 · D1 定稿）

三项机制全部落在插件 host 半 + 工作区数据文件，**零核心改动**。

### 10.1 强制拦截（阶段未完成不得收尾）

- `ctx.hooks.register('Stop', fn)`：会话收尾时当前阶段存在 `required && blocking` 门禁未过 → 返回
  `{block:true, feedback:'未过门禁 g-ver-1：go test ./... 未通过（证据见 .pair/flow-runs/evidence/…）'}`，把 agent 拉回流程继续。
- `ctx.hooks.register('PreToolUse', fn)`：未过前置门禁时拦破坏性操作（`rm`、批量覆写、`git reset` 等），返回 `{block:true, feedback}`。
- 强度可配（`ctx.registerSettings`）：`off`（**只记录**：不拦截、不注入提示，证据与看板展示照常留档）/
  `warn`（注入提示但**不**拦截）/ `enforce`（拦截破坏性操作 + 注入未过门禁，默认按 D1 取 enforce）。

> ★ **定案：`Stop` 硬拦在当前核心不可实现（2026-10-16 实测，正式写入设计）**
>
> 核心 `internal/hook/hook.go` 的 `IsBlocking` **只**对 `PreToolUse` / `UserPromptSubmit` 返回 true；
> `Stop` 钩子即便返回 `{block:true, feedback}`，核心也不会据此阻止会话收尾（当前 hook 门禁机制不含
> 「Stop 可阻断」语义）。因此本小节首条「停不下来」的**硬拦**在当前核心下无法实现。
>
> **降级影响（诚实标注）**：无法用机制「卡住」收尾，只能软拉回 ——
> ① 每轮把「当前阶段 + 未过必过门禁 + 下一步动作」注入系统提示（`systemPrompt.variable`）；
> ② `PreToolUse` 拦破坏性操作（`rm -rf` / `git reset --hard` / 文件删除），enforce 档下生效；
> ③ 工具与 RPC 返回值带进度摘要，便于 agent 与用户即时看到「还差哪些门禁」。
> 结论一句话：**阻止破坏性动作可信；阻止「提前收尾」不可信**（只提示，不阻断）。
>
> **用户可选两条路**：
> (a) **改核心**：让 `Stop` 进入可阻断集合（`IsBlocking` 放行 `Stop` + 调用点消费 `block`），需核心发版，
>     且影响面覆盖所有注册 `Stop` 钩子的消费者（语义变更是破坏性的）；
> (b) **接受替代**（当前默认，零核心改动）：按上面 ①+②+③ 落地；三档强度（`enforce`/`warn`/`off`）行为
>     已全部固化为可复现断言（见 §14.20 验证表）。

### 10.2 审批卡点（kind = `approval`）

`flow_advance` 遇到 `approval` 门禁**不自动通过**，而是把「待批准」写入 run 并对外暴露，两条批准通道（都在插件内）：

1. **对话通道**：工具返回待批准提示 → agent 用现有 `ask_user` 询问用户 → 用户作答后 `flow_advance` 带 `approvalGateId` 落库。
2. **看板通道**：看板「批准」按钮 → `ui.invoke`（RPC 到 host 半 `ctx.registerClientMethod`）→ host 半校验后写入 `gateResults`（记录 `approvedBy:'user'`、时间、附言）。

审批记录同样进 `gateResults`，因此**审批走哪条通道都留痕**，刷新/换标签页不丢。

### 10.3 证据留档与失败回滚建议

**证据留档**（每次门禁执行都落盘）：

```
.pair/flow-runs/evidence/<runId>/<gateId>.json    # {gateId, kind, status, at, cmd?, exitCode?, summary}
.pair/flow-runs/evidence/<runId>/<gateId>.log     # command 门禁的原始输出（截断保留尾部）
.pair/flow-runs/evidence/<runId>/<gateId>.png     # artifact 门禁的截图/报告引用（路径记录，不拷贝大文件）
```

run 的 `evidenceDir` 字段指向该目录 → 看板门禁行可直接展开证据。

**改动跟踪**：`ctx.hooks.register('PostToolUse', fn)` 捕获 `write`/`edit`/`apply_patch` 等写操作 → 追加 `modifiedFiles`（去重），并在 `flow_start` 时用 `ctx.bash.exec('git rev-parse HEAD')` 记录 `gitHead` 基线。

**失败回滚建议**（生成建议，**不自动执行**）：

门禁失败且 `onFail='rollback'` 时，插件输出结构化建议（由 agent 呈现给用户，或看板门禁行展示）：

```jsonc
{
  "failedGate": "g-ver-1",
  "reason": "go test ./... 失败（3 个用例）",
  "evidence": ".pair/flow-runs/evidence/run-…/g-ver-1.log",
  "rollbackOptions": [
    { "kind": "inspect",  "cmd": "git diff --stat",                  "risk": "none",  "text": "先看改动范围" },
    { "kind": "discard",  "cmd": "git checkout -- <modifiedFiles>",  "risk": "high",  "text": "丢弃本次改动（需用户批准）" },
    { "kind": "stash",    "cmd": "git stash push -m flow-<runId>",   "risk": "medium","text": "暂存以便对比" },
    { "kind": "reset",    "cmd": "git reset --hard <gitHead>",       "risk": "high",  "text": "回到流程起点（危险，必须用户批准）" }
  ]
}
```

> ⚠️ **诚实限制**：文件快照能力已于 2026-09-21 整体移除（见 `internal/agent/preset_toolsets.go` 注记），因此回滚**只能依赖 git 基线**（无 git 仓库的项目降级为「列出 `modifiedFiles` 供人工处理」）。所有 `high` 风险命令**必须经用户批准**（对话确认或看板按钮），插件绝不自行执行。

---

## 11. 实施计划（按「重」强度排期 · 状态：**P1–P3 已落地**，见 §14.19；D5 的「暂不动手实现」已于 2026-10-16 解除）

**P0 接线验证（最先做，成本最低、风险最高）**
1. 非默认端口 + 独立二进制启动，POC 探测 `ctx.hostTool.names()` 是否含 `update_tasks`（§9 路径 B）。
2. 定路径：B 可用则复用宿主执行器；否则走路径 A（直写 `.pair/tasks/*.json`，格式见 §5.3）并加**格式守卫集成测试**。
3. 产出结论写入本文档 §9（把「未验证」改为实测结论）。

**P1 最小闭环（阶段 + 命令门禁 + 证据留档）**
1. 插件包 `tool-flow`（host 半）：`flow_list` / `flow_start` / `flow_status` / `flow_advance` / `flow_define`；`.pair/flows`、`.pair/flow-runs`、`evidence/` 读写；`/api/flow/state` 只读接口。
2. 证据留档（§10.3）：`command` 门禁输出落 `.log`、汇总结论落 `.json`；`PostToolUse` 记录 `modifiedFiles`；`flow_start` 记 `gitHead`。
3. client 半：`ui.registerView('flow-board')` 看板（**唯一挂载**，kanban 一种形态即可）。
4. 验证（按 `docs/plugin-development.md` §12.3）：CDP 断言「启动流程 → 任务出现在右栏任务面板 → 看板显示阶段/门禁/证据可展开」。

**P2 审批卡点 + 门禁强制**
5. `approval` 门禁（§10.2）：对话通道（`ask_user`）+ 看板通道（`ui.invoke` RPC）双通路，均写 `gateResults` 留痕。
6. `ctx.hooks`（`Stop` / `PreToolUse`）拦截 + 强度配置（`off`/`warn`/`enforce`）。
7. 失败回滚建议（§10.3）：生成结构化建议，`high` 风险命令一律待用户批准。

**P3 预设与识别**
8. 8 类预设流程 JSON（`assets/flows/*.json`）+ 项目类型识别 + 播种/迁移策略 + `flow_recommend`。

**P4 体验强化**：`/flow`、`/flow-board` 命令；SSE 实时刷新；多看板形态（checklist/timeline/swimlane）；并排对话打磨；徽标槽位（若用户反馈「找不到看板」再启用，§6）。

---

## 12. 风险与未验证项（诚实清单）

| 项 | 风险 | 处置 |
|---|---|---|
| `ctx.hostTool.exec('update_tasks')` 是否可达 | 未验证 | POC 首项；不行走路径 A |
| 直写 `.pair/tasks/*.json` 格式漂移 | 中（宿主改结构则失配） | 加集成测试 + 版本守卫 |
| `.pair/tasks` 工作区级共享导致会话串扰 | 高（历史缺陷） | 一律写 `convId`；看板按 current convId 过滤 |
| client 半自动 global 作用域 | 中 | 流程定义放**工作区**数据文件，插件只带预设模板 |
| 事件桥 2s 轮询延迟 | 低 | 关键写操作后本地立即重拉；需要更实时再上 SSE |
| 全局插件面污染 | 中 | 遵循技能铁律：仅用户明确要的插件才落盘；先做 POC 再发布 |
| 与 agent-teams / workflow 语义重叠 | 低 | 明确分工：agent-teams=团队 DAG、workflow=脚本编排、tool-flow=工程阶段流程 |

---

## 13. 决策记录

### 已定（2026-09，用户确认）

| 编号 | 决策 | 落点 |
|---|---|---|
| D1 | 流程强度 = **重**（阶段 + 可执行门禁 + 审批卡点 + 证据留档 + 失败回滚建议） | §10 定稿；数据契约 §5.1/§5.2 相应扩字段 |
| D2 | 看板位置 = **主内容区 tab**（可并排对话）；徽标**非默认**，留作可选增强 | §6 定稿 |
| D5 | **暂不动手实现**，先谈定设计（本轮交付物 = 本文档 + 知识库索引） | §11 状态标「未启动」 |

### 仍待定（不影响本轮定稿）

- **D3 预设粒度**：按项目类型（§7 表，当前设计基准）/ 按任务类型（开发/调试/重构/发布）/ 两者并存。
- **D4 自定义入口优先级**：编辑 JSON / 对话生成 / 表单（当前设计以前两者为主，表单不做）。

---

# 第二部分 · 扩展：完备流程管理系统（R5–R10）

> 触发（2026-09 追加需求）：流程**可导出可导入**；分**项目级 / 全局级**；项目**可随时接入**流程；**项目管理里可指定与切换流程**；流程管理相关实现；**交互完备 + agent 感知 + 用户感知 + 进度感知**。
> 硬约束不变：**框架核心零改动**，新增能力全部落插件包 + 工作区/全局数据文件。

## 14.1 需求 → 机制映射（逐条）

| 需求 | 机制 | 依据 / 状态 |
|---|---|---|
| R5 导出 / 导入 | 工具 `flow_export` / `flow_import` + 自包含 JSON 包 | 范式对齐 `toolset_export`/`toolset_import`（`internal/agent/toolset.go:1035 ExportToolsetJSON`；`toolset_tools.go`） |
| R6 项目级 / 全局级 | 三层层级：插件预设（只读）< 全局库 < 项目库；同名覆盖 | 双层先例：技能的 `scope=project|global`；工具集预设的「不回冲」范式 |
| R7 项目随时接入 | 绑定文件 `<主根>/.pair/flows/_binding.json` + `flow_attach`/`flow_detach` | 工作区多根来自 `core.Folders` / `GET /api/workspace`（`internal/server/handler/workspace.go:14-25`） |
| R8 项目管理里指定/切换 | 看板内「项目 × 流程」矩阵页签（项目列表只读消费 `/api/workspace`） | 若要求嵌进 FileExplorer 则需改前端壳 → 非零改动（见 D8） |
| R9 流程管理 | 工具面 `flow_list/show/define/clone/delete/validate/sync` + 版本与乐观锁 | 全部 `ctx.tools.register` + `ctx.fs` |
| R10 三感知 | agent：`ctx.systemPrompt.section` + `ctx.hooks`；用户：看板 + 通知 + 命令；进度：三层（任务/阶段/项目） | `ctx.systemPrompt.section`、`ui.registerView`、`window.$toast`（`plugins-src/ui-app/src/ui-state.js:89`） |

## 14.2 三层流程库

| 层 | 位置 | 可写 | 语义 |
|---|---|---|---|
| **预设 preset** | 插件包 `assets/flows/*.json` | ❌ 只读 | 随插件发布/升级；作为模板来源 |
| **全局 global** | 全局库（载体见 §14.7） | ✅ | 跨项目共享；团队/个人沉淀 |
| **项目 project** | `<root>/.pair/flows/*.json` | ✅ | 随项目走（可进 git 评审） |

**解析优先级**：`project > global > preset`（同名取上层）。**升级不回冲**：上层（预设/全局）更新**不自动覆盖**下游副本，需显式 `flow_sync`（对齐 `preset_toolsets.go` 的「用户改动优先、不复活」原则）。

## 14.3 导出 / 导入（对齐既有范式，不发明新概念）

**导出** `flow_export { id, scope?, to?, includeMeta? }` → 单文件 JSON（建议扩展名 `.flow.json`，纯文本可 diff/评审）：

```jsonc
{
  "kind": "paircode.flow",
  "schemaVersion": 1,
  "exportedAt": "2026-09-…",
  "tool": "tool-flow@1.0.0",
  "flow": { /* FlowDef（§5.1） */ },
  "meta": { "author": "…", "tags": ["go","backend"], "repo": "github:owner/repo" }  // 预留市场发布（D9）
}
```

**导入** `flow_import { file? | json?, scope: "global"|"project", onConflict: "skip"|"overwrite"|"rename", dryRun? }`：

- `schemaVersion` 不匹配 → **拒绝并明确报错**（不静默降级、不猜测）；
- `onConflict` 必须显式指定（默认 `skip`，绝不静默覆盖）；
- `dryRun:true` → 只报告「将新增 / 将覆盖 / 将跳过」清单，不落盘；
- 导入后返回校验结果（`flow_validate` 同一实现）。

范式来源：`toolset_export`/`toolset_import`（可移植 JSON、scope 参数、可发布市场）与技能的 project/global 双层 —— 复用既有语义与用户心智。

## 14.4 项目接入 / 切换（随时接入，不破坏历史）

绑定文件：`<主根>/.pair/flows/_binding.json`

```jsonc
{
  "version": 1,
  "bindings": {
    "F:/syproject/gou-ide": { "flowId": "go-backend-feature", "scope": "project", "since": "…", "note": "主项目" },
    "F:/syproject/ref":     { "flowId": "generic",          "scope": "global",  "since": "…" }
  }
}
```

- **多根工作区**：`core.Folders` 每根各自一条绑定（`GET /api/workspace` 的 `folders` 即项目清单）。
- **工具**：`flow_attach {project, flowId}`、`flow_detach {project}`、`flow_bindings {}`（列矩阵）。
- **解析顺序**（流程启动时）：① 会话级临时选择（`FlowRun.flowId`）→ ② 项目绑定 → ③ 项目类型推荐（§7）→ ④ 全局默认 → ⑤ 无（**提示接入，不静默套用**）。
- **随时切换不清空历史**：绑定变更只影响**后续**新建的 `FlowRun`；既有 run 保留自己的 `flowId@version` 与进度（避免「换了流程，历史进度消失」）。

## 14.5 流程管理面（CRUD + 校验 + 版本）

| 工具 | 作用 | 备注 |
|---|---|---|
| `flow_list {scope?, type?}` | 全量列表（标来源层级 / 适用项目类型 / 版本） | 含预设，标注只读 |
| `flow_show {id, scope?}` | 详情 + 校验结果 + 门禁与证据要求 | — |
| `flow_define {json, scope, expectedVersion?}` | 创建 / 更新（schema 校验 + 乐观锁） | 版本冲突 → 拒绝并提示重拉 |
| `flow_clone {id, newId, scope}` | 以预设/全局为模板生成项目级副本 | 「改自己的，不动上游」 |
| `flow_delete {id, scope}` | 删除（预设只读，不可删） | 引用检查：被绑定/被 run 引用 → 警告 |
| `flow_validate {id \| json}` | 独立校验：阶段 id 唯一、`gate.kind` 合法、`cmd` 非空、引用有效 | 导入/编辑共用 |
| `flow_sync {id, from, to}` | 显式把上游新版本拉到下游 | **不自动回冲** |
| `flow_export` / `flow_import` | §14.3 | — |
| `flow_attach` / `flow_detach` / `flow_bindings` | §14.4 | — |
| `flow_start` / `flow_status` / `flow_advance` / `flow_recommend` | 执行与推荐（§4、§7） | — |

编辑入口：**JSON（`flow_define`）+ 对话生成**（P1/P3）；看板内表单编辑列为 P4 可选。

## 14.6 三感知（交互完备）

### ① agent 感知（每轮可见，零核心改动）

`ctx.systemPrompt.section({name:'flow', order:…, text})` 每轮注入：

```
[流程] 项目 go-ide · 流程 go-backend-feature（项目级 v2）· 阶段 3/7「验证」
未过门禁：g-ver-1（go test ./...，kind=command）
下一步：先执行 go test ./... 并把输出留证（evidence），再 flow_advance。
```

未绑定流程时改为提示（`flow_list` 查看 / `flow_attach` 接入），**不强制**。
配合 §10.1 的 `Stop` 钩子（required 门禁未过 → block）与 `flow_advance` 的人话摘要，agent 不会「忘了自己在流程里」。

### ② 用户感知

- **看板**（主区 tab，§6）：三页签 —— 「当前流程」「项目 × 流程矩阵」「流程库（含导入导出）」。
- **阶段变更通知**：`overlay` 槽位自绘轻提示（推荐，不依赖内部 API）或 `window.$toast(msg, 'info', 4000)`（已确证存在于 `ui-state.js:89`）。
- **命令**：`/flow`（状态 / 切换 / 推进）、`/flow-board`（打开看板）、`/flow-import <文件>`。
- **对话流可见**：工具的返回文本含进度摘要（即使用户不看板也能读到「阶段 3/7」）。

### ③ 进度感知（三层）

| 层 | 数据真相 | 展示位置 |
|---|---|---|
| 任务 | `.pair/tasks/*.json`（现成） | 右栏 TaskPanel（**零改动复用**） |
| 阶段 | `.pair/flow-runs/*.json` | 看板泳道 + 顶部进度条 + 门禁状态 + 证据展开 |
| 项目 | `_binding.json` + 各 run | 矩阵视图：每个项目当前阶段 / 完成度 / 门禁阻塞数 |

刷新：写操作后本地立即重拉；跨会话/跨标签用事件桥（2s）或 `ctx.sse.register`。

## 14.7 全局库载体（P0 待验证，二选一 + 混合）

| 路径 | 手段 | 证据 / 风险 |
|---|---|---|
| **A 设置项** | `ctx.setSettings('tool-flow', { globalFlows: {…} })` → `core.Settings.PluginSettings` | **已确证**：跨工作区、跨重启（`internal/server/handler/workspace.go:193-202` 的 `pluginSettings` 命名空间；前端由 `core.PluginSettingSchemas` 渲染）。缺点：定义混在 `settings.json` 里，不便 diff/评审 |
| **B 安装目录文件** | `<InstallDir>/.pair/flowlib/*.json`，经 `ctx.bash` 绝对路径或宿主存档执行器写 | 文件化、天然可导出/评审/版本管理；**未验证**（`ctx.fs` 限工作区越界拦截；`ctx.bash` 路径限制与是否有通用写执行器需实测） |
| **C 混合（目标态）** | 绑定与索引存设置（A），流程定义存文件（B） | MVP 可先全 A，B 验证通过后平滑迁移 |

> 若 B 不可行，回退 A 属于**完全零风险**方案（功能不减，仅存储形态不同）。

## 14.8 实施计划（增量，接 §11）

**P0 追加验证项**
1. 全局写入路径定论（§14.7 A/B）；
2. `GET /api/workspace` 只读消费项目列表（多根工作区语义）；
3. `_binding.json` 读写 + 多根绑定回归测试。

**P1 追加**
4. `flow_export` / `flow_import`（含 `dryRun` 报告与冲突策略）；
5. `flow_attach` / `flow_detach` / `flow_bindings` + 看板「项目 × 流程矩阵」页签。

**P3 追加**
6. 三层解析优先级 + `flow_clone` / `flow_sync`（显式、不回冲）。

**P4 追加**
7. 通知（overlay/toast）、`/flow*` 命令、市场发布（可选，元数据已预留）。

## 14.9 新决策点（**已全部定稿 → 见 §14.10**）

- ~~**D6 全局库载体**：A 设置项 / B 安装目录文件 / C 混合~~ → **定 B（+ D11 半自动退路）**
- ~~**D7 导出格式**~~ → **定：目录 bundle**
- ~~**D8 项目管理入口**~~ → **定：嵌进 FileExplorer（前端壳改造）**
- ~~**D9 市场发布**~~ → **定：对齐 toolset 市场（可一键发布/安装）**

---

## 14.10 决策定稿（2026-09，用户逐项确认）

| 决策 | 结论 | 说明 |
|---|---|---|
| **D3 预设粒度** | **两者并存** | 项目类型预设定**默认流程**；任务类型预设**可临时覆盖**（会话/任务级 `override`，不改项目绑定） |
| **D4 自定义入口** | **四入口全要** | ① `flow_clone` 从预设改（主路径）② 对话生成（agent 写 `flow_define`）③ 直接编辑 `.pair/flows/*.json`（**文件监听热加载**）④ 看板内表单可视化编辑（阶段/门禁字段） |
| **D6 全局库载体** | **B：安装目录文件** `<InstallDir>/.pair/flowlib/` | 文件化、可 diff/评审；**写权限待 POC** |
| **D7 导出格式** | **目录 bundle** | `flow.json` + `gates/` + `templates/` + `README.md` 分文件（§14.11） |
| **D8 项目管理入口** | **嵌进 FileExplorer 工作区列表** | 体验优先；**「核心零改动」在此有界放宽**（§14.12），仅改前端壳、不碰 `internal/` |
| **D9 市场发布** | **对齐 toolset 市场（一键发布/安装）** | 元数据 `author/tags/repo/version` 同构；`kind: "paircode.flow"`（§14.13） |
| **D10 FileExplorer 改造范围** | **完整** | 列表标签 + 右键菜单 + **流程详情/编辑面板** |
| **D11 若写不了安装目录** | **半自动降级** | 插件导出 bundle 到工作区 → 用户手工拷入 `flowlib/` → 插件只读消费；**读权限同属 P0 验证项** |

**推进方式（用户定）**：先把设计补齐（本 §14 完成）→ 再开工（P0 接线与权限验证起）。

## 14.11 目录 bundle 契约（D7 定稿）

导出产物（`flow_export { id, scope?, to? }`）：

```
flow-<id>/
  flow.json        # FlowDef 真相：阶段 / 门禁(gate) / 证据要求 / 适用项目类型+任务类型 / version
  README.md        # 人读说明：流程目的、阶段含义、门禁如何过关、适用场景
  gates/           # 门禁校验脚本（如 check-test.sh / check-build.ps1 / lint.mjs）
  templates/       # 证据模板 / 文档模板（PR 描述、评审清单、发布检查表）
```

`flow.json` 内的库元数据（`library`）承载市场与版本信息：

```jsonc
{
  "schemaVersion": 1,
  "id": "go-backend-feature",
  "version": 2,
  "projectTypes": ["go-backend"],      // D3：定默认
  "taskTypes": ["feature", "refactor"], // D3：临时覆盖用
  "stages": [ /* … §5.1 … */ ],
  "library": { "author": "…", "tags": ["go","backend"], "repo": "github:owner/repo" }
}
```

导入：`flow_import { dir | zip | json, scope, onConflict, dryRun }` —— **本地交换用目录 bundle**；**市场分发用 zip/单包**（由发布步骤压制，不改变本地导出形态，§14.13）。缺 `flow.json` → 明确报错；`schemaVersion` 不符 → 拒绝。

## 14.12 前端壳改造边界（D8 + D10 定稿）

**已确证的改造点**（`plugins-src/ui-app/src/components/FileExplorer.vue`，703 行）：

| # | 位置 | 改动 |
|---|---|---|
| 1 | 工作区列表项（L13–25 `wsItems`） | 追加「流程标签」：当前流程名 + 阶段 `3/7`，未绑定显示「未接入流程」 |
| 2 | 右键菜单 `showWsContextMenu`（L248–313，items L253–262） | 加 `指定流程…` / `切换流程…` / `查看流程进度`；`switch` 内加对应 `case` |
| 3 | 流程详情/编辑面板（D10「完整」） | 面板**由插件视图承载**（`ui.registerView('flow-board')`），壳内只加「打开面板」动作 → 壳内不放业务逻辑 |
| 4 | 数据来源 | 一律走插件 API：`GET/POST /api/flow/*`（`ctx.webServer.register`）——**壳只展示与触发** |

**边界纪律（必须遵守）**：

- 只改 `plugins-src/ui-app/src/components/FileExplorer.vue`（+ 必要时 1 个新组件）；**不改 `internal/` 任何文件**。
- 壳内**零业务逻辑**：不解析流程 JSON、不判断门禁、不写数据文件 —— 全部转发给插件 API，避免业务污染框架前端。
- 改完须重建前端产物（`scripts/build-ui.mjs`）并做页面验证（标签渲染 / 右键菜单 / 空态）。

> ⚠️ 诚实标注：D8/D10 使「核心零改动」**有界放宽**（前端壳 UI 层）。备选零改动方案（插件视图矩阵）保留在 §14.4，若后续希望完全零改动可回退。

## 14.13 市场对接（D9 定稿）

**已确证的市场机制**（`.pair/plugins/marketplace/index.js`，516 行）：三通道 `ctx.market.register` ——

| 通道 | 检索源 | 安装方式 |
|---|---|---|
| `mcp` | npm registry（`text=` 搜索） | `ctx.mcp` |
| `skill` | GitHub 仓库（`keywords:paircode`） | `ctx.skill` |
| `plugin` | npm registry（keywords `paircode`） | `ctx.npm` |
| `toolset`（既有先例） | 发布 JSON（含 `repository` 字段，`ExportToolsetJSON(ts, tags, author, repo)`） | `ctx.toolset.save` |

**流程包对齐策略**：

1. **元数据同构**：`author / tags / repo / version` 与 `toolset_export` 完全一致（用户已备的心智与工具链可复用）；
2. **发布载体**：目录 bundle 压成单包（zip/JSON）→ 走 `skill`（GitHub 仓库）或 `plugin`（npm）通道分发，与工具集当前做法保持一致；
3. **安装端**：由流程插件自实现（`flow_import` 消费单包 → 落 `scope` 指定库），不改市场核心；
4. **原生「流程」市场分类**（可选增强）：市场 UI 增加 `kind: flow` 属**插件层改动**（marketplace 插件），不涉及 `internal/` —— 若需要再排期。

> ✅ **落地状态（2026-10-16 监督轮补齐）**：第 1 条「元数据同构」与第 3 条「安装端由流程插件自实现」**已落地** ——
> `flow_publish` 产出单 JSON 发布包（`kind:"paircode.flow"` + `spec` + `author/tags/repo/version` +
> `stats` + `checksum`），`flow_import` 以 `file` / `json` 消费发布包（校验 `kind` 与 `checksum`）并回导到指定层，
> `dryRun:true` 报告冲突（`action` / 现有版本 / 导入版本 / 原因 / 元数据）；`/flow-import` 亦支持发布包文件路径。
> **仍属后续**：第 2 条「推送外部市场」（走市场插件 / npm / GitHub 通道，需外部发布动作与账号）、
> 第 4 条「市场 UI 原生 `flow` 分类」（marketplace 插件层改动）。

## 14.14 实施计划（终版增量，接 §11）

| 阶段 | 内容 | 状态（2026-10-16） |
|---|---|---|
| **P0**（先验证） | ① **安装目录读写 POC**：`<InstallDir>/.pair/flowlib/` 能否经 `ctx.bash` / 宿主执行器**读 + 写**（读不通 → 触发 D11 半自动或退 A）② `update_tasks` 接线（直写 `.pair/tasks/*.json` 优先）③ `GET /api/workspace` 多根消费 + `_binding.json` 多根绑定 | ✅ §14.15/§14.17（结论：走 `ctx.install`，非 bash） |
| **P1** | `flow_list/show/define/validate` + `flow_start/status/advance` + 看板最小闭环 + **bundle 导出/导入**（含 `dryRun`） | ✅ 落地（§14.19） |
| **P2** | `flow_attach/detach/bindings` + **FileExplorer 改造**（§14.12 三改动点）+ 项目×流程矩阵 | ✅ 落地（§14.19） |
| **P3** | 三层解析优先级（D3 项目类型默认 + 任务类型覆盖）+ `flow_clone` / `flow_sync` + JSON 热加载 | ✅ 落地（热加载以「实时读盘」等价达成） |
| **P4** | 看板内表单编辑（D4-④）+ 通知（overlay/toast）+ `/flow*` 命令 + 市场发布（D9） | ✅ **全部落地**（字段级表单与 D9 本地发布包见 §14.20；外部市场推送与市场 UI 分类留后续） |

---

## 14.15 P0 验证结论（2026-10-16 实测；探针插件与残留**已全部清理**）

用临时 cordis 探针（`p0-probe2` / `p0-probe3`，装载于本机运行实例）实测三项接线，结论如下：

| 验证项 | 结论 | 实测证据 |
|---|---|---|
| 安装目录**读**（ctx.fs） | ❌ **越界拦截（读也拦）** | `ctx.fs.readFile('E:/Program Files (x86)/PairCode/.pair/plugins/marketplace/package.json')` → `路径 "…" 超出工作区范围（root: F:\syproject\gou-ide）`；`ctx.fs.exists(<安装目录>)` → `false` |
| 安装目录**读+写**（ctx.bash） | ✅ **完全可行** | `mkdir -p "/e/Program Files (x86)/PairCode/.pair/flowlib" && printf "P0PROBE\n" > …/__poc.txt` → `EXIT=0`；回读 `P0PROBE`；`ls -la` 确认 8 字节文件落盘 |
| update_tasks 接线 | ✅ 可用 | `ctx.hostTool.names()` 含 `update_tasks`；`ctx.hostTool.meta('update_tasks')` 返回完整 schema（`tasks` 必填、`subject` 必填、`readOnly:false`） |
| 宿主 API 消费（插件 host 半） | ✅ 可用 | `ctx.web.fetch('http://127.0.0.1:9090/api/workspace')` → `ok:true, status:200`，body `{folders:["F:\\syproject\\gou-ide","F:\\syproject\\ref"], loaded:true, root:"F:\\syproject\\gou-ide"}`；`/api/tasks` 同样 200 |
| 多根工作区真相源 | ✅ 双通道 | ① 上述 `/api/workspace`；② `<InstallDir>/config/settings.json` 的 `workspaceFolders` / `workspaceFolderLists`（bash 可读） |

**结论：D6 = B 可行（全局库 `<InstallDir>/.pair/flowlib/` 经 ctx.bash 绝对路径读写）→ 无需启用 D11 半自动降级。**

### 由此确定的三条实现约束

1. **跨工作区访问一律走 `ctx.bash`**：`<InstallDir>` = `E:\Program Files (x86)\PairCode`（**当前用户对其可写**，Program Files 未拦）；`ctx.fs` 越界是**读写双向**的，故全局库与安装目录配置必须封装 `globalFs` 助手（bash 实现 `mkdir / read / write / list / rm`，路径 UTF-8 + 空格引号转义）。
2. **插件 define 即落盘** `<InstallDir>/.pair/plugins/<name>/`（对所有工作区生效、重启自动装配）→ 临时探针必须 `undefine` **并** `rm` 包目录。本轮已清理：`p0-probe*` 三个包、`<InstallDir>/.pair/flowlib`、工作区 `bin/.pair/flowlib`、`.pair/_p0probe*.json`。
3. **任务数据字段名不一致（坑）**：`GET /api/tasks` 返回 `{taskId, step, status, description, created_at}`，而 `update_tasks` 入参是 `subject`/`status` —— 流程插件读写任务时**必须做字段映射**（`step ↔ subject`），否则回写会丢标题。

**P1 启动条件已满足**（`flow_*` 工具 + 看板最小闭环 + bundle 导出导入）。

---

## 14.17 安装目录能力现状：核心**已有**正规服务（实测 2026-10-16；探针已清理）

> 起因：用户指出「核心应该提供安装目录相关能力，比如插件 / MCP / Skill 不就可以么」。
> 实测结论：**用户判断正确** —— 核心确有这类服务，但**全部是「具名资源」形态**，**无通用文件库服务**。

### 14.17.1 插件 ctx 服务面全表（`inject: [...]` 声明即得，实测 v5）

| 服务 | 方法（实测 keys） | 落盘位置 |
|---|---|---|
| `ctx.toolset` | `list` / `save` / `remove` / `install` | **`<InstallDir>/.pair/toolsets/<name>.json`**（实测 list 返回 6 个预置集：计划讨论/全栈开发/办公/调试/基础/全功能）|
| `ctx.mcp` | `list` / `upsert` / `remove` / `setEnabled` | `level:'user'` → **`<InstallDir>/.pair/mcp.json`**（实测 2 条：live-rig、photoshop-mcp）；`'project'` → `<ws>/.pair/mcp.json` |
| `ctx.skill` | `list` / `write` / `remove` | **仅工作区级** `<ws>/.pair/skills/`（实测 list 返回 10 个，含 system 层）← **缺 `scope`** |
| `ctx.npm` | `install` / `uninstall` / `installed` / `checkUpdates` / `update` | **`<InstallDir>/.pair/plugins/<pkg>/`** |
| `ctx.plugins` | `reloadDisk` | 重扫磁盘插件包 |
| `ctx.market` | `register` / `unregister` / `list` | 市场注册面 |
| `ctx.kernel` | `routes` / `installed` / `total` / `install` | 内核 REST 路由 |
| `ctx.commands` | `register` / `list` / `run` | slash 命令面 |
| `ctx.agents` | `ready` / `followup` | 会话唤醒 |
| `ctx.hostTool` | `exec` / `meta` / `names` | 40 个宿主执行器（含 `skill_write`、`mcp_add`；**无** `toolset_import` / `cordis`）|
| `ctx.http` | `register` / `list` | 插件自注册 REST（与宿主 `/api/*` 并存，插件路由优先拦截）|
| `ctx.webServer` | `register` | Node 风格路由注册（新形态）|
| `ctx.binary` | `dir` / `exec` | `<InstallDir>/bin/` |
| `ctx.process` | `runBackground` / `readOutput` / `runCommand` / `writeStdin` / `kill` / `list` / `exec` | — |
| `ctx.app` | `workspaceRoot` / `root` / `folders` / `projectName` / **`installDir`** / `configDir` / `recentProjects` / `workspaceFolders` / `settings` | `installDir` 实测 = `E:\Program Files (x86)\PairCode`（**只读路径**）|

### 14.17.2 关键机制与坑（写插件必知）

1. **服务按 `inject` 声明才注入**（`jsplugin.go:1922-1961` 的 `switch s { case "mcp": … }`）；未声明则 `ctx.xxx === undefined`。
2. **`ctx.get(name)` 不能用来判注入服务**（只对 `ctx.provide` 注册的动态服务有效）——实测把全部服务名传进去一律返回 undefined，**会误判「核心没这个能力」**（本设计文档初判即被此坑误导，后经 v5 纠正）。
3. **`cordis(op=services)` 的服务表不完整**（缺 toolset/mcp/skill/npm/market/http/webServer/binary/process/kernel/commands/agents）；完整名单见 `js_native_guard.go:80-84`。
4. `ctx.toolset` 有**两套形态**：无条件设置的「模板注册面」（仅 `registerTemplate`，`jsplugin.go:1644`）与 `inject:['toolset']` 时**覆盖**的「工具集管理面」（`list/save/remove/install`，`:1948`）。

### 14.17.3 对 flowlib（`<InstallDir>/.pair/flowlib/`）的适配结论

上述服务**无一可承载自定义 JSON 库**：

- `ctx.toolset.save` 只存 `Toolset{name,description,plugins[]}`（自定义字段被裁），且**立即触发 `installToolset`**、会出现在工具集/场景选择器 → 语义污染；
- `ctx.mcp.upsert` 只存 MCP 条目（command/args/enabled）；`ctx.npm.install` 需真实 npm 包；`ctx.skill.write` 只写 SKILL.md。

→ 四条路（**R1 待用户拍板**）：

| 方案 | 核心改动 | 代价 |
|---|---|---|
| **R1（推荐）** 新增 `ctx.install` 通用安装目录文件服务（签名对齐 `ctx.fs`：readFile/readFileBase64/writeFile/writeFileBase64/exists/readdir/stat/mkdir/rm；根 `<InstallDir>`，白名单限 `.pair/**`） | 1 个 builder（照 `buildFSService`，`jsplugin.go:1987+`）+ inject switch 1 case + `js_native_guardServices` 白名单 + 技能文档全表 | 需编译 + 独立二进制/非默认端口验证；**一次解决所有插件的全局库需求** |
| R2 复用 `ctx.toolset.save` 承载流程库 | 零 | 污染工具集/场景面 + 触发装载 + 字段被裁 → **不可取** |
| R3 降级 `ctx.setSettings`（PluginSettings，跨工作区/重启） | 零 | 只能存单个配置对象，无法多文件库 + 目录 bundle 导入导出 → 能力降级（可留给「当前激活流程」索引）|
| R4 `ctx.bash` 绕行 | 零 | 非正规通道（用户已否决）|

**伴生建议**：`ctx.skill.write` 补 `scope:'global'`（对齐 agent 工具 `skill_write`），否则插件无法安装**全局**技能。

---

## 14.18 R1 实施记录：`ctx.install` 安装目录数据区文件服务（2026-10-16 已落地）

> 用户拍板 R1。本节是**实施方案 + 实测证据 + 踩坑**的定稿记录；§14.17 保留为调研留档。
> 结论：**D6=B（全局流程库落 `<InstallDir>/.pair/flowlib/`）的正规通道 = `ctx.install`**，
> `ctx.bash` 绝对路径绕行方案作废（不再需要）。

### 14.18.1 改动清单

| 文件 | 改动 |
|---|---|
| `internal/agent/jsplugin.go` | ① 新增 `buildInstallService()`（+241 行，插在 `buildFSService` 与 `buildWebService` 之间）② `buildContextObject` 的 inject switch 增 `case "install"` ③ **`hasService` 与 `availableServices` 白名单增 `install`**（★ 见 14.18.4 坑）④ 新增 `skillDirForCtxScope()`，`ctx.skill.write/remove` 支持 `scope:'global'` |
| `internal/agent/js_native_guard.go` | `jsNativeGuardServices` 增 `"install"`（文件 IO 需原生调用感知，避免长 IO 被误判死循环）|
| `.pair/plugins/marketplace/index.js` | 技能安装跟随 scope（`user→global`/`project→工作区`）、卸载覆盖全局+工作区两处 |
| `.pair/skills/cordis-plugin-development/SKILL.md`、`config/skills/.../SKILL.md` | ctx 能力全表增 `install`（服务数 25→26）+ 用法样例 + 「新服务必须登记白名单」告警 |

### 14.18.2 `ctx.install` API 契约

**根与白名单**：根 = `<InstallDir>/`（`core.InstallDir()` 实时解析）；可访问区 **仅 `<InstallDir>/.pair/**`**（读写同限）。
安装根下的程序本体、`config/`、日志、exe 同级路径一律拒绝；`.pair` 数据区根目录本身不可写/删。

**路径语义**：相对路径**相对安装根**（与 `ctx.fs` 相对工作区根一致）——`'.pair/flowlib/a.json'` → `<InstallDir>/.pair/flowlib/a.json`。

**方法（13 个）**：

| 方法 | 返回 / 说明 |
|---|---|
| `root()` | `<InstallDir>/.pair`（数据区根绝对路径，插件拼路径用）|
| `exists(path)` | bool；**越界返 false**（与 ctx.fs 同语义）|
| `readFile` / `readFileBase64` | string / base64 string |
| `writeFile` / `writeFileBase64` | 写入（**自动建父目录**，与 ctx.fs 的 writeFile 不同）|
| `appendFile(path, text)` | 追加（自动建父目录）|
| `readdir(path)` | `[name, …]` |
| `list(path)` | `[{name, path, isDir, size, mtime}]`（path 相对数据区根；列表页免二次 stat）|
| `stat(path)` | `{name, size, isDir, mtime}` |
| `mkdir(path, recursive?)` / `rm(path, recursive?)` / `rename(from, to)` | — |

### 14.18.3 实测证据（独立二进制 + 隔离 InstallDir + 端口 9095，探针已清理）

环境隔离手法：exe 放 `_verify_install/companion_test.exe`（InstallDir 随之隔离到该目录，**不碰**仓库 `.pair` 真源与安装目录），`WEB_PORT=9095`（9090 宿主全程未动）。

**探针（磁盘插件包，inject `['install','skill','app','bash','fs']`）结果 —— 全项通过：**

```
① typeof.install=object；13 方法齐全；root()=…\_verify_install\.pair；app.installDir 一致
② writeFile/readFile/exists/stat/list/readdir/mkdir + 父目录自动创建 → 全 OK（flowlib 场景）
   list('.pair/flowlib') => [{"path":"flowlib/p0-flow.json","size":34,"isDir":false,…}]（path 相对数据区）
③ base64 往返 / appendFile / rename / rm(file) / rm(dir,true) → 全 OK
④ 安全边界（全部被拒）：
   写 config/p0.json、写安装根 p0.txt、读 companion_test.exe、写工作区绝对路径、rm('.pair',true)、readdir('.')
   → ctx.install: 路径 "…" 超出安装数据区（只允许 <InstallDir>\.pair 内的路径；安装根下程序本体与 config/ 不可访问）
   exists 越界 → false
⑤ skill.write scope:'global' → 「已安装技能 p0-probe-global（全局，跨工作区）」+ 落盘 <InstallDir>/.pair/skills/…/SKILL.md 确认
   skill.remove(name,'global') / remove({name,scope}) → 「已删除技能 …（全局，跨工作区）」，落盘复核已清
```

**marketplace 端到端（真插件、真 HTTP 接口）：**

```
POST /api/marketplace/install {"kind":"skill","id":"p0-mkt-global","scope":"user"}
  → ✅ 已安装技能「p0-mkt-global」（用户级（全局，跨工作区））+ 落盘 <InstallDir>/.pair/skills/p0-mkt-global/SKILL.md
POST /api/marketplace/install {"kind":"skill","id":"p0-mkt-project","scope":"project"}
  → ✅ 已安装技能「p0-mkt-project」（工作区级）   ← 未打开工作区时按核心 skillTargetDir 同语义回落
POST /api/marketplace/uninstall {"kind":"skill","id":"p0-mkt-global"} → 已卸载，落盘复核已清
（修复前：两条 install 都写工作区级、消息恒为「工作区级 .pair/skills」）
```

### 14.18.4 ★ 关键坑：新服务漏登记 `hasService` → 插件进 waiting（静默不执行）

**现象**：探针插件包已就位、启动日志无任何错误，但 `apply` 从不执行、无诊断、无日志（`ph.List()` 也看不到），排查一度怀疑磁盘装载机制。

**根因**：`inject` 是「apply 前**校验宿主是否提供**的硬依赖」（`jsPluginDef.inject` 注释）；判定用 `hasService(name)`（`jsplugin.go:3890`）——它是**硬编码白名单**，`install` 不在其中 → `missingServices` 返回 `['install']` → 插件登记为 **waiting**。而 **waiting 不是错误**，故无日志。

**修复**：`install` 同时登记 `hasService` 与 `availableServices`（各 +1 项）→ 探针立刻全项通过（前后对比即双重证据）。

**纪律**：给核心加任何新 ctx 服务，必须**三处同时登记** —— ① `buildContextObject` 的 inject switch ② `hasService`/`availableServices` ③ `jsNativeGuardServices`。

### 14.18.5 现有插件排查结论（无需更新者已逐项确认）

| 范围 | 结论 |
|---|---|
| `.pair/plugins/*`（21 个插件真源） | 无一使用 `ctx.app.installDir`；`ctx.bash` 仅用于 git/命令编排（git-api、fs-api 的 mv、tool-bug、tool-harness、ui-quick-exec）——**无绕行写安装目录者** |
| `plugins-src/**` | 命中项均为文档/UI 文案（MarketplacePanel、changelog、faq）与 Go 侧 PATH 探测（run_code.go、officetools.go）——无绕行 |
| `plugins-dist/**`（创作域 6 包） | 零命中 |
| 安装目录实际运行插件（36 包） | 零 `installDir`/`flowlib` 用法；`ctx.bash` 同限于命令编排 |
| `.pair/plugins/marketplace` | ✅ **已更新**（技能 scope 跟随 + 卸载双清，见 14.18.1） |
| `.pair/plugins/tool-workflow` | 无需改：它是「workflow 脚本编排执行器」（inject 仅 logger，execute 路由宿主 `ctx.hostTool`），不承载流程库落盘 |
| `.pair/plugins/autopilot` / `tool-memory` | 无需改：落盘 `.pair/autopilot/`、`.pair/memory/` 属**工作区级**语义（按会话/项目隔离），非全局库 |

### 14.18.6 对 flowlib（P1）的接口约定

```js
// 全局流程库：<InstallDir>/.pair/flowlib/<flowId>.json
ctx.install.writeFile('.pair/flowlib/' + id + '.json', JSON.stringify(def))
const ids = ctx.install.readdir('.pair/flowlib').filter(n => n.endsWith('.json'))
const def = JSON.parse(ctx.install.readFile('.pair/flowlib/' + id + '.json'))
// 目录 bundle 导入导出：readdir + readFile / writeFile（父目录自动建）
```

> 插件必须 `inject: ['install']`；读安装目录**不要**再用 `ctx.bash`。

---

## 14.19 tool-flow 实施记录：P1–P3 落地 + P2/P4 补齐（2026-10-16）

> 触发：用户「流程相关实现都落地了么 没落地继续落地」。本文档是落地判据 —— 逐项对照结果如下，
> **凡未落地者如实标注**（不粉饰）。

### 14.19.1 落地清单（对照 §11 / §14.14）

| 排期项 | 状态 | 落点 |
|---|---|---|
| P1 插件包 `tool-flow` host 半（`flow_list/show/define/validate/start/status/advance` …） | ✅ | `.pair/plugins/tool-flow/index.js`（**17 个工具**，含 `flow_publish`） |
| P1 `/api/flow/state` 只读接口 | ✅ | `ctx.http.register('GET','/api/flow/state')` |
| P1 证据留档（`.json` + `.log`） | ✅ | `.pair/flow-runs/evidence/<runId>/<gateId>.{json,log}` |
| P1 看板最小闭环 | ✅ | `.pair/plugins/tool-flow/client.js`（`registerView('flow-board')`，三页签） |
| P1 bundle 导出/导入（`dryRun` / `onConflict`） | ✅ | `flow_export` / `flow_import`（`flow.json` + `README.md` + `gates/` + `templates/`） |
| P2 `flow_attach/detach/bindings` + 项目×流程矩阵 | ✅ | `_binding.json` + 看板「项目 × 流程」页签 |
| P2 FileExplorer 改造（§14.12 三改动点） | ✅ | `plugins-src/ui-app/src/components/FileExplorer.vue`（列表标签 + 右键三项 + 打开面板） |
| P3 三层解析优先级 + `flow_clone` / `flow_sync` | ✅ | `project > global > preset`；同步默认不回冲 |
| P3 JSON 热加载 | ✅（等价） | 每次调用实时读盘、无内存缓存 → 编辑 `.pair/flows/*.json` 后即刻生效，无需重启 |
| P4 `/flow`、`/flow-board`、`/flow-import` 命令 | ✅ | `ctx.commands.register`（`inject:['commands']`） |
| P4 通知（阶段变更 / 操作结果） | ✅ | client 半 `window.$toast`，不可用时兜底消息条 |
| P4 看板内编辑（D4-④） | ✅ | RPC `show/validate/define/clone` + 看板编辑面板（校验 + 乐观锁） |
| P4 字段级表单编辑（阶段/门禁逐字段） | ✅（2026-10-16 监督轮补齐） | `client.js`「表单 / JSON」双模式（默认表单）+「＋ 新建流程」纯表单入口 + kind 收敛字段；详见 §14.20 |
| P4 市场发布（D9） | ✅ **本地能力**（推送外部市场留后续） | `flow_publish` 单 JSON 发布包 + `flow_import` 回导 + `dryRun` 冲突报告；详见 §14.20 |

### 14.19.2 数据落点与预设

预设 **8 类 / 38 个门禁**：`go-backend-feature`、`web-frontend-feature`、`python-feature`、`docs-office`、
`lib-cli`、`plugin-dev`（本 IDE 自举流程）、`debug-troubleshoot`、`generic`。

三层：`preset`（插件包 `assets/flows/`，只读）< `global`（`<InstallDir>/.pair/flowlib/`，`ctx.install`）
< `project`（`<ws>/.pair/flows/`，`ctx.fs`）；同名上层覆盖，升级不回冲。

### 14.19.3 ★ 实测出的三处「设计—实现」落差（诚实标注）

1. **`Stop` 钩子无法硬拦会话收尾**：核心 `hook.IsBlocking` 只对 `PreToolUse` / `UserPromptSubmit` 返回 true
   （`internal/hook/hook.go:65`）→ §10.1 的「Stop 拦截未过门禁的收尾」**在当前核心不可实现**（需改核心，
   超出零改动约束）。落地替代 = `systemPrompt.variable` 每轮注入阶段与未过门禁 + `PreToolUse` 拦破坏性操作
   + 工具返回文本带进度摘要。**降级影响 + 用户可选（改核心 vs 接受替代）已正式定案写入 §10.1**。
2. **`section.text` 是静态字符串**（`jsplugin.go:1526`，注册时求值一次）→ 动态注入必须用
   `ctx.systemPrompt.variable({name, provider})`（组装时求值）。已实现为变量 `paircode_flow_state`。
3. **系统提示 provider 组装期求值、无会话上下文** → 提示按**项目级**（最近活跃 run）聚合；
   多会话并行时以显式传 `convId` 的工具调用 / RPC 为准（`FlowRun.convId` 已落盘）。

### 14.19.4 实测证据（三层）

**① 逻辑层**（`.pair/plugins/tool-flow/smoke-test.js`：mock ctx + 真实文件系统 + 真实 bash 执行；
原 `_verifyflow/mock-run.js` 已迁入插件包随包分发）：**148 项断言全通过**（2026-10-16 监督轮后）——
三层解析 / 校验 / 启动 / 四类门禁（check·command·artifact·approval）/ 推进 / 证据留档 / 任务同步 /
乐观锁 / 导出导入 / **发布包（D9）** / 绑定矩阵 / 接口快照 / RPC / 提示注入 / 命令 / 钩子拦截 /
**三档强度（§10.1）** / 安全边界 / **表单编辑器 UI（D4-④）**。
期间修出 **2 处真 bug**：`startRun` 判重误用 `||` 导致多会话并行被误拒；门禁**通过**路径未回传 `gate` 记录。

**② 实例层**（独立 InstallDir + 最新源码构建的独立二进制 + 端口 9095 + 工作区 `_verifyflow/ws2`）：

- `GET /api/flow/state` → **HTTP 200 / `ok:true`**（若 `inject:['install']` 不满足，插件会静默 waiting → 404，
  故 200 本身就是「装载成功」的证据）
- RPC `flow_action`：`start` / `advance` / `attach` / `show` / `clone` / `define` 全部通过；**乐观锁冲突被正确拒绝**
- 落盘核验：`flow-runs/run-*.json`、`evidence/g-recon-1.json`、`tasks/*.json`（阶段 1 三条 completed + 阶段 2 两条 pending，均带 `convId`）、`_binding.json`
- `gitHead` 基线正常记录；非 git 工作区降级为提示（不阻塞）

**③ UI 层**（浏览器 DOM 断言 + 截图）：

- 标签栏出现「流程」tab（`registerView` 生效）；看板渲染进度条/阶段卡片/门禁徽标/证据展开/操作按钮
  （`.ftb-bar` / `.ftb-stage` / `.ftb-gate` 命中）；编辑器面板渲染（JSON + 校验/保存/取消）
- **表单编辑器（D4-④，2026-10-16 监督轮补）**：用最小 DOM 桩驱动**真实 `client.js`**
  （`.pair/plugins/tool-flow/ui-form-test.js`）断言 29 项 —— 默认表单模式 / 纯表单新建 2 阶段 3 门禁并落盘 /
  增删阶段与门禁、上下移 / kind 收敛与缺字段被拦 / 表单↔JSON 双向切换不丢字段 / preset 只读入口
- FileExplorer：工作区条目显示流程标签 `go-backend-feature · 2/5 实施修改`（悬浮提示含「1 个必过门禁阻塞」）；
  右键菜单逐字含「指定流程… / 切换流程… / 查看流程进度」

### 14.19.5 启用方式（部署后）

1. **插件随包分发**：`.pair/plugins/tool-flow/`（已同步 `bin/.pair/plugins/` 安装镜像与发布包）；
2. **agent 可见性**：在「工具集」面板勾选 `flow_*` 工具，或导入随包提供的
   `assets/toolset-flow.json`（「工程流程」工具集，含 tool-flow + harness/exec + 内置系统工具）；
3. **零配置可用**：项目根含 `go.mod` / `package.json` / `.pair/plugins` 等特征时自动识别项目类型并推荐流程；
   未接入流程不影响正常对话（提示注入仅在已接入或可推荐时出现）。

> ⚠️ 部署注意：本插件 `inject: ['install']` 依赖 **v1.6.17 及以后**的核心（`ctx.install`）。装到更旧的核心上会
> 静默进入 waiting（无报错、无日志、接口 404）—— 排查时先核对 `/api/system/info` 的 `version`。

> ⚠️ **部署前必做（2026-10-16 踩坑，监督轮补）**：`release/PairCode` 是 packager 的**产物快照**，不随仓库改动自动更新。
> 改了 `.pair/plugins/`（本次的 tool-flow / ui-sidebar）后**必须重跑完整 `./packager.exe`**（10 步全跑完，含最后的 `package`），
> 否则 `scripts/overwrite-install.mjs`（默认源 `--src release/PairCode`）装进去的还是旧包 —— 表现为「安装目录 `.pair/plugins/`
> 里没有 tool-flow、ui-sidebar 是旧版无流程入口」，覆盖安装后流程功能**不会出现**。
> 就绪自查两行：`ls release/PairCode/.pair/plugins/tool-flow/`（应有 index.js / client.js / assets/flows 8 个预设）
> 与 `grep -c 未接入流程 release/PairCode/.pair/plugins/ui-sidebar/assets/ui-sidebar.js`（应非 0）。

---

## 14.20 监督轮补齐：D4-④ 表单编辑 / D9 本地发布包 / §10.1 三档定案（2026-10-16）

> 触发：监督者第 1 次监督意见 —— 已定稿的「D4-④ 字段级表单编辑」「D9 市场发布」未落地，
> 且 §10.1 三档强度缺可复现断言。本轮**只做这三件事**，不重做已达标项（P1–P3、FileExplorer、
> 命令面 / 通知 / JSON 编辑）。

### 14.20.1 落地内容

| 项 | 落地要点 | 落点 |
|---|---|---|
| **D4-④ 表单编辑** | 「表单 / JSON」双模式、**默认表单**；阶段（id / 标题 / 目标 / 任务）+ 上下移 + 增删；门禁（id / kind / 说明 / 必过 / 阻塞）+ 增删；**kind 收敛字段**（`command`→命令 `cmd`、`artifact`→产物路径 `artifact`、`approval`→审批提示语）；本地同规则校验（必填 / 唯一 / kind 相关）→ `flow_validate` → `flow_define`（乐观锁） | `client.js`：`renderForm` / `renderGateForm` / `formErrors` / `setMode` / `currentJson` |
| **D4-④ 新建入口** | 流程库「＋ 新建流程」→ 空白骨架（1 阶段 1 门禁）→ **纯表单**建到 ≥2 阶段 ≥3 门禁并保存；新建时 id 可编辑，项目层同名冲突前置拦截 | `client.js`：`openEditor(id, {blank:true})` / `editorSave` |
| **D4-④ 编辑态保护** | `editor.def` 为唯一真相源（未渲染字段如 `board` / `appliesTo` / `onFail` 原样保留）；**编辑态不因 4s 轮询重绘**（`editor.dirty` 标记），避免打断输入 / 丢焦点 | `client.js`：`touchEditor` / `draw()` |
| **D9 发布包** | `flow_publish`：单 JSON 发布包 `{kind:"paircode.flow", spec, schemaVersion, name/title/version/author/tags/repo/publishedAt, source, stats, flow, checksum}`，元数据形态对齐 `toolset_export`；**发布必须署名**（缺 `author` 拒绝）；发布前必须通过 `flow_validate` | `index.js`：`publishPack` / `sha8` |
| **D9 回导** | `flow_import` 支持 `file`（发布包或裸 JSON 文件）/ `json`（文本）/ `dir`（目录 bundle）；发布包校验 `kind` 与 `checksum`（不符即拒绝，不猜测降级）；`dryRun` 报告 `action / 现有版本 / 导入版本 / 原因 / 元数据` | `index.js`：`importBundle` / `dryRunText` |
| **§10.1 三档** | `enforce` = 拦破坏性操作 + 每轮注入未过门禁；`warn` = 注入但不拦；`off` = **只记录**（不拦、不注入；证据与看板照常留档 —— 为此在 `flowPromptText()` 增加 `off` 短路） | `index.js`：`PreToolUse` / `flowPromptText` / settings hint |

### 14.20.2 验证（可复跑）

```bash
node .pair/plugins/tool-flow/smoke-test.js      # → 148 通过 / 0 失败
```

| 断言组 | 条数 | 关键输出 |
|---|---|---|
| ⑦b 发布包（D9，host 半） | 14 | `已生成发布包：.pair/flow-exports/go-backend-feature.publish.json`；`dryRun（未落盘）：发现 1 处冲突`；`已导入流程 go-backend-feature → global`；checksum 不符 / kind 不符 / 缺 author 均被拒 |
| ⑧b 三档强度（§10.1） | 8 | `[enforce] 破坏性操作被拦`、`[enforce] 每轮注入未过门禁 + 下一步动作`、`[warn] 不拦破坏性操作`、`[warn] 仍每轮提示未过门禁`、`[off] 不拦破坏性操作`、`[off] 不注入流程提示`、`[off] 门禁结果照常落盘`、`[off] 证据文件仍写入 evidence 目录` |
| ⑩ 表单编辑器 UI（D4-④） | 29 | `纯表单新建保存成功（落盘项目层 .pair/flows/my-flow.json）`、`落盘内容 = 表单输入（2 阶段 / 3 门禁 / command 带 cmd / artifact 带产物路径）`、`双向切换不丢字段：表单未渲染的 board 仍在 JSON 中`、`preset 层行只读：无「编辑」入口，只有「克隆到项目」` |

**UI 断言的机制（为什么可信）**：`.pair/plugins/tool-flow/ui-form-test.js` 用 ~60 行最小 DOM 桩
（`createElement` / `textContent` 赋值清空子节点 / `appendChild` / `addEventListener` + 事件派发）
在 Node 中执行**真实** `client.js`，并通过 `ui.invoke` 调用**真实 host 半** RPC（`flow_action`）、
`ui.http.get` 调用**真实快照接口**（`GET /api/flow/state`）。因此断言覆盖的是真实渲染与真实保存链路，
不是另写一份等价逻辑。定位控件一律按「label 文本 / 阶段 id / 行内流程 id」语义查找（不依赖索引），
避免 DOM 顺序变化造成的假通过。

### 14.20.3 本轮诚实边界

1. **外部市场推送未做**：`flow_publish` 只产出可分享的单 JSON 发布包（本地能力）；推送到 npm / GitHub
   与市场 UI 的 `kind: flow` 分类属市场插件层改动（§14.13 第 2、4 条），需外部账号与发布动作，留后续。
2. **`Stop` 硬拦仍不可实现**（核心限制，定案见 §10.1）：当前替代链路为「每轮注入 + PreToolUse 拦截 + 摘要」。
3. **表单只渲染 流程 / 阶段 / 门禁 三组字段**：`board` / `appliesTo` / `onFail` / `library` 等未渲染字段
   经 JSON 模式编辑，表单不会抹掉它们（已断言）。
