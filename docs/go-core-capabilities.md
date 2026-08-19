# PairCode Go 核心能力清单

> 插件可触达的 Go 内核能力全景：接口清单、工具面、能力边界与扩展点。
> 与 [plugin-development.md](./plugin-development.md) 配套阅读（后者是插件写法）。

---

## 目录

1. [架构总览：能力层与挂载权分离](#1-架构总览能力层与挂载权分离)
2. [内核路由表全清单（HTTP 接口）](#2-内核路由表全清单http-接口)
3. [内核工具面](#3-内核工具面)
4. [能力边界：哪些必须留内核](#4-能力边界哪些必须留内核)
5. [扩展点（Go 开发者）](#5-扩展点go-开发者)

---

## 1. 架构总览：能力层与挂载权分离

PairCode 的「一切皆插件」遵循统一模式：**实现留内核（Go），挂载/编排在插件（JS）**。

```
┌─────────────────────────────────────────────────────────────┐
│  浏览器 / Agent                                             │
│    │  HTTP 请求 / 工具调用                                    │
└────┼────────────────────────────────────────────────────────┘
     ▼
┌─────────────────────────────────────────────────────────────┐
│  插件 ext 路由表（ExtRouteMiddleware，宿主 mux 之前拦截）      │
│    ├─ core-api 插件 ctx.kernel.install → 内核接口（Go handler）│
│    ├─ web-api / git-api 等插件 ctx.http/webServer → 插件接口  │
│    └─ 未命中 → 回落到宿主 mux（/ws 等）                        │
└────┼────────────────────────────────────────────────────────┘
     ▼
┌─────────────────────────────────────────────────────────────┐
│  内核能力层（Go）                                            │
│    ├─ 内核路由表  kernel_api.go（~90 条，kernel_register.go 注册）
│    ├─ 宿主执行器  host_executors.go（hostTool 存档，被插件引用）
│    ├─ 框架协议工具 RegisterHostFrameworkTools（SystemTool 组）
│    └─ 共享服务    EventBus / PluginContext / core 包          │
└─────────────────────────────────────────────────────────────┘
```

关键语义：

- **接口**：实现（Go handler）注册进内核路由表；挂载权归 core-api 插件（`ctx.kernel.install`）；插件停用/删除 → 接口消失（随插件生命周期生灭）；
- **工具**：实现（Go Handler）在插件接管时自动存档（`ArchiveHostTool`）；插件注册同名工具接管 agent 可见面，execute 内可调 `ctx.hostTool` 复用宿主能力（编排在插件、能力在宿主）；
- **二进制**：依赖 Go 内核的复杂工具可独立成插件目录下单独二进制项目（`ctx.binary.exec` 调度），主程序只做框架。

---

## 2. 内核路由表全清单（HTTP 接口）

> 全部接口实现保留在 Go（`cmd/companion/kernel_register.go` 注册，`internal/agent/kernel_api.go` 存表）。
> 插件可用 `ctx.kernel.routes()` 实时查询（含 key/method/path/desc），
> 用 `ctx.kernel.install([{key}, ...])` 批量挂载（core-api 插件已挂载全部）。
> 增删 core-api 插件 `ROUTES` 数组 = 增删接口挂载。

### 系统（3）
| key | method | path | 说明 |
|---|---|---|---|
| `health` | GET | `/api/health` | 健康检查（工作区根/文件夹） |
| `system.info` | GET | `/api/system/info` | 系统信息 |
| `system.exec` | POST | `/api/system/exec` | 执行 shell 命令 |

### 文件系统（11）
| key | method | path | 说明 |
|---|---|---|---|
| `fs.drives` | GET | `/api/fs/drives` | 磁盘驱动器列表 |
| `fs.list` | GET | `/api/fs/list` | 目录列表 |
| `fs.read` | GET | `/api/fs/read` | 读文本文件 |
| `fs.write` | POST | `/api/fs/write` | 写文件 |
| `fs.rename` | POST | `/api/fs/rename` | 重命名/移动 |
| `fs.delete` | POST | `/api/fs/delete` | 删除文件/目录 |
| `fs.mkdir` | POST | `/api/fs/mkdir` | 创建目录 |
| `fs.search` | GET | `/api/fs/search` | 文件搜索 |
| `fs.image` | GET | `/api/fs/image` | 图片读取（原始字节）★ 内核保留 |
| `fs.file-info` | GET | `/api/fs/file-info` | 文件类型信息 |
| `fs.hex` | GET | `/api/fs/hex` | 文件十六进制转储 |

### 工作区 / 设置（3）
| key | method | path | 说明 |
|---|---|---|---|
| `workspace` | GET,POST | `/api/workspace` | 工作区查询/变更 |
| `settings` | GET,PUT | `/api/settings` | 设置读取/保存 ★ 内核保留 |
| `ui-assembly` | GET,PUT | `/api/ui-assembly` | UI 装配状态磁盘持久化 |

### 对话（7）
| key | method | path | 说明 |
|---|---|---|---|
| `chat.send` | POST | `/api/chat/send` | 启动 agent 会话 |
| `chat.stop` | POST | `/api/chat/stop` | 停止会话 |
| `chat.answer` | POST | `/api/chat/answer` | ask_user 回答 |
| `chat.approve` | POST | `/api/chat/approve` | 审批结果 |
| `chat.feedback` | POST | `/api/chat/feedback` | 运行时反馈 |
| `chat.rollback` | POST | `/api/chat/rollback` | 回滚到指定消息 |
| `chat.compact` | POST | `/api/chat/compact` | 会话压缩 |

### 会话列表 / 消息（2）
| key | method | path | 说明 |
|---|---|---|---|
| `conversations` | GET,POST | `/api/conversations` | 会话列表/新建 |
| `conversations.byID` | GET,PUT,DELETE | `/api/conversations/*` | 会话详情/重命名/删除（前缀） |

### Tasks / Plan（2）
| key | method | path | 说明 |
|---|---|---|---|
| `tasks` | GET | `/api/tasks` | 任务列表 |
| `taskplan` | GET | `/api/taskplan` | 任务计划 |

### 模型 / 指令（2）
| key | method | path | 说明 |
|---|---|---|---|
| `models` | GET,POST,PUT | `/api/models` | 模型列表读取/全量保存 |
| `instructions` | GET,PUT | `/api/instructions` | 指令读取/保存 ★ 内核保留 |

### 工具配置（2）
| key | method | path | 说明 |
|---|---|---|---|
| `tools` | GET | `/api/tools` | 工具列表 |
| `tools.review` | GET,PUT | `/api/tools/review` | 审核黑白名单配置 |

### MCP / Skills（6）
| key | method | path | 说明 |
|---|---|---|---|
| `mcp.list` | GET | `/api/mcp/list` | MCP 列表 |
| `mcp.save` | POST | `/api/mcp/save` | MCP 保存 |
| `skills.list` | GET | `/api/skills/list` | 技能列表 |
| `skills.read` | GET | `/api/skills/read` | 技能详情 |
| `skills.save` | POST | `/api/skills/save` | 技能保存 |
| `skills.delete` | POST | `/api/skills/delete` | 技能删除 |

### Token / Debug（3）
| key | method | path | 说明 |
|---|---|---|---|
| `tokens.stats` | GET | `/api/tokens/stats` | token 统计 |
| `debug.logs` | GET | `/api/debug/logs` | 调试日志列表 ★ 内核保留（内存环形缓冲） |
| `debug.logs.byID` | GET | `/api/debug/logs/*` | 调试日志详情（前缀） |

### Git（17）
| key | method | path | 说明 |
|---|---|---|---|
| `git.status` | GET | `/api/git/status` | Git 状态 |
| `git.init` | GET | `/api/git/init` | Git 初始化 |
| `git.diff` | GET | `/api/git/diff` | Git diff |
| `git.add` | POST | `/api/git/add` | Git add |
| `git.reset` | POST | `/api/git/reset` | Git reset |
| `git.commit` | POST | `/api/git/commit` | Git commit |
| `git.log` | GET | `/api/git/log` | Git log |
| `git.log.alias` | GET | `/api/git-log` | Git log（避广告拦截器别名） |
| `git.branch` | POST | `/api/git/branch` | Git 分支 |
| `git.checkout` | POST | `/api/git/checkout` | Git checkout |
| `git.stash` | POST | `/api/git/stash` | Git stash |
| `git.stash-list` | GET | `/api/git/stash-list` | Git stash 列表 |
| `git.ignore` | GET | `/api/git/ignore` | Git ignore 读取 |
| `git.discard` | POST | `/api/git/discard` | Git discard |
| `git.push` | POST | `/api/git/push` | Git push |
| `git.pull` | POST | `/api/git/pull` | Git pull |
| `git.remote` | GET | `/api/git/remote` | Git remote |

### 市场（5）
| key | method | path | 说明 |
|---|---|---|---|
| `marketplace.search` | GET | `/api/marketplace/search` | 市场搜索 |
| `marketplace.install` | POST | `/api/marketplace/install` | 市场安装 |
| `marketplace.uninstall` | POST | `/api/marketplace/uninstall` | 市场卸载 |
| `marketplace.refresh` | POST | `/api/marketplace/refresh` | 市场刷新 |
| `marketplace.sources` | GET | `/api/marketplace/sources` | 市场源列表（插件化市场：skill/mcp/plugin） |

### 记忆（3）
| key | method | path | 说明 |
|---|---|---|---|
| `memory.search` | GET | `/api/memory/search` | 记忆搜索 |
| `memory.list` | GET | `/api/memory/list` | 记忆列表 |
| `memory.rebuild` | POST | `/api/memory/rebuild` | 记忆重建 |

### 插件管理（11）
| key | method | path | 说明 |
|---|---|---|---|
| `plugins` | GET | `/api/plugins` | 插件列表 |
| `plugins.detail` | GET | `/api/plugins/detail` | 插件详情 |
| `plugins.action` | POST | `/api/plugins/action` | 启停/删除插件 |
| `plugins.define` | POST | `/api/plugins/define` | 定义插件 |
| `plugins.event` | POST | `/api/plugins/event` | client→host 事件桥 |
| `plugins.invoke` | POST | `/api/plugins/invoke` | client 远程调用 host |
| `plugins.client-failure` | POST | `/api/plugins/client-failure` | client 失败上报 |
| `plugins.client-events` | GET | `/api/plugins/client-events` | host→浏览器事件轮询 |
| `plugins.client-state` | GET,POST | `/api/plugins/client-state` | client 快照上报/读取 |
| `plugins.builtin` | GET,POST | `/api/plugins/builtin` | 内置工具包开关 |
| `plugins.tool` | POST | `/api/plugins/tool` | 工具级开关 |

### 工具集（6）
| key | method | path | 说明 |
|---|---|---|---|
| `toolsets` | GET | `/api/toolsets` | 工具集列表 |
| `toolsets.build` | POST | `/api/toolsets/build` | 工具集构建 |
| `toolsets.export` | GET | `/api/toolsets/export` | 工具集导出 |
| `toolsets.import` | POST | `/api/toolsets/import` | 工具集导入 |
| `toolsets.remove` | POST | `/api/toolsets/remove` | 工具集删除 |
| `toolsets.edit` | POST | `/api/toolsets/edit` | 工具集编辑（add_plugin/rm_plugin/rm_tool/enable_tool） |

> 合计约 90 条。**非 JSON API**（`/ws` 与 `/api/terminal/ws` WebSocket 传输端点）不在内核表，仍在宿主 mux 注册。

---

## 3. 内核工具面

### 3.1 框架协议工具（SystemTool 组）

宿主只直接注册框架协议工具（`RegisterHostFrameworkTools`，会话绑定，随 Agent 会话生灭）：

- `update_tasks` / `update_plan` —— 任务追踪/计划
- `tool_stats` —— 工具调用统计
- `history_*` —— 对话历史查询

供 `tool-system` 插件 `ctx.hostTool` 承载（同名接管时自动存档）。

### 3.2 宿主执行器库（hostTool 存档）

内置 Go 工具组（20 组，`builtinPluginSpecs`）迁移为磁盘插件后，**原 Go 实现自动存档**进
`host_executors.go` 索引（`ArchiveHostTool`）；插件 execute 经 `ctx.hostTool.exec(name, args)`
复用宿主能力（工具编排在插件、底层能力在宿主）：

| 组 | 覆盖工具 |
|---|---|
| `core` | 文件读写/编辑/命令执行（read_file/write_file/edit_file/multi_edit/run_command/move_file/delete_file） |
| `fs-search` | 全文/文件名搜索（search_content/search_files） |
| `git` | Git 操作（git_status/diff/log/show/blame/add/commit/…） |
| `web` | 联网（web_fetch/web_search） |
| `shell` | 后台命令（run_background/read_output/kill_process） |
| `memory` | 跨会话记忆（memory_write/read/list/search） |
| `verify` | 知识库过期验证（memory_verify/project_info_verify） |
| `task` | 任务追踪（update_tasks） |
| `project-info` | 项目知识库（project_info_write/read/list/search/delete/explore） |
| `binary` | 二进制读写 + 逆向分析（inspect_binary/write_binary/binary_strings/find/patch/info/hash/entropy + binary-re 逆向 6 工具） |
| …其余组 | （codegraph/bug/screenshot/office/vision/web-debug/debug/harness 等） |

> 完整存档清单运行时可用 `ctx.hostTool.meta(name)` / 宿主 `HostToolNames()` 查询。
> Registry 中的工具 = agent 可见面（磁盘插件注册，可插拔）；hostExecutors = 宿主实现库（Go Handler，被插件引用）。

---

## 4. 能力边界：哪些必须留内核

插件 ctx 服务覆盖绝大多数场景，但以下能力因**信息形态不匹配**必须留内核（已实测）：

| 能力 | 内核入口 | 为什么插件化不了 |
|---|---|---|
| 原始字节流 | `fs.image`（/api/fs/image） | `ctx.fs` 只文本（readFile→string），图片等二进制必须内核 |
| AppSettings 顶层 | `instructions`（/api/instructions） | `ctx.getSettings` 只插件级 pluginSettings 命名空间 |
| Go 内存结构 | `debug.logs`（环形缓冲） | 进程内内存态，非文件/服务可表达 |
| 输出分离 + 任意 cwd | `system.exec`（/api/system/exec） | `ctx.bash` 合并 stdout/stderr 且限工作区 |
| 系统信息 | `system.info`（/api/system/info） | `ctx.app` 只有 root/folders/configDir 等，缺 hostname 等 |
| 管理面自身 | `plugins.*` / `toolsets.*` | 自锁设计：管理接口由 core-api 装配，停用即面板消失（重启恢复） |
| WebSocket 传输端点 | `/ws`、`/api/terminal/ws` | 非 JSON API，宿主 mux 直接注册 |

> 判断准则：**凡是「信息形态与 ctx 服务一致」的能力都可以插件化**（文本读写、命令执行、HTTP 注册、
> 事件、配置、路由挂载、市场源、工具集模板……）；形态不匹配（字节/内存态/系统信息）才留内核。

---

## 5. 扩展点（Go 开发者）

在 Go 侧新增/扩展内核能力的入口（`internal/agent/` + `cmd/companion/`）：

| 扩展点 | 函数 | 说明 |
|---|---|---|
| 新增内核接口 | `KernelAPIRegister(key, method, path, desc, handler)` | 注册进内核路由表（key 唯一；method 支持逗号分隔；path 以 `/*` 结尾=前缀）；core-api 插件 `ROUTES` 数组加条目即挂载 |
| 存档宿主执行器 | `ArchiveHostTool(tool)` | 插件接管宿主工具时自动存档；也可手动登记供 `ctx.hostTool.exec` 引用 |
| 直接注册插件路由 | `RegisterExtRoute(method, path, handler)` / `RegisterExtRouteAny(path, handler)` | 跳过内核表直接挂 ext 路由（与 ctx.http/webServer 同表） |
| 事件总线 | `EventBus.On/Emit` + `SetClientHook` | `ui:`/`client:` 前缀事件自动进浏览器队列 |
| 动态服务 | `PluginContext.Provide/Get` + `retryWaiting` | 跨插件服务与 D3 等待激活 |
| 框架工具 | `RegisterHostFrameworkTools(registry, root)` | 注册会话绑定工具（update_tasks 等） |
| 工具集模板 | `RegisterTemplate(tpl)` / `registerBuiltinTemplates(h)` | 供 `toolset_build` 动态组合（插件可 `ctx.toolset.registerTemplate` 追加） |
| 市场源 | `RegisterMarketSource(meta)` | 插件化市场（skill/mcp/plugin；插件可 `ctx.market.register` 追加） |
| 后台进程 | `globalBG.start(cmd, dir)` | 全局单例后台进程（跨 agent 轮次存活） |

生命周期纪律：

- 内核路由/服务登记均带 disposer（`ctx.kernel.install` 返回 `{installed,total,missing}`；插件卸载自动摘除）；
- 插件停用/删除 → 其挂载的内核接口、注册的工具、提供的服务、监听的监听器全部回收；
- 磁盘插件（`.pair/plugins/`）启动自动装载（全局）+ 工具集装载（项目）；`cordis.patch.json` 承载静态插件持久化。
