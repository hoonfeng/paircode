# PairCode 在线更新系统设计（基于 GitHub Releases）

> 2026-09-19 设计定稿。分发端点 = 已发布的 GitHub Release（v1.6.3 起每个版本三平台 zip + `digest`）。
> 目标：**检查更新 → 下载 → 校验 → 替换 → 重启**全链路在应用内完成，不依赖任何外部工具（无 `gh`、无 `curl`）。

---

## 1. 现状与约束（实测事实，非推测）

| 事实 | 证据 / 影响 |
|---|---|
| 主程序**没有**任何自更新代码 | 全仓 `selfupdate/checkupdate/updateAvailable` 只命中插件市场的 `npmPluginCheckUpdates`（插件更新，非程序更新） |
| 版本号来自 `-X main.version`，`/api/system/info` 已回传 | `cmd/companion/main.go:16`、`web_server.go:912` → 更新检查的「本地版本」唯一来源 |
| 发布包**无顶层目录前缀**，解压即散开 | `PairCode-1.6.3.zip` 顶层 = `pair.exe` / `assets` / `bin` / `config` / `lib` / `models` / `plugins-src` / `.pair` |
| `api.github.com` 稳定；`github.com` 直连间歇 21s 超时 | v1.6.3 发布时实测：`releases/download` HEAD 超时，`api.github.com` 资产端点 206 正常 |
| `/releases/latest` **直接返回 `digest: sha256:…`** | 实测（2026-09-19，0.78s / 24KB）：三个资产全部带 digest → **校验值无需额外 SHA256SUMS 资产** |
| 资产命名稳定 | `PairCode-<ver>.zip`(win) / `PairCode-linux-<ver>.zip` / `PairCode-darwin-<ver>.zip` |
| 内置 API 必须「注册内核路由表 + core-api 插件清单」两步 | `cmd/companion/kernel_register.go` + `.pair/plugins/core-api/index.js` 的 `ROUTES`；漏第二步 = 接口不挂载 |
| 配置项一律插件 `ctx.registerSettings` 注册 | `internal/core/settings_registry.go`；设置面板无内置 tab（纯 schema 驱动） |
| 用户数据落 `config/`（settings/ai-presets）与 `.pair/`（toolsets/memory/tasks/skills/plugins） | 决定替换时的**保护名单** |

**设计红线**：① 更新内容只来自 GitHub Release 且**必须 sha256 校验**（校验不过一律不落地）；② 只写安装目录内文件，解压防 zip-slip；③ 不覆盖用户配置与用户数据；④ 替换失败必须能回到旧 exe。

---

## 2. 清单来源与格式

### 2.1 主源：GitHub Releases API

```
GET https://api.github.com/repos/{repo}/releases/latest      （channel=stable）
GET https://api.github.com/repos/{repo}/releases?per_page=30 （channel=prerelease：取首个 !draft）
```

`repo` 默认 `hoonfeng/paircode`，可配置。取字段：

| 用途 | 字段 |
|---|---|
| 远端版本 | `tag_name`（`v1.6.3` → 版本 `1.6.3`） |
| 展示 | `name` / `body`(release notes) / `published_at` |
| 资产匹配 | `assets[].name` → 平台映射（见 §3） |
| 下载 | `assets[].id`（→ API 端点）、`assets[].browser_download_url`（→ 回退端点） |
| 校验 | `assets[].digest` = `sha256:<hex>`（**强制**）、`assets[].size` |

### 2.2 自定义源（降级/镜像/内网）

`feedURL` 配置项指向任意 JSON（自建 CDN、内网文件服务器、`file://` 路径用于测试）：

```json
{
  "version": "1.7.0",
  "notes": "更新说明…",
  "publishedAt": "2026-10-01T00:00:00Z",
  "assets": {
    "windows": { "name": "PairCode-1.7.0.zip",        "url": "https://…/PairCode-1.7.0.zip",        "size": 86165907, "sha256": "173cef…" },
    "linux":   { "name": "PairCode-linux-1.7.0.zip",  "url": "https://…/PairCode-linux-1.7.0.zip",  "size": 0, "sha256": "…" },
    "darwin":  { "name": "PairCode-darwin-1.7.0.zip", "url": "https://…/PairCode-darwin-1.7.0.zip", "size": 0, "sha256": "…" }
  }
}
```

