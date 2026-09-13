# D:\PairCode ↔ master 差异清单

生成时间：2026-09-12T14:41:57.748Z
基准：GitHub hoonfeng/paircode master = `e6c1e625`（E 侧已逐字节一致）

## 汇总

比较口径：**行尾归一化**（CRLF↔LF 视为相同）；`仅行尾` 列 = 内容实质相同、只是换行符不同 → 不建议覆盖。

| 组 | E 文件 | D 文件 | 仅 E（D 缺） | 仅 D（D 本地） | **内容不同** | 仅行尾 |
|---|---|---|---|---|---|---|
| .pair/plugins | 112 | 135 | 0 | 23 | **8** | 89 |
| .pair/assets/runtime/web | 3 | 3 | 1 | 1 | **1** | 1 |
| .pair/assets/runtime | 6 | 5 | 2 | 1 | **1** | 3 |
| plugins-src/ui-app | 71 | 71 | 0 | 0 | **7** | 48 |
| plugins-src/plugins | 71 | 81 | 0 | 10 | **0** | 46 |
| docs | 17 | 17 | 0 | 0 | **0** | 8 |
| config/skills | 3 | 3 | 0 | 0 | **0** | 1 |
| assets | 4 | 4 | 0 | 0 | **0** | 1 |
| scripts | 27 | 2 | 25 | 0 | **1** | 1 |
| README.md | 1 | 1 | 0 | 0 | **0** | 1 |
| config/models → models | — | — | — | — | — | — *(跳过：E=missing D=dir)* |
| config/skills → bin? | — | — | — | — | — | — *(跳过：E=missing D=dir)* |

合计：仅 E（D 缺失）**28** 个、仅 D（本地独有）**35** 个、内容不同**18** 个、仅行尾不同**199** 个。

> ⚠️ 本报告**只读**产出，未修改 E 或 D 任何文件。

---

## .pair/plugins（插件产物）

> D 多出 9 个本地插件 + agentloop 的 .bak

### ▶ 仅 D 存在（23）— 覆盖同步将**删除**这些

- `agentloop\index.js.bak-20260910-193019`
- `agentloop\index.js.bak-before-fingerprint-fix`
- `agentloop\index.js.bak-before-gitfields-20260910-200426`
- `agentloop\index.js.bak-before-reviewview-20260910-194235`
- `agentloop\index.js.bak-before-reviewview2-194318`
- `host-capability-probe\index.js`
- `host-capability-probe\package.json`
- `tool-asset\index.js`
- `tool-asset\package.json`
- `tool-core\index.js`
- `tool-core\package.json`
- `tool-debug\index.js`
- `tool-debug\package.json`
- `tool-entryconfig\index.js`
- `tool-entryconfig\package.json`
- `tool-git\index.js`
- `tool-git\package.json`
- `tool-snapshot\index.js`
- `tool-snapshot\package.json`
- `tool-vision\index.js`
- `tool-vision\package.json`
- `web-api\index.js`
- `web-api\package.json`

### ✗ 内容不同（8）

