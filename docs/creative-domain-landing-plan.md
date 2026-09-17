# 创作域落地计划（插件独立发布 + npm 市场同步）

> 日期：2026-09-12　前置：`docs/creative-domain-plan.md`（总方案）、`docs/voice-creation-plan.md`（人声子域详规）
> 本文只回答三件事：**(1) Go 层要不要动；(2) 插件怎么拆出来独立发布；(3) npm 滞后怎么收口。**
> 所有数字均为本文撰写时的实测值（附录 A 给出可复跑命令）。

---

## 0. 结论摘要

| 问题 | 结论 | 依据 | 代价 |
|---|---|---|---|
| Go 层要不要动？ | **不用动**（零改动，已验证能力面够用） | §2.1 十项能力逐条对照现有 `ctx.*` / 已有 HTTP 扩展面 | 无 |
| 能否完全插件化？ | **能**（host 半 + ui 半 = 一个自包含包） | 插件三形态 + `ctx.tools`/`ctx.http`/`ctx.fs` 已就位 | 首版 UI 为**独立面板**形态 |
| 会话内嵌 / 编辑器内视图？ | 需要**薄壳**扩展（E1/E2），**不是 Go 改动**，且不阻塞 P0 | `MarkdownRenderer.vue` / `EditorArea.vue` 均为前端 Vue | 可选增强，独立排期 |
| 插件能否独立版本、不跟随主程序发版？ | **可以**（机制已具备） | 各包已有独立 `version`；市场按包名装 latest；`dependencies` 非空自动走 Node 桥拉依赖 | 需补版本兼容锚（§3.3） |
| npm 滞后现状 | 本地 30 包 vs 线上 36 包：**22 版本漂移 + 6 未发布 + 12 孤儿** | §1.1 逐包对照表 | 需一次集中同步（§5） |
| 三个包本地版本低于线上 | `tool-binary`、`tool-bug`、`tool-office` | §5.1 | 必须先 bump（npm 不可降版本重发） |

---

## 1. 实测盘点

### 1.1 本地 30 个插件 × registry 逐包对照

线上查询：`GET https://registry.npmjs.org/-/package/@paircode/<name>/dist-tags`（逐包 30 次）。

| # | 插件 | 本地 | registry | 状态 |
|---|---|---|---|---|
| 1 | agent-teams | 1.1.0 | — | 未发布 |
| 2 | agentloop | 1.1.4 | 1.1.3 | 不一致 |
| 3 | core-api | 1.0.2 | 1.0.1 | 不一致 |
| 4 | fs-api | 0.1.4 | 0.1.3 | 不一致 |
| 5 | git-api | 0.1.3 | 0.1.2 | 不一致 |
| 6 | llm-trace | 1.0.0 | — | 未发布 |
| 7 | marketplace | 0.1.5 | 0.1.4 | 不一致 |
| 8 | tool-binary | 1.1.1 | **1.1.2** | **本地更低** |
| 9 | tool-bug | 1.0.1 | **1.0.2** | **本地更低** |
| 10 | tool-codegraph | 1.1.0 | 1.0.1 | 不一致 |
| 11 | tool-exec | 0.1.0 | — | 未发布 |
| 12 | tool-harness | 1.2.0 | 1.0.2 | 不一致 |
| 13 | tool-memory | 1.1.0 | 1.0.2 | 不一致 |
| 14 | tool-office | 1.0.2 | **1.0.3** | **本地更低** |
| 15 | tool-project-info | 1.1.0 | 1.0.2 | 不一致 |
| 16 | tool-resource | 1.0.0 | — | 未发布 |
| 17 | tool-scenario | 1.0.0 | — | 未发布 |
| 18 | tool-system | 1.1.0 | 1.0.2 | 不一致 |
| 19 | tool-web | 0.1.3 | 0.1.2 | 不一致 |
| 20 | tool-workflow | 1.0.0 | — | 未发布 |
| 21 | ui-activitybar | 0.1.3 | 0.1.2 | 不一致 |
| 22 | ui-appearance | 0.1.1 | 0.1.0 | 不一致 |
| 23 | ui-editor | 0.1.1 | 0.1.0 | 不一致 |
| 24 | ui-modals | 0.1.1 | 0.1.0 | 不一致 |
| 25 | ui-quick-exec | 1.0.1 | 1.0.1 | 一致 |
| 26 | ui-right-panel | 0.1.1 | 0.1.0 | 不一致 |
| 27 | ui-sidebar | 0.1.1 | 0.1.0 | 不一致 |
| 28 | ui-statusbar | 0.1.1 | 0.1.0 | 不一致 |
| 29 | ui-statusbar-conn | 0.1.0 | 0.1.0 | 一致 |
| 30 | ui-titlebar | 0.1.1 | 0.1.0 | 不一致 |

**三类问题（总计 40 项待处理）**

1. **版本漂移 22 个**：本地已改（多数超出线上 1 个以上 minor/patch），registry 未同步。
2. **未发布 6 个**：`agent-teams`、`llm-trace`、`tool-exec`、`tool-resource`、`tool-scenario`、`tool-workflow`。
3. **孤儿包 12 个**：本地目录已删，registry 仍在（见 §5.3 映射表）。
4. **危险项 3 个**：`tool-binary`(1.1.1<1.1.2)、`tool-bug`(1.0.1<1.0.2)、`tool-office`(1.0.2<1.0.3)
   —— 线上版本更高，**npm 不允许降版本重发**，直接发会 `EPUBLISHCONFLICT` 失败；§5.1 给出处置。

### 1.2 发布基础设施现状

| 组件 | 位置 | 作用 | 状态 |
|---|---|---|---|
| 官方批量发布 | `scripts/publish-official-plugins.mjs`（289 行） | 打包 `.pair/plugins/*` → `@paircode/<name>`；`--check` 查线上版本；内容指纹（src/artifact 分层）避免"改内容不 bump"漏发；自动 bump patch；发布前自动 `build-ui.mjs`；包间 15 s 冷却 | 可用，**3 处缺口**（§4.4） |
| 交互式发布器 | `scripts/plugin-publisher.mjs` | 独立单文件工具：token 管理（`--set-token`）、列表状态、Web UI（`--serve`）；可拷到任意 PairCode 项目 | 可用 |
| 安装镜像同步 | `scripts/sync-plugins-to-bin.mjs` | `.pair/plugins` → `bin/.pair/plugins`（安装包镜像） | 可用 |
| 市场（消费端） | `.pair/plugins/marketplace/index.js` | `searchNpmPlugins` 走 `/-/v1/search`：`@paircode/*` 前缀权威匹配 + `keywords:paircode` 全集兜底；安装经 `ctx.npm.install` | 可用 |
| 安装实现 | `internal/agent/npm_plugin.go` | `npmMarketInstall`：`dependencies` 非空 → Node 桥 + 真实 `npm install`；否则 goja 沙箱（只取 main） | 可用 |
| CI | `.github/workflows/ci.yml` | 仅 `go vet` / `go build ./cmd/companion` / 循环依赖检查 / `go test -short` | **无任何发布入口** |

> 结论：**"发布到插件市场"这件事不需要新建基础设施** —— 市场 = npm `@paircode` scope + `keywords:paircode`，
> 已有打包/发布/版本检测脚本。缺的是"独立版本的纪律"与 CI 入口。

---

## 2. Go 层改动量判定

### 2.1 能力逐项对照（创作域需求 → 现有接口 → 证据）

| # | 创作域需要 | 现有能力 | 判定 | 证据 |
|---|---|---|---|---|
| 1 | Agent 可用创作工具（`voice_analyze`…） | `ctx.tools.register({name,description,parameters,execute})` | **零改动** | `tool-web`/`tool-exec` 等 18 处使用 |
| 2 | 读音频/MIDI 字节（二进制安全） | `ctx.fs.readFileBase64` / `writeFileBase64`；`jsBytesOf` 支持 `Uint8Array`/`ArrayBuffer` | **零改动** | `jsplugin.go:1943-1966, 3215` |
| 3 | 写工程/素材文件 | `ctx.fs.readFile/writeFile/appendFile/stat/readdir/mkdir/rm`（工作区越界拦截） | **零改动** | `jsplugin.go` fs 服务 |
| 4 | 创作 UI（波形/画布/钢琴卷帘） | 浏览器侧自由 DOM/Vue + WebAudio（纯前端计算） | **零改动** | `ui.registerSlot/registerPanel` |
| 5 | 自定义 HTTP 端点（分片/流式/回调） | `ctx.http.register(method,path,fn)`（前缀匹配 `/api/ext/*`）、`ctx.webServer.register`、`ctx.sse`、`ctx.ws` | **零改动** | `internal/agent/ext_routes.go` + README §二进制插件协议 |
| 6 | 内置接口装配 | `ctx.kernel.routes()/install()`（82 条内核接口清单由 `core-api` 持有） | **零改动** | README「接口插件化」 |
| 7 | UI ↔ host 双向调用 | client `ui.invoke` ⇄ host `ctx.registerClientMethod`；`ui.emit`/`ui.on` | **零改动** | `plugin-runtime.js:410-418` |
| 8 | 设置项/命令/激活 | `ctx.registerSettings`、`ctx.commands`、`ctx.activation` | **零改动** | 各插件 7/5/5 处使用 |
| 9 | 大音频文件（数十 MB）流式传输 | fs 无 range 读；HTTP body `io.ReadAll` 无上限但全量进内存 | **需绕行**（浏览器 `File`/`FileReader` 本地直读，见 §2.3） | `jsplugin.go:1932-1966, 787` |
| 10 | 宿主 API 版本兼容校验 | 无（`__PAIRCODE_CORE.version` 锚存在但零使用点） | **需补**（插件侧自检即可，§3.3） | `main.js:36`；全仓 grep 零命中 |

### 2.2 结论

**Go 层零改动成立**：创作域（含人声子域）所需的工具注册、二进制读写、HTTP 扩展、UI 挂载、双向 RPC、
设置/命令/激活，全部由现有插件面覆盖。故**创作域插件可以独立成包、独立版本发布，不跟随主程序发版**。

### 2.3 两条"非 Go、但非纯插件"的边界（如实标注）