两种源归一化成同一个 `Manifest` 结构，后续流程完全共用。

### 2.3 镜像

`mirrorPrefix`（如 `https://ghproxy.net/`）非空时，**下载 URL = mirrorPrefix + 原始 URL**，用于 `github.com` 不可达而镜像可达的场景；`api.github.com` 端点不受影响（它本身稳定）。

---

## 3. 平台 → 资产映射

| GOOS/GOARCH | 资产名 | 包内主程序 |
|---|---|---|
| windows/amd64 | `PairCode-<ver>.zip` | `pair.exe` |
| linux/amd64、linux/arm64 | `PairCode-linux-<ver>.zip` | `pair-linux-amd64` |
| darwin/arm64、darwin/amd64 | `PairCode-darwin-<ver>.zip` | `pair-darwin-arm64` |

匹配规则（宽容）：先按 `PairCode-<suffix>-<ver>.zip` 精确找；找不到再按「名字含平台的 zip」模糊找；仍找不到 → `error: 该平台无可用更新包`。**主程序名按包内实际存在性判定**（`pair.exe` / `pair-linux-*` / `pair-darwin-*`），避免平台表写错导致替换到错误文件。

---

## 4. 流程与状态机

```
idle ──check──▶ checking ──▶ up-to-date（结束）
                    │
                    └─▶ available ──download──▶ downloading ──▶ verifying ──▶ extracting ──▶ ready
                                                                                            │
                                                                                       apply│
                                                                                            ▼
                                                                              applying ─▶ restarting ─▶ (进程退出)
    任何阶段 ──▶ error（附 message；可重试）
```

状态对象（`GET /api/update/status` 返回，前端轮询）：

```json
{
  "stage": "downloading",
  "current": "1.6.3", "latest": "1.7.0",
  "hasUpdate": true,
  "notes": "…", "publishedAt": "…",
  "asset": { "name": "PairCode-1.7.0.zip", "size": 86165907, "sha256": "…", "url": "…" },
  "progress": { "downloaded": 12345678, "total": 86165907, "percent": 14.3, "speedBps": 1048576 },
  "error": "", "message": "下载中…",
  "checkedAt": "2026-09-19T10:00:00Z", "source": "github:hoonfeng/paircode"
}
```

**自动检查**：启动后延迟 20s 做一次（配置 `autoCheck`），结果只写状态不自动下载；`checkIntervalHours` 内不重复请求（记录 `last-check.json`）。

---

## 5. 下载与校验

1. **端点选择**（按序，前者失败或不可用时降级）：
   ① `mirrorPrefix + browser_download_url`（若配了镜像）
   ② `https://api.github.com/repos/{repo}/releases/assets/{id}` + `Accept: application/octet-stream`（**默认首选**，实测稳定、自动 302 到 objects.githubusercontent.com）
   ③ `browser_download_url`（github.com 直连，作为最后回退）
2. **落盘**：`<ConfigDir>/updates/<asset.name>.part` → 校验通过后 `rename` 为 `<asset.name>`。
3. **续传**：`.part` 已存在则带 `Range: bytes=<size>-`；服务返回 200（不支持 Range）则截断重下；206 则续写。
4. **进度**：每 256KB 更新一次 `progress`（前端 700ms 轮询）；同时累计 `speedBps`（滑动窗口）。
5. **停滞检测**：读循环 60s 无字节增长 → 取消 + `error: 下载停滞`。
6. **sha256**：下载流上 `io.MultiWriter(file, hasher)` 同步计算，**不二次读盘**；完成后与 manifest 的 digest 比对（大小写无关、容忍 `sha256:` 前缀）。不一致 → 删除 `.part` + `error: 校验失败`（`requireSha256=false` 时仅告警，但仍记录实际值）。
7. **超时**：`ResponseHeaderTimeout 30s`（连接层）；下载整体不设硬超时（85MB 视带宽）。

---

## 6. 解压（安全）

