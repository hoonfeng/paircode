# tool-flow —— 工程流程与看板

> 设计真源：`docs/flow-board-plugin-design.md`（§5 数据契约 / §7 预设 / §10 门禁 / §14 扩展）。
> 硬约束：**框架核心零改动** —— 本插件只消费既有 ctx 能力（tools/webServer/systemPrompt/hooks/commands/fs/install）。

## 数据落点（真相全部在磁盘）

| 层 | 路径 | 读 | 写 | 通道 |
|---|---|---|---|---|
| 预设 preset | `<InstallDir>/.pair/plugins/tool-flow/assets/flows/*.json` | ✅ | ❌ 只读 | `ctx.install` |
| 全局 global | `<InstallDir>/.pair/flowlib/*.json` | ✅ | ✅ | `ctx.install` |
| 项目 project | `<工作区>/.pair/flows/*.json` | ✅ | ✅ | `ctx.fs` |
| 项目绑定 | `<工作区>/.pair/flows/_binding.json` | ✅ | ✅ | `ctx.fs` |
| 流程实例 | `<工作区>/.pair/flow-runs/<runId>.json` | ✅ | ✅ | `ctx.fs` |
| 证据留档 | `<工作区>/.pair/flow-runs/evidence/<runId>/<gateId>.{json,log}` | ✅ | ✅ | `ctx.fs` |
| 任务条目 | `<工作区>/.pair/tasks/<id>.json` | ✅ | ✅（复用宿主任务真相） | `ctx.fs` |

解析优先级：**project > global > preset**（同名取上层）；上传更新**不回冲**下游，需显式 `flow_sync`。

## 工具面

管理：`flow_list` / `flow_show` / `flow_define` / `flow_validate` / `flow_clone` / `flow_delete` / `flow_sync` / `flow_export` / `flow_import` / `flow_publish`
接入：`flow_attach` / `flow_detach` / `flow_bindings` / `flow_recommend`
执行：`flow_start` / `flow_status` / `flow_advance`

## 看板（D4-④：表单 / JSON 双模式）

主内容区「流程」tab 三页签（当前流程 / 项目 × 流程 / 流程库）。流程库页签提供：

- **＋ 新建流程**：给空白骨架（1 阶段 1 门禁），**纯表单**即可建到任意规模（阶段可增删、可上下移；
  门禁可增删；门禁按 `kind` 收敛字段 —— `command` 必填命令、`artifact` 必填产物路径、`approval` 须用户批准）；
- **编辑**：表单态（默认）与 JSON 态可**双向切换**且字段不丢 —— 表单未渲染的字段（`board` / `appliesTo` /
  `onFail` / `library` 等）原样保留；
- 保存链路统一：本地表单校验（必填 / 唯一 / kind 相关）→ `flow_validate` → `flow_define`（`expectedVersion` 乐观锁）；
- **预设层只读**：不提供「编辑」入口，只能「克隆到项目」后再编辑。

## 发布包（D9：本地能力）

`flow_publish {id, author?, tags?, repo?, version?, to?}` 产出单 JSON 发布包
（`kind:"paircode.flow"` + `author/tags/repo/version` + `stats` + `checksum`），默认落
`<工作区>/.pair/flow-exports/<id>.publish.json`。回导：`flow_import {file:"<包路径>", scope:"project"|"global"}`，
先加 `dryRun:true` 可看冲突（动作 / 现有版本 / 导入版本 / 原因）。包会校验 `kind` 与 `checksum`，
不符即拒绝（不猜测降级）；发布必须署名（缺 `author` 拒绝）。**推送外部市场**（npm / GitHub）由市场插件通道负责，不在本插件内。

## 接口面（只读，给看板）

- `GET /api/flow/state?convId=&project=` —— 当前流程/阶段/门禁/证据 + 项目矩阵 + 三层库列表

写操作一律走 RPC（client 半 `ui.invoke('tool-flow','flow_action',{...})`），保持「事实数据走持久化接口、写走显式动作」的既有纪律。

## 门禁（「重」强度）

`check` 自评 / `command` 代跑并存证 / `artifact` 产物存在性 / `approval` 审批卡点（必须用户显式批准，插件不得自动通过）。

强度设置三档（设置面板「门禁强度」）：

| 档位 | 行为 |
|---|---|
| `enforce`（默认） | `PreToolUse` **拦**破坏性操作（`rm -rf` / `git reset --hard` / 文件删除）+ 每轮注入未过门禁与下一步 |
| `warn` | 每轮注入未过门禁，但**不拦**任何操作 |
| `off` | **只记录**：不拦、不注入提示；门禁证据与看板展示照常留档 |

> [注意] 诚实限制：核心 `hook.IsBlocking` 仅对 `PreToolUse` / `UserPromptSubmit` 生效，故设计 §10.1 中「`Stop` 钩子硬拦会话收尾」**在当前核心不可实现**（需改核心，超出零改动约束）；本插件以「系统提示每轮注入 + PreToolUse 拦截 + 工具返回摘要」三重软强制替代，并如实标注。
>
> **降级影响**：能可信地阻止**破坏性动作**，但不能阻止「阶段未完成就收尾」（只提示，不阻断）。
> **用户可选**：(a) 改核心（让 `Stop` 进入可阻断集合 `IsBlocking` + 调用点消费 `block`，语义变更影响所有 `Stop` 钩子消费者）；
> (b) 接受替代（当前默认，零核心改动）。详见设计真源 §10.1 定案。

## 自测

```bash
node smoke-test.js        # host 半 119 项 + 表单编辑器 UI 29 项 = 148 通过 / 0 失败
```

`smoke-test.js` 用 mock ctx（真实文件系统 + 真实 bash）驱动插件全部工具 / 接口 / 钩子 / 命令；
`ui-form-test.js` 用最小 DOM 桩在 Node 中驱动**真实 `client.js`**，并走真实 host 半 RPC 与快照接口，
断言表单新建 / 增删 / kind 收敛 / 双向切换 / preset 只读等 UI 行为。
