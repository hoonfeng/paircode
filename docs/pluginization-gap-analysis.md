# PairCode IDE 插件化缺口分析报告（「一切皆插件」审查）

> 审查人：analyst（插件化架构分析师）｜日期：2026-09（基于当前工作树）
> 标尺：**一切皆插件** —— 能力实现留内核（Go），挂载/编排/内容在插件（JS/磁盘），
> 插件可装卸、可替换、随生命周期生灭；「信息形态与 ctx 服务一致的能力」都应可插件化。
> 配套文档：`docs/plugin-development.md`、`docs/go-core-capabilities.md`、`.pair/plugins/README.md`。
>
> ★ 实施状态（t2，2026-09）：本报告 P0/P1 缺口已按 `docs/plugin-gaps.md` 闭环
> （P0 全部、P1 全部；P2 记录理由），改动文件清单与验证见该文件。

---

## 0. 审查结论速览

| 维度 | 总体状态 | 关键缺口数 | 综合评级 |
|---|---|---|---|
| 1 工具/工具集 | 18 组磁盘工具插件 + hostTool 存档，基本闭环 | 4 | ★★★☆ 良好 |
| 2 插件宿主生命周期 | 6 态 + inject 等待 + 版本化 + npm 桥，成熟 | 4 | ★★★☆ 良好（桌面端除外） |
| 3 内容资产 | skills/指令/知识库目录化 + 提示段插件化 | 4 | ★★☆☆ 偏弱 |
| 4 前端面板 | 7 区域插件 + 槽位系统 + 面板一体化，优秀 | 4 | ★★★★ 优秀 |
| 5 服务集成 | 接口/市场/MCP/provider 装配器插件化，优秀 | 4 | ★★★☆ 良好 |
| 6 配置存储 | pluginSettings 命名空间 + schema 驱动，良好 | 4 | ★★★☆ 良好 |