- 目标：`<ConfigDir>/updates/staging-<ver>/`（每次先清空）。
- 逐条目校验：
  - `filepath.Clean` 后**必须**以 staging 绝对路径为前缀（防 `../` zip-slip）；
  - 跳过符号链接与非普通文件（只接收 `Mode().IsRegular()` 与目录）；
  - 累计解压字节 > `500MB` 或条目数 > `50000` → 中止（防 zip bomb）；
  - 单个文件 > `300MB` → 中止。
- 解压后**冒烟检查**：staging 根必须存在主程序文件（`pair.exe` / `pair-linux-*` / `pair-darwin-*` 之一）、`README.md`；缺失 → `error: 更新包结构异常`。

---

## 7. 替换与回滚（关键设计）

### 7.1 保护名单（永不覆盖）

```
config/**                    用户配置（含 config/updates 自身）
logs/**  screenshots/**     运行数据
_temp/** release/**          构建/验证产物
.pair/toolsets/**            工具集
.pair/memory/**              记忆
.pair/tasks/**               会话任务
.pair/skills/**              技能
.pair/project.md .pair/*.json 项目环境与用户级配置
.pair/plugins/<用户自建>/**   包内不存在的插件目录（zip 只含官方文件 → 天然保留）
```

替换策略 =「**按 zip 条目覆盖 + 保护名单过滤**」：不做整目录删除（避免误删用户文件），包里已移除的旧文件保留（无害，向后兼容）。

### 7.2 步骤（Windows / Linux / macOS 同一套逻辑）

1. `preflight`：staging 主程序存在；`InstallDir` 可写（写探针文件）；staging 与 InstallDir 同卷（否则用复制而非 rename）。
2. 备份主程序：`pair.exe` → `pair.exe.old`（**Windows 允许同卷 rename 正在运行的 exe**，这是无需外部脚本即可替换的关键）。
   - rename 失败（杀软/权限/跨卷）→ **降级路径**：不替换主程序，写 `pending.json`(清单)，由重启脚本在进程退出后复制（见 7.3）。
3. 逐文件复制 staging → InstallDir（跳过保护名单），统计 `written/skipped/failed`。
4. 失败处理（分两类，均如实上报，不静默）：
   - **主程序替换失败**（备份 rename 或 `new → 正式名` rename 失败）→ 立即回滚：`pair.exe.old` → `pair.exe`，状态 `error`（旧版本仍可运行）；
   - **资源文件写失败** → **不回滚主程序**：新主程序可正常启动，资源文件新旧混存向后兼容；失败清单记入 `plan.Missing` 与 `last-apply.json.failed`，状态仍标完成但提示「有 N 个文件写入失败」。
   （实测落地版本即此语义：`internal/update/apply.go` 的 `ApplyStaged`。）
5. 成功：写 `config/updates/last-apply.json`（`{version, appliedAt, files, backup, source}`）。
6. 重启：启动 detached 脚本 → 本进程 `os.Exit(0)`。

### 7.3 重启脚本（本进程退出后接管）

- Windows（`config/updates/restart-<ts>.bat`）：
  ```bat
  @echo off
  ping -n 4 127.0.0.1 >nul            :: 等旧进程退出（约 3s）
  if exist "<dir>\pair.exe.old" del /f /q "<dir>\pair.exe.old" 2>nul
  start "" "<dir>\pair.exe"
  timeout /t 5 /nobreak >nul
  del /f /q "%~f0"
  ```
  （降级路径时，`pair.exe.old` 步骤换成「把 `pair.exe.new` 复制成 `pair.exe`」）
- Linux / macOS（`config/updates/restart-<ts>.sh`）：`sleep 3; rm -f *.old; nohup ./pair-<platform> … &`。
- 脚本以 `Setsid/Detached` 方式启动，**不随父进程退出被杀**（Windows 用 `cmd /c start`；Unix 用 `Setsid: true`）。

### 7.4 回滚兜底

- 下次启动若发现 `pair.exe.old` 与 `last-apply.json` 标记 `fresh=true`（新版本首次启动）→ 启动 30s 后无异常则删除 `.old`；若本次启动即崩溃（日志有 FATAL）→ 用户可手工把 `.old` 改名回去（README/状态面板给出提示文案）。
- `keepBackup=false` 时立即删 `.old`。

### 7.5 前端流程

