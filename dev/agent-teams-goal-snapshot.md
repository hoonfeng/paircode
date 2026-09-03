# v1.6 UX 打磨——Team Agents 目标快照 + 面板填写指南

> 用途：①用户在 UI 团队面板手动创建团队时照此填写；②新会话自动创建的依据。
> 工具面现状：「计划讨论」工具集已含 agent-teams 插件（14 个 agent_teams_* 工具启用，toolset_show 已验证）。

## 团队：v1.6-ux-polish（两阶段审批 approval=required）

**目标**：v1.6 UX 打磨 + 常规代码质量提升——近期用户高频反馈 3 个交互缺陷；
同时全量代码质量自检（bug_detect/bug_fix 循环）+ 提交改进。

### 成员（5）
| ID | 展示名 | 模型路线 | 职责 |
|----|--------|----------|------|
| analyst | 需求分析师 | deepseek-chat | 三个 UX 缺陷的现场复现与根因定位，产出复现步骤+证据截图+根因分析 |
| engineer | 交互修复工程师 | deepseek-chat | 依据根因修复三个交互缺陷，遵守 30 行小步提交纪律 |
| quality-guardian | 质量守护 | deepseek-chat | 全库 bug_detect/bug_fix 循环修复直至零错误，保存经验胶囊 |
| committer | 提交管理员 | deepseek-chat | 分组提交（feat/docs 各自独立提交）并推送 |
| reviewer | 交付审查员 | deepseek-chat | 逐缺陷验证+契约审查，verdict=pass 才放行集成 |

### 任务 DAG（8 节点）
```
T1 分析三缺陷 ─┬─→ T2 修缺陷 ─┬─→ T7 验收+审查 ─→ T8 集成+提交
T4 全库质检 ──→ T5 修复循环 ──┘
T6 知识沉淀（与 T5 并行，quality-guardian）
```

### 任务清单与契约
| ID | 标题 | objective | acceptance | inScope | verify | assignee | 依赖 |
|----|------|-----------|------------|---------|--------|----------|------|
| T1 | 分析三个 UX 缺陷并定位根因 | 现场复现三个缺陷，精确定位根因与文件行号 | 缺陷A/B/C 各有：复现步骤+证据截图+根因分析+文件定位 | 仅分析与定位，不改代码 | 产出分析报告 | analyst | 无 |
| T2 | 修复三个 UX 缺陷 | 修复 T1 定位的三个缺陷 | 三缺陷修复后 web_debug 无控制台错误+截图对比 | 仅 T1 列出的文件 | web_debug 复验+前后截图对比 | engineer | T1 |
| T4 | 全库代码质量自检 | 运行 bug_detect 全库扫描并汇总 | 输出完整缺陷清单 | 只读分析 | 报告无遗漏入口 | quality-guardian | 无 |
| T5 | 修复质检发现的缺陷 | 修复 T4 全部发现 | 修复后 bug_detect 重跑零错误 | 仅 T4 报告涉及文件 | bug_detect 重跑验证 | quality-guardian | T4 |
| T6 | 经验沉淀 | 保存修复经验胶囊+更新知识库 | ≥1 条新经验胶囊+知识库根因篇 | 记忆/知识库 | resource_list 可见 | quality-guardian | 无（与 T5 并行） |
| T7 | 验收与审查 | 逐缺陷验证+契约审查 | verdict=pass | 全部交付物 | 验证脚本+人工核对 | reviewer | T2, T5 |
| T8 | 集成与提交 | feat/docs 分组提交+推送 | git log ≥2 提交，工作区 clean | git 操作 | git_status 验证 | committer | T7 |

## ⚠️ 调用异常根因（最终结论，勿再误判为运行时拦截）
- 表象：本会话多次「打算调用 agent_teams_create」，实际生成的是 write 占位文件。
- 根因：**本会话的函数 schema 是会话开始时的快照**，当时工具集尚未加入 agent-teams，
  schema 中没有 agent_teams_* 的函数定义；生成层缺锚点导致调用漂移到 write。
- 运行时工具面本身已打通（toolset_edit 成功 + toolset_show 验证 14 工具在列）。
- **推论：新开会话 → schema 重新快照 → agent_teams_* 出现在函数面 → 可直接调用。**

## 两条执行路径
- **路径 A（用户手动）**：按下方「面板填写对照」在 UI 团队面板逐字段创建。
- **路径 B（新会话自动）**：新开会话，指令：
  「读取 dev/agent-teams-goal-snapshot.md，按其中团队定义调用 agent_teams_create
  （name=v1.6-ux-polish, approval=required），创建 5 个成员与 T1-T8 任务，停在 staged 等审批。」

## 面板填写对照（路径 A 用）
- 团队面板 → 新建团队：名称 `v1.6-ux-polish`，目标=上方「目标」段原文，审批模式=**需审批（两阶段）**
- 添加成员（5 个）：名称/模型路线/职责照「成员」表逐行填
- 新建任务（8 个）：标题=T1~T8 编号+名称；目标/验收照「任务清单与契约」表填；
  实现类任务（T2/T5）补 inScope 与 verify；依赖按表中「依赖」列设置
- 填完保存：面板应显示 8 任务待审批状态，点「批准并运行」开始调度
