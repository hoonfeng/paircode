# PairCode UI 插件层：外部兼容分布式 UI 插件契约与重构规格

> 文档状态：**需求契约（contract spec）** —— 供 `frontend-plugins` / `frontend-shell` / `backend` 三路实现、
> `verifier` 逐条验收、`reviewer` 对照审查。本文只定义**契约与验收标准**，不含实现。
> 术语约定：**外部模型** = `@deepseek-ai/dsh-*` 分布式 client-ui 插件模型；
> **PairCode** = 本项目（当前 `plugins-src/ui-app`）。

---

## 0. 目标（一句话）

把 PairCode UI 层从「**单一集中 repo/脚本 + 宿主 slots 硬编码**」改造成「**每个 UI 区域/功能是独立、可构建、可发现、可版本化的分布式插件包**」，
并让布局对齐 外部的 **chat 优先薄壳**（对话为主视图、编辑器为辅助、默认隐藏、点文件树按需打开）。

外部实现：外部项目的 `packages/client/ui-*`（数十个独立 workspace 包）+ `dsh-client-web`/`dsh-client-web-frontend` 薄壳；
PairCode 现状：`plugins-src/ui-app`（单包）+ `plugin-runtime.js` 的 Slot 系统 + `build-ui.mjs`（一脚本产 7 区域 bundle）。

---

## 1. 概念对齐：外部模型 ↔ PairCode 现状 ↔ 目标契约

下表是整个重构的「词汇表」，后续所有章节都建立在这张映射表上。

| 概念 | 外部实现 | PairCode 现状 | 目标契约（本文） |
|---|---|---|---|
| 应用壳（shell） | `apps/web`(thin) → `dsh-client-web` boot kernel | `ShellApp.vue`（Vue 网格骨架）+ `main.js` | 薄壳：只渲染 `host.main` root 槽位 + 装载 client 插件；不再持有区域实现 |
| 插件形态 | 每个 `ui-*`/`dsh-*` 是一个 **npm workspace 包**（cordis plugin） | `plugins-src/ui-app/src` 单包 + `ui-main-*.js` 入口 | 每个 UI 区域/功能是一个**独立目录/独立构建目标**的插件包 |
| 插件 manifest | `package.json` `dsh.client.{inject,platform}` 声明 client 半 cordis entry | `plugin package.json` `{name,main,client,purpose,scope,type,version,config}` | **扩展为 外部兼容二段式**（见 §3）：`package.json`（宿主元数据）+ `dsh.ui`（client 半清单） |
| client 半代码 | 包内 `.tsx`/`.ts` client 半，经 **tsdown** 构建为 `lib/client.js` | `client.js`（`(ui)=>void`）经 `new Function` 求值 | 每包独立构建为 **IIFE/ESM bundle**（客户端零运行时编译，见 §4） |
| 插件发现 | host 组装 `window.__PLUGIN_BOOT__`（`WebBootGraph.entries[{id,url,rev,inject,immediately,external}]`） | `GET /api/plugins` 列表 + `GET /api/plugins/detail`（补 `clientCode`） | 保留现有 `/api/plugins*` 接口为**传输层**，新增 `GET /api/ui-boot` 输出 **外部兼容 boot 图**（见 §3.2） |
| client 装载 | `window.__ModuleLoader__` 注册 bundle factory（lazy CJS），cordis Loader `create` 激活 | `syncClientHalves(list)` 装载、`startPolling` 事件轮询 | 统一为「**装配器加载 bundle → 执行 client 半 → 触发槽位装配**」（见 §3.3） |
| 共享核心 | `window.__PLUGIN_BOOT__` 静态模块表（seed）：`react`/`react-dom`/`cordis`/`ui-slots`/`ui-primitives` | `window.__PAIRCODE_CORE`（Vue/uiState/api/pluginRuntime/agentEvents/actions） | **保留 `__PAIRCODE_CORE` 为唯一共享单例**，新增契约字段与版本锚（见 §4.3） |
| 槽位（slot） | `dsh-client-ui-slots` `SlotCore` + `SlotMap`（TS 类型化：kind/scope/owner/children） | `plugin-runtime.js` `registerSlot({slotId,kind,render})` + `clientSlots` 数组 | **对齐 Slot 语义轴**：`kind`(single/list)、`scope`(root/session/maybe)、状态持久化（见 §5） |
| 槽位声明 | 宿主 `AppFrame` 构造时把子槽写进 `children` 表（**声明即认领**，一个槽一位声明者） | 宿主预定义 `slotId` 列表（`getSlotOwner` 单选） | 薄壳 root 槽声明一组**具名子槽**，区域包向其注册（见 §5.2） |
| 编辑器 | `details` 列（`ui-conversation`/`ui-details` 槽位），**默认折叠 width 0，永不 unmount** | `editor` 槽位，默认与 chat 并排；`focusMode` 用 CSS `v-show` 隐藏 | **默认隐藏、点文件树按需打开**；折叠=`width:0`（保持挂载），见 §6 |
| 跨插件服务 | `ctx.layout`（`LayoutController`: toggleSidebar/openDetails/closeDetails） | `state.*` 全局响应式 + `window.$*` | 提供 `ctx.uiLayout`（或 `__PAIRCODE_CORE.layout`）面板转换服务（见 §3.4） |
| 构建 | tsdown per-package（`bundle` script） | `scripts/build-ui.mjs`（一脚本产 7 IIFE + 2 面板） | 拆分：**每区域独立构建目标 + 独立输出目录**（见 §4.1） |
| 生命周期 | cordis fiber（active/pending/waiting/...） | 插件状态 `stopped/running/waiting/rejected/failed/cancelled` | 沿用现有状态机（§3.5），client 半快照上报给 `cordis_frontend_inspect` |