| 边界 | 影响 | 绕行（P0 可行） | 增强（可选） |
|---|---|---|---|
| **大音频文件**：`fs` 无 range 读、HTTP 无流式上限 | > 20 MB 文件经 goja 有内存风险 | 浏览器 `<input type=file>`/`FileReader` 直读本地（**已有先例**：`RightPanel.vue:730` 附件读取），处理完按需回写工作区 | 新增 `ctx.fs.readFileRange` 分片读（Go 小改，按需排期） |
| **UI 落点**：会话内嵌（` ```vocal `）与编辑器内视图需要薄壳注册表 | 首版只能在**独立面板**（`registerPanel`）或**整块占位**（`editor` slot）里呈现 | 用 `registerPanel` + `chat-tools`/`statusbar-items` 叠加槽位 | E1 渲染器注册表 / E2 编辑器视图注册表（`plugin-runtime.js` + `EditorArea.vue`，**前端薄壳改动**，纯加法） |

> 关键区分：E1/E2 改的是 `plugins-src/ui-app`（Vue 前端壳），**不是 Go 内核**；
> 且它只影响"嵌入宿主界面的深度"，不影响创作域插件能否独立发布与运行。

---

## 3. 插件拆分方案

### 3.1 为什么按"依赖形态"划线

`nodePluginRuntime()`（`internal/agent/node_plugins.go:129-138`）的判定：

| 形态 | 判据 | 安装行为 | 适用 |
|---|---|---|---|
| **goja 沙箱** | 无 `dependencies` 且无 cordis4 peer | 只取 `main` 文件注入沙箱，**不装 node_modules** | 纯 JS 逻辑插件（最适合独立发布） |
| **Node 桥** | `dependencies` 非空，或 `peerDependencies` 含 cordis ^4 | 真实 `npm install`（含 peer 显式安装）+ `plugins.json` 注册 | 需要 npm 生态依赖的插件（如 `@audio/*`） |

两种形态**都支持市场安装**，故创作域插件两种都能独立发布；区别只是安装体积与运行时开销。

### 3.2 创作域包清单（与总方案 §4.2 对齐）

| 域 | host 半 | ui 半 | 形态 | 依赖（许可） |
|---|---|---|---|---|
| 音乐 | `tool-music` | `ui-music` | goja（MIDI/ABC 文本处理）+ Node 桥（WAV 渲染） | `@tonejs/midi`(MIT)、Tone(MIT)、VexFlow(MIT) |
| 音乐·人声 | `tool-voice` | `ui-voice` | **Node 桥**（`@audio/*` 原子库） | `@audio/*`(MIT，40+ 包实测)、wavesurfer.js(BSD-3) |
| 矢量/图像 | `tool-art` | `ui-art` | goja + 浏览器画布 | Konva(MIT) |
| UI 设计 | `tool-design` | `ui-design` | goja | mermaid(MIT，已有) |
| 3D/CAD | `tool-model`（★ 2026-09-17 起 **UI 与工具同包**，原 `ui-model` 已并入） | —— | goja 零依赖（自研内核，已偏离本表原设想） | @jscad/modeling(MIT，仅测试交叉验证)、gltf-validator(官方) |
| 2D 角色 | `tool-rig` | `ui-rig` | goja + 浏览器 WebGL | ag-psd(MIT) |

**拆分粒度原则**

1. 一域两包（host/ui）——host 半可被 LLM 调用，ui 半只负责呈现，各自独立版本、独立修复；
2. 域之间零 import（不共享构建产物），只共享宿主契约（`ctx.*` / `@paircode/core`）；
3. 第三方库走 `/plugins-assets/<plugin>/vendor/*` 懒加载，不塞进宿主 bundle；
4. **首发试点选"音乐·人声"**：PoC 已实证六环跑通（`docs/research/voice-poc/`），风险最低、价值最直观。

### 3.3 版本与兼容策略（独立发版的必要纪律）

| 项 | 规则 |
|---|---|
| 版本号 | 各包独立语义化版本；**与主程序版本解耦**（不跟随 Packager 的 `version:bump`） |
| bump 判据 | patch=修 bug/文案；minor=新增工具或参数；major=**破坏性契约变更**（改包名、改工具名、改 manifest 契约、改 core 依赖） |
| 兼容锚 | 插件 `package.json` 声明 `dsh.minCore`（ui 半，对齐 `__PAIRCODE_CORE.version`）与 `dsh.minHost`（host 半 API 版本）；插件 `apply()` 开头自检，不满足则**降级到只读能力并提示升级**，不抛错炸宿主 |
| 现状缺口 | `main.js:36` 已有 `version: '1.0.0'`，但**全仓无任何校验点**（grep 零命中） → 独立发版后"新插件 + 旧宿主"会静默失败，必须补 |
| 依赖锁定 | Node 桥插件锁 `dependencies` 精确版本；关键原子库建议 vendor 到包内（`vendor/`），降低上游单点风险 |
| 不可降版本 | npm 不允许覆盖已发布版本 —— 发现"本地 < 线上"必须先 bump 到 `> 线上`（§5.1） |

### 3.4 升级路径（独立发版的另一半 —— 当前是缺口）

独立发版的前提是"用户能拿到新版"。实测：**能力在、产品面不在**。

| 环节 | 现状（实测） | 缺口 | 方案（零 Go 改动） |
|---|---|---|---|
| 单个 npm 插件更新 | `ctx.npm.checkUpdates(pkg)`（`jsplugin.go:3058-3066`）→ `npmPluginUpdate`（`npm_plugin.go:613`）：预查 latest → 卸载旧版 → `npmMarketInstall` 覆盖安装 | **全仓无调用方**（市场插件、UI、工具面均未触发）；且只认 `config.npm` 元数据 | marketplace 插件新增 `marketplace_updates` 工具 + 面板"可更新"徽标 / 一键更新 |
| 官方插件（随安装包分发） | 安装包含 `bin/.pair/plugins` 镜像作为**基线**；这些目录不是 npm 安装 → `config.npm` 为空 | `npmPluginUpdate` 对它们直接报"非 npm 来源或未安装" → **官方插件无法单独升级** | 面板对比"安装目录 `package.json#version`"与 registry `latest`：有新版 → `ctx.npm.install('@paircode/<name>')`（npm 安装 = 覆盖固化为同名磁盘插件包，重启装配） |
| 兼容守门 | 无 | "新插件 + 旧宿主"缺 API 时静默失败 | 安装后 `apply()` 自检 `dsh.minHost`/`dsh.minCore`：不满足 → 降级为只读能力 + 明确提示，不炸宿主 |

> 这意味着"插件独立发布"要落地，**必须同时落地升级面**（L1 验收项包含"安装目录基线插件可被市场新版覆盖"）。

---

## 4. 发布流水线

### 4.1 本地发布

```bash
# 只读盘点（不写 registry）
node scripts/publish-official-plugins.mjs --check
# 打包验证（npm pack --dry-run，不发）
node scripts/publish-official-plugins.mjs --only tool-voice
# 真实发布（token 由环境变量注入，见 4.3）
node scripts/publish-official-plugins.mjs --publish --only tool-voice
```

### 4.2 CI 发布（新增，当前完全空白）

新增 `.github/workflows/publish-plugins.yml`：

- 触发：`workflow_dispatch`（手填 `--only`），或主库打 `plugin-v*` tag；
- 步骤：checkout → setup-node(20, registry-url) → `npm ci`（UI 依赖）→ `node scripts/publish-official-plugins.mjs --publish --only "${{ inputs.only }}"`；
- 凭据：`NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}` —— setup-node 自动写 `.npmrc` 且用 `${NODE_AUTH_TOKEN}` 插值；
- 失败即停（脚本已 `process.exitCode = 1`），产物留 `npm pack` dry-run 日志便于回溯。

### 4.3 token 处理（不落盘，符合"只注入环境变量/CI secret"）

| 环境 | 做法 |
|---|---|
| 本地 | `.pair/publish/.npmrc` 只写占位符 `//registry.npmjs.org/:_authToken=${NPM_TOKEN}`（npm 支持 env 插值）；真实 token 仅存在于 shell 环境变量，**不写入仓库、不写入文档、不写入脚本** |
| CI | `secrets.NPM_TOKEN` → `NODE_AUTH_TOKEN`（GitHub 官方模式） |
| 仓库卫生 | `.pair/*` 已在 `.gitignore:23` 忽略 → `.npmrc`/`.content-hashes.json` 均不入库（已核对 `git check-ignore` 命中） |

### 4.4 发布工具缺口（本次要修）

| 编号 | 缺口 | 影响 | 修法 |
|---|---|---|---|
| T1 | 本地版本 < 线上时无处理 | 直接发必失败（`EPUBLISHCONFLICT`），当前只打印"重新打包并发布" | 检测到 `semver(local) < semver(online)` → 自动 bump 到 `online` 的下一 patch，并打印原因 |
| T2 | 无孤儿包检测/清理 | 删除插件后 registry 永久留垃圾（本次 12 个） | 新增 `--orphans`（对比 `.pair/plugins` 与 registry 中 `keywords:paircode` 全集）与 `--deprecate <pkg> --message <msg>` |
| T3 | token 硬编码路径 | CI/临时环境无法注入 | 支持 `PAIRCODE_NPM_USERCONFIG` 环境变量覆盖，或直接用 env 插值（4.3） |
| T4 | 无发布前 dry-run 报告 | 批量发布前无法一次看清"将发什么" | `--plan` 输出：将发布/将 bump/将跳过/孤儿 四类清单 |

---

## 5. npm 滞后处置（立即执行）

### 5.1 三类问题的处置矩阵

| 类别 | 数量 | 动作 | 命令 |
|---|---|---|---|
| 未发布 | 6 | 直接发布 | `--publish --only agent-teams,llm-trace,tool-exec,tool-resource,tool-scenario,tool-workflow` |
| 本地更高 | 19 | 发布（版本不同 → 脚本自动打包发布） | `--publish`（全量，已一致项自动跳过） |
| **本地更低（危险）** | 3 | **先核实内容 → bump 到线上+1 → 再发** | 见下 |
| 孤儿包 | 12 | `npm deprecate`（保留可下载，标注替代品） | §5.3 |
| 已一致 | 2 | 跳过（`ui-quick-exec`、`ui-statusbar-conn`） | — |

**三个危险包的处理纪律（先核实，再 bump）**

1. 下载线上 tarball，与本地目录逐文件 SHA-256 比对（`index.js` / `client.js` / `assets`）；
2. 若**内容等价**（仅版本号漂移）→ bump 到线上+1 重发（幂等归档，安全）；
3. 若**本地内容更旧**（线上含本地没有的修复）→ **不发**，先补回线上差异，再 bump；
4. 若**本地内容更新** → bump 到线上+1 发布（正常路径）。

### 5.2 发布顺序

1. `core-api`/`fs-api`/`git-api`（基础设施，先保证新宿主可用）；
2. `tool-*` 工具包；
3. `ui-*` 区域包（脚本发布前会自动 `build-ui.mjs`，避免发旧 assets）；
4. 新包最后（`agent-teams`/`tool-exec`/`tool-scenario` 等按需激活类）。

包间 15 s 冷却（脚本内置，npm 限流防护）→ 约 28 × 15 s ≈ 7 min。

### 5.3 孤儿包 deprecate 映射（12 个，均有代码/文档依据）

| registry 包 | 现状版本 | 处置消息（替代品） | 依据 |
|---|---|---|---|
| `tool-vision` | 1.0.2 | 并入 `@paircode/tool-web`（`read_image`） | `.pair/plugins/README.md:306` |
| `tool-vision-llm` | 1.0.0 | 同上（源已移除） | `.pair/project.md:1005` |
| `tool-screenshot` | 1.0.1 | 并入 `@paircode/tool-web` | `.pair/project.md:922` |
| `tool-web-debug` | 1.0.1 | 并入 `@paircode/tool-web` | `.pair/project.md:922` |
| `tool-shell` | 0.1.1 | 由 `@paircode/tool-exec` 取代（`exec_command`/`write_stdin`/`kill_process`） | `tool-exec/package.json` purpose |
| `tool-core` | 0.1.1 | 移除：`multi_edit`/`move_file`/`delete_file` 由 `apply_patch` 覆盖 | `.pair/plugins/README.md:150-152` |
| `tool-debug` | 1.0.3 | 移除（Round4.5，纯命令包装壳无独立价值） | `.pair/project.md:35` |
| `tool-git` | 1.0.2 | 移除：git 操作统一走命令执行 | `.pair/plugins/README.md:302` |
| `tool-verify` | 1.0.2 | 并入 `@paircode/tool-resource` | `.pair/project.md:905` |
| `tool-codegraph-extra` | 1.0.1 | 并入 `@paircode/tool-codegraph` | `.pair/project.md:907` |
| `host-capability-probe` | 1.0.1 | 移除（接口探针，验证后删除） | `.pair/plugins/README.md:304` |
| `web-api` | 1.0.0 | 移除（`/api/ext` 示例插件，能力由 `ext_routes` 覆盖） | `.pair/plugins/README.md:60-61` |

**deprecate 文案模板**：`不再维护：已并入 @paircode/<替代包>。请安装 @paircode/<替代包>（npm i @paircode/<x>）。`

**unpublish 不采用**（除用户明确要求）：npm 的 unpublish 在 24 h 后他人可抢注同 scope 名，且会破坏已装版本的可复现下载；
`deprecate` 保留下载 + 安装时警告，是更安全的归档方式。

### 5.4 回滚

| 场景 | 手段 |
|---|---|
| 发错内容 | `npm deprecate @paircode/<x>@<bad> "该版本有缺陷，请用 <good>"` + 立即发 patch 修正版（不可 unpublish 覆盖） |
| 插件导致宿主异常 | 市场卸载：`marketplace_install` 反向 `ctx.npm.uninstall`，或删 `<InstallDir>/.pair/plugins/<name>/` 后重启；主程序安装包内 `bin/.pair/plugins` 镜像不受 registry 影响 |
| 整个发布批次有问题 | 回滚 `main` 分支的 version bump 提交；registry 侧按包 deprecate 逐条标注（分批小步发布 = 天然限损） |
| 误删本地插件目录 | `.pair/publish/<name>/` 保有上次发布快照；`bin/.pair/plugins/` 保有安装镜像 → 可反向恢复（**注意**：publish 副本会漂移，禁止无脑反向覆盖 `.pair/plugins`） |

---

## 6. 落地分期与验收

| 期 | 内容 | 验收（可自动） | 预估 |
|---|---|---|---|
| **L0 发布基建** | 修 T1–T4；token env 化；新增 CI workflow | ① `--plan` 四类清单输出正确；② 3 个危险包内容比对有结论；③ CI workflow 在 `workflow_dispatch` 下 dry-run 通过 | 本次 |
| **L0+ npm 同步** | 28 包发布 + 12 包 deprecate | registry 与本地清单**逐包一致**（附录 A 脚本全绿）；每个包可 `npm view` 到新版本 | 本次 |
| **L1 独立发布试点** | `tool-voice` + `ui-voice`（人声 PoC 已实证）打包、独立版本、发布、市场安装验证 | ① 干净工作区 `marketplace_install @paircode/tool-voice` → 重启后工具可用；② `voice_verify` 六项自检通过；③ 宿主 Go 二进制**零重编译**（`git diff` 无 `internal/`、`cmd/` 变更） | 1–2 会话 |
| **L2 创作域 P0** | `tool-music` + `ui-music`（MIDI/乐谱/播放）；`tool-art` + `ui-art`（SVG 画布） | ① 会话产出 → 工作区文件闭环；② 校验器（SMF 解析 / SVG 合法性）通过；③ 截图 + `read_image` 视觉验证 | 2–3 会话 |
| **L3 薄壳增强（可选）** | E1 渲染器注册表（` ```vocal ` 等会话内嵌）+ E2 编辑器视图注册表 | ① 未注册时行为与今天完全一致（纯加法）；② 注册后 ` ```vocal ` 在会话内渲染并通过 `read_image` 验证 | 1 会话 |
| **L4 其余域** | `tool-model`（含 UI）、`tool-rig`/`ui-rig`、`tool-design`/`ui-design` | 逐域校验器 + 导出物断言 | 按需 |

**每期硬门槛**：`go vet ./...` + `go build ./cmd/companion` + `go test -short ./internal/...` 全绿；
插件侧 `node -e "require('./.pair/plugins/<x>/index.js')"` 语法检查 + 独立端口（非 9090）冒烟。

---

## 7. 风险与遗留

| 风险 | 级别 | 缓解 |
|---|---|---|
| 3 个"本地更低"包内容未知（可能降级线上） | **高** | §5.1 强制先比对 tarball 再决定发/不发 |
| 独立发版后"新插件 + 旧宿主"静默失败 | 中 | §3.3 兼容锚 + `apply()` 降级自检（L1 落地） |
| Node 桥插件依赖体积/上游单点（`@audio/*` 新包） | 中 | vendor 关键原子 + 精确版本锁；goja 优先 |
| 首版 UI 无会话内嵌（体验落差） | 低 | 独立面板可自足；E1/E2 排在 L3 |
| npm 限流（28 包连续发布） | 低 | 脚本 15 s 冷却；失败可 `--only` 断点续发 |
| `bin/.pair/plugins` 镜像与 registry 版本漂移 | 低 | 发布后跑 `node scripts/sync-plugins-to-bin.mjs`；文档标注"镜像=随安装包的基线，不是 latest" |

**遗留 / 未在本次范围**

- 大文件流式（`ctx.fs.readFileRange`）未做，靠浏览器本地直读绕行；
- E1/E2 薄壳扩展未做（L3）；
- 人声子域 PoC 仅合成素材实测，真实录音素材验收留待 L1；
- 创作域六域全部插件尚不存在（L2/L4 产出）。

---

## 附录 A 可复跑命令（本次盘点的真相源）

```bash
# A1 registry 逐包版本对照（本地 vs 线上）
node scripts/publish-official-plugins.mjs --check

# A2 registry 上全部官方包（含孤儿）——注意 search 索引可能滞后，用 keywords 兜底
curl -s "https://registry.npmjs.org/-/v1/search?text=keywords:paircode&size=250" \
  | node -e "let s='';process.stdin.on('data',d=>s+=d).on('end',()=>{const j=JSON.parse(s);console.log(j.total);for(const o of j.objects)console.log(o.package.name,o.package.version)})"

# A3 单包 dist-tags（判断是否已发布/最新版本）
curl -s "https://registry.npmjs.org/-/package/@paircode/tool-web/dist-tags"

# A4 仓库卫生核对（token 文件不入库）
git check-ignore -v .pair/publish/.npmrc .pair/publish/.content-hashes.json

# A5 插件形态判据（goja 沙箱 vs Node 桥）
grep -n "func nodePluginRuntime" -A 18 internal/agent/node_plugins.go
```

---

## 附录 B L0 执行记录（2026-09-12，实测）

### B.1 发布结果

| 项 | 结果 |
|---|---|
| 发布 | **30/30 成功，0 失败**（6 个新包 + 19 个更新 + 3 个自动升版 + 2 个已一致跳过） |
| 自动升版（本地 < 线上） | `tool-binary` 1.1.1 → **1.1.3**；`tool-bug` 1.0.1 → **1.0.3**；`tool-office` 1.0.2 → **1.0.4** |
| 新发布的 6 个 | `agent-teams@1.1.0`、`llm-trace@1.0.0`、`tool-exec@0.1.0`、`tool-resource@1.0.0`、`tool-scenario@1.0.0`、`tool-workflow@1.0.0` |
| 孤儿包 deprecate | **12/12 成功**（每条 registry `deprecated` 字段已核实，消息见 §5.3） |
| 终检 | **30/30 包 `本地 version === registry latest`** |
| UI 产物 | 发布前 `node scripts/build-ui.mjs` 全绿（9 个区域） |

### B.2 发布前的内容核实（3 个"本地更低"包）

下载线上 tarball 与本地逐文件 SHA-256 比对，结论 **本地内容更新**（不存在降级风险）：

| 包 | 工具集合 | 本地独有证据 |
|---|---|---|
| tool-binary | 线上 8 = 本地 8 | 本地含 2026-09-11「相对主项目根解析，跨项目请传绝对路径」文案 × 8 处（线上 0 处） |
| tool-office | 线上 11 = 本地 11 | 同上文案 × 10 处 |
| tool-bug | 线上 3 / 本地 2 | `bug_analyze` 为**有意削减**（`.pair/project.md:33`「已削减工具（勿恢复）」） |

> 附带修正：`tool-bug/index.js` 的 `purpose` 仍写 `bug_analyze`（陈旧，与头部注释矛盾）→ 已同步为 2 工具口径。

### B.3 修复的工具缺口（`scripts/publish-official-plugins.mjs`）

| 编号 | 实现 |
|---|---|
| T1 | 本地 < 线上 → 自动 bump 到「线上+1 patch」并写回源 `package.json`（实测 3 包生效，日志含 `⤴ ... 自动升到 X`） |
| T2 | 新增 `--orphans`（列出 registry 有/本地无的包）与 `--deprecate-orphans`（带替代品映射 `ORPHAN_HINTS` 执行 `npm deprecate`） |
| T3 | `PAIRCODE_NPM_USERCONFIG` 环境变量覆盖 userconfig（CI/临时注入），并**强制 resolve 为绝对路径** |
| T4 | 新增 `--plan`：一次输出「未发布 / 本地更高 / 本地更低 / 已一致 / 孤儿」五类清单（只读） |

> **`--orphans` 的检出语义（实测重要）**：它以 registry 的 `keywords:paircode` **搜索**为口径 ——
> npm 搜索索引会**排除已 deprecate 的包**（实测：deprecate 12 个前后，搜索 total 36 → 30，
> 与本地 30 个插件目录正好相等，`--plan` 的"孤儿"项归零）。因此：
> ① "已处理过的孤儿"不会重复报警（检出即闭环）；② 但**包本身仍在 registry 上**
> （`@paircode/tool-core` 等仍 HTTP 200，只是带 `deprecated` 标记）——
> 若确需物理删除只能 `npm unpublish`（本次不采用，理由见 §5.3）。

### B.4 两个实操坑（已修复/已规避）

1. **`--userconfig` 相对路径必然认证失败**：脚本 `execSync` 的 `cwd` 是目标包目录，
   相对路径 userconfig 解析错位 → `ENEEDAUTH`。已改为 `path.resolve(root, ...)` 强制绝对路径。
2. **后台发布的输出会被管道缓冲**：`... | grep -v ...` 使日志直到进程结束才可见，
   期间无法判断进度。规避方式：用**事实源**（registry `dist-tags`）做进度观察，而不是看日志。

### B.5 遗留（未在本次完成）

- 新增的 `.github/workflows/publish-plugins.yml` **尚未在真实 CI 上跑过**（本地无法验证 GitHub runner 行为）；
  首次使用建议以 `dry_run: true` 跑一遍确认 `--plan` 输出。
- §3.4「升级路径」三项（`marketplace_updates` 工具、官方插件覆盖升级、兼容守门）为 **L1 范围**，本次未实现。
- 本次发布未包含任何创作域插件（创作域插件尚不存在，属 L1/L2 产出）。

---

## 附录 C L1 执行记录（人声插件试点，2026-09-16 实测）

### C.1 交付物

| 文件 | 作用 |
|---|---|
| `.pair/plugins/tool-voice/package.json` | 包元数据 + **6 个 `@audio/*` 精确锁版本依赖**（判 Node 桥轨的关键） |
| `.pair/plugins/tool-voice/index.js` | 双轨入口：Node 桥注册 5 工具；goja 沙箱**运行时自检让位**（不注册任何工具） |
| `lib/wav.js` | WAV 读写（自研，零依赖）：PCM u8/i16/i24/i32 + IEEE float32/64 + EXTENSIBLE |
| `lib/fixture.js` | 确定性「人声样」素材合成（谐波堆 + 共振峰包络 + 颤音），供回归 |
| `lib/dsp.js` | 纯算法层：包络/静音、音符分段、切片后处理、多源候选聚类、谐波 Goertzel、互相关、时间映射 |
| `lib/analyze.js` | 分析段：f0(YIN) / notes / vad / slices / loudness |
| `lib/ops.js` | 编辑命令 + **分段线性 sourceMap 合成** |
| `lib/project.js` | 工程读写 + 渲染纯函数 + 段表摘要 |
| `lib/verify.js` | §7 六项自检（含判据分层，见 C.3） |
| `scripts/publish-official-plugins.mjs` | **白名单加 `lib`**（Node 桥插件多文件实现必须随包发布——tool-voice 是首个用 `lib/` 的插件，此前会静默丢包） |

### C.2 实测验收数据

素材：确定性合成 2.2s / 48 kHz 单声道（三段 220 / 246.94 / 289.2 Hz，5.5 Hz 颤音，14 次谐波）。
编辑链：`speech.removeSilence(minGap=0.3,keep=0.05)` → `time.warp(factor=0.85)` → `pitch.snap(minor, root=C, tolerance=12)`。

| 检查 | 实测 | 结论 |
|---|---|---|
| 7.1 依赖审计 | 声明 6 / 安装 18（含传递）/ 权重文件 0，禁止命中 **0** | PASS |
| 7.2 帧级可追溯 | 段表覆盖 **100%**；含改波形 op 时包络相关 **0.967**、有声段最低包络相关 0.909；**保波形链**逐段波形互相关 **min = 1.000** | PASS |
| 7.3 音色一致性 | H2 **0.4%** / H3 **1.2%** / H4·H5 **3.5%**（阈值 10/15/45%） | PASS |
| 7.4 确定性 | 两次渲染 **maxDiff = 0**（位级一致；审 WAV 文件时按同位深量化后同样为 0） | PASS |
| 7.5 链可见 | 3 个阶段全部带算法名+版本+参数 | PASS |
| 7.6 无新增音源 | 有声帧可解释率 **100%（158/158）**，能量偏差中位 **-0.02 dB** | PASS |

分析能力实测：YIN 检出 3 音符段（220.30 / 247.20 / 289.58 Hz，相对合成目标 **+1.8 ~ +2.3 ¢**，conf ≥ 0.99）；
VAD 7 段（3 有声 + 4 静音，边界 0.105/0.705/0.755/1.355/1.455/2.055 s 全部命中）；
切片：**多源候选 60 → 后处理 3**（恰好三个真实音节段）；响度 BS.1770-4 LUFS -14.14 / true peak -6.66 dBTP；
吸附后输出 f0 293.485 Hz vs D4 293.665 Hz（**-1.1 ¢**，优于 §9 要求的 ≤5 ¢）。

端到端链路验证（**非单元测试，真实运行时**）：`bridge.js` 装载 `@paircode/tool-voice@0.1.0`（runtime=node）
→ 5 工具进桥注册表 → 经 JSON 协议逐工具调用 `voice_import` / `voice_edit` / `voice_render` / `voice_verify`
全部 `ok=true`，`voice_verify` 在桥进程内 **6/6 PASS**，产物含 `.wav` + `.sourcemap.json` + `.recipe.json`。

### C.3 判据口径修正（重要，避免后续误读）

PoC 与方案 §7 的判据在实现时暴露出**物理不可达**之处，已按事实分层（数值如实上报，不掩盖）：

1. **7.2 不能对变调输出要求波形级互相关**：PSOLA/WSOLA 改变周期结构与相位，
   实测波形级逐段互相关中位 ≈ -0.02（噪声级）、逐样本相关 ≈ 0.002 —— 这不是溯源失败。
   故判据分层：**保波形链**（切片/停顿删减=样本级搬运）断言波形级 min ≥ 0.99（实测 = 1.000）；
   **含改波形 op 的链**断言结构级（包络相关 ≥ 0.95 且每个有声段包络相关 ≥ 0.9）。
   每个 op 的 recipe 现带 `waveformPreserving` 标记，判定自动选择口径。
2. **静音窗不参与互相关判定**：双静音窗方差为 0，"相关系数"数学上无定义（pearson 返回 0），
   计入会把保波形链误判为 min = 0。
3. **7.4 的位深量化必须先补偿**：审「已保存的 WAV」时，第二次渲染需按同位深量化后再比较，
   否则 24-bit 量化步长（≈1.2e-7）会被误报为"不确定性"（-127 dBFS）。
4. **7.1 审计口径限定为「本插件引入的东西」**：默认只扫插件自身目录 + 其依赖树；
   把整个用户工作区纳入会误报（工作区里别的工具的 `.onnx` 与本插件无关）。`auditWorkspace=true` 可加严。

### C.4 本次暴露的机制缺口（L1 的两条真实发现）

| # | 缺口 | 影响 | 最小修法 |
|---|---|---|---|
| **G1** | **磁盘插件扫描不识别运行时轨**：`LoadGlobalPlugins`（`toolset.go:630`）只按 `name`+`main` 判定，不检查 `dependencies` → 声明了 npm 依赖的包（Node 桥插件）会被当 **goja 插件**白装载一次（apply 由插件自检让位，但 `TestToolLandingMatrix` 要求每个 `tool-*` 目录在模式表声明 → CI 红） | 低（功能无损）～中（测试与整洁性） | `LoadGlobalPlugins` 里加 `if nodePluginNeedsNode(manifest) { continue }`（~5 行）+ 测试表加 `tool-voice` 项 |
| **G2** | **UI 半无法经市场安装交付**：`npmMarketInstall` 的 goja 分支只落 `index.js` + `package.json`（丢 `assets/`、`client.js`、`lib/`），且 `BuildUIBootGraph` 只扫 `.pair/plugins/`（不含 Node 桥目录） | 高（`ui-voice` 这类 UI 插件装不上） | goja 分支改为**整包复制**到 `.pair/plugins/<name>/`（而非只写 main），或 `BuildUIBootGraph` 兼扫桥目录 |

> 两条都属**插件装载/分发层**，与「创作能力是否需要改内核」无关：创作域能力本身由现有插件面覆盖，
> §2.2「Go 零改动」结论不受影响。但 G1/G2 若不修，Node 桥插件（人声链的关键形态）
> 在**官方随包分发**与 **UI 交付**两条路上都走不通，只能在"市场安装 host 半"这一条路上工作。

### C.5 G1 / G2 修复实现（本期，用户决策：两条都修）

**G1 —— 磁盘扫描按运行时轨过滤**（`internal/agent/toolset.go`）：

- `GlobalPluginPackage` 新增 `Dependencies` / `PeerDependencies` 字段；
- `LoadGlobalPlugins` 在装载前用 `nodePluginRuntime()` 判定，非空 → **跳过 goja 轨**（打日志说明）；
- 工具落地矩阵（`tool_landing_test.go`）新增 `nodeBridge` 模式并跳过该轨插件。
- 实测：测试实例启动日志出现
  `[global-plugin] tool-voice 是 Node 桥轨插件（声明了运行期 npm 依赖）——交由 Node 桥装载，跳过 goja 轨`；
  `TestToolLandingMatrix` PASS（落地矩阵 123 工具）。

**G2 —— 市场安装整包固化**（`internal/agent/toolset.go` + `npm_plugin.go`）：

- 新增 `syncGlobalPluginPackage()`：按白名单**整包复制**（index.js / client.js / assets / lib / bin / README.md）
  + 保留原始 manifest 字段（`dsh.ui` 等），覆盖 name/purpose/main/version/type/scope/config；
- `npmMarketInstall` 的 goja 分支改用它 —— 此前只把 main 源码写成 index.js，
  UI 半（assets bundle / client.js）、多文件实现（lib/）与 `dsh.ui` 声明全部丢失。
- 副作用收益：多文件 goja 插件的相对 import 现在也能在安装后正常解析。

### C.6 L1 交付：ui-voice（UI 半）

| 文件 | 说明 |
|---|---|
| `.pair/plugins/ui-voice/{package.json,index.js,client.js}` | UI 插件包：client 半用 `ui.registerPanel({id:'voice-panel'})` 注册进**通用插件面板区**（纯加法、不动壳、卸载即消失） |
| `plugins-src/ui-app/src/ui-main-voice.js` | bundle 入口（IIFE → `window.VoicePanel`） |
| `plugins-src/ui-app/src/components/VoicePanel.vue` | 面板：源素材 / 波形 canvas（10 ms 峰值）/ F0 曲线 + 音符段叠加 / 音符段表 / 切片 / 响度 / 编辑链 / 渲染产物 / 六项自检徽标 |
| `.pair/plugins/ui-voice/assets/voice-panel.{js,css}` | 构建产物（`node scripts/build-ui.mjs --region voice`，41 KB + 3.6 KB） |

数据面**不新增 host 接口**：面板经 `GET /api/fs/read` 直读 tool-voice 的工程与旁挂文件
（与 Agent 共用同一份真相源）。为此 `voice_import` / `voice_analyze` 新增**波形峰值缓存**
旁挂 `<project>.<id>.wave.json`（10 ms/帧）。

实测（独立测试实例 **9097**、非默认端口；二进制放仓库根使 InstallDir 正确）：

- `/api/plugins`：31 个插件含 `ui-voice`（hasClient=true）；
- `/api/ui-boot`：10 条 entries 含 `ui-voice` bundle URL（带内容 hash rev 锚）；
- 无头浏览器：`clientPanels` 含 `{id:'voice-panel', title:'人声', hasRender:true}`、
  instance `status=loaded`、`window.VoicePanel.mount` 存在；手动 `p.render(el)` 挂载成功（816 字符骨架含输入框）；
- 数据通道：4 个文件经 `/api/fs/read` 全部可读（4526 / 1358 / 1511 / 1461 字节）；
- ⚠️ **视觉验证受限**（如实标注）：本环境 `web_debug` 截图不落盘（工具侧问题），
  且其 element_query/eval 不在同一页面上下文 —— 因此改用 DOM 级验证（上述四项）；
  用户可在 Web UI 的「插件面板」中直接查看该面板（默认工程 `voice.project.json`）。

### C.7 决策记录与后续

1. **npm 凭据（未决）**：本期按用户决定**跳过发布**。本地可用凭据均无 `@paircode` 写权限
   （`~/.npmrc` → E404；`_temp/npmrc.bak`、`_temp/npmrc-publish` → E401 失效）。待新 token 注入后执行：
   ```bash
   PAIRCODE_NPM_USERCONFIG=<一次性 userconfig> node scripts/publish-official-plugins.mjs --publish --only tool-voice
   PAIRCODE_NPM_USERCONFIG=<一次性 userconfig> node scripts/publish-official-plugins.mjs --publish --only ui-voice
   ```
   ⚠️ **安全提示**：本轮为验证 UI 而做页面文本抽取时，输出里出现了**历史对话消息中的 token 明文**
   （用户此前在对话中提供过）。建议**轮换该 token**，并在后续会话中改用「终端自行设置环境变量」的方式。
2. **G1 / G2**：已修（见 C.5）——本期「零 Go 改动」口径**不再成立**，
   但改动集中在**插件装载/分发层**，创作能力本身仍零内核改动（§2.2 结论不变）。
3. **发布前待办**：`node scripts/sync-plugins-to-bin.mjs`（安装镜像同步）+ 两包 `--plan` 核对。
4. **后续项**：创作域其余五域（`tool-music`/`ui-music` 等）按 L2 推进；
   `ui-voice` 的试听需要 host 提供音频流端点（当前仅有 `/api/fs/read|hex|image`，无原始音频流）。

---

## 附录 D 发布包隔离与截图落盘修复（2026-09-17 实测）

### D.1 审计结论：独立插件**确实会**随发布包出厂

`packager.json` 的 `dist.include` 有一项：

```json
{ "src": ".pair/plugins", "dst": ".pair/plugins", "optional": true, "recursive": true }
```

→ `.pair/plugins` **整目录递归**进入发布包。同时 `scripts/publish-official-plugins.mjs`
又是以 `.pair/plugins` 为**npm 官方包发布源**（`listPlugins()`）——同一目录身兼
「出厂基线」与「独立发版源」两职：新插件只要放进 `.pair/plugins`，就会立刻
**既上架 npm、又固化进下一个 IDE 安装包**（体积 + 版本双份锁死），与「独立发版、
用户按需安装」的定位冲突。**结论：必须迁出。**

### D.2 迁移实施：`plugins-dist/`（独立发布插件真源）

| 项 | 变更 |
|---|---|
| 目录 | 新建 `plugins-dist/`（git 入库）；`tool-voice`、`ui-voice` 由 `.pair/plugins/` 迁入 |
| 打包 | `packager.json` → `.pair/plugins` 项增 `exclude: ["tool-voice","ui-voice"]`（`copyDistEntry` 支持，`main.go:429`） |
| 打包管线 | 新增首步 `node scripts/verify-dist-isolation.mjs`（**打包前中止**语义） |
| 发布 | `publish-official-plugins.mjs` 扫描面 = `.pair/plugins` + `plugins-dist`；同名以基线为准并告警 |
| 构建 | `build-ui.mjs` 区域发现同时扫两目录，并按 `pkg.name` **去重**（挂载态会重复发现） |
| 本地装载 | 新增 `scripts/dev-sync-dist-plugins.mjs`（默认 junction 挂载；`--copy`/`--clean`；自管 `.git/info/exclude` 防误提交） |
| 装载器 | `isPluginDirEntry()`（toolset.go）—— Windows junction 在 Go 中 `Type()==0`（既非 ModeDir 也非 ModeSymlink），须 `Stat` 跟随后判定；`LoadGlobalPlugins`/`BuildUIBootGraphFrom`/`presetAllToolPlugins` 三处接入 |
| 测试 | `tool_landing_test.go` 双源扫描 + `diskPluginDir()` 双源定位（独立插件仍受模式表守卫） |

**护栏四断言**（`verify-dist-isolation.mjs`）：① `plugins-dist/*` 均被 exclude；
② `.pair/plugins` 无同名**真实副本**（junction 挂载放行并提示）；③ exclude 无陈旧条目；
④ 目录形态合法。

### D.3 验收证据（实测）

| 项 | 结果 |
|---|---|
| 护栏（未挂载） | PASS（独立插件 2 / 基线 30 / exclude 2） |
| 护栏（junction 挂载态） | PASS + 提示「开发挂载，packager 跳过 symlink，不进包」 |
| 护栏（负例：`--copy` 真实副本） | **exit=1**，两条 ✗ 明确定位（证明护栏有效） |
| `build-ui.mjs --region voice` | 去重前「构建 ui-voice」×2 → 去重后 ×1（`plugins-dist/ui-voice/assets/voice-panel.js` 41 834 B） |
| `publish --plan` | 从 `plugins-dist` 发现两包：`未发布 2：tool-voice@0.1.0, ui-voice@0.1.0` |
| 独立实例 9097 装载 | `全局插件宿主已初始化（31 个插件）` + `[global-plugin] tool-voice 是 Node 桥轨插件…跳过 goja 轨` |
| `/api/ui-boot` | entries **10**（含 `ui-voice` bundle URL + rev）；`/api/plugins` 31 项含 `ui-voice`（hasClient=true） |
| 硬门槛 | `go vet ./...` / `go build ./...` exit 0；`go test -short ./internal/agent` **ok 81.0s** |

> ⚠️ 坑：验证二进制**不能放 `_temp/`** —— 路径含 `\temp\` 时 `InstallDir()` 走 cwd 回退；
> 若 cwd 非仓库根则 InstallDir 错 → 装载 0 插件、`/api/ui-boot` entries=0。放仓库根即正常。

### D.4 截图「不落盘」根因与修复（用户判断正确：是工具问题）

**审计对象**：`screenshot_*`（screenshot_tool.go）与 `web_debug`（webdebug.go）——两者都
`filepath.Join(root, "screenshots")`。**截图本身是会落盘的**（`screenshots/` 下确有历史
`webdebug_*.png`），但 root 有两类退化，使产物落到**用户找不到的地方**：

1. **跨工作区串根（主因）**：`InitEmbeddedToolRegistry(root)` 原为「首次 root 永久缓存」
   的单例（`if embeddedToolRegistry != nil { return ... }`），而 root 来自
   `jsplugin.go:697 ctxServiceRoot(pc)`（**每次调用的会话根**，设计上会话/工作区隔离）。
   → 第二个工作区/新会话调用时仍用第一个 root，截图/图谱产物写进**旧工作区**。
   修复：按 root 键控缓存（root 变化即重建，加锁；注册函数仅声明注册、handler 惰性，
   重建成本可忽略）。
2. **root 为空退化相对路径**：`filepath.Join("", "screenshots")` = 相对路径 →
   落进程 cwd。仓库里的 `internal/agent/screenshots/`、`cmd/screenshots/`「孤儿目录」
   即此退化痕迹（已归档到 `_temp/orphan-screenshots/`）。
   修复：新增 `screenshot_path.go` 的 `artifactDir/screenshotOutputDir` ——
   **调用方 root → `core.Root()`（工作区主根，实时）→ `core.InstallDir()`** 逐级兜底，
   并统一返回**绝对路径**；`webdebug.go` 的 `MkdirAll` 错误不再静默（记入报告 consoleMsgs）。

**回归测试**（均 PASS）：
`TestEmbeddedToolRegistryFollowsRoot`（同 root 幂等 / root 变化与回切必重建）、
`TestScreenshotLandsInCallingRoot`（端到端：先以 rootA 初始化内核，再以 rootB 调
`screenshot_area` → **产物落在 rootB/screenshots**、rootA 无产物）、
`TestScreenshotOutputDirFallback`（三级兜底 + 绝不退化为相对路径）。

> 同类排查：`getCachedCodeGraph(root)`、`LoadEmbeddingCache(root)` 均为**按 root 索引**，
> 无串根；宿主框架工具（glob/grep/project_info 等）以「主项目根」为口径属**设计如此**，
> 不在本次修复范围。内嵌内核一处的修复同时覆盖 screenshot / web_debug / codegraph /
> office / harness 全部回退工具。

### D.5 遗留

1. **发布仍未执行**：`tool-voice` / `ui-voice` 待 npm 凭据就绪后
   `--publish --only …`（源目录已含 `plugins-dist`，CI 无需改动）；
2. **本地开发挂载**：默认 junction（`dev-sync-dist-plugins.mjs`）——Windows 下由
   `isPluginDirEntry` 支持；若在非 Windows 环境用 symlink 亦可（同判定分支）；
3. 创作域其余五域（`tool-music`/`ui-music`、`tool-art`/`ui-art`、`tool-design`/`ui-design`、
   `tool-model`（工具 + UI 同包）、`tool-rig`/`ui-rig`）按 L2 推进 —— 其一经实现即应落在
   `plugins-dist/`（护栏会强制其进 exclude，漏了就打不了包）。

---

## 附录 E L2 执行记录（音乐域：tool-music + ui-music，2026-09-17 实测）

### E.1 交付物

| 包 | 形态 | 内容 |
| --- | --- | --- |
| `tool-music` | **磁盘 goja 轨**（零 npm 依赖，单文件 1540 行） | 5 工具：`music_project` / `music_edit` / `music_import` / `music_export` / `music_verify` |
| `ui-music` | 磁盘插件（host 半 + client 半 + IIFE bundle 42 KB） | 面板：工程信息 / 五线谱 / 钢琴卷帘 / 原生 Web Audio 试听 / 轨道表 / 7 项校验 / 产物清单 |

真源位于 `plugins-dist/`（不进 IDE 发布包），`packager.json` 的 exclude 已同步；本地由
`dev-sync-dist-plugins.mjs` junction 挂载。

### E.2 关键设计：文本真相源 + 三向一致契约

- **真相源 = 工程 JSON**（`music.project.json`）：tracks/notes/tempo/meter/key，音符时间一律
  **整数 tick**（ppq 480）→ 可 diff、无浮点漂移；
- MIDI（SMF format 1）/ ABC / MusicXML / SVG 乐谱**都只是产物**，随时可由工程重建；
- 二进制安全：SMF 读写走宿主 `ctx.fs.readFileBase64` / `writeFileBase64`（base64 编解码自带，
  不依赖 btoa / Buffer）；这使音乐域**无需 Node 桥**（对比 tool-voice 的 `@audio/*` 依赖）。

### E.3 验收数据（实测）

| 断言 | 结果 |
| --- | --- |
| 7 项判据 V1..V7 | ✅ 全绿（示例工程 44 音符 / 2 轨） |
| V4 SMF 回读一致 | 515 字节 → 回读 44/44 音符，速度/拍号/ppq/轨道通道与音色均一致 |
| V5 确定性 | 两次导出 515 字节**逐字节一致** |
| V6 ABC 往返（含和弦 `[CEG]`） | 233 字符 → 44/44 音符集合一致 |
| MIDI 回读 → 新工程 → 再校验 | ✅ 7/7（音色 32 保留、中文轨道名保留） |
| 实例装配（9097 独立二进制） | 插件 33；`/api/tools` 128（新增 5 个 `music_*`）；`/api/ui-boot` 11 entries |
| 面板端到端（CDP） | sections 5 / checks 7 / 轨道行 2 / canvas 431×182（dataURL 13 KB）；`registeredPanels=["music-panel","voice-panel"]` |
| 视觉验证 | 乐谱 SVG 与面板截图经 `read_image` 确认：标题/双五线谱/谱号/小节线/符头高度/时值空心实心/轨道表/7 项校验均正确 |

### E.4 本次暴露并修复的缺陷（均实测定位，勿回退）

1. **面板读文件 404（既有 bug，ui-voice 同中招）**：`api.apiGet('/api/fs/read', …)` —— `apiURL`
   已自动补 `/api` 前缀 ⇒ 实际请求 `/api/api/fs/read`。改为 `/fs/read`。证据：CDP 拦截 fetch
   打印真实 URL。**ui-voice 面板此前一直读不到工程文件**（上轮只验了 host 侧接口，未验面板）。
2. **ABC 声部标记被当头部吞掉**：`V:` 命中头部正则 ⇒ 不进 body ⇒ 多轨第二声部整体偏移
   （实测 t2 音符 +3840 tick）。修正：头部集合去掉 V，body 遇 `V:` 重置时间轴。
3. **ABC 时值无法表示分数**：`dur/unit` 非整数时生成非法记号 ⇒ V6 必然失败。修正：新增
   `abcLenSuffix`（整数 / `/2` / `3/2` 形式）+ 和弦 `[...]` 的生成与解析。
4. **SMF 解析不读 program change** ⇒ MIDI 往返丢音色（实测 32 → 0）。修正并把轨道通道/音色
   纳入 V4 断言。
5. **SMF track name 非 ASCII 被写成 `?`** ⇒ 中文轨道名丢失（「主旋律」→「???」）。修正：
   自实现 UTF-8 编解码（编码 + 解码，非法序列回退 Latin-1）。
6. **V7 标签计数误匹配** `<part-list>` / `<part-name>`（用 `<part\b`）⇒ 假失败。改为
   `<part\s+id="`。
7. **工具落地矩阵遗漏**：`TestToolLandingMatrix` 报「tool-music——需在 toolPluginModes 声明」。
   修正：声明为 `native`（全 JS 原生，无 hostTool / binary 回退分支）。

### E.5 验证方法学（重要）

本会话宿主是**安装目录旧构建**（`E:\Program Files (x86)\PairCode\pair.exe`），其 `web_debug`：
① `eval` **不等待 Promise**（async 脚本只能拿到 `{}`）；② 截图**不落盘**。故本轮改用
**CDP 直连**（Node 22 原生 `WebSocket` + `--remote-debugging-port`）：
`Runtime.evaluate(awaitPromise:true)` 挂载面板并等待异步数据 → `Page.captureScreenshot` 落盘 →
`read_image` 视觉确认。脚本：`_temp/ui-music-panel-verify.mjs`。

### E.6 遗留

1. **宿主需覆盖安装**（`node scripts/overwrite-install.mjs --apply`）：否则附录 D 的内核修复
   与本节的组件修复对用户实例**均不生效**（仍会看到白屏误报 / 截图落偏 / 面板读不到文件）。
   注意：该命令会结束当前运行实例，**须由用户执行**（Agent 自执行会中断会话）。
2. 发布（`tool-voice` / `ui-voice` / `tool-music` / `ui-music`）待 npm 凭据就绪。
3. 其余四域（art / design / model / rig）按 L2 继续。
4. 工作区根残留的历史畸形目录 `syprojectgou-ide/`（2026-09-11 路径拼接产物，非本轮产生）。

---

## 附录 F L2 执行记录（矢量/图像域：tool-art + ui-art，2026-09-17 实测）

### F.1 交付物

| 包 | 形态 | 内容 |
| --- | --- | --- |
| `tool-art` | **磁盘 goja 轨**（零 npm 依赖，单文件 2121 行） | 5 工具：`art_project` / `art_edit` / `art_import` / `art_export` / `art_verify` |
| `ui-art` | 磁盘插件（host 半 + client 半 + IIFE bundle 47 KB） | 面板：画布预览 / 图元表（选中高亮 + op 片段）/ 图层 / 颜色对比度 / 7 项校验 / **浏览器侧导出 PNG** |

真源 `plugins-dist/{tool-art,ui-art}`，`packager.json` exclude 与 `.git/info/exclude` 已同步；
本地由 `dev-sync-dist-plugins.mjs` junction 挂载（现共 6 个独立插件）。

### F.2 关键设计

- **真相源 = 画板工程 JSON**（`art.project.json`：canvas / palette / layers / shapes）；
  **SVG 是唯一文本产物**（零依赖生成，可回读 → V4 契约）。图元几何扁平存字段（diff 友好），
  可选 `transform` 用 **2D 仿射矩阵 [a,b,c,d,e,f]**：导出为 `matrix(...)`（与 SVG 语义完全等价），
  导入时把 `translate/scale/rotate/matrix/skew` 任意列表归一为矩阵 —— 无损往返。
- 图元子集 8 类：rect / circle / ellipse / line / polyline / polygon / path / text；
  包围盒对 path 取**保守 AABB（含贝塞尔控制点）**、对圆/椭圆做 16 点圆周采样（旋转后仍准确）。
- **7 项判据**：V1 画板合法 / V2 图元结构 / V3 视口可见性 / V4 **SVG 往返一致** /
  V5 确定性（位级）/ V6 **颜色合法 + 文本对比度达标（WCAG AA）** / V7 图层层级 + 产物无脚本面。
  V6 的底色是「文本中心点下方最上层的实心图元」，无则取画板底色（缺省白）——
  能真正抓到「白字压在浅色块上」这类看不见的缺陷（实测抓到 1 处）。
- 安全：导入时 `<script>` / `<style>` 丢弃、`javascript:` 与 `url()` 颜色值剥离；
  导出后 V7 再断言产物无 `<script>` / `on*` / `javascript:` URL。
  面板预览用 `<img src="data:image/svg+xml;...">`（SVG 内脚本不执行），不做信任假设。
- **边界如实标注**：PNG 光栅化不在 goja 沙箱能力内 —— `art_export format=png` **不假装完成**，
  而是返回两条可行路径（面板浏览器 canvas / 后续 Node 桥 resvg 导出链）。面板侧已实现前者。

### F.3 验收数据（实测）

| 断言 | 结果 |
| --- | --- |
| host 半全链路（`node _temp/tool-art-test.cjs`） | **17 步全绿**（含 3 组负向 + 9 项边界拒绝） |
| 示例工程（26 图元 / 3 图层 / 720×420） | 7 项判据全绿；SVG 3229 字符；V4 往返逐字段一致；V5 位级一致 |
| 导入外部 SVG（含 g/style/transform） | 画布取自 width/height、`<title>` 取标题、translate 累积到子图元矩阵、rotate 归一为矩阵、defs/image/script 正确忽略 |
| 实例装配（9097 独立二进制） | 插件 35；`ui-art hasClient=true`；`/api/ui-boot` entries 12；`/api/tools` 含 5 个 `art_*` |
| 面板端到端（CDP） | sections 6 / 图元行 26 / 校验 7（全通过）/ 色块 34 / `<img>` naturalW×H = 720×420 / 选中高亮框缩放正确（587/720）/ 导出 PNG 成功（720×420）/ canvas 非白像素 72243 |
| 注册面板 | `["art-panel","music-panel","voice-panel"]`（插件面板三件套共存） |
| 面板 fetch 路径 | `/api/fs/read?path=…`（**无** `/api/api` 双前缀——上轮修复的回归点） |
| 视觉验证 | SVG 独立截图 + 面板两张截图（含 Emulation 放大视口的全览），经 `read_image` 逐项确认 |

### F.4 本次暴露并修复的缺陷（均实测定位，勿回退）

1. **`d` 字段被当数字解析**：`shape.add type=path` 的 `d` 走了数值分支 → `sh.d = 0` → 报「path 必须提供 d」。
   修正：字符串类字段集合加入 `d`。
2. **V3 误判视口内直线为「画板外」**：水平/垂直线包围盒高/宽为 0，`bboxIntersectArea` 恒为 0 ⇒ 误判。
   修正：改用**区间重叠**判定 `bboxOverlaps`（含退化情形），并保留「部分越界」提示项。
3. **V6 抓到真实 WCAG 问题**：示例徽章用白字 + Emerald 600(#059669) 仅 **3.77:1**（15px bold 未达大字门槛）。
   修正：改用 Emerald 700(#047857) → **5.48:1** 通过。这正是该判据的价值（挡住"看不见的文字"）。
4. **空画板语义**：V3 在 0 图元时判 FAIL，指标文案改为「画板为空（尚无任何图元）」，避免误读为几何错误。
5. **落地与打包声明**：`toolPluginModes` 加 `tool-art: native`（否则 `TestToolLandingMatrix` 报未落地）；
   `packager.json` exclude 补 `tool-art` / `ui-art`（否则打进 IDE 发布包）。

### F.5 验收方法学补充（面板全览截图）

CDP 的 `Page.captureScreenshot({captureBeyondViewport:true})` 在 `--headless=new` 下**不生效**
（截图仍限视口，长面板只能看到上半屏）。可行做法：先解除面板固定高度（`height:auto; overflow:visible`），
再 `Emulation.setDeviceMetricsOverride({width:700, height:2300})` 放大视口后截图 —— 一次拿到全内容，
可直接核对下半区（颜色/校验/选中详情）。脚本：`_temp/ui-art-panel-verify.mjs`。

### F.6 遗留

1. **PNG 光栅化与位图转矢量**：属「服务端导出链」阶段（Node 桥 `@resvg/resvg-js` / satori，MPL-2.0 ——
   随分发须保留声明并同步 `docs/THIRD_PARTY_NOTICES.md`）。面板侧 PNG 导出已可用。
2. **画布编辑（选择/变形/对齐回写）**：属 E3/S3 阶段（`artifact_apply`）。当前面板只读 +
   提供可复制的 op 片段，保持「单一写路径（Agent 工具）」纪律。
3. 其余三域（design / model / rig）按 L2 继续。
4. 宿主需覆盖安装（同附录 E.6：否则本轮与 D/E 节修复对用户实例均不生效）。

---

## 附录 G L2 执行记录（UI 设计域：tool-design + ui-design，2026-09-17 实测）

### G.1 交付物

| 包 | 形态 | 内容 |
| --- | --- | --- |
| `tool-design` | **磁盘 goja 轨**（零 npm 依赖，单文件 2100+ 行） | 5 工具：`design_tokens` / `design_project` / `design_edit` / `design_export` / `design_verify` |
| `ui-design` | 磁盘插件（host 半 + client 半 + IIFE bundle 39 KB） | 面板：iframe 预览 / 颜色令牌 + 对比度 / 屏幕结构 / 9 项校验 / 复制命令 |

真源 `plugins-dist/{tool-design,ui-design}`；`packager.json` exclude 与 `dev-sync-dist-plugins.mjs` 已同步
（现共 8 个独立插件）。

### G.2 关键设计

- **双文本真相源**：① `design.tokens.json`（颜色/间距/圆角/字号/字重/行高/阴影/字体族，
  语义命名，默认值取自 Tailwind 公开标准值 + 4px 网格）；② `design.project.json`
  （屏幕 → 节点树，13 类节点：容器 frame/row/col/card/list，叶子 text/button/input/badge/icon/image/divider/spacer/checkbox）。
  节点样式**一律引用令牌**（`$color.bg` / `$space.md`），禁止散落裸值。
- **确定性布局引擎**（goja 内自实现 flex 堆叠模型）：容器 w 默认撑满、h 默认内容高；
  row/col 主轴排布 + gap + pad；叶子按内容估算（文本用确定性字宽模型估宽换行）。
  ★ **导出 HTML 用同一份布局盒子做绝对定位** → 「校验通过」与「截图所见」严格同源。
- **产物全为文本**：自包含 HTML 预览（无脚本、无外链、可截图）/ CSS 自定义属性（`--color-accent` 等，
  可直接交付前端）/ mermaid 图（本插件只**生成 mermaid 文本**，渲染交给宿主 —— 零依赖）。
- **9 项判据**：D1 结构与必填（id 唯一、类型合法、容器叶子约束、深度/规模上限）｜
  D2 令牌引用有效（无悬空 + **语义组匹配**：抓"颜色令牌当宽度用"+ 尺寸可解析）｜
  D3 文本对比度（WCAG AA：正文 4.5 / 大字 3.0；按钮与徽章按 variant/tone 实际配色）｜
  D4 间距落在 4px 网格｜D5 字号必须来自 `$fontSize.*`（禁裸字号）｜
  D6 触达区（屏宽 ≤480px 要 44px，否则 32px）｜D7 图标白名单 + **无 emoji/符号图标**（技能 emoji-icons）｜
  D8 布局无溢出/越界（与预览渲染同源）｜D9 产物确定且自包含（两次渲染位级一致、无 script/on*/javascript:/外链）。
