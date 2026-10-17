# PairCode IDE 审计报告：冗余设计 · 冗余功能 · 未实装功能

> 审计对象：`gou-ide`（PairCode IDE，module `github.com/hoonfeng/paircode`）
> 审计日期：2026-10-17
> 审计性质：**只读审计**——本报告为唯一交付物，未修改任何源码、未删除任何文件、未执行构建/部署。
> 审计基线：git `HEAD = d3359e3c`（含工作区未提交改动 4 个插件/文档文件）

---

## 0. 审计方法与规模

### 0.1 方法
| 手段 | 用途 |
|---|---|
| `git status` / `git log` | 基线确认、改动面识别 |
| 引用图检索（`grep` 全仓 import/调用） | 判定「代码是否可达」——无调用者 = 冗余 |
| 逐文件 `diff`（去 CR 归一化后 md5） | 判定「双份实现是否真重复、是否已漂移」 |
| `go list ./...` | 判定「哪些包进构建」 |
| 目录体积 `du -sh` | 量化产物堆积 |
| 现场进程/安装目录核查 | 判定「源码 vs 运行时」一致性 |

> ★ 关键方法纪律：**CRLF/LF 会伪造出 100% 的 diff**。首次直接 `diff` 得出 `lang_builders.go` 3986 行差异，
> 去 CR 后该文件**完全相同**。本报告所有「相同/差异」结论均基于 `tr -d '\r'` 归一化后的比对，
> 未归一化的行数差异一律不作为证据。

### 0.2 规模基线
| 维度 | 数量 |
|---|---|
| Go 生产代码 | **170,090 行 / 396 文件** |
| Go 测试代码 | 60,220 行 / 216 文件 |
| 磁盘插件 JS（`index.js` + `client.js`） | 22,411 行 / 34 插件 |
| `plugins-src` 插件 Go 源码 | **22,865 行 / 30 包** |
| 文档（`docs/` + `doc/`） | 12,748 行 / 34 篇 |
| `go list ./...` 包总数 | 63（其中 `plugins-src` 占 **30 = 47.6%**） |

### 0.3 历史审计衔接
仓库已有两份相关历史文档，本报告与之**不冲突、且做了增量核实**：
- `docs/plugin-tools-overlap-audit.md`（Round4.5）：结论「工具面无同名双注册，不存在真重复」——本次复核**确认成立**（详见 §3.2）。
- `dev/v16-t1-frontend-dead-code-report.md`（2026-09-03，基线 5f2c41a3）：发现 `chat-utils.js` 死文件——本次核实**已删除**（文件不存在）。

---

## 1. 结论摘要