> **核心结论**：PairCode 不需要改写为“cordis + React + TS”。需要的是**把现有 Slot 系统的『装配/协作』语义抽成与 外部对等的契约**（manifest、分区域构建、共享核心、chat 薄壳、编辑器按需），并**在现有 `/api/plugins*`/`__PAIRCODE_CORE` 基座上落地**。外部是“契约的语义参照”，不是“必须拷贝的技术栈”。

---

## 2. 范围与边界（in-scope / out-of-scope）

**In scope（本文契约覆盖）**
- ① UI 插件 manifest（发现/装载/服务）契约
- ② 每区域独立分布式插件包的构建模型与 `__PAIRCODE_CORE` 共享核心单例契约
- ③ chat 优先薄壳布局规格
- ④ 编辑器默认隐藏 + 文件树点击按需打开的装配默认值与交互状态机

**Out of scope（由其他任务/后续处理，见队内分工）**
- `frontend-plugins`（t4）负责**分布式插件包构建与装配**的实现落地
- `frontend-shell`（t5）负责**薄壳布局 + 编辑器按需打开**的实现落地
- `backend`（t3）负责 **manifest/发现/服务**的后端实现落地
- 本文**不改任何现有源码**，只定义契约与验收标准（供 t4/t5/t3 实现时对齐）。

---

## 3. ① 外部兼容 UI 插件 manifest / 发现 / 装载 / 服务契约

### 3.1 插件包 manifest（二段式）

现状（`plugin-development.md §13`）插件包结构：

```
.pair/plugins/<name>/
├── package.json     # {name, main, client?, purpose, scope, type, version, config}
├── index.js         # host 半源码（(ctx)=>void/对象形态）
└── client.js        # client 半源码（可选；(ui)=>void）【经 new Function 求值，运行时编译】
```

**目标契约**（新增 `dsh.ui` 段，兼容旧 `client` 字段；`client`/`client.js` 仍保留作 fallback）：

```jsonc
// .pair/plugins/<name>/package.json —— 外部兼容 manifest
{
  "name": "paircode-ui-editor",
  "version": "1.0.0",
  "type": "plugin",
  "main": "index.js",                 // host 半（保留）
  "purpose": "编辑器区域：CM6 + 终端",
  "scope": "global",                  // 含 client 半的 UI 类插件默认 global
  // ── 新增：外部兼容 client 半清单（等价 dsh-client 的 dsh.client + cordis entry）──
  "dsh": {
    "ui": {
      "platform": "web",              // 客户端平台（恒 "web"）
      "slot": "editor",               // 本包注册到哪个薄壳槽位（target slot，见 §5）
      "kind": "single",               // single | list（对齐 外部 SlotKind / 现 registerSlot kind）
      "scope": "root",                // root | session | session-maybe（数据作用域）
      "inject": [                     // client 半的 cordis/服务依赖（对齐 外部 dsh.client.inject）
        "@paircode/core",             // 共享核心单例（见 §4.3）——所有 UI 包必带
        "@paircode/layout"            // 薄壳布局服务（可选：需要面板转换时声明）
      ],
      "immediately": true             // 是否 stage-one 预取（对齐 外部 WebBootEntry.immediately）
    }
  },
  "config": { /* 装配参数，apply(ctx, config) 第二参 */ }
}
```

**必须遵守的不变式**
- `package.json` 的 `name` 是唯一装配 key（= 插件/插件包名），缺 `name`/`main` 即视为无效包（沿用现有 `LoadGlobalPlugins` 判定）。
- 含 `dsh.ui` 段 → 服务端必须标记 `hasClient=true` 并把 `clientCode` 提供到列表/详情（沿用现有字段）。
- `dsh.ui.slot` 是**本包对薄壳声明的目标槽位**，一个包只声明一个 `target slot`（primary），可通过 `dsh.ui.subSlots` 声明它**向下开放的子槽**（见 §5.2，声明即认领，一个子槽一位认领者）。

### 3.2 发现（discovery）：新增 外部兼容 boot 图

现状发现链路：`GET /api/plugins`（列表，含 `hasClient`/缺省不带完整 `clientCode`）→ 前端对 `hasClient && !clientCode` 者补 `GET /api/plugins/detail?name=`。

缺陷：① 列表/详情两次往返；② 无 `rev` 无 cache-busting；③ 无 `inject/immediately` 排序依赖，区域包之间无声明式装配顺序。

