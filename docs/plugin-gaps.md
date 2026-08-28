# 插件化缺口实施记录（t2 实施 · 2026-09）

> 本文件对应 `docs/pluginization-gap-analysis.md`（t1 审查报告）的缺口清单，
> 记录 t2 实施结果：每个缺口标注「已实现」+ 改动文件清单；遗留项标注原因。
> 验证：`go build ./...`、`go vet ./...` 全通过；`go test ./internal/agent/ -count=1 -timeout 10m`
> 除环境依赖测试外全通过（见文末「环境依赖测试说明」）。

---

## 实施结果（t5 集成收尾 · 2026-09）

### 已闭环缺口清单（t2 实施 + t4 审查 pass + t5 集成）

| 缺口 | 优先级 | 状态 | 集成验证 |
|---|---|---|---|
| L1 桌面端无插件宿主 | P0 | ✅ 已实现 | pair.exe 冒烟启动装载 43 个插件、工具集 44 个（18 插件），无 FATAL/panic |
| C1 roles/philosophy 死内容 | P0 | ✅ 已实现 | 角色 loader 随包生效（config/roles 保留发布、philosophy 移出 packager） |
| C2 角色提示不可覆盖 | P1 | ✅ 已实现（并入 C1） | `TestRolePromptDiskOverride` 通过 |
| T1 7 组孤儿工具零装配 | P0 | ✅ 已实现 | 冒烟启动工具集含新插件；`TestArchiveHostLegacyToolsAndPluginLoad` 通过 |
| T2 插件描述漂移 | P1 | ✅ 已实现 | 插件 JS 重生成，无 image_analyze/image_ocr 残留 |
| L2 hook 系统未接线 | P1 | ✅ 已实现 | `TestLoopHookConfigBlock/PluginBlock` 通过 |
| S1 Provider 实现不可插拔 | P1 | ✅ 已实现 | `TestCreateProviderRouting/JSProviderImplBridge` 通过 |
| G1 审核配置双存储分叉 | P1 | ✅ 已实现 | `TestReviewConfigSingleSourceMigration` 通过 |
| F1 主题选项与 CSS 漂移 | P1 | ✅ 已实现 | ui-appearance 注册 4 主题 |

### 集成构建与冒烟（t5）

- **后端完整构建**：`CGO_ENABLED=1 go build -o pair.exe ./cmd/companion` ✅（pair.exe 74.7MB）。
- **启动冒烟**：`WEB_PORT=9322 ./pair.exe` 启动 → 进程持续运行、`/api/health` 返回 200、启动日志无 FATAL/panic、全局插件宿主 43 个插件、工具集 44 个（18 插件，含 t2 新增 7 个 tool-*）→ 验证后立即终止，端口释放。
- **前端构建**：⚠️ **沙箱环境阻断**——`npm run build`（入口 `cmd/companion/web-ui`（兼容 package.json，2026-08-29 新增）→ `scripts/build-ui.mjs` + `plugins-src/ui-app` vite 壳 + `scripts/sync-web-dist.mjs` 同步 dist）在 DSH 沙箱内无法执行：vite/esbuild 服务进程 spawn（pipe stdio）被 DSH 沙箱设计边界拦截（`spawn EPERM`，Node `child_process.spawn` 的 pipe/overlapped stdio 均被禁；本会话无批准升级通道）。**前端构建需在沙箱外执行**（同一命令即可），dist 产物为既有构建结果。

### 集成收尾处理（2026-08-29）

