# 9096 流程 × 任务看板 多场景实机测试报告

- 测试时间：2026-10-17（下午）
- 测试实例：`http://localhost:9096`（PID 28708，启动于 2026-10-17 12:57）
- 对照实例：9090（PID 38916）—— **全程未做任何操作**
- 被测插件：`tool-flow`（流程库 / 流程运行 / 门禁 / 任务看板同包）
- 执行通道：`POST /api/plugins/invoke` → `flow_action` / `taskboard_action`
  （与页面按钮走同一 RPC 通道，属真实后端行为，不是只读模拟）
- 测试脚本：`scripts/e2e-flow-taskboard-9096.js`（自动清理，可 `--keep` 保留现场）
- 截图脚本：`scripts/cdp-shot-taskboard-9096.js`

## 一、结果总览

| 项目 | 结果 |
|---|---|
| 最终断言行 | **62 通过 / 0 失败** |
| 首轮断言行 | 55 通过 / 5 失败（5 项均为**断言与时序/语义不符**，非产品缺陷，详见第五节） |
| 测试流程 | `e2e-scenario-flow`（3 阶段 / 6 门禁，覆盖 check、command、artifact、approval 四类） |
| 测试卡片 | 3 张（流程绑定卡、未绑流程对照卡、startFlow 入口卡），测试后已 trash |
| 测试流程实例 | 2 个，测试后已 purge（无残留） |
| 页面视觉 | 看板卡片视图 + 流程库画布均正常，无重叠/错乱 |

## 二、测试范围为（场景清单）

1. 流程定义与校验（新增可写流程、非法定义拒绝、preset 只读）
2. 建卡 → 流程启动 → 卡片绑定与 DoD 清单生成
3. 门禁未过时推进被拦截
4. 四类门禁逐个实测（check / command / artifact / approval）
5. 门禁通过后阶段推进
6. 中断 / 恢复
7. 流程完成 → 卡片自动转「待验收」
8. 验收权限（人 vs agent）与未绑流程对照
9. 看板「启动流程」入口（startFlow）
10. 边界与异常（未知卡/流程/runId、非法状态、跨会话不越界）
11. 页面 UI 侧截图确认

## 三、场景明细与实测结论

### 场景 0 · 现状盘点（只读）
- 看板快照可读（revision 176），流程库 8 条预设流程可读。
- 存在已绑流程的进行中演示卡、存在未绑流程的待验收对照卡（`t-1792202391427-xzsd`）。

### 场景 1 · 流程定义与校验
| 断言 | 结果 | 实测返回 |
|---|---|---|
| 合法流程保存（project 层） | PASS | `已保存流程 e2e-scenario-flow（project 层 v1）` |
| 回读 3 阶段 / 6 门禁 | PASS | `prep:2, impl:2, wrap:2` |
| command 门禁缺 `cmd` 被拒 | PASS | `校验未通过：command 门禁 g-bad 缺少 cmd` |
| 非法定义不落盘 | PASS | `.pair/flows/e2e-bad-flow.json` 不存在 |
| preset 只读层拒写 | PASS | `scope 必须是 project \| global（preset 只读，不可写）` |
| 非法 JSON 校验报错 | PASS | `JSON 解析失败：SyntaxError…` |

### 场景 2 · 建卡 + 流程启动 + 绑定与 DoD
| 断言 | 结果 | 实测返回 |
|---|---|---|
| 看板建卡 | PASS | 生成 `t-…` 卡 |
| 流程启动并绑卡 | PASS | 返回 `run-…`，`taskId` 被绑定 |
| 卡 `runId` 回写 | PASS | 卡上 runId == 启动返回 runId |
| 卡 `flowId` 补全 | PASS | `e2e-scenario-flow / project / v1` |
| DoD 清单 = 门禁数（6） | PASS | `g-check-1, g-cmd-1, g-art-1, g-appr-1, g-art-bad, g-check-2` |
| DoD 初始全未勾选 | PASS | — |
| 同会话重复启动被拒 | PASS | `已有进行中的流程实例：run-…` |

### 场景 3 · 门禁未过 → 推进被拦截
- `advance`（不带 gateId）：`advanced=false`，`pending` 列出 2 个必过门禁（`g-check-1`、`g-cmd-1`）。
- 拦截文案包含「处置：check 门禁用 evidence；command 门禁代跑；artifact 确认产物；approval 必须用户批准」——可操作性良好。

