# agent-teams — 团队任务编排（单 Agent 多步执行）

> ★ 2026-09 架构调整：**去掉成员/角色/子 Agent**。当前会话（队长）是唯一执行者，
> 按任务 DAG 一步步执行；长任务由宿主的**工具调用轮次预算**（默认 120 次/段）
> 自动分段续跑（同会话，历史保留、上下文自动压缩）。
> 团队协作的编排能力（任务 DAG、质量门禁、两阶段批准、Web 面板、归档）全部保留。

一句话：当前会话成为**队长**，把目标拆成带依赖的**任务 DAG**；用户批准后，队长
自己逐任务执行（claim → 干活 → update_task），质量门禁把关评审/修复循环。

## 工作方式

1. 队长创建团队（staged 两阶段：默认**先计划后批准**，用户审 Web 面板后才开始执行）
2. 队长创建任务 DAG（可带依赖：依赖全部 completed 才可领取；质量种类带契约字段）
3. 结束本轮 → 用户在 Web 面板点「批准并运行」（或在对话里明确批准）
4. 队长按 `agent_teams_status` 的**就绪任务**列表逐个执行：
   `claim_task`（取得 attempt_id）→ 用工具实际干活 → `update_task`（携带 attempt_id）
5. 每次 update 返回 `Next ready: …`；长任务达到工具调用预算（120）自动分段续跑
6. 质量门禁：review/requirements 仅 verdict=pass 可完成；needs_revision/reject 必须
   failed + findings，系统自动派生 repair + 复审（受 codeMaxRounds/maxRepairAttempts 限制）
7. 全部终态后向用户汇报 → `agent_teams_delete`（取消未完成任务并归档；等价面板「删除团队」按钮）

## 工具清单（10 个，全部队长可用）

| 工具 | 只读 | 说明 |
| --- | --- | --- |
| agent_teams_create | | 建团队（approval=required 默认两阶段；profile 可 seed 任务模板） |
| agent_teams_edit_plan | | 原子批量修订暂存计划（update_task/add_task/remove_task） |
| agent_teams_approve | | 批准暂存计划并开始执行（不派生任何会话） |
| agent_teams_create_task | | 创建任务（kind 质量契约 + dependencies） |
| agent_teams_delete_task | | 删除已结束任务（自动清理依赖引用） |
| agent_teams_claim_task | | 领取就绪任务（返回 attempt_id 执行凭据） |
| agent_teams_update_task | | 更新状态/输出（校验 attempt_id + 质量门禁） |
| agent_teams_status | ✓ | 全景状态（进度/就绪任务/DAG/覆盖矩阵） |
| agent_teams_resume | | 恢复暂停团队（需非空 reason） |
| agent_teams_delete | | 结束团队（取消未完成任务 + 归档） |

已移除（去成员）：`agent_teams_add_member`、`agent_teams_remove_member`、
`agent_teams_reassign_task`、`agent_teams_send_message`。

## 状态层（磁盘即真相）

```
<workspace>/.agent-teams/<teamId>/
  team.json          团队记录（任务 DAG/phase/halted/escalated/reviewPolicy）
  archive/<teamId>/  删除时归档
```

（旧版本含 `inbox/*.jsonl` 邮箱与 `retired-members.json`；去成员后不再写入，
旧数据留在磁盘不影响读取——读取时忽略 `members` 字段。）

## Web 面板

标题栏「团队」按钮 → 活动浮层（事件驱动刷新 `/api/agent-teams/teams`）：
- 任务进度（完成/总数、就绪数）+ **下一步**（就绪任务提示）
- 任务 DAG（状态徽章 + 依赖弱化 + 点击聚焦上下游 + 终态任务可删）
- staged 团队：批准并运行 / 返回对话修改 / 废弃（归档）
- running 团队：暂停（halt，取消未完成任务）/ 删除团队（deleteTeam，取消未完成任务 + 归档）
- halted 团队：删除团队（deleteTeam，归档后从面板消失）

面板动作经宿主 client 方法（`approve` / `discard` / `continuePlanning` / `halt` /
`deleteTeam` / `deleteTask`；HTTP 侧为 `POST /api/agent-teams/teams/<id>/<action>`，
action=`approve|discard|continue|halt|delete`）。「删除团队」等价队长工具
`agent_teams_delete`：取消未完成任务 → 向队长会话投递停止通知（归档前投递）→
完整记录归档到 `<stateDir>/archive/`，团队从活动面板消失。

批准按钮 → 宿主 client 方法 `approve` → 团队进入 running，并**向队长会话投递**
一条「开始执行」消息（`ctx.agents.followup` 投递到已存在的队长会话；不是派生子 Agent）。

## 宿主能力依赖

- `ctx.agents.followup` —— 仅用于「面板批准后唤醒队长会话」（投递，不派生）
- `ctx.systemPrompt.section` —— 用法协议注入（order 117）
- `ctx.http` —— 面板快照与 approve/discard/continue/halt/delete 路由
- 工具执行注入 `_convID`（调用会话）/`_wsRoot`（会话工作区根）——身份与状态根解析
- **工具调用轮次预算**（`internal/agent/tool_budget.go`）：段结束/自动续跑由宿主负责，
  插件不感知（协议段只提示模型「持续执行，不要因轮次长而提前收尾」）

## 配置（设置 → 团队任务编排）

| 字段 | 默认 | 说明 |
| --- | --- | --- |
| stateDir | `.agent-teams` | 团队状态目录（工作区相对） |
| codeMaxRounds | 3 | 代码审查轮次上限 |
| maxRepairAttempts | 2 | 修复尝试上限 |

## 使用示例

> 使用 AgentTeams 审查 v0.5.3 之后的提交：先出需求/范围，再实施修复，
> 然后独立验证与代码审查，最后输出一份汇总报告。
