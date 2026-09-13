# 同步修复成果：E:\paircode-master → D:\PairCode

- **时间**：2026-09-12
- **源**：`E:\paircode-master`（master @ `e6c1e625`，含本次四类修复的源码正式修复 + 构建产物）
- **目标**：`D:\PairCode`（**非 git 部署目录**，无可回滚兜底 → 一切改动均先备份）
- **状态**：✅ **已执行完成（2026-09-12 22:57）**，四补丁已在宿主实证在位（见文末「执行结果」）

---

## 1. 为什么可以覆盖（安全检查结论）

早期记录曾判断「工作区与 D 是两份不同源构建，勿覆盖」。本轮做了**符号级核查**，结论相反：

| 核查项 | 结果 |
|---|---|
| `plugins-src/ui-app` 内容不同文件 | 7 个（`agent-events.js`、`RightPanel.vue`、`ShellApp.vue`、`ui-state.js` + 3 个 `docs/*.md`） |
| D 独有行性质 | **全是旧实现形态**（`markHistoryLoaded` 无 flush 抽取、run-stats 前端内联累加、phase-bar 取值内联） |
| E 独有行性质 | **修复与重构**：`// ★ 2026-09-12 修复「执行中前端突然不再输出 agent 内容」`、`flushPendingEvents`（E=4 / D=0）、`WS_PENDING_MAX_MS = 3000`、`// ★ 运行统计经后端接口拉取（fetchRunStats）` |
| 自动探测「D 独有符号在 E 中缺失」 | **零缺失** |
| UI 特性命中数（两侧） | `phase-bar` 7/7、`phs-item` 6/6、`runTokenSpeed` 3/3、`convListVisible` 2/2、`toggleConvList` 4/4、`setFocusMode` 4/4 → 完全一致 |

→ **E 是 D 的超集**，覆盖不会丢失现网功能。

## 2. 同步清单（13 项）

| # | 相对路径（E 与 D 同构） | E 大小 | 动作 |
|---|---|---|---|
| 1 | `.pair\assets\runtime\web\index.html` | 10772 | 覆盖（引用改为新壳） |
| 2 | `.pair\assets\runtime\web\assets\index-D6jSa2R5.js` | 611019 | **新增** |
| 3 | `.pair\plugins\ui-titlebar\assets\ui-titlebar.js` | 41048 | 覆盖 |
| 4 | `.pair\plugins\ui-titlebar\assets\ui-titlebar.css` | 2894 | 覆盖 |
| 5 | `.pair\plugins\ui-editor\assets\ui-editor.js` | 4605995 | 覆盖 |
| 6 | `.pair\plugins\ui-editor\assets\ui-editor.css` | 29042 | 覆盖 |
| 7 | `.pair\plugins\ui-right-panel\assets\ui-right-panel.js` | 3647769 | 覆盖 |
| 8 | `.pair\plugins\ui-right-panel\assets\ui-right-panel.css` | 58317 | 覆盖 |
| 9 | `.pair\plugins\agentloop\index.js` | 69014 | 覆盖（含 master 上游指纹修复） |
| 10 | `plugins-src\ui-app\src\agent-events.js` | 51477 | 覆盖 |
| 11 | `plugins-src\ui-app\src\ui-state.js` | 21604 | 覆盖 |
| 12 | `plugins-src\ui-app\src\ShellApp.vue` | 22637 | 覆盖 |
| 13 | `plugins-src\ui-app\src\components\RightPanel.vue` | 164947 | 覆盖 |

**移走（移入备份后删除）**：`.pair\assets\runtime\web\assets\index-CTZe2XhI.js`（607211B，旧壳）

**明确不在范围**：`app-actions.js` / `MenuBar.vue` / `EditorArea.vue`（与 D 零行差）、`src\docs\*.md`、`scripts\`（25 个 E 独有）、D 侧其余旧产物（如 `*.bak-*`）。

## 3. 怎么执行

```
双击：E:\paircode-master\temp\sync-D-PairCode\RUN-SYNC.cmd
```

等价命令行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File "E:\paircode-master\temp\sync-D-PairCode\sync-fix-to-D.ps1"
```

脚本 7 步：预检源文件 → 停 `pair.exe` → 备份 → 覆盖 → 移旧壳 → SHA256 校验 → 重启并探测 9090。

**若脚本在「停止 pair.exe」处报 `[失败]`**（实例以管理员权限启动）：右键 `RUN-SYNC.cmd` →「以管理员身份运行」。

### 回滚

```powershell
# 从最近一次备份回滚（脚本会在备份时写 last-backup.txt）
powershell -ExecutionPolicy Bypass -File "...\sync-fix-to-D.ps1" -Restore

# 或指定备份目录
powershell -ExecutionPolicy Bypass -File "...\sync-fix-to-D.ps1" -Restore -BackupDir "E:\paircode-master\temp\sync-D-PairCode\backup-manual-20260912-xxxxxx"
```

回滚逻辑：备份里有的文件 → 覆盖回去；备份里没有的清单文件（即本次新增的新壳）→ 删除；旧壳 → 从备份还原。

### 其他开关

| 开关 | 作用 |
|---|---|
| `-NoRestart` | 同步后不自动重启服务（自行启动） |
| `-KeepObsolete` | 保留旧壳 `index-CTZe2XhI.js`（默认移入备份后删除） |
| `-Src` / `-Dst` | 指定源/目标目录（默认 `E:\paircode-master` / `D:\PairCode`） |

## 4. 执行后验证要点