| # | 文件 | E 大小 | D 大小 | E独有行 | D独有行 | D 独有行示例 |
|---|---|---|---|---|---|---|
| 1 | `agentloop\index.js` | 69014 | 53011 | 266 | 0 |  |
| 2 | `llm-trace\index.js` | 4116 | 3338 | 10 | 3 | //            usage?, content?, reasoning?, toolCalls?, stopReason, err?} ⏎ //   usage 含 promptCacheHitTokens/promptCach |
| 3 | `ui-editor\assets\ui-editor.css` | 29042 | 29042 | 2 | 2 | @keyframes stopPulse-faf69761{0%,to{opacity:.6}50%{opacity:1}}@keyframes stopRingPulse-faf69761{0%,to{opacity: ⏎ */.xter |
| 4 | `ui-editor\assets\ui-editor.js` | 4605995 | 4605993 | — | — | — |
| 5 | `ui-right-panel\assets\ui-right-panel.css` | 58317 | 58275 | 1 | 1 | @keyframes stopPulse-faf69761{0%,to{opacity:.6}50%{opacity:1}}@keyframes stopRingPulse-faf69761{0%,to{opacity: |
| 6 | `ui-right-panel\assets\ui-right-panel.js` | 3647769 | 3646458 | — | — | — |
| 7 | `ui-titlebar\assets\ui-titlebar.css` | 2894 | 2894 | 1 | 1 | .menubar[data-v-5c2b578e]{display:flex;flex-direction:row;align-items:center;height:100%;gap:0}.menu-group[dat |
| 8 | `ui-titlebar\assets\ui-titlebar.js` | 41048 | 41048 | 1 | 1 | var UiTitlebar=(function(V,e,l,p,B,w){"use strict";const E=(n,d)=>{const t=n.__vccOpts\|\|n;for(const[a,y]of d)t |

### ␍ 仅行尾差异（89）— 内容实质相同（CRLF↔LF），**不建议**为此覆盖

- `.npmignore`
- `agent-teams\client.js`
- `agent-teams\index.js`
- `agent-teams\package.json`
- `agent-teams\README.md`
- `agent-teams\smoke-test.js`
- `agentloop\package.json`
- `agentloop\prompts\explorer.md`
- `agentloop\prompts\judge.md`
- `agentloop\prompts\planner.md`
- `agentloop\prompts\reviewer.md`
- `agentloop\prompts\verifier.md`
- `core-api\index.js`
- `core-api\package.json`
- `fs-api\index.js`
- `fs-api\package.json`
- `git-api\assets\favicon.svg`
- `git-api\assets\git-panel.css`
- `git-api\assets\git-panel.js`
- `git-api\client.js`
- `git-api\index.js`
- `git-api\package.json`
- `llm-trace\package.json`
- `marketplace\assets\favicon.svg`
- `marketplace\assets\marketplace-panel.css`
- `marketplace\assets\marketplace-panel.js`
- `marketplace\client.js`
- `marketplace\package.json`
- `tool-codegraph\index.js`
- `tool-exec\index.js`
- `tool-exec\package.json`
- `tool-harness\index.js`
- `tool-harness\package.json`
- `tool-memory\index.js`
- `tool-project-info\index.js`
- `tool-resource\index.js`
- `tool-resource\package.json`
- `tool-scenario\index.js`
- `tool-scenario\package.json`
- `tool-scenario\README.md`
- `tool-scenario\smoke-test.js`
- `tool-system\index.js`
- `tool-web\index.js`
- `tool-web\package.json`
- `tool-workflow\index.js`
- `tool-workflow\package.json`
- `ui-activitybar\assets\favicon.svg`
- `ui-activitybar\assets\ui-activitybar.css`
- `ui-activitybar\assets\ui-activitybar.js`
- `ui-activitybar\client.js`
- `ui-activitybar\index.js`
- `ui-activitybar\package.json`
- `ui-appearance\index.js`
- `ui-appearance\package.json`
- `ui-editor\assets\favicon.svg`
- `ui-editor\client.js`
- `ui-editor\index.js`
- `ui-editor\package.json`
- `ui-modals\assets\favicon.svg`
- `ui-modals\assets\ui-modals.css`
- `ui-modals\assets\ui-modals.js`
- `ui-modals\client.js`
- `ui-modals\index.js`
- `ui-modals\package.json`
- `ui-quick-exec\client.js`
- `ui-quick-exec\index.js`
- `ui-quick-exec\package.json`
- `ui-right-panel\assets\favicon.svg`
- `ui-right-panel\client.js`
- `ui-right-panel\index.js`
- `ui-right-panel\package.json`
- `ui-sidebar\assets\favicon.svg`
- `ui-sidebar\assets\ui-sidebar.css`
- `ui-sidebar\assets\ui-sidebar.js`
- `ui-sidebar\client.js`
- `ui-sidebar\index.js`
- `ui-sidebar\package.json`
- `ui-statusbar\assets\favicon.svg`
- `ui-statusbar\assets\ui-statusbar.css`
- `ui-statusbar\assets\ui-statusbar.js`
- `ui-statusbar\client.js`
- `ui-statusbar\index.js`
- `ui-statusbar\package.json`
- `ui-statusbar-conn\client.js`
- `ui-statusbar-conn\index.js`
- `ui-titlebar\assets\favicon.svg`
- `ui-titlebar\client.js`
- `ui-titlebar\index.js`
- `ui-titlebar\package.json`

---

## .pair/assets/runtime/web（壳产物）

> 壳 js 文件名不同（DjWrYO0q vs CTZe2XhI）

### ▶ 仅 D 存在（1）— 覆盖同步将**删除**这些

- `assets\index-CTZe2XhI.js`

### ◀ 仅 E 存在（1）— D 缺失，同步将**新增**

- `assets\index-D6jSa2R5.js`

### ✗ 内容不同（1）

| # | 文件 | E 大小 | D 大小 | E独有行 | D独有行 | D 独有行示例 |
|---|---|---|---|---|---|---|
| 1 | `index.html` | 10772 | 10772 | 1 | 1 | <script type="module" crossorigin src="./assets/index-CTZe2XhI.js"></script> |

### ␍ 仅行尾差异（1）— 内容实质相同（CRLF↔LF），**不建议**为此覆盖

- `favicon.svg`

---

## .pair/assets/runtime（运行时运行时资产）

> bridge_node.js / cordis.bundle.js / README.md

### ▶ 仅 D 存在（1）— 覆盖同步将**删除**这些

- `web\assets\index-CTZe2XhI.js`

### ◀ 仅 E 存在（2）— D 缺失，同步将**新增**

- `README.md`
- `web\assets\index-D6jSa2R5.js`

### ✗ 内容不同（1）

| # | 文件 | E 大小 | D 大小 | E独有行 | D独有行 | D 独有行示例 |
|---|---|---|---|---|---|---|
| 1 | `web\index.html` | 10772 | 10772 | 1 | 1 | <script type="module" crossorigin src="./assets/index-CTZe2XhI.js"></script> |

### ␍ 仅行尾差异（3）— 内容实质相同（CRLF↔LF），**不建议**为此覆盖

- `bridge_node.js`
- `cordis.bundle.js`
- `web\favicon.svg`

---

## plugins-src/ui-app（UI 源码）

> D 含 setFocusMode/convListVisible 本地改动（未提交）


### ✗ 内容不同（7）

| # | 文件 | E 大小 | D 大小 | E独有行 | D独有行 | D 独有行示例 |
|---|---|---|---|---|---|---|
| 1 | `src\agent-events.js` | 51477 | 44451 | 131 | 40 | const wsPendingByConv = new Map()    // convId → { snapshot: null\|data, events: [] } ⏎ try { processAgentEvent(convId,  |
| 2 | `src\components\RightPanel.vue` | 164947 | 158852 | 51 | 18 | <!-- 阶段指示器（自主模式多阶段切换）+ 本次运行统计（耗时/步数/token 速度） --> ⏎ <span class="phase-text">{{ currentPhase \|\| (agentRunningConv ? '执 |
| 3 | `src\docs\api-docs.md` | 46470 | 46385 | 1 | 1 | > **安全提示**：所有 API 仅监听本地回环地址（127.0.0.1），默认不对外暴露。请勿将服务端口暴露到公网或局域网。 |
| 4 | `src\docs\faq.md` | 4663 | 4598 | 1 | 1 | 所有操作都在你的本地计算机上执行，代码和对话内容不会发送到外部服务器（AI 模型调用除外，你可以选择使用本地模型避免数据外出）。API 服务只监听本地回环地址，默认不对外暴露。文件操作限定在工作区范围内。 |
| 5 | `src\docs\features.md` | 13350 | 12973 | 2 | 2 | 所有 API 仅监听本地回环地址，安全可控。 ⏎ - **本地地址** — IDE 服务仅监听本地回环地址，默认不对外暴露 |
| 6 | `src\ShellApp.vue` | 22637 | 22130 | 1 | 0 |  |
| 7 | `src\ui-state.js` | 21604 | 20853 | 5 | 1 | runStatsByConv: {},        // { [convId]: { startAt, endAt, steps, toolCalls, llmCalls, promptTokens, completi |

### ␍ 仅行尾差异（48）— 内容实质相同（CRLF↔LF），**不建议**为此覆盖

- `package-lock.json`
- `public\favicon.svg`
- `src\api.js`
- `src\app-actions.js`
- `src\assets\logo.svg`
- `src\chat-utils.js`
- `src\components\ActivityBar.vue`
- `src\components\ApprovalBar.vue`
- `src\components\CodeEditor.vue`
- `src\components\ContextMenu.vue`
- `src\components\ConvSidebar.vue`
- `src\components\DebugLogPanel.vue`
- `src\components\EditorArea.vue`
- `src\components\FileExplorer.vue`
- `src\components\FileTreeItem.vue`
- `src\components\FindPanel.vue`
- `src\components\GitPanel.vue`
- `src\components\GlobalDialogs.vue`
- `src\components\HexViewer.vue`
- `src\components\ImageViewer.vue`
- `src\components\MarkdownRenderer.vue`
- `src\components\MarketplacePanel.vue`
- `src\components\Modal.vue`
- `src\components\PluginPanel.vue`
- `src\components\PresetManager.vue`
- `src\components\ProviderManager.vue`
- `src\components\SearchPanel.vue`
- `src\components\SettingsModal.vue`
- `src\components\Sidebar.vue`
- `src\components\SourceModal.vue`
- `src\components\StatusBar.vue`
- `src\components\SvgIcon.vue`
- `src\components\SystemModal.vue`
- `src\components\TaskPanel.vue`
- `src\components\UiEditor.vue`
- `src\components\UiRightPanel.vue`
- `src\main.js`
- `src\plugin-runtime.js`
- `src\ui-main-activitybar.js`
- `src\ui-main-editor.js`
- `src\ui-main-git.js`
- `src\ui-main-marketplace.js`
- `src\ui-main-modals.js`
- `src\ui-main-right-panel.js`
- `src\ui-main-sidebar.js`
- `src\ui-main-statusbar.js`
- `src\ui-main-titlebar.js`
- `vite.config.js`

---

## plugins-src/plugins（Go 插件源码）

> D 多 tool-debug/tool-git

### ▶ 仅 D 存在（10）— 覆盖同步将**删除**这些

- `tool-debug\impl\debug_tools.go`
- `tool-debug\main.go`
- `tool-debug\toolbin\registry.go`
- `tool-debug\toolbin\toolbin.go`
- `tool-debug\toolbin\util.go`
- `tool-git\impl\git.go`
- `tool-git\main.go`
- `tool-git\toolbin\registry.go`
- `tool-git\toolbin\toolbin.go`
- `tool-git\toolbin\util.go`

### ␍ 仅行尾差异（46）— 内容实质相同（CRLF↔LF），**不建议**为此覆盖

- `tool-binary\toolbin\registry.go`
- `tool-binary\toolbin\toolbin.go`
- `tool-binary\toolbin\util.go`
- `tool-bug\toolbin\registry.go`
- `tool-bug\toolbin\toolbin.go`
- `tool-bug\toolbin\util.go`
- `tool-codegraph\codegraph\builder.go`
- `tool-codegraph\codegraph\builder_go.go`
- `tool-codegraph\codegraph\builder_import.go`
- `tool-codegraph\codegraph\git_history.go`
- `tool-codegraph\codegraph\lang_builders.go`
- `tool-codegraph\codegraph\py_builder.go`
- `tool-codegraph\codegraph\query_advanced_test.go`
- `tool-codegraph\codegraph\search.go`
- `tool-codegraph\codegraph\sqlite_store.go`
- `tool-codegraph\codegraph\sqlite_store_test.go`
- `tool-codegraph\codegraph\store.go`
- `tool-codegraph\codegraph\tools.go`
- `tool-codegraph\codegraph\tsit\api.go`
- `tool-codegraph\codegraph\tsit\cursor.go`
- `tool-codegraph\codegraph\tsit\language.go`
- `tool-codegraph\codegraph\tsit\node.go`
- `tool-codegraph\codegraph\tsit\parser.go`
- `tool-codegraph\codegraph\tsit\query.go`
- `tool-codegraph\codegraph\tsit\tree.go`
- `tool-codegraph\toolbin\registry.go`
- `tool-codegraph\toolbin\toolbin.go`
- `tool-codegraph\toolbin\util.go`
- `tool-harness\toolbin\registry.go`
- `tool-harness\toolbin\toolbin.go`
- `tool-harness\toolbin\util.go`
- `tool-memory\toolbin\registry.go`
- `tool-memory\toolbin\toolbin.go`
- `tool-memory\toolbin\util.go`
- `tool-office\toolbin\registry.go`
- `tool-office\toolbin\toolbin.go`
- `tool-office\toolbin\util.go`
- `tool-project-info\toolbin\registry.go`
- `tool-project-info\toolbin\toolbin.go`
- `tool-project-info\toolbin\util.go`
- `tool-web\toolbin\registry.go`
- `tool-web\toolbin\toolbin.go`
- `tool-web\toolbin\util.go`
- `ui-quick-exec\toolbin\registry.go`
- `ui-quick-exec\toolbin\toolbin.go`
- `ui-quick-exec\toolbin\util.go`

---

## docs（文档）


### ␍ 仅行尾差异（8）— 内容实质相同（CRLF↔LF），**不建议**为此覆盖

- `cache_chain_analysis.md`
- `plugin-development.md`
- `plugin-round3-plan.md`
- `plugin-tools-overlap-audit.md`
- `plugin-verification.md`
- `round4-go-core-audit.md`
- `runtime-upgrade-plan.md`
- `runtime-upgrade-verification.md`

---

## config/skills（内置技能）


### ␍ 仅行尾差异（1）— 内容实质相同（CRLF↔LF），**不建议**为此覆盖

- `cordis-plugin-development\SKILL.md`

---

## assets（图标等）


### ␍ 仅行尾差异（1）— 内容实质相同（CRLF↔LF），**不建议**为此覆盖

- `icon.svg`

---

## scripts（脚本）

> D 只有 2 个构建脚本


### ◀ 仅 E 存在（25）— D 缺失，同步将**新增**

- `cdp-probe-eventstream.js`
- `cdp-shot-add-plugin.js`
- `cdp-shot-chat.js`
- `cdp-shot-toolset.js`
- `cdp-verify-chat-input.js`
- `cdp-verify-conv-tasks.js`
- `cdp-verify-runstats-backend.js`
- `cdp-verify-toolset-panel.js`
- `cdp-verify-toolset-tab.js`
- `cdp-verify-ws-gate-fallback.js`
- `cdp-verify.js`
- `cdp-ws-reconnect.js`
- `check_cycles.go`
- `deploy-to-install.mjs`
- `llm-trace-usage-report.mjs`
- `overwrite-install.mjs`
- `packager\main.go`
- `plugin-publisher.mjs`
- `probe-cache-hitrate.js`
- `publish-official-plugins.mjs`
- `sync-plugins-to-bin.mjs`
- `validate.sh`
- `verify-budget-cfg.mjs`
- `verify-no-maxiter.mjs`
- `verify-release.mjs`

### ✗ 内容不同（1）

| # | 文件 | E 大小 | D 大小 | E独有行 | D独有行 | D 独有行示例 |
|---|---|---|---|---|---|---|
| 1 | `build-ui.mjs` | 9421 | 8956 | 12 | 10 | // ★ Round3 ⑥.4 防堆积：构建前清理 cmd/companion/web-ui/dist 的历史 index-*.js ⏎ //   bundle（14 个历史产物问题根因：壳构建输出名带 hash，未引用旧包越积越多； ⏎  |

### ␍ 仅行尾差异（1）— 内容实质相同（CRLF↔LF），**不建议**为此覆盖

- `sync-web-dist.mjs`

---

## README.md（单文件）


### ␍ 仅行尾差异（1）— 内容实质相同（CRLF↔LF），**不建议**为此覆盖

- ``

---

## config/models → models（模型配置）

跳过：E=missing D=dir

---

## config/skills → bin?（二进制配置）

跳过：E=missing D=dir

---

## 🛡 不在同步范围（用户数据/运行时，勿覆盖）

- `.env`
- `config/settings.json`
- `config/mcp.json`
- `config/models.json`
- `config/roles`
- `config/philosophy`
- `config/skills(用户自定义)`
- `logs`
- `temp`
- `tools`
- `fonts`
- `lib`
- `pair.exe`
- `.pair/skills`
- `.pair/toolsets`
- `.pair/.ui-backup`
- `models`
- `bin`