- **事务语义**：`design_edit` 一次 ops 批量应用，**任一 op 抛错则整个批次不落盘**（全有或全无），
  避免"改了一半"的工程留在磁盘上。
- **能力边界如实标注**：布局引擎只实现堆叠 + 交叉轴拉伸（**未实现 justify/grow**，"底部固定栏"需显式留白）；
  位图导出同 art 域属后续「服务端导出链」；面板只读工作区（编辑一律经 Agent 的 `design_edit`）。

### G.3 验收数据（实测）

| 断言 | 结果 |
| --- | --- |
| host 半全链路（`node _temp/tool-design-test.cjs`） | **全绿（0 失败）**：正向 10 步 + D2–D8 负向 10 例 + 边界拒绝 20 例 |
| 负向覆盖 | 悬空令牌 / 语义错用 / 白字压白底（1.05:1 被抓）/ pad=10 越网格 / 裸字号 / 按钮高 20px / 未知图标 + emoji / row 溢出 408>328 / 节点越界 —— **每例都精确落到对应判据** |
| 示例工程（`node _temp/tool-design-demo.cjs`） | 3 屏 / 52 节点 / 9 判据全绿；HTML 9767 字符；CSS 42 个变量；mermaid 3 subgraph + 2 导航边；对比度最低 4.63:1 |
| 实例装配（9097，新构建 `companion_test.exe`） | 插件 37；`ui-design hasClient=true`；`/api/ui-boot` entries 13；工具 128，含 5 个 `design_*` |
| 面板端到端（CDP） | sections 6 / 色块 13 / 校验 9（**9 全过**）/ 键值行 19；iframe `sandbox=""` + srcdoc 9788 字符（预览真渲染）；`registeredPanels` = `["art-panel","design-panel","music-panel","voice-panel"]` |
| 面板 fetch 路径 | `/api/fs/read?path=…` 读 4 个文件（工程/令牌/报告/HTML），**无 `/api/api` 双前缀** |
| 视觉验证 | HTML 全览 + 近景、面板全览截图，经 `read_image` 逐项确认（含修复后的文字完整性与图标清晰度） |
| 护栏 | `verify-dist-isolation` **PASS**；`TestToolLandingMatrix` **ok**（`tool-design=native`）｜`go vet`/`go build` exit 0 |

### G.4 本次暴露并修复的缺陷（均实测定位，勿回退）

1. **模板节点 id 跨屏冲突**：两屏同用 `n1…n18` → D1 报 id 重复、目标选择出现歧义。
   修复：套模板后统一加屏幕前缀（`prefixIds` → `chat_n6`）。
2. **同批次 id 分配撞号**：`allocNodeId` 看不到"正在构建/复制、尚未挂进树"的节点 →
   `node.duplicate` 后子树 id 重复（D1 抓到）。修复：引入 `reserved` 集合贯穿 build/renumber。
3. **`token.set` 未走校验**：间距写成 10、颜色写成 `not-a-color` 都能落盘。
   修复：`applyOps` 的 token 分支调用 `validateTokenValue`。
4. **emoji 检测漏检**：`charCodeAt` 只能取代理对高位（0xD83D 不在 emoji 区间）→ 🚀 假通过。
   修复：手工合成码点后判定。
5. **文本宽度估算偏窄 → 末字被裁**（截图上"Agent 回复"变"Agent 回"、长句丢"端。"）：
   字宽系数上调（大写 .70 / 小写 .57）+ 文本盒子留 4%+2px 安全余量。
6. **图标在 20px 下渲染成模糊星芒**（settings 齿轮弧线版）→ 换成简单几何（滑杆样式）；
   面板预览改用更清晰的图标语义。
7. **面板屏幕行空白**：模板读 `s.w/s.h/s.nodes`，而工程里是 `width/height/root` →
   面板把原始对象映射为视图模型。
8. **默认色板自身不达 WCAG**：`accent-soft #DBEAFE` 上的 accent 仅 4.31:1、`warning #B45309` 4.15:1、
   中性徽章用 muted 3.72:1 → 调整为 `#EFF6FF` / `#92400E`，中性徽章前景改用正文色。
   （这正是 D3 判据的价值：**连插件自带模板都不放过**。）

### G.5 验证方法学补充（三条实测坑）

