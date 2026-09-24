# PairCode 插件开发文档

> 面向用户的完整插件开发指南。编写插件前请先通读本文件；
> Agent 侧另有精简版技能（`cordis-plugin-development`）供 LLM 写作时参考。
>
> 最近更新 2026-09-11：工具面合并（单工具 + op 分派规范）、`dynamicApproval`
> 动态审批、hostTool 存档语义、`ctx.provider.register` 实现级插槽。
> ★ 2026-09-11：deferred 按需工具（tool_search 发现）已移除——工具直接注册，
> 按需性由「场景（工具集）+ /命令 激活」承担。

---

## 目录

1. [插件是什么](#1-插件是什么)
2. [插件存在的三种形态](#2-插件存在的三种形态)
3. [插件代码形态（host 半）](#3-插件代码形态host-半)
4. [ctx 能力全表](#4-ctx-能力全表)
5. [inject 声明式服务](#5-inject-声明式服务)
6. [插件间协作：动态服务 provide / get](#6-插件间协作动态服务-provide--get)
7. [注册工具（让 Agent 可用）](#7-注册工具让-agent-可用)
8. [HTTP 接口插件化](#8-http-接口插件化)
9. [事件总线与浏览器桥](#9-事件总线与浏览器桥)
10. [系统提示词注入](#10-系统提示词注入)
11. [插件配置（设置面板）](#11-插件配置设置面板)
12. [UI 插件（client 半）](#12-ui-插件client-半)
13. [磁盘插件包结构](#13-磁盘插件包结构)
14. [工具集与 Agent 可见性](#14-工具集与-agent-可见性)
15. [完整示例](#15-完整示例)
16. [版本化工作流](#16-版本化工作流)
17. [最佳实践与常见坑](#17-最佳实践与常见坑)

---

## 1. 插件是什么

插件是运行在 **goja JS 沙箱** 中的 JS/TS 代码，可扩展 PairCode 的任意能力：

- **注册工具**（Agent 可调用的函数）
- **注册 HTTP 接口**（/api/* 或自定义路径）
- **监听/广播事件**、**提供/消费跨插件服务**
- **注入系统提示词**、**注册设置项**、**渲染 UI**（client 半）
- **装配 Agent 循环**、**注册工具集模板**、**贡献市场源**

关键约束：

- 沙箱内**没有 Node.js API**（`require`/`setTimeout`/`fetch`/`process` 等均不可用），一律走 `ctx` 服务；
- 沙箱提供 `console` / `btoa` / `atob` / `TextEncoder` / `TextDecoder` / `CordisApi`（内置真 cordis 运行时）与 `harness` 全局；
- 插件行为与 `bash` 同级信任（沙箱隔离全局，但不是安全边界）。

---

## 2. 插件存在的形态

| 形态 | 存放位置 | 生命周期 | 适用场景 |
|---|---|---|---|
| **动态插件** | 进程内存（`cordis(op=define)` 创建） | 随进程结束消失（需 `cordis.patch.json` 持久化） | 快速试验、调试 |
| **磁盘插件包** | `<InstallDir>/.pair/plugins/<name>/` | 启动自动装载，跨重启存续 | 正式插件、UI 插件（全局） |

> **工具集不是插件形态**。`.pair/toolsets/*.json` 是「插件定义集合 + 工具挑选清单」：
> 文件内嵌插件定义（`plugins: [{name, purpose, code}]`），装载工具集 = 把其中引用的插件装载进
> cordis 运行时，并按工具白名单筛出 Agent 可用工具。插件本体先存在（动态或磁盘包），
> 工具集只是**从中挑选工具**的清单；同一插件可被多个工具集引用。

动态插件是**临时**形式；磁盘插件包是**持久化**形式（`cordis(op=define)` 同步到全局插件包可转持久化）。
工具集本身持久化在磁盘（`.pair/toolsets/`），但它装载的是插件定义，不是独立插件。

---

## 3. 插件代码形态（host 半）

插件代码是 `async 函数体`，`return` 一个插件对象或函数：

```js
// ① 对象形态（推荐）
return {
  name: 'my-plugin',              // 插件名（唯一；缺省会报错）
  purpose: '做什么的',             // 描述（插件面板展示）
  inject: ['fs', 'web'],          // 可选：硬依赖服务（缺失→插件进入 waiting，
                                  //        服务出现后自动激活装载）
  apply(ctx, config) {            // config = cordis(op=run) 传入 / package.json "config"
    // 注册工具 / 监听事件 / 提供服务 / 注册接口 ...
  }
}

// ② 函数形态（cordis 生态惯例）
return function myPlugin(ctx, config) {
  // 函数名作插件名；匿名函数用 dyn id
}
myPlugin.inject = ['fs']          // 函数形态用静态属性声明硬依赖
```

**inject 等待语义（D3）**：插件声明 `inject: ['foo']` 而 `foo` 服务尚未被提供时，插件进入 `waiting` 状态；提供方 `ctx.provide('foo', ...)` 运行后，等待插件**自动激活**——无需重启、无需手动干预。

---

## 4. ctx 能力全表

`ctx` 是插件可用的全部宿主能力入口。**无条件注入**的成员（31 个），另有按需注入的服务（见 §5）。

★ 2026-09 新增能力面（旧文档缺失，务必先知）：

| 新增 | 位置 | 一句话 |
|---|---|---|
| `ctx.subagent` | §4.8 | 派生「独立 agent 回合」（独立系统提示 / 工具白名单 / 来源标注） |
| `ctx.loopFactory.registerAutopilot` | §4.8 | 自主模式决策器（工作 agent 自然结束时宿主回调） |
| `ctx.loopFactory.registerLoop` / `registerHandoff` | §4.5 | JS 循环内核 / JS 会话交接策略 |
| `ctx.commands` | §4.7 | slash 命令注册面（需 `inject:['commands']`） |
| `ctx.llmtrace` | §4.5 | LLM 请求/响应完整追踪 |
| `ctx.hooks` | §4.5 | 循环钩子（PreToolUse / PostToolUse / UserPromptSubmit / Stop） |
| `ctx.activation` | §4.5 | 按需激活入口声明 |

并且**语义已变**：`ctx.loopFactory.register` 从「单槽位后注册覆盖」改为**装配器链（多插件按序叠加）**——
后装载插件不再吃掉先装载插件的装配参数（§4.5 有详解，写装配器前必读）。

### 4.1 生命周期与协作（核心）

| 成员 | 签名 | 说明 |
|---|---|---|
| `ctx.get(name)` | `get(name) → any \| undefined` | 取动态服务（可选依赖；不存在返回 undefined） |
| `ctx.provide(name, obj)` | `provide(name, obj) → void` | 注册动态服务（含函数/数组/嵌套对象）；卸载自动撤销；触发等待者激活 |
| `ctx.on(name, fn)` | `on(event, (payload) => {}) → void` | 订阅事件（`ui:`/`client:` 前缀自动转发浏览器） |
| `ctx.emit(name, payload)` | `emit(event, payload)` | 广播事件（跨插件；`ui:`/`client:` 前缀进浏览器队列） |
| `ctx.effect(fn)` | `effect(() => {})` | 插件卸载时执行清理（关闭连接、释放资源） |
| `ctx.timeout(fn, ms)` | `timeout(fn, ms) → cancel()` | 一次性定时器（卸载自动清理） |
| `ctx.interval(fn, ms)` | `interval(fn, ms) → cancel()` | 周期定时器（卸载自动清理） |

### 4.2 工具注册

| 成员 | 签名 | 说明 |
|---|---|---|
| `ctx.tools.register(toolDef)` | `register({name, description, parameters, execute})` | 注册工具到全局 Registry（同名冲突报错） |
| `ctx.tools.list()` | `list() → [toolName, ...]` | 已注册工具名（只含对 cordis 可见的） |
| `ctx.hostTool.exec(name, args)` | `exec(name, argsObj) → string` | 调用宿主 Go 存档执行器（迁移模式：编排在插件、能力在宿主） |
| `ctx.hostTool.meta(name)` | `meta(name) → {name, description, parameters}` | 宿主工具元数据（声明 schema 时对齐用） |
| `ctx.hostTool.names()` | `names() → [name, ...]` | 宿主已存档执行器清单 |

工具 `execute(args)` 返回 `{text}` 或任意 JSON 值；schema 校验：type 限 `string/number/integer/boolean/object/array/null`，`$ref` 只允许 `#/` 内部引用。

**toolDef 字段全表**（除 name/description/parameters/execute 外均可选）：

| 字段 | 说明 |
|---|---|
| `usageGuide` | 详细使用指导（何时用此工具、注意事项、对比替代方案）——按需注入系统提示，帮模型选对工具 |
| `category` | 分类标签（如 system/file/web），工具面板分组与检索用 |
| `readOnly` | 只读工具标记（审批面提示依据之一） |
| `requiresApproval` | 静态审批：每次调用都需人工确认 |
| `dynamicApproval` | ★ 动态审批：`(args) => bool` 按本次调用参数决定是否走审批门（见 §7.2） |
| `systemTool` | 系统级工具（恒可用，不受工具集白名单收敛影响） |
| `timeout` | 秒数（>0 启用 goja Interrupt 强制中断护栏；默认不限时，执行时长由插件自律） |

### 4.3 HTTP / 实时通道

| 成员 | 签名 | 说明 |
|---|---|---|
| `ctx.webServer.register(route)` | `register({kind:'exact'\|'prefix', path, handler}) → disposer()` | ★ 新接口推荐。Node 风格 `handler(req, res)`；不区分 HTTP 方法（handler 自查 `req.method`）；兼容三参数形态 `register(kind, path, handler)` |
| `ctx.http.register(method, path, fn)` | `register('GET'\|'POST'\|..., '/path', fn) → unregister()` | 旧形态（向后兼容）；`fn(req) → resp`，resp 为字符串或 `{status, body, headers}`；`path` 以 `/*` 结尾=前缀匹配；另有 `list()` |
| `ctx.sse.register(path, handler)` | `register(path, (emit, params) => cleanup)` | SSE 推送接口 |
| `ctx.ws.register(path, handler)` | `register(path, (conn, params) => cleanup)` | WebSocket 接口；`conn.send(data)` / `conn.close()` / `conn.onMessage(fn)` |
| `ctx.kernel.routes()` | `routes() → [{key, method, path, desc}, ...]` | 内核路由表清单（Go 能力层全部接口） |
| `ctx.kernel.installed()` | `installed() → [key, ...]` | 已挂载到插件路由表的内核接口 |
| `ctx.kernel.total()` | `total() → number` | 内核表容量 |
| `ctx.kernel.install(list)` | `install([{key}, ...]) → {installed, total, missing}` | 把内核接口挂到插件路由表（卸载自动摘除） |

### 4.4 进程 / 文件 / 网络（inject 服务之外的内置）

| 成员 | 签名 | 说明 |
|---|---|---|
| `ctx.process.runBackground(cmd, cwd?)` | `→ {id}` | 后台进程（全局单例跨轮次存活） |
| `ctx.process.readOutput(id)` | `→ {output, done, exitErr, status}` | 读后台进程输出 |
| `ctx.process.kill(id)` | `→ void` | 终止后台进程 |
| `ctx.binary.exec(tool, args, opts?)` | `→ text` | 执行插件独立二进制（Go 能力外置）；`opts.bin` 指定二进制名，`opts.timeout` 毫秒 |
| `ctx.binary.dir()` | `→ 插件目录绝对路径` | 定位 `assets/` 等资源 |

### 4.5 配置 / 提示词 / 装配

| 成员 | 签名 | 说明 |
|---|---|---|
| `ctx.registerSettings(schema)` | `registerSettings({key, title, fields}) → {key, value}` | 注册设置项（配置 UI 插件化）；fields 含 `{name, label, type, default, hint, group, options, min, max, step, placeholder}` |
| `ctx.getSettings(key?)` | `→ {field: value}` | 读插件设置（缺省=插件名） |
| `ctx.setSettings(key, value)` | `→ true` | 写插件设置（持久化到 core settings） |
| `ctx.systemPrompt.section({name, order, text})` | `→ void` | 注入系统提示片段（order 默认 100，排序组装） |
| `ctx.systemPrompt.variable({name, provider})` | `→ void` | 注册 `{{name}}` 提示词变量（组装时调用 provider 求值） |
| `ctx.loopFactory.register(apply)` | `register((opts) => overrides\|null)` | ★ 循环装配器**链**：多插件按注册顺序**依次叠加**；同一插件名重复注册 = 替换该项（插件重装载不叠加）；卸载自动摘除。<br>`opts = {system, stepBudget, toolCallBudget, maxToolBudgetSegments, maxContextTokens, autonomous, maxAutonomousMinutes, checkpointInterval, workspaceRoot, reviewMode}`（`maxIterations` 已移除）；返回同形状对象 → 非空字段覆盖，返回 `null`/`undefined` → 不改动。<br>⚠️ 历史语义（2026-09-21 废弃）：单槽位「后注册整体覆盖」——会让后装载插件把先装载插件（agentloop 的 systemAppend / 分段预算 / 审核模式）的装配参数全部吃掉 |
| `ctx.loopFactory.registerLoop(impl)` | `registerLoop({...})` | JS 循环内核（宿主 Run 委托 JS 驱动；`agentloop` 插件即此形态）。未注册 → Go 默认循环 |
| `ctx.loopFactory.registerHandoff({id?, onUserTurn?, onSegment?})` | `→ {id, ok}` | JS 会话交接策略（用户输入 / 段续跑两个跨轮边界：判断 / 生成 / 复用刷新 / 锚点定位 / 视图组装）；未注册或执行失败 → Go 默认实现；卸载自动还原 |
| `ctx.loopFactory.registerAutopilot({id?, decide})` | `decide(req) → {action, task?, assessment?}` | ★ 自主模式决策器（§4.8）：工作 agent 自然结束时宿主调用它决定「继续 / 收尾」；未注册 → 无监督 |
| `ctx.toolset.registerTemplate({id, title, match?, generate})` | `→ void` | 注册工具集构建模板（`generate(profile, requirement)` 返回插件定义数组）。⚠️ 一旦声明 `inject:['toolset']`，本键被工具集服务（§5）**覆盖**（同名 `ctx.toolset`，服务形态后写优先）——要模板注册就别 inject toolset |
| `ctx.market.register({kind, source, name, desc})` | `→ true` | 注册市场源（kind: skill/mcp/plugin）；另有 `unregister(kind)` / `list()` |
| `ctx.registerClientMethod(method, fn)` | `→ void` | host 半暴露方法给浏览器 client 半（`ui.invoke(plugin, method, args)` 远程调用） |
| `ctx.provider.register(name, impl)` | `register(name, (params) => Provider实例) → 还原函数` | ★ 实现级插槽（2026-09）：注册服务商名的 Provider 实现（impl 返回含 `chat(session)` 等能力的对象）；同名覆盖返回还原函数（卸载自动回退 OpenAI 实现）；未命中回退内置协议路由 |
| `ctx.providerFactory.register(name, factory)` | `→ 还原函数` | 注册 provider **装配工厂**（与 `ctx.provider.register` 不同：后者注册协议级 Provider 实现，前者提供预设/服务商表的装配决策） |
| `ctx.aiPresets.get(name)` / `.list()` | `→ preset \| null` / `{name: preset}` | AI 预设**数据面**（Go 只提供表，整套展开决策在插件） |
| `ctx.models.get(provider)` | `→ {baseURL, apiKey, protocol, contextMaxTokens, models} \| null` | 服务商表数据面（装配器做服务商级兜底用） |
| `ctx.hooks.register(event, fn)` | `→ void` | 注册循环钩子：`PreToolUse` / `PostToolUse` / `UserPromptSubmit` / `Stop`；`fn({event, cwd, turn, toolName, toolArgs, toolResult, prompt, message}) → {block, feedback} \| null`（`block` 仅门事件 PreToolUse / UserPromptSubmit 生效，feedback 回灌 LLM）；注册即生效，卸载自动注销 |
| `ctx.prompts.provide({name, text})` / `.remove({name})` | `→ void` | 提供 / 移除命名提示词片段（跨插件共享，归属 `js:<插件名>`） |
| `ctx.llmtrace.register(fn)` / `.unregister(fn)` | `fn(trace) → void` | LLM 请求/响应完整追踪（llm-trace 缓存分析数据面）；卸载自动注销 |
| `ctx.activation.declare({command})` | `→ bool` | 声明按需激活入口：用户执行该 slash 命令时激活本插件 |
| `ctx.handoff.*` | 属性 + 函数 | 会话交接无状态能力（口径桥接 Go 单一真源，避免两侧漂移破坏缓存前缀）：`title` / `marker`(恒空串，向后兼容) / `enabled`、`thresholds(maxCtx?)`、`estimateTokens(...)`、`ruleSummary(...)`、`stripSystem(...)`、`isHandoffText(...)`、`fingerprint(...)`、`keepForRelevance(...)`、`parseRelevance(...)` |
| `ctx.app.workspaceRoot` | 字符串 | 当前工作区根 |
| `ctx.app.root` | 字符串 | 主工作区根（实时） |
| `ctx.app.folders` / `projectName` / `installDir` / `configDir` / `recentProjects` / `workspaceFolders` | 只读属性 | 宿主环境信息（实时读取） |

### 4.6 harness 全局（与 ctx 同作用域）

| 成员 | 签名 | 说明 |
|---|---|---|
| `harness.defineTool(tool)` | 只校验不注册（返回原对象） | 预检工具定义 |
| `harness.registerTool(tool)` | 同 `ctx.tools.register` | 注册工具 |
| `harness.handle(method, fn)` | 同 `ctx.registerClientMethod` | 注册可被调用的方法（Go 侧 Invoke） |

### 4.7 slash 命令注册（ctx.commands，需 inject）

`inject:['commands']` 后可用（未 inject 时为 `undefined`）：

| 成员 | 签名 | 说明 |
|---|---|---|
| `ctx.commands.register({name, description, handler})` | `handler(args) → string` | 注册 slash 命令 `/<name>`；归属本插件，卸载自动注销；`name` 为空或重名报错 |
| `ctx.commands.list()` | `→ [{name, description, owner}, ...]` | 当前命令清单 |
| `ctx.commands.run(name, args)` | `→ string` | 以编程方式执行命令（等价用户敲 `/<name> 参数`） |

★ handler 是**同步**函数（宿主经 VM 锁进入，不能在里面 await 异步链路后才返回）；
返回值会被 `fmt.Sprintf("%v")` 字符串化——要返回结构化内容请自行 `JSON.stringify`；
抛错 → 命令执行失败，错误文本回灌前端。

### 4.8 自主模式：决策器 + 子 agent 回合（★ 2026-09-21 新增）

「策略在插件、能力在宿主」的旗舰范例。宿主只提供两个原语，其余全部由插件决定：

1. `ctx.subagent.run(spec)` —— 同步跑完一个**独立 agent 回合**（自己的系统提示 / 工具白名单 / 来源标注）；
2. `ctx.loopFactory.registerAutopilot({id?, decide})` —— 工作 agent 自然结束时宿主回调，由插件决定「继续 / 收尾」。

#### 4.8.1 决策器何时被调用、返回什么

宿主在 SessionManager 续轮处调用（前置：自主模式开关打开 + 工作 agent 本轮自然结束 + 插件已注册决策器）：
`continue` → 宿主把 `task` 作为新任务唤醒工作 agent；`done` / 未返回 / 抛错 → 整轮结束。

**req**：`{convId, workspaceRoot, round, maxRounds, objective, workerReport, workerTurns, recentHistory}`

- `objective` = 会话首条真实用户消息（供插件构造任务书）；
- `workerReport` = 工作 agent 本轮汇报（自然结束时的最终正文）；
- `workerTurns` = 工作 agent 已完成的任务轮数；`recentHistory` = 工作会话最近消息（时间正序）。

**返回**：`{action: 'continue' | 'done', task?, assessment?}`

- `task` 在 `continue` 时必填（下一步指令）；`assessment` 仅展示/日志，不影响宿主行为；
- 宿主内部字段 `handled` / `error` 用于诊断，插件无需返回。

#### 4.8.2 子 agent 回合 spec（ctx.subagent.run）

| 字段 | 说明 |
|---|---|
| `name` | 来源标注：本回合全部事件带 `AgentName=name`（前端看板分区 / 溯源） |
| `round` | 回合序号（内核只透传标注，语义由插件定义，如「第 N 次监督」） |
| `system` | 系统提示（角色设定，策略由插件提供） |
| `task` | 起始任务（**必填**） |
| `tools` | 工具白名单（`null` / `[]` = 继承会话全部已启用工具；白名单外不可见/不可调用） |
| `extraTools` | 白名单之外额外保留的工具（如插件自带工具） |
| `stepBudget` / `toolCallBudget` | 段内步数 / 工具调用预算（0 = 内核默认） |
| `maxContextTokens` | 上下文窗口（0 = 继承会话配置） |
| `timeoutMs` | 本回合超时（0 = 不限；**始终受会话取消约束**，用户停止即中止） |
| `resultTool` | 可选结构化提交工具 `{name, description, parameters}`：模型一旦调用即记录参数并结束本回合 |
| `convId` | 目标会话（缺省 = 决策器绑定的当前会话） |

**返回**：`{content, segments, submitted, submittedRaw, steps, toolCalls, promptTokens, outputTokens, durationMs, error, ended}`

`ended` 取值：`completed`（自然结束）/ `submitted`（结构化提交）/ `budget`（预算耗尽）/ `error`；
`segments` 与前端展示结构同构（thinking / content / tool_call），便于插件落盘溯源。

**关键语义（写之前必须知道）**：

- **同步阻塞**：宿主在插件回调栈上直接跑完整个子回合（含 LLM 调用与工具执行），期间子 agent 事件实时下发前端；
- 子 agent 循环**强制走 Go 路径**（不能在 JS 回调栈上重入 agentloop 的 JS 循环实现）；
- 先探测再派发：`ctx.subagent.available() → bool`（无会话运行环境时 `run` 抛错）；
- 子 agent 的工具执行走宿主注册表，**审批 / 预算 / 事件复用宿主管线**，插件不必自建执行器。

#### 4.8.3 最小骨架

```js
return {
  name: 'autopilot',
  inject: ['fs'],
  apply(ctx) {
    const log = ctx.logger('autopilot');

    ctx.loopFactory.registerAutopilot({
      id: 'autopilot',
      decide: async (req) => {
        // ① 可选：派生「监督者」回合自行核查（工具白名单收敛，同步返回）
        let extra = '';
        if (ctx.subagent.available()) {
          const r = ctx.subagent.run({
            name: 'supervisor', round: req.round,
            system: '你是任务监督者：核查工作 agent 的产出是否满足目标，只回事实与缺口。',
            task: `原始目标：${req.objective}\n\n工作汇报：${req.workerReport}`,
            tools: ['read', 'grep', 'glob'],
            stepBudget: 20, toolCallBudget: 20, timeoutMs: 600000,
          });
          extra = r.content || '';
          ctx.emit('ui:autopilot:trace', { round: req.round, steps: r.steps, ended: r.ended });
        }
        // ② 决策：continue（给下一步 task）/ done（收尾）
        if (req.round >= (req.maxRounds || 3)) return { action: 'done', assessment: extra };
        return { action: 'continue', task: `继续推进：${req.objective}`, assessment: extra };
      },
    });
  },
};
```

#### 4.8.4 与前端 / HTTP 的约定（实测定型）

- 子 agent 事件带 `AgentName`，前端按 `name` 分区渲染监督轨迹；
- 进度推进用 `ctx.emit('ui:xxx', payload)`；要暴露读接口用 `ctx.webServer.register(...)`，
  **handler 必须返回字符串 body**（返回裸对象 → 响应体为空，前端 `JSON.parse` 直接报错）；
- 决策器在**影子实例**（并行会话 VM 副本）里不写全局槽位：`registerAutopilot` 返回 `{ok:true, shadow:true}`，
  避免会话结束销毁 VM 后全局槽位指向已死 Runtime。

---

## 5. inject 声明式服务

`inject` 数组中声明的服务会作为 `ctx.xxx` 属性注入（未声明访问为 `undefined`）。
宿主内置可用服务 **25 个**：

`fs` `web` `bash` `sse` `ws` `logger` `timer` `tools` `events` `store` `handoff` `app` `workspaceRoot`
`kernel` `market` `process` `mcp` `skill` `toolset` `npm` `plugins` `agents` `llm` `http` `commands`
（另有插件经 `ctx.provide` 注册的动态服务，见 §6；清单见 `PluginHost.availableServices`）

| 服务 | 访问 | 签名 | 说明 |
|---|---|---|---|
| `fs` | `ctx.fs` | `readFile(path)→string` / `writeFile(path, content)` / `appendFile(path, content)` / `exists(path)→bool` / `readdir(path)→[]string` / `stat(path)→{name,size,isDir,mtime}` / `mkdir(path, recursive?)` / `rm(path, recursive?)` | 工作区受限（越界拦截） |
| `web` | `ctx.web` | `fetch(url)→{ok, status, text}` | GET，60s 超时，4MB 上限 |
| `bash` | `ctx.bash` | `exec(cmd, cwd?)→{output, error}` | 120s 超时，cwd 相对工作区根；★ git-bash（非 cmd，`move` 不存在、中文输出可能乱码） |
| `sse` | `ctx.sse` | `register(path, (emit, params)=>cleanup)` | SSE 推送 |
| `ws` | `ctx.ws` | `register(path, (conn, params)=>cleanup)` | WebSocket（conn.send/close/onMessage） |
| `logger` | `ctx.logger(scope)` | `{log, info, warn, debug, error}` | 带插件标签写透宿主 stdout |
| `timer` | `ctx.timer` | `timeout(fn, ms)→cancel` / `interval(fn, ms)→cancel` | 同 ctx.timeout/interval |
| `kernel` | `ctx.kernel` | 见 4.3 | 内核路由表 |
| `market` | `ctx.market` | 见 4.5 | 市场源 |
| `mcp` | `ctx.mcp` | `list()` / `upsert(server)` / `remove(name)` | MCP 服务器管理（插件化配置面） |
| `skill` | `ctx.skill` | `list()` / `write({name, content, description?})` / `remove(name)` | 技能读写（写工作区 `.pair/skills/`） |
| `toolset` | `ctx.toolset` | `list()` / `save(toolset)` / `remove(name)` / `install(...)` | 工具集管理。⚠️ 会**覆盖**默认的 `ctx.toolset.registerTemplate` 形态（同键，见 4.5） |
| `npm` | `ctx.npm` | `install(pkg)` / `uninstall(name)` / `installed()` / `checkUpdates()` / `update(name?)` | 插件运行时 npm 依赖管理 |
| `plugins` | `ctx.plugins` | `reloadDisk()` + `console` / `env` / `platform` / `arch` / `version` / `cwd` / `exit` / `process` / `btoa` / `atob` | 插件运行时工具面（对齐 Node 常用能力） |
| `agents` | `ctx.agents` | `ready(convId?)` / `followup(convId, text)` | 会话唤醒投递（跟随 agent 的 followup 面；旧的子 Agent 派生面已删除） |
| `commands` | `ctx.commands` | 见 4.7 | slash 命令注册面 |

> 还有 `app` / `workspaceRoot` / `store` 三个静态服务：`app` 已无条件注入（见 4.5），
> `workspaceRoot` 可用 `ctx.get('workspaceRoot')` 取（宿主固有服务），`store` 为会话存储（ConversationStore）。

### 5.1 相对路径的根从哪来（★ 2026-09-20 起统一）

`ctx.fs` / `ctx.bash` / `ctx.binary` / `ctx.process` 等一切需要「工作区根」的能力，
都按同一优先级解析（宿主 `jsPluginAdapter.ctxServiceRoot`，单一真相源）：

| 档 | 来源 | 生效场景 |
|---|---|---|
| 1 | 当前**工具调用会话**根 | agent 执行本插件的工具时自动绑定（并发多会话隔离） |
| 2 | UI invoke 绑定根 | 浏览器 `ui.invoke` 发起时刻的当前主工作区 |
| 3 | 装载期会话根 | `cordis(op=run)` 装载本插件的会话工作区（apply 期间及其后无更精确绑定的回调） |
| 4 | 插件上下文根 | 随宿主主工作区切换**实时更新**（`PluginHost.SetWorkspaceRoot`） |
| 5 | 全局主工作区 | 最后兜底；全空 → 显式报错「工作区根为空」，**不会**静默落到别的工作区 |

约定：

- 写工程产物用**相对路径**，让它跟随「当前会话」；要「用户当前所见工作区」用 `ctx.app.workspaceRoot`；
  要插件自身目录（缓存/bundle 资源）用插件目录语义，别拿工作区根凑。
- 不要在 `apply`（装载）里写工程产物：装载期的根是第 3/4 档，不一定是用户正在操作的会话。
- 历史坑（已修）：宿主根只在 `NewPluginHost` 时快照、`cordis(op=run)` 不绑定会话根、
  `ctx.fs` 自己维护一份手写根解析副本 → 插件产物写进了「IDE 启动时那个工作区」。

---

## 6. 插件间协作：动态服务 provide / get

```js
// 插件 A（提供方）
apply(ctx) {
  ctx.provide('greeter', {
    greet(name) { return `hello from A: ${name}` },
    version: 42,
    meta: { tags: ['a', 'b'] },
    nested: { value() { return 123 } },
  })
}
```

```js
// 插件 B（消费方）——可选依赖
apply(ctx) {
  const svc = ctx.get('greeter')          // 无则 undefined，判空使用
  if (svc) console.log(svc.greet('B'))    // 'hello from A: B'
}
```

```js
// 插件 C（消费方）——硬依赖：inject 等待
return { name: 'C', inject: ['greeter'], apply(ctx) {
  // A 未运行 → C 进 waiting；A 提供 greeter 后 C 自动激活
  ctx.get('greeter').greet('C')
} }
```

- 服务对象**含函数可跨插件调用**（goja Export/ToValue 保留 Callable）；
- 提供方卸载 → 服务自动撤销；等待者若因服务消失进入 waiting，再次提供可重新激活；
- 同类需求优先用动态服务，而不是 HTTP 往返或复制代码。

---

## 7. 注册工具（让 Agent 可用）

```js
apply(ctx) {
  ctx.tools.register({
    name: 'my_tool',
    description: '做什么（给 LLM 看的说明，写清楚参数用途）',
    parameters: {
      type: 'object',
      properties: {
        arg1: { type: 'string', description: '参数说明' },
        count: { type: 'integer', description: '数量', default: 1 },
      },
      required: ['arg1'],
    },
    async execute(args) {
      // args = { arg1, count }
      return { text: '结果文本（给 LLM 看）' }
    },
  })
}
```

要点：

- **同名冲突会被拒绝**（不能覆盖宿主或其他插件工具）——换名或先 `cordis(op=stop)` 占用方；
- schema type 限 `string/number/integer/boolean/object/array/null`；
- **Agent 是否可用由工具集决定**（见 §14）：工具注册进 Registry 只是「存在」，加入工具集后 Agent 才能调用；
- 结果有体积上限：不要返回完整大文件/大数组，只回摘要或路径引用；
- `ctx.hostTool.exec(name, args)` 可在插件里复用宿主 Go 执行器（迁移模式）；
- 补 `usageGuide`（何时用/注意事项）——与 description 一起是模型选工具的主要依据。

### 7.1 单工具 + op 分派（2026-09 工具面规范）

工具面合并后的官方设计规范：**一个能力域 = 一个工具 + `op`（或 `mode`）参数分派**，
不拆成一堆同族小工具（避免工具面膨胀、模型选择困难）。范例见
`.pair/plugins/tool-memory/index.js`（`memory(op=write/read/search/list/delete)`）：

```js
const impls = { write, read, search, list, delete: del }
ctx.tools.register({
  name: 'memory',
  description: '跨会话记忆统一入口（.pair/memory/）。op=write 写入/read 读/search 搜/list 总览/delete 删。',
  parameters: {
    type: 'object',
    properties: {
      op: { type: 'string', enum: ['write', 'read', 'search', 'list', 'delete'], description: '操作' },
      // ...各 op 参数（一个工具内合并且均在 description 里逐 op 说明）
    },
    required: ['op'],
  },
  dynamicApproval: (args) => args && (args.op === 'write' || args.op === 'delete'),
  execute: (args) => { const fn = impls[args.op]; return fn ? fn(ctx, args) : 'op 无效（可用 write/read/search/list/delete）' },
})
```

- `op` 用 `enum` 限制取值；description 里**逐 op** 说明用途与必填参数；
- 未匹配 op 时返回明确引导（列出可用 op），不要抛裸错；
- 只读 op 与写 op 混在同一个工具时，用 `dynamicApproval` 按 op 决定审批面（见下）；
- 合并/改名工具后记得同步工具集白名单（旧名移除、新名加入，见 §14）。

### 7.2 动态审批（dynamicApproval）

```js
ctx.tools.register({
  name: 'memory',
  // ...
  dynamicApproval: (args) => args && (args.op === 'write' || args.op === 'delete'),
})
```

- 每次调用前宿主用**本次参数**求值该回调：`true` → 走审批门（等人工确认）；`false` → 直接执行；
- 与 `requiresApproval` **互斥使用**：`requiresApproval: true` = 每次都必须审批；
  「只读免批、写删必批」场景用 `dynamicApproval`，别两个都设；
- 回调在 VM 锁保护下执行，可安全读取 `args`。

### 7.3 宿主工具存档（hostTool 语义）

插件声明与宿主**同名**工具时（迁移模式的接管路径）：

1. 装载期 `claimTool` 检查归属：宿主同名实现**自动存档**进 hostTool 档案
   （`ArchiveHostTool`——内存档案，进程退出即清空）；
2. 插件 `execute` 内可 `ctx.hostTool.exec(name, args)` 调回宿主实现——
   「编排在插件、能力在宿主」；
3. `ctx.hostTool.names()` 列出全部存档；`ctx.hostTool.meta(name)` 取元数据对齐 schema。

> 反向案例：某个宿主能力**没有**插件同名声明时不会被自动存档（如 `load_skill`
> 内路由的 `load_skill_resource`）——这类由宿主启动期显式存档，插件侧无感直接 `exec`。

---

## 8. HTTP 接口插件化

```js
// ① 新形态（推荐）：Node 风格 handler
apply(ctx) {
  const dispose = ctx.webServer.register({
    kind: 'exact',            // exact=逐字匹配 | prefix=前缀匹配（path/<anything>）
    path: '/api/my/hello',
    handler(req, res) {
      // req = { method, url, path, query, headers, body, httpVersion, json(), on('data'|'end') }
      //   ★ req.query 是 RawQuery 字符串（非对象），需自行解析
      // res = { writeHead(status, headers), setHeader(k,v), write(chunk), end(body), on(...) }
      if (req.method === 'GET') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ message: 'hello from plugin' }))
      } else {
        res.writeHead(405)
        res.end('Method Not Allowed')
      }
    },
  })
  // 插件卸载自动注销；也可手动 dispose()
}
```

```js
// ② 旧形态：注册自定义 JSON 接口
apply(ctx) {
  ctx.http.register('POST', '/api/my/echo', async (req) => {
    // req = { method, path, query, headers, body, json() }
    return { status: 200, body: req.body }
  })
}
```

```js
// ③ 挂载内置内核接口（core-api 插件的做法）
apply(ctx) {
  const res = ctx.kernel.install([{ key: 'health' }, { key: 'system.info' }])
  // → { installed: 2, total: 2, missing: 0 }
}
```

- 插件路由在宿主 mux **之前**拦截（未命中插件路由才走内置 /api/* 与静态文件）；
- 重复 `(method, path)` 注册报错；插件卸载自动注销全部路由；
- 注意 `req.query` 是 **RawQuery 字符串**（非对象），需自行解析。

---

## 9. 事件总线与浏览器桥

```js
apply(ctx) {
  // 订阅（跨插件可见）
  const cancel = ctx.on('conversation:created', (payload) => {
    console.log('新会话:', payload.id)
  })
  // 广播
  ctx.emit('my-plugin:did-something', { detail: 1 })
  // ui:/client: 前缀事件 → 自动转发浏览器 client 半
  ctx.emit('ui:toast', { text: '插件提示' })
}
```

事件命名建议带插件前缀（`my-plugin:xxx`）避免碰撞。

---

## 10. 系统提示词注入

```js
apply(ctx) {
  // 追加系统提示片段（order 决定位置，越小越靠前，默认 100）
  ctx.systemPrompt.section({
    name: 'my-plugin-rules',
    order: 50,
    text: '当用户提到 X 时，请优先使用 my_tool 工具。',
  })
  // 注册动态变量 {{workspace_name}}（组装提示词时求值）
  ctx.systemPrompt.variable({
    name: 'workspace_name',
    provider: () => ctx.app.projectName || 'unknown',
  })
}
```

---

## 11. 插件配置（设置面板）

```js
apply(ctx) {
  // 注册设置项（出现在设置面板「插件」分组）
  ctx.registerSettings({
    key: 'my-plugin',                 // 唯一键（= 插件名惯例）
    title: '我的插件',
    fields: [
      { name: 'apiKey', label: 'API Key', type: 'text', hint: '必填' },
      { name: 'maxRetry', label: '最大重试', type: 'number', default: 3, min: 0, max: 10 },
      { name: 'enabled', label: '启用', type: 'boolean', default: true },
      { name: 'mode', label: '模式', type: 'select', options: ['fast', 'safe'] },
    ],
  })
  // 读/写（key 缺省=插件名）
  const cfg = ctx.getSettings('my-plugin')       // { apiKey, maxRetry, enabled, mode }
  ctx.setSettings('my-plugin', { apiKey: 'xxx' }) // 整对象覆盖该 key 的设置（传全量字段）
}
```

field 支持：`name`（键）/ `label`（展示名）/ `type`（text/number/boolean/select/...）/ `default` / `hint` / `group` / `options` / `min` / `max` / `step` / `placeholder` / `binding`。

---

## 12. UI 插件（client 半）

含 client 半的插件自动 **global 作用域**（跨工作区生效）。client 半形态：

```js
// client 半：(ui) => void
(ui) => {
  // 收 host 事件（ui:/client: 前缀）
  ui.on('client:my-plugin:update', (payload) => { ... })

  // 发事件回 host（host: 前缀给 host 半消费：ctx.on('host:xxx'))
  ui.emit('host:my-plugin:clicked', { x: 1 })

  // 远程调用 host 半方法（host 侧 ctx.registerClientMethod 注册）
  const result = await ui.invoke('my-plugin', 'getData', { page: 1 })

  // 失败上报（render/guard/boot 阶段）
  ui.reportFailure('render', '渲染失败原因')

  // 注册自定义面板（渲染进插件面板「客户端面板」区）
  ui.registerPanel({
    id: 'my-panel',
    title: '我的面板',
    icon: 'svg...',
    render(el, ui) { el.innerHTML = '<div>面板内容</div>' },  // el 为容器 DOM
    props: { field: 'type' },   // 面板数据契约（轻量 Slot）
  })

  // 注册**中间区域视图**（在 IDE 主内容区 tab 栏开一个 tab，与 对话/编辑器/市场/工具集 同级）
  ui.registerView({
    id: 'my-view',
    title: '我的视图',
    icon: 'svg...',
    order: 30,          // tab 顺序（小者靠前；缺省 100）
    open: true,         // true = 默认打开为「后台 tab」：tab 出现但不抢占对话主视图
    render(el, ui) { el.innerHTML = '<div>视图内容</div>' },  // 返回 cleanup 可清理
  })

  // 调后端 API（受限）
  const data = await ui.http.get('/api/plugins')       // 或 ui.http.post(path, body)
}
```

> ⚠️ **渲染纪律（★ 2026-09-21 实测）**：渲染模型/用户产出的**文本**一律 `el.textContent = text`
> （LLM 输出常含 `<` `>` `&`，用 `innerHTML` 会被当标签解析 → 排版错乱/内容消失）；
> `innerHTML` 只用于插件自建、可信的结构。/ `render(el, ui)` 可返回 cleanup 函数
> （`registerPanel` 打开面板即挂载、`registerView` 切 tab 不卸载，关闭 tab 或卸载插件才清理）。

**`registerPanel` vs `registerView`（2026-09 新增）**：

| | `registerPanel` | `registerView` |
|---|---|---|
| 出现位置 | 插件面板（壳级逃生口浮动窗口）内的「客户端面板」小 tab 区 | 主内容区（`.main-area`）tab 栏，与 对话/编辑器/市场/工具集 同级 |
| 适用场景 | 管理/总览类面板（藏在插件面板里） | 需要**长期占用主工作区**的界面（编辑器式工作台） |
| 可关闭 | 跟随插件面板 | 每个 tab 有 ×，关闭状态持久化（`localStorage: viewOpen:<插件>:<视图 id>`） |
| 与对话 | — | 可**并排**：tab 栏「并排对话」按钮 → 主区分左右两栏（对话 + 当前视图，可换边）；再点回单栏 |
| 挂载策略 | 打开插件面板即挂载 | **懒挂载 + 保持**：首次激活才挂载（render(el, ui)），之后切 tab 不卸载（保住 3D 视角/滚动等状态），关闭 tab 或卸载插件才 cleanup |

> 视图与对话并排时的布局状态在 `ui-state.js` 的 `layout` 服务（`toggleSplit()` /
> `setSplitChatSide('left'|'right')` / `openViewTab()` / `closeViewTab()`），
> 不持久化（临时视图态，刷新回单栏）。

**预定义 UI 槽位**（`ui.registerSlot({slotId, ...})` 注册占用；替换型 single 区域内下拉切换占用者，叠加型 list 勾选激活）：

- **替换型（single）**：`titlebar` / `activitybar` / `sidebar` / `editor` / `right-panel` / `chat` / `statusbar`
- **叠加型（list）**：`overlay`（浮动层）/ `titlebar-right` / `activitybar`（图标列）/ `editor-toolbar` / `chat-tools` / `statusbar-items`

> 同一 slotId 可同时存在 single 与 list 两类占用，机制按 kind 分流。

事件流：host→浏览器 经 `/api/plugins/client-events` 每 2s 轮询取增量；浏览器→host 经 `/api/plugins/event`。

### 12.1 三种挂载点怎么选（★ 2026-09 实战补充）

| | `ui.registerSlot({slotId})` | `ui.registerView({...})` | `ui.registerPanel({...})` |
|---|---|---|---|
| 出现位置 | 预定义槽位（如 `titlebar-right`） | 主内容区 tab 栏 | 插件面板内的「客户端面板」区 |
| 形态 | **状态徽标 / 小组件**（list 叠加型：多占用者共存勾选） | **完整工作台**（长期占用主区，可并排对话） | 管理/总览小面板（藏在插件面板里） |
| 数据密度 | 极低（一行文字 + 一个脉冲点） | 高（卡片列表 / 表格 / 详情） | 中 |
| 实例 | autopilot `.ap-tbadge`（「自主模式 · N 轮」，运行中带 `running` 类 + 脉冲点） | autopilot `registerView('autopilot-board')` 监督回合看板 | 插件自身的设置/统计面板 |

**选型经验**：状态类插件「徽标 + 独立视图」双挂载最实用 —— 徽标负责**常驻可发现性**
（不抢主区、点击即切到视图），视图负责**深度信息**（可滚动、可刷新、可并排对话看）。
徽标点击切换视图用 `layout.openViewTab(viewId)`（`ui-state.js` 的 layout 服务）。

### 12.2 状态型 UI 插件的运行态契约（★ 2026-09-21 修复，重要）

前端「某会话正在运行」的**唯一真相**是 `window.__state.agentRunningByConv[convId]`
（以及 `loadingByConv`），它由后端 WebSocket 的 **status 帧**写入：

```jsonc
{ "type": "status", "runningConvs": ["conv_xxx"], ... }   // 后端 buildStatusPayload（internal/agent/event_ws.go）
```

**时序坑（本次踩到并已修）**：自主模式下 `done` 事件**早于会话真正结束** ——
工作 agent 每轮自然结束都发一次 `done`，之后仍有监督者回合与收尾。WS 端点在收到
`done`/`error` 后延迟 50ms 推一份 status（当时会话确实还在跑 → 含本会话，正确）；
而会话**真正结束**时（Loop goroutine 的 defer 里 `Running=false`）原本不再产生任何事件，
于是前端最后一次 status 仍把它标成运行中 → **「运行中」永久残留**（徽标一直脉冲、
输入框禁用、会话列表状态错）。

修复机制：`EventStatusRefresh`（`internal/agent/loop.go` 事件常量）

1. `session_manager.go` 会话 defer 中，置 `Running=false` 之后、`close(Events)` 之前
   **非阻塞**补发该事件（缓冲满丢弃无妨，下轮 done 仍会校正）；
2. `event_ws.go` 收到它**只补推一条 status 帧**，事件本体不下发前端（避免未知消息）；
3. `agent.go` 的全局监听显式跳过它（内部信号不进宿主 `OnEvent`）。

**插件侧纪律**：不要把「运行中」当唯一真相渲染最终态；卡片/轮次等**事实数据**一律来自
持久化接口（如 `GET /api/autopilot/rounds?convId=`），运行态只用来控制「进行中」的
视觉提示。这样即使事件偶发丢失，刷新/切会话后界面仍自洽。

### 12.3 UI 插件验证方法（CDP 无头 Chrome，2026-09 定型）

1. **环境隔离**：一律用**非默认端口 + 独立二进制**（宿主 9090 不动），
   如 `WEB_PORT=9097 ./companion_test.exe`；验证脚本放 `_temp/*.cjs`（配 `_temp/cdp-lib.cjs`）。
2. **不动全局配置**：测试会话的模型只改**会话级 preset**
   （`PUT /api/conversations/<id>` body `{preset:'基元flash'}`），不要写 `config/settings.json`。
3. **驱动流程**：`Page.navigate` → 轮询等 `window.__PAIRCODE_CORE.layout` 就绪 →
   `fetch('/api/chat/send', {convId, message, autonomous:true, workspaceRoot})` → 断言 DOM/state。
4. **状态类断言必须高密度采样（2s）**：运行态是瞬态（本轮实测 `runFlag` true→false
   只隔一个采样间隔）。5s 采样会直接漏掉整个运行窗口、得出错误结论 ——
   verify5（5s）判「未观察到运行中」，verify6（2s，独立会话）才拿到
   `2s:false → 4s:true + 徽标 .ap-tbadge.running → 8s:false → 10s:收尾文案+卡片` 的完整证据链。
5. **留证据**：每轮落盘 `_temp/*-result.json`（采样序列 + 控制台错误）+ `Page.captureScreenshot` 截图。

---

## 13. 磁盘插件包结构

正式插件 = 插件包目录（`<InstallDir>/.pair/plugins/<name>/`），启动自动装载：

```
<name>/
├── package.json     # 插件元数据（必填 name/main）
├── index.js         # host 半源码（必填；同 §3 形态）
└── client.js        # client 半源码（可选；(ui) => void）
```

`package.json` 字段：

```json
{
  "name": "my-plugin",
  "main": "index.js",
  "client": "client.js",
  "purpose": "插件用途描述",
  "scope": "global",
  "type": "plugin",
  "version": "1.0.0",
  "config": { "any": "装配参数，apply(ctx, config) 第二参" }
}
```

- `scope`：`global`=跨工作区（UI 类插件默认）；`project`=项目级；
- 含 client 半自动 global；
- 插件包内可带 `assets/`（资源）与 `bin/`（独立二进制，经 `ctx.binary.exec` 调用）；
  **2026-08 起官方插件全量 JS 化，不再携带 bin/ 独立二进制**——`ctx.binary.exec` 无
  exe 时**回退宿主内嵌内核**同名执行器（能力不变，见 tool-binary/tool-web 等现役插件）；
  plugins-src/ 下的 Go 源码为独立二进制归档源（改实现重编译即更换）；
- 项目级持久插件也可放 `.pair/plugins/`（工作区）——重启自动装载；
- 动态插件（cordis(op=define)）会同步为插件包（全局插件包目录），重启存续。

---

## 14. 工具集与 Agent 可见性

- **工具集**（`.pair/toolsets/*.json`）是「插件定义集合 + Agent 可用工具挑选清单」：内嵌插件定义（name/purpose/code），装载时把插件装载进 cordis 并按**工具白名单**筛出 Agent 可用工具；
- 插件面板里的**工具对勾 = 是否加入工具集**：
  - 勾上 → 加入工具集 → **Agent 可用**；
  - 已加入的工具去掉勾 → 从工具集移除 → Agent 不再自动调用（但工具仍注册在 Registry，可被插件/前端引用）；
- 装载 ≠ 可用：全部插件照常装载（cordis 可见可管理），未加入工具集的插件工具对 Agent 隐藏（Enabled=false）；
- 恢复 = 工具集编辑（`toolset_edit add_plugin` 加入工具集）。

---

## 15. 完整示例

一个「状态服务」插件：提供服务 + 注册工具 + 注册接口 + 注入提示词。

```js
// host 半（index.js）
return {
  name: 'state-service',
  purpose: '示例：跨插件状态服务 + 工具 + 接口',
  inject: ['logger', 'kernel'],
  apply(ctx) {
    const log = ctx.logger('state-service')

    // 内存状态
    const state = { count: 0 }

    // ① 提供动态服务（可含函数）
    ctx.provide('state', {
      get() { return { ...state } },
      inc() { state.count++; return state.count },
    })

    // ② 注册工具（Agent 可调；加入工具集后生效）
    ctx.tools.register({
      name: 'state_inc',
      description: '状态计数 +1 并返回当前值',
      parameters: { type: 'object', properties: {} },
      execute: async () => {
        const v = state.inc()
        return { text: `当前计数: ${v}` }
      },
    })

    // ③ 注册接口
    ctx.webServer.register({
      kind: 'exact',
      path: '/api/state',
      handler(req, res) {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ count: state.count }))
      },
    })

    // ④ 注入系统提示
    ctx.systemPrompt.section({
      name: 'state-hint',
      text: '状态服务可用：state_inc 工具可操作全局计数。',
    })

    // ⑤ 挂载一个内核接口示例
    const res = ctx.kernel.install([{ key: 'health' }])
    log(`内核接口: ${res.installed}/${res.total}`)
  },
}
```

```js
// client 半（client.js，可选）：面板显示实时计数
(ui) => {
  ui.registerPanel({
    id: 'state-panel',
    title: '状态面板',
    render(el, ui) {
      el.innerHTML = '<div>状态面板加载中…</div>'
      ui.invoke('state-service', 'getState').then((s) => {
        el.innerHTML = `<div>count = ${s.count}</div>`
      })
    },
  })
}
```

---

## 16. 版本化工作流

```text
1. 编写代码 → cordis(op=define)（返回稳定 id dyn-<n>）
2. cordis(op=run) id=dyn-<n>       → 装载运行（验证）
3. 修改代码 → cordis(op=define) pluginId=dyn-<n> code=...（追加新版本）
4. cordis(op=run) id=dyn-<n>       → 装载最新版（自动先停旧实例）
5. 回滚 → cordis(op=run) id=dyn-<旧版本号>（指定精确版本）
6. 固化 → 写成磁盘插件包（.pair/plugins/），重启存续
```

- `cordis(op=inspect) id=xxx` 看版本链与状态；`version=vN` 读指定版本源码与诊断；
- `cordis(op=stop)` 停止插件（定义保留可再 run）；`cordis(op=undefine)` 永久删除（含磁盘包）；
- 插件状态：`stopped / running / waiting（缺依赖服务）/ rejected（装配拒绝）/ failed（装载失败）/ cancelled`。

---

## 17. 最佳实践与常见坑

### 最佳实践

- 先 `cordis(op=services)` / `cordis(op=query)` 查精确服务签名，不要凭记忆写；
- 跨插件共享逻辑用 `ctx.provide/get`（含函数），比 HTTP 或复制代码干净；
- 回调（事件/timer/工具 execute）在 VM 锁保护下执行，可安全访问插件闭包变量；**不要在回调外持有 goja 值跨 goroutine 使用**；
- 命名加插件前缀：工具名、事件名、服务名、设置 key；
- 需要清理的资源（定时器/连接）用 `ctx.effect` 或返回 cancel/disposer；
- 长耗时任务用 `ctx.process.runBackground`（后台进程跨轮次存活）或 `ctx.binary.exec`（独立二进制）；
- 新工具优先设计成「单工具 + op 分派」（§7.1）；混合读写语义用 `dynamicApproval` 控审批面；
- 改工具名/合并工具后，务必同步工具集白名单与相关测试引用（见 §14）。
- 装配型插件（`ctx.loopFactory.register`）只**覆盖自己关心的字段**，别整包返回 `opts` 的半成品——
  装配器链是「按序叠加」（§4.5），返回 `null` 就是「本轮不改动」，比返回近似值安全；
- 需要「插件自己当监督者」时，把核查逻辑放进 `ctx.subagent.run`（独立思考 + 收敛工具），
  决策器 `decide` 只做轻量判断——同步阻塞语义下，`decide` 里做的每件事都在占用户等待时间（§4.8）。

### 常见坑

| 现象 | 原因 | 处理 |
|---|---|---|
| `cordis(op=run)` 报求值失败/语法错误 | JS/TS 语法或顶层异常 | 修代码 → define append → run；TS 类型注解可用 |
| 插件进入 `waiting` | inject 服务未就绪 | 等服务提供方运行；或改用 `ctx.get` 判 undefined |
| 工具注册报同名冲突 | 工具名被占用 | 换名，或 `cordis(op=stop)` 占用方 |
| `apply` 失败（diag 可见） | 运行期异常 | `cordis(op=inspect) id=xxx` 看 diag/lastError 定位 |
| 插件 stop 后工具还在 | 未走 Unload 回收 | `cordis(op=stop)` 正确回收；自注册全局资源用 `ctx.effect` 清理 |
| `req.query` 不是对象 | RawQuery 字符串 | 自行 `new URLSearchParams(req.query)` 解析 |
| `ctx.bash` 报 `move: command not found` | 执行器是 git-bash | 用 `mv`/`cp`；中文输出注意编码 |
| `ctx.fs` 越界 | 文件服务限工作区 | 工作区外文件走内核接口（如 `fs.image` 等） |
| `ctx.web.fetch` 只 GET | 设计约束 | POST 走 `ctx.http` 注册接口反向调用或 bash curl |
| 工具改造后「消失」/新旧并存 | 改名/合并未同步工具集白名单 | 编辑工具集（`toolset_edit`）：旧名移除、新名加入 |
| 手改的磁盘插件被重跑覆盖 | `tool_plugin_gen` 生成器整文件重写（genToolGroups） | 把插件移出生成器组后再手改（或改生成器源） |
| 动态审批工具仍每次都弹审批 | 同时设了 `requiresApproval: true` | 移除 `requiresApproval`，只留 `dynamicApproval` |
| 工具对勾勾了 Agent 却不用 | 工具集收敛 | 确认已加入工具集（勾选即加入，去掉即移除） |
| 沙箱里 `require`/`setTimeout` 不可用 | 无 Node API | 一律走 ctx 服务（`ctx.fs`/`ctx.timeout`/`ctx.web`/...） |
| `CordisApi` 插件里用 `ctx.set('svc', impl)` | cordis 3 语义 | `app.set('service', impl)` / `app.get('service')` |
| 插件 HTTP 接口返回 200 但 body 为空 | handler 返回了裸对象/`res.json()` 而宿主只认字符串 body | 显式返回字符串：`return JSON.stringify(data)`（或 `{status, body, headers}`，body 也要字符串） |
| UI 渲染把模型输出当 HTML 解析（排版错乱/内容消失） | client 半用了 `innerHTML` | 文本一律 `textContent`（LLM 输出常含 `<` `>` `&`）；只有自建结构才拼 HTML |
| 自主模式流是「多插件并存」的假象 | 误以为 `register` 是单槽位覆盖，写了整包覆盖 | 2026-09-21 起为装配器**链**（§4.5）；同插件名重复注册才替换该项 |
| `ctx.subagent.run` 报「无会话运行环境」 | 在装载期 / 无会话回调里派发 | 先 `ctx.subagent.available()` 判；不可用则降级为纯文本判断（§4.8.2） |
| 声明 `inject:['toolset']` 后 `ctx.toolset.registerTemplate` undefined | 同键被工具集服务覆盖 | 二选一：要模板注册就别 inject toolset（§5） |
| 决策器抛错后自主模式「静默停止」 | 宿主按「无监督」处理（日志可见） | 决策器内 try/catch + 日志；不要把宿主异常当控制流（§4.8.1） |

### 数据纪律

- 工具 execute 返回结果有体积上限——只回摘要或路径引用；
- console 输出写透宿主日志（带 `[cordis:id]` 标签），避免刷屏；
- 插件行为与 bash 同级信任（沙箱隔离全局，但不是安全边界）。

---

## 附：Go 核心能力清单

见 [go-core-capabilities.md](./go-core-capabilities.md)——内核保留接口全清单、工具面、能力边界与扩展点。