1. 脚本尾部应输出 `一致 13/13`、`首页已引用新壳 index-D6jSa2R5.js`。
2. 浏览器 **Ctrl+F5 强刷** `http://127.0.0.1:9090/`（壳文件名已换，普通刷新可能命中缓存）。
3. 四类修复行为复核：
   - **会话列表面板**：右栏头部 `message-square` 按钮 / `Ctrl+Shift+L` 切换 `.conv-sidebar`；收起时按钮呈 `.rp-btn-off` 半透明；刷新后状态保留（localStorage `paircode-ide-state` 的 `convListVisible`）。
   - **专注模式**：`Ctrl+K` 进入 → 左栏与会话列表一起收起；退出 → 还原「专注前」的值（原本就隐藏的不被强行打开）；`Ctrl+B` 在专注态内不落盘。
   - **中断复位**：一次异常中断后，下一轮正常完成应清掉「继续任务」黄色提示条（`interrupted` 仅 `doneReason === 'stopped'` 时保留）。
   - **agentloop 驳回指纹**：同一工具「同参数」5 分钟内才免审驳回；换参数不会被误短路。
4. 状态探针（浏览器控制台）：`window.__PAIRCODE_CORE`、`window.__state`。

## 5. 备份位置

| 备份 | 位置 | 内容 |
|---|---|---|
| 执行前（AI 侧预建） | `temp\sync-D-PairCode\backup-D-sync2-20260912-224256\` | D 侧 13 文件副本（含原壳） |
| 脚本执行时自动建 | `temp\sync-D-PairCode\backup-manual-<时间戳>\` | 被覆盖文件 + 旧壳；路径同时写入 `last-backup.txt` |
| 更早的全量对照 | `temp\sync-D-PairCode\backup-D\` | 319 文件哈希清单（早期对照用） |

## 6. 备注

- `pair.exe` 由 `explorer.exe` 手动启动，**无守护/计划任务**（不会自动拉起）；停服后若不重启需自行双击 `D:\PairCode\pair.exe`。
- 本次**只同步 web 资源与源码**；`pair.exe` 由用户自行编译替换（工作区无 Go 工具链）。
- 字节比较需行尾归一化（CRLF↔LF 会产生约 199 处假差异）。

---

## 7. 执行结果（2026-09-12 23:00 核查）

### 7.1 执行事实
| 项 | 实测 |
|---|---|
| 覆盖时间 | `.pair\assets\runtime\web\assets` 目录变更 **22:57:14** |
| 进程重启 | `pair.exe` **PID 14556 → 11972**，启动 **22:57:14**（旧进程已停） |
| 插件重载 | `logs/paircode.log` L29422-29424：**22:57:15** `agentloop 注册提示词资产 7 个` + `registerLoop 已注册` |
| 插件可用性 | L29459：**22:59:58** 已跑过真实 agent 会话（`Run 开始 … stepBudget=200`） |
| 文件一致性 | 16 文件（13 清单 + 3 零行差）行尾归一化哈希 **全部一致** |
| 唯一字节差 | `app-actions.js`：E=CRLF 18966B / D=LF 18554B，差 **412B = 412 行 × 1 字节**，`diff` 归一化后为空 → 纯行尾 |

### 7.2 四补丁在位证据（三层）
| 补丁 | 源码锚点 | 产物锚点 | 运行时实测 |
|---|---|---|---|
| ① agentloop 驳回指纹 | L705 `REJECT_REMIND_MS=5*60*1000`、L707 `lastRejectFp`、L709 `argsFingerprint()`、**L716 `Math.imul(h,16777619)`**、L726 三重判定、L922 记录；与 E 逐字节同 | 插件 22:57:15 重新装载、state=running | —（宿主侧） |
| ② interrupted 复位 | `agent-events.js` **L688**（`processAgentDone` 内，非 `stopped` 才复位）；异常置 true 在 L515/712/753 | 壳 `interrupted=false` 1、`flushPendingEvents` 3、`WS_PENDING_MAX_MS` 3、`fetchRunStats` 6 | 页面加载 `./assets/index-D6jSa2R5.js` |
| ③ 会话列表显隐 | `ui-state.js` L99/247-249/403/432；`app-actions.js` **L246 Ctrl+Shift+L**；`RightPanel.vue` L10/L273/**L2604 `.rp-btn-off{opacity:.45}`** | 区域包 `convListVisible` 3、`rp-btn-off` 1、CSS `.rp-btn-off[data-v-917eb64f]{opacity:.45}`；壳 15 处 + `shiftKey && e.key === "L"` | **Ctrl+Shift+L** true→false→true；`.conv-sidebar` 渲染 250×681 visible；localStorage 存 `convListVisible` |
| ④ 专注模式唯一入口 | `ui-state.js` **L295-324 `setFocusMode`**（原值变量 L306-307 / 幂等 L311 / 收起 L314-317 / 还原 L320-321）；**无 `watch(focusMode)` 竞态** | 壳 9；`ui-titlebar.js` 7；`ui-editor.js` 2；`ui-right-panel.js` 1 | **Ctrl+K** 进入 `sidebar/conv=false`、退出完整还原；localStorage **不含** `focusMode` |

### 7.3 服务端产物流转
页面 `?rev=` 与磁盘哈希逐位一致：`ui-editor` `1453d4ef6e696aa6`、`ui-right-panel` `47656055b7bfdedc`、`ui-titlebar` `fc6300880a45e40c`。

### 7.4 结论
**四个补丁问题在宿主 `D:\PairCode` 确实已全部修复**——非仅采信 E 侧结论，而是以「源码行级锚点 + 产物符号计数 + 9090 运行时行为实测 + 服务端 `?rev=` 哈希」四重证据闭环。
