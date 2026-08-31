# Round2 插件化专项：独立验证报告（t3 · verifier）

> 独立验证 t2 实施结果（docs/plugin-round2-plan.md §6）。验证人：verifier。
> 验证范围：构建/测试、范围合规、插件装载冒烟（临时端口）、去重安全性抽查、
> agent-teams 参考对齐核对。**本报告不修改任何实现代码**（唯一改动文件为本报告）。
> 基线：master @ 94063c6f；go1.26.4 windows/amd64；node v22.17.0。

---

## 1. 命令结果表（逐条运行 t2 verify 命令）

| # | 命令 | 退出码 | 结果 | 说明 |
|---|---|---|---|---|
| 1 | `go build ./...`（GOCACHE/GOMODCACHE 重定向 `.verify-tmp/`，CGO_ENABLED=1） | 0 | ✅ 通过 | stderr 仅 go telemetry upload token 沙箱噪音（Access denied，不影响构建） |
| 2 | `go vet ./...`（同上） | 0 | ✅ 通过 | 同上 |
| 3 | `go test ./internal/agent/ -count=1 -timeout 10m`（同上） | 1 | ⚠️ 环境依赖失败（非回归） | 失败清单与 t2 §6.13 完全一致，全部为沙箱环境限制（见下） |
| 4 | 关键回归测试集（`-run` 过滤，见 §1.2） | 0 | ✅ 通过 | 2.0s 全绿 |
| 5 | `node --check` 全部插件（t2 自测项，抽样复核 git diff 语法一致性） | — | ✅ 通过（测试装载验证） | 插件均经 go test 装载执行，语法由装载验证覆盖 |
| 6 | 插件装载冒烟（WEB_PORT=9479 临时端口，`/api/health`、`/api/plugins`） | 0 | ✅ 通过 | 见 §3 |

### 1.1 全量测试环境依赖失败清单（与 t2 §6.13 逐项吻合，判定非回归）

| 失败测试 | 根因 | 判定 |
|---|---|---|
| TestToolRunCommand / TestRunShellWithBash / TestRunShellBacktick / TestRunBackground / TestRunCommandInLoop / TestDetectBash / TestJSPluginCtxBashLogger / TestToolCoreJSNative（run_command 步）/ TestToolHarnessJSNative（bash ⑥ 步）/ TestHarnessAlias_GlobGrepBash（bash 步） | msys bash `couldn't create signal pipe, Win32 error 5`（受限 token 无法建命名管道，受限沙箱文档化边界） | 环境依赖 |
| TestWebDebugTimeoutMs | rod/Chromium 下载写 `%APPDATA%\rod` 被拒（Access denied） | 环境依赖 |
| TestRunCodeNested | bash 信号管道（同 msys 根因） | 环境依赖 |
| TestNodeBridgeE2EHelloBridge | 需先 `npm install local-test-plugin` | 环境依赖（t2 同注） |

> 判别依据：失败信息均为 `exit status 0xc0000142` + `couldn't create signal pipe`/`Access is denied`，
> 非断言语义错误；t2 报告的 5 个 binary 插件失败 + office csv_read/word_read 断言失败**本轮全部通过**（见 §1.2）。

### 1.2 关键回归测试（t2 声称修复项，独立复跑全绿）

`go test ./internal/agent/ -run 'TestToolGitJSNative|TestToolMemoryJSNative|TestToolProjectInfoJSNative|TestToolVerifyJSNative|TestToolBugJSNative|TestToolGitJSNativeMultiProject|TestToolLanding|TestToolOfficeJSNative|TestAgentTeamsDiskLoad|TestLoopHookConfigBlock|TestReviewConfigSingleSourceMigration|TestRolePromptDiskDefaultSync|TestApplyWorkspaceToolsetWhitelist|TestToolBinaryJSNative|TestLoopHookProjectTrustGate' -count=1` → **ok（2.0s）**

另：`TestToolHarnessJSNative` 在 bash ⑥ 步前完成 read 行号块 / edit replace_all / file_path 别名等 R2-7 断言（步序①-⑤ 通过后失败于 bash 环境步）；`TestNodeBridgeServiceTools / TestNodeBridgeManagerSet / TestNodeBridgeProtocolPing / TestNodeBridgeBindHost` 全过（旧名→新名桥映射生效）。