| # | 类别 | 对象 | 规模 | 证据强度 | 风险 | 处置建议 |
|---|---|---|---|---|---|---|
| **R1** | 冗余设计 | `plugins-src/plugins/*` 独立插件二进制整条路径 | **22,865 行 / 30 包** | 强（无产物/无构建入口/无运行时引用） | 高（维护双份、误改旧副本） | 归档或删除；保留前先合并漂移 |
| **R2** | 冗余设计 | `internal/server/handler` 归档实现 | 3,213 行 / 92 导出函数，**约 63 个无调用者** | 强（`RegisterAll` 自述无调用者且实测 0 调用） | 中（误认为仍生效） | 标注归档 / 移入 `internal/legacy` |
| **R3** | 冗余设计 | 路由双份（插件 ExtRoute 优先 + 宿主 mux 兜底） | `git.go` 439 + `health_fs.go` 393 行被遮蔽 | 强（`ext_routes.go:142` 优先级） | 低（有意兜底） | 保留但在文件头标注「兜底实现」 |
| **R4** | 冗余设计 | `internal/bridge` 桌面模式死分支 | 2 文件 / 约 230 行 | 强（`HandleBridgeCall` 0 调用方） | 低 | 精简为 Web-only 注册表 |
| **R5** | 冗余设计 | 插件双目录 `.pair/plugins` ↔ `bin/.pair/plugins` | 43 处差异 + `.smoke` 残留 610K | 强（`diff -rq`） | 中（安装污染） | 修正同步脚本，排除测试残留 |
| **R6** | 冗余设计 | 构建产物与工作区污染 | **合计约 4.1 GB** + 根目录 6 个 exe（221 MB） | 强（`du`） | 中（仓库臃肿、误提交） | 清理 + 落 `.gitignore` |
| **R7** | 冗余设计 | `plugins-dist/` 空目录（文档称创作域真源） | 0 文件 vs 文档声称 6 包 | 强 | 中（文档误导） | 修正文档或补真源 |
| **F1** | 冗余功能 | **codegraph 双份实现**（`pkg/codegraph` vs 插件内 `codegraph`） | 11,028 行 ↔ 10,861 行，**15/23 文件完全相同**，8 文件已漂移 | 强（逐文件 md5 + diff） | **高**（漂移方向为「pkg 新、副本旧」） | 删除副本，改为单一真源 |
| **F2** | 冗余功能 | 内置工具组 vs 磁盘插件工具 | 20 组规格 | 强 | 无（分层设计，非重复） | 不改，仅文档澄清 |
| **F3** | 冗余功能 | 编排体系概念重叠（6 套并存） | 6 插件 / 约 12K 行 JS | 中（职能不同、概念重叠） | 低-中（认知负担） | 文档收敛，不合并代码 |
| **U1** | 未实装 | Unix PTY | 1 文件 11 行桩 | 强（显式 error） | 低（Windows 主目标） | 标注平台限制 |
| **U2** | 未实装 | 桌面模式（bridge 双模式承诺） | 见 R4 | 强 | 低（已 web-only） | 文档更正 |
| **U3** | 未实装 | `plugins-src` 「可编译回退」路径 | 见 R1 | 强（无产物、无构建入口） | 中（panic 提示误导） | 文档更正或补构建脚本 |
| **D1** | 源码/运行时不一致 | 安装目录独有 `tool-art`、`wechat-bridge` | 2 插件（2,248 行 + 约 1,300 行） | 强（目录比对） | **高**（覆盖安装即丢） | 回流入库 |
| **D2** | 源码/运行时不一致 | `tool-flow` 仓库领先安装目录 683 行 | 3,355 vs 2,672 行 | 强（md5） | 低（正常开发领先） | 发版时同步 |

**一句话结论**：项目**工具面没有真重复**（历史审计成立）；真正的冗余集中在**「实现层的双份拷贝」**
——`plugins-src` 整条独立二进制路径（22.9K 行，含与 `pkg/codegraph` 98% 相同的副本）与
`internal/server/handler` 的归档实现（3.2K 行），二者均已**不可达或仅测试可达**，却仍参与全量编译与维护。
未实装功能只有 Unix PTY 一处硬缺口，其余「未实装」实为**文档承诺与实现状态的漂移**。

---

## 2. 冗余设计详述

### R1（高优先）`plugins-src` 独立插件二进制路径已成「无产物、无入口、无引用」的第三份实现

**结构**
| 插件 | Go 文件 | 行数 |
|---|---|---|
| `tool-codegraph` | 29 | **12,621** |
| `tool-office` | 5 | 2,901 |
| `tool-bug` | 6 | 1,641 |
| `tool-binary` | 6 | 1,143 |
| `tool-project-info` | 5 | 1,088 |
| `tool-harness` | 5 | 1,045 |
| `tool-memory` | 5 | 942 |
| `tool-web` | 5 | 789 |
| `ui-quick-exec` | 5 | 695 |
| **合计** | **71** | **22,865** |

**四条「不可达」证据**
1. **无产物**：`find . -name 'tool-*.exe'` → **0 命中**；`.pair/plugins/tool-codegraph/` 仅有 `index.js` + `package.json`。
2. **无构建入口**：`grep -rn "go build.*plugins-src" scripts/ *.mjs *.cmd` → **0 命中**；`scripts/publish-official-plugins.mjs` 仅在**过时映射表**中提及 `tool-codegraph-extra`（第 255 行），不编译源码。
3. **无运行时引用**：`grep -rn "plugins-src" cmd internal pkg --include='*.go'` → 仅 **5 处注释**，无 import。
4. **但仍在编译**：`go list ./...` = 63 包，其中 **30 包属于 `plugins-src`** → `go build ./...` / `go vet ./...` / `go test ./...` 都会编译它们，形成持续的编译与维护成本。

