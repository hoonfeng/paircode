# 插件化改进独立验证报告（t3 验证 · 2026-09）

> 验证对象：t2 实施的「一切皆插件」高优先级缺口闭环（P0×3 + P1×5，见
> `docs/plugin-gaps.md`）。工作目录 `<仓库根>`（Go 1.26.4，模块
> github.com/hoonfeng/paircode，主程序 cmd/companion）。
> 验证方式：独立命令运行 + 文件 mtime 时间窗归因 + 源码抽查 + 启动冒烟。
> 本验证**未改动任何实现代码**；唯一新增产物为本报告与临时日志目录 `.verify-tmp/`
>（验证日志留存作证据）。

---

## 0. 结论（TL;DR）

| 验收项 | 结果 |
|---|---|
| 1. verify 命令逐条运行并记录退出码，e2e 失败区分环境依赖/真实回归 | ✅ PASS |
| 2. git status/diff 改动范围核对（全部在 t2 inScope 内，outOfScope 零改动） | ✅ PASS（2 项文档化例外，见 §3） |
| 3. pair.exe 构建（CGO_ENABLED=1）+ 启动冒烟（临时端口，无 panic） | ✅ PASS |
| 4. 「插件优先」原则抽查（新增工具/UI 入口走插件/工具集机制） | ✅ PASS |
| 5. 输出本验证报告，不改动实现代码 | ✅ PASS |