**目标契约**：新增 `/api/ui-boot`，输出 外部 `WebBootGraph` 等价结构（节点为「已装配完成的 UI 区域/功能包」，供薄壳一次性装载）：

```jsonc
// GET /api/ui-boot 返回
{
  "rev": "a1b2c3...",                 // 全图一致性锚（内容+各 bundle hash 的摘要）
  "entries": [                        // 按模块图顺序（被 external 依赖的包排前）
    {
      "id": "paircode-ui-editor",     // == package.json name
      "url": "/plugins-assets/paircode-ui-editor/assets/ui-editor.js?rev=a1b2c3", // 客户端 bundle 端点
      "rev": "deadbeef",              // 该 bundle 内容 hash（cache-busting 一致性锚）
      "inject": ["@paircode/core", "@paircode/layout"],  // cordis 服务依赖（信息性/装配等待依据）
      "immediately": true,            // stage-one 预取标记
      "external": ["@paircode/core"]  // 非基线模块说明符：该 bundle 从模块表 resolve（= __PAIRCODE_CORE 词条）
    }
  ]
}
```

`/api/ui-boot` 由后端（t3）从「已装配的区域包清单 + 各包 `dsh.ui` 段 + 各 bundle 内容 hash」组装。前端薄壳**只消费 `/api/ui-boot` 一张图**，不再逐包拼 `listPlugins`+`detail`。

**验收对应**：外部的 `window.__PLUGIN_BOOT__` 由 host 组装、`window.__ModuleLoader__` 注册 bundle factory；PairCode 等价物是 `/api/ui-boot` + `loader.import(url)`。二者结构字段一一对应（`id/url/rev/inject/immediately/external`）。

### 3.3 装载（load）：统一“装配器”

现状装载（`plugin-runtime.js syncClientHalves`）：
1. `loadAssemblyFile()` 合并磁盘装配状态 → 2. 拉列表+补 `clientCode` → 3. `syncClientHalves`（`new Function` 求值 `(ui)=>` 并 `registerSlot`/`registerPanel`）→ 4. `startPolling`。

**目标契约**：装载器通过 `/api/ui-boot` 拿到**有序** region 包清单，按 `external/immediately` 做预取与依赖排序；每个 region 包执行其 client 半并触发槽位装配。**关键并发不变式**：

- 共享核心 `__PAIRCODE_CORE` **先于**任何 region 包就绪（薄壳 `mount` 前注入）。
- 任一 region 包 client 半执行失败 → 该槽位保持**空态占位**（`slot-empty`），**不影响其他槽位**（故障隔离；沿用现有「渲染失败 → 占位提示」语义）。
- 装载完成后上报 client 半快照（`buildSnapshot`）给 `cordis_frontend_inspect`，供 Agent inspect 排查。

### 3.4 服务（service）：跨插件能力契约

现状跨插件共享：`ctx.provide/ctx.get`（host 侧）、`ctx.registerClientMethod`+`ui.invoke`（host→client RPC）、事件桥（`ui.emit`/`ctx.on('host:*')`）、`ui.http`（受限 API）。

**目标契约**（对齐 外部 `ctx.layout` 的 `LayoutController`）：**薄壳提供 `ctx.uiLayout` 服务**，覆盖区域包需要的所有面板转换（这是“编辑器按需打开”的服务支撑，见 §6）：

```ts
interface UiLayoutService {
  toggleSidebar(): void                       // 折叠/展开侧栏（文件树）
  openEditor(filePath?: string): void          // 打开编辑器/details 列（默认 file=当前 active）
  closeEditor(): void                          // 关闭编辑器/details 列（保持挂载，width:0）
  toggleEditor(): void                         // 往返
  isEditorOpen(): boolean
  // 可选：detailsColumnWidth()/setDetailsWidth(px) 供拖拽
}
```

区域包 client 半通过 `__PAIRCODE_CORE.layout`（或注入的 `@paircode/layout` 服务）调用；**不允许**区域包直接改 `state` 私有字段的布局/编辑器开关（封死在服务内，便于统一状态机与持久化）。

Host 侧仍按现约 (`ctx.provide('uiLayout', ...)` + `ctx.registerClientMethod`) 暴露；host→client 通过注入 layout 服务，client→host 通过 `ui.invoke('@paircode/shell', 'openEditor', {path})`。

### 3.5 manifest / 发现 / 装载 / 服务 —— 验收标准

