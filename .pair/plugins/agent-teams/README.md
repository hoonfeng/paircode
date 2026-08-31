# agent-teams — 多智能体团队

移植自 [dsh-agent-teams](https://github.com/NanmiCoder/dsh-agent-teams)（DeepSeek Harness 生态），
适配本项目宿主（goja 沙箱磁盘插件 + 可续聊子 Agent）。

一句话：当前会话成为**队长**，派生可续聊**成员会话**、把目标拆成带依赖的**任务 DAG**、
共享调度器自动派活（成员空闲即续领）、成员直达**邮箱通信**、队长汇总结果。

## 工作方式

1. 队长创建团队（staged 两阶段：默认**先计划后批准**，用户审 Web 面板后才启动）
2. 队长按角色添加成员（researcher/engineer/reviewer…）
3. 队长创建任务（可带依赖：依赖全部 completed 才可被领取）
4. 用户批准 → 成员原子 spawn → 调度器开始派活
5. 成员认领任务（取得 attempt_id 凭据）→ 工作 → update_task 更新状态 → send_message 汇报
6. 质量门禁：review 仅 verdict=pass 可完成；needs_revision 自动生成 repair+复审循环
7. 队长汇总结果 → agent_teams_delete（归档，磁盘保留完整记录）

## 工具清单（13 个）

| 工具 | 权限 | 说明 |
| --- | --- | --- |
| agent_teams_create | 队长 | 建团队（approval=required 默认两阶段） |
| agent_teams_edit_plan | 队长 | 原子批量修订暂存计划 |
| agent_teams_approve | 队长 | 批准暂存计划并启动 |
| agent_teams_add_member | 队长 | 添加成员（staged=计划行 / running=立即 spawn） |
| agent_teams_remove_member | 队长 | 安全移除（撤销 attempt、任务回收、禁唤醒） |
| agent_teams_create_task | 队长 | 创建任务（kind 质量契约 + dependencies） |
| agent_teams_reassign_task | 队长 | 转派任务（旧 attempt 失效） |
| agent_teams_claim_task | 成员/队长 | 认领任务（返回 attempt_id） |
| agent_teams_update_task | 成员/队长 | 更新状态/输出（校验 attempt_id + 质量门禁） |
| agent_teams_send_message | 成员/队长 | 直达消息（成员收件人即被唤醒） |
| agent_teams_status | 成员/队长 | 全景状态（DAG/覆盖矩阵/未读） |
| agent_teams_resume | 队长 | 恢复暂停团队 |
| agent_teams_delete | 队长 | 结束团队（归档） |

成员 spawn 时队长专属工具（create/add_member/remove_member/reassign_task/create_task/resume/delete）
经 `denyTools` 对其不可见。

## 状态层（磁盘即真相）

```
<workspace>/.agent-teams/<teamId>/
  team.json          团队记录（成员/任务 DAG/phase/halted/reviewPolicy）
  inbox/captain.jsonl  队长邮箱（成员汇报直达）
  inbox/<member>.jsonl 成员邮箱（按名字 sanitizeKey 存储）
  retired-members.json 退役成员会话黑名单（跨进程禁唤醒）
  archive/<teamId>/    删除时归档
```

## Web 面板

标题栏「团队」按钮 → 活动浮层（3s 轮询 `/api/agent-teams/teams`）：
- 成员卡片（状态/进度/当前任务/未读）
- 任务 DAG（状态徽章 + 依赖弱化显示）
- staged 团队：批准并运行 / 返回对话修改 / 废弃
- running 团队：暂停（halt）

## 宿主能力依赖

- `ctx.agents`（start/fork/followup/stop/status/list/lastText/report/ready）——成员会话编排（Round3 ④.1：fork 以队长快照派生）
- `ctx.llm`（models/current）——成员模型快照
- `ctx.systemPrompt.section`——用法协议注入（order 117）
- `ctx.http` / `ctx.webServer`——快照与批准路由
- `subagent/idle` 宿主事件——成员轮次结束 → 调度器续领
- 工具执行注入 `_convID`（调用会话）/`_wsRoot`（会话工作区根）——身份与状态根解析

## 配置（设置 → 多智能体团队）

| 字段 | 默认 | 说明 |
| --- | --- | --- |
| stateDir | `.agent-teams` | 团队状态目录（工作区相对） |
| memberModel | （跟随队长） | 成员默认模型 |
| memberProvider | `spawn` | 成员提供方式：`spawn`=全新会话（默认，行为不变）；`fork`=以队长会话消息快照派生（subagent_fork 对齐，persona 经 system 覆盖；fork 能力缺失自动回落 spawn）。也可用环境变量 `AGENT_TEAMS_MEMBER_PROVIDER` 覆盖 |
| executionPrompt | — | 成员全局执行指导 |
| maxMembers | 8 | 团队规模上限 |
| codeMaxRounds | 3 | 代码审查轮次上限 |
| maxRepairAttempts | 2 | 修复尝试上限 |

## 使用示例

> 使用 AgentTeams 审查 v0.5.3 之后的提交，分别从性能、安全和产品角度分工，最后输出一份汇总报告。