### 场景 4 · 四类门禁实测
| 门禁类型 | 断言 | 结果 | 实测 |
|---|---|---|---|
| check（无证据） | status 变 pending 且不推进 | PASS | `待提供证据（check 门禁需说明如何确认）` |
| check（带证据） | status=pass | PASS | 证据文本落 `evidence` + `evidenceFile` |
| command | 自动执行并通过 | PASS | `命令通过（输出 12 字符）`，输出 `e2e-gate-ok` |
| 未知门禁 id | 明确报错并列出已有门禁 | PASS | `阶段「实施」中不存在门禁 g-no-such-gate（现有: g-art-1, g-appr-1）` |
| artifact（存在） | pass | PASS | `产物存在: .pair/project.md` |
| artifact（不存在） | fail（非必过不阻断） | PASS | `产物不存在: .pair/__not_exist_e2e__.md` |
| approval（未批准） | pending | PASS | `待用户批准（approval 门禁不得自动通过）` |
| advance 传 `approve:false` | 不产生批准，仍 pending | PASS | 同上（不报错，属设计语义） |
| `approve` 缺 `approve:true` | 被拒 | PASS | `approve 必须为 true（审批卡点由用户在界面显式批准）` |
| `approve:true` | approved | PASS | `用户已批准（E2E 用户显式批准）` |

**DoD 回填**：门禁通过后，卡片清单项自动勾选并标记 `by=flow`（前 2 项实测 `["g-check-1","g-cmd-1"]`）。

### 场景 5 · 中断 / 恢复
- `interrupt` → 再 `advance` 被拒：`流程实例不是进行中状态（interrupted）——用 resume 恢复后继续`（PASS）。
- `resume` 成功后继续推进，仍被未过的收尾门禁拦截（可继续操作，PASS）。

### 场景 6 · 流程完成 → 卡自动转待验收
- 最后一个必过门禁通过时 `completed=true`（PASS）。
- 卡片自动从「进行中」转「待验收」（PASS）。
- 6 项 DoD 中 5 项勾选，失败的非必过门禁（`g-art-bad`）保持未勾（PASS）。
- 已完成实例再 `advance`：`流程实例不是进行中状态（completed）——实例已完成`（PASS）。

### 场景 7 · 验收权限 + 未绑流程对照
| 断言 | 结果 | 说明 |
|---|---|---|
| 人可将「待验收」卡移到「已完成」 | PASS | RPC 通道身份固定为 user |
| 未绑流程待验收卡存在（对照） | PASS | `t-1792202391427-xzsd` |
| 未绑流程卡无 DoD 清单 | PASS | checklist 长度 0 |
| 未绑流程建卡无 flowId / 无清单 | PASS | — |
| 状态机禁止 `todo → done` 跳级 | PASS | `无法从「待办」移到「已完成」` |
| 未绑流程卡 `in_review → done` 允许 | PASS | 无门禁约束，仅受状态机约束 |

> 说明：实机 HTTP invoke 通道的 actor 固定为 `user`（`taskboard_action` 内 `byUser = {kind:'user'}`），
> 无法在实机伪造 agent 身份。「agent 不能把卡移到已完成」的协议闸由插件自带 smoke 回归覆盖
> （278 项中的「协议闸：agent 不能把任务移到「已完成」」），本次未在 9096 上以 agent 身份复现，
> 属测试手段限制，非缺陷。

### 场景 8 · 看板「启动流程」入口
- 未绑流程卡 `startFlow` 被拒：`任务未绑定流程（建卡时可选流程模板）`（PASS）。
- 卡可直接 `update {flowId}`（不启动）（PASS）。
- `startFlow` 启动流程并回写 `runId`，卡上 `runId` 绑定正确（PASS）。

### 场景 9 · 边界与异常
| 断言 | 结果 | 实测返回 |
|---|---|---|
| 绑定不存在的卡 | PASS | `看板任务不存在：t-no-such-task…` |
| 不存在的流程 | PASS | `流程不存在（三层库里没有）: no-such-flow` |
| 不存在的 runId | PASS | `没有可用的流程实例（用 flow_start 启动）` |
| 未知看板状态 | PASS | `未知状态：no-such-status` |
| 显式会话无实例不越界 | PASS | `没有可用的流程实例…`（不会打到其他会话） |

