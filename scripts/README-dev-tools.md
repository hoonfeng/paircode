# dev 工具与诊断脚本索引

本文件索引 2026-09-13 从 `temp/` 整理归档进仓库的开发/诊断脚本。
（`temp/` 本身是本地临时目录，被 `.git/info/exclude` 排除，不入库；只有确认有长期价值的文件才迁移到这里。）

## 本机构建环境（全局安装，已脱离 temp）

| 组件 | 版本 | 位置 |
|---|---|---|
| Go | go1.26.8 windows/amd64 | `D:\Go`（GOROOT） |
| MinGW-w64 | 16.2.0（x86_64-ucrt-posix-seh, Brecht Sanders r1） | `D:\mingw64`（gcc/g++ 在 `bin\`） |
| ONNX Runtime | 1.26.0 | 留档 `D:\onnxruntime\onnxruntime-win-x64-1.26.0`；运行库 `bin\config\onnx\onnxruntime.dll` |
| GOPATH | `C:\Users\<user>\go`（含已下载的 module cache） | GOMODCACHE = `%GOPATH%\pkg\mod` |

系统级环境变量：`GOROOT=D:\Go`、`GOPATH=…\go`、`GOPROXY=https://goproxy.cn,direct`、
`GOSUMDB=sum.golang.google.cn`、`GOTOOLCHAIN=local`、`CGO_ENABLED=1`，`PATH` 追加 `D:\Go\bin;D:\mingw64\bin`。

构建（无需再 `source temp/gotool/env.sh`）：

```bash
CGO_ENABLED=1 go build -o pair.exe ./cmd/companion
```

## 目录说明

| 路径 | 用途 |
|---|---|
| `scripts/env/cgotest/` | CGO/gcc 冒烟测试（独立 module）：`go run .` 输出 `cgo 可用 → 42` 即编译器可用 |
| `scripts/env/onnxtest/` | ONNX Runtime 端到端实证：加载 onnxruntime.dll + bge-small-zh-v1.5 模型并跑一次真实推理。在仓库根执行 `go run ./scripts/env/onnxtest`（复用项目 go.mod 的 onnxruntime_go 依赖） |
| `scripts/env/check-deploy-pair-exe.ps1` | `scripts/deploy-pair-exe.ps1` 的语法 + 正则自检 |
| `scripts/patches/` | 宿主/UI 锚点补丁（Python，可一键重放）：agentloop 写通道审核、面板显隐、专注模式侧栏联动、「继续任务」提示条锚点 |
| `scripts/github-sync/` | 工作区 ↔ GitHub master 差异分析、安全备份、对齐与验证（5 阶段，`SYNC_WT` 指定工作区；删除类操作有白名单 + 沙箱 + dry-run 保护） |
| `scripts/d-sync/` | `E:\paircode-master` ↔ `D:\PairCode` 逐文件差异清单（`build-diff.mjs`）与 D 侧备份（`backup-d.mjs`） |
| `scripts/pair-switch/` | 替换运行中 `pair.exe` 的**旧流程**：`switch-pair.ps1`（停服→备份→替换→SHA 校验→重启）+ `launch-switch.cmd`（脱离 agent 会话异步执行，避免替换过程被会话中断打断）。**新流程见 `scripts/deploy-pair-exe.ps1`** |
| `scripts/deploy-pair-exe.ps1` | 部署**只换 exe**：停服 → 替换 → SHA 校验 → 重启 → 健康检查 |
| `scripts/deploy-pair-full.ps1` | 部署**全量**：按 manifest 替换 `pair.exe` **并**同步 UI 资产（壳 + 区域包），带 sha256 预检、逐文件写入校验，失败自动回滚并重启旧版。源自 `temp/deploy-20260913/deploy-fix.ps1`（已把硬编码的 temp/ 与 D:\PairCode 路径参数化）。`-DryRun` 可预演 |
| `scripts/fixtures/webdebug/` | `web_debug` 误报回归夹具：`big-dom.html`（元素过多）、`empty-dom.html` / `empty2.html`（空 DOM / 极简页）、`gen.js`（生成器） |

## deploy-pair-full.ps1 的 manifest 格式

JSON 数组，每项一个待部署文件：

```json
[{ "src": "<绝对源路径>", "dst": "<绝对目标路径>", "kind": "shell|exe|ui",
   "srcSha": "<大写 SHA256>", "srcSize": 12345 }]
```

预检会因「src 不存在」或「src 的 SHA256 ≠ srcSha」而中止（不触碰目标）。
常用参数：`-Root`（日志/备份/manifest 所在目录，默认 `%TEMP%\paircode-deploy`）、
`-Manifest`、`-ExePath`、`-InstallDir`、`-BaseUrl`（默认 `http://127.0.0.1:9090`）、
`-RollbackRoot`；`-StatsUrl` / `-ExpectShell` 为可选增强校验（不传则跳过）。
`ExePath` / `InstallDir` / `RollbackRoot` 未显式给出时会从 manifest 推导。

## 相关文档

- `docs/d-sync-plan.md` —— E→D 同步方案（范围、顺序、回退）
- `docs/d-sync-diff-report.md`、`docs/d-sync-diff-report-before-fix.md` —— 同步前后差异报告