设置面板「软件更新」tab（插件 schema 自动渲染）+ 「关于」弹窗内嵌更新卡片：
`检查更新` → 有新版则显示版本/说明/大小 → `下载并安装`（进度条 + 速率）→ 校验/解压 → `立即重启`（或 `稍后`，稍后则在状态栏保留提示）。

---

## 8. API 契约（内核路由表 + core-api 装配）

| key | 方法/路径 | 说明 |
|---|---|---|
| `update.check` | `GET /api/update/check?force=1` | 拉清单、比版本，返回状态快照（同步，≤30s） |
| `update.status` | `GET /api/update/status` | 状态快照（前端轮询，含进度） |
| `update.download` | `POST /api/update/download` | 启动后台下载（幂等：已在下载则返回当前状态） |
| `update.apply` | `POST /api/update/apply` `{dryRun?:bool, restart?:bool}` | 替换并（默认）重启；`dryRun` 只报告将写/跳过/失败的文件清单 |
| `update.cancel` | `POST /api/update/cancel` | 取消进行中的检查/下载（删除 `.part`） |
| `update.config` | `GET /api/update/config` | 生效配置（插件 settings + 环境变量覆盖，供前端展示源） |

配置项（插件 `app-update` 注册，存 `pluginSettings.app-update`）：
`feedType(github|custom)`、`repo`、`feedURL`、`mirrorPrefix`、`channel(stable|prerelease)`、`autoCheck`、`checkIntervalHours`、`requireSha256`、`keepBackup`。

**测试/离线覆盖（环境变量，优先级最高）**：`PAIRCODE_UPDATE_FEED`（清单 URL 或本地路径）、`PAIRCODE_UPDATE_DIR`（下载目录）、`PAIRCODE_UPDATE_REPO`。
→ 使本机能在不改代码的前提下验证「有新版本」分支与整条替换链路（指向本地 zip + 正确 digest）。

---

## 9. 安全与容错

- **无远程代码执行**：只下载 zip、只写安装目录文件、重启只启动本安装目录的主程序（脚本内容由程序生成，不含远端字符串——文件名经白名单校验）。
- **校验强制**：`requireSha256` 默认 true；digest 缺失且自定义源未给 sha256 → 拒绝安装（除非用户显式关闭）。
- **HTTPS only**（`file://` 仅测试）；下载 URL 必须与清单同源或镜像前缀同源。
- **并发互斥**：检查/下载/替换共享一把互斥锁，同一时刻只允许一个变更中的状态。
- **失败不留脏**：任何阶段失败都清 `.part`/staging，状态含 `error` 与可读 `message`（写 `config/updates/update.log`）。
- **降级链**：镜像 → API 端点 → 直链；pre-release 频道可选；自定义 feed 支持内网。

---

## 10. 验证方案（落地后执行）

1. **单测**（`internal/update/*_test.go`）：版本比较（`v1.6.3` vs `1.6.10`、前缀 v、非数字后缀）、资产匹配（三平台 + 缺资产）、zip-slip 拒绝、保护名单过滤、sha256 比对（容忍前缀/大小写）、清单解析（GitHub 真响应样本 + 自定义 feed）。
2. **接口**（独立实例 9099）：`/api/update/check` 真实走 GitHub → 期望 `up-to-date`（当前 1.6.3 = 最新）；`/api/update/status` 字段齐全。
3. **有新版本分支**：`PAIRCODE_UPDATE_FEED` 指向本地 feed（版本 9.9.9 + 真实 zip 副本 + 正确 digest）→ 期望 `available` → `download` 全量 85MB 且 sha256 通过 → `extracting` → `ready`。
4. **替换 dry-run**：`POST /api/update/apply {dryRun:true}` → 报告将写文件数、跳过（保护名单）清单、主程序替换可行性。
5. **失败路径**：篡改 digest → 期望 `error: 校验失败` 且 `.part` 被清；断网 → `error` 可读；不在 9090 上做任何验证（用户宿主）。

---

## 11. 真实 GitHub 源端到端验证记录（2026-09-19 已执行）

**目标**：把「检查 → 下载 → 校验 → 替换 → 重启」放到**真实 GitHub Releases** 上完整跑一遍（而非本地 feed），
验证发布包形态下的真实行为，结束后清理测试 tag/Release。