| # | 验收标准（verifier 逐条核对） | 判据 |
|---|---|---|
| M1 | `/api/ui-boot` 返回 外部兼容 boot 图（`rev` + `entries[{id,url,rev,inject,immediately,external}]`），字段名与 外部 `WebBootGraph` 一一对应 | 返回图可对照 外部字段逐项对齐；无缺字段 |
| M2 | 每个 UI 区域/功能包的 `package.json` 含 `dsh.ui` 段（platform/slot/kind/scope/inject/immediately），可选 `subSlots` | 抽查 ≥3 个现区域包均有 `dsh.ui`；旧 `client` 字段仍合法（fallback） |
| M3 | 含 `dsh.ui` 段的包 → 服务端 `hasClient=true` 且 `clientCode` 被提供 | `/api/plugins` 列表 `hasClient` 正确、`/api/plugins/detail` 含 `clientCode` |
| M4 | 薄壳仅消费 `/api/ui-boot` 一张图装载 region 包（不再逐包拼 listPlugins+detail） | 前端装载主路径无 listPlugins+detail 合并逻辑；或该逻辑已封装为 `boot()` 单一入口 |
| M5 | 共享核心 `__PAIRCODE_CORE` 先于任何 region 包就绪；任一 region 失败只影响其槽位（空态占位） | 故障注入：disable 某区域包 → 其余槽位正常渲染 |
| M6 | `ctx.uiLayout` 服务提供 `openEditor/closeEditor/toggleEditor/isEditorOpen`，区域包通过它而非直接改 `state` 开关布局 | 布局开关只经服务；代码搜索无 region 包直接写 `state.*Editor*` 私有开关 |

> 对照 外部依据：`packages/client/modules/src/client/manifest.ts`（boot 图结构）、`packages/client/ui-sidebar/package.json`（`dsh.client` 段）、`packages/client/ui-layout/src/client/service.ts`（`ctx.layout` 服务面）。

---

## 4. ② 每区域独立分布式插件包构建模型 + `__PAIRCODE_CORE` 单例契约

### 4.1 分区域独立构建目标（对齐 外部每包独立 tsdown）

现状（`scripts/build-ui.mjs`）用一个脚本循环 7 区域 + 2 面板，全部产 IIFE 到 `.pair/plugins/ui-<region>/assets/`。**问题**：① 改一个区域要重跑全量；② 区域包之间无各自版本/声明；③ 区域实现强耦合在 `plugins-src/ui-app/src` 单目录。

**目标契约**：**一区域 = 一独立构建目标 = 一独立输出目录 = 一独立可版本化插件包**。

- 每个区域包源码独立（如 `plugins-src/ui-regions/editor/` 或保留 `plugins-src/ui-app/src/ui-main-editor.js` 为入口），但**构建按包产出**，输出到 `.pair/plugins/ui-<region>/assets/`（与 外部的 `/plugins/<id>/client.js` 端点等价，经现有 `/plugins-assets/ui-<region>/assets/*` 提供）。
- 构建产物**固定命名**：`ui-<region>.js` + `ui-<region>.css`（沿用现有 `entryFileNames/assetFileNames` 约定），`rev`=内容 hash。
- **构建目标表**（示例，具体实现由 frontend-plugins 定）：

| region | 入口 | 输出目录 | 产物 |
|---|---|---|---|
| `titlebar` | `src/ui-main-titlebar.js` | `.pair/plugins/ui-titlebar/assets/` | `ui-titlebar.js` |
| `activitybar` | `src/ui-main-activitybar.js` | `.pair/plugins/ui-activitybar/assets/` | `ui-activitybar.js` |
| `sidebar` | `src/ui-main-sidebar.js` | `.pair/plugins/ui-sidebar/assets/` | `ui-sidebar.js` |
| `editor` | `src/ui-main-editor.js` | `.pair/plugins/ui-editor/assets/` | `ui-editor.js` |
| `right-panel`（= chat） | `src/ui-main-right-panel.js` | `.pair/plugins/ui-right-panel/assets/` | `ui-right-panel.js` |
| `statusbar` | `src/ui-main-statusbar.js` | `.pair/plugins/ui-statusbar/assets/` | `ui-statusbar.js` |
| `modals` | `src/ui-main-modals.js` | `.pair/plugins/ui-modals/assets/` | `ui-modals.js` |
| `git-api` | `src/ui-main-git.js` | `.pair/plugins/git-api/assets/` | `git-panel.js` |
| `marketplace` | `src/ui-main-marketplace.js` | `.pair/plugins/marketplace/assets/` | `marketplace-panel.js` |

> 目标：任何区域包可独立 `build`、独立升级（`rev` 变化被 `/api/ui-boot` 感知）、独立被 `/api/ui-boot` 剔除/加入（发现层幂等）。

### 4.2 共享核心 `__PAIRCODE_CORE` 单例契约

现状（`build-ui.mjs` externals）已把 `vue`/`ui-state.js`/`api.js`/`plugin-runtime.js`/`agent-events.js`/`app-actions.js` external 到 `window.__PAIRCODE_CORE.*`，保证 `reactive` 状态与槽位注册表跨壳/UI bundle 副本一致（`plugin-runtime.js` 用 `window.__SLOT_REGISTRY` 同源数组实现跨副本共享）。

**目标契约（强化为正式对外 API，供第三方区域包依赖）**：

```
window.__PAIRCODE_CORE = {
  Vue,                      // Vue 单例（reactive 互通）
  uiState,                  // 全局响应式状态（ui-state.js，含 layout 面板开关的权威只读面）
  api,                      // 受限后端 API
  pluginRuntime,            // 槽位/面板注册表（同一 clientSlots/clientPanels/instances）
  agentEvents,              // Agent 事件流
  actions,                  // app-actions（全局动作）
  layout,                   // ★ 新增：ctx.uiLayout 服务面（见 §3.4/§6）
  version: "1.0.0",         // ★ 新增：共享核心契约版本锚，供 region 包做兼容检测
  bootGraph: { rev, entries } // ★ 可选：作为 /api/ui-boot 的浏览器侧缓存（加载自 boot()）
}
```

