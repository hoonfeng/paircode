# T1 前端死组件与构建产物盘点（只读分析报告）

> 项目：PairCode IDE（gou-ide）v1.6 UX Polish 第 1 项 · 前端死组件清理核实
> 性质：只读盘点。本报告为唯一交付物，未修改任何源码/构建产物。
> 基线：git HEAD = 5f2c41a3（2026-09-03 19:28）

## 1. 基线说明（git status）

当前工作区未提交改动仅 2 项，**均非垃圾、不属于本轮清理范围**：

| 项 | 状态 | 归属说明 |
|---|---|---|
| ` M cmd/companion/kernel_register.go` | 已修改 | `git diff` 为空内容差异（仅 CRLF/LF 行尾警告，无 +/- 行）→ 实质无改动，行尾符噪声 |
| `?? dev/` | 未跟踪 | 含 `dev/agent-teams-plan-draft.md`（旧草稿）+ 本报告（任务交付物），非垃圾 |

任务背景提及的「未提交改动」（RightPanel.vue/SheetPicker.vue/ToolsetPanel.vue、
.pair/assets/runtime/web 新旧 bundle、ui-right-panel 产物、scripts/cdp-*.js、
packager.json）**均已在 HEAD 提交 5f2c41a3 中落库**（提交信息：SheetPicker 改回原生
select 科技风 + 工具集面板添加插件浮层卡片墙 + 模型 desc 精简 + 版本 1.5.1；10 files
changed）。构建产物当前仅单一活跃 bundle（见 §4），不存在待提交的旧产物堆积。

## 2. 死组件清单（src 全量引用图验证）

方法：遍历 plugins-src/ui-app/src 全部 62 个 .vue/.js 文件（43 组件 + 19 顶层/入口），收集静态
import、动态 import()、模板 <Xxx> 标签引用，构建完整引用图。动态 import() 在 src 内 **0 个**
（vite inlineDynamicImports 单 bundle 架构，无异步分割）。

### 2.1 本轮发现的死文件（唯一：chat-utils.js）