**高优先级（P0）缺口 3 项**（实施 t2 时应优先闭环）：
1. **桌面端（cmd/desktop + internal/desktopbridge）完全没有插件宿主** —— 无 PluginHost、无工具集、无 UI 插件装配。
2. **config/roles/*.md 与 config/philosophy/*.txt 是「已发布但永不被读取」的死内容资产**；角色提示实际是 Go 硬编码且不可覆盖。
3. **7 组孤儿工具注册函数（约 20 个工具）既未插件化也未装配**（零调用点）；另有 vision 工具（image_analyze/image_ocr）已删除但插件描述仍引用。

---

## 1. 工具 / 工具集（Tools / Toolsets）

### 1.1 已插件化的部分（现状基线）
- 内置 20 组工具实现已迁移磁盘插件：`.pair/plugins/tool-*`（18 个目录）经 `LoadGlobalPlugins` 启动自动装载（`internal/agent/toolset.go:540`）。
- 宿主只注册框架协议工具 `RegisterHostFrameworkTools`（SystemTool 组：update_tasks/update_plan/tool_stats/history_*），见 `internal/agent/builtin_plugins.go:105`。
- 插件同名接管时 Go 实现自动存档进 `hostExecutors`（`ArchiveHostTool`，`internal/agent/host_executors.go`），插件经 `ctx.hostTool.exec` 复用宿主能力（seam 模式）。
- 工具集 = 插件定义集合 + 工具挑选清单（`.pair/toolsets/*.json`），工具可见性由工具集白名单控制（`internal/agent/builtin_toolset.go`）。
- 工具集构建模板可插件化：`ctx.toolset.registerTemplate`（`internal/agent/toolset_templates.go`）；内置 5 模板内联宿主（文档明确为框架能力）。
- 独立二进制插件：`plugins-src/plugins/tool-*/` + `ctx.binary.exec`，内嵌内核回退 `embedded_tools.go`。

### 1.2 缺口（带证据）

**[T1｜P0] 7 组孤儿工具注册函数：约 20 个工具既未插件化也未装配（零调用点）**
| 函数 | 工具 | 文件 |
|---|---|---|
| `registerAssetTools` | asset_list / asset_search / asset_delete | `internal/agent/asset_tools.go:24` |
| `registerBridgeTools` | bridge_status / bridge_takeover / bridge_lockdown / bridge_exec / bridge_register_system_tool | `internal/agent/bridge_tools.go:26` |
| `registerEntryConfigTools` | find_entry_points / find_config_files | `internal/agent/entryconfig.go:247` |
| `registerEvolutionTools` | evolution_save_capsule / evolution_search_capsules / evolution_save_gene / evolution_status | `internal/agent/evolution_tools.go:14` |
| `registerProgressChecker` | progress_checker | `internal/agent/progress_checker.go:14` |
| `registerResourceTools` | resource_list / resource_search / resource_stats | `internal/agent/resource_tools.go:18` |
| `RegisterSnapshotTools` | restore_snapshot / list_snapshots | `internal/agent/snapshot.go:164` |

证据：全仓库 grep `registerAssetTools(` 等仅命中函数定义，无任何调用点；不在 `builtinPluginSpecs`（`builtin_plugins.go` 现表 17 组不含这些）、不在 `RegisterHostFrameworkTools`、不在任何工具集 JSON、不在任何磁盘插件、不在测试中。**这些工具实现了但 Agent 永远用不到** —— 既非宿主工具也非插件工具（历史迁移遗漏或未迁移）。

**[T2｜P1] vision 工具删除但插件描述仍引用（内容漂移）**
- `registerVisionTools` 无调用点；`.pair/plugins/tool-vision/index.js` 只注册 `submit_image`。
- 但 `.pair/plugins/tool-screenshot/index.js:11` 与 `.pair/plugins/tool-web-debug/index.js:11` 的描述仍写「可用 image_analyze 分析截图…或用 image_ocr 识别文字」—— 引用已不存在的工具，LLM 会被误导。
- `internal/agent/toolset_visibility_test.go:42` 也把 image_ocr 当示例（测试本身无碍，但反映清单陈旧）。

**[T3｜P2] 工具描述「文言文精简」硬编码在 Go（内容未插件化）**
- `internal/agent/tools_concise.go` 的 `conciseToolDescriptions` map + `ApplyConciseToolDescriptions` 在 Go 侧覆盖 35 个工具描述。工具描述本应由插件声明（`usageGuide/description`），Go 侧再覆盖一层压缩版 → 内容资产留在内核，插件无法控制最终送 LLM 的描述形态。

**[T4｜P2] 工具集模板注册在 Go 内联（设计决策，记录即可）**
- `registerBuiltinTemplates`（`internal/agent/plugin.go:746` → `toolset_templates.go:382`）5 个内置模板内联宿主，文档明确为框架能力（提供者即宿主）。可接受，但「模板即内容」未来可考虑迁到磁盘插件。

---

## 2. 插件宿主生命周期（Plugin Host Lifecycle）

### 2.1 已插件化的部分
- `PluginHost`：6 态（stopped/running/waiting/rejected/failed/cancelled），`internal/agent/plugin.go:345-410`。
- inject 等待语义（D3）：服务缺失 → waiting，服务提供后自动激活（`plugin.go:1255` `retryWaiting`）。
- 版本化工作流：`cordis_define`（追加版本）/ `cordis_run`（指定版本）/ `cordis_stop` / `cordis_undefine`（`internal/agent/plugin_tools.go`）。
- 三形态插件：Go 插件（plugin.go）、JS 动态插件（goja 沙箱，jsplugin.go）、Node 桥 npm 插件（node_plugins.go + npm_plugin.go）。
- 磁盘插件包（`.pair/plugins/<name>/`）启动自动装载；`cordis.patch.json` 静态装配；动态插件经 `syncGlobalPlugin` 固化为插件包。
- client 半激活审批机制已取消（IsClientApproved 恒 true，对齐外部项目）。

### 2.2 缺口

**[L1｜P0] 桌面端（cmd/desktop + internal/desktopbridge）完全没有插件宿主**
- `internal/desktopbridge/desktopbridge.go`：无 `NewPluginHost`、无 `LoadGlobalPlugins`、无 `LoadAllToolsets`、无 `SetPluginHost`、无 `RegisterCordisTools/RegisterToolsetTools`、无 `LoadCordisPatch`（grep 零命中）。
- `buildDesktopLoopOpts`（desktopbridge.go:395）只 `RegisterHostFrameworkTools` + MCP 注册，**不合并插件工具**（无 `MergePluginTools`）→ 桌面会话 Agent 只有框架工具，业务工具（read_file/write_file/git/…）全部缺失。
- 桌面加载同一前端壳（`cmd/companion/web-ui/dist`，bundle 含 `__PAIRCODE_CORE`），但 `/api/plugins` 因 PluginHost nil 返回空 → 7 区域 UI 槽位全部「未装配」。
- 已知遗留记录：`.pair/memory/全UI插件化-槽位细粒度7区域包2026-08-16.md` 明确「wb-ui 桌面端需适配 7 区域插件装配（主端 web 不受影响）」。**若桌面端仍是交付物，这是最高优先级缺口；若已废弃，应显式声明。**

**[L2｜P1] 12 个 internal 包移植自 DeepSeek-Reasonix 但生产代码零导入（死包 / 未接线能力）**
`internal/hook`（PreToolUse/PostToolUse/UserPromptSubmit/Stop 钩子系统）、`internal/jobs`、`internal/permission`、`internal/provider`、`internal/vterm`、`internal/agenttools`、`internal/codetypes`、`internal/event`、`internal/store`、`internal/model`、`internal/summary`、`internal/verify` —— 全仓库仅 `scripts/relocate_imports.go` 提及，生产代码（cmd/、internal/agent 等）零导入。
- 尤其 **hook 系统**：`internal/hook` 完整实现了钩子 Runner/信任门（PreToolUse 阻塞语义等），但 `internal/agent/loop.go` 中零调用 → 钩子能力「移植了但从未接线」，且未插件化。
- 处置建议：要么接入 Loop（作为宿主能力 + 插件可注册钩子），要么删除包并清理 `scripts/relocate_imports.go`。

**[L3｜P2] AgentBase（独立基座）生产未使用，初始化逻辑双份**
- `NewAgentBase/Init/Run/Shutdown`（`internal/agent/agent.go:85`）只有测试调用（`plugin_test.go:739`）；生产 web 端在 `cmd/companion/web_server.go:257-300` 手写约 40 行初始化（NewPluginHost + RegisterCordisTools + LoadAllToolsets + LoadCordisPatch），desktop 又一份 `desktopbridge.Init`。**三处生命周期装配逻辑重复**，容易漂移（桌面端漂移已发生，见 L1）。

**[L4｜P2] 内置 inspect provider / 审批遗留字段**
- `plugin.go:469-476` 保留 approvedClients/approvedGlobal/approvedProject 废弃字段与 load/save 函数（仅兼容，无效果）—— 死代码建议清理。

---

## 3. 内容资产（Content Assets）

### 3.1 已插件化的部分
- Skills 目录化：`config/skills/`（system 级）+ `.pair/skills/`（project 级），三级渐进披露（`internal/agent/skill_loader.go`）。
- 指令：`.pair/instructions.md`（项目级）+ settings.systemInstructions（系统级），经 `/api/instructions` 读写。
- 项目知识库 `.pair/project-info/`、记忆 `.pair/memory/`、背景快照注入。
- 系统提示段/变量：插件 `ctx.systemPrompt.section/variable` 贡献，`deployment:persona/rules` 槽位可替换（`internal/agent/prompt_registry.go`）。

### 3.2 缺口

**[C1｜P0] config/roles/*.md 与 config/philosophy/*.txt：已发布但永不被读取（死内容资产）**
- `packager.json` 的 dist.include 明确打包 `config/roles` 与 `config/philosophy`（递归）。
- 但全仓库 grep：**没有任何 loader 读取这些文件**。`internal/agent/reviewer.go:42`、`planner.go:35`、`evaluator.go:39` 注释自称「config/roles/xxx.md 缺失时的回退」—— 实际根本没有读取逻辑，角色提示 100% 是 Go 硬编码 const（`reviewerSystemPrompt`/`plannerSystemPrompt`/`judgeSystemPrompt`）。
- 「思想/哲学」UI 功能已被移除（commit 33c4583b 删除 PhilosophyEnabled 等字段），`config/philosophy/*.txt`（道德经/孙子兵法等）随之成为无消费者资产。
- 处置建议：**二选一** —— ① 实现磁盘优先 loader（reviewer/planner/judge 提示支持 `config/roles/<name>.md` 覆盖，对齐插件「内容在磁盘」理念）；② 从发布清单移除死资产。

**[C2｜P1] 角色提示不可由插件/磁盘覆盖（与「内容资产插件化」相悖）**
- `Reviewer/Planner/Evaluator` 的 `SystemPrompt` 字段注释称「宿主可从 config 加载覆盖（非硬编码）」，但没有任何代码注入磁盘/插件内容；Loop 构造处直接 `DefaultReviewerPrompt()`（`internal/agent/loop.go:512`）。建议：支持 `ctx.systemPrompt.section(name='deployment:reviewer')` 或磁盘文件覆盖。

**[C3｜P2] 内置模型兜底硬编码在 Go**
- `internal/core/models.go:31` `defaultModels` 在 Go 硬编码 6 家服务商；models.json 缺失/损坏时兜底。属合理 fallback，但与「配置来源收敛到 ai-presets.json」的新原则并存，建议明确兜底策略或改从模板文件读。

**[C4｜P2] 前端内置文档与实现漂移**
- `plugins-src/ui-app/src/docs/api-docs.md` 仍描述「八、指令与思想」；`docs/` 目录只有 plugin-development / go-core-capabilities 两篇，**插件体系缺一份「现状 vs 设计」的版本化文档**（本报告可作起点）。

---

## 4. 前端面板（Frontend Panels）

### 4.1 已插件化的部分（优秀）
- 壳 = 纯骨架（`web/index.html` + `plugins-src/ui-app/src/ShellApp.vue`：8 槽位挂载点 + 壳级逃生口 PluginPanel）。
- 7 区域磁盘插件：ui-titlebar / ui-activitybar / ui-sidebar / ui-editor / ui-right-panel / ui-statusbar / ui-modals（`build-ui.mjs` 构建 IIFE bundle 到各插件 assets/）。
- 面板一体化：git-api（Git 面板）、marketplace（市场面板）、agent-teams（团队面板）、ui-quick-exec（快速执行）、ui-statusbar-conn（连接状态）。
- 槽位系统：single（替换型）+ list（叠加型）双机制；`window.__SLOT_REGISTRY` 跨 bundle 共享。
- 配置面板纯 schema 驱动（SettingsModal 无内置 tab，全部来自 `ctx.registerSettings`）。

### 4.2 缺口

**[F1｜P1] 主题（Theme）未插件化且选项与实现不一致**
- 4 套主题 CSS（dark/light/warm/night）硬编码在壳 `web/index.html:19-174`（及 `cmd/companion/web-ui/dist/index.html`），`ui-state.js` 也硬编码 4 主题类名。
- 但 `.pair/plugins/ui-appearance/index.js` 只注册 `options: ['dark', 'light']` → **warm/night 主题无法从设置面板选择**（选项与 CSS 漂移）。
- 建议：主题定义随 ui-appearance 插件提供（CSS 变量 + 选项同源），或至少选项补全 4 项。

**[F2｜P2] 壳层残留死代码**
- `router.js`（vue-router）未被 `main.js` 引用，PlanView/TaskBoard 路由不可达。
- 组件 `OutputPanel.vue` / `SubAgentBlock.vue` / `TasksPanel.vue` 零引用（SubAgentBlock 已被 RightPanel 内联时间线取代）。
- `ui-state.js.bak` 残留 13KB 备份文件。

**[F3｜P2] 前端产物三处堆积**
- `web/`（index-CrzpNXyK.js）、`.pair/assets/runtime/web/`（index-BY0sezMm.js）、`cmd/companion/web-ui/dist/assets/`（14 个 index-*.js 旧产物）三份并存；dist 目录历史 bundle 未清理（525-694KB × 14）。

**[F4｜P2] 桌面端 UI 槽位未装配**（与 L1 同根，见维度 2）。

---

## 5. 服务集成（Service Integration）

### 5.1 已插件化的部分（优秀）
- 内核路由表（`internal/agent/kernel_api.go`）+ core-api 插件挂载（`ctx.kernel.install`，约 82 条接口清单在 `.pair/plugins/core-api/index.js`）。
- fs-api / git-api / web-api 插件：接口实现搬进插件（ctx.webServer/ctx.bash），内核只留 fs.image（字节流形态不匹配，文档明确）。
- 市场全插件化：marketplace 插件（搜索/安装/卸载/市场源 ctx.market.register；npm/GitHub 实时搜索）。
- MCP：user/project 两级 mcp.json（`internal/agent/mcp_config.go`）+ ctx.mcp 服务。
- Provider 参数装配器插件化（`ctx.providerFactory.register`，agentloop 插件承载）；Loop 工厂插件化（`ctx.loopFactory`）。
- 事件总线 + client 事件桥（`ui:`/`client:` 前缀 → 浏览器轮询）；ext 路由/SSE/WS 插件化。

### 5.2 缺口

**[S1｜P1] LLM Provider 实现不可插拔（只有参数可装配）**
- `internal/provider`（移植的多 Provider 抽象）生产零导入；生产唯一实现是 `OpenAIProvider`（`internal/agent/llm.go:73`）。
- `ProviderFactory`（`provider_factory.go`）只做**参数**覆盖（温度/模型/Key），Provider 实现本身（Chat/Name）无插件注册面 → 新协议（如 Anthropic 原生、本地推理）必须改 Go 内核。建议：对齐 harness 增加 `ctx.provider.register`（实现级插件槽位）。

**[S2｜P2] web_server.go 遗留 ~25 个未注册 handler（能力层死代码）**
- `cmd/companion/web_server.go` 仍定义 `handleFSDrives/handleFSList/handleFSRead/handleFSWrite/handleFSRename/handleFSDelete/handleFSMkdir/handleFSSearch/handleFSFileInfo/handleFSHex`、`handleGit*`（17 个）等，但 `kernel_register.go` 只注册了 `fs.image`，git 组已整体注释（由 git-api 插件接管）→ 这些 Go handler 是「已插件化但未删除」的死实现，保留双份实现增加漂移风险。
- 建议：从 web_server.go 删除被插件接管的 handler（保留内核表入口即可），或标注 deprecated。

**[S3｜P2] hook / jobs / permission / provider / vterm 包未接线**（与 L2 同根，见维度 2）。

**[S4｜P2] desktop 与 web 的 handler/loop 装配双份**（`desktopbridge.buildDesktopLoopOpts` vs `web_server.buildWebLoopOpts`，与 L3 同根）。

---

## 6. 配置存储（Config Storage）

### 6.1 已插件化的部分
- settings.json 扁平结构 + `pluginSettings` 命名空间（`internal/core/settings.go:53`）。
- 配置 schema 注册（`ctx.registerSettings` → `core.PluginSettingSchemas`，`internal/core/settings_registry.go`），前端设置面板纯 schema 驱动；`binding` 字段把插件字段绑到 AppSettings 顶层。
- AI 配置来源收敛：ai-presets.json 唯一来源，settings 只存 preset 名（`internal/core/ai_presets.go`）。
- 配置面板特殊类型：provider-manager / preset-manager / project（instructions）由组件承载。

### 6.2 缺口

**[G1｜P1] 审核配置双存储分叉（tools.json vs settings.json）**
- 工作区 `.pair/tools.json` 的 `WorkspaceToolConfig`（reviewMode/reviewBlacklist/reviewWhitelist，`internal/agent/toolconfig.go`）与 settings.json 顶层同名字段（`internal/core/settings.go:35-37`）并存。
- 读取路径：`LoadWorkspaceReviewConfig` 读 tools.json；UI 设置面板经 `/api/settings` 写 settings.json 顶层 → **两处存储互相不知道对方**，改一处另一处不生效，属配置存储未收敛。
- 建议：审核配置统一到一处（工具集/pluginSettings 或 settings 顶层），tools.json 作为迁移遗留删除。

**[G2｜P2] AppSettings 顶层字段仍是 Go 结构体硬编码**
- 新核心配置字段需改 Go struct + settings.template.json + 前端三处同步；`settings.template.json` 缺 `workspaceFolderLists/pluginSettings/modelParams` 等实际字段（模板与 struct 漂移）。
- 插件只能写 pluginSettings 命名空间或 binding 已有字段，**无法声明全新的顶层字段** —— 与「配置无内置全靠插件」的声明不完全一致（插件字段必须落在 pluginSettings 内，顶层字段仍内核独占）。

**[G3｜P2] APIKey 明文存储**
- `config/ai-presets.json` 与 settings 的 apiKey 均为明文（非插件化问题，但属配置存储安全缺口，建议标记或加密/系统钥匙串）。

**[G4｜P2] 配置模板未纳入构建校验**
- packager 用 `models.template.json/settings.template.json/mcp.template.json` 复制为运行时文件，但模板与 struct 无一致性校验（G2 同根）。

---

## 7. 优先级需求清单（供 t2 实施）

| ID | 优先级 | 维度 | 缺口 | 证据文件 | 建议动作 |
|---|---|---|---|---|---|
| L1 | **P0** | 插件宿主 | 桌面端无插件宿主/无工具/无 UI 装配 | `internal/desktopbridge/desktopbridge.go`、`cmd/desktop/main.go` | 桌面端接入 PluginHost（LoadGlobalPlugins + LoadAllToolsets + MergePluginTools + SetPluginHost），或显式废弃桌面端并清理 |
| C1 | **P0** | 内容资产 | roles/philosophy 死内容（发布但不读） | `packager.json`、`internal/agent/reviewer.go:42` | 实现磁盘优先角色 loader，或移出发布清单 |
| T1 | **P0** | 工具/工具集 | 7 组约 20 个孤儿工具零装配 | `internal/agent/{asset_tools,bridge_tools,entryconfig,evolution_tools,progress_checker,resource_tools,snapshot}.go` | 迁移为磁盘插件（tool-* 生成器）或从代码库删除 |
| T2 | P1 | 工具/工具集 | 插件描述引用已删除的 image_analyze/image_ocr | `tool-screenshot/index.js:11`、`tool-web-debug/index.js:11` | 修正描述（引 submit_image），同步测试清单 |
| L2 | P1 | 插件宿主 | 12 个 internal 包零导入（hook/jobs/permission/provider/vterm…） | `internal/hook/*`、`internal/jobs/*` 等 | hook 系统接入 Loop（插件可注册钩子）或整体删除；其余包清理 |
| S1 | P1 | 服务集成 | Provider 实现不可插拔 | `internal/agent/llm.go:73`、`provider_factory.go` | 增加 `ctx.provider.register` 实现级插件槽位 |
| G1 | P1 | 配置存储 | 审核配置 tools.json vs settings.json 双源 | `internal/agent/toolconfig.go`、`internal/core/settings.go:35` | 统一到单一存储 |
| F1 | P1 | 前端面板 | 主题 4 套 CSS 硬编码壳，选项只有 2 项 | `web/index.html:19-174`、`ui-appearance/index.js` | 主题内容插件化或选项对齐 |
| C2 | P1 | 内容资产 | 角色提示不可覆盖 | `internal/agent/loop.go:512`、`reviewer.go` | 支持磁盘/插件覆盖 reviewer/planner/judge 提示 |
| T3 | P2 | 工具 | 工具描述文言文精简在 Go 硬编码 | `internal/agent/tools_concise.go` | 描述覆盖迁插件（或标注为宿主框架能力） |
| L3/L4 | P2 | 生命周期 | AgentBase 未用 + 初始化三份 + 审批遗留字段 | `internal/agent/agent.go:85`、`plugin.go:469` | 收敛初始化路径，清理废弃字段 |
| S2 | P2 | 服务集成 | web_server.go 遗留 25+ 未注册 handler | `cmd/companion/web_server.go:477-3729` | 删除被插件接管的死 handler |
| F2/F3/F4 | P2 | 前端 | 死组件/router/产物堆积/桌面槽位 | `router.js`、`OutputPanel.vue` 等 | 清理 + 产物收敛 |
| G2/G3/G4 | P2 | 配置存储 | struct/模板漂移 + APIKey 明文 + 无校验 | `internal/core/settings.go`、`settings.template.json` | 模板自动生成/校验，Key 加密 |

---

## 8. 方法论与覆盖范围

- **审查方法**：全仓库静态扫描（grep/glob/read + git 历史追踪），覆盖 Go 内核（internal/、cmd/）、磁盘插件（.pair/plugins/）、插件源码（plugins-src/）、前端（plugins-src/ui-app/、web/）、配置（config/）、打包（packager.json）、记忆/文档（.pair/memory、docs）。
- **判定准则**：① 实现是否在 Go 内核（能力层）而挂载/内容在插件；② 插件装卸是否随生命周期生灭；③ 是否有磁盘可覆盖形态（不重编译）；④ 是否存在「实现存在但零调用/零消费」的死资产；⑤ 描述/文档与实现是否漂移。
- **未覆盖**：运行时行为验证（未启动 IDE 实测）、npm 插件市场端到端、桌面端实机验证（建议由 t3 验证任务补冒烟）。