1. **测试二进制必须放仓库根（或 `bin/`）**：`_temp/` 下运行时 `InstallDir()` 落到 `_temp`
   （判定串是 `\temp\`，而 `_temp` 不匹配）→ 扫不到 `.pair/plugins` → **0 插件装载**。
2. **必须用含最新源码的二进制**：早期二进制不跟随 Windows junction（`DirEntry.Type()==0`）→
   8 个独立插件全部未装载（其余 30 个真实目录正常）。`CGO_ENABLED=1 go build -o companion_test.exe ./cmd/companion`
   后重跑即 37 个全装载。
3. **面板验证前先打开工作区**：`POST /api/workspace {"action":"open","root":"<绝对路径>"}`。
   ★ root 用脚本文件传（`_temp/open-ws.cjs`），不要用 shell 内联 JSON —— 多层转义会把反斜杠吞掉，
   得到 `F:syprojectgou-ide` 这种路径，面板随后报"文件名、目录名或卷标语法不正确"。
4. 截图仍走自建 CDP（`_temp/ui-design-panel-verify.mjs`）：宿主 `web_debug` 的截图在安装版旧构建下不落盘。

### G.6 遗留

1. **布局引擎缺 justify/grow**：底部固定栏只能靠显式留白（示例中已用，但脆弱）。
   建议后续给容器加 `justify: between|end` 与 spacer 的 `grow` 语义（渲染与 D8 同步）。
2. **位图/切图导出**：同 art 域，属「服务端导出链」（Node 桥 + 许可声明）。
3. **设计稿 → 前端代码**：当前交付到 CSS 变量 + HTML 预览；生成组件代码（Vue/React）属后续域。
4. 其余两域（model / rig）按 L2 继续；发布侧遗留同附录 E.6 / F.6。

---

## 附录 H L2 执行记录（2D 角色域：tool-rig + ui-rig，2026-09-17 实测）

### H.1 交付物

| 包 | 形态 | 内容 |
|---|---|---|
| `tool-rig` | goja 轨、**零 npm 依赖**（单文件 ~2960 行） | 5 工具：`rig_model` / `rig_import` / `rig_edit` / `rig_export` / `rig_verify` |
| `ui-rig` | host 半 + client 半 + bundle 45 KB | 面板：可交互预览（`sandbox="allow-scripts"`）/ 骨骼树 / 部件 / 参数与驱动来源 / 二级运动 / 9 项校验 |

### H.2 关键设计

- **真相源 = 上游 `.iki` v1 文档**（不是自家方言）：字段与 `@ikijs/format` 的 TypeScript 契约逐项一致
  （`version/name/canvas/parameters/textures/parts/deformers/physics/physicsChains`），坐标口径照上游
  （模型空间原点 = 画布中心、+y 向上、rotation 逆时针为正、`*_L` = **角色自身**左侧）。
  → 产出的 `rig.model.iki.json` 可直接被 `@ikijs/engine`（WebGL2 运行时）加载；验收用**上游真实校验器**
  `parseIkiModel` 交叉校验（不是"格式自洽"就算过）。
- **骨架 = matrix deformer 树**，id 一律 `bone_` 前缀 —— .iki 里 part 与 deformer **共用同一命名空间**
  （上游会报 `deformers[2].id "neck" collides with parts[3].id`，实测踩到）。
- **参数绑定按上游语义**：`t = (clamp(v)-min)/(max-min)`、`value = from + (to-from)·t`；
  translate/rotate/scale **累加到基变换**、opacity **相乘**；deformer 局部矩阵 = `T(pivot)·T(x,y)·R·S·T(-pivot)`；
  warp 子部件**不继承 matrix 父链**（= partAffine + grid 形变）。
- **PSD 分层导入（自研最小结构解析）**：只读文件头 / 图层矩形 / 可见性 / 不透明度 / 混合模式 /
  图层名（Pascal + `luni` Unicode）/ 图层组（`lsct`）；**通道像素一律跳过**（不解 RLE、不光栅化）；
  沙箱无 Buffer → `ctx.fs.readFileBase64` + 自研 base64 解码；上限 32MB（超限走「图层清单」路径）。
  组树算法与 `ag-psd` 一致，并以其写出的真实 PSD 交叉验证。
- **命名约定自动绑定**（对齐上游 auto-rig 的 role layer 约定）：词根表 → 部位 role + 侧别；
  自动生成 deformer 树（root/body/neck/head/arm_LR/hand_LR/cloth/hair）、deformer 级绑定
  （头部倾斜/转向/俯仰 + 呼吸）、部件级绑定（眼睑**与虹膜** scaleY、眼球 translate、嘴 scaleY、眉 translate/rotate）、
  头发多段角链（→ `ParamHairSwayX/Z`）、衣物弹簧（→ 自定义参数）。
  ★ 部件级 scale 绑定会把**基变换 scale 归零**（否则上游的"累加"语义会让"闭眼"变成放大）。
- **产物全为文本**：自包含可交互预览 HTML（内置 canvas 渲染器：deformer 树复合 + 线性绑定 +
  弹簧/角链二级运动（1/60s 半隐式欧拉，公式逐行对齐上游）+ 参数滑块 + 演示动画；暴露
  `window.__RIG_DEBUG` 供自动化核验几何）/ 部件布局 SVG / 图层命名清单 JSON（交代美术）/ 规范化 .iki 副本。
- **9 项判据**：R1 结构契约（含 **part/deformer id 空间冲突**、order 唯一且连续）｜R2 命名规范
  （词根 + `_L/_R` 侧别、禁 emoji、成对部件缺侧别）｜R3 deformer 树（父存在/无环/深度/warp 父须 matrix/
  **rest 网格晶格规则性**——上游校验要求）｜R4 蒙皮完整性（含"warp 子部件必须有 mesh"）｜
  R5 参数与绑定（含 deformer 禁 opacity、标准参数范围被改）｜R6 物理语义（含 input≠output、
  多弹簧/多链段共用输出参数）｜R7 部件重叠（★ 语义精确化：**完全重合 = FAIL，前后层次重叠 = WARN**）｜
  R8 变换与网格（mesh 顶点/UV/索引一致性；**scale=0 且有绑定时不算退化**）｜R9 产物确定且自包含。

### H.3 验收数据（实测）

- **本地全量测试 116/116 通过**（`_temp/tool-rig-test.cjs`）：含 PSD 解析与 `ag-psd` 真值逐层比对
  （20 叶子 / 2 组 / 隐藏位）、上游 `parseIkiModel` 交叉校验、批量 ops 事务性（失败批次字节不变）、
  产物位级确定性、`bind.auto` 幂等、R1–R8 负向各一例、`rig_edit` 30+ 边界拒绝。
- **示例产物**（工作区根）：19 部件 / 11 deformer / 19 参数 / 2 弹簧 + 1 角链；9 项判据全过（2 警告）。
- **装配（9097 + 新构建二进制）**：插件 **39**、工具 124（含 5 个 `rig_*`）、`/api/ui-boot` **entries 14**。
- **面板端到端（CDP）**：sections 8 / 参数行 19 / 骨骼行 10 / 校验 **9 全过** / `badge 9/9` /
  iframe `sandbox="allow-scripts"` + srcdoc 23723 字符 / `registeredPanels` = **5 个面板共存** /
  fetch 全走 `/api/fs/read`（**无 `/api/api` 双前缀**）/ 预览 HTML 自包含（外部 URL 0）。
- **渲染一致性交叉核验**：浏览器内渲染器 vs 插件侧几何计算，同模型同参数下四角坐标最大偏差 **≤5e-5**
  （纯浮点舍入；无旋转/缩放的部件为 0）→「校验说没重叠」与「图上所见」同源。
- **视觉验证**（`read_image` 真看 5 张图）：默认姿态 / 参数驱动后（头部旋转+位移、眼睛压扁成细条、
  嘴由细线变矩形、眉毛一高一低、眼球偏移）/ 头部近景 / SVG 产物 / 面板全览。
- **硬门槛**：`go vet` / `go build` exit 0；`verify-dist-isolation` **PASS**（10 项 exclude；
  10 个 junction 插件不进发布包）；`go test -short ./internal/agent` **ok 68.0s**。

### H.4 本次暴露并修复的缺陷（均实测定位，勿回退）

1. **PSD 附加信息块双重消耗**：读 `luni` 内容已前进 `blen`，又 `seek(r.p + blen)` 再跳一次 →
   后续块（`lsct`）全被跳过 → **组信息完全丢失**（`groups=0`、`leaves` 把组记录算进去）。
   修复：一律用块起点推进（`dataStart + blen + blen%2`）。
2. **`luni` 长度前缀被当字符**：Unicode 图层名 = 4 字节字符数 + UTF-16BE → 名字变成 `"\tface_base"`。
   修复：先读字符数再取 `uniChars*2`。
3. **组树算法口径错**（最初按"分隔符/组记录相邻配对"）：真实 PSD 里两者**并不相邻**
   （head 组记录在 index 23、其分隔符在 9）。修复：改用与 `ag-psd` 一致的**栈算法**
   （自列表末尾向 0 遍历：`lsct` 1/2 入栈、3 出栈），并用嵌套组 PSD 交叉验证通过。
4. **deformer id 与部件 id 撞名**：上游报 `collides with parts[j].id`。修复：骨骼一律 `bone_` 前缀 +
   R1 增加冲突判据。
5. **warp keyform `value` 未递增**：`from == to == 0` 产出上游必拒的文件。修复：缺省取参数范围 min/max；
   显式 `from >= to` 直接报错。
6. **warp 子部件被当成"未知 deformer"**：上游语义里 warp deformer **没有矩阵**。修复：
   `partWorldMatrix` 查不到就返回局部矩阵；`partQuad` 再对四角做 grid 双线性采样（与预览渲染同公式）。
7. **部件级 scale 绑定未归零基变换**：上游是 `result.scaleY += value` → 眨眼变成"越眨越大"。
   修复：`partBind` 对 scale 通道把基变换归零；R8 排除"scale=0 且有绑定"的误报。
8. **`bind.auto` 不幂等**：每次 push 弹簧/角链 → 模型每次都变（破坏可 diff）。修复：按 id 替换。
9. **`bind.auto` 覆盖手工 warp 挂载**：warp 子部件被重新 attach 回 `bone_head`。修复：保留原挂载。
10. **`argStr` 处理对象型参数**：`physics.add` 的 `input/output` 传对象被 `String()` 成 `"[object Object]"` →
    误报"input 与 output 是同一参数"。修复：对象分支取 `.parameter`。
11. **R7 误报正常美术结构**（前发盖后发必然重叠）。修复：语义精确化（完全重合 FAIL / 一般重叠 WARN）。
12. **R8 越界判定用错坐标语义**（模型空间是 ±W/2，原判据用 0..W → 误报底部部件越界）。
13. **虹膜不随眨眼闭合**（截图发现：眼白压扁了、虹膜还立着）。修复：auto-rig 给虹膜补绑同参数 scaleY。
14. **SVG 小部件标签互相压字**（`eye_white_L` 与 `iris_L` 叠成一团）。修复：只在标签装得下时画。

### H.5 验证方法学补充（本域新增）

1. **上游契约当"真值"**：`npm view` 取 tarball 解包直接读 `.d.ts`/README（`@ikijs/format` 是 .iki 的单一真相源），
   再用其 `parseIkiModel` 跑交叉校验 —— 比"自己写 schema 自证"强得多。
2. **外部实现当"真值"做字节级交叉验证**：用 `ag-psd`（成熟实现）**写出真实 PSD** 再让自研解析器读，
   逐层比对名字/矩形/隐藏位/组路径；反过来也 dump 我们读到的记录去对照它的栈算法。
3. **两套独立渲染实现的几何互证**：预览 HTML 的渲染器（浏览器）与插件侧几何函数（goja/Node）对同一模型
   同一参数输出四角坐标，偏差 ≤5e-5 才算"同源"；`window.__RIG_DEBUG` 就是为此留的钩子。
4. **参数驱动截图要看"是否真的变形"**：走真实交互路径（改 `input.value` + 派发 `input` 事件）而不是直接改
   内部状态 —— 否则滑块读数与画面不同步，看不出绑定是否生效；几何对比要挑**物理无关**的部件
   （弹簧/角链在跑，抖动部件的坐标不稳定）。
5. **启动测试实例必须显式指定端口**：`WEB_PORT=9097 ./companion_test.exe`（端口由 `InitCore()` 读环境变量，
   缺省 9090）。★ 忘了它就是撞用户实例：本次默认端口启动直接 `bind: Only one usage…`（默认端口 9090 绝不动）。

### H.6 遗留

1. **PSD 像素/纹理接入**：只读结构、不读像素（同 art 域的边界）→ `part.texture.uv` 已支持写入，
   把 PSD 图层真实像素拆成图集属"服务端导出链"（Node 桥 + 许可）。
2. **网格形变精修**：支持 1D grid warp（`preset: bend/sway/bulge`）+ 网格生成；2D warp（`warp2d`）
   与多网格组合在上游 v1 本身也是受限能力（只允许 1 个 grid warp）。
3. **动作/表情时间线不在 .iki v1 契约内**（上游只有 idle + physics/chain 驱动）→ 未做 motions；
   「会话形象」（S4）需宿主按参数驱动，属后续。
4. **超大 PSD**（>32MB）走「图层清单」路径 —— 清单格式已在 `rig.parts.json` 输出
   （含模型空间矩形与命名约定），美术/外部工具可照此回填。
5. 剩余一域（model / 3D-CAD，Node 桥 + WASM 几何）按 L2 继续；发布侧遗留同附录 E.6 / F.6。

---

## 附录 I L2 执行记录（3D/CAD 域：tool-model + ui-model，2026-09-17 实测）

> ★ **I.0 后续修订（同日，用户反馈驱动）**：原设计把工具（`tool-model`）与 UI（`ui-model`）
> 拆成两个独立发布包，实测暴露三个问题——① 用户看不出「AI 用的工具」与「面板」是同一件事，
> 包/版本/文档各一份；② 面板没有入口指引（不知道有什么用、怎么从工程打开预览）；③ 预览是
> **写死的固定三件套**（model.json / model.preview.html / model.verify.json），体现不出
> 「工具操作到具体文件」的链路。修订为：
> **UI 与工具同包**（`plugins-dist/tool-model/` = host 半 index.js + client 半 client.js +
> `assets/model-panel.js`，`dsh.ui` 段与工具面共存），并新增「**识别到的文件**」数据流——
> 工具写文件即 `noteArtifact` 登记 + `ctx.emit('ui:tool-model/artifacts')` 广播 → 面板列出
> 识别到的文件 → **点击即登记并当场预览** → 文件一变预览**自动重绘**（事件广播 + 2s
> `statArtifact` 复核）。面板内补「这个面板是什么 · 怎么用」与三种打开入口指引。
> 端到端验证：`_temp/model-panel-verify.cjs`（CDP，31 项）+ `_temp/model-ui-live-test.cjs`
> （host 半，19 项）。下表的 `ui-model` 行即修订前的历史形态，保留以存证。

### I.1 交付物

| 包 | 形态 | 内容 |
|---|---|---|
| `tool-model` | goja 轨、**零 npm 依赖**（单文件 ~3000 行） | 5 工具：`model_doc` / `model_add` / `model_edit` / `model_export` / `model_verify` |
| `ui-model` | host 半 + client 半 + bundle 42 KB | 面板：3D 预览（iframe srcdoc 挂自包含 WebGL 页，可拖拽旋转/缩放/线框）/ 部件表 / 参数与引用 / 判据 M1–M9 / 产物清单 / 命令 |

### I.2 关键设计（含对 §3.2 的**形态修订**，已实测）

- **★ 形态修订：本域未采用 §3.2 写的"Node 桥（WASM 几何）+ three + @jscad"，改为 goja 零依赖 + 自研内核。**
  理由（逐条实测）：① `@jscad/modeling` 2.13 是**纯 JS、零依赖、无 WASM**（`npm view` 已核实），但 goja 沙箱
  不支持 `require`/多文件模块，直接复用不可能；② Node 桥要落 `node_modules`（jscad ~1MB + three ~1.2MB）+
  真实 `npm install`，引入装配体积与离线风险，而本域几何是**纯数值计算**，没有必须用 npm 生态的原语；
  ③ 改为"自研内核 + 把 jscad 当**外部真值实现**做交叉验证"后，验证强度**不降反升**（见 I.3 的 rel ≤ 1e-14）。
  three 也不需要：预览渲染的是**工程自己的 mesh**（不是 glTF 文件），自研 WebGL 渲染器 ~200 行即可，
  且无需在浏览器里跑 glTF 解析器。
- **真相源 = 公开格式，不是自家方言**：工程是文本 JSON（`model.json`，可 diff/review），
  产物是 **glTF 2.0**（Khronos 标准；自包含 base64 buffer；导出时按规范把工程的 z-up 转成 **y-up**）+
  **binary/ASCII STL**（3D 打印）+ OBJ。验收用**官方 gltf-validator** 做门槛（`errors=0` 才算过）。
- **坐标口径照 CAD 通行惯例**（OpenSCAD/JSCAD/STL）：右手系、**+z 向上**、单位 mm；primitive 的
  `center` 默认在原点、`size` 为全尺寸 → 与 JSCAD primitives 同口径，故能逐项交叉验证。
- **参数化 = 表达式**：几何字段可写表达式（`"wall*2"`、`"thickness*4"`），自研递归下降求值器
  （不用 `eval`/`Function`），参数之间可互相引用（迭代求值 + 循环引用检测）。
- **几何内核自研**：索引三角网格 + 球/椭球/圆柱/圆锥/圆台/圆环/多面体/挤出（earcut 风格耳切三角化，
  **含带孔轮廓挤出**：eliminateHoles 桥接 + 零宽通道）/旋转体 +
  **BSP 实体布尔**（csg.js 同源算法：平面分裂 → 树裁剪，迭代式建树避免深递归爆栈）+ 变换/阵列
  （linear/circular/mirror）+ 拓扑统计（边界边/非流形边/共边同向/欧拉特征/连通体）。
- **二进制产物**：沙箱无 Buffer，故自研 **IEEE754 单精度编码** + **base64 编码/解码**，经
  `ctx.fs.writeFileBase64` 落盘（binary STL / GLB 都是这么写出来的）。

### I.3 验收数据（实测）

| 项 | 结果 |
|---|---|
| 本地测试（`_temp/model-plugin-test.cjs`，工具链端到端） | **78/78 通过** |
| 内核算术（`_temp/model-core-test.cjs`） | **45/45 通过** |
| 表达式/形状树/阵列（`_temp/model-stage2-test.cjs`） | **46/46 通过** |
| **带孔轮廓挤出**（`_temp/model-holes-test.cjs` + `_temp/model-holes-e2e.cjs`，见 I.7） | **18/18 + 24/24 通过**；体积对解析值 rel ≤ 5e-15；χ=0/−2/−4 与通孔数一致；同尺寸挖孔比布尔**少 20× 面、快 50×** |
| **官方 glTF 校验器**（Khronos gltf-validator 2.0） | **errors=0 / warnings=0**；accessor min/max、bufferView 4 字节对齐、data URI 自包含全部合规 |
| **外部真值①（JSCAD 交叉）** | 基本体体积/包围盒与 JSCAD **rel ≤ 1e-14**（cuboid/sphere/cylinder/torus/extrude 逐项） |
| **外部真值②（JSCAD 布尔交叉）** | subtract/union/intersect 体积 **rel ≤ 1e-6**；面积**完全相同**（1e-12） |
| **外部真值③（STL 往返）** | 自研 binary STL → `@jscad/stl-deserializer` 读回 → 体积与解析值吻合（rel 1e-2 内，含 float32 与碎片效应） |
| 预览（`_temp/model-preview-shot.mjs`，真浏览器 WebGL） | WebGL 初始化 ✓ 几何上传 ✓ 无 JS 异常 ✓ 无 console.error ✓ 4 张多视角截图（默认/旋转/线框/近景） |
| 面板（`_temp/ui-model-panel-verify.mjs`，CDP） | **15/15 断言通过**：分区齐全、iframe sandbox 正确、srcdoc 256 KB、9 项判据全渲染、徽标 8/9、部件 3/参数 10/产物 5、**无 `/api/api` 双前缀**、确实读了 model.json+verify+preview；截图 225 KB / 全览 248 KB |
| 装配（9097 测试实例） | 插件 **41**（39+2）、工具 **≥129**（124+5）、ui-boot entries **≥15** 且含 ui-model、5 个 `model_*` 工具已注册 |
| 硬门槛 | `go vet` exit 0；`go build ./cmd/companion` exit 0；`go test -short ./internal/agent` **ok 80.3s**；`verify-dist-isolation` **PASS**；落地矩阵 PASS |
| 示例（工作区根 `model.json`，带肋法兰支架） | 3 部件 / 7,901 面 / 10 参数（含表达式参数 `bossH = thk*4`）/ 体积 21861.08 mm³；判据 **fails=0**、warns=1（M4） |

### I.4 本次暴露并修复的缺陷（均实测定位，勿回退）

1. **revolve 的 profile 未按"闭合轮廓"环回** → 内柱面缺失：体积为负且网格不闭合（b=256）。挤出/旋转体的
   轮廓语义必须一致（末点自动连回首点，重复点退化由跳过逻辑处理）。
2. **BSP 的共面三角形归属**：建树时共面三角形应**存本节点**、裁剪时按朝向分流（csg.js 语义）；
   写成"再判一次是否共面"会因浮点比较产生误分类。
3. **★ 自研"T 型接缝缝合"有害，已撤销**：把落在边内部的顶点纳入三角形重新三角化，实测引入
   **1160 条非流形边**并把三角形数量顶到 **74092**（细分爆炸，原 10160）。改用与 JSCAD generalize 同法的
   **顶点吸附**（空间分辨率 `EPS_SPATIAL = 1e-5`，照 JSCAD `maths.EPS`）+ 焊接 + 去零面积面 —— 网格保持干净
   （无非流形、无重叠），缝的**残余量如实交给 M4 报告**。
4. **`b64DecodeBytes` 未先剥掉尾部 `=` 填充** → 主循环把"含填充的末块"也当完整块，大文件**多解 1 字节**
   （正是 M8 的"binary STL 长度 = 84+50n"校验抓到的；小文件往返测不出来，必须用真实产物）。
5. **片元着色器 `return}` 缺分号** → 预览页整页静默白屏（只有真浏览器才暴露；`srcdoc` 静态检查看不出来）。
   已修，并在着色器编译失败时把原因写进页面 HUD（不白屏）。
6. **预览渲染器的法线缓冲布局写错**（`verts[o*2]` 应为独立法线数组的 `nrm[o*3]`）→ 光照全错。
7. **WebGL1 大网格需要 `OES_element_index_uint`**：顶点数 > 65535 时必须探测扩展并切 `Uint32Array`，
   否则索引溢出（静默画错或丢部件）。
8. **2D 矩形轮廓误用三元组校验** → `size:[20,20]` 被当成"长度不足 3"拒绝（2D 轮廓是 `[w,h]`）。
9. **半边"配对"判定写错**：有向半边在闭合网格里本就各出现一次，"配对"= **反向半边存在**；
   写成"同一有向边计数 > 1"会把全部半边判成未配对（cuboid 也会报 36 条未配对）。
10. **圆周阵列两处错**：基向量依赖 helper 选择（起始角不可预期）→ 改为"z 轴取 +x / y 轴取 +z / x 轴取 +y"；
    且"绕中心旋转 + 平移到环上"双重位移 → 改为 `M = T(center)·R(axis,ang)·T(u·radius)`。
11. **★ `plugins-dist/tool-model/` 缺 `package.json` → `dev-sync-dist-plugins.mjs` 静默跳过**
    （"link 完成 1/11"里没有它）→ 后续 `verify-dist-isolation` 报"exclude 中的 tool-model 不存在"。
    这正是上一轮写进 `.pair/project.md` 的坑，本轮自己又踩了一次：**独立发布插件必须有 manifest**。
12. **文本产物的字节统计**：`files[].bytes` 必须用 UTF-8 字节数（`utf8Bytes(text).length`），
    用字符数会让"文件实际大小 ≠ 报告大小"（中文/多字节内容）。
13. **面板需要每部件明细**：面数/体积/尺寸只有构建后才知道，故 `model_verify` 的旁挂报告补 `parts[]`
    （id/类型/运算数/阵列/面数/体积/尺寸/可见性）供面板直接读。

### I.5 验证方法学补充（本域新增）

- **官方校验器当验收门槛**：Khronos gltf-validator 直接给 `errors/warnings` 计数与结构化 messages，
  比自研结构断言硬（本域 `errors=0 / warnings=0`）。同时它暴露了自研判据的一处实现 bug（M8 曾把 NORMAL
  accessor 也当 POSITION 要求 min/max —— 规范只要求 POSITION）。
- **外部实现当"真值"**（第二次验证这条方法学，上次是 rig 域用 ag-psd）：JSCAD 既能做 CSG 的数值对照
  （rel 1e-14），又能读回我们导出的 STL（`deserialize` 要传 latin1 字符串，Node 22 下传 Buffer/Uint8Array 会崩）。
- **"srcdoc 静态检查" ≠ "预览真的能跑"**：预览页必须用真浏览器加载断言（本域的着色器缺分号就是静态检查漏掉的）。
- **iframe 内预览的 CDP 上下文不可达**（`sandbox="allow-scripts"` 无 `allow-same-origin` → opaque 源）：
  正确做法是**用独立脚本在真浏览器里直接打开同一份内容**验证，而不是在父页里钻 iframe。
- **水密性必须分级报告，不能一刀切**：BSP 三角网格布尔的输出在"严格边匹配"意义下**本来就不水密**
  （JSCAD 原始 subtract 输出 boundary=152 / 1200 条半边 = 12.7%）。本域 M4 用**有向半边配对 + 投影区间
  覆盖**把"T 型接缝（几何闭合、分段不同）"与"真缺口"分开数，判据只对真缺口报警 —— 既不给用户假安心，
  也不把算法固有代价说成缺陷。

### I.6 遗留

1. **BSP 碎片**（本域最大质量短板）：布尔结果面数偏大 —— 60×40×6 底板挖 4 个 ∅6 孔得 **5823 面**
   （理想 ~100 面），直接后果是 glTF 826 KB / STL 395 KB 与预览里的锯齿感。改进方向明确：
   **共面三角形合并**（按平面分组 → 簇内边界环 → 单环重三角化 + 面积守恒校验，失败则回退），
   同时能消掉大部分 T 缝。未在本轮做（工作量大、风险需单独验证）。
   → **本轮已做**（见 §I.6-1）：干净簇可合并（单孔场景 **883 → 607 面 / −31%**），
     但含孔密集切割的顶/底面簇**因 BSP 输出的重叠而无法安全合并**（详见该节的根因分析），
     四个孔的主场景实测 3479 → 2975（−14.5%）。**根治已由 §I.6-2 完成并接线为默认路径** —— 多边形 BSP
     路径把同一场景做到 768 面（−74.2%）、边界边 1451 → 928（见 §I.6-2）。
2. **M4 的真缺口**（连续布尔累积）：示例实测 2256 条 / 23703 条半边 = **9.5%**（其中 T 缝 149，
   真缺口 2107）。与 JSCAD 同量级（其原始输出 12.7%，靠"多边形表示 + generalize"才降到 0）——
   要做严格水密必须换多边形表示（BSP 输出保持多边形到最后一刻再三角化），属结构性改造。
   → **本轮已接线并显著改善，但未归零**（见 §I.6-2）：工具级实测（4 次串行布尔挖 4 孔）
     真缺口 **1753 → 832（−52.5%）**、未配对半边 1922/23166 → 1040/6630、非流形边 2 → 0；
     但 M4 在**两条路径下均为 fail** ⇒ 这是**既有**结构性缺口（多边形表示只消除 T 缝类离散，
     未消除真缺口），**不是本次切换默认路径引入的回归**。后续需单独排查 BSP 裁剪的边界情形。
3. ~~**2D 带孔轮廓挤出**（earcut with holes）未做 → 挖孔只能靠布尔~~ → **本轮已完成**（见 I.7）：挖孔
   不必再走布尔，碎片与耗时都显著下降（同尺寸板挖 2 孔：**276 面 / 4 ms** vs 布尔 **5763 面 / 218 ms**）。
4. ~~**扭转挤出（twist）/ 扫掠（sweep）/ 圆角-倒角（fillet/chamfer）** 未做~~ → **twist/sweep 已完成**（见 I.8）；
   **chamfer 已完成**（凸体走 H-rep：解析精确 + 严格水密），**fillet 已完成到「互不相邻的棱」**（见 I.9）；
   ~~圆角的顶点混合未做~~ → **顶点混合已完成**（见 I.9.6：凸体全棱 = 形态学开运算 `(P ⊖ Ball_r) ⊕ Ball_r`
   的解析拼接，`P'` 顶点处直接生成球片 ⇒ 立方体 12 棱严格水密）；
   剩余仅**凹边填角**未做（明确报错而非静默降级）；
   `polyhedron` 仍需手写顶点面表。

#### I.6-1 共面三角形合并（2026-09 清遗留，已完成/部分达成）

> **目标**：把 BSP 布尔输出里同一平面上的碎片三角形合并回大面，同时消掉大部分 T 缝。
> **口径**：合并**只允许变小，绝不允许变样** —— 三条不变量优先于降幅：
> ① 不引入任何新顶点（合并后每个坐标逐位来自原表）；② 体积/面积守恒；③ 拓扑不劣化
> （boundary / nonManifold / flipped 不得增加）。任一环节不通过就**整簇回退**。

**算法**（`meshMergeCoplanar` / `coplanarMergeCluster`，接在 `meshRepair` 之后）：

1. **平面分组**：每个三角形算出单位法线 + 偏移 `d = n·p`；先按法线粗桶（1e-3）分桶，再桶内按
   `d` 聚类（容差 = 空间分辨率 `eps`）。只合并**同朝向**的（薄壁内外两面共面但朝向相反）。
2. **簇内 T 缝细分**：顶点落在某条**三角形边**内部 ⇒ 按投影参数插入该边，三角形 1→n+1 分裂。
   ★ 必须对**所有**边做，不能只对边界边 —— 实测最大簇的 160 处 T 缝**宿主边多数是内部边**。
3. **噪声清理**（仅在检测到非流形时触发）：完全重复的三角形去重 + 面积 ≤ 4 格²的微碎片剔除
   （BSP 在孔/边附近浮点噪声的产物，正是"一条短边被 5 个面共享"的制造者）。
4. **边界环提取**：边界跟随（平面图的面遍历）—— 在顶点处选"相对入边反向的最小左转"出边，
   因此允许顶点出度 > 1（BSP 边界可能在一点自接触）。零面积环（零宽通道）忽略。
5. **环嵌套分类**：按包含关系建父链，深度偶 = 外环（CCW ⇒ 有向面积 > 0）、奇 = 洞（CW ⇒ < 0）；
   方向与深度不符即回退（这条正是拦下 BSP 重叠输出的关口）。
6. **重三角化**：每个外环 + 其直接子环（洞）交给既有的 `triangulatePolygonWithHoles`
   （I.7 的 earcut 桥接实现）；坐标先除以 `eps` 变成整数格点，让该函数的 `EPS` 判据落在正确量级。
7. **双重守恒校验**：① 强 —— 三角化输出必须恰好铺满环区域（浮点级）；② 弱 —— 环区域与原三角形
   覆盖一致（容差 1e-5 相对，用来记账 T 缝细分引入的 O(eps·L) 漂移）；再加**输出不得重叠**
   （每条有向边与反向边次数之和 ≤ 2 且必须一正一反，否则回退）。最后要求面数确实减少。

**实测**（`_temp/model-merge-coplanar-test.cjs`，40 项断言全通过）：

| 场景 | 面数 | 降幅 | 合并/回退簇 |
|---|---|---|---|
| 60×40×6 板 − 1×∅6 孔 | **883 → 607** | **−31.3%** | — |
| 60×40×6 板 − 4×∅6 孔（I.6 记录的 5823 面场景） | 3479 → 2975 | −14.5% | 3 / 20 |
| 60×40×6 板 − 8×∅5 孔 | 6757 → 5797 | −14.2% | 4 / 78 |
| 板 ∪ 4 柱 | 3855 → 3351 | −13.1% | 3 / 111 |
| 板 ∩ 4 柱 | 1103 → 1097 | −0.5% | 2 / 39 |
| 不变量 | 不引入新顶点 ✓ / 体积·面积守恒 ✓（实测 rel ≤ 4e-8）/ 拓扑不劣化 ✓ / 幂等 ✓ | | |
| 无回归 | model-core 45 / holes 18 / stage2 46 / twist 85 / sweep 52 / sweep-e2e 36 / plugin 78 / chamfer 55 / rb-fillet 47 = **462** | | |

**根因（本轮最有价值的发现）**：孔密集切割的顶/底面簇**合并失败是正确行为**，不是实现缺陷：

1. T 缝的宿主边**多数是内部边** ⇒ 只消解边界边不够（首版就卡在这里：21 个顶点出度 = 2）。
2. 孔边缘的 BSP 输出**存在重叠** —— 表现为环提取后出现「**同号嵌套**」（一个正面积的小环落在
   正面积的外环内）。若"聪明地"把这类小环当噪声过滤掉，就会把孔**填平**（覆盖率偏差仅 1e-6 相对，
   弱校验放行）⇒ 静默产出错误几何。因此实现的判据是**直接回退**：宁可留碎片，绝不改形状。