| 文件 | 零引用证据 | 删除风险 |
|---|---|---|
| plugins-src/ui-app/src/chat-utils.js（159 行，导出 12 函数：useMessageCombos/cleanMsgContent/
isFeedback/isDelegation/delegationAgent/isSystemMsg/msgSummary/toolMeta/toolResultSummary/
isTerminalTool/formatTerminalCommand/mergeConsecutiveAssistant） | ① 文件头注释自述「供
RightPanel.vue 使用（ChatView.vue 已废弃删除）」——唯一消费者 ChatView.vue 已删；② 全部 src 62
文件 grep chat-utils **0 命中**；③ 动态 import() 全 src 0 个；④ 全工作区（.pair/plugins +
scripts + cmd/companion/web-ui）grep **0 命中**，仅 docs/*.md 历史记录提及 2 处
（docs/leftover-round3-verification.md:117 与 :127）；⑤ RightPanel.vue 全文双确认无 chat-utils
字样 | **低**（历史消费者已删、无动态/字符串引用，删除安全） |

### 2.2 Round3 ⑥.4 已清理项——复检确认零残留

| 项 | 复检证据 | 结论 |
|---|---|---|
| OutputPanel/SubAgentBlock/TasksPanel/PlanView/TaskBoard（5 旧组件） | 全工作区 grep（排除
docs/release）命中 13 处全部为 docs/*.md 历史记录 + RightPanel.vue 注释（:273「SubAgentBlock
不再使用」、:2491、:2662 样式注释）；src/components 目录无这些文件 | ✅ 无残留 |
| src/router.js | 物理不存在（fs.existsSync=false）；全 src grep vue-router|createRouter|
createWebHistory **0 命中** | ✅ 无残留 |
| vue-router 依赖 | package.json dependencies 已无（见 §5） | ✅ 源码层已清（lock 残留见 §5.2） |
| ui-state.js.bak | 工作区 find ui-state*.bak **0 命中**（仅 docs 提及 6 处） | ✅ 无残留 |
| 根 web/ 旧壳（index.html + favicon.svg + web/assets/index-CrzpNXyK.js） | git log -- web/index.html
末次提交 59f028fb「Round3-11: ⑥.4 前端死件/产物——删 … 根 web/ 旧壳 3 文件」；物理 web/ 目录
不存在（dir 报「找不到文件」） | ✅ 无残留 |

### 2.3 非死代码（确认存活，勿删）

- **ui-main-*.js × 9**（activitybar/editor/git/marketplace/modals/right-panel/sidebar/statusbar/
  titlebar）：「零引用」是正常形态——它们是 **build-ui.mjs 分布式区域插件构建入口**，由各区域
  插件 package.json 的 dsh.ui.build.entry 声明消费（git-api/marketplace/ui-activitybar/
  ui-editor/ui-modals/ui-right-panel/ui-sidebar/ui-statusbar/ui-titlebar 9 区一一对应）。
- **main.js**：vite 单一入口（plugins-src/ui-app/index.html → <script type="module"
  src="/src/main.js">）。
- **ui-state.js**（25 处引用）、**api.js**（21）、**plugin-runtime.js**（9）、**app-actions.js**（5）、
  **agent-events.js**（3）——全部活跃。
- **components/ 43 个组件全部有引用链**：UiModals.vue 挂载 6 个 modal 系组件（About/GlobalDialogs/
  Help/Settings/Source/System），UiEditor.vue→EditorArea.vue、UiRightPanel.vue→RightPanel.vue、
  UiTitlebar.vue→MenuBar.vue、UiSidebar→Sidebar.vue 等，ToolsetPanel.vue 由 ShellApp.vue 主 tab
  直接挂载（ShellApp.vue:77-78）。FileTreeItem.vue 模板自引用 <FileTreeItem> 为递归组件（正常）。

## 3. 孤儿产物清单（构建 bundle）

### 3.1 .pair/assets/runtime/web（壳运行产物）

| 文件 | 状态 |
|---|---|
| assets/index-Cokov9cE.js | 活跃——index.html 唯一引用（script type=module src=./assets/index-Cokov9cE.js） |
| assets/index-9L31CVnN.js（旧） | **已无残留**：0afea20e 提交 git show --stat 显示 rename 替换（index-9L31CVnN.js => index-Cokov9cE.js）；物理不存在（find 仅 1 个 bundle）、git ls-files 无此路径、全工作区 grep hash 0 命中 |
| index.html / favicon.svg | 活跃 |

结论：**无孤儿 bundle**——旧 hash 产物已被 0afea20e 提交清理干净。

### 3.2 cmd/companion/web-ui/dist（go:embed 兜底）

- dist/assets/ 仅有 index-Cokov9cE.js；dist/index.html 引用 ./assets/index-Cokov9cE.js——一致，无孤儿。
- build-ui.mjs 内置「预清理历史 index-*.js bundle」逻辑（scripts/build-ui.mjs 注释：Round3 ⑥.4 防堆积——构建前清理 dist/assets 下未引用 hash 产物）→ 机制上防再堆积。

### 3.3 区域插件 assets（9 区）

.pair/plugins/{git-api,marketplace,ui-activitybar,ui-editor,ui-modals,ui-right-panel,ui-sidebar,ui-statusbar,ui-titlebar}/assets/ 每区仅 favicon.svg + 1 css + 1 js（如 ui-right-panel/assets/ui-right-panel.css|ui-right-panel.js），**无历史堆积、无孤儿**。

### 3.4 已清理历史产物（存档记录）

- 根 web/assets/index-CrzpNXyK.js：59f028fb 已删（docs/plugin-round3-plan.md:165 记录）。


## 4. docs 目录消费确认（src/docs 非死代码）

| 文件 | 消费方 |
|---|---|
| changelog.md / features.md / api-docs.md / tools.md / shortcuts.md / faq.md / getting-started.md（共 7 篇） | 全部被 HelpModal.vue:78-84 消费（featuresMd/apiDocsMd/toolsMd/shortcutsMd/faqMd/gettingStartedMd/changelogMd 共 7 个 raw 导入变量，渲染进帮助弹层） |

结论：docs 目录 7 篇全部存活，无路由/面板消费死角（Router 已整体移除，docs 不经路由，直接 raw 导入进 HelpModal——消费链完整）。

## 5. 零使用依赖（package.json / package-lock）

### 5.1 package.json 声明但源码零引用

| 依赖 | 声明位置 | 证据 | 删除风险 |
|---|---|---|---|
| pinia ^4.0.1 | dependencies | 全部 src 66 文件 grep pinia **0 命中**（非注释行）——Vue 状态用 ui-state.js 自研单例，pinia 未启用 | **低** |
| xterm ^5.3.0 | dependencies | 源码 grep xterm 包名 **0 命中**（仅 @xterm/xterm v6 被 TerminalPanel 使用） | **低** |

### 5.2 package-lock.json 残留（Round3 删包未同步 lock）

- lock 根包 dependencies 仍含 vue-router ^4.6.4；node_modules/vue-router（4.6.4）条目 + 子依赖 @vue/devtools-api 6.6.4 仍在 lock；node_modules 物理存在。
- 源码引用 vue-router **0 命中**；package.json 无此依赖 → **纯 lock 残留**。npm ci 仍会安装 vue-router（体积浪费）。修复：npm install 或 npm prune 重新生成 lock（**低风险**，无需手删）。

### 5.3 反向缺口（未声明但被使用——建议补声明，非死代码）

- @codemirror/lint、@lezer/highlight：CodeEditor.vue:16/:18 使用，但 package.json dependencies 未列出（当前靠传递依赖可用，属依赖声明缺口，不影响本轮清理）。

### 5.4 已确认无残留

- vue-router 已从 package.json dependencies 移除（Round3 ⑥.4）✅；router.js 已删 ✅。

## 6. 验证脚本盘点（scripts/cdp-*.js × 7）

| 脚本 | 对应 UI | 状态 |
|---|---|---|
| cdp-shot-add-plugin.js | 工具集面板「添加插件」浮层卡片墙截图（5f2c41a3 新增 26 行） | ✅ 活 |
| cdp-shot-chat.js | SheetPicker 弹层截图（聊天输入现代化后） | ✅ 活 |
| cdp-shot-toolset.js | 工具集面板截图 | ✅ 活 |
| cdp-verify-chat-input.js | 聊天输入原生 select 科技风验证（5f2c41a3 适配 105 行改写） | ✅ 活 |
| cdp-verify-toolset-panel.js | 工具集面板 master-detail + 添加插件浮层验证（5f2c41a3 适配） | ✅ 活 |
| cdp-verify-toolset-tab.js | 主 tab 工具集面板验证（1e2d13db 主 tab 迁移后新增） | ✅ 活 |
| **cdp-verify.js** | ★ 头部注释「点击活动栏『工具集』」；L126-135 实际查询 .activity-bar button[title=工具集] 并点击——**活动栏「工具集」按钮已随 1e2d13db 移除**（迁入主 tab），该路径在当前 UI 必然 FAIL「活动栏无工具集入口」 | ⚠️ **历史脚本** |

cdp-verify.js 删除风险：**中**——验证路径失效且功能已被 cdp-verify-toolset-tab.js / cdp-verify-toolset-panel.js 完全取代；6 个存活脚本均为独立复制骨架，删除不影响其余脚本。若保留作为通用 CDP 骨架参考亦可，但不应再作为验证入口使用。

## 7. 汇总：删除建议与风险标注

### 建议删除（本轮实际可清理项）

| # | 项 | 风险 | 理由 |
|---|---|---|---|
| 1 | plugins-src/ui-app/src/chat-utils.js | **低** | 唯一死源码文件：零引用（唯一消费者 ChatView.vue 已删；src 内 0 grep 命中；无动态/字符串引用；全工作区仅 docs 历史记录提及） |
| 2 | pinia（package.json dependencies） | **低** | 源码 0 引用；ui-state.js 已取代 |
| 3 | xterm v5（package.json dependencies） | **低** | 源码 0 引用；@xterm/xterm v6 已取代 |
| 4 | vue-router lock 残留（package-lock + node_modules） | **低** | 纯 lock/物理残留，npm install 即可同步；package.json/源码已无 |
| 5 | scripts/cdp-verify.js | **中** | 验证路径失效（活动栏入口已移除）、有两替代脚本；删除前确认不保留骨架需求 |

### 明确不可删（非死代码）

- ui-main-*.js × 9（区域插件构建入口）、main.js（vite 入口）、ui-state.js / api.js / plugin-runtime.js / app-actions.js / agent-events.js
- components/ 43 个组件（引用图全活；ToolsetPanel 由 ShellApp 主 tab 挂载）
- src/docs 7 篇（HelpModal 消费）
- 所有当前 bundle（.pair/assets/runtime/web、dist、区域插件 assets——均为活跃单一产物，无孤儿）

### 附：已验证无残留（无需处理）

OutputPanel/SubAgentBlock/TasksPanel/PlanView/TaskBoard、src/router.js、vue-router package.json 依赖、ui-state.js.bak、根 web/ 旧壳 3 文件、index-9L31CVnN.js 旧 bundle。

---
复盘备注：本报告经逐项 grep/引用图/git 历史验证，全部结论附证据行；唯一待办为 chat-utils.js 删除（低风险）与 cdp-verify.js 停用（中风险），其余为 lock 同步级清理。