**实际执行链（当前生效路径）**
```
磁盘插件 .pair/plugins/tool-*/index.js
   └─ ctx.tools.register({ execute: (a) => ctx.binary.exec(name, a) })
        └─ internal/agent/jsplugin.go:726  找不到 exe → callEmbeddedTool(...)
             └─ internal/agent/embedded_tools.go:29 embeddedToolRegistrars
                  └─ internal/agent/*.go 内置实现（+ pkg/codegraph、pkg/memory…）
```
即：**插件声明（JS）+ 宿主内嵌内核（Go）** 是当前唯一链路；`plugins-src` 是 2026-08 之前
「独立二进制插件」时代的实现，已无位置。

**风险**
- 任何「照着 `jsplugin.go:733` 的 panic 提示去编译」的人，会得到一个**与生产实现不一致**的旧二进制
  （见 F1：副本落后于主实现）。
- 双份代码导致搜索结果、审计、重构都被污染（本次审计首轮即被 `plugins-src` 干扰）。

**建议**
1. P0：先确认 `plugins-src` 无发布依赖 → 整体移入 `_archive/plugins-src/` 并从主模块排除（或加 `//go:build ignore`）。
2. 若坚持保留：必须先消除 F1 的漂移，并在 `docs/plugin-development.md` 标注「备用路径、无产物」。

---

### R2（中优先）`internal/server/handler`：92 个导出函数中约 63 个无调用者

