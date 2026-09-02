---
name: cordis-plugin-development
description: 编写/修改 cordis 动态插件（JS/TS，goja 沙箱）的完整指南：插件形态、ctx 能力全表、harness 用法、版本化工作流、常见错误排查。在 cordis_define 写插件之前或 cordis_run 失败后先加载本技能。
---

# cordis 插件开发指南

本环境支持动态插件：JS/TS 代码在 goja 沙箱中执行（对齐 deepseek-harness cordis-host-runner）。
插件代码只存在于进程内存（不落盘、跨重启不存续）；需要跨重启存续用磁盘插件包或 .pair/cordis.patch.json。

★ 完整版用户文档：docs/plugin-development.md（ctx 全表/示例/坑）+ docs/go-core-capabilities.md（Go 内核能力清单）。
★ 写插件前先 cordis_service_list 查精确签名；动手前可看磁盘插件现成范例（.pair/plugins/core-api、tool-git 等）。

## 1. 插件形态（两种）

```js
// ① 对象形态（apply 必填）
return {
  name: 'my-plugin',
  inject: ['fs', 'web'],           // 可选：硬依赖服务（缺失→插件 waiting，服务出现自动激活）
  apply(ctx, config) {             // config 来自 cordis_run 的 config 参数 / package.json "config"（无则 undefined）
    // 注册工具 / 监听事件 / 提供服务 ...
  }
}

// ② 函数形态（cordis 生态惯例 module.exports = function(ctx, config){}）
return function myPlugin(ctx, config) {
  // 函数名作插件名；匿名函数用 dyn id
}
// 函数形态可用静态属性声明硬依赖：
myPlugin.inject = ['fs']
```

## 2. ctx 能力面（★ 以源码 jsplugin.go buildContextObject 为准；下表全）

### 无条件注入成员
| 成员 | 签名 |
|---|---|
| ctx.get(name) / ctx.provide(name, obj) | 动态服务读写（provide 含函数可用；卸载自动撤销；新服务触发等待者激活） |
| ctx.on(name, fn) / ctx.emit(name, payload) | 事件订阅/广播（ui:/client: 前缀自动转发浏览器） |
| ctx.effect(fn) | 插件卸载清理 |
| ctx.timeout(fn, ms) / ctx.interval(fn, ms) | 定时器（返回 cancel；卸载自动清理） |
| ctx.tools.register({name,description,parameters,execute}) / list() | 工具注册/清单（同名冲突报错；list 只含对 cordis 可见的） |
| ctx.hostTool.exec(name, args) / .meta(name) / .names() | 宿主 Go 存档执行器（迁移模式：编排在插件、能力在宿主） |
| ctx.http.register(method, path, fn) | 旧形态接口注册（fn(req)→resp；"/*" 结尾前缀匹配；另有 list()） |
| ctx.webServer.register({kind:'exact'\|'prefix', path, handler}) | ★ 新接口推荐；Node 风格 handler(req,res)，不区分方法（自查 req.method）；兼容三参形态；req.query 是 RawQuery 字符串 |
| ctx.sse.register(path, (emit, params)=>cleanup) / ctx.ws.register(path, (conn, params)=>cleanup) | 实时推送/WebSocket（conn.send/close/onMessage） |
| ctx.kernel.routes()/installed()/total()/install([{key}]) | 内核路由表查询/挂载（卸载自动摘除；core-api 插件范例） |
| ctx.process.runBackground(cmd, cwd?)/readOutput(id)/kill(id) | 后台进程（全局单例跨轮次存活） |
| ctx.binary.exec(tool, args, {bin?, timeout?}) / .dir() | 独立插件二进制（<插件目录>/bin/<name>.exe；opts.bin 跨插件共用） |
| ctx.registerSettings({key,title,fields})/getSettings(key?)/setSettings(key, obj) | 插件配置（fields 含 name/label/type/default/hint/group/options/min/max/step/placeholder；setSettings 整对象覆盖） |
| ctx.systemPrompt.section({name,order,text}) / .variable({name,provider}) | 系统提示片段/动态变量 {{name}} |
| ctx.loopFactory.register(apply) | Agent 循环装配器（单槽位；apply(opts)→overrides\|null） |
| ctx.provider.register(name, impl) | ★ Provider 实现级插槽（2026-09-03 协议层重构）：注册服务商名的实现（impl(params)→Provider 实例）；同名覆盖返回还原函数（卸载自动回退） |
| ctx.toolset.registerTemplate({id,title,match?,generate}) | 工具集构建模板（generate 返回插件定义数组） |
| ctx.market.register({kind,source,name,desc})/unregister/list() | 市场源（skill/mcp/plugin） |
| ctx.registerClientMethod(method, fn) | host 半方法给浏览器 client 半远程调用（ui.invoke） |
| ctx.app.{workspaceRoot,root,folders,projectName,installDir,configDir,recentProjects,workspaceFolders} | 宿主信息（动态只读） |