3. 结论：**在当前三角网格表示下，BSP 的共面簇不保证是有效平面剖分** ⇒ 重三角化没有安全前提。
   根治要等 §I.6-2（BSP 输出保持多边形到最后一刻再三角化）。

**本轮踩掉的两个坑**（都是"看起来对、跑起来错"）：

| # | 现象 | 根因 | 修法 |
|---|---|---|---|
| 1 | 合并率恒为 0，面积基准算出 `0` | 串环游标也命名为 `cur`，与细分后的三角形数组**同名** ⇒ `var` 提升后数组被覆盖成数字 | 游标改名（`wcur`）；耗时最久的一次定位 |
| 2 | 单孔场景冒出 **4 条非流形边**、体积偏差 4e-8 | T 缝距离容差取了 1 个吸附格（eps）⇒ 把"靠近边的其他顶点"也当成 T 缝插入 ⇒ 产生重叠面；另试过「短边塌缩」，它**移动顶点改几何**且没解决非流形 | 容差收紧到 eps/4；短边塌缩**删除**；并补「输出不得重叠」校验（正是它拦下了那 4 条非流形边） |

**能力边界（如实标注）**：合并只在"该平面簇的原三角形已是有效剖分（无重叠、无真缺口）"时生效；
含孔密集切割的簇一律回退，回退原因可在 `CP_REASON.counts` 里查到（不是黑盒）。
**要拿到理想面数，优先走 I.7 的 earcut 带孔路径**（同尺寸板挖 2 孔 **276 面 / 4 ms**），
布尔只用于真需要 CSG 的场合。

#### I.6-2 多边形 BSP 路径（2026-09 清遗留，**已接线为默认路径**）

I.6-1 的结论是"在三角网格域无法根治"，本轮按该结论动手：**让 BSP 全程保持多边形表示，导出前才三角化**
（这正是 JSCAD 靠 `generalize` 拿到 boundary=0 的原因 —— 代码里早有 JSCAD 对照记录）。
新增独立函数族 `polyBsp*` + `meshBooleanPoly`；原型达标后已**接线为 `meshBoolean` 的默认路径**
（三角版改名 `meshBooleanTrig` 保留为回退/对照，两者仍可逐位比对 —— 见本节末「接线」）。

关键在于：**多边形表示下"共面分组"是免费的**。同一个 BSP 节点的 `polys` 天然共面（节点平面即其平面），
于是合并退化成两步 —— ① 有向边对消（内部边正反各一次，删掉）；② 剩余边串环（出度 ≤ 1 时直接沿边前进）。
I.6-1 在三角域必须做的 T 缝细分、噪声清理、容差匹配**全都不需要**：边是严格共享的。

| 场景（60×40×6 板，∅6 柱 @24 段） | 三角路径 面/边界 | 多边形路径 面/边界 | 面数降幅 |
|---|---|---|---|
| 板 − 1 孔 | 771 / 317 | **222 / 214** | −71.2% |
| 板 − 4 孔（I.6 主场景） | 2975 / 1451 | **768 / 928** | **−74.2%** |
| 板 − 8 孔 | 7392 / 3226 | **1405 / 1787** | −81.0% |
| 板 ∪ 4 柱 | 3300 / 1474 | **1148 / 952** | −65.2% |
| 板 ∩ 4 柱 | 1100 / 1108 | **600 / 590** | −45.5% |

I.6-1 只能到 2975 面且**整簇回退**，I.6-2 直接到 **768 面**（目标"数百面"达成），边界边同时降 36%。
两路径体积/面积互差 < 1e-6，非流形边 0、反向边对 0、两次运行逐位一致（37 项断言全通过）。

**安全回退（本路径独有优势）**：任一节点合并失败 ⇒ 该节点退化为「逐多边形扇形三角化」。
因为初始多边形是三角形、凸多边形被平面切割仍凸 ⇒ **凸性不变量**保证扇形三角化永远正确，
所以多边形路径**不存在产出坏几何的风险面**，最差只是少合并一些面（对比 I.6-1 的"必须整簇回退"）。

**回退点只剩一处**：2/106 个节点，原因固定为「边界在顶点处分支」—— 共面多边形集合在某顶点处出度 > 1
（多边形只在该点汇聚）。当前直接回退；补「最小左转」串环即可吃掉，面数还能再降一点。

**已知遗留（非本路径引入）**：subtract 的挖孔体积相对解析正多边形真值偏大 ≈0.85 mm³（≈0.5%），
而 intersect 与解析值**完全吻合**（4×∅8 柱 = 1192.63 vs 解析 1192.63）。该偏差在本次改动前即存在
（三角路径数值一字未变），属既有 BSP/`meshRepair` 行为；不水密网格的体积积分本身有该量级误差，
需另行排查。

#### I.6-2 接线（2026-09，已完成）

**改法**（`plugins-dist/tool-model/index.js`；面很小 —— 两路径签名完全一致 `(meshA, meshB, op, tol, statsOut)`）：
原 `meshBoolean` 改名 `meshBooleanTrig`（三角路径 + I.6-1 共面合并，一字未改）；新 `meshBoolean` 只做分发：

```js
var meshBoolLegacy = false;
function meshBoolean(meshA, meshB, op, tol, statsOut) {
  if (statsOut) statsOut.path = meshBoolLegacy ? 'trig' : 'poly';
  if (meshBoolLegacy) return meshBooleanTrig(meshA, meshB, op, tol, statsOut);
  return meshBooleanPoly(meshA, meshB, op, tol, statsOut);
}
```

**回退开关**（无需改代码）：`setMeshBooleanLegacy(true)`，或插件 config 给 `legacyMeshBoolean: true`
（`apply(ctx, config)` 读它）。`statsOut.path` 回填实际所走路径，供诊断区分。

**接线前的三项实测调研**（决定了**不**接"poly + 收尾共面合并"）：

1. **poly 输出已是合并饱和态**：对 4 个场景的输出再跑 `meshMergeCoplanar`，3 个 `merged=0`（含 4 孔主场景），
   唯一有增益的 ∩ 仅 **−6 面（1%）**，却带来 22~123 簇回退 ⇒ 不接收尾合并。
2. **边界场景对照**（poly vs 三角）：不相交 / 完全包含（结果为空）/ 完全重合 / 同尺寸 intersect
   **四项结果逐位一致**；L 形凹体两例（501→184 面、828→262 面）与相切 union 三项 poly **全面更优**；
   仅「两实体**仅共面接触**」一项轻微劣化（50 面/22 边界 vs 46 面/18 边界，面 +4）—— 两者都不水密（退化输入），
   体积一致、非流形 0，判为**可接受**。
3. **性能**：poly 快 3~6×（板 − 24 孔：**9962 面/350ms → 2887 面/59ms**）。

**工具级端到端验收**（`_temp/poly-wire-e2e.cjs`：**真工具链** `model_doc` → `model_add` → `model_verify` →
`model_export`，两套独立模块实例对照，**18/18 通过**）——场景 60×40×6 板 + 4 个 ∅6 柱 subtract
（工具层是 4 次**串行**布尔）：

| 指标 | 默认（poly） | 回退（`legacyMeshBoolean`） | 变化 |
|---|---|---|---|
| 三角面 | **2210** | 7722 | **−71.4%** |
| M4 真缺口 | **832** | 1753 | **−52.5%** |
| 非流形边 | **0** | 2 | 更干净 |
| 体积 | 13730.0797 | 13730.0967 | Δ=1.24e-6 |
| 9 项判据硬失败 | 无 | 无 | — |

⇒ **poly 全面优于或等于三角路径、无任何指标劣化**，故切为默认。注意工具层面数是测试脚本（一次性布尔）的数倍：
`ops` 数组是**逐个串行**布尔的（4 次），此为既有实现语义，非本次改动引入。

**回归口径的调整（重要）**：切默认后 `meshBoolean` 语义变了，**I.6 系列测试/诊断脚本的基线必须显式指向
`meshBooleanTrig`**，否则会"自己对照自己"而失去意义（`model-poly-bsp-test` / `model-merge-coplanar-test` /
`model-merge-diag` / `model-merge-goja-cmp` 已改；替换后逐文件**回读校验**，并确认未误伤
`meshBooleanPoly` / `meshBooleanRaw`）。`model-core-test` / `model-chamfer-test` / `model-rb-fillet-test` 等
**故意不改** —— 它们测的是布尔**结果正确性**（路径无关），现在顺带覆盖默认路径，价值更高：实测仍**全绿**。

**接线后回归**：I.6-2 **37/37**、I.6-1 **40/40**、接线自检 **14/14**（默认=poly、开关可回退且逐位一致、
复位一致、三算子两路径几何守恒）、通用回归 5 套全绿。**并用接线前版本跑同一批脚本建立基线**，确认失败项归属：
本次改动**只影响 2 个 I.6 诊断脚本的基线**（已修）；`model-diag`/`model-diag2`（导出名陈旧）、
`model-holes-e2e`（带孔挤出 cap 反向三角形）、`model-demo`（1 项）在**接线前即失败**，与本次无关。

**下一步**（2026-09-17 更新）：带孔挤出 cap 的反向三角形已修复（见下节）；M4 真缺口的根因已定位到
**布尔内核拓扑**，需独立课题。补最小左转串环吃掉最后的 2/106 回退节点（面数还能再降一点）；
I.6-1 的三角域共面合并已随 `meshBooleanTrig` 退居回退路径。

#### I.6-2 后续：cap 三角化重写为 earcut + M4 真缺口根因定位（2026-09-17，已完成）

**问题**（`_temp/model-holes-e2e.cjs`：带孔挤出 40×30 板 + 2 圆洞 + 1 方洞，**接线前即失败**）：
直接抛错「带孔挤出的底/顶盖三角化出现反向三角形 —— 检查第 51 个底盖三角形」。

**根因**（诊断法：对 `index.js` **真实源码**注入日志跑，不手抄算法）：
cap 三角化是自研的「earcut 风格」桥接 + 耳切，但它把 earcut 的**两处兜底换掉了** —— 代码注释写着
「earcut 用 `cureLocalIntersections` 兜底，这里改用对角线可见性从源头拦住」。拦得过狠 ⇒ 一轮找不到耳 ⇒
兜底退化成「取最大凸角强切」⇒ 强切切穿零宽桥接通道。实测 3 个洞时 70 个 cap 三角形里 **19 个是负面积（27%）**；
桥接还会退化到「最近可见顶点」，把通道从 `(3,12)` 拉到 `(-20,-15)` 横穿整个形状。

**修法**：完整移植 mapbox/earcut（`ec` 前缀、零依赖、ES5）：`filterPoints` → `cureLocalIntersections` →
`splitEarcut` **三级递进兜底**，替换「硬拦 + 强切」。省略 z-order 哈希（数据量小、无收益）；
坐标约定与 earcut 同构（外环有向面积 > 0），故**逐字照搬、零符号翻转**。

**效果**：该 e2e **24/24 全绿**；M4 从「208 条真缺口」变为「**全部部件严格水密**（有向半边全部配对）」，
jscad 读回 STL 体积 rel 6.7e-10，glTF 零 error/warning。

**★ 但 earcut 会让布尔路径劣化 ⇒ 必须分层**（实测，勿把两条路径合并成一个实现）：
earcut 为绕开自接触会走 `splitEarcut` 分割，产出的三角形**邻接更碎**，使下一轮 `meshToPolys` 的共面合并变差：

| 4 次串行 subtract 场景 | 旧（桥接+耳切） | 换成 earcut |
|---|---|---|
| polyNodes | 388 | **677** |
| polyInput | 1913 | **2502** |
| 三角面 | 2216 | **2797** |
| M4 真缺口 | 893 | **1255** |

⇒ 最终分工：**挤出/扭转挤出/扫掠/回转的端盖** 用 earcut 版（`triangulatePolygon` /
`triangulatePolygonWithHoles`）；**I.6-1/I.6-2 的共面轮廓收尾** 保留旧实现
（`...Legacy` 两个函数，由 `coplanarMergeCluster` 与 `polyNodeToTris` 显式调用 —— 它们**不是死代码**）。
实测布尔路径**逐位回到基线**（poly 2210 面 / M4 832；trig 7722 / 1753）。

**新增防护：边界完整性校验**（`ecFinalize`）。面积守恒抓不住「重叠与缺失互相抵消」，移植时就踩了一个真实的坑：
earcut 的 `filterPoints` 会**删共线点**，而轮廓上的共线中间点是**侧壁的顶点** —— 删掉后 cap 边界比侧壁少边
（实测「板 + 2 圆孔 + 1 方洞」boundary=6、χ=−6，`model-holes-test` 与 `model-twist-test` 各挂 1 项）。
修法：只删重复点、不删共线点；并补**边界一致性校验**（边界边 = 找不到反向边的有向边，须与输入各环边一一对应），
让这类缺陷当场报错而不是流到下游。

**★ M4 真缺口：根因已定位，但未归零（需独立课题）**。它既不是三角化问题、也不是焊接容差问题：

- 把 `meshRepair` 的焊接系数从 0.5×eps 一路放到 **100×eps（≈35 μm）**，真缺口只从 893 降到 **763**（平台期）；
- 缺口边抽样全部落在 `z≈-3.0002`（板底面）：`meshSnap` 把顶点吸附到 eps 格点（3.53e-4）后，本应重合的点
  会落在**相邻格点**（间距恰为 eps），而 `meshWeld` 容差只有 eps/2 ⇒ 永远焊不上（这是 0.5→1 那 9% 增益的来源）；
  平台期的 763 条属**真开边**（BSP 输出拓扑不闭合）。

⇒ 结论：**修它要动布尔内核的切割/缝合（顶点与边的拓扑一致性），是独立课题**，本轮不在这条线上冒险。

#### I.6-2 后续三：M4 真缺口**归零**——T 缝消解（2026-09-17）

「后续二」把真缺口从 893 压到 756 后，残余被定性为"eps 尺度零面积细缝"，本轮把它**做到 0**。

**真根因（诊断链终点）**：**各次布尔使用各自的 TOL**——`TOL = weldTolerance(合并后输入)` 随输入
精度递增，于是同一条长边在相邻两侧的**分割点跨次错开 ~1~1.4 eps**，任何焊接容差都合并不掉，
在半边的严格意义上永远找不到配对。实例：`(-20.2330616,-15.1227675)` 与
`(-20.2334149,-15.1231208)`，相距 5e-4（1.4 eps）。

**决定性实验（分辨"挪坐标"还是"改拓扑"）**：把 541 个"落在边内部"的顶点吸附到边/端点 —— 真缺口
**758 → 750**（几乎无效），因为只挪坐标不改分段方式，对面依然没有对应的边。改为**拓扑分割**
（把分割点真正插进边、切开它）立刻 **758 → 88**；修掉三角化方式后 → **6**；再把焊接比调到 1.5×eps
即 **0**。

**落地四项**：

1. `meshAbsorbToEnds` —— 顶点落在未配对边内部、且**沿边方向距端点 ≤ eps** ⇒ 并入端点（吞掉微边）。
2. `meshFillTJunctions` —— 落在未配对边内部的**真分割点**插入该边。★ 两个硬伤正是历史
   "T 缝缝合有害（非流形 + 30 万面）"的真正原因：
   · **必须从该边的对顶点出发、逐条边分割并递归**。对整条边界链扇形会把"插入点 + 该边两端点"
     三点共线拼成零面积三角形 ⇒ 4 条非流形边，同时丢掉角三角形 ⇒ 11 条单边。
   · 插入点排序**必须稳定**（t 相同时按顶点索引），否则相邻三角形里顺序不同 ⇒ 同向重复边。
3. `WELD_RATIO = 1.5`（原 0.5）—— 消解会暴露出"横向错开 1~1.4 eps"的成对顶点。扫描：
   0.5×→真缺口 10 / 单边 11；1×→1 / 3；**1.5×→0 / 0（拐点）**；2×/3× 无进一步收益。
   `meshRepair` 与 `meshWatertightReport` **必须同容差**，否则"修好了却仍报缺口"。
4. **不劣化保护**（`meshSeamScore`）—— 消解是启发式的，在"多簇共面合并"类输入上会帮倒忙
   （板 − 8 孔实测 10 → 30）⇒ 消解后严格未配对数变差即**回退**。附带好处：已水密则跳过消解。

**效果**（60×40×6 板 − 4×∅6 柱 @24 段，4 次串行 subtract）：真缺口 **758 → 0**、严格未配对
**1000 → 0**、T 缝 **242 → 0**、**缝总长 0 mm**、非流形 0、绕向错 0、连通分量 1、体积相对变化 1.3e-6；
union / intersect / 单孔场景同样为 0。**代价**：面数 **4416 → 4536（+2.7%**，消解补回面）、耗时 558 ms
——对可打印性是划算的取舍。

> ★ **更正（2026-09-17 复核）**：本节曾写"2102 → 4536（+116%）"，那是**错的口径对比**。2102 出自
> `_diag-m4-gap.cjs` 中"诊断脚本自身用不同 eps 焊接后得到的另一个网格"（该脚本第 29 行已注明
> "两边的数字就不可比"，实测 tris 2103 vs 2102）。**同口径实测**（`_temp/_diag-m4-size.cjs`）：
> repair 前的 base（已焊接/去退化）**4416 面** → repair 后 **4536 面**，真实增长 **+2.7%**；
> 表面积 6227.6263 → 6227.6260（−4.7e-6%，即 T 缝那点微面积）；无零面积面（最小 2.871e-6）。
>
> ★ **为何不做"共面合并压缩"（实测结论，此后勿再试）**：对已水密的 4536 面网格调用
> `meshMergeCoplanar` 得 1187 面，但**同时把水密性弄坏**（非流形 1、严格未配对 2、缝长 7.86）——
> 合并按簇重三角化，新边与邻居的分段方式对不上，这些缝是**合并自身的产物**；再走 `meshRepair`
> 也修不回来（1187 → 1187、缝量 2 不变，`meshSeamScore` 判定消解帮倒忙而回退）。
> ⇒ **面数换水密是必要取舍**，两者同时追求必然振荡；这也正是默认路径 `meshBooleanPoly` 不做
> 共面合并的原因（`meshMergeCoplanar` 只存在于 legacy 分支 `meshBooleanTrig`＝测试基线）。

**M4 判据口径调整（选项 4）**：报告新增 `gapLength`（未覆盖缝总长，mm）/`gapMax`/`printableClosed`
（缝长 ≤ eps/10 视为闭合）。理由：**"条数"会被同一条缝被切成很多段而放大，长度才是真实缺陷量**。
`model-merge-coplanar-test` 的 `topoKey` 也从"严格索引口径"改为"几何闭合口径"——合并按簇重三角化
必然产生 T 缝，在严格索引口径下永远计入 boundary，但它几何闭合、不影响可打印性（这正是判据要
量"几何"而非"拓扑记账"的理由）。core-test 的断言同步由 `strictUnpaired > 0`（历史"本就不水密"）
升级为 `strictUnpaired === 0 && tolerantUnpaired === 0` + `printableClosed === true`。

**回归**：13 套全绿（core 46 / holes 18 / twist 85 / holes-e2e 24 / sweep 52+36 / poly-bsp 37 /
plugin 78 / merge-coplanar 40 / chamfer 55 / rb-fillet 47 / stage2 46 / poly-wire-e2e 18）。

**方法论教训（口径统一）**：诊断脚本必须与生产**同 eps、同容差**。曾出现"用 `eps0×0.5` 焊接出的
网格"却拿"焊接后的 `eps`"判定（tris 2103 vs 2102）⇒ 基于它的"726 条覆盖满"结论全属假象；
正确做法是把 `tol` 显式传给 `meshWatertightReport(m, eps0)`。另外 `onSeg` 的 **t 范围检查**不可省：
只查"落在直线上"会把边外延的候选也算进来。

#### I.6-2 后续二：M4 真缺口根因修复（2026-09-17，三项，均零回退）

**诊断方法**（这是本课题真正的产出 —— 每条结论都有可复现的量化证据）：

1. **真实源码注入日志**（不手抄算法）：把 `meshWatertightReport` 的判定分支改成记录
   `[顶点索引 a → b, ivs, cov]`，把 `polyNodeToTris` 入口记录节点平面 —— 定位到"缺陷集中在
   板面、且 762/893 条对面完全无反向覆盖"。
2. **覆盖测量**：把板面三角形投到 2D，用 0.02 格网格采样，以**柱的真实 24 边形**（不是圆！）
   扣除孔内，统计每点落入几个三角形（0 = 洞 / ≥2 = 重叠）。首次用"圆"做真值算出 Δ=+1.32 的
   假象（那只是 24 边形与圆的面积差 0.32），改用 24 边形后 Δ 仅 0.0014%。
3. **暴力 O(n²) 独立判定**：对每条严格未配对半边遍历全部半边做"方向相反 + 落在线段上"检查，
   与 `meshWatertightReport` 逐项对照 —— **完全一致**（993/202/791）⇒ 报告逻辑无误，缺口真实。

**被证伪的假设（记录以免后人重走）**：

| 假设 | 实验 | 结果 |
|---|---|---|
| cap 三角化实现有问题 | 换成 earcut 版 | **更差**：面 2797、真缺口 1255、z 层 185 |
| 微环被 `abs(ar) <= 4` 丢弃 | 改成不丢 | **逐位不变**（该分支从未触发）|
| 环在某顶点处入度 > 1 | 注入入度检查 + 强制回退 | **逐位不变**（入度全部 ≤ 1）|
| 焊接容差不足 | 容差 0.25×eps → 256×eps 扫描 | 边界边仅 1114→899，**明显平台期** |
| 退化面被丢导致开边 | 保留退化三角形 | **逐位不变** |
| 在多边形上插点对齐 | 首版实现 | **有害**：10 条非流形边、面数 +50%、6 个反向三角形 |

**三项修复**：

1. **`EPS_PLANE` 按输入尺度动态设定**（`bspEnterPlaneEps`，取 2×meshEpsilon，try/finally 恢复）。
   原值 1e-6 比本场景空间分辨率（eps≈3.5e-4）小 350 倍 ⇒ 把"近共面"误判成"跨平面"：同一几何
   平面被拆进多个 BSP 节点，切割平面因此带微小倾斜，在离该多边形很远处切出 1e-2 量级坐标偏差。
   实测：**顶点 z 层 39→2**、面 2216→2099、边界边 1114→1021、反向三角形 **11→0**、真缺口 893→817。
   （倍数继续放大到 4×/8×/16× 无进一步收益，32× 体积变差 ⇒ **2× 已是拐点**。）
2. **量化坐标表跨节点共享**（`POLY_QUANT_COORD`，每次布尔清空）：板面节点与孔壁节点对"同一个
   几何顶点"各取各的代表坐标，边界错开约 1 格。实测：面 2099→2075、边界边 1021→993、真缺口 817→791。
3. **环边跨节点对齐**（`polyNodeToTris` 新增 ③′）：串环后把"落在环边内部"的其他节点顶点按参数序
   插入该环。**时机是关键**：插点必须在对消**之后** —— 首版做在"多边形"上（对消之前）会破坏对消
   （见上表），插在环上则对消已完成，只延长顶点序列、不改任何既有拓扑关系；两侧读同一张全局量化表
   故插点集合一致。实测：面 2075→2103、非流形 0、T 缝 202→243（错位边确实被转成 T 缝）、
   真缺口 791→756、耗时 0.63s。

**累计效果**（60×40×6 板 − 4×∅6 柱 @24 段，4 次串行 subtract）：

| 指标 | 修复前 | 修复后 |
|---|---|---|
| 顶点 z 层 | 39 | **2** |
| 反向三角形 | 11 | **0** |
| 三角面 | 2216 | **2103** |
| 真缺口 | 893 | **756** |
| 非流形边 | 0 | 0 |

工具级（真工具链）：面 2210→2103、**M4 真缺口 832→689**；回退路径 1753→1438。

**残余缺口的性质（已定性 ⇒ 独立课题）**：覆盖测量表明真实洞面积 **0.035**、重叠 **0.025**
（板面 24 边形真值 2288.19，Δ 仅 **0.0014%**）⇒ **几何已闭合**，残余是 eps 尺度的**零面积细缝**
（焊接把宽度 < eps 的细缝压成退化三角形）。**判据对照很重要**：JSCAD 原生 subtract 输出同样
不满足"严格边匹配"（boundary=152，靠 generalize 才归零）—— "严格口径归零"是**多边形化**才有的
性质。消除残余需要"多边形级**边吸附**/缝合"（把错位的平行边吸附到一起），
与本次的"点落在边上"（③′）不同类，风险与工作量另计。

**回归**：全量 model 脚本 + poly 脚本 **exit=0 全绿**（holes-e2e 24、holes-test 18、twist 85、sweep 36/52、
poly-bsp 37、plugin 78、merge-coplanar 40、core 45、chamfer 55、rb-fillet 47、stage2 46、poly-wire-e2e 18）。
逐项用**改动前基线**（`git stash`）确认归属；其中 `model-holes-test` / `model-twist-test` 的两处回归
（共线点被删）已在本轮修复。

### I.8 扭转挤出 / 扫掠 / 多洞 cap 修复（2026-09 清遗留续，已完成）

> 承接 I.6 第 4 项。原则不变：**几何真相源仍是参数化工程**（形状表达式树 → 三角网格），
> 新能力一律**解析可验证**，并在**真 goja 沙箱**里跑同一份源码复验（不只 Node 模拟）。

#### I.8.1 扭转挤出（`extrude` + `twist`）

**口径**：`extrude` 新增 `twist`（总扭转角，度，默认 0）+ `segments`（**扭转分层数**，默认 12，与轮廓自身的
圆周分段各管一段）。轮廓沿 z 分 `segments` 层，第 j 层绕 z 轴旋转 `twist·j/segments`：

```json
{ "id": "auger", "type": "extrude", "profile": { "type": "circle", "radius": 8, "segments": 24 },
  "height": 40, "twist": 180, "segments": 24,
  "holes": [ { "type": "circle", "radius": 3, "segments": 16 } ] }   // 带孔 ⇒ 内螺旋槽
```

**实现**（`extrudeTwistMesh`）：

1. 顶点表按「**层优先**」（层 j × 环内下标）——与直线挤出的「下块 + 上块」不同，故独立实现、互不影响；
   `twist` 缺省/为 0 时**走原路径**，网格逐位不变（零回归）。