**环境**：隔离实例 `_temp/updg` = v1.6.3 发布包解压出的真实安装形态 + 当前代码构建的 `pair.exe`
（`-ldflags -X main.version=1.6.3`），插件为**真实副本（非 junction）**故可安全真实替换；
更新源 `github:hoonfeng/paircode`、频道 `prerelease`；**9090 用户宿主全程 health=200 未受影响**。

**测试发布**：轻量 tag `v9.9.9` → `1dfcf88e`，`prerelease=true`（★ 不占用 `releases/latest`）；
资产 `PairCode-9.9.9.zip`（86,165,907 B，GitHub digest `sha256:173cefb9…30ff75`）。

| # | 步骤 | 结果 |
|---|------|------|
| 1 | `GET /api/update/check` | 0.83s；`available`，`1.6.3 → 9.9.9`；asset id/size/digest 与发布元数据一致 |
| 2 | `POST /api/update/download` | 86,165,907 B，峰值 10.3 MB/s；`verifying → ready`（sha256 通过）；落盘文件独立 sha256 = 上传源 = 本地源 **三方一致** |
| 3 | 解压 + 包结构冒烟 | `staging-9.9.9` 243 个文件，含主程序 `pair.exe` |
| 4 | `apply {dryRun:true}` | 写 240 / 跳过 3（`config/skills/*`）/ 失败 0 / 164.6 MB；主程序在写清单内 |
| 5 | `apply {dryRun:false}` | `renamed=true`（在线 rename 运行中的 exe）；`pair.exe` 37,779,968 → 37,723,648 B（= 包内 sha）；`pair.exe.old` 备份；`last-apply.json` 落盘；**进程存活 health=200** |
| 6 | 保护名单强证明 | 篡改包内**同样存在**的 `config/skills/emoji-icons/SKILL.md`（写 `USER-CUSTOM-MARKER`）→ 重新 apply（240 写/3 跳过）→ 该文件**原样保留**；对照 `README.md` 被重写为包内版本 |
| 7 | `apply {dryRun:false, restart:true}` | 生成 `config/updates/restart-<ts>.bat` → 进程 `os.Exit` → 脚本等待/删 `.old`/`start` 新程序（继承 `WEB_PORT`）→ **新 PID 33136 起来（health=200，约 2s）**；`/api/update/status` → **404**（证明运行的是替换后的 v1.6.3 发布二进制，无更新接口）；`.old` 已被脚本清除、脚本自删 |
| 8 | 浏览器端到端（CDP，1600×1100） | 关于弹窗 → 卡片「当前 v1.6.3 → 新版本 v9.9.9 … 已就绪（243 个文件）」→ 就绪态复用缓存未重复下载 → 「预览替换清单：将写 240 个文件（164.6 MB），保护名单跳过 3 个」→ **控制台 0 错误**；截图 `screenshots/update-{1-about,2-ready,3-plan}.png` |
| 9 | 清理 | `DELETE /releases/<id>` + `DELETE /git/refs/tags/v9.9.9` → 均 204；随后 tag/release 查询 404、`releases/latest` 恢复 **v1.6.3**（3 资产）、列表仅剩 v1.6.3/v1.6.2；实例重新 check → 「已是最新版本（1.6.3）」（删除即回退，反向验证通过） |

**本次新增经验**：
1. ★ **`restart=true` 必须紧跟下载之后**：apply 成功会 `CleanupStaging` 并把状态重置为 `up-to-date`，
   之后再 apply 得到「更新包尚未下载或已清理」（幂等保护，不是缺陷）。
2. ★ **替换会覆盖 `.pair/plugins/**`**：若测试实例的 `core-api` 是旧包版本（缺 `update.*` ROUTES），
   替换后**运行中的进程**仍能调 API（路由在内存），但**重启后接口消失**。准备实例时应先把当前代码的
   `core-api` 复制进去；真实用户由「新包自带新版 core-api」保证。
3. **自重启的端口继承**：Windows 脚本用 `start ""` 启动新程序 → 继承父进程环境，隔离实例的
   `WEB_PORT=9099` 会被新进程沿用（真实用户默认 9090，不受影响）。