**总评：t2 实现可构建、可测试（环境依赖项外全通过）、范围合规、启动健康，
「插件优先」原则落地属实，无 blocker/high 级问题。** 遗留 2 项 low 级范围备注
（packager.json 与 .pair/plugins/* 不在 inScope 字面清单内，但为缺口闭环所必需
且已在 docs/plugin-gaps.md 记录），建议 captain/审查知悉即可。

---

## 1. 验证环境说明（重要）

受限沙箱（workspace-write 策略）对验证命令有两项环境约束，**与 t2 代码无关**：

1. `%AppData%\go\env` 全局设置了 `CGO_ENABLED=0`（且 GOPATH 指向他处）。而
   `wb-ui/platform/graphics`（依赖 goskia 的 cgo 绑定）在 CGO=0 下无法编译
   （`undefined: skia.Surface`）。**以 CGO_ENABLED=1（项目文档化构建模式）运行
   后 `go build ./...`、`go vet ./...` 均 exit 0**。
2. 沙箱禁止写入 `<外部安装路径>\AppData\Local\go-build` 与
   `F:\MyGolangPrograms\pkg\mod\cache`（模块 stat cache 刷新被拒）。规避：
   `GOCACHE=$env:TEMP\dsh-gocache` + `GOPROXY=off` + `GOWORK=off`。模块缓存
   已完整（无网络访问，GOPROXY=off 仅用本地缓存）；`go env` 的
   `upload.token: Access is denied` 为无害噪音。

结论：构建/测试命令在项目标准配置（CGO_ENABLED=1）下全部通过；CGO=0 下的失败
为环境配置所致，非 t2 回归。

---

## 2. 命令结果表

| # | 命令 | 退出码 | 关键输出 | 判定 |
|---|---|---|---|---|
| 1 | `go build ./...`（CGO_ENABLED=1） | 0 | 无编译错误 | ✅ PASS |
| 2 | `go vet ./...`（CGO_ENABLED=1） | 0 | 无 vet 告警 | ✅ PASS |
| 3 | `go test ./internal/agent/ -count=1 -timeout 10m`（CGO_ENABLED=1） | 1 | 22 个失败**全部为环境依赖**（见 §2.1），无真实回归 | ✅ PASS* |
| 4 | `CGO_ENABLED=1 go build -o pair.exe ./cmd/companion` | 0 | pair.exe 74,668,054 B（v1.4.0） | ✅ PASS |
| 5 | 启动冒烟（`WEB_PORT=18090 .\pair.exe`） | 0（正常终止） | 见 §4 | ✅ PASS |
| 6 | 目标测试：9 个 t2 新增/相关测试 `-v` | 1 | 8 PASS + 1 环境依赖（TestToolLandingSpotCheck，插件二进制未构建） | ✅ PASS* |

\* 环境依赖失败项在改动前机理即存在（t2 文档「环境依赖测试说明」逐条对应），
详见下。

### 2.1 测试失败分类（22 项，全部环境依赖）

| 类别 | 失败测试 | 机理证据（日志原文摘录） |
|---|---|---|
| ① Git-bash 受限（沙箱禁止创建信号管道） | TestRunShellWithBash / TestRunShellBacktick / TestRunBackground / TestRunCommandInLoop / TestDetectBash / TestBGCrossRegistry / TestHarnessAlias_GlobGrepBash / TestJSPluginCtxBashLogger / TestRunCode_Go / TestRunCodeNested / TestToolRunCommand / TestToolGitJSNative / TestToolGitJSNativeMultiProject / TestToolMemoryJSNative / TestToolHarnessJSNative / TestToolCoreJSNative | `bash: *** fatal error - couldn't create signal pipe, Win32 error 5` |
| ② 插件二进制未构建（`.pair/plugins/tool-*/bin/*.exe` 为构建脚本产物，检出树未编译） | TestToolProjectInfoJSNative / TestToolVerifyJSNative / TestToolOfficeJSNative / TestToolBugJSNative / TestToolLandingSpotCheck | `GoError: ctx.binary.exec: 插件二进制不存在 ...\tool-*.exe（编译：go build -o ... ./plugins-src/plugins/<name>）` |
| ③ Chromium 下载受限（沙箱禁写 AppData\Roaming\rod） | TestWebDebugTimeoutMs | `启动 Chromium 失败: ... mkdir <外部安装路径>\AppData\Roaming\rod\browser\chromium-1321438: Access is denied.` |

与 `docs/plugin-gaps.md`「环境依赖测试说明」所列完全一致（含 CGO=0 与 CGO=1
两次全量运行的失败集一致）；**t2 改动路径的测试（loop/plugin/jsplugin/toolset/
hook/provider/role/toolconfig）全部通过**。目标运行证据：

```
=== RUN   TestRolePromptDiskOverride          --- PASS (0.01s)   [C1/C2]
=== RUN   TestArchiveHostLegacyToolsAndPluginLoad --- PASS (0.07s) [T1]
=== RUN   TestCreateProviderRouting           --- PASS (0.00s)   [S1]
=== RUN   TestJSProviderImplBridge            --- PASS (0.01s)   [S1]
=== RUN   TestReviewConfigSingleSourceMigration --- PASS (0.01s)  [G1]
=== RUN   TestLoopHookConfigBlock             --- PASS (0.05s)   [L2]
=== RUN   TestLoopHookPluginBlock             --- PASS (0.00s)   [L2]
=== RUN   TestToolLandingMatrix               --- PASS (0.30s)   [T1] 落地矩阵: 240 工具 binary:43 hostTool:135 mixed:26 native:36
=== RUN   TestToolLandingSpotCheck            --- FAIL (0.06s)   [② 环境依赖]
```

---

## 3. 范围核对表（git status/diff × mtime 时间窗归因）

基线判定：团队启动 2026-08-28 23:30，t1 完成 23:59，t2 窗口 = **2026-08-29
00:05–00:41（本地）**。工作树中另有大量**团队启动前**的 WIP 改动（批量 mtime
2026-08-27 23:50、8/28 22:59–23:00 等，如 internal/core、pkg/codegraph、
plugins-src/ui-app、config/*、go.mod、.pair/assets 前端产物、tool-vision-llm
删除等）——按 mtime 均不落在 t2 窗口，**不属于 t2 改动**（但属于未提交 WIP，
t5 集成时需注意）。

t2 窗口内改动的 33 个唯一跟踪路径：

| 路径 | 归属 | 判定 |
|---|---|---|
| internal/agent/（18 个：planner/evaluator/reviewer/tool_plugin_gen/plugin/tools/loop/jsplugin/llm_analyze/toolconfig/toolset_visibility_test 等 + 新增 role_prompts.go/legacy_host_tools.go/provider_impls.go/jsplugin_providerimpl.go/loop_hooks.go/pluginization_fixes_test.go/tool_landing_test.go） | L1/C1/C2/T1/L2/S1/G1 | ✅ inScope |
| internal/server/handler/stub.go | S1 | ✅ inScope |
| cmd/companion/web_server.go、subagent_spawn.go | L2/S1 | ✅ inScope |
| internal/desktopbridge/desktopbridge.go | L1（t1 明确标注桌面插件化缺口） | ✅ inScope（条件项成立） |
| docs/plugin-gaps.md、docs/pluginization-gap-analysis.md | 交付物 | ✅ inScope |
| packager.json | C1（移除 config/philosophy 死资产随包发布） | ⚠ 例外①：不在 inScope 字面清单，但为 C1 闭环所必需，已在 docs/plugin-gaps.md 记录；diff 中另有版本号/targets/onnx 拆分等疑似团队前 WIP hunk（无法与 t2 剥离，mtime 属 t2 窗口） |
| .pair/plugins/tool-{asset,bridge,entryconfig,evolution,progress,resource,snapshot}/（7 个新包）、tool-screenshot/index.js、tool-web-debug/index.js、ui-appearance/index.js | T1/T2/F1 的插件产物 | ⚠ 例外②：不在 inScope 字面清单，属「插件即产物」的闭环本体，已记录 |
| .pair/toolsets/default.json（git-ignored） | T1 工具集装配（运行时数据，git 不跟踪） | ✅ 不出现于 git status（git-ignored），与 t2 文档一致 |

**outOfScope 零改动结论：** pkg/、internal/core/、internal/hook/、internal/ui/、
cmd/desktop/、cmd/evaluator/、config/、dev/、plugins-src/、scripts/、go.mod、
.pair/assets/ 在 t2 窗口内均无写入。git 可见的 outOfScope 文件零改动成立
（packager.json 与 .pair/plugins/* 为上述两条文档化例外）。

**其他说明：** 根目录 `nul` 与 `7ea919d4-….png` 为团队启动前的既有杂物
（8/21），非 t2；`commitmsg_tool.go`/`loop_rejection.go` 删除与 desktopbridge
diff（移除 RegisterCommitMessageTool 调用）一致，`grep` 无悬挂引用，构建通过。

---

## 4. 启动冒烟结论

方式：`WEB_PORT=18090`（避开 9090/9097 既有实例；9090 实测无监听）运行
`pair.exe`，后台任务托管，观察启动日志后终止进程。

- 启动序列（`logs/paircode.log` + stderr 捕获）：
  ```
  [core] 端口通过 WEB_PORT 环境变量覆盖: 18090
  [main] PairCode IDE v1.4.0 启动中 (go1.26.4, windows/amd64)
  [js-plugin:dyn-16:core-api] 已装配内置接口 52/52
  [toolset] 已装载 44 个工具集（18 个插件）
  [WebUI] 全局插件宿主已初始化（43 个插件）
  [server] Web 服务已启动 (端口 18090)
  [main] 已启动，请打开 http://localhost:18090
  [WebUI] 代码知识图谱已就绪
  ```
- **无 panic、无 [FATAL]、无「未捕获的异常」**；进程持续存活，HTTP 探活：
  - `GET /` → 200（9538 B）
  - `GET /api/plugins` → 200，43 个插件，**包含全部 7 个新增包**
    （tool-asset/bridge/entryconfig/evolution/progress/resource/snapshot）
  - `GET /api/tools` → 200，含 `asset_list` 等孤儿组工具（entryconfig 组实为
    find_entry_points/find_config_files，snapshot 组实为 restore_snapshot/
    list_snapshots——按插件声明名注册）
- 冒烟后 `Stop-Process` 干净终止，**未产生任何跟踪文件改动**（git status 与
  冒烟前一致）。

---

## 5. 「插件优先」原则抽查表（新增入口走插件/工具集机制）

| 缺口 | 抽查点 | 证据 |
|---|---|---|
| L1 桌面插件宿主 | internal/desktopbridge/desktopbridge.go | diff 确认 `initPluginHostDesktop()`（NewPluginHost → RegisterCordisTools → RegisterToolsetTools → MigrateLegacyBuiltinJSON → LoadAllToolsets → LoadCordisPatch → SetPluginHost/SetGlobalPluginHost，sync.Once 幂等）；`buildDesktopLoopOpts` 合并 MergePluginTools + ApplyHarnessToolFilter（插件工具豁免回调） |
| T1 孤儿工具插件化 | .pair/plugins/tool-asset/index.js + plugin.go + tool_landing_test.go | 插件 `apply(ctx)` 经 `ctx.tools.register` 声明、`execute: (args) => ctx.hostTool.exec(...)` 复用宿主执行器（seam：编排在插件、能力在宿主）；`NewPluginHost` 内 `ArchiveHostLegacyTools(root)`；`TestToolLandingMatrix` 240 工具落地矩阵含 hostTool 135 |
| C1/C2 角色磁盘优先 | internal/agent/role_prompts.go + reviewer/planner/evaluator | `LoadRolePrompt(name)` 磁盘优先（缺失/空白回退内置）；`DefaultReviewerPrompt/DefaultPlannerPrompt/DefaultJudgePrompt` 全部经 loader；config/roles/*.md 5 份在盘 |
| L2 hook 接线 | internal/agent/loop_hooks.go + tools.go + loop.go | 配置钩子（.pair/settings.json + ~/.pair/settings.json）经 internal/hook.Load；插件钩子 RegisterLoopHook / ctx.hooks.register；Registry.Execute 单闸口 + Loop.Run 接线；无配置全 no-op |
| S1 Provider 插拔 | provider_impls.go + 7 个生产构造点 | `CreateProvider` 统一路由（web_server×3、subagent_spawn×2、desktopbridge、stub、llm_analyze 均改走）；未注册回退 OpenAI 兼容；插件卸载自动还原 |
| G1 审核配置单源 | internal/agent/toolconfig.go | settings.json 顶层为唯一来源，.pair/tools.json 一次性迁移后删除；`TestReviewConfigSingleSourceMigration` 通过 |
| F1 主题选项 | .pair/plugins/ui-appearance/index.js | options 补齐 `['dark','light','warm','night']`（与壳 4 套 .theme-* CSS 对应） |
| T2 描述修正 | .pair/plugins/tool-screenshot、tool-web-debug | 全 .pair/plugins 与 .pair/toolsets/default.json 中 `image_analyze|image_ocr` 零残留 |

---

## 6. 备注与建议（非阻塞）

1. **low**：packager.json 与 .pair/plugins/* 不在 t2 inScope 字面清单（见 §3 例外
   ①②）；建议 captain 在收尾时在任务记录中显式认可这两类「缺口闭环必需产物」。
2. **info**：团队启动前 WIP 大量未提交（§3 基线说明）；t5 集成「git status 干净」
   的验收需先与 captain 确认 WIP 处置策略（提交 or 保留）。
3. **info**：`TestToolLandingSpotCheck` 等 5 项依赖插件二进制（build-plugin-bins
   产物）；t5 全量集成构建（含 scripts/build-plugin-bins.bat）后应复跑确认转绿。
4. **info**：验证环境需 CGO_ENABLED=1 + 本地模块缓存（GOPROXY=off）；已确认
   机器级 `%AppData%\go\env` 的 CGO_ENABLED=0 为环境配置，非项目问题。

---

*验证执行：verifier（独立 QA），2026-08-29。验证日志留存于
`.verify-tmp/`（test-full-cgo1.log / test-targeted.log / smoke2.err），
清理后本报告即完整证据。*

---

# 附：t4 审查结论与 t5 集成结论（2026-08-29 追加）

## A. t4 代码审查（reviewer，verdict = pass）

审查覆盖 t2 全部 9 项闭环的最终代码，核对结论：
1. 每项缺口均有代码落地且与 t1 报告对应；
2. harness 对齐未破坏（ApplyHarnessToolFilter 插件豁免回调 web/desktop 一致，
   Registry.Execute 为 Go/JS 双循环唯一工具闸口）；
3. 插件优先原则未违反（T1 不注册进 Registry、S1 注册表优先路由、C1 磁盘优先、
   L2 插件钩子同事件表）。

独立验证：`go build ./...`、`go vet ./...` 全通过；t2 七个专项测试全过；
internal/agent 全量测试排除环境依赖项后仅 TestRunShellBacktick 失败
（bash 信号管道环境限制，与上文 §2.1 清单一致，非 t2 引入）。

**Findings（2 medium + 5 low，均不阻断，建议后续 repair 闭环）：**

| ID | 级别 | 问题 | 建议修复 |
|---|---|---|---|
| M1 | medium | 随包 config/roles/reviewer.md 与内置 reviewerSystemPrompt 内容漂移（实测 disk 3121B vs builtin 3271B，决策标准段不一致），磁盘优先 loader 使其在生产静默生效 | 同步 roles 文件与内置提示 + 漂移守卫测试 |
| M2 | medium | L2 钩子接线硬编码 Trusted:true 旁路 internal/hook 信任门且 web 监听全接口——打开恶意工作区即执行钩子 shell 命令 | 信任门控（工作区信任标记）后再装载项目钩子 |
| L1 | low | bridge_release → bridge_lockdown 描述漂移 | 同步描述 |
| L2 | low | 钩子 payload.turn 恒 0 | Loop 传入当前 turn |
| L3 | low | G1 toolconfig.go 注释与实现不一致 | 注释同步 |
| L4 | low | ProviderImpl 还原顺序敏感 | 文档化语义 |
| L5 | low | config/philosophy 目录残留（已移出 packager） | 归档/删除确认 |

**处理状态（2026-08-29 t2 集成收尾）：**
- **M1 ✅ 已修复**：`config/roles/reviewer.md` 决策标准段已同步补齐（与内置 reviewerSystemPrompt 展开一致）；新增漂移守卫测试 `TestRolePromptDiskDefaultSync`（仓库 config/roles/{reviewer,planner,judge}.md 必须与内置回退一致，防未来再漂移），通过。
- **M2 ✅ 已修复**：信任门控落地——`SetLoopHookRoots` 默认不装载项目钩子（`<root>/.pair/settings.json` 需显式信任），全局钩子（`~/.pair/settings.json`）不受影响；生产入口 `InitLoopHooks` 默认不信任，`PAIRCODE_TRUST_PROJECT_HOOKS=1` 显式信任；新增 `TestLoopHookProjectTrustGate`（不信任不拦截/信任后拦截）与 `TestLoopHookConfigBlock`（走显式信任路径），通过。默认行为变化：项目级钩子需显式信任才生效（更安全）。
- **generate_commit_message 失配 ✅ 已修复**：从 `.pair/plugins/tool-system/index.js` 移除该死工具（宿主无实现、零消费方，与生成器白名单对齐），`tool_landing_test.go` 的 `knownSystemToolDrift` 豁免移除。
- L1–L5 低危项：保留记录，建议后续 cleanup 任务处理。

## B. t5 端到端集成（engineer）

- **后端完整构建**：`CGO_ENABLED=1 go build -o pair.exe ./cmd/companion` ✅（74.7MB）。
- **启动冒烟**：`WEB_PORT=9322 ./pair.exe` → 进程持续运行、`/api/health` 200、
  启动日志无 FATAL/panic、全局插件宿主 43 插件、工具集 44 个（18 插件，含 t2 新增
  7 个 tool-*）；验证后立即终止并释放端口。
- **前端构建**：⚠️ **沙箱环境阻断**——`npm run build`（正确入口 `plugins-src/ui-app`；
  `cmd/companion/web-ui` 无 package.json）在 受限沙箱内无法执行：vite 依赖 esbuild
  JS API，esbuild 服务进程 spawn（pipe stdio）被 受限沙箱设计边界拦截（`spawn EPERM`；
  最小复现实验：Node `child_process.spawn` 的 pipe/overlapped stdio 均 EPERM、
  inherit 可用；esbuild-wasm node 入口同样 spawn）；本会话无批准升级通道。
  **前端构建需在沙箱外执行**（`cd plugins-src/ui-app && npm run build`，产物
  `.pair/assets/runtime/web`，packager 经 robocopy 同步 `cmd/companion/web-ui/dist`）。
- **git status**：集成后仅预期的源码/文档改动；pair.exe 等产物受 .gitignore 保护。
- **t3 建议跟进项**：build-plugin-bins.bat 已废弃（插件全 JS 化，禁止再跑）；
  TestToolLandingSpotCheck 等二进制依赖测试属既有 WIP 失配（tool-project-info 等
  插件仍 binary 形态而内嵌内核未覆盖 project-info/verify/memory 组），建议后续
  专项修复（插件 JS 原生迁移或内嵌内核补齐）。