2. 每层是**刚体旋转** ⇒ 洞随外轮廓同步转、环的拓扑与绕向都不变 ⇒ `holes` 的 2D 内嵌校核依然成立。
3. 端盖复用 `triangulatePolygon` / `triangulatePolygonWithHoles`（底盖反绕、顶盖正绕）；侧壁绕向与直线挤出同式。
4. 校验：**每层扭转角必须 < 180°**（否则相邻层顶点「走近路」反向跨接、侧壁自交）、面数上限
   （`2·segments·顶点数 + 2·端盖面` vs `LIMIT_TRIS_PART`）。

#### I.8.2 扫掠（`type: "sweep"`）

**口径**：`{type:"sweep", profile, holes?, along, closed?, twist?, scale?, up?}` ——
轮廓按局部 xy 平面贴在路径每一站的框架上。★ 路径字段叫 **`along`**：工具层的 `path` 已经是「工程文件路径」，
不能复用（这是实测踩到的参数冲突）。

```json
{ "id": "elbow", "type": "sweep", "profile": { "type": "circle", "radius": 4, "segments": 24 },
  "holes": [ { "type": "circle", "radius": 2.5, "segments": 16 } ],
  "along": [[0,0,0],[0,0,20],[0,8,30],[0,24,36]] }                 // 空心弯管
{ "id": "ring", "type": "sweep", "profile": { "type": "circle", "radius": 3, "segments": 24 },
  "along": [[20,0,0],[0,20,0],[-20,0,0],[0,-20,0]], "closed": true }  // 闭合 ⇒ 无端盖
```

**实现**（关键设计选择）：

1. **框架用平行传输（Rodrigues），不用 Frenet**：Frenet 的 `n = t'/|t'|` 在直线段（曲率 0）处**无定义**、
   拐点处还会突然翻转 —— 扫掠侧壁因此常常自交。平行传输（等价 RMF / double-reflection）对任意路径连续、
   且无扭转突变。起始法向默认取「与起始切线最不平行的坐标轴」，保证沿 +z 的直线扫掠退化为标准挤出
   （n=+x、b=+y）。
2. **闭合路径的 holonomy 修正**：把「末站框架转回首站」的角差 φ 按站号线性摊掉，接缝处才不拧。
3. **自交处理（三条，全部明确报错而非静默产出坏网格）**：
   - 路径非相邻段相交（3D 段段最短距离 ≤ `EPS_SPATIAL`）；
   - 站点**曲率半径 < 轮廓外接半径**（内凹侧侧壁必然穿透自身）—— 扫掠最常见的失败模式；
   - 相邻段 180° 折返（切线无定义）。
4. `closed` 默认自动（首尾点重合即闭合）；`scale` 支持标量（恒定）或与站数等长的数组（锥形/渐缩）；
   闭合时无端盖、非闭合时两端封盖（首站朝 −t、末站朝 +t）。

#### I.8.3 本轮发现并修复的既有缺陷：多洞 cap 三角化产生重叠面

> 做「带孔扭转」时暴露：**twist=0 的双洞挤出同样存在** ⇒ 是 I.7 带孔挤出的遗留缺陷，不是本轮引入。
> 症状隐蔽：**体积完全正确**（重叠三角形有向面积成对抵消），但拓扑坏（非流形、不可 3D 打印）。

| 缺陷 | 症状（实测） | 根因 | 修复 |
|---|---|---|---|
| 多洞桥接共用同一外轮廓顶点 | 双洞 cap 出现重叠三角形：Σ\|A\| 超解析 **133.8**、`nonManifold=2`、**7.2% 采样点被多次覆盖**（最大 5 次） | 两洞最右点在同一水平线上 ⇒ 射线命中同一顶点 ⇒ 环出现多次自接触 | 桥接**按「洞最右点 x」降序**处理（先右后左，通道不横穿未处理的洞）+ 候选点**避开已被桥接占用的顶点** |
| 耳切切穿自接触环 | 同上的重叠/缺口互补（面积守恒但覆盖重复） | 仅「三角形内无凹顶点」不足以判定耳：对角线可能切穿通道或跨到环的另一段 | 新增 `diagClear`：耳的对角线**不得与环的任一边真交叉**（earcut 用 `cureLocalIntersections` 兜底，这里从判定源头拦住） |
| 洞相交/嵌套未校验 | 环自交 ⇒ 三角化必然产生重叠面（同心双洞实测 26 个反向三角形） | 此前只校验「洞落在外轮廓内」 | 新增 `validateHoles`：洞之间**互不相交、互不嵌套**，违规明确报错；三角化输出若出现**反向三角形**立即报错 |

修复后实测：单洞 / 双洞 / 三洞 / 四洞 / 同 y 一排多洞 / 六边形+双孔 —— 采样**无缺口且零重复覆盖**、
`Σ|A|` 与解析面积差 ≤ 5e-13、`boundary=0 / nonManifold=0 / flipped=0`。

#### I.8.4 验收数据（实测）

| 项 | 结果 |
|---|---|
| `_temp/model-twist-test.cjs`（内核） | **85/85**：与 **@jscad/modeling `extrudeFromSlices`** 独立构造的扭转体逐项对比（体积 rel ≤ 6e-15、包围盒偏差 0）；解析判据（**包围盒 = 旋转不变量**、twist=0 体积严格 = A·h）；**收敛表** 0.538→0.205→0.0666→0.0227→0.00858→0.00358（O(1/seg²)）；**体积只取决于每层扭转角**（(90,12)/(180,24)/(360,48) 三者 rel ≤ 9e-15）；L 形方向判据（第 j 层顶点 = Rz(twist·j/seg)·p_i，偏差 0） |
| 带孔扭转 | 单洞与 JSCAD **多环切片**交叉 rel ≤ 2.2e-14；双洞水密 + 单调收敛到 `(A外−ΣA洞)·h`（seg=64 时 rel 6.3e-3）；twist=0 时体积严格 = 解析值（rel 1.3e-15） |
| `_temp/model-sweep-test.cjs`（内核） | **52/52**：**直线扫掠 ≡ 挤出**（体积严格相等、端点顶点层逐点一致）；直线扫掠+twist ≡ 扭转挤出（rel ≤ 7.5e-16）；**环面 2π²Rr²**（S/M 加倍误差 4.8e-2→1.2e-2→3.1e-3→2.9e-3）；圆弧按角度比例（rel ≤ 5e-3）；**圆台** πh(r0²+r0r1+r1²)/3（rel 7.1e-4）；非平面闭合路径 holonomy 后仍水密无翻转；8 类非法输入全部报错 |
| `_temp/model-sweep-e2e.cjs`（工具链端到端） | **36/36**：表达式 twist/scale/along、参数联动（改 twist 重算且复原确定性 rel 0）、**model_verify M1–M9 全过 fails=0**、导出 5 产物、**glTF 官方校验 errors=0/warnings=0**、STL 读回体积 rel **2.42e-8**、STL 字节 = 84+50×面数 |
| **真 goja 沙箱**（cordis 探针，非 Node 模拟） | 加载**真源码**跑全链路：twist[212 面, 3611.315]、sweep[384 面, 3306.3849]、带孔扭转[572 面, 2895.9447]、双洞板[116 面, **6876 = (1200−2×A₁₂)×6 精确**]、**fails=0/warns=0、M4+M5+M6 全过**、导出 STL 64284 B = 84+50×1284、三类非法输入均正确报错、失败的 add 未写入（parts 保持 4） |
| **跨引擎一致性** | goja 四个场景的**面数与体积与 V8 逐位相同**，STL 字节一致 ⇒ 两个引擎的数值路径无差异（这正是不只做 Node 模拟的原因） |
| 无回归 | `model-holes-test` 18/18、`model-core-test` 45/45、`model-stage2-test` 46/46、`model-plugin-test` 78/78 |

#### I.8.5 fillet / chamfer 结构方案（评估结论：另立专项）

**问题本质**：圆角/倒角不是参数化特征，而是**网格级边重建**。当前几何管线是
「形状表达式树 → 三角网格 →（BSP 布尔）」，要做 fillet/chamfer 需要三步，且每步都有硬骨头：

1. **边识别**：焊接顶点 → 统计每条边的两个相邻面 → 二面角超阈值即候选边（凸/凹分类）。
   成本低（可复用现有 `meshTopology` 的边统计），但**判据要稳定**（近共面、T 缝、非流形边都要处理）。
2. **局部重建**：
   - **倒角（chamfer）**：把边两侧面各内缩距离 d，插入一条新面。凸边切掉棱、凹边填角。
   - **圆角（fillet）**：沿边插入**圆柱面片段**（半径 r，绕边轴），面数上升明显。
3. **顶点混合（blend）**：三条以上边交会处的球面/过渡片 —— **这是真正的大头**（rolling-ball 类方法），
   凸顶点、凹顶点、混合类型的处理规则都不同；圆角半径大于相邻面尺寸时还会自交。

**三条可行路线（按成本排序）**：

| 路线 | 做法 | 成本 | 评价 |
|---|---|---|---|
| **A. chamfer 走 BSP 半空间**（推荐先做） | 凸边 e 的两侧面法线 n1,n2 ⇒ 斜面法线 = normalize(n1+n2)，用一个包住该边的**盒子 ∩ 半空间**与实体 `intersect` | **中（~100 行 + 验证）** | 复用现有 BSP 布尔，结果由布尔保证水密；可按边逐个处理（O(边数) 次布尔，成本可控）。**验证可解析**：长方体倒角 d 的体积 = 原体积 − Σ(棱长×d²/2) + 顶点修正 |
| **B. fillet 走布尔近似** | 沿边减去「棱边附近的材料」再拼上圆柱片段（两步布尔） | 中高（~150-200 行） | 近似质量够工程用；面数增长明显（圆角面离散化），顶点处仍需单独处理 |
| **C. 真正的网格级边重建** | 识别边 → 收缩面 → 建圆角面 → 顶点混合，全程精确 | **高（300+ 行）** | 质量最好、可参数化（半径随位置变化），但顶点混合与自交处理是独立的数学工作 |

**结论**：本轮不做（twist/sweep 已占满预算且它们有解析真值可闭环）。**建议下一轮单独立专项做路线 A**，
把 chamfer 打通并验证，再视需要评估 B/C；路线 C 只在「高精度参数化圆角」成为真实需求时投入。

> **→ 已在附录 I.9 落地**：路线 A 走通（凸体 = 面平面 ∩ 斜面半空间，H-rep 直接构造 ⇒ 解析精确 + 严格水密，
> 全 12 棱 60 面 / boundary 0；对照同场景布尔 1274 面 / 1274 条开边界）。
> 圆角先按路线 B 的布尔近似做到「互不相邻的棱完全水密」；**路线 C 的顶点混合**随后由 I.9.6 解决 ——
> 但对**凸体**没走"逐边补面 + 顶点补片"，而是直接上**形态学开运算** `(P ⊖ Ball_r) ⊕ Ball_r` 一次性构造
> （平面片 + 棱圆柱片 + 顶点球片），天然水密、且顶点球片本身就是混合面。非凸体仍走路线 B。

---

### I.9 倒角 / 圆角（chamfer / fillet，2026-09 清遗留续，已完成）

> 承接 I.8.5 的评估结论（路线 A 可解析验证、建议先做一个专项）。本轮把 **chamfer 打通到「解析精确 +
> 严格水密」**，**fillet 做到「互不相邻的棱完全水密」**，并把做不到的部分变成**明确报错**而不是坏网格。
> **续（I.9.6）**：凸体圆角的**顶点混合**已落地 —— 不逐边补面，而是走形态学开运算的解析构造，
> 立方体 12 棱（含相邻棱交会）严格水密。

#### I.9.1 口径

倒角/圆角是**网格级**后处理（与 `align`/`snap`/`repair` 同类），不是参数化特征：

```js
model_edit({ ops: [{ op:'mesh.chamfer', id:'base', distance:1.5 }] })          // 全部凸棱
model_edit({ ops: [{ op:'mesh.fillet',  id:'base', radius:2,
                     edges:[[[20,15,-3],[20,15,3]], …] }] })                    // 只做指定棱
```

| 参数 | 含义 |
|---|---|
| `distance` / `radius` | **面内**尺寸（用户直觉的"倒角量"），必须为正；差值口径见 I.9.2-1 |
| `edges` | 缺省 = 全部**折角 ≥ minAngle** 的凸棱；或坐标对数组 `[[[x,y,z],[x,y,z]],…]` **精确指定**（与顶点编号顺序无关 ⇒ 可复现） |
| `minAngle` | 认定为"棱"的最小折角（默认 30°）：圆柱侧面的分段边（折角 11.25°）被视为同一曲面 |
| `segments`（仅 fillet） | 弧面离散段数（默认 8） |

#### I.9.2 关键设计（四条，都是实测逼出来的）

1. **斜面平移量 = d·cos(θ/2)**（θ = 内二面角）：这样斜面恰好经过「两个邻面内**距棱 d**」的两点 ⇒
   `d` 的口径与直觉一致（立方体 θ=90° ⇒ 平移 d/√2）。
2. **凸体完全绕开布尔：H-rep 直接构造**（顶点枚举 → 逐平面收集共面顶点 → 面内角度排序 → 扇形三角化）。
   倒角结果 = `面平面 ∩ 各斜面半空间`，**全是凸约束** ⇒ 解析精确、**天然水密**、面数最少。
   凸性判据复用边表（"所有内部边都是凸边"），O(E) 而不需要 O(V·F)。
   > 对照实测：同一场景走 BSP `intersect` 得到 **1274 面 / 1274 条开边界 / χ=−126**（T 缝遍地）；
   > H-rep 得到 **60 面 / boundary 0 / χ=2**。这是本轮最大的一处"换个表示就干净了"。
3. **fillet 的加回块必须是「弓形」，不能是整个四分之一圆柱**：完整圆柱块与「已倒角的实体」只沿
   **切点线**相切（零体积接触），BSP union 处理这种退化接触会留下大量未配对边（单棱 49 面里 9 条开边界）。
   把加回块切成 `圆柱 ∩ 斜面外侧` 的**弓形**后，二者沿**斜面片**相碰（面接触）⇒ boundary 0、χ=2。
   几何等价性：`结果 = 实体 ∩ ({斜面内侧} ∪ 弓形) = 实体 ∩ ({斜面内侧} ∪ 四分之一圆柱)`。
4. **弧块按包围盒分组**：组内两两不相交 ⇒ 直接 `meshMerge`，只有跨组才付布尔代价
   （逐块 union 会让 BSP 每次按新块的全部平面切碎整个网格：12 条棱实测 **76 s / 5.2 万面**）。

#### I.9.3 本轮踩到并修掉的缺陷（都是"看起来对、跑起来错"那一类）

| 缺陷 | 症状（实测） | 根因 | 修复 |
|---|---|---|---|
| 共面对角线被判成**凹边** | 圆柱（凸体）被判"非凸" → 白走布尔路径：**1274 面 / 1274 条开边界 / χ=−126** | 凸凹判定用严格的 `sgn <= 0`；共面边的 sgn 理论是 0，浮点误差给了 `+1e-16` | 加相对容差 `sgn <= max(1e-12, 1e-9·边长)` ⇒ 圆柱回到 H-rep 路径：boundary 0 / χ=2 |
| 加回块与实体**零体积接触** | 单棱圆角 49 面里 **9 条开边界**、χ=−8 | 四分之一圆柱与倒角结果只沿切点线相切 | 改用弓形（多加一道"斜面外侧"反向约束）⇒ 面接触 ⇒ boundary 0 / χ=2 |
| fillet 逐块 union **爆炸** | 12 条棱 **76 s / 5.2 万面** | 每次 union 按新块的所有平面切碎整个网格 | 弧块按包围盒分组、组内 meshMerge；超限时明确报错 |
| cutter 盒子**贴太近** | 非凸体倒角 boundary 178 | 盒子平面贴近实体表面 ⇒ BSP 平白多切出碎片 | 盒子 margin 由 `0.05·diag` 提到 `2·diag`（盒子只需包住实体） |
| 端面**共面** union | 单棱圆角一度 boundary 17（比不外扩更差） | 弧块端面与实体端面严格共面 | 放弃"端面外扩"这条路，改从接触方式入手（见上一条弓形） |

#### I.9.4 验收数据（实测）

**解析真值**（全 12 棱立方体 40×30×6，d=1）：`V − Σ_棱 (d²/2)(L_e−2d) − 8·0.75d³ = 7054` ✓

> 角部系数 0.75 是**算出来的**：三条棱的斜面在角部小方块 `[0,d]³` 内的切除 =
> `{u+v<d} ∪ {v+w<d} ∪ {w+u<d}`，按 w 分层积分得 `∫₀¹[1−(1−α)²+max(0,(1−2α)²/2)]dw = 2/3+1/12 = 3/4`。
> （第一版按容斥算成 0.625 是**雅可比算错**——`∂(u,v,w)/∂(p,q,r)` 是 1/2 不是 1/4，实测与实现的 1 个单位差
> 正好等于这 `d³/8`，反过来定位到是公式错、不是代码错。）

| 项 | 结果 |
|---|---|
| `_temp/model-chamfer-test.cjs` | **53/53** |
| 倒角解析精确性 | 单棱 7197 / 4 平行棱 7188 / 全 12 棱 **7054**（rel 2.6e-16）/ 3 棱交于一角 / d∈{0.5,1,1.5,2} 逐一 rel ≤ 5.5e-16 |
| 倒角拓扑 | 以上全部 `boundary=0 / nonManifold=0 / flipped=0 / χ=2 / 容忍判据 0`；全 12 棱仅 **60 面** |
| 圆柱（32 段曲面体） | 全边倒角水密、χ=2、面数受控（分段棱各自一块斜面 ⇒ 等价于多边形锥面倒角） |
| 圆角（互不相邻棱） | 单棱体积 rel 3.6e-5（弧面为内接多边形，公式吻合）；单棱/4 平行棱均 `boundary=0 / χ=2`；段数↑体积单调↑并收敛到真圆（`r²−πr²/4`） |
| 非凸体（带孔板 2 孔）倒角 | 走布尔：体积 rel 6.7e-6、**无非流形边**、面数 349 未膨胀（如实记录 boundary=149） |
| 报错集 | distance 缺失/为 0/过大、棱找不到、`edges` 非法、球体无棱、minAngle 越界、相邻棱圆角、超上限、凹边指定 —— **12 项全部明确报错** |
| **真 goja 沙箱**（cordis 探针加载真源码，192 975 字节） | 与 V8 **逐位一致**：chamfer12 `7054.000000000002` / 60 面 / boundary 0 / χ2；chamfer1 `7197`；fillet4 `7194.688312741451` / 140 面 / boundary 0 / tolUnpaired 0；带孔板布尔路径 `6852.524331458106` 逐位相同；两类报错信息一致 |
| 无回归 | model-core 45 / holes 18 / stage2 46 / twist 85 / sweep 52 / sweep-e2e 36 / plugin 78（**合计 360**） |

#### I.9.5 能力边界（如实标注，不静默降级）

| 能力 | 状态 | 依据 |
|---|---|---|
| 倒角：凸体的凸棱（单次 ≤ 64 条） | ✅ 解析精确 + 严格水密 | H-rep，无布尔 |
| 倒角：非凸体（带孔板/镂空） | ⚠️ 可用但走布尔：体积正确、无非流形边；水密性受 BSP 固有能力限制（同 M4 的分级口径） | 实测 boundary 149 |
| 圆角：**互不相邻**的棱（如同方向 4 条竖棱、异面棱；单次 ≤ 8 条） | ✅ 严格水密、面数小 | 实测 boundary 0 / χ2 |
| 圆角：**凸体全棱（含相邻棱交会，如立方体 12 棱）** | ✅ **严格水密 + 解析可验证** | 走 rolling-ball 开运算（见 I.9.6）：12 棱 **1164 面 / boundary 0 / χ2**，体积 vs 闭式真值 rel 1.5e-4（内接离散，恒不超真值）；此前"硬做"是 boundary 206 |
| 圆角：**非凸体**的相邻棱、或**只选部分棱** | ❌ **明确报错**（非凸）／走布尔近似（部分棱） | rolling-ball 以"整体开运算"为前提（只圆一部分棱时结果不再是 P 的子集）；报错文案已指向"凸体不指定 edges 即可走精确路径" |
| 凹边（内二面角 > 180°） | ❌ **明确报错** | 凹边倒角要「填」材料（另一套逻辑）；默认只处理凸棱，诊断里如实报告跳过条数 |

**下一步（未做）**：I.6-1 的**共面三角形合并** —— 能同时缓解 BSP 碎片、M4 的 T 缝与 fillet 的成本。

### I.9.6 rolling-ball 顶点混合（凸体圆角，2026-09 清遗留续，已完成）

> **目标**（I.9.5 的"下一步①"）：让 `mesh.fillet` 支持**相邻棱**，即"立方体 12 棱全圆角"。
> I.9 的路线 B（逐边切棱 + 拼回弧块）在顶点处靠 BSP 处理多个弧块相交，实测 boundary 206~554。

#### I.9.6.1 关键认识：换表示，而不是补面

圆角的几何本质是**形态学开运算** —— 先用半径 r 的球把实体磨小、再滚回去。对**凸体**它可以解析构造，
根本不需要"逐边补面 + 顶点补片"：

1. `P ⊖ Ball_r`：每个面平面沿法线内缩 r ⇒ 仍是 H-rep 凸体 `P'`（就是把 `w_i` 改成 `w_i − r`）。
2. `P' ⊕ Ball_r`：半径 r 的球绕 `P'` 边界滚一圈 ⇒ 边界恰好是**三类面片**的拼合：

   | 片 | 所在位置 | 参数化 |
   |---|---|---|
   | 平面片 | `P'` 的面外移 r | 面顶点环整体 `+ r·n_i`，扇形成三角形 |
   | 圆柱片 | 轴 = `P'` 的棱 | 径向 = 从 `n_i` 转到 `n_j` 的大圆弧；沿轴只有两端点 |
   | **球片** | 球心 = `P'` 的顶点 | 方向域 = 该顶点相邻面法线张成的凸锥 —— **这就是"顶点混合面"** |

**为什么天然水密**：棱 (i,j) 的方向 ⊥ `n_i`、`n_j` ⇒ `span(n_i,n_j)` 正好是轴的正交补
⇒ 圆柱片在端点的径向，恰好就是球面上由 `n_i`、`n_j` 张成的那条**大圆弧**。两类片在接缝上用**同一个**
`slerpDir` 取点 ⇒ 顶点坐标逐位相同，不需要任何焊接容差兜底（实测 boundary / nonManifold / flipped /
tolerantUnpaired **全 0**、χ = 2）。

**结果 ⊆ 原体（自证，不需实测兜底）**：`Q ∈ P'` ⇒ 对任意面 m 有 `n_m·Q ≤ w_m − r`；片上的点
`X = Q + r·d`（|d| = 1）⇒ `n_m·X ≤ w_m − r + r·n_m·d ≤ w_m`。所以圆角只会切料、绝不外扩。

#### I.9.6.2 本轮踩到并修掉的缺陷（两条都是"看起来对、跑起来错"）

| # | 现象 | 根因 | 修法 |
|---|---|---|---|
| 1 | `flipped=108`、体积偏小 **17.7%** | 三类片的顶点顺序由各自参数化决定（面内角序／弧段方向／轴方向），**部分面反向**；而"按整体有向体积翻转"只能翻全部面，修不了部分反向 | 逐片按"三角形重心相对内缩体质心的方位"定向（凸体的外法线判据） |
| 2 | 体积缺口 **7.9e-5 且不随 segments 收敛** | 球片用"单极点扇形"：apex（重心方向）到边界的角距可达 50°+，弦面把球顶削掉（r=2 时偏差 ≈0.32）；缺陷来自**径向一刀切**，与边界分几段无关 | 扇形内再按**径向分 rings 层同心环**（rings = `ceil(√segs)`，限 2~6）；边界点仍用同一个 `slerpDir` ⇒ 水密不破 |

#### I.9.6.3 路径选择（不破坏既有能力）

`sel.list` = **全部**"折角 ≥ minAngle 的棱"时走 rolling-ball；否则回落 I.9 的布尔近似路径
（那里的 8 条上限与相邻棱拒绝照旧）。凸体平面数超过 `LIMIT_RB_PLANES`(32) 也自动回落。

> ★ 判据**不能**用"convex 的边数"：共面边（同一平面被三角形切开的对角线）`sgn ≈ 0`，按凸凹判定
> 也算 convex —— 立方体 18 条内部边里 6 条对角线会被算进来，导致永远判否、rolling-ball 形同虚设。
> 正确口径是"折角 ≥ minAngle 的边总数"（那才是"棱"），与 `sel.list` 比数量即集合相等。

只圆**一部分棱**时不能走 rolling-ball：开运算的前提是"整体"，只圆部分棱的结果不再是 `P` 的子集。
**非凸体**同理（要先做"凹边填角"，见 I.9.5 遗留）。

#### I.9.6.4 验收（实测）

| 项 | 结果 |
|---|---|
| 立方体 40³ 全 12 棱 r=2 | `mode=rolling-ball`、12 棱、8 个球片、26 个面片；**1164 三角面**（12 平面 + 192 圆柱 + 960 球） |
| 拓扑 | boundary 0 / nonManifold 0 / flipped 0 / **χ=2** / tolUnpaired 0（严格水密） |
| 体积 | `63589.28384117101` vs 闭式真值 `63598.67834798908`，**rel 1.48e-4** 且**恒 ≤ 真值** |
| 半径扫描 r ∈ {0.5,1,1.5,2,3} | 全部 rel < 1e-3、全部严格水密、体积单调下降 |
| 长方体 40×30×6 r=1（非立方体） | `7134.604` vs `7136.100`（rel 2.1e-4），严格水密 |
| 收敛性 segments 2→32 | 缺口 2.19e-3 → 5.73e-4 → 1.48e-4 → 3.96e-5 → **1.08e-5**，单调收敛且全程水密 |
| 外接尺寸 | bbox 逐位不变（平面片仍落在原面平面上） |
| 路径回落 | 只选 2 条平行棱 → `mode=direct`（布尔路径）；非凸 L 形 → `mode=boolean`；二者仍严格水密 |
| 新测试 | `_temp/model-rb-fillet-test.cjs` **47/47**（解析真值 + 拓扑 + 收敛 + bbox + 路径选择 + 报错 + 接入层） |
| 回归 | **462 项全通过**：core 45 / holes 18 / chamfer 55 / plugin 78 / rb-fillet 47 / stage2 46 / sweep-e2e 36 / sweep 52 / twist 85 |
| 真 goja 沙箱（202 996 字节真源码） | 与 V8 **逐位一致**：volume `63589.28384117101`、cutRatio `0.006417439981703068`、1164 面、boundary 0 / χ2；布尔路径 `63982.440844799996`；chamfer `63766.00000000001`；报错文案一致 |