4. **prerelease 频道是发布前演练的正确姿势**：`channel=prerelease` 走 `/releases` 列表 → 能发现测试包，
   且**不污染 `releases/latest`**（stable 用户完全无感）。
5. 下载速率受网络波动影响明显（实测 1.7 → 10.3 MB/s），进度/速率/续传字段刷新正常。

---

## 12. v1.6.4 发布记录（2026-09-19，本功能首次随正式版发布）

**产物**：轻量 tag `v1.6.4` → `d49f87a2`（与 `origin/master` 一致）；正式 Release
`releases/tag/v1.6.4`（id 391793325，`prerelease=false`、`draft=false`）→ `releases/latest` 指向本版。

| 资产 | 大小 | sha256（GitHub `digest` 与本地 `sha256sum` 一致） |
|------|------|---------------------------------------------------|
| `PairCode-1.6.4.zip`（windows/amd64） | 86,272,956 B | `fa4ecfcf0395a2c996b4aa1c5fa59ae3f450855e4fa470fc6d6f7b4f42845221` |
| `PairCode-linux-1.6.4.zip`（linux/amd64） | 89,800,009 B | `d92812f002688338b69c3fdd0b3665f0e86137866740fada56ad1ddb875e2591` |
| `PairCode-darwin-1.6.4.zip`（darwin/arm64） | 91,334,551 B | `2c8432c604ee455798a4921db84231df19dd71e5424ae0c917fb5b61af216041` |

**构建**：`./packager.exe` 全流程（verify-dist-isolation → build-ui → sync-plugins-to-bin →
vite build → sync-embed-shell → verify-release → 三平台编译 → 打包）。版本唯一源是
`packager.json.version`，编译时经 `-ldflags "-X main.version=1.6.4"` 注入三平台二进制。

**发布后验收（真实发布包 `PairCode-1.6.4.zip` 解压启动，端口 9098）**：
- `/api/system/info` → `"version":"1.6.4"`；启动日志 `PairCode IDE 1.6.4 启动中`。
- `core-api 已装配内置接口 61/61（内核表共 61，缺失 0）` → 6 条 `update.*` 路由在**发布包形态**下可用。
- 启动 20s 后自动检查（真实 GitHub，无任何环境变量覆盖）→ `[update] 自动检查：已是最新（1.6.4）`；
  `/api/update/status` → `stage=up-to-date`、`source=github:hoonfeng/paircode`、
  `asset.url = https://github.com/hoonfeng/paircode/releases/download/v1.6.4/PairCode-1.6.4.zip`，
  `asset.sha256` 与上表一致。
- 前端：状态栏**不显示**更新徽标（已最新），控制台 0 错误。

**升级路径验收（1.6.3 实例 = 发布形态 + 新 UI/插件面，端口 9099，真实 GitHub）**：
- 启动 20s 后自动检查 → `[update] 发现新版本 1.6.4（当前 1.6.3）`。
- 状态栏徽标「新版本 v1.6.4 可用」（title「…— 点击打开软件更新」）；点击 → 「软件更新」弹窗显示
  「当前 v1.6.3 → 新版本 v1.6.4」「发现新版本 1.6.4（当前 1.6.3）」+「重新检查 / 下载并安装」，
  「更新说明」即本 release 的 body；控制台 0 错误（截图 `screenshots/webdebug_*.png`）。

**★ 用户升级路径的关键事实（对外必须说明）**：
`/api/update/*` 需要**新版 `core-api` 插件**声明 ROUTES。1.6.3 及更早的发布包里没有这 6 条——
实测旧包实例启动时打印「内核表有 6 个接口未在清单中（未挂载）: update.apply, update.cancel,
update.check, update.config, update.download, update.status」，此时接口 404。
**因此「应用内在线更新」自 1.6.4 起才可用：1.6.3 及更早用户需手动下载一次 1.6.4**，
此后即可用应用内更新升到 1.6.5+（这也是本版把 `app-update` 设置段随包发布的意义所在）。

**v1.6.4 附带的功能增量**（同版发布，非更新引擎改动）：状态栏「新版本可用」全局提示 +
「软件更新」弹窗（`update-state.js` / `UpdateModal.vue` / `StatusBar` 徽标），使新版本在界面上直接可见、
点击即进更新流程。