### inject 声明服务（9 个，声明后 ctx.xxx 可用；未声明访问 undefined）
fs（工作区受限，越界拦截）/ web（GET，60s 超时 4MB）/ bash（120s 超时；git-bash 非 cmd）/ sse / ws / logger(scope)→{log,info,warn,debug,error} / timer / kernel / market
另有静态服务：app（已无条件注入）、workspaceRoot（ctx.get('workspaceRoot')）、store（会话存储）。

## 3. 注册工具（harness / ctx.tools）

```js
apply(ctx) {
  ctx.tools.register({
    name: 'my_tool',
    description: '做什么（给 LLM 看的说明）',
    parameters: { type: 'object', properties: { arg1: { type: 'string', description: '...' } }, required: ['arg1'] },
    async execute(args) {
      // args 是调用参数；返回 {text} 或任意 JSON 值
      return { text: '结果' }
    }
  })
}
```

- schema 校验：type 限 string/number/integer/boolean/object/array/null；`$ref` 只允许 `#/` 内部引用（防逃逸）。
- ★ 执行超时由插件自身控制（2026-08-22 起宿主不再强加 30s）：工具 execute 默认**不限时**
  （阻塞型交互工具如 ask_user 靠会话层超时）；如需死循环护栏，在工具定义上声明
  `timeout: 秒数`（如 `timeout: 30`，>0 才启用 goja Interrupt 强制中断）。
- 工具同名冲突会被拒绝（不能覆盖宿主或他人插件工具）——换名或先 cordis_stop 占用方。
- ★ agent 可见性由工具集决定（对勾=加入工具集；去掉勾=从工具集移除；加入才可用）。
- `harness.defineTool(tool)` 只校验不注册；`harness.registerTool` 同 ctx.tools.register；`harness.handle(method, fn)` 注册可调用方法（Go 侧 Invoke，同 registerClientMethod）。

### Provider 实现级插槽（2026-09-03 新增）

```js
apply(ctx) {
  ctx.provider.register('my-provider', (params) => ({ // impl(params)→Provider 实例
    async chat(session) { /* ... */ },              // Chat 能力（agentloop 调用）
  }))
}
```

- 业务层统一经 `CreateProvider(params)` 创建：注册表命中 → 插件实现（新协议无需改
  Go 内核）；未命中按协议路由内置实现（anthropic-messages / openai-responses /
  openai-completions）。
- 插件卸载自动还原（注册时登记 cleanup，还原后回退 OpenAI 实现）；
  参考：internal/agent/provider_impls.go、jsplugin_providerimpl.go。

## 4. 生命周期与事件

```js
apply(ctx) {
  const cancel = ctx.on('some:event', payload => { ... })  // 订阅事件
  ctx.effect(() => { /* 插件卸载时执行清理 */ })
  ctx.provide('mySvc', { hello() { return 'world' } })  // 提供服务；卸载自动撤销
}
```