**闭环验算（这条最值一提）**：球面片精度"应有的样子"可以先算出来再验证 —— 圆柱片缺口
= `(π/4 − m·sin(π/2m))·r²·ΣL`（m = segs），segs=8 时 = **8.72**；加上球片残差 ⇒ 理论总缺口 ≈ **9.4**，
实测 **9.394**。正是这一步反过来定位到"缺口几乎全来自圆柱片的固有内接离散"（O(1/segs²)，不可能随 segs 归零），
才没有继续在球片上过度投入。

**遗留**：① **凹边填角**（内二面角 > 180°，要补材料）；② 非凸体 + 相邻棱仍走布尔路径，
需先做逐面"凹边填角"再做形态学开运算；③ **非均匀半径**（半径随位置变化）需换"变半径开运算"，
本轮只支持常数 r。

---

5. **GLB 与 binary STL 已可用但未进 `format=all` 默认集**（GLB 需 `withGlb=true`），
   因为 GLB 在 sha/体积上对用户价值低于 glTF+STL，先保守。
6. 发布侧遗留同附录 E.6 / F.6（npm 发布 + 镜像同步 + 市场验收，待凭据）。

### I.7 2D 带孔轮廓挤出（2026-09 清遗留，已完成）

**口径**：`extrude` 新增可选 `holes`（孔轮廓数组，每项与 `profile` 同口径：点集或 `{type:"circle"|"rect"|"star"|"polygon",…}`），
一步生成带孔板，不再需要 `subtract` 布尔：

```json
{ "id": "plate", "type": "extrude", "profile": { "type": "rect", "size": [40, 30] }, "height": 6,
  "holes": [ { "type": "circle", "radius": 3, "segments": 32, "center": [12, 0] },
             { "type": "rect", "size": [6, 6], "center": [0, 9] } ] }
```

**实现**（`plugins-dist/tool-model/index.js`）：

1. `holesPoints`：洞轮廓解析 + 方向统一为**顺时针**（外轮廓 CCW）⇒ 洞的侧壁法线自动朝腔内。
2. `triangulatePolygonWithHoles`：earcut 的 eliminateHoles 思路 —— 洞最右点向 +x 射线找外轮廓上**可见**顶点 P，
   把洞循环桥接进外轮廓：`… P, M, 洞循环…, M, P, …`（M = 洞最右点）。**P 与 M 各出现两次构成零宽通道**（关键）。
3. `ringInsideRing`：洞必须完全落在外轮廓内（顶点全在内 + 两环互不相交），否则**明确报错**而非静默降级。
4. 侧壁：外轮廓与每个洞各出一圈；盖面用带孔三角化（退化三角形过滤）。
5. `profileCenter`：`center` 偏移现在对 `circle/star/polygon` 同样生效（此前仅 `rect`），偏心孔才能定位。

**本轮实测定位并修复的缺陷**（勿回退）：

| 缺陷 | 症状（实测） | 根因 |
|---|---|---|
| 桥接点只插入一次 | L 形/方形挖孔**面积缺 70/371（19%）**、boundary=6 | 通道未闭合，外轮廓被「斜切」丢失一块 |
| 耳切遮挡判定写反 | L 形+孔出现 10 个**翻转三角形**、nonManifold=2、χ=2（应 0） | 应为「**凹**点落在耳内才拒绝」，误写成凸点 |
| 选耳策略 | 兜底强切 3 次、翻转到兜底为止 | 改为 earcut 的「顺序切耳 + 一轮无耳先清共线点(`filterPoints`) + 最大凸角兜底」 |

**验收（实测）**：

| 项 | 结果 |
|---|---|
| `_temp/model-holes-test.cjs`（内核） | **18/18 通过**；体积对**解析值 rel ≤ 5e-15**（方形/3 孔/非凸 L 形+2 孔）；JSCAD `geom2.subtract+extrudeLinear` 交叉 rel ≤ 4e-5（差值是 JSCAD 布尔重采样误差，本实现更精确） |
| 拓扑正确性 | 严格水密 boundary=0 / nonManifold=0；欧拉特征 **χ=0（1 通孔）/ −2（2 孔）/ −4（3 孔）** 与 genus 一致；洞内不被覆盖、无缺口（2500 点采样） |
| `_temp/model-holes-e2e.cjs`（工具链端到端） | **24/24 通过**：顶文字段与 `shape` 两种写法、表达式尺寸、偏心孔、越界/非法 center 报错且不写入、`model_verify` M4/M5/M6 全过、导出 5 产物、**glTF 官方校验 errors=0/warnings=0**、STL 读回体积 rel 7e-10 |
| 与布尔方案对比（60×40 板挖 2×∅6） | 带孔 **276 面 / 4 ms**；布尔 **5763 面 / 218 ms**（体积 rel 5e-6，差值来自布尔） |
| 无回归 | `model-core-test` 45/45、`model-stage2-test` 46/46、`model-plugin-test` 78/78 |
| **真 goja 沙箱运行**（cordis 探针，非 Node 模拟） | **ok=true**：体积 6862.8839 = 解析值（rel **3.43e-9**）/ 276 面 / **M4 水密 + M5 绕向 + M6 实体性全过、fails=0** / STL 13884 B = 84+50×276（goja 内单精度+base64 编码链路）/ 洞越界正确报错 |
| **宿主装载**（9097 测试实例，重载后） | 插件 state=**running**、lastError 空、5 个 `model_*` 工具在 `/api/tools`、`model_add.usageGuide` 已是新版（含带孔挤出示例） |

---

## 附录 J 呈现方式升级 + 前端设计落地（2026-09-17 实测）

### J.0 需求（用户提出，两点）

1. **支持前端设计，并把设计出的前端「落地到项目」**（此前只出预览，完全缺失）；
2. **新增 UI 的呈现位置**：此前 6 个创作域 UI 全部注册为**插件面板**（藏在壳级逃生口的浮动面板里），
   需要支持**在主内容区以 tab 展示**，以及**与对话切换/并排显示（可切换）**。

### J.1 现状盘点（扩展点定位）

| 机制 | 现状 | 结论 |
|---|---|---|
| 主内容区 tab（`.main-area`） | 已有：`state.panels.mainTab = 'conversation'｜'editor'｜'market'｜'toolsets'`，且 market/toolsets 就是**动态挂载 bundle 到主区**的现成范式 | 可扩展为「插件视图 tab」，但要改壳 |
| 插件面板 | `ui.registerPanel({id,title,render})` → 渲染进 PluginPanel 的客户端面板小 tab 区 | **没有**任何 API 能让插件进主区 tab |
| 并排布局 | 对话/编辑器/市场 tab **互斥切换**，不能并存 | 需新增 split 布局状态与几何 |
| design 域 | 5 个工具（tokens/project/edit/export/verify），产物只有 **HTML 预览 + CSS 变量 + mermaid** | **无前端工程代码生成、无落盘到项目源码** |

### J.2 需求 2：中间区域视图 + 与对话并排

#### J.2.1 新 API：`ui.registerView(spec)`

```js
ui.registerView({ id, title, icon, order, open, render(el, ui) })
```

- 注册表 `clientViews`（`window.__SLOT_REGISTRY.clientViews`，跨壳/UI bundle 副本一致）；
- 与 `registerPanel` 的边界：**panel = 插件面板内的小 tab**（管理/总览），
  **view = 主内容区同级 tab**（长期工作区）；
- `open:true` = **后台 tab 语义**：tab 出现但**不抢占对话主视图**（`mainTab` 仍是 `conversation`），
  用户点 tab 才激活；
- 打开状态持久化 `localStorage: viewOpen:<插件名>:<视图 id>`（未设置 → 取注册时的 `open` 默认值）；
- 卸载插件 / `syncClientHalves` 对账 → 自动清理孤儿视图（`emitViewChanged`）。

#### J.2.2 壳层改造（`ShellApp.vue`）

| 改动 | 要点 |
|---|---|
| tab 栏 | 内置 4 视图 + `openViews`（`clientViews` 中 `isViewOpen` 者为真）各一个 tab（可关闭 ×） |
| 「视图」菜单 | tab 栏右侧下拉：勾选 = tab 打开/关闭（**关闭后能重开**，否则用户关掉就找不回来） |
| 内容区 | 新增 `.main-views` 容器：单栏（flex column，tab 互斥）或**并排两栏**（flex row） |
| 并排 | 「并排对话」按钮 → `layout.toggleSplit()`；`split-chat-right` 用 `row-reverse` 换边（对话 pane 始终是 DOM 首元素，只改方向不动挂载）；对话栏固定 `--split-chat-w, 42%` |
| 挂载策略 | **懒挂载 + 保持**：首次激活才 `render(el)`，之后切 tab **不卸载**（保住 3D 视角/滚动位置），关闭 tab / 插件卸载才 cleanup |
| 语义细节 | `canSplit` = 当前不是纯对话视图（对话本身无法与自己并排）；`isSplitActive()` = split 开关 **且** `mainTab !== 'conversation'` |

> 状态机落在 `ui-state.js`：`panels.splitView` / `panels.splitChatSide`（**不持久化** —— 与
> `focusMode` / `editorOpen` 同规则，避免「上次并排 → 下次启动还是并排」）。

#### J.2.3 六个创作域接入

`ui-art`(order 10) / `ui-design`(20) / 3D 模型(30，**由 `tool-model` 包注册**，2026-09-17 合并) /
`ui-music`(40) / `ui-rig`(50) / `ui-voice`(60)，
统一 `open:true`（后台 tab），**与既有 registerPanel 共用同一个 bundle**（`render` 每次调用各建独立 Vue 实例）。

#### J.2.4 实测

`_temp/ui-view-tabs-verify.mjs`（CDP + headless Chrome，9097）：**36 项断言全通过** ——
tab 注册与顺序 ✓／默认后台打开不抢占对话 ✓／6 个 pane 懒挂载（初始 `innerHTML` 为空）✓／
点击激活并挂载 ✓／并排两栏同时可见（对话栏 >200px）✓／换边 `split-chat-right` ✓／取消并排回单栏 ✓／
关闭 × 后 `viewOpen=0` 且回对话 ✓／视图菜单勾选恢复 ✓／6 视图逐一激活均可挂载 ✓／刷新后状态持久 ✓／
无 console 错误、无 `/api/api` 双前缀 ✓。截图 `_temp/view-tabs-single.png` / `view-tabs-split.png`
（read_image 视觉核对：单栏创作 UI 占满、并排左对话右画板，布局无错乱）。

### J.3 需求 1：前端设计落地（`design_codegen`）

#### J.3.1 产物（targets 可多选）

| target | 文件 | 说明 |
|---|---|---|
| `html` | `<targetDir>/html/<屏幕 id>.html` + `.css` | 零依赖静态页（link 引用 `../tokens.css`） |
| `vue` | `<targetDir>/vue/<Screen>.vue` | Vue 3 SFC：`<template>` + `<style scoped>` |
| `react` | `<targetDir>/react/<Screen>.jsx` + `.module.css` | 函数组件 + CSS Modules（`styles['d-x']`） |
| 公共 | `<targetDir>/tokens.css`、`README.md`、`codegen.report.json` | 令牌变量表 + 集成说明 + 落地报告 |

#### J.3.2 与预览的本质区别（这是「工程代码」的判据）

| | `design_export format=html`（预览） | `design_codegen`（代码） |
|---|---|---|
| 布局 | **绝对定位**（布局引擎算出 left/top/width/height，截图即所见） | **flex 流式**（容器 `display:flex` + `flex-direction` + `gap` + `padding`，尺寸自适应，不写死画布像素） |
| 尺寸 | 每节点写死 px | 默认由内容/父容器决定（屏幕容器用 `max-width` + `min-height`） |
| class | `class="nd"` + 内联 style | 语义 class `.d-<节点 id>`（+ 屏幕容器 `.screen-<屏幕 id>`） |
| 标签 | 一律 `<div>` | 语义标签：`<p>` / `<button>` / `<input>` / `<label>` / `<hr>` / 内联 `<svg>` |
| 颜色 | 解析后的 hex 内联 | **令牌变量** `var(--color-fg)`（改 tokens 重新生成即全站同步，代码里零裸值） |

#### J.3.3 类名三级策略（可维护性）

① `props.name`（显式命名，最优）→ `.d-<name>`；
② 语义化 id（非模板自动编号）→ `.d-<id>`（如 `hero-title`）；
③ 模板自动编号 id（`home-panel_n3`）→ `.d-<类型>-<序号>`（`.d-text-3`，比 `n3` 可读）；
同名冲突追加 `-2/-3`（去重表**每个 target 独立**，产物才彼此一致）；
仍有 ③ 时返回里提示「建议给节点起语义 id」。

> ★ `props.name` 对 **icon 类型必须排除**：icon 的 `props.name` 是**图标名**（ICONS 白名单键，
> 如 `home`/`success`），当类名会得到 `.d-home` / `.d-success`（跨节点撞名 + 丢掉节点语义）—— 实测踩过。

#### J.3.4 落地语义（安全第一）

- `targetDir`（默认 **`design-export/`**，不碰既有源码树；要落进项目用 `targetDir=src/components`）；
  含 `..` → **拒绝**（越界保护）；
- **覆盖前自动备份** `<文件>.bak`（可人工回滚），备份清单进落地报告；
- `dryRun=true` → 只列将生成的文件、不写盘（AI 先看再落）；
- **幂等**：同一工程两次生成**位级一致**（无时间戳/无随机/无绝对路径）；
- 空屏（`template:'none'` 无根节点）→ 显式提示「生成的组件只有屏幕容器，先用 `node.add` 加内容」。

#### J.3.5 实测（四层验证）

1. **Node 侧行为测试** `_temp/tool-design-codegen-test.cjs`：**13 组全通过**（dryRun/三 target 结构/
   单位修复/标签映射/令牌变量化/类名策略/备份/幂等/单屏过滤/边界拒绝/白名单外图标占位）；
   既有 `_temp/tool-design-test.cjs` **78 项回归通过**。
2. **真实 goja 沙箱**（cordis 探测插件 + `new Function` 加载真实插件源码）：13.1 万字符源码求值 ✓、
   6 个工具注册 ✓、三 target 产物 ✓、10 项内容断言 ✓、幂等 ✓、7 个 `.bak` ✓、
   **`design_verify` D1–D9 全绿** ✓、边界拒绝 ✓ —— 证明新代码（400+ 行）在沙箱与 V8 下行为一致。
3. **端到端落地** `_temp/design-codegen-e2e.cjs`：3 屏 × 3 target = **17 文件 52.2 KB**；
   **`@vue/compiler-sfc` 真编译**（parse + compileTemplate 无错、style scoped）✓、
   **esbuild 真转译 JSX** ✓、CSS 括号平衡/无裸色值/变量全部有定义 ✓、
   语义类名与标签映射 ✓、幂等 + `.bak` + `design_verify` 回归 ✓。
4. **真浏览器渲染**：headless Chrome 打开生成的 HTML → 截图 + read_image 视觉核对（标题/按钮/幽灵按钮/
   复选框/分隔线/徽章/图片占位/图标/输入框全部正常，无重叠错位裁切，`var(--color-bg)` 实际解析为
   `rgb(255,255,255)`）✓。

#### J.3.6 面板呈现（ui-design「前端代码」区）

读 `design-export/codegen.report.json`（或工程 `artifacts.codegen`）展示：targets / 落点 / 文件数 / 总大小 /
生成时间（**设计变更晚于生成 → 提示「建议重新生成」**）/ 屏幕 / 产物清单 / `.bak` 数 / 警告 + 一键复制生成命令。
CDP 实测 **10 项断言全通过**（含「设计未变更不显示过期提示」），截图
`_temp/ui-design-codegen-panel.png` / `ui-design-codegen-section.png`。

### J.4 本轮修掉的缺陷

1. **`--space-md: 12;` 是无效 CSS**（尺寸类令牌是裸数字，`renderTokensCss` 原样输出）→
   补单位（space/radius/fontSize → px）—— 否则 `design_export format=css` 的产物拿进前端**完全不生效**。
2. **类名去重表跨 target 共享** → html 先占 `.d-x`，vue 里变成 `.d-x-2`（module.css 与 html 版对不上）；
   改为每 target 独立去重表 + 自动编号统计按节点 id 跨 target 去重。
3. **icon 的 `props.name`（图标名）被当节点名** → `.d-home` / `.d-success` 撞名（见 J.3.3）。
4. **空屏静默产出空组件** → 加显式提示（J.3.4 末条）。
5. CORDIS 探测脚本踩坑（记录）：沙箱 `ctx.fs` 的**相对路径基准不是工作区**（实测是进程固定根），
   用绝对路径会被「超出工作区范围」拦截 → 探测改用**内存 mock fs**；
   `ctx.app.workspaceRoot` 才是工作区，`new Function` 在沙箱**可用**。

### J.5 验证矩阵（本轮）

| 项 | 结果 |
|---|---|
| 视图 tab CDP（`ui-view-tabs-verify.mjs`） | 36/36 通过 + 双截图视觉核对 |
| 面板「前端代码」区 CDP | 10/10 通过 + 双截图 |
| codegen Node 测试 / 既有回归 | 13 组 + 78 项全通过 |
| codegen 真实 goja 沙箱 | 10 阶段全绿（含 D1–D9、幂等、`.bak`、6 工具） |
| 端到端落地（真编译 + 真浏览器） | `@vue/compiler-sfc` ✓ / esbuild ✓ / Chrome 渲染 ✓ |
| 硬门槛 | `go vet` 0 / `go build` 0 / `go test -short ./internal/agent` **ok 64.4s** / `verify-dist-isolation` PASS |
| 装配 | `/api/plugins` 41 插件、tool-design **6 工具**（含 `design_codegen`）、`/api/ui-boot` 15 entries |

### J.6 遗留与下一步

1. ~~设计 → 代码的组件化粒度~~ **已实现，见 J.7**：`design_codegen component=<节点 id>` → 子树抽成独立组件 + 屏幕内组件引用。
2. **交互/状态**：生成的是**静态展示组件**（无事件绑定、无 props）。下一步可加「节点 action 字段 →
   `@click` / props 声明」，但这需要先在设计工程里引入交互模型（属设计域扩展）。
3. **样式输出策略**：当前每屏一份 CSS（含 box-sizing 重置）。大项目更适合「一次出 tokens.css +
   可选 Tailwind 配置映射」；Tailwind/utility 目标未做。
4. **`targetDir` 落进 `src/` 时的**冲突检测（新文件 vs 既有同名组件）只靠 `.bak` + 用户确认，
   未做「已存在且非生成产物 → 拒绝」。建议后续加产物指纹（报告里记 sha，重新生成时校验一致性）。
5. **视图并排的栏宽可拖拽**（当前固定 42%）与**多视图并排**（>2 栏）未做。
6. 发布侧遗留同附录 E.6 / F.6（npm 发布 + 镜像同步 + 市场验收，待凭据）；本轮新增的
   `ui.registerView` 需随插件文档（`docs/plugin-development.md` §12）一并发布。

### J.7 子树抽组件（`component=<节点 id>`，2026-09-17 追加）

**动机**：`design_codegen` 此前以「屏幕」为唯一粒度出组件（整屏一个 SFC）——真实项目里
重复/独立区块（卡片行、导航、列表项）需要**抽成可复用组件**，否则代码只能整体复制。

#### 契约

```js
design_codegen({ component: 'hero-card' })          // 单个：节点 id 或 props.name
design_codegen({ component: 'hero-card,nav-bar' })  // 多个：逗号/空格/数组
```

| 项 | 规则 |
|---|---|
| 匹配 | 先按**节点 id**，找不到再按 **props.name**（同 name 命中多个 → 报错要求改用 id）；只搜**当前选中屏幕** |
| 组件名 | `props.name`（icon 除外，那是图标名）> 语义 id > 类型-序号，再 **PascalCase**；同名自动追加 `2/3` |
| 重复指定 | 同一节点重复出现只抽一次（幂等） |
| 产物 | `vue/components/<Name>.vue`（SFC scoped）／`react/components/<Name>.jsx` + `<Name>.module.css`／`html/components/<Name>.html` + `<Name>.css` |

#### 引用语义（三 target 的差异是关键设计）

- **Vue**：屏幕 SFC 原位写 `<Name />`，并**自动追加** `<script setup>` + `import Name from './components/Name.vue'`；
  屏幕 `<style scoped>` 里**不再含**被抽子树的类规则（真正抽干净）。
- **React**：屏幕 JSX 顶部 `import Name from './components/Name'` + 原位 `<Name />`；屏幕 `.module.css` 同样不再含该子树规则。
- **HTML**：静态页没有组件机制 → 屏幕**保持内联渲染**（保证单文件自包含、可直接打开预览），
  组件以 `html/components/<Name>.html` 片段另出供其它页面复用；此时返回里提示「HTML 没有组件机制」。

#### 嵌套与作用域

- 抽出的组件内部若还包含**另一个被抽节点** → 组件里同样写引用（`import Bd1 from './Bd1.vue'`，同目录相对路径），
  但**排除自身**（避免自引用递归）；屏幕只 import **顶层**组件。
- 组件内类名用**独立去重表**（与屏幕互不影响）：作用域是 scoped / CSS Modules，同名不冲突。

#### 实测

| 层 | 结果 |
|---|---|
| Node 行为测试 | 新增 **3 组 27 条断言**（产物/引用/嵌套/幂等/去重/按 name 指定/边界/未传 component 时行为不变）+ 既有 78 项回归 |
| 真实 goja 沙箱 | **23/23 PASS**：14.4 万字符源码求值、6 工具、三 target 组件产物、Vue `<script setup>` 引用、组件样式令牌化、幂等、报告、边界报错、`design_verify` **D1–D9 全绿** |
| 端到端 | `@vue/compiler-sfc` **parse + compileTemplate + compileScript** 全通过；esbuild **真 bundle**（解析 import 依赖树）→ 证明屏幕 → 组件 → `module.css` 引用链真实可解析；CSS 令牌变量全部有定义 |
| 真浏览器 | 屏幕 `hero.html` 渲染核对（标题/按钮/徽章/复选框/图标/输入框齐全、布局无错乱）；组件片段 `components/HeroBadges.html` **单独打开**渲染核对（徽章底色/边框/圆角正常） |

#### 本轮修掉的缺陷（真浏览器验证抓到的）

- **组件片段的 `tokens.css` 相对路径少一层**：片段在 `html/components/` 下，写成 `../tokens.css` 会解析到
  `html/tokens.css` → **404、片段样式全丢**（控制台 `ERR_FILE_NOT_FOUND`，页面白屏）。修为 `../../tokens.css`。
  → 顺手给 e2e 加了**通用「HTML 相对链接可达」校验**（遍历所有产物 HTML，逐个解析 `href` 并检查文件存在），
  这类「链接断层」以后会被自动抓住，而不是靠人眼看。

#### 遗留（→ 已在**附录 K** 实现）

- ~~组件**参数化**未做~~ → 见 K：内容字段提升为 props、`props.slot` 插槽、屏幕侧按差异传参。
- ~~未做「自动识别重复结构并建议抽取」~~ → 见 K：`detect=true` 出建议、`component=auto` 自动抽取并多实例复用。

---

## 附录 K 组件参数化 + 同构重复识别（2026-09-17 实测）

> 承接 J.7 的两条遗留。原则不变：**真相源仍是设计工程**（改设计 → 重新生成即全站同步）；
> 产物确定（同工程两次生成位级一致）；屏幕侧只写「差异」，不写冗余。

### K.0 契约一览

```js
design_codegen({ component: 'message-card' })   // 抽指定子树（节点 id 或 props.name）
design_codegen({ component: 'auto' })           // 自动识别同构重复区块 → 一个组件多实例复用
design_codegen({ detect: true })                // 只报告候选（不抽取、不改产物）——「先看再决定」
design_codegen({ props: 'none' })               // 关掉参数化：组件内容写死
design_edit({ op:'node.set', id:'n26', props:{ prop:'sender' } })   // 指定 prop 名
design_edit({ op:'node.set', id:'n29', props:{ prop:false } })      // 该节点内容不提升
design_edit({ op:'node.set', id:'n27', props:{ slot:true } })       // 该节点内容改由外部传入（插槽）
```

### K.1 参数化：内容字段 → props

**提升对象**（每种类型的「用户可见文案」，`CONTENT_FIELDS`）：`text.text` / `button.label` /
`badge.label` / `checkbox.label` / `image.alt` / `input.value|placeholder`（有 value 用 value）。

| 规则 | 说明 |
|---|---|
| prop 名 | `props.prop`（显式）> `props.name` > 节点 id（模板自动编号 `c1_n7` → `text7`），再 **camelCase**；保留字（`key/ref/class/style/children/slot/props`）加 `text` 前缀；同名自动 `2/3` |
| 默认值 | **设计原值** —— 不传 props 时渲染与设计稿逐字一致（降级无损） |
| 屏幕侧传参 | **只传与默认值不同**的项 → 单实例抽取时屏幕产物保持 `<MessageCard />` 干净；同构实例各自带自己的文案 |
| 上限 | 单个组件超过 **12** 个内容 prop → 退化为不参数化 + 告警（整屏抽组件会出现荒诞的长参数表，应拆小） |

**三 target 差异**：

| target | 组件产物 | 屏幕产物 |
|---|---|---|
| Vue | `<script setup>` + `defineProps({ sender: { type: String, default: '…' } })`，模板 `{{ sender }}` | `<MessageCard />` 或 `<MessageCard sender="…" />`（属性名统一 camelCase，Vue/React 两侧都能匹配到 prop） |
| React | `export default function MessageCard({ sender = '…', children })`，JSX `{sender}` | 同上；`children` 用于插槽 |
| HTML | 片段**保持字面量**（静态页没有组件机制） | 屏幕保持内联渲染（自包含可预览） |

### K.2 插槽：`props.slot`

被标记的节点在组件里**保留外壳与样式**、内容换成插槽口；屏幕侧把该节点的子树写进组件标签内部：

