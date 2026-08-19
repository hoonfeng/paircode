# PairCode 插件开发文档

> 面向用户的完整插件开发指南。编写插件前请先通读本文件；
> Agent 侧另有精简版技能（`cordis-plugin-development`）供 LLM 写作时参考。

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

## 2. 插件存在的三种形态

| 形态 | 存放位置 | 生命周期 | 适用场景 |
|---|---|---|---|
| **动态插件** | 进程内存（`cordis_define` 创建） | 随进程结束消失（需 `cordis.patch.json` 持久化） | 快速试验、调试 |
| **磁盘插件包** | `<InstallDir>/.pair/plugins/<name>/`（全局）或工具集 | 启动自动装载，跨重启存续 | 正式插件、UI 插件（全局） |
| **工具集插件** | `.pair/toolsets/*.json` | 启动随工具集装载 | 项目级工具插件 |

磁盘插件包与工具集插件是**持久化**形式；动态插件是**临时**形式（`cordis_define` 同步到全局插件包可转持久化）。

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
  apply(ctx, config) {            // config = cordis_run 传入 / package.json "config"
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

`ctx` 是插件可用的全部宿主能力入口。**无条件注入**的成员（29 个）：

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
| `ctx.loopFactory.register(apply)` | `register((opts) => overrides\|null)` | 注册 Agent 循环装配器（单槽位，对齐 harness setFactory）；返回同形状对象则非空覆盖 |
| `ctx.toolset.registerTemplate({id, title, match?, generate})` | `→ void` | 注册工具集构建模板（`generate(profile, requirement)` 返回插件定义数组） |
| `ctx.market.register({kind, source, name, desc})` | `→ true` | 注册市场源（kind: skill/mcp/plugin）；另有 `unregister(kind)` / `list()` |
| `ctx.registerClientMethod(method, fn)` | `→ void` | host 半暴露方法给浏览器 client 半（`ui.invoke(plugin, method, args)` 远程调用） |
| `ctx.app.workspaceRoot` | 字符串 | 当前工作区根 |
| `ctx.app.root` | 字符串 | 主工作区根（实时） |
| `ctx.app.folders` / `projectName` / `installDir` / `configDir` / `recentProjects` / `workspaceFolders` | 只读属性 | 宿主环境信息（实时读取） |

### 4.6 harness 全局（与 ctx 同作用域）

| 成员 | 签名 | 说明 |
|---|---|---|
| `harness.defineTool(tool)` | 只校验不注册（返回原对象） | 预检工具定义 |
| `harness.registerTool(tool)` | 同 `ctx.tools.register` | 注册工具 |
| `harness.handle(method, fn)` | 同 `ctx.registerClientMethod` | 注册可被调用的方法（Go 侧 Invoke） |

---

## 5. inject 声明式服务

`inject` 数组中声明的服务会作为 `ctx.xxx` 属性注入（未声明访问为 undefined）。可用服务（9 个）：

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

> 还有 `app` / `workspaceRoot` / `store` 三个静态服务：`app` 已无条件注入（见 4.5），
> `workspaceRoot` 可用 `ctx.get('workspaceRoot')` 取（宿主固有服务），`store` 为会话存储（ConversationStore）。

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

- **同名冲突会被拒绝**（不能覆盖宿主或其他插件工具）——换名或先 `cordis_stop` 占用方；
- schema type 限 `string/number/integer/boolean/object/array/null`；
- **Agent 是否可用由工具集决定**（见 §14）：工具注册进 Registry 只是「存在」，加入工具集后 Agent 才能调用；
- 结果有体积上限：不要返回完整大文件/大数组，只回摘要或路径引用；
- `ctx.hostTool.exec(name, args)` 可在插件里复用宿主 Go 执行器（迁移模式）。

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

  // 调后端 API（受限）
  const data = await ui.http.get('/api/plugins')       // 或 ui.http.post(path, body)
}
```

**预定义 UI 槽位**（`ui.registerSlot({slotId, ...})` 注册占用；替换型 single 区域内下拉切换占用者，叠加型 list 勾选激活）：

- **替换型（single）**：`titlebar` / `activitybar` / `sidebar` / `editor` / `right-panel` / `chat` / `statusbar`
- **叠加型（list）**：`overlay`（浮动层）/ `titlebar-right` / `activitybar`（图标列）/ `editor-toolbar` / `chat-tools` / `statusbar-items`

> 同一 slotId 可同时存在 single 与 list 两类占用，机制按 kind 分流。

事件流：host→浏览器 经 `/api/plugins/client-events` 每 2s 轮询取增量；浏览器→host 经 `/api/plugins/event`。

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
- 插件包内可带 `bin/<name>.exe`（独立二进制，经 `ctx.binary.exec` 调用）与 `assets/`（资源）；
- 项目级持久插件也可放 `.pair/plugins/`（工作区）——重启自动装载；
- 动态插件（cordis_define）会同步为插件包（全局插件包目录），重启存续。

---

## 14. 工具集与 Agent 可见性

- **工具集**（`.pair/toolsets/*.json`）是「Agent 可用工具」的声明集合；
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
1. 编写代码 → cordis_define（返回稳定 id dyn-<n>）
2. cordis_run id=dyn-<n>       → 装载运行（验证）
3. 修改代码 → cordis_define pluginId=dyn-<n> code=...（追加新版本）
4. cordis_run id=dyn-<n>       → 装载最新版（自动先停旧实例）
5. 回滚 → cordis_run id=dyn-<旧版本号>（指定精确版本）
6. 固化 → 写成磁盘插件包（.pair/plugins/）或工具集，重启存续
```

- `cordis_inspect id=xxx` 看版本链与状态；`version=vN` 读指定版本源码与诊断；
- `cordis_stop` 停止插件（定义保留可再 run）；`cordis_undefine` 永久删除（含磁盘包）；
- 插件状态：`stopped / running / waiting（缺依赖服务）/ rejected（装配拒绝）/ failed（装载失败）/ cancelled`。

---

## 17. 最佳实践与常见坑

### 最佳实践

- 先 `cordis_service_list` / `cordis_inspect_query` 查精确服务签名，不要凭记忆写；
- 跨插件共享逻辑用 `ctx.provide/get`（含函数），比 HTTP 或复制代码干净；
- 回调（事件/timer/工具 execute）在 VM 锁保护下执行，可安全访问插件闭包变量；**不要在回调外持有 goja 值跨 goroutine 使用**；
- 命名加插件前缀：工具名、事件名、服务名、设置 key；
- 需要清理的资源（定时器/连接）用 `ctx.effect` 或返回 cancel/disposer；
- 长耗时任务用 `ctx.process.runBackground`（后台进程跨轮次存活）或 `ctx.binary.exec`（独立二进制）。

### 常见坑

| 现象 | 原因 | 处理 |
|---|---|---|
| `cordis_run` 报求值失败/语法错误 | JS/TS 语法或顶层异常 | 修代码 → define append → run；TS 类型注解可用 |
| 插件进入 `waiting` | inject 服务未就绪 | 等服务提供方运行；或改用 `ctx.get` 判 undefined |
| 工具注册报同名冲突 | 工具名被占用 | 换名，或 `cordis_stop` 占用方 |
| `apply` 失败（diag 可见） | 运行期异常 | `cordis_inspect id=xxx` 看 diag/lastError 定位 |
| 插件 stop 后工具还在 | 未走 Unload 回收 | `cordis_stop` 正确回收；自注册全局资源用 `ctx.effect` 清理 |
| `req.query` 不是对象 | RawQuery 字符串 | 自行 `new URLSearchParams(req.query)` 解析 |
| `ctx.bash` 报 `move: command not found` | 执行器是 git-bash | 用 `mv`/`cp`；中文输出注意编码 |
| `ctx.fs` 越界 | 文件服务限工作区 | 工作区外文件走内核接口（如 `fs.image` 等） |
| `ctx.web.fetch` 只 GET | 设计约束 | POST 走 `ctx.http` 注册接口反向调用或 bash curl |
| 工具对勾勾了 Agent 却不用 | 工具集收敛 | 确认已加入工具集（勾选即加入，去掉即移除） |
| 沙箱里 `require`/`setTimeout` 不可用 | 无 Node API | 一律走 ctx 服务（`ctx.fs`/`ctx.timeout`/`ctx.web`/...） |
| `CordisApi` 插件里用 `ctx.set('svc', impl)` | cordis 3 语义 | `app.set('service', impl)` / `app.get('service')` |

### 数据纪律

- 工具 execute 返回结果有体积上限——只回摘要或路径引用；
- console 输出写透宿主日志（带 `[cordis:id]` 标签），避免刷屏；
- 插件行为与 bash 同级信任（沙箱隔离全局，但不是安全边界）。

---

## 附：Go 核心能力清单

见 [go-core-capabilities.md](./go-core-capabilities.md)——内核保留接口全清单、工具面、能力边界与扩展点。