**必须遵守的不变式**
- **单例关键**：`__PAIRCODE_CORE` 内共享的 `pluginRuntime`/`uiState` 必须在**薄壳与所有 region 包**中指向**同一对象实例**（对齐 外部 `seed.ts` 的“每个 bundle 看到同一实例”，即 外部用 `<script>`/静态模块表保证单例；PairCode 用 `window.__PAIRCODE_CORE`）。
- region 包 **不打包** `__PAIRCODE_CORE` 的词条（external），运行时从 `window.__PAIRCODE_CORE` 取。
- `version` 锚：`__PAIRCODE_CORE.version`（或 `/api/ui-boot` 的 `rev`）变化时，region 包可做兼容检测；不匹配 → 该 region 报**兼容性错误**（`reportFailure('boot', ...)`），不影响其他 region。
- **禁止** region 包二次 `createApp` 共享核心或再打包一份 `ui-state.js`（会分裂 reactive 状态）。

### 4.3 部署/装载产物路径

- region 包 bundle 打到 `.pair/plugins/ui-<region>/assets/`，由 `/plugins-assets/ui-<region>/assets/ui-<region>.js` 提供（现有 `web_server.go` 的 `/plugins-assets/` 路由已支持）。
- 薄壳（web-ui dist）由 `vite.config.js` 独立构建，含 `__PAIRCODE_CORE`（现约）；区域 bundle 不含。
- `rev`：每个 bundle 内容 hash 作为 cache-busting 一致性锚；`/api/ui-boot` 的图级 `rev`＝所有 bundle hash 的摘要（外部 `WebBootGraph.rev` 语义）。

### 4.4 构建模型 —— 验收标准

| # | 验收标准 | 判据 |
|---|---|---|
| B1 | 每个 UI 区域/功能为独立构建目标，产出到独立目录（`.pair/plugins/<region>/assets/`）且命名固定 | `npm run build:ui`（新脚本）可只构建/产出单区域；产物名 `ui-<region>.js`/`ui-<region>.css` |
| B2 | region 包不打包 `__PAIRCODE_CORE` 词条，运行时从 `window.__PAIRCODE_CORE` 取 | 区域 bundle 反查无对 `vue`/`ui-state`/`plugin-runtime`/`api` 的重复打包；运行期 `__PAIRCODE_CORE` 单例存在且 region 正常渲染 |
| B3 | `__PAIRCODE_CORE` 提供 `version` 锚与 `layout` 服务面 | 运行期 `window.__PAIRCODE_CORE.version` 存在；`__PAIRCODE_CORE.layout.*` 可用 |
| B4 | 任一 region bundle 内容变化 → `/api/ui-boot` 该 entry `rev` 变化，图级 `rev` 摘要同步变化 | 改动一个 region 源码并重建 → `/api/ui-boot` 仅该 entry `rev` 变化 |
| B5 | 区域包可独立从 `/api/ui-boot` 移除（发现层幂等） | 删除某 region 包目录并重启 → `/api/ui-boot` 无该 entry，对应槽位空态占位，其余正常 |

> 对照 外部依据：`packages/client/ui-sidebar`、`packages/client/ui-layout` 各为独立包（tsdown build）；`packages/client/web/src/seed.ts` 静态模块表单例（外部版 `__PAIRCODE_CORE`）。

---

## 5. ③ chat 优先薄壳布局规格

### 5.1 现状（4 列 IDE 网格）→ 目标（chat 优先薄壳）

**现状**（`ShellApp.vue` grid）—— 4 列 IDE：

```
┌─────────┬────────────┬──────────────────┬──────────────┐
│ titlebar (col1 / -1)                │
├─────────┼────────────┼──────────────────┼──────────────┤
│ activity│ sidebar    │ editor(主,min340)│ right-panel  │
│ bar 48  │            │                  │ (chat + conv)│
│         │            │ ──────────────── │              │
│         │            │ bottom terminal  │              │
├─────────┴────────────┴──────────────────┴──────────────┤
│ statusbar (col1 / -1)                                  │
└─────────────────────────────────────────────────────────┘
```
`grid-template-columns: 48px auto minmax(340px,1fr) var(--right-w,525px)`。chat 在**最右**、编辑器为**主区**；`focusMode` 用 CSS `v-show` 隐藏编辑器（保留 DOM）。

**目标（chat 优先薄壳，对齐 外部 AppFrame 三列）**—— **对话为主视图（居中）**，编辑器为**辅助面板（默认隐藏，右侧/或覆盖）**：

```
┌────────────────────────┬────────────────────────┬─────────────┐
│ sidebar(session/文件树) │  conversation(chat,主)  │ details     │
│ width 280 (折叠=56 rail)│  center —— 主视图       │ (editor)    │
│                        │                        │ width 0 默认│
│                        │                        │ 折叠(details│
│                        │                        │ 永不 unmount│
│                        │                        │ 点文件树打开)│
├────────────────────────┴────────────────────────┴─────────────┤
│  overlay（浮动层：toast/approval/badge，fixed 不占 grid）      │
└────────────────────────────────────────────────────────────────┘
```