---

## 2. 范围核对（git status/diff 全量）

- 改动路径 **49 个**（48 修改/删除 + 1 未跟踪），逐项对照 t2 inScope（R2-1~R2-10）：
  - `.pair/plugins/` 11 个 JS（agent-teams/tool-bridge/tool-bug/tool-git/tool-harness/tool-memory/
    tool-project-info/tool-shell/tool-system/tool-verify/web-api）→ R2-1/R2-6/R2-7/R2-10 ✅
  - `cmd/companion/`（web_server.go −1157 行纯删除、subagent_spawn.go +8）→ R2-4 ✅
  - `internal/agent/` 18 个 Go 文件（agentloop/diskplugin_jsnative_test/harness_test/jsplugin/
    jsplugin_agents/legacy_host_tools/loop_hooks/node_bridge/node_bridge_test/planner/reviewer/
    shell/subagent_registry/task_tools/tool_plugin_gen/toolconfig/toolset）→ R2-2/R2-3/R2-7/R2-8/R2-9 ✅
  - `internal/agenttools/tools.go`、`internal/codetypes/codetypes.go` 删除 → R2-5 ✅
  - `config/philosophy/` 10 个 txt 删除、`config/roles/` 4 个 md 旧名→新名 → R2-3 L5/R2-9 ✅
  - `docs/`（plugin-gaps、pluginization-gap-analysis、plugin-round2-plan 未跟踪）→ 文档 ✅