- **✅ 基线快照与提交**：团队既有 WIP + t2 实施已提交（`07faf08e` 集成提交、`0c7f5ff0` docs 收尾）；集成收尾另提交 `18eb91b2`（敏感配置移出跟踪/运行时状态忽略/杂物清理）与 `0bde4354`（web-ui 构建兼容入口路径修正）；提交后 `git status` 完全干净。
- **⚠️ 敏感信息处置（已上报 captain）**：`config/ai-presets.json` 与 `cmd/desktop/config/settings.json` 含**真实 API key**（sk_tr_…/sk-56cf…）且早于本次已入 git 历史（未 push，本地 master ahead 1062）。本次收尾已将 4 个本地配置（ai-presets.json、cmd/desktop/config/*、dev/desktop_probe/config/*）`git rm --cached` 移出跟踪并加入 `.gitignore`（防再犯）；**建议轮换上述 key**，历史清除（filter-repo 或接受现状）由 captain 决策。
- **✅ M1/M2 修复**：见「遗留项与后续建议」第 1 条与 docs/plugin-verification.md「处理状态」。
- **✅ generate_commit_message 失配修复**：tool-system 插件移除该死工具（宿主无实现、零消费方）。
- **⚠️ 环境依赖测试复跑结论**：TestToolLandingSpotCheck / TestToolProjectInfoJSNative / TestToolVerifyJSNative 复跑仍失败——`ctx.binary.exec: 插件二进制不存在`（tool-project-info/tool-verify 插件仍 binary 形态；build-plugin-bins.bat 已废弃（插件全 JS 化，禁止再跑）；内嵌内核未覆盖 project-info/verify 组）。属**团队既有 WIP 失配**（非 t2/t5 引入），复跑/修复路径：将 tool-project-info/tool-verify/tool-memory 插件 JS 原生迁移（对齐 tool-core 模式），或在内嵌内核（`internal/agent/embedded_tools.go`）补齐对应注册函数；建议后续专项任务处理。
- **✅ 已闭环（2026-09 Round2 R2-1/R2-2）**：上条 5 个 binary 插件（tool-git/tool-memory/tool-project-info/tool-verify/tool-bug）已全部 JS 原生迁移，`TestToolGitJSNative/Memory/ProjectInfo/Verify/Bug` 全部通过；office csv_read/word_read 陈旧断言已修正；tool_landing_test.go 模式表（原计划态 native）现为实际态。详见 docs/plugin-round2-plan.md §6.1-6.2。

### 遗留项与后续建议

1. **t4 审查 findings（verdict=pass，2 medium + 5 low）**：M1（roles 漂移）与 M2（钩子信任门）**✅ 已修复（2026-08-29 集成收尾）**——reviewer.md 已同步 + 新增漂移守卫测试 `TestRolePromptDiskDefaultSync`；钩子装载默认不信任项目钩子（`PAIRCODE_TRUST_PROJECT_HOOKS=1` 显式信任）+ 新增 `TestLoopHookProjectTrustGate`；详见 docs/plugin-verification.md「处理状态」。L1–L5 低危项 **✅ 已处置（2026-09 Round2 R2-3）**——L1 bridge 描述（4 处字符串）、L2 hook turn 透传（loopHookCurrentTurn）、L3 G1 迁移语义统一、L5 config/philosophy 目录删除；L4 RegisterProviderImpl 还原顺序按 t1 建议 defer/文档化（单插件场景无影响）。详见 docs/plugin-round2-plan.md §6.3。
2. **11 个 internal 死包**：✅ 已处置 7 个（2026-09 Round2 R2-5：event/model/store/summary/verify 空目录 + agenttools/codetypes 薄壳，grep 零导入）；`jobs/permission/provider/vterm` 4 个保留**待产品确认**（有实现有测试，可作参考移植源）。
3. **S2/F2/F3/L3/L4/G2/G3/G4（P2）**：S2 **✅ 已处置**（2026-09 Round2 R2-4：web_server.go 未注册 handler 1155 行删除，fs-api/git-api 插件已接管，/api/fs/image 留内核）；F2/F3 前端专项、G2/G4 产品决策、L3(旧)/L4(旧) 架构级统一留后续专项。
4. **前端构建流水线**：沙箱外执行 `cd cmd/companion/web-ui && npm run build`（兼容入口，含 9 个 UI 区域插件 + vite 壳 + dist 同步；路径已修正）；或直接 `cd plugins-src/ui-app && npm run build`。
5. **验证命令目录**：`cmd/companion/web-ui` 现已有兼容 package.json（构建入口有效），vite 壳产物输出 `.pair/assets/runtime/web`，`scripts/sync-web-dist.mjs` 同步 embed 兜底 dist。
6. **Round2 新增遗留（2026-09）**：goal/subagent/workflow 机制（需宿主 Loop 支持，P1/P2）；agent-teams memberProvider fork 与 slash 命令（宿主无对应能力面）；ask_user 多问题前端渲染（前端专项）；Go 旧名工具实现保留为测试/归档专用（89 处测试引用，生产零注册）；agent-teams profiles 已同步（default/captain-planning）+ reasoning_effort 透传已实施。完整清单与风险记录见 docs/plugin-round2-plan.md §7。
7. **Round2 集成收尾（t5 · 2026-09）**：t4 审查 5 个 findings 全部处置（F1/F2/F4 修复、F3 文档化、F5 文档修正，见 docs/plugin-round2-plan.md §7.3）；`CGO_ENABLED=1 go build -o pair.exe ./cmd/companion` ✅；冒烟（WEB_PORT=9323）43 插件/17 工具集条目/探活 200/零 FATAL ✅；前端构建沙箱 EPERM（无前端改动，队长全权限通道可复跑 `cd cmd/companion/web-ui && npm run build`）；独立验证报告 docs/plugin-round2-verification.md（t3 + t5 §8）结论 PASS；Round2 全部改动已提交，`git status` 仅预期 git-ignored 产物。

---

## 高优先级（P0）——全部已实现

### [L1｜P0] 桌面端无插件宿主 → **已实现**
- **改动**：`internal/desktopbridge/desktopbridge.go`
  - `Init` 新增 `initPluginHostDesktop()`：NewPluginHost + RegisterCordisTools + RegisterToolsetTools + MigrateLegacyBuiltinJSON + LoadAllToolsets（工作区工具集 + 全局磁盘插件）+ LoadCordisPatch + `handler.SetPluginHost` + `agent.SetGlobalPluginHost`（与 web 端 startWebUI 同装配语义，`sync.Once` 幂等）。
  - `buildDesktopLoopOpts` 新增插件合并：RegisterCordisTools / RegisterToolsetTools / ApplyHarnessToolFilter（插件豁免回调）/ ApplyToolsetBuiltinState / MergePluginTools —— 桌面会话 Agent 不再只有框架工具，`/api/plugins` 与 7 区域 UI 槽位随宿主就绪。
- **验证**：`go build ./...` 通过（桌面端实机冒烟由 t3 补）。

### [C1｜P0] roles/philosophy 死内容 → **已实现**
- **改动**：`internal/agent/role_prompts.go`（新）、`internal/agent/reviewer.go`、`internal/agent/planner.go`、`internal/agent/evaluator.go`、`packager.json`
  - 实现磁盘优先角色 loader：`config/roles/<name>.md` 存在且非空 → 覆盖内置提示；缺失/空白 → 回退 Go 内置。`DefaultReviewerPrompt/DefaultPlannerPrompt/DefaultJudgePrompt` 全部改走 loader（C2 一并闭环）；planner/evaluator 内部兜底点同步改用 Default*Prompt。
  - `config/philosophy/*.txt`（思想/哲学功能已于 commit 33c4583b 删除）从 `packager.json` dist.include 移除——不再随包发布死资产；roles 保留（现在运行时读取）。

### [T1｜P0] 7 组孤儿工具零装配 → **已实现**
- **改动**：`internal/agent/tool_plugin_gen.go`（genToolGroups 新增 7 组）、`internal/agent/legacy_host_tools.go`（新，ArchiveHostLegacyTools）、`internal/agent/plugin.go`（NewPluginHost 调用存档）、`.pair/plugins/tool-{asset,bridge,entryconfig,evolution,progress,resource,snapshot}/`（生成器产物，7 个插件包）、`.pair/toolsets/default.json`（加入工作区工具集，工具对 agent 可用）、`internal/agent/tool_landing_test.go`（模式表 + hostNames 刷新）
  - 处置：Go 实现经 `ArchiveHostLegacyTools(root)` 存档为宿主能力（hostExecutors，seam「能力在宿主」），磁盘插件声明工具 api、execute 走 `ctx.hostTool.exec` 复用宿主实现（seam「编排在插件」）；**不注册进任何 Registry**——agent 可见面由插件决定，插件停用即工具消失。
  - 生成命令：`go run -tags toolsgen ./dev/tool_plugin_gen`（幂等）。
- **验证**：`TestArchiveHostLegacyToolsAndPluginLoad`（存档 + 插件装载 + hostTool 执行闭环）。

---

## 高优先级（P1）——全部已实现

### [T2｜P1] 插件描述引用已删除的 image_analyze/image_ocr → **已实现**
- **改动**：`.pair/plugins/tool-screenshot/index.js`、`.pair/plugins/tool-web-debug/index.js`（重新生成，Go 源描述已更新为「多模态模型分析」）、`.pair/toolsets/default.json`（内嵌 tool-web-debug code 同步）、`internal/agent/toolset_visibility_test.go`（示例工具名 image_ocr → submit_image）
- 说明：Go 侧描述早已不含 image_analyze/image_ocr（vision 工具删除时已同步），插件 JS 是陈旧生成产物——重跑生成器即修复。

### [L2｜P1] 12 个 internal 包零导入（hook 未接线）→ **已实现（hook 系统）**
- **改动**：`internal/agent/loop_hooks.go`（新）、`internal/agent/tools.go`（Registry.Execute 接线 PreToolUse/PostToolUse）、`internal/agent/loop.go`（Run 接线 UserPromptSubmit + Stop defer）、`internal/agent/jsplugin.go`（`ctx.hooks.register` JS 插件钩子桥）、`cmd/companion/web_server.go` + `internal/desktopbridge/desktopbridge.go`（生产入口 `agent.InitLoopHooks()`）
  - 配置钩子：`.pair/settings.json`（项目）+ `~/.pair/settings.json`（全局），经 internal/hook 装载；PreToolUse 阻塞语义（exit 2/超时 → 工具拦截并回灌反馈）。
  - 插件钩子：`RegisterLoopHook`（Go 插件）/ `ctx.hooks.register(event, fn)`（JS 插件），与配置钩子同事件表，卸载自动注销。
  - 接线点覆盖 Go 循环与 JS 循环（agentloop）两条执行路径（Registry.Execute 单一闸口）。
  - 安全语义：未调用 InitLoopHooks 且无插件钩子时全部 no-op（测试/无配置零行为变化）。
- **验证**：`TestLoopHookConfigBlock`、`TestLoopHookPluginBlock`。
- **遗留（书面理由）**：`internal/jobs`、`internal/permission`、`internal/provider`、`internal/vterm`、`internal/agenttools`、`internal/codetypes`、`internal/event`、`internal/store`、`internal/model`、`internal/summary`、`internal/verify` 仍为零导入死包——**超出本轮范围**（各自对应独立子系统移植决策，删除需产品确认），建议后续单独清理任务处理；`scripts/relocate_imports.go` 一并保留。

### [S1｜P1] Provider 实现不可插拔 → **已实现**
- **改动**：`internal/agent/provider_impls.go`（新，RegisterProviderImpl/CreateProvider 实现注册表）、`internal/agent/jsplugin_providerimpl.go`（新，ctx.provider.register JS 实现桥 + ctx.provider.http 宿主 HTTP 通道）、`internal/agent/jsplugin.go`（ctx.provider 挂载）、`cmd/companion/web_server.go`、`cmd/companion/subagent_spawn.go`、`internal/desktopbridge/desktopbridge.go`、`internal/server/handler/stub.go`、`internal/agent/llm_analyze.go`（全部生产构造点路由 CreateProvider）
  - 契约：插件注册 `{ chat(params, messages, tools) }`（非流式，Go 侧一次性 Done chunk）；新协议（Anthropic 原生/本地推理等）100% 插件内实现，无需改 Go 内核；未注册服务商回退 OpenAI 兼容（行为不变）；插件卸载自动还原。
- **验证**：`TestCreateProviderRouting`、`TestJSProviderImplBridge`。

### [G1｜P1] 审核配置双存储分叉 → **已实现**
- **改动**：`internal/agent/toolconfig.go`
  - 单一来源 = settings.json 顶层（core.Settings，与 UI 设置面板同存储）；`.pair/tools.json` 仅作一次性迁移遗留：读取时合并进 settings 并删除（含未知字段时仅删审核字段保留文件）；Save 路径同样写 settings 并清理遗留。
- **验证**：`TestReviewConfigSingleSourceMigration`。

### [F1｜P1] 主题选项与 CSS 漂移 → **已实现**
- **改动**：`.pair/plugins/ui-appearance/index.js`——选项补齐 `['dark', 'light', 'warm', 'night']`（与壳 4 套主题 CSS `.theme-*` 一一对应）；「主题内容插件化（CSS 变量随插件下发）」列为后续演进（壳 CSS 迁移时保持同源）。

### [C2｜P1] 角色提示不可覆盖 → **已实现**（并入 C1，见上）

---

## 中优先级（P2）——记录/理由

| ID | 缺口 | 状态 | 说明 |
|---|---|---|---|
| T3 | 工具描述文言文精简硬编码（tools_concise.go） | **保留（宿主框架能力）** | ApplyConciseToolDescriptions 属 harness 对齐的宿主框架行为（与 ApplyHarnessToolFilter 同层），t1 报告亦认可「或标注为宿主框架能力」；本轮不迁插件。 |
| L3 | AgentBase 未用 + 初始化三份 | **部分（L1 已收敛桌面/ web 两处语义）** | AgentBase 生产未用属历史演进产物（web/desktop 各有宿主上下文），统一改造风险高，超出本轮范围；桌面端装配已与 web 端对齐（L1）。 |
| L4 | 审批遗留字段 | **遗留** | 纯兼容死代码清理，无行为影响；列入后续清理。 |
| S2 | web_server.go 遗留 ~25 个未注册 handler | **遗留** | 删除被插件接管的死 handler 涉及大量删除与回归验证，超出本轮范围；建议后续专项清理。 |
| F2/F3 | 前端死组件/router/产物堆积 | **遗留（超范围）** | 属前端 web/ 构建产物，不在本轮 in-scope（internal/agent、handler、cmd/companion、pkg、docs、desktopbridge）；建议后续前端专项。 |
| G2/G3/G4 | 配置模板漂移/APIKey 明文/无校验 | **遗留** | 配置存储安全与模板工程化，超出本轮范围；APIKey 明文已在 packager stripSecrets 有发布侧脱敏。 |

---

## 未实施项汇总（需 captain 知晓）

1. **12 个 internal 死包中的 11 个**（jobs/permission/provider/vterm/agenttools/codetypes/event/store/model/summary/verify）——各自对应独立子系统移植决策，删除需产品确认（L2 的 hook 系统已闭环，其余为「清理决策」类）。
2. **S2 / F2 / F3 / L3 / L4 / G2 / G3 / G4**——P2 级，多数超出本轮 in-scope 或需专项回归，见上表理由。
3. **既有失配（非 t2 引入，团队 WIP 遗留）：`tool-system` 插件声明 `generate_commit_message`（SystemTool）但宿主无对应 Go 实现/存档**——hostTool 执行会报「宿主执行器不存在」。**✅ 已修复（t2 集成收尾 2026-08-29）**：从 `.pair/plugins/tool-system/index.js` 移除该死工具（与生成器白名单对齐），`tool_landing_test.go` 的 `knownSystemToolDrift` 豁免同步移除。

## 新增验证（internal/agent/pluginization_fixes_test.go）

- `TestRolePromptDiskOverride` — C1/C2 磁盘优先 + 回退 + 空白文件语义
- `TestArchiveHostLegacyToolsAndPluginLoad` — T1 存档/插件接管/hostTool 执行/7 组包齐全
- `TestCreateProviderRouting` / `TestJSProviderImplBridge` — S1 Go/JS 实现路由 + 卸载还原
- `TestReviewConfigSingleSourceMigration` — G1 迁移 + 单源读写
- `TestLoopHookConfigBlock` / `TestLoopHookPluginBlock` — L2 配置钩子拦截 + 插件钩子拦截/还原

## 环境依赖测试说明（t3 验证参考）

`go test ./internal/agent/` 在 DSH 沙箱环境下以下测试无法通过，均为**环境依赖**（
与本次改动无关，失败机理在改动前即存在）：
1. **Git bash 受限**（沙箱禁止 bash 创建信号管道，`couldn't create signal pipe, Win32 error 5`）：
   `TestToolRunCommand`、`TestRunShellWithBash/Backtick`、`TestRunBackground`、`TestRunCommandInLoop`、
   `TestDetectBash`、`TestBGCrossRegistry`、`TestHarnessAlias_GlobGrepBash`、`TestJSPluginCtxBashLogger`、
   `TestRunCode_Go`、`TestRunCodeNested`、`TestToolGitJSNative*`、`TestToolMemoryJSNative`、
   `TestToolHarnessJSNative`、`TestToolCoreJSNative` 等（bash 冒烟失败）。
2. **插件二进制未构建**（`.pair/plugins/tool-*/bin/*.exe` 为构建脚本产物，本检出树未编译，
   内嵌内核未覆盖 memory/verify/project-info 组——团队 WIP 测试依赖）：
   `TestToolProjectInfoJSNative`、`TestToolVerifyJSNative`、`TestToolOfficeJSNative`、
   `TestToolBugJSNative`、`TestToolLandingSpotCheck`。
3. **Chromium 下载受限**（沙箱禁止写 AppData\Roaming\rod）：`TestWebDebug*`。

排除上述环境依赖项后：`go test ./internal/agent/ -count=1 -timeout 10m` **全通过**（ok，~17s）。