**结构对应（外部 AppFrame 子槽 → PairCode 目标）**：

| 外部槽位 | 外部语义 | PairCode 目标槽位 | 说明 |
|---|---|---|---|
| `root` | 壳唯一渲染槽 | `host.main`（薄壳 root） | 壳只渲染这一个；根框架/几何在这声明 |
| `sidebar` | 会话/目录树 | `sidebar` | 文件树 + 会话/工作区（默认宽 280） |
| `conversation` | **对话主视图**（center） | `conversation` | **chat 优先**：主列 |
| `details` | **编辑器/details**（辅助） | `editor` | **默认折叠 width 0**；点文件树打开 |
| `shell.overlay` | 浮动层 | `overlay` | toast/approval/fixed |
| （外部无需） | — | `titlebar`/`activitybar`/`statusbar` | 保留为壳级薄条（可继续插件化，非主视图） |

### 5.2 薄壳 = 纯几何骨架 + root 槽位（声明即认领）

`host.main`（root 槽）由薄壳在构造时声明一组**具名子槽**（声明即认领，一个子槽一位声明者——对齐 外部 `AppFrame` 的 `children` 表 / `SlotCore.register` 的 `children` 声明）：

```
host.main.children := {
  'sidebar':    { kind: 'single', scope: 'root' },
  'conversation':{ kind: 'single', scope: 'root' },
  'editor':     { kind: 'single', scope: 'root' },   // 默认折叠
  'overlay':    { kind: 'list',   scope: 'root' },
  'titlebar':   { kind: 'single', scope: 'root' },
  'activitybar':{ kind: 'single', scope: 'root' },
  'statusbar':  { kind: 'single', scope: 'root' },
}
```

region 包用 `registerSlot({ slotId, title, render, kind, scope })` 向某子槽注册占用者（`slotId`==上述 `host.main` children key）；`kind='single'` 单选（面板切换）、`kind='list'` 叠加（overlay）。**薄壳每个子槽一个挂载点**：owner 非空 → 渲染 region 到 `hostRef`；owner 空 → `slot-empty` 占位（含恢复入口）。

> 对齐点：外部 `SlotMap` 用 TypeScript 声明 merge 表达“一个槽一位声明者”；PairCode 用薄壳 root 的 `children` 表 + `registerSlot(slotId)` 表达同样约束。**几何与子槽声明**是壳的职责；**内容**是 region 包的职责。

### 5.3 布局验收标准

| # | 验收标准 | 判据 |
|---|---|---|
| L1 | 加载后**对话（conversation）为主视图**，位于主列/居中；编辑器为辅助且**默认不展示** | 首屏无编辑器占据主区；对话占主列（宽≥某阈值） |
| L2 | 薄壳只渲染 root 槽（`host.main`），其为纯几何骨架（不含任何区域实现） | 壳源码无区域具体组件实现；全部区域经 `registerSlot` 装配 |
| L3 | 薄壳产物不含 `__PAIRCODE_CORE` 之外的区域实现（薄壳 = 壳） | 壳 bundle 无 `CodeEditor`/`TerminalPanel` 等区域组件 |
| L4 | 折叠侧栏为紧凑 rail（对齐 外部 SIDEBAR_COLLAPSED=56），展开默认 280 | 折叠态为图标列；展开还原 280 或用户偏好 |
| L5 | 有 `conversation`/`editor`/`sidebar`/`overlay` 具名子槽声明（声明即认领） | 薄壳 root `children` 表包含这些 key；region 包向其注册 |
| L6 | 视图切换不触发整个 app 重挂（仅区域状态变化） | 打开/关闭编辑器、折叠侧栏 → 无整页刷新、无其余槽位重挂 |

> 对照 外部依据：`packages/client/ui-layout/src/client/AppFrame.tsx`（三列 grid + `renderSlot('conversation'|'details'|'sidebar'|'shell.overlay')`）、`columns.ts`（默认几何：SIDEBAR_DEFAULT=280、SIDEBAR_COLLAPSED=56、DETAILS_DEFAULT=360、CENTER_MIN=640）。

---

## 6. ④ 编辑器默认隐藏 + 文件树点击按需打开：默认值与交互状态机

### 6.1 关键坑背景（必须规避）

- 编辑器区域含 **CM6**（`CodeEditor.vue`）+ **终端**（`TerminalPanel.vue`，持有 **terminal WebSocket**）。
- `TerminalPanel.vue` 在 `onBeforeUnmount` 时 `term.ws.close()`；`useSingleSlot`（plugin-runtime.js 注释）已明确历史坑：**复杂槽位（ui-editor）卸载重挂 → 终端 WebSocket 断开重建，用户可见“切到插件面板终端 WS 断开、疯狂重建”**。
- **结论**：编辑器**折叠 = 隐藏（保持挂载）**，**绝不 unmount**。当前 `focusMode` 用 CSS `v-show`（保留 DOM）就是为此；目标契约延续并强化为「**默认折叠 width 0，永不 unmount**」。