- **outOfScope 零改动**：
  - `.agent-teams/` ✅ 零改动（git status 无条目）
  - `release/` ✅ 零改动（tracked 0 文件）
  - 敏感配置（`config/settings.json`、APIKey/密钥类）✅ 零改动
  - `E:\` 安装目录：仓库内无任何指向其的改动；本轮 git 改动面不含安装目录
  - 未跟踪文件仅 `docs/plugin-round2-plan.md`（t1 交付物）✅
- web_server.go 删除内容核验：`git diff` 增加行仅 diff 头（纯删除 1155 行：FS 死 handler + Git API 区 +
  未用导入 bufio/strconv），无功能增改。

---

## 3. 插件装载冒烟（临时端口 9479，避开 9090）

- 方法：构建 `companion_r3.exe`（`go build -o`，67MB）置于仓库根（InstallDir 语义），
  `WEB_PORT=9479` 启动 → 探活 → 终止并清理二进制。
- 结果：
  - `/api/health` → **200** `{"status":"ok","workspace":"<仓库根>",...}`
  - `/api/plugins` → **200**（87.6KB，含 agent-teams client 半等全量插件数据）
  - 启动日志（`logs/paircode.log` 03:06 段）关键行：
    - `[WebUI] 全局插件宿主已初始化（43 个插件）` ✅ 与 t2 预期一致
    - `[toolset] 已装载 44 个工具集（17 个插件）` ✅ 与 t2 R2-8 预期一致（基线 18 → 17）
    - `[js-plugin:dyn-43:web-api] "已注册 /api/ext/* 共 5 条 HTTP 路由"` ✅ R2-10（8 → 5）
    - agent-teams / agentloop / core-api 52/52 / fs-api 10 / git-api 17 / marketplace（5 接口/2 工具）/
      tool-vision / ui-quick-exec 全部装载 ✅
  - **日志全量 423 行扫描：FATAL/panic/重复/冲突 = 0 处** ✅ 无重复工具注册冲突
- 运行时工具集数据核验（`.pair/toolsets/default.json`，git-ignored）：17 条目，
  **内嵌 code 全为 0**（13 个插件条目 name-only）、`builtin:shell`/`builtin:web` 已删、
  `tool-web` name-only 条目补齐、tool-core 陈旧 DisabledTools 已清 ✅

---

## 4. 去重安全抽查表（被删/被合并工具消费方零残留）

| 去重项 | 消费方扫描 | 结果 |
|---|---|---|
| Go 旧名工具（read_file/write_file/edit_file/run_command/search_content/search_files） | `config/`（4 个角色 md + 其余） | ✅ 零残留（roles 已改 read/bash/glob/grep） |
| 同上 | 生产注册面 `RegisterHostFrameworkTools`（builtin_plugins.go:105） | ✅ 仅框架工具，零旧名；旧名 Go 实现仅 builtinPluginSpecs 归档 + 测试引用（t2 文档化保留） |
| 同上 | `internal/agent/node_bridge.go` 服务映射 | ✅ 已改 read/write/bash 直连（list_files→os.ReadDir），桥测试全过 |
| 同上 | `web/`、`cmd/companion/web-ui/dist`（前端 built 产物） | ✅ 零残留（旧名图标分支保留为兼容壳，built 产物无源码改动，符合 F2/F3 defer） |
| `builtin:shell`/`builtin:web` 冗余条目 | `.pair/` 全树 | ✅ 零残留 |
| `/api/ext/fs/{read,exists,list}` 三路由 | 全仓库（前端 dist、internal/agent、docs） | ✅ 仅 web-api/index.js 注释性提及（描述删除动作），无消费者 |
| `bridge_release` → `bridge_lockdown`（L1） | internal/agent、tool-bridge/index.js、docs | ✅ 已同步 4 处 |
| reviewBlacklist 默认值 | internal/agent/toolconfig.go | ✅ 无默认黑名单引用旧名（测试 fixture 的 `read_file` 为遗留配置迁移样例，非引用） |
| 内嵌 code vs 磁盘插件双份 | default.json 运行时态 | ✅ 13 条内嵌已清，装载逻辑磁盘优先（toolset.go applyToolsetPlugin） |

### 4.1 发现（低危，不阻塞）

| ID | 级别 | 内容 | 建议 |
|---|---|---|---|
| F1 | low | `.pair/plugins/tool-bridge/package.json:4` purpose 仍写 `bridge_release`（自动生成描述串；工具名/描述本体已全部改对，无功能消费者） | 后续随手同步该字符串 |
| F2 | info | `TestHarnessAlias_GlobGrepBash`/`TestToolHarnessJSNative` 在本沙箱止于 msys bash 环境步（t2 §6.13 标注含歧义：前者列于环境失败清单但注释"通过"） | 正常开发环境复跑确认；非代码回归（bash.exe 无法启动与代码无关） |
| F3 | info | go telemetry upload token stderr 噪音（沙箱限制） | 可忽略 |

---

## 5. agent-teams 参考对齐核对（参考 @nanmicoder/dsh-agent-teams v0.1.14）

| 能力 | 参考 | repo（t2 后） | 核对 |
|---|---|---|---|
| 13 工具名 | agent_teams_* ×13（tools.js 实测） | 同名 ×13（index.js 实测） | ✅ 完全一致 |
| 两阶段计划 + 质量门禁 | staged 全流程 + quality-gates 契约 | 同（既有） | ✅ |
| profiles 命名模板 | profiles.js `MAX_TEAM_PROFILES=16` | `MAX_TEAM_PROFILES=16` + `TEAM_PROFILES`（default/captain-planning）+ `create(profile=…)` | ✅ 同值上限；内置模板替代配置驱动（宿主无插件配置 schema，裁剪合理） |
| reasoning_effort 透传 | 成员目录原生字段 | SubAgentSpec.ReasoningEffort → ctx.agents.start 映射 → buildSubAgentProvider（仅档位时也构造 Provider 覆盖 ThinkingMode） | ✅ 全链路透传 |
| memberProvider fork | spawn/fork | 未实施（宿主 ctx.agents 无 fork 能力面） | ⚠️ defer，与 t2 §7.1 记录一致 |
| slash 命令 /agent-teams | command.js | 未实施（宿主无 ctx.commands 命令面） | ⚠️ defer，与 t2 §7.1 记录一致 |
| harness 工具语义 | read 行号 / edit replace_all / bash description+timeoutMs / write file_path | tool-harness 实测：`file_path`+`path` 双参数、`N: text` 行号 + footer、`replace_all`、`timeoutMs`（>0 覆盖 120s）、bash `description` 可选（文档化差异） | ✅ 已对齐（测试断言步通过） |
| jobs/todo 别名 | job_output/job_list/job_kill / todo_write | tool-shell 3 别名（bgRegistry.list + ctx.process.list）+ tool-system todo_write（registerTaskTools 共享 handler + ArchiveHostTool） | ✅ 别名方式零破坏 |

---

## 6. 结论

- **可构建性**：`go build ./...`、`go vet ./...` 全绿。
- **测试**：非环境依赖测试全绿；环境依赖失败与 t2 文档清单逐项吻合（msys bash 信号管道 /
  rod 下载 / npm 插件安装），**t1 遗留的 5 个 binary 插件失败与 office 断言失败已修复**。
- **范围合规**：49 个改动路径全部落在 t2 inScope；`.agent-teams/`、`release/`、敏感配置、
  `E:\` 安装目录零改动。
- **装载健康**：43 插件、工具集 44（17 插件条目）、web-api 5 路由，与 t2 预期完全一致；
  日志零 FATAL/panic/重复注册冲突；`/api/health`、`/api/plugins` 200。
- **去重安全**：所有被删/被合并工具在消费方零残留（F1 为描述串低危，不阻塞）。
- **对齐度**：agent-teams 13 工具名/MAX_TEAM_PROFILES/质量门禁对齐，profiles + reasoning_effort
  已落地，fork/slash 按宿主能力 defer（有记录）。

**总体判定：PASS（可进入 t4 代码审查）**，附 1 个低危建议（F1 package.json 描述串）与 2 个信息项。

---

## 7. 补充核实（队长质询项 · 2026-09）

### 7.1 config/ 改动范围核验（t2 合同 inScope 未列 config——已记录例外复核）

- 改动面 = **恰好 14 个路径**：`config/philosophy/` 10 个 txt 删除（R2-3 L5）+ `config/roles/`
  4 个 md 修改（R2-9 消费方同步）。git status 无其他 config 条目。
- **消费方同步最小集判定（合理）**：4 个 roles md 的 diff 均为「旧工具名→新工具名」最小替换
  （explorer 1 行、planner 1 行、reviewer 2 行、verifier 2 行）；未改动的 `config/roles/judge.md`
  （mtime 8/16，t2 前）实测零旧名引用——**改动集恰为引用旧名的文件全集，无多余触碰**。
- **philosophy 删除安全性**：全 Go 代码 grep `config/philosophy|philosophy/` 零引用；packager
  已移除（t1 依据），删除无悬挂引用。
- **敏感配置零触碰**：git-ignored 运行时文件 `config/ai-presets.json`（mtime 8/27 23:50）、
  `config/mcp.json`（8/20）、`config/models.json`（8/27 18:39）均在 t2 窗口（8/29 00:40–02:48）之前，
  **未被 t2 修改**；`config/settings.json` mtime 2:46:03 落在 t2 冒烟（02:45:46 启动）后 17s——
  判定为**冒烟实例启动的运行时副作用**（lastProject/recentProjects 等保存，内容为合法 JSON 1854B，
  键清单无异常），非代码改动、非刻意触碰；`git status` 对这些文件零记录（ignored）。
- 结论：config 改动属 t1 计划 R2-3 L5 / R2-9 已记录例外的**必要最小同步**，不判范围违规。

### 7.2 去重一致性交叉验证（18→17、43 插件、243 工具）

| 核对点 | t2 断言 | 独立证据 | 判定 |
|---|---|---|---|
| 工具集插件条目 18→17 | R2-8：基线 18 → 17（−2 builtin +1 tool-web） | 改前实测日志（.verify-tmp/smoke.log.err 00:51）「44 个工具集（**18** 个插件）」+「web-api 8 条路由」；改后实测日志（paircode.log 02:45 与本次 03:06 复跑）「44 个工具集（**17** 个插件）」+「web-api 5 条路由」——前后构建日志直接对比成立 | ✅ 一致 |
| 工具总数 243 零未落地 | §6.2：243 = hostTool 135 / binary 43 / mixed 26 / native 39，零未落地 | `TestToolLandingMatrix -v` 独立复跑：`落地矩阵: 共 243 工具 map[binary:43 hostTool:135 mixed:26 native:39]` + `--- PASS`；测试对 hostTool 未存档/内嵌内核缺失均 `t.Errorf`（通过即零问题） | ✅ 一致 |
| 插件总数 43 不变 | 去重只动工具集条目不动插件包 | 改前/改后日志均「全局插件宿主已初始化（43 个插件）」；`TestToolLandingSpotCheck` 代表性工具真实执行通过（skill_list/history_count/tool_stats/binary_hash/project_info_list） | ✅ 一致 |
| 无重复工具注册冲突 | 冒烟无冲突 | 423 行日志 FATAL/panic/重复/冲突 = 0；landing 矩阵每工具单归属无重名冲突 | ✅ 一致 |
| default.json 运行时态 | 17 条目、内嵌 code 清空 | 实测 17 条目（14 非 builtin + 3 builtin），codeLen 全 0，builtin:shell/web 不存在，tool-web 条目在位 | ✅ 一致 |

---

## 8. 集成收尾（t5 · engineer · 2026-09）

> t4 审查 verdict=pass 后，t5 端到端集成：t4 findings 处置 + 全量构建 + 启动冒烟 + git 收尾。

### 8.1 t4 findings（5 个非阻塞）处置

| ID | 严重度 | 处置 | 复验 |
|---|---|---|---|
| F4 | medium | **已修复**：agent-teams profile=default 种子任务 DAG 接线（TEAM_PROFILES 增 ref/deps/reviewedTaskRef，seed 两遍建任务静态接线 requirements→implementation→verification；review 依赖 implementation 且 reviewedTaskId 指向实施任务） | node --check ✅；TestAgentTeamsDiskLoad ✅ |
| F1 | medium | **已修复**：session_context.go fileModifyTools/toolPathParams 增 edit/write（file_path/path）；storm_breaker.go repeatSuccessSignature 增 write/edit（旧名保留兼容历史） | go build ✅；TestStormBreaker*/session_context 相关测试 ✅ |
| F2 | low | **已修复（全插件清扫）**：14 个 tool-* 插件 LLM 面向文案旧名→新名（read/glob/grep/bash/edit/write）；tool-core purpose/头注释更正为实际 3 工具；tool-bridge package.json purpose 同步 lockdown | 全插件 node --check ✅；非 assets 插件 grep 旧名零残留 |
| F3 | low | **已文档化**：node_bridge.go fs.read 映射注释说明 外部行号块输出与桥接剥离方案（当前零消费者） | go build ✅ |
| F5 | low | **已修正**：plugin-round2-plan.md §6.7 ask_user 表述更正（仅插件侧 schema 增 questions，Go 宿主侧未同步、已文档化降级） | 文档核验 ✅ |

### 8.2 全量构建与冒烟（t5 独立复跑）

| # | 命令 | 退出码 | 结果 |
|---|---|---|---|
| 1 | `cd cmd/companion/web-ui && npm run build`（兼容入口：build-ui.mjs → ui-app vite → sync-web-dist） | 0（构建失败不阻断脚本） | ⚠️ **沙箱 EPERM**：`[commonjs--resolver] spawn EPERM`（vite/esbuild 子进程 spawn 被 受限沙箱拦截，文档化边界）——已报队长走全权限通道；**本轮无前端源码改动**，dist 为既有产物 |
| 2 | `CGO_ENABLED=1 go build -o pair.exe ./cmd/companion`（GOCACHE/GOMODCACHE 重定向） | 0 | ✅ pair.exe 67.5MB（git-ignored，不作为提交物） |
| 3 | 启动冒烟 WEB_PORT=9323（临时端口，避开 9090/9479） | 0 | ✅ `/api/health` 200、`/api/plugins` 200（43 插件 91.7KB）、`/api/tools` 200（51KB）；日志「全局插件宿主已初始化（43 个插件）」「已装载 44 个工具集（17 个插件）」；FATAL/panic/重复/冲突 = 0 → 探活后终止并清理 |

### 8.3 结论（t5 集成判定）

- t4 5 个 findings 全部处置（3 修复 + 1 文档化 + 1 文档修正），无遗留阻塞项。
- pair.exe 全量构建成功；冒烟 43 插件 / 17 工具集条目 / 探活 200 / 零 FATAL——与 t2/t3/t4 预期完全一致。
- 前端构建沙箱 EPERM（记录在案），前端无改动，dist 无需重建；队长全权限通道可复跑同一命令。
- git 收尾：见 t5 提交（Round2 全部改动 + docs 收尾）；`git status` 仅剩预期产物（pair.exe/运行时数据 git-ignored）。