```vue
<!-- 组件 --> <div class="d-message-card"><slot /></div>
<!-- 屏幕 --> <MessageCard><p class="d-n26">帮我做一个登录页</p></MessageCard>
```

- React 侧：组件解构 `children` 并渲染 `{children}`。
- 插槽子内容在**屏幕的类名表**里渲染、插槽外壳在**组件的类名表**里 → 两套 scoped/module 互不干扰
  （e2e 断言：屏幕 CSS 含 `.d-<子树节点>`、组件 CSS 不含）。
- 插槽宿主自身的内容字段**不**提升为 prop（它由插槽传入）；当前只支持**一个**插槽宿主，多个时告警并只用第一个。

### K.3 同构重复识别（`detect` / `auto`）

**结构指纹**：节点类型 + 关键样式（`props` 里**剔除** `name`/`prop`/`slot`/内容字段；键值写成 `键=长度:值`
防分隔符假碰撞），递归组合子节点（带长度前缀）→ 逐字符比较。

| 规则 | 说明 |
|---|---|
| 候选门槛 | 出现 **≥2 次**；子树 **≥3 个节点**；**屏幕根不参与**（它是屏幕容器不是区块） |
| 范围 | **跨屏**（同一组件天然可跨屏复用）；忽略节点 id/名称与文案（文案会参数化） |
| 组语义 | **一个组件 = 一组同构实例**；原型取组内**先序第一个**（确定）→ 其文案成为 props 默认值 |
| 去噪 | 大组优先；成员**全部**落在已选组子树内的候选被丢弃（不抽碎片组件）；已被显式抽走的不再列为候选 |
| `detect` | 只报告（区块名/规模/出现次数/各实例位置 + 抽取命令提示），**不写组件产物** |
| `auto` | 自动抽取；**一个都没识别到 → 明确报错**（静默无操作最难排查） |

### K.4 实测矩阵（本轮）

| 层 | 结果 |
|---|---|
| Node 行为测试 | `_temp/tool-design-codegen-test.cjs` **181 项全绿**（新增 3 组：参数化 17 项、重复识别 15 项、插槽 11 项；含 `props:none`/改名/排除/上限退化/幂等/边界报错） |
| 真实 goja 沙箱 | 探测插件用 `new Function` 加载**真实源码**（16.5 万字符）+ 内存 mock fs：**24/24 PASS**（含 `component=auto` 差异传参、`<slot />`、幂等、三类边界报错） |
| 端到端真编译 | `_temp/design-codegen-e2e.cjs` **全绿**（新增 19 项）：`compileScript(inlineTemplate)` 编译产物 → **`vue/server-renderer` SSR 真渲染** —— 不传 props 得到设计原值、传 props 得到传入文案（证明参数化在**运行时**生效，不是文本替换）；`compileTemplate` 编译**含组件传参的屏幕模板**；esbuild 真转译屏幕 JSX；HTML 相对链接可达校验 |
| 真浏览器 | ① SSR 预览页截图核对：两组徽章文案正确、胶囊样式一致；② SSR 输出带 `data-v-hb1`（scoped 生效）；③ ui-design 面板截图核对（组件行 + 候选行） |
| 面板（CDP） | `_temp/ui-design-codegen-panel-verify.mjs` 全绿：`抽出组件 MessageCard ← chat/n23（3 props）`、`重复区块候选 共 3 组：N39（card）×2 · N24（row）×3 等（component=auto 可抽）`、无 console 错误 |
| 硬门槛 | `go vet` 0 / `go build` 0 / `go test -short ./internal/agent` ok / `verify-dist-isolation` PASS / 9097 实例 41 插件、`tool-design` 6 工具 |

### K.5 本轮修掉的缺陷（都是「看起来对、跑起来错」那一类）

1. **参数化开关与插槽耦合**：插槽判定挂在 `opts.props` 上 → `props:'none'` 时插槽也不生成。拆成
   `opts.comp = { params, used, list }`（插槽由 `props.slot` 决定，参数化由 `params` 决定）。
2. **`props:'none'` 未真正生效**：`genComponentArtifacts` 自己新开了 `props:{used,list}`，与 `comp.props` 脱节
   → `none` / 超上限时仍生成 `defineProps`。改为按 `comp.props.length` 决定是否开启。
3. **引用节点吞掉插槽子树 CSS**：`treeCssRules` 遇 `ref` 直接 `return`，插槽子内容（在屏幕类名表里）的规则被跳过
   → 屏幕渲染无样式。改为对 `ref` 的 children 继续递归。
4. **空元素分支提前返回**：`<slot />` / `{children}` 排在「无 children 无 text → 直接输出空对标签」之后，
   从未被输出（`<div class="d-sc1"></div>`）。把插槽分支提前到空元素判断之前。

> 另有一个**验证脚手架**层面的坑（非产物缺陷）：Vue 3.5 的 `compileScript` **不注入 `__scopeId`**
> （由 `@vitejs/plugin-vue` 负责）→ SSR 输出缺 `data-v-xxx`，scoped 选择器全不匹配、看起来「样式全丢」。
> 脚手架按插件行为补 `Comp.__scopeId = 'data-v-hb1'` 后恢复正常；e2e 增加了对应断言。

### K.6 遗留

1. ~~**具名多插槽**未做~~ → **已做**，见附录 L.1（多插槽表 + 命名/报错契约 + Vue `<template #x>` / React 具名 props）。
2. 组件**事件/交互**未建模（按钮回调、表单绑定）—— 设计工程仍只描述静态外观 → 边界已确认，见附录 L.7-2。
3. ~~**近似同构**不识别~~ → **已做报告**（只报告不误抽，含差异字段与自动抽取的失败线索），见附录 L.2。
4. 发布侧（12 个包 npm 发布 + dist-tags + 市场真链路）待有效 NPM_TOKEN（现有凭据 401）；
   **不依赖发布**的镜像同步 / 本地包校验 / 本地安装验收已完成，见附录 L.3。

---

## 附录 L 清遗留（具名多插槽 / 近似同构 / 镜像与本地包验收，2026-09-17 实测）

> 本轮范围：把附录 K.6 的功能遗留与工程遗留继续清完；**发布相关一律不做**（NPM_TOKEN 仍失效）。

### L.1 具名多插槽（一个组件多个 `props.slot` 宿主）

附录 K 的插槽只支持一个宿主（多个时「只用第一个 + 告警」）。本轮改为**多插槽表**：

| 层 | 实现 |
|---|---|
| 契约 | `comp.slots = [{ nodeId, name, declared, hostType }]`（DFS 顺序 = 实例对齐顺序）；旧字段 `slotNodeId/slotLabel` 仍输出（= 首个槽，兼容旧消费者） |
| 命名 | `props.slot:'名字'` → 归一化 camelCase（`left-panel` → `leftPanel`，Vue `#name` 与 React props 名共用一套写法）；`props.slot:true` → **第一个未命名宿主 = 默认插槽**，其余按 DFS 位次 `slot2/slot3…` 并告警 |
| 产物（Vue） | 组件：`<slot />` / `<slot name="header" />`；屏幕：默认槽内容写在标签内、具名槽 `<template #header>…</template>` |
| 产物（React） | 组件：签名逐个声明插槽 prop（`{ header, children, footer }`），占位 `{header}` / `{children}`；屏幕：`header={<></>}` 多行属性形式（Fragment 包多子节点） |
| 产物（HTML） | 静态片段无组件机制 → 不抽插槽（子树内联），与参数化同样处理 |
| 报错契约 | 插槽名非法（归一化后为空）／两名重复／叫 `children`（React 保留名）／与文案 prop 重名 → **直接报错**（产物会非法，不生成含糊结果） |
| CSS | 每块插槽子内容在**屏幕的类名表**里渲染（组件侧 CSS 不含），`treeCssRules` 对引用节点继续递归其全部插槽子内容 |

### L.2 近似同构（结构相同、只有颜色不同）：**报告但不误抽**

- 新增宽松指纹 `structFingerprintLoose`（`bg/color/border/shadow/tone` 归一化为 `<color>`）+ `findNearGroups`：
  同组内**严格指纹不唯一**（确实存在差异）且差异落在颜色字段 → 近似组，附 `diffKeys`。
- **为什么不自动抽**：组件默认值只能有一份，配色不同的实例会被压成同一外观 —— 设计上本来要区分的配色就丢了。
  所以近似组只出现在 `detect` 报告里（含差异字段 + 为什么 + 两条行动路径：对齐 token 后 `auto`／确认有意区分则分别抽）。
- `component=auto` 且**零严格候选**时的报错信息里会带上近似线索（`N 组近似… 差异字段 bg`）——
  否则用户只看到「没识别到」，不知道其实只差配色对齐。
- 判据不混淆：近似组**不计入**严格候选（`duplicates` 与 `nearDuplicates` 分开记录），对齐配色后同一组自动升级为严格候选并可 `auto` 抽取（用例 20 端到端验证了这条升级路径）。

### L.3 工程遗留（不依赖发布的部分，全部做完）

| 项 | 结论与处置 |
|---|---|
| 镜像同步 | **不依赖发布**，已执行 `node scripts/sync-plugins-to-bin.mjs`（plugins 112 / toolsets 8 / assets 7 文件）。剩余差异 = 12 个**开发态 junction** 插件（符号链接，脚本按规则跳过）—— 与 `packager.json` 的 `exclude`（同样这 12 个）**完全对应**：安装包不含它们，镜像也不含它们 ⇒ 不存在「旧版随「只补缺」迁移复活」的污染路径。 |
| 本地包校验 | `_temp/plugin-pack-check.mjs`：12 个包逐包 `npm pack --dry-run --json --ignore-scripts` ⇒ 12/12 通过：版本号一致 `0.1.0`、`package.json`+入口齐、6 个 UI 包带 `client.js` 与构建产物 `assets/*-panel.js`、无 `node_modules/.bak/.pair` 残留。**（只校验，不发布）** |
| 本地安装验收 | `_temp/plugin-install-verify.mjs`：真打包 → **自写 tar 解包**（zlib + pax 解析，不依赖系统 tar）→ 与宿主同构 mock ctx 装载 ⇒ tool-design 注册 **6 工具**且与源目录版清单**逐一致**；解包条目数 = pack 清单数；UI 包 `client.js`+产物齐、语法校验通过；与实例 `/api/plugins` 对照 ⇒ `tool-design state=running tools=6`、`ui-design hasClient=true`。 |
| `.bak` | 结论：**保留**（`design_codegen` 覆盖前备份，属回滚特性，面板也展示其数量）。它们位于未跟踪的 `design-export/`；已在 `.gitignore` 增加 `*.bak`（备份文件不入库）。 |

### L.4 环境事实（本机构建，避免重复探测）

1. **必须 `CGO_ENABLED=0` 构建**：`CGO_ENABLED=1` 时 `internal/agent/embedding_onnx.go`（`//go:build cgo`）启用 →
   依赖 `onnxruntime_go`，而该库在 windows/amd64 无可用文件（build constraints）⇒ 构建失败。
   另外 `scripts/env/onnxtest/main.go` **无 build tag** 却 import onnx ⇒ `go vet ./...` 必然失败。
   ⇒ 本机回归用 `go vet ./cmd/... ./internal/... ./pkg/...` + `go build ./cmd/companion` + `go test -short ./internal/agent`（均 0/ok）。
2. **两类前端产物**（别再混）：`DesignPanel.vue` 等**区域面板**在 `ui-design` 插件里 → 构建用
   `node scripts/build-ui.mjs --region design`（产物 `plugins-dist/ui-design/assets/design-panel.js`，**磁盘插件运行时读盘，改完即生效、无需重启**）；
   而壳（`npm run build` in `plugins-src/ui-app`）产物 `.pair/assets/runtime/web` 是**编译期内嵌**进二进制
   （`cmd/companion/web-ui/dist`）⇒ 改壳必须**重编译 + 重启实例**才可见。
3. 测试实例：`WEB_PORT=9097 ./companion_test.exe`（9090 属用户宿主，绝不触碰）；本轮已重编译并重启为含新前端的版本。

### L.5 验证矩阵（本轮）

| 层 | 结果 |
|---|---|
| Node 行为 | `_temp/tool-design-codegen-test.cjs` **全绿**（新增 31 项：具名多插槽 19、命名回退与非法名报错 3、近似同构 9） |
| 真实 goja 沙箱 | 探针在沙箱内 `new Function` 求值**真实源码** + 内存 mock fs：**14/14 PASS**（三插槽口、屏幕 `<template #x>`、React 具名 props、报告 3 槽元数据、未命名告警、近似报告与 auto 提示、幂等） |
| 端到端真编译/真渲染 | `_temp/design-codegen-e2e.cjs` 新增 22 项全绿：屏幕模板 `compileTemplate` 真编译 → **SSR 真渲染** ⇒ header/默认/footer 三块内容分别落在**对应元素内部**、顺序 = 设计顺序、`data-v-*` scoped 生效；React 组件/屏幕 JSX 真转译 |
| 真浏览器 | Edge 无头截图（`_temp/design-e2e/shot-slot2.png`）目检：卡片内「卡片标题 / 卡片正文 / 关闭」三段依次呈现、样式与外壳完整、无重叠残缺 |
| 面板（CDP） | `_temp/ui-design-codegen-panel-verify.mjs` 全绿（含**双态**验证「近似区块」行：报告含近似组 → 显示 `共 1 组：N39（card）×2 差异 bg`；无近似组 → 不显示，不占面板） |
| 硬门槛 | `go vet` 0 / `go build` 0 / `go test -short -count=1 ./internal/agent` ok 66.2s / 镜像一致性核对 / 12 包 pack 校验 / 本地安装验收 全绿；9097 实例 41 插件、`tool-design` running 6 工具 |

### L.6 本轮踩的坑

1. `findNearGroups(screens, minCount, minNodes)` 的第二参是 **minCount**，我照抄 `detectDuplicates(screens, maxItems)`
   的调用位置传了 `20` ⇒ 近似组全被过滤（症状：报告永远为空）。教训：同名前缀的查询函数参数语义要逐个确认。
2. Windows 下 `execFileSync('npm.cmd')` 报 `EINVAL`（需走 shell）；`execSync('tar …')` 走 `cmd.exe` 时静默 status 2
   ⇒ 包校验改用 `execSync` + 自写 tar 解包（zlib + pax），跨平台且可断言「解包条目数 = pack 清单数」。
3. 「改了面板却看不到」的根因是**构建对象选错**（区域 bundle vs 内嵌壳），见 L.4-2 —— 两次误判后靠
   `grep 产物字符串` 定位（产物里没有新字符串 ⇒ 构建没覆盖到该产物）。

### L.7 遗留

1. ~~**近似组不自动抽取**~~ → **已做**，见附录 M.2：`component=near` 把配色差异提升为 **CSS 变量**
   （组件 `var(--c<序号>-bg, 原型值)` + 实例内联覆盖），tone（语义色）/ 插槽宿主等不可变量化的边界明确不抽并说明。
2. ~~**组件事件/交互**仍未建模~~ → **已做**，见附录 M.1：节点 `props.action` 受控词汇表
   （`link:<目标>` / `emit:<名字>` / `toggle`）→ 三端绑定（Vue `@click`+`defineEmits` / React `onX` 回调 prop /
   HTML `data-action`），`design_verify` 增 **D10** 判据。原判断（"四处同时扩展"）成立：确实改了 schema 允许字段的
   写入面（`props.action` 自由通道即可，无需改 `NODE_FIELDS`）+ 面板 + 校验器 + 三端 codegen。
3. 发布侧（12 包 npm 发布、dist-tags、市场安装真链路）待有效 `NPM_TOKEN` —— 仍未做（附录 M.6-1）。

## 附录 M 清功能遗留（交互建模 / 近似组自动抽取 / 真缺陷修复，2026-09-17 实测）

> 本轮范围：把附录 K.6 / L.7 里剩下的**功能遗留**清完（具名多插槽、近似同构已在附录 L 完成）。
> 发布相关（npm publish / dist-tags / 市场 registry 安装）**一律不做** —— `NPM_TOKEN` 仍失效。

### M.1 交互建模：节点 `props.action` 受控词汇表

**为什么不给设计稿自由脚本**：① 自由 JS 无法校验（Agent 写错只在运行时炸）；② 预览产物（`design_export format=html`）
是**截图式自包含**的，D9 明确禁止 `<script>` / 内联 `on*` / `javascript:` / 外链 —— 设计稿带脚本这条判据就永远 FAIL；
③ 同一份"意图"要落到 Vue（`emit`）/ React（回调 prop）/ 静态 HTML（`data-action`）三种惯用写法，只有声明式动词能各自生成。

| 写法 | 含义 | 目标白名单 |
|---|---|---|
| `link:<目标>` | 跳转 → **真链接** `<a href>` | `https://` `http://` `mailto:` `tel:` `/站内路径` `#屏幕id` |
| `emit:<事件名>` | 交给宿主处理 | `^[A-Za-z][A-Za-z0-9]*$` |
| `toggle` | 切换（原生交互） | 仅 `checkbox` |

- `#屏幕id` → 产物 `./<屏幕id>.html`（"屏幕即页面"）；`link` 只对 `frame/row/col/card/list/text/button/badge/icon/image` 有意义。
- 三端产物：**Vue** `@click="emit('x')"` + `<script setup>` 里 `defineEmits(['x'])`（组件与屏幕都可用 —— 屏幕是根组件，
  父级照样监听）；**React** `onClick={onX}`（组件解构 `onX = () => {}`，屏幕内 `const onX = () => {}` 占位可直接替换）；
  **HTML 静态页**无脚本 → `data-action="emit:x"` 由宿主接线；`link` 三端都是真 `<a href>`（换标签时补 `text-decoration: none`）。
- **预览只落动词**：带 `action` 的节点加 `data-action="<动词>"`（不含 URL/处理器）⇒ **D9 依旧 PASS**，预览仍自包含。
- **抽组件时**：组件内部 `emit:` 自然变成组件对外事件（Vue `defineEmits` / React 回调 prop），屏幕引用处**不重复绑定**。
- **新增判据 D10（第 10 项）**：动词/目标/名字/类型适配 + `link:#屏幕` 必须真实存在；codegen 出**告警**（该节点不出绑定，
  其余照常生成），D10 FAIL（构建门槛）。报告 `interactions=[{screen,id,type,action,verb,target,error}]`，
  面板「交互」行条件显示（`3 处：link ×1 / emit ×1 / toggle ×1`，有问题另列明细）。

### M.2 近似组自动抽取：`component=near`（配色差异 → CSS 变量）

附录 L 只做到"报告不抽"，理由是"组件默认值只能一份，硬抽会把设计上要区分的配色压平"。本轮用 CSS 变量解开这个矛盾：

1. `nearDiffTable(members)`：按**先序对齐**逐节点比较 `COLOR_KEYS={bg,color,border,shadow,tone}` →
   `{先序序号: {fields, <字段>: 原型值}}`；
2. `nearWrapTree`：组件侧把命中声明包成 `var(--c<序号>-<后缀>, 原型值)`（后缀 bg / color→fg / border / shadow）；
3. `nearStylesFor`：实例侧只对**与原型不同**的字段产出 `--c<序号>-<后缀>: <CSS 值>`（与组件侧同一套 CSS 解析 ⇒ 可比可覆盖）；
4. 引用处：Vue/HTML `style="--c0-bg: var(--color-accent-soft)"`；**React 走对象形式** `style={{ '--c0-bg': '…' }}`。

⇒ **一份组件、多实例**，默认值 = 原型实例值，配色差异由实例内联覆盖 —— 复用与区分都不丢。

**明确边界（不静默降级）**：`tone` 差异（语义色同时决定前景+背景两条声明，不是单一可变量化单位）→ 不抽并说明；
组内含**插槽宿主** → 不抽并说明；一个可抽组都没有 → 明确报错附原因。报告 `components[].nearVars=[{seq,node,fields:[{field,cssVar}]}]`，
面板组件行显示 `（配色 → 变量 --c0-bg）`。

### M.3 顺带修的真缺陷：col 容器里自宽叶子被 flex 拉伸

**真浏览器截图才抓到**（Node 断言全绿）：`card`（col）里的 `badge` 被 flex 默认 `align-items: stretch` 拉成
一条贯穿整行的色带 —— 与设计预览（绝对定位、宽度=内容宽）不一致。
修法：`SELF_WIDTH_TYPES={badge,button,icon,checkbox}` 在父容器为 `col` 时补 `align-self: flex-start`
（满宽按钮 / `w=fill` / 100% 宽除外），父方向由 `optsWithDir(opts, dir)` 递归传递。
**一般化教训**：预览=绝对定位、产物=flex 流式 ⇒ 凡是"flex 默认行为与设计稿不同"的点，codegen 必须显式写死；
这类差异 Node 断言与模板编译都发现不了，必须**真浏览器 + `getComputedStyle`**。

### M.4 验证矩阵（本轮实测）

| 层 | 手段 | 结果 |
|---|---|---|
| Node 单测 | `_temp/tool-design-codegen-test.cjs`（新增 step 21/22/23/24/25） | 全绿（多插槽 31 项 + 本轮新增 32 项） |
| 真实 goja 沙箱 | cordis 探针加载真实源码跑全场景（`_temp/goja-act-near-probe.json`） | **13/13 PASS** |
| 端到端 | `_temp/design-codegen-e2e.cjs`（新增 step 14/15）：SFC 真编译 + **真 SSR** + esbuild 真转译 | 全绿（`@click` 编译成 `onClick`、`<a href>` 落地、near 组件 `compileScript` 真编译、两实例渲染出不同 style） |
| 真浏览器 | `_temp/design-act-near-shot.mjs`（CDP DOM 探针 + 截图目检） | 两实例背景 `rgb(249,250,251)` vs `rgb(239,246,255)`；`A[href=./task-board.html]`、`data-action="emit:openTask"`；徽章已是紧凑胶囊 |
| 面板 | `_temp/ui-design-codegen-panel-verify.mjs`（新增交互行 / 配色变量断言） | 全绿（条件显示按**报告实际内容**判定，不靠环境变量猜） |
| Go / 包 / 隔离 | `go vet` 包集 0、`go build ./cmd/companion` OK、`go test -short ./internal/agent` ok 65s、`verify-dist-isolation` PASS、12 包 pack 校验 PASS | 全绿 |

### M.5 本轮踩的坑（细节见知识库《交互与配色变量抽取》§五）

1. `findNearGroups` 的差异判定只看**组根节点自身** → "差异在子节点上"（badge 的 tone）的组被静默丢弃；改为逐节点对齐取并集。
2. 有 `<script setup>` 的组件只 `compileTemplate` 手搓 → props/emits 声明丢失，SSR 报 `Property "x" … not defined`。
3. Vue 会把内联 style 的 CSS 变量**规范化**成 `--c0-bg:var(--x);`（无空格）→ 断言要 `\s*` 容错。
4. 沙箱 `ctx.fs` **写盘只能落在宿主固定根内**、且 `writeFile` **不建父目录** → 探针前先拷源码 + `mkdir -p` 产物目录。
5. `apply_patch` 定位纪律：只给 `-旧行` 无上下文可能被插到**文件开头**；同名分隔行会命中第一处 → 用唯一多行上下文或"写 snippet + 脚本按唯一锚点插入"。
6. 演示脚本要**幂等**：重复 `node.add` 会叠加重复区块（实测 3 处交互变 6 处）。

### M.6 导航与交互联动：flow ↔ action:link（D11 + flow.sync）

**问题**：屏幕跳转在设计真相源里有**两处**声明 —— `flow`（信息架构层：屏幕之间可达，导出 mermaid 架构图）与
节点 `props.action=link:#屏幕`（实现层：哪个元素触发跳转，codegen 落成 `<a href>`）。语义重叠又各写一份
⇒ 架构图与实际行为会**悄悄漂移**（改了交互忘补边，评审看不出来）。

**做法**（两侧各补一步，不改真相源结构 —— 两者仍是各自领域的合法声明）：

1. `design_verify` 新增 **D11「导航与交互一致（flow ↔ action:link）」**：
   ① 节点声明 `link:#屏幕` 却没有对应导航边 → **FAIL**（详情直接给出修复路径 `design_edit op=flow.sync`）；
   ② 有导航边但**没有任何节点**声明跳转 → 只**提示**（允许系统/外部触发的跳转存在，不判 FAIL）；
   ③ 外链 / 站内路径链接不参与（它们不是"屏幕之间可达"，也不进架构图）。
2. `design_edit` 新增 **op=`flow.sync`**：从所有 `action:link:#屏幕` 推导导航边，**只增不删**（幂等，可反复执行）；
   `prune=true` 才一并删掉「无交互支撑」的边（保守：手写的边、系统触发的边不会被误删）。

**验证（真 goja 沙箱，不是 Node 模拟）**：cordis 探针在 goja 里 `eval` 加载 `plugins-dist/tool-design/index.js`
**真源码**，在**工作区真工程副本**上跑闭环 —— 先删掉 `plugins → verify` 边 + 给按钮 `n50` 标 `link:#verify`：
D11 **FAIL**（详情「n50（屏幕 plugins）声明了 link:#verify，但导航图里没有 plugins → verify 这条边（用 design_edit op=flow.sync 补齐…）」）
→ `flow.sync` 补齐 → D11 **PASS**（「跳屏幕 2 处，全部有导航边」）→ 重复 `flow.sync` 幂等。
工作区真工程实测：11 项判据、D11 PASS（「跳屏幕 1 处，全部有导航边；1 条导航边无节点触发（plugins → verify）」）。

### M.7 遗留

1. **发布真链路**（12 包 npm publish、dist-tags、市场 registry 安装）待有效 `NPM_TOKEN`（现有 401）—— 本轮按约定不做。
2. 其余域的 L2 后续（非本轮遗留）：扭转/扫掠/圆角倒角、rig 动作时间线、Tailwind 目标、
   产物指纹与"拒绝覆盖非生成产物"、多视图并排与栏宽拖拽 —— 按各域计划继续。
3. 工作区真工程（`design.project.json`，3 屏 chat/plugins/verify）当前 **D6 触达区 + D8 布局溢出 FAIL**（既有状态，
   与本轮改动无关）：chat 屏的 `act-open`/`act-send` 最短边 37px < 44px、`act-remember` 复选框 16×16、
   `chat_root` 纵向溢出 783.2px > 688px 且 `near-b-*` 越出屏幕 —— 属设计稿本身待收敛项。