### 6.2 装配状态默认值

`state`（`ui-state.js`）新增/复用一组**编辑器装配开关**（权威面只在 `ctx.uiLayout`/`__PAIRCODE_CORE.layout`，区域包经服务读写）：

```js
// ui-state.js 新增（默认值 = chat 优先、编辑器隐藏）
state.panels = {
  sidebarVisible: true,    // 文件树默认展开（宽 280）
  editorOpen: false,       // ★ 默认折叠（编辑器隐藏）
  editorWidth: 360,        // 折叠后打开时的默认详情列宽（对齐 外部 DETAILS_DEFAULT=360）
  editorLastWidth: 360,    // 上次打开宽（折叠还原用）
  rightPanelVisible: true, // 对话主视图常驻
}
```

**默认值语义**：
- `editorOpen=false`（**默认隐藏**）——首屏对话为主，编辑器不占空间。
- 与会话无关的**临时视图**（`editorOpen/editorWidth`）**不持久化**到 localStorage（沿用 `focusMode` 不持久化先例，避免“上次打开→下次一启动就显示编辑器”，也避免“跨会话残留 true”的经典坑）。
- `sidebarVisible`（文件树）可按需持久化（用户偏好），但键名与新名一致。

### 6.3 交互状态机（点文件树打开 / 关折叠保持挂载）

```
状态机状态：
  E0  editor=closed（折叠 width:0 / display:none；DOM 保持挂载，CM6+终端 WS 存活）
  E1  editor=opening（点文件树某文件 → 加载 file content → 请求 openEditor）
  E2  editor=open（width: editorWidth；显示 CM6 + 底部终端）
  E3  editor=opening_from_closed（从关闭态打开，仍保持挂载，仅切可见性）
  E_open→close  editor=closing（收窄 width → 0；不 unmount）

转化（事件驱动）：
  文件树文件 click（FileExplorer.openFile(path)）
      → state.activeFile=path；loadFileContent(path)
      → layout.openEditor(path)（若 editorOpen=false → 置 true，展示 CM6+终端；若已 open → 切换 tab/文件）
      → 【副作用】进入 E2；**期间不卸载既有 editor 子树**（仅更新文件内容/切 tab）

  编辑器关闭 X（UI 上可点收起）
      → layout.closeEditor() → E0（保持挂载；CM6 状态/终端 WS 保留）
      → 可选：保留当前 openFiles/activeFile 内存态（下次打开免重载），不持久化

  切换工作区（switchToWorkspace）
      → 清 openFiles/activeFile（现有行为）；editorOpen 保持 false（或按需要）

  侧栏折叠（toggleSidebar）
      → sidebarVisible 翻转；不影响 editor

约束（防坑）：
  ① editorOpen=false 时，编辑器子树【必须】保持挂载（CSS 隐藏，不 v-if）——绝不触发 CM6/终端卸载。
  ② 只在真正“换编辑器槽位 owner”时（插件装卸/切换）才 cleanup 重挂（对齐 useSingleSlot 的
     “owner 未变不重渲染”约束）。
  ③ 折叠/展开走 CSS（width/transform/visibility），不改 v-if；折叠态参与 grid 布局（width:0 不占额外空间）。
  ④ 打开文件不重新创建编辑器实例（CM6 会话/滚动/未保存 dirty 保留）；loadFileContent 遵守
     “fileDirty 时保留缓存不覆盖”。
```

**状态机不变量**
- `editor` 槽位 owner 唯一；`kind='single'`；owner 变化才重挂。
- 编辑器可见性（`editorOpen`）由 `ctx.uiLayout` 服务读写，是**独立于 `slotOwner` 的第二轴**（可见性≠占用者）。
- 折叠不 unmount 是对 CM6/终端 WS 的**硬约束**，验收要求终端 WS 在打开/关闭编辑器全程**不重建**。

### 6.4 编辑器按需打开 —— 验收标准

| # | 验收标准 | 判据 |
|---|---|---|
| E1 | **默认** `editorOpen=false`（编辑器隐藏）；首屏对话为主 | 首屏无编辑器占用主区；`state.panels.editorOpen===false` |
| E2 | 文件树点击某文件 → 打开编辑器并加载/切换该文件 | click 后编辑器可见、activeFile 正确、内容载入 |
| E3 | 关闭编辑器/收起 → 编辑器**保持挂载**（width:0/隐藏），**终端 WebSocket 全程不重建** | 打开→关闭→再打开：终端连接数/`term.ws` 连接不重建（可用 WS 计数断言；捕获多路 WS 判增长） |
| E4 | 编辑器可见性状态只经 `ctx.uiLayout` 服务读写（区域包不直接写 `state.panels.editorOpen`） | 代码搜索无 region 包直接写布局开关；全走服务 |
| E5 | `editorOpen/editorWidth` 为临时视图，不持久化到 localStorage | 关闭编辑器后刷新 → 编辑器仍隐藏 |
| E6 | 切换编辑器槽位 owner 才 cleanup/重挂；打开/关闭/切文件不触发重挂 | 换文件不产生 CM6 重建（可断言的编辑器实例/挂载计数） |
| E7 | 折叠态参与布局（width:0 不占额外空间），展开还原 `editorLastWidth` | 折叠后其余列不受挤压；展开还原偏好宽 |