### 场景 10 · 页面 UI 侧
- 任务看板卡片视图：5 列（待规划 / 待办 / 进行中 / 待验收 / 已完成），4 张卡布局正常。
- 流程库：左侧 8 条流程（含 4 阶段/门禁数），中间画布 4 节点纵向排列，右侧节点属性检查器；`预设流程（只读）` 无直接编辑入口（只有「克隆到项目」）。
- 截图：`screenshots/taskboard-cards-9096.png`、`screenshots/taskboard-flowlib-9096.png`、`screenshots/taskboard-board-9096.png`

## 四、实测确认的关键行为（供后续使用）

1. **门禁通过会「立即推进」**：若该门禁是本阶段最后一个必过门禁，`advance {gateId, evidence}` 会同时返回
   `advanced=true` 与新阶段 id；`approve` 同理。不要假设「过一次门禁仍停在本阶段」。
2. **审批只能走 `approve` op**：`advance {approve:false}` 不报错，只记录 pending；
   `approve` 必须显式 `approve:true`，否则被拒。
3. **非必过门禁（required:false / blocking:false）失败不阻断推进**，但 DoD 不勾选（进度可信）。
4. **启动判重按会话**：同 `convId` 已有活跃实例即拒绝；无 convId 时按 project 判重；`force:true` 可另起。
5. **`pickRun` 的会话边界**：传了 `convId` 不会越界到其他会话；不传则取最后活跃实例（会对其他会话生效）。
6. **卡片 DoD 生成时机**：`flow_action start {taskId}` 或 `taskboard_action startFlow` 时生成（= 流程门禁数）；
   未绑流程的卡没有 DoD。
7. **流程结束 → 卡片自动转 in_review**（不自动 done），done 必须由人验收。

## 五、首轮 5 项失败的原因（均为断言问题，已修正）

| 首轮失败项 | 原因 | 结论 |
|---|---|---|
| 必过门禁全通过 → 推进到 impl | 推进实际发生在最后一个门禁通过那次调用内（连带推进） | 断言时序写错 |
| approve=false 被拒绝 | `advance` 不校验 approve 值，只记录 pending；审批走 `approve` op | 断言期望错 |
| 恢复后推进到 wrap | `approve:true` 已连带推进到收尾阶段 | 断言时序写错 |
| 全流程完成（completed=true） | 完成发生在收尾门禁通过那次调用内 | 断言时序写错 |
| 无实例时推进给出明确提示 | 不传 convId 时按最后活跃实例兜底（设计如此）；改测「显式会话不越界」 | 断言设计错 |

修正断言后第二轮 **62/62 全绿**，未发现产品功能缺陷。

## 六、遗留问题与建议

1. **测试手段限制**：agent 身份验收闸未在 9096 实机复现（HTTP invoke 固定 user）。建议后续用 agent 工具面
   （`taskboard_move`）在真实会话中补一条端到端证据；插件 smoke 已覆盖该断言。
2. **UI 小体验点**：从侧栏进入「任务看板」视图时，若上次停留在「流程库」页签，会保留页签状态
   （首次探测 `.ftb-tcard` 为 0）。不是缺陷，但建议脚本/用户操作时先点「任务看板」页签。
3. **页面截图依赖 CDP**：`Page.reload` 后 CDP 会话偶发失去响应，截图脚本已改为不 reload + 15s 超时保护；
   后续脚本沿用该做法。
4. **测试数据清理**：3 张测试卡已 trash、2 个 run 已 purge、`e2e-scenario-flow.json` 已删除；
   未绑流程对照卡 C 保持原状未被改动。

## 七、证据文件

- `scripts/e2e-flow-taskboard-9096.js`（62 项断言的场景脚本，可重复执行）
- `scripts/cdp-shot-taskboard-9096.js`（页面截图脚本）
- `screenshots/taskboard-cards-9096.png`（看板卡片视图）
- `screenshots/taskboard-flowlib-9096.png`（流程库左列表 + 右画布）
- `screenshots/taskboard-board-9096.png`（任务看板视图）