**自述证据**（文件自己的注释）：
> `internal/server/handler/register.go:6-9`
> 「★ 2026-09-13 现状：**无调用者**——内置 /api/* 接口已全部进内核路由表（internal/agent/kernel_api.go），
> 由 core-api 磁盘插件经 ctx.kernel.install 挂载……本文件与本包 handler 仅保留给测试与参考。」

**实测核实**
| 项 | 结果 |
|---|---|
| `RegisterAll(` 调用方 | **0**（仅定义处） |
| 包内导出函数 | 92 |
| `cmd/companion/kernel_register.go` 实际使用 | 22 个 |
| `cmd/companion/web_server.go` 实际使用 | 7 个 |
| 其余 | **约 63 个无任何调用方**（`HandleFSRead`/`HandleGitStatus`/`HandleChatSend` 等整套旧注册表） |
| 包规模 | 3,213 行 / 12 文件 |

**风险**：误改 `handler/git.go` 以为在修线上行为（实际生效的是 `webServer` 方法 + 插件路由）；
新人按 `register.go` 追加接口，接口不会生效。

**建议**：文件头已有注释，但**函数级**仍无提示。P1：
- 将 63 个无引用函数整体移入 `internal/legacy/handler/`（保留测试可达路径），或
- 逐文件加 `// Deprecated: 无调用者，见 kernel_register.go` 标注，并在 `register.go` 顶部加 `Deprecated` 包注释。

---

### R3（低优先）路由双份：插件优先 + mux 兜底

**机制**：`internal/agent/ext_routes.go:142-149`
```go
// ExtRouteMiddleware 顶层处理器：插件路由优先，未命中交给 next（宿主 mux）。
func ExtRouteMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w, r) {
		if ServeExtRoute(w, r) { return }
		next.ServeHTTP(w, r)
	})
}
```

**被遮蔽的实现**：
| 文件 | 行数 | 对应插件 |
|---|---|---|
| `internal/server/handler/git.go` | 439 | `git-api`（17 条 `/api/git/*`） |
| `internal/server/handler/health_fs.go` | 393 | `fs-api`（23 条 fs 调用） |

**判定**：这是**有意的兜底设计**（插件停用后接口仍可用），不是缺陷。但两份实现必须同步演进，
否则「插件开着和插件关着行为不同」，属于隐性风险。

**建议**：不改代码，仅在两个文件头注明「插件启用时被 `ExtRouteMiddleware` 遮蔽，修改需同步插件」。

---

### R4（低优先）`internal/bridge` 桌面模式死分支

- `internal/bridge/registry.go:157` 的 `HandleBridgeCall`（桌面桥接统一入口）**无任何调用方**。
- `desktopbridge` 在全仓仅出现在**注释**中（`internal/agent/builtin_plugins.go:95,103` 等），对应目录不存在。
- 桌面端已于 2026-08 移除（`3cda9179`），当前为 web-only 运行时。

**现状**：`bridge.Registry` 仍被 `handler.Router` 双向注册使用（`router.Reg.Register(...)`），
即 Web 侧仍在用该注册表，**但桌面导出能力（`HandleBridgeCall` + `virtualResponse`）是纯死代码**。

**建议**：P2，删除 `HandleBridgeCall`/`virtualResponse`/`errorResponse`/`successResponse`（约 80 行），
保留 `Register/Dispatch/AllRoutes/ServeHTTPMux`。

---

### R5（中优先）插件双目录：`.pair/plugins` 与 `bin/.pair/plugins` 已漂移，且夹带测试残留

```
diff -rq .pair/plugins bin/.pair/plugins → 43 处差异
```

**差异主因**（抽样）：
```
Only in .pair/plugins/tool-flow/.smoke/ws: .git
Only in .../.smoke/ws/.pair/taskboard: .scheduler.lock, driver.json, .last-conv.json
Only in .../.smoke/ws/.pair/flow-runs: run-1792206444008-gtol.json …
```
即 `tool-flow` 的 smoke 测试工作区（`.smoke/`，**610 KB，含一个 `.git` 目录**）被当作插件内容镜像到了安装目录。

**风险**
1. `.smoke/ws/.git` 若被发布流程打包，会污染安装包并在用户机器上生成异常仓库；
2. `.scheduler.lock` / `driver.json` 属运行时状态，不应随插件分发；
3. 镜像漂移会导致「开发改 A、安装目录还是 B」（`tool-flow` 当前就有 683 行差异，见 D2）。

**建议**：P1
- 在 `scripts/sync-plugins-to-bin.mjs` 中排除 `.smoke/`、`*.lock`、`node_modules/`；
- 清理 `bin/.pair/plugins/tool-flow/.smoke/`。

---

### R6（中优先）构建产物与工作区污染：约 4.1 GB

| 目录/文件 | 体积 | 说明 |
|---|---|---|
| `release/` | **3.0 GB** | 历史发布包堆积 |
| `bin/` | 348 MB | 安装镜像 |
| `tmp/` | 288 MB | 临时产物（含 `tmp/gouide-shell` 4 个 Go 文件） |
| `screenshots/` | 232 MB | 验证截图（本次审计期间新增多张） |
| `video/` | 165 MB | 视频素材 |
| `out/` | 71 MB | 构建输出 |
| 根目录 exe ×6 | **221 MB** | `companion.exe` 54M、`companion_test.exe` 71M、`pair-9098.exe` 40M、`pair_iso_test.exe` 40M、`evaluator.exe` 8.7M、`ui-quick-exec.exe` 3.7M |
| 垃圾文件 | — | `nul`(0B)、`[object Object]`(30B)、`@AutomationLog.txt`、`_obs_backend_9098.log`、`_obs_perf_9200.log`、`companion_test.exe~` |

**判定**：其中 `release/`、`bin/`、`out/` 是正常构建产物（应 ignore 而非删除）；
根目录 6 个 exe 与 `nul`/`[object Object]` 是**调试残留**，属真冗余。

**建议**：P1——根目录 exe 移至 `bin/` 或删除；`nul`、`[object Object]`、`*.log`、`*.exe~` 删除并加入 `.gitignore`。

---

### R7（中优先）`plugins-dist/` 空目录 vs 文档承诺的「创作域六包真源」

- 环境文档（`.pair/project.md`）声称：`plugins-dist/<name>/` 是 `tool-art/design/model/music/rig/voice` **6 个创作域插件的真源**。
- 实测：`plugins-dist/` **完全为空**（0 文件）。
- 创作域实际状况：仅 `tool-design` 存在于 `.pair/plugins`；`tool-art` 仅存在于**安装目录**；
  `tool-model`/`tool-music`/`tool-rig`/`tool-voice` **全仓不存在**。

**影响**：按文档去找真源的开发者会得到空目录；`docs/creative-domain-plan.md`、`docs/voice-creation-plan.md`
所述能力与仓库状态不符。

**建议**：P1——二选一：补齐 `plugins-dist` 真源，或修改 `project.md` 与计划文档，标注「已并入 tool-design / 未入库」。

---

## 3. 冗余功能详述

### F1（高优先，本报告最重要发现）codegraph 双份实现，且副本已落后于主实现

**双份位置**
| 侧 | 路径 | 文件 | 行数 | 谁在用 |
|---|---|---|---|---|
| 主实现 | `pkg/codegraph/` | 24 | **11,028** | `internal/agent/codegraph_tools.go:22`、`codegraph_extra.go:13`（生产） |
| 副本 | `plugins-src/plugins/tool-codegraph/codegraph/` | 23 | **10,861** | 仅 `plugins-src/.../main.go`（不可达，见 R1） |

**逐文件比对（去 CR 归一化）**
```
共有文件 23 个：完全相同 15 个 / 有差异 8 个
差异文件：builder.go(73 行) builder_go.go(0*) git_history.go(14) graph.go(1)
          lang_builders.go(0*) py_builder.go(0*) search.go(0*) sqlite_store.go(33)
          tools.go(0*) tsit_builder.go(2) tsit/api.go(78) tsit/language.go(56)
          （0* = 去 CR 后完全相同，仅换行符不同）
```

**漂移方向：`pkg` 更新，副本是旧版**（关键证据）
```diff
pkg/codegraph/builder.go
+ info.Name() == "_temp" ||      // 副本缺失：不跳过 _temp/tmp
+ info.Name() == "tmp" {
+ // ★ 重建文件索引：BuildFull 前旧索引可能缺失/陈旧 → 增量检测会把大量文件误判为变更
+ if err := b.store.SaveIndex(b.collectIndex()); ...
+ // ★ 索引严重过期检测：变更占比 >50% 或 >=800 个 → 回退全量构建
+ if len(changedFiles) >= 800 || (total > 0 && len(changedFiles)*100/total > 50) {
+ 	return b.BuildFull()
+ }
```
即：若有人按 panic 提示编译 `plugins-src/plugins/tool-codegraph` 并在 future 恢复该路径，
会得到一个**缺少索引自愈、且会把 `_temp`/`tmp` 扫描进图谱**的旧版本。

**其它差异**：`tsit/api.go` 78 行差异中大部分是 **gofmt 对齐**（副本未格式化），
说明副本自创建后**从未被维护**。

**建议**（P0）
1. 若 `plugins-src` 路径保留 → 让副本 `import github.com/hoonfeng/paircode/pkg/codegraph`，删除物理拷贝；
2. 若路径废弃 → 随 R1 一并归档，**不要**保留两份可编译的实现。

---

### F2（无需处置）内置工具组 × 磁盘插件工具：确认无同名双注册

本次复核历史结论（`docs/plugin-tools-overlap-audit.md`）**成立**，补充三条独立证据：

1. **宿主不 apply 内置组**：`internal/agent/builtin_plugins.go:36`
   「★ 仅作二进制实现库的组规格……**宿主进程不再 apply 本表**」。
2. **宿主唯一内置注册**是框架工具：`RegisterHostFrameworkTools`（`builtin_plugins.go:107`）
   ——`update_tasks`/`tool_stats`/`history_*`，与插件 `tool-system` 的 hostTool 承接同源。
3. **业务工具可见面 = 磁盘插件**：34 个插件里 20 个 `tool-*` 用 `ctx.tools.register` 声明，
   执行走 `ctx.binary.exec` → 内嵌内核。**声明层唯一**，实现层由内核承载。

**唯一需注意**：`internal/agent` 里的 `registerCodeGraphTools` 等**同时**被
(a) `embedded_tools.go` 的内嵌内核回退 与 (b) `builtin_plugins.go` 的二进制库规格 引用，
这是**同一份 Go 代码的两种用途**，不是重复实现——**不要**据此判定重复。

---

### F3（低-中优先）编排体系：6 套概念并存

| 插件 | 定位 | 规模 |
|---|---|---|
| `agentloop` | Agent 循环 JS 实现（策略在 JS） | 1,216 行 |
| `autopilot` | 自主模式监督者（自然结束后审核/决定下一步） | 900 行 |
| `agent-teams` | 任务 DAG + 质量门禁 + 两阶段批准 | 1,932 行 |
| `tool-flow` | 三层流程库 + 阶段门禁 + **任务看板** | 5,447 行 |
| `tool-workflow` | workflow 脚本编排（pipeline/parallel/phase） | 约 300 行 |
| `tool-scenario` | 场景（工具集白名单） | 约 200 行 |

**判定**：六者**职能不重叠**（循环 / 监督 / DAG / 流程看板 / 脚本 / 工具集），
但共享「阶段、任务、门禁、批准、编排」同一套词汇，且 `agent-teams` 与 `tool-flow` 都实现
「任务 + 门禁 + 批准」两套概念（一个面向团队 DAG，一个面向看板卡），**文档层面易混淆**。

**建议**：不合并代码（语义不同）；在 `docs/plugin-development.md` 增加一张「编排类插件职责边界」对照表。

---

## 4. 未实装功能

### U1 Unix PTY（唯一硬缺口）
```
internal/pty/pty_unix.go:7-10
// Start（Unix）：TODO 用 openpty/forkpty 实现真伪终端（pts 即 tty，无缓冲问题）。
// 当前为桩——解释器探测（DetectShells）已就绪，PTY 启动待实现。
return nil, fmt.Errorf("pty: Unix forkpty 尚未实现（Windows ConPTY 已就绪，Unix 待接）")
```
**影响面**：仅 Unix 平台的交互式终端；主目标平台 Windows（ConPTY）不受影响。
**建议**：在用户文档标注「Unix 暂不支持交互终端」，或移除 Unix 构建标签下的桩函数暴露。

### U2 桌面模式（见 R4）
文档/注释仍多处提及 `desktopbridge` / `buildDesktopLoopOpts`，实际目录与实现均已移除。
**建议**：清理注释中的「Desktop 模式」表述，避免误导。

### U3 `plugins-src`「可编译回退」路径（见 R1）
`internal/agent/jsplugin.go:733` 的 panic 文案：
```
"ctx.binary.exec: 插件二进制不存在 %s（编译：go build -o %s ./plugins-src/plugins/<name>）"
```
向用户承诺了一条**项目已不再走的构建路径**，且编译产物是 F1 所述旧版本。
**建议**：改为「宿主内嵌内核已接管，无需编译插件二进制」。

### U4 其它核查结论（明确「未发现」）
| 检查项 | 方法 | 结果 |
|---|---|---|
| Go 侧 `尚未实现/not implemented` | 全仓 grep（排除测试/注释） | 仅 U1 一处 |
| JS 插件侧 `未实现/暂不支持` | `.pair/plugins/*/index.js` grep | **0 命中** |
| 空实现工具（`execute: () => {}`） | grep | **0 命中** |
| 死插件（缺 `package.json`） | 34 插件遍历 | **0 个** |
| 声明了但无 handler 的工具 | 内置/插件名对照 | 仅历史已移除的 `tool-debug`（`debug_tools.go` 保留供独立二进制） |

---

## 5. 源码与运行时不一致（附带发现，非冗余但风险更高）

### D1（高优先）仓库缺 2 个插件的源码真源
```
仓库 .pair/plugins:  34 项，无 tool-art、无 wechat-bridge
安装目录 E:\Program Files (x86)\PairCode\.pair\plugins: 含 tool-art、wechat-bridge
```
| 插件 | 规模 | 现状 |
|---|---|---|
| `tool-art` | 2,248 行 `index.js`（`art_add/edit/export/import/project/verify`） | 仅存在于安装目录 |
| `wechat-bridge` | `index.js` 37 KB + `client.js` 31 KB + `bin/` | 仅存在于安装目录 |

**风险**：执行「覆盖安装」或从仓库重建安装目录时，这 2 个插件**会丢失**（`overwrite-install` 流程以仓库为准）。

**建议**：P0——把安装目录这 2 个插件的源码回流入 `.pair/plugins/` 并提交。

### D2（低优先）`tool-flow` 仓库领先安装目录
```
仓库 .pair/plugins/tool-flow/index.js : 3,355 行
安装目录 .../tool-flow/index.js       : 2,672 行（差 683 行，含本轮新增的驱动模式）
```
属正常「开发领先」，仅需发版时执行 `scripts/sync-plugins-to-bin.mjs` + 覆盖安装。

---

## 6. 处置优先级建议

| 优先级 | 事项 | 预期收益 | 风险 |
|---|---|---|---|
| **P0-1** | 回流入库 `tool-art`、`wechat-bridge`（D1） | 消除「覆盖安装丢插件」隐患 | 极低 |
| **P0-2** | 消除 codegraph 双份（F1）：副本改为 import `pkg/codegraph`，或随 R1 归档 | 消除 11K 行漂移副本 | 低（需确认无构建依赖） |
| **P0-3** | `plugins-src` 归档并从主模块排除（R1） | 减少 30 包 / 22.9K 行编译与维护面 | 中（须先确认发布链无依赖） |
| **P1-1** | 清理 `bin/.pair/plugins/tool-flow/.smoke/` + 同步脚本排除测试残留（R5） | 防止 `.git`/锁文件进安装包 | 低 |
| **P1-2** | 根目录 6 个 exe、`nul`、`[object Object]`、日志清理 + `.gitignore`（R6） | 释放 221 MB、消除垃圾 | 低 |
| **P1-3** | `handler` 包 63 个无引用函数标注 `Deprecated` 或移入 legacy（R2） | 防止误改失效代码 | 低 |
| **P1-4** | 修正 `plugins-dist` 文档或补真源（R7） | 消除文档误导 | 低 |
| **P2-1** | 删除 `bridge` 桌面死分支（R4） | -80 行死代码 | 低 |
| **P2-2** | `git.go`/`health_fs.go` 标注「插件遮蔽的兜底实现」（R3） | 防双份实现不同步 | 极低 |
| **P2-3** | Unix PTY 文档标注（U1）；`jsplugin.go:733` 文案更正（U3） | 消除误导 | 极低 |
| **P2-4** | 编排类插件职责边界文档（F3） | 降低认知负担 | 极低 |

---

## 7. 审计边界与未覆盖项（诚实声明）

本次审计**已核实**的部分均有文件级/行级证据；以下**未覆盖或仅抽样**，不应据本报告下结论：

1. **前端死组件**：未重跑完整引用图（历史报告 `dev/v16-t1-frontend-dead-code-report.md` 覆盖过，本次仅核实其唯一发现 `chat-utils.js` 已删除）。`plugins-src/ui-app` 未逐文件审计。
2. **`internal/agent`（159 文件、项目最大包）**：未做逐函数死代码扫描；仅审计了工具注册/嵌入内核/流程相关路径。
3. **`goja` 目录（54 文件）**：为 JS 引擎移植代码，视为第三方同源，未审计冗余。
4. **UI 静态资源与打包产物**：`internal/ui`、`.pair/assets/runtime` 未做重复 bundle 比对。
5. **功能等价性重复**：未做「两个不同名的工具是否做同一件事」的语义去重（如 `grep` vs `fs-api /api/fs/search`），仅核实了**同名**重复。
6. **运行时行为验证**：本审计为**只读静态审计**，未启动服务、未跑 `go build`/`go test`（避免污染基线）；R1 的「不可达」结论基于静态引用，若存在外部构建流水线（CI 之外的手工脚本）需另行确认。

---

## 附录：证据命令清单（可复现）

```bash
# 规模
find . -name '*.go' -not -name '*_test.go' -not -path './_temp/*' | xargs cat | wc -l   # 170090
go list ./... | wc -l                                                                   # 63
go list ./... | grep -c plugins-src                                                     # 30

# R1 不可达
find . -name 'tool-*.exe' | wc -l                                                       # 0
grep -rn "go build.*plugins-src" scripts/ *.mjs *.cmd                                   # 0

# F1 双份与漂移（关键：必须去 CR）
for f in builder.go tsit/api.go; do
  diff <(tr -d '\r' < pkg/codegraph/$f) \
       <(tr -d '\r' < plugins-src/plugins/tool-codegraph/codegraph/$f) | head
done

# R2 归档
grep -rn "RegisterAll(" --include='*.go' . | grep -v "func RegisterAll"                 # 0
grep -rhoE "^func [A-Z][A-Za-z]*" internal/server/handler/*.go | sort -u | wc -l         # 92

# R5 双目录漂移
diff -rq .pair/plugins bin/.pair/plugins | wc -l                                        # 43

# R6 体积
du -sh release bin out tmp screenshots video

# D1 源码/运行时
comm -3 <(ls -1 .pair/plugins | sort) <(ls -1 "/e/Program Files (x86)/PairCode/.pair/plugins" | sort)
```

---

*本报告为只读审计产物，未对仓库做任何修改。所有「建议」均需单独评审后执行。*