回调（事件/timer/工具 execute）在 VM 锁保护下执行，可安全访问插件闭包变量；**不要在回调外持有 goja 值跨 goroutine 使用**。

## 5. 版本化工作流（修改插件）

```text
1. cordis_inspect id=xxx             → 看版本链与当前状态
2. cordis_inspect id=xxx version=vN  → 读当前源码与诊断（不要凭记忆臆测）
3. 修改代码后：cordis_define pluginId=xxx code=...  → 追加版本（existing append）
4. cordis_run id=xxx                 → 装载最新版（restart：自动先停旧实例）
5. 回滚：cordis_run id=dyn-<旧版本号>  → 指定精确版本装载
```

- 首次 cordis_define 返回的 `dyn-<n>` 就是稳定 pluginId；后续追加版本保持同一 pluginId。
- cordis_undefine 删除整个插件（定义 + 磁盘包）。

## 6. 内置 cordis 运行时（CordisApi）

沙箱全局 `CordisApi`（@cordisjs/core bundle），可建真 cordis app 跑生态插件协作：

```js
return async function(ctx) {
  const app = new CordisApi.api.Context()
  app.plugin(/* cordis 生态插件 */)
  await app.start()          // 触发 ready
  // ctx.set('service', impl) / ctx.get('service') —— cordis 3 用 set 而非 provide！
}
```

Node API（require/setTimeout/fetch/process 等）沙箱中**不可用**，调用即抛引导错误——一律走 ctx 服务。

## 7. client 半（UI 插件；含 client 自动 global 作用域）

```js
// client 半：(ui) => void
(ui) => {
  ui.on(event, fn)                       // 收 host 事件（ui:/client: 前缀）
  ui.emit(event, payload)                // 发事件回 host（host: 前缀给 ctx.on('host:xxx') 消费）
  ui.invoke(plugin, method, args?)       // 远程调用 host 半 ctx.registerClientMethod 注册方法（RPC）
  ui.reportFailure(phase, message)       // 失败上报（render/guard/boot）
  ui.registerPanel({id,title,icon,render,props})  // 自定义面板（render(el, ui)，el 为容器 DOM）
  ui.http.get(path) / ui.http.post(path, body)   // 受限后端 API
}
// UI 槽位：single 替换型 titlebar/activitybar/sidebar/editor/right-panel/chat/statusbar
//          list 叠加型 overlay/titlebar-right/activitybar/editor-toolbar/chat-tools/statusbar-items
```

## 8. 常见错误排查

| 现象 | 原因 | 处理 |
|---|---|---|
| cordis_run 报"求值失败/语法错误" | JS/TS 语法或顶层异常 | 修代码 → define append → run；可用 TS 类型注解（内置编译器转译） |
| 插件进入 waiting | inject 服务未就绪 | 等提供服务方运行；或改用 ctx.get 判 undefined（可选依赖） |
| 工具注册报同名冲突 | 工具名被宿主/他人占用 | 换工具名，或先 cordis_stop 占用方 |
| apply 失败（diag 可见） | 运行期异常（如调用不存在的方法） | cordis_inspect id=xxx 看 diag/lastError 定位阶段 |
| 插件 stop 后工具还在 | 未走 Unload 回收 | cordis_stop 正确回收；自己注册的全局资源用 ctx.effect 清理 |
| req.query 不是对象 | RawQuery 字符串 | 自行 URLSearchParams 解析 |
| ctx.bash 报 move 不存在 | 执行器是 git-bash | 用 mv/cp；中文输出注意编码 |
| ctx.fs 越界 / web.fetch 只 GET | 设计约束 | 工作区外走内核接口；POST 走 ctx.http 反向或 bash curl |

## 9. 数据纪律

- 工具 execute 返回结果有体积上限：不要返回完整大文件/大数组，只回摘要或路径引用。
- console.log 输出会写透宿主日志（带 `[cordis:id]` 标签），避免刷屏。
- 插件行为应与 bash 同等信任级别（沙箱隔离全局，但不是安全边界）。