> 对照 外部依据：外部 `details` 列**默认折叠且永不 unmount**（`columns.ts`：`details=0` 时 `width:0`，`AppFrame.tsx` `DetailsColumn` `width 0 keeps the subtree mounted (never unmount on close)`）；`LayoutController.closeDetails()` 折叠、`openDetails()` 打开。

---

## 7. 停机/回滚与兼容策略（供实现参考）

- **向后兼容**：保留 `client.js`（`(ui)=>void` 直载）与 `client` 字段作为 fallback；`dsh.ui` 段为**新增**，旧包（无 `dsh.ui`）仍按现约装载（`/api/plugins`+`new Function`）。新薄壳对两类包并存可装载。
- **增量迁移**：优先迁移 `editor`/`sidebar`/`right-panel(chat)` 三个主 region（影响聊天/编辑器主路径），其余逐步。
- **回滚**：任一 region bundle 出错 → 该槽位空态占位，其余正常；可整图回退到旧 shell 装载路径（`/api/plugins`+`clientCode`）。

---

## 8. 附录：字段/术语速查

| 术语/字段 | 定义 | 出处 |
|---|---|---|
| `dsh.ui` | 本包 client 半清单（platform/slot/kind/scope/inject/immediately/subSlots） | §3.1 |
| `/api/ui-boot` | 外部兼容 boot 图（`rev` + `entries[{id,url,rev,inject,immediately,external}]`） | §3.2 |
| `host.main` | 薄壳 root 槽（唯一渲染槽），其 `children` 表声明具名子槽 | §5.2 |
| `__PAIRCODE_CORE` | 共享核心单例（Vue/uiState/api/pluginRuntime/agentEvents/actions/layout/version） | §4.3 |
| `ctx.uiLayout` | 跨区域布局服务（toggleSidebar/openEditor/closeEditor/toggleEditor/isEditorOpen） | §3.4 |
| `editorOpen` | 编辑器可见性（独立于 slotOwner 的第二轴），默认 false | §6.2 |
| `rev` | bundle 内容 hash（cache-busting 一致性锚）；图级 `rev`=各 bundle hash 摘要 | §4.3 |

---

## 9. 附件：可执行验收清单（供 verifier 直接复制核对）

> 以下为「可验收标准」的**扁平化核对清单**，与 §3.5/§4.4/§5.3/§6.4 的验收表同构，verifier 逐条打勾。

```
[ ] M1  /api/ui-boot 返回 外部兼容 boot 图（rev+entries[{id,url,rev,inject,immediately,external}]）
[ ] M2  现区域包 package.json 含 dsh.ui 段（platform/slot/kind/scope/inject/immediately，可含 subSlots）
[ ] M3  含 dsh.ui 段 → hasClient=true 且 detail 返回 clientCode
[ ] M4  薄壳仅消费 /api/ui-boot 一张图装载 region 包
[ ] M5  __PAIRCODE_CORE 先于 region 包就绪；单 region 失败→仅该槽位空态占位
[ ] M6  ctx.uiLayout 提供 openEditor/closeEditor/toggleEditor/isEditorOpen；region 不直写布局私有开关
[ ] B1  每区域独立构建目标→独立输出目录，产物命名固定 ui-<region>.js/.css
[ ] B2  区域 bundle 不打包 __PAIRCODE_CORE 词条（运行时从 window.__PAIRCODE_CORE 取）
[ ] B3  __PAIRCODE_CORE.version 锚 + .layout 服务面存在
[ ] B4  改单 region → /api/ui-boot 仅该 entry rev 变化，图级 rev 同步
[ ] B5  移除单 region 包 → /api/ui-boot 缺该 entry，槽位空态，其余正常
[ ] L1  首屏对话为主视图，编辑器默认不展示
[ ] L2  薄壳只渲染 host.main(root)，其为纯几何骨架
[ ] L3  薄壳产物不含区域实现（无 CodeEditor/TerminalPanel）
[ ] L4  侧栏折叠=紧凑 rail(56)，展开=280 或偏好
[ ] L5  有 conversation/editor/sidebar/overlay 具名子槽声明（声明即认领）
[ ] L6  视图切换不整 app 重挂
[ ] E1  默认 editorOpen=false；首屏对话为主
[ ] E2  文件树 click 文件→编辑器打开并按需加载/切换
[ ] E3  编辑器关闭→保持挂载，终端 WS 全程不重建
[ ] E4  编辑器可见性只经 ctx.uiLayout 服务读写
[ ] E5  editorOpen/editorWidth 为临时视图，不持久化
[ ] E6  仅换槽位 owner 才重挂；开/关/切文件不触发 CM6 重建
[ ] E7  折叠参与布局(width:0)，展开还原 editorLastWidth
```
