# 插件目录（.pair/plugins/）——插件 = 自包含包（源码 + 二进制 + 资源）

> 主程序（companion.exe）只保留框架：装载插件、调度工具、审批、对话。
> 插件的**实现源码 / 独立二进制 / 加载资源**全部位于本目录，用户可自行修改、
> 重新构建、替换——改完重启生效，无需重新编译主程序。

## 目录结构约定

| 形态 | 路径 | 说明 |
|---|---|---|
| 插件包（可装载） | `<name>/package.json + index.js[+client.js]` | 启动扫描装载（LoadGlobalPlugins）；index.js = host 半（api 声明 + 调度），client.js = 浏览器半（UI 渲染） |
| 源码包（不可装载） | `<name>-src/` | ★ 插件**源码**（前端 Vite 工程、工具实现源码等）。装载器按 `-src` 后缀跳过；用户改源码 → 重新构建 → 产物进插件包/assets |
| 独立二进制 | `<name>/bin/<name>.exe` | ★ 依赖 Go 内核的工具独立成**单独二进制项目**（源码在 `cmd/plugins/<name>/`），产物放本插件目录 bin/——插件自包含，改源码重编译即更换实现 |
| 加载资源 | `<name>/assets/` | 二进制/JS 运行所需资源（字体、模板、索引数据等），随插件目录分发 |

## 二进制插件协议（宿主 ctx.binary 服务）

```text
JS 插件 execute → ctx.binary.exec(tool, args[, {timeout}]) → text
  宿主定位 <插件目录>/bin/<插件名>.exe（Windows 加 .exe）
  stdin  JSON {"tool":"binary_strings","args":{...},"root":"<工作区根>"}
  stdout JSON {"ok":true,"text":"..."} | {"ok":false,"error":"..."}
  exit 0（协议错误 exit 2）
```

- `ctx.binary.dir()` → 插件目录绝对路径（JS 可拼接 assets/ 资源路径）
- `ctx.http.register(method, path, fn)` → 注册自定义 HTTP API 路由（接口插件化）：
  - path 以 `/*` 结尾=前缀匹配（如 `/api/ext/*` 匹配其下任意路径）
  - `fn(req) → resp`：req=`{method, path, query, headers, body}`；
    resp=`{status, body, headers}` 或字符串
  - 返回 unregister 函数；插件卸载自动注销；重复 (method, path) 注册报错
  - 插件路由在宿主 mux 之前拦截：命中走插件、未命中走内置 /api/* 与静态文件
  - 实现：internal/agent/ext_routes.go（ExtRouteMiddleware）+ jsplugin.go ctx.http
- `ctx.binary.exec(tool, args, {bin})` → opts.bin 指定**其它插件目录的二进制**
  （跨插件共用统一二进制；如各工具组 JS `{bin:"tool-binary"}` 指向统一宿主
  二进制，无需各自编译）
- 二进制内可用 `os.Executable()` 定位自身目录 → 上级即插件目录（读 assets/）
- 超时默认 60s（opts.timeout 毫秒可覆盖）
- 示例：`.pair/plugins/tool-binary-re/`（6 个逆向工具）——协议实现
  参考 `cmd/plugins/tool-binary-re/main.go`

## 统一宿主二进制（tool-binary）

依赖 Go 内核的工具组（codegraph/lsp/office/memory/verify/binary/git/
vision/screenshot/web-debug/bug/project-info…）由**一个统一二进制**承载：

- 源码：`cmd/plugins/tool-binary/main.go`（import agent → RegisterDefaultTools
  注册全部内置组 → stdin JSON 分发 Registry.Execute）
- 产物：`.pair/plugins/tool-binary/bin/tool-binary.exe`（39MB，CGO=0）
- 各组插件 JS 的 execute 统一 `ctx.binary.exec(t.name, args, {bin:"tool-binary"})`
- **改实现**：改 `internal/agent/*.go`（工具实现库）或 `cmd/plugins/tool-binary/main.go`
  → `go build -o .pair/plugins/tool-binary/bin/tool-binary.exe ./cmd/plugins/tool-binary`
  → 重启 → 全部切换组生效（主程序 exe 无需重编译）
- 会话绑定工具（update_tasks 等 SystemTool、ask_user/task_create）由宿主框架
  执行，二进制排除（excludedTools）；tool-debug 依赖宿主后台进程，保持 hostTool

当前 execute 形态分布（生成器 tool_plugin_gen.go 的 binary 字段控制）：
- `tool-binary`：git/memory/verify/project-info/binary/office/lsp/codegraph/
  codegraph-extra/vision/screenshot/web-debug/bug（14 组）
- `self`：tool-binary-re（独立二进制，自己 bin/ 下的实现）
- `hostTool`：tool-debug（宿主后台进程依赖）、tool-system（会话绑定）

## 三层工具实现（从易到难，用户可改程度递增）

1. **api 壳（hostTool 代理）**：`index.js` 只声明 schema，execute 调
   `ctx.hostTool.exec` 复用主程序内 Go 执行器——只能改描述/参数。
2. **JS 原生实现**：`index.js` 的 execute 直接写 JS（jsToolToGo 支持
   `execute: (args) => result`），用 `ctx.fs` / `ctx.bash` / `ctx.web` 等
   宿主服务实现逻辑——纯 JS 可改，无需重新编译。
3. **独立二进制**：依赖 Go 内核/系统能力（PE 解析、哈希、文档转换、索引等）
   → 独立 Go 项目 `cmd/plugins/<name>/` → 编译产物进 `<插件包>/bin/`
   → JS 壳 `ctx.binary.exec` 调度。改二进制源码 → `go build` → 重启生效。

## 修改指南（用户自助）

### 改工具行为（示例：tool-binary-re）
```bash
# ① 改实现（JS 壳 or 独立二进制源码）
vim .pair/plugins/tool-binary-re/index.js          # 改 api/调度
vim cmd/plugins/tool-binary-re/main.go             # 改真实实现
# ② 重新编译二进制（改了 main.go 时）
go build -o .pair/plugins/tool-binary-re/bin/tool-binary-re.exe ./cmd/plugins/tool-binary-re
# ③ 重启 companion → 生效
```

### 改前端 UI
```bash
# 源码位于 .pair/plugins/ui-app-src/（Vue3 + Vite；node_modules 为 junction
# 指向 cmd/companion/web-ui/node_modules，勿删）
# ① 改组件/壳
vim .pair/plugins/ui-app-src/src/components/*.vue
# ② 构建 7 个 UI 区域插件包（产物 → .pair/plugins/ui-*/assets/）
node scripts/build-ui.mjs
# ③ 构建壳（产物 → .pair/assets/runtime/web/，宿主外部优先加载）
cd .pair/plugins/ui-app-src && node_modules/.bin/vite build
# ④ 重启 companion → 生效
```

### 通用工具组迁移（内置 Go 组 → 磁盘插件）
```bash
go run -tags toolsgen ./dev/tool_plugin_gen   # 幂等：已有插件不覆盖，Go 侧变更重跑同步
```

## 其他目录

- `.pair/assets/runtime/` — 运行时资源（cordis.bundle.js/bridge_node.js/
  ide_ref*/web 前端产物），外部优先 + embed 兜底（见其 README）
- `.pair/toolsets/` — 工具集（插件组合包）